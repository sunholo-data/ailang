// Transliteration of the fixed-point CELT math helpers from libopus
// celt/mathops.h and celt/mathops.c (v1.6.1, commit 3da9f7a6), FIXED_POINT +
// DISABLE_FLOAT_API. These are integer routines, so the port is bit-exact; the
// polynomial constants and the approximation steps are kept verbatim. Exported
// with C names (ST1003 is relaxed here); helper-only functions stay unexported.
//
// The nested Horner polynomials in the source are unrolled here into named
// temporaries. Each temporary sits exactly on an opus_val16 truncation boundary
// (the (opus_val16) cast that ADD16 applies to a MULT16_16_Q15 result), so the
// integer result is identical to the fully-nested C expression.

package fixedmath

import "math/bits"

// EC_ILOG returns the integer base-2 log + 1 of _x (the number of significant
// bits). On our targets the C macro is EC_CLZ0-EC_CLZ(_x) == bits.Len32.
// (celt/ecintrin.h:86)
func EC_ILOG(x uint32) int { return bits.Len32(x) }

// Ec_ilog is the portable fallback that EC_ILOG resolves to when no CLZ
// intrinsic exists; it is defined for zero (returns 0). (celt/entcode.c:41)
func Ec_ilog(v uint32) int { return bits.Len32(v) }

// Celt_ilog2 is the integer log in base 2. Undefined for zero and negative
// numbers. (celt/mathops.h:365)
func Celt_ilog2(x int32) int {
	// celt_sig_assert(x>0)
	return EC_ILOG(uint32(x)) - 1
}

// Celt_zlog2 is the integer log in base 2, defined for zero but not for negative
// numbers. (celt/mathops.h:374)
func Celt_zlog2(x int32) int {
	if x <= 0 {
		return 0
	}
	return Celt_ilog2(x)
}

// Isqrt32 computes floor(sqrt(_val)) with exact arithmetic. _val must be greater
// than 0. (celt/mathops.c:45)
func Isqrt32(val uint32) uint32 {
	var b uint32
	var g uint32
	var bshift int
	g = 0
	bshift = (EC_ILOG(val) - 1) >> 1
	b = 1 << bshift
	for {
		var t uint32
		t = ((g << 1) + b) << bshift
		if t <= val {
			g += b
			val -= t
		}
		b >>= 1
		bshift--
		if bshift < 0 {
			break
		}
	}
	return g
}

// Celt_rsqrt_norm is the reciprocal sqrt approximation in the range [0.25,1)
// (Q16 in, Q14 out). (celt/mathops.c:98)
func Celt_rsqrt_norm(x int32) int16 {
	var n int16
	var r int16
	var r2 int16
	var y int16
	// Range of n is [-16384,32767] ([-0.5,1) in Q15).
	n = EXTRACT16(x - 32768)
	// Rough initial guess for the root; coefficients and r are Q14.
	//   r = ADD16(23557, MULT16_16_Q15(n, ADD16(-13490, MULT16_16_Q15(n, 6713))))
	r = ADD16(-13490, EXTRACT16(MULT16_16_Q15(n, 6713)))
	r = ADD16(23557, EXTRACT16(MULT16_16_Q15(n, r)))
	// y = x*r*r-1 in Q15, computed from n and r; range of y is [-1564,1594].
	r2 = EXTRACT16(MULT16_16_Q15(r, r))
	y = SHL16(EXTRACT16(SUB16(ADD16(EXTRACT16(MULT16_16_Q15(r2, n)), r2), 16384)), 1)
	// 2nd-order Householder iteration: r += r*y*(y*0.375-0.5).
	//   ADD16(r, MULT16_16_Q15(r, MULT16_16_Q15(y, SUB16(MULT16_16_Q15(y, 12288), 16384))))
	s := SUB16(EXTRACT16(MULT16_16_Q15(y, 12288)), 16384)
	t := EXTRACT16(MULT16_16_Q15(y, EXTRACT16(s)))
	return ADD16(r, EXTRACT16(MULT16_16_Q15(r, t)))
}

// Celt_rsqrt_norm32 is the reciprocal sqrt approximation in the range [0.25,1)
// (Q31 in, Q29 out). One first-order Newton-Raphson step (r = r*(1.5-0.5*x*r*r))
// refines the Q14 celt_rsqrt_norm estimate. Split into named steps exactly as the
// C source to avoid changing the intermediate truncation points. (celt/mathops.c:126)
func Celt_rsqrt_norm32(x int32) int32 {
	var tmp int32
	rQ29 := SHL32(EXTEND32(Celt_rsqrt_norm(SHR32(x, 31-16))), 15)
	tmp = MULT32_32_Q31(rQ29, rQ29)
	tmp = MULT32_32_Q31(1073741824 /* Q31 */, tmp)
	tmp = MULT32_32_Q31(x, tmp)
	return SHL32(MULT32_32_Q31(rQ29, SUB32(201326592 /* Q27 */, tmp)), 4)
}

