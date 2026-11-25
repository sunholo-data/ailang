package smt_test

import (
	"os"
	"path/filepath"
	"testing"
)

// Functions over std/json's Json (and a user ADT of the same shape) reach the
// solver. Through the CLI, every one of them was SKIPPED before: the record
// nested in JObject([{key, value}]) was referenced by Json's declaration but
// never declared. This harness loads std/json by a route that declared it
// anyway, so here only leafOr (a user ADT of the same shape) guards the fix;
// cmd/ailang TestVerify_JsonReachesSolver guards Json through the CLI.
func TestJsonVerifies(t *testing.T) {
	path := filepath.Join("testdata", "json", "json_verify.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"nameOr":      "verified",
		"nameBad":     "counterexample",
		"isNull":      "verified",
		"isNullBad":   "counterexample",
		"emptyObjKey": "counterexample",
		"leafOr":      "verified",
		"hasA":        "verified",
	}
	report, _ := verifySource(t, path, string(src))
	seen := map[string]bool{}
	for _, r := range report.Results {
		w, ok := want[r.Function]
		if !ok {
			continue
		}
		seen[r.Function] = true
		if r.Status != w {
			t.Errorf("%s: %s, want %s (%s)", r.Function, r.Status, w, r.Reason)
		}
	}
	for fn := range want {
		if !seen[fn] {
			t.Errorf("%s: no result", fn)
		}
	}
}
