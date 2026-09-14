package main

// Refuse a coordinator flag the subcommand does not understand.
//
// Nine subcommands parse their own arguments with a `switch args[i]` and no
// default case, so an unrecognised flag is skipped in silence. The measured
// consequence is not a usability nit: `ailang coordinator list --remote gcp`
// answered from this machine's SQLite and returned a task from May 2026 while
// production held live work, and `coordinator diff <task> --remote gcp` said
// "failed to get task: sql: no rows in result set" for a task that exists. In
// both cases the operator asked for the prod plane, was given the local one,
// and was told nothing.
//
// approve, reject and approvals DO honour --remote. That asymmetry is the worst
// part: the flag works often enough to be trusted, then silently does not.
//
// This guard is the cheap half of the fix and applies to all nine at once — a
// wrong answer becomes a loud refusal. The expensive half, making a command
// actually read the remote plane, is per-command work; `list` is done, and
// until the rest follow this says so in as many words rather than pretending.

import (
	"fmt"
	"sort"
	"strings"
)

// handParsedCoordinatorFlags declares what each hand-rolled parser understands.
// The bool is "this flag consumes the next argument".
//
// Keep in step with the parsers themselves. TestCoordinatorFlagGuardIsExhaustive
// fails if a subcommand in the dispatch switch appears in neither this map nor
// flagSetCoordinatorSubcommands, so a new subcommand cannot quietly inherit the
// silent-swallow behaviour.
var handParsedCoordinatorFlags = map[string]map[string]bool{
	"list": {
		"--state-dir": true, "--json": false, "--limit": true, "--status": true,
		"--running": false, "--pending": false, "--completed": false, "--failed": false,
		"--remote": true, "--help": false, "-h": false,
	},
	"pending":  {"--state-dir": true, "--json": false, "--help": false, "-h": false},
	"diff":     {"--state-dir": true, "--stat": false, "--help": false, "-h": false},
	"logs":     {"--state-dir": true, "--follow": false, "--json": false, "--limit": true, "--help": false, "-h": false},
	"worktree": {"--state-dir": true, "--open": false, "--help": false, "-h": false},
	"retry":    {"--state-dir": true, "--all": false, "--yes": false, "--help": false, "-h": false},
	"reopen":   {"--state-dir": true, "--yes": false, "--help": false, "-h": false},
	"cleanup":  {"--state-dir": true, "--yes": false, "--older-than": true, "--help": false, "-h": false},
	"status":   {"--state-dir": true, "--json": false, "--help": false, "-h": false},
}

// flagSetCoordinatorSubcommands use flag.FlagSet, which already refuses unknown
// flags. They must NOT be double-guarded: the FlagSet owns their vocabulary and
// duplicating it here would drift.
var flagSetCoordinatorSubcommands = map[string]bool{
	"start": true, "stop": true, "config": true, "routing": true,
	"agents": true, "agent-set": true, "agent-check": true, "lint": true,
	"approvals": true, "approve": true, "reject": true,
	"watcher-status": true, "sync-threads": true, "execute-job": true,
	"workers": true, "help": true, "--help": true, "-h": true,
}

// remoteAwareSubcommands actually read the plane named by --remote. Everything
// else only knows the flag well enough to refuse it honestly.
var remoteAwareSubcommands = []string{"approve", "approvals", "list", "reject"}

func rejectUnknownCoordinatorFlags(sub string, args []string) error {
	known, ok := handParsedCoordinatorFlags[sub]
	if !ok {
		return nil // flag.FlagSet subcommand: it does its own refusing
	}
	skipNext := false
	for _, a := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if !strings.HasPrefix(a, "-") {
			continue
		}
		name := a
		if i := strings.IndexByte(a, '='); i > 0 {
			name = a[:i] // --limit=5
		}
		takesValue, isKnown := known[name]
		if isKnown {
			if takesValue && name == a {
				skipNext = true
			}
			continue
		}
		return unknownCoordinatorFlagError(sub, name)
	}
	return nil
}

func unknownCoordinatorFlagError(sub, flag string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "`coordinator %s` does not understand %s", sub, flag)

	if flag == "--remote" {
		// The specific mistake worth naming, because the flag is real elsewhere.
		fmt.Fprintf(&b, ".\n  It is a real flag on: %s — but NOT here, and this command\n",
			strings.Join(remoteAwareSubcommands, ", "))
		fmt.Fprintf(&b, "  reads this machine's local SQLite. Until now it accepted --remote and\n")
		fmt.Fprintf(&b, "  answered from the local store anyway, which is how a task from May was\n")
		fmt.Fprintf(&b, "  reported as the state of production.\n")
		fmt.Fprintf(&b, "  For the cloud plane use: ailang coordinator approvals --remote gcp")
		return fmt.Errorf("%s", b.String())
	}

	names := make([]string, 0, len(handParsedCoordinatorFlags[sub]))
	for k := range handParsedCoordinatorFlags[sub] {
		if k != "-h" {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	fmt.Fprintf(&b, ".\n  Known flags: %s", strings.Join(names, " "))
	return fmt.Errorf("%s", b.String())
}
