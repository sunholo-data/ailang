package loader

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type layout struct {
	dir, prefix string
	deps        []string
}

// layoutResolver is a mock package resolver that also describes package
// layouts (pkg.PackageLoader implements both).
type layoutResolver struct {
	mockPkgResolver
	layouts map[string]layout
}

func (l *layoutResolver) PackageLayout(name string) (string, string, []string, bool) {
	lay, ok := l.layouts[name]
	return lay.dir, lay.prefix, lay.deps, ok
}

func TestCanonicalImportPath_IdentityFromResolvedFile(t *testing.T) {
	// A package without an exports list: pkg/acme/shapes/types/shape and
	// pkg/acme/shapes/shp/types/shape name one file.
	dir := t.TempDir()
	shape := writeModule(t, dir, "shp/types/shape.ail", "module shp/types/shape\nexport func n() -> int = 1\n")
	lr := &layoutResolver{
		mockPkgResolver: mockPkgResolver{files: map[string]string{
			"acme/shapes/types/shape":     shape,
			"acme/shapes/shp/types/shape": shape,
		}},
		layouts: map[string]layout{"acme/shapes": {dir: dir, prefix: "shp"}},
	}
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(lr)
	ml.SetModulePrefixMap(map[string]string{"acme/shapes": "shp"})
	assertOneInstance(t, ml, shape, "pkg/acme/shapes/types/shape",
		[]string{"pkg/acme/shapes/shp/types/shape", "shp/types/shape", "pkg/acme/shapes/types/shape"})
	if c := ml.IdentityConflicts(); len(c) != 0 {
		t.Fatalf("identity conflicts: %v", c)
	}
}

// scopeFixture: aaa/one and bbb/two both declare module_prefix "px" and both
// have px/t; acme/mid declares bbb/two; acme/both declares both; bbb/two also
// declares aaa/one (its own prefix must still win); the root is acme/root
// (prefix "rt", module rt/secret).
func scopeFixture(t *testing.T) *ModuleLoader {
	t.Helper()
	dir := t.TempDir()
	oneT := writeModule(t, dir, "one/px/t.ail", "module px/t\n")
	twoT := writeModule(t, dir, "two/px/t.ail", "module px/t\n")
	secret := writeModule(t, dir, "root/rt/secret.ail", "module rt/secret\n")
	lr := &layoutResolver{
		mockPkgResolver: mockPkgResolver{files: map[string]string{
			"aaa/one/t": oneT, "bbb/two/t": twoT, "acme/root/secret": secret,
			// A case variant the resolver answers, as on a case-insensitive disk.
			"aaa/one/T": filepath.Join(dir, "one", "px", "T.ail"),
		}},
		layouts: map[string]layout{
			"aaa/one":   {dir: filepath.Join(dir, "one"), prefix: "px"},
			"bbb/two":   {dir: filepath.Join(dir, "two"), prefix: "px", deps: []string{"aaa/one"}},
			"acme/mid":  {dir: filepath.Join(dir, "mid"), prefix: "md", deps: []string{"bbb/two"}},
			"acme/both": {dir: filepath.Join(dir, "both"), prefix: "bo", deps: []string{"bbb/two", "aaa/one"}},
			"acme/root": {dir: filepath.Join(dir, "root"), prefix: "rt", deps: []string{"acme/mid"}},
		},
	}
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(lr)
	ml.SetModulePrefixMap(map[string]string{"aaa/one": "px", "bbb/two": "px", "acme/root": "rt"})
	ml.SetCurrentPackageName("acme/root")
	return ml
}

func TestCanonicalImportPath_PrefixScopedToImportingPackage(t *testing.T) {
	ml := scopeFixture(t)
	for owner, want := range map[string]string{
		"bbb/two":  "pkg/bbb/two/t", // its own prefix, though its dependency shares it
		"aaa/one":  "pkg/aaa/one/t",
		"acme/mid": "pkg/bbb/two/t", // the one declared dependency with that prefix
	} {
		got, err := ml.canonicalImportPathFor(owner, "px/t")
		if err != nil || got != want {
			t.Errorf("owner %q: px/t -> %q, %v; want %q", owner, got, err, want)
		}
	}
}

