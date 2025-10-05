package effects

import (
	"os"
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/eval"
)

// TestClockNow_RealTime verifies that now() returns a reasonable timestamp
func TestClockNow_RealTime(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Clock"))

	// Real time mode (monotonic)
	before := int(time.Now().UnixMilli())
	result, err := clockNow(ctx, []eval.Value{})
	after := int(time.Now().UnixMilli())

	if err != nil {
		t.Fatalf("clockNow failed: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", result)
	}

	if intVal.Value < before || intVal.Value > after {
		t.Errorf("clockNow returned %d, expected between %d and %d", intVal.Value, before, after)
	}
}

// TestClockNow_Monotonic is a flaky-guard test that verifies time never goes backwards
//
// This test calls now() multiple times and ensures the returned values are
// strictly non-decreasing. This protects against NTP adjustments, DST changes,
// and manual clock changes.
func TestClockNow_Monotonic(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Clock"))

	// Call now() 10 times with small delays
	times := make([]int, 10)
	for i := 0; i < 10; i++ {
		result, err := clockNow(ctx, []eval.Value{})
		if err != nil {
			t.Fatalf("clockNow failed on iteration %d: %v", i, err)
		}

		intVal := result.(*eval.IntValue)
		times[i] = intVal.Value

		// Small delay to allow time to advance
		time.Sleep(1 * time.Millisecond)
	}

	// Verify monotonic (never decreases)
	for i := 1; i < 10; i++ {
		if times[i] < times[i-1] {
			t.Errorf("time went backwards! times[%d]=%d < times[%d]=%d", i, times[i], i-1, times[i-1])
		}
	}
}

// TestClockSleep_RealDelay verifies that sleep() actually blocks for the specified duration
func TestClockSleep_RealDelay(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Clock"))

	start := time.Now()
	_, err := clockSleep(ctx, []eval.Value{&eval.IntValue{Value: 100}})
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		t.Fatalf("clockSleep failed: %v", err)
	}

	if elapsed < 100 {
		t.Errorf("clockSleep returned too early: %dms", elapsed)
	}

	// Allow 50ms variance for OS scheduling
	if elapsed > 150 {
		t.Errorf("clockSleep took too long: %dms", elapsed)
	}
}

// TestClockVirtualTime_Deterministic is a flaky-guard test that verifies full determinism
//
// This test runs the same sequence 100 times in virtual time mode and verifies
// that every run produces identical results. This ensures reproducibility for
// benchmarks and AI training data generation.
func TestClockVirtualTime_Deterministic(t *testing.T) {
	// Run 100 times to catch any flakiness
	for run := 0; run < 100; run++ {
		// Set AILANG_SEED to enable virtual time
		os.Setenv("AILANG_SEED", "42")

		ctx := NewEffContext()
		ctx.Grant(NewCapability("Clock"))

		// Virtual now (should be 0)
		result, err := clockNow(ctx, []eval.Value{})
		if err != nil {
			t.Fatalf("run %d: clockNow failed: %v", run, err)
		}

		intVal := result.(*eval.IntValue)
		if intVal.Value != 0 {
			t.Errorf("run %d: initial time not 0, got %d", run, intVal.Value)
		}

		// Virtual sleep 500ms (no actual delay)
		start := time.Now()
		_, err = clockSleep(ctx, []eval.Value{&eval.IntValue{Value: 500}})
		elapsed := time.Since(start).Milliseconds()

		if err != nil {
			t.Fatalf("run %d: clockSleep failed: %v", run, err)
		}

		// Verify no real time elapsed (virtual sleep is instant)
		if elapsed > 10 {
			t.Errorf("run %d: virtual sleep took real time: %dms", run, elapsed)
		}

		// Virtual now (should be 500)
		result, err = clockNow(ctx, []eval.Value{})
		if err != nil {
			t.Fatalf("run %d: clockNow failed after sleep: %v", run, err)
		}

		intVal = result.(*eval.IntValue)
		if intVal.Value != 500 {
			t.Errorf("run %d: time not advanced to 500, got %d", run, intVal.Value)
		}

		// Clean up env var
		os.Unsetenv("AILANG_SEED")
	}
}

// TestClockSleep_NegativeDuration verifies that sleep() rejects negative durations
func TestClockSleep_NegativeDuration(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Clock"))

	_, err := clockSleep(ctx, []eval.Value{&eval.IntValue{Value: -100}})

	if err == nil {
		t.Fatal("expected error for negative duration, got nil")
	}

	// Verify error code
	if err.Error()[:23] != "E_CLOCK_NEGATIVE_SLEEP:" {
		t.Errorf("expected E_CLOCK_NEGATIVE_SLEEP error, got: %v", err)
	}
}

// TestClockNow_NoCapability verifies that now() fails without Clock capability
func TestClockNow_NoCapability(t *testing.T) {
	ctx := NewEffContext()
	// Do NOT grant Clock capability

	_, err := Call(ctx, "Clock", "now", []eval.Value{})

	if err == nil {
		t.Fatal("expected capability error, got nil")
	}

	// Verify it's a capability error
	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Errorf("expected CapabilityError, got %T: %v", err, err)
	}

	if capErr.Effect != "Clock" {
		t.Errorf("expected effect 'Clock', got '%s'", capErr.Effect)
	}
}

// TestClockSleep_NoCapability verifies that sleep() fails without Clock capability
func TestClockSleep_NoCapability(t *testing.T) {
	ctx := NewEffContext()
	// Do NOT grant Clock capability

	_, err := Call(ctx, "Clock", "sleep", []eval.Value{&eval.IntValue{Value: 100}})

	if err == nil {
		t.Fatal("expected capability error, got nil")
	}

	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Errorf("expected CapabilityError, got %T: %v", err, err)
	}

	if capErr.Effect != "Clock" {
		t.Errorf("expected effect 'Clock', got '%s'", capErr.Effect)
	}
}

// TestClockNow_WrongArgCount verifies error handling for wrong argument count
func TestClockNow_WrongArgCount(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Clock"))

	_, err := clockNow(ctx, []eval.Value{&eval.IntValue{Value: 42}})

	if err == nil {
		t.Fatal("expected error for wrong arg count, got nil")
	}

	// Should mention expected 0 arguments
	if err.Error()[:21] != "E_CLOCK_TYPE_ERROR: n" {
		t.Errorf("expected E_CLOCK_TYPE_ERROR for arg count, got: %v", err)
	}
}

// TestClockSleep_WrongArgType verifies error handling for wrong argument type
func TestClockSleep_WrongArgType(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("Clock"))

	_, err := clockSleep(ctx, []eval.Value{&eval.StringValue{Value: "not an int"}})

	if err == nil {
		t.Fatal("expected error for wrong arg type, got nil")
	}

	// Should mention expected Int
	if err.Error()[:21] != "E_CLOCK_TYPE_ERROR: s" {
		t.Errorf("expected E_CLOCK_TYPE_ERROR for arg type, got: %v", err)
	}
}
