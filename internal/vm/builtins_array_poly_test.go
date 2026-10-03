package vm

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Native polymorphic std/array builtins (stapledons_godot report 2026-10-01:
// std/array was unusable under --strict-bytecode). Same parity discipline as
// builtins_list_poly_test.go: VM function vs the registered evaluator Impl.

func floats(xs ...float64) bytecode.Value { return bytecode.NewFloatArray(xs) }

func boxed(xs ...int64) bytecode.Value {
	vs := make([]bytecode.Value, len(xs))
	for i, x := range xs {
		vs[i] = bytecode.NewInt(x)
	}
	return bytecode.NewArray(vs)
}

func pair(i int64, v bytecode.Value) bytecode.Value {
	return bytecode.NewTuple([]bytecode.Value{bytecode.NewInt(i), v})
}

func f64(x float64) bytecode.Value { return bytecode.NewFloat(x) }

func TestPolyArrayBuiltinsMatchEvaluator(t *testing.T) {
	cases := []struct {
		ir   string
		args []bytecode.Value
	}{
		{"__array_from_list", []bytecode.Value{ints(5, 6, 7)}},
		{"__array_from_list", []bytecode.Value{ints()}},
		{"__array_from_list", []bytecode.Value{bytecode.NewList([]bytecode.Value{f64(1.5), f64(2)})}},
		{"__array_to_list", []bytecode.Value{boxed(1, 2)}},
		{"__array_to_list", []bytecode.Value{floats(1.5, 2.5)}},
		{"__array_get", []bytecode.Value{boxed(5, 6, 7), i64(1)}},
		{"__array_get", []bytecode.Value{floats(1.5, 2.5), i64(0)}},
		{"__array_get", []bytecode.Value{boxed(5, 6, 7), i64(3)}},
		{"__array_get", []bytecode.Value{boxed(5, 6, 7), i64(-1)}},
		{"__array_unsafe_get", []bytecode.Value{boxed(5, 6, 7), i64(2)}},
		{"__array_unsafe_get", []bytecode.Value{boxed(), i64(0)}},
		{"__array_length", []bytecode.Value{boxed(5, 6, 7)}},
		{"__array_length", []bytecode.Value{floats(1, 2)}},
		{"__array_set", []bytecode.Value{boxed(5, 6, 7), i64(1), i64(9)}},
		{"__array_set", []bytecode.Value{floats(1, 2), i64(0), f64(9.5)}},
		{"__array_set", []bytecode.Value{boxed(5, 6, 7), i64(7), i64(9)}},
		{"__array_make", []bytecode.Value{i64(3), i64(4)}},
		{"__array_make", []bytecode.Value{i64(2), f64(0.5)}},
		{"__array_make", []bytecode.Value{i64(0), i64(4)}},
		{"__array_make", []bytecode.Value{i64(-1), i64(4)}},
		{"__array_append", []bytecode.Value{boxed(1, 2), i64(3)}},
		{"__array_append", []bytecode.Value{floats(1.5), f64(2.5)}},
		{"__array_append", []bytecode.Value{boxed(), i64(3)}},
		{"__array_update_many", []bytecode.Value{boxed(1, 2, 3), bytecode.NewList([]bytecode.Value{pair(0, i64(9)), pair(0, i64(8)), pair(2, i64(7))})}},
		{"__array_update_many", []bytecode.Value{floats(1, 2), bytecode.NewList([]bytecode.Value{pair(1, f64(5.5))})}},
		{"__array_update_many", []bytecode.Value{boxed(1, 2), bytecode.NewList([]bytecode.Value{pair(2, i64(9))})}},
		{"__array_update_many", []bytecode.Value{boxed(1, 2), bytecode.NewList(nil)}},
		{"__array_update_many", []bytecode.Value{boxed(1, 2), bytecode.NewList([]bytecode.Value{pair(-1, i64(9))})}},
	}
	for _, c := range cases {
		got, vmErr := nativeBuiltin(t, c.ir)(c.args)

		spec, ok := builtins.GetSpec(c.ir[1:])
		if !ok {
			t.Fatalf("%s: not in registry", c.ir)
		}
		evArgs := make([]eval.Value, len(c.args))
		for i, a := range c.args {
			ev, err := BytecodeToEval(a)
			if err != nil {
				t.Fatalf("%s: convert arg %d: %v", c.ir, i, err)
			}
			evArgs[i] = ev
		}
		want, evErr := spec.Impl(nil, evArgs)

		if (vmErr != nil) != (evErr != nil) {
			t.Errorf("%s%v: vm err=%v, evaluator err=%v", c.ir, c.args, vmErr, evErr)
			continue
		}
		if vmErr != nil {
			continue
		}
		wantBC, err := EvalToBytecode(want)
		if err != nil {
			t.Fatalf("%s: convert result: %v", c.ir, err)
		}
		if !runtimeEq(got, wantBC) {
			t.Errorf("%s%v = %s, evaluator says %s", c.ir, c.args, got, wantBC)
		}
		if got.Tag == bytecode.TagArray && wantBC.Tag == bytecode.TagArray &&
			(got.AsArray().Floats != nil) != (wantBC.AsArray().Floats != nil) {
			t.Errorf("%s%v: packed=%v, evaluator packed=%v", c.ir, c.args,
				got.AsArray().Floats != nil, wantBC.AsArray().Floats != nil)
		}
	}
}

func TestPolyArrayEmpty(t *testing.T) {
	for _, args := range [][]bytecode.Value{nil, {bytecode.Unit()}} {
		got, err := nativeBuiltin(t, "__array_empty")(args)
		if err != nil || got.Tag != bytecode.TagArray || got.AsArray().Len() != 0 {
			t.Errorf("__array_empty(%v) = %v, %v", args, got, err)
		}
	}
}

// TestPolyArrayCarriesADTs: the point of a native port.
func TestPolyArrayCarriesADTs(t *testing.T) {
	some := bytecode.NewADT(0, "Some", []bytecode.Value{bytecode.NewInt(1)})
	arr, err := nativeBuiltin(t, "__array_from_list")([]bytecode.Value{bytecode.NewList([]bytecode.Value{some})})
	if err != nil {
		t.Fatal(err)
	}
	got, err := nativeBuiltin(t, "__array_get")([]bytecode.Value{arr, i64(0)})
	if err != nil || !runtimeEq(got, some) {
		t.Errorf("get(fromList([Some 1]), 0) = %v, %v", got, err)
	}
}
