/* Copyright (c) 2010-2011 Xiph.Org Foundation, Skype Limited
   Written by Jean-Marc Valin and Koen Vos */
/*
   Redistribution and use in source and binary forms, with or without
   modification, are permitted provided that the following conditions
   are met:

   - Redistributions of source code must retain the above copyright
   notice, this list of conditions and the following disclaimer.

   - Redistributions in binary form must reproduce the above copyright
   notice, this list of conditions and the following disclaimer in the
   documentation and/or other materials provided with the distribution.

   THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
   ``AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
   LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
   A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER
   OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
   EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
   PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
   PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
   LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
   NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
   SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package opusenc

// pcm.go is the PCM-DOMAIN FRONT END of opus_encode_frame_native
// (opus_encoder.c:1855): everything that happens to the caller's int16 samples
// between entering the encoder and handing a buffer to CELT, plus the two
// cross-frame buffers that front end owns (st->hp_mem and st->delay_buffer) and
// the stereo-width crossfade state (st->hybrid_stereo_width_Q14).
//
// The per-frame order is LOAD-BEARING and is spelled out in PCMFront /
// UpdateDelayBuffer / PCMFades below. In the C the three are separated by work
// that is either dead (the SILK block) or independent of the PCM buffers (the
// CELT ctl setup), which is why they can be split here without changing a bit.
//
// Frozen config: OPUS_APPLICATION_AUDIO, forced MODE_CELT_ONLY, FIXED_POINT, no
// RES24 (so opus_res is opus_int16 and RES_SHIFT is 0), no QEXT. Everything here is
// expressed AT Fs, which is any of 8/12/16/24/48 kHz: encoder_buffer = Fs/100 and
// delay_compensation = Fs/250, so total_buffer = Fs/250 and the delay-compensation
// ring is live at every rate (480/192 samples at 48 kHz, 80/32 at 8 kHz). dc_reject
// and stereo_fade both take Fs and scale with it; stereo_fade additionally decimates
// the CELT window by IMAX(1, 48000/Fs), because the window belongs to the 48 kHz
// mode CELT always runs.

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"
)

const (
	// resShift is RES_SHIFT (celt/arch.h:165) for the non-ENABLE_RES24 build:
	// opus_res is opus_int16 and carries no extra fractional bits.
	resShift = 0

	// dcRejectCutoffHz is the cutoff opus_encode_frame_native HARD-CODES at the
	// dc_reject call site (opus_encoder.c:2008). It is NOT the cutoff_Hz that
	// UpdateVariableHPSmth2 computes: that one only feeds hp_cutoff, which is the
	// OPUS_APPLICATION_VOIP path and is dead here.
	dcRejectCutoffHz = 3

	// variableHPSmthCoef2Q16 is SILK_FIX_CONST(VARIABLE_HP_SMTH_COEF2, 16) with
	// VARIABLE_HP_SMTH_COEF2 = 0.015f (silk/tuning_parameters.h:63).
	//
	// DERIVATION. The C literal is f-SUFFIXED, so SILK_FIX_CONST's product
	// ((C)*((opus_int64)1<<(Q))) is evaluated in FLOAT, not double: by the usual
	// arithmetic conversions the int64 is converted to float, not the float to
	// double. float32(0.015) is 0.0149999996647238731384277343750 and 1<<16 is a
	// power of two, so the product is EXACTLY 983.03997802734375; + 0.5 (double)
	// = 983.53997802734375; truncated to opus_int32 = 983. Reading the literal as
	// a double gives 983.04 + 0.5 = 983.54 -> 983 as well, so the two readings
	// agree HERE. The suffix is recorded because it is the rule (it caused two
	// real packet divergences in earlier checkpoints), not because it changes the
	// answer this time.
	//
	// NOT OBSERVABLE IN THE FROZEN CONFIG, AND SAY SO HONESTLY. On the CELT-only
	// path hp_freq_smth1 (:1970) is identically equal to the value
	// opus_encoder_init seeds st->variable_HP_smth2_Q15 with (:319), so the delta
	// silk_SMLAWB is handed is always ZERO and silk_SMLAWB(a, 0, c) returns a
	// unchanged FOR ANY c. No differential test in this configuration can
	// distinguish a wrong coefficient from a right one; what the differential
	// does prove is that the state is preserved exactly, which is what closes the
	// state hash. The moment SILK becomes reachable hp_freq_smth1 starts moving
	// and this constant goes live, so it is derived correctly now rather than
	// guessed later.
	variableHPSmthCoef2Q16 = 983
)

// FloatLiteralConsts exposes the constants in this package that are derived from a
// float literal in the C, so the oracle can assert each one against the value the C
// COMPILER produces from that literal rather than against a hand-derivation in a
// comment. It is the internal/opusenc counterpart of celt.FloatLiteralConsts and is
// consumed by TestFloatLiteralConstsMatchC.
//
// The rule it enforces: SILK_FIX_CONST / QCONST32 / GCONST evaluate their product IN
// THE PRECISION OF THE LITERAL, so an "f"-suffixed literal is rounded to float32
// before the shift and an unsuffixed one is not. The two readings differ by a few
// ULPs, which is enough to flip a comparison and diverge an entire packet; that was a
// real bug twice in CP8c. Every new float-derived constant belongs in this map.
func FloatLiteralConsts() map[string]int32 {
	return map[string]int32{
		"SILK_FIX_CONST(VARIABLE_HP_SMTH_COEF2,16)": variableHPSmthCoef2Q16,
	}
}

// DCReject is dc_reject (opus_encoder.c:479), the FIXED_POINT variant. It is a
// one-pole DC-blocking highpass run independently per channel over interleaved
// samples, with the pole state carried across frames in hp_mem.
//
// hp_mem is [4] but this filter only ever touches hp_mem[0] and hp_mem[2] (the
// stride is 2*c, not c). hp_mem[1] and hp_mem[3] belong to hp_cutoff / silk_biquad
// (:441), which is the VOIP path and is not ported. The odd indices are NOT a
// bug and must not be "tidied": OPUS_RESET_STATE clears all four, and the C
// encoder's field-level state dump compares all four.
//
// in and out are interleaved opus_res (int16) of length*channels samples. They
// may alias in the C; the Opus caller always passes distinct buffers (pcm and
// &pcm_buf[total_buffer*channels]), and so does this port.
func DCReject(in []int16, cutoffHz int32, out []int16, hpMem *[4]int32, length, channels int, fs int32) {
	// Approximates -round(log2(6.3*cutoff_Hz/Fs)).
	// At 48 kHz with the hard-coded cutoff 3: celt_ilog2(48000/12) =
	// celt_ilog2(4000) = 11.
	shift := fixedmath.Celt_ilog2(fs / (cutoffHz * 4))
	for c := 0; c < channels; c++ {
		for i := 0; i < length; i++ {
			// Saturate at +6 dBFS to avoid any wrap-around. With opus_res ==
			// opus_int16 the bound is (1<<16<<0)-1 = 65535, so this clamp cannot
			// actually fire; it is kept because it is what the C does and it does
			// fire under ENABLE_RES24.
			x := fixedmath.SATURATE(int32(in[channels*i+c]), (1<<16<<resShift)-1)
			x = fixedmath.SHL32(x, 14-resShift)
			y := x - hpMem[2*c]
			// The C reads the OLD hp_mem[2*c] on both sides; Go evaluates the RHS
			// first, so += is exact here.
			hpMem[2*c] += fixedmath.PSHR32(x-hpMem[2*c], shift)
			// The ENABLE_RES24 branch skips the saturation; this is the RES16 one.
			out[channels*i+c] = int16(fixedmath.SATURATE(fixedmath.PSHR32(y, 14-resShift), 32767))
		}
	}
}

// StereoFade is stereo_fade (opus_encoder.c:548): it narrows the stereo image by
// subtracting a fraction of the side signal, cross-fading the gain from g1 to g2
// over the CELT MDCT overlap so the change between frames is not a step.
//
// g1 and g2 are Q15 (Q15ONE == 32767 means "no narrowing"); the caller maps the
// Q14 stereoWidth values onto them at :2333-2340. window is celt_mode->window
// (celt_coef, which is opus_int16 here, so COEF2VAL16 at :562 is the identity)
// and overlap48 is celt_mode->overlap (120 for the 48 kHz / 960 mode).
//
// in and out alias in every call site (:2344 passes pcm_buf twice). That is safe
// because each iteration reads in[i*channels..+1] before writing
// out[i*channels..+1].
func StereoFade(in, out []int16, g1, g2 int16, overlap48, frameSize, channels int, window []int16, fs int32) {
	inc := 1
	if v := int(48000 / fs); v > inc {
		inc = v // IMAX(1, 48000/Fs)
	}
	overlap := overlap48 / inc

	// MULT16_RES_Q15 is MULT16_16_Q15 (celt/arch.h:175) and MULT16_16 CASTS both
	// operands to opus_val16 (celt/fixed_generic.h:176), so the opus_val32 diff is
	// genuinely narrowed to int16 by the macro. That is lossless: diff is
	// (int16 - int16) >> 1, i.e. exactly [-32768, 32767].
	g1 = int16(int32(q15One) - int32(g1))
	g2 = int16(int32(q15One) - int32(g2))

	i := 0
	for ; i < overlap; i++ {
		w := window[i*inc] // COEF2VAL16 is the identity (celt/arch.h:199)
		w = int16(fixedmath.MULT16_16_Q15(w, w))
		g := int16(fixedmath.SHR32(
			fixedmath.MAC16_16(fixedmath.MULT16_16(w, g2), int16(int32(q15One)-int32(w)), g1), 15))
		diff := fixedmath.HALF32(int32(in[i*channels]) - int32(in[i*channels+1]))
		diff = fixedmath.MULT16_16_Q15(g, int16(diff))
		out[i*channels] = int16(int32(out[i*channels]) - diff)
		out[i*channels+1] = int16(int32(out[i*channels+1]) + diff)
	}
	for ; i < frameSize; i++ {
		diff := fixedmath.HALF32(int32(in[i*channels]) - int32(in[i*channels+1]))
		diff = fixedmath.MULT16_16_Q15(g2, int16(diff))
		out[i*channels] = int16(int32(out[i*channels]) - diff)
		out[i*channels+1] = int16(int32(out[i*channels+1]) + diff)
	}
}

// UpdateVariableHPSmth2 is opus_encoder.c:1969-1982: the smoothed SILK highpass
// cutoff. On the CELT-only path hp_freq_smth1 is the CONSTANT
// silk_lin2log(VARIABLE_HP_MIN_CUTOFF_HZ)<<8 (:1970), which is exactly what
// opus_encoder_init seeds st->variable_HP_smth2_Q15 with (:319), so the smoother's
// input delta is zero and the state never moves.
//
// It is ported anyway because variable_HP_smth2_Q15 IS cross-frame state and the
// field-level state comparison against the C encoder has to close on it, and
// because the moment SILK becomes reachable hp_freq_smth1 stops being constant.
//
// The returned cutoff_Hz is UNUSED on the AUDIO path: the only consumer is
// hp_cutoff (:1984), the OPUS_APPLICATION_VOIP filter, and the AUDIO branch calls
// dc_reject with a hard-coded 3 instead (:2008). It is returned rather than
// dropped so the dead value is visible rather than silently missing.
func (st *Encoder) UpdateVariableHPSmth2() int32 {
	// MODE_CELT_ONLY: :1970. The MODE_SILK_ONLY / MODE_HYBRID alternative at
	// :1972 reads the SILK encoder's variable_HP_smth1_Q15 and is not reachable.
	hpFreqSmth1 := silkmath.Silk_LSHIFT(silkmath.Silk_lin2log(variableHPMinCutoffHz), 8)

	st.variableHPSmth2Q15 = silkmath.Silk_SMLAWB(st.variableHPSmth2Q15,
		hpFreqSmth1-st.variableHPSmth2Q15, variableHPSmthCoef2Q16)

	// Convert from log scale to Hertz (:1982).
	return silkmath.Silk_log2lin(silkmath.Silk_RSHIFT(st.variableHPSmth2Q15, 8))
}

// PCMFront is opus_encoder.c:1966-2008, the first half of the PCM front end:
//
//  1. allocate pcm_buf for total_buffer + frame_size samples per channel;
//  2. prime its HEAD with the delay-compensation history, which is the NEWEST
//     total_buffer samples of delay_buffer, i.e. its TAIL:
//     OPUS_COPY(pcm_buf, &delay_buffer[(encoder_buffer-total_buffer)*C], total_buffer*C);
//  3. advance the variable_HP_smth2_Q15 smoother;
//  4. dc_reject the caller's new frame into pcm_buf just PAST the history.
//
// The result is pcm_buf = [total_buffer old samples] ++ [frame_size new samples],
// all DC-rejected, which is exactly what celt_encode_with_ec is handed at :2493 —
// from index 0, i.e. STARTING AT THE HISTORY. CELT's input is deliberately
// time-shifted back by total_buffer samples and the newest total_buffer samples of
// this frame are not coded until the NEXT frame. That is the whole point of the
// ring.
//
// totalBuffer is delay_compensation for VOIP/AUDIO and 0 for the RESTRICTED_*
// applications (:1948-1951); the frozen config is AUDIO, so it is Fs/250 = 192.
func (st *Encoder) PCMFront(pcm []int16, frameSize, totalBuffer int) []int16 {
	// ALLOC(pcm_buf, (total_buffer+frame_size)*st->channels, opus_res) (:1966),
	// pooled on the encoder. The two writes below define the whole buffer — the
	// history copy fills [0,total_buffer*C) and DCReject fills the rest — so the
	// carried-over contents are never observed. See internal/celt/scratch.go.
	n := (totalBuffer + frameSize) * st.channels
	if cap(st.pcmBuf) < n {
		st.pcmBuf = make([]int16, n)
	}
	pcmBuf := st.pcmBuf[:n]
	copy(pcmBuf[:totalBuffer*st.channels],
		st.delayBuffer[(st.encoderBuffer-totalBuffer)*st.channels:])

	st.UpdateVariableHPSmth2()

	// :2008. The OPUS_APPLICATION_VOIP branch (:1984, hp_cutoff with the computed
	// cutoff_Hz) and the ENABLE_QEXT passthrough (:2004) are both dead here.
	DCReject(pcm, dcRejectCutoffHz, pcmBuf[totalBuffer*st.channels:], &st.hpMem,
		frameSize, st.channels, st.Fs)
	return pcmBuf
}

// UpdateDelayBuffer is opus_encoder.c:2304-2312. It refreshes delay_buffer from
// pcm_buf so the next frame's PCMFront finds the newest encoder_buffer samples.
//
// BOTH BRANCHES ARE REACHABLE at 48 kHz with encoder_buffer 480 and total_buffer
// 192, and a sweep that misses either one is vacuous:
//
//   - MOVE branch: channels*(480 - (frame_size+192)) > 0, i.e. frame_size < 288.
//     LM0 (120) and LM1 (240). The surviving tail of delay_buffer slides down by
//     frame_size and the whole of pcm_buf is appended after it.
//   - COPY branch: frame_size >= 288. LM2 (480) and LM3 (960). pcm_buf is at least
//     encoder_buffer long, so delay_buffer is simply overwritten with pcm_buf's
//     last encoder_buffer samples and the old contents are dropped entirely.
//
// It MUST run BEFORE the fades. The C says so at :2313-2314 ("gain_fade() and
// stereo_fade() need to be after the buffer copying because we don't want any of
// this to affect the SILK part"), so the stored history is PRE-FADE. Reverse the
// two and every subsequent frame diverges.
func (st *Encoder) UpdateDelayBuffer(pcmBuf []int16, frameSize, totalBuffer int) {
	c := st.channels
	if c*(st.encoderBuffer-(frameSize+totalBuffer)) > 0 {
		// OPUS_MOVE: the regions overlap; Go's copy is memmove-safe.
		n := c * (st.encoderBuffer - frameSize - totalBuffer)
		copy(st.delayBuffer[:n], st.delayBuffer[c*frameSize:])
		copy(st.delayBuffer[n:], pcmBuf[:(frameSize+totalBuffer)*c])
	} else {
		copy(st.delayBuffer, pcmBuf[(frameSize+totalBuffer-st.encoderBuffer)*c:])
	}
}

// PCMFades is opus_encoder.c:2313-2349, the second half of the PCM front end. It
// runs AFTER UpdateDelayBuffer (see there) and mutates pcm_buf in place.
//
// gain_fade (:2315-2318) is DEAD and is NOT ported: HB_gain is assigned Q15ONE at
// :2044 and only moved off it inside `if (st->mode != MODE_CELT_ONLY)` (:2061), so
// on the forced-CELT-only path both st->prev_HB_gain and HB_gain are permanently
// Q15ONE and the `prev_HB_gain < Q15ONE || HB_gain < Q15ONE` guard is never true.
// The ASSIGNMENT at :2319 is still live (it is cross-frame state that the
// field-level comparison covers), so hbGain is taken as a parameter and stored,
// even though the caller can only ever pass Q15ONE.
//
// equivRate is the "equivalent 20-ms rate" the caller computed at :1573 from
// st->bitrate_bps, st->stream_channels, the frame rate, use_vbr, st->mode,
// complexity and packet loss.
func (st *Encoder) PCMFades(pcmBuf []int16, frameSize int, equivRate int32, hbGain int16) {
	// :2315-2318 gain_fade: unreachable, see above.
	st.prevHBGain = hbGain // :2319

	// :2320-2328. In CELT-only st->mode is never MODE_HYBRID, so the guard is
	// always true; it is kept verbatim.
	if st.mode != ModeHybrid || st.streamChannels == 1 {
		switch {
		case equivRate > 32000:
			st.silkMode.stereoWidthQ14 = 16384
		case equivRate < 16000:
			st.silkMode.stereoWidthQ14 = 0
		default:
			// All opus_int32; the division truncates toward zero in C and in Go,
			// and both operands are non-negative over [16000, 32000] anyway.
			st.silkMode.stereoWidthQ14 = 16384 - 2048*(32000-equivRate)/(equivRate-14000)
		}
	}

	// :2329-2349. energy_masking is only ever set by the multistream (surround)
	// encoder, so on the single-stream path this reduces to "stereo".
	if st.energyMasking == nil && st.channels == 2 {
		// Apply stereo width reduction (at low bitrates).
		if int32(st.hybridStereoWidthQ14) < (1<<14) || st.silkMode.stereoWidthQ14 < (1<<14) {
			g1 := st.hybridStereoWidthQ14
			g2 := int16(st.silkMode.stereoWidthQ14)
			// Q14 -> Q15, with 16384 (unity) mapped onto Q15ONE == 32767 rather
			// than 32768, which does not fit. FIXED_POINT branch at :2337-2338.
			if g1 == 16384 {
				g1 = q15One
			} else {
				g1 = fixedmath.SHL16(g1, 1)
			}
			if g2 == 16384 {
				g2 = q15One
			} else {
				g2 = fixedmath.SHL16(g2, 1)
			}
			StereoFade(pcmBuf, pcmBuf, g1, g2, st.celt.Overlap(), frameSize, st.channels,
				st.celt.Window(), st.Fs)
			st.hybridStereoWidthQ14 = int16(st.silkMode.stereoWidthQ14)
		}
	}
}

// SetPCMFrameContext injects the two per-frame decisions that opus_encode_native
// makes UPSTREAM of the PCM front end and that the front end then reads back off
// the encoder: st->mode (:1528-1530, pinned to MODE_CELT_ONLY by the forced mode)
// and st->stream_channels (:1428-1456, the rate-dependent stereo/mono downmix).
//
// They belong to the encoder driver, not to this unit. This setter exists so the
// PCM front end can be driven — and differentially tested against the C oracle —
// in isolation, before the driver lands. Once opus_encode_native is ported it will
// assign both fields itself and this becomes a no-op seam that the integrator may
// drop.
func (st *Encoder) SetPCMFrameContext(mode, streamChannels int) {
	st.mode = mode
	st.streamChannels = streamChannels
}
