// Transliteration of libopus silk/macros.h and the fixed-point macros in
// silk/SigProc_FIX.h (v1.6.1, commit 3da9f7a6) for the frozen FIXED_POINT +
// DISABLE_FLOAT_API + OPUS_FAST_INT64 configuration.
//
// SILK uses a fixed-point macro dialect distinct from CELT's fixed_generic.h,
// which is why this lives in its own package. opus_int16 maps to Go int16,
// opus_int32 to int32, opus_int64 to int64; opus_int (the C `int`) maps to Go
// int. Every helper mirrors one C macro; the C name is preserved with only the
// leading letter capitalized so the symbol is exported (Silk_SMULWB <- silk_
// SMULWB), exactly as internal/fixedmath does (Celt_ilog2 <- celt_ilog2).
// ST1003 is relaxed for this package. The OPUS_FAST_INT64 branch of every
// SMUL*/SMLA* macro is the one transliterated here, because the oracle build
// asserts OPUS_FAST_INT64 at compile time (see docs/hard-parts.md section 4).
// Go's signed >> is arithmetic (matches C) and signed overflow wraps (defined),
// so the _ovflw variants are plain operators and the shift macros keep the exact
// C forms (uint cast for left shifts).

package silkmath

import "math/bits"

// Integer range constants (silk/typedef.h:40-48). Kept unexported; the exported
// helpers below return them where the C returns silk_intXX_MIN/MAX.
const (
	silk_int64_MAX = int64(0x7FFFFFFFFFFFFFFF)      //  2^63 - 1
	silk_int64_MIN = int64(-0x7FFFFFFFFFFFFFFF - 1) // -2^63
	silk_int32_MAX = int32(0x7FFFFFFF)              //  2^31 - 1 =  2147483647
	silk_int32_MIN = int32(-0x7FFFFFFF - 1)         // -2^31     = -2147483648
	silk_int16_MAX = int16(0x7FFF)                  //  2^15 - 1 =  32767
	silk_int16_MIN = int16(-0x8000)                 // -2^15     = -32768
	silk_int8_MAX  = int8(0x7F)                     //  2^7 - 1  =  127
	silk_int8_MIN  = int8(-0x80)                    // -2^7      = -128
)

// ------------------------------------------------------------------ //
// silk/macros.h: OPUS_FAST_INT64 SMUL*/SMLA* forms.                    //
// ------------------------------------------------------------------ //

// Silk_SMULWB computes (a32 * (opus_int16)(b32)) >> 16 as a 32-bit int. The
// OPUS_FAST_INT64 form widens to int64 before the shift. (macros.h:43)
func Silk_SMULWB(a32, b32 int32) int32 {
	return int32((int64(a32) * int64(int16(b32))) >> 16)
}

// Silk_SMLAWB computes a32 + ((b32 * (opus_int16)(c32)) >> 16). (macros.h:50)
func Silk_SMLAWB(a32, b32, c32 int32) int32 {
	return int32(int64(a32) + ((int64(b32) * int64(int16(c32))) >> 16))
}

// Silk_SMULWT computes (a32 * (b32 >> 16)) >> 16. (macros.h:57)
func Silk_SMULWT(a32, b32 int32) int32 {
	return int32((int64(a32) * int64(b32>>16)) >> 16)
}

// Silk_SMLAWT computes a32 + ((b32 * (c32 >> 16)) >> 16). (macros.h:64)
func Silk_SMLAWT(a32, b32, c32 int32) int32 {
	return int32(int64(a32) + ((int64(b32) * (int64(c32) >> 16)) >> 16))
}

// Silk_SMULBB computes (opus_int16)(a32) * (opus_int16)(b32). (macros.h:70)
func Silk_SMULBB(a32, b32 int32) int32 {
	return int32(int16(a32)) * int32(int16(b32))
}

// Silk_SMLABB computes a32 + (opus_int16)(b32) * (opus_int16)(c32). (macros.h:73)
func Silk_SMLABB(a32, b32, c32 int32) int32 {
	return a32 + int32(int16(b32))*int32(int16(c32))
}

// Silk_SMULBT computes (opus_int16)(a32) * (b32 >> 16). (macros.h:76)
func Silk_SMULBT(a32, b32 int32) int32 {
	return int32(int16(a32)) * (b32 >> 16)
}

