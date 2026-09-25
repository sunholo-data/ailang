// Port of celt/rate.c and the cache-driven helpers of celt/rate.h (libopus
// v1.6.1, commit 3da9f7a6) for the frozen FIXED_POINT, DISABLE_FLOAT_API,
// non-CUSTOM_MODES, non-ENABLE_QEXT build: CELT bit allocation.
// clt_compute_allocation reserves the skip/intensity/dual-stereo bits (exact
// eighth-bit BITRES arithmetic, exact order), runs a 6-step bisection over the
// static allocVectors quality curves, then interp_bits2pulses walks bands
// top-down deciding skips WHILE encoding/decoding the skip flags interleaved with
// the budget math. init_caps builds the per-band cap[] from the prebuilt
// cache_caps50 table.
//
// The eighth-bit (BITRES) arithmetic must reproduce C's exact form, operator
// precedence and evaluation order: correctness is proved by the EXHAUSTIVE
// differential sweep in internal/reftest/oracle (rate_test.go), not by review,
// and the thresh, cap and balance-carry corner cases are unenumerable by
// inspection. Index shape may be idiomatic Go where the gate confirms it (the
// bounds-check-eliding slice views below are bit-exact); the arithmetic may not.
//
// C's single ec_ctx serves both encode and decode; the Go range coder is two
// distinct types, so interpBits2pulses/cltComputeAllocation take BOTH an
// *rangecoding.Encoder and an *rangecoding.Decoder and pick by the `encode` flag
// (encode!=0 uses enc, encode==0 uses dec), exactly as C's `encode` gate does.
//
// The pulse cache that these helpers consume is prebuilt in static_modes_gen.go
// (docs/hard-parts.md section 6), so compute_pulse_cache and its cwrs.c helper
// get_required_bits/log2_frac are NOT ported here: they run only under
// CUSTOM_MODES to BUILD the cache, are unreachable on the decode path, and are
// already deferred to phase 4 by internal/celt/cwrs.go.

package celt

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
)

// Constants from rate.h / entcode.h (BITRES). maxFineBits (MAX_FINE_BITS=8) is
// already defined in quant_bands.go and is reused here.
const (
	bitRes       = 3  // BITRES (celt/entcode.h): eighth-bit fixed-point for rates
	allocSteps   = 6  // ALLOC_STEPS (rate.c)
	fineOffset   = 21 // FINE_OFFSET (rate.h)
	logMaxPseudo = 6  // LOG_MAX_PSEUDO (rate.h)
)

// log2FracTable mirrors LOG2_FRAC_TABLE (rate.c:43): the number of eighth-bits
// (BITRES units) to code a value uniformly in [0, i).
var log2FracTable = [24]int{
	0,
	8, 13,
	16, 19, 21, 23,
	24, 26, 27, 28, 29, 30, 31, 32,
	32, 33, 34, 34, 35, 36, 36, 37, 37,
}

// getPulses mirrors get_pulses (rate.h:48): the pseudo-pulse count for index i.
func getPulses(i int) int {
	if i < 8 {
		return i
	}
	return (8 + (i & 7)) << ((i >> 3) - 1)
}

// bits2pulses mirrors bits2pulses (rate.h:53): binary-search the prebuilt pulse
// cache for the pulse count whose bit cost is closest to `bits`.
func bits2pulses(m *celtMode, band, LM, bits int) int {
	var i int
	var lo, hi int
	var cache []byte

	LM++
	cache = m.cache.bits[int(m.cache.index[LM*m.nbEBands+band]):]

	lo = 0
	hi = int(cache[0])
	bits--
	for i = 0; i < logMaxPseudo; i++ {
		mid := (lo + hi + 1) >> 1
		/* OPT: Make sure this is implemented with a conditional move */
		if int(cache[mid]) >= bits {
			hi = mid
		} else {
			lo = mid
		}
	}
	loBits := -1
	if lo != 0 {
		loBits = int(cache[lo])
	}
	if bits-loBits <= int(cache[hi])-bits {
		return lo
	}
	return hi
}

// pulses2bits mirrors pulses2bits (rate.h:80): the bit cost of `pulses` pulses.
func pulses2bits(m *celtMode, band, LM, pulses int) int {
	var cache []byte

	LM++
	cache = m.cache.bits[int(m.cache.index[LM*m.nbEBands+band]):]
	if pulses == 0 {
		return 0
	}
	return int(cache[pulses]) + 1
}

