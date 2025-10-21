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
	registerListConcat()
}

// ============================================================================
// List Operations
// ============================================================================

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
