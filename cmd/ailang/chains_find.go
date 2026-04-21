package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

func chainsFindCommand() {
	fs := flag.NewFlagSet("chains find", flag.ExitOnError)
	messageID := fs.String("message-id", "", "Find chain by source message ID")
	taskID := fs.String("task-id", "", "Find chain by coordinator task ID")
	github := fs.String("github", "", "Find chain by GitHub issue (repo#number)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	// Validate exactly one lookup key
	keyCount := 0
	if *messageID != "" {
		keyCount++
	}
	if *taskID != "" {
		keyCount++
	}
	if *github != "" {
		keyCount++
	}
	if keyCount != 1 {
		fmt.Println("Usage: ailang chains find <lookup-key>")
		fmt.Println()
		fmt.Println("Lookup keys (exactly one required):")
		fmt.Println("  --message-id <uuid>           Find by collaboration message ID")
		fmt.Println("  --task-id <task-id>           Find by coordinator task ID")
		fmt.Println("  --github <repo>#<number>      Find by GitHub issue")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --json                        Output as JSON")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang chains find --task-id task-29404032")
		fmt.Println("  ailang chains find --github sunholo-data/ailang#42")
		fmt.Println("  ailang chains find --message-id 29404032-74b3-40c6-acc3-23d6bbe14b68 --json")
		os.Exit(1)
	}

	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()
	var chain *observatory.ExecutionChain

	switch {
	case *messageID != "":
		chain, err = backend.GetChainByMessageID(ctx, *messageID)
	case *taskID != "":
		chain, err = backend.GetChainByTaskID(ctx, *taskID)
	case *github != "":
		repo, number, parseErr := parseGitHubRef(*github)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", parseErr)
			fmt.Fprintf(os.Stderr, "Expected format: owner/repo#number (e.g., sunholo-data/ailang#42)\n")
			os.Exit(1)
		}
		chain, err = backend.GetChainByGitHubIssue(ctx, repo, number)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If no chain found, try span summary fallback (for user sessions, evals, etc.)
	if chain == nil && *taskID != "" {
		summary, summaryErr := backend.GetTaskSpanSummary(ctx, *taskID)
		if summaryErr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", summaryErr)
			os.Exit(1)
		}
		if summary == nil {
			fmt.Fprintln(os.Stderr, "No chain or spans found for this task ID.")
			os.Exit(1)
		}
		printTaskSpanSummary(summary, *jsonOutput)
		return
	}

	if chain == nil {
		fmt.Fprintln(os.Stderr, "No chain found.")
		os.Exit(1)
	}

	// Populate stages
	stages, stageErr := backend.GetChainStages(ctx, chain.ID, observatory.ChainReadOptions{IncludeStages: true})
	if stageErr == nil {
		chain.Stages = stages
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(chain)
		return
	}

	// Reuse the same display format as chains view
	fmt.Printf("Chain: %s\n", chain.ID)
	fmt.Printf("Status: %s\n", colorizeStatus(string(chain.Status)))
	fmt.Printf("Source: %s", chain.SourceType)
	if chain.SourceRef != "" {
		fmt.Printf(" (%s)", truncateChainID(chain.SourceRef))
	}
	fmt.Println()
	if chain.GitHubRepo != "" {
		fmt.Printf("GitHub: %s#%d\n", chain.GitHubRepo, chain.GitHubIssueNumber)
	}
	fmt.Printf("Created: %s\n", chain.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Cost: $%.4f  Tokens: %d\n", chain.TotalCost, chain.TotalTokens)
	fmt.Printf("Stages: %d completed\n", chain.StagesCompleted)

	if len(chain.Stages) > 0 {
		fmt.Println()
		for i, stage := range chain.Stages {
			fmt.Printf("  %d. %s [%s]", i+1, stage.AgentID, colorizeStatus(string(stage.Status)))
			if stage.Cost > 0 {
				fmt.Printf(" $%.4f", stage.Cost)
			}
			fmt.Println()
		}
	}
}

// printTaskSpanSummary displays span statistics when no chain exists.
// This matches what the dashboard shows for "virtual chains".
func printTaskSpanSummary(s *observatory.TaskSpanSummary, jsonOutput bool) {
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(s)
		return
	}

	duration := s.EndTime.Sub(s.StartTime)
	fmt.Printf("Session: %s\n", s.TaskID)
	fmt.Printf("Type: user_session (no execution chain)\n")
	fmt.Printf("Started: %s\n", s.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %s\n", formatSessionDuration(duration))
	fmt.Printf("Cost: $%.4f  Tokens: %d in / %d out\n", s.CostUSD, s.TokensIn, s.TokensOut)
	fmt.Printf("Spans: %d across %d traces\n", s.SpanCount, s.TraceCount)

	if len(s.TopSpanNames) > 0 {
		fmt.Println()
		fmt.Println("Top operations:")
		for _, nc := range s.TopSpanNames {
			fmt.Printf("  %-40s %d\n", nc.Name, nc.Count)
		}
	}

	fmt.Println()
	fmt.Printf("Dashboard: http://127.0.0.1:1957  (select this event to see full detail)\n")
}

// formatSessionDuration formats a time.Duration in a human-readable way.
func formatSessionDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm %.0fs", d.Minutes(), d.Seconds()-d.Minutes()*60)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) - hours*60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// parseGitHubRef parses "owner/repo#number" into (repo, number, error).
func parseGitHubRef(ref string) (string, int, error) {
	parts := strings.SplitN(ref, "#", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid GitHub reference: %q (expected owner/repo#number)", ref)
	}
	repo := parts[0]
	if repo == "" || !strings.Contains(repo, "/") {
		return "", 0, fmt.Errorf("invalid repo: %q (expected owner/repo)", repo)
	}
	number, err := strconv.Atoi(parts[1])
	if err != nil || number <= 0 {
		return "", 0, fmt.Errorf("invalid issue number: %q", parts[1])
	}
	return repo, number, nil
}
