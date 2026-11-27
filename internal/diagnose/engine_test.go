package diagnose

import (
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/monitor"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats.go"
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
	nc, err := nats.Connect(c.Servers[0].ClientURL(), nats.UserInfo("sys", "Lm2KZ22PCKB9GjwUYzxnXAm9IyDM8svZZUSp7tSOYcQ="))
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
