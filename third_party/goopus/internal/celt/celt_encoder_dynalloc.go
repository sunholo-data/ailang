// CELT encoder dynamic-allocation analysis, phase 4 (CP8b, unit DYNALLOC). This
// file ports celt/celt_encoder.c median_of_5 (:990), median_of_3 (:1029) and
// dynalloc_analysis (:1049): the per-band boost / importance / spread-weight
// analysis that celt_encode_with_ec runs before the allocation and band-shape
// coding stages.
//
// Frozen config (celt/arch.h): FIXED_POINT + DISABLE_FLOAT_API + OPUS_FAST_INT64,
// no ENABLE_QEXT / RES24 / CUSTOM_MODES. Consequences, all load-bearing here:
//
//   - celt_glog is opus_int32 (32-bit, NOT 16-bit), DB_SHIFT 24, so GCONST(x) is
//     QCONST32(x, 24) and MAXG/MING are MAX32/MIN32.
//   - ARG_QEXT(arg) expands to NOTHING (celt.h:49), so dynalloc_analysis has NO
//     qext_scale parameter, and QEXT_SCALE(x) is the identity (celt.h:270).
//   - #ifdef DISABLE_FLOAT_API at :1223 selects `(void)analysis;`, so the
//     AnalysisInfo* argument is INERT and the analysis->leak_boost LEAK_BANDS loop
//     (:1226-1230) is dead code. The Go signature therefore has no analysis param.
//   - The three live #ifdef FIXED_POINT branches ported here are :1106-1111
//     (spread_weight shift), :1186-1190 (importance) and :1207-1211 (freq_bin);
//     their float twins are gated out.
//
// Layer C: transliterated verbatim, including the C's quirks (e.g. `last` is
// declared OUTSIDE the per-channel do/while at :1123 and is deliberately NOT
// reset between channels).

package celt

import (
	"math"

	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
)

// GCONST constants for dynalloc_analysis. GCONST(x) is GCONST2(x, DB_SHIFT) =
// (celt_glog)(.5+(x)*(((celt_glog)1)<<24)) (fixed_generic.h:98,101). The C
// literals are FLOATS, so the multiply is evaluated in float32 before the .5 is
// added in double; for the two non-dyadic constants (31.9f and .0062f) that
// rounding is observable, so they go through qconst32Float (float32 semantics,
// bands_math.go:41) rather than fixedmath.QCONST32 (float64). Checked against C:
// GCONST(31.9f)=535193184 (float64 would give 535193190) and GCONST(.0062f)=104019.
// gconst1 (celt_encoder.go:431), gconst1_5 / gconst2 (celt_decoder.go:30-31) and
// gconst3 (quant_bands.go:37) already exist in this package and are reused.
var (
	gconst31_9       = qconst32Float(31.9, dbShift)                // GCONST(31.9f)
	gconst0_0625     = fixedmath.QCONST32(0.0625, dbShift)         // GCONST(0.0625f)
	gconst0_5        = fixedmath.QCONST32(0.5, dbShift)            // GCONST(.5f)
	gconst0_0062     = qconst32Float(0.0062, dbShift)              // GCONST(.0062f)
	gconst4          = fixedmath.QCONST32(4, dbShift)              // GCONST(4.f) == GCONST(4)
	gconst5          = fixedmath.QCONST32(5, dbShift)              // GCONST(5.f)
	gconst12         = fixedmath.QCONST32(12, dbShift)             // GCONST(12.f)
	qconst0_98Q29    = qconst32Float(0.98, 29)                     // QCONST32(.98f, 29)
	qconst120DivPiQ9 = fixedmath.QCONST16(120/float64(math.Pi), 9) // QCONST16(120/M_PI, 9)

	// The C source writes these as *float* literals, so the constant is rounded in
	// float32 before the shift, which does not agree with a float64 evaluation:
	// QCONST32(.99f,29) is 531502208 but the float64 form gives 531502203, and
	// QCONST32(1.999999f,29) is 1073741312 against 1073741287. The difference is a
	// handful of ULPs, but these constants are compared against toneishness and
	// lpc[0] directly, so a value landing inside the gap takes a different branch
	// than the C and the whole packet diverges from there on.
	//
	// Note QCONST32(3.999999, 29) at celt_encoder.c:1387 is deliberately NOT here:
	// that literal has no "f" suffix, so it is a double and fixedmath.QCONST32 is
	// the correct helper for it. The suffix decides, per call site.
	qconst0_99Q29     = qconst32Float(0.99, 29)     // QCONST32(.99f, 29)      celt_encoder.c:1440
	qconst1_999999Q29 = qconst32Float(1.999999, 29) // QCONST32(1.999999f, 29) celt_encoder.c:1354
)

