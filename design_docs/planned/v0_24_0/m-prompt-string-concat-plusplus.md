# M-PROMPT-STRING-CONCAT-PLUSPLUS: The `++` string-concat reflex — #1 compile failure cause

**Status**: Planned
**Target**: v0.24.0
**Priority**: P0 (Highest — single largest compile-failure cause across ALL models)
**Estimated**: 1 day (prompt salience redesign + eval verification)
**Dependencies**: None

## Axiom Compliance

| Axiom | Score | Rationale |
|---|---|---|
| A3 Fail loudly | +1 | Reduces the most common silent-assumption failure |
| A7 AI-friendly | +2 | Single highest-leverage AI-teachability fix in the project |
| A1 Explicit over implicit | +1 | Salient teaching of the list-only `++` rule |

**Net score: +4** ✅

---

## Problem Statement

**`++` used for string concatenation appears in 46% of ALL AILANG compile failures** (1374 of 2948 compile_error results across the full eval corpus). It is, by a wide margin, the single largest cause of AILANG compile failures across every model tier — frontier and local alike.

Models write:
```ailang
-- ❌ What models generate (TYPE ERROR — ++ is list-only since v0.13.0):
"(" ++ show(count) ++ ", " ++ char ++ ")"
let msg = "Error: " ++ reason
```

AILANG made `++` **list-only** in v0.13.0 (M-CONCAT-DISAMBIG). For strings, AILANG requires `"${...}"` interpolation, `concat([parts])`, or `join(sep, parts)`. But every model's training corpus contains billions of examples of `++` for string concatenation:
- Haskell: `"a" ++ "b"`
- PureScript: `"a" <> "b"` (and `++` historically)
- Elm: `"a" ++ "b"`

This is a **trained reflex**, not an information gap.

### The critical insight: it's ALREADY in the prompt

The v0.16.1 teaching prompt *already* documents this rule in three places:
1. Common Mistakes table: `` `concat(a, b)` for strings → `"${a}${b}"` interpolation — `++` is list-only ``
2. Operators table: `String | "${expr}" interpolation; concat([parts]); join(sep, parts)`
3. A dedicated callout: "**`++` is for lists only (v0.13.0+)...**"

**Yet 46% of compile failures still use it.** This means the problem is not *coverage* — it's *salience*. The rule is buried in tables the model skims past, and the `++` reflex from training is strong enough to override a single mention. This is a teaching-design problem, not a content gap.

**Evidence (June 2026):**
- 1374/2948 compile failures (46%) contain `++` with string operands
- 658/1357 frontier-model compile failures (48%) contain it
- The "EOF truncation" failures we initially attributed to token limits (198 cases) are mostly *downstream* of a `++` error earlier in the file — the parser bails after the type error, producing an EOF-looking error later

---

## Goals

**Primary goal:** Cut the `++`-string-concat compile-failure rate from 46% to under 15% by making the rule impossible to miss.

**Success metrics:**
- `++`-with-strings present in <15% of compile failures (from 46%)
- Estimated CPR improvement: +8–12 pts for mid-tier models (this dwarfs all other prompt fixes combined)
- Overall AILANG pass rate +5–8 pts across all tiers

---

## Solution Design

This is a **prompt salience** problem. Options, in increasing order of intervention:

### Option A: Salience redesign (recommended first step)
Move the `++` rule from a buried table row to a **top-of-prompt hard-rules box** that the model sees first, with a memorable framing:

```
🚫 THE #1 AILANG MISTAKE: `++` IS LIST-ONLY
   "a" ++ "b"        ❌ TYPE ERROR (this is not Haskell/Elm)
   "${a}${b}"        ✅ string concatenation
   "(${x}, ${y})"    ✅ build formatted strings
   [1,2] ++ [3,4]    ✅ ++ works for LISTS only
   
   Before writing ANY ++, ask: are both sides LISTS? If not, use "${...}".
```

Place this in the first 500 tokens of the prompt (high attention zone) AND keep the existing table entries.

### Option B: Error-message improvement (complementary)
When `++` is used on strings, AILANG's type error should explicitly suggest the fix:
```
Error: cannot use ++ on string (++ is list-only since v0.13.0)
  Did you mean string interpolation?  "${a}${b}"  or  concat([a, b])
```
This helps agent-mode self-repair recover (the model reads the error and fixes it). Touches `internal/types/` error rendering. Check current error text first — if it already says this, the agent-mode repair isn't using it.

### Option C: Lenient parse + auto-suggest (larger, defer)
Consider whether the parser could special-case `string ++ string` to emit a *targeted* error with a fix-it, rather than a generic type-unification failure. This is the highest-effort option; evaluate only if A+B don't move the needle.

### Recommended sequencing
1. Ship Option A (salience redesign) in the next prompt version — measure impact
2. Ship Option B (error message) in parallel — helps agent-mode repair
3. Re-measure; only pursue C if `++` failures remain >20%

---

## Files to Modify

| File | Change | LOC |
|---|---|---|
| `prompts/v0.17.0.md` (new version) | Top-of-prompt hard-rules box for `++` | +12 |
| `internal/types/*.go` (error rendering) | Targeted `++`-on-string error + fix-it | +15 |
| `internal/types/*_test.go` | Test the new error message | +20 |

---

## Success Criteria

- [ ] `++` hard-rules box in first 500 tokens of teaching prompt v0.17.0
- [ ] Type error for `string ++ string` suggests `"${...}"` interpolation
- [ ] Next eval rotation: `++`-string-concat present in <15% of compile failures (from 46%)
- [ ] `make test` passes (new error-message test + prompt hash bump)

---

## Conflict Surface

**Option A (prompt):** No compiler change — no conflict surface.

**Option B (error message):** Touches `internal/types/` error rendering only — pure additive string change to the error path, no parse/type behaviour change. Low risk. The 3-5 programs that MUST still work: any existing `list ++ list` (unaffected — error only fires on string operands), `"${a}${b}"` interpolation (unaffected), `concat([...])` (unaffected).

---

## Why This Is P0

Every other prompt-gap design doc in the v0.24.0 queue (import-alias, type-constraints, match-guard, option-none, split-list) addresses 5–9 model failures each. **This one addresses ~1374 compile failures** — more than all the others combined, by an order of magnitude. It should be done first.

---

## Related Documents

- `design_docs/implemented/v0_13_0/m-concat-disambiguation.md` — the v0.13.0 change that made `++` list-only (the root design decision)
- `design_docs/planned/v0_24_0/m-import-alias.md`, `m-type-constraints.md` — lower-impact companion prompt fixes
- This supersedes the truncation theory in `m-prompt-concise-recursive-solutions.md` (which misattributed `++` failures to token limits)
