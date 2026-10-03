// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-GO file.
//
// Derived from go1.26.6 src/math/exp.go (FreeBSD /usr/src/lib/msun/src/e_exp.c).
// AILANG change: every product is wrapped in float64(...) so no architecture
// can fuse it into an FMA (see doc.go).

package mathx

import "math"

// Exp returns e**x, the base-e exponential of x.
//
// Special cases are:
//
//	Exp(+Inf) = +Inf
//	Exp(NaN) = NaN
//
// Very large values overflow to 0 or +Inf.
// Very small values underflow to 1.
func Exp(x float64) float64 {
	const (
		Ln2Hi = 6.93147180369123816490e-01
		Ln2Lo = 1.90821492927058770002e-10
		Log2e = 1.44269504088896338700e+00

		Overflow  = 7.09782712893383973096e+02
		Underflow = -7.45133219101941108420e+02
		NearZero  = 1.0 / (1 << 28) // 2**-28
	)

	// special cases
	switch {
	case math.IsNaN(x):
		return x
	case x > Overflow: // handles case where x is +∞
		return math.Inf(1)
	case x < Underflow: // handles case where x is -∞
		return 0
	case -NearZero < x && x < NearZero:
		return 1 + x
	}

	// reduce; computed as r = hi - lo for extra precision.
	var k int
	switch {
	case x < 0:
		k = int(float64(Log2e*x) - 0.5)
	case x > 0:
		k = int(float64(Log2e*x) + 0.5)
	}
	hi := x - float64(float64(k)*Ln2Hi)
	lo := float64(float64(k) * Ln2Lo)

	// compute
	return expmulti(hi, lo, k)
}

// expmulti returns e**r × 2**k where r = hi - lo and |r| ≤ ln(2)/2.
func expmulti(hi, lo float64, k int) float64 {
	const (
		P1 = 1.66666666666666657415e-01  /* 0x3FC55555; 0x55555555 */
		P2 = -2.77777777770155933842e-03 /* 0xBF66C16C; 0x16BEBD93 */
		P3 = 6.61375632143793436117e-05  /* 0x3F11566A; 0xAF25DE2C */
		P4 = -1.65339022054652515390e-06 /* 0xBEBBBD41; 0xC5D26BF1 */
		P5 = 4.13813679705723846039e-08  /* 0x3E663769; 0x72BEA4D0 */
	)

	r := hi - lo
	t := float64(r * r)
	// c := r - t*(P1+t*(P2+t*(P3+t*(P4+t*P5))))
	p := float64(t*P5) + P4
	p = float64(t*p) + P3
	p = float64(t*p) + P2
	p = float64(t*p) + P1
	c := r - float64(t*p)
	// y := 1 - ((lo - (r*c)/(2-c)) - hi)
	y := 1 - ((lo - float64(r*c)/(2-c)) - hi)
	return math.Ldexp(y, k)
}
