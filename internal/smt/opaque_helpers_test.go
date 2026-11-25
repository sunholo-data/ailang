package smt_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// Opaque helpers (ruled 2026-11-25): a helper whose body does not encode and
// that has no ensures is an uninterpreted function. A proof not depending on
// its value holds, with a caveat naming it; a refutation through it is
// inconclusive; its requires is still checked at each call; a cycle of such
// helpers is opaque too; a helper with an ensures stands in by its contract.
func TestOpaqueHelpers(t *testing.T) {
	report := verifyFixture(t, "opaque_helpers.ail")
	want := map[string]struct {
		status, reasonCode string
		caveat             string // fragment of the helpers_not_checked caveat; "" = none
	}{
		"ignoresHelper":       {"verified", "", "swapSum (tuple expressions cannot be encoded"},
		"dependsOnHelper":     {"inconclusive", "", ""},
		"callsWithinRequires": {"verified", "", "halfOfPos ("},
		"breaksRequires":      {"counterexample", "requires_violation", ""},
		"parityLabel":         {"verified", "", "evenLen → oddLen → evenLen"},
		"viaContract":         {"verified", "", ""},
		"emptyListArg":        {"verified", "", "countWords ("},
		"paramNamedResult":    {"verified", "", ""},
		"definedEmptyArg":     {"verified", "", ""},
		"hoistedEmptyArgs":    {"verified", "", "nested ("},
		"viaUntypedReturn":    {"skipped", "", ""},
		"throughMasked":       {"verified", "", `masked (calls user function "escapePairs" that is not SMT-encodable in this context: its signature uses an unencodable type`},
		"callsPairsDirectly":  {"skipped", "", ""},
		"throughBadMap":       {"verified", "", "bumpAll ("},
		"throughRecordResult": {"verified", "", "splitAt ("},
		"flattenLen":          {"verified", "", ""},
		// Was a false VERIFIED: Tagged was declared without pairs.
		"same":               {"skipped", "", ""},
		"throughNamedResult": {"verified", "", "mkPt ("},
		"sameBag":            {"skipped", "", ""},
	}
	for _, r := range report.Results {
		w, ok := want[r.Function]
		if !ok {
			continue
		}
		delete(want, r.Function)
		if r.Status != w.status {
			t.Errorf("%s: status %q (%s), want %q", r.Function, r.Status, r.Reason, w.status)
		}
		if w.reasonCode != "" && string(r.ReasonCode) != w.reasonCode {
			t.Errorf("%s: reason code %q, want %q", r.Function, r.ReasonCode, w.reasonCode)
		}
		detail := ""
		for _, c := range r.Caveats {
			if c.Kind == smt.CaveatOpaqueHelpers {
				detail = c.Detail
			}
		}
		if w.caveat == "" && detail != "" {
			t.Errorf("%s: unexpected helpers caveat %q", r.Function, detail)
		}
		if w.caveat != "" && !strings.Contains(detail, w.caveat) {
			t.Errorf("%s: helpers caveat %q lacks %q", r.Function, detail, w.caveat)
		}
		if r.Function == "callsPairsDirectly" && !strings.Contains(r.Reason, `calls "escapePairs" whose signature uses an unencodable type`) {
			t.Errorf("callsPairsDirectly: reason %q does not name the direct callee", r.Reason)
		}
		if r.Function == "dependsOnHelper" && !strings.Contains(r.Reason, "$spec_swapSum") {
			t.Errorf("dependsOnHelper: reason %q does not name the opaque helper", r.Reason)
		}
	}
	for fn := range want {
		t.Errorf("%s: no result", fn)
	}
}
