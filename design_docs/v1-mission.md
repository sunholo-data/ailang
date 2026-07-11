# V1 Mission — work the backlog to a v1.0.0 release

**Type**: Long-running mission (peer of [motoko-mission.md](motoko-mission.md)); advanced by a
scheduled nightly outer loop on the always-on rig, coordinated by Fable.
**North star**: Ship AILANG v1.0.0 — a release whose bar is *written down, met, and verified*,
with the backlog worked through the honed inner loop (design-doc → sprint-plan → execute → evaluate)
rather than ad-hoc sessions.
**Traces to**: [PROGRAM.md](PROGRAM.md) — this mission is an operational instance of the program's
loop; every friction found here routes to a lane (skill fix / process fix / backlog item).
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md)
runs ONE iteration. **Scheduling: launchd `dev.ailang.mission-control`** (CONTINUOUS since
2026-07-10 per Mark: StartInterval 2h + overlap guard = back-to-back iterations, ≤2h idle; was
22:00-nightly for the first supervised runs) behind the billing guard — API keys are stripped from the environment
(subscription-or-nothing by construction) and a cheap auth probe runs first: keychain OAuth
suffices while the rig is logged in (verified 2026-07-10); `CLAUDE_CODE_OAUTH_TOKEN` in
secrets.env is an optional belt-and-braces for post-reboot login screens. Probe failure refuses
loudly with zero spend. The Claude Code
scheduled-tasks path was TESTED AND RULED OUT for this job (2026-07-10 canary): that system is
desktop-side — tasks landed on /Users/mark (Mark's machine, not the rig) and a probe task never
dispatched even there (a June one-time task was also found a month overdue). Wrong machine +
unreliable dispatch → launchd is primary, not fallback.
**Log**: [v1-mission-log.md](v1-mission-log.md) — append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue
[#329](https://github.com/sunholo-data/ailang/issues/329) — every iteration posts its morning
report there as a comment (Mark follows by email via issue subscription, no Claude login
needed); driver crashes post there too.

---

## STATUS 2026-07-11 (day) — ITERATION 7 COMPLETE: EVAL-BAR CLAUSE MACHINERY LANDED (frontier tier + curation)

m-eval-frontier-tier landed via the full loop headless, round-1 clean (third consecutive
round-1-clean full loop). The suite regains discrimination structure: `frontier` tier exists
(8 benchmarks re-tiered with parked-validation provenance), 7 saturated core benchmarks demoted
to stretch via a conservative 4-dimension rule computed ONLY from banked re-graded v0.25.0 data
(now codified in CURATION.md §5, both directions), and the decision_block_capture free-text
exact-match anti-pattern is retired via a new `grading: prefix_line` structural grader
(GradeStdout centralizes 6 call sites). Eval PASS 96/100 round 1 with independent
distinct-sample recount (5 benchmarks × 4 dims from raw JSONs — all matched). PR #339 →
0515578ae, dev CI green per-workflow. Tier distribution now smoke 23 / core 19 / stretch 29 /
frontier 8 / vision 9. **The eval-bar clause is NOT fully closed**: frontier-failure validation
(each of the 8 must fail ≥1 frontier model, else demote back) is API-billed → PARKED for
human/next frontier rotation; 4 sketched benchmarks remain unauthored. Next: #9
m-effect-mode-validation (effect-refinement sprint 1/4).

## STATUS 2026-07-11 (morning) — ITERATION 6 COMPLETE: EFFECT-REFINEMENT DECOMPOSED (last strategic v1.0 item now sprint-sized)

Queue #7 executed as a decomposition iteration (standing rule for multi-week items; Fable lane —
no Opus sprint needed). Reality check found the parent doc's ~90h claim STALE by more than
a third: Phases 1+2 AND the AI port shipped v0.15.0 (M-EFFECT-REFINEMENT-PHASE1 +
M-AI-EFFECT-MODES), Phase 7's CryptoRand alias was scope-reduced away in v0.15.0 because
**M-CRYPTORAND never landed at all** — its doc sits in implemented/v0_15_0 with "Status:
Planned", swept there by the 48-doc bulk relocation 645467e13 (header corrected to Superseded).
Remaining ~64h decomposed into 4 sprint docs (all premises live-verified at
v0.28.0-148-g6c25f45e9): m-effect-mode-validation (1d) → m-effect-replay-contracts (3d) →
m-effect-clock-net-fs-modes (3d) → m-effect-scope-params (2.5d, release-gate re-score
candidate). Phase 6 (M-ENTROPY) routed OUT — ships with M-ENTROPY itself, not v1.0-required.
**Discovered en route**: the public parameterised-effects guide claims "the typechecker rejects
unknown values" — FALSE (`Rand[mode=banana]` passes `ailang check`, live transcript in sprint-1
doc); interim accuracy note shipped, enforcement is sprint 1. With this, every required-for-v1
queue item is sprint-sized: the bar's critical path is eval-frontier-tier (#8, [NEXT]) + the
four effect sprints + two 1–2d P1s.

