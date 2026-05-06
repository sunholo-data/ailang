package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/loader"
)

// makeLoadedModule builds a minimal LoadedModule with a declared module header
// suitable for feeding detectModulePathCollisions.
func makeLoadedModule(canonicalID, filePath, declared string) *loader.LoadedModule {
	file := &ast.File{
		Path: filePath,
	}
	if declared != "" {
		file.Module = &ast.ModuleDecl{Path: declared}
	}
	return &loader.LoadedModule{
		Path: canonicalID,
		File: file,
	}
}

// writeTempFile creates a real file on disk so the collision detector can
// resolve it via filepath.EvalSymlinks. Returns the resolved absolute path
// (matching what detectModulePathCollisions will compute internally).
func writeTempFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Resolve the same way detectModulePathCollisions does (Abs + EvalSymlinks)
	// so that assertions on error messages match on all platforms (Windows
	// EvalSymlinks can change casing or resolve 8.3 short names).
	abs, err := filepath.Abs(full)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestDetectModulePathCollisions_NoCollision(t *testing.T) {
	dir := t.TempDir()
	fileA := writeTempFile(t, dir, "a.ail", "")
	fileB := writeTempFile(t, dir, "b.ail", "")
	mods := map[string]*loader.LoadedModule{
		"docparse/services/mcp_tools": makeLoadedModule(
			"docparse/services/mcp_tools", fileA, "docparse/services/mcp_tools",
		),
		"docparse/services/auth": makeLoadedModule(
			"docparse/services/auth", fileB, "docparse/services/auth",
		),
	}
	if err := detectModulePathCollisions(mods); err != nil {
		t.Fatalf("unexpected collision: %v", err)
	}
}

func TestDetectModulePathCollisions_SameFileTwoCanonicalIDs(t *testing.T) {
	// The SAME physical file loaded under two canonical IDs (e.g. via
	// module_prefix aliasing) is NOT a collision. This is the docparse
	// regression case: one file, two canonical paths.
	dir := t.TempDir()
	shared := writeTempFile(t, dir, "services/csv_parser.ail", "")
	mods := map[string]*loader.LoadedModule{
		"pkg/sunholo/ailang_parse/services/csv_parser": makeLoadedModule(
			"pkg/sunholo/ailang_parse/services/csv_parser", shared,
			"docparse/services/csv_parser",
		),
		"docparse/services/csv_parser": makeLoadedModule(
			"docparse/services/csv_parser", shared,
			"docparse/services/csv_parser",
		),
	}
	if err := detectModulePathCollisions(mods); err != nil {
		t.Fatalf("same-file aliasing should not be a collision: %v", err)
	}
}

