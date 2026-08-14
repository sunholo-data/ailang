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

// countDependencyKey counts keys named `name` inside [dependencies].
//
// It deliberately does NOT call the production stripLineComment/dependencyLineKey:
// an instrument that shares the code under test's blind spots cannot detect them.
// It grew its own comment handling because the first version compared the raw
// line against "[dependencies]" and so reported 0 for the perfectly-correct
// output of a manifest whose header carried a trailing comment — the instrument
// failing in exactly the shape the arm was added to catch.
//
// It is not multi-line-string aware; arms that need that assert through
// pkg.LoadManifest instead.
func countDependencyKey(content, name string) int {
	count := 0
	inDependencies := false
	for _, line := range strings.Split(content, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 && strings.Count(line[:i], `"`)%2 == 0 {
			line = line[:i]
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inDependencies = trimmed == "[dependencies]"
			continue
		}
		key := strings.TrimSpace(strings.SplitN(trimmed, "=", 2)[0])
		key = strings.Trim(key, `"'`)
		if inDependencies && key == name {
			count++
		}
	}
	return count
}

func countDependenciesTables(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "[dependencies]" || line == `["dependencies"]` || line == "['dependencies']" {
			count++
		}
	}
	return count
}

func countNamedKeyOccurrences(content, name string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		key, _, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && strings.Trim(strings.TrimSpace(key), `"'`) == name {
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

// The four arms below each pin a shape the first line-scanner could not see.
// All four were found by the iteration-202 evaluator, reproduced first-party
// against the real binary, and are TOML-legal — none needs adversarial intent.

// B1: `[dependencies]` as the file's last line with no newline after it. The
// insert glued the key onto the header (`[dependencies]"x" = "1"`), which does
// not parse.
// Killing mutation: drop the `if !strings.HasSuffix(header, "\n")` repair.
func TestAppendDependencyToFile_HeaderIsLastLineWithoutNewline(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[dependencies]")

	if err := appendDependencyToFile(dir, "sunholo/newdep", "0.5.0", false); err != nil {
		t.Fatalf("append: %v", err)
	}

	content := readTestManifest(t, dir)
	if strings.Contains(content, `[dependencies]"`) {
		t.Errorf("key glued onto the section header:\n%s", content)
	}
	if _, err := pkg.LoadManifest(dir); err != nil {
		t.Errorf("manifest does not parse after upsert: %v\n%s", err, content)
	}
}

// B2: a header carrying a trailing comment. The exact-string compare did not
// recognise it, so a SECOND [dependencies] table was appended — reproducing
// #718's own failure ("Key 'dependencies' has already been defined"). This is a
// regression the upsert rewrite introduced: the pre-rewrite substring match
// handled it.
// Killing mutation: compare against the raw line instead of stripLineComment's
// output in upsertDependencyLine.
func TestAppendDependencyToFile_HeaderWithTrailingComment(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[dependencies] # deps go here\n\"sunholo/existing\" = \"1.0.0\"\n")

	if err := appendDependencyToFile(dir, "sunholo/existing", "2.0.0", false); err != nil {
		t.Fatalf("append: %v", err)
	}

	content := readTestManifest(t, dir)
	if got := strings.Count(content, "[dependencies]"); got != 1 {
		t.Errorf("[dependencies] table count = %d, want 1\n%s", got, content)
	}
	if got := countDependencyKey(content, "sunholo/existing"); got != 1 {
		t.Errorf("key count = %d, want 1\n%s", got, content)
	}
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("manifest does not parse: %v\n%s", err, content)
	}
	if got := manifest.Dependencies["sunholo/existing"].Version; got != "2.0.0" {
		t.Errorf("version = %q, want 2.0.0", got)
	}
}

// B3: a multi-line string whose content contains a line that looks like a
// section header. The scanner broke out of [dependencies] at that line and
// never saw the real key below it, then inserted a duplicate and printed
// "Added" over a corrupted file.
// Killing mutation: remove the openString tracking in upsertDependencyLine.
func TestAppendDependencyToFile_MultilineStringIsNotStructure(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[dependencies]\nnotes = \"\"\"\n[dependencies.fake]\njust prose inside a string\n\"\"\"\n\"sunholo/existing\" = \"1.0.0\"\n")

	if err := appendDependencyToFile(dir, "sunholo/existing", "2.0.0", false); err != nil {
		t.Fatalf("append: %v", err)
	}

	content := readTestManifest(t, dir)
	// countDependencyKey shares the scanner's old blind spot — it also treats the
	// string's `[dependencies.fake]` line as a section header — so this arm counts
	// the raw key occurrence and defers table membership to the real parser.
	if got := strings.Count(content, `"sunholo/existing" =`); got != 1 {
		t.Errorf("key occurrence count = %d, want 1\n%s", got, content)
	}
	if !strings.Contains(content, "[dependencies.fake]") {
		t.Errorf("multi-line string content was rewritten\n%s", content)
	}
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("manifest does not parse after upsert: %v\n%s", err, content)
	}
	if got := manifest.Dependencies["sunholo/existing"].Version; got != "2.0.0" {
		t.Errorf("version = %q, want 2.0.0\n%s", got, content)
	}
}

