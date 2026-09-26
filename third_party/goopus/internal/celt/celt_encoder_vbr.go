package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// This file ports the CELT encoder's VBR rate controller from libopus
// celt/celt_encoder.c (v1.6.1): compute_vbr (:1604). It is file-static in C, it
// touches the range coder not at all, and -- this is worth spelling out -- it
// touches NO st-> field either: it is a PURE function of its arguments. All of
// the cross-frame VBR state (vbr_reservoir, vbr_drift, vbr_offset, vbr_count)
// lives in the caller at :2435-2533, not here.
//
// Frozen build config (FIXED_POINT + DISABLE_FLOAT_API + OPUS_FAST_INT64):
//
//   - ARG_QEXT(x) expands to nothing (celt.h:49), so there is NO enable_qext
//     parameter in this build and the `#ifdef ENABLE_QEXT` bins override at
//     :1686-1688 is dead.
//   - the two `#ifndef DISABLE_FLOAT_API` blocks (:1630-1633, the analysis
//     activity deduction, and :1655-1669, the tonality boost) are DEAD; the live
//     branch is `(void)analysis; (void)pitch_change;` at :1671-1672. Neither the
//     AnalysisInfo pointer nor pitch_change can affect the result, so neither is
//     part of the Go signature (the differential test proves the pitch_change
//     drop is safe by feeding the C oracle random pitch_change values and
//     requiring the target not to move).
//   - opus_val16 is int16 and opus_val32 / celt_glog are int32. stereo_saving,
//     max_frac, tf_calibration, amount and tvbr_factor are all opus_val16 and so
//     TRUNCATE TO 16 BITS on every assignment, and every MULT16_16 /
//     MULT16_16_Q15 / MULT16_32_Q15 / DIV32_16 operand is cast to opus_val16 by
//     the macro itself. The port reproduces each of those truncations at exactly
//     the C's point. target is opus_int32 throughout.
//
// Two superficially identical divides are NOT the same and must not be unified:
// :1679 uses `target/4` (C division, truncates toward zero) while :1691 uses
// `target>>2` (arithmetic shift, floors). They disagree on a negative target, and
// base_target genuinely goes negative at low rates (:2448), so both forms are
// transliterated exactly as C spells them. The :1679 form IS observable at the
// output (a differential test that swaps it for >>2 fails). The :1691 form is
// NOT: IMAX(floor_depth, target>>2) feeds IMIN(target, floor_depth), and for
// target < 0 both floor(target/4) and trunc(target/4) are >= target, so the IMIN
// always returns target regardless; for target >= 0 the two forms are identical.
// No differential test can pin that line, so do not read the passing sweep as
// evidence for it. (Same situation as the PSHR32 in alloc_trim_analysis.)

// ComputeVBRWitness records compute_vbr intermediates and the side each clamp
// took, so the differential test can prove (non-vacuously) that every branch and
// every clamp of celt_encoder.c:1604-1716 actually fired. It is pure observation:
// nothing recorded here feeds back into the ported arithmetic, and production
// callers pass a nil witness.
type ComputeVBRWitness struct {
	CodedBands          int   // :1622
	CodedBins           int   // :1623-1625
	StereoBlock         bool  // the C==2 stereo-savings block ran (:1635)
	StereoSavingClamped bool  // the MIN16 at :1644 actually clamped stereo_saving
	MaxFrac             int16 // max_frac after :1643 (Q15, truncated to opus_val16)
	StereoFracWon       bool  // MULT16_32_Q15(max_frac,target) won the MIN32 at :1646
	SurroundBlock       bool  // the has_surround_mask && !lfe block ran (:1675)
	SurroundQuarterWon  bool  // target/4 won the IMAX at :1679
	FloorDepth          int32 // floor_depth after :1690, before the IMAX at :1691
	FloorQuarterWon     bool  // target>>2 won the IMAX at :1691
	FloorApplied        bool  // the IMIN at :1692 actually lowered target
	ConstrainedBlock    bool  // the constrained-VBR block ran (:1698)
	TemporalBlock       bool  // the temporal-VBR block ran (:1703)
	Amount              int16 // amount after :1707 (truncated to opus_val16)
	TvbrFactor          int16 // tvbr_factor after :1708 (truncated to opus_val16)
	DoubleCapped        bool  // 2*base_target won the IMIN at :1713
}

