package builtins

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

func TestIntToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected float64
	}{
		{"zero", 0, 0.0},
		{"positive", 42, 42.0},
		{"negative", -100, -100.0},
		{"large", 1000000000, 1000000000.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := []eval.Value{&eval.IntValue{Value: tc.input}}
			result, err := intToFloatImpl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			floatVal, ok := result.(*eval.FloatValue)
			if !ok {
				t.Fatalf("expected FloatValue, got %T", result)
			}

			if floatVal.Value != tc.expected {
				t.Errorf("expected %f, got %f", tc.expected, floatVal.Value)
			}
		})
	}
}

func TestIntToFloat_WrongType(t *testing.T) {
	args := []eval.Value{&eval.StringValue{Value: "not an int"}}
	_, err := intToFloatImpl(nil, args)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestFloatToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected int
	}{
		{"zero", 0.0, 0},
		{"positive", 42.0, 42},
		{"negative", -100.0, -100},
		{"truncate_down", 3.7, 3},
		{"truncate_up", -3.7, -3},
		{"large", 1000000000.5, 1000000000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := []eval.Value{&eval.FloatValue{Value: tc.input}}
			result, err := floatToIntImpl(nil, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			intVal, ok := result.(*eval.IntValue)
			if !ok {
				t.Fatalf("expected IntValue, got %T", result)
			}

			if intVal.Value != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, intVal.Value)
			}
		})
	}
}

func TestFloatToInt_WrongType(t *testing.T) {
	args := []eval.Value{&eval.StringValue{Value: "not a float"}}
	_, err := floatToIntImpl(nil, args)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestIntFloatRoundTrip(t *testing.T) {
	// Test round-trip: int -> float -> int
	original := 12345
	args1 := []eval.Value{&eval.IntValue{Value: original}}

	floatResult, err := intToFloatImpl(nil, args1)
	if err != nil {
		t.Fatalf("int_to_float error: %v", err)
	}

	args2 := []eval.Value{floatResult}
	intResult, err := floatToIntImpl(nil, args2)
	if err != nil {
		t.Fatalf("float_to_int error: %v", err)
	}

	finalVal := intResult.(*eval.IntValue).Value
	if finalVal != original {
		t.Errorf("round-trip failed: %d -> %d", original, finalVal)
	}
}
