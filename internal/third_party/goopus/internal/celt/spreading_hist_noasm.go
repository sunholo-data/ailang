//go:build !arm64 && !amd64

package celt

// spreadingHist on architectures without a hand-written kernel is the pure-Go
// reference. TestSpreadingHistMatchesGeneric is a no-op equality here, but
// TestSpreadingDecisionMatchesC still pins the reference to libopus.
func spreadingHist(x []int32, N int) (t0, t1, t2 int) {
	return spreadingHistGeneric(x, N)
}
