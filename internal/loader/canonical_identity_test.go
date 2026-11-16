package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// canonOf canonicalises spelling in the root package's scope, failing the
// test on an error.
func canonOf(t *testing.T, ml *ModuleLoader, spelling string) string {
	t.Helper()
	canon, err := ml.canonicalImportPath(spelling)
	if err != nil {
		t.Fatalf("canonicalImportPath(%q): %v", spelling, err)
	}
	return canon
}

// writeModule writes source to dir/rel and returns the absolute path.
func writeModule(t *testing.T, dir, rel, source string) string {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const (
	parsePkg       = "sunholo/ailang_parse"
	docPublished   = "pkg/sunholo/ailang_parse/types/document"
	docPrefix      = "docparse/types/document"
	docBareSelf    = "sunholo/ailang_parse/types/document"
	blogPkg        = "sunholo/blog"
	postsPublished = "pkg/sunholo/blog/posts"
	postsBare      = "sunholo/blog/posts"
)

// parseLoader is a loader that can see package sunholo/ailang_parse
// (module_prefix "docparse", types/document exported). root names the package
// being compiled: "acme/app" makes ailang_parse a dependency, parsePkg makes it
// the root.
func parseLoader(t *testing.T, root string) (*ModuleLoader, string) {
	t.Helper()
	pkgDir := t.TempDir()
	doc := writeModule(t, pkgDir, "docparse/types/document.ail",
		"module docparse/types/document\nexport type Block = Para(string)\nexport func newDoc() -> string = \"doc\"\n")
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{docBareSelf: doc}})
	prefixes := map[string]string{parsePkg: "docparse"}
	if root != parsePkg {
		prefixes[root] = "app"
	}
	ml.SetModulePrefixMap(prefixes)
	ml.SetCurrentPackageName(root)
	return ml, doc
}

// assertOneInstance loads every spelling and checks they all name one module,
// read once from file, under canonical.
func assertOneInstance(t *testing.T, ml *ModuleLoader, file, canonical string, spellings []string) {
	t.Helper()
	var first *LoadedModule
	for _, spelling := range spellings {
		if got := canonOf(t, ml, spelling); got != canonical {
			t.Errorf("canonicalImportPath(%q) = %q, want %q", spelling, got, canonical)
		}
		loaded, err := ml.Load(spelling)
		if err != nil {
			t.Fatalf("Load(%q): %v", spelling, err)
		}
		if loaded.Path != canonical {
			t.Errorf("Load(%q).Path = %q, want %q", spelling, loaded.Path, canonical)
		}
		if loaded.File.Path != file {
			t.Errorf("Load(%q) read %s, want %s", spelling, loaded.File.Path, file)
		}
		if first == nil {
			first = loaded
		} else if loaded != first {
			t.Errorf("Load(%q) returned a second instance of the same file", spelling)
		}
	}
	if len(ml.cache) != 1 {
		keys := make([]string, 0, len(ml.cache))
		for k := range ml.cache {
			keys = append(keys, k)
		}
		t.Fatalf("loader cache holds %d entries %v for one file, want 1", len(ml.cache), keys)
	}
}

func TestCanonicalImportPath_DependencyIsPublishedPath(t *testing.T) {
	ml, doc := parseLoader(t, "acme/app")
	// A dependency's own files write the prefix; its consumers write pkg/.
	assertOneInstance(t, ml, doc, docPublished, []string{docPrefix, docPublished, "./" + docPublished})
	// The bare vendor/name form is a self-import only inside the root package.
	if got := canonOf(t, ml, docBareSelf); got != docBareSelf {
		t.Errorf("bare form of a dependency = %q, want it unchanged (not a self-import)", got)
	}
}

func TestCanonicalImportPath_RootKeepsItsOwnSpelling(t *testing.T) {
	ml, doc := parseLoader(t, parsePkg)
	// Inside the package itself, the prefix spelling is the identity those
	// modules always had; the other spellings fold into it.
	assertOneInstance(t, ml, doc, docPrefix, []string{docPublished, docBareSelf, docPrefix})
}

