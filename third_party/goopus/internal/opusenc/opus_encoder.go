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

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/celt"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"
)

// Opus error codes (include/opus_defines.h).
const (
	opusOK             = 0  // OPUS_OK
	opusBadArg         = -1 // OPUS_BAD_ARG
	opusBufferTooSmall = -2 // OPUS_BUFFER_TOO_SMALL
	opusInternalError  = -3 // OPUS_INTERNAL_ERROR
)

// Ctl sentinels and enums (include/opus_defines.h:211-245).
const (
	OpusAuto       = -1000 // OPUS_AUTO
	OpusBitrateMax = -1    // OPUS_BITRATE_MAX

	ApplicationVOIP               = 2048 // OPUS_APPLICATION_VOIP
	ApplicationAudio              = 2049 // OPUS_APPLICATION_AUDIO
	ApplicationRestrictedLowdelay = 2051 // OPUS_APPLICATION_RESTRICTED_LOWDELAY

	SignalVoice = 3001 // OPUS_SIGNAL_VOICE
	SignalMusic = 3002 // OPUS_SIGNAL_MUSIC

	FramesizeArg   = 5000 // OPUS_FRAMESIZE_ARG
	Framesize2_5Ms = 5001 // OPUS_FRAMESIZE_2_5_MS
	Framesize5Ms   = 5002 // OPUS_FRAMESIZE_5_MS
	Framesize10Ms  = 5003 // OPUS_FRAMESIZE_10_MS
	Framesize20Ms  = 5004 // OPUS_FRAMESIZE_20_MS
	Framesize40Ms  = 5005 // OPUS_FRAMESIZE_40_MS
	Framesize60Ms  = 5006 // OPUS_FRAMESIZE_60_MS
	Framesize80Ms  = 5007 // OPUS_FRAMESIZE_80_MS
	Framesize100Ms = 5008 // OPUS_FRAMESIZE_100_MS
	Framesize120Ms = 5009 // OPUS_FRAMESIZE_120_MS
)

// Coding modes (src/opus_private.h:148-150).
const (
	ModeSilkOnly = 1000 // MODE_SILK_ONLY
	ModeHybrid   = 1001 // MODE_HYBRID
	ModeCeltOnly = 1002 // MODE_CELT_ONLY
)

// Audio bandwidths (include/opus_defines.h:230-234).
const (
	BandwidthNarrowband    = 1101 // OPUS_BANDWIDTH_NARROWBAND
	BandwidthMediumband    = 1102 // OPUS_BANDWIDTH_MEDIUMBAND
	BandwidthWideband      = 1103 // OPUS_BANDWIDTH_WIDEBAND
	BandwidthSuperwideband = 1104 // OPUS_BANDWIDTH_SUPERWIDEBAND
	BandwidthFullband      = 1105 // OPUS_BANDWIDTH_FULLBAND
)

const (
	// maxEncoderBuffer is MAX_ENCODER_BUFFER (opus_encoder.c:66), the non-QEXT
	// value. delay_buffer is opus_res[MAX_ENCODER_BUFFER*2] where the *2 is
	// CHANNELS, not double buffering.
	maxEncoderBuffer = 480
	// q15One is Q15ONE (celt/arch.h:204) in this fixed-point build.
	q15One = 32767
	// variableHPMinCutoffHz is VARIABLE_HP_MIN_CUTOFF_HZ (silk/tuning_parameters.h:67).
	variableHPMinCutoffHz = 60
)

// silkEncControl holds the subset of silk_EncControlStruct that the frozen
// CELT-only path actually reads or writes. The full struct is a SILK concern; the
// fields here are the ones the ctls set (opus_encoder.c:2943, :2966, :2986, :3009,
// :3163) and the ones opus_encode_native computes (stereoWidth_Q14 at :2320-2327).
// stereoWidth_Q14 is NOT cross-frame state: it is recomputed from equiv_rate every
// frame before any read. It lives here rather than on the Encoder so the field
// names stay diffable against the C.
type silkEncControl struct {
	complexity           int
	packetLossPercentage int
	useInBandFEC         bool
	useCBR               bool
	useDTX               bool
	reducedDependency    int
	stereoWidthQ14       int32
}

