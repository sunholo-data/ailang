package celt

// Transliteration of libopus celt/mdct.c (v1.6.1, commit 3da9f7a6) for the
// frozen FIXED_POINT + DISABLE_FLOAT_API + non-QEXT configuration. The CELT
// decoder uses the inverse (backward) MDCT: pre-rotation of the frequency
// coefficients, an N/4-point complex FFT, post-rotation, and the weighted
// overlap-add mirroring for time-domain aliasing cancellation.
//
// The pre/post trig rotation reads the mode's int16 .trig table and the
// rounding-brittle S_MUL/PSHR32_ovflw macros; they are transliterated exactly
// (docs/peer-review.md section 7). The forward MDCT (encoder side) is the
// transpose of the backward one and lives in cltMDCTForward below.

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// clt_mdct_backward computes a backward MDCT (no scaling) and performs the
// weighted overlap-add (implicit 1/2 scale). l is the mode's mdct_lookup, in the
// frequency-domain input (read with the given stride), out the time-domain
// overlap-add buffer, window the overlap window, overlap/shift/stride the usual
// CELT parameters (shift = maxLM-LM for a non-transient block). (celt/mdct.c:268)
func cltMDCTBackward(l *mdctLookup, in, out []int32, window []int16, overlap, shift, stride int, sc *scratch) {
	var N, N2, N4 int
	trigOff := 0

	N = l.n
	for i := 0; i < shift; i++ {
		N >>= 1
		trigOff += N
	}
	N2 = N >> 1
	N4 = N >> 2
	trig := l.trig

	// Fixed-point headroom: derive pre/post/fft shifts from the input range.
	var preShift, postShift, fftShift int
	{
		sumval := int32(N2)
		maxval := int32(0)
		for i := 0; i < N2; i++ {
			maxval = fixedmath.MAX32(maxval, fixedmath.ABS32(in[i*stride]))
			sumval = fixedmath.ADD32_ovflw(sumval, fixedmath.ABS32(fixedmath.SHR32(in[i*stride], 11)))
		}
		preShift = fixedmath.IMAX(0, 29-fixedmath.Celt_zlog2(1+maxval))
		// Worst-case where all the energy goes to a single sample.
		postShift = fixedmath.IMAX(0, 19-fixedmath.Celt_ilog2(fixedmath.ABS32(sumval)))
		postShift = fixedmath.IMIN(postShift, preShift)
		fftShift = preShift - postShift
	}

	// The FFT works on a private N4-length complex buffer. In C the pre-rotation
	// is stored directly into out+(overlap/2) in bitrev order and the FFT runs
	// in place there; keeping a separate f2 slice reproduces the exact values
	// (the post-rotation only reads what was already written) without a []int32
	// -> []kissFFTCpx reinterpretation. Pooled on the codec scratch (shared with
	// the forward MDCT's f2; see scratch.go): the pre-rotation below writes all N4
	// entries in bitrev order before the post-rotation reads any, so a carried-over
	// buffer is never observed.
	f2 := alloc(&sc.mdctF2, N4) // VARDECL(kiss_fft_cpx, f2)

	// Pre-rotate.
	{
		xp1 := 0
		xp2 := stride * (N2 - 1)
		bitrev := l.kfft[shift].bitrev
		for i := 0; i < N4; i++ {
			rev := int(bitrev[i])
			x1 := fixedmath.SHL32_ovflw(in[xp1], preShift)
			x2 := fixedmath.SHL32_ovflw(in[xp2], preShift)
			yr := fixedmath.ADD32_ovflw(sMul(x2, trig[trigOff+i]), sMul(x1, trig[trigOff+N4+i]))
			yi := fixedmath.SUB32_ovflw(sMul(x1, trig[trigOff+i]), sMul(x2, trig[trigOff+N4+i]))
			// We swap real and imag because we use an FFT instead of an IFFT;
			// storing the pre-rotation directly in the bitrev order.
			f2[rev].i = yr
			f2[rev].r = yi
			xp1 += 2 * stride
			xp2 -= 2 * stride
		}
	}

	opusFFTImpl(l.kfft[shift], f2, fftShift)

	// Post-rotate and de-shuffle from both ends of the FFT output into out,
	// starting at out+(overlap/2). o0 walks the low half up, o1 the high half
	// down; k1 is the descending complex index (matching yp1 in C).
	{
		o := overlap >> 1
		for i := 0; i < (N4+1)>>1; i++ {
			k0 := i
			k1 := N4 - i - 1

			// We swap real and imag because we're using an FFT instead of an IFFT.
			re := f2[k0].i
			im := f2[k0].r
			t0 := trig[trigOff+i]
			t1 := trig[trigOff+N4+i]
			// We'd scale up by 2 here, but it's done when mixing the windows.
			yr0 := fixedmath.PSHR32_ovflw(fixedmath.ADD32_ovflw(sMul(re, t0), sMul(im, t1)), postShift)
			yi0 := fixedmath.PSHR32_ovflw(fixedmath.SUB32_ovflw(sMul(re, t1), sMul(im, t0)), postShift)

			re = f2[k1].i
			im = f2[k1].r
			// out[o+2*k0] and out[o+N2-1-2*k0] correspond to yp0[0] and yp1[1].
			out[o+2*k0] = yr0
			out[o+N2-1-2*k0] = yi0

			t0 = trig[trigOff+(N4-i-1)]
			t1 = trig[trigOff+(N2-i-1)]
			yr1 := fixedmath.PSHR32_ovflw(fixedmath.ADD32_ovflw(sMul(re, t0), sMul(im, t1)), postShift)
			yi1 := fixedmath.PSHR32_ovflw(fixedmath.SUB32_ovflw(sMul(re, t1), sMul(im, t0)), postShift)
			// out[o+N2-2-2*k0] and out[o+2*k0+1] correspond to yp1[0] and yp0[1].
			out[o+N2-2-2*k0] = yr1
			out[o+2*k0+1] = yi1
		}
	}

	// Mirror on both sides for TDAC.
	{
		xp1 := overlap - 1
		yp1 := 0
		wp1 := 0
		wp2 := overlap - 1
		for i := 0; i < overlap/2; i++ {
			x1 := out[xp1]
			x2 := out[yp1]
			out[yp1] = fixedmath.SUB32_ovflw(sMul(x2, window[wp2]), sMul(x1, window[wp1]))
			out[xp1] = fixedmath.ADD32_ovflw(sMul(x2, window[wp1]), sMul(x1, window[wp2]))
			yp1++
			xp1--
			wp1++
			wp2--
		}
	}
}