// medianOf5 is median_of_5 (celt_encoder.c:990): the median of x[0..5), computed
// with the C's exact comparison ladder (MSWAP + MING).
func medianOf5(x []int32) int32 {
	var t0, t1, t2, t3, t4 int32
	t2 = x[2]
	if x[0] > x[1] {
		t0 = x[1]
		t1 = x[0]
	} else {
		t0 = x[0]
		t1 = x[1]
	}
	if x[3] > x[4] {
		t3 = x[4]
		t4 = x[3]
	} else {
		t3 = x[3]
		t4 = x[4]
	}
	if t0 > t3 {
		// MSWAP(t0,t3) then MSWAP(t1,t4) (:1012-1013). The C's MSWAP writes t0 even
		// though nothing reads t0 again; the assignment is kept so the swap ladder
		// stays a 1:1 image of the C.
		//nolint:ineffassign,staticcheck,wastedassign // verbatim MSWAP(t0,t3): the C writes t0 here and never reads it again; the dead store is kept so the ladder stays a 1:1 image of :1012.
		t0, t3 = t3, t0
		t1, t4 = t4, t1
	}
	if t2 > t1 {
		if t1 < t3 {
			return fixedmath.MIN32(t2, t3)
		}
		return fixedmath.MIN32(t4, t1)
	}
	if t2 < t3 {
		return fixedmath.MIN32(t1, t3)
	}
	return fixedmath.MIN32(t2, t4)
}

// medianOf3 is median_of_3 (celt_encoder.c:1029): the median of x[0..3).
func medianOf3(x []int32) int32 {
	var t0, t1, t2 int32
	if x[0] > x[1] {
		t0 = x[1]
		t1 = x[0]
	} else {
		t0 = x[0]
		t1 = x[1]
	}
	t2 = x[2]
	if t1 < t2 {
		return t1
	} else if t0 < t2 {
		return t2
	}
	return t0
}

