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

// Transliteration of silk/dec_API.c (libopus v1.6.1): the top-level SILK decoder that
// ties the whole decode chain together. It carries the two per-channel decoder states
// and the stereo un-mixing state, and per call decodes ONE SILK frame per internal
// channel (the caller loops it nFramesPerPacket times per packet, as opus_decode_frame
// does), handling: the VAD/LBRR header on the first frame, stereo MS prediction decode
// and un-mixing, the FEC/LBRR redundant-frame path (FLAG_DECODE_LBRR) and the
// packet-loss concealment path (FLAG_PACKET_LOST), and finally resampling each channel
// to the API output rate and interleaving.
//
// The frozen FIXED_POINT + DISABLE_FLOAT_API build with ENABLE_OSCE / ENABLE_DEEP_PLC /
// ENABLE_OSCE_BWE off drops the OSCE model, the deep-PLC LPCNet hook and the OSCE
// bandwidth-extension resampler branch; opus_res is opus_int16 (no RES24), so INT16TORES
// is the identity and the API output is written as int16. Names and control flow follow
// the C for diffability.

package silk

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"
)

// Constants from silk/define.h and silk/errors.h.
const (
	decoderNumChannels = 2  // DECODER_NUM_CHANNELS
	maxAPIFSKHz        = 48 // MAX_API_FS_KHZ (no QEXT; 48 kHz cap)

	silkNoError                     = 0    // SILK_NO_ERROR
	silkDecInvalidSamplingFrequency = -200 // SILK_DEC_INVALID_SAMPLING_FREQUENCY
	silkDecInvalidFrameSize         = -203 // SILK_DEC_INVALID_FRAME_SIZE
)

// DecControlStruct mirrors the decode-relevant fields of silk/control.h
// silk_DecControlStruct: the caller-owned control the top-level decode reads and
// writes. The OSCE / deep-PLC members compile away in this build; PrevPitchLag is the
// one output field (pitch lag at 48 kHz, 0 if unvoiced).
type DecControlStruct struct {
	NChannelsAPI       int   // nChannelsAPI (1/2)
	NChannelsInternal  int   // nChannelsInternal (1/2)
	APISampleRate      int32 // API_sampleRate (Hz)
	InternalSampleRate int32 // internalSampleRate (Hz; 8000/12000/16000)
	PayloadSizeMs      int   // payloadSize_ms (10/20/40/60; 0 => assume loss, use 10 ms)
	PrevPitchLag       int   // prevPitchLag (O)
	EnableDeepPLC      int   // enable_deep_plc (I; unused, deep PLC out of scope)
}

// Decoder mirrors silk/dec_API.c's silk_decoder super struct: the two per-channel
// decoder states, the shared stereo un-mixing state, the current API/internal channel
// counts and the previous frame's decode_only_middle flag. The OSCE model member
// compiles away in this build.
type Decoder struct {
	ChannelState         [decoderNumChannels]DecoderState // channel_state
	SStereo              StereoDecState                   // sStereo
	NChannelsAPI         int                              // nChannelsAPI
	NChannelsInternal    int                              // nChannelsInternal
	PrevDecodeOnlyMiddle int                              // prev_decode_only_middle
}

