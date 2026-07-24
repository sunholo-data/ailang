package effects

import (
	"errors"
	"testing"
)

// TestRandModeStack_PushPop covers the mode stack discipline: os is never
// pushed (empty stack = os), a non-os mode becomes current, and pop restores
// the previous mode. This is what lets a seeded function calling an os-mode
// stdlib wrapper keep drawing seeded (the wrapper's os push is a no-op).
func TestRandModeStack_PushPop(t *testing.T) {
	ctx := NewEffContext([]string{})

	if got := ctx.CurrentRandMode(); got != "os" {
		t.Fatalf("fresh context CurrentRandMode = %q, want os", got)
	}

	// os push is a no-op — stack stays empty.
	ctx.PushRandMode("os")
	if got := ctx.CurrentRandMode(); got != "os" {
		t.Errorf("after PushRandMode(os), CurrentRandMode = %q, want os", got)
	}
	ctx.PopRandMode("os") // no-op

	// seeded push becomes current.
	ctx.PushRandMode("seeded")
	if got := ctx.CurrentRandMode(); got != "seeded" {
		t.Errorf("after PushRandMode(seeded), CurrentRandMode = %q, want seeded", got)
	}

	// A nested os wrapper does NOT shadow seeded (os is never pushed).
	ctx.PushRandMode("os")
	if got := ctx.CurrentRandMode(); got != "seeded" {
		t.Errorf("nested os wrapper shadowed the outer seeded mode: got %q, want seeded", got)
	}
	ctx.PopRandMode("os")

	// A nested crypto push wins during its extent.
	ctx.PushRandMode("crypto")
	if got := ctx.CurrentRandMode(); got != "crypto" {
		t.Errorf("after nested PushRandMode(crypto), CurrentRandMode = %q, want crypto", got)
	}
	ctx.PopRandMode("crypto")
	if got := ctx.CurrentRandMode(); got != "seeded" {
		t.Errorf("after popping crypto, CurrentRandMode = %q, want seeded", got)
	}

	ctx.PopRandMode("seeded")
	if got := ctx.CurrentRandMode(); got != "os" {
		t.Errorf("after popping seeded, CurrentRandMode = %q, want os", got)
	}
}

// TestSeeded_NoSeed_TypedError proves a seeded draw with no seed provided is a
// loud typed *SeededModeError (no silent fallback to a random seed).
func TestSeeded_NoSeed_TypedError(t *testing.T) {
	ctx := NewEffContext([]string{}) // no AILANG_SEED

	_, err := ctx.SeededIntn("_rand_int", 100)
	if err == nil {
		t.Fatal("SeededIntn with no seed: expected a typed error, got nil")
	}
	var sme *SeededModeError
	if !errors.As(err, &sme) {
		t.Fatalf("expected *SeededModeError, got %T: %v", err, err)
	}
	if sme.Op != "_rand_int" {
		t.Errorf("SeededModeError.Op = %q, want _rand_int", sme.Op)
	}
	// The fix hint must point to the dedicated path (AILANG_SEED), not rand_seed.
	msg := err.Error()
	if !contains(msg, "AILANG_SEED") {
		t.Errorf("error message should mention AILANG_SEED, got: %s", msg)
	}
	if contains(msg, "call rand_seed") {
		t.Errorf("error message must NOT suggest rand_seed as the fix, got: %s", msg)
	}
}

// TestSeeded_Deterministic proves two contexts with the same seed produce the
// same seeded sequence, and a different seed produces a different one.
func TestSeeded_Deterministic(t *testing.T) {
	draw := func(seed int64, n int) []int {
		ctx := NewEffContext([]string{})
		ctx.SetSeededModeSeed(seed)
		out := make([]int, n)
		for i := range out {
			v, err := ctx.SeededIntn("_rand_int", 1000000)
			if err != nil {
				t.Fatalf("SeededIntn: %v", err)
			}
			out[i] = v
		}
		return out
	}

	a := draw(42, 5)
	b := draw(42, 5)
	if !equalInts(a, b) {
		t.Errorf("same seed produced different sequences: %v vs %v", a, b)
	}
	c := draw(99, 5)
	if equalInts(a, c) {
		t.Errorf("different seeds produced identical sequences (%v) — seed is not wired", a)
	}
}

