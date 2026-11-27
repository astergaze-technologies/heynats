package monitor

import (
	"encoding/json"
	"fmt"
	"sort"
)

// TopoNode is a server of ours, a remote gateway cluster, or a leaf node's
// remote side. Clusters group servers for layout.
type TopoNode struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"` // server, gateway, leaf
	Label      string `json:"label"`
	Cluster    string `json:"cluster,omitempty"`
	Responding bool   `json:"responding"`
	Version    string `json:"version,omitempty"`
}

// TopoEdge sums every connection between two nodes. Counters are cumulative:
// the UI turns two polls into msgs/s.
type TopoEdge struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Kind    string `json:"kind"` // route, gateway, leaf
	Links   int    `json:"links"`
	InMsgs  int64  `json:"in_msgs"`
	OutMsgs int64  `json:"out_msgs"`
}

type Topology struct {
	Partial  bool       `json:"partial"`
	Expected int        `json:"expected"`
	Nodes    []TopoNode `json:"nodes"`
	Edges    []TopoEdge `json:"edges"`
}

type linkCounts struct {
	InMsgs  int64 `json:"in_msgs"`
	OutMsgs int64 `json:"out_msgs"`
}

// BuildTopology turns the servers list and each server's ROUTEZ, GATEWAYZ
// and LEAFZ (keyed by server name) into one graph.
func BuildTopology(servers []ServerRow, routez, gatewayz, leafz map[string]json.RawMessage) *Topology {
	t := &Topology{}
	nodes := map[string]*TopoNode{}
	edges := map[string]*TopoEdge{}
	add := func(n TopoNode) {
		if nodes[n.ID] == nil {
			nodes[n.ID] = &n
		}
	}
	link := func(from, to, kind string, c linkCounts, undirected bool) {
		if undirected && to < from {
			from, to = to, from
			c.InMsgs, c.OutMsgs = c.OutMsgs, c.InMsgs
		}
		key := kind + "|" + from + "|" + to
		e := edges[key]
		if e == nil {
			e = &TopoEdge{From: from, To: to, Kind: kind}
			edges[key] = e
		}
		e.Links++
		e.InMsgs += c.InMsgs
		e.OutMsgs += c.OutMsgs
	}

	for _, s := range servers {
		add(TopoNode{ID: s.Name, Kind: "server", Label: s.Name, Cluster: s.Cluster, Responding: s.Responding, Version: s.Version})
	}
	for name, raw := range routez {
		var d struct {
			Routes []struct {
				RemoteName string `json:"remote_name"`
				RemoteID   string `json:"remote_id"`
				linkCounts
			} `json:"routes"`
		}
		if json.Unmarshal(raw, &d) != nil {
			continue
		}
		for _, r := range d.Routes {
			peer := r.RemoteName
			if peer == "" {
				peer = r.RemoteID
			}
			// Both ends report the same route; count it from one side only.
			if name < peer {
				link(name, peer, "route", r.linkCounts, true)
			}
		}
	}
	for name, raw := range gatewayz {
		var d struct {
			Outbound map[string]struct {
				Connection *linkCounts `json:"connection"`
			} `json:"outbound_gateways"`
		}
		if json.Unmarshal(raw, &d) != nil {
			continue
		}
		for remote, g := range d.Outbound {
			if g.Connection == nil {
				continue
			}
			id := "gateway:" + remote
			add(TopoNode{ID: id, Kind: "gateway", Label: remote, Cluster: remote, Responding: true})
			link(name, id, "gateway", *g.Connection, false)
		}
	}
	for name, raw := range leafz {
		var d struct {
			Leafs []struct {
				Name    string `json:"name"`
				Account string `json:"account"`
				IP      string `json:"ip"`
				Port    int    `json:"port"`
				linkCounts
			} `json:"leafs"`
		}
		if json.Unmarshal(raw, &d) != nil {
			continue
		}
		for _, l := range d.Leafs {
			label := l.Name
			if label == "" {
				label = fmt.Sprintf("%s:%d", l.IP, l.Port)
			}
			id := "leaf:" + label
			add(TopoNode{ID: id, Kind: "leaf", Label: label, Responding: true})
			link(name, id, "leaf", l.linkCounts, false)
		}
	}

	for _, n := range nodes {
		t.Nodes = append(t.Nodes, *n)
	}
	sort.Slice(t.Nodes, func(i, j int) bool { return t.Nodes[i].ID < t.Nodes[j].ID })
	for _, e := range edges {
		t.Edges = append(t.Edges, *e)
	}
	sort.Slice(t.Edges, func(i, j int) bool {
		a, b := t.Edges[i], t.Edges[j]
		return a.Kind+a.From+a.To < b.Kind+b.From+b.To
	})
	return t
}

func (m *Monitor) Topology() (*Topology, error) {
	list, err := m.Servers() // system only; also lists servers that stopped answering
	if err != nil {
		return nil, err
	}
	kinds := []string{"ROUTEZ", "GATEWAYZ", "LEAFZ"}
	data := make([]map[string]json.RawMessage, len(kinds))
	errs := make(chan error, len(kinds))
	for i, kind := range kinds {
		go func() {
			res, err := m.Ping(kind)
			data[i] = map[string]json.RawMessage{}
			if err == nil {
				for _, r := range res.Replies {
					if r.Error == nil {
						data[i][r.Server.Name] = r.Data
					}
				}
			}
			errs <- err
		}()
	}
	for range kinds {
		if err := <-errs; err != nil {
			return nil, err
		}
	}
	t := BuildTopology(list.Servers, data[0], data[1], data[2])
	t.Partial, t.Expected = list.Partial, list.Expected
	return t, nil
}
