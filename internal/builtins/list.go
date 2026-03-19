package builtins

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// List builtin functions for AILANG
// These provide operations on lists

func init() {
	registerListCons()
	registerListConcat()
	registerListLength()
	registerListHead()
	registerListNth()
	registerListContains()
	registerListExtract()
	registerListReverse()
	registerListTake()
	registerListDrop()
	// Set operations (member, dedup, intersect, union, difference) registered in list_set.go
}

// ============================================================================
// List Operations
// ============================================================================

// registerListCons registers the :: (cons) builtin
// This implements the S-CONS sugar (v0.4.1): x :: xs desugars to ::(x, xs)
func registerListCons() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/list",
		Name:    "::",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "", // Pure function
		Type:    makeListConsType,
		Impl:    listConsImpl,

		Metadata: &BuiltinMetadata{
			Description: "Prepend an element to a list (cons operator)",
			LongDesc:    "Constructs a new list with the given element prepended to the existing list. This is the fundamental list construction operation. Does not modify the input list.",
			Params: []ParamDoc{
				{Name: "head", Description: "Element to prepend"},
				{Name: "tail", Description: "List to prepend to"},
			},
			Returns: "New list with head prepended to tail",
			Examples: []Example{
				{Code: `1 :: [2, 3]`, Description: "Returns [1, 2, 3]"},
				{Code: `"a" :: ["b", "c"]`, Description: "Returns [\"a\", \"b\", \"c\"]"},
				{Code: `1 :: []`, Description: "Returns [1]"},
				{Code: `1 :: 2 :: []`, Description: "Returns [1, 2] (right-associative)"},
			},
			SeeAlso:   []string{"concat_List"},
			Since:     "v0.4.1",
			Stability: StabilityStable,
			Tags:      []string{"list", "cons", "prepend", "construct"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register :: cons: %v", err))
	}
}

// makeListConsType builds the type signature for :: (cons)
// Type: forall a. (a, list[a]) -> list[a]
func makeListConsType() types.Type {
	T := types.NewBuilder()
	// Create a type variable 'a' for the element type
	a := T.Var("a")
	listA := T.List(a) // DX-17: Use T.List() for lowercase "list" constructor
	return T.Func(a, listA).Returns(listA).Build()
}

// listConsImpl is the implementation for :: (cons)
// Prepends an element to a list by creating a new list.
//
// IMPORTANT: This function does NOT mutate the input list. It creates a new
// list with the element prepended. The input list remains unchanged,
// preserving referential transparency and enabling safe reuse.
func listConsImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Extract arguments: head element and tail list
	head := args[0]

	tail, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf(":: expected List for second argument, got %T", args[1])
	}

	// Create result list with pre-allocated capacity to avoid extra allocations
	// Capacity = 1 + len(tail) ensures single allocation
	result := make([]eval.Value, 0, 1+len(tail.Elements))

	// Prepend head, then append tail elements (shallow copy of element references)
	// Input list is NOT modified
	result = append(result, head)
	result = append(result, tail.Elements...)

	return &eval.ListValue{Elements: result}, nil
}

// registerListConcat registers the concat_List builtin
// This implements the ++ operator for lists
func registerListConcat() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/list",
		Name:    "concat_List",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "", // Pure function
		Type:    makeListConcatType,
		Impl:    listConcatImpl,

		Metadata: &BuiltinMetadata{
			Description: "Concatenate two lists",
			LongDesc:    "Returns a new list containing all elements from the first list followed by all elements from the second list. Does not modify the input lists.",
			Params: []ParamDoc{
				{Name: "a", Description: "First list"},
				{Name: "b", Description: "Second list"},
			},
			Returns: "New list containing elements from both input lists",
			Examples: []Example{
				{Code: `[1, 2, 3] ++ [4, 5]`, Description: "Returns [1, 2, 3, 4, 5]"},
				{Code: `[] ++ [1, 2]`, Description: "Returns [1, 2]"},
				{Code: `[[1], [2]] ++ [[3]]`, Description: "Returns [[1], [2], [3]]"},
			},
			SeeAlso:   []string{"concat_String"},
			Since:     "v0.3.16",
			Stability: StabilityStable,
			Tags:      []string{"list", "concat", "append", "merge"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register concat_List: %v", err))
	}
}

// makeListConcatType builds the type signature for concat_List
// Type: forall a. (List[a], List[a]) -> List[a]
func makeListConcatType() types.Type {
	T := types.NewBuilder()
	// Create a type variable 'a' for the list element type
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(listA, listA).Returns(listA).Build()
}

// listConcatImpl is the implementation for concat_List
// Concatenates two lists by creating a new list with all elements.
//
// IMPORTANT: This function does NOT mutate the input lists. It creates a new
// list with copies of the element references. The input lists remain unchanged,
// preserving referential transparency and enabling safe reuse.
func listConcatImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Extract list arguments
	list1, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("concat_List: expected List for first argument, got %T", args[0])
	}

	list2, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("concat_List: expected List for second argument, got %T", args[1])
	}

	// Create result list with pre-allocated capacity to avoid extra allocations
	// Capacity = len(list1) + len(list2) ensures single allocation
	result := make([]eval.Value, 0, len(list1.Elements)+len(list2.Elements))

	// Append all elements from both lists (shallow copy of element references)
	// Input lists are NOT modified
	result = append(result, list1.Elements...)
	result = append(result, list2.Elements...)

	return &eval.ListValue{Elements: result}, nil
}

// ============================================================================
// SMT-Verifiable List Builtins (M-SMT-LISTS)
// ============================================================================
// These builtins mirror std/list functions but are implemented in Go,
// making them usable in contracts and encodable to Z3 seq.* operations.

// registerListLength registers the _list_length builtin
func registerListLength() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_length",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeListLengthType,
		Impl:    listLengthImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get the length of a list",
			Params:      []ParamDoc{{Name: "xs", Description: "The list"}},
			Returns:     "Number of elements in the list",
			Since:       "v0.7.4",
			Stability:   StabilityStable,
			Tags:        []string{"list", "length", "smt"},
			Category:    "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_length: %v", err))
	}
}

func makeListLengthType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(listA).Returns(&types.TCon{Name: "int"}).Build()
}

func listLengthImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_length: expected List, got %T", args[0])
	}
	return &eval.IntValue{Value: len(list.Elements)}, nil
}

// registerListHead registers the _list_head builtin
func registerListHead() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_head",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeListHeadType,
		Impl:    listHeadImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get the first element of a list",
			Params:      []ParamDoc{{Name: "xs", Description: "The list (must be non-empty)"}},
			Returns:     "First element",
			Since:       "v0.7.4",
			Stability:   StabilityStable,
			Tags:        []string{"list", "head", "smt"},
			Category:    "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_head: %v", err))
	}
}

func makeListHeadType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(listA).Returns(a).Build()
}

func listHeadImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_head: expected List, got %T", args[0])
	}
	if len(list.Elements) == 0 {
		return nil, fmt.Errorf("_list_head: empty list")
	}
	return list.Elements[0], nil
}

// registerListNth registers the _list_nth builtin
func registerListNth() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_nth",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListNthType,
		Impl:    listNthImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get the element at a given index",
			Params: []ParamDoc{
				{Name: "xs", Description: "The list"},
				{Name: "idx", Description: "Zero-based index"},
			},
			Returns:   "Element at the given index",
			Since:     "v0.7.4",
			Stability: StabilityStable,
			Tags:      []string{"list", "nth", "index", "smt"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_nth: %v", err))
	}
}

func makeListNthType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(listA, &types.TCon{Name: "int"}).Returns(a).Build()
}

func listNthImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_nth: expected List, got %T", args[0])
	}
	idxVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_list_nth: expected Int for index, got %T", args[1])
	}
	idx := idxVal.Value
	if idx < 0 || idx >= len(list.Elements) {
		return nil, fmt.Errorf("_list_nth: index %d out of bounds for list of length %d", idx, len(list.Elements))
	}
	return list.Elements[idx], nil
}

// ============================================================================
// Additional SMT-Verifiable List Builtins (M3_RECURSIVE_LIST_OPS)
// ============================================================================

// registerListContains registers the _list_contains builtin
func registerListContains() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_contains",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListContainsType,
		Impl:    listContainsImpl,
		Metadata: &BuiltinMetadata{
			Description: "Check if a list contains a given element",
			Params: []ParamDoc{
				{Name: "xs", Description: "The list to search"},
				{Name: "elem", Description: "The element to find"},
			},
			Returns:   "true if the element is in the list, false otherwise",
			Since:     "v0.9.0",
			Stability: StabilityStable,
			Tags:      []string{"list", "contains", "search", "smt"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_contains: %v", err))
	}
}

func makeListContainsType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(listA, a).Returns(&types.TCon{Name: "bool"}).Build()
}

func listContainsImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_contains: expected List, got %T", args[0])
	}
	elem := args[1]
	for _, v := range list.Elements {
		if valuesEqual(v, elem) {
			return &eval.BoolValue{Value: true}, nil
		}
	}
	return &eval.BoolValue{Value: false}, nil
}

// valuesEqual compares two eval.Value instances for structural equality.
// M-HASH-COLLECTIONS Phase 1: Replaced reflect.DeepEqual fallback with
// recursive structural comparison for all value types.
func valuesEqual(left, right eval.Value) bool {
	if left == right {
		return true
	}
	switch l := left.(type) {
	case *eval.IntValue:
		r, ok := right.(*eval.IntValue)
		return ok && l.Value == r.Value
	case *eval.FloatValue:
		r, ok := right.(*eval.FloatValue)
		return ok && l.Value == r.Value
	case *eval.StringValue:
		r, ok := right.(*eval.StringValue)
		return ok && l.Value == r.Value
	case *eval.BoolValue:
		r, ok := right.(*eval.BoolValue)
		return ok && l.Value == r.Value
	case *eval.UnitValue:
		_, ok := right.(*eval.UnitValue)
		return ok
	case *eval.ListValue:
		r, ok := right.(*eval.ListValue)
		if !ok || len(l.Elements) != len(r.Elements) {
			return false
		}
		for i := range l.Elements {
			if !valuesEqual(l.Elements[i], r.Elements[i]) {
				return false
			}
		}
		return true
	case *eval.ArrayValue:
		r, ok := right.(*eval.ArrayValue)
		if !ok || len(l.Elements) != len(r.Elements) {
			return false
		}
		for i := range l.Elements {
			if !valuesEqual(l.Elements[i], r.Elements[i]) {
				return false
			}
		}
		return true
	case *eval.TupleValue:
		r, ok := right.(*eval.TupleValue)
		if !ok || len(l.Elements) != len(r.Elements) {
			return false
		}
		for i := range l.Elements {
			if !valuesEqual(l.Elements[i], r.Elements[i]) {
				return false
			}
		}
		return true
	case *eval.RecordValue:
		r, ok := right.(*eval.RecordValue)
		if !ok || len(l.Fields) != len(r.Fields) {
			return false
		}
		for k, lv := range l.Fields {
			rv, exists := r.Fields[k]
			if !exists || !valuesEqual(lv, rv) {
				return false
			}
		}
		return true
	case *eval.TaggedValue:
		r, ok := right.(*eval.TaggedValue)
		if !ok || l.CtorName != r.CtorName || len(l.Fields) != len(r.Fields) {
			return false
		}
		for i := range l.Fields {
			if !valuesEqual(l.Fields[i], r.Fields[i]) {
				return false
			}
		}
		return true
	case *eval.BytesValue:
		r, ok := right.(*eval.BytesValue)
		if !ok || len(l.Value) != len(r.Value) {
			return false
		}
		for i := range l.Value {
			if l.Value[i] != r.Value[i] {
				return false
			}
		}
		return true
	}
	return false
}

// registerListExtract registers the _list_extract builtin
func registerListExtract() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_extract",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeListExtractType,
		Impl:    listExtractImpl,
		Metadata: &BuiltinMetadata{
			Description: "Extract a subsequence from a list",
			Params: []ParamDoc{
				{Name: "xs", Description: "The list"},
				{Name: "offset", Description: "Starting index (0-based)"},
				{Name: "length", Description: "Number of elements to extract"},
			},
			Returns:   "Subsequence of the list",
			Since:     "v0.9.0",
			Stability: StabilityStable,
			Tags:      []string{"list", "extract", "slice", "smt"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_extract: %v", err))
	}
}

func makeListExtractType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(listA, &types.TCon{Name: "int"}, &types.TCon{Name: "int"}).Returns(listA).Build()
}

func listExtractImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_extract: expected List, got %T", args[0])
	}
	offsetVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_list_extract: expected Int for offset, got %T", args[1])
	}
	lengthVal, ok := args[2].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_list_extract: expected Int for length, got %T", args[2])
	}

	offset := offsetVal.Value
	length := lengthVal.Value
	n := len(list.Elements)

	// Clamp to valid range (matching Z3 seq.extract semantics)
	if offset < 0 {
		offset = 0
	}
	if offset > n {
		offset = n
	}
	end := offset + length
	if end > n {
		end = n
	}
	if length < 0 || offset >= n {
		return &eval.ListValue{Elements: []eval.Value{}}, nil
	}

	result := make([]eval.Value, end-offset)
	copy(result, list.Elements[offset:end])
	return &eval.ListValue{Elements: result}, nil
}

