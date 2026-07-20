package format

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// node_coverage_test.go exercises EVERY concrete AST node kind's printer,
// including nodes the current parser does not emit (e.g. TypeClass, Instance,
// QuasiQuote, Send, Recv, ForallExpr, Interpolation-free quasiquotes). Each node
// is constructed directly and rendered via a small harness, proving there is a
// real printer (or an explicit error) for every kind and no debug String()
// fallback. This is the exhaustive-coverage release gate for M1.

// renderExpr formats a single expression by wrapping it in a top-level let value.
func renderExpr(t *testing.T, e ast.Expr) (string, error) {
	t.Helper()
	prog := &ast.Program{File: &ast.File{
		Module: &ast.ModuleDecl{Path: "m"},
		Decls:  []ast.Node{&ast.Let{Name: "v", Value: e}},
	}}
	out, err := Source(prog, Options{})
	return string(out), err
}

// renderDecl formats a single top-level declaration node.
func renderDecl(t *testing.T, d ast.Node) (string, error) {
	t.Helper()
	prog := &ast.Program{File: &ast.File{
		Module: &ast.ModuleDecl{Path: "m"},
		Decls:  []ast.Node{d},
	}}
	out, err := Source(prog, Options{})
	return string(out), err
}

func mustContain(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("expected output to contain %q, got:\n%s", want, got)
	}
}

// TestExprNodeCoverage renders every concrete expression node.
func TestExprNodeCoverage(t *testing.T) {
	id := &ast.Identifier{Name: "x"}
	one := &ast.Literal{Kind: ast.IntLit, Value: int64(1)}
	two := &ast.Literal{Kind: ast.IntLit, Value: int64(2)}

	cases := []struct {
		name string
		expr ast.Expr
		want string
	}{
		{"Identifier", id, "v = x"},
		{"IntLit", one, "v = 1"},
		{"FloatLit", &ast.Literal{Kind: ast.FloatLit, Value: 3.5}, "v = 3.5"},
		{"FloatWhole", &ast.Literal{Kind: ast.FloatLit, Value: 4.0}, "v = 4.0"},
		{"StringLit", &ast.Literal{Kind: ast.StringLit, Value: "hi"}, `v = "hi"`},
		{"BoolLit", &ast.Literal{Kind: ast.BoolLit, Value: true}, "v = true"},
		{"UnitLit", &ast.Literal{Kind: ast.UnitLit}, "v = ()"},
		{"BinaryOp", &ast.BinaryOp{Left: one, Op: "+", Right: two}, "v = 1 + 2"},
		{"UnaryOpSymbol", &ast.UnaryOp{Op: "-", Expr: id}, "v = -x"},
		{"UnaryOpWord", &ast.UnaryOp{Op: "not", Expr: id}, "v = not x"},
		{"Lambda", &ast.Lambda{Params: []*ast.Param{{Name: "y"}}, Body: id}, `v = \y. x`},
		{"FuncLit", &ast.FuncLit{Params: []*ast.Param{{Name: "y"}}, Body: id}, "func(y) {"},
		{"FuncCall", &ast.FuncCall{Func: id, Args: []ast.Expr{one}}, "v = x(1)"},
		{"LetIn", &ast.Let{Name: "a", Value: one, Body: id}, "let a = 1 in x"},
		{"LetRecIn", &ast.LetRec{Name: "a", Value: one, Body: id}, "letrec a = 1 in x"},
		// Two literal statements: the second is a non-starter, so a `;` separates
		// them for round-trip safety (see startsWithStatementStarter).
		{"Block", &ast.Block{Exprs: []ast.Expr{one, two}}, "{\n  1;\n  2\n}"},
		{"If", &ast.If{Condition: id, Then: one, Else: two}, "if x then 1 else 2"},
		{
			"Match",
			&ast.Match{Expr: id, Cases: []*ast.Case{{Pattern: &ast.WildcardPattern{}, Body: one}}},
			"match x {\n  _ => 1\n}",
		},
		{"List", &ast.List{Elements: []ast.Expr{one, two}}, "v = [1, 2]"},
		{"Array", &ast.Array{Elements: []ast.Expr{one, two}}, "v = #[1, 2]"},
		{"Tuple", &ast.Tuple{Elements: []ast.Expr{one, two}}, "v = (1, 2)"},
		{"Tuple1", &ast.Tuple{Elements: []ast.Expr{one}}, "v = (1,)"},
		{"Record", &ast.Record{Fields: []*ast.Field{{Name: "a", Value: one}}}, "v = { a: 1 }"},
		{"RecordAccess", &ast.RecordAccess{Record: id, Field: "f"}, "v = x.f"},
		{
			"RecordUpdate",
			&ast.RecordUpdate{Base: id, Fields: []*ast.Field{{Name: "a", Value: one}}},
			"v = { x | a: 1 }",
		},
		{"QuasiQuote", &ast.QuasiQuote{Kind: "sql", Template: "SELECT 1"}, `sql"""SELECT 1"""`},
		{"Send", &ast.Send{Channel: id, Value: one}, "x <- 1"},
		{"Recv", &ast.Recv{Channel: id}, "<- x"},
		{
			"ForallExpr",
			&ast.ForallExpr{Var: "i", Lo: one, Hi: two, Body: &ast.Literal{Kind: ast.BoolLit, Value: true}},
			"forall i: 1..2 => true",
		},
		{"AssertStmt", &ast.AssertStmt{Condition: id}, "assert x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := renderExpr(t, tc.expr)
			if err != nil {
				t.Fatalf("render %s: %v", tc.name, err)
			}
			mustContain(t, got, tc.want)
		})
	}
}