// Silk_SMLABT computes a32 + (opus_int16)(b32) * (c32 >> 16). (macros.h:79)
func Silk_SMLABT(a32, b32, c32 int32) int32 {
	return a32 + int32(int16(b32))*(c32>>16)
}

// Silk_SMLAL computes a64 + (opus_int64)(b32) * (opus_int64)(c32). (macros.h:82)
func Silk_SMLAL(a64 int64, b32, c32 int32) int64 {
	return Silk_ADD64(a64, int64(b32)*int64(c32))
}

// Silk_SMULWW computes (a32 * b32) >> 16 (int64 form). (macros.h:86)
func Silk_SMULWW(a32, b32 int32) int32 {
	return int32((int64(a32) * int64(b32)) >> 16)
}

// Silk_SMLAWW computes a32 + ((b32 * c32) >> 16) (int64 form). (macros.h:93)
func Silk_SMLAWW(a32, b32, c32 int32) int32 {
	return int32(int64(a32) + ((int64(b32) * int64(c32)) >> 16))
}

// Silk_SMULTT computes (a32 >> 16) * (b32 >> 16). (macros.h:435)
func Silk_SMULTT(a32, b32 int32) int32 {
	return (a32 >> 16) * (b32 >> 16)
}

// Silk_SMLATT computes a32 + (b32 >> 16) * (c32 >> 16). (macros.h:438)
func Silk_SMLATT(a32, b32, c32 int32) int32 {
	return Silk_ADD32(a32, (b32>>16)*(c32>>16))
}

// Silk_SMLALBB computes a64 + (opus_int64)((opus_int16)(b16) * (opus_int16)(c16)).
// (macros.h:440)
func Silk_SMLALBB(a64 int64, b16, c16 int16) int64 {
	return Silk_ADD64(a64, int64(int32(b16)*int32(c16)))
}

// Silk_SMULL computes (opus_int64)(a32) * b32. (macros.h:443)
func Silk_SMULL(a32, b32 int32) int64 {
	return int64(a32) * int64(b32)
}

// Silk_SMMUL is a signed top-word multiply: SMULL(a32,b32) >> 32. (macros.h:608)
func Silk_SMMUL(a32, b32 int32) int32 {
	return int32(Silk_RSHIFT64(Silk_SMULL(a32, b32), 32))
}

// ------------------------------------------------------------------ //
// silk/macros.h: plain multiply / multiply-accumulate.                //
// ------------------------------------------------------------------ //

// Silk_MUL computes a32 * b32 (32-bit result). (macros.h:423)
func Silk_MUL(a32, b32 int32) int32 { return a32 * b32 }

// Silk_MUL_uint computes a32 * b32 as a 32-bit unsigned. (macros.h:426)
func Silk_MUL_uint(a32, b32 uint32) uint32 { return a32 * b32 }

// Silk_MLA computes a32 + (b32 * c32). (macros.h:429)
func Silk_MLA(a32, b32, c32 int32) int32 { return Silk_ADD32(a32, b32*c32) }

// Silk_MLA_uint computes a32 + (b32 * c32) as uint32. (macros.h:432)
func Silk_MLA_uint(a32, b32, c32 uint32) uint32 { return a32 + b32*c32 }

// Silk_MLA_ovflw computes a32 + b32*c32 allowing overflow. (macros.h:453)
func Silk_MLA_ovflw(a32, b32, c32 int32) int32 {
	return Silk_ADD32_ovflw(a32, int32(uint32(b32)*uint32(c32)))
}

// Silk_SMLABB_ovflw computes a32 + (opus_int16)(b32) * (opus_int16)(c32),
// allowing overflow. (macros.h:454)
func Silk_SMLABB_ovflw(a32, b32, c32 int32) int32 {
	return Silk_ADD32_ovflw(a32, int32(int16(b32))*int32(int16(c32)))
}

// ------------------------------------------------------------------ //
// silk/macros.h: add/subtract (plain and overflow-safe).              //
// ------------------------------------------------------------------ //

// Silk_ADD16 adds two 16-bit values. The C macro ((a)+(b)) has NO outer cast,
// so the int16 operands promote to int and the result is 32-bit
// (Silk_ADD16(-32768,-32768) == -65536, not 0). (macros.h:460)
func Silk_ADD16(a, b int16) int32 { return int32(a) + int32(b) }