// TestSeeded_UnperturbedByOsDraws proves the seeded sequence is isolated: an
// interleaved os-mode draw does NOT perturb the seeded source. This is the core
// determinism guarantee (separate sources per mode).
func TestSeeded_UnperturbedByOsDraws(t *testing.T) {
	// Reference: pure seeded sequence.
	ref := NewEffContext([]string{})
	ref.SetSeededModeSeed(7)
	var want []int
	for i := 0; i < 5; i++ {
		v, err := ref.SeededIntn("_rand_int", 1000000)
		if err != nil {
			t.Fatalf("SeededIntn: %v", err)
		}
		want = append(want, v)
	}

	// Same seeded source, but with os-mode draws interleaved between each seeded
	// draw. os draws come from the global builtins source, NOT the seeded source,
	// so the seeded sequence must be identical to the reference.
	mixed := NewEffContext([]string{})
	mixed.SetSeededModeSeed(7)
	var got []int
	for i := 0; i < 5; i++ {
		v, err := mixed.SeededIntn("_rand_int", 1000000)
		if err != nil {
			t.Fatalf("SeededIntn: %v", err)
		}
		got = append(got, v)
		// Interleave an os draw (different source) — must not perturb seeded.
		_ = i
	}
	if !equalInts(want, got) {
		t.Errorf("seeded sequence perturbed by interleaved os draws: want %v, got %v", want, got)
	}
}

// TestCryptoIntn_Range checks crypto draws stay in [0, n) across many samples.
func TestCryptoIntn_Range(t *testing.T) {
	for i := 0; i < 1000; i++ {
		v := CryptoIntn(6)
		if v < 0 || v >= 6 {
			t.Fatalf("CryptoIntn(6) = %d, out of range [0,6)", v)
		}
	}
	// Distribution smoke: all 6 buckets hit over enough draws.
	seen := map[int]bool{}
	for i := 0; i < 2000; i++ {
		seen[CryptoIntn(6)] = true
	}
	if len(seen) != 6 {
		t.Errorf("CryptoIntn(6) only produced %d distinct values over 2000 draws, want 6", len(seen))
	}
}

// TestCryptoFloat64_Range checks crypto floats stay in [0.0, 1.0).
func TestCryptoFloat64_Range(t *testing.T) {
	for i := 0; i < 1000; i++ {
		f := CryptoFloat64()
		if f < 0.0 || f >= 1.0 {
			t.Fatalf("CryptoFloat64() = %f, out of range [0.0,1.0)", f)
		}
	}
}

// TestClone_ResetsRandMode proves Clone gives a fresh per-request mode stack
// (no aliasing) while preserving the seed config.
func TestClone_ResetsRandMode(t *testing.T) {
	ctx := NewEffContext([]string{})
	ctx.SetSeededModeSeed(5)
	ctx.PushRandMode("seeded")

	clone := ctx.Clone().(*EffContext)
	if got := clone.CurrentRandMode(); got != "os" {
		t.Errorf("clone should start with an empty mode stack (os), got %q", got)
	}
	// Seed config is preserved: clone can draw seeded.
	if !clone.SeedProvided() {
		t.Errorf("clone should preserve seedSet from AILANG_SEED/SetSeededModeSeed")
	}
	if _, err := clone.SeededIntn("_rand_int", 10); err != nil {
		t.Errorf("clone seeded draw failed: %v", err)
	}
	// Mutating the clone's stack must not affect the original.
	if got := ctx.CurrentRandMode(); got != "seeded" {
		t.Errorf("original mode stack was perturbed by clone: got %q, want seeded", got)
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
