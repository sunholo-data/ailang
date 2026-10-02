### Fixed — full-width hex/binary/octal int literals; out-of-range pattern literals no longer silently clamp (#1481) (2026-10-02)

A radix literal with the top bit set, such as SplitMix64's `0x9e3779b97f4a7c15`, was a parse
error, so 64-bit hash constants had to be hand-converted to signed decimals. Radix literals
(`0x`, `0b`, `0o`) are now 64-bit patterns. Values in [2^63, 2^64−1] read as their
two's-complement `int`, like Go's `int64(uint64(x))`: `0x9e3779b97f4a7c15 == -7046029254386353131`
and `0xffffffffffffffff == -1`. Every literal that parsed before keeps its value. Decimal literals
stay signed, and one above `9223372036854775807` is still an error.

- The same literal in a **match pattern** used to compile silently to a pattern matching
  `9223372036854775807`, because the range error was discarded. An overflowing float pattern
  (`1e999`) silently became `+Inf`. Pattern literals now get the wrapped value, or an error.
- New positioned diagnostic **PAR021** for an int/float literal that does not fit. It replaces the
  bare, unpositioned `could not parse "…" as integer`, and its suggestion gives the hex form of
  the intended bit pattern.
- `ailang fmt` prints a negative literal as its hex pattern. The decimal form re-parses as unary
  minus, which failed fmt's round-trip check, and `-9223372036854775808` does not parse at all.
- The SMT contract encoder emits `(- 9223372036854775808)` for the minimum int, where it used to
  emit a malformed `(- -9223372036854775808)`.

### Added — `std/math.shiftRightLogical` (logical right shift, the reference `>>>`) (#1481) (2026-10-02)

`>>` is arithmetic (sign-extending): `-1 >> 1 == -1`. `shiftRightLogical(x, n)` shifts the 64-bit
pattern in zeros: `shiftRightLogical(-1, 1) == 9223372036854775807`. A count of 64 or more gives
`0`, and a negative count is `RT_SHIFT`, as for `<<` and `>>`. It is a pure builtin
(`shiftRightLogical_Int`) on the interpreter, the strict VM and `--emit-go`. There is no new
operator; `>>>` is deferred per the design doc. The teaching prompt now teaches both features, and
`examples/runnable/splitmix64.ail` writes SplitMix64 exactly like the reference implementation.
Design: `design_docs/implemented/v0_51_1/m-fullwidth-int-literals-and-logical-shift.md`.
