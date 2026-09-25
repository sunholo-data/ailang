package celt

// Transliteration of libopus celt/kiss_fft.c and celt/_kiss_fft_guts.h (v1.6.1,
// commit 3da9f7a6) for the frozen FIXED_POINT + DISABLE_FLOAT_API + non-QEXT +
// non-CUSTOM_MODES configuration. The complex FFT core (opus_fft_impl and the
// kf_bfly2/3/4/5 butterflies) plus the inverse transform (opus_ifft) are the
// heart of the CELT inverse MDCT the decoder runs.
//
// Type mapping (celt/kiss_fft.h, celt/arch.h):
//   kiss_fft_scalar     = opus_int32  -> int32   (FFT working values)
//   kiss_twiddle_scalar = celt_coef = opus_int16 -> int16 (twiddles, non-QEXT)
//   COEF_SHIFT          = 16
//   kiss_fft_cpx        -> kissFFTCpx {r,i int32}
//   kiss_twiddle_cpx    -> kissTwiddleCpx {r,i int16}  (defined in modes.go)
//
// The per-stage fixed-point downshifts (fft_downshift) are NOT pre-baked into
// the mode data; they are applied here exactly as in _kiss_fft_guts.h, driven by
// the `downshift` budget threaded through opus_fft_impl (docs/hard-parts.md
// section 8). Every _ovflw add/sub is Go's defined signed wrap, matching the C
// uint32 round-trip; the S_MUL macros are the OPUS_FAST_INT64 int64 forms in
// internal/fixedmath (docs/hard-parts.md section 4).

import (
	"unsafe"

	"github.com/tphakala/simd/cint"

	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
)

// coefShift mirrors COEF_SHIFT for the non-QEXT build (celt/kiss_fft.h:52).
const coefShift = 16

// maxFFTRun bounds the largest contiguous per-block span a radix butterfly
// vectorizes: the radix-5 stage of the nfft=480 transform has m=96 (N=1 block of
// 96 complex). Scratch arrays sized to this live on the stack, so the cint paths
// stay zero-allocation. The butterfly dispatch treats it as a hard upper bound
// and falls back to the scalar path for any m above it, so a future mode with a
// longer run stays correct (scalar) instead of overrunning the stack scratch.
const maxFFTRun = 96

// minCintBfly3 / minCintBfly5 are the per-butterfly run-length thresholds below
// which the scalar inner loop wins: the cint call plus scratch setup does not
// amortize on short runs. The values are the empirical crossovers from
// BenchmarkBfly3_* / BenchmarkBfly5_* on the arm64 NEON kernels. radix-3
// vectorizes both C_MULs plus the add/sub/scale and crosses over near m=16
// (-25% at m=16, +2% at m=8). radix-5 vectorizes only its four C_MULs (the large
// ya/yb combine stays scalar and reads the results back from scratch), so it
// needs a longer run to pay back and crosses over near m=48 (-11% at m=48, +3.5%
// at m=24). radix-4 non-CUSTOM runs are only ever m<=8, below any crossover, so
// it is left fully scalar.
const (
	minCintBfly3 = 16
	minCintBfly5 = 48
)

// Compile-time guard: kissFFTCpx must stay exactly two adjacent int32 (8 bytes,
// no padding) for cpxAsInt32's reinterpret to []int32 to be sound. If a future
// field or padding (e.g. a QEXT variant) changes the size, one of these array
// lengths leaves the [0, ...] range and the build fails loudly rather than
// silently misreading memory. Both are [0]byte at the correct size.
var (
	_ [unsafe.Sizeof(kissFFTCpx{}) - 8]byte
	_ [8 - unsafe.Sizeof(kissFFTCpx{})]byte
)

// cpxAsInt32 reinterprets a run of kissFFTCpx as the interleaved [r,i,r,i,...]
// []int32 that the cint package operates on. kissFFTCpx is struct{r,i int32}
// (8 bytes, no padding), so a []kissFFTCpx is bit-for-bit a []int32 of twice the
// length; this is a view, not a copy. cint's data model is exactly this layout.
func cpxAsInt32(c []kissFFTCpx) []int32 {
	if len(c) == 0 {
		return nil
	}
	return unsafe.Slice((*int32)(unsafe.Pointer(&c[0])), 2*len(c))
}

// init precomputes the packed twiddle runs for every static FFT state before any
// FFT can run. Package-level vars (the states and the mode) are fully initialized
// before init functions execute, so mode48000_960.mdct.kfft is ready here. Doing
// this once, single-threaded, at init means the per-state runs are read-only for
// the process lifetime and the hot butterflies index them directly, with no atomic
// and no map hash.
func init() {
	for _, st := range mode48000_960.mdct.kfft {
		st.buildPackedTwiddles()
	}
}

