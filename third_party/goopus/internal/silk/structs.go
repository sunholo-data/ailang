/***********************************************************************
Copyright (c) 2006-2011, Skype Limited. All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:
- Redistributions of source code must retain the above copyright notice,
this list of conditions and the following disclaimer.
- Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.
- Neither the name of Internet Society, IETF or IETF Trust, nor the
names of specific contributors, may be used to endorse or promote
products derived from this software without specific prior written
permission.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE.
***********************************************************************/

// Shared SILK decoder structs, transliterated from silk/structs.h (libopus
// v1.6.1): SideInfoIndices (the quantization index side-information the range
// decoder produces per frame), the decode-relevant subset of silk_decoder_state,
// and silk_decoder_control. The full decode chain is now present: the synthesis and
// output buffers, the per-channel resampler state, the PLC and CNG sub-states, and
// the sampling-rate-selected tables. The encoder-only members of the C structs remain
// deliberately omitted. Field names follow the C for diffability.
//
// The value-shaped DecoderState.DecoderSetFs below is the geometry-only helper (table
// and size selection) used by the decode_core / decode_parameters unit tests; the full
// silk_decoder_set_fs (adding the resampler init and cross-frame reset) is the method
// silkDecoderSetFs in decoder_set_fs.go, which the top-level dec_api.go decode uses.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// Constants from silk/define.h and silk/pitch_est_defines.h. maxLPCOrder (16) and
// nlsfQuantMaxAmplitude (4) are already defined in nlsf_decode.go and reused here.
const (
	maxNBSubfr         = 4  // MAX_NB_SUBFR: 5 ms subframes per 20 ms frame
	minLPCOrder        = 10 // MIN_LPC_ORDER (NB/MB filter order)
	ltpOrder           = 5  // LTP_ORDER: long-term-prediction taps per subframe
	maxFramesPerPacket = 3  // MAX_FRAMES_PER_PACKET

	subFrameLengthMS = 5  // SUB_FRAME_LENGTH_MS
	ltpMemLengthMS   = 20 // LTP_MEM_LENGTH_MS

	// Buffer sizing from silk/define.h. maxFSKHz is MAX_FS_KHZ (16). The frame and
	// subframe maxima size the persistent synthesis buffers decode_core writes.
	maxFSKHz          = 16                            // MAX_FS_KHZ
	maxFrameLengthMS  = subFrameLengthMS * maxNBSubfr // MAX_FRAME_LENGTH_MS (20)
	maxFrameLength    = maxFrameLengthMS * maxFSKHz   // MAX_FRAME_LENGTH (320)
	maxSubFrameLength = subFrameLengthMS * maxFSKHz   // MAX_SUB_FRAME_LENGTH (80)
	outBufLength      = maxFrameLength + 2*maxSubFrameLength

	// QUANT_LEVEL_ADJUST_Q10 (define.h): excitation quantization-level adjustment.
	quantLevelAdjustQ10 = 80

	// Signal types (define.h).
	typeNoVoiceActivity = 0 // TYPE_NO_VOICE_ACTIVITY
	typeUnvoiced        = 1 // TYPE_UNVOICED
	typeVoiced          = 2 // TYPE_VOICED

	// Conditional coding types (define.h).
	codeIndependently             = 0 // CODE_INDEPENDENTLY
	codeIndependentlyNoLTPScaling = 1 // CODE_INDEPENDENTLY_NO_LTP_SCALING
	codeConditionally             = 2 // CODE_CONDITIONALLY

	bweAfterLossQ16 = 63570 // BWE_AFTER_LOSS_Q16

	// Pitch estimator ranges (pitch_est_defines.h) used by silk_decode_pitch.
	peMaxNBSubfr       = 4  // PE_MAX_NB_SUBFR
	peMaxLagMS         = 18 // PE_MAX_LAG_MS (18 ms -> 56 Hz)
	peMinLagMS         = 2  // PE_MIN_LAG_MS (2 ms -> 500 Hz)
	peNBCbksStage2Ext  = 11 // PE_NB_CBKS_STAGE2_EXT (20 ms, 8 kHz)
	peNBCbksStage210MS = 3  // PE_NB_CBKS_STAGE2_10MS (10 ms, 8 kHz)
	peNBCbksStage3Max  = 34 // PE_NB_CBKS_STAGE3_MAX (20 ms, >8 kHz)
	peNBCbksStage310MS = 12 // PE_NB_CBKS_STAGE3_10MS (10 ms, >8 kHz)
)

