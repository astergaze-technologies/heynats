package monitor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Conn is one client connection, tagged with the server it is on (cid is
// only unique per server).
type Conn struct {
	Server        string    `json:"server"`
	CID           uint64    `json:"cid"`
	Name          string    `json:"name,omitempty"`
	Account       string    `json:"account,omitempty"`
	User          string    `json:"authorized_user,omitempty"`
	Lang          string    `json:"lang,omitempty"`
	Version       string    `json:"version,omitempty"`
	IP            string    `json:"ip"`
	Port          int       `json:"port"`
	Start         time.Time `json:"start"`
	LastActivity  time.Time `json:"last_activity"`
	Stop          time.Time `json:"stop,omitzero"`
	Reason        string    `json:"reason,omitempty"`
	RTT           string    `json:"rtt,omitempty"`
	Uptime        string    `json:"uptime"`
	Idle          string    `json:"idle"`
	Pending       int       `json:"pending_bytes"`
	InMsgs        int64     `json:"in_msgs"`
	OutMsgs       int64     `json:"out_msgs"`
	InBytes       int64     `json:"in_bytes"`
	OutBytes      int64     `json:"out_bytes"`
	Subscriptions uint32    `json:"subscriptions"`
}

type ConnState int

const (
	Open ConnState = iota
	Closed
)

// ConnQuery: Account and User are filtered by the servers; Server, Name, IP
// and Lang are substring filters applied here, after the merge.
type ConnQuery struct {
	State                  ConnState
	Sort                   string
	Offset, Limit          int
	Account, User          string
	Server, Name, IP, Lang string
	SlowOnly               bool // closed for being slow consumers
}

type ConnPage struct {
	Source   Source `json:"source"`
	Partial  bool   `json:"partial"`
	Servers  int    `json:"servers"`
	Expected int    `json:"expected"`
	// Total is what the servers hold before our own filters; Matched is how
	// many of the scanned rows passed them.
	Total   int    `json:"total"`
	Matched int    `json:"matched"`
	Conns   []Conn `json:"connections"`
}

// less orders connections the way CONNZ sorts them, so merging each server's
// top rows gives the true top rows overall.
var connSorts = map[string]func(a, b *Conn) bool{
	"cid":        func(a, b *Conn) bool { return a.CID < b.CID },
	"start":      func(a, b *Conn) bool { return a.Start.Before(b.Start) },
	"subs":       func(a, b *Conn) bool { return a.Subscriptions > b.Subscriptions },
	"pending":    func(a, b *Conn) bool { return a.Pending > b.Pending },
	"msgs_to":    func(a, b *Conn) bool { return a.OutMsgs > b.OutMsgs },
	"msgs_from":  func(a, b *Conn) bool { return a.InMsgs > b.InMsgs },
	"bytes_to":   func(a, b *Conn) bool { return a.OutBytes > b.OutBytes },
	"bytes_from": func(a, b *Conn) bool { return a.InBytes > b.InBytes },
	"last":       func(a, b *Conn) bool { return a.LastActivity.After(b.LastActivity) },
	"idle":       func(a, b *Conn) bool { return a.LastActivity.Before(b.LastActivity) },
	"uptime":     func(a, b *Conn) bool { return a.Start.Before(b.Start) },
	"stop":       func(a, b *Conn) bool { return a.Stop.After(b.Stop) },
}

const maxScan = 5000 // rows asked from each server per request

// MergeConns merges per-server pages already sorted by key, applies the local
// filters and returns rows [offset, offset+limit).
func MergeConns(perServer [][]Conn, q ConnQuery) (page []Conn, matched int) {
	less := connSorts[q.Sort]
	var all []Conn
	for _, rows := range perServer {
		all = append(all, rows...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if less(&all[i], &all[j]) {
			return true
		}
		if less(&all[j], &all[i]) {
			return false
		}
		// Ties: by server then cid, so pages are stable between requests.
		if all[i].Server != all[j].Server {
			return all[i].Server < all[j].Server
		}
		return all[i].CID < all[j].CID
	})
	contains := func(field, want string) bool {
		return want == "" || strings.Contains(strings.ToLower(field), strings.ToLower(want))
	}
	var kept []Conn
	for _, c := range all {
		if contains(c.Server, q.Server) && contains(c.Name, q.Name) &&
			contains(fmt.Sprintf("%s:%d", c.IP, c.Port), q.IP) && contains(c.Lang+" "+c.Version, q.Lang) &&
			(!q.SlowOnly || strings.Contains(c.Reason, "Slow Consumer")) {
			kept = append(kept, c)
		}
	}
	if q.Offset >= len(kept) {
		return []Conn{}, len(kept)
	}
	return kept[q.Offset:min(len(kept), q.Offset+q.Limit)], len(kept)
}

func (m *Monitor) Connections(q ConnQuery) (*ConnPage, error) {
	if _, ok := connSorts[q.Sort]; !ok {
		return nil, fmt.Errorf("%w: unknown sort %q", ErrBadRequest, q.Sort)
	}
	if q.Sort == "stop" && q.State != Closed {
		return nil, fmt.Errorf("%w: sort by stop needs closed connections", ErrBadRequest)
	}
	if q.Limit <= 0 || q.Offset < 0 {
		return nil, fmt.Errorf("%w: bad offset or limit", ErrBadRequest)
	}
	scan := q.Offset + q.Limit
	if q.Server != "" || q.Name != "" || q.IP != "" || q.Lang != "" || q.SlowOnly {
		// shortcut: local filters scan each server's top maxScan rows only;
		// add server-side filters to CONNZ if clusters outgrow that.
		scan = maxScan
	}
	opts := map[string]any{
		"sort": q.Sort, "state": int(q.State), "offset": 0, "limit": min(scan, maxScan),
		"auth": true, "subscriptions": false,
	}
	if q.Account != "" {
		opts["acc"] = q.Account
	}
	if q.User != "" {
		opts["user"] = q.User
	}
	res, err := m.PingWith("CONNZ", opts)
	if err != nil {
		return nil, err
	}
	out := &ConnPage{Source: res.Source, Partial: res.Partial, Servers: len(res.Replies), Expected: res.Expected}
	var perServer [][]Conn
	for _, r := range res.Replies {
		if r.Error != nil {
			continue
		}
		var d struct {
			Total int    `json:"total"`
			Conns []Conn `json:"connections"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, fmt.Errorf("CONNZ from %s: %w", r.Server.Name, err)
		}
		out.Total += d.Total
		for i := range d.Conns {
			d.Conns[i].Server = r.Server.Name
		}
		perServer = append(perServer, d.Conns)
	}
	out.Conns, out.Matched = MergeConns(perServer, q)
	return out, nil
}

// Subscription is one entry of a connection's subscription list.
type Subscription struct {
	Subject string `json:"subject"`
	Queue   string `json:"qgroup,omitempty"`
	Msgs    int64  `json:"msgs"`
	Max     int64  `json:"max,omitempty"`
	SID     string `json:"sid"`
}

// ConnDetail is one connection as its server reports it, with subscriptions.
type ConnDetail struct {
	Conn
	ServerID   string         `json:"server_id"` // for "copy as CLI"
	TLSVersion string         `json:"tls_version,omitempty"`
	Subs       []Subscription `json:"subscriptions_list_detail"`
}

// Connection looks up cid on the named server, open or closed.
func (m *Monitor) Connection(server string, cid uint64) (*ConnDetail, error) {
	res, err := m.PingWith("CONNZ", map[string]any{"cid": cid, "state": 2, "subscriptions_detail": true, "auth": true})
	if err != nil {
		return nil, err
	}
	for _, r := range res.Replies {
		if r.Server.Name != server || r.Error != nil {
			continue
		}
		var d struct {
			Conns []ConnDetail `json:"connections"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, err
		}
		if len(d.Conns) > 0 {
			c := d.Conns[0]
			c.Server, c.ServerID = server, r.Server.ID
			return &c, nil
		}
	}
	return nil, fmt.Errorf("%w: no connection %d on %s", ErrNotFound, cid, server)
}

// ErrNotAllowed carries the user-facing reason a feature is unavailable.
type ErrNotAllowed struct{ Reason string }

func (e ErrNotAllowed) Error() string { return e.Reason }

// Kick disconnects one client. The server name is looked up among servers
// that have answered, so only a real server id goes into the subject.
func (m *Monitor) Kick(server string, cid uint64) error {
	if f := m.Feature("connections.kick"); !f.OK {
		return ErrNotAllowed{f.Reason}
	}
	s, ok := m.Known()[server]
	if !ok {
		// A fresh session hasn't heard from any server yet: ask once.
		if _, err := m.Ping("STATZ"); err != nil {
			return err
		}
		if s, ok = m.Known()[server]; !ok {
			return fmt.Errorf("%w: unknown server %q", ErrNotFound, server)
		}
	}
	body, _ := json.Marshal(map[string]uint64{"cid": cid})
	msg, err := m.nc.Request("$SYS.REQ.SERVER."+s.ID+".KICK", body, hardWait)
	if err != nil {
		return err
	}
	var r Reply
	if err := json.Unmarshal(msg.Data, &r); err != nil {
		return err
	}
	if r.Error != nil {
		return fmt.Errorf("%w: %s", ErrNotFound, r.Error.Description)
	}
	return nil
}
