/* Copyright (c) 2010-2011 Xiph.Org Foundation, Skype Limited
   Written by Jean-Marc Valin and Koen Vos */
/*
   Redistribution and use in source and binary forms, with or without
   modification, are permitted provided that the following conditions
   are met:

   - Redistributions of source code must retain the above copyright
   notice, this list of conditions and the following disclaimer.

   - Redistributions in binary form must reproduce the above copyright
   notice, this list of conditions and the following disclaimer in the
   documentation and/or other materials provided with the distribution.

   THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
   ``AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
   LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
   A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER
   OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
   EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
   PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
   PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
   LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
   NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THE
   SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package opusenc

// rates.go is the PURE-INTEGER rate / decision / framing half of
// opus_encode_native (opus_encoder.c:1182). No PCM enters any function here: every
// value is derived from the encoder configuration, the frame size, the byte budget
// and the previous frame's decisions. That is what makes this half independently
// testable against the C oracle's flat wrappers and its field-level state dump.
//
// The chain, in the order opus_encode_native runs it:
//
//	:1221      max_data_bytes = IMIN(packet_size_cap*6, out_data_bytes)
//	:1224-35   the rejection rules
//	:1325      st->bitrate_bps = user_bitrate_to_bitrate(...)     -> UserBitrateToBitrate
//	:1327-34   the CBR byte-budget derivation                     -> applyCBRBudget
//	:1340-1406 the degenerate TOC-only ("PLC") packet             -> IsDegenerate/EncodeDegenerate
//	:1407      max_rate = bits_to_bitrate(max_data_bytes*8, ...)
//	:1410      equiv_rate (mode unknown)                          -> ComputeEquivRate
//	:1413-24   voice_est                                          -> ComputeVoiceEst
//	:1426-56   the stereo <-> mono stream_channels decision       -> decideStreamChannels
//	:1458      equiv_rate again, now with stream_channels
//	:1465-1539 mode selection (forced here)                       -> selectMode
//	:1571      equiv_rate again, now with the mode
//	:1584-1650 automatic bandwidth selection + the clamps         -> selectBandwidth
//	:1681-85   the CELT mediumband / LFE fixups, curr_bandwidth
//	:2550-51   the TOC byte                                       -> GenTOC
//
// DecideRates below runs :1407 through :1685 as one unit; the caller (the encode
// driver) owns the surrounding PCM work.

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/celt"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/packet"
)

// ---------------------------------------------------------------------------
// celt/celt.h:147-153. The exact integer round-trip between a bit count for one
// frame and a bitrate in bits per second. Both truncate, and they are NOT
// inverses: bits_to_bitrate(bitrate_to_bits(r)) <= r.
//
// These DELEGATE to internal/celt rather than re-deriving the arithmetic. The C
// static inlines live in celt/celt.h and are shared verbatim between
// celt_encode_with_ec's VBR budget and the Opus layer's CBR budget; two copies
// would be two places to get the truncation order wrong.
// ---------------------------------------------------------------------------

// bitsToBitrate is bits_to_bitrate (celt/celt.h:147).
func bitsToBitrate(bits, fs, frameSize int32) int32 {
	return celt.BitsToBitrate(bits, fs, frameSize)
}

// bitrateToBits is bitrate_to_bits (celt/celt.h:151).
func bitrateToBits(bitrate, fs, frameSize int32) int32 {
	return celt.BitrateToBits(bitrate, fs, frameSize)
}

// ---------------------------------------------------------------------------
// Threshold tables (opus_encoder.c:151-174).
// ---------------------------------------------------------------------------

// The bandwidth transition tables (opus_encoder.c:148-174). Each pair is
// {middle (memoryless) threshold, hysteresis}, ordered NB<->MB, MB<->WB,
// WB<->SWB, SWB<->FB, so index 2*(bandwidth-OPUS_BANDWIDTH_MEDIUMBAND) selects the
// threshold for stepping DOWN out of `bandwidth`.
//
// The mono and stereo tables are byte-identical in libopus v1.6.1. They are kept
// as four distinct tables anyway, exactly as the C does, because the selection at
// :1589-1596 is a real branch on (channels == 2 && force_channels != 1) and
// collapsing them would silently hide a future divergence between the two.
var (
	monoVoiceBandwidthThresholds = [8]int32{
		9000, 700, // NB<->MB
		9000, 700, // MB<->WB
		13500, 1000, // WB<->SWB
		14000, 2000, // SWB<->FB
	}
	monoMusicBandwidthThresholds = [8]int32{
		9000, 700, // NB<->MB
		9000, 700, // MB<->WB
		11000, 1000, // WB<->SWB
		12000, 2000, // SWB<->FB
	}
	stereoVoiceBandwidthThresholds = [8]int32{
		9000, 700, // NB<->MB
		9000, 700, // MB<->WB
		13500, 1000, // WB<->SWB
		14000, 2000, // SWB<->FB
	}
	stereoMusicBandwidthThresholds = [8]int32{
		9000, 700, // NB<->MB
		9000, 700, // MB<->WB
		11000, 1000, // WB<->SWB
		12000, 2000, // SWB<->FB
	}
)

// Threshold bit-rates for switching between mono and stereo (opus_encoder.c:166-167).
const (
	stereoVoiceThreshold = 19000
	stereoMusicThreshold = 17000
)

// ---------------------------------------------------------------------------
// gen_toc (opus_encoder.c:330).
// ---------------------------------------------------------------------------

// GenTOC is gen_toc (opus_encoder.c:330): the packet's first byte, encoding the
// mode, the bandwidth, the frame duration and the stereo flag.
//
// framerate is Fs/frame_size, so 50 for 20 ms; the loop turns it into the log2
// "period" the TOC actually carries (20 ms -> 3).
//
// channels IS st->stream_channels AT THE CALL SITE (:2551), NOT st->channels: a
// stereo encoder that decideStreamChannels rate-downmixed to mono emits a MONO
// TOC. Getting this wrong produces a packet the decoder expands to the wrong
// channel count.
//
// The frame-count bits (the low 2 bits) stay 0 (code 0, one frame) on the normal
// path; only the degenerate path at :1401 ORs a packet code in.
func GenTOC(mode, framerate, bandwidth, channels int) byte {
	period := 0
	for framerate < 400 {
		framerate <<= 1
		period++
	}

	var toc byte
	switch mode {
	case ModeSilkOnly:
		toc = byte((bandwidth - BandwidthNarrowband) << 5)
		toc |= byte((period - 2) << 3)
	case ModeCeltOnly:
		tmp := bandwidth - BandwidthMediumband
		if tmp < 0 {
			tmp = 0
		}
		toc = 0x80
		toc |= byte(tmp << 5)
		toc |= byte(period << 3)
	default: // Hybrid
		toc = 0x60
		toc |= byte((bandwidth - BandwidthSuperwideband) << 4)
		toc |= byte((period - 2) << 3)
	}
	if channels == 2 {
		toc |= 1 << 2
	}
	return toc
}

// ---------------------------------------------------------------------------
// user_bitrate_to_bitrate (opus_encoder.c:733).
// ---------------------------------------------------------------------------

// UserBitrateToBitrate is user_bitrate_to_bitrate (opus_encoder.c:733): resolve
// the user's OPUS_SET_BITRATE into an actual bits-per-second figure, then clamp it
// to whatever the caller's output buffer can physically hold at this frame size.
//
// OPUS_AUTO is 60*Fs/frame_size + Fs*channels (a per-packet overhead allowance
// plus 1 bit per sample per channel); OPUS_BITRATE_MAX is a flat 1500000.
//
// It is a free function, not a method, because the C reads only three fields of
// st (Fs, user_bitrate_bps, channels) and the oracle's flat wrapper drives it from
// a scratch encoder with exactly those three set. bitrateFromUser below is the
// method the encode driver calls.
func UserBitrateToBitrate(fs int32, channels int, userBitrateBps int32, frameSize, maxDataBytes int) int32 {
	if frameSize == 0 {
		frameSize = int(fs / 400)
	}
	maxBitrate := bitsToBitrate(int32(maxDataBytes)*8, fs, int32(frameSize))

	var userBitrate int32
	switch userBitrateBps {
	case OpusAuto:
		userBitrate = 60*fs/int32(frameSize) + fs*int32(channels)
	case OpusBitrateMax:
		userBitrate = 1500000
	default:
		userBitrate = userBitrateBps
	}

	if userBitrate < maxBitrate {
		return userBitrate
	}
	return maxBitrate
}

// bitrateFromUser is the :1325 call site: it MUTATES st.bitrateBps, which every
// later decision (equiv_rate, the stereo/mono downmix, the bandwidth tiers) then
// reads.
func (st *Encoder) bitrateFromUser(frameSize, maxDataBytes int) {
	st.bitrateBps = UserBitrateToBitrate(st.Fs, st.channels, st.userBitrateBps, frameSize, maxDataBytes)
}

// ---------------------------------------------------------------------------
// compute_equiv_rate (opus_encoder.c:1027).
// ---------------------------------------------------------------------------

// ComputeEquivRate is compute_equiv_rate (opus_encoder.c:1027): the bitrate an
// equivalent 20 ms / complexity 10 / VBR encoder would need to reach the same
// quality. Every rate-driven decision below (stereo/mono, bandwidth) compares
// against this, never against the raw bitrate, so that a 5 ms CBR complexity-0
// stream is not judged as if it were a 20 ms VBR complexity-10 one.
//
// mode is 0 ("not known yet") on the first two calls (:1410, :1458) and the real
// mode on the third (:1571). The 0 case takes the final else and applies half the
// SILK loss penalty.
//
// All of this is 32-bit integer arithmetic in C, with truncating division; the
// int32 here reproduces that exactly.
func ComputeEquivRate(bitrate int32, channels, frameRate, vbr, mode, complexity, loss int) int32 {
	equiv := bitrate
	// Take into account overhead from smaller frames.
	if frameRate > 50 {
		equiv -= int32((40*channels + 20) * (frameRate - 50))
	}
	// CBR is about a 8% penalty for both SILK and CELT.
	if vbr == 0 {
		equiv -= equiv / 12
	}
	// Complexity makes about 10% difference (from 0 to 10) in general.
	equiv = equiv * int32(90+complexity) / 100

	switch mode {
	case ModeSilkOnly, ModeHybrid:
		// SILK complexity 0-1 uses the non-delayed-decision NSQ, which costs
		// about 20%.
		if complexity < 2 {
			equiv = equiv * 4 / 5
		}
		equiv -= equiv * int32(loss) / int32(6*loss+10)
	case ModeCeltOnly:
		// CELT complexity 0-4 doesn't have the pitch filter, which costs
		// about 10%.
		if complexity < 5 {
			equiv = equiv * 9 / 10
		}
	default:
		// Mode not known yet. Half the SILK loss.
		equiv -= equiv * int32(loss) / int32(12*loss+20)
	}
	return equiv
}

// equivRate is the three :1410 / :1458 / :1571 call sites, which differ only in
// the channel count and the mode they pass.
func (st *Encoder) equivRate(channels, frameSize, mode int) int32 {
	return ComputeEquivRate(st.bitrateBps, channels, int(st.Fs)/frameSize, st.useVbr, mode,
		st.silkMode.complexity, st.silkMode.packetLossPercentage)
}

// ---------------------------------------------------------------------------
// frame_size_select (opus_encoder.c:827).
// ---------------------------------------------------------------------------

// FrameSizeSelect is frame_size_select (opus_encoder.c:827). It returns the frame
// size to actually encode, or -1 to REJECT, which opus_encode_native turns into
// OPUS_BAD_ARG at :1224 (frame_size <= 0).
//
// The rejection rules, in the C's order:
//
//  1. frameSize < Fs/400 (below 2.5 ms) -> -1.
//  2. variableDuration must be OPUS_FRAMESIZE_ARG (use the caller's frame size
//     verbatim) or one of OPUS_FRAMESIZE_2_5_MS..OPUS_FRAMESIZE_120_MS; anything
//     else -> -1. Up to 40 ms the duration enum is a power-of-two multiple of
//     2.5 ms; above it, it is a multiple of 20 ms.
//  3. A ctl-selected size LARGER than the caller's buffer of samples -> -1.
//  4. The result must be exactly one of the nine legal Opus durations
//     (2.5/5/10/20/40/60/80/100/120 ms), expressed as the nine divisibility tests
//     against Fs. At 48 kHz that is 120, 240, 480, 960, 1920, 2880, 3840, 4800,
//     5760 and nothing else, so an arbitrary sample count is rejected here.
//  5. OPUS_APPLICATION_RESTRICTED_SILK cannot do sub-10 ms frames.
func FrameSizeSelect(application int, frameSize int32, variableDuration int, fs int32) int32 {
	var newSize int32

	if frameSize < fs/400 {
		return -1
	}
	switch {
	case variableDuration == FramesizeArg:
		newSize = frameSize
	case variableDuration >= Framesize2_5Ms && variableDuration <= Framesize120Ms:
		if variableDuration <= Framesize40Ms {
			newSize = (fs / 400) << (variableDuration - Framesize2_5Ms)
		} else {
			newSize = int32(variableDuration-Framesize2_5Ms-2) * fs / 50
		}
	default:
		return -1
	}
	if newSize > frameSize {
		return -1
	}
	if 400*newSize != fs && 200*newSize != fs && 100*newSize != fs &&
		50*newSize != fs && 25*newSize != fs && 50*newSize != 3*fs &&
		50*newSize != 4*fs && 50*newSize != 5*fs && 50*newSize != 6*fs {
		return -1
	}
	if application == applicationRestrictedSILK && newSize < fs/100 {
		return -1
	}
	return newSize
}

// applicationRestrictedSILK is OPUS_APPLICATION_RESTRICTED_SILK
// (include/opus_defines.h:224). It is not exported from this package because the
// port does not implement a SILK encoder; frame_size_select still tests against it
// and so must know the value.
const applicationRestrictedSILK = 2052

// ---------------------------------------------------------------------------
// voice_est (opus_encoder.c:1413-1424).
// ---------------------------------------------------------------------------

// ComputeVoiceEst is the voice_est block (opus_encoder.c:1413-1424): the encoder's
// probability that the signal is speech, in Q7 (0 = certainly music, 127 =
// certainly voice). It drives the bandwidth-threshold interpolation and the
// stereo/mono threshold.
//
// IN THE FROZEN CONFIGURATION THIS IS THE CONSTANT 48: DISABLE_FLOAT_API pins
// st->voice_ratio to -1 every frame (:1307-1308), signal_type defaults to
// OPUS_AUTO, and the application is OPUS_APPLICATION_AUDIO, so the chain falls all
// the way through to the final else. The expression is ported in full anyway,
// because OPUS_SET_SIGNAL is a live ctl (voice -> 127, music -> 0) and both of
// those flip real decisions.
//
// voiceRatio*327>>8 is a Q7 rescale of a 0..100 percentage: 100*327>>8 == 127.
func ComputeVoiceEst(signalType, voiceRatio, application int) int {
	switch {
	case signalType == SignalVoice:
		return 127
	case signalType == SignalMusic:
		return 0
	case voiceRatio >= 0:
		voiceEst := voiceRatio * 327 >> 8
		// For AUDIO, never be more than 90% confident of having speech.
		if application == ApplicationAudio && voiceEst > 115 {
			voiceEst = 115
		}
		return voiceEst
	case application == ApplicationVOIP:
		return 115
	default:
		return 48
	}
}

// ---------------------------------------------------------------------------
// The CBR byte-budget derivation (opus_encoder.c:1327-1334).
// ---------------------------------------------------------------------------

// applyCBRBudget is the CBR byte-budget derivation (opus_encoder.c:1327-1334). It
// is a no-op under VBR. It returns the new max_data_bytes and cbr_bytes; -1 for
// cbr_bytes means "VBR, not computed", matching the C initialiser at :1210.
//
//	cbr_bytes = IMIN((bitrate_to_bits(st->bitrate_bps, Fs, frame_size) + 4)/8, max_data_bytes);
//	st->bitrate_bps = bits_to_bitrate(cbr_bytes*8, Fs, frame_size);
//	max_data_bytes = IMAX(1, cbr_bytes);
//
// Three things here are load-bearing and easy to lose:
//
//  1. THE +4 BEFORE THE /8 IS ROUND-TO-NEAREST-BYTE, not a fudge factor: bits are
//     converted to whole bytes with a half-byte bias, so 96 bits -> 12 bytes but
//     100 bits -> 13. A bitrate landing exactly on a multiple of 8 bits per frame
//     sits on that rounding boundary, and one bit either side changes the packet
//     LENGTH.
//  2. st->bitrate_bps IS WRITTEN BACK from the quantized byte count. Every later
//     decision - equiv_rate, the stereo/mono downmix threshold, the bandwidth
//     tiers - therefore sees the RATE THE PACKET WILL ACTUALLY CARRY, not the rate
//     the user asked for. Skipping the write-back leaves the packet size right and
//     the decisions subtly wrong.
//  3. IMAX(1, cbr_bytes) guarantees at least one byte so the coder cannot fail on
//     a zero-length budget; cbr_bytes is only 0 when the requested rate is under
//     ~4 bits per frame, which the degenerate check at :1340 catches immediately
//     afterwards anyway.
//
// The IMIN against max_data_bytes is what caps a CBR request at the caller's
// buffer. Note max_data_bytes was already capped at packet_size_cap*6 == 7656 at
// :1221, so cbr_bytes > 1276 IS reachable (roughly above 510 kbps at 20 ms) and is
// the only way opus_packet_pad at :2646 does real work.
func (st *Encoder) applyCBRBudget(frameSize, maxDataBytes int) (newMaxDataBytes, cbrBytes int) {
	if st.useVbr != 0 {
		return maxDataBytes, -1
	}
	cbrBytes = int((bitrateToBits(st.bitrateBps, st.Fs, int32(frameSize)) + 4) / 8)
	if cbrBytes > maxDataBytes {
		cbrBytes = maxDataBytes
	}
	st.bitrateBps = bitsToBitrate(int32(cbrBytes)*8, st.Fs, int32(frameSize))
	// Make sure we provide at least one byte to avoid failing.
	newMaxDataBytes = cbrBytes
	if newMaxDataBytes < 1 {
		newMaxDataBytes = 1
	}
	return newMaxDataBytes, cbrBytes
}

// ---------------------------------------------------------------------------
// The degenerate TOC-only packet (opus_encoder.c:1340-1406).
// ---------------------------------------------------------------------------

// IsDegenerate is the :1340-1342 test: is there so little room that nothing useful
// can be coded? THIS IS NOT AN ERROR. When it is true the encoder emits a 1- or
// 2-byte TOC-only packet telling the decoder to run its packet-loss concealment,
// and returns a POSITIVE length.
//
//	max_data_bytes < 3
//	|| st->bitrate_bps < 3*frame_rate*8            (under 3 bytes per frame)
//	|| (frame_rate < 50 && (max_data_bytes*frame_rate < 300 || st->bitrate_bps < 2400))
//
// The last clause only applies to frames longer than 20 ms, where a packet can be
// large in bytes while still being starved in bits per second.
//
// It must be called AFTER applyCBRBudget, because both max_data_bytes and
// st.bitrateBps are the post-CBR values.
func (st *Encoder) IsDegenerate(maxDataBytes, frameRate int) bool {
	if maxDataBytes < 3 || st.bitrateBps < int32(3*frameRate*8) {
		return true
	}
	return frameRate < 50 && (maxDataBytes*frameRate < 300 || st.bitrateBps < 2400)
}

// EncodeDegenerate is the degenerate-packet body (opus_encoder.c:1343-1406). It
// writes the packet into data and returns opus_encode_native's return value: a
// POSITIVE byte count, or opusInternalError if the CBR padding fails.
//
// It reads st.mode, st.bandwidth and st.streamChannels as they stand ON ENTRY,
// i.e. the PREVIOUS frame's decisions, because it runs before this frame's
// stereo/mono and bandwidth decisions are made. On the very first frame those are
// the opus_encoder_init values (MODE_HYBRID, FULLBAND, channels), which is why a
// degenerate first frame emits a HYBRID TOC even in a forced-CELT-only encoder.
// It writes NO state: st->first, st->prev_mode and friends are only updated at
// :2556-2562, past the early return.
//
// frameRate is the local, already-computed Fs/frame_size; outDataBytes is the
// caller's ORIGINAL buffer size (not the :1221-capped max_data_bytes), and the C
// reads it at :1370 to decide whether a 1-byte buffer forces a SILK TOC.
func (st *Encoder) EncodeDegenerate(data []byte, outDataBytes, maxDataBytes, frameRate int) int {
	tocmode := st.mode
	bw := st.bandwidth
	if bw == 0 {
		bw = BandwidthNarrowband
	}
	packetCode := 0
	numMultiframes := 0

	if tocmode == 0 {
		tocmode = ModeSilkOnly
	}
	if frameRate > 100 {
		tocmode = ModeCeltOnly
	}
	// 40 ms -> 2 x 20 ms if in CELT_ONLY or HYBRID mode.
	if frameRate == 25 && tocmode != ModeSilkOnly {
		frameRate = 50
		packetCode = 1
	}

	// >= 60 ms frames.
	if frameRate <= 16 {
		// 1 x 60 ms, 2 x 40 ms, 2 x 60 ms.
		if outDataBytes == 1 || (tocmode == ModeSilkOnly && frameRate != 10) {
			tocmode = ModeSilkOnly
			packetCode = 0
			if frameRate <= 12 {
				packetCode = 1
			}
			if frameRate == 12 {
				frameRate = 25
			} else {
				frameRate = 16
			}
		} else {
			numMultiframes = 50 / frameRate
			frameRate = 50
			packetCode = 3
		}
	}

	switch {
	case tocmode == ModeSilkOnly && bw > BandwidthWideband:
		bw = BandwidthWideband
	case tocmode == ModeCeltOnly && bw == BandwidthMediumband:
		bw = BandwidthNarrowband
	case tocmode == ModeHybrid && bw <= BandwidthSuperwideband:
		bw = BandwidthSuperwideband
	}

	data[0] = GenTOC(tocmode, frameRate, bw, st.streamChannels)
	data[0] |= byte(packetCode)

	ret := 1
	if packetCode > 1 {
		ret = 2
	}
	if maxDataBytes < ret {
		maxDataBytes = ret
	}
	if packetCode == 3 {
		data[1] = byte(numMultiframes)
	}

	if st.useVbr == 0 {
		if err := packet.Pad(data, ret, maxDataBytes); err != nil {
			return opusInternalError
		}
		ret = maxDataBytes
	}
	return ret
}

// ---------------------------------------------------------------------------
// The stereo <-> mono stream_channels decision (opus_encoder.c:1426-1456).
// ---------------------------------------------------------------------------

// decideStreamChannels is the stream_channels decision (opus_encoder.c:1426-1456).
// A STEREO ENCODER DOWNMIXES TO MONO when the equivalent rate falls below a
// threshold around 17.3 kbps, which changes BOTH the TOC's stereo bit (GenTOC
// takes stream_channels) and the channel count CELT is run with. It is a per-frame
// decision, so a rate ramp flips it mid-stream.
//
// The threshold interpolates between the music (17000) and voice (19000) values on
// voice_est^2, then applies +-1000 of HYSTERESIS around the CURRENT state:
//
//	stream_channels == 2 -> threshold -= 1000   (stay stereo until well below)
//	stream_channels == 1 -> threshold += 1000   (stay mono until well above)
//
// At the frozen voice_est == 48 that is 17281 nominal, so 16281 to fall out of
// stereo and 18281 to climb back in. The comparison is STRICTLY GREATER: equal to
// the threshold means mono.
//
// OPUS_SET_FORCE_CHANNELS overrides the whole thing, and a mono encoder is always
// mono. The FUZZING branch (:1433-1438) is a debug-only randomiser and is not
// ported.
func (st *Encoder) decideStreamChannels(equivRate int32, voiceEst int) {
	if st.forceChannels != OpusAuto && st.channels == 2 {
		st.streamChannels = st.forceChannels
		return
	}
	// Rate-dependent mono-stereo decision.
	if st.channels == 2 {
		stereoThreshold := int32(stereoMusicThreshold +
			((voiceEst * voiceEst * (stereoVoiceThreshold - stereoMusicThreshold)) >> 14))
		if st.streamChannels == 2 {
			stereoThreshold -= 1000
		} else {
			stereoThreshold += 1000
		}
		if equivRate > stereoThreshold {
			st.streamChannels = 2
		} else {
			st.streamChannels = 1
		}
		return
	}
	st.streamChannels = st.channels
}

// ---------------------------------------------------------------------------
// Mode selection (opus_encoder.c:1465-1539).
// ---------------------------------------------------------------------------

// selectMode is the mode decision (opus_encoder.c:1465-1539) restricted to what
// the frozen configuration can reach. It reports false when the caller has asked
// for the OPUS_AUTO mode decision (:1473-1527), which is DELIBERATELY NOT PORTED:
// it interpolates the mode_thresholds table on stereo_width, and compute_stereo_width
// is omitted here because its only consumer is that block (see the State doc). The
// phase-4 encoder pins OPUS_SET_FORCE_MODE(MODE_CELT_ONLY), which lands on the
// else at :1528-1530.
//
// The two overrides that follow (:1532-1539) ARE ported: a sub-10 ms frame and LFE
// both force CELT-only. Both are no-ops once the mode is already CELT-only, but
// they are the C and they cost nothing.
//
// The redundancy / celt_to_silk / to_celt block at :1541-1557 and the SILK
// stereo->mono delay at :1560-1568 are unreachable without a mode transition
// (prev_mode is 0 on frame 1 and MODE_CELT_ONLY forever after), so they are not
// ported.
func (st *Encoder) selectMode(frameSize int) bool {
	switch {
	case st.application == ApplicationRestrictedLowdelay:
		st.mode = ModeCeltOnly
	case st.userForcedMode == OpusAuto:
		return false
	default:
		st.mode = st.userForcedMode
	}

	// Override the chosen mode to make sure we meet the requested frame size.
	if st.mode != ModeCeltOnly && frameSize < int(st.Fs)/100 {
		st.mode = ModeCeltOnly
	}
	if st.lfe != 0 {
		st.mode = ModeCeltOnly
	}
	return true
}

// ---------------------------------------------------------------------------
// Automatic bandwidth selection (opus_encoder.c:1584-1650).
// ---------------------------------------------------------------------------

// selectBandwidth is the automatic bandwidth selection (opus_encoder.c:1584-1627)
// followed by the clamps (:1629-1650) and the CELT/LFE fixups (:1681-1685). It
// sets st.bandwidth and st.autoBandwidth, and returns curr_bandwidth, which the
// encode driver turns into CELT's `end` band and passes to GenTOC.
//
// The selection walks DOWN from fullband. For each candidate it reads that
// candidate's threshold and hysteresis out of the interpolated table and stops at
// the first band the equivalent rate can afford:
//
//	FB   >= 12000 (music) .. 14000 (voice)
//	SWB  >= 11000 .. 13500
//	WB   >= 9000
//	MB   >= 9000   (immediately rewritten to WB, see below)
//	NB   the floor, reached by falling off the end of the loop
//
// The tables are interpolated on voice_est^2 exactly as the stereo threshold is,
// so a music-flagged stream drops to SWB below 12 kbps while a voice-flagged one
// holds out to 14 kbps.
//
// THE HYSTERESIS (:1606-1613) is asymmetric around the PREVIOUS auto_bandwidth:
// widening the band requires clearing threshold+hysteresis, narrowing it requires
// falling below threshold-hysteresis. st->first suppresses it entirely on the
// first frame, which is why frame 1 can land on a band that frame 2 at the same
// rate would not have moved to.
//
// The do-while at :1601-1616 is what keeps the negative table index unreachable:
// the loop body only ever runs for bandwidth in MB..FB (indices 0..6), and the
// `--bandwidth > NARROWBAND` condition exits with NB before the body could index
// 2*(NB-MB) == -2.
//
// MEDIUMBAND IS NEVER SELECTED (:1618-1620 rewrites it to wideband before it is
// stored), so st->auto_bandwidth can only ever be NB, WB, SWB or FB.
//
// The :1625 guard (no SWB/FB until SILK has settled) and the :1637 hybrid CBR
// guard are ported with their `mode != MODE_CELT_ONLY` conditions intact; both are
// dead in the frozen config. detected_bandwidth (:1655-1672) is DISABLE_FLOAT_API
// and absent. decide_fec (:1674) returns 0 immediately in CELT-only WITHOUT
// touching *bandwidth (:943), so it cannot move the band here.
func (st *Encoder) selectBandwidth(equivRate int32, voiceEst int, maxRate int32) int {
	// The C condition is `mode == MODE_CELT_ONLY || st->first ||
	// st->silk_mode.allowBandwidthSwitch`. allowBandwidthSwitch is written only by
	// the SILK encoder, which never runs here, so it is a constant 0 and omitted.
	if st.mode == ModeCeltOnly || st.first != 0 {
		var voiceThresholds, musicThresholds *[8]int32
		bandwidth := BandwidthFullband

		if st.channels == 2 && st.forceChannels != 1 {
			voiceThresholds = &stereoVoiceBandwidthThresholds
			musicThresholds = &stereoMusicBandwidthThresholds
		} else {
			voiceThresholds = &monoVoiceBandwidthThresholds
			musicThresholds = &monoMusicBandwidthThresholds
		}

		// Interpolate bandwidth thresholds depending on voice estimation.
		var bandwidthThresholds [8]int32
		for i := 0; i < 8; i++ {
			bandwidthThresholds[i] = musicThresholds[i] +
				int32((voiceEst*voiceEst*int(voiceThresholds[i]-musicThresholds[i]))>>14)
		}

		for {
			threshold := bandwidthThresholds[2*(bandwidth-BandwidthMediumband)]
			hysteresis := bandwidthThresholds[2*(bandwidth-BandwidthMediumband)+1]
			if st.first == 0 {
				if st.autoBandwidth >= bandwidth {
					threshold -= hysteresis
				} else {
					threshold += hysteresis
				}
			}
			if equivRate >= threshold {
				break
			}
			bandwidth--
			if bandwidth <= BandwidthNarrowband {
				break // the do-while condition at :1616
			}
		}
		// We don't use mediumband anymore, except when explicitly requested or
		// during mode transitions.
		if bandwidth == BandwidthMediumband {
			bandwidth = BandwidthWideband
		}
		st.bandwidth = bandwidth
		st.autoBandwidth = bandwidth
		// Prevents any transition to SWB/FB until the SILK layer has fully
		// switched to WB mode and turned the variable LP filter off. Dead in
		// CELT-only (inWBmodeWithoutVariableLP is SILK state, constant 0 here, but
		// the mode guard short-circuits first).
		if st.first == 0 && st.mode != ModeCeltOnly && st.bandwidth > BandwidthWideband {
			st.bandwidth = BandwidthWideband
		}
	}

	if st.bandwidth > st.maxBandwidth {
		st.bandwidth = st.maxBandwidth
	}
	if st.userBandwidth != OpusAuto {
		st.bandwidth = st.userBandwidth
	}

	// This prevents us from using hybrid at unsafe CBR/max rates. Dead in
	// CELT-only.
	if st.mode != ModeCeltOnly && maxRate < 15000 && st.bandwidth > BandwidthWideband {
		st.bandwidth = BandwidthWideband
	}

	// Prevents Opus from wasting bits on frequencies that are above the Nyquist
	// rate of the input signal.
	if st.Fs <= 24000 && st.bandwidth > BandwidthSuperwideband {
		st.bandwidth = BandwidthSuperwideband
	}
	if st.Fs <= 16000 && st.bandwidth > BandwidthWideband {
		st.bandwidth = BandwidthWideband
	}
	if st.Fs <= 12000 && st.bandwidth > BandwidthMediumband {
		st.bandwidth = BandwidthMediumband
	}
	if st.Fs <= 8000 && st.bandwidth > BandwidthNarrowband {
		st.bandwidth = BandwidthNarrowband
	}

	// CELT mode doesn't support mediumband, use wideband instead (:1680-1682).
	if st.mode == ModeCeltOnly && st.bandwidth == BandwidthMediumband {
		st.bandwidth = BandwidthWideband
	}
	if st.lfe != 0 {
		st.bandwidth = BandwidthNarrowband
	}
	return st.bandwidth
}

// ---------------------------------------------------------------------------
// The whole chain.
// ---------------------------------------------------------------------------

// RateDecision is everything the pure-integer half of opus_encode_native hands to
// the PCM half.
type RateDecision struct {
	// MaxDataBytes is the byte budget after the :1221 cap and the CBR
	// derivation. The range coder is still initialised over the ORIGINAL buffer
	// (:1964 uses orig_max_data_bytes-1); this is what it is SHRUNK to at :2413.
	MaxDataBytes int
	// CBRBytes is cbr_bytes, or -1 under VBR (the C initialiser at :1210).
	CBRBytes int
	// FrameRate is Fs/frame_size.
	FrameRate int
	// EquivRate is the final equiv_rate (:1571), computed with stream_channels
	// and the settled mode. It feeds the stereo width / stereo_fade decision.
	EquivRate int32
	// MaxRate is :1407.
	MaxRate int32
	// VoiceEst is Q7; 48 in the frozen configuration.
	VoiceEst int
	// CurrBandwidth is curr_bandwidth (:1685): the value passed to GenTOC and
	// turned into CELT's end band.
	CurrBandwidth int
	// Degenerate reports that :1340 fired. When it is set, no other field is
	// meaningful: the caller must emit EncodeDegenerate's packet and return.
	Degenerate bool
}

// DecideRates runs opus_encode_native's pure-integer decision chain, from
// user_bitrate_to_bitrate (:1325) through curr_bandwidth (:1685). It MUTATES
// st.bitrateBps, st.streamChannels, st.mode, st.bandwidth and st.autoBandwidth,
// which is exactly what the C does, and it touches nothing else.
//
// maxDataBytes must already be IMIN(1276*6, out_data_bytes) (:1221), and the
// caller must have applied the :1224-1235 rejection rules first.
//
// When Degenerate is set the caller must call EncodeDegenerate and return its
// value, WITHOUT running any of the epilogue state updates (:2556-2562): the C
// returns from inside the branch.
//
// It returns false only when the configuration needs the unported OPUS_AUTO mode
// decision (see selectMode).
func (st *Encoder) DecideRates(frameSize, maxDataBytes int) (RateDecision, bool) {
	var d RateDecision

	// :1307-1308. Under DISABLE_FLOAT_API the encoder RE-PINS voice_ratio to -1 on
	// every frame, overwriting whatever OPUS_SET_VOICE_RATIO put there. That is what
	// makes voice_est the constant 48 in this build.
	st.voiceRatio = -1

	// :1325. Written back into st, so everything below sees it.
	st.bitrateFromUser(frameSize, maxDataBytes)

	// :1326.
	d.FrameRate = int(st.Fs) / frameSize

	// :1327-1334. Quantizes st.bitrateBps under CBR.
	maxDataBytes, d.CBRBytes = st.applyCBRBudget(frameSize, maxDataBytes)
	d.MaxDataBytes = maxDataBytes

	// :1340. Not an error: a TOC-only PLC packet.
	if st.IsDegenerate(maxDataBytes, d.FrameRate) {
		d.Degenerate = true
		return d, true
	}

	// :1407.
	d.MaxRate = bitsToBitrate(int32(maxDataBytes)*8, st.Fs, int32(frameSize))

	// :1409-1411. NOTE the channel count is st.channels, NOT stream_channels: this
	// equiv_rate is the INPUT to the stereo/mono decision, so it cannot depend on
	// its output. Mode is not known yet, hence the 0.
	equivRate := st.equivRate(st.channels, frameSize, 0)

	// :1413-1424.
	d.VoiceEst = ComputeVoiceEst(st.signalType, st.voiceRatio, st.application)

	// :1426-1456.
	st.decideStreamChannels(equivRate, d.VoiceEst)

	// :1458-1459 IS DELIBERATELY NOT PORTED. The C recomputes equiv_rate here with
	// the freshly decided stream_channels and mode 0, but that value's SOLE consumer
	// is the threshold comparison at :1513 -- and :1513 lives inside the
	// `user_forced_mode == OPUS_AUTO` arm, which selectMode does not port and a
	// forced-CELT-only encoder never enters. compute_equiv_rate is pure, so skipping
	// the call is provably output-neutral; the :1571 recompute below overwrites the
	// variable before anything else reads it.
	//
	// REINSTATE IT the moment the OPUS_AUTO mode decision is ported, or that
	// decision will silently read the stale :1410 value (computed with st.channels,
	// not stream_channels).

	// :1465-1539.
	if !st.selectMode(frameSize) {
		return d, false
	}

	// :1570-1572. Update equivalent rate with the mode decision. THIS is the one
	// that survives: selectBandwidth reads it, and the caller feeds it to the
	// stereo-width / stereo_fade decision at :2320-2349.
	equivRate = st.equivRate(st.streamChannels, frameSize, st.mode)
	d.EquivRate = equivRate

	// :1584-1685.
	d.CurrBandwidth = st.selectBandwidth(equivRate, d.VoiceEst, d.MaxRate)

	return d, true
}

// AdvanceDecisionState is the per-frame epilogue at opus_encoder.c:2555-2562: the
// three "previous frame" fields the next frame's hysteresis reads, plus st->first.
// It runs ONLY on a frame that was actually coded; the degenerate branch returns at
// :1406, before it, so a degenerate frame leaves st->first (and therefore the
// bandwidth hysteresis suppression) exactly as it found them.
//
// to_celt is always 0 here (it needs a SILK->CELT transition, which a forced
// CELT-only encoder can never make), so prev_mode is simply st->mode.
//
// INTEGRATION NOTE: this is the tail of opus_encode_native, not of the rate chain.
// It lives here because the fields it writes are precisely the ones the rate
// decisions read back, and the CP9-RATES differential test needs it to step a
// multi-frame sequence. If the encode driver grows its own epilogue, this must be
// folded into it, not duplicated.
func (st *Encoder) AdvanceDecisionState(frameSize int) {
	st.prevMode = st.mode
	st.prevChannels = st.streamChannels
	st.prevFramesize = frameSize
	st.first = 0
}

// ---------------------------------------------------------------------------
// The rejection rules (opus_encoder.c:1221-1235).
// ---------------------------------------------------------------------------

// PacketSizeCap is packet_size_cap (opus_encoder.c:1205), the non-QEXT value. The
// :1221 cap is packet_size_cap*6 == 7656 bytes, which is why cbr_bytes can exceed
// 1276 and make opus_packet_pad at :2646 do real code-3 padding.
const PacketSizeCap = 1276

// CheckEncodeArgs applies opus_encode_native's entry rejection rules
// (opus_encoder.c:1221-1235) and returns the capped max_data_bytes.
//
//	max_data_bytes = IMIN(packet_size_cap*6, out_data_bytes)     :1221
//	frame_size <= 0 || max_data_bytes <= 0  -> OPUS_BAD_ARG      :1224
//	max_data_bytes == 1 && Fs == frame_size*10 -> OPUS_BUFFER_TOO_SMALL :1231
//
// The "cannot encode 100 ms in 1 byte" rule at :1231 is a genuine special case: a
// 100 ms frame needs a code-3 TOC (two bytes), so one byte physically cannot carry
// even the degenerate packet.
//
// frame_size is REJECTED here rather than in FrameSizeSelect: opus_encode (:2662)
// calls FrameSizeSelect first, and its -1 arrives here as frame_size <= 0.
//
// The returned error is nil (OPUS_OK), ErrBadArg (OPUS_BAD_ARG, -1) or
// ErrBufferTooSmall (OPUS_BUFFER_TOO_SMALL, -2).
func (st *Encoder) CheckEncodeArgs(frameSize, outDataBytes int) (maxDataBytes int, err error) {
	// Just avoid insane packet sizes here, but the real bounds are applied later on.
	maxDataBytes = PacketSizeCap * 6
	if outDataBytes < maxDataBytes {
		maxDataBytes = outDataBytes
	}
	if frameSize <= 0 || maxDataBytes <= 0 {
		return maxDataBytes, ErrBadArg
	}
	// Cannot encode 100 ms in 1 byte.
	if maxDataBytes == 1 && st.Fs == int32(frameSize*10) {
		return maxDataBytes, ErrBufferTooSmall
	}
	return maxDataBytes, nil
}
