package emitgo

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/gen/stmt"
)

func TestEmit_SimpleFunc(t *testing.T) {
	prog := &stmt.Program{
		Package: "main",
		FuncDecls: []stmt.FuncDecl{
			{
				Name:       "add",
				Exported:   true,
				Params:     []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}, {Name: "y", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimInt},
				Return: stmt.BinOp{
					Op:    stmt.OpAdd,
					Left:  stmt.VarRef{Name: "x"},
					Right: stmt.VarRef{Name: "y"},
				},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "package main") {
		t.Error("missing package declaration")
	}
	if !strings.Contains(src, "func Add(x int64, y int64) int64") {
		t.Errorf("expected exported func Add, got:\n%s", src)
	}
	if !strings.Contains(src, "return (x + y)") {
		t.Errorf("expected return (x + y), got:\n%s", src)
	}
}

func TestEmit_Literals(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		FuncDecls: []stmt.FuncDecl{
			{
				Name:       "intLit",
				Exported:   true,
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimInt},
				Return:     stmt.LitInt{Value: 42},
			},
			{
				Name:       "boolLit",
				Exported:   true,
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimBool},
				Return:     stmt.LitBool{Value: true},
			},
			{
				Name:       "strLit",
				Exported:   true,
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimString},
				Return:     stmt.LitString{Value: "hello"},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "int64(42)") {
		t.Errorf("missing int literal:\n%s", src)
	}
	if !strings.Contains(src, "true") {
		t.Errorf("missing bool literal:\n%s", src)
	}
	if !strings.Contains(src, `"hello"`) {
		t.Errorf("missing string literal:\n%s", src)
	}
}

func TestEmit_VarDecl(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		FuncDecls: []stmt.FuncDecl{
			{
				Name:       "letBinding",
				Exported:   true,
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimInt},
				Body: []stmt.Stmt{
					stmt.VarDecl{
						Name:  "a",
						Type:  stmt.PrimitiveType{Kind: stmt.PrimInt},
						Value: stmt.LitInt{Value: 10},
					},
				},
				Return: stmt.VarRef{Name: "a"},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "var a int64 = int64(10)") {
		t.Errorf("expected var decl, got:\n%s", src)
	}
}

