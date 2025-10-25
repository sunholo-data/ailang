package builtins

import (
	"time"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/types"
)

func init() {
	registerClockNow()
	registerClockSleep()
}

// _clock_now: Get current Unix timestamp in milliseconds
func registerClockNow() {
	RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/clock",
		Name:    "_clock_now",
		NumArgs: 0,
		Effect:  "Clock",
		Type:    makeClockNowType,
		Impl:    clockNowImpl,
		Metadata: &BuiltinMetadata{
			Description: "Get current Unix timestamp in milliseconds",
			Params:      []ParamDoc{},
			Returns:     "Unix timestamp in milliseconds",
			Examples: []Example{
				{Code: "_clock_now()", Description: "Returns current timestamp (e.g., 1698765432123)"},
			},
			Since:     "v0.3.0",
			Stability: StabilityStable,
			Tags:      []string{"time", "clock", "timestamp"},
			Category:  "clock",
		},
	})
}

func makeClockNowType() types.Type {
	T := types.NewBuilder()
	return T.Func().Returns(T.Int()).Effects("Clock")
}

func clockNowImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check Clock capability
	if err := ctx.RequireCap("Clock"); err != nil {
		return nil, err
	}

	// Get current time in milliseconds since Unix epoch
	now := time.Now().UnixMilli()
	return &eval.IntValue{Value: int(now)}, nil
}

// _clock_sleep: Sleep for specified milliseconds
func registerClockSleep() {
	RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/clock",
		Name:    "_clock_sleep",
		NumArgs: 1,
		Effect:  "Clock",
		Type:    makeClockSleepType,
		Impl:    clockSleepImpl,
		Metadata: &BuiltinMetadata{
			Description: "Sleep for specified milliseconds",
			Params: []ParamDoc{
				{Name: "ms", Description: "Number of milliseconds to sleep"},
			},
			Returns: "Unit (no return value)",
			Examples: []Example{
				{Code: "_clock_sleep(1000)", Description: "Sleep for 1 second (1000ms)"},
				{Code: "_clock_sleep(100)", Description: "Sleep for 100 milliseconds"},
			},
			Since:     "v0.3.20",
			Stability: StabilityStable,
			Tags:      []string{"time", "clock", "sleep", "delay"},
			Category:  "clock",
		},
	})
}

func makeClockSleepType() types.Type {
	T := types.NewBuilder()
	return T.Func(T.Int()).Returns(T.Unit()).Effects("Clock")
}

func clockSleepImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
	// Check Clock capability
	if err := ctx.RequireCap("Clock"); err != nil {
		return nil, err
	}

	// Extract milliseconds argument
	ms := args[0].(*eval.IntValue).Value
	if ms < 0 {
		ms = 0 // Treat negative values as 0 (no sleep)
	}

	// Sleep for the specified duration
	time.Sleep(time.Duration(ms) * time.Millisecond)

	// Return unit value
	return &eval.UnitValue{}, nil
}