func TestCanonicalImportPath_ScopeErrors(t *testing.T) {
	ml := scopeFixture(t)
	for _, tc := range []struct{ owner, spelling, want string }{
		// Two declared dependencies provide the prefix: ambiguous, both named in order.
		{"acme/both", "px/t", "LDR005: package acme/both imports \"px/t\", but its dependencies aaa/one and bbb/two all provide it"},
		// A dependency's own module that does not exist fails in ITS scope, showing
		// the published path it was looked up as and what was written.
		{"bbb/two", "px/missing", "LDR001: module pkg/bbb/two/missing (imported as px/missing), imported by package bbb/two, was not found in its scope"},
		// A published path into a package the importer does not declare.
		{"acme/mid", "pkg/aaa/one/t", "LDR006: package acme/mid imports \"pkg/aaa/one/t\" from package aaa/one, which it does not declare"},
		// A prefix only an undeclared package provides (the build-wide table would have).
		{"acme/mid", "px/t-not", "LDR001"},
		// A dependency importing the root's modules, by prefix and by bare name.
		{"acme/mid", "rt/secret", "LDR006: package acme/mid imports \"rt/secret\", which is a module of the root package acme/root"},
		{"acme/mid", "acme/root/secret", "LDR006: package acme/mid imports \"acme/root/secret\", which is a module of the root package acme/root"},
		// Nothing anywhere provides it.
		{"acme/mid", "zz/q", "LDR001: module zz/q, imported by package acme/mid, was not found in its scope"},
		// A spelling whose case differs from the file on disk, on every host.
		{"acme/root", "pkg/aaa/one/T", "LDR006"}, // undeclared by the root first
		{"acme/both", "pkg/aaa/one/T", "LDR001: import \"pkg/aaa/one/T\" names \"T.ail\" in package aaa/one, but the file on disk is spelled \"t.ail\""},
	} {
		_, err := ml.canonicalImportPathFor(tc.owner, tc.spelling)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("owner %q imports %q: err = %v, want it to contain %q", tc.owner, tc.spelling, err, tc.want)
		}
	}
	// The error reaches Load of the importing module.
	file := &ast.File{Imports: []*ast.ImportDecl{{Path: "px/t"}}}
	if _, err := ml.extractImportsFor("acme/both", file); err == nil || !strings.Contains(err.Error(), "LDR005") {
		t.Errorf("extractImportsFor must return the scope error, got %v", err)
	}
}

func TestCanonicalImportPath_UnscopedKeepsLegacyTable(t *testing.T) {
	ml := scopeFixture(t)
	ml.SetCurrentPackageName("")
	got, err := ml.canonicalImportPathFor("", "px/t")
	if err != nil || got != "pkg/aaa/one/t" {
		t.Errorf("no package scope: px/t -> %q, %v; want the sorted build-wide table's pkg/aaa/one/t", got, err)
	}
	plain := NewModuleLoader(t.TempDir())
	plain.SetPackageResolver(&mockPkgResolver{files: map[string]string{}})
	if got, err := plain.canonicalImportPathFor("acme/x", "px/t"); err != nil || got != "px/t" {
		t.Errorf("a resolver without layouts leaves an unresolved spelling: %q, %v", got, err)
	}
}

func TestModulePathCandidates(t *testing.T) {
	got := modulePathCandidates("src/shp/types/shape", "shp", "acme/shapes")
	want := []string{"/types/shape", "/shp/types/shape", "/src/shp/types/shape"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("candidates = %v, want %v", got, want)
	}
	if got := modulePathCandidates("core", "", "acme/shapes"); got[0] != "" {
		t.Errorf("core.ail is the package's root module first, got %v", got)
	}
	if got := modulePathCandidates("acme/shapes/x", "", "acme/shapes"); got[0] != "/x" {
		t.Errorf("checkout layout strips vendor/name, got %v", got)
	}
}

func TestIdentityConflicts_SameFileTwoIDs(t *testing.T) {
	// One file read through a symlinked directory and through its real path is
	// one file under two module IDs: the loader records the conflict.
	dir := t.TempDir()
	realPath := writeModule(t, dir, "real/a.ail", "module a\nexport func n() -> int = 1\n")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Join(dir, "real"), link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	ml := NewModuleLoader(dir)
	if _, err := ml.Load(realPath); err != nil {
		t.Fatal(err)
	}
	if c := ml.IdentityConflicts(); len(c) != 0 {
		t.Fatalf("one load, conflicts %v", c)
	}
	if _, err := ml.Load(filepath.Join(link, "a.ail")); err != nil {
		t.Fatal(err)
	}
	c := ml.IdentityConflicts()
	if len(c) != 1 || !strings.Contains(c[0], "under two identities") {
		t.Fatalf("want one same-file conflict, got %v", c)
	}
	// And the other direction: one ID read from two files.
	ml2 := NewModuleLoader(dir)
	ml2.recordIdentity("m", realPath)
	ml2.recordIdentity("m", writeModule(t, dir, "other.ail", "module m\n"))
	if c := ml2.IdentityConflicts(); len(c) != 1 || !strings.Contains(c[0], "read from two files") {
		t.Fatalf("want one two-file conflict, got %v", c)
	}
}
