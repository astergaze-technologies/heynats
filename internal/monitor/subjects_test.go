package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestSubjectsListeningAndStored(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	listener, err := nats.Connect(c.Servers[1].ClientURL(), nats.UserInfo("admin", devPassword), nats.Name("eu-listener"), natstest.ConnectTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(listener.Close)
	if _, err := listener.Subscribe("orders.eu.created", func(*nats.Msg) {}); err != nil {
		t.Fatal(err)
	}
	if _, err := listener.QueueSubscribe("orders.*.created", "workers", func(*nats.Msg) {}); err != nil {
		t.Fatal(err)
	}
	_ = listener.Flush()

	admin := connect(t, c, 0, "admin")
	js, _ := jetstream.New(admin)
	ctx := context.Background()
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		if _, err := js.CreateStream(ctx, jetstream.StreamConfig{Name: "ORDERS", Subjects: []string{"orders.>"}, Replicas: 3}); err == nil {
			break
		} else if time.Now().After(deadline) {
			t.Fatal(err)
		}
	}
	for range 3 {
		_, _ = js.Publish(ctx, "orders.eu.created", nil)
	}
	_, _ = js.Publish(ctx, "orders.us.created", nil)

	for _, user := range []string{"sys", "admin"} { // admin: account-scope CONNZ must return subscriptions too
		m := monitorFor(t, c, user)
		sm, err := m.Subjects()
		if err != nil {
			t.Fatal(err)
		}
		exact, wild := sm.Listening["orders.eu.created"], sm.Listening["orders.*.created"]
		if len(exact) != 1 || exact[0].Name != "eu-listener" || exact[0].Server != "n2" {
			t.Fatalf("%s: exact subscriber %+v", user, exact)
		}
		if len(wild) != 1 || wild[0].Queue != "workers" {
			t.Fatalf("%s: wildcard subscriber kept on its own subject: %+v", user, wild)
		}
		var orders *StoredStream
		for i := range sm.Streams {
			if sm.Streams[i].Name == "ORDERS" {
				orders = &sm.Streams[i]
			}
		}
		if orders == nil || orders.Subjects[0] != "orders.>" {
			t.Fatalf("%s: stream patterns %+v", user, sm.Streams)
		}
		if user == "admin" && (orders.Counts["orders.eu.created"] != 3 || orders.Counts["orders.us.created"] != 1) {
			t.Fatalf("own account counts per subject: %+v", orders.Counts)
		}
	}
}
