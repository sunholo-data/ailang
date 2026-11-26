package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// TC_DEPRECATED_001 (M-V1-STDLIB-SURFACE-POLISH §6.1): one warning per use
// site of an @deprecated function, from another module, on a cold and a warm
// compile cache.

const deprecatedLib = `module lib

export pure func g(x: int) -> int = x + 1

export pure func g2(a: int, b: int) -> int = b - a

@deprecated("g")
export pure func f(x: int) -> int = g(x)

@deprecated("g2", "arguments swap: f2(a, b) becomes g2(b, a)")
export pure func f2(a: int, b: int) -> int = a - b

-- The definition and the module's own uses never warn.
export pure func usesOwn(x: int) -> int = f(x)
`

func deprecatedWarnings(ws []elaborate.Warning) []*DeprecatedUseWarning {
	var out []*DeprecatedUseWarning
	for _, w := range ws {
		if dw, ok := w.(*DeprecatedUseWarning); ok {
			out = append(out, dw)
		}
	}
	return out
}

func checkDeprecatedModule(t *testing.T, file string) []*DeprecatedUseWarning {
	t.Helper()
	res, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: file}, productionCacheDependencies())
	if err != nil {
		t.Fatalf("%s: a deprecation warning must not block compilation: %v", file, err)
	}
	return deprecatedWarnings(res.Warnings)
}

