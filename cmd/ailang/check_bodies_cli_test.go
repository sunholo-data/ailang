package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// #2232/#2248 review round: the body checks end to end through the binary —
// check, run on both engines, and test — for the shapes a reviewer found.

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, src := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

type cliRun struct {
	out  string
	code int
}

func ailangIn(t *testing.T, bin, dir string, args ...string) cliRun {
	t.Helper()
	stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, bin, args...)
	return cliRun{out: stdout + stderr, code: code}
}

func (r cliRun) want(t *testing.T, what string, code int, has ...string) {
	t.Helper()
	if r.code != code {
		t.Fatalf("%s: exit %d, want %d\n%s", what, r.code, code, r.out)
	}
	if strings.Contains(r.out, "panic") || strings.Contains(r.out, "goroutine ") {
		t.Fatalf("%s panicked:\n%s", what, r.out)
	}
	for _, h := range has {
		if !strings.Contains(r.out, h) {
			t.Fatalf("%s: output lacks %q\n%s", what, h, r.out)
		}
	}
}

// `forall` outside a contract is TC_FORALL_001 on every command (it used to
// type-check there and then panic in ValidateCoreTypeInfo).
func TestCLI_ForallOutsideContract(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	dir := writeTree(t, map[string]string{
		"body.ail":   "module body\nimport std/io (println)\nexport pure func allPos(n: int) -> bool = forall i: 0..n => i >= 0\nexport func main() -> () ! {IO} = println(show(allPos(3)))\n",
		"inlet.ail":  "module inlet\nimport std/io (println)\nexport func main() -> () ! {IO} {\n  let ok = forall i: 0..3 => i >= 0;\n  println(show(ok))\n}\n",
		"intest.ail": "module intest\nexport pure func k(x: int) -> int = x\ntest \"q\" { forall i: 0..3 => k(i) >= 0 }\n",
		"ok.ail":     "module ok\nimport std/io (println)\nexport func f(n: int) -> int ! {}\nrequires { forall i: 0..n => i >= 0 }\n{ n }\nexport func main() -> () ! {IO} = println(show(f(2)))\n",
	})
	ailangIn(t, bin, dir, "check", "body.ail").want(t, "check body", 1, "TC_FORALL_001", "body.ail:3:")
	ailangIn(t, bin, dir, "run", "--caps", "IO", "body.ail").want(t, "run body", 1, "TC_FORALL_001")
	ailangIn(t, bin, dir, "run", "--bytecode", "--caps", "IO", "body.ail").want(t, "run --bytecode body", 1, "TC_FORALL_001")
	ailangIn(t, bin, dir, "run", "--caps", "IO", "inlet.ail").want(t, "run let", 1, "TC_FORALL_001", "inlet.ail:4:")
	ailangIn(t, bin, dir, "check", "intest.ail").want(t, "check test body", 1, "TC_TEST_001", "TC_FORALL_001")
	ailangIn(t, bin, dir, "test", "intest.ail").want(t, "test test body", 1, "TC_FORALL_001")
	ailangIn(t, bin, dir, "run", "--caps", "IO", "ok.ail").want(t, "forall in requires", 0, "2")
}

// A dependency's ill-typed test block or row is its own business: importers
// check, run and test; checking the dependency itself still reports it, warm
// or cold.
func TestCLI_DependencyTestBodiesDoNotFailImporters(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	dir := writeTree(t, map[string]string{
		"lib/dept.ail": "module lib/dept\nexport pure func tri(x: int) -> int = x * 3\ntest \"bad\" { tri(\"a\") == 3 }\n",
		"lib/dep.ail":  "module lib/dep\nexport pure func inc(x: int) -> int\n  tests [(1, \"two\")]\n{ x + 1 }\n",
		"app.ail":      "module app\nimport std/io (println)\nimport lib/dept (tri)\nimport lib/dep (inc)\nexport func main() -> () ! {IO} = println(show(tri(inc(1))))\n",
		"app_t.ail":    "module app_t\nimport lib/dept (tri)\ntest \"uses tri\" { tri(1) == 3 }\n",
	})
	for round := 0; round < 2; round++ { // cold, then warm
		ailangIn(t, bin, dir, "check", "app.ail").want(t, "check importer", 0)
		ailangIn(t, bin, dir, "run", "--caps", "IO", "app.ail").want(t, "run importer", 0, "6")
		ailangIn(t, bin, dir, "run", "--bytecode", "--caps", "IO", "app.ail").want(t, "run --bytecode importer", 0, "6")
		ailangIn(t, bin, dir, "test", "app_t.ail").want(t, "test importer", 0)
		ailangIn(t, bin, dir, "check", "lib/dept.ail").want(t, "check dependency (test block)", 1, "TC_TEST_001")
		ailangIn(t, bin, dir, "check", "lib/dep.ail").want(t, "check dependency (row)", 1, "TC_TEST_001")
	}
}

