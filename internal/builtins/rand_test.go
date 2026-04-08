package builtins

import (
	"regexp"
	"testing"

	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
)

// mockRandContext creates a mock effect context with Rand capability
func mockRandContext() *effects.EffContext {
	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("Rand"))
	return ctx
}

func TestRandIntDeterminism(t *testing.T) {
	ctx := mockRandContext()

	// Set seed and generate sequence
	SetRandSeed(42)
	var seq1 []int
	for i := 0; i < 10; i++ {
		result, err := randIntImpl(ctx, []eval.Value{
			&eval.IntValue{Value: 1},
			&eval.IntValue{Value: 100},
		})
		if err != nil {
			t.Fatalf("randIntImpl failed: %v", err)
		}
		seq1 = append(seq1, result.(*eval.IntValue).Value)
	}

	// Reset to same seed
	SetRandSeed(42)
	var seq2 []int
	for i := 0; i < 10; i++ {
		result, err := randIntImpl(ctx, []eval.Value{
			&eval.IntValue{Value: 1},
			&eval.IntValue{Value: 100},
		})
		if err != nil {
			t.Fatalf("randIntImpl failed: %v", err)
		}
		seq2 = append(seq2, result.(*eval.IntValue).Value)
	}

	// Sequences must match
	for i := range seq1 {
		if seq1[i] != seq2[i] {
			t.Errorf("Determinism failure at index %d: seq1=%d, seq2=%d", i, seq1[i], seq2[i])
		}
	}
}

func TestRandFloatDeterminism(t *testing.T) {
	ctx := mockRandContext()

	// Set seed and generate sequence
	SetRandSeed(123)
	var seq1 []float64
	for i := 0; i < 10; i++ {
		result, err := randFloatImpl(ctx, []eval.Value{
			&eval.FloatValue{Value: 0.0},
			&eval.FloatValue{Value: 1.0},
		})
		if err != nil {
			t.Fatalf("randFloatImpl failed: %v", err)
		}
		seq1 = append(seq1, result.(*eval.FloatValue).Value)
	}

	// Reset to same seed
	SetRandSeed(123)
	var seq2 []float64
	for i := 0; i < 10; i++ {
		result, err := randFloatImpl(ctx, []eval.Value{
			&eval.FloatValue{Value: 0.0},
			&eval.FloatValue{Value: 1.0},
		})
		if err != nil {
			t.Fatalf("randFloatImpl failed: %v", err)
		}
		seq2 = append(seq2, result.(*eval.FloatValue).Value)
	}

	// Sequences must match
	for i := range seq1 {
		if seq1[i] != seq2[i] {
			t.Errorf("Determinism failure at index %d: seq1=%f, seq2=%f", i, seq1[i], seq2[i])
		}
	}
}

func TestRandBoolDeterminism(t *testing.T) {
	ctx := mockRandContext()

	// Set seed and generate sequence
	SetRandSeed(999)
	var seq1 []bool
	for i := 0; i < 20; i++ {
		result, err := randBoolImpl(ctx, []eval.Value{&eval.UnitValue{}})
		if err != nil {
			t.Fatalf("randBoolImpl failed: %v", err)
		}
		seq1 = append(seq1, result.(*eval.BoolValue).Value)
	}

	// Reset to same seed
	SetRandSeed(999)
	var seq2 []bool
	for i := 0; i < 20; i++ {
		result, err := randBoolImpl(ctx, []eval.Value{&eval.UnitValue{}})
		if err != nil {
			t.Fatalf("randBoolImpl failed: %v", err)
		}
		seq2 = append(seq2, result.(*eval.BoolValue).Value)
	}

	// Sequences must match
	for i := range seq1 {
		if seq1[i] != seq2[i] {
			t.Errorf("Determinism failure at index %d: seq1=%v, seq2=%v", i, seq1[i], seq2[i])
		}
	}
}

func TestRandIntRange(t *testing.T) {
	ctx := mockRandContext()
	SetRandSeed(42)

	// Test 100 random ints are within range
	for i := 0; i < 100; i++ {
		result, err := randIntImpl(ctx, []eval.Value{
			&eval.IntValue{Value: 5},
			&eval.IntValue{Value: 10},
		})
		if err != nil {
			t.Fatalf("randIntImpl failed: %v", err)
		}
		v := result.(*eval.IntValue).Value
		if v < 5 || v > 10 {
			t.Errorf("Value %d out of range [5, 10]", v)
		}
	}
}

