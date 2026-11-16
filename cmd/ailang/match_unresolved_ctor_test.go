package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// #2303: a constructor matched by name (#323: neither its type nor the
// constructor is imported) against a value whose type nothing fixes was
// accepted under a catch-all, and the back ends guessed its type: the VM took
// the first loaded ADT declaring the name, the Go emitter whichever its map
// yielded. With lib/x's `Color = Red | Green | Blue` and lib/y's
// `Light = Green | Red`, `isRed[a](x: a) = match x { Red => 1, _ => 0 }`
// printed [1, 0, 0, 1] on the evaluator, [1, 0, 1, 0] on both VMs (exit 0)
// and panicked in emitted Go. Such a pattern is now TC_MATCH_007 on every
// path, by every route that leaves the value's type unfixed; patterns whose
// type IS fixed — by an annotation, an import, the value matched, a deferred
// constructor choice (#2228) or an alias (#2244) — run identically on the
// evaluator, both VMs and emitted Go.

var byNameLibs = map[string]string{
	"lib/x.ail": "module lib/x\nexport type Color = Red | Green | Blue\n" +
		"export pure func mkc(i: int) -> Color = if i == 0 then Red else if i == 1 then Green else Blue\n",
	"lib/y.ail": "module lib/y\nexport type Light = Green | Red\n" +
		"export pure func mkl(i: int) -> Light = if i == 0 then Green else Red\n",
	// Declares no type: Color and Light reach its importers only transitively.
	"lib/w.ail": "module lib/w\nimport lib/x (mkc)\nimport lib/y (mkl)\n" +
		"export pure func c(i: int) -> int = 0\nexport pure func cc(i: int) = mkc(i)\nexport pure func ll(i: int) = mkl(i)\n",
	"lib/z.ail": "module lib/z\nimport lib/w (c)\n" +
		"export pure func isRedZ(x) -> int = match x { Red => 1, _ => 0 }\n",
	"lib/r.ail": "module lib/r\nimport std/result (Result, Ok, Err)\n" +
		"export pure func parse(s: string) -> Result[int, string] = if s == \"1\" then Ok(1) else Err(s)\n",
}

func writeByNameFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for _, m := range []map[string]string{byNameLibs, files} {
		for p, src := range m {
			full := filepath.Join(dir, filepath.FromSlash(p))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

func TestUnresolvedByNameCtorPatternRejectedOnEveryPath(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := buildAilang(t)
	const head = "import lib/w (cc, ll)\n\n"
	routes := []struct {
		name, file, src, at string // at: "file:line:" of the TC_MATCH_007
	}{
		{"rigid type variable (the issue's program)", "r1.ail",
			"module r1\n" + head + "pure func isRed[a](x: a) -> int = match x { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string = show([isRed(cc(0)), isRed(cc(1)), isRed(ll(0)), isRed(ll(1))])\n", "r1.ail:4:"},
		{"unannotated parameter", "r2.ail",
			"module r2\n" + head + "pure func isRed(x) -> int = match x { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string = show([isRed(cc(0)), isRed(ll(1))])\n", "r2.ail:4:"},
		{"let-bound lambda", "r3.ail",
			"module r3\n" + head + "export pure func summary() -> string {\n  let k = \\v. match v { Green => 10, _ => 20 };\n  show([k(cc(1)), k(ll(0))])\n}\n", "r3.ail:5:"},
		{"function literal", "r4.ail",
			"module r4\n" + head + "export pure func summary() -> string {\n  let k = func(v) -> int { match v { Green => 10, _ => 20 } };\n  show([k(cc(1)), k(ll(0))])\n}\n", "r4.ail:5:"},
		{"nested in a tuple pattern", "r5.ail",
			"module r5\n" + head + "pure func firstRed(p) -> int = match p { (Red, n) => n, _ => 0 }\n\n" +
				"export pure func summary() -> string = show([firstRed((cc(0), 1)), firstRed((ll(1), 4))])\n", "r5.ail:4:"},
		{"lambda passed to a HOF", "r6.ail",
			"module r6\nimport std/list (filter, length)\n" + head +
				"pure func countRed[a](xs: [a]) -> int = length(filter(\\v. match v { Red => true, _ => false }, xs))\n\n" +
				"export pure func summary() -> string = show([countRed([cc(0), cc(1)]), countRed([ll(0), ll(1)])])\n", "r6.ail:5:"},
		{"another module's function", "r7.ail",
			"module r7\nimport lib/w (cc, ll)\nimport lib/z (isRedZ)\n\n" +
				"export pure func summary() -> string = show([isRedZ(cc(0)), isRedZ(ll(1))])\n", "lib/z.ail:3:"},
		{"contract clause", "r8.ail",
			"module r8\n" + head + "pure func keep[a](x: a) -> int\nrequires { match x { Red => true, _ => false } }\n{ 1 }\n\n" +
				"export pure func summary() -> string = show(keep(cc(0)))\n", "r8.ail:5:"},
		{"test body", "r9.ail",
			"module r9\n" + head + "export pure func summary() -> string = show(cc(0))\n\n" +
				"test \"by-name in a test body\" {\n  let k = \\v. match v { Red => 1, _ => 0 };\n  assert k(cc(0)) == 1\n}\n", "r9.ail:7:"},
	}
	files := map[string]string{}
	for _, r := range routes {
		files[r.file] = r.src
	}
	dir := writeByNameFixture(t, files)
	run := func(args ...string) (string, int) {
		stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, bin, args...)
		return filepath.ToSlash(stdout + stderr), code
	}
	for _, r := range routes {
		t.Run(r.name, func(t *testing.T) {
			stdout, _, code := testutil.RunBounded(t, dir, 90*time.Second, bin, "check", "--json", r.file)
			var out checkJSONOutput
			if err := json.Unmarshal([]byte(stdout), &out); err != nil || code != 1 || out.Passed || len(out.Errors) == 0 {
				t.Fatalf("check --json: exit %d, %v\n%s", code, err, stdout)
			}
			if e := out.Errors[0]; e.Code != "TC_MATCH_007" || !strings.Contains(e.Message, "cannot tell which type") {
				t.Errorf("check --json: first error %+v, want TC_MATCH_007", e)
			}
			paths := [][]string{
				{"check", r.file},
				{"run", "--quiet", "--entry", "summary", r.file},
				{"run", "--quiet", "--bytecode", "--entry", "summary", r.file},
				{"run", "--quiet", "--strict-bytecode", "--entry", "summary", r.file},
				{"compile", "--emit-go", "--out", t.TempDir(), r.file},
			}
			for _, args := range paths {
				all, code := run(args...)
				if code == 0 || !strings.Contains(all, r.at) || !strings.Contains(all, "TC_MATCH_007") ||
					strings.Contains(all, "panic:") {
					t.Errorf("%s: exit %d, want TC_MATCH_007 at %s before anything runs:\n%s", strings.Join(args, " "), code, r.at, all)
				}
			}
		})
	}
}

// Negative controls: by-name and imported constructor patterns whose type is
// fixed give the same answer on the evaluator, both VMs and emitted Go (built
// and run), and the Go emitted for them is the same on every emit.
func TestFixedCtorPatternsAgreeOnEveryEngine(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := buildAilang(t)
	cases := []struct {
		name, module, src, want string
	}{
		{"the issue's shape with the type imported", "ok1",
			"module ok1\nimport lib/w (cc, ll)\nimport lib/x (Color)\n\n" +
				"pure func isRed(x) -> int = match x { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string {\n  let k = \\v. match v { Green => 10, _ => 20 };\n" +
				"  show([isRed(cc(0)), isRed(cc(1)), k(cc(1)), k(cc(0))])\n}\n", "[1, 0, 10, 20]"},
		{"by-name on a fixed type: top level, tuple, Option, guard", "ok2",
			"module ok2\nimport std/list (nth)\nimport std/option (Option, Some, None)\nimport lib/w (cc, ll)\n\n" +
				"export pure func summary() -> string {\n" +
				"  let a = match cc(0) { Red => 1, _ => 0 };\n" +
				"  let b = match ll(1) { Red => 1, _ => 0 };\n" +
				"  let c = match ll(0) { Red => 1, Green => 2 };\n" +
				"  let e = match nth([ll(1)], 0) { Some(Red) => 1, Some(_) => 2, None => 3 };\n" +
				"  let f = match cc(0) { Red if a == 0 => 9, Red => 1, _ => 0 };\n" +
				"  let g = match ll(1) { Red if a == 0 => 9, Red => 1, _ => 0 };\n" +
				"  show([a, b, c, e, f, g])\n}\n", "[1, 1, 2, 1, 1, 1]"},
		// A by-name constructor inside a tuple pattern, on a fixed type. The
		// emitted Go does not build for any constructor in a tuple (#2358,
		// pre-existing), so this one runs on the evaluator and the VMs only.
		{"by-name in a tuple on a fixed type (no Go: #2358)", "ok6",
			"module ok6\nimport lib/w (cc, ll)\n\n" +
				"export pure func summary() -> string = show([match (cc(1), 5) { (Green, n) => n, _ => 0 }, match (ll(0), 6) { (Green, n) => n, _ => 0 }])\n",
			"[5, 6]"},
		{"deferred constructor choice (#2228)", "ok3",
			"module ok3\nimport lib/x (Color, mkc)\nimport lib/y (Light, mkl)\n\n" +
				"pure func f(c: Color) -> int = match c { Red => 1, _ => 0 }\n" +
				"pure func g(l: Light) -> int = match l { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string = show([f(mkc(0)), f(mkc(1)), g(mkl(0)), g(mkl(1))])\n", "[1, 0, 0, 1]"},
		{"alias of an ADT (#2244)", "ok4",
			"module ok4\nimport lib/x (Color, mkc)\n\ntype Hue = Color\n\n" +
				"pure func f(h: Hue) -> int = match h { Red => 1, Green => 2, Blue => 3 }\n\n" +
				"export pure func summary() -> string = show([f(mkc(0)), f(mkc(1)), f(mkc(2))])\n", "[1, 2, 3]"},
		{"prelude Option and Result by name, from library modules", "ok5",
			"module ok5\nimport std/list (nth)\nimport lib/r (parse)\n\n" +
				"export pure func summary() -> string {\n" +
				"  let a = match nth([1, 2], 1) { Some(v) => v, None => 0 };\n" +
				"  let b = match parse(\"1\") { Ok(n) => n, Err(_) => 0 };\n" +
				"  let c = match parse(\"x\") { Ok(n) => n, Err(_) => 7 };\n" +
				"  show([a, b, c])\n}\n", "[2, 1, 7]"},
	}
	files := map[string]string{}
	for _, c := range cases {
		files[c.module+".ail"] = c.src
	}
	dir := writeByNameFixture(t, files)
	libs := make([]string, 0, len(byNameLibs))
	for p := range byNameLibs {
		if p != "lib/z.ail" { // the cross-module route: rejected
			libs = append(libs, p)
		}
	}
	sort.Strings(libs)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			file := c.module + ".ail"
			for _, flags := range [][]string{nil, {"--bytecode"}, {"--strict-bytecode"}} {
				args := append(append([]string{"run", "--quiet"}, flags...), "--entry", "summary", file)
				stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, bin, args...)
				if got := strings.TrimSpace(stdout); code != 0 || got != c.want || strings.Contains(stderr, "falling back") {
					t.Errorf("run %v: exit %d, got %q, want %q\n%s", flags, code, got, c.want, stderr)
				}
			}
			if c.module == "ok6" {
				return
			}
			emit := func(out string) string {
				args := append([]string{"compile", "--emit-go", "--package-name", c.module, "--out", out, file}, libs...)
				if stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, bin, args...); code != 0 {
					t.Fatalf("compile --emit-go: exit %d\n%s\n%s", code, stdout, stderr)
				}
				return filepath.Join(out, c.module)
			}
			gen := emit(filepath.Join(t.TempDir(), "go"))
			if c.module == "ok2" {
				// The bare-name lookup this replaced iterated a Go map.
				first := goSources(t, gen)
				for i := 0; i < 32; i++ {
					if again := goSources(t, emit(filepath.Join(t.TempDir(), "go"))); again != first {
						t.Fatalf("emit %d differs from the first", i+1)
					}
				}
			}
			if err := os.WriteFile(filepath.Join(gen, "go.mod"), []byte("module "+c.module+"\n\ngo 1.21\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			test := "package " + c.module + "\nimport \"testing\"\nfunc TestSummary(t *testing.T) {\n if got := " + c.module +
				"__Summary(); got != " + quoteGo(c.want) + " { t.Fatalf(\"generated Go: %v\", got) }\n}\n"
			if err := os.WriteFile(filepath.Join(gen, "summary_test.go"), []byte(test), 0o644); err != nil {
				t.Fatal(err)
			}
			if stdout, stderr, code := testutil.RunBounded(t, gen, 120*time.Second, "go", "test", "-count=1", "."); code != 0 {
				t.Errorf("generated Go disagrees with the evaluator or does not build: exit %d\n%s\n%s", code, stdout, stderr)
			}
		})
	}
}

// goSources concatenates the .go files under dir in name order.
func goSources(t *testing.T, dir string) string {
	t.Helper()
	var names []string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(p, ".go") {
			names = append(names, p)
		}
		return nil
	})
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		data, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString(strings.TrimPrefix(n, dir))
		b.Write(data)
	}
	return b.String()
}
