// CELT-only fixed-point encoder, phase 4. This file ports the FRONT HALF of
// celt/celt_encoder.c (celt_encode_with_ec, lines ~1726-2240): the encoder state
// struct and the time-domain analysis stages that run before residual coding:
// celt_preemphasis, tone_detect, transient_analysis, patch_transient_decision,
// run_prefilter (via internal/celt/pitch.go remove_doubling and celt.go
// comb_filter) and compute_mdcts, then the CP4 band-energy/normalise wiring. The
// back-half analysis (tf_analysis, dynalloc_analysis, alloc_trim_analysis,
// stereo_analysis, spreading_decision) is CP8b and the coding path
// (quant_coarse_energy, allocation, quant_all_bands, anti-collapse, finalise) is
// CP8c; both stop at the TODO boundary marked below.
//
// Frozen config (celt/arch.h): FIXED_POINT + DISABLE_FLOAT_API + OPUS_FAST_INT64,
// no ENABLE_QEXT/RES24/RESYNTH/CUSTOM_MODES. opus_res is opus_int16 (RES_SHIFT 0),
// so RES2SIG(a) = SHL32(EXTEND32(a), SIG_SHIFT); celt_sig/celt_norm/celt_ener/
// celt_glog are opus_int32; opus_val16/celt_coef are opus_int16. analysis.c is
// omitted (mode forced CELT-only), so the AnalysisInfo-dependent branches
// (#ifndef DISABLE_FLOAT_API) are gated out here.
//
// Memory management is idiomatic Go (Layer A): the trailing single-alloc VLA of
// struct OpusCustomEncoder becomes explicit named slices. The DSP itself is Layer
// C, transliterated verbatim from celt_encoder.c against libopus v1.6.1.

package celt

import (
	"hash/fnv"

	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
)

// Encoder is OpusCustomEncoder (celt_encoder.c:63) for the frozen 48 kHz / 960
// CELT-only config. The trailing VLA arrays (in_mem, prefilter_mem, oldBandE,
// oldLogE, oldLogE2, energyError) are explicit slices; everything from rng
// onward is the reset region (ENCODER_RESET_START).
type Encoder struct {
	mode           *celtMode
	overlap        int
	channels       int // CC: physical channels
	streamChannels int // C: coded channels

	forceIntra     int
	clip           int
	disablePf      int
	complexity     int
	upsample       int
	start, end     int
	bitrate        int32
	vbr            int
	signalling     int
	constrainedVbr int
	lossRate       int
	lsbDepth       int
	lfe            int
	disableInv     int
	arch           int

	// --- reset region (ENCODER_RESET_START rng) ---
	rng            uint32
	spreadDecision int
	delayedIntra   int32 // opus_val32
	tonalAverage   int
	lastCodedBands int
	hfAverage      int
	tapsetDecision int

	prefilterPeriod int
	prefilterGain   int16 // opus_val16
	prefilterTapset int
	consecTransient int

	// silk_info fields consulted by the CELT front half (allow_weak_transients).
	// In C silk_info is INSIDE ENCODER_RESET_START (celt_encoder.c:109-110), so it
	// is cleared by OPUS_RESET_STATE; it lives here (not above the reset marker) to
	// match that. Always 0 on the CELT-only path.
	silkSignalType int

	preemphMemE [2]int32 // celt_sig
	preemphMemD [2]int32 // celt_sig

	vbrReservoir int32
	vbrDrift     int32
	vbrOffset    int32
	vbrCount     int32
	overlapMax   int32 // opus_val32
	stereoSaving int16 // opus_val16
	intensity    int
	energyMask   []int32 // celt_glog*, nil on the non-surround path
	specAvg      int32   // celt_glog

	// Per-frame working buffers. In C these are stack VARDECL/ALLOCs, reclaimed on
	// frame pop; here they are pooled on the encoder so a steady-state Encode
	// allocates nothing (see scratch.go, and its zeroing contract). This is state
	// only in the sense that a C thread's stack is: it carries nothing across
	// frames that any frame reads, and Reset does not need to touch it.
	sc scratch

	// Cross-frame VLA arrays (sizes use CC per opus_custom_encoder_get_size).
	inMem        []int32 // channels*overlap                 (celt_sig)
	prefilterMem []int32 // channels*COMBFILTER_MAXPERIOD     (celt_sig)
	oldBandE     []int32 // channels*nbEBands                 (celt_glog)
	oldLogE      []int32 // channels*nbEBands                 (celt_glog)
	oldLogE2     []int32 // channels*nbEBands                 (celt_glog)
	energyError  []int32 // channels*nbEBands                 (celt_glog)
}

// NewEncoder allocates and initializes a CELT encoder, mirroring
// celt_encoder_init(st, sampleRate, channels, 0) (celt_encoder.c:239) followed by
// OPUS_RESET_STATE. It returns nil if channels is not 1 or 2, or if sampleRate is
// not one of 8000/12000/16000/24000/48000.
//
// THE MODE IS ALWAYS THE 48 kHz / 960 ONE, at every sampleRate: that is what
// celt_encoder_init does (:250, opus_custom_mode_create(48000, 960)). The rate
// enters through ONE field, st.upsample = resampling_factor(sampleRate) (:255),
// which celt_preemphasis zero-stuffs the input with and compute_mdcts scales and
// band-limits by. Everything downstream of the MDCT - the LM search, vbr_rate,
// the allocation - is expressed in 48 kHz samples, which is why celt_encode_with_ec
// starts by doing frame_size *= st->upsample (:1837).
func NewEncoder(sampleRate int32, channels int) *Encoder {
	if channels < 1 || channels > 2 {
		return nil
	}
	upsample := ResamplingFactor(sampleRate)
	if upsample == 0 {
		return nil
	}
	st := &Encoder{}
	m := &mode48000_960
	st.mode = m
	st.overlap = m.overlap
	st.streamChannels = channels
	st.channels = channels

	// opus_custom_encoder_init_arch defaults (celt_encoder.c:194). The C seeds
	// st->upsample = 1 there (:208) and celt_encoder_init OVERWRITES it at :255;
	// the two are collapsed here because opus_custom_encoder_init is not exposed.
	st.upsample = upsample
	st.start = 0
	st.end = m.effEBands
	st.signalling = 1
	st.arch = 0
	st.constrainedVbr = 1
	st.clip = 1
	st.bitrate = opusBitrateMax
	st.vbr = 0
	st.forceIntra = 0
	st.complexity = 5
	st.lsbDepth = 24

	nb := m.nbEBands
	st.inMem = make([]int32, channels*st.overlap)
	st.prefilterMem = make([]int32, channels*combfilterMaxperiod)
	st.oldBandE = make([]int32, channels*nb)
	st.oldLogE = make([]int32, channels*nb)
	st.oldLogE2 = make([]int32, channels*nb)
	st.energyError = make([]int32, channels*nb)

	st.Reset()
	return st
}

// opusBitrateMax is OPUS_BITRATE_MAX (-1).
const opusBitrateMax = -1