// B4: a literal-quoted (single-quote) key. dependencyLineKey handled only the
// basic form, so the existing entry was never found and a duplicate followed.
// Killing mutation: drop '\” from dependencyLineKey's quote loop.
func TestAppendDependencyToFile_LiteralQuotedKey(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[dependencies]\n'sunholo/existing' = \"1.0.0\"\n")

	// Control: the fixture is valid TOML before we touch it, so a later parse
	// failure is about the upsert and not about the fixture.
	if _, err := pkg.LoadManifest(dir); err != nil {
		t.Fatalf("instrument failure: fixture does not parse: %v", err)
	}

	if err := appendDependencyToFile(dir, "sunholo/existing", "2.0.0", false); err != nil {
		t.Fatalf("append: %v", err)
	}

	content := readTestManifest(t, dir)
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("manifest does not parse after upsert: %v\n%s", err, content)
	}
	if got := manifest.Dependencies["sunholo/existing"].Version; got != "2.0.0" {
		t.Errorf("version = %q, want 2.0.0\n%s", got, content)
	}
}

// A CRLF manifest must not gain a lone LF mid-file.
// Killing mutation: hardcode ending to "\n" in the replacement branch.
func TestAppendDependencyToFile_PreservesCRLF(t *testing.T) {
	dir := t.TempDir()
	writeTestManifest(t, dir, "[package]\r\nname = \"sunholo/upsert_test\"\r\nversion = \"0.1.0\"\r\nedition = \"1\"\r\n\r\n[dependencies]\r\n\"sunholo/existing\" = \"1.0.0\"\r\n")

	if err := appendDependencyToFile(dir, "sunholo/existing", "2.0.0", false); err != nil {
		t.Fatalf("append: %v", err)
	}

	content := readTestManifest(t, dir)
	if strings.Count(content, "\n") != strings.Count(content, "\r\n") {
		t.Errorf("mixed line endings after upsert: %d LF vs %d CRLF\n%q",
			strings.Count(content, "\n"), strings.Count(content, "\r\n"), content)
	}
}

// writeManifestChecked is the structural bound on every blind spot the scanner
// still has: a write that would make a previously-parseable manifest
// unparseable is rolled back and reported, never left on disk.
// Killing mutation: drop the post-write LoadManifest re-check.
func TestWriteManifestChecked_RollsBackACorruptingWrite(t *testing.T) {
	dir := t.TempDir()
	const good = "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\n"
	writeTestManifest(t, dir, good)
	path := filepath.Join(dir, pkg.ManifestFile)

	// Control: the guard passes a write that keeps the file parseable.
	okContent := good + "\n[dependencies]\n\"sunholo/existing\" = \"1.0.0\"\n"
	if err := writeManifestChecked(dir, path, []byte(good), okContent); err != nil {
		t.Fatalf("instrument failure: a valid write was rejected: %v", err)
	}
	if readTestManifest(t, dir) != okContent {
		t.Fatalf("instrument failure: a valid write did not land")
	}

	before := readTestManifest(t, dir)
	err := writeManifestChecked(dir, path, []byte(before), "[package]\nname = = = broken\n")
	if err == nil {
		t.Fatal("corrupting write was accepted")
	}
	if !strings.Contains(err.Error(), "unparseable") {
		t.Errorf("error does not say what happened: %v", err)
	}
	if got := readTestManifest(t, dir); got != before {
		t.Errorf("file was not restored\nwant:\n%s\ngot:\n%s", before, got)
	}
}

