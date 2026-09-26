package main

import (
	"fmt"
	"io"
)

// The fold (M-V1-SIMPLIFY-S5 M3, Phase 3 item 3).
//
// `trace`, `observatory`, `dashboard` and `eval-chains` are four separate
// top-level commands that all answer the same question — what did a run
// actually do — over the same three stores. `chains` is the one the fleet
// reaches for: measured with `git grep` over tracked files, 216 references
// against 66 for `trace`, 63 for `observatory`, 37 for `eval-chains` and 32
// for `dashboard`. So the four fold INTO `chains` as namespaces:
//
//	ailang chains trace status
//	ailang chains observatory backfill
//	ailang chains dashboard spans --provider gemini
//	ailang chains eval list
//
// and each keeps its top-level spelling as an ALIAS for one release (D1). The
// alias is not a copy: the top-level row routes through `chains`, which routes
// straight back out to the same function, so the two spellings are the same
// code path by construction rather than by two edits that can drift. The
// round trip is argv-exact — runAsName rewrites os.Args and re-parses
// flag.CommandLine, so `ailang trace list --limit 5` rebuilds exactly the argv
// it started with. commands_folded_test.go byte-diffs the pairs.
//
// Why the round trip rather than pointing the row straight at traceCommand:
// 44 call sites across cmd/ailang read os.Args[2:] / os.Args[3:] and
// flag.Arg(1) rather than the tail the dispatcher hands them (the reason
// runAsName exists at all, S5 M2). Routing both spellings through one place
// means a later change to the chains dispatcher reaches the aliases too,
// instead of leaving them behind in the release where they were deprecated.
type foldedNamespace struct {
	// Sub is the word after `ailang chains`.
	Sub string
	// Legacy is the top-level spelling kept as an alias for one release. It
	// is the name runAsName rebuilds argv under, so the folded commands see
	// exactly the argv they saw before the fold.
	Legacy string
	// Summary is the line `ailang chains --help` prints.
	Summary string
	// Run is the folded command's own dispatcher.
	Run func()
	// Help prints the namespace's subcommand list for `--help`, exit 0. It is
	// separate from Run's no-argument path only because that path now exits 1
	// (M1 finding 3: `trace`, `observatory` and `dashboard` exited 0 for "you
	// named no subcommand" while `chains`, `pkg` and `daemon` exited 1).
	Help func()
}

func foldedNamespaces() []foldedNamespace {
	return []foldedNamespace{
		{
			Sub:     "trace",
			Legacy:  "trace",
			Summary: "Distributed trace management (GCP + local spans)",
			Run:     traceCommand,
			Help:    printTraceHelp,
		},
		{
			Sub:     "observatory",
			Legacy:  "observatory",
			Summary: "Observatory maintenance: sync-chat, backfill, repair-ids, cleanup",
			Run:     observatoryCommand,
			Help:    printObservatoryHelp,
		},
		{
			Sub:     "dashboard",
			Legacy:  "dashboard",
			Summary: "Query the dashboard server's HTTP API (spans, inbox, stats)",
			Run:     dashboardCommand,
			Help:    printDashboardHelp,
		},
		{
			Sub:     "eval",
			Legacy:  "eval-chains",
			Summary: "Chains filtered to eval runs: list, view, failures, stats",
			Run:     evalChainsCommand,
			Help:    printEvalChainsHelp,
		},
	}
}

// lookupFoldedNamespace resolves `ailang chains <sub>`.
func lookupFoldedNamespace(sub string) *foldedNamespace {
	ns := foldedNamespaces()
	for i := range ns {
		if ns[i].Sub == sub {
			return &ns[i]
		}
	}
	return nil
}

// runFoldedNamespace is `ailang chains <ns> <args...>`.
func runFoldedNamespace(ns *foldedNamespace, args []string) {
	if wantsHelp(args) {
		ns.Help()
		return
	}
	// The error is always nil: noArgs never returns one, and every folded
	// command reports and exits on its own failures, exactly as it did when
	// the dispatcher called it directly.
	_ = runAsName(ns.Legacy, noArgs(ns.Run), args)
}

// runFoldedLegacy is the top-level alias: `ailang trace ...` becomes
// `ailang chains trace ...` and comes straight back out here.
func runFoldedLegacy(sub string) func([]string) error {
	return func(args []string) error {
		return runAsName("chains", noArgs(chainsCommand), append([]string{sub}, args...))
	}
}

// printFoldedNamespaces writes the fold's section of `ailang chains --help`.
func printFoldedNamespaces(w io.Writer) {
	fmt.Fprintln(w, "Folded namespaces (each also keeps its own top-level spelling for one release):")
	for _, ns := range foldedNamespaces() {
		label := "chains " + ns.Sub
		fmt.Fprintf(w, "  %s%s%s\n", label, helpPad(label, 22), ns.Summary)
	}
}
