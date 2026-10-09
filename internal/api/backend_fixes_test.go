package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func newTestRouter(t *testing.T) (*gin.Engine, *nats.Conn) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	nc := natstest.Connect(t, natstest.Start(t))
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}
	jsc, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	cred := &pkg.NATSCredential{Conn: nc, JSConn: &jsc, JetStream: &js}

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(NatsConnectionKey, cred) })
	r.POST("/kv/buckets", (&KVAPI{}).CreateBucket)
	r.GET("/streams/consumers/:stream", (&StreamAPI{}).ListConsumers)
	r.POST("/publish/message", (&PublishAPI{}).publishMessage)
	r.POST("/publish/request", (&PublishAPI{}).requestReply)
	return r, nc
}

func doJSON(r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateBucketStatuses(t *testing.T) {
	r, _ := newTestRouter(t)

	if w := doJSON(r, http.MethodPost, "/kv/buckets", gin.H{"bucket": "b", "history": 300}); w.Code != http.StatusBadRequest {
		t.Errorf("history 300: got %d, want 400", w.Code)
	}
	if w := doJSON(r, http.MethodPost, "/kv/buckets", gin.H{"bucket": "b", "history": 5}); w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("create: got %d %s", w.Code, w.Body)
	}
	if w := doJSON(r, http.MethodPost, "/kv/buckets", gin.H{"bucket": "b", "history": 10}); w.Code != http.StatusConflict {
		t.Errorf("duplicate: got %d, want 409", w.Code)
	}
}

func TestListConsumersKey(t *testing.T) {
	r, nc := newTestRouter(t)
	if _, err := nc.Request("$JS.API.STREAM.CREATE.S", []byte(`{"name":"S","subjects":["s.>"]}`), time.Second); err != nil {
		t.Fatal(err)
	}

	w := doJSON(r, http.MethodGet, "/streams/consumers/S", nil)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if _, ok := body["consumers"]; !ok || w.Code != http.StatusOK {
		t.Errorf("got %d %s", w.Code, w.Body)
	}
}

func TestPublishEmptyPayload(t *testing.T) {
	r, nc := newTestRouter(t)
	sub, _ := nc.SubscribeSync("empty")

	if w := doJSON(r, http.MethodPost, "/publish/message", gin.H{"subject": "empty"}); w.Code != http.StatusOK {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if msg, err := sub.NextMsg(time.Second); err != nil || len(msg.Data) != 0 {
		t.Errorf("got %v, %v", msg, err)
	}
}

func TestRequestErrorStatuses(t *testing.T) {
	r, nc := newTestRouter(t)
	_, _ = nc.Subscribe("slow", func(*nats.Msg) {})

	if w := doJSON(r, http.MethodPost, "/publish/request", gin.H{"subject": "nobody"}); w.Code != http.StatusServiceUnavailable {
		t.Errorf("no responders: got %d, want 503", w.Code)
	}
	if w := doJSON(r, http.MethodPost, "/publish/request", gin.H{"subject": "slow", "timeout": 1}); w.Code != http.StatusGatewayTimeout {
		t.Errorf("timeout: got %d, want 504", w.Code)
	}
	if got := requestErrorStatus(nats.ErrConnectionClosed); got != http.StatusBadGateway {
		t.Errorf("other: got %d, want 502", got)
	}
}

func TestRequestCustomReplySubject(t *testing.T) {
	r, nc := newTestRouter(t)
	seen := make(chan string, 1)
	_, _ = nc.Subscribe("echo", func(m *nats.Msg) {
		seen <- m.Reply
		_ = m.Respond([]byte("pong"))
	})

	w := doJSON(r, http.MethodPost, "/publish/request", gin.H{"subject": "echo", "data": "ping", "reply_subject": "my.reply"})
	if w.Code != http.StatusOK {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if got := <-seen; got != "my.reply" {
		t.Errorf("responder saw reply %q", got)
	}
	var resp RequestReplyResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.ReplyData != "pong" {
		t.Errorf("reply data %q", resp.ReplyData)
	}
}
