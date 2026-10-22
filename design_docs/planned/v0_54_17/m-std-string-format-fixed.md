# M-STD-STRING-FORMAT-FIXED: fixed-precision float formatting for reports

**Status**: Planned
**Target**: v0.54.17
**Priority**: P1 (Medium — every rounded report value prints binary noise or carries int round-trip string surgery; no engine provides a fix)
**Estimated**: 1–2 days (4h impl + 3h tests + 1h docs, 2x buffer)
**Dependencies**: None (pairs with [M-STD-AUDIO-WAV-READ](./m-std-audio-wav-read.md), same DX report)

## Problem Statement

`std/string` has exactly one float→string formatter: `floatToStr`, which is
round-trip shortest (`strconv.FormatFloat(x, 'g', -1, 64)`). For a value that
was *computed and rounded* — the standard report pattern — it prints binary
noise:

```ailang
println(floatToStr(floor(-27.0612/0.01+0.5)*0.01))
-- prints: -27.060000000000002
```

This is not a formatting bug in `floatToStr`: the `*0.01` rounding idiom
produces a float64 one ULP away from the float64 nearest to `-27.06`
(exact value: `-27.0600000000000022737…`), and **every round-trip-exact
formatter must show those digits**. Verified on this repo's binary —
`floatToStr` and `show` (which uses `'f', -1`, `internal/eval/show.go:124`)
both print `-27.060000000000002`, identically on the evaluator and
`--bytecode` (V1, V4). The fix is decimal-level rounding, which no existing
path provides:

- `std/string` exports no fixed-precision formatter — `grep` for
  `toFixed|formatFloat|formatFixed` over `std/` → empty (V2).
- There is no way to say "2 decimal places" in AILANG today, on any engine.

**Workarounds in the wild** (sunholo-audio reports, and every report writer
rediscovers): print noise digits, or round-trip through ints —
`intToStr(...)` on a scaled int plus hand-inserted decimal point — which is
fragile (negatives, |x| ≥ 2³¹, no thousands grouping) and buries the intent.

**Impact:** every AILANG report that shows a measured or rounded quantity —
audio RMS/peak levels (the sunholo-audio case), percentages, durations,
costs — either prints noise that looks like a correctness bug
(`-27.060000000000002`) or carries private string surgery. Deterministic
reporting is one of AILANG's selling points; this is a std gap, not a
language gap.

## Goals

**Primary Goal:** `std/string.formatFixed(x: float, decimals: int) -> string`
formats a float64 with exactly `decimals` digits after the point, byte-identical
on the evaluator, `--bytecode`, and AILANG→Go transpiled output.

**Success Metrics:**
- The repro line prints `-27.06`: `formatFixed(floor(-27.0612/0.01+0.5)*0.01, 2)`.
- Evaluator, `--bytecode`, and `ailang compile -emit-go` outputs are byte-identical (V7/V8 mechanisms).
- One call replaces the int round-trip workaround; no report in this repo needs string surgery for decimals again.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| Normative semantics = `strconv.FormatFloat(x, 'f', decimals, 64)` — not a hand-rolled formatter | Any deviation between interpreter, VM and transpiled Go silently forks report output across engines | human | design | high |
| Plain `string` return + `requires { decimals >= 0 }`; negative `decimals` is a hard error, never a silent `%g` fallback | `decimals < 0` passed to `FormatFloat` silently means "shortest form" — a silent semantic flip; A11 forbids it | human | design | med |
| New builtin name `_str_formatFixed` (canonical `_str_` family), NOT `_string_*` | `_string_floatToStr` is documented legacy; the resolution test pins family prefixes for new code | agent (convention) | design | low |
| `GoCodegenSpec.Inline` added so transpiled Go matches (same `strconv.FormatFloat` call) | A builtin without a spec fails loudly at `ailang compile`; with a wrong spec it silently disagrees | agent | design | low |
| `decimals` has no upper cap beyond `>= 0` | A cap is an invented constraint; `prec > 17` deterministically prints zeros (documented, not clamped) | human | design | low |
| NaN/±Inf pass through as `"NaN"`/`"+Inf"`/`"-Inf"`, negative zero keeps its sign | Same pass-through policy as `floatToStr`; report writers see the anomaly, not a mask | compiler (Go semantics) | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] `FormatFloat(x, 'f', prec, 64)` is the single source of truth on all
      three engines (evaluator `Impl`, VM adapted call to the same `Impl`,
      transpiled `Inline` — same Go function).
