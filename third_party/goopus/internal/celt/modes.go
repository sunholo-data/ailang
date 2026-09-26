package celt

// This file hand-writes the CELT mode struct types that mirror libopus's
// CELTMode (celt/modes.h OpusCustomMode) and its sub-structs, for the frozen
// build config: FIXED_POINT, DISABLE_FLOAT_API, non-CUSTOM_MODES, non-QEXT.
// The actual data (mode48000_960 and its tables) is machine-generated into
// static_modes_gen.go by cmd/gentables; see docs/hard-parts.md section 6.
//
// Type mapping for the frozen config (celt/arch.h, celt/kiss_fft.h):
//   celt_coef            = opus_val16 = opus_int16 -> int16 (non-QEXT)
//   kiss_twiddle_scalar  = celt_coef              -> int16
//   kiss_fft_scalar      = opus_int32             -> int32 (runtime, not stored here)
//   opus_val16           = opus_int16             -> int16
// The QEXT tables (qext_cache, qext twiddles/states) and the qext_cache field
// are intentionally omitted: they only exist under ENABLE_QEXT, which is off.

//go:generate go run github.com/sunholo-data/ailang/third_party/goopus/cmd/gentables -out static_modes_gen.go

// maxFactors mirrors MAXFACTORS in celt/kiss_fft.h; the factors array holds
// 2*MAXFACTORS entries.
const maxFactors = 8

// kissTwiddleCpx mirrors celt/kiss_fft.h kiss_twiddle_cpx in the FIXED_POINT,
// non-QEXT build: r and i are celt_coef (opus_int16).
type kissTwiddleCpx struct {
	r int16
	i int16
}

// kissFFTState mirrors celt/kiss_fft.h kiss_fft_state (FIXED_POINT, non-QEXT).
// scaleShift exists only under FIXED_POINT (it is present in the frozen build).
// The upstream arch_fft pointer is always NULL in the static, non-OVERRIDE_FFT
// build, so it is omitted here. bitrev and twiddles are shared slices into the
// package-level tables (all four states point at the same twiddle table).
type kissFFTState struct {
	nfft       int
	scale      int16 // celt_coef
	scaleShift int   // scale_shift
	shift      int
	factors    [2 * maxFactors]int16
	bitrev     []int16
	twiddles   []kissTwiddleCpx
	// packedTw holds the strided-to-contiguous twiddle runs the radix-3/5
	// butterflies consume, indexed by [stage][k] (k selects the j*(k+1)*stride
	// gather). It is written only during init and read-only thereafter, so
	// concurrent FFT calls index it without synchronization. Rows for radix-2/4
	// stages stay nil. See buildPackedTwiddles in kiss_fft.go.
	packedTw [maxFactors][4][]int16
}

// mdctLookup mirrors celt/mdct.h mdct_lookup. kfft holds the four decimation
// stages; trig is the shared MDCT twiddle table (kiss_twiddle_scalar = int16).
type mdctLookup struct {
	n        int
	maxshift int
	kfft     [4]*kissFFTState
	trig     []int16
}

// pulseCache mirrors celt/modes.h PulseCache: the prebuilt PVQ pulse cache that
// rate.c consumes. size is the length of bits (cache.size in the C struct).
type pulseCache struct {
	size  int
	index []int16
	bits  []byte
	caps  []byte
}

// celtMode mirrors celt/modes.h OpusCustomMode (typedef CELTMode) for the
// frozen FIXED_POINT, DISABLE_FLOAT_API, non-CUSTOM_MODES, non-QEXT build.
// Field order and names follow the C struct so the port stays diffable.
type celtMode struct {
	Fs        int32
	overlap   int
	nbEBands  int
	effEBands int
	preemph   [4]int16 // opus_val16
	eBands    []int16  // "pseudo-critical band" boundaries

	maxLM         int
	nbShortMdcts  int
	shortMdctSize int

	nbAllocVectors int
	allocVectors   []byte // bits per band for several rates
	logN           []int16

	window []int16 // celt_coef
	mdct   mdctLookup
	cache  pulseCache
}
