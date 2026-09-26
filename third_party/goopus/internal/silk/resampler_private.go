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

package silk

// Private SILK resampler filters, transliterated from
// silk/resampler_private_up2_HQ.c, silk/resampler_private_AR2.c,
// silk/resampler_private_IIR_FIR.c and silk/resampler_private_down_FIR.c
// (libopus v1.6.1). Names, control flow and comments follow the C; fixed-point
// operations route through internal/silkmath. The scratch "buf" arrays that C
// allocates on the stack (VARDECL/ALLOC) are plain Go slices here.

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// silkResamplerPrivateUp2HQ upsamples by a factor 2, high quality
// (silk/resampler_private_up2_HQ.c). Uses 2nd order allpass filters for the 2x
// upsampling, followed by a notch filter just above Nyquist. state is the 6-element
// SIIR memory; out receives 2*length samples.
func silkResamplerPrivateUp2HQ(state []int32, out []int16, in []int16, length int32) {
	var in32, out32Q1, out32Q2, Y, X int32

	/* Internal variables and state are in Q10 format */
	for k := 0; k < int(length); k++ {
		/* Convert to Q10 */
		in32 = silkmath.Silk_LSHIFT(int32(in[k]), 10)

		/* First all-pass section for even output sample */
		Y = silkmath.Silk_SUB32(in32, state[0])
		X = silkmath.Silk_SMULWB(Y, int32(silkResamplerUp2HQ0[0]))
		out32Q1 = silkmath.Silk_ADD32(state[0], X)
		state[0] = silkmath.Silk_ADD32(in32, X)

		/* Second all-pass section for even output sample */
		Y = silkmath.Silk_SUB32(out32Q1, state[1])
		X = silkmath.Silk_SMULWB(Y, int32(silkResamplerUp2HQ0[1]))
		out32Q2 = silkmath.Silk_ADD32(state[1], X)
		state[1] = silkmath.Silk_ADD32(out32Q1, X)

		/* Third all-pass section for even output sample */
		Y = silkmath.Silk_SUB32(out32Q2, state[2])
		X = silkmath.Silk_SMLAWB(Y, Y, int32(silkResamplerUp2HQ0[2]))
		out32Q1 = silkmath.Silk_ADD32(state[2], X)
		state[2] = silkmath.Silk_ADD32(out32Q2, X)

		/* Apply gain in Q15, convert back to int16 and store to output */
		out[2*k] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(out32Q1, 10)))

		/* First all-pass section for odd output sample */
		Y = silkmath.Silk_SUB32(in32, state[3])
		X = silkmath.Silk_SMULWB(Y, int32(silkResamplerUp2HQ1[0]))
		out32Q1 = silkmath.Silk_ADD32(state[3], X)
		state[3] = silkmath.Silk_ADD32(in32, X)

		/* Second all-pass section for odd output sample */
		Y = silkmath.Silk_SUB32(out32Q1, state[4])
		X = silkmath.Silk_SMULWB(Y, int32(silkResamplerUp2HQ1[1]))
		out32Q2 = silkmath.Silk_ADD32(state[4], X)
		state[4] = silkmath.Silk_ADD32(out32Q1, X)

		/* Third all-pass section for odd output sample */
		Y = silkmath.Silk_SUB32(out32Q2, state[5])
		X = silkmath.Silk_SMLAWB(Y, Y, int32(silkResamplerUp2HQ1[2]))
		out32Q1 = silkmath.Silk_ADD32(state[5], X)
		state[5] = silkmath.Silk_ADD32(out32Q2, X)

		/* Apply gain in Q15, convert back to int16 and store to output */
		out[2*k+1] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(out32Q1, 10)))
	}
}

// silkResamplerPrivateUp2HQWrapper wraps silkResamplerPrivateUp2HQ over the
// resampler state (silk/resampler_private_up2_HQ.c).
func silkResamplerPrivateUp2HQWrapper(S *ResamplerState, out []int16, in []int16, length int32) {
	silkResamplerPrivateUp2HQ(S.SIIR[:], out, in, length)
}

