package testing

// Regression tests for module-scoped name resolution in the test harness
// (#1461, #1516; design m-test-harness-module-scoped-envs).
//
// `ailang run` evaluates every module in its own environment, so a module's
// private helpers are visible only inside that module. The harness used to
// flatten every loaded module's functions into ONE shared environment under
// their bare names, so the module that sorted last won:
//
//   #1461 — two imported modules each define a private `helper`; whichever
//           sorted last served BOTH modules ("record has no field: s").
//   #1516 — the module under test defines a private `isErr`/`words`; the
//           stdlib module std/result / std/string (loaded as a dependency,
//           never imported for those names) sorted after it and replaced it
//           ("no pattern matched", "_str_words: expected String").

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/runtime"
)

// #1516: a private helper in the module under test must not be replaced by a
// same-named export of a stdlib module the file loads but did not import that
// name from. Exercised on the named-test path and the inline-test path.
func TestPrivateHelperNotReplacedByNonImportedStdlibExport(t *testing.T) {
	source := `module shadow

import std/result (Result, Ok, Err)
import std/string (join)

type Box = { n: int }

pure func isErr(b: Box) -> bool = b.n < 0

pure func words(xs: [string]) -> [string] = xs

pure func checkIt() -> bool = !isErr({ n: 1 }) && join(",", words(["a", "b"])) == "a,b"

pure func viaInline(n: int) -> bool
  tests [
    (1, true)
  ]
  { !isErr({ n: n }) && join(",", words(["a", "b"])) == "a,b" }

test "local helpers are not replaced by non-imported stdlib names" { checkIt() }

test "a test body naming the private helper directly gets the local one" { !isErr({ n: 1 }) }
`
	// Run from the file's directory with a RELATIVE path, exactly as the CLI
	// is used (`ailang test shadow.ail`). The harness's module keys derive from
	// file paths, and a relative path is what made the root module sort BEFORE
	// std/* — an absolute temp path ("var/folders/...") happened to sort after
	// it and masked the bug.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shadow.ail"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	result := runTestsOnFile(t, "shadow.ail", source)
	if result.FailedTests > 0 {
		t.Fatalf("expected 0 failures, got %d; first error: %s",
			result.FailedTests, firstFailureError(result))
	}
	if result.PassedTests != 3 {
		t.Fatalf("expected 3 passing tests (2 named + inline), got %d", result.PassedTests)
	}
}

// writeHelperCollisionPackage lays out the #1461 reproducer: two modules that
// each define a private `helper` with a different record shape, and a test
// module that imports one export from each. first/second name the two
// helper-defining modules so a caller can flip their lexical sort order.
func writeHelperCollisionPackage(t *testing.T, first, second string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"ailang.toml": "[package]\nname = \"x/pkg\"\nversion = \"0.1.0\"\nedition = \"1\"\n",
		first + ".ail": "module x/pkg/" + first + "\n\n" +
			"pure func helper(r: {n: int}) -> int = r.n + 1\n\n" +
			"export pure func fa(x: int) -> int = helper({n: x})\n",
		second + ".ail": "module x/pkg/" + second + "\n\n" +
			"pure func helper(r: {s: string}) -> string = r.s\n\n" +
			"export pure func fb(x: string) -> string = helper({s: x})\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// #1461: same-named private helpers in two imported modules each resolve to
// their own module's definition — under BOTH lexical orders of the module
// names (the bug was order-deterministic: the last-sorted module won).
func TestSameNamedPrivateHelpersInTwoModules(t *testing.T) {
	for _, order := range [][2]string{{"a", "b"}, {"z", "b"}} {
		order := order
		t.Run(order[0]+"_"+order[1], func(t *testing.T) {
			dir := writeHelperCollisionPackage(t, order[0], order[1])
			testSrc := "module x/pkg/t_test\n\n" +
				"import x/pkg/" + order[0] + " (fa)\n" +
				"import x/pkg/" + order[1] + " (fb)\n\n" +
				"pure func checkBoth() -> bool = fa(1) == 2 && fb(\"x\") == \"x\"\n\n" +
				"pure func inlineBoth(n: int) -> bool\n" +
				"  tests [\n    (1, true)\n  ]\n" +
				"  { fa(n) == 2 && fb(\"x\") == \"x\" }\n\n" +
				"test \"both\" { checkBoth() }\n"
			path := filepath.Join(dir, "t_test.ail")
			if err := os.WriteFile(path, []byte(testSrc), 0o644); err != nil {
				t.Fatal(err)
			}
			result := runTestsOnFile(t, path, testSrc)
			if result.FailedTests > 0 {
				t.Fatalf("expected 0 failures, got %d; first error: %s",
					result.FailedTests, firstFailureError(result))
			}
			if result.PassedTests != 2 {
				t.Fatalf("expected 2 passing tests (named + inline), got %d", result.PassedTests)
			}
		})
	}
}

// A qualified reference whose qualified key is missing falls back to the
// OWNING module's own bindings — never to a same-named bare binding in the
// shared env, which belongs to the module under test.
func TestCombinedResolverQualifiedFallbackIsModuleScoped(t *testing.T) {
	shared := eval.NewEnvironment()
	rootHelper := &eval.StringValue{Value: "root helper"}
	ownHelper := &eval.StringValue{Value: "x/a helper"}
	shared.Set("helper", rootHelper)
	r := &CombinedResolver{
		Builtins: runtime.NewBuiltinRegistry(eval.NewCoreEvaluator()),
		Env:      shared,
		Modules: map[string]*loader.LoadedModule{
			"x/a": {Core: &core.Program{}},
			"x/b": {Core: &core.Program{}},
		},
		ModuleBindings: map[string]map[string]eval.Value{
			"x/a": {"helper": ownHelper},
			"x/b": {},
		},
	}
	got, err := r.ResolveValue(core.GlobalRef{Module: "x/a", Name: "helper"})
	if err != nil || got != ownHelper {
		t.Fatalf("x/a.helper resolved to %v (err %v), want the owning module's binding", got, err)
	}
	if got, err := r.ResolveValue(core.GlobalRef{Module: "x/b", Name: "helper"}); err == nil {
		t.Fatalf("x/b.helper resolved to %v; x/b has no helper, so it must fail loudly, not borrow the root's", got)
	}
}
