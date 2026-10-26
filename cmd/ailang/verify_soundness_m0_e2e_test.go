package main

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// M-SMT-WELL-SORTED-PREFLIGHT M0 end-to-end tests: two false-VERIFIED paths.
//   - #1960: Int `/` and `%` were encoded as SMT-LIB Euclidean div/mod, but AILANG
//     truncates toward zero (Go semantics).
//   - #1961: every call to a contract-only callee collapsed into one shared constant,
//     so total(a) - total(a + 1) "proved" equal to 0.

type m0ModelBinding struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type m0Row struct {
	Function string           `json:"function"`
	Status   string           `json:"status"`
	Reason   string           `json:"reason"`
	Model    []m0ModelBinding `json:"model"`
}

func m0Verify(t *testing.T, file string) map[string]m0Row {
	t.Helper()
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	stdout, stderr, _ := runAilangBin(t, bin, "verify", "--json", file)
	var payload struct {
		Results []m0Row `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("verify --json invalid: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	out := make(map[string]m0Row, len(payload.Results))
	for _, r := range payload.Results {
		out[r.Function] = r
	}
	return out
}

func m0WantStatus(t *testing.T, got map[string]m0Row, fn, want string) m0Row {
	t.Helper()
	r, ok := got[fn]
	if !ok {
		t.Fatalf("%s missing from verify results: %+v", fn, got)
	}
	if r.Status != want {
		t.Fatalf("%s status = %q (reason %q), want %q", fn, r.Status, r.Reason, want)
	}
	return r
}

// m0ParamValue reads a parameter's value from a counterexample model. Parameters
// are declared as `$p_<name>`; Z3 prints negatives as `(- 7)`.
func m0ParamValue(t *testing.T, r m0Row, param string) int {
	t.Helper()
	for _, b := range r.Model {
		if b.Name != smt.ParamRenamePrefix+param {
			continue
		}
		v := b.Value
		if len(v) > 4 && v[:3] == "(- " {
			v = "-" + v[3:len(v)-1]
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("%s: model value for %s is not an integer: %q", r.Function, param, b.Value)
		}
		return n
	}
	t.Fatalf("%s: counterexample has no binding for %s: %+v", r.Function, param, r.Model)
	return 0
}

// TestVerify_IntDivModTruncates (#1960): the false contracts give a counterexample at a
// negative input, and contracts that hold only under truncation now VERIFY.
func TestVerify_IntDivModTruncates(t *testing.T) {
	got := m0Verify(t, "cmd/ailang/testdata/verify_preflight/int_divmod.ail")
	for _, fn := range []string{"parity", "halfDown"} {
		r := m0WantStatus(t, got, fn, "counterexample")
		if x := m0ParamValue(t, r, "x"); x >= 0 {
			t.Fatalf("%s counterexample x = %d, want a negative input", fn, x)
		}
	}
	for _, fn := range []string{"halfNonNeg", "parityBounded", "halfTowardZero", "remSign"} {
		m0WantStatus(t, got, fn, "verified")
	}
}

// TestVerify_ContractCalleePerCallSite (#1961): each call to a contract-only callee has
// its own result, constrained by the callee's ensures at that call's arguments and
// guarded by the callee's requires.
func TestVerify_ContractCalleePerCallSite(t *testing.T) {
	got := m0Verify(t, "cmd/ailang/testdata/verify_preflight/call_site_collapse.ail")

	r := m0WantStatus(t, got, "gap", "counterexample")
	if a := m0ParamValue(t, r, "a"); a < 0 {
		t.Fatalf("gap counterexample a = %d, want a >= 0 (the requires holds)", a)
	}
	r = m0WantStatus(t, got, "atLeastUnguarded", "counterexample")
	if a := m0ParamValue(t, r, "a"); a >= 0 {
		t.Fatalf("atLeastUnguarded counterexample a = %d, want a < 0 (outside lowBound's requires)", a)
	}
	for _, fn := range []string{"selfGap", "sumBoth", "atLeast"} {
		m0WantStatus(t, got, fn, "verified")
	}
	for fn, row := range got {
		for _, b := range row.Model {
			if b.Name == "total__spec" || b.Name == "lowBound__spec" {
				t.Fatalf("%s: counterexample lists the uninterpreted function %q as a value: %+v", fn, b.Name, b)
			}
		}
	}
}
