// CELT top-level decoder pipeline ported from libopus celt/celt_decoder.c for
// the frozen build config (FIXED_POINT, DISABLE_FLOAT_API, non-CUSTOM_MODES,
// non-QEXT, non-RES24, non-DEEP_PLC). This assembles every previously ported
// layer (energy unquant, tf_decode, allocation, PVQ band decode, inverse MDCT,
// post-filter, de-emphasis) into a working CELT decode.
//
// Source: libopus v1.6.1 (commit 3da9f7a6). The struct and celt_decode_with_ec
// are transliterated in C order. The lost-frame (PLC) branch calls
// celtDecodeLost, which implements both concealment regimes (pitch extrapolation
// and noise/CNG) on the pitch.go / celt_lpc.go primitives.
//
// Type mapping (celt/arch.h): celt_sig / celt_norm / celt_glog = int32,
// celt_coef / opus_val16 / opus_res = int16. So the decoded PCM is int16 and the
// energy state arrays are int32 (Q24, DB_SHIFT=24).

package celt

import (
	"errors"
	"hash/fnv"

	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
)

// Log-energy (celt_glog, Q24) constants used by the decoder that are not already
// defined by quant_bands.go (which provides gconst9, gconst28, ghalf). GCONST(x)
// = QCONST32(x, DB_SHIFT); all the values below are exact in float64.
var (
	gconst1_5   = fixedmath.QCONST32(1.5, dbShift)   // GCONST(1.5f)
	gconst2     = fixedmath.QCONST32(2, dbShift)     // GCONST(2.f)
	gconst20    = fixedmath.QCONST32(20, dbShift)    // GCONST(20.f)
	gconst0_001 = fixedmath.QCONST32(0.001, dbShift) // GCONST(0.001f)
)

// Decoder errors, mirroring the negative OPUS_* return codes of the C decoder.
var (
	ErrBadArg         = errors.New("celt: bad argument")
	ErrBufferTooSmall = errors.New("celt: output buffer too small")
	ErrInternal       = errors.New("celt: internal error")
)

// Decoder mirrors celt/celt_decoder.c struct OpusCustomDecoder for the frozen
// config. The trailing C flexible arrays (_decode_mem, oldEBands/oldLogE/
// oldLogE2/backgroundLogE, lpc) become explicit slices; everything below
// resetStart is cleared by ResetState (the DECODER_RESET_START boundary).
type Decoder struct {
	mode           *celtMode
	overlap        int
	channels       int // CC: physical channels of the decode buffers
	streamChannels int // C: channels actually coded in the current stream

	downsample int
	start, end int
	signalling int
	disableInv int
	complexity int

	// --- reset region (DECODER_RESET_START rng) ---
	rng                 uint32
	err                 int
	lastPitchIndex      int
	lossDuration        int
	plcDuration         int
	lastFrameType       int
	skipPlc             int
	postfilterPeriod    int
	postfilterPeriodOld int
	postfilterGain      int16 // opus_val16
	postfilterGainOld   int16 // opus_val16
	postfilterTapset    int
	postfilterTapsetOld int
	prefilterAndFold    int

	preemphMemD [2]int32 // celt_sig

	decodeMem      []int32 // channels*(DECODE_BUFFER_SIZE+overlap)
	oldEBands      []int32 // 2*nbEBands
	oldLogE        []int32 // 2*nbEBands
	oldLogE2       []int32 // 2*nbEBands
	backgroundLogE []int32 // 2*nbEBands
	lpc            []int16 // channels*CELT_LPC_ORDER (opus_val16), PLC only

	// Per-frame coverage flags (not decoder state; for differential-test
	// coverage assertions only). Updated by every DecodeWithEC call.
	lastTransient    bool
	lastPostfilterOn bool
	lastAntiCollapse bool

	// Per-frame working buffers. In C these are stack VARDECL/ALLOCs, reclaimed on
	// frame pop; here they are pooled on the decoder so a steady-state Decode
	// allocates nothing (see scratch.go, and its decode zeroing audit). This is
	// state only in the sense that a C thread's stack is: it carries nothing across
	// frames that any frame reads, and ResetState does not need to touch it.
	sc scratch

	// The [][]int32 pointer arrays celt_decode_with_ec and celt_synthesis build
	// from decode_mem each frame (C uses stack pointer arrays). Backed here by fixed
	// [2][]int32 storage so decMem := decMemSlices[:CC] and outSyn := outSynSlices[:CC]
	// reuse the same headers every call instead of make([][]int32, CC).
	decMemSlices [2][]int32
	outSynSlices [2][]int32
}

// NewDecoder allocates and initializes a CELT decoder for the 48 kHz / 960 mode
// with the given channel count (1 or 2), mirroring celt_decoder_init(st, 48000,
// channels) followed by OPUS_RESET_STATE. It returns nil if channels is invalid.
func NewDecoder(channels int) *Decoder {
	if channels < 1 || channels > 2 {
		return nil
	}
	st := &Decoder{}
	m := &mode48000_960
	st.mode = m
	st.overlap = m.overlap
	st.channels = channels
	st.streamChannels = channels
	st.downsample = 1 // resampling_factor(48000)
	st.start = 0
	st.end = m.effEBands
	st.signalling = 1
	if channels == 1 { // !DISABLE_UPDATE_DRAFT: disable_inv = channels==1
		st.disableInv = 1
	}

	per := decodeBufferSize + st.overlap
	nb := m.nbEBands
	st.decodeMem = make([]int32, channels*per)
	st.oldEBands = make([]int32, 2*nb)
	st.oldLogE = make([]int32, 2*nb)
	st.oldLogE2 = make([]int32, 2*nb)
	st.backgroundLogE = make([]int32, 2*nb)
	st.lpc = make([]int16, channels*celtLpcOrder)

	st.ResetState()
	return st
}

// ResetState implements opus_custom_decoder_ctl(st, OPUS_RESET_STATE): clear the
// whole reset region, then seed oldLogE/oldLogE2 to -GCONST(28), skip_plc=1 and
// last_frame_type=FRAME_NONE (celt_decoder.c:1790-1808).
func (st *Decoder) ResetState() {
	st.rng = 0
	st.err = 0
	st.lastPitchIndex = 0
	st.lossDuration = 0
	st.plcDuration = 0
	st.lastFrameType = frameNone
	st.skipPlc = 0
	st.postfilterPeriod = 0
	st.postfilterPeriodOld = 0
	st.postfilterGain = 0
	st.postfilterGainOld = 0
	st.postfilterTapset = 0
	st.postfilterTapsetOld = 0
	st.prefilterAndFold = 0
	st.preemphMemD[0] = 0
	st.preemphMemD[1] = 0
	for i := range st.decodeMem {
		st.decodeMem[i] = 0
	}
	for i := range st.oldEBands {
		st.oldEBands[i] = 0
	}
	for i := range st.backgroundLogE {
		st.backgroundLogE[i] = 0
	}
	for i := range st.lpc {
		st.lpc[i] = 0
	}
	for i := 0; i < 2*st.mode.nbEBands; i++ {
		st.oldLogE[i] = -gconst28
		st.oldLogE2[i] = -gconst28
	}
	st.skipPlc = 1
	st.lastFrameType = frameNone
}

