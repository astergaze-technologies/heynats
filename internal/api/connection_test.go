package api

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func requestFor(addr string) *pkg.ConnectionRequest {
	host, port, _ := net.SplitHostPort(addr)
	return &pkg.ConnectionRequest{Host: host, Port: port}
}

func connRequest(srv *server.Server) *pkg.ConnectionRequest {
	return requestFor(srv.Addr().String())
}

func dialAddr(t *testing.T, addr string) *pkg.NATSCredential {
	t.Helper()
	conn := pkg.NewNATSCredential(requestFor(addr))
	if err := conn.Connect(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(conn.Disconnect)
	return conn
}

func dial(t *testing.T, srv *server.Server) *pkg.NATSCredential {
	t.Helper()
	return dialAddr(t, srv.Addr().String())
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

// freezableProxy forwards TCP to target until freeze is called, then stops
// passing bytes while keeping connections open, like `docker pause`.
func freezableProxy(t *testing.T, target string) (addr string, freeze func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var gate sync.Mutex
	frozen := false
	t.Cleanup(func() {
		ln.Close()
		if frozen {
			gate.Unlock()
		}
	})

	pipe := func(dst, src net.Conn) {
		defer dst.Close()
		buf := make([]byte, 32*1024)
		for {
			n, err := src.Read(buf)
			if err != nil {
				return
			}
			gate.Lock()
			gate.Unlock()
			if _, err := dst.Write(buf[:n]); err != nil {
				return
			}
		}
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			up, err := net.Dial("tcp", target)
			if err != nil {
				c.Close()
				continue
			}
			go pipe(up, c)
			go pipe(c, up)
		}
	}()
	return ln.Addr().String(), func() {
		gate.Lock()
		frozen = true
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

func TestGetOrReconnectNotBlockedByFrozenServer(t *testing.T) {
	healthy := natstest.Start(t)
	frozen := natstest.Start(t)
	proxy, freeze := freezableProxy(t, frozen.Addr().String())
	store := newStore(t)
	store.AddConnection("a", dial(t, healthy), connRequest(healthy))
	store.AddConnection("b", dialAddr(t, proxy), requestFor(proxy))

	freeze()
	done := make(chan struct{})
	go func() {
		defer close(done)
		store.GetOrReconnect("b")
	}()
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	if _, ok, err := store.GetOrReconnect("a"); !ok || err != nil {
		t.Fatalf("GetOrReconnect(a) = %v, %v", ok, err)
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Errorf("healthy session waited %v behind a frozen one", d)
	}
	<-done
}

func TestGetOrReconnectReturnsReconnectingConn(t *testing.T) {
	srv := natstest.Start(t)
	store := newStore(t)
	conn := dial(t, srv)
	store.AddConnection("id", conn, connRequest(srv))

	srv.Shutdown()
	waitReconnecting(t, conn)

	start := time.Now()
	got, ok, err := store.GetOrReconnect("id")
	if got != conn || !ok || err != nil {
		t.Fatalf("GetOrReconnect = %v, %v, %v; want the reconnecting conn", got, ok, err)
	}
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Errorf("GetOrReconnect took %v", d)
	}
}

func TestGetOrReconnectRedialsClosedConn(t *testing.T) {
	srv := natstest.Start(t)
	store := newStore(t)
	old := dial(t, srv)
	store.AddConnection("id", old, connRequest(srv))
	old.Conn.Close()

	var wg sync.WaitGroup
	got := make([]*pkg.NATSCredential, 10)
	for i := range got {
		wg.Go(func() { got[i], _, _ = store.GetOrReconnect("id") })
	}
	wg.Wait()

	for _, c := range got {
		if c == nil || c == old || c != got[0] || !c.IsConnected() {
			t.Fatalf("want one shared new connection, got %v", got)
		}
	}
	waitClients(t, srv, 1)
	t.Cleanup(got[0].Disconnect)
}

func TestGetOrReconnectRemovesSessionWhenRedialFails(t *testing.T) {
	srv := natstest.Start(t)
	store := newStore(t)
	old := dial(t, srv)
	store.AddConnection("id", old, connRequest(srv))
	old.Conn.Close()
	srv.Shutdown()

	if _, _, err := store.GetOrReconnect("id"); err == nil {
		t.Fatal("want redial error")
	}
	if _, ok := store.GetConnection("id"); ok {
		t.Fatal("session kept after failed redial")
	}
}

func waitClients(t *testing.T, srv *server.Server, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for srv.NumClients() != want {
		if time.Now().After(deadline) {
			t.Fatalf("clients = %d, want %d", srv.NumClients(), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
