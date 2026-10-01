package vm

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Native polymorphic list ports (#1447, M3). Each case runs the VM function
// and the evaluator's registered Impl on the same arguments and requires the
// same result, or an error from both.

func ints(xs ...int64) bytecode.Value {
	vs := make([]bytecode.Value, len(xs))
	for i, x := range xs {
		vs[i] = bytecode.NewInt(x)
	}
	return bytecode.NewList(vs)
}

func i64(n int64) bytecode.Value { return bytecode.NewInt(n) }

func nativeBuiltin(t *testing.T, ir string) BuiltinFunc {
	t.Helper()
	for i, n := range bytecode.BuiltinNames {
		if n == ir {
			return BuiltinTable[i]
		}
	}
	t.Fatalf("%s is not a native VM builtin", ir)
	return nil
}

func TestPolyListBuiltinsMatchEvaluator(t *testing.T) {
	nested := bytecode.NewList([]bytecode.Value{ints(1), ints(), ints(2, 3)})
	cases := []struct {
		ir   string
		args []bytecode.Value
	}{
		{"__list_reverse", []bytecode.Value{ints(1, 2, 3)}},
		{"__list_reverse", []bytecode.Value{ints()}},
		{"__list_reverse", []bytecode.Value{nested}},
		{"__list_take", []bytecode.Value{i64(2), ints(1, 2, 3)}},
		{"__list_take", []bytecode.Value{i64(0), ints(1, 2, 3)}},
		{"__list_take", []bytecode.Value{i64(-1), ints(1, 2, 3)}},
		{"__list_take", []bytecode.Value{i64(9), ints(1, 2, 3)}},
		{"__list_drop", []bytecode.Value{i64(1), ints(1, 2, 3)}},
		{"__list_drop", []bytecode.Value{i64(-2), ints(1, 2, 3)}},
		{"__list_drop", []bytecode.Value{i64(3), ints(1, 2, 3)}},
		{"__list_drop", []bytecode.Value{i64(9), ints()}},
		{"__list_zip", []bytecode.Value{ints(1, 2, 3), ints(7, 8)}},
		{"__list_zip", []bytecode.Value{ints(), ints(7)}},
		{"__list_contains", []bytecode.Value{ints(1, 2, 3), i64(2)}},
		{"__list_contains", []bytecode.Value{ints(1, 2, 3), i64(5)}},
		{"__list_contains", []bytecode.Value{nested, ints(2, 3)}},
		{"__list_head", []bytecode.Value{ints(4, 5)}},
		{"__list_head", []bytecode.Value{ints()}},
		{"__list_extract", []bytecode.Value{ints(1, 2, 3, 4), i64(1), i64(2)}},
		{"__list_extract", []bytecode.Value{ints(1, 2, 3, 4), i64(-3), i64(2)}},
		{"__list_extract", []bytecode.Value{ints(1, 2, 3, 4), i64(2), i64(9)}},
		{"__list_extract", []bytecode.Value{ints(1, 2, 3, 4), i64(1), i64(-1)}},
		{"__list_extract", []bytecode.Value{ints(1, 2, 3, 4), i64(4), i64(1)}},
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
	}
}

// TestPolyListBuiltinsCarryADTs is the reason these are native rather than
// adapted: elements the converter cannot name (ADTs, closures) must pass
// through untouched.
func TestPolyListBuiltinsCarryADTs(t *testing.T) {
	some := func(n int64) bytecode.Value { return bytecode.NewADT(0, []bytecode.Value{bytecode.NewInt(n)}) }
	xs := bytecode.NewList([]bytecode.Value{some(1), some(2), bytecode.NewADT(1, nil)})

	got, err := nativeBuiltin(t, "__list_reverse")([]bytecode.Value{xs})
	if err != nil {
		t.Fatal(err)
	}
	if l := got.AsList(); len(l) != 3 || l[0].AsADT().Tag != 1 || !runtimeEq(l[2], some(1)) {
		t.Errorf("reverse of ADT list = %s", got)
	}
	has, err := nativeBuiltin(t, "__list_contains")([]bytecode.Value{xs, some(2)})
	if err != nil || !has.Bool {
		t.Errorf("contains(Some 2) = %v, %v; want true", has, err)
	}
}