// TestTypeNodeCoverage renders every concrete type node via a let annotation.
func TestTypeNodeCoverage(t *testing.T) {
	cases := []struct {
		name string
		typ  ast.Type
		want string
	}{
		{"SimpleType", &ast.SimpleType{Name: "int"}, ": int ="},
		{"TypeVar", &ast.TypeVar{Name: "a"}, ": a ="},
		{"ListType", &ast.ListType{Element: &ast.SimpleType{Name: "int"}}, ": [int] ="},
		{"ArrayType", &ast.ArrayType{Element: &ast.SimpleType{Name: "int"}}, ": Array[int] ="},
		{
			"TupleType",
			&ast.TupleType{Elements: []ast.Type{&ast.SimpleType{Name: "int"}, &ast.SimpleType{Name: "bool"}}},
			": (int, bool) =",
		},
		{
			"TypeApp",
			&ast.TypeApp{Constructor: "Option", Args: []ast.Type{&ast.SimpleType{Name: "int"}}},
			": Option[int] =",
		},
		{
			"FuncType",
			&ast.FuncType{Params: []ast.Type{&ast.SimpleType{Name: "int"}}, Return: &ast.SimpleType{Name: "bool"}},
			": (int) -> bool =",
		},
		{
			"RecordType",
			&ast.RecordType{Fields: []*ast.RecordField{{Name: "a", Type: &ast.SimpleType{Name: "int"}}}},
			": { a: int } =",
		},
		{
			"LabelledType",
			&ast.LabelledType{Base: &ast.SimpleType{Name: "string"}, Label: &ast.LabelExpr{Name: "email"}},
			": string<email> =",
		},
		{
			"RefinementType",
			&ast.LabelledType{Base: &ast.SimpleType{Name: "string"}, Refinement: &ast.RefinementExpr{NotLabel: "pii"}},
			": string{not pii} =",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &ast.Let{Name: "v", Type: tc.typ, Value: &ast.Literal{Kind: ast.IntLit, Value: int64(0)}}
			got, err := renderDecl(t, e)
			if err != nil {
				t.Fatalf("render %s: %v", tc.name, err)
			}
			mustContain(t, got, tc.want)
		})
	}
}

