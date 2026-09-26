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

package silk

// SILK sample-rate converter, transliterated from silk/resampler.c (libopus
// v1.6.1): silk_resampler_init sets up the state (delay compensation, filter
// selection, Fs ratio) and silk_resampler is the top-level dispatch. The decoder
// converts the SILK internal rate (8/12/16 kHz) to the API output rate
// (8/12/16/24/48 kHz), so ResamplerInit/Resampler cover, per the C method matrix:
//
//	                              Fs_out (kHz)
//	                     8      12     16     24     48
//	           8         C      UF     U      UF     UF
//	          12         AF     C      UF     U      UF
//	Fs_in     16         D      AF     C      UF     UF
//
//	C   -> Copy (no resampling)          (ResamplerFunction = copy)
//	D   -> Allpass-based 2x downsampling  }
//	AF  -> AR2 filter followed by FIR     } down_FIR
//	U   -> Allpass-based 2x upsampling    (up2_HQ_wrapper)
//	UF  -> 2x upsampling followed by FIR  (IIR_FIR)
//
// Every decoder cell above is reached by ResamplerInit(forEnc=0), including the
// three down-sampling cells (16->8 = 1:2, 12->8 = 2:3, 16->12 = 3:4) that dispatch
// to down_FIR; those are ported here alongside the copy/up/fractional-up paths so
// the whole decoder-path matrix is bit-exact. The encoder-rate input cells (Fs_in
// 24/48 kHz) and the QEXT-only 96 kHz rates are out of scope: ResamplerInit still
// carries the forEnc=1 input checks and delay table for diffability, but the
// separate silk_resampler_down2 / silk_resampler_down2_3 helpers (not reached by
// silk_resampler_init) are not ported.
//
// The persistent filter memory (SIIR, SFIR, DelayBuf) carries across calls, so a
// decoder that resamples must keep one ResamplerState per channel and feed frames
// through it in order (docs/hard-parts.md 5).
//
// Field, function and comment names follow the C for diffability. Fixed-point
// operations route through internal/silkmath (docs/hard-parts.md 4).

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// Resampler state sizing (silk/resampler_structs.h).
const (
	silkResamplerMaxFIROrder = 36 // SILK_RESAMPLER_MAX_FIR_ORDER
	silkResamplerMaxIIROrder = 6  // SILK_RESAMPLER_MAX_IIR_ORDER
)

// Inner-loop batch size (silk/resampler_private.h). RESAMPLER_MAX_FS_KHZ and
// RESAMPLER_MAX_BATCH_SIZE_IN are compile-time only in C and not needed here.
const resamplerMaxBatchSizeMS = 10 // RESAMPLER_MAX_BATCH_SIZE_MS

// Resampler function selectors (silk/resampler.c). USE_silk_resampler_private_down_FIR
// is reached on the decoder path for the down-sampling rate pairs.
const (
	useSilkResamplerCopy                = 0 // USE_silk_resampler_copy
	useSilkResamplerPrivateUp2HQWrapper = 1 // USE_silk_resampler_private_up2_HQ_wrapper
	useSilkResamplerPrivateIIRFIR       = 2 // USE_silk_resampler_private_IIR_FIR
	useSilkResamplerPrivateDownFIR      = 3 // USE_silk_resampler_private_down_FIR
)

// Tables with delay compensation values to equalize total delay for different
// modes (silk/resampler.c). delay_matrix_enc is indexed [rateID(in)][rateID(out)]
// on the encoder path; delay_matrix_dec on the decoder path.
var (
	// delay_matrix_enc[ 6 ][ 3 ]   in \ out  8  12  16
	delayMatrixEnc = [6][3]int8{
		{6, 0, 3},    //  8
		{0, 7, 3},    // 12
		{0, 1, 10},   // 16
		{0, 2, 6},    // 24
		{18, 10, 12}, // 48
		{0, 0, 44},   // 96
	}
	// delay_matrix_dec[ 3 ][ 6 ]   in \ out  8  12  16  24  48  96
	delayMatrixDec = [3][6]int8{
		{4, 0, 2, 0, 0, 0},  //  8
		{0, 9, 4, 7, 4, 4},  // 12
		{0, 3, 12, 7, 7, 7}, // 16
	}
)

