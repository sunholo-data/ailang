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

// Transliteration of silk/PLC.c (libopus v1.6.1): SILK packet-loss concealment.
// On a good frame silk_PLC_update saves the prediction parameters into sPLC; on a
// lost frame silk_PLC_conceal extrapolates the missing signal by LTP synthesis from
// the stored pitch/LTP taps mixed with a scaled random excitation drawn from the last
// good excitation, followed by LPC synthesis, with the harmonic and random gains
// faded over consecutive losses. silk_PLC_glue_frames energy-matches the first good
// frame after a loss to the concealed energy. The deep-PLC (ENABLE_DEEP_PLC) and
// arch-specific branches are out of scope; this is the scalar FIXED_POINT reference.
// Names and control flow follow the C for diffability.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// Constants from silk/PLC.h.
const (
	bweCoef                 = 0.99  // BWE_COEF
	vPitchGainStartMinQ14   = 11469 // V_PITCH_GAIN_START_MIN_Q14 (0.7 in Q14)
	vPitchGainStartMaxQ14   = 15565 // V_PITCH_GAIN_START_MAX_Q14 (0.95 in Q14)
	maxPitchLagMS           = 18    // MAX_PITCH_LAG_MS
	randBufSize             = 128   // RAND_BUF_SIZE
	randBufMask             = randBufSize - 1
	log2InvLPCGainHighThres = 3   // LOG2_INV_LPC_GAIN_HIGH_THRES (2^3 = 8 dB LPC gain)
	log2InvLPCGainLowThres  = 8   // LOG2_INV_LPC_GAIN_LOW_THRES (2^8 = 24 dB LPC gain)
	pitchDriftFacQ16        = 655 // PITCH_DRIFT_FAC_Q16 (0.01 in Q16)

	nbAtt = 2 // NB_ATT

	// silk_int32_MAX >> 1, the cap on the inverse-gain scaling in silk_PLC_conceal.
	silkInt32MaxHalf = 0x7FFFFFFF >> 1
)

// Attenuation gains from silk/PLC.c, indexed by min(NB_ATT-1, lossCnt).
var (
	harmAttQ15           = [nbAtt]int16{32440, 31130} // 0.99, 0.95
	plcRandAttenuateVQ15 = [nbAtt]int16{31130, 26214} // 0.95, 0.8
	plcRandAttenuateUV15 = [nbAtt]int16{32440, 29491} // 0.99, 0.9
)

// PLCReset is silk/PLC.c silk_PLC_Reset: initialize the concealment sub-state at
// decoder reset (and lazily on a sampling-rate change). The pitch lag defaults to
// half the frame length and the previous gains to unity.
func PLCReset(psDec *DecoderState) {
	psDec.SPLC.PitchLQ8 = silkmath.Silk_LSHIFT(int32(psDec.FrameLength), 8-1)
	psDec.SPLC.PrevGainQ16[0] = silkFixConst(1, 16)
	psDec.SPLC.PrevGainQ16[1] = silkFixConst(1, 16)
	psDec.SPLC.SubfrLength = 20
	psDec.SPLC.NbSubfr = 2
}

// PLC is silk/PLC.c silk_PLC: the concealment control function. On a rate change it
// re-runs PLCReset. When lost != 0 it conceals the frame (extrapolation) and bumps
// lossCnt; otherwise it updates the saved parameters for a possible future loss.
func PLC(psDec *DecoderState, psDecCtrl *DecoderControl, frame []int16, lost int) {
	/* PLC control function */
	if psDec.FsKHz != psDec.SPLC.FsKHz {
		PLCReset(psDec)
		psDec.SPLC.FsKHz = psDec.FsKHz
	}

	if lost != 0 {
		/****************************/
		/* Generate Signal          */
		/****************************/
		plcConceal(psDec, psDecCtrl, frame)

		psDec.LossCnt++
	} else {
		/****************************/
		/* Update state             */
		/****************************/
		plcUpdate(psDec, psDecCtrl)
	}
}

