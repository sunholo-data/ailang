// Package fixedmath holds the CELT fixed-point macro helpers (celt/fixed_generic.h,
// mathops.h) as exact Go expressions, with exhaustive tests against the C
// semantics. It uses the OPUS_FAST_INT64 forms. Each helper reproduces its C
// macro bit-for-bit: that arithmetic is the contract and must not be
// "simplified", even though the surrounding Go may be idiomatic.
package fixedmath
