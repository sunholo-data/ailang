# Prompt Gap: Numeric & String Conversion Functions (`parseInt`/`toFloat`/`str` hallucinations)

> **📊 FREQUENCY UNVERIFIED — pending nightly-eval segmentation.** This doc was
> generated from a nightly-eval design-doc trigger (task-11e68c7e, 2026-06-05). The
> *gap itself* is verified with live `ailang check` (see Evidence), but the failure
> *frequency* against the latest Apr–Jun 2026 runs has NOT yet been segmented. The
> author could not reach the nightly-eval result payload from the task environment.
> **Confirming the frequency is a Design-Freeze gate (below) before this proceeds to
> a sprint.** Do not cite a percentage until it is measured.

**Status**: Planned
**Target**: v0.24.0
**Priority**: P2 (Low — prompt-only fix; high leverage if frequency confirms)
**Estimated**: 0.5 day
**Dependencies**: None. Complementary to [M-PRELUDE-OPTION-RESULT](m-prelude-option-result.md) (structural import-removal) and sibling prompt-gap docs ([Option/None](m-prompt-option-none-idiom.md), [Split](m-prompt-split-list-operations.md), [String concat](m-prompt-string-concat-plusplus.md)).

## Problem Statement

Models converting between `int`, `float`, and `string` reach for the names they
learned from Python / JavaScript / Rust, and those names do not exist in AILANG.
The canonical functions *do* exist — but under different names, in different
modules, with **inconsistent import requirements**. The result is a cluster of
`undefined variable` failures on a near-universal operation.

**Verified failing idioms** (all produce `undefined variable`, see Evidence):

```ailang
parseInt("42")    -- ❌ undefined variable: parseInt   (real name: stringToInt)
toFloat(5)        -- ❌ undefined variable: toFloat     (real name: intToFloat)
str(42)           -- ❌ undefined variable: str         (real name: intToStr / show)
floor(3.7)        -- ❌ undefined variable: floor       (exists, but needs `import std/math`)
```

**Root cause — three overlapping friction points:**

1. **Wrong names.** AILANG uses `<from>To<To>` (`floatToInt`, `intToFloat`,
   `stringToInt`, `intToStr`), not the `parse*` / `to*` / `str()` / `int()`
   families models default to.

2. **Inconsistent import boundary** (the subtle one). `float ↔ int` conversions
   are **core builtins, no import**. `string ↔ X` conversions live in
   **`std/string`** and `floor`/`ceil`/`round` live in **`std/math`** — both
   require an explicit import. A model that successfully writes `floatToInt(x)`
   with no import then writes `stringToInt(s)` with no import and gets
   `undefined variable` — the inconsistency actively misleads.

3. **`Option` return is silent.** `stringToInt`/`stringToFloat` return
   `Option[int]`/`Option[float]` (parsing can fail), not a bare `int`/`float`.
   Even a model that finds the right name then mis-types the result by assuming a
   bare number.

The teaching prompt does not contain a conversion reference table, so models
discover this by trial-and-error across multiple failed turns.

**Impact:**

- AI models — hard failure on first use of any numeric/string conversion, which
  appears in a broad class of tasks (config parsers, calculators, CSV/log
  processing, anything reading numbers from text input).
- Compounds with the Option/None gap: the *correct* fix for `stringToInt` also
  requires `import std/option` to handle the `Option` result.

## Goals

**Primary Goal:** Eliminate `undefined variable` failures on numeric/string
conversion by adding a single canonical conversion reference table to the
teaching prompt, with explicit import lines and the `Option` return called out.

**Success Metrics:**

- Teaching prompt contains a conversion table covering all 8 canonical functions
  (below), each with its import requirement and return type.
- Conversion-heavy benchmarks (config/CSV/number-parsing tasks) stop emitting
  `undefined variable: parseInt | toFloat | str | floor`.
- The `Option` return of `stringToInt`/`stringToFloat` is shown unwrapped in at
  least one prompt example.

## The Canonical Conversion Table (all verified 2026-06-05)