// SetStartBand mirrors CELT_SET_START_BAND (0 for CELT-only, 17 for hybrid).
func (st *Decoder) SetStartBand(v int) error {
	if v < 0 || v >= st.mode.nbEBands {
		return ErrBadArg
	}
	st.start = v
	return nil
}

// SetEndBand mirrors CELT_SET_END_BAND.
func (st *Decoder) SetEndBand(v int) error {
	if v < 1 || v > st.mode.nbEBands {
		return ErrBadArg
	}
	st.end = v
	return nil
}

// SetStreamChannels mirrors CELT_SET_CHANNELS (the coded channel count C).
func (st *Decoder) SetStreamChannels(v int) error {
	if v < 1 || v > 2 {
		return ErrBadArg
	}
	st.streamChannels = v
	return nil
}

// SetDownsample sets the integer output downsampling factor, matching the
// resampling_factor(Fs) that celt_decoder_init derives from the decode rate:
// 1 for 48 kHz, 2 for 24 kHz, 3 for 16 kHz, 4 for 12 kHz, 6 for 8 kHz. The
// value chosen by NewDecoder is 1 (the 48 kHz mode); the top-level Opus decoder
// calls this so a decoder opened below 48 kHz produces output at that rate. It
// returns ErrBadArg for a factor that is not one of the CELT-supported values.
func (st *Decoder) SetDownsample(factor int) error {
	switch factor {
	case 1, 2, 3, 4, 6:
		st.downsample = factor
		return nil
	default:
		return ErrBadArg
	}
}

// Rng returns the decoder range-coder state after the last frame
// (OPUS_GET_FINAL_RANGE); bit-exact agreement is the primary differential check.
func (st *Decoder) Rng() uint32 { return st.rng }

// Window returns the mode's overlap window (celt_mode->window). The Opus-level
// transition machine reads it for smooth_fade at SILK<->CELT and mode-transition
// boundaries (src/opus_decoder.c:633, CELT_GET_MODE then celt_mode->window).
func (st *Decoder) Window() []int16 { return st.mode.window }

// LastFrameTransient / LastFramePostfilter / LastFrameAntiCollapse report
// per-frame coverage flags from the most recent DecodeWithEC (test support so
// the differential suite can assert the transient / post-filter / anti-collapse
// paths were exercised). They are not part of the decoder's persistent state.
func (st *Decoder) LastFrameTransient() bool    { return st.lastTransient }
func (st *Decoder) LastFramePostfilter() bool   { return st.lastPostfilterOn }
func (st *Decoder) LastFrameAntiCollapse() bool { return st.lastAntiCollapse }

// tfDecode is celt_decoder.c:513 tf_decode: read the per-band time-frequency
// resolution flags and resolve them through tf_select_table.
func tfDecode(start, end, isTransient int, tfRes []int, LM int, dec *rangecoding.Decoder) {
	budget := int(dec.Storage()) * 8
	tell := dec.Tell()
	logp := 4
	if isTransient != 0 {
		logp = 2
	}
	tfSelectRsv := 0
	if LM > 0 && tell+logp+1 <= budget {
		tfSelectRsv = 1
	}
	budget -= tfSelectRsv
	tfChanged := 0
	curr := 0
	for i := start; i < end; i++ {
		if tell+logp <= budget {
			curr ^= dec.DecBitLogp(uint32(logp))
			tell = dec.Tell()
			tfChanged |= curr
		}
		tfRes[i] = curr
		if isTransient != 0 {
			logp = 4
		} else {
			logp = 5
		}
	}
	tfSelect := 0
	if tfSelectRsv != 0 &&
		tfSelectTable[LM][4*isTransient+0+tfChanged] != tfSelectTable[LM][4*isTransient+2+tfChanged] {
		tfSelect = dec.DecBitLogp(1)
	}
	for i := start; i < end; i++ {
		tfRes[i] = int(tfSelectTable[LM][4*isTransient+2*tfSelect+tfRes[i]])
	}
}

// deemphasis is celt_decoder.c:318 for the frozen config (non-CUSTOM_MODES so
// coef[1] is unused, non-RES24 so opus_res is int16). It runs the preemphasis
// inverse (a 1-tap IIR) over each channel of out_syn, optional integer
// downsampling, and writes interleaved int16 PCM. mem carries preemph_memD.
func deemphasis(outSyn [][]int32, pcm []int16, N, C, downsample int, coef [4]int16, mem *[2]int32, accum int, sc *scratch) {
	coef0 := coef[0]
	Nd := N / downsample
	// dScratch is C's VARDECL(celt_sig, scratch) (celt_decoder.c:335), pooled here.
	// Each channel writes dScratch[0:N] in full before the downsampling read, so the
	// carried-over buffer is never observed. downsample==1 (the 48 kHz path) never
	// touches it. See scratch.go's decode zeroing audit. (Renamed off "scratch" to
	// not shadow the scratch type.)
	var dScratch []int32
	if downsample > 1 {
		dScratch = alloc(&sc.decDeemph, N)
	}
	for c := 0; c < C; c++ {
		m := mem[c]
		// Slice to N once so the per-sample reads carry no bounds check. The
		// strided pcm[j*C+c] stores keep their check (the stride defeats the
		// prover), but the outSyn/dScratch reads no longer do.
		x := outSyn[c][:N]
		applyDown := false
		switch {
		case downsample > 1:
			db := dScratch[:N]
			for j := range x {
				tmp := fixedmath.SATURATE(x[j]+m, sigSat) // VERY_SMALL == 0
				m = fixedmath.MULT16_32_Q15(coef0, tmp)
				db[j] = tmp
			}
			applyDown = true
		case accum != 0:
			for j := range x {
				tmp := fixedmath.SATURATE(x[j]+m, sigSat)
				m = fixedmath.MULT16_32_Q15(coef0, tmp)
				pcm[j*C+c] = sat16(int32(pcm[j*C+c]) + int32(sig2word16(tmp)))
			}
		default:
			for j := range x {
				tmp := fixedmath.SATURATE(x[j]+m, sigSat)
				m = fixedmath.MULT16_32_Q15(coef0, tmp)
				pcm[j*C+c] = sig2word16(tmp)
			}
		}
		mem[c] = m
		if applyDown {
			if accum != 0 {
				for j := 0; j < Nd; j++ {
					pcm[j*C+c] = sat16(int32(pcm[j*C+c]) + int32(sig2word16(dScratch[j*downsample])))
				}
			} else {
				for j := 0; j < Nd; j++ {
					pcm[j*C+c] = sig2word16(dScratch[j*downsample])
				}
			}
		}
	}
}

