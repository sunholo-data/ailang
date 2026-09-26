// Transliteration of the DECODE side of celt/vq.c (libopus v1.6.1, commit
// 3da9f7a6) for the frozen FIXED_POINT + DISABLE_FLOAT_API build: PVQ codeword
// dequantization. AlgUnquant decodes a pulse vector (DecodePulses) and normalises
// it to unit norm, ExpRotation applies the spread rotation (shared by encode and
// decode; transliterated exactly per docs/peer-review.md 7), and
// RenormaliseVector rescales a band to unit norm.
//
// In this build celt_norm is opus_val32 (arch.h:145) and NORM_SHIFT is 24
// (arch.h:183), so the norm buffers are int32 and the rotation runs in a scaled
// Q14 domain via normScaledown/normScaleup by NORM_SHIFT-14. Because celt_norm is
// 32-bit here, MULT16_16(coef, x) in the C source degenerates to a plain 32-bit
// product (the norm sample already fits well inside int16 after scaledown), so
// the rotation arithmetic is written out with explicit int32 products rather than
// the int16-typed fixedmath.MULT16_16 helper.
//
// The phase-4 encode side (opPvqSearch / AlgQuant / StereoItheta, plus the
// celt_atan2p_norm math helper stereo_itheta needs) is transliterated below. The
// ENABLE_QEXT op_pvq_search_N2 / _extra / cubic_* refine paths are excluded from
// the frozen build and are NOT ported.

package celt

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
)

// normShift is celt/arch.h NORM_SHIFT (24): the fixed-point scale of a celt_norm
// (opus_val32) unit-norm band sample. normScaledown/normScaleup shift between the
// storage domain and the Q14 rotation domain by normShift-14. (arch.h:183)
const normShift = 24

// spreadNone is celt/bands.h SPREAD_NONE; exp_rotation is a no-op for it.
const spreadNone = 0

// spreadFactor is celt/vq.c exp_rotation's SPREAD_FACTOR[3] for spread values
// LIGHT/NORMAL/AGGRESSIVE (1/2/3, indexed spread-1). (vq.c:106)
var spreadFactor = [3]int{15, 10, 5}

// normScaleup shifts each band sample left by shift (a no-op for shift<=0). The
// FIXED_POINT norm_scaleup (vq.c:44).
func normScaleup(X []int32, n, shift int) {
	if shift <= 0 {
		return
	}
	for i := 0; i < n; i++ {
		X[i] = fixedmath.SHL32(X[i], shift)
	}
}

// normScaledown shifts each band sample right (rounding) by shift (a no-op for
// shift<=0). The FIXED_POINT norm_scaledown (vq.c:51).
func normScaledown(X []int32, n, shift int) {
	if shift <= 0 {
		return
	}
	for i := 0; i < n; i++ {
		X[i] = fixedmath.PSHR32(X[i], shift)
	}
}

// celtInnerProdNorm is the FIXED_POINT celt_inner_prod_norm: a plain 32-bit
// accumulation of x[i]*y[i] over the scaled norm samples. (vq.c:58)
func celtInnerProdNorm(x, y []int32, length int) int32 {
	var sum int32
	for i := 0; i < length; i++ {
		sum += x[i] * y[i]
	}
	return sum
}

