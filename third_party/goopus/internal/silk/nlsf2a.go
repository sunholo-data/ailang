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

// Transliteration of silk/NLSF2A.c (libopus v1.6.1) for the frozen FIXED_POINT +
// DISABLE_FLOAT_API build: convert normalized line spectral frequencies (Q15) to
// the monic short-term whitening filter coefficients a_Q12. A piecewise linear
// approximation maps LSF <-> cos(LSF), so the result is not accurate LSFs, but the
// forward and inverse functions are accurate inverses of each other. The order
// must be even (10 or 16 on the decode path). Names follow the C for diffability;
// the consumed cosine table silkLSFCosTabFIXQ12 lives in tables_gen.go and the
// fixed-point math imports internal/silkmath.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// QA for silk_NLSF2A (NLSF2A.c #define QA 16).
const nlsf2aQA = 16

// silkNLSF2AFindPoly is the static helper silk_NLSF2A_find_poly: generate an
// intermediate polynomial (Q domain QA, dd+1 entries) by convolution from the
// interleaved 2*cos(LSF) vector cLSF (Q domain QA).
func silkNLSF2AFindPoly(out []int32, cLSF []int32, dd int) {
	var k, n int
	var ftmp int32

	out[0] = silkmath.Silk_LSHIFT(1, nlsf2aQA)
	out[1] = -cLSF[0]
	for k = 1; k < dd; k++ {
		ftmp = cLSF[2*k] /* QA*/
		out[k+1] = silkmath.Silk_LSHIFT(out[k-1], 1) - int32(silkmath.Silk_RSHIFT_ROUND64(silkmath.Silk_SMULL(ftmp, out[k]), nlsf2aQA))
		for n = k; n > 1; n-- {
			out[n] += out[n-2] - int32(silkmath.Silk_RSHIFT_ROUND64(silkmath.Silk_SMULL(ftmp, out[n-1]), nlsf2aQA))
		}
		out[1] -= ftmp
	}
}

// NLSF2A is silk/NLSF2A.c silk_NLSF2A: compute the monic whitening filter
// coefficients a_Q12 (Q12) from the normalized line spectral frequencies NLSF
// (Q15). d is the filter order and must be even (10 or 16). The C arch argument is
// dropped: this is the scalar reference path.
func NLSF2A(a_Q12 []int16, NLSF []int16, d int) {
	/* This ordering was found to maximize quality. It improves numerical accuracy of
	   silk_NLSF2A_find_poly() compared to "standard" ordering. */
	ordering16 := [16]byte{
		0, 15, 8, 7, 4, 11, 12, 3, 2, 13, 10, 5, 6, 9, 14, 1,
	}
	ordering10 := [10]byte{
		0, 9, 6, 3, 4, 5, 8, 1, 2, 7,
	}
	var ordering []byte
	var k, i, dd int
	var cos_LSF_QA [silkMaxOrderLPC]int32
	var P, Q [silkMaxOrderLPC/2 + 1]int32
	var Ptmp, Qtmp, f_int, f_frac, cos_val, delta int32
	var a32_QA1 [silkMaxOrderLPC]int32

	// silk_assert( LSF_COS_TAB_SZ_FIX == 128 );
	// celt_assert( d==10 || d==16 );

	/* convert LSFs to 2*cos(LSF), using piecewise linear curve from table */
	if d == 16 {
		ordering = ordering16[:]
	} else {
		ordering = ordering10[:]
	}
	for k = 0; k < d; k++ {
		// silk_assert( NLSF[k] >= 0 );

		/* f_int on a scale 0-127 (rounded down) */
		f_int = silkmath.Silk_RSHIFT(int32(NLSF[k]), 15-7)

		/* f_frac, range: 0..255 */
		f_frac = int32(NLSF[k]) - silkmath.Silk_LSHIFT(f_int, 15-7)

		// silk_assert(f_int >= 0);
		// silk_assert(f_int < LSF_COS_TAB_SZ_FIX );

		/* Read start and end value from table */
		cos_val = int32(silkLSFCosTabFIXQ12[f_int])           /* Q12 */
		delta = int32(silkLSFCosTabFIXQ12[f_int+1]) - cos_val /* Q12, with a range of 0..200 */

		/* Linear interpolation */
		cos_LSF_QA[ordering[k]] = silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_LSHIFT(cos_val, 8)+silkmath.Silk_MUL(delta, f_frac), 20-nlsf2aQA) /* QA */
	}

	dd = d >> 1 /* silk_RSHIFT( d, 1 ) */

	/* generate even and odd polynomials using convolution */
	silkNLSF2AFindPoly(P[:], cos_LSF_QA[0:], dd)
	silkNLSF2AFindPoly(Q[:], cos_LSF_QA[1:], dd)

	/* convert even and odd polynomials to opus_int32 Q12 filter coefs */
	for k = 0; k < dd; k++ {
		Ptmp = P[k+1] + P[k]
		Qtmp = Q[k+1] - Q[k]

		/* the Ptmp and Qtmp values at this stage need to fit in int32 */
		a32_QA1[k] = -Qtmp - Ptmp    /* QA+1 */
		a32_QA1[d-k-1] = Qtmp - Ptmp /* QA+1 */
	}

	/* Convert int32 coefficients to Q12 int16 coefs */
	silkLPCFit(a_Q12, a32_QA1[:], 12, nlsf2aQA+1, d)

	for i = 0; LPCInversePredGain(a_Q12, d) == 0 && i < maxLPCStabilizeIterations; i++ {
		/* Prediction coefficients are (too close to) unstable; apply bandwidth expansion   */
		/* on the unscaled coefficients, convert to Q12 and measure again                   */
		Bwexpander32(a32_QA1[:], d, 65536-silkmath.Silk_LSHIFT(2, i))
		for k = 0; k < d; k++ {
			a_Q12[k] = int16(silkmath.Silk_RSHIFT_ROUND(a32_QA1[k], nlsf2aQA+1-12)) /* QA+1 -> Q12 */
		}
	}
}
