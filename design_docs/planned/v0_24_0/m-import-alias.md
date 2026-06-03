# M-IMPORT-ALIAS: Import alias syntax (`import X as Y`) — prohibition + workaround

> **📊 RECENT-VERIFIED: 6% of recent compile failures (16/230). Real but moderate; multi-causal with ++ on log_file_analyzer.** (verified 2026-06-03 against Apr-Jun 2026 data only — not all-time aggregate.)

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (High)
**Estimated**: 0.5 day (prompt/docs only) + 2 days if implementing syntax
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No determinism impact |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Clear import errors aid local verification |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | Reduces a recurring compile-failure class for AI codegen |
| A8: Minimal Syntax | 0 | Phase 1 adds no syntax (prompt only) |
| A9: Cost Visibility | 0 | No resource-cost changes |
| A10: Composability | 0 | No composition change |
| A11: Structured Failure | +1 | Replaces cryptic PAR error path with clear guidance |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Proceed** (no −1 on A1/A3/A4/A7)

---

## Problem Statement

Every widely-used language with module systems supports import aliases:
- Python: `import numpy as np`
- Haskell: `import Data.List as L`
- Rust: `use std::collections::HashMap as Map`
- TypeScript: `import * as R from 'ramda'`

Models trained on these languages **universally attempt** `import std/list (map, filter) as L` in AILANG when they want to shorten module names or avoid verbosity. AILANG does not support this syntax. The result is an immediate parse error:

```
PAR_UNEXPECTED_TOKEN at benchmark/solution.ail:2:33: expected next token to be }, got IDENT instead
```

**Evidence from eval (June 2026):**
- `log_file_analyzer` fails in **9/9 frontier models** — the single highest-impact benchmark. Import aliases are a primary cause (alongside dot-notation string methods).
- `polymorphic_ord_defaulting` fails in 5 frontier models; some attempts use `as L` pattern.
- Affects every model family (Claude, GPT, Gemini) equally — it's not a capability gap, it's a universal false assumption.

**Concrete example** (from `gemini-3-flash` eval run, April 2026):
```ailang
-- ❌ What models generate (PARSE ERROR):
import std/list (map, filter, length as listLen, foldl, sortBy, any)

-- AILANG does not support `as` renaming in import lists
-- Result: PAR_UNEXPECTED_TOKEN at :6:38
```

---

## Goals

**Primary goal:** Eliminate import-alias parse errors across all models, recovering ~2–3 benchmark failures per model tier.

**Success metrics:**
- log_file_analyzer compile_error rate improves (note: multi-causal — alias is one of ++, alias, parse; expect partial recovery)
- No model generates `import X as Y` syntax after prompt update
- CPR (Conditional Pass Rate) improves by ≥2 percentage points for mid-tier models

---

## Solution Design

### Phase 1 (Ship in v0.24.0): Prompt prohibition + canonical workaround

Add to the AILANG teaching prompt under "Common Mistakes":

```
| `import std/list (map) as L`    | NOT SUPPORTED — no import aliases in AILANG. Import |
|                                   | individual names directly: `import std/list (map, filter)` |
|                                   | For long lists, use multiple imports or qualified calls. |
```

Plus a callout box:

```
⚠️ AILANG HAS NO IMPORT ALIASES
`import M as X` and `import M (f) as X` are NOT valid syntax.
Import by name: `import std/list (map, filter, foldl)`
There is no way to rename an import. Use the full function name or
write a local alias: `let listLen = length` inside a function body.
```

**Workaround example** (to add to prompt):
```ailang
-- ❌ WRONG: import aliases not supported
import std/list (length as listLen, sortBy)

-- ✅ CORRECT: import by full name, alias via let if needed  
import std/list (length, sortBy)

-- If you need a shorter local name:
export func process(items: list[string]) -> int {
  let listLen = length;   -- local alias via let binding is fine
  listLen(items)
}
```

### Phase 2 (Future, separate design doc): Implement import alias syntax

If/when import aliases are implemented as a language feature, it should be:
```ailang
import std/list (length as listLen, sortBy)  -- rename imported name
import std/list as L                          -- module alias (all exports)
```

This is a parser + elaboration change. It touches `internal/parser/parser_file.go` and the module system. Conflict surface: must not conflict with existing `(f, g)` import list syntax. A separate design doc (`m-import-alias-syntax`) should be created before implementation with full conflict surface analysis.

---

## Files to Modify (Phase 1 only)

| File | Change | LOC |
|---|---|---|
| `prompts/v0.17.0.md` (new prompt version) | Add prohibition + workaround example | +8 |
| `docs/docs/guides/serve-api.md` or lang reference | Document the limitation | +5 |

---

## Success Criteria

- [ ] Teaching prompt v0.17.0+ contains explicit `import X as Y` prohibition
- [ ] Workaround (local `let` alias) example in prompt
- [ ] `log_file_analyzer` benchmark re-run shows <50% compile_error rate
- [ ] `make test` passes (prompt hash update for new version)

---

## Conflict Surface

**Phase 1 (prompt only):** No parser/compiler changes — no conflict surface.

**Phase 2 (syntax implementation):** Would require conflict surface analysis. The `as` keyword is currently unused in import position. Must verify no ambiguity with record syntax `{ x as y }` (which also doesn't exist yet) or type alias `type X = Y`.

---

## Related Documents

- `design_docs/planned/v0_24_0/m-prompt-log-file-analyzer-string-ops.md` — companion fix for log_file_analyzer (dot-notation string ops)
- `design_docs/planned/v0_24_0/m-prompt-match-guard-syntax.md` — similar prompt-gap pattern (models assume syntax that doesn't exist)
