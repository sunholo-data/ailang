// LPC analysis/synthesis primitives ported from libopus celt/celt_lpc.c for the
// frozen build config (FIXED_POINT, OPUS_FAST_INT64, non-SMALL_FOOTPRINT). These
// are used exclusively by the CELT packet-loss concealment (celt_decode_lost):
// _celt_lpc (Levinson-Durbin + 16-bit coefficient fitting), celt_fir (the
// analysis filter that maps signal to excitation), celt_iir (the synthesis
// filter that maps excitation back to signal), and _celt_autocorr (windowed
// autocorrelation feeding _celt_lpc). Bit-exactness against celt/celt_lpc.c is
// validated by the function-level differential tests in
// internal/reftest/oracle (plc_test.go).
//
// Type mapping (celt/arch.h): opus_val16 = int16, opus_val32/celt_sig = int32.
//
// SMALL_FOOTPRINT is NOT defined in the oracle, so celt_fir/celt_iir compile to
// the unrolled xcorr_kernel form there. The Go ports use the scalar recurrences,
// which are integer-identical: celt_fir accumulates an order-independent integer
// MAC sum (two's-complement addition is associative), and celt_iir feeds back the
// same SROUND16-rounded outputs the unrolled path does. celt_iir's returned mem
// is set the way the unrolled path leaves it (the truncated last outputs); the
// PLC discards mem, so only _y is load-bearing there, but matching mem keeps the
// function-level differential test exact.

package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// mult32_32_q16 is MULT32_32_Q16 for the OPUS_FAST_INT64 build (fixed_generic.h:62):
// (opus_val32)SHR((opus_int64)a*(opus_int64)b, 16).
func mult32_32_q16(a, b int32) int32 { return int32((int64(a) * int64(b)) >> 16) }

// mult32_32_32 is MULT32_32_32 (fixed_generic.h:172): a plain 32-bit product that
// wraps on overflow. Go's signed multiply is defined to wrap, matching the C.
func mult32_32_32(a, b int32) int32 { return a * b }

// div32 is DIV32 (fixed_generic.h:203): truncating 32-bit signed division.
// Delegates to fixedmath.DIV32 so there is a single implementation to keep
// bit-exact.
func div32(a, b int32) int32 { return fixedmath.DIV32(a, b) }

// sround16 is SROUND16 (fixed_generic.h:141): EXTRACT16(SATURATE(PSHR32(x,a), 32767)).
// Round-to-nearest right shift, saturated to the int16 magnitude range.
// Delegates to fixedmath.SROUND16 so there is a single implementation to keep
// bit-exact.
func sround16(x int32, a int) int16 {
	return fixedmath.SROUND16(x, a)
}

// celtMaxabs16 is celt_maxabs16 (mathops.h:86): max |x[i]| as opus_val32.
func celtMaxabs16(x []int16, len_ int) int32 {
	var maxval, minval int16
	// Bound the window against len(x), the same bound the original per-sample x[i]
	// enforced, then range over it so the reduction is check-free. The guard keeps
	// a mis-sized caller panicking deterministically rather than letting the
	// two-index reslice widen silently into spare capacity (x[:len_] bounds against
	// cap, not len). One check per call replaces one per element; the arithmetic is
	// untouched, so the result stays bit-identical.
	if len_ < 0 || len_ > len(x) {
		panic("celt: celtMaxabs16: len_ out of range")
	}
	for _, v := range x[:len_] {
		if v > maxval {
			maxval = v
		}
		if v < minval {
			minval = v
		}
	}
	return fixedmath.MAX32(fixedmath.EXTEND32(maxval), -fixedmath.EXTEND32(minval))
}

