// Transliteration of libopus celt/fixed_generic.h (v1.6.1, commit 3da9f7a6) for
// the frozen FIXED_POINT + DISABLE_FLOAT_API + OPUS_FAST_INT64 configuration.
//
// Every helper mirrors one C macro. opus_val16 maps to Go int16, opus_val32 to
// int32; shift amounts are Go int (defined for 0..31 as in CELT usage). Go's
// signed >> is arithmetic (matches C) and signed overflow wraps (defined), so
// the _ovflw variants are plain operators and PSHR/SHL keep the exact C forms
// (uint cast for SHL, int32 rounding addend for PSHR). See docs/hard-parts.md
// section 4 for the normative Go expressions and the overflow/rounding traps.

package fixedmath

// EXTEND32 changes a 16-bit value into a 32-bit value. (fixed_generic.h:111)
func EXTEND32(x int16) int32 { return int32(x) }

// EXTRACT16 changes a 32-bit value into a 16-bit value; assumed to fit.
// (fixed_generic.h:109)
func EXTRACT16(x int32) int16 { return int16(x) }

// NEG16 negates a 16-bit value. The C macro is (-(x)) with no narrowing cast,
// so the operand is promoted to int and the result is 32-bit (NEG16(-32768) ==
// 32768, not -32768). (fixed_generic.h:104)
func NEG16(x int16) int32 { return -int32(x) }

// NEG32 negates a 32-bit value. (fixed_generic.h:106)
func NEG32(x int32) int32 { return -x }

// SHR16 is an arithmetic shift-right of a 16-bit value. (fixed_generic.h:114)
func SHR16(a int16, shift int) int16 { return a >> shift }

// SHL16 is an arithmetic shift-left of a 16-bit value. C shifts UNSIGNED then
// casts back so the wrap on overflow is intended. (fixed_generic.h:116)
func SHL16(a int16, shift int) int16 { return int16(uint16(a) << shift) }

// SHR32 is an arithmetic shift-right of a 32-bit value. Go's signed >> is
// arithmetic, matching C. (fixed_generic.h:118)
func SHR32(a int32, shift int) int32 { return a >> shift }

// SHL32 is an arithmetic shift-left of a 32-bit value. C shifts UNSIGNED then
// casts back to make the overflow wrap well-defined. (fixed_generic.h:120)
func SHL32(a int32, shift int) int32 { return int32(uint32(a) << shift) }

// PSHR32 is a 32-bit arithmetic shift right with rounding-to-nearest. The
// rounding addend EXTEND32(1)<<shift>>1 is computed in int32 (it can, and is
// meant to, overflow into a+addend); keep the exact C expression.
// (fixed_generic.h:123)
func PSHR32(a int32, shift int) int32 {
	return SHR32(a+((int32(1)<<shift)>>1), shift)
}

// VSHR32 is a 32-bit arithmetic shift right where the argument can be negative:
// shift>0 shifts right, otherwise left by -shift. (fixed_generic.h:125)
func VSHR32(a int32, shift int) int32 {
	if shift > 0 {
		return SHR32(a, shift)
	}
	return SHL32(a, -shift)
}

// SATURATE clamps x to [-a, a]. (fixed_generic.h:134)
func SATURATE(x, a int32) int32 {
	switch {
	case x > a:
		return a
	case x < -a:
		return -a
	default:
		return x
	}
}

// SATURATE16 clamps x to the int16 range. (fixed_generic.h:136)
func SATURATE16(x int32) int16 {
	switch {
	case x > 32767:
		return 32767
	case x < -32768:
		return -32768
	default:
		return int16(x)
	}
}

// ROUND16 shifts by a with round-to-nearest; result is a 16-bit value.
// (fixed_generic.h:139)
func ROUND16(x int32, a int) int16 { return EXTRACT16(PSHR32(x, a)) }

// SROUND16 shifts by a with round-to-nearest, saturates to the int16 range, then
// truncates to 16 bits. (fixed_generic.h:141)
//
// NOTE: internal/celt keeps package-local wrappers (sround16 and div32 in
// celt_lpc.go, div32_16 in bands_math.go) for their pre-existing decoder call
// sites; they delegate to these three functions, so there is a single
// implementation to keep bit-exact.
func SROUND16(x int32, a int) int16 { return EXTRACT16(SATURATE(PSHR32(x, a), 32767)) }

// DIV32 is a 32-bit integer division. (fixed_generic.h:203)
func DIV32(a, b int32) int32 { return a / b }

// DIV32_16 divides a 32-bit numerator by a 16-bit denominator and truncates the
// quotient back to 16 bits. (fixed_generic.h:200)
func DIV32_16(a int32, b int16) int16 { return int16(a / int32(b)) }

