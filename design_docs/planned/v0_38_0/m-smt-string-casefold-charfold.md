# M-SMT-STRING-CASEFOLD-AND-CHARFOLD: Verify `toUpper`/`toLower` and `foldChars` in contracts

**Status**: Planned
**Target**: v0.38.x (after the v0.38.0 stdlib/CLI batch)
**Priority**: P2 — Medium (moves a router's proof boundary from *canonical* input to *raw* input; no consumer is blocked)
**Estimated**: ~3 days (2 milestones, ~450 LOC)
**Dependencies**: None — reuses the HOF specialise-and-unroll machinery (`internal/smt/hof_inline.go`, `unroll.go`) and the `RecursiveDepth` bound already used for `map`/`filter`/`foldl`
**Consumer on record**: Daneel router port — `inbox_1789120118007_a83c58a6` (2026-09-11). Their priority order was membership > case-fold > foldChars; membership shipped the same day as `std/list.contains` (`066895252`), so this doc is the remaining two.
**Quorum trigger**: none of the four fired (no freeze items a human must ratify, no override of shared machinery — additive encoder entries — no cost/KPI/schema surface, premises all in-repo). Skipped.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Pure functions; encoding does not change runtime behaviour |
| A2: Replayability | 0 | No effect surface |
| A3: Effect Legibility | 0 | Unchanged |
| A4: Explicit Authority | 0 | Unchanged |
| A5: Bounded Verification | +2 | **Primary goal.** Two more pure string operations become provable under an explicit, stated bound (string length ≤ N), the same bound the list HOFs already carry |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | A contract that reads `enforce(raw) = decide(toUpper(raw))` is proven end-to-end instead of the model-facing boundary being one call short |
| A8: Minimal Syntax | 0 | No syntax; existing functions gain encodings |
| A9: Cost Visibility | 0 | Verification cost grows with the unroll depth, which is already a visible knob (`RecursiveDepth`) |
| A10: Composability | +1 | `foldChars` with a literal lambda composes with the existing `map`/`filter`/`foldl` inlining — one machinery, one bound |
| A11: Structured Failure | +1 | A string longer than the bound is a structured SKIP with the depth named, not a wrong proof |
| A12: System Boundary | 0 | — |

**Net Score: +5** → **Decision: Proceed**

### Hard Violation Check
- [x] A1 — no runtime change
- [x] A3 — no effects
- [x] A4 — no authority
- [x] A7 — contracts remain machine-checkable; the bound is stated in the skip reason

## Problem Statement

A closed-set decision in Daneel's router is proven today only over **already-canonical** input:

```ailang
export pure func decideLabel(u: string) -> string
  ensures { result == "" || result == "HELP" || result == "SUMMARISE" }   -- ✓ VERIFIED
-- but the function the model output actually reaches:
export pure func enforceLabel(raw: string) -> string = decideLabel(toUpper(raw))  -- ⚠ SKIPPED
```

Measured on HEAD (v0.37.2-21, 2026-09-11) with `ailang ai-check`:

| Function | Status | Reason (verbatim) |
|---|---|---|
| `enforce(raw)` using `toUpper` | skipped | `uses an unencodable builtin: std/string.toUpper` |
| `keep(s)` using `foldChars(\acc c. …, "", s)` | skipped | `uses higher-order functions; uses an unencodable builtin: std/string.foldChars; calls "foldChars" whose signature uses an unencodable type "a"` |

So the proof stops one call short of the boundary, on the two normalising steps every model-output check starts with: case-fold, then strip to an allowed alphabet.

**Why not "just" declare them:** Z3's string theory has no `str.to_upper`. `_str_upper`/`_str_lower` are on the encoder's *unencodable* list deliberately (`internal/smt/encodable.go:574`); `foldChars` is parametric in its accumulator (`(a, string) -> a`), and the encoder rejects any signature with an unencodable type variable.

## Goals

**Primary:** `ensures` clauses over `toUpper(x)` / `toLower(x)` and over `foldChars(<literal lambda>, <init>, x)` verify for strings of length ≤ N, and SKIP with the bound named otherwise.

**Success metrics:**
1. Daneel's `enforceLabel(raw) = decideLabel(toUpper(raw))` verifies.
2. `keepLabelChars` (`foldChars` stripping to `[A-Z-]`, string accumulator) and `isHex6` (`foldChars` counting, int/bool accumulator) verify.
3. A counterexample is still found when the contract is wrong (e.g. `ensures { result == toUpper(s) }` on the identity).
4. No existing verified contract regresses (`internal/smt` golden corpus, `examples/runnable/contracts/*`).

## High-Impact Decisions

| # | Decision | Options | Recommendation | Who | Change cost later |
|---|---|---|---|---|---|
| D1 | Case-fold scope | (a) ASCII letters only, everything else identity; (b) Unicode simple case mapping | **(a)** — Z3 strings are code-point sequences, a full Unicode table is thousands of `ite`s per character; the consumer's inputs are ASCII labels. The runtime `toUpper` handles Unicode; the *proof* is explicitly "for ASCII input", stated in the skip reason for non-ASCII literals and in the docs | agent | Low — (b) is a bigger table in the same slot |
| D2 | How the per-character map is encoded | (a) one `define-fun upc ((c String)) String` as a 26-way `ite` chain, applied character-wise by an unrolled recursion; (b) `declare-fun` + 52 axioms | **(a)** — total, no quantifiers, stays in the decidable fragment the encoder already uses | agent | Low |
| D3 | String length bound | Reuse `RecursiveDepth` (default 3 for HOF specialisation) vs a separate string bound | **Separate default for strings, 16**, still capped by `RecursiveDepth` semantics — depth 3 is useless for a label like `SUMMARISE` (9 chars). The bound is named in every SKIP | agent | Low — it is one constant |
| D4 | `foldChars` accumulator sorts | int, bool, string (infer from `init`), else SKIP | Infer from the `init` literal's type; anything else SKIP with reason "accumulator type X not encodable" | agent | Medium — extending to records needs sort plumbing |

### Design Freeze
- [x] D1 — ASCII-only case map (agent-resolvable; documented limitation)
- [x] D2 — total `define-fun` ite-chain
- [x] D3 — string bound 16, named in SKIP
- [x] D4 — accumulator from `init` type; else SKIP

No decision needs a human ruling before sprint-executor starts.

## Solution Design

### Overview

Both features are the same shape the encoder already has for lists: **specialise a recursive definition, unroll it to a bound, and SKIP beyond the bound.** `map`/`filter`/`foldl` with a literal lambda already go through `hof_inline.go → hof_specialize.go → UnrollRecursiveFunction` (`hof_inline.go:357`) at `RecursiveDepth`. This doc adds:

1. a **string recursion scheme** — recurse on `(str.at s i)` / `i+1` up to `(str.len s)`, bounded by N — used by both features;
2. **`toUpper`/`toLower`** as fixed specialisations over that scheme with a total per-character map;
3. **`foldChars`** as a fourth `HOFKind` over that scheme, with the accumulator sort inferred from `init`.

### Architecture

```
ensures { result == "" || contains(allowed, toUpper(raw)) }
                                   │
   encodable.go: _str_upper no longer "unencodable" when the arg is a string-sorted expr
                                   │
   codegen_apps.go: toUpper(s) → (_str_upper_N s)
                                   │
   string_unroll.go (NEW):
     (define-fun _upc ((c String)) String
        (ite (= c "a") "A" (ite (= c "b") "B" … c)))          ; total; non-letters identity
     (define-fun _str_upper_0 ((s String)) String "")          ; beyond bound → handled as SKIP, see below
     (define-fun _str_upper_k ((s String)) String
        (ite (= (str.len s) 0) ""
             (str.++ (_upc (str.at s 0)) (_str_upper_{k-1} (str.substr s 1 (- (str.len s) 1))))))
     …up to k = N
   + assertion (<= (str.len s) N) on every string parameter that flows into an unrolled op
     (same technique list_unroll.go uses for reverse/take/drop)
```

`foldChars(λacc c. body, init, s)`:

```
hof_inline.go: new HOFKind HOFFoldChars, matched on std/string.foldChars / _str_foldChars
               with a literal lambda (same literal-lambda rule as foldl)
hof_specialize.go: SpecializeFoldChars →
     f_k(acc, s) = ite (= (str.len s) 0) acc
                       (f_{k-1}(body[acc := acc, c := (str.at s 0)], (str.substr s 1 …)))
   accumulator sort from init: Int | Bool | String (D4)
   unrolled by UnrollRecursiveFunction at the string bound (D3)
```

**Soundness of the bound.** As with lists, the proof is over inputs of length ≤ N. The encoder already asserts the length bound on unrolled list parameters; strings get the identical treatment, and the verify result names it: `verified (strings ≤ 16 chars)`. A contract whose counterexample needs a longer string is therefore *not* claimed proven — the bound is part of the reported status, never silent.

### Conflict Surface

This touches `internal/smt/` only (encoder), not parser/typechecker/runtime. Still enumerated, because the encoder's *rejection* logic is shared machinery:

| Position extended | What already lives there | Disambiguation | Must still work |
|---|---|---|---|
| `firstUnencodableBuiltin` (`encodable.go:580`) — `_str_upper`/`_str_lower` currently return as blockers | every other `_str_*` without a mapping (`_str_trim`, `_str_split`…) stays a blocker | only the two names move from the unencodable set to `StdlibStringToSMT`; the list is explicit, nothing pattern-based | `examples/runnable/contracts/string_ops.ail` (uses `startsWith`/`length`), `internal/smt/codegen_string_list_test.go` corpus |
| `hasHigherOrder`/`AllHigherOrderIsInlinable` — `foldChars` is currently "uses higher-order functions" | `map`/`filter`/`foldl` with literal lambdas are inlinable; with a *variable* function arg they are rejected | `HOFFoldChars` follows the same literal-lambda rule; a variable `f` stays rejected with the existing reason | `examples/runnable/contracts/hof_verify.ail` |
| `matchHOFCall` name table (`hof_inline.go:53-60`) — keyed by `foldl`, `foldl_List`, … | user functions named `foldChars` in a non-std module | the table is keyed on the resolved `$builtin`/`std/string` ref, not the bare name — verify by reading `matchHOFCall`'s ref check (V6) | a user module exporting its own `foldChars` must not be inlined |
| `RecursiveDepth` option (`codegen.go:93-96`) | list unroll depth 1–10 | strings get their own default (16) but honour an explicit `RecursiveDepth` if larger; document | existing `--depth` behaviour on list contracts unchanged |

**Deliberate change:** none — every currently-verified contract verifies identically; two SKIP reasons become proofs.

### Implementation Plan

**M1 — `toUpper`/`toLower` (≈200 LOC)**
- `internal/smt/string_unroll.go` (new): `_upc`/`_lwc` total maps; `GenerateStringCaseUnrolling(op, depth)`; length-bound assertion helper shared with M2.
- `internal/smt/types.go`: `StdlibStringToSMT["toUpper"] = "_str_upper"`, `"toLower" = "_str_lower"`; builtin specs with an `UnrolledStringMode` flag (mirrors `ContainsMode`/`SubstrMode`).
- `internal/smt/encodable.go`: remove the two names from the unencodable set; add the string-bound SKIP reason.
- Tests: encode-shape goldens; live-Z3 (guarded by `smt.Z3Available()` — **Windows CI has no z3**) proving `enforce` from the problem statement, a deliberate counterexample, and a non-ASCII literal producing the documented SKIP.

**M2 — `foldChars` (≈250 LOC)**
- `internal/smt/hof_inline.go`: `HOFFoldChars`; match on `std/string.foldChars` and `$builtin._str_foldChars` with a literal lambda.
- `internal/smt/hof_specialize.go`: `SpecializeFoldChars`; accumulator sort from `init` (D4).
- `internal/smt/encodable.go`: `foldChars` with a literal lambda is inlinable; the "unencodable type a" signature rejection is bypassed for the inlined call only (the wrapper's signature is never encoded — the specialisation is).
- Tests: `keepLabelChars` (string acc) and `isHex6` (int acc) from the consumer, both verified; a variable-`f` call still rejected; bound exceeded → SKIP.

### Files to Modify/Create

- `internal/smt/string_unroll.go` — NEW (~120 LOC): case maps, string recursion scheme, bound assertion
- `internal/smt/types.go` — +2 stdlib mappings, +2 builtin specs (~20 LOC)
- `internal/smt/encodable.go` — move two names out of the unencodable set; string-bound SKIP reason (~30 LOC)
- `internal/smt/hof_inline.go` — `HOFFoldChars` kind + matcher (~50 LOC)
- `internal/smt/hof_specialize.go` — `SpecializeFoldChars` (~80 LOC)
- `internal/smt/string_unroll_test.go` — NEW (~150 LOC, z3-guarded)
- `docs/docs/guides/contracts.mdx` — encodable-operations list gains `toUpper`, `toLower`, `foldChars` (literal lambda, string/int/bool accumulator, ≤16 chars)
- `examples/runnable/contracts/string_casefold_verify.ail` — NEW + manifest entry

## Examples

### Example 1: the consumer's boundary function

```ailang
import std/string (toUpper)
import std/list (contains)

export pure func enforceLabel(raw: string, allowed: [string]) -> string
  ensures { result == "" || contains(allowed, result) }
{
  let u = toUpper(raw);
  if contains(allowed, u) then u else ""
}
-- today:  ⚠ SKIPPED  uses an unencodable builtin: std/string.toUpper
-- after:  ✓ VERIFIED (strings ≤ 16 chars)
```

### Example 2: strip to an alphabet, then decide

```ailang
import std/string (foldChars)

export pure func keepLabelChars(s: string) -> string
  ensures { length(result) <= length(s) }
{
  foldChars(\acc c. if c == "-" || (c >= "A" && c <= "Z") then concat_String(acc, c) else acc, "", s)
}
-- today:  ⚠ SKIPPED  uses higher-order functions; … foldChars … unencodable type "a"
-- after:  ✓ VERIFIED (strings ≤ 16 chars)
```

## Success Criteria
- [ ] Both problem-statement probes verify; the counterexample probe reports a counterexample
- [ ] SKIP reasons name the bound (`strings ≤ N chars`) and, for case-fold, the ASCII scope
- [ ] `go test ./internal/smt/...` green on macOS with z3 **and** on Windows CI without it (guards, not failures)
- [ ] No change in status for any contract in `examples/runnable/contracts/` (before/after diff of `ai-check --json`)
- [ ] `contracts.mdx` lists the three operations with their bound; CHANGELOG entry

## Testing Strategy
- Encoding goldens (no z3): the emitted SMT-LIB for each op at depth 3, byte-compared.
- Live z3 (guarded): verified / counterexample / skip triples for each feature.
- Regression: the full `internal/smt` corpus plus `ai-check` over `examples/runnable/contracts/` before and after — statuses identical.
- Windows: every new test that invokes z3 is behind `smt.Z3Available()` (rule #10 of sprint-executor; the M-SMT-CALLEE-SORT-GATE red was exactly this).

## Deferred Decisions
- Unicode case mapping (D1b) — only if a consumer has non-ASCII closed sets.
- Record/ADT accumulators for `foldChars` (D4) — needs the record-sort plumbing from `codegen_records.go`; no consumer.
- `chars(s)`/`join` as string↔list bridges — a different feature (list-of-char sorts).

## Non-Goals
- Making `trim`, `split`, `replace` encodable — separate asks, none filed.
- Changing runtime `toUpper`/`toLower` (they stay Unicode-correct).
- Raising the general `RecursiveDepth` ceiling.

## Timeline
- Day 1: M1 (case-fold), goldens, z3 tests
- Day 2: M2 (foldChars), consumer probes
- Day 3: regression diff over the contracts corpus, docs, example, CHANGELOG

## Risks & Mitigations
- **Z3 blow-up at depth 16 with nested `str.++`** — Mitigation: the case map is a `define-fun` (inlined once), recursion is linear in N; measure solve time on the consumer probes and lower the default if >2s.
- **A bound-exceeding input silently "verified"** — Mitigation: the length assertion is on the *parameter*, so the proof is explicitly conditional and the status string carries the bound; tested by the "bound exceeded → SKIP" case.
- **Name-keyed HOF matching catches a user's own `foldChars`** — V6: the matcher keys on the resolved module ref; a user-module `foldChars` is never inlined.

## Related Documents
- [M-SMT-INTERP-SHOW](../../implemented/v0_36_0/m-smt-interp-show.md) — the last encoder extension (string interpolation/`show`), same file set and test pattern
- [m-dx-utf8-string-ops](../../implemented/v0_10_0/m-dx-utf8-string-ops.md) — the runtime string ops this encodes (neural 0.35 — related, not overlapping)
- `std/list.contains` (`066895252`, 2026-09-11) — the membership half of the consumer's ask, shipped

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | `toUpper` in a contract is SKIPPED with `unencodable builtin: std/string.toUpper` | `ailang ai-check` on the problem-statement probe, HEAD v0.37.2-21 | Confirmed (verbatim in Problem Statement) |
| V2 | `foldChars` with a literal lambda is SKIPPED for HOF + unencodable + type-var reasons | same probe | Confirmed (verbatim) |
| V3 | `_str_upper`/`_str_lower` are on the encoder's unencodable list | `internal/smt/encodable.go:574` comment names `_str_upper` as the example | Confirmed |
| V4 | `map`/`filter`/`foldl` with a literal lambda are specialised and unrolled at `RecursiveDepth` (default 3) | `hof_inline.go:310` (`depth = 3`), `:357` (`UnrollRecursiveFunction`) | Confirmed |
| V5 | `foldl` inlining hardcodes an `int` accumulator | `hof_inline.go:343-348` (`_hof_acc` typed `int`) — hence D4 is new work, not reuse | Confirmed |
| V6 | HOF matching keys on the resolved ref, not the bare name | `matchHOFCall` (`hof_inline.go:142-152`): checks `vg.Ref.Module == "$builtin"` then `== "std/list"` before the name tables — M2 adds a `"std/string"` arm with `foldChars` and a `$builtin` entry for `_str_foldChars` | Confirmed — a user module's own `foldChars` is not matched |
| V7 | Z3 has no `str.to_upper` | SMT-LIB Unicode strings theory (`str.<`, `str.at`, `str.substr`, `str.replace`, `str.to_code`…; no case ops) | Confirmed by reference |
| V8 | No existing planned/implemented doc covers this | create-script search: max neural similarity 0.35 (implemented), 0.33 (planned) | Confirmed |
| V9 | `std/list.contains` in `ensures` verifies today | `ailang ai-check` on `pick(candidate, allowed)`, 2026-09-11 | Confirmed (`verified: 1`) |

## References
- `internal/smt/hof_inline.go`, `hof_specialize.go`, `unroll.go`, `list_unroll.go` — the machinery reused
- SMT-LIB Unicode Strings theory — the operator vocabulary available
- Consumer message `inbox_1789120118007_a83c58a6`

## Future Work
- Unicode case map behind a flag
- `chars`/`join` bridges so list contracts and string contracts compose
