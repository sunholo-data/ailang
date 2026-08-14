package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

const testPackageManifest = `[package]
name = "sunholo/upsert_test"
version = "0.1.0"
edition = "1"

[dependencies]
`

func writeTestManifest(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, pkg.ManifestFile), []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func readTestManifest(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, pkg.ManifestFile))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return string(data)
}

func countDependencyKey(content, name string) int {
	count := 0
	inDependencies := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inDependencies = trimmed == "[dependencies]"
			continue
		}
		key := strings.TrimSpace(strings.SplitN(trimmed, "=", 2)[0])
		key = strings.Trim(key, `"`)
		if inDependencies && key == name {
			count++
		}
	}
	return count
}

func TestAppendDependencyToFile_IdempotentUpsert(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, testPackageManifest)

	if err := appendDependencyToFile(dir, "sunholo/gemini_files", "0.2.0", false); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := appendDependencyToFile(dir, "sunholo/gemini_files", "0.2.0", false); err != nil {
		t.Fatalf("second append: %v", err)
	}

	content := readTestManifest(t, dir)
	if got := countDependencyKey(content, "sunholo/gemini_files"); got != 1 {
		t.Fatalf("dependency key count = %d, want exactly 1\n%s", got, content)
	}
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("updated manifest must parse: %v", err)
	}
	if got := manifest.Dependencies["sunholo/gemini_files"].Version; got != "0.2.0" {
		t.Fatalf("dependency version = %q, want second value %q", got, "0.2.0")
	}
}

func TestAppendDependencyToFile_UpgradePreservesPosition(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, testPackageManifest+"\"sunholo/first\" = \"1.0.0\"\n   sunholo/gemini_files = \"0.1.0\"\n\"sunholo/last\" = \"1.0.0\"\n")

	if err := appendDependencyToFile(dir, "sunholo/gemini_files", "0.2.0", false); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	content := readTestManifest(t, dir)
	if got := countDependencyKey(content, "sunholo/gemini_files"); got != 1 {
		t.Fatalf("dependency key count = %d, want exactly 1\n%s", got, content)
	}
	first := strings.Index(content, `"sunholo/first"`)
	updated := strings.Index(content, `"sunholo/gemini_files" = "0.2.0"`)
	last := strings.Index(content, `"sunholo/last"`)
	if !(first < updated && updated < last) {
		t.Fatalf("upgraded dependency did not preserve position\n%s", content)
	}
}

func TestAppendGitDependencyToFile_IdempotentUpsert(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, testPackageManifest)
	name := "sunholo/git_dep"

	if err := appendGitDependencyToFile(dir, name, []string{`git = "https://example.test/repo"`, `tag = "v0.1.0"`}, "https://example.test/repo", "v0.1.0", ""); err != nil {
		t.Fatalf("first git append: %v", err)
	}
	if err := appendGitDependencyToFile(dir, name, []string{`git = "https://example.test/repo"`, `tag = "v0.2.0"`}, "https://example.test/repo", "v0.2.0", ""); err != nil {
		t.Fatalf("second git append: %v", err)
	}

	content := readTestManifest(t, dir)
	if got := countDependencyKey(content, name); got != 1 {
		t.Fatalf("git dependency key count = %d, want exactly 1\n%s", got, content)
	}
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("updated git manifest must parse: %v", err)
	}
	if got := manifest.Dependencies[name].Tag; got != "v0.2.0" {
		t.Fatalf("git dependency tag = %q, want %q", got, "v0.2.0")
	}
}

func TestAppendDependencyToFile_DoesNotMatchPrefix(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, testPackageManifest+"\"sunholo/gemini_files_extra\" = \"9.9.9\"\n")

	if err := appendDependencyToFile(dir, "sunholo/gemini_files", "0.2.0", false); err != nil {
		t.Fatalf("append: %v", err)
	}

	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("manifest must parse: %v", err)
	}
	if got := manifest.Dependencies["sunholo/gemini_files_extra"].Version; got != "9.9.9" {
		t.Fatalf("prefix dependency changed to %q", got)
	}
	if got := manifest.Dependencies["sunholo/gemini_files"].Version; got != "0.2.0" {
		t.Fatalf("new dependency = %q, want %q", got, "0.2.0")
	}
}