// SideInfoIndices mirrors silk/structs.h SideInfoIndices: the per-frame
// quantization indices the range decoder reads in silk_decode_indices and the
// parameter decode later turns into gains, LPC coefficients, pitch lags and LTP
// taps. Array sizes follow the C ([MAX_NB_SUBFR], [MAX_LPC_ORDER+1]).
type SideInfoIndices struct {
	GainsIndices     [maxNBSubfr]int8
	LTPIndex         [maxNBSubfr]int8
	NLSFIndices      [maxLPCOrder + 1]int8
	LagIndex         int16
	ContourIndex     int8
	SignalType       int8
	QuantOffsetType  int8
	NLSFInterpCoefQ2 int8
	PERIndex         int8
	LTPScaleIndex    int8
	Seed             int8
}

// PLCStruct mirrors silk/structs.h silk_PLC_struct: the persistent packet-loss
// concealment sub-state. silk_PLC_update saves the last good frame's LTP taps,
// LPC coefficients, pitch lag, LTP scale and last two gains here; silk_PLC_conceal
// reads them to extrapolate a lost frame and mutates several in place (bwexpands
// prevLPC_Q12, attenuates LTPCoef_Q14, drifts pitchL_Q8, advances the random seed
// and excitation scale); silk_PLC_glue_frames stores the concealed-frame energy for
// the recovery-frame match. Field names follow the C. The ENABLE_DEEP_PLC-only
// enable_deep_plc member is omitted (deep PLC out of scope).
type PLCStruct struct {
	PitchLQ8        int32              // pitchL_Q8
	LTPCoefQ14      [ltpOrder]int16    // LTPCoef_Q14
	PrevLPCQ12      [maxLPCOrder]int16 // prevLPC_Q12
	LastFrameLost   int                // last_frame_lost
	RandSeed        int32              // rand_seed
	RandScaleQ14    int16              // randScale_Q14
	ConcEnergy      int32              // conc_energy
	ConcEnergyShift int                // conc_energy_shift
	PrevLTPScaleQ14 int16              // prevLTP_scale_Q14
	PrevGainQ16     [2]int32           // prevGain_Q16
	FsKHz           int                // fs_kHz
	NbSubfr         int                // nb_subfr
	SubfrLength     int                // subfr_length
}

// CNGStruct mirrors silk/structs.h silk_CNG_struct: the persistent comfort-noise
// generation sub-state. On good inactive frames silk_CNG smooths the background
// LSFs (CNG_smth_NLSF_Q15), gain (CNG_smth_Gain_Q16) and excitation buffer
// (CNG_exc_buf_Q14); on lost frames it synthesizes comfort noise, carrying the
// filter history in CNG_synth_state and the random index generator in rand_seed.
// Field names follow the C.
type CNGStruct struct {
	CNGExcBufQ14   [maxFrameLength]int32 // CNG_exc_buf_Q14
	CNGSmthNLSFQ15 [maxLPCOrder]int16    // CNG_smth_NLSF_Q15
	CNGSynthState  [maxLPCOrder]int32    // CNG_synth_state
	CNGSmthGainQ16 int32                 // CNG_smth_Gain_Q16
	RandSeed       int32                 // rand_seed
	FsKHz          int                   // fs_kHz
}

