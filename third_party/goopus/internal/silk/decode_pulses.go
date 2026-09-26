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

// Transliteration of silk/decode_pulses.c (libopus v1.6.1) for the frozen
// FIXED_POINT + DISABLE_FLOAT_API build: silk_decode_pulses, the entry point that
// decodes the quantization indices of the SILK excitation. It reads the rate
// level, the sum-of-pulses per shell block (with the LSB-shift escape), runs the
// shell decoder per block, decodes the low-order bits, and applies the signs.
//
// Names follow the C for diffability; the consumed tables (silkRateLevelsICDF,
// silkPulsesPerBlockICDF, silkLSBICDF, ...) live in tables_gen.go, the shell tree
// in shell_coder.go and the sign decode in code_signs.go.

package silk

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"
)

// Constants from silk/define.h.
const (
	shellCodecFrameLength     = 16 // SHELL_CODEC_FRAME_LENGTH
	log2ShellCodecFrameLength = 4  // LOG2_SHELL_CODEC_FRAME_LENGTH
	maxNbShellBlocks          = 20 // MAX_NB_SHELL_BLOCKS = MAX_FRAME_LENGTH(320) / SHELL_CODEC_FRAME_LENGTH
	nRateLevels               = 10 // N_RATE_LEVELS
	silkMaxPulses             = 16 // SILK_MAX_PULSES
)

// DecodePulses is silk/decode_pulses.c silk_decode_pulses: decode the
// quantization indices of the excitation, writing the pulse amplitudes into
// pulses. The caller must supply a pulses slice with room for iter *
// SHELL_CODEC_FRAME_LENGTH samples, where iter = ceil(frameLength /
// SHELL_CODEC_FRAME_LENGTH) (the 10 ms @ 12 kHz frame_length of 120 rounds up to
// 128, one extra shell block).
func DecodePulses(psRangeDec *rangecoding.Decoder, pulses []int16, signalType, quantOffsetType, frameLength int) {
	var sumPulses [maxNbShellBlocks]int
	var nLshifts [maxNbShellBlocks]int

	/*********************/
	/* Decode rate level */
	/*********************/
	rateLevelIndex := psRangeDec.DecIcdf(silkRateLevelsICDF[signalType>>1][:], 8)

	/* Calculate number of shell blocks */
	// silk_assert( 1 << LOG2_SHELL_CODEC_FRAME_LENGTH == SHELL_CODEC_FRAME_LENGTH )
	iter := int(silkmath.Silk_RSHIFT(int32(frameLength), log2ShellCodecFrameLength))
	if iter*shellCodecFrameLength < frameLength {
		// celt_assert( frame_length == 12 * 10 ) -- only happens for 10 ms @ 12 kHz
		iter++
	}

	/***************************************************/
	/* Sum-Weighted-Pulses Decoding                    */
	/***************************************************/
	cdfPtr := silkPulsesPerBlockICDF[rateLevelIndex][:]
	for i := 0; i < iter; i++ {
		nLshifts[i] = 0
		sumPulses[i] = psRangeDec.DecIcdf(cdfPtr, 8)

		/* LSB indication */
		for sumPulses[i] == silkMaxPulses+1 {
			nLshifts[i]++
			/* When we've already got 10 LSBs, we shift the table to not allow (SILK_MAX_PULSES + 1) */
			extra := 0
			if nLshifts[i] == 10 {
				extra = 1
			}
			sumPulses[i] = psRangeDec.DecIcdf(silkPulsesPerBlockICDF[nRateLevels-1][extra:], 8)
		}
	}

	/***************************************************/
	/* Shell decoding                                  */
	/***************************************************/
	for i := 0; i < iter; i++ {
		off := int(silkmath.Silk_SMULBB(int32(i), shellCodecFrameLength))
		if sumPulses[i] > 0 {
			silkShellDecoder(pulses[off:], psRangeDec, sumPulses[i])
		} else {
			clear(pulses[off : off+shellCodecFrameLength])
		}
	}

	/***************************************************/
	/* LSB Decoding                                    */
	/***************************************************/
	for i := 0; i < iter; i++ {
		if nLshifts[i] > 0 {
			nLS := nLshifts[i]
			pulsesPtr := int(silkmath.Silk_SMULBB(int32(i), shellCodecFrameLength))
			for k := 0; k < shellCodecFrameLength; k++ {
				absQ := int32(pulses[pulsesPtr+k])
				for j := 0; j < nLS; j++ {
					absQ = silkmath.Silk_LSHIFT(absQ, 1)
					absQ += int32(psRangeDec.DecIcdf(silkLSBICDF, 8))
				}
				pulses[pulsesPtr+k] = int16(absQ)
			}
			/* Mark the number of pulses non-zero for sign decoding. */
			sumPulses[i] |= nLS << 5
		}
	}

	/****************************************/
	/* Decode and add signs to pulse signal */
	/****************************************/
	silkDecodeSigns(psRangeDec, pulses, frameLength, signalType, quantOffsetType, sumPulses[:])
}