- [x] Plain-string return; `requires { decimals >= 0 }`; violation aborts
      loudly naming the argument (no `Result`, no clamp, no `%g` fallback).
- [x] Builtin `_str_formatFixed`; `GoCodegenSpec` added in
      `registry_codegen_string.go` (Inline form).
- [x] Export name `formatFixed` (requester's `formatFloat(x, decimals)`
      alternative folded in: `formatFixed` names the *behaviour*; the module
      already has `floatToStr` for the other one).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Registration file: extend `internal/builtins/string_convert.go` (where
  `_string_floatToStr` lives) vs a new `string_format.go` — agent may choose.
- Exact `Metadata.LongDesc` wording and example set — agent may choose.
- Whether the tie-behaviour golden (`formatFixed(0.125, 2)`) is pinned in the
  Go table test, an `.ail` fixture, or both — agent may choose (must be
  generated from a live Go run, not hand-written; see V9).
- Test placement: extend `internal/builtins/string_convert_test.go` vs a new
  test file; extend `tests/codegen-harness/string_ops.ail` vs leave it —
  agent may choose.

## Solution Design

### Overview

One new std function wrapping one new pure builtin, plus its transpile spec:

```ailang
-- std/string.ail (new export; module already imports std/option, needs nothing new)

-- Fixed-precision decimal formatting: exactly `decimals` digits after the
-- point, rounding the float64's exact binary value at the decimal level.
-- The complement of floatToStr (shortest round-trip): formatFixed keeps
-- reports free of representation noise.
--   formatFixed(floor(-27.0612/0.01+0.5)*0.01, 2)  = "-27.06"
--   formatFixed(1234.5678, 2)                       = "1234.57"
--   formatFixed(3.14159, 0)                         = "3"
-- Semantics are exactly Go strconv.FormatFloat(x, 'f', decimals, 64) --
-- identical bytes on the interpreter, the bytecode VM and compiled Go.
-- NaN / +Inf / -Inf pass through ("NaN", "+Inf", "-Inf"); negative zero
-- keeps its sign. decimals > 17 prints zeros beyond float precision.
export pure func formatFixed(x: float, decimals: int) -> string
requires { decimals >= 0 } {
  _str_formatFixed(x, decimals)
}
```

### Architecture

**Components:**

1. **`_str_formatFixed` builtin** (`internal/builtins/string_format.go` or
   `string_convert.go`), registered like `_string_floatToStr`
   (`string_convert.go:18–61`):

   ```go
   formatFixedImpl := func(_ *effects.EffContext, args []eval.Value) (eval.Value, error) {
       f, ok := args[0].(*eval.FloatValue)
       if !ok {
           return nil, fmt.Errorf("formatFixed: expected float, got %s", ailangTypeName(args[0], "float"))
       }
       d, ok := args[1].(*eval.IntValue)
       if !ok {
           return nil, fmt.Errorf("formatFixed: expected int, got %s", ailangTypeName(args[1], "int"))
       }
       if d.Value < 0 {
           return nil, fmt.Errorf("formatFixed: decimals must be >= 0 (got %d); FormatFloat would silently switch to shortest-form", d.Value)
       }
       return &eval.StringValue{Value: strconv.FormatFloat(f.Value, 'f', d.Value, 64)}, nil
   }
   // Type: T.Func(T.Float(), T.Int()).Returns(T.String()).Build()
   ```

   The `d.Value < 0` hard error is load-bearing: `strconv.FormatFloat` with a
   negative precision silently means *shortest representation* — i.e. the
   contract violation would silently produce `floatToStr` output. Fail loudly
   (CLAUDE.md "No silent fallbacks"; the `requires` documents it at the call
   site and `--verify-contracts` can check it).

