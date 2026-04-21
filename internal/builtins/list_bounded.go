package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-EVAL-BOUNDED-PIPELINE: Fused bounded list combinators
//
// These builtins fuse take+map and take+flatMap into single-pass operations
// with early exit. They avoid materializing full intermediate lists, preventing
// OOM on large inputs.
//
// Semantic note: For effectful f, takeFlatMap(n, f, xs) evaluates f only for
// as many input elements as needed to produce the first n output elements.
// This is intentional short-circuiting — the bounded behavior is explicit
// in the function name.

func init() {
	registerTakeMap()
	registerTakeFlatMap()
}

// ============================================================================
// _list_takeMap: Bounded map — map f over xs, stop after n results
// ============================================================================

func registerTakeMap() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_takeMap",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeTakeMapType,
		Impl:    takeMapImpl,
		Metadata: &BuiltinMetadata{
			Description: "Map a function over a list, collecting at most n results",
			LongDesc:    "Fused take+map: applies f to elements of xs, stopping as soon as n results are collected. Avoids materializing the full mapped list when only a prefix is needed.",
			Params: []ParamDoc{
				{Name: "n", Description: "Maximum number of results to collect"},
				{Name: "f", Description: "Function to apply to each element"},
				{Name: "xs", Description: "Input list"},
			},
			Returns:   "List of at most n elements, each the result of applying f",
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"list", "map", "take", "bounded", "performance"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_takeMap: %v", err))
	}
}

// Type: forall a b. (Int, a -> b, List[a]) -> List[b]
func makeTakeMapType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	b := T.Var("b")
	listA := T.List(a)
	listB := T.List(b)
	fn := T.Func(a).Returns(b).Build()
	intT := &types.TCon{Name: "Int"}
	return T.Func(intT, fn, listA).Returns(listB).Build()
}

func takeMapImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_list_takeMap: expected Int for first argument, got %T", args[0])
	}
	n := nVal.Value

	fn := args[1]

	list, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_takeMap: expected List for third argument, got %T", args[2])
	}

	if n <= 0 {
		return &eval.ListValue{Elements: []eval.Value{}}, nil
	}

	if ctx == nil || ctx.FnCaller == nil {
		return nil, fmt.Errorf("_list_takeMap: FnCaller not set (evaluator not wired)")
	}

	limit := n
	if limit > len(list.Elements) {
		limit = len(list.Elements)
	}
	result := make([]eval.Value, 0, limit)
	for i := 0; i < len(list.Elements) && len(result) < n; i++ {
		val, err := ctx.FnCaller(fn, list.Elements[i])
		if err != nil {
			return nil, fmt.Errorf("_list_takeMap: callback error at index %d: %w", i, err)
		}
		result = append(result, val)
	}
	return &eval.ListValue{Elements: result}, nil
}

// ============================================================================
// _list_takeFlatMap: Bounded flatMap — flatMap f over xs, stop after n results
// ============================================================================

func registerTakeFlatMap() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_takeFlatMap",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeTakeFlatMapType,
		Impl:    takeFlatMapImpl,
		Metadata: &BuiltinMetadata{
			Description: "FlatMap a function over a list, collecting at most n results",
			LongDesc:    "Fused take+flatMap: applies f to elements of xs (where f returns a list), flattening results and stopping as soon as n total output elements are collected. Avoids materializing the full flattened list when only a prefix is needed. For effectful f, only evaluates f for as many input elements as needed to produce n outputs.",
			Params: []ParamDoc{
				{Name: "n", Description: "Maximum number of output elements to collect"},
				{Name: "f", Description: "Function to apply to each element (must return a list)"},
				{Name: "xs", Description: "Input list"},
			},
			Returns:   "Flattened list of at most n elements",
			Since:     "v0.9.4",
			Stability: StabilityStable,
			Tags:      []string{"list", "flatMap", "take", "bounded", "performance"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_takeFlatMap: %v", err))
	}
}

// Type: forall a b. (Int, a -> List[b], List[a]) -> List[b]
func makeTakeFlatMapType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	b := T.Var("b")
	listA := T.List(a)
	listB := T.List(b)
	fn := T.Func(a).Returns(listB).Build()
	intT := &types.TCon{Name: "Int"}
	return T.Func(intT, fn, listA).Returns(listB).Build()
}

func takeFlatMapImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_list_takeFlatMap: expected Int for first argument, got %T", args[0])
	}
	n := nVal.Value

	fn := args[1]

	list, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_takeFlatMap: expected List for third argument, got %T", args[2])
	}

	if n <= 0 {
		return &eval.ListValue{Elements: []eval.Value{}}, nil
	}

	if ctx == nil || ctx.FnCaller == nil {
		return nil, fmt.Errorf("_list_takeFlatMap: FnCaller not set (evaluator not wired)")
	}

	result := make([]eval.Value, 0, n)
	for _, elem := range list.Elements {
		if len(result) >= n {
			break
		}
		innerVal, err := ctx.FnCaller(fn, elem)
		if err != nil {
			return nil, fmt.Errorf("_list_takeFlatMap: callback error: %w", err)
		}
		innerList, ok := innerVal.(*eval.ListValue)
		if !ok {
			return nil, fmt.Errorf("_list_takeFlatMap: callback must return a List, got %T", innerVal)
		}
		for _, inner := range innerList.Elements {
			result = append(result, inner)
			if len(result) >= n {
				break
			}
		}
	}
	return &eval.ListValue{Elements: result}, nil
}
