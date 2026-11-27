package monitor

import (
	"fmt"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats.go"
)

func TestMergeConnsAcrossServers(t *testing.T) {
	c := func(server string, cid uint64, pending int, name string) Conn {
		return Conn{Server: server, CID: cid, Pending: pending, Name: name}
	}
	// Each server's rows arrive already sorted by pending, as CONNZ returns them.
	per := [][]Conn{
		{c("n1", 1, 900, "api"), c("n1", 2, 10, "worker")},
		{c("n2", 1, 500, "api"), c("n2", 7, 10, "billing")},
		{c("n3", 4, 700, "worker")},
	}
	page, matched := MergeConns(per, ConnQuery{Sort: "pending", Offset: 1, Limit: 3})
	if matched != 5 || len(page) != 3 {
		t.Fatalf("matched=%d page=%d", matched, len(page))
	}
	want := []string{"n3/4", "n2/1", "n1/2"} // 700, 500, then the 10s tied, broken by server
	for i, w := range want {
		if got := fmt.Sprintf("%s/%d", page[i].Server, page[i].CID); got != w {
			t.Errorf("row %d = %s, want %s", i, got, w)
		}
	}

	page, matched = MergeConns(per, ConnQuery{Sort: "pending", Limit: 10, Name: "WORK"})
	if matched != 2 || page[0].Server != "n3" {
		t.Fatalf("name filter: %+v", page)
	}
	if page, _ = MergeConns(per, ConnQuery{Sort: "pending", Offset: 9, Limit: 10}); len(page) != 0 {
		t.Fatalf("offset past the end: %+v", page)
	}
}

func TestConnectionsOpenThenClosedWithReason(t *testing.T) {
	c := natstest.StartCluster(t, 3, "../../nats.conf")
	m := monitorFor(t, c, "sys")
	var clients []*nats.Conn
	for i, s := range c.Servers {
		nc, err := nats.Connect(s.ClientURL(), nats.UserInfo("admin", devPassword), nats.Name("app-"+string(rune('a'+i))))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(nc.Close)
		clients = append(clients, nc)
	}

	page, err := m.Connections(ConnQuery{State: Open, Sort: "cid", Limit: 100, Name: "app-"})
	if err != nil {
		t.Fatal(err)
	}
	servers := map[string]bool{}
	for _, cn := range page.Conns {
		servers[cn.Server] = true
		if cn.Account != "APP" || cn.User != "admin" {
			t.Errorf("conn %+v", cn)
		}
	}
	if page.Matched != 3 || len(servers) != 3 {
		t.Fatalf("want app-a..c on 3 servers, got %+v", page.Conns)
	}

	clients[1].Close()
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		page, err = m.Connections(ConnQuery{State: Closed, Sort: "stop", Limit: 100, Name: "app-b"})
		if err == nil && len(page.Conns) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("closed client not listed: %+v, %v", page, err)
		}
	}
	if got := page.Conns[0]; got.Reason != "Client Closed" || got.Server != "n2" {
		t.Fatalf("closed conn %+v", got)
	}

	detail, err := m.Connection("n2", page.Conns[0].CID)
	if err != nil || detail.Name != "app-b" {
		t.Fatalf("detail %+v, %v", detail, err)
	}
	if _, err := m.Connections(ConnQuery{Sort: "nope", Limit: 1}); err == nil {
		t.Fatal("unknown sort accepted")
	}
}