2. **VM parity — free by construction.** Pure registry builtin,
   `float,int -> string`: all-scalar parameters, scalar result → the generic
   adapter picks it up (`internal/bytecode/builtin_adapt.go:45–70`), the VM
   calls the *same registered Go `Impl`*, and
   `TestAdaptedBuiltinsNeverPanic` sweeps it with sample args `(1.5, 1)`
   → `"1.5"`, no panic. The coverage ratchet
   (`internal/vm/builtin_coverage_test.go:31`) passes automatically
   (`AdaptReason` returns "" for this signature).

3. **Transpile parity — `GoCodegenSpec.Inline`**
   (`internal/builtins/registry_codegen_string.go`, alongside the
   `_string_floatToStr` spec at :273):

   ```go
   registerIfMissing("_str_formatFixed", 2, true, &GoCodegenSpec{
       Inline:     `strconv.FormatFloat({{arg0}}.(float64), 'f', int(toInt64({{arg1}})), 64)`,
       Imports:    []string{"strconv"},
       StdlibName: "formatFixed", StdlibModule: "std/string",
   })
   ```

   `{{arg0}}.(float64)` and `toInt64({{arg1}})` are the established
   marshalling idioms in that file (`_string_floatToStr` uses
   `fmt.Sprintf("%g", {{arg0}}.(float64))`, `_string_intToStr` uses
   `toInt64`). The Inline emits *the same `strconv.FormatFloat` call* the
   interpreter `Impl` makes — one source of truth, three engines.
   With `StdlibName`/`StdlibModule` set, `StdlibIndex` resolves
   `std/string.formatFixed → _str_formatFixed`, and the family check in
   `stdlib_codegen_resolution_test.go` passes because the name carries the
   module's `_str_` prefix (V6).

### Implementation Plan

**Phase 1: builtin + std export (~2 hours)**
- [ ] Register `_str_formatFixed` (impl + type + metadata, `Since: v0.54.17`).
- [ ] Add the `formatFixed` export to `std/string.ail` with the doc comment above.
- [ ] Add the `GoCodegenSpec.Inline` registration.

**Phase 2: tests (~3 hours)**
- [ ] Go table test: the repro value; `(-27.0612 rounded) → "-27.06"`;
      `(1234.5678, 2) → "1234.57"`; `(3.14159, 0) → "3"`;
      `(2.675, 2) → "2.67"` (exact value 2.6749999…822 — *not* a tie);
      `(42.0, 2) → "42.00"`; `(0.0, 3) → "0.000"`; `(1e21, 2)`; NaN/Inf
      pass-through; `decimals < 0` → hard error (not `"2.675"`-shaped `%g`).
- [ ] Generate the tie-behaviour golden (`formatFixed(0.125, 2)`) **from a
      live Go run** at sprint time and pin it in the table (V9: the policy
      is whatever `strconv` does; the doc deliberately does not guess it).
- [ ] `make test`: VM coverage ratchet + never-panic + resolution tests green.
- [ ] `.ail` integration fixture (in the codegen harness or a stdlib test):
      repro program, evaluator vs `--bytecode` byte-identical.

**Phase 3: docs + example (~1 hour)**
- [ ] New runnable example `examples/string_format_fixed.ail`: the repro
      before/after plus a small report table (value, `floatToStr`,
      `formatFixed(…, 2)`).
- [ ] Changelog entry under v0.54.17.

### Files to Modify/Create

