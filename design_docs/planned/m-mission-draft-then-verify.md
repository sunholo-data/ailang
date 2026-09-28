# M-MISSION-DRAFT-THEN-VERIFY: cheap models draft, an expensive model verifies — measured per role before anything is routed

**Status**: PROPOSED — a *possible* direction, not approved to route. **Quorum guardrail spent (round 0 + one re-quorum, both BLOCKED 3/3; all six objections accepted and addressed in-session, see §Quorum History). Awaiting Mark's ratification of the Design Freeze, not a third review round.** Phase 0 is measurement over evidence the loops already bank; nothing past it runs until Mark rules.
**Target**: mission harness (no AILANG release); the fleet mission can build Phases 1–2 (all paths are
in its allowlist)
**Priority**: P2 — a cost lever, not a bar clause. It rises if Anthropic weekly quota keeps binding
(72 % of the 80 % ration on 2026-09-27)
**Estimated**: Phase 0 ~0.5 day · Phase 1 ~1 day · Phase 2 ~2 weeks of wall clock at normal cadence
(≈15 world iterations), ~1 day of build
**Dependencies**: none blocking. Consumes the rotation in `gate-3-route.md`, the typed pi verdicts
(`scripts/mission_pi_run.sh`), the quorum (`internal/mission/quorum`). Complements
[m-mission-elo-routing](m-mission-elo-routing.md) and
[m-mission-role-elo-and-tier-order](m-mission-role-elo-and-tier-order.md) — see §Related Documents.
**Created**: 2026-09-28 (Mark, attended: *"first drafts are made with the cheap models (with
variations?) and then a more expensive model just reads and verifies them … will that be effective?"*)
**Quorum trigger**: #1 (design-freeze items) and #3 (cost/KPI semantics) both fire.

---

## Axiom Compliance

Harness work: no language surface changes, so most axioms score 0.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Routing stays explicit and pre-registered. The mechanical half of verification (sampling, re-runs, comparison, the disposition floor) is a script, not model judgement, and it fails closed, so a timeout can never turn a refutation into an accept (§Architecture) |
| A2: Replayability | +1 | Every draft/verify pair is banked as a disposition row with both artifacts' SHAs, so a routing outcome can be re-read later |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | Verifier and drafter keep today's scopes; the fleet builds under its existing allowlist |
| A5: Bounded Verification | +1 | At most 7 re-runs of 40 s each, 280 s per draft, enforced by the script. Anything not verified inside that bound counts against the draft |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | 0 | Loop-internal |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Per-role, per-side cost becomes a banked number instead of an inference |
| A10: Composability | 0 | Composes with the rotation and the quorum without replacing either |
| A11: Structured Failure | +1 | A draft that is empty, rubber-stamped or over-edited gets a typed disposition, not a silent pass |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +4** → Proceed (to Phase 0 measurement only). No −1 on A1/A3/A4/A7.

## Problem Statement

The mission loops spend expensive models (Opus as controller, and in the designer rotation) on work
where a cheaper model might produce most of the value. The proposal: cheap models write first drafts,
possibly several variants, and an expensive model reads and verifies them. Sometimes it says "carry on";
sometimes it edits and upgrades.

**This is not hypothetical — World already runs a version of it**, and its record is mixed:

- **Cheap drafts often produce nothing.** Iteration 171's designer looped for 599 s / 182 tool calls /
  1.9M tokens and wrote no file (V1). Iteration 196's Kimi designer researched for 629 s / 110 reads,
  then hit the Ollama weekly limit and wrote nothing (V2). A nothing cannot be upgraded; the loop paid
  for the draft AND the fallback — in 196 an Opus design at $5.11 list price (V3).
- **Design drafts that do exist rarely survive review unchanged.** Quorum round 1 BLOCKED in 171, 187
  and 196, round 2 again in 196, and 169/175/196 ended in a controller carve-out revision (V4). The
  objections were code facts, and the controller "measured both premises before routing" — i.e. the
  verifier re-did the research the draft was meant to save.
- **Constrained drafts do well.** A Kimi planner run took 111 s and measured its baselines rather than
  assuming them; another prototyped a whole design 21 ok / 0 FAIL (V5). Executor output checked by the
  Sonnet judge lands at 88–98, and the judge catches real blocking defects (V6).

**The gap this doc closes:** we cannot tell from banked data what the pattern saves or costs, because
no instrument records what the verifier did to a draft (V13), pi verdicts carry no token or cost
fields (V9), and the slot log has no per-role cost (V12). Routing on it today would be routing on
anecdote.