// Encoder is struct OpusEncoder (opus_encoder.c:76) for the frozen config. The
// SILK sub-encoder, the DRED encoder and the TonalityAnalysisState are absent
// (forced CELT-only, no DRED, DISABLE_FLOAT_API); the CELT sub-encoder is a real
// pointer rather than a byte offset into one allocation.
type Encoder struct {
	// --- configuration, above OPUS_ENCODER_RESET_START (:80-107) ---
	application       int
	channels          int
	delayCompensation int
	forceChannels     int
	signalType        int
	userBandwidth     int
	maxBandwidth      int
	userForcedMode    int
	voiceRatio        int
	Fs                int32
	useVbr            int
	vbrConstraint     int
	variableDuration  int
	bitrateBps        int32
	userBitrateBps    int32
	lsbDepth          int
	encoderBuffer     int
	lfe               int
	useDtx            int
	fecConfig         int

	silkMode silkEncControl

	// --- reset region: everything from OPUS_ENCODER_RESET_START (:111) on, i.e.
	// exactly what OPUS_RESET_STATE clears (:3253-3254). ---
	streamChannels       int
	hybridStereoWidthQ14 int16
	variableHPSmth2Q15   int32
	prevHBGain           int16
	hpMem                [4]int32 // opus_val32
	mode                 int
	prevMode             int
	prevChannels         int
	prevFramesize        int
	bandwidth            int
	autoBandwidth        int
	silkBwSwitch         int
	first                int
	energyMasking        []int32 // celt_glog*, nil on the single-stream path
	// nbNoActivityMsQ1 (nb_no_activity_ms_Q1) and peakSignalEnergy
	// (peak_signal_energy) sit here to mirror the C struct order (they follow
	// width_mem / detected_bandwidth and precede nonfinal_frame). Both are cleared
	// by OPUS_RESET_STATE. nbNoActivityMsQ1 is the LIVE consecutive-silence counter
	// generalized DTX advances; peakSignalEnergy is tracked for state-hash
	// defense-in-depth (it is output-dead for the DTX decision in this config; see
	// dtx.go and the State doc).
	nbNoActivityMsQ1 int
	peakSignalEnergy int32 // opus_val32
	nonfinalFrame    int
	rangeFinal       uint32
	delayBuffer      []int16 // opus_res[encoder_buffer*channels]

	// DELIBERATELY ABSENT: width_mem (StereoWidthState). Its only consumer is the
	// OPUS_AUTO mode decision, which forced CELT-only never enters and which stays
	// unreachable without SILK, so it is output-dead AND excluded. peak_signal_energy
	// used to share this exclusion; it is now carried (above) and compared, because
	// DTX brings it into the function this port modifies. See the State doc.

	celt *celt.Encoder

	// pcmBuf is the pooled stand-in for opus_encoder.c's ALLOC(pcm_buf, ...) stack
	// buffer (:1966), the largest allocation on the opusenc side. NOT state: every
	// frame rewrites it completely (delay-buffer history into the head, DCReject
	// into the tail) before anything reads it, and PCMFront's caller is done with
	// the returned slice before the next frame. Not safe for concurrent use, like
	// the rest of the Encoder.
	pcmBuf []int16

	// tmpData is the pooled stand-in for opus_encode_native's ALLOC(tmp_data, ...)
	// stack buffer (:1757), used only on the multiframe (40-120 ms) path to hold the
	// concatenated sub-frame payloads. NOT state: encodeMultiframe rewrites it from
	// offset 0 every call and the repacketizer's frame slices into it do not outlive
	// the call. Pooling it keeps the multiframe path at 0 allocs/op, matching the
	// single-frame path. Not safe for concurrent use, like the rest of the Encoder.
	tmpData []byte

	// wit records which branches the last EncodeRaw took. It is NOT encoder state:
	// nothing reads it, it is not reset by OPUS_RESET_STATE, and it has no C
	// counterpart. It exists so the differential test can prove its sweep is not
	// vacuous. See Witness in encode.go.
	wit Witness
}