// Celt_sqrt is the sqrt approximation (QX input, QX/2 output). (celt/mathops.c:140)
func Celt_sqrt(x int32) int32 {
	var k int
	var n int16
	var rt int32
	// Coeffs optimized in fixed-point to minimize RMS and max error of sqrt(x)
	// over .25<x<1 without exceeding 32767.
	C := [6]int16{23171, 11574, -2901, 1592, -1002, 336}
	if x == 0 {
		return 0
	} else if x >= 1073741824 {
		return 32767
	}
	k = (Celt_ilog2(x) >> 1) - 7
	x = VSHR32(x, 2*k)
	n = EXTRACT16(x - 32768)
	// rt = ADD32(C[0], MULT16_16_Q15(n, ADD16(C[1], ... ADD16(C[4], MULT16_16_Q15(n, C[5])))))
	t := ADD16(C[4], EXTRACT16(MULT16_16_Q15(n, C[5])))
	t = ADD16(C[3], EXTRACT16(MULT16_16_Q15(n, t)))
	t = ADD16(C[2], EXTRACT16(MULT16_16_Q15(n, t)))
	t = ADD16(C[1], EXTRACT16(MULT16_16_Q15(n, t)))
	rt = ADD32(EXTEND32(C[0]), MULT16_16_Q15(n, t))
	rt = VSHR32(rt, 7-k)
	return rt
}

// Celt_sqrt32 performs fixed-point arithmetic to approximate the square root.
// When the input is in Qx format, the output will be in Q(x/2 + 16) format.
// (celt/mathops.c:164)
func Celt_sqrt32(x int32) int32 {
	var k int
	var xFrac int32
	if x == 0 {
		return 0
	} else if x >= 1073741824 {
		return 2147483647 // 2^31 - 1
	}
	k = Celt_ilog2(x) >> 1
	xFrac = VSHR32(x, 2*(k-14)-1)
	xFrac = MULT32_32_Q31(Celt_rsqrt_norm32(xFrac), xFrac)
	if k < 12 {
		return PSHR32(xFrac, 12-k)
	}
	return SHL32(xFrac, k-12)
}

// celt_cos_pi_2 is _celt_cos_pi_2 (celt/mathops.c:184).
func celt_cos_pi_2(x int16) int16 {
	var x2 int16
	const (
		L1 = 32767
		L2 = -7651
		L3 = 8277
		L4 = -626
	)
	x2 = EXTRACT16(MULT16_16_P15(x, x))
	// ADD16(1, MIN16(32766, ADD32(SUB16(L1,x2),
	//   MULT16_16_P15(x2, ADD32(L2, MULT16_16_P15(x2, ADD32(L3, MULT16_16_P15(L4, x2))))))))
	t := ADD32(L3, MULT16_16_P15(L4, x2))
	t = ADD32(L2, MULT16_16_P15(x2, EXTRACT16(t)))
	t = ADD32(SUB16(L1, x2), MULT16_16_P15(x2, EXTRACT16(t)))
	return ADD16(1, EXTRACT16(MIN16(32766, t)))
}

// Celt_cos_norm computes cos(PI/2 * x) with x in Q16 (celt/mathops.c:198).
func Celt_cos_norm(x int32) int16 {
	x = x & 0x0001ffff
	if x > SHL32(EXTEND32(1), 16) {
		x = SUB32(SHL32(EXTEND32(1), 17), x)
	}
	if x&0x00007fff != 0 {
		if x < SHL32(EXTEND32(1), 15) {
			return celt_cos_pi_2(EXTRACT16(x))
		}
		return EXTRACT16(NEG16(celt_cos_pi_2(EXTRACT16(65536 - x))))
	}
	if x&0x0000ffff != 0 {
		return 0
	} else if x&0x0001ffff != 0 {
		return -32767
	}
	return 32767
}

// celt_rcp_norm16 computes a 16-bit approximate reciprocal (1/x) for a
// normalized Q15 input, resulting in a Q15 output. (celt/mathops.c:245)
func celt_rcp_norm16(x int16) int16 {
	var r int16
	// Linear approximation r = 1.8823529411764706-0.9411764705882353*n, Q14.
	r = ADD16(30840, EXTRACT16(MULT16_16_Q15(-15420, x)))
	// Two Newton iterations: r -= r*((r*n)+(r-1.Q15)).
	p := ADD16(EXTRACT16(MULT16_16_Q15(r, x)), ADD16(r, -32768))
	r = EXTRACT16(SUB16(r, EXTRACT16(MULT16_16_Q15(r, p))))
	// Subtract an extra 1 in the second iteration to avoid overflow.
	p = ADD16(EXTRACT16(MULT16_16_Q15(r, x)), ADD16(r, -32768))
	return EXTRACT16(SUB16(r, ADD16(1, EXTRACT16(MULT16_16_Q15(r, p)))))
}

// Celt_rcp is the reciprocal approximation (Q15 input, Q16 output).
// (celt/mathops.c:287)
func Celt_rcp(x int32) int32 {
	var i int
	var r int16
	// celt_sig_assert(x>0)
	i = Celt_ilog2(x)
	// Compute the reciprocal of a Q15 number in the range [0, 1).
	r = celt_rcp_norm16(EXTRACT16(VSHR32(x, i-15) - 32768))
	// r is now the Q15 solution to 2/(n+1).
	return VSHR32(EXTEND32(r), i-16)
}

