// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-GO file.
//
// Derived from go1.26.6 src/math/atan.go, atan2.go and asin.go (Cephes Math
// Library Release 2.8, atan.c; see the Go sources for the Cephes notice).
// AILANG change: every product is wrapped in float64(...) so no architecture
// can fuse it into an FMA (see doc.go).

package mathx

import "math"

// xatan evaluates a series valid in the range [0, 0.66].
func xatan(x float64) float64 {
	const (
		P0 = -8.750608600031904122785e-01
		P1 = -1.615753718733365076637e+01
		P2 = -7.500855792314704667340e+01
		P3 = -1.228866684490136173410e+02
		P4 = -6.485021904942025371773e+01
		Q0 = +2.485846490142306297962e+01
		Q1 = +1.650270098316988542046e+02
		Q2 = +4.328810604912902668951e+02
		Q3 = +4.853903996359136964868e+02
		Q4 = +1.945506571482613964425e+02
	)
	z := float64(x * x)
	// z = z * ((((P0*z+P1)*z+P2)*z+P3)*z + P4) / (((((z+Q0)*z+Q1)*z+Q2)*z+Q3)*z + Q4)
	n := float64(P0*z) + P1
	n = float64(n*z) + P2
	n = float64(n*z) + P3
	n = float64(n*z) + P4
	d := z + Q0
	d = float64(d*z) + Q1
	d = float64(d*z) + Q2
	d = float64(d*z) + Q3
	d = float64(d*z) + Q4
	z = float64(z*n) / d
	return float64(x*z) + x
}

// satan reduces its argument (known to be positive)
// to the range [0, 0.66] and calls xatan.
func satan(x float64) float64 {
	const (
		Morebits = 6.123233995736765886130e-17 // pi/2 = PIO2 + Morebits
		Tan3pio8 = 2.41421356237309504880      // tan(3*pi/8)
	)
	if x <= 0.66 {
		return xatan(x)
	}
	if x > Tan3pio8 {
		return math.Pi/2 - xatan(1/x) + Morebits
	}
	return math.Pi/4 + xatan((x-1)/(x+1)) + 0.5*Morebits
}

// Atan returns the arctangent, in radians, of x.
//
// Special cases are:
//
//	Atan(±0) = ±0
//	Atan(±Inf) = ±Pi/2
func Atan(x float64) float64 {
	if x == 0 {
		return x
	}
	if x > 0 {
		return satan(x)
	}
	return -satan(-x)
}

// Atan2 returns the arc tangent of y/x, using
// the signs of the two to determine the quadrant
// of the return value.
//
// Special cases are (in order):
//
//	Atan2(y, NaN) = NaN
//	Atan2(NaN, x) = NaN
//	Atan2(+0, x>=0) = +0
//	Atan2(-0, x>=0) = -0
//	Atan2(+0, x<=-0) = +Pi
//	Atan2(-0, x<=-0) = -Pi
//	Atan2(y>0, 0) = +Pi/2
//	Atan2(y<0, 0) = -Pi/2
//	Atan2(+Inf, +Inf) = +Pi/4
//	Atan2(-Inf, +Inf) = -Pi/4
//	Atan2(+Inf, -Inf) = 3Pi/4
//	Atan2(-Inf, -Inf) = -3Pi/4
//	Atan2(y, +Inf) = 0
//	Atan2(y>0, -Inf) = +Pi
//	Atan2(y<0, -Inf) = -Pi
//	Atan2(+Inf, x) = +Pi/2
//	Atan2(-Inf, x) = -Pi/2
func Atan2(y, x float64) float64 {
	// special cases
	switch {
	case math.IsNaN(y) || math.IsNaN(x):
		return math.NaN()
	case y == 0:
		if x >= 0 && !math.Signbit(x) {
			return math.Copysign(0, y)
		}
		return math.Copysign(math.Pi, y)
	case x == 0:
		return math.Copysign(math.Pi/2, y)
	case math.IsInf(x, 0):
		if math.IsInf(x, 1) {
			switch {
			case math.IsInf(y, 0):
				return math.Copysign(math.Pi/4, y)
			default:
				return math.Copysign(0, y)
			}
		}
		switch {
		case math.IsInf(y, 0):
			return math.Copysign(3*math.Pi/4, y)
		default:
			return math.Copysign(math.Pi, y)
		}
	case math.IsInf(y, 0):
		return math.Copysign(math.Pi/2, y)
	}

	// Call atan and determine the quadrant.
	q := Atan(y / x)
	if x < 0 {
		if q <= 0 {
			return q + math.Pi
		}
		return q - math.Pi
	}
	return q
}

// Asin returns the arcsine, in radians, of x.
//
// Special cases are:
//
//	Asin(±0) = ±0
//	Asin(x) = NaN if x < -1 or x > 1
func Asin(x float64) float64 {
	if x == 0 {
		return x // special case
	}
	sign := false
	if x < 0 {
		x = -x
		sign = true
	}
	if x > 1 {
		return math.NaN() // special case
	}

	temp := math.Sqrt(1 - float64(x*x))
	if x > 0.7 {
		temp = math.Pi/2 - satan(temp/x)
	} else {
		temp = satan(x / temp)
	}

	if sign {
		temp = -temp
	}
	return temp
}

// Acos returns the arccosine, in radians, of x.
//
// Special case is:
//
//	Acos(x) = NaN if x < -1 or x > 1
func Acos(x float64) float64 {
	return math.Pi/2 - Asin(x)
}
