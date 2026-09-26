package celt

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
)

// This file assembles the CELT encoder driver, celt_encode_with_ec
// (celt/celt_encoder.c:1725-2832, libopus v1.6.1). Everything it calls is
// already ported: the front half (CP8a: celt_preemphasis, tone_detect,
// transient_analysis, patch_transient_decision, run_prefilter, compute_mdcts),
// the analysis stages (CP8b: tf_analysis, tf_encode, dynalloc_analysis,
// alloc_trim_analysis, stereo_analysis, spreading_decision), the coding stages
// (quant_coarse_energy, clt_compute_allocation, quant_fine_energy,
// quant_all_bands, quant_energy_finalise) and compute_vbr (CP8c wave 2). Only
// the top-level assembly, the inline surround-masking and temporal-VBR blocks,
// and the inline VBR rate-control loop are new here.
//
// THE ec_enc PARAMETER: both C call shapes are ported. Encode passes enc == NULL
// (own ec_enc over `compressed`; tell0_frac = tell = 1, nbFilledBytes = 0 at
// :1862), which is how opus_custom_encode drives CELT. EncodeWithEC passes an
// external coder with compressed == NULL, which is how opus_encode_frame_native
// drives it (opus_encoder.c:2493), taking the :1865-1867 and :1919-1920 branches.
//
// Gated out of this build (do NOT look for them here):
//
//   - The whole `#ifdef RESYNTH` block (:2724-2776). There is therefore NO
//     celt_synthesis, NO anti_collapse CALL, NO comb_filter and NO deemphasis in
//     the encoder tail, and st->prefilter_*_old are never written. The ONLY thing
//     the encoder needs from anti-collapse is the DECISION BIT at :2697-2704:
//     anti_collapse_on = st->consec_transient < 2, coded with ec_enc_bits(...,1)
//     when anti_collapse_rsv > 0.
//   - The CUSTOM_MODES / ENABLE_OPUS_CUSTOM_API blocks at :1870-1896 and
//     :2822-2825: no signalling byte is written and nbCompressedBytes is not
//     incremented. BUT st->signalling is still 1 (celt_encoder.c:211) and is LIVE
//     at :1918 through `-!!st->signalling`, which shaves one byte off the CBR
//     size. That is not a simplifiable no-op.
//   - ENABLE_QEXT everywhere (ARG_QEXT expands to nothing, QEXT_SCALE is the
//     identity): no ext_enc, no qext_bytes, no padding, no
//     clt_compute_extra_allocation. In particular the ec_enc_shrink at :2000 and
//     the second compute_vbr call at :2545 are dead.
//   - DISABLE_FLOAT_API: every st->analysis / AnalysisInfo path. st->analysis.valid
//     is permanently 0, so the min_bandwidth block at :2607-2623 is dead
//     (signalBandwidth stays end-1) and the analysis half of the pitch_change
//     condition at :2042 collapses to `!st->analysis.valid` == true.
//   - st->silk_info: hybrid-only. signalType stays 0 and offset stays 0.
//   - `need_clip` (:2011) is always 0: opus_res is opus_int16, so
//     sample_max <= 32767 < 65536<<RES_SHIFT.
//   - `#ifdef FUZZING` (the random silence and random anti_collapse_on overrides).
//   - The "last chance to catch a transient" recompute at :2211-2226 is present
//     but UNREACHABLE: its guard, patch_transient_decision, is structurally dead
//     in this config. At celt_encoder.c:498-501 the per-band terms are declared
//     `opus_val16 x1, x2` and fed MAXG(0, <celt_glog>), so with 32-bit celt_glog
//     each truncates to 16 bits and is capped at ~2^15; their average therefore
//     can never exceed GCONST(1.f) == 2^24 (:506) and the function always returns
//     0. That is an upstream bug and it is reproduced, not fixed. The differential
//     test asserts the branch stays dead, so a future libopus fix cannot slip
//     through untested.
//
// Only THREE of the four ec_enc_shrink call sites in the C function are live on
// this path: :1957 (the constrained-VBR bust-prevention clamp), :1993 (the
// silence path in VBR) and :2532 (the end of the VBR rate loop). The one at
// :1920 sits inside `if (enc != NULL)` and can never run here. The nbits_total
// bump at :2006 IS live and is spelled Encoder.SetTellForSilence.
//
// Memory is Layer A (idiomatic Go slices instead of the C VARDECL/ALLOC stack),
// but every arithmetic and range-coder statement is Layer C: transliterated
// verbatim, in C's order, including the truncations. The RANGE-CODER SYMBOL ORDER
// IS THE BITSTREAM; it is, in order:
//
//	 1. silence bit           ec_enc_bit_logp(silence, 15)              :1981
//	 2. prefilter on/off      ec_enc_bit_logp(pf_on, 1)                 :2045-2061
//	                          + octave / pitch / qg / tapset if on
//	 3. isTransient           ec_enc_bit_logp(isTransient, 3)           :2234
//	 4. coarse energy         quant_coarse_energy                       :2295
//	 5. tf_encode                                                       :2300
//	 6. spread decision       ec_enc_icdf(spread_decision, 5)           :2345
//	 7. dynalloc boost loop   ec_enc_bit_logp per iteration             :2356-2389
//	 8. alloc trim            ec_enc_icdf(alloc_trim, 7)                :2420
//	 9. clt_compute_allocation (skip / intensity / dual_stereo bits)    :2626
//	10. quant_fine_energy                                               :2634
//	11. quant_all_bands                                                 :2670
//	12. anti-collapse bit     ec_enc_bits(anti_collapse_on, 1)          :2703
//	13. quant_energy_finalise                                           :2706
//	14. ec_enc_done                                                     :2813

// opusBadArg / opusInternalError are OPUS_BAD_ARG (-1) and OPUS_INTERNAL_ERROR
// (-3) from opus_defines.h:48,52; celt_encode_with_ec returns them in place of a
// packet length.
const (
	opusBadArg        = -1
	opusInternalError = -3
)

// packetSizeCap is the 1275-byte main-payload cap (celt_encoder.c:1801). Without
// ENABLE_QEXT it is never raised to QEXT_PACKET_SIZE_CAP.
const packetSizeCap = 1275

// GCONST(x) == QCONST32(x, DB_SHIFT) constants used by the driver's inline
// blocks that are not already defined elsewhere in the package (gconst1 in
// celt_encoder.go; gconst3 / gconst28 / ghalf in quant_bands.go; gconst1_5 /
// gconst2 in celt_decoder.go).
var (
	gconstQuarter = fixedmath.QCONST32(0.25, dbShift)  // GCONST(.25f)
	gconstFifth   = fixedmath.QCONST32(0.2, dbShift)   // GCONST(.2f)
	gconst031     = fixedmath.QCONST32(0.031, dbShift) // GCONST(.031f)
)