// celtSynthesis is celt_decoder.c:413: denormalise each channel's bands and run
// the inverse MDCT with overlap-add into out_syn. It handles the normal
// (C==CC) case plus the mono->stereo and stereo->mono downmix cases used by the
// Opus wrapper. out_syn[c] must be a view into decode_mem[c] starting at
// decode_buffer_size-N (length N+overlap) so the IMDCT overlap-add and the
// mono->stereo scratch copy have room.
func celtSynthesis(m *celtMode, X []int32, outSyn [][]int32, oldBandE []int32, start, effEnd, C, CC, isTransient, LM, downsample, silence int, sc *scratch) {
	overlap := m.overlap
	nbEBands := m.nbEBands
	N := m.shortMdctSize << LM
	// Pooled (VARDECL(celt_sig, freq), celt_decoder.c:432): denormalise_bands writes
	// every one of freq's N samples before the inverse MDCT reads them, so the
	// carried-over buffer is never observed. See scratch.go's decode zeroing audit.
	freq := alloc(&sc.decFreq, N)
	M := 1 << LM
	var B, NB, shift int
	if isTransient != 0 {
		B = M
		NB = m.shortMdctSize
		shift = m.maxLM
	} else {
		B = 1
		NB = m.shortMdctSize << LM
		shift = m.maxLM - LM
	}

	switch {
	case CC == 2 && C == 1:
		// Copy a mono stream to two channels.
		denormaliseBands(m, X, freq, oldBandE, start, effEnd, M, downsample, silence)
		freq2Off := overlap / 2
		freq2 := outSyn[1]
		copy(freq2[freq2Off:freq2Off+N], freq[:N])
		for b := 0; b < B; b++ {
			cltMDCTBackward(&m.mdct, freq2[freq2Off+b:], outSyn[0][NB*b:], m.window, overlap, shift, B, sc)
		}
		for b := 0; b < B; b++ {
			cltMDCTBackward(&m.mdct, freq[b:], outSyn[1][NB*b:], m.window, overlap, shift, B, sc)
		}
	case CC == 1 && C == 2:
		// Downmix a stereo stream to mono.
		freq2Off := overlap / 2
		freq2 := outSyn[0]
		denormaliseBands(m, X, freq, oldBandE, start, effEnd, M, downsample, silence)
		denormaliseBands(m, X[N:], freq2[freq2Off:], oldBandE[nbEBands:], start, effEnd, M, downsample, silence)
		fb := freq[:N]
		f2 := freq2[freq2Off : freq2Off+N]
		for i := range fb {
			fb[i] = fixedmath.ADD32(fixedmath.HALF32(fb[i]), fixedmath.HALF32(f2[i]))
		}
		for b := 0; b < B; b++ {
			cltMDCTBackward(&m.mdct, freq[b:], outSyn[0][NB*b:], m.window, overlap, shift, B, sc)
		}
	default:
		// Normal case (mono or stereo).
		for c := 0; c < CC; c++ {
			denormaliseBands(m, X[c*N:], freq, oldBandE[c*nbEBands:], start, effEnd, M, downsample, silence)
			for b := 0; b < B; b++ {
				cltMDCTBackward(&m.mdct, freq[b:], outSyn[c][NB*b:], m.window, overlap, shift, B, sc)
			}
		}
	}
	// Saturate IMDCT output so the post-filter and de-emphasis can't overflow.
	for c := 0; c < CC; c++ {
		oc := outSyn[c][:N]
		for i := range oc {
			oc[i] = fixedmath.SATURATE(oc[i], sigSat)
		}
	}
}

// foldPrefilter is prefilter_and_fold (celt_decoder.c:576): after a concealed
// (PLC) frame, apply the pre-filter to the MDCT overlap and simulate TDAC so the
// concealed signal blends into the next real frame. It is only reachable through
// the PLC path (prefilter_and_fold is set by celtDecodeLost), which is stubbed
// here, so this runs only if a future PLC port sets the flag. Ported for
// completeness; it uses only comb_filter and the mode window.
func (st *Decoder) foldPrefilter(decMem [][]int32, N int) {
	m := st.mode
	overlap := st.overlap
	CC := st.channels
	// etmp is pooled: comb_filter below writes all `overlap` samples of it before the
	// TDAC fold reads them, so the reused buffer never exposes a prior frame's data. It
	// is local to this call; prefilter_and_fold's cross-frame span is in the flag and
	// decode_mem, not here.
	etmp := alloc(&st.sc.decEtmp, overlap)
	for c := 0; c < CC; c++ {
		base := decodeBufferSize - N
		// comb_filter into etmp from decode_mem[c]+decode_buffer_size-N.
		combFilter(etmp, 0, decMem[c], base,
			st.postfilterPeriodOld, st.postfilterPeriod, overlap,
			-st.postfilterGainOld, -st.postfilterGain,
			st.postfilterTapsetOld, st.postfilterTapset, 0, m.window)
		for i := 0; i < overlap/2; i++ {
			decMem[c][base+i] =
				fixedmath.MULT16_32_Q15(m.window[i], etmp[overlap-1-i]) +
					fixedmath.MULT16_32_Q15(m.window[overlap-i-1], etmp[i])
		}
	}
}

