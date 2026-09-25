package celt

import (
	"github.com/tphakala/simd/i32"

	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
)

// minHaar1ButterflyStride is the comb width at and above which haar1's add/sub
// butterfly routes through i32.Butterfly (via haar1WideButterfly); narrower
// combs keep the inlined scalar loop. The crossover is set from a clean pinned
// benchstat on both arches, not from the library's own SIMD-activation length:
// i32.Butterfly first beats the inlined scalar butterfly here only once a comb
// spans two or more vector blocks. At stride 16 (two AVX2 int32 blocks on amd64,
// four NEON blocks on arm64) it wins clearly, roughly -17% on amd64 and -19% on
// arm64. At stride 8 (one AVX2 block) the per-comb call plus a single block is
// break-even to slightly negative on amd64 (about +3%) and only a marginal win
// on arm64 (about -2%), so 8 does not carry its weight as a single
// arch-independent gate and stays scalar. Production reaches strides 16, 32 and
// beyond through the time-resolution recombine loops (haar1(X, N_B, B) with B
// doubling, and haar1(X, N0>>k, 1<<k)), so the wide-stride path serves real
// calls. Strides below the gate run byte-identical scalar code; any movement on
// them in a microbenchmark is pure code-alignment noise, not a real change.
const minHaar1ButterflyStride = 16

// haar1 is the CELT Hadamard/Haar step (bands.c:623), the SIMD-backed form of
// haar1Generic (haar1_ref.go). The per-element 1/sqrt(2) scale is the expensive
// part (a 32x32 -> Q31 multiply run twice per pair); it is hoisted out of the
// comb loop into a single vectorized i32.ScaleQ31 pass over the whole touched
// region. The wrapping add/sub butterfly then runs per comb through i32.Butterfly
// when the comb is wide enough to reach the vector kernel
// (stride >= minHaar1ButterflyStride) and stays a scalar inlined loop below that.
//
// Bit-exactness. i32.ScaleQ31 computes int32(int64(a[i]) * int64(k) >> 31), the
// integer MULT32_32_Q31: a truncating (arithmetic, no rounding) Q31 multiply
// that wraps in int32, identical to fixedmath.MULT32_32_Q31 for every input (the
// multiply is commutative, so scaling by sqrt2Inv31 on either operand is the
// same value). ScaleQ31 documents exact element-for-element aliasing, so the
// in-place dst==src scale is defined. Scaling every touched element up front and
// combining afterwards yields the same values as haar1Generic's interleaved
// order, because the scale is elementwise and carries no dependency into the
// add/sub. i32.Butterfly writes lo[i], hi[i] = lo[i]+hi[i], lo[i]-hi[i] with
// two's-complement wrapping int32 adds and subtracts, exactly ADD32/SUB32 per
// lane with no saturation, reading both lanes before writing either; evens and
// odds are the disjoint halves [0,stride) and [stride,2*stride) of one comb, so
// they satisfy i32.Butterfly's no-overlap contract and the stride gate selects
// between two implementations of the same arithmetic (a codegen choice, never a
// value change). The differential proof over the (N0, stride) matrix, the int32
// wrapping edges, and fuzz is TestHaar1SIMDMatchesScalar / TestHaar1SIMDExtremes
// / FuzzHaar1 in haar1_simd_test.go.
func haar1(X []int32, N0, stride int) {
	m := N0 >> 1
	// stride<=0 or m<=0 makes the original double loop empty; touch nothing.
	if stride <= 0 || m <= 0 {
		return
	}
	// haar1 only ever reaches the first m*stride*2 elements of X (m combs, each
	// 2*stride wide); the exact-bound reslice is also what ScaleQ31 scales.
	rest := X[:m*stride*2]
	// One contiguous pass scales every even and odd sample by sqrt2Inv31 in
	// place, amortizing the one expensive op (the Q31 multiply) across the
	// buffer and reaching the vector kernel once.
	i32.ScaleQ31(rest, rest, sqrt2Inv31)
	if stride >= minHaar1ButterflyStride {
		// Wide combs go through a separate helper so this hot dispatcher stays
		// small: inlining the vector loop here bloats haar1 and shifts the
		// narrow-stride loop below to a worse code alignment, measurably
		// perturbing the common strides (1, 2, 4, 8) that dominate production.
		haar1WideButterfly(rest, stride)
		return
	}
	// Narrow combs (production strides 1, 2, 4, 8): the per-comb i32.Butterfly
	// path does not beat this inlined loop until stride 16 (see
	// minHaar1ButterflyStride), so everything below the gate stays scalar. The
	// scale is the ALU-bound cost; the add/sub is L1-bound and best left inline.
	for len(rest) >= 2*stride {
		evens := rest[:stride]
		odds := rest[stride:][:stride]
		for i := range evens {
			t1 := evens[i]
			t2 := odds[i]
			evens[i] = fixedmath.ADD32(t1, t2)
			odds[i] = fixedmath.SUB32(t1, t2)
		}
		rest = rest[2*stride:]
	}
}

// haar1WideButterfly runs the add/sub butterfly through i32.Butterfly once per
// comb over the already-scaled region rest, for strides at or above
// minHaar1ButterflyStride. It is kept out of haar1's body (and out of line) so
// haar1's narrow-stride path keeps its original small-function code layout; the
// per-comb halves rest[:stride] and rest[stride:][:stride] are the disjoint
// [0,stride) and [stride,2*stride) windows Butterfly requires, and its wrapping
// (lo+hi, lo-hi) is exactly the scalar ADD32/SUB32 lane for lane.
//
//go:noinline
func haar1WideButterfly(rest []int32, stride int) {
	for len(rest) >= 2*stride {
		i32.Butterfly(rest[:stride], rest[stride:][:stride])
		rest = rest[2*stride:]
	}
}