// Reset implements opus_custom_encoder_ctl(st, OPUS_RESET_STATE)
// (celt_encoder.c:3073): clear the reset region, seed oldLogE/oldLogE2 to
// -GCONST(28), and restore delayedIntra=1, spread_decision=SPREAD_NORMAL,
// tonal_average=256, hf_average=0, tapset_decision=0, vbr_offset=0.
func (st *Encoder) Reset() {
	st.rng = 0
	st.spreadDecision = 0
	st.delayedIntra = 0
	st.tonalAverage = 0
	st.lastCodedBands = 0
	st.hfAverage = 0
	st.tapsetDecision = 0
	st.prefilterPeriod = 0
	st.prefilterGain = 0
	st.prefilterTapset = 0
	st.consecTransient = 0
	st.silkSignalType = 0
	st.preemphMemE[0], st.preemphMemE[1] = 0, 0
	st.preemphMemD[0], st.preemphMemD[1] = 0, 0
	st.vbrReservoir = 0
	st.vbrDrift = 0
	st.vbrOffset = 0
	st.vbrCount = 0
	st.overlapMax = 0
	st.stereoSaving = 0
	st.intensity = 0
	st.energyMask = nil
	st.specAvg = 0

	for i := range st.inMem {
		st.inMem[i] = 0
	}
	for i := range st.prefilterMem {
		st.prefilterMem[i] = 0
	}
	for i := range st.oldBandE {
		st.oldBandE[i] = 0
	}
	for i := range st.energyError {
		st.energyError[i] = 0
	}
	for i := 0; i < st.channels*st.mode.nbEBands; i++ {
		st.oldLogE[i] = -gconst28
		st.oldLogE2[i] = -gconst28
	}

	st.delayedIntra = 1
	st.spreadDecision = spreadNormal
	st.tonalAverage = 256
	st.hfAverage = 0
	st.tapsetDecision = 0
}

// SetComplexity mirrors OPUS_SET_COMPLEXITY (0..10).
func (st *Encoder) SetComplexity(v int) { st.complexity = v }

// SetLossRate mirrors OPUS_SET_PACKET_LOSS_PERC (0..100).
func (st *Encoder) SetLossRate(v int) { st.lossRate = v }

// SetLFE mirrors CELT_SET_LFE (0 or 1).
func (st *Encoder) SetLFE(v int) { st.lfe = v }

// SetDisablePf disables the prefilter/postfilter (CELT_SET_DISABLE_PREFILTER).
func (st *Encoder) SetDisablePf(v int) { st.disablePf = v }

// SetForceIntra forces intra coarse-energy coding. In C, disable_pf and
// force_intra are set together by CELT_SET_PREDICTION (celt_encoder.c:2972:
// disable_pf = value<=1, force_intra = value==0); the port keeps the two fields
// independently settable, which is a strict superset of the ctl.
func (st *Encoder) SetForceIntra(v int) { st.forceIntra = v }

// SetVBR mirrors OPUS_SET_VBR (celt_encoder.c:2995). The ctl does not validate.
func (st *Encoder) SetVBR(v int) { st.vbr = v }

// SetConstrainedVBR mirrors OPUS_SET_VBR_CONSTRAINT (celt_encoder.c:2989). The
// ctl does not validate.
func (st *Encoder) SetConstrainedVBR(v int) { st.constrainedVbr = v }

// SetBitrate mirrors OPUS_SET_BITRATE (celt_encoder.c:3001): OPUS_BITRATE_MAX
// (-1) means "no limit"; any other value <= 500 is OPUS_BAD_ARG and leaves the
// bitrate unchanged; otherwise the value is clamped to 750000*channels (CC).
// Reports whether the value was accepted.
func (st *Encoder) SetBitrate(v int32) bool {
	if v <= 500 && v != opusBitrateMax {
		return false
	}
	st.bitrate = fixedmath.MIN32(v, 750000*int32(st.channels))
	return true
}

// SetLsbDepth mirrors OPUS_SET_LSB_DEPTH (celt_encoder.c:3018): 8..24, otherwise
// OPUS_BAD_ARG (unchanged). Reports whether the value was accepted.
func (st *Encoder) SetLsbDepth(v int) bool {
	if v < 8 || v > 24 {
		return false
	}
	st.lsbDepth = v
	return true
}

// SetDisableInv mirrors OPUS_SET_PHASE_INVERSION_DISABLED
// (celt_encoder.c:3032): 0 or 1, otherwise OPUS_BAD_ARG (unchanged). Reports
// whether the value was accepted.
func (st *Encoder) SetDisableInv(v int) bool {
	if v < 0 || v > 1 {
		return false
	}
	st.disableInv = v
	return true
}

// SetStartBand mirrors CELT_SET_START_BAND (celt_encoder.c:2956): 0..nbEBands-1,
// otherwise OPUS_BAD_ARG (unchanged). Reports whether the value was accepted.
func (st *Encoder) SetStartBand(v int) bool {
	if v < 0 || v >= st.mode.nbEBands {
		return false
	}
	st.start = v
	return true
}

// SetEndBand mirrors CELT_SET_END_BAND (celt_encoder.c:2964): 1..nbEBands,
// otherwise OPUS_BAD_ARG (unchanged). Reports whether the value was accepted.
func (st *Encoder) SetEndBand(v int) bool {
	if v < 1 || v > st.mode.nbEBands {
		return false
	}
	st.end = v
	return true
}

// SetStreamChannels mirrors CELT_SET_CHANNELS (celt_encoder.c:3010): the number
// of CODED channels C (1 or 2), which may be less than the number of physical
// channels CC the encoder was allocated for. Reports whether the value was
// accepted.
func (st *Encoder) SetStreamChannels(v int) bool {
	if v < 1 || v > 2 {
		return false
	}
	st.streamChannels = v
	return true
}

// SetSignalling mirrors CELT_SET_SIGNALLING (celt_encoder.c:3024). celt_encoder_init
// leaves it at 1 (celt_encoder.c:211), but opus_encoder_init immediately clears it
// with CELT_SET_SIGNALLING(0) (opus_encoder.c:283) because the Opus layer writes
// its own TOC byte instead of CELT's self-delimiting signalling byte.
//
// It is OUTPUT-NEUTRAL when driven from opus_encoder: signalling is read only in
// the CUSTOM_MODES signalling-byte block (gated out of this build) and at
// celt_encoder.c:1918 (`-!!st->signalling` in the CBR nbCompressedBytes clamp),
// and that clamp needs `!vbr && bitrate != OPUS_BITRATE_MAX`, which never holds
// here: opus_encode_frame_native pins CELT's bitrate to OPUS_BITRATE_MAX
// (opus_encoder.c:2286) and only raises it off OPUS_BITRATE_MAX inside
// `if (st->use_vbr)` (:2462). The setter exists so the Go CELT encoder's state
// still matches the C encoder's field for field.
func (st *Encoder) SetSignalling(v int) { st.signalling = v }

// SetEnergyMask mirrors OPUS_SET_ENERGY_MASK (celt_encoder.c:3144): the surround
// masking curve (C*nbEBands celt_glog), or nil for none. Only the multistream
// (surround) encoder ever sets it, so on the frozen single-stream CELT path it
// stays nil and the surround-masking block of celt_encode_with_ec is inert. The
// slice is aliased, not copied, exactly like the C pointer.
func (st *Encoder) SetEnergyMask(mask []int32) { st.energyMask = mask }

// Rng returns the encoder range-coder state after the last frame
// (OPUS_GET_FINAL_RANGE).
func (st *Encoder) Rng() uint32 { return st.rng }

// Overlap is celt_mode->overlap. The Opus layer fetches the CELT mode with
// CELT_GET_MODE (opus_encoder.c:1957) purely so it can pass celt_mode->overlap
// and celt_mode->window to gain_fade (:2317) and stereo_fade (:2344), which run
// in the Opus PCM domain, not inside CELT. This accessor is that seam.
func (st *Encoder) Overlap() int { return st.mode.overlap }

