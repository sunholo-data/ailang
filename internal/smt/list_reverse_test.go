package smt_test

import (
	"os"
	"path/filepath"
	"testing"
)

// std/list.reverse is uninterpreted per list sort with the facts every
// reversal has (same length; identity on lists of at most one element; the
// ends swap). Every caller used to be skipped ("unsupported application").
func TestListReverse(t *testing.T) {
	path := filepath.Join("testdata", "stdlib", "list_reverse.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{"revLen": "verified", "revShort": "verified", "revEnds": "verified", "revSame": "inconclusive", "flushClears": "verified"}
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
