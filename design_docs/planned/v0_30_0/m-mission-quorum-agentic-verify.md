# M-MISSION-QUORUM-AGENTIC-VERIFY: quorum reviewers that VERIFY and HONE (agentic, repo-armed), not just reason

**Status**: Planned — **PARKED needs-human-review 2026-07-16 (iteration 34)** at the Gate-2
quorum-at-pick gate: one bounded revision round exhausted, re-quorum still BLOCKED on a real
contract-wording decision (see "Quorum-at-pick round 2" note below — reuse premise verified TRUE
in code; only objection #2's optional-vs-required `proposed_fix` decision remains). **Preconditions
SATISFIED** (Tier-1 quorum fired live iterations 28–30, artifacts on disk; Phase C executor plumbing
proven by iteration 32's codex live-fire, PR #397). **SCOPE EXPANDED 2026-07-16 (Mark)**: reviewers don't just object — each verified
objection carries a **concrete `proposed_fix`** the author accepts/rejects. This is the
"Sol + Gemini Pro + Fable hone the design" capability, kept inside the single-author model.
(Original ask Mark 2026-07-14 — "an agent that is verifying, not just an api call".)
**Target**: v0.30.x — **queued NEXT after fleet (b)** (gemini M1c wiring, iteration 33) — it
reuses that lane the moment it lands
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

### HONE: verified objections carry a proposed fix (added 2026-07-16, Mark)
The verdict schema gains one ADDITIVE, OPTIONAL field: `proposed_fix` — a concrete revision
(replacement paragraph / corrected claim / added verification-log row) for the objection, grounded
in what the agent actually verified. **This does not violate the "do NOT change the shipped verdict
contract" rule above**: the three existing fields (`verdict`, `strongest_objection`, `catch`) keep
their exact meaning and validation, and `proposed_fix` is optional-absent — every existing consumer
and the `ValidateReviewResult` reject-by-default check are byte-compatible with a verdict that omits
it. Additive-optional is a backward-compatible extension of the contract, not a breaking change to it. **Authority model unchanged**: reviewers still have zero write access —
the AUTHOR (the designer role; true-Fable via the Gate-3 `claude:claude-fable-5` CLI lane, added
same day) integrates, accepting or rejecting each proposal by name in the doc's revision note.
This is deliberately single-author + adversarial-proposers, NOT co-authoring: the quorum's value
has been sharp disagreement (live 2-reject/1-pass splits), which design-by-committee would smooth
away. **DECIDED 2026-07-16 (Mark, option (a) — resolves iteration 34's parked blocker):**
`proposed_fix` is **OPTIONAL and not validated** — the shipped verdict contract stays frozen
(`ValidateReviewResult` and the Go struct unchanged; the field is purely additive). Reviewers are
PROMPTED to include a concrete fix with every reject and the author pushes back on fix-less
rejects; a fix-less reject is recorded as reviewer friction, not a validation error. (The earlier
"must carry a fix" wording contradicted "contract unchanged" — gemini-3-1-pro's round-2 catch.)

### Agentic reviewer backend
Each reviewer becomes a **tool-using agent with read-only repo access**, ridden on the EXISTING
executor registry rather than new plumbing — but repo access differs by sandbox locality:
- OpenAI reviewer → `codex` executor: **local read-only worktree of `origin/dev`** (already
  integrated, `provider_executor.go`; iteration 32 proved the lane).
- Claude reviewer → `claude -p` (the controller's own harness): local read-only worktree.
- Gemini reviewer → `managed_agents`: **NO local worktree exists** — the agent runs in a
  Google-hosted sandbox with no shared filesystem that **starts empty** (`CapRemoteSandbox`,
  README verified 2026-07-16). Repo access = **clone-in-sandbox**: the repo is PUBLIC, so the
  directive instructs `git clone --depth 1` at the PINNED review SHA + fetch the linux `ailang`
  release binary to run `ailang check`. Read-only holds by construction (its writes stay in
  Google's sandbox). **UNVERIFIED PREMISE — probe FIRST (M0 of this sprint):** whether the
  managed sandbox allows outbound network (git/curl). One cheap interaction: clone + fetch binary
  + `ailang --version`, report. If network is BLOCKED → gemini falls back to prompt-packed
  excerpts (reviewer sees only what we send — weaker, flag it) or stays a Tier-1 text reviewer;
  do NOT silently pretend it verified.

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

## Precondition — ✅ SATISFIED 2026-07-16
The Tier-1 quorum HAS fired live: artifacts in `.ailang/state/mission-quorum/` (iterations 28–30,
e.g. `m-dx-examples-coverage-2026-07-14T09-23-43Z.json` — gpt5-6-sol reject + gemini-3-1-pro
reject + Claude pass, a real 3-provider split). Phase C plumbing proven by iteration 32's codex
live-fire (PR #397). Nothing blocks this doc.

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
| Reuse premise: cancellation, timeout, cost, read-only tools, worktree ALL already exposed by the executor layer | Controller code-read 2026-07-16 (iteration 34, refuting sol's round-2 objection): `Execute(ctx …)` (cancellation), `opts.Timeout`/`IdleTimeout`, `result.Cost = execResult.CostUSD`, `AllowedTools=["Read","Grep","Glob","WebFetch","WebSearch"]` (provider_executor.go:122–124), `WorkingDir` (worktree) | Confirmed — reuse-don't-rebuild HOLDS |
| `proposed_fix` optionality vs contract freeze | Mark decision 2026-07-16: option (a) — optional, not validated, contract frozen | Decided (unblocks the iteration-34 park) |
| Live quorum artifacts on disk (precondition SATISFIED) | `ls .ailang/state/mission-quorum/` | Confirmed 2026-07-16 — iter 28–30 `m-dx-examples-coverage-2026-07-14T09-{19,21,23}*.json` present (the earlier `find -name '*quorum*'` returned none because artifacts are named after the DOC, not the literal "quorum" — that command was the wrong probe, not evidence of absence) |

## Related
- [m-mission-adaptive-multiprovider-routing](m-mission-adaptive-multiprovider-routing.md) — Phase B (this extends), Phase C (executor reuse), Phase E (cross-family eval this enables)
- `.claude/skills/design-doc-creator/SKILL.md` — the agentic verification standard the reviewers now match

## Revision note — quorum-at-pick round 1 (2026-07-16, iteration 34)
Text quorum at pick (`m-mission-quorum-agentic-verify-2026-07-16T09-42-56Z.json`) returned BLOCKED,
both reviewers on one objection; controller integrated (single-author model):
- **gpt5-6-sol + gemini-3-1-pro (ACCEPTED):** the Verification Log's "no live quorum artifact yet"
  row contradicted the SATISFIED precondition. Root cause found on re-probe: the row's `find
  .ailang/state -name '*quorum*'` is the WRONG command — artifacts are named after the doc, so it
  matches nothing even though `ls .ailang/state/mission-quorum/` shows the iter 28–30 artifacts.
  Fixed the row to the correct probe + result; precondition is genuinely satisfied.
- **gpt5-6-sol #2 (ACCEPTED):** reconciled the `proposed_fix` "schema gains a field" wording with
  the "do NOT change the verdict contract" constraint — clarified it as additive-OPTIONAL,
  backward-compatible with `ValidateReviewResult` and existing consumers.

## Quorum-at-pick round 2 → PARKED needs-human-review (2026-07-16, iteration 34)
Re-quorum after round-1 integration (`m-mission-quorum-agentic-verify-2026-07-16T09-44-59Z.json`)
returned BLOCKED again, but on DEEPER, different objections (round-1 fixes accepted). Per the
Gate-2 QUORUM-AT-PICK rule (one bounded revision round, then park), the item is parked. The two
round-2 objections, characterized by the controller:

1. **gpt5-6-sol — "reuse premise unverified" → REFUTED BY CODE (controller investigation).** The
   objection: the doc proves executors *exist* but not that they expose tool-execution, worktree
   selection, cancellation, or cost accounting. **This is the text tier's structural blind spot —
   it cannot read the repo (the exact gap this doc closes).** Controller verified in
   `internal/coordinator/provider_executor.go`: cancellation = `Execute(ctx context.Context, …)`
   threaded to `p.exec.Execute(ctx, …)`; timeout = `opts.Timeout`/`IdleTimeout`; cost =
   `result.Cost = execResult.CostUSD` (+ Input/OutputTokens); **read-only tool mode = lines 122–124,
   `AllowedTools = ["Read","Grep","Glob","WebFetch","WebSearch"]` for `Kind=="question"`** — exactly
   the read-only reviewer capability; worktree = `WorkingDir` (agent_registry.go:37, `{{.Workspace}}`)
   + the worktree machinery in approval_processor.go. **The reuse-don't-rebuild premise HOLDS.** Fix
   is CHEAP, not a re-scope: add these code-cited rows to the Verification Log. (NOT done here — that
   would be a second revision round, which the gate forbids.)
2. **gemini-3-1-pro — real contract contradiction → NEEDS AN AUTHORIAL DECISION.** "A reject MUST
   carry `proposed_fix`" makes the field *required-on-reject*, which means `ValidateReviewResult` +
   the Go struct DO change — contradicting "contract unchanged / only how a reviewer produces its
   verdict." This is a genuine, small design decision the doc must settle one way:
   **(a)** `proposed_fix` stays truly optional (strongly-encouraged-on-reject, NOT validated →
   `ValidateReviewResult` unchanged, wording softened from "MUST"), OR
   **(b)** accept a bounded contract extension (struct + `ValidateReviewResult` gain the field, drop
   the "zero changes" claim). Recommended: **(a)** — keeps the shipped contract frozen, matches the
   "additive-optional" framing.

**Unblock path (≈2-minute human/author call):** decide (a) vs (b) for objection #2, add the
code-cited Verification-Log rows for objection #1, then re-route to sprint-planner. Preconditions
otherwise satisfied. See Gate-5 retro (iteration 34) for the meta-finding: the text quorum-at-pick
blocked a doc whose premises are TRUE-in-code precisely because text reviewers cannot verify code —
the motivating case for this very doc.

---
**Document created**: 2026-07-14
