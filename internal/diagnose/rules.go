// Package diagnose turns what HeyNATS knows about a cluster right now into
// findings: snapshot -> rules (pure functions) -> findings.
package diagnose

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Severity string

const (
	Critical Severity = "critical"
	Warning  Severity = "warning"
	Info     Severity = "info"
)

type Evidence struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Source string `json:"source"` // e.g. "VARZ", "HEALTHZ"
}

// Fix is something the user can do: open a page, or copy a CLI command.
type Fix struct {
	Label string `json:"label"`
	Link  string `json:"link,omitempty"`
	CLI   string `json:"cli,omitempty"`
}

type Finding struct {
	RuleID    string     `json:"rule_id"`
	Severity  Severity   `json:"severity"`
	Resource  string     `json:"resource"` // e.g. "server:n2"
	Title     string     `json:"title"`
	Detail    string     `json:"detail"`
	Evidence  []Evidence `json:"evidence"`
	Fixes     []Fix      `json:"fixes,omitempty"`
	FirstSeen time.Time  `json:"first_seen"`
	LastSeen  time.Time  `json:"last_seen"`
}

// Server is one server as of this check. Missed counts checks in a row it
// didn't answer; SlowDelta is the slow-consumer increase over 5 minutes.
type Server struct {
	Name, Cluster, Version string
	Responding             bool
	Missed                 int
	Health                 string
	Connections            int
	MaxConnections         int
	SlowDelta              int64
	JetStream              bool
	Storage                uint64
	MaxStorage             int64
	MetaLeader             string
	MetaClusterSize        int
}

// Stream is one stream's replica state. Lagging lists replicas that have
// been behind for longer than the engine's lag window.
type Stream struct {
	Name      string // account/stream, or stream in account scope
	Clustered bool
	Leader    string
	Offline   []string
	Lagging   []string
}

// Consumer: Growing is set by the engine when pending has only gone up over
// its growth window; Since is the pending count at the window's start.
type Consumer struct {
	Name    string
	Pending uint64
	Since   uint64
	Growing bool
}

// Snapshot is everything the rules may look at. System says whether the
// per-server fields beyond Name/Responding/Missed are known.
type Snapshot struct {
	System  bool
	Servers []Server
	// Account JetStream usage for users without the system account; limit -1 = none.
	AccountStorage    uint64
	AccountMaxStorage int64
	Streams           []Stream
	Consumers         []Consumer
}

type Rule struct {
	ID         string
	Doc        string
	SystemOnly bool
	Check      func(Snapshot) []Finding
}

const unreachableAfter = 2 // missed checks, about 10s at the 5s default