func writeModule(t *testing.T, name, src string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDeprecatedUseWarning(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	writeModule(t, "lib.ail", deprecatedLib)
	// mid forwards to f: the use is IN mid, so mid's line warns once and a
	// caller of h does not warn at all (it never names f).
	writeModule(t, "mid.ail", "module mid\nimport lib (f)\nexport pure func h(x: int) -> int = f(x)\n")

	type want struct {
		line int
		name string
		file string // basename of the file holding the use
	}
	cases := []struct {
		name string
		src  string
		want []want
	}{
		{"callHead", "module callHead\nimport lib (f)\nexport pure func k() -> int = f(1)\n",
			[]want{{3, "f", "callHead.ail"}}},
		{"valueArg", "module valueArg\nimport lib (f)\nimport std/list (map)\nexport pure func k(xs: [int]) -> [int] = map(f, xs)\n",
			[]want{{4, "f", "valueArg.ail"}}},
		{"inTuple", "module inTuple\nimport lib (f)\nexport pure func k() -> (int -> int, int) = (f, 1)\n",
			[]want{{3, "f", "inTuple.ail"}}},
		{"qualified", "module qualified\nimport lib as L\nexport pure func k() -> int = L.f(1)\n",
			[]want{{3, "f", "qualified.ail"}}},
		{"renamedImport", "module renamedImport\nimport lib (f as old)\nexport pure func k() -> int = old(1)\n",
			[]want{{3, "f", "renamedImport.ail"}}},
		{"twoUses", "module twoUses\nimport lib (f, f2)\nexport pure func k() -> int = f(1)\nexport pure func k2() -> int = f2(1, 2)\n",
			[]want{{3, "f", "twoUses.ail"}, {4, "f2", "twoUses.ail"}}},
		{"sameNameTwice", "module sameNameTwice\nimport lib (f)\nexport pure func k() -> int = f(1)\nexport pure func k2() -> int = f(f(2))\n",
			[]want{{3, "f", "sameNameTwice.ail"}, {4, "f", "sameNameTwice.ail"}, {4, "f", "sameNameTwice.ail"}}},
		// Imported and never used: still breaks at alias removal, so the
		// import itself warns, exactly once.
		{"importOnly", "module importOnly\nimport lib (f)\nexport pure func k() -> int = 1\n",
			[]want{{2, "f", "importOnly.ail"}}},
		{"importOnlyRenamed", "module importOnlyRenamed\nimport lib (f as old)\nexport pure func k() -> int = 1\n",
			[]want{{2, "f", "importOnlyRenamed.ail"}}},
		// One used (warns at the use only), one unused (warns at the import).
		{"usedAndUnused", "module usedAndUnused\nimport lib (f, f2)\nexport pure func k() -> int = f(1)\n",
			[]want{{2, "f2", "usedAndUnused.ail"}, {3, "f", "usedAndUnused.ail"}}},
		// Used only in a test block: the use warns, the import does not.
		{"usedOnlyInTest", "module usedOnlyInTest\nimport lib (f)\nexport pure func k() -> int = 1\ntest \"t\" {\n  assert f(1) == 2\n}\n",
			[]want{{5, "f", "usedOnlyInTest.ail"}}},
		{"replacementOnly", "module replacementOnly\nimport lib (g)\nexport pure func k() -> int = g(1)\n", nil},
		{"viaForwarder", "module viaForwarder\nimport mid (h)\nexport pure func k() -> int = h(1)\n",
			[]want{{3, "f", "mid.ail"}}},
		{"inLambda", "module inLambda\nimport lib (f)\nexport pure func k() -> int = (\\x. f(x))(1)\n",
			[]want{{3, "f", "inLambda.ail"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			writeModule(t, c.name+".ail", c.src)
			for _, run := range []string{"cold", "warm"} {
				ws := checkDeprecatedModule(t, c.name+".ail")
				if len(ws) != len(c.want) {
					t.Fatalf("%s run: %d TC_DEPRECATED_001 warnings, want %d: %v", run, len(ws), len(c.want), ws)
				}
				for i, w := range ws {
					line, _ := w.Position()
					if line != c.want[i].line || w.Name != c.want[i].name || w.Module != "lib" ||
						filepath.Base(w.WarningFile()) != c.want[i].file {
						t.Errorf("%s run: warning %d = %s.%s at %s:%d, want lib.%s at %s:%d",
							run, i, w.Module, w.Name, w.WarningFile(), line, c.want[i].name, c.want[i].file, c.want[i].line)
					}
					if w.Code() != CodeDeprecated || w.Replacement == "" ||
						!strings.Contains(w.Message(), "is deprecated; use "+w.Replacement) ||
						!strings.Contains(w.Suggestion(), w.Replacement) {
						t.Errorf("%s run: malformed warning: %s", run, w.String())
					}
					if w.Name == "f2" && !strings.Contains(w.Suggestion(), "arguments swap") {
						t.Errorf("%s run: the @deprecated note is missing: %s", run, w.Suggestion())
					}
				}
			}
		})
	}

	// The defining module alone never warns (its own uses are not VarGlobals).
	if ws := checkDeprecatedModule(t, "lib.ail"); len(ws) != 0 {
		t.Errorf("defining module warned about itself: %v", ws)
	}
}

// Each unused deprecated import warns at its own name in the import list, not
// at the `import` keyword: two on one line get two distinct positions.
func TestDeprecatedUnusedImportPosition(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	writeModule(t, "lib.ail", deprecatedLib)
	writeModule(t, "app.ail", "module app\nimport lib (g,  f as old,\n  f2)\nexport pure func k() -> int = g(1)\n")
	for _, run := range []string{"cold", "warm"} {
		var got []string
		for _, w := range checkDeprecatedModule(t, "app.ail") {
			line, col := w.Position()
			got = append(got, fmt.Sprintf("%s@%d:%d", w.Name, line, col))
		}
		if want := "f@2:17 f2@3:3"; strings.Join(got, " ") != want {
			t.Errorf("%s run: unused-import positions %v, want %s", run, got, want)
		}
	}
}

// The defining module never warns about its own deprecated function, even when
// a reference to it reaches Core as a VarGlobal (today the elaborator gives a
// module's own top-level names as local Vars, so this guard is what keeps a
// future elaborator change from making every std alias module warn).
func TestDetectDeprecatedUses_OwnModuleVarGlobalIsNotAUse(t *testing.T) {
	dep := &ast.FuncDecl{Name: "f", IsExport: true, Annotations: []*ast.Annotation{{
		Name: ast.AnnotationDeprecated,
		Args: []ast.Expr{&ast.Literal{Kind: ast.StringLit, Value: "g"}},
	}}}
	ref := func(file string, line int) core.CoreExpr {
		return &core.VarGlobal{CoreNode: core.CoreNode{OrigSpan: ast.Pos{File: file, Line: line, Column: 1}}, Ref: core.GlobalRef{Module: "lib", Name: "f"}}
	}
	units := map[string]*CompileUnit{
		"lib":  {ID: "lib", Surface: &ast.File{Funcs: []*ast.FuncDecl{dep}}, Core: &core.Program{Decls: []core.CoreExpr{ref("lib.ail", 3)}}},
		"user": {ID: "user", Surface: &ast.File{}, Core: &core.Program{Decls: []core.CoreExpr{ref("user.ail", 5)}}},
	}
	ws := deprecatedWarnings(DetectDeprecatedUses(units))
	if len(ws) != 1 || ws[0].WarningFile() != "user.ail" {
		t.Fatalf("want exactly the use in user.ail, got %v", ws)
	}
}

// A use inside an installed dependency package (pkg/<other>/...) is not the
// importing project's to fix: it is not reported there. The project's own
// modules — including its own pkg/<self>/ modules — still are.
func TestDetectDeprecatedUses_SkipsInstalledDependencies(t *testing.T) {
	dep := &ast.FuncDecl{Name: "head_or", IsExport: true, Annotations: []*ast.Annotation{{
		Name: ast.AnnotationDeprecated,
		Args: []ast.Expr{&ast.Literal{Kind: ast.StringLit, Value: "headOr"}},
	}}}
	use := func(file string) *core.Program {
		return &core.Program{Decls: []core.CoreExpr{&core.VarGlobal{
			CoreNode: core.CoreNode{OrigSpan: ast.Pos{File: file, Line: 1, Column: 1}},
			Ref:      core.GlobalRef{Module: "std/list", Name: "head_or"},
		}}}
	}
	units := map[string]*CompileUnit{
		"std/list":           {ID: "std/list", Surface: &ast.File{Funcs: []*ast.FuncDecl{dep}}, Core: &core.Program{}},
		"pkg/other/lib/util": {ID: "pkg/other/lib/util", Surface: &ast.File{}, Core: use("dep.ail")},
		"pkg/acme/self/mod":  {ID: "pkg/acme/self/mod", Surface: &ast.File{}, Core: use("self.ail")},
		"app/main":           {ID: "app/main", Surface: &ast.File{}, Core: use("main.ail")},
	}
	files := func() []string {
		var fs []string
		for _, w := range deprecatedWarnings(DetectDeprecatedUses(units)) {
			fs = append(fs, w.WarningFile())
		}
		sort.Strings(fs)
		return fs
	}
	if got := strings.Join(files(), ","); got != "main.ail" {
		t.Errorf("no package manifest: warned in %q, want main.ail only", got)
	}
	prev := currentPackageManifest
	defer func() { currentPackageManifest = prev }()
	currentPackageManifest = &pkg.PackageManifest{}
	currentPackageManifest.Package.Name = "acme/self"
	if got := strings.Join(files(), ","); got != "main.ail,self.ail" {
		t.Errorf("inside package acme/self: warned in %q, want main.ail,self.ail", got)
	}
}

// @deprecated must name a replacement that exists: an export of the same
// module, or "mod/path.name" from a module this one imports. A typo is a
// compile error (TC_DEPRECATED_002) at the definition, not a warning that
// tells every caller to use a name that is not there.
func TestDeprecatedReplacementMustExist(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	// Synthetic local slash-path module with a deprecated export (std/ has 0
	// deprecated exports after pre-soak alias removal).
	writeModule(t, "deplib/list.ail", "module deplib/list\nexport pure func headOr(xs: [int], d: int) -> int = d\n@deprecated(\"headOr\")\nexport pure func head_or(xs: [int], d: int) -> int = headOr(xs, d)\n")
	cases := []struct {
		name, src string
		wantErr   string // "" = compiles
	}{
		{"sameModule", "export pure func g(x: int) -> int = x\n@deprecated(\"g\")\nexport pure func f(x: int) -> int = g(x)\n", ""},
		{"typo", "export pure func gNew(x: int) -> int = x\n@deprecated(\"gNw\")\nexport pure func f(x: int) -> int = gNew(x)\n", "did you mean @deprecated(\"gNew\")"},
		{"missing", "@deprecated(\"nonexistent\")\nexport pure func f(x: int) -> int = x\n", "does not export nonexistent"},
		{"self", "@deprecated(\"f\")\nexport pure func f(x: int) -> int = x\n", "cannot be its own replacement"},
		{"qualifiedImported", "import std/list (headOr)\n@deprecated(\"std/list.headOr\")\nexport pure func first(xs: [int]) -> int = headOr(xs, 0)\n", ""},
		{"qualifiedMissing", "import std/list (headOr)\n@deprecated(\"std/list.headOrr\")\nexport pure func first(xs: [int]) -> int = headOr(xs, 0)\n", "std/list does not export headOrr"},
		{"qualifiedNotImported", "@deprecated(\"std/math.absInt\")\nexport pure func a(x: int) -> int = x\n", "is not imported"},
		// A typo in a qualified target suggests the qualified spelling.
		{"qualifiedTypoHint", "import std/list (headOr)\n@deprecated(\"std/list.headOR\")\nexport pure func first(xs: [int]) -> int = headOr(xs, 0)\n", "did you mean @deprecated(\"std/list.headOr\")"},
		// A target that is itself deprecated, or a cycle, is legal: whether a
		// name in another module is deprecated is not in this module's cache
		// key, so rejecting it would make the verdict depend on the cache. The
		// use-site warning names the live end of the chain instead.
		{"targetDeprecated", "export pure func c(x: int) -> int = x\n@deprecated(\"c\")\nexport pure func b(x: int) -> int = c(x)\n@deprecated(\"b\")\nexport pure func a(x: int) -> int = b(x)\n", ""},
		{"targetDeprecatedStd", "import deplib/list (head_or)\n@deprecated(\"deplib/list.head_or\")\nexport pure func first(xs: [int]) -> int = head_or(xs, 0)\n", ""},
		{"cycle", "@deprecated(\"b\")\nexport pure func a(x: int) -> int = x\n@deprecated(\"a\")\nexport pure func b(x: int) -> int = x\n", ""},
		// A module path without a slash is still a qualified target.
		{"qualifiedNoSlash", "@deprecated(\"lib.g\")\nexport pure func a(x: int) -> int = x\n", "is not imported"},
		// The error is placed at the annotation (line 3 here), not the func.
		{"position", "export pure func g(x: int) -> int = x\n@deprecated(\"gg\")\nexport pure func f(x: int) -> int = g(x)\n", "position.ail:3:1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			writeModule(t, c.name+".ail", "module "+c.name+"\n"+c.src)
			_, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: c.name + ".ail"}, productionCacheDependencies())
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("want it to compile, got %v", err)
			case c.wantErr != "" && err == nil:
				t.Fatalf("want %s, it compiled", CodeDeprecatedTarget)
			case c.wantErr != "" && (!strings.Contains(err.Error(), CodeDeprecatedTarget) || !strings.Contains(err.Error(), c.wantErr)):
				t.Fatalf("want %s mentioning %q, got %v", CodeDeprecatedTarget, c.wantErr, err)
			}
		})
	}
	// Cross-module chain check: a caller of targetDeprecatedStd.first gets the
	// live end of the chain (deplib/list.headOr), not the deprecated intermediate.
	writeModule(t, "chainCaller.ail", "module chainCaller\nimport targetDeprecatedStd (first)\nexport pure func run(xs: [int]) -> int = first(xs)\n")
	ws := checkDeprecatedModule(t, "chainCaller.ail")
	var callerWarns []*DeprecatedUseWarning
	for _, w := range ws {
		if filepath.Base(w.WarningFile()) == "chainCaller.ail" {
			callerWarns = append(callerWarns, w)
		}
	}
	if len(callerWarns) != 1 || callerWarns[0].Replacement != "deplib/list.headOr" {
		t.Errorf("cross-module chain warning = %v, want 1 warning naming deplib/list.headOr", callerWarns)
	}
}

