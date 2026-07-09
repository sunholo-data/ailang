# M-DIAGNOSTIC-COVERAGE: The footgun coverage table — error-time teaching with CI-enforced fix-carrying diagnostics (R1.1)

**Status**: Planned
**Target**: v0.29.0 (Phase 1 of the strategy review routing table)
**Priority**: P0 (the cheapest cost-per-success lever with fresh causal evidence: one missing diagnostic cost a full mid-tier benchmark cell)
**Estimated**: 3–4 days for the mechanism + first three diagnostics (table+CI 1d, diagnostics 1.5d, prompt-deletion pass + rig A/B 1d)
**Dependencies**: [m-fable-strategy-review](../m-fable-strategy-review.md) R1 (this IS R1.1 from its Phase-1 routing table)

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | CI fixture per footgun: diagnostics become tested contracts |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | Converts a per-run prompt tax into error-time teaching delivered only when needed |
| A8: Minimal Syntax | 0 | No change |
| A9: Cost Visibility | +2 | KPI is literally prompt tokens deleted per release; repair rounds stop being wasted |
| A10: Composability | 0 | No change |
| A11: Structured Failure | +2 | Every covered footgun carries a concrete fix in `check --json` suggestion fields |
| A12: System Boundary | 0 | No change |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check
- [x] A1 / A3 / A4 / A7: no violations

## Problem Statement

The strategy review's R1 argument, now with causal evidence from the wave-5 campaign:

- **#325 (import placement)**: `import` after declarations produces
  `PAR_NO_PREFIX_PARSE: unexpected token in expression: import` — no statement of the
  rule. Measured cost: claude-haiku-4-5 lost the entire `legal_obligation_engine`
  AILANG cell to exactly this (its one self-repair round couldn't act on the
  unexplained error), while its Python attempt computed every value correctly.
  claude-sonnet-4-6 lost two wave-2 cells the same way. The author of the deontic
  package (a frontier model) hit the same footgun three times in one sprint.
- **#327 (record-update resolution)**: the diagnostic says "undefined variable" about
  a defined function — it actively lies. Until the bug fix ships
  ([m-record-update-local-resolution](m-record-update-local-resolution.md)), the
  diagnostic must at least tell the truth and carry the hoist-to-let workaround.
- **The gold standard already exists**: the `++`-on-strings diagnostic
  (`` `++` is for lists only → use "${a}${b}", concat([...]), or join(sep, ...) ``)
  teaches the fix in ~25 tokens at exactly the right moment. R1's thesis: every
  Common-Mistakes prompt line should either become one of these or justify why not.

Current state: no inventory maps footguns → diagnostics → fixtures → prompt lines;
diagnostic quality is accidental; the 2,518-line teaching prompt pays for all of it
on every run.

## Goals

**Primary Goal:** a CI-enforced footgun coverage table where each entry has a
fix-carrying diagnostic, a fixture asserting the fix text, and (once shipped) its
teaching-prompt lines deleted.

