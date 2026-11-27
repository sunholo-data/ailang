// Transliteration of libopus silk/Inlines.h and the standalone scalar
// approximations silk/log2lin.c and silk/lin2log.c (v1.6.1, commit 3da9f7a6),
// FIXED_POINT + DISABLE_FLOAT_API. These are integer routines, so the port is
// bit-exact; the constants and the approximation steps are kept verbatim.
// Names carry the C name with a capitalized leading letter so they export
// (Silk_SQRT_APPROX <- silk_SQRT_APPROX), matching internal/fixedmath.
//
// silk_CLZ_FRAC writes its two outputs through pointers in C; the Go form
// returns (lz, frac_Q7) instead, which the callers destructure.

package silkmath

// Silk_CLZ64 counts the leading zeros of a 64-bit value. (Inlines.h:37)
func Silk_CLZ64(in int64) int32 {
	var in_upper int32

	in_upper = int32(Silk_RSHIFT64(in, 32))
	if in_upper == 0 {
		/* Search in the lower 32 bits */
		return 32 + Silk_CLZ32(int32(in))
	}
	/* Search in the upper 32 bits */
	return Silk_CLZ32(in_upper)
}

// Silk_CLZ_FRAC returns the number of leading zeros of in and the 7 bits right
// after the leading one (frac_Q7). (Inlines.h:52)
func Silk_CLZ_FRAC(in int32) (lz, frac_Q7 int32) {
	lzeros := Silk_CLZ32(in)

	lz = lzeros
	frac_Q7 = Silk_ROR32(in, int(24-lzeros)) & 0x7f
	return
}

// Silk_SQRT_APPROX is an approximation of the square root. Accuracy: < +/- 10%
// for output values > 15, < +/- 2.5% for output values > 120. (Inlines.h:67)
func Silk_SQRT_APPROX(x int32) int32 {
	var y, lz, frac_Q7 int32

	if x <= 0 {
		return 0
	}

	lz, frac_Q7 = Silk_CLZ_FRAC(x)

	if lz&1 != 0 {
		y = 32768
	} else {
		y = 46214 /* 46214 = sqrt(2) * 32768 */
	}

	/* get scaling right */
	y = y >> Silk_RSHIFT(lz, 1)

	/* increment using fractional part of input */
	y = Silk_SMLAWB(y, y, Silk_SMULBB(213, frac_Q7))

	return y
}

// Silk_DIV32_varQ returns a good approximation of "(a32 << Qres) / b32". Qres
// must be >= 0 and b32 must be nonzero. (Inlines.h:93)
func Silk_DIV32_varQ(a32, b32 int32, Qres int) int32 {
	var a_headrm, b_headrm, lshift int
	var b32_inv, a32_nrm, b32_nrm, result int32

	// silk_assert( b32 != 0 );
	// silk_assert( Qres >= 0 );

	/* Compute number of bits head room and normalize inputs */
	a_headrm = int(Silk_CLZ32(Silk_abs(a32))) - 1
	a32_nrm = Silk_LSHIFT(a32, a_headrm) /* Q: a_headrm                  */
	b_headrm = int(Silk_CLZ32(Silk_abs(b32))) - 1
	b32_nrm = Silk_LSHIFT(b32, b_headrm) /* Q: b_headrm                  */

	/* Inverse of b32, with 14 bits of precision */
	b32_inv = Silk_DIV32_16(silk_int32_MAX>>2, Silk_RSHIFT(b32_nrm, 16)) /* Q: 29 + 16 - b_headrm        */

	/* First approximation */
	result = Silk_SMULWB(a32_nrm, b32_inv) /* Q: 29 + a_headrm - b_headrm  */

	/* Compute residual by subtracting product of denominator and first approximation */
	/* It's OK to overflow because the final value of a32_nrm should always be small */
	a32_nrm = Silk_SUB32_ovflw(a32_nrm, Silk_LSHIFT_ovflw(Silk_SMMUL(b32_nrm, result), 3)) /* Q: a_headrm   */

	/* Refinement */
	result = Silk_SMLAWB(result, a32_nrm, b32_inv) /* Q: 29 + a_headrm - b_headrm  */

	/* Convert to Qres domain */
	lshift = 29 + a_headrm - b_headrm - Qres
	if lshift < 0 {
		return Silk_LSHIFT_SAT32(result, -lshift)
	}
	if lshift < 32 {
		return Silk_RSHIFT(result, lshift)
	}
	/* Avoid undefined result */
	return 0
}

