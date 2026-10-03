// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE-GO file.
//
// Derived from go1.26.6 src/math/sin.go and tan.go (Cephes Math Library
// Release 2.8, sin.c and tan.c; see the Go sources for the Cephes notice).
// AILANG change: every product is wrapped in float64(...) so no architecture
// can fuse it into an FMA (see doc.go).

package mathx

import "math"

// sin coefficients
var _sin = [...]float64{
	1.58962301576546568060e-10, // 0x3de5d8fd1fd19ccd
	-2.50507477628578072866e-8, // 0xbe5ae5e5a9291f5d
	2.75573136213857245213e-6,  // 0x3ec71de3567d48a1
	-1.98412698295895385996e-4, // 0xbf2a01a019bfdf03
	8.33333333332211858878e-3,  // 0x3f8111111110f7d0
	-1.66666666666666307295e-1, // 0xbfc5555555555548
}

// cos coefficients
var _cos = [...]float64{
	-1.13585365213876817300e-11, // 0xbda8fa49a0861a9b
	2.08757008419747316778e-9,   // 0x3e21ee9d7b4e3f05
	-2.75573141792967388112e-7,  // 0xbe927e4f7eac4bc6
	2.48015872888517045348e-5,   // 0x3efa01a019c844f5
	-1.38888888888730564116e-3,  // 0xbf56c16c16c14f91
	4.16666666666665929218e-2,   // 0x3fa555555555554b
}

// tan coefficients
var _tanP = [...]float64{
	-1.30936939181383777646e4, // 0xc0c992d8d24f3f38
	1.15351664838587416140e6,  // 0x413199eca5fc9ddd
	-1.79565251976484877988e7, // 0xc1711fead3299176
}
var _tanQ = [...]float64{
	1.00000000000000000000e0,
	1.36812963470692954678e4,  // 0x40cab8a5eeb36572
	-1.32089234440210967447e6, // 0xc13427bc582abc96
	2.50083801823357915839e7,  // 0x4177d98fc2ead8ef
	-5.38695755929454629881e7, // 0xc189afe03cbe5a31
}

// Pi/4 split into three parts
const (
	pi4A = 7.85398125648498535156e-1  // 0x3fe921fb40000000
	pi4B = 3.77489470793079817668e-8  // 0x3e64442d00000000
	pi4C = 2.69515142907905952645e-15 // 0x3ce8469898cc5170
)

// reduceOctant maps x ≥ 0 to its octant j and the reduced argument z.
// It is the shared prologue of sin, cos and tan; j is NOT reduced mod 8 here
// (tan does not reduce it, sin and cos do).
func reduceOctant(x float64) (j uint64, z float64) {
	if x >= reduceThreshold {
		return trigReduce(x)
	}
	j = uint64(float64(x * (4 / math.Pi))) // integer part of x/(Pi/4)
	y := float64(j)                        // integer part of x/(Pi/4), as float
	// map zeros to origin
	if j&1 == 1 {
		j++
		y++
	}
	// Extended precision modular arithmetic
	z = ((x - float64(y*pi4A)) - float64(y*pi4B)) - float64(y*pi4C)
	return j, z
}

// sinPoly evaluates z + z*zz*S(zz), the sine kernel on the reduced argument.
func sinPoly(z, zz float64) float64 {
	p := float64(_sin[0]*zz) + _sin[1]
	p = float64(p*zz) + _sin[2]
	p = float64(p*zz) + _sin[3]
	p = float64(p*zz) + _sin[4]
	p = float64(p*zz) + _sin[5]
	return z + float64(float64(z*zz)*p)
}

// cosPoly evaluates 1 - 0.5*zz + zz*zz*C(zz), the cosine kernel.
func cosPoly(zz float64) float64 {
	p := float64(_cos[0]*zz) + _cos[1]
	p = float64(p*zz) + _cos[2]
	p = float64(p*zz) + _cos[3]
	p = float64(p*zz) + _cos[4]
	p = float64(p*zz) + _cos[5]
	return (1.0 - float64(0.5*zz)) + float64(float64(zz*zz)*p)
}

// Cos returns the cosine of the radian argument x.
//
// Special cases are:
//
//	Cos(±Inf) = NaN
//	Cos(NaN) = NaN
func Cos(x float64) float64 {
	// special cases
	switch {
	case math.IsNaN(x) || math.IsInf(x, 0):
		return math.NaN()
	}

	// make argument positive
	sign := false
	x = math.Abs(x)

	j, z := reduceOctant(x)
	if x < reduceThreshold {
		j &= 7 // octant modulo 2Pi radians (360 degrees)
	}

	// reflect in x axis
	if j > 3 {
		j -= 4
		sign = !sign
	}
	if j > 1 {
		sign = !sign
	}

	zz := float64(z * z)
	var y float64
	if j == 1 || j == 2 {
		y = sinPoly(z, zz)
	} else {
		y = cosPoly(zz)
	}
	if sign {
		y = -y
	}
	return y
}

// Sin returns the sine of the radian argument x.
//
// Special cases are:
//
//	Sin(±0) = ±0
//	Sin(±Inf) = NaN
//	Sin(NaN) = NaN
func Sin(x float64) float64 {
	// special cases
	switch {
	case x == 0 || math.IsNaN(x):
		return x // return ±0 || NaN()
	case math.IsInf(x, 0):
		return math.NaN()
	}

	// make argument positive but save the sign
	sign := false
	if x < 0 {
		x = -x
		sign = true
	}

	j, z := reduceOctant(x)
	if x < reduceThreshold {
		j &= 7 // octant modulo 2Pi radians (360 degrees)
	}

	// reflect in x axis
	if j > 3 {
		sign = !sign
		j -= 4
	}

	zz := float64(z * z)
	var y float64
	if j == 1 || j == 2 {
		y = cosPoly(zz)
	} else {
		y = sinPoly(z, zz)
	}
	if sign {
		y = -y
	}
	return y
}

// Tan returns the tangent of the radian argument x.
//
// Special cases are:
//
//	Tan(±0) = ±0
//	Tan(±Inf) = NaN
//	Tan(NaN) = NaN
func Tan(x float64) float64 {
	// special cases
	switch {
	case x == 0 || math.IsNaN(x):
		return x // return ±0 || NaN()
	case math.IsInf(x, 0):
		return math.NaN()
	}

	// make argument positive but save the sign
	sign := false
	if x < 0 {
		x = -x
		sign = true
	}

	j, z := reduceOctant(x)

	zz := float64(z * z)
	var y float64
	if zz > 1e-14 {
		// y = z + z*(zz*(((_tanP[0]*zz)+_tanP[1])*zz+_tanP[2])/((((zz+_tanQ[1])*zz+_tanQ[2])*zz+_tanQ[3])*zz+_tanQ[4]))
		p := float64(_tanP[0]*zz) + _tanP[1]
		p = float64(p*zz) + _tanP[2]
		q := zz + _tanQ[1]
		q = float64(q*zz) + _tanQ[2]
		q = float64(q*zz) + _tanQ[3]
		q = float64(q*zz) + _tanQ[4]
		y = z + float64(z*(float64(zz*p)/q))
	} else {
		y = z
	}
	if j&2 == 2 {
		y = -1 / y
	}
	if sign {
		y = -y
	}
	return y
}
