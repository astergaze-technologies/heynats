package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/astergaze-solutions/heynats/internal/infrastructure"
	"github.com/astergaze-solutions/heynats/internal/pkg"
	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go/jetstream"
)

// newKVRouter uses the production router so path settings match the real server.
func newKVRouter(t *testing.T) http.Handler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	nc := natstest.Connect(t, natstest.Start(t))
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}
	cred := &pkg.NATSCredential{Conn: nc, JetStream: &js}

	r := infrastructure.NewRouter(fstest.MapFS{})
	g := r.Group("/kv", func(c *gin.Context) { c.Set(NatsConnectionKey, cred) })
	kv := &KVAPI{}
	g.POST("/buckets", kv.CreateBucket)
	g.GET("/buckets/:bucket/keys", kv.GetBucketKeys)
	g.PUT("/buckets/:bucket/keys/:key", kv.PutKeyValue)
	g.GET("/buckets/:bucket/keys/:key", kv.GetKeyValue)
	g.DELETE("/buckets/:bucket/keys/:key", kv.DeleteKey)
	return r
}

func TestKVKeyWithSlash(t *testing.T) {
	r := newKVRouter(t)
	if w := doJSON(r, http.MethodPost, "/kv/buckets", gin.H{"bucket": "cfg"}); w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body)
	}

	path := "/kv/buckets/cfg/keys/app%2Fdb.url"
	if w := doJSON(r, http.MethodPut, path, gin.H{"value": "postgres://x"}); w.Code != http.StatusOK {
		t.Fatalf("put: %d %s", w.Code, w.Body)
	}
	w := doJSON(r, http.MethodGet, path, nil)
	var got map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["key"] != "app/db.url" || got["value"] != "postgres://x" {
		t.Errorf("get: %d %v", w.Code, got)
	}
	if w := doJSON(r, http.MethodDelete, path, nil); w.Code != http.StatusOK {
		t.Errorf("delete: %d %s", w.Code, w.Body)
	}
}

func TestKVListsAllKeys(t *testing.T) {
	r := newKVRouter(t)
	doJSON(r, http.MethodPost, "/kv/buckets", gin.H{"bucket": "many"})
	for i := range 25 {
		doJSON(r, http.MethodPut, fmt.Sprintf("/kv/buckets/many/keys/k%d", i), gin.H{"value": "v"})
	}

	var body struct {
		Items []any `json:"items"`
	}
	w := doJSON(r, http.MethodGet, "/kv/buckets/many/keys", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Items) != 25 {
		t.Errorf("got %d items, want 25", len(body.Items))
	}

	w = doJSON(r, http.MethodGet, "/kv/buckets/many/keys?pageSize=10&page=2", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Items) != 5 {
		t.Errorf("last page: got %d items, want 5", len(body.Items))
	}
}

func TestKVPutNonStringValue(t *testing.T) {
	r := newKVRouter(t)
	doJSON(r, http.MethodPost, "/kv/buckets", gin.H{"bucket": "obj"})

	cases := map[string]any{"o": gin.H{"a": 1}, "n": 42}
	want := map[string]string{"o": `{"a":1}`, "n": "42"}
	for key, val := range cases {
		path := "/kv/buckets/obj/keys/" + key
		if w := doJSON(r, http.MethodPut, path, gin.H{"value": val}); w.Code != http.StatusOK {
			t.Fatalf("put %s: %d %s", key, w.Code, w.Body)
		}
		var got map[string]string
		_ = json.Unmarshal(doJSON(r, http.MethodGet, path, nil).Body.Bytes(), &got)
		if got["value"] != want[key] {
			t.Errorf("%s: got %q, want %q", key, got["value"], want[key])
		}
	}
}
