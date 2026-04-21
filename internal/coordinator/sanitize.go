package coordinator

import "strings"

// SanitizeLog strips CR, LF, and ASCII control characters from a string so it
// cannot forge log lines when interpolated into a log message. Wrap any
// user-controlled value (task IDs, labels, repo names, issue titles) at the
// log call site. The generic constraint allows named string types (e.g.
// ApprovalEventType) without an explicit cast.
func SanitizeLog[S ~string](s S) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || (r >= 0 && r < 0x20) {
			return -1
		}
		return r
	}, string(s))
}