// Silk_ADD32 adds two 32-bit values. (macros.h:461)
func Silk_ADD32(a, b int32) int32 { return a + b }

// Silk_ADD64 adds two 64-bit values. (macros.h:462)
func Silk_ADD64(a, b int64) int64 { return a + b }

// Silk_SUB16 subtracts two 16-bit values. Like Silk_ADD16 the C macro has no
// outer cast, so the result is 32-bit (Silk_SUB16(-32768,1) == -32769).
// (macros.h:464)
func Silk_SUB16(a, b int16) int32 { return int32(a) - int32(b) }

// Silk_SUB32 subtracts two 32-bit values. (macros.h:465)
func Silk_SUB32(a, b int32) int32 { return a - b }

// Silk_SUB64 subtracts two 64-bit values. (macros.h:466)
func Silk_SUB64(a, b int64) int64 { return a - b }

// Silk_ADD32_ovflw adds two signed 32-bit values allowing overflow. C routes
// through uint32 to dodge UB; Go's signed overflow is defined, so keeping the
// uint32 form documents intent without changing the result. (macros.h:447)
func Silk_ADD32_ovflw(a, b int32) int32 { return int32(uint32(a) + uint32(b)) }

// Silk_SUB32_ovflw subtracts two signed 32-bit values allowing overflow.
// (macros.h:450)
func Silk_SUB32_ovflw(a, b int32) int32 { return int32(uint32(a) - uint32(b)) }

// ------------------------------------------------------------------ //
// silk/macros.h: saturating add/subtract.                             //
// ------------------------------------------------------------------ //

// Silk_ADD_SAT16 adds two 16-bit values, saturating to the int16 range.
// (macros.h:479)
func Silk_ADD_SAT16(a, b int16) int16 {
	return int16(Silk_SAT16(Silk_ADD32(int32(a), int32(b))))
}

// Silk_SUB_SAT16 subtracts two 16-bit values, saturating to int16. (macros.h:484)
func Silk_SUB_SAT16(a, b int16) int16 {
	return int16(Silk_SAT16(Silk_SUB32(int32(a), int32(b))))
}

// Silk_ADD_SAT32 adds two 32-bit values, saturating at INT32_MIN/MAX. The sign
// tests mirror the C: the 0x80000000 masks are computed in uint32, matching the
// unsigned literal C uses. (macros.h:99)
func Silk_ADD_SAT32(a, b int32) int32 {
	if (uint32(a)+uint32(b))&0x80000000 == 0 {
		if (uint32(a)&uint32(b))&0x80000000 != 0 {
			return silk_int32_MIN
		}
		return a + b
	}
	if (uint32(a)|uint32(b))&0x80000000 == 0 {
		return silk_int32_MAX
	}
	return a + b
}

// Silk_SUB_SAT32 subtracts two 32-bit values, saturating at INT32_MIN/MAX.
// (macros.h:103)
func Silk_SUB_SAT32(a, b int32) int32 {
	if (uint32(a)-uint32(b))&0x80000000 == 0 {
		if uint32(a)&(uint32(b)^0x80000000)&0x80000000 != 0 {
			return silk_int32_MIN
		}
		return a - b
	}
	if (uint32(a)^0x80000000)&uint32(b)&0x80000000 != 0 {
		return silk_int32_MAX
	}
	return a - b
}

// Silk_ADD_SAT64 adds two 64-bit values, saturating at INT64_MIN/MAX.
// (macros.h:480)
func Silk_ADD_SAT64(a, b int64) int64 {
	if (a+b)&silk_int64_MIN == 0 {
		if (a&b)&silk_int64_MIN != 0 {
			return silk_int64_MIN
		}
		return a + b
	}
	if (a|b)&silk_int64_MIN == 0 {
		return silk_int64_MAX
	}
	return a + b
}

// Silk_SUB_SAT64 subtracts two 64-bit values, saturating at INT64_MIN/MAX.
// (macros.h:485)
func Silk_SUB_SAT64(a, b int64) int64 {
	if (a-b)&silk_int64_MIN == 0 {
		if a&(b^silk_int64_MIN)&silk_int64_MIN != 0 {
			return silk_int64_MIN
		}
		return a - b
	}
	if (a^silk_int64_MIN)&b&silk_int64_MIN != 0 {
		return silk_int64_MAX
	}
	return a - b
}

