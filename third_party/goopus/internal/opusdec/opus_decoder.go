/* Copyright (c) 2010 Xiph.Org Foundation, Skype Limited
   Copyright (c) 2024 Arm Limited
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

package opusdec

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/celt"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/packet"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/silk"
)

// Opus error codes (include/opus_defines.h). opusDecodeFrame / opusDecodeNative
// return either a non-negative sample count or one of these negative codes,
// exactly as the C functions do.
const (
	opusOK             = 0  // OPUS_OK
	opusBadArg         = -1 // OPUS_BAD_ARG
	opusBufferTooSmall = -2 // OPUS_BUFFER_TOO_SMALL
	opusInternalError  = -3 // OPUS_INTERNAL_ERROR
	opusInvalidPacket  = -4 // OPUS_INVALID_PACKET
)

// Coding modes (src/opus_private.h:148-150).
const (
	modeSilkOnly = 1000 // MODE_SILK_ONLY
	modeHybrid   = 1001 // MODE_HYBRID
	modeCeltOnly = 1002 // MODE_CELT_ONLY
)

// Audio bandwidths (include/opus_defines.h:230-234).
const (
	opusBandwidthNarrowband    = 1101 // OPUS_BANDWIDTH_NARROWBAND
	opusBandwidthMediumband    = 1102 // OPUS_BANDWIDTH_MEDIUMBAND
	opusBandwidthWideband      = 1103 // OPUS_BANDWIDTH_WIDEBAND
	opusBandwidthSuperwideband = 1104 // OPUS_BANDWIDTH_SUPERWIDEBAND
	opusBandwidthFullband      = 1105 // OPUS_BANDWIDTH_FULLBAND
)

// SILK lost-flag protocol values (silk/control.h): the lost_flag argument to
// silk_Decode selects normal / packet-loss / FEC(LBRR) decoding.
const (
	silkFlagDecodeNormal = 0 // FLAG_DECODE_NORMAL
	silkFlagPacketLost   = 1 // FLAG_PACKET_LOST
	silkFlagDecodeLBRR   = 2 // FLAG_DECODE_LBRR
)

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// opusPacketGetMode is opus_packet_get_mode (opus_decoder.c:256-269): the coding
// mode carried by the TOC byte.
func opusPacketGetMode(toc byte) int {
	var mode int
	if toc&0x80 != 0 {
		mode = modeCeltOnly
	} else if (toc & 0x60) == 0x60 {
		mode = modeHybrid
	} else {
		mode = modeSilkOnly
	}
	return mode
}

// opusPacketGetBandwidth is opus_packet_get_bandwidth (opus_decoder.c:1250-1266).
func opusPacketGetBandwidth(toc byte) int {
	var bandwidth int
	switch {
	case toc&0x80 != 0:
		bandwidth = opusBandwidthMediumband + int((toc>>5)&0x3)
		if bandwidth == opusBandwidthMediumband {
			bandwidth = opusBandwidthNarrowband
		}
	case (toc & 0x60) == 0x60:
		if toc&0x10 != 0 {
			bandwidth = opusBandwidthFullband
		} else {
			bandwidth = opusBandwidthSuperwideband
		}
	default:
		bandwidth = opusBandwidthNarrowband + int((toc>>5)&0x3)
	}
	return bandwidth
}

// opusPacketGetNbChannels is opus_packet_get_nb_channels (opus_decoder.c:1268).
func opusPacketGetNbChannels(toc byte) int {
	if toc&0x4 != 0 {
		return 2
	}
	return 1
}

// opusPacketGetSamplesPerFrame is opus_packet_get_samples_per_frame (src/opus.c):
// the per-channel sample count of one frame at sample rate Fs.
func opusPacketGetSamplesPerFrame(toc byte, Fs int) int {
	var audiosize int
	switch {
	case toc&0x80 != 0:
		audiosize = int((toc >> 3) & 0x3)
		audiosize = (Fs << audiosize) / 400
	case (toc & 0x60) == 0x60:
		if toc&0x08 != 0 {
			audiosize = Fs / 50
		} else {
			audiosize = Fs / 100
		}
	default:
		audiosize = int((toc >> 3) & 0x3)
		if audiosize == 3 {
			audiosize = Fs * 60 / 1000
		} else {
			audiosize = (Fs << audiosize) / 100
		}
	}
	return audiosize
}

// OpusDecoder mirrors struct OpusDecoder (opus_decoder.c:65-94) for the frozen
// FIXED_POINT + DISABLE_FLOAT_API build: the SILK + CELT decoder handles, the
// DecControl for SILK, and the small transition state (mode / prev_mode /
// prev_redundancy / bandwidth / frame_size / last_packet_duration / rangeFinal).
// The reset boundary OPUS_DECODER_RESET_START is stream_channels.
type OpusDecoder struct {
	silkDec *silk.Decoder
	celtDec *celt.Decoder

	channels   int
	Fs         int32 // Sampling rate (at the API level)
	decControl silk.DecControlStruct
	decodeGain int
	complexity int

	/* Everything beyond this point gets cleared on a reset, except pkt and
	   pktFrames at the end: those are overwritten on every parse (see their
	   comment), so ResetState leaves them alone. */
	/* OPUS_DECODER_RESET_START stream_channels */
	streamChannels int

	bandwidth          int
	mode               int
	prevMode           int
	frameSize          int
	prevRedundancy     int
	lastPacketDuration int

	rangeFinal uint32

	// Per-packet scratch for packet.ParseInto, so opusDecodeNative parses without
	// allocating (C parses into a stack opus_packet). pktFrames backs pkt.Frames;
	// both are overwritten on every parse and carry nothing across calls, so a reset
	// need not touch them. Not for concurrent use, like the decoder that owns them.
	pkt       packet.Packet
	pktFrames [packet.MaxFrames][]byte
}

