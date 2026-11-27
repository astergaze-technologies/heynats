package monitor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// JetStream is usage against limits plus asset counts. Limits are -1 when
// unlimited. For the system source, usage counts every replica (real disk),
// while streams and consumers count each one once.
type JetStream struct {
	Source     Source `json:"source"`
	Partial    bool   `json:"partial"`
	Servers    int    `json:"servers"`
	Expected   int    `json:"expected"`
	Memory     uint64 `json:"memory"`
	Storage    uint64 `json:"storage"`
	MaxMemory  int64  `json:"max_memory"`
	MaxStorage int64  `json:"max_storage"`
	Streams    int    `json:"streams"`
	Consumers  int    `json:"consumers"`
}

func (m *Monitor) JetStream() (*JetStream, error) {
	if m.Source() == Account {
		return m.accountJetStream()
	}
	res, err := m.PingWith("JSZ", map[string]bool{"accounts": true, "streams": true, "consumer": true})
	if err != nil {
		return nil, err
	}
	out := &JetStream{Source: System, Partial: res.Partial, Servers: len(res.Replies), Expected: res.Expected}
	streams, consumers := map[string]bool{}, map[string]bool{}
	for _, r := range res.Replies {
		if r.Error != nil {
			continue
		}
		var d struct {
			Memory  uint64 `json:"memory"`
			Storage uint64 `json:"storage"`
			Config  struct {
				MaxMemory  int64 `json:"max_memory"`
				MaxStorage int64 `json:"max_storage"`
			} `json:"config"`
			Accounts []struct {
				Name    string `json:"name"`
				Streams []struct {
					Name      string `json:"name"`
					Consumers []struct {
						Name string `json:"name"`
					} `json:"consumer_detail"`
				} `json:"stream_detail"`
			} `json:"account_details"`
		}
		if err := json.Unmarshal(r.Data, &d); err != nil {
			return nil, err
		}
		out.Memory += d.Memory
		out.Storage += d.Storage
		out.MaxMemory += d.Config.MaxMemory
		out.MaxStorage += d.Config.MaxStorage
		for _, a := range d.Accounts {
			for _, s := range a.Streams {
				streams[a.Name+"/"+s.Name] = true
				for _, c := range s.Consumers {
					consumers[a.Name+"/"+s.Name+"/"+c.Name] = true
				}
			}
		}
	}
	out.Streams, out.Consumers = len(streams), len(consumers)
	return out, nil
}

func (m *Monitor) accountJetStream() (*JetStream, error) {
	js, err := jetstream.New(m.nc)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	info, err := js.AccountInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &JetStream{
		Source:     Account,
		Memory:     info.Memory,
		Storage:    info.Store,
		MaxMemory:  info.Limits.MaxMemory,
		MaxStorage: info.Limits.MaxStore,
		Streams:    info.Streams,
		Consumers:  info.Consumers,
	}, nil
}
