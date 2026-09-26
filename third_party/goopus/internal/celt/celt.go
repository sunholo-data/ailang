// Shared CELT helpers ported from libopus celt/celt.c and celt/celt.h for the
// frozen build config (FIXED_POINT, DISABLE_FLOAT_API, non-CUSTOM_MODES,
// non-QEXT, non-RES24). These are the pieces the decoder pipeline in
// celt_decoder.go pulls in but that live outside celt_decoder.c in libopus:
// the post-filter (comb_filter / comb_filter_const), the small static ICDF
// tables and the tf_select table, and the SIG2WORD16 / SAT16 conversions.
//
// Source: libopus v1.6.1 (commit 3da9f7a6), celt/celt.c, celt/celt.h,
// celt/arch.h, celt/fixed_generic.h. Names and structure mirror the C so the
// port stays diffable. celt_coef is opus_val16 (int16) in this config, so the
// window and gain tables are int16 and MULT_COEF/MULT_COEF_TAPS are the
// int16-operand macro forms.

package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// Constants from celt/celt.h, celt/modes.h and celt/arch.h.
const (
	// combfilterMinperiod / combfilterMaxperiod are COMBFILTER_MINPERIOD /
	// COMBFILTER_MAXPERIOD (celt.h:236-237).
	combfilterMinperiod = 15
	combfilterMaxperiod = 1024
	// maxPeriodC is MAX_PERIOD (modes.h:40); decodeBufferSize is
	// DEC_PITCH_BUF_SIZE (modes.h:42) which DECODE_BUFFER_SIZE aliases.
	maxPeriodC       = 1024
	decodeBufferSize = 2048
	// celtLpcOrder is CELT_LPC_ORDER (celt_lpc.h): the LPC order of the
	// pitch-based PLC synthesis filter. Only the PLC (stubbed) uses it, but the
	// trailing lpc[] buffer is sized by it to match the C allocation.
	celtLpcOrder = 24

	// sigShift is SIG_SHIFT (arch.h:207); sigSat is SIG_SAT (arch.h:215), the
	// safe 32-bit signal saturation bound.
	sigShift = 12
	sigSat   = 536870911

	// spreadNormal is SPREAD_NORMAL (bands.h): the default spread decision when
	// the bitstream has no room to code one. (SPREAD_NONE lives in vq.go.)
	spreadNormal = 2

	// q15one is Q15ONE (arch.h:204): the celt_coef unit gain in this config.
	q15one = 32767
)

// Frame type tags mirror celt_decoder.c:67-72. Only NONE/NORMAL and the two PLC
// regimes are reachable in the frozen (non-DEEP_PLC) config.
const (
	frameNone        = 0
	frameNormal      = 1
	framePLCNoise    = 2
	framePLCPeriodic = 3
)

// tfSelectTable is celt/celt.c:320 const signed char tf_select_table[4][8]
// (indexed [LM][4*isTransient + 2*tf_select + tf_res]).
var tfSelectTable = [4][8]int8{
	{0, -1, 0, -1, 0, -1, 0, -1}, // 2.5 ms
	{0, -1, 0, -2, 1, 0, 1, -1},  // 5 ms
	{0, -2, 0, -3, 2, 0, 1, -1},  // 10 ms
	{0, -2, 0, -3, 3, 0, 1, -1},  // 20 ms
}

// Static ICDF tables from celt/celt.h:194-198, read by the decoder pipeline.
var (
	trimIcdf   = []byte{126, 124, 119, 109, 87, 41, 19, 9, 4, 2, 0} // trim_icdf[11]
	spreadIcdf = []byte{25, 23, 2, 0}                               // spread_icdf[4]
	tapsetIcdf = []byte{2, 1, 0}                                    // tapset_icdf[3]
)

// combGains is the post-filter tap-gain table gains[3][3] from celt.c:246-249,
// one row per tapset. QCONST16(x,15) of each float literal; the literals are all
// exact multiples of 1/32768, so the integers below equal the C constants.
var combGains = [3][3]int16{
	{10048, 7112, 4248}, // 0.3066406250, 0.2170410156, 0.1296386719
	{15200, 8784, 0},    // 0.4638671875, 0.2680664062, 0.f
	{26208, 3280, 0},    // 0.7998046875, 0.1000976562, 0.f
}

// ResamplingFactor is resampling_factor (celt.c:62): the integer ratio between
// the 48 kHz CELT mode and the rate the caller is running at. It is the ONLY
// place a sample rate enters the CELT layer: the mode is always the 48 kHz / 960
// one, and a lower rate is coded by zero-stuffing the input up to 48 kHz (the
// encoder's st->upsample, celt_encoder.c:255) and decimating the output back down
// (the decoder's st->downsample, celt_decoder.c:235).
//
// The C returns 0 for an unsupported rate, after a celt_assert(0) that is compiled
// out in the frozen (no ENABLE_ASSERTIONS) config; a 0 would then divide by zero in
// celt_preemphasis. This port returns 0 too, and every CALLER rejects it, which is
// how the "never accept a rate we cannot encode" invariant is enforced rather than
// asserted. 96000 is ENABLE_QEXT-only and is therefore NOT supported here.
func ResamplingFactor(rate int32) int {
	switch rate {
	case 48000:
		return 1
	case 24000:
		return 2
	case 16000:
		return 3
	case 12000:
		return 4
	case 8000:
		return 6
	default:
		return 0
	}
}

