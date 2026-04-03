package lower

import (
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/types"
)

func TestProjectType_Primitives(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
		want string
	}{
		{"int", types.TInt, "int64"},
		{"float", types.TFloat, "float64"},
		{"bool", types.TBool, "bool"},
		{"string", types.TString, "string"},
		{"unit", types.TUnit, "struct{}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectType(tt.typ)
			if got.GoString() != tt.want {
				t.Errorf("ProjectType(%v) = %q, want %q", tt.typ, got.GoString(), tt.want)
			}
		})
	}
}

func TestProjectType_NamedADT(t *testing.T) {
	// Unknown TCon names are treated as user-defined ADTs (pointer semantics).
	got := ProjectType(&types.TCon{Name: "Color"})
	named, ok := got.(stmt.NamedType)
	if !ok {
		t.Fatalf("expected NamedType, got %T", got)
	}
	if !named.Pointer {
		t.Error("ADT types should use pointer semantics")
	}
	if named.Name != "Color" {
		t.Errorf("expected name Color, got %s", named.Name)
	}
}

func TestProjectType_List(t *testing.T) {
	got := ProjectType(&types.TList{Element: types.TInt})
	slice, ok := got.(stmt.SliceType)
	if !ok {
		t.Fatalf("expected SliceType, got %T", got)
	}
	if slice.Elem.GoString() != "int64" {
		t.Errorf("expected []int64 element, got %s", slice.Elem.GoString())
	}
}

func TestProjectType_Array(t *testing.T) {
	got := ProjectType(&types.TArray{Element: types.TString})
	slice, ok := got.(stmt.SliceType)
	if !ok {
		t.Fatalf("expected SliceType, got %T", got)
	}
	if slice.Elem.GoString() != "string" {
		t.Errorf("expected []string element, got %s", slice.Elem.GoString())
	}
}

func TestProjectType_Tuple(t *testing.T) {
	got := ProjectType(&types.TTuple{Elements: []types.Type{types.TInt, types.TBool}})
	tuple, ok := got.(stmt.TupleType)
	if !ok {
		t.Fatalf("expected TupleType, got %T", got)
	}
	if len(tuple.Elems) != 2 {
		t.Errorf("expected 2 elements, got %d", len(tuple.Elems))
	}
}

func TestProjectType_Func(t *testing.T) {
	got := ProjectType(&types.TFunc2{
		Params: []types.Type{types.TInt, types.TString},
		Return: types.TBool,
	})
	fn, ok := got.(stmt.FuncType)
	if !ok {
		t.Fatalf("expected FuncType, got %T", got)
	}
	if len(fn.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(fn.Params))
	}
	if fn.Return.GoString() != "bool" {
		t.Errorf("expected bool return, got %s", fn.Return.GoString())
	}
}

func TestProjectType_TVar(t *testing.T) {
	// Type variables erase to interface{}.
	got := ProjectType(&types.TVar{Name: "a"})
	if _, ok := got.(stmt.InterfaceType); !ok {
		t.Errorf("expected InterfaceType for TVar, got %T", got)
	}
}

func TestProjectType_NominalRecord(t *testing.T) {
	got := ProjectType(&types.TRecord{
		Fields:   map[string]types.Type{"x": types.TInt, "y": types.TInt},
		TypeName: "Position",
	})
	named, ok := got.(stmt.NamedType)
	if !ok {
		t.Fatalf("expected NamedType for nominal record, got %T", got)
	}
	if named.Name != "Position" {
		t.Errorf("expected name Position, got %s", named.Name)
	}
	if named.Pointer {
		t.Error("records should NOT use pointer semantics")
	}
}

func TestProjectType_StructuralRecord(t *testing.T) {
	// Structural record without TypeName → interface{}.
	got := ProjectType(&types.TRecord{
		Fields: map[string]types.Type{"x": types.TInt},
	})
	if _, ok := got.(stmt.InterfaceType); !ok {
		t.Errorf("expected InterfaceType for structural record, got %T", got)
	}
}

func TestProjectType_TApp(t *testing.T) {
	// list[int] → SliceType
	got := ProjectType(&types.TApp{
		Constructor: &types.TCon{Name: "list"},
		Args:        []types.Type{types.TInt},
	})
	slice, ok := got.(stmt.SliceType)
	if !ok {
		t.Fatalf("expected SliceType for list[int], got %T", got)
	}
	if slice.Elem.GoString() != "int64" {
		t.Errorf("expected int64 element, got %s", slice.Elem.GoString())
	}
}

