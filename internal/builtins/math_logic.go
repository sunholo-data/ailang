package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Logic builtins (and, or, not)

func registerLogic() {
	registerLogicOpWithMeta("and_Bool", func(a, b bool) bool { return a && b },
		"Logical AND operation", []string{"logic", "boolean", "and", "conjunction"})
	registerLogicOpWithMeta("or_Bool", func(a, b bool) bool { return a || b },
		"Logical OR operation", []string{"logic", "boolean", "or", "disjunction"})
	registerLogicUnaryWithMeta("not_Bool", func(a bool) bool { return !a },
		"Logical NOT operation (negation)", []string{"logic", "boolean", "not", "negation"})
}

// registerLogicOpWithMeta registers a binary logic operation with metadata
func registerLogicOpWithMeta(name string, fn func(bool, bool) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a, ok := args[0].(*eval.BoolValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected BoolValue for arg 0, got %T", name, args[0])
		}
		b, ok := args[1].(*eval.BoolValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected BoolValue for arg 1, got %T", name, args[1])
		}
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
			Returns:   "Result of logical operation",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "logic",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}

// registerLogicUnaryWithMeta registers a unary logic operation with metadata
func registerLogicUnaryWithMeta(name string, fn func(bool) bool, description string, tags []string) {
	impl := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a, ok := args[0].(*eval.BoolValue)
		if !ok {
			return nil, fmt.Errorf("%s: expected BoolValue for arg 0, got %T", name, args[0])
		}
		return &eval.BoolValue{Value: fn(a.Value)}, nil
	}
	typeFunc := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bool()).Returns(T.Bool()).Build()
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/prelude",
		Name:    name,
		NumArgs: 1,
		IsPure:  true,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params: []ParamDoc{
				{Name: "a", Description: "Boolean value"},
			},
			Returns:   "Negated boolean value",
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      tags,
			Category:  "logic",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}