// buildPackedTwiddles fills st.packedTw[stage] with the contiguous twiddle runs the
// radix-3/5 butterfly at that stage consumes: entries 0..1 for radix 3 and 0..3 for
// radix 5, where entry k is the twiddles[j*(k+1)*stride] gather. It mirrors the
// fstride/m walk in opusFFTImpl exactly, so the stage index it fills is the same one
// the driver passes back down to kfBfly3/kfBfly5; radix-2/4 stages and the m==1
// terminal consume no packed twiddles and leave their rows nil.
func (st *kissFFTState) buildPackedTwiddles() {
	var fstride [maxFactors]int
	shift := 0
	if st.shift > 0 {
		shift = st.shift
	}
	fstride[0] = 1
	L := 0
	var m, m2 int
	for {
		p := int(st.factors[2*L])
		m = int(st.factors[2*L+1])
		fstride[L+1] = fstride[L] * p
		L++
		if m == 1 {
			break
		}
	}
	m = int(st.factors[2*L-1])
	for i := L - 1; i >= 0; i-- {
		if i != 0 {
			m2 = int(st.factors[2*i-1])
		} else {
			m2 = 1
		}
		bf := fstride[i] << shift
		switch st.factors[2*i] {
		case 3:
			st.packedTw[i][0] = packTwiddleRun(st.twiddles, bf, m)
			st.packedTw[i][1] = packTwiddleRun(st.twiddles, 2*bf, m)
		case 5:
			st.packedTw[i][0] = packTwiddleRun(st.twiddles, bf, m)
			st.packedTw[i][1] = packTwiddleRun(st.twiddles, 2*bf, m)
			st.packedTw[i][2] = packTwiddleRun(st.twiddles, 3*bf, m)
			st.packedTw[i][3] = packTwiddleRun(st.twiddles, 4*bf, m)
		}
		m = m2
	}
}

// packTwiddleRun gathers the strided twiddles tw[j*stride] for j in 0..count-1 into
// the contiguous Q15 [tr,ti,tr,ti,...] int16 layout cint.Mul consumes.
func packTwiddleRun(tw []kissTwiddleCpx, stride, count int) []int16 {
	p := make([]int16, 2*count)
	for j := 0; j < count; j++ {
		c := tw[j*stride]
		p[2*j] = c.r
		p[2*j+1] = c.i
	}
	return p
}

// kissFFTCpx mirrors celt/kiss_fft.h kiss_fft_cpx in the FIXED_POINT build: r
// and i are kiss_fft_scalar (opus_int32).
type kissFFTCpx struct {
	r int32
	i int32
}

// S_MUL(a,b) = MULT16_32_Q15(b, a): a is a 32-bit FFT value, b a 16-bit twiddle
// (_kiss_fft_guts.h:61, non-QEXT). The int64 (OPUS_FAST_INT64) form is required.
func sMul(a int32, b int16) int32 { return fixedmath.MULT16_32_Q15(b, a) }

// S_MUL2(a,b) = MULT16_32_Q16(b, a) (_kiss_fft_guts.h:62). Used by the forward
// FFT input scaling.
func sMul2(a int32, b int16) int32 { return fixedmath.MULT16_32_Q16(b, a) }

// cMul mirrors C_MUL(m,a,b): m = a*b where a is a complex FFT value (int32) and
// b a complex twiddle (int16). (_kiss_fft_guts.h:65)
func cMul(a kissFFTCpx, b kissTwiddleCpx) kissFFTCpx {
	return kissFFTCpx{
		r: fixedmath.SUB32_ovflw(sMul(a.r, b.r), sMul(a.i, b.i)),
		i: fixedmath.ADD32_ovflw(sMul(a.r, b.i), sMul(a.i, b.r)),
	}
}

// cAdd mirrors C_ADD(res,a,b): res = a + b. (_kiss_fft_guts.h:84)
func cAdd(a, b kissFFTCpx) kissFFTCpx {
	return kissFFTCpx{fixedmath.ADD32_ovflw(a.r, b.r), fixedmath.ADD32_ovflw(a.i, b.i)}
}

// cSub mirrors C_SUB(res,a,b): res = a - b. (_kiss_fft_guts.h:87)
func cSub(a, b kissFFTCpx) kissFFTCpx {
	return kissFFTCpx{fixedmath.SUB32_ovflw(a.r, b.r), fixedmath.SUB32_ovflw(a.i, b.i)}
}

// cMulByScalar mirrors C_MULBYSCALAR(c,s): c.r=S_MUL(c.r,s); c.i=S_MUL(c.i,s),
// with s a 16-bit twiddle scalar. (_kiss_fft_guts.h:73)
func cMulByScalar(c kissFFTCpx, s int16) kissFFTCpx {
	return kissFFTCpx{sMul(c.r, s), sMul(c.i, s)}
}

