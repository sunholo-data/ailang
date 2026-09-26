package opusenc

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// This file ports the three generalized-DTX helpers from opus_encoder.c that the
// frozen forced-CELT-only + DISABLE_FLOAT_API configuration needs:
// is_digital_silence (:1060), compute_frame_energy (:1080, FIXED_POINT) and
// decide_dtx_mode (:1115). The wiring that calls them lives in encode.go.
//
// # What generalized DTX actually does in this configuration
//
// The DTX decision in opus_encode_frame_native (:2564-2576) is gated by
// `st->use_dtx && !st->silk_mode.useDTX`. In the DISABLE_FLOAT_API build
// silk_mode.useDTX is set unconditionally, before mode selection, to
// `use_dtx && !is_silence` (:1463). The guard therefore collapses to
//
//	use_dtx && is_silence
//
// so decide_dtx_mode runs ONLY on a frame that is EXACT digital silence
// (celt_maxabs16 == 0), and :1911-1913 forces activity == 0 in that case. The
// energy-based activity at :1929 and peak_signal_energy are computed but are
// DEAD for the decision here: no near-silence frame can ever trigger DTX in this
// config. peak_signal_energy is nonetheless tracked and compared against the C
// oracle for state-hash defense-in-depth (it sits inside the very function we
// modify to ship DTX) and SILK-forward-compatibility, NOT because its value
// gates anything today. See state.go for the full rationale and the width_mem
// distinction.
//
// nb_no_activity_ms_Q1 is genuinely LIVE: it is the consecutive-silence counter
// decide_dtx_mode advances, and it is now part of the compared encoder state.

// DTX bound constants, from silk/define.h. NB_SPEECH_FRAMES_BEFORE_DTX is the
// number of 20 ms frames of silence tolerated before DTX may start (200 ms), and
// MAX_CONSECUTIVE_DTX caps a DTX run (400 ms) before one active frame is forced.
// Both bounds are expressed in the SILK headers for 20 ms frames, so
// decide_dtx_mode converts them to milliseconds-in-Q1 as 20*2 per frame.
const (
	nbSpeechFramesBeforeDTX = 10 // eq 200 ms
	maxConsecutiveDTX       = 20 // eq 400 ms
)

// celtMaxabs16 is celt_maxabs16 (celt/mathops.h:86): max |x[i]| as opus_val32
// over x[:n]. celt's own copy is unexported and lives in a different package;
// this is a local reimplementation so DTX does not force CELT internals into the
// exported surface. opus_res is opus_int16 in the frozen config (no ENABLE_RES24)
// and celt_maxabs_res is a #define for celt_maxabs16, so this is exactly the
// function opus_encode_native calls at :1246 and :1315.
func celtMaxabs16(x []int16, n int) int32 {
	var maxval, minval int16
	for i := 0; i < n; i++ {
		if x[i] > maxval {
			maxval = x[i]
		}
		if x[i] < minval {
			minval = x[i]
		}
	}
	return fixedmath.MAX32(fixedmath.EXTEND32(maxval), -fixedmath.EXTEND32(minval))
}

// isDigitalSilence is is_digital_silence (opus_encoder.c:1060) for FIXED_POINT:
// a frame is silent iff its peak magnitude is exactly zero. lsb_depth is unused
// in the fixed-point branch (:1071 `(void)lsb_depth`), so it is not a parameter.
func isDigitalSilence(pcm []int16, frameSize, channels int) bool {
	return celtMaxabs16(pcm, frameSize*channels) == 0
}

// computeFrameEnergy is compute_frame_energy (opus_encoder.c:1080, FIXED_POINT):
// the mean per-sample energy of the frame, kept in the same fixed-point position
// as the input via a data-dependent shift that keeps the MAC from overflowing
// opus_val32. opus_res is opus_int16 so RES2INT16 is the identity and the samples
// enter the MAC directly.
func computeFrameEnergy(pcm []int16, frameSize, channels int) int32 {
	length := frameSize * channels

	// Max amplitude in the signal (RES2INT16 is the identity here).
	sampleMax := celtMaxabs16(pcm, length)

	// Right shift needed in the MAC to avoid an overflow. 1+sampleMax is >= 1, so
	// the celt_ilog2 argument is always positive.
	maxShift := fixedmath.Celt_ilog2(int32(length))
	shift := fixedmath.IMAX(0, (fixedmath.Celt_ilog2(1+sampleMax)<<1)+maxShift-28)

	// Energy, accumulated with each square pre-shifted so the sum stays in range.
	var energy int32
	for i := 0; i < length; i++ {
		energy += fixedmath.SHR32(fixedmath.MULT16_16(pcm[i], pcm[i]), shift)
	}

	// Normalize by the frame size, then left-shift back to the original position.
	energy /= int32(length)
	energy = fixedmath.SHL32(energy, shift)

	return energy
}

// decideDTXMode is decide_dtx_mode (opus_encoder.c:1115). It advances the
// consecutive-no-activity counter (in ms, Q1) and reports whether THIS frame may
// be dropped to a DTX (TOC-only) packet. It mutates *nbNoActivityMsQ1 in place,
// exactly as the C mutates *nb_no_activity_ms_Q1.
//
// activity is non-zero for a speech/music frame. In this configuration it is only
// ever called with activity == 0 (see the file comment), but the full branch is
// ported so the counter evolves identically to C on any input.
func decideDTXMode(activity int, nbNoActivityMsQ1 *int, frameSizeMsQ1 int) bool {
	if activity == 0 {
		// The number of consecutive DTX frames must stay within the allowed bounds.
		// The bounds are defined for 20 ms frames, so they are converted to
		// milliseconds (Q1) before the comparisons.
		*nbNoActivityMsQ1 += frameSizeMsQ1
		if *nbNoActivityMsQ1 > nbSpeechFramesBeforeDTX*20*2 {
			if *nbNoActivityMsQ1 <= (nbSpeechFramesBeforeDTX+maxConsecutiveDTX)*20*2 {
				// Valid frame for DTX.
				return true
			}
			*nbNoActivityMsQ1 = nbSpeechFramesBeforeDTX * 20 * 2
		}
	} else {
		*nbNoActivityMsQ1 = 0
	}
	return false
}
