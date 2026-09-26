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

// Transliteration of silk/decode_indices.c (libopus v1.6.1): silk_decode_indices
// reads the per-frame side-information indices from the range decoder into
// psDec.Indices, the raw quantization indices the parameter decode later expands
// into gains, LPC coefficients, pitch lags and LTP taps. This is where the SILK
// bitstream is actually parsed (via ec_dec_icdf); the iCDF tables and the NLSF
// codebook come from tables_gen.go / tables.go. Names, control flow and comments
// follow the C for diffability.

package silk

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"
)

// DecodeIndices is silk/decode_indices.c silk_decode_indices: decode the
// side-information parameters from the payload. FrameIndex selects the VAD flag;
// decodeLBRR is nonzero when LBRR (redundant) data is being decoded; condCoding is
// CODE_INDEPENDENTLY / CODE_INDEPENDENTLY_NO_LTP_SCALING / CODE_CONDITIONALLY.
func DecodeIndices(psDec *DecoderState, psRangeDec *rangecoding.Decoder, FrameIndex, decodeLBRR, condCoding int) {
	var i, k, Ix int
	var decodeAbsoluteLagIndex, deltaLagIndex int
	var ecIx [maxLPCOrder]int16
	var predQ8 [maxLPCOrder]byte

	/*******************************************/
	/* Decode signal type and quantizer offset */
	/*******************************************/
	if decodeLBRR != 0 || psDec.VADFlags[FrameIndex] != 0 {
		Ix = psRangeDec.DecIcdf(silkTypeOffsetVADICDF, 8) + 2
	} else {
		Ix = psRangeDec.DecIcdf(silkTypeOffsetNoVADICDF, 8)
	}
	psDec.Indices.SignalType = int8(silkmath.Silk_RSHIFT(int32(Ix), 1))
	psDec.Indices.QuantOffsetType = int8(Ix & 1)

	/****************/
	/* Decode gains */
	/****************/
	/* First subframe */
	if condCoding == codeConditionally {
		/* Conditional coding */
		psDec.Indices.GainsIndices[0] = int8(psRangeDec.DecIcdf(silkDeltaGainICDF, 8))
	} else {
		/* Independent coding, in two stages: MSB bits followed by 3 LSBs */
		psDec.Indices.GainsIndices[0] = int8(silkmath.Silk_LSHIFT(int32(psRangeDec.DecIcdf(silkGainICDF[psDec.Indices.SignalType][:], 8)), 3))
		psDec.Indices.GainsIndices[0] += int8(psRangeDec.DecIcdf(silkUniform8ICDF, 8))
	}

	/* Remaining subframes */
	for i = 1; i < psDec.NbSubfr; i++ {
		psDec.Indices.GainsIndices[i] = int8(psRangeDec.DecIcdf(silkDeltaGainICDF, 8))
	}

	/**********************/
	/* Decode LSF Indices */
	/**********************/
	cb1Off := int(psDec.Indices.SignalType>>1) * int(psDec.psNLSFCB.nVectors)
	psDec.Indices.NLSFIndices[0] = int8(psRangeDec.DecIcdf(psDec.psNLSFCB.cb1ICDF[cb1Off:], 8))
	silkNLSFUnpack(ecIx[:], predQ8[:], psDec.psNLSFCB, int(psDec.Indices.NLSFIndices[0]))
	// celt_assert( psDec->psNLSF_CB->order == psDec->LPC_order );
	for i = 0; i < int(psDec.psNLSFCB.order); i++ {
		Ix = psRangeDec.DecIcdf(psDec.psNLSFCB.ecICDF[int(ecIx[i]):], 8)
		if Ix == 0 {
			Ix -= psRangeDec.DecIcdf(silkNLSFEXTICDF, 8)
		} else if Ix == 2*nlsfQuantMaxAmplitude {
			Ix += psRangeDec.DecIcdf(silkNLSFEXTICDF, 8)
		}
		psDec.Indices.NLSFIndices[i+1] = int8(Ix - nlsfQuantMaxAmplitude)
	}

	/* Decode LSF interpolation factor */
	if psDec.NbSubfr == maxNBSubfr {
		psDec.Indices.NLSFInterpCoefQ2 = int8(psRangeDec.DecIcdf(silkNLSFInterpolationFactorICDF, 8))
	} else {
		psDec.Indices.NLSFInterpCoefQ2 = 4
	}

	if psDec.Indices.SignalType == typeVoiced {
		/*********************/
		/* Decode pitch lags */
		/*********************/
		/* Get lag index */
		decodeAbsoluteLagIndex = 1
		if condCoding == codeConditionally && psDec.ECPrevSignalType == typeVoiced {
			/* Decode Delta index */
			deltaLagIndex = int(int16(psRangeDec.DecIcdf(silkPitchDeltaICDF, 8)))
			if deltaLagIndex > 0 {
				deltaLagIndex = deltaLagIndex - 9
				psDec.Indices.LagIndex = int16(int(psDec.ECPrevLagIndex) + deltaLagIndex)
				decodeAbsoluteLagIndex = 0
			}
		}
		if decodeAbsoluteLagIndex != 0 {
			/* Absolute decoding */
			psDec.Indices.LagIndex = int16(int(int16(psRangeDec.DecIcdf(silkPitchLagICDF, 8))) * int(silkmath.Silk_RSHIFT(int32(psDec.FsKHz), 1)))
			psDec.Indices.LagIndex += int16(psRangeDec.DecIcdf(psDec.pitchLagLowBitsICDF, 8))
		}
		psDec.ECPrevLagIndex = psDec.Indices.LagIndex

		/* Get contour index */
		psDec.Indices.ContourIndex = int8(psRangeDec.DecIcdf(psDec.pitchContourICDF, 8))

		/********************/
		/* Decode LTP gains */
		/********************/
		/* Decode PERIndex value */
		psDec.Indices.PERIndex = int8(psRangeDec.DecIcdf(silkLTPPerIndexICDF, 8))

		for k = 0; k < psDec.NbSubfr; k++ {
			psDec.Indices.LTPIndex[k] = int8(psRangeDec.DecIcdf(silkLTPGainICDFPtrs[psDec.Indices.PERIndex], 8))
		}

		/**********************/
		/* Decode LTP scaling */
		/**********************/
		if condCoding == codeIndependently {
			psDec.Indices.LTPScaleIndex = int8(psRangeDec.DecIcdf(silkLTPscaleICDF, 8))
		} else {
			psDec.Indices.LTPScaleIndex = 0
		}
	}
	psDec.ECPrevSignalType = int(psDec.Indices.SignalType)

	/***************/
	/* Decode seed */
	/***************/
	psDec.Indices.Seed = int8(psRangeDec.DecIcdf(silkUniform4ICDF, 8))
}
