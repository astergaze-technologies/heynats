// Package monitor reads server, connection and JetStream data over NATS,
// from the system account when the user has it, else from its own account.
package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
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
	Source  Source  `json:"source"`
	Partial bool    `json:"partial"`
	Replies []Reply `json:"replies"`
}

type Monitor struct {
	nc *nats.Conn

	mu     sync.Mutex
	source Source
	// shortcut: highest server count seen; a node removed for good keeps
	// results partial until reconnect. Use cluster membership once we track it.
	expected int
}

func New(nc *nats.Conn) *Monitor { return &Monitor{nc: nc} }

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
func (m *Monitor) Ping(kind string) (*Result, error) {
	src := m.Source()
	var subject string
	var body []byte
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
	res.Partial = len(res.Replies) < m.expected
	m.expected = max(m.expected, len(res.Replies))
	m.mu.Unlock()
	return res, nil
}