// sig2word16 is SIG2WORD16 / SIG2WORD16_generic (fixed_generic.h:209): shift a
// celt_sig down by SIG_SHIFT and clamp to the int16 range.
func sig2word16(x int32) int16 {
	x = fixedmath.PSHR32(x, sigShift)
	if x > 32767 {
		x = 32767
	}
	if x < -32768 {
		x = -32768
	}
	return int16(x)
}

// sat16 is SAT16 (arch.h:230): saturating narrow of an int32 to int16. Used by
// the accum path of deemphasis (ADD_RES).
func sat16(x int32) int16 {
	if x > 32767 {
		return 32767
	}
	if x < -32768 {
		return -32768
	}
	return int16(x)
}

// combFilterConst is comb_filter_const_c (celt.c:166), the constant-filter
// portion of the post-filter. y[yb..] and x[xb..] may alias the same slice with
// the same base (the decoder always applies the filter in place); negative
// x indices reach back into the decode-memory history. FIXED_POINT bias of -1
// and SIG_SAT saturation match the C exactly.
func combFilterConst(y []int32, yb int, x []int32, xb int, T, N int, g10, g11, g12 int16) {
	x4 := x[xb-T-2]
	x3 := x[xb-T-1]
	x2 := x[xb-T]
	x1 := x[xb-T+1]
	for i := 0; i < N; i++ {
		x0 := x[xb+i-T+2]
		v := x[xb+i] +
			fixedmath.MULT16_32_Q15(g10, x2) +
			fixedmath.MULT16_32_Q15(g11, x1+x3) +
			fixedmath.MULT16_32_Q15(g12, x0+x4)
		v-- // FIXED_POINT bias (celt.c:184)
		y[yb+i] = fixedmath.SATURATE(v, sigSat)
		x4 = x3
		x3 = x2
		x2 = x1
		x1 = x0
	}
}

// combFilter is comb_filter (celt.c:238) for the non-QEXT (overlap!=240) path.
// It cross-fades between the old (T0,g0,tapset0) and new (T1,g1,tapset1)
// post-filters over the first `overlap` samples, then runs the constant filter.
// y[yb..] and x[xb..] alias in the decoder; window is celt_coef (int16).
func combFilter(y []int32, yb int, x []int32, xb int, T0, T1, N int, g0, g1 int16, tapset0, tapset1, overlap int, window []int16) {
	if g0 == 0 && g1 == 0 {
		// The decoder always calls in place (y aliases x at the same base), so
		// the OPUS_MOVE is a self-copy / no-op; copy keeps the general case
		// correct without a x!=y test.
		copy(y[yb:yb+N], x[xb:xb+N])
		return
	}
	if T0 < combfilterMinperiod {
		T0 = combfilterMinperiod
	}
	if T1 < combfilterMinperiod {
		T1 = combfilterMinperiod
	}
	// MULT_COEF_TAPS(g,gains) = MULT16_16_P15 truncated back to celt_coef (int16).
	g00 := int16(fixedmath.MULT16_16_P15(g0, combGains[tapset0][0]))
	g01 := int16(fixedmath.MULT16_16_P15(g0, combGains[tapset0][1]))
	g02 := int16(fixedmath.MULT16_16_P15(g0, combGains[tapset0][2]))
	g10 := int16(fixedmath.MULT16_16_P15(g1, combGains[tapset1][0]))
	g11 := int16(fixedmath.MULT16_16_P15(g1, combGains[tapset1][1]))
	g12 := int16(fixedmath.MULT16_16_P15(g1, combGains[tapset1][2]))
	x1 := x[xb-T1+1]
	x2 := x[xb-T1]
	x3 := x[xb-T1-1]
	x4 := x[xb-T1-2]
	if g0 == g1 && T0 == T1 && tapset0 == tapset1 {
		overlap = 0
	}
	for i := 0; i < overlap; i++ {
		x0 := x[xb+i-T1+2]
		// f = MULT_COEF(window,window) truncated to celt_coef (int16).
		f := int16(fixedmath.MULT16_16_Q15(window[i], window[i]))
		oneMinusF := int16(int32(q15one) - int32(f))
		v := x[xb+i] +
			fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(oneMinusF, g00)), x[xb+i-T0]) +
			fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(oneMinusF, g01)), x[xb+i-T0+1]+x[xb+i-T0-1]) +
			fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(oneMinusF, g02)), x[xb+i-T0+2]+x[xb+i-T0-2]) +
			fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(f, g10)), x2) +
			fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(f, g11)), x1+x3) +
			fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(f, g12)), x0+x4)
		v -= 3 // FIXED_POINT bias (celt.c:294)
		y[yb+i] = fixedmath.SATURATE(v, sigSat)
		x4 = x3
		x3 = x2
		x2 = x1
		x1 = x0
	}
	if g1 == 0 {
		copy(y[yb+overlap:yb+N], x[xb+overlap:xb+N])
		return
	}
	combFilterConst(y, yb+overlap, x, xb+overlap, T1, N-overlap, g10, g11, g12)
}
