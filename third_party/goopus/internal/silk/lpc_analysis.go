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

// Transliteration of silk/LPC_fit.c and silk/LPC_inv_pred_gain.c (libopus
// v1.6.1) for the frozen FIXED_POINT + DISABLE_FLOAT_API + OPUS_FAST_INT64 build.
// silk_LPC_fit converts int32 AR coefficients to int16 without wrap-around;
// silk_LPC_inverse_pred_gain computes the inverse prediction gain and tests LPC
// stability (all poles within the unit circle). Both feed silk_NLSF2A's stability
// loop. Names follow the C for diffability; all fixed-point math imports
// internal/silkmath.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// lpcFit999Q16 is SILK_FIX_CONST( 0.999, 16 ) from silk_LPC_fit.
var lpcFit999Q16 = silkFixConst(0.999, 16)

// silkLPCFit is silk/LPC_fit.c silk_LPC_fit: convert int32 coefficients a_QIN
// (Q domain QIN) to int16 coefficients a_QOUT (Q domain QOUT), making sure there
// is no wrap-around by bandwidth-expanding a_QIN until the coefficients fit in
// int16. a_QIN is modified in place. This logic is reused in _celt_lpc(); any bug
// fixes should also be applied there.
func silkLPCFit(a_QOUT []int16, a_QIN []int32, QOUT, QIN, d int) {
	var i, k, idx int
	var maxabs, absval, chirp_Q16 int32

	/* Limit the maximum absolute value of the prediction coefficients, so that they'll fit in int16 */
	for i = 0; i < 10; i++ {
		/* Find maximum absolute value and its index */
		maxabs = 0
		for k = 0; k < d; k++ {
			absval = silkmath.Silk_abs(a_QIN[k])
			if absval > maxabs {
				maxabs = absval
				idx = k
			}
		}
		maxabs = silkmath.Silk_RSHIFT_ROUND(maxabs, QIN-QOUT)

		if maxabs > silkInt16MAX {
			/* Reduce magnitude of prediction coefficients */
			maxabs = silkmath.Silk_min_32(maxabs, 163838) /* ( silk_int32_MAX >> 14 ) + silk_int16_MAX = 163838 */
			chirp_Q16 = lpcFit999Q16 - silkmath.Silk_DIV32(silkmath.Silk_LSHIFT(maxabs-silkInt16MAX, 14),
				silkmath.Silk_RSHIFT32(silkmath.Silk_MUL(maxabs, int32(idx+1)), 2))
			Bwexpander32(a_QIN, d, chirp_Q16)
		} else {
			break
		}
	}

	if i == 10 {
		/* Reached the last iteration, clip the coefficients */
		for k = 0; k < d; k++ {
			a_QOUT[k] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(a_QIN[k], QIN-QOUT)))
			a_QIN[k] = silkmath.Silk_LSHIFT(int32(a_QOUT[k]), QIN-QOUT)
		}
	} else {
		for k = 0; k < d; k++ {
			a_QOUT[k] = int16(silkmath.Silk_RSHIFT_ROUND(a_QIN[k], QIN-QOUT))
		}
	}
}

// QA and A_LIMIT for silk_LPC_inverse_pred_gain (LPC_inv_pred_gain.c).
const lpcInvGainQA = 24

var (
	aLimitQA24 = silkFixConst(0.99975, lpcInvGainQA) // A_LIMIT = SILK_FIX_CONST( 0.99975, QA )
	oneQ30     = silkFixConst(1, 30)                 // SILK_FIX_CONST( 1, 30 )
	// maxPredGainInvQ30 is SILK_FIX_CONST( 1.0f / MAX_PREDICTION_POWER_GAIN, 30 ).
	maxPredGainInvQ30 = computeMaxPredGainInvQ30()
)

// computeMaxPredGainInvQ30 evaluates SILK_FIX_CONST( 1.0f / MAX_PREDICTION_POWER_GAIN,
// 30 ). MAX_PREDICTION_POWER_GAIN is 1e4f, so the reciprocal and the scaling are
// done in float32 (matching C's float arithmetic), then + 0.5 in double and
// truncated. Kept as a runtime computation (not a const) so the float32 rounding
// is applied exactly as the oracle does.
func computeMaxPredGainInvQ30() int32 {
	const maxPredictionPowerGain float32 = 1e4
	inv := float32(1.0) / maxPredictionPowerGain
	prod := inv * float32(int64(1)<<30)
	return int32(float64(prod) + 0.5)
}

// mul32FracQ is the MUL32_FRAC_Q macro from LPC_inv_pred_gain.c:
// (opus_int32)( silk_RSHIFT_ROUND64( silk_SMULL(a32, b32), Q ) ).
func mul32FracQ(a32, b32 int32, Q int) int32 {
	return int32(silkmath.Silk_RSHIFT_ROUND64(silkmath.Silk_SMULL(a32, b32), Q))
}