// fft_downshift scales the working buffer down by up to `step` bits, spending
// from the remaining `total` budget, so the fixed-point FFT does not overflow.
// (celt/kiss_fft.c:539)
func fftDownshift(x []kissFFTCpx, N int, total *int, step int) {
	shift := fixedmath.IMIN(step, *total)
	*total -= shift
	if shift == 1 {
		for i := 0; i < N; i++ {
			x[i].r = fixedmath.SHR32(x[i].r, 1)
			x[i].i = fixedmath.SHR32(x[i].i, 1)
		}
	} else if shift > 0 {
		for i := 0; i < N; i++ {
			x[i].r = fixedmath.PSHR32(x[i].r, shift)
			x[i].i = fixedmath.PSHR32(x[i].i, shift)
		}
	}
}

// kf_bfly2 is the radix-2 butterfly. Under non-CUSTOM_MODES m is always 4 (the
// radix-2 stage follows a radix-4), so only that branch exists. fout is the
// working buffer positioned at the stage's Fout. (celt/kiss_fft.c:52)
func kfBfly2(fout []kissFFTCpx, m, N int) {
	_ = m
	// tw = QCONST32(0.7071067812f, COEF_SHIFT-1) stored in a celt_coef (int16).
	tw := int16(fixedmath.QCONST32(0.7071067812, coefShift-1))
	pos := 0
	for i := 0; i < N; i++ {
		// The radix-2 group is the fixed 8-element window fout[pos:pos+8]. Slicing
		// it once per block lets the constant-index accesses below drop their
		// per-element bounds checks (Fout2 = Fout + 4).
		w := fout[pos : pos+8 : pos+8]
		var t kissFFTCpx
		t = w[4]
		w[4] = cSub(w[0], t)
		w[0] = cAdd(w[0], t)

		t.r = sMul(fixedmath.ADD32_ovflw(w[5].r, w[5].i), tw)
		t.i = sMul(fixedmath.SUB32_ovflw(w[5].i, w[5].r), tw)
		w[5] = cSub(w[1], t)
		w[1] = cAdd(w[1], t)

		t.r = w[6].i
		t.i = fixedmath.NEG32_ovflw(w[6].r)
		w[6] = cSub(w[2], t)
		w[2] = cAdd(w[2], t)

		t.r = sMul(fixedmath.SUB32_ovflw(w[7].i, w[7].r), tw)
		t.i = sMul(fixedmath.NEG32_ovflw(fixedmath.ADD32_ovflw(w[7].i, w[7].r)), tw)
		w[7] = cSub(w[3], t)
		w[3] = cAdd(w[3], t)
		pos += 8
	}
}

// kf_bfly4 is the radix-4 butterfly, with a degenerate m==1 branch (all twiddles
// are 1) used for the final stage. fstride is fstride[i]<<st.shift; N and mm are
// the stage's fstride[i] and the outer m2. (celt/kiss_fft.c:108)
func kfBfly4(fout []kissFFTCpx, fstride int, st *kissFFTState, m, N, mm int) {
	if m == 1 {
		// Degenerate case where all the twiddles are 1.
		pos := 0
		for i := 0; i < N; i++ {
			// The block is the fixed 4-element window fout[pos:pos+4]; slicing it
			// drops the constant-index bounds checks below.
			w := fout[pos : pos+4 : pos+4]
			var scratch0, scratch1 kissFFTCpx
			scratch0 = cSub(w[0], w[2])
			w[0] = cAdd(w[0], w[2])
			scratch1 = cAdd(w[1], w[3])
			w[2] = cSub(w[0], scratch1)
			w[0] = cAdd(w[0], scratch1)
			scratch1 = cSub(w[1], w[3])

			w[1].r = fixedmath.ADD32_ovflw(scratch0.r, scratch1.i)
			w[1].i = fixedmath.SUB32_ovflw(scratch0.i, scratch1.r)
			w[3].r = fixedmath.SUB32_ovflw(scratch0.r, scratch1.i)
			w[3].i = fixedmath.ADD32_ovflw(scratch0.i, scratch1.r)
			pos += 4
		}
	} else {
		tw := st.twiddles
		for i := 0; i < N; i++ {
			pos := i * mm
			// The four radix-4 groups are the contiguous length-m windows at
			// pos, pos+m, pos+2m, pos+3m. Slicing them (with the pos+m+m form so
			// the prover keeps a known length m) drops the per-element fout bounds
			// checks in the j-loop; the strided twiddle gathers keep theirs.
			f0 := fout[pos : pos+m : pos+m]
			fm := fout[pos+m : pos+m+m : pos+m+m]
			fm2 := fout[pos+m+m : pos+m+m+m : pos+m+m+m]
			fm3 := fout[pos+m+m+m : pos+m+m+m+m : pos+m+m+m+m]
			tw1, tw2, tw3 := 0, 0, 0
			// m is guaranteed to be a multiple of 4.
			for j := range f0 {
				var scratch [6]kissFFTCpx
				scratch[0] = cMul(fm[j], tw[tw1])
				scratch[1] = cMul(fm2[j], tw[tw2])
				scratch[2] = cMul(fm3[j], tw[tw3])

				scratch[5] = cSub(f0[j], scratch[1])
				f0[j] = cAdd(f0[j], scratch[1])
				scratch[3] = cAdd(scratch[0], scratch[2])
				scratch[4] = cSub(scratch[0], scratch[2])
				fm2[j] = cSub(f0[j], scratch[3])
				tw1 += fstride
				tw2 += fstride * 2
				tw3 += fstride * 3
				f0[j] = cAdd(f0[j], scratch[3])

				fm[j].r = fixedmath.ADD32_ovflw(scratch[5].r, scratch[4].i)
				fm[j].i = fixedmath.SUB32_ovflw(scratch[5].i, scratch[4].r)
				fm3[j].r = fixedmath.SUB32_ovflw(scratch[5].r, scratch[4].i)
				fm3[j].i = fixedmath.ADD32_ovflw(scratch[5].i, scratch[4].r)
			}
		}
	}
}