// ------------------------------------------------------------------ //
// silk/macros.h: shifts.                                              //
// ------------------------------------------------------------------ //

// Silk_LSHIFT8 shifts an 8-bit value left. C shifts UNSIGNED then casts back.
// (macros.h:497)
func Silk_LSHIFT8(a int8, shift int) int8 { return int8(uint8(a) << shift) }

// Silk_LSHIFT16 shifts a 16-bit value left. (macros.h:498)
func Silk_LSHIFT16(a int16, shift int) int16 { return int16(uint16(a) << shift) }

// Silk_LSHIFT32 shifts a 32-bit value left. (macros.h:499)
func Silk_LSHIFT32(a int32, shift int) int32 { return int32(uint32(a) << shift) }

// Silk_LSHIFT64 shifts a 64-bit value left. (macros.h:500)
func Silk_LSHIFT64(a int64, shift int) int64 { return int64(uint64(a) << shift) }

// Silk_LSHIFT is Silk_LSHIFT32. (macros.h:501)
func Silk_LSHIFT(a int32, shift int) int32 { return Silk_LSHIFT32(a, shift) }

// Silk_RSHIFT8 shifts an 8-bit value right (arithmetic). (macros.h:503)
func Silk_RSHIFT8(a int8, shift int) int8 { return a >> shift }

// Silk_RSHIFT16 shifts a 16-bit value right (arithmetic). (macros.h:504)
func Silk_RSHIFT16(a int16, shift int) int16 { return a >> shift }

// Silk_RSHIFT32 shifts a 32-bit value right (arithmetic). (macros.h:505)
func Silk_RSHIFT32(a int32, shift int) int32 { return a >> shift }

// Silk_RSHIFT64 shifts a 64-bit value right (arithmetic). (macros.h:506)
func Silk_RSHIFT64(a int64, shift int) int64 { return a >> shift }

// Silk_RSHIFT is Silk_RSHIFT32. (macros.h:507)
func Silk_RSHIFT(a int32, shift int) int32 { return Silk_RSHIFT32(a, shift) }

// Silk_LSHIFT_SAT32 limits a into the range that survives a left shift, then
// shifts. (macros.h:510)
func Silk_LSHIFT_SAT32(a int32, shift int) int32 {
	return Silk_LSHIFT32(
		Silk_LIMIT_32(a, Silk_RSHIFT32(silk_int32_MIN, shift), Silk_RSHIFT32(silk_int32_MAX, shift)),
		shift)
}

// Silk_LSHIFT_ovflw shifts a 32-bit value left, allowed to overflow.
// (macros.h:513)
func Silk_LSHIFT_ovflw(a int32, shift int) int32 { return int32(uint32(a) << shift) }

// Silk_LSHIFT_uint shifts an unsigned value left. (macros.h:514)
func Silk_LSHIFT_uint(a uint32, shift int) uint32 { return a << shift }

// Silk_RSHIFT_uint shifts an unsigned value right. (macros.h:515)
func Silk_RSHIFT_uint(a uint32, shift int) uint32 { return a >> shift }

// Silk_ADD_LSHIFT computes a + (b << shift). (macros.h:517)
func Silk_ADD_LSHIFT(a, b int32, shift int) int32 { return a + Silk_LSHIFT(b, shift) }

// Silk_ADD_LSHIFT32 computes a + (b << shift). (macros.h:518)
func Silk_ADD_LSHIFT32(a, b int32, shift int) int32 { return Silk_ADD32(a, Silk_LSHIFT32(b, shift)) }

// Silk_ADD_LSHIFT_uint computes a + (b << shift), unsigned. (macros.h:519)
func Silk_ADD_LSHIFT_uint(a, b uint32, shift int) uint32 { return a + Silk_LSHIFT_uint(b, shift) }

// Silk_ADD_RSHIFT computes a + (b >> shift). (macros.h:520)
func Silk_ADD_RSHIFT(a, b int32, shift int) int32 { return a + Silk_RSHIFT(b, shift) }