func TestProjectType_TAppADT(t *testing.T) {
	// Option[int] → NamedType
	got := ProjectType(&types.TApp{
		Constructor: &types.TCon{Name: "Option"},
		Args:        []types.Type{types.TInt},
	})
	named, ok := got.(stmt.NamedType)
	if !ok {
		t.Fatalf("expected NamedType for Option[int], got %T", got)
	}
	if named.Name != "Option" {
		t.Errorf("expected name Option, got %s", named.Name)
	}
}

func TestProjectType_Nil(t *testing.T) {
	got := ProjectType(nil)
	if _, ok := got.(stmt.InterfaceType); !ok {
		t.Errorf("expected InterfaceType for nil, got %T", got)
	}
}

func TestProjectType_Deterministic(t *testing.T) {
	typs := []types.Type{
		types.TInt, types.TFloat, types.TBool, types.TString,
		&types.TList{Element: types.TInt},
		&types.TTuple{Elements: []types.Type{types.TInt, types.TString}},
		&types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TBool},
		&types.TVar{Name: "a"},
		&types.TCon{Name: "Color"},
	}
	for _, typ := range typs {
		first := ProjectType(typ).GoString()
		for i := 0; i < 20; i++ {
			got := ProjectType(typ).GoString()
			if got != first {
				t.Errorf("non-deterministic ProjectType: first=%q, got=%q on iter %d for %v", first, got, i, typ)
			}
		}
	}
}

func TestProjectASTType_Primitives(t *testing.T) {
	tests := []struct {
		name string
		typ  ast.Type
		want string
	}{
		{"int", &ast.SimpleType{Name: "int"}, "int64"},
		{"float", &ast.SimpleType{Name: "float"}, "float64"},
		{"bool", &ast.SimpleType{Name: "bool"}, "bool"},
		{"string", &ast.SimpleType{Name: "string"}, "string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectASTType(tt.typ)
			if got.GoString() != tt.want {
				t.Errorf("ProjectASTType(%v) = %q, want %q", tt.typ, got.GoString(), tt.want)
			}
		})
	}
}

func TestProjectASTType_TypeVar(t *testing.T) {
	got := ProjectASTType(&ast.TypeVar{Name: "a"})
	if _, ok := got.(stmt.InterfaceType); !ok {
		t.Errorf("expected InterfaceType for TypeVar, got %T", got)
	}
}

func TestLowerTypeDecl_ADT(t *testing.T) {
	astDecl := &ast.TypeDecl{
		Name:     "Color",
		Exported: true,
		Definition: &ast.AlgebraicType{
			Constructors: []*ast.Constructor{
				{Name: "Red"},
				{Name: "Green"},
				{Name: "Blue"},
				{Name: "Custom", Fields: []*ast.ConstructorField{
					{Name: "r", Type: &ast.SimpleType{Name: "int"}},
					{Name: "g", Type: &ast.SimpleType{Name: "int"}},
					{Name: "b", Type: &ast.SimpleType{Name: "int"}},
				}},
			},
		},
	}

	result := LowerTypeDecl(astDecl)
	if result.Name != "Color" {
		t.Errorf("expected name Color, got %s", result.Name)
	}
	if !result.Exported {
		t.Error("expected exported")
	}

	adt, ok := result.Kind.(stmt.ADTDecl)
	if !ok {
		t.Fatalf("expected ADTDecl, got %T", result.Kind)
	}
	if len(adt.Variants) != 4 {
		t.Fatalf("expected 4 variants, got %d", len(adt.Variants))
	}
	if adt.Variants[0].Tag != "Red" {
		t.Errorf("expected first variant Red, got %s", adt.Variants[0].Tag)
	}
	if len(adt.Variants[3].Fields) != 3 {
		t.Errorf("expected 3 fields on Custom, got %d", len(adt.Variants[3].Fields))
	}
}

func TestLowerTypeDecl_Record(t *testing.T) {
	astDecl := &ast.TypeDecl{
		Name:     "Position",
		Exported: true,
		Definition: &ast.RecordType{
			Fields: []*ast.RecordField{
				{Name: "x", Type: &ast.SimpleType{Name: "float"}},
				{Name: "y", Type: &ast.SimpleType{Name: "float"}},
			},
		},
	}

	result := LowerTypeDecl(astDecl)
	rec, ok := result.Kind.(stmt.RecordDecl)
	if !ok {
		t.Fatalf("expected RecordDecl, got %T", result.Kind)
	}
	if len(rec.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(rec.Fields))
	}
	// Fields should be sorted alphabetically.
	if rec.Fields[0].Name != "x" {
		t.Errorf("expected first field x, got %s", rec.Fields[0].Name)
	}
}
