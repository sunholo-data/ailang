package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// M-VERIFY-SOUNDNESS-COVERAGE M6: an inconclusive bounded model is replayed
// through the real function. A runtime ensures failure (with requires holding)
// becomes a counterexample carrying the replayed inputs; a correct function
// stays inconclusive — replay never promotes what the runtime does not confirm.
func TestVerify_WitnessReplay(t *testing.T) {
	requireZ3(t)
	bin := buildAilang(t)
	stdout, stderr, code := runAilangBin(t, bin, "verify", "--json", "--verify-recursive-depth", "2", "cmd/ailang/testdata/verify_replay/replay.ail")
	var c boundedCounters
	if err := json.Unmarshal([]byte(stdout), &c); err != nil {
		t.Fatalf("verify --json: invalid JSON (exit %d): %v\n%s\n%s", code, err, stdout, stderr)
	}
	rows := c.byFunc()
	want := map[string]string{"sumBad": "counterexample", "kept": "counterexample", "sumOk": "inconclusive", "doubledNonNeg": "verified"}
	for fn, status := range want {
		if rows[fn].Status != status {
			t.Errorf("%s: status %q, want %q (%s)", fn, rows[fn].Status, status, rows[fn].Reason)
		}
	}
	for _, fn := range []string{"sumBad", "kept"} {
		r := rows[fn]
		if !strings.HasPrefix(r.Reason, "witness replayed at runtime: ensures clause 1 is false for ") {
			t.Errorf("%s: reason %q does not report the replayed witness", fn, r.Reason)
		}
		if len(r.Model) == 0 {
			t.Errorf("%s: a replayed counterexample must carry its inputs", fn)
		}
	}
	if !strings.Contains(rows["sumOk"].Reason, "witness replay: contracts hold at runtime") {
		t.Errorf("sumOk: reason %q should say the replayed input satisfied the contracts", rows["sumOk"].Reason)
	}
}
