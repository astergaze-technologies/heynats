package diagnose

import (
	"context"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/monitor"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func waitReport(t *testing.T, d *Diagnoser, what string, f func(*Report) bool) *Report {
	t.Helper()
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		if r := d.Report(); f(r) {
			return r
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s: %+v", what, d.Report())
		}
	}
}

func finding(r *Report, rule, resource string) *Finding {
	for i, f := range r.Findings {
		if f.RuleID == rule && f.Resource == resource {
			return &r.Findings[i]
		}
	}
	return nil
}

func TestStoppedNodeDegradesThenRecovers(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	nc, err := nats.Connect(c.Servers[0].ClientURL(), nats.UserInfo("sys", "Lm2KZ22PCKB9GjwUYzxnXAm9IyDM8svZZUSp7tSOYcQ="), natstest.ConnectTimeout)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	d := Start(monitor.New(nc), 500*time.Millisecond)
	defer d.Stop()

	waitReport(t, d, "healthy cluster", func(r *Report) bool { return r.Verdict == "healthy" && len(r.Servers) == 3 })

	c.Stop(2)
	start := time.Now()
	r := waitReport(t, d, "n3 unreachable", func(r *Report) bool {
		return finding(r, "server.unreachable", "server:n3") != nil
	})
	if took := time.Since(start); took > 10*time.Second || r.Verdict != "degraded" {
		t.Fatalf("verdict %s after %s", r.Verdict, took)
	}
	first := finding(r, "server.unreachable", "server:n3").FirstSeen
	time.Sleep(time.Second)
	if f := finding(d.Report(), "server.unreachable", "server:n3"); f == nil || !f.FirstSeen.Equal(first) {
		t.Fatalf("first_seen moved or finding vanished: %+v", f)
	}

	c.Restart(2)
	waitReport(t, d, "recovery", func(r *Report) bool {
		return finding(r, "server.unreachable", "server:n3") == nil && r.Verdict == "healthy"
	})
}

func TestReplicaOfflineThenResolves(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	sys := connectAs(t, c, "sys")
	js, _ := jetstream.New(connectAs(t, c, "admin"))
	ctx := context.Background()
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		if _, err := js.CreateStream(ctx, jetstream.StreamConfig{Name: "ORDERS", Subjects: []string{"orders.>"}, Replicas: 3}); err == nil {
			break
		} else if time.Now().After(deadline) {
			t.Fatal(err)
		}
	}
	d := start(monitor.New(sys), 500*time.Millisecond, defaultWindows)
	defer d.Stop()
	waitReport(t, d, "healthy", func(r *Report) bool { return r.Verdict == "healthy" })

	// Stop a node that isn't the one our clients are on.
	c.Stop(2)
	waitReport(t, d, "offline replica", func(r *Report) bool {
		return finding(r, "js.replica_offline", "stream:APP/ORDERS") != nil
	})
	c.Restart(2)
	waitReport(t, d, "replica back", func(r *Report) bool {
		return finding(r, "js.replica_offline", "stream:APP/ORDERS") == nil
	})
}

func TestConsumerLagGrowing(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	admin := connectAs(t, c, "admin")
	js, _ := jetstream.New(admin)
	ctx := context.Background()
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		if _, err := js.CreateStream(ctx, jetstream.StreamConfig{Name: "ORDERS", Subjects: []string{"orders.>"}, Replicas: 3}); err == nil {
			break
		} else if time.Now().After(deadline) {
			t.Fatal(err)
		}
	}
	if _, err := js.CreateConsumer(ctx, "ORDERS", jetstream.ConsumerConfig{Durable: "billing"}); err != nil {
		t.Fatal(err)
	}
	// Account scope: the admin user's own JetStream API.
	w := defaultWindows
	w.growFor = 2 * time.Second
	d := start(monitor.New(admin), 300*time.Millisecond, w)
	defer d.Stop()

	stop := make(chan struct{})
	defer close(stop)
	go func() { // steady publishing, nobody reading
		for {
			select {
			case <-stop:
				return
			case <-time.After(100 * time.Millisecond):
				_ = admin.Publish("orders.new", []byte("x"))
			}
		}
	}()
	r := waitReport(t, d, "growing lag", func(r *Report) bool {
		return finding(r, "js.consumer_lag_growing", "consumer:ORDERS/billing") != nil
	})
	if f := finding(r, "js.consumer_lag_growing", "consumer:ORDERS/billing"); f.Severity != Warning {
		t.Fatalf("finding %+v", f)
	}
}

