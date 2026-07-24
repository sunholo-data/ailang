package builtins

import (
	"math/rand"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// TestRandGolden_OsSeedByteIdentical is the M5 HARD GATE for
// M-EFFECT-REPLAY-CONTRACTS: it pins the exact draw sequence produced by the os
// (global) source after rand_seed(42), proving that mode-aware dispatch did NOT
// change bare-!{Rand} + rand_seed behaviour. A bare !{Rand} program (empty mode
// stack → os mode) that calls rand_seed(42) must reproduce this golden sequence
// byte-for-byte; any drift means the os path regressed.
//
// The golden values are the math/rand.NewSource(42) Intn(1000000) sequence used
// by the global randSource. They are cross-platform stable (Go's math/rand is
// deterministic per seed and version-stable) and independent of AILANG_SEED /
// the seeded-mode source.
func TestRandGolden_OsSeedByteIdentical(t *testing.T) {
	ctx := randCtx()
	if ctx.CurrentRandMode() != "os" {
		t.Fatalf("golden must run in os mode, got %q", ctx.CurrentRandMode())
	}

	// Seed the os/global source exactly as std/rand.rand_seed(42) does.
	SetRandSeed(42)

	// The pinned golden os sequence for seed=42, Intn(1000000).
	want := goldenOsSeed42Sequence(t)

	got := make([]int, len(want))
	for i := range got {
		v, err := randIntImpl(ctx, intArgs(0, 999999)) // span 1_000_000 → Intn(1000000)
		if err != nil {
			t.Fatalf("randIntImpl(os): %v", err)
		}
		got[i] = v.(*eval.IntValue).Value
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("os-source golden drift at draw %d: got %d, want %d (full got=%v)\n"+
				"bare-!{Rand} + rand_seed behaviour changed — mode-aware dispatch must leave os untouched",
				i, got[i], want[i], got)
		}
	}
}

// goldenOsSeed42Sequence computes the reference sequence from a fresh
// math/rand source with seed 42, matching what the os path must produce. Using
// the stdlib source directly (rather than hardcoding integers) keeps the golden
// pinned to Go's deterministic PRNG contract while still proving the AILANG os
// path routes to that exact source unchanged. It is re-derived from a SECOND,
// independent source so a bug that perturbs the global randSource cannot also
// perturb the reference.
func goldenOsSeed42Sequence(t *testing.T) []int {
	t.Helper()
	// A dedicated reference source, seeded identically, NOT the global one under
	// test. If the os path is unchanged, randIntImpl draws will match these.
	ref := rand.New(rand.NewSource(42))
	out := make([]int, 8)
	for i := range out {
		out[i] = ref.Intn(1000000)
	}
	return out
}
