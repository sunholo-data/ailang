package eval_harness

import (
	"path/filepath"
	"testing"
)

func TestVersionFromPath(t *testing.T) {
	cases := map[string]string{
		filepath.FromSlash("eval_results/rotation/os-rolling/v0.52.3/agent/x.json"): "v0.52.3",
		filepath.FromSlash("eval_results/baselines/0.3.14/x.json"):                  "v0.3.14",
		filepath.FromSlash("eval_results/baselines/v0.26.0-rc1/x.json"):             "v0.26.0-rc1",
		filepath.FromSlash("eval_results/rotation/cloud-rolling/x.json"):            "",
	}
	for in, want := range cases {
		if got := versionFromPath(in); got != want {
			t.Errorf("versionFromPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReleaseTag(t *testing.T) {
	for in, want := range map[string]string{
		"v0.52.3-44-gc92739681":       "v0.52.3",
		"v0.26.0-26-g9249a66bf-dirty": "v0.26.0",
		"v0.26.0-rc1-5-gabc1234":      "v0.26.0-rc1",
		"dev":                         "dev",
	} {
		if got := ReleaseTag(in); got != want {
			t.Errorf("ReleaseTag(%q) = %q, want %q", in, got, want)
		}
	}
}