func connectAs(t *testing.T, c *natstest.Cluster, user string) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(c.Servers[0].ClientURL(), nats.UserInfo(user, "Lm2KZ22PCKB9GjwUYzxnXAm9IyDM8svZZUSp7tSOYcQ="), natstest.ConnectTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	return nc
}

// One scenario per consumer rule, all in one cluster and one diagnoser.
func TestConsumerRulesOnACluster(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	admin := connectAs(t, c, "admin")
	js, _ := jetstream.New(admin)
	ctx := context.Background()
	stream := func(name string, subjects ...string) {
		t.Helper()
		for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
			_, err := js.CreateStream(ctx, jetstream.StreamConfig{Name: name, Subjects: subjects})
			if err == nil {
				return
			}
			if time.Now().After(deadline) {
				t.Fatal(err)
			}
		}
	}
	consumer := func(s string, cfg jetstream.ConsumerConfig) jetstream.Consumer {
		t.Helper()
		cons, err := js.CreateConsumer(ctx, s, cfg)
		if err != nil {
			t.Fatal(err)
		}
		return cons
	}
	publish := func(subject string, n int) {
		for range n {
			if _, err := js.Publish(ctx, subject, []byte("x")); err != nil {
				t.Fatal(err)
			}
		}
	}

	stream("ORDERS", "orders.>", "legacy.>")
	publish("orders.new", 20)

	full := consumer("ORDERS", jetstream.ConsumerConfig{Durable: "full", MaxAckPending: 5, AckPolicy: jetstream.AckExplicitPolicy})
	if _, err := full.Fetch(5); err != nil { // received, never acked
		t.Fatal(err)
	}
	consumer("ORDERS", jetstream.ConsumerConfig{Durable: "paused"})
	if _, err := js.PauseConsumer(ctx, "ORDERS", "paused", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	consumer("ORDERS", jetstream.ConsumerConfig{Durable: "pushed", DeliverSubject: "deliver.nobody"})
	consumer("ORDERS", jetstream.ConsumerConfig{Durable: "idle"})
	consumer("ORDERS", jetstream.ConsumerConfig{Durable: "legacy", FilterSubject: "legacy.>"})
	// Dropping legacy.> from the stream orphans the consumer's filter.
	if _, err := js.UpdateStream(ctx, jetstream.StreamConfig{Name: "ORDERS", Subjects: []string{"orders.>"}}); err != nil {
		t.Fatal(err)
	}
	flaky := consumer("ORDERS", jetstream.ConsumerConfig{Durable: "flaky", AckPolicy: jetstream.AckExplicitPolicy, AckWait: 200 * time.Millisecond})
	stop := make(chan struct{})
	defer close(stop)
	go func() { // a client that keeps taking messages and never acks them
		for {
			select {
			case <-stop:
				return
			default:
				_, _ = flaky.Fetch(10, jetstream.FetchMaxWait(300*time.Millisecond))
			}
		}
	}()

	d := start(monitor.New(admin), 300*time.Millisecond, windows{
		lagFor: time.Minute, growFor: time.Hour, redeliverFor: 2 * time.Second, idleFor: time.Second,
	})
	defer d.Stop()
	want := map[string]string{
		"consumer.ack_pending_full":       "consumer:ORDERS/full",
		"consumer.paused":                 "consumer:ORDERS/paused",
		"consumer.no_interest":            "consumer:ORDERS/pushed",
		"consumer.nobody_pulling":         "consumer:ORDERS/idle",
		"consumer.filter_matches_nothing": "consumer:ORDERS/legacy",
		"consumer.high_redelivery":        "consumer:ORDERS/flaky",
	}
	r := waitReport(t, d, "every consumer rule", func(r *Report) bool {
		for rule, res := range want {
			if finding(r, rule, res) == nil {
				return false
			}
		}
		return true
	})
	resume := finding(r, "consumer.paused", "consumer:ORDERS/paused").Fixes[0]
	if resume.Action != "consumer.resume" || resume.Params["consumer"] != "paused" {
		t.Fatalf("own-account pause should offer Resume: %+v", resume)
	}
}
