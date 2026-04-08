package compiler

import (
	"testing"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/vm"
)

// runCompiled compiles a single function (named "test") with the given params,
// runs it through the VM with the supplied args, and returns the result.
func runCompiled(t *testing.T, params []stmt.Param, body []stmt.Stmt, ret stmt.Expr, args []bytecode.Value) bytecode.Value {
	t.Helper()
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:     "test",
				Params:   params,
				Body:     body,
				Return:   ret,
				Exported: true,
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if img.EntryPoint < 0 {
		t.Fatalf("no entry point")
	}
	machine := vm.NewVM(img)
	got, err := machine.Run(img.Prototypes[img.EntryPoint], args)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return got
}

// --- Literals ---------------------------------------------------------------

func TestCompile_LitInt(t *testing.T) {
	got := runCompiled(t, nil, nil, stmt.LitInt{Value: 42}, nil)
	if got.Tag != bytecode.TagInt || got.Int != 42 {
		t.Errorf("got %v, want Int(42)", got)
	}
}

func TestCompile_LitFloat(t *testing.T) {
	got := runCompiled(t, nil, nil, stmt.LitFloat{Value: 3.14}, nil)
	if got.Tag != bytecode.TagFloat || got.Flt != 3.14 {
		t.Errorf("got %v, want Float(3.14)", got)
	}
}

func TestCompile_LitBool(t *testing.T) {
	got := runCompiled(t, nil, nil, stmt.LitBool{Value: true}, nil)
	if got.Tag != bytecode.TagBool || !got.Bool {
		t.Errorf("got %v, want Bool(true)", got)
	}
}

func TestCompile_LitString(t *testing.T) {
	got := runCompiled(t, nil, nil, stmt.LitString{Value: "hello"}, nil)
	if got.Tag != bytecode.TagString || got.AsString() != "hello" {
		t.Errorf("got %v, want String(\"hello\")", got)
	}
}

func TestCompile_LitUnit(t *testing.T) {
	got := runCompiled(t, nil, nil, stmt.LitUnit{}, nil)
	if got.Tag != bytecode.TagUnit {
		t.Errorf("got %v, want Unit", got)
	}
}

// --- VarRef (params) --------------------------------------------------------

func TestCompile_Param_Identity(t *testing.T) {
	params := []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}}
	got := runCompiled(t, params, nil, stmt.VarRef{Name: "x"}, []bytecode.Value{bytecode.NewInt(99)})
	if got.Int != 99 {
		t.Errorf("got %d, want 99", got.Int)
	}
}

// --- BinOp arithmetic -------------------------------------------------------

func TestCompile_BinOp_AddInts(t *testing.T) {
	params := []stmt.Param{
		{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
		{Name: "y", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
	}
	ret := stmt.BinOp{
		Op:    stmt.OpAdd,
		Left:  stmt.VarRef{Name: "x"},
		Right: stmt.VarRef{Name: "y"},
	}
	got := runCompiled(t, params, nil, ret, []bytecode.Value{bytecode.NewInt(3), bytecode.NewInt(4)})
	if got.Int != 7 {
		t.Errorf("got %d, want 7", got.Int)
	}
}

func TestCompile_BinOp_AllArith(t *testing.T) {
	cases := []struct {
		op   stmt.BinOpKind
		a, b int64
		want int64
	}{
		{stmt.OpAdd, 10, 3, 13},
		{stmt.OpSub, 10, 3, 7},
		{stmt.OpMul, 10, 3, 30},
		{stmt.OpDiv, 10, 3, 3},
		{stmt.OpMod, 10, 3, 1},
	}
	for _, tc := range cases {
		ret := stmt.BinOp{
			Op:    tc.op,
			Left:  stmt.LitInt{Value: tc.a},
			Right: stmt.LitInt{Value: tc.b},
		}
		got := runCompiled(t, nil, nil, ret, nil)
		if got.Int != tc.want {
			t.Errorf("op %v: got %d, want %d", tc.op, got.Int, tc.want)
		}
	}
}

func TestCompile_BinOp_Nested(t *testing.T) {
	// (1 + 2) * (3 + 4) = 21
	ret := stmt.BinOp{
		Op:    stmt.OpMul,
		Left:  stmt.BinOp{Op: stmt.OpAdd, Left: stmt.LitInt{Value: 1}, Right: stmt.LitInt{Value: 2}},
		Right: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.LitInt{Value: 3}, Right: stmt.LitInt{Value: 4}},
	}
	got := runCompiled(t, nil, nil, ret, nil)
	if got.Int != 21 {
		t.Errorf("got %d, want 21", got.Int)
	}
}

// --- UnOp -------------------------------------------------------------------

func TestCompile_UnOp_Neg(t *testing.T) {
	ret := stmt.UnOp{Op: stmt.OpNeg, Operand: stmt.LitInt{Value: 5}}
	got := runCompiled(t, nil, nil, ret, nil)
	if got.Int != -5 {
		t.Errorf("got %d, want -5", got.Int)
	}
}

// --- VarDecl / let-bindings -------------------------------------------------

func TestCompile_VarDecl_Single(t *testing.T) {
	// let a = 5; return a + 1
	body := []stmt.Stmt{
		stmt.VarDecl{Name: "a", Value: stmt.LitInt{Value: 5}},
	}
	ret := stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "a"}, Right: stmt.LitInt{Value: 1}}
	got := runCompiled(t, nil, body, ret, nil)
	if got.Int != 6 {
		t.Errorf("got %d, want 6", got.Int)
	}
}

