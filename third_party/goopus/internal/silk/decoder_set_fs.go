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

// Transliteration of silk/decoder_set_fs.c (libopus v1.6.1): silk_decoder_set_fs, the
// full sampling-rate reconfiguration the top-level silk_Decode runs once per packet.
// It sets the (sub)frame geometry, initializes the SILK-internal -> API-rate resampler
// (ResamplerInit), selects the rate-dependent tables (NLSF codebook, pitch-lag and
// pitch-contour iCDFs) and, on an internal-rate change, resets the cross-frame
// synthesis state. nb_subfr must already be set on psDec by the caller (as silk_Decode
// does from decControl->payloadSize_ms), matching the C which reads psDec->nb_subfr.
//
// The value-shaped geometry-only helper DecoderState.DecoderSetFs (structs.go) predates
// this port and is retained for the decode_core / decode_parameters unit tests that do
// not resample; this file is the full C function the dec_API path uses. Names and
// control flow follow the C for diffability.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// silkDecoderSetFs is silk/decoder_set_fs.c silk_decoder_set_fs. fsKHz is the SILK
// internal rate (8/12/16); fsAPIHz is the API output rate in Hz. Returns the
// accumulated error code (0 ok; a negative resampler-init error propagates).
func (psDec *DecoderState) silkDecoderSetFs(fsKHz int, fsAPIHz int32) int {
	var frameLength, ret int

	/* celt_assert( fs_kHz == 8 || fs_kHz == 12 || fs_kHz == 16 ); */
	/* celt_assert( psDec->nb_subfr == MAX_NB_SUBFR || psDec->nb_subfr == MAX_NB_SUBFR/2 ); */

	/* New (sub)frame length */
	psDec.SubfrLength = int(silkmath.Silk_SMULBB(subFrameLengthMS, int32(fsKHz)))
	frameLength = int(silkmath.Silk_SMULBB(int32(psDec.NbSubfr), int32(psDec.SubfrLength)))

	/* Initialize resampler when switching internal or external sampling frequency */
	if psDec.FsKHz != fsKHz || psDec.FsAPIhz != fsAPIHz {
		/* Initialize the resampler for dec_API.c preparing resampling from fs_kHz to API_fs_Hz */
		ret += ResamplerInit(&psDec.ResamplerState, silkmath.Silk_SMULBB(int32(fsKHz), 1000), fsAPIHz, 0)

		psDec.FsAPIhz = fsAPIHz
	}

	if psDec.FsKHz != fsKHz || frameLength != psDec.FrameLength {
		if fsKHz == 8 {
			if psDec.NbSubfr == maxNBSubfr {
				psDec.pitchContourICDF = silkPitchContourNBICDF
			} else {
				psDec.pitchContourICDF = silkPitchContour10msNBICDF
			}
		} else {
			if psDec.NbSubfr == maxNBSubfr {
				psDec.pitchContourICDF = silkPitchContourICDF
			} else {
				psDec.pitchContourICDF = silkPitchContour10msICDF
			}
		}
		if psDec.FsKHz != fsKHz {
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
			default:
				/* unsupported sampling rate */
				/* celt_assert( 0 ); */
			}
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
		psDec.FrameLength = frameLength
	}

	/* Check that settings are valid */
	/* celt_assert( psDec->frame_length > 0 && psDec->frame_length <= MAX_FRAME_LENGTH ); */

	return ret
}
