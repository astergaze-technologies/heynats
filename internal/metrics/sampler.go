package metrics

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"time"

	"github.com/astergaze-solutions/heynats/internal/monitor"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// jsEvery: JetStream is sampled every third tick (15s at the 5s default).
const (
	jsEvery      = 3
	maxConsumers = 500
)

// counters are cumulative; rates come from the delta between two samples.
type counters struct {
	at                                 time.Time
	inMsgs, outMsgs, inBytes, outBytes int64
}

// Sampler fills Server (every tick) and JetStream (every jsEvery ticks).
type Sampler struct {
	nc        *nats.Conn
	mon       *monitor.Monitor
	Server    *History
	JetStream *History
	prev      map[string]counters
	stop      chan struct{}
	done      chan struct{}
}

// Start samples every interval until Stop. Pass the previous sampler's
// histories (or nil) to keep history across a reconnect.
func Start(nc *nats.Conn, mon *monitor.Monitor, interval time.Duration, keep *Sampler) *Sampler {
	s := &Sampler{
		nc: nc, mon: mon, prev: map[string]counters{},
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	if keep != nil {
		s.Server, s.JetStream = keep.Server, keep.JetStream
	} else {
		s.Server, s.JetStream = NewHistory(interval), NewHistory(jsEvery*interval)
	}
	go s.run(interval)
	return s
}

func (s *Sampler) Stop() {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	<-s.done
}

func (s *Sampler) run(interval time.Duration) {
	defer close(s.done)
	t := time.NewTicker(interval)
	defer t.Stop()
	for tick := 0; ; tick++ {
		now := time.Now()
		if err := s.sampleServers(now); err != nil {
			slog.Debug("metrics: server sample failed", "err", err)
		}
		if tick%jsEvery == 0 {
			if err := s.sampleJetStream(now); err != nil {
				slog.Debug("metrics: jetstream sample failed", "err", err)
			}
		}
		select {
		case <-s.stop:
			return
		case <-t.C:
		}
	}
}

func (s *Sampler) sampleServers(now time.Time) error {
	system := s.mon.Source() == monitor.System
	kind := "STATZ"
	if system {
		kind = "VARZ"
	}
	res, err := s.mon.Ping(kind)
	if err != nil {
		return err
	}
	row := map[string]float32{}
	for _, r := range res.Replies {
		if r.Error != nil {
			continue
		}
		name := r.Server.Name
		var c counters
		if system {
			var v struct {
				InMsgs        int64   `json:"in_msgs"`
				OutMsgs       int64   `json:"out_msgs"`
				InBytes       int64   `json:"in_bytes"`
				OutBytes      int64   `json:"out_bytes"`
				Connections   int     `json:"connections"`
				Subscriptions uint32  `json:"subscriptions"`
				SlowConsumers int64   `json:"slow_consumers"`
				CPU           float64 `json:"cpu"`
				Mem           int64   `json:"mem"`
			}
			if json.Unmarshal(r.Data, &v) != nil {
				continue
			}
			c = counters{now, v.InMsgs, v.OutMsgs, v.InBytes, v.OutBytes}
			row["connections|"+name] = float32(v.Connections)
			row["subscriptions|"+name] = float32(v.Subscriptions)
			row["slow_consumers|"+name] = float32(v.SlowConsumers)
			row["cpu|"+name] = float32(v.CPU)
			row["mem|"+name] = float32(v.Mem)
		} else {
			var v struct {
				Accounts []struct {
					Conns    int    `json:"conns"`
					NumSubs  uint32 `json:"num_subscriptions"`
					Slow     int64  `json:"slow_consumers"`
					Sent     struct{ Msgs, Bytes int64 }
					Received struct{ Msgs, Bytes int64 }
				} `json:"account_statz"`
			}
			if json.Unmarshal(r.Data, &v) != nil || len(v.Accounts) == 0 {
				continue
			}
			a := v.Accounts[0] // the connection's own account
			c = counters{now, a.Received.Msgs, a.Sent.Msgs, a.Received.Bytes, a.Sent.Bytes}
			row["connections|"+name] = float32(a.Conns)
			row["subscriptions|"+name] = float32(a.NumSubs)
			row["slow_consumers|"+name] = float32(a.Slow)
		}
		s.addRates(row, name, c)
	}
	s.Server.Add(now, row)
	return nil
}

// addRates writes per-second rates since the previous sample. A counter that
// went down means the server restarted: start a new baseline, no spike.
func (s *Sampler) addRates(row map[string]float32, name string, c counters) {
	p, ok := s.prev[name]
	s.prev[name] = c
	dt := c.at.Sub(p.at).Seconds()
	if !ok || dt <= 0 || c.inMsgs < p.inMsgs || c.outMsgs < p.outMsgs || c.inBytes < p.inBytes || c.outBytes < p.outBytes {
		return
	}
	row["msgs_in|"+name] = float32(float64(c.inMsgs-p.inMsgs) / dt)
	row["msgs_out|"+name] = float32(float64(c.outMsgs-p.outMsgs) / dt)
	row["bytes_in|"+name] = float32(float64(c.inBytes-p.inBytes) / dt)
	row["bytes_out|"+name] = float32(float64(c.outBytes-p.outBytes) / dt)
}

type consumerPoint struct {
	key                          string
	pending, ackPending, redeliv uint64
}

func (s *Sampler) sampleJetStream(now time.Time) error {
	row := map[string]float32{}
	var consumers []consumerPoint
	var err error
	if s.mon.Source() == monitor.System {
		consumers, err = s.jszSystem(row)
	} else {
		consumers, err = s.jsAccount(row)
	}
	if err != nil {
		return err
	}
	// Keep the busiest consumers only, to cap memory.
	sort.Slice(consumers, func(i, j int) bool { return consumers[i].pending > consumers[j].pending })
	for _, c := range consumers[:min(len(consumers), maxConsumers)] {
		row["consumer_pending|"+c.key] = float32(c.pending)
		row["consumer_ack_pending|"+c.key] = float32(c.ackPending)
		row["consumer_redelivered|"+c.key] = float32(c.redeliv)
	}
	s.JetStream.Add(now, row)
	return nil
}

// jszSystem reads JSZ from every server: usage per server and per account,
// and each consumer from its own leader, the only replica that tracks pending.
func (s *Sampler) jszSystem(row map[string]float32) ([]consumerPoint, error) {
	res, err := s.mon.PingWith("JSZ", map[string]bool{"accounts": true, "streams": true, "consumer": true})
	if err != nil {
		return nil, err
	}
	var consumers []consumerPoint
	for _, r := range res.Replies {
		if r.Error != nil {
			continue
		}
		var v struct {
			Memory   uint64 `json:"memory"`
			Storage  uint64 `json:"storage"`
			Accounts []struct {
				Name    string `json:"name"`
				Memory  uint64 `json:"memory"`
				Storage uint64 `json:"storage"`
				Streams []struct {
					Name      string                   `json:"name"`
					Consumers []jetstream.ConsumerInfo `json:"consumer_detail"`
				} `json:"stream_detail"`
			} `json:"account_details"`
		}
		if json.Unmarshal(r.Data, &v) != nil {
			continue
		}
		row["js_memory|"+r.Server.Name] = float32(v.Memory)
		row["js_storage|"+r.Server.Name] = float32(v.Storage)
		for _, a := range v.Accounts {
			row["js_account_memory|"+a.Name] += float32(a.Memory)
			row["js_account_storage|"+a.Name] += float32(a.Storage)
			for _, st := range a.Streams {
				for _, ci := range st.Consumers {
					// Unclustered (single server) consumers have no Cluster info.
					if ci.Cluster == nil || ci.Cluster.Leader == "" || ci.Cluster.Leader == r.Server.Name {
						consumers = append(consumers, point(a.Name+"/"+st.Name, &ci))
					}
				}
			}
		}
	}
	return consumers, nil
}

// jsAccount is the fallback for users without the system account: their own
// account's usage and consumers through the JetStream API.
func (s *Sampler) jsAccount(row map[string]float32) ([]consumerPoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	js, err := jetstream.New(s.nc)
	if err != nil {
		return nil, err
	}
	info, err := js.AccountInfo(ctx)
	if err != nil {
		return nil, err
	}
	row["js_account_memory|account"] = float32(info.Memory)
	row["js_account_storage|account"] = float32(info.Store)

	var consumers []consumerPoint
	streams := js.ListStreams(ctx)
	for si := range streams.Info() {
		st, err := js.Stream(ctx, si.Config.Name)
		if err != nil {
			return nil, err
		}
		lister := st.ListConsumers(ctx)
		for ci := range lister.Info() {
			consumers = append(consumers, point(si.Config.Name, ci))
		}
		if err := lister.Err(); err != nil {
			return nil, err
		}
	}
	return consumers, streams.Err()
}

func point(stream string, ci *jetstream.ConsumerInfo) consumerPoint {
	return consumerPoint{
		key:        stream + "/" + ci.Name,
		pending:    ci.NumPending,
		ackPending: uint64(ci.NumAckPending),
		redeliv:    uint64(ci.NumRedelivered),
	}
}
