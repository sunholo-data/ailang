package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
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

	// Cost attribution split (M-MISSION-COST-CHAINS M1). reported = self-reported;
	// estimated = inferred tokens×rate for token-bearing/no-cost/metered stages;
	// quota = subscription lanes ($0-by-design); unknown = token-bearing but no
	// resolvable model (surfaced separately, NEVER faked as $0 metered spend).
	ReportedCost    float64 `json:"reported_cost"`
	EstimatedCost   float64 `json:"estimated_cost"`
	KnownCost       float64 `json:"known_cost"` // reported + estimated (the credible total)
	ReportedStages  int     `json:"reported_stages"`
	EstimatedStages int     `json:"estimated_stages"`
	QuotaStages     int     `json:"quota_stages"`
	UnknownStages   int     `json:"unknown_stages"`
	IncompleteData  bool    `json:"incomplete_data"`
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
	byMission := fs.Bool("by-mission", false, "Group by mission (source_ref prefix 'mission:'): per-mission metered total vs MISSION_METERED_BUDGET_USD, per-bucket quota counts, top-N stages")
	bySourcePrefix := fs.String("by-source-prefix", "", "Group by an arbitrary source_ref prefix (e.g. 'mission:v1/')")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	strict := fs.Bool("strict", false, "Exit non-zero if any stage has unattributable (unknown) cost")
	costPerVerifiedSuccess := fs.Bool("cost-per-verified-success", false, "Compute the frozen-cohort cost-per-verified-success KPI (M-COST-PER-SUCCESS-KPI)")
	baseline := fs.String("baseline", "", "Frozen cohort baseline id/source_ref prefix for --cost-per-verified-success (e.g. 'v1.0')")
	fs.Parse(flag.Args()[2:])

	// M-COST-PER-SUCCESS-KPI: the headline KPI is its own strict surface. It
	// reuses the observatory rollup (never recomputes cost) and emits the exact
	// same struct the HTTP handler and latest.json publisher serialize.
	if *costPerVerifiedSuccess {
		chainsStatsCostPerVerifiedSuccess(*baseline, *hours, *jsonOutput, *strict)
		return
	}

	// M3: --by-mission / --by-source-prefix delegate to the mission rollup surface.
	if *byMission || *bySourcePrefix != "" {
		prefix := *bySourcePrefix
		if *byMission && prefix == "" {
			prefix = "mission:"
		}
		chainsStatsByMission(prefix, *hours, *jsonOutput, *strict)
		return
	}

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

	// M1: Go per-stage classification pass (the SQL SUM above cannot estimate from
	// tokens×rate — estimation needs a per-stage model). Splits the total into
	// reported / estimated / quota / unknown so a token-bearing $0 stage is no
	// longer a misleading "free" signal.
	rollup, rollupErr := backend.GetCostRollup(ctx, createdAfter, "")
	if rollupErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to classify stage costs: %v\n", rollupErr)
	} else {
		result.ReportedCost = rollup.ReportedCost
		result.EstimatedCost = rollup.EstimatedCost
		result.KnownCost = rollup.TotalKnownCost()
		result.ReportedStages = rollup.ReportedStages
		result.EstimatedStages = rollup.EstimatedStages
		result.QuotaStages = rollup.QuotaStages
		result.UnknownStages = rollup.UnknownStages
		result.IncompleteData = rollup.HasIncompleteData()
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
		if *strict && result.IncompleteData {
			os.Exit(2)
		}
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

	// M1: cost-attribution split (only shown when the classifier ran).
	if rollupErr == nil {
		fmt.Println()
		fmt.Println("Cost attribution (per-stage):")
		fmt.Printf("  Reported:   $%.4f  (%d stages, self-reported)\n", result.ReportedCost, result.ReportedStages)
		fmt.Printf("  Estimated:  $%.4f  (%d stages, tokens×rate)\n", result.EstimatedCost, result.EstimatedStages)
		fmt.Printf("  Known total:$%.4f  (reported + estimated)\n", result.KnownCost)
		fmt.Printf("  Quota:      %d stages ($0-by-design, subscription lanes)\n", result.QuotaStages)
		if result.UnknownStages > 0 {
			fmt.Printf("  %s %d stages have tokens but no resolvable model — cost NOT attributed (shown as unknown, never $0).\n",
				yellow("Unknown:"), result.UnknownStages)
			fmt.Printf("  %s budget totals are INCOMPLETE (%d unattributed stages). Re-run with --strict to fail.\n",
				yellow("⚠ Warning:"), result.UnknownStages)
		}
	}

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

	if *strict && result.IncompleteData {
		os.Exit(2)
	}
}