// Window is celt_mode->window, the MDCT overlap window (celt_coef, Q15 int16 in
// this fixed-point build) that stereo_fade cross-fades with. See Overlap.
func (st *Encoder) Window() []int16 { return st.mode.window }

// BitrateToBits is bitrate_to_bits (celt.h:151), the static inline the VBR path
// of celt_encode_with_ec (:1903) uses to turn st->bitrate into a per-frame bit
// budget: bitrate*6/(6*Fs/frame_size). All three operands are opus_int32 and the
// inner 6*Fs/frame_size division truncates BEFORE the outer one, so the two
// divisions must not be folded.
//
// Exported because the Opus layer's CBR byte-budget derivation
// (opus_encoder.c:1330-1333) is built out of this pair, and both must be the SAME
// integer math as CELT's, not a re-derivation.
func BitrateToBits(bitrate, Fs, frameSize int32) int32 {
	return bitrate * 6 / (6 * Fs / frameSize)
}

// BitsToBitrate is bits_to_bitrate (celt.h:147), the inverse of BitrateToBits:
// bits*(6*Fs/frame_size)/6. Same truncation rule: the inner 6*Fs/frame_size
// division happens first. opus_encode_native uses it at :1325 (the max_bitrate
// ceiling inside user_bitrate_to_bitrate) and at :1331 to requantize
// st->bitrate_bps to the CBR byte budget it just rounded to.
func BitsToBitrate(bits, Fs, frameSize int32) int32 {
	return bits * (6 * Fs / frameSize) / 6
}

// celt_preemphasis (celt_encoder.c:557) for the frozen config: apply the mode's
// pre-emphasis high-pass to one channel of int16 PCM, writing celt_sig into inp
// and threading the per-channel filter memory *mem. pcmp is read interleaved with
// stride CC starting at pcmOff (the driver passes pcm+c). The coef[1]!=0 path
// (CUSTOM_MODES/QEXT only) and the FIXED_POINT clip block (RES24 only) are gated
// out of this build.
func celtPreemphasis(pcmp []int16, pcmOff int, inp []int32, N, CC, upsample int, coef []int16, mem *int32, clip int) {
	coef0 := coef[0]
	m := *mem

	// Fast path for the normal 48 kHz case and no clipping.
	if coef[1] == 0 && upsample == 1 && clip == 0 {
		// Slice to N so the destination store drops its per-sample check; the
		// strided pcmp[pcmOff+CC*i] read keeps its check (stride defeats BCE).
		ib := inp[:N]
		for i := range ib {
			// RES2SIG(pcmp[CC*i]) = SHL32(EXTEND32(a), SIG_SHIFT).
			x := fixedmath.SHL32(fixedmath.EXTEND32(pcmp[pcmOff+CC*i]), sigShift)
			ib[i] = x - m
			m = fixedmath.MULT16_32_Q15(coef0, x)
		}
		*mem = m
		return
	}

	Nu := N / upsample
	ib := inp[:N]
	if upsample != 1 {
		for i := range ib {
			ib[i] = 0
		}
	}
	for i := 0; i < Nu; i++ {
		inp[i*upsample] = fixedmath.SHL32(fixedmath.EXTEND32(pcmp[pcmOff+CC*i]), sigShift)
	}
	// clip is a no-op here (FIXED_POINT, non-RES24); the coef[1]!=0 branch is
	// gated out (non-CUSTOM_MODES/QEXT), leaving only the coef0 loop.
	for i := range ib {
		x := ib[i]
		ib[i] = x - m
		m = fixedmath.MULT16_32_Q15(coef0, x)
	}
	*mem = m
}

// invTable is transient_analysis' 6*64/x table (celt_encoder.c:286), trained to
// minimize the average error.
var invTable = [128]byte{
	255, 255, 156, 110, 86, 70, 59, 51, 45, 40, 37, 33, 31, 28, 26, 25,
	23, 22, 21, 20, 19, 18, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12,
	12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 8, 8,
	8, 8, 8, 7, 7, 7, 7, 7, 7, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6, 6, 5, 5, 5, 5, 5, 5, 5,
	5, 5, 5, 5, 5, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4,
	4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2,
}

