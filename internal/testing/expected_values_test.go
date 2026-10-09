package testing

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
	"os"
	"strings"
	"testing"
)

func TestExpectedCompositeValues(t *testing.T) {
	lit := &ast.Literal{Kind: ast.IntLit, Value: int64(2)}
	cases := []struct {
		name string
		expr ast.Expr
		want eval.Value
	}{
		{"list", &ast.List{Elements: []ast.Expr{lit}}, &eval.ListValue{Elements: []eval.Value{&eval.IntValue{Value: 2}}}},
		{"empty", &ast.List{}, &eval.ListValue{}},
		{"tuple", &ast.Tuple{Elements: []ast.Expr{lit, lit}}, &eval.TupleValue{Elements: []eval.Value{&eval.IntValue{Value: 2}, &eval.IntValue{Value: 2}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewExecutor("")
			got, err := e.EvaluateExpectedExpr(tc.expr)
			if err != nil {
				t.Fatal(err)
			}
			if !e.CompareValues(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			if e.CompareValues(got, &eval.IntValue{Value: 99}) {
				t.Fatal("mismatch passed")
			}
		})
	}
}

func TestInlineExpectedValuesFixture(t *testing.T) {
	source, err := os.ReadFile("testdata/inline_expected_values.ail")
	if err != nil {
		t.Fatal(err)
	}
	res := runInlineTestsOnSource(t, string(source))
	if res.TotalTests != 10 {
		t.Fatalf("expected 10 rows, got %d", res.TotalTests)
	}
	for _, result := range res.Tests {
		if result.Status != StatusPass {
			t.Fatalf("%s: %s", result.Name, result.Error)
		}
	}
}

func TestUnsupportedRowsFailBeforeCompilation(t *testing.T) {
	for _, tc := range []struct{ file, code string }{{"inline_rows_binary.ail", "TST002"}, {"inline_rows_unsupported.ail", "TST001"}} {
		source, err := os.ReadFile("testdata/" + tc.file)
		if err != nil {
			t.Fatal(err)
		}
		res := runInlineTestsOnSource(t, string(source))
		for _, result := range res.Tests {
			if result.Status != StatusFail || !strings.Contains(result.Error, tc.code) {
				t.Fatalf("%+v", result)
			}
		}
	}
	// No source file or function binding exists: syntax rejection must precede both.
	result := NewRunner("").runTest(TestCase{IsInline: true, Body: []ast.Expr{&ast.Lambda{}}})
	if !strings.Contains(result.Error, "TST001") {
		t.Fatalf("%+v", result)
	}
}

func TestHarnessConversionReturnsErrors(t *testing.T) {
	for _, expr := range []ast.Expr{&ast.Lambda{}, &ast.List{Elements: []ast.Expr{&ast.Lambda{}}}} {
		if _, err := astExprToCore(expr); err == nil {
			t.Fatalf("accepted %T", expr)
		}
		row := TestCase{Body: []ast.Expr{&ast.Tuple{Elements: []ast.Expr{expr, &ast.Literal{Kind: ast.UnitLit}}}}}
		if _, err := BuildInlineTestHarness(core.RecBinding{Name: "f"}, []TestCase{row}); err == nil {
			t.Fatal("single harness accepted unsupported input")
		}
		if _, err := BuildClusterTestHarness(&PureCluster{FuncName: "f"}, []TestCase{row}); err == nil {
			t.Fatal("cluster harness accepted unsupported input")
		}
	}
}

// Grammar/evaluation parity uses real bindings: acceptance guarantees syntax,
// while unresolved names and ill-typed calls remain runtime errors.
func TestRowGrammarEvaluationParity(t *testing.T) {
	_, r := newRoundTripEvaluator(t)
	lit := &ast.Literal{Kind: ast.IntLit, Value: int64(2)}
	exprs := []ast.Expr{
		lit, &ast.Literal{Kind: ast.UnitLit}, &ast.Literal{Kind: ast.FloatLit, Value: 2.5},
		&ast.Literal{Kind: ast.StringLit, Value: "hi"}, &ast.Literal{Kind: ast.BoolLit, Value: true},
		&ast.Identifier{Name: "SPRING"},
		&ast.FuncCall{Func: &ast.Identifier{Name: "Para"}, Args: []ast.Expr{&ast.Literal{Kind: ast.StringLit, Value: "hi"}}},
		&ast.Tuple{Elements: []ast.Expr{lit, lit}}, &ast.List{Elements: []ast.Expr{lit}},
		&ast.Record{Fields: []*ast.Field{{Name: "value", Value: lit}}},
		&ast.UnaryOp{Op: "-", Expr: lit},
		&ast.BinaryOp{Op: "+", Left: lit, Right: lit}, &ast.Lambda{}, &ast.FuncLit{},
		&ast.Let{}, &ast.LetRec{}, &ast.Block{}, &ast.If{}, &ast.Match{}, &ast.Array{},
		&ast.RecordAccess{}, &ast.RecordUpdate{}, &ast.Error{}, &ast.QuasiQuote{},
		&ast.Send{}, &ast.Recv{}, &ast.ForallExpr{},
		&ast.List{Elements: []ast.Expr{&ast.Lambda{}}},
	}
	for _, expr := range exprs {
		problem := ast.RowExprSupported(expr)
		_, err := r.executor.EvaluateExpectedExpr(expr)
		if (problem == nil) != (err == nil) {
			t.Errorf("%T: grammar=%v evaluation=%v", expr, problem, err)
		}
	}
}

func TestCompositeExpectedMismatchFailsRow(t *testing.T) {
	res := runInlineTestsOnSource(t, `module mismatch
export func values(x: int) -> list[int] ! {}
tests [ (2, [3]) ]
{ [x] }
`)
	if res.TotalTests != 1 || res.FailedTests != 1 || !strings.Contains(res.Tests[0].Error, "expected") {
		t.Fatalf("%+v", res)
	}
}

func TestInvalidRowDoesNotContaminateNeighbors(t *testing.T) {
	res := runInlineTestsOnSource(t, `module mixed_rows
export func id(x: int) -> int ! {}
tests [ (1, 1), (2, 1 + 1) ]
{ x }
test "neighbor" { id(3) == 3 }
`)
	if res.TotalTests != 3 || res.PassedTests != 2 || res.FailedTests != 1 {
		t.Fatalf("invalid row contaminated neighbors: %+v", res)
	}
}
