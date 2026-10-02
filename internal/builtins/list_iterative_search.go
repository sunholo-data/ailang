package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-ITERATIVE-LIST, continued (#1518, #1501): any, findIndex, foldr and
// mapAccumL as iterative Go loops. Their std/list forms recursed on
// `[x, ...rest]`, which copies the tail on the bytecode VM (O(n^2)), and
// foldr was not tail recursive at all (RT_REC_003 on the interpreter).
//
// The VM ports live in internal/vm/builtins_hof_list.go and must agree with
// these on every input (internal/vm/list_hof_parity_test.go).

func init() {
	registerListHOF("_list_any", 2, makeListAnyType, listAnyImpl,
		"True if any element satisfies the predicate; stops at the first match (iterative)",
		[]ParamDoc{{Name: "p", Description: "Predicate"}, {Name: "xs", Description: "Input list"}},
		"true if p(x) holds for some x in xs; false for []")
	registerListHOF("_list_findIndex", 2, makeListFindIndexType, listFindIndexImpl,
		"Index of the first element satisfying the predicate; stops at the first match (iterative)",
		[]ParamDoc{{Name: "p", Description: "Predicate"}, {Name: "xs", Description: "Input list"}},
		"Some(i) for the first i with p(xs[i]), or None")
	registerListHOF("_list_foldr", 3, makeListFoldrType, listFoldrImpl,
		"Right fold over a list with an accumulator (iterative, right-to-left)",
		[]ParamDoc{
			{Name: "f", Description: "Step (elem, acc) -> acc"},
			{Name: "acc", Description: "Initial accumulator (combined with the last element first)"},
			{Name: "xs", Description: "Input list"},
		},
		"f(x0, f(x1, ... f(xn, acc)))")
	registerListHOF("_list_mapAccumL", 3, makeListMapAccumLType, listMapAccumLImpl,
		"Stateful map in one left-to-right pass; the output list is one allocation (iterative)",
		[]ParamDoc{
			{Name: "f", Description: "Step (state, elem) -> (output, newState)"},
			{Name: "s0", Description: "Initial state"},
			{Name: "xs", Description: "Input list"},
		},
		"(outputs, finalState); ([], s0) for []")
}

func registerListHOF(name string, numArgs int, typ func() types.Type,
	impl func(*effects.EffContext, []eval.Value) (eval.Value, error),
	desc string, params []ParamDoc, returns string) {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    name,
		NumArgs: numArgs,
		IsPure:  true,
		Type:    typ,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: desc,
			Params:      params,
			Returns:     returns,
			Since:       "v0.51.1",
			Stability:   StabilityStable,
			Tags:        []string{"list", "iterative", "performance"},
			Category:    "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// Type: forall a. (a -> bool, [a]) -> bool
func makeListAnyType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	pred := T.Func(a).Returns(T.Bool()).Build()
	return T.Func(pred, T.List(a)).Returns(T.Bool()).Build()
}

// Type: forall a. (a -> bool, [a]) -> Option[int]
func makeListFindIndexType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	pred := T.Func(a).Returns(T.Bool()).Build()
	return T.Func(pred, T.List(a)).Returns(T.App("Option", T.Int())).Build()
}

// Type: forall a b. ((a, b) -> b, b, [a]) -> b
func makeListFoldrType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	b := T.Var("b")
	fn := T.Func(a, b).Returns(b).Build()
	return T.Func(fn, b, T.List(a)).Returns(b).Build()
}

// Type: forall a b s. ((s, a) -> (b, s), s, [a]) -> ([b], s)
func makeListMapAccumLType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	b := T.Var("b")
	s := T.Var("s")
	step := T.Func(s, a).Returns(&types.TTuple{Elements: []types.Type{b, s}}).Build()
	result := &types.TTuple{Elements: []types.Type{T.List(b), s}}
	return T.Func(step, s, T.List(a)).Returns(result).Build()
}

// listPredicateIndex returns the index of the first element of args[1] for
// which the predicate args[0] returns true, or -1. Shared by any/findIndex.
func listPredicateIndex(name string, ctx *effects.EffContext, args []eval.Value) (int, error) {
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return 0, fmt.Errorf("%s: expected List for second argument, got %T", name, args[1])
	}
	if ctx == nil || ctx.FnCaller == nil {
		return 0, fmt.Errorf("%s: FnCaller not set (evaluator not wired)", name)
	}
	for i, elem := range list.Elements {
		val, err := ctx.FnCaller(args[0], elem)
		if err != nil {
			return 0, callbackErr(err, "%s: callback error at index %d", name, i)
		}
		b, ok := val.(*eval.BoolValue)
		if !ok {
			return 0, fmt.Errorf("%s: predicate must return bool, got %T at index %d", name, val, i)
		}
		if b.Value {
			return i, nil
		}
	}
	return -1, nil
}

func listAnyImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	i, err := listPredicateIndex("_list_any", ctx, args)
	if err != nil {
		return nil, err
	}
	return &eval.BoolValue{Value: i >= 0}, nil
}

func listFindIndexImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	i, err := listPredicateIndex("_list_findIndex", ctx, args)
	if err != nil {
		return nil, err
	}
	if i < 0 {
		return mapMakeNone(), nil
	}
	return mapMakeSome(&eval.IntValue{Value: i}), nil
}

func listFoldrImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	fn, acc := args[0], args[1]
	list, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_foldr: expected List for third argument, got %T", args[2])
	}
	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_list_foldr: FnCallerN not set (evaluator not wired)")
	}
	for i := len(list.Elements) - 1; i >= 0; i-- {
		var err error
		acc, err = ctx.FnCallerN(fn, []eval.Value{list.Elements[i], acc})
		if err != nil {
			return nil, callbackErr(err, "_list_foldr: callback error at index %d", i)
		}
	}
	return acc, nil
}

func listMapAccumLImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	fn, state := args[0], args[1]
	list, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_mapAccumL: expected List for third argument, got %T", args[2])
	}
	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_list_mapAccumL: FnCallerN not set (evaluator not wired)")
	}
	out := make([]eval.Value, len(list.Elements))
	for i, elem := range list.Elements {
		val, err := ctx.FnCallerN(fn, []eval.Value{state, elem})
		if err != nil {
			return nil, callbackErr(err, "_list_mapAccumL: callback error at index %d", i)
		}
		pair, ok := val.(*eval.TupleValue)
		if !ok || len(pair.Elements) != 2 {
			return nil, fmt.Errorf("_list_mapAccumL: step must return a pair (output, state), got %T at index %d", val, i)
		}
		out[i], state = pair.Elements[0], pair.Elements[1]
	}
	return &eval.TupleValue{Elements: []eval.Value{&eval.ListValue{Elements: out}, state}}, nil
}