// Rules is the one list of everything HeyNATS checks.
var Rules = []Rule{
	{
		ID:  "server.unreachable",
		Doc: "A server that answered before has missed two checks in a row.",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			down := 0
			for _, sv := range s.Servers {
				if sv.Missed >= unreachableAfter {
					down++
				}
			}
			for _, sv := range s.Servers {
				if sv.Missed < unreachableAfter {
					continue
				}
				sev := Warning
				if down*2 >= len(s.Servers) { // half or more gone: quorum is at risk
					sev = Critical
				}
				out = append(out, Finding{
					RuleID: "server.unreachable", Severity: sev, Resource: "server:" + sv.Name,
					Title:  sv.Name + " is not responding",
					Detail: fmt.Sprintf("%s answered earlier but missed the last %d checks. Clients and replicas on it may be failing over.", sv.Name, sv.Missed),
					Evidence: []Evidence{
						{"Missed checks", fmt.Sprint(sv.Missed), "STATZ/VARZ"},
						{"Servers not responding", fmt.Sprintf("%d of %d", down, len(s.Servers)), "STATZ/VARZ"},
					},
					Fixes: []Fix{{Label: "List servers from the CLI", CLI: "nats server list"}},
				})
			}
			return out
		},
	},
	{
		ID: "server.healthz", SystemOnly: true,
		Doc: "A responding server reports itself unhealthy (for example JetStream not current).",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			for _, sv := range s.Servers {
				if !sv.Responding || sv.Health == "" || sv.Health == "ok" {
					continue
				}
				out = append(out, Finding{
					RuleID: "server.healthz", Severity: Warning, Resource: "server:" + sv.Name,
					Title:    sv.Name + " reports itself unhealthy",
					Detail:   sv.Health,
					Evidence: []Evidence{{"Health", sv.Health, "HEALTHZ"}},
					Fixes: []Fix{
						{Label: "Open server", Link: "/dashboard/servers/" + sv.Name},
						{Label: "JetStream report", CLI: "nats server report jetstream"},
					},
				})
			}
			return out
		},
	},
	{
		ID: "server.version_mismatch", SystemOnly: true,
		Doc: "Servers in one cluster run different NATS versions.",
		Check: func(s Snapshot) []Finding {
			byCluster := map[string]map[string][]string{}
			for _, sv := range s.Servers {
				if !sv.Responding || sv.Version == "" {
					continue
				}
				if byCluster[sv.Cluster] == nil {
					byCluster[sv.Cluster] = map[string][]string{}
				}
				byCluster[sv.Cluster][sv.Version] = append(byCluster[sv.Cluster][sv.Version], sv.Name)
			}
			var out []Finding
			for cluster, versions := range byCluster {
				if len(versions) < 2 {
					continue
				}
				var parts []string
				for v, names := range versions {
					sort.Strings(names)
					parts = append(parts, fmt.Sprintf("%s: %s", v, strings.Join(names, ", ")))
				}
				sort.Strings(parts)
				out = append(out, Finding{
					RuleID: "server.version_mismatch", Severity: Info, Resource: "cluster:" + cluster,
					Title:    "Servers run different NATS versions",
					Detail:   "Fine during a rolling upgrade; finish it so every server behaves the same.",
					Evidence: []Evidence{{"Versions", strings.Join(parts, "; "), "VARZ"}},
					Fixes:    []Fix{{Label: "Open servers", Link: "/dashboard/servers"}},
				})
			}
			return out
		},
	},
	{
		ID: "server.slow_consumers", SystemOnly: true,
		Doc: "The slow-consumer count went up in the last 5 minutes.",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			for _, sv := range s.Servers {
				if sv.SlowDelta <= 0 {
					continue
				}
				out = append(out, Finding{
					RuleID: "server.slow_consumers", Severity: Warning, Resource: "server:" + sv.Name,
					Title:    fmt.Sprintf("%s disconnected %d slow consumers", sv.Name, sv.SlowDelta),
					Detail:   "Clients that can't keep up get disconnected and lose messages. Look for subscribers that process too slowly or need a queue group.",
					Evidence: []Evidence{{"New slow consumers (5m)", fmt.Sprint(sv.SlowDelta), "VARZ"}},
					Fixes:    []Fix{{Label: "Connections by pending bytes", CLI: "nats server report connections --sort pending"}},
				})
			}
			return out
		},
	},
	{
		ID: "server.connections_near_limit", SystemOnly: true,
		Doc: "A server has more than 90% of max_connections in use.",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			for _, sv := range s.Servers {
				if sv.MaxConnections <= 0 || sv.Connections*10 <= sv.MaxConnections*9 {
					continue
				}
				out = append(out, Finding{
					RuleID: "server.connections_near_limit", Severity: Warning, Resource: "server:" + sv.Name,
					Title:    sv.Name + " is close to its connection limit",
					Detail:   "New clients will be refused at the limit. Raise max_connections or spread clients across servers.",
					Evidence: []Evidence{{"Connections", fmt.Sprintf("%d of %d", sv.Connections, sv.MaxConnections), "VARZ"}},
					Fixes:    []Fix{{Label: "Open server", Link: "/dashboard/servers/" + sv.Name}},
				})
			}
			return out
		},
	},
	{
		ID: "js.meta_no_leader", SystemOnly: true,
		Doc: "A clustered JetStream server sees no meta leader, so streams can't be created or changed.",
		Check: func(s Snapshot) []Finding {
			var seen []string
			for _, sv := range s.Servers {
				if sv.Responding && sv.JetStream && sv.MetaClusterSize > 1 && sv.MetaLeader == "" {
					seen = append(seen, sv.Name)
				}
			}
			if len(seen) == 0 {
				return nil
			}
			return []Finding{{
				RuleID: "js.meta_no_leader", Severity: Critical, Resource: "jetstream:meta",
				Title:    "JetStream has no meta leader",
				Detail:   "Without a meta leader no stream or consumer can be created, changed or deleted. Usually a lost quorum: bring stopped servers back.",
				Evidence: []Evidence{{"Servers without a leader", strings.Join(seen, ", "), "VARZ"}},
				Fixes:    []Fix{{Label: "JetStream report", CLI: "nats server report jetstream"}},
			}}
		},
	},
	{
		ID:  "js.storage_near_limit",
		Doc: "JetStream file storage is over 80% (warning) or 95% (critical) of a server or account limit.",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			check := func(resource, what string, used uint64, limit int64) {
				if limit <= 0 {
					return
				}
				ratio := float64(used) / float64(limit)
				sev := Warning
				switch {
				case ratio >= 0.95:
					sev = Critical
				case ratio < 0.8:
					return
				}
				out = append(out, Finding{
					RuleID: "js.storage_near_limit", Severity: sev, Resource: resource,
					Title:    fmt.Sprintf("%s JetStream storage is %.0f%% full", what, ratio*100),
					Detail:   "At the limit new messages are rejected. Lower stream limits, purge old data or raise the storage limit.",
					Evidence: []Evidence{{"Used of limit", fmt.Sprintf("%d of %d bytes", used, limit), "JSZ"}},
					Fixes:    []Fix{{Label: "Streams by size", CLI: "nats stream report --sort size"}},
				})
			}
			for _, sv := range s.Servers {
				if sv.Responding && sv.JetStream {
					check("server:"+sv.Name, sv.Name, sv.Storage, sv.MaxStorage)
				}
			}
			if !s.System {
				check("account", "Account", s.AccountStorage, s.AccountMaxStorage)
			}
			return out
		},
	},
	{
		ID:  "js.stream_no_leader",
		Doc: "A replicated stream has no leader, so it accepts no writes.",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			for _, st := range s.Streams {
				if !st.Clustered || st.Leader != "" {
					continue
				}
				out = append(out, Finding{
					RuleID: "js.stream_no_leader", Severity: Critical, Resource: "stream:" + st.Name,
					Title:    "Stream " + st.Name + " has no leader",
					Detail:   "Publishes to it fail until its replicas elect a leader. Usually too many of its servers are down.",
					Evidence: []Evidence{{"Leader", "none", "JSZ"}},
					Fixes:    []Fix{{Label: "Stream report", CLI: "nats stream report"}},
				})
			}
			return out
		},
	},
	{
		ID:  "js.replica_offline",
		Doc: "A stream replica's server hasn't been seen by the stream leader recently.",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			for _, st := range s.Streams {
				if len(st.Offline) == 0 || st.Leader == "" {
					continue
				}
				out = append(out, Finding{
					RuleID: "js.replica_offline", Severity: Warning, Resource: "stream:" + st.Name,
					Title:    fmt.Sprintf("Stream %s has %d offline replica(s)", st.Name, len(st.Offline)),
					Detail:   "The stream still works but has less redundancy; losing more servers could stop it.",
					Evidence: []Evidence{{"Offline", strings.Join(st.Offline, ", "), "JSZ"}, {"Leader", st.Leader, "JSZ"}},
					Fixes:    []Fix{{Label: "Stream info", CLI: "nats stream info " + lastPart(st.Name)}},
				})
			}
			return out
		},
	},
	{
		ID:  "js.replica_lagging",
		Doc: "A stream replica has been behind its leader for more than a minute.",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			for _, st := range s.Streams {
				if len(st.Lagging) == 0 {
					continue
				}
				out = append(out, Finding{
					RuleID: "js.replica_lagging", Severity: Warning, Resource: "stream:" + st.Name,
					Title:    "Stream " + st.Name + " has a replica that keeps falling behind",
					Detail:   "A slow disk or network on that server, or more writes than it can keep up with.",
					Evidence: []Evidence{{"Lagging replicas", strings.Join(st.Lagging, ", "), "JSZ"}},
					Fixes:    []Fix{{Label: "Stream info", CLI: "nats stream info " + lastPart(st.Name)}},
				})
			}
			return out
		},
	},
	{
		ID:  "js.consumer_lag_growing",
		Doc: "A consumer's pending count has only gone up for the last 15 minutes.",
		Check: func(s Snapshot) []Finding {
			var out []Finding
			for _, c := range s.Consumers {
				if !c.Growing {
					continue
				}
				out = append(out, Finding{
					RuleID: "js.consumer_lag_growing", Severity: Warning, Resource: "consumer:" + c.Name,
					Title:    "Consumer " + c.Name + " is falling behind",
					Detail:   "Messages arrive faster than it processes them, or nothing is reading. Check its subscribers.",
					Evidence: []Evidence{{"Pending", fmt.Sprintf("%d, up from %d", c.Pending, c.Since), "JSZ"}},
					Fixes:    []Fix{{Label: "Consumer report", CLI: "nats consumer report " + streamOf(c.Name)}},
				})
			}
			return out
		},
	},
}

func lastPart(name string) string { return name[strings.LastIndex(name, "/")+1:] }

// streamOf returns the stream from "account/stream/consumer" or "stream/consumer".
func streamOf(name string) string {
	parts := strings.Split(name, "/")
	return parts[len(parts)-2]
}

// Run applies every rule the snapshot can support.
func Run(s Snapshot) []Finding {
	var out []Finding
	for _, r := range Rules {
		if r.SystemOnly && !s.System {
			continue
		}
		out = append(out, r.Check(s)...)
	}
	return out
}
