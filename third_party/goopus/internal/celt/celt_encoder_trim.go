package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// This file ports the two CELT encoder allocation-trim analysis stages from
// libopus celt/celt_encoder.c (v1.6.1): alloc_trim_analysis (:865) and
// stereo_analysis (:957). Both are file-static in C and neither touches the range
// coder. The frozen build config is FIXED_POINT + DISABLE_FLOAT_API +
// OPUS_FAST_INT64, so:
//
//   - opus_val16 is int16 and celt_norm / celt_ener / celt_glog / opus_val32 are
//     int32. Every C local declared opus_val16 therefore TRUNCATES TO 16 BITS on
//     each assignment, and the ports below reproduce that truncation at exactly
//     the same points (trim, sum, minXC, logXC, logXC2).
//   - the `#ifndef DISABLE_FLOAT_API` block at :934-942 is dead; the live branch is
//     `(void)analysis;`, so the AnalysisInfo tonality_slope trim adjustment does NOT
//     apply and no AnalysisInfo is passed in Go.
//   - the `#ifdef FUZZING` block at :951-953 is dead.
//   - the `arch` parameter only selects a celt_inner_prod_norm_shift SIMD variant,
//     which is bit-identical to the C fallback, so it is dropped in Go.

// AllocTrimWitness records alloc_trim_analysis intermediates so the differential
// test can prove (non-vacuously) that each documented branch actually fired. It is
// pure observation: nothing recorded here ever feeds back into the ported
// arithmetic, and production callers pass a nil witness.
type AllocTrimWitness struct {
	StereoBlock  bool  // the C==2 inter-channel correlation block ran (:884)
	MinXCLoop    bool  // the i=8..intensity minXC refinement loop body ran (:899)
	Sum          int16 // sum after the clamp at :897 (Q10)
	MinXC        int16 // minXC after the clamp at :906 (Q10)
	LogXC        int16 // logXC after the Q20->Q8 fixup at :914
	LogXC2       int16 // logXC2 after the Q20->Q8 fixup at :915
	TrimBase     int16 // trim after :874-920 (rate branch + stereo adjustment)
	DiffPre      int32 // diff after the :923-928 accumulation, before the division
	Diff         int32 // diff after `diff /= C*(end-1)` at :929
	TiltRaw      int32 // the unclamped tilt term inside the MAX32/MIN32 at :931
	Trim         int16 // final trim, just before trim_index = PSHR32(trim, 8) (:945)
	TrimIndexRaw int   // trim_index before the IMAX/IMIN saturation at :949
}

// allocTrimAnalysis is alloc_trim_analysis (celt_encoder.c:865) for FIXED_POINT:
// pick the allocation trim (0..10) from the equivalent bitrate, the inter-channel
// correlation, the spectral tilt, the surround trim and the transient estimate,
// and update the encoder's running stereo_saving estimate.
//
// Signature is 1:1 with C except that `AnalysisInfo *analysis` and `int arch` are
// dropped (both inert in the frozen config, see the file comment). stereoSaving is
// the in/out opus_val16 the C mutates at :919 (C==2 only).
func allocTrimAnalysis(m *celtMode, X, bandLogE []int32, end, LM, C, N0 int,
	stereoSaving *int16, tfEstimate int16, intensity int, surroundTrim, equivRate int32) int {
	return allocTrimAnalysisObserved(m, X, bandLogE, end, LM, C, N0, stereoSaving,
		tfEstimate, intensity, surroundTrim, equivRate, nil)
}

