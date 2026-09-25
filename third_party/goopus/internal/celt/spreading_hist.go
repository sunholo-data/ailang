package celt

import "github.com/sunholo-data/ailang/third_party/goopus/internal/fixedmath"

// Q13 thresholds of the spreading-decision CDF (celt/bands.c spreading_decision):
// QCONST16(0.25,13)=2048, QCONST16(0.0625,13)=512, QCONST16(0.015625,13)=128.
// The float->fixed conversions fold to these constants at compile time.
const (
	spreadThr0 = int32(2048) // 0.25     in Q13
	spreadThr1 = int32(512)  // 0.0625   in Q13
	spreadThr2 = int32(128)  // 0.015625 in Q13
)

// spreadingHist computes the 3-bin magnitude CDF of one band, the vectorizable
// core of spreadingDecision (celt/bands.c). For each position j in [0, N):
//
//	xj  = int16(x[j] >> (normShift-14))         // to Q14, truncated to int16
//	x2N = int16((xj*xj) >> 15) * int16(N)        // |x|^2 * N, Q13 (int32)
//
// then t0/t1/t2 count how many x2N fall below the nested thresholds 2048/512/128.
// Because the thresholds are nested, t2 <= t1 <= t0, but the three counts stay
// independent so the result is bit-identical to the scalar reference. The counts
// are order-independent (no cross-position dependency beyond the three sums), so
// this is safe to reassociate and to vectorize. It is dispatched per architecture
// (spreading_hist_{arm64,amd64,noasm}.go); this file holds the shared constants
// and the pure-Go reference.
//
// spreadingHistGeneric is bit-exact against C spreading_decision through
// TestSpreadingDecisionMatchesC, and is the oracle the SIMD kernels are checked
// against in TestSpreadingHistMatchesGeneric. Callers pass N with N <= len(x).
func spreadingHistGeneric(x []int32, N int) (t0, t1, t2 int) {
	if N <= 0 {
		return 0, 0, 0
	}
	x = x[:N] // BCE: make the per-position read below check-free (one slice check)
	nT := int16(N)
	for j := 0; j < N; j++ {
		xj := int16(fixedmath.SHR32(x[j], normShift-14))
		x2N := fixedmath.MULT16_16(int16(fixedmath.MULT16_16_Q15(xj, xj)), nT)
		if x2N < spreadThr0 {
			t0++
		}
		if x2N < spreadThr1 {
			t1++
		}
		if x2N < spreadThr2 {
			t2++
		}
	}
	return
}
