package builtins

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
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

	// _io_print — routes through effects.Call() for trace recording
	impl1 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "IO", "print", args)
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

	// _io_println — routes through effects.Call() for trace recording
	impl2 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "IO", "println", args)
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

	// _io_readLine — routes through effects.Call() for trace recording
	// FIXED (v0.4.2): Changed from 0-arg to unit-arg to fix S-CALL0 compatibility
	// Zero-arg builtins now take unit as their parameter: () -> T means (()) -> T
	impl3 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		// Validate unit argument (defense against type system bugs)
		if len(args) != 1 {
			panic("internal invariant violation: _io_readLine expects exactly 1 argument (unit)")
		}
		if _, ok := args[0].(*eval.UnitValue); !ok {
			panic("internal invariant violation: _io_readLine expected unit argument")
		}
		// Route through effects.Call() — pass empty args since effects ioReadLine takes 0 args
		return effects.Call(ctx, "IO", "readLine", nil)
	}
	type3 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Unit()).Returns(T.String()).Effects("IO") // Fixed: () -> string means (()) -> string
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_readLine", NumArgs: 1, IsPure: false, Effect: "IO", Type: type3, Impl: impl3,

		Metadata: &BuiltinMetadata{
			Description: "Read a line from stdin (stub implementation)",
			LongDesc:    "Currently returns empty string. Full implementation pending. Takes unit as parameter for S-CALL0 compatibility.",
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

	// _io_exit — terminate process with exit code
	impl5 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "IO", "exit", args)
	}
	type5 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Int()).Returns(T.Unit()).Effects("IO")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_exit", NumArgs: 1, IsPure: false, Effect: "IO", Type: type5, Impl: impl5,

		Metadata: &BuiltinMetadata{
			Description: "Terminate the process with an explicit exit code",
			LongDesc:    "Terminates the AILANG process with the given exit code. Uses a sentinel panic that the runtime catches after flushing telemetry. Code after exit() is unreachable.",
			Params: []ParamDoc{
				{Name: "code", Description: "Exit code (0 = success, non-zero = failure)"},
			},
			Returns: "Unit (never actually returns — process terminates)",
			Examples: []Example{
				{Code: `_io_exit(0)`, Description: "Exit successfully"},
				{Code: `_io_exit(1)`, Description: "Exit with error"},
			},
			SeeAlso:   []string{"_io_println"},
			Since:     "v0.10.0",
			Stability: StabilityStable,
			Tags:      []string{"io", "exit", "process", "cli"},
			Category:  "io",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _io_exit: %v", err))
	}

	// _io_writeBytes — write raw bytes to stdout
	impl4 := func(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
		return effects.Call(ctx, "IO", "writeBytes", args)
	}
	type4 := func() types.Type {
		T := types.NewBuilder()
		return T.Func(T.Bytes()).Returns(T.Unit()).Effects("IO")
	}
	err = RegisterEffectBuiltin(BuiltinSpec{
		Module: "std/io", Name: "_io_writeBytes", NumArgs: 1, IsPure: false, Effect: "IO", Type: type4, Impl: impl4,

		Metadata: &BuiltinMetadata{
			Description: "Write raw bytes to stdout",
			LongDesc:    "Writes binary data directly to stdout. Enables piping to external tools: `ailang run ... | afplay -f LEI16 -r 24000 -c 1 -`.",
			Params: []ParamDoc{
				{Name: "data", Description: "Bytes to write to stdout"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: `_io_writeBytes(pcmData)`, Description: "Writes raw PCM audio to stdout"},
			},
			SeeAlso:   []string{"_io_print", "_io_println"},
			Since:     "v0.8.2",
			Stability: StabilityStable,
			Tags:      []string{"io", "write", "bytes", "binary", "stdout", "pipe"},
			Category:  "io",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register _io_writeBytes: %v", err))
	}
}