## STATUS 2026-07-10/11 (night) — ITERATION 5 COMPLETE: STABILITY-PROMISE BAR CLAUSE CLOSED

m-v1-stability-promise landed via the full loop headless, round-1 clean at every stage — the
first queue item of the mission that was genuinely NEW (no stale status to catch; instead the
Opus planner caught 2 premise errors in Fable's design doc: stdlib is 42 modules not 39, and
LIMITATIONS.md is double-maintained with a diverged public website copy — both copies fixed).
Shipped: the 1.x stability promise page (Stable/Experimental/Internal tiers, full stdlib + CLI
tier tables, RATIFICATION-pending stamp), live-verified accuracy pass on BOTH LIMITATIONS files
(every entry re-verified at HEAD with transcripts; poly-arith-lambda and match-in-HOF moved to
Recently Resolved — they had been documented as broken for ~15 minor versions), 4 stale website
version-promises retracted. Eval PASS 96/100 round 1 (independent distinct-sample verification).
PR #337 → fcccd7208, dev CI green per-workflow. Two bar clauses now satisfied: "stability
promise defined" ✓ and LIMITATIONS-accuracy under core-frozen ✓ (ratification of tier
ASSIGNMENTS parked for Mark at the v1.0.0 release gate — not a merge blocker). Backlog: issue
#338 (deflake TestRunCommand_PipedStdoutFlushesPerLine — hit twice this iteration, proven
non-regression twice). Critical path remaining: effect-refinement decomposition (#7, [NEXT]) →
eval-frontier-tier (#8).

## STATUS 2026-07-10 (night) — ITERATION 4 COMPLETE: LAST SPRINT-SIZED P0 CLOSED FOR V1

m-diagnostic-coverage reality-check found its "Planned" status STALE (M1–M3 shipped 2026-07-09,
ff58a3259/e59197554 — second stale-status catch of the mission); the genuinely open remainder ran
as sprint M-DIAG-FIXTURE-PROMOTION via the full loop headless: Opus plan (2 real discrepancies
found live: %/Fractional's claimed diagnostic is UNREACHABLE via `ailang check`; stdlib-hint
fixture needs TestMain wiring) → Opus execute in worktree → Fable eval **PASS 96/100 round 1**
(non-vacuity proven twice, independently). 4 footgun rows promoted to `covered` → 7 CI-enforced
fixtures. Integrated via **PR #336 → fe807aac8** — a sibling agent held a conflicted merge open
in the shared main tree all iteration, so integration went worktree-branch→PR→auto-merge without
ever committing in the main tree (new Gate-2 rule 4 codifies this). Dev CI green per-workflow on
the merge. DEFERRED with rationale in the design doc: prompt-deletion pass + rig A/B (35
deletable lines < the ≥100 gate — widen coverage first); PARKED for human: haiku causal re-run
(API-billed, blocked by the headless billing guard). With this, all four sprint-sized P0s from
the ratified bar are closed; the critical path is now the stability promise + effect-refinement
decomposition + eval-frontier tier.

## STATUS 2026-07-10 (evening) — ITERATION 3 COMPLETE: P0 OPERATIONALLY CLOSED (CODE-SIDE), DEV UN-REDDED

m-feedback-gate-cloud-adapter landed via the full loop headless: Fable design doc (all 6
premises later verified accurate by the Opus planner — zero discrepancies, a first), Opus
plan + execute in isolated worktree, **first round-1 evaluation FAIL of the mission** (a
fail-open bug: Firestore read errors silently reset cooldown/budget windows; numeric 92 but
CLAUDE.md CP2 policy violation) → surgical round-2 fix → PASS 97/100, merge 842d7d501.
En route, OBSERVE's "dev green" turned out WRONG — dev had been red since 12:57Z behind a
dependabot-flooded run list (Gate-1 gap, skill-fixed): three pre-existing breaks fixed forward
(Windows codegen escape bug in contract panic literals 9d2e32ac1 + docs npm peer-dep pin +
go-test timeout 60s→300s 4c22032de). Dev fully green on 4c22032de. CALIBRATED: gate code is
now complete INCLUDING cloud adapters, still off by default; production activation awaits
HUMAN ops (terraform TTL + ANTHROPIC_API_KEY secret in sibling repo, then DRY_RUN week 1) —
parked in #329.

## STATUS 2026-07-10 (afternoon) — ITERATION 2 COMPLETE: FIRST HEADLESS FULL-LOOP RUN, P0 GATE LANDED

m-feedback-triage-gate landed via the full inner loop with NO human present — the friction
flagged at 13:30 (headless run reached planning without routing evidence) is answered: both
plan and execute carried verified `claude-opus-4-8` attestations, evaluation by Fable with
independent re-verification (full make test rc=0 in worktree, drill run live). Eval 93/100 PASS
round 1, merge 40f1cdc3f. The killed 13:03 run's leftovers (dead-locked worktree + uncommitted
scaffold) were quarantined and cleared. CALIBRATED: gate logic is complete and merged, off by
default; production protection is rules-only until m-feedback-gate-cloud-adapter (queued [NEXT])
ships the Firestore adapters — the P0 is not operationally closed yet. New "Dev stays GREEN"
guardrail honored: merge pushed, remote CI verified before tagging [LANDED] (result recorded in
the queue entry).

## STATUS 2026-07-10 (13:30) — SCHEDULING MOVED INTO CLAUDE CODE; BILLING INCIDENT CLOSED

The first kickstarted headless run billed ~13 min of API credits (`ANTHROPIC_API_KEY` leaked
from secrets.env into `claude -p`; killed rc=143). Two fixes: (1) the launchd driver gained a
billing guard (strips API keys, refuses without a subscription token) and was then (2)
**superseded as primary by the Claude Code scheduled task `v1-mission-nightly`** — runs inside
the app on the session OAuth, no token handling at all. The killed run's unreviewed plan
artifacts for m-feedback-triage-gate were deleted (produced in 13 min, no Opus routing
verified, no controller review) — tonight replans through the proper loop. Friction recorded:
the headless controller reached Gate 3 planning without evidence it honored the model routing —
watch for this in tonight's #329 report.

## STATUS 2026-07-10 (evening) — ITERATION 1 COMPLETE: FIRST FULL INNER-LOOP RUN, 2 QUEUE ITEMS LANDED

- **1a** m-named-test-blocks closed out (shipped 2026-07-09, verified live incl. duckdb's
  formerly-silent-skipped tests now 2/2; deontic criterion deferred — package absent locally).
- **1b** m-typeenv-sub-fix RESOLVED via the full loop: Opus planner (found 6 stale design-doc
  items incl. 2 interacting post-doc fixes) → Opus executor in isolated worktree (proved the P0
  no longer reproduces; declined the 135-LOC repair; shipped regression guards instead) → Fable
  evaluation 92/100 PASS with adversarial non-vacuity proof (tests FAIL at the bug-live commit,
  PASS from M-TYPE-LIST-SOUND round 3). Merged f59421ac8.
- **Found + fixed en route**: stale `bin/ailang` (v0.26.0) silently breaking `make test` on this
  rig — test helpers prefer `bin/ailang`; systemic fix spun off as a background task.
- **Retro executed the ≥2 rule for real**: three same-class frictions (stale binaries ×2,
  parked-test-as-status ×1) → ONE skill edit: Gate-2 verification protocol added to
  mission-control (rebuild both binaries, un-skip-and-run parked tests, PIPESTATUS).
- Routing evidence rows 1–2 recorded (Opus plan + execute: both high quality; 1 attribution
  correction at evaluation).

## STATUS 2026-07-10 (later) — ITERATION 0 COMPLETE: BAR RATIFIED, BACKLOG RE-SCORED

Mark ratified the v1.0 bar (interactive session) and made the scope calls:
- **Effect refinement IN** (public docs promise; decompose ~90h into sprints before executing);
  **effect handlers OUT → v1.1** (largest new surface; no bake time under a fresh stability
  promise is the risky combination).
- **CSP session types, quasiquotes, perf4-bytecode, D4 all OUT → v1.1** (plus agent-orchestration
  and zero-language-learnings by ordering policy — multi-week strategic, not selected IN).
- **Both v1_0_0 P0s downgraded to P1/nice-for-v1** (global-collaboration-hub,
  m-eu-compliance-effects — multi-week non-language items; dated notes in the docs).
- Reality-check finding: **m-named-test-blocks core scope already SHIPPED** (M1 commits
  ec4996e45/7389e84c1 + fixes; verified live post-rebuild: failing named test → FAIL + exit 1).
  Reduced to a closeout item. m-feedback-triage-gate confirmed genuinely open (the shipped
  M-MCP-EDGE-THROTTLE rate limit is its precondition, not its scope).

## STATUS 2026-07-10 — MISSION INITIALIZED, ITERATION 0 PENDING

Exploration findings that shaped this charter (full census in log entry 0):

- **No written v1.0.0 criteria exist anywhere.** Scope is currently implicit folder membership:
  66 docs in `planned/v0_29_0/`, 27 in `planned/v1_0_0/`, 4 in `planned/v1_1_0/`. Iteration 0
  must define the bar before any sprint is picked.
- **6 P0s are open**: m-typeenv-sub-fix (type-safety hole), m-feedback-triage-gate (public-endpoint
  cost/safety), m-named-test-blocks (silent-green test runner), m-diagnostic-coverage (cheapest
  cost-per-success lever) in v0_29_0; global-collaboration-hub, m-eu-compliance-effects in v1_0_0.
- **The inner-loop skills are sound but had no model policy and no self-improvement path** —
  `dev-cycle.md` pinned `model: sonnet` (fixed 2026-07-10 → opus), retros written to
  `docs/sprint-retros/` were never folded back into the skills.

## CURRENT GOAL

1. **Iteration 0 (definition)**: write the v1.0 bar (see "The v1.0 bar" below — draft to be
   ratified by Mark), re-score all 93+ planned docs against it into: `required-for-v1` /
   `nice-for-v1` / `post-v1`. Output: updated folder assignments + ordered queue in this doc.
2. **Then**: work the queue P0-first through the inner loop, one sprint-sized item per iteration,
   recording routing evidence every time.

## The v1.0 bar (RATIFIED 2026-07-10, Mark)

A v1.0.0 declaration requires:
- **Zero open P0s** (at ratification: 3 open sprint-sized — typeenv-sub-fix, feedback-triage-gate,
  diagnostic-coverage — plus the named-test-blocks closeout).
- **Language core frozen**: no known type-safety holes (m-typeenv-sub-fix class), parser/typechecker
  regressions gated by CI, LIMITATIONS.md accurate.
- **The eval bar**: frontier models ≥ Python-parity on the standard suite (already ~even after
  regrade); agent-mode suite discriminating (not saturated) with published dashboard
  (→ m-eval-frontier-tier).
- **Stability promise defined**: what syntax/stdlib/CLI surface is stable in 1.x, written into docs
  (→ new design doc m-v1-stability-promise, queued).
- **Strategic items — DECIDED**: effect refinement IN (docs promise; decomposed first); effect
  handlers, CSP session types, quasiquotes, perf4-bytecode, D4 OUT → v1.1.

## How the mission runs (each iteration — codified in the mission-control skill)

1. **OBSERVE** — read this doc's backlog + last log entry + agent inbox + eval health. Deterministic, cheap.
2. **PICK** — top open queue item per the ordering policy. **Verify against repo reality first**
   (git log + code + tests), never trust a status header — stale-status docs are how we shipped
   M-EVAL-BENCH-UI twice (2026-07-10 lesson: doc said Planned, all 4 milestones were long done).
3. **ROUTE + EXECUTE** — through the honed inner loop with the model routing policy below:
   design-doc-creator → sprint-planner → sprint-executor → sprint-evaluator. Sprint work runs in
   an isolated git worktree (concurrent-agent safety). Max 3 evaluator rounds, then park as
   `needs-human-review` and move on.
4. **RECORD** — append a log entry (fixed template in v1-mission-log.md): what shipped, evaluator
   score, routing evidence row, ruled-out ledger additions, next.
5. **RETRO** — route observed friction into exactly one lane: **skill fix** (edit the offending
   SKILL.md — max ONE skill edit per iteration, each traced to ≥2 recorded frictions), **process
   fix** (edit this doc), or **backlog item** (new/re-prioritized design doc). Then send the
   morning report to controlplane.

## Model routing policy (evidence-updated, not vibes)

| Role | Model | Why / evidence |
|---|---|---|
| Mission controller (this loop: triage, pick, judge, retro) | **Fable** (claude-fable-5) | Strategy + judgment work; the nightly headless session |
| Design docs (create/review) | **Fable** | Spec quality gates every downstream token |
| Sprint planning | **Opus** (claude-opus-4-8) | Plan quality determined execution success historically |
| Sprint execution | **Opus** — the default, per Mark 2026-07-10 | Sonnet execution was a false economy (needed corrections); also `dev-cycle.md` had silently pinned sonnet |
| Sprint evaluation | **Fable** | Independent judge ≠ the model that wrote the code |
| Mechanical tasks (doc moves, regen, banking) | Sonnet allowed | Only with deterministic verification; promotion beyond this requires evidence |

**Evidence rule**: every sprint's log entry records `(model, task class, evaluator round-1 score,
rounds-to-pass, corrections)`. A routing change (either direction) requires ≥3 data points and is
made in RETRO, recorded here with a dated stamp.

## Rig integration — the two-tier rule

`rig.lock` (`~/.ailang/state/rig.lock.d`) is a **GPU mutex, nothing more** (Mark, 2026-07-10).

1. **Default iteration (cloud models: Fable/Opus coding, `make test`, git): NEVER touches
   rig.lock.** CPU/disk co-tenancy with the eval rotation is fine; the loop runs 24/7 without
   starving the rotation and vice versa.
2. **GPU-touching steps only** (a sprint whose acceptance includes local-model validation, wire
   diagnostics, anything driving ollama): `rig_lock_acquire wait` for **that step only** — never
   held across a whole sprint. Same discipline as `os-rotation-filler.sh`.

Hygiene: a sprint must not *accidentally* reach the GPU (the port-8080-zombie class). "Does this
step touch the GPU?" is an explicit routing question in the skill, not an accident of what a test
invokes.

## Guardrails (the loop may not…)

- **No releases.** release-manager stays human-triggered; the loop stops at "ready-to-release"
  and reports.
- **No pushes without account check** (`gh auth status` → `sunholo-voight-kampff`).
- **No work on a dirty main worktree** — sprints run in coordinator-managed worktrees; the
  controller session itself is read-mostly + doc edits.
- **Budgeted**: hard wall-clock kill in the driver (default 6h); one backlog item per iteration.
- **Kill switch**: `touch ~/.ailang/state/mission-control.disabled` (checked in preflight) or
  `launchctl unload ~/Library/LaunchAgents/dev.ailang.mission-control.plist`.
- **Subscription billing only** (2026-07-10): the nightly bills the Claude subscription via
  `CLAUDE_CODE_OAUTH_TOKEN` (`claude setup-token`, stored in secrets.env) — the driver strips
  `ANTHROPIC_API_KEY` and refuses to start without the token. The first kickstarted run billed
  ~13 min of API credits before this was caught; never again.
- **Escalation**: evaluator `needs-human-review`, merge conflicts, or any guardrail trip →
  `ailang messages send controlplane`, park the item, pick the next; never force through.
- **Skill edits**: max one per iteration, ≥2 recorded frictions each, called out in the morning
  report (git history is the rollback).
- **Dev stays GREEN** (2026-07-10, Mark): an item is not [LANDED] until remote CI passes on its
  merge commit (Gate 3b — local gates miss fmt-check/govulncheck/file-sizes/docs build), and a
  red dev CI outranks the queue at OBSERVE, including time-based reds from newly published vuln
  advisories.

## Backlog ordering policy

1. Open **P0s** first (list above), oldest-known-risk first.
2. **Unblockers** — items other queued items depend on (e.g. m-effect-row-poly-params blocks
   sunholo/demos).
3. **P1 by impact-per-day** (the census has estimates; prefer ≤3-day items to keep iterations
   sprint-sized).
4. **Strategic multi-week items enter only after decomposition** into sprint-sized design docs
   (a decomposition is itself a valid iteration deliverable).
5. Anything re-scored `post-v1` in iteration 0 leaves the queue.

## Queue (top = next; tags: [NEXT] [IN-SPRINT] [PARKED] [LANDED] [RULED OUT])

**Required-for-v1 (the bar's critical path):**

1. [LANDED 2026-07-10] m-named-test-blocks closeout (iteration 1a; deontic criterion deferred,
   package absent locally)
2. [LANDED 2026-07-10] m-typeenv-sub-fix (iteration 1b: RESOLVED — pre-closed by adjacent
   M-TYPE-LIST-SOUND work, regression-guarded, eval 92/100, merge f59421ac8)
3. [LANDED 2026-07-10] m-feedback-triage-gate (iteration 2: full loop headless, eval 93/100
   PASS round 1, merge 40f1cdc3f, remote CI green on dev post-merge; gate logic complete +
   merged, off by default — production activation gated on the next item)
4. [LANDED 2026-07-10] m-feedback-gate-cloud-adapter (iteration 3: full loop headless, round-1
   eval FAIL → round-2 PASS 97/100, merge 842d7d501, dev CI fully green on 4c22032de; gate
   code complete incl. cloud adapters, OFF by default — production enablement is a HUMAN ops
   task: sibling-repo terraform TTL + ANTHROPIC_API_KEY secret, then DRY_RUN=1 week 1)
5. [LANDED 2026-07-10] m-diagnostic-coverage (iteration 4: M1–M3 found pre-shipped 2026-07-09
   under a stale "Planned" status; remainder sprint M-DIAG-FIXTURE-PROMOTION promoted 4 rows
   to covered — 7 CI fixtures across 6 footgun rows — eval PASS 96/100 round 1, PR #336 →
   fe807aac8, dev CI green per-workflow. DEFERRED, rationale in doc: deletion pass + rig A/B
   until deletable surface ≥ 100 lines; PARKED for human: haiku causal re-run, API-billed)
6. [LANDED 2026-07-10] m-v1-stability-promise (iteration 5: FULL loop headless round-1 clean —
   Fable design doc → Opus plan (caught 2 premise errors: 42 modules not 39; LIMITATIONS
   double-maintained + diverged, both copies fixed) → Opus execute in worktree → Fable eval
   PASS 96/100 round 1. Stability page docs/docs/reference/stability.md (3 tiers, full stdlib +
   CLI tables), both LIMITATIONS files live-verified at HEAD, 4 website vX-promises retracted,
   PR #337 → fcccd7208, dev CI green per-workflow. PARKED for human at RELEASE gate: tier-
   assignment ratification — ⚠ proposed: std/net, crypto, jwt, xml, zip, process, CLI
   watch/serve-api)
7. [LANDED 2026-07-11] m-effect-refinement **decomposition** (iteration 6: repo-verified phase
   census — P1/P2 + AI port shipped v0.15.0 under the parent's stale "Planned"; P7 CryptoRand
   never existed (m-cryptorand.md swept to implemented/ in error — header corrected); P6 routed
   OUT to M-ENTROPY. Remaining ~64h split into 4 sprint docs (below, items 9/12/13/14) with live-
   verified premises; parent doc now the umbrella. BONUS finding: the public guide's "typechecker
   rejects unknown values" is FALSE (`Rand[mode=banana]` passes check) — interim accuracy note
   shipped, enforcement is sprint 1)
8. [LANDED 2026-07-11] m-eval-frontier-tier (iteration 7: full loop headless, round-1 clean —
   Opus plan (9 discrepancies) → Opus execute (frontier tier + 8 re-tiered + prefix_line
   structural grader + 7 core→stretch demotions via 4-dim rule from banked data) → Fable eval
   PASS 96/100 round 1 w/ independent distinct-sample recount. PR #339 → 0515578ae, dev CI
   green per-workflow. PARKED for human: frontier-failure validation of the 8 (API-billed —
   each must fail ≥1 frontier model or demote back per CURATION.md §5) + 4 remaining sketches)
9. [NEXT] m-effect-mode-validation (P1, ~1d — effect-refinement sprint 1/4; makes the public
   guide's closed-set claim true; prerequisite for the other three)
10. m-check-strict-fallbacks (P1, silent-failure class, ~1d — core-frozen clause)
11. m-bytecode-vm-parity-bugs (P1, blocks bytecode parity gate, 1–2d — correctness)
12. m-effect-replay-contracts (P1, ~3d — effect-refinement sprint 2/4; Rand modes get real
    runtime semantics incl. crypto; supersedes never-landed M-CRYPTORAND)
13. m-effect-clock-net-fs-modes (P1, ~3d — effect-refinement sprint 3/4; surfaces existing
    AILANG_SEED/sandbox runtime switches at type level)
14. m-effect-scope-params (P1, ~2.5d — effect-refinement sprint 4/4; weakest v1.0 forcing
    function, flagged as release-gate re-score candidate for Mark)

**Nice-for-v1** (worked opportunistically if the critical path is blocked): the remaining
`planned/v1_0_0/` docs (incl. the two downgraded P1s) and `planned/v0_29_0/` P1s — those ship on
the normal v0.29 road regardless; the queue only tracks what gates v1.0.

**Post-v1**: everything in `planned/v1_1_0/` (7 strategic docs moved there in iteration 0).

## Ruled out / resolved

- **Sonnet as default executor** — ruled out 2026-07-10 (Mark: corrections needed; false economy).
  Re-entry only via the evidence rule.
- **Scheduling via cron / scheduled-tasks MCP** — ruled out; this rig's substrate is launchd
  (nightly-eval + os-rotation-filler precedents), and the coordinator has no internal timer.

## Done / superseded

*(nothing yet — mission initialized 2026-07-10)*
