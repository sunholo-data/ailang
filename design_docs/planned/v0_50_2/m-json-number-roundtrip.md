# M-JSON-NUMBER-ROUNDTRIP: std/json float numbers do not round-trip — −0.0 sign loss, ≥1e21 integer-digit expansion, ≥2^63 decode saturation, silent ±Inf on out-of-range literals

**Status**: PLANNED
**Target**: v0.50.2
**Priority**: P1 — silent data corruption (wrong values, lost signs) at the std/json system boundary, plus architecture-conditional output text (A1 violation). Not P0: no memory-safety or capability impact; the corrupted values are recoverable from source JSON text.
**Estimated**: 2–3 days (one small sprint; ~200 LOC incl. tests)
**Dependencies**: None. Self-contained to the three JSON number codecs in `internal/eval`, `internal/builtins`, `internal/vm`.
**Author**: Claude (design-doc-creator, coordinator task `task-0039cf2e`, 2026-10-01). Report source: user bug report against v0.50.0-6-g021c46907-dirty (workaround carried in stapledons-godot `sim/protocol.ail`, `num`). All repro commands in this doc were run with the v0.50.1 binary built at `021c469` (same commit as the report binary) — interpreter and `--strict-bytecode`, identical results.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes an implementation-defined `int64(float64)` conversion from the live format path (Go spec: result is implementation-dependent for out-of-range values); after the fix, output text cannot depend on host CPU. Also makes the three backends byte-identical. |
| A2: Replayability | 0 | Pure builtins; trace semantics unchanged. |
| A3: Effect Legibility | 0 | No effect changes (`_json_encode`/`_json_decode` stay pure). |
| A4: Explicit Authority | 0 | No capability surface touched. |
| A5: Bounded Verification | 0 | No type-system change. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | Shortest round-trip text = fewer tokens for AI consumers; a machine-checkable round-trip law replaces three divergent heuristics. |
| A8: Minimal Syntax | 0 | No syntax change. |
| A9: Cost Visibility | 0 | No resource-cost change. |
| A10: Composability | 0 | Same API surface. |
| A11: Structured Failure | +1 | Out-of-range number literals decode to `Err(...)` instead of silently becoming ±Inf/0 (CLAUDE.md principle 2: no silent fallbacks affecting data integrity). |
| A12: System Boundary | +1 | The JSON boundary stops corrupting values crossing it; emitted number text is byte-compatible with the surrounding Go/JS ecosystems. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): fix *removes* implicit nondeterminism (implementation-defined conversion, backend divergence).
- [x] A3 (Effects): no hidden side effects introduced.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): shortest-form text serves machine consumers; not a human-convenience trade.

---

## Problem Statement

`std/json` is AILANG's system boundary for structured data (API calls, protocol messages, config). Seven defect classes — the three reported plus four more found while auditing the same code paths (CLAUDE.md principle 3: audit before patching) — break the round-trip law `decode(encode(x)) == x` for `JNumber`:

### Current State (all verified live, v0.50.1 @ 021c469, interpreter AND `--strict-bytecode` — byte-identical)

| # | Expression (via `std/json`) | v0.50.1 output | Defect |
|---|---|---|---|
| E1 | `encode(jnum(-0.0))` | `0` | Negative-zero sign lost by the encoder. |
| E2a | `encode(jnum(1.0e21))` | `1000000000000000000000` (22 chars) | Whole floats ≥ 1e21 expand to integer digits instead of exponent form. |
| E2b | `encode(jnum(1.0e308))` | 309 chars | Same, large end. |
| E2c | `encode(jnum(5.0e-324))` | 324 chars | Same, small end (subnormals via `'f'` format). |
| E3 | `encode(jnum(9223372036854775808.0))` (2^63) | `9223372036854776000` (via `'f'`) on amd64 | The whole-float guard `f == float64(int64(f))` uses an implementation-defined conversion for \|f\| ≥ 2^63; on saturating architectures (e.g. arm64 `FCVTZS`) the guard can be true and `FormatInt` emits the *saturated* int64 digits — output text architecture-conditional. |
| E4a | `decode("{\"x\":10000000000000000000}")` re-encoded | `{"x":9223372036854776000}` | Integer literals ≥ 2^63 saturate at int64 max (`ParseInt` range-clamp, error ignored) — **wrong value** (should be exactly 1e19). |
| E4b | `decode("{\"x\":-10000000000000000000}")` re-encoded | `{"x":-9223372036854775808}` | Same, clamped at int64 min (should be −1e19). |
| E5 | `decode("-0")` re-encoded | `0` | `-0` (no `.`/`e`) routes to the Int64 branch → +0.0 — sign lost on decode. |
| E6 | `decode("[1e400, 1e-400]")` re-encoded | `[null,0]` | Out-of-range float literals (valid RFC 8259 grammar) silently become ±Inf/0 because `n.Float64()`'s error is swallowed (`f, _ :=`); ±Inf then re-encodes as `null`. Silent data corruption. |
| E7 | legacy encoder (`internal/eval/builtins_json.go`, `%g`) | `1e10` → `1e+10` (vs `10000000000` on the live path); NaN/±Inf → `NaN`/`+Inf` — **invalid JSON** | Three encoders disagree with each other; the legacy one emits non-RFC text for non-finite values. |

