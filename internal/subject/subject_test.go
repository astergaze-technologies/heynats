package subject

import (
	"encoding/json"
	"os"
	"testing"
)

// The same cases run against the TypeScript matcher.
func TestMatchTable(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/subject-match.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Pattern, Subject string
		Match            bool
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if got := Match(c.Pattern, c.Subject); got != c.Match {
			t.Errorf("Match(%q, %q) = %v, want %v", c.Pattern, c.Subject, got, c.Match)
		}
		if c.Match && !Overlap(c.Pattern, c.Subject) {
			t.Errorf("a match must overlap: %q, %q", c.Pattern, c.Subject)
		}
	}
}
