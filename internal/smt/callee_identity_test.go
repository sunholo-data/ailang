package smt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// S7 (M-VERIFY-SOUNDNESS-COVERAGE M1): callee bodies were looked up by bare
// name, first hit over a Go map, so two loaded modules defining the same name
// let the verifier prove a contract about the other module's function. Before
// the fix: two_imports.usesBad was VERIFIED in 5 of 6 runs although a/bad.clamp0
// returns x - 1, and local_shadow.usesImported was VERIFIED every run because
// the checked module's own clamp0 shadowed the imported one.
func verifyCalleeIdentityFixture(t *testing.T, name string) map[string]smt.VerifyResult {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("a", name+".ail"))
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, filepath.Join("a", name+".ail"), string(src))
	out := make(map[string]smt.VerifyResult, len(report.Results))
	for _, r := range report.Results {
		out[r.Function] = r
	}
	return out
}

func TestCalleeIdentityNeverBindsAnotherModulesFunction(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("testdata", "callee_identity"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	cases := []struct {
		fixture, function, wantStatus, reasonHas string
	}{
		// a/bad.clamp0 = x - 1: the contract is false, and the bare lookup
		// (sorted: a/bad before a/good) binds the right body.
		{"two_imports", "usesBad", "counterexample", ""},
		// The bare lookup would bind a/bad.clamp0 for a call to a/good.clamp0.
		{"two_imports", "usesGood", "skipped", "a/good.clamp0"},
		// The checked module's own clamp0 is the one called: no ambiguity.
		{"local_shadow", "usesLocal", "verified", ""},
		// The bare lookup would bind the LOCAL clamp0 (correct, x >= 0) for a call
		// to a/bad.clamp0 (x - 1): the old false VERIFIED.
		{"local_shadow", "usesImported", "skipped", "a/bad.clamp0"},
		// A parameter named like a top-level function is not a call.
		{"param_shadow", "pick", "verified", ""},
	}
	for _, tc := range cases {
		t.Run(tc.fixture+"/"+tc.function, func(t *testing.T) {
			// Repeat: the bug was nondeterministic (Go map iteration).
			for i := 0; i < 5; i++ {
				got := verifyCalleeIdentityFixture(t, tc.fixture)[tc.function]
				if got.Status != tc.wantStatus {
					t.Fatalf("run %d: %s.%s status = %q (reason %q), want %q", i, tc.fixture, tc.function, got.Status, got.Reason, tc.wantStatus)
				}
				if tc.reasonHas != "" && !strings.Contains(got.Reason, tc.reasonHas) {
					t.Fatalf("%s.%s reason %q does not name %q", tc.fixture, tc.function, got.Reason, tc.reasonHas)
				}
			}
		})
	}
}
