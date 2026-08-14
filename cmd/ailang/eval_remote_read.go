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

func guardEvalRemoteRead(command string, args []string, warningWriter io.Writer) error {
	if _, guarded := evalRemoteReadCommands[command]; !guarded {
		return nil
	}
	displayCommand := command
	if command == "eval-chains" && len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		displayCommand += " " + args[0]
	}
	for i, arg := range args {
		if arg == "--remote" || strings.HasPrefix(arg, "--remote=") {
			if arg == "--remote" && i+1 == len(args) {
				return fmt.Errorf("%s: --remote requires a mode", displayCommand)
			}
			return fmt.Errorf("%s remote read is view-scoped per D-15; register demand at #698 part 1", displayCommand)
		}
	}
	if mode := os.Getenv("AILANG_CHAINS_READ"); mode != "" && mode != "local" {
		fmt.Fprintf(warningWriter, "WARNING: %s ignores AILANG_CHAINS_READ=%s; remote read is view-scoped per D-15; register demand at #698 part 1\n", displayCommand, mode)
	}
	return nil
}
