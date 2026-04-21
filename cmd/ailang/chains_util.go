package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// runChainsInteractive shows an interactive menu for viewing execution chains.
func runChainsInteractive() {
	// Connect to observatory database
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to connect to observatory: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)
	selectedIdx := -1

	for {
		// Get chains
		chains, err := backend.ListChains(ctx, observatory.ChainListOptions{
			Limit: 20,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			fmt.Println("Press Enter to retry...")
			reader.ReadString('\n')
			continue
		}

		// Count by status
		activeCount := 0
		pendingCount := 0
		for _, c := range chains {
			if c.Status == observatory.ChainStatusActive {
				activeCount++
			} else if c.Status == observatory.ChainStatusPendingApproval {
				pendingCount++
			}
		}

		// Clear screen and print header
		fmt.Print("\033[H\033[2J") // ANSI clear screen
		fmt.Println("┌────────────────────────────────────────────────────────────────────────────────┐")
		fmt.Printf("│ AILANG Execution Chains (%d active, %d pending)                                 │\n", activeCount, pendingCount)
		fmt.Println("├────────────────────────────────────────────────────────────────────────────────┤")

		if len(chains) == 0 {
			fmt.Println("│ No execution chains found.                                                     │")
		} else {
			// Header row
			fmt.Println("│  # │ STATUS           │ SOURCE                   │ STAGES │  COST  │ AGE     │")
			fmt.Println("├────┼──────────────────┼──────────────────────────┼────────┼────────┼─────────┤")

			for i, chain := range chains {
				// Status with color
				statusStr := formatChainStatus(chain.Status)

				// Source info
				source := string(chain.SourceType)
				if chain.GitHubRepo != "" && chain.GitHubIssueNumber > 0 {
					// Show shortened repo name + issue number
					repoParts := strings.Split(chain.GitHubRepo, "/")
					repoName := chain.GitHubRepo
					if len(repoParts) > 1 {
						repoName = repoParts[len(repoParts)-1]
					}
					source = fmt.Sprintf("%s#%d", repoName, chain.GitHubIssueNumber)
				} else if chain.SourceRef != "" {
					source = fmt.Sprintf("%s:%s", chain.SourceType, truncateString(chain.SourceRef, 10))
				}
				if len(source) > 24 {
					source = source[:21] + "..."
				}

				// Cost formatting
				cost := fmt.Sprintf("$%.2f", chain.TotalCost)

				// Age
				age := formatAge(chain.CreatedAt)

				// Selection indicator (use same visible width for both)
				prefix := "  "
				if i == selectedIdx {
					prefix = cyan(">") + " "
				}

				fmt.Printf("│%s%d │ %s │ %-24s │ %6d │ %6s │ %-7s │\n",
					prefix, i+1,
					statusStr,
					source,
					chain.StagesCompleted,
					cost,
					age,
				)
			}
		}

		fmt.Println("└────────────────────────────────────────────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("Actions: [1-9] select  [v]iew  [t]ree  [d]etailed  [j]son  [i]d copy  [r]efresh  [q]uit")
		fmt.Println()

		if selectedIdx >= 0 && selectedIdx < len(chains) {
			chain := chains[selectedIdx]
			fmt.Printf("Selected: %s\n", cyan(chain.ID))
			fmt.Printf("  Status: %s  |  Created: %s\n",
				formatChainStatus(chain.Status),
				chain.CreatedAt.Format("2006-01-02 15:04:05"))
		}

		fmt.Print("\nEnter command: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("\nExiting...")
			return
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		switch input {
		case "q", "quit", "exit":
			fmt.Println("Goodbye!")
			return

		case "r", "refresh":
			// Just continue to refresh
			continue

		case "v", "view":
			if selectedIdx < 0 || selectedIdx >= len(chains) {
				fmt.Println(red("No chain selected. Press 1-9 to select."))
				waitForEnter(reader)
				continue
			}
			chain := chains[selectedIdx]
			displayChainDetails(backend, ctx, chain.ID)
			waitForEnter(reader)

		case "t", "tree":
			if selectedIdx < 0 || selectedIdx >= len(chains) {
				fmt.Println(red("No chain selected. Press 1-9 to select."))
				waitForEnter(reader)
				continue
			}
			chain := chains[selectedIdx]
			displayChainTree(backend, ctx, chain.ID)
			waitForEnter(reader)

		case "d", "detailed":
			if selectedIdx < 0 || selectedIdx >= len(chains) {
				fmt.Println(red("No chain selected. Press 1-9 to select."))
				waitForEnter(reader)
				continue
			}
			chain := chains[selectedIdx]
			displayChainTreeDetailed(backend, ctx, chain.ID)
			waitForEnter(reader)

		case "j", "json":
			if selectedIdx < 0 || selectedIdx >= len(chains) {
				fmt.Println(red("No chain selected. Press 1-9 to select."))
				waitForEnter(reader)
				continue
			}
			chain := chains[selectedIdx]
			displayChainJSON(backend, ctx, chain.ID)
			waitForEnter(reader)

		case "i", "id":
			if selectedIdx < 0 || selectedIdx >= len(chains) {
				fmt.Println(red("No chain selected. Press 1-9 to select."))
				waitForEnter(reader)
				continue
			}
			chain := chains[selectedIdx]
			fmt.Printf("\n%s Full Chain ID (copy this):\n", green(""))
			fmt.Printf("\n  %s\n", bold(chain.ID))
			fmt.Printf("\nCommands:\n")
			fmt.Printf("  ailang chains view %s\n", chain.ID)
			fmt.Printf("  ailang chains tree %s\n", chain.ID)
			waitForEnter(reader)

		default:
			// Try to parse as number for selection
			if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(chains) {
				selectedIdx = num - 1
			} else {
				fmt.Printf("%s Unknown command: %s\n", yellow("?"), input)
				waitForEnter(reader)
			}
		}
	}
}

// formatChainStatus returns a colored status string with consistent width.
// The width parameter ensures alignment in tables (pad before coloring).
func formatChainStatus(status observatory.ChainStatus) string {
	s := string(status)
	// Pad to consistent width BEFORE adding color codes
	padded := fmt.Sprintf("%-16s", s)
	switch status {
	case observatory.ChainStatusActive:
		return cyan(padded)
	case observatory.ChainStatusCompleted:
		return green(padded)
	case observatory.ChainStatusFailed:
		return red(padded)
	case observatory.ChainStatusPendingApproval:
		return yellow(fmt.Sprintf("%-16s", "pending_appr"))
	default:
		return padded
	}
}

// displayChainDetails shows detailed information about a chain.
func displayChainDetails(backend *observatory.SQLiteBackend, ctx context.Context, chainID string) {
	opts := observatory.ChainReadOptions{
		IncludeStages: true,
	}

	chain, err := backend.GetChain(ctx, chainID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to get chain: %v\n", red("Error"), err)
		return
	}
	if chain == nil {
		fmt.Fprintf(os.Stderr, "%s: chain not found\n", red("Error"))
		return
	}

	stages, err := backend.GetChainStages(ctx, chainID, opts)
	if err == nil {
		chain.Stages = stages
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("Chain: %s\n", bold(chain.ID))
	fmt.Println("═══════════════════════════════════════════════════════════════════════════════")
	fmt.Printf("Status:      %s\n", colorizeStatus(string(chain.Status)))
	fmt.Printf("Source:      %s", chain.SourceType)
	if chain.SourceRef != "" {
		fmt.Printf(" (%s)", chain.SourceRef)
	}
	fmt.Println()
	if chain.GitHubRepo != "" {
		fmt.Printf("GitHub:      %s#%d\n", chain.GitHubRepo, chain.GitHubIssueNumber)
	}
	fmt.Printf("Created:     %s\n", chain.CreatedAt.Format(time.RFC3339))
	if chain.CompletedAt != nil {
		fmt.Printf("Completed:   %s\n", chain.CompletedAt.Format(time.RFC3339))
	}
	fmt.Println("───────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("Total Cost:   $%.4f\n", chain.TotalCost)
	fmt.Printf("Total Tokens: %d\n", chain.TotalTokens)
	fmt.Printf("Total Turns:  %d\n", chain.TotalTurns)
	fmt.Println("───────────────────────────────────────────────────────────────────────────────")

	if len(chain.Stages) > 0 {
		fmt.Println("\nStages:")
		for i, stage := range chain.Stages {
			stageStatus := colorizeStatus(string(stage.Status))
			fmt.Printf("\n  %d. %s [%s]\n", i+1, bold(stage.AgentID), stageStatus)
			if stage.TaskID != "" {
				fmt.Printf("     Task:     %s\n", stage.TaskID)
			}
			if stage.SessionID != "" {
				fmt.Printf("     Session:  %s\n", stage.SessionID)
			}
			if stage.ApprovalStatus != "" {
				approvalStr := string(stage.ApprovalStatus)
				if stage.ApprovalStatus == "approved" {
					approvalStr = green(approvalStr)
				} else if stage.ApprovalStatus == "pending" {
					approvalStr = yellow(approvalStr)
				} else if stage.ApprovalStatus == "rejected" {
					approvalStr = red(approvalStr)
				}
				fmt.Printf("     Approval: %s\n", approvalStr)
			}
			if stage.Cost > 0 {
				fmt.Printf("     Cost:     $%.4f (%d in / %d out tokens)\n",
					stage.Cost, stage.TokensIn, stage.TokensOut)
			}
			if stage.Turns > 0 {
				fmt.Printf("     Turns:    %d\n", stage.Turns)
			}
			if stage.HandoffTo != "" {
				fmt.Printf("     Handoff:  → %s\n", cyan(stage.HandoffTo))
			}
		}
	}
	fmt.Println()
}

// resolveChainID resolves a short ID prefix to a full chain ID.
// Returns error if no match or multiple matches (ambiguous prefix).
func resolveChainID(backend *observatory.SQLiteBackend, ctx context.Context, prefix string) (string, error) {
	// If prefix looks like a full UUID, use it directly
	if len(prefix) >= 36 {
		return prefix, nil
	}

	// Query chains to find matching prefix
	chains, err := backend.ListChains(ctx, observatory.ChainListOptions{
		Limit: 100,
	})
	if err != nil {
		return "", err
	}

	var matches []string
	for _, chain := range chains {
		if strings.HasPrefix(chain.ID, prefix) {
			matches = append(matches, chain.ID)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no chain found with prefix '%s'", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous prefix '%s' matches %d chains, use a longer prefix", prefix, len(matches))
	}
}

// displayChainTree shows a tree view of the chain hierarchy.
func displayChainTree(backend *observatory.SQLiteBackend, ctx context.Context, chainID string) {
	opts := observatory.ChainReadOptions{
		IncludeStages: true,
	}

	chain, err := backend.GetChain(ctx, chainID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to get chain: %v\n", red("Error"), err)
		return
	}
	if chain == nil {
		fmt.Fprintf(os.Stderr, "%s: chain not found\n", red("Error"))
		return
	}

	stages, err := backend.GetChainStages(ctx, chainID, opts)
	if err == nil {
		chain.Stages = stages
	}

	fmt.Println()
	printChainTree(chain)
	fmt.Println()
}

// displayChainTreeDetailed shows a detailed tree view with execution info (turns, tools, session).
func displayChainTreeDetailed(backend *observatory.SQLiteBackend, ctx context.Context, chainID string) {
	opts := observatory.ChainReadOptions{
		IncludeStages: true,
	}

	chain, err := backend.GetChain(ctx, chainID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to get chain: %v\n", red("Error"), err)
		return
	}
	if chain == nil {
		fmt.Fprintf(os.Stderr, "%s: chain not found\n", red("Error"))
		return
	}

	stages, err := backend.GetChainStages(ctx, chainID, opts)
	if err == nil {
		chain.Stages = stages
	}

	fmt.Println()
	printChainTreeDetailed(ctx, backend, chain, true) // detailed=true
	fmt.Println()
}

// displayChainJSON outputs full chain data as JSON for debugging/data export.
func displayChainJSON(backend *observatory.SQLiteBackend, ctx context.Context, chainID string) {
	opts := observatory.ChainReadOptions{
		IncludeStages: true,
	}

	chain, err := backend.GetChain(ctx, chainID, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to get chain: %v\n", red("Error"), err)
		return
	}
	if chain == nil {
		fmt.Fprintf(os.Stderr, "%s: chain not found\n", red("Error"))
		return
	}

	stages, err := backend.GetChainStages(ctx, chainID, opts)
	if err == nil {
		chain.Stages = stages
	}

	fmt.Println()
	printChainJSON(chain)
}
