package main

import (
	"flag"
	"fmt"
	"strings"
)

// misplacedRunFlag returns the first token after the file path that names one
// of `ailang run`'s own flags, or "" if there is none.
//
// Go's flag package stops at the first positional argument, and `run` hands
// everything after the file to the program as its arguments. So
// `ailang run --entry main file.ail --args-json '{…}'` used to give the entry
// no arguments and the program a stray "--args-json", and the failure surfaced
// later as ARG_DECODE_MISMATCH "got <nil>" (stapledons_godot, 2026-10-01).
// Tokens after a literal "--" are the program's by explicit request and are
// never reported.
func misplacedRunFlag(fs *flag.FlagSet, afterFile []string) string {
	for _, tok := range afterFile {
		if tok == "--" {
			return ""
		}
		if !strings.HasPrefix(tok, "-") || tok == "-" {
			continue
		}
		name := strings.TrimLeft(tok, "-")
		if i := strings.IndexByte(name, '='); i >= 0 {
			name = name[:i]
		}
		if fs.Lookup(name) != nil {
			return tok
		}
	}
	return ""
}

func misplacedRunFlagError(tok string) error {
	return fmt.Errorf("flag %s comes after the file path, so it would be passed to the program instead of to `ailang run`; "+
		"put it before the file (ailang run [flags] <file.ail>), or put `--` before program arguments that look like flags", tok)
}