**Success Metrics:**
- `internal/diag/footguns.md` (or equivalent) table exists with ≥ 10 entries inventoried from the prompt's Common Mistakes + LIMITATIONS.md.
- First three diagnostics shipped with fixtures: (1) import placement (#325), (2) #327 interim truth-telling, (3) one Common-Mistakes entry chosen by prompt-line count.
- ≥ 100 teaching-prompt lines deleted with rig A/B showing no pass-rate loss (R1 gate).
- KPI wired: "prompt tokens deleted per release" reported at each baseline.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Prompt lines are deleted ONLY after the replacement diagnostic ships + rig A/B confirms no loss | The R1 safety gate against mid-tier regression | human (already framed in strategy review) | design | low |
| Table location + CI mechanism (fixture per entry asserting diagnostic text contains the fix) | Makes coverage enforceable, not aspirational | agent proposes | sprint start | low |

### Design Freeze
- [ ] Mark confirms the three first entries (#325, #327-interim, +1 by prompt-line count)
- [ ] Mark confirms the deletion gate (diagnostic shipped + A/B green before prompt lines go)

## Conflict Surface

Touches `internal/parser/` (for #325's placement diagnostic) and error-message plumbing.

1. **Positions extended:** none — diagnostics only. #325's fix point: when the
   declaration-level parser encounters `import` after any non-import declaration,
   emit `PAR_IMPORT_PLACEMENT: imports must appear immediately after the module
   declaration → move this import above the first declaration` instead of falling
   into expression-parse errors.
2. **What already lives there:** parser error recovery — the current triple-error
   cascade per misplaced import (verified in the sonnet wave-2 failures). The new
   diagnostic replaces the cascade for this token; other PAR_NO_PREFIX_PARSE causes
   must be untouched (fixtures: `import` mid-file vs genuinely stray tokens).
3. **Programs that must still work:** all currently-valid modules (imports first) —
   the diagnostic only fires on today's error path.
4. **Deliberate changes:** error text/codes for the covered footguns (agents keying
   on old strings — the eval harness's `CategorizeErrorWithCode` patterns must be
   updated in the same PR; grep for affected `errorRule` entries).

## Solution Design

1. **Inventory** (~0.5d): sweep `prompts/v0.16.2.md` Common Mistakes + `docs/LIMITATIONS.md`
   into the table: footgun | trigger example | current diagnostic (verbatim, live-verified) |
   target diagnostic | fixture path | prompt lines to delete | status.
2. **Mechanism** (~1d): `internal/diag/footgun_fixtures_test.go` — table-driven: each
   entry has a `.ail` snippet; the test runs `ailang check --json` and asserts the
   diagnostic code + that the message/suggestion contains the fix substring. CI-red on drift.
3. **First three diagnostics** (~1.5d): #325 placement (parser), #327 interim message
   (elaborator, retired when the real fix ships), +1 chosen entry.
4. **Deletion pass + rig A/B** (~1d): delete covered prompt lines on a branch, run the
   prompt-size A/B per R3.1 tooling, merge only on no-loss.

## Success Criteria
- [ ] Table with ≥ 10 inventoried entries, each row live-verified
- [ ] 3 diagnostics shipped, fixtures green in CI
- [ ] haiku re-run on `legal_obligation_engine` AILANG: the #325 cell flips (the causal test of the whole thesis)
- [ ] ≥ 100 prompt lines deleted, rig A/B no-loss, KPI reported

## Verification Log

| Claim | Method | Result |
|---|---|---|
| #325 error text carries no rule statement | live wave-2/wave-5 failure stderr | `unexpected token in expression: import` ×3 lines, no placement rule |
| #325 cost a full benchmark cell | delta_probe_midtier forensics (2026-07-09) | haiku AILANG compile_error on exactly this; repair failed |
| #327 diagnostic misleads | live check, deontic engine | "undefined variable" for a defined function |
| `++` diagnostic is the gold standard | live check (strategy review Verification Log) | fix-carrying text confirmed v0.28.0 |
| Prompt size | wc on prompts/v0.16.2.md (strategy review) | 2,518 lines |

## Non-Goals
- Fixing the underlying bugs themselves (#327 has [its own doc](m-record-update-local-resolution.md)); this doc ships the teaching layer.
- Prompt rewriting beyond deletion of covered lines (R3's compression program is separate).
- "Did you mean?" typo suggestions — worthy, separate entry once the mechanism exists.

## Related Documents
- [m-fable-strategy-review](../m-fable-strategy-review.md) — R1; this doc is its Phase-1 item R1.1
- [m-record-update-local-resolution](m-record-update-local-resolution.md) — real fix for #327; this doc's interim entry retires with it
- [m-agent-ergonomics](../v0_29_0/m-ailang-semantic-context.md) — friction data source for prioritizing entries
- #325, #327 — the fresh causal evidence

---
**Document created**: 2026-07-09
