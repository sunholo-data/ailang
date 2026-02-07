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

// chainStatsResult holds the aggregated stats
type chainStatsResult struct {
	TimeWindow      string       `json:"time_window"`
	TotalChains     int          `json:"total_chains"`
	Completed       int          `json:"completed"`
	Active          int          `json:"active"`
	Pending         int          `json:"pending_approval"`
	Failed          int          `json:"failed"`
	TotalCost       float64      `json:"total_cost"`
	TotalTokens     int64        `json:"total_tokens"`
	AvgCostPerChain float64      `json:"avg_cost_per_chain"`
	ByAgent         []agentStats `json:"by_agent,omitempty"`
}

type agentStats struct {
	AgentID        string  `json:"agent_id"`
	Stages         int     `json:"stages"`
	Completed      int     `json:"completed"`
	Failed         int     `json:"failed"`
	TotalCost      float64 `json:"total_cost"`
	TotalTokensIn  int     `json:"total_tokens_in"`
	TotalTokensOut int     `json:"total_tokens_out"`
}

func chainsStatsCommand() {
	fs := flag.NewFlagSet("chains stats", flag.ExitOnError)
	hours := fs.Int("hours", 0, "Time window in hours (0 = all time)")
	byAgent := fs.Bool("by-agent", false, "Show breakdown by agent")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	fs.Parse(flag.Args()[2:])

	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	// Get all chains (with high limit)
	chains, err := backend.ListChains(ctx, observatory.ChainListOptions{Limit: 1000})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to list chains: %v\n", err)
		os.Exit(1)
	}

	// Filter by time window
	var cutoff time.Time
	timeWindow := "all time"
	if *hours > 0 {
		cutoff = time.Now().Add(-time.Duration(*hours) * time.Hour)
		timeWindow = fmt.Sprintf("last %d hours", *hours)
	}

	result := chainStatsResult{TimeWindow: timeWindow}
	agentMap := make(map[string]*agentStats)

	for _, chain := range chains {
		if !cutoff.IsZero() && chain.CreatedAt.Before(cutoff) {
			continue
		}

		result.TotalChains++
		result.TotalCost += chain.TotalCost
		result.TotalTokens += int64(chain.TotalTokens)

		switch chain.Status {
		case observatory.ChainStatusCompleted:
			result.Completed++
		case observatory.ChainStatusActive:
			result.Active++
		case observatory.ChainStatusPendingApproval:
			result.Pending++
		case observatory.ChainStatusFailed:
			result.Failed++
		}

		// Aggregate per-agent stats if requested
		if *byAgent {
			stages, err := backend.GetChainStages(ctx, chain.ID, observatory.ChainReadOptions{})
			if err != nil {
				continue
			}
			for _, stage := range stages {
				as, ok := agentMap[stage.AgentID]
				if !ok {
					as = &agentStats{AgentID: stage.AgentID}
					agentMap[stage.AgentID] = as
				}
				as.Stages++
				as.TotalCost += stage.Cost
				as.TotalTokensIn += stage.TokensIn
				as.TotalTokensOut += stage.TokensOut
				switch stage.Status {
				case observatory.StageStatusCompleted:
					as.Completed++
				case observatory.StageStatusFailed:
					as.Failed++
				}
			}
		}
	}

	if result.TotalChains > 0 {
		result.AvgCostPerChain = result.TotalCost / float64(result.TotalChains)
	}

	// Convert agent map to sorted slice
	if *byAgent {
		for _, as := range agentMap {
			result.ByAgent = append(result.ByAgent, *as)
		}
		// Sort by cost descending
		for i := 0; i < len(result.ByAgent); i++ {
			for j := i + 1; j < len(result.ByAgent); j++ {
				if result.ByAgent[j].TotalCost > result.ByAgent[i].TotalCost {
					result.ByAgent[i], result.ByAgent[j] = result.ByAgent[j], result.ByAgent[i]
				}
			}
		}
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
		return
	}

	// Print formatted output
	fmt.Printf("Chain Stats (%s)\n", result.TimeWindow)
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	fmt.Printf("  Chains:     %d total", result.TotalChains)
	if result.Completed > 0 {
		fmt.Printf(" (%s completed", green(fmt.Sprintf("%d", result.Completed)))
	}
	if result.Active > 0 {
		fmt.Printf(", %s active", cyan(fmt.Sprintf("%d", result.Active)))
	}
	if result.Pending > 0 {
		fmt.Printf(", %s pending", yellow(fmt.Sprintf("%d", result.Pending)))
	}
	if result.Failed > 0 {
		fmt.Printf(", %s failed", red(fmt.Sprintf("%d", result.Failed)))
	}
	if result.TotalChains > 0 {
		fmt.Print(")")
	}
	fmt.Println()
	fmt.Printf("  Total Cost: $%.4f\n", result.TotalCost)
	fmt.Printf("  Avg/Chain:  $%.4f\n", result.AvgCostPerChain)
	fmt.Printf("  Tokens:     %d\n", result.TotalTokens)

	if *byAgent && len(result.ByAgent) > 0 {
		fmt.Println()
		fmt.Println("By Agent:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  AGENT\tSTAGES\tDONE\tFAILED\tCOST\tTOKENS")
		for _, as := range result.ByAgent {
			fmt.Fprintf(w, "  %s\t%d\t%d\t%d\t$%.4f\t%d in / %d out\n",
				as.AgentID, as.Stages, as.Completed, as.Failed,
				as.TotalCost, as.TotalTokensIn, as.TotalTokensOut)
		}
		w.Flush()
	}
}
