package natstest

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestClusterR3StreamSurvivesNodeLoss(t *testing.T) {
	c := StartCluster(t, 3, "")
	nc, err := nats.Connect(c.URLs())
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}

	var stream jetstream.Stream
	eventually(t, func() error { // the meta leader takes a moment to elect
		stream, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
			Name: "R3", Subjects: []string{"r3.>"}, Replicas: 3,
		})
		return err
	})
	info, err := stream.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := 1 + len(info.Cluster.Replicas); got != 3 {
		t.Fatalf("stream has %d copies, want 3", got)
	}

	oldLeader := info.Cluster.Leader
	for i, s := range c.Servers {
		if s.Name() == oldLeader {
			c.Stop(i)
		}
	}
	eventually(t, func() error {
		info, err := stream.Info(context.Background())
		if err == nil && (info.Cluster.Leader == "" || info.Cluster.Leader == oldLeader) {
			err = context.DeadlineExceeded
		}
		return err
	})
}

func eventually(t *testing.T, f func() error) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		err := f()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
