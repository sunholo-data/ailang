# M-STD-STRING-FORMAT-FIXED: bounded fixed-decimal reporting

**Status**: Approved — implementation and integration in progress (Mark, 2026-10-23)
**Target**: v0.54.18 candidate
**Priority**: P1
**Estimated**: 1–2 engineering days including all-engine error parity
**Dependencies**: None
**Revision**: 2026-10-23; continues pending PR #1931 at the same path

## Problem and provenance

[Issue #1930](https://github.com/sunholo-data/ailang/issues/1930) reports
`floatToStr(floor(-27.0612/0.01+0.5)*0.01)` printing
`-27.060000000000002`. A live checked AILANG fixture reproduces this on the
interpreter and strict bytecode VM. `floatToStr` correctly prints the value it
receives; reports need a separate decimal-precision API. Today's audio follow-up
`inbox_1792742861419_98f58254` explicitly retains this requirement.

The original proposal in [PR #1931](https://github.com/sunholo-data/ailang/pull/1931)
used an unguarded generated-Go Inline and unbounded precision. This revision
requires ordinary runtime guards in every engine. It also corrects the claim
that precision above 17 only adds zeros: live Go prints
`FormatFloat(0.1, 'f', 20, 64) == "0.10000000000000000555"`.

## Goal and high-impact decisions

Add the proposed API `std/string.formatFixed(x: float, decimals: int) -> string`.
Finite values use exactly `strconv.FormatFloat(x, 'f', decimals, 64)` for an
accepted precision. This is formatting a binary float, not exact decimal
arithmetic. Existing floatToStr/show behavior stays intact.

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| One normative Go formatter | Prevents engine-dependent report output | human approval | design | high |
| Precision 0..1000 inclusive, hard error outside it | Bounds allocation and blocks negative-precision shortest-form fallback | human approval | design | med |
| Keep string return and fail loudly on invalid precision | Determines error-handling contract | human approval | design | med |
| NaN/Inf strings and signed zero follow strconv | Keeps anomalies observable | human approval | design | med |

### Design freeze

- [x] Approve precision cap and string/error contract.
- [x] Approve shared semantics for interpreter, strict VM and generated Go.

The 1000-digit cap is a proposed resource policy, not a floating-point precision
limit: even the largest finite float64 then needs only roughly 1311 characters
including sign and decimal point. The cap must be documented, not clamped.

## Solution and implementation phases

Register a pure `_str_formatFixed` builtin with float,int -> string. Check the
precision before calling strconv, even when runtime contracts are disabled.
The std wrapper may also declare a requires clause to document the range; that
annotation is not the runtime guard. Wrong Go argument types use the standard
builtin diagnostic convention.

Use a GoHelperSpec in `internal/builtins/registry_codegen_string.go` that applies
the same range check before formatting. The previous raw Inline would allow
negative precision to silently select shortest form in generated Go. Reuse the
same guard/format helper where the codegen architecture permits; otherwise pin
the duplicated boundary with cross-engine invalid-input tests. Generated Go may
use its established loud runtime-error mechanism; it must exit unsuccessfully
and name decimals/range, rather than return a plausible string.

| Input | Required result |
|---|---|
| reported rounded -27.0612 value, 2 | "-27.06" |
| 42, 2 | "42.00" |
| 0.125, 2 | "0.12" (exact tie, nearest even) |
| 2.675, 2 | "2.67" (binary value is below tie) |
| 0.1, 20 | "0.10000000000000000555" |
| negative zero, 2 | "-0.00" |
| NaN / positive or negative infinity | "NaN" / "+Inf" / "-Inf" |
| any x with decimals -1 or 1001 | hard error naming decimals |

1. Builtin and wrapper, about half a day: `std/string.ail`, new
   `internal/builtins/string_format.go`, metadata and builtin type golden.
2. Engine/codegen wiring and fixtures, about one day:
   `registry_codegen_string.go`, builtin tests, strict VM and codegen harness.
3. Docs/example and changelog fragment, about half a day. Use the actual landed
   version for Since metadata. Helper names and fixture placement are delegated
   to the implementer; no format-string language, locale, padding or separators.

## Conflict surface and acceptance

The fresh export and builtin keys share std/string's registry/codegen namespaces;
inspect duplicate StdlibName claims and adapted scalar-call coverage at sprint
time. No syntax, parser, type inference, or effect changes. Existing
`std/string.ail`, string conversion tests and codegen harness are regression
fixtures. No existing output changes are intentional.

- [ ] Every table row agrees on interpreter, strict VM and generated Go.
- [ ] Negative and excessive precision fail with contracts disabled, before
      formatting/allocation. No fallback to floatToStr.
- [ ] Precision 0 and 1000 succeed; large magnitudes, subnormals and signed zero
      are covered without hand-written approximations of the binary value.
- [ ] Existing string behavior, required test/lint gates, docs and changelog pass.

## Verification log (2026-10-23)

| Claim | Evidence | Result |
|---|---|---|
| Existing conversion implementation | Read string_convert.go | strconv 'g', -1, 64 |
| Proposed identifiers are unallocated in runtime/std code | rg formatFixed in std/string.ail and string registry/conversion files; existing floatToStr is positive control | No formatFixed implementation |
| Repro and current array/audio constructs type-check | ailang check on scratch audit fixture; both execution engines | Printed -27.060000000000002 identically |
| Rounding and long-precision examples | go1.26.10 run of strconv.FormatFloat with math negative-zero/NaN/Inf | Table values above, measured locally |
| Guarded codegen helper mechanism exists | Read GoHelperSpec examples in registry_codegen_string.go | Helper bodies can check args before formatting |
| PR state/provenance | gh pr list/diff 1931; canonical task-68aabd73 | Design awaits approval; no implementation |

Source inspected at `731fa1b`; installed AILANG `v0.54.17-dirty / 8f58117`.
Proposed signatures above are specifications, not examples claimed to compile.

## Axiom compliance and risks

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | +1 | Normative formatter and rounding |
| A2 Replayability | +1 | Stable report bytes |
| A3 Effect legibility | 0 | Pure operation |
| A4 Explicit authority | 0 | No new capability |
| A5 Bounded verification | +1 | Finite precision range |
| A6 Safe concurrency | 0 | Stateless |
| A7 Machines first | +1 | Explicit precision and errors |
| A8 Minimal syntax | 0 | Function, no mini-language |
| A9 Cost visibility | +1 | Output size bounded |
| A10 Composability | +1 | Fits package reports |
| A11 Structured failure | 0 | Standard loud builtin failure |
| A12 System boundary | 0 | Existing std boundary |

Net +6; no A1/A3/A4/A7 hard violation. Primary risk is losing validation in
generated Go; ordinary-run failure tests are a release criterion. NaN strings
are display evidence, not validation of DSP inputs.

## Related documents

- [WAV/PCM reader](m-std-audio-wav-read.md), the other half of #1930.
- [Verification coverage](../m-contract-verification-coverage.md): separately
  scoped reporting; no formatter dependency.
- [Design axioms](../../../docs/docs/references/axioms.mdx).

**Created**: 2026-10-22 (PR #1931). **Revised**: 2026-10-23.
