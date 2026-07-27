package main

// `ailang chains stats --cost-per-verified-success --baseline <id> [--json] [--strict]`
// (M-COST-PER-SUCCESS-KPI, M2).
//
// This is a THIN presentation surface. It reuses the observatory rollup
// (Store.CostPerVerifiedSuccess) — the SINGLE authoritative computation shared
// with the HTTP handler and the latest.json publisher — and never recomputes
// cost or re-derives the verified-success predicate. In --strict mode it exits
// non-zero whenever the KPI is unavailable (unknown cost, missing verification /
// zero denominator, or an empty cohort).

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// chainsStatsCostPerVerifiedSuccess implements the KPI surface.
//
// baseline is BOTH the provenance id and the chains.source_ref prefix used to
// scope the frozen cohort (e.g. "v1.0" matches "v1.0/agent/baseline"). A
// trailing '/' is appended for the prefix match if not already present, so
// "v1.0" never accidentally also matches "v1.05".
func chainsStatsCostPerVerifiedSuccess(baseline string, hours int, jsonOutput, strict bool) {
	if baseline == "" {
		fmt.Fprintln(os.Stderr, "Error: --cost-per-verified-success requires --baseline <id> (the frozen cohort source_ref prefix, e.g. v1.0)")
		os.Exit(1)
	}

	// BF-2: the baseline id becomes an UNESCAPED SQL LIKE pattern in
	// store_chains_eval.go (`c.source_ref LIKE ?`, no ESCAPE clause), so '_' and
	// '%' are wildcards that would silently WIDEN the queried cohort. Reuse the
	// SAME validator the freeze side uses (cmd/ailang/eval_suite_cohort.go) so a
	// frozen id is always queryable and a queryable id could always have been
	// frozen — one charset, both sides.
	if err := validateBaselineID(baseline); err != nil {
		fmt.Fprintf(os.Stderr, "Error: --baseline: %v\n", err)
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

	var createdAfter *time.Time
	if hours > 0 {
		t := time.Now().Add(-time.Duration(hours) * time.Hour)
		createdAfter = &t
	}

	res, err := backend.Store().CostPerVerifiedSuccess(ctx, observatory.CostPerVerifiedSuccessOptions{
		BaselineID:   baseline,
		SourceRef:    cohortSourceRefPrefix(baseline),
		CreatedAfter: createdAfter,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to compute cost-per-verified-success: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		// Serialize the canonical struct verbatim — this is the exact wire shape
		// the HTTP handler and latest.json publisher also emit (field-for-field).
		if encErr := enc.Encode(res); encErr != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to encode result: %v\n", encErr)
			os.Exit(1)
		}
		if strict && !res.Available {
			os.Exit(2)
		}
		return
	}

	printCostPerVerifiedSuccess(res)
	if strict && !res.Available {
		os.Exit(2)
	}
}

// cohortSourceRefPrefix normalizes a baseline id into a source_ref prefix.
// A trailing '/' delimits the cohort family so "v1.0" doesn't also match "v1.05".
func cohortSourceRefPrefix(baseline string) string {
	if baseline == "" {
		return ""
	}
	if baseline[len(baseline)-1] == '/' {
		return baseline
	}
	return baseline + "/"
}

func printCostPerVerifiedSuccess(res *observatory.CostPerVerifiedSuccessResult) {
	fmt.Printf("Cost per verified success — baseline %q\n", res.BaselineID)
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("  Language / mode:  %s / %s\n", res.Language, res.EvalMode)
	fmt.Printf("  Cohort source:    %s\n", res.SourceRef)
	fmt.Println()
	fmt.Printf("  Total runs:            %d\n", res.TotalRuns)
	fmt.Printf("  Passed (exec grade):   %d\n", res.PassedRuns)
	fmt.Printf("  Verified successes:    %d\n", res.VerifiedSuccesses)
	fmt.Printf("  Unverified passes:     %d\n", res.UnverifiedPasses)
	fmt.Printf("  Verification failures: %d\n", res.VerificationFailures)
	fmt.Println()
	fmt.Printf("  Reported cost:  $%.4f\n", res.ReportedCostUSD)
	fmt.Printf("  Estimated cost: $%.4f\n", res.EstimatedCostUSD)
	fmt.Printf("  Known total:    $%.4f  (reported + estimated, includes failed runs)\n", res.KnownCostUSD)
	fmt.Printf("  Quota stages:   %d ($0-by-design)\n", res.QuotaStages)
	if res.UnknownStages > 0 {
		fmt.Printf("  %s %d stages have unattributable cost.\n", yellow("Unknown:"), res.UnknownStages)
	}
	fmt.Println()
	if res.Available {
		fmt.Printf("  %s $%.4f per verified success\n", green("KPI:"), res.CostPerVerifiedSuccessUSD)
	} else {
		fmt.Printf("  %s KPI unavailable (reason: %s). Refusing to emit $0/stale value.\n",
			yellow("⚠ Incomplete:"), res.Reason)
	}
}
