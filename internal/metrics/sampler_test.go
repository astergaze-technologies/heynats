package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/monitor"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/goleak"
)

const devPassword = "Lm2KZ22PCKB9GjwUYzxnXAm9IyDM8svZZUSp7tSOYcQ="

func connect(t *testing.T, c *natstest.Cluster, user string) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(c.Servers[0].ClientURL(), nats.UserInfo(user, devPassword))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	return nc
}

// seed publishes 1,000 messages into a stream with a consumer that reads none.
func seed(t *testing.T, c *natstest.Cluster) {
	t.Helper()
	nc := connect(t, c, "admin")
	js, _ := jetstream.New(nc)
	ctx := context.Background()
	var err error
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		if _, err = js.CreateStream(ctx, jetstream.StreamConfig{Name: "ORDERS", Subjects: []string{"orders.>"}, Replicas: 3}); err == nil {
			break
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.CreateConsumer(ctx, "ORDERS", jetstream.ConsumerConfig{Durable: "billing"}); err != nil {
		t.Fatal(err)
	}
	for range 1000 {
		if err := nc.Publish("orders.new", []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}
}

// waitFor polls f until it is true or 15s pass.
func waitFor(t *testing.T, what string, f func() bool) {
	t.Helper()
	for deadline := time.Now().Add(15 * time.Second); !f(); time.Sleep(200 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

func sum(w Window, name string) (total float32) {
	for _, vals := range w.Series[name] {
		for _, v := range vals {
			if v != nil {
				total += *v * float32(w.Step)
			}
		}
	}
	return total
}

func last(w Window, name, member string) float32 {
	vals := w.Series[name][member]
	for i := len(vals) - 1; i >= 0; i-- {
		if vals[i] != nil {
			return *vals[i]
		}
	}
	return -1
}

func TestSamplerSeesTrafficAndPending(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	for _, tc := range []struct{ user, consumer string }{
		{"sys", "APP/ORDERS/billing"}, // JSZ from every server
		{"admin", "ORDERS/billing"},   // own account via the JetStream API
	} {
		t.Run(tc.user, func(t *testing.T) {
			nc := connect(t, c, tc.user)
			s := Start(nc, monitor.New(nc), time.Second, nil)
			defer s.Stop()
			if tc.user == "sys" {
				seed(t, c)
			} else {
				nc := connect(t, c, "admin") // more traffic for this sampler's first window
				for range 1000 {
					_ = nc.Publish("orders.new", []byte("x"))
				}
				_ = nc.Flush()
			}

			waitFor(t, "1,000 messages in the msgs_in rate", func() bool {
				return sum(s.Server.Window([]string{"msgs_in"}, time.Hour, time.Now()), "msgs_in") >= 1000
			})
			waitFor(t, "consumer pending", func() bool {
				return last(s.JetStream.Window([]string{"consumer_pending"}, time.Hour, time.Now()), "consumer_pending", tc.consumer) >= 1000
			})
		})
	}
}

func TestStopLeavesNoGoroutines(t *testing.T) {
	c := natstest.StartCluster(t, 1, "../../nats.conf")
	nc := connect(t, c, "sys")
	mon := monitor.New(nc)
	mon.Source()
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	s := Start(nc, mon, time.Second, nil)
	time.Sleep(1500 * time.Millisecond)
	s.Stop()
}
