package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression arms for ailang#671.
//
// The package manifest belongs to the SOURCE FILE, not to the process CWD.
// Anchoring the manifest search at "." meant an installed AILANG CLI invoked
// from anywhere but its own project root could not resolve its own package
// imports, and the error blamed a missing ailang.toml/ailang.lock that were
// sitting next to the source file.
//
// Note why the existing intra-package suite could not catch this: every arm in
// pipeline_module_intra_pkg_test.go chdirs INTO the package before running, so
// all of them exercise the cwd-inside case exclusively. These arms deliberately
// run from a directory that is not the package and is not one of its parents.

const entryDirManifest = `[package]
name = "sunholo/entry_dir_test"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/entry_dir_test/types", "sunholo/entry_dir_test/service"]
`

const entryDirTypes = `module sunholo/entry_dir_test/types

export pure func name() -> string = "Foo"
`

// chdirTo moves the process into dir for the duration of the test. The whole
// point of these arms is that the CWD is NOT the package, so getting this
// wrong would make them vacuous.
func chdirTo(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
}

// assertNotAncestor is the anti-vacuity floor: if the CWD were the package or
// one of its parents, FindManifest would walk up and find the manifest anyway,
// and these arms would pass on the unfixed code.
func assertNotAncestor(t *testing.T, cwd, pkgDir string) {
	t.Helper()
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		t.Fatalf("abs cwd: %v", err)
	}
	absPkg, err := filepath.Abs(pkgDir)
	if err != nil {
		t.Fatalf("abs pkg: %v", err)
	}
	rel, err := filepath.Rel(absCwd, absPkg)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "." {
		t.Fatalf("instrument failure: cwd %s is an ancestor of package %s (rel=%q), so this arm would pass without the fix", absCwd, absPkg, rel)
	}
}

func TestPackageImport_ResolvesFromEntryFileDir_NotCWD(t *testing.T) {
	pkgDir := t.TempDir()
	elsewhere := t.TempDir()

	mustWrite(t, pkgDir, "ailang.toml", entryDirManifest)
	mustWrite(t, pkgDir, "types.ail", entryDirTypes)
	mustWrite(t, pkgDir, "service.ail", `module sunholo/entry_dir_test/service

import pkg/sunholo/entry_dir_test/types (name)

export pure func greet() -> string = "Hello ${name()}"
`)

	chdirTo(t, elsewhere)
	assertNotAncestor(t, elsewhere, pkgDir)

	src := Source{Filename: filepath.Join(pkgDir, "service.ail")}
	cfg := Config{Mode: ModeCheck, RelaxModules: true}

	if _, err := Run(cfg, src); err != nil {
		t.Fatalf("pkg/<self>/sibling import must resolve when the entry file is given by absolute path from an unrelated CWD; got: %v", err)
	}
}

func TestRelativeImport_ResolvesFromEntryFileDir_NotCWD(t *testing.T) {
	pkgDir := t.TempDir()
	elsewhere := t.TempDir()

	mustWrite(t, pkgDir, "ailang.toml", entryDirManifest)
	mustWrite(t, pkgDir, "types.ail", entryDirTypes)
	mustWrite(t, pkgDir, "service.ail", `module sunholo/entry_dir_test/service

import ./types (name)

export pure func greet() -> string = "Hello ${name()}"
`)

	chdirTo(t, elsewhere)
	assertNotAncestor(t, elsewhere, pkgDir)

	src := Source{Filename: filepath.Join(pkgDir, "service.ail")}
	cfg := Config{Mode: ModeCheck, RelaxModules: true}

	if _, err := Run(cfg, src); err != nil {
		t.Fatalf("./sibling import must resolve when the entry file is given by absolute path from an unrelated CWD; got: %v", err)
	}
}

// TestPackageResolverAbsentReason_DistinguishesTheTwoCauses is the arm that
// pins the *message*. The shipped defect was not that an error was raised —
// it was that ONE string was raised for two opposite situations, so a user
// whose manifest existed was told to create it.
func TestPackageResolverAbsentReason_DistinguishesTheTwoCauses(t *testing.T) {
	noManifest := t.TempDir()
	withManifest := t.TempDir()
	mustWrite(t, withManifest, "ailang.toml", entryDirManifest)

	missing := packageResolverAbsentReason(noManifest)
	present := packageResolverAbsentReason(withManifest)

	if missing == present {
		t.Fatalf("one message for two opposite causes — this is the ailang#671 defect:\n  %s", missing)
	}
	if !strings.Contains(missing, "no ailang.toml was found") {
		t.Fatalf("a genuinely absent manifest must say so; got: %s", missing)
	}
	if strings.Contains(present, "no ailang.toml was found") {
		t.Fatalf("message claims no ailang.toml while one exists at %s; got: %s", withManifest, present)
	}
	if !strings.Contains(present, filepath.Join(withManifest, "ailang.toml")) {
		t.Fatalf("message must name the manifest it could not load; got: %s", present)
	}
}