// dynallocAnalysis is dynalloc_analysis (celt_encoder.c:1049) for the frozen
// FIXED_POINT / DISABLE_FLOAT_API config (no qext_scale parameter, no
// AnalysisInfo). It returns maxDepth and writes *totBoost, offsets, importance
// and spreadWeight.
//
// Buffer contracts, straight from the C and asymmetric on purpose:
//   - offsets is OPUS_CLEAR'd over [0,nbEBands) at :1066, so it is fully defined.
//   - spreadWeight is written over [0,end) unconditionally (:1112).
//   - importance is written ONLY over [start,end) (:1187 or :1268). Outside that
//     window the C leaves the caller's ALLOC untouched (i.e. stack garbage), so
//     nothing may be asserted about it.
//
// bandLogE / bandLogE2 / oldBandE are C*nbEBands celt_glog; surroundDynalloc is
// nbEBands celt_glog. logN and eBands come from the mode.
func dynallocAnalysis(bandLogE, bandLogE2, oldBandE []int32,
	nbEBands, start, end, C int, offsets []int, lsbDepth int, logN []int16,
	isTransient, vbr, constrainedVbr int, eBands []int16, LM, effectiveBytes int,
	totBoost_ *int32, lfe int, surroundDynalloc []int32,
	importance, spreadWeight []int, toneFreq int16, toneishness int32, sc *scratch) int32 {
	var i, c int
	totBoost := int32(0)
	var maxDepth int32

	follower := alloc(&sc.follower, C*nbEBands)     // VARDECL(celt_glog, follower)
	noiseFloor := alloc(&sc.noiseFloor, C*nbEBands) // VARDECL(celt_glog, noise_floor)
	bandLogE3 := alloc(&sc.bandLogE3, nbEBands)     // VARDECL(celt_glog, bandLogE3)
	// OPUS_CLEAR(offsets, nbEBands) (:1066).
	for i = 0; i < nbEBands; i++ {
		offsets[i] = 0
	}
	// Dynamic allocation code.
	maxDepth = -gconst31_9
	for i = 0; i < end; i++ {
		// Noise floor must take into account eMeans, the depth, the width of the
		// bands and the preemphasis filter (approx. square of bark band ID).
		noiseFloor[i] = gconst0_0625*int32(logN[i]) +
			gconst0_5 + fixedmath.SHL32(int32(9-lsbDepth), dbShift) -
			fixedmath.SHL32(int32(eMeans[i]), dbShift-4) +
			gconst0_0062*int32(i+5)*int32(i+5)
	}
	c = 0
	for {
		for i = 0; i < end; i++ {
			maxDepth = fixedmath.MAX32(maxDepth, bandLogE[c*nbEBands+i]-noiseFloor[i])
		}
		c++
		if c >= C {
			break
		}
	}
	{
		// Compute a really simple masking model to avoid taking into account
		// completely masked bands when computing the spreading decision.
		mask := alloc(&sc.mask, nbEBands) // VARDECL(celt_glog, mask)
		sig := alloc(&sc.sig, nbEBands)   // VARDECL(celt_glog, sig)
		for i = 0; i < end; i++ {
			mask[i] = bandLogE[i] - noiseFloor[i]
		}
		if C == 2 {
			for i = 0; i < end; i++ {
				mask[i] = fixedmath.MAX32(mask[i], bandLogE[nbEBands+i]-noiseFloor[i])
			}
		}
		copy(sig[:end], mask[:end])
		for i = 1; i < end; i++ {
			mask[i] = fixedmath.MAX32(mask[i], mask[i-1]-gconst2)
		}
		for i = end - 2; i >= 0; i-- {
			mask[i] = fixedmath.MAX32(mask[i], mask[i+1]-gconst3)
		}
		for i = 0; i < end; i++ {
			// Compute SMR: Mask is never more than 72 dB below the peak and never
			// below the noise floor.
			smr := sig[i] - fixedmath.MAX32(fixedmath.MAX32(0, maxDepth-gconst12), mask[i])
			// Clamp SMR to make sure we're not shifting by something negative or
			// too large. (FIXED_POINT branch, :1106-1108.)
			shift := int(-fixedmath.PSHR32(fixedmath.MAX32(-gconst5, fixedmath.MIN32(0, smr)), dbShift))
			spreadWeight[i] = 32 >> shift
		}
	}
	// Make sure that dynamic allocation can't make us bust the budget. We enable
	// the feature starting at 24 kb/s for 20-ms frames and 96 kb/s for 2.5 ms
	// frames.
	if effectiveBytes >= (30+5*LM) && lfe == 0 {
		// NOTE: `last` is declared outside the per-channel loop in the C (:1123)
		// and is NOT reset for the second channel. Reproduced verbatim.
		last := 0
		c = 0
		for {
			var offset int32
			var tmp int32
			copy(bandLogE3[:end], bandLogE2[c*nbEBands:c*nbEBands+end])
			if LM == 0 {
				// For 2.5 ms frames, the first 8 bands have just one bin, so the
				// energy is highly unreliable (high variance). For that reason, we
				// take the max with the previous energy so that at least 2 bins are
				// getting used.
				for i = 0; i < fixedmath.IMIN(8, end); i++ {
					bandLogE3[i] = fixedmath.MAX32(bandLogE2[c*nbEBands+i], oldBandE[c*nbEBands+i])
				}
			}
			f := follower[c*nbEBands:]
			f[0] = bandLogE3[0]
			for i = 1; i < end; i++ {
				// The last band to be at least 3 dB higher than the previous one is
				// the last we'll consider. Otherwise, we run into problems on
				// bandlimited signals.
				if bandLogE3[i] > bandLogE3[i-1]+gconst0_5 {
					last = i
				}
				f[i] = fixedmath.MIN32(f[i-1]+gconst1_5, bandLogE3[i])
			}
			for i = last - 1; i >= 0; i-- {
				f[i] = fixedmath.MIN32(f[i], fixedmath.MIN32(f[i+1]+gconst2, bandLogE3[i]))
			}

			// Combine with a median filter to avoid dynalloc triggering
			// unnecessarily. The "offset" value controls how conservative we are:
			// a higher offset reduces the impact of the median filter and makes
			// dynalloc use more bits.
			offset = gconst1
			for i = 2; i < end-2; i++ {
				f[i] = fixedmath.MAX32(f[i], medianOf5(bandLogE3[i-2:])-offset)
			}
			tmp = medianOf3(bandLogE3[0:]) - offset
			f[0] = fixedmath.MAX32(f[0], tmp)
			f[1] = fixedmath.MAX32(f[1], tmp)
			tmp = medianOf3(bandLogE3[end-3:]) - offset
			f[end-2] = fixedmath.MAX32(f[end-2], tmp)
			f[end-1] = fixedmath.MAX32(f[end-1], tmp)

			for i = 0; i < end; i++ {
				f[i] = fixedmath.MAX32(f[i], noiseFloor[i])
			}
			c++
			if c >= C {
				break
			}
		}
		if C == 2 {
			for i = start; i < end; i++ {
				// Consider 24 dB "cross-talk".
				follower[nbEBands+i] = fixedmath.MAX32(follower[nbEBands+i], follower[i]-gconst4)
				follower[i] = fixedmath.MAX32(follower[i], follower[nbEBands+i]-gconst4)
				follower[i] = fixedmath.HALF32(fixedmath.MAX32(0, bandLogE[i]-follower[i]) +
					fixedmath.MAX32(0, bandLogE[nbEBands+i]-follower[nbEBands+i]))
			}
		} else {
			for i = start; i < end; i++ {
				follower[i] = fixedmath.MAX32(0, bandLogE[i]-follower[i])
			}
		}
		for i = start; i < end; i++ {
			follower[i] = fixedmath.MAX32(follower[i], surroundDynalloc[i])
		}
		for i = start; i < end; i++ {
			// FIXED_POINT branch, :1187.
			importance[i] = int(fixedmath.PSHR32(13*celtExp2Db(fixedmath.MIN32(follower[i], gconst4)), 16))
		}
		// For non-transient CBR/CVBR frames, halve the dynalloc contribution.
		if (vbr == 0 || constrainedVbr != 0) && isTransient == 0 {
			for i = start; i < end; i++ {
				follower[i] = fixedmath.HALF32(follower[i])
			}
		}
		for i = start; i < end; i++ {
			if i < 8 {
				follower[i] *= 2
			}
			if i >= 12 {
				follower[i] = fixedmath.HALF32(follower[i])
			}
		}
		// Compensate for Opus' under-allocation on tones.
		if toneishness > qconst0_98Q29 {
			// FIXED_POINT branch, :1208. QEXT_SCALE is the identity here.
			freqBin := int(fixedmath.PSHR32(int32(toneFreq)*int32(qconst120DivPiQ9), 13+9))
			for i = start; i < end; i++ {
				if freqBin >= int(eBands[i]) && freqBin <= int(eBands[i+1]) {
					follower[i] += gconst2
				}
				if freqBin >= int(eBands[i])-1 && freqBin <= int(eBands[i+1])+1 {
					follower[i] += gconst1
				}
				if freqBin >= int(eBands[i])-2 && freqBin <= int(eBands[i+1])+2 {
					follower[i] += gconst1
				}
				if freqBin >= int(eBands[i])-3 && freqBin <= int(eBands[i+1])+3 {
					follower[i] += gconst0_5
				}
			}
			if freqBin >= int(eBands[end]) {
				follower[end-1] += gconst2
				follower[end-2] += gconst1
			}
		}
		// #ifdef DISABLE_FLOAT_API (:1223): (void)analysis. The analysis->leak_boost
		// loop is dead code in this config and is deliberately not ported.
		for i = start; i < end; i++ {
			var width int
			var boost int
			var boostBits int

			follower[i] = fixedmath.MIN32(follower[i], gconst4)

			follower[i] = fixedmath.SHR32(follower[i], 8)
			width = C * int(eBands[i+1]-eBands[i]) << LM
			if width < 6 {
				boost = int(fixedmath.SHR32(follower[i], dbShift-8))
				boostBits = boost * width << bitRes
			} else if width > 48 {
				boost = int(fixedmath.SHR32(follower[i]*8, dbShift-8))
				boostBits = (boost * width << bitRes) / 8
			} else {
				boost = int(fixedmath.SHR32(follower[i]*int32(width)/6, dbShift-8))
				boostBits = boost * 6 << bitRes
			}
			// For CBR and non-transient CVBR frames, limit dynalloc to 2/3 of the
			// bits.
			if (vbr == 0 || (constrainedVbr != 0 && isTransient == 0)) &&
				int((totBoost+int32(boostBits))>>bitRes>>3) > 2*effectiveBytes/3 {
				capBits := int32((2 * effectiveBytes / 3) << bitRes << 3)
				offsets[i] = int(capBits - totBoost)
				totBoost = capBits
				break
			} else {
				offsets[i] = boost
				totBoost += int32(boostBits)
			}
		}
	} else {
		for i = start; i < end; i++ {
			importance[i] = 13
		}
	}
	*totBoost_ = totBoost
	return maxDepth
}