// celtMaxabsRes is celt_maxabs_res (mathops.h:118) with an explicit base offset,
// so a caller can express C's celt_maxabs_res(pcm + off, len). ENABLE_RES24 is
// not defined in the frozen config, so celt_maxabs_res is a #define for
// celt_maxabs16 and opus_res is opus_int16: this is exactly celtMaxabs16 over
// x[xOff:xOff+len_]. celt_encode_with_ec:1972-1973 uses both the offset form
// (st->overlap_max) and the plain form (sample_max); note the lengths there are
// scaled by C == stream_channels, not CC == channels.
func celtMaxabsRes(x []int16, xOff, len_ int) int32 {
	var maxval, minval int16
	// Bound the [xOff, xOff+len_) window against len(x), the same bound the original
	// per-sample x[xOff+i] enforced, then range over it check-free. The guard keeps
	// a mis-sized caller panicking deterministically rather than letting the
	// two-index reslice widen silently into spare capacity (it bounds against cap,
	// not len). One check per call replaces one per element; the arithmetic is
	// untouched, so the result stays bit-identical.
	if xOff < 0 || len_ < 0 || xOff+len_ > len(x) {
		panic("celt: celtMaxabsRes: window exceeds operand length")
	}
	for _, v := range x[xOff : xOff+len_] {
		if v > maxval {
			maxval = v
		}
		if v < minval {
			minval = v
		}
	}
	return fixedmath.MAX32(fixedmath.EXTEND32(maxval), -fixedmath.EXTEND32(minval))
}

// celtMaxabs32 is celt_maxabs32 (mathops.h:122): max |x[i]| for opus_val32 input.
func celtMaxabs32(x []int32, xOff, len_ int) int32 {
	var maxval, minval int32
	// Bound the [xOff, xOff+len_) window against len(x), the same bound the original
	// per-sample x[xOff+i] enforced, then range over it so the reduction is
	// check-free (xOff+i is otherwise not provably in range). The guard keeps a
	// mis-sized caller panicking deterministically rather than letting the two-index
	// reslice widen silently into spare capacity (it bounds against cap, not len).
	// len_==0 yields an empty window and the original zero result. The arithmetic is
	// unchanged, so the result stays bit-identical.
	if xOff < 0 || len_ < 0 || xOff+len_ > len(x) {
		panic("celt: celtMaxabs32: window exceeds operand length")
	}
	for _, v := range x[xOff : xOff+len_] {
		if v > maxval {
			maxval = v
		}
		if v < minval {
			minval = v
		}
	}
	return fixedmath.MAX32(maxval, -minval)
}

// celtLPC is _celt_lpc (celt_lpc.c:37) for FIXED_POINT + OPUS_FAST_INT64: run the
// Levinson-Durbin recursion over the autocorrelation ac[0..p], then fit the Q25
// int32 coefficients into Q12 int16 with bandwidth expansion (the silk_LPC_fit /
// silk_bwexpander_32 logic), falling back to A(z)=1 if they will not fit in 10
// iterations. Writes p coefficients into lpcOut.
func celtLPC(lpcOut []int16, ac []int32, p int) {
	var lpc [celtLpcOrder]int32
	error := ac[0]

	for i := 0; i < p; i++ {
		lpc[i] = 0
	}
	if ac[0] != 0 {
		for i := 0; i < p; i++ {
			// Sum up this iteration's reflection coefficient.
			var rr int32
			var acc int64
			for j := 0; j < i; j++ {
				acc += int64(lpc[j]) * int64(ac[i-j])
			}
			rr = int32(fixedmath.SHR64(acc, 31))
			rr += fixedmath.SHR32(ac[i+1], 6)
			r := -fixedmath.Frac_div32(fixedmath.SHL32(rr, 6), error)
			// Update LPC coefficients and total error.
			lpc[i] = fixedmath.SHR32(r, 6)
			for j := 0; j < (i+1)>>1; j++ {
				tmp1 := lpc[j]
				tmp2 := lpc[i-1-j]
				lpc[j] = tmp1 + fixedmath.MULT32_32_Q31(r, tmp2)
				lpc[i-1-j] = tmp2 + fixedmath.MULT32_32_Q31(r, tmp1)
			}

			error -= fixedmath.MULT32_32_Q31(fixedmath.MULT32_32_Q31(r, r), error)
			// Bail out once we get 30 dB gain.
			if error <= fixedmath.SHR32(ac[0], 10) {
				break
			}
		}
	}
	// Convert the int32 lpcs to int16 and ensure there are no wrap-arounds. This
	// reuses the logic in silk_LPC_fit() and silk_bwexpander_32().
	var iter, idx int
	for iter = 0; iter < 10; iter++ {
		maxabs := int32(0)
		for i := 0; i < p; i++ {
			absval := fixedmath.ABS32(lpc[i])
			if absval > maxabs {
				maxabs = absval
				idx = i
			}
		}
		maxabs = fixedmath.PSHR32(maxabs, 13) // Q25->Q12

		if maxabs > 32767 {
			maxabs = fixedmath.MIN32(maxabs, 163838)
			chirpQ16 := fixedmath.QCONST32(0.999, 16) - div32(fixedmath.SHL32(maxabs-32767, 14),
				fixedmath.SHR32(mult32_32_32(maxabs, int32(idx+1)), 2))
			chirpMinusOneQ16 := chirpQ16 - 65536

			// Apply bandwidth expansion.
			for i := 0; i < p-1; i++ {
				lpc[i] = mult32_32_q16(chirpQ16, lpc[i])
				chirpQ16 += fixedmath.PSHR32(mult32_32_32(chirpQ16, chirpMinusOneQ16), 16)
			}
			lpc[p-1] = mult32_32_q16(chirpQ16, lpc[p-1])
		} else {
			break
		}
	}

	if iter == 10 {
		// If the coeffs still do not fit into the 16 bit range after 10
		// iterations, fall back to the A(z)=1 filter. As in the C, only lpcOut[0]
		// is written; lpcOut[1..p-1] retain their prior contents.
		lpcOut[0] = 4096 // Q12
	} else {
		for i := 0; i < p; i++ {
			lpcOut[i] = fixedmath.EXTRACT16(fixedmath.PSHR32(lpc[i], 13)) // Q25->Q12
		}
	}
}