// kf_bfly3 is the radix-3 butterfly. epi3.i is the fixed constant
// -QCONST32(0.86602540f, COEF_SHIFT-1). (celt/kiss_fft.c:180)
//
// Small runs do not amortize the cint call and scratch-setup overhead (the
// crossover sits near m=16 for this two-C_MUL stage, -25% at m=16, +2% at m=8),
// so they keep the scalar inner loop. See BenchmarkBfly3_*.
func kfBfly3(fout []kissFFTCpx, fstride int, st *kissFFTState, m, N, mm, stage int) {
	if m < minCintBfly3 || m > maxFFTRun {
		kfBfly3Scalar(fout, fstride, st, m, N, mm)
		return
	}
	kfBfly3Cint(fout, m, N, mm, st.packedTw[stage][0], st.packedTw[stage][1])
}

func kfBfly3Scalar(fout []kissFFTCpx, fstride int, st *kissFFTState, m, N, mm int) {
	epi3i := int16(-fixedmath.QCONST32(0.86602540, coefShift-1))
	tw := st.twiddles
	for i := 0; i < N; i++ {
		pos := i * mm
		b2 := pos + 2*m
		// Contiguous length-m windows at pos, pos+m, pos+2m; indexing them by k
		// drops the per-element fout bounds checks (the strided twiddle gathers
		// keep theirs).
		f0 := fout[pos : pos+m : pos+m]
		fm := fout[pos+m : pos+m+m : pos+m+m]
		fm2 := fout[b2 : b2+m : b2+m]
		tw1, tw2 := 0, 0
		for k := range f0 {
			var scratch [5]kissFFTCpx
			scratch[1] = cMul(fm[k], tw[tw1])
			scratch[2] = cMul(fm2[k], tw[tw2])

			scratch[3] = cAdd(scratch[1], scratch[2])
			scratch[0] = cSub(scratch[1], scratch[2])
			tw1 += fstride
			tw2 += fstride * 2

			// HALF_OF(x) = x>>1.
			fm[k].r = fixedmath.SUB32_ovflw(f0[k].r, scratch[3].r>>1)
			fm[k].i = fixedmath.SUB32_ovflw(f0[k].i, scratch[3].i>>1)

			scratch[0] = cMulByScalar(scratch[0], epi3i)

			f0[k] = cAdd(f0[k], scratch[3])

			fm2[k].r = fixedmath.ADD32_ovflw(fm[k].r, scratch[0].i)
			fm2[k].i = fixedmath.SUB32_ovflw(fm[k].i, scratch[0].r)

			fm[k].r = fixedmath.SUB32_ovflw(fm[k].r, scratch[0].i)
			fm[k].i = fixedmath.ADD32_ovflw(fm[k].i, scratch[0].r)
		}
	}
}

