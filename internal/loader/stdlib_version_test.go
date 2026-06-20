package loader

import "testing"

func TestBaseVersion(t *testing.T) {
	cases := map[string]string{
		"v0.25.0":                      "v0.25.0",
		"v0.25.0-177-g5878c2204-dirty": "v0.25.0",
		"v0.26.0-1-gabc":               "v0.26.0",
		"dev":                          "dev",
		"":                             "",
	}
	for in, want := range cases {
		if got := baseVersion(in); got != want {
			t.Errorf("baseVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
