package fileguard

import (
	"path/filepath"
	"testing"
)

// #1559: fs_deny_write is matched case- and normalization-insensitively, so a
// case variant cannot reach a deny-listed directory on APFS/NTFS.
func TestMatchDenyWrite_CaseFolded(t *testing.T) {
	deny := []string{".claude/**", ".github/**", ".ailang/**", "Makefile", "*.yml", "./scripts/deploy.sh"}
	protected := map[string]string{
		".claude/settings.json":        ".claude/**",
		".CLAUDE/settings.json":        ".claude/**",
		".Claude/Settings.JSON":        ".claude/**",
		".cLaUdE":                      ".claude/**",
		"sub/../.CLAUDE/hooks.json":    ".claude/**",
		"./.GitHub/workflows/ci.yml":   ".github/**",
		".AILANG/cache/z":              ".ailang/**",
		"MAKEFILE":                     "Makefile",
		"makefile":                     "Makefile",
		"deep/dir/deploy.YML":          "*.yml",
		"SCRIPTS/Deploy.SH":            "./scripts/deploy.sh",
		filepath.Join(".CLAUDE", "x"):  ".claude/**",
		".ailang/state/mission-quorum": ".ailang/**",
	}
	for rel, want := range protected {
		if got := MatchDenyWrite(deny, rel); got != want {
			t.Errorf("MatchDenyWrite(%q) = %q, want %q", rel, got, want)
		}
	}
	for _, rel := range []string{"src/main.ail", ".claudex/x", "claude/x", "Makefile.am", "notes.yaml", "scripts/other.sh"} {
		if got := MatchDenyWrite(deny, rel); got != "" {
			t.Errorf("MatchDenyWrite(%q) = %q, want no match", rel, got)
		}
	}
}

// An upper-case pattern protects the lower-case spelling too: folding is
// applied to the pattern, not only to the candidate.
func TestMatchDenyWrite_PatternFolded(t *testing.T) {
	if got := MatchDenyWrite([]string{".CLAUDE/**"}, ".claude/settings.json"); got != ".CLAUDE/**" {
		t.Fatalf("got %q", got)
	}
}

// Unicode: the long s folds to s, the Kelvin sign to k, and a decomposed é
// names the same APFS entry as a precomposed one.
func TestMatchDenyWrite_UnicodeFolding(t *testing.T) {
	if got := MatchDenyWrite([]string{".claude/settings.json"}, ".claude/ſettings.json"); got == "" {
		t.Error("long-s variant of settings.json must match")
	}
	if got := MatchDenyWrite([]string{"kit/**"}, "Kit/x"); got == "" {
		t.Error("Kelvin-sign variant of kit/ must match")
	}
	if got := MatchDenyWrite([]string{"café/**"}, "CAFÉ/menu"); got == "" {
		t.Error("decomposed upper-case variant of café/ must match")
	}
}

func TestUnderGitDir_Folded(t *testing.T) {
	for _, p := range []string{".git/config", ".GIT/config", ".Git/hooks/x", "sub/.gIT", "sub/../.GIT/config", filepath.Join(".GIT", "config")} {
		if !UnderGitDir(p) {
			t.Errorf("%s must be under .git", p)
		}
	}
	for _, p := range []string{".gitignore", ".GITATTRIBUTES", "git/config", "a.git/x", ".git.bak/x"} {
		if UnderGitDir(p) {
			t.Errorf("%s is an ordinary path", p)
		}
	}
}

func TestProtection_Check(t *testing.T) {
	p := Protection{GitDir: true, DenyWrite: []string{".claude/**"}}
	if v := p.Check(".GIT/config"); !v.GitDir || !v.Protected() {
		t.Errorf(".GIT/config: %+v", v)
	}
	if v := p.Check(".CLAUDE/settings.json"); v.Pattern != ".claude/**" {
		t.Errorf(".CLAUDE/settings.json: %+v", v)
	}
	if v := p.Check("src/x.ail"); v.Protected() {
		t.Errorf("src/x.ail: %+v", v)
	}
	if v := (Protection{DenyWrite: []string{"x"}}).Check(".git/config"); v.Protected() {
		t.Errorf("GitDir off: .git is not protected by Check: %+v", v)
	}
}
