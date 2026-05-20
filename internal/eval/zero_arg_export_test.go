package eval

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestCallFunction_ZeroArgUnitInjection is a regression test for M-ZERO-ARG-SURFACES
// (v0.22.0). It exercises the central unit-injection behavior in CoreEvaluator.CallFunction
// directly, without going through any external surface wrapper (apiserver, REPL, bytecode).
// The contract: a FunctionValue with the parser's zero-arg shape (single param named "_")
// invoked with no args must succeed by binding UnitValue to the implicit slot.
//
// Background: AILANG compiles `export func f() -> T` to a FunctionValue with
// Params=["_"]. Three external surfaces (apiserver, WASM/REPL InvokeExport, bytecode
// CallEntrypoint) used to paper over this with their own workarounds — see commit
// history under M-S-CALL0 (8cc21027, 4075a402) plus the recent point fix this
// systemic fix replaces. Surface-local workarounds have been deleted; CallFunction
// is now the single source of truth.
func TestCallFunction_ZeroArgUnitInjection(t *testing.T) {
	e := NewCoreEvaluator()

	// fn = λ_. 42 -- exactly the shape the parser emits for `export func f() -> int { 42 }`
	body := &core.Lit{Kind: core.IntLit, Value: int64(42)}
	fn := &FunctionValue{
		Params: []string{"_"},
		Body:   body,
		Env:    NewEnvironment(),
	}

	// Calling with zero args MUST succeed and return 42.
	// Without the central injection, this would fail with "function expects 1 arguments, got 0".
	result, err := e.CallFunction(fn, nil)
	if err != nil {
		t.Fatalf("CallFunction(fn, nil) failed: %v", err)
	}
	intVal, ok := result.(*IntValue)
	if !ok {
		t.Fatalf("expected *IntValue, got %T", result)
	}
	if intVal.Value != 42 {
		t.Fatalf("expected 42, got %d", intVal.Value)
	}

	// Calling with an explicit UnitValue must still work (existing callers).
	result2, err := e.CallFunction(fn, []Value{&UnitValue{}})
	if err != nil {
		t.Fatalf("CallFunction(fn, [unit]) failed: %v", err)
	}
	if intVal2, ok := result2.(*IntValue); !ok || intVal2.Value != 42 {
		t.Fatalf("expected 42 with explicit unit arg, got %v", result2)
	}
}

// TestCallFunction_OneArgFunctionIsUnchanged verifies the heuristic doesn't
// false-positive on user-written `\x. body` lambdas with a non-"_" param —
// passing 0 args to a 1-arg function with a meaningful name must still error.
func TestCallFunction_OneArgFunctionIsUnchanged(t *testing.T) {
	e := NewCoreEvaluator()

	body := &core.Var{Name: "n"}
	fn := &FunctionValue{
		Params: []string{"n"},
		Body:   body,
		Env:    NewEnvironment(),
	}

	_, err := e.CallFunction(fn, nil)
	if err == nil {
		t.Fatal("expected error when calling 1-arg fn with 0 args, got nil")
	}
}
