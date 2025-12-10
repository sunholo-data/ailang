package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Comparison builtins (eq, ne, lt, le, gt, ge for Int, Float, String, Bool)

func registerComparisons() {
	// Int comparisons
	registerCmpWithMeta("eq_Int", func(a, b int) bool { return a == b },
		"Test if two integers are equal", []string{"comparison", "equality", "int"})
	registerCmpWithMeta("ne_Int", func(a, b int) bool { return a != b },
		"Test if two integers are not equal", []string{"comparison", "inequality", "int"})
	registerCmpWithMeta("lt_Int", func(a, b int) bool { return a < b },
		"Test if first integer is less than second", []string{"comparison", "ordering", "int"})
	registerCmpWithMeta("le_Int", func(a, b int) bool { return a <= b },
		"Test if first integer is less than or equal to second", []string{"comparison", "ordering", "int"})
	registerCmpWithMeta("gt_Int", func(a, b int) bool { return a > b },
		"Test if first integer is greater than second", []string{"comparison", "ordering", "int"})
	registerCmpWithMeta("ge_Int", func(a, b int) bool { return a >= b },
		"Test if first integer is greater than or equal to second", []string{"comparison", "ordering", "int"})

	// Float comparisons
	registerCmpFloatWithMeta("eq_Float", func(a, b float64) bool { return a == b },
		"Test if two floats are equal (IEEE 754 equality)", []string{"comparison", "equality", "float"})
	registerCmpFloatWithMeta("ne_Float", func(a, b float64) bool { return a != b },
		"Test if two floats are not equal (IEEE 754 equality)", []string{"comparison", "inequality", "float"})
	registerCmpFloatWithMeta("lt_Float", func(a, b float64) bool { return a < b },
		"Test if first float is less than second", []string{"comparison", "ordering", "float"})
	registerCmpFloatWithMeta("le_Float", func(a, b float64) bool { return a <= b },
		"Test if first float is less than or equal to second", []string{"comparison", "ordering", "float"})
	registerCmpFloatWithMeta("gt_Float", func(a, b float64) bool { return a > b },
		"Test if first float is greater than second", []string{"comparison", "ordering", "float"})
	registerCmpFloatWithMeta("ge_Float", func(a, b float64) bool { return a >= b },
		"Test if first float is greater than or equal to second", []string{"comparison", "ordering", "float"})

	// String comparisons
	registerCmpStringWithMeta("eq_String", func(a, b string) bool { return a == b },
		"Test if two strings are equal", []string{"comparison", "equality", "string"})
	registerCmpStringWithMeta("ne_String", func(a, b string) bool { return a != b },
		"Test if two strings are not equal", []string{"comparison", "inequality", "string"})
	registerCmpStringWithMeta("lt_String", func(a, b string) bool { return a < b },
		"Test if first string is lexicographically less than second", []string{"comparison", "ordering", "string", "lexicographic"})
	registerCmpStringWithMeta("le_String", func(a, b string) bool { return a <= b },
		"Test if first string is lexicographically less than or equal to second", []string{"comparison", "ordering", "string", "lexicographic"})
	registerCmpStringWithMeta("gt_String", func(a, b string) bool { return a > b },
		"Test if first string is lexicographically greater than second", []string{"comparison", "ordering", "string", "lexicographic"})
	registerCmpStringWithMeta("ge_String", func(a, b string) bool { return a >= b },
		"Test if first string is lexicographically greater than or equal to second", []string{"comparison", "ordering", "string", "lexicographic"})

	// Bool comparisons
	registerCmpBoolWithMeta("eq_Bool", func(a, b bool) bool { return a == b },
		"Test if two booleans are equal", []string{"comparison", "equality", "bool"})
	registerCmpBoolWithMeta("ne_Bool", func(a, b bool) bool { return a != b },
		"Test if two booleans are not equal", []string{"comparison", "inequality", "bool"})
}

// registerCmpWithMeta registers an Int comparison with metadata
func registerCmpWithMeta(name string, fn func(int, int) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int(), T.Int()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First integer"},
				{Name: "b", Description: "Second integer"},
			},
			Returns:   "true if comparison holds, false otherwise",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "comparison",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerCmpFloatWithMeta registers a Float comparison with metadata
func registerCmpFloatWithMeta(name string, fn func(float64, float64) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		b := args[1].(*eval.FloatValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Float(), T.Float()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First float"},
				{Name: "b", Description: "Second float"},
			},
			Returns:   "true if comparison holds, false otherwise",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "comparison",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerCmpStringWithMeta registers a String comparison with metadata
func registerCmpStringWithMeta(name string, fn func(string, string) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.StringValue)
		b := args[1].(*eval.StringValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String(), T.String()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First string"},
				{Name: "b", Description: "Second string"},
			},
			Returns:   "true if comparison holds, false otherwise",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "comparison",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerCmpBoolWithMeta registers a Bool comparison with metadata
func registerCmpBoolWithMeta(name string, fn func(bool, bool) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.BoolValue)
		b := args[1].(*eval.BoolValue)
		return &eval.BoolValue{Value: fn(a.Value, b.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bool(), T.Bool()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 2,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "First boolean"},
				{Name: "b", Description: "Second boolean"},
			},
			Returns:   "true if comparison holds, false otherwise",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "comparison",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}
