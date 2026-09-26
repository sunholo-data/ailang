package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// bandGainRequantGeneric is the scalar reference and frozen source of truth for
// the SIMD-backed bandGainRequant (bandgain_simd.go). It is the per-band requant
// shared by both band-scaling passes in bands.c, each output element being
// dst[k] = PSHR32(MULT32_32_Q31(SHL32(src[k], preShift), g), postShift): the
// sample src[k] is left-shifted by preShift, multiplied by the Q31 band gain g,
// then rounded back down by postShift.
//
// The two callers differ only in which shift is the per-band variable:
//   - normaliseBands (bands.c:normalise_bands, encode): dst=X, src=freq,
//     preShift = 30-celt_zlog2(E) (per band), postShift = 30-normShift (const 6).
//   - denormaliseBands (bands.c:denormalise_bands, decode): dst=freq, src=X,
//     preShift = 30-normShift (const 6), postShift = shift (per band).
//
// MULT32_32_Q31's product is formed in int64 and is commutative, so the C
// operand order (g first in normalise_bands, the shifted sample first in
// denormalise_bands) maps onto this single g-second form without any change.
//
// Like haar1_ref.go and pitch_ref.go this file carries no build tag on purpose:
// it compiles into every build so TestBandGainRequantMatchesScalar,
// TestBandGainRequantExtremes and FuzzBandGainRequant always have the reference
// to diff bandGainRequant against, sample for sample, even in the CI jobs that
// do not build the cgo reftest oracle.
//
// Each output element is an independent expression with no accumulation across
// elements, so unlike the pitch kernels there is no reassociation argument to
// make: the SIMD path earns its bit-exactness by computing the same per-element
// expression, not by regrouping a reduction. n = min(len(dst), len(src)) matches
// i32.GainQ31's length contract exactly, so a short dst leaves the same tail
// untouched on both paths.
func bandGainRequantGeneric(dst, src []int32, g int32, preShift, postShift int) {
	n := min(len(dst), len(src))
	for k := 0; k < n; k++ {
		dst[k] = fixedmath.PSHR32(fixedmath.MULT32_32_Q31(fixedmath.SHL32(src[k], preShift), g), postShift)
	}
}
