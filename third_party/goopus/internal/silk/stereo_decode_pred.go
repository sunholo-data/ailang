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

// Transliteration of silk/stereo_decode_pred.c (libopus v1.6.1): the range-decode
// side of the SILK stereo predictor. silk_stereo_decode_pred reads the two mid/side
// prediction weights pred_Q13[] the stereo un-mixing (silk_stereo_MS_to_LR) later
// applies, and silk_stereo_decode_mid_only reads the flag that the encoder coded only
// the mid channel. Both run once per packet, before the per-channel silk_decode_frame,
// off the same range decoder. Names, control flow and comments follow the C for
// diffability. The iCDF tables (silk_stereo_pred_joint_iCDF, silk_uniform3_iCDF,
// silk_uniform5_iCDF, silk_stereo_only_code_mid_iCDF) and the dequant codebook
// (silk_stereo_pred_quant_Q13) come from tables_gen.go.

package silk

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"
)

// Constants from silk/define.h. STEREO_QUANT_TAB_SIZE (16) is implicit in the length
// of silkStereoPredQuantQ13.
const (
	stereoQuantSubSteps = 5 // STEREO_QUANT_SUB_STEPS
	stereoInterpLenMS   = 8 // STEREO_INTERP_LEN_MS (must be even)
)

// stereoQuantStepFactorQ16 is SILK_FIX_CONST( 0.5 / STEREO_QUANT_SUB_STEPS, 16 ), the
// half-sub-step scaling silk_stereo_decode_pred multiplies the codebook interval by
// when placing a predictor between two quantizer levels. Evaluated as a float64 (== C
// double) so the truncation matches the oracle.
var stereoQuantStepFactorQ16 = silkFixConst(0.5/stereoQuantSubSteps, 16)

// StereoDecodePred is silk/stereo_decode_pred.c silk_stereo_decode_pred: decode the two
// mid/side predictors into predQ13. It reads a joint index (silk_stereo_pred_joint_iCDF)
// that splits into the two coarse indices, then a uniform3/uniform5 pair per channel,
// dequantizes each weight from silk_stereo_pred_quant_Q13, and finally subtracts the
// second predictor from the first (which is how silk_stereo_MS_to_LR consumes them).
func StereoDecodePred(psRangeDec *rangecoding.Decoder, predQ13 *[2]int32) {
	var ix [2][3]int

	/* Entropy decoding */
	n := psRangeDec.DecIcdf(silkStereoPredJointICDF, 8)
	ix[0][2] = int(silkmath.Silk_DIV32_16(int32(n), 5))
	ix[1][2] = n - 5*ix[0][2]
	for k := 0; k < 2; k++ {
		ix[k][0] = psRangeDec.DecIcdf(silkUniform3ICDF, 8)
		ix[k][1] = psRangeDec.DecIcdf(silkUniform5ICDF, 8)
	}

	/* Dequantize */
	for k := 0; k < 2; k++ {
		ix[k][0] += 3 * ix[k][2]
		lowQ13 := int32(silkStereoPredQuantQ13[ix[k][0]])
		stepQ13 := silkmath.Silk_SMULWB(int32(silkStereoPredQuantQ13[ix[k][0]+1])-lowQ13,
			stereoQuantStepFactorQ16)
		predQ13[k] = silkmath.Silk_SMLABB(lowQ13, stepQ13, int32(2*ix[k][1]+1))
	}

	/* Subtract second from first predictor (helps when actually applying these) */
	predQ13[0] -= predQ13[1]
}

// StereoDecodeMidOnly is silk/stereo_decode_pred.c silk_stereo_decode_mid_only: decode
// the flag that only the mid channel was coded (silk_stereo_only_code_mid_iCDF). Returns
// the flag (the C writes it through *decode_only_mid).
func StereoDecodeMidOnly(psRangeDec *rangecoding.Decoder) int {
	/* Decode flag that only mid channel is coded */
	return psRangeDec.DecIcdf(silkStereoOnlyCodeMidICDF, 8)
}
