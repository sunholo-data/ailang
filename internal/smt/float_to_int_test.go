package smt_test

import (
	"os"
	"path/filepath"
	"testing"
)

// floatToInt was encoded as Z3 to_int, which floors; the runtime truncates
// toward zero, so `ensures { result == -3 }` for floatToInt(-2.5) was VERIFIED
// while the program returns -2.
func TestFloatToIntTruncatesTowardZero(t *testing.T) {
	path := filepath.Join("testdata", "numeric", "float_to_int.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{"truncNeg": "verified", "truncPos": "verified", "floorNegBad": "counterexample"}
	for _, r := range report.Results {
		if w, ok := want[r.Function]; ok && r.Status != w {
			t.Errorf("%s: status %q, want %q (%s)", r.Function, r.Status, w, r.Reason)
		}
	}
}
