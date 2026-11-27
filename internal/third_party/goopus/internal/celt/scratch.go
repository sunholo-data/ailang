// Codec scratch pool. libopus takes every per-frame working buffer off the C
// stack with VARDECL(x,n) / ALLOC(x,n,T) (alloca): free to obtain, reclaimed on
// frame pop. The transliteration mapped each ALLOC to a Go make(), which is a heap
// allocation: 501 objects and 145 KB per stereo 20 ms encode, 93 objects and 38 KB
// per stereo 20 ms decode. This file holds them on the Encoder and the Decoder
// instead, so a steady-state Encode or Decode allocates nothing. One scratch type
// serves both: a given codec instance only ever runs one side, so an encode field
// and a decode field never hold live data in the same scratch at the same time.
// Most decode-only buffers live in the "DECODE PATH" group at the end; the decode
// path ALSO draws on fields declared in the shared-function groups below, because
// the declaring C function runs on both sides: norm and hadamardTmp (quant_all_bands),
// bits1/bits2/thresh/trimOffset (clt_compute_allocation), and mdctF2 (the MDCT).
// All of them are part of the decode zeroing audit exactly like the dec* fields.
//
// THE ZEROING CONTRACT: the one thing that can silently break bit-exactness here.
// Three behaviours must not be confused:
//
//	C's stack ALLOC   is UNINITIALISED: it carries whatever the last stack frame left.
//	Go's make()       ZEROES.
//	A pooled buffer   carries THE PREVIOUS FRAME'S VALUES.
//
// Every site that draws from this scratch was audited to write its entire read
// window before reading it, which collapses those three behaviours into one. That
// is not luck: libopus is valgrind-clean, so a plain ALLOC it reads before writing
// would be reading uninitialised stack, and the codec would not be deterministic.
// The ENCODE sites where C does NOT rely on that keep an explicit clear at the
// point of use, and there are three:
//
//	offsets           C itself issues OPUS_CLEAR(offsets, nbEBands) (celt_encoder.c:1066),
//	                  so dynalloc_analysis re-initialises it on every call.
//	surround_dynalloc C issues OPUS_CLEAR(surround_dynalloc, end) (celt_encoder.c:2107).
//	importance        no OPUS_CLEAR in C, but its read window in tf_analysis exceeds
//	                  dynalloc_analysis' write window when start>0, so a pooled buffer
//	                  would read the previous frame there.
//
// Each is commented at its clear.
//
// THE DECODE PATH NEEDS NO CLEARS. The comment at celt_decoder.go's quantAllBands
// call once warned that the encoder audit above did NOT cover decode; this is that
// audit. Every decode buffer was checked the same way, and all of them write their
// entire read window before any read, so none needs a clear:
//
//	decIy             cwrsi (decode_pulses) fills all N before normalise_residual /
//	                  extract_collapse_mask read it.
//	decTfRes          tf_decode writes [start,end); quant_all_bands reads [start,end).
//	decCap            init_caps writes all nbEBands; only [start,end) is read.
//	decOffsets        the dynalloc loop writes [start,end); clt_compute_allocation
//	                  reads offsets[j] only for j in [start,end). Note C clears offsets
//	                  in the ENCODER (OPUS_CLEAR at celt_encoder.c:1066) but NOT in the
//	                  decoder: the decoder's read window never leaves the write window.
//	decFineQuant      }
//	decPulses         } outputs of clt_compute_allocation, written over [start,end),
//	decFinePriority   } read over [start,end) by the fine-energy / band / anti-collapse
//	                    passes.
//	decX              quant_all_bands writes every bin of bands [start,end); celt_synthesis
//	                  reads X only over [M*eBands[start], M*eBands[effEnd]) with effEnd<=end.
//	                  Reused by celtDecodeLost's noise-PLC branch: there the LCG-noise fill
//	                  writes bands [start,effEnd) and celt_synthesis reads exactly that same
//	                  window, so the identical write-before-read bound holds on that path too.
//	decCollapseMasks  each band writes collapse_masks[i*C..] as it finishes; the fold
//	                  reads only lower, already-written bands (foldI<i) and anti_collapse
//	                  reads [start,end). Intra-frame write-before-read.
//	decFreq           denormalise_bands writes all N (band bins, then OPUS_CLEAR to N).
//	mdctF2            clt_mdct_backward writes all N4 (bitrev is a permutation) before
//	                  the post-rotation reads them; shared with the forward MDCT, which a
//	                  Decoder never runs (Encoder and Decoder own separate scratch).
//	decDeemph         written [0,N) per channel before the downsample read (downsample>1).
//	                  Only allocated for sub-48 kHz output; TestDecoderScratchCarryover
//	                  decodes at 24 kHz so the poison run reaches it.
//
//	The PLC buffers (lost-frame path only) follow the same rule:
//	decExcBuf         celtDecodeLost's per-channel copy loop rewrites all MAX_PERIOD+
//	                  CELT_LPC_ORDER samples before celt_fir / celt_autocorr read them.
//	decFirTmp         celt_fir writes all exc_length outputs before the copy-back into
//	                  exc reads them; exc_length shrinks freely since the read window is
//	                  the same [0,exc_length) that was just written.
//	decLpPitchBuf     pitch_downsample writes every element (the downsample loop fills
//	                  [0,len), then celt_fir5 filters in place) before pitch_search reads.
//	decEtmp           comb_filter writes all `overlap` samples before the TDAC fold reads
//	                  them; etmp is local to prefilter_and_fold (the cross-frame coupling
//	                  of prefilter_and_fold lives in the flag and decode_mem, not here).
//	norm, hadamardTmp (shared with encode; declared in the quant_all_bands and
//	bits1, bits2,      Hadamard groups below) and the clt_compute_allocation
//	thresh, trimOffset quartet: on decode these follow the same [start,end)-bounded
//	                  write-before-read pattern as their encode use; norm's lowband
//	                  fill is contiguous from start, and rate.go writes bits1/bits2/
//	                  thresh/trimOffset over [start,end) before any read.
//
// This was not taken on trust: a poison pass that filled every pooled buffer with
// 0x5A before each decode call left the differential and conformance suites bit-exact.
//
// NOT SAFE FOR CONCURRENT USE, exactly like the Encoder or Decoder that owns it (one
// per goroutine, as documented on EncodeWithEC / DecodeWithEC). That is why the
// buffers need neither a mutex nor a sync.Pool: C gets the same isolation for free
// from having one stack per thread, and locking here would be pure overhead.

