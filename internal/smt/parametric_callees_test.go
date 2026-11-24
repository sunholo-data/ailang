package smt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/verification"
)

// Helpers whose signatures carry a parametric ADT at a concrete type
// (Option[int], Result[int, DivErr], a user Box[int]) are inlined like any
// other helper. Before, every caller here except usePlain and useTuple was
// SKIPPED by the callee-sort gate. A type variable, tuple, function type or
// bytes in a helper's signature still skips, and an unannotated helper is
// refused with a hint naming the fix.
func TestParametricCallees(t *testing.T) {
	path := filepath.Join("testdata", "parametric_callees", "param_callees.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	type want struct {
		status string
		code   verification.ReasonCode
		text   string // in the reason or the first rejection's hint
	}
	cases := map[string]want{
		"usePos":        {"verified", "", ""},
		"usePosBad":     {"counterexample", "", ""},
		"useDivOk":      {"verified", "", ""},
		"useDivBad":     {"counterexample", "", ""},
		"useOrZero":     {"verified", "", ""},
		"needsSome":     {"verified", "", ""},
		"callNeedsSome": {"counterexample", verification.RequiresViolation, "call to needsSome"},
		"useBox":        {"verified", "", ""},
		"useHalf":       {"verified", "", ""},
		"usePlain":      {"skipped", verification.UnsupportedEncoding, "Mark \"plain\" `pure`"},
		"useTuple":      {"skipped", verification.UnsupportedEncoding, "unencodable type \"(int, int)\""},
	}
	report, _ := verifySource(t, path, string(src))
	seen := map[string]bool{}
	for _, r := range report.Results {
		w, ok := cases[r.Function]
		if !ok {
			continue
		}
		seen[r.Function] = true
		if r.Status != w.status || (w.code != "" && r.ReasonCode != w.code) {
			t.Errorf("%s: %s/%s, want %s/%s (%s)", r.Function, r.Status, r.ReasonCode, w.status, w.code, r.Reason)
		}
		text := r.Reason
		if len(r.Rejections) > 0 {
			text += " " + r.Rejections[0].Hint
		}
		if w.text != "" && !strings.Contains(text, w.text) {
			t.Errorf("%s: %q should mention %q", r.Function, text, w.text)
		}
	}
	for fn := range cases {
		if !seen[fn] {
			t.Errorf("%s: no result", fn)
		}
	}
}
