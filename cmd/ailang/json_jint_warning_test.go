package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TC_MATCH_005 reaches every output channel of `ailang check`, on a cold and
// on a warm compile cache.
func TestJSONIntArmWarningCheckOutputs(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "jsonjint", "warn_jnumber_only.ail")
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	for _, run := range []string{"cold", "warm"} {
		_, stderr, code := runWithStdin(t, bin, "", "check", src)
		if code != 0 || !strings.Contains(stderr, "TC_MATCH_005 at ") || !strings.Contains(stderr, "warn_jnumber_only.ail:6:") {
			t.Errorf("%s: human check: exit %d, stderr:\n%s", run, code, stderr)
		}

		stdout, _, code := runWithStdin(t, bin, "", "check", "--json", src)
		var out struct {
			Passed   bool `json:"passed"`
			Warnings []struct {
				Code       string `json:"code"`
				File       string `json:"file"`
				Line       int    `json:"line"`
				Suggestion string `json:"suggestion"`
			} `json:"warnings"`
		}
		if err := json.Unmarshal([]byte(stdout), &out); err != nil {
			t.Fatalf("%s: check --json: %v\n%s", run, err, stdout)
		}
		n := 0
		for _, w := range out.Warnings {
			if w.Code == "TC_MATCH_005" {
				n++
				if w.Line != 6 || !strings.HasSuffix(filepath.ToSlash(w.File), "warn_jnumber_only.ail") || w.Suggestion == "" {
					t.Errorf("%s: check --json warning malformed: %+v", run, w)
				}
			}
		}
		if code != 0 || !out.Passed || n != 1 {
			t.Errorf("%s: check --json: exit %d passed %v, %d TC_MATCH_005 warnings\n%s", run, code, out.Passed, n, stdout)
		}

		stdout, _, _ = runWithStdin(t, bin, "", "check", "--format", "agent", src)
		if !strings.Contains(stdout, "warn: ") || !strings.Contains(stdout, "TC_MATCH_005") {
			t.Errorf("%s: --format agent has no TC_MATCH_005 line:\n%s", run, stdout)
		}
	}
}