**Modified files:**
- `std/string.ail` (+~15 LOC) — `formatFixed` export + doc comment.
- `internal/builtins/string_convert.go` or new `string_format.go` (+~50 LOC) — registration + impl.
- `internal/builtins/registry_codegen_string.go` (+~7 LOC) — `GoCodegenSpec.Inline`.
- `internal/builtins/string_convert_test.go` or new test file (+~60 LOC) — table tests.
- `examples/` — new `string_format_fixed.ail` (~30 LOC).
- `changelogs/v0.32-current.md` (+~10 LOC) — v0.54.17 entry.

**New files:** none beyond the example (and optionally the builtin/test file).

## Conflict Surface

### Syntactic positions touched

No parser, lexer, type, elaboration, codegen-pipeline or eval changes. Three
namespaces:

1. `std/string`'s export set (new name: `formatFixed`).
2. The builtin registry (`_str_formatFixed`).
3. The `GoCodegenSpec` claim table (`StdlibIndex` gains `std/string.formatFixed`).

### What else lives in that position

| Position | Existing occupants | Coexistence |
|---|---|---|
| `std/string` float/int→string exports | `floatToStr`, `intToStr`, `stringToInt`, `stringToFloat` | Add-only; no shadowing (`formatFixed` free repo-wide, V2) |
| `_str_*` builtin family | `_str_len`, `_str_slice`, `_str_repeat`, … (canonical family per `stdlib_codegen_resolution_test.go:33`) | Exact-name registry keying; family prefix required by the resolution test — satisfied |
| Same *behavioural* slot | `_string_floatToStr` (`'g', -1`), `show` (`'f', -1`, prelude) | Different rounding level, not different answers for the same promise: all three are deterministic; `formatFixed` is the only fixed-precision one |
| `GoCodegenSpec.StdlibName` claims | one claim per `(module, name)` — duplicate claims are a test error (`stdlib_codegen_resolution_test.go:81–90`) | New key, no collision (V6) |

### Disambiguation strategy

None needed: fresh identifiers, exact-name registry, derived VM adapted set.
The one real trap — a *silently divergent* second implementation — is
designed out: the `Inline` spec and the interpreter `Impl` are the same
`strconv.FormatFloat(x, 'f', prec, 64)` expression, and the VM calls the
interpreter's registered `Impl` rather than any copy. (The drift failure mode
is real and precedented in this file's own history: `charAt`'s compiled body
once indexed by BYTE while the interpreter indexed by RUNE —
`registry_codegen_string.go:176–179`. Same function on every engine is the
whole design.)

### Programs that MUST still work

- `std/string.ail`'s existing 30+ exports — untouched, add-only.
- `examples/string_char_access.ail`, `examples/string_replace.ail` — unchanged output.
- Every std module importing `std/string` (`std/audio`'s planned `intToStr`
  use in M-STD-AUDIO-WAV-READ, `std/datetime`, …) — resolution unchanged.
- The repro line's `floatToStr` behaviour — *unchanged*: `floatToStr` keeps
  its round-trip contract; `formatFixed` is additive, it does not "fix"
  `floatToStr`.

### What deliberately changes

Nothing. No existing program changes output. Anything not listed here that
breaks is a regression.

## Examples

### The DX repro (verified, V1)

**Before:**
```ailang
println(floatToStr(floor(-27.0612/0.01+0.5)*0.01))
-- -27.060000000000002   (evaluator AND --bytecode, byte-identical)
```

**After:**
```ailang
println(formatFixed(floor(-27.0612/0.01+0.5)*0.01, 2))
-- -27.06
```

### The report pattern this retires

**Before (int round-trip + string surgery, fragile):**
```ailang
-- scale to cents, integer round, re-insert the decimal point by hand:
-- breaks on |x| >= 2^31 / 100, hides intent, 6 lines
```

**After:**
```ailang
-- sunholo-audio measure report:
println("peak=${formatFixed(peakDb, 2)} dB, rms=${formatFixed(rms, 4)}")
```

## Success Criteria

