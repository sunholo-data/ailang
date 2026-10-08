package pipeline

import (
	"github.com/sunholo-data/ailang/internal/ast"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateTestRows(t *testing.T) {
	for _, expectedArm := range []bool{false, true} {
		lit := &ast.Literal{Kind: ast.IntLit, Value: int64(1)}
		bad := &ast.BinaryOp{Op: "+", Left: lit, Right: lit, Pos: ast.Pos{File: "row.ail", Line: 3, Column: 10}}
		row := &ast.TestCase{Inputs: []ast.Expr{lit}, Expected: lit}
		if expectedArm {
			row.Expected = bad
		} else {
			row.Inputs[0] = bad
		}
		err := validateTestRows(&ast.File{Funcs: []*ast.FuncDecl{{Name: "f", Tests: []*ast.TestCase{row}}}})
		rep, ok := ailerrors.AsReport(err)
		if !ok || rep.Code != "TST002" || rep.Span.Start.Line != 3 {
			t.Fatalf("got %v", err)
		}
	}
}

func TestRunnerCompileCannotCacheAwayRowValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cached_rows.ail")
	const source = `module cached_rows
export func id(x: int) -> int ! {}
tests [ (1 + 2, 3) ]
{ x }
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	src := Source{Filename: path, Code: source}
	if _, err := Run(Config{RelaxModules: true, SkipTestRowValidation: true}, src); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		_, err := Run(Config{RelaxModules: true}, src)
		rep, ok := ailerrors.AsReport(err)
		if !ok || rep.Code != "TST002" {
			t.Fatalf("run %d read false green: %v", i, err)
		}
	}
}
