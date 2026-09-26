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

// Transliteration of silk/bwexpander.c and silk/bwexpander_32.c (libopus v1.6.1)
// for the frozen FIXED_POINT + DISABLE_FLOAT_API build. Chirp (bandwidth expand)
// of an LP AR filter: Bwexpander operates on Q12 int16 coefficients (used after a
// packet loss on the decode path), Bwexpander32 on unscaled int32 coefficients
// (used inside silk_NLSF2A / silk_LPC_fit's stability loop). Names follow the C
// for diffability; all fixed-point math imports internal/silkmath.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// Bwexpander is silk/bwexpander.c silk_bwexpander: chirp (bandwidth expand) an LP
// AR filter of length d given in Q12 int16, in place. chirp_Q16 is typically in
// the range 0 to 1 (Q16). NB: the C deliberately uses silk_RSHIFT_ROUND(
// silk_MUL(), 16 ) instead of silk_SMULWB, because the bias in silk_SMULWB can
// lead to unstable filters.
func Bwexpander(ar []int16, d int, chirp_Q16 int32) {
	var i int
	chirp_minus_one_Q16 := chirp_Q16 - 65536

	for i = 0; i < d-1; i++ {
		ar[i] = int16(silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_MUL(chirp_Q16, int32(ar[i])), 16))
		chirp_Q16 += silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_MUL(chirp_Q16, chirp_minus_one_Q16), 16)
	}
	ar[d-1] = int16(silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_MUL(chirp_Q16, int32(ar[d-1])), 16))
}

// Bwexpander32 is silk/bwexpander_32.c silk_bwexpander_32: chirp (bandwidth
// expand) an LP AR filter of length d given as unscaled int32 coefficients, in
// place. chirp_Q16 is in Q16. This logic is reused in _celt_lpc(); any bug fixes
// should also be applied there.
func Bwexpander32(ar []int32, d int, chirp_Q16 int32) {
	var i int
	chirp_minus_one_Q16 := chirp_Q16 - 65536

	for i = 0; i < d-1; i++ {
		ar[i] = silkmath.Silk_SMULWW(chirp_Q16, ar[i])
		chirp_Q16 += silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_MUL(chirp_Q16, chirp_minus_one_Q16), 16)
	}
	ar[d-1] = silkmath.Silk_SMULWW(chirp_Q16, ar[d-1])
}
