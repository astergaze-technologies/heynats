package monitor

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
)

func TestServersKeepsAStoppedNode(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	m := monitorFor(t, c, "sys")

	healthy := func(l *ServerList) bool {
		for _, s := range l.Servers {
			if !s.Responding || s.Health != "ok" || !s.JetStream || s.Version == "" {
				return false
			}
		}
		return len(l.Servers) == 3 && !l.Partial
	}
	// Health stays "no meta leader" until JetStream elects one.
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(300 * time.Millisecond) {
		list, err := m.Servers()
		if err == nil && healthy(list) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("cluster never listed healthy: %+v, %v", list, err)
		}
	}

	c.Stop(2)
	list, err := m.Servers()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Servers) != 3 || !list.Partial || list.Servers[2].Name != "n3" || list.Servers[2].Responding {
		t.Fatalf("after stopping n3: %+v", list)
	}
}

func TestServerRaw(t *testing.T) {
	c := natstest.StartCluster(t, 1, "../../nats.conf")
	m := monitorFor(t, c, "sys")
	id := c.Servers[0].ID()

	data, err := m.ServerRaw(id, "VARZ")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Name string `json:"server_name"`
	}
	if json.Unmarshal(data, &v); v.Name != "n1" {
		t.Fatalf("VARZ for wrong server: %s", data)
	}

	for _, bad := range [][2]string{{id + ".>", "VARZ"}, {"*", "VARZ"}, {id, "PROFILEZ"}} {
		if _, err := m.ServerRaw(bad[0], bad[1]); !errors.Is(err, ErrBadRequest) {
			t.Errorf("ServerRaw(%q, %q) = %v, want bad request", bad[0], bad[1], err)
		}
	}
	if _, err := monitorFor(t, c, "admin").ServerRaw(id, "VARZ"); !errors.Is(err, ErrSystemOnly) {
		t.Errorf("admin got %v, want system only", err)
	}
}