func TestAppendDependencyToFile_OtherTableUntouched(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, testPackageManifest+"\n[metadata]\n\"sunholo/gemini_files\" = \"metadata-value\"\n")

	if err := appendDependencyToFile(dir, "sunholo/gemini_files", "0.2.0", false); err != nil {
		t.Fatalf("append: %v", err)
	}

	content := readTestManifest(t, dir)
	if !strings.Contains(content, `[metadata]
"sunholo/gemini_files" = "metadata-value"`) {
		t.Fatalf("same-named key in metadata table was modified\n%s", content)
	}
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("manifest must parse: %v", err)
	}
	if got := manifest.Dependencies["sunholo/gemini_files"].Version; got != "0.2.0" {
		t.Fatalf("dependency = %q, want %q", got, "0.2.0")
	}
}

// TestAppendDependencyToFile_NewSectionKeepsBlankLineSeparator pins the formatting
// of a freshly-created [dependencies] section. Before the upsert rewrite every
// generated manifest carried a blank line ahead of the new header; the first
// rewrite dropped it, producing a header flush against the previous key. That is
// valid TOML, so no parse-based assertion can see it — this arm reads the two
// lines around the header directly.
//
// Killing mutation: in upsertDependencyLine's final return, set prefix to "" when
// content already ends in a newline.
func TestAppendDependencyToFile_NewSectionKeepsBlankLineSeparator(t *testing.T) {
	dir := t.TempDir()
	const noDepsManifest = `[package]
name = "sunholo/upsert_test"
version = "0.1.0"
edition = "1"

[stability]
level = "experimental"
`
	writeTestManifest(t, dir, noDepsManifest)

	if err := appendDependencyToFile(dir, "sunholo/gemini_files", "0.2.1", false); err != nil {
		t.Fatalf("appendDependencyToFile: %v", err)
	}

	content := readTestManifest(t, dir)
	lines := strings.Split(content, "\n")

	header := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "[dependencies]" {
			header = i
			break
		}
	}
	if header < 0 {
		t.Fatalf("instrument failure: no [dependencies] header in:\n%s", content)
	}
	// Known-positive control: the pre-existing [stability] header keeps its own
	// blank-line separator, so a failure below is about the new section, not
	// about the fixture.
	stability := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "[stability]" {
			stability = i
			break
		}
	}
	if stability < 1 || strings.TrimSpace(lines[stability-1]) != "" {
		t.Fatalf("instrument failure: fixture's [stability] header is not blank-line separated")
	}

	if header < 1 || strings.TrimSpace(lines[header-1]) != "" {
		t.Errorf("new [dependencies] header is not preceded by a blank line; got:\n%s", content)
	}
	if got := countDependencyKey(content, "sunholo/gemini_files"); got != 1 {
		t.Errorf("dependency key count = %d, want 1", got)
	}
	if _, err := pkg.LoadManifest(dir); err != nil {
		t.Errorf("manifest does not parse after upsert: %v", err)
	}
}

// TestAppendDependencyToFile_EarlierTableUntouched covers the direction
// TestAppendDependencyToFile_OtherTableUntouched cannot reach. That arm places the
// rival table AFTER [dependencies], where the scan's break-on-next-section already
// stops it — so it passes even when the in-section guard is removed (measured: the
// `inDependencies &&` mutant survives it). Every real ailang.toml puts [package],
// [exports], [effects] and [stability] BEFORE [dependencies], so the untested
// direction is the common one.
//
// Killing mutation: drop the `inDependencies &&` conjunct from the match in
// upsertDependencyLine.
func TestAppendDependencyToFile_EarlierTableUntouched(t *testing.T) {
	dir := t.TempDir()
	const earlierTableManifest = `[package]
name = "sunholo/upsert_test"
version = "0.1.0"
edition = "1"

[metadata]
"sunholo/gemini_files" = "metadata-value"

[dependencies]
`
	writeTestManifest(t, dir, earlierTableManifest)

	if err := appendDependencyToFile(dir, "sunholo/gemini_files", "0.2.0", false); err != nil {
		t.Fatalf("append: %v", err)
	}

	content := readTestManifest(t, dir)
	if !strings.Contains(content, "[metadata]\n\"sunholo/gemini_files\" = \"metadata-value\"") {
		t.Errorf("same-named key in an EARLIER table was modified\n%s", content)
	}
	if got := countDependencyKey(content, "sunholo/gemini_files"); got != 1 {
		t.Errorf("dependency key count = %d, want 1\n%s", got, content)
	}
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("manifest must parse: %v", err)
	}
	if got := manifest.Dependencies["sunholo/gemini_files"].Version; got != "0.2.0" {
		t.Errorf("dependency = %q, want %q", got, "0.2.0")
	}
}
