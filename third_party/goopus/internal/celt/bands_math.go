// Leaf math helpers and constants that celt/bands.c (the quant_all_bands
// machinery in bands.go) needs but that are provided as macros / other-TU
// functions in libopus (v1.6.1, commit 3da9f7a6), FIXED_POINT +
// DISABLE_FLOAT_API, non-QEXT. Kept in a separate file from the band quantizer
// itself so bands.go stays a close mirror of bands.c.
//
// celt_exp2_db / celt_exp2_db_frac live in celt/mathops.h (the non-QEXT
// FIXED_POINT macro forms); celt_inner_prod_norm_shift lives in celt/vq.c;
// FRAC_MUL16 in celt/mathops.h; DIV32_16 in celt/fixed_generic.h; eMeans in
// celt/quant_bands.c. They are transliterated here because the port only needs
// them for bands.go and the frozen fixedmath package does not expose them yet.

package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// Constants from celt/arch.h and celt/rate.h.
const (
	// q31one is Q31ONE (arch.h:205): the Q31 unit gain passed as the band gain.
	q31one int32 = 2147483647
	// normScaling is NORM_SCALING = 1<<NORM_SHIFT (arch.h:217): a unit-norm
	// celt_norm sample.
	normScaling int32 = 1 << normShift
	// qthetaOffset / qthetaOffsetTwophase are QTHETA_OFFSET / _TWOPHASE
	// (rate.h:40-41): the theta-resolution offsets used by compute_theta.
	qthetaOffset         = 4
	qthetaOffsetTwophase = 16
	// spreadAggressive is SPREAD_AGGRESSIVE (bands.h:71); the other SPREAD_*
	// values (SPREAD_NONE) live in vq.go.
	spreadAggressive = 3
)

// gconst16 is GCONST(16.f) = QCONST32(16, DB_SHIFT) (fixed_generic.h:101): the
// anti_collapse Ediff cutoff, in Q(DB_SHIFT).
var gconst16 = fixedmath.QCONST32(16, dbShift)

// qconst32Float mirrors QCONST32(x,bits) (fixed_generic.h:95) when the C source
// writes x as a *float* literal (an "f" suffix): the multiply is done in float32
// precision, then rounded to int32. Evaluated at package init (a constant
// float-to-int truncation is not permitted in a Go const expression).
func qconst32Float(x float32, bits int) int32 {
	return int32(0.5 + float64(x*float32(int64(1)<<bits)))
}

// sqrt2Inv31 is QCONST32(.70710678f, 31): the 1/sqrt(2) Hadamard/haar1 rotation
// constant (bands.c:631). The C source uses a *float* literal (.70710678f), so
// the constant is rounded in float32 precision (= 1518500224). fixedmath.QCONST32
// evaluates in float64 (= 1518500247, 23 ULPs high), which after haar1's
// MULT32_32_Q31 shows up as +/-1 errors in the decoded spectrum, so it is
// computed in float32 to match the C literal exactly.
var sqrt2Inv31 = qconst32Float(0.70710678, 31)

// eMeans is the FIXED_POINT const signed char eMeans[25] (quant_bands.c:44): the
// mean energy in each band, quantized in Q4. denormalise_bands shifts it into
// Q(DB_SHIFT) with DB_SHIFT-4.
var eMeans = [25]int8{
	103, 100, 92, 85, 81,
	77, 72, 70, 78, 75,
	73, 71, 78, 74, 69,
	72, 70, 74, 76, 71,
	60, 60, 60, 60, 60,
}

// fracMul16 is FRAC_MUL16(a,b) = ((16384+((opus_int32)(opus_int16)(a)*
// (opus_int16)(b)))>>15) (mathops.h:50). Both operands are truncated to
// opus_int16 first, exactly as the macro's casts require.
func fracMul16(a, b int32) int32 {
	return (16384 + int32(int16(a))*int32(int16(b))) >> 15
}

// div32_16 is DIV32_16(a,b) = ((opus_val16)(((opus_val32)(a))/((opus_val16)(b))))
// (fixed_generic.h:200): int32/int16 truncating division, narrowed to int16.
// Delegates to fixedmath.DIV32_16 so there is a single implementation to keep
// bit-exact.
func div32_16(a int32, b int16) int16 {
	return fixedmath.DIV32_16(a, b)
}

// celtExp2DbFrac is the non-QEXT FIXED_POINT macro celt_exp2_db_frac(x) =
// SHL32(celt_exp2_frac(PSHR32(x, DB_SHIFT-10)), 14) (mathops.h:530). The
// PSHR32 result is passed to celt_exp2_frac as opus_val16, so it is truncated
// with EXTRACT16 first.
func celtExp2DbFrac(x int32) int32 {
	return fixedmath.SHL32(fixedmath.Celt_exp2_frac(fixedmath.EXTRACT16(fixedmath.PSHR32(x, dbShift-10))), 14)
}

// celtExp2Db is the non-QEXT FIXED_POINT macro celt_exp2_db(x) =
// celt_exp2(PSHR32(x, DB_SHIFT-10)) (mathops.h:531). The PSHR32 result is
// truncated to opus_val16 before celt_exp2, matching the macro's implicit cast.
func celtExp2Db(x int32) int32 {
	return fixedmath.Celt_exp2(fixedmath.EXTRACT16(fixedmath.PSHR32(x, dbShift-10)))
}

// celtInnerProdNormShift is the FIXED_POINT celt_inner_prod_norm_shift (vq.c:65):
// a 64-bit accumulation of x[i]*y[i] over the celt_norm samples, right-shifted by
// 2*(NORM_SHIFT-14) and returned as opus_val32.
//
// x and y are re-sliced to length first so the loop below walks them by range
// instead of an independently bounds-checked index; that gives BCE the two
// one-time slice-header checks instead of two per iteration.
func celtInnerProdNormShift(x, y []int32, length int) int32 {
	// The pre-BCE loop was a silent no-op for length <= 0 (as is the C, whose
	// for loop never runs); the re-slices below would panic instead. No current
	// caller passes a non-positive length, but the guard keeps the degenerate
	// domain identical to the original, matching the guards on expRotation1 and
	// haar1 from the same restructuring.
	if length <= 0 {
		return 0
	}
	x = x[:length]
	y = y[:length]
	var sum int64
	for i := range x {
		sum += int64(x[i]) * int64(y[i])
	}
	return int32(sum >> (2 * (normShift - 14)))
}
