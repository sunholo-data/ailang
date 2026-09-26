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

// Transliteration of silk/CNG.c (libopus v1.6.1): comfort-noise generation. On good
// inactive frames (lossCnt == 0 and the previous frame was TYPE_NO_VOICE_ACTIVITY)
// silk_CNG smooths the background LSFs, gain and excitation buffer into sCNG; on lost
// frames it synthesizes comfort noise from that smoothed background (random
// excitation drawn from CNG_exc_buf, LPC synthesis with the smoothed NLSFs) and adds
// it to the concealed signal. Names and control flow follow the C for diffability.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// Constants from silk/define.h.
const (
	cngBufMaskMax           = 255   // CNG_BUF_MASK_MAX (2^floor(log2(MAX_FRAME_LENGTH))-1)
	cngGainSmthQ16          = 4634  // CNG_GAIN_SMTH_Q16 (0.25^(1/4))
	cngGainSmthThresholdQ16 = 46396 // CNG_GAIN_SMTH_THRESHOLD_Q16 (-3 dB)
	cngNLSFSmthQ16          = 16348 // CNG_NLSF_SMTH_Q16 (0.25)

	silkInt16Max = 32767 // silk_int16_MAX
)

// cngExc is silk/CNG.c silk_CNG_exc: generate the CNG excitation by drawing random
// samples (masked pseudo-random index) from the smoothed excitation buffer.
func cngExc(excQ14, excBufQ14 []int32, length int, randSeed *int32) {
	var seed int32
	var i, idx, excMask int

	excMask = cngBufMaskMax
	for excMask > length {
		excMask = int(silkmath.Silk_RSHIFT(int32(excMask), 1))
	}

	seed = *randSeed
	for i = 0; i < length; i++ {
		seed = silkmath.Silk_RAND(seed)
		idx = int(silkmath.Silk_RSHIFT(seed, 24)) & excMask
		/* silk_assert( idx >= 0 ); silk_assert( idx <= CNG_BUF_MASK_MAX ); */
		excQ14[i] = excBufQ14[idx]
	}
	*randSeed = seed
}

// CNGReset is silk/CNG.c silk_CNG_Reset: initialize the comfort-noise sub-state, an
// evenly spaced NLSF ramp and a fixed random seed. Run at decoder reset and lazily
// on a sampling-rate change.
func CNGReset(psDec *DecoderState) {
	var nlsfStepQ15, nlsfAccQ15 int

	nlsfStepQ15 = int(silkmath.Silk_DIV32_16(silkInt16Max, int32(psDec.LPCOrder+1)))
	nlsfAccQ15 = 0
	for i := 0; i < psDec.LPCOrder; i++ {
		nlsfAccQ15 += nlsfStepQ15
		psDec.SCNG.CNGSmthNLSFQ15[i] = int16(nlsfAccQ15)
	}
	psDec.SCNG.CNGSmthGainQ16 = 0
	psDec.SCNG.RandSeed = 3176576
}