// Killing mutation: replace pkg.ParseManifestFile with pkg.LoadManifest in writeManifestChecked.
func TestWriteManifestChecked_ParseabilityNotValidity(t *testing.T) {
	const withoutEdition = "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nnotes = 'contains \"\"\" triple double quotes'\n\n[dependencies]\n\"sunholo/existing\" = \"1.0.0\"\n"
	const withEdition = "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\nnotes = 'contains \"\"\" triple double quotes'\n\n[dependencies]\n\"sunholo/existing\" = \"1.0.0\"\n"

	for _, tc := range []struct {
		name     string
		original string
	}{
		{name: "validation-invalid missing edition", original: withoutEdition},
		{name: "validation-valid control", original: withEdition},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTestManifest(t, dir, tc.original)
			path := filepath.Join(dir, pkg.ManifestFile)
			corrupt := tc.original + "\n[dependencies]\n\"sunholo/existing\" = \"2.0.0\"\n"

			err := writeManifestChecked(dir, path, []byte(tc.original), corrupt)
			if err == nil {
				t.Fatal("corrupting write was accepted")
			}
			if !strings.Contains(err.Error(), "unparseable") {
				t.Errorf("error does not identify unparseable result: %v", err)
			}
			if got := readTestManifest(t, dir); got != tc.original {
				t.Errorf("file changed despite rollback\nwant:\n%s\ngot:\n%s", tc.original, got)
			}
		})
	}
}

// Killing mutation: force parsedBefore=true in writeManifestChecked.
func TestWriteManifestChecked_AllowsWriteOverUnparseableInput(t *testing.T) {
	dir := t.TempDir()
	const broken = "[package]\nname = = = broken\n"
	const replacement = "[package]\nname = = = still-broken\n"
	writeTestManifest(t, dir, broken)
	path := filepath.Join(dir, pkg.ManifestFile)

	if err := writeManifestChecked(dir, path, []byte(broken), replacement); err != nil {
		t.Fatalf("already-unparseable manifest should not hold the write: %v", err)
	}
	if got := readTestManifest(t, dir); got != replacement {
		t.Errorf("write did not land\nwant:\n%s\ngot:\n%s", replacement, got)
	}
}

// Killing mutation: restore openMultilineString's strings.Count(delimiter)%2 scanner.
func TestUpsertDependencyLine_TripleQuoteInsideSingleLineString(t *testing.T) {
	const original = "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\nnotes = 'contains \"\"\" triple double quotes'\n\n[dependencies]\n\"sunholo/existing\" = \"1.0.0\"\n"
	updated, _, replaced := upsertDependencyLine(original, "sunholo/existing", `"sunholo/existing" = "2.0.0"`)
	if !replaced {
		t.Fatal("existing dependency was not replaced")
	}
	dir := t.TempDir()
	writeTestManifest(t, dir, updated)
	if err := pkg.ParseManifestFile(filepath.Join(dir, pkg.ManifestFile)); err != nil {
		t.Fatalf("updated manifest does not parse: %v\n%s", err, updated)
	}
	if got := countDependenciesTables(updated); got != 1 {
		t.Fatalf("dependencies table count = %d, want 1\n%s", got, updated)
	}
}

// Killing mutation: restore the dependencies-header check to trimmed == "[dependencies]".
func TestUpsertDependencyLine_QuotedDependenciesHeader(t *testing.T) {
	const original = "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[\"dependencies\"]\n\"sunholo/existing\" = \"1.0.0\"\n"
	updated, _, replaced := upsertDependencyLine(original, "sunholo/existing", `"sunholo/existing" = "2.0.0"`)
	if !replaced {
		t.Fatal("existing dependency was not replaced")
	}
	dir := t.TempDir()
	writeTestManifest(t, dir, updated)
	if err := pkg.ParseManifestFile(filepath.Join(dir, pkg.ManifestFile)); err != nil {
		t.Fatalf("updated manifest does not parse: %v\n%s", err, updated)
	}
	if got := countDependenciesTables(updated); got != 1 {
		t.Fatalf("dependencies table count = %d, want 1\n%s", got, updated)
	}
}