// plcUpdate is silk/PLC.c silk_PLC_update: save the last good frame's LTP taps, LPC
// coefficients, pitch lag, LTP scale and last two gains into sPLC so a subsequent
// loss can extrapolate from them. For voiced frames it picks the strongest LTP tap
// set from the subframes covering the last pitch period and clamps its gain.
func plcUpdate(psDec *DecoderState, psDecCtrl *DecoderControl) {
	var ltpGainQ14, tempLTPGainQ14 int32
	var i, j int
	psPLC := &psDec.SPLC

	/* Update parameters used in case of packet loss */
	psDec.PrevSignalType = int(psDec.Indices.SignalType)
	ltpGainQ14 = 0
	if int(psDec.Indices.SignalType) == typeVoiced {
		/* Find the parameters for the last subframe which contains a pitch pulse */
		for j = 0; j*psDec.SubfrLength < psDecCtrl.PitchL[psDec.NbSubfr-1]; j++ {
			if j == psDec.NbSubfr {
				break
			}
			tempLTPGainQ14 = 0
			for i = 0; i < ltpOrder; i++ {
				tempLTPGainQ14 += int32(psDecCtrl.LTPCoefQ14[(psDec.NbSubfr-1-j)*ltpOrder+i])
			}
			if tempLTPGainQ14 > ltpGainQ14 {
				ltpGainQ14 = tempLTPGainQ14
				off := int(silkmath.Silk_SMULBB(int32(psDec.NbSubfr-1-j), ltpOrder))
				copy(psPLC.LTPCoefQ14[:], psDecCtrl.LTPCoefQ14[off:off+ltpOrder])

				psPLC.PitchLQ8 = silkmath.Silk_LSHIFT(int32(psDecCtrl.PitchL[psDec.NbSubfr-1-j]), 8)
			}
		}

		for i = 0; i < ltpOrder; i++ {
			psPLC.LTPCoefQ14[i] = 0
		}
		psPLC.LTPCoefQ14[ltpOrder/2] = int16(ltpGainQ14)

		/* Limit LT coefs */
		if ltpGainQ14 < vPitchGainStartMinQ14 {
			var scaleQ10 int32
			var tmp int32

			tmp = silkmath.Silk_LSHIFT(vPitchGainStartMinQ14, 10)
			scaleQ10 = silkmath.Silk_DIV32(tmp, silkmath.Silk_max_32(ltpGainQ14, 1))
			for i = 0; i < ltpOrder; i++ {
				psPLC.LTPCoefQ14[i] = int16(silkmath.Silk_RSHIFT(silkmath.Silk_SMULBB(int32(psPLC.LTPCoefQ14[i]), scaleQ10), 10))
			}
		} else if ltpGainQ14 > vPitchGainStartMaxQ14 {
			var scaleQ14 int32
			var tmp int32

			tmp = silkmath.Silk_LSHIFT(vPitchGainStartMaxQ14, 14)
			scaleQ14 = silkmath.Silk_DIV32(tmp, silkmath.Silk_max_32(ltpGainQ14, 1))
			for i = 0; i < ltpOrder; i++ {
				psPLC.LTPCoefQ14[i] = int16(silkmath.Silk_RSHIFT(silkmath.Silk_SMULBB(int32(psPLC.LTPCoefQ14[i]), scaleQ14), 14))
			}
		}
	} else {
		psPLC.PitchLQ8 = silkmath.Silk_LSHIFT(silkmath.Silk_SMULBB(int32(psDec.FsKHz), 18), 8)
		for i = 0; i < ltpOrder; i++ {
			psPLC.LTPCoefQ14[i] = 0
		}
	}

	/* Save LPC coefficients */
	copy(psPLC.PrevLPCQ12[:psDec.LPCOrder], psDecCtrl.PredCoefQ12[1][:psDec.LPCOrder])
	psPLC.PrevLTPScaleQ14 = int16(psDecCtrl.LTPScaleQ14)

	/* Save last two gains */
	psPLC.PrevGainQ16[0] = psDecCtrl.GainsQ16[psDec.NbSubfr-2]
	psPLC.PrevGainQ16[1] = psDecCtrl.GainsQ16[psDec.NbSubfr-1]

	psPLC.SubfrLength = psDec.SubfrLength
	psPLC.NbSubfr = psDec.NbSubfr
}

