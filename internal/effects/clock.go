package effects

import (
	"fmt"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// init registers Clock effect operations
func init() {
	RegisterOp("Clock", "now", clockNow)
	RegisterOp("Clock", "sleep", clockSleep)
}

// clockNow implements Clock.now() -> Int
//
// Returns the current Unix timestamp in milliseconds.
//
// Production mode (AILANG_SEED unset):
//   - Uses monotonic time: epoch + time.Since(startTime)
//   - Immune to NTP adjustments, DST, manual clock changes
//   - Guarantees time never goes backwards
//
// Deterministic mode (AILANG_SEED set):
//   - Returns virtual time (starts at 0)
//   - Fully reproducible across multiple runs
//   - No real time dependency
//
// Parameters:
//   - ctx: Effect context (capability check already done by Call())
//   - args: [] - no arguments
//
// Returns:
//   - IntValue with current timestamp in milliseconds
//   - Error if wrong number of arguments
//
// Example AILANG code:
//
//	let start = now()  -- e.g., 1730000000000
func clockNow(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("E_CLOCK_TYPE_ERROR: now: expected 0 arguments, got %d", len(args))
	}

	// Deterministic mode: use virtual time (starts at epoch 0)
	if ctx.Env.Seed != 0 {
		return &eval.IntValue{Value: int(ctx.Clock.virtual)}, nil
	}

	// Production mode: monotonic time (epoch + elapsed)
	elapsed := time.Since(ctx.Clock.startTime).Milliseconds()
	return &eval.IntValue{Value: int(ctx.Clock.epoch + elapsed)}, nil
}

// clockSleep implements Clock.sleep(ms: Int) -> ()
//
// Sleeps for the specified number of milliseconds.
//
// Production mode (AILANG_SEED unset):
//   - Blocks for the specified duration
//   - Uses time.Sleep with cancellation support structure
//   - Future: Can be interrupted with context cancellation
//
// Deterministic mode (AILANG_SEED set):
//   - Advances virtual time (no actual delay)
//   - Returns immediately (instant execution)
//   - Fully reproducible for benchmarking
//
// Parameters:
//   - ctx: Effect context
//   - args: [IntValue] - milliseconds to sleep
//
// Returns:
//   - UnitValue on success
//   - Error if wrong number/type of arguments or negative duration
//
// Example AILANG code:
//
//	sleep(1000)  -- sleep for 1 second
func clockSleep(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("E_CLOCK_TYPE_ERROR: sleep: expected 1 argument, got %d", len(args))
	}

	ms, ok := args[0].(*eval.IntValue)
	if !ok {
		return nil, fmt.Errorf("E_CLOCK_TYPE_ERROR: sleep: expected Int, got %T", args[0])
	}

	if ms.Value < 0 {
		return nil, fmt.Errorf("E_CLOCK_NEGATIVE_SLEEP: sleep: negative duration %d", ms.Value)
	}

	// Deterministic mode: advance virtual time (no actual sleep)
	if ctx.Env.Seed != 0 {
		ctx.Clock.virtual += int64(ms.Value)
		return &eval.UnitValue{}, nil
	}

	// Production mode: real sleep
	// Note: Using select for future cancellation support
	<-time.After(time.Duration(ms.Value) * time.Millisecond)
	return &eval.UnitValue{}, nil
}