// Celt_rcp_norm32 computes a 32-bit approximate reciprocal (1/x) for a
// normalized Q31 input, resulting in a Q30 output. The expected input range is
// [0.5f, 1.0f) in Q31 and the expected output range is [1.0f, 2.0f) in Q30.
// (celt/mathops.c:266)
func Celt_rcp_norm32(x int32) int32 {
	var rQ30 int32
	// celt_sig_assert(x >= 1073741824)
	rQ30 = SHL32(EXTEND32(celt_rcp_norm16(EXTRACT16(SHR32(x, 15)-32768))), 16)
	// Solving f(y) = a - 1/y using the Newton Method (f(y)' = 1/y^2):
	//   r = r - f(r)/f(r)' = r - r*(r*x - 1)
	// where r is 1/y's approximation and x is a, the function input. It adds 1 to
	// avoid overflow; -1.0f in Q30 is -1073741824.
	return SUB32(rQ30, ADD32(SHL32(
		MULT32_32_Q31(ADD32(MULT32_32_Q31(rQ30, x), -1073741824),
			rQ30), 1), 1))
}

// D0..D3 constants for Celt_exp2_frac (celt/mathops.h:413-416).
const (
	d0 = 16383
	d1 = 22804
	d2 = 14819
	d3 = 10204
)

// Celt_exp2_frac computes the fractional part of the base-2 exponential.
// (celt/mathops.h:418)
func Celt_exp2_frac(x int16) int32 {
	var frac int16
	frac = SHL16(x, 4)
	// ADD16(D0, MULT16_16_Q15(frac, ADD16(D1, MULT16_16_Q15(frac, ADD16(D2, MULT16_16_Q15(D3, frac))))))
	t := ADD16(d2, EXTRACT16(MULT16_16_Q15(d3, frac)))
	t = ADD16(d1, EXTRACT16(MULT16_16_Q15(frac, t)))
	return EXTEND32(ADD16(d0, EXTRACT16(MULT16_16_Q15(frac, t))))
}

// Celt_exp2 is the base-2 exponential approximation (2^x). (Q10 input, Q16
// output). (celt/mathops.h:431)
func Celt_exp2(x int16) int32 {
	var integer int
	var frac int16
	integer = int(SHR16(x, 10))
	if integer > 14 {
		return 0x7f000000
	} else if integer < -15 {
		return 0
	}
	frac = EXTRACT16(Celt_exp2_frac(x - SHL16(int16(integer), 10)))
	return VSHR32(EXTEND32(frac), -integer-2)
}

// Celt_log2 is the base-2 logarithm approximation (log2(x)). (Q14 input, Q10
// output). (celt/mathops.h:392)
func Celt_log2(x int32) int16 {
	var i int
	var n, frac int16
	// -0.41509302963303146, 0.9609890551383969, -0.31836011537636605,
	//  0.15530808010959576, -0.08556153059057618
	C := [5]int16{-6801 + (1 << (13 - 10)), 15746, -5217, 2545, -1401}
	if x == 0 {
		return -32767
	}
	i = Celt_ilog2(x)
	n = EXTRACT16(VSHR32(x, i-15) - 32768 - 16384)
	// ADD16(C[0], MULT16_16_Q15(n, ADD16(C[1], ... ADD16(C[3], MULT16_16_Q15(n, C[4])))))
	frac = ADD16(C[3], EXTRACT16(MULT16_16_Q15(n, C[4])))
	frac = ADD16(C[2], EXTRACT16(MULT16_16_Q15(n, frac)))
	frac = ADD16(C[1], EXTRACT16(MULT16_16_Q15(n, frac)))
	frac = ADD16(C[0], EXTRACT16(MULT16_16_Q15(n, frac)))
	return EXTRACT16(SHL32(int32(i-13), 10) + SHR32(EXTEND32(frac), 14-10))
}

// Frac_div32_q29 divides a by b, result in Q29. (celt/mathops.c:72)
func Frac_div32_q29(a, b int32) int32 {
	var rcp int16
	var result, rem int32
	shift := Celt_ilog2(b) - 29
	a = VSHR32(a, shift)
	b = VSHR32(b, shift)
	// 16-bit reciprocal
	rcp = ROUND16(Celt_rcp(EXTEND32(ROUND16(b, 16))), 3)
	result = MULT16_32_Q15(rcp, a)
	rem = PSHR32(a, 2) - MULT32_32_Q31(result, b)
	result = ADD32(result, SHL32(MULT16_32_Q15(rcp, rem), 2))
	return result
}

// Frac_div32 divides a by b, saturating the result. (celt/mathops.c:87)
func Frac_div32(a, b int32) int32 {
	result := Frac_div32_q29(a, b)
	if result >= 536870912 { //  2^29
		return 2147483647 //  2^31 - 1
	} else if result <= -536870912 { // -2^29
		return -2147483647 // -2^31
	}
	return SHL32(result, 2)
}