// NewDecoder is opus_decoder_create + opus_decoder_init (opus_decoder.c:135-217):
// allocate and initialize an Opus decoder at API rate Fs (one of
// 8000/12000/16000/24000/48000) with 1 or 2 channels. Returns a nil decoder and
// an error (ErrBadArg / ErrInternal) for an invalid config.
func NewDecoder(Fs int32, channels int) (*OpusDecoder, error) {
	st, code := newDecoder(Fs, channels)
	if code != opusOK {
		return nil, codeErr(code)
	}
	return st, nil
}

// newDecoder is the int-code opus_decoder_init body; NewDecoder wraps it.
func newDecoder(Fs int32, channels int) (*OpusDecoder, int) {
	if (Fs != 48000 && Fs != 24000 && Fs != 16000 && Fs != 12000 && Fs != 8000) ||
		(channels != 1 && channels != 2) {
		return nil, opusBadArg
	}

	st := &OpusDecoder{}
	st.streamChannels = channels
	st.channels = channels
	st.complexity = 0

	st.Fs = Fs
	st.decControl.APISampleRate = st.Fs
	st.decControl.NChannelsAPI = st.channels

	/* Reset decoder */
	st.silkDec = &silk.Decoder{}
	st.silkDec.InitDecoder()

	/* Initialize CELT decoder */
	st.celtDec = celt.NewDecoder(channels)
	if st.celtDec == nil {
		return nil, opusInternalError
	}
	// celt_decoder_init(celt_dec, Fs, channels): CELT always runs the 48 kHz mode
	// and serves API rates 8-24 kHz by integer decimation in deemphasis. Set the
	// downsample factor resampling_factor(Fs); 48000 keeps factor 1.
	if ds := 48000 / int(Fs); ds != 1 {
		if err := st.celtDec.SetDownsample(ds); err != nil {
			return nil, opusInternalError
		}
	}
	/* celt_decoder_ctl(celt_dec, CELT_SET_SIGNALLING(0)) is a decoder no-op in
	   this port: the CELT decode path never reads st->signalling. */

	st.prevMode = 0
	st.frameSize = int(Fs) / 400
	return st, opusOK
}

// ResetState is opus_decoder_ctl(OPUS_RESET_STATE) (opus_decoder.c:1109-1122):
// clear everything from OPUS_DECODER_RESET_START on, reset the CELT and SILK
// decoders, and restore stream_channels / frame_size.
func (st *OpusDecoder) ResetState() {
	st.streamChannels = 0
	st.bandwidth = 0
	st.mode = 0
	st.prevMode = 0
	st.frameSize = 0
	st.prevRedundancy = 0
	st.lastPacketDuration = 0
	st.rangeFinal = 0

	st.celtDec.ResetState()
	st.silkDec.ResetDecoder()
	st.streamChannels = st.channels
	st.frameSize = int(st.Fs) / 400
}

