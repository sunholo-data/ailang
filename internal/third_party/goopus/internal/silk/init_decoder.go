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

// Transliteration of silk/init_decoder.c (libopus v1.6.1): the per-channel decoder
// state reset/init (silk_reset_decoder / silk_init_decoder), plus the super-struct
// reset/init from silk/dec_API.c (silk_ResetDecoder / silk_InitDecoder) that live in
// the same logical layer. Names and control flow follow the C for diffability.
//
// In the frozen FIXED_POINT + DISABLE_FLOAT_API build with ENABLE_OSCE / ENABLE_DRED /
// ENABLE_DEEP_PLC off, the OSCE header that precedes SILK_DECODER_STATE_RESET_START in
// silk_decoder_state compiles away, so the reset region is the entire DecoderState and
// "clear the entire state, except anything copied" becomes a full struct zeroing.

package silk

// resetDecoder is silk/init_decoder.c silk_reset_decoder: clear the persistent
// per-channel state, then set the two non-zero init fields (first_frame_after_reset,
// prev_gain_Q16) and reset the CNG and PLC sub-states. arch/OSCE are out of scope.
func (psDec *DecoderState) resetDecoder() {
	/* Clear the entire decoder state, except anything copied (nothing, in this build) */
	*psDec = DecoderState{}

	/* Used to deactivate LSF interpolation */
	psDec.FirstFrameAfterReset = 1
	psDec.PrevGainQ16 = 65536

	/* Reset CNG state */
	CNGReset(psDec)

	/* Reset PLC state */
	PLCReset(psDec)
}

// initDecoder is silk/init_decoder.c silk_init_decoder: zero the whole per-channel
// state, then run the reset. The extra zeroing is redundant with resetDecoder in this
// build (the reset region already covers the whole struct) but is kept for structure.
func (psDec *DecoderState) initDecoder() {
	*psDec = DecoderState{}
	psDec.resetDecoder()
}

// ResetDecoder is silk/dec_API.c silk_ResetDecoder: reset every channel state and clear
// the stereo un-mixing state and the prev_decode_only_middle flag.
func (psDec *Decoder) ResetDecoder() {
	for n := 0; n < decoderNumChannels; n++ {
		psDec.ChannelState[n].resetDecoder()
	}
	psDec.SStereo = StereoDecState{}
	/* Not strictly needed, but it's cleaner that way */
	psDec.PrevDecodeOnlyMiddle = 0
}

// InitDecoder is silk/dec_API.c silk_InitDecoder: initialize every channel state and
// clear the stereo un-mixing state and the prev_decode_only_middle flag.
func (psDec *Decoder) InitDecoder() {
	for n := 0; n < decoderNumChannels; n++ {
		psDec.ChannelState[n].initDecoder()
	}
	psDec.SStereo = StereoDecState{}
	/* Not strictly needed, but it's cleaner that way */
	psDec.PrevDecodeOnlyMiddle = 0
}