**Impact**: anyone round-tripping floats through `std/json` — the reporter's stapledons-godot protocol serialization carries a hand-rolled `num` workaround (emit explicit `-0.0`, exponent form for long integer text) that exists *only* because of E1/E2. Saturation (E4) silently corrupts values like `10000000000000000000` (a plausible shard size / timestamp-in-ns / money value). E6 converts unrepresentable literals into `null` without any error signal.

### Root cause

Three *encoder* implementations and three *decoder* implementations of the same JSON number, each hand-rolled, none sharing code:

| Site | Encoder logic | Decoder logic |
|---|---|---|
| `internal/builtins/json_encode.go` `formatNumber` (lines 226–239) — live interpreter path | whole → `FormatInt(int64(f))` at :235 (E1, E3); else `'f',-1` at :239 (E2); NaN/Inf → `null` | — |
| `internal/vm/builtins_json.go` `vmFormatNumber` (149–157) — `--strict-bytecode` path | same as above (:154/:156) | `vmMakeJNumber` (345–352): `ContainsAny(".eE")` → `Float64()` at :348; else `Int64()` at :351 (E4, E5), errors swallowed |
| `internal/eval/builtins_json.go` `encodeJSON` (81; JNumber branch at :106, `%g`) — legacy registry; asserted by `internal/eval/json_test.go` | `%g` (E7) | `interfaceToJSON` (364; `case json.Number` at :384): same `Int64()` saturation (E4, E5) |
| `internal/builtins/json_decode.go` `makeJNumber` (477–501) — live decode path | — | `ContainsAny(".eE")` at :481 → `Float64()` at :482; else `Int64()` at :492, `f, _ :=` swallows range errors (E4, E5, E6) |

Notably `std/json.decodeFloatArray` (`internal/builtins/float_codec.go:212`, shipped v0.47.0, M-NUMERICS-VEC-ARRAY-INGEST) already parses numbers correctly with `strconv.ParseFloat` — so the two decode surfaces of the *same module* disagree with each other today.

---

## Goals

**Primary Goal:** Make `std/json` number text exact: for every finite `float64 f`, `decode(encode(JNumber(f)))` yields `JNumber(g)` with `g` bit-identical to `f` (sign bit included), and every JSON number literal inside float64 range decodes to its nearest float64.

**Success Metrics:**
1. Round-trip property test passes for a table of ~40 boundary values plus ≥10,000 random bit patterns, on interpreter and `--strict-bytecode` (bit-identity via `Float64bits`, since `-0.0 == 0.0` would hide E1/E5).
2. `encode(jnum(-0.0))` → `"-0.0"`; `encode(jnum(1.0e21))` → `"1e+21"`; `encode(jnum(1.0e308))` → `"1e+308"`; `encode(jnum(5.0e-324))` → `"5e-324"`.
3. `decode("{\"x\":10000000000000000000}")` → `JNumber(1e19)` exactly (no ±2^63 saturation anywhere).
4. `decode("[1e400]")` → `Err(...)` (no silent ±Inf).
5. All three Go implementations produce byte-identical text for the full test table (differential test).
6. Zero churn in the currently-correct range: all existing `json_encode_test.go`, `json_decode_test.go`, `internal/vm/builtins_json_test.go` cases and the `examples/runnable/` JSON fixtures (`ai_call.ail`, `claude_haiku_call.ail`, `json_jint.ail`) keep their current output.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Canonical number text = **Go `encoding/json` / ECMAScript window rule** (shortest digits; `'f'` for 1e-6 ≤ \|f\| < 1e21, `'e'` outside with exponent zero-strip) — *not* blanket `FormatFloat(f,'g',-1)` | Byte-parity with the Go and JS JSON emitters AILANG text must interoperate with; churn confined to the broken ranges (a blanket `'g'` would rewrite every mid-range whole float, e.g. `10000000` → `1e+07`, `12345678` → `1.2345678e+07`) | **human** — the report asked for `'g',-1`; this doc recommends the window rule instead (see Alternatives); reporter must sign off | design | med |
| D2: Negative zero encodes as `"-0.0"` | Round-trips the sign; token stays syntactically non-integer (contains `.`) so syntax-discriminating decoders keep it a float; matches the report's stated example and Python `json.dumps(-0.0)` | **human** (report example); Go emits `"-0"`, JS `"0"` (both lose-by-default alternatives considered) | design | low |
| D3: One canonical formatter `eval.FormatJSONNumber`, consumed by all three backends | `internal/builtins` and `internal/vm` already import `internal/eval`; `internal/eval` cannot import `internal/builtins` (cycle) — `eval` is the only zero-cycle host (VL-9) | agent | design | low |
| D4: Decode via `json.Number.Float64()` (`strconv.ParseFloat`) on all paths; drop the `ContainsAny(".eE")`/`Int64()` branch; propagate errors | Exact for every in-range literal; `Err` instead of silent ±Inf (no-silent-fallbacks); makes the Json-tree decoders match `decodeFloatArray` | agent (principle-driven) | design | low |
| D5: NaN/±Inf encode as `"null"` everywhere (retire legacy `%g`'s invalid `NaN`/`+Inf`) | RFC 8259 has no non-finite tokens; `"null"` is the existing registry/VM behavior — one behavior, not three | agent | design | low |
| D6: `JNumber(IntValue)` (WASM-bridge shape) keeps `%d` integer text | Bridge contract (whole JS numbers arrive as `IntValue`); int64 text is exact and deterministic | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1** — human confirms the window rule (default) or overrides to strict `FormatFloat(f,'g',-1)` (alternative table provided below; the sprint swaps one function body either way).
- [ ] **D2** — human confirms `"-0.0"` (vs Go-parity `"-0"`).

All remaining decisions (D3–D6) are agent-chosen and frozen in this doc.

---

## Solution Design

### Overview

One canonical number codec pair, applied at all six sites:

- **Encode** — a single exported `FormatJSONNumber(f float64) string` in `internal/eval`, called by `formatNumber` (registry), `vmFormatNumber` (VM), and `encodeJSON` (legacy). The `IntValue` branches stay `%d` (D6).
- **Decode** — all three `makeJNumber`-shaped sites call `n.Float64()` and propagate its error through the existing `Result[Json, string]` envelope (D4).

### The number format (D1) — exact specification

For finite `f`:

1. `f == 0`: emit `"0"`, or `"-0.0"` when `math.Signbit(f)` (D2).
2. Otherwise let `abs = math.Abs(f)`:
   - `1e-6 <= abs < 1e21`: `strconv.FormatFloat(f, 'f', -1, 64)` — shortest decimal digits, no exponent.
   - `abs < 1e-6` or `abs >= 1e21`: `strconv.FormatFloat(f, 'e', -1, 64)`, then strip a leading zero from a two-digit exponent (`1e-07` → `1e-7`), matching Go `encoding/json`'s cleanup and ECMAScript.
3. NaN/±Inf: `"null"` (D5).

This is byte-identical to Go `encoding/json`'s float encoder for finite nonzero values and to `JSON.stringify` for the same range — the two ecosystems AILANG JSON travels through. Verified expected outputs (window rule via `node`'s `JSON.stringify` for algorithm; `'g'` column via live `floatToStr` — see Verification Log):

