//go:build amd64

package celt

// spreadingHistSSE2 is the hand-written SSE2 kernel (spreading_hist_amd64.s),
// processing 4 int32 lanes per iteration with a scalar tail. On Intel the pure-Go
// restructure does not help (the original inline is already optimal there), so
// amd64 needs real vectorization for a win. Bit-exact against spreadingHistGeneric
// (TestSpreadingHistMatchesGeneric). n must satisfy n <= len(x).
//
// The two int16 truncations that int16(...) performs are done with PSRAL/PAND
// 0xFFFF plus PMADDWL (not a saturating PACKSSDW), so a >>10 or (xj*xj)>>15 that
// leaves int16 range wraps exactly as the scalar reference does.
//
//go:noescape
func spreadingHistSSE2(x []int32, n int) (t0, t1, t2 int)

func spreadingHist(x []int32, N int) (t0, t1, t2 int) {
	if N <= 0 {
		return 0, 0, 0
	}
	_ = x[N-1] // bounds-check the extent the kernel reads before it runs
	return spreadingHistSSE2(x, N)
}
