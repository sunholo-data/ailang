package builtins

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
)

// #1318: interpreter implementations for std/list functions whose AILANG bodies
// were `[x] ++ recursive-call`. Lists are flat arrays, so each `++` copies: those
// bodies were O(n) deep and O(n²) in allocation (sortBy over 84k strings peaked
// at 99.7 GB and failed RT_REC_003 at 20k under the default limit). Before this,
// the three names below had Go codegen helpers only (registry_codegen_list.go),
// which the interpreter cannot call.

// effectPolyRow is shared by a callback's effect row and the builtin's own, so
// the builtin carries exactly the callback's effects — the same polymorphism the
// AILANG bodies had by inference. std/list.sortBy and flatMap accept effectful
// callbacks (docparse's epub parser passes flatMap an `! {FS}` lambda), so a
// closed pure row here would be a breaking change.
const effectPolyRow = "e"

func init() {
	registerListSortBy()
	registerListZip()
	registerListFlatMap()
}

// ============================================================================
// _list_sortBy: stable sort with an AILANG comparator
// ============================================================================

func registerListSortBy() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_sortBy",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListSortByType,
		Impl:    listSortByImpl,
		Metadata: &BuiltinMetadata{
			Description: "Stable sort with a comparison function (iterative)",
			LongDesc:    "Sorts by cmp(a, b) < 0 meaning a precedes b. Stable: elements that compare equal keep their input order, matching the merge sort std/list.sortBy used to be. O(n log n) comparisons, constant AILANG stack.",
			Params: []ParamDoc{
				{Name: "cmp", Description: "Comparison function returning negative, zero or positive"},
				{Name: "xs", Description: "Input list"},
			},
			Returns:   "A new sorted list",
			Since:     "v0.44.2",
			Stability: StabilityStable,
			Tags:      []string{"list", "sort", "iterative", "performance"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_sortBy: %v", err))
	}
}

// Type: forall a. ((a, a) -> int, list[a]) -> list[a]
func makeListSortByType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	cmp := T.Func(a, a).Returns(T.Int()).RowTail(effectPolyRow).Build()
	return T.Func(cmp, listA).Returns(listA).RowTail(effectPolyRow).Build()
}

func listSortByImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	cmp := args[0]
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_sortBy: expected List for second argument, got %T", args[1])
	}
	if ctx == nil || ctx.FnCallerN == nil {
		return nil, fmt.Errorf("_list_sortBy: FnCallerN not set (evaluator not wired)")
	}

	result := make([]eval.Value, len(list.Elements))
	copy(result, list.Elements)

	// sort.SliceStable cannot return an error, so the first comparator failure
	// is recorded and every later comparison short-circuits.
	var cmpErr error
	sort.SliceStable(result, func(i, j int) bool {
		if cmpErr != nil {
			return false
		}
		v, err := ctx.FnCallerN(cmp, []eval.Value{result[i], result[j]})
		if err != nil {
			cmpErr = callbackErr(err, "_list_sortBy: comparator error")
			return false
		}
		iv, ok := v.(*eval.IntValue)
		if !ok {
			cmpErr = fmt.Errorf("_list_sortBy: comparator must return int, got %T", v)
			return false
		}
		return iv.Value < 0
	})
	if cmpErr != nil {
		return nil, cmpErr
	}
	return &eval.ListValue{Elements: result}, nil
}

// ============================================================================
// _list_zip: pair elements, truncating to the shorter list
// ============================================================================

func registerListZip() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_zip",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListZipType,
		Impl:    listZipImpl,
		Metadata: &BuiltinMetadata{
			Description: "Pair up elements of two lists (iterative)",
			LongDesc:    "Returns [(x0, y0), (x1, y1), ...], truncated to the shorter list.",
			Params: []ParamDoc{
				{Name: "xs", Description: "First list"},
				{Name: "ys", Description: "Second list"},
			},
			Returns:   "List of pairs",
			Since:     "v0.44.2",
			Stability: StabilityStable,
			Tags:      []string{"list", "zip", "iterative", "performance"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_zip: %v", err))
	}
}

// Type: forall a b. (list[a], list[b]) -> list[(a, b)]
func makeListZipType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	b := T.Var("b")
	pair := &types.TTuple{Elements: []types.Type{a, b}}
	return T.Func(T.List(a), T.List(b)).Returns(T.List(pair)).Build()
}

func listZipImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	xs, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_zip: expected List for first argument, got %T", args[0])
	}
	ys, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_zip: expected List for second argument, got %T", args[1])
	}
	n := min(len(xs.Elements), len(ys.Elements))
	result := make([]eval.Value, n)
	for i := 0; i < n; i++ {
		result[i] = &eval.TupleValue{Elements: []eval.Value{xs.Elements[i], ys.Elements[i]}}
	}
	return &eval.ListValue{Elements: result}, nil
}

// ============================================================================
// _list_flatMap: map then concatenate, in one pass
// ============================================================================

func registerListFlatMap() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_flatMap",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListFlatMapType,
		Impl:    listFlatMapImpl,
		Metadata: &BuiltinMetadata{
			Description: "Map each element to a list and concatenate the results (iterative)",
			Params: []ParamDoc{
				{Name: "f", Description: "Function returning a list per element"},
				{Name: "xs", Description: "Input list"},
			},
			Returns:   "Concatenation of f(x) for every x, in order",
			Since:     "v0.44.2",
			Stability: StabilityStable,
			Tags:      []string{"list", "flatMap", "iterative", "performance"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_flatMap: %v", err))
	}
}

// Type: forall a b. (a -> list[b], list[a]) -> list[b]
func makeListFlatMapType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	b := T.Var("b")
	fn := T.Func(a).Returns(T.List(b)).RowTail(effectPolyRow).Build()
	return T.Func(fn, T.List(a)).Returns(T.List(b)).RowTail(effectPolyRow).Build()
}

func listFlatMapImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	fn := args[0]
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_flatMap: expected List for second argument, got %T", args[1])
	}
	if ctx == nil || ctx.FnCaller == nil {
		return nil, fmt.Errorf("_list_flatMap: FnCaller not set (evaluator not wired)")
	}
	result := make([]eval.Value, 0, len(list.Elements))
	for i, elem := range list.Elements {
		v, err := ctx.FnCaller(fn, elem)
		if err != nil {
			return nil, callbackErr(err, "_list_flatMap: callback error at index %d", i)
		}
		inner, ok := v.(*eval.ListValue)
		if !ok {
			return nil, fmt.Errorf("_list_flatMap: f must return a List, got %T at index %d", v, i)
		}
		result = append(result, inner.Elements...)
	}
	return &eval.ListValue{Elements: result}, nil
}