| From → To | Canonical function | Import required | Returns |
|-----------|--------------------|-----------------|---------|
| float → int | `floatToInt(x)` | **none** (core) | `int` |
| int → float | `intToFloat(n)` | **none** (core) | `float` |
| string → int | `stringToInt(s)` | `import std/string` | **`Option[int]`** |
| string → float | `stringToFloat(s)` | `import std/string` | **`Option[float]`** |
| int → string | `intToStr(n)` | `import std/string` | `string` |
| float → string | `floatToStr(f)` | `import std/string` | `string` |
| any → string (debug) | `show(x)` | **none** | `string` |
| floor / ceil / round | `floor(x)` / `ceil(x)` / `round(x)` | `import std/math` | `float` |

**Hallucinated names to explicitly warn against:** `parseInt`, `parseFloat`,
`toFloat`, `toInt`, `toString`, `str`, `int()`, `float()`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix in the prompt only — no renames, no new aliases | Renaming/aliasing conversion functions is a breaking stdlib change with a large blast radius; the names are already correct AILANG | human | design | low (prompt) / high (if aliasing chosen instead) |
| Present conversions as ONE table, not scattered examples | Models read top-to-bottom; a single lookup table is the format they can scan and copy | agent | implementation | low |
| Show `stringToInt` result unwrapped (it is `Option`) | Finding the name is necessary but not sufficient — the `Option` return is the second failure | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Confirm canonical names and import boundaries (verified via `ailang check`, 2026-06-05 — see Evidence).
- [x] Confirm `stringToInt`/`stringToFloat` return `Option[_]` (verified: `std/string.ail:40-41`).
- [ ] **Segment the latest nightly-eval (Apr–Jun 2026) for `undefined variable: (parseInt|parseFloat|toFloat|toInt|str|toString|floor|ceil|round)` and record the real frequency + affected benchmarks.** Until done, the priority stays P2 and the frequency banner stays UNVERIFIED. (This is the gate the task environment could not satisfy automatically.)
- [ ] Decide scope overlap with [M-PRELUDE-OPTION-RESULT](m-prelude-option-result.md): if string conversions are *also* promoted to the prelude/core later, the import-boundary half of this doc becomes moot. Confirm this doc ships as prompt-only regardless.

## Deferred Decisions

- Whether to *also* promote `std/string` conversion functions (`stringToInt`,
  `intToStr`, …) into core/prelude so the import boundary disappears entirely —
  the structural sibling to this prompt fix. Larger change; out of scope here,
  noted in Future Work.
- Whether to add a linter hint (`did you mean stringToInt?` on `parseInt`) — see
  Future Work; out of scope.
- Exact wording of the "hallucinated names" warning — agent may choose, provided
  it names `parseInt`, `toFloat`, and `str` explicitly.

## Solution Design

### Overview

Add one conversion-reference section to the teaching/system prompt. No compiler
or stdlib changes. Two pieces:

1. The canonical conversion table (above), with import column and return-type column.
2. A short "common mistakes" callout listing the hallucinated names and their
   correct replacements, plus one worked example that unwraps the `Option` from
   `stringToInt`.

### Architecture

Only `prompts/` files are affected. The change is additive text.

**Components:**
1. **Conversion table** — placed near the existing stdlib/idiom section.
2. **Common-mistakes callout** — `parseInt → stringToInt`, `toFloat → intToFloat`,
   `str → intToStr`/`show`, and "remember to `import std/string` / `std/math`".
3. **Worked example** — parse a string to int and handle the `Option`.

### Implementation Plan

**Phase 1: Locate and update prompt** (~2 hours)
- [ ] Find the stdlib/idiom section in `prompts/` (grep for existing conversion or `show` references).
- [ ] Insert the conversion table verbatim (it is already verified).
- [ ] Add the common-mistakes callout and the `Option`-unwrapping example.

**Phase 2: Verify** (~1 hour)
- [ ] Run `ailang prompt` and confirm the table and callout render.
- [ ] Compile every code snippet in the new section with `ailang check` (they are pre-verified; re-confirm after editing).
- [ ] Audit `examples/` for any conversion examples missing imports; fix if found.

### Files to Modify/Create