// rateID maps a sampling rate in Hz to a small index (silk/resampler.c):
// [8000, 12000, 16000, 24000, 48000] -> [0, 1, 2, 3, 4].
//
//	#define rateID(R) IMIN(5, ( ( ( ((R)>>12) - ((R)>16000) ) >> ((R)>24000) ) - 1 ))
func rateID(R int32) int {
	var gt16000 int32
	if R > 16000 {
		gt16000 = 1
	}
	var gt24000 int32
	if R > 24000 {
		gt24000 = 1
	}
	v := (((R >> 12) - gt16000) >> gt24000) - 1
	if v > 5 {
		v = 5
	}
	return int(v)
}

// ResamplerState mirrors silk/resampler_structs.h silk_resampler_state_struct: the
// persistent resampler filter memory and configuration. Field names follow the C.
//
// The C union { opus_int32 i32[36]; opus_int16 i16[36]; } sFIR is represented by the
// single SFIR array. Go has no unions and a given resampler instance uses exactly one
// view: the IIR_FIR path stores its 8 int16 filter-memory samples sign-extended into
// SFIR[0:8]; the down_FIR path stores its FIR_Order int32 samples in SFIR. Either way
// the stored values equal the corresponding C interpretation, so SFIR is directly
// state-hash comparable against the C union read back through the same view.
type ResamplerState struct {
	SIIR              [silkResamplerMaxIIROrder]int32 // sIIR (must precede sFIR in the C struct; irrelevant here)
	SFIR              [silkResamplerMaxFIROrder]int32 // sFIR (union i32/i16, see above)
	DelayBuf          [96]int16                       // delayBuf
	ResamplerFunction int                             // resampler_function
	BatchSize         int                             // batchSize
	InvRatioQ16       int32                           // invRatio_Q16
	FIROrder          int                             // FIR_Order
	FIRFracs          int                             // FIR_Fracs
	FsInKHz           int                             // Fs_in_kHz
	FsOutKHz          int                             // Fs_out_kHz
	InputDelay        int                             // inputDelay
	Coefs             []int16                         // Coefs (slice into a ROM COEFS table; nil unless down_FIR)
}

// ResamplerInit initializes/resets the resampler state for a given pair of
// input/output sampling rates (silk/resampler.c silk_resampler_init). forEnc selects
// the encoder (1) or decoder (0) rate constraints and delay table. Returns 0 on
// success, -1 on an unsupported rate pair.
func ResamplerInit(S *ResamplerState, fsHzIn, fsHzOut int32, forEnc int) int {
	var up2x int

	/* Clear state */
	*S = ResamplerState{}

	/* Input checking */
	if forEnc != 0 {
		if (fsHzIn != 8000 && fsHzIn != 12000 && fsHzIn != 16000 && fsHzIn != 24000 && fsHzIn != 48000) ||
			(fsHzOut != 8000 && fsHzOut != 12000 && fsHzOut != 16000) {
			return -1
		}
		S.InputDelay = int(delayMatrixEnc[rateID(fsHzIn)][rateID(fsHzOut)])
	} else {
		if (fsHzIn != 8000 && fsHzIn != 12000 && fsHzIn != 16000) ||
			(fsHzOut != 8000 && fsHzOut != 12000 && fsHzOut != 16000 && fsHzOut != 24000 && fsHzOut != 48000) {
			return -1
		}
		S.InputDelay = int(delayMatrixDec[rateID(fsHzIn)][rateID(fsHzOut)])
	}

	S.FsInKHz = int(silkmath.Silk_DIV32_16(fsHzIn, 1000))
	S.FsOutKHz = int(silkmath.Silk_DIV32_16(fsHzOut, 1000))

	/* Number of samples processed per batch */
	S.BatchSize = S.FsInKHz * resamplerMaxBatchSizeMS

	/* Find resampler with the right sampling ratio */
	up2x = 0
	if fsHzOut > fsHzIn {
		/* Upsample */
		if fsHzOut == silkmath.Silk_MUL(fsHzIn, 2) { /* Fs_out : Fs_in = 2 : 1 */
			/* Special case: directly use 2x upsampler */
			S.ResamplerFunction = useSilkResamplerPrivateUp2HQWrapper
		} else {
			/* Default resampler */
			S.ResamplerFunction = useSilkResamplerPrivateIIRFIR
			up2x = 1
		}
	} else if fsHzOut < fsHzIn {
		/* Downsample */
		S.ResamplerFunction = useSilkResamplerPrivateDownFIR
		switch {
		case silkmath.Silk_MUL(fsHzOut, 4) == silkmath.Silk_MUL(fsHzIn, 3): /* Fs_out : Fs_in = 3 : 4 */
			S.FIRFracs = 3
			S.FIROrder = resamplerDownOrderFIR0
			S.Coefs = silkResampler34COEFS
		case silkmath.Silk_MUL(fsHzOut, 3) == silkmath.Silk_MUL(fsHzIn, 2): /* Fs_out : Fs_in = 2 : 3 */
			S.FIRFracs = 2
			S.FIROrder = resamplerDownOrderFIR0
			S.Coefs = silkResampler23COEFS
		case silkmath.Silk_MUL(fsHzOut, 2) == fsHzIn: /* Fs_out : Fs_in = 1 : 2 */
			S.FIRFracs = 1
			S.FIROrder = resamplerDownOrderFIR1
			S.Coefs = silkResampler12COEFS
		case silkmath.Silk_MUL(fsHzOut, 3) == fsHzIn: /* Fs_out : Fs_in = 1 : 3 */
			S.FIRFracs = 1
			S.FIROrder = resamplerDownOrderFIR2
			S.Coefs = silkResampler13COEFS
		case silkmath.Silk_MUL(fsHzOut, 4) == fsHzIn: /* Fs_out : Fs_in = 1 : 4 */
			S.FIRFracs = 1
			S.FIROrder = resamplerDownOrderFIR2
			S.Coefs = silkResampler14COEFS
		case silkmath.Silk_MUL(fsHzOut, 6) == fsHzIn: /* Fs_out : Fs_in = 1 : 6 */
			S.FIRFracs = 1
			S.FIROrder = resamplerDownOrderFIR2
			S.Coefs = silkResampler16COEFS
		default:
			/* None available */
			return -1
		}
	} else {
		/* Input and output sampling rates are equal: copy */
		S.ResamplerFunction = useSilkResamplerCopy
	}

	/* Ratio of input/output samples */
	S.InvRatioQ16 = silkmath.Silk_LSHIFT32(silkmath.Silk_DIV32(silkmath.Silk_LSHIFT32(fsHzIn, 14+up2x), fsHzOut), 2)
	/* Make sure the ratio is rounded up */
	for silkmath.Silk_SMULWW(S.InvRatioQ16, fsHzOut) < silkmath.Silk_LSHIFT32(fsHzIn, up2x) {
		S.InvRatioQ16++
	}

	return 0
}

