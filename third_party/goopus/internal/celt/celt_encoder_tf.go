package celt

import (
	"github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"
	"github.com/sunholo-data/ailang/third_party/goopus/internal/rangecoding"
)

// This file ports the CELT encoder's time-frequency resolution stage from
// celt/celt_encoder.c: l1_metric (:650), tf_analysis (:663) and tf_encode
// (:824). Frozen build config: FIXED_POINT + DISABLE_FLOAT_API +
// OPUS_FAST_INT64, no ENABLE_QEXT / RES24 / CUSTOM_MODES / FUZZING. The
// `#ifdef FUZZING` block at celt_encoder.c:815-820 is therefore dead and is not
// ported.

// l1Metric is l1_metric (celt_encoder.c:650): the (biased) L1 norm of a band,
// used to score a candidate time-frequency split.
//
// Type note: C's ABS16 (arch.h:227) and EXTEND32 (arch.h:310) are type-agnostic
// macros, and here they are applied to SHR32(tmp[i], NORM_SHIFT-14), which is an
// opus_val32 because tmp is celt_norm (int32 in v1.6). So NO 16-bit narrowing
// happens on that path and the Go port must use ABS32, not ABS16, whose Go
// signature takes an int16 and would silently truncate the value.
//
// MAC16_32_Q15 (fixed_generic.h:183) has no OPUS_FAST_INT64 variant: it is
// always the two-part MULT16_16 decomposition, whose (opus_val16) casts are what
// narrows LM*bias to 16 bits. int16(int32(LM)*int32(bias)) reproduces that cast.
func l1Metric(tmp []int32, N, LM int, bias int16) int32 {
	var L1 int32
	L1 = 0
	for i := 0; i < N; i++ {
		// EXTEND32 is the identity on an operand that is already opus_val32.
		L1 += fixedmath.ABS32(fixedmath.SHR32(tmp[i], normShift-14))
	}
	// When in doubt, prefer good freq resolution.
	L1 = fixedmath.MAC16_32_Q15(L1, int16(int32(LM)*int32(bias)), L1)
	return L1
}

