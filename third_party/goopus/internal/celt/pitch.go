// Pitch analysis primitives ported from libopus celt/pitch.c, the decoder subset
// used by the CELT packet-loss concealment: pitch_downsample (48 kHz -> LP,
// decimate by `factor`, whiten with an order-4 LPC), pitch_search (coarse 4x then
// fine 2x cross-correlation search with pseudo-interpolation), celt_pitch_xcorr_c
// (the scalar reference cross-correlation), find_best_pitch, celt_fir5 and
// celt_inner_prod. The encoder (phase 4) additionally needs remove_doubling plus
// its dual_inner_prod / compute_pitch_gain helpers, ported at the bottom of this
// file; the decoder never reached them (celt_plc_pitch_search calls only
// pitch_downsample + pitch_search).
//
// These are pure functions (docs/hard-parts.md section 8); their bit-exactness
// against celt/pitch.c is proven by the function-level differential tests in
// internal/reftest/oracle (plc_test.go). Type mapping (celt/arch.h):
// opus_val16 = int16, opus_val32/celt_sig = int32.
//
// celt_inner_prod and xcorr_kernel are the two hot kernels. They are backed by
// github.com/tphakala/simd/i16 (DotProduct and XCorr) in pitch_simd.go; the
// scalar reference lives in pitch_ref.go and is proven bit-identical to the
// library path in pitch_simd_test.go. See pitch_ref.go for why reassociating a
// wrapping-int32 accumulation is exact, which is what lets the library's
// vectorized horizontal sum match the scalar loop for every input.
//
// celt_pitch_xcorr itself (pitch.c:230) stays in Go: it is just the driver loop
// around xcorr_kernel and the work is all inside the kernel.

package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// PLC pitch-lag search bounds (celt_decoder.c:62-65). PLC_PITCH_LAG_MAX (720)
// corresponds to a 66.67 Hz pitch, PLC_PITCH_LAG_MIN (100) to 480 Hz.
const (
	plcPitchLagMax = 720
	plcPitchLagMin = 100
)

// pitchIabs is abs() for the small int lag arithmetic in pitch_search.
func pitchIabs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// celtPitchXcorr is celt_pitch_xcorr_c (pitch.c:230) for FIXED_POINT, the live
// (unrolled) branch: fill xcorr[i] = sum_j x[j]*y[i+j] for i in [0,maxPitch) and
// return the running max (>=1). Lags are done four at a time through xcorrKernel,
// with a scalar tail when maxPitch is not a multiple of 4. x and y are indexed from
// offset 0.
//
// The blocked loop reads y up to index (maxPitch-4)+len_+2 == maxPitch+len_-2, which
// is exactly the last index the scalar form reads, so it needs no extra slack: as in
// C, len(y) >= len_+maxPitch-1 suffices.
func celtPitchXcorr(x, y []int16, xcorr []int32, len_, maxPitch int) int32 {
	var i int
	maxcorr := int32(1)
	// celt_assert(max_pitch>0)
	for i = 0; i < maxPitch-3; i += 4 {
		sum := [4]int32{0, 0, 0, 0}
		xcorrKernel(x, y[i:], &sum, len_)
		xcorr[i] = sum[0]
		xcorr[i+1] = sum[1]
		xcorr[i+2] = sum[2]
		xcorr[i+3] = sum[3]
		sum[0] = fixedmath.MAX32(sum[0], sum[1])
		sum[2] = fixedmath.MAX32(sum[2], sum[3])
		sum[0] = fixedmath.MAX32(sum[0], sum[2])
		maxcorr = fixedmath.MAX32(maxcorr, sum[0])
	}
	// In case maxPitch isn't a multiple of 4, do the non-unrolled version.
	for ; i < maxPitch; i++ {
		sum := celtInnerProd(x, 0, y, i, len_)
		xcorr[i] = sum
		maxcorr = fixedmath.MAX32(maxcorr, sum)
	}
	return maxcorr
}

