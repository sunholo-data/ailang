package vm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// The VM ports of _list_any/_list_findIndex/_list_foldr/_list_mapAccumL
// (builtins_hof_list.go) must agree with the evaluator's registered Impl on
// every input. Each case runs one Go callback through both engines' calling
// conventions and compares results, call counts and call order.

type hofParityCase struct {
	name string
	ir   string // VM HOF name; the registry name drops one leading "_"
	vm   HOFBuiltinFunc
	// cb is the callback, over bytecode values (converted for the evaluator)
	cb   func(args []bytecode.Value) (bytecode.Value, error)
	args []bytecode.Value // without the leading closure
}

func TestListHOFParityWithEvaluator(t *testing.T) {
	eq := func(want int64) func([]bytecode.Value) (bytecode.Value, error) {
		return func(a []bytecode.Value) (bytecode.Value, error) { return bytecode.NewBool(a[0].Int == want), nil }
	}
	digits := func(a []bytecode.Value) (bytecode.Value, error) { // foldr: acc*10 + x
		return bytecode.NewInt(a[1].Int*10 + a[0].Int), nil
	}
	running := func(a []bytecode.Value) (bytecode.Value, error) { // mapAccumL: (st*x, st+x)
		return bytecode.NewTuple([]bytecode.Value{bytecode.NewInt(a[0].Int * a[1].Int), bytecode.NewInt(a[0].Int + a[1].Int)}), nil
	}
	lists := []bytecode.Value{ints(), ints(4), ints(1, 7, 7, 3), ints(3, 1, 4, 1, 5, 9, 2, 6)}
	var cases []hofParityCase
	for i, xs := range lists {
		for _, want := range []int64{1, 7, 42} {
			cases = append(cases,
				hofParityCase{fmt.Sprintf("any/%d/%d", i, want), "__list_any", hofBuiltinListAny, eq(want), []bytecode.Value{xs}},
				hofParityCase{fmt.Sprintf("findIndex/%d/%d", i, want), "__list_findIndex", hofBuiltinListFindIndex, eq(want), []bytecode.Value{xs}})
		}
		cases = append(cases,
			hofParityCase{fmt.Sprintf("foldr/%d", i), "__list_foldr", hofBuiltinListFoldr, digits, []bytecode.Value{bytecode.NewInt(0), xs}},
			hofParityCase{fmt.Sprintf("mapAccumL/%d", i), "__list_mapAccumL", hofBuiltinListMapAccumL, running, []bytecode.Value{bytecode.NewInt(1), xs}})
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var vmTrace, evTrace []string
			caller := mockCaller(func(a []bytecode.Value) (bytecode.Value, error) {
				vmTrace = append(vmTrace, fmt.Sprint(a))
				return c.cb(a)
			})
			vmOut, err := c.vm(caller, append([]bytecode.Value{dummyClosure}, c.args...))
			if err != nil {
				t.Fatalf("vm: %v", err)
			}

			spec, ok := builtins.GetSpec(c.ir[1:])
			if !ok {
				t.Fatalf("%s is not registered for the evaluator", c.ir[1:])
			}
			ctx := effects.NewEffContext(nil)
			callN := func(_ eval.Value, args []eval.Value) (eval.Value, error) {
				bargs := make([]bytecode.Value, len(args))
				for i, a := range args {
					if bargs[i], err = EvalToBytecode(a); err != nil {
						return nil, err
					}
				}
				evTrace = append(evTrace, fmt.Sprint(bargs))
				r, err := c.cb(bargs)
				if err != nil {
					return nil, err
				}
				return BytecodeToEval(r)
			}
			ctx.FnCallerN = callN
			ctx.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) { return callN(fn, []eval.Value{arg}) }
			evArgs := []eval.Value{&eval.BuiltinFunction{Name: "cb"}}
			for _, a := range c.args {
				ea, err := BytecodeToEval(a)
				if err != nil {
					t.Fatal(err)
				}
				evArgs = append(evArgs, ea)
			}
			evOut, err := spec.Impl(ctx, evArgs)
			if err != nil {
				t.Fatalf("evaluator: %v", err)
			}

			vmAsEval, err := BytecodeToEvalForDisplay(vmOut)
			if err != nil {
				t.Fatal(err)
			}
			if vmAsEval.String() != evOut.String() {
				t.Errorf("result: vm %s, evaluator %s", vmAsEval, evOut)
			}
			if strings.Join(vmTrace, ";") != strings.Join(evTrace, ";") {
				t.Errorf("callback calls differ:\n vm  %v\n eval %v", vmTrace, evTrace)
			}
		})
	}
}

func TestListHOFErrors(t *testing.T) {
	boom := fmt.Errorf("boom")
	failing := mockCaller(func([]bytecode.Value) (bytecode.Value, error) { return bytecode.Value{}, boom })
	notBool := mockCaller(func([]bytecode.Value) (bytecode.Value, error) { return bytecode.NewInt(1), nil })
	for _, f := range []HOFBuiltinFunc{hofBuiltinListAny, hofBuiltinListFindIndex} {
		if _, err := f(failing, []bytecode.Value{dummyClosure, ints(1)}); err == nil || !strings.Contains(err.Error(), "boom") {
			t.Errorf("callback error not propagated: %v", err)
		}
		if _, err := f(notBool, []bytecode.Value{dummyClosure, ints(1)}); err == nil || !strings.Contains(err.Error(), "must return bool") {
			t.Errorf("non-bool predicate: %v", err)
		}
		if _, err := f(notBool, []bytecode.Value{dummyClosure, bytecode.NewInt(1)}); err == nil {
			t.Error("non-list accepted")
		}
	}
	if _, err := hofBuiltinListMapAccumL(notBool, []bytecode.Value{dummyClosure, bytecode.NewInt(0), ints(1)}); err == nil || !strings.Contains(err.Error(), "pair") {
		t.Errorf("non-pair step: %v", err)
	}
	if _, err := hofBuiltinListFoldr(failing, []bytecode.Value{dummyClosure, bytecode.NewInt(0), ints(1)}); err == nil {
		t.Error("foldr swallowed a callback error")
	}
}
