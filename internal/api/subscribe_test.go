package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func newSubscribeServer(t *testing.T, nc *nats.Conn) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	inject := func(c *gin.Context) {
		c.Set(NatsConnectionKey, &pkg.NATSCredential{Conn: nc})
	}
	r.GET("/subscribe/messages/:subject", inject, (&SubscribeAPI{}).subscribeToSubject)
	r.GET("/streams/:stream/subjects/:subject/subscribe", inject, (&StreamAPI{}).SubscribeToStreamSubject)

	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)
	return ts.URL
}

func openSSE(t *testing.T, ctx context.Context, url string) <-chan map[string]any {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	events := make(chan map[string]any, 1024)
	go func() {
		defer close(events)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			data, ok := strings.CutPrefix(sc.Text(), "data: ")
			ev := map[string]any{}
			if ok && json.Unmarshal([]byte(data), &ev) == nil {
				events <- ev
			}
		}
	}()
	return events
}

func nextEvent(t *testing.T, events <-chan map[string]any) map[string]any {
	t.Helper()
	select {
	case ev, ok := <-events:
		if !ok {
			t.Fatal("stream closed")
		}
		return ev
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for event")
	}
	return nil
}

func waitForSubs(t *testing.T, srv *server.Server, want uint32) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for srv.NumSubscriptions() != want {
		if time.Now().After(deadline) {
			t.Fatalf("subscriptions = %d, want %d", srv.NumSubscriptions(), want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSubscribeMaxMessages(t *testing.T) {
	srv := natstest.Start(t)
	nc := natstest.Connect(t, srv)
	base := newSubscribeServer(t, natstest.Connect(t, srv))
	baseline := srv.NumSubscriptions()

	events := openSSE(t, t.Context(), base+"/subscribe/messages/test.max?max_messages=3")
	if ev := nextEvent(t, events); ev["type"] != "connected" {
		t.Fatalf("first event = %v, want connected", ev)
	}
	for i := range 10 {
		_ = nc.Publish("test.max", fmt.Appendf(nil, "m%d", i))
	}
	_ = nc.Flush()

	for i := range 3 {
		if ev := nextEvent(t, events); ev["data"] != fmt.Sprintf("m%d", i) {
			t.Fatalf("message %d = %v", i, ev)
		}
	}
	if ev := nextEvent(t, events); ev["type"] != "completed" || ev["count"] != float64(3) {
		t.Fatalf("got %v, want completed with count 3", ev)
	}
	if _, ok := <-events; ok {
		t.Fatal("stream still open after completed")
	}
	waitForSubs(t, srv, baseline)

	for range 100 {
		_ = nc.Publish("test.max", []byte("late"))
	}
	_ = nc.Flush()
	if ev := nextEvent(t, openSSE(t, t.Context(), base+"/subscribe/messages/test.max")); ev["type"] != "connected" {
		t.Fatalf("server not serving after completed: %v", ev)
	}
}

func TestSubscribeClientDisconnect(t *testing.T) {
	for name, path := range map[string]string{
		"subscribe":   "/subscribe/messages/test.burst",
		"stream tail": "/streams/S/subjects/test.burst/subscribe",
	} {
		t.Run(name, func(t *testing.T) {
			srv := natstest.Start(t)
			nc := natstest.Connect(t, srv)
			base := newSubscribeServer(t, natstest.Connect(t, srv))
			baseline := srv.NumSubscriptions()

			stop := make(chan struct{})
			done := make(chan struct{})
			go func() {
				defer close(done)
				for {
					select {
					case <-stop:
						return
					default:
						_ = nc.Publish("test.burst", []byte("x"))
					}
				}
			}()

			for range 50 {
				ctx, cancel := context.WithCancel(t.Context())
				events := openSSE(t, ctx, base+path)
				for range 6 {
					nextEvent(t, events)
				}
				cancel()
			}
			close(stop)
			<-done

			waitForSubs(t, srv, baseline)
		})
	}
}
