package natstest

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

// Cluster is a set of in-process JetStream servers joined by routes.
type Cluster struct {
	t       testing.TB
	opts    []*server.Options
	Servers []*server.Server
}

// StartCluster starts n clustered JetStream servers. Ports are fixed per
// node so Restart brings a node back on the same URL and store.
func StartCluster(t testing.TB, n int) *Cluster {
	t.Helper()
	routes := make([]string, n)
	c := &Cluster{t: t, opts: make([]*server.Options, n), Servers: make([]*server.Server, n)}
	for i := range n {
		routePort := freePort(t)
		routes[i] = fmt.Sprintf("nats://127.0.0.1:%d", routePort)
		c.opts[i] = &server.Options{
			ServerName: fmt.Sprintf("n%d", i+1),
			Host:       "127.0.0.1",
			Port:       freePort(t),
			JetStream:  true,
			StoreDir:   t.TempDir(),
			NoLog:      true,
			NoSigs:     true,
			Cluster:    server.ClusterOpts{Name: "test", Host: "127.0.0.1", Port: routePort},
		}
	}
	for i := range n {
		c.opts[i].Routes = server.RoutesFromStr(strings.Join(routes, ","))
		c.Restart(i)
	}
	t.Cleanup(func() {
		for i := range c.Servers {
			c.Stop(i)
		}
	})
	return c
}

// URLs returns the client URLs of all nodes, comma separated for nats.Connect.
func (c *Cluster) URLs() string {
	urls := make([]string, len(c.opts))
	for i, o := range c.opts {
		urls[i] = fmt.Sprintf("nats://%s:%d", o.Host, o.Port)
	}
	return strings.Join(urls, ",")
}

func (c *Cluster) Stop(i int) {
	if s := c.Servers[i]; s != nil {
		s.Shutdown()
		s.WaitForShutdown()
		c.Servers[i] = nil
	}
}

// Restart starts node i (again) with its original options and store.
func (c *Cluster) Restart(i int) {
	c.t.Helper()
	c.Stop(i)
	srv, err := server.NewServer(c.opts[i].Clone())
	if err != nil {
		c.t.Fatal(err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		c.t.Fatalf("node %d not ready", i)
	}
	c.Servers[i] = srv
}

// shortcut: port can be taken between close and reuse; fine for tests.
func freePort(t testing.TB) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
