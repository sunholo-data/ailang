package builtins

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestClockNow(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Clock")

	// Get current time before calling builtin
	before := time.Now().UnixMilli()

	// Call _clock_now with unit argument
	result, err := clockNowImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	// Get current time after calling builtin
	after := time.Now().UnixMilli()

	assert.NoError(t, err)
	timestamp := testctx.GetInt(result)

	// Timestamp should be between before and after
	assert.GreaterOrEqual(t, timestamp, int(before), "Timestamp should be >= time before call")
	assert.LessOrEqual(t, timestamp, int(after), "Timestamp should be <= time after call")
}

func TestClockNowRequiresCapability(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	// Don't grant Clock capability

	_, err := clockNowImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Clock")
}

func TestClockSleep(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Clock")

	// Sleep for 50ms
	start := time.Now()
	result, err := clockSleepImpl(ctx.EffContext, []eval.Value{
		testctx.MakeInt(50),
	})
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.IsType(t, &eval.UnitValue{}, result)

	// Should have slept at least 40ms (with some tolerance for timing)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(40),
		"Should sleep for at least 40ms (50ms requested with 10ms tolerance)")
}

func TestClockSleepNegative(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Clock")

	// Negative sleep should be treated as 0 (no sleep)
	start := time.Now()
	result, err := clockSleepImpl(ctx.EffContext, []eval.Value{
		testctx.MakeInt(-100),
	})
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.IsType(t, &eval.UnitValue{}, result)

	// Should complete almost immediately (less than 10ms)
	assert.Less(t, elapsed.Milliseconds(), int64(10),
		"Negative sleep should not block")
}

func TestClockSleepZero(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Clock")

	// Zero sleep should return immediately
	start := time.Now()
	result, err := clockSleepImpl(ctx.EffContext, []eval.Value{
		testctx.MakeInt(0),
	})
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.IsType(t, &eval.UnitValue{}, result)

	// Should complete almost immediately (less than 10ms)
	assert.Less(t, elapsed.Milliseconds(), int64(10),
		"Zero sleep should not block")
}

func TestClockSleepRequiresCapability(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	// Don't grant Clock capability

	_, err := clockSleepImpl(ctx.EffContext, []eval.Value{
		testctx.MakeInt(100),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Clock")
}

// TestClockNowWithBudget tests that clock operations respect budget limits
func TestClockNowWithBudget(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Clock")
	ctx.SetBudget(map[string]*int{"Clock": intPtr(2)}) // Allow only 2 Clock operations

	// First call should succeed
	result1, err1 := clockNowImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})
	assert.NoError(t, err1)
	assert.NotNil(t, result1)

	// Second call should succeed
	result2, err2 := clockNowImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})
	assert.NoError(t, err2)
	assert.NotNil(t, result2)

	// Third call should fail with budget exhausted
	_, err3 := clockNowImpl(ctx.EffContext, []eval.Value{&eval.UnitValue{}})
	assert.Error(t, err3)
	assert.Contains(t, err3.Error(), "budget exhausted")
}

// TestClockSleepWithBudget tests that sleep operations respect budget limits
func TestClockSleepWithBudget(t *testing.T) {
	ctx := testctx.NewMockEffContext()
	ctx.GrantAll("Clock")
	ctx.SetBudget(map[string]*int{"Clock": intPtr(1)}) // Allow only 1 Clock operation

	// First call should succeed
	result1, err1 := clockSleepImpl(ctx.EffContext, []eval.Value{testctx.MakeInt(0)})
	assert.NoError(t, err1)
	assert.NotNil(t, result1)

	// Second call should fail with budget exhausted
	_, err2 := clockSleepImpl(ctx.EffContext, []eval.Value{testctx.MakeInt(0)})
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "budget exhausted")
}

func intPtr(i int) *int {
	return &i
}
