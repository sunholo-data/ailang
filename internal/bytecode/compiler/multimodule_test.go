package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/gen/stmt"
)

// TestMultiModule_NameCollision verifies that functions with the same bare
// name but different Module fields do not collide in the compiler's funcIdx.
// Both must appear as distinct prototypes in the resulting image, and a
// cross-module GlobalRef must resolve to the correct one.
//
// This is the M1+M2 foundation test: without canonical-name keying, the
// second registration silently overwrites the first and one of the
// functions disappears from the image.
func TestMultiModule_NameCollision(t *testing.T) {
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Module: "lib/a",
				Name:   "helper",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.BinOp{
					Op:    stmt.OpAdd,
					Left:  stmt.VarRef{Name: "x"},
					Right: stmt.LitInt{Value: 1},
				},
			},
			{
				Module: "lib/b",
				Name:   "helper",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.BinOp{
					Op:    stmt.OpAdd,
					Left:  stmt.VarRef{Name: "x"},
					Right: stmt.LitInt{Value: 100},
				},
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	// Count non-EvalOnly prototypes whose canonical name ends in ".helper".
	var helperCount int
	for _, p := range img.Prototypes {
		if p.EvalOnly {
			continue
		}
		if p.Name == "lib/a.helper" || p.Name == "lib/b.helper" {
			helperCount++
		}
	}
	if helperCount < 2 {
		t.Fatalf("expected 2 helper prototypes (one per module), got %d", helperCount)
	}
}

// TestMultiModule_CrossModuleCall verifies that a GlobalRef call from one
// module to a function in another module resolves correctly at compile
// time and executes without bridging to the evaluator.
func TestMultiModule_CrossModuleCall(t *testing.T) {
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Module: "lib/math",
				Name:   "double",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.BinOp{
					Op:    stmt.OpMul,
					Left:  stmt.VarRef{Name: "x"},
					Right: stmt.LitInt{Value: 2},
				},
			},
			{
				Module: "main",
				Name:   "run",
				Return: stmt.Call{
					// Cross-module GlobalRef — compiler must resolve this
					// to the math.double prototype, not bridge.
					Func: stmt.GlobalRef{Module: "lib/math", Name: "double"},
					Args: []stmt.Expr{stmt.LitInt{Value: 21}},
				},
				Exported: true,
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	// Find the main.run prototype and assert it's NOT EvalOnly.
	var runProto string
	for _, p := range img.Prototypes {
		if p.Name == "main.run" {
			if p.EvalOnly {
				t.Fatalf("run was bridged to evaluator: %s", p.EvalReason)
			}
			runProto = p.Name
		}
	}
	if runProto == "" {
		t.Fatalf("main.run prototype not found in image")
	}

	// The lib/math.double prototype must also exist and not be EvalOnly.
	var doubleFound bool
	for _, p := range img.Prototypes {
		if p.Name == "lib/math.double" && !p.EvalOnly {
			doubleFound = true
		}
	}
	if !doubleFound {
		t.Fatalf("double prototype missing or bridged")
	}
}

// TestMultiModule_SameModuleVarRef verifies that an intra-module bare
// VarRef still resolves correctly after canonical-name keying is
// introduced. The compiler needs to know which module the currently-
// compiling function belongs to in order to canonicalize VarRef lookups.
func TestMultiModule_SameModuleVarRef(t *testing.T) {
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Module: "lib/a",
				Name:   "inc",
				Params: []stmt.Param{{Name: "x", Type: stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				Return: stmt.BinOp{
					Op:    stmt.OpAdd,
					Left:  stmt.VarRef{Name: "x"},
					Right: stmt.LitInt{Value: 1},
				},
			},
			{
				Module: "lib/a",
				Name:   "twice",
				Return: stmt.Call{
					// Bare VarRef — lowered from a same-module Core.Var.
					// Must resolve to lib/a.inc, not bridge.
					Func: stmt.VarRef{Name: "inc"},
					Args: []stmt.Expr{stmt.LitInt{Value: 41}},
				},
				Exported: true,
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	for _, p := range img.Prototypes {
		if p.Name == "lib/a.twice" && p.EvalOnly {
			t.Fatalf("twice was bridged: %s", p.EvalReason)
		}
	}
}

// TestMultiModule_UnknownGlobalStillBridges preserves the M3 bridge
// behavior: calling an undefined global (e.g. a stdlib function we haven't
// lowered into the program) still produces an EvalOnly stub rather than
// crashing the whole compile.
func TestMultiModule_UnknownGlobalStillBridges(t *testing.T) {
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Module: "main",
				Name:   "run",
				Return: stmt.Call{
					Func: stmt.GlobalRef{Module: "std/io", Name: "println"},
					Args: []stmt.Expr{stmt.LitString{Value: "hi"}},
				},
				Exported: true,
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var ran *struct {
		name     string
		evalOnly bool
		reason   string
	}
	for _, p := range img.Prototypes {
		if p.Name == "main.run" {
			ran = &struct {
				name     string
				evalOnly bool
				reason   string
			}{p.Name, p.EvalOnly, p.EvalReason}
		}
	}
	if ran == nil {
		t.Fatalf("run prototype not found")
	}
	if !ran.evalOnly {
		t.Fatalf("expected run to be EvalOnly (unknown global), got compiled prototype")
	}
	if !strings.Contains(ran.reason, "unknown global") {
		t.Fatalf("expected EvalReason mentioning 'unknown global', got %q", ran.reason)
	}
}