func TestRandIntSwappedRange(t *testing.T) {
	ctx := mockRandContext()
	SetRandSeed(42)

	// Test that swapped min/max still works (auto-corrected)
	result, err := randIntImpl(ctx, []eval.Value{
		&eval.IntValue{Value: 100}, // max
		&eval.IntValue{Value: 1},   // min
	})
	if err != nil {
		t.Fatalf("randIntImpl failed: %v", err)
	}
	v := result.(*eval.IntValue).Value
	if v < 1 || v > 100 {
		t.Errorf("Value %d out of range [1, 100]", v)
	}
}

func TestRandFloatRange(t *testing.T) {
	ctx := mockRandContext()
	SetRandSeed(42)

	// Test 100 random floats are within range
	for i := 0; i < 100; i++ {
		result, err := randFloatImpl(ctx, []eval.Value{
			&eval.FloatValue{Value: -5.0},
			&eval.FloatValue{Value: 5.0},
		})
		if err != nil {
			t.Fatalf("randFloatImpl failed: %v", err)
		}
		v := result.(*eval.FloatValue).Value
		if v < -5.0 || v >= 5.0 {
			t.Errorf("Value %f out of range [-5.0, 5.0)", v)
		}
	}
}

func TestRandRequiresCapability(t *testing.T) {
	// Context WITHOUT Rand capability
	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("IO")) // Grant IO but NOT Rand

	_, err := randIntImpl(ctx, []eval.Value{
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 10},
	})
	if err == nil {
		t.Error("Expected error when Rand capability is missing")
	}
}

func TestRandSeedImpl(t *testing.T) {
	ctx := mockRandContext()

	// Test setting seed through impl
	result, err := randSeedImpl(ctx, []eval.Value{
		&eval.IntValue{Value: 12345},
	})
	if err != nil {
		t.Fatalf("randSeedImpl failed: %v", err)
	}
	if _, ok := result.(*eval.UnitValue); !ok {
		t.Errorf("Expected UnitValue, got %T", result)
	}

	// Verify seed took effect by checking determinism
	v1, _ := randIntImpl(ctx, []eval.Value{
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 1000000},
	})

	// Reset same seed
	randSeedImpl(ctx, []eval.Value{&eval.IntValue{Value: 12345}})
	v2, _ := randIntImpl(ctx, []eval.Value{
		&eval.IntValue{Value: 1},
		&eval.IntValue{Value: 1000000},
	})

	if v1.(*eval.IntValue).Value != v2.(*eval.IntValue).Value {
		t.Error("randSeedImpl did not produce deterministic behavior")
	}
}

func TestUuid4Format(t *testing.T) {
	ctx := mockRandContext()
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	for i := 0; i < 20; i++ {
		result, err := uuid4Impl(ctx, []eval.Value{&eval.UnitValue{}})
		if err != nil {
			t.Fatalf("uuid4Impl failed: %v", err)
		}
		s := result.(*eval.StringValue).Value
		if !uuidRegex.MatchString(s) {
			t.Errorf("uuid4 output %q does not match UUID v4 format", s)
		}
	}
}

func TestUuid4Uniqueness(t *testing.T) {
	ctx := mockRandContext()
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		result, err := uuid4Impl(ctx, []eval.Value{&eval.UnitValue{}})
		if err != nil {
			t.Fatalf("uuid4Impl failed: %v", err)
		}
		s := result.(*eval.StringValue).Value
		if seen[s] {
			t.Errorf("uuid4 produced duplicate value: %s", s)
		}
		seen[s] = true
	}
}

// TestCryptoSeed_NonDeterministic guards against the docparse cold-start
// regression: prior to the fix, init() seeded math/rand with NewSource(0),
// so independent processes produced identical rand_int sequences. This
// caused API key collisions across Cloud Run cold starts. cryptoSeed() must
// pull fresh entropy from crypto/rand on every call.
func TestCryptoSeed_NonDeterministic(t *testing.T) {
	seeds := make(map[int64]struct{}, 16)
	for i := 0; i < 16; i++ {
		seeds[cryptoSeed()] = struct{}{}
	}
	if len(seeds) < 15 {
		t.Fatalf("cryptoSeed produced %d unique values out of 16 — entropy source is broken", len(seeds))
	}
}

func TestUuid4RequiresCapability(t *testing.T) {
	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("IO")) // Grant IO but NOT Rand

	_, err := uuid4Impl(ctx, []eval.Value{&eval.UnitValue{}})
	if err == nil {
		t.Error("Expected error when Rand capability is missing")
	}
}