// celtFir is celt_fir_c (celt_lpc.c:146): the FIR analysis filter y[i] =
// x[i] + sum_j num[j]*x[i-1-j], run in the SIG_SHIFT signal domain with a
// SROUND16 output. x is indexed x[xOff + i + j - ord] so the caller supplies the
// ord samples of history before xOff. y receives N outputs. (The C's num is
// reversed into rnum internally; the resulting integer MAC sum is
// order-independent, so this scalar form is bit-identical to the unrolled C.)
func celtFir(x []int16, xOff int, num []int16, y []int16, N, ord int) {
	rnum := make([]int16, ord)
	for i := 0; i < ord; i++ {
		rnum[i] = num[ord-i-1]
	}
	for i := 0; i < N; i++ {
		sum := fixedmath.SHL32(fixedmath.EXTEND32(x[xOff+i]), sigShift)
		for j := 0; j < ord; j++ {
			sum = fixedmath.MAC16_16(sum, rnum[j], x[xOff+i+j-ord])
		}
		y[i] = sround16(sum, sigShift)
	}
}

// celtIir is celt_iir (celt_lpc.c:194): the all-pole synthesis filter
// _y[i] = _x[i] - sum_j den[j]*SROUND16(_y[i-1-j]), run in the SIG_SHIFT signal
// domain. mem[0..ord-1] carries the ord rounded output-history samples (most
// recent first); it is updated on return the way the non-SMALL_FOOTPRINT C leaves
// it (the truncated last outputs). x and y may alias the same slice (the PLC
// filters in place).
func celtIir(x []int32, den []int16, y []int32, N, ord int, mem []int16) {
	m := make([]int16, ord)
	copy(m, mem[:ord])
	for i := 0; i < N; i++ {
		sum := x[i]
		for j := 0; j < ord; j++ {
			sum -= fixedmath.MULT16_16(den[j], m[j])
		}
		for j := ord - 1; j >= 1; j-- {
			m[j] = m[j-1]
		}
		m[0] = sround16(sum, sigShift)
		y[i] = sum
	}
	// Match the non-SMALL_FOOTPRINT mem output: mem[i] = (opus_val16)_y[N-1-i],
	// a plain int16 truncation of the unrounded output.
	for i := 0; i < ord; i++ {
		mem[i] = int16(y[N-1-i])
	}
}