// The alloc helper (the Go stand-in for ALLOC(x, n, T)) lives in scratch_alloc.go;
// a build-tagged copy in scratch_poison.go fills each window with 0x5A under
// -tags poison, the read-before-write audit harness this file's contract rests on.

package celt

// scratch pools one Encoder's or Decoder's VARDECL buffers. Fields are named after the C
// variable they stand in for and grouped by the function that declares them; two
// sites share a field only where their lifetimes provably cannot overlap.
type scratch struct {
	// celt_encode_with_ec (celt_encoder.c:1726). All live at once across the frame.
	in               []int32 // :1968  CC*(N+overlap)
	freq             []int32 // :2059  CC*N
	bandE            []int32 // :2060  nbEBands*CC
	bandLogE         []int32 // :2061  nbEBands*CC
	bandLogE2        []int32 // :2074  C*nbEBands
	surroundDynalloc []int32 // :2106  C*nbEBands   (explicitly cleared; see below)
	X                []int32 // :2238  C*N
	offsets          []int   // :2245  nbEBands     (explicitly cleared; see below)
	importance       []int   // :2246  nbEBands     (explicitly cleared; see below)
	spreadWeight     []int   // :2247  nbEBands
	tfRes            []int   // :2255  nbEBands
	energyErr        []int32 // :2290  C*nbEBands   (C: error[])
	caps             []int   // :2356  nbEBands
	fineQuant        []int   // :2591  nbEBands
	pulses           []int   // :2592  nbEBands
	finePriority     []int   // :2593  nbEBands
	collapseMasks    []byte  // :2646  C*nbEBands

	// transient_analysis (:267) / tone_detect (:1362). Sequential, but kept apart:
	// sharing would silently couple two independent analyses if one ever nested.
	transientTmp []int16 // :283   N+overlap
	toneX        []int16 // :1368  N

	// run_prefilter (:1404) and the pitch analysis it drives.
	pre      []int32    // :1415  CC*(N+maxPeriod), the largest single buffer
	pitchBuf []int16    // :1440  (maxPeriod+N)>>1
	preCh    [2][]int32 // :1441  the 2-element pointer array; a value field, never allocated

	// pitch_search (pitch.c:307) / remove_doubling (pitch.c:454).
	xLp4       []int16 // x_lp4
	yLp4       []int16 // y_lp4
	pitchXcorr []int32 // xcorr, maxPitch>>1
	yyLookup   []int32 // yy_lookup, maxperiod+1

	// _celt_autocorr (celt_lpc.c:284).
	autocorrXX []int16 // xx, n

	// clt_mdct_forward (mdct.c:107) on the encode side, clt_mdct_backward (mdct.c:268)
	// on the decode side. Called up to CC*B times per frame, so a large share of the
	// object count even though each buffer is small. mdctF2 is shared by the forward
	// and backward transforms: a single codec instance runs only one of the two, so
	// they never contend (Encoder and Decoder own separate scratch instances). The
	// backward transform uses only f2 (its FFT runs in the caller's overlap buffer).
	mdctF  []int32      // f,  N2  (forward only)
	mdctF2 []kissFFTCpx // f2, N4

	// dynalloc_analysis (celt_encoder.c:1049).
	follower   []int32 // C*nbEBands
	noiseFloor []int32 // C*nbEBands
	bandLogE3  []int32 // nbEBands
	mask       []int32 // nbEBands
	sig        []int32 // nbEBands

	// tf_analysis (celt_encoder.c:663).
	tfMetric []int   // metric, len
	tfTmp    []int32 // tmp,  widest band << LM
	tfTmp1   []int32 // tmp_1
	tfPath0  []int   // path0, len
	tfPath1  []int   // path1, len

	// quant_coarse_energy (quant_bands.c:260).
	oldEBandsIntra []int32 // C*nbEBands
	errorIntra     []int32 // C*nbEBands
	intraBits      []byte  // intra_buf window, bounded by the 1275-byte packet cap

	// clt_compute_allocation (rate.c:377).
	bits1      []int // len
	bits2      []int // len
	thresh     []int // len
	trimOffset []int // trim_offset

	// quant_all_bands (bands.c:1589).
	norm           []int32 // _norm, C*normLen
	lowbandScratch []int32 // only allocated on the encode+resynth path
	XSave          []int32 // the five theta-RDO save buffers, each resynthAlloc
	YSave          []int32
	XSave2         []int32
	YSave2         []int32
	normSave2      []int32
	bytesSave      []byte // 1275, theta-RDO only

	// deinterleave_hadamard (bands.c:574) / interleave_hadamard (bands.c:600). One
	// field for both: each call copies tmp back into X before returning, and
	// quant_band is not recursive (only quant_partition is, and it never reorders).
	hadamardTmp []int32

	// op_pvq_search (vq.c:205) / alg_quant (vq.c:550). alg_quant is a leaf: it
	// consumes iy entirely before returning, so quant_partition's recursion cannot
	// hold two of these live at once. N is bounded by the widest band times M (176).
	pvqY     []int32 // y
	pvqSignx []int   // signx
	pvqIy    []int32 // iy

	// --- DECODE PATH ---
	// celt_decode_with_ec (celt_decoder.c:1100) and the functions it calls. A
	// Decoder funnels its per-frame VARDECLs through this same pool; it never runs
	// any of the encode functions above, so a decode field and an encode field never
	// hold live data in one scratch at once. Every buffer below writes its entire
	// read window before reading it (the decode zeroing audit is in the file
	// comment), so none is ever cleared. C line numbers are celt_decoder.c.
	decIy            []int32 // vq.c:619 alg_unquant iy, widest band times M
	decTfRes         []int   // :1395 tf_res,         nbEBands
	decCap           []int   // :1403 cap,            nbEBands
	decOffsets       []int   // :1407 offsets,        nbEBands
	decFineQuant     []int   // :1440 fine_quant,     nbEBands
	decPulses        []int   // :1448 pulses,         nbEBands
	decFinePriority  []int   // :1449 fine_priority,  nbEBands
	decX             []int32 // :1457 X,              C*N
	decCollapseMasks []byte  // :1487 collapse_masks, C*nbEBands
	decFreq          []int32 // :432  celt_synthesis freq, N
	decDeemph        []int32 // :335  deemphasis scratch,  N (downsample>1 only)

	// PLC (packet-loss concealment) working buffers: celtDecodeLost (celt_decode_lost)
	// and the two helpers it calls, celtPlcPitchSearch (celt_plc_pitch_search) and
	// foldPrefilter (prefilter_and_fold). Reached only on a lost frame, so they are
	// never live while the normal-decode buffers above are, and they never carry data
	// a later frame reads. The noise-PLC spectrum reuses decX (same C*N buffer, same
	// celt_synthesis read window) and the outSyn pointer array reuses the Decoder's
	// outSynSlices, so neither needs a field here. Every buffer below writes its whole
	// read window before any read, so none is cleared (audit in the file comment above).
	decExcBuf     []int16 // celt_decode_lost _exc,          MAX_PERIOD+CELT_LPC_ORDER
	decFirTmp     []int16 // celt_decode_lost fir_tmp,       exc_length
	decLpPitchBuf []int16 // celt_plc_pitch_search lp_pitch_buf, DECODE_BUFFER_SIZE>>1
	decEtmp       []int32 // prefilter_and_fold etmp,        overlap
}
