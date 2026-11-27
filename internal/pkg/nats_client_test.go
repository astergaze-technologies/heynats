package pkg

import "testing"

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