// CNG is silk/CNG.c silk_CNG: update the comfort-noise estimate on good inactive
// frames, and on a lost frame synthesize comfort noise and add it to the (concealed)
// signal in frame[0:length].
func CNG(psDec *DecoderState, psDecCtrl *DecoderControl, frame []int16, length int) {
	var i, subfr int
	var lpcPredQ10, maxGainQ16, gainQ16, gainQ10 int32
	var aQ12 [maxLPCOrder]int16
	psCNG := &psDec.SCNG

	if psDec.FsKHz != psCNG.FsKHz {
		/* Reset state */
		CNGReset(psDec)

		psCNG.FsKHz = psDec.FsKHz
	}
	if psDec.LossCnt == 0 && psDec.PrevSignalType == typeNoVoiceActivity {
		/* Update CNG parameters */

		/* Smoothing of LSF's  */
		for i = 0; i < psDec.LPCOrder; i++ {
			psCNG.CNGSmthNLSFQ15[i] = int16(int32(psCNG.CNGSmthNLSFQ15[i]) +
				silkmath.Silk_SMULWB(int32(psDec.PrevNLSFQ15[i])-int32(psCNG.CNGSmthNLSFQ15[i]), cngNLSFSmthQ16))
		}
		/* Find the subframe with the highest gain */
		maxGainQ16 = 0
		subfr = 0
		for i = 0; i < psDec.NbSubfr; i++ {
			if psDecCtrl.GainsQ16[i] > maxGainQ16 {
				maxGainQ16 = psDecCtrl.GainsQ16[i]
				subfr = i
			}
		}
		/* Update CNG excitation buffer with excitation from this subframe */
		copy(psCNG.CNGExcBufQ14[psDec.SubfrLength:psDec.SubfrLength+(psDec.NbSubfr-1)*psDec.SubfrLength],
			psCNG.CNGExcBufQ14[:(psDec.NbSubfr-1)*psDec.SubfrLength])
		copy(psCNG.CNGExcBufQ14[:psDec.SubfrLength],
			psDec.ExcQ14[subfr*psDec.SubfrLength:subfr*psDec.SubfrLength+psDec.SubfrLength])

		/* Smooth gains */
		for i = 0; i < psDec.NbSubfr; i++ {
			psCNG.CNGSmthGainQ16 += silkmath.Silk_SMULWB(psDecCtrl.GainsQ16[i]-psCNG.CNGSmthGainQ16, cngGainSmthQ16)
			/* If the smoothed gain is 3 dB greater than this subframe's gain, use this subframe's gain to adapt faster. */
			if silkmath.Silk_SMULWW(psCNG.CNGSmthGainQ16, cngGainSmthThresholdQ16) > psDecCtrl.GainsQ16[i] {
				psCNG.CNGSmthGainQ16 = psDecCtrl.GainsQ16[i]
			}
		}
	}

	/* Add CNG when packet is lost or during DTX */
	if psDec.LossCnt != 0 {
		cngSigQ14 := make([]int32, length+maxLPCOrder)

		/* Generate CNG excitation */
		gainQ16 = silkmath.Silk_SMULWW(int32(psDec.SPLC.RandScaleQ14), psDec.SPLC.PrevGainQ16[1])
		if gainQ16 >= (1<<21) || psCNG.CNGSmthGainQ16 > (1<<23) {
			gainQ16 = silkmath.Silk_SMULTT(gainQ16, gainQ16)
			gainQ16 = silkmath.Silk_SUB_LSHIFT32(silkmath.Silk_SMULTT(psCNG.CNGSmthGainQ16, psCNG.CNGSmthGainQ16), gainQ16, 5)
			gainQ16 = silkmath.Silk_LSHIFT32(silkmath.Silk_SQRT_APPROX(gainQ16), 16)
		} else {
			gainQ16 = silkmath.Silk_SMULWW(gainQ16, gainQ16)
			gainQ16 = silkmath.Silk_SUB_LSHIFT32(silkmath.Silk_SMULWW(psCNG.CNGSmthGainQ16, psCNG.CNGSmthGainQ16), gainQ16, 5)
			gainQ16 = silkmath.Silk_LSHIFT32(silkmath.Silk_SQRT_APPROX(gainQ16), 8)
		}
		gainQ10 = silkmath.Silk_RSHIFT(gainQ16, 6)

		cngExc(cngSigQ14[maxLPCOrder:], psCNG.CNGExcBufQ14[:], length, &psCNG.RandSeed)

		/* Convert CNG NLSF to filter representation */
		NLSF2A(aQ12[:], psCNG.CNGSmthNLSFQ15[:], psDec.LPCOrder)

		/* Generate CNG signal, by synthesis filtering */
		copy(cngSigQ14[:maxLPCOrder], psCNG.CNGSynthState[:])
		/* celt_assert( psDec->LPC_order == 10 || psDec->LPC_order == 16 ); */
		for i = 0; i < length; i++ {
			/* Avoids introducing a bias because silk_SMLAWB() always rounds to -inf */
			lpcPredQ10 = silkmath.Silk_RSHIFT(int32(psDec.LPCOrder), 1)
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-1], int32(aQ12[0]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-2], int32(aQ12[1]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-3], int32(aQ12[2]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-4], int32(aQ12[3]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-5], int32(aQ12[4]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-6], int32(aQ12[5]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-7], int32(aQ12[6]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-8], int32(aQ12[7]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-9], int32(aQ12[8]))
			lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-10], int32(aQ12[9]))
			if psDec.LPCOrder == 16 {
				lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-11], int32(aQ12[10]))
				lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-12], int32(aQ12[11]))
				lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-13], int32(aQ12[12]))
				lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-14], int32(aQ12[13]))
				lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-15], int32(aQ12[14]))
				lpcPredQ10 = silkmath.Silk_SMLAWB(lpcPredQ10, cngSigQ14[maxLPCOrder+i-16], int32(aQ12[15]))
			}

			/* Update states */
			cngSigQ14[maxLPCOrder+i] = silkmath.Silk_ADD_SAT32(cngSigQ14[maxLPCOrder+i], silkmath.Silk_LSHIFT_SAT32(lpcPredQ10, 4))

			/* Scale with Gain and add to input signal */
			frame[i] = silkmath.Silk_ADD_SAT16(frame[i], int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_SMULWW(cngSigQ14[maxLPCOrder+i], gainQ10), 8))))
		}
		copy(psCNG.CNGSynthState[:], cngSigQ14[length:length+maxLPCOrder])
	} else {
		for i = 0; i < psDec.LPCOrder; i++ {
			psCNG.CNGSynthState[i] = 0
		}
	}
}