// celtPlcPitchSearch is celt_plc_pitch_search (celt_decoder.c:552): downsample
// the decode history by 2, run the pitch search over it bounded to
// [PLC_PITCH_LAG_MIN, PLC_PITCH_LAG_MAX], and return the pitch lag.
func (st *Decoder) celtPlcPitchSearch(decMem [][]int32, C int) int {
	// lpPitchBuf is pooled: pitch_downsample writes every element before pitch_search
	// reads it, so the reused buffer is fully reinitialised each call. The searches
	// borrow the decoder's own scratch (st.sc); pitch_downsample / pitch_search are
	// leaf calls here and hold nothing else live on the PLC path.
	lpPitchBuf := alloc(&st.sc.decLpPitchBuf, decodeBufferSize>>1)
	pitchDownsample(decMem, lpPitchBuf, decodeBufferSize>>1, C, 2, &st.sc)
	var pitchIndex int
	pitchSearch(lpPitchBuf[plcPitchLagMax>>1:], lpPitchBuf,
		decodeBufferSize-plcPitchLagMax, plcPitchLagMax-plcPitchLagMin, &pitchIndex, &st.sc)
	pitchIndex = plcPitchLagMax - pitchIndex
	return pitchIndex
}

// celtDecodeLost is celt_decode_lost (celt_decoder.c:675) for the frozen config
// (no ENABLE_DEEP_PLC, so the neural/DRED regimes and their frame types compile
// away; curr_neural/last_neural are always 0). It runs one of two concealment
// regimes selected by plc_duration/start/skip_plc:
//
//   - Noise PLC/CNG (plc_duration >= 40 || start != 0 || skip_plc): decay each
//     band energy toward backgroundLogE, fill the spectrum with LCG noise,
//     renormalise, synthesize, re-run the post-filter, and lock out pitch PLC.
//   - Pitch PLC otherwise: LPC-analyse the decode history into an excitation,
//     extrapolate one pitch period with a fade/decay, and re-synthesize.
//
// It writes the concealed frame into out_syn (decode_mem[c][decode_buffer_size-N:])
// and updates loss_duration/plc_duration/last_frame_type/prefilter_and_fold.
func (st *Decoder) celtDecodeLost(decMem [][]int32, N, LM int) {
	m := st.mode
	C := st.channels
	nbEBands := m.nbEBands
	overlap := m.overlap
	eBands := m.eBands
	per := decodeBufferSize + overlap

	// Reuse the Decoder's pointer-array storage (as celt_decode_with_ec does) instead of
	// make([][]int32, C); every header is overwritten below before any read.
	outSyn := st.outSynSlices[:C]
	for c := 0; c < C; c++ {
		outSyn[c] = decMem[c][decodeBufferSize-N:]
	}
	oldBandE := st.oldEBands
	backgroundLogE := st.backgroundLogE
	lpc := st.lpc

	lossDuration := st.lossDuration
	start := st.start

	currFrameType := framePLCPeriodic
	if st.plcDuration >= 40 || start != 0 || st.skipPlc != 0 {
		currFrameType = framePLCNoise
	}

	if currFrameType == framePLCNoise {
		// Noise-based PLC/CNG.
		end := st.end
		effEnd := fixedmath.IMAX(start, fixedmath.IMIN(end, m.effEBands))
		// X reuses decX (the normal path's C*N normalised-MDCT buffer; the two paths are
		// never live at once). The LCG-noise fill below writes bands [start,effEnd) and the
		// celtSynthesis call reads exactly that window, so decX's normal-path write-before-
		// read bound holds here too and no clear is needed.
		X := alloc(&st.sc.decX, C*N) // Interleaved normalised MDCTs.
		for c := 0; c < C; c++ {
			copy(decMem[c][:per-N], decMem[c][N:per])
		}

		if st.prefilterAndFold != 0 {
			st.foldPrefilter(decMem, N)
		}

		// Energy decay.
		decay := ghalf
		if lossDuration == 0 {
			decay = gconst1_5
		}
		for c := 0; c < C; c++ {
			for i := start; i < end; i++ {
				oldBandE[c*nbEBands+i] = fixedmath.MAX32(backgroundLogE[c*nbEBands+i], oldBandE[c*nbEBands+i]-decay)
			}
		}
		seed := st.rng
		for c := 0; c < C; c++ {
			for i := start; i < effEnd; i++ {
				boffs := N*c + (int(eBands[i]) << LM)
				blen := (int(eBands[i+1]) - int(eBands[i])) << LM
				for j := 0; j < blen; j++ {
					seed = celtLcgRand(seed)
					X[boffs+j] = fixedmath.SHL32(int32(seed)>>20, normShift-14)
				}
				RenormaliseVector(X[boffs:], blen, q31one)
			}
		}
		st.rng = seed

		celtSynthesis(m, X, outSyn, oldBandE, start, effEnd, C, C, 0, LM, st.downsample, 0, &st.sc)

		// Run the postfilter with the last parameters.
		for c := 0; c < C; c++ {
			st.postfilterPeriod = fixedmath.IMAX(st.postfilterPeriod, combfilterMinperiod)
			st.postfilterPeriodOld = fixedmath.IMAX(st.postfilterPeriodOld, combfilterMinperiod)
			base := decodeBufferSize - N
			combFilter(decMem[c], base, decMem[c], base, st.postfilterPeriodOld, st.postfilterPeriod, m.shortMdctSize,
				st.postfilterGainOld, st.postfilterGain, st.postfilterTapsetOld, st.postfilterTapset, overlap, m.window)
			if LM != 0 {
				off := base + m.shortMdctSize
				combFilter(decMem[c], off, decMem[c], off, st.postfilterPeriod, st.postfilterPeriod, N-m.shortMdctSize,
					st.postfilterGain, st.postfilterGain, st.postfilterTapset, st.postfilterTapset, overlap, m.window)
			}
		}
		st.postfilterPeriodOld = st.postfilterPeriod
		st.postfilterGainOld = st.postfilterGain
		st.postfilterTapsetOld = st.postfilterTapset

		st.prefilterAndFold = 0
		// Skip regular PLC until we get two consecutive packets.
		st.skipPlc = 1
	} else {
		// Pitch-based PLC.
		window := m.window
		fade := int16(q15one)
		var pitchIndex int
		if st.lastFrameType != framePLCPeriodic {
			pitchIndex = st.celtPlcPitchSearch(decMem, C)
			st.lastPitchIndex = pitchIndex
		} else {
			pitchIndex = st.lastPitchIndex
			fade = fixedmath.QCONST16(0.8, 15)
		}

		// We want the excitation for 2 pitch periods in order to look for a
		// decaying signal, but we can't get more than MAX_PERIOD.
		excLength := fixedmath.IMIN(2*pitchIndex, maxPeriodC)
		// excBuf and firTmp are pooled. The per-channel copy loop below rewrites all of
		// excBuf before celtFir / celtAutocorr read it, and celtFir writes all excLength
		// outputs of firTmp before the copy-back into exc reads them, so neither reused
		// buffer exposes a prior frame's data.
		excBuf := alloc(&st.sc.decExcBuf, maxPeriodC+celtLpcOrder)
		firTmp := alloc(&st.sc.decFirTmp, excLength)
		exc := excBuf[celtLpcOrder:]

		for c := 0; c < C; c++ {
			buf := decMem[c]
			for i := 0; i < maxPeriodC+celtLpcOrder; i++ {
				excBuf[i] = sround16(buf[decodeBufferSize-maxPeriodC-celtLpcOrder+i], sigShift)
			}

			if st.lastFrameType != framePLCPeriodic {
				var ac [celtLpcOrder + 1]int32
				// Compute LPC coefficients for the last MAX_PERIOD samples before
				// the first loss so we can work in the excitation-filter domain.
				// celtAutocorr borrows st.sc (autocorrXX): it writes its whole read
				// window before reading, and nothing else on this path is live in it.
				celtAutocorr(exc, ac[:], window, overlap, celtLpcOrder, maxPeriodC, &st.sc)
				// Add a noise floor of -40 dB.
				ac[0] += fixedmath.SHR32(ac[0], 13)
				// Use lag windowing to stabilize the Levinson-Durbin recursion.
				for i := 1; i <= celtLpcOrder; i++ {
					ac[i] -= fixedmath.MULT16_32_Q15(int16(2*i*i), ac[i])
				}
				celtLPC(lpc[c*celtLpcOrder:], ac[:], celtLpcOrder)
				// Apply bandwidth expansion until we can guarantee that no overflow
				// can happen in the IIR filter: 32768*sum(abs(filter)) < 2^31.
				for {
					tmp := int16(q15one)
					sum := int32(fixedmath.QCONST16(1.0, sigShift))
					for i := 0; i < celtLpcOrder; i++ {
						sum += fixedmath.ABS16(lpc[c*celtLpcOrder+i])
					}
					if sum < 65535 {
						break
					}
					for i := 0; i < celtLpcOrder; i++ {
						tmp = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(fixedmath.QCONST16(0.99, 15), tmp))
						lpc[c*celtLpcOrder+i] = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(lpc[c*celtLpcOrder+i], tmp))
					}
				}
			}
			// Compute the excitation for exc_length samples before the loss. We
			// need the copy because celt_fir() cannot filter in-place.
			celtFir(excBuf, celtLpcOrder+maxPeriodC-excLength, lpc[c*celtLpcOrder:], firTmp, excLength, celtLpcOrder)
			copy(exc[maxPeriodC-excLength:maxPeriodC], firTmp[:excLength])

			// Check if the waveform is decaying, and if so how fast, to avoid
			// adding energy when concealing in a segment with decaying energy.
			var decay int16
			{
				E1, E2 := int32(1), int32(1)
				shift := fixedmath.IMAX(0, 2*fixedmath.Celt_zlog2(celtMaxabs16(exc[maxPeriodC-excLength:], excLength))-20)
				decayLength := excLength >> 1
				for i := 0; i < decayLength; i++ {
					e := exc[maxPeriodC-decayLength+i]
					E1 += fixedmath.SHR32(fixedmath.MULT16_16(e, e), shift)
					e = exc[maxPeriodC-2*decayLength+i]
					E2 += fixedmath.SHR32(fixedmath.MULT16_16(e, e), shift)
				}
				E1 = fixedmath.MIN32(E1, E2)
				decay = int16(fixedmath.Celt_sqrt(fixedmath.Frac_div32(fixedmath.SHR32(E1, 1), E2)))
			}

			// Move the decoder memory one frame to the left to give room for the
			// new frame. We ignore the overlap past the end of the buffer.
			copy(buf[:decodeBufferSize-N], buf[N:decodeBufferSize])

			// Extrapolate from the end of the excitation with a period of
			// "pitch_index", scaling down each period by a factor of "decay".
			extrapolationOffset := maxPeriodC - pitchIndex
			// We need enough samples to cover a complete MDCT window (including
			// overlap/2 samples on both sides).
			extrapolationLen := N + overlap
			// We also apply fading if this is not the first loss.
			attenuation := fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(fade, decay))
			S1 := int32(0)
			j := 0
			for i := 0; i < extrapolationLen; i, j = i+1, j+1 {
				if j >= pitchIndex {
					j -= pitchIndex
					attenuation = fixedmath.EXTRACT16(fixedmath.MULT16_16_Q15(attenuation, decay))
				}
				buf[decodeBufferSize-N+i] = fixedmath.SHL32(fixedmath.MULT16_16_Q15(attenuation, exc[extrapolationOffset+j]), sigShift)
				// Energy of the previously decoded signal whose excitation we copy.
				tmp := sround16(buf[decodeBufferSize-maxPeriodC-N+extrapolationOffset+j], sigShift)
				S1 += fixedmath.SHR32(fixedmath.MULT16_16(tmp, tmp), 11)
			}

			// Copy the last decoded samples (prior to the overlap region) to the
			// synthesis filter memory so we have a continuous signal, then apply
			// the synthesis filter to convert the excitation back to the signal.
			var lpcMem [celtLpcOrder]int16
			for i := 0; i < celtLpcOrder; i++ {
				lpcMem[i] = sround16(buf[decodeBufferSize-N-1-i], sigShift)
			}
			celtIir(buf[decodeBufferSize-N:], lpc[c*celtLpcOrder:], buf[decodeBufferSize-N:], extrapolationLen, celtLpcOrder, lpcMem[:])
			for i := 0; i < extrapolationLen; i++ {
				buf[decodeBufferSize-N+i] = fixedmath.SATURATE(buf[decodeBufferSize-N+i], sigSat)
			}

			// Check if the synthesis energy is higher than expected, which can
			// happen with signal changes during our window. If so, attenuate.
			{
				S2 := int32(0)
				for i := 0; i < extrapolationLen; i++ {
					tmp := sround16(buf[decodeBufferSize-N+i], sigShift)
					S2 += fixedmath.SHR32(fixedmath.MULT16_16(tmp, tmp), 11)
				}
				// This checks for an "explosion" in the synthesis.
				if !(S1 > fixedmath.SHR32(S2, 2)) {
					for i := 0; i < extrapolationLen; i++ {
						buf[decodeBufferSize-N+i] = 0
					}
				} else if S1 < S2 {
					ratio := fixedmath.EXTRACT16(fixedmath.Celt_sqrt(fixedmath.Frac_div32(fixedmath.SHR32(S1, 1)+1, S2+1)))
					for i := 0; i < overlap; i++ {
						tmpG := fixedmath.EXTRACT16(int32(q15one) - fixedmath.MULT16_16_Q15(window[i], fixedmath.EXTRACT16(int32(q15one)-int32(ratio))))
						buf[decodeBufferSize-N+i] = fixedmath.MULT16_32_Q15(tmpG, buf[decodeBufferSize-N+i])
					}
					for i := overlap; i < extrapolationLen; i++ {
						buf[decodeBufferSize-N+i] = fixedmath.MULT16_32_Q15(ratio, buf[decodeBufferSize-N+i])
					}
				}
			}
		}
		st.prefilterAndFold = 1
	}

	// Saturate to something large to avoid wrap-around.
	st.lossDuration = fixedmath.IMIN(10000, lossDuration+(1<<LM))
	st.plcDuration = fixedmath.IMIN(10000, st.plcDuration+(1<<LM))
	st.lastFrameType = currFrameType
}

