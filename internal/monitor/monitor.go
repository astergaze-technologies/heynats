// Package monitor reads server, connection and JetStream data over NATS,
// from the system account when the user has it, else from its own account.
package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type Source string

const (
	System  Source = "system"  // $SYS.REQ.SERVER.PING.*: every server, every account
	Account Source = "account" // $SYS.REQ.ACCOUNT.PING.*: own account only
)

const (
	quietWait = 300 * time.Millisecond
	hardWait  = 2 * time.Second
)

type ServerInfo struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Cluster string `json:"cluster,omitempty"`
	Version string `json:"ver"`
}

// Reply is one server's answer; Data is kept raw for "Raw" tabs.
type Reply struct {
	Server ServerInfo      `json:"server"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  *struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"error,omitempty"`
}

type Result struct {
	Source   Source  `json:"source"`
	Partial  bool    `json:"partial"`
	Expected int     `json:"expected"` // most servers ever seen
	Replies  []Reply `json:"replies"`
}

type Monitor struct {
	nc *nats.Conn

	mu     sync.Mutex
	source Source
	// shortcut: every server ever seen, by name; one removed for good keeps
	// results partial until reconnect. Use cluster membership once we track it.
	known map[string]ServerInfo
}

func New(nc *nats.Conn) *Monitor { return &Monitor{nc: nc, known: map[string]ServerInfo{}} }

// Known returns every server that has answered since connecting.
func (m *Monitor) Known() map[string]ServerInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return maps.Clone(m.known)
}

// Source reports which data source this connection can use; checked once.
func (m *Monitor) Source() Source {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.source == "" {
		m.source = Account
		// Only system-account users can reach the server ping subjects.
		if _, err := m.nc.Request("$SYS.REQ.SERVER.PING", nil, time.Second); err == nil {
			m.source = System
		}
	}
	return m.source
}

// Ping asks every server for kind (VARZ, CONNZ, STATZ, ...) and collects the
// replies until 300ms pass without one, or 2s in total. Fewer replies than
// the most ever seen marks the result partial.
func (m *Monitor) Ping(kind string) (*Result, error) { return m.PingWith(kind, nil) }

// PingWith is Ping with request options, e.g. {"accounts": true} for JSZ.
func (m *Monitor) PingWith(kind string, opts any) (*Result, error) {
	src := m.Source()
	var subject string
	var body []byte
	if opts != nil {
		var err error
		if body, err = json.Marshal(opts); err != nil {
			return nil, err
		}
	}
	switch {
	case src == Account:
		subject = "$SYS.REQ.ACCOUNT.PING." + kind
		if kind == "STATZ" { // else only servers with clients of this account answer
			body = []byte(`{"include_unused":true}`)
		}
	case kind == "STATZ": // server STATZ has no suffix
		subject = "$SYS.REQ.SERVER.PING"
	default:
		subject = "$SYS.REQ.SERVER.PING." + kind
	}
	return m.gather(subject, body)
}

// gather publishes one request on a ping subject and collects every
// server's reply (see Ping for when it stops).
func (m *Monitor) gather(subject string, body []byte) (*Result, error) {
	src := m.Source()
	inbox := m.nc.NewInbox()
	sub, err := m.nc.SubscribeSync(inbox)
	if err != nil {
		return nil, err
	}
	defer sub.Unsubscribe() //nolint:errcheck
	if err := m.nc.PublishRequest(subject, inbox, body); err != nil {
		return nil, err
	}

	res := &Result{Source: src}
	deadline := time.Now().Add(hardWait)
	wait := hardWait
	// shortcut: always waits out the quiet window (+300ms per call), because
	// stopping at the known count would never discover a node that joins.
	// Stop early once we know cluster membership for sure.
	for {
		msg, err := sub.NextMsg(min(wait, time.Until(deadline)))
		if errors.Is(err, nats.ErrTimeout) {
			break
		}
		if err != nil {
			return nil, err
		}
		if msg.Header.Get("Status") == "503" {
			return nil, fmt.Errorf("%s: %w", subject, nats.ErrNoResponders)
		}
		var r Reply
		if err := json.Unmarshal(msg.Data, &r); err != nil {
			return nil, fmt.Errorf("%s: bad reply: %w", subject, err)
		}
		res.Replies = append(res.Replies, r)
		wait = quietWait
	}
	if len(res.Replies) == 0 {
		return nil, fmt.Errorf("%s: no server answered", subject)
	}

	m.mu.Lock()
	for _, r := range res.Replies {
		m.known[r.Server.Name] = r.Server
	}
	res.Partial = len(res.Replies) < len(m.known)
	res.Expected = len(m.known)
	m.mu.Unlock()
	return res, nil
}
