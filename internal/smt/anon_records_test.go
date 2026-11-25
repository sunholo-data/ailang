package smt_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Anonymous records in parameters (M-VERIFY-SOUNDNESS-COVERAGE): a record in
// a list was never declared, and one with an ADT field was declared before its
// ADT, so every such function was skipped. Two inline records sharing field
// names but not field types share one sort name; a function needing it was a
// Z3 sort-mismatch ERROR and is now a skip that names the collision.
func TestAnonymousRecordParams(t *testing.T) {
	path := filepath.Join("testdata", "unannotated", "anon_records.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{
		"listOfRecords": "verified", "recordWithAdt": "verified",
		"listOfRecordsWithAdt": "verified", "listOfRecordsFalse": "counterexample",
		"richDesc": "skipped",
	}
	for _, r := range report.Results {
		w, ok := want[r.Function]
		if !ok {
			continue
		}
		delete(want, r.Function)
		if r.Status != w {
			t.Errorf("%s: status %q (%s), want %q", r.Function, r.Status, r.Reason, w)
		}
		if r.Function == "richDesc" && !strings.Contains(r.Reason, "Record_desc_key have different field types") {
			t.Errorf("richDesc: reason %q does not name the collision", r.Reason)
		}
	}
	for fn := range want {
		t.Errorf("%s: no result", fn)
	}
}