// TestPackageResolverAbsentReason_NamesADirectoryThatExists takes the path
// OUT of the product's own message and stats it, rather than re-deriving what
// the message ought to say. A message naming a directory the user never
// searched is exactly as useless as one naming a file that already exists.
func TestPackageResolverAbsentReason_NamesADirectoryThatExists(t *testing.T) {
	noManifest := t.TempDir()
	msg := packageResolverAbsentReason(noManifest)

	// Extract by delimiter, not by whitespace: a temp directory containing a
	// space (plausible on Windows, invisible from darwin) would silently
	// truncate a Fields-based split and this arm would pin the wrong string.
	const lead, tail = "was found in ", " or any parent directory"
	i := strings.Index(msg, lead)
	j := strings.Index(msg, tail)
	// Anti-vacuity floor: the assertions below are meaningless if the message
	// does not carry a directory at all.
	if i < 0 || j < 0 || j <= i+len(lead) {
		t.Fatalf("instrument failure: message carries no directory between %q and %q, so nothing is being pinned; got: %s", lead, tail, msg)
	}
	found := msg[i+len(lead) : j]
	if !filepath.IsAbs(found) {
		t.Fatalf("message must name an absolute directory so it is unambiguous from any CWD; got %q in: %s", found, msg)
	}

	st, err := os.Stat(found)
	if err != nil {
		t.Fatalf("message names directory %s, which does not exist: %v", found, err)
	}
	if !st.IsDir() {
		t.Fatalf("message names %s as a search directory, but it is not a directory", found)
	}

	absSearched, err := filepath.Abs(noManifest)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if found != absSearched {
		t.Fatalf("message names %s but the directory actually searched was %s", found, absSearched)
	}
}

// TestEntrySourceDir_IgnoresNonFiles pins the guard that keeps synthetic
// entry names (REPL buffers, "<stdin>", code compiled from a string) on the
// previous CWD-anchored behaviour instead of inventing a directory from a
// name that never named a file.
func TestEntrySourceDir_IgnoresNonFiles(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.ail")
	if err := os.WriteFile(real, []byte("module real\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if got := entrySourceDir(real); got != dir {
		t.Fatalf("entrySourceDir(%q) = %q, want %q", real, got, dir)
	}
	if got := entrySourceDir(""); got != "" {
		t.Fatalf(`entrySourceDir("") = %q, want ""`, got)
	}
	if got := entrySourceDir("<stdin>"); got != "" {
		t.Fatalf(`entrySourceDir("<stdin>") = %q, want ""`, got)
	}
	if got := entrySourceDir("<embedded>"); got != "" {
		t.Fatalf(`entrySourceDir("<embedded>") = %q, want ""`, got)
	}
	// A bare relative name is already relative to the CWD, so the existing
	// base is correct and must not be overridden.
	if got := entrySourceDir("service.ail"); got != "" {
		t.Fatalf(`entrySourceDir("service.ail") = %q, want ""`, got)
	}
	// A name with a directory component yields it even if nothing is there:
	// this is a pure path computation, and FindManifest owns filesystem truth.
	if got := entrySourceDir(filepath.Join(dir, "does-not-exist.ail")); got != dir {
		t.Fatalf("entrySourceDir on a nonexistent path = %q, want %q", got, dir)
	}
}

// TestUnresolvablePackageImport_ErrorNamesTheSearchedDir pins the whole chain
// pipeline -> SetPackageResolverAbsentReason -> loader message. Without it the
// diagnosis is computed and then dropped on the floor, and every arm above
// still passes: the helper is tested, the wiring is not.
func TestUnresolvablePackageImport_ErrorNamesTheSearchedDir(t *testing.T) {
	srcDir := t.TempDir()
	elsewhere := t.TempDir()

	mustWrite(t, srcDir, "orphan.ail", `module orphan

import pkg/some/other/thing (x)

export pure func use() -> int = x()
`)

	chdirTo(t, elsewhere)
	assertNotAncestor(t, elsewhere, srcDir)

	src := Source{Filename: filepath.Join(srcDir, "orphan.ail")}
	cfg := Config{Mode: ModeCheck, RelaxModules: true}

	_, err := Run(cfg, src)
	if err == nil {
		t.Fatalf("instrument failure: importing pkg/some/other/thing with no manifest anywhere must fail")
	}
	msg := err.Error()

	absSrc, _ := filepath.Abs(srcDir)
	if !strings.Contains(msg, absSrc) {
		t.Fatalf("error must name the directory actually searched (%s) so the user can see where ailang looked; got: %s", absSrc, msg)
	}
	// The shipped defect: the message instructed the user to create files
	// without ever saying where it had looked for them.
	if strings.Contains(msg, "requires ailang.toml and ailang.lock") {
		t.Fatalf("error reverted to the ailang#671 wording, which asserts a cause it cannot know; got: %s", msg)
	}
}
