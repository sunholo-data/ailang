package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/verification"
)

// The exit lanes of `ai-check` / `check --verify` are no longer decided here:
// every verification command exits through verification.ExitCode
// (internal/verification/outcome_test.go pins the table) and
// verify_exit_contract_test.go drives each lane end to end on every command
// spelling with fake solvers. What stays in this file is ai-check's own
// projection and the behaviours it shares with `verify` through smt.Verify.

// TestAICheckVerifyOptions_Contract pins the option set ai-check hands the
// shared loop: the flags flow through and nothing else differs from `verify`.
// (Until M-V1-VERIFICATION-HONESTY M1 ai-check also set
// UnsupportedConstructIsError, reporting an unsupported construct as an error
// where `verify` skipped it — the commands disagreed on exit status for the
// same file. That lane is gone; see TestVerifyExit_UnsupportedConstructIsASkipEverywhere.)
func TestAICheckVerifyOptions_Contract(t *testing.T) {
	got := aiCheckVerifyOptions(7*time.Second, 4)
	want := smt.VerifyOptions{Timeout: 7 * time.Second, RecursiveDepth: 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("aiCheckVerifyOptions() = %+v, want %+v", got, want)
	}
}

// TestAIVerifySectionFromReport_Projection pins the report→JSON projection:
// the counters copy across, obligations counts the contract-bearing rows, and
// "uncontracted" rows (which verify lists for denominator honesty) are NOT
// surfaced — ai-check's results have only ever carried contract-bearing
// functions, and its consumers read the counters.
func TestAIVerifySectionFromReport_Projection(t *testing.T) {
	report := &smt.VerifyReport{
		Results: []smt.VerifyResult{
			{Function: "a", Status: "verified"},
			{Function: "main", Status: "uncontracted"},
			{Function: "b", Status: "skipped"},
			{Function: "c", Status: "inconclusive"},
		},
		Verified: 1, Skipped: 1, Uncontracted: 1, Inconclusive: 1,
	}
	got := aiVerifySectionFromReport(report)
	if !got.Available || got.Verified != 1 || got.Skipped != 1 || got.Errors != 0 || got.Counterexample != 0 || got.Inconclusive != 1 {
		t.Fatalf("counters not projected: %+v", got)
	}
	if got.Obligations != 3 {
		t.Fatalf("obligations = %d, want 3 (uncontracted is not an obligation)", got.Obligations)
	}
	if len(got.Results) != 3 {
		t.Fatalf("want 3 contract-bearing results, got %d: %+v", len(got.Results), got.Results)
	}
	for _, r := range got.Results {
		if r.Status == "uncontracted" {
			t.Fatalf("uncontracted row leaked into ai-check results: %+v", r)
		}
	}
	if tally := got.tally(true); verification.Classify(tally) != verification.OutcomeInconclusive {
		t.Fatalf("tally %+v classifies as %q, want inconclusive", tally, verification.Classify(tally))
	}
}

type aiCheckPayload struct {
	Check struct {
		Passed bool `json:"passed"`
	} `json:"check"`
	Verify struct {
		Outcome        string `json:"outcome"`
		Obligations    int    `json:"obligations"`
		Available      bool   `json:"available"`
		Verified       int    `json:"verified"`
		Counterexample int    `json:"counterexample"`
		Inconclusive   int    `json:"inconclusive"`
		Skipped        int    `json:"skipped"`
		Errors         int    `json:"errors"`
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
// reported 3 verified while `ai-check` reported 3 errors and exited 1. One
// loop, one answer.
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
// private loop ignored @verify(depth: N) and verified at the global default
// of 2. The fixture now declares fibonacci depth 6 and factorial depth 5 (both
// past their requires bounds, so they verify rather than stay inconclusive);
// ai-check reports each declared depth, exactly as `verify` does, and exits 0.
func TestAICheck_PerFunctionDepthOverrideIsShared(t *testing.T) {
	p, combined, code := runAICheckJSON(t, "examples/runnable/contracts/per_function_depth_verify.ail")
	if !p.Check.Passed {
		t.Fatalf("instrument failure: fixture no longer type-checks:\n%s", combined)
	}
	if code != 0 || p.Verify.Outcome != "verified" {
		t.Fatalf("exit %d, outcome %q; the example must verify as documented\n%s", code, p.Verify.Outcome, combined)
	}
	depths := map[string]int{}
	for _, r := range p.Verify.Results {
		depths[r.Function] = r.BoundedDepth
	}
	if depths["fibonacci"] != 6 {
		t.Fatalf("fibonacci bounded_depth = %d, want 6 from @verify(depth: 6); results: %+v", depths["fibonacci"], p.Verify.Results)
	}
	if depths["factorial"] != 5 {
		t.Fatalf("factorial bounded_depth = %d, want 5 from @verify(depth: 5)", depths["factorial"])
	}
}

// TestContractExamples_PassAsDocumented: the three contract examples whose
// recursive contracts were inconclusive at the default depth used to pass
// `ailang verify` (inconclusive exited 0). Under the shared exit contract an
// inconclusive result fails, so each example now passes the command its
// header documents: per-function @verify depths past the requires bound
// (recursive_verify, per_function_depth_verify), or --allow-inconclusive for
// the library whose sumTo has an unbounded input.
func TestContractExamples_PassAsDocumented(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — these are real-solver runs")
	}
	bin := buildAilang(t)
	for _, args := range [][]string{
		{"verify", "examples/runnable/contracts/recursive_verify.ail"},
		{"verify", "examples/runnable/contracts/per_function_depth_verify.ail"},
		{"verify", "--verify-recursive-depth", "1", "examples/runnable/contracts/per_function_depth_verify.ail"},
		{"verify", "--allow-inconclusive", "examples/runnable/contracts/cross_module_functions_lib.ail"},
		{"ai-check", "examples/runnable/contracts/recursive_verify.ail"},
	} {
		stdout, stderr, code := runAilangBin(t, bin, args...)
		if code != 0 {
			t.Fatalf("%v: exit %d, want 0 as the example documents\nstdout:\n%s\nstderr:\n%s", args, code, stdout, stderr)
		}
	}
}
