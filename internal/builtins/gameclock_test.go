package builtins

import (
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// mockClockContext creates a mock effect context with Clock capability
func mockClockContext() *effects.EffContext {
	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("Clock"))
	return ctx
}

func TestSetAndGetGameClock(t *testing.T) {
	// Set game clock
	SetGameClock(0.016, 5.5, 330)

	// Get and verify
	state := GetGameClock()
	if state.DeltaTime != 0.016 {
		t.Errorf("Expected DeltaTime=0.016, got %f", state.DeltaTime)
	}
	if state.TotalTime != 5.5 {
		t.Errorf("Expected TotalTime=5.5, got %f", state.TotalTime)
	}
	if state.FrameCount != 330 {
		t.Errorf("Expected FrameCount=330, got %d", state.FrameCount)
	}
}

func TestDeltaTimeImpl(t *testing.T) {
	ctx := mockClockContext()

	// Set known delta time
	SetGameClock(0.0166667, 10.0, 600)

	result, err := deltaTimeImpl(ctx, []eval.Value{&eval.UnitValue{}})
	if err != nil {
		t.Fatalf("deltaTimeImpl failed: %v", err)
	}

	floatVal, ok := result.(*eval.FloatValue)
	if !ok {
		t.Fatalf("Expected FloatValue, got %T", result)
	}

	// Check value matches what we set
	if floatVal.Value != 0.0166667 {
		t.Errorf("Expected 0.0166667, got %f", floatVal.Value)
	}
}

func TestTotalTimeImpl(t *testing.T) {
	ctx := mockClockContext()

	// Set known total time
	SetGameClock(0.016, 123.456, 7500)

	result, err := totalTimeImpl(ctx, []eval.Value{&eval.UnitValue{}})
	if err != nil {
		t.Fatalf("totalTimeImpl failed: %v", err)
	}

	floatVal, ok := result.(*eval.FloatValue)
	if !ok {
		t.Fatalf("Expected FloatValue, got %T", result)
	}

	if floatVal.Value != 123.456 {
		t.Errorf("Expected 123.456, got %f", floatVal.Value)
	}
}

func TestFrameCountImpl(t *testing.T) {
	ctx := mockClockContext()

	// Set known frame count
	SetGameClock(0.016, 100.0, 6000)

	result, err := frameCountImpl(ctx, []eval.Value{&eval.UnitValue{}})
	if err != nil {
		t.Fatalf("frameCountImpl failed: %v", err)
	}

	intVal, ok := result.(*eval.IntValue)
	if !ok {
		t.Fatalf("Expected IntValue, got %T", result)
	}

	if intVal.Value != 6000 {
		t.Errorf("Expected 6000, got %d", intVal.Value)
	}
}

func TestGameClockRequiresCapability(t *testing.T) {
	// Context WITHOUT Clock capability
	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("IO")) // Grant IO but NOT Clock

	_, err := deltaTimeImpl(ctx, []eval.Value{&eval.UnitValue{}})
	if err == nil {
		t.Error("Expected error when Clock capability is missing")
	}

	_, err = totalTimeImpl(ctx, []eval.Value{&eval.UnitValue{}})
	if err == nil {
		t.Error("Expected error when Clock capability is missing")
	}

	_, err = frameCountImpl(ctx, []eval.Value{&eval.UnitValue{}})
	if err == nil {
		t.Error("Expected error when Clock capability is missing")
	}
}

func TestGameClockDeterminism(t *testing.T) {
	ctx := mockClockContext()

	// Simulate a game loop - values should be stable within a frame
	SetGameClock(0.016, 50.0, 3000)

	// Multiple reads in same "frame" should return same values
	for i := 0; i < 5; i++ {
		dt, _ := deltaTimeImpl(ctx, []eval.Value{&eval.UnitValue{}})
		tt, _ := totalTimeImpl(ctx, []eval.Value{&eval.UnitValue{}})
		fc, _ := frameCountImpl(ctx, []eval.Value{&eval.UnitValue{}})

		if dt.(*eval.FloatValue).Value != 0.016 {
			t.Errorf("Iteration %d: DeltaTime changed unexpectedly", i)
		}
		if tt.(*eval.FloatValue).Value != 50.0 {
			t.Errorf("Iteration %d: TotalTime changed unexpectedly", i)
		}
		if fc.(*eval.IntValue).Value != 3000 {
			t.Errorf("Iteration %d: FrameCount changed unexpectedly", i)
		}
	}

	// Advance to next frame
	SetGameClock(0.017, 50.016, 3001)

	// Values should update
	dt, _ := deltaTimeImpl(ctx, []eval.Value{&eval.UnitValue{}})
	tt, _ := totalTimeImpl(ctx, []eval.Value{&eval.UnitValue{}})
	fc, _ := frameCountImpl(ctx, []eval.Value{&eval.UnitValue{}})

	if dt.(*eval.FloatValue).Value != 0.017 {
		t.Error("DeltaTime didn't update after frame advance")
	}
	if tt.(*eval.FloatValue).Value != 50.016 {
		t.Error("TotalTime didn't update after frame advance")
	}
	if fc.(*eval.IntValue).Value != 3001 {
		t.Error("FrameCount didn't update after frame advance")
	}
}
