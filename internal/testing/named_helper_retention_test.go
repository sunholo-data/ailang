package testing

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"os"
	"strings"
	"testing"
)

func TestNamedBatch_UnannotatedExport(t *testing.T) {
	path := writeEngineSource(t, "helpers.ail", `module helpers
import std/fs (readFile)
export func inc(x: int) -> int { x + 1 }
pure func twice(x: int) -> int = x * 2
func empty(x: int) -> int ! {} = x
func unused() -> string ! {FS} = readFile("does-not-exist")
test "helpers" { inc(1) == 2 && twice(2) == 4 && empty(3) == 3 }
property "increment" { forall(n: int) => inc(n) == n + 1 }
`)
	for _, bc := range []bool{false, true} {
		res, exec := runWithExecutor(t, path, bc, false)
		if byName(res)["helpers"].Status != StatusPass || propsByName(res)["increment"].Status != StatusPass {
			t.Fatalf("bytecode=%v: %+v %+v", bc, res.Tests, res.Properties)
		}
		// Force the property's single-entry fallback and compare seeded behavior.
		exec.batch = nil
		for _, d := range exec.sourceFile.Decls {
			if pd, ok := d.(*ast.PropertyDecl); ok {
				call, err := exec.forallCaller(pd.Property)
				if err != nil {
					t.Fatal(err)
				}
				value, err := call([]eval.Value{&eval.IntValue{Value: 42}})
				if err != nil {
					t.Fatal(err)
				}
				if !value.(*eval.BoolValue).Value {
					t.Fatalf("fallback value: %v", value)
				}
			}
		}
	}
}

func TestForall_UnannotatedHelperSeededFallback(t *testing.T) {
	path := writeEngineSource(t, "seedhelpers.ail", `module seedhelpers
export func inc(x: int) -> int { x + 1 }
property "small increment" { forall(n: int) => inc(n) < 50 }
`)
	batched, _ := runWithExecutor(t, path, false, false)
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	src = append(src, []byte("\ntest \"force fallback\" { inc(\"bad\") == 1 }\n")...)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatal(err)
	}
	fallback, _ := runWithExecutor(t, path, false, false)
	a, b := propsByName(batched)["small increment"], propsByName(fallback)["small increment"]
	if len(fallback.NamedBatchFailures) != 1 || a.Status != StatusFail || a.Status != b.Status || a.Seed != b.Seed || a.TestsRun != b.TestsRun || a.Error != b.Error || a.FailingInput != b.FailingInput {
		t.Fatalf("seeded batch/fallback mismatch: %+v / %+v", a, b)
	}
}

func TestStripTestBlocks_RetainsDeclarationsAndLines(t *testing.T) {
	src := `module retain
import std/fs (readFile)
export func inc(x: int) -> int { x + 1 }
func unused() -> string ! {FS} = readFile("missing")
test "first" { inc(1) == 2 }
pure func later() -> bool = true
property "second" { forall(n: int) => inc(n) == n + 1 }
`
	p := parser.New(lexer.New(src, "retain.ail"))
	file := p.ParseFile()
	if len(p.Errors()) != 0 {
		t.Fatal(p.Errors())
	}
	e := &Executor{}
	// Use the named-test base policy once implemented.
	got, lines := e.stripTestBlocks(src, file)
	if !strings.Contains(got, "export func inc") || !strings.Contains(got, "func unused") || strings.Contains(got, `test "`) || strings.Contains(got, `property "`) {
		t.Fatalf("incorrect retention: %s", got)
	}
	for i, line := range splitLines(got) {
		if line != splitLines(src)[lines[i]-1] {
			t.Fatalf("line map %v", lines)
		}
	}
}

func TestStripTestBlocks_PreservesContractAndAnnotationFixtures(t *testing.T) {
	for _, name := range []string{"named_test_contract.ail", "named_test_annotated.ail", "named_test_effectful.ail"} {
		_, src, file := parseStripFixture(t, name)
		got, lines := new(Executor).stripTestBlocks(src, file)
		original := splitLines(src)
		for i, line := range splitLines(got) {
			if line != original[lines[i]-1] {
				t.Fatalf("%s: line map %v", name, lines)
			}
		}
		for _, fn := range file.Funcs {
			if !strings.Contains(got, "func "+fn.Name) {
				t.Errorf("%s: missing %s", name, fn.Name)
			}
		}
		if strings.Contains(src, "@verify") && !strings.Contains(got, "@verify") {
			t.Error("annotation removed")
		}
		if strings.Contains(src, "requires {") && !strings.Contains(got, "requires {") {
			t.Error("contract removed")
		}
	}
}