// kfBfly3Cint vectorizes the two C_MULs plus the add/sub/scale via cint, then
// finishes each block with the scalar cross-lane combine. The inner k-loop
// touches disjoint fout indices per k (pos+k, pos+m+k, pos+m2+k), and the two
// per-k C_MUL inputs (the m- and m2-groups) are read before any combine writes
// them, so a whole block splits cleanly into a vectorized phase and a scalar
// cross-lane combine. The three groups are the contiguous runs fout[pos:pos+3m].
// tw1p and tw2p are the precomputed twiddles[j*fstride] and twiddles[j*2*fstride]
// runs for this stage (see buildPackedTwiddles); the caller supplies them so the
// core stays a pure kernel with no plan lookup.
func kfBfly3Cint(fout []kissFFTCpx, m, N, mm int, tw1p, tw2p []int16) {
	epi3i := int16(-fixedmath.QCONST32(0.86602540, coefShift-1))

	var s1a, s2a, s3a, s0a [maxFFTRun]kissFFTCpx
	s1i := cpxAsInt32(s1a[:m])
	s2i := cpxAsInt32(s2a[:m])
	s3 := s3a[:m]
	s0 := s0a[:m]
	s3i := cpxAsInt32(s3)
	s0i := cpxAsInt32(s0)

	for i := 0; i < N; i++ {
		pos := i * mm
		// The three radix-3 groups are the contiguous length-m windows
		// fout[pos:pos+m], fout[pos+m:pos+2m], fout[pos+2m:pos+3m]. Slicing them
		// to a known length m up front lets the combine loop index each by k with
		// no per-element bounds check (the same idiom that already keeps s3[k]/
		// s0[k] check-free); fm/fm2 double as the C_MUL int32 sources.
		f0 := fout[pos : pos+m : pos+m]
		fm := fout[pos+m : pos+m+m : pos+m+m]
		fm2 := fout[pos+m+m : pos+m+m+m : pos+m+m+m]
		foutM := cpxAsInt32(fm)   // scratch[1] source
		foutM2 := cpxAsInt32(fm2) // scratch[2] source

		cint.Mul(s1i, foutM, tw1p)  // scratch[1] = C_MUL(fout[pos+m], tw1)
		cint.Mul(s2i, foutM2, tw2p) // scratch[2] = C_MUL(fout[pos+2m], tw2)
		cint.Add(s3i, s1i, s2i)     // scratch[3] = scratch[1] + scratch[2]
		cint.Sub(s0i, s1i, s2i)     // scratch[0] = scratch[1] - scratch[2]
		cint.MulByScalar(s0i, epi3i)

		for k := range f0 {
			// HALF_OF(x) = x>>1 (arithmetic). Computed from the pre-add fout[pos].
			fmr := fixedmath.SUB32_ovflw(f0[k].r, s3[k].r>>1)
			fmi := fixedmath.SUB32_ovflw(f0[k].i, s3[k].i>>1)

			f0[k] = cAdd(f0[k], s3[k])

			fm2[k].r = fixedmath.ADD32_ovflw(fmr, s0[k].i)
			fm2[k].i = fixedmath.SUB32_ovflw(fmi, s0[k].r)
			fm[k].r = fixedmath.SUB32_ovflw(fmr, s0[k].i)
			fm[k].i = fixedmath.ADD32_ovflw(fmi, s0[k].r)
		}
	}
}

// kf_bfly5 is the radix-5 butterfly. ya and yb are the fixed constants derived
// from the fifth roots of unity. (celt/kiss_fft.c:239)
//
// radix-5 vectorizes only its four C_MULs; the large ya/yb combine stays scalar
// and reads the results back from scratch, so it needs a longer run than radix-3
// to pay back and crosses over near m=48 (-11% at m=48, +3.5% at m=24). See
// BenchmarkBfly5_*.
func kfBfly5(fout []kissFFTCpx, fstride int, st *kissFFTState, m, N, mm, stage int) {
	if m < minCintBfly5 || m > maxFFTRun {
		kfBfly5Scalar(fout, fstride, st, m, N, mm)
		return
	}
	kfBfly5Cint(fout, m, N, mm, st.packedTw[stage][0], st.packedTw[stage][1], st.packedTw[stage][2], st.packedTw[stage][3])
}

// bfly5Consts returns the fifth-root-of-unity twiddle constants ya, yb shared by
// the scalar and cint radix-5 cores.
func bfly5Consts() (ya, yb kissTwiddleCpx) {
	ya = kissTwiddleCpx{
		r: int16(fixedmath.QCONST32(0.30901699, coefShift-1)),
		i: int16(-fixedmath.QCONST32(0.95105652, coefShift-1)),
	}
	yb = kissTwiddleCpx{
		r: int16(-fixedmath.QCONST32(0.80901699, coefShift-1)),
		i: int16(-fixedmath.QCONST32(0.58778525, coefShift-1)),
	}
	return ya, yb
}