// DecoderState is the decode-relevant subset of silk/structs.h silk_decoder_state.
// It carries the cross-frame state silk_decode_indices / silk_decode_parameters
// consume (prevNLSF_Q15, LastGainIndex, ec_prevSignalType, ec_prevLagIndex,
// first_frame_after_reset, lossCnt, prev_gain_Q16), the frame geometry set by
// DecoderSetFs (fs_kHz, nb_subfr, subfr_length, frame_length, ltp_mem_length,
// LPC_order), the per-frame indices, and the sampling-rate-selected tables
// (psNLSF_CB, pitch_lag_low_bits_iCDF, pitch_contour_iCDF). The VAD_flags array
// selects the signal-type iCDF in silk_decode_indices.
type DecoderState struct {
	PrevGainQ16 int32 // prev_gain_Q16

	// Persistent synthesis state carried across frames (silk/structs.h, the
	// SILK_DECODER_STATE_RESET_START block). decode_core writes ExcQ14 (the
	// reconstructed excitation), SLPCQ14Buf (the short-term LPC filter history)
	// and, via decode_frame, OutBuf (the ltp_mem_length + lookahead output buffer
	// re-whitened for the next frame's LTP synthesis).
	ExcQ14     [maxFrameLength]int32 // exc_Q14
	SLPCQ14Buf [maxLPCOrder]int32    // sLPC_Q14_buf
	OutBuf     [outBufLength]int16   // outBuf

	LagPrev              int                     // lagPrev
	LastGainIndex        int8                    // LastGainIndex
	FsKHz                int                     // fs_kHz (8/12/16)
	FsAPIhz              int32                   // fs_API_hz (API sample frequency, Hz)
	NbSubfr              int                     // nb_subfr (MAX_NB_SUBFR or MAX_NB_SUBFR/2)
	FrameLength          int                     // frame_length (samples)
	SubfrLength          int                     // subfr_length (samples)
	LtpMemLength         int                     // ltp_mem_length (samples)
	LPCOrder             int                     // LPC_order (10 or 16)
	PrevNLSFQ15          [maxLPCOrder]int16      // prevNLSF_Q15
	FirstFrameAfterReset int                     // first_frame_after_reset
	NFramesDecoded       int                     // nFramesDecoded
	NFramesPerPacket     int                     // nFramesPerPacket
	ECPrevSignalType     int                     // ec_prevSignalType
	ECPrevLagIndex       int16                   // ec_prevLagIndex
	VADFlags             [maxFramesPerPacket]int // VAD_flags
	LBRRFlag             int                     // LBRR_flag
	LBRRFlags            [maxFramesPerPacket]int // LBRR_flags

	// Per-channel resampler state (silk/structs.h silk_decoder_state.resampler_state).
	// silk_decoder_set_fs initializes it (SILK internal rate -> API rate) and dec_API's
	// silk_Decode feeds each channel's decoded frame through it, so it carries filter
	// memory across frames (docs/hard-parts.md 5). Added by the dec_API port, the
	// "later phase-3 port" the DecoderState doc anticipates.
	ResamplerState ResamplerState // resampler_state

	Indices        SideInfoIndices // indices
	LossCnt        int             // lossCnt
	PrevSignalType int             // prevSignalType

	// Packet-loss concealment and comfort-noise generation sub-state (silk/PLC.c,
	// silk/CNG.c). Carried across frames: silk_PLC_update / silk_CNG populate them on
	// good frames and silk_PLC / silk_CNG read (and mutate) them on lost/inactive
	// frames. Exported so the differential harness can hash them alongside the rest of
	// the persistent decoder state.
	SCNG CNGStruct // sCNG
	SPLC PLCStruct // sPLC

	// Sampling-rate-selected tables (silk_decoder_set_fs). Unexported: their types
	// are internal to package silk; DecoderSetFs assigns them.
	psNLSFCB            *silkNLSFCBStruct // psNLSF_CB
	pitchLagLowBitsICDF []byte            // pitch_lag_low_bits_iCDF
	pitchContourICDF    []byte            // pitch_contour_iCDF
}

// DecoderControl mirrors silk/structs.h silk_decoder_control: the per-frame
// prediction and coding parameters silk_decode_parameters produces (pitch lags,
// linear gains, the two interpolated LPC coefficient sets, LTP taps and the LTP
// scale), consumed by the excitation reconstruction (silk_decode_core, a later
// port).
type DecoderControl struct {
	PitchL      [maxNBSubfr]int              // pitchL
	GainsQ16    [maxNBSubfr]int32            // Gains_Q16
	PredCoefQ12 [2][maxLPCOrder]int16        // PredCoef_Q12 (interpolated + final)
	LTPCoefQ14  [ltpOrder * maxNBSubfr]int16 // LTPCoef_Q14
	LTPScaleQ14 int                          // LTP_scale_Q14
}

