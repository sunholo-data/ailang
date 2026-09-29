package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// ailang#1354 / #1355: field slots must come from the receiver's own type,
// never from another registered record that merely shares the field name,
// and the same program must compile to the same bytecode every time.
//
// The fixture registers two records that share `x` at different sorted
// positions — M {a,b,x} (x=2) and V {x,y} (x=0) — plus an alias P = V.
// Go randomises map range order on every range statement, so compiling it
// many times in one process exercises any remaining map-order dependence.

func floatT() stmt.ResolvedType { return stmt.PrimitiveType{Kind: stmt.PrimFloat} }

func sharedFieldTypeDecls() []stmt.TypeDecl {
	return []stmt.TypeDecl{
		{Name: "M", Kind: stmt.RecordDecl{Fields: []stmt.RecordField{
			{Name: "a", Type: floatT()}, {Name: "b", Type: floatT()}, {Name: "x", Type: floatT()},
		}}},
		{Name: "V", Kind: stmt.RecordDecl{Fields: []stmt.RecordField{
			{Name: "x", Type: floatT()}, {Name: "y", Type: floatT()},
		}}},
		{Name: "P", Kind: stmt.TypeAliasDecl{Target: stmt.NamedType{Name: "V"}}},
	}
}

func vLit(x, y float64) stmt.RecordLit {
	return stmt.RecordLit{TypeName: "V", Fields: []stmt.FieldInit{
		{Name: "x", Value: stmt.LitFloat{Value: x}},
		{Name: "y", Value: stmt.LitFloat{Value: y}},
	}}
}

// sharedFieldProgram returns main() = [v.x via V, p.x via alias P,
// v.x with no type hint, {v | x: 99}.y with no type hint].
func sharedFieldProgram() *stmt.Program {
	v := vLit(10, 20)
	return &stmt.Program{
		TypeDecls: sharedFieldTypeDecls(),
		FuncDecls: []stmt.FuncDecl{{
			Name:     "main",
			Exported: true,
			Return: stmt.ListLit{Elems: []stmt.Expr{
				stmt.FieldAccess{Record: v, Field: "x", RecordType: "V"},
				stmt.FieldAccess{Record: v, Field: "x", RecordType: "P"},
				stmt.FieldAccess{Record: v, Field: "x"},
				stmt.FieldAccess{
					Record: stmt.RecordUpdate{Base: v, Fields: []stmt.FieldInit{
						{Name: "x", Value: stmt.LitFloat{Value: 99}},
					}},
					Field: "y",
				},
			}},
		}},
	}
}

func TestSharedFieldNames_CorrectAndDeterministic(t *testing.T) {
	want := []float64{10, 10, 10, 20}
	var first string
	for i := 0; i < 200; i++ {
		prog := sharedFieldProgram()
		img, err := Compile(prog)
		if err != nil {
			t.Fatalf("compile %d: %v", i, err)
		}
		dis := bytecode.Disassemble(img)
		if i == 0 {
			first = dis
			if !strings.Contains(dis, "UPDATE_RECORD") {
				t.Fatal("unknown-shape update did not lower to UPDATE_RECORD")
			}
		} else if dis != first {
			t.Fatalf("compile %d produced different bytecode than compile 0:\n--- 0 ---\n%s\n--- %d ---\n%s", i, first, i, dis)
		}
		if i%50 != 0 {
			continue
		}
		got := runProgram(t, prog, "main", nil).AsList()
		if len(got) != len(want) {
			t.Fatalf("got %d results, want %d", len(got), len(want))
		}
		for j, w := range want {
			if got[j].Flt != w {
				t.Errorf("compile %d result %d: got %v, want %v", i, j, got[j].Flt, w)
			}
		}
	}
}

// A record update whose base type is unknown must keep the base's own shape
// (update by name), not rebuild it with another type's field list.
func TestRecordUpdate_UnknownBase_KeepsShape(t *testing.T) {
	prog := &stmt.Program{
		TypeDecls: sharedFieldTypeDecls(),
		FuncDecls: []stmt.FuncDecl{{
			Name:     "main",
			Exported: true,
			Return: stmt.RecordUpdate{Base: vLit(1, 2), Fields: []stmt.FieldInit{
				{Name: "x", Value: stmt.LitFloat{Value: 99}},
			}},
		}},
	}
	got := runProgram(t, prog, "main", nil).AsRecord()
	if len(got) != 2 || got[0].Name != "x" || got[0].Value.Flt != 99 || got[1].Name != "y" || got[1].Value.Flt != 2 {
		t.Fatalf("got %+v, want {x: 99, y: 2}", got)
	}
}

func TestRegisterRecordType_ConflictingRedeclarationIsAmbiguous(t *testing.T) {
	rt := map[string]recordTypeInfo{}
	registerRecordType(rt, "Point", []string{"x", "y"})
	registerRecordType(rt, "Point", []string{"x", "y"}) // identical: still usable
	if rt["Point"].ambiguous {
		t.Fatal("identical redeclaration must not be ambiguous")
	}
	registerRecordType(rt, "Point", []string{"x", "y", "z"})
	if !rt["Point"].ambiguous {
		t.Fatal("conflicting redeclaration must be ambiguous")
	}
	fc := &funcCompiler{recordTypes: rt}
	if idx := fc.lookupFieldIndex("x", nil, "Point"); idx != -1 {
		t.Errorf("ambiguous type resolved x to slot %d, want -1 (by-name)", idx)
	}
}

func TestResolveRecordAliases_ChainsAndCycles(t *testing.T) {
	rt := map[string]recordTypeInfo{}
	registerRecordType(rt, "V", []string{"x", "y"})
	resolveRecordAliases(rt, map[string]string{
		"P": "V", "Q": "P", // chain to a record
		"A": "B", "B": "A", // cycle
		"D": "Missing", // dangling
	})
	for _, name := range []string{"P", "Q"} {
		if got := rt[name].sortedFields; len(got) != 2 || got[0] != "x" {
			t.Errorf("%s: got %v, want [x y]", name, got)
		}
	}
	for _, name := range []string{"A", "B", "D"} {
		if _, ok := rt[name]; ok {
			t.Errorf("%s must stay unregistered", name)
		}
	}
}
