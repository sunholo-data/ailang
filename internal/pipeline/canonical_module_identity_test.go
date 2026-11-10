package pipeline

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// M1b (m-v1-nominal-type-identity): one module file, one identity. These tests
// fail if a second module ID for the same source file reappears.

// assertOneIdentityPerFile fails when two loaded modules were read from the
// same source file — the double load M1b removes.
func assertOneIdentityPerFile(t *testing.T, modules map[string]*loader.LoadedModule) {
	t.Helper()
	byFile := map[string][]string{}
	for id, mod := range modules {
		if mod == nil || mod.File == nil || mod.File.Path == "" || strings.HasPrefix(id, "std/") {
			continue
		}
		byFile[canonicalFilePath(mod.File.Path)] = append(byFile[canonicalFilePath(mod.File.Path)], id)
	}
	for file, ids := range byFile {
		if len(ids) > 1 {
			sort.Strings(ids)
			t.Errorf("%s loaded under %d module identities %v; want exactly one", file, len(ids), ids)
		}
	}
}

// assertImportsResolveToLoaded checks that every import in every loaded module
// names a module that is itself loaded under that exact ID, so no consumer can
// reach a second instance through a spelling.
func assertImportsResolveToLoaded(t *testing.T, modules map[string]*loader.LoadedModule) {
	t.Helper()
	for id, mod := range modules {
		for _, imp := range mod.Imports {
			if _, ok := modules[imp]; !ok {
				t.Errorf("module %s imports %q, which is not a loaded module ID", id, imp)
			}
		}
	}
}

func checkFixture(t *testing.T, entry string) Result {
	t.Helper()
	result, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: entry})
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("check %s: %v %v", entry, err, result.Errors)
	}
	return result
}

func TestCanonicalModuleIdentity_DependencyReachedByTwoSpellings(t *testing.T) {
	app, shapes := prefixpkg.New(t, nil)
	result := checkFixture(t, filepath.Join(app, "main.ail"))

	assertOneIdentityPerFile(t, result.Modules)
	assertImportsResolveToLoaded(t, result.Modules)
	shape, ok := result.Modules[prefixpkg.Canonical]
	if !ok {
		t.Fatalf("canonical module %s not loaded; modules: %v", prefixpkg.Canonical, sortedKeys(result.Modules))
	}
	if _, dup := result.Modules[prefixpkg.PrefixSpelling]; dup {
		t.Fatalf("module also loaded under its module_prefix spelling %s", prefixpkg.PrefixSpelling)
	}
	if got, want := canonicalFilePath(shape.File.Path), canonicalFilePath(filepath.Join(shapes, prefixpkg.ShapeFile)); got != want {
		t.Fatalf("canonical module read from %s, want %s", got, want)
	}
	// The dependency's own import keeps the author's spelling for diagnostics.
	area := result.Modules["pkg/acme/shapes/ops/area"]
	if area == nil {
		t.Fatalf("area not loaded under its canonical ID; modules: %v", sortedKeys(result.Modules))
	}
	imp := area.File.Imports[0]
	if imp.Path != prefixpkg.Canonical || imp.WrittenPath != prefixpkg.PrefixSpelling {
		t.Fatalf("area's import = (Path %q, WrittenPath %q), want (%q, %q)",
			imp.Path, imp.WrittenPath, prefixpkg.Canonical, prefixpkg.PrefixSpelling)
	}
	if !imp.IsPackage || imp.PackageName != "acme/shapes" {
		t.Fatalf("rewritten import not marked as a package import: IsPackage=%v PackageName=%q", imp.IsPackage, imp.PackageName)
	}
	// Constructors cross from the instance area saw to the one main imported:
	// there is only one interface, and both name the same module.
	if shape.Iface == nil || shape.Iface.Module != prefixpkg.Canonical {
		t.Fatalf("shape iface module = %v, want %s", shape.Iface, prefixpkg.Canonical)
	}
}

func TestCanonicalModuleIdentity_InsideOwnPackageThreeSpellings(t *testing.T) {
	_, shapes := prefixpkg.New(t, nil)
	result := checkFixture(t, filepath.Join(shapes, "shp", "both.ail"))

	assertOneIdentityPerFile(t, result.Modules)
	assertImportsResolveToLoaded(t, result.Modules)
	if _, ok := result.Modules[prefixpkg.Canonical]; !ok {
		t.Fatalf("canonical module %s not loaded; modules: %v", prefixpkg.Canonical, sortedKeys(result.Modules))
	}
	for _, spelling := range []string{prefixpkg.PrefixSpelling, "acme/shapes/types/shape"} {
		if _, dup := result.Modules[spelling]; dup {
			t.Errorf("module also loaded under the spelling %s", spelling)
		}
	}
}

func TestCanonicalModuleIdentity_IMP010NamesCanonicalAndSpelling(t *testing.T) {
	app, _ := prefixpkg.New(t, map[string]string{
		"shapes/shp/ops/area.ail": `module shp/ops/area

import shp/types/shape (Shape, Triangle)

export func area(s: Shape) -> int = 0

export func unitSquare() -> Shape = area(unitSquare())
`,
	})
	_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: filepath.Join(app, "main.ail")})
	if err == nil {
		t.Fatal("expected IMP010 for a symbol the module does not export")
	}
	msg := err.Error()
	want := "symbol 'Triangle' not exported by '" + prefixpkg.Canonical + "' (imported as '" + prefixpkg.PrefixSpelling + "')"
	if !strings.Contains(msg, want) {
		t.Fatalf("IMP010 should name the canonical module and the written spelling\n got: %s\nwant substring: %s", msg, want)
	}
}

// A dependency module reached through its module_prefix spelling used to get
// a non-pkg/ module ID, which isOwnPackageModule reads as "the root's own
// module": its effects were charged to the ROOT's [effects].max ("effect
// ceiling violation in package acme/app" for code acme/app does not own),
// while the same file under its pkg/ spelling was exempt. With one identity
// the dependency is judged like every other dependency.
func TestCanonicalModuleIdentity_DependencyNotChargedToRootCeiling(t *testing.T) {
	files := prefixpkg.Files()
	app, _ := prefixpkg.New(t, map[string]string{
		"app/ailang.toml": files["app/ailang.toml"] + "\n[effects]\nmax = [\"IO\"]\n",
		"shapes/" + prefixpkg.ShapeFile: strings.Replace(files["shapes/"+prefixpkg.ShapeFile],
			"import std/debug (log)", "import std/debug (log)\nimport std/fs (readFile)", 1) +
			"\nexport func slurp(p: string) -> string ! {FS} = readFile(p)\n",
	})
	result := checkFixture(t, filepath.Join(app, "main.ail"))
	assertOneIdentityPerFile(t, result.Modules)
	// The root's own modules are still held to the ceiling.
	if !isOwnPackageModule("main", "acme/app") || isOwnPackageModule(prefixpkg.Canonical, "acme/app") {
		t.Fatal("isOwnPackageModule: root module must be own, the dependency must not")
	}
}

func sortedKeys(m map[string]*loader.LoadedModule) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