// Decode decodes one CELT packet into interleaved int16 PCM (data-present path),
// creating a fresh range decoder over data. It returns the number of samples per
// channel produced. It is celt_decode_with_ec(st, data, len, pcm, frame_size,
// NULL, 0).
func (st *Decoder) Decode(data []byte, pcm []int16, frameSize int) (int, error) {
	return st.DecodeWithEC(data, pcm, frameSize, nil, 0)
}

// DecodeWithEC is celt_decode_with_ec (the celt_decode_with_ec_dred body with
// all DRED/QEXT/DEEP_PLC/CUSTOM_MODES branches removed for the frozen config).
// If dec is nil a range decoder is initialized over data; accum!=0 accumulates
// into pcm (used by the hybrid Opus path). Only the NORMAL (data present) path
// is implemented; a lost frame routes to the celtDecodeLost stub.
//
// Not safe for concurrent use: one Decoder per goroutine. The decoder state and
// its pooled scratch (see scratch.go) are mutated without synchronization.
func (st *Decoder) DecodeWithEC(data []byte, pcm []int16, frameSize int, dec *rangecoding.Decoder, accum int) (int, error) {
	m := st.mode
	CC := st.channels
	C := st.streamChannels
	nbEBands := m.nbEBands
	overlap := m.overlap
	eBands := m.eBands
	start := st.start
	end := st.end
	frameSize *= st.downsample

	oldBandE := st.oldEBands
	oldLogE := st.oldLogE
	oldLogE2 := st.oldLogE2
	backgroundLogE := st.backgroundLogE

	// Derive LM from the frame size (non-CUSTOM_MODES else branch,
	// celt_decoder.c:1256-1260).
	LM := -1
	for lm := 0; lm <= m.maxLM; lm++ {
		if m.shortMdctSize<<lm == frameSize {
			LM = lm
			break
		}
	}
	if LM < 0 || LM > m.maxLM {
		return 0, ErrBadArg
	}
	M := 1 << LM

	dlen := len(data)
	if dlen < 0 || dlen > 1275 || pcm == nil {
		return 0, ErrBadArg
	}

	N := M * m.shortMdctSize

	per := decodeBufferSize + overlap
	// Reuse the decoder's pointer-array storage instead of make([][]int32, CC) each
	// call (C uses stack pointer arrays). The headers are overwritten below before
	// any read, so nothing carries across frames.
	decMem := st.decMemSlices[:CC]
	outSyn := st.outSynSlices[:CC]
	for c := 0; c < CC; c++ {
		decMem[c] = st.decodeMem[c*per : (c+1)*per]
		outSyn[c] = decMem[c][decodeBufferSize-N:]
	}

	effEnd := end
	if effEnd > m.effEBands {
		effEnd = m.effEBands
	}

	if data == nil || dlen <= 1 {
		st.celtDecodeLost(decMem, N, LM)
		deemphasis(outSyn, pcm, N, CC, st.downsample, m.preemph, &st.preemphMemD, accum, &st.sc)
		return frameSize / st.downsample, nil
	}

	// Two consecutive received packets are required before turning on the
	// pitch-based PLC.
	if st.lossDuration == 0 {
		st.skipPlc = 0
	}

	var localDec rangecoding.Decoder
	if dec == nil {
		localDec.Init(data)
		dec = &localDec
	}

	if C == 1 {
		for i := 0; i < nbEBands; i++ {
			oldBandE[i] = fixedmath.MAX32(oldBandE[i], oldBandE[nbEBands+i])
		}
	}

	totalBits := dlen * 8
	tell := dec.Tell()

	var silence int
	switch {
	case tell >= totalBits:
		silence = 1
	case tell == 1:
		silence = dec.DecBitLogp(15)
	}
	if silence != 0 {
		// Pretend we've read all the remaining bits (celt_decoder.c:1322-1325).
		tell = dlen * 8
		dec.SetTellForSilence(tell)
	}

	postfilterGain := int16(0)
	postfilterPitch := 0
	postfilterTapset := 0
	if start == 0 && tell+16 <= totalBits {
		if dec.DecBitLogp(1) != 0 {
			octave := int(dec.DecUint(6))
			postfilterPitch = (16 << octave) + int(dec.DecBits(uint32(4+octave))) - 1
			qg := int(dec.DecBits(3))
			if dec.Tell()+2 <= totalBits {
				postfilterTapset = dec.DecIcdf(tapsetIcdf, 2)
			}
			postfilterGain = int16(3072 * (qg + 1)) // QCONST16(.09375f,15)*(qg+1)
		}
		tell = dec.Tell()
	}

	isTransient := 0
	if LM > 0 && tell+3 <= totalBits {
		isTransient = dec.DecBitLogp(3)
		tell = dec.Tell()
	}

	shortBlocks := 0
	if isTransient != 0 {
		shortBlocks = M
	}

	// Global flags: intra energy.
	intraEner := 0
	if tell+3 <= totalBits {
		intraEner = dec.DecBitLogp(3)
	}
	// Packet-loss recovery safety on the energy prediction (loss_duration != 0);
	// never taken in the no-loss sequence tests but ported for fidelity.
	if intraEner == 0 && st.lossDuration != 0 {
		for c := 0; c < 2; c++ {
			var safety int32
			missing := fixedmath.IMIN(10, st.lossDuration>>LM)
			switch LM {
			case 0:
				safety = gconst1_5
			case 1:
				safety = ghalf
			}
			for i := start; i < end; i++ {
				if oldBandE[c*nbEBands+i] < fixedmath.MAX32(oldLogE[c*nbEBands+i], oldLogE2[c*nbEBands+i]) {
					e0 := oldBandE[c*nbEBands+i]
					e1 := oldLogE[c*nbEBands+i]
					e2 := oldLogE2[c*nbEBands+i]
					slope := fixedmath.MAX32(e1-e0, fixedmath.HALF32(e2-e0))
					slope = fixedmath.MIN32(slope, gconst2)
					e0 -= fixedmath.MAX32(0, int32(1+missing)*slope)
					oldBandE[c*nbEBands+i] = fixedmath.MAX32(-gconst20, e0)
				} else {
					oldBandE[c*nbEBands+i] = fixedmath.MIN32(
						fixedmath.MIN32(oldBandE[c*nbEBands+i], oldLogE[c*nbEBands+i]),
						oldLogE2[c*nbEBands+i])
				}
				oldBandE[c*nbEBands+i] -= safety
			}
		}
	}

	// Coarse band energies.
	unquantCoarseEnergy(m, start, end, oldBandE, intraEner, dec, C, LM)

	tfRes := alloc(&st.sc.decTfRes, nbEBands) // VARDECL(int, tf_res)
	tfDecode(start, end, isTransient, tfRes, LM, dec)

	tell = dec.Tell()
	spreadDecision := spreadNormal
	if tell+4 <= totalBits {
		spreadDecision = dec.DecIcdf(spreadIcdf, 5)
	}

	cap := alloc(&st.sc.decCap, nbEBands) // VARDECL(int, cap)
	initCaps(m, cap, LM, C)

	offsets := alloc(&st.sc.decOffsets, nbEBands) // VARDECL(int, offsets)
	dynallocLogp := 6
	totalBits <<= bitRes // now in 1/8-bit units for dynalloc/trim
	tellFrac := int(dec.TellFrac())
	for i := start; i < end; i++ {
		width := C * (int(eBands[i+1]) - int(eBands[i])) << LM
		// quanta: 6 bits, but no more than 1 bit/sample and no less than 1/8.
		quanta := fixedmath.IMIN(width<<bitRes, fixedmath.IMAX(6<<bitRes, width))
		dynallocLoopLogp := dynallocLogp
		boost := 0
		for tellFrac+(dynallocLoopLogp<<bitRes) < totalBits && boost < cap[i] {
			flag := dec.DecBitLogp(uint32(dynallocLoopLogp))
			tellFrac = int(dec.TellFrac())
			if flag == 0 {
				break
			}
			boost += quanta
			totalBits -= quanta
			dynallocLoopLogp = 1
		}
		offsets[i] = boost
		if boost > 0 {
			dynallocLogp = fixedmath.IMAX(2, dynallocLogp-1)
		}
	}

	fineQuant := alloc(&st.sc.decFineQuant, nbEBands) // VARDECL(int, fine_quant)
	allocTrim := 5
	if tellFrac+(6<<bitRes) <= totalBits {
		allocTrim = dec.DecIcdf(trimIcdf, 7)
	}

	bits := (int32(dlen*8) << bitRes) - int32(dec.TellFrac()) - 1
	antiCollapseRsv := 0
	if isTransient != 0 && LM >= 2 && bits >= int32((LM+2)<<bitRes) {
		antiCollapseRsv = 1 << bitRes
	}
	bits -= int32(antiCollapseRsv)

	pulses := alloc(&st.sc.decPulses, nbEBands)             // VARDECL(int, pulses)
	finePriority := alloc(&st.sc.decFinePriority, nbEBands) // VARDECL(int, fine_priority)

	var intensity, dualStereo int
	var balance int32
	codedBands := cltComputeAllocation(m, start, end, offsets, cap, allocTrim,
		&intensity, &dualStereo, bits, &balance, pulses, fineQuant, finePriority,
		C, LM, nil, dec, 0, 0, 0, &st.sc)

	unquantFineEnergy(m, start, end, oldBandE, nil, fineQuant, dec, C)

	X := alloc(&st.sc.decX, C*N) // VARDECL(celt_norm, X)

	// Move the decode memory one frame to the left to make room for this frame.
	for c := 0; c < CC; c++ {
		copy(decMem[c][:per-N], decMem[c][N:per])
	}

	// Decode the fixed codebook (PVQ bands + folding + LCG noise).
	collapseMasks := alloc(&st.sc.decCollapseMasks, C*nbEBands) // VARDECL(unsigned char, collapse_masks)
	var Y []int32
	if C == 2 {
		Y = X[N:]
	}
	// The decoder's pooled scratch. Issue #21 pooled the ENCODE path and left decode
	// on per-call make(); issue #5 (this change) extends the pooling to decode after
	// auditing every decode buffer for write-before-read (see scratch.go). The same
	// st.sc also fed cltComputeAllocation above; that call finishes before this one,
	// and the two use disjoint fields, so sharing the instance is safe.
	quantAllBands(0, m, start, end, X, Y, collapseMasks, nil, pulses, shortBlocks,
		spreadDecision, dualStereo, intensity, tfRes,
		int32(dlen*(8<<bitRes)-antiCollapseRsv), balance, nil, dec, LM, codedBands,
		&st.rng, 0, st.disableInv, &st.sc)

	antiCollapseOn := 0
	if antiCollapseRsv > 0 {
		antiCollapseOn = int(dec.DecBits(1))
	}
	// Record per-frame coverage flags (test-only, no effect on decode state).
	st.lastTransient = isTransient != 0
	st.lastPostfilterOn = postfilterGain != 0
	st.lastAntiCollapse = antiCollapseOn != 0

	unquantEnergyFinalise(m, start, end, oldBandE, fineQuant, finePriority, dlen*8-dec.Tell(), dec, C)

	if antiCollapseOn != 0 {
		antiCollapse(m, X, collapseMasks, LM, C, N, start, end,
			oldBandE, oldLogE, oldLogE2, pulses, st.rng, 0)
	}

	if silence != 0 {
		for i := 0; i < C*nbEBands; i++ {
			oldBandE[i] = -gconst28
		}
	}

	if st.prefilterAndFold != 0 {
		st.foldPrefilter(decMem, N)
	}

	celtSynthesis(m, X, outSyn, oldBandE, start, effEnd, C, CC, isTransient, LM, st.downsample, silence, &st.sc)

	for c := 0; c < CC; c++ {
		if st.postfilterPeriod < combfilterMinperiod {
			st.postfilterPeriod = combfilterMinperiod
		}
		if st.postfilterPeriodOld < combfilterMinperiod {
			st.postfilterPeriodOld = combfilterMinperiod
		}
		base := decodeBufferSize - N
		combFilter(decMem[c], base, decMem[c], base,
			st.postfilterPeriodOld, st.postfilterPeriod, m.shortMdctSize,
			st.postfilterGainOld, st.postfilterGain,
			st.postfilterTapsetOld, st.postfilterTapset, overlap, m.window)
		if LM != 0 {
			off := base + m.shortMdctSize
			combFilter(decMem[c], off, decMem[c], off,
				st.postfilterPeriod, postfilterPitch, N-m.shortMdctSize,
				st.postfilterGain, postfilterGain,
				st.postfilterTapset, postfilterTapset, overlap, m.window)
		}
	}
	st.postfilterPeriodOld = st.postfilterPeriod
	st.postfilterGainOld = st.postfilterGain
	st.postfilterTapsetOld = st.postfilterTapset
	st.postfilterPeriod = postfilterPitch
	st.postfilterGain = postfilterGain
	st.postfilterTapset = postfilterTapset
	if LM != 0 {
		st.postfilterPeriodOld = st.postfilterPeriod
		st.postfilterGainOld = st.postfilterGain
		st.postfilterTapsetOld = st.postfilterTapset
	}

	if C == 1 {
		copy(oldBandE[nbEBands:2*nbEBands], oldBandE[:nbEBands])
	}

	// Update energy prediction history.
	if isTransient == 0 {
		copy(oldLogE2, oldLogE[:2*nbEBands])
		copy(oldLogE, oldBandE[:2*nbEBands])
	} else {
		for i := 0; i < 2*nbEBands; i++ {
			oldLogE[i] = fixedmath.MIN32(oldLogE[i], oldBandE[i])
		}
	}
	// Background-noise estimate update.
	maxBackgroundIncrease := int32(fixedmath.IMIN(160, st.lossDuration+M)) * gconst0_001
	for i := 0; i < 2*nbEBands; i++ {
		backgroundLogE[i] = fixedmath.MIN32(backgroundLogE[i]+maxBackgroundIncrease, oldBandE[i])
	}
	// Clear the bands outside [start,end) in case they change.
	for c := 0; c < 2; c++ {
		for i := 0; i < start; i++ {
			oldBandE[c*nbEBands+i] = 0
			oldLogE[c*nbEBands+i] = -gconst28
			oldLogE2[c*nbEBands+i] = -gconst28
		}
		for i := end; i < nbEBands; i++ {
			oldBandE[c*nbEBands+i] = 0
			oldLogE[c*nbEBands+i] = -gconst28
			oldLogE2[c*nbEBands+i] = -gconst28
		}
	}
	st.rng = dec.Rng()

	deemphasis(outSyn, pcm, N, CC, st.downsample, m.preemph, &st.preemphMemD, accum, &st.sc)
	st.lossDuration = 0
	st.plcDuration = 0
	st.lastFrameType = frameNormal
	st.prefilterAndFold = 0

	if dec.Tell() > 8*dlen {
		return 0, ErrInternal
	}
	if dec.Error() != 0 {
		st.err = 1
	}
	return frameSize / st.downsample, nil
}

