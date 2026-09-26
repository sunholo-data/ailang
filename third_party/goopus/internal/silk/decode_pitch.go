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

// Transliteration of silk/decode_pitch.c (libopus v1.6.1): silk_decode_pitch
// expands the decoded pitch lag index and contour index into the per-subframe
// pitch lags (in samples), clamped to the [min_lag, max_lag] pitch range. The
// contour codebook is selected by internal rate (8 kHz uses the stage-2 tables,
// wideband/mediumband use stage-3) and frame length (nb_subfr). Names follow the C
// for diffability; fixed-point math imports internal/silkmath.

package silk

import "github.com/sunholo-data/ailang/third_party/goopus/internal/silkmath"

// silkDecodePitch is silk/decode_pitch.c silk_decode_pitch. lagIndex and
// contourIndex are the decoded pitch indices; pitchLags receives nb_subfr pitch
// values; FsKHz is the internal sampling rate (8/12/16 kHz); nbSubfr is the
// subframe count (PE_MAX_NB_SUBFR for 20 ms, PE_MAX_NB_SUBFR/2 for 10 ms).
func silkDecodePitch(lagIndex int16, contourIndex int8, pitchLags []int, FsKHz, nbSubfr int) {
	var lag, k, minLag, maxLag int

	// Select the contour codebook (silk_CB_lags_stage2 / _stage3 and their 10 ms
	// variants). matrix_ptr( Lag_CB_ptr, k, contourIndex, cbk_size ) indexes the
	// row-major table as Lag_CB_ptr[ k*cbk_size + contourIndex ]; because cbk_size
	// equals each table's column count, lagCB(k, contourIndex) below is exactly that
	// element on the 2-D generated tables.
	var lagCB func(row, col int) int
	if FsKHz == 8 {
		if nbSubfr == peMaxNBSubfr {
			lagCB = func(row, col int) int { return int(silkCBLagsStage2[row][col]) }
		} else {
			lagCB = func(row, col int) int { return int(silkCBLagsStage210ms[row][col]) }
		}
	} else {
		if nbSubfr == peMaxNBSubfr {
			lagCB = func(row, col int) int { return int(silkCBLagsStage3[row][col]) }
		} else {
			lagCB = func(row, col int) int { return int(silkCBLagsStage310ms[row][col]) }
		}
	}

	minLag = int(silkmath.Silk_SMULBB(peMinLagMS, int32(FsKHz)))
	maxLag = int(silkmath.Silk_SMULBB(peMaxLagMS, int32(FsKHz)))
	lag = minLag + int(lagIndex)

	for k = 0; k < nbSubfr; k++ {
		pitchLags[k] = lag + lagCB(k, int(contourIndex))
		pitchLags[k] = silkmath.Silk_LIMIT_int(pitchLags[k], minLag, maxLag)
	}
}