// initCaps mirrors init_caps (celt/celt.c:329): the per-band bit cap in
// eighth-bits, derived from the prebuilt cache_caps50 table.
func initCaps(m *celtMode, cap []int, LM, C int) {
	var i int
	for i = 0; i < m.nbEBands; i++ {
		var N int
		N = (int(m.eBands[i+1]) - int(m.eBands[i])) << LM
		cap[i] = ((int(m.cache.caps[m.nbEBands*(2*LM+C-1)+i]) + 64) * C * N) >> 2
	}
}

// interpBits2pulses mirrors interp_bits2pulses (rate.c:249). The skip loop
// mutates the same psum/total that the reservation logic reads; the intensity
// reservation shrinks from LOG2_FRAC_TABLE[end-start] to LOG2_FRAC_TABLE[j-start]
// as bands are skipped. Ported exactly, both directions (see file header).
func interpBits2pulses(m *celtMode, start, end, skip_start int,
	bits1, bits2, thresh, cap []int, total int32, _balance *int32,
	skip_rsv int, intensity *int, intensity_rsv int, dual_stereo *int, dual_stereo_rsv int,
	bits, ebits, fine_priority []int, C, LM int,
	enc *rangecoding.Encoder, dec *rangecoding.Decoder, encode, prev, signalBandwidth int) int {
	var psum int32
	var lo, hi int
	var i, j int
	var logM int
	var stereo int
	var codedBands int // C: int codedBands=-1 (defensive default, always set by the skip loop)
	var alloc_floor int
	var left, percoeff int32
	var done int
	var balance int32

	alloc_floor = C << bitRes
	if C > 1 {
		stereo = 1
	} else {
		stereo = 0
	}

	logM = LM << bitRes
	lo = 0
	hi = 1 << allocSteps
	// Index the per-band allocation arrays through views resliced to their exact
	// working extent [0,end) so the bounds prover elides the per-band IsInBounds
	// checks in the bisection and final-walk loops below (j is always < end).
	// Same backing arrays, same indices, identical eighth-bit arithmetic and
	// order: bit-exact by construction, gate-verified (rate_test.go decode,
	// rateenc_test.go encode).
	b1 := bits1[:end]
	b2 := bits2[:end]
	th := thresh[:end]
	cp := cap[:end]
	bo := bits[:end]
	for i = 0; i < allocSteps; i++ {
		mid := (lo + hi) >> 1
		psum = 0
		done = 0
		for j := end - 1; j >= start; j-- {
			tmp := b1[j] + int(int32(mid)*int32(b2[j])>>allocSteps)
			if tmp >= th[j] || done != 0 {
				done = 1
				/* Don't allocate more than we can actually use */
				psum += int32(fixedmath.IMIN(tmp, cp[j]))
			} else {
				if tmp >= alloc_floor {
					psum += int32(alloc_floor)
				}
			}
		}
		if psum > total {
			hi = mid
		} else {
			lo = mid
		}
	}
	psum = 0
	done = 0
	for j := end - 1; j >= start; j-- {
		tmp := b1[j] + int(int32(lo)*int32(b2[j])>>allocSteps)
		if tmp < th[j] && done == 0 {
			if tmp >= alloc_floor {
				tmp = alloc_floor
			} else {
				tmp = 0
			}
		} else {
			done = 1
		}
		/* Don't allocate more than we can actually use */
		tmp = fixedmath.IMIN(tmp, cp[j])
		bo[j] = tmp
		psum += int32(tmp)
	}

	/* Decide which bands to skip, working backwards from the end. */
	for codedBands = end; ; codedBands-- {
		var band_width int
		var band_bits int
		var rem int
		j = codedBands - 1
		/* Never skip the first band, nor a band that has been boosted by
		   dynalloc. */
		if j <= skip_start {
			/* Give the bit we reserved to end skipping back. */
			total += int32(skip_rsv)
			break
		}
		/*Figure out how many left-over bits we would be adding to this band.
		  This can include bits we've stolen back from higher, skipped bands.*/
		left = total - psum
		percoeff = int32(fixedmath.Celt_udiv(uint32(left), uint32(int32(int(m.eBands[codedBands])-int(m.eBands[start])))))
		left -= int32(int(m.eBands[codedBands])-int(m.eBands[start])) * percoeff
		rem = fixedmath.IMAX(int(left)-(int(m.eBands[j])-int(m.eBands[start])), 0)
		band_width = int(m.eBands[codedBands]) - int(m.eBands[j])
		band_bits = int(int32(bits[j]) + percoeff*int32(band_width) + int32(rem))
		/*Only code a skip decision if we're above the threshold for this band.
		  Otherwise it is force-skipped.*/
		if band_bits >= fixedmath.IMAX(thresh[j], alloc_floor+(1<<bitRes)) {
			if encode != 0 {
				/*This if() block is the only part of the allocation function that
				  is not a mandatory part of the bitstream: any bands we choose to
				  skip here must be explicitly signaled.*/
				var depth_threshold int
				/*We choose a threshold with some hysteresis to keep bands from
				  fluctuating in and out.*/
				if codedBands > 17 {
					if j < prev {
						depth_threshold = 7
					} else {
						depth_threshold = 9
					}
				} else {
					depth_threshold = 0
				}
				if codedBands <= start+2 || (band_bits > (depth_threshold*band_width<<LM<<bitRes)>>4 && j <= signalBandwidth) {
					enc.EncBitLogp(1, 1)
					break
				}
				enc.EncBitLogp(0, 1)
			} else if dec.DecBitLogp(1) != 0 {
				break
			}
			/*We used a bit to skip this band.*/
			psum += 1 << bitRes
			band_bits -= 1 << bitRes
		}
		/*Reclaim the bits originally allocated to this band.*/
		psum -= int32(bits[j] + intensity_rsv)
		if intensity_rsv > 0 {
			intensity_rsv = log2FracTable[j-start]
		}
		psum += int32(intensity_rsv)
		if band_bits >= alloc_floor {
			/*If we have enough for a fine energy bit per channel, use it.*/
			psum += int32(alloc_floor)
			bits[j] = alloc_floor
		} else {
			/*Otherwise this band gets nothing at all.*/
			bits[j] = 0
		}
	}

	/* celt_assert(codedBands > start) */
	/* Code the intensity and dual stereo parameters. */
	if intensity_rsv > 0 {
		if encode != 0 {
			*intensity = fixedmath.IMIN(*intensity, codedBands)
			enc.EncUint(uint32(*intensity-start), uint32(codedBands+1-start))
		} else {
			*intensity = start + int(dec.DecUint(uint32(codedBands+1-start)))
		}
	} else {
		*intensity = 0
	}
	if *intensity <= start {
		total += int32(dual_stereo_rsv)
		dual_stereo_rsv = 0
	}
	if dual_stereo_rsv > 0 {
		if encode != 0 {
			enc.EncBitLogp(*dual_stereo, 1)
		} else {
			*dual_stereo = dec.DecBitLogp(1)
		}
	} else {
		*dual_stereo = 0
	}

	/* Allocate the remaining bits */
	left = total - psum
	percoeff = int32(fixedmath.Celt_udiv(uint32(left), uint32(int32(int(m.eBands[codedBands])-int(m.eBands[start])))))
	left -= int32(int(m.eBands[codedBands])-int(m.eBands[start])) * percoeff
	// eb views eBands over [0,codedBands]; j+1 <= codedBands < len(eb), so the
	// band-width reads fold check-free (same values, same order).
	eb := m.eBands[:codedBands+1]
	for j := start; j < codedBands; j++ {
		bo[j] += int(percoeff * int32(int(eb[j+1])-int(eb[j])))
	}
	for j := start; j < codedBands; j++ {
		tmp := int(fixedmath.MIN32(left, int32(int(eb[j+1])-int(eb[j]))))
		bo[j] += tmp
		left -= int32(tmp)
	}

	balance = 0
	for j = start; j < codedBands; j++ {
		var N0, N, den int
		var offset int
		var NClogN int
		var excess, bit int32

		/* celt_assert(bits[j] >= 0) */
		N0 = int(m.eBands[j+1]) - int(m.eBands[j])
		N = N0 << LM
		bit = int32(bits[j]) + balance

		if N > 1 {
			excess = fixedmath.MAX32(bit-int32(cap[j]), 0)
			bits[j] = int(bit - excess)

			/* Compensate for the extra DoF in stereo */
			den = C*N + b2i(C == 2 && N > 2 && *dual_stereo == 0 && j < *intensity)

			NClogN = den * (int(m.logN[j]) + logM)

			/* Offset for the number of fine bits by log2(N)/2 + FINE_OFFSET
			   compared to their "fair share" of total/N */
			offset = (NClogN >> 1) - den*fineOffset

			/* N=2 is the only point that doesn't match the curve */
			if N == 2 {
				offset += den << bitRes >> 2
			}

			/* Changing the offset for allocating the second and third
			   fine energy bit */
			if bits[j]+offset < den*2<<bitRes {
				offset += NClogN >> 2
			} else if bits[j]+offset < den*3<<bitRes {
				offset += NClogN >> 3
			}

			/* Divide with rounding */
			ebits[j] = fixedmath.IMAX(0, bits[j]+offset+(den<<(bitRes-1)))
			ebits[j] = int(fixedmath.Celt_udiv(uint32(ebits[j]), uint32(den))) >> bitRes

			/* Make sure not to bust */
			if C*ebits[j] > (bits[j] >> bitRes) {
				ebits[j] = bits[j] >> stereo >> bitRes
			}

			/* More than that is useless because that's about as far as PVQ can go */
			ebits[j] = fixedmath.IMIN(ebits[j], maxFineBits)

			/* If we rounded down or capped this band, make it a candidate for the
			   final fine energy pass */
			fine_priority[j] = b2i(ebits[j]*(den<<bitRes) >= bits[j]+offset)

			/* Remove the allocated fine bits; the rest are assigned to PVQ */
			bits[j] -= C * ebits[j] << bitRes
		} else {
			/* For N=1, all bits go to fine energy except for a single sign bit */
			excess = fixedmath.MAX32(0, bit-int32(C<<bitRes))
			bits[j] = int(bit - excess)
			ebits[j] = 0
			fine_priority[j] = 1
		}

		/* Fine energy can't take advantage of the re-balancing in
		   quant_all_bands(). Instead, do the re-balancing here.*/
		if excess > 0 {
			var extra_fine int
			var extra_bits int
			extra_fine = fixedmath.IMIN(int(excess>>(stereo+bitRes)), maxFineBits-ebits[j])
			ebits[j] += extra_fine
			extra_bits = extra_fine * C << bitRes
			fine_priority[j] = b2i(int32(extra_bits) >= excess-balance)
			excess -= int32(extra_bits)
		}
		balance = excess

		/* celt_assert(bits[j] >= 0) */
		/* celt_assert(ebits[j] >= 0) */
	}
	/* Save any remaining bits over the cap for the rebalancing in
	   quant_all_bands(). */
	*_balance = balance

	/* The skipped bands use all their bits for fine energy. */
	for ; j < end; j++ {
		ebits[j] = bits[j] >> stereo >> bitRes
		/* celt_assert(C*ebits[j]<<BITRES == bits[j]) */
		bits[j] = 0
		fine_priority[j] = b2i(ebits[j] < 1)
	}
	return codedBands
}