// HALF16 divides a 16-bit value by two. (fixed_generic.h:144)
func HALF16(x int16) int16 { return SHR16(x, 1) }

// HALF32 divides a 32-bit value by two. (fixed_generic.h:145)
func HALF32(x int32) int32 { return SHR32(x, 1) }

// ADD16 adds two 16-bit values (int16 wrap matches the C truncation).
// (fixed_generic.h:148)
func ADD16(a, b int16) int16 { return a + b }

// SUB16 subtracts two 16-bit values. Unlike ADD16, the C macro
// ((opus_val16)(a)-(opus_val16)(b)) has NO outer (opus_val16) cast, so the
// operands are promoted to int and the result is a 32-bit int, not truncated to
// 16 bits (SUB16(-32768,1) == -32769). Callers that store into an opus_val16
// truncate with EXTRACT16. (fixed_generic.h:150)
func SUB16(a, b int16) int32 { return int32(a) - int32(b) }

// ADD32 adds two 32-bit values. (fixed_generic.h:152)
func ADD32(a, b int32) int32 { return a + b }

// SUB32 subtracts two 32-bit values. (fixed_generic.h:154)
func SUB32(a, b int32) int32 { return a - b }

// ADD32_ovflw adds two 32-bit values ignoring overflow. C routes through uint32
// to dodge UBSan; Go's signed overflow is defined to wrap, so this is a plain
// add. (fixed_generic.h:157)
func ADD32_ovflw(a, b int32) int32 { return int32(uint32(a) + uint32(b)) }

// SUB32_ovflw subtracts two 32-bit values ignoring overflow. (fixed_generic.h:159)
func SUB32_ovflw(a, b int32) int32 { return int32(uint32(a) - uint32(b)) }

// NEG32_ovflw negates a 32-bit value ignoring overflow. (fixed_generic.h:162)
func NEG32_ovflw(a int32) int32 { return int32(0 - uint32(a)) }

// SHL32_ovflw is a 32-bit shift left ignoring overflows. (fixed_generic.h:164)
func SHL32_ovflw(a int32, shift int) int32 { return SHL32(a, shift) }

// PSHR32_ovflw is PSHR32 with the addition done ignoring overflow.
// (fixed_generic.h:166)
func PSHR32_ovflw(a int32, shift int) int32 {
	return SHR32(ADD32_ovflw(a, (int32(1)<<shift)>>1), shift)
}

