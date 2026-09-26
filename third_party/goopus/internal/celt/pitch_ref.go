// Scalar reference implementations of the two pitch kernels that are backed by
// SIMD (celt_inner_prod and xcorr_kernel). These are the transliteration of the
// C generic path and they are the source of truth: the library-backed
// celtInnerProd / xcorrKernel in pitch_simd.go (github.com/tphakala/simd/i16,
// i16.DotProduct and i16.XCorr) are verified against them, sample for sample, by
// TestCeltInnerProdSIMDMatchesScalar / TestXcorrKernelSIMDMatchesScalar and
// FuzzCeltInnerProd in pitch_simd_test.go.
//
// This file carries no build tag on purpose. It compiles into *every* build, so
// the differential test always has the reference to compare the library-backed
// kernels against.
//
// Why reordering the accumulation is legal here, and why that is the whole basis
// for vectorizing these two kernels at all: both accumulate via MAC16_16, i.e.
// int16 x int16 widened to an exact int32 product (|product| <= 2^30, so the
// multiply itself never overflows) accumulated into a *wrapping* int32. Go
// defines signed integer overflow as two's-complement wraparound, and
// wraparound addition over int32 is associative and commutative -- it is the
// abelian group Z/2^32 Z. So any lane grouping, any partial-sum tree, and any
// horizontal-reduction order yields the bit-identical int32, including in the
// cases that overflow. libopus relies on exactly this property for its own SIMD
// (celt/x86/pitch_sse2.c, celt/arm/pitch_neon_intr.c both reassociate freely and
// assert bit-exactness against the C under OPUS_CHECK_ASM).

package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// celtInnerProdGeneric is celt_inner_prod_c (pitch.h:159):
// sum_i x[xOff+i]*y[yOff+i], accumulated in a wrapping int32.
func celtInnerProdGeneric(x []int16, xOff int, y []int16, yOff, N int) int32 {
	xy := int32(0)
	for i := 0; i < N; i++ {
		xy = fixedmath.MAC16_16(xy, x[xOff+i], y[yOff+i])
	}
	return xy
}

// xcorrKernelGeneric is xcorr_kernel_c (pitch.h:65), the kernel celt_pitch_xcorr
// is built out of: sum[k] += sum_j x[j]*y[k+j] for the four lags k in [0,4) at
// once. Each x sample is loaded once and fed to all four accumulators, and the
// four y samples it meets are held in a rotating y0..y3 register window, so the
// 4x4 block costs one x load and one y load per multiply-accumulate column
// instead of two. The four accumulators are also four independent dependency
// chains.
//
// Preconditions (celt_assert(len>=3) in C): len_ >= 3, len(x) >= len_, and
// len(y) >= len_+3 -- the window runs three samples past x, exactly as the C
// pointer walk does.
//
// The four accumulators are held in locals s0..s3 and written back to sum once, at the
// end. C gets this for free -- xcorr_kernel_c is static OPUS_INLINE, so gcc scalar-
// replaces the sum[4] array into registers -- but Go cannot prove *[4]int32 does not
// alias the []int16 operands, so through the pointer every MAC16_16 would reload and
// respill its accumulator: 32 memory ops per 4x4 block instead of none. The arithmetic
// and its order are untouched; only where the running totals live changes.
//
// C walks x and y with post-increment pointers. Here x[j] is C's *x++ (j and the x
// cursor advance in lockstep), and yw is C's y cursor after the three preloading
// *y++: every later `*y++` is yw[j] at the same j. Both slices are cut to their exact
// extent up front, which is what lets the compiler drop every bounds check from the
// unrolled loop below (verified with -d=ssa/check_bce/debug=1).
func xcorrKernelGeneric(x, y []int16, sum *[4]int32, len_ int) {
	var j int
	// C's `y_3=0` (it is always written before it is read; the store is only there to
	// quiet gcc) is Go's zero value.
	var y0, y1, y2, y3 int16
	s0, s1, s2, s3 := sum[0], sum[1], sum[2], sum[3]
	x = x[:len_]
	y0 = y[0]
	y1 = y[1]
	y2 = y[2]
	yw := y[3 : 3+len(x)]
	for j = 0; j < len(x)-3; j += 4 {
		var tmp int16
		tmp = x[j]
		y3 = yw[j]
		s0 = fixedmath.MAC16_16(s0, tmp, y0)
		s1 = fixedmath.MAC16_16(s1, tmp, y1)
		s2 = fixedmath.MAC16_16(s2, tmp, y2)
		s3 = fixedmath.MAC16_16(s3, tmp, y3)
		tmp = x[j+1]
		y0 = yw[j+1]
		s0 = fixedmath.MAC16_16(s0, tmp, y1)
		s1 = fixedmath.MAC16_16(s1, tmp, y2)
		s2 = fixedmath.MAC16_16(s2, tmp, y3)
		s3 = fixedmath.MAC16_16(s3, tmp, y0)
		tmp = x[j+2]
		y1 = yw[j+2]
		s0 = fixedmath.MAC16_16(s0, tmp, y2)
		s1 = fixedmath.MAC16_16(s1, tmp, y3)
		s2 = fixedmath.MAC16_16(s2, tmp, y0)
		s3 = fixedmath.MAC16_16(s3, tmp, y1)
		tmp = x[j+3]
		y2 = yw[j+3]
		s0 = fixedmath.MAC16_16(s0, tmp, y3)
		s1 = fixedmath.MAC16_16(s1, tmp, y0)
		s2 = fixedmath.MAC16_16(s2, tmp, y1)
		s3 = fixedmath.MAC16_16(s3, tmp, y2)
	}
	// C's three `if (j++<len)` tail steps. They are not symmetric: each consumes one
	// more x sample and rotates the y window on by one, so the sum[k] each x sample
	// lands on shifts by one every step.
	if j < len(x) { // if (j++<len)
		tmp := x[j]
		y3 = yw[j]
		s0 = fixedmath.MAC16_16(s0, tmp, y0)
		s1 = fixedmath.MAC16_16(s1, tmp, y1)
		s2 = fixedmath.MAC16_16(s2, tmp, y2)
		s3 = fixedmath.MAC16_16(s3, tmp, y3)
	}
	j++
	if j < len(x) { // if (j++<len)
		tmp := x[j]
		y0 = yw[j]
		s0 = fixedmath.MAC16_16(s0, tmp, y1)
		s1 = fixedmath.MAC16_16(s1, tmp, y2)
		s2 = fixedmath.MAC16_16(s2, tmp, y3)
		s3 = fixedmath.MAC16_16(s3, tmp, y0)
	}
	j++
	if j < len(x) { // if (j<len)
		tmp := x[j]
		y1 = yw[j]
		s0 = fixedmath.MAC16_16(s0, tmp, y2)
		s1 = fixedmath.MAC16_16(s1, tmp, y3)
		s2 = fixedmath.MAC16_16(s2, tmp, y0)
		s3 = fixedmath.MAC16_16(s3, tmp, y1)
	}
	sum[0], sum[1], sum[2], sum[3] = s0, s1, s2, s3
}