// transientAnalysis is transient_analysis (celt_encoder.c:267) for FIXED_POINT:
// decide whether the frame is transient by comparing a bitrate-normalized
// temporal noise-to-mask ratio against a threshold, per channel, and report the
// dominant channel (tfChan), a VBR-boost metric (tfEstimate) and a weak-transient
// flag. in is C*length celt_sig, channel c at in[i+c*length].
func transientAnalysis(in []int32, length, C int, tfEstimate *int16, tfChan *int,
	allowWeakTransients int, weakTransient *int, toneFreq int16, toneishness int32, sc *scratch) int {
	forwardShift := 4
	isTransient := 0
	maskMetric := int32(0)
	inShift := fixedmath.IMAX(0, fixedmath.Celt_ilog2(1+celtMaxabs32(in, 0, C*length))-14)

	tmp := alloc(&sc.transientTmp, length) // VARDECL(opus_val16, tmp)
	// Re-establish len(tmp) == length for the bounds-check prover: alloc returns
	// a slice of length length, but that fact is opaque across the call, so every
	// unit-stride tmp[i] in the length- and len2-bounded loops below would
	// otherwise carry a per-sample check. One reslice check here elides those. The
	// strided forward-pass reads (tmp[2*i]) are not covered; see that loop.
	tmp = tmp[:length]

	*weakTransient = 0
	// For lower bitrates, use a more conservative forward masking decay to avoid
	// coding transients at very low bitrate (mostly hybrid).
	if allowWeakTransients != 0 {
		forwardShift = 5
	}
	len2 := length / 2
	for c := 0; c < C; c++ {
		var mean int32
		var unmask int32
		var norm int32
		var maxE int16
		mem0 := int32(0)
		mem1 := int32(0)
		// High-pass filter: (1 - 2z^-1 + z^-2) / (1 - z^-1 + .5z^-2).
		// Reslice this channel's window to exactly length so the in reads below are
		// check-free; the strided base c*length is otherwise opaque to the prover.
		// The two-step form (skip c*length, then cap to length) is deliberate: the
		// single expression in[c*length : c*length+length] leaves the length as a
		// symbolic difference the bounds-check prover will not cancel, so the check
		// would survive. Taking [:length] off the tail slice gives a 0-based high
		// index the prover reads as length directly.
		inc := in[c*length:][:length]
		for i := 0; i < length; i++ {
			x := fixedmath.SHR32(inc[i], inShift)
			y := fixedmath.ADD32(mem0, x)
			mem0 = mem1 + y - fixedmath.SHL32(x, 1)
			mem1 = x - fixedmath.SHR32(y, 1)
			tmp[i] = fixedmath.SROUND16(y, 2)
		}
		// First few samples are bad because we don't propagate the memory.
		for i := 0; i < 12; i++ {
			tmp[i] = 0
		}

		// Normalize tmp to max range.
		{
			shift := 14 - fixedmath.Celt_ilog2(fixedmath.MAX16(1, celtMaxabs16(tmp, length)))
			if shift != 0 {
				for i := 0; i < length; i++ {
					tmp[i] = fixedmath.SHL16(tmp[i], shift)
				}
			}
		}

		mean = 0
		mem0 = 0
		// Grouping by two to reduce complexity. Forward pass computes the post-echo
		// threshold. The strided reads tmp[2*i], tmp[2*i+1] keep a per-sample check:
		// the prover cannot derive 2*i+1 < len(tmp) from i < len2, and a 2*len2 view
		// does not help (the 2*i index defeats the same induction). Left as-is rather
		// than restructure the arithmetic. The tmp[i] write is elided by tmp[:length].
		for i := 0; i < len2; i++ {
			x2 := fixedmath.PSHR32(fixedmath.MULT16_16(tmp[2*i], tmp[2*i])+
				fixedmath.MULT16_16(tmp[2*i+1], tmp[2*i+1]), 4)
			mean += fixedmath.PSHR32(x2, 12)
			mem0 = mem0 + fixedmath.PSHR32(x2-mem0, forwardShift)
			tmp[i] = fixedmath.EXTRACT16(fixedmath.PSHR32(mem0, 12))
		}

		mem0 = 0
		maxE = 0
		// The forward pass has written tmp[0:len2]; view that half so the backward
		// pass reads and writes it check-free (len2 is otherwise not proven <=
		// len(tmp)). tb aliases tmp, so writes through it are unchanged. The stride-4
		// harmonic-mean loop below keeps its check: the prover cannot discharge the
		// +4 induction, and at len2/4 iterations it is not worth restructuring.
		tb := tmp[:len2]
		// Backward pass computes the pre-echo threshold. Backward masking 13.9 dB/ms.
		for i := len2 - 1; i >= 0; i-- {
			mem0 = mem0 + fixedmath.PSHR32(fixedmath.SHL32(fixedmath.EXTEND32(tb[i]), 4)-mem0, 3)
			tb[i] = fixedmath.EXTRACT16(fixedmath.PSHR32(mem0, 4))
			maxE = fixedmath.EXTRACT16(fixedmath.MAX16(int32(maxE), int32(tb[i])))
		}

		// Ratio of frame energy over harmonic mean of the energy. Frame energy is
		// the geometric mean of the energy and half the max. Two sqrt() to avoid
		// overflow.
		mean = fixedmath.MULT16_16(int16(fixedmath.Celt_sqrt(mean)),
			int16(fixedmath.Celt_sqrt(fixedmath.MULT16_16(maxE, int16(len2>>1)))))
		// Inverse of the mean energy in Q15+6.
		norm = fixedmath.SHL32(int32(len2), 6+14) / fixedmath.ADD32(1 /*EPSILON*/, fixedmath.SHR32(mean, 1))
		// Harmonic mean discarding the unreliable boundaries; the data is smooth, so
		// only 1/4th of the samples are taken.
		unmask = 0
		for i := 12; i < len2-5; i += 4 {
			// Do not round to nearest.
			id := fixedmath.MAX32(0, fixedmath.MIN32(127,
				fixedmath.MULT16_32_Q15(tb[i]+1 /*EPSILON*/, norm)))
			unmask += int32(invTable[id])
		}
		// Normalize, compensate for the 1/4th sampling and the factor of 6 in the
		// inverse table.
		unmask = 64 * unmask * 4 / (6 * (int32(len2) - 17))
		if unmask > maskMetric {
			*tfChan = c
			maskMetric = unmask
		}
	}
	isTransientBool := maskMetric > 200
	if isTransientBool {
		isTransient = 1
	}
	// Prevent confusing the partial cycle of a very low frequency tone with a
	// transient.
	if toneishness > qconst0_98Q29 && toneFreq < fixedmath.QCONST16(0.026, 13) {
		isTransient = 0
		maskMetric = 0
	}
	// For low bitrates, define "weak transients" handled differently to avoid
	// partial collapse.
	if allowWeakTransients != 0 && isTransient != 0 && maskMetric < 600 {
		isTransient = 0
		*weakTransient = 1
	}
	// Arbitrary metric for VBR boost.
	tfMax := fixedmath.EXTRACT16(fixedmath.MAX16(0, fixedmath.Celt_sqrt(27*maskMetric)-42))
	*tfEstimate = fixedmath.EXTRACT16(fixedmath.Celt_sqrt(fixedmath.MAX32(0,
		fixedmath.SHL32(fixedmath.MULT16_16(fixedmath.QCONST16(0.0069, 14),
			int16(fixedmath.MIN16(163, int32(tfMax)))), 14)-fixedmath.QCONST32(0.139, 28))))
	return isTransient
}

// patchTransientDecision is patch_transient_decision (celt_encoder.c:473): look
// for sudden energy increases (relative to an aggressively spread copy of the old
// frame) to decide whether the time-domain transient decision must be patched.
func patchTransientDecision(newE, oldE []int32, nbEBands, start, end, C int) int {
	var meanDiff int32
	var spreadOld [26]int32
	// Aggressive (-6 dB/Bark) spreading of the old frame to avoid false detection
	// from irrelevant bands.
	if C == 1 {
		spreadOld[start] = oldE[start]
		for i := start + 1; i < end; i++ {
			spreadOld[i] = fixedmath.MAX32(spreadOld[i-1]-gconst1, oldE[i]) // MAXG
		}
	} else {
		spreadOld[start] = fixedmath.MAX32(oldE[start], oldE[start+nbEBands])
		for i := start + 1; i < end; i++ {
			spreadOld[i] = fixedmath.MAX32(spreadOld[i-1]-gconst1,
				fixedmath.MAX32(oldE[i], oldE[i+nbEBands]))
		}
	}
	for i := end - 2; i >= start; i-- {
		spreadOld[i] = fixedmath.MAX32(spreadOld[i], spreadOld[i+1]-gconst1)
	}
	// Compute mean increase. x1/x2 are opus_val16 in the C, so MAXG(0, ...) is
	// truncated to 16 bits before the SUB32 sign-extends it back.
	c := 0
	for {
		for i := fixedmath.IMAX(2, start); i < end-1; i++ {
			x1 := fixedmath.EXTRACT16(fixedmath.MAX32(0, newE[i+c*nbEBands]))
			x2 := fixedmath.EXTRACT16(fixedmath.MAX32(0, spreadOld[i]))
			meanDiff = fixedmath.ADD32(meanDiff, fixedmath.MAX32(0, fixedmath.SUB32(int32(x1), int32(x2))))
		}
		c++
		if c >= C {
			break
		}
	}
	meanDiff = fixedmath.DIV32(meanDiff, int32(C*(end-1-fixedmath.IMAX(2, start))))
	if meanDiff > gconst1 {
		return 1
	}
	return 0
}

// gconst1 is GCONST(1.f) = QCONST32(1, DB_SHIFT), the -1 dB/Bark spreading step
// and the mean-increase threshold used by patch_transient_decision.
var gconst1 = fixedmath.QCONST32(1, dbShift)

