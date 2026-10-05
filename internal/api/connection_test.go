package api

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func connRequest(srv *server.Server) *pkg.ConnectionRequest {
	port := srv.Addr().(*net.TCPAddr).Port
	return &pkg.ConnectionRequest{Host: "127.0.0.1", Port: strconv.Itoa(port)}
}

func dial(t *testing.T, srv *server.Server) *pkg.NATSCredential {
	t.Helper()
	conn := pkg.NewNATSCredential(connRequest(srv))
	if err := conn.Connect(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(conn.Disconnect)
	return conn
}

func newStore(t *testing.T) *NatsConnectionStore {
	t.Helper()
	store := NewNatsConnection()
	t.Cleanup(store.Shutdown)
	return store
}

func waitReconnecting(t *testing.T, conn *pkg.NATSCredential) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for conn.Conn.Status() != nats.RECONNECTING {
		if time.Now().After(deadline) {
			t.Fatalf("status = %v, want RECONNECTING", conn.Conn.Status())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRemoveConnectionClosesReconnectingConn(t *testing.T) {
	srv := natstest.Start(t)
	store := newStore(t)
	conn := dial(t, srv)
	store.AddConnection("id", conn, connRequest(srv))

	srv.Shutdown()
	waitReconnecting(t, conn)
	store.RemoveConnection("id")

	if !conn.Conn.IsClosed() {
		t.Fatal("reconnecting connection not closed")
	}
}

func TestGetOrReconnectClosesReplacedConn(t *testing.T) {
	down := natstest.Start(t)
	up := natstest.Start(t)
	store := newStore(t)
	old := dial(t, down)
	store.AddConnection("id", old, connRequest(up))

	down.Shutdown()
	waitReconnecting(t, old)
	got, ok, err := store.GetOrReconnect("id")
	if err != nil || !ok || got == old {
		t.Fatalf("GetOrReconnect = %v, %v, %v; want new connection", got, ok, err)
	}

	if !old.Conn.IsClosed() {
		t.Fatal("replaced connection not closed")
	}
}

func TestGetOrReconnectClosesConnWhenRedialFails(t *testing.T) {
	srv := natstest.Start(t)
	store := newStore(t)
	old := dial(t, srv)
	store.AddConnection("id", old, connRequest(srv))

	srv.Shutdown()
	waitReconnecting(t, old)
	if _, _, err := store.GetOrReconnect("id"); err == nil {
		t.Fatal("want redial error")
	}

	if !old.Conn.IsClosed() {
		t.Fatal("dropped connection not closed")
	}
}
