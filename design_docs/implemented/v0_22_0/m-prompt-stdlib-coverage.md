# M-PROMPT-STDLIB-COVERAGE

**Status**: PLANNED (v0.17.0 teaching prompt revision)
**Date**: 2026-05-21
**Type**: Teaching-prompt expansion / no compiler changes

## Problem

The v0.16.0 teaching prompt at [`prompts/v0.16.0.md`](../../../prompts/v0.16.0.md) documents a subset of the stdlib that's smaller than what's actually shipped. The VeraBench integration (M-VERA-BENCH) surfaced one concrete case — `std/bytes.byteAt` existed in the binary but was missing from the prompt, causing Claude Haiku 4.5 to fail `VB_T2_013_get_char_code` even though the builtin was available.

The fix for `byteAt` shipped in v0.21.0, but it raised the obvious question: **how many other stdlib functions are invisible to an LLM consulting the teaching prompt?**

## Audit findings (2026-05-21)

Audited 42 std modules / ~280 exported functions against the v0.16.0 prompt. Full report saved as agent output; condensed here:

### MUST ADD (LLMs will hit these)

| # | Function | Module | Why |
|---|----------|--------|-----|
| 1 | `now`, `sleep` | clock | Entire `std/clock` module is unmentioned in the prompt. Time-since is a basic primitive. |
| 2 | `getOrElse`, `isSome`, `isNone` | option | 3 of 6 missing — these are the idiomatic accessors LLMs reach for. |
| 3 | `unwrap`, `isOk`, `isErr`, `mapErr` | result | 4 of 6 missing — `Result` without `isOk`/`unwrap` is unusable in idiomatic code. |
| 4 | `sin`, `cos`, `tan`, `PI`, `E`, `exp`, `log10`, trig family | math | Half the math module (trig + constants) undocumented. Breaks any graphics/sim/numeric task. |
| 5 | `dedup`, `union`, `intersect`, `difference`, min/max family | list | Set ops + typed min/max — common idioms LLMs hand-roll incorrectly otherwise. |
| 6 | `concatList` | bytes | Companion to documented `concat`. |
| 7 | `make`, `append`, `unsafeGet` | array | Core array construction primitives. |
| 8 | `words`, `splitAny`, `charCode`, `foldChars` | string | `words` and `splitAny` are extremely common. |
| 9 | `repair`, `asObject`, `getStringArray`/`getNumberArray` | json | LLM-output JSON repair + typed accessors. |
| 10 | All 21 of 22 datetime exports | datetime | Module effectively undocumented (1/22). `formatRFC3339`, `parseISODate`, `addDays` are bread-and-butter. |
| 11 | `sha256Hex`, `hmacSha256`, `constantTimeEqual` | crypto | Standard hashing — LLMs will guess imports otherwise. |
| 12 | `writeBytes` | io | Companion to documented file I/O. |

### SHOULD ADD (next revision)

- `env.hasEnv`
- `fs.*Result` family (5 functions — entire Result-returning fs API hidden)
- `net.urlEncode`, `urlEncodeForm`, `httpRequestBytes`
- `rand.uuid4`, `rand_float`, `rand_bool`
- `iter.isStop`, `fromFoldStep`
- `xml` module — 19 of 25 missing including constructors and serializers

### SKIP (low priority / specialized)

- `ai`, `cognition`, `dom`, `embedding`, `extension`, `game`, `sem`, `sharedindex`, `sharedmem`, `simhash`, `smoke`, `stream`, `trace_test`, `jwt.checkAudience`, `package` — document only when a specific benchmark or user need surfaces them.

## Module-level gap counts (high priority)

| Module | Documented / Total | Missing count |
|--------|-------------------|---------------|
| clock | 0/2 | 2 (!) |
| datetime | 1/22 | 21 (!) |
| math | 10/21 | 11 |
| list | 25/35 | 10 |
| string | 26/30 | 4 |
| result | 2/6 | 4 |
| crypto | 1/5 | 4 |
| option | 3/6 | 3 |
| array | 7/10 | 3 |
| bytes | 10/11 | 1 |
| io | 4/5 | 1 |
| map | 10/10 | 0 (clean) |

## Recommended approach

1. **Ship a v0.16.1 patch prompt** (no syntax changes — purely additive stdlib coverage):
   - Add the MUST-ADD table contents to existing module sections
   - Add a full `std/clock` mini-section (currently absent)
   - Expand `std/datetime` from a single line to a proper reference
   - Backfill `std/option` and `std/result` accessor functions

2. **Acceptance**: re-run VeraBench against Claude Haiku 4.5 (current 100% post-v0.21.0). Goal: no regression. Stretch: pick 5-10 problems from the audit's missing functions and write benchmark-style test cases to confirm a model with the v0.16.1 prompt can solve them where v0.16.0 fails.

3. **Versioning**: bump to `v0.16.1` in `prompts/versions.json` (additive patch — same syntax, broader coverage). Recompute hash. Mention in changelog as a non-breaking prompt improvement.

## Risk

Low. This is documentation-only — adding signatures to a markdown prompt. No compiler or stdlib changes. The only risk is prompt token-budget bloat; the prompt is currently ~13k tokens — adding ~50 function signatures should add <500 tokens (rough estimate 10 tokens each).

## Out of scope

- Implementing new stdlib functions (none needed — these all already exist)
- Restructuring the teaching prompt (separate sprint if needed)
- Documenting the SKIP-tier modules

## References

- v0.21.0 release notes: `changelogs/v0.10-current.md` (M-BYTES-TOINTS-BYTEAT motivation)
- VeraBench results showing the gap impact: https://github.com/aallan/vera-bench/pull/70
- Current prompt: `prompts/v0.16.0.md`
- Stdlib source: `std/*.ail`