// cltMDCTForward computes a forward MDCT (encoder side). It windows/shuffles/
// folds the time-domain input into a real N2 buffer, pre-rotates while folding
// in the forward FFT's input prescale (S_MUL2 by st.scale) and choosing a
// headroom so the N/4 complex FFT runs at maximum precision, then post-rotates
// into the strided half-spectrum output. l is the mode's mdct_lookup, in the
// time-domain windowed/overlapped input (read with unit stride, length >=
// N2+overlap), out the frequency-domain output (written with the given stride),
// window the overlap window, overlap/shift/stride the usual CELT parameters.
//
// This is the transpose of cltMDCTBackward; the differences from the already-
// ported inverse are: (1) the input is windowed/folded first (three-loop
// [a,b,c,d] shuffle) rather than pre-shifted by a range-derived pre_shift;
// (2) the pre/post trig rotations do NOT swap real and imag (forward direction,
// where the backward swaps to run an IFFT through the forward engine); (3) the
// forward FFT's S_MUL2 prescale is folded into the pre-rotation, so the shared
// engine is entered directly with the scale_shift-headroom downshift budget
// instead of a range-derived fft_shift; (4) there is no TDAC overlap-add on the
// output. (celt/mdct.c:122, clt_mdct_forward_c)
func cltMDCTForward(l *mdctLookup, in, out []int32, window []int16, overlap, shift, stride int, sc *scratch) {
	var N, N2, N4 int
	st := l.kfft[shift]
	// Allows us to scale with MULT16_32_Q16(), which is faster than
	// MULT16_32_Q15() on ARM.
	scaleShift := st.scaleShift - 1
	scale := st.scale

	N = l.n
	trigOff := 0
	for i := 0; i < shift; i++ {
		N >>= 1
		trigOff += N
	}
	N2 = N >> 1
	N4 = N >> 2
	trig := l.trig

	// The FFT works on a private N4-length complex buffer; f holds the folded
	// real input before the pre-rotation. Unlike the inverse MDCT (which runs
	// the FFT in place in the output overlap-add buffer), the forward transform
	// keeps f and f2 separate exactly as the C does.
	f := alloc(&sc.mdctF, N2)   // VARDECL(kiss_fft_scalar, f)
	f2 := alloc(&sc.mdctF2, N4) // VARDECL(kiss_fft_cpx, f2)

	// Consider the input to be composed of four blocks: [a, b, c, d].
	// Window, shuffle, fold.
	{
		xp1 := overlap >> 1
		xp2 := N2 - 1 + (overlap >> 1)
		yp := 0
		wp1 := overlap >> 1
		wp2 := (overlap >> 1) - 1
		i := 0
		for ; i < (overlap+3)>>2; i++ {
			// Real part arranged as -d-cR, Imag part arranged as -b+aR.
			f[yp] = sMul(in[xp1+N2], window[wp2]) + sMul(in[xp2], window[wp1])
			yp++
			f[yp] = sMul(in[xp1], window[wp1]) - sMul(in[xp2-N2], window[wp2])
			yp++
			xp1 += 2
			xp2 -= 2
			wp1 += 2
			wp2 -= 2
		}
		wp1 = 0
		wp2 = overlap - 1
		for ; i < N4-((overlap+3)>>2); i++ {
			// Real part arranged as a-bR, Imag part arranged as -c-dR.
			f[yp] = in[xp2]
			yp++
			f[yp] = in[xp1]
			yp++
			xp1 += 2
			xp2 -= 2
		}
		for ; i < N4; i++ {
			// Real part arranged as a-bR, Imag part arranged as -c-dR.
			f[yp] = -sMul(in[xp1-N2], window[wp1]) + sMul(in[xp2], window[wp2])
			yp++
			f[yp] = sMul(in[xp1], window[wp2]) + sMul(in[xp2+N2], window[wp1])
			yp++
			xp1 += 2
			xp2 -= 2
			wp1 += 2
			wp2 -= 2
		}
	}
	// Pre-rotation. No real/imag swap (forward direction). The forward FFT's
	// S_MUL2(., scale) input prescale is folded in here; maxval then bounds the
	// prescaled data so the FFT can be entered directly with the remaining
	// scale_shift-headroom downshift budget (see opus_fft_c, which prescales and
	// runs opus_fft_impl with scale_shift). For non-QEXT it is best to scale
	// after the rotation but before the FFT, as done here.
	var headroom int
	{
		yp := 0
		maxval := int32(1)
		for i := 0; i < N4; i++ {
			t0 := trig[trigOff+i]
			t1 := trig[trigOff+N4+i]
			re := f[yp]
			yp++
			im := f[yp]
			yp++
			yr := sMul(re, t0) - sMul(im, t1)
			yi := sMul(im, t0) + sMul(re, t1)
			var yc kissFFTCpx
			yc.r = sMul2(yr, scale)
			yc.i = sMul2(yi, scale)
			maxval = fixedmath.MAX32(maxval, fixedmath.MAX32(fixedmath.ABS32(yc.r), fixedmath.ABS32(yc.i)))
			f2[st.bitrev[i]] = yc
		}
		headroom = fixedmath.IMAX(0, fixedmath.IMIN(scaleShift, 28-fixedmath.Celt_ilog2(maxval)))
	}

	// N/4 complex FFT, does not downscale anymore.
	opusFFTImpl(st, f2, scaleShift-headroom)

	// Post-rotate. No real/imag swap (forward direction); the residual headroom
	// shift is applied here via PSHR32 and the result is de-interleaved into the
	// strided output from both ends (yp1 up, yp2 down).
	{
		fp := 0
		yp1 := 0
		yp2 := stride * (N2 - 1)
		for i := 0; i < N4; i++ {
			t0 := trig[trigOff+i]
			t1 := trig[trigOff+N4+i]
			yr := fixedmath.PSHR32(sMul(f2[fp].i, t1)-sMul(f2[fp].r, t0), headroom)
			yi := fixedmath.PSHR32(sMul(f2[fp].r, t1)+sMul(f2[fp].i, t0), headroom)
			out[yp1] = yr
			out[yp2] = yi
			fp++
			yp1 += 2 * stride
			yp2 -= 2 * stride
		}
	}
}

