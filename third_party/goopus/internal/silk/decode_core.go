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

// Transliteration of silk/decode_core.c (libopus v1.6.1): the SILK core decoder,
// i.e. the inverse NSQ operation (LTP + LPC synthesis). It reconstructs the
// excitation from the decoded pulses scaled by Gains_Q16 (with the quantization
// offset and the seed-driven sign randomization), runs long-term prediction for
// voiced subframes over the re-whitened LTP history, then the short-term LPC
// synthesis filter, and scales by the linear gain to produce the internal-rate
// decoded speech xq. The fixed-point (saturation, the Q14/Q13/Q10 shifts) is
// preserved exactly. arch is dropped: the port is scalar-only. Names and control
// flow follow the C for diffability.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// b25Q14 is SILK_FIX_CONST( 0.25, 14 ), the default center LTP tap the voiced-PLC
// avoidance branch installs.
var b25Q14 = int16(silkFixConst(0.25, 14))

// DecodeCore is silk/decode_core.c silk_decode_core. psDec carries the persistent
// synthesis state (ExcQ14, SLPCQ14Buf, OutBuf, PrevGainQ16); psDecCtrl holds the
// per-frame prediction parameters; xq receives the decoded speech at the internal
// sample rate (length psDec.FrameLength); pulses is the decoded pulse signal.
func DecodeCore(psDec *DecoderState, psDecCtrl *DecoderControl, xq []int16, pulses []int16) {
	var i, k, lag int
	var startIdx, sLTPBufIdx, NLSFInterpolationFlag, signalType int
	var A_Q12Tmp [maxLPCOrder]int16
	var LTPPredQ13, LPCPredQ10, GainQ10, invGainQ31, gainAdjQ16, randSeed, offsetQ10 int32

	/* silk_assert( psDec->prev_gain_Q16 != 0 ); */

	sLTP := make([]int16, psDec.LtpMemLength)
	sLTPQ15 := make([]int32, psDec.LtpMemLength+psDec.FrameLength)
	resQ14 := make([]int32, psDec.SubfrLength)
	sLPCQ14 := make([]int32, psDec.SubfrLength+maxLPCOrder)

	offsetQ10 = int32(silkQuantizationOffsetsQ10[psDec.Indices.SignalType>>1][psDec.Indices.QuantOffsetType])

	if psDec.Indices.NLSFInterpCoefQ2 < 1<<2 {
		NLSFInterpolationFlag = 1
	} else {
		NLSFInterpolationFlag = 0
	}

	/* Decode excitation */
	randSeed = int32(psDec.Indices.Seed)
	for i = 0; i < psDec.FrameLength; i++ {
		randSeed = silkmath.Silk_RAND(randSeed)
		psDec.ExcQ14[i] = silkmath.Silk_LSHIFT(int32(pulses[i]), 14)
		if psDec.ExcQ14[i] > 0 {
			psDec.ExcQ14[i] -= quantLevelAdjustQ10 << 4
		} else if psDec.ExcQ14[i] < 0 {
			psDec.ExcQ14[i] += quantLevelAdjustQ10 << 4
		}
		psDec.ExcQ14[i] += offsetQ10 << 4
		if randSeed < 0 {
			psDec.ExcQ14[i] = -psDec.ExcQ14[i]
		}

		randSeed = silkmath.Silk_ADD32_ovflw(randSeed, int32(pulses[i]))
	}

	/* Copy LPC state */
	copy(sLPCQ14[:maxLPCOrder], psDec.SLPCQ14Buf[:])

	pexcOff := 0 // pexc_Q14 = psDec->exc_Q14
	pxqOff := 0  // pxq      = xq
	sLTPBufIdx = psDec.LtpMemLength
	/* Loop over subframes */
	for k = 0; k < psDec.NbSubfr; k++ {
		var pres []int32
		A_Q12 := psDecCtrl.PredCoefQ12[k>>1][:]

		/* Preload LPC coefficients to array on stack. Gives small performance gain */
		copy(A_Q12Tmp[:psDec.LPCOrder], A_Q12[:psDec.LPCOrder])
		B_Q14 := psDecCtrl.LTPCoefQ14[k*ltpOrder:]
		signalType = int(psDec.Indices.SignalType)

		GainQ10 = silkmath.Silk_RSHIFT(psDecCtrl.GainsQ16[k], 6)
		invGainQ31 = silkmath.Silk_INVERSE32_varQ(psDecCtrl.GainsQ16[k], 47)

		/* Calculate gain adjustment factor */
		if psDecCtrl.GainsQ16[k] != psDec.PrevGainQ16 {
			gainAdjQ16 = silkmath.Silk_DIV32_varQ(psDec.PrevGainQ16, psDecCtrl.GainsQ16[k], 16)

			/* Scale short term state */
			for i = 0; i < maxLPCOrder; i++ {
				sLPCQ14[i] = silkmath.Silk_SMULWW(gainAdjQ16, sLPCQ14[i])
			}
		} else {
			gainAdjQ16 = int32(1) << 16
		}

		/* Save inv_gain */
		/* silk_assert( inv_gain_Q31 != 0 ); */
		psDec.PrevGainQ16 = psDecCtrl.GainsQ16[k]

		/* Avoid abrupt transition from voiced PLC to unvoiced normal decoding */
		if psDec.LossCnt != 0 && psDec.PrevSignalType == typeVoiced &&
			int(psDec.Indices.SignalType) != typeVoiced && k < maxNBSubfr/2 {

			for i = 0; i < ltpOrder; i++ {
				B_Q14[i] = 0
			}
			B_Q14[ltpOrder/2] = b25Q14

			signalType = typeVoiced
			psDecCtrl.PitchL[k] = psDec.LagPrev
		}

		if signalType == typeVoiced {
			/* Voiced */
			lag = psDecCtrl.PitchL[k]

			/* Re-whitening */
			if k == 0 || (k == 2 && NLSFInterpolationFlag != 0) {
				/* Rewhiten with new A coefs */
				startIdx = psDec.LtpMemLength - lag - psDec.LPCOrder - ltpOrder/2
				/* celt_assert( start_idx > 0 ); */

				if k == 2 {
					copy(psDec.OutBuf[psDec.LtpMemLength:psDec.LtpMemLength+2*psDec.SubfrLength], xq[:2*psDec.SubfrLength])
				}

				silkLPCAnalysisFilter(sLTP[startIdx:], psDec.OutBuf[startIdx+k*psDec.SubfrLength:],
					A_Q12, psDec.LtpMemLength-startIdx, psDec.LPCOrder)

				/* After rewhitening the LTP state is unscaled */
				if k == 0 {
					/* Do LTP downscaling to reduce inter-packet dependency */
					invGainQ31 = silkmath.Silk_LSHIFT(silkmath.Silk_SMULWB(invGainQ31, int32(psDecCtrl.LTPScaleQ14)), 2)
				}
				for i = 0; i < lag+ltpOrder/2; i++ {
					sLTPQ15[sLTPBufIdx-i-1] = silkmath.Silk_SMULWB(invGainQ31, int32(sLTP[psDec.LtpMemLength-i-1]))
				}
			} else {
				/* Update LTP state when Gain changes */
				if gainAdjQ16 != int32(1)<<16 {
					for i = 0; i < lag+ltpOrder/2; i++ {
						sLTPQ15[sLTPBufIdx-i-1] = silkmath.Silk_SMULWW(gainAdjQ16, sLTPQ15[sLTPBufIdx-i-1])
					}
				}
			}
		}

		/* Long-term prediction */
		if signalType == typeVoiced {
			/* Set up pointer */
			predLagPtr := sLTPBufIdx - lag + ltpOrder/2
			pres = resQ14
			for i = 0; i < psDec.SubfrLength; i++ {
				/* Unrolled loop */
				/* Avoids introducing a bias because silk_SMLAWB() always rounds to -inf */
				LTPPredQ13 = 2
				LTPPredQ13 = silkmath.Silk_SMLAWB(LTPPredQ13, sLTPQ15[predLagPtr+0], int32(B_Q14[0]))
				LTPPredQ13 = silkmath.Silk_SMLAWB(LTPPredQ13, sLTPQ15[predLagPtr-1], int32(B_Q14[1]))
				LTPPredQ13 = silkmath.Silk_SMLAWB(LTPPredQ13, sLTPQ15[predLagPtr-2], int32(B_Q14[2]))
				LTPPredQ13 = silkmath.Silk_SMLAWB(LTPPredQ13, sLTPQ15[predLagPtr-3], int32(B_Q14[3]))
				LTPPredQ13 = silkmath.Silk_SMLAWB(LTPPredQ13, sLTPQ15[predLagPtr-4], int32(B_Q14[4]))
				predLagPtr++

				/* Generate LPC excitation */
				pres[i] = silkmath.Silk_ADD_LSHIFT32(psDec.ExcQ14[pexcOff+i], LTPPredQ13, 1)

				/* Update states */
				sLTPQ15[sLTPBufIdx] = silkmath.Silk_LSHIFT(pres[i], 1)
				sLTPBufIdx++
			}
		} else {
			pres = psDec.ExcQ14[pexcOff : pexcOff+psDec.SubfrLength]
		}

		for i = 0; i < psDec.SubfrLength; i++ {
			/* Short-term prediction */
			/* celt_assert( psDec->LPC_order == 10 || psDec->LPC_order == 16 ); */
			/* Avoids introducing a bias because silk_SMLAWB() always rounds to -inf */
			LPCPredQ10 = silkmath.Silk_RSHIFT(int32(psDec.LPCOrder), 1)
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-1], int32(A_Q12Tmp[0]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-2], int32(A_Q12Tmp[1]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-3], int32(A_Q12Tmp[2]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-4], int32(A_Q12Tmp[3]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-5], int32(A_Q12Tmp[4]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-6], int32(A_Q12Tmp[5]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-7], int32(A_Q12Tmp[6]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-8], int32(A_Q12Tmp[7]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-9], int32(A_Q12Tmp[8]))
			LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-10], int32(A_Q12Tmp[9]))
			if psDec.LPCOrder == 16 {
				LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-11], int32(A_Q12Tmp[10]))
				LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-12], int32(A_Q12Tmp[11]))
				LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-13], int32(A_Q12Tmp[12]))
				LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-14], int32(A_Q12Tmp[13]))
				LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-15], int32(A_Q12Tmp[14]))
				LPCPredQ10 = silkmath.Silk_SMLAWB(LPCPredQ10, sLPCQ14[maxLPCOrder+i-16], int32(A_Q12Tmp[15]))
			}

			/* Add prediction to LPC excitation */
			sLPCQ14[maxLPCOrder+i] = silkmath.Silk_ADD_SAT32(pres[i], silkmath.Silk_LSHIFT_SAT32(LPCPredQ10, 4))

			/* Scale with gain */
			xq[pxqOff+i] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_SMULWW(sLPCQ14[maxLPCOrder+i], GainQ10), 8)))
		}

		/* Update LPC filter state */
		copy(sLPCQ14[:maxLPCOrder], sLPCQ14[psDec.SubfrLength:psDec.SubfrLength+maxLPCOrder])
		pexcOff += psDec.SubfrLength
		pxqOff += psDec.SubfrLength
	}

	/* Save LPC state */
	copy(psDec.SLPCQ14Buf[:], sLPCQ14[:maxLPCOrder])
}