// Resampler converts from one sampling rate to another (silk/resampler.c
// silk_resampler). Input and output sampling rate are at most 48000 Hz. out must be
// sized to hold (inLen / Fs_in_kHz) * Fs_out_kHz samples. Returns 0.
func Resampler(S *ResamplerState, out []int16, in []int16, inLen int32) int {
	var nSamples int32

	/* Need at least 1 ms of input data */
	/* celt_assert( inLen >= S->Fs_in_kHz ) */
	/* Delay can't exceed the 1 ms of buffering */
	/* celt_assert( S->inputDelay <= S->Fs_in_kHz ) */

	nSamples = int32(S.FsInKHz) - int32(S.InputDelay)

	/* Copy to delay buffer */
	copy(S.DelayBuf[S.InputDelay:S.InputDelay+int(nSamples)], in[:nSamples])

	secondLen := int(inLen) - S.FsInKHz
	switch S.ResamplerFunction {
	case useSilkResamplerPrivateUp2HQWrapper:
		silkResamplerPrivateUp2HQWrapper(S, out, S.DelayBuf[:], int32(S.FsInKHz))
		silkResamplerPrivateUp2HQWrapper(S, out[S.FsOutKHz:], in[nSamples:], inLen-int32(S.FsInKHz))
	case useSilkResamplerPrivateIIRFIR:
		silkResamplerPrivateIIRFIR(S, out, S.DelayBuf[:], int32(S.FsInKHz))
		silkResamplerPrivateIIRFIR(S, out[S.FsOutKHz:], in[nSamples:], inLen-int32(S.FsInKHz))
	case useSilkResamplerPrivateDownFIR:
		silkResamplerPrivateDownFIR(S, out, S.DelayBuf[:], int32(S.FsInKHz))
		silkResamplerPrivateDownFIR(S, out[S.FsOutKHz:], in[nSamples:], inLen-int32(S.FsInKHz))
	default:
		copy(out[:S.FsInKHz], S.DelayBuf[:S.FsInKHz])
		copy(out[S.FsOutKHz:S.FsOutKHz+secondLen], in[int(nSamples):int(nSamples)+secondLen])
	}

	/* Copy to delay buffer */
	copy(S.DelayBuf[:S.InputDelay], in[int(inLen)-S.InputDelay:inLen])

	return 0
}
