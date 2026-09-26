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

// Transliteration of the DECODE side of silk/shell_coder.c (libopus v1.6.1) for
// the frozen FIXED_POINT + DISABLE_FLOAT_API build: the SILK shell decoder that
// turns a per-block sum of pulses into the SHELL_CODEC_FRAME_LENGTH individual
// pulse magnitudes via a binary split tree coded against silk_shell_code_table0..3.
//
// The encode side (combine_pulses / encode_split / silk_shell_encoder) is a
// phase-5 encoder concern and is omitted here; only decode_split and
// silk_shell_decoder are ported. Names follow the C for diffability; the leaf
// tables (silkShellCodeTable0..3, silkShellCodeTableOffsets) live in tables_gen.go.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"

// decodeSplit is silk/shell_coder.c decode_split: split the current subframe
// pulse amplitude p into its two child amplitudes. The first child is read from
// the range decoder against shell_table offset by silk_shell_code_table_offsets[p];
// the second is p - child1. For p <= 0 both children are 0.
func decodeSplit(pChild1, pChild2 *int16, psRangeDec *rangecoding.Decoder, p int, shellTable []byte) {
	if p > 0 {
		*pChild1 = int16(psRangeDec.DecIcdf(shellTable[silkShellCodeTableOffsets[p]:], 8))
		*pChild2 = int16(p) - *pChild1
	} else {
		*pChild1 = 0
		*pChild2 = 0
	}
}

// silkShellDecoder is silk/shell_coder.c silk_shell_decoder: it operates on one
// shell code frame of SHELL_CODEC_FRAME_LENGTH (16) pulses, splitting the total
// pulse count pulses4 down the tree pulses4 -> pulses3 -> pulses2 -> pulses1 ->
// pulses0 (the output). pulses0 must have at least SHELL_CODEC_FRAME_LENGTH
// elements.
func silkShellDecoder(pulses0 []int16, psRangeDec *rangecoding.Decoder, pulses4 int) {
	var pulses3 [2]int16
	var pulses2 [4]int16
	var pulses1 [8]int16

	// this function operates on one shell code frame of 16 pulses
	// silk_assert( SHELL_CODEC_FRAME_LENGTH == 16 )

	decodeSplit(&pulses3[0], &pulses3[1], psRangeDec, pulses4, silkShellCodeTable3)

	decodeSplit(&pulses2[0], &pulses2[1], psRangeDec, int(pulses3[0]), silkShellCodeTable2)

	decodeSplit(&pulses1[0], &pulses1[1], psRangeDec, int(pulses2[0]), silkShellCodeTable1)
	decodeSplit(&pulses0[0], &pulses0[1], psRangeDec, int(pulses1[0]), silkShellCodeTable0)
	decodeSplit(&pulses0[2], &pulses0[3], psRangeDec, int(pulses1[1]), silkShellCodeTable0)

	decodeSplit(&pulses1[2], &pulses1[3], psRangeDec, int(pulses2[1]), silkShellCodeTable1)
	decodeSplit(&pulses0[4], &pulses0[5], psRangeDec, int(pulses1[2]), silkShellCodeTable0)
	decodeSplit(&pulses0[6], &pulses0[7], psRangeDec, int(pulses1[3]), silkShellCodeTable0)

	decodeSplit(&pulses2[2], &pulses2[3], psRangeDec, int(pulses3[1]), silkShellCodeTable2)

	decodeSplit(&pulses1[4], &pulses1[5], psRangeDec, int(pulses2[2]), silkShellCodeTable1)
	decodeSplit(&pulses0[8], &pulses0[9], psRangeDec, int(pulses1[4]), silkShellCodeTable0)
	decodeSplit(&pulses0[10], &pulses0[11], psRangeDec, int(pulses1[5]), silkShellCodeTable0)

	decodeSplit(&pulses1[6], &pulses1[7], psRangeDec, int(pulses2[3]), silkShellCodeTable1)
	decodeSplit(&pulses0[12], &pulses0[13], psRangeDec, int(pulses1[6]), silkShellCodeTable0)
	decodeSplit(&pulses0[14], &pulses0[15], psRangeDec, int(pulses1[7]), silkShellCodeTable0)
}