// --- Persistent-state snapshot and hash for the differential test ----------

// State is a copy of the CELT decoder's cross-frame persistent state
// (celt_decoder.c struct fields that carry between frames). The differential
// suite hashes / compares this after every frame so a divergence is caught on
// the frame it first appears (docs/hard-parts.md section 5).
type State struct {
	Rng                 uint32
	PostfilterPeriod    int
	PostfilterPeriodOld int
	PostfilterGain      int16
	PostfilterGainOld   int16
	PostfilterTapset    int
	PostfilterTapsetOld int
	PrefilterAndFold    int
	LossDuration        int
	PlcDuration         int
	SkipPlc             int
	LastFrameType       int
	LastPitchIndex      int
	PreemphMemD         [2]int32
	DecodeMem           []int32
	OldEBands           []int32
	OldLogE             []int32
	OldLogE2            []int32
	BackgroundLogE      []int32
	Lpc                 []int16
}

// State returns a deep copy of the decoder's persistent cross-frame state.
func (st *Decoder) State() State {
	s := State{
		Rng:                 st.rng,
		PostfilterPeriod:    st.postfilterPeriod,
		PostfilterPeriodOld: st.postfilterPeriodOld,
		PostfilterGain:      st.postfilterGain,
		PostfilterGainOld:   st.postfilterGainOld,
		PostfilterTapset:    st.postfilterTapset,
		PostfilterTapsetOld: st.postfilterTapsetOld,
		PrefilterAndFold:    st.prefilterAndFold,
		LossDuration:        st.lossDuration,
		PlcDuration:         st.plcDuration,
		SkipPlc:             st.skipPlc,
		LastFrameType:       st.lastFrameType,
		LastPitchIndex:      st.lastPitchIndex,
		PreemphMemD:         st.preemphMemD,
		DecodeMem:           append([]int32(nil), st.decodeMem...),
		OldEBands:           append([]int32(nil), st.oldEBands...),
		OldLogE:             append([]int32(nil), st.oldLogE...),
		OldLogE2:            append([]int32(nil), st.oldLogE2...),
		BackgroundLogE:      append([]int32(nil), st.backgroundLogE...),
		Lpc:                 append([]int16(nil), st.lpc...),
	}
	return s
}