// NewEncoder is opus_encoder_init (opus_encoder.c:204) for the frozen config: Fs
// must be one of 8000/12000/16000/24000/48000 and channels 1 or 2, and the
// application one of VOIP / AUDIO / RESTRICTED_LOWDELAY. It returns nil on a bad
// argument, matching OPUS_BAD_ARG.
func NewEncoder(fs int32, channels, application int) *Encoder {
	if fs != 48000 && fs != 24000 && fs != 16000 && fs != 12000 && fs != 8000 {
		return nil
	}
	if channels != 1 && channels != 2 {
		return nil
	}
	if application != ApplicationVOIP && application != ApplicationAudio &&
		application != ApplicationRestrictedLowdelay {
		return nil
	}

	st := &Encoder{}
	st.channels = channels
	st.Fs = fs

	// opus_encoder.c:281. Fs is THREADED INTO CELT, where it becomes
	// st->upsample = resampling_factor(Fs) (celt_encoder.c:255) and nothing else:
	// the CELT mode stays the 48 kHz / 960 one at every rate. A rate CELT cannot
	// resample makes this nil, which is the second line of defence behind the Fs
	// check above.
	st.celt = celt.NewEncoder(fs, channels)
	if st.celt == nil {
		return nil
	}
	// opus_encoder.c:283-284. SIGNALLING(0) because the Opus layer writes its own
	// TOC byte instead of CELT's self-delimiting signalling byte.
	st.celt.SetSignalling(0)
	st.celt.SetComplexity(9) // silk_mode.complexity default (:270)

	// :288-317, the non-reset defaults.
	st.silkMode.complexity = 9
	st.silkMode.packetLossPercentage = 0
	st.silkMode.useInBandFEC = false
	st.silkMode.useDTX = false
	st.silkMode.useCBR = false
	st.silkMode.reducedDependency = 0

	st.useVbr = 1
	st.vbrConstraint = 1 // constrained VBR is the default
	st.userBitrateBps = OpusAuto
	st.bitrateBps = 3000 + fs*int32(channels)
	st.application = application
	st.signalType = OpusAuto
	st.userBandwidth = OpusAuto
	st.maxBandwidth = BandwidthFullband
	st.forceChannels = OpusAuto
	st.userForcedMode = OpusAuto
	st.voiceRatio = -1
	st.encoderBuffer = int(fs / 100)
	st.lsbDepth = 24
	st.variableDuration = FramesizeArg
	// Delay compensation of 4 ms (2.5 ms for SILK's extra look-ahead + 1.5 ms for
	// SILK resamplers and stereo prediction). :313.
	st.delayCompensation = int(fs / 250)

	st.delayBuffer = make([]int16, st.encoderBuffer*channels)

	st.resetRegion()
	return st
}

// resetRegion applies the OPUS_ENCODER_RESET_START clear plus the explicit
// re-initialisation the C does in BOTH opus_encoder_init (:315-321) and
// OPUS_RESET_STATE (:3253-3270). Keeping it in one place is how the port
// guarantees the two paths cannot drift apart, which they can in C.
func (st *Encoder) resetRegion() {
	// OPUS_CLEAR from OPUS_ENCODER_RESET_START to the end (:3254).
	st.hybridStereoWidthQ14 = 0
	st.variableHPSmth2Q15 = 0
	st.prevHBGain = 0
	st.hpMem = [4]int32{}
	st.mode = 0
	st.prevMode = 0
	st.prevChannels = 0
	st.prevFramesize = 0
	st.bandwidth = 0
	st.autoBandwidth = 0
	st.silkBwSwitch = 0
	st.first = 0
	st.energyMasking = nil
	st.nbNoActivityMsQ1 = 0
	st.peakSignalEnergy = 0
	st.nonfinalFrame = 0
	st.rangeFinal = 0
	for i := range st.delayBuffer {
		st.delayBuffer[i] = 0
	}

	// The explicit re-init that follows the clear.
	st.streamChannels = st.channels
	st.hybridStereoWidthQ14 = 1 << 14
	st.prevHBGain = q15One
	st.first = 1
	st.mode = ModeHybrid
	st.bandwidth = BandwidthFullband
	st.variableHPSmth2Q15 = silkmath.Silk_lin2log(variableHPMinCutoffHz) << 8
}