// lpcInversePredGainQA is the static LPC_inverse_pred_gain_QA_c helper: compute
// the inverse of the LPC prediction gain (energy domain, Q30) and test whether the
// LPC coefficients are stable (all poles within the unit circle). A_QA is modified
// in place. Returns 0 when unstable.
func lpcInversePredGainQA(A_QA []int32, order int) int32 {
	var k, n, mult2Q int
	var invGain_Q30, rc_Q31, rc_mult1_Q30, rc_mult2, tmp1, tmp2 int32

	invGain_Q30 = oneQ30
	for k = order - 1; k > 0; k-- {
		/* Check for stability */
		if (A_QA[k] > aLimitQA24) || (A_QA[k] < -aLimitQA24) {
			return 0
		}

		/* Set RC equal to negated AR coef */
		rc_Q31 = -silkmath.Silk_LSHIFT(A_QA[k], 31-lpcInvGainQA)

		/* rc_mult1_Q30 range: [ 1 : 2^30 ] */
		rc_mult1_Q30 = silkmath.Silk_SUB32(oneQ30, silkmath.Silk_SMMUL(rc_Q31, rc_Q31))

		/* Update inverse gain */
		/* invGain_Q30 range: [ 0 : 2^30 ] */
		invGain_Q30 = silkmath.Silk_LSHIFT(silkmath.Silk_SMMUL(invGain_Q30, rc_mult1_Q30), 2)
		if invGain_Q30 < maxPredGainInvQ30 {
			return 0
		}

		/* rc_mult2 range: [ 2^30 : silk_int32_MAX ] */
		mult2Q = 32 - int(silkmath.Silk_CLZ32(silkmath.Silk_abs(rc_mult1_Q30)))
		rc_mult2 = silkmath.Silk_INVERSE32_varQ(rc_mult1_Q30, mult2Q+30)

		/* Update AR coefficient */
		for n = 0; n < (k+1)>>1; n++ {
			var tmp64 int64
			tmp1 = A_QA[n]
			tmp2 = A_QA[k-n-1]
			tmp64 = silkmath.Silk_RSHIFT_ROUND64(silkmath.Silk_SMULL(silkmath.Silk_SUB_SAT32(tmp1,
				mul32FracQ(tmp2, rc_Q31, 31)), rc_mult2), mult2Q)
			if tmp64 > int64(silkInt32MAX) || tmp64 < int64(silkInt32MIN) {
				return 0
			}
			A_QA[n] = int32(tmp64)
			tmp64 = silkmath.Silk_RSHIFT_ROUND64(silkmath.Silk_SMULL(silkmath.Silk_SUB_SAT32(tmp2,
				mul32FracQ(tmp1, rc_Q31, 31)), rc_mult2), mult2Q)
			if tmp64 > int64(silkInt32MAX) || tmp64 < int64(silkInt32MIN) {
				return 0
			}
			A_QA[k-n-1] = int32(tmp64)
		}
	}

	/* Check for stability */
	if (A_QA[k] > aLimitQA24) || (A_QA[k] < -aLimitQA24) {
		return 0
	}

	/* Set RC equal to negated AR coef */
	rc_Q31 = -silkmath.Silk_LSHIFT(A_QA[0], 31-lpcInvGainQA)

	/* Range: [ 1 : 2^30 ] */
	rc_mult1_Q30 = silkmath.Silk_SUB32(oneQ30, silkmath.Silk_SMMUL(rc_Q31, rc_Q31))

	/* Update inverse gain */
	/* Range: [ 0 : 2^30 ] */
	invGain_Q30 = silkmath.Silk_LSHIFT(silkmath.Silk_SMMUL(invGain_Q30, rc_mult1_Q30), 2)
	if invGain_Q30 < maxPredGainInvQ30 {
		return 0
	}

	return invGain_Q30
}

// LPCInversePredGain is silk/LPC_inv_pred_gain.c silk_LPC_inverse_pred_gain_c: for
// int16 Q12 input coefficients A_Q12, return the inverse prediction gain (energy
// domain, Q30), or 0 if the filter is unstable.
func LPCInversePredGain(A_Q12 []int16, order int) int32 {
	var k int
	var Atmp_QA [silkMaxOrderLPC]int32
	var DC_resp int32

	/* Increase Q domain of the AR coefficients */
	for k = 0; k < order; k++ {
		DC_resp += int32(A_Q12[k])
		Atmp_QA[k] = silkmath.Silk_LSHIFT32(int32(A_Q12[k]), lpcInvGainQA-12)
	}
	/* If the DC is unstable, we don't even need to do the full calculations */
	if DC_resp >= 4096 {
		return 0
	}
	return lpcInversePredGainQA(Atmp_QA[:], order)
}
