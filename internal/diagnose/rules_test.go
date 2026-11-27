package diagnose

import "testing"

func ok(name string) Server {
	return Server{Name: name, Cluster: "c", Version: "2.12.0", Responding: true, Health: "ok",
		Connections: 10, MaxConnections: 100, JetStream: true, Storage: 10, MaxStorage: 100,
		MetaLeader: "n1", MetaClusterSize: 3}
}

func healthy() Snapshot {
	return Snapshot{System: true, Servers: []Server{ok("n1"), ok("n2"), ok("n3")}}
}

func TestRules(t *testing.T) {
	tests := []struct {
		rule     string
		change   func(*Snapshot)
		severity Severity // "" = must not fire
		resource string
	}{
		{"server.unreachable", func(s *Snapshot) { s.Servers[2].Missed = 1 }, "", ""},
		{"server.unreachable", func(s *Snapshot) { s.Servers[2].Missed, s.Servers[2].Responding = 2, false }, Warning, "server:n3"},
		{"server.unreachable", func(s *Snapshot) {
			for i := 1; i < 3; i++ {
				s.Servers[i].Missed, s.Servers[i].Responding = 3, false
			}
		}, Critical, "server:n2"},
		{"server.healthz", func(s *Snapshot) { s.Servers[1].Health = "JetStream not current" }, Warning, "server:n2"},
		{"server.healthz", func(s *Snapshot) { s.Servers[1].Responding, s.Servers[1].Health = false, "" }, "", ""},
		{"server.version_mismatch", func(s *Snapshot) { s.Servers[0].Version = "2.11.4" }, Info, "cluster:c"},
		{"server.slow_consumers", func(s *Snapshot) { s.Servers[0].SlowDelta = 4 }, Warning, "server:n1"},
		{"server.connections_near_limit", func(s *Snapshot) { s.Servers[0].Connections = 90 }, "", ""},
		{"server.connections_near_limit", func(s *Snapshot) { s.Servers[0].Connections = 91 }, Warning, "server:n1"},
		{"js.meta_no_leader", func(s *Snapshot) { s.Servers[2].MetaLeader = "" }, Critical, "jetstream:meta"},
		{"js.meta_no_leader", func(s *Snapshot) {
			s.Servers = s.Servers[:1]
			s.Servers[0].MetaLeader, s.Servers[0].MetaClusterSize = "", 1 // standalone
		}, "", ""},
		{"js.storage_near_limit", func(s *Snapshot) { s.Servers[0].Storage = 79 }, "", ""},
		{"js.storage_near_limit", func(s *Snapshot) { s.Servers[0].Storage = 80 }, Warning, "server:n1"},
		{"js.storage_near_limit", func(s *Snapshot) { s.Servers[0].Storage = 96 }, Critical, "server:n1"},
		{"js.storage_near_limit", func(s *Snapshot) {
			*s = Snapshot{AccountStorage: 85, AccountMaxStorage: 100}
		}, Warning, "account"},
		{"js.storage_near_limit", func(s *Snapshot) {
			*s = Snapshot{AccountStorage: 1 << 40, AccountMaxStorage: -1} // no limit
		}, "", ""},
		{"js.stream_no_leader", func(s *Snapshot) { s.Streams = []Stream{{Name: "APP/ORDERS", Clustered: true}} }, Critical, "stream:APP/ORDERS"},
		{"js.stream_no_leader", func(s *Snapshot) { s.Streams = []Stream{{Name: "APP/LOCAL"}} }, "", ""}, // R1, unclustered
		{"js.replica_offline", func(s *Snapshot) {
			s.Streams = []Stream{{Name: "APP/ORDERS", Clustered: true, Leader: "n1", Offline: []string{"n3"}}}
		}, Warning, "stream:APP/ORDERS"},
		{"js.replica_lagging", func(s *Snapshot) {
			s.Streams = []Stream{{Name: "APP/ORDERS", Clustered: true, Leader: "n1", Lagging: []string{"n2 (500 behind)"}}}
		}, Warning, "stream:APP/ORDERS"},
		{"js.consumer_lag_growing", func(s *Snapshot) {
			s.Consumers = []Consumer{{Name: "APP/ORDERS/billing", Pending: 900, Since: 100, Growing: true}}
		}, Warning, "consumer:APP/ORDERS/billing"},
		{"js.consumer_lag_growing", func(s *Snapshot) {
			s.Consumers = []Consumer{{Name: "APP/ORDERS/billing", Pending: 900}}
		}, "", ""},
		{"consumer.paused", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", Paused: true}} }, Warning, "consumer:c"},
		{"consumer.ack_pending_full", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", AckPending: 5, MaxAckPending: 5}} }, Warning, "consumer:c"},
		{"consumer.ack_pending_full", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", AckPending: 4, MaxAckPending: 5}} }, "", ""},
		{"consumer.high_redelivery", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", Deliveries: 100, Redelivery: 0.25}} }, Warning, "consumer:c"},
		{"consumer.high_redelivery", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", Deliveries: 10, Redelivery: 0.9}} }, "", ""}, // too few to judge
		{"consumer.no_interest", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", DeliverSubject: "d.x"}} }, Warning, "consumer:c"},
		{"consumer.no_interest", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", DeliverSubject: "d.x", PushBound: true}} }, "", ""},
		{"consumer.nobody_pulling", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", Pending: 3, IdleTooLong: true}} }, Warning, "consumer:c"},
		{"consumer.nobody_pulling", func(s *Snapshot) { s.Consumers = []Consumer{{Name: "c", Pending: 3, IdleTooLong: true, Waiting: 1}} }, "", ""},
		{"consumer.filter_matches_nothing", func(s *Snapshot) {
			s.Consumers = []Consumer{{Name: "c", Filters: []string{"legacy.>"}, StreamSubjects: []string{"orders.>"}}}
		}, Warning, "consumer:c"},
		{"consumer.filter_matches_nothing", func(s *Snapshot) {
			s.Consumers = []Consumer{{Name: "c", Filters: []string{"orders.eu"}, StreamSubjects: []string{"orders.>"}}}
		}, "", ""},
	}
	for _, tt := range tests {
		snap := healthy()
		tt.change(&snap)
		var got []Finding
		for _, f := range Run(snap) {
			if f.RuleID == tt.rule {
				got = append(got, f)
			}
		}
		switch {
		case tt.severity == "" && len(got) > 0:
			t.Errorf("%s fired unexpectedly: %+v", tt.rule, got)
		case tt.severity != "" && (len(got) == 0 || got[0].Severity != tt.severity || !has(got, tt.resource)):
			t.Errorf("%s: got %+v, want %s on %s", tt.rule, got, tt.severity, tt.resource)
		}
	}
	if fs := Run(healthy()); len(fs) != 0 {
		t.Errorf("healthy cluster has findings: %+v", fs)
	}
}

func has(fs []Finding, resource string) bool {
	for _, f := range fs {
		if f.Resource == resource {
			return true
		}
	}
	return false
}

func TestAccountScopeSkipsSystemRules(t *testing.T) {
	snap := healthy()
	snap.System = false
	snap.Servers[0].Health = "broken" // invisible without the system account
	if fs := Run(snap); len(fs) != 0 {
		t.Errorf("got %+v", fs)
	}
}
