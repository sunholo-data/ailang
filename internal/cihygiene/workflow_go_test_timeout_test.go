package cihygiene

import (
	"os"
	"strings"
	"testing"
)

// TestGoTestTimeoutIsDerived pins the go test -timeout value on both CI test
// legs to the DERIVED, PROVISIONAL budget (416s) rather than the unexplained
// constant (300s) it replaced. The derivation is written into ci.yml's comment
// blocks (commits e5a325a20/72f9cfeca/8e3927950 named there); this test guards
// the value AND the derivation tokens so the budget cannot silently drift back
// to an unexplained constant.
//
// Three assertions:
//  1. Value (YAML parse): job `test` and job `test-windows` each have a step
//     whose Run contains `go test -timeout 416s ./...`. Asserted by
//     presence-of-match over both jobs' steps, not by step index (steps shift).
//  2. Anti-revert: NO step in either job's Run contains `go test -timeout 300s
//     ./...` — the old unexplained constant must not come back.
//  3. Comment (raw text): YAML comments are not part of the parsed Run payloads,
//     so the derivation comment can only be checked on raw bytes. The ci.yml
//     file must contain all three derivation factors: 172.5, 1.80, 1.34.
func TestGoTestTimeoutIsDerived(t *testing.T) {
	wfs := loadWorkflows(t)

	ci, ok := wfs["ci.yml"]
	if !ok {
		t.Fatal("instrument failure: ci.yml not loaded by loadWorkflows")
	}

	const want = "go test -timeout 416s ./..."
	const stale = "go test -timeout 300s ./..."

	for _, jobID := range []string{"test", "test-windows"} {
		job, ok := ci.Jobs[jobID]
		if !ok {
			t.Errorf("ci.yml: job %q not found", jobID)
			continue
		}
		var found bool
		for _, step := range job.Steps {
			if strings.Contains(step.Run, want) {
				found = true
			}
			if strings.Contains(step.Run, stale) {
				t.Errorf("ci.yml: job %q step %q reverted to the unexplained %q; "+
					"the budget must stay the derived 416s", jobID, step.Name, stale)
			}
		}
		if !found {
			t.Errorf("ci.yml: job %q has no step running %q — the derived budget is missing", jobID, want)
		}
	}

	// Comment tokens (raw bytes — YAML comments are not in the parsed Run payloads).
	raw, err := os.ReadFile("../../.github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("read ci.yml for derivation comment: %v", err)
	}
	for _, tok := range []string{"172.5", "1.80", "1.34"} {
		if !strings.Contains(string(raw), tok) {
			t.Errorf("ci.yml derivation comment is missing factor %q; the budget must be "+
				"derived (172.5s worst steady-state package x 1.80x runner variance x 1.34x safety factor)", tok)
		}
	}
}