// FinalRange returns st->rangeFinal (OPUS_GET_FINAL_RANGE): the range-coder state
// after the last decoded frame, the per-packet bit-exactness check.
func (st *OpusDecoder) FinalRange() uint32 { return st.rangeFinal }

// LastPacketDuration returns st->last_packet_duration
// (OPUS_GET_LAST_PACKET_DURATION): the per-channel sample count of the last
// decode.
func (st *OpusDecoder) LastPacketDuration() int { return st.lastPacketDuration }

// Channels returns the physical output channel count.
func (st *OpusDecoder) Channels() int { return st.channels }

// SampleRate returns the API sample rate in Hz.
func (st *OpusDecoder) SampleRate() int { return int(st.Fs) }

// smoothFade is the non-RES24 smooth_fade (opus_decoder.c:237-254). It crossfades
// in1 -> in2 over overlap samples per channel, weighting by the squared mode
// window sampled at the API rate (inc = 48000/Fs). celt_coef is opus_int16 in
// this build, so COEF2VAL16 is the identity and the arithmetic is the int16-operand
// MULT16 form. out may alias in1 (each index is read before it is written).
func smoothFade(in1, in2, out []int16, overlap, channels int, window []int16, Fs int) {
	inc := 48000 / Fs
	for c := 0; c < channels; c++ {
		for i := 0; i < overlap; i++ {
			w := int32(window[i*inc])
			w = (w * w) >> 15 // MULT16_16_Q15(w, w)
			// SHR32(MAC16_16(MULT16_16(w,in2), Q15ONE-w, in1), 15)
			v := (w*int32(in2[i*channels+c]) + (32767-w)*int32(in1[i*channels+c])) >> 15
			out[i*channels+c] = int16(v)
		}
	}
}

