package main

import (
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

const devPassword = "Lm2KZ22PCKB9GjwUYzxnXAm9IyDM8svZZUSp7tSOYcQ="

func startDevServer(t *testing.T) *server.Server {
	t.Helper()
	opts, err := server.ProcessConfigFile("nats.conf")
	if err != nil {
		t.Fatal(err)
	}
	opts.Host, opts.Port, opts.HTTPPort = "127.0.0.1", -1, 0
	opts.StoreDir, opts.NoLog, opts.NoSigs = t.TempDir(), true, true
	srv, err := server.NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("nats server not ready")
	}
	t.Cleanup(srv.Shutdown)
	return srv
}

func connectAs(t *testing.T, srv *server.Server, user string) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(srv.ClientURL(), nats.UserInfo(user, devPassword), natstest.ConnectTimeout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func TestDevNatsConf(t *testing.T) {
	srv := startDevServer(t)

	js, err := connectAs(t, srv, "admin").JetStream()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.AccountInfo(); err != nil {
		t.Errorf("admin has no JetStream: %v", err)
	}

	if _, err := connectAs(t, srv, "sys").Request("$SYS.REQ.SERVER.PING", nil, 2*time.Second); err != nil {
		t.Errorf("sys can't reach system subjects: %v", err)
	}
}
