package builtins

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// Math, comparison, logic, and conversion builtins for AILANG
// These provide the fundamental numeric and boolean operations
//
// This file contains shared initialization and helper functions.
// Individual categories are in separate files:
//   - math_arithmetic.go - Int/Float arithmetic operations
//   - math_comparison.go - Comparison operations for all types
//   - math_logic.go      - Boolean logic operations
//   - math_conversion.go - Type conversion operations
//   - math_trig.go       - Trigonometry and advanced math

func init() {
	registerArithmetic()
	registerComparisons()
	registerLogic()
	registerConversions()
	registerTrigonometry()
}

// ============================================================================
// Shared Helper Functions
// ============================================================================

// Helper: wrap a simple int->int function
func intToInt(fn func(int) int) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		return &eval.IntValue{Value: fn(a.Value)}, nil
	}
}

// Helper: wrap a simple (int,int)->int function
func intIntToInt(fn func(int, int) int) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		return &eval.IntValue{Value: fn(a.Value, b.Value)}, nil
	}
}

// Helper: wrap a (int,int)->(int,error) function
func intIntToIntErr(fn func(int, int) (int, error)) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.IntValue)
		b := args[1].(*eval.IntValue)
		result, err := fn(a.Value, b.Value)
		if err != nil {
			return nil, err
		}
		return &eval.IntValue{Value: result}, nil
	}
}

// Helper: wrap a (float,float)->float function
func floatFloatToFloat(fn func(float64, float64) float64) func(*effects.EffContext, []eval.Value) (eval.Value, error) {
	return func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		a := args[0].(*eval.FloatValue)
		b := args[1].(*eval.FloatValue)
		return &eval.FloatValue{Value: fn(a.Value, b.Value)}, nil
	}
}

// registerBuiltinWithMeta is a convenience wrapper for arithmetic builtins with metadata
func registerBuiltinWithMeta(name string, numArgs int, isPure bool, impl func(*effects.EffContext, []eval.Value) (eval.Value, error), description string, tags []string) {
	// Determine type based on name suffix
	var typeFunc func() types.Type
	var paramDocs []ParamDoc
	var returns string

	if strings.HasSuffix(name, "_Int") {
		if numArgs == 1 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Int()).Returns(T.Int()).Build()
			}
			paramDocs = []ParamDoc{{Name: "a", Description: "Integer operand"}}
			returns = "Integer result"
		} else if numArgs == 2 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Int(), T.Int()).Returns(T.Int()).Build()
			}
			paramDocs = []ParamDoc{
				{Name: "a", Description: "First integer operand"},
				{Name: "b", Description: "Second integer operand"},
			}
			returns = "Integer result"
		}
	} else if strings.HasSuffix(name, "_Float") {
		if numArgs == 1 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Float()).Returns(T.Float()).Build()
			}
			paramDocs = []ParamDoc{{Name: "a", Description: "Float operand"}}
			returns = "Float result"
		} else if numArgs == 2 {
			typeFunc = func() types.Type {
				T := types.NewBuilder()
				return T.Func(T.Float(), T.Float()).Returns(T.Float()).Build()
			}
			paramDocs = []ParamDoc{
				{Name: "a", Description: "First float operand"},
				{Name: "b", Description: "Second float operand"},
			}
			returns = "Float result"
		}
	}

	if typeFunc == nil {
		panic(fmt.Sprintf("cannot infer type for builtin %s", name))
	}

	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/math",
		Name:    name,
		NumArgs: numArgs,
		IsPure:  isPure,
		Type:    typeFunc,
		Impl:    impl,
		Metadata: &BuiltinMetadata{
			Description: description,
			Params:      paramDocs,
			Returns:     returns,
			Since:       "v0.1.0",
			Stability:   StabilityStable,
			Tags:        tags,
			Category:    "math",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register %s: %v", name, err))
	}
}
