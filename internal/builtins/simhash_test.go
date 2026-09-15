package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/simhash"
)

// Builtin implementation tests

func TestSimHashBuiltin(t *testing.T) {
	text := "hello world"
	args := []eval.Value{&eval.StringValue{Value: text}}

	result, err := simHashImpl(nil, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}

	// Verify it matches the Go implementation
	expected := simhash.Hash(text)
	if int64(intVal.Value) != expected {
		t.Errorf("builtin returned %d, expected %d", intVal.Value, expected)
	}
}

func TestHammingDistanceBuiltin(t *testing.T) {
	a := int64(12345)
	b := int64(12346)

	args := []eval.Value{
		&eval.IntValue{Value: int(a)},
		&eval.IntValue{Value: int(b)},
	}

	result, err := hammingDistanceImpl(nil, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}

	expected := simhash.HammingDistance(a, b)
	if intVal.Value != expected {
		t.Errorf("builtin returned %d, expected %d", intVal.Value, expected)
	}
}