// findBestPitch is find_best_pitch (pitch.c:45) for FIXED_POINT: pick the two
// lags maximising xcorr[i]^2 / Syy, where Syy is the sliding energy of y over a
// window of length len_. bestPitch receives the top-two lags. y is indexed from
// yOff.
func findBestPitch(xcorr []int32, y []int16, yOff, len_, maxPitch int, bestPitch []int, yshift int, maxcorr int32) {
	Syy := int32(1)
	var bestNum [2]int16
	var bestDen [2]int32
	xshift := fixedmath.Celt_ilog2(maxcorr) - 14

	bestNum[0] = -1
	bestNum[1] = -1
	bestDen[0] = 0
	bestDen[1] = 0
	bestPitch[0] = 0
	bestPitch[1] = 1
	for j := 0; j < len_; j++ {
		Syy = fixedmath.ADD32(Syy, fixedmath.SHR32(fixedmath.MULT16_16(y[yOff+j], y[yOff+j]), yshift))
	}
	for i := 0; i < maxPitch; i++ {
		if xcorr[i] > 0 {
			xcorr16 := fixedmath.EXTRACT16(fixedmath.VSHR32(xcorr[i], xshift))
			num := fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(xcorr16, xcorr16))
			if fixedmath.MULT16_32_Q15(num, bestDen[1]) > fixedmath.MULT16_32_Q15(bestNum[1], Syy) {
				if fixedmath.MULT16_32_Q15(num, bestDen[0]) > fixedmath.MULT16_32_Q15(bestNum[0], Syy) {
					bestNum[1] = bestNum[0]
					bestDen[1] = bestDen[0]
					bestPitch[1] = bestPitch[0]
					bestNum[0] = num
					bestDen[0] = Syy
					bestPitch[0] = i
				} else {
					bestNum[1] = num
					bestDen[1] = Syy
					bestPitch[1] = i
				}
			}
		}
		Syy += fixedmath.SHR32(fixedmath.MULT16_16(y[yOff+i+len_], y[yOff+i+len_]), yshift) -
			fixedmath.SHR32(fixedmath.MULT16_16(y[yOff+i], y[yOff+i]), yshift)
		Syy = fixedmath.MAX32(1, Syy)
	}
}

// celtFir5 is celt_fir5 (pitch.c:105): an in-place order-5 FIR whitening filter
// with a SROUND16/ROUND16 output. mem holds the last five input samples.
func celtFir5(x []int16, num []int16, N int) {
	num0 := num[0]
	num1 := num[1]
	num2 := num[2]
	num3 := num[3]
	num4 := num[4]
	var mem0, mem1, mem2, mem3, mem4 int16
	for i := 0; i < N; i++ {
		sum := fixedmath.SHL32(fixedmath.EXTEND32(x[i]), sigShift)
		sum = fixedmath.MAC16_16(sum, num0, mem0)
		sum = fixedmath.MAC16_16(sum, num1, mem1)
		sum = fixedmath.MAC16_16(sum, num2, mem2)
		sum = fixedmath.MAC16_16(sum, num3, mem3)
		sum = fixedmath.MAC16_16(sum, num4, mem4)
		mem4 = mem3
		mem3 = mem2
		mem2 = mem1
		mem1 = mem0
		mem0 = x[i]
		x[i] = fixedmath.ROUND16(sum, sigShift)
	}
}

// pitchDownsample is pitch_downsample (pitch.c:140) for FIXED_POINT: downmix the
// C channels of x (48 kHz celt_sig) into the half-rate int16 x_lp with a [.25 .5
// .25] kernel, then whiten with an order-4 LPC (lag-windowed autocorrelation ->
// _celt_lpc -> celt_fir5). x[c] are the per-channel signal buffers; len_ is the
// output length; factor is the decimation factor (2 for the PLC).
func pitchDownsample(x [][]int32, xLp []int16, len_, C, factor int, sc *scratch) {
	var ac [5]int32
	tmp := int16(q15one)
	var lpc [4]int16
	var lpc2 [5]int16
	c1 := fixedmath.QCONST16(0.8, 15)
	offset := factor / 2

	maxabs := celtMaxabs32(x[0], 0, len_*factor)
	if C == 2 {
		maxabs1 := celtMaxabs32(x[1], 0, len_*factor)
		maxabs = fixedmath.MAX32(maxabs, maxabs1)
	}
	if maxabs < 1 {
		maxabs = 1
	}
	shift := fixedmath.Celt_ilog2(maxabs) - 10
	if shift < 0 {
		shift = 0
	}
	if C == 2 {
		shift++
	}
	for i := 1; i < len_; i++ {
		xLp[i] = fixedmath.EXTRACT16(fixedmath.SHR32(x[0][factor*i-offset], shift+2) +
			fixedmath.SHR32(x[0][factor*i+offset], shift+2) +
			fixedmath.SHR32(x[0][factor*i], shift+1))
	}
	xLp[0] = fixedmath.EXTRACT16(fixedmath.SHR32(x[0][offset], shift+2) + fixedmath.SHR32(x[0][0], shift+1))
	if C == 2 {
		for i := 1; i < len_; i++ {
			xLp[i] = fixedmath.EXTRACT16(fixedmath.EXTEND32(xLp[i]) +
				fixedmath.SHR32(x[1][factor*i-offset], shift+2) +
				fixedmath.SHR32(x[1][factor*i+offset], shift+2) +
				fixedmath.SHR32(x[1][factor*i], shift+1))
		}
		xLp[0] = fixedmath.EXTRACT16(fixedmath.EXTEND32(xLp[0]) +
			fixedmath.SHR32(x[1][offset], shift+2) + fixedmath.SHR32(x[1][0], shift+1))
	}

	celtAutocorr(xLp, ac[:], nil, 0, 4, len_, sc)

	// Noise floor -40 dB.
	ac[0] += fixedmath.SHR32(ac[0], 13)
	// Lag windowing.
	for i := 1; i <= 4; i++ {
		ac[i] -= fixedmath.MULT16_32_Q15(int16(2*i*i), ac[i])
	}

	celtLPC(lpc[:], ac[:], 4)
	for i := 0; i < 4; i++ {
		tmp = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.9, 15), tmp))
		lpc[i] = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(lpc[i], tmp))
	}
	// Add a zero.
	lpc2[0] = lpc[0] + fixedmath.QCONST16(0.8, sigShift)
	lpc2[1] = lpc[1] + fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(c1, lpc[0]))
	lpc2[2] = lpc[2] + fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(c1, lpc[1]))
	lpc2[3] = lpc[3] + fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(c1, lpc[2]))
	lpc2[4] = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(c1, lpc[3]))
	celtFir5(xLp, lpc2[:], len_)
}

