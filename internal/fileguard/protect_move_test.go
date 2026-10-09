package fileguard

import "testing"

func TestProtectionCheckMove(t *testing.T) {
	cases := []struct {
		pattern, rel     string
		denied, ancestor bool
	}{
		{".claude/settings.json", ".claude", true, true},
		{".claude/settings.json", ".claude/settings.json", true, false},
		{"a/b/**", "a", true, true}, {"a/b/**", "a/b", true, false},
		{"a/*/x.txt", "a/b", true, true}, {"a/*/x.txt", "a", true, true},
		{"a/?/[xy].txt", "a/b", true, true},
		{"./.ClAuDe/settings.json", "x/../.claude", true, true},
		{"café/x", "cafe\u0301", true, true},
		{"a/**/x", "a/b/c", true, true}, {"a/[/x", "a/b", true, true},
		{"a/b/x", "unrelated", false, false},
		{".claude/settings.json", ".claude/sub", false, false},
		{"*.yml", "src", false, false}, {"Makefile", "src", false, false},
		{"*.yml", "src/file.yml", true, false},
		{"a/b/x", ".", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.pattern+":"+tc.rel, func(t *testing.T) {
			p := Protection{DenyWrite: []string{tc.pattern}}
			v := p.CheckMove(tc.rel)
			if v.Protected() != tc.denied || v.Ancestor != tc.ancestor {
				t.Fatalf("CheckMove = %+v, want denied=%v ancestor=%v", v, tc.denied, tc.ancestor)
			}
			if tc.denied && v.Pattern != tc.pattern {
				t.Fatalf("pattern = %q", v.Pattern)
			}
			if tc.ancestor && p.Check(tc.rel).Protected() {
				t.Fatal("plain write check changed")
			}
		})
	}
	v := (Protection{GitDir: true, DenyWrite: []string{".git/config"}}).CheckMove(".GIT")
	if !v.GitDir || v.Ancestor || v.Pattern != "" {
		t.Fatalf("git precedence: %+v", v)
	}
	if got := MatchDenyMove([]string{"a/x", "a/y"}, "a"); got != "a/x" {
		t.Fatalf("first pattern = %q", got)
	}
}