// expRotation1 applies one Givens-style rotation pass with the given stride and
// (c,s) coefficients, working in the Q14 rotation domain. Rounding-brittle: the
// PSHR32 rounding and the int16 (EXTRACT16) lane truncation are transliterated
// exactly (vq.c:75). The norm_scaledown/norm_scaleup that bridge celt_norm's
// wider storage shift (normShift) down to this Q14 domain are hoisted to the
// caller (expRotation), which brackets a whole block's rotation passes once; a
// scaleup at the end of one pass followed by a scaledown at the start of the next
// is the identity for the int16-range values EXTRACT16 produces, so hoisting them
// drops two full passes per block in the two-pass case without changing a byte.
// Because celt_norm is int32 here, MULT16_16(c,x) is a 32-bit product and
// MAC16_16 a 32-bit multiply-add; the results still fit int32 for scaled norm
// inputs.
func expRotation1(X []int32, length, stride int, c, s int16) {
	ms := int16(fixedmath.NEG16(s))

	// Both passes below touch the pair (X[i], X[i+stride]); length<=stride makes
	// the original forward loop (i<length-stride) empty, and by extension the
	// original backward loop too (its start length-2*stride-1 is then also
	// negative), so this guard reproduces both no-ops without slicing on a
	// negative bound.
	if length > stride {
		// Forward pass, i from 0 while i<length-stride. The overlapping windows
		// a, b give a[i]==X[i] and b[i]==X[i+stride] for i in [0,length-stride),
		// so walking a by range drops the per-iteration check entirely.
		a := X[: length-stride : length]
		b := X[stride:length]
		for i := range a {
			x1 := a[i]
			x2 := b[i]
			b[i] = int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(c)*x2+int32(s)*x1, 15)))
			a[i] = int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(c)*x1+int32(ms)*x2, 15)))
		}

		// Backward pass, i from length-2*stride-1 down to 0: the same pair shape
		// over a shorter prefix. Re-cap a and b to that prefix so BCE can prove
		// this loop too.
		if n2 := length - 2*stride; n2 > 0 {
			a2, b2 := a[:n2], b[:n2]
			for i := n2 - 1; i >= 0; i-- {
				x1 := a2[i]
				x2 := b2[i]
				b2[i] = int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(c)*x2+int32(s)*x1, 15)))
				a2[i] = int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(c)*x1+int32(ms)*x2, 15)))
			}
		}
	}
}

// expRotation1Stride1 is expRotation1 specialized to stride == 1, the rotation
// pass that runs for every block on every call. At stride 1 the pair windows
// overlap so that the forward pass's store to b[i] (X[i+1]) is consumed only as
// the next iteration's x1 (a[i+1]) and is then overwritten by that iteration's
// a[i+1] store; the backward pass is the mirror image on a[i]. So the genuine
// sequential recurrence (the part of the spread rotation issue #44 says can never
// be vectorized) is carried in a register instead of round-tripping through
// memory every iteration, and only the surviving store per index is written: a[i]
// in the forward pass with the single boundary element X[length-1] flushed after
// it, and b[i] in the backward pass with X[0] flushed after it.
//
// Bit-exact against expRotation1 called with stride 1: every computed value,
// operand, PSHR32 round-half-up at 15, EXTRACT16 narrowing, and the ms = NEG16(s)
// wrap is textually identical, and the intermediate stores skipped here are
// exactly the ones expRotation1 overwrites one iteration later, so the final
// contents of X are the same byte for byte. Guarded on length > 1 to reproduce
// expRotation1's `length > stride` no-op.
func expRotation1Stride1(X []int32, length int, c, s int16) {
	if length <= 1 {
		return
	}
	ms := int16(fixedmath.NEG16(s))

	// Forward pass, i in [0, length-1). a[i]==X[i], b[i]==X[i+1]. x1 carries X[i]
	// (the value expRotation1 reads as a[i] each iteration); the b[i] result is
	// held in t and promoted to the next iteration's x1 rather than stored.
	a := X[: length-1 : length]
	b := X[1:length]
	x1 := a[0]
	for i := range a {
		x2 := b[i]
		t := int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(c)*x2+int32(s)*x1, 15)))
		a[i] = int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(c)*x1+int32(ms)*x2, 15)))
		x1 = t
	}
	// X[length-1] is written only by the forward pass (a never reaches it), so
	// flush the final carry once.
	b[len(b)-1] = x1

	// Backward pass, i from length-2-1 down to 0 over the shorter prefix. x2
	// carries X[i+1] (the value expRotation1 reads as b[i] each iteration); the
	// a[i] result is held in x2 and overwritten next iteration rather than stored.
	if n2 := length - 2; n2 > 0 {
		a2, b2 := a[:n2], b[:n2]
		x2 := b2[n2-1]
		for i := n2 - 1; i >= 0; i-- {
			x1 := a2[i]
			b2[i] = int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(c)*x2+int32(s)*x1, 15)))
			x2 = int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(c)*x1+int32(ms)*x2, 15)))
		}
		// X[0] is written only by the backward pass, so flush its final carry once.
		a2[0] = x2
	}
}

