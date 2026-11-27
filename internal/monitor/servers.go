package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"
)

var ErrSystemOnly = errors.New("requires a system account user")

// ServerRow is one line of the servers table. A known server that didn't
// answer this round is listed with Responding false, not dropped.
type ServerRow struct {
	Name          string    `json:"name"`
	ID            string    `json:"id"`
	Cluster       string    `json:"cluster,omitempty"`
	Version       string    `json:"version"`
	Responding    bool      `json:"responding"`
	Health        string    `json:"health,omitempty"` // "ok" or the server's reason
	Start         time.Time `json:"start,omitzero"`
	CPU           float64   `json:"cpu"`
	Mem           int64     `json:"mem"`
	Connections   int       `json:"connections"`
	Subscriptions uint32    `json:"subscriptions"`
	SlowConsumers int64     `json:"slow_consumers"`
	JetStream     bool      `json:"jetstream"`
	// For diagnostics: limits and JetStream state as this server sees them.
	MaxConnections  int      `json:"max_connections"`
	Storage         uint64   `json:"js_storage"`
	MaxStorage      int64    `json:"js_max_storage"`
	MetaLeader      string   `json:"js_meta_leader,omitempty"`
	MetaClusterSize int      `json:"js_meta_cluster_size,omitempty"`
	MsgsIn          *float32 `json:"msgs_in"` // per second, from the sampler
	MsgsOut         *float32 `json:"msgs_out"`
}

type ServerList struct {
	Partial  bool        `json:"partial"`
	Expected int         `json:"expected"`
	Servers  []ServerRow `json:"servers"`
}

func (m *Monitor) Servers() (*ServerList, error) {
	if m.Source() != System {
		return nil, ErrSystemOnly
	}
	var health *Result
	var healthErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		health, healthErr = m.Ping("HEALTHZ")
	}()
	varz, err := m.Ping("VARZ")
	<-done
	if err != nil {
		return nil, err
	}

	rows := map[string]*ServerRow{}
	for name, s := range m.Known() {
		rows[name] = &ServerRow{Name: name, ID: s.ID, Cluster: s.Cluster, Version: s.Version}
	}
	for _, r := range varz.Replies {
		row := rows[r.Server.Name]
		row.Responding = true
		var v struct {
			Start         time.Time `json:"start"`
			CPU           float64   `json:"cpu"`
			Mem           int64     `json:"mem"`
			Connections   int       `json:"connections"`
			Subscriptions uint32    `json:"subscriptions"`
			SlowConsumers int64     `json:"slow_consumers"`
			MaxConns      int       `json:"max_connections"`
			JetStream     struct {
				Config *struct {
					MaxStorage int64 `json:"max_storage"`
				} `json:"config"`
				Stats struct {
					Storage uint64 `json:"storage"`
				} `json:"stats"`
				Meta *struct {
					Leader      string `json:"leader"`
					ClusterSize int    `json:"cluster_size"`
				} `json:"meta"`
			} `json:"jetstream"`
		}
		if r.Error == nil && json.Unmarshal(r.Data, &v) == nil {
			row.Start, row.CPU, row.Mem = v.Start, v.CPU, v.Mem
			row.Connections, row.Subscriptions, row.SlowConsumers = v.Connections, v.Subscriptions, v.SlowConsumers
			row.JetStream = v.JetStream.Config != nil
			row.MaxConnections = v.MaxConns
			if row.JetStream {
				row.Storage, row.MaxStorage = v.JetStream.Stats.Storage, v.JetStream.Config.MaxStorage
			}
			if m := v.JetStream.Meta; m != nil {
				row.MetaLeader, row.MetaClusterSize = m.Leader, m.ClusterSize
			}
		}
	}
	if healthErr == nil {
		for _, r := range health.Replies {
			row := rows[r.Server.Name]
			var h struct {
				Status string `json:"status"`
				Error  string `json:"error"`
			}
			switch {
			case r.Error != nil:
				row.Health = r.Error.Description
			case json.Unmarshal(r.Data, &h) == nil && h.Error != "":
				row.Health = h.Error
			default:
				row.Health = h.Status
			}
		}
	}

	out := &ServerList{Partial: varz.Partial, Expected: varz.Expected}
	for _, row := range rows {
		out.Servers = append(out.Servers, *row)
	}
	sort.Slice(out.Servers, func(i, j int) bool { return out.Servers[i].Name < out.Servers[j].Name })
	return out, nil
}

// Server IDs are public nkeys: "N" plus 55 base32 characters. Checked so an
// id can never add tokens or wildcards to the request subject.
var serverID = regexp.MustCompile(`^N[A-Z2-7]{55}$`)

var rawKinds = map[string]bool{
	"VARZ": true, "HEALTHZ": true, "ROUTEZ": true, "GATEWAYZ": true,
	"LEAFZ": true, "JSZ": true, "SUBSZ": true, "CONNZ": true, "ACCOUNTZ": true,
}

var (
	ErrBadRequest = errors.New("bad request")
	ErrNotFound   = errors.New("not found")
)

// ServerRaw asks one server for one kind and returns its data unchanged.
func (m *Monitor) ServerRaw(id, kind string) (json.RawMessage, error) {
	if !serverID.MatchString(id) || !rawKinds[kind] {
		return nil, fmt.Errorf("%w: unknown server id or kind", ErrBadRequest)
	}
	if m.Source() != System {
		return nil, ErrSystemOnly
	}
	msg, err := m.nc.Request("$SYS.REQ.SERVER."+id+"."+kind, nil, hardWait)
	if err != nil {
		return nil, err
	}
	var r Reply
	if err := json.Unmarshal(msg.Data, &r); err != nil {
		return nil, err
	}
	if r.Error != nil {
		return nil, errors.New(r.Error.Description)
	}
	return r.Data, nil
}