// --- Differential-test entry points (used by internal/reftest/oracle) --------

// DynallocResult carries the outputs of DynallocAnalysis. Importance is only
// DEFINED over [start,end) and SpreadWeight only over [0,end): the C leaves the
// rest of those buffers untouched, so nothing outside those windows may be
// compared against the oracle.
type DynallocResult struct {
	MaxDepth     int32
	TotBoost     int32
	Offsets      []int
	Importance   []int
	SpreadWeight []int
}

// DynallocAnalysis runs dynalloc_analysis (celt_encoder.c:1049) against the frozen
// 48 kHz / 960 CELT mode (which supplies logN and eBands). bandLogE / bandLogE2 /
// oldBandE are C*nbEBands celt_glog and surroundDynalloc is nbEBands celt_glog.
func DynallocAnalysis(bandLogE, bandLogE2, oldBandE []int32, nbEBands, start, end, C,
	lsbDepth, isTransient, vbr, constrainedVbr, LM, effectiveBytes, lfe int,
	surroundDynalloc []int32, toneFreq int16, toneishness int32) DynallocResult {
	m := &mode48000_960
	res := DynallocResult{
		Offsets:      make([]int, nbEBands),
		Importance:   make([]int, nbEBands),
		SpreadWeight: make([]int, nbEBands),
	}
	var totBoost int32
	var sc scratch
	res.MaxDepth = dynallocAnalysis(bandLogE, bandLogE2, oldBandE, nbEBands, start, end, C,
		res.Offsets, lsbDepth, m.logN, isTransient, vbr, constrainedVbr, m.eBands, LM,
		effectiveBytes, &totBoost, lfe, surroundDynalloc, res.Importance, res.SpreadWeight,
		toneFreq, toneishness, &sc)
	res.TotBoost = totBoost
	return res
}

// FloatLiteralConsts exposes, for the differential oracle, every QCONST/GCONST whose
// value depends on whether the C source writes the literal as a float (rounded in
// float32 before the shift) or as a double. The oracle asserts each against the
// compile-time C constant, so the per-site choice between qconst32Float and
// fixedmath.QCONST32 is pinned by the reference rather than by a comment. Note the
// deliberate mix: QCONST32(3.999999, 29) at celt_encoder.c:1387 carries no "f" suffix
// and so must NOT use the float32 helper.
func FloatLiteralConsts() map[string]int32 {
	return map[string]int32{
		"QCONST32(.98f,29)":       qconst0_98Q29,
		"QCONST32(.99f,29)":       qconst0_99Q29,
		"QCONST32(1.999999f,29)":  qconst1_999999Q29,
		"QCONST32(3.999999,29)":   fixedmath.QCONST32(3.999999, 29),
		"QCONST32(.70710678f,31)": sqrt2Inv31,
		"GCONST(31.9f)":           gconst31_9,
		"GCONST(.0062f)":          gconst0_0062,
	}
}
