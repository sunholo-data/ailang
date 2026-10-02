# M-FULLWIDTH-INT-LITERALS-AND-LOGICAL-SHIFT: Hex/binary/octal int literals above 2^63−1 do not parse (u64 hash constants unwritable), pattern literals silently misparse, and there is no logical right shift

**Status**: Implemented (2026-10-02, #1481 — see Implementation Report)
**Target**: v0.51.2
**Priority**: P1 (High) — blocks every port of 64-bit hash/PRNG reference code (SplitMix64, FNV-1a, murmur3, xxHash) at the *constant* level, and the arithmetic-vs-logical shift distinction is a classic silent-corruption hazard for exactly that class of code. Not P0: nothing already written is corrupted (the loud parse error is honest, and the silent pattern-position misparse is reachable but has not been observed in a real consumer); the workaround exists but is error-prone (see Problem Statement).
**Estimated**: 2 days (one short sprint: ~55 LOC Go + ~10 LOC stdlib + tests/docs/prompt)
**Dependencies**: None. Both halves are self-contained; Part 1 (literals) and Part 2 (logical shift) land independently if needed.

**Reported from**: coordinator task `task-685c1642` (AILANG v0.51.0 b99dd25, darwin arm64), external benchmark consumer stapledons-godot `sim/rng.ail` (SplitMix64 constants carried as `0 - 7046029254386353131` with the hex in a comment). All in-container probes below were run with `ailang` **v0.51.0 @ b99dd25** (the same commit as the report binary) on linux — the literal-parse, pattern-misparse, wrap, and shift-semantics behavior is not architecture-sensitive (identical `int64` semantics in Go on both).

---

## Problem Statement

**Current State — three manifestations of one root gap, plus a missing primitive:**

1. **Full-width base-prefixed int literals are a parse error (the reported repro).**
   ```ailang
   module h
   export func f(n: int) -> int = 0x9e3779b97f4a7c15 + n
   ```
   ```
   $ ailang run --quiet --entry f --args-json 1 h.ail
   Error: ... parse errors in h.ail: could not parse "0x9e3779b97f4a7c15" as integer
   ```
   `0x1e3779b97f4a7c15` and `0x7fffffffffffffff` parse fine (live-verified, probe P1). Root cause, read from the code: `parseIntLiteralValue` ([internal/parser/parser_literals.go:23](../../../internal/parser/parser_literals.go)) routes base-prefixed literals through Go's **signed** `strconv.ParseInt(s, 0, 64)`, which rejects any value above `MaxInt64`. The reporter's expected semantics — a 16-hex-digit literal is read as the 64-bit pattern, wrapping to the two's-complement value "like Go's uint64→int64 conversion" — is exactly what every reference implementation of 64-bit hashers relies on when it writes `0x9E3779B97F4A7C15`.

2. **The same literal in *pattern* position is a silent wrong match (worse than the report, found by the systemic audit this skill mandates).** `literalValue()` ([internal/parser/parser_pattern.go:371](../../../internal/parser/parser_pattern.go)) calls the same helper but **discards the error** (`v, _ := parseIntLiteralValue(...)`). Go's `ParseInt` returns the clamped `MaxInt64` on range error, so today:
   ```ailang
   export func f(x: int) -> int =
     match x { 0x9e3779b97f4a7c15 => 1, _ => 3 }
   ```
   silently compiles to a pattern that matches **`9223372036854775807`** — not the intended constant and not zero. Live-verified: `f(9223372036854775807)` returns `1`, `f(-7046029254386353131)` returns `3` (probe P2). No diagnostic is emitted. This is a direct **no-silent-fallbacks** violation in security-adjacent parsing code. The float sibling is also broken: `1e999` in a pattern position silently becomes a `+Inf` pattern that *matches* `inf` (probe P3, same discarded-error line).

3. **The diagnostic itself is sub-par when it does fire.** The expression-position error is a bare `fmt.Errorf("could not parse %q as integer")` with **no line/column, no PAR code, no suggestion** (see repro output), while sibling parse diagnostics (e.g. `PAR016`, `PAR020`) are positioned, coded, and actionable. A model hitting this wall gets no pointer to the workaround.

4. **No logical (unsigned) right shift exists.** `>>` is arithmetic/sign-extending (verified live on both backends: `-1 >> 1 = -1`, probe P5; confirmed in code: `shiftRightImpl` shifts Go `int64`). Reference hash/PRNG code needs `>>>` in every mixer (`z ^ (z >>> k)`); models must emulate it, and the emulation is a *measured* mutation hazard — the m-pure-prng quorum round 1 rejected an `lsr` mask helper as off-by-one (`(1<<(63-k))-1` needed `(1<<(64-k))-1`), and the reporter's own emulation `(x >> k) & ((1 << (64 - k)) - 1)` is exactly the form that class of review flags. `std/math` exports `bitwiseOr` but **no shift function at all**; the internal `shiftRight_Int` builtin is arithmetic-only (verified: `ailang builtins list` shows `shiftRight_Int` under std/math and no logical variant, probe P7).

**Impact:** AI-generated code porting any 64-bit hash/PRNG/mixer from reference material (SplitMix64, FNV-1a-64, murmur3, xxHash, wyhash — the staple mixers of simulation and hashing benchmarks) hits a parse wall at the constants, then a semantic wall at `>>>`. The stapledons-godot consumer carries hand-converted signed decimals with the hex in comments — a value the *reader* must re-verify — and the m-pure-prng design doc (parked) had to ship the same workaround plus a quorum-hardened `ushr` helper. This is the same lane as v0.26.0's hex/binary/octal literal support, which was adopted precisely because "supporting what models naturally write removes the wall" (a model then burnt its step budget `sed`-converting `0xE000` to decimal — [changelogs/v0.26-v0.31-measurement-cost.md](../../../changelogs/v0.26-v0.31-measurement-cost.md)). The remaining wall is the top bit.

---

## Goals

**Primary Goal:** Every 64-bit integer bit pattern is writable in AILANG in the exact form reference implementations use (`0x9e3779b97f4a7c15`), and logical right shift is a one-call primitive on both backends.

**Success Metrics:**
- The report's repro `0x9e3779b97f4a7c15 + n` runs with entry-arg `1` → `-7046029254386353130`, identical on the interpreter and `--strict-bytecode`.
- All three SplitMix64 constants (top bit set on all) parse in source form.
- Pattern-position out-of-range literals produce a positioned `PAR021` error instead of a silent `MaxInt64`/`+Inf` match; pattern `0x9e3779b97f4a7c15` matches exactly `-7046029254386353131`.
- `shiftRightLogical(x, n)` from std/math gives the m-pure-prng §6 known-answer vectors (`ushr(-1,1)=9223372036854775807`, `ushr(-1,63)=1`, `ushr(minInt,63)=1`, `ushr(minInt,1)=4611686018427387904`) on both backends.
- Zero behavior change for every literal that parses today (in-range hex/binary/octal, all decimals, leading-zero decimals stay decimal).

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Base-prefixed literals ≥ 2^63 wrap to the two's-complement int64 (bit-pattern reading); **decimal literals stay signed** and error above `MaxInt64` | Fixes the constant wall without silently reinterpreting magnitude-intuitive decimals (`18446744073709551615` → `-1` would be a trap); matches Go's model (hex constants wrap on int64 conversion; decimal overflow errors) and C/C++ (hex adopts unsigned when ≥ 2^63) | human | design | med |
| Pattern-position literal errors propagate (`PAR021`), ending the silent `MaxInt64`/`+Inf` match | Silent wrong match in a pattern is the worst failure mode AILANG permits; could alternatively keep clamping (rejected: no-silent-fallbacks) | human | design | low |
| Logical shift ships as a **builtin + std/math function** (`shiftRightLogical_Int` / `shiftRightLogical`), **not** a `>>>` operator | Keeps the change in the AILANG-fix/stdlib lane per PROGRAM.md default-bias (core lexer/parser/CoreOp enum stay frozen); zero new syntax (A8) | human | design | med |
| New diagnostic code `PAR021` for unparseable integer literals | Shared namespace; must be verified unallocated (done — see Verification Log) and reserved once | agent | design | low |
| `0x8000000000000000` (min-int pattern) must also parse — and the SMT contract encoder's negative-literal path must be fixed in the same sprint | MinInt64 makes the latent `(- %d)` negation in `internal/smt/codegen_expr.go` overflow; reachable only once negative IntLit constants exist | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Wrap semantics apply only to base-prefixed literals (0x/0X/0b/0B/0o/0O); decimals remain signed-parse.
- [x] Pattern literals share the expression-path semantics and error channel (no clamping).
- [x] Logical shift = builtin + std/math export this sprint; `>>>` operator is deferred with an explicit demand-evidence gate (Non-Goals).
- [x] `PAR021` is the diagnostic code (verified unallocated at design time).

---

## Solution Design

### Overview

One root gap (out-of-range int literals) currently has three divergent behaviors: loud unpositioned error (expression), silent `MaxInt64` match (pattern, int), silent `+Inf` match (pattern, float). The unified fix routes **all** integer-literal conversion through one helper with wrap semantics for base-prefixed forms, propagates errors in pattern position, and upgrades the failure to a positioned, coded, actionable `PAR021`. The logical right shift is added as a pure builtin — `int64(uint64(a) >> n)` — registered exactly like its five bitwise siblings, which makes it automatically available to the tree-walking evaluator, the strict VM (registry-driven adapter), `ailang builtins list`, and generated Go, plus a friendly `std/math` export and a teaching-prompt line (the established `bitwiseOr` pattern).

### Architecture

**Part 1 — full-width literals (parser only; everything downstream already carries int64 verbatim):**

1. `parseIntLiteralValue` (internal/parser/parser_literals.go): for base-prefixed literals, replace `strconv.ParseInt(s, 0, 64)` with `strconv.ParseUint(s, 0, 64)` + `int64(u)` reinterpretation. Every literal that parses today parses to the **same** value tomorrow (ParseUint and ParseInt agree on the in-range prefix space — the only newly-accepted values are those in [2^63, 2^64−1]). Decimals keep `strconv.ParseInt(s, 10, 64)`; leading-zero decimals stay decimal (existing rule, unchanged).
2. `parseIntegerLiteral` (same file): on error, emit `NewSuggestionError("PAR021", pos, token, …)` with line/column and suggestions:
   - decimal above `MaxInt64` → suggest the hex form of the intended bit pattern **and** the two's-complement decimal (both machine-computed in the message: for `9223372036854775808` → "use `0x8000000000000000` or `-9223372036854775808`");
   - base-prefixed above `2^64−1` → "does not fit in 64 bits; the largest writable pattern is `0xFFFFFFFFFFFFFFFF`".
3. `literalValue` (internal/parser/parser_pattern.go): return `(interface{}, error)`, propagate both the int and float errors as the same `PAR021` diagnostic (single call site: `parseBasePattern`, parser_pattern.go:73 — verified the only caller). After the helper change, `0x9e3779b97f4a7c15` in a pattern already parses **correctly** (the wrap is shared); the error propagation closes the residual silent-clamp cases (literals > 2^64−1 — `ParseUint` clamps to `MaxUint64`, i.e. a silent `-1` pattern, if the error were still discarded — and float overflow).

**Part 2 — logical right shift (builtin lane, mirrors the five existing bitwise builtins):**

1. `internal/builtins/math_bitwise.go`: register `shiftRightLogical_Int` (arity 2, pure) with impl `int64(uint64(a) >> uint(b))`; negative shift amount reuses the existing `RT_SHIFT` runtime error (same contract as the arithmetic shift); shift count ≥ 64 yields 0 (Go unsigned-shift semantics, matching every uint64 reference).
2. `registerBuiltinWithMeta` auto-derives the type (`Int -> Int -> Int`), module attribution (`std/math`), and docs metadata — the same one-call registration the bitwise siblings use. The strict-VM adapter picks pure registry builtins up automatically (`AdaptedBuiltinNames` is registry-derived; the #1447/#1450 coverage tests enforce three-way coverage — verified mechanism, Verification Log V9/V10). **No** core-op, lexer, or iface-freeze change: bitwise operator builtins are not in `FrozenBuiltinInterface` today (verified V12), so no digest churn.
3. `std/math.ail`: `export pure func shiftRightLogical(x: int, n: int) -> int = shiftRightLogical_Int(x, n)` with a comment block teaching the `>>>`↔`shiftRightLogical` mapping and the `>>`-is-arithmetic warning (the `bitwiseOr` precedent: prompt and std agree; the `intToFloat`/`_int_to_float` precedent for calling a registered builtin from stdlib).
4. Teaching prompt (`ailang prompt`, mirrored at docs/docs/prompts/current.md): extend the bitwise section — "`>>` is arithmetic (sign-extending); logical shift is `shiftRightLogical(x, n)` from std/math (the `>>>` of reference hash code). Full-width hex literals (`0x9e3779b97f4a7c15`) parse as the two's-complement bit pattern."
5. `examples/integer_literals.ail`: add the full-width constants + a min-int case (this example currently demonstrates hex only in comments/decimals — it becomes the living fixture); `examples/runnable/` gains a SplitMix64 runnable using the natural constants and `shiftRightLogical`, bit-checked against the m-pure-prng known-answer vectors (seed 0/42), gated by `make verify-examples`.

**Out-of-scope but pre-audited for the future `>>>` option** (the full file list, so a follow-up doc does not re-derive it): lexer token (`internal/lexer/token.go`, `lexer.go` — `>>>` is lexically unambiguous: no valid program today has three adjacent `>`), parser infix registration + precedence, `core.IntrinsicOp` + `String` maps (internal/core/core.go), elaborate mapping, `op_table.go` OperatorTable + signature table + `CreateTypeMismatchError` opStr, `eval_operations.go` case, `format/precedence.go` precShift, SMT `bvlshr` mapping (internal/smt/codegen_operators.go — the slot adjacent to the existing `bvashr`), editors grammar (no `>>` token in syntaxes/ailang.tmLanguage.json today — verified V10).

### Implementation Plan

**Phase 1: Literal semantics + diagnostics** (~4 hours)
- [ ] `parseIntLiteralValue` → ParseUint wrap for prefixed literals (parser_literals.go, ~8 LOC + comment)
- [ ] `PAR021` suggestion error in `parseIntegerLiteral` (position/code/suggestions; ~15 LOC)
- [ ] `literalValue` error propagation in patterns, int + float (parser_pattern.go, ~10 LOC)
- [ ] Unit tests: extend `TestParseIntLiteralValue` (all 16-digit top-bit hex, `0x8000000000000000` → min-int, `0xFFFFFFFFFFFFFFFF` → −1, `0b`/`0o` full-width, in-range unchanged, leading-zero decimal unchanged); new pattern tests (top-bit hex pattern matches the wrapped value; out-of-range → `PAR021` with position; float `1e999` pattern → `PAR021`)

**Phase 2: Logical shift builtin** (~3 hours)
- [ ] Register + implement `shiftRightLogical_Int` (math_bitwise.go, ~15 LOC)
- [ ] std/math export + doc comment (std/math.ail, ~10 LOC)
- [ ] Unit tests mirroring math_bitwise_test.go: known-answer vectors, negative shift → `RT_SHIFT`, shift ≥ 64 → 0, strict-VM adapter coverage (`TestOperatorBuiltinsAreAdapted` list extended)

**Phase 3: Latent-negative-constant fixes, docs, prompt** (~4 hours)
- [ ] Fix `encodeLit` MinInt64 path in internal/smt/codegen_expr.go (`(- %d)` negation wraps for `v == math.MinInt64`; emit the SMT-LIB arbitrary-precision form `(- 9223372036854775808)` via a `uint64(-v)` magnitude) + unit test with a min-int contract expression
- [ ] Extend `examples/integer_literals.ail`; new runnable SplitMix64 example; `make verify-examples` + `make verify-examples-toplevel` green
- [ ] Prompt + docs updates (bitwiseOr precedent); `ailang check`-verify every example before committing docs claims

### Files to Modify/Create

**New files:**
- `examples/runnable/splitmix64.ail` (~60 LOC) — SplitMix64 with natural hex constants + `shiftRightLogical`, printed values pinned to the m-pure-prng known-answer vectors

**Modified files:**
- `internal/parser/parser_literals.go` (+~20/−4 LOC) — ParseUint wrap + PAR021 diagnostic
- `internal/parser/parser_pattern.go` (+~10/−3 LOC) — `literalValue` error propagation
- `internal/builtins/math_bitwise.go` (+~15 LOC) — `shiftRightLogical_Int`
- `std/math.ail` (+~10 LOC) — `shiftRightLogical` export
- `internal/smt/codegen_expr.go` (+~6/−1 LOC) — MinInt64-safe literal encoding
- `internal/parser/parser_literals_test.go`, `internal/builtins/math_bitwise_test.go`, `internal/smt` lit-encoding test (+~120 LOC tests)
- `examples/integer_literals.ail` (+~10 LOC); prompt (`docs/docs/prompts/current.md` + prompt version flow) (+~6 LOC)

---

## Examples

### Example 1: The report's repro, before/after

**Before (v0.51.0, live):** parse error; workaround requires hand-conversion the reader must verify:
```ailang
-- GOLDEN = 0x9E3779B97F4A7C15 (SplitMix64 golden gamma)
export func f(n: int) -> int = 0 - 7046029254386353131 + n   -- hex in comment only
```
**After:** the reference form, identical bits:
```ailang
export func f(n: int) -> int = 0x9e3779b97f4a7c15 + n
```
`ailang run --quiet --entry f --args-json 1` → `-7046029254386353130` (machine-computed: `int64(0x9e3779b97f4a7c15) + 1`), same on `--strict-bytecode`.

### Example 2: SplitMix64 mixer, before/after

**Before** (the stapledons-godot / m-pure-prng form — three hand-converted decimals + a mask/step emulation that survived a two-reviewer quorum to get right):
```ailang
pure func ushr(x: int, k: int) -> int = ((x >> 1) & 9223372036854775807) >> (k - 1)
pure func mix(s: int) -> int = {
  let z1 = (s ^ ushr(s, 30)) * -4658895280553007687;
  let z2 = (z1 ^ ushr(z1, 27)) * -7723592293110705685;
  z2 ^ ushr(z2, 31)
}
```
**After:**
```ailang
import std/math (shiftRightLogical)
pure func mix(s: int) -> int = {
  let z1 = (s ^ shiftRightLogical(s, 30)) * 0xbf58476d1ce4e5b9;
  let z2 = (z1 ^ shiftRightLogical(z1, 27)) * 0x94d049bb133111eb;
  z2 ^ shiftRightLogical(z2, 31)
}
```
— line-for-line the reference algorithm; the known-answer vectors (seed 0 → `-2152535657050944081`, seed 42 → `-4767286540954276203`, …) still hold, so m-pure-prng's cross-language bit-exactness proof carries over unchanged.

### Example 3: Pattern position, before/after

**Before (live, probe P2):** `match x { 0x9e3779b97f4a7c15 => 1, _ => 3 }` silently matches `9223372036854775807`.
**After:** matches exactly `-7046029254386353131`; `0x1fffffffffffffffff` (17 hex digits) is a positioned `PAR021` error instead of a silent `-1` pattern.

---

## Success Criteria

- [ ] Report repro parses and evaluates to `-7046029254386353130` on interpreter and `--strict-bytecode`
- [ ] All in-scope constants parse: `0x9e3779b97f4a7c15`, `0xbf58476d1ce4e5b9`, `0x94d049bb133111eb`, `0xcbf29ce484222325` (FNV-1a offset), `0x8000000000000000`, `0xffffffffffffffff`; binary/octal full-width equivalents
- [ ] In-range literals byte-identical: `0xFF`→255, `0b1010`→10, `0o755`→493, `0123`→123 (existing test cases extended, not replaced)
- [ ] Pattern `0x9e3779b97f4a7c15` matches `-7046029254386353131` only; out-of-range int and overflow float patterns fail with positioned `PAR021`
- [ ] `shiftRightLogical` known answers on both backends: `(-1,1)→9223372036854775807`, `(-1,63)→1`, `(minInt,63)→1`, `(minInt,1)→4611686018427387904`; `RT_SHIFT` on negative; `0` on count ≥ 64
- [ ] SMT `encodeLit` round-trips `math.MinInt64` (contract over `0x8000000000000000` type-checks via Z3 path without malformed S-expression)
- [ ] `make test-core`, `make verify-examples`, `make verify-examples-toplevel` pass; prompt and std/math docs updated in the same change

## Testing Strategy

**Unit tests:** `TestParseIntLiteralValue` extension (wrap table incl. MinInt64/MaxUint64 boundaries, in-range identity, leading-zero decimal regression); pattern-literal tests (correct match value, `PAR021` on > 2^64−1 and float overflow); `shiftRightLogicalImpl` (vectors, `RT_SHIFT`, ≥64); SMT `encodeLit` MinInt64.

**Integration tests:** SplitMix64 runnable example (known answers, both backends); the strict-VM adapter coverage test extended to `_shiftRightLogical_Int`; a fmt round-trip check pinning that `ailang fmt` canonicalizes `0x9e3779b97f4a7c15` → `-7046029254386353131` (existing numeric canonicalization — no source-text echo — so the round-trip is stable).

**Regression-surface tests** (one per Conflict Surface "MUST still work" fixture): `examples/integer_literals.ail`, `examples/runnable/bitwise.ail`, `tests/binops_int.ail`, the m-pure-prng §3.3 `ushr` construction (unchanged stdlib behavior), and the `0123`-stays-decimal case.

**Manual:** `ailang run --strict-bytecode` on the SplitMix64 example; `ailang builtins list | grep shiftRightLogical`; `ailang prompt` shows the new teaching lines.

## Conflict Surface

### Syntactic positions touched

- `lexer.INT` tokens in **expression** position: `parseIntegerLiteral` (prefix fn, parser.go:83). Base-prefixed and decimal.
- `lexer.INT`/`lexer.FLOAT` tokens in **pattern** position: `parseBasePattern` → `literalValue` (parser_pattern.go:67-75, 371-389).
- No lexer change: `readNumber` already emits any-length radix digit runs as one INT token (verified reading lexer.go readNumber; the 16-digit hex token exists today — it is the *parser* that rejects it).
- Builtin registry surface (Part 2): a new pure builtin name in the global registry — additive only.

### What else lives here

| Position | Existing valid form | Interaction |
|----------|--------------------|-------------|
| INT in expression | in-range hex/binary/octal, decimals incl. leading-zero decimals, interpolation holes (`"${show(0xE000)}"`), guard expressions, contract expressions (`requires`/`ensures` — parsed by the same parser) | ParseUint path returns **identical** values for everything in-range; only [2^63, 2^64−1] newly parses. Contract expressions gain min-int literals → SMT `encodeLit` fix required (folded into Phase 3) |
| INT in pattern | int literal match arms (incl. `0xFF` today), char/string/bool literals, guards | Same helper now; correct wrapped value instead of silent clamp; errors propagate identically to expression position |
| FLOAT in pattern | float literal arms; `1e999` was silent `+Inf` | Loud `PAR021` — previously "valid" only in the sense that it silently miscompiled; treat any complaint as a bug report, not a regression |
| Downstream IntLit consumers | bytecode constant pool stores raw `int64` (`bytecode.NewInt`, Value.TagInt); generated Go prints `int64(%d)`; fmt canonicalizes to signed decimal; show prints `%d` | All verified negative-safe by reading (V11); the only latent non-negative assumption found is SMT `encodeLit`'s `(- %d)` (V5), fixed in-scope |

### Disambiguation strategy

No grammar ambiguity is introduced: the literal token shape is unchanged; only the value-conversion function and its error channel change. Part 2 adds a function name, no syntax.

### Programs that MUST still work

1. `examples/integer_literals.ail` — radix prefixes + leading-zero decimals (verify-examples-toplevel gate)
2. `examples/runnable/bitwise.ail` — `& ^ << >> ~` operator surface, incl. `255 >> 4 & 15`
3. `tests/binops_int.ail` — integer arithmetic/wrap corpus
4. `std/math.ail` `bitwiseOr` (De Morgan form) and the m-pure-prng §3.3 `ushr` construction if parked-unparked — both remain valid
5. `internal/parser/parser_literals_test.go::TestParseIntLiteralValue` and `internal/lexer/lexer_test.go::TestHexBinOctalLiterals` — the v0.26.0 regression cases

### What deliberately changes

- Hex/binary/octal literals in [2^63, 2^64−1]: parse error → valid, value = two's-complement int64. **Intended — this is the feature.**
- Pattern literals that today silently clamp (`MaxInt64` int / `MaxUint64`→`-1` for > 2^64−1 / `+Inf` float): now either parse **correctly** (wrap range) or error loudly (`PAR021`). Any program that "worked" only via the clamp was miscompiled; if a real consumer surfaces, the fix is to write the intended constant.
- No previously-valid program changes meaning (in-range values identical by construction of the ParseUint path).

## Deferred Decisions

- Exact `PAR021` message wording and suggestion phrasing — agent may choose (keep the machine-computed replacement constant in the suggestion).
- Whether `shiftRightLogical` also gets an `ushr` alias (m-pure-prng's name) — agent may choose; if added, both are documented as the same builtin.
- Test file organization (extend existing vs. new files) — agent may choose.
- Whether the SplitMix64 example lands in `examples/runnable/` or `examples/` top-level — agent may choose (verify-examples vs verify-examples-toplevel gate).
- Prompt version bump mechanics — agent may choose per the prompt-manager flow.

## Non-Goals

- **A `>>>` operator** — deferred, not rejected. Evidence gate: adopt when a benchmark/eval shows models fail to map reference `>>>` to `shiftRightLogical` *despite* the prompt line (the `bitwiseOr` precedent shows prompt-teaching works), or when a second high-frequency consumer asks. The full pre-audited file list is in Solution Design so the follow-up is a short doc, not a re-derivation.
- **Unsigned/uint64 type, wrapping decimal literals, literal type suffixes (`u64`)** — out of scope; AILANG `int` is signed two's-complement and stays so (the m-pure-prng value-fork debate settled that signed int + bit-pattern notation is sufficient).
- **`std/prng` itself** — that is [m-pure-prng](../../planned/v0_29_0/m-pure-prng.md) (parked, awaiting a `split` scope decision); this doc only removes the two language-level frictions it worked around. If it unships, its `ushr` helper and signed-decimal constants can be simplified to the natural forms, but that is its sprint's call.
- **Float literal policy** — only the silent pattern clamp is fixed; float semantics (print, NaN, etc.) untouched.

## Timeline

**Day 1 (7 hours):** Phase 1 literals + tests (4h), Phase 2 builtin + tests (3h)
**Day 2 (6 hours):** Phase 3 SMT fix, examples, prompt/docs, full gates (`make test-core`, verify-examples, fmt round-trip), buffer
**Total: ~13 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A consumer relied on the silent pattern clamp | Med | Loud error with the intended constant in the suggestion; migration is mechanical; clamp was a miscompile, not a contract |
| Negative IntLit constants hit an un-audited consumer (traces, LSP, observatory) | Med | Phase 3 audit sweep `grep -rn "IntLit"` consumers; all found are raw-int64 (V11); SplitMix64 example exercises the full pipeline on both backends |
| `ParseUint` clamp value (`MaxUint64`) leaks through a discarded error somewhere | Low | `literalValue` signature change makes the error channel structural; new unit tests pin both discard sites |
| Digest/golden churn from the builtin addition | Low | Bitwise builtins are outside `FrozenBuiltinInterface` (V12); no freeze-table edit, no digest update |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Bit-exact portability of reference constants and `>>>` — same bits on interpreter, strict VM, and peer languages (m-pure-prng's bit-for-bit Python proof carries over) |
| A2: Replayability | 0 | No trace/replay change |
| A3: Effect Legibility | 0 | Pure builtin, no effect surface |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | Constants become Z3-representable contract subjects (with the MinInt64 encoder fix); logical shift maps to existing `bvlshr` slot |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Models write the exact reference form instead of hand-converting decimals (a step-budget wall, v0.26.0 precedent) and stop re-deriving off-by-one mask emulations (a quorum-measured hazard) |
| A8: Minimal Syntax | +1 | Zero new syntax (builtin + stdlib function only; `>>>` deferred) |
| A9: Cost Visibility | 0 | No resource change |
| A10: Composability | +1 | Reuses the literal token, the builtin registry, the std/math export pattern, and the prompt-teaching lane |
| A11: Structured Failure | +1 | Silent pattern misparse → positioned coded `PAR021` with machine-computed suggestion; bare unpositioned error → structured diagnostic |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +6** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): no nondeterminism introduced (wrap is defined two's-complement on both backends — verified live)
- [x] A3 (Effects): pure builtin only
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): removes walls models measurably hit; adds no human-only convenience

## Verification Log

All probes run 2026-10-01 at HEAD with `ailang` v0.51.0 (commit `b99dd25`, the same commit as the report binary) on linux. Language claims are live `ailang check`/`run` results, not assumptions.

**P1 — Repro confirmed (expression position):** `0x9e3779b97f4a7c15` → `could not parse "0x9e3779b97f4a7c15" as integer`, **no line/col, no code, no suggestion** (the bare-error claim in Problem Statement ¶3 is from this same output). `0x1e3779b97f4a7c15 + 1` → `2177342782468422678`; `0x7fffffffffffffff + 1` → `-9223372036854775808` (in-range parses; + wraps).

**P2 — Pattern silent misparse (int):** with `f(x) = match x { 0x9e3779b97f4a7c15 => 1, _ => 3 }`: `f(0)`→`2` (arm did **not** match 0), `f(-7046029254386353131)`→`3`, `f(9223372036854775807)`→`1` — the arm silently matches `MaxInt64`. No diagnostic. Cause read from code: `literalValue()` at parser_pattern.go:374 `v, _ := parseIntLiteralValue(...)`; Go's `ParseInt` returns the `MaxInt64` clamp with `ErrRange`.

**P3 — Pattern silent misparse (float):** `match x { 1e999 => 1, _ => 3 }` with `x = 1.0 * 1e308 * 10.0` (inf) → `1`. Same discarded error at parser_pattern.go:381 (`strconv.ParseFloat` returns `+Inf` on overflow). In expression position `1e999` errors loudly — the divergence proves the pattern path's error channel, not the lexer, is the bug.

**P4 — Boundary literals today:** `9223372036854775808` (decimal > MaxInt64) → parse error; `0x8000000000000000` → parse error (both probed).

**P5 — Backend agreement (report premise):** `-7046029254386353131 * 3` → `-2691343689449507777`, `1 << 63` → `-9223372036854775808`, `-1 >> 1` → `-1` — **identical** on `ailang run` and `ailang run --strict-bytecode`. `>>` arithmetic confirmed on both.

**P6 — Root cause code-read:** `parseIntLiteralValue` (parser_literals.go:20-31) routes `0x`/`0b`/`0o` through `strconv.ParseInt(s, 0, 64)` (signed). The lexer (`readNumber`, lexer.go:324+) already tokenizes arbitrary-length radix runs as one INT token — no lexer change needed for the fix.

**P7 — `shiftRight_Int` callable, no logical sibling:** `export func f(a: int, b: int) -> int = shiftRight_Int(a, b)` type-checks clean; `ailang builtins list | grep shift` → `shiftLeft_Int`, `shiftRight_Int` (both `[pure] std/math`), no logical variant. Direct `_shiftRight_Int(a, b)` → undefined (underscore convention differs — the std wrapper must use the registry name).

**V8 — Negative-existence: no logical-shift primitive anywhere.** `grep -rn "shiftRightLogical|ushr|>>>" internal/ std/ examples/` → no non-test hits (the `>>>` in docs is m-pure-prng pseudocode). `internal/lexer/token.go` defines only `SHL`/`SHR`.

**V12 — Negative-existence: bitwise builtins are absent from the frozen builtin interface.** `grep -n "shiftRight\|bitwise" internal/iface/builtin_freeze.go` → empty. `FrozenBuiltinInterface` is a hand-maintained 38-entry list (arithmetic/comparison/string/show/io) with a golden-digest stability test (`internal/iface/builtin_freeze_test.go`); the bitwise operator builtins (including `shiftRight_Int`) were never added to it. Precedent: adding `shiftRightLogical_Int` to the registry follows its five bitwise siblings — registry-only, no freeze-table edit, no digest update (supports the Risk-table claim).

**V9 — Builtin registry mechanics (Part 2 feasibility):** `registerBuiltinWithMeta` (internal/builtins/math.go:97) auto-derives type/arity/module("std/math")/metadata from name+arity; `AdaptedBuiltinNames` (internal/bytecode/builtin_adapt.go:166) is **computed from the pure registry** ("every pure registry builtin reaches the VM in exactly one of three ways" — comment + coverage tests at internal/vm/builtin_coverage_test.go:213-225, which pin `_shiftRight_Int` etc. as adapted). Adding a pure builtin needs no adapter/freeze edit.

**V10 — Operator-builtin chain (for the deferred `>>>` only):** op_table.go:51 `core.OpShiftRight → "shiftRight"` lowering, eval_operations.go:336-338 string-dispatch `case core.OpShiftRight: op = ">>"`, smt/codegen_operators.go:22 `bvashr` (a `bvlshr` slot exists in SMT-LIB), format/precedence.go:46 precShift. No `>>>` in syntaxes/ailang.tmLanguage.json.

**V11 — Downstream int64 fidelity (negative-constant safety):** bytecode constant pool → `bytecode.NewInt(int64)` stored raw in `Value{Tag: TagInt, Int}` (compiler/expr.go:121, value.go:75); generated Go → `int64(%d)` (gen/golang/codegen_expr_simple.go:38) — negative-safe; fmt → `strconv.FormatInt(v, 10)` canonicalizes hex to signed decimal **today** (`ailang fmt` on `0xE000` prints `57344`; format/literal.go "there is no source-text echo"); elaborate passes the int64 through unchanged (expr_simple.go).

**V5 — Latent SMT bug the fix exposes:** smt/codegen_expr.go `encodeLit`: `if v < 0 { return fmt.Sprintf("(- %d)", -v) }` — for `v == math.MinInt64`, Go's `-v` wraps to `MinInt64`, emitting the malformed S-expression `(- -9223372036854775808)`. Unreachable today (no negative IntLit can exist pre-fix: unary minus is an operator, not part of the literal — m-pure-prng probe 1 verified `-9223372036854775808` fails to parse); reachable the moment `0x8000000000000000` parses. Hence Phase 3.

**V13 — Error code unallocated:** `grep -rn "PAR021" internal/ cmd/` → empty. PAR001–PAR020 + PAR999 allocated (PAR020 = missing-semicolon diagnostic, parser_expr.go:389). PAR021 is free.

**V14 — Prior art / precedent claims:** v0.26.0 added radix literals after a model burnt its step budget on hex→decimal (changelogs/v0.26-v0.31-measurement-cost.md, "Supporting what models naturally write removes the wall"); m-pure-prng round-1 quorum rejected an off-by-one `lsr` mask helper (objection 2) — the measured basis for calling mask emulation a hazard; the reporter's workaround form (`0 - 7046029254386353131`) and constants are byte-identical to m-pure-prng probe 0's machine-computed values, re-verified independently here by Python (GOLDEN = −7046029254386353131, MIX1 = −4658895280553007687, MIX2 = −7723592293110705685; `int64(0x9e3779b97f4a7c15 + 1)` = −7046029254386353130).

**V15 — Duplicate/coverage gate:** repo-wide search for literal/shift design docs: the **only** related doc is [m-pure-prng](../../planned/v0_29_0/m-pure-prng.md) (planned, PARKED), which explicitly deferred both halves of this work — §3.2: "a full-width-hex lexer extension can be proposed independently if a second consumer (hashing, crypto, bit-twidding) shows demand"; §7: logical-shift builtin "Defer unless a second use case appears." This report (stapledons-godot sim/rng.ail) **is** that second consumer — the demand-evidence gate m-pure-prng set is satisfied. No implemented doc touches literal parsing (changelog search: only v0.26.0's radix-prefix addition). SimHash doc-search results returned generic 1.00 scores on unrelated docs (m-openai-agents-api-executor etc.) — noise, disregarded; the authoritative check was the repo-wide content grep.

## Related Documents

**Planned (check for overlap):**
- [m-pure-prng](../../planned/v0_29_0/m-pure-prng.md) — the parked `std/prng` design. **Distinction:** m-pure-prng is a *stdlib module* that works around both gaps (signed-decimal constants §3.2, pure `ushr` helper §3.3); this doc fixes the *language-level* gaps themselves. No overlap in scope; m-pure-prng's demand-evidence gate is what this report satisfies, and its known-answer vectors (§6) are reused as this doc's acceptance tests.
- [m-cross-arch-float-determinism](m-cross-arch-float-determinism.md) — adjacent (determinism of numeric builtins across GOARCH) but disjoint: that doc is about transcendental floats; this one is exact integer semantics, which are arch-stable.

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- SplitMix64: Guy L. Steele Jr., Doug Lea, Christine H. Flood, "Fast Splittable Pseudorandom Number Generators" (2014) — the constants `0x9E3779B97F4A7C15`, `0xBF58476D1CE4E5B9`, `0x94D049BB133111EB`
- [v0.26.0 radix-literal changelog](../../../changelogs/v0.26-v0.31-measurement-cost.md) — the precedent for "support what models naturally write"
- Report source: coordinator task `task-685c1642` (stapledons-godot `sim/rng.ail` workaround)

## Future Work

- **`>>>` operator** — the pre-audited file list is in Solution Design; adopt on the demand-evidence gate in Non-Goals.
- **m-pure-prng simplification** — when unparked, its `ushr` helper and §3.2 signed-decimal constants collapse to `shiftRightLogical` and natural hex; its `1 << 63` min-int idiom becomes `0x8000000000000000`.
- **More hash-constant examples** — FNV-1a-64 (`0xcbf29ce484222325` offset basis now parses), murmur3, wyhash as runnable examples once needed by a benchmark.

---

**Document created**: 2026-10-01
**Last updated**: 2026-10-01

---

## Implementation Report (2026-10-02)

Shipped as designed, with these additions found while verifying:
- **`ailang fmt` round-trip** (conflict surface the doc assumed was stable): a negative `IntLit`
  printed as decimal `-7046029254386353131` re-parses as *unary minus* on a positive literal, a
  different AST, so fmt refused the file ("round-trip verification failed"). `-9223372036854775808`
  does not re-parse at all. fmt now prints a negative literal as its 64-bit hex pattern. A negative
  literal can only come from a full-width radix literal, so the canonical form *preserves*
  `0x9e3779b97f4a7c15` instead of turning it into a decimal (`internal/format/literal.go`, test
  `TestFullWidthIntLiteralsRoundTrip`).
- **PAR021 suggestion for 2^63**: the doc proposed suggesting `-9223372036854775808`, but that is
  unary minus on an out-of-range literal and is itself a PAR021. For that one value the suggestion
  names only `0x8000000000000000`. For other overflowing decimals ≤ 2^64−1 it names both forms,
  e.g. `18446744073709551615` → `0xFFFFFFFFFFFFFFFF` or `-1`.
- **Compiled Go**: `shiftRightLogical` got a `GoCodegenSpec`
  (`int64(uint64(toInt64(a)) >> toInt64(n))`). Without it `--emit-go` emitted an undefined
  `ShiftRightLogical`. A negative count panics in generated Go, as the native `>>` does.
- Prompt: the unfrozen head `v0.16.6` is amended in place (two lines under "Signed hash semantics").
  The registry hash is updated and mirrored to `cmd/ailang/prompts` and `docs/docs/prompts/current.md`.
  The stdlib interface for `std/math` is re-frozen (`shiftRightLogical` added).

Verified on both engines (`ailang run` and `--bytecode --strict-bytecode`): `0x9e3779b97f4a7c15 + 1`
= `-7046029254386353130`; the pattern `0x9e3779b97f4a7c15` matches `-7046029254386353131` and not
`MaxInt64` (before: it silently matched `MaxInt64` on both engines). The `shiftRightLogical`
known answers hold, and a negative count raises `RT_SHIFT`. `examples/runnable/splitmix64.ail`
reproduces the Python/C reference vectors (seed 0 → `-2152535657050944081`, seed 42 →
`-4767286540954276203`) with manifest-pinned stdout. Mutation-tested: reverting ParseUint→ParseInt,
discarding the pattern error, making the shift arithmetic, reverting the fmt hex form, or reverting
the SMT `uint64` magnitude each fails its test.