## Goals

**Primary Goal:** Decide, per mission role and from measurement, where "cheap draft → expensive verify"
lowers cost at equal quality — and route it only there.

**Success Metrics** (pre-registered; the trial reports them whether they pass or fail):
- Per role in the trial: expensive-model spend per landed iteration falls ≥ 30 % against that role's
  baseline, at an evaluator score no lower than baseline minus 3 points (median)
- No rise in downstream defects on routed roles: quorum round-2 rate, evaluator FAIL-first-round rate,
  and dev CI red after merge, each within baseline + 1 occurrence over the trial window
- Verifier acceptance rate is reported with its downstream failure rate, so rubber-stamping is visible
  (a verifier that accepts > 90 % while accepted drafts fail later is the named failure mode)

## Analysis — where it works and why

The saving depends on one question: **can the expensive model check the draft much more cheaply than
it could write it?**

| Work | Is there a mechanical check? | Expected result |
|---|---|---|
| Executor code | Yes: tests, mutation kills, CI, the Sonnet judge | Works (and largely already runs this way) |
| Planner (sprint plan) | Mostly: plan vs the design's ACs; baselines are re-runnable commands | Likely works |
| Designer (design doc) | Weakly: correctness lives in premises about the codebase | Breaks even or loses **unless split** (below) |
| Controller / pick | No: judgement and priority | Keep expensive |
| Evaluator | Its independence *is* the check | Keep independent; never the drafter's vendor |

Four mechanisms decide the outcome:

1. **Reading is cheaper than writing only when the draft carries its evidence.** An Opus design pass in
   196 cost 8.1M cache-read + 92k output tokens (V3). A review of a finished doc plus the files it cites
   is far smaller *if* the reviewer can re-run a sample of the draft's recorded commands instead of
   re-deriving every claim. Hence the **fact-sheet split** for designers.
2. **Anchoring.** Model reviewers lean towards accepting what they are shown. The quorum counters this
   by rejecting by default and requiring a `strongest_objection` (V10); the verifier inherits that
   posture.
3. **Edit-and-upgrade can cost more than writing.** Past some edit fraction you have paid for the cheap
   draft, the expensive read and most of the expensive write. The verifier must discard and regenerate
   above a cutoff, not patch.
4. **Variations help only with a cheap, reliable selector.** Best-of-N executor attempts selected by
   tests is sound. N design variants multiply the expensive review, and a model choosing between
   designs is a weak judge; the cheaper diversity is N cheap critiques of one draft — the quorum, at
   $0.01–0.27 per round (V7).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Which roles enter the Phase 2 trial | Sets where expensive quota is withdrawn | human | design | med |
| D2: Edit-fraction cutoff above which the verifier regenerates instead of patching (proposed 0.33) | Too low wastes drafts; too high pays twice | human | design | low |
| D3: Success thresholds (§Goals) and the stop rule | Pre-registration is what makes the trial a measurement | human | design | low |
| D4: Host mission for the trial (proposed: world) | World already rotates cheap and expensive designers, so a baseline exists | human | design | low |
| D5: Does the designer fact-sheet split replace a rotation turn or run alongside the rotation | Touches the rotation Mark amended on 2026-09-25 | human | design | med |
| D6: Disposition-row schema and where it is banked | Becomes the data every later routing decision reads | agent (fleet) | compile | low |

### Design Freeze

