// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-GO file.
//
// Derived from go1.26.6 src/math/log.go and log10.go (FreeBSD
// /usr/src/lib/msun/src/e_log.c). AILANG change: every product is wrapped in
// float64(...) so no architecture can fuse it into an FMA (see doc.go).

package mathx

import "math"

// Log returns the natural logarithm of x.
//
// Special cases are:
//
//	Log(+Inf) = +Inf
//	Log(0) = -Inf
//	Log(x < 0) = NaN
//	Log(NaN) = NaN
func Log(x float64) float64 {
	const (
		Ln2Hi = 6.93147180369123816490e-01 /* 3fe62e42 fee00000 */
		Ln2Lo = 1.90821492927058770002e-10 /* 3dea39ef 35793c76 */
		L1    = 6.666666666666735130e-01   /* 3FE55555 55555593 */
		L2    = 3.999999999940941908e-01   /* 3FD99999 9997FA04 */
		L3    = 2.857142874366239149e-01   /* 3FD24924 94229359 */
		L4    = 2.222219843214978396e-01   /* 3FCC71C5 1D8E78AF */
		L5    = 1.818357216161805012e-01   /* 3FC74664 96CB03DE */
		L6    = 1.531383769920937332e-01   /* 3FC39A09 D078C69F */
		L7    = 1.479819860511658591e-01   /* 3FC2F112 DF3E5244 */
	)

	// special cases
	switch {
	case math.IsNaN(x) || math.IsInf(x, 1):
		return x
	case x < 0:
		return math.NaN()
	case x == 0:
		return math.Inf(-1)
	}

	// reduce
	f1, ki := math.Frexp(x)
	if f1 < math.Sqrt2/2 {
		f1 *= 2
		ki--
	}
	f := f1 - 1
	k := float64(ki)

	// compute
	s := f / (2 + f)
	s2 := float64(s * s)
	s4 := float64(s2 * s2)
	// t1 := s2 * (L1 + s4*(L3+s4*(L5+s4*L7)))
	t1 := float64(s4*L7) + L5
	t1 = float64(s4*t1) + L3
	t1 = float64(s4*t1) + L1
	t1 = float64(s2 * t1)
	// t2 := s4 * (L2 + s4*(L4+s4*L6))
	t2 := float64(s4*L6) + L4
	t2 = float64(s4*t2) + L2
	t2 = float64(s4 * t2)
	R := t1 + t2
	hfsq := float64(float64(0.5*f) * f)
	// k*Ln2Hi - ((hfsq - (s*(hfsq+R) + k*Ln2Lo)) - f)
	return float64(k*Ln2Hi) - ((hfsq - (float64(s*(hfsq+R)) + float64(k*Ln2Lo))) - f)
}

// Log10 returns the decimal logarithm of x.
// The special cases are the same as for Log.
func Log10(x float64) float64 {
	return float64(Log(x) * (1 / math.Ln10))
}
