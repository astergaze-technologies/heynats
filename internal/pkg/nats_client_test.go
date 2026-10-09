package pkg

import "testing"

func TestAccountPingData(t *testing.T) {
	if _, ok := accountPingData([]byte(`{"error":{"code":503,"description":"no system account"}}`)); ok {
		t.Error("error reply must not yield data")
	}
	if _, ok := accountPingData([]byte(`not json`)); ok {
		t.Error("invalid json must not yield data")
	}
	data, ok := accountPingData([]byte(`{"data":{"conns":2}}`))
	if !ok || data["conns"] != float64(2) {
		t.Errorf("got %v, %v", data, ok)
	}
}

func TestNeverExpires(t *testing.T) {
	cases := map[string]struct {
		in   any
		want bool
	}{
		"json zero": {float64(0), true},
		"set":       {float64(1700000000), false},
		"missing":   {nil, false},
	}
	for name, tc := range cases {
		if got := neverExpires(tc.in); got != tc.want {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}
}
