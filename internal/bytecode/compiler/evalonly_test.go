package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/vm"
)

// TestCompile_EvalOnly_PerFunctionTagging is the M-BYTECODE-2D M3 unit test
// for per-function compilable tagging. A program containing one compilable
// function and one function that the compiler can't lower must produce an
// image where:
//
//   - the compilable function is a normal prototype with bytecode
//   - the uncompilable function is marked EvalOnly with a non-empty reason
//   - the image still validates (Validate accepts EvalOnly stubs)
//
// We trigger the compile failure with a `_dict_foo` builtin call, which the
// compiler intentionally rejects via isLowerPassDictFallback in builtins.go.
// This is the path the lower pass would take if a polymorphic dictionary
// failed to monomorphize — we use it here precisely because it's the one
// path that's *guaranteed* to error during per-function compile.
func TestCompile_EvalOnly_PerFunctionTagging(t *testing.T) {
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				// good() is a trivial compilable function: returns 42.
				Name:     "good",
				Exported: true,
				Return:   stmt.LitInt{Value: 42},
			},
			{
				// bad() calls _dict_foo, which the compiler rejects in
				// isLowerPassDictFallback. This causes a per-function compile
				// error and (per M3) marks the prototype EvalOnly.
				Name:     "bad",
				Exported: true,
				Return: stmt.BuiltinCall{
					Name: "_dict_foo",
					Args: nil,
				},
			},
		},
	}

	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile returned error; per-function trapping should mask it: %v", err)
	}
	if err := img.Validate(); err != nil {
		t.Fatalf("Validate after mixed compile: %v", err)
	}

	var goodProto, badProto *bytecode.FuncPrototype
	for _, p := range img.Prototypes {
		switch p.Name {
		case "good":
			goodProto = p
		case "bad":
			badProto = p
		}
	}
	if goodProto == nil {
		t.Fatalf("good prototype missing")
	}
	if badProto == nil {
		t.Fatalf("bad prototype missing")
	}

	if goodProto.EvalOnly {
		t.Errorf("good was marked EvalOnly: reason=%q", goodProto.EvalReason)
	}
	if len(goodProto.Instructions) == 0 {
		t.Errorf("good has no instructions; expected a normal compiled body")
	}
	if !badProto.EvalOnly {
		t.Errorf("bad was NOT marked EvalOnly")
	}
	if badProto.EvalReason == "" {
		t.Errorf("bad EvalReason is empty")
	}
	if !strings.Contains(badProto.EvalReason, "_dict_foo") {
		t.Errorf("bad EvalReason should mention the offending builtin; got %q", badProto.EvalReason)
	}
	if len(badProto.Instructions) != 0 || len(badProto.LineInfo) != 0 {
		t.Errorf("bad EvalOnly stub has body: instr=%d line=%d", len(badProto.Instructions), len(badProto.LineInfo))
	}

	// good() must still execute correctly through the VM despite the sibling
	// stub: the rollback path should not have left img in a corrupt state.
	machine := vm.NewVM(img)
	got, err := machine.Run(goodProto, nil)
	if err != nil {
		t.Fatalf("Run good: %v", err)
	}
	if got.Tag != bytecode.TagInt || got.Int != 42 {
		t.Errorf("good returned %v, want Int(42)", got)
	}
}

// TestCompile_EvalOnly_VM_Trap_NoBridge verifies that calling an EvalOnly
// function through the VM without an interop bridge wired produces a clear
// error mentioning the function name and the original compile reason.
func TestCompile_EvalOnly_VM_Trap_NoBridge(t *testing.T) {
	// caller() simply returns bad(): the compiler emits an OpClosure +
	// OpCall referencing bad's prototype. At runtime the VM detects the
	// EvalOnly tag and traps.
	prog := &stmt.Program{
		FuncDecls: []stmt.FuncDecl{
			{
				Name:     "bad",
				Exported: false,
				Return: stmt.BuiltinCall{
					Name: "_dict_foo",
					Args: nil,
				},
			},
			{
				Name:     "caller",
				Exported: true,
				Return: stmt.Call{
					Func: stmt.VarRef{Name: "bad"},
				},
			},
		},
	}
	img, err := Compile(prog)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var callerProto *bytecode.FuncPrototype
	for _, p := range img.Prototypes {
		if p.Name == "caller" {
			callerProto = p
			break
		}
	}
	if callerProto == nil {
		t.Fatalf("caller prototype missing")
	}
	if callerProto.EvalOnly {
		t.Fatalf("caller should be compilable; it is just a Call to bad")
	}

	machine := vm.NewVM(img) // no Interop wired
	_, err = machine.Run(callerProto, nil)
	if err == nil {
		t.Fatalf("expected error trapping into EvalOnly without bridge")
	}
	if !strings.Contains(err.Error(), "evaluator-only") || !strings.Contains(err.Error(), "bad") {
		t.Errorf("error message missing expected text: %v", err)
	}
}