**Modified files:**
- `prompts/<stdlib-or-idioms-section>.md` (+~20/−0 LOC) — conversion table, callout, example.

**Possibly also modified:**
- `examples/*.ail` — any file using `stringToInt`/`intToStr`/`floor` without the
  required import (`grep -ln "stringToInt\|intToStr\|floatToStr\|stringToFloat" examples/`).

## Examples

### Common mistakes — names that do not exist (all verified `undefined variable`)

```ailang
-- WRONG: these are Python/JS/Rust names, not AILANG
let n = parseInt("42")      -- ❌ undefined variable: parseInt
let x = toFloat(5)          -- ❌ undefined variable: toFloat
let s = str(42)             -- ❌ undefined variable: str
let f = floor(3.7)          -- ❌ undefined variable: floor   (exists, needs import std/math)
```

### Correct — float ↔ int need no import (core builtins)

```ailang
module example/convert

export func main() -> () ! {IO} {
  let half = intToFloat(7) / 2.0 in   -- int → float, no import
  let n = floatToInt(half) in          -- float → int, no import
  println(show(n))                     -- 3
}
```

### Correct — string conversions need `import std/string`, and parsing returns `Option`

```ailang
module example/parse

import std/string (stringToInt, intToStr)
import std/option (Option, Some, None)

-- stringToInt returns Option[int] — parsing can fail
export func main() -> () ! {IO} {
  match stringToInt("42") {
    Some(n) => println(intToStr(n * 2)),   -- "84"
    None    => println("not a number")
  }
}
```

### Correct — rounding lives in `std/math` and returns `float`

```ailang
module example/round

import std/math (floor, round)

export func main() -> () ! {IO} {
  let _ = println(show(floor(3.7))) in   -- 3.0  (float, not int)
  println(show(round(3.5)))              -- 4.0
}
```

### Callout for the teaching prompt

> **Converting between `int`, `float`, and `string`:**
> - `float→int` / `int→float`: `floatToInt(x)` / `intToFloat(n)` — **no import**.
> - `string→int` / `string→float`: `stringToInt(s)` / `stringToFloat(s)` from
>   `std/string` — these return **`Option`** (parsing can fail), so `match` them.
> - `int→string` / `float→string`: `intToStr(n)` / `floatToStr(f)` from
>   `std/string`; or `show(x)` for any value (no import).
> - `floor` / `ceil` / `round`: from `std/math`, return `float`.
>
> There is **no** `parseInt`, `toFloat`, `str()`, `int()`, or `toString` in AILANG.

## Success Criteria

- [ ] Teaching prompt contains the conversion table with import + return-type columns.
- [ ] Prompt lists the hallucinated names (`parseInt`, `toFloat`, `str`, …) with replacements.
- [ ] At least one prompt example unwraps the `Option` from `stringToInt`.
- [ ] Nightly-eval segmentation recorded (Design-Freeze gate) and frequency banner updated to RECENT-VERIFIED.
- [ ] All snippets in the new section compile (`ailang check`).
- [ ] `make verify-examples` passes.

## Testing Strategy

**Manual:**
- Re-run conversion-heavy benchmarks (config/CSV/number-parsing) with the updated prompt; confirm `undefined variable` on conversion names is gone.

**Pre-verified snippets** (re-run after editing to guard against copy errors):
```bash
# each of the Examples blocks above should report "✓ No errors found!"
ailang check <snippet>.ail
```

**Audit:**
- `grep -rn "parseInt\|toFloat\|\bstr(" examples/` — confirm no example uses a non-existent name.
- `grep -ln "stringToInt\|intToStr\|floor\|ceil\|round" examples/` — confirm each has its import.

**Regression:**
- Prompt-only change (+ possible example-import fixes); no compiler/stdlib changes.

## Non-Goals

- **Not** renaming or aliasing any conversion function (the names are correct AILANG; renaming is a breaking change).
- **Not** promoting `std/string` conversions into the prelude (that is the structural sibling — see Future Work and [M-PRELUDE-OPTION-RESULT](m-prelude-option-result.md)).
- **Not** changing `stringToInt`'s `Option` return to throw/panic (structured failure is by design — A11).
- **Not** covering every stdlib import gap — only numeric/string conversion specifically.