func kfBfly5Scalar(fout []kissFFTCpx, fstride int, st *kissFFTState, m, N, mm int) {
	ya, yb := bfly5Consts()
	tw := st.twiddles
	for i := 0; i < N; i++ {
		b0 := i * mm
		b1 := b0 + m
		b2 := b0 + 2*m
		b3 := b0 + 3*m
		b4 := b0 + 4*m
		// Five contiguous length-m windows at b0..b4; indexing them by u drops the
		// per-element fout bounds checks (the strided tw gathers keep theirs).
		w0 := fout[b0 : b0+m : b0+m]
		w1 := fout[b1 : b1+m : b1+m]
		w2 := fout[b2 : b2+m : b2+m]
		w3 := fout[b3 : b3+m : b3+m]
		w4 := fout[b4 : b4+m : b4+m]
		for u := range w0 {
			var scratch [13]kissFFTCpx
			scratch[0] = w0[u]

			scratch[1] = cMul(w1[u], tw[u*fstride])
			scratch[2] = cMul(w2[u], tw[2*u*fstride])
			scratch[3] = cMul(w3[u], tw[3*u*fstride])
			scratch[4] = cMul(w4[u], tw[4*u*fstride])

			scratch[7] = cAdd(scratch[1], scratch[4])
			scratch[10] = cSub(scratch[1], scratch[4])
			scratch[8] = cAdd(scratch[2], scratch[3])
			scratch[9] = cSub(scratch[2], scratch[3])

			w0[u].r = fixedmath.ADD32_ovflw(w0[u].r, fixedmath.ADD32_ovflw(scratch[7].r, scratch[8].r))
			w0[u].i = fixedmath.ADD32_ovflw(w0[u].i, fixedmath.ADD32_ovflw(scratch[7].i, scratch[8].i))

			scratch[5].r = fixedmath.ADD32_ovflw(scratch[0].r, fixedmath.ADD32_ovflw(sMul(scratch[7].r, ya.r), sMul(scratch[8].r, yb.r)))
			scratch[5].i = fixedmath.ADD32_ovflw(scratch[0].i, fixedmath.ADD32_ovflw(sMul(scratch[7].i, ya.r), sMul(scratch[8].i, yb.r)))

			scratch[6].r = fixedmath.ADD32_ovflw(sMul(scratch[10].i, ya.i), sMul(scratch[9].i, yb.i))
			scratch[6].i = fixedmath.NEG32_ovflw(fixedmath.ADD32_ovflw(sMul(scratch[10].r, ya.i), sMul(scratch[9].r, yb.i)))

			w1[u] = cSub(scratch[5], scratch[6])
			w4[u] = cAdd(scratch[5], scratch[6])

			scratch[11].r = fixedmath.ADD32_ovflw(scratch[0].r, fixedmath.ADD32_ovflw(sMul(scratch[7].r, yb.r), sMul(scratch[8].r, ya.r)))
			scratch[11].i = fixedmath.ADD32_ovflw(scratch[0].i, fixedmath.ADD32_ovflw(sMul(scratch[7].i, yb.r), sMul(scratch[8].i, ya.r)))
			scratch[12].r = fixedmath.SUB32_ovflw(sMul(scratch[9].i, ya.i), sMul(scratch[10].i, yb.i))
			scratch[12].i = fixedmath.SUB32_ovflw(sMul(scratch[10].r, yb.i), sMul(scratch[9].r, ya.i))

			w2[u] = cAdd(scratch[11], scratch[12])
			w3[u] = cSub(scratch[11], scratch[12])
		}
	}
}

