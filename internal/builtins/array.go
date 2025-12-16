package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Array builtin functions for AILANG
// These provide O(1) indexed access operations on arrays

func init() {
	registerArrayMake()
	registerArrayGet()
	registerArrayUnsafeGet()
	registerArraySet()
	registerArrayLength()
	registerArrayFromList()
	registerArrayToList()
}

// ============================================================================
// Array Operations
// ============================================================================

// registerArrayMake registers the array_make builtin
// Creates an array of given size with all elements set to default value
func registerArrayMake() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/array",
		Name:    "_array_make",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeArrayMakeType,
		Impl:    arrayMakeImpl,
		Metadata: &BuiltinMetadata{
			Description: "Create an array of given size with default value",
			LongDesc:    "Creates a new array with the specified size where all elements are initialized to the given default value.",
			Params: []ParamDoc{
				{Name: "size", Description: "Number of elements in the array"},
				{Name: "default", Description: "Default value for all elements"},
			},
			Returns:   "New array with size elements, all set to default",
			Since:     "v0.5.0",
			Stability: StabilityExperimental,
			Tags:      []string{"array", "create", "allocate"},
			Category:  "array",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register array_make: %v", err))
	}
}

// Type: forall a. (int, a) -> Array[a]
func makeArrayMakeType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	arrayA := &types.TArray{Element: a}
	return T.Func(T.Int(), a).Returns(arrayA).Build()
}

func arrayMakeImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	sizeVal, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("array_make: expected int for size, got %T", args[0])
	}
	size := sizeVal.Value
	if size < 0 {
		return nil, fmt.Errorf("array_make: size cannot be negative, got %d", size)
	}

	defaultVal := args[1]
	elements := make([]eval.Value, size)
	for i := range elements {
		elements[i] = defaultVal
	}

	return &eval.ArrayValue{Elements: elements}, nil
}

// registerArrayGet registers the array_get builtin
// Returns Option[a] - Some(element) if index valid, None if out of bounds
func registerArrayGet() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/array",
		Name:    "_array_get",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeArrayGetType,
		Impl:    arrayGetImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get element at index (safe, returns Option)",
			LongDesc:    "Returns Some(element) if the index is valid, None if out of bounds. O(1) access time.",
			Params: []ParamDoc{
				{Name: "arr", Description: "Array to access"},
				{Name: "idx", Description: "Index of element to get"},
			},
			Returns:   "Some(element) if valid index, None otherwise",
			Since:     "v0.5.0",
			Stability: StabilityExperimental,
			Tags:      []string{"array", "access", "index", "safe"},
			Category:  "array",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register array_get: %v", err))
	}
}

// Type: forall a. (Array[a], int) -> Option[a]
// Note: Returns the element directly for now (Option not yet implemented)
func makeArrayGetType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	arrayA := &types.TArray{Element: a}
	// TODO: Return Option[a] when Option is available
	// For now, return a directly (unsafe version behavior)
	return T.Func(arrayA, T.Int()).Returns(a).Build()
}

func arrayGetImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	arr, ok := args[0].(*eval.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("array_get: expected Array, got %T", args[0])
	}

	idxVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("array_get: expected int for index, got %T", args[1])
	}

	elem, found := arr.Get(int64(idxVal.Value))
	if !found {
		// TODO: Return None when Option is available
		return nil, fmt.Errorf("array_get: index %d out of bounds (array length: %d)", idxVal.Value, len(arr.Elements))
	}
	return elem, nil
}

// registerArrayUnsafeGet registers the array_unsafe_get builtin
// Returns element directly, panics if out of bounds
func registerArrayUnsafeGet() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/array",
		Name:    "_array_unsafe_get",
		NumArgs: 2,
		IsPure:  true,
		Effect:  "",
		Type:    makeArrayUnsafeGetType,
		Impl:    arrayUnsafeGetImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get element at index (unsafe, panics if out of bounds)",
			LongDesc:    "Returns the element at the given index. Panics if the index is out of bounds. Use array_get for safe access with Option return type.",
			Params: []ParamDoc{
				{Name: "arr", Description: "Array to access"},
				{Name: "idx", Description: "Index of element to get"},
			},
			Returns:   "Element at index (panics if invalid)",
			Since:     "v0.5.0",
			Stability: StabilityExperimental,
			Tags:      []string{"array", "access", "index", "unsafe"},
			Category:  "array",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register array_unsafe_get: %v", err))
	}
}

// Type: forall a. (Array[a], int) -> a
func makeArrayUnsafeGetType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	arrayA := &types.TArray{Element: a}
	return T.Func(arrayA, T.Int()).Returns(a).Build()
}

func arrayUnsafeGetImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	arr, ok := args[0].(*eval.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("array_unsafe_get: expected Array, got %T", args[0])
	}

	idxVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("array_unsafe_get: expected int for index, got %T", args[1])
	}

	elem, found := arr.Get(int64(idxVal.Value))
	if !found {
		panic(fmt.Sprintf("array_unsafe_get: index %d out of bounds (array length: %d)", idxVal.Value, len(arr.Elements)))
	}
	return elem, nil
}

