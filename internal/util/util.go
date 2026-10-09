package util

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/nats-io/nats.go"
)

var semVerRe = regexp.MustCompile(`\Av?([0-9]+)(?:\.([0-9]+))?(?:\.([0-9]+))?`)

// versionComponents parses "2.11.3"; missing minor/patch parts count as 0.
func versionComponents(version string) (major, minor, patch int, err error) {
	m := semVerRe.FindStringSubmatch(version)
	if m == nil {
		return 0, 0, 0, errors.New("invalid semver")
	}
	parts := [3]int{}
	for i, s := range m[1:] {
		if s == "" {
			continue
		}
		if parts[i], err = strconv.Atoi(s); err != nil {
			return -1, -1, -1, err
		}
	}
	return parts[0], parts[1], parts[2], nil
}

func ServerMinVersion(nc *nats.Conn, major, minor, patch int) bool {
	smajor, sminor, spatch, _ := versionComponents(nc.ConnectedServerVersion())
	if smajor < major || (smajor == major && sminor < minor) || (smajor == major && sminor == minor && spatch < patch) {
		return false
	}

	return true
}
