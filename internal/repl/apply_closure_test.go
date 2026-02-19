package repl

import (
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

func TestApplyClosure_NilFunction(t *testing.T) {
	r := New()
	_, err := r.ApplyClosure(nil, []eval.Value{})
	if err == nil {
		t.Fatal("expected error for nil function")
	}
}

func TestApplyClosure_NonFunction(t *testing.T) {
	r := New()
	_, err := r.ApplyClosure(&eval.IntValue{Value: 42}, []eval.Value{&eval.IntValue{Value: 1}})
	if err == nil {
		t.Fatal("expected error for non-function value")
	}
}

func TestApplyClosure_BuiltinFunction(t *testing.T) {
	r := New()
	// Create a simple builtin that returns its argument
	identity := &eval.BuiltinFunction{
		Name: "identity",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return args[0], nil
		},
	}
	result, err := r.ApplyClosure(identity, []eval.Value{&eval.IntValue{Value: 42}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}
	if intVal.Value != 42 {
		t.Fatalf("expected 42, got %d", intVal.Value)
	}
}

func TestApplyClosure_ZeroArgs(t *testing.T) {
	r := New()
	// A "thunk" that ignores its arg and returns a constant
	thunk := &eval.BuiltinFunction{
		Name: "thunk",
		Fn: func(args []eval.Value) (eval.Value, error) {
			return &eval.StringValue{Value: "hello"}, nil
		},
	}
	result, err := r.ApplyClosure(thunk, []eval.Value{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	strVal, ok := result.(*eval.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", result)
	}
	if strVal.Value != "hello" {
		t.Fatalf("expected 'hello', got %q", strVal.Value)
	}
}
