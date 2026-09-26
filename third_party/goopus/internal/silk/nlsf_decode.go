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

// Transliteration of silk/NLSF_decode.c, silk/NLSF_unpack.c and
// silk/NLSF_stabilize.c (libopus v1.6.1) for the frozen FIXED_POINT +
// DISABLE_FLOAT_API build: reconstruct the quantized NLSF vector (Q15) from the
// already-decoded codebook path indices and the NLSF codebook, then stabilize it.
// This is a pure function of the indices; the range coder produced NLSFIndices
// earlier (silk_decode_indices). Names follow the C for diffability; the consumed
// codebooks (silkNLSFCBNBMB / silkNLSFCBWB and their leaf tables) live in
// tables_gen.go and all fixed-point math imports internal/silkmath.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// Constants from silk/define.h and silk/SigProc_FIX.h.
const (
	silkMaxOrderLPC = 24 // SILK_MAX_ORDER_LPC (SigProc_FIX.h): stack-array sizing
	maxLPCOrder     = 16 // MAX_LPC_ORDER (define.h): largest actual filter order

	maxLPCStabilizeIterations = 16 // MAX_LPC_STABILIZE_ITERATIONS (define.h)
	nlsfQuantMaxAmplitude     = 4  // NLSF_QUANT_MAX_AMPLITUDE (define.h)

	// Integer range constants used where the C compares against silk_intXX_MIN/MAX.
	silkInt16MAX = 32767
	silkInt32MAX = 2147483647
	silkInt32MIN = -2147483648
)

// nlsfQuantLevelAdjQ10 is SILK_FIX_CONST( NLSF_QUANT_LEVEL_ADJ, 10 ) with
// NLSF_QUANT_LEVEL_ADJ == 0.1 (define.h), used by silk_NLSF_residual_dequant.
var nlsfQuantLevelAdjQ10 = silkFixConst(0.1, 10)

// silkFixConst mirrors SILK_FIX_CONST(C, Q) =
// (opus_int32)( (C) * ((opus_int64)1 << (Q)) + 0.5 ). C is a compile-time double
// literal in libopus; evaluating in float64 (== C double) makes the truncation
// bit-identical to the oracle.
func silkFixConst(c float64, q int) int32 {
	return int32(c*float64(int64(1)<<q) + 0.5)
}

// silkNLSFResidualDequant is the static predictive dequantizer for NLSF residuals
// (silk/NLSF_decode.c silk_NLSF_residual_dequant): reconstruct the residual x_Q10
// from the quantization indices, the backward predictor coefs and the step size.
// out_Q10/pred_Q10 are opus_int (int32 here); the silk_SUB16/silk_ADD16 level
// adjust is plain int arithmetic (the C macros have no outer cast, so out_Q10 is
// NOT truncated to int16 there).
func silkNLSFResidualDequant(x_Q10 []int16, indices []int8, pred_coef_Q8 []byte, quant_step_size_Q16 int32, order int) {
	var i int
	var out_Q10, pred_Q10 int32

	out_Q10 = 0
	for i = order - 1; i >= 0; i-- {
		pred_Q10 = silkmath.Silk_RSHIFT(silkmath.Silk_SMULBB(out_Q10, int32(int16(pred_coef_Q8[i]))), 8)
		out_Q10 = silkmath.Silk_LSHIFT(int32(indices[i]), 10)
		if out_Q10 > 0 {
			out_Q10 -= nlsfQuantLevelAdjQ10 /* silk_SUB16( out_Q10, SILK_FIX_CONST( NLSF_QUANT_LEVEL_ADJ, 10 ) ) */
		} else if out_Q10 < 0 {
			out_Q10 += nlsfQuantLevelAdjQ10 /* silk_ADD16( out_Q10, SILK_FIX_CONST( NLSF_QUANT_LEVEL_ADJ, 10 ) ) */
		}
		out_Q10 = silkmath.Silk_SMLAWB(pred_Q10, out_Q10, quant_step_size_Q16)
		x_Q10[i] = int16(out_Q10)
	}
}

