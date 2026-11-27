// Package subject matches NATS subjects against patterns ("*" is one token,
// ">" one or more trailing tokens). Kept in step with client/src/lib/subjects.ts:
// both test against testdata/subject-match.json.
package subject

import "strings"

// Match reports whether a literal subject matches a pattern.
func Match(pattern, subject string) bool {
	p, s := strings.Split(pattern, "."), strings.Split(subject, ".")
	for i, tok := range p {
		switch {
		case tok == ">":
			return len(s) > i
		case i >= len(s):
			return false
		case tok != "*" && tok != s[i]:
			return false
		}
	}
	return len(p) == len(s)
}

// Overlap reports whether some subject matches both patterns.
func Overlap(a, b string) bool {
	at, bt := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; ; i++ {
		switch {
		case i == len(at) && i == len(bt):
			return true
		case i == len(at) || i == len(bt):
			return false
		case at[i] == ">" || bt[i] == ">":
			return true
		case at[i] != "*" && bt[i] != "*" && at[i] != bt[i]:
			return false
		}
	}
}
