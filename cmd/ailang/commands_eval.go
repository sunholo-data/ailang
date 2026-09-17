package main

import "os"

// evalCommands are the benchmark and eval-analysis commands.
//
// S5 M2 made `eval` a GROUP: `ailang eval suite`, `ailang eval report`,
// `ailang eval elo` and the rest now work, and the subcommand name is the
// top-level name with the `eval-` prefix removed (groupSubName derives it, so
// there is no second list to keep in step). Every `eval-*` spelling is
// untouched and stays a top-level route — 182 fleet references call
// `ailang eval-suite` alone, and D1 says not one of them changes.
//
// `ailang eval --benchmark fizzbuzz --mock` also still runs a benchmark: a
// first argument that is not a subcommand falls through to runEval, which is
// what that spelling has always done.
func evalCommands() []Command {
	return []Command{
		{
			Name:    "eval",
			Summary: "Benchmarks: run one, or a subcommand (suite, report, elo, ...)",
			Run:     evalGroupCommand,
		},
		{
			Name:    "eval-analyze",
			Group:   groupEval,
			Summary: "Analyze eval results and generate design docs",
			Run:     noArgs(runEvalAnalyze),
		},
		{
			Name:    "eval-compare",
			Group:   groupEval,
			Summary: "Compare two eval runs",
			Run:     noArgs(runEvalCompare),
		},
		{
			Name:    "eval-paired",
			Group:   groupEval,
			Summary: "Paired A/B comparison + McNemar (not aggregate rates)",
			Run:     noArgs(runEvalPaired),
		},
		{
			Name:    "eval-censored-pairs",
			Group:   groupEval,
			Summary: "Censored fmt pairs with treatment/order integrity gates",
			Run:     noArgs(runEvalCensoredPairs),
		},
		{
			Name:    "eval-matrix",
			Group:   groupEval,
			Summary: "Performance matrix with statistics",
			Run:     noArgs(runEvalMatrix),
		},
		{
			Name:    "eval-sweet-spot",
			Group:   groupEval,
			Summary: "Cost-vs-time-vs-success sweet-spot ranking",
			Run:     noArgs(runEvalSweetSpot),
		},
		{
			Name:    "eval-summary",
			Group:   groupEval,
			Summary: "Summarize eval results",
			Run:     noArgs(runEvalSummary),
		},
		{
			Name:    "eval-report",
			Group:   groupEval,
			Summary: "Generate a comprehensive eval report",
			Run:     noArgs(runEvalReport),
		},
		{
			Name:    "eval-suite",
			Group:   groupEval,
			Summary: "Run the full benchmark suite (parallel)",
			Run:     noArgs(runEvalSuite),
		},
		{
			Name:  "browser-profile",
			Group: groupEval,
			// The tail, not os.Args[2:]. M1 kept os.Args[2:] to preserve the
			// pre-S5 switch byte-for-byte; M2 has to change it, because the
			// group route `ailang eval browser-profile list` hands this
			// command an os.Args whose [2] is "browser-profile", not "list".
			// runAsTopLevel rewrites os.Args for every OTHER command so they
			// can keep reading it; this one reads the tail instead, which is
			// identical for every invocation that does not put a global flag
			// before the command name — and in that one case the tail is the
			// correct reading and os.Args[2:] was the bug.
			Summary: "Manage persistent authenticated browser identities",
			Run: func(args []string) error {
				runBrowserProfile(args)
				return nil
			},
		},
		{
			Name:    "eval-elo",
			Group:   groupEval,
			Summary: "Per-language (AILANG vs Python) ELO leaderboard and benchmark difficulty",
			Run:     noArgs(runEvalELO),
		},
		{
			Name:    "eval-trend",
			Group:   groupEval,
			Summary: "Failure-feedback candidate triage across eval history",
			Run:     noArgs(runEvalTrend),
		},
		{
			Name:    "eval-publish",
			Group:   groupEval,
			Summary: "Per-release Docusaurus publication of eval results",
			Run:     noArgs(runEvalPublish),
		},
		{
			Name:    "eval-chains",
			Group:   groupEval,
			Summary: "Link eval runs to their execution chains",
			Run:     noArgs(evalChainsCommand),
		},
	}
}

// evalGroupCommand is `ailang eval`. It is both a group entry point and a
// command, which is why it is written by hand instead of coming from
// groupEntryCommands().
//
// Order matters: a KNOWN subcommand wins, then `--help`, and anything else
// falls through to runEval. The fall-through is the compatibility guarantee —
// `ailang eval --benchmark fizzbuzz --mock` and `ailang eval --model X` reach
// the benchmark runner exactly as before, and only a bare word that matches a
// subcommand is routed.
func evalGroupCommand(args []string) error {
	if len(args) > 0 {
		if args[0] == "run" {
			// `ailang eval run ...` is spelled `ailang eval ...` on the wire:
			// runEval parses os.Args[2:], so the rewrite drops the "run".
			return runAsName("eval", noArgs(runEval), args[1:])
		}
		if c := lookupGroupMember(groupEval, args[0]); c != nil {
			if wantsHelp(args[1:]) && helpFallbackCommands[c.Name] {
				printCommandHelp(os.Stdout, c)
				return nil
			}
			return runAsTopLevel(c, args[1:])
		}
	}
	if wantsHelp(args) {
		printGroupHelp(os.Stdout, groupEval)
		return nil
	}
	runEval()
	return nil
}
