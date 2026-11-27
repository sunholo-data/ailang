package opusenc

import (
	"encoding/binary"
	"hash/fnv"
)

// State is the field-level dump of the cross-frame OpusEncoder state, in the
// CANONICAL ORDER shared with the C oracle (internal/reftest/oracle/opusenc_shim.h,
// oracle_topenc_state) and with Hash below. Everything here lives at or after
// OPUS_ENCODER_RESET_START (opus_encoder.c:111), i.e. it is exactly the region
// OPUS_RESET_STATE clears.
//
// THE CANONICAL ORDER (do not reorder; the CP9 driver and the shim both depend on
// it, and Hash mixes the fields in exactly this sequence):
//
//	StreamChannels, HybridStereoWidthQ14, VariableHPSmth2Q15, PrevHBGain,
//	HPMem[4], Mode, PrevMode, PrevChannels, PrevFramesize, Bandwidth,
//	AutoBandwidth, First, BitrateBps, NbNoActivityMsQ1, PeakSignalEnergy,
//	RangeFinal, DelayBuffer[encoderBuffer*channels]
//
// # NbNoActivityMsQ1 and PeakSignalEnergy (added when DTX landed)
//
// Both are cross-frame OpusEncoder fields that generalized DTX reaches:
//
//   - nb_no_activity_ms_Q1 is LIVE: it is the consecutive-silence counter
//     decide_dtx_mode (opus_encoder.c:1115) advances, and it directly gates
//     whether a frame is dropped to a 1-byte DTX packet.
//   - peak_signal_energy is mutated at :1317 but is OUTPUT-DEAD for the DTX
//     DECISION in this config: the guard at :2565 reduces to `use_dtx &&
//     is_silence` (see dtx.go), so DTX fires only on exact digital silence, where
//     the energy path that reads peak is bypassed. It is nonetheless computed
//     always-on and compared, for state-hash defense-in-depth (it is written in
//     encodeNative, one of the functions DTX modifies, so a divergence must fail
//     the gate rather than silently pass) and SILK-forward-compatibility.
//
// # width_mem STAYS DELIBERATELY EXCLUDED (and this asymmetry is intentional)
//
// st->width_mem (the StereoWidthState that compute_stereo_width mutates at
// opus_encoder.c:1322) is EXECUTED but OUTPUT-DEAD: stereo_width, the value it
// accumulates into, is consumed only inside the user_forced_mode == OPUS_AUTO
// mode-decision block, which OPUS_SET_FORCE_MODE(MODE_CELT_ONLY) never enters and
// which stays unreachable without SILK. This port pins stereo_width to 0 and does
// not compute or compare width_mem. Do NOT "fix" the asymmetry with
// peak_signal_energy: peak is written in encodeNative, a function DTX modifies,
// width_mem is not, so peak is compared and width_mem is not.
//
// Also absent, and correctly so, because they are not cross-frame state on this
// path: silk_mode.stereoWidth_Q14 (recomputed from equiv_rate every frame at
// :2320-2327 before any read), nonfinal_frame (written only inside the multiframe
// path, and read only on the SILK path), and silk_bw_switch (SILK only, never
// written in CELT-only).
type State struct {
	StreamChannels       int32
	HybridStereoWidthQ14 int32
	VariableHPSmth2Q15   int32
	PrevHBGain           int32
	HPMem                [4]int32
	Mode                 int32
	PrevMode             int32
	PrevChannels         int32
	PrevFramesize        int32
	Bandwidth            int32
	AutoBandwidth        int32
	First                int32
	BitrateBps           int32
	// NbNoActivityMsQ1 and PeakSignalEnergy mirror the C struct positions (they
	// follow width_mem / detected_bandwidth and precede nonfinal_frame / rangeFinal).
	// NbNoActivityMsQ1 is the LIVE generalized-DTX silence counter; PeakSignalEnergy
	// is output-dead for the DTX decision in this config but is carried and compared
	// for state-hash defense-in-depth. See the doc above and dtx.go.
	NbNoActivityMsQ1 int32
	PeakSignalEnergy int32
	RangeFinal       uint32
	// DelayBuffer is the VALID prefix of st->delay_buffer: encoder_buffer*channels
	// samples. The C declares opus_res[MAX_ENCODER_BUFFER*2] but shortens the
	// allocation by MAX_ENCODER_BUFFER*sizeof(opus_res) for mono (:235), so only
	// this prefix exists.
	DelayBuffer []int16
}

// State returns the cross-frame encoder state in the canonical order. The
// DelayBuffer slice is a copy, so the caller can hold it across frames.
func (st *Encoder) State() State {
	s := State{
		StreamChannels:       int32(st.streamChannels),
		HybridStereoWidthQ14: int32(st.hybridStereoWidthQ14),
		VariableHPSmth2Q15:   st.variableHPSmth2Q15,
		PrevHBGain:           int32(st.prevHBGain),
		HPMem:                st.hpMem,
		Mode:                 int32(st.mode),
		PrevMode:             int32(st.prevMode),
		PrevChannels:         int32(st.prevChannels),
		PrevFramesize:        int32(st.prevFramesize),
		Bandwidth:            int32(st.bandwidth),
		AutoBandwidth:        int32(st.autoBandwidth),
		First:                int32(st.first),
		BitrateBps:           st.bitrateBps,
		NbNoActivityMsQ1:     int32(st.nbNoActivityMsQ1),
		PeakSignalEnergy:     st.peakSignalEnergy,
		RangeFinal:           st.rangeFinal,
	}
	s.DelayBuffer = make([]int16, len(st.delayBuffer))
	copy(s.DelayBuffer, st.delayBuffer)
	return s
}

// Hash is an FNV-1a digest of State's fields mixed in the canonical order, with
// every scalar written little-endian. It is a convenience for spotting the FRAME
// on which a sequence diverges; it is NOT a substitute for the field-level
// comparison, which is what tells you WHICH field went wrong. The CP9 gate
// compares fields, not just this.
func (s State) Hash() uint64 {
	h := fnv.New64a()
	var b [4]byte
	put := func(v uint32) {
		binary.LittleEndian.PutUint32(b[:], v)
		_, _ = h.Write(b[:])
	}

	put(uint32(s.StreamChannels))
	put(uint32(s.HybridStereoWidthQ14))
	put(uint32(s.VariableHPSmth2Q15))
	put(uint32(s.PrevHBGain))
	for _, v := range s.HPMem {
		put(uint32(v))
	}
	put(uint32(s.Mode))
	put(uint32(s.PrevMode))
	put(uint32(s.PrevChannels))
	put(uint32(s.PrevFramesize))
	put(uint32(s.Bandwidth))
	put(uint32(s.AutoBandwidth))
	put(uint32(s.First))
	put(uint32(s.BitrateBps))
	put(uint32(s.NbNoActivityMsQ1))
	put(uint32(s.PeakSignalEnergy))
	put(s.RangeFinal)
	for _, v := range s.DelayBuffer {
		put(uint32(int32(v)))
	}
	return h.Sum64()
}

// Hash returns State().Hash().
func (st *Encoder) Hash() uint64 { return st.State().Hash() }
