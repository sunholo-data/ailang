package builtins

import (
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

func init() {
	registerDebugLog()
	registerDebugCheck()
}

// _debug_log: Log a message to the debug context
func registerDebugLog() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/debug",
		Name:    "_debug_log",
		NumArgs: 2, // msg: string, location: string (hidden arg)
		Effect:  "Debug",
		Type:    makeDebugLogType,
		Impl:    debugLogImpl,
		Metadata: &BuiltinMetadata{
			Description: "Log a message to the debug context",
			Params: []ParamDoc{
				{Name: "msg", Description: "Message to log"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: `Debug.log("updating entity")`, Description: "Log a debug message"},
				{Code: `Debug.log("health: " ++ show(health))`, Description: "Log with interpolation"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"debug", "logging", "trace"},
			Category:  "debug",
		},
	})
	if err != nil {
		panic("failed to register _debug_log builtin: " + err.Error())
	}
}

func makeDebugLogType() types.Type {
	T := types.NewBuilder()
	// (msg: string, location: string) -> () ! {Debug}
	// Note: location is a hidden argument injected by the compiler
	return T.Func(T.String(), T.String()).Returns(T.Unit()).Effects("Debug")
}

func debugLogImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check Debug capability
	if err := ctx.RequireCapWithBudget("Debug", ""); err != nil {
		return nil, err
	}

	// Call through to the effect operation
	return effects.Call(ctx, "Debug", "log", args)
}

// _debug_check: Record an assertion result (renamed from assert - reserved keyword)
func registerDebugCheck() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/debug",
		Name:    "_debug_check",
		NumArgs: 3, // cond: bool, msg: string, location: string (hidden arg)
		Effect:  "Debug",
		Type:    makeDebugCheckType,
		Impl:    debugCheckImpl,
		Metadata: &BuiltinMetadata{
			Description: "Record an assertion result (collected, not thrown)",
			Params: []ParamDoc{
				{Name: "cond", Description: "Condition to assert (true = passed)"},
				{Name: "msg", Description: "Message describing the assertion"},
			},
			Returns: "Unit (no return value, even if assertion fails)",
			Examples: []Example{
				{Code: `Debug.check(health >= 0, "health must be non-negative")`, Description: "Check invariant"},
				{Code: `Debug.check(x < 100, "x out of bounds: " ++ show(x))`, Description: "Check with context"},
			},
			Since:     "v0.5.1",
			Stability: StabilityStable,
			Tags:      []string{"debug", "assertion", "invariant", "trace"},
			Category:  "debug",
		},
	})
	if err != nil {
		panic("failed to register _debug_check builtin: " + err.Error())
	}
}

func makeDebugCheckType() types.Type {
	T := types.NewBuilder()
	// (cond: bool, msg: string, location: string) -> () ! {Debug}
	// Note: location is a hidden argument injected by the compiler
	return T.Func(T.Bool(), T.String(), T.String()).Returns(T.Unit()).Effects("Debug")
}

func debugCheckImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check Debug capability
	if err := ctx.RequireCapWithBudget("Debug", ""); err != nil {
		return nil, err
	}

	// Call through to the effect operation
	return effects.Call(ctx, "Debug", "check", args)
}
