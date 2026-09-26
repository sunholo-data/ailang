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

// Transliteration of silk/LPC_analysis_filter.c (libopus v1.6.1). The FIXED_POINT
// oracle build compiles the reference path with USE_CELT_FIR == 0, so this port
// mirrors that scalar branch exactly (the celt_fir variant is a SIMD optimization,
// not the bit-exact reference). silk_decode_core's re-whitening step is the only
// decode-side caller. Names and control flow follow the C for diffability.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// silkLPCAnalysisFilter is silk/LPC_analysis_filter.c silk_LPC_analysis_filter:
// run the LPC analysis (whitening) filter with MA prediction coefficients B (Q12,
// order d) over in[0:length], writing the residual to out[0:length]. The filter
// starts with zero state; the first d output samples are set to zero. Wrap-around
// in the intermediate accumulator is intentional (silk_SMLABB_ovflw), so that two
// wraps can cancel; the rare wrap cases are only reachable from invalid streams.
func silkLPCAnalysisFilter(out, in, B []int16, length, d int) {
	var j, ix int
	var out32Q12, out32 int32

	/* celt_assert( d >= 6 ); celt_assert( (d & 1) == 0 ); celt_assert( d <= len ); */

	for ix = d; ix < length; ix++ {
		/* in_ptr = &in[ ix - 1 ] */
		inPtr := ix - 1

		out32Q12 = silkmath.Silk_SMULBB(int32(in[inPtr+0]), int32(B[0]))
		/* Allowing wrap around so that two wraps can cancel each other. The rare
		   cases where the result wraps around can only be triggered by invalid streams*/
		out32Q12 = silkmath.Silk_SMLABB_ovflw(out32Q12, int32(in[inPtr-1]), int32(B[1]))
		out32Q12 = silkmath.Silk_SMLABB_ovflw(out32Q12, int32(in[inPtr-2]), int32(B[2]))
		out32Q12 = silkmath.Silk_SMLABB_ovflw(out32Q12, int32(in[inPtr-3]), int32(B[3]))
		out32Q12 = silkmath.Silk_SMLABB_ovflw(out32Q12, int32(in[inPtr-4]), int32(B[4]))
		out32Q12 = silkmath.Silk_SMLABB_ovflw(out32Q12, int32(in[inPtr-5]), int32(B[5]))
		for j = 6; j < d; j += 2 {
			out32Q12 = silkmath.Silk_SMLABB_ovflw(out32Q12, int32(in[inPtr-j]), int32(B[j]))
			out32Q12 = silkmath.Silk_SMLABB_ovflw(out32Q12, int32(in[inPtr-j-1]), int32(B[j+1]))
		}

		/* Subtract prediction */
		out32Q12 = silkmath.Silk_SUB32_ovflw(silkmath.Silk_LSHIFT(int32(in[inPtr+1]), 12), out32Q12)

		/* Scale to Q0 */
		out32 = silkmath.Silk_RSHIFT_ROUND(out32Q12, 12)

		/* Saturate output */
		out[ix] = int16(silkmath.Silk_SAT16(out32))
	}

	/* Set first d output samples to zero */
	for ix = 0; ix < d; ix++ {
		out[ix] = 0
	}
}
