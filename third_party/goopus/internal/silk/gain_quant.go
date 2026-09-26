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

// Transliteration of the decode-side gain dequantizer from silk/gain_quant.c
// (libopus v1.6.1): silk_gains_dequant reconstructs the linear per-subframe gains
// (Q16) from the decoded gain indices, carrying the previous quantized index
// (psDec.LastGainIndex) across subframes and frames. The encode-side
// silk_gains_quant / silk_gains_ID are not part of the decode path and are omitted.
// Names and control flow follow the C for diffability; fixed-point math imports
// internal/silkmath.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// Constants from silk/define.h consumed by the gain (de)quantizer.
const (
	nLevelsQGain      = 64 // N_LEVELS_QGAIN
	maxDeltaGainQuant = 36 // MAX_DELTA_GAIN_QUANT
	minDeltaGainQuant = -4 // MIN_DELTA_GAIN_QUANT
	minQGainDB        = 2  // MIN_QGAIN_DB
	maxQGainDB        = 88 // MAX_QGAIN_DB
)

// OFFSET / INV_SCALE_Q16 from silk/gain_quant.c, evaluated with integer division
// exactly as the C preprocessor does (all operands are integer literals):
//
//	OFFSET        = ( MIN_QGAIN_DB * 128 ) / 6 + 16 * 128
//	INV_SCALE_Q16 = ( 65536 * ( ( ( MAX_QGAIN_DB - MIN_QGAIN_DB ) * 128 ) / 6 ) ) / ( N_LEVELS_QGAIN - 1 )
//
// Go integer constant division truncates toward zero like C for these positive
// operands, so gainOFFSET == 2090 and gainINVSCALEQ16 == 1907825 as in the oracle.
const (
	gainOFFSET      = (minQGainDB*128)/6 + 16*128
	gainINVSCALEQ16 = (65536 * (((maxQGainDB - minQGainDB) * 128) / 6)) / (nLevelsQGain - 1)
)

// silkGainsDequant is silk/gain_quant.c silk_gains_dequant: gains scalar
// dequantization, uniform on a log scale. gainQ16 receives the linear gains (Q16);
// ind are the decoded gain indices; prevInd is the last quantized index carried in
// and out (psDec.LastGainIndex); conditional is 1 when the first gain is delta
// coded (condCoding == CODE_CONDITIONALLY); nbSubfr is the subframe count.
func silkGainsDequant(gainQ16 []int32, ind []int8, prevInd *int8, conditional, nbSubfr int) {
	var k, indTmp, doubleStepSizeThreshold int

	for k = 0; k < nbSubfr; k++ {
		if k == 0 && conditional == 0 {
			/* Gain index is not allowed to go down more than 16 steps (~21.8 dB) */
			*prevInd = int8(silkmath.Silk_max_int(int(ind[k]), int(*prevInd)-16))
		} else {
			/* Delta index */
			indTmp = int(ind[k]) + minDeltaGainQuant

			/* Accumulate deltas */
			doubleStepSizeThreshold = 2*maxDeltaGainQuant - nLevelsQGain + int(*prevInd)
			if indTmp > doubleStepSizeThreshold {
				*prevInd = int8(int(*prevInd) + int(silkmath.Silk_LSHIFT(int32(indTmp), 1)) - doubleStepSizeThreshold)
			} else {
				*prevInd = int8(int(*prevInd) + indTmp)
			}
		}
		*prevInd = int8(silkmath.Silk_LIMIT_int(int(*prevInd), 0, nLevelsQGain-1))

		/* Scale and convert to linear scale */
		gainQ16[k] = silkmath.Silk_log2lin(silkmath.Silk_min_32(silkmath.Silk_SMULWB(gainINVSCALEQ16, int32(*prevInd))+gainOFFSET, 3967)) /* 3967 = 31 in Q7 */
	}
}