// pitchSearch is pitch_search (pitch.c:307) for FIXED_POINT: a coarse cross-
// correlation search at 4x decimation, a finer search at 2x decimation restricted
// to the coarse candidates, then pseudo-interpolation, writing the estimated
// pitch lag to *pitch. xLp is the whitened analysis signal (length >= len_>>1);
// y is the history it is correlated against (length >= lag>>1). len_ and maxPitch
// are the full-rate search parameters.
func pitchSearch(xLp []int16, y []int16, len_, maxPitch int, pitch *int, sc *scratch) {
	bestPitch := [2]int{0, 0}
	var offset int

	lag := len_ + maxPitch

	xLp4 := alloc(&sc.xLp4, len_>>2)            // VARDECL(opus_val16, x_lp4)
	yLp4 := alloc(&sc.yLp4, lag>>2)             // VARDECL(opus_val16, y_lp4)
	xcorr := alloc(&sc.pitchXcorr, maxPitch>>1) // VARDECL(opus_val32, xcorr)

	// Downsample by 2 again.
	for j := 0; j < len_>>2; j++ {
		xLp4[j] = xLp[2*j]
	}
	for j := 0; j < lag>>2; j++ {
		yLp4[j] = y[2*j]
	}

	xmax := celtMaxabs16(xLp4, len_>>2)
	ymax := celtMaxabs16(yLp4, lag>>2)
	shift := fixedmath.Celt_ilog2(fixedmath.MAX32(1, fixedmath.MAX32(xmax, ymax))) - 14 + fixedmath.Celt_ilog2(int32(len_))/2
	if shift > 0 {
		for j := 0; j < len_>>2; j++ {
			xLp4[j] = fixedmath.SHR16(xLp4[j], shift)
		}
		for j := 0; j < lag>>2; j++ {
			yLp4[j] = fixedmath.SHR16(yLp4[j], shift)
		}
		// Use double the shift for a MAC.
		shift *= 2
	} else {
		shift = 0
	}

	// Coarse search with 4x decimation.
	maxcorr := celtPitchXcorr(xLp4, yLp4, xcorr, len_>>2, maxPitch>>2)
	findBestPitch(xcorr, yLp4, 0, len_>>2, maxPitch>>2, bestPitch[:], 0, maxcorr)

	// Finer search with 2x decimation.
	maxcorr = 1
	for i := 0; i < maxPitch>>1; i++ {
		xcorr[i] = 0
		if pitchIabs(i-2*bestPitch[0]) > 2 && pitchIabs(i-2*bestPitch[1]) > 2 {
			continue
		}
		sum := int32(0)
		for j := 0; j < len_>>1; j++ {
			sum += fixedmath.SHR32(fixedmath.MULT16_16(xLp[j], y[i+j]), shift)
		}
		xcorr[i] = fixedmath.MAX32(-1, sum)
		maxcorr = fixedmath.MAX32(maxcorr, sum)
	}
	findBestPitch(xcorr, y, 0, len_>>1, maxPitch>>1, bestPitch[:], shift+1, maxcorr)

	// Refine by pseudo-interpolation.
	if bestPitch[0] > 0 && bestPitch[0] < (maxPitch>>1)-1 {
		a := xcorr[bestPitch[0]-1]
		b := xcorr[bestPitch[0]]
		c := xcorr[bestPitch[0]+1]
		if (c - a) > fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.7, 15), b-a) {
			offset = 1
		} else if (a - c) > fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.7, 15), b-c) {
			offset = -1
		} else {
			offset = 0
		}
	} else {
		offset = 0
	}
	*pitch = 2*bestPitch[0] - offset
}

