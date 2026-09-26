// Transliteration of celt/laplace.c (libopus v1.6.1, commit 3da9f7a6): the
// Laplace-like entropy coder used by the CELT coarse-energy quantizer
// (quant_coarse_energy / unquant_coarse_energy in quant_bands.go). Both
// directions live here: ec_laplace_decode (phase 2) and ec_laplace_encode
// (phase 4), sharing the ec_laplace_get_freq1 helper. The ec_laplace_*_p0
// variants are ENABLE_QEXT-only (their sole callers are QEXT code, which is off
// in the frozen build), so they are intentionally not ported.

package celt

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
)

// Laplace model constants (celt/laplace.c:37-41).
const (
	// laplaceLogMinP: LAPLACE_LOG_MINP, log2 of the minimum probability of an
	// energy delta (out of 32768).
	laplaceLogMinP = 0
	// laplaceMinP: LAPLACE_MINP = 1<<LAPLACE_LOG_MINP.
	laplaceMinP = 1 << laplaceLogMinP
	// laplaceNmin: LAPLACE_NMIN, the minimum number of guaranteed representable
	// energy deltas (in one direction).
	laplaceNmin = 16
)

// ecLaplaceGetFreq1 mirrors the static ec_laplace_get_freq1 (celt/laplace.c:44).
// When called, decay is positive and at most 11456. All arithmetic is unsigned
// 32-bit, matching C's `unsigned` result (the (opus_int32) cast of 16384-decay
// is folded back into the unsigned multiply by the usual conversions).
func ecLaplaceGetFreq1(fs0 uint32, decay int) uint32 {
	ft := uint32(32768-laplaceMinP*(2*laplaceNmin)) - fs0
	return (ft * uint32(16384-decay)) >> 15
}

// ecLaplaceEncode ports ec_laplace_encode (celt/laplace.c:51): it encodes a
// Laplace-distributed value. fs is the probability of 0 (times 32768); decay is
// the probability of +/-1 (times 16384, at most 11456). value is an in/out
// pointer: when the value lands in the flat tail of the PDF the function clamps
// *value to the representable magnitude, exactly like the C out-parameter, which
// quant_coarse_energy_impl relies on for its error/badness accounting. fl/fs are
// unsigned so the wrapping arithmetic matches C bit for bit.
func ecLaplaceEncode(enc *rangecoding.Encoder, value *int, fs uint32, decay int) {
	var fl uint32
	val := *value
	fl = 0
	if val != 0 {
		var s int
		var i int
		s = 0
		if val < 0 {
			s = -1
		}
		val = (val + s) ^ s
		fl = fs
		fs = ecLaplaceGetFreq1(fs, decay)
		// Search the decaying part of the PDF.
		for i = 1; fs > 0 && i < val; i++ {
			fs *= 2
			fl += fs + 2*laplaceMinP
			fs = (fs * uint32(decay)) >> 15
		}
		// Everything beyond that has probability LAPLACE_MINP.
		if fs == 0 {
			var di int
			var ndiMax int
			// C's ndi_max is a 32-bit int, so the unsigned expression is
			// reinterpreted through int32 (matching a wrap to negative), not
			// widened to Go's 64-bit int.
			ndiMax = int(int32((32768 - fl + laplaceMinP - 1) >> laplaceLogMinP))
			ndiMax = (ndiMax - s) >> 1
			di = fixedmath.IMIN(val-i, ndiMax-1)
			fl += uint32((2*di + 1 + s) * laplaceMinP)
			// fs = IMIN(LAPLACE_MINP, 32768-fl), computed unsigned to match C.
			t := uint32(32768) - fl
			if uint32(laplaceMinP) < t {
				fs = uint32(laplaceMinP)
			} else {
				fs = t
			}
			*value = (i + di + s) ^ s
		} else {
			fs += laplaceMinP
			fl += fs & uint32(^s)
		}
	}
	enc.EncodeBin(fl, fl+fs, 15)
}

// ecLaplaceDecode ports ec_laplace_decode (celt/laplace.c:94): it decodes a
// value assumed to be the realisation of a Laplace-distributed process. fs is
// the probability of 0 (times 32768); decay is the probability of +/-1 (times
// 16384). fl/fs/fm are unsigned, exactly as in C, so the wrapping arithmetic
// matches bit for bit.
func ecLaplaceDecode(dec *rangecoding.Decoder, fs uint32, decay int) int {
	val := 0
	var fl uint32
	fm := dec.DecodeBin(15)
	fl = 0
	if fm >= fs {
		val++
		fl = fs
		fs = ecLaplaceGetFreq1(fs, decay) + laplaceMinP
		// Search the decaying part of the PDF.
		for fs > laplaceMinP && fm >= fl+2*fs {
			fs *= 2
			fl += fs
			fs = ((fs - 2*laplaceMinP) * uint32(decay)) >> 15
			fs += laplaceMinP
			val++
		}
		// Everything beyond that has probability LAPLACE_MINP.
		if fs <= laplaceMinP {
			di := (fm - fl) >> (laplaceLogMinP + 1)
			val += int(di)
			fl += 2 * di * laplaceMinP
		}
		if fm < fl+fs {
			val = -val
		} else {
			fl += fs
		}
	}
	// ec_dec_update(dec, fl, IMIN(fl+fs,32768), 32768).
	fh := fl + fs
	if fh > 32768 {
		fh = 32768
	}
	dec.DecUpdate(fl, fh, 32768)
	return val
}
