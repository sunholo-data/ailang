# V1 Mission — work the backlog to a v1.0.0 release

**Type**: Long-running mission (peer of [motoko-mission.md](motoko-mission.md)); advanced by a
scheduled nightly outer loop on the always-on rig, coordinated by Fable.
**North star**: Ship AILANG v1.0.0 — a release whose bar is *written down, met, and verified*,
with the backlog worked through the honed inner loop (design-doc → sprint-plan → execute → evaluate)
rather than ad-hoc sessions.
**Traces to**: [PROGRAM.md](PROGRAM.md) — this mission is an operational instance of the program's
loop; every friction found here routes to a lane (skill fix / process fix / backlog item).
**Skill**: [.claude/skills/mission-control/SKILL.md](../.claude/skills/mission-control/SKILL.md)
runs ONE iteration. **Scheduling (since 2026-07-10): the Claude Code scheduled task
`v1-mission-nightly`** (22:00 daily, managed in the app's Scheduled sidebar) — it runs inside
Claude Code with the subscription OAuth, so the billing question never arises. Runs while the
app is open (this rig's daemon is always on; a missed slot fires on next launch). The launchd
path (`tools/launchd/mission-control.sh` + plist) is the UNLOADED fallback — if ever re-armed
it hard-requires `CLAUDE_CODE_OAUTH_TOKEN` (billing guard) since headless CLI outside the app
cannot use the Keychain OAuth.
**Log**: [v1-mission-log.md](v1-mission-log.md) — append-only, one entry per iteration.
**Human-facing reporting**: GitHub issue
[#329](https://github.com/sunholo-data/ailang/issues/329) — every iteration posts its morning
report there as a comment (Mark follows by email via issue subscription, no Claude login
needed); driver crashes post there too.

---

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
3. [NEXT] m-feedback-triage-gate (P0, public endpoint cost/safety, 2d)
4. m-diagnostic-coverage (P0, cheapest cost-per-success lever, 3–4d)
5. m-v1-stability-promise (NEW — design doc via design-doc-creator: the 1.x stable surface,
   LIMITATIONS.md accuracy pass, confirm/retract remaining vX promises in docs)
6. m-effect-refinement **decomposition** (split ~90h into ≤3–4d sprint docs; execution follows
   as individual queue items)
7. m-eval-frontier-tier (P1, suite no longer discriminates frontier, 2.5–3.5d — eval-bar clause)
8. m-check-strict-fallbacks (P1, silent-failure class, ~1d — core-frozen clause)
9. m-bytecode-vm-parity-bugs (P1, blocks bytecode parity gate, 1–2d — correctness)

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
