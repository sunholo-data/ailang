package main

// `ailang chains stats --by-mission` / `--by-source-prefix` (M-MISSION-COST-CHAINS M3).
// Groups chains by source_ref prefix (mission:<name>/iter-<N>) and reports the
// per-mission metered total vs MISSION_METERED_BUDGET_USD, per-bucket quota counts,
// and the top-N most expensive stages — reusing the M1 cost classifier. CLI only.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// missionRollupJSON is the JSON shape for `chains stats --by-mission`.
type missionRollupJSON struct {
	Mission          string             `json:"mission"`
	MeteredBudgetUSD *float64           `json:"metered_budget_usd,omitempty"` // nil => budget unset
	ReportedCost     float64            `json:"reported_cost"`
	EstimatedCost    float64            `json:"estimated_cost"`
	KnownCost        float64            `json:"known_cost"` // reported + estimated (metered total)
	OverBudget       bool               `json:"over_budget"`
	ReportedStages   int                `json:"reported_stages"`
	EstimatedStages  int                `json:"estimated_stages"`
	QuotaStages      int                `json:"quota_stages"`
	UnknownStages    int                `json:"unknown_stages"`
	QuotaByBucket    map[string]int     `json:"quota_by_bucket"`
	TopStages        []missionStageJSON `json:"top_stages"`
}

type missionStageJSON struct {
	AgentID string  `json:"agent_id"`
	Status  string  `json:"status"`
	CostUSD float64 `json:"cost_usd"`
	Tokens  int64   `json:"tokens"`
	Model   string  `json:"model,omitempty"`
}

// chainsStatsByMission implements `chains stats --by-mission` / `--by-source-prefix`.
// Per mission: metered total (reported+estimated) vs MISSION_METERED_BUDGET_USD,
// per-bucket quota counts, and top-N most expensive stages. Reuses the M1 classifier.
func chainsStatsByMission(sourcePrefix string, hours int, jsonOutput, strict bool) {
	dbPath := observatory.DefaultDatabasePath()
	backend, err := observatory.NewSQLiteBackendFromPath(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to observatory: %v\n", err)
		os.Exit(1)
	}
	defer backend.Close()

	ctx := context.Background()

	var createdAfter *time.Time
	timeWindow := "all time"
	if hours > 0 {
		t := time.Now().Add(-time.Duration(hours) * time.Hour)
		createdAfter = &t
		timeWindow = fmt.Sprintf("last %d hours", hours)
	}

	rollups, err := backend.GetMissionRollups(ctx, createdAfter, sourcePrefix, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get mission rollups: %v\n", err)
		os.Exit(1)
	}

	// Budget: NO silent fallback. If MISSION_METERED_BUDGET_USD is unset, we show
	// "budget unset" and never compare against 0.
	var budget *float64
	if raw := os.Getenv("MISSION_METERED_BUDGET_USD"); raw != "" {
		if v, perr := strconv.ParseFloat(raw, 64); perr == nil {
			budget = &v
		} else {
			fmt.Fprintf(os.Stderr, "Warning: MISSION_METERED_BUDGET_USD=%q is not a number; treating as unset\n", raw)
		}
	}

	anyIncomplete := false
	out := make([]missionRollupJSON, 0, len(rollups))
	for _, mr := range rollups {
		known := mr.Rollup.TotalKnownCost()
		over := budget != nil && known > *budget
		if mr.Rollup.HasIncompleteData() {
			anyIncomplete = true
		}
		mj := missionRollupJSON{
			Mission:          mr.Mission,
			MeteredBudgetUSD: budget,
			ReportedCost:     mr.Rollup.ReportedCost,
			EstimatedCost:    mr.Rollup.EstimatedCost,
			KnownCost:        known,
			OverBudget:       over,
			ReportedStages:   mr.Rollup.ReportedStages,
			EstimatedStages:  mr.Rollup.EstimatedStages,
			QuotaStages:      mr.Rollup.QuotaStages,
			UnknownStages:    mr.Rollup.UnknownStages,
			QuotaByBucket:    mr.QuotaByBucket,
		}
		for _, ts := range mr.TopStages {
			mj.TopStages = append(mj.TopStages, missionStageJSON{
				AgentID: ts.AgentID,
				Status:  string(ts.Status),
				CostUSD: ts.CostUSD,
				Tokens:  ts.Tokens,
				Model:   ts.Model,
			})
		}
		out = append(out, mj)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"time_window": timeWindow,
			"prefix":      sourcePrefix,
			"missions":    out,
		})
		if strict && anyIncomplete {
			os.Exit(2)
		}
		return
	}

	fmt.Printf("Mission Cost Rollup (%s, prefix %q)\n", timeWindow, sourcePrefix)
	fmt.Println("═══════════════════════════════════════════")
	if len(out) == 0 {
		fmt.Println("  (no chains matched this prefix)")
		return
	}
	for _, mj := range out {
		fmt.Println()
		fmt.Printf("%s\n", cyan(mj.Mission))
		fmt.Printf("  Metered total: $%.4f  (reported $%.4f + estimated $%.4f)\n",
			mj.KnownCost, mj.ReportedCost, mj.EstimatedCost)
		if mj.MeteredBudgetUSD == nil {
			fmt.Printf("  Budget:        %s (MISSION_METERED_BUDGET_USD)\n", yellow("unset"))
		} else if mj.OverBudget {
			fmt.Printf("  Budget:        $%.2f — %s\n", *mj.MeteredBudgetUSD, red("OVER BUDGET"))
		} else {
			fmt.Printf("  Budget:        $%.2f (within budget)\n", *mj.MeteredBudgetUSD)
		}
		if mj.UnknownStages > 0 {
			fmt.Printf("  %s %d stages unattributed (tokens but no resolvable model) — metered total may be low\n",
				yellow("⚠"), mj.UnknownStages)
		}
		if len(mj.QuotaByBucket) > 0 {
			fmt.Printf("  Quota stages by bucket:\n")
			for bucket, n := range mj.QuotaByBucket {
				fmt.Printf("    %-12s %d\n", bucket, n)
			}
		}
		if len(mj.TopStages) > 0 {
			fmt.Printf("  Top stages by cost:\n")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "    AGENT\tSTATUS\tCOST\tTOKENS")
			for _, ts := range mj.TopStages {
				fmt.Fprintf(w, "    %s\t%s\t$%.4f\t%d\n", ts.AgentID, ts.Status, ts.CostUSD, ts.Tokens)
			}
			w.Flush()
		}
	}

	if strict && anyIncomplete {
		os.Exit(2)
	}
}
