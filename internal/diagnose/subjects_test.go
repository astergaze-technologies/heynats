package diagnose

import "testing"

func TestSubjectsOverlap(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"orders.>", "orders.new", true},
		{"orders.*", "orders.new.eu", false},
		{"orders.*.eu", "orders.new.*", true},
		{">", "anything.at.all", true},
		{"orders.>", "legacy.>", false},
		{"orders", "orders.new", false},
		{"a.b", "a.b", true},
		{"a.*", "*.b", true},
	} {
		if got := subjectsOverlap(tc.a, tc.b); got != tc.want {
			t.Errorf("overlap(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		if got := subjectsOverlap(tc.b, tc.a); got != tc.want {
			t.Errorf("overlap(%q, %q) not symmetric", tc.b, tc.a)
		}
	}
}