## Timeline

**0.5 day (4 hours):**
- Segment nightly-eval for frequency + affected benchmarks (~1h, unblocks the Design Freeze gate).
- Insert table + callout + example into the prompt (~1h).
- Verify snippets and run targeted benchmarks (~2h).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Frequency turns out negligible after segmentation | Low | P2 already; if <2% across recent runs, downgrade to "monitor only" like [string-concat](m-prompt-string-concat-plusplus.md) rather than ship |
| Prompt grows too long; table crowds out other content | Low | Single compact table replaces scattered ad-hoc examples; net-neutral length |
| M-PRELUDE-OPTION-RESULT later removes the import for `Option`, changing the `stringToInt` example | Low | Example still valid (import is harmless if redundant); revisit if string conversions also move to prelude |
| Models still forget the `Option` unwrap even with the table | Med | The worked `match` example is the mitigation; pairs with the [Option/None](m-prompt-option-none-idiom.md) callout |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language change |
| A2: Replayability | 0 | No impact |
| A3: Effect Legibility | 0 | Conversions are pure; no effect change |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification change |
| A6: Safe Concurrency | 0 | No impact |
| A7: Machines First | +2 | Directly unblocks model success on a near-universal operation; pure prompt leverage |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Teaches the `Option`-returning forms, which compose with `std/option` and the rest of stdlib |
| A11: Structured Failure | +1 | Reinforces `Option` return for fallible parsing rather than teaching a panic |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → Proceed to implementation (no −1 on A1/A3/A4/A7).

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No capability changes
- [x] A7 (Machines First): Directly improves model success rate

## References

- **Verified gap (HARD GATE), `ailang check` 2026-06-05** (`ailang dev`, commit `d4ba009`):
  - `parseInt("42")` → `undefined variable: parseInt`
  - `toFloat(5)` → `undefined variable: toFloat`
  - `str(42)` → `undefined variable: str`
  - `floor(3.7)` without import → `undefined variable: floor`
  - `floatToInt(3.7)` / `intToFloat(5)` → ✓ compile, **no import** (core builtins `_float_to_int`, `_int_to_float`)
  - `stringToInt("42")` / `stringToFloat` → ✓ compile **only with** `import std/string`
  - `floor`/`ceil`/`round` → ✓ compile **only with** `import std/math`
- **Signatures**: `std/string.ail:40-41,57,60` (`stringToInt -> Option[int]`, `stringToFloat -> Option[float]`, `floatToStr -> string`, `intToStr -> string`); `std/math.ail:59,62,65` (`floor`/`ceil`/`round -> float`).
- **Prior eval evidence (in-repo)**: the `eval-analyzer` skill documents `floatToInt` as a hallucinated function in the `config_file_parser` benchmark (`.claude/skills/eval-analyzer/SKILL.md`). That specific name is now a working builtin; the *remaining* hallucinations (`parseInt`, `toFloat`, `str`) and the import asymmetry are what this doc addresses.
- **Trigger**: nightly-eval Pub/Sub notification → coordinator task `task-11e68c7e` (2026-06-05), agent `design-doc-creator`.
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

## Future Work

- **Structural sibling**: promote `std/string` conversion functions (`stringToInt`,
  `stringToFloat`, `intToStr`, `floatToStr`) into core/prelude so the import
  asymmetry with `floatToInt`/`intToFloat` disappears entirely — mirrors
  [M-PRELUDE-OPTION-RESULT](m-prelude-option-result.md). This would make the
  prompt fix largely unnecessary, the same way prelude-Option supersedes the
  Option/None prompt callout.
- **Linter hint**: when `parseInt`/`toFloat`/`str`/`toString` appears undefined,
  suggest the canonical AILANG name (`did you mean stringToInt?`). Eliminates the
  surprise at the source rather than via prompt teaching.
- **Unified "conversion & stdlib import" prompt section** grouping this with the
  [Option/None](m-prompt-option-none-idiom.md) and [Split](m-prompt-split-list-operations.md)
  gaps under one "functions you must import" heading.
