// Package opusenc is a Go port of libopus src/opus_encoder.c (v1.6.1): the
// top-level Opus encoder that wraps the CELT (and, later, SILK) core so it emits
// a real Opus packet, TOC byte plus payload. It mirrors internal/opusdec on the
// encode side, preserves C names and provenance references to the pinned libopus
// source, and is held bit-exact against it by the refc differential gate.
//
// # Frozen phase-4 configuration
//
// OPUS_APPLICATION_AUDIO + OPUS_SET_FORCE_MODE(MODE_CELT_ONLY). The application
// matters: only AUDIO (and VOIP) keep encoder_buffer = Fs/100 and
// delay_compensation = Fs/250 (opus_encoder.c:305,:313), so the delay-compensation
// ring exists. RESTRICTED_LOWDELAY zeroes both and the ring vanishes.
//
// ALL FIVE SAMPLE RATES are supported (8/12/16/24/48 kHz), and every rate is
// pinned byte-for-byte against the C by TestOpusencEncodeSampleRateSweep. The rate
// is NOT a CELT mode: celt_encoder_init builds the 48 kHz / 960 mode whatever Fs
// is, and threads the rate in as st->upsample = resampling_factor(Fs)
// (celt_encoder.c:255), which zero-stuffs the input up to 48 kHz. Everything at
// this layer is expressed at Fs (encoder_buffer, delay_compensation, the frame-size
// domain Fs/400..Fs/50, the Nyquist bandwidth clamps at :1643-1650) and CELT
// converts once, at celt_encode_with_ec:1837. Frame durations above 20 ms are
// coded through the multiframe repacketizer (opus_encode_native:1698).
//
// The build config is FIXED_POINT + DISABLE_FLOAT_API with no QEXT/RES24/DRED, so
// opus_res is opus_int16 and the API input is int16 throughout. Consequently the
// whole analysis.c / TonalityAnalysisState surface (run_analysis, AnalysisInfo,
// detected_bandwidth) compiles away, voice_ratio is pinned to -1
// (opus_encoder.c:1307) and voice_est is the constant 48.
//
// # What is deliberately absent
//
// Forcing MODE_CELT_ONLY makes several large blocks of opus_encoder.c
// unreachable, and they are NOT ported (each is called out at its site):
//
//   - The SILK block (:2043-2261) and every hybrid path.
//   - Mode selection (:1473-1527): a forced mode takes the else at :1528-1530.
//   - redundancy / celt_to_silk / to_celt / prefill: all need a mode transition,
//     which cannot happen (prev_mode is 0 on frame 1, MODE_CELT_ONLY forever
//     after).
//   - hp_cutoff + silk_biquad (:441-476): VOIP only; dc_reject is the AUDIO path
//     and hard-codes cutoff 3.
//   - gain_fade (:581): HB_gain is always Q15ONE in CELT-only so it never fires,
//     though the st->prev_HB_gain = HB_gain assignment (:2319) is still live.
//   - decide_fec (returns 0 at :943), compute_redundancy_bytes,
//     compute_silk_rate_for_hybrid.
//   - DTX (deferred).
//
// # Executed but output-dead
//
// compute_stereo_width (:854, mutates st->width_mem) and compute_frame_energy /
// is_digital_silence (mutate st->peak_signal_energy) RUN in C but their results
// are consumed only by paths that are dead here: stereo_width only inside the
// user_forced_mode == OPUS_AUTO mode decision, and peak_signal_energy only by
// DTX. Both are omitted, stereo_width is pinned to 0, and width_mem /
// peak_signal_energy are EXCLUDED from State(). See the State doc for the full
// argument and the condition under which that must be revisited.
package opusenc