func TestCanonicalImportPath_RootWithoutPrefixIsBareForm(t *testing.T) {
	dir := t.TempDir()
	posts := writeModule(t, dir, "posts.ail", "module sunholo/blog/posts\nexport func n() -> int = 1\n")
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{postsBare: posts}})
	ml.SetCurrentPackageName(blogPkg)
	// Measured in the packages corpus (a package _smoke.ail): its posts module was
	// loaded under both the bare and the pkg/ self-import spelling.
	assertOneInstance(t, ml, posts, postsBare, []string{postsPublished, postsBare})
}

func TestCanonicalImportPath_UnresolvedSpellingUnchanged(t *testing.T) {
	ml, _ := parseLoader(t, parsePkg)
	for _, spelling := range []string{
		"docparse/types/missing",             // prefix matches, resolver rejects
		"sunholo/ailang_parse/types/missing", // self-import, resolver rejects
		"pkg/sunholo/ailang_parse/missing",   // published path, resolver rejects
		"pkg/onesegment",                     // not vendor/name
		"utils/helpers",                      // no prefix: project-relative
		"std/io",                             // stdlib is never a package
		"some/file.ail",                      // explicit file path
		"../up/one",                          // relative path
	} {
		if got := canonOf(t, ml, spelling); got != spelling {
			t.Errorf("canonicalImportPath(%q) = %q, want it unchanged", spelling, got)
		}
	}
	// Without a package resolver nothing is a package spelling.
	bare := NewModuleLoader(t.TempDir())
	bare.SetModulePrefixMap(map[string]string{parsePkg: "docparse"})
	if got := canonOf(t, bare, docPrefix); got != docPrefix {
		t.Errorf("without a resolver: got %q, want the spelling unchanged", got)
	}
}

func TestCanonicalImportPath_SharedPrefixIsDeterministic(t *testing.T) {
	// Two dependencies share module_prefix "shared" and both export x. The
	// identity must not depend on Go's map iteration order: the sorted first
	// package name wins, every time.
	dir := t.TempDir()
	a := writeModule(t, dir, "a/x.ail", "module shared/x\n")
	b := writeModule(t, dir, "b/x.ail", "module shared/x\n")
	for i := 0; i < 20; i++ {
		ml := NewModuleLoader(t.TempDir())
		ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{"v/b_pkg/x": b, "v/a_pkg/x": a}})
		ml.SetModulePrefixMap(map[string]string{"v/b_pkg": "shared", "v/a_pkg": "shared"})
		if got := canonOf(t, ml, "shared/x"); got != "pkg/v/a_pkg/x" {
			t.Fatalf("run %d: canonicalImportPath(shared/x) = %q, want pkg/v/a_pkg/x", i, got)
		}
	}
}

func TestCanonicalImportPath_MemoClearedWhenInputsChange(t *testing.T) {
	// Each setter changes an input of the mapping, so a spelling memoised
	// before it must be recomputed after it.
	for _, tc := range []struct {
		name, root, spelling, before, after string
		change                              func(*ModuleLoader)
	}{
		{"SetModulePrefixMap", "acme/app", docPrefix, docPublished, docPrefix,
			func(ml *ModuleLoader) { ml.SetModulePrefixMap(map[string]string{}) }},
		{"SetCurrentPackageName", parsePkg, docBareSelf, docPrefix, docBareSelf,
			func(ml *ModuleLoader) { ml.SetCurrentPackageName("") }},
		{"SetPackageResolver", "acme/app", docPrefix, docPublished, docPrefix,
			func(ml *ModuleLoader) { ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{}}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ml, _ := parseLoader(t, tc.root)
			if got := canonOf(t, ml, tc.spelling); got != tc.before {
				t.Fatalf("setup: canonicalImportPath(%q) = %q, want %q", tc.spelling, got, tc.before)
			}
			tc.change(ml)
			if got := canonOf(t, ml, tc.spelling); got != tc.after {
				t.Errorf("after %s: got %q, want %q (stale memo)", tc.name, got, tc.after)
			}
		})
	}
}

func TestLoad_SpellingFirstStillCachedCanonically(t *testing.T) {
	// The first Load of a module arrives with a non-canonical spelling: the
	// instance must still be stored and named under the canonical ID, so a
	// later canonical Load finds the same instance.
	ml, _ := parseLoader(t, "acme/app")
	first, err := ml.Load(docPrefix)
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != docPublished {
		t.Fatalf("Load(prefix spelling).Path = %q, want %q", first.Path, docPublished)
	}
	again, err := ml.Load(docPublished)
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatal("canonical Load after a spelling Load returned a second instance")
	}
}