// StereoDecState mirrors silk/structs.h stereo_dec_state: the persistent decode-side
// stereo un-mixing state that silk_stereo_MS_to_LR carries across frames. pred_prev_Q13
// holds the two prediction weights from the previous frame (the start point the 8 ms
// interpolation ramp walks from toward this frame's weights); sMid/sSide are the
// one-sample mid/side delay buffers (the last two samples of the previous frame,
// re-injected at the head of this frame so the 3-tap low-pass mid contribution and the
// output are continuous across the frame boundary). Field names follow the C. This is
// the decode-only stereo state; the encoder's larger stereo_enc_state is out of scope.
type StereoDecState struct {
	PredPrevQ13 [2]int16 // pred_prev_Q13
	SMid        [2]int16 // sMid
	SSide       [2]int16 // sSide
}

// DecoderSetFs configures the decode-relevant sampling-rate-dependent fields of
// the decoder state: the (sub)frame lengths, the LTP memory length, the LPC order,
// the NLSF codebook, the pitch-lag / pitch-contour iCDF pointers, and (on a rate
// change) the cross-frame synthesis reset silk/decoder_set_fs.c performs
// (first_frame_after_reset, lagPrev, LastGainIndex, prevSignalType, cleared outBuf
// and sLPC_Q14_buf). The resampler init (out of scope, later port) and prev_gain_Q16
// (reset by silk_init_decoder, i.e. the caller's one-time state init) are not touched
// here. nbSubfr is MAX_NB_SUBFR (20 ms) or MAX_NB_SUBFR/2 (10 ms); fsKHz is 8, 12
// or 16.
func (psDec *DecoderState) DecoderSetFs(fsKHz, nbSubfr int) {
	fsChanged := psDec.FsKHz != fsKHz
	psDec.NbSubfr = nbSubfr

	/* New (sub)frame length */
	psDec.SubfrLength = int(silkmath.Silk_SMULBB(subFrameLengthMS, int32(fsKHz)))
	psDec.FrameLength = int(silkmath.Silk_SMULBB(int32(nbSubfr), int32(psDec.SubfrLength)))

	if fsKHz == 8 {
		if nbSubfr == maxNBSubfr {
			psDec.pitchContourICDF = silkPitchContourNBICDF
		} else {
			psDec.pitchContourICDF = silkPitchContour10msNBICDF
		}
	} else {
		if nbSubfr == maxNBSubfr {
			psDec.pitchContourICDF = silkPitchContourICDF
		} else {
			psDec.pitchContourICDF = silkPitchContour10msICDF
		}
	}
	psDec.LtpMemLength = int(silkmath.Silk_SMULBB(ltpMemLengthMS, int32(fsKHz)))
	if fsKHz == 8 || fsKHz == 12 {
		psDec.LPCOrder = minLPCOrder
		psDec.psNLSFCB = &silkNLSFCBNBMB
	} else {
		psDec.LPCOrder = maxLPCOrder
		psDec.psNLSFCB = &silkNLSFCBWB
	}
	switch fsKHz {
	case 16:
		psDec.pitchLagLowBitsICDF = silkUniform8ICDF
	case 12:
		psDec.pitchLagLowBitsICDF = silkUniform6ICDF
	case 8:
		psDec.pitchLagLowBitsICDF = silkUniform4ICDF
	}
	if fsChanged {
		/* Reset the cross-frame synthesis state on a rate change (decoder_set_fs.c
		   lines 91-96). prev_gain_Q16 is reset separately by silk_init_decoder. */
		psDec.FirstFrameAfterReset = 1
		psDec.LagPrev = 100
		psDec.LastGainIndex = 10
		psDec.PrevSignalType = typeNoVoiceActivity
		for i := range psDec.OutBuf {
			psDec.OutBuf[i] = 0
		}
		for i := range psDec.SLPCQ14Buf {
			psDec.SLPCQ14Buf[i] = 0
		}
	}
	psDec.FsKHz = fsKHz
}
