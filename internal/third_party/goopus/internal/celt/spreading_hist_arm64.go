//go:build arm64

package celt

// spreadingHistNEON is the hand-written NEON kernel (spreading_hist_arm64.s). Go
// 1.26's arm64 assembler has no mnemonics for the integer NEON ops this needs
// (SSHR, XTN, SMULL, CMGT), so the vector body is WORD-encoded (encodings taken
// from clang, cross-checked, and pinned bit-for-bit against spreadingHistGeneric
// by TestSpreadingHistMatchesGeneric across every N and adversarial pattern). It
// processes 4 int32 lanes per iteration with a scalar tail. n must satisfy
// n <= len(x); the wrapper below enforces it before the pointer reaches asm.
//
//go:noescape
func spreadingHistNEON(x []int32, n int) (t0, t1, t2 int)

func spreadingHist(x []int32, N int) (t0, t1, t2 int) {
	if N <= 0 {
		return 0, 0, 0
	}
	_ = x[N-1] // bounds-check the extent the kernel reads before it runs
	return spreadingHistNEON(x, N)
}