// TestPatternNodeCoverage renders every concrete pattern node in a match arm.
func TestPatternNodeCoverage(t *testing.T) {
	body := &ast.Literal{Kind: ast.IntLit, Value: int64(0)}
	cases := []struct {
		name    string
		pattern ast.Pattern
		want    string
	}{
		{"Wildcard", &ast.WildcardPattern{}, "_ => 0"},
		{"IdentifierBinder", &ast.Identifier{Name: "x"}, "x => 0"},
		{"LiteralPattern", &ast.Literal{Kind: ast.IntLit, Value: int64(5)}, "5 => 0"},
		{
			"ConsPattern",
			&ast.ConsPattern{Head: &ast.Identifier{Name: "h"}, Tail: &ast.Identifier{Name: "t"}},
			"[h, ...t] => 0",
		},
		{
			"ListPattern",
			&ast.ListPattern{Elements: []ast.Pattern{&ast.Identifier{Name: "a"}}, Rest: &ast.Identifier{Name: "r"}},
			"[a, ...r] => 0",
		},
		{
			"TuplePattern",
			&ast.TuplePattern{Elements: []ast.Pattern{&ast.Identifier{Name: "a"}, &ast.Identifier{Name: "b"}}},
			"(a, b) => 0",
		},
		{
			"RecordPattern",
			&ast.RecordPattern{Fields: []*ast.FieldPattern{{Name: "a", Pattern: &ast.Identifier{Name: "x"}}}, Rest: true},
			"{ a: x, ... } => 0",
		},
		{
			"ConstructorPattern",
			&ast.ConstructorPattern{Name: "Some", Patterns: []ast.Pattern{&ast.Identifier{Name: "x"}}},
			"Some(x) => 0",
		},
		{"ConstructorNoArgs", &ast.ConstructorPattern{Name: "None"}, "None => 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &ast.Match{Expr: &ast.Identifier{Name: "e"}, Cases: []*ast.Case{{Pattern: tc.pattern, Body: body}}}
			got, err := renderExpr(t, m)
			if err != nil {
				t.Fatalf("render %s: %v", tc.name, err)
			}
			mustContain(t, got, tc.want)
		})
	}
}