// expRotation applies (dir<0) or undoes (dir>0) the spread rotation over the band
// X of the given length, split into stride interleaved blocks with K pulses and
// the given spread. No-op when 2*K>=len or spread==SPREAD_NONE. (vq.c:104)
func expRotation(X []int32, length, dir, stride, k, spread int) {
	if 2*k >= length || spread == spreadNone {
		return
	}
	factor := spreadFactor[spread-1]

	// gain = celt_div(MULT16_16(Q15_ONE,len), len+factor*K); celt_div(a,b) is
	// MULT32_32_Q31(a, celt_rcp(b)) in FIXED_POINT (mathops.h:539).
	num := int32(32767) * int32(length) // (opus_val32)MULT16_16(Q15_ONE,len)
	den := int32(length + factor*k)     // (opus_val32)(len+factor*K)
	gain := fixedmath.EXTRACT16(fixedmath.MULT32_32_Q31(num, fixedmath.Celt_rcp(den)))
	// theta = HALF16(MULT16_16_Q15(gain,gain)); the >>1 acts on the opus_val32
	// product before the opus_val16 truncation.
	theta := fixedmath.EXTRACT16(fixedmath.SHR32(fixedmath.MULT16_16_Q15(gain, gain), 1))

	c := fixedmath.Celt_cos_norm(fixedmath.EXTEND32(theta))
	// s = celt_cos_norm(Q15ONE - theta) = sin(theta).
	s := fixedmath.Celt_cos_norm(fixedmath.SUB16(32767, theta))

	stride2 := 0
	if length >= 8*stride {
		stride2 = 1
		// Equivalent to sqrt(len/stride) with rounding: increment while
		// (stride2+0.5)^2 < len/stride.
		for (stride2*stride2+stride2)*stride+(stride>>2) < length {
			stride2++
		}
	}
	length = int(fixedmath.Celt_udiv(uint32(length), uint32(stride)))
	for i := 0; i < stride; i++ {
		block := X[i*length:]
		// Bridge celt_norm's storage shift to the Q14 rotation domain once around
		// all of this block's passes (the C celt_norm is already opus_val16 so it
		// has no equivalent). Consecutive expRotation1 passes then share the Q14
		// domain rather than each round-tripping it through scaleup/scaledown, which
		// is the identity for the int16-range values EXTRACT16 produces.
		normScaledown(block, length, normShift-14)
		if dir < 0 {
			if stride2 != 0 {
				expRotation1(block, length, stride2, s, c)
			}
			expRotation1Stride1(block, length, c, s)
		} else {
			expRotation1Stride1(block, length, c, int16(fixedmath.NEG16(s)))
			if stride2 != 0 {
				expRotation1(block, length, stride2, s, int16(fixedmath.NEG16(c)))
			}
		}
		normScaleup(block, length, normShift-14)
	}
}

// ExpRotation exposes expRotation for the differential test (both directions).
func ExpRotation(X []int32, length, dir, stride, k, spread int) {
	expRotation(X, length, dir, stride, k, spread)
}

// normaliseResidual normalises the decoded integer pulse vector iy to unit norm,
// writing the result into X. Ryy is the sum of squares from DecodePulses, gain is
// the band gain (Q31). The ENABLE_QEXT shift path is omitted (shift is always 0
// here). (vq.c:150)
func normaliseResidual(iy []int32, X []int32, n int, ryy, gain int32) {
	k := fixedmath.Celt_ilog2(ryy) >> 1
	t := fixedmath.VSHR32(ryy, 2*(k-7)-15)
	g := fixedmath.MULT32_32_Q31(fixedmath.Celt_rsqrt_norm32(t), gain)
	for i := 0; i < n; i++ {
		X[i] = fixedmath.VSHR32(fixedmath.MULT16_32_Q15(int16(iy[i]), g), k+15-normShift)
	}
}

