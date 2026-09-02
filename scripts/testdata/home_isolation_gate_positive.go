// Positive fixture for scripts/check_home_isolation.sh. It is NOT compiled
// (Go's toolchain ignores every directory named testdata) and it is pruned from
// the gate's own scan; its only job is to make the matcher prove it fires, in
// each shape the gate claims to cover. Do not "fix" the calls below.
package fixture

type tb interface{ Setenv(string, string) }

// shape 1: the bare single-line call that reddened dev four times.
func bare(t tb, dir string) {
	t.Setenv("HOME", dir)
}

// shape 2: os.Setenv, which the first sweep missed entirely.
func viaOS(os interface{ Setenv(string, string) error }, dir string) {
	_ = os.Setenv("HOME", dir)
}

// shape 3: gofmt-canonical and newline-spanning, which a line-oriented matcher
// cannot see. gofmt leaves this form alone, so a contributor reaches it by
// accident, not by evasion.
func multiline(t tb, dir string) {
	t.Setenv(
		"HOME",
		dir,
	)
}