// kfBfly5Cint vectorizes the four C_MULs (scratch[1..4]) over the contiguous
// m-length groups via precomputed contiguous twiddle runs. The ya/yb combine
// mixes real and imag lanes with per-lane constants, which cint does not express,
// so it stays a scalar pass reading the C_MUL results from scratch. The four
// C_MUL inputs (groups f1..f4) are fully read before the combine overwrites any
// group, so there is no read-after-write hazard.
// tw1p..tw4p are the precomputed twiddles[j*k*fstride] runs (k=1..4) for this
// stage (see buildPackedTwiddles); the caller supplies them so the core stays a
// pure kernel with no plan lookup.
func kfBfly5Cint(fout []kissFFTCpx, m, N, mm int, tw1p, tw2p, tw3p, tw4p []int16) {
	ya, yb := bfly5Consts()

	var s1a, s2a, s3a, s4a [maxFFTRun]kissFFTCpx
	s1 := s1a[:m]
	s2 := s2a[:m]
	s3 := s3a[:m]
	s4 := s4a[:m]
	s1i := cpxAsInt32(s1)
	s2i := cpxAsInt32(s2)
	s3i := cpxAsInt32(s3)
	s4i := cpxAsInt32(s4)

	for i := 0; i < N; i++ {
		b0 := i * mm
		b1 := b0 + m
		b2 := b0 + 2*m
		b3 := b0 + 3*m
		b4 := b0 + 4*m

		// The five radix-5 groups are the contiguous length-m windows starting at
		// b0..b4. Slicing them to a known length m lets the combine loop index each
		// by u with no per-element bounds check; w1..w4 double as the C_MUL sources.
		w0 := fout[b0 : b0+m : b0+m]
		w1 := fout[b1 : b1+m : b1+m]
		w2 := fout[b2 : b2+m : b2+m]
		w3 := fout[b3 : b3+m : b3+m]
		w4 := fout[b4 : b4+m : b4+m]

		cint.Mul(s1i, cpxAsInt32(w1), tw1p)
		cint.Mul(s2i, cpxAsInt32(w2), tw2p)
		cint.Mul(s3i, cpxAsInt32(w3), tw3p)
		cint.Mul(s4i, cpxAsInt32(w4), tw4p)

		for u := range w0 {
			var scratch [13]kissFFTCpx
			scratch[0] = w0[u]
			scratch[1] = s1[u]
			scratch[2] = s2[u]
			scratch[3] = s3[u]
			scratch[4] = s4[u]

			scratch[7] = cAdd(scratch[1], scratch[4])
			scratch[10] = cSub(scratch[1], scratch[4])
			scratch[8] = cAdd(scratch[2], scratch[3])
			scratch[9] = cSub(scratch[2], scratch[3])

			w0[u].r = fixedmath.ADD32_ovflw(scratch[0].r, fixedmath.ADD32_ovflw(scratch[7].r, scratch[8].r))
			w0[u].i = fixedmath.ADD32_ovflw(scratch[0].i, fixedmath.ADD32_ovflw(scratch[7].i, scratch[8].i))

			scratch[5].r = fixedmath.ADD32_ovflw(scratch[0].r, fixedmath.ADD32_ovflw(sMul(scratch[7].r, ya.r), sMul(scratch[8].r, yb.r)))
			scratch[5].i = fixedmath.ADD32_ovflw(scratch[0].i, fixedmath.ADD32_ovflw(sMul(scratch[7].i, ya.r), sMul(scratch[8].i, yb.r)))

			scratch[6].r = fixedmath.ADD32_ovflw(sMul(scratch[10].i, ya.i), sMul(scratch[9].i, yb.i))
			scratch[6].i = fixedmath.NEG32_ovflw(fixedmath.ADD32_ovflw(sMul(scratch[10].r, ya.i), sMul(scratch[9].r, yb.i)))

			w1[u] = cSub(scratch[5], scratch[6])
			w4[u] = cAdd(scratch[5], scratch[6])

			scratch[11].r = fixedmath.ADD32_ovflw(scratch[0].r, fixedmath.ADD32_ovflw(sMul(scratch[7].r, yb.r), sMul(scratch[8].r, ya.r)))
			scratch[11].i = fixedmath.ADD32_ovflw(scratch[0].i, fixedmath.ADD32_ovflw(sMul(scratch[7].i, yb.r), sMul(scratch[8].i, ya.r)))
			scratch[12].r = fixedmath.SUB32_ovflw(sMul(scratch[9].i, ya.i), sMul(scratch[10].i, yb.i))
			scratch[12].i = fixedmath.SUB32_ovflw(sMul(scratch[10].r, yb.i), sMul(scratch[9].r, ya.i))

			w2[u] = cAdd(scratch[11], scratch[12])
			w3[u] = cSub(scratch[11], scratch[12])
		}
	}
}

// opus_fft_impl is the shared in-place FFT engine used by both the forward and
// inverse transforms (direction is handled by the caller via twiddle
// conjugation and bitrev). downshift is the fixed-point scaling budget spent by
// fft_downshift across the stages. (celt/kiss_fft.c:562)
func opusFFTImpl(st *kissFFTState, fout []kissFFTCpx, downshift int) {
	var fstride [maxFactors]int
	// st.shift can be -1.
	shift := 0
	if st.shift > 0 {
		shift = st.shift
	}

	fstride[0] = 1
	L := 0
	var m, m2 int
	for {
		p := int(st.factors[2*L])
		m = int(st.factors[2*L+1])
		fstride[L+1] = fstride[L] * p
		L++
		if m == 1 {
			break
		}
	}
	m = int(st.factors[2*L-1])
	for i := L - 1; i >= 0; i-- {
		if i != 0 {
			m2 = int(st.factors[2*i-1])
		} else {
			m2 = 1
		}
		switch st.factors[2*i] {
		case 2:
			fftDownshift(fout, st.nfft, &downshift, 1)
			kfBfly2(fout, m, fstride[i])
		case 4:
			fftDownshift(fout, st.nfft, &downshift, 2)
			kfBfly4(fout, fstride[i]<<shift, st, m, fstride[i], m2)
		case 3:
			fftDownshift(fout, st.nfft, &downshift, 2)
			kfBfly3(fout, fstride[i]<<shift, st, m, fstride[i], m2, i)
		case 5:
			fftDownshift(fout, st.nfft, &downshift, 3)
			kfBfly5(fout, fstride[i]<<shift, st, m, fstride[i], m2, i)
		}
		m = m2
	}
	fftDownshift(fout, st.nfft, &downshift, downshift)
}