// extractCollapseMask returns the anti-collapse mask: bit i is set when the i'th
// of B interleaved sub-blocks received at least one pulse. (vq.c:183)
func extractCollapseMask(iy []int32, n, b int) uint32 {
	if b <= 1 {
		return 1
	}
	n0 := int(fixedmath.Celt_udiv(uint32(n), uint32(b)))
	var collapseMask uint32
	for i := 0; i < b; i++ {
		var tmp uint32
		for j := 0; j < n0; j++ {
			tmp |= uint32(iy[i*n0+j])
		}
		collapseMask |= uint32(b2i(tmp != 0)) << i
	}
	return collapseMask
}

// AlgUnquant is the differential-test entry point over algUnquant, standing in for
// the decoder-held scratch the production path threads through.
func AlgUnquant(X []int32, n, k, spread, b int, dec *rangecoding.Decoder, gain int32) uint32 {
	var sc scratch
	return algUnquant(X, n, k, spread, b, dec, gain, &sc)
}

// algUnquant decodes a PVQ codeword and combines it into the final normalised band
// signal X (length N), returning the anti-collapse mask. K pulses, spread rotation,
// B interleaved blocks, band gain (Q31). (vq.c:619). Requires K>0, N>1. iy comes
// from the pooled scratch: cwrsi (DecodePulses) fills all N entries before
// normalise_residual and extract_collapse_mask read them, so the pooled buffer's
// carried-over values are never observed (see scratch.go's decode zeroing audit).
func algUnquant(X []int32, n, k, spread, b int, dec *rangecoding.Decoder, gain int32, sc *scratch) uint32 {
	iy := alloc(&sc.decIy, n) // VARDECL(int, iy)
	ryy := DecodePulses(iy, n, k, dec)
	normaliseResidual(iy, X, n, ryy, gain)
	expRotation(X, n, -1, b, k, spread)
	return extractCollapseMask(iy, n, b)
}

// RenormaliseVector rescales the band X (length N) to unit norm times gain (Q31).
// The decode side uses this for bands with no pulses. (vq.c:693)
func RenormaliseVector(X []int32, n int, gain int32) {
	normScaledown(X, n, normShift-14)
	e := int32(1) /* EPSILON */ + celtInnerProdNorm(X, X, n)
	k := fixedmath.Celt_ilog2(e) >> 1
	t := fixedmath.VSHR32(e, 2*(k-7))
	g := fixedmath.EXTRACT16(fixedmath.MULT32_32_Q31(fixedmath.EXTEND32(fixedmath.Celt_rsqrt_norm(t)), gain))
	for i := 0; i < n; i++ {
		X[i] = int32(fixedmath.EXTRACT16(fixedmath.PSHR32(int32(g)*X[i], k+15-14)))
	}
	normScaleup(X, n, normShift-14)
}

