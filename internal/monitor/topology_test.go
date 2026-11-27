package monitor

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
)

func TestBuildTopology(t *testing.T) {
	servers := []ServerRow{
		{Name: "n1", Cluster: "west", Responding: true},
		{Name: "n2", Cluster: "west", Responding: true},
		{Name: "n3", Cluster: "west"}, // stopped
	}
	raw := func(s string) json.RawMessage { return json.RawMessage(s) }
	// Route pooling: three connections between n1 and n2, reported by both.
	pool := `{"routes":[{"remote_name":"%[1]s","in_msgs":10,"out_msgs":1},{"remote_name":"%[1]s","in_msgs":10,"out_msgs":1},{"remote_name":"%[1]s","in_msgs":10,"out_msgs":1}]}`
	routez := map[string]json.RawMessage{
		"n1": raw(fmt.Sprintf(pool, "n2")),
		"n2": raw(fmt.Sprintf(pool, "n1")),
	}
	gatewayz := map[string]json.RawMessage{
		"n1": raw(`{"outbound_gateways":{"east":{"connection":{"in_msgs":5,"out_msgs":7}}}}`),
	}
	leafz := map[string]json.RawMessage{
		"n2": raw(`{"leafs":[{"name":"edge-1","ip":"10.0.0.9","port":7422,"in_msgs":2,"out_msgs":3}]}`),
	}
	topo := BuildTopology(servers, routez, gatewayz, leafz)

	nodes := map[string]TopoNode{}
	for _, n := range topo.Nodes {
		nodes[n.ID] = n
	}
	if len(nodes) != 5 || nodes["n3"].Responding || nodes["gateway:east"].Kind != "gateway" || nodes["leaf:edge-1"].Kind != "leaf" {
		t.Fatalf("nodes: %+v", topo.Nodes)
	}
	want := map[string]TopoEdge{
		"route":   {From: "n1", To: "n2", Kind: "route", Links: 3, InMsgs: 30, OutMsgs: 3},
		"gateway": {From: "n1", To: "gateway:east", Kind: "gateway", Links: 1, InMsgs: 5, OutMsgs: 7},
		"leaf":    {From: "n2", To: "leaf:edge-1", Kind: "leaf", Links: 1, InMsgs: 2, OutMsgs: 3},
	}
	if len(topo.Edges) != 3 {
		t.Fatalf("edges: %+v", topo.Edges)
	}
	for _, e := range topo.Edges {
		if e != want[e.Kind] {
			t.Errorf("%s edge = %+v, want %+v", e.Kind, e, want[e.Kind])
		}
	}
}

func TestTopologyGreysAStoppedNode(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	m := monitorFor(t, c, "sys")
	topo, err := m.Topology()
	if err != nil {
		t.Fatal(err)
	}
	if len(topo.Nodes) != 3 || len(topo.Edges) != 3 {
		t.Fatalf("want 3 servers fully meshed: %+v", topo)
	}

	c.Stop(2)
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		topo, err = m.Topology()
		if err == nil && len(topo.Edges) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("routes to n3 never dropped: %+v, %v", topo, err)
		}
	}
	for _, n := range topo.Nodes {
		if n.ID == "n3" && n.Responding {
			t.Fatal("n3 should be listed as not responding")
		}
	}
	if len(topo.Nodes) != 3 || !topo.Partial {
		t.Fatalf("n3 must stay on the graph: %+v", topo)
	}
}
