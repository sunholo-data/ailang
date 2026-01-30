package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

func chainsCommand() {
	if flag.NArg() < 2 {
		// No subcommand - show interactive mode if terminal, else show help
		if isTerminal() {
			runChainsInteractive()
			return
		}
		fmt.Println("Usage: ailang chains <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  list      List execution chains")
		fmt.Println("  view      View a chain with all stages")
		fmt.Println("  tree      ASCII tree view of chain hierarchy")
		fmt.Println("  diagnose  Quick health report for a specific chain")
		fmt.Println("  health    System-wide data capture validation")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang chains list                  # List all chains")
		fmt.Println("  ailang chains list --status active  # Filter by status")
		fmt.Println("  ailang chains view <chain-id>       # View chain details")
		fmt.Println("  ailang chains tree <chain-id>       # View as tree")
		fmt.Println("  ailang chains diagnose <chain-id>   # Quick issue check")
		fmt.Println("  ailang chains health                # System-wide validation")
		fmt.Println()
		fmt.Println("Run 'ailang chains' in a terminal for interactive mode.")
		os.Exit(1)
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "list":
		chainsListCommand()
	case "view":
		chainsViewCommand()
	case "tree":
		chainsTreeCommand()
	case "diagnose":
		chainsDiagnoseCommand()
	case "health":
		chainsHealthCommand()
	default:
		fmt.Printf("Unknown subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func chainsListCommand() {
	fs := flag.NewFlagSet("chains list", flag.ExitOnError)
	status := fs.String("status", "", "Filter by status (active, pending_approval, completed, failed)")
	sourceType := fs.String("source", "", "Filter by source type (github_issue, message, manual)")
	limit := fs.Int("limit", 20, "Maximum number of chains to show")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fullIDs := fs.Bool("full", false, "Show full chain IDs (for copy-paste)")
	fs.Parse(flag.Args()[2:])

	// Connect to observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()
	opts := observatory.ChainListOptions{
		Limit: *limit,
	}
	if *status != "" {
		opts.Status = observatory.ChainStatus(*status)
	}
	if *sourceType != "" {
		opts.SourceType = *sourceType
	}

	chains, err := backend.ListChains(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list chains: %v\n", err)
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(chains)
		return
	}

	if len(chains) == 0 {
		fmt.Println("No execution chains found.")
		return
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tSOURCE\tSTAGES\tCOST\tCREATED")
	for _, chain := range chains {
		source := chain.SourceType
		if chain.SourceRef != "" {
			source = fmt.Sprintf("%s:%s", chain.SourceType, chain.SourceRef)
		}
		cost := fmt.Sprintf("$%.2f", chain.TotalCost)
		created := chain.CreatedAt.Format("2006-01-02 15:04")

		// Show full or truncated ID based on flag
		chainID := truncateChainID(chain.ID)
		if *fullIDs {
			chainID = chain.ID
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			chainID,
			chain.Status,
			source,
			chain.StagesCompleted,
			cost,
			created,
		)
	}
	w.Flush()
}

func chainsViewCommand() {
	fs := flag.NewFlagSet("chains view", flag.ExitOnError)
	includeSpans := fs.Bool("spans", false, "Include spans for each stage")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	if fs.NArg() < 1 {
		fmt.Println("Usage: ailang chains view <chain-id>")
		os.Exit(1)
	}

	chainIDPrefix := fs.Arg(0)

	// Connect to observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	// Resolve short ID prefix to full ID
	chainID, err := resolveChainID(backend, ctx, chainIDPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	opts := observatory.ChainReadOptions{
		IncludeStages: true,
		IncludeSpans:  *includeSpans,
	}

	chain, err := backend.GetChain(ctx, chainID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get chain: %v\n", err)
		os.Exit(1)
	}
	if chain == nil {
		fmt.Fprintf(os.Stderr, "Error: chain not found: %s\n", chainID)
		os.Exit(1)
	}

	// Get stages
	stages, err := backend.GetChainStages(ctx, chainID, opts)
	if err == nil {
		chain.Stages = stages
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(chain)
		return
	}

	// Print chain details
	fmt.Printf("Chain: %s\n", chain.ID)
	fmt.Printf("Status: %s\n", colorizeStatus(string(chain.Status)))
	fmt.Printf("Source: %s", chain.SourceType)
	if chain.SourceRef != "" {
		fmt.Printf(" (%s)", chain.SourceRef)
	}
	fmt.Println()
	if chain.GitHubRepo != "" {
		fmt.Printf("GitHub: %s#%d\n", chain.GitHubRepo, chain.GitHubIssueNumber)
	}
	fmt.Printf("Created: %s\n", chain.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Total Cost: $%.4f\n", chain.TotalCost)
	fmt.Printf("Total Tokens: %d\n", chain.TotalTokens)
	fmt.Println()

	// Print stages
	if len(chain.Stages) > 0 {
		fmt.Println("Stages:")
		for i, stage := range chain.Stages {
			fmt.Printf("  %d. %s [%s]\n", i+1, stage.AgentID, colorizeStatus(string(stage.Status)))
			if stage.TaskID != "" {
				fmt.Printf("     Task: %s\n", stage.TaskID)
			}
			if stage.SessionID != "" {
				fmt.Printf("     Session: %s\n", truncateChainID(stage.SessionID))
			}
			if stage.ApprovalStatus != "" {
				fmt.Printf("     Approval: %s\n", stage.ApprovalStatus)
			}
			if stage.Cost > 0 {
				fmt.Printf("     Cost: $%.4f (%d tokens in, %d tokens out)\n",
					stage.Cost, stage.TokensIn, stage.TokensOut)
			}
			if stage.HandoffTo != "" {
				fmt.Printf("     Handoff: -> %s\n", stage.HandoffTo)
			}
		}
	}
}

// Formatting helper functions

func formatChainDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	} else if ms < 60000 {
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	} else {
		mins := ms / 60000
		secs := (ms % 60000) / 1000
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
}

func truncateChainID(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12] + "..."
}

func colorizeStatus(status string) string {
	switch status {
	case "active", "running":
		return cyan(status)
	case "completed":
		return green(status)
	case "failed":
		return red(status)
	case "pending", "pending_approval", "awaiting_approval":
		return yellow(status)
	default:
		return status
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "completed":
		return green("✓")
	case "running", "active":
		return cyan("●")
	case "pending", "pending_approval", "awaiting_approval":
		return yellow("○")
	case "failed":
		return red("✗")
	default:
		return "○"
	}
}

func formatDurationHuman(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		if m > 0 {
			return fmt.Sprintf("%dh %dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}