// computeMdcts is compute_mdcts (celt_encoder.c:511): window and forward-MDCT all
// sub-frames of all channels, interleaving the sub-frames. in is CC*(N+overlap)
// celt_sig, out is CC*N celt_sig. Keep |in| within ~2^28: the CP3 forward MDCT
// windows/folds at full int32, so a larger magnitude overflows the C oracle
// (signed-overflow UB) as well.
func computeMdcts(m *celtMode, shortBlocks int, in, out []int32, C, CC, LM, upsample int, sc *scratch) {
	overlap := m.overlap
	var N, B, shift int
	if shortBlocks != 0 {
		B = shortBlocks
		N = m.shortMdctSize
		shift = m.maxLM
	} else {
		B = 1
		N = m.shortMdctSize << LM
		shift = m.maxLM - LM
	}
	c := 0
	for {
		for b := 0; b < B; b++ {
			// Interleave the sub-frames while doing the MDCTs.
			cltMDCTForward(&m.mdct, in[c*(B*N+overlap)+b*N:], out[b+c*N*B:],
				m.window, overlap, shift, B, sc)
		}
		c++
		if c >= CC {
			break
		}
	}
	if CC == 2 && C == 1 {
		bn := B * N
		ob := out[:bn]
		o2 := out[bn : bn+bn]
		for i := range ob {
			ob[i] = fixedmath.ADD32(fixedmath.HALF32(ob[i]), fixedmath.HALF32(o2[i]))
		}
	}
	if upsample != 1 {
		c = 0
		for {
			bound := B * N / upsample
			for i := 0; i < bound; i++ {
				out[c*B*N+i] *= int32(upsample)
			}
			for i := bound; i < B*N; i++ {
				out[c*B*N+i] = 0
			}
			c++
			if c >= C {
				break
			}
		}
	}
}

// normalizeToneInput is normalize_tone_input (celt_encoder.c:1276): scale x so its
// energy sits in a range where tone_lpc's covariance sums do not overflow.
func normalizeToneInput(x []int16, length int) {
	ac0 := int32(length)
	for i := 0; i < length; i++ {
		ac0 = fixedmath.ADD32(ac0, fixedmath.SHR32(fixedmath.MULT16_16(x[i], x[i]), 10))
	}
	shift := 5 - (28-fixedmath.Celt_ilog2(ac0))/2
	if shift > 0 {
		for i := 0; i < length; i++ {
			x[i] = int16(fixedmath.PSHR32(int32(x[i]), shift))
		}
	}
}

// acosApprox is acos_approx (celt_encoder.c:1290): a polynomial approximation of
// acos(x) in the tone_detect angle units (Q13-ish), for FIXED_POINT.
func acosApprox(x int32) int {
	flip := x < 0
	if x < 0 {
		x = -x
	}
	// x14 is opus_val16 in the C: narrow x>>15 to 16 bits (unreachable divergence
	// for the lpc[0]>>1 range the caller uses, but verbatim to celt_encoder.c:1295).
	x14 := int32(fixedmath.EXTRACT16(x >> 15))
	tmp := (762 * x14 >> 14) - 3308
	tmp = (tmp * x14 >> 14) + 25726
	tmp = tmp * fixedmath.Celt_sqrt(fixedmath.MAX32(0, (int32(1)<<30)-(x<<1))) >> 16
	if flip {
		tmp = 25736 - tmp
	}
	return int(tmp)
}

// toneLpc is tone_lpc (celt_encoder.c:1305): fit a 2-tap LPC using a symmetric
// (forward+backward) covariance method and return whether the fit failed
// (ill-conditioned). lpc receives the two Q29 coefficients.
func toneLpc(x []int16, length, delay int, lpc []int32) int {
	var r00, r01, r11, r02, r12, r22 int32
	var edges int32
	// Correlations as if using the forward prediction covariance method.
	for i := 0; i < length-2*delay; i++ {
		r00 += fixedmath.MULT16_16(x[i], x[i])
		r01 += fixedmath.MULT16_16(x[i], x[i+delay])
		r02 += fixedmath.MULT16_16(x[i], x[i+2*delay])
	}
	edges = 0
	for i := 0; i < delay; i++ {
		edges += fixedmath.MULT16_16(x[length+i-2*delay], x[length+i-2*delay]) - fixedmath.MULT16_16(x[i], x[i])
	}
	r11 = r00 + edges
	edges = 0
	for i := 0; i < delay; i++ {
		edges += fixedmath.MULT16_16(x[length+i-delay], x[length+i-delay]) - fixedmath.MULT16_16(x[i+delay], x[i+delay])
	}
	r22 = r11 + edges
	edges = 0
	for i := 0; i < delay; i++ {
		edges += fixedmath.MULT16_16(x[length+i-2*delay], x[length+i-delay]) - fixedmath.MULT16_16(x[i], x[i+delay])
	}
	r12 = r01 + edges
	// Reverse and sum to get the backward contribution. R22 (== r00+r22) is not
	// read after this block, so unlike the C its store is elided.
	{
		R00 := r00 + r22
		R01 := r01 + r12
		R11 := 2 * r11
		R02 := 2 * r02
		R12 := r12 + r01
		r00 = R00
		r01 = R01
		r11 = R11
		r02 = R02
		r12 = R12
	}
	// Solve A*x=b, where A=[r00, r01; r01, r11] and b=[r02; r12].
	den := fixedmath.MULT32_32_Q31(r00, r11) - fixedmath.MULT32_32_Q31(r01, r01)
	if den <= fixedmath.SHR32(fixedmath.MULT32_32_Q31(r00, r11), 10) {
		return 1
	}
	num1 := fixedmath.MULT32_32_Q31(r02, r11) - fixedmath.MULT32_32_Q31(r01, r12)
	if num1 >= den {
		lpc[1] = fixedmath.QCONST32(1, 29)
	} else if num1 <= -den {
		lpc[1] = -fixedmath.QCONST32(1, 29)
	} else {
		lpc[1] = fixedmath.Frac_div32_q29(num1, den)
	}
	num0 := fixedmath.MULT32_32_Q31(r00, r12) - fixedmath.MULT32_32_Q31(r02, r01)
	if fixedmath.HALF32(num0) >= den {
		lpc[0] = qconst1_999999Q29
	} else if fixedmath.HALF32(num0) <= -den {
		lpc[0] = -qconst1_999999Q29
	} else {
		lpc[0] = fixedmath.Frac_div32_q29(num0, den)
	}
	return 0
}

// toneDetect is tone_detect (celt_encoder.c:1362): detect pure or nearly-pure
// tones (so the caller can keep them from destabilizing the encoder). Returns the
// tone frequency (Q13-ish, or -1 if not tonal) and writes the toneishness (Q29
// squared pole radius). in is CC*N celt_sig.
func toneDetect(in []int32, CC, N int, toneishness *int32, Fs int32, sc *scratch) int16 {
	delay := 1
	var lpc [2]int32
	var freq int16
	x := alloc(&sc.toneX, N) // VARDECL(opus_val16, x)
	// Shift by SIG_SHIFT+2 (+3 for stereo) to account for the HF gain of the
	// preemphasis filter.
	if CC == 2 {
		for i := 0; i < N; i++ {
			x[i] = fixedmath.EXTRACT16(fixedmath.PSHR32(
				fixedmath.ADD32(fixedmath.SHR32(in[i], 1), fixedmath.SHR32(in[i+N], 1)), sigShift+2))
		}
	} else {
		for i := 0; i < N; i++ {
			x[i] = fixedmath.EXTRACT16(fixedmath.PSHR32(in[i], sigShift+2))
		}
	}
	normalizeToneInput(x, N)
	fail := toneLpc(x, N, delay, lpc[:])
	// If the LPC filter resonates too close to DC, retry with down-sampling.
	for delay <= int(Fs)/3000 && (fail != 0 || (lpc[0] > fixedmath.QCONST32(1, 29) && lpc[1] < 0)) {
		delay *= 2
		fail = toneLpc(x, N, delay, lpc[:])
	}
	// Check that the filter has complex roots.
	if fail == 0 && fixedmath.MULT32_32_Q31(lpc[0], lpc[0])+fixedmath.MULT32_32_Q31(fixedmath.QCONST32(3.999999, 29), lpc[1]) < 0 {
		// Squared radius of the poles.
		*toneishness = -lpc[1]
		freq = int16((acosApprox(lpc[0]>>1) + delay/2) / delay)
	} else {
		freq = -1
		*toneishness = 0
	}
	return freq
}