// allocTrimAnalysisObserved is the implementation of allocTrimAnalysis with an
// optional (nil-able) witness hook for the differential test.
func allocTrimAnalysisObserved(m *celtMode, X, bandLogE []int32, end, LM, C, N0 int,
	stereoSaving *int16, tfEstimate int16, intensity int, surroundTrim, equivRate int32,
	w *AllocTrimWitness) int {
	var i int
	diff := int32(0)
	var c int
	var trimIndex int
	trim := fixedmath.QCONST16(5.0, 8)
	var logXC, logXC2 int16
	// At low bitrate, reducing the trim seems to help. At higher bitrates, it's
	// less clear what's best, so we're keeping it as it was before, at least for
	// now.
	if equivRate < 64000 {
		trim = fixedmath.QCONST16(4.0, 8)
	} else if equivRate < 80000 {
		frac := (equivRate - 64000) >> 10
		trim = int16(int32(fixedmath.QCONST16(4.0, 8)) + int32(fixedmath.QCONST16(1.0/16.0, 8))*frac)
	}
	if C == 2 {
		sum := int16(0) // Q10
		var minXC int16 // Q10
		// Compute inter-channel correlation for low frequencies.
		for i = 0; i < 8; i++ {
			var partial int32
			partial = celtInnerProdNormShift(X[int(m.eBands[i])<<LM:], X[N0+(int(m.eBands[i])<<LM):],
				(int(m.eBands[i+1])-int(m.eBands[i]))<<LM)
			sum = fixedmath.ADD16(sum, fixedmath.EXTRACT16(fixedmath.SHR32(partial, 18)))
		}
		sum = int16(fixedmath.MULT16_16_Q15(fixedmath.QCONST16(1.0/8, 15), sum))
		sum = int16(fixedmath.MIN16(int32(fixedmath.QCONST16(1.0, 10)), fixedmath.ABS16(sum)))
		minXC = sum
		for i = 8; i < intensity; i++ {
			var partial int32
			partial = celtInnerProdNormShift(X[int(m.eBands[i])<<LM:], X[N0+(int(m.eBands[i])<<LM):],
				(int(m.eBands[i+1])-int(m.eBands[i]))<<LM)
			minXC = int16(fixedmath.MIN16(int32(minXC),
				fixedmath.ABS16(fixedmath.EXTRACT16(fixedmath.SHR32(partial, 18)))))
			if w != nil {
				w.MinXCLoop = true
			}
		}
		minXC = int16(fixedmath.MIN16(int32(fixedmath.QCONST16(1.0, 10)), fixedmath.ABS16(minXC)))
		// Mid-side savings estimations based on the LF average.
		logXC = fixedmath.Celt_log2(fixedmath.QCONST32(1.001, 20) - fixedmath.MULT16_16(sum, sum))
		// Mid-side savings estimations based on min correlation.
		logXC2 = int16(fixedmath.MAX16(int32(fixedmath.HALF16(logXC)),
			int32(fixedmath.Celt_log2(fixedmath.QCONST32(1.001, 20)-fixedmath.MULT16_16(minXC, minXC)))))
		// Compensate for Q20 vs Q14 input and convert output to Q8.
		logXC = int16(fixedmath.PSHR32(int32(logXC)-int32(fixedmath.QCONST16(6.0, 10)), 10-8))
		logXC2 = int16(fixedmath.PSHR32(int32(logXC2)-int32(fixedmath.QCONST16(6.0, 10)), 10-8))

		trim = int16(int32(trim) + fixedmath.MAX16(-int32(fixedmath.QCONST16(4.0, 8)),
			fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.75, 15), logXC)))
		*stereoSaving = int16(fixedmath.MIN16(int32(*stereoSaving)+int32(fixedmath.QCONST16(0.25, 8)),
			-int32(fixedmath.HALF16(logXC2))))
		if w != nil {
			w.StereoBlock = true
			w.Sum = sum
			w.MinXC = minXC
			w.LogXC = logXC
			w.LogXC2 = logXC2
		}
	}
	if w != nil {
		w.TrimBase = trim
	}

	// Estimate spectral tilt.
	c = 0
	for {
		for i = 0; i < end-1; i++ {
			diff += fixedmath.SHR32(bandLogE[i+c*m.nbEBands], 5) * int32(2+2*i-end)
		}
		c++
		if c >= C {
			break
		}
	}
	if w != nil {
		w.DiffPre = diff
	}
	// C truncation toward zero on a possibly negative dividend; Go's / matches.
	// This is deliberately NOT a shift (which would floor).
	diff /= int32(C * (end - 1))
	// :931 genuinely mixes SHR32 (arithmetic shift, floors) with /6 (truncates
	// toward zero). Both are kept exactly as C spells them.
	trim = int16(int32(trim) - fixedmath.MAX32(-int32(fixedmath.QCONST16(2.0, 8)),
		fixedmath.MIN32(int32(fixedmath.QCONST16(2.0, 8)),
			fixedmath.SHR32(diff+fixedmath.QCONST32(1.0, dbShift-5), dbShift-13)/6)))
	// C spells :932 SHR16(surround_trim, DB_SHIFT-8), but surround_trim is a
	// celt_glog (int32), so the type-generic C macro degenerates to a 32-bit
	// arithmetic shift. Using fixedmath.SHR16 here would wrongly narrow to int16.
	trim = int16(int32(trim) - fixedmath.SHR32(surroundTrim, dbShift-8))
	trim = int16(int32(trim) - 2*int32(fixedmath.SHR16(tfEstimate, 14-8)))
	// :934-942 (#ifndef DISABLE_FLOAT_API) is dead; the live branch is
	// `(void)analysis;`, so no tonality_slope adjustment is applied here.

	// PSHR32 arithmetic-shifts, so it FLOORS a negative trim. That is what C does
	// and what is transliterated here, but note it is not observable: it can only
	// differ from a truncating divide when trim+128 < 0, and every such raw index is
	// <= 0, so the IMAX(0, ...) below maps both forms to 0. (Verified exhaustively
	// over all 65536 opus_val16 trim values; no differential test can pin this line.)
	trimIndex = int(fixedmath.PSHR32(int32(trim), 8))
	if w != nil {
		w.Diff = diff
		w.TiltRaw = fixedmath.SHR32(diff+fixedmath.QCONST32(1.0, dbShift-5), dbShift-13) / 6
		w.Trim = trim
		w.TrimIndexRaw = trimIndex
	}
	trimIndex = fixedmath.IMAX(0, fixedmath.IMIN(10, trimIndex))
	// :951-953 (#ifdef FUZZING) is dead.
	return trimIndex
}

