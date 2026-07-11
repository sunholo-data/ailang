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

## STATUS 2026-07-11 (evening) — CONTROLLER MODEL: Fable → Opus (quota relief, Mark)

The outer-loop controller runs on **Opus** (`claude-opus-4-8`) through a **time-boxed override**
that AUTO-REVERTS to Fable at **Mon 2026-07-13 07:00 CEST** (when Fable quota resets) — no session
or human needed. Mechanism: driver default is Fable; `~/.ailang/state/mission-model` holds
`claude-opus-4-8 <expiry-epoch>`; the first iteration past expiry deletes it, falls back to Fable,
and posts to #329. Reason: Fable quota relief (Mark). Both paths test-verified. Consequence in the
routing table: while on Opus, evaluation is model-homogeneous (Opus judges Opus) — mitigation
available (distinct-model skeptic evaluator) but left off. To go back to Fable EARLY (more quota
sooner): `rm ~/.ailang/state/mission-model`. To extend Opus: bump the epoch in that file.

## STATUS 2026-07-11 (night) — ITERATION 11 COMPLETE: m-stdlib-regex LANDED (AILANG now has linear-time RE2 regex; bar clause 4's regex half closed)

Full loop headless, round-1 clean (sixth consecutive). `std/regex` ships — `compile → Result[Regex,
string]` + total `isMatch`/`findFirst`/`findAll`/`replaceAll`/`split`, backed by Go's `regexp` (RE2,
linear-time by construction; `(a+)+$` < 100ms). RE2 subset (no backref/lookaround) → `compile` Err,
never panics. Opus plan→execute→evaluate; eval PASS 97/100 round 1 with INDEPENDENT reproduction
(CJK rune-span proof of the byte→rune conversion). PR #343 → squash-merge 0b0ed7ea0, all required
checks green. Builtins landed in the modern `internal/builtins/` system (D-ARCH: the doc's
`internal/eval` path was outdated). Next: queue #12 m-stdlib-url-parse (clause 4's other half).
See log entry 12.

## STATUS 2026-07-11 (night) — ITERATION 10: m-stdlib-regex DESIGN DOC CREATED (clause-4 regex, NEW-DOC stage)

Queue #11 (NEW-DOC) routed to design-doc-creator; deliverable is the verified design doc, next
scheduled iteration plans+executes. `planned/v0_30_0/m-stdlib-regex.md` created — a linear-time
(RE2) regex builtin closing the first half of bar clause 4's "linear-time regex + URL-parse
builtins" mandate. **Scope-defining decision: wrap Go's stdlib `regexp` (which IS RE2 —
linear-time guaranteed) instead of hand-rolling an engine → a 2-day builtin, not a multi-week
project.** Two-stage API (`compile -> Result[Regex,string]` then total match fns), RE2 subset
(no backreferences/lookaround — the documented price of linearity), all pure `! {}`. Hard gates
passed: binary rebuilt (v0.29.2-29-gc533bb51c = `git describe`), the `Regex`/`RegexMatch` + 6
signatures `ailang check`-clean; Conflict Surface = purely additive (namespace-collision only,
no grammar); Axiom net +8. Dev CI green per-workflow on HEAD (the two prior tier-drift reds were
fixed by c423490d8, included in the green HEAD run). No skill fix (clean stage). Next: #11
sprint-planner → executor → evaluator.

## STATUS 2026-07-11 (evening) — ITERATION 9 COMPLETE: M-SYNTAX-AI-FORGIVING LANDED (the ~32% small-model parse-failure class is dead)