func TestEmit_ADT(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		TypeDecls: []stmt.TypeDecl{
			{
				Name:     "Color",
				Exported: true,
				Kind: stmt.ADTDecl{Variants: []stmt.ADTVariant{
					{Tag: "Red"},
					{Tag: "Green"},
					{Tag: "Blue"},
				}},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "type ColorKind int") {
		t.Errorf("missing kind type:\n%s", src)
	}
	if !strings.Contains(src, "ColorKindRed") {
		t.Errorf("missing Red constant:\n%s", src)
	}
	if !strings.Contains(src, "type Color struct") {
		t.Errorf("missing Color struct:\n%s", src)
	}
	if !strings.Contains(src, "func NewColorRed()") {
		t.Errorf("missing constructor:\n%s", src)
	}
}

func TestEmit_ADTWithFields(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		TypeDecls: []stmt.TypeDecl{
			{
				Name:     "Option",
				Exported: true,
				Kind: stmt.ADTDecl{Variants: []stmt.ADTVariant{
					{Tag: "Some", Fields: []stmt.ResolvedType{stmt.PrimitiveType{Kind: stmt.PrimInt}}},
					{Tag: "None"},
				}},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "type OptionSome struct") {
		t.Errorf("missing variant struct:\n%s", src)
	}
	if !strings.Contains(src, "Value0 int64") {
		t.Errorf("missing field:\n%s", src)
	}
	if !strings.Contains(src, "func NewOptionSome(field0 int64) *Option") {
		t.Errorf("missing constructor with params:\n%s", src)
	}
}

func TestEmit_Record(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		TypeDecls: []stmt.TypeDecl{
			{
				Name:     "Position",
				Exported: true,
				Kind: stmt.RecordDecl{Fields: []stmt.RecordField{
					{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimFloat}},
					{Name: "y", Type: stmt.PrimitiveType{Kind: stmt.PrimFloat}},
				}},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "type Position struct") {
		t.Errorf("missing struct:\n%s", src)
	}
	if !strings.Contains(src, "X float64") {
		t.Errorf("missing field X:\n%s", src)
	}
}

func TestEmit_IfExpr(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		FuncDecls: []stmt.FuncDecl{
			{
				Name:       "max",
				Exported:   true,
				Params:     []stmt.Param{{Name: "a", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}, {Name: "b", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimInt},
				Return: stmt.IfExpr{
					Cond: stmt.BinOp{Op: stmt.OpGt, Left: stmt.VarRef{Name: "a"}, Right: stmt.VarRef{Name: "b"}},
					Then: stmt.VarRef{Name: "a"},
					Else: stmt.VarRef{Name: "b"},
				},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	// Should generate an IIFE for the ternary.
	if !strings.Contains(src, "func() interface{}") {
		t.Errorf("expected IIFE for if-expr:\n%s", src)
	}
}

func TestEmit_SwitchStmt(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		FuncDecls: []stmt.FuncDecl{
			{
				Name:       "isRed",
				Exported:   true,
				Params:     []stmt.Param{{Name: "c", Type: stmt.NamedType{Name: "Color", Pointer: true}}},
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimBool},
				Body: []stmt.Stmt{
					stmt.SwitchStmt{
						Scrutinee: stmt.VarRef{Name: "c"},
						Cases: []stmt.SwitchCase{
							{
								Tag:  "Red",
								Body: []stmt.Stmt{stmt.ReturnStmt{Value: stmt.LitBool{Value: true}}},
							},
						},
						Default: []stmt.Stmt{stmt.ReturnStmt{Value: stmt.LitBool{Value: false}}},
					},
				},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "_scrutinee :=") {
		t.Errorf("missing scrutinee assignment:\n%s", src)
	}
	if !strings.Contains(src, "switch _scrutinee.Kind") {
		t.Errorf("missing switch:\n%s", src)
	}
}

func TestEmit_ListLit(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		FuncDecls: []stmt.FuncDecl{
			{
				Name:       "nums",
				Exported:   true,
				ReturnType: stmt.SliceType{Elem: stmt.PrimitiveType{Kind: stmt.PrimInt}},
				Return: stmt.ListLit{
					ElemType: stmt.PrimitiveType{Kind: stmt.PrimInt},
					Elems:    []stmt.Expr{stmt.LitInt{Value: 1}, stmt.LitInt{Value: 2}, stmt.LitInt{Value: 3}},
				},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "[]int64{int64(1), int64(2), int64(3)}") {
		t.Errorf("expected list literal:\n%s", src)
	}
}

func TestEmit_GoKeywordSanitization(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		FuncDecls: []stmt.FuncDecl{
			{
				Name:       "test",
				Exported:   true,
				Params:     []stmt.Param{{Name: "type", Type: stmt.PrimitiveType{Kind: stmt.PrimString}}},
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimString},
				Return:     stmt.VarRef{Name: "type"},
			},
		},
	}

	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}

	src := string(out)
	if !strings.Contains(src, "type_") {
		t.Errorf("expected sanitized 'type_':\n%s", src)
	}
}

func TestEmit_EmptyProgram(t *testing.T) {
	prog := &stmt.Program{Package: "empty"}
	out, err := Emit(prog)
	if err != nil {
		t.Fatalf("Emit failed: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "package empty") {
		t.Errorf("missing package:\n%s", src)
	}
}

func TestEmit_Deterministic(t *testing.T) {
	prog := &stmt.Program{
		Package: "test",
		FuncDecls: []stmt.FuncDecl{
			{
				Name:       "f",
				Exported:   true,
				Params:     []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				ReturnType: stmt.PrimitiveType{Kind: stmt.PrimInt},
				Return:     stmt.BinOp{Op: stmt.OpMul, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 2}},
			},
		},
	}

	first, err := Emit(prog)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		got, err := Emit(prog)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(first) {
			t.Errorf("non-deterministic Emit on iteration %d", i)
		}
	}
}
