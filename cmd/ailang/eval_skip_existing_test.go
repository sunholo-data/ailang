package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

func writeRow(t *testing.T, dir, name string, m eval_harness.RunMetrics) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestHasValidBankedResult is the retry mechanism.
//
// --skip-existing used to glob for ANY matching file. A run that CRASHED still
// wrote a row, so the rotation saw a file, skipped the benchmark, and the hole
// was permanent. On the local rig that is most of the failures: 76% of all
// baseline failures are api_error, and their message is "motoko terminated
// without emitting run_summary" — a harness crash, not the model. Those are
// ours to retry, not the model's to be blamed for.
//
// Only a VALID row counts as banked. Invalid rows are re-attempted next
// rotation, for free.
func TestHasValidBankedResult(t *testing.T) {
	tests := []struct {
		name string
		rows []eval_harness.RunMetrics
		want bool
	}{
		{
			name: "no rows at all",
			rows: nil,
			want: false,
		},
		{
			name: "a valid row counts as banked",
			rows: []eval_harness.RunMetrics{{ID: "b", StdoutOk: true}},
			want: true,
		},
		{
			// The crash case: a row exists, but it measured nothing.
			name: "only an invalid row does NOT count — retry it",
			rows: []eval_harness.RunMetrics{
				{ID: "b", Validity: eval_harness.MarkInvalid(eval_harness.ReasonHarnessError)},
			},
			want: false,
		},
		{
			// A genuine model failure IS a measurement and must not be retried
			// forever — that would turn a hard benchmark into an infinite loop.
			name: "a failing but VALID row counts as banked",
			rows: []eval_harness.RunMetrics{{ID: "b", StdoutOk: false}},
			want: true,
		},
		{
			name: "one valid among invalids counts",
			rows: []eval_harness.RunMetrics{
				{ID: "b", Validity: eval_harness.MarkInvalid(eval_harness.ReasonCanaryFailed)},
				{ID: "b", StdoutOk: true},
			},
			want: true,
		},
		{
			name: "all invalid — retry",
			rows: []eval_harness.RunMetrics{
				{ID: "b", Validity: eval_harness.MarkInvalid(eval_harness.ReasonCanaryFailed)},
				{ID: "b", Validity: eval_harness.MarkInvalid(eval_harness.ReasonTreatmentUnproven)},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			var patterns []string
			for i, r := range tt.rows {
				writeRow(t, dir, filepathName(i), r)
			}
			patterns = append(patterns, filepath.Join(dir, "*.json"))

			if got := hasValidBankedResult(patterns); got != tt.want {
				t.Errorf("hasValidBankedResult = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHasValidBankedResult_UnparseableCountsAsBanked: a corrupt file is not a
// reason to re-run a benchmark forever. Absent validity already means valid, so
// the conservative reading of an unreadable row is "leave it alone".
func TestHasValidBankedResult_UnparseableCountsAsBanked(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "junk.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !hasValidBankedResult([]string{filepath.Join(dir, "*.json")}) {
		t.Error("an unparseable row should count as banked, not trigger an endless retry")
	}
}

func filepathName(i int) string {
	return string(rune('a'+i)) + ".json"
}
