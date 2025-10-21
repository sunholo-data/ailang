package builtins

import (
	"fmt"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

// IO effect builtins for AILANG
// These provide console I/O operations

func init() {
	registerIO()
}

// ============================================================================
// IO Effect Builtins (_io_print, _io_println, _io_readLine)
// ============================================================================

func registerIO() {
	// NOTE: 'print' is NOT registered here - it's entry-module prelude only
	// See internal/pipeline/prelude.go for prelude injection
	// Libraries must use explicit 'import std/io (_io_println)'

	// _io_print
	impl1 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		s := args[0].(*eval.StringValue)
		fmt.Print(s.Value)
		return &eval.UnitValue{}, nil
	}
	type1 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(T.Unit()).Effects("IO")
	}
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_print", NumArgs: 1, IsPure: false, Effect: "IO", Type: type1, Impl: impl1,

		Metadata: &BuiltinMetadata{
			Description: "Print a string to stdout without newline",
			Params: []ParamDoc{
				{Name: "s", Description: "String to print"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: `_io_print("Hello")`, Description: "Prints 'Hello' without newline"},
			},
			SeeAlso:   []string{"_io_println"},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"io", "print", "output", "console"},
			Category:  "io",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _io_print: %v", err))
	}

	// _io_println
	impl2 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		s := args[0].(*eval.StringValue)
		fmt.Println(s.Value)
		return &eval.UnitValue{}, nil
	}
	type2 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.String()).Returns(T.Unit()).Effects("IO")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_println", NumArgs: 1, IsPure: false, Effect: "IO", Type: type2, Impl: impl2,

		Metadata: &BuiltinMetadata{
			Description: "Print a string to stdout with newline",
			Params: []ParamDoc{
				{Name: "s", Description: "String to print"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: `_io_println("Hello")`, Description: "Prints 'Hello' followed by newline"},
			},
			SeeAlso:   []string{"_io_print"},
			Since:     "v0.1.0",
			Stability: StabilityStable,
			Tags:      []string{"io", "println", "output", "console"},
			Category:  "io",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _io_println: %v", err))
	}

	// _io_readLine (stub for v0.3.10)
	impl3 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		// Stub: return empty string
		return &eval.StringValue{Value: ""}, nil
	}
	type3 := func() types.Type {
		T := types.NewBuilder()
		return T.Func().Returns(T.String()).Effects("IO")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_readLine", NumArgs: 0, IsPure: false, Effect: "IO", Type: type3, Impl: impl3,

		Metadata: &BuiltinMetadata{
			Description: "Read a line from stdin (stub implementation)",
			LongDesc:    "Currently returns empty string. Full implementation pending.",
			Returns:     "String read from stdin (currently always empty)",
			Since:       "v0.3.10",
			Stability:   StabilityExperimental,
			Tags:        []string{"io", "read", "input", "console", "stub"},
			Category:    "io",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _io_readLine: %v", err))
	}
}