// intensityThresholds / intensityHisteresis are the static tables at
// celt_encoder.c:2393-2398 (opus_val16), indexed by equiv_rate/1000 through
// hysteresis_decision to pick st->intensity. C==2 only.
var (
	intensityThresholds = [21]int16{1, 2, 3, 4, 5, 6, 7, 8, 16, 24, 36, 44, 50, 56, 62, 67, 72, 79, 88, 106, 134}
	intensityHisteresis = [21]int16{1, 1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 3, 3, 4, 5, 6, 8, 8}
)

// EncodeWitness records the branch/clamp decisions celt_encode_with_ec took on
// one frame. It is pure observation for the differential test's non-vacuity
// guards: nothing recorded here feeds back into the coded output, and production
// callers pass a nil witness.
type EncodeWitness struct {
	// Frame geometry / rate control.
	LM                int
	NbCompressedBytes int // final, i.e. the return value on success
	NbAvailableBytes  int
	EffectiveBytes    int
	EquivRate         int32
	VbrRate           int32
	Target            int32 // the VBR target after :2477 (0 when !vbr)

	// Analysis decisions.
	Silence              bool // :1973
	IsTransient          bool // after the :2213 patch, i.e. what :2235 coded
	WeakTransient        bool // :2030
	TransientGotDisabled bool // :2069
	ShortBlocks          int  // :2067
	SecondMdct           bool // :2076
	PatchedTransient     bool // :2215 fired; STRUCTURALLY DEAD, see the note below
	LfeBandEClamp        bool // :2098 (st->lfe band-energy clamp)
	PfEnabled            bool // :2038
	PfOn                 bool // :2041
	PitchChange          bool // :2042; INERT (compute_vbr does (void)pitch_change)
	EnableTfAnalysis     bool // :2242
	TfSelect             int  // :2213 / the forced values below it
	TfRes                []int
	SpreadDecision       int
	SpreadNoRoom         bool // the :2347 else-branch (no room to code a spread)
	SpreadFromDecision   bool // spreading_decision() itself ran (:2337)
	MaxDepth             int32
	TotBoost             int32 // dynalloc_analysis' *tot_boost_
	TotalBoost           int32 // the boost actually CODED by the :2360 loop
	DynallocBoosted      bool  // at least one boost bit was coded as 1
	AllocTrim            int
	AllocTrimCoded       bool // the :2409 budget check passed
	AllocTrimFromLfe     bool // the start>0 || lfe branch at :2412
	Intensity            int
	IntensityChanged     bool
	DualStereo           bool
	CodedBands           int
	AntiCollapseRsv      int
	AntiCollapseOn       bool
	CoarseIntra          bool // quant_coarse_energy settled on intra

	// Surround / temporal VBR.
	SurroundBlock    bool // :2111 ran (energy_mask != NULL)
	TemporalVbrBlock bool // :2187 ran (!lfe)
	SurroundMasking  int32
	SurroundTrim     int32
	TemporalVbr      int32

	// Range-coder resizes.
	CbrClamped           bool // the :1917 CBR nbCompressedBytes clamp lowered it
	ConstrainedVbrShrink bool // :1957
	SilenceVbrShrink     bool // :1993
	VbrShrink            bool // :2532 actually lowered nbCompressedBytes
	VbrReservoirClamped  bool // :2520 (vbr_reservoir < 0 -> refund)
	VbrCountSaturated    bool // :2508 (vbr_count hit its 970 ceiling)
	MinAllowedWon        bool // min_allowed won the IMAX at :2480
}

// Encode is celt_encode_with_ec (celt_encoder.c:1725) with enc == NULL: encode
// one frame of interleaved int16 PCM (frameSize samples per PHYSICAL channel CC)
// into compressed[0:nbCompressedBytes], and return the packet length in bytes or
// a negative OPUS_* error.
//
// compressed must be at least nbCompressedBytes long. The range coder's own
// ec_enc_done zeroes every byte it did not write inside its (possibly shrunk)
// storage window, so the caller does not have to pre-zero the buffer; bytes past
// the returned length are not part of the packet.
func (st *Encoder) Encode(pcm []int16, frameSize int, compressed []byte, nbCompressedBytes int) int {
	return st.encodeObserved(pcm, frameSize, compressed, nbCompressedBytes, nil, nil)
}

// EncodeWithEC is celt_encode_with_ec (celt_encoder.c:1725) with an EXTERNAL
// range coder, i.e. the `enc != NULL` / `compressed == NULL` call shape that
// opus_encode_frame_native uses at opus_encoder.c:2493. ec must already be
// Init'ed over the packet buffer (the Opus layer does ec_enc_init at :1964 and
// ec_enc_shrink at :2413 before calling), and this function writes the CELT
// payload into it, continuing from wherever it is.
//
// vs. Encode (enc == NULL), only three things change, all of them in C:
//
//   - :1860-1868 tell0_frac / tell / nbFilledBytes come from the coder instead of
//     the 1 / 1 / 0 constants.
//   - :1919-1920 the CBR nbCompressedBytes clamp additionally ec_enc_shrink()s.
//   - :1929-1933 no ec_enc_init: the caller's coder is used as-is.
//
// On the frozen CELT-only Opus path the two shapes are NUMERICALLY EQUIVALENT
// (a fresh coder has ec_tell()==1, ec_tell_frac()==8 -> nbFilledBytes==0; tell0_frac
// is read only in the hybrid branch at :2433; and the CBR clamp needs
// bitrate != OPUS_BITRATE_MAX, which opus_encode_frame_native never leaves).
// EncodeWithEC exists so the port does not DEPEND on that equivalence.
//
// Not safe for concurrent use: one Encoder per goroutine. The encoder state and
// its pooled scratch (see scratch.go) are mutated without synchronization.
func (st *Encoder) EncodeWithEC(pcm []int16, frameSize int, ec *rangecoding.Encoder, nbCompressedBytes int) int {
	return st.encodeObserved(pcm, frameSize, nil, nbCompressedBytes, ec, nil)
}

