package main

import "testing"

// M-EVAL-VERSION-BANKING: --bank-by-version must bucket by RELEASE, not by every dev build, so a
// dev `make install` does NOT trigger a fresh rotation sweep — only a new release tag does.
func TestReleaseTag(t *testing.T) {
	cases := map[string]string{
		"v0.26.0-26-g9249a66bf":       "v0.26.0",     // dev build -> its release
		"v0.26.0-26-g9249a66bf-dirty": "v0.26.0",     // dirty dev build -> its release
		"v0.27.0-1-gabcdef0":          "v0.27.0",     // one commit past a release -> the release
		"v0.26.0-rc1-5-gabc1234":      "v0.26.0-rc1", // pre-release tags are preserved
		"v0.26.0":                     "v0.26.0",     // clean release unchanged
		"dev":                         "dev",         // non-ldflags build unchanged
	}
	for in, want := range cases {
		if got := releaseTag(in); got != want {
			t.Errorf("releaseTag(%q) = %q, want %q", in, got, want)
		}
	}
}
