package monitor

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Peer is one replica as its group leader sees it (the leader isn't listed).
type Peer struct {
	Name    string        `json:"name"`
	Current bool          `json:"current"`
	Offline bool          `json:"offline,omitempty"`
	Active  time.Duration `json:"active"` // since last seen, nanoseconds
	Lag     uint64        `json:"lag,omitempty"`
}

type MetaGroup struct {
	Leader string `json:"leader"`
	Size   int    `json:"cluster_size"`
	Peers  []Peer `json:"peers"`
}

type StreamHealth struct {
	Account   string `json:"account"`
	Name      string `json:"name"`
	Leader    string `json:"leader"`
	Replicas  []Peer `json:"replicas"`
	Clustered bool   `json:"clustered"` // false for R1 on a single server: no leader to lose
	Messages  uint64 `json:"messages"`
	Bytes     uint64 `json:"bytes"`
	Consumers int    `json:"consumers"`
}

type ConsumerHealth struct {
	Account     string `json:"account"`
	Stream      string `json:"stream"`
	Name        string `json:"name"`
	Leader      string `json:"leader"`
	Replicas    []Peer `json:"replicas"`
	Pending     uint64 `json:"pending"`
	AckPending  int    `json:"ack_pending"`
	Redelivered int    `json:"redelivered"`
}

type JetStreamHealth struct {
	Source    Source           `json:"source"`
	Partial   bool             `json:"partial"`
	Meta      *MetaGroup       `json:"meta,omitempty"` // system account only
	Streams   []StreamHealth   `json:"streams"`
	Consumers []ConsumerHealth `json:"consumers"`
}

type jszCluster struct {
	Leader   string `json:"leader"`
	Replicas []Peer `json:"replicas"`
}

func (m *Monitor) JetStreamHealth() (*JetStreamHealth, error) {
	if m.Source() != System {
		return m.accountJetStreamHealth()
	}
	res, err := m.PingWith("JSZ", map[string]bool{"accounts": true, "streams": true, "consumer": true})
	if err != nil {
		return nil, err
	}
	out := &JetStreamHealth{Source: System, Partial: res.Partial, Streams: []StreamHealth{}, Consumers: []ConsumerHealth{}}
	streams := map[string]StreamHealth{}
	consumers := map[string]ConsumerHealth{}
	for _, r := range res.Replies {
		var d struct {
			Meta *struct {
				Leader   string `json:"leader"`
				Size     int    `json:"cluster_size"`
				Replicas []Peer `json:"replicas"`
			} `json:"meta_cluster"`
			Accounts []struct {
				Name    string `json:"name"`
				Streams []struct {
					Name    string      `json:"name"`
					Cluster *jszCluster `json:"cluster"`
					State   struct {
						Msgs      uint64 `json:"messages"`
						Bytes     uint64 `json:"bytes"`
						Consumers int    `json:"consumer_count"`
					} `json:"state"`
					Consumers []struct {
						Name           string      `json:"name"`
						Cluster        *jszCluster `json:"cluster"`
						NumPending     uint64      `json:"num_pending"`
						NumAckPending  int         `json:"num_ack_pending"`
						NumRedelivered int         `json:"num_redelivered"`
					} `json:"consumer_detail"`
				} `json:"stream_detail"`
			} `json:"account_details"`
		}
		if r.Error != nil || json.Unmarshal(r.Data, &d) != nil {
			continue
		}
		me := r.Server.Name
		// The meta leader's view is the one with every peer's state.
		if d.Meta != nil && (out.Meta == nil || d.Meta.Leader == me) {
			out.Meta = &MetaGroup{Leader: d.Meta.Leader, Size: d.Meta.Size, Peers: d.Meta.Replicas}
		}
		for _, a := range d.Accounts {
			for _, s := range a.Streams {
				key := a.Name + "/" + s.Name
				leader, peers := "", []Peer(nil)
				if s.Cluster != nil {
					leader, peers = s.Cluster.Leader, s.Cluster.Replicas
				}
				// Keep the leader's report; without a leader, any replica's.
				if _, seen := streams[key]; !seen || leader == me {
					streams[key] = StreamHealth{
						Account: a.Name, Name: s.Name, Leader: leader, Replicas: peers,
						Clustered: s.Cluster != nil && (len(peers) > 0 || leader != ""),
						Messages:  s.State.Msgs, Bytes: s.State.Bytes, Consumers: s.State.Consumers,
					}
				}
				for _, c := range s.Consumers {
					ckey := key + "/" + c.Name
					cl := ""
					var cp []Peer
					if c.Cluster != nil {
						cl, cp = c.Cluster.Leader, c.Cluster.Replicas
					}
					// Only a consumer's own leader tracks pending.
					if _, seen := consumers[ckey]; !seen || cl == me {
						consumers[ckey] = ConsumerHealth{
							Account: a.Name, Stream: s.Name, Name: c.Name, Leader: cl, Replicas: cp,
							Pending: c.NumPending, AckPending: c.NumAckPending, Redelivered: c.NumRedelivered,
						}
					}
				}
			}
		}
	}
	for _, s := range streams {
		out.Streams = append(out.Streams, s)
	}
	for _, c := range consumers {
		out.Consumers = append(out.Consumers, c)
	}
	sortHealth(out)
	return out, nil
}

func sortHealth(h *JetStreamHealth) {
	sort.Slice(h.Streams, func(i, j int) bool {
		return h.Streams[i].Account+"/"+h.Streams[i].Name < h.Streams[j].Account+"/"+h.Streams[j].Name
	})
	// Biggest backlog first: that's what people look for.
	sort.Slice(h.Consumers, func(i, j int) bool { return h.Consumers[i].Pending > h.Consumers[j].Pending })
}

func peers(ci *jetstream.ClusterInfo) (string, []Peer, bool) {
	if ci == nil {
		return "", nil, false
	}
	var out []Peer
	for _, p := range ci.Replicas {
		out = append(out, Peer{Name: p.Name, Current: p.Current, Offline: p.Offline, Active: p.Active, Lag: p.Lag})
	}
	return ci.Leader, out, len(out) > 0 || ci.Leader != ""
}

// accountJetStreamHealth reads the user's own streams and consumers through
// the JetStream API, which carries the same replica state.
func (m *Monitor) accountJetStreamHealth() (*JetStreamHealth, error) {
	js, err := jetstream.New(m.nc)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out := &JetStreamHealth{Source: Account, Streams: []StreamHealth{}, Consumers: []ConsumerHealth{}}
	list := js.ListStreams(ctx)
	for si := range list.Info() {
		leader, ps, clustered := peers(si.Cluster)
		out.Streams = append(out.Streams, StreamHealth{
			Name: si.Config.Name, Leader: leader, Replicas: ps, Clustered: clustered,
			Messages: si.State.Msgs, Bytes: si.State.Bytes, Consumers: si.State.Consumers,
		})
		st, err := js.Stream(ctx, si.Config.Name)
		if err != nil {
			return nil, err
		}
		cl := st.ListConsumers(ctx)
		for ci := range cl.Info() {
			l, cp, _ := peers(ci.Cluster)
			out.Consumers = append(out.Consumers, ConsumerHealth{
				Stream: si.Config.Name, Name: ci.Name, Leader: l, Replicas: cp,
				Pending: ci.NumPending, AckPending: ci.NumAckPending, Redelivered: ci.NumRedelivered,
			})
		}
		if err := cl.Err(); err != nil {
			return nil, err
		}
	}
	if err := list.Err(); err != nil {
		return nil, err
	}
	sortHealth(out)
	return out, nil
}