// encodeObserved is the implementation of Encode / EncodeWithEC with an optional
// (nil-able) witness hook for the differential test. extEnc is the C `ec_enc *enc`
// argument: nil reproduces `enc == NULL` (own coder over `compressed`), non-nil
// reproduces an external coder (and then `compressed` is unused, as it is NULL in C).
//
//nolint:gocyclo,gocognit,maintidx // Verbatim transliteration of a 1100-line C function; splitting it would break the 1:1 mapping the differential oracle depends on.
func (st *Encoder) encodeObserved(pcm []int16, frameSize int, compressed []byte, nbCompressedBytes int, extEnc *rangecoding.Encoder, w *EncodeWitness) int {
	var i, c, N int
	var bits int32
	// C declares `ec_enc _enc;` (:1730) and only uses it when enc == NULL (:1931).
	// `enc` below is the C parameter of the same name: it aliases either the
	// caller's coder or our own storage, so every enc.* call site stays verbatim.
	var ownEnc rangecoding.Encoder
	enc := &ownEnc
	if extEnc != nil {
		enc = extEnc
	}
	// C spells these `int shortBlocks=0; int isTransient=0;` (:1747-1748), but
	// those initialisers are dead: both are unconditionally re-zeroed at :2022-2023
	// before anything reads them.
	var shortBlocks int
	var isTransient int
	CC := st.channels
	C := st.streamChannels
	var LM, M int
	var tfSelect int
	var nbFilledBytes, nbAvailableBytes int
	var minAllowed int32
	start := st.start
	end := st.end
	var effEnd int
	var codedBands int
	var allocTrim int
	pitchIndex := combfilterMinperiod
	var gain1 int16
	dualStereo := 0
	var effectiveBytes int
	var dynallocLogp int
	var vbrRate int32
	var totalBits int32
	var totalBoost int32
	var balance int32
	var tell int32
	var tell0Frac int32
	// C's `int prefilter_tapset=0;` (:1771) is likewise dead: :2040 overwrites it
	// with st->tapset_decision before any read.
	var prefilterTapset int
	var pfOn int
	antiCollapseRsv := 0
	antiCollapseOn := 0
	silence := 0
	tfChan := 0
	var tfEstimate int16
	var totBoost int32
	var sampleMax int32
	var maxDepth int32
	m := st.mode
	nbEBands := m.nbEBands
	overlap := m.overlap
	eBands := m.eBands
	secondMdct := 0
	var signalBandwidth int
	transientGotDisabled := 0
	surroundMasking := int32(0)
	temporalVbr := int32(0)
	surroundTrim := int32(0)
	var equivRate int32
	weakTransient := 0
	// C's `opus_val16 tone_freq=-1;` (:1797) is dead too: tone_detect (:2021)
	// assigns it before any read. The -1 sentinel it carries is tone_detect's OWN
	// "not tonal" return, which is where the value actually comes from.
	var toneFreq int16
	toneishness := int32(0)

	// hybrid == (start != 0). Unreachable on the frozen single-stream CELT-only
	// path (only the Opus hybrid mode drives CELT with CELT_SET_START_BAND=17),
	// but the branches are ported so that setting start>0 behaves like C.
	hybrid := 0
	if start != 0 {
		hybrid = 1
	}
	tfEstimate = 0
	if nbCompressedBytes < 2 || pcm == nil {
		return opusBadArg
	}

	frameSize *= st.upsample
	for LM = 0; LM <= m.maxLM; LM++ {
		if m.shortMdctSize<<LM == frameSize {
			break
		}
	}
	if LM > m.maxLM {
		return opusBadArg
	}
	M = 1 << LM
	N = M * m.shortMdctSize

	// The C carves prefilter_mem / oldBandE / oldLogE / oldLogE2 / energyError out
	// of the single trailing in_mem VLA (:1854-1858); here they are already
	// separate slices on the Encoder.
	prefilterMem := st.prefilterMem
	oldBandE := st.oldBandE
	oldLogE := st.oldLogE
	oldLogE2 := st.oldLogE2
	energyError := st.energyError

	if extEnc == nil {
		// enc == NULL (:1862).
		tell0Frac = 1
		tell = 1
		nbFilledBytes = 0
	} else {
		// enc != NULL (:1865-1867).
		tell0Frac = int32(enc.TellFrac())
		tell = int32(enc.Tell())
		nbFilledBytes = (int(tell) + 4) >> 3
	}

	// The CUSTOM_MODES signalling-byte block (:1870-1896) is gated out; st->end,
	// compressed and nbCompressedBytes are left alone.

	// Can't produce more than 1275 output bytes for the main payload.
	nbCompressedBytes = fixedmath.IMIN(nbCompressedBytes, packetSizeCap)

	if st.vbr != 0 && st.bitrate != opusBitrateMax {
		vbrRate = BitrateToBits(st.bitrate, m.Fs, int32(frameSize)) << bitRes
		// The `if (st->signalling) vbr_rate -= 8<<BITRES;` at :1905 is CUSTOM_MODES.
		effectiveBytes = int(vbrRate >> (3 + bitRes))
	} else {
		var tmp int32
		vbrRate = 0
		tmp = st.bitrate * int32(frameSize)
		if tell > 1 {
			tmp += tell * m.Fs
		}
		if st.bitrate != opusBitrateMax {
			// -!!st->signalling: signalling is 1 (celt_encoder.c:211) even though the
			// signalling BYTE is never written in this build, so this really does
			// shave one byte off the CBR budget. Do not simplify it away.
			sig := 0
			if st.signalling != 0 {
				sig = 1
			}
			before := nbCompressedBytes
			nbCompressedBytes = fixedmath.IMAX(2, fixedmath.IMIN(nbCompressedBytes,
				int((tmp+4*m.Fs)/(8*m.Fs))-sig))
			if w != nil && nbCompressedBytes != before {
				w.CbrClamped = true
			}
			// :1919-1920, inside `if (enc != NULL)`.
			if extEnc != nil {
				enc.EncShrink(uint32(nbCompressedBytes))
			}
		}
		effectiveBytes = nbCompressedBytes - nbFilledBytes
	}
	nbAvailableBytes = nbCompressedBytes - nbFilledBytes
	equivRate = ((int32(nbCompressedBytes) * 8 * 50) << (3 - LM)) - int32((40*C+20)*((400>>LM)-50))
	if st.bitrate != opusBitrateMax {
		equivRate = fixedmath.MIN32(equivRate, st.bitrate-int32((40*C+20)*((400>>LM)-50)))
	}

	// :1929-1933. enc == NULL: run our own coder over the caller's buffer. With an
	// external coder the caller already Init'ed it (and possibly shrank it), so it
	// is used as-is.
	if extEnc == nil {
		enc.Init(compressed[:nbCompressedBytes])
	}

	if vbrRate > 0 {
		// Compute the max bit-rate allowed in VBR mode to avoid violating the target
		// rate and buffering. Up front, so bust-prevention triggers correctly.
		if st.constrainedVbr != 0 {
			var vbrBound, maxAllowed int32
			vbrBound = vbrRate
			lo := int32(0)
			if tell == 1 {
				lo = 2
			}
			maxAllowed = fixedmath.MIN32(fixedmath.MAX32(lo,
				(vbrRate+vbrBound-st.vbrReservoir)>>(bitRes+3)), int32(nbAvailableBytes))
			if maxAllowed < int32(nbAvailableBytes) {
				nbCompressedBytes = nbFilledBytes + int(maxAllowed)
				nbAvailableBytes = int(maxAllowed)
				enc.EncShrink(uint32(nbCompressedBytes))
				if w != nil {
					w.ConstrainedVbrShrink = true
				}
			}
		}
	}
	totalBits = int32(nbCompressedBytes * 8)

	effEnd = end
	if effEnd > m.effEBands {
		effEnd = m.effEBands
	}

	in := alloc(&st.sc.in, CC*(N+overlap)) // VARDECL(celt_sig, in)

	// NOTE the lengths use C (stream_channels), not CC (:1969-1971). With CC==2,
	// C==1 the C only scans half the interleaved buffer. Reproduced verbatim.
	sampleMax = fixedmath.MAX32(st.overlapMax, celtMaxabsRes(pcm, 0, C*(N-overlap)/st.upsample))
	st.overlapMax = celtMaxabsRes(pcm, C*(N-overlap)/st.upsample, C*overlap/st.upsample)
	sampleMax = fixedmath.MAX32(sampleMax, st.overlapMax)
	// FIXED_POINT: silence = (sample_max==0).
	if sampleMax == 0 {
		silence = 1
	}
	if tell == 1 {
		enc.EncBitLogp(silence, 15)
	} else {
		silence = 0
	}
	if silence != 0 {
		// In VBR mode there is no need to send more than the minimum.
		if vbrRate > 0 {
			nbCompressedBytes = fixedmath.IMIN(nbCompressedBytes, nbFilledBytes+2)
			effectiveBytes = nbCompressedBytes
			totalBits = int32(nbCompressedBytes * 8)
			nbAvailableBytes = 2
			enc.EncShrink(uint32(nbCompressedBytes))
			if w != nil {
				w.SilenceVbrShrink = true
			}
		}
		// Pretend we've filled all the remaining bits with zeros (that's what the
		// initialiser did anyway): enc->nbits_total += tell - ec_tell(enc) (:2006).
		tell = int32(nbCompressedBytes * 8)
		enc.SetTellForSilence(int(tell))
	}
	c = 0
	for {
		// need_clip is always 0 here (opus_res is int16, so sample_max <= 32767).
		needClip := 0
		celtPreemphasis(pcm, c, in[c*(N+overlap)+overlap:], N, CC, st.upsample,
			m.preemph[:], &st.preemphMemE[c], needClip)
		copy(in[c*(N+overlap):c*(N+overlap)+overlap],
			prefilterMem[(1+c)*combfilterMaxperiod-overlap:(1+c)*combfilterMaxperiod])
		c++
		if c >= CC {
			break
		}
	}

	toneFreq = toneDetect(in, CC, N+overlap, &toneishness, m.Fs, &st.sc)
	isTransient = 0
	shortBlocks = 0
	if st.complexity >= 1 && st.lfe == 0 {
		// Reduces the likelihood of energy instability on fricatives at low bitrate
		// in hybrid mode. silk_info.signalType is 0 on the CELT-only path.
		allowWeakTransients := 0
		if hybrid != 0 && effectiveBytes < 15 && st.silkSignalType != 2 {
			allowWeakTransients = 1
		}
		isTransient = transientAnalysis(in, N+overlap, CC, &tfEstimate, &tfChan,
			allowWeakTransients, &weakTransient, toneFreq, toneishness, &st.sc)
	}
	toneishness = fixedmath.MIN32(toneishness,
		fixedmath.QCONST32(1, 29)-fixedmath.SHL32(int32(tfEstimate), 15))

	// Find pitch period and gain.
	{
		var enabled int
		var qg int
		if ((st.lfe != 0 && nbAvailableBytes > 3) || nbAvailableBytes > 12*C) &&
			hybrid == 0 && silence == 0 && tell+16 <= totalBits && st.disablePf == 0 {
			enabled = 1
		}

		prefilterTapset = st.tapsetDecision
		pfOn = st.runPrefilter(in, prefilterMem, CC, N, prefilterTapset, &pitchIndex, &gain1, &qg,
			enabled, st.complexity, tfEstimate, nbAvailableBytes, toneFreq, toneishness)
		if w != nil {
			w.PfEnabled = enabled != 0
			// pitch_change (:2042-2044). st->analysis.valid is 0 under
			// DISABLE_FLOAT_API, so the analysis half of the condition is always true.
			// The value is INERT: its only consumer is compute_vbr, which reduces it
			// to `(void)pitch_change;` (:1672). Recorded for the witness only.
			if (gain1 > fixedmath.QCONST16(0.4, 15) || st.prefilterGain > fixedmath.QCONST16(0.4, 15)) &&
				(float64(pitchIndex) > 1.26*float64(st.prefilterPeriod) ||
					float64(pitchIndex) < 0.79*float64(st.prefilterPeriod)) {
				w.PitchChange = true
			}
		}
		if pfOn == 0 {
			if hybrid == 0 && tell+16 <= totalBits {
				enc.EncBitLogp(0, 1)
			}
		} else {
			// This block is not gated by a total-bits check only because of the
			// nbAvailableBytes check above.
			var octave int
			enc.EncBitLogp(1, 1)
			pitchIndex++
			octave = fixedmath.EC_ILOG(uint32(pitchIndex)) - 5
			enc.EncUint(uint32(octave), 6)
			enc.EncBits(uint32(pitchIndex-(16<<octave)), uint32(4+octave))
			pitchIndex--
			enc.EncBits(uint32(qg), 3)
			enc.EncIcdf(prefilterTapset, tapsetIcdf, 2)
		}
	}

	if LM > 0 && int32(enc.Tell())+3 <= totalBits {
		if isTransient != 0 {
			shortBlocks = M
		}
	} else {
		isTransient = 0
		transientGotDisabled = 1
	}

	freq := alloc(&st.sc.freq, CC*N) // VARDECL(celt_sig, freq): interleaved signal MDCTs
	bandE := alloc(&st.sc.bandE, nbEBands*CC)
	bandLogE := alloc(&st.sc.bandLogE, nbEBands*CC)

	if shortBlocks != 0 && st.complexity >= 8 {
		secondMdct = 1
	}
	bandLogE2 := alloc(&st.sc.bandLogE2, C*nbEBands)
	if secondMdct != 0 {
		computeMdcts(m, 0, in, freq, C, CC, LM, st.upsample, &st.sc)
		computeBandEnergies(m, freq, bandE, effEnd, C, LM, st.arch)
		amp2Log2(m, effEnd, end, bandE, bandLogE2, C)
		for c = 0; c < C; c++ {
			for i = 0; i < end; i++ {
				bandLogE2[nbEBands*c+i] += fixedmath.HALF32(fixedmath.SHL32(int32(LM), dbShift))
			}
		}
	}

	computeMdcts(m, shortBlocks, in, freq, C, CC, LM, st.upsample, &st.sc)
	// The celt_isnan assert at :2072 is a no-op in fixed point.
	if CC == 2 && C == 1 {
		tfChan = 0
	}
	computeBandEnergies(m, freq, bandE, effEnd, C, LM, st.arch)

	if st.lfe != 0 {
		for i = 2; i < end; i++ {
			bandE[i] = fixedmath.MIN32(bandE[i],
				fixedmath.MULT16_32_Q15(fixedmath.QCONST16(1e-4, 15), bandE[0]))
			bandE[i] = fixedmath.MAX32(bandE[i], 1) // EPSILON
		}
		if w != nil {
			w.LfeBandEClamp = true
		}
	}
	amp2Log2(m, effEnd, end, bandE, bandLogE, C)

	// ALLOC(surround_dynalloc, C*nbEBands) + OPUS_CLEAR(surround_dynalloc, end).
	//
	// THE CLEAR IS LOAD-BEARING and must not be dropped now that the buffer is
	// pooled rather than make()'d. C clears it explicitly (celt_encoder.c:2107); the
	// energy_mask writer below is structurally dead on this single-stream path, so
	// the clear is the ONLY write, and dynalloc_analysis reads [start,end). A pooled
	// buffer without it would feed the previous frame's values into follower[] and
	// move packet bytes. Clearing all C*nbEBands (a superset of C's `end`) keeps the
	// exact semantics the make() had, for 42 words.
	surroundDynalloc := alloc(&st.sc.surroundDynalloc, C*nbEBands)
	for i = 0; i < C*nbEBands; i++ {
		surroundDynalloc[i] = 0
	}

	// This computes how much masking takes place between surround channels
	// (:2108-2185). STRUCTURALLY DEAD on the frozen single-stream CELT path:
	// st->energy_mask is only ever set by OPUS_SET_ENERGY_MASK, which only the
	// multistream/surround encoder issues. Ported verbatim behind the same guard,
	// so that when a caller DOES install a mask the behaviour is identical; when
	// energyMask is nil, surround_masking / surround_trim stay 0 and
	// surround_dynalloc stays all-zero, exactly as C leaves them.
	if hybrid == 0 && st.energyMask != nil && st.lfe == 0 {
		var maskEnd, midband, countDynalloc int
		var maskAvg, diff int32
		count := 0
		maskEnd = fixedmath.IMAX(2, st.lastCodedBands)
		for c = 0; c < C; c++ {
			for i = 0; i < maskEnd; i++ {
				var mask int32
				var mask16 int16
				mask = fixedmath.MAX32(fixedmath.MIN32(st.energyMask[nbEBands*c+i],
					gconstQuarter), -gconst2)
				if mask > 0 {
					mask = fixedmath.HALF32(mask)
				}
				mask16 = int16(fixedmath.SHR32(mask, dbShift-10))
				maskAvg += fixedmath.MULT16_16(mask16, int16(eBands[i+1]-eBands[i]))
				count += int(eBands[i+1] - eBands[i])
				diff += fixedmath.MULT16_16(mask16, int16(1+2*i-maskEnd))
			}
		}
		// celt_assert(count>0)
		maskAvg = fixedmath.SHL32(int32(fixedmath.DIV32_16(maskAvg, int16(count))), dbShift-10)
		maskAvg += gconstFifth
		diff = fixedmath.SHL32(diff*6/int32(C*(maskEnd-1)*(maskEnd+1)*maskEnd), dbShift-10)
		// Again, being conservative.
		diff = fixedmath.HALF32(diff)
		diff = fixedmath.MAX32(fixedmath.MIN32(diff, gconst031), -gconst031)
		// Find the band that's in the middle of the coded spectrum.
		for midband = 0; int(eBands[midband+1]) < int(eBands[maskEnd])/2; midband++ {
		}
		countDynalloc = 0
		for i = 0; i < maskEnd; i++ {
			var lin int32
			var unmask int32
			lin = maskAvg + diff*int32(i-midband)
			if C == 2 {
				unmask = fixedmath.MAX32(st.energyMask[i], st.energyMask[nbEBands+i])
			} else {
				unmask = st.energyMask[i]
			}
			unmask = fixedmath.MIN32(unmask, 0) // MING(unmask, GCONST(.0f))
			unmask -= lin
			if unmask > gconstQuarter {
				surroundDynalloc[i] = unmask - gconstQuarter
				countDynalloc++
			}
		}
		if countDynalloc >= 3 {
			// If we need dynalloc in many bands, it's probably because our initial
			// masking rate was too low.
			maskAvg += gconstQuarter
			if maskAvg > 0 {
				// Something went really wrong in the original calculations, disabling
				// masking.
				maskAvg = 0
				diff = 0
				for i = 0; i < maskEnd; i++ {
					surroundDynalloc[i] = 0
				}
			} else {
				for i = 0; i < maskEnd; i++ {
					surroundDynalloc[i] = fixedmath.MAX32(0, surroundDynalloc[i]-gconstQuarter)
				}
			}
		}
		maskAvg += gconstFifth
		// Convert to 1/64th units used for the trim.
		surroundTrim = 64 * diff
		surroundMasking = maskAvg
		if w != nil {
			w.SurroundBlock = true
		}
	}
	// Temporal VBR (but not for LFE). LIVE (:2186-2203): reads and writes
	// st->spec_avg, which is cross-frame state.
	if st.lfe == 0 {
		follow := -fixedmath.QCONST32(10, dbShift-5)
		frameAvg := int32(0)
		offset := int32(0)
		if shortBlocks != 0 {
			offset = fixedmath.HALF32(fixedmath.SHL32(int32(LM), dbShift-5))
		}
		for i = start; i < end; i++ {
			follow = fixedmath.MAX32(follow-fixedmath.QCONST32(1, dbShift-5),
				fixedmath.SHR32(bandLogE[i], 5)-offset)
			if C == 2 {
				follow = fixedmath.MAX32(follow, fixedmath.SHR32(bandLogE[i+nbEBands], 5)-offset)
			}
			frameAvg += follow
		}
		frameAvg /= int32(end - start)
		temporalVbr = fixedmath.SUB32(fixedmath.SHL32(frameAvg, 5), st.specAvg)
		temporalVbr = fixedmath.MIN32(gconst3, fixedmath.MAX32(-gconst1_5, temporalVbr))
		st.specAvg += fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.02, 15), temporalVbr)
		if w != nil {
			w.TemporalVbrBlock = true
		}
	}

	if secondMdct == 0 {
		copy(bandLogE2, bandLogE[:C*nbEBands])
	}

	// Last chance to catch any transient we might have missed in the time-domain
	// analysis.
	if LM > 0 && int32(enc.Tell())+3 <= totalBits && isTransient == 0 && st.complexity >= 5 &&
		st.lfe == 0 && hybrid == 0 {
		if patchTransientDecision(bandLogE, oldBandE, nbEBands, start, end, C) != 0 {
			isTransient = 1
			shortBlocks = M
			computeMdcts(m, shortBlocks, in, freq, C, CC, LM, st.upsample, &st.sc)
			computeBandEnergies(m, freq, bandE, effEnd, C, LM, st.arch)
			amp2Log2(m, effEnd, end, bandE, bandLogE, C)
			// Compensate for the scaling of short vs long mdcts.
			for c = 0; c < C; c++ {
				for i = 0; i < end; i++ {
					bandLogE2[nbEBands*c+i] += fixedmath.HALF32(fixedmath.SHL32(int32(LM), dbShift))
				}
			}
			tfEstimate = fixedmath.QCONST16(0.2, 14)
			if w != nil {
				w.PatchedTransient = true
			}
		}
	}

	if LM > 0 && int32(enc.Tell())+3 <= totalBits {
		enc.EncBitLogp(isTransient, 3)
	}

	X := alloc(&st.sc.X, C*N) // VARDECL(celt_norm, X): interleaved normalised MDCTs

	// Band normalisation.
	normaliseBands(m, freq, X, bandE, effEnd, C, M)

	enableTfAnalysis := effectiveBytes >= 15*C && hybrid == 0 && st.complexity >= 2 &&
		st.lfe == 0 && toneishness < qconst0_98Q29

	offsets := alloc(&st.sc.offsets, nbEBands)
	importance := alloc(&st.sc.importance, nbEBands)
	spreadWeight := alloc(&st.sc.spreadWeight, nbEBands)
	// dynalloc_analysis writes importance ONLY over [start,end) (celt_encoder.c:1187
	// / :1268), but tf_analysis reads [0,effEnd) (:755,:762,:772). When start>0 the
	// read window exceeds the write window: C reads uninitialised stack there, and a
	// pooled buffer would read the PREVIOUS FRAME's values, where make() read zeros.
	// Today that divergence is unreachable — tf_analysis runs only under
	// enableTfAnalysis, which requires hybrid==0, and hybrid==(start!=0) (:294-297) —
	// but the invariant spans three files, so pin it here for 21 words rather than
	// leave a future change to the start band silently observing stale data.
	for i = 0; i < nbEBands; i++ {
		importance[i] = 0
	}

	maxDepth = dynallocAnalysis(bandLogE, bandLogE2, oldBandE, nbEBands, start, end, C, offsets,
		st.lsbDepth, m.logN, isTransient, st.vbr, st.constrainedVbr, eBands, LM, effectiveBytes,
		&totBoost, st.lfe, surroundDynalloc, importance, spreadWeight, toneFreq, toneishness, &st.sc)

	tfRes := alloc(&st.sc.tfRes, nbEBands)
	// Disable variable tf resolution for hybrid and at very low bitrate.
	if enableTfAnalysis {
		var lambda int
		lambda = fixedmath.IMAX(80, 20480/effectiveBytes+2)
		tfSelect = tfAnalysis(m, effEnd, isTransient, tfRes, lambda, X, N, LM, tfEstimate, tfChan, importance, &st.sc)
		for i = effEnd; i < end; i++ {
			tfRes[i] = tfRes[effEnd-1]
		}
	} else if hybrid != 0 && weakTransient != 0 {
		// For weak transients, we rely on the fact that improving time resolution
		// using TF on a long window is imperfect and will not result in an energy
		// collapse at low bitrate.
		for i = 0; i < end; i++ {
			tfRes[i] = 1
		}
		tfSelect = 0
	} else if hybrid != 0 && effectiveBytes < 15 && st.silkSignalType != 2 {
		// For low bitrate hybrid, force temporal resolution to 5 ms rather than 2.5 ms.
		for i = 0; i < end; i++ {
			tfRes[i] = 0
		}
		tfSelect = isTransient
	} else {
		for i = 0; i < end; i++ {
			tfRes[i] = isTransient
		}
		tfSelect = 0
	}

	energyErr := alloc(&st.sc.energyErr, C*nbEBands) // VARDECL(celt_glog, error)
	c = 0
	for {
		for i = start; i < end; i++ {
			// When the energy is stable, slightly bias energy quantization towards the
			// previous error to make the gain more stable.
			if fixedmath.ABS32(fixedmath.SUB32(bandLogE[i+c*nbEBands], oldBandE[i+c*nbEBands])) < gconst2 {
				bandLogE[i+c*nbEBands] -= fixedmath.MULT16_32_Q15(fixedmath.QCONST16(0.25, 15),
					energyError[i+c*nbEBands])
			}
		}
		c++
		if c >= C {
			break
		}
	}
	twoPass := 0
	if st.complexity >= 4 {
		twoPass = 1
	}
	coarseIntra := quantCoarseEnergy(m, start, end, effEnd, bandLogE, oldBandE, uint32(totalBits),
		energyErr, enc, C, LM, nbAvailableBytes, st.forceIntra, &st.delayedIntra,
		twoPass, st.lossRate, st.lfe, &st.sc)

	// tf_encode MUTATES tf_res in place (unaffordable bands forced to curr, then
	// every entry remapped through tf_select_table); the remapped array is what
	// quant_all_bands consumes below.
	tfEncode(start, end, isTransient, tfRes, LM, tfSelect, enc)

	if int32(enc.Tell())+4 <= totalBits {
		switch {
		case st.lfe != 0:
			st.tapsetDecision = 0
			st.spreadDecision = spreadNormal
		case hybrid != 0:
			if st.complexity == 0 {
				st.spreadDecision = spreadNone
			} else if isTransient != 0 {
				st.spreadDecision = spreadNormal
			} else {
				st.spreadDecision = spreadAggressive
			}
		case shortBlocks != 0 || st.complexity < 3 || nbAvailableBytes < 10*C:
			if st.complexity == 0 {
				st.spreadDecision = spreadNone
			} else {
				st.spreadDecision = spreadNormal
			}
		default:
			// The `#if 0` analysis-driven hysteresis_decision block (:2320-2331) is
			// disabled upstream; spreading_decision() is the live path.
			updateHf := 0
			if pfOn != 0 && shortBlocks == 0 {
				updateHf = 1
			}
			st.spreadDecision = spreadingDecision(m, X, &st.tonalAverage, st.spreadDecision,
				&st.hfAverage, &st.tapsetDecision, updateHf, effEnd, C, M, spreadWeight)
			if w != nil {
				w.SpreadFromDecision = true
			}
		}
		enc.EncIcdf(st.spreadDecision, spreadIcdf, 5)
	} else {
		st.spreadDecision = spreadNormal
		if w != nil {
			w.SpreadNoRoom = true
		}
	}

	// For LFE, everything interesting is in the first band.
	if st.lfe != 0 {
		offsets[0] = fixedmath.IMIN(8, effectiveBytes/3)
	}
	caps := alloc(&st.sc.caps, nbEBands) // VARDECL(int, cap)
	initCaps(m, caps, LM, C)

	dynallocLogp = 6
	totalBits <<= bitRes
	totalBoost = 0
	tell = int32(enc.TellFrac())
	for i = start; i < end; i++ {
		var width, quanta int
		var dynallocLoopLogp int
		var boost int
		var j int
		width = (C * int(eBands[i+1]-eBands[i])) << LM
		// quanta is 6 bits, but no more than 1 bit/sample and no less than 1/8
		// bit/sample.
		quanta = fixedmath.IMIN(width<<bitRes, fixedmath.IMAX(6<<bitRes, width))
		dynallocLoopLogp = dynallocLogp
		boost = 0
		for j = 0; tell+int32(dynallocLoopLogp<<bitRes) < totalBits-totalBoost && boost < caps[i]; j++ {
			var flag int
			if j < offsets[i] {
				flag = 1
			}
			enc.EncBitLogp(flag, uint32(dynallocLoopLogp))
			tell = int32(enc.TellFrac())
			if flag == 0 {
				break
			}
			boost += quanta
			totalBoost += int32(quanta)
			dynallocLoopLogp = 1
			if w != nil {
				w.DynallocBoosted = true
			}
		}
		// Making dynalloc more likely.
		if j != 0 {
			dynallocLogp = fixedmath.IMAX(2, dynallocLogp-1)
		}
		offsets[i] = boost
	}

	if C == 2 {
		// Always use MS for 2.5 ms frames until we can do a better analysis.
		if LM != 0 {
			dualStereo = stereoAnalysis(m, X, LM, N)
		}
		prevIntensity := st.intensity
		st.intensity = hysteresisDecision(int16(equivRate/1000), intensityThresholds[:],
			intensityHisteresis[:], 21, st.intensity)
		st.intensity = fixedmath.IMIN(end, fixedmath.IMAX(start, st.intensity))
		if w != nil && st.intensity != prevIntensity {
			w.IntensityChanged = true
		}
	}

	allocTrim = 5
	if tell+(6<<bitRes) <= totalBits-totalBoost {
		if start > 0 || st.lfe != 0 {
			st.stereoSaving = 0
			allocTrim = 5
			if w != nil {
				w.AllocTrimFromLfe = true
			}
		} else {
			allocTrim = allocTrimAnalysis(m, X, bandLogE, end, LM, C, N, &st.stereoSaving,
				tfEstimate, st.intensity, surroundTrim, equivRate)
		}
		enc.EncIcdf(allocTrim, trimIcdf, 7)
		tell = int32(enc.TellFrac())
		if w != nil {
			w.AllocTrimCoded = true
		}
	}

	// In VBR mode the frame size must not be reduced so much that it would result
	// in the encoder running out of bits. The margin of 2 bytes ensures that none
	// of the bust-prevention logic in the decoder will have triggered so far.
	minAllowed = ((tell + totalBoost + (1 << (bitRes + 3)) - 1) >> (bitRes + 3)) + 2
	// Take into account the 37 bits needed to signal a redundant frame in hybrid.
	if hybrid != 0 {
		minAllowed = fixedmath.MAX32(minAllowed,
			(tell0Frac+(37<<bitRes)+totalBoost+(1<<(bitRes+3))-1)>>(bitRes+3))
	}

	// Variable bitrate (:2435-2533). ALL the cross-frame VBR state lives here.
	if vbrRate > 0 {
		var alpha int16
		var delta int32
		// The target rate in 8th bits per frame.
		var target, baseTarget int32
		lmDiff := m.maxLM - LM

		// Don't attempt to use more than 510 kb/s, even for frames smaller than 20 ms.
		nbCompressedBytes = fixedmath.IMIN(nbCompressedBytes, packetSizeCap>>(3-LM))
		if hybrid == 0 {
			baseTarget = vbrRate - int32((40*C+20)<<bitRes)
		} else {
			baseTarget = fixedmath.MAX32(0, vbrRate-int32((9*C+4)<<bitRes))
		}

		if st.constrainedVbr != 0 {
			baseTarget += st.vbrOffset >> lmDiff
		}

		if hybrid == 0 {
			hasSurroundMask := 0
			if st.energyMask != nil {
				hasSurroundMask = 1
			}
			target = computeVbr(m, baseTarget, LM, equivRate, st.lastCodedBands, C, st.intensity,
				st.constrainedVbr, st.stereoSaving, int(totBoost), tfEstimate, maxDepth,
				st.lfe, hasSurroundMask, surroundMasking, temporalVbr)
		} else {
			target = baseTarget
			// st->silk_info.offset is 0 on the CELT-only path (silk_info is never
			// populated), so only the <100 branch can fire. Both are transliterated.
			const silkInfoOffset = 0
			// Tonal frames (offset<100) need more bits than noisy (offset>100) ones.
			if silkInfoOffset < 100 {
				target += (12 << bitRes) >> (3 - LM)
			}
			if silkInfoOffset > 100 {
				target -= (18 << bitRes) >> (3 - LM)
			}
			// Boosting bitrate on transients and vowels with significant temporal spikes.
			target += fixedmath.MULT16_16_Q14(
				int16(int32(tfEstimate)-int32(fixedmath.QCONST16(0.25, 14))), int16(50<<bitRes))
			// If we have a strong transient, make sure it has enough bits to code the
			// first two bands, so that it can use folding rather than noise.
			if tfEstimate > fixedmath.QCONST16(0.7, 14) {
				target = fixedmath.MAX32(target, 50<<bitRes)
			}
		}
		// The current offset is removed from the target and the space used so far is
		// added.
		target += tell

		nbAvailableBytes = int((target + (1 << (bitRes + 2))) >> (bitRes + 3))
		if w != nil && minAllowed > int32(nbAvailableBytes) {
			w.MinAllowedWon = true
		}
		nbAvailableBytes = int(fixedmath.MAX32(minAllowed, int32(nbAvailableBytes)))
		nbAvailableBytes = fixedmath.IMIN(nbCompressedBytes, nbAvailableBytes)

		// By how much did we "miss" the target on that frame.
		delta = target - vbrRate

		target = int32(nbAvailableBytes << (bitRes + 3))

		// If the frame is silent we don't adjust our drift, otherwise the encoder
		// will shoot to very high rates after hitting a span of silence, but we do
		// allow the bitres to refill.
		if silence != 0 {
			nbAvailableBytes = 2
			target = 2 * 8 << bitRes
			delta = 0
		}

		if st.vbrCount < 970 {
			st.vbrCount++
			// EXTEND32 is a plain widening cast in C (fixed_generic.h), not a 16-bit
			// narrowing; celt_rcp returns opus_val16, hence the int16 truncation.
			alpha = int16(fixedmath.Celt_rcp(fixedmath.SHL32(st.vbrCount+20, 16)))
		} else {
			alpha = fixedmath.QCONST16(0.001, 15)
			if w != nil {
				w.VbrCountSaturated = true
			}
		}
		// How many bits have we used in excess of what we're allowed.
		if st.constrainedVbr != 0 {
			st.vbrReservoir += target - vbrRate
		}

		// Compute the offset we need to apply in order to reach the target.
		if st.constrainedVbr != 0 {
			st.vbrDrift += fixedmath.MULT16_32_Q15(alpha,
				(delta*(1<<lmDiff))-st.vbrOffset-st.vbrDrift)
			st.vbrOffset = -st.vbrDrift
		}

		if st.constrainedVbr != 0 && st.vbrReservoir < 0 {
			// We're under the min value: increase rate.
			adjust := int((-st.vbrReservoir) / int32(8<<bitRes))
			// Unless we're just coding silence.
			if silence == 0 {
				nbAvailableBytes += adjust
			}
			st.vbrReservoir = 0
			if w != nil {
				w.VbrReservoirClamped = true
			}
		}
		beforeShrink := nbCompressedBytes
		nbCompressedBytes = fixedmath.IMIN(nbCompressedBytes, nbAvailableBytes)
		if w != nil {
			w.Target = target
			if nbCompressedBytes < beforeShrink {
				w.VbrShrink = true
			}
		}
		// This moves the raw bits to take into account the new compressed size.
		enc.EncShrink(uint32(nbCompressedBytes))
	}

	// Bit allocation.
	fineQuant := alloc(&st.sc.fineQuant, nbEBands)       // VARDECL(int, fine_quant)
	pulses := alloc(&st.sc.pulses, nbEBands)             // VARDECL(int, pulses)
	finePriority := alloc(&st.sc.finePriority, nbEBands) // VARDECL(int, fine_priority)

	// bits = packet size - where we are - safety.
	bits = ((int32(nbCompressedBytes) * 8) << bitRes) - int32(enc.TellFrac()) - 1
	if isTransient != 0 && LM >= 2 && bits >= int32((LM+2)<<bitRes) {
		antiCollapseRsv = 1 << bitRes
	}
	bits -= int32(antiCollapseRsv)
	signalBandwidth = end - 1
	// The `#ifndef DISABLE_FLOAT_API` min_bandwidth block (:2607-2623) is dead:
	// st->analysis.valid is always 0.
	if st.lfe != 0 {
		signalBandwidth = 1
	}
	codedBands = cltComputeAllocation(m, start, end, offsets, caps, allocTrim, &st.intensity,
		&dualStereo, bits, &balance, pulses, fineQuant, finePriority, C, LM, enc, nil, 1,
		st.lastCodedBands, signalBandwidth, &st.sc)
	if st.lastCodedBands != 0 {
		st.lastCodedBands = fixedmath.IMIN(st.lastCodedBands+1,
			fixedmath.IMAX(st.lastCodedBands-1, codedBands))
	} else {
		st.lastCodedBands = codedBands
	}

	quantFineEnergy(m, start, end, oldBandE, energyErr, nil, fineQuant, enc, C)
	for i = 0; i < nbEBands*CC; i++ {
		energyError[i] = 0
	}

	// Residual quantisation.
	collapseMasks := alloc(&st.sc.collapseMasks, C*nbEBands) // VARDECL(unsigned char, collapse_masks)
	var Y []int32
	if C == 2 {
		Y = X[N:]
	}
	quantAllBands(1, m, start, end, X, Y, collapseMasks, bandE, pulses, shortBlocks,
		st.spreadDecision, dualStereo, st.intensity, tfRes,
		int32(nbCompressedBytes*(8<<bitRes)-antiCollapseRsv), balance, enc, nil, LM,
		codedBands, &st.rng, st.complexity, st.disableInv, &st.sc)

	if antiCollapseRsv > 0 {
		// The RESYNTH anti_collapse() CALL is gated out; only the decision bit is
		// coded (the decoder does the actual anti-collapse).
		if st.consecTransient < 2 {
			antiCollapseOn = 1
		}
		enc.EncBits(uint32(antiCollapseOn), 1)
	}
	// qext_bytes is always 0 without ENABLE_QEXT, so this always runs.
	quantEnergyFinalise(m, start, end, oldBandE, energyErr, fineQuant, finePriority,
		nbCompressedBytes*8-enc.Tell(), enc, C)
	c = 0
	for {
		for i = start; i < end; i++ {
			energyError[i+c*nbEBands] = fixedmath.MAX32(-ghalf,
				fixedmath.MIN32(ghalf, energyErr[i+c*nbEBands]))
		}
		c++
		if c >= C {
			break
		}
	}
	if silence != 0 {
		for i = 0; i < C*nbEBands; i++ {
			oldBandE[i] = -gconst28
		}
	}

	// The whole #ifdef RESYNTH block (:2724-2776) is gated out.

	st.prefilterPeriod = pitchIndex
	st.prefilterGain = gain1
	st.prefilterTapset = prefilterTapset

	if CC == 2 && C == 1 {
		copy(oldBandE[nbEBands:2*nbEBands], oldBandE[:nbEBands])
	}

	if isTransient == 0 {
		copy(oldLogE2[:CC*nbEBands], oldLogE[:CC*nbEBands])
		copy(oldLogE[:CC*nbEBands], oldBandE[:CC*nbEBands])
	} else {
		for i = 0; i < CC*nbEBands; i++ {
			oldLogE[i] = fixedmath.MIN32(oldLogE[i], oldBandE[i])
		}
	}
	// In case start or end were to change.
	c = 0
	for {
		for i = 0; i < start; i++ {
			oldBandE[c*nbEBands+i] = 0
			oldLogE[c*nbEBands+i] = -gconst28
			oldLogE2[c*nbEBands+i] = -gconst28
		}
		for i = end; i < nbEBands; i++ {
			oldBandE[c*nbEBands+i] = 0
			oldLogE[c*nbEBands+i] = -gconst28
			oldLogE2[c*nbEBands+i] = -gconst28
		}
		c++
		if c >= CC {
			break
		}
	}

	if isTransient != 0 || transientGotDisabled != 0 {
		st.consecTransient++
	} else {
		st.consecTransient = 0
	}
	st.rng = enc.Rng()

	// If there's any room left (can only happen for very high rates), it's already
	// filled with zeros.
	enc.EncDone()

	// The CUSTOM_MODES `if (st->signalling) nbCompressedBytes++;` (:2822-2825) is
	// gated out.

	if w != nil {
		w.LM = LM
		w.NbCompressedBytes = nbCompressedBytes
		w.NbAvailableBytes = nbAvailableBytes
		w.EffectiveBytes = effectiveBytes
		w.EquivRate = equivRate
		w.VbrRate = vbrRate
		w.Silence = silence != 0
		w.IsTransient = isTransient != 0
		w.WeakTransient = weakTransient != 0
		w.TransientGotDisabled = transientGotDisabled != 0
		w.ShortBlocks = shortBlocks
		w.SecondMdct = secondMdct != 0
		w.PfOn = pfOn != 0
		w.EnableTfAnalysis = enableTfAnalysis
		w.TfSelect = tfSelect
		w.TfRes = append([]int(nil), tfRes...)
		w.SpreadDecision = st.spreadDecision
		w.MaxDepth = maxDepth
		w.TotBoost = totBoost
		w.TotalBoost = totalBoost
		w.AllocTrim = allocTrim
		w.Intensity = st.intensity
		w.DualStereo = dualStereo != 0
		w.CodedBands = codedBands
		w.AntiCollapseRsv = antiCollapseRsv
		w.AntiCollapseOn = antiCollapseOn != 0
		w.CoarseIntra = coarseIntra != 0
		w.SurroundMasking = surroundMasking
		w.SurroundTrim = surroundTrim
		w.TemporalVbr = temporalVbr
	}

	if enc.Error() != 0 {
		return opusInternalError
	}
	return nbCompressedBytes
}

// --- Differential-test entry points (used by internal/reftest/oracle) --------

// EncodeFrame is the differential-test seam over celt_encode_with_ec: it encodes
// one frame into a freshly allocated nbCompressedBytes buffer and returns the raw
// return value, the packet bytes (nil on error) and the branch witness. The
// encoder's cross-frame state is mutated exactly as Encode mutates it, so a test
// can drive a multi-frame sequence through this and compare State() every frame.
func (st *Encoder) EncodeFrame(pcm []int16, frameSize, nbCompressedBytes int) (int, []byte, EncodeWitness) {
	var w EncodeWitness
	buf := make([]byte, nbCompressedBytes)
	ret := st.encodeObserved(pcm, frameSize, buf, nbCompressedBytes, nil, &w)
	if ret < 0 {
		return ret, nil, w
	}
	return ret, buf[:ret], w
}