// runPrefilter is run_prefilter (celt_encoder.c:1404): estimate the pitch period
// and gain, and comb-filter the input in place with the previous frame's
// post-filter (continuity). It threads in_mem and prefilter_mem (cross-frame
// state) and reports the pitch index, quantized gain, tapset and whether the
// post-filter is on. The analysis (float-only) argument and the QEXT scale are
// gated out of this build. It mutates st.prefilterPeriod (the COMBFILTER_MINPERIOD
// floor) and st.inMem; the driver updates prefilterPeriod/Gain/Tapset from the
// returned values afterward.
func (st *Encoder) runPrefilter(in, prefilterMem []int32, CC, N, prefilterTapset int,
	pitch *int, gain *int16, qgain *int, enabled, complexity int, tfEstimate int16,
	nbAvailableBytes int, toneFreq int16, toneishness int32) int {
	maxPeriod := combfilterMaxperiod
	minPeriod := combfilterMinperiod
	mode := st.mode
	overlap := mode.overlap

	var before, after [2]int32
	cancelPitch := 0

	pre := alloc(&st.sc.pre, CC*(N+maxPeriod)) // VARDECL(celt_sig, pre)
	preBase := [2]int{0, N + maxPeriod}

	c := 0
	for {
		copy(pre[preBase[c]:preBase[c]+maxPeriod], prefilterMem[c*maxPeriod:c*maxPeriod+maxPeriod])
		copy(pre[preBase[c]+maxPeriod:preBase[c]+maxPeriod+N], in[c*(N+overlap)+overlap:c*(N+overlap)+overlap+N])
		c++
		if c >= CC {
			break
		}
	}

	var pitchIndex int
	var gain1 int16

	// If the signal is dominated by a single tone, don't rely on the standard
	// pitch estimator (it can become unreliable).
	if enabled != 0 && toneishness > qconst0_99Q29 {
		multiple := 1
		// Using an aliased postfilter above 24 kHz. First value is purposely just
		// above pi to avoid triggering for Fs=48kHz.
		if toneFreq >= fixedmath.QCONST16(3.1416, 13) {
			toneFreq = fixedmath.QCONST16(3.141593, 13) - toneFreq
		}
		// If the pitch is too high for our post-filter, apply pitch doubling.
		for int(toneFreq) >= multiple*int(fixedmath.QCONST16(0.39, 13)) {
			multiple++
		}
		if toneFreq > fixedmath.QCONST16(0.006148, 13) {
			pitchIndex = fixedmath.IMIN((51472*multiple+int(toneFreq)/2)/int(toneFreq), combfilterMaxperiod-2)
		} else {
			// If the pitch is too low, a very high pitch actually helps due to the
			// filter's DC component being close to our tone.
			pitchIndex = combfilterMinperiod
		}
		gain1 = fixedmath.QCONST16(0.75, 15)
	} else if enabled != 0 && complexity >= 5 {
		pitchBuf := alloc(&st.sc.pitchBuf, (maxPeriod+N)>>1) // VARDECL(opus_val16, pitch_buf)
		// C's `const celt_sig *pre[2]` (:1441) is a stack pointer array; a fixed
		// 2-element field is its exact analogue and allocates nothing.
		preCh := st.sc.preCh[:]
		preCh[0] = pre[preBase[0]:]
		if CC == 2 {
			preCh[1] = pre[preBase[1]:]
		}
		pitchDownsample(preCh, pitchBuf, (maxPeriod+N)>>1, CC, 2, &st.sc)
		// Don't search the last 1.5 octave of the range (too many false-positives
		// from short-term correlation).
		pitchSearch(pitchBuf[maxPeriod>>1:], pitchBuf, N, maxPeriod-3*minPeriod, &pitchIndex, &st.sc)
		pitchIndex = maxPeriod - pitchIndex

		gain1 = removeDoubling(pitchBuf, maxPeriod, minPeriod, N, &pitchIndex,
			st.prefilterPeriod, st.prefilterGain, &st.sc)
		if pitchIndex > maxPeriod-2 {
			pitchIndex = maxPeriod - 2
		}
		gain1 = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.7, 15), gain1))
		if st.lossRate > 2 {
			gain1 = int16(fixedmath.HALF32(int32(gain1)))
		}
		if st.lossRate > 4 {
			gain1 = int16(fixedmath.HALF32(int32(gain1)))
		}
		if st.lossRate > 8 {
			gain1 = 0
		}
	} else {
		gain1 = 0
		pitchIndex = combfilterMinperiod
	}

	// Gain threshold for enabling the prefilter/postfilter.
	pfThreshold := fixedmath.QCONST16(0.2, 15)

	// Adjust the threshold based on rate and continuity.
	if pitchIabs(pitchIndex-st.prefilterPeriod)*10 > pitchIndex {
		pfThreshold += fixedmath.QCONST16(0.2, 15)
		// Completely disable the prefilter on strong transients without continuity.
		if tfEstimate > fixedmath.QCONST16(0.98, 14) {
			gain1 = 0
		}
	}
	if nbAvailableBytes < 25 {
		pfThreshold += fixedmath.QCONST16(0.1, 15)
	}
	if nbAvailableBytes < 35 {
		pfThreshold += fixedmath.QCONST16(0.1, 15)
	}
	if st.prefilterGain > fixedmath.QCONST16(0.4, 15) {
		pfThreshold -= fixedmath.QCONST16(0.1, 15)
	}
	if st.prefilterGain > fixedmath.QCONST16(0.55, 15) {
		pfThreshold -= fixedmath.QCONST16(0.1, 15)
	}

	// Hard threshold at 0.2.
	pfThreshold = fixedmath.EXTRACT16(fixedmath.MAX16(int32(pfThreshold), int32(fixedmath.QCONST16(0.2, 15))))
	var pfOn, qg int
	if gain1 < pfThreshold {
		gain1 = 0
		pfOn = 0
		qg = 0
	} else {
		// C computes gain1-st->prefilter_gain in int (both promoted), not int16, so
		// widen before ABS (celt_encoder.c:1533; unreachable divergence for gains in
		// [0, Q15ONE], but verbatim).
		if fixedmath.ABS32(int32(gain1)-int32(st.prefilterGain)) < int32(fixedmath.QCONST16(0.1, 15)) {
			gain1 = st.prefilterGain
		}
		qg = int(((int32(gain1)+1536)>>10)/3 - 1)
		qg = fixedmath.IMAX(0, fixedmath.IMIN(7, qg))
		gain1 = int16(int32(fixedmath.QCONST16(0.09375, 15)) * int32(qg+1))
		pfOn = 1
	}

	c = 0
	for {
		offset := mode.shortMdctSize - overlap
		st.prefilterPeriod = fixedmath.IMAX(st.prefilterPeriod, combfilterMinperiod)
		copy(in[c*(N+overlap):c*(N+overlap)+overlap], st.inMem[c*overlap:c*overlap+overlap])
		for i := 0; i < N; i++ {
			before[c] += fixedmath.ABS32(fixedmath.SHR32(in[c*(N+overlap)+overlap+i], 12))
		}
		if offset != 0 {
			combFilter(in, c*(N+overlap)+overlap, pre, preBase[c]+maxPeriod,
				st.prefilterPeriod, st.prefilterPeriod, offset,
				-st.prefilterGain, -st.prefilterGain,
				st.prefilterTapset, st.prefilterTapset, 0, nil)
		}
		combFilter(in, c*(N+overlap)+overlap+offset, pre, preBase[c]+maxPeriod+offset,
			st.prefilterPeriod, pitchIndex, N-offset,
			-st.prefilterGain, -gain1,
			st.prefilterTapset, prefilterTapset, overlap, mode.window)
		for i := 0; i < N; i++ {
			after[c] += fixedmath.ABS32(fixedmath.SHR32(in[c*(N+overlap)+overlap+i], 12))
		}
		c++
		if c >= CC {
			break
		}
	}

	if CC == 2 {
		var thresh [2]int16
		thresh[0] = fixedmath.EXTRACT16(
			fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.25, 15), gain1)), before[0]) +
				fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.01, 15), before[1]))
		thresh[1] = fixedmath.EXTRACT16(
			fixedmath.MULT16_32_Q15(int16(fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.25, 15), gain1)), before[1]) +
				fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.01, 15), before[0]))
		// Don't use the filter if one channel gets significantly worse.
		if after[0]-before[0] > int32(thresh[0]) || after[1]-before[1] > int32(thresh[1]) {
			cancelPitch = 1
		}
		// Use the filter only if at least one channel gets significantly better.
		if before[0]-after[0] < int32(thresh[0]) && before[1]-after[1] < int32(thresh[1]) {
			cancelPitch = 1
		}
	} else {
		// Check that the mono channel actually got better.
		if after[0] > before[0] {
			cancelPitch = 1
		}
	}
	// If needed, revert to a gain of zero.
	if cancelPitch != 0 {
		c = 0
		for {
			offset := mode.shortMdctSize - overlap
			copy(in[c*(N+overlap)+overlap:c*(N+overlap)+overlap+N], pre[preBase[c]+maxPeriod:preBase[c]+maxPeriod+N])
			combFilter(in, c*(N+overlap)+overlap+offset, pre, preBase[c]+maxPeriod+offset,
				st.prefilterPeriod, pitchIndex, overlap,
				-st.prefilterGain, 0,
				st.prefilterTapset, prefilterTapset, overlap, mode.window)
			c++
			if c >= CC {
				break
			}
		}
		gain1 = 0
		pfOn = 0
		qg = 0
	}

	c = 0
	for {
		copy(st.inMem[c*overlap:c*overlap+overlap], in[c*(N+overlap)+N:c*(N+overlap)+N+overlap])
		if N > maxPeriod {
			copy(prefilterMem[c*maxPeriod:c*maxPeriod+maxPeriod], pre[preBase[c]+N:preBase[c]+N+maxPeriod])
		} else {
			// OPUS_MOVE (overlapping): shift the tail forward, then append the new N.
			copy(prefilterMem[c*maxPeriod:c*maxPeriod+maxPeriod-N], prefilterMem[c*maxPeriod+N:c*maxPeriod+maxPeriod])
			copy(prefilterMem[c*maxPeriod+maxPeriod-N:c*maxPeriod+maxPeriod], pre[preBase[c]+maxPeriod:preBase[c]+maxPeriod+N])
		}
		c++
		if c >= CC {
			break
		}
	}

	*gain = gain1
	*pitch = pitchIndex
	*qgain = qg
	return pfOn
}