// silkResamplerPrivateAR2 is a second order AR filter with single delay elements
// (silk/resampler_private_AR2.c). state is the 2-element AR memory; outQ8 receives
// the Q8 output; AQ14 holds the AR coefficients in Q14.
func silkResamplerPrivateAR2(state []int32, outQ8 []int32, in []int16, AQ14 []int16, length int32) {
	var out32 int32

	for k := 0; k < int(length); k++ {
		out32 = silkmath.Silk_ADD_LSHIFT32(state[0], int32(in[k]), 8)
		outQ8[k] = out32
		out32 = silkmath.Silk_LSHIFT(out32, 2)
		state[0] = silkmath.Silk_SMLAWB(state[1], out32, int32(AQ14[0]))
		state[1] = silkmath.Silk_SMULWB(out32, int32(AQ14[1]))
	}
}

// silkResamplerPrivateIIRFIRInterpol interpolates the upsampled signal and stores it
// in the output array (silk/resampler_private_IIR_FIR.c). It writes into out starting
// at outIdx and returns the advanced index.
func silkResamplerPrivateIIRFIRInterpol(out []int16, outIdx int, buf []int16, maxIndexQ16, indexIncrementQ16 int32) int {
	var resQ15 int32

	/* Interpolate upsampled signal and store in output array */
	for indexQ16 := int32(0); indexQ16 < maxIndexQ16; indexQ16 += indexIncrementQ16 {
		tableIndex := silkmath.Silk_SMULWB(indexQ16&0xFFFF, 12)
		bufPtr := buf[indexQ16>>16:]

		resQ15 = silkmath.Silk_SMULBB(int32(bufPtr[0]), int32(silkResamplerFracFIR12[tableIndex][0]))
		resQ15 = silkmath.Silk_SMLABB(resQ15, int32(bufPtr[1]), int32(silkResamplerFracFIR12[tableIndex][1]))
		resQ15 = silkmath.Silk_SMLABB(resQ15, int32(bufPtr[2]), int32(silkResamplerFracFIR12[tableIndex][2]))
		resQ15 = silkmath.Silk_SMLABB(resQ15, int32(bufPtr[3]), int32(silkResamplerFracFIR12[tableIndex][3]))
		resQ15 = silkmath.Silk_SMLABB(resQ15, int32(bufPtr[4]), int32(silkResamplerFracFIR12[11-tableIndex][3]))
		resQ15 = silkmath.Silk_SMLABB(resQ15, int32(bufPtr[5]), int32(silkResamplerFracFIR12[11-tableIndex][2]))
		resQ15 = silkmath.Silk_SMLABB(resQ15, int32(bufPtr[6]), int32(silkResamplerFracFIR12[11-tableIndex][1]))
		resQ15 = silkmath.Silk_SMLABB(resQ15, int32(bufPtr[7]), int32(silkResamplerFracFIR12[11-tableIndex][0]))
		out[outIdx] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(resQ15, 15)))
		outIdx++
	}
	return outIdx
}

// silkResamplerPrivateIIRFIR upsamples using a combination of allpass-based 2x
// upsampling and FIR interpolation (silk/resampler_private_IIR_FIR.c).
func silkResamplerPrivateIIRFIR(S *ResamplerState, out []int16, in []int16, inLen int32) {
	var nSamplesIn int32

	buf := make([]int16, 2*S.BatchSize+resamplerOrderFIR12)

	/* Copy buffered samples to start of buffer */
	for i := 0; i < resamplerOrderFIR12; i++ {
		buf[i] = int16(S.SFIR[i])
	}

	/* Iterate over blocks of frameSizeIn input samples */
	indexIncrementQ16 := S.InvRatioQ16
	outIdx := 0
	for {
		nSamplesIn = silkmath.Silk_min_32(inLen, int32(S.BatchSize))

		/* Upsample 2x */
		silkResamplerPrivateUp2HQ(S.SIIR[:], buf[resamplerOrderFIR12:], in, nSamplesIn)

		maxIndexQ16 := silkmath.Silk_LSHIFT32(nSamplesIn, 16+1) /* + 1 because 2x upsampling */
		outIdx = silkResamplerPrivateIIRFIRInterpol(out, outIdx, buf, maxIndexQ16, indexIncrementQ16)
		in = in[nSamplesIn:]
		inLen -= nSamplesIn

		if inLen > 0 {
			/* More iterations to do; copy last part of filtered signal to beginning of buffer */
			n2 := int(nSamplesIn) << 1
			copy(buf[:resamplerOrderFIR12], buf[n2:n2+resamplerOrderFIR12])
		} else {
			break
		}
	}

	/* Copy last part of filtered signal to the state for the next call */
	n2 := int(nSamplesIn) << 1
	for i := 0; i < resamplerOrderFIR12; i++ {
		S.SFIR[i] = int32(buf[n2+i])
	}
}