// MIN16 (with MAX16/MIN32/MAX32) mirrors the generic C compare macros; they
// operate on the full 32-bit value (arch.h:100-103).
func MIN16(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func MAX16(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func MIN32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func MAX32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// IMIN (with IMAX) is the int form (arch.h:104-105).
func IMIN(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func IMAX(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ABS16 (with ABS32) takes the absolute value; the C macro
// ((x)<0?(-(x)):(x)) promotes to int, so for the 16-bit form the result is
// 32-bit (ABS16(-32768) == 32768). (arch.h:227-228)
func ABS16(x int16) int32 {
	if x < 0 {
		return -int32(x)
	}
	return int32(x)
}

func ABS32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}

// MULT16_16SU multiplies a 16-bit signed value by a 16-bit unsigned value; the
// result is a 32-bit signed value. (fixed_generic.h:37)
func MULT16_16SU(a int16, b uint16) int32 { return int32(a) * int32(b) }

// MULT16_16 is a 16x16 multiplication whose result fits in 32 bits.
// (fixed_generic.h:176)
func MULT16_16(a, b int16) int32 { return int32(a) * int32(b) }

// MAC16_16 is a 16x16 multiply-add whose result fits in 32 bits.
// (fixed_generic.h:179)
func MAC16_16(c int32, a, b int16) int32 { return ADD32(c, MULT16_16(a, b)) }

// MULT16_16_Q11 (with Q13/Q14/Q15) is a truncating (not rounding) 16x16
// multiply with the given right shift. (fixed_generic.h:190-193)
func MULT16_16_Q11(a, b int16) int32 { return SHR32(MULT16_16(a, b), 11) }
func MULT16_16_Q13(a, b int16) int32 { return SHR32(MULT16_16(a, b), 13) }
func MULT16_16_Q14(a, b int16) int32 { return SHR32(MULT16_16(a, b), 14) }
func MULT16_16_Q15(a, b int16) int32 { return SHR32(MULT16_16(a, b), 15) }

// MULT16_16_P13 (with P14/P15) is a round-to-nearest 16x16 multiply.
// (fixed_generic.h:195-197)
func MULT16_16_P13(a, b int16) int32 { return SHR32(ADD32(4096, MULT16_16(a, b)), 13) }
func MULT16_16_P14(a, b int16) int32 { return SHR32(ADD32(8192, MULT16_16(a, b)), 14) }
func MULT16_16_P15(a, b int16) int32 { return SHR32(ADD32(16384, MULT16_16(a, b)), 15) }

// MULT16_32_Q16 is a 16x32 multiply followed by a 16-bit shift right, using the
// OPUS_FAST_INT64 form. (fixed_generic.h:41)
func MULT16_32_Q16(a int16, b int32) int32 {
	return int32(SHR64(int64(a)*int64(b), 16))
}

// MULT16_32_P16 is MULT16_32_Q16 with round-to-nearest (OPUS_FAST_INT64 form).
// The rounding addend EXTEND32(1)<<16>>1 = 32768 is added in int64.
// (fixed_generic.h:48)
func MULT16_32_P16(a int16, b int32) int32 {
	return int32((int64(a)*int64(b) + ((int64(1) << 16) >> 1)) >> 16)
}

// MULT16_32_Q15 is a 16x32 multiply followed by a 15-bit shift right. REQUIRES
// the OPUS_FAST_INT64 int64 form (the two-part decomposition is not bit-identical
// in overflow corners). (fixed_generic.h:55)
func MULT16_32_Q15(a int16, b int32) int32 {
	return int32(SHR64(int64(a)*int64(b), 15))
}

// MULT32_32_Q31 is a 32x32 multiply followed by a 31-bit shift right
// (OPUS_FAST_INT64 form). (fixed_generic.h:69)
func MULT32_32_Q31(a, b int32) int32 {
	return int32(SHR64(int64(a)*int64(b), 31))
}

// MULT32_32_P31 is a 32x32 multiply followed by a 31-bit shift right with
// rounding; the rounding constant is 2^30 (OPUS_FAST_INT64 form).
// (fixed_generic.h:76)
func MULT32_32_P31(a, b int32) int32 {
	return int32(SHR64(1073741824+int64(a)*int64(b), 31))
}

// MULT32_32_P31_ovflw equals MULT32_32_P31 under OPUS_FAST_INT64. (fixed_generic.h:77)
func MULT32_32_P31_ovflw(a, b int32) int32 { return MULT32_32_P31(a, b) }

// MAC16_32_Q15 is a 16x32 multiply, 15-bit shift right, and 32-bit add. Note
// this macro is defined ONLY as the two-part decomposition (never the int64
// form), so it is transliterated exactly. b must fit in 31 bits.
// (fixed_generic.h:183)
func MAC16_32_Q15(c int32, a int16, b int32) int32 {
	return ADD32(c, ADD32(MULT16_16(a, EXTRACT16(SHR32(b, 15))),
		SHR32(MULT16_16(a, EXTRACT16(b&0x00007fff)), 15)))
}

// MAC16_32_Q16 is a 16x32 multiply, 16-bit shift right, and 32-bit add. Like
// MAC16_32_Q15 it is only ever the two-part decomposition. (fixed_generic.h:187)
func MAC16_32_Q16(c int32, a int16, b int32) int32 {
	return ADD32(c, ADD32(MULT16_16(a, EXTRACT16(SHR32(b, 16))),
		SHR32(MULT16_16SU(a, uint16(b&0x0000ffff)), 16)))
}

// SHR64 is an arithmetic shift-right of a 64-bit value. (fixed_generic.h:128)
func SHR64(a int64, shift int) int64 { return a >> shift }

// QCONST16 is the compile-time conversion of a float constant to a 16-bit value.
// The float math is evaluated in float64, which is bit-identical to the C double
// the macro uses. (fixed_generic.h:92)
func QCONST16(x float64, bits int) int16 {
	return int16(0.5 + x*float64(int32(1)<<bits))
}

// QCONST32 is the compile-time conversion of a float constant to a 32-bit value.
// (fixed_generic.h:95)
func QCONST32(x float64, bits int) int32 {
	return int32(0.5 + x*float64(int64(1)<<bits))
}

// Celt_udiv divides two unsigned 32-bit values. The oracle build is not
// USE_SMALL_DIV_TABLE (that path is ARM-only and produces the same result), so
// this is plain truncating unsigned division. (entcode.h:124, entcode.h:136)
func Celt_udiv(n, d uint32) uint32 { return n / d }

// Celt_sudiv divides two signed 32-bit values (truncating toward zero, matching
// C). (entcode.h:140, entcode.h:148)
func Celt_sudiv(n, d int32) int32 { return n / d }