// =============================================================================
// TODO(CP8b/CP8c) boundary: the driver stages that follow compute_mdcts /
// normalise_bands in celt_encode_with_ec are NOT ported here.
//   CP8b (analysis):  tf_analysis, tf_encode, spreading_decision wiring,
//                     dynalloc_analysis, alloc_trim_analysis, stereo_analysis.
//   CP8c (coding):    quant_coarse_energy, compute_vbr, clt_compute_allocation,
//                     quant_all_bands, anti-collapse, quant_energy_finalise,
//                     the celt_encode_with_ec top-level assembly and the
//                     post-frame state updates (prefilter_period/gain/tapset,
//                     consec_transient, oldBandE/oldLogE/energyError, rng).
// =============================================================================

// --- Differential-test entry points (used by internal/reftest/oracle) --------

// CeltPreemphasis runs celt_preemphasis over one channel of interleaved int16 PCM
// (stride CC) at the 48 kHz mode's preemph coefficients and returns the N celt_sig
// output plus the updated filter memory.
func CeltPreemphasis(pcm []int16, N, CC, upsample int, mem int32) ([]int32, int32) {
	m := &mode48000_960
	inp := make([]int32, N)
	celtPreemphasis(pcm, 0, inp, N, CC, upsample, m.preemph[:], &mem, 0)
	return inp, mem
}

// TransientAnalysis runs transient_analysis over in (C*length celt_sig) and
// returns is_transient, tf_estimate, tf_chan and the weak_transient flag.
func TransientAnalysis(in []int32, length, C, allowWeakTransients int, toneFreq int16, toneishness int32) (isTransient int, tfEstimate int16, tfChan, weakTransient int) {
	var sc scratch
	isTransient = transientAnalysis(in, length, C, &tfEstimate, &tfChan, allowWeakTransients, &weakTransient, toneFreq, toneishness, &sc)
	return
}

// PatchTransientDecision runs patch_transient_decision and returns its boolean
// result as 0/1.
func PatchTransientDecision(newE, oldE []int32, nbEBands, start, end, C int) int {
	return patchTransientDecision(newE, oldE, nbEBands, start, end, C)
}

// ComputeMdcts runs compute_mdcts over in (CC*(N+overlap) celt_sig) and returns
// the CC*N interleaved frequency-domain output.
func ComputeMdcts(shortBlocks int, in []int32, C, CC, LM, upsample int) []int32 {
	m := &mode48000_960
	N := m.shortMdctSize << LM
	out := make([]int32, CC*N)
	var sc scratch
	computeMdcts(m, shortBlocks, in, out, C, CC, LM, upsample, &sc)
	return out
}

// ToneDetect runs tone_detect over in (CC*N celt_sig) and returns the tone
// frequency and the toneishness.
func ToneDetect(in []int32, CC, N int) (int16, int32) {
	m := &mode48000_960
	var toneishness int32
	var sc scratch
	freq := toneDetect(in, CC, N, &toneishness, m.Fs, &sc)
	return freq, toneishness
}

// SetPrefilterState seeds the previous-frame post-filter parameters that
// run_prefilter reads (the driver sets these from the prior frame's output).
func (st *Encoder) SetPrefilterState(period int, gain int16, tapset int) {
	st.prefilterPeriod = period
	st.prefilterGain = gain
	st.prefilterTapset = tapset
}

// PrefilterPeriod / PrefilterGain expose the (possibly mutated) post-filter state
// after run_prefilter.
func (st *Encoder) PrefilterPeriod() int { return st.prefilterPeriod }
func (st *Encoder) PrefilterGain() int16 { return st.prefilterGain }

// InMem returns the encoder's overlap memory slice (channels*overlap), so the
// test can read the cross-frame state run_prefilter updated.
func (st *Encoder) InMem() []int32 { return st.inMem }

// PrefilterMem returns the encoder's prefilter history (channels*COMBFILTER_MAXPERIOD),
// which is the buffer the driver passes to run_prefilter as prefilter_mem.
func (st *Encoder) PrefilterMem() []int32 { return st.prefilterMem }