// computeVbr is compute_vbr (celt_encoder.c:1604) for the frozen config: turn the
// base target (in 8th bits per frame) into the frame's VBR target, applying the
// stereo savings, the dynalloc boost, the transient boost, the surround-masking
// boost, the maxDepth floor, the constrained-VBR damping and the temporal-VBR
// factor, then cap at twice the base target.
//
// Signature is 1:1 with C except that `AnalysisInfo *analysis` and `pitch_change`
// are dropped (both provably inert in the frozen config, see the file comment).
// stereoSaving and tfEstimate are opus_val16; maxDepth, surroundMasking and
// temporalVbr are celt_glog (Q24). bitrate is the caller's equiv_rate (:2458).
func computeVbr(m *celtMode, baseTarget int32, LM int, bitrate int32, lastCodedBands, C, intensity,
	constrainedVbr int, stereoSaving int16, totBoost int, tfEstimate int16, maxDepth int32,
	lfe, hasSurroundMask int, surroundMasking, temporalVbr int32) int32 {
	return computeVbrObserved(m, baseTarget, LM, bitrate, lastCodedBands, C, intensity,
		constrainedVbr, stereoSaving, totBoost, tfEstimate, maxDepth, lfe, hasSurroundMask,
		surroundMasking, temporalVbr, nil)
}