func TestDetectModulePathCollisions_TwoDifferentFiles(t *testing.T) {
	// Two different files both declaring the same module header — the
	// original docparse footgun. Must error.
	dir := t.TempDir()
	localFile := writeTempFile(t, dir, "local/mcp_tools.ail", "")
	pkgFile := writeTempFile(t, dir, "pkg/mcp_tools.ail", "")
	mods := map[string]*loader.LoadedModule{
		"local/services/mcp_tools": makeLoadedModule(
			"local/services/mcp_tools", localFile,
			"docparse/services/mcp_tools",
		),
		"pkg/sunholo/ailang_parse/services/mcp_tools": makeLoadedModule(
			"pkg/sunholo/ailang_parse/services/mcp_tools", pkgFile,
			"docparse/services/mcp_tools",
		),
	}
	err := detectModulePathCollisions(mods)
	if err == nil {
		t.Fatal("expected MOD011 collision error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "MOD011") {
		t.Errorf("error should mention MOD011: %s", msg)
	}
	if !strings.Contains(msg, "docparse/services/mcp_tools") {
		t.Errorf("error should include the colliding module path: %s", msg)
	}
	if !strings.Contains(msg, localFile) {
		t.Errorf("error should include the first file path: %s", msg)
	}
	if !strings.Contains(msg, pkgFile) {
		t.Errorf("error should include the second file path: %s", msg)
	}
}

func TestDetectModulePathCollisions_NoModuleHeader(t *testing.T) {
	dir := t.TempDir()
	fileA := writeTempFile(t, dir, "a.ail", "")
	fileB := writeTempFile(t, dir, "b.ail", "")
	mods := map[string]*loader.LoadedModule{
		"a": makeLoadedModule("a", fileA, ""),
		"b": makeLoadedModule("b", fileB, ""),
	}
	if err := detectModulePathCollisions(mods); err != nil {
		t.Fatalf("header-less modules should not collide: %v", err)
	}
}

func TestDetectModulePathCollisions_DeterministicOrder(t *testing.T) {
	// Error message must be stable regardless of Go map iteration order.
	dir := t.TempDir()
	fileA := writeTempFile(t, dir, "aaa.ail", "")
	fileZ := writeTempFile(t, dir, "zzz.ail", "")
	mods := map[string]*loader.LoadedModule{
		"zzz/later/claimant": makeLoadedModule(
			"zzz/later/claimant", fileZ, "shared/path",
		),
		"aaa/earlier/claimant": makeLoadedModule(
			"aaa/earlier/claimant", fileA, "shared/path",
		),
	}
	var first string
	for i := 0; i < 50; i++ {
		err := detectModulePathCollisions(mods)
		if err == nil {
			t.Fatal("expected collision error")
		}
		if i == 0 {
			first = err.Error()
			continue
		}
		if err.Error() != first {
			t.Fatalf("non-deterministic error message:\n  run 0: %s\n  run %d: %s", first, i, err.Error())
		}
	}
	if !strings.Contains(first, fileA) || !strings.Contains(first, fileZ) {
		t.Errorf("error missing expected file paths: %s", first)
	}
}

func TestDetectModulePathCollisions_EmptyFilePathFallback(t *testing.T) {
	// When file paths are empty (e.g., synthetic test modules or loader bugs),
	// fall back to comparing by canonical ID — distinct canonical IDs with
	// empty paths declaring the same module still get flagged.
	mods := map[string]*loader.LoadedModule{
		"x": makeLoadedModule("x", "", "shared"),
		"y": makeLoadedModule("y", "", "shared"),
	}
	if err := detectModulePathCollisions(mods); err == nil {
		t.Fatal("expected collision when two empty-path modules share a declared name")
	}
}

// --- MOD013: module_prefix overlap tests ---

func TestDetectModulePrefixOverlap_SamePrefixRootAndDep(t *testing.T) {
	// Motoko scenario: root "local/motoko_agent" and dep "sunholo/motoko_core"
	// both use module_prefix = "src". Must emit MOD013.
	prefixMap := map[string]string{
		"local/motoko_agent":  "src",
		"sunholo/motoko_core": "src",
	}
	err := detectModulePrefixOverlap(prefixMap, "local/motoko_agent")
	if err == nil {
		t.Fatal("expected MOD013 error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "MOD013") {
		t.Errorf("error should mention MOD013, got: %s", msg)
	}
	if !strings.Contains(msg, "local/motoko_agent") {
		t.Errorf("error should name root package, got: %s", msg)
	}
	if !strings.Contains(msg, "sunholo/motoko_core") {
		t.Errorf("error should name dep package, got: %s", msg)
	}
	if !strings.Contains(msg, `"src"`) {
		t.Errorf("error should show the shared prefix, got: %s", msg)
	}
}

func TestDetectModulePrefixOverlap_DifferentPrefixes(t *testing.T) {
	// Root and dep have different prefixes — no overlap, no error.
	prefixMap := map[string]string{
		"local/my_project": "src",
		"sunholo/some_dep": "lib",
	}
	if err := detectModulePrefixOverlap(prefixMap, "local/my_project"); err != nil {
		t.Fatalf("different prefixes should not trigger MOD013: %v", err)
	}
}

func TestDetectModulePrefixOverlap_RootOnlyPrefix(t *testing.T) {
	// Root has a prefix, no deps share it — no error.
	prefixMap := map[string]string{
		"local/my_project": "src",
	}
	if err := detectModulePrefixOverlap(prefixMap, "local/my_project"); err != nil {
		t.Fatalf("single-owner prefix should not trigger MOD013: %v", err)
	}
}

func TestDetectModulePrefixOverlap_TwoDepsSharePrefix_NoRoot(t *testing.T) {
	// Two deps share a prefix but neither is the root — no MOD013.
	// (Root not in prefix map at all means root has no module_prefix.)
	prefixMap := map[string]string{
		"sunholo/dep_a": "shared",
		"sunholo/dep_b": "shared",
	}
	if err := detectModulePrefixOverlap(prefixMap, "local/my_root"); err != nil {
		t.Fatalf("dep-only prefix conflict should not trigger MOD013 (root uninvolved): %v", err)
	}
}

func TestDetectModulePrefixOverlap_EmptyMap(t *testing.T) {
	if err := detectModulePrefixOverlap(map[string]string{}, "local/root"); err != nil {
		t.Fatalf("empty map should not error: %v", err)
	}
}

func TestDetectModulePrefixOverlap_ErrorMessageContainsFixHints(t *testing.T) {
	// Error message must include all three fix options.
	prefixMap := map[string]string{
		"local/root": "src",
		"ext/dep":    "src",
	}
	err := detectModulePrefixOverlap(prefixMap, "local/root")
	if err == nil {
		t.Fatal("expected MOD013 error")
	}
	msg := err.Error()
	// All three fix hints must be present.
	for _, hint := range []string{"Remove", "Change", "pkg/"} {
		if !strings.Contains(msg, hint) {
			t.Errorf("error message missing fix hint %q: %s", hint, msg)
		}
	}
}