// registerArraySet registers the array_set builtin
// Returns new array with element at index replaced
func registerArraySet() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/array",
		Name:    "_array_set",
		NumArgs: 3,
		IsPure:  true,
		Effect:  "",
		Type:    makeArraySetType,
		Impl:    arraySetImpl,
		Metadata: &BuiltinMetadata{
			Description: "Set element at index (returns new array)",
			LongDesc:    "Returns a new array with the element at the given index replaced. The original array is not modified. Returns the original array unchanged if index is out of bounds.",
			Params: []ParamDoc{
				{Name: "arr", Description: "Array to update"},
				{Name: "idx", Description: "Index of element to set"},
				{Name: "val", Description: "New value for the element"},
			},
			Returns:   "New array with element at index replaced",
			Since:     "v0.5.0",
			Stability: StabilityExperimental,
			Tags:      []string{"array", "update", "set", "immutable"},
			Category:  "array",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register array_set: %v", err))
	}
}

// Type: forall a. (Array[a], int, a) -> Array[a]
func makeArraySetType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	arrayA := &types.TArray{Element: a}
	return T.Func(arrayA, T.Int(), a).Returns(arrayA).Build()
}

func arraySetImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	arr, ok := args[0].(*eval.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("array_set: expected Array, got %T", args[0])
	}

	idxVal, ok := args[1].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("array_set: expected int for index, got %T", args[1])
	}

	newVal := args[2]
	return arr.Set(int64(idxVal.Value), newVal), nil
}

// registerArrayLength registers the array_length builtin
func registerArrayLength() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/array",
		Name:    "_array_length",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeArrayLengthType,
		Impl:    arrayLengthImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get the length of an array",
			LongDesc:    "Returns the number of elements in the array. O(1) operation.",
			Params: []ParamDoc{
				{Name: "arr", Description: "Array to get length of"},
			},
			Returns:   "Number of elements in the array",
			Since:     "v0.5.0",
			Stability: StabilityExperimental,
			Tags:      []string{"array", "length", "size"},
			Category:  "array",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register array_length: %v", err))
	}
}

// Type: forall a. Array[a] -> int
func makeArrayLengthType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	arrayA := &types.TArray{Element: a}
	return T.Func(arrayA).Returns(T.Int()).Build()
}

func arrayLengthImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	arr, ok := args[0].(*eval.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("array_length: expected Array, got %T", args[0])
	}
	return &eval.IntValue{Value: len(arr.Elements)}, nil
}

// registerArrayFromList registers the array_from_list builtin
func registerArrayFromList() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/array",
		Name:    "_array_from_list",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeArrayFromListType,
		Impl:    arrayFromListImpl,
		Metadata: &BuiltinMetadata{
			Description: "Convert a list to an array",
			LongDesc:    "Creates a new array containing all elements from the input list. O(n) operation.",
			Params: []ParamDoc{
				{Name: "xs", Description: "List to convert"},
			},
			Returns:   "New array with same elements as the list",
			Since:     "v0.5.0",
			Stability: StabilityExperimental,
			Tags:      []string{"array", "list", "convert"},
			Category:  "array",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register array_from_list: %v", err))
	}
}

// Type: forall a. [a] -> Array[a]
func makeArrayFromListType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	listA := T.List(a) // DX-17: Use T.List() for lowercase "list" constructor
	arrayA := &types.TArray{Element: a}
	return T.Func(listA).Returns(arrayA).Build()
}

func arrayFromListImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	list, ok := args[0].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("array_from_list: expected List, got %T", args[0])
	}

	elements := make([]eval.Value, len(list.Elements))
	copy(elements, list.Elements)
	return &eval.ArrayValue{Elements: elements}, nil
}

// registerArrayToList registers the array_to_list builtin
func registerArrayToList() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/array",
		Name:    "_array_to_list",
		NumArgs: 1,
		IsPure:  true,
		Effect:  "",
		Type:    makeArrayToListType,
		Impl:    arrayToListImpl,
		Metadata: &BuiltinMetadata{
			Description: "Convert an array to a list",
			LongDesc:    "Creates a new list containing all elements from the input array. O(n) operation.",
			Params: []ParamDoc{
				{Name: "arr", Description: "Array to convert"},
			},
			Returns:   "New list with same elements as the array",
			Since:     "v0.5.0",
			Stability: StabilityExperimental,
			Tags:      []string{"array", "list", "convert"},
			Category:  "array",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register array_to_list: %v", err))
	}
}

// Type: forall a. Array[a] -> [a]
func makeArrayToListType() types.Type {
	T := types.NewBuilder()
	a := T.Var("a")
	arrayA := &types.TArray{Element: a}
	listA := T.List(a) // DX-17: Use T.List() for lowercase "list" constructor
	return T.Func(arrayA).Returns(listA).Build()
}

func arrayToListImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	arr, ok := args[0].(*eval.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("array_to_list: expected Array, got %T", args[0])
	}

	elements := make([]eval.Value, len(arr.Elements))
	copy(elements, arr.Elements)
	return &eval.ListValue{Elements: elements}, nil
}