// Reset is OPUS_RESET_STATE (opus_encoder.c:3243). It clears the reset region and
// resets the CELT sub-encoder; the configuration ctls above
// OPUS_ENCODER_RESET_START are left alone.
func (st *Encoder) Reset() {
	st.resetRegion()
	st.celt.Reset()
}

// ---------------------------------------------------------------------------
// Ctl setters. Each mirrors its OPUS_SET_* case in opus_encoder_ctl and returns
// the same validation verdict: nil for OPUS_OK, ErrBadArg for the `goto bad_arg`
// paths (which leave the field UNCHANGED, exactly as in C).
// ---------------------------------------------------------------------------

// SetApplication is OPUS_SET_APPLICATION (:2786). The application may only be
// changed before the first frame (`!st->first` rejects it afterwards).
func (st *Encoder) SetApplication(v int) error {
	if v != ApplicationVOIP && v != ApplicationAudio && v != ApplicationRestrictedLowdelay {
		return ErrBadArg
	}
	if st.first == 0 && st.application != v {
		return ErrBadArg
	}
	st.application = v
	return nil
}

// SetBitrate is OPUS_SET_BITRATE (:2817). OPUS_AUTO and OPUS_BITRATE_MAX pass
// through; anything else is clamped into [500, 750000*channels], and <= 0 is
// rejected.
func (st *Encoder) SetBitrate(v int32) error {
	if v != OpusAuto && v != OpusBitrateMax {
		switch {
		case v <= 0:
			return ErrBadArg
		case v <= 500:
			v = 500
		case v > 750000*int32(st.channels):
			v = 750000 * int32(st.channels)
		}
	}
	st.userBitrateBps = v
	return nil
}

// SetForceChannels is OPUS_SET_FORCE_CHANNELS (:2842).
func (st *Encoder) SetForceChannels(v int) error {
	if (v < 1 || v > st.channels) && v != OpusAuto {
		return ErrBadArg
	}
	st.forceChannels = v
	return nil
}

// SetMaxBandwidth is OPUS_SET_MAX_BANDWIDTH (:2862). The
// silk_mode.maxInternalSampleRate side effect is SILK-only and omitted.
func (st *Encoder) SetMaxBandwidth(v int) error {
	if v < BandwidthNarrowband || v > BandwidthFullband {
		return ErrBadArg
	}
	st.maxBandwidth = v
	return nil
}

// SetBandwidth is OPUS_SET_BANDWIDTH (:2889): sets st->user_bandwidth, which
// OVERRIDES the automatic bandwidth selection at :1629-1650. OPUS_AUTO restores
// automatic selection. The silk_mode.maxInternalSampleRate side effect is
// SILK-only and omitted.
func (st *Encoder) SetBandwidth(v int) error {
	if (v < BandwidthNarrowband || v > BandwidthFullband) && v != OpusAuto {
		return ErrBadArg
	}
	st.userBandwidth = v
	return nil
}

// SetDTX is OPUS_SET_DTX (:2916). DTX itself is deferred; the field is carried so
// the configuration round-trips and so the deferral is explicit rather than
// silent.
func (st *Encoder) SetDTX(v int) error {
	if v < 0 || v > 1 {
		return ErrBadArg
	}
	st.useDtx = v
	return nil
}

