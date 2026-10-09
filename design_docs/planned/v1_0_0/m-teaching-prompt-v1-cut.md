# M-TEACHING-PROMPT-V1-CUT — the R1.2 deletion pass, measured tranche by tranche

**Status**: DRAFT, BLOCKED at quorum r1 (2026-10-09, 3/3: the A/A noise-floor protocol is self-contradictory, and the eval machinery is verified only at flag level). Not revised yet; parked until the flagship docs are ruled. Design Freeze items open for Mark.
**Target**: v1.0.0, clause 3 of the [v1.0 bar](../../v1-mission.md#the-v10-bar--v2-product-shaped-ratified-2026-07-11-mark-supersedes-the-2026-07-10-hygiene-bar)
**Priority**: P0 for v1.0. This is the NEW-DOC unit "prompt-deletion pass R1.2" that `D-51` counts under clause 3.
**Estimated**: ~1 week (inventory 1d, A/A noise floor 0.5d, 3–4 tranches × 1d each, ratchet 0.5d)
**Dependencies**: [m-diagnostic-coverage](../../implemented/v0_29_0/m-diagnostic-coverage.md). Its mechanism and fixtures landed, and this doc takes over its OPEN tail ("the A/B-gated deletion pass + haiku causal re-run").

**Created**: 2026-10-09 · **Author**: attended session with Mark (Claude Opus 5.5)
**Quorum trigger**: #1 (Design Freeze items) and #3 (it touches the eval KPI's measured inputs) → the quorum runs.

---

## Why this doc, and why not the slim prompt again

Clause 3 requires *"the teaching prompt ≤1,500 lines with a rig-A/B showing no pass-rate loss
(R3.1 measures the curve first; the deletion pass stays gated on replacement diagnostics landing)."*

**The obvious route already failed.** [m-eval-slim-prompt-self-discovery](../v0_29_0/m-eval-slim-prompt-self-discovery.md)
was RULED OUT on 2026-09-02. Its slim prompt moved reference material behind tool calls
(progressive disclosure), and the A/B went **82% → 65%** (V4). The lesson is that the prompt's
*reference* material earns its place, and that a cut must be measured, not assumed.

**R1.2 is a different mechanism.** It deletes a prompt line only when the **compiler now teaches the
same thing at the moment of the mistake**: a diagnostic with a fix-carrying suggestion, pinned by a
fixture in `internal/diag/footgun_fixtures_test.go` (V7). The model learns from the error instead
of the preamble. That is the strategy review's R1, *"make the compiler the prompt"*. Reference
material is cut only where an A/B shows it can be.

**Meanwhile the prompt is growing.** The active prompt `v0.16.6` is **2,600 lines** (V1). The charter
recorded 2,552 on 2026-09-01, and three commits since 2026-10-02 added to it (V3). Nothing stops
regrowth.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime change. |
| A2: Replayability | 0 | No trace change. |
| A3: Effect Legibility | 0 | No effect change. |
| A4: Explicit Authority | 0 | No authority change. |
| A5: Bounded Verification | 0 | No verification change. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | Teaching moves to the point of the error, and the preamble every call pays for shrinks. |
| A8: Minimal Syntax | 0 | No syntax change. |
| A9: Cost Visibility | +1 | Prompt tokens per call drop, and the drop is reported per tranche with its pass-rate delta. |
| A10: Composability | 0 | Neutral. |
| A11: Structured Failure | +1 | Every deleted line must have a structured diagnostic that replaces it. |
| A12: System Boundary | 0 | Neutral. |

**Net Score: +3** → **Proceed.** Hard violations: none.

## Goals

**Primary goal:** reach the smallest active teaching prompt that shows no measured pass-rate loss on
the fleet tier, and stop the prompt growing back.

**Success metrics:**
- An A/A noise floor is measured and recorded before any cut is judged.
- Each tranche is accepted or rejected on the protocol below, with the per-tier delta banked.
- The final prompt is ≤1,500 lines (the bar), **or** it sits at the measured floor and F1 is ruled.
- A CI ratchet fails any PR that grows the active prompt without a ledger note.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| **F1. If 1,500 lines cannot be reached without loss, which wins?** | Decides whether clause 3 can close | **human** | design | high |
| **F2. Which models define "no loss"?** | The clause targets the fleet tier, and a frontier model masks every cut | **human** | design | med |
| **F3. Is the devtools prompt in scope?** | The serve-api additions grew the devtools prompt, not the teaching prompt | **human** | design | low |
| F4. Tranche boundaries and order | The implementer's call within the inventory classes | agent | compile | low |

### Design Freeze

- [ ] **F1:** recommendation **pass rate wins**. Amend the clause to "the measured floor, with
  no loss", and record the number. 1,500 was a target set before the slim result showed reference
  material earns its place.
- [ ] **F2:** recommendation **one mid-tier and one small model from the current eval rotation**,
  plus a frontier model as a non-gating control. The clause is about the fleet tier, and the bar
  already says the sonnet-class outcome is measured, not gating.
- [ ] **F3:** recommendation **out of scope**. Clause 3 names the teaching prompt (`prompts/versions.json` `active`), and the devtools prompt is a separate surface.

## Solution Design

### Phase 0 — Inventory (no deletions)

Classify every section of `prompts/v0.16.6.md`: 62 `##` sections (V2), the largest being Standard
Library 208 lines, JSON 182, Polymorphic ADTs 113, Quick Reference 94 and AI Effect 93. Each
section, or line block within one, gets one class:

| Class | Rule | Fate |
|---|---|---|
| **D: diagnostic-replaced** | A footgun fixture asserts that a diagnostic fires *with a fix-carrying suggestion* for exactly this mistake | Delete in tranche 1 |
| **N: needs a diagnostic** | It teaches a mistake that has no fixture yet | Either file the diagnostic (m-diagnostic-coverage's table) or keep it |
| **R: reference** | API listings and worked examples, the material the slim A/B showed matters | Cut only by A/B, smallest blocks first, never by moving it behind a tool |
| **K: keep** | Syntax core, and anything whose removal fails the A/B | Stays |

The inventory is committed as a table with section, lines, class, and fixture or reason, so the
loop and reviewers can audit every deletion.

### Phase 1 — A/A noise floor

Run the unchanged prompt against itself using `ailang eval-suite --prompt-version v0.16.6 --seed <s>`
(V6). Use ≥3 seeds per F2 model on the `eval-core` set. The spread of pass rate between identical
runs is the noise floor. Without it, a "no loss" verdict means nothing.

### Phase 2 — Tranches

Each tranche becomes a new prompt version (`v0.16.7-cutN`), A/B'd against the current active one on
the same seeds and models.
- **Accept** when the pass-rate delta on every gating model is no worse than the A/A floor.
- **Reject** a tranche that loses more than the floor. Bisect it once, keep the half that passes,
  and mark the other half K.

The order is all D blocks first, then R blocks smallest-first. Every verdict, accepted or rejected,
is banked with its delta, and the version is activated only on accept.

### Phase 3 — Ratchet

A CI check records the active prompt's line count. A PR that raises it must carry a
`prompt-growth:` line in its description naming the reason, or the check fails. This turns the
silent 2,552 → 2,600 drift (V3) into a reviewed decision.

### Files

- `prompts/v0.16.7-cut*.md` and `prompts/versions.json`: the tranches
- `design_docs/planned/v1_0_0/m-teaching-prompt-v1-cut-inventory.md`: the Phase 0 table
- `internal/diag/footgun_fixtures_test.go`: new fixtures for N blocks chosen to be diagnosed
- `.github/workflows/` or `make` CI target: the ratchet (Phase 3)

## Success Criteria

- [ ] F1–F3 ruled and recorded in the V1 ledger.
- [ ] The inventory is committed and every section is classified.
- [ ] The A/A floor is recorded (models, seeds, spread).
- [ ] Each tranche's verdict is banked. The final active prompt is ≤1,500 lines, or at the floor with F1 = pass rate.
- [ ] The ratchet is live and fails a test PR that grows the prompt without a `prompt-growth:` line.

## Non-Goals

- Progressive disclosure or tool-served reference material, which was ruled out with a measured −17pp.
- New diagnostics beyond those needed for N blocks the inventory chooses to diagnose (m-diagnostic-coverage owns the general table).
- The devtools prompt (F3).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Eval noise swamps small deltas | High | The Phase 1 floor gates every verdict, and more seeds are added if the floor is wide |
| A deletion helps one tier and hurts another | Med | A tranche is gated per model and must pass on all F2 models |
| The rig's quota cannot carry the runs | Med | eval-core, F2's two gating models and ≥3 seeds bound the cost; the runs use the flat-rate lanes |

## Verification Log

All checks were run on 2026-10-09 against `origin/dev`.

| # | Claim | Evidence |
|---|---|---|
| V1 | The active prompt is `v0.16.6` at 2,600 lines | `prompts/versions.json` `"active": "v0.16.6"`; `git show origin/dev:prompts/v0.16.6.md \| grep -c ""` → 2600 |
| V2 | 62 `##` sections; largest are Standard Library 208, JSON 182, Polymorphic ADTs 113, Quick Reference 94, AI Effect 93 | `grep -nE "^## "` with line arithmetic |
| V3 | The prompt grew after 2026-09-01 (2,552 recorded by the charter) | `git log -- prompts/v0.16.6.md`: `f610b897b` 10-08, `54b97676c` 10-02, `1239826df` 10-02 |
| V4 | The slim prompt was ruled out with 82% → 65% | `m-eval-slim-prompt-self-discovery.md` status header |
| V5 | The deletion pass is m-diagnostic-coverage's open tail | its status header: "OPEN: … then the A/B-gated deletion pass + haiku causal re-run" |
| V6 | eval-suite can A/B prompt versions on fixed seeds | `ailang eval-suite --help`: `-prompt-version string`, `-seed int` |
| V7 | The fixture mechanism exists and covers ≥9 diagnostic codes | `internal/diag/footgun_fixtures_test.go`: `PAR_IMPORT_PLACEMENT`, `PAR_MODULE_PLACEMENT`, `PAR_HYPHEN_IN_MODULE`, `PAR_RESERVED_KEYWORD`, `PAR_NO_PREFIX_PARSE`, `EFF_UNKNOWN_MODE`, `EFF_UNKNOWN_PARAM_KEY`, `EFF_PARAMS_NOT_SUPPORTED`, `LIST_TAKE_AFTER_FLATMAP` |

## Related Documents

- [m-fable-strategy-review](../m-fable-strategy-review.md): R1, R1.2 and R3.1
- [m-diagnostic-coverage](../../implemented/v0_29_0/m-diagnostic-coverage.md): the mechanism, whose tail this doc takes over
- [m-eval-slim-prompt-self-discovery](../v0_29_0/m-eval-slim-prompt-self-discovery.md): the ruled-out alternative and its measurement

---

**Document created**: 2026-10-09
**Last updated**: 2026-10-09
