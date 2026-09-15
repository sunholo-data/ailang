package main

import (
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// hasValidBankedResult reports whether any file matching patterns is a real
// MEASUREMENT, and so whether --skip-existing should skip this job.
//
// # WHY THE VALIDITY CHECK MATTERS HERE
//
// --skip-existing used to skip on the mere EXISTENCE of a matching file. A run
// that crashed still wrote a row, so the rotation saw a file, skipped the
// benchmark, and the gap became permanent — the harness's own failure recorded
// as the model's, and never re-attempted.
//
// On the local rig that is the dominant case: 76% of all baseline failures are
// api_error (118 of 155; 94% of failures since 2026-07-20), and their message
// is "motoko terminated without emitting run_summary (likely crash)" — the same
// startup-crash shape as the six-day outage. Those are ours to retry, not the
// model's to be blamed for.
//
// A FAILING but valid row still counts as banked. That distinction is the whole
// point: a benchmark the model genuinely cannot pass must not be retried
// forever, or a hard benchmark becomes an infinite loop. Only rows that
// measured nothing are re-attempted.
func hasValidBankedResult(patterns []string) bool {
	var files []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		files = append(files, matches...)
	}
	if len(files) == 0 {
		return false
	}
	// The one row loader (LoadRows/LoadRowFiles). Validity filter ON: rows
	// that survive are measurements. KeepDuplicates because the question is
	// "does ANY file in this slot hold a measurement" — the per-slot
	// newest-wins collapse would let a crashed re-run hide an earlier valid
	// row. Unreadable or corrupt files land in ParseErrors, and the
	// conservative reading of those is "leave it alone" — a slot that cannot
	// be decoded must not be re-run forever.
	rows, stats, err := eval_harness.LoadRowFiles(files, eval_harness.LoadOptions{KeepDuplicates: true})
	if err != nil || len(stats.ParseErrors) > 0 {
		return true
	}
	return len(rows) > 0
}
