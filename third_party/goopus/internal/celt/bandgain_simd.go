package celt

import "github.com/tphakala/simd/i32"

// bandGainRequant is the per-band requant shared by normaliseBands (encode) and
// denormaliseBands (decode) in bands.c, the SIMD-backed form of
// bandGainRequantGeneric (bandgain_ref.go). It applies one Q31 band gain g to a
// run of samples, with an input pre-shift and a rounding output post-shift, all
// fused into a single vectorized pass by i32.GainQ31.
//
// Bit-exactness. i32.GainQ31 writes, for k in [0, min(len(dst), len(src))):
//
//	dst[k] = PSHR32(MULT32_32_Q31(SHL32(src[k], preShift), g), postShift)
//
// which is bandGainRequantGeneric's body, term for term. Every stage wraps in
// int32: SHL32 is the wrapping left shift, MULT32_32_Q31 forms the product in
// int64 and truncates the Q31 result toward -inf with no rounding, and PSHR32
// adds the round-half-up bias and arithmetic-shifts. GainQ31 documents itself
// bit-identical to that fixedmath composition across AVX2, NEON and the pure-Go
// fallback, with no relaxed tier. There is no accumulation, so lane grouping
// cannot change any result. The differential proof over the (len, g, preShift,
// postShift, input) matrix, the int32 wrapping edges and fuzz is in
// bandgain_simd_test.go.
//
// Domain. GainQ31 panics unless preShift and postShift are both in [0,31]. Both
// callers stay inside it:
//   - denormaliseBands: preShift = 30-normShift = 6 (const); postShift = shift,
//     kept in [0,30] by its own clamps (shift>=31 becomes 0 with g=0; shift<0
//     becomes 0).
//   - normaliseBands: postShift = 30-normShift = 6 (const); preShift = shift =
//     30-celt_zlog2(E), and celt_zlog2 returns 0 for E<=0 and ilog2(E) in [0,30]
//     for positive int32 E, so shift is in [0,30] for every reachable E with no
//     clamp needed.
//
// Aliasing. GainQ31 allows dst to alias src exactly element for element (each
// lane reads src[k] before its own dst[k] store) but not a shifted overlap. Both
// callers pass the identical band window of two buffers; the C source declares
// freq and X OPUS_RESTRICT in both normalise_bands and denormalise_bands, so
// real callers never overlap at all, and GainQ31's exact-alias allowance is
// strictly more permissive than what they need.
func bandGainRequant(dst, src []int32, g int32, preShift, postShift int) {
	i32.GainQ31(dst, src, g, preShift, postShift)
}