m-syntax-ai-forgiving landed via the full loop — the first iteration split across two scheduled
runs (run A: reality-check + Opus plan + Opus execute, died pre-evaluation; run B resumed cleanly
at sprint-evaluator from the committed artifacts). The parser now accepts both statement-separator
forms small models naturally write: R1 `;`-sequences in `=` bodies (kills PAR017) and R2
newline-as-soft-separator in blocks (narrows PAR020 to the genuine same-line case, actionable
error preserved). Backward-compatibility PROVEN, not asserted: corpus AST-diff fuzz gate
(old parser rebuilt from pinned base via temp worktree + new cmd/astdump) = zero re-parse diffs
over all 389 currently-valid corpus files, re-run independently by the evaluator. Executor's
systemic find: FOUR block loops needed the R2 patch, not the plan's two — if/then blocks and
\-lambda bodies route through parseRecordLiteral. Eval PASS 96/100 round 1 (fifth consecutive).
PR #342 merged, dev CI green per-workflow. Bar v2 clause 3's centerpiece is done: remaining
clause-3 items are the three R4 diagnostics (#13–15) + prompt work (#16, #23). PARKED: rig A/B
compile_error Δ (GPU; rotation held the rig) — the real success metric, measured post-merge.
Skill fix (2 frictions, its 8+9): sprint-executor completion gate for sprint artifacts.
Next: #11 m-stdlib-regex (NEW-DOC, clause 4).

## STATUS 2026-07-11 (afternoon) — ITERATION 8 COMPLETE: EFFECT SPRINT 1/4 LANDED (closed mode set ENFORCED)

m-effect-mode-validation landed via the full loop headless, round-1 clean (FOURTH consecutive).
The public guide's "typechecker rejects unknown values" claim is now TRUE: frozen `effectSchema`
(Rand + AI) enforced at effect-row elaboration with 3 fix-carrying diagnostics (EFF_UNKNOWN_MODE /
EFF_UNKNOWN_PARAM_KEY / EFF_PARAMS_NOT_SUPPORTED), CI-fixtured in the footgun harness; interim
accuracy note removed; teaching prompt names the codes. Eval PASS 96/100 round 1 with the
evaluator re-producing the acceptance transcript from a self-rebuilt worktree binary. PR #340 →
8faa49de9, dev CI green per-workflow. Unlocks effect sprints 2-4 (replay-contracts, clock-net-fs,
scope-params). EN ROUTE: dev-health issue #341 — 5 runnable examples fail type-check on dev
(pre-existing, proven twice independently; verify-examples is NOT a remote CI gate — same
invisibility class as iteration 3's gofmt miss). Next: #10 m-syntax-ai-forgiving.

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

## STATUS 2026-07-11 (midday) — v1.0 BAR v2 RATIFIED (product-shaped): "the verified AI-orchestration language". Cutoff rule live: gates-v1 ⟺ serves an open clause. Queue re-derived (16 open, 7 NEW-DOC); 17 docs v1_0_0→v1_1_0; strategy review ACTIVE w/ Design Freeze 1+2 ratified. See bar section + log entry 9.

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

## The v1.0 bar — v2, PRODUCT-SHAPED (RATIFIED 2026-07-11, Mark; supersedes the 2026-07-10 hygiene bar)

**The 1.0 claim**: ***the verified AI-orchestration language*** — an AI author gets a
verified-correct program at the lowest cost, and AI orchestration is type-checked. Derived from
[m-fable-strategy-review](planned/m-fable-strategy-review.md) (Design Freeze items 1+2 ratified
by Mark 2026-07-11: cost-per-success is the headline KPI; orchestration is the vertical. Item 3,
trace publication, stays deferred — post-v1).

**The cutoff rule**: a design doc gates v1.0 **only if it serves an open clause below.**
Everything else ships on the normal v0.2x road or is post-v1 — regardless of folder history.
The v1 hygiene bar (2026-07-10) is absorbed: its clauses are 1–2 below, both essentially done.

1. **STABLE** ✅ — the 1.x surface promise (docs/docs/reference/stability.md, iteration 5;
   tier-assignment ratification parked for Mark).
2. **SOUND** — zero P0s ✅ (all four closed, iterations 1–4); residue: m-check-strict-fallbacks,
   m-bytecode-vm-parity-bugs (both ≤2d, queued).
3. **ACCESSIBLE TO THE FLEET TIER** (strategy R1+R4): the finite, documented mid-tier footgun
   list burned down — the 3 parser/type inconsistencies fixed (match-in-HOF-lambda parse,
   polymorphic-arithmetic panic, arity call-style diagnostic), m-syntax-ai-forgiving landed
   (kills the ~32% small-model failure class), and the teaching prompt ≤1,500 lines with a
   rig-A/B showing no pass-rate loss (R3.1 measures the curve first; the deletion pass stays
   gated on replacement diagnostics landing, per m-diagnostic-coverage's deferred section).
   **Gate = this finite work.** The sonnet-class ≥ −5pts outcome is measured and published at
   release, NOT blocking (per Mark: partially vendor-dependent).
4. **ORCHESTRATION FLAGSHIP** (R6 + R7 + effect refinement): the four effect sprints (public
   docs promise); a **verified multi-step AI pipeline** as the flagship example (typed LLM calls
   + budgets + secret-flow + replay) with orchestration benchmarks promoted into the default
   rotation and README/site positioning led by it; **linear-time regex + URL-parse builtins**
   (both verified absent — an orchestration 1.0 without them is a credibility hole).
5. **COST CREDIBILITY** (R3): the dashboard headline KPI flips to **cost-per-verified-success
   vs Python, per tier**, and v1.0 ships with the measured baseline + trajectory. The ≤3×
   zero-shot / ≤1.5× agent targets are the tracked post-1.0 trajectory, NOT release gates.

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
| Mission controller (this loop: triage, pick, judge, retro) | **Opus** (claude-opus-4-8) — TEMP 2026-07-11 (was Fable) | Fable quota relief (Mark). Revert the driver default to `claude-fable-5` when quota clears |
| Design docs (create/review) | **Opus** (was Fable, same TEMP switch) | Runs on the controller model; spec quality still gates downstream |
| Sprint planning | **Opus** (claude-opus-4-8) | Plan quality determined execution success historically |
| Sprint execution | **Opus** — the default, per Mark 2026-07-10 | Sonnet execution was a false economy (needed corrections); also `dev-cycle.md` had silently pinned sonnet |
| Sprint evaluation | **Opus** (was Fable, same TEMP switch) — ⚠ independence caveat below | Judge now shares the executor's model; relies on BEHAVIORAL independence (fresh sub-agent, re-runs tests, cross-history/adversarial checks), not model diversity |

> **⚠ Evaluation-independence caveat (2026-07-11):** while the controller is Opus, Opus evaluates
> Opus-executed work — the generator≠judge *model* diversity is gone. The evaluation's proven
> value has been mostly behavioral (independent test re-runs, cross-history non-vacuity proofs,
> distinct-sample recounts), which survives; but rubber-stamp risk rises. **Mitigation available
> on request:** route the evaluator sub-agent to a distinct-model skeptic (e.g. Sonnet — cheap,
> behavioral role, no Fable spend) via a Gate-3 change. Left OFF for now to keep the switch
> minimal; revisit if any evaluation looks lenient. Full model diversity returns when the
> controller reverts to Fable.
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
*(Queue re-derived 2026-07-11 from bar v2 — clause tag on every open item. NEW-DOC items start
with design-doc-creator; existing-doc items start at reality-check.)*

9. [LANDED 2026-07-11] m-effect-mode-validation (iteration 8: full loop headless, round-1 clean —
   Opus plan (2 discrepancies: bridge carries no params, scope-reduced; EFF_* codes frozen) →
   Opus execute (effectSchema + validateEffectParams at elaboration, 3 fix-carrying diagnostics
   CI-fixtured, guide truth-up: the public closed-set claim is now TRUE, prompt names the codes)
   → Fable eval PASS 96/100 round 1 w/ independent transcript re-production. PR #340 → 8faa49de9,
   dev CI green per-workflow. Unlocks effect sprints 2-4. BONUS: dev-health issue #341 filed
   (5 pre-existing example type-check failures; verify-examples not a CI gate))
10. [LANDED 2026-07-11] m-syntax-ai-forgiving (iteration 9 — the first iteration SPLIT ACROSS
    TWO scheduled runs: run A did reality-check 192a79149 + Opus plan a7bd8257c + Opus execute
    (worktree, M1–M4 64ddd6021) then died pre-evaluation; run B resumed at sprint-evaluator.
    Fable eval PASS 96/100 round 1 (FIFTH consecutive) w/ independent fuzz-gate re-run (zero
    AST diffs over 389 currently-valid corpus files), rebuilt-binary transcripts, non-vacuity
    vs v0.29.2 (PAR017/PAR020 fire on exactly the now-accepted fixtures). R1+R2 BOTH landed —
    R2 systemically patched FOUR block loops (plan's D6 knew two; if/then + \-lambda route via
    parseRecordLiteral). PR #342 → merge, dev CI green per-workflow. DEFERRED: ailang fmt →
    m-ailang-fmt.md stub (D1). PARKED for controller/human: the rig A/B compile_error Δ on
    ;-family benchmarks — the REAL success metric, GPU step, rotation held the rig)
11. [LANDED 2026-07-11] m-stdlib-regex (iteration 11: full loop headless, round-1 clean — Opus
    plan (3 de-risking findings: F1 `_str_slice`/`_str_len` are RUNE-indexed but Go `regexp`
    returns BYTE offsets → span conversion is load-bearing; F2 embed is a glob; F4 changelog
    path) → Opus execute (worktree: 6 `_regex_*` builtins in the MODERN `internal/builtins/`
    RegisterEffectBuiltin system — NOT the doc's outdated `internal/eval` path, **D-ARCH**;
    memoized RE2 cache; `std/regex.ail` + 3 examples incl. the log-orchestration clause-4 use
    case) → Opus eval PASS 97/100 round 1 w/ INDEPENDENT reproduction (backref reject, CJK
    `日本語 world` rune span [4,9) not byte [10,15), findAll). PR #343 → squash-merge 0b0ed7ea0,
    all required checks green. `std/regex` = linear-time (RE2): compile/isMatch/findFirst/findAll/
    replaceAll/split; RE2 subset (no backref/lookaround) → `compile` Err, never panics. Docs:
    LIMITATIONS + stability (Experimental) + CHANGELOG. Design → implemented/v0_30_0)
12. [NEXT] NEW-DOC m-stdlib-url-parse (clause 4; R7 — parse/split into scheme/host/path/query;
    ~1d — the OTHER half of clause-4's builtin mandate, now that regex landed)
13. NEW-DOC m-dx-match-hof (clause 3; R4a — `match` in block-body lambdas in HOF args fails to
    parse; Conflict Surface mandatory, ~2–3d)
14. NEW-DOC m-poly-arith-lambda (clause 3; R4b — polymorphic arithmetic in lambdas panics while
    comparison works; ~2–3d)
15. NEW-DOC m-arity-style-diagnostic (clause 3; R4c — curried vs multi-arg mistake gets a
    call-style-naming diagnostic, not a generic type error; ~1–2d)
16. m-eval-slim-prompt-self-discovery (clauses 3+5; P1, v0_29_0 — R3.1 pass-rate-per-prompt-token
    curve per tier; the data that authorizes the deletion pass)
17. m-effect-replay-contracts (clause 4; P1, ~3d — effect-refinement sprint 2/4)
18. m-effect-clock-net-fs-modes (clause 4; P1, ~3d — effect-refinement sprint 3/4)
19. NEW-DOC m-v1-orchestration-flagship (clause 4; R6 — verified multi-step AI-pipeline flagship
    example, orchestration benchmarks into default rotation, README/site lead; ~2–3d)
20. NEW-DOC m-cost-per-success-kpi (clause 5; R3 — dashboard headline flip + v1.0 measured
    baseline publication; ~1–2d)
21. m-check-strict-fallbacks (clause 2; P1, ~1d)
22. m-bytecode-vm-parity-bugs (clause 2; P1, 1–2d)
23. prompt deletion pass R1.2 (clause 3; GATED — unblocks when items 13–15 diagnostics land AND
    item 16's curve authorizes; lives as m-diagnostic-coverage's deferred section)
24. m-effect-scope-params (clause 4; P1, ~2.5d — sprint 4/4; weakest forcing function — Mark
    re-score candidate at release gate)

**Mission-infrastructure backlog** (improves HOW the loop runs; not a v1.0 gate):
- **m-mission-adaptive-multiprovider-routing** ([planned/v0_30_0](planned/v0_30_0/m-mission-adaptive-multiprovider-routing.md), 2026-07-11) — quota-aware probe-based model selection + cross-provider fallback (OpenAI Codex / Gemini), replacing the hardcoded Monday override; Phase 3 restores evaluation independence cross-family. Try Phase 1 next week (after Fable quota resets). Requested by Mark.

**Not gating** (the other ~80 open docs): ship on the normal v0.2x road or post-v1 per the
clause rule. `planned/v1_0_0/` now contains ONLY gating docs (17 non-gating docs re-bucketed to
v1_1_0 on 2026-07-11); v0_29_0 docs that appear above gate v1 via the queue, not the folder.

**Post-v1**: everything in `planned/v1_1_0/`.

## Ruled out / resolved

- **Sonnet as default executor** — ruled out 2026-07-10 (Mark: corrections needed; false economy).
  Re-entry only via the evidence rule.
- **Scheduling via cron / scheduled-tasks MCP** — ruled out; this rig's substrate is launchd
  (nightly-eval + os-rotation-filler precedents), and the coordinator has no internal timer.

## Done / superseded

*(nothing yet — mission initialized 2026-07-10)*
