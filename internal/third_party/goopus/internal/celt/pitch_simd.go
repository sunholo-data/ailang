// The two hot pitch kernels (celt_inner_prod and xcorr_kernel) are backed by
// github.com/tphakala/simd/i16 on every architecture: a single portable file
// with no build tags. The library carries the per-arch assembly (SMLAL/SMLAL2
// on arm64, PMADDWD/AVX2 on amd64) and a pure-Go fallback elsewhere, so there
// is no in-tree assembly and no cgo.
//
// Both library functions accumulate each result in a *wrapping* int32, exactly
// as MAC16_16 does, and document that property as a guarantee: wrapping addition
// is associative and commutative modulo 2^32, so every SIMD lane grouping and
// horizontal reduction is bit-identical to the scalar loop for all inputs,
// including operands engineered to overflow (INT16_MIN products, accumulator
// wrap). That is the same reasoning pitch_ref.go relies on, so the library
// results are bit-identical to celtInnerProdGeneric / xcorrKernelGeneric. The
// differential proof is TestCeltInnerProdSIMDMatchesScalar /
// TestXcorrKernelSIMDMatchesScalar / FuzzCeltInnerProd / FuzzXcorrKernel in
// pitch_simd_test.go, which now compare the library-backed kernels against the
// scalar reference on every length 0..600 plus adversarial and fuzz inputs.

package celt

import "github.com/tphakala/simd/i16"

// celtInnerProd is celt_inner_prod_c (pitch.h:159): sum_i x[xOff+i]*y[yOff+i],
// accumulated in a wrapping int32.
//
// i16.DotProduct sums over min(len(a), len(b)), so the operands are sliced to
// exactly N. The explicit index checks below make a mis-sized caller panic here
// (a length violation, not a read into spare capacity) before the library runs,
// preserving the original bounds-check-before-call contract.
func celtInnerProd(x []int16, xOff int, y []int16, yOff, N int) int32 {
	if N <= 0 {
		return 0
	}
	_ = x[xOff+N-1]
	_ = y[yOff+N-1]
	return i16.DotProduct(x[xOff:xOff+N], y[yOff:yOff+N])
}

// xcorrKernel is xcorr_kernel_c (pitch.h:65): sum[k] += sum_j x[j]*y[k+j] for
// the four lags k in [0,4). Preconditions: len(x) >= len_, len(y) >= len_+3.
//
// i16.XCorr computes the same four lags (m = min(len(dst), len(y)-len(x)+1) = 4
// for x[:len_] against y[:len_+3]), but it WRITES dst[k] = ... where this kernel
// ACCUMULATES sum[k] += ... . To keep xcorrKernel's documented += semantics the
// library writes into a fresh temp that is then folded into sum. Wrapping int32
// addition is associative, so sum[k]+tmp[k] is bit-identical to seeding the
// accumulator at sum[k], which is exactly what xcorrKernelGeneric does.
//
// Operand order matters: i16.XCorr takes the short/stationary operand (x)
// SECOND and the long slid operand (y) THIRD; passing them swapped trips the
// len(y) < len(x) guard and writes nothing with no panic.
func xcorrKernel(x, y []int16, sum *[4]int32, len_ int) {
	if len_ <= 0 {
		// Matches the scalar reference, which accumulates nothing for len_ <= 0.
		return
	}
	// The lag-3 window reaches y[len_+2]; bounds-check the full extent the
	// library will read so a mis-sized caller panics here.
	_ = x[len_-1]
	_ = y[len_+2]
	var tmp [4]int32
	i16.XCorr(tmp[:], x[:len_], y[:len_+3])
	sum[0] += tmp[0]
	sum[1] += tmp[1]
	sum[2] += tmp[2]
	sum[3] += tmp[3]
}