// dualInnerProd is dual_inner_prod_c (pitch.h:137): computes both
// sum_i x[xb+i]*y01[y01b+i] and sum_i x[xb+i]*y02[y02b+i] over the shared int16
// buffer in a single pass. run_prefilter's remove_doubling correlates a signal
// against two lags at once, so this halves the multiplies.
func dualInnerProd(x []int16, xb, y01b, y02b, N int) (xy1, xy2 int32) {
	var xy01, xy02 int32
	for i := 0; i < N; i++ {
		xy01 = fixedmath.MAC16_16(xy01, x[xb+i], x[y01b+i])
		xy02 = fixedmath.MAC16_16(xy02, x[xb+i], x[y02b+i])
	}
	return xy01, xy02
}

// computePitchGain is compute_pitch_gain (pitch.c:419) for FIXED_POINT: the
// normalized correlation xy / sqrt(xx*yy) clamped to [-1, 1] in Q15, computed with
// per-operand ilog2 normalization and celt_rsqrt_norm to keep the intermediates in
// range.
func computePitchGain(xy, xx, yy int32) int16 {
	if xy == 0 || xx == 0 || yy == 0 {
		return 0
	}
	sx := fixedmath.Celt_ilog2(xx) - 14
	sy := fixedmath.Celt_ilog2(yy) - 14
	shift := sx + sy
	x2y2 := fixedmath.SHR32(fixedmath.MULT16_16(
		int16(fixedmath.VSHR32(xx, sx)), int16(fixedmath.VSHR32(yy, sy))), 14)
	if shift&1 != 0 {
		if x2y2 < 32768 {
			x2y2 <<= 1
			shift--
		} else {
			x2y2 >>= 1
			shift++
		}
	}
	den := fixedmath.Celt_rsqrt_norm(x2y2)
	g := fixedmath.MULT16_32_Q15(den, xy)
	g = fixedmath.VSHR32(g, (shift>>1)-1)
	return fixedmath.EXTRACT16(fixedmath.MAX32(-int32(q15one), fixedmath.MIN32(g, int32(q15one))))
}

// secondCheck is second_check[16] (pitch.c:453): the alternate divisor used when
// remove_doubling probes for a strong sub-harmonic correlation.
var secondCheck = [16]int{0, 0, 3, 2, 3, 2, 5, 2, 3, 2, 3, 2, 5, 2, 3, 2}

