package monitor

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// Subscriber is one subscription, with the connection that holds it.
type Subscriber struct {
	Server  string `json:"server"`
	CID     uint64 `json:"cid"`
	Name    string `json:"name,omitempty"`
	Account string `json:"account,omitempty"`
	Queue   string `json:"queue,omitempty"`
	Msgs    int64  `json:"msgs"`
}

// StoredStream is a stream with the subject patterns it captures; Counts are
// messages per stored subject, own account only and capped per stream.
type StoredStream struct {
	Account   string            `json:"account,omitempty"`
	Name      string            `json:"name"`
	Subjects  []string          `json:"subjects"`
	Counts    map[string]uint64 `json:"counts,omitempty"`
	Truncated bool              `json:"truncated,omitempty"`
}

type SubjectMap struct {
	Source  Source `json:"source"`
	Partial bool   `json:"partial"`
	// Listening maps a subscribed subject (wildcards kept as written) to its subscribers.
	Listening map[string][]Subscriber `json:"listening"`
	Streams   []StoredStream          `json:"streams"`
	// HiddenInboxes counts reply-inbox subscriptions left out of Listening.
	HiddenInboxes int `json:"hidden_inboxes"`
}

const maxStoredSubjects = 10_000

// shortcut: each server reports its first 5000 connections with their
// subscriptions; page through CONNZ when clusters outgrow that.
const subjectConnLimit = 5000

func (m *Monitor) Subjects() (*SubjectMap, error) {
	res, err := m.PingWith("CONNZ", map[string]any{"subscriptions_detail": true, "auth": true, "limit": subjectConnLimit})
	if err != nil {
		return nil, err
	}
	out := &SubjectMap{Source: res.Source, Partial: res.Partial, Listening: map[string][]Subscriber{}, Streams: []StoredStream{}}
	for _, r := range res.Replies {
		var d struct {
			Conns []struct {
				CID     uint64 `json:"cid"`
				Name    string `json:"name"`
				Account string `json:"account"`
				Subs    []struct {
					Subject string `json:"subject"`
					Queue   string `json:"qgroup"`
					Msgs    int64  `json:"msgs"`
				} `json:"subscriptions_list_detail"`
			} `json:"connections"`
		}
		if r.Error != nil || json.Unmarshal(r.Data, &d) != nil {
			continue
		}
		for _, c := range d.Conns {
			for _, s := range c.Subs {
				if strings.HasPrefix(s.Subject, "_INBOX.") {
					out.HiddenInboxes++
					continue
				}
				out.Listening[s.Subject] = append(out.Listening[s.Subject], Subscriber{
					Server: r.Server.Name, CID: c.CID, Name: c.Name, Account: c.Account, Queue: s.Queue, Msgs: s.Msgs,
				})
			}
		}
	}

	if out.Source == System {
		// Stream patterns only: per-subject counts need the owning account's JetStream API.
		if h, err := m.JetStreamHealth(); err == nil {
			for _, s := range h.Streams {
				out.Streams = append(out.Streams, StoredStream{Account: s.Account, Name: s.Name, Subjects: s.Subjects})
			}
		}
		return out, nil
	}
	streams, err := m.storedSubjects()
	if err == nil {
		out.Streams = streams
	}
	return out, nil
}

// storedSubjects reads the user's streams with how many messages each
// stored subject holds.
func (m *Monitor) storedSubjects() ([]StoredStream, error) {
	js, err := jetstream.New(m.nc)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var out []StoredStream
	names := js.StreamNames(ctx)
	for name := range names.Name() {
		st, err := js.Stream(ctx, name)
		if err != nil {
			return nil, err
		}
		info, err := st.Info(ctx, jetstream.WithSubjectFilter(">"))
		if err != nil {
			return nil, err
		}
		s := StoredStream{Name: name, Subjects: info.Config.Subjects, Counts: map[string]uint64{}}
		keys := make([]string, 0, len(info.State.Subjects))
		for k := range info.State.Subjects {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) > maxStoredSubjects {
			keys, s.Truncated = keys[:maxStoredSubjects], true
		}
		for _, k := range keys {
			s.Counts[k] = info.State.Subjects[k]
		}
		out = append(out, s)
	}
	return out, names.Err()
}