// SetComplexity is OPUS_SET_COMPLEXITY (:2936): 0..10, forwarded to CELT.
func (st *Encoder) SetComplexity(v int) error {
	if v < 0 || v > 10 {
		return ErrBadArg
	}
	st.silkMode.complexity = v
	st.celt.SetComplexity(v)
	return nil
}

// SetInbandFEC is OPUS_SET_INBAND_FEC (:2958): 0..2. FEC is SILK-only, so on the
// CELT-only path this only moves fec_config.
func (st *Encoder) SetInbandFEC(v int) error {
	if v < 0 || v > 2 {
		return ErrBadArg
	}
	st.fecConfig = v
	st.silkMode.useInBandFEC = v != 0
	return nil
}

// SetPacketLossPerc is OPUS_SET_PACKET_LOSS_PERC (:2979): 0..100, forwarded to
// CELT. It is LIVE on the CELT-only path: it feeds compute_equiv_rate's loss
// argument and CELT's own loss handling.
func (st *Encoder) SetPacketLossPerc(v int) error {
	if v < 0 || v > 100 {
		return ErrBadArg
	}
	st.silkMode.packetLossPercentage = v
	st.celt.SetLossRate(v)
	return nil
}

// SetVBR is OPUS_SET_VBR (:3001).
func (st *Encoder) SetVBR(v int) error {
	if v < 0 || v > 1 {
		return ErrBadArg
	}
	st.useVbr = v
	st.silkMode.useCBR = v == 0
	return nil
}

// SetVoiceRatio is OPUS_SET_VOICE_RATIO (:3022): -1..100. Note that
// opus_encode_native unconditionally re-pins voice_ratio to -1 under
// DISABLE_FLOAT_API (:1307), so this setter cannot influence the output in this
// build; it exists so the ctl surface is complete.
func (st *Encoder) SetVoiceRatio(v int) error {
	if v < -1 || v > 100 {
		return ErrBadArg
	}
	st.voiceRatio = v
	return nil
}

// SetVBRConstraint is OPUS_SET_VBR_CONSTRAINT (:3042).
func (st *Encoder) SetVBRConstraint(v int) error {
	if v < 0 || v > 1 {
		return ErrBadArg
	}
	st.vbrConstraint = v
	return nil
}

// SetSignal is OPUS_SET_SIGNAL (:3062): OPUS_AUTO, OPUS_SIGNAL_VOICE or
// OPUS_SIGNAL_MUSIC.
func (st *Encoder) SetSignal(v int) error {
	if v != OpusAuto && v != SignalVoice && v != SignalMusic {
		return ErrBadArg
	}
	st.signalType = v
	return nil
}

// SetLsbDepth is OPUS_SET_LSB_DEPTH (:3114): 8..24.
func (st *Encoder) SetLsbDepth(v int) error {
	if v < 8 || v > 24 {
		return ErrBadArg
	}
	st.lsbDepth = v
	return nil
}

// SetExpertFrameDuration is OPUS_SET_EXPERT_FRAME_DURATION (:3134). It feeds
// frame_size_select (:827), which may REJECT the frame size the caller passes to
// Encode.
func (st *Encoder) SetExpertFrameDuration(v int) error {
	switch v {
	case FramesizeArg, Framesize2_5Ms, Framesize5Ms, Framesize10Ms, Framesize20Ms,
		Framesize40Ms, Framesize60Ms, Framesize80Ms, Framesize100Ms, Framesize120Ms:
		st.variableDuration = v
		return nil
	default:
		return ErrBadArg
	}
}

// SetPredictionDisabled is OPUS_SET_PREDICTION_DISABLED (:3158). On the CELT-only
// path it reaches CELT through CELT_SET_PREDICTION at :2292-2294 (celt_pred = 0
// when reducedDependency is set), not through a ctl here.
func (st *Encoder) SetPredictionDisabled(v int) error {
	if v < 0 || v > 1 {
		return ErrBadArg
	}
	st.silkMode.reducedDependency = v
	return nil
}