| f | v0.50.1 (live) | **new** | `floatToStr` (`'g',-1`) — rejected |
|---|---|---|---|
| -0.0 | `0` | `-0.0` | `-0` |
| 0.0 | `0` | `0` | `0` |
| 42.0 | `42` | `42` | `42` |
| 3.14 | `3.14` | `3.14` | `3.14` |
| 12345678.0 | `12345678` | `12345678` | `1.2345678e+07` |
| 1.0e7 | `10000000` | `10000000` | `1e+07` |
| 2^63 (9223372036854775808.0) | `9223372036854776000` | `9223372036854776000` | `9.223372036854776e+18` |
| 1.0e20 | `100000000000000000000` | `100000000000000000000` | `1e+20` |
| 1.0e21 | `1000000000000000000000` | **`1e+21`** | `1e+21` |
| 1.0e308 | 309 chars | **`1e+308`** | `1e+308` |
| 1.0e-6 | `0.000001` | `0.000001` | `1e-06` |
| 1.0e-7 | `0.0000001` | **`1e-7`** | `1e-07` |
| 5.0e-324 | 324 chars | **`5e-324`** | `5e-324` |
| ±Inf / NaN | `null` (legacy: `+Inf`/`NaN`) | `null` | — |

### The decode rule (D4) — exact specification

Every `json.Number` token (the Go decoder has already validated RFC 8259 number grammar):

```go
f, err := n.Float64()      // strconv.ParseFloat — exact for every in-range literal
if err != nil {
    return Err(fmt.Sprintf("json: number %s out of float64 range: %v", n, err))
}
return JNumber(f)
```

