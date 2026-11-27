package diagnose

import "strings"

// subjectsOverlap reports whether some subject matches both patterns
// ("*" is one token, ">" one or more trailing tokens).
func subjectsOverlap(a, b string) bool {
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