// celtAutocorr is _celt_autocorr (celt_lpc.c:284) for FIXED_POINT: optionally
// window the n input samples, compute the (lag+1) autocorrelation coefficients
// into ac via celt_pitch_xcorr plus the ragged tail, applying the fixed-point
// pre-scale and post-scale, and return the total scaling shift. window may be nil
// when overlap == 0.
func celtAutocorr(x []int16, ac []int32, window []int16, overlap, lag, n int, sc *scratch) int {
	fastN := n - lag
	var shift int
	var xptr []int16
	xx := alloc(&sc.autocorrXX, n) // VARDECL(opus_val16, xx)
	if overlap == 0 {
		xptr = x
	} else {
		for i := 0; i < n; i++ {
			xx[i] = x[i]
		}
		for i := 0; i < overlap; i++ {
			w := window[i] // COEF2VAL16 identity in this config
			xx[i] = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(x[i], w))
			xx[n-i-1] = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(x[n-i-1], w))
		}
		xptr = xx
	}
	{
		ac0Shift := fixedmath.Celt_ilog2(int32(n + (n >> 4)))
		ac0 := int32(1 + (n << 7))
		if n&1 != 0 {
			ac0 += fixedmath.SHR32(fixedmath.MULT16_16(xptr[0], xptr[0]), ac0Shift)
		}
		for i := n & 1; i < n; i += 2 {
			ac0 += fixedmath.SHR32(fixedmath.MULT16_16(xptr[i], xptr[i]), ac0Shift)
			ac0 += fixedmath.SHR32(fixedmath.MULT16_16(xptr[i+1], xptr[i+1]), ac0Shift)
		}
		// Consider the effect of rounding-to-nearest when scaling the signal.
		ac0 += fixedmath.SHR32(ac0, 7)

		shift = fixedmath.Celt_ilog2(ac0) - 30 + ac0Shift + 1
		shift = shift / 2
		if shift > 0 {
			for i := 0; i < n; i++ {
				xx[i] = fixedmath.EXTRACT16(fixedmath.PSHR32(fixedmath.EXTEND32(xptr[i]), shift))
			}
			xptr = xx
		} else {
			shift = 0
		}
	}
	celtPitchXcorr(xptr, xptr, ac, fastN, lag+1)
	for k := 0; k <= lag; k++ {
		d := int32(0)
		for i := k + fastN; i < n; i++ {
			d = fixedmath.MAC16_16(d, xptr[i], xptr[i-k])
		}
		ac[k] += d
	}
	shift = 2 * shift
	if shift <= 0 {
		ac[0] += fixedmath.SHL32(int32(1), -shift)
	}
	if ac[0] < 268435456 {
		shift2 := 29 - fixedmath.EC_ILOG(uint32(ac[0]))
		for i := 0; i <= lag; i++ {
			ac[i] = fixedmath.SHL32(ac[i], shift2)
		}
		shift -= shift2
	} else if ac[0] >= 536870912 {
		shift2 := 1
		if ac[0] >= 1073741824 {
			shift2++
		}
		for i := 0; i <= lag; i++ {
			ac[i] = fixedmath.SHR32(ac[i], shift2)
		}
		shift += shift2
	}
	return shift
}

// --- Differential-test entry points (used by internal/reftest/oracle) --------
// These exported wrappers give the cgo differential harness clean Go signatures
// over the C-shaped internal functions, matching the convention already used by
// RenormaliseVector / DenormaliseBands. They are not used by the decoder itself.

// CeltLPC runs celtLPC over ac[0..p] and returns the p fitted Q12 coefficients.
func CeltLPC(ac []int32, p int) []int16 {
	lpc := make([]int16, p)
	celtLPC(lpc, ac, p)
	return lpc
}

// CeltFir runs celtFir: xfull holds ord history samples then N samples (length
// N+ord); it returns the N filtered outputs.
func CeltFir(xfull []int16, N, ord int, num []int16) []int16 {
	y := make([]int16, N)
	celtFir(xfull, ord, num, y, N, ord)
	return y
}

// CeltIir runs celtIir on x (length N) with den/mem (length ord) and returns the
// N outputs and the updated mem (a fresh copy; the input mem is not mutated).
func CeltIir(x []int32, den []int16, N, ord int, mem []int16) ([]int32, []int16) {
	y := make([]int32, N)
	memOut := append([]int16(nil), mem...)
	celtIir(x, den, y, N, ord, memOut)
	return y, memOut
}

// CeltAutocorr runs celtAutocorr and returns the lag+1 autocorrelation values
// and the scaling shift. window may be nil when overlap is 0.
func CeltAutocorr(x []int16, window []int16, overlap, lag, n int) ([]int32, int) {
	ac := make([]int32, lag+1)
	var sc scratch
	shift := celtAutocorr(x, ac, window, overlap, lag, n, &sc)
	return ac, shift
}
