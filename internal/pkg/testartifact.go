package pkg

import "strings"

// NamedTestBodyPrefix starts the name of the temporary copy of a test module
// that `ailang test` compiles a named-test body from. Releases up to v0.51.0
// wrote that copy into the package directory, so an interrupted run could
// leave it there (#1502); the copy now lives in a private temp dir, and this
// predicate lets tools recognise leftovers from older binaries.
const NamedTestBodyPrefix = "_namedtest_body_"

// IsNamedTestBodyFile reports whether base (a file's base name) is a leftover
// named-test body copy: "_namedtest_body_<digits>.ail".
func IsNamedTestBodyFile(base string) bool {
	rest, ok := strings.CutPrefix(base, NamedTestBodyPrefix)
	if !ok {
		return false
	}
	digits, ok := strings.CutSuffix(rest, ".ail")
	if !ok || digits == "" {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
