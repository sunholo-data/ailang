package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/smt"
)

// TestAICheckExitCode_Lanes is the AC3.2 guard.
//
// `ai-check` is documented to AI agents as THE convergence signal, so its process
// exit status must never disagree with the JSON it just printed. Before
// M-Z3-ADT-RECORD-SORT the verifier-errors lane was missing entirely: a file whose
// report said `verify.errors: 1` still exited 0.
//
// Red mutation: delete `|| verify.Errors > 0` from aiCheckExitCode. It compiles,
// and the "verifier errors" and "errors dominate a clean check" cases below fail.
// (Measured end-to-end: with that clause removed, cross_module_functions.ail —
// check.passed=true, verify.errors=3 — exits 0 instead of 1.)
func TestAICheckExitCode_Lanes(t *testing.T) {
	tests := []struct {
		name  string
		check aiCheckSection
		verif aiVerifySection
		want  int
	}{
		{
			name:  "clean check, nothing verified",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true},
			want:  0,
		},
		{
			name:  "verified only",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Verified: 3},
			want:  0,
		},
		{
			// A skip is "not proved", NOT "disproved" — it must stay green, else
			// every unencodable-but-correct program would fail the gate.
			name:  "skipped only",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Verified: 1, Skipped: 2},
			want:  0,
		},
		{
			name:  "counterexample",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Counterexample: 1},
			want:  1,
		},
		{
			// THE regression this sprint closed.
			name:  "verifier errors",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Errors: 1},
			want:  1,
		},
		{
			name:  "errors dominate an otherwise clean run",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Verified: 5, Skipped: 1, Errors: 3},
			want:  1,
		},
		{
			name:  "check failure",
			check: aiCheckSection{Passed: false},
			verif: aiVerifySection{Available: false},
			want:  1,
		},
		{
			name:  "check failure outranks a clean verify section",
			check: aiCheckSection{Passed: false},
			verif: aiVerifySection{Available: true, Verified: 2},
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aiCheckExitCode(tt.check, tt.verif); got != tt.want {
				t.Fatalf("aiCheckExitCode() = %d, want %d (check.Passed=%v verify=%+v)",
					got, tt.want, tt.check.Passed, tt.verif)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// M-V1-SIMPLIFY-S4 M3A: ai-check runs THE verification loop (smt.Verify).
//
// Until this sprint ai-check carried a private copy of `verify`'s loop that
// had drifted in three measured ways. The tests below pin each one so the
// loops cannot silently diverge again. Two behaviours are now SHARED with
// `ailang verify`; the third is a DELIBERATE ai-check difference behind
// smt.VerifyOptions.UnsupportedConstructIsError.
// ---------------------------------------------------------------------------

// TestAICheckVerifyOptions_Contract pins the option set ai-check hands the
// shared loop: the flags flow through, and the one deliberate difference is
// on. Red mutation: drop UnsupportedConstructIsError from aiCheckVerifyOptions
// — this fails, and so does TestAICheck_UnsupportedConstructIsAnError below.
func TestAICheckVerifyOptions_Contract(t *testing.T) {
	got := aiCheckVerifyOptions(7*time.Second, 4)
	want := smt.VerifyOptions{Timeout: 7 * time.Second, RecursiveDepth: 4, UnsupportedConstructIsError: true}
	if got != want {
		t.Fatalf("aiCheckVerifyOptions() = %+v, want %+v", got, want)
	}
}

// TestAIVerifySectionFromReport_Projection pins the report→JSON projection:
// the four counters copy across, and "uncontracted" rows (which verify lists
// for denominator honesty) are NOT surfaced — ai-check's results have only
// ever carried contract-bearing functions, and its consumers read the counters.
func TestAIVerifySectionFromReport_Projection(t *testing.T) {
	report := &smt.VerifyReport{
		Results: []smt.VerifyResult{
			{Function: "a", Status: "verified"},
			{Function: "main", Status: "uncontracted"},
			{Function: "b", Status: "skipped"},
		},
		Verified: 1, Skipped: 1, Uncontracted: 1,
	}
	got := aiVerifySectionFromReport(report)
	if !got.Available || got.Verified != 1 || got.Skipped != 1 || got.Errors != 0 || got.Counterexample != 0 {
		t.Fatalf("counters not projected: %+v", got)
	}
	if len(got.Results) != 2 {
		t.Fatalf("want 2 contract-bearing results, got %d: %+v", len(got.Results), got.Results)
	}
	for _, r := range got.Results {
		if r.Status == "uncontracted" {
			t.Fatalf("uncontracted row leaked into ai-check results: %+v", r)
		}
	}
}

type aiCheckPayload struct {
	Check struct {
		Passed bool `json:"passed"`
	} `json:"check"`
	Verify struct {
		Verified       int `json:"verified"`
		Counterexample int `json:"counterexample"`
		Skipped        int `json:"skipped"`
		Errors         int `json:"errors"`
		Results        []struct {
			Function     string `json:"function"`
			Status       string `json:"status"`
			Reason       string `json:"reason"`
			BoundedDepth int    `json:"bounded_depth"`
		} `json:"results"`
	} `json:"verify"`
}

func runAICheckJSON(t *testing.T, args ...string) (aiCheckPayload, string, int) {
	t.Helper()
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — ai-check e2e needs the solver")
	}
	bin := buildAilang(t)
	stdout, stderr, code := runAilangBin(t, bin, append([]string{"ai-check"}, args...)...)
	var p aiCheckPayload
	if err := json.Unmarshal([]byte(stdout), &p); err != nil {
		t.Fatalf("ai-check did not emit valid JSON (exit %d): %v\nstdout:\n%s\nstderr:\n%s", code, err, stdout, stderr)
	}
	return p, stdout + stderr, code
}

// TestAICheck_CrossModuleCalleesAreShared — SHARED behaviour #1. Before the
// fold ai-check passed no ImportedPrograms to the encoder, so every cross-
// module callee was an "unknown constant" Z3 error: on this fixture `verify`
// reported 3 verified while `ai-check` reported 3 errors and exited 1 (the
// exact discrepancy the AC3.2 comment above records). One loop, one answer.
func TestAICheck_CrossModuleCalleesAreShared(t *testing.T) {
	p, combined, code := runAICheckJSON(t, "examples/runnable/contracts/cross_module_functions.ail")
	if !p.Check.Passed {
		t.Fatalf("instrument failure: fixture no longer type-checks:\n%s", combined)
	}
	if strings.Contains(combined, "unknown constant") {
		t.Fatalf("ai-check still leaks the pre-fold Z3 error (no ImportedPrograms):\n%s", combined)
	}
	if p.Verify.Verified != 3 || p.Verify.Errors != 0 || p.Verify.Counterexample != 0 {
		t.Fatalf("want 3 verified / 0 errors (what `verify` reports), got %+v", p.Verify)
	}
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, r := range p.Verify.Results {
		if r.Status == "uncontracted" {
			t.Fatalf("uncontracted row surfaced in ai-check JSON: %+v", r)
		}
	}
}

// TestAICheck_PerFunctionDepthOverrideIsShared — SHARED behaviour #2. The
// private loop ignored @verify(depth: N); fibonacci in this fixture declares
// depth 5 and was verified at the global default of 2 (bounded_depth 2 in the
// pre-fold JSON). It now reports 5, exactly as `verify` does; factorial keeps
// its own declared 2.
func TestAICheck_PerFunctionDepthOverrideIsShared(t *testing.T) {
	p, combined, _ := runAICheckJSON(t, "examples/runnable/contracts/per_function_depth_verify.ail")
	if !p.Check.Passed {
		t.Fatalf("instrument failure: fixture no longer type-checks:\n%s", combined)
	}
	depths := map[string]int{}
	for _, r := range p.Verify.Results {
		depths[r.Function] = r.BoundedDepth
	}
	if depths["fibonacci"] != 5 {
		t.Fatalf("fibonacci bounded_depth = %d, want 5 from @verify(depth: 5); results: %+v", depths["fibonacci"], p.Verify.Results)
	}
	if depths["factorial"] != 2 {
		t.Fatalf("factorial bounded_depth = %d, want 2 from @verify(depth: 2)", depths["factorial"])
	}
}

// TestAICheck_UnsupportedConstructIsAnError — DELIBERATE difference #3. A
// contracted function matching on list patterns is outside the decidable
// fragment. `ailang verify` reports it as skipped (#757); ai-check has always
// reported it as a verifier ERROR and exited 1, and the eval harness banks
// verify_errors from that. This pins today's contract explicitly — changing
// it is a bank-forward ruling. Both lanes name the construct in plain words;
// neither may leak a Go type name.
func TestAICheck_UnsupportedConstructIsAnError(t *testing.T) {
	src := `module m

export func sumList(ls: [int]) -> int ! {}
ensures { result >= 0 }
{
  match ls {
    [] => 0,
    x :: rest => x + sumList(rest)
  }
}
`
	f := filepath.Join(t.TempDir(), "m.ail")
	if err := os.WriteFile(f, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	p, combined, code := runAICheckJSON(t, "--relax-modules", f)
	if !p.Check.Passed {
		t.Fatalf("instrument failure: fixture no longer type-checks:\n%s", combined)
	}
	if strings.Contains(combined, "*core.ListPattern") {
		t.Fatalf("ai-check leaked the internal pattern type name:\n%s", combined)
	}
	if p.Verify.Errors != 1 || p.Verify.Skipped != 0 {
		t.Fatalf("want errors=1 skipped=0 (ai-check's contract), got %+v", p.Verify)
	}
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (verify.errors > 0)", code)
	}
	if len(p.Verify.Results) != 1 || p.Verify.Results[0].Status != "error" ||
		!strings.Contains(p.Verify.Results[0].Reason, "list patterns") {
		t.Fatalf("want one error result naming the construct, got %+v", p.Verify.Results)
	}
}
