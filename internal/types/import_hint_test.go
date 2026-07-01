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

// TestCollisionHint (Felix fb_942b7f) — a callee exported by >1 stdlib module gets a
// name-collision + alias note on a failing application; single-exporter and self-only
// cases stay silent so correct code is never annotated.
func TestCollisionHint(t *testing.T) {
	orig := ImportSuggester
	defer func() { ImportSuggester = orig }()

	cases := []struct {
		name       string
		suggest    func(string) []string
		symbol     string
		from       string
		wantSubstr string // "" means expect empty
	}{
		{"nil suggester inert", nil, "length", "std/list", ""},
		{"single exporter -> no collision", func(string) []string { return []string{"std/list"} }, "length", "std/list", ""},
		{"self-only -> no collision", func(string) []string { return []string{"std/list"} }, "length", "std/list", ""},
		{"collision names the others + alias", func(string) []string { return []string{"std/array", "std/list", "std/string"} }, "length", "std/list", "std/array, std/string"},
		{"collision suggests aliasing", func(string) []string { return []string{"std/list", "std/string"} }, "length", "std/list", "length as lengthAlt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ImportSuggester = tc.suggest
			got := collisionHint(tc.symbol, tc.from)
			if tc.wantSubstr == "" {
				if got != "" {
					t.Errorf("expected empty, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.wantSubstr) {
				t.Errorf("hint %q does not contain %q", got, tc.wantSubstr)
			}
			// The module the callee resolved FROM must never appear in the "others".
			if strings.Contains(got, "std/list") && tc.from == "std/list" {
				t.Errorf("hint should exclude the resolved module std/list, got %q", got)
			}
		})
	}
}
