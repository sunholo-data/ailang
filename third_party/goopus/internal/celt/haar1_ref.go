package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// haar1Generic is haar1 (bands.c:haar1), the scalar reference and source of
// truth for the SIMD-backed haar1 in haar1_simd.go. It is the CELT Hadamard /
// Haar step: for each stride-wide comb the pair (evens[i], odds[i]) is scaled by
// 1/sqrt(2) in Q31 and replaced by its sum and difference.
//
// Like pitch_ref.go this file carries no build tag on purpose: it compiles into
// every build so TestHaar1SIMDMatchesScalar, TestHaar1SIMDExtremes and FuzzHaar1
// always have the reference to diff the library-backed haar1 against, sample for
// sample.
//
// Each output element is an independent MULT32_32_Q31 (a truncating 32x32 -> Q31
// multiply that wraps in int32) followed by a wrapping ADD32 / SUB32. There is
// no accumulation across elements, so unlike the pitch kernels there is no
// reassociation argument to make; the SIMD path wins its bit-exactness by
// computing the same per-element expression in the same order, not by regrouping.
//
// The loop is the comb-grouped form of the C double loop. For a given j the
// stride evens (index stride*2*j+i, i in [0,stride)) sit immediately next to the
// stride odds (index stride*(2*j+1)+i), so together they are one contiguous
// 2*stride comb at offset stride*2*j. Walking j as a shrinking window over that
// comb, and i as a plain range over the two halves, addresses every element the
// original double loop touches with the same values, just grouped by comb
// instead of by i; the two halves of a comb never depend on any other comb, so
// comb order cannot change the result. The exact-bound reslice to m*stride*2 is
// what lets bounds-check elimination prove the per-comb windows in range.
func haar1Generic(X []int32, N0, stride int) {
	m := N0 >> 1
	// stride<=0 makes the original outer loop (i<stride) empty, and m<=0 makes
	// the original inner loop (j<N0>>1) empty; either way nothing is touched.
	if stride <= 0 || m <= 0 {
		return
	}
	rest := X[:m*stride*2]
	for len(rest) >= 2*stride {
		evens := rest[:stride]
		odds := rest[stride:][:stride]
		for i := range evens {
			tmp1 := fixedmath.MULT32_32_Q31(sqrt2Inv31, evens[i])
			tmp2 := fixedmath.MULT32_32_Q31(sqrt2Inv31, odds[i])
			evens[i] = fixedmath.ADD32(tmp1, tmp2)
			odds[i] = fixedmath.SUB32(tmp1, tmp2)
		}
		rest = rest[2*stride:]
	}
}
