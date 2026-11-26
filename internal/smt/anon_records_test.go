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
// names but not field types are named by their field sorts too; names that
// still coincide are a skip that names the clash.
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
		"richDesc": "verified", "twinTag": "skipped",
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
		if r.Function == "twinTag" && !strings.Contains(r.Reason, "two record types with different field types are both named Record_a__Seq_Int") {
			t.Errorf("twinTag: reason %q does not name the clash", r.Reason)
		}
	}
	for fn := range want {
		t.Errorf("%s: no result", fn)
	}
}
