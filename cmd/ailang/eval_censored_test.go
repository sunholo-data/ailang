package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_analysis"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// writeArmRow banks one agent row into dir, mimicking the real bank layout.
//
// The file NAME must not carry the row's RFC3339 timestamp: `:` is illegal in a
// Windows filename, and the loader keys on the JSON body rather than the name,
// so the timestamp belongs only inside the row. Gate 3b caught this — the test
// passed on darwin and failed on windows-latest with "The filename, directory
// name, or volume label syntax is incorrect".
func writeArmRow(t *testing.T, dir string, index int, row map[string]any) {
	t.Helper()
	agent := filepath.Join(dir, "agent")
	if err := os.MkdirAll(agent, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("%s_%03d.json", row["id"].(string), index)
	if err := os.WriteFile(filepath.Join(agent, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func baseRow(id, ts string) map[string]any {
	return map[string]any{
		"id": id, "lang": "ailang", "model": "m", "trial": 1,
		"timestamp": ts, "stdout_ok": true, "total_tokens": 100,
	}
}

// TestEvalCensoredPairsSeesQuarantinedRows pins the COMMAND's loader choice,
// not the analyzer helper.
//
// The ">20% of ON rows quarantined -> VOID" gate is defined over the banked
// set. eval_analysis.LoadArmForPairing wraps FilterValidResults and drops
// invalid rows at LOAD time, so a command loading through it can never observe
// a quarantined row and the gate is unreachable — the analyzer's own unit tests
// pass regardless, because they construct rows directly and bypass the loader.
// This test therefore exercises evalCensoredPairs, the code path the CLI runs.
//
// Killed mutation: swapping loadCensoredArm's body to
// eval_analysis.LoadArmForPairing. Under that mutant this test reads
// verdict=INCONCLUSIVE (the quarantined row is invisible) instead of VOID.
// That mutant SURVIVED the milestone as first delivered.
func TestEvalCensoredPairsSeesQuarantinedRows(t *testing.T) {
	onDir, offDir := t.TempDir(), t.TempDir()

	// 4 ON rows, 1 of them quarantined => 25% > 20% => VOID.
	// Counterbalanced order, so the section 5.3 order gate passes and the
	// treatment gate is genuinely the thing under test.
	order := []struct {
		id, arm, ts string
	}{
		{"b0", "on", "2026-07-31T10:00:00Z"}, {"b0", "off", "2026-07-31T10:01:00Z"},
		{"b1", "off", "2026-07-31T10:02:00Z"}, {"b1", "on", "2026-07-31T10:03:00Z"},
		{"b2", "on", "2026-07-31T10:04:00Z"}, {"b2", "off", "2026-07-31T10:05:00Z"},
		{"b3", "off", "2026-07-31T10:06:00Z"}, {"b3", "on", "2026-07-31T10:07:00Z"},
	}
	for i, o := range order {
		row := baseRow(o.id, o.ts)
		dir := offDir
		if o.arm == "on" {
			dir = onDir
			if o.id == "b3" {
				row["validity"] = eval_harness.Validity{
					Valid: false, Reason: "treatment_unproven",
					Detail: "fmt ON arm banked zero fmt_hook_events",
				}
			}
		}
		writeArmRow(t, dir, i, row)
	}

	// Control: the raw loader must actually surface the quarantined row, else
	// this fixture cannot discriminate and the assertion below is vacuous.
	rawOn, err := loadCensoredArm(onDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rawOn) != 4 {
		t.Fatalf("control failed: command loader returned %d ON rows, want 4", len(rawOn))
	}
	seen := 0
	for _, r := range rawOn {
		if r.Validity != nil && !r.Validity.Valid {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("control failed: command loader surfaced %d quarantined ON rows, want 1", seen)
	}

	got, err := evalCensoredPairs(onDir, offDir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Verdict != eval_analysis.CensoredVerdictVoid {
		t.Fatalf("verdict = %q, want VOID (1 of 4 ON rows quarantined = 25%% > 20%%)", got.Verdict)
	}
	// Assert the specific reason, not merely VOID: every other refusal branch
	// also produces VOID, so the enum alone does not discriminate.
	if got.VoidReason != "treatment_unproven_rate" {
		t.Fatalf("void reason = %q, want treatment_unproven_rate", got.VoidReason)
	}
}
