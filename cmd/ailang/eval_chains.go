package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/sunholo/ailang/internal/observatory"
)

// evalChainsCommand handles the "eval-chains" subcommand.
// A convenience wrapper for chain queries filtered to eval_suite source type.
func evalChainsCommand() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: ailang eval-chains <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  list               List recent eval chains")
		fmt.Println("  view <chain-id>    View chain with eval assessments")
		fmt.Println("  failures <chain-id> Show only failing stages")
		fmt.Println("  stats <chain-id>   Pass rate by model/language/benchmark")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang eval-chains list")
		fmt.Println("  ailang eval-chains view e9c7501d")
		fmt.Println("  ailang eval-chains failures e9c7501d")
		fmt.Println("  ailang eval-chains stats e9c7501d")
		os.Exit(1)
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "list":
		evalChainsListCommand()
	case "view":
		evalChainsViewCommand()
	case "failures":
		evalChainsFailuresCommand()
	case "stats":
		evalChainsStatsCommand()
	default:
		fmt.Printf("Unknown eval-chains subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func evalChainsListCommand() {
	fs := flag.NewFlagSet("eval-chains list", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Maximum number of chains to show")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	store, err := observatory.OpenDefaultStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open observatory: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	chains, err := store.ListEvalChains(ctx, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list eval chains: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(chains)
		return
	}

	if len(chains) == 0 {
		fmt.Println("No eval chains found.")
		fmt.Println("Run agent evals to create chains: ailang eval-suite --agent --benchmarks fizzbuzz")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tSOURCE REF\tSTAGES\tCOST\tCREATED")
	for _, chain := range chains {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t$%.4f\t%s\n",
			truncateChainID(chain.ID),
			colorizeStatus(string(chain.Status)),
			chain.SourceRef,
			chain.StagesCompleted,
			chain.TotalCost,
			chain.CreatedAt.Format("2006-01-02 15:04"),
		)
	}
	w.Flush()
}

func evalChainsViewCommand() {
	fs := flag.NewFlagSet("eval-chains view", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang eval-chains view <chain-id>")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)

	store, err := observatory.OpenDefaultStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open observatory: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Resolve short ID
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Query stages with eval assessments
	stages, err := store.QueryEvalResults(ctx, observatory.EvalQueryOptions{
		ChainID: chainID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to query eval results: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(stages)
		return
	}

	// Get chain for header info
	chain, err := backend.GetChain(ctx, chainID, observatory.ChainReadOptions{})
	if err == nil && chain != nil {
		fmt.Printf("Eval Chain: %s\n", chain.ID[:12])
		fmt.Printf("Source: %s\n", chain.SourceRef)
		fmt.Printf("Status: %s\n", colorizeStatus(string(chain.Status)))
		fmt.Printf("Created: %s\n", chain.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	if len(stages) == 0 {
		fmt.Println("No eval stages found.")
		return
	}

	// Print stages with assessment
	passCount := 0
	for i, stage := range stages {
		a := stage.EvalAssessment
		if a == nil {
			continue
		}

		icon := red("✗")
		if a.StdoutOk {
			icon = green("✓")
			passCount++
		}
		fmt.Printf("  %s %d. %s / %s / %s\n", icon, i+1, a.BenchmarkID, a.Model, a.Language)

		if stage.Cost > 0 || stage.Turns > 0 {
			fmt.Printf("       Cost: $%.4f | Turns: %d | Duration: %s\n",
				stage.Cost, stage.Turns, formatChainDuration(stage.DurationMs))
		}
		if a.ErrorCategory != "" {
			fmt.Printf("       Error: %s\n", yellow(a.ErrorCategory))
		}
	}

	fmt.Println()
	total := len(stages)
	fmt.Printf("Pass rate: %d/%d (%.1f%%)\n", passCount, total, float64(passCount)/float64(total)*100)
}

func evalChainsFailuresCommand() {
	fs := flag.NewFlagSet("eval-chains failures", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang eval-chains failures <chain-id>")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)

	store, err := observatory.OpenDefaultStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open observatory: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	stages, err := store.QueryEvalResults(ctx, observatory.EvalQueryOptions{
		ChainID:     chainID,
		FailureOnly: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(stages)
		return
	}

	if len(stages) == 0 {
		fmt.Println("No failures found!")
		return
	}

	fmt.Printf("Failures (%d):\n\n", len(stages))
	for i, stage := range stages {
		a := stage.EvalAssessment
		if a == nil {
			continue
		}

		fmt.Printf("  %d. %s / %s / %s\n", i+1, a.BenchmarkID, a.Model, a.Language)

		// Show which step failed
		if !a.CompileOk {
			fmt.Printf("     %s compile failed\n", red("✗"))
		} else if !a.RuntimeOk {
			fmt.Printf("     %s runtime error\n", red("✗"))
		} else if !a.StdoutOk {
			fmt.Printf("     %s output mismatch\n", red("✗"))
		}

		if a.ErrorCategory != "" {
			fmt.Printf("     Category: %s\n", a.ErrorCategory)
		}
		if a.Stderr != "" {
			stderr := a.Stderr
			if len(stderr) > 200 {
				stderr = stderr[:200] + "..."
			}
			fmt.Printf("     Stderr: %s\n", stderr)
		}
		fmt.Println()
	}
}

func evalChainsStatsCommand() {
	fs := flag.NewFlagSet("eval-chains stats", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang eval-chains stats <chain-id>")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)

	store, err := observatory.OpenDefaultStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open observatory: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	stages, err := store.QueryEvalResults(ctx, observatory.EvalQueryOptions{
		ChainID: chainID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(stages) == 0 {
		fmt.Println("No eval stages found.")
		return
	}

	// Aggregate stats
	type key struct{ dim, val string }
	stats := map[key]struct{ pass, total int }{}

	for _, stage := range stages {
		a := stage.EvalAssessment
		if a == nil {
			continue
		}

		pass := 0
		if a.StdoutOk {
			pass = 1
		}

		for _, k := range []key{
			{"model", a.Model},
			{"language", a.Language},
			{"benchmark", a.BenchmarkID},
		} {
			s := stats[k]
			s.pass += pass
			s.total++
			stats[k] = s
		}
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(stats)
		return
	}

	// Print by dimension
	for _, dim := range []string{"model", "language", "benchmark"} {
		fmt.Printf("\nBy %s:\n", dim)
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  %s\tPASS\tTOTAL\tRATE\n", dim)
		for k, s := range stats {
			if k.dim == dim {
				rate := float64(s.pass) / float64(s.total) * 100
				fmt.Fprintf(w, "  %s\t%d\t%d\t%.1f%%\n", k.val, s.pass, s.total, rate)
			}
		}
		w.Flush()
	}
}