// cltComputeAllocation mirrors clt_compute_allocation (rate.c:535). It reserves
// the skip/intensity/dual-stereo bits, runs the 6-step bisection over the static
// allocVectors, sets up the interpolation endpoints, then calls
// interpBits2pulses. Fills pulses[]/ebits[]/fine_priority[], writes *balance and
// returns codedBands.
func cltComputeAllocation(m *celtMode, start, end int, offsets, cap []int, alloc_trim int,
	intensity, dual_stereo *int, total int32, balance *int32, pulses, ebits, fine_priority []int,
	C, LM int, enc *rangecoding.Encoder, dec *rangecoding.Decoder, encode, prev, signalBandwidth int,
	sc *scratch) int {
	var lo, hi, length, j int
	var codedBands int
	var skip_start int
	var skip_rsv int
	var intensity_rsv int
	var dual_stereo_rsv int

	total = fixedmath.MAX32(total, 0)
	length = m.nbEBands
	skip_start = start
	/* Reserve a bit to signal the end of manually skipped bands. */
	if total >= 1<<bitRes {
		skip_rsv = 1 << bitRes
	} else {
		skip_rsv = 0
	}
	total -= int32(skip_rsv)
	/* Reserve bits for the intensity and dual stereo parameters. */
	intensity_rsv = 0
	dual_stereo_rsv = 0
	if C == 2 {
		intensity_rsv = log2FracTable[end-start]
		if int32(intensity_rsv) > total {
			intensity_rsv = 0
		} else {
			total -= int32(intensity_rsv)
			if total >= 1<<bitRes {
				dual_stereo_rsv = 1 << bitRes
			} else {
				dual_stereo_rsv = 0
			}
			total -= int32(dual_stereo_rsv)
		}
	}
	bits1 := alloc(&sc.bits1, length)            // VARDECL(int, bits1)
	bits2 := alloc(&sc.bits2, length)            // VARDECL(int, bits2)
	thresh := alloc(&sc.thresh, length)          // VARDECL(int, thresh)
	trim_offset := alloc(&sc.trimOffset, length) // VARDECL(int, trim_offset)

	for j = start; j < end; j++ {
		/* Below this threshold, we're sure not to allocate any PVQ bits */
		thresh[j] = fixedmath.IMAX(C<<bitRes, (3*(int(m.eBands[j+1])-int(m.eBands[j]))<<LM<<bitRes)>>4)
		/* Tilt of the allocation curve */
		trim_offset[j] = C * (int(m.eBands[j+1]) - int(m.eBands[j])) * (alloc_trim - 5 - LM) * (end - j - 1) * (1 << (LM + bitRes)) >> 6
		/* Giving less resolution to single-coefficient bands because they get
		   more benefit from having one coarse value per coefficient*/
		if (int(m.eBands[j+1])-int(m.eBands[j]))<<LM == 1 {
			trim_offset[j] -= C << bitRes
		}
	}
	lo = 1
	hi = m.nbAllocVectors - 1
	for {
		done := 0
		psum := 0
		mid := (lo + hi) >> 1
		for j = end - 1; j >= start; j-- {
			var bitsj int
			N := int(m.eBands[j+1]) - int(m.eBands[j])
			bitsj = (C * N * int(m.allocVectors[mid*length+j]) << LM) >> 2
			if bitsj > 0 {
				bitsj = fixedmath.IMAX(0, bitsj+trim_offset[j])
			}
			bitsj += offsets[j]
			if bitsj >= thresh[j] || done != 0 {
				done = 1
				/* Don't allocate more than we can actually use */
				psum += fixedmath.IMIN(bitsj, cap[j])
			} else {
				if bitsj >= C<<bitRes {
					psum += C << bitRes
				}
			}
		}
		if int32(psum) > total {
			hi = mid - 1
		} else {
			lo = mid + 1
		}
		if !(lo <= hi) {
			break
		}
	}
	hi = lo
	lo--
	for j = start; j < end; j++ {
		var bits1j, bits2j int
		N := int(m.eBands[j+1]) - int(m.eBands[j])
		bits1j = (C * N * int(m.allocVectors[lo*length+j]) << LM) >> 2
		if hi >= m.nbAllocVectors {
			bits2j = cap[j]
		} else {
			bits2j = (C * N * int(m.allocVectors[hi*length+j]) << LM) >> 2
		}
		if bits1j > 0 {
			bits1j = fixedmath.IMAX(0, bits1j+trim_offset[j])
		}
		if bits2j > 0 {
			bits2j = fixedmath.IMAX(0, bits2j+trim_offset[j])
		}
		if lo > 0 {
			bits1j += offsets[j]
		}
		bits2j += offsets[j]
		if offsets[j] > 0 {
			skip_start = j
		}
		bits2j = fixedmath.IMAX(0, bits2j-bits1j)
		bits1[j] = bits1j
		bits2[j] = bits2j
	}
	codedBands = interpBits2pulses(m, start, end, skip_start, bits1, bits2, thresh, cap,
		total, balance, skip_rsv, intensity, intensity_rsv, dual_stereo, dual_stereo_rsv,
		pulses, ebits, fine_priority, C, LM, enc, dec, encode, prev, signalBandwidth)
	return codedBands
}