// Silk_ADD_RSHIFT32 computes a + (b >> shift). (macros.h:521)
func Silk_ADD_RSHIFT32(a, b int32, shift int) int32 { return Silk_ADD32(a, Silk_RSHIFT32(b, shift)) }

// Silk_ADD_RSHIFT_uint computes a + (b >> shift), unsigned. (macros.h:522)
func Silk_ADD_RSHIFT_uint(a, b uint32, shift int) uint32 { return a + Silk_RSHIFT_uint(b, shift) }

// Silk_SUB_LSHIFT32 computes a - (b << shift). (macros.h:523)
func Silk_SUB_LSHIFT32(a, b int32, shift int) int32 { return Silk_SUB32(a, Silk_LSHIFT32(b, shift)) }

// Silk_SUB_RSHIFT32 computes a - (b >> shift). (macros.h:524)
func Silk_SUB_RSHIFT32(a, b int32, shift int) int32 { return Silk_SUB32(a, Silk_RSHIFT32(b, shift)) }

// Silk_RSHIFT_ROUND shifts right with round-to-nearest. Requires shift > 0.
// (macros.h:527)
func Silk_RSHIFT_ROUND(a int32, shift int) int32 {
	if shift == 1 {
		return (a >> 1) + (a & 1)
	}
	return ((a >> (shift - 1)) + 1) >> 1
}

// Silk_RSHIFT_ROUND64 shifts a 64-bit value right with round-to-nearest.
// Requires shift > 0. (macros.h:528)
func Silk_RSHIFT_ROUND64(a int64, shift int) int64 {
	if shift == 1 {
		return (a >> 1) + (a & 1)
	}
	return ((a >> (shift - 1)) + 1) >> 1
}

// ------------------------------------------------------------------ //
// silk/macros.h: divide.                                              //
// ------------------------------------------------------------------ //

// Silk_DIV32_16 divides a 32-bit value by a 16-bit-valued divisor, truncating
// toward zero. (macros.h:456)
func Silk_DIV32_16(a32, b16 int32) int32 { return a32 / b16 }

// Silk_DIV32 divides two 32-bit values, truncating toward zero. (macros.h:457)
func Silk_DIV32(a32, b32 int32) int32 { return a32 / b32 }

// ------------------------------------------------------------------ //
// silk/macros.h: saturation to a narrower width.                      //
// ------------------------------------------------------------------ //

// Silk_SAT8 clamps a to the int8 range. (macros.h:468)
func Silk_SAT8(a int32) int32 {
	if a > int32(silk_int8_MAX) {
		return int32(silk_int8_MAX)
	}
	if a < int32(silk_int8_MIN) {
		return int32(silk_int8_MIN)
	}
	return a
}

// Silk_SAT16 clamps a to the int16 range. (macros.h:470)
func Silk_SAT16(a int32) int32 {
	if a > int32(silk_int16_MAX) {
		return int32(silk_int16_MAX)
	}
	if a < int32(silk_int16_MIN) {
		return int32(silk_int16_MIN)
	}
	return a
}

// Silk_SAT32 clamps a to the int32 range. For a 32-bit input this is the
// identity, matching the C macro (whose comparisons against INT32_MIN/MAX can
// never fire for an opus_int32). (macros.h:472)
func Silk_SAT32(a int32) int32 { return a }

// ------------------------------------------------------------------ //
// silk/macros.h: CLZ, rotate.                                         //
// ------------------------------------------------------------------ //

// ec_ilog is EC_ILOG: the number of significant bits (base-2 log + 1), defined
// as 0 for 0. On our targets the C macro resolves to a CLZ intrinsic; Go's
// bits.Len32 is the exact equivalent. (celt/ecintrin.h)
func ec_ilog(x uint32) int32 { return int32(bits.Len32(x)) }

// Silk_CLZ16 counts the leading zeros of a 16-bit value. The C ORs in 0x8000 so
// the result is bounded to [0,16]. (macros.h:113)
func Silk_CLZ16(in16 int16) int32 {
	return 32 - ec_ilog(uint32(int32(in16)<<16|0x8000))
}

// Silk_CLZ32 counts the leading zeros of a 32-bit value; 0 has 32. (macros.h:120)
func Silk_CLZ32(in32 int32) int32 {
	if in32 != 0 {
		return 32 - ec_ilog(uint32(in32))
	}
	return 32
}