// tfAnalysis is tf_analysis (celt_encoder.c:663): it picks, per band, the
// time-frequency resolution that minimizes a biased L1 metric, then runs a
// Viterbi search (over both tf_select hypotheses) to trade the per-band decisions
// off against a switching cost `lambda`. It writes tf_res[0..len) and returns
// tf_select. It is stateless and does not mutate X (each band is copied into a
// scratch buffer first).
//
// KNOWN C DIVERGENCE (deliberately reproduced, do not "fix"): the cost loops at
// celt_encoder.c:751 and :776 read importance[i] for i in [0,len), but
// dynalloc_analysis (celt_encoder.c:1186) only ever writes importance[start..end).
// When start>0 the C therefore reads uninitialized stack. This is unreachable in
// the frozen config: tf_analysis is only called under the enable_tf_analysis gate
// (celt_encoder.c:2210), which requires !hybrid, and start>0 only happens in
// hybrid mode. The port keeps the C indexing verbatim; the differential test
// consequently only drives start=0 (i.e. importance defined over [0,len)).
//
// The `#ifdef FUZZING` block at :815-820 is dead in the frozen config.
func tfAnalysis(m *celtMode, length, isTransient int, tfRes []int, lambda int,
	X []int32, N0, LM int, tfEstimate int16, tfChan int, importance []int, sc *scratch) int {
	var cost0 int
	var cost1 int
	var selcost [2]int
	// C spells this `int tf_select=0;` (:677), but that initialiser is dead: the
	// value is never read before the redundant `tf_select = 0;` reset at :750,
	// which is kept verbatim below. Go's zero value carries the same meaning.
	var tfSelect int

	// MAX16 (arch.h:101) is type-agnostic and QCONST16(.5,14)-tf_estimate is an
	// int (both operands promote), so this is 32-bit arithmetic; only the
	// MULT16_16_Q14 operand cast narrows to 16 bits.
	bias := int16(fixedmath.MULT16_16_Q14(
		fixedmath.QCONST16(.04, 15),
		fixedmath.EXTRACT16(fixedmath.MAX16(
			-int32(fixedmath.QCONST16(.25, 14)),
			int32(fixedmath.QCONST16(.5, 14))-int32(tfEstimate)))))

	metric := alloc(&sc.tfMetric, length)                                          // VARDECL(int, metric)
	tmp := alloc(&sc.tfTmp, (int(m.eBands[length])-int(m.eBands[length-1]))<<LM)   // VARDECL(celt_norm, tmp)
	tmp1 := alloc(&sc.tfTmp1, (int(m.eBands[length])-int(m.eBands[length-1]))<<LM) // VARDECL(celt_norm, tmp_1)
	path0 := alloc(&sc.tfPath0, length)                                            // VARDECL(int, path0)
	path1 := alloc(&sc.tfPath1, length)                                            // VARDECL(int, path1)

	for i := 0; i < length; i++ {
		var k, N int
		var narrow int
		var L1, bestL1 int32
		bestLevel := 0
		N = (int(m.eBands[i+1]) - int(m.eBands[i])) << LM
		// Band is too narrow to be split down to LM=-1.
		narrow = 0
		if (int(m.eBands[i+1]) - int(m.eBands[i])) == 1 {
			narrow = 1
		}
		copy(tmp[:N], X[tfChan*N0+(int(m.eBands[i])<<LM):])
		// (The C's commented-out "just add the right channel if we're in stereo"
		// block at :700-702 is not code.)
		if isTransient != 0 {
			L1 = l1Metric(tmp, N, LM, bias)
		} else {
			L1 = l1Metric(tmp, N, 0, bias)
		}
		bestL1 = L1
		// Check the -1 case for transients.
		if isTransient != 0 && narrow == 0 {
			copy(tmp1[:N], tmp[:N])
			haar1(tmp1, N>>LM, 1<<LM)
			L1 = l1Metric(tmp1, N, LM+1, bias)
			if L1 < bestL1 {
				bestL1 = L1
				bestLevel = -1
			}
		}
		// C: for (k=0;k<LM+!(isTransient||narrow);k++)
		kEnd := LM
		if isTransient == 0 && narrow == 0 {
			kEnd = LM + 1
		}
		for k = 0; k < kEnd; k++ {
			var B int

			if isTransient != 0 {
				B = LM - k - 1
			} else {
				B = k + 1
			}

			haar1(tmp, N>>k, 1<<k)

			L1 = l1Metric(tmp, N, B, bias)

			if L1 < bestL1 {
				bestL1 = L1
				bestLevel = k + 1
			}
		}
		// metric is in Q1 to be able to select the mid-point (-0.5) for narrower
		// bands.
		if isTransient != 0 {
			metric[i] = 2 * bestLevel
		} else {
			metric[i] = -2 * bestLevel
		}
		// For bands that can't be split to -1, set the metric to the half-way
		// point to avoid biasing the decision.
		if narrow != 0 && (metric[i] == 0 || metric[i] == -2*LM) {
			metric[i] -= 1
		}
	}
	// Search for the optimal tf resolution, including tf_select.
	tfSelect = 0
	for sel := 0; sel < 2; sel++ {
		cost0 = importance[0] * iabs(int32(metric[0]-2*int(tfSelectTable[LM][4*isTransient+2*sel+0])))
		cost1 = importance[0] * iabs(int32(metric[0]-2*int(tfSelectTable[LM][4*isTransient+2*sel+1])))
		if isTransient == 0 {
			cost1 += lambda
		}
		for i := 1; i < length; i++ {
			var curr0, curr1 int
			curr0 = fixedmath.IMIN(cost0, cost1+lambda)
			curr1 = fixedmath.IMIN(cost0+lambda, cost1)
			cost0 = curr0 + importance[i]*iabs(int32(metric[i]-2*int(tfSelectTable[LM][4*isTransient+2*sel+0])))
			cost1 = curr1 + importance[i]*iabs(int32(metric[i]-2*int(tfSelectTable[LM][4*isTransient+2*sel+1])))
		}
		cost0 = fixedmath.IMIN(cost0, cost1)
		selcost[sel] = cost0
	}
	// For now, we're conservative and only allow tf_select=1 for transients.
	// If tests confirm it's useful for non-transients, we could allow it.
	if selcost[1] < selcost[0] && isTransient != 0 {
		tfSelect = 1
	}
	cost0 = importance[0] * iabs(int32(metric[0]-2*int(tfSelectTable[LM][4*isTransient+2*tfSelect+0])))
	cost1 = importance[0] * iabs(int32(metric[0]-2*int(tfSelectTable[LM][4*isTransient+2*tfSelect+1])))
	if isTransient == 0 {
		cost1 += lambda
	}
	// Viterbi forward pass.
	for i := 1; i < length; i++ {
		var curr0, curr1 int
		var from0, from1 int

		from0 = cost0
		from1 = cost1 + lambda
		if from0 < from1 {
			curr0 = from0
			path0[i] = 0
		} else {
			curr0 = from1
			path0[i] = 1
		}

		from0 = cost0 + lambda
		from1 = cost1
		if from0 < from1 {
			curr1 = from0
			path1[i] = 0
		} else {
			curr1 = from1
			path1[i] = 1
		}
		cost0 = curr0 + importance[i]*iabs(int32(metric[i]-2*int(tfSelectTable[LM][4*isTransient+2*tfSelect+0])))
		cost1 = curr1 + importance[i]*iabs(int32(metric[i]-2*int(tfSelectTable[LM][4*isTransient+2*tfSelect+1])))
	}
	if cost0 < cost1 {
		tfRes[length-1] = 0
	} else {
		tfRes[length-1] = 1
	}
	// Viterbi backward pass to check the decisions.
	for i := length - 2; i >= 0; i-- {
		if tfRes[i+1] == 1 {
			tfRes[i] = path1[i+1]
		} else {
			tfRes[i] = path0[i+1]
		}
	}
	return tfSelect
}