- [ ] `formatFixed(floor(-27.0612/0.01+0.5)*0.01, 2)` prints `-27.06`.
- [ ] Evaluator and `--bytecode` byte-identical on the example program.
- [ ] `ailang compile -emit-go` path emits the same `strconv.FormatFloat` call (codegen harness green).
- [ ] `decimals < 0` aborts loudly naming the argument (never silent `%g`).
- [ ] `make test`, `make fmt`, `make lint` green; resolution + coverage + never-panic tests unchanged.
- [ ] Changelog + example updated.

## Testing Strategy

**Unit tests (Go):** the Phase 2 table — repro value, round-half cases
verified non-tie by exact expansion (V5), magnitude cases (`1e21`),
pass-through cases (NaN/±Inf), and the hard-error case for negative
`decimals`.

**Integration tests (AILANG):** repro fixture on evaluator and `--bytecode`
with byte-identical stdout; the new example runs under `make verify-examples`.

**Regression-surface tests:** existing string examples and stdlib string tests
unchanged; `tests/codegen-harness/string_ops.ail` (or the addition to it) via
`make codegen-harness` (`ailang compile -emit-go` + `go build`, Makefile:214).

**Manual testing:** REPL: `formatFixed(0.1 + 0.2, 2)` → `"0.30"`.

## Verification Log

