// Package metrics samples server and JetStream numbers on an interval and
// keeps them in memory: fine points for the last hour, minute averages for
// the last day.
package metrics

import (
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

var nan = float32(math.NaN())

// frames is a ring of rows sharing one time axis; each key ("series|member")
// owns a column of the same length, NaN where it had no value.
type frames struct {
	size, next, n int
	t             []int64 // unix seconds
	cols          map[string][]float32
}

func newFrames(size int) *frames {
	return &frames{size: size, t: make([]int64, size), cols: map[string][]float32{}}
}

func (f *frames) push(t int64, row map[string]float32) {
	i := f.next
	f.t[i] = t
	for k, col := range f.cols {
		col[i] = nan
		if v, ok := row[k]; ok {
			col[i] = v
		}
	}
	for k, v := range row {
		if _, ok := f.cols[k]; !ok {
			col := make([]float32, f.size)
			for j := range col {
				col[j] = nan
			}
			col[i] = v
			f.cols[k] = col
		}
	}
	f.next = (i + 1) % f.size
	f.n = min(f.n+1, f.size)
}

// rows returns ring indexes oldest first with t >= from.
func (f *frames) rows(from int64) []int {
	var idx []int
	for j := range f.n {
		i := (f.next - f.n + j + f.size) % f.size
		if f.t[i] >= from {
			idx = append(idx, i)
		}
	}
	return idx
}

// History holds every series of one sampler. Safe for concurrent use.
type History struct {
	mu       sync.Mutex
	step     time.Duration
	fine     *frames // one row per sample, covering an hour
	coarse   *frames // one row per minute, covering a day
	since    time.Time
	minute   int64
	sum      map[string]float64
	count    map[string]int
	lastSeen map[string]int64
}

const (
	fineSpan   = time.Hour
	coarseSpan = 24 * time.Hour
)

func NewHistory(step time.Duration) *History {
	return &History{
		step:     step,
		fine:     newFrames(int(fineSpan / step)),
		coarse:   newFrames(int(coarseSpan / time.Minute)),
		sum:      map[string]float64{},
		count:    map[string]int{},
		lastSeen: map[string]int64{},
	}
}

// Budget is the worst-case memory of one sampler: servers x 13 server series,
// plus 3 series per consumer and 2 per server and account for JetStream.
func Budget(interval time.Duration, servers, accounts, consumers int) int {
	perKey := func(step time.Duration) int { return 4 * (int(fineSpan/step) + int(coarseSpan/time.Minute)) }
	return servers*13*perKey(interval) + (3*consumers+2*servers+2*accounts)*perKey(jsEvery*interval)
}

func (h *History) Add(at time.Time, row map[string]float32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.since.IsZero() {
		h.since = at
	}
	t := at.Unix()
	if m := t / 60; m != h.minute {
		h.flushMinute()
		h.minute = m
	}
	h.fine.push(t, row)
	for k, v := range row {
		if !math.IsNaN(float64(v)) {
			h.sum[k] += float64(v)
			h.count[k]++
		}
		h.lastSeen[k] = t
	}
}

// flushMinute writes the finished minute's averages and forgets members
// (e.g. deleted consumers) not seen for a whole day.
func (h *History) flushMinute() {
	if len(h.count) == 0 {
		return
	}
	row := make(map[string]float32, len(h.count))
	for k, n := range h.count {
		row[k] = float32(h.sum[k] / float64(n))
	}
	h.coarse.push(h.minute*60, row)
	clear(h.sum)
	clear(h.count)

	cutoff := h.minute*60 - int64(coarseSpan/time.Second)
	for k, seen := range h.lastSeen {
		if seen < cutoff {
			delete(h.lastSeen, k)
			delete(h.fine.cols, k)
			delete(h.coarse.cols, k)
		}
	}
}

// Window is columnar, ready for a chart: Series[name][member][i] is the value
// at T[i], nil where the member had none.
type Window struct {
	HistorySince time.Time                        `json:"history_since"`
	Step         int64                            `json:"step"` // seconds between points
	T            []int64                          `json:"t"`
	Series       map[string]map[string][]*float32 `json:"series"`
}

// Window returns the last span of the named series, from the fine ring when
// it covers the span, else from the minute ring.
func (h *History) Window(names []string, span time.Duration, now time.Time) Window {
	h.mu.Lock()
	defer h.mu.Unlock()
	f, step := h.fine, h.step
	if span > fineSpan {
		f, step = h.coarse, time.Minute
	}
	rows := f.rows(now.Add(-span).Unix())
	w := Window{HistorySince: h.since, Step: int64(step / time.Second), T: make([]int64, len(rows)), Series: map[string]map[string][]*float32{}}
	for j, i := range rows {
		w.T[j] = f.t[i]
	}
	for _, name := range names {
		w.Series[name] = map[string][]*float32{}
	}
	keys := make([]string, 0, len(f.cols))
	for k := range f.cols {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		name, member, _ := strings.Cut(k, "|")
		members, wanted := w.Series[name]
		if !wanted {
			continue
		}
		col := f.cols[k]
		vals := make([]*float32, len(rows))
		for j, i := range rows {
			if v := col[i]; !math.IsNaN(float64(v)) {
				vals[j] = &v
			}
		}
		members[member] = vals
	}
	return w
}