// plcEnergy is silk/PLC.c silk_PLC_energy: scale the last two subframes of the good
// excitation by the previous gains, saturate to int16 and return each subframe's
// shifted sum-of-squares, used to pick the lower-energy subframe as the random-noise
// source for concealment.
func plcEnergy(excQ14 []int32, prevGainQ10 [2]int32, subfrLength, nbSubfr int) (energy1 int32, shift1 int, energy2 int32, shift2 int) {
	excBuf := make([]int16, 2*subfrLength)
	/* Find random noise component */
	/* Scale previous excitation signal */
	for k := 0; k < 2; k++ {
		for i := 0; i < subfrLength; i++ {
			excBuf[k*subfrLength+i] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT(
				silkmath.Silk_SMULWW(excQ14[i+(k+nbSubfr-2)*subfrLength], prevGainQ10[k]), 8)))
		}
	}
	/* Find the subframe with lowest energy of the last two and use that as random noise generator */
	energy1, shift1 = silkSumSqrShift(excBuf[:subfrLength], subfrLength)
	energy2, shift2 = silkSumSqrShift(excBuf[subfrLength:], subfrLength)
	return energy1, shift1, energy2, shift2
}

// plcConceal is silk/PLC.c silk_PLC_conceal: synthesize a replacement for a lost
// frame. It rewhitens the LTP state from the output buffer, runs LTP synthesis with
// the saved taps mixed with a faded random excitation, then LPC synthesis with the
// bandwidth-expanded previous LPC, scaling to the previous gain. The harmonic gain,
// random gain, LTP taps, pitch lag, random seed and excitation scale all decay over
// consecutive losses.
func plcConceal(psDec *DecoderState, psDecCtrl *DecoderControl, frame []int16) {
	var i, j, k int
	var lag, idx, sLTPBufIdx, shift1, shift2 int
	var randSeed, harmGainQ15, randGainQ15, invGainQ30 int32
	var energy1, energy2 int32
	var lpcPredQ10, ltpPredQ12 int32
	var randScaleQ14 int16
	var aQ12 [maxLPCOrder]int16
	psPLC := &psDec.SPLC
	var prevGainQ10 [2]int32

	sLTPQ14 := make([]int32, psDec.LtpMemLength+psDec.FrameLength)
	sLTP := make([]int16, psDec.LtpMemLength)

	prevGainQ10[0] = silkmath.Silk_RSHIFT(psPLC.PrevGainQ16[0], 6)
	prevGainQ10[1] = silkmath.Silk_RSHIFT(psPLC.PrevGainQ16[1], 6)

	if psDec.FirstFrameAfterReset != 0 {
		for i = range psPLC.PrevLPCQ12 {
			psPLC.PrevLPCQ12[i] = 0
		}
	}

	energy1, shift1, energy2, shift2 = plcEnergy(psDec.ExcQ14[:], prevGainQ10, psDec.SubfrLength, psDec.NbSubfr)

	var randPtr []int32
	if silkmath.Silk_RSHIFT(energy1, shift2) < silkmath.Silk_RSHIFT(energy2, shift1) {
		/* First sub-frame has lowest energy */
		randPtr = psDec.ExcQ14[silkmath.Silk_max_int(0, (psPLC.NbSubfr-1)*psPLC.SubfrLength-randBufSize):]
	} else {
		/* Second sub-frame has lowest energy */
		randPtr = psDec.ExcQ14[silkmath.Silk_max_int(0, psPLC.NbSubfr*psPLC.SubfrLength-randBufSize):]
	}

	/* Set up Gain to random noise component */
	bQ14 := psPLC.LTPCoefQ14[:]
	randScaleQ14 = psPLC.RandScaleQ14

	/* Set up attenuation gains */
	harmGainQ15 = int32(harmAttQ15[silkmath.Silk_min_int(nbAtt-1, psDec.LossCnt)])
	if psDec.PrevSignalType == typeVoiced {
		randGainQ15 = int32(plcRandAttenuateVQ15[silkmath.Silk_min_int(nbAtt-1, psDec.LossCnt)])
	} else {
		randGainQ15 = int32(plcRandAttenuateUV15[silkmath.Silk_min_int(nbAtt-1, psDec.LossCnt)])
	}

	/* LPC concealment. Apply BWE to previous LPC */
	Bwexpander(psPLC.PrevLPCQ12[:], psDec.LPCOrder, silkFixConst(bweCoef, 16))

	/* Preload LPC coefficients to array on stack. Gives small performance gain */
	copy(aQ12[:psDec.LPCOrder], psPLC.PrevLPCQ12[:psDec.LPCOrder])

	/* First Lost frame */
	if psDec.LossCnt == 0 {
		randScaleQ14 = 1 << 14

		/* Reduce random noise Gain for voiced frames */
		if psDec.PrevSignalType == typeVoiced {
			for i = 0; i < ltpOrder; i++ {
				randScaleQ14 -= bQ14[i]
			}
			randScaleQ14 = silkmath.Silk_max_16(3277, randScaleQ14) /* 0.2 */
			randScaleQ14 = int16(silkmath.Silk_RSHIFT(silkmath.Silk_SMULBB(int32(randScaleQ14), int32(psPLC.PrevLTPScaleQ14)), 14))
		} else {
			/* Reduce random noise for unvoiced frames with high LPC gain */
			var lpcInvGainQ30, downScaleQ30 int32

			lpcInvGainQ30 = LPCInversePredGain(psPLC.PrevLPCQ12[:], psDec.LPCOrder)

			downScaleQ30 = silkmath.Silk_min_32(silkmath.Silk_RSHIFT(int32(1)<<30, log2InvLPCGainHighThres), lpcInvGainQ30)
			downScaleQ30 = silkmath.Silk_max_32(silkmath.Silk_RSHIFT(int32(1)<<30, log2InvLPCGainLowThres), downScaleQ30)
			downScaleQ30 = silkmath.Silk_LSHIFT(downScaleQ30, log2InvLPCGainHighThres)

			randGainQ15 = silkmath.Silk_RSHIFT(silkmath.Silk_SMULWB(downScaleQ30, randGainQ15), 14)
		}
	}

	randSeed = psPLC.RandSeed
	lag = int(silkmath.Silk_RSHIFT_ROUND(psPLC.PitchLQ8, 8))
	sLTPBufIdx = psDec.LtpMemLength

	/* Rewhiten LTP state */
	idx = psDec.LtpMemLength - lag - psDec.LPCOrder - ltpOrder/2
	/* celt_assert( idx > 0 ); */
	silkLPCAnalysisFilter(sLTP[idx:], psDec.OutBuf[idx:], aQ12[:], psDec.LtpMemLength-idx, psDec.LPCOrder)
	/* Scale LTP state */
	invGainQ30 = silkmath.Silk_INVERSE32_varQ(psPLC.PrevGainQ16[1], 46)
	invGainQ30 = silkmath.Silk_min_32(invGainQ30, silkInt32MaxHalf)
	for i = idx + psDec.LPCOrder; i < psDec.LtpMemLength; i++ {
		sLTPQ14[i] = silkmath.Silk_SMULWB(invGainQ30, int32(sLTP[i]))
	}

	/***************************/
	/* LTP synthesis filtering */
	/***************************/
	for k = 0; k < psDec.NbSubfr; k++ {
		/* Set up pointer */
		predLagIdx := sLTPBufIdx - lag + ltpOrder/2
		for i = 0; i < psDec.SubfrLength; i++ {
			/* Unrolled loop */
			/* Avoids introducing a bias because silk_SMLAWB() always rounds to -inf */
			ltpPredQ12 = 2
			ltpPredQ12 = silkmath.Silk_SMLAWB(ltpPredQ12, sLTPQ14[predLagIdx+0], int32(bQ14[0]))
			ltpPredQ12 = silkmath.Silk_SMLAWB(ltpPredQ12, sLTPQ14[predLagIdx-1], int32(bQ14[1]))
			ltpPredQ12 = silkmath.Silk_SMLAWB(ltpPredQ12, sLTPQ14[predLagIdx-2], int32(bQ14[2]))
			ltpPredQ12 = silkmath.Silk_SMLAWB(ltpPredQ12, sLTPQ14[predLagIdx-3], int32(bQ14[3]))
			ltpPredQ12 = silkmath.Silk_SMLAWB(ltpPredQ12, sLTPQ14[predLagIdx-4], int32(bQ14[4]))
			predLagIdx++

			/* Generate LPC excitation */
			randSeed = silkmath.Silk_RAND(randSeed)
			idx = int(silkmath.Silk_RSHIFT(randSeed, 25)) & randBufMask
			sLTPQ14[sLTPBufIdx] = silkmath.Silk_LSHIFT32(silkmath.Silk_SMLAWB(ltpPredQ12, randPtr[idx], int32(randScaleQ14)), 2)
			sLTPBufIdx++
		}

		/* Gradually reduce LTP gain */
		for j = 0; j < ltpOrder; j++ {
			bQ14[j] = int16(silkmath.Silk_RSHIFT(silkmath.Silk_SMULBB(harmGainQ15, int32(bQ14[j])), 15))
		}
		/* Gradually reduce excitation gain */
		randScaleQ14 = int16(silkmath.Silk_RSHIFT(silkmath.Silk_SMULBB(int32(randScaleQ14), randGainQ15), 15))

		/* Slowly increase pitch lag */
		psPLC.PitchLQ8 = silkmath.Silk_SMLAWB(psPLC.PitchLQ8, psPLC.PitchLQ8, pitchDriftFacQ16)
		psPLC.PitchLQ8 = silkmath.Silk_min_32(psPLC.PitchLQ8, silkmath.Silk_LSHIFT(silkmath.Silk_SMULBB(maxPitchLagMS, int32(psDec.FsKHz)), 8))
		lag = int(silkmath.Silk_RSHIFT_ROUND(psPLC.PitchLQ8, 8))
	}

	/***************************/
	/* LPC synthesis filtering */
	/***************************/
	sLPCQ14Ptr := sLTPQ14[psDec.LtpMemLength-maxLPCOrder:]

	/* Copy LPC state */
	copy(sLPCQ14Ptr[:maxLPCOrder], psDec.SLPCQ14Buf[:])

	/* celt_assert( psDec->LPC_order >= 10 ); */ /* check that unrolling works */
	for i = 0; i < psDec.FrameLength; i++ {
		/* partly unrolled */
		/* Avoids introducing a bias because silk_SMLAWB() always rounds to -inf */
		lpcPredQ10 = silkmath.Silk_RSHIFT(int32(psDec.LPCOrder), 1)
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-1], int32(aQ12[0]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-2], int32(aQ12[1]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-3], int32(aQ12[2]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-4], int32(aQ12[3]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-5], int32(aQ12[4]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-6], int32(aQ12[5]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-7], int32(aQ12[6]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-8], int32(aQ12[7]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-9], int32(aQ12[8]))
		lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-10], int32(aQ12[9]))
		for j = 10; j < psDec.LPCOrder; j++ {
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, sLPCQ14Ptr[maxLPCOrder+i-j-1], int32(aQ12[j]))
		}

		/* Add prediction to LPC excitation */
		sLPCQ14Ptr[maxLPCOrder+i] = silkmath.Silk_ADD_SAT32(sLPCQ14Ptr[maxLPCOrder+i],
			silkmath.Silk_LSHIFT_SAT32(lpcPredQ10, 4))

		/* Scale with Gain */
		frame[i] = int16(silkmath.Silk_SAT16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_SMULWW(sLPCQ14Ptr[maxLPCOrder+i], prevGainQ10[1]), 8))))
	}

	/* Save LPC state */
	copy(psDec.SLPCQ14Buf[:], sLPCQ14Ptr[psDec.FrameLength:psDec.FrameLength+maxLPCOrder])

	/**************************************/
	/* Update states                      */
	/**************************************/
	psPLC.RandSeed = randSeed
	psPLC.RandScaleQ14 = randScaleQ14
	for i = 0; i < maxNBSubfr; i++ {
		psDecCtrl.PitchL[i] = lag
	}
}