// tfEncode is tf_encode (celt_encoder.c:824): it writes the per-band tf
// decisions and the tf_select flag to the range coder.
//
// IT MUTATES tfRes[start..end) IN PLACE, twice over:
//  1. :849 - any band the bit budget cannot afford is FORCED to `curr` (the last
//     successfully coded value) rather than its requested value; and
//  2. :859-860 - EVERY entry is then remapped through
//     tf_select_table[LM][4*isTransient + 2*tf_select + tf_res[i]], turning the
//     0/1 decisions into the signed per-band resolution shifts (which may be
//     negative) that quant_all_bands consumes downstream.
//
// The budget arithmetic is UNSIGNED in C (opus_uint32 budget/tell), so it is
// reproduced in uint32 here: an int comparison could differ from C's if the
// expression ever wrapped or saturated.
func tfEncode(start, end, isTransient int, tfRes []int, LM, tfSelect int, enc *rangecoding.Encoder) {
	var curr, i int
	var tfSelectRsv int
	var tfChanged int
	var logp int
	var budget uint32
	var tell uint32
	budget = enc.Storage() * 8
	tell = uint32(enc.Tell())
	if isTransient != 0 {
		logp = 2
	} else {
		logp = 4
	}
	// Reserve space to code the tf_select decision.
	tfSelectRsv = 0
	if LM > 0 && tell+uint32(logp)+1 <= budget {
		tfSelectRsv = 1
	}
	budget -= uint32(tfSelectRsv)
	curr = 0
	tfChanged = 0
	for i = start; i < end; i++ {
		if tell+uint32(logp) <= budget {
			enc.EncBitLogp(tfRes[i]^curr, uint32(logp))
			tell = uint32(enc.Tell())
			curr = tfRes[i]
			tfChanged |= curr
		} else {
			tfRes[i] = curr
		}
		if isTransient != 0 {
			logp = 4
		} else {
			logp = 5
		}
	}
	// Only code tf_select if it would actually make a difference.
	if tfSelectRsv != 0 &&
		tfSelectTable[LM][4*isTransient+0+tfChanged] !=
			tfSelectTable[LM][4*isTransient+2+tfChanged] {
		enc.EncBitLogp(tfSelect, 1)
	} else {
		tfSelect = 0
	}
	for i = start; i < end; i++ {
		tfRes[i] = int(tfSelectTable[LM][4*isTransient+2*tfSelect+tfRes[i]])
	}
}

// --- Differential-test entry points (used by internal/reftest/oracle) --------

// TfAnalysis runs tf_analysis (celt_encoder.c:663) against the frozen
// mode48000_960. X is C*N0 celt_norm (read-only); importance is read over
// [0,len). It returns tf_select and the tf_res[0..len) decisions.
func TfAnalysis(length, isTransient, lambda int, X []int32, N0, LM int,
	tfEstimate int16, tfChan int, importance []int) (tfSelect int, tfRes []int) {
	m := &mode48000_960
	tfRes = make([]int, length)
	var sc scratch
	tfSelect = tfAnalysis(m, length, isTransient, tfRes, lambda, X, N0, LM, tfEstimate, tfChan, importance, &sc)
	return tfSelect, tfRes
}

// TfEncodeResult is one TfEncode outcome. TfRes is the array AFTER tf_encode's
// in-place mutation (budget-forced entries at celt_encoder.c:849 plus the
// tf_select_table remap at :859-860). Tell / Rng / Val are captured after
// tf_encode but BEFORE EncDone, matching the C oracle wrapper; Buf holds the
// finalized bitstream (after EncDone).
type TfEncodeResult struct {
	TfRes      []int
	TellBefore int
	Tell       int
	Rng        uint32
	Val        uint32
	ErrFlag    int
	Buf        []byte
}

// TfEncode runs tf_encode (celt_encoder.c:824) over a fresh range encoder bound
// to a bufLen-byte buffer, so the unsigned `budget = storage*8` arithmetic is
// exercised for real. prefillBits EncBitLogp(bit, 1) calls are issued first,
// with bit i taken from bit (i&31) of prefillPat, to advance `tell` before
// tf_encode runs (this is what lets a caller reach the budget-exhaustion path at
// :849). tfRes is copied, so the caller's slice is not touched; the mutated array
// comes back in the result.
func TfEncode(start, end, isTransient int, tfRes []int, LM, tfSelect,
	prefillBits int, prefillPat uint32, bufLen int) TfEncodeResult {
	buf := make([]byte, bufLen)
	var enc rangecoding.Encoder
	enc.Init(buf)
	for i := 0; i < prefillBits; i++ {
		enc.EncBitLogp(int((prefillPat>>(i&31))&1), 1)
	}
	tellBefore := enc.Tell()

	work := make([]int, len(tfRes))
	copy(work, tfRes)
	tfEncode(start, end, isTransient, work, LM, tfSelect, &enc)

	res := TfEncodeResult{
		TfRes:      work,
		TellBefore: tellBefore,
		Tell:       enc.Tell(),
		Rng:        enc.Rng(),
		Val:        enc.Val(),
		ErrFlag:    enc.Error(),
	}
	enc.EncDone()
	res.Buf = buf
	return res
}
