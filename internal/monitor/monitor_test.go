package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const devPassword = "Lm2KZ22PCKB9GjwUYzxnXAm9IyDM8svZZUSp7tSOYcQ="

func connect(t *testing.T, c *natstest.Cluster, node int, user string) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(c.Servers[node].ClientURL(), nats.UserInfo(user, devPassword), nats.DontRandomize())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	return nc
}

// monitorFor waits until all servers answer, i.e. the routes are up.
func monitorFor(t *testing.T, c *natstest.Cluster, user string) *Monitor {
	t.Helper()
	m := New(connect(t, c, 0, user))
	deadline := time.Now().Add(15 * time.Second)
	for {
		res, err := m.Ping("STATZ")
		if err == nil && len(res.Replies) == len(c.Servers) {
			return m
		}
		if time.Now().After(deadline) {
			t.Fatalf("cluster never answered in full: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func TestSystemUserSeesEveryServer(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	m := monitorFor(t, c, "sys")

	res, err := m.Ping("VARZ")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != System || len(res.Replies) != 3 || res.Partial {
		t.Fatalf("got source=%s servers=%d partial=%v", res.Source, len(res.Replies), res.Partial)
	}

	c.Stop(2)
	start := time.Now()
	res, err = m.Ping("VARZ")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Replies) != 2 || !res.Partial || res.Expected != 3 {
		t.Fatalf("after stopping a node: servers=%d of %d partial=%v", len(res.Replies), res.Expected, res.Partial)
	}
	if took := time.Since(start); took > 2500*time.Millisecond {
		t.Fatalf("partial answer took %s", took)
	}
}

func TestAppUserFallsBackToAccountScope(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	m := monitorFor(t, c, "admin")
	for i := range c.Servers {
		connect(t, c, i, "admin") // one extra client per node
	}

	connz, err := m.Connz()
	if err != nil {
		t.Fatal(err)
	}
	// 3 extra clients plus the monitor's own connection, spread over all nodes.
	if connz.Source != Account || connz.Servers != 3 || connz.NumConns != 4 || len(connz.Connections) != 4 {
		t.Fatalf("got source=%s servers=%d conns=%d listed=%d", connz.Source, connz.Servers, connz.NumConns, len(connz.Connections))
	}

	caps, err := m.Capabilities()
	if err != nil {
		t.Fatal(err)
	}
	if f := caps.Features["servers"]; f.OK || f.Reason == "" {
		t.Errorf("servers feature for app user: %+v", f)
	}
	if !caps.Features["trace"].OK || caps.Servers != 3 {
		t.Errorf("caps: %+v", caps)
	}
}

func TestJetStreamCountsEachStreamOnce(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	sys, app := monitorFor(t, c, "sys"), monitorFor(t, c, "admin")
	js, _ := jetstream.New(connect(t, c, 0, "admin"))
	ctx := context.Background()
	if _, err := js.CreateStream(ctx, jetstream.StreamConfig{Name: "ORDERS", Subjects: []string{"orders.>"}, Replicas: 3}); err != nil {
		t.Fatal(err)
	}
	if _, err := js.CreateConsumer(ctx, "ORDERS", jetstream.ConsumerConfig{Durable: "billing"}); err != nil {
		t.Fatal(err)
	}

	for _, m := range []*Monitor{sys, app} {
		deadline := time.Now().Add(10 * time.Second)
		for {
			got, err := m.JetStream()
			if err == nil && got.Streams == 1 && got.Consumers == 1 {
				if got.Source == System && (got.Servers != 3 || got.MaxStorage <= 0) {
					t.Fatalf("system totals: %+v", got)
				}
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: got %+v, %v; want 1 stream and 1 consumer", m.Source(), got, err)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}
