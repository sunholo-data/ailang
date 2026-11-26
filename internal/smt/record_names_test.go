package smt_test

import (
	"os"
	"path/filepath"
	"testing"
)

// Two inline records with the same field names and different field sorts are
// named by their field sorts too (record_names.go). Every function here was
// skipped before ("two record types with the fields of Record_desc_key");
// the false contracts are refuted, not proved.
func TestRecordsSharingFieldNames(t *testing.T) {
	checkFixtureStatuses(t, filepath.Join("testdata", "records", "same_fields.ail"), map[string]string{
		"plainCount": "verified", "firstDesc": "verified", "scoredTotal": "verified",
		"mkPlain": "verified", "mkScored": "verified",
		"mkScoredBad": "counterexample", "firstDescBad": "counterexample",
		"literalOnlyText": "verified", "literalOnlyScore": "verified",
	})
}

// Bugs the record-name skip hid: an inline record in an alias's field was
// never kept by the sort filter (a skip), and a [] passed to a parametric
// constructor was (Seq Int) (a Z3 ERROR).
func TestRecordSortsTheCollisionHid(t *testing.T) {
	checkFixtureStatuses(t, filepath.Join("testdata", "records", "hidden.ail"), map[string]string{
		"pickTotal": "verified", "namesOk": "verified", "namesNonEmpty": "counterexample",
	})
}

func checkFixtureStatuses(t *testing.T, path string, want map[string]string) {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	for _, r := range report.Results {
		if w, ok := want[r.Function]; ok {
			delete(want, r.Function)
			if r.Status != w {
				t.Errorf("%s: status %q (%s), want %q", r.Function, r.Status, r.Reason, w)
			}
		}
	}
	for fn := range want {
		t.Errorf("%s: no result", fn)
	}
}
