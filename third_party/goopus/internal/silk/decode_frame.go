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

// Transliteration of silk/decode_frame.c (libopus v1.6.1): the per-frame decode
// glue. It sets up the decoder control, decodes the side-information indices, the
// excitation pulses and the prediction parameters, then runs the core synthesis
// (silk_decode_core) and updates the persistent output buffer.
//
// Scope for this port: both the normal (good-frame) decode path and the packet-loss
// path. A good frame decodes the indices/pulses/parameters, runs the core synthesis
// (silk_decode_core), updates the output buffer, then runs silk_PLC (state update).
// A lost frame (lostFlag != FLAG_DECODE_NORMAL) is handled by extrapolation in
// silk_PLC (concealment) followed by the output-buffer update. Both paths then run
// silk_CNG (comfort-noise estimation / generation) and silk_PLC_glue_frames
// (energy match across the loss boundary), and update lagPrev. FEC / LBRR decoding
// (lostFlag == FLAG_DECODE_LBRR) is out of scope: the LBRR-present sub-branch is not
// wired, so an LBRR request without decoded data falls through to concealment.
//
// arch is dropped (scalar-only port); the OSCE / deep-PLC hooks are out of scope.
// Names and control flow follow the C for diffability.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"

// Loss flags from silk/control.h, matching the lostFlag argument.
const (
	flagDecodeNormal = 0 // FLAG_DECODE_NORMAL
	flagPacketLost   = 1 // FLAG_PACKET_LOST
	flagDecodeLBRR   = 2 // FLAG_DECODE_LBRR
)

// DecodeFrame is silk/decode_frame.c silk_decode_frame for the normal decode path.
// psDec is the decoder state; psRangeDec is the range decoder positioned at this
// frame's payload; pOut receives the decoded speech (at least FrameLength samples);
// lostFlag selects normal / packet-loss / LBRR decoding; condCoding selects the
// conditional-coding mode. It returns the output frame length N (samples).
func DecodeFrame(psDec *DecoderState, psRangeDec *rangecoding.Decoder, pOut []int16, lostFlag, condCoding int) int {
	var psDecCtrl DecoderControl
	var L, mvLen int

	L = psDec.FrameLength
	psDecCtrl.LTPScaleQ14 = 0

	/* Safety checks */
	/* celt_assert( L > 0 && L <= MAX_FRAME_LENGTH ); */

	if lostFlag == flagDecodeNormal ||
		(lostFlag == flagDecodeLBRR && psDec.LBRRFlags[psDec.NFramesDecoded] == 1) {
		pulsesLen := (L + shellCodecFrameLength - 1) & ^(shellCodecFrameLength - 1)
		pulses := make([]int16, pulsesLen)
		/*********************************************/
		/* Decode quantization indices of side info  */
		/*********************************************/
		DecodeIndices(psDec, psRangeDec, psDec.NFramesDecoded, lostFlag, condCoding)

		/*********************************************/
		/* Decode quantization indices of excitation */
		/*********************************************/
		DecodePulses(psRangeDec, pulses, int(psDec.Indices.SignalType),
			int(psDec.Indices.QuantOffsetType), psDec.FrameLength)

		/********************************************/
		/* Decode parameters and pulse signal       */
		/********************************************/
		DecodeParameters(psDec, &psDecCtrl, condCoding)

		/********************************************************/
		/* Run inverse NSQ                                      */
		/********************************************************/
		DecodeCore(psDec, &psDecCtrl, pOut, pulses)

		/*************************/
		/* Update output buffer. */
		/*************************/
		/* celt_assert( psDec->ltp_mem_length >= psDec->frame_length ); */
		mvLen = psDec.LtpMemLength - psDec.FrameLength
		copy(psDec.OutBuf[:mvLen], psDec.OutBuf[psDec.FrameLength:psDec.FrameLength+mvLen])
		copy(psDec.OutBuf[mvLen:mvLen+psDec.FrameLength], pOut[:psDec.FrameLength])

		/********************************************************/
		/* Update PLC state                                     */
		/********************************************************/
		PLC(psDec, &psDecCtrl, pOut, 0)

		psDec.LossCnt = 0
		psDec.PrevSignalType = int(psDec.Indices.SignalType)

		/* A frame has been decoded without errors */
		psDec.FirstFrameAfterReset = 0
	} else {
		/* Handle packet loss by extrapolation */
		PLC(psDec, &psDecCtrl, pOut, 1)

		/*************************/
		/* Update output buffer. */
		/*************************/
		/* celt_assert( psDec->ltp_mem_length >= psDec->frame_length ); */
		mvLen = psDec.LtpMemLength - psDec.FrameLength
		copy(psDec.OutBuf[:mvLen], psDec.OutBuf[psDec.FrameLength:psDec.FrameLength+mvLen])
		copy(psDec.OutBuf[mvLen:mvLen+psDec.FrameLength], pOut[:psDec.FrameLength])
	}

	/************************************************/
	/* Comfort noise generation / estimation        */
	/************************************************/
	CNG(psDec, &psDecCtrl, pOut, L)

	/****************************************************************/
	/* Ensure smooth connection of extrapolated and good frames     */
	/****************************************************************/
	PLCGlueFrames(psDec, pOut, L)

	/* Update some decoder state variables */
	psDec.LagPrev = psDecCtrl.PitchL[psDec.NbSubfr-1]

	/* Set output frame length */
	return L
}