// opus_ifft is the inverse complex FFT used by the CELT decoder path. It
// bit-reverses the input, conjugates, runs the shared engine with no extra
// downscaling (downshift 0), then conjugates back. fin and fout must not alias.
// (celt/kiss_fft.c:638)
func opusIFFT(st *kissFFTState, fin, fout []kissFFTCpx) {
	// Bit-reverse the input.
	for i := 0; i < st.nfft; i++ {
		fout[st.bitrev[i]] = fin[i]
	}
	for i := 0; i < st.nfft; i++ {
		fout[i].i = -fout[i].i
	}
	opusFFTImpl(st, fout, 0)
	for i := 0; i < st.nfft; i++ {
		fout[i].i = -fout[i].i
	}
}

// opus_fft is the forward complex FFT. It bit-reverses and pre-scales the input
// by st.scale (S_MUL2), then runs the shared engine with the scale_shift-1
// budget. fin and fout must not alias. (celt/kiss_fft.c:615)
func opusFFT(st *kissFFTState, fin, fout []kissFFTCpx) {
	scale := st.scale
	scaleShift := st.scaleShift - 1
	// Bit-reverse the input.
	for i := 0; i < st.nfft; i++ {
		x := fin[i]
		fout[st.bitrev[i]].r = sMul2(x.r, scale)
		fout[st.bitrev[i]].i = sMul2(x.i, scale)
	}
	opusFFTImpl(st, fout, scaleShift)
}

// InverseFFT drives opus_ifft on mode48000_960's kfft[idx] (idx 0..3, with nfft
// 480/240/120/60) over the complex input inR/inI (each length nfft) and returns
// the complex output (each length nfft). It is exported only so the refc cgo
// differential harness (internal/reftest/oracle) can drive the pure-Go FFT
// against libopus; it is not part of the decoder API.
func InverseFFT(idx int, inR, inI []int32) (outR, outI []int32) {
	st := mode48000_960.mdct.kfft[idx]
	fin := make([]kissFFTCpx, st.nfft)
	fout := make([]kissFFTCpx, st.nfft)
	for i := 0; i < st.nfft; i++ {
		fin[i] = kissFFTCpx{inR[i], inI[i]}
	}
	opusIFFT(st, fin, fout)
	outR = make([]int32, st.nfft)
	outI = make([]int32, st.nfft)
	for i := 0; i < st.nfft; i++ {
		outR[i] = fout[i].r
		outI[i] = fout[i].i
	}
	return outR, outI
}

// ForwardFFT drives opus_fft on mode48000_960's kfft[idx] over the complex input
// inR/inI (each length nfft) and returns the complex output. Exported only for
// the refc cgo differential harness; not part of the decoder API.
func ForwardFFT(idx int, inR, inI []int32) (outR, outI []int32) {
	st := mode48000_960.mdct.kfft[idx]
	fin := make([]kissFFTCpx, st.nfft)
	fout := make([]kissFFTCpx, st.nfft)
	for i := 0; i < st.nfft; i++ {
		fin[i] = kissFFTCpx{inR[i], inI[i]}
	}
	opusFFT(st, fin, fout)
	outR = make([]int32, st.nfft)
	outI = make([]int32, st.nfft)
	for i := 0; i < st.nfft; i++ {
		outR[i] = fout[i].r
		outI[i] = fout[i].i
	}
	return outR, outI
}

// FFTStateNFFT returns the FFT length of mode48000_960's kfft[idx] (480/240/
// 120/60 for idx 0..3). Exported for the refc differential harness.
func FFTStateNFFT(idx int) int { return mode48000_960.mdct.kfft[idx].nfft }

// FFTImplWithDownshift runs opus_fft_impl directly on mode48000_960's kfft[idx]
// with an explicit downshift budget over the given complex buffer (already in
// working order, not bit-reversed). It exists so the refc differential harness
// can drive the per-stage fixed-point downshift path (docs/hard-parts.md section
// 8) directly against libopus; it is not part of the decoder API.
func FFTImplWithDownshift(idx int, inR, inI []int32, downshift int) (outR, outI []int32) {
	st := mode48000_960.mdct.kfft[idx]
	buf := make([]kissFFTCpx, st.nfft)
	for i := 0; i < st.nfft; i++ {
		buf[i] = kissFFTCpx{inR[i], inI[i]}
	}
	opusFFTImpl(st, buf, downshift)
	outR = make([]int32, st.nfft)
	outI = make([]int32, st.nfft)
	for i := 0; i < st.nfft; i++ {
		outR[i] = buf[i].r
		outI[i] = buf[i].i
	}
	return outR, outI
}