// computeVbrObserved is the implementation of computeVbr with an optional
// (nil-able) witness hook for the differential test.
func computeVbrObserved(m *celtMode, baseTarget int32, LM int, bitrate int32, lastCodedBands, C, intensity,
	constrainedVbr int, stereoSaving int16, totBoost int, tfEstimate int16, maxDepth int32,
	lfe, hasSurroundMask int, surroundMasking, temporalVbr int32,
	w *ComputeVBRWitness) int32 {
	// The target rate in 8th bits per frame.
	var target int32
	var codedBins int
	var codedBands int
	var tfCalibration int16
	var nbEBands int
	var eBands []int16

	nbEBands = m.nbEBands
	eBands = m.eBands

	codedBands = nbEBands
	if lastCodedBands != 0 {
		codedBands = lastCodedBands
	}
	codedBins = int(eBands[codedBands]) << LM
	if C == 2 {
		codedBins += int(eBands[fixedmath.IMIN(intensity, codedBands)]) << LM
	}

	target = baseTarget

	// :1630-1633 (#ifndef DISABLE_FLOAT_API) is dead: no analysis activity
	// deduction is applied.

	// Stereo savings.
	if C == 2 {
		var codedStereoBands int
		var codedStereoDof int
		var maxFrac int16
		codedStereoBands = fixedmath.IMIN(intensity, codedBands)
		codedStereoDof = (int(eBands[codedStereoBands]) << LM) - codedStereoBands
		// Maximum fraction of the bits we can save if the signal is mono.
		maxFrac = fixedmath.DIV32_16(
			fixedmath.MULT16_16(fixedmath.QCONST16(0.8, 15), int16(codedStereoDof)),
			int16(codedBins))
		if w != nil {
			w.StereoBlock = true
			w.MaxFrac = maxFrac
			w.StereoSavingClamped = int32(stereoSaving) > int32(fixedmath.QCONST16(1.0, 8))
		}
		stereoSaving = int16(fixedmath.MIN16(int32(stereoSaving), int32(fixedmath.QCONST16(1.0, 8))))
		if w != nil {
			w.StereoFracWon = fixedmath.MULT16_32_Q15(maxFrac, target) <
				fixedmath.SHR32(fixedmath.MULT16_16(
					int16(int32(stereoSaving)-int32(fixedmath.QCONST16(0.1, 8))),
					int16(codedStereoDof<<bitRes)), 8)
		}
		// stereo_saving-QCONST16(0.1f,8) is evaluated in int by C and then cast
		// back to opus_val16 by MULT16_16 itself, so it wraps for a stereo_saving
		// near INT16_MIN. int16() reproduces that wrap.
		target -= fixedmath.MIN32(fixedmath.MULT16_32_Q15(maxFrac, target),
			fixedmath.SHR32(fixedmath.MULT16_16(
				int16(int32(stereoSaving)-int32(fixedmath.QCONST16(0.1, 8))),
				int16(codedStereoDof<<bitRes)), 8))
	}
	// Boost the rate according to dynalloc (minus the dynalloc average for calibration).
	target += int32(totBoost - (19 << LM))
	// Apply transient boost, compensating for average boost.
	tfCalibration = fixedmath.QCONST16(0.044, 14)
	// tf_estimate-tf_calibration is evaluated in int and then truncated to
	// opus_val16 by MULT16_32_Q15's own (opus_val16)(a) cast.
	target += fixedmath.SHL32(fixedmath.MULT16_32_Q15(
		int16(int32(tfEstimate)-int32(tfCalibration)), target), 1)

	// :1655-1669 (#ifndef DISABLE_FLOAT_API) is dead: no tonality boost, and
	// pitch_change is unused (:1671-1672).

	if hasSurroundMask != 0 && lfe == 0 {
		surroundTarget := target + fixedmath.SHR32(fixedmath.MULT16_16(
			int16(fixedmath.SHR32(surroundMasking, dbShift-10)),
			int16(codedBins<<bitRes)), 10)
		if w != nil {
			w.SurroundBlock = true
			w.SurroundQuarterWon = target/4 > surroundTarget
		}
		// C division, truncates toward zero. NOT the >>2 used at :1691.
		target = int32(fixedmath.IMAX(int(target/4), int(surroundTarget)))
	}

	{
		var floorDepth int32
		var bins int
		bins = int(eBands[nbEBands-2]) << LM
		// :1686-1688 (#ifdef ENABLE_QEXT) is dead: bins is never the shortMdctSize form.
		floorDepth = fixedmath.SHR32(fixedmath.MULT16_32_Q15(
			int16((C*bins)<<bitRes), maxDepth), dbShift-15)
		if w != nil {
			w.FloorDepth = floorDepth
			w.FloorQuarterWon = (target >> 2) > floorDepth
		}
		// Arithmetic shift: floors a negative target. NOT the /4 used at :1679.
		// Unobservable at the output (see the file comment), but kept verbatim.
		floorDepth = int32(fixedmath.IMAX(int(floorDepth), int(target>>2)))
		if w != nil {
			w.FloorApplied = floorDepth < target
		}
		target = int32(fixedmath.IMIN(int(target), int(floorDepth)))
	}

	// Make VBR less aggressive for constrained VBR because we can't keep a higher
	// bitrate for long. Needs tuning.
	if (hasSurroundMask == 0 || lfe != 0) && constrainedVbr != 0 {
		target = baseTarget + fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.67, 15), target-baseTarget)
		if w != nil {
			w.ConstrainedBlock = true
		}
	}

	if hasSurroundMask == 0 && tfEstimate < fixedmath.QCONST16(0.2, 14) {
		var amount int16
		var tvbrFactor int16
		amount = int16(fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.0000031, 30),
			int16(fixedmath.IMAX(0, fixedmath.IMIN(32000, int(int32(96000)-bitrate))))))
		tvbrFactor = int16(fixedmath.SHR32(fixedmath.MULT16_16(
			int16(fixedmath.SHR32(temporalVbr, dbShift-10)), amount), 10))
		target += fixedmath.MULT16_32_Q15(tvbrFactor, target)
		if w != nil {
			w.TemporalBlock = true
			w.Amount = amount
			w.TvbrFactor = tvbrFactor
		}
	}

	// Don't allow more than doubling the rate.
	if w != nil {
		w.CodedBands = codedBands
		w.CodedBins = codedBins
		w.DoubleCapped = 2*baseTarget < target
	}
	target = int32(fixedmath.IMIN(int(2*baseTarget), int(target)))

	return target
}

// --- Differential-test entry points (used by internal/reftest/oracle) --------

// ComputeVBR runs compute_vbr (celt_encoder.c:1604) at the frozen 48 kHz mode and
// returns the VBR target in 8th bits per frame plus the witness of the branches
// and clamps the call took. There is no analysis / pitch_change argument: both are
// inert in the frozen config (see the file comment).
func ComputeVBR(baseTarget int32, LM int, bitrate int32, lastCodedBands, C, intensity,
	constrainedVbr int, stereoSaving int16, totBoost int, tfEstimate int16, maxDepth int32,
	lfe, hasSurroundMask int, surroundMasking, temporalVbr int32) (int32, ComputeVBRWitness) {
	var w ComputeVBRWitness
	target := computeVbrObserved(&mode48000_960, baseTarget, LM, bitrate, lastCodedBands, C,
		intensity, constrainedVbr, stereoSaving, totBoost, tfEstimate, maxDepth, lfe,
		hasSurroundMask, surroundMasking, temporalVbr, &w)
	return target, w
}
