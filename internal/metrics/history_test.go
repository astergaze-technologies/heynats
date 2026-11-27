package metrics

import (
	"testing"
	"time"
)

func values(w Window, name, member string) []float32 {
	var out []float32
	for _, v := range w.Series[name][member] {
		if v == nil {
			out = append(out, -1)
		} else {
			out = append(out, *v)
		}
	}
	return out
}

func TestFineRingWrapsAround(t *testing.T) {
	h := NewHistory(20 * time.Minute) // 3 points per hour
	start := time.Unix(1_000_020, 0)
	for i := range 5 {
		h.Add(start.Add(time.Duration(i)*time.Second), map[string]float32{"x|a": float32(i)})
	}
	w := h.Window([]string{"x"}, time.Hour, start.Add(time.Minute))
	if got := values(w, "x", "a"); len(got) != 3 || got[0] != 2 || got[2] != 4 {
		t.Fatalf("got %v, want the last 3 oldest first", got)
	}
	if !w.HistorySince.Equal(start) {
		t.Errorf("history_since = %s", w.HistorySince)
	}
}

func TestMinuteAveragesAndGaps(t *testing.T) {
	h := NewHistory(5 * time.Second)
	m0 := time.Unix(60*1000, 0)
	h.Add(m0, map[string]float32{"x|a": 10, "x|b": 1})
	h.Add(m0.Add(30*time.Second), map[string]float32{"x|a": 20})
	h.Add(m0.Add(time.Minute), map[string]float32{"x|a": 99}) // closes minute 0

	w := h.Window([]string{"x"}, 2*time.Hour, m0.Add(time.Minute))
	if w.Step != 60 || len(w.T) != 1 || w.T[0] != m0.Unix() {
		t.Fatalf("coarse window: step=%d t=%v", w.Step, w.T)
	}
	if got := values(w, "x", "a"); got[0] != 15 {
		t.Errorf("minute average = %v, want 15", got)
	}

	fine := h.Window([]string{"x"}, time.Hour, m0.Add(time.Minute))
	if got := values(fine, "x", "b"); len(got) != 3 || got[0] != 1 || got[1] != -1 {
		t.Errorf("member missing from a sample must be null, got %v", got)
	}
}

func TestRateRestartsAfterCounterReset(t *testing.T) {
	s := &Sampler{prev: map[string]counters{}}
	t0 := time.Unix(0, 0)
	row := map[string]float32{}
	s.addRates(row, "n1", counters{at: t0, inMsgs: 100})
	s.addRates(row, "n1", counters{at: t0.Add(10 * time.Second), inMsgs: 600})
	if row["msgs_in|n1"] != 50 {
		t.Fatalf("rate = %v, want 50/s", row["msgs_in|n1"])
	}

	row = map[string]float32{}
	s.addRates(row, "n1", counters{at: t0.Add(20 * time.Second), inMsgs: 7}) // server restarted
	if _, ok := row["msgs_in|n1"]; ok {
		t.Fatalf("reset must not produce a rate, got %v", row)
	}
	s.addRates(row, "n1", counters{at: t0.Add(30 * time.Second), inMsgs: 27})
	if row["msgs_in|n1"] != 2 {
		t.Fatalf("rate after new baseline = %v, want 2/s", row["msgs_in|n1"])
	}
}