// PLCGlueFrames is silk/PLC.c silk_PLC_glue_frames: smooth the transition from
// concealed to good frames. During loss it records the concealed residual energy;
// on the first good frame after a loss it fades the good frame in from that energy
// so a recovered onset does not jump in level.
func PLCGlueFrames(psDec *DecoderState, frame []int16, length int) {
	var i, energyShift int
	var energy int32
	psPLC := &psDec.SPLC

	if psDec.LossCnt != 0 {
		/* Calculate energy in concealed residual */
		psPLC.ConcEnergy, psPLC.ConcEnergyShift = silkSumSqrShift(frame[:length], length)

		psPLC.LastFrameLost = 1
	} else {
		if psPLC.LastFrameLost != 0 {
			/* Calculate residual in decoded signal if last frame was lost */
			energy, energyShift = silkSumSqrShift(frame[:length], length)

			/* Normalize energies */
			if energyShift > psPLC.ConcEnergyShift {
				psPLC.ConcEnergy = silkmath.Silk_RSHIFT(psPLC.ConcEnergy, energyShift-psPLC.ConcEnergyShift)
			} else if energyShift < psPLC.ConcEnergyShift {
				energy = silkmath.Silk_RSHIFT(energy, psPLC.ConcEnergyShift-energyShift)
			}

			/* Fade in the energy difference */
			if energy > psPLC.ConcEnergy {
				var lz, gainQ16, slopeQ16 int32

				lz = silkmath.Silk_CLZ32(psPLC.ConcEnergy)
				lz = lz - 1
				psPLC.ConcEnergy = silkmath.Silk_LSHIFT(psPLC.ConcEnergy, int(lz))
				energy = silkmath.Silk_RSHIFT(energy, int(silkmath.Silk_max_32(24-lz, 0)))

				fracQ24 := silkmath.Silk_DIV32(psPLC.ConcEnergy, silkmath.Silk_max_32(energy, 1))

				gainQ16 = silkmath.Silk_LSHIFT(silkmath.Silk_SQRT_APPROX(fracQ24), 4)
				slopeQ16 = silkmath.Silk_DIV32_16((int32(1)<<16)-gainQ16, int32(length))
				/* Make slope 4x steeper to avoid missing onsets after DTX */
				slopeQ16 = silkmath.Silk_LSHIFT(slopeQ16, 2)

				for i = 0; i < length; i++ {
					frame[i] = int16(silkmath.Silk_SMULWB(gainQ16, int32(frame[i])))
					gainQ16 += slopeQ16
					if gainQ16 > int32(1)<<16 {
						break
					}
				}
			}
		}
		psPLC.LastFrameLost = 0
	}
}
