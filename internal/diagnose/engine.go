package diagnose

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/astergaze-solutions/heynats/internal/monitor"
)

const (
	slowWindow   = 5 * time.Minute
	lagThreshold = 100 // replica operations behind before it counts as lagging
)

type ServerCheck struct {
	Name       string `json:"name"`
	Responding bool   `json:"responding"`
	Health     string `json:"health,omitempty"`
}

type RuleInfo struct {
	ID        string `json:"id"`
	Doc       string `json:"doc"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// Report is the latest check: a verdict, per-server checks and open findings.
type Report struct {
	Verdict   string         `json:"verdict"` // healthy, degraded or critical
	CheckedAt time.Time      `json:"checked_at"`
	Source    monitor.Source `json:"source"`
	Servers   []ServerCheck  `json:"servers"`
	Findings  []Finding      `json:"findings"`
	Rules     []RuleInfo     `json:"rules"`
}

type pendingPoint struct {
	at    time.Time
	count uint64
}

type slowPoint struct {
	at    time.Time
	count int64
}

// Diagnoser checks one connection's cluster on an interval and keeps
// findings open across checks, so first_seen survives a flapping issue.
type Diagnoser struct {
	mon *monitor.Monitor

	checkMu sync.Mutex // one check at a time
	mu      sync.Mutex
	missed  map[string]int
	slow    map[string][]slowPoint
	open    map[string]Finding
	report  *Report
	// How long lag must last, and how long pending must only grow, to count.
	lagFor, growFor time.Duration
	lagSince        map[string]time.Time
	pending         map[string][]pendingPoint

	stop, done chan struct{}
}

func Start(mon *monitor.Monitor, interval time.Duration) *Diagnoser {
	return start(mon, interval, time.Minute, 15*time.Minute)
}

// start lets tests shorten the lag and growth windows.
func start(mon *monitor.Monitor, interval, lagFor, growFor time.Duration) *Diagnoser {
	d := &Diagnoser{
		mon: mon, missed: map[string]int{}, slow: map[string][]slowPoint{}, open: map[string]Finding{},
		lagFor: lagFor, growFor: growFor,
		lagSince: map[string]time.Time{}, pending: map[string][]pendingPoint{},
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	go func() {
		defer close(d.done)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			d.Check(time.Now())
			select {
			case <-d.stop:
				return
			case <-t.C:
			}
		}
	}()
	return d
}

func (d *Diagnoser) Stop() {
	select {
	case <-d.stop:
	default:
		close(d.stop)
	}
	<-d.done
}

// Report returns the latest check, running one first if none finished yet.
func (d *Diagnoser) Report() *Report {
	d.mu.Lock()
	r := d.report
	d.mu.Unlock()
	if r == nil {
		return d.Check(time.Now())
	}
	return r
}

func (d *Diagnoser) Check(now time.Time) *Report {
	d.checkMu.Lock()
	defer d.checkMu.Unlock()
	snap, err := d.snapshot(now)
	if err != nil {
		// Nothing answered at all: every known server missed this check.
		slog.Debug("diagnose: snapshot failed", "err", err)
		snap = Snapshot{System: d.mon.Source() == monitor.System}
		for name := range d.mon.Known() {
			snap.Servers = append(snap.Servers, Server{Name: name, Missed: d.miss(name, false)})
		}
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	still := map[string]Finding{}
	for _, f := range Run(snap) {
		key := f.RuleID + "|" + f.Resource
		f.FirstSeen, f.LastSeen = now, now
		if prev, ok := d.open[key]; ok {
			f.FirstSeen = prev.FirstSeen
		}
		still[key] = f
	}
	d.open = still

	r := &Report{Verdict: "healthy", CheckedAt: now, Source: d.mon.Source(), Findings: []Finding{}}
	for _, f := range still {
		r.Findings = append(r.Findings, f)
		switch {
		case f.Severity == Critical:
			r.Verdict = "critical"
		case f.Severity == Warning && r.Verdict == "healthy":
			r.Verdict = "degraded"
		}
	}
	rank := map[Severity]int{Critical: 0, Warning: 1, Info: 2}
	sort.Slice(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		if rank[a.Severity] != rank[b.Severity] {
			return rank[a.Severity] < rank[b.Severity]
		}
		return a.Resource < b.Resource
	})
	for _, s := range snap.Servers {
		r.Servers = append(r.Servers, ServerCheck{Name: s.Name, Responding: s.Missed == 0, Health: s.Health})
	}
	sort.Slice(r.Servers, func(i, j int) bool { return r.Servers[i].Name < r.Servers[j].Name })
	for _, rule := range Rules {
		info := RuleInfo{ID: rule.ID, Doc: rule.Doc, Available: !rule.SystemOnly || snap.System}
		if !info.Available {
			info.Reason = "Requires a system account user"
		}
		r.Rules = append(r.Rules, info)
	}
	d.report = r
	return r
}

// miss records whether a server answered and returns its missed-in-a-row count.
func (d *Diagnoser) miss(name string, answered bool) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if answered {
		d.missed[name] = 0
	} else {
		d.missed[name]++
	}
	return d.missed[name]
}

// slowDelta records the cumulative count and returns its rise over 5 minutes.
// A drop means the server restarted, which starts the history over.
func (d *Diagnoser) slowDelta(name string, now time.Time, count int64) int64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	pts := d.slow[name]
	if n := len(pts); n > 0 && count < pts[n-1].count {
		pts = nil
	}
	pts = append(pts, slowPoint{now, count})
	for len(pts) > 1 && now.Sub(pts[0].at) > slowWindow {
		pts = pts[1:]
	}
	d.slow[name] = pts
	return count - pts[0].count
}

// addJetStream fills streams and consumers, tracking how long each replica
// has lagged and whether each consumer's pending has only grown.
func (d *Diagnoser) addJetStream(snap *Snapshot, now time.Time) {
	h, err := d.mon.JetStreamHealth()
	if err != nil {
		return // no JetStream for this account, or it didn't answer: nothing to judge
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	lagging := map[string]bool{}
	for _, st := range h.Streams {
		name := st.Name
		if st.Account != "" {
			name = st.Account + "/" + st.Name
		}
		s := Stream{Name: name, Clustered: st.Clustered, Leader: st.Leader}
		for _, p := range st.Replicas {
			key := name + "|" + p.Name
			switch {
			case p.Offline:
				s.Offline = append(s.Offline, p.Name)
			case p.Lag > lagThreshold:
				lagging[key] = true
				if _, ok := d.lagSince[key]; !ok {
					d.lagSince[key] = now
				}
				if now.Sub(d.lagSince[key]) >= d.lagFor {
					s.Lagging = append(s.Lagging, fmt.Sprintf("%s (%d behind)", p.Name, p.Lag))
				}
			}
		}
		snap.Streams = append(snap.Streams, s)
	}
	for key := range d.lagSince {
		if !lagging[key] {
			delete(d.lagSince, key)
		}
	}

	seen := map[string]bool{}
	for _, c := range h.Consumers {
		name := c.Stream + "/" + c.Name
		if c.Account != "" {
			name = c.Account + "/" + name
		}
		seen[name] = true
		pts := d.pending[name]
		if n := len(pts); n > 0 && c.Pending < pts[n-1].count {
			pts = nil // it caught up a little: start over
		}
		pts = append(pts, pendingPoint{now, c.Pending})
		for len(pts) > 1 && now.Sub(pts[1].at) >= d.growFor {
			pts = pts[1:]
		}
		d.pending[name] = pts
		first := pts[0]
		snap.Consumers = append(snap.Consumers, Consumer{
			Name: name, Pending: c.Pending, Since: first.count,
			Growing: now.Sub(first.at) >= d.growFor && c.Pending > first.count,
		})
	}
	for name := range d.pending {
		if !seen[name] {
			delete(d.pending, name)
		}
	}
}

func (d *Diagnoser) snapshot(now time.Time) (Snapshot, error) {
	if d.mon.Source() == monitor.System {
		list, err := d.mon.Servers()
		if err != nil {
			return Snapshot{}, err
		}
		snap := Snapshot{System: true}
		for _, s := range list.Servers {
			sv := Server{
				Name: s.Name, Cluster: s.Cluster, Version: s.Version, Responding: s.Responding,
				Missed: d.miss(s.Name, s.Responding), Health: s.Health,
				Connections: s.Connections, MaxConnections: s.MaxConnections,
				JetStream: s.JetStream, Storage: s.Storage, MaxStorage: s.MaxStorage,
				MetaLeader: s.MetaLeader, MetaClusterSize: s.MetaClusterSize,
			}
			if s.Responding {
				sv.SlowDelta = d.slowDelta(s.Name, now, s.SlowConsumers)
			}
			snap.Servers = append(snap.Servers, sv)
		}
		d.addJetStream(&snap, now)
		return snap, nil
	}

	statz, err := d.mon.Ping("STATZ")
	if err != nil {
		return Snapshot{}, err
	}
	answered := map[string]bool{}
	for _, r := range statz.Replies {
		answered[r.Server.Name] = true
	}
	snap := Snapshot{}
	for name := range d.mon.Known() {
		snap.Servers = append(snap.Servers, Server{Name: name, Responding: answered[name], Missed: d.miss(name, answered[name])})
	}
	// Account without JetStream: nothing to check for storage.
	if js, err := d.mon.JetStream(); err == nil {
		snap.AccountStorage, snap.AccountMaxStorage = js.Storage, js.MaxStorage
	}
	d.addJetStream(&snap, now)
	return snap, nil
}