// silkResamplerPrivateDownFIRInterpol interpolates the AR2-filtered signal and stores
// it in the output array (silk/resampler_private_down_FIR.c). firCoefs is the FIR part
// of the COEFS table (i.e. Coefs[2:]); it writes into out from outIdx and returns the
// advanced index.
func silkResamplerPrivateDownFIRInterpol(out []int16, outIdx int, buf []int32, firCoefs []int16, firOrder, firFracs int, maxIndexQ16, indexIncrementQ16 int32) int {
	var resQ6 int32

	switch firOrder {
	case resamplerDownOrderFIR0:
		for indexQ16 := int32(0); indexQ16 < maxIndexQ16; indexQ16 += indexIncrementQ16 {
			/* Integer part gives pointer to buffered input */
			bufPtr := buf[silkmath.Silk_RSHIFT(indexQ16, 16):]

			/* Fractional part gives interpolation coefficients */
			interpolInd := int(silkmath.Silk_SMULWB(indexQ16&0xFFFF, int32(firFracs)))

			/* Inner product */
			interpolPtr := firCoefs[(resamplerDownOrderFIR0/2)*interpolInd:]
			resQ6 = silkmath.Silk_SMULWB(bufPtr[0], int32(interpolPtr[0]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[1], int32(interpolPtr[1]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[2], int32(interpolPtr[2]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[3], int32(interpolPtr[3]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[4], int32(interpolPtr[4]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[5], int32(interpolPtr[5]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[6], int32(interpolPtr[6]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[7], int32(interpolPtr[7]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[8], int32(interpolPtr[8]))
			interpolPtr = firCoefs[(resamplerDownOrderFIR0/2)*(firFracs-1-interpolInd):]
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[17], int32(interpolPtr[0]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[16], int32(interpolPtr[1]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[15], int32(interpolPtr[2]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[14], int32(interpolPtr[3]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[13], int32(interpolPtr[4]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[12], int32(interpolPtr[5]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[11], int32(interpolPtr[6]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[10], int32(interpolPtr[7]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, bufPtr[9], int32(interpolPtr[8]))

			/* Scale down, saturate and store in output array */
			out[outIdx] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(resQ6, 6)))
			outIdx++
		}
	case resamplerDownOrderFIR1:
		for indexQ16 := int32(0); indexQ16 < maxIndexQ16; indexQ16 += indexIncrementQ16 {
			/* Integer part gives pointer to buffered input */
			bufPtr := buf[silkmath.Silk_RSHIFT(indexQ16, 16):]

			/* Inner product */
			resQ6 = silkmath.Silk_SMULWB(silkmath.Silk_ADD32(bufPtr[0], bufPtr[23]), int32(firCoefs[0]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[1], bufPtr[22]), int32(firCoefs[1]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[2], bufPtr[21]), int32(firCoefs[2]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[3], bufPtr[20]), int32(firCoefs[3]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[4], bufPtr[19]), int32(firCoefs[4]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[5], bufPtr[18]), int32(firCoefs[5]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[6], bufPtr[17]), int32(firCoefs[6]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[7], bufPtr[16]), int32(firCoefs[7]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[8], bufPtr[15]), int32(firCoefs[8]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[9], bufPtr[14]), int32(firCoefs[9]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[10], bufPtr[13]), int32(firCoefs[10]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[11], bufPtr[12]), int32(firCoefs[11]))

			/* Scale down, saturate and store in output array */
			out[outIdx] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(resQ6, 6)))
			outIdx++
		}
	case resamplerDownOrderFIR2:
		for indexQ16 := int32(0); indexQ16 < maxIndexQ16; indexQ16 += indexIncrementQ16 {
			/* Integer part gives pointer to buffered input */
			bufPtr := buf[silkmath.Silk_RSHIFT(indexQ16, 16):]

			/* Inner product */
			resQ6 = silkmath.Silk_SMULWB(silkmath.Silk_ADD32(bufPtr[0], bufPtr[35]), int32(firCoefs[0]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[1], bufPtr[34]), int32(firCoefs[1]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[2], bufPtr[33]), int32(firCoefs[2]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[3], bufPtr[32]), int32(firCoefs[3]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[4], bufPtr[31]), int32(firCoefs[4]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[5], bufPtr[30]), int32(firCoefs[5]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[6], bufPtr[29]), int32(firCoefs[6]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[7], bufPtr[28]), int32(firCoefs[7]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[8], bufPtr[27]), int32(firCoefs[8]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[9], bufPtr[26]), int32(firCoefs[9]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[10], bufPtr[25]), int32(firCoefs[10]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[11], bufPtr[24]), int32(firCoefs[11]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[12], bufPtr[23]), int32(firCoefs[12]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[13], bufPtr[22]), int32(firCoefs[13]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[14], bufPtr[21]), int32(firCoefs[14]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[15], bufPtr[20]), int32(firCoefs[15]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[16], bufPtr[19]), int32(firCoefs[16]))
			resQ6 = silkmath.Silk_SMLAWB(resQ6, silkmath.Silk_ADD32(bufPtr[17], bufPtr[18]), int32(firCoefs[17]))

			/* Scale down, saturate and store in output array */
			out[outIdx] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(resQ6, 6)))
			outIdx++
		}
	default:
		/* celt_assert( 0 ); */
	}
	return outIdx
}

// silkResamplerPrivateDownFIR resamples with a 2nd order AR filter followed by FIR
// interpolation (silk/resampler_private_down_FIR.c).
func silkResamplerPrivateDownFIR(S *ResamplerState, out []int16, in []int16, inLen int32) {
	var nSamplesIn int32

	buf := make([]int32, S.BatchSize+S.FIROrder)

	/* Copy buffered samples to start of buffer */
	for i := 0; i < S.FIROrder; i++ {
		buf[i] = S.SFIR[i]
	}

	firCoefs := S.Coefs[2:]

	/* Iterate over blocks of frameSizeIn input samples */
	indexIncrementQ16 := S.InvRatioQ16
	outIdx := 0
	for {
		nSamplesIn = silkmath.Silk_min_32(inLen, int32(S.BatchSize))

		/* Second-order AR filter (output in Q8) */
		silkResamplerPrivateAR2(S.SIIR[:], buf[S.FIROrder:], in, S.Coefs, nSamplesIn)

		maxIndexQ16 := silkmath.Silk_LSHIFT32(nSamplesIn, 16)

		/* Interpolate filtered signal */
		outIdx = silkResamplerPrivateDownFIRInterpol(out, outIdx, buf, firCoefs, S.FIROrder,
			S.FIRFracs, maxIndexQ16, indexIncrementQ16)

		in = in[nSamplesIn:]
		inLen -= nSamplesIn

		if inLen > 1 {
			/* More iterations to do; copy last part of filtered signal to beginning of buffer */
			copy(buf[:S.FIROrder], buf[nSamplesIn:int(nSamplesIn)+S.FIROrder])
		} else {
			break
		}
	}

	/* Copy last part of filtered signal to the state for the next call */
	for i := 0; i < S.FIROrder; i++ {
		S.SFIR[i] = buf[int(nSamplesIn)+i]
	}
}