Premises checked before this doc was written (binary v0.54.16, std v0.54.16,
2026-10-22; no Go toolchain in the reporting environment — `ailang` binary +
code reading + Python `decimal` exact expansions):

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | The repro prints `-27.060000000000002` | `ailang run --caps IO` on the repro file | Confirmed, byte-exact vs the DX report |
| V2 | No fixed-precision formatter exists (`toFixed` / `formatFloat` / `formatFixed`) | `grep` over `std/ examples/ internal/ docs/` → empty (dashboard JS bundles aside, which are vendored front-end) | Confirmed |
| V3 | The noise comes from the *value*, not `floatToStr`: `floor(-27.0612/0.01+0.5)*0.01` is 1 ULP off float64("-27.06") | Python `decimal` exact expansion: `-2706*0.01 = -27.0600000000000022737…`; round-trip shortest repr of that float64 is `-27.060000000000002` | Confirmed (the value itself carries the noise) |
| V4 | `show` prints the same noise (so no existing path fixes it) | `ailang run`: `show(v)` → `-27.060000000000002`; `internal/eval/show.go:124` uses `'f', -1` (shortest *fixed*, still round-trip-exact) | Confirmed |
| V5 | Doc examples are tie-free and deterministic: exact expansions of `1234.5678`, `2.675`, `0.125` | Python `decimal`: `1234.5678000000000338…` → 2 places → `1234.57`; `2.674999…822` → `2.67`; `0.125` is an **exact tie** at 2 places | Confirmed (hence V9) |
| V6 | New builtin must be `_str_*` (family prefix), and a duplicate `(module, name)` claim is a test error | read `stdlib_codegen_resolution_test.go:28–47, 33 (std/string → _str_), 81–90`; `formatFixed`/`_str_formatFixed` free (V2 grep) | Confirmed |
| V7 | `_string_floatToStr` is documented legacy naming, new code should not follow it | read `stdlib_codegen_resolution_test.go:63–67` (exemption: "legacy numeric formatting builtin predates the canonical string family spelling") | Confirmed |
| V8 | VM parity is automatic for `float,int -> string` pure builtins (same Go `Impl`), and the coverage ratchet enforces presence | read `internal/bytecode/builtin_adapt.go:16–30, 45–80, 166–191`; `internal/vm/builtin_coverage_test.go:31–46, 85–105` | Confirmed |
| V9 | The **tie policy** (`formatFixed(0.125, 2)`) is `strconv`'s and must be pinned from a live Go run, not asserted from memory | cannot run Go here (no toolchain in the reporting environment); design therefore *defines* semantics as "whatever `strconv.FormatFloat` does" + golden-generation task | Deliberately deferred to implementation (not guessed) |
| V10 | Negative `decimals` would silently flip `FormatFloat` to shortest-form | Go `strconv` documented prec<0 semantics (read `internal/ast/ast_expr.go:119–133`, the repo's own canonical float spelling, which guards the same way) | Confirmed — hence the hard error |
| V11 | `Inline` marshalling idioms (`{{arg0}}.(float64)`, `toInt64`) exist in generated-Go context | read `registry_codegen_string.go:273–283` (`_string_floatToStr`, `_string_intToStr`) | Confirmed |
| V12 | `std/string` has no per-function reference page to update; docs live in the module + stdlib pages | `ls docs/docs/reference/` (no `std-string.md`); `grep floatToStr docs/docs/reference/stdlib.md` → empty | Confirmed |

## Non-Goals

- **A printf/format-string mini-language** (`printf`, `%5.2f`) — template
  formatters pull in width/alignment/locale policy; `formatFixed` is the
  90% case. A future `std/format` can build on this builtin.
- **Thousands separators, padding, alignment, locales** — not a rounding
  concern; belongs in that future module if ever.
- **Scientific-notation control** — `floatToStr` already round-trips;
  `'e'` formatting has no reported consumer.
- **Changing `floatToStr` or `show`** — their round-trip contracts are
  correct and load-bearing (traces, JSON); only *additive* formatting here.
- **Introspectable "how many digits" API** — no demand.

## Timeline

**Day 1** (4h): Phase 1 (2h) + Phase 2 tests (2h).
**Day 2** (2h): Phase 2 finish + Phase 3 docs/example.
**Total: ~6 hours across 1–2 days** (estimate already 2x'd).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Tie-rounding surprises a report author (`0.125` → not `0.13`) | Low | Semantics defined as `strconv` (the normative formatter); golden pinned from Go; doc comment says "rounds the exact binary value" |
| Silent engine divergence (interpreter vs VM vs transpiled Go) | High | Single source of truth: VM calls the interpreter's `Impl`; `Inline` emits the identical `strconv.FormatFloat` call; parity tests on all three |
| `decimals < 0` treated as `%g` by a future refactor | Med | Hard error with a message naming the silent-flip hazard; `requires` at the wrapper; negative test case pins it |
| Name collision when a future `std/format` module lands | Low | `formatFixed` is specific enough; `format`-prefixed names would live in a different module |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure function of its args; byte-identical across all three engines by construction |
| A2: Replayability | 0 | No trace/effect changes |
| A3: Effect Legibility | 0 | Stays `pure` |
| A4: Explicit Authority | 0 | No capabilities touched |
| A5: Bounded Verification | +1 | `requires` is locally checkable; output pinned by goldens |
| A6: Safe Concurrency | 0 | No concurrency |
| A7: Machines First | +1 | Reports stop carrying noise digits that look like bugs to every reader, human and model |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Drops into `"${formatFixed(…)}"` interpolation and any fold producing a float |
| A11: Structured Failure | +1 | Contract violation aborts loudly naming the argument instead of silently flipping format |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): one Go function, three engines, same bytes
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access
- [x] A7 (Machines First): closes a report-noise gap, no sugar

## Related Documents

**Implemented (may inform design):**
- [M-CODEGEN-SUSTAINABILITY](../implemented/) lineage — the builtin↔codegen spec pairing this doc's `Inline` entry follows (`registry.go:10–23`)

**Planned (check for overlap):**
- [M-STD-AUDIO-WAV-READ](./m-std-audio-wav-read.md) — same DX report (sunholo-audio); its measure example prints through this function

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [std/string.ail](../../../std/string.ail) — `floatToStr` (`v0.5.10`) this complements
- Go `strconv.FormatFloat` — the normative semantics (prec `< 0` = shortest, hence the hard error)
- DX report: sunholo-audio (stapledons-godot `design/sunholo-audio`), 2026-10

## Future Work

- `std/format` module (width/alignment/grouping) building on this builtin.
- Fixed-notation variants of `show`/JSON number printing for report contexts
  (if ever requested — both currently round-trip-exact by design).

---

**Document created**: 2026-10-22
**Last updated**: 2026-10-22