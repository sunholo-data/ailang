package builtins

import (
	"fmt"

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