// RunPrefilter runs run_prefilter with the encoder's current state, mutating in
// and prefilterMem in place, and returns the pitch index, quantized gain, tapset
// gain code and post-filter enable flag.
func (st *Encoder) RunPrefilter(in, prefilterMem []int32, CC, N, prefilterTapset, enabled, complexity int,
	tfEstimate int16, nbAvailableBytes int, toneFreq int16, toneishness int32) (pitch int, gain int16, qgain, pfOn int) {
	pfOn = st.runPrefilter(in, prefilterMem, CC, N, prefilterTapset, &pitch, &gain, &qgain,
		enabled, complexity, tfEstimate, nbAvailableBytes, toneFreq, toneishness)
	return
}

// --- Cross-frame state snapshot (mirrors celt_decoder.go State/Hash) ----------

// EncoderState is a copyable snapshot of the encoder's persistent (reset-region)
// state. It is hashed for the per-frame differential comparison, exactly like the
// decoder's celt.State.
//
// CANONICAL STATE ORDER (the contract shared with the C-side dump in
// internal/reftest/oracle/celtenc_shim.h, oracle_celtenc_h_dump_state). It is the
// declaration order of struct OpusCustomEncoder's reset region
// (celt_encoder.c:91-127), skipping the fields that are dead in the frozen
// CELT-only config (analysis, silk_info) and the energy_mask POINTER (an input,
// not evolved state):
//
//	 1 rng              (u32)
//	 2 spread_decision  (i32)
//	 3 delayedIntra     (i32, opus_val32)
//	 4 tonal_average    (i32)
//	 5 lastCodedBands   (i32)
//	 6 hf_average       (i32)
//	 7 tapset_decision  (i32)
//	 8 prefilter_period (i32)
//	 9 prefilter_gain   (i16, opus_val16)
//	10 prefilter_tapset (i32)
//	11 consec_transient (i32)
//	12 preemph_memE[0]  (i32)
//	13 preemph_memE[1]  (i32)
//	14 preemph_memD[0]  (i32)
//	15 preemph_memD[1]  (i32)
//	16 vbr_reservoir    (i32)
//	17 vbr_drift        (i32)
//	18 vbr_offset       (i32)
//	19 vbr_count        (i32)
//	20 overlap_max      (i32, opus_val32)
//	21 stereo_saving    (i16, opus_val16)
//	22 intensity        (i32)
//	23 spec_avg         (i32, celt_glog)
//	24 in_mem[]         (i32 x channels*overlap)
//	25 prefilter_mem[]  (i32 x channels*COMBFILTER_MAXPERIOD)
//	26 oldBandE[]       (i32 x channels*nbEBands)
//	27 oldLogE[]        (i32 x channels*nbEBands)
//	28 oldLogE2[]       (i32 x channels*nbEBands)
//	29 energyError[]    (i32 x channels*nbEBands)
//
// Fields 16-22 (the VBR reservoir/drift/offset/count, stereo_saving and
// intensity) are mutated by celt_encode_with_ec and persist across frames; they
// were missing from the snapshot until CP8c, which meant a broken VBR loop could
// pass a state-hash comparison unnoticed.
type EncoderState struct {
	Rng             uint32
	SpreadDecision  int
	DelayedIntra    int32
	TonalAverage    int
	LastCodedBands  int
	HfAverage       int
	TapsetDecision  int
	PrefilterPeriod int
	PrefilterGain   int16
	PrefilterTapset int
	ConsecTransient int
	PreemphMemE     [2]int32
	PreemphMemD     [2]int32
	VbrReservoir    int32
	VbrDrift        int32
	VbrOffset       int32
	VbrCount        int32
	OverlapMax      int32
	StereoSaving    int16
	Intensity       int
	SpecAvg         int32
	InMem           []int32
	PrefilterMem    []int32
	OldBandE        []int32
	OldLogE         []int32
	OldLogE2        []int32
	EnergyError     []int32
}

// State returns a deep copy of the encoder's persistent state.
func (st *Encoder) State() EncoderState {
	return EncoderState{
		Rng:             st.rng,
		SpreadDecision:  st.spreadDecision,
		DelayedIntra:    st.delayedIntra,
		TonalAverage:    st.tonalAverage,
		LastCodedBands:  st.lastCodedBands,
		HfAverage:       st.hfAverage,
		TapsetDecision:  st.tapsetDecision,
		PrefilterPeriod: st.prefilterPeriod,
		PrefilterGain:   st.prefilterGain,
		PrefilterTapset: st.prefilterTapset,
		ConsecTransient: st.consecTransient,
		PreemphMemE:     st.preemphMemE,
		PreemphMemD:     st.preemphMemD,
		VbrReservoir:    st.vbrReservoir,
		VbrDrift:        st.vbrDrift,
		VbrOffset:       st.vbrOffset,
		VbrCount:        st.vbrCount,
		OverlapMax:      st.overlapMax,
		StereoSaving:    st.stereoSaving,
		Intensity:       st.intensity,
		SpecAvg:         st.specAvg,
		InMem:           append([]int32(nil), st.inMem...),
		PrefilterMem:    append([]int32(nil), st.prefilterMem...),
		OldBandE:        append([]int32(nil), st.oldBandE...),
		OldLogE:         append([]int32(nil), st.oldLogE...),
		OldLogE2:        append([]int32(nil), st.oldLogE2...),
		EnergyError:     append([]int32(nil), st.energyError...),
	}
}

// Hash returns an FNV-1a-64 hash over a canonical little-endian serialization of
// the persistent state (the CANONICAL STATE ORDER documented on EncoderState),
// matching the decoder State.Hash convention so a C-side dump built into the same
// struct hashes identically.
func (s EncoderState) Hash() uint64 {
	h := fnv.New64a()
	var b [8]byte
	putU32 := func(v uint32) {
		b[0] = byte(v)
		b[1] = byte(v >> 8)
		b[2] = byte(v >> 16)
		b[3] = byte(v >> 24)
		_, _ = h.Write(b[:4])
	}
	putI32 := func(v int32) { putU32(uint32(v)) }
	putI := func(v int) { putU32(uint32(int32(v))) }
	putI16 := func(v int16) {
		b[0] = byte(uint16(v))
		b[1] = byte(uint16(v) >> 8)
		_, _ = h.Write(b[:2])
	}

	putU32(s.Rng)
	putI(s.SpreadDecision)
	putI32(s.DelayedIntra)
	putI(s.TonalAverage)
	putI(s.LastCodedBands)
	putI(s.HfAverage)
	putI(s.TapsetDecision)
	putI(s.PrefilterPeriod)
	putI16(s.PrefilterGain)
	putI(s.PrefilterTapset)
	putI(s.ConsecTransient)
	putI32(s.PreemphMemE[0])
	putI32(s.PreemphMemE[1])
	putI32(s.PreemphMemD[0])
	putI32(s.PreemphMemD[1])
	putI32(s.VbrReservoir)
	putI32(s.VbrDrift)
	putI32(s.VbrOffset)
	putI32(s.VbrCount)
	putI32(s.OverlapMax)
	putI16(s.StereoSaving)
	putI(s.Intensity)
	putI32(s.SpecAvg)
	for _, v := range s.InMem {
		putI32(v)
	}
	for _, v := range s.PrefilterMem {
		putI32(v)
	}
	for _, v := range s.OldBandE {
		putI32(v)
	}
	for _, v := range s.OldLogE {
		putI32(v)
	}
	for _, v := range s.OldLogE2 {
		putI32(v)
	}
	for _, v := range s.EnergyError {
		putI32(v)
	}
	return h.Sum64()
}