// opusDecodeFrame is opus_decode_frame (opus_decoder.c:271-714): decode one Opus
// frame, routing to CELT-only / SILK-only / hybrid, handling mode transitions,
// the CELT redundancy frames and their crossfades, and PLC (data==NULL) / FEC
// (decodeFec). data is the frame bytes (nil for PLC); its C length "len" is
// len(data) at entry and is mutated locally as len below. pcm receives the
// interleaved int16 output. Returns the per-channel sample count or a negative
// Opus error code.
func (st *OpusDecoder) opusDecodeFrame(data []byte, pcm []int16, frameSize, decodeFec int) int {
	silkDec := st.silkDec
	celtDec := st.celtDec
	var silkRet, celtRet int
	var dec rangecoding.Decoder
	var silkFrameSize int

	length := len(data)
	if data == nil {
		length = 0
	}

	var pcmTransitionSilk []int16
	var pcmTransitionCelt []int16
	var pcmTransition []int16
	var redundantAudio []int16

	var audiosize int
	var mode int
	var bandwidth int
	transition := 0
	redundancy := 0
	redundancyBytes := 0
	celtToSilk := 0
	var window []int16
	var redundantRng uint32
	var celtAccum int

	F20 := int(st.Fs) / 50
	F10 := F20 >> 1
	F5 := F10 >> 1
	F2_5 := F5 >> 1
	if frameSize < F2_5 {
		return opusBufferTooSmall
	}
	/* Limit frame_size to avoid excessive stack allocations. */
	frameSize = imin(frameSize, int(st.Fs)/25*3)
	/* Payloads of 1 (2 including ToC) or 0 trigger the PLC/DTX */
	if length <= 1 {
		data = nil
		/* In that case, don't conceal more than what the ToC says */
		frameSize = imin(frameSize, st.frameSize)
	}
	if data != nil {
		audiosize = st.frameSize
		mode = st.mode
		bandwidth = st.bandwidth
		dec.Init(data)
	} else {
		audiosize = frameSize
		/* Run PLC using last used mode (CELT if we ended with CELT redundancy) */
		if st.prevRedundancy != 0 {
			mode = modeCeltOnly
		} else {
			mode = st.prevMode
		}
		bandwidth = 0

		if mode == 0 {
			/* If we haven't got any packet yet, all we can do is return zeros */
			for i := 0; i < audiosize*st.channels; i++ {
				pcm[i] = 0
			}
			return audiosize
		}

		/* Avoids trying to run the PLC on sizes other than 2.5 (CELT), 5 (CELT),
		   10, or 20 (e.g. 12.5 or 30 ms). */
		if audiosize > F20 {
			for {
				ret := st.opusDecodeFrame(nil, pcm, imin(audiosize, F20), 0)
				if ret < 0 {
					return ret
				}
				pcm = pcm[ret*st.channels:]
				audiosize -= ret
				if audiosize <= 0 {
					break
				}
			}
			return frameSize
		} else if audiosize < F20 {
			if audiosize > F10 {
				audiosize = F10
			} else if mode != modeSilkOnly && audiosize > F5 && audiosize < F10 {
				audiosize = F5
			}
		}
	}

	/* In fixed-point, we can tell CELT to do the accumulation on top of the
	   SILK PCM buffer. This saves some stack space. */
	if mode != modeCeltOnly {
		celtAccum = 1
	}

	if data != nil && st.prevMode > 0 &&
		((mode == modeCeltOnly && st.prevMode != modeCeltOnly && st.prevRedundancy == 0) ||
			(mode != modeCeltOnly && st.prevMode == modeCeltOnly)) {
		transition = 1
		/* Decide where to allocate the stack memory for pcm_transition */
		if mode == modeCeltOnly {
			pcmTransitionCelt = make([]int16, F5*st.channels)
		} else {
			pcmTransitionSilk = make([]int16, F5*st.channels)
		}
	}
	if transition != 0 && mode == modeCeltOnly {
		pcmTransition = pcmTransitionCelt
		st.opusDecodeFrame(nil, pcmTransition, imin(F5, audiosize), 0)
	}
	if audiosize > frameSize {
		return opusBadArg
	}
	frameSize = audiosize

	/* SILK processing */
	if mode != modeCeltOnly {
		var lostFlag, decodedSamples int
		var pcmPtr []int16
		pcmTooSmall := frameSize < F10
		var pcmSilk []int16
		if pcmTooSmall {
			pcmSilk = make([]int16, F10*st.channels)
			pcmPtr = pcmSilk
		} else {
			pcmPtr = pcm
		}

		if st.prevMode == modeCeltOnly {
			silkDec.ResetDecoder()
		}

		/* The SILK PLC cannot produce frames of less than 10 ms */
		st.decControl.PayloadSizeMs = imax(10, 1000*audiosize/int(st.Fs))

		if data != nil {
			st.decControl.NChannelsInternal = st.streamChannels
			if mode == modeSilkOnly {
				switch bandwidth {
				case opusBandwidthNarrowband:
					st.decControl.InternalSampleRate = 8000
				case opusBandwidthMediumband:
					st.decControl.InternalSampleRate = 12000
				case opusBandwidthWideband:
					st.decControl.InternalSampleRate = 16000
				default:
					st.decControl.InternalSampleRate = 16000
					/* celt_assert( 0 ) */
				}
			} else {
				/* Hybrid mode */
				st.decControl.InternalSampleRate = 16000
			}
		}
		st.decControl.EnableDeepPLC = 0 // st->complexity >= 5, deep PLC out of scope

		if data == nil {
			lostFlag = silkFlagPacketLost
		} else if decodeFec != 0 {
			lostFlag = silkFlagDecodeLBRR
		} else {
			lostFlag = silkFlagDecodeNormal
		}
		decodedSamples = 0
		for {
			/* Call SILK decoder */
			firstFrame := 0
			if decodedSamples == 0 {
				firstFrame = 1
			}
			var nSilk int32
			nSilk, silkRet = silkDec.Decode(&st.decControl, lostFlag, firstFrame, &dec,
				pcmPtr[decodedSamples*st.channels:])
			silkFrameSize = int(nSilk)
			if silkRet != 0 {
				if lostFlag != 0 {
					/* PLC failure should not be fatal */
					silkFrameSize = frameSize
					for i := 0; i < frameSize*st.channels; i++ {
						pcmPtr[decodedSamples*st.channels+i] = 0
					}
				} else {
					return opusInternalError
				}
			}
			decodedSamples += silkFrameSize
			if decodedSamples >= frameSize {
				break
			}
		}
		if pcmTooSmall {
			copy(pcm[:frameSize*st.channels], pcmSilk[:frameSize*st.channels])
		}
	}

	startBand := 0
	if decodeFec == 0 && mode != modeCeltOnly && data != nil &&
		dec.Tell()+17+20*b2i(mode == modeHybrid) <= 8*length {
		/* Check if we have a redundant 0-8 kHz band */
		if mode == modeHybrid {
			redundancy = dec.DecBitLogp(12)
		} else {
			redundancy = 1
		}
		if redundancy != 0 {
			celtToSilk = dec.DecBitLogp(1)
			/* redundancy_bytes will be at least two, in the non-hybrid
			   case due to the ec_tell() check above */
			if mode == modeHybrid {
				redundancyBytes = int(dec.DecUint(256)) + 2
			} else {
				redundancyBytes = length - ((dec.Tell() + 7) >> 3)
			}
			length -= redundancyBytes
			/* This is a sanity check. It should never happen for a valid
			   packet, so the exact behaviour is not normative. */
			if length*8 < dec.Tell() {
				length = 0
				redundancyBytes = 0
				redundancy = 0
			}
			/* Shrink decoder because of raw bits */
			dec.ShrinkStorage(uint32(redundancyBytes))
		}
	}
	if mode != modeCeltOnly {
		startBand = 17
	}

	if redundancy != 0 {
		transition = 0
		pcmTransitionSilk = nil
	}

	if transition != 0 && mode != modeCeltOnly {
		pcmTransition = pcmTransitionSilk
		st.opusDecodeFrame(nil, pcmTransition, imin(F5, audiosize), 0)
	}

	if bandwidth != 0 {
		endband := 21
		switch bandwidth {
		case opusBandwidthNarrowband:
			endband = 13
		case opusBandwidthMediumband, opusBandwidthWideband:
			endband = 17
		case opusBandwidthSuperwideband:
			endband = 19
		case opusBandwidthFullband:
			endband = 21
		default:
			/* celt_assert(0) */
		}
		if err := celtDec.SetEndBand(endband); err != nil {
			return opusInternalError
		}
	}
	if err := celtDec.SetStreamChannels(st.streamChannels); err != nil {
		return opusInternalError
	}

	/* Only allocation memory for redundancy if/when needed */
	if redundancy != 0 {
		redundantAudio = make([]int16, F5*st.channels)
	}

	/* 5 ms redundant frame for CELT->SILK */
	if redundancy != 0 && celtToSilk != 0 {
		/* If the previous frame did not use CELT (the first redundancy frame in
		   a transition from SILK may have been lost) then the CELT decoder is
		   stale at this point and the redundancy audio is not useful, however
		   the final range is still needed (for testing), so the redundancy is
		   always decoded but the decoded audio may not be used */
		if err := celtDec.SetStartBand(0); err != nil {
			return opusInternalError
		}
		if _, err := celtDec.DecodeWithEC(data[length:length+redundancyBytes], redundantAudio, F5, nil, 0); err != nil {
			return opusInternalError
		}
		redundantRng = celtDec.Rng()
	}

	/* MUST be after PLC */
	if err := celtDec.SetStartBand(startBand); err != nil {
		return opusInternalError
	}

	if mode != modeSilkOnly {
		celtFrameSize := imin(F20, frameSize)
		/* Make sure to discard any previous CELT state */
		if mode != st.prevMode && st.prevMode > 0 && st.prevRedundancy == 0 {
			celtDec.ResetState()
		}
		/* Decode CELT. In C this is "decode_fec ? NULL : data"; data itself is
		   NULL on the PLC/DTX path (len<=1), so guard on data != nil too before
		   slicing off the redundancy tail. A nil celtData routes CELT to its
		   loss concealment, exactly as a NULL data pointer does in C. */
		var celtData []byte
		if decodeFec == 0 && data != nil {
			celtData = data[:length]
		}
		var err error
		celtRet, err = celtDec.DecodeWithEC(celtData, pcm, celtFrameSize, &dec, celtAccum)
		if err != nil {
			celtRet = opusInternalError
		}
		st.rangeFinal = celtDec.Rng()
	} else {
		silence := []byte{0xFF, 0xFF}
		if celtAccum == 0 {
			for i := 0; i < frameSize*st.channels; i++ {
				pcm[i] = 0
			}
		}
		/* For hybrid -> SILK transitions, we let the CELT MDCT
		   do a fade-out by decoding a silence frame */
		if st.prevMode == modeHybrid && !(redundancy != 0 && celtToSilk != 0 && st.prevRedundancy != 0) {
			if err := celtDec.SetStartBand(0); err != nil {
				return opusInternalError
			}
			if _, err := celtDec.DecodeWithEC(silence, pcm, F2_5, nil, celtAccum); err != nil {
				return opusInternalError
			}
		}
		st.rangeFinal = dec.Rng()
	}

	window = celtDec.Window()

	/* 5 ms redundant frame for SILK->CELT */
	if redundancy != 0 && celtToSilk == 0 {
		celtDec.ResetState()
		if err := celtDec.SetStartBand(0); err != nil {
			return opusInternalError
		}
		if _, err := celtDec.DecodeWithEC(data[length:length+redundancyBytes], redundantAudio, F5, nil, 0); err != nil {
			return opusInternalError
		}
		redundantRng = celtDec.Rng()
		smoothFade(pcm[st.channels*(frameSize-F2_5):], redundantAudio[st.channels*F2_5:],
			pcm[st.channels*(frameSize-F2_5):], F2_5, st.channels, window, int(st.Fs))
	}
	/* 5ms redundant frame for CELT->SILK; ignore if the previous frame did not
	   use CELT (the first redundancy frame in a transition from SILK may have
	   been lost) */
	if redundancy != 0 && celtToSilk != 0 && (st.prevMode != modeSilkOnly || st.prevRedundancy != 0) {
		for c := 0; c < st.channels; c++ {
			for i := 0; i < F2_5; i++ {
				pcm[st.channels*i+c] = redundantAudio[st.channels*i+c]
			}
		}
		smoothFade(redundantAudio[st.channels*F2_5:], pcm[st.channels*F2_5:],
			pcm[st.channels*F2_5:], F2_5, st.channels, window, int(st.Fs))
	}
	if transition != 0 {
		if audiosize >= F5 {
			for i := 0; i < st.channels*F2_5; i++ {
				pcm[i] = pcmTransition[i]
			}
			smoothFade(pcmTransition[st.channels*F2_5:], pcm[st.channels*F2_5:],
				pcm[st.channels*F2_5:], F2_5, st.channels, window, int(st.Fs))
		} else {
			/* Not enough time to do a clean transition, but we do it anyway
			   This will not preserve amplitude perfectly and may introduce
			   a bit of temporal aliasing, but it shouldn't be too bad and
			   that's pretty much the best we can do. In any case, generating this
			   transition it pretty silly in the first place */
			smoothFade(pcmTransition, pcm, pcm, F2_5, st.channels, window, int(st.Fs))
		}
	}

	/* decode_gain (OPUS_SET_GAIN, opus_decoder.c:681-695) is omitted: the public
	   API exposes no gain control, so st->decode_gain is always 0 and the gain
	   multiply is a no-op. */

	if st.decodeGain != 0 {
		return opusInternalError // unreachable: no API sets decode_gain
	}

	if length <= 1 {
		st.rangeFinal = 0
	} else {
		st.rangeFinal ^= redundantRng
	}

	st.prevMode = mode
	st.prevRedundancy = b2i(redundancy != 0 && celtToSilk == 0)

	if celtRet < 0 {
		return celtRet
	}
	return audiosize
}