Properties (all testable): `10000000000000000000` → exactly 1e19; `-10000000000000000000` → −1e19; `9007199254740993` → 9007199254740992.0 (IEEE nearest, same as today); `"-0"` → −0.0 (sign preserved); `"1e400"` → `Err` (proven saturating today — VL-2); `"1e-400"` → whatever `ParseFloat` reports is propagated verbatim (underflow detail asserted in the executor's test table; either `Ok(0.0)` or `Err` is spec-conformant — the contract is *no silent ±Inf*).

### Implementation Plan

**Phase 1: canonical formatter + unit tests** (~4h)
- [ ] Add `FormatJSONNumber(f float64) string` to `internal/eval` (beside `encodeJSON` in `builtins_json.go` or a new `json_number.go` — executor's choice), with the spec above and a doc comment naming it the single source of truth.
- [ ] Table-driven unit test for the full spec table (incl. `1e-6`/`1e21` ± 1 ulp, denormal min, int64 min/max, 2^53, 2^63).

**Phase 2: swap the six sites** (~4h)
- [ ] `internal/builtins/json_encode.go`: `formatNumber` body → `return eval.FormatJSONNumber(f)`; keep `IntValue` `%d` case. Update `BuiltinMetadata` examples (add `-0.0` → `"-0.0"`, `1e21` → `"1e+21"`).
- [ ] `internal/vm/builtins_json.go`: `vmFormatNumber` → delegate; keep `TagInt` `%d` case.
- [ ] `internal/eval/builtins_json.go`: `encodeJSON` JNumber `%g` → `FormatJSONNumber`; `interfaceToJSON` number branch → `n.Float64()` + error propagation.
- [ ] `internal/builtins/json_decode.go`: `makeJNumber` → `n.Float64()` + error propagation (wire the error into the existing `Err`-string envelope like the parse error path already does).
- [ ] `internal/vm/builtins_json.go`: `vmMakeJNumber` → same.

**Phase 3: differential + round-trip tests, docs, fixtures** (~4h)
- [ ] Round-trip property test: for the boundary table + ≥10,000 random `Float64bits` patterns (skip NaN/Inf → `"null"`): `decode(encode(f))` bit-equals `f`. Run on interpreter and `--strict-bytecode` (a small `.ail` driver or the existing Go-level paths).
- [ ] Differential test: registry vs eval vs VM encoders byte-identical over the table.
- [ ] Update `internal/eval/json_test.go:64` `{1e10, "1e+10"}` → `"10000000000"` (intentional change, E7); add NaN/Inf → `null` cases.
- [ ] Add decode cases: ±1e19, `-0`, `1e400`, `1e21`, `5e-324` to `json_decode_test.go`.
- [ ] Update `_json_encode`/`_json_decode` `BuiltinMetadata` (number formatting documented); CHANGELOG entry at ship time.
- [ ] `make test`, `make check-boundaries`, `make simplicity-audit` green.

### Files to Modify/Create

**New files:**
- `internal/eval/json_number.go` — canonical `FormatJSONNumber` (~30 LOC) + its test (~120 LOC), unless the executor hosts it in `builtins_json.go` (then no new file).

**Modified files:**
- `internal/builtins/json_encode.go` — `formatNumber` delegates (~5 LOC), metadata examples (~10 LOC).
- `internal/builtins/json_decode.go` — `makeJNumber` rewrite (~15 LOC).
- `internal/vm/builtins_json.go` — `vmFormatNumber` delegates, `vmMakeJNumber` rewrite (~20 LOC).
- `internal/eval/builtins_json.go` — `encodeJSON` number branch, `interfaceToJSON` number branch (~20 LOC).
- Tests: `internal/eval/json_test.go` (1 expectation + additions), `internal/vm/builtins_json_test.go`, `internal/builtins/json_{encode,decode}_test.go` (additions).

No `std/json.ail` change — `encode`/`decode` already delegate to the builtins. No new packages, no import-direction changes (A-none).

---

## Conflict Surface

This design touches `internal/eval`, `internal/vm` (and `internal/builtins`) — the mandatory section applies. No parser/lexer/AST/type positions are touched; the surfaces are *semantic*.

### Syntactic positions touched

None. The change is confined to (a) the `JNumber(FloatValue)` branch of three encoders, (b) the `json.Number` branch of three decoders, (c) the `Err` string of `_json_decode` for out-of-range literals (a new, previously impossible outcome).

### What else lives here

| Position | Existing occupant | Disposition |
|---|---|---|
| `JNumber` field, encoder side | `FloatValue` (normal AILANG path) | **changed** — routes through `FormatJSONNumber` |
| `JNumber` field, encoder side | `IntValue` (WASM bridge `jsToAILANGValue` converts whole JS numbers to `IntValue`; also `embed.FromGo` within ±1e15) | **unchanged** — stays `%d` (exact, deterministic; `TestJSONEncodeNumberWithIntValue` pins it) |
| JSON number token, decoder side | `ContainsAny(".eE")` → Float64 / else Int64 heuristic | **heuristic deleted** — single `Float64()` branch (removes a discrimination, adds none) |
| Same module, sibling decoder | `decodeFloatArray` → `strconv.ParseFloat` (float_codec.go:212) | **unchanged, becomes the reference** the tree decoders now match |
| `json.repair` | `quickValidator.parseNumber` — syntax validation only, no numeric conversion | unchanged |
| Compiled-Go backend | `registry_codegen_json.go:20` `_json_encode` stub `return "{}"`; `:15-18` `_json_decode` stub `Err("JSON decode not yet available in compiled Go mode")` | **out of scope** (JSON is unimplemented there, not wrong-numbered) — documented as an adjacent defect below |
| `_json_decode` Result envelope | `Err` strings from the Go JSON parser | shape reused for the new range error |
| Downstream consumers | `floatToInt`/`getInt` truncation; `jint` constructor | unchanged semantics |

### Disambiguation strategy

Not applicable in the parsing sense — the change removes the only text-inspection heuristic (`ContainsAny(".eE")`) rather than adding one. Encoders keep discriminating on value shape (`FloatValue` vs `IntValue`), which is structural, not textual.

### Programs that MUST still work

1. `examples/runnable/json_jint.ail` — `jint(n) == jnum(intToFloat(n))` integer numbers.
2. `examples/runnable/ai_call.ail` — `kv("max_tokens", jnum(100.0))` must still serialize `100` (window rule keeps whole mid-range floats as integer text).
3. `examples/runnable/claude_haiku_call.ail` — same pattern.
4. `internal/builtins/json_encode_test.go` current cases `{42,"42"}, {-17,"-17"}, {0,"0"}, {3.14,"3.14"}, {-2.5,"-2.5"}, {0.001,"0.001"}` and `TestJSONEncodeNumberWithIntValue` — unchanged.
5. `internal/vm/builtins_json_test.go` cases `{42,"42"}, {3.14,"3.14"}, {0,"0"}, {-1,"-1"}` — unchanged.

### What deliberately changes

The encode table above (−0.0, ≥1e21, <1e-6, subnormals), the legacy encoder's `1e10 → 10000000000` and `NaN/+Inf → null`, and on decode: ≥2^63 literals exact instead of saturated, `"-0"` sign preserved, `1e400` → `Err`. The stapledons-godot `sim/protocol.ail` `num` workaround becomes redundant once both sides upgrade — flag for consumer coordination at release time.

---

## Examples

### Example 1: The reporter's three cases

**Before** (v0.50.1, live, both execution paths):
```ailang
import std/json (encode, decode, jnum)
-- encode(jnum(-0.0))                 "0"
-- encode(jnum(1.0e21))               "1000000000000000000000"
-- decode("{\"x\":10000000000000000000}")  →  Ok(JObject([{"x", JNumber(9223372036854775808.0)}]))
--   (saturated at int64 max; re-encodes as {"x":9223372036854776000})
```

**After**:
```ailang
import std/json (encode, decode, jnum)
-- encode(jnum(-0.0))                 "-0.0"
-- encode(jnum(1.0e21))               "1e+21"
-- decode("{\"x\":10000000000000000000}")  →  Ok(JObject([{"x", JNumber(10000000000000000000.0)}]))
--   exactly 1e19; re-encodes as {"x":10000000000000000000}
-- decode("[1e400]")                  →  Err("json: number 1e400 out of float64 range: …")
```

### Example 2: Protocol round-trip without workarounds

stapledons-godot `sim/protocol.ail` currently wraps numbers (`num`) to emit explicit `-0.0` and exponent form for long integer text. After this ships, `json.encode(json.jnum(v))` alone round-trips every finite float — the workaround's number branch can be deleted on upgrade.

---

## Success Criteria

- [ ] `encode(jnum(-0.0))` == `"-0.0"` and the sign survives `decode` (asserted via `Float64bits`, not `==`)
- [ ] `encode(jnum(1.0e21))` == `"1e+21"`, `encode(jnum(1.0e308))` == `"1e+308"`, `encode(jnum(5.0e-324))` == `"5e-324"`, `encode(jnum(1.0e-7))` == `"1e-7"`
- [ ] `decode("{\"x\":10000000000000000000}")` yields exactly `1e19`; `…-10000000000000000000` yields `-1e19`
- [ ] `decode("1e400")` yields `Err`
- [ ] Round-trip property test (boundary table + ≥10k random bit patterns) green on interpreter and `--strict-bytecode`
- [ ] Registry / eval / VM encoders byte-identical over the full table
- [ ] The five Conflict-Surface fixtures above pass unchanged
- [ ] All tests passing (`make test`), `make check-boundaries` and `make simplicity-audit` green
- [ ] `_json_encode`/`_json_decode` metadata documents the number format; CHANGELOG updated at ship time

## Testing Strategy

**Unit tests:** the spec table per backend; `IntValue` cases pinned; `interfaceToJSON`/`makeJNumber`/`vmMakeJNumber` decode cases incl. ±1e19, `-0`, `9007199254740993`, `1e400`.

**Integration tests:** round-trip property test driven from `.ail` through the public `std/json` API on both execution paths (the repro programs in this doc become the fixtures); differential test across the three Go implementations.

**Manual testing:** none beyond the above (pure text functions).

## Deferred Decisions

- Exact `Err` message wording for out-of-range literals — agent may choose (envelope is `Result[_, string]`).
- Whether to also expose `std/json.encodeNumber(f: float) -> string` as an AILANG-level helper (useful for protocol code; not required by this fix) — agent may add or skip.
- Host file for `FormatJSONNumber` (`builtins_json.go` vs new `json_number.go`) — agent decides.

## Non-Goals

- **Arbitrary-precision JSON numbers** (RFC 8259 permits them; `JNumber` is float64 by design). A future `JDecimal`/`JBigInt` constructor is a language change, not a bug fix.
- **Compiled-Go mode JSON** — `registry_codegen_json.go` stubs encode/decode entirely (`return "{}"` / not-available `Err`). Out of scope; recorded as an adjacent defect.
- **`show`/`floatToStr` formatting** — deliberately different presentation layer (e.g. `show(1e21)` = `1000000000000000000000.0`); untouched.
- **The wasm/Go bridge's whole-float → `IntValue` conversion** (`embed.FromGo` ±1e15 window) — upstream of the encoder; `-0.0` through that bridge still loses its sign before JSON sees it. Recorded as an adjacent defect, not fixed here.
- **Full structural unification of the three JSON encoders** — this doc unifies only the number branch; whole-tree unification is future work.

## Timeline

**Week 1** (12h + buffer ≈ 2–3 days):
- Phase 1: canonical formatter + unit tests (4h)
- Phase 2: swap six sites (4h)
- Phase 3: differential/round-trip tests, metadata, CHANGELOG, gates (4h)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Consumers diffing against v0.50.x long-integer text (≥1e21, sub-1e-6) see changed bytes | Med | Exactly the regions the report asks to change; flagged in CHANGELOG as intentional; the reporter's own workaround anticipates exponent form |
| Legacy-encoder behavior change (`1e+10` → `10000000000`, `NaN` → `null`) surprises a caller relying on the legacy registry | Low | Legacy `%g` output was already divergent from the live interpreter path and non-RFC for non-finite values; unification is the fix, and `internal/eval/json_test.go` is updated in the same commit |
| Executor machine is arm64 and current tests pass there via the saturated `FormatInt` branch | Low | The new formatter has no int64 conversion at all; the spec table is architecture-independent by construction |
| Exponent zero-strip (`1e-07` → `1e-7`) mishandled | Low | Pinned by explicit table cases (`1e-7`, `5e-324`, `1e+21`) in all three backends' tests |

## Related Documents

The create-script's neural search returned no matches (and crashed mid-run — see adjacent defects); these are curated by code citation:

**Implemented (may inform design):**
- [m-json-convenience-builders](../implemented/v0_3_9/m-json-convenience-builders.md) — `jnum`/`jint` constructors (the API this fix makes honest)
- [m-json-accessors-enabled](../implemented/v0_6_0/m-json-accessors-enabled.md) — accessor layer
- [m-dx-json-bool-coercion](../implemented/v0_30_0/m-dx-json-bool-coercion.md) — precedent for boundary-tolerance decisions in this module (`asBoolLoose`)
- [m-numerics-vec-array-ingest](../implemented/v0_47_0/m-numerics-vec-array-ingest.md) — shipped `decodeFloatArray` with correct `ParseFloat` number handling (the in-repo precedent this fix generalizes)

**Planned (check for overlap):**
- [m-bytecode-vm-parity-bugs](../planned/v1_0_0/m-bytecode-vm-parity-bugs.md) — adjacent VM-parity genre (effect rows/EVAL_SKIP files); no number-text overlap (verified by read)

**External:**
- stapledons-godot `sim/protocol.ail` (`num`) — the consumer workaround this fix retires.

## References

- [RFC 8259 §6](https://www.rfc-editor.org/rfc/rfc8259#section-6) — number grammar and interoperability advice
- Go `strconv` `FormatFloat`/`ParseFloat` semantics; Go `encoding/json` float encoder (window rule + exponent cleanup); ECMAScript `JSON.stringify` (same window, verified via `node` — VL-5)
- Go spec, Conversions between numeric types — implementation-dependent result for out-of-range float→int (VL-13)
- Report: v0.50.0 release, binary `021c469` (task `task-0039cf2e` attachment)

## Future Work

- Full unification of the three JSON tree encoders into one (this doc unifies the number branch only).
- Compiled-Go mode JSON support (replacing the `registry_codegen_json.go` stubs).
- A `JDecimal`/arbitrary-precision JSON number constructor if a consumer needs exact decimal semantics.

---

## Verification Log

Run 2026-10-01 with `ailang` v0.50.1 @ `021c469` (same commit as the report's v0.50.0-6-g021c46907-dirty binary). Repro programs: `/tmp/repro/jsonnum/{jsonnum,sign,exp,fmt,edge,inf}.ail` (contents inlined in the commands where load-bearing; every program passed `ailang check` first). This container has no Go toolchain and no `make` binary — Go-behavior claims below are live-probed through the AILANG binary or marked as citations; each is separately assertable by the executor's unit tests.

| # | Claim | Command | Observed |
|---|---|---|---|
| VL-1 | All reported defects reproduce on the interpreter | `ailang run --caps IO jsonnum.ail` (prints `encode(jnum(-0.0))`, `encode(jnum(0.0))`, `encode(jnum(1.0e21))`, `encode(jnum(1.0e308))`, `encode(jnum(10000000000000000000.0))`, `encode(jnum(1.5))`, then re-encodes `decode("-0")`, `decode("-0.0")`, `decode("{\"x\":10000000000000000000}")`, `decode("{\"x\":-10000000000000000000}")`, `decode("[1e21, -0.0, 100000000000000000000]")`) | `0`, `0`, `1000000000000000000000`, 309-char `1`+308 zeros, `10000000000000000000`, `1.5`, `0`, `0`, `{"x":9223372036854776000}`, `{"x":-9223372036854775808}`, `[1000000000000000000000,0,9223372036854776000]` |
| VL-2 | Same on `--strict-bytecode` | `ailang run --caps IO --strict-bytecode jsonnum.ail` | byte-identical to VL-1 (all 11 lines) |
| VL-3 | E1 is the JSON formatter's doing, not the evaluator's — `-0.0` survives evaluation into `JNumber` | `sign.ail`: `show(-0.0)`, `show(0.0)`, `show(-0.0 * 1.0)`, `show(0.0 - 0.0)`, `show(-1.0 * 0.0)` | `-0.0`, `0.0`, `-0.0`, `0.0`, `-0.0` — evaluator preserves the sign bit; only JSON text loses it |
| VL-4 | Reporter's cited algorithm `FormatFloat(x,'g',-1,64)` differs from both the report's `-0.0` example and current JSON text for mid-range values | `fmt.ail`: `floatToStr(x)` for `-0.0, 0.0, 42.0, 1.5, 10000000.0, 12345678.0, 1e20, 1e21, 1e308, 1e-6, 1e-7, 1e19, 2^63` (floatToStr is `FormatFloat(f,'g',-1,64)` — internal/builtins/string_convert.go:27) | `-0`, `0`, `42`, `1.5`, `1e+07`, `1.2345678e+07`, `1e+20`, `1e+21`, `1e+308`, `1e-06`, `1e-07`, `1e+19`, `9.223372036854776e+18` — `'g'` gives `-0` (not `-0.0`) and would churn `10000000` → `1e+07` |
| VL-5 | Window-rule outputs (the D1 spec) match ECMAScript `JSON.stringify` | `node -e '… JSON.stringify(x) …'` for `-0.0, 0.0, 1e21, 1e20, 10000000, 1e-7, 1e-6, 2**63, 1e308` | `0` (JS loses −0 — why D2 chooses `-0.0`), `0`, `1e+21`, `100000000000000000000`, `10000000`, `1e-7`, `0.000001`, `9223372036854776000`, `1e+308` |
| VL-6 | Three divergent encoders exist at the stated lines | `grep -n "FormatInt\|%g\|FormatFloat" internal/builtins/json_encode.go internal/vm/builtins_json.go internal/eval/builtins_json.go` | json_encode.go:235 `FormatInt(int64(f))` + :239 `'f',-1`; vm/builtins_json.go:154 `FormatInt` + :156 `'f',-1`; eval/builtins_json.go:106 `%g` |
| VL-7 | Three decoders saturate via ignored `Int64()`/`Float64()` errors | `grep -n "Int64()\|Float64()\|ContainsAny" internal/builtins/json_decode.go internal/eval/builtins_json.go internal/vm/builtins_json.go` | json_decode.go:481 `ContainsAny(".eE")` / :482 `f, _ := n.Float64()` / :492 `i, _ := n.Int64()`; eval `interfaceToJSON` same shape (`case json.Number` at :384); vm/builtins_json.go:348/351 same |
| VL-8 | `decodeFloatArray` (same stdlib module) already parses correctly — the module disagrees with itself | `sed -n '210,214p' internal/builtins/float_codec.go` | `x, err := strconv.ParseFloat(s[start:end], 64); if err != nil { return … }` — error propagated |
| VL-9 | `eval` is the only zero-cycle host for the canonical formatter | `grep -rh "sunholo-data/ailang/internal/" internal/vm/*.go \| grep -v _test \| sort -u` ; same for `internal/builtins/json_encode.go` imports ; `grep -rn "internal/builtins" internal/eval/*.go` (non-test) | vm imports `builtins`, `bytecode`, `eval`, `types`; builtins/json_encode.go imports `eval`; **no non-test file in `internal/eval` imports `internal/builtins`** (empty grep) → hosting in `eval` reaches all three without a cycle or a new package |
| VL-10 | `JNumber(IntValue)` encodes via `%d` and is pinned by tests (D6) | `sed -n '73,112p' internal/builtins/json_encode_test.go`; `grep -n "TagInt" internal/vm/builtins_json.go` | `TestJSONEncodeNumberWithIntValue` cases `{42,"42"}, {-17,"-17"}, {0,"0"}, {1000000,"1000000"}` (comment: "as WASM bridge produces"); vm `case bytecode.TagInt: FormatInt` |
| VL-11 | ±Inf encode as `null` on the live path (D5 baseline) | `inf.ail`: `encode(jnum(1.0e308 * 10.0))`, `encode(jnum(-1.0e308 * 10.0))` | `null`, `null` |
| VL-12 | Out-of-range literals decode silently today (E6) | `edge.ail`: `decode("[1e400, 1e-400]")` re-encoded | `[null,0]` — `1e400` became +Inf (re-encodes `null`), error swallowed by `f, _ :=` |
| VL-13 | `f == float64(int64(f))` relies on implementation-defined conversion (E3); amd64 falls through to `'f'` | `edge.ail`: `encode(jnum(9223372036854775808.0))` (2^63) and VL-1's `encode(jnum(1.0e21))`; Go spec, Conversions between numeric types ("…the conversion succeeds but the result value is implementation-dependent") | 2^63 → `9223372036854776000` = `'f',-1` shortest text (NOT `FormatInt` of any int64 — a saturated arm64 `FormatInt(9223372036854775807)` would print `…807`); 1e21 → 22-char `'f'` text proves the guard compared unequal on amd64. arm64 saturation is a spec-reading prediction (no arm64 box in this container) — the fix deletes the conversion, making the point moot |
| VL-14 | Tiny magnitudes expand today (E2c/E2 small end) | `edge.ail`: `encode(jnum(1.0e-7))`, `encode(jnum(5.0e-324))` | `0.0000001`; 324-char `0.0000…0005` |
| VL-15 | Existing test expectations inventoried (no invention) | `sed -n '46,64p' internal/builtins/json_encode_test.go`; `grep -n "1e10\|1.5e10\|42.0" internal/eval/json_test.go internal/builtins/json_decode_test.go`; `grep -n "NaN\|Inf(" internal/eval/json_test.go` (rc=1, no match); `sed -n '40,60p' internal/vm/builtins_json_test.go` | encode cases `{42,-17,0,3.14,-2.5,0.001}` (all unchanged by D1); eval/json_test.go:64 `{1e10,"1e+10"}` (the ONE intentional change, E7) and **no NaN/Inf case** (grep empty); decode cases `42.0`, `3.14`, `1.5e10` (unchanged — `1.5e10` = 15e9 < 2^53, exactly representable, same float via ParseFloat); vm cases `{42,3.14,0,-1}` (unchanged) |
| VL-16 | Conflict-Surface fixtures exist | `ls examples/runnable/json_jint.ail examples/runnable/ai_call.ail examples/runnable/claude_haiku_call.ail`; `grep -n "jnum(100.0)" examples/runnable/ai_call.ail examples/runnable/claude_haiku_call.ail` | all three exist; both API examples carry `kv("max_tokens", jnum(100.0))` → must keep emitting `100` (window rule does) |
| VL-17 | Compiled-Go backend JSON is a stub (Non-Goal premise) | `sed -n '13,22p' internal/builtins/registry_codegen_json.go` | `_json_decode` → `Err("JSON decode not yet available in compiled Go mode")`; `_json_encode` → `return "{}"` |
| VL-18 | `json.repair` never converts numbers (safe to ignore) | `grep -n "parseNumber" internal/builtins/json_repair.go` | `quickValidator.parseNumber` — returns bool (grammar check only) |
| VL-19 | `make check-boundaries` exists (executor gate) | `grep -rn "check-boundaries" make/*.mk` (no `make` binary in this container — file-level verification) | make/code-health.mk:181 `check-boundaries: ## Check architecture layer boundaries (CI gate)` |
| VL-20 | No existing/queued design doc covers JSON number text (duplicate gate) | `ailang docs search --stream implemented/planned` via the create script (crashed — see adjacent defects); targeted: `find design_docs -iname "*json*" -o -iname "*numerics*"`; `grep -n "json\|JNumber\|float" design_docs/planned/v1_0_0/m-bytecode-vm-parity-bugs.md` | nearest docs are accessors/builders/bool-coercion (v0_3_9, v0_6_0, v0_30_0) and vec-array-ingest (v0_47_0, `decodeFloatArray`); the planned VM-parity doc concerns effect rows/EVAL_SKIP — no overlap |
| VL-21 | `std/json.ail` needs no change (encode/decode delegate to builtins — backs the Files table) | `grep -n "_json_encode(obj)\|_json_decode(s)" std/json.ail` | :14 `encode` body is `_json_encode(obj)`; :19 `decode` body is `_json_decode(s)` — the AILANG layer is pure delegation |

## Adjacent defects found while verifying (recorded, not fixed here)

1. **`create_planned_doc.sh` crashes** before creating the file whenever doc search returns no matches: `grep -E "^\d+\."` uses PCRE-only `\d` (always empty under `-E`), so `merge_results`'s inner grep exits 1 and `set -e` + pipefail abort the script at the first `IMPLEMENTED=$(merge_results …)` call. Reproduced twice this session (exit 1, no file). This doc was scaffolded by hand to match the script's documented output; the script fix (`\d` → `[0-9]`) is a one-liner for a future trivial-fix task.
2. **Compiled-Go mode has no JSON**: `_json_encode` compiles to a stub returning `"{}"` and `_json_decode` to a not-available `Err` (VL-17). Any compiled program using std/json silently emits `{}`.
3. **The Go-value bridge loses −0.0 before JSON sees it**: `embed.FromGo` converts whole floats within ±1e15 to `IntValue` (by design, for JS interop), so a −0.0 arriving through that bridge is +0 by encode time. Same-module, different-layer; out of scope.

## Out of scope (see Non-Goals)

Arbitrary-precision numbers; compiled-mode JSON; `show`/`floatToStr` presentation; bridge conversion policy; full encoder-tree unification.
