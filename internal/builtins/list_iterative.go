package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// M-ITERATIVE-LIST: Go-level iterative list builtins
//
// These replace recursive AILANG implementations in std/list.ail with
// iterative Go loops. Benefits:
//   - O(1) stack usage (no recursion depth limit)
//   - ~10-100x faster (no evaluator frame overhead per element)
//   - Handles 50K+ element lists that would blow the recursion limit

func init() {
	registerListMap()
	registerListFilter()
	registerListFoldl()
}

// ============================================================================
// _list_map: Iterative map
// ============================================================================

func registerListMap() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_map",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListMapType,
		Impl:    listMapImpl,
		Metadata: &BuiltinMetadata{
			Description: "Apply a function to each element of a list (iterative)",
			LongDesc:    "Iterative Go-level implementation of map. Handles arbitrarily large lists without recursion limits. Each element is transformed by calling the provided function.",
			Params: []ParamDoc{
				{Name: "f", Description: "Function to apply to each element"},
				{Name: "xs", Description: "Input list"},
			},
			Returns:   "New list with f applied to each element",
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"list", "map", "iterative", "performance"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_map: %v", err))
	}
}

// Type: forall a b. (a -> b, list[a]) -> list[b]
func makeListMapType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	b := T.Var("b")
	listA := T.List(a)
	listB := T.List(b)
	fn := T.Func(a).Returns(b).Build()
	return T.Func(fn, listA).Returns(listB).Build()
}

func listMapImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	fn := args[0]
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_map: expected List for second argument, got %T", args[1])
	}

	if ctx == nil || ctx.FnCaller == nil {
		return nil, fmt.Errorf("_list_map: FnCaller not set (evaluator not wired)")
	}

	result := make([]eval.Value, len(list.Elements))
	for i, elem := range list.Elements {
		val, err := ctx.FnCaller(fn, elem)
		if err != nil {
			return nil, fmt.Errorf("_list_map: callback error at index %d: %w", i, err)
		}
		result[i] = val
	}
	return &eval.ListValue{Elements: result}, nil
}

// ============================================================================
// _list_filter: Iterative filter
// ============================================================================

func registerListFilter() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_filter",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListFilterType,
		Impl:    listFilterImpl,
		Metadata: &BuiltinMetadata{
			Description: "Keep elements that satisfy a predicate (iterative)",
			LongDesc:    "Iterative Go-level implementation of filter. Handles arbitrarily large lists without recursion limits. Elements for which the predicate returns true are kept.",
			Params: []ParamDoc{
				{Name: "p", Description: "Predicate function"},
				{Name: "xs", Description: "Input list"},
			},
			Returns:   "New list containing only elements where p returns true",
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"list", "filter", "iterative", "performance"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_filter: %v", err))
	}
}

// Type: forall a. (a -> bool, list[a]) -> list[a]
func makeListFilterType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	pred := T.Func(a).Returns(&types.TCon{Name: "bool"}).Build()
	return T.Func(pred, listA).Returns(listA).Build()
}

func listFilterImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	fn := args[0]
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_filter: expected List for second argument, got %T", args[1])
	}

	if ctx == nil || ctx.FnCaller == nil {
		return nil, fmt.Errorf("_list_filter: FnCaller not set (evaluator not wired)")
	}

	result := make([]eval.Value, 0, len(list.Elements))
	for i, elem := range list.Elements {
		val, err := ctx.FnCaller(fn, elem)
		if err != nil {
			return nil, fmt.Errorf("_list_filter: callback error at index %d: %w", i, err)
		}
		boolVal, ok := val.(*eval.BoolValue)
		if !ok {
			return nil, fmt.Errorf("_list_filter: predicate must return bool, got %T at index %d", val, i)
		}
		if boolVal.Value {
			result = append(result, elem)
		}
	}
	return &eval.ListValue{Elements: result}, nil
}

// ============================================================================
// _list_foldl: Iterative left fold
// ============================================================================

func registerListFoldl() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_foldl",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeListFoldlType,
		Impl:    listFoldlImpl,
		Metadata: &BuiltinMetadata{
			Description: "Left fold over a list with an accumulator (iterative)",
			LongDesc:    "Iterative Go-level implementation of foldl. Handles arbitrarily large lists without recursion limits. Processes elements left-to-right, threading an accumulator through each step.",
			Params: []ParamDoc{
				{Name: "f", Description: "Accumulation function (acc, elem) -> acc"},
				{Name: "acc", Description: "Initial accumulator value"},
				{Name: "xs", Description: "Input list"},
			},
			Returns:   "Final accumulator value after processing all elements",
			Since:     "v0.9.3",
			Stability: StabilityStable,
			Tags:      []string{"list", "fold", "reduce", "iterative", "performance"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_foldl: %v", err))
	}
}

// Type: forall a b. ((b, a) -> b, b, list[a]) -> b
func makeListFoldlType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	b := T.Var("b")
	listA := T.List(a)
	fn := T.Func(b, a).Returns(b).Build()
	return T.Func(fn, b, listA).Returns(b).Build()
}

func listFoldlImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	fn := args[0]
	acc := args[1]
	list, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_foldl: expected List for third argument, got %T", args[2])
	}

	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_list_foldl: FnCallerN not set (evaluator not wired)")
	}

	for i, elem := range list.Elements {
		var err error
		acc, err = ctx.FnCallerN(fn, []eval.Value{acc, elem})
		if err != nil {
			return nil, fmt.Errorf("_list_foldl: callback error at index %d: %w", i, err)
		}
	}
	return acc, nil
}