// silkNLSFUnpack is silk/NLSF_unpack.c silk_NLSF_unpack: unpack the predictor
// values and the entropy-table indices for the given first-stage codebook index.
func silkNLSFUnpack(ec_ix []int16, pred_Q8 []byte, psNLSF_CB *silkNLSFCBStruct, CB1_index int) {
	var i int
	var entry byte

	order := int(psNLSF_CB.order)
	ec_sel_ptr := psNLSF_CB.ecSel[CB1_index*order/2:]
	sel := 0
	for i = 0; i < order; i += 2 {
		entry = ec_sel_ptr[sel]
		sel++
		ec_ix[i] = int16(silkmath.Silk_SMULBB(silkmath.Silk_RSHIFT(int32(entry), 1)&7, 2*nlsfQuantMaxAmplitude+1))
		pred_Q8[i] = psNLSF_CB.predQ8[i+int(entry&1)*(order-1)]
		ec_ix[i+1] = int16(silkmath.Silk_SMULBB(silkmath.Silk_RSHIFT(int32(entry), 5)&7, 2*nlsfQuantMaxAmplitude+1))
		pred_Q8[i+1] = psNLSF_CB.predQ8[i+int(silkmath.Silk_RSHIFT(int32(entry), 4)&1)*(order-1)+1]
	}
}

// silkNLSFDecode is silk/NLSF_decode.c silk_NLSF_decode: reconstruct the quantized
// NLSF vector pNLSF_Q15 (Q15) from the codebook path vector NLSFIndices
// ([order+1], NLSFIndices[0] is the first-stage codebook index) and the codebook.
func silkNLSFDecode(pNLSF_Q15 []int16, NLSFIndices []int8, psNLSF_CB *silkNLSFCBStruct) {
	var i int
	var pred_Q8 [maxLPCOrder]byte
	var ec_ix [maxLPCOrder]int16
	var res_Q10 [maxLPCOrder]int16
	var NLSF_Q15_tmp int32

	order := int(psNLSF_CB.order)

	/* Unpack entropy table indices and predictor for current CB1 index */
	silkNLSFUnpack(ec_ix[:], pred_Q8[:], psNLSF_CB, int(NLSFIndices[0]))

	/* Predictive residual dequantizer */
	silkNLSFResidualDequant(res_Q10[:], NLSFIndices[1:], pred_Q8[:], int32(psNLSF_CB.quantStepSizeQ16), order)

	/* Apply inverse square-rooted weights to first stage and add to output */
	off := int(NLSFIndices[0]) * order
	pCB_element := psNLSF_CB.cb1NLSFQ8[off:]
	pCB_Wght_Q9 := psNLSF_CB.cb1WghtQ9[off:]
	for i = 0; i < order; i++ {
		NLSF_Q15_tmp = silkmath.Silk_ADD_LSHIFT32(silkmath.Silk_DIV32_16(silkmath.Silk_LSHIFT(int32(res_Q10[i]), 14), int32(pCB_Wght_Q9[i])), int32(int16(pCB_element[i])), 7)
		pNLSF_Q15[i] = int16(silkmath.Silk_LIMIT_32(NLSF_Q15_tmp, 0, 32767))
	}

	/* NLSF stabilization */
	NLSFStabilize(pNLSF_Q15, psNLSF_CB.deltaMinQ15, order)
}

// NLSFDecode reconstructs the Q15 NLSF vector from the codebook path indices,
// selecting the SILK NLSF codebook by filter order: 16 for the wideband codebook
// (silk_NLSF_CB_WB), otherwise the NB/MB codebook (silk_NLSF_CB_NB_MB, order 10),
// matching how the SILK decoder picks psDec->psNLSF_CB from the internal sample
// rate. pNLSF_Q15 receives order values; NLSFIndices is [order+1].
func NLSFDecode(pNLSF_Q15 []int16, NLSFIndices []int8, order int) {
	cb := &silkNLSFCBNBMB
	if order == 16 {
		cb = &silkNLSFCBWB
	}
	silkNLSFDecode(pNLSF_Q15, NLSFIndices, cb)
}

// stabilizeMaxLoops is MAX_LOOPS from silk/NLSF_stabilize.c.
const stabilizeMaxLoops = 20

