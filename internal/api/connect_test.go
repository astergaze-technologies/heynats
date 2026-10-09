package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/gin-gonic/gin"
)

type connectClient struct {
	t     *testing.T
	base  string
	http  *http.Client
	store *NatsConnectionStore
}

func newConnectClient(t *testing.T) *connectClient {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := newStore(t)
	NewHeyNats(r.Group("/api/nats"), store, NewConnectionMiddleware(store)).RegisterRoutes()
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	jar, _ := cookiejar.New(nil)
	return &connectClient{t: t, base: ts.URL, http: &http.Client{Jar: jar}, store: store}
}

func (c *connectClient) connect(req *pkg.ConnectionRequest) int {
	c.t.Helper()
	body, _ := json.Marshal(req)
	resp, err := c.http.Post(c.base+"/api/nats/connect", "application/json", bytes.NewReader(body))
	if err != nil {
		c.t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

func (c *connectClient) connectedPort() string {
	c.t.Helper()
	resp, err := c.http.Get(c.base + "/api/nats/status")
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var status struct{ Port string }
	_ = json.NewDecoder(resp.Body).Decode(&status)
	return status.Port
}

func (c *connectClient) sessions() []*pkg.NATSCredential {
	c.store.mutex.RLock()
	defer c.store.mutex.RUnlock()
	var conns []*pkg.NATSCredential
	for _, info := range c.store.nastsConns {
		conns = append(conns, info.Connection)
	}
	return conns
}

func TestConnectWithNewCredentialsSwitchesServer(t *testing.T) {
	a := natstest.Start(t)
	b := natstest.Start(t)
	c := newConnectClient(t)

	if code := c.connect(connRequest(a)); code != http.StatusOK {
		t.Fatalf("connect to A: %d", code)
	}
	old := c.sessions()[0]

	if code := c.connect(connRequest(b)); code != http.StatusOK {
		t.Fatalf("connect to B: %d", code)
	}
	if got, want := c.connectedPort(), connRequest(b).Port; got != want {
		t.Fatalf("connected to port %s, want %s", got, want)
	}
	if n := len(c.sessions()); n != 1 {
		t.Fatalf("%d sessions, want 1", n)
	}
	if !old.Conn.IsClosed() {
		t.Fatal("old connection not closed")
	}
}

func TestConnectWithSameCredentialsKeepsConnection(t *testing.T) {
	srv := natstest.Start(t)
	c := newConnectClient(t)

	c.connect(connRequest(srv))
	first := c.sessions()[0]
	if code := c.connect(connRequest(srv)); code != http.StatusOK {
		t.Fatalf("reconnect: %d", code)
	}

	if s := c.sessions(); len(s) != 1 || s[0] != first {
		t.Fatal("same credentials should keep the existing connection")
	}
}

func TestConnectFailureKeepsCurrentSession(t *testing.T) {
	srv := natstest.Start(t)
	down := natstest.Start(t)
	downReq := connRequest(down)
	down.Shutdown()
	c := newConnectClient(t)

	c.connect(connRequest(srv))
	if code := c.connect(downReq); code == http.StatusOK {
		t.Fatal("connect to a stopped server succeeded")
	}

	if got, want := c.connectedPort(), connRequest(srv).Port; got != want {
		t.Fatalf("connected to port %s, want %s (current session lost)", got, want)
	}
}
