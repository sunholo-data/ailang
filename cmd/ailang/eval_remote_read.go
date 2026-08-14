package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var evalRemoteReadCommands = map[string]struct{}{
	"eval": {}, "eval-analyze": {}, "eval-compare": {}, "eval-paired": {},
	"eval-matrix": {}, "eval-sweet-spot": {}, "eval-summary": {}, "eval-report": {},
	"eval-suite": {}, "eval-elo": {}, "eval-trend": {}, "eval-publish": {},
	"eval-chains": {},
}

// normalizeRemoteFlag reduces one argv token to (flagName, carriesInlineValue).
//
// Go's flag package treats `-remote` and `--remote` as THE SAME FLAG, so a guard
// that matches only the double-dash spelling is bypassable by typing one fewer
// dash. That is not cosmetic here: the D-15 message is the whole mechanism by
// which wanting remote eval read becomes a dated, attributable signal, and
// `-remote gcp` used to fall through to the subcommand's own FlagSet and die
// with a generic "flag provided but not defined". Found by the iteration-198
// evaluator and reproduced before the fix (`-remote gcp` rc=2 generic vs
// `--remote gcp` rc=1 with D-15).
//
// Returns "" for anything that is not a flag, so `--` and bare positionals are
// left alone.
func normalizeRemoteFlag(arg string) (string, bool) {
	if len(arg) < 2 || arg[0] != '-' || arg == "--" {
		return "", false
	}
	name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
	if before, _, found := strings.Cut(name, "="); found {
		return before, true
	}
	return name, false
}

func guardEvalRemoteRead(command string, args []string, warningWriter io.Writer) error {
	if _, guarded := evalRemoteReadCommands[command]; !guarded {
		return nil
	}
	displayCommand := command
	if command == "eval-chains" && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		displayCommand += " " + args[0]
	}
	for i, arg := range args {
		name, hasValue := normalizeRemoteFlag(arg)
		if name != "remote" {
			continue
		}
		if !hasValue && i+1 == len(args) {
			return fmt.Errorf("%s: --remote requires a mode", displayCommand)
		}
		return fmt.Errorf("%s remote read is view-scoped per D-15; register demand at #698 part 1", displayCommand)
	}
	if mode := os.Getenv("AILANG_CHAINS_READ"); mode != "" && mode != "local" {
		fmt.Fprintf(warningWriter, "WARNING: %s ignores AILANG_CHAINS_READ=%s; remote read is view-scoped per D-15; register demand at #698 part 1\n", displayCommand, mode)
	}
	return nil
}
