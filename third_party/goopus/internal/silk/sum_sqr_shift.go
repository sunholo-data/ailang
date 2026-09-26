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

// Transliteration of silk/sum_sqr_shift.c (libopus v1.6.1). Ported here as a
// helper for the SILK packet-loss concealment (silk/PLC.c silk_PLC_energy and
// silk_PLC_glue_frames call it). The intermediate accumulator is unsigned and the
// per-sample products are allowed to wrap (silk_SMLABB_ovflw), matching the C.
// Names and control flow follow the C for diffability.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// silkSumSqrShift is silk/sum_sqr_shift.c silk_sum_sqr_shift: compute the number of
// bits (shift) to right-shift the sum of squares of the int16 vector x[0:length] so
// it fits in an int32 with two bits of headroom, and return that shifted energy. The
// first pass estimates the shift with the maximum possible headroom; the second pass
// recomputes the energy at the tightened shift.
func silkSumSqrShift(x []int16, length int) (energy int32, shift int) {
	var i int
	var nrgTmp uint32
	var nrg int32

	/* Do a first run with the maximum shift we could have. */
	shft := 31 - int(silkmath.Silk_CLZ32(int32(length)))
	/* Let's be conservative with rounding and start with nrg=len. */
	nrg = int32(length)
	for i = 0; i < length-1; i += 2 {
		nrgTmp = uint32(silkmath.Silk_SMULBB(int32(x[i]), int32(x[i])))
		nrgTmp = uint32(silkmath.Silk_SMLABB_ovflw(int32(nrgTmp), int32(x[i+1]), int32(x[i+1])))
		nrg = int32(silkmath.Silk_ADD_RSHIFT_uint(uint32(nrg), nrgTmp, shft))
	}
	if i < length {
		/* One sample left to process */
		nrgTmp = uint32(silkmath.Silk_SMULBB(int32(x[i]), int32(x[i])))
		nrg = int32(silkmath.Silk_ADD_RSHIFT_uint(uint32(nrg), nrgTmp, shft))
	}
	/* silk_assert( nrg >= 0 ); */

	/* Make sure the result will fit in a 32-bit signed integer with two bits
	   of headroom. */
	shft = int(silkmath.Silk_max_32(0, int32(shft)+3-silkmath.Silk_CLZ32(nrg)))
	nrg = 0
	for i = 0; i < length-1; i += 2 {
		nrgTmp = uint32(silkmath.Silk_SMULBB(int32(x[i]), int32(x[i])))
		nrgTmp = uint32(silkmath.Silk_SMLABB_ovflw(int32(nrgTmp), int32(x[i+1]), int32(x[i+1])))
		nrg = int32(silkmath.Silk_ADD_RSHIFT_uint(uint32(nrg), nrgTmp, shft))
	}
	if i < length {
		/* One sample left to process */
		nrgTmp = uint32(silkmath.Silk_SMULBB(int32(x[i]), int32(x[i])))
		nrg = int32(silkmath.Silk_ADD_RSHIFT_uint(uint32(nrg), nrgTmp, shft))
	}
	/* silk_assert( nrg >= 0 ); */

	/* Output arguments */
	return nrg, shft
}
