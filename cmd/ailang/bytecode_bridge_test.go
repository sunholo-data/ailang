package main

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/eval"
)

// TestBridge_RoundTrip_Tier1 verifies that every Tier-1 value shape can be
// converted bytecode → eval → bytecode and back to a structurally equal
// value. This is the M-BYTECODE-2D M3 acceptance test for the value bridge.
func TestBridge_RoundTrip_Tier1(t *testing.T) {
	cases := []struct {
		name string
		v    bytecode.Value
	}{
		{"int_zero", bytecode.NewInt(0)},
		{"int_pos", bytecode.NewInt(42)},
		{"int_neg", bytecode.NewInt(-1234567890123)},
		{"float_zero", bytecode.NewFloat(0.0)},
		{"float_pi", bytecode.NewFloat(3.14159)},
		{"float_neg", bytecode.NewFloat(-2.5e10)},
		{"bool_true", bytecode.NewBool(true)},
		{"bool_false", bytecode.NewBool(false)},
		{"unit", bytecode.Unit()},
		{"string_empty", bytecode.NewString("")},
		{"string_ascii", bytecode.NewString("hello world")},
		{"string_unicode", bytecode.NewString("héllo 世界 🌍")},
		{"list_empty", bytecode.NewList(nil)},
		{"list_int", bytecode.NewList([]bytecode.Value{
			bytecode.NewInt(1), bytecode.NewInt(2), bytecode.NewInt(3),
		})},
		{"list_mixed_nested", bytecode.NewList([]bytecode.Value{
			bytecode.NewString("a"),
			bytecode.NewList([]bytecode.Value{bytecode.NewBool(true), bytecode.NewBool(false)}),
		})},
		{"tuple_pair", bytecode.NewTuple([]bytecode.Value{
			bytecode.NewInt(7), bytecode.NewString("seven"),
		})},
		{"tuple_triple", bytecode.NewTuple([]bytecode.Value{
			bytecode.NewBool(true), bytecode.NewFloat(1.5), bytecode.Unit(),
		})},
		{"record_basic", bytecode.NewRecord([]bytecode.RecordField{
			{Name: "name", Value: bytecode.NewString("Alice")},
			{Name: "age", Value: bytecode.NewInt(30)},
		})},
		{"record_nested", bytecode.NewRecord([]bytecode.RecordField{
			{Name: "items", Value: bytecode.NewList([]bytecode.Value{bytecode.NewInt(1), bytecode.NewInt(2)})},
			{Name: "header", Value: bytecode.NewRecord([]bytecode.RecordField{
				{Name: "title", Value: bytecode.NewString("hi")},
			})},
		})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := bytecodeValueToEval(tc.v)
			if err != nil {
				t.Fatalf("bytecodeValueToEval: %v", err)
			}
			back, err := evalValueToBytecode(ev)
			if err != nil {
				t.Fatalf("evalValueToBytecode: %v", err)
			}
			if !back.Equal(tc.v) {
				t.Errorf("round-trip mismatch:\n  in:  %s\n  out: %s", tc.v, back)
			}
		})
	}
}

// TestBridge_Unsupported_Closure_Errors_Both_Directions confirms that the
// bridge refuses to silently mis-convert function/closure values. This is
// the M3 honest scope-narrowing — closures are M-BYTECODE-2E scope and
// passing them across the boundary must be a loud error.
func TestBridge_Unsupported_Closure_Errors_Both_Directions(t *testing.T) {
	// bytecode → eval: a TagClosure should fail.
	c := bytecode.NewClosure(&bytecode.FuncPrototype{Name: "stub"}, nil)
	if _, err := bytecodeValueToEval(c); err == nil {
		t.Errorf("bytecodeValueToEval should reject TagClosure")
	}

	// eval → bytecode: a *FunctionValue should fail.
	fn := &eval.FunctionValue{Params: []string{"x"}}
	if _, err := evalValueToBytecode(fn); err == nil {
		t.Errorf("evalValueToBytecode should reject *FunctionValue")
	}
}

// TestBridge_Unsupported_ADT_Errors confirms that ADT crossings produce a
// clear error mentioning the deferred milestone, instead of corrupting tag
// information.
func TestBridge_Unsupported_ADT_Errors(t *testing.T) {
	// bytecode → eval
	v := bytecode.NewADT(0, []bytecode.Value{bytecode.NewInt(1)})
	_, err := bytecodeValueToEval(v)
	if err == nil {
		t.Errorf("bytecodeValueToEval should reject TagADT")
	}
	if err != nil && !strings.Contains(err.Error(), "ADT") {
		t.Errorf("error should mention ADT, got %v", err)
	}

	// eval → bytecode
	tagged := &eval.TaggedValue{TypeName: "Option", CtorName: "Some", Fields: []eval.Value{&eval.IntValue{Value: 1}}}
	_, err = evalValueToBytecode(tagged)
	if err == nil {
		t.Errorf("evalValueToBytecode should reject *TaggedValue")
	}
}
