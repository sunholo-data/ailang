package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChangelogSectionFrom_HeadingForms(t *testing.T) {
	content := "# Changelog\n\n## [0.9.0] - 2026-09-20\n\n- future\n\n## v0.8.2 — 2026-09-17\n\n- token refresh no longer logs the refresh token\n- second line\n\n## 0.8.1\n\n- older\n"
	for _, version := range []string{"0.8.2", "0.9.0", "0.8.1"} {
		body, ok := ChangelogSectionFrom(content, version)
		if !ok || body == "" {
			t.Errorf("version %s: not found (ok=%v body=%q)", version, ok, body)
		}
	}
	body, _ := ChangelogSectionFrom(content, "0.8.2")
	if !strings.Contains(body, "refresh token") || strings.Contains(body, "older") || strings.Contains(body, "future") {
		t.Errorf("section body bleeds across headings: %q", body)
	}
	if _, ok := ChangelogSectionFrom(content, "0.7.0"); ok {
		t.Error("absent version reported present")
	}
	if _, ok := ChangelogSectionFrom("## 0.1.0\n\n\n## 0.0.9\n- x\n", "0.1.0"); ok {
		t.Error("EMPTY section must not count as a description")
	}
	long := "## 1.0.0\n" + strings.Repeat("x", ChangelogNotesLimit+100)
	if body, ok := ChangelogSectionFrom(long, "1.0.0"); !ok || len(body) != ChangelogNotesLimit {
		t.Errorf("notes not capped: ok=%v len=%d", ok, len(body))
	}
}

func TestChangelogSection_MissingFile(t *testing.T) {
	if _, ok := ChangelogSection(t.TempDir(), "0.1.0"); ok {
		t.Error("missing CHANGELOG.md reported as present")
	}
}

// Grace window (design M4): badges before ReleaseGateHardFrom, gates from it.
// A version with no semver token ("dev") is inside the window.
func TestReleaseGatesHard_Window(t *testing.T) {
	cases := map[string]bool{
		"v0.40.0":                              false,
		"0.40.9-12-gabc-dirty":                 false,
		"AILANG v0.39.4-11-g1f78cd2bc\nCommit": false,
		"dev":                                  false,
		"unknown":                              false,
		"v0.41.0":                              true,
		"AILANG v0.41.3-2-gdef\nCommit: x":     true,
		"v1.0.0":                               true,
	}
	for in, want := range cases {
		if got := ReleaseGatesHard(in); got != want {
			t.Errorf("ReleaseGatesHard(%q) = %v, want %v", in, got, want)
		}
	}
	if ReleaseGateHardFrom != "0.41.0" {
		t.Errorf("grace window constant moved without a design change: %s", ReleaseGateHardFrom)
	}
}

func TestValidateReleaseKind(t *testing.T) {
	for _, k := range append([]string{""}, ReleaseKinds...) {
		if err := ValidateReleaseKind(k); err != nil {
			t.Errorf("kind %q rejected: %v", k, err)
		}
	}
	if err := ValidateReleaseKind("hotfix"); err == nil {
		t.Error("unknown kind accepted")
	}
}

// [release] round-trips through the manifest and is rejected when garbage.
func TestManifest_ReleaseSection(t *testing.T) {
	dir := t.TempDir()
	base := "[package]\nname = \"test/rel\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[exports]\nmodules = [\"test/rel/core\"]\n\n[release]\nkind = %q\n"
	writeFile(t, filepath.Join(dir, ManifestFile), strings.Replace(base, "%q", `"security"`, 1))
	m, err := LoadManifest(dir)
	if err != nil || m.Release.Kind != "security" {
		t.Fatalf("LoadManifest: %v kind=%q", err, m.Release.Kind)
	}
	writeFile(t, filepath.Join(dir, ManifestFile), strings.Replace(base, "%q", `"hotfix"`, 1))
	if _, err := LoadManifest(dir); err == nil || !strings.Contains(err.Error(), "[release].kind") {
		t.Errorf("garbage kind must be refused at load: %v", err)
	}
}

// PUB001/PUB002 route to gates or badges by the grace flag (design D2/M4).
func TestBuildQualityReport_ReleaseGatesFollowGraceWindow(t *testing.T) {
	m := qualityManifest("experimental", []string{})
	in := cleanInputs() // no changelog, no kind
	grace := BuildQualityReport(m, ModeServer, in, false)
	if !hasCode(grace.Badges, "PUB001") || !hasCode(grace.Badges, "PUB002") || hasCode(grace.Gates, "PUB001") {
		t.Errorf("grace: want PUB001/PUB002 badges, got gates=%v badges=%v", grace.Gates, grace.Badges)
	}
	in.ReleaseGatesHard = true
	hard := BuildQualityReport(m, ModeServer, in, false)
	if !hasCode(hard.Gates, "PUB001") || !hasCode(hard.Gates, "PUB002") {
		t.Errorf("hard: want PUB001/PUB002 gates, got %v", hard.Gates)
	}
	m.Release.Kind = "fix"
	in.HasChangelogSection, in.ChangelogNotes = true, "- fixed"
	clean := BuildQualityReport(m, ModeServer, in, false)
	if hasCode(clean.Gates, "PUB001") || hasCode(clean.Gates, "PUB002") || clean.Release.Kind != "fix" || clean.Release.Notes != "- fixed" {
		t.Errorf("declared release still flagged: gates=%v release=%+v", clean.Gates, clean.Release)
	}
}

// The tarball must ship CHANGELOG.md, and `ailang init package` must scaffold
// both artifacts so a fresh package passes with zero release findings.
func TestTarballShipsChangelog_AndInitScaffoldsRelease(t *testing.T) {
	dir := t.TempDir()
	if err := InitManifest(dir, "test/scaffold", "v0.40.0"); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Release.Kind == "" {
		t.Error("init package must declare [release] kind")
	}
	if _, ok := ChangelogSection(dir, "0.1.0"); !ok {
		t.Error("init package must scaffold a non-empty `## 0.1.0` changelog section")
	}
	writeFile(t, filepath.Join(dir, "core.ail"), "module test/scaffold/core\nexport pure func one() -> int = 1\n")
	tar, err := CreateTarball(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := ExtractTarball(tar, out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, ChangelogFile)); err != nil {
		t.Errorf("CHANGELOG.md not in tarball: %v", err)
	}
}
