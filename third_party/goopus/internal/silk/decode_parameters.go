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

// Transliteration of silk/decode_parameters.c (libopus v1.6.1):
// silk_decode_parameters turns the decoded side-information indices into the
// per-frame prediction parameters (linear gains, the two interpolated Q12 LPC
// coefficient sets, pitch lags, LTP taps and LTP scale). It is a pure function of
// psDec.Indices plus the cross-frame state (prevNLSF_Q15, LastGainIndex): the range
// decoder is not touched here. The NLSF->LPC handoff reuses the already-committed
// NLSFDecode / NLSF2A / Bwexpander; gains and pitch use the helpers in this
// package. Names and control flow follow the C for diffability.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// DecodeParameters is silk/decode_parameters.c silk_decode_parameters. psDecCtrl
// receives the decoded prediction/coding parameters; condCoding selects the gain
// and LTP-scale coding mode (CODE_CONDITIONALLY delta-codes the first gain and
// suppresses independent LTP scaling).
func DecodeParameters(psDec *DecoderState, psDecCtrl *DecoderControl, condCoding int) {
	var i, k, Ix int
	var pNLSFQ15 [maxLPCOrder]int16
	var pNLSF0Q15 [maxLPCOrder]int16

	/* Dequant Gains */
	conditional := 0
	if condCoding == codeConditionally {
		conditional = 1
	}
	silkGainsDequant(psDecCtrl.GainsQ16[:], psDec.Indices.GainsIndices[:], &psDec.LastGainIndex, conditional, psDec.NbSubfr)

	/****************/
	/* Decode NLSFs */
	/****************/
	silkNLSFDecode(pNLSFQ15[:], psDec.Indices.NLSFIndices[:], psDec.psNLSFCB)

	/* Convert NLSF parameters to AR prediction filter coefficients */
	NLSF2A(psDecCtrl.PredCoefQ12[1][:], pNLSFQ15[:], psDec.LPCOrder)

	/* If just reset, e.g., because internal Fs changed, do not allow interpolation */
	/* improves the case of packet loss in the first frame after a switch           */
	if psDec.FirstFrameAfterReset == 1 {
		psDec.Indices.NLSFInterpCoefQ2 = 4
	}

	if psDec.Indices.NLSFInterpCoefQ2 < 4 {
		/* Calculation of the interpolated NLSF0 vector from the interpolation factor, */
		/* the previous NLSF1, and the current NLSF1                                   */
		for i = 0; i < psDec.LPCOrder; i++ {
			pNLSF0Q15[i] = int16(int32(psDec.PrevNLSFQ15[i]) + silkmath.Silk_RSHIFT(silkmath.Silk_MUL(int32(psDec.Indices.NLSFInterpCoefQ2),
				int32(pNLSFQ15[i])-int32(psDec.PrevNLSFQ15[i])), 2))
		}

		/* Convert NLSF parameters to AR prediction filter coefficients */
		NLSF2A(psDecCtrl.PredCoefQ12[0][:], pNLSF0Q15[:], psDec.LPCOrder)
	} else {
		/* Copy LPC coefficients for first half from second half */
		copy(psDecCtrl.PredCoefQ12[0][:psDec.LPCOrder], psDecCtrl.PredCoefQ12[1][:psDec.LPCOrder])
	}

	copy(psDec.PrevNLSFQ15[:psDec.LPCOrder], pNLSFQ15[:psDec.LPCOrder])

	/* After a packet loss do BWE of LPC coefs */
	if psDec.LossCnt != 0 {
		Bwexpander(psDecCtrl.PredCoefQ12[0][:], psDec.LPCOrder, bweAfterLossQ16)
		Bwexpander(psDecCtrl.PredCoefQ12[1][:], psDec.LPCOrder, bweAfterLossQ16)
	}

	if psDec.Indices.SignalType == typeVoiced {
		/*********************/
		/* Decode pitch lags */
		/*********************/

		/* Decode pitch values */
		silkDecodePitch(psDec.Indices.LagIndex, psDec.Indices.ContourIndex, psDecCtrl.PitchL[:], psDec.FsKHz, psDec.NbSubfr)

		/* Decode Codebook Index */
		cbkPtrQ7 := silkLTPVQPtrsQ7[psDec.Indices.PERIndex] /* set pointer to start of codebook */

		for k = 0; k < psDec.NbSubfr; k++ {
			Ix = int(psDec.Indices.LTPIndex[k])
			for i = 0; i < ltpOrder; i++ {
				psDecCtrl.LTPCoefQ14[k*ltpOrder+i] = int16(silkmath.Silk_LSHIFT(int32(cbkPtrQ7[Ix*ltpOrder+i]), 7))
			}
		}

		/**********************/
		/* Decode LTP scaling */
		/**********************/
		Ix = int(psDec.Indices.LTPScaleIndex)
		psDecCtrl.LTPScaleQ14 = int(silkLTPScalesTableQ14[Ix])
	} else {
		for k = 0; k < psDec.NbSubfr; k++ {
			psDecCtrl.PitchL[k] = 0
		}
		for i = 0; i < ltpOrder*psDec.NbSubfr; i++ {
			psDecCtrl.LTPCoefQ14[i] = 0
		}
		psDec.Indices.PERIndex = 0
		psDecCtrl.LTPScaleQ14 = 0
	}
}
