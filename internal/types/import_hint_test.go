package types

import (
	"strings"
	"testing"
)

// TestImportHint (M-AGENT-ERGONOMICS) — the undefined-variable import hint formats correctly and
// stays inert when no suggester is wired (pure-library use).
func TestImportHint(t *testing.T) {
	orig := ImportSuggester
	defer func() { ImportSuggester = orig }()

	cases := []struct {
		name    string
		suggest func(string) []string
		want    string // substring expected in the hint
	}{
		{"nil suggester is inert", nil, ""},
		{"unknown symbol -> no hint", func(string) []string { return nil }, ""},
		{"single exporter", func(string) []string { return []string{"std/list"} }, "add `import std/list (nth)`"},
		{"multiple exporters", func(string) []string { return []string{"std/list", "std/string"} }, "exported by std/list, std/string"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ImportSuggester = tc.suggest
			got := importHint("nth")
			if tc.want == "" {
				if got != "" {
					t.Errorf("expected empty hint, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("hint %q does not contain %q", got, tc.want)
			}
		})
	}
}