// Hash returns an FNV-1a-64 hash over a canonical little-endian serialization of
// the persistent state. The same function hashes both the Go decoder's own
// State and the State built from the C decoder's dumped fields, so the two
// hashes are directly comparable with no cross-language serialization risk.
func (s State) Hash() uint64 {
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
	putI(s.PostfilterPeriod)
	putI(s.PostfilterPeriodOld)
	putI16(s.PostfilterGain)
	putI16(s.PostfilterGainOld)
	putI(s.PostfilterTapset)
	putI(s.PostfilterTapsetOld)
	putI(s.PrefilterAndFold)
	putI(s.LossDuration)
	putI(s.PlcDuration)
	putI(s.SkipPlc)
	putI(s.LastFrameType)
	putI(s.LastPitchIndex)
	putI32(s.PreemphMemD[0])
	putI32(s.PreemphMemD[1])
	for _, v := range s.DecodeMem {
		putI32(v)
	}
	for _, v := range s.OldEBands {
		putI32(v)
	}
	for _, v := range s.OldLogE {
		putI32(v)
	}
	for _, v := range s.OldLogE2 {
		putI32(v)
	}
	for _, v := range s.BackgroundLogE {
		putI32(v)
	}
	for _, v := range s.Lpc {
		putI16(v)
	}
	return h.Sum64()
}