// TestDeclNodeCoverage renders every concrete declaration node kind.
func TestDeclNodeCoverage(t *testing.T) {
	intT := &ast.SimpleType{Name: "int"}
	one := &ast.Literal{Kind: ast.IntLit, Value: int64(1)}

	t.Run("FuncDecl_full", func(t *testing.T) {
		d := &ast.FuncDecl{
			Name:       "f",
			IsExport:   true,
			IsPure:     true,
			TypeParams: []string{"a"},
			Params:     []*ast.Param{{Name: "x", Type: intT}},
			ReturnType: intT,
			Effects:    []ast.EffectAnnotation{{Name: "IO"}},
			Body:       &ast.Block{Exprs: []ast.Expr{&ast.Identifier{Name: "x"}}},
		}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, "export pure func f[a](x: int) -> int ! {IO} = x")
	})

	t.Run("FuncDecl_extern", func(t *testing.T) {
		d := &ast.FuncDecl{Name: "g", IsExtern: true, Params: []*ast.Param{{Name: "x", Type: intT}}, ReturnType: intT}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, "extern func g(x: int) -> int")
	})

	t.Run("FuncDecl_annotation", func(t *testing.T) {
		d := &ast.FuncDecl{
			Name:        "h",
			Annotations: []*ast.Annotation{{Name: "verify", Args: []ast.Expr{one}}},
			Body:        &ast.Block{Exprs: []ast.Expr{one}},
		}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		// @verify re-emits the `depth:` key so the annotation round-trips
		// (parseVerifyAnnotation requires it).
		mustContain(t, got, "@verify(depth: 1)")
	})

	t.Run("FuncDecl_tests", func(t *testing.T) {
		d := &ast.FuncDecl{
			Name:   "add",
			Params: []*ast.Param{{Name: "a", Type: intT}},
			Tests:  []*ast.TestCase{{Inputs: []ast.Expr{one}, Expected: one}},
			Body:   &ast.Block{Exprs: []ast.Expr{one}},
		}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, "tests [")
		mustContain(t, got, "(1, 1)")
	})

	t.Run("TypeDecl_alias", func(t *testing.T) {
		d := &ast.TypeDecl{Name: "Id", Definition: &ast.TypeAlias{Target: intT}}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, "type Id = int")
	})

	t.Run("TypeDecl_record", func(t *testing.T) {
		d := &ast.TypeDecl{Name: "R", Definition: &ast.RecordType{Fields: []*ast.RecordField{{Name: "a", Type: intT}}}}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, "type R = { a: int }")
	})

	t.Run("TypeClass", func(t *testing.T) {
		d := &ast.TypeClass{Name: "Show", TypeParam: "a", Methods: []*ast.Method{{Name: "show", Type: &ast.FuncType{Params: []ast.Type{&ast.TypeVar{Name: "a"}}, Return: &ast.SimpleType{Name: "string"}}}}}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, "class Show[a] {")
		mustContain(t, got, "show: (a) -> string")
	})

	t.Run("Instance_empty", func(t *testing.T) {
		d := &ast.Instance{ClassName: "Show", Type: intT}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, "instance Show[int] {")
	})

	t.Run("Instance_populated_errors", func(t *testing.T) {
		d := &ast.Instance{ClassName: "Show", Type: intT, Methods: map[string]ast.Expr{"show": one}}
		if _, err := renderDecl(t, d); err == nil {
			t.Fatal("expected error for populated instance (map-order nondeterminism), got nil")
		}
	})

	t.Run("TestDecl", func(t *testing.T) {
		d := &ast.TestDecl{Name: "t1", Body: []ast.Expr{one}}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, `test "t1" {`)
	})

	t.Run("PropertyDecl", func(t *testing.T) {
		d := &ast.PropertyDecl{Name: "p1", Property: &ast.Property{
			Binders: []*ast.Binder{{Name: "x", Type: intT}},
			Expr:    &ast.BinaryOp{Left: &ast.Identifier{Name: "x"}, Op: "==", Right: &ast.Identifier{Name: "x"}},
		}}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, `property "p1" {`)
		mustContain(t, got, "forall(x: int) => x == x")
	})

	t.Run("AssertStmt_decl", func(t *testing.T) {
		d := &ast.AssertStmt{Condition: &ast.Identifier{Name: "ok"}}
		got, err := renderDecl(t, d)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, got, "assert ok")
	})
}

// TestAlgebraicNamedFields covers ADT constructors with named fields.
func TestAlgebraicNamedFields(t *testing.T) {
	d := &ast.TypeDecl{Name: "T", Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{
		{Name: "C", Fields: []*ast.ConstructorField{{Name: "x", Type: &ast.SimpleType{Name: "int"}}}},
	}}}
	got, err := renderDecl(t, d)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, got, "type T = C(x: int)")
}

// TestNilChildErrors verifies nil-required children fail loudly.
func TestNilChildErrors(t *testing.T) {
	// A binary op with a nil operand.
	if _, err := renderExpr(t, &ast.BinaryOp{Left: nil, Op: "+", Right: &ast.Literal{Kind: ast.IntLit, Value: int64(1)}}); err == nil {
		t.Error("expected error for nil binary operand")
	}
	// A type node that is nil.
	e := &ast.Let{Name: "v", Type: &ast.ListType{Element: nil}, Value: &ast.Literal{Kind: ast.IntLit, Value: int64(0)}}
	if _, err := renderDecl(t, e); err == nil {
		t.Error("expected error for nil list element type")
	}
}
