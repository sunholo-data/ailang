package loader

import (
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

func TestCanonicalImportPath_PrefixScopedToImportingPackage(t *testing.T) {
	dir := t.TempDir()
	oneT := writeModule(t, dir, "one/px/t.ail", "module px/t\n")
	twoT := writeModule(t, dir, "two/px/t.ail", "module px/t\n")
	lr := &layoutResolver{
		mockPkgResolver: mockPkgResolver{files: map[string]string{"aaa/one/t": oneT, "bbb/two/t": twoT}},
		layouts: map[string]layout{
			"aaa/one":  {dir: filepath.Join(dir, "one"), prefix: "px"},
			"bbb/two":  {dir: filepath.Join(dir, "two"), prefix: "px"},
			"acme/mid": {dir: filepath.Join(dir, "mid"), prefix: "md", deps: []string{"bbb/two"}},
		},
	}
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(lr)
	ml.SetModulePrefixMap(map[string]string{"aaa/one": "px", "bbb/two": "px"})
	for owner, want := range map[string]string{
		"bbb/two":  "pkg/bbb/two/t", // its own prefix
		"aaa/one":  "pkg/aaa/one/t",
		"acme/mid": "pkg/bbb/two/t", // a declared dependency's prefix
		"":         "pkg/aaa/one/t", // no scope: the build-wide table, sorted
	} {
		if got := ml.canonicalImportPathFor(owner, "px/t"); got != want {
			t.Errorf("owner %q: px/t -> %q, want %q", owner, got, want)
		}
	}
	// A dependency's own prefix spelling that does not resolve is pinned to the
	// dependency, never handed to another package that shares the prefix.
	if got := ml.canonicalImportPathFor("bbb/two", "px/missing"); got != "pkg/bbb/two/missing" {
		t.Errorf("unresolved own-prefix spelling = %q, want pkg/bbb/two/missing", got)
	}
	if _, err := ml.Load(ml.canonicalImportPathFor("bbb/two", "px/missing")); err == nil {
		t.Error("loading a dependency's missing own module must fail")
	}
	// A spelling no package in scope claims, and none in the table, is unchanged.
	if got := ml.canonicalImportPathFor("acme/mid", "zz/q"); got != "zz/q" {
		t.Errorf("unclaimed prefix = %q, want unchanged", got)
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
	if len(c) != 1 || !strings.Contains(c[0], "loaded as") {
		t.Fatalf("want one same-file conflict, got %v", c)
	}
	// And the other direction: one ID read from two files.
	ml2 := NewModuleLoader(dir)
	ml2.recordIdentity("m", realPath)
	ml2.recordIdentity("m", writeModule(t, dir, "other.ail", "module m\n"))
	if c := ml2.IdentityConflicts(); len(c) != 1 || !strings.Contains(c[0], "read from") {
		t.Fatalf("want one two-file conflict, got %v", c)
	}
}