// registerListReverse registers the _list_reverse builtin
func registerListReverse() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_reverse",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeListReverseType,
		Impl:    listReverseImpl,
		Metadata: &BuiltinMetadata{
			Description: "Reverse a list",
			Params:      []ParamDoc{{Name: "xs", Description: "The list to reverse"}},
			Returns:     "A new list with elements in reverse order",
			Since:       "v0.9.0",
			Stability:   StabilityStable,
			Tags:        []string{"list", "reverse", "smt"},
			Category:    "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_reverse: %v", err))
	}
}

func makeListReverseType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(listA).Returns(listA).Build()
}

func listReverseImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_reverse: expected List, got %T", args[0])
	}
	n := len(list.Elements)
	result := make([]eval.Value, n)
	for i, v := range list.Elements {
		result[n-1-i] = v
	}
	return &eval.ListValue{Elements: result}, nil
}

// registerListTake registers the _list_take builtin
func registerListTake() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_take",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListTakeType,
		Impl:    listTakeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Take the first n elements from a list",
			Params: []ParamDoc{
				{Name: "n", Description: "Number of elements to take"},
				{Name: "xs", Description: "The list"},
			},
			Returns:   "A new list containing the first n elements",
			Since:     "v0.9.0",
			Stability: StabilityStable,
			Tags:      []string{"list", "take", "slice", "smt"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_take: %v", err))
	}
}

func makeListTakeType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(&types.TCon{Name: "int"}, listA).Returns(listA).Build()
}

func listTakeImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_list_take: expected Int for n, got %T", args[0])
	}
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_take: expected List, got %T", args[1])
	}

	n := nVal.Value
	if n <= 0 {
		return &eval.ListValue{Elements: []eval.Value{}}, nil
	}

	// M-EVAL-BOUNDED-PIPELINE: warn when take discards a large portion of the list.
	// This suggests the list was fully materialized before take could cap it —
	// takeFlatMap or takeMap would avoid the intermediate allocation.
	if len(list.Elements) > 10000 && n < len(list.Elements)/2 {
		fmt.Fprintf(os.Stderr, "note: take(%d) on a %d-element list — %d elements were materialized then discarded.\n"+
			"      Consider takeFlatMap(n, f, xs) or takeMap(n, f, xs) to avoid intermediate allocation.\n",
			n, len(list.Elements), len(list.Elements)-n)
	}

	if n > len(list.Elements) {
		n = len(list.Elements)
	}

	result := make([]eval.Value, n)
	copy(result, list.Elements[:n])
	return &eval.ListValue{Elements: result}, nil
}

// registerListDrop registers the _list_drop builtin
func registerListDrop() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "$builtin",
		Name:    "_list_drop",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeListDropType,
		Impl:    listDropImpl,
		Metadata: &BuiltinMetadata{
			Description: "Drop the first n elements from a list",
			Params: []ParamDoc{
				{Name: "n", Description: "Number of elements to drop"},
				{Name: "xs", Description: "The list"},
			},
			Returns:   "A new list with the first n elements removed",
			Since:     "v0.9.0",
			Stability: StabilityStable,
			Tags:      []string{"list", "drop", "slice", "smt"},
			Category:  "list",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _list_drop: %v", err))
	}
}

func makeListDropType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a)
	return T.Func(&types.TCon{Name: "int"}, listA).Returns(listA).Build()
}

func listDropImpl(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
	nVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("_list_drop: expected Int for n, got %T", args[0])
	}
	list, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("_list_drop: expected List, got %T", args[1])
	}

	n := nVal.Value
	if n <= 0 {
		result := make([]eval.Value, len(list.Elements))
		copy(result, list.Elements)
		return &eval.ListValue{Elements: result}, nil
	}
	if n >= len(list.Elements) {
		return &eval.ListValue{Elements: []eval.Value{}}, nil
	}

	result := make([]eval.Value, len(list.Elements)-n)
	copy(result, list.Elements[n:])
	return &eval.ListValue{Elements: result}, nil
}
