package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/smt"
)

// TestVerify_JsonReachesSolver runs the real CLI on functions over std/json's
// Json. Json's JObject([{key, value}]) nests a record in a list; until that
// record was declared, every function here was SKIPPED ("sort
// Record_key_value referenced in declaration of Json is not declared"). The
// in-package harness (internal/smt TestJsonVerifies) loads std/json by another
// route that never hit the gap, so this e2e is the guard for Json itself.
func TestVerify_JsonReachesSolver(t *testing.T) {
	if !smt.Z3Available() {
		t.Skip("Z3 not installed (e.g. Windows CI) — verify e2e needs the solver")
	}
	bin := buildAilang(t)
	got := verifyStatuses(t, bin, "--relax-modules", "internal/smt/testdata/json/json_verify.ail")
	want := map[string]string{
		"nameOr":      "verified",
		"nameBad":     "counterexample",
		"isNull":      "verified",
		"isNullBad":   "counterexample",
		"emptyObjKey": "counterexample",
		"leafOr":      "verified",
		"hasA":        "verified",
	}
	for fn, status := range want {
		if r := got[fn]; r.Status != status {
			t.Errorf("%s = %+v, want %s", fn, r, status)
		}
	}
}