// One ill-typed contract makes the module fail to compile: `ailang test`
// reports that once, as itself, instead of one "failed to extract function
// binding" per test.
func TestCLI_TestReportsModuleCompileErrorOnce(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	dir := writeTree(t, map[string]string{
		"mix.ail": "module mix\nexport pure func dbl(x: int) -> int\n  tests [(1, 2), (2, 4)]\n{ x * 2 }\nexport pure func neg(x: int) -> int\n  requires { x + \"s\" > 0 }\n{ 0 - x }\ntest \"fine\" { dbl(2) == 4 }\n",
		// rows only (the shared inline compile) and named tests only (the batch)
		"rows.ail":  "module rows\nexport pure func dbl(x: int) -> int\n  tests [(1, 2), (2, 4)]\n{ x * 2 }\nexport pure func neg(x: int) -> int\n  requires { x + \"s\" > 0 }\n{ 0 - x }\n",
		"named.ail": "module named\nexport pure func dbl(x: int) -> int = x * 2\nexport pure func neg(x: int) -> int\n  requires { x + \"s\" > 0 }\n{ 0 - x }\ntest \"a\" { dbl(2) == 4 }\ntest \"b\" { dbl(3) == 6 }\n",
		// a broken DEPENDENCY: the tested module cannot compile either
		"lib/broken.ail": "module lib/broken\nexport pure func h(x: int) -> int = x + \"s\"\n",
		"depbroken.ail":  "module depbroken\nimport lib/broken (h)\nexport pure func g(x: int) -> int\n  tests [(2, 2)]\n{ x }\ntest \"uses h\" { h(1) == 1 }\ntest \"independent\" { g(2) == 2 }\n",
		"depnamed.ail":   "module depnamed\nimport lib/broken (h)\ntest \"uses h\" { h(1) == 1 }\ntest \"other\" { 1 == 1 }\n",
		// a broken TEST body is still that test's failure alone
		"onebad.ail": "module onebad\nexport pure func dbl(x: int) -> int = x * 2\ntest \"good\" { dbl(2) == 4 }\ntest \"bad\" { dbl(\"x\") == 4 }\n",
	})
	for _, f := range []string{"mix", "rows", "named"} {
		r := ailangIn(t, bin, dir, "test", f+".ail")
		r.want(t, "test "+f, 1, "module "+f+" does not compile", "TC_CONTRACT_001", f+".ail:")
		if strings.Contains(r.out, "failed to extract function binding") || strings.Count(r.out, "TC_CONTRACT_001") != 1 {
			t.Fatalf("%s: the module error must be reported once, as itself:\n%s", f, r.out)
		}
	}
	for _, f := range []string{"depbroken", "depnamed"} {
		r := ailangIn(t, bin, dir, "test", f+".ail")
		r.want(t, "test "+f, 1, "module "+f+" does not compile", filepath.FromSlash("lib/broken.ail")+":2:")
		if strings.Contains(r.out, "failed to extract function binding") || strings.Count(r.out, "cannot unify") != 1 {
			t.Fatalf("%s: a dependency's compile error must be reported once:\n%s", f, r.out)
		}
	}
	r := ailangIn(t, bin, dir, "test", "onebad.ail")
	r.want(t, "test onebad", 1, "1 passed", "1 failed")
	if strings.Contains(r.out, "does not compile") {
		t.Fatalf("a broken test body must not be reported as the module's error:\n%s", r.out)
	}
}

// check and the harness agree on a named test whose single final expression
// is not a bool: the runner fails it ("expected bool result"), and check
// reports it first.
func TestCLI_NonBoolFinalTestExpression(t *testing.T) {
	bin := buildAilang(t)
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	dir := writeTree(t, map[string]string{
		"nb.ail": "module nb\nexport pure func f(x: int) -> int = x\ntest \"nonbool final\" { f(1) }\n",
	})
	ailangIn(t, bin, dir, "check", "nb.ail").want(t, "check", 1, "TC_TEST_001", `test "nonbool final"`)
	ailangIn(t, bin, dir, "test", "nb.ail").want(t, "test", 1, "expected bool result")
}
