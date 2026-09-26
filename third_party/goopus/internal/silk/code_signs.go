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

// Transliteration of the DECODE side of silk/code_signs.c (libopus v1.6.1) for
// the frozen FIXED_POINT + DISABLE_FLOAT_API build: silk_decode_signs reads one
// range-coded sign bit for each nonzero pulse and attaches it to the (positive)
// pulse magnitudes produced by the shell decoder.
//
// The encode side (silk_enc_map / silk_encode_signs) is a phase-5 encoder
// concern and is omitted here. Names follow the C for diffability; silkSignICDF
// lives in tables_gen.go.

package silk

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"
)

// silkDecMap is the code_signs.c silk_dec_map macro: silk_LSHIFT(a,1) - 1 maps a
// decoded sign symbol {0,1} to {-1,+1} without a branch.
func silkDecMap(a int) int { return (a << 1) - 1 }

// silkDecodeSigns is silk/code_signs.c silk_decode_signs: for every shell block
// with a nonzero pulse sum it reads one sign bit per positive pulse from the
// range decoder against the silk_sign_iCDF row selected by (signalType,
// quantOffsetType) and the block's pulse count, negating the magnitude in place
// when the decoded sign is 0. pulses is the in/out excitation, length is the
// frame length in samples, sumPulses holds the per-block pulse sums (with the LSB
// shift count OR'd into bits above bit 5, mirroring silk_decode_pulses).
func silkDecodeSigns(psRangeDec *rangecoding.Decoder, pulses []int16, length, signalType, quantOffsetType int, sumPulses []int) {
	var icdf [2]byte

	icdf[1] = 0
	qPtr := 0 // index into pulses, mirrors the moving q_ptr
	i := int(silkmath.Silk_SMULBB(7, silkmath.Silk_ADD_LSHIFT(int32(quantOffsetType), int32(signalType), 1)))
	icdfPtr := silkSignICDF[i:]
	length = int(silkmath.Silk_RSHIFT(int32(length+shellCodecFrameLength/2), log2ShellCodecFrameLength))
	for i = 0; i < length; i++ {
		p := sumPulses[i]
		if p > 0 {
			icdf[0] = icdfPtr[silkmath.Silk_min_int(p&0x1F, 6)]
			for j := 0; j < shellCodecFrameLength; j++ {
				if pulses[qPtr+j] > 0 {
					// attach sign (implementation with shift, subtraction, multiplication)
					pulses[qPtr+j] *= int16(silkDecMap(psRangeDec.DecIcdf(icdf[:], 8)))
				}
			}
		}
		qPtr += shellCodecFrameLength
	}
}