- [ ] D1 — roles in the trial (recommend: **planner** and **designer-split**; executor best-of-2 only as an optional arm when cheap quota has headroom)
- [ ] D2 — edit-fraction cutoff (recommend 0.33 of the draft's changed lines)
- [ ] D3 — thresholds and stop rule as written in §Goals and §Risks
- [ ] D4 — host mission (recommend world)
- [ ] D5 — fact-sheet split alongside the rotation, not replacing a turn (recommend alongside: the rotation's vendor-independence argument stays intact)

## Solution Design

### Overview

Three phases, each gated on the one before. Only Phase 0 is approved by merging this doc.

1. **Phase 0 — retrospective (no code).** Classify every World iteration since 2026-09-07 that used a
   cheap-lane draft: accepted / patched / rewritten / redone / produced nothing, with the cost on each
   side where recoverable. This answers "is the pattern already paying?" from data we have.
2. **Phase 1 — instrument.** Bank a *disposition row* per draft, so the question is answerable
   continuously.
3. **Phase 2 — trial.** Route the D1 roles through draft-then-verify in the D4 mission for ≈15
   iterations, compared against the baseline Phase 0 established.

### Architecture

**The verify contract** (applies to every routed role):

- The drafter must emit, beside its artifact, an **evidence block**: every load-bearing claim with the
  command that proves it, that command's output, and a **kind**:
  - `exact` — the output is a fact of the tree at the draft's base SHA (a file:line lookup, a grep
    count, a symbol's existence). Compared after trimming trailing whitespace; any difference is a
    mismatch.
  - `measured` — the output is a number that varies run to run (a timing, a pass rate). The drafter
    declares a tolerance (default ±10 %); outside it is a mismatch.
  - `observed` — not re-runnable in bounds (an external API response, a long benchmark, a one-off
    event). Never sampled; counted and reported as `unverified_claims`.
- The verifier works **reject-by-default**: it must state its strongest objection before it may
  accept, as quorum reviewers do (V10).
- **The mechanical half is a script, not the model.** The evidence block is machine-readable
  (`<artifact>.evidence.jsonl`: one object per claim with `cmd`, `output`, `kind`, `tolerance`).
  `scripts/mission_verify_evidence.sh` does the sampling, the re-runs, the timeouts and the
  comparison, and writes `verify.json`. The verifier model reads that file and supplies only the
  judgement half (its strongest objection, and the edit). It cannot choose the sample or skip a check.
- **Sampling is fixed:** the first 5 `exact` claims and the first 2 `measured` claims in document order
  (all of them if fewer). Re-runs execute in a worktree at the draft's base SHA, **40 s per command,
  at most 7 commands, 280 s per draft** — the per-draft cap is the sum of the per-command caps, so the
  budget can never cut the sample short. A command that times out or errors is `unverified`.
- **Fail closed.** The draft passes the mechanical floor only if **every sampled claim verified**.
  Any `unverified` sampled claim means the floor is not met. A draft below the floor cannot be
  `accept` or `patch`; its disposition is `unverifiable` and the expensive model either regenerates it
  or parks the role for that iteration. So a refuting command that times out yields `unverifiable`,
  never `accept`, and the same draft cannot bank `accept` on one run and `refuted` on another.
- Disposition is one of `accept` · `patch` (floor met, edit fraction ≤ D2) · `regenerate` (floor met,
  edit fraction > D2; the expensive model writes from the draft's evidence, not its prose) · `empty`
  (no artifact: verdict `empty_worktree`, V8) · `refuted` (a sampled `exact` claim differed, or a sampled
  `measured` claim fell outside its tolerance) · `unverifiable` (floor not met without a refutation).
  The script computes `refuted` / `unverifiable` / floor-met; the model chooses only between `accept`,
  `patch` and `regenerate`, and `edit_fraction` (computed) overrules it above D2.

**Designer fact-sheet split** (the only new role shape):

1. Cheap model: produce `<doc>.facts.md` — premises, file:line citations, measured baselines, each
   with its command and output. No decisions, no solution prose.
2. Expensive model: verify a sample of the facts, then write the design's decisions and solution on
   top of them.
3. Quorum as today, with the author's vendor benched (`--author`).

**Disposition row** (Phase 1; one JSON line per draft, beside the iteration's evidence):

```json
{"mission":"world","iter":203,"role":"planner","drafter":"pi:ollama/kimi-k3:cloud",
 "verifier":"claude:claude-opus-5-5","disposition":"patch","edit_fraction":0.12,
 "draft_sha":"…","final_sha":"…","drafter_cost":{"elapsed_s":111,"tokens":null},
 "verifier_cost":{"usd_list":0.84,"output_tokens":6210,"cache_read_tokens":402113},
 "sampled":3,"refuted":0}
```

`edit_fraction` = changed lines between the draft commit and the final artifact, divided by the
draft's lines (`git diff --numstat`). Token sources, all measured (V16–V18): pi lanes sum
`.message.usage.totalTokens` over assistant `message_end` events in the banked ndjson; Claude lanes read
the `claude -p` result JSON (`usage`, `total_cost_usd`); Codex lanes read `turn.completed.usage` from
`codex exec --json`, which the loops do not pass today (Phase 1 adds it). A field a lane still does not
expose is `null`, never a guessed zero.

### Implementation Plan

**Phase 0 — retrospective** (~0.5 day; attended or fleet)
- [ ] List World iterations since 2026-09-07 with a pi/codex designer or planner (evidence dirs + log)
- [ ] Classify each draft by the disposition vocabulary above, citing the log line
- [ ] Recover cost where banked: Claude lanes from `claude -p` JSON (`total_cost_usd`, usage), pi lanes from the ndjson's `message_end` usage (V16). Codex drafts before Phase 1 have no banked usage (V18): report their elapsed time and mark tokens `null`
- [ ] Write the table into this doc as §Phase 0 Results; Mark decides whether Phase 1 proceeds

**Phase 1 — instrument** (~1 day; fleet)
- [ ] `scripts/mission_verify_evidence.sh <artifact> <base-sha>` — fixed sample, 40 s / 7 / 280 s bounds, fail-closed floor, writes `verify.json`
- [ ] `scripts/mission_draft_disposition.sh <draft-sha> <final-sha> <role> …` writes the row, computes `edit_fraction`, sums tokens per V16–V18, and refuses `accept`/`patch` when `verify.json` says the floor was not met
- [ ] `gate-3-route.md`: Codex role invocations (`:120`, `:132`) gain `--json`, output banked beside the evidence, so Codex usage stops being lost
- [ ] Gate 3 in `.claude/skills/mission-control/resources/gate-3-route.md`: after any drafted role, run it
- [ ] Gate 5 digest `**Cost**` line gains a disposition summary (`draft: planner patch 0.12`)
- [ ] Tests under `tools/launchd/` for the script (bash 3.2)

**Phase 2 — trial** (~1 day build, ≈15 iterations wall clock)
- [ ] Gate 3: the D1 roles route draft → verify under the verify contract; the fact-sheet split for designers
- [ ] Weekly readout against §Goals; the stop rule in §Risks applies at every readout
- [ ] Mark rules per role: adopt, adjust, or drop

### Files to Modify/Create

- `scripts/mission_verify_evidence.sh` — new, ~120 LOC (Phase 1): the mechanical half of verification
- `scripts/mission_draft_disposition.sh` — new, ~80 LOC (Phase 1)
- `tools/launchd/test_mission_draft_disposition.sh` — new, ~100 LOC, wired into `make/test.mk`: covers both scripts, including a timeout that must yield `unverifiable`, never `accept`
- `.claude/skills/mission-control/resources/gate-3-route.md` — verify contract, fact-sheet split (Phase 2)
- `.claude/skills/mission-control/resources/gate-5-retro.md` — disposition in the `**Cost**` line
- `design_docs/planned/m-mission-draft-then-verify.md` — Phase 0 results appended

No Go, no AILANG, no driver change. Every path is in the fleet allowlist
(`scripts/mission_*`, `tools/launchd/**`, `.claude/skills/mission-*`, `design_docs/**`,
`make/test.mk`).

## Conflict Surface

Not a parser/typechecker change, but it overrides shared mission machinery, so the surface is listed:

| Shared thing | Interaction | Decision |
|---|---|---|
| Designer rotation (`gate-3-route.md:17`, V11) | Fact-sheet split adds a cheap first step | Reuse: runs alongside, rotation order unchanged (D5) |
| Quorum author bench (`--author`) | The final author is now the verifier's vendor | Reuse: pass the verifier as `--author`; the drafter's vendor may still review |
| Generator ≠ judge (evaluator independence) | The verifier is a *second generator*, not the judge | Reuse: the evaluator must differ from both drafter and verifier vendor where the pool allows; else FLAG |
| Controller | Opus controller already verifies routed work informally | Override for trial roles only: verification is a separate, recorded step with a disposition |
| Ration gate / fallback chains | A drafted role consumes two buckets | Reuse: each side goes through its own lane's probe and ration; an over-ration drafter falls to "no draft", never to a second expensive run |

## Examples

**Planner, accepted with a patch.** Kimi drafts the sprint plan with measured baselines B1–B5 (as in
iteration 171). Opus re-runs the fixed sample (the exact-kind file and grep claims, and the measured baselines within tolerance) — all match — objects that M2's AC is not falsifiable, fixes that
AC (edit fraction 0.08). Disposition `patch`. Opus spend: one read, not one plan.

**Designer, refuted.** A cheap fact sheet cites `resolver.go:129` as passing a bounded context. The
verifier re-runs the grep: it passes `context.Background()` (the iteration-187 objection). Disposition
`refuted`; the verifier rewrites that premise before any design prose exists, instead of discovering it at
quorum round 1.

## Success Criteria

- [ ] Phase 0 table covers every qualifying World iteration since 2026-09-07, each with a log citation
- [ ] Phase 1 rows are banked for 100 % of drafted roles, with `null` (not 0) for unexposed tokens
- [ ] Phase 2 readout reports every §Goals metric, pass or fail, per role
- [ ] `make test-launchd-drivers` passes with the new suite wired

## Testing Strategy

- Scripts: fixture repos with a known draft→final diff; assert `edit_fraction` and every disposition,
  including `empty`, `refuted`, and `unverifiable` from a command that sleeps past 40 s (it must never
  reach `accept`); assert the sample is identical across two runs of the same draft — mutation-check each branch
- Skill changes: a dry-run iteration (`MISSION_DRY_RUN=1`) must reach Gate 3 with the new text loaded
- Trial validity: baseline and trial iterations are compared on the same mission and queue class
  (PRODUCT vs HARNESS), since HARNESS iterations are shorter and would flatter the trial

## Deferred Decisions

- The disposition-row storage beyond evidence dirs (observatory table vs JSONL) — agent, after Phase 1
- Best-of-N executor selection details (N, tie-break) — only if D1 includes the executor arm
- Whether dispositions feed [m-mission-elo-routing](m-mission-elo-routing.md) as a rating signal

## Non-Goals

- Changing which model fills a role (that is the ELO docs' job)
- Removing the quorum, the evaluator, or the controller's final say
- Any AILANG language or runtime change

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Cheap tier unreliability (quota 429s, empty worktrees) eats the saving | `empty` is a disposition, so its cost is counted, not hidden; an over-ration drafter means "no draft", never a doubled expensive run |
| Rubber-stamping | Reject-by-default posture + sampled re-runs + the acceptance-vs-downstream-failure pairing in §Goals |
| Patch spiral (the expensive model rewrites while calling it a patch) | `edit_fraction` is computed, not self-reported; above D2 it must be `regenerate` |
| Trial degrades a mission's output | **Stop rule:** at any weekly readout, if a trial role's median evaluator score is > 5 points below baseline or two dev-CI reds trace to accepted drafts, that role reverts immediately |
| Confounds (HARNESS vs PRODUCT iterations, queue difficulty) | Compare within class; report n per class; ≈15 iterations is small, so results are directional and say so |

## Verification Log

World citations are to `sunholo-data/ailang-world` `design_docs/world-mission-log.md` at `aa3e36a`.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | Iter 171's designer looped 599 s / 182 calls / 1,918,927 tok and wrote no file | world log :129 | Confirmed |
| V2 | Iter 196's Kimi designer hit the Ollama weekly limit after 629 s / 110 tool calls; verdict `empty_worktree` rc 10 | world log :844; `~/.ailang/state/mission-world-iter196-evidence/pi_design_iter196.ndjson.verdict.json` | Confirmed |
| V3 | The Opus design pass in 196 cost `total_cost_usd` 5.11 (list price; subscription-billed), 8,128,618 cache-read, 91,968 output tokens | `…/iter196-evidence/claude_design_iter196.json` | Confirmed |
| V4 | Quorum r1 BLOCKED in 171, 187, 196; r2 BLOCKED in 196; carve-out revisions in 169, 175, 196 | world log :131, :398, :849, :853, :48, :250, :840 | Confirmed |
| V5 | Kimi planner 111 s / 8 calls with measured baselines (171); a planner prototyped 21 ok / 0 FAIL (187) | world log :133, :555 | Confirmed |
| V6 | 13 "judged N" scores in the world log, range 88–98; iter 175 round 1 FAIL 77 with one blocking finding, fixed | `grep -oE 'judged [0-9]+'` on the log; :262 | Confirmed |
| V7 | Quorum rounds cost $0.01459 (171 r1), $0.21 (196 r1), $0.27 (196 r2) | world log :131, :849, :853 | Confirmed |
| V8 | `mission_pi_run.sh` emits typed verdicts; rc 10 = `empty_worktree` | `scripts/mission_pi_run.sh:58-60` | Confirmed |
| V9 | pi verdict JSON has elapsed, tool executions and changed-file count, but **no token or cost field** | field list of `iter187-evidence/designer.verdict.json` | Confirmed (negative) |
| V10 | Quorum reviewers return `strongest_objection` | `internal/mission/quorum/reviewer.go:45` | Confirmed |
| V11 | Designer rotation is `claude-opus-5-5 → gpt-6-astra → glm-5.3 → kimi-k3` (amended 2026-09-25) | `.claude/skills/mission-control/resources/gate-3-route.md:17` | Confirmed |
| V12 | Slot-verdict log records verdict, rc, elapsed, stamps, controller — **no per-role cost** | `~/.ailang/state/mission-world-slot-verdicts.log` line format | Confirmed (negative) |
| V13 | **No** draft-disposition or edit-fraction instrument exists | two fixed-string greps, `grep -rniF edit_fraction` and `grep -rniF draft_disposition` over `tools scripts internal .claude/skills` (excluding this doc) → 0 and 0; positive control `grep -rniF empty_worktree` over the same paths → 12 | Confirmed (negative, with control) |
| V14 | The ELO docs rank models per role; neither splits a role into draft and verify | read of both docs' problem statements | Confirmed |
| V15 | The cost-per-verified-success KPI covers eval-benchmark cohorts only, not mission roles, so Phase 1 computes its own | `cmd/ailang/chains_stats.go:58`; `internal/observatory/cost_per_verified_success.go:131` reads a frozen cohort via `QueryEvalResults` | Confirmed |
| V16 | pi token usage is recoverable from the banked ndjson | `iter187-evidence/designer.ndjson`: usage on 49 assistant `message_end` events; summed `totalTokens` = 1,952,024; `cost` fields are 0 on the flat-rate lane | Confirmed |
| V17 | Claude lanes bank `usage` and `total_cost_usd` | V3; also `iter185-evidence/designer_out.json` (`total_cost_usd` 2.87) | Confirmed |
| V18 | `codex exec --json` emits `turn.completed` with `usage` (input, cached, output, reasoning tokens); the loops' Codex invocations do not pass `--json` | one-line probe 2026-09-28: `{"type":"turn.completed","usage":{"input_tokens":18962,"cached_input_tokens":8960,…,"output_tokens":5}}`; `grep -c -- --json` over the Codex invocations in `gate-3-route.md`, the driver and `scripts/mission_*.sh` → 0 | Confirmed |

No premise is left pending: the three open in the first draft (pi tokens, the KPI's scope, Codex usage) are V15–V18.

## Quorum History

- **Round 0 (2026-09-28): BLOCKED 3/3**, controller pass. astra: V13's `\|` inside `grep -E` would not have proved absence as written. gemini: P1–P3 unmeasured. glm: the verify contract had no time bound and no definition of mismatch, so `refuted` was not deterministic and the A1/A5 scores were unearned. **All three accepted and fixed in-session**: V13 re-run as fixed-string greps with a positive control; P1–P3 measured (V15–V18); the contract gained claim kinds, a fixed sample, 60 s / 5 min bounds and an `unverified` outcome.
- **Round 1 (re-quorum, 2026-09-28): BLOCKED 3/3**, controller pass. astra: `unverified` claims still
  let a draft be accepted, so the contract did not fail closed. gemini: an LLM cannot enforce fixed
  sampling, exact matching or wall-clock limits on itself. glm: 7 × 60 s exceeded the 5-minute cap, the
  truncation case was unspecified, and a timeout could dodge a refutation, so dispositions were not
  reproducible. **All three accepted and fixed in-session:** the mechanical half moved into
  `scripts/mission_verify_evidence.sh`; bounds are now 40 s × 7 = 280 s, so the budget cannot truncate
  the sample; a fail-closed floor (every sampled claim verified) with a new `unverifiable` disposition
  means a timeout can never produce `accept`. The re-quorum-once guardrail is spent, so these fixes go
  to Mark for ratification rather than a third round.

## Related Documents

- [m-mission-elo-routing](m-mission-elo-routing.md) — which model per role, by rating. Distinct: this
  doc changes a role's *shape*; dispositions could feed its ratings (Deferred)
- [m-mission-role-elo-and-tier-order](m-mission-role-elo-and-tier-order.md) — evidence-ordered fallback
  chains. Distinct for the same reason
- [m-quota-rationing-routing](m-quota-rationing-routing.md) — the ration gate each side of a drafted role passes through
- [m-harness-mission-loop](m-harness-mission-loop.md) — the fleet mission that would build Phases 1–2

## References

- World mission log and evidence directories (V1–V7)
- Analysis given to Mark in session, 2026-09-28

## Future Work

- If the fact-sheet split works for designers, apply it to the evaluator's evidence gathering (the judge
  verifies a cheap evidence pack rather than collecting it)
- Best-of-N executor selection by tests once cheap-lane quota is reliable