// opusDecodeNative is opus_decode_native (opus_decoder.c:716-879) for the frozen
// build (no self_delimited / soft_clip / DRED): packet parse, the FEC entry, and
// the multi-frame loop. data is the packet (nil/empty for PLC), pcm the output,
// frameSize the caller's per-channel buffer capacity, decodeFec the FEC request.
// Returns the per-channel sample count or a negative Opus error code.
func (st *OpusDecoder) opusDecodeNative(data []byte, pcm []int16, frameSize, decodeFec int) int {
	length := len(data)

	if decodeFec < 0 || decodeFec > 1 {
		return opusBadArg
	}
	/* For FEC/PLC, frame_size has to be to have a multiple of 2.5 ms */
	if (decodeFec != 0 || length == 0 || data == nil) && frameSize%(int(st.Fs)/400) != 0 {
		return opusBadArg
	}
	if length == 0 || data == nil {
		pcmCount := 0
		for {
			ret := st.opusDecodeFrame(nil, pcm[pcmCount*st.channels:], frameSize-pcmCount, 0)
			if ret < 0 {
				return ret
			}
			pcmCount += ret
			if pcmCount >= frameSize {
				break
			}
		}
		/* celt_assert(pcm_count == frame_size) */
		st.lastPacketDuration = pcmCount
		return pcmCount
	} else if length < 0 {
		return opusBadArg
	}

	packetMode := opusPacketGetMode(data[0])
	packetBandwidth := opusPacketGetBandwidth(data[0])
	packetFrameSize := opusPacketGetSamplesPerFrame(data[0], int(st.Fs))
	packetStreamChannels := opusPacketGetNbChannels(data[0])

	// Parse into decoder-held storage (no per-packet allocation). The recursive
	// opusDecodeNative calls on the FEC/PLC path below always pass data==nil and
	// return before reaching here, so they never clobber st.pkt while it is live.
	if err := packet.ParseInto(data, &st.pkt, &st.pktFrames); err != nil {
		return opusInvalidPacket
	}
	p := &st.pkt
	count := len(p.Frames)
	// The frame slices alias the caller's data buffer and are consumed entirely
	// within this call; drop the references on the way out so an idle decoder
	// does not pin the last packet's backing buffer until the next parse.
	defer func() {
		clear(st.pktFrames[:count])
		st.pkt.Frames = nil
		st.pkt.Padding = nil
	}()

	if decodeFec != 0 {
		/* If no FEC can be present, run the PLC (recursive call) */
		if frameSize < packetFrameSize || packetMode == modeCeltOnly || st.mode == modeCeltOnly {
			return st.opusDecodeNative(nil, pcm, frameSize, 0)
		}
		/* Otherwise, run the PLC on everything except the size for which we might have FEC */
		durationCopy := st.lastPacketDuration
		if frameSize-packetFrameSize != 0 {
			ret := st.opusDecodeNative(nil, pcm, frameSize-packetFrameSize, 0)
			if ret < 0 {
				st.lastPacketDuration = durationCopy
				return ret
			}
			/* celt_assert(ret==frame_size-packet_frame_size) */
		}
		/* Complete with FEC */
		st.mode = packetMode
		st.bandwidth = packetBandwidth
		st.frameSize = packetFrameSize
		st.streamChannels = packetStreamChannels
		ret := st.opusDecodeFrame(p.Frames[0], pcm[st.channels*(frameSize-packetFrameSize):],
			packetFrameSize, 1)
		if ret < 0 {
			return ret
		}
		st.lastPacketDuration = frameSize
		return frameSize
	}

	if count*packetFrameSize > frameSize {
		return opusBufferTooSmall
	}

	/* Update the state as the last step to avoid updating it on an invalid packet */
	st.mode = packetMode
	st.bandwidth = packetBandwidth
	st.frameSize = packetFrameSize
	st.streamChannels = packetStreamChannels

	nbSamples := 0
	for i := 0; i < count; i++ {
		ret := st.opusDecodeFrame(p.Frames[i], pcm[nbSamples*st.channels:], frameSize-nbSamples, 0)
		if ret < 0 {
			return ret
		}
		/* celt_assert(ret==packet_frame_size) */
		nbSamples += ret
	}
	st.lastPacketDuration = nbSamples
	return nbSamples
}

// Decode is opus_decode (opus_decoder.c:887-894, the FIXED_POINT non-RES24 path):
// decode one packet into interleaved int16 pcm. frameSize is the caller's
// per-channel buffer capacity; a nil/empty packet requests PLC. Returns the
// per-channel sample count decoded, or an error.
func (st *OpusDecoder) Decode(data []byte, pcm []int16, frameSize, decodeFec int) (int, error) {
	if frameSize <= 0 {
		return 0, codeErr(opusBadArg)
	}
	ret := st.opusDecodeNative(data, pcm, frameSize, decodeFec)
	if ret < 0 {
		return 0, codeErr(ret)
	}
	return ret, nil
}

// b2i converts a bool to 0/1, standing in for C's implicit bool-to-int coercion.
func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
