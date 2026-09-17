package main

import "os"

// evalCommands are the benchmark and eval-analysis commands. They are platform
// commands (Group ""), exactly as the pre-S5 isLanguageCommand switch had them.
// S5 M2 folds them into an `eval` group with these names kept as aliases — do
// not rename anything here before that lands.
func evalCommands() []Command {
	return []Command{
		{
			Name:    "eval",
			Summary: "Run AI benchmarks (AILANG vs Python)",
			Run:     noArgs(runEval),
		},
		{
			Name:    "eval-analyze",
			Summary: "Analyze eval results and generate design docs",
			Run:     noArgs(runEvalAnalyze),
		},
		{
			Name:    "eval-compare",
			Summary: "Compare two eval runs",
			Run:     noArgs(runEvalCompare),
		},
		{
			Name:    "eval-paired",
			Summary: "Paired A/B comparison + McNemar (not aggregate rates)",
			Run:     noArgs(runEvalPaired),
		},
		{
			Name:    "eval-censored-pairs",
			Summary: "Censored fmt pairs with treatment/order integrity gates",
			Run:     noArgs(runEvalCensoredPairs),
		},
		{
			Name:    "eval-matrix",
			Summary: "Performance matrix with statistics",
			Run:     noArgs(runEvalMatrix),
		},
		{
			Name:    "eval-sweet-spot",
			Summary: "Cost-vs-time-vs-success sweet-spot ranking",
			Run:     noArgs(runEvalSweetSpot),
		},
		{
			Name:    "eval-summary",
			Summary: "Summarize eval results",
			Run:     noArgs(runEvalSummary),
		},
		{
			Name:    "eval-report",
			Summary: "Generate a comprehensive eval report",
			Run:     noArgs(runEvalReport),
		},
		{
			Name:    "eval-suite",
			Summary: "Run the full benchmark suite (parallel)",
			Run:     noArgs(runEvalSuite),
		},
		{
			Name:    "browser-profile",
			Summary: "Manage persistent authenticated browser identities",
			// os.Args[2:], not the dispatched tail: this is what the pre-S5
			// switch passed, and the two differ when a global flag precedes
			// the command (`ailang --compact browser-profile list` hands the
			// command its own name). Preserved byte-for-byte here; the fix
			// belongs with the flag work in S5 M5.
			Run: func([]string) error {
				runBrowserProfile(os.Args[2:])
				return nil
			},
		},
		{
			Name:    "eval-elo",
			Summary: "Per-language (AILANG vs Python) ELO leaderboard and benchmark difficulty",
			Run:     noArgs(runEvalELO),
		},
		{
			Name:    "eval-trend",
			Summary: "Failure-feedback candidate triage across eval history",
			Run:     noArgs(runEvalTrend),
		},
		{
			Name:    "eval-publish",
			Summary: "Per-release Docusaurus publication of eval results",
			Run:     noArgs(runEvalPublish),
		},
		{
			Name:    "eval-chains",
			Summary: "Link eval runs to their execution chains",
			Run:     noArgs(evalChainsCommand),
		},
	}
}