// opPvqSearch is the FIXED_POINT base op_pvq_search_c (vq.c:205): the greedy
// pyramid pulse search that places K unit pulses over the length-N band X to
// maximise the projection onto X, returning the coded pulse vector in iy and yy =
// the (int16-truncated) running sum of squares that normalise_residual consumes.
//
// Rounding-brittle Layer-C code, transliterated verbatim. The load-bearing detail
// is that yy is an opus_val16 (int16) accumulator: every MAC16_16/ADD16 result is
// truncated back to int16 on assignment, and MULT16_16 truncates its celt_norm
// (int32) operands to int16 first. The leading norm_scaledown keeps X small enough
// that these int16 lanes do not overflow. EXTEND32 of a celt_norm is a plain
// widening (no int16 truncation), so those sites use the int32 value directly.
// The ENABLE_QEXT op_pvq_search_N2 / _extra refine variants are excluded.
func opPvqSearch(X []int32, iy []int32, K, N int, sc *scratch) int16 {
	// X and iy are caller-owned windows (algQuant passes its own X parameter and
	// an alloc(&sc.pvqIy, N) slice); re-slicing both to the exact length N here,
	// where N is otherwise opaque to this function's compiled body, is what lets
	// BCE prove the N-bounded loops below in range. y and signx come back from
	// alloc() through a pointer indirection, which loses the same length fact,
	// so they get the same treatment.
	//
	// A re-slice can WIDEN a slice whose len is under N (up to cap), which would
	// turn a misuse that previously panicked mid-loop into silent writes past the
	// caller's window. Refuse that explicitly; every real caller passes len == N.
	if N > len(X) || N > len(iy) {
		panic("celt: opPvqSearch: N exceeds operand length")
	}
	X = X[:N]
	iy = iy[:N]
	y := alloc(&sc.pvqY, N)         // VARDECL(celt_norm, y)
	signx := alloc(&sc.pvqSignx, N) // VARDECL(int, signx)
	y = y[:N]
	signx = signx[:N]
	var i, j int
	var pulsesLeft int
	var sum int32 // opus_val32
	var xy int32  // opus_val32
	var yy int16  // opus_val16

	// FIXED_POINT: scale X down so the int16 accumulators below cannot overflow.
	shift := (fixedmath.Celt_ilog2(1+celtInnerProdNormShift(X, X, N)) + 1) / 2
	shift = fixedmath.IMAX(0, shift+(normShift-14)-14)
	normScaledown(X, N, shift)

	// Get rid of the sign. ABS16(X[j]) on a celt_norm (int32) is a plain int32 abs.
	sum = 0
	for j = 0; j < N; j++ {
		signx[j] = b2i(X[j] < 0)
		X[j] = fixedmath.ABS32(X[j])
		iy[j] = 0
		y[j] = 0
	}

	xy = 0
	yy = 0

	pulsesLeft = K

	// Do a pre-search by projecting on the pyramid.
	if K > (N >> 1) {
		var rcp int16
		for j = 0; j < N; j++ {
			sum += X[j]
		}
		// If X is too small, just replace it with a pulse at 0.
		if sum <= int32(K) {
			X[0] = 16384 // QCONST16(1.f,14)
			for j = 1; j < N; j++ {
				X[j] = 0
			}
			sum = 16384 // QCONST16(1.f,14)
		}
		rcp = fixedmath.EXTRACT16(fixedmath.MULT16_32_Q16(int16(K), fixedmath.Celt_rcp(sum)))
		for j = 0; j < N; j++ {
			// It's really important to round *towards zero* here (X[j]>=0).
			iy[j] = fixedmath.MULT16_16_Q15(fixedmath.EXTRACT16(X[j]), rcp)
			y[j] = iy[j]
			yy = fixedmath.EXTRACT16(fixedmath.MAC16_16(int32(yy), fixedmath.EXTRACT16(y[j]), fixedmath.EXTRACT16(y[j])))
			xy = fixedmath.MAC16_16(xy, fixedmath.EXTRACT16(X[j]), fixedmath.EXTRACT16(y[j]))
			y[j] *= 2
			pulsesLeft -= int(iy[j])
		}
	}
	// celt_sig_assert(pulsesLeft>=0)

	// This should never happen, but just in case it does (e.g. on silence) we
	// fill the first bin with pulses.
	if pulsesLeft > N+3 {
		tmp := int16(pulsesLeft)
		yy = fixedmath.EXTRACT16(fixedmath.MAC16_16(int32(yy), tmp, tmp))
		yy = fixedmath.EXTRACT16(fixedmath.MAC16_16(int32(yy), tmp, fixedmath.EXTRACT16(y[0])))
		iy[0] += int32(pulsesLeft)
		pulsesLeft = 0
	}

	for i = 0; i < pulsesLeft; i++ {
		var Rxy, Ryy, bestDen int16
		var bestNum int32
		var bestID int
		rshift := 1 + fixedmath.Celt_ilog2(int32(K-pulsesLeft+i+1))
		bestID = 0
		// The squared magnitude term gets added anyway, so add it out of the loop.
		yy = fixedmath.ADD16(yy, 1)

		// Calculations for position 0 are out of the loop. We're multiplying y[j]
		// by two (done above) so we don't have to do it here.
		Rxy = fixedmath.EXTRACT16(fixedmath.SHR32(fixedmath.ADD32(xy, X[0]), rshift))
		Ryy = fixedmath.ADD16(yy, fixedmath.EXTRACT16(y[0]))
		Rxy = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(Rxy, Rxy))
		bestDen = Ryy
		bestNum = int32(Rxy)
		for j = 1; j < N; j++ {
			Rxy = fixedmath.EXTRACT16(fixedmath.SHR32(fixedmath.ADD32(xy, X[j]), rshift))
			Ryy = fixedmath.ADD16(yy, fixedmath.EXTRACT16(y[j]))
			Rxy = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(Rxy, Rxy))
			// Check num/den >= best_num/best_den without any division.
			if fixedmath.MULT16_16(bestDen, Rxy) > fixedmath.MULT16_16(Ryy, fixedmath.EXTRACT16(bestNum)) {
				bestDen = Ryy
				bestNum = int32(Rxy)
				bestID = j
			}
		}

		// Updating the sums of the new pulse(s).
		xy = fixedmath.ADD32(xy, X[bestID])
		yy = fixedmath.ADD16(yy, fixedmath.EXTRACT16(y[bestID]))
		// Only now that we've made the final choice, update y/iy.
		y[bestID] += 2
		iy[bestID]++
	}

	// Put the original sign back.
	for j = 0; j < N; j++ {
		iy[j] = (iy[j] ^ int32(-signx[j])) + int32(signx[j])
	}
	return yy
}