// Killing mutation: make tableHeaderName accept a [[-prefixed array-of-tables header.
func TestTableHeaderName_RejectsArrayOfTables(t *testing.T) {
	if name, ok := tableHeaderName("[[dependencies]]"); ok || name != "" {
		t.Fatalf("array-of-tables accepted as simple header: name=%q ok=%v", name, ok)
	}
}

// Killing mutation: restore openMultilineString's strings.Count(delimiter)%2 scanner.
func TestOpenMultilineString_QuotedDelimiterCorpus(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{line: `notes = """`, want: `"""`},
		{line: `text = """abc`, want: `"""`},
		{line: `notes = 'contains """ triple double quotes'`, want: ""},
		{line: `notes = "he said \"\"\" once"`, want: ""},
	}
	for _, tc := range tests {
		if got := openMultilineString(tc.line); got != tc.want {
			t.Errorf("openMultilineString(%q) = %q, want %q", tc.line, got, tc.want)
		}
	}
}

// Killing mutation: restore either the delimiter-count scanner or literal header comparison.
func TestAppendDependencyToFile_ManifestShapeCorpus(t *testing.T) {
	const base = "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\nedition = \"1\"\n"
	const dep = "sunholo/gemini_files"
	rows := []struct {
		name     string
		manifest string
	}{
		{name: "no dependencies", manifest: base},
		{name: "different dependency", manifest: base + "\n[dependencies]\n\"sunholo/other\" = \"1.0.0\"\n"},
		{name: "same dependency older", manifest: base + "\n[dependencies]\n\"" + dep + "\" = \"0.1.0\"\n"},
		{name: "header trailing comment", manifest: base + "\n[dependencies] # managed\n"},
		{name: "header final line", manifest: base + "\n[dependencies]"},
		{name: "multiline string with header-shaped line", manifest: base + "description = \"\"\"prose\n[not-a-table]\nmore prose\"\"\"\n\n[dependencies]\n"},
		{name: "literal triple double quote", manifest: base + "notes = 'contains \"\"\" triple double quotes'\n\n[dependencies]\n"},
		{name: "quoted dependencies header", manifest: base + "\n[\"dependencies\"]\n"},
		{name: "literal quoted key", manifest: base + "\n[dependencies]\n'" + dep + "' = \"0.1.0\"\n"},
		{name: "CRLF", manifest: strings.ReplaceAll(base+"\n[dependencies]\n", "\n", "\r\n")},
		{name: "followed by another table", manifest: base + "\n[dependencies]\n\n[metadata]\nowner = \"test\"\n"},
		{name: "missing edition", manifest: "[package]\nname = \"sunholo/upsert_test\"\nversion = \"0.1.0\"\n\n[dependencies]\n"},
	}
	if len(rows) < 12 {
		t.Fatalf("corpus shrank to %d rows; want at least 12", len(rows))
	}
	run := 0
	for _, tc := range rows {
		t.Run(tc.name, func(t *testing.T) {
			run++
			dir := t.TempDir()
			writeTestManifest(t, dir, tc.manifest)
			for i := 0; i < 2; i++ {
				if err := appendDependencyToFile(dir, dep, "0.2.0", false); err != nil {
					t.Fatalf("install %d: %v", i+1, err)
				}
			}
			content := readTestManifest(t, dir)
			if err := pkg.ParseManifestFile(filepath.Join(dir, pkg.ManifestFile)); err != nil {
				t.Errorf("manifest does not parse: %v\n%s", err, content)
			}
			if got := countNamedKeyOccurrences(content, dep); got != 1 {
				t.Errorf("dependency key count = %d, want 1\n%s", got, content)
			}
			if got := countDependenciesTables(content); got != 1 {
				t.Errorf("dependencies table count = %d, want 1\n%s", got, content)
			}
		})
	}
	if run != len(rows) {
		t.Fatalf("ran %d corpus rows, want %d", run, len(rows))
	}
}
