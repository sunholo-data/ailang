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

	// Compute time cutoff
	var createdAfter *time.Time
	timeWindow := "all time"
	if *hours > 0 {
		t := time.Now().Add(-time.Duration(*hours) * time.Hour)
		createdAfter = &t
		timeWindow = fmt.Sprintf("last %d hours", *hours)
	}

	// Single SQL query for chain counts by status (replaces fetch-all + Go loop, M-PERF-OBSERVATORY)
	counts, err := backend.GetChainStatusCounts(ctx, createdAfter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get chain stats: %v\n", err)
		os.Exit(1)
	}

	result := chainStatsResult{
		TimeWindow:  timeWindow,
		TotalChains: counts.Total,
		Completed:   counts.Completed,
		Active:      counts.Active,
		Pending:     counts.Pending,
		Failed:      counts.Failed,
		TotalCost:   counts.TotalCost,
		TotalTokens: counts.TotalTokens,
	}
	if result.TotalChains > 0 {
		result.AvgCostPerChain = result.TotalCost / float64(result.TotalChains)
	}

	// Single SQL query for per-agent stats (replaces N+1, M-PERF-OBSERVATORY)
	if *byAgent {
		agentResults, err := backend.GetChainStatsByAgent(ctx, createdAfter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get agent stats: %v\n", err)
		} else {
			for _, ar := range agentResults {
				result.ByAgent = append(result.ByAgent, agentStats{
					AgentID:        ar.AgentID,
					Stages:         ar.Stages,
					Completed:      ar.Completed,
					Failed:         ar.Failed,
					TotalCost:      ar.TotalCost,
					TotalTokensIn:  ar.TokensIn,
					TotalTokensOut: ar.TokensOut,
				})
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
