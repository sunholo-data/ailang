package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

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

// prefixLoader is a loader inside root package `sunholo/ailang_parse`
// (module_prefix "docparse") whose types/document module is exported.
func prefixLoader(t *testing.T) (*ModuleLoader, string) {
	t.Helper()
	pkgDir := t.TempDir()
	doc := writeModule(t, pkgDir, "docparse/types/document.ail",
		"module docparse/types/document\nexport type Block = Para(string)\nexport func newDoc() -> string = \"doc\"\n")
	ml := NewModuleLoader(t.TempDir())
	ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{
		"sunholo/ailang_parse/types/document": doc,
	}})
	ml.SetModulePrefixMap(map[string]string{"sunholo/ailang_parse": "docparse"})
	ml.SetCurrentPackageName("sunholo/ailang_parse")
	return ml, doc
}

func TestCanonicalImportPath_EverySpellingOneIdentity(t *testing.T) {
	ml, doc := prefixLoader(t)
	const canonical = "pkg/sunholo/ailang_parse/types/document"
	spellings := []string{
		canonical,                             // published path
		"docparse/types/document",             // module_prefix spelling
		"sunholo/ailang_parse/types/document", // bare self-import
		"./" + canonical,                      // a leading ./ is cosmetic
	}
	var first *LoadedModule
	for _, spelling := range spellings {
		if got := ml.canonicalImportPath(spelling); got != canonical {
			t.Errorf("canonicalImportPath(%q) = %q, want %q", spelling, got, canonical)
		}
		loaded, err := ml.Load(spelling)
		if err != nil {
			t.Fatalf("Load(%q): %v", spelling, err)
		}
		if loaded.Path != canonical {
			t.Errorf("Load(%q).Path = %q, want %q", spelling, loaded.Path, canonical)
		}
		if loaded.File.Path != doc {
			t.Errorf("Load(%q) read %s, want %s", spelling, loaded.File.Path, doc)
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

func TestCanonicalImportPath_UnresolvedSpellingUnchanged(t *testing.T) {
	ml, _ := prefixLoader(t)
	for _, spelling := range []string{
		"docparse/types/missing",             // prefix matches, resolver rejects
		"sunholo/ailang_parse/types/missing", // self-import, resolver rejects
		"utils/helpers",                      // no prefix: project-relative
		"std/io",                             // stdlib is never a package
		"some/file.ail",                      // explicit file path
		"../up/one",                          // relative path
	} {
		if got := ml.canonicalImportPath(spelling); got != spelling {
			t.Errorf("canonicalImportPath(%q) = %q, want it unchanged", spelling, got)
		}
	}
	// Without a package resolver nothing is a package spelling.
	bare := NewModuleLoader(t.TempDir())
	bare.SetModulePrefixMap(map[string]string{"sunholo/ailang_parse": "docparse"})
	if got := bare.canonicalImportPath("docparse/types/document"); got != "docparse/types/document" {
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
		if got := ml.canonicalImportPath("shared/x"); got != "pkg/v/a_pkg/x" {
			t.Fatalf("run %d: canonicalImportPath(shared/x) = %q, want pkg/v/a_pkg/x", i, got)
		}
	}
}

func TestCanonicalImportPath_MemoClearedWhenInputsChange(t *testing.T) {
	const prefixSpelling = "docparse/types/document"
	const selfSpelling = "sunholo/ailang_parse/types/document"
	// Each setter changes an input of the mapping, so a spelling memoised
	// before it must be recomputed after it.
	for _, tc := range []struct {
		name     string
		spelling string
		change   func(*ModuleLoader)
	}{
		{"SetModulePrefixMap", prefixSpelling, func(ml *ModuleLoader) { ml.SetModulePrefixMap(map[string]string{}) }},
		{"SetCurrentPackageName", selfSpelling, func(ml *ModuleLoader) { ml.SetCurrentPackageName("") }},
		{"SetPackageResolver", prefixSpelling, func(ml *ModuleLoader) {
			ml.SetPackageResolver(&mockPkgResolver{files: map[string]string{}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ml, _ := prefixLoader(t)
			if got := ml.canonicalImportPath(tc.spelling); !strings.HasPrefix(got, "pkg/") {
				t.Fatalf("setup: canonicalImportPath(%q) = %q", tc.spelling, got)
			}
			tc.change(ml)
			if got := ml.canonicalImportPath(tc.spelling); got != tc.spelling {
				t.Errorf("after %s: got %q, want the spelling unchanged (stale memo)", tc.name, got)
			}
		})
	}
}

func TestLoad_SpellingFirstStillCachedCanonically(t *testing.T) {
	// The first Load of a module arrives with a non-canonical spelling: the
	// instance must still be stored and named under the canonical ID, so a
	// later canonical Load finds the same instance.
	ml, _ := prefixLoader(t)
	first, err := ml.Load("docparse/types/document")
	if err != nil {
		t.Fatal(err)
	}
	if first.Path != "pkg/sunholo/ailang_parse/types/document" {
		t.Fatalf("Load(prefix spelling).Path = %q, want the canonical ID", first.Path)
	}
	again, err := ml.Load("pkg/sunholo/ailang_parse/types/document")
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Fatal("canonical Load after a spelling Load returned a second instance")
	}
}

func TestExtractImports_RewritesSpellingKeepsWrittenPath(t *testing.T) {
	ml, _ := prefixLoader(t)
	file := &ast.File{
		Module: &ast.ModuleDecl{Path: "docparse/services/x"},
		Imports: []*ast.ImportDecl{
			{Path: "docparse/types/document", Symbols: []string{"Block"}},
			{Path: "pkg/sunholo/ailang_parse/types/document"},
			{Path: "std/io"},
		},
	}
	got := ml.extractImports(file)
	want := []string{"pkg/sunholo/ailang_parse/types/document", "pkg/sunholo/ailang_parse/types/document", "std/io"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("imports[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	rewritten := file.Imports[0]
	if rewritten.WrittenPath != "docparse/types/document" || !rewritten.IsPackage || rewritten.PackageName != "sunholo/ailang_parse" {
		t.Errorf("rewritten import = %+v", rewritten)
	}
	if file.Imports[1].WrittenPath != "" || file.Imports[2].WrittenPath != "" {
		t.Errorf("an import already canonical must keep an empty WrittenPath: %q %q",
			file.Imports[1].WrittenPath, file.Imports[2].WrittenPath)
	}
	if got := ImportDisplay(rewritten); got != "pkg/sunholo/ailang_parse/types/document (imported as docparse/types/document)" {
		t.Errorf("ImportDisplay = %q", got)
	}
	if got := ImportDisplay(file.Imports[2]); got != "std/io" {
		t.Errorf("ImportDisplay(std/io) = %q", got)
	}
	if ImportDisplay(nil) != "" {
		t.Error("ImportDisplay(nil) should be empty")
	}
}

func TestGetImportExport_IMP010NamesBoth(t *testing.T) {
	ml, _ := prefixLoader(t)
	imp := &ast.ImportDecl{Path: "pkg/sunholo/ailang_parse/types/document", WrittenPath: "docparse/types/document"}
	if _, err := ml.GetImportExport(imp, "newDoc"); err != nil {
		t.Fatalf("exported symbol: %v", err)
	}
	_, err := ml.GetImportExport(imp, "Missing")
	if err == nil {
		t.Fatal("expected IMP010")
	}
	want := "not exported by 'pkg/sunholo/ailang_parse/types/document' (imported as 'docparse/types/document')"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("IMP010 = %q, want substring %q", err.Error(), want)
	}
	// An import written canonically names the module once.
	_, err = ml.GetExport("pkg/sunholo/ailang_parse/types/document", "Missing")
	if err == nil || strings.Contains(err.Error(), "imported as") {
		t.Fatalf("canonical spelling should not add an alias note: %v", err)
	}
}
