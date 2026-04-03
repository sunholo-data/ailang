package stmt

import "testing"

func TestResolvedTypeGoString(t *testing.T) {
	tests := []struct {
		name string
		typ  ResolvedType
		want string
	}{
		{"int", PrimitiveType{PrimInt}, "int64"},
		{"float", PrimitiveType{PrimFloat}, "float64"},
		{"bool", PrimitiveType{PrimBool}, "bool"},
		{"string", PrimitiveType{PrimString}, "string"},
		{"unit", PrimitiveType{PrimUnit}, "struct{}"},
		{"named value", NamedType{Name: "Position", Pointer: false}, "Position"},
		{"named pointer", NamedType{Name: "Expr", Pointer: true}, "*Expr"},
		{"slice of int", SliceType{Elem: PrimitiveType{PrimInt}}, "[]int64"},
		{"slice of named", SliceType{Elem: NamedType{Name: "Color", Pointer: true}}, "[]*Color"},
		{"func", FuncType{
			Params: []ResolvedType{PrimitiveType{PrimInt}},
			Return: PrimitiveType{PrimBool},
		}, "func(int64) bool"},
		{"tuple2", TupleType{Elems: []ResolvedType{PrimitiveType{PrimInt}, PrimitiveType{PrimString}}}, "Tuple2"},
		{"interface", InterfaceType{}, "interface{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.GoString()
			if got != tt.want {
				t.Errorf("GoString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolvedTypeDeterminism(t *testing.T) {
	// Type projection must be deterministic: same input → same output.
	types := []ResolvedType{
		PrimitiveType{PrimInt},
		PrimitiveType{PrimFloat},
		NamedType{Name: "Color", Pointer: true},
		SliceType{Elem: PrimitiveType{PrimString}},
		FuncType{
			Params: []ResolvedType{PrimitiveType{PrimInt}, PrimitiveType{PrimInt}},
			Return: PrimitiveType{PrimBool},
		},
		TupleType{Elems: []ResolvedType{PrimitiveType{PrimInt}, PrimitiveType{PrimString}}},
		InterfaceType{},
	}
	for _, typ := range types {
		first := typ.GoString()
		for i := 0; i < 10; i++ {
			got := typ.GoString()
			if got != first {
				t.Errorf("non-deterministic GoString: first=%q, got=%q on iteration %d", first, got, i)
			}
		}
	}
}

func TestValidateProgram(t *testing.T) {
	valid := &Program{
		Package: "test",
		TypeDecls: []TypeDecl{
			{Name: "Color", Kind: ADTDecl{Variants: []ADTVariant{
				{Tag: "Red"},
				{Tag: "Green"},
				{Tag: "Blue"},
			}}, Exported: true},
		},
		FuncDecls: []FuncDecl{
			{
				Name:       "isRed",
				Params:     []Param{{Name: "c", Type: NamedType{Name: "Color", Pointer: true}}},
				ReturnType: PrimitiveType{PrimBool},
				Body:       nil,
				Return:     LitBool{Value: true},
				Exported:   true,
			},
		},
	}
	if err := Validate(valid); err != nil {
		t.Errorf("valid program failed validation: %v", err)
	}
}

func TestValidateRejectsInvalid(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
	}{
		{"empty package", &Program{Package: ""}},
		{"nil type kind", &Program{Package: "test", TypeDecls: []TypeDecl{{Name: "T", Kind: nil}}}},
		{"empty func name", &Program{Package: "test", FuncDecls: []FuncDecl{{Name: "", ReturnType: PrimitiveType{PrimInt}}}}},
		{"nil return type", &Program{Package: "test", FuncDecls: []FuncDecl{{Name: "f", ReturnType: nil}}}},
		{"nil param type", &Program{Package: "test", FuncDecls: []FuncDecl{{
			Name: "f", ReturnType: PrimitiveType{PrimInt},
			Params: []Param{{Name: "x", Type: nil}},
		}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Validate(tt.prog); err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestStmtInterfaces(t *testing.T) {
	// Verify all statement types implement Stmt.
	stmts := []Stmt{
		VarDecl{Name: "x", Value: LitInt{42}},
		IfStmt{Cond: LitBool{true}},
		SwitchStmt{Scrutinee: VarRef{Name: "x"}},
		AssignStmt{Name: "x", Value: LitInt{1}},
		ReturnStmt{Value: LitInt{0}},
		ExprStmt{Value: Call{Func: VarRef{Name: "f"}}},
	}
	if len(stmts) != 6 {
		t.Errorf("expected 6 statement types, got %d", len(stmts))
	}
}

func TestExprInterfaces(t *testing.T) {
	// Verify all expression types implement Expr.
	exprs := []Expr{
		LitInt{42},
		LitFloat{3.14},
		LitBool{true},
		LitString{"hello"},
		LitUnit{},
		VarRef{Name: "x"},
		GlobalRef{Module: "m", Name: "f"},
		BinOp{Op: OpAdd, Left: LitInt{1}, Right: LitInt{2}},
		UnOp{Op: OpNeg, Operand: LitInt{1}},
		Call{Func: VarRef{Name: "f"}, Args: []Expr{LitInt{1}}},
		FieldAccess{Record: VarRef{Name: "r"}, Field: "x"},
		RecordLit{TypeName: "Pos", Fields: []FieldInit{{Name: "x", Value: LitInt{0}}}},
		RecordUpdate{Base: VarRef{Name: "r"}, Fields: []FieldInit{{Name: "x", Value: LitInt{1}}}},
		ListLit{ElemType: PrimitiveType{PrimInt}, Elems: []Expr{LitInt{1}}},
		Cons{Head: LitInt{1}, Tail: ListLit{ElemType: PrimitiveType{PrimInt}}},
		TupleLit{Elems: []Expr{LitInt{1}, LitString{"a"}}},
		ArrayLit{ElemType: PrimitiveType{PrimInt}, Elems: []Expr{LitInt{1}}},
		ADTConstructor{TypeName: "Option", Tag: "Some", Args: []Expr{LitInt{42}}},
		Lambda{Params: []Param{{Name: "x", Type: PrimitiveType{PrimInt}}}, Return: VarRef{Name: "x"}},
		TypeAssert{Value: VarRef{Name: "x"}, Type: PrimitiveType{PrimInt}},
		IfExpr{Cond: LitBool{true}, Then: LitInt{1}, Else: LitInt{0}},
		BuiltinCall{Name: "_str_trim", Args: []Expr{VarRef{Name: "s"}}},
	}
	if len(exprs) != 22 {
		t.Errorf("expected 22 expression types, got %d", len(exprs))
	}
}