// Silk_INVERSE32_varQ returns a good approximation of "(1 << Qres) / b32". Qres
// must be > 0 and b32 must be nonzero. (Inlines.h:139)
func Silk_INVERSE32_varQ(b32 int32, Qres int) int32 {
	var b_headrm, lshift int
	var b32_inv, b32_nrm, err_Q32, result int32

	// silk_assert( b32 != 0 );
	// silk_assert( Qres > 0 );

	/* Compute number of bits head room and normalize input */
	b_headrm = int(Silk_CLZ32(Silk_abs(b32))) - 1
	b32_nrm = Silk_LSHIFT(b32, b_headrm) /* Q: b_headrm                */

	/* Inverse of b32, with 14 bits of precision */
	b32_inv = Silk_DIV32_16(silk_int32_MAX>>2, Silk_RSHIFT(b32_nrm, 16)) /* Q: 29 + 16 - b_headrm    */

	/* First approximation */
	result = Silk_LSHIFT(b32_inv, 16) /* Q: 61 - b_headrm            */

	/* Compute residual by subtracting product of denominator and first approximation from one */
	err_Q32 = Silk_LSHIFT((int32(1)<<29)-Silk_SMULWB(b32_nrm, b32_inv), 3) /* Q32                        */

	/* Refinement */
	result = Silk_SMLAWW(result, err_Q32, b32_inv) /* Q: 61 - b_headrm            */

	/* Convert to Qres domain */
	lshift = 61 - b_headrm - Qres
	if lshift <= 0 {
		return Silk_LSHIFT_SAT32(result, -lshift)
	}
	if lshift < 32 {
		return Silk_RSHIFT(result, lshift)
	}
	/* Avoid undefined result */
	return 0
}

// Silk_log2lin is an approximation of 2^() (a very close inverse of
// silk_lin2log). Convert input on a log scale to a linear scale. (log2lin.c:36)
func Silk_log2lin(inLog_Q7 int32) int32 {
	var out, frac_Q7 int32

	if inLog_Q7 < 0 {
		return 0
	} else if inLog_Q7 >= 3967 {
		return silk_int32_MAX
	}

	out = Silk_LSHIFT(1, int(Silk_RSHIFT(inLog_Q7, 7)))
	frac_Q7 = inLog_Q7 & 0x7F
	if inLog_Q7 < 2048 {
		/* Piece-wise parabolic approximation */
		out = Silk_ADD_RSHIFT32(out, Silk_MUL(out, Silk_SMLAWB(frac_Q7, Silk_SMULBB(frac_Q7, 128-frac_Q7), -174)), 7)
	} else {
		/* Piece-wise parabolic approximation */
		out = Silk_MLA(out, Silk_RSHIFT(out, 7), Silk_SMLAWB(frac_Q7, Silk_SMULBB(frac_Q7, 128-frac_Q7), -174))
	}
	return out
}

// Silk_lin2log is an approximation of 128 * log2() (a very close inverse of
// silk_log2lin). Convert input on a linear scale to a log scale. (lin2log.c:35)
func Silk_lin2log(inLin int32) int32 {
	var lz, frac_Q7 int32

	lz, frac_Q7 = Silk_CLZ_FRAC(inLin)

	/* Piece-wise parabolic approximation */
	return Silk_ADD_LSHIFT32(Silk_SMLAWB(frac_Q7, Silk_MUL(frac_Q7, 128-frac_Q7), 179), 31-lz, 7)
}
