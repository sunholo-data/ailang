package testing

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
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
