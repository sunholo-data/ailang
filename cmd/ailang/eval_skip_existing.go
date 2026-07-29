package main

import (
	"encoding/json"
	"os"
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
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			data, err := os.ReadFile(m)
			if err != nil {
				// Unreadable is not a reason to re-run forever.
				return true
			}
			var row eval_harness.RunMetrics
			if err := json.Unmarshal(data, &row); err != nil {
				// Corrupt row: conservative reading is "leave it alone".
				return true
			}
			if row.IsValid() {
				return true
			}
		}
	}
	return false
}
