package monitor

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/astergaze-solutions/heynats/internal/util"
)

type Feature struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

type Capabilities struct {
	Source        Source             `json:"source"`
	ServerVersion string             `json:"server_version"`
	Servers       int                `json:"servers"`
	Expected      int                `json:"expected"`
	Partial       bool               `json:"partial"`
	Features      map[string]Feature `json:"features"`
}

// requirements is the one place that decides what each feature needs;
// the UI only reads the result.
var requirements = map[string]struct {
	system     bool
	minVersion [3]int
}{
	"servers":                  {system: true},
	"connections.all_accounts": {system: true},
	"connections.kick":         {system: true},
	"trace":                    {minVersion: [3]int{2, 11, 0}},
}

func (m *Monitor) Capabilities() (*Capabilities, error) {
	statz, err := m.Ping("STATZ")
	if err != nil {
		return nil, err
	}
	caps := &Capabilities{
		Source:        statz.Source,
		ServerVersion: m.nc.ConnectedServerVersion(),
		Servers:       len(statz.Replies),
		Expected:      statz.Expected,
		Partial:       statz.Partial,
		Features:      map[string]Feature{},
	}
	for name, req := range requirements {
		v := req.minVersion
		switch {
		case req.system && caps.Source != System:
			caps.Features[name] = Feature{Reason: "Requires a system account user"}
		case v != [3]int{} && !util.ServerMinVersion(m.nc, v[0], v[1], v[2]):
			caps.Features[name] = Feature{Reason: fmt.Sprintf("Requires NATS %d.%d", v[0], v[1])}
		default:
			caps.Features[name] = Feature{OK: true}
		}
	}
	return caps, nil
}

// Connz is CONNZ summed over all servers. Connections stay raw so no field
// the UI shows gets dropped.
type Connz struct {
	Source      Source            `json:"source"`
	Partial     bool              `json:"partial"`
	Servers     int               `json:"servers"`
	Expected    int               `json:"expected"`
	Now         time.Time         `json:"now"`
	NumConns    int               `json:"num_connections"`
	Total       int               `json:"total"`
	Limit       int               `json:"limit"`
	Connections []json.RawMessage `json:"connections"`
	Errors      []string          `json:"errors,omitempty"`
}

func (m *Monitor) Connz() (*Connz, error) {
	res, err := m.Ping("CONNZ")
	if err != nil {
		return nil, err
	}
	out := &Connz{Source: res.Source, Partial: res.Partial, Servers: len(res.Replies), Expected: res.Expected, Connections: []json.RawMessage{}}
	for _, r := range res.Replies {
		if r.Error != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("%s: %s", r.Server.Name, r.Error.Description))
			continue
		}
		var d struct {
			Now         time.Time         `json:"now"`
			NumConns    int               `json:"num_connections"`
			Total       int               `json:"total"`
			Limit       int               `json:"limit"`
			Connections []json.RawMessage `json:"connections"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, fmt.Errorf("CONNZ from %s: %w", r.Server.Name, err)
		}
		if d.Now.After(out.Now) {
			out.Now = d.Now
		}
		out.NumConns += d.NumConns
		out.Total += d.Total
		out.Limit += d.Limit
		out.Connections = append(out.Connections, d.Connections...)
	}
	return out, nil
}
