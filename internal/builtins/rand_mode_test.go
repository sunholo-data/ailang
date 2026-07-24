package builtins

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

func randCtx() *effects.EffContext {
	ctx := effects.NewEffContext([]string{})
	ctx.Grant(effects.NewCapability("Rand"))
	return ctx
}

func intArgs(min, max int) []eval.Value {
	return []eval.Value{&eval.IntValue{Value: min}, &eval.IntValue{Value: max}}
}

// TestRandIntImpl_OsMode_Unchanged proves os mode (the default, empty mode
// stack) still draws from the global source and stays in range — the byte-
// identical-behaviour guarantee for bare !{Rand}.
func TestRandIntImpl_OsMode_Unchanged(t *testing.T) {
	ctx := randCtx()
	if ctx.CurrentRandMode() != "os" {
		t.Fatalf("default mode should be os, got %q", ctx.CurrentRandMode())
	}
	for i := 0; i < 200; i++ {
		v, err := randIntImpl(ctx, intArgs(1, 6))
		if err != nil {
			t.Fatalf("randIntImpl(os): %v", err)
		}
		n := v.(*eval.IntValue).Value
		if n < 1 || n > 6 {
			t.Fatalf("os draw %d out of [1,6]", n)
		}
	}
}

// TestRandIntImpl_SeededMode_Deterministic proves that under mode=seeded with a
// seed provided, two fresh contexts draw identical sequences via the builtin.
func TestRandIntImpl_SeededMode_Deterministic(t *testing.T) {
	seq := func(seed int64) []int {
		ctx := randCtx()
		ctx.SetSeededModeSeed(seed)
		ctx.PushRandMode("seeded")
		out := make([]int, 5)
		for i := range out {
			v, err := randIntImpl(ctx, intArgs(1, 1000000))
			if err != nil {
				t.Fatalf("randIntImpl(seeded): %v", err)
			}
			out[i] = v.(*eval.IntValue).Value
		}
		return out
	}
	a, b := seq(42), seq(42)
	if !equalInts(a, b) {
		t.Errorf("seeded builtin draws not deterministic: %v vs %v", a, b)
	}
	if equalInts(a, seq(1)) {
		t.Errorf("different seeds gave identical seeded builtin draws: %v", a)
	}
}

// TestRandIntImpl_SeededMode_NoSeed_Error proves the builtin surfaces the typed
// SeededModeError when mode=seeded is active but no seed was provided.
func TestRandIntImpl_SeededMode_NoSeed_Error(t *testing.T) {
	ctx := randCtx() // no seed
	ctx.PushRandMode("seeded")
	_, err := randIntImpl(ctx, intArgs(1, 6))
	if err == nil {
		t.Fatal("expected SeededModeError from seeded draw with no seed, got nil")
	}
	var sme *effects.SeededModeError
	if !errors.As(err, &sme) {
		t.Fatalf("expected *effects.SeededModeError, got %T: %v", err, err)
	}
}

// TestRandIntImpl_CryptoMode_Range proves crypto mode draws stay in range and
// span the full interval (statistical smoke).
func TestRandIntImpl_CryptoMode_Range(t *testing.T) {
	ctx := randCtx()
	ctx.PushRandMode("crypto")
	seen := map[int]bool{}
	for i := 0; i < 2000; i++ {
		v, err := randIntImpl(ctx, intArgs(1, 6))
		if err != nil {
			t.Fatalf("randIntImpl(crypto): %v", err)
		}
		n := v.(*eval.IntValue).Value
		if n < 1 || n > 6 {
			t.Fatalf("crypto draw %d out of [1,6]", n)
		}
		seen[n] = true
	}
	if len(seen) != 6 {
		t.Errorf("crypto draws covered %d/6 buckets over 2000 draws", len(seen))
	}
}

// TestRandFloatImpl_SeededMode_Deterministic mirrors the int test for floats.
func TestRandFloatImpl_SeededMode_Deterministic(t *testing.T) {
	seq := func(seed int64) []float64 {
		ctx := randCtx()
		ctx.SetSeededModeSeed(seed)
		ctx.PushRandMode("seeded")
		out := make([]float64, 5)
		for i := range out {
			v, err := randFloatImpl(ctx, []eval.Value{
				&eval.FloatValue{Value: 0.0}, &eval.FloatValue{Value: 1.0},
			})
			if err != nil {
				t.Fatalf("randFloatImpl(seeded): %v", err)
			}
			out[i] = v.(*eval.FloatValue).Value
		}
		return out
	}
	a, b := seq(42), seq(42)
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("seeded float draws not deterministic at %d: %v vs %v", i, a, b)
		}
	}
}

// TestRandBoolImpl_SeededMode_Deterministic mirrors for bool.
func TestRandBoolImpl_SeededMode_Deterministic(t *testing.T) {
	seq := func(seed int64) []bool {
		ctx := randCtx()
		ctx.SetSeededModeSeed(seed)
		ctx.PushRandMode("seeded")
		out := make([]bool, 8)
		for i := range out {
			v, err := randBoolImpl(ctx, []eval.Value{&eval.UnitValue{}})
			if err != nil {
				t.Fatalf("randBoolImpl(seeded): %v", err)
			}
			out[i] = v.(*eval.BoolValue).Value
		}
		return out
	}
	a, b := seq(42), seq(42)
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("seeded bool draws not deterministic at %d", i)
		}
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
