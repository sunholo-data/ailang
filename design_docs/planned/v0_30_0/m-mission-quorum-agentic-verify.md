# M-MISSION-QUORUM-AGENTIC-VERIFY: quorum reviewers that VERIFY (agentic, repo-armed), not just reason

**Status**: Planned (follow-up to the shipped Phase B text quorum; Mark 2026-07-14 — "we need a
follow up where it's an agent that is verifying, not just an api call")
**Target**: v0.30.x (mission infrastructure; sequenced AFTER fleet Phase C — cross-provider
executors — because it REUSES that executor plumbing)
**Priority**: P1 (closes the one gap the text quorum structurally cannot: independent premise
verification — the exact class that has cost mid-sprint corrections)
**Estimated**: ~2–3d (escalation tier ~1d; full agentic-reviewer backend ~2d) — phase-gated
**Dependencies**: the shipped quorum (`internal/mission/quorum/*` — reuse its verdict contract,
reject-by-default validation, N-1 degradation, artifact); `internal/coordinator/provider_executor.go`
(codex = OpenAI agent, managed_agents = Gemini agent — the agentic backends);
[m-mission-adaptive-multiprovider-routing](m-mission-adaptive-multiprovider-routing.md) Phase C
**Author**: Opus session, requested by Mark 2026-07-14

---

## Problem statement

The Phase B quorum (shipped iteration 28, `internal/mission/quorum/`) gives each design doc N
independent frontier reviews under a reject-by-default rubric scoring the three design-doc-creator
hard gates (premise verification, conflict surface, axiom compliance). But each reviewer is a
**single text-in / JSON-out call** (`run.go:120 caller.CallJSON`) with **no repo access**. Two
consequences, verified in the code:

1. **It reasons, it does not verify.** Gate 1 ("is every factual claim about the codebase
   verified or asserted?") can only be judged from what the doc SAYS about itself — a reviewer
   cannot open the file, run `ailang check`, or grep to confirm a claim. So the quorum catches
   *unjustified/hand-waved* premises but **cannot catch a confidently-wrong premise that the doc
   claims it verified**. That confident-but-wrong class is exactly what iterations 22 and 27's
   Opus planners had to correct mid-sprint (whole-mechanism errors that read as verified).
2. The strong form of the independence property (Phase E cross-family eval) wants a judge that
   can *run the tests*, not just read the diff. A text call can't.

The text quorum is the right cheap first pass. This doc adds the deep pass.

## Design — reuse the contract, swap the backend, escalate by default

### Keep (do NOT rebuild — the shipped quorum is the frame)
The verdict schema (`{verdict, strongest_objection, catch}`), the reject-by-default validation
(`ValidateReviewResult` — empty objection is a hard error, never coerced to pass), the N-1
graceful degradation, and the artifact recording ALL stay. This follow-up changes only *how a
reviewer produces its verdict*, from a text call to an agentic run.

### Agentic reviewer backend
Each reviewer becomes a **tool-using agent in a read-only worktree of `origin/dev`**, ridden on
the EXISTING executor registry rather than new plumbing:
- OpenAI reviewer → `codex` executor (already integrated, `provider_executor.go`).
- Gemini reviewer → `managed_agents` path (already integrated).
- Claude reviewer → `claude -p` (the controller's own harness).

The agent gets the doc body + the same reject-by-default rubric, PLUS repo tools (`ailang check`,
grep, read — read-only; no writes, no network beyond the model API). It is instructed to
**actually verify each premise gate 1 claim against the code** and cite the check it ran in its
`catch` field. Same JSON verdict out; now the objection can be "premise X is FALSE — I ran
`ailang check` on the cited repro and it passed, contradicting the doc," which the text tier
cannot produce.

### Two-tier escalation (the cost-smart default)
Agentic review is multi-turn and dollar-billed (vs the text tier's cents), so it does not run on
every doc:
1. **Tier 1 (always): the shipped text quorum.** Fast, cheap, catches reasoning/conflict-surface/
   axiom gaps.
2. **Tier 2 (escalation): agentic verification.** Triggered when — any Tier-1 reviewer's
   objection is PREMISE-class (a factual codebase claim in dispute), OR the doc self-declares
   high-stakes (touches shared infra / a Conflict Surface), OR Tier-1 is split (pass/reject
   disagreement). The escalated agent verifies the specific contested premise, not the whole doc.

Result: cheap by default, deep exactly when a premise is contested — the moment verification
actually pays for the agentic cost.

## Constraints & guardrails
- **Read-only worktree per reviewer** (concurrent-agent safety; no writes to the shared tree).
- **Per-review budget cap** in the caller (extends the existing `MaxCostUSD` the quorum already
  threads); a reviewer that blows the cap fails to N-1 degradation, never blocks the loop.
- **Bounded** — agentic reviews get a hard turn/time cap (the loop's bounded-wait discipline).
- **No quality discount, no authority** — a reviewer verifies and objects; it does not edit the
  doc or gate merges by itself. The controller still synthesizes verdicts (unanimous-pass →
  proceed; any-reject → objection back to the author).

## Precondition (verify FIRST)
Confirm the shipped Tier-1 quorum has actually fired a LIVE 3-provider review (no quorum
artifacts found in `.ailang/state/` as of 2026-07-14 — it may have only run under the test stub).
A live Tier-1 run with real Sol+Gemini verdicts is the baseline this builds on; if it has never
fired live, that's step 0.

## Conflict surface
Touches the quorum package (additive: a second reviewer backend behind the existing `JSONCaller`
interface — `call.go:35` is already the seam) and the executor registry (read-only consumer).
Must NOT: change the shipped verdict contract; let agentic reviewers write to the shared tree;
run agentic review on every doc (cost); grant reviewers merge authority.

## Non-goals
- Replacing the text quorum (it's Tier 1, kept).
- A general agentic code-review product (this is design-doc premise verification specifically).
- Giving reviewers write/fix authority (they object; the author fixes).

## Verification log
| Claim | Method | Result |
|---|---|---|
| Text reviewers have no repo access | `internal/mission/quorum/run.go:120` (CallJSON), reviewer.go BuildPrompt (doc body only) | Confirmed |
| Rubric = the 3 design-doc-creator hard gates | reviewer.go systemPrompt | Confirmed |
| codex/managed_agents already integrated | provider_executor.go | Confirmed (redundancy audit 2026-07-14) |
| No live quorum artifact yet | `find .ailang/state -name '*quorum*'` | none found — precondition flagged |

## Related
- [m-mission-adaptive-multiprovider-routing](m-mission-adaptive-multiprovider-routing.md) — Phase B (this extends), Phase C (executor reuse), Phase E (cross-family eval this enables)
- `.claude/skills/design-doc-creator/SKILL.md` — the agentic verification standard the reviewers now match

---
**Document created**: 2026-07-14
