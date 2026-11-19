package pkg

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/astergaze-solutions/heynats/internal/testutil/natstest"
	"github.com/nats-io/nats.go"
)

func newStreamCred(t *testing.T) *NATSCredential {
	t.Helper()
	nc := natstest.Connect(t, natstest.Start(t))
	js, err := nc.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	return &NATSCredential{Conn: nc, JSConn: &js}
}

func TestSearchPagingCountsMatches(t *testing.T) {
	cred := newStreamCred(t)
	js := *cred.JSConn
	if _, err := js.AddStream(&nats.StreamConfig{Name: "S", Subjects: []string{"s"}}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 40; i++ {
		data := "other"
		if i%2 == 1 {
			data = "match"
		}
		if _, err := js.Publish("s", []byte(data)); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := cred.GetStreamMessagesWithSearch("S", 5, 3, "match")
	if err != nil {
		t.Fatal(err)
	}
	var got []uint64
	for _, m := range resp.Messages {
		got = append(got, m.Sequence)
	}
	if len(got) != 3 || got[0] != 11 || got[1] != 13 || got[2] != 15 {
		t.Errorf("got sequences %v, want [11 13 15]", got)
	}
}

func TestCreateStreamAppliesAllFields(t *testing.T) {
	cred := newStreamCred(t)
	info, err := cred.CreateStream(&StreamConfig{
		Name:              "FULL",
		Subjects:          []string{"full.>"},
		AllowMsgTTL:       true,
		MaxMsgsPerSubject: 7,
		MaxMsgSize:        1024,
		DuplicateWindow:   int64(30 * time.Second),
		Compression:       "s2",
	})
	if err != nil {
		t.Fatal(err)
	}

	cfg := info.Config
	if !cfg.AllowMsgTTL || cfg.MaxMsgsPerSubject != 7 || cfg.MaxMsgSize != 1024 ||
		cfg.Duplicates != 30*time.Second || cfg.Compression != nats.S2Compression {
		t.Errorf("config not applied: %+v", cfg)
	}
}

func TestCreateStreamRejectsUnknownCompression(t *testing.T) {
	cred := newStreamCred(t)
	if _, err := cred.CreateStream(&StreamConfig{Name: "X", Subjects: []string{"x"}, Compression: "zip"}); err == nil {
		t.Error("expected error for unknown compression")
	}
}

func TestListConsumersWithoutJetStream(t *testing.T) {
	cred := newStreamCred(t)
	cred.JSConn = nil
	if _, err := cred.ListConsumers("S"); err == nil {
		t.Error("expected error without JetStream")
	}
}

func TestSequencesBeyondFloat64Precision(t *testing.T) {
	cred := newStreamCred(t)
	js := *cred.JSConn
	const first = uint64(1)<<60 + 1
	if _, err := js.AddStream(&nats.StreamConfig{Name: "BIG", Subjects: []string{"big"}, FirstSeq: first}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := js.Publish("big", []byte(`{"id":323439282914070533}`)); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := cred.GetStreamMessages("BIG", 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(resp)
	for i := uint64(0); i < 3; i++ {
		if want := `"sequence":"` + strconv.FormatUint(first+i, 10) + `"`; !strings.Contains(string(b), want) {
			t.Errorf("missing %s in %s", want, b)
		}
	}
	if !strings.Contains(string(b), `323439282914070533`) {
		t.Errorf("payload must pass through untouched: %s", b)
	}

	entry, _ := json.Marshal(KVEntry{Revision: first})
	if !strings.Contains(string(entry), `"revision":"1152921504606846977"`) {
		t.Errorf("kv revision = %s", entry)
	}
}