func TestExtractImports_RewritesSpellingKeepsWrittenPath(t *testing.T) {
	ml, _ := parseLoader(t, "acme/app")
	file := &ast.File{
		Module: &ast.ModuleDecl{Path: "docparse/services/x"},
		Imports: []*ast.ImportDecl{
			{Path: docPrefix, Symbols: []string{"Block"}},
			{Path: docPublished},
			{Path: "std/io"},
		},
	}
	got, err := ml.extractImports(file)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{docPublished, docPublished, "std/io"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("imports[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	rewritten := file.Imports[0]
	if rewritten.WrittenPath != docPrefix || !rewritten.IsPackage || rewritten.PackageName != parsePkg {
		t.Errorf("rewritten import = %+v", rewritten)
	}
	if file.Imports[1].WrittenPath != "" || file.Imports[2].WrittenPath != "" {
		t.Errorf("an import already canonical must keep an empty WrittenPath: %q %q",
			file.Imports[1].WrittenPath, file.Imports[2].WrittenPath)
	}
	if got := DisplayModule(rewritten.Path, rewritten.WrittenPath); got != docPublished+" (imported as "+docPrefix+")" {
		t.Errorf("DisplayModule = %q", got)
	}
	if got := DisplayModule(file.Imports[2].Path, file.Imports[2].WrittenPath); got != "std/io" {
		t.Errorf("DisplayModule(std/io) = %q", got)
	}
}

func TestExtractImports_RootRelativeAndPublishedFoldIntoOwnSpelling(t *testing.T) {
	// Root package sunholo/blog (no prefix): `./posts` becomes
	// pkg/sunholo/blog/posts in extractImports and then the root's own
	// spelling; so does an explicit pkg/ self-import.
	dir := t.TempDir()
	posts := writeModule(t, dir, "posts.ail", "module sunholo/blog/posts\nexport func n() -> int = 1\n")
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{postsBare: posts}})
	ml.SetCurrentPackageName(blogPkg)
	file := &ast.File{
		Module: &ast.ModuleDecl{Path: "sunholo/blog/feed"},
		Imports: []*ast.ImportDecl{
			{Path: "./posts", IsRelative: true, RelativePath: "posts"},
			{Path: postsPublished, IsPackage: true, PackageName: blogPkg},
			{Path: postsBare},
		},
	}
	got, err := ml.extractImports(file)
	if err != nil {
		t.Fatal(err)
	}
	for i, imp := range file.Imports {
		if got[i] != postsBare || imp.Path != postsBare {
			t.Errorf("import %d = (%q, %q), want %q", i, got[i], imp.Path, postsBare)
		}
		if imp.IsPackage || imp.PackageName != "" {
			t.Errorf("import %d rewritten to the root's own spelling must not be marked a pkg/ import: %+v", i, imp)
		}
	}
	if file.Imports[0].WrittenPath != "./posts" || file.Imports[1].WrittenPath != postsPublished || file.Imports[2].WrittenPath != "" {
		t.Errorf("WrittenPath = %q, %q, %q", file.Imports[0].WrittenPath, file.Imports[1].WrittenPath, file.Imports[2].WrittenPath)
	}
}

func TestGetImportExport_IMP010NamesBoth(t *testing.T) {
	ml, _ := parseLoader(t, "acme/app")
	imp := &ast.ImportDecl{Path: docPublished, WrittenPath: docPrefix}
	if _, err := ml.GetImportExport(imp, "newDoc"); err != nil {
		t.Fatalf("exported symbol: %v", err)
	}
	_, err := ml.GetImportExport(imp, "Missing")
	if err == nil {
		t.Fatal("expected IMP010")
	}
	want := "not exported by '" + docPublished + " (imported as " + docPrefix + ")'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("IMP010 = %q, want substring %q", err.Error(), want)
	}
	// An import written canonically names the module once.
	_, err = ml.GetExport(docPublished, "Missing")
	if err == nil || strings.Contains(err.Error(), "imported as") {
		t.Fatalf("canonical spelling should not add an alias note: %v", err)
	}
}