// SetForceMode is OPUS_SET_FORCE_MODE (:3273): MODE_SILK_ONLY..MODE_CELT_ONLY or
// OPUS_AUTO. The phase-4 gate pins this to MODE_CELT_ONLY.
func (st *Encoder) SetForceMode(v int) error {
	if (v < ModeSilkOnly || v > ModeCeltOnly) && v != OpusAuto {
		return ErrBadArg
	}
	st.userForcedMode = v
	return nil
}

// SetLFE is OPUS_SET_LFE (:3283), forwarded to CELT. The ctl does not validate.
func (st *Encoder) SetLFE(v int) error {
	st.lfe = v
	st.celt.SetLFE(v)
	return nil
}

// SetEnergyMask is OPUS_SET_ENERGY_MASK (:3291), forwarded to CELT. Only the
// multistream (surround) encoder ever sets it; on the single-stream path it stays
// nil. The slice is aliased, not copied, exactly like the C pointer.
func (st *Encoder) SetEnergyMask(mask []int32) error {
	st.energyMasking = mask
	st.celt.SetEnergyMask(mask)
	return nil
}

// FinalRange is OPUS_GET_FINAL_RANGE (:3104): the range coder state after the
// last frame. Bit-exact agreement of this value with libopus is the primary
// encoder differential check.
func (st *Encoder) FinalRange() uint32 { return st.rangeFinal }

// ClearFinalRange zeroes rangeFinal. opus_encode_native does this before its
// rejection rules (opus_encoder.c:1223), so a rejected encode leaves
// OPUS_GET_FINAL_RANGE reading 0 rather than the previous packet's range. Encode
// already mirrors that internally; this exists for the public wrapper, whose own
// argument checks reject before reaching Encode at all.
func (st *Encoder) ClearFinalRange() { st.rangeFinal = 0 }

// Lookahead is OPUS_GET_LOOKAHEAD (opus_encoder.c:3082): the encoder's algorithmic
// delay in samples.
//
//	*value = st->Fs/400;
//	if (application != RESTRICTED_LOWDELAY && application != RESTRICTED_CELT)
//	    *value += st->delay_compensation;
//
// Both terms are required. Fs/400 is the CELT MDCT overlap lookahead and
// delay_compensation is the ring-buffer delay, so at 48 kHz with the AUDIO
// application this is 120 + 192 = 312, not 192. The value does not affect any
// packet byte, which is why the differential gate cannot see an error here, but it
// is what the Ogg Opus pre-skip field is derived from: returning 192 would misalign
// every decoded stream by 120 samples. TestOpusencLookaheadMatchesC pins it against
// the C ctl.
// C also excludes OPUS_APPLICATION_RESTRICTED_CELT (2053) from the
// delay_compensation term, but NewEncoder does not accept that application, so that
// arm is unreachable here.
func (st *Encoder) Lookahead() int {
	v := int(st.Fs / 400)
	if st.application != ApplicationRestrictedLowdelay {
		v += st.delayCompensation
	}
	return v
}

// DelayCompensation is st->delay_compensation (Fs/250, so 192 at 48 kHz for the AUDIO
// application). It is the depth of the delay-compensation ring buffer, and it is NOT
// the same as Lookahead, which additionally includes the Fs/400 CELT overlap term.
// Conflating the two is how the missing overlap term in Lookahead went unnoticed, so
// callers that want the ring depth must use this and not Lookahead.
func (st *Encoder) DelayCompensation() int { return st.delayCompensation }

// CELT exposes the CELT sub-encoder. The Opus layer drives it through the ctl
// setters above and celt.EncodeWithEC; this accessor exists for the differential
// test, which compares the CELT state field for field alongside the Opus state.
func (st *Encoder) CELT() *celt.Encoder { return st.celt }
