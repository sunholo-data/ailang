package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// M-VERIFY-BOUNDED-UNROLLING-FALSE-COUNTEREXAMPLE M2 — end to end on real Z3.
//
// Bounded unrolling expands a recursive f into f_N → … → f_1 → f_0, where f_0
// is an UNINTERPRETED stub. A sat on that encoding can rest on an invented
// value for f_0, so before this fix a correct recursive function was reported
// as a VIOLATION (measured on the fixtures below: sLength, cRec), ai-check
// exited 1, and the eval harness fed the "counterexample" back to the model as
// a repair turn. Such a sat is now "inconclusive": not a proof, not a
// violation.

const boundedFixtures = "cmd/ailang/testdata/verify_bounded/"

type boundedRow struct {
	Function     string `json:"function"`
	Status       string `json:"status"`
	Reason       string `json:"reason"`
	ReasonCode   string `json:"reason_code"`
	BoundedDepth int    `json:"bounded_depth"`
	Model        []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"model"`
}

type boundedCounters struct {
	Verified       int          `json:"verified"`
	Counterexample int          `json:"counterexample"`
	Inconclusive   int          `json:"inconclusive"`
	Skipped        int          `json:"skipped"`
	Errors         int          `json:"errors"`
	Uncontracted   int          `json:"uncontracted"`
	Results        []boundedRow `json:"results"`
}

func (c boundedCounters) byFunc() map[string]boundedRow {
	m := make(map[string]boundedRow, len(c.Results))
	for _, r := range c.Results {
		m[r.Function] = r
	}
	return m
}

// assertPartition is INV-1: the counters partition the per-function results,
// so a status emitted without a counter (or the reverse) is loud.
func assertPartition(t *testing.T, label string, c boundedCounters) {
	t.Helper()
	sum := c.Verified + c.Counterexample + c.Inconclusive + c.Skipped + c.Errors + c.Uncontracted
	if sum != len(c.Results) {
		t.Fatalf("%s: INV-1 broken — counters sum %d != %d results: %+v", label, sum, len(c.Results), c)
	}
}

func boundedAICheck(t *testing.T, bin, fixture string) (boundedCounters, int) {
	t.Helper()
	stdout, stderr, code := runAilangBin(t, bin, "ai-check", "-verify-recursive-depth", "2", boundedFixtures+fixture)
	var p struct {
		Check struct {
			Passed bool `json:"passed"`
		} `json:"check"`
		Verify boundedCounters `json:"verify"`
	}
	if err := json.Unmarshal([]byte(stdout), &p); err != nil {
		t.Fatalf("ai-check %s: invalid JSON (exit %d): %v\n%s\n%s", fixture, code, err, stdout, stderr)
	}
	if !p.Check.Passed {
		t.Fatalf("instrument failure: %s no longer type-checks:\n%s", fixture, stdout)
	}
	assertPartition(t, "ai-check "+fixture, p.Verify)
	return p.Verify, code
}

func boundedVerifyJSON(t *testing.T, bin, fixture string, extra ...string) (boundedCounters, int) {
	t.Helper()
	args := append([]string{"verify", "--json", "--verify-recursive-depth", "2"}, extra...)
	stdout, stderr, code := runAilangBin(t, bin, append(args, boundedFixtures+fixture)...)
	var c boundedCounters
	if err := json.Unmarshal([]byte(stdout), &c); err != nil {
		t.Fatalf("verify --json %s: invalid JSON (exit %d): %v\n%s\n%s", fixture, code, err, stdout, stderr)
	}
	assertPartition(t, "verify "+fixture, c)
	return c, code
}

func requireZ3(t *testing.T) {
	t.Helper()
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — bounded-unrolling e2e needs the solver")
	}
}

func assertInconclusive(t *testing.T, r boundedRow, depth int) {
	t.Helper()
	if r.Status != "inconclusive" {
		t.Fatalf("%s status = %q, want inconclusive (a bounded-unrolling sat is not a violation)", r.Function, r.Status)
	}
	if r.BoundedDepth != depth || r.Reason == "" || r.ReasonCode != "bounded_unrolling_inconclusive" {
		t.Fatalf("%s lost its qualification: bounded_depth=%d reason=%q reason_code=%q", r.Function, r.BoundedDepth, r.Reason, r.ReasonCode)
	}
	if len(r.Model) != 0 {
		t.Fatalf("%s: frontier-dependent model surfaced by default: %+v", r.Function, r.Model)
	}
}

// TestBounded_ADT_CorrectRecursionIsInconclusive (AC-3): sLength's clause is
// true for every input. Base: rc=1, counterexample=1 with a "witness" over the
// stub sLength_0. After: inconclusive=1, the control headOr0 verified. The
// exit is 1 since M-V1-VERIFICATION-HONESTY M1 (inconclusive is "not proved",
// and an unproved obligation fails by default; --allow-inconclusive passes it).
func TestBounded_ADT_CorrectRecursionIsInconclusive(t *testing.T) {
	requireZ3(t)
	bin := buildAilang(t)
	c, code := boundedAICheck(t, bin, "adt.ail")
	if c.Counterexample != 0 || c.Inconclusive != 1 || c.Verified != 1 {
		t.Fatalf("want verified=1 counterexample=0 inconclusive=1, got %+v", c)
	}
	if code != 1 {
		t.Fatalf("ai-check exit = %d, want 1 (inconclusive fails by default)", code)
	}
	rows := c.byFunc()
	assertInconclusive(t, rows["sLength"], 2)
	if rows["headOr0"].Status != "verified" {
		t.Fatalf("control headOr0 = %q, want verified", rows["headOr0"].Status)
	}
}

// TestBounded_Discrim_GenuineNonRecursiveViolationSurvives (AC-4, kills
// MUT-OVERSUPPRESS): same clause on three shapes. bBad is non-recursive and
// genuinely wrong — it keeps its concrete counterexample and rc=1.
func TestBounded_Discrim_GenuineNonRecursiveViolationSurvives(t *testing.T) {
	requireZ3(t)
	bin := buildAilang(t)
	c, code := boundedAICheck(t, bin, "discrim.ail")
	if c.Counterexample != 1 || c.Inconclusive != 1 || c.Verified != 1 {
		t.Fatalf("want verified=1 counterexample=1 inconclusive=1, got %+v", c)
	}
	if code != 1 {
		t.Fatalf("ai-check exit = %d, want 1 (bBad is a real violation)", code)
	}
	rows := c.byFunc()
	if rows["aOk"].Status != "verified" {
		t.Fatalf("aOk = %q, want verified", rows["aOk"].Status)
	}
	bad := rows["bBad"]
	if bad.Status != "counterexample" || len(bad.Model) == 0 || bad.BoundedDepth != 0 {
		t.Fatalf("bBad lost its genuine counterexample: %+v", bad)
	}
	for _, b := range bad.Model {
		if strings.Contains(b.Value, "_0") {
			t.Fatalf("bBad model references a frontier stub: %+v", bad.Model)
		}
	}
	assertInconclusive(t, rows["cRec"], 2)
}

// TestBounded_Q3_RecursiveBugsAreInconclusive (AC-5): three recursive
// functions with genuinely false clauses. The fix deliberately errs toward
// inconclusive — the measured "witness" for lenAtLeast1 satisfied its own
// clause — so none is a counterexample. Since M-V1-VERIFICATION-HONESTY M1 the
// inconclusive lane itself fails (exit 1), which is what keeps such a bug from
// passing any gate, strict or not.
func TestBounded_Q3_RecursiveBugsAreInconclusive(t *testing.T) {
	requireZ3(t)
	bin := buildAilang(t)
	c, code := boundedAICheck(t, bin, "q3.ail")
	if c.Counterexample != 0 || c.Inconclusive != 3 {
		t.Fatalf("want counterexample=0 inconclusive=3, got %+v", c)
	}
	if code != 1 {
		t.Fatalf("ai-check exit = %d, want 1", code)
	}
	for _, fn := range []string{"lenAtLeast1", "sumNonNeg", "lenAtMost2"} {
		assertInconclusive(t, c.byFunc()[fn], 2)
	}
}

// TestBounded_VerifyAndAICheckAgree (AC-6, kills MUT-ONE-LADDER): the two
// commands report identical per-function statuses on every fixture.
func TestBounded_VerifyAndAICheckAgree(t *testing.T) {
	requireZ3(t)
	bin := buildAilang(t)
	for _, fx := range []string{"adt.ail", "discrim.ail", "q3.ail"} {
		ac, _ := boundedAICheck(t, bin, fx)
		vc, _ := boundedVerifyJSON(t, bin, fx)
		got, want := map[string]string{}, map[string]string{}
		for _, r := range ac.Results {
			got[r.Function] = r.Status
		}
		for _, r := range vc.Results {
			if r.Status != "uncontracted" {
				want[r.Function] = r.Status
			}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: ai-check %v != verify %v", fx, got, want)
		}
		if ac.Inconclusive != vc.Inconclusive || ac.Counterexample != vc.Counterexample {
			t.Fatalf("%s: counters drift: ai-check %+v verify %+v", fx, ac, vc)
		}
	}
}

// TestBounded_VerifyExitCodes (AC-7, revised by M-V1-VERIFICATION-HONESTY
// M1): inconclusive fails by default and under --strict; --allow-inconclusive
// is the explicit opt-in that passes it.
func TestBounded_VerifyExitCodes(t *testing.T) {
	requireZ3(t)
	bin := buildAilang(t)
	if _, code := boundedVerifyJSON(t, bin, "adt.ail"); code != 1 {
		t.Fatalf("verify (default) exit = %d, want 1 on inconclusive > 0", code)
	}
	if _, code := boundedVerifyJSON(t, bin, "adt.ail", "--strict"); code != 1 {
		t.Fatalf("verify --strict exit = %d, want 1 on inconclusive > 0", code)
	}
	if _, code := boundedVerifyJSON(t, bin, "adt.ail", "--allow-inconclusive"); code != 0 {
		t.Fatalf("verify --allow-inconclusive exit = %d, want 0", code)
	}
}

// TestBounded_HumanOutput (kills MUT-PRINT-SWALLOW): the human report names
// the outcome honestly and never prints it as a violation or an error.
func TestBounded_HumanOutput(t *testing.T) {
	requireZ3(t)
	bin := buildAilang(t)
	stdout, stderr, _ := runAilangBin(t, bin, "verify", "--verify-recursive-depth", "2", boundedFixtures+"adt.ail")
	out := stdout + stderr
	for _, want := range []string{"? INCONCLUSIVE (bounded: depth 2)", "sLength", "neither a proof nor a confirmed violation", "1 inconclusive"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q:\n%s", want, out)
		}
	}
	for _, bad := range []string{"VIOLATION", "! ERROR", "violations"} {
		if strings.Contains(out, bad) {
			t.Fatalf("human output reports inconclusive as %q:\n%s", bad, out)
		}
	}
}

// TestBounded_UnencodableErrorSurfaceUntouched (AC-11): the sibling
// "unencodable" surface is out of scope and must not move. (The design pinned
// 3 verified / 1 error; that row has since landed — the base binary measured
// 3 verified / 1 skipped on 2026-11-09, which is what is pinned here.)
func TestBounded_UnencodableErrorSurfaceUntouched(t *testing.T) {
	requireZ3(t)
	bin := buildAilang(t)
	stdout, stderr, code := runAilangBin(t, bin, "ai-check", "examples/runnable/contracts/shapes_verify.ail")
	var p struct {
		Verify boundedCounters `json:"verify"`
	}
	if err := json.Unmarshal([]byte(stdout), &p); err != nil {
		t.Fatalf("invalid JSON (exit %d): %v\n%s", code, err, stderr)
	}
	if p.Verify.Verified != 3 || p.Verify.Skipped != 1 || p.Verify.Errors != 0 || p.Verify.Inconclusive != 0 || p.Verify.Counterexample != 0 {
		t.Fatalf("shapes_verify surface moved: %+v", p.Verify)
	}
}