// stereoAnalysis is stereo_analysis (celt_encoder.c:957) for FIXED_POINT: compare
// the L1 norm of the L/R signal against the L1 norm of the M/S signal (plus the
// cost of the extra thetas) to decide whether to code the frame as dual stereo.
// Stateless and read-only on X (2*N0 celt_norm).
//
// Note the band loop is hard-coded to i<13 in C, so eBands[0..13] are read no
// matter what `end` is; the function does not take an `end` at all.
func stereoAnalysis(m *celtMode, X []int32, LM, N0 int) int {
	return stereoAnalysisObserved(m, X, LM, N0, nil)
}

// StereoWitness records the two sides of stereo_analysis's final comparison (:985)
// so the differential test can measure how close a frame sits to the decision
// boundary. Pure observation; production callers pass a nil witness.
type StereoWitness struct {
	Thetas int
	Lhs    int32 // MULT16_32_Q15((eBands[13]<<(LM+1))+thetas, sumMS)
	Rhs    int32 // MULT16_32_Q15(eBands[13]<<(LM+1), sumLR)
}

// stereoAnalysisObserved is the implementation of stereoAnalysis with an optional
// (nil-able) witness hook for the differential test.
func stereoAnalysisObserved(m *celtMode, X []int32, LM, N0 int, w *StereoWitness) int {
	var i int
	var thetas int
	sumLR := int32(1) // EPSILON
	sumMS := int32(1) // EPSILON

	// Use the L1 norm to model the entropy of the L/R signal vs the M/S signal.
	for i = 0; i < 13; i++ {
		var j int
		for j = int(m.eBands[i]) << LM; j < int(m.eBands[i+1])<<LM; j++ {
			var L, R, M, S int32
			// We cast to 32-bit first because of the -32768 case.
			L = fixedmath.SHR32(X[j], normShift-14)
			R = fixedmath.SHR32(X[N0+j], normShift-14)
			M = fixedmath.ADD32(L, R)
			S = fixedmath.SUB32(L, R)
			sumLR = fixedmath.ADD32(sumLR, fixedmath.ADD32(fixedmath.ABS32(L), fixedmath.ABS32(R)))
			sumMS = fixedmath.ADD32(sumMS, fixedmath.ADD32(fixedmath.ABS32(M), fixedmath.ABS32(S)))
		}
	}
	sumMS = fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.707107, 15), sumMS)
	thetas = 13
	// We don't need thetas for lower bands with LM<=1.
	if LM <= 1 {
		thetas -= 8
	}
	if w != nil {
		w.Thetas = thetas
		w.Lhs = fixedmath.MULT16_32_Q15(int16((int(m.eBands[13])<<(LM+1))+thetas), sumMS)
		w.Rhs = fixedmath.MULT16_32_Q15(int16(int(m.eBands[13])<<(LM+1)), sumLR)
	}
	if fixedmath.MULT16_32_Q15(int16((int(m.eBands[13])<<(LM+1))+thetas), sumMS) >
		fixedmath.MULT16_32_Q15(int16(int(m.eBands[13])<<(LM+1)), sumLR) {
		return 1
	}
	return 0
}

// --- Differential-test entry points (used by internal/reftest/oracle) --------

// AllocTrimAnalysis runs alloc_trim_analysis over x (C*N0 celt_norm) and bandLogE
// (C*nbEBands celt_glog) at the frozen 48 kHz mode, threading stereoSaving in and
// out. Returns trim_index, the (possibly mutated) stereo_saving and the witness of
// the branches the call took.
func AllocTrimAnalysis(x, bandLogE []int32, end, LM, C, N0 int, stereoSaving, tfEstimate int16,
	intensity int, surroundTrim, equivRate int32) (int, int16, AllocTrimWitness) {
	m := &mode48000_960
	var w AllocTrimWitness
	ss := stereoSaving
	trimIndex := allocTrimAnalysisObserved(m, x, bandLogE, end, LM, C, N0, &ss, tfEstimate,
		intensity, surroundTrim, equivRate, &w)
	return trimIndex, ss, w
}

// StereoAnalysis runs stereo_analysis over x (2*N0 celt_norm) at the frozen 48 kHz
// mode and returns the dual-stereo decision (0/1) plus the witness of the final
// comparison, so the test can measure the decision margin.
func StereoAnalysis(x []int32, LM, N0 int) (int, StereoWitness) {
	var w StereoWitness
	d := stereoAnalysisObserved(&mode48000_960, x, LM, N0, &w)
	return d, w
}