// ComputeAllocation is the exported seam over cltComputeAllocation bound to the
// frozen mode48000_960 (mirroring the quant_bands.go adapter shape). encode!=0
// drives the encoder path (writes skip/intensity/dual flags to enc); encode==0
// drives the decoder path (reads them from dec). The decoder calls this with
// encode=0, prev=0, signalBandwidth=0 (celt/celt_decoder.c:1451).
func ComputeAllocation(start, end int, offsets, cap []int, allocTrim int,
	intensity, dualStereo *int, total int, pulses, ebits, finePriority []int,
	C, LM int, enc *rangecoding.Encoder, dec *rangecoding.Decoder, encode, prev, signalBandwidth int) (codedBands, balance int) {
	var bal int32
	var sc scratch
	codedBands = cltComputeAllocation(&mode48000_960, start, end, offsets, cap, allocTrim,
		intensity, dualStereo, int32(total), &bal, pulses, ebits, finePriority,
		C, LM, enc, dec, encode, prev, signalBandwidth, &sc)
	return codedBands, int(bal)
}

// InitCaps is the exported seam over initCaps bound to mode48000_960; it fills
// cap[i] for i in [0, nbEBands), as the decoder does before ComputeAllocation.
func InitCaps(cap []int, LM, C int) { initCaps(&mode48000_960, cap, LM, C) }

// GetPulses exposes get_pulses for the differential test.
func GetPulses(i int) int { return getPulses(i) }

// Bits2Pulses exposes bits2pulses (bound to mode48000_960) for the test.
func Bits2Pulses(band, LM, bits int) int { return bits2pulses(&mode48000_960, band, LM, bits) }

// Pulses2Bits exposes pulses2bits (bound to mode48000_960) for the test.
func Pulses2Bits(band, LM, pulses int) int { return pulses2bits(&mode48000_960, band, LM, pulses) }
