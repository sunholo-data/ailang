package smt_test

import (
	"os"
	"path/filepath"
	"testing"
)

// std/string trim/toLower/split/chars/join/repeat/replace (uninterpreted, with
// facts the runtime result always has) and charAt (exact): every caller used to
// be skipped. A model over an uninterpreted one is inconclusive — never a
// violation by itself (the CLI's witness replay confirms real ones).
func TestStdStringExtraVerify(t *testing.T) {
	path := filepath.Join("testdata", "stdlib", "string_extra.ail")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, _ := verifySource(t, path, string(src))
	want := map[string]string{
		"t1": "verified", "lower": "verified", "parts": "verified", "cs": "verified",
		"j1": "verified", "rep": "verified", "first": "verified", "dash": "verified",
		"t1Bad": "inconclusive", "partsBad": "inconclusive", "dashBad": "inconclusive",
	}
	got := map[string]string{}
	for _, r := range report.Results {
		got[r.Function] = r.Status
	}
	for fn, w := range want {
		if got[fn] != w {
			t.Errorf("%s: status %q, want %q", fn, got[fn], w)
		}
	}
}
