package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/vm"
)

// runProgram compiles a multi-function program and runs the named entry.
func runProgram(t *testing.T, prog *stmt.Program, entry string, args []bytecode.Value) bytecode.Value {
	t.Helper()
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var entryProto *bytecode.FuncPrototype
	for _, p := range img.Prototypes {
		if p.Name == entry {
			entryProto = p
			break
		}
	}
	if entryProto == nil {
		t.Fatalf("entry %q not found", entry)
	}
	machine := vm.NewVM(img)
	got, err := machine.Run(entryProto, args)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return got
}

// --- Identity (zero-arg call via top-level reference) ----------------------

func TestCompile_Call_TopLevelIdentity(t *testing.T) {
	// identity(x: int) = x
	// caller() = identity(42)
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:   "identity",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.VarRef{Name: "x"},
			},
			{
				Name: "caller",
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "identity"},
					Args: []stmt.Expr{stmt.LitInt{Value: 42}},
				},
				Exported: true,
			},
		},
	}
	got := runProgram(t, prog, "caller", nil)
	if got.Int != 42 {
		t.Errorf("got %d, want 42", got.Int)
	}
}

// --- Recursive factorial ---------------------------------------------------

func TestCompile_Recursive_Factorial(t *testing.T) {
	// factorial(n) = if n <= 1 then 1 else n * factorial(n-1)
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:   "factorial",
				Params: []stmt.Param{{Name: "n", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.IfExpr{
					Cond: stmt.BinOp{Op: stmt.OpLte, Left: stmt.VarRef{Name: "n"}, Right: stmt.LitInt{Value: 1}},
					Then: stmt.LitInt{Value: 1},
					Else: stmt.BinOp{
						Op:   stmt.OpMul,
						Left: stmt.VarRef{Name: "n"},
						Right: stmt.Call{
							Func: stmt.VarRef{Name: "factorial"},
							Args: []stmt.Expr{stmt.BinOp{Op: stmt.OpSub, Left: stmt.VarRef{Name: "n"}, Right: stmt.LitInt{Value: 1}}},
						},
					},
				},
				Exported: true,
			},
		},
	}
	cases := []struct{ n, want int64 }{
		{0, 1}, {1, 1}, {2, 2}, {3, 6}, {5, 120}, {10, 3628800},
	}
	for _, tc := range cases {
		got := runProgram(t, prog, "factorial", []bytecode.Value{bytecode.NewInt(tc.n)})
		if got.Int != tc.want {
			t.Errorf("factorial(%d): got %d, want %d", tc.n, got.Int, tc.want)
		}
	}
}

// --- First-class function (apply) -----------------------------------------

func TestCompile_Apply_FirstClassFunc(t *testing.T) {
	// double(x) = x * 2
	// apply(f, x) = f(x)
	// caller() = apply(double, 21)
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:   "double",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.BinOp{Op: stmt.OpMul, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 2}},
			},
			{
				Name: "apply",
				Params: []stmt.Param{
					{Name: "f", Type: stmt.FuncType{}},
					{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}},
				},
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "f"},
					Args: []stmt.Expr{stmt.VarRef{Name: "x"}},
				},
			},
			{
				Name: "caller",
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "apply"},
					Args: []stmt.Expr{stmt.VarRef{Name: "double"}, stmt.LitInt{Value: 21}},
				},
				Exported: true,
			},
		},
	}
	got := runProgram(t, prog, "caller", nil)
	if got.Int != 42 {
		t.Errorf("got %d, want 42", got.Int)
	}
}

// --- Tail call: countdown sanity ------------------------------------------

func TestCompile_TailCall_Countdown(t *testing.T) {
	// countdown(n) = if n == 0 then 0 else countdown(n - 1)
	// Tail-recursive: should run with bounded stack.
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:   "countdown",
				Params: []stmt.Param{{Name: "n", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.IfExpr{
					Cond: stmt.BinOp{Op: stmt.OpEq, Left: stmt.VarRef{Name: "n"}, Right: stmt.LitInt{Value: 0}},
					Then: stmt.LitInt{Value: 0},
					Else: stmt.Call{
						Func: stmt.VarRef{Name: "countdown"},
						Args: []stmt.Expr{stmt.BinOp{Op: stmt.OpSub, Left: stmt.VarRef{Name: "n"}, Right: stmt.LitInt{Value: 1}}},
					},
				},
				Exported: true,
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Verify TAIL_CALL was emitted (and not CALL+RETURN).
	proto := img.Prototypes[0]
	hasTailCall := false
	for _, inst := range proto.Instructions {
		if inst.Op() == bytecode.OpTailCall {
			hasTailCall = true
		}
	}
	if !hasTailCall {
		t.Errorf("countdown: expected OpTailCall, got instructions:")
		for i, inst := range proto.Instructions {
			t.Logf("  %2d: %s", i, inst)
		}
	}
	// Run with a tiny stack ceiling — if TCO is broken, this overflows.
	machine := vm.NewVM(img)
	machine.MaxStack = 5
	got, err := machine.Run(proto, []bytecode.Value{bytecode.NewInt(10000)})
	if err != nil {
		t.Fatalf("Run countdown(10000): %v", err)
	}
	if got.Int != 0 {
		t.Errorf("countdown(10000): got %d, want 0", got.Int)
	}
}

// --- Non-tail recursion overflows under tight stack -----------------------

func TestCompile_NonTailRecursion_OverflowsTightStack(t *testing.T) {
	// factorial is non-tail (multiplication after the recursive call).
	// Under a tight stack ceiling, deep input must error.
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:   "factorial",
				Params: []stmt.Param{{Name: "n", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.IfExpr{
					Cond: stmt.BinOp{Op: stmt.OpLte, Left: stmt.VarRef{Name: "n"}, Right: stmt.LitInt{Value: 1}},
					Then: stmt.LitInt{Value: 1},
					Else: stmt.BinOp{
						Op:   stmt.OpMul,
						Left: stmt.VarRef{Name: "n"},
						Right: stmt.Call{
							Func: stmt.VarRef{Name: "factorial"},
							Args: []stmt.Expr{stmt.BinOp{Op: stmt.OpSub, Left: stmt.VarRef{Name: "n"}, Right: stmt.LitInt{Value: 1}}},
						},
					},
				},
				Exported: true,
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	machine := vm.NewVM(img)
	machine.MaxStack = 3
	_, err = machine.Run(img.Prototypes[0], []bytecode.Value{bytecode.NewInt(50)})
	if err == nil || !strings.Contains(err.Error(), "stack") {
		t.Errorf("expected stack overflow, got err=%v", err)
	}
}

// --- compose example (let-binding chain inside body) -----------------------

func TestCompile_Compose(t *testing.T) {
	// compose(x) = let doubled = x*2 in let incremented = doubled+1 in incremented
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:   "compose",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Body: []stmt.Stmt{
					stmt.VarDecl{Name: "doubled", Value: stmt.BinOp{Op: stmt.OpMul, Left: stmt.VarRef{Name: "x"}, Right: stmt.LitInt{Value: 2}}},
					stmt.VarDecl{Name: "incremented", Value: stmt.BinOp{Op: stmt.OpAdd, Left: stmt.VarRef{Name: "doubled"}, Right: stmt.LitInt{Value: 1}}},
				},
				Return:   stmt.VarRef{Name: "incremented"},
				Exported: true,
			},
		},
	}
	got := runProgram(t, prog, "compose", []bytecode.Value{bytecode.NewInt(5)})
	if got.Int != 11 {
		t.Errorf("compose(5): got %d, want 11", got.Int)
	}
}
