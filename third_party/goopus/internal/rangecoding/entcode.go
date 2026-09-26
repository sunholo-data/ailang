/* Copyright (c) 2001-2011 Timothy B. Terriberry
   Copyright (c) 2008-2009 Xiph.Org Foundation */
/*
   Redistribution and use in source and binary forms, with or without
   modification, are permitted provided that the following conditions
   are met:

   - Redistributions of source code must retain the above copyright
   notice, this list of conditions and the following disclaimer.

   - Redistributions in binary form must reproduce the above copyright
   notice, this list of conditions and the following disclaimer in the
   documentation and/or other materials provided with the distribution.

   THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
   ``AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
   LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
   A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER
   OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
   EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
   PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
   PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF
   LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
   NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
   SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

// This file transliterates celt/entcode.h, celt/entcode.c and celt/mfrngcod.h:
// the shared constants, ec_ilog / celt_udiv, and ec_tell / ec_tell_frac, which
// are common to the encoder (entenc.go) and the decoder (entdec.go). The C code
// uses one struct (ec_ctx) for both directions; Go splits it into Encoder and
// Decoder with the same field set, so the tell helpers take the two fields they
// read (nbits_total, rng) instead of a *ec_ctx.

package rangecoding

import "math/bits"

// Constants used by the entropy encoder/decoder (celt/mfrngcod.h).
const (
	// The number of bits to output at a time.
	ecSymBits = 8
	// The total number of bits in each of the state registers.
	ecCodeBits = 32
	// The maximum symbol value.
	ecSymMax = (1 << ecSymBits) - 1
	// Bits to shift by to move a symbol into the high-order position.
	ecCodeShift = ecCodeBits - ecSymBits - 1
	// Carry bit of the high-order range symbol.
	ecCodeTop = 1 << (ecCodeBits - 1)
	// Low-order bit of the high-order range symbol.
	ecCodeBot = ecCodeTop >> ecSymBits
	// The number of bits available for the last, partial symbol in the code
	// field.
	ecCodeExtra = (ecCodeBits-2)%ecSymBits + 1
)

// The number of bits to use for the range-coded part of unsigned integers
// (celt/entcode.h).
const ecUintBits = 8

// ec_window must be at least 32 bits (celt/entcode.h); EC_WINDOW_SIZE is its
// bit width.
const ecWindowSize = 32

// The resolution of fractional-precision bit usage measurements, i.e.,
// 3 => 1/8th bits (celt/entcode.h).
const bitres = 3

// ecWindow mirrors the C `ec_window` type (opus_uint32).
type ecWindow = uint32

// ecIlog is EC_ILOG(_x): the index of the most significant set bit of _x, i.e.,
// EC_CODE_BITS-EC_CLZ(_x) for _x!=0 and 0 for _x==0. bits.Len32 has exactly this
// definition, so it matches both the EC_CLZ path and the ec_ilog() fallback in
// entcode.c.
func ecIlog(v uint32) int {
	return bits.Len32(v)
}

// celtUdiv is celt_udiv, tested exhaustively for all n and for 1<=d<=256. Our
// frozen build never defines OPUS_ARM_ASM, so USE_SMALL_DIV_TABLE is off and
// celt_udiv() is plain unsigned division; the table path is documented to return
// the identical result. Go's uint32 division truncates toward zero, as C's.
func celtUdiv(n, d uint32) uint32 {
	return n / d
}

// ecTell returns the number of bits "used" by the encoded or decoded symbols so
// far (ec_tell). This same number can be computed in either the encoder or the
// decoder, and is suitable for making coding decisions. It will always be
// slightly larger than the exact value (all rounding error is positive).
func ecTell(nbitsTotal int, rng uint32) int {
	return nbitsTotal - ecIlog(rng)
}

// ecTellFrac returns the number of bits used so far scaled by 2**BITRES
// (ec_tell_frac). This is the faster ec_tell_frac() that takes advantage of the
// low (1/8 bit) resolution to use just a linear function followed by a lookup to
// determine the exact transition thresholds.
func ecTellFrac(nbitsTotal int, rng uint32) uint32 {
	correction := [8]uint32{
		35733, 38967, 42495, 46340,
		50535, 55109, 60097, 65535,
	}
	var nbits uint32
	var r uint32
	var l int
	var b uint32
	nbits = uint32(nbitsTotal) << bitres
	l = ecIlog(rng)
	r = rng >> (l - 16)
	b = (r >> 12) - 8
	if r > correction[b] {
		b++
	}
	l = (l << 3) + int(b)
	return nbits - uint32(l)
}
