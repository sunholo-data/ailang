package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// chainsImportMotokoCommand imports a motoko run log into the observatory DB so the
// motoko run becomes a first-class chain (ailang chains view/chat/tree/diagnose).
func chainsImportMotokoCommand() {
	fs := flag.NewFlagSet("chains import-motoko", flag.ExitOnError)
	fs.Parse(flag.Args()[2:])

	arg := fs.Arg(0)
	if arg == "" {
		fmt.Fprintln(os.Stderr, "Usage: ailang chains import-motoko <session.jsonl | session-id>")
		fmt.Fprintln(os.Stderr, "Imports a .motoko/logfile/session_*.jsonl run into ailang chains.")
		os.Exit(1)
	}

	path, err := resolveMotokoLog(arg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	store, err := observatory.OpenDefaultStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open observatory: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	res, err := store.ImportMotokoSession(context.Background(), path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: import failed: %v\n", err)
		os.Exit(1)
	}

	errSuffix := ""
	if res.Error != "" {
		errSuffix = "/" + res.Error
	}
	fmt.Printf("Imported motoko run -> chain %s\n", res.ChainID)
	fmt.Printf("  session=%s  status=%s (%s%s)  steps=%d  tools=%d  tokens=%d->%d  peak_input=%d\n",
		res.SessionLabel, res.Status, res.FinishReason, errSuffix,
		res.Steps, res.ToolCalls, res.TokensIn, res.TokensOut, res.PeakInput)
	fmt.Println()
	fmt.Printf("  ailang chains view %s\n", res.ChainID)
	fmt.Printf("  ailang chains chat %s --stage 1\n", res.ChainID)
	fmt.Printf("  ailang chains diagnose %s\n", res.ChainID)
}

// resolveMotokoLog resolves a session id or path to a motoko log file. A literal .jsonl
// path is used as-is; otherwise it globs the standard motoko logfile locations and returns
// the most recent match.
func resolveMotokoLog(arg string) (string, error) {
	if strings.HasSuffix(arg, ".jsonl") {
		if _, err := os.Stat(arg); err == nil {
			return arg, nil
		}
	}
	home, _ := os.UserHomeDir()
	patterns := []string{
		filepath.Join(home, "dev", "*", ".motoko", "logfile", "*"+arg+"*.jsonl"),
		filepath.Join(home, "*", ".motoko", "logfile", "*"+arg+"*.jsonl"),
	}
	for _, p := range patterns {
		hits, _ := filepath.Glob(p)
		if len(hits) > 0 {
			return hits[len(hits)-1], nil
		}
	}
	return "", fmt.Errorf("no motoko log found for %q (looked under ~/dev/*/.motoko/logfile/)", arg)
}