// removeDoubling is remove_doubling (pitch.c:454) for FIXED_POINT: given the
// coarse pitch estimate *T0, look for a stronger correlation at a sub-multiple of
// the period (pitch doubling/tripling avoidance), biased against very short
// periods and toward continuity with the previous frame (prevPeriod/prevGain). It
// refines *T0 by pseudo-interpolation and returns the pitch gain in Q15. x is the
// half-rate whitened analysis buffer produced by pitch_downsample; it is advanced
// by maxperiod internally so negative indices reach the pitch history.
func removeDoubling(x []int16, maxperiod, minperiod, N int, T0 *int, prevPeriod int, prevGain int16, sc *scratch) int16 {
	var xy, xx, yy, xy2 int32
	var xcorr [3]int32
	var offset int

	minperiod0 := minperiod
	maxperiod /= 2
	minperiod /= 2
	*T0 /= 2
	prevPeriod /= 2
	N /= 2
	xb := maxperiod // x += maxperiod
	if *T0 >= maxperiod {
		*T0 = maxperiod - 1
	}

	T := *T0
	T0v := *T0
	yyLookup := alloc(&sc.yyLookup, maxperiod+1) // VARDECL(opus_val32, yy_lookup)
	xx, xy = dualInnerProd(x, xb, xb, xb-T0v, N)
	yyLookup[0] = xx
	yy = xx
	for i := 1; i <= maxperiod; i++ {
		yy = yy + fixedmath.MULT16_16(x[xb-i], x[xb-i]) - fixedmath.MULT16_16(x[xb+N-i], x[xb+N-i])
		yyLookup[i] = fixedmath.MAX32(0, yy)
	}
	yy = yyLookup[T0v]
	bestXy := xy
	bestYy := yy
	g := computePitchGain(xy, xx, yy)
	g0 := g
	// Look for any pitch at T/k.
	for k := 2; k <= 15; k++ {
		var T1, T1b int
		var g1 int16
		var cont int16
		var thresh int16
		T1 = int(fixedmath.Celt_udiv(uint32(2*T0v+k), uint32(2*k)))
		if T1 < minperiod {
			break
		}
		// Look for another strong correlation at T1b.
		if k == 2 {
			if T1+T0v > maxperiod {
				T1b = T0v
			} else {
				T1b = T0v + T1
			}
		} else {
			T1b = int(fixedmath.Celt_udiv(uint32(2*secondCheck[k]*T0v+k), uint32(2*k)))
		}
		xy, xy2 = dualInnerProd(x, xb, xb-T1, xb-T1b, N)
		xy = fixedmath.HALF32(xy + xy2)
		yy = fixedmath.HALF32(yyLookup[T1] + yyLookup[T1b])
		g1 = computePitchGain(xy, xx, yy)
		if pitchIabs(T1-prevPeriod) <= 1 {
			cont = prevGain
		} else if pitchIabs(T1-prevPeriod) <= 2 && 5*k*k < T0v {
			cont = fixedmath.HALF16(prevGain)
		} else {
			cont = 0
		}
		thresh = fixedmath.EXTRACT16(fixedmath.MAX16(int32(fixedmath.QCONST16(0.3, 15)),
			fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.7, 15), g0)-int32(cont)))
		// Bias against very high pitch (very short period) to avoid false-positives
		// due to short-term correlation.
		if T1 < 3*minperiod {
			thresh = fixedmath.EXTRACT16(fixedmath.MAX16(int32(fixedmath.QCONST16(0.4, 15)),
				fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.85, 15), g0)-int32(cont)))
		} else if T1 < 2*minperiod {
			thresh = fixedmath.EXTRACT16(fixedmath.MAX16(int32(fixedmath.QCONST16(0.5, 15)),
				fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.9, 15), g0)-int32(cont)))
		}
		if g1 > thresh {
			bestXy = xy
			bestYy = yy
			T = T1
			g = g1
		}
	}
	bestXy = fixedmath.MAX32(0, bestXy)
	var pg int16
	if bestYy <= bestXy {
		pg = int16(q15one)
	} else {
		pg = fixedmath.EXTRACT16(fixedmath.SHR32(fixedmath.Frac_div32(bestXy, bestYy+1), 16))
	}

	for k := 0; k < 3; k++ {
		xcorr[k] = celtInnerProd(x, xb, x, xb-(T+k-1), N)
	}
	if (xcorr[2] - xcorr[0]) > fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.7, 15), xcorr[1]-xcorr[0]) {
		offset = 1
	} else if (xcorr[0] - xcorr[2]) > fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.7, 15), xcorr[1]-xcorr[2]) {
		offset = -1
	} else {
		offset = 0
	}
	if pg > g {
		pg = g
	}
	*T0 = 2*T + offset

	if *T0 < minperiod0 {
		*T0 = minperiod0
	}
	return pg
}

// --- Differential-test entry points (used by internal/reftest/oracle) --------

// CeltPitchXcorr runs celtPitchXcorr and returns xcorr[0..maxPitch) plus maxcorr.
func CeltPitchXcorr(x, y []int16, len_, maxPitch int) ([]int32, int32) {
	xcorr := make([]int32, maxPitch)
	maxcorr := celtPitchXcorr(x, y, xcorr, len_, maxPitch)
	return xcorr, maxcorr
}

// PitchDownsample runs pitchDownsample over the C channel buffers and returns the
// len_ downsampled samples.
func PitchDownsample(x [][]int32, len_, C, factor int) []int16 {
	xLp := make([]int16, len_)
	var sc scratch
	pitchDownsample(x, xLp, len_, C, factor, &sc)
	return xLp
}

// PitchSearch runs pitchSearch and returns the estimated pitch lag.
func PitchSearch(xLp, y []int16, len_, maxPitch int) int {
	var pitch int
	var sc scratch
	pitchSearch(xLp, y, len_, maxPitch, &pitch, &sc)
	return pitch
}

// RemoveDoubling runs removeDoubling and returns the pitch gain and the refined
// pitch lag T0 (in/out in C).
func RemoveDoubling(x []int16, maxperiod, minperiod, N, T0, prevPeriod int, prevGain int16) (pg int16, t0Out int) {
	t := T0
	var sc scratch
	pg = removeDoubling(x, maxperiod, minperiod, N, &t, prevPeriod, prevGain, &sc)
	return pg, t
}