// InverseMDCT drives clt_mdct_backward on mode48000_960 for LM in 0..3 (mono,
// non-transient, stride 1): shift = maxLM-LM, so the FFT length is 60/120/240/
// 480 and the input holds shortMdctSize<<LM = 120<<LM frequency coefficients.
// The returned slice is the written region of the overlap-add output,
// overlap/2 + (120<<LM) samples long. Exported only so the refc cgo differential
// harness (internal/reftest/oracle) can drive the pure-Go inverse MDCT against
// libopus; it is not part of the decoder API.
func InverseMDCT(lm int, in []int32) []int32 {
	m := &mode48000_960
	shift := m.maxLM - lm
	overlap := m.overlap
	N := m.mdct.n >> shift
	N2 := N >> 1
	// Backward MDCT reads in[0..N2-1] (stride 1); copy defensively so the caller
	// input is never mutated (the C entry point treats it as read-only here).
	inCopy := make([]int32, N2)
	copy(inCopy, in)
	out := make([]int32, overlap/2+N2)
	var sc scratch
	cltMDCTBackward(&m.mdct, inCopy, out, m.window, overlap, shift, 1, &sc)
	return out
}

// ForwardMDCT drives clt_mdct_forward on mode48000_960 for LM in 0..3 with the
// given output stride (1 for the mono/single-block call shape, 2 for the
// interleaved stereo/two-block one): shift = maxLM-LM, so the FFT length is
// 60/120/240/480 and the input holds N2+overlap = (120<<LM)+overlap time-domain
// samples. The returned slice is the strided frequency output, stride*(120<<LM)
// samples long (the transform writes only the strided slots; the rest stay 0).
// Exported only so the refc cgo differential harness (internal/reftest/oracle)
// can drive the pure-Go forward MDCT against libopus; it is not part of the
// encoder API.
func ForwardMDCT(lm, stride int, in []int32) []int32 {
	m := &mode48000_960
	shift := m.maxLM - lm
	overlap := m.overlap
	N := m.mdct.n >> shift
	N2 := N >> 1
	// clt_mdct_forward reads in[0..N2+overlap-1] read-only; copy defensively so
	// the caller input is never observed as mutated.
	inCopy := make([]int32, N2+overlap)
	copy(inCopy, in)
	out := make([]int32, stride*N2)
	var sc scratch
	cltMDCTForward(&m.mdct, inCopy, out, m.window, overlap, shift, stride, &sc)
	return out
}