// Silk_ROR32 rotates a32 right by rot bits; negative rot rotates left.
// (SigProc_FIX.h:394)
func Silk_ROR32(a32 int32, rot int) int32 {
	x := uint32(a32)
	r := uint32(rot)
	m := uint32(-rot)
	if rot == 0 {
		return a32
	} else if rot < 0 {
		return int32((x << m) | (x >> (32 - m)))
	}
	return int32((x << (32 - r)) | (x >> r))
}

// ------------------------------------------------------------------ //
// silk/macros.h: min/max, abs, sign, limit.                           //
// ------------------------------------------------------------------ //

// Silk_min_int returns the smaller of two ints. (macros.h:542)
func Silk_min_int(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Silk_min_16 returns the smaller of two int16 values. (macros.h:546)
func Silk_min_16(a, b int16) int16 {
	if a < b {
		return a
	}
	return b
}

// Silk_min_32 returns the smaller of two int32 values. (macros.h:550)
func Silk_min_32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

// Silk_min_64 returns the smaller of two int64 values. (macros.h:554)
func Silk_min_64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// Silk_max_int returns the larger of two ints. (macros.h:560)
func Silk_max_int(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Silk_max_16 returns the larger of two int16 values. (macros.h:564)
func Silk_max_16(a, b int16) int16 {
	if a > b {
		return a
	}
	return b
}

// Silk_max_32 returns the larger of two int32 values. (macros.h:568)
func Silk_max_32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// Silk_max_64 returns the larger of two int64 values. (macros.h:572)
func Silk_max_64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// Silk_LIMIT_32 clamps a between limit1 and limit2 in either order. (macros.h:577)
func Silk_LIMIT_32(a, limit1, limit2 int32) int32 {
	if limit1 > limit2 {
		if a > limit1 {
			return limit1
		}
		if a < limit2 {
			return limit2
		}
		return a
	}
	if a > limit2 {
		return limit2
	}
	if a < limit1 {
		return limit1
	}
	return a
}

// Silk_LIMIT_int clamps a between limit1 and limit2 (int form). (macros.h:580)
func Silk_LIMIT_int(a, limit1, limit2 int) int {
	if limit1 > limit2 {
		if a > limit1 {
			return limit1
		}
		if a < limit2 {
			return limit2
		}
		return a
	}
	if a > limit2 {
		return limit2
	}
	if a < limit1 {
		return limit1
	}
	return a
}

// Silk_LIMIT_16 clamps a between limit1 and limit2 (int16 form). (macros.h:581)
func Silk_LIMIT_16(a, limit1, limit2 int16) int16 {
	if limit1 > limit2 {
		if a > limit1 {
			return limit1
		}
		if a < limit2 {
			return limit2
		}
		return a
	}
	if a > limit2 {
		return limit2
	}
	if a < limit1 {
		return limit1
	}
	return a
}

// Silk_abs returns |a| for an int32. Wrong for a == INT32_MIN (the C macro
// documents this: -INT32_MIN wraps back to INT32_MIN). (macros.h:584)
func Silk_abs(a int32) int32 {
	if a > 0 {
		return a
	}
	return -a
}

// Silk_abs_int32 returns |a| branchlessly for an int32 (also wraps at INT32_MIN).
// (macros.h:586)
func Silk_abs_int32(a int32) int32 {
	return (a ^ (a >> 31)) - (a >> 31)
}

// Silk_abs_int64 returns |a| for an int64. (macros.h:587)
func Silk_abs_int64(a int64) int64 {
	if a > 0 {
		return a
	}
	return -a
}

// Silk_sign returns the sign of a as 1, -1 or 0. (macros.h:589)
func Silk_sign(a int32) int32 {
	if a > 0 {
		return 1
	}
	if a < 0 {
		return -1
	}
	return 0
}

// ------------------------------------------------------------------ //
// silk/macros.h: pseudo-random generator.                             //
// ------------------------------------------------------------------ //

const (
	RAND_MULTIPLIER = 196314165
	RAND_INCREMENT  = 907633515
)

// Silk_RAND advances the LCG seed. Store the result as the next seed.
// (macros.h:597)
func Silk_RAND(seed int32) int32 {
	return Silk_MLA_ovflw(RAND_INCREMENT, seed, RAND_MULTIPLIER)
}
