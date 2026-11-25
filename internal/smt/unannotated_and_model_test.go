package smt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// S5: a parameter without a type annotation has no SMT sort. It used to be
// dropped from the declarations and reach Z3 as an unknown constant (an
// ERROR); it is now a skip that names the parameter, in the verified function
// and in a helper it calls.
func TestUnannotatedParamsSkip(t *testing.T) {
	want := map[string]struct{ status, reason string }{
		"inc":        {"skipped", `unannotated parameter "x"`},
		"usesHelper": {"skipped", `calls "helper", whose parameter "x" has no type annotation`},
		"annotated":  {"verified", ""},
	}
	report := verifyFixture(t, "unannotated_params.ail")
	seen := map[string]bool{}
	for _, r := range report.Results {
		w, ok := want[r.Function]
		if !ok {
			continue
		}
		seen[r.Function] = true
		if r.Status != w.status || !strings.Contains(r.Reason, w.reason) {
			t.Errorf("%s: %s %q, want %s containing %q", r.Function, r.Status, r.Reason, w.status, w.reason)
		}
	}
	for fn := range want {
		if !seen[fn] {
			t.Errorf("%s: no result", fn)
		}
	}
}

// S6: a counterexample's result is the value the body evaluates to on the
// model's inputs, not the body term over those inputs.
func TestCounterexampleResultIsLiteral(t *testing.T) {
	report := verifyFixture(t, "model_values.ail")
	checked := 0
	for _, r := range report.Results {
		if r.Status != "counterexample" {
			t.Errorf("%s: %s (%s), want counterexample", r.Function, r.Status, r.Reason)
			continue
		}
		var result *smt.ModelBinding
		for i := range r.Model {
			if r.Model[i].Name == "result" {
				result = &r.Model[i]
			}
		}
		if result == nil {
			t.Errorf("%s: model has no result binding: %+v", r.Function, r.Model)
			continue
		}
		checked++
		for _, term := range []string{"ite", "let", "div", "$p_", "match", "n ", "x "} {
			if strings.Contains(result.Value+" ", term) {
				t.Errorf("%s: result %q is a term over the inputs (contains %q), want a value", r.Function, result.Value, term)
			}
		}
	}
	if checked != 4 {
		t.Errorf("checked %d counterexamples, want 4", checked)
	}
}

func verifyFixture(t *testing.T, name string) *smt.VerifyReport {
	t.Helper()
	path := filepath.Join("testdata", "unannotated", name)
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	return report
}

// The get-value response overrides the model's echo of the result define-const.
func TestParseModelPrefersResultValue(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable")
	}
	script := `(set-logic ALL)
(declare-const x Int)
(define-const result Int (ite (>= x 0) (+ x 7) (- x)))
(assert (= x 3))
(check-sat)
(get-model)
(get-value (result))
`
	res, err := smt.Solve(script, smt.DefaultSolverConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range res.Model {
		if b.Name == "result" {
			if b.Value != "10" {
				t.Errorf("result = %q, want 10 (raw: %s)", b.Value, res.RawOutput)
			}
			return
		}
	}
	t.Errorf("no result binding (raw: %s)", res.RawOutput)
}

// A model that omits the result define-const still reports the get-value
// answer, with the sort the script declared.
func TestSolveResultFromGetValueOnly(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 unavailable")
	}
	script := `(set-logic ALL)
(declare-const n Int)
(define-const result (Seq Int) (ite (> n 0) (seq.unit n) (as seq.empty (Seq Int))))
(assert (= n 4))
(assert (not (> (seq.len result) 2)))
(check-sat)
(get-model)
(get-value (result))
`
	res, err := smt.Solve(script, smt.DefaultSolverConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range res.Model {
		if b.Name == "result" {
			if b.Sort != "(Seq Int)" || strings.Contains(b.Value, "ite") || !strings.Contains(b.Value, "4") {
				t.Errorf("result = %+v, want sort (Seq Int) and a value containing 4 (raw: %s)", b, res.RawOutput)
			}
			return
		}
	}
	t.Errorf("no result binding (raw: %s)", res.RawOutput)
}
