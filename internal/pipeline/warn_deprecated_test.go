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
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// TC_DEPRECATED_001 (M-V1-STDLIB-SURFACE-POLISH §6.1): one warning per use
// site of an @deprecated function, from another module, on a cold and a warm
// compile cache.

const deprecatedLib = `module lib

export pure func g(x: int) -> int = x + 1

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
		{"importOnly", "module importOnly\nimport lib (f)\nexport pure func k() -> int = 1\n", nil},
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

// stdDeprecations parses every std/*.ail and returns its @deprecated exports.
func stdDeprecations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	stdDir := stdSourceDir(t)
	files, err := filepath.Glob(filepath.Join(stdDir, "*.ail"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*ast.FuncDecl{}
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
		mod := "std/" + strings.TrimSuffix(filepath.Base(f), ".ail")
		for _, fd := range prog.File.Funcs {
			if _, _, ok := fd.Deprecation(); ok && fd.IsExport {
				out[mod+"."+fd.Name] = fd
			}
		}
	}
	return out
}

// The removed-name table (importhint.StdlibRenames, which outlives the aliases)
// and the @deprecated annotations in std/ say the same thing.
func TestStdlibRenamesMatchDeprecatedAnnotations(t *testing.T) {
	deps := stdDeprecations(t)
	seen := map[string]bool{}
	for _, r := range importhint.StdlibRenames {
		key := r.Module + "." + r.Old
		seen[key] = true
		fd, ok := deps[key]
		if !ok {
			t.Errorf("%s is in importhint.StdlibRenames but std/ has no @deprecated %s", key, key)
			continue
		}
		repl, note, _ := fd.Deprecation()
		if repl != r.New || note != r.Note {
			t.Errorf("%s: annotation says (%q, %q), table says (%q, %q)", key, repl, note, r.New, r.Note)
		}
	}
	for key := range deps {
		if !seen[key] {
			t.Errorf("@deprecated %s is missing from importhint.StdlibRenames", key)
		}
	}
	if len(importhint.StdlibRenames) != 16 {
		t.Errorf("%d renames, the design doc lists 16 (R1-R14, D1, D2)", len(importhint.StdlibRenames))
	}
}

// Every rename alias is a one-line forwarder to EXACTLY its replacement,
// with its own parameters in the same order (a forwarder to the wrong target,
// or with swapped arguments, type-checks when the parameters share a type).
// member and unsafeGet are deprecated, not renamed: they keep their own body.
func TestDeprecatedAliasesForwardToTheirReplacement(t *testing.T) {
	deps := stdDeprecations(t)
	for _, r := range importhint.StdlibRenames {
		key := r.Module + "." + r.Old
		fd := deps[key]
		if fd == nil {
			continue // reported by TestStdlibRenamesMatchDeprecatedAnnotations
		}
		if r.Swapped || r.Old == "unsafeGet" {
			continue
		}
		body := fd.Body
		if b, isBlock := body.(*ast.Block); isBlock && len(b.Exprs) == 1 {
			body = b.Exprs[0] // `= expr` parses as a one-expression block
		}
		call, ok := body.(*ast.FuncCall)
		if !ok {
			t.Errorf("%s: body is %T, want a call to %s", key, fd.Body, r.New)
			continue
		}
		callee, ok := call.Func.(*ast.Identifier)
		if !ok || callee.Name != r.New {
			t.Errorf("%s forwards to %v, want %s", key, call.Func, r.New)
			continue
		}
		if len(call.Args) != len(fd.Params) {
			t.Errorf("%s: %d args for %d params", key, len(call.Args), len(fd.Params))
			continue
		}
		for i, a := range call.Args {
			if lit, isLit := a.(*ast.Literal); isLit && lit.Kind == ast.UnitLit && fd.Params[i].Name == "_" {
				continue // f() forwarding g(): the parser's unit parameter on both sides
			}
			id, ok := a.(*ast.Identifier)
			if !ok || id.Name != fd.Params[i].Name {
				t.Errorf("%s: argument %d is %T %v, want parameter %s", key, i, a, a, fd.Params[i].Name)
			}
		}
	}
}

// tools/migrate-stdlib-renames.sed rewrites every drop-in rename whose old name
// is a unique token, and documents the four done by hand.
func TestMigrateSedCoversRenames(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(stdSourceDir(t), "..", "tools", "migrate-stdlib-renames.sed"))
	if err != nil {
		t.Fatal(err)
	}
	sed := string(data)
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
