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

// Transliteration of silk/stereo_MS_to_LR.c (libopus v1.6.1): silk_stereo_MS_to_LR
// converts the adaptive mid/side representation the SILK decoder reconstructs into the
// left/right stereo signal (dec_API.c:371, after per-channel synthesis, before the
// resampler). It interpolates the two prediction weights from the previous frame's
// values over the first STEREO_INTERP_LEN_MS (8 ms) of the frame, adds the predicted
// side contribution (a 3-tap low-pass of mid plus a scaled mid tap), then forms L = mid
// + side and R = mid - side. The one-sample mid/side delay (StereoDecState.SMid/SSide)
// makes the low-pass and the output continuous across the frame boundary. Names, the
// fixed-point macros and comments follow the C for diffability.
//
// Buffer layout (matches the C): x1 and x2 are frameLength+2 int16 buffers. On entry
// x1[2:frameLength+2] / x2[2:frameLength+2] hold this frame's mid / side samples; the
// function overwrites x1[0:2] / x2[0:2] with the carried delay from state and, on exit,
// x1[1:frameLength+1] / x2[1:frameLength+1] hold the left / right output.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// StereoMStoLR is silk/stereo_MS_to_LR.c silk_stereo_MS_to_LR. state carries the
// cross-frame prediction and delay memory; x1 (mid, becomes left) and x2 (side, becomes
// right) are the frameLength+2 sample buffers described above; predQ13 are this frame's
// decoded predictors; fsKHz is the internal sample rate (8/12/16) and frameLength the
// per-channel sample count.
func StereoMStoLR(state *StereoDecState, x1, x2 []int16, predQ13 *[2]int32, fsKHz, frameLength int) {
	/* Buffering */
	x1[0] = state.SMid[0]
	x1[1] = state.SMid[1]
	x2[0] = state.SSide[0]
	x2[1] = state.SSide[1]
	state.SMid[0] = x1[frameLength]
	state.SMid[1] = x1[frameLength+1]
	state.SSide[0] = x2[frameLength]
	state.SSide[1] = x2[frameLength+1]

	/* Interpolate predictors and add prediction to side channel */
	pred0Q13 := int32(state.PredPrevQ13[0])
	pred1Q13 := int32(state.PredPrevQ13[1])
	denomQ16 := silkmath.Silk_DIV32_16(int32(1)<<16, int32(stereoInterpLenMS*fsKHz))
	delta0Q13 := silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_SMULBB(predQ13[0]-int32(state.PredPrevQ13[0]), denomQ16), 16)
	delta1Q13 := silkmath.Silk_RSHIFT_ROUND(silkmath.Silk_SMULBB(predQ13[1]-int32(state.PredPrevQ13[1]), denomQ16), 16)
	interpLen := stereoInterpLenMS * fsKHz
	for n := 0; n < interpLen; n++ {
		pred0Q13 += delta0Q13
		pred1Q13 += delta1Q13
		sum := silkmath.Silk_LSHIFT(silkmath.Silk_ADD_LSHIFT32(int32(x1[n])+int32(x1[n+2]), int32(x1[n+1]), 1), 9) /* Q11 */
		sum = silkmath.Silk_SMLAWB(silkmath.Silk_LSHIFT(int32(x2[n+1]), 8), sum, pred0Q13)                         /* Q8 */
		sum = silkmath.Silk_SMLAWB(sum, silkmath.Silk_LSHIFT(int32(x1[n+1]), 11), pred1Q13)                        /* Q8 */
		x2[n+1] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(sum, 8)))
	}
	pred0Q13 = predQ13[0]
	pred1Q13 = predQ13[1]
	for n := interpLen; n < frameLength; n++ {
		sum := silkmath.Silk_LSHIFT(silkmath.Silk_ADD_LSHIFT32(int32(x1[n])+int32(x1[n+2]), int32(x1[n+1]), 1), 9) /* Q11 */
		sum = silkmath.Silk_SMLAWB(silkmath.Silk_LSHIFT(int32(x2[n+1]), 8), sum, pred0Q13)                         /* Q8 */
		sum = silkmath.Silk_SMLAWB(sum, silkmath.Silk_LSHIFT(int32(x1[n+1]), 11), pred1Q13)                        /* Q8 */
		x2[n+1] = int16(silkmath.Silk_SAT16(silkmath.Silk_RSHIFT_ROUND(sum, 8)))
	}
	state.PredPrevQ13[0] = int16(predQ13[0])
	state.PredPrevQ13[1] = int16(predQ13[1])

	/* Convert to left/right signals */
	for n := 0; n < frameLength; n++ {
		sum := int32(x1[n+1]) + int32(x2[n+1])
		diff := int32(x1[n+1]) - int32(x2[n+1])
		x1[n+1] = int16(silkmath.Silk_SAT16(sum))
		x2[n+1] = int16(silkmath.Silk_SAT16(diff))
	}
}