// Decode is silk/dec_API.c silk_Decode: decode one SILK frame per internal channel.
// decControl carries the channel counts, API/internal rates and payload size; lostFlag
// selects normal / packet-loss / FEC(LBRR) decoding; newPacketFlag marks the first
// decoder call for a packet; psRangeDec is the range decoder over the packet (unread on
// the packet-loss path); samplesOut receives the API-rate int16 PCM (interleaved for
// stereo). It returns the per-channel API-rate sample count and the accumulated error
// code.
func (psDec *Decoder) Decode(
	decControl *DecControlStruct,
	lostFlag int,
	newPacketFlag int,
	psRangeDec *rangecoding.Decoder,
	samplesOut []int16,
) (nSamplesOut int32, ret int) {
	var (
		i, n             int
		decodeOnlyMiddle = 0
		nSamplesOutDec   int
		lbrrSymbol       int
		msPredQ13        [2]int32
		hasSide          bool
		stereoToMono     bool
		channelState     = &psDec.ChannelState
	)
	ret = silkNoError

	/* celt_assert( decControl->nChannelsInternal == 1 || decControl->nChannelsInternal == 2 ); */

	/**********************************/
	/* Test if first frame in payload */
	/**********************************/
	if newPacketFlag != 0 {
		for n = 0; n < decControl.NChannelsInternal; n++ {
			channelState[n].NFramesDecoded = 0 /* Used to count frames in packet */
		}
	}

	/* If Mono -> Stereo transition in bitstream: init state of second channel */
	if decControl.NChannelsInternal > psDec.NChannelsInternal {
		channelState[1].initDecoder()
	}

	stereoToMono = decControl.NChannelsInternal == 1 && psDec.NChannelsInternal == 2 &&
		(decControl.InternalSampleRate == 1000*int32(channelState[0].FsKHz))

	if channelState[0].NFramesDecoded == 0 {
		for n = 0; n < decControl.NChannelsInternal; n++ {
			var fsKHzDec int
			switch decControl.PayloadSizeMs {
			case 0:
				/* Assuming packet loss, use 10 ms */
				channelState[n].NFramesPerPacket = 1
				channelState[n].NbSubfr = 2
			case 10:
				channelState[n].NFramesPerPacket = 1
				channelState[n].NbSubfr = 2
			case 20:
				channelState[n].NFramesPerPacket = 1
				channelState[n].NbSubfr = 4
			case 40:
				channelState[n].NFramesPerPacket = 2
				channelState[n].NbSubfr = 4
			case 60:
				channelState[n].NFramesPerPacket = 3
				channelState[n].NbSubfr = 4
			default:
				/* celt_assert( 0 ); */
				return nSamplesOut, silkDecInvalidFrameSize
			}
			fsKHzDec = int(decControl.InternalSampleRate>>10) + 1
			if fsKHzDec != 8 && fsKHzDec != 12 && fsKHzDec != 16 {
				/* celt_assert( 0 ); */
				return nSamplesOut, silkDecInvalidSamplingFrequency
			}
			ret += channelState[n].silkDecoderSetFs(fsKHzDec, decControl.APISampleRate)
		}
	}

	if decControl.NChannelsAPI == 2 && decControl.NChannelsInternal == 2 &&
		(psDec.NChannelsAPI == 1 || psDec.NChannelsInternal == 1) {
		psDec.SStereo.PredPrevQ13 = [2]int16{}
		psDec.SStereo.SSide = [2]int16{}
		channelState[1].ResamplerState = channelState[0].ResamplerState
	}
	psDec.NChannelsAPI = decControl.NChannelsAPI
	psDec.NChannelsInternal = decControl.NChannelsInternal

	if decControl.APISampleRate > int32(maxAPIFSKHz)*1000 || decControl.APISampleRate < 8000 {
		ret = silkDecInvalidSamplingFrequency
		return nSamplesOut, ret
	}

	if lostFlag != flagPacketLost && channelState[0].NFramesDecoded == 0 {
		/* First decoder call for this payload */
		/* Decode VAD flags and LBRR flag */
		for n = 0; n < decControl.NChannelsInternal; n++ {
			for i = 0; i < channelState[n].NFramesPerPacket; i++ {
				channelState[n].VADFlags[i] = psRangeDec.DecBitLogp(1)
			}
			channelState[n].LBRRFlag = psRangeDec.DecBitLogp(1)
		}
		/* Decode LBRR flags */
		for n = 0; n < decControl.NChannelsInternal; n++ {
			channelState[n].LBRRFlags = [maxFramesPerPacket]int{}
			if channelState[n].LBRRFlag != 0 {
				if channelState[n].NFramesPerPacket == 1 {
					channelState[n].LBRRFlags[0] = 1
				} else {
					lbrrSymbol = psRangeDec.DecIcdf(silkLBRRFlagsICDFPtr[channelState[n].NFramesPerPacket-2], 8) + 1
					for i = 0; i < channelState[n].NFramesPerPacket; i++ {
						channelState[n].LBRRFlags[i] = int(silkmath.Silk_RSHIFT(int32(lbrrSymbol), i) & 1)
					}
				}
			}
		}

		if lostFlag == flagDecodeNormal {
			/* Regular decoding: skip all LBRR data */
			for i = 0; i < channelState[0].NFramesPerPacket; i++ {
				for n = 0; n < decControl.NChannelsInternal; n++ {
					if channelState[n].LBRRFlags[i] != 0 {
						pulses := make([]int16, maxFrameLength)
						var condCoding int

						if decControl.NChannelsInternal == 2 && n == 0 {
							StereoDecodePred(psRangeDec, &msPredQ13)
							if channelState[1].LBRRFlags[i] == 0 {
								decodeOnlyMiddle = StereoDecodeMidOnly(psRangeDec)
							}
						}
						/* Use conditional coding if previous frame available */
						if i > 0 && channelState[n].LBRRFlags[i-1] != 0 {
							condCoding = codeConditionally
						} else {
							condCoding = codeIndependently
						}
						DecodeIndices(&channelState[n], psRangeDec, i, 1, condCoding)
						DecodePulses(psRangeDec, pulses, int(channelState[n].Indices.SignalType),
							int(channelState[n].Indices.QuantOffsetType), channelState[n].FrameLength)
					}
				}
			}
		}
	}

	/* Get MS predictor index */
	if decControl.NChannelsInternal == 2 {
		if lostFlag == flagDecodeNormal ||
			(lostFlag == flagDecodeLBRR && channelState[0].LBRRFlags[channelState[0].NFramesDecoded] == 1) {
			StereoDecodePred(psRangeDec, &msPredQ13)
			/* For LBRR data, decode mid-only flag only if side-channel's LBRR flag is false */
			if (lostFlag == flagDecodeNormal && channelState[1].VADFlags[channelState[0].NFramesDecoded] == 0) ||
				(lostFlag == flagDecodeLBRR && channelState[1].LBRRFlags[channelState[0].NFramesDecoded] == 0) {
				decodeOnlyMiddle = StereoDecodeMidOnly(psRangeDec)
			} else {
				decodeOnlyMiddle = 0
			}
		} else {
			for n = 0; n < 2; n++ {
				msPredQ13[n] = int32(psDec.SStereo.PredPrevQ13[n])
			}
		}
	}

	/* Reset side channel decoder prediction memory for first frame with side coding */
	if decControl.NChannelsInternal == 2 && decodeOnlyMiddle == 0 && psDec.PrevDecodeOnlyMiddle == 1 {
		for j := range psDec.ChannelState[1].OutBuf {
			psDec.ChannelState[1].OutBuf[j] = 0
		}
		for j := range psDec.ChannelState[1].SLPCQ14Buf {
			psDec.ChannelState[1].SLPCQ14Buf[j] = 0
		}
		psDec.ChannelState[1].LagPrev = 100
		psDec.ChannelState[1].LastGainIndex = 10
		psDec.ChannelState[1].PrevSignalType = typeNoVoiceActivity
		psDec.ChannelState[1].FirstFrameAfterReset = 1
	}

	/* Check if the temp buffer fits into the output PCM buffer. If it fits,
	   we can delay allocating the temp buffer until after the SILK peak stack
	   usage. We need to use a < and not a <= because of the two extra samples. */
	tmpStride := channelState[0].FrameLength + 2
	samplesOut1TmpStorage := make([]int16, decControl.NChannelsInternal*tmpStride)
	var samplesOut1Tmp [2][]int16
	samplesOut1Tmp[0] = samplesOut1TmpStorage[0:tmpStride]
	if decControl.NChannelsInternal == 2 {
		samplesOut1Tmp[1] = samplesOut1TmpStorage[tmpStride : 2*tmpStride]
	}

	if lostFlag == flagDecodeNormal {
		hasSide = decodeOnlyMiddle == 0
	} else {
		hasSide = psDec.PrevDecodeOnlyMiddle == 0 ||
			(decControl.NChannelsInternal == 2 && lostFlag == flagDecodeLBRR &&
				channelState[1].LBRRFlags[channelState[1].NFramesDecoded] == 1)
	}
	/* channel_state[0].sPLC.enable_deep_plc = decControl->enable_deep_plc (deep PLC out of scope) */

	/* Call decoder for one frame */
	for n = 0; n < decControl.NChannelsInternal; n++ {
		if n == 0 || hasSide {
			var frameIndex, condCoding int

			frameIndex = channelState[0].NFramesDecoded - n
			/* Use independent coding if no previous frame available */
			if frameIndex <= 0 {
				condCoding = codeIndependently
			} else if lostFlag == flagDecodeLBRR {
				if channelState[n].LBRRFlags[frameIndex-1] != 0 {
					condCoding = codeConditionally
				} else {
					condCoding = codeIndependently
				}
			} else if n > 0 && psDec.PrevDecodeOnlyMiddle != 0 {
				/* If we skipped a side frame in this packet, we don't
				   need LTP scaling; the LTP state is well-defined. */
				condCoding = codeIndependentlyNoLTPScaling
			} else {
				condCoding = codeConditionally
			}
			nSamplesOutDec = DecodeFrame(&channelState[n], psRangeDec, samplesOut1Tmp[n][2:], lostFlag, condCoding)
		} else {
			for j := 0; j < nSamplesOutDec; j++ {
				samplesOut1Tmp[n][2+j] = 0
			}
		}
		channelState[n].NFramesDecoded++
	}

	if decControl.NChannelsAPI == 2 && decControl.NChannelsInternal == 2 {
		/* Convert Mid/Side to Left/Right */
		StereoMStoLR(&psDec.SStereo, samplesOut1Tmp[0], samplesOut1Tmp[1], &msPredQ13, channelState[0].FsKHz, nSamplesOutDec)
	} else {
		/* Buffering */
		copy(samplesOut1Tmp[0][0:2], psDec.SStereo.SMid[:])
		copy(psDec.SStereo.SMid[:], samplesOut1Tmp[0][nSamplesOutDec:nSamplesOutDec+2])
	}

	/* Number of output samples */
	nSamplesOut = silkmath.Silk_DIV32(int32(nSamplesOutDec)*decControl.APISampleRate,
		silkmath.Silk_SMULBB(int32(channelState[0].FsKHz), 1000))

	/* Set up pointers to temp buffers */
	samplesOut2Tmp := make([]int16, nSamplesOut)
	resampleOutPtr := samplesOut2Tmp

	for n = 0; n < silkmath.Silk_min_int(decControl.NChannelsAPI, decControl.NChannelsInternal); n++ {
		/* Resample decoded signal to API_sampleRate */
		ret += Resampler(&channelState[n].ResamplerState, resampleOutPtr, samplesOut1Tmp[n][1:], int32(nSamplesOutDec))

		/* Interleave if stereo output and stereo stream */
		if decControl.NChannelsAPI == 2 {
			for i = 0; i < int(nSamplesOut); i++ {
				samplesOut[n+2*i] = resampleOutPtr[i] /* INT16TORES identity */
			}
		} else {
			for i = 0; i < int(nSamplesOut); i++ {
				samplesOut[i] = resampleOutPtr[i]
			}
		}
	}

	/* Create two channel output from mono stream */
	if decControl.NChannelsAPI == 2 && decControl.NChannelsInternal == 1 {
		if stereoToMono {
			/* Resample right channel for newly collapsed stereo just in case
			   we weren't doing collapsing when switching to mono */
			ret += Resampler(&channelState[1].ResamplerState, resampleOutPtr, samplesOut1Tmp[0][1:], int32(nSamplesOutDec))

			for i = 0; i < int(nSamplesOut); i++ {
				samplesOut[1+2*i] = resampleOutPtr[i]
			}
		} else {
			for i = 0; i < int(nSamplesOut); i++ {
				samplesOut[1+2*i] = samplesOut[0+2*i]
			}
		}
	}

	/* Export pitch lag, measured at 48 kHz sampling rate */
	if channelState[0].PrevSignalType == typeVoiced {
		multTab := [3]int{6, 4, 3}
		decControl.PrevPitchLag = channelState[0].LagPrev * multTab[(channelState[0].FsKHz-8)>>2]
	} else {
		decControl.PrevPitchLag = 0
	}

	if lostFlag == flagPacketLost {
		/* On packet loss, remove the gain clamping to prevent having the energy "bounce back"
		   if we lose packets when the energy is going down */
		for i = 0; i < psDec.NChannelsInternal; i++ {
			psDec.ChannelState[i].LastGainIndex = 10
		}
	} else {
		psDec.PrevDecodeOnlyMiddle = decodeOnlyMiddle
	}

	return nSamplesOut, ret
}