// AlgQuant is the differential-test entry point over algQuant, standing in for the
// encoder-held scratch the production path threads through.
func AlgQuant(X []int32, N, K, spread, B int, enc *rangecoding.Encoder, gain int32, resynth int) uint32 {
	var sc scratch
	return algQuant(X, N, K, spread, B, enc, gain, resynth, &sc)
}

// algQuant is the FIXED_POINT base alg_quant (vq.c:550): rotate X into the search
// domain, run the PVQ pulse search, code the pulses, and (when resynth != 0)
// reconstruct the unit-norm band into X. Returns the anti-collapse mask. The
// ENABLE_QEXT extra-bits refine branches are excluded; the arch parameter is
// dropped. Requires K>0, N>1.
func algQuant(X []int32, N, K, spread, B int, enc *rangecoding.Encoder, gain int32, resynth int, sc *scratch) uint32 {
	// celt_assert2(K>0); celt_assert2(N>1)
	iy := alloc(&sc.pvqIy, N) // VARDECL(int, iy)

	// expRotation1 (below) applies a fixed norm_scaledown by NORM_SHIFT-14 (=10)
	// whose int16 lane stays bit-exact with the C MULT16_16 truncation only while
	// |X[i]| < 33,553,920 (one past 32767<<10 after PSHR rounding; at that value
	// |X|>>10 rounds to 32768 and wraps). Every reachable encoder input satisfies
	// this: the input band is unit-norm (a full unit is 1.0 = 2^NORM_SHIFT = 2^24,
	// and theta-split sub-bands have norm <= 1), and the stereo_split/haar1 worst
	// case reaches only ~23.7M (~1.41x margin). haar1 is norm-preserving and the
	// rotation preserves norm, so no component ever crosses the threshold.
	expRotation(X, N, 1, B, K, spread)

	yy := int32(opPvqSearch(X, iy, K, N, sc))
	collapseMask := extractCollapseMask(iy, N, B)
	EncodePulses(iy, N, K, enc)
	if resynth != 0 {
		normaliseResidual(iy, X, N, yy, gain)
	}

	if resynth != 0 {
		expRotation(X, N, -1, B, K, spread)
	}
	return collapseMask
}

// atan approximation constants for celtAtanNorm (mathops.h:550-557); each is a
// 32-bit polynomial coefficient (the /* Qxx */ comments name the Q format of the
// coefficient, but the stored value is a plain int32 constant).
const (
	atanTwoOverPi int32 = 1367130551  // Q31
	atanCoeffA03  int32 = -715791936  // Q31
	atanCoeffA05  int32 = 857391616   // Q32
	atanCoeffA07  int32 = -1200579328 // Q33
	atanCoeffA09  int32 = 1682636672  // Q34
	atanCoeffA11  int32 = -1985085440 // Q35
	atanCoeffA13  int32 = 1583306112  // Q36
	atanCoeffA15  int32 = -598602432  // Q37
)