func TestCompile_VarDecl_NestedLetBindings(t *testing.T) {
	// Mirror let_bindings.ail nested(x):
	//   let a = x + 1
	//   let b = a * 2
	//   let c = b + a
	//   return c
	// nested(5) = (5+1)*2 + (5+1) = 12 + 6 = 18
	params := []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}}
	body := []stmt.Stmt{
		stmt.VarDecl{Name: "a", Value: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 1}}},
		stmt.VarDecl{Name: "b", Value: stmt.BinOp{Op: stmt.OpMul, Left: stmt.VarRef{Name: "a"}, Right: stmt.LitInt{Value: 2}}},
		stmt.VarDecl{Name: "c", Value: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "b"}, Right: stmt.VarRef{Name: "a"}}},
	}
	ret := stmt.VarRef{Name: "c"}
	got := runCompiled(t, params, body, ret, []bytecode.Value{bytecode.NewInt(5)})
	if got.Int != 18 {
		t.Errorf("nested(5): got %d, want 18", got.Int)
	}
}

func TestCompile_VarDecl_WithRebind(t *testing.T) {
	// withRebind(x):
	//   let a = x + 1
	//   let b = a * 2
	//   return a + b
	// withRebind(5) = 6 + 12 = 18
	params := []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}}
	body := []stmt.Stmt{
		stmt.VarDecl{Name: "a", Value: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 1}}},
		stmt.VarDecl{Name: "b", Value: stmt.BinOp{Op: stmt.OpMul, Left: stmt.VarRef{Name: "a"}, Right: stmt.LitInt{Value: 2}}},
	}
	ret := stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "a"}, Right: stmt.VarRef{Name: "b"}}
	got := runCompiled(t, params, body, ret, []bytecode.Value{bytecode.NewInt(5)})
	if got.Int != 18 {
		t.Errorf("withRebind(5): got %d, want 18", got.Int)
	}
}

// --- Constant pool dedup ----------------------------------------------------

func TestCompile_ConstantPoolDedup(t *testing.T) {
	// 1 + 1 + 1 should produce a single Int(1) constant in the local table.
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:     "test",
				Exported: true,
				Return: stmt.BinOp{
					Op:   stmt.OpAdd,
					Left: stmt.LitInt{Value: 1},
					Right: stmt.BinOp{
						Op:    stmt.OpAdd,
						Left:  stmt.LitInt{Value: 1},
						Right: stmt.LitInt{Value: 1},
					},
				},
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	proto := img.Prototypes[0]
	if len(proto.Constants) != 1 {
		t.Errorf("local constant table: got %d entries, want 1 (deduplication failed): %v", len(proto.Constants), proto.Constants)
	}
	if len(img.Constants) != 1 {
		t.Errorf("image pool: got %d entries, want 1: %v", len(img.Constants), img.Constants)
	}
}

// --- Multi-function program -------------------------------------------------

func TestCompile_MultiFunction_PicksFirstExportedAsEntry(t *testing.T) {
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{Name: "internal_helper", Return: stmt.LitInt{Value: 1}, Exported: false},
			{Name: "entry", Return: stmt.LitInt{Value: 42}, Exported: true},
			{Name: "other", Return: stmt.LitInt{Value: 99}, Exported: true},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if img.EntryPoint != 1 {
		t.Errorf("entry point: got %d, want 1 (first exported)", img.EntryPoint)
	}
	machine := vm.NewVM(img)
	got, err := machine.Run(img.Prototypes[img.EntryPoint], nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got.Int != 42 {
		t.Errorf("got %d, want 42", got.Int)
	}
}