// Adding @deprecated to a dependency warns at once, even when the importer is
// served from the compile cache and the dependency's interface digest (which
// ignores the annotation) is unchanged.
func TestDeprecatedUseWarning_AnnotationAddedToCachedDependency(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	plain := "module dep\nexport pure func g(x: int) -> int = x + 1\nexport pure func f(x: int) -> int = g(x)\n"
	writeModule(t, "dep.ail", plain)
	writeModule(t, "app.ail", "module app\nimport dep (f)\nexport pure func k() -> int = f(1)\n")
	if ws := checkDeprecatedModule(t, "app.ail"); len(ws) != 0 {
		t.Fatalf("no annotation yet, got %v", ws)
	}
	writeModule(t, "dep.ail", strings.Replace(plain, "export pure func f", "@deprecated(\"g\")\nexport pure func f", 1))
	for _, run := range []string{"first", "second"} {
		if ws := checkDeprecatedModule(t, "app.ail"); len(ws) != 1 {
			t.Fatalf("%s run after annotating dep: %d warnings, want 1: %v", run, len(ws), ws)
		}
	}
}

// Every std module compiles without a deprecation warning: the stdlib itself
// never calls a deprecated name, or every importer would inherit the warning.
func TestDeprecatedUseWarning_StdlibIsClean(t *testing.T) {
	stdDir := stdSourceDir(t)
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))

	var imports []string
	for _, m := range goldenModules(t, stdDir) {
		data, err := os.ReadFile(filepath.Join(stdDir, "..", ".stdlib-golden", m+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var g struct {
			Funcs []struct {
				Name       string `json:"name"`
				ReplacedBy string `json:"replaced_by"`
			} `json:"funcs"`
		}
		if err := json.Unmarshal(data, &g); err != nil {
			t.Fatal(err)
		}
		for _, f := range g.Funcs {
			if f.ReplacedBy == "" {
				imports = append(imports, fmt.Sprintf("import std/%s (%s as use_%s)", m, f.Name, strings.ReplaceAll(m, "/", "_")))
				break
			}
		}
	}
	src := "module usesall\n" + strings.Join(imports, "\n") + "\nexport pure func k() -> int = 1\n"
	writeModule(t, "usesall.ail", src)
	for _, run := range []string{"cold", "warm"} {
		for _, w := range checkDeprecatedModule(t, "usesall.ail") {
			t.Errorf("%s run: the stdlib uses a deprecated name: %s", run, w)
		}
	}
}

// stdSourceDir is the repository's std/ directory.
func stdSourceDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "std")
}