// celtAtanNorm computes atan(x)*2/pi for x in [-1,1] Q30, result Q30. Verbatim
// transliteration of the FIXED_POINT celt_atan_norm (mathops.h:547).
func celtAtanNorm(x int32) int32 {
	// celt_sig_assert((x <= 1073741824) && (x >= -1073741824))
	if x == 1073741824 {
		return 536870912 // 0.5f (Q30)
	}
	if x == -1073741824 {
		return -536870912 // -0.5f (Q30)
	}
	xQ31 := fixedmath.SHL32(x, 1)
	xSqQ30 := fixedmath.MULT32_32_Q31(xQ31, x)
	// Split evaluation in steps to avoid exploding macro expansion.
	tmp := fixedmath.MULT32_32_Q31(xSqQ30, atanCoeffA15)
	tmp = fixedmath.MULT32_32_Q31(xSqQ30, fixedmath.ADD32(atanCoeffA13, tmp))
	tmp = fixedmath.MULT32_32_Q31(xSqQ30, fixedmath.ADD32(atanCoeffA11, tmp))
	tmp = fixedmath.MULT32_32_Q31(xSqQ30, fixedmath.ADD32(atanCoeffA09, tmp))
	tmp = fixedmath.MULT32_32_Q31(xSqQ30, fixedmath.ADD32(atanCoeffA07, tmp))
	tmp = fixedmath.MULT32_32_Q31(xSqQ30, fixedmath.ADD32(atanCoeffA05, tmp))
	tmp = fixedmath.MULT32_32_Q31(xSqQ30, fixedmath.ADD32(atanCoeffA03, tmp))
	tmp = fixedmath.ADD32(x, fixedmath.MULT32_32_Q31(xQ31, tmp))
	return fixedmath.MULT32_32_Q31(atanTwoOverPi, tmp)
}

// celtAtan2pNorm computes atan2(y,x)*2/pi in Q30 for x,y >= 0 in Q30, at least one
// nonzero. Verbatim transliteration of celt_atan2p_norm (mathops.h:592).
func celtAtan2pNorm(y, x int32) int32 {
	// celt_sig_assert(x>=0 && y>=0)
	if y == 0 && x == 0 {
		return 0
	} else if y < x {
		return celtAtanNorm(fixedmath.SHR32(fixedmath.Frac_div32(y, x), 1))
	}
	// celt_sig_assert(y > 0)
	return 1073741824 /* 1.0f Q30 */ - celtAtanNorm(fixedmath.SHR32(fixedmath.Frac_div32(x, y), 1))
}

// StereoItheta is the FIXED_POINT stereo_itheta (vq.c:722): the mid/side energy
// angle (itheta, Q30 via celt_atan2p_norm) for the stereo pair X, Y over N
// samples. stereo != 0 selects the per-sample mid/side accumulation; otherwise it
// uses the plain per-channel norm energies.
func StereoItheta(X, Y []int32, stereo, N int) int32 {
	var Emid, Eside int32 // opus_val32
	if stereo != 0 {
		for i := 0; i < N; i++ {
			m := fixedmath.PSHR32(fixedmath.ADD32(X[i], Y[i]), normShift-13) // celt_norm
			s := fixedmath.PSHR32(fixedmath.SUB32(X[i], Y[i]), normShift-13)
			Emid = fixedmath.MAC16_16(Emid, fixedmath.EXTRACT16(m), fixedmath.EXTRACT16(m))
			Eside = fixedmath.MAC16_16(Eside, fixedmath.EXTRACT16(s), fixedmath.EXTRACT16(s))
		}
	} else {
		Emid += celtInnerProdNormShift(X, X, N)
		Eside += celtInnerProdNormShift(Y, Y, N)
	}
	mid := fixedmath.Celt_sqrt32(Emid)
	side := fixedmath.Celt_sqrt32(Eside)
	return celtAtan2pNorm(side, mid)
}