// NLSFStabilize is silk/NLSF_stabilize.c silk_NLSF_stabilize: move NLSFs apart if
// they are too close, and away from the [0, 1<<15) borders, with minimum Euclidean
// distance to the input; the output is a sorted NLSF vector. NDeltaMin_Q15 is the
// [L+1] minimum-distance vector (NDeltaMin_Q15[L] must be >= 1).
func NLSFStabilize(NLSF_Q15 []int16, NDeltaMin_Q15 []int16, L int) {
	var i, I, k, loops int
	var center_freq_Q15 int16
	var diff_Q15, min_diff_Q15, min_center_Q15, max_center_Q15 int32

	/* This is necessary to ensure an output within range of a opus_int16 */
	// silk_assert( NDeltaMin_Q15[L] >= 1 );

	for loops = 0; loops < stabilizeMaxLoops; loops++ {
		/**************************/
		/* Find smallest distance */
		/**************************/
		/* First element */
		min_diff_Q15 = int32(NLSF_Q15[0]) - int32(NDeltaMin_Q15[0])
		I = 0
		/* Middle elements */
		for i = 1; i <= L-1; i++ {
			diff_Q15 = int32(NLSF_Q15[i]) - (int32(NLSF_Q15[i-1]) + int32(NDeltaMin_Q15[i]))
			if diff_Q15 < min_diff_Q15 {
				min_diff_Q15 = diff_Q15
				I = i
			}
		}
		/* Last element */
		diff_Q15 = (1 << 15) - (int32(NLSF_Q15[L-1]) + int32(NDeltaMin_Q15[L]))
		if diff_Q15 < min_diff_Q15 {
			min_diff_Q15 = diff_Q15
			I = L
		}

		/***************************************************/
		/* Now check if the smallest distance non-negative */
		/***************************************************/
		if min_diff_Q15 >= 0 {
			return
		}

		switch I {
		case 0:
			/* Move away from lower limit */
			NLSF_Q15[0] = NDeltaMin_Q15[0]

		case L:
			/* Move away from higher limit */
			NLSF_Q15[L-1] = int16((1 << 15) - int32(NDeltaMin_Q15[L]))

		default:
			/* Find the lower extreme for the location of the current center frequency */
			min_center_Q15 = 0
			for k = 0; k < I; k++ {
				min_center_Q15 += int32(NDeltaMin_Q15[k])
			}
			min_center_Q15 += silkmath.Silk_RSHIFT(int32(NDeltaMin_Q15[I]), 1)

			/* Find the upper extreme for the location of the current center frequency */
			max_center_Q15 = 1 << 15
			for k = L; k > I; k-- {
				max_center_Q15 -= int32(NDeltaMin_Q15[k])
			}
			max_center_Q15 -= silkmath.Silk_RSHIFT(int32(NDeltaMin_Q15[I]), 1)

			/* Move apart, sorted by value, keeping the same center frequency */
			center_freq_Q15 = int16(silkmath.Silk_LIMIT_32(silkmath.Silk_RSHIFT_ROUND(int32(NLSF_Q15[I-1])+int32(NLSF_Q15[I]), 1),
				min_center_Q15, max_center_Q15))
			NLSF_Q15[I-1] = int16(int32(center_freq_Q15) - silkmath.Silk_RSHIFT(int32(NDeltaMin_Q15[I]), 1))
			NLSF_Q15[I] = int16(int32(NLSF_Q15[I-1]) + int32(NDeltaMin_Q15[I]))
		}
	}

	/* Safe and simple fall back method, which is less ideal than the above */
	if loops == stabilizeMaxLoops {
		/* Insertion sort (fast for already almost sorted arrays):   */
		/* Best case:  O(n)   for an already sorted array            */
		/* Worst case: O(n^2) for an inversely sorted array          */
		silkInsertionSortIncreasingAllValuesInt16(NLSF_Q15, L)

		/* First NLSF should be no less than NDeltaMin[0] */
		NLSF_Q15[0] = int16(silkmath.Silk_max_int(int(NLSF_Q15[0]), int(NDeltaMin_Q15[0])))

		/* Keep delta_min distance between the NLSFs */
		for i = 1; i < L; i++ {
			NLSF_Q15[i] = int16(silkmath.Silk_max_int(int(NLSF_Q15[i]), int(silkmath.Silk_ADD_SAT16(NLSF_Q15[i-1], NDeltaMin_Q15[i]))))
		}

		/* Last NLSF should be no higher than 1 - NDeltaMin[L] */
		NLSF_Q15[L-1] = int16(silkmath.Silk_min_int(int(NLSF_Q15[L-1]), (1<<15)-int(NDeltaMin_Q15[L])))

		/* Keep NDeltaMin distance between the NLSFs */
		for i = L - 2; i >= 0; i-- {
			NLSF_Q15[i] = int16(silkmath.Silk_min_int(int(NLSF_Q15[i]), int(NLSF_Q15[i+1])-int(NDeltaMin_Q15[i+1])))
		}
	}
}