func goldenModules(t *testing.T, stdDir string) []string {
	t.Helper()
	entries, err := filepath.Glob(filepath.Join(stdDir, "..", ".stdlib-golden", "*.json"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("no goldens: %v", err)
	}
	var mods []string
	for _, e := range entries {
		mods = append(mods, strings.TrimSuffix(filepath.Base(e), ".json"))
	}
	sort.Strings(mods)
	return mods
}

// stdDeprecations parses every std module and returns any @deprecated functions.
func stdDeprecations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	stdDir := stdSourceDir(t)
	out := map[string]*ast.FuncDecl{}
	for _, pattern := range []string{"*.ail", "*/*.ail"} {
		files, err := filepath.Glob(filepath.Join(stdDir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			p := parser.New(lexer.New(string(src), f))
			prog := p.Parse()
			if errs := p.Errors(); len(errs) > 0 {
				t.Fatalf("%s: %v", f, errs[0])
			}
			rel, err := filepath.Rel(stdDir, f)
			if err != nil {
				t.Fatal(err)
			}
			mod := "std/" + strings.TrimSuffix(filepath.ToSlash(rel), ".ail")
			for _, fd := range prog.File.Funcs {
				if _, _, ok := fd.Deprecation(); ok {
					out[mod+"."+fd.Name] = fd
				}
			}
		}
	}
	return out
}

// stdExports parses every std module and returns module -> exported function names.
func stdExports(t *testing.T) map[string]map[string]bool {
	t.Helper()
	stdDir := stdSourceDir(t)
	out := map[string]map[string]bool{}
	for _, pattern := range []string{"*.ail", "*/*.ail"} {
		files, err := filepath.Glob(filepath.Join(stdDir, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			p := parser.New(lexer.New(string(src), f))
			prog := p.Parse()
			if errs := p.Errors(); len(errs) > 0 {
				t.Fatalf("%s: %v", f, errs[0])
			}
			rel, err := filepath.Rel(stdDir, f)
			if err != nil {
				t.Fatal(err)
			}
			mod := "std/" + strings.TrimSuffix(filepath.ToSlash(rel), ".ail")
			names := map[string]bool{}
			for _, fd := range prog.File.Funcs {
				if fd.IsExport {
					names[fd.Name] = true
				}
			}
			out[mod] = names
		}
	}
	return out
}

// All 16 pre-1.0 @deprecated stdlib aliases have been removed before the v1.0.0
// soak, while importhint.StdlibRenames retains all 16 entries so IMP010 keeps
// naming their live replacements.
func TestStdlibHasNoDeprecatedAliases(t *testing.T) {
	deps := stdDeprecations(t)
	if len(deps) != 0 {
		var keys []string
		for k := range deps {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Errorf("std/ still has %d @deprecated declarations, want 0: %v", len(deps), keys)
	}
	if len(importhint.StdlibRenames) != 16 {
		t.Errorf("%d renames in importhint.StdlibRenames, want 16 (R1-R14, D1, D2)", len(importhint.StdlibRenames))
	}
	exports := stdExports(t)
	for _, r := range importhint.StdlibRenames {
		modExports, ok := exports[r.Module]
		if !ok {
			t.Errorf("importhint.StdlibRenames references missing module %s", r.Module)
			continue
		}
		if modExports[r.Old] {
			t.Errorf("%s still exports removed alias %s", r.Module, r.Old)
		}
		if !modExports[r.New] {
			t.Errorf("%s does not export replacement %s", r.Module, r.New)
		}
	}
}

// Importing any of the 16 removed stdlib aliases fails with IMP010 naming its
// replacement (and Note where non-empty).
func TestRemovedStdlibAliasesEmitIMP010Hint(t *testing.T) {
	t.Setenv("AILANG_STDLIB_PATH", stdSourceDir(t))
	root := t.TempDir()
	for i, r := range importhint.StdlibRenames {
		t.Run(r.Module+"."+r.Old, func(t *testing.T) {
			modName := fmt.Sprintf("removed_%d", i)
			path := filepath.Join(root, modName+".ail")
			src := fmt.Sprintf("module %s\nimport %s (%s)\nexport pure func one() -> int = 1\n", modName, r.Module, r.Old)
			if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			res, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: path})
			msg := ""
			if err != nil {
				msg = err.Error()
			} else if len(res.Errors) > 0 {
				msg = fmt.Sprint(res.Errors)
			}
			if msg == "" {
				t.Fatalf("importing removed %s.%s succeeded; want IMP010", r.Module, r.Old)
			}
			wantRepl := fmt.Sprintf("use '%s' from '%s'", r.New, r.Module)
			if !strings.Contains(msg, "IMP010") || !strings.Contains(msg, wantRepl) {
				t.Errorf("import %s (%s): want IMP010 containing %q, got %s", r.Module, r.Old, wantRepl, msg)
			}
			if r.Note != "" && !strings.Contains(msg, r.Note) {
				t.Errorf("import %s (%s): want IMP010 note %q, got %s", r.Module, r.Old, r.Note, msg)
			}
		})
	}
}

// tools/migrate-stdlib-renames.sed rewrites every drop-in rename whose old name
// is a unique token, and documents the four done by hand.
func TestMigrateSedCoversRenames(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(stdSourceDir(t), "..", "tools", "migrate-stdlib-renames.sed"))
	if err != nil {
		t.Fatal(err)
	}
	sed := strings.ReplaceAll(string(data), "\r\n", "\n") // a Windows checkout may have CRLF
	byHand := map[string]bool{"glob": true, "slice": true, "member": true, "unsafeGet": true}
	for _, r := range importhint.StdlibRenames {
		rule := regexp.MustCompile(`(?m)^s/\(\^\|\[\^A-Za-z0-9_\]\)` + regexp.QuoteMeta(r.Old) + `\(\[\^A-Za-z0-9_\]\|\$\)/\\1` + regexp.QuoteMeta(r.New) + `\\2/g$`)
		n := len(rule.FindAllString(sed, -1))
		switch {
		case byHand[r.Old]:
			if n != 0 || !strings.Contains(sed, r.New) {
				t.Errorf("%s must be listed as a by-hand edit, not a rule (rules: %d)", r.Old, n)
			}
		case n != 2:
			t.Errorf("%s -> %s: %d sed rules, want 2 (one per overlapping use)", r.Old, r.New, n)
		}
	}
}
