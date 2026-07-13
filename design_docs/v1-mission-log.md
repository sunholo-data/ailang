# V1 Mission — iteration log (append-only)

One entry per mission-control iteration, newest LAST (append). Fixed template — keep every
section, write "none" rather than omitting:

```markdown
## N — YYYY-MM-DD — <headline>
**Picked**: <backlog item + why it was top>
**Reality check**: <what git/code verification of the doc's status found>
**Shipped**: <commits/branches/PRs, evaluator result + score, or "parked: reason">
**Routing evidence**: model=<m> task-class=<design|plan|execute|evaluate|mechanical>
  round1-score=<n> rounds=<n> corrections=<n>
**Ruled out**: <hypotheses/approaches refuted this iteration — the anti-re-chase ledger>
**Retro lane**: <skill-fix: file+change | process-fix: change | backlog: new doc | none>
**Next**: <what iteration N+1 should pick up>
```

---

## 0 — 2026-07-10 — Mission initialized (exploration census; no sprint run)

**Picked**: n/a — charter-writing session (interactive, Fable + Mark).

**Reality check**: the census itself. Backlog: 66 docs `planned/v0_29_0/`, 27 `planned/v1_0_0/`,
4 `planned/v1_1_0/`, 8 loose strategy docs. **No written v1.0.0 release criteria exist** — scope
was implicit folder membership. Open P0s (6): m-typeenv-sub-fix, m-named-test-blocks,
m-feedback-triage-gate, m-diagnostic-coverage (v0_29_0); global-collaboration-hub,
m-eu-compliance-effects (v1_0_0). Multi-week strategic weight dominates v1_0_0 (effect-handlers,
effect-refinement, quasiquotes, CSP session types, d4, perf4-bytecode).

**Shipped**: this charter (`v1-mission.md`), this log, the `mission-control` skill, launchd
wiring (`tools/launchd/mission-control.sh` + `dev.ailang.mission-control.plist`, written NOT
armed), dev-cycle.md model pin sonnet→opus.

**Routing evidence**: none (no sprint). Prior data informing the initial policy: Sonnet-executed
sprint needed corrections (Mark, 2026-07-10, "false economy"); `dev-cycle.md` was the only file
in the pipeline with a model directive and it pinned sonnet — the four skills specify none.

**Ruled out**:
- cron / scheduled-tasks MCP as the scheduler (rig substrate is launchd; coordinator daemon has
  no internal timer — external timer is mandatory).
- rig.lock for cloud-only iterations (it is a GPU mutex only — Mark clarified 2026-07-10).
- Trusting design-doc status headers when picking work (M-EVAL-BENCH-UI sat in planned/ for a
  month after all 4 milestones shipped; a triage pass re-bucketed it as still-outstanding because
  the gitignored sprint JSON said in_progress).

**Retro lane**: process-fix — the self-improvement gap itself: sprint retros
(`docs/sprint-retros/`) and evaluator reports (`.ailang/state/evaluations/`) existed but nothing
ever fed back into the skills; RETRO is now a mandatory gate. Existing retros + 9 eval reports
are raw material for the first real retro pass.

**Next**: Iteration 0 proper — Mark ratifies the v1.0 bar draft; re-score the backlog; rewrite
the queue. Then iteration 1 picks the top P0 (m-typeenv-sub-fix, pending re-score).

---

## 1 — 2026-07-10 — Iteration 0: bar ratified, backlog re-scored, queue rewritten

**Picked**: Iteration 0 (queue head). Interactive with Mark; ran via the mission-control skill
(its first invocation).

**Reality check**: paid off immediately — 2 of 4 v0_29_0 P0s were stale:
- m-named-test-blocks: M1 shipped 2026-07-09 (ec4996e45, 7389e84c1 + fixes fd75ce8d4/71d0d43a3).
  Verified LIVE after `make quick-install`: deliberately-failing named test → FAIL line + exit 1
  (pre-rebuild the stale binary showed the old silent-green). Reduced to closeout.
- m-feedback-triage-gate: genuinely open — shipped M-MCP-EDGE-THROTTLE (2d8d5e937) is its stated
  PRECONDITION, not its scope.
- m-typeenv-sub-fix / m-diagnostic-coverage: genuinely open. Both v1_0_0 P0s: never started.

**Shipped**: v1.0 bar RATIFIED (Mark). Scope decisions: effect refinement IN (decompose first) /
handlers OUT→v1.1 (bake-time argument); CSP, quasiquotes, perf4, D4 OUT→v1.1 (+
agent-orchestration, zero-language-learnings by policy); both v1_0_0 P0s downgraded to P1 with
dated notes. 7 docs git-mv'd v1_0_0→v1_1_0. Charter: bar marked ratified, 9-item
required-for-v1 queue written (head: named-test-blocks closeout). Identified missing artifact:
m-v1-stability-promise design doc (queued #5).

**Routing evidence**: model=fable task-class=mission-coordination round1-score=n/a rounds=1
corrections=0 (interactive; Mark redirected zero conclusions — but note 3 scope questions
needed his input by design).

**Ruled out**:
- Effect handlers for v1.0 (Mark: bake-time risk under a fresh stability promise beats Koka
  parity now).
- "All 4 v0_29_0 P0s are open work" — refuted by reality-check (1 of 4 substantially shipped).
- Trusting `$?` after a piped command for exit-code verification (caught own error: `| tail`
  masks the exit code; use direct invocation or PIPESTATUS).

**Retro lane**: none this iteration — two frictions RECORDED (each 1 instance; the ≥2 rule
correctly blocks skill edits until they recur):
1. Skill `!'...'` frontmatter injections rendered as raw text when invoked via the Skill tool —
   preflight had to be run manually. If it recurs in the nightly headless run → skill fix
   (inline the preflight commands as explicit Gate-0 bash instead of frontmatter injection).
2. Stale installed binary nearly falsified the live reality-check (old `ailang` showed
   pre-M1 behavior; the binary's own staleness warning saved it). If it recurs → skill fix
   (Gate 2: `make quick-install` before any live verification).

**Next**: Iteration 1 — m-named-test-blocks closeout (bookkeeping pick, so per standing rule 1
it may also take m-typeenv-sub-fix: design decided, Opus sprint via planner→executor→evaluator,
the first full inner-loop run with routing evidence).

---

## 2 — 2026-07-10 — Iteration 1: first full inner-loop run; 2 items landed; the loop's honesty held

**Picked**: m-named-test-blocks closeout (bookkeeping) + m-typeenv-sub-fix (standing rule 1
allows a second on a bookkeeping pick). Interactive with Mark; mission-control skill invoked.

**Reality check**:
- 1a: all in-repo criteria verified live post-rebuild; duckdb's formerly-silent-skipped shipped
  tests now execute 2/2 via `--package .`. Deontic criterion unverifiable (package absent from
  every local checkout) — deferred honestly, not hand-waved.
- 1b: the "open P0" premise was STALE at a depth Gate 2 missed — I read the 3 `t.Skip`-parked
  tests as "bug still open"; the executor's re-verify directive revealed all 3 PASS un-skipped.
  Evaluator cross-history runs: FAIL at ebd23a67c (2026-03-27, bug documented live), PASS at
  M-TYPE-LIST-SOUND round 3 (66aceed79). The hole was closed ~Jun 8 as a side effect; nobody
  un-skipped the tests.

**Shipped**: 1a closeout commit 246b89b1e. 1b: plan 7f5ca8615 (Opus; reconciled 6 stale
design-doc items); executor branch (Opus, isolated worktree, 2 commits): 3 tests un-skipped +
new TestTypeSafety_CrossModule_ExportedADTRoundtrip + honest CHANGELOG; evaluation 92/100 PASS
(eval_M-TYPEENV-SUB_round_1.json); merge f59421ac8 with attribution corrected at merge (round 4
→ by round 3, e67e24cb1 disproven as closer). Post-merge `make test` failure root-caused to a
STALE `bin/ailang` (v0.26.0, Jun 26 — test helpers prefer it over PATH) — rebuilt, full suite
green, 0 FAILs. Design doc + sprint plan → implemented/v0_29_0 (status: Resolved).

**Routing evidence**:
- model=opus task-class=plan round1-score=n/a rounds=1 corrections=0 (plan approved unchanged;
  independently caught 6 stale doc items — high value add)
- model=opus task-class=execute round1-score=92 rounds=1 corrections=1 (causal attribution wrong:
  claimed round 4 + tvar-collision; proof shows ≤ round 3 and e67e24cb1 predates the live bug.
  Scope discipline exemplary: declined dead 135-LOC shared-infra repair, recorded M1
  passes:false honestly)

**Ruled out**:
- "The typeenv hole is open" — the biggest ruled-out yet: closed since ~2026-06-08 by
  M-TYPE-LIST-SOUND (by round 3). The 3-month-old P0 was a bookkeeping ghost.
- "The post-merge make test failure was caused by the merge" — it was the stale bin/ailang;
  merge diff (test file + changelog) could not affect internal/pkg.
- Executor's attribution of closure to round 4 + M-TVAR-COLLISION-FIX — disproven by
  cross-history test runs (e67e24cb1 is 2026-02-13, bug documented live 2026-03-27).

**Retro lane**: skill-fix (the ≥2 rule satisfied by THREE same-class frictions: 1a stale
installed binary, 1b-eval stale bin/ailang, 1b parked-tests-read-as-open) — mission-control
Gate 2 gained a Verification protocol: rebuild BOTH binaries + confirm --version vs git
describe; un-skip-and-run parked tests before treating a bug as open; PIPESTATUS for piped exit
codes. Systemic fix for the bin/ailang hazard (test helper prefers stale local builds) spun off
as a background task chip.

**Next**: Iteration 2 — m-feedback-triage-gate (P0, genuinely open, 2d): full inner loop again;
apply the new Gate-2 protocol from the start. Also pending for Mark: arm the launchd nightly
after this supervised run's evidence (2 iterations, both clean).

## 3 — 2026-07-10 — Iteration 2: m-feedback-triage-gate landed via full loop (first unsupervised nightly-mode run)

**Picked**: m-feedback-triage-gate (P0, top [NEXT]; public unauthenticated endpoint fans out to
Sonnet with no triage between Firestore and dispatch). Iteration ran headless — no human present.

**Reality check**: genuinely open — `internal/triage/` absent, no gate in the cloud dispatch
path. All 3 preconditions verified shipped: `auto_dispatch` on submit_feedback
(feedback_tool.go), M-PKG-FEEDBACK-LOOP M2 (228d5c0a3/64bd31032), M-MCP-EDGE-THROTTLE
(2d8d5e937). Found a NAMING-COLLISION hazard the design doc predates: M-MSG-TRIAGE-ROUTER
already owns `coordinator.triage`/`Decision`/`TriageConfig` — plan mandated disambiguated names
(package `feedbackgate`, `Verdict`, `coordinator.feedback_gate`). Also found + cleared leftovers
from today's killed 13:03 run: a dead-locked worktree holding branch `sprint/m-feedback-triage-gate`
(0 unique commits) with an uncommitted 205-line scaffold — quarantined to
`~/.ailang/state/quarantine/2026-07-10-killed-run/`, worktree/branch removed. Binaries rebuilt,
both == git describe before any live check (Gate-2 protocol).

**Shipped**: plan commit dfc1bfd25 (Opus; reconciled 6 doc-vs-reality discrepancies: Message
field mapping, NO IP on the wire → cooldown keys From+category+bodyHash, `auto:` is a category
prefix not a bool, native JSON mode in internal/ai, sibling-repo terraform + dashboard UI cut to
follow-ups). Executor (Opus, isolated worktree, 6 milestone commits 46dcef0bf→def5b9042):
`internal/feedbackgate` (rules → sliding cooldown → fail-closed Haiku classifier via injected
ai.Provider → daily budget cap) + opt-in coordinator wiring (off by default, feedback-gate-audit,
dry-run, env kill-switch) + offline flood drill (1000 msgs → 30 dispatched, $0.90 vs $30
baseline). Evaluation (Fable) 93/100 PASS round 1 (eval_M-FEEDBACK-TRIAGE-GATE_round_1.json);
independent verification: full `make test` rc=0/101 ok in worktree, lint 0 issues, drill run
live. Merge 40f1cdc3f; docs → implemented/v0_29_0 (8d234f2fc). CALIBRATED STATUS: gate logic
complete + merged ✓; PRODUCTION protection is rules-only until the Firestore cooldown/budget
adapters ship (executor deviation, recorded) — the P0 is NOT fully closed operationally; queued
m-feedback-gate-cloud-adapter as its completion.

**Routing evidence**:
- model=opus task-class=plan round1-score=n/a rounds=1 corrections=0 (plan approved unchanged;
  independently found 6 doc-vs-reality discrepancies; premise spot-check auto:-prefix verified)
- model=opus task-class=execute round1-score=93 rounds=1 corrections=0 (honest report incl.
  correctly-attributed environmental flake; 1 recorded scope deviation — cloud store adapters
  deferred — judged acceptable-with-follow-up, not a correction)
- model=fable task-class=evaluate rounds=1 (independent re-run of full suite + drill; deviation
  impact analysis produced the follow-up item)

**Ruled out**:
- "M-MSG-TRIAGE-ROUTER overlaps this scope" — no: it is local intake-inbox promotion
  (hold/promote/drop), a different wiring point; only the NAMES collided.
- "The killed run's message-reported blockers (no subscription token / rc=143) are open" — both
  superseded by the keychain-probe guard at HEAD (c8d56d509); this run launching proves it.
- "Post-merge FAILs were caused by the merge" — TestRunCommand_PipedStdoutFlushesPerLine passes
  2/2 isolated (load-sensitive 4s deadline; passed in code-identical worktree full run);
  TestNetHttpPost is the known live-network test. Neither is in sprint-touched packages.

**Retro lane**: process-fix (mission doc): queue insertion of m-feedback-gate-cloud-adapter
directly after the landed item (a PASS whose deviation defers the operational point of a P0 must
queue its completion, not bury it in follow-up notes). No skill edit (single-friction classes
only this iteration; ≥2 rule not met).

**Next**: Iteration 3 — m-feedback-gate-cloud-adapter (~0.5–1d: Firestore CooldownStore +
BudgetStore adapters, classifier provider construction in cloud wiring, enable w/ DRY_RUN=1
first week) to operationally close the P0; then m-diagnostic-coverage (P0, 3–4d). Parked for
human: none blocking — flood-drill-vs-live-env ops task and sibling-repo terraform alert remain
follow-ups on the queue's nice-list.
## 4 — 2026-07-10 — Iteration 3: cloud-adapter P0 completion landed (first round-1 eval FAIL, caught a fail-open bug) + dev un-redded (3 pre-existing CI breaks fixed)

**Picked**: m-feedback-gate-cloud-adapter (top [NEXT]; completes P0 m-feedback-triage-gate
operationally — merged gate was rules-only in production, Cooldown/Classifier deps never
constructed). Headless run, no human present.

**Reality check**: genuinely open — `daemon_tasks_init.go:52-65` wires the gate but leaves both
deps nil (in-code comment even says "attached by the cloud adapter path (follow-up)"). Gate-2
protocol: both binaries rebuilt, == git describe. Duplicate gate passed (top neural 0.35 < 0.45).
Design doc written by Fable (e507287ab); Opus planner re-verified all six load-bearing premises
LIVE and found **zero discrepancies** (plan 053425ae6) — first zero-discrepancy plan of the
mission (contrast: 6 found in iteration 2's doc, which predated the sprint by weeks).

**Shipped**: `internal/storage/firestore` FeedbackGateCooldownStore (sliding 24h window,
hashed doc IDs, saturation cap 64) + FeedbackGateBudgetStore (per-UTC-day counter), pure
table-tested window math (no-emulator convention); `Daemon.SetFeedbackGateDeps` + call-site-
tested attach + three-stage startup log; CLI cloud construction with FAIL-CLOSED nil-provider
path when ANTHROPIC_API_KEY absent; DRY-RUN-first runbook + terraform handoff (TTL policies,
secret) + CHANGELOG. Executor commits ee516e5c5/8201b4205/f7654aae7 + round-2 fix 408375545.
**Evaluation: round 1 FAIL** (numeric 92, policy fail — readAttempts/readCount swallowed
non-NotFound Firestore read errors → degraded read commits a RESET window over stored state:
fail-open in a flood gate; CLAUDE.md CP2 violation, contradicted the doc's "returns errors
honestly"); surgical round-2 fix → **round 2 PASS 97/100**. Merge 842d7d501 → push 6fc5d7ee9.
**Plus the red-dev deliverable** (see below): 9d2e32ac1 (Windows codegen escape fix +
regression test; docs npm peer-dep pin) and 4c22032de (per-package go-test timeout 60s→300s).
Dev CI fully green on 4c22032de (CI, Build+Release, CodeQL, Scorecard; docs deploy green on
a1f66c333). [LANDED].

**Routing evidence**:
- model=fable task-class=design rounds=1 corrections=0 (all 6 premises survived Opus live
  re-verification — the Gate-2 reality-check discipline is paying for itself)
- model=opus task-class=plan round1-score=n/a rounds=1 corrections=0
- model=opus task-class=execute round1-score=92(policy-FAIL) rounds=2 corrections=1 (the
  error-swallow; its own in-code justification was plausible but wrong — evaluator refuted it
  against Firestore commit semantics; round-2 fix exact and clean)
- model=fable task-class=evaluate rounds=2 (independent full-suite + lint both rounds; caught
  the fail-open bug by reading the diff against the cited messaging.go precedent)
- model=fable task-class=mechanical(CI-fix) rounds=1 corrections=0 (strconv.Quote systemic fix
  + audit of all writef sites; npm pin; timeout bump — all verified green remotely)

**Ruled out**:
- "Iteration-3 merge caused the red CI" — no: identical test-windows failure on parent commits
  and on a DOCS-ONLY commit (0f3a5d95d, 12:57Z); npm ERESOLVE from dependabot #314; the reds
  pre-date the merge by ~3h.
- "test-windows red = new code bug from today's PRs" — no: latent codegen bug (contract panic
  literals embed OS paths raw; backslashes = invalid escapes) exposed by an environment/cache
  change, not introduced today. Real bug, fixed properly (strconv.Quote), not masked.
- "TestRunCommand_PipedStdoutFlushesPerLine failures = regression" — re-confirmed known
  load-sensitive flake (passes 2/2 isolated, twice this iteration).
- "Executor rationale 'the transaction commit reports read failures anyway'" — refuted: a
  failed read never enters Firestore's commit validation set; commit can succeed and clobber.
- "cmd/ailang 1m0s timeout = hang" — no: package legitimately runs 35-50s; 60s budget was
  boundary-flaky (green 15:45, red 16:10 on near-identical code).

**Retro lane**: skill-fix (mission-control SKILL.md Gate 1) — TWO recorded frictions, same gap:
(a) session-start OBSERVE read `gh run list --branch dev --limit 6` as green while dev CI had
been red since 12:57Z — the list was flooded by Dependabot-Updates entries; (b) the also-red
Build-and-Release and Docs-Deploy workflows were equally invisible (one showed as "queued").
Fix: Gate 1 now requires per-workflow conclusion checks (CI, Build and Release, Docs deploy),
not a raw limit-6 list. Single-instance frictions logged for the ledger, no action yet:
worktree isolation branches from origin/dev not local dev (executor had to vendor the plan
docs; add/add conflict at merge), SendMessage advertised by Agent tool but unavailable
(round-2 feedback needed a fresh agent), self-inflicted conflict-marker slip (caught by grep
count, amended before push — verify markers=0 BEFORE commit).

**Next**: Iteration 4 — m-diagnostic-coverage (P0, cheapest cost-per-success lever, 3-4d;
may need decomposition check at PICK). Parked for HUMAN (in #329 report): feedback-gate
enablement ops (sibling-repo terraform: TTL policies on expires_at for the two new collections,
ANTHROPIC_API_KEY secret on the coordinator service; then enable with DRY_RUN=1 for week 1) —
the gate defaults OFF until then; Mark may veto the fail-closed no-key posture before enabling.

## 5 — 2026-07-10 — Iteration 4: m-diagnostic-coverage closed for v1 (stale-status found, 4 footgun rows promoted to covered); first PR-route integration around a contended main tree

**Picked**: m-diagnostic-coverage (top [NEXT], P0, "cheapest cost-per-success lever").
Headless run, no human present. Dev CI at OBSERVE: in_progress on 22b3f0c62 (PR #335, test-
coverage-only) — completed green mid-iteration; per-workflow check used per the iteration-3
skill fix.

**Reality check**: the doc's "Planned" status was STALE — M1–M3 (footgun table 14 rows,
CI fixture mechanism, first 3 diagnostics incl. #325 PAR_IMPORT_PLACEMENT + #327 interim
hint) shipped 2026-07-09 (ff58a3259/e59197554), and #327's REAL fix (01fb8676a) shipped in
the same sprint. Fixtures re-verified green on dev before planning. Genuinely open remainder:
5 fixture-promotions, the A/B-gated deletion pass, the haiku causal re-run. Gate-2 protocol
followed; NOTE: rebuild produced `-dirty` binaries because a sibling agent opened a conflicted
merge in the shared main tree mid-iteration (see Retro).

**Shipped**: sprint M-DIAG-FIXTURE-PROMOTION via full loop headless — Opus plan (verified all
premises live; found 2 real discrepancies: %/Fractional row's claimed diagnostic UNREACHABLE
via `ailang check` → stays inventoried; stdlib-hint fixture needs TestMain wiring of
AILANG_STDLIB_PATH + types.ImportSuggester) → Opus execute in isolated worktree (3 commits:
467a398ec fixtures, 793b9177c footguns.md truth-pass, 9106ddfe1 changelog) → Fable evaluation
**PASS 96/100 round 1** (independent test re-run; non-vacuity proven TWICE — executor inverted
stdlib_import_hint, evaluator independently inverted reserved_keyword; both FAIL→restore→PASS).
4 rows promoted to `covered` (PAR_RESERVED_KEYWORD, PAR_HYPHEN_IN_MODULE, PAR017,
stdlib ImportSuggester hint): 7 fixtures now CI-enforced across 6 footgun rows. Plus design-doc
stale-status correction (2bf4ecafe). Integrated via **PR #336** (main tree was commit-unsafe,
see Retro) → merge fe807aac8; dev CI green per-workflow on the merge (CI, Build+Release,
Docs-Deploy). [LANDED].
**Parked** (both recorded in the design doc's dated deferred-step note + #329 report):
(1) prompt-line deletion + rig A/B — only ~35 deletable lines today vs the ≥100 gate; sequencing
decision: widen covered set first, then ONE A/B amortizes a ≥100-line deletion; (2) haiku causal
re-run on legal_obligation_engine — API-billed, impossible under the headless billing guard;
needs a supervised session or explicit human ops.

**Routing evidence**:
- model=fable task-class=triage/reality-check rounds=1 corrections=0 (caught the stale doc
  status before any sprint tokens were spent — second stale-status catch of the mission)
- model=opus task-class=plan rounds=1 corrections=0 (2 genuine discrepancies found by live
  verification; both held up at execution — the fixture would have been silently red without
  the TestMain finding)
- model=opus task-class=execute round1-score=96 rounds=1 corrections=0 (zero deviations; did
  the adversarial non-vacuity check unprompted-quality; footguns.md truth-pass was honest,
  including writing down that its own table had claimed an unreachable diagnostic)
- model=fable task-class=evaluate rounds=1 (independent re-run + independent non-vacuity
  inversion on a different fixture than the executor's)

**Ruled out**:
- "%/Fractional row is fixturable as claimed" — refuted by the planner live: `3.5 % 2.0`
  type-checks clean; the `No instance for Fractional[int] ... intToFloat` message only fires
  via the internal InstanceEnv.Lookup unit path, NOT via `ailang check` on any .ail snippet.
  Fixturing it requires new diagnostic work (future entry), not a fixture.
- "stdlib import hint works out of the box in package tests" — refuted: types.ImportSuggester
  is nil outside cmd/ailang init(), and stdlibindex scans ./std relative to CWD; without both
  TestMain steps the fixture is silently red with a bare "undefined variable".
- "The sibling's open merge blocks integration" — routed around, not through: worktree branch
  + PR #336 with --auto merge integrated cleanly without ever committing in the main tree.
- "#327 interim hint already retired by the real fix" — NOT verified either way: its unit test
  drives inferVar directly so passing proves nothing about reachability; retirement needs the
  original cross-module repro. Left as an explicit follow-up note in footguns.md, not scope.

**Retro lane**: skill-fix (mission-control SKILL.md Gate-2 protocol, new point 4) — TWO
recorded frictions, same gap ("the shared main checkout is mutable mid-iteration"): (a) Gate-2
rebuild went `-dirty` under a sibling agent's conflicted in-progress merge (binaries from a
half-merged tree; caught by the --version≠clean check the protocol already required); (b) a
persisted Bash `cd` into the eval worktree made a later "main-tree" MERGE_HEAD check read the
worktree's .git and report the merge cleared when it wasn't. New rules: pwd/absolute-path
before main-tree checks, git-status-at-moment-of-use, never commit while a sibling's
MERGE_HEAD exists (it would complete THEIR merge — integrate via worktree branch + PR),
-dirty rebuild → rebuild in the isolated worktree. Single-instance frictions logged, no
action: PR-CI + auto-merge added ~25 min wall-clock vs a local merge (acceptable; correctness
won); planner flagged M3-changelog as merge-blocked but the worktree copy was clean (plan
conservatism, harmless).

**Next**: Iteration 5 — m-v1-stability-promise (queue 6, now [NEXT]): design doc via
design-doc-creator for the 1.x stable-surface promise + LIMITATIONS.md accuracy pass. Note
for a FUTURE iteration (not 5): first re-check `%`-row + m-record-update-local-resolution doc
status before any new diagnostic work — this doc family has now been stale twice. Parked for
HUMAN (repeat, still open from iteration 3): feedback-gate enablement ops (terraform TTL +
ANTHROPIC_API_KEY secret + DRY_RUN week 1); NEW: haiku causal re-run needs a supervised
(API-billed) session — cheap, one benchmark cell, high thesis value.

## 6 — 2026-07-10/11 — Iteration 5: m-v1-stability-promise landed — the v1.0 bar's stability clause closed (design→plan→execute→evaluate all in one headless pass)

**Picked**: m-v1-stability-promise (queue #6, top [NEXT] per log entry 5). Design-doc-creation
item — Fable lane. Headless run, no human present. Dev CI at OBSERVE: all three workflows green
per-workflow (CI + Build-and-Release @ ece12eab9, Docs-Deploy @ a2ede0fee). Inbox: only our own
iteration-4 report (acked).

**Reality check**: no stability/promise doc exists anywhere (find + git-log-grep confirmed
genuinely NEW — first queue item of the mission that wasn't stale-status). Supporting premises
verified live at v0.28.0-139-gece12eab9 (both binaries rebuilt, clean, versions match): root
docs/LIMITATIONS.md frozen at v0.13.0 (99f76ec7a) AND actively wrong — its two flagship "still
broken" entries are FIXED (poly-arith lambda → 5.85; match-in-HOF block lambda → [zero, ok, ok],
its example even uses the retired `match…with |` syntax); 4 website pages carry v0.3–v0.8-era
promises for shipped features. Bonus finding at session start: stale M-SECRET-EFFECT sprint JSON
(v0.26.0, shipped) still bannered as ACTIVE — flipped to completed.

**Shipped**: full loop, all four stages, round-1 pass —
- Fable design doc d53d7d800 (planned/v1_0_0, HARD-GATE live transcripts for every language claim).
- Opus plan 379990ad5 — found 2 REAL premise discrepancies: stdlib is 42 modules not the doc's
  39; LIMITATIONS.md is DOUBLE-MAINTAINED and diverged (website copy docs/docs/reference/
  limitations.md "verified v0.14.2" vs root v0.13) — plan re-scoped M2 to fix BOTH.
- Opus execute in isolated worktree, 4 commits (d28da7ba9 M1 stability page; 12473d76c M2 both
  LIMITATIONS files, every entry live-verified w/ committed transcripts; 707749344 M3 4 website
  promises + CHANGELOG; a2a067ae3 std/secret index drift fix). Docs-only diff: 0 Go files.
- Fable evaluation **PASS 96/100 round 1** (eval_M-V1-STABILITY-PROMISE_round_1.json):
  independent re-verification on DISTINCT samples — programmatic tier-coverage (every std module
  exactly once), duplicate-record re-scope confirmed (interpreter correct → Go-codegen-only),
  `?`→PAR_NO_PREFIX_PARSE, Y-combinator→occurs-check, sweep grep re-run (roadmap page only),
  Recently-Resolved contains both pre-verified fixes, elaborate/expr_control.go anchor preserved.
- Integrated via PR #337 → merge fcccd7208; all PR checks green; dev CI per-workflow on the merge:
  see status stamp ([LANDED] only on green).
**Parked for HUMAN (release gate, not merge blocker)**: tier-ASSIGNMENT ratification before the
v1.0.0 tag — ⚠ proposed rows: std/net, std/crypto, std/jwt, std/xml, std/zip, std/process, CLI
watch/serve-api (all held conservatively Experimental — the cheap direction to be wrong in).
Still open from prior iterations: feedback-gate enablement ops; haiku causal re-run (API-billed).

**Routing evidence**:
- model=fable task-class=design rounds=1 corrections=2 (both factual slips — module count,
  missed diverged LIMITATIONS copy — caught cheaply by the Opus planner's premise verification;
  all flagship claims held. The verify-premises-at-plan-time protocol is earning its cost.)
- model=opus task-class=plan rounds=1 corrections=0 (2 genuine discrepancies found live; also
  caught the auto-linker false-positiving GH issues #6/#7 into the sprint JSON)
- model=opus task-class=execute round1-score=96 rounds=1 corrections=0 (deviations all documented
  + justified: root-LIMITATIONS→pointer resolved a Deferred Decision; anchor preservation for the
  live CLI error link was unprompted-quality; honest flake reporting with isolation re-run)
- model=fable task-class=evaluate rounds=1 (independent full make test — caught the same flake
  independently, proving the executor's "pre-existing" claim rather than trusting it)

**Ruled out**:
- "The full-suite failure is a sprint regression" — refuted twice independently: zero Go files
  in the diff, TestRunCommand_PipedStdoutFlushesPerLine passes 3/3 isolated (-count=3), history
  shows it's the M-PERF6B stdout-buffer timing area. It's a load-dependent flake → issue #338.
- "docs/LIMITATIONS.md is the single source to fix" — refuted by the planner: the website copy
  is the PUBLIC one (sidebar-linked) and had diverged; fixing only the root file would have
  left the public page wrong. Deferred-Decision #4 ("whether mirrored") was moot — it already was.
- "The design doc's stdlib count (39) was verifiable-enough" — it came from a truncated ls
  head; enumerate programmatically (`ls std/*.ail | wc`), never from a scrolled listing.

**Retro lane**: backlog — GH issue #338 (deflake TestRunCommand_PipedStdoutFlushesPerLine; TWO
observations this iteration: executor + evaluator full-suite runs, same failure, both proved
non-regression by isolation). NO skill edit (nothing reached ≥2 same-gap frictions: design-doc
factual slips ×1-class-but-caught, eval-report script zeroed score_breakdown needing manual
fill ×1, stale sprint banner ×1 — all single-instance, logged here for future accumulation).

**Next**: Iteration 6 — queue #7: m-effect-refinement DECOMPOSITION (multi-week strategic →
the iteration's deliverable is decomposition into ≤3–4d sprint docs, NOT execution — standing
rule for strategic items). Note: the ratified bar's remaining open clauses after this landing
are the eval-bar (m-eval-frontier-tier, queue #8) + effect-refinement decision execution. Carry
forward for a FUTURE iteration: re-check %-row + m-record-update-local-resolution doc status
before new diagnostic work (from iteration 4). Parked-for-human items above.


## 7 — 2026-07-11 — Iteration 6: m-effect-refinement DECOMPOSED — last strategic v1.0 item is now sprint-sized queue items

**Picked**: m-effect-refinement decomposition (queue #7, top [NEXT] per log entry 6).
Multi-week strategic item → deliverable is DECOMPOSITION into ≤3–4d sprint docs, not execution
(standing rule 4). Fable lane throughout (design docs). Headless run — the 03:24 scheduled run
died on a transient API socket error (rc=1, message a1aa4026); this 05:35 run is its retry.
Dev CI at OBSERVE: all three workflows green per-workflow (CI + Build-and-Release @ 6c25f45e9,
Docs-Deploy @ fcccd7208). Inbox: the rc=1 crash report + an eval-suite start notice (49
benchmarks, agent mode — GPU busy; this iteration never touches the rig, GPU rule honored by
staying off ollama entirely). Both acked.

**Reality check** (both binaries rebuilt, v0.28.0-148-g6c25f45e9, versions match git describe):
the parent doc's "Planned / ~90h" header was stale by more than a third — fourth stale-status
catch of the mission:
- Phases 1+2 (parser/AST + row algebra/invariant unification/default-mode table) shipped
  v0.15.0 as M-EFFECT-REFINEMENT-PHASE1 (d1abd8ceb…de7fe9a7d), full outcome report in doc.
- The AI port shipped v0.15.0 as M-AI-EFFECT-MODES (bare !{AI}→mode=fixed, routeable at type
  level); its loose ends already cataloged in m-ai-effect-modes-followups (P2).
- Phase 7 (CryptoRand alias): scope-reduced away in v0.15.0 because **M-CRYPTORAND never
  landed at all** — no !{CryptoRand} token, no std/crypto/rand, no CSPRNG builtins (grep
  sweep empty); yet its doc sits in implemented/v0_15_0 with "Status: Planned", swept there by
  the 48-doc bulk relocation 645467e13. Header corrected to Superseded this iteration.
- Live checks: !{Clock[mode=pinned]}, !{Rand[mode=banana]}, scope=identity ALL pass
  `ailang check` — the grammar is generic and **no mode-value validation exists anywhere**
  (grep internal/types + parser: none). The public guide explicitly claims "the typechecker
  rejects unknown values" — FALSE. New sprint-sized item discovered.
- Runtime: internal/effects/* has zero access to effect params (grep Params: empty);
  builtins/rand.go = one global math/rand source for all modes. Clock ALREADY has wall/pinned
  runtime behavior via AILANG_SEED; FS has sandbox machinery — both invisible in types.

**Shipped** (docs-only, 0 Go files):
- 4 sprint docs in planned/v1_0_0, all premises live-verified with transcripts, each with
  Conflict Surface + Design Freeze + explicit boundaries against m-ai-effect-modes-followups:
  1. m-effect-mode-validation (~1d, P1) — enforce the closed mode set; makes the guide's claim true.
  2. m-effect-replay-contracts (~3d, P1) — registry + mode-aware Rand dispatch (seeded=
     deterministic, crypto=CSPRNG — supersedes M-CRYPTORAND's runtime intent).
  3. m-effect-clock-net-fs-modes (~3d, P1) — surface existing runtime switches at type level;
     Net scoped honestly (recorded may be label-only pending planner reality-check).
  4. m-effect-scope-params (~2.5d, P1) — capability narrowing; flagged as release-gate
     re-score candidate (weakest v1.0 forcing function; no public doc promises scope semantics).
- Parent doc → umbrella with repo-verified phase-status table; Phase 6 (M-ENTROPY) routed OUT
  (ships with M-ENTROPY itself; not v1.0-required).
- Public-guide accuracy note (interim truth-up of the false closed-set claim).
- m-cryptorand.md header corrected (Superseded, never-implemented, relocation error named).
- Queue renumbered: #8 eval-frontier-tier [NEXT]; effect sprints at #9/#12/#13/#14 by
  impact-per-day; CHANGELOG entry.
- Dedup gate run per doc (create-script dual search): no match ≥ thresholds; closest was the
  historic v0_3 M-R6 clock/net doc (0.52, distinct scope) and m-process-modes v1.1 (consumer,
  referenced).

**Routing evidence**:
- model=fable task-class=design(decomposition ×4 docs) rounds=1 corrections=0-so-far (premise
  verification done at design time this iteration — every language claim carries a live
  transcript; the Opus planner premise-check remains the independent gate when each sprint runs)
- (no plan/execute/evaluate rows — decomposition iteration, no sprint executed)

**Ruled out**:
- "The parent doc's ~90h is the remaining work" — refuted: ~1/3 shipped v0.15.0 under a stale
  Planned header; remaining ~64h.
- "M-CRYPTORAND shipped in v0.13/v0.15 (as its implemented/ location implies)" — refuted by
  grep sweep + git log: commissioned 5f18e350e, never implemented, mis-swept by 645467e13.
- "The closed mode set is compiler-enforced (as the public guide claims)" — refuted live:
  Rand[mode=banana] passes check at v0.28.0-148. Transcript in sprint-1 doc.
- "Phase 6 (M-ENTROPY integration) must be decomposed for v1.0" — no: M-ENTROPY itself is not
  v1.0-required (not in the ratified bar's queue); the integration ships with M-ENTROPY.

**Retro lane**: none of the single-instance frictions reached the ≥2 rule this iteration
(bulk-relocation mis-sweep ×1 — but note iteration 4+5+6 have now each found a stale/mis-stated
doc status: the Gate-2 reality-check protocol is WORKING as designed, not failing; scheduled-run
API socket crash ×1 — driver already reports + retries by schedule, no fix needed). Backlog
lane: the 4 sprint docs ARE the backlog output. No skill edit.

**Next**: Iteration 7 — queue #8 m-eval-frontier-tier (P1, 2.5–3.5d, eval-bar clause) [NEXT].
It may need GPU/model access — apply the two-tier GPU rule at routing time; an eval-suite run
was in flight this morning (fa586599), so check rig state at pick time. Carry forward: %-row +
m-record-update-local-resolution doc-status re-check (from iteration 4); parked-for-human items
unchanged (tier ratification at release gate; feedback-gate ops; haiku causal re-run; NEW:
scope-params release-gate re-score flag for Mark).

---

## 8 — 2026-07-11 — Iteration 7: m-eval-frontier-tier LANDED — the eval-bar clause's machinery shipped (full loop headless, round-1 clean)

**Picked**: m-eval-frontier-tier (queue #8, top [NEXT] per log entry 7). P1, eval-bar clause
("agent-mode suite discriminating, not saturated"). Scheduled headless run. Inbox at OBSERVE:
nightly eval 69/98 with 6 non-regression failures (flaky/known-gap, gap-finder candidates — did
NOT outrank the queue) + own iteration-6 report. Dev CI green per-workflow ×3 @ 45bbbf8f9.
GPU routing question answered explicitly at Gate 3: NO step touches the GPU (demotion audit =
banked data only; frontier validation runs = API-billed → parked).

**Reality check** (both binaries rebuilt, v0.28.0-149-g45bbbf8f9, versions match git describe):
the doc's 2026-07-08 authoring update is ACCURATE (all 8 benchmarks exist as tier: stretch;
frontier absent from ValidTiers) — first pick of the mission whose status header was
substantially true. Genuinely open: tier machinery, re-tier, ELO wiring, decision_block_capture
(still exact-matches the free-text CHOICE sentence), demotion (all 12+3 still on old tiers).
Both dependencies confirmed shipped v0.26.0 (v0.25.0 baseline re-graded, 44 recoveries;
ratings.go exists). ZERO banked results for the 8 new benchmarks (nightly = smoke+core) →
frontier-failure validation needs API-billed runs, parked for human. Premises stamped into the
doc as a dated reality-check block (c826c8d8a) BEFORE handing to the planner.

**Shipped** (full loop, round-1 clean at every stage):
- Opus plan (9 premise discrepancies found live, incl.: sibling enum test the doc missed;
  stretch tier-count assertions re-centered 30→22; ELO "high provisional rating" is aspirational
  — flat 1500 for all unseen benchmarks; CURATION.md had NO 4-dimension demote rule, only the
  design doc did; "agent data sparse" REFUTED — baselines/v0.25.0/agent/ has 444 files / 6
  models covering all 15 candidates).
- Opus execute in isolated worktree, 5 milestones / 6 commits: frontier in ValidTiers + full
  plumbing; 8 benchmarks re-tiered with PARKED-validation provenance comments; ELO flat-seed
  verified + locked with 2 regression tests; decision_block_capture fixed via PRIMARY path
  (new `grading: prefix_line` structural grader + GradeStdout centralizing 6 call sites);
  demotion audit from banked data → 7 core→stretch DEMOTED, 7 KEPT (conservative 4-dim rule,
  now codified in CURATION.md §5 both directions incl. frontier→stretch demote-back), 1
  keep-for-coverage. Final distribution: smoke 23 / core 19 / stretch 29 / frontier 8 / vision 9.
- Fable eval round 1: **PASS 96/100** (tests 20/20 — rc=0 re-run independently in worktree;
  lint 10/10; AC 30/30 — 27/27 verified; code quality 12/15; docs 15/15; fidelity 9/10).
  Independent distinct-sample verification: demotion audit re-computed from raw banked JSONs
  for 5 benchmarks × 4 dims (graph_bfs, cli_args, merge_sort, float_eq, api_call_json) — all
  matched the executor's table exactly.
- Integration: PR #339 → merge 0515578ae (auto-merge after checks; direct merge blocked by
  branch policy). Design doc + demotion-audit doc → implemented/v0_29_0 (1e2eebe73). Dev CI
  green on the merge (Gate 3b verified per-workflow). Worktree removed post-merge.

**Routing evidence**:
- model=opus task-class=plan round1-quality=high (9 live-verified discrepancies, 0 controller
  corrections) rounds=1
- model=opus task-class=execute round1-score=96 rounds=1 corrections=0 (one honest deviation
  recorded: baseline data read from main checkout — gitignored, absent in worktree; one miss:
  sprint JSON bookkeeping skipped, cost -3)
- model=fable task-class=design(reality-check note)+evaluate rounds=1

**Ruled out**:
- "Agent-mode data is too sparse for the 4-dim demotion audit" — REFUTED (my own Gate-2 claim,
  corrected by the planner): baselines/v0.25.0/agent/ covers all 15 candidates × 6 models ×
  both languages. Only the top-level eval_results/agent/ scratch dir is sparse.
- "The doc's 12-benchmark demotion list is correct" — refuted: only 7 meet the 4-dim rule;
  cli_args is a strong agent discriminator (agent-Python 0/6, agent-AILANG 4/6) and stays core.
- "ELO gives new frontier benchmarks a high provisional rating" (doc's integration section) —
  refuted: flat DefaultInitialRating=1500 regardless of tier; difficulty is emergent. Now
  documented in ratings.go + guarded by TestFitFromTrials_UnseenBenchmarkEntersFlat/TierAgnostic.
- "TestRunCommand_PipedStdoutFlushesPerLine failure = sprint regression" — refuted again (3rd
  time): known flake #338, passes standalone ×2 at 1.48s/1.5s window; fails only under
  full-suite parallel load.

**Retro lane**: none (no ≥2 same-class frictions). Single instances recorded for the watch
list: (a) executor skipped sprint-JSON bookkeeping (status/passes/completed) despite SKILL.md
§272-273/789 requiring it — FIRST occurrence (M-V1-STABILITY-PROMISE and M-TYPEENV-SUB JSONs
were updated correctly); if it recurs, the fix is a pre-push hard gate in sprint-executor.
(b) flake #338 hit again — backlog item already exists, priority unchanged (proven
non-regression 3×).

**Next**: Iteration 8 — queue #9 m-effect-mode-validation (P1, ~1d, effect-refinement sprint
1/4; prerequisite for the other three effect sprints; makes the public guide's closed-set
claim true). Carry forward: %-row + m-record-update-local-resolution doc-status re-check
(iteration 4). PARKED for human (cumulative): tier-assignment ratification (release gate);
feedback-gate production ops; haiku causal re-run; scope-params release-gate re-score; NEW —
frontier-failure validation of the 8 frontier benchmarks (API-billed frontier runs; each must
fail ≥1 frontier model or demote back to stretch per CURATION.md §5) + authoring the 4
remaining sketched benchmarks (benchmark-manager follow-up).

---

## 9 — 2026-07-11 — v1.0 SHAPE ratified: product-thesis bar v2 replaces the hygiene bar

**Picked**: n/a — interactive shape session (Fable + Mark). Mark's challenge: "these tasks don't
seem very critical to v1; 80+ open docs; I want a good cutoff" — correct: the 2026-07-10 bar was
hygiene (safe-to-stamp), not product (what 1.0 IS).

**Reality check**: fresh census 96 open docs (55 v0_29_0 / 23 v1_0_0 / 10 v1_1_0 / 8 loose).
m-fable-strategy-review (2026-07-08) held the unused product thesis: frontier models already
prefer AILANG (97% vs 91% Fable); the deficits are COST (10×/2.8× per success) and MID-TIER
accessibility (−16pts sonnet-class); the moat is verified AI-orchestration (no competitor per
THREE-CAMPS). Its Design Freeze items had sat unratified since 2026-07-08.

**Shipped**: bar v2 in the charter — the 1.0 claim is "the verified AI-orchestration language";
5 clauses (stable ✅ / sound ~✅ / fleet-tier accessible / orchestration flagship / cost
credibility); cutoff rule = "gates v1.0 only if it serves an open clause". Mark ratified 4 calls:
orchestration-led positioning; KPI flip + measured baseline gates (targets track, don't gate);
footguns gate + mid-tier metric tracked; regex + URL-parse both gating. Queue re-derived: 16 open
items incl. 7 NEW-DOCs (R4×3, regex, url-parse, orchestration-flagship, cost-KPI). 17 non-gating
docs re-bucketed v1_0_0→v1_1_0 — the folder now equals the gate. Strategy doc → ACTIVE with
Design Freeze 1+2 ticked, 3 deferred post-v1.

**Routing evidence**: model=fable task-class=mission-coordination round1-score=n/a rounds=1
corrections=1 (Mark's challenge WAS the correction: the loop optimized a queue whose shape was
folder history, not thesis — for ~5 iterations. Caught because a human read the output stream.)

**Ruled out**:
- "The iteration-0 bar was sufficient" — it gated hygiene, not product; the loop landed 8 items
  before the gap surfaced. Lesson: a bar needs a THESIS clause, not just safety clauses.
- Hard-gating v1.0 on outcome metrics (≤1.5× cost, −5pts mid-tier) — Mark: measured+published
  gates, outcomes track (vendor-dependent).

**Retro lane**: process-fix (this bar v2 + cutoff rule). Candidate skill friction RECORDED
(1 instance): mission-control has no "challenge the bar itself" step — OBSERVE reads the queue
as given; nothing forces periodic re-derivation from the thesis. If a second
drifting-queue instance appears → skill fix (add a bar-review trigger, e.g. every N iterations
or on Mark's challenge).

**Next**: loop continues at [NEXT] m-effect-mode-validation, then the re-derived queue. First
NEW-DOC items exercise design-doc-creator headless for the first time.

---

## 10 — 2026-07-11 — Iteration 8: m-effect-mode-validation LANDED — the guide's closed-mode-set claim is now TRUE (effect-refinement 1/4; fourth consecutive round-1-clean full loop)

**Picked**: m-effect-mode-validation (queue #9, top [NEXT] per log entries 8/9). P1, bar clause 4
(orchestration flagship — effect sprints), ~1d, prerequisite for the other three effect sprints.
Scheduled headless run. Inbox at OBSERVE: eval-suite 105/108 (97.2% — above the 69/98 precedent,
non-regression, did not outrank) + v0.29.0 release notice (informational). Dev CI: in-progress at
OBSERVE on 8dad7e80b, confirmed green per-workflow before integration. GPU routing question
answered explicitly at Gate 3: NO step touches the GPU (compiler validation + make test only).

**Reality check** (both binaries rebuilt, v0.29.0-2-g8dad7e80b, versions match git describe):
first pick of the mission that was BOTH genuinely open AND accurately documented — the doc was
authored by iteration 6's decomposition with live-verified premises and they all still held:
`Clock[mode=banana]` / `Rand[mode=banana]` / `Rand[flavor=hot]` all passed `ailang check` rc=0
at HEAD (live transcript), no validation code or commits since authoring. Premises re-stamped
into the doc (b91338dab) before handing to the planner.

**Shipped** (full loop, round-1 clean at every stage — fourth consecutive):
- Opus plan 689001311 (2 premise discrepancies found live: D1 the `stringSliceToEffectRow`
  bridge carries NO params — doc over-scoped it; validation placed at the true chokepoint
  `ElaborateEffectRowWithBudgets`, bridge covered by a nil-params regression test instead.
  D2 `validate_effects.go` has no error-code convention to follow — `EFF_*` codes chosen from
  the doc's own Examples. Also verified the doc's GUESSED AI schema correct against the
  M-AI-EFFECT-MODES outcome report before freezing).
- Opus execute in isolated worktree, 4 milestones: frozen `effectSchema` (Rand: mode∈{os,seeded,
  crypto}; AI: mode∈{fixed,routeable,replay-only}, scope∈{byok}) + `validateEffectParams` wired
  into elaboration with 3 fix-carrying diagnostics (EFF_UNKNOWN_MODE / EFF_UNKNOWN_PARAM_KEY /
  EFF_PARAMS_NOT_SUPPORTED, code embedded verbatim, deterministic ordering); legal-matrix +
  error-shape + bridge-regression tests; 3 footgun CI fixtures; guide truth-up (interim accuracy
  note REMOVED — the public claim is now true), teaching prompt v0.16.2 names the codes (hash
  recomputed + embedded copy synced — an unplanned-but-required step the executor caught),
  CHANGELOG pre-1.0 narrowing entry.
- Fable eval round 1: **PASS 96/100** (tests 20/20, lint 10/10, AC 29/30, quality 13/15, docs
  14/15, fidelity 10/10, regression-surface +10). Independent verification distinct from
  executor: evaluator rebuilt the worktree binary and re-produced the acceptance transcript from
  scratch; prompt load hash-verified end-to-end; both full-suite reds proven non-regression
  (flake #338 standalone-pass ×4; TestNetHttpPost fails IDENTICALLY on pre-sprint dev —
  httpbin.org outage); executor's "5 verify-examples reds pre-existing" claim independently
  re-verified with the PRE-sprint binary.
- Integration: PR #340 → merge 8faa49de9 (auto-merge, MERGE method matching #339 precedent —
  first attempt armed SQUASH, caught and switched). Design doc + sprint plan →
  implemented/v0_30_0 + guide link fix + sprint-JSON committed (bee845466). Dev CI green on the
  merge per-workflow (Gate 3b verified). Worktree removed post-merge.

**Routing evidence**:
- model=opus task-class=plan round1-quality=high (2 live-verified discrepancies incl. a scope
  REDUCTION the doc missed; 0 controller corrections) rounds=1
- model=opus task-class=execute round1-score=96 rounds=1 corrections=0 (sprint-JSON bookkeeping
  DONE this time — iteration-7 watch item did not recur; two artifact nits: plan checkboxes
  unticked, sprint JSON left uncommitted)
- model=fable task-class=evaluate rounds=1 (one self-caught evaluator error: ran the 5 failing
  examples at wrong paths first — bogus rc=1 "confirmation" from missing files; re-anchored via
  pwd + absolute paths per Gate-2 rule 4a and redid the check properly)

**Ruled out**:
- "The 5 verify-examples failures are sprint-caused" — REFUTED twice independently (executor:
  identical at plan commit via throwaway baseline binary; evaluator: identical with pre-sprint
  main binary). They are a dev-health issue: type-unification failures on examples untouched
  since v0.13.0, invisible because verify-examples is NOT a remote CI gate → issue #341.
- "TestNetHttpPost failure = sprint regression" — refuted: fails identically on pre-sprint dev;
  httpbin.org returning an HTML error page (live-network test inside make test; noted in #341).
- "Validation must also cover the stringSliceToEffectRow bridge" (design doc's risk table) —
  refuted by the planner: that bridge builds rows from effect NAMES only, params always nil;
  a nil-params regression test suffices.

**Retro lane**: backlog — issue #341 (5 runnable examples fail type-check on dev, pre-existing,
+ verify-examples-not-in-CI gap + TestNetHttpPost network flake). No skill edit (no ≥2 same-class
frictions this iteration; single instances on the watch list: (a) planner asserted "verify-examples
expected green pre/post" without running it — an unverified-green claim in a plan; if a second
unverified-green plan claim appears, the fix is a sprint-planner gate "any 'expected green'
assertion must carry a live rc=0 transcript"; (b) evaluator wrong-path dead end, self-caught,
already covered by Gate-2 rule 4a).

**Next**: Iteration 9 — queue #10 m-syntax-ai-forgiving (clause 3; P1, ~3d, v0_29_0 — kills the
~32% small-model failure class). NOTE for the picker: 3d is at the sprint-size ceiling; verify
the doc's premises against v0.29.0 reality first (it predates the frontier-tier re-grade).
Carry forward: %-row + m-record-update-local-resolution doc-status re-check (iteration 4).
PARKED for human (cumulative): tier-assignment ratification (release gate); feedback-gate
production ops; haiku causal re-run; scope-params release-gate re-score; frontier-failure
validation of the 8 frontier benchmarks + 4 remaining sketches; NEW — issue #341 triage call
(bisect first; fix vs update-examples vs promote verify-examples into CI).

## 11 — 2026-07-11 — Iteration 9: m-syntax-ai-forgiving LANDED — the ~32% small-model parse-failure class is dead (fifth consecutive round-1-clean; first iteration split across two scheduled runs)

**Picked**: m-syntax-ai-forgiving (queue #10, top [NEXT] per log entry 10). P1, bar v2 clause 3
(fleet-tier accessibility), ~3d ceiling. THE ITERATION SPANNED TWO SCHEDULED RUNS: run A
(afternoon) did reality-check 192a79149 + Opus plan a7bd8257c + Opus execute (worktree
agent-af44352e3fb7d4708, M1–M4, 64ddd6021 at 14:54) and died before evaluation; run B (this
run, 16:56) found the mid-flight state via commits + worktree census — no eval file, no PR, no
log entry, no live executor process — and resumed at sprint-evaluator. Inbox at OBSERVE: 6
eval-suite status broadcasts (informational, 78–99% partials in line with baselines — did not
outrank). Dev CI green per-workflow @ c4dd9aa1f at OBSERVE. GPU routing question: evaluation +
integration touch NO GPU; the sprint's rig A/B is the deferred GPU step → PARKED (rotation
held the rig — 16:31 suite driving 3 local models).

**Reality check** (run A, stamped 192a79149): premises HOLD live at v0.29.2-2-g07aa1062f
(PAR017/PAR020 transcripts at HEAD); DISCREPANCY found: `ailang fmt` DOES NOT EXIST — Phase-3
canonicalization re-scoped (plan option b: golden parse fixtures + doc guidance in-sprint,
formatter split to m-ailang-fmt.md stub). Run B re-verified the execution state independently
rather than re-doing the reality check (worktree clean at M4, plan commit = merge-base).

**Shipped** (full loop; plan+execute run A, evaluate+integrate run B):
- Opus plan a7bd8257c: 9 discrepancies verified live (fmt non-existence D1; changelog target D2;
  fuzz-corpus glob D3; `=`-branch needs its OWN loop D4; peekStartsBlockStatement is type-only
  D5; TWO block loops D6; TWO PAR020 guards D7; D8 dialect-card wording rule; D9 example home).
- Opus execute 4 milestones: R1 `parseEquationBody` + `peekIsDeclBoundary` (boundary set safely
  extended with extern/@/pure beyond the plan's list); R2 via shared conservative helper
  `peekStartsNewlineBlockStatement` (line-check AND statement-starter) patched **FOUR** block
  loops — executor discovered if/then blocks and \-lambda bodies route through
  parseRecordLiteral's block paths, NOT parseBlockOrExpression as D6 believed (the plan's own
  if/then fixture caught it); reusable corpus AST-diff fuzz harness (cmd/astdump spew dumper,
  old parser rebuilt from pinned base a7bd8257c in a temp git worktree, env-gated
  AILANG_CORPUS_FUZZ=1); 314-line Conflict-Surface test file; PAR017 footgun fixture honestly
  migrated (old source became VALID → now a genuinely-misplaced `;;`); dialect-traps trap #2
  rewritten per D8 both-landed branch w/ embedded copy synced; CHANGELOG R1+R2, LIMITATIONS,
  example gated, m-ailang-fmt.md stub.
- Fable eval round 1: **PASS 96/100** (tests 20/20 — sole make-test red = known flake #338,
  4/4 standalone; lint 10/10; AC 30/30; quality 12/15 — sprint-artifact gap, see retro; docs
  14/15; fidelity 10/10; regression-surface +10). Independent verification: evaluator RE-RAN
  the corpus fuzz gate himself (400 files: 389 currently-valid ALL byte-identical, 1
  newly-accepted, 10 still-invalid — zero diffs), rebuilt the binary from worktree sources and
  re-produced all acceptance transcripts (R1/R2 accept; same-line-no-`;` still PAR020;
  comma-less multi-line record errors AS a record, never statement-split), proved non-vacuity
  (v0.29.2 emits PAR017/PAR020 on exactly the fixtures the sprint binary accepts), and
  re-verified verify-examples 183/5/5 with the 5 fails exactly = issue #341's pre-existing set.
- Integration: PR #342 → merge 224404391 (auto-merge, MERGE method per #339/#340 precedent).
  Bookkeeping: docs → implemented/v0_30_0, links fixed (roadmap/changelog/fmt-stub), sprint
  JSON reconstructed + eval report committed. Dev CI on the merge: per-workflow watch (Gate 3b).

**Routing evidence**:
- model=opus task-class=plan round1-quality=high (9 live discrepancies incl. a scope re-route
  the doc itself flagged as needing planner decision) rounds=1 corrections=0
- model=opus task-class=execute round1-score=96 rounds=1 corrections=0 — QUALITY HIGHLIGHT:
  found a systemic gap the plan missed (4 loops not 2) instead of implementing the plan
  literally; artifact gap: sprint JSON never updated + plan checkboxes unticked (2nd
  consecutive iteration → skill fix below)
- model=fable task-class=evaluate rounds=1 (two self-caught evaluator dead ends: (a) `make test
  | tail` masked rc — Gate-2 rule 3's exact pipes-lie case, re-run with captured rc; (b) first
  verify-examples tail read the SKIPPED list as failures — full-log grep corrected it before
  any wrong conclusion shipped)

**Ruled out**:
- "The 5 verify-examples failures are sprint-caused" — refuted: exactly the issue #341 set
  (effectful_list ×2, mcp_tools, stream ×2), and the sprint's +1 (183 vs 182 passed) is the
  new gated example.
- "make test FAIL = sprint regression" — refuted: sole failure TestRunCommand_
  PipedStdoutFlushesPerLine = known flake #338, 4/4 standalone passes in the sprint worktree.
- "worktree binary stamps prove provenance" — REFUTED as a method: go build in nested
  .claude/worktrees stamps the MAIN repo's vcs.revision (+its dirty flag) — version output
  said c4dd9aa1f-dirty for a binary built from 64ddd6021 sources. Verified behaviorally
  instead (fixtures only the new parser accepts). Recorded to agent memory.
- "R2 needs exactly two block-loop patches" (plan D6) — refuted by the executor: four loops
  (parseRecordLiteral's two block paths added); record/record-update field parsing untouched.

**Retro lane**: skill fix (the one allowed) — sprint-executor SKILL.md gains a MANDATORY
completion gate for sprint artifacts (tick plan checkboxes + update/CREATE sprint JSON in the
WORKTREE so it rides the PR), traced to 2 recorded frictions: iteration 8 ("plan checkboxes
unticked, sprint JSON left uncommitted") + iteration 9 (sprint JSON absent from the worktree
entirely — root cause: the planner created it in the main tree uncommitted, so the executor's
existing instructions had nothing to act on; evaluator reconstructed bookkeeping from commit
history). Watch list (single instances, no edit): (a) mid-flight resume required inferring
state from commits/worktrees — if a second iteration dies mid-loop, add a resume-detection
step to mission-control's Gate 2; (b) go-vcs-stamp-in-worktrees (memory note written).

**Next**: Iteration 10 — queue #11 m-stdlib-regex (NEW-DOC, clause 4; linear-time RE2-style
engine MANDATED; via builtin-developer skill, ~1–2d). Carry forward: %-row +
m-record-update-local-resolution doc-status re-check (iteration 4). PARKED for human
(cumulative): tier-assignment ratification (release gate); feedback-gate production ops; haiku
causal re-run; scope-params release-gate re-score; frontier-failure validation of the 8
frontier benchmarks + 4 remaining sketches; issue #341 triage call; NEW — **rig A/B for
m-syntax-ai-forgiving** (old vs new parser, `;`-family benchmarks, 5 trials, 1M-token cap,
compile_error Δ — the sprint's REAL success metric; GPU step under rig_lock_acquire wait, or
fold into the next OS-rotation window).

## 10 — 2026-07-11 — m-stdlib-regex design doc created (NEW-DOC stage; queue #11, bar clause 4)

**Picked**: Queue #11 m-stdlib-regex — top open item (all of #1–10 LANDED), explicitly named
"Next" by iteration 9's log. NEW-DOC item → Gate-3 routes to design-doc-creator as the stage.
Serves v1.0 bar clause 4 (ORCHESTRATION FLAGSHIP, R7: linear-time regex builtin MANDATED — "an
orchestration 1.0 without them is a credibility hole").

**Reality check**: regex genuinely absent at v0.29.2 — `grep -rn "_regex_\|std/regex" internal/
std/` → 0 matches; `ls std/ | grep regex` → nothing. No prior regex design doc (find + SimHash/
grep dedup, no neural GPU call). Builtin architecture confirmed: `internal/eval/builtins_*.go`
register `Builtins["_name"] = &BuiltinFunc{Name,Impl,NumArgs,IsPure}`; exposed via `std/*.ail`
`export pure func` wrappers (`std/string.ail`, `std/json.ail`). Result-error convention =
`json.decode -> Result[Json,string]`; opaque-handle precedent = `std/process.ail`
`ProcessHandle = ProcessHandle(int)`. **Key finding: Go's stdlib `regexp` IS the RE2 engine** —
linear-time guaranteed by construction → "wrap, don't build" turns a multi-week engine into a
2-day builtin.

**Shipped**: design doc `design_docs/planned/v0_30_0/m-stdlib-regex.md` (Planned, P1, 2d est).
Two-stage API (`compile -> Result[Regex,string]` + total match fns), RE2 subset contract
(no backref/lookaround, documented as the linear-time price), pure `! {}` throughout, full
Conflict Surface (purely additive — namespace-collision only, no grammar change), Axiom net +8,
Risks, Testing (headline: catastrophic-backtracking wall-time bound test). HARD GATE satisfied:
binary rebuilt to v0.29.2-29-gc533bb51c (matches `git describe`), the exact `Regex`/`RegexMatch`
+ 6 signatures `ailang check`-clean (`✓ No errors found!`). NOT executed — NEW-DOC stage
deliverable is the verified doc; continuous 2h schedule picks up sprint-planner next.

**Routing evidence**: model=opus task-class=design round1-score=n/a rounds=1 corrections=0
(controller-model design per the 2026-07-11 Opus TEMP switch; design-doc-creator hard gates
applied — live `ailang check` verification + Conflict Surface for the `internal/eval/` touch).

**Ruled out**:
- "A linear-time regex needs a hand-rolled RE2 engine (multi-week)" — refuted: Go's stdlib
  `regexp` is already RE2/linear-time (golang.org/pkg/regexp + Russ Cox swtch.com); wrapping it
  satisfies the R7 mandate directly. This is THE scope-defining decision (High-Impact table).
- "Regex might already exist partially" — refuted: 0 `_regex_`/`std/regex` matches anywhere.
- "This is a parser/grammar change needing disambiguation" — refuted: patterns are ordinary
  string literals; the feature adds no syntax/AST/token — Conflict Surface is namespace-only.

**Retro lane**: none. No ≥2-friction skill gap surfaced this iteration (clean design stage,
hard gates passed first time). Single observation (no edit): design-doc-creator's create script
runs a neural (ollama) related-doc search — a latent GPU touch; sidestepped by manual
SimHash/grep dedup per the mission GPU rule. If it recurs as friction, add an explicit
`--no-neural`/GPU-rule note to the skill.

**Next**: Iteration 11 — sprint-planner for m-stdlib-regex (Design Freeze is all-but-closed: 2
open items = final capture-group record naming + replaceAll `$1` syntax subset), then
sprint-executor in a worktree, then sprint-evaluator. Carry forward unchanged: %-row +
m-record-update-local-resolution doc-status re-check. PARKED for human (cumulative, unchanged):
tier-assignment ratification (release gate); feedback-gate production ops; haiku causal re-run;
scope-params release-gate re-score; frontier-failure validation of the 8 + 4 sketches; issue
#341 triage; rig A/B for m-syntax-ai-forgiving (GPU step, the `;`-family compile_error Δ).

## 12 — 2026-07-11 — Iteration 11: m-stdlib-regex LANDED — AILANG has linear-time (RE2) regex; v1.0 bar clause 4's regex half closed (sixth consecutive round-1-clean full loop)

**Picked**: m-stdlib-regex (queue #11, [DOC-READY] top [NEXT]). P1, bar v2 clause 4
(orchestration flagship — "linear-time regex + URL-parse builtins, both verified absent, a
credibility hole"). ~2d. Design doc created iteration 10; this run did plan→execute→evaluate→
land. Inbox at OBSERVE: 4 eval-suite broadcasts (Started + 52.7%/81.6% partials — informational,
in line with baselines, did NOT outrank). Dev CI green per-workflow @ 4f8f087af at OBSERVE. GPU:
pure Go builtin + AILANG stdlib + cloud eval → touches NO GPU, no rig.lock (asked at routing).

**Reality check** (read-only @ v0.29.2-35-gb62ab5433, rebuilt, --version==git describe): regex
VERIFIED absent (`grep _regex_/std/regex` → 0; no internal/eval/builtins_regex.go); no sprint
plan existed. De-risking findings for the executor (recorded in plan + JSON): **F1** — `_str_slice`/
`_str_len` are RUNE-indexed but Go `regexp` returns BYTE offsets (the doc's #1 risk, CONFIRMED
real); **F2** — `std/embed.go` is `//go:embed *.ail` glob → `std/regex.ail` auto-embeds, no
embed.go edit; **F4** — current changelog is `changelogs/v0.18-current.md`.

**Shipped** (full loop, all Opus; round-1 clean):
- Plan (Opus): 3 milestones, sprint JSON + `m-stdlib-regex-sprint-plan.md` (commit 7aa24ba99 to dev).
- Execute (Opus, worktree `sprint/m-stdlib-regex`): M1 `internal/builtins/regex.go` — 6 `_regex_*`
  builtins via the MODERN `RegisterEffectBuiltin` system, NOT the legacy `internal/eval/builtins_json.go`
  path the doc drafted (**D-ARCH** — biggest deviation; the doc's file path was outdated, semantics/
  API identical), mutex-guarded memoized `*regexp.Regexp` cache, byte→rune span conversion (F1);
  tests pass `-count=20` incl. linear-time `(a+)+$`<100ms + multibyte rune fixture + backref/lookaround
  Err-never-panics + cache memoization; golden `builtin_types` regenerated (6454c3fe0). M2 `std/regex.ail`
  (opaque `Regex(string)` + `RegexMatch` + 6 pure wrappers) + 3 runnable examples (basics/capture/
  **log_orchestration** = the clause-4 use case) — groups via `nth_or` since lists use `nth` not `[i]`
  subscript (the doc's `m.groups[1]` example was not runnable) (0ca5c0bfd). M3 stdlib parse-test guard
  + LIMITATIONS (RE2 subset) + stability tier (`std/regex`→Experimental) + CHANGELOG (6478aff99).
- Evaluate (Opus, independence caveat mitigated by INDEPENDENT reproduction with FRESH cases):
  **PASS 97/100** round 1 — tests 20/20 (regex pkgs green ×20; full make-test's sole red =
  pre-existing flake `TestRunCommand_PipedStdoutFlushesPerLine`, passes standalone), lint 10/10,
  AC 28/30 (−2 website raw-loader N/A: no regex feature-doc page), quality 15/15, docs 15/15,
  fidelity 9/10 (−1 D-ARCH). Independent checks: backref `(a)\1`→Err; **CJK `日本語 world` → "world"
  at RUNE span [4,9) not byte [10,15) — F1 proven on a script the executor never tested**; findAll
  3 matches; coverage recomputed 89.5%; verify-examples 32/29. Report committed.
- Integrate: docs→`implemented/v0_30_0`; branch merged origin/dev during work (sibling docs commit
  bf4937ec3, index.jsx — this branch never touched it, merged clean). PR #343 → squash-merge
  0b0ed7ea0 (auto-merge SQUASH; all required checks green = the merge gate). Dev post-merge CI
  in-progress (identical content to the green PR checks — expected green).

**Routing evidence**:
- model=opus task-class=plan round1-quality=high (3 live de-risking findings F1/F2/F4 that
  materially changed execution: F1 became M1's headline correctness requirement) rounds=1 corrections=0
- model=opus task-class=execute round1-score=97 rounds=1 corrections=0 — QUALITY HIGHLIGHT:
  correctly routed to the MODERN builtins subsystem instead of implementing the doc's outdated
  `internal/eval` path literally (D-ARCH), and caught two non-runnable design-doc examples
  (`m.groups[1]` subscript; qualified type `R.Regex` in a signature) — fixed both. Sprint artifacts
  COMPLETE this iteration (JSON status=completed + notes + plan checkboxes) — the iteration-8/9
  completion-gate skill fix HELD (no reconstruction needed).
- model=opus task-class=evaluate rounds=1 — independence caveat handled by reproducing headline
  claims with fresh inputs (CJK subject, larger findAll) rather than trusting executor tests.

**Ruled out**:
- "regex needs a hand-rolled multi-week engine" — refuted at design: Go `regexp` IS RE2 (linear-time
  by construction) → 2-day wrap. Held.
- "the design doc's `internal/eval/builtins_regex.go` + `m.groups[1]` are correct" — refuted live:
  current builtins live in `internal/builtins/` (RegisterEffectBuiltin); lists use `nth`/`nth_or`,
  not `[i]`. Both corrected; recorded as D-ARCH + example fix.
- "`make test` FAIL = regex regression" — refuted: sole red = pre-existing timing flake
  (`TestRunCommand_PipedStdoutFlushesPerLine`), passes in isolation + package-alone; regex is
  additive in a different package.
- "byte offsets are fine to expose" — refuted by F1 + independently on CJK: exposing Go's byte
  offsets would break consistency with `std/string`'s rune indexing; conversion is load-bearing.

**Retro lane**: NO skill edit (max one/iteration, requires ≥2 frictions at the same gap — none
recurred; the iteration-8/9 completion-gate fix already HELD this run). Process: none. Watch list
(single instances, no edit): (a) the design-doc-creator's `ailang check`-verification signed off
the `internal/eval` file path + `m.groups[1]`/`R.Regex`-in-signature examples that don't actually
run — the HARD GATE checks *type signatures* but not *example runnability* nor *current subsystem*;
if a 2nd doc ships non-runnable examples, add "run every example fragment" to design-doc-creator's
gate. (b) `.ailang/state` is gitignored so the sprint JSON can't ride the PR branch — mirrored to
main-tree + worktree by hand; if this recurs, mission-control should note the evaluator reads it
from disk, not the branch.

**Next**: Iteration 12 — queue #12 m-stdlib-url-parse (NEW-DOC, clause 4; the OTHER half of the
builtin mandate, ~1d) starts at design-doc-creator; OR #13 m-dx-match-hof (clause 3, Conflict
Surface mandatory). Carry forward unchanged: %-row + m-record-update-local-resolution doc-status
re-check. PARKED for human (cumulative, unchanged): tier-assignment ratification (release gate);
feedback-gate production ops; haiku causal re-run; scope-params release-gate re-score; frontier-
failure validation of the 8 + 4 sketches; issue #341 triage; rig A/B for m-syntax-ai-forgiving
(GPU step). NEW parked: confirm dev post-merge CI 0b0ed7ea0 green (in-progress at close).

## 13 — 2026-07-12 — Iteration 12: m-stdlib-url-parse DESIGN DOC CREATED (NEW-DOC stage; queue #12, bar clause 4) — plus iteration-11 confirmed already-landed (no duplication) + Gate-1 stale-state skill fix

**Picked**: queue #12 m-stdlib-url-parse (NEW-DOC, clause 4 orchestration flagship, ~1d). Reached
only after the pick's headline surprise: this scheduled run booted on a STALE local checkout and
initially mis-read the mission as mid-flight on iteration 11 (regex). See "Ruled out".

**The already-landed catch (the real story of this run)**: local dev was `bf4937ec3` while
origin/dev was 2 commits ahead — `0b0ed7ea0` (regex feature, PR #343, MERGED 22:53) +
`c88e1cf93` (iteration-11 record). The local mission log had no iter-11 entry, the local sprint
JSON had no eval score, and the local queue still said `[DOC-READY]` — all STALE. Gate 1/2 read
that stale local state and I began a redundant "resume at sprint-evaluator": restored the lost
`sprint/m-stdlib-regex` branch, built + independently verified the executor's work in a worktree
(Go tests green, 12 non-vacuous regex tests incl. TestRegexLinearTime + TestRegexRuneIndices,
verify-examples 183/5/5 with the 5 = pre-existing #341 set, all clean), and ran a fresh Fable/Opus
sprint-eval that returned **PASS 96/100**. Only at Gate-3b push-prep — `git fetch origin` +
`git merge-base --is-ancestor` BEFORE any push — did the already-merged state surface, preventing
a duplicate/conflicting merge. Verified the merged `0b0ed7ea0` code is byte-identical to what I
evaluated, so the independent eval stands as corroboration of the landed regex work, not waste.
Cleaned up: removed the redundant worktree + `sprint/m-stdlib-regex` branch.

**Reality check (url-parse, live at origin/dev c88e1cf93 + built binary v0.29.2-39-gc88e1cf93)**:
no url-parse design doc exists (url-ENCODE shipped v0.19.2 in std/net; url-PARSE is the unbuilt
complement). Verified premises: URL fns live in `std/net` (`std/net.ail` + `internal/builtins/net.go`),
existing `urlEncode`/`urlEncodeForm` are PURE (`IsPure:true`, no Net cap) → url-parse mirrors that
pattern exactly. GPU: NONE (pure Go wrap + design doc + cloud model). Worked in a worktree branched
from origin/dev (up-to-date, regex as live sibling reference; main tree is dirty w/ sibling work).

**Shipped** (NEW-DOC deliverable):
- design-doc-creator (Opus) → `design_docs/planned/v0_30_0/m-stdlib-url-parse.md` (567 lines),
  committed `b633bdbd5`. Wrap Go `net/url` (RFC-3986), extend `std/net`, pure `! {}`, no new module.
  Public API `ailang check`-clean (HARD GATE), re-verified independently by the controller:
    - `type Url = { scheme, host, port, path, query, fragment : string }`
    - `parseUrl(s) -> Result[Url, string]`  (Err on malformed — CP2, no silent fallback)
    - `parseQuery(s) -> [{name, value}]`     (order-preserving inverse of urlEncodeForm)
  Decisions: `Url` = plain record (data, not opaque handle like regex's Regex); `port: string`
  (empty=absent, no 0/-1 sentinel); parseQuery keeps order+duplicates (not Go's sorted map).
  Conflict Surface = additive namespace-only, zero collisions. Avoided the regex doc's `groups[i]`
  trap: examples use `nth_or` (lists have no subscript) + `${}` (no `++`) — both caught live by
  `ailang check` during authoring.
- Integration: PR #344 → merge (auto-merge, MERGE method per #343 precedent), dev CI green
  per-workflow (Gate 3b). Queue #12 → [DOC-READY]. NEXT stage: sprint-planner → executor → evaluator.

**Routing evidence**:
- model=opus task-class=design(NEW-DOC) round1-quality=high (ailang check-clean on first
  independent re-run; caught 2 example-syntax traps live before shipping; premises all confirmed)
  rounds=1 corrections=0
- model=opus task-class=evaluate(regex, corroborative) round1-score=96 rounds=1 — independent
  re-verification of an already-landed sprint; matched the origin iter-11 eval, confirming the
  landed work rather than duplicating it

**Ruled out**:
- "Iteration 11 (regex) is mid-flight and needs resuming" — REFUTED: it fully LANDED via PR #343
  (`0b0ed7ea0`) + record (`c88e1cf93`) at ~22:53–22:57 the prior night. The mid-flight signal was
  entirely a STALE LOCAL CHECKOUT artifact (local mission log/queue/sprint-JSON never synced after
  the GitHub squash-merge). Lesson → skill fix below.
- "The restored sprint branch's binary provenance proves which sources built it" — N/A here (I
  verified behaviorally + by diffing content against the merged commit), but re-confirms the
  standing go-vcs-stamp-in-worktrees caveat.
- "verify-examples' 5 fails are regex/url regressions" — REFUTED: exactly the pre-existing #341
  set (effectful_list ×2, mcp_tools, stream ×2), none import the new modules.

**Retro lane**: skill fix (the one allowed) — mission-control Gate 1 (OBSERVE) gains a MANDATORY
"**sync to origin FIRST**" step (`git fetch origin`; compare local dev ↔ origin/dev; if behind,
treat origin as ground truth) AND Gate 2's reality-check now explicitly includes "is the picked
item already LANDED on origin / merged via a PR". Traced to 2 recorded frictions pointing at the
same gap: iteration 9's watch-list ("add a resume-detection step to Gate 2" — recorded, not yet
acted) + iteration 12 (this run: stale-local-state drove a redundant regex resume all the way to a
full re-evaluation before the Gate-3b fetch caught it). Both = mission-control never reconciles the
local checkout against origin at OBSERVE, so a stale tree causes redundant/duplicate work.
Watch list (single instances, no edit): main-tree dirtiness from concurrent sibling agents
(uplift.go/uplift_test.go untracked, docs modified) — handled by working in origin-based worktrees,
already covered by existing guidance.

**Next**: Iteration 13 — queue #12 m-stdlib-url-parse sprint-planner → executor → evaluator (the
design doc is DOC-READY, Public API frozen + check-clean, ~1d, GPU: none). Carry forward
unchanged: %-row + m-record-update-local-resolution doc-status re-check. PARKED for human
(cumulative, unchanged): tier-assignment ratification (release gate); feedback-gate production
ops; haiku causal re-run; scope-params release-gate re-score; frontier-failure validation of the
8 + 4 sketches; issue #341 triage; rig A/B for m-syntax-ai-forgiving (GPU, the `;`-family
compile_error Δ).

---

## 10 — 2026-07-12 — v1.0 SCOPE SET: full backlog triage (interactive, Mark + Opus)

**Picked**: n/a — decision session. Mark: "triage the other design docs and see what I want in v1.0."

**Reality check**: all 69 non-gating planned docs reality-checked by 5 parallel agents (used the
docs-sync skill's frame; per-doc verification via git/code). Decomposition of the "~85 backlog":
- **~12 ghosts** (shipped, headers stale) — 10 confirmed + evidence, reconciled to
  implemented/v0_29_0 (GHOSTS-RECONCILED-2026-07-12.md); 2 CONFLICTED (agents split ghost-vs-open:
  m-dx-record-cons-pattern, m-dx-tapp-trecord-unification) kept OPEN pending repro.
- **~30 not-v1.0** — eval-infra (rig/harness), cloud-infra (coordinator/dashboard), motoko-fork
  (separate mission), post-v1. Ruled OUT of the v1.0 gate.
- **~18 genuine candidates** — almost all clause-3 accessibility (footgun/DX/prompt), a few clause-4.

**Shipped**: charter STATUS stamp + queue restructured from a flat 14-item list into clause groups,
now ~33 open items (Mark chose FULL SCOPE across all 4 AskUserQuestion cluster decisions: whole
clause-3 cluster IN, both DX tooling investments IN, full clause-4 orchestration surface IN,
rig/cloud/motoko OUT). 10 ghost docs git-mv'd to implemented/. "~80 not-gating" corrected to ~30.

**Routing evidence**: model=opus task-class=triage-coordination round1-score=n/a rounds=1
corrections=1 (one reality-check agent HUNG mid-batch — same unbounded-wait class as iteration 13;
re-ran it tighter, it completed; the cross-check of both runs caught the 2 ghost-vs-open
disagreements, which is why they're kept OPEN not archived). Parallel-agent triage = ~65k tokens/
agent × 5.

**Ruled out**:
- "~85 real backlog" — inflated; ~12 ghosts + ~30 infra/tooling/motoko = ~42 that don't represent
  open v1.0 language work. Real v1.0-relevant open set was ~18 before Mark's scope expansion.
- Archiving the 2 conflicted docs on ambiguous evidence (wrongly archiving a live doc is worse than
  leaving a possibly-fixed one in the queue — the loop's reality-check catches the latter).

**Retro lane**: process-fix (v1.0 scope expansion in the charter + ghost reconciliation). RECORDED
(not yet actionable): iteration-12 booted on a stale checkout and nearly re-did the landed regex
sprint — 2nd instance of the stale-checkout-at-Gate-1 gap; the loop already flagged a "sync to
origin FIRST" skill fix (log 13). If it recurs → that skill edit.

**Next**: loop continues at #12 (url-parse build), then the clause-3 accessibility cluster —
now the bulk of v1.0. Timeline honest: ~33 open items ≈ 40–55 sprint-days.

## 14 — 2026-07-12 — Iteration 13: m-stdlib-url-parse BUILT + LANDED — AILANG parses URLs; v1.0 bar clause 4's URL-parse half closed (regex + URL-parse gate now fully met)

**Picked**: queue #12 m-stdlib-url-parse build (top `[NEXT]`; design doc was DOC-READY from iter 12,
API frozen + `ailang check`-clean; ~1d, GPU none). The URL-parse complement to the regex half (#11)
and the existing `urlEncode`/`urlEncodeForm` — both named v1.0 bar clause-4 release gates.

**Reality check**: design doc present at `planned/v0_30_0/m-stdlib-url-parse.md` (DOC-READY, not yet
built). `parseUrl`/`parseQuery` builtins **genuinely absent** — the only `parseQuery*` hits in the
tree are unrelated `parseQueryArgs` in `internal/apiserver/` (HTTP query-arg extraction, different
thing). Only PR #344 (the doc) merged; no build PR. origin/dev queue tag = `[DOC-READY]`, `[NEXT]`
= build. Confirmed a real, unbuilt task. Gate-1 origin-sync clean this run: local dev == origin/dev
== HEAD at start (the new skill step held).

**Shipped**: full build loop headless in an origin/dev worktree.
- Opus executor: `_net_url_parse` + `_net_url_parse_query` pure builtins (`IsPure:true, Effect:""`,
  no Net cap) in the modern `internal/builtins/net.go` (+206), wrapping Go `net/url`; `Url` flat
  record + `parseUrl`/`parseQuery` wrappers in `std/net.ail` (+50); 26 non-vacuous Go tests
  (`net_url_parse_test.go`, +240) incl. IPv6 `[::1]:80`→host="::1"/port="80", no-port→"", userinfo,
  error-never-panics on `%zz`, percent-decode, source-order + duplicate-key preservation, bare-key,
  empty→[], purity, round-trip `parseQuery(urlEncodeForm(pairs))`; 2 runnable examples
  (`url_parse_basics`, `url_route_dispatch`); docs (CHANGELOG, LIMITATIONS resolved-entry, stability
  Experimental tier). Commits `5ea30d8b1`/`ea5a60f86`/`34e518a13`.
- Independent Opus evaluator: **round-1 FAIL 80/100** — single BLOCKER: `internal/pipeline/testdata/
  builtin_types.golden` not regenerated after registering the two builtins → repo-wide `make test`
  red (auto-reject gate), everything else 100%. **Round-2**: golden regenerated (exactly 2 correct
  lines, commit `52ac50e3b`) → repo-wide `make test` green except a pre-existing flaky → **PASS
  100/100**.
- Controller independent verification (not taking the executor's word): rebuilt binary, ran the 26
  tests + full builtins package green, both examples produce the design-doc's expected output
  (field decomposition + percent-decoded params; host/path routing + query branching), consumer
  type-checks clean.
- Integration: PR #347 → squash-merge `a8628a40c` on dev, auto-merge fired on green required checks
  (Gate 3b — the required checks that gated the merge were the observed green; post-merge dev CI run
  on the merge commit was still in-flight at report time, no failures). Queue #12 → `[LANDED]`,
  design doc → `implemented/v0_30_0/`.

**Routing evidence**: model=opus task-class=execute round1-quality=high (faithful to every frozen
decision; only miss = the golden-regen step, a known mechanical follow-on to adding a builtin)
rounds=1 corrections=1 (the golden fix, applied by the controller deterministically — a single
`UPDATE_GOLDEN=1` regen, not a re-loop). model=opus task-class=evaluate round1-score=80 (FAIL, hard
gate on tests-red) → round2-score=100 rounds=2. Evaluator independence caveat holds (Opus judges
Opus while on the Monday-reverting override) — mitigated here by the evaluator reproducing the red
`make test` and the exact 2-line golden diff itself, and by the controller's own independent
re-run; the BLOCKER was a real, reproducible test failure, not a judgment call.

**Ruled out**:
- "`cmd/ailang` `TestRunCommand_PipedStdoutFlushesPerLine` is a url-parse regression" — REFUTED: it
  fails only under parallel `make test` load, passes 3/3 in isolation, and this sprint touches zero
  `cmd/`/stdout code. Pre-existing timing-flaky; flagged for dev-health (sibling of #341), not a gate.
- "origin/dev's `BenchmarkDashboard/*` changes in my range-diff are executor contamination" —
  REFUTED: the 3 executor commits touch only url-parse files; origin/dev advanced under me
  (sibling dashboard commits `2de953067`, then the M-EVAL-DATA-HOSTING-DECOUPLE set) with zero file
  overlap → a clean divergence, clean merge.

**Retro lane**: none. No skill edit (the one-allowed slot needs ≥2 frictions at the same gap; this
run had none — the golden-regen miss is a known mechanical follow-on, already the evaluator's job to
catch, and it did). Gate-1 origin-sync + Gate-2 already-landed checks (added last iteration) worked
as intended: clean sync, no redundant work. Every wait bounded (Standing rule 6): the Gate-3b CI
poll used a 29-min hard deadline and exited on MERGED well inside it. Watch list (single instances,
no edit): (a) the pre-existing `PipedStdoutFlushesPerLine` flaky under parallel `make test` — if it
bites another iteration, file a dev-health issue; (b) `ailang check <bare-stdlib-file>` derives the
module name from the filename (`net.ail` → "invalid characters") so a stdlib module can't be checked
in isolation — check via a consumer instead; minor, single instance.

**Next**: Iteration 14 — the clause-3 accessibility cluster (now the bulk of v1.0, ~33 open items ≈
40–55 sprint-days). Cheapest DOC-READY starter: m-module-less-run-fail-loud (MOD011, 0.5d, doc at
`planned/v0_30_0/`); the NEW-DOC footgun fixes (m-dx-match-hof R4a, m-poly-arith-lambda R4b,
m-arity-style-diagnostic R4c, …) each need design-doc-creator first (Conflict Surface mandatory).
PARKED for human (cumulative, unchanged): tier-assignment ratification (release gate); feedback-gate
production ops; haiku causal re-run; scope-params release-gate re-score; frontier-failure validation
of the 8 + 4 sketches; issue #341 triage; rig A/B for m-syntax-ai-forgiving (GPU). Carry forward:
%-row + m-record-update-local-resolution doc-status re-check; the two dev-health flakies above.

---

## 15 — 2026-07-12 — Iteration 14: m-module-less-run-fail-loud BUILT + LANDED (MOD014) — module-less files now FAIL LOUDLY; clause-3 footgun burn-down opened

**Picked**: queue #13 `m-module-less-run-fail-loud` (top `[NEXT]` of the clause-3 accessibility
cluster; iteration-13's **Next** named it as the cheapest DOC-READY starter — MOD011 hint, 0.5d,
doc at `planned/v0_30_0/`, GPU none). A silent-success footgun (violates CP2 NO-SILENT-FALLBACKS).

**Reality check**: Gate-1 origin-sync clean (local dev == origin/dev == f72313e33 at start; all 3
dev CI workflows green — no red outranking the queue). Item is REAL and UNBUILT: only the design
doc was committed (`a14789f20`, "design(dx)", 135 lines + queue line — no implementation).
`validateModulePath` still had the `Module==nil { return nil }` early-accept (pipeline_module.go:665).
Reproduced the bug live at HEAD after rebuilding both binaries: module-less file `ailang run` prints
"✓ Running", exit 0, `SHOULD_PRINT` never prints; `ailang check` → "✓ No errors found!". **Two
Conflict-Surface defects found in the doc and corrected before/during the sprint** (see Ruled out).

**Shipped**: full build loop headless in an origin/dev worktree → **PR #349 → merge `c2ffd1b5c`**,
design doc + sprint plan → `implemented/v0_30_0/`.
- Opus sprint-planner → 2-milestone plan + JSON (M1 MOD014 diagnostic; M2 block_demo remediation +
  docs + doc-move), carrying the MOD011→MOD014 correction.
- Opus executor (worktree, commits `138eebef2` M1 / `42b1cd601` M2): replaced the early-accept with
  a loud **MOD014** error gated on `len(Funcs) > 0` (fires for both `run` AND `check` at the shared
  chokepoint — systemic); +120-line `internal/pipeline/module_less_test.go` (5 tests, 2 fail-before);
  footgun fixture in `internal/diag/` (count 17→18); `block_demo.ail` given `module
  examples/runnable/block_demo`; CHANGELOG + LIMITATIONS resolved-entries; MOD011→MOD014 across the
  design doc + `git mv` to implemented/. **No golden regenerated** (all golden tests passed).
- Controller independent spot-check (rebuilt worktree binary): module-less run/check → MOD014 exit 1;
  `1+1`→`2` exit 0; proper module → prints, exit 0. All four behaviors confirmed.
- Independent Opus evaluator (built a BASE origin/dev binary in a throwaway worktree to prove
  non-vacuity + pre-existing-failure claims): **round-1 PASS 100/100**. Verified the guard=Funcs-only
  decision in code (`parser_file.go:72-78`), proved the 2 new tests FAIL on base / PASS on fix,
  confirmed the 5 `verify-examples` failures + the `PipedStdoutFlushesPerLine` flaky are pre-existing
  and unrelated (identical on base binary). Recommended merge.
- Gate 3b: PR #349 auto-merged on green required checks; **post-merge CI on `c2ffd1b5c` = success**
  across all 3 workflows (CI, Build and Release, Deploy Docs). Bounded 30-min poll exited on green at
  ~17 min. Queue #13 → `[LANDED]`.

**Routing evidence**: model=opus task-class=plan (2-milestone, correct scope, folded the reality-check
correction) rounds=1 corrections=0. model=opus task-class=execute round1-quality=high (found and
corrected a SECOND doc defect mid-sprint — the guard predicate — with a code-verified rationale;
faithful otherwise) rounds=1 corrections=0. model=opus task-class=evaluate round1-score=100 rounds=1
(FIFTH-plus consecutive round-1 pass). Evaluator-independence caveat holds (Opus judges Opus) —
mitigated HARD this run: the evaluator built a base-origin/dev binary and independently reproduced the
original bug, the test non-vacuity (fail-on-base), and every pre-existing-failure claim; nothing rested
on the executor's report.

**Ruled out**:
- "The design doc's proposed error code MOD011 is available" — REFUTED at Gate 2: MOD011 is the LIVE
  module-path-collision diagnostic (since v0.10.9, `pipeline_module.go:566`). Allocated = MOD001–013
  (only MOD008 a free gap). Reassigned to **MOD014** (next fresh). One `grep` would have caught it.
- "Bare-expression eval never reaches `validateModulePath` (routes to runSingle), so a 3-way
  `Funcs||Statements||Decls` guard is safe" (the doc's Conflict-Surface premise) — REFUTED by the
  executor in code: a bare-expr FILE does route through the module pipeline and reach
  `validateModulePath`, and the parser mirrors the expression into `Statements`+`Decls` — so the
  doc's OR would have broken `ailang run 1+1`. Correct guard = `len(Funcs) > 0` ONLY. The observable
  (`1+1`→`2`) was right; the doc's stated MECHANISM was wrong.
- "The 5 verify-examples failures / the PipedStdout flaky are this sprint's regressions" — REFUTED:
  all identical on a base origin/dev binary; blast radius stays exactly 1 (block_demo). Statement-only
  module-less files (no funcs) still exit 0 by design — a degenerate, documented, non-footgun gap.

**Retro lane**: **skill-fix** — `.claude/skills/design-doc-creator/SKILL.md`: extended the existing
"Verify Every Language Claim (HARD GATE)" section with two more must-verify claim classes + a case
study. Justified by TWO same-gap frictions THIS iteration (both = the doc's Verification Log marked a
technical claim "Confirmed" that a one-command code-check refuted): (F1) the MOD011 error-code
collision, (F2) the bare-expr routing-mechanism claim. New requirements: (1) grep any newly-proposed
error/diagnostic code for allocation before writing it into a doc; (2) verify Conflict-Surface
routing/mechanism claims by reading the code path, not inferring from observable output.

**Next**: Iteration 15 — continue the clause-3 accessibility cluster. Remaining DOC-READY/small
diagnostics: m-match-xcheck-error-quality (0.5d) · m-dx-split-argument-warning (1d) ·
m-dx-json-bool-coercion (0.5d). The NEW-DOC footgun fixes (m-dx-match-hof R4a, m-poly-arith-lambda
R4b, m-arity-style-diagnostic R4c) each need design-doc-creator first (Conflict Surface mandatory —
and now the error-code + mechanism verification gates). PARKED for human (cumulative, unchanged):
tier-assignment ratification (release gate); feedback-gate production ops; haiku causal re-run;
scope-params release-gate re-score; frontier-failure validation of the 8 + 4 sketches; issue #341
triage; rig A/B for m-syntax-ai-forgiving (GPU). Carry forward: %-row + m-record-update-local-resolution
doc-status re-check; the two dev-health flakies (`PipedStdoutFlushesPerLine`; 5 effect-row example
failures — candidate dev-health issue, sibling of #341).

## 16 — 2026-07-12 — Iteration 15: m-match-xcheck-error-quality BUILT + LANDED — foreign-ctor errors now enumerate transitively-known constructors

**Picked**: queue #14 `m-match-xcheck-error-quality` (top DOC-READY of the clause-3 accessibility
cluster; iteration-14's **Next** named it as the cheapest 0.5d starter). A P2 diagnostic-quality
paper-cut: `MatchForeignConstructorError` correctly names both ADTs + the offending constructor, but
the `<scrutinee ADT>'s constructors are:` suggestion line was EMPTY when the scrutinee's ADT was
known only transitively.

**Reality check**: Gate-1 origin-sync caught local dev 4 commits BEHIND origin/dev (`f72313e33` vs
`363164dab` — iteration 14 had landed via PR #350). Read all mission state from origin. dev CI green
per-workflow (CI/Build/Docs all success @ 363164dab) — no red outranking the queue. Item REAL and
UNBUILT: not in `implemented/`, no merged PR. Reproduced live at origin/dev HEAD in a worktree
(rebuilt binary): a module importing only `std/json`+`std/result` (NOT `std/option`), matching
`asNumber(jnum(3.0))` with `Ok/Err`, printed `Option's constructors are: ` (empty) while
`Result's constructors are: Err, Ok` was populated. Design doc's cited code (`lookupADTConstructors`,
`pipeline_module_imports.go` M-CTOR-AUTO block) verified present; **Option A** chosen per the doc's
own recommendation.

**Shipped**: full build loop headless in an origin/dev worktree → **PR #352 → squash-merge `5aaaff2ed`**,
design doc + sprint plan → `implemented/v0_30_0/`.
- Opus sprint-planner → 2-milestone plan + JSON, with root-cause verified in code (topo-order
  deps-first compile ⇒ every transitive iface registered in `modLinker.GetLoadedModules()` before the
  root type-checks) and the `types` cannot import `link` cycle noted (⇒ pass a plain `map[string]string`).
- Opus executor (worktree, commits `3ded459cc` M1 / `f5498ca0e` M2 / `ecca08b3b` hardening):
  a **diagnostic-only** `Constructor→ADT` registry (`moduleImports.AllCtorTypes`) built from all
  loaded ifaces, plumbed via new `SetDiagnosticConstructorTypes`, consulted by `lookupADTConstructors`
  ONLY when the primary direct/local scan is empty — so it never enters scope. Strengthened the
  existing function-call repro test to assert `Some`+`None` now appear; non-vacuity proven by
  disabling the fallback (line reverts to empty). CHANGELOG + doc-move.
- Independent Opus evaluator: **round-1 PASS 96/100**, no blockers. Re-proved non-vacuity with a
  BASE origin/dev binary (empty on base, `None, Some` on fix), confirmed `Some` stays uncallable
  without importing `std/option` (no scope leak — identical on base+fix), confirmed the error-message
  format string is byte-identical (out-of-scope respected), and that direct/local ctors always win.
  Two −2 deductions (benign same-name-ctor collision undocumented; scope invariant not locked into
  CI) → BOTH addressed in the `ecca08b3b` hardening commit (added
  `TestSchemeImport_DiagnosticRegistryDoesNotLeakIntoScope` + a collision note in the design doc).
- Gate 3b: PR #352 auto-merge on green required checks; bounded 29-min CI poll observed `CI: completed success` on the PR head (`ecca08b3b`) — the required-check green that gated auto-merge; post-merge dev CI re-running on `5aaaff2ed` at record time.
  Queue #14 → `[LANDED]`.

**Routing evidence**: model=opus task-class=plan (2-milestone, correct scope, root-cause code-verified
before planning) rounds=1 corrections=0. model=opus task-class=execute round1-quality=high (clean
Option-A impl; proactively hardened per the evaluator's two non-blocking deductions) rounds=1
corrections=0. model=opus task-class=evaluate round1-score=96 rounds=1 (consecutive round-1 pass
continues). Evaluator-independence caveat holds (Opus judges Opus) — mitigated as before: the
evaluator built a base-origin/dev binary and independently reproduced the empty-vs-populated delta,
the scope-non-leak invariant, and the format-unchanged claim; nothing rested on the executor's report.

**Ruled out**:
- "Bringing transitive constructors into `constructorTypes` would fix the message" — REJECTED by
  design: that would put `Some`/`None` in scope (auto-import), the doc's explicit out-of-scope
  hazard. Kept a SEPARATE diagnostic map consulted only for error text.
- "`types` can reach the linker to enumerate ifaces at error time" — REFUTED: `internal/types` is
  below `internal/link`; importing it is a cycle. The registry is passed down as a plain
  `map[string]string` (the doc's "slimmed-down ADT registry").
- "The strengthened test might be vacuous" — REFUTED twice: executor disabled the fallback (line
  empty); evaluator ran the identical repro on a base binary (empty) vs fix (`None, Some`).

**Retro lane**: **none** (no skill or process edit). One iteration, zero corrections, clean round-1
PASS — no ≥2-friction pattern to route. The only friction (local dev 4 commits behind origin/dev) is
EXACTLY the case Gate 1's origin-sync step already handles, and it worked as designed (caught it
before any stale-state work) — the process fix from iteration 12/13 is paying off, no further change
warranted.

**Next**: Iteration 16 — continue the clause-3 accessibility cluster. Remaining DOC-READY/small
diagnostics: m-dx-json-bool-coercion (0.5d) · m-dx-split-argument-warning (1d). Then the NEW-DOC
footgun fixes (m-dx-match-hof R4a, m-poly-arith-lambda R4b, m-arity-style-diagnostic R4c) each need
design-doc-creator first (Conflict Surface mandatory — incl. the error-code + mechanism verification
gates added iter 14). Also VERIFY-then-route: m-dx-record-cons-pattern, m-dx-tapp-trecord-unification
(may be ghosts — run the repro FIRST). PARKED for human (cumulative, unchanged): tier-assignment
ratification (release gate); feedback-gate production ops; haiku causal re-run; scope-params
release-gate re-score; frontier-failure validation of the 8 + 4 sketches; issue #341 triage; rig A/B
for m-syntax-ai-forgiving (GPU). Carry forward: %-row + m-record-update-local-resolution doc-status
re-check; the two dev-health flakies (`PipedStdoutFlushesPerLine`; 5 effect-row example failures).

## 17 — 2026-07-12 — Iteration 16: m-dx-json-bool-coercion BUILT + LANDED (in-repo half) — std/json.asBoolLoose; Phase-1 firestore-package fix parked out-of-repo

**Picked**: queue #15 `m-dx-json-bool-coercion` (0.5d) — the cheapest remaining DOC-READY diagnostic
in the clause-3 accessibility cluster; iteration-15's **Next** named it as the top starter.

**Reality check**: Gate-1 origin-sync clean (local dev == origin/dev @ `a4d8e58d8`); dev CI green
per-workflow (CI/Build/Docs all success @ a4d8e58d8) — no red outranking the queue. Inbox: 2
eval-suite infos (rig rotation started + 44.4% partial — the known-stale os-rolling banked data, not
a regression) + iteration-15 recap; all ack'd. **The doc's scope did NOT survive contact with the
repo** and split cleanly in two:
- **Already correct**: `std/json.jb(b)` (→ JSON `true`) and `asBool` (JBool → Option) both exist and
  round-trip (`asBool(jb(true)) == Some(true)` verified live). The core encoder/decoder are NOT
  broken — the doc's premise that they are is stale. The real bug is *use of the wrong constructor*
  (`js("true")` = a JSON string) at the package layer.
- **Phase 1 is OUT-OF-REPO**: the buggy `sunholo/firestore/fields.ail` lives in the separate
  `ailang-packages` repo — `find '*firestore*fields.ail'` returns nothing here (only unrelated Go
  infra: `internal/server/auth/firestore.go`, `internal/storage/firestore`). The doc's own dependency
  M-DX-XPKG-RESOLVE also gates testing it. Cannot fix or test here → PARKED for a coordinator task.
- **In-repo deliverable = Phase 2** (`asBoolLoose`) + Phase 3 (teaching, folded into the example).

**Shipped**: full build loop headless in an origin/dev worktree → **PR #354 → squash-merge `5b41b3835`**,
design doc → `implemented/v0_30_0/`. As controller (Opus) I executed directly given the sub-0.5d
mechanical scope, then routed an INDEPENDENT Opus evaluator.
- `std/json.asBoolLoose(j) -> Option[bool]`: accepts `JBool(b)` OR `JString "true"/"false"`, else
  `None` — structured failure, never a silent default (CP2); the "system boundary" (A12) decoder for
  APIs (e.g. Firestore `booleanValue`) that may return a boolean stringified. `asBool` stays the
  choice for values you control. Purely additive — no existing json fn touched (zero regression surface).
- `examples/runnable/json_bool_encoding.ail`: demonstrates the `jb` vs `js` footgun + `asBool`/
  `asBoolLoose` distinction end-to-end (Phase 3 teaching lives in its header comment — the embedded
  prompt was deliberately NOT edited; prompt-budget is GATED in the mission under prompt-diet).
- `internal/repl/json_asboolloose_test.go`: 7-case Firestore `{"field":{"booleanValue": …}}`
  round-trip (real-bool, stringified-bool, non-bool → structured None, missing field). Non-vacuity
  PROVEN — substituting plain `asBool` flips exactly `string_true`/`string_false` to `NOT_BOOL`.
- Independent Opus evaluator: **round-1 PASS 92/100**, no blockers — rebuilt the binary, reran the
  test, independently reproduced non-vacuity in a throwaway copy, ran+type-checked the example,
  confirmed CP2 structured-failure (not silent fallback), verified the firestore `.ail` is genuinely
  absent (parking is honest, not dodging), and that docs match what shipped.
- Gate 3b: PR #354 auto-merge on green required checks; bounded ~19-min CI poll (30-min hard deadline)
  observed all required checks pass (`test` 9m50s, `lint`, `govulncheck`, `build` ×macOS/ubuntu/windows,
  `test-windows`) → squash-merged `5b41b3835`; origin/dev advanced `a4d8e58d8..5b41b3835`.
  Queue #15 → `[LANDED]`. std/json.ail is NOT in the frozen STDLIB manifest → no golden to regen
  (proactively confirmed after iteration-13's stale-golden blocker).

**Routing evidence**: model=opus task-class=execute round1-quality=high (clean additive Phase-2 impl;
reality-check correctly de-scoped the out-of-repo Phase 1 before any wasted work) rounds=1
corrections=0. model=opus task-class=evaluate round1-score=92 rounds=1 (round-1 pass streak continues).
Evaluator-independence caveat holds (Opus judges Opus) — mitigated as before: the evaluator rebuilt
the binary, reproduced non-vacuity itself in a throwaway copy, and independently verified the
out-of-repo parking claim; nothing rested on the executor's report. No sprint-planner this iteration:
sub-0.5d additive mechanical scope after the de-scope, so controller executed directly — flagged as a
Gate-5 watch-item below, not yet a process change.

**Ruled out**:
- "std/json's boolean encode/decode is broken (the doc's premise)" — REFUTED live: `jb`+`asBool`
  round-trip correctly. The defect is wrong-constructor USE at the package layer, not a core bug.
- "The whole 0.5d item is in-repo" — REFUTED: Phase 1's target file is in the separate `ailang-packages`
  repo (absent) and gated on M-DX-XPKG-RESOLVE. Only Phase 2/3 are reachable here.
- "Add the fix to the embedded teaching prompt (Phase 3 as written)" — DECLINED (not refuted): prompt
  edits are GATED behind prompt-diet in the mission; folded the `jb`-vs-`js` teaching into the runnable
  example instead (zero prompt-budget cost, still discoverable).
- "asBoolLoose is a silent fallback (CP2 risk)" — REFUTED: it returns `Option`, `None` for non-bool
  input; the caller explicitly opts into loose-vs-strict. Structured, not silent.

**Retro lane**: **none** (no skill or process edit this iteration). Candidate watch-item (needs a 2nd
instance before routing): the inner loop assumes a sprint-planner→executor split, but a reality-check
that de-scopes an item to a sub-0.5d additive change makes a formal plan pure overhead — I executed
directly + kept the independent evaluator. If a FUTURE iteration hits the same "de-scoped to trivial,
plan is ceremony" friction, that's the ≥2-friction signal to add a mission-doc note authorizing a
plan-skip for sub-0.5d mechanical items (with independent evaluation still mandatory). One instance
so far → recorded, not yet acted on (Standing rule: max one skill edit, ≥2 frictions).

**Next**: Iteration 17 — continue clause-3. Next DOC-READY/small diagnostic: `m-dx-split-argument-warning`
(1d). Then the NEW-DOC footgun fixes (`m-dx-match-hof` R4a, `m-poly-arith-lambda` R4b,
`m-arity-style-diagnostic` R4c) each need design-doc-creator first (Conflict Surface mandatory — incl.
the error-code + mechanism verification gates). Also VERIFY-then-route: `m-dx-record-cons-pattern`,
`m-dx-tapp-trecord-unification` (may be ghosts — run the repro FIRST). PARKED for human/coordinator
(cumulative): **NEW → the M-DX-JSON-BOOL Phase-1 firestore-package fix in `ailang-packages`** (one-line
`js("true")`→`jb(b)` encoder + decoder swap to `asBoolLoose`, now a drop-in); tier-assignment
ratification (release gate); feedback-gate production ops; haiku causal re-run; scope-params
release-gate re-score; frontier-failure validation of the 8 + 4 sketches; issue #341 triage; rig A/B
for m-syntax-ai-forgiving (GPU). Carry forward: %-row + m-record-update-local-resolution doc-status
re-check; the two dev-health flakies (`PipedStdoutFlushesPerLine`; 5 effect-row example failures).

## 18 — 2026-07-12 — Iteration 17: m-dx-split-argument-warning BUILT + LANDED — compile-time non-blocking warning for the reversed-`split` footgun

**Picked**: queue `m-dx-split-argument-warning` (1d) — iteration-16's **Next** named it as the top
DOC-READY/small diagnostic in the clause-3 accessibility cluster. Existing design doc → started at
reality-check.

**Reality check**: Gate-1 origin-sync clean (local dev == origin/dev @ `c94c67417`); dev CI green
per-workflow (CI/Build/Docs all success @ c94c67417). Inbox empty; no red outranking the queue. Rebuilt
+ installed (`v0.29.2-101-gc94c67417`, `-dirty` only from a sibling's 5 pre-existing uncommitted
doc/skill edits in the main tree — NOT touched, Critical Principle 0). Verified live:
- Footgun is REAL: `split("/", name)` → `["/"]` (silently wrong); `split(name, "/")` →
  `["api","keys","user123"]`. `split` is data-first `split(s, delimiter)` — matches the doc.
- No `warn_split` pass exists — genuinely unimplemented.
- **CONFLICT SURFACE (doc premise stale)**: the doc said "hook into existing diagnostics/warning
  infrastructure", but the ONLY warning infra was exhaustiveness-specific
  (`[]*elaborate.ExhaustivenessWarning`, sourced from `elaborator.GetWarnings()`, 2 render sites calling
  `.String()`). → M1 generalized it to an `elaborate.Warning { String() string }` interface.

**Shipped**: Opus plan (sprint-planner) → sprint JSON + de-risked plan naming the Conflict Surface and
the Core detection point → Opus executor in an isolated worktree off origin/dev → independent Opus
evaluator. Three milestones, one commit each:
- **M1 `3cfcc8a6a`** — `pipeline.Result.Warnings` generalized to `[]elaborate.Warning`
  (`*ExhaustivenessWarning` already satisfies it; render sites untouched).
- **M2 `9dcaa01ce`** — extensible `swapTraps` detection pass (`internal/pipeline/warn_split_args.go`):
  walks the FINAL module Core for `App{VarGlobal{std/string.split}, [lit, non-lit]}`, emits
  `ArgOrderWarning` (hint + note) when arg0 is a 1–3-rune string literal and arg1 is not a literal.
  Module-guarded (a user-defined `split` elaborates to `core.Var`, never the guarded `VarGlobal`).
- **M3 `56e716953`** — CHANGELOG, `examples/runnable/split_argument_order.ail` (Phase-2 teaching in the
  header — embedded prompt NOT edited, prompt-diet gated), LIMITATIONS heuristic-bounds note. Plus a
  1-word evaluator-nit fix `83006dfdc` (false-negative→false-positive wording).
- **Executor deviations (all improvements, verified by the evaluator)**: (1) detection runs
  post-compile over every unit's final `unit.Core` in sorted module-ID order — NOT the plan's
  freshly-elaborated hook — because the user-code call keeps its `VarGlobal{std/string.split}` shape in
  stored Core (the `_str_split` substitution happens only INSIDE `std/string` at link time), and this
  also covers the module CACHE-HIT branch that skips elaboration. (2) `ailang check` rendered NO
  pipeline warnings at all (pre-existing gap) → added `printCheckWarnings` so `check` surfaces both the
  new warning AND exhaustiveness warnings. (3) `repl/planning.go` (listed in my reality-check as a
  render site) is a DIFFERENT `planning.ValidationResult.Warnings` type — correctly left untouched.
- **Independent Opus evaluator: round-1 PASS 97/100**, no blockers. Built its own worktree binary,
  reran the full suite (`go test ./internal/... ./cmd/...` clean, 11/11 new tests), reproduced
  non-vacuity TWO ways (arg0/arg1 flip breaks all trigger tests; adding a user-`Var` match branch makes
  the warning fire on a user `split` → proves the module-guard is load-bearing), and verified the
  non-blocking behavior live (reversed → warning on stderr + program runs, exit 0). One non-blocking
  nit (LIMITATIONS wording) fixed before merge.
- **Gate 3b**: PR #356 auto-merge on green required checks; bounded ~11-min CI poll (30-min hard
  deadline) observed all required checks pass → squash-merge `8339b6421`; origin/dev advanced
  `c94c67417..8339b6421`; post-merge dev CI green. Queue → `[LANDED]`. Design + sprint plan →
  implemented/v0_30_0. `internal/pipeline/warn_split_args.go` 215 lines (< 800).

**Routing evidence**: model=opus task-class=plan (sprint-planner produced a de-risked 3-milestone plan
that correctly surfaced the warning-generalization Conflict Surface before any code). model=opus
task-class=execute round1-quality=high (clean 3-milestone impl; the executor IMPROVED on the plan's
hook-point hypothesis via live Core inspection — exactly the "confirm, don't assume" instruction).
rounds=1 corrections=0. model=opus task-class=evaluate round1-score=97 rounds=1 (SIXTH consecutive
round-1 pass this mission). Evaluator-independence caveat holds (Opus judges Opus) — mitigated: the
evaluator rebuilt the binary and reproduced non-vacuity itself in throwaway copies; nothing rested on
the executor's report. Full inner loop (plan→execute→evaluate) used — this was a genuine 1-day task
with a real Conflict Surface, NOT the sub-0.5d de-scoped case from iter 16, so no plan-skip.

**Ruled out**:
- "There is a generic warning/diagnostics infrastructure to hook into (doc premise)" — REFUTED: the
  only one was exhaustiveness-specific; a split-arg warning REQUIRED generalizing the warning type.
- "Detect on freshly-elaborated Core (plan hypothesis)" — REFUTED by the executor's live inspection:
  that hook silently drops the warning on module cache hits; final-Core detection covers cold+warm and
  both `run`/`check`. Recorded as a plan-refinement, not a failure (the plan flagged it as a hypothesis
  to verify).
- "split resolves to the `_str_split` builtin in stored user Core" — REFUTED: the builtin substitution
  is internal to `std/string` at link time; user code retains `VarGlobal{std/string.split}`.
- "`repl/planning.go` is a warning render site for `pipeline.Result.Warnings`" — REFUTED: it's a
  different `ValidationResult.Warnings` type; no change needed there.

**Retro lane**: **none** (no skill or process edit). The iter-16 candidate watch-item (authorize a
plan-skip for sub-0.5d de-scoped mechanical items) did NOT get a 2nd instance — this iteration was a
genuine 1-day task needing a full plan, so the watch-item stays at one instance, not yet actionable.
No new ≥2-friction pattern emerged. One observation logged for future watch (needs a 2nd instance
before routing): my reality-check listed `repl/planning.go` as a render site and proposed a hook point
that were both slightly off — the executor's "confirm the actual representation, don't assume"
instruction caught both. If a future iteration's reality-check again hands the executor a wrong
code-location premise that only live inspection catches, that's the signal to add a mission-doc note
requiring the CONTROLLER to live-verify each named code location during reality-check (not just the
behavior). One instance so far.

**Next**: Iteration 18 — continue clause-3. The DOC-READY/small diagnostics are now EXHAUSTED
(module-less/xcheck/json-bool/split-arg landed iters 14–17). Recommended: **VERIFY-then-route**
m-dx-record-cons-pattern / m-dx-tapp-trecord-unification (run the doc repro FIRST — may be ghosts, a
ghost is a near-zero-cost close). Then the NEW-DOC footgun fixes (m-dx-match-hof R4a, m-poly-arith-lambda
R4b, m-arity-style-diagnostic R4c) each need design-doc-creator first (Conflict Surface mandatory).
PARKED for human/coordinator (cumulative, unchanged): M-DX-JSON-BOOL Phase-1 firestore-package fix in
`ailang-packages`; tier-assignment ratification (release gate); feedback-gate production ops; haiku
causal re-run; scope-params release-gate re-score; frontier-failure validation of the 8 + 4 sketches;
issue #341 triage; rig A/B for m-syntax-ai-forgiving (GPU). Carry forward: %-row +
m-record-update-local-resolution doc-status re-check; the two dev-health flakies
(`PipedStdoutFlushesPerLine`; 5 effect-row example failures).

## 19 — 2026-07-13 — Iteration 18: m-dx-record-cons-pattern + m-dx-tapp-trecord-unification BOTH GHOSTS — verified-closed + regression-guarded

**Picked**: the clause-3 **VERIFY-then-route** pair (`m-dx-record-cons-pattern`,
`m-dx-tapp-trecord-unification`) — iteration-17's **Next** named it as the recommended next step
("run the doc repro FIRST — may be ghosts, a ghost is a near-zero-cost close"). Both are existing
docs → started at reality-check, not design-doc-creator. Bookkeeping-double allowed under Standing
rule 1 (both are the one VERIFY-then-route batch).

**Reality check**: Gate-1 origin-sync clean (local dev == origin/dev @ `8f505e633`); dev CI green
per-workflow (CI/Build/Docs all success @ 8f505e633). Inbox: one `eval-suite` partial (19/27, 70.4%)
— routine status, not a regression/directive, ack'd, did NOT outrank the queue. Rebuilt both binaries
(`v0.29.2-106-g8f505e633`; `-dirty` only from a sibling's 5 pre-existing uncommitted doc/skill edits
in the main tree — NOT touched, Critical Principle 0). Ran each doc's repro LIVE at HEAD:
- **m-dx-record-cons-pattern = GHOST.** `{text: s, bold: b} :: rest => s` type-checks (`✓ No errors`);
  canonical `::({…}, rest)` and baseline `first :: rest` also clean. (First repro attempt FALSE-failed
  because I wrote match arms with a leading `|` — corrected against a real example: AILANG arms have
  NO `|` prefix. The doc's `PAT_INVALID_CONS` bug itself does not reproduce.)
- **m-dx-tapp-trecord-unification = GHOST.** `makeTable(rows: [[TableCell]]) -> Block` type-checks in
  both the modern `func` form AND the doc's original `let makeTable: [[TableCell]] -> Block = \rows. …`
  lambda-binding form (`✓ No errors`). The doc's cited error `cannot unify type application with
  *types.TRecord` (`%T`) was reworded to `.String()` in `5cf6287bf`; the remaining fallback
  (`unification_types.go:250`) fires only for genuinely-unrelated type apps — `unifyTypeApps`
  decomposes `TApp~TApp` and recursively unifies element types, alias-expanding the record alias.

**Shipped** (durable close, NOT just markdown — the record-cons doc flagged its OWN gap: *"Missing:
Cons with record patterns (no test)"*). One worktree off origin/dev, one commit:
- **`adde9e9d0`** (squash of `7e1881586`) — parser regression test
  `internal/parser/list_cons_pattern_test.go::TestListConsPatternWithRecord` (table: infix + canonical);
  two runnable top-level examples `examples/record_cons_pattern.ail` (→ `hello`) and
  `examples/record_list_extraction.ail` (→ `headers: 2`) — both **CI-gated** by
  `verify-examples-toplevel`; both design docs → `implemented/v0_30_0/` with resolution notes;
  CHANGELOG entry.
- **No inner-loop skills invoked** (no design-doc-creator/planner/executor/evaluator): Gate 2's
  "already done → bookkeeping" path. The deliverable is verification + a CI-enforced regression guard,
  not a feature — the full sprint machinery would be overkill for a 4-file additive change (2 examples,
  1 test func, 2 doc moves + CHANGELOG). Zero Go production-code change.
- **Local gates green in-worktree**: `internal/parser` package tests; `verify-examples-toplevel`
  (36 type-checked, 33 ran clean, incl. both new examples); `gofmt`/`go vet` clean.

**Routing evidence**: model=controller(opus) task-class=verify+bookkeeping. NO plan/execute/evaluate
roles used this iteration — it was a reality-check that resolved to two ghosts, so the routing table's
sprint roles didn't apply. No routing-policy change (this is 1 evidence row, not the ≥3 required, and
it's a null case — no sprint executed).

**Ruled out**:
- "m-dx-record-cons-pattern is an open parser bug (doc status: Planned)" — REFUTED live: record head
  in `::` type-checks in all three forms at HEAD. Ghost.
- "m-dx-tapp-trecord-unification is an open High-pri type-inference bug (doc status: Planned)" —
  REFUTED live: `[[TableCell]]` extraction type-checks in both `func` and `let`-lambda forms. Ghost.
- "The `cannot unify type application with *types.TRecord` error still guards this case" — REFUTED:
  the `%T` form was reworded (`5cf6287bf`); the surviving fallback is for unrelated type apps only.
- (Self-correction, not a code claim) "match arms take a leading `|`" — REFUTED by
  `examples/first_non_repeat.ail`: AILANG arms are `pat => expr,` with NO `|`. My first repro
  false-failed on this; re-ran correctly before concluding.

**Gate 3b**: PR #358 squash auto-merge on green required checks; bounded CI poll in two ~7-8 min
chunks (30-min hard deadline, never open-ended) observed all required checks pass → merge `adde9e9d0`;
origin/dev advanced `8f505e633..adde9e9d0`. Post-merge dev CI: the `CI` job's **Install dependencies**
step hit a transient `proxy.golang.org` flake (`golang.org/x/tools@v0.47.0` "stream ID 1;
INTERNAL_ERROR" during `make deps` — NOT code; no dependency change in this PR, parent `8f505e633`
was green). Fix-forward = one `gh run rerun --failed` → **all 3 workflows green** on `adde9e9d0`
(CI/Build/Docs). The PR's required checks had already passed pre-merge on identical content, so this
was an environmental red on the post-merge re-run only, not a real regression.

**Next**: Iteration 19 — clause-3 continues. VERIFY-then-route backlog is now EXHAUSTED (both ghosts
closed). All remaining clause-3 starters are heavier NEW-DOC footgun fixes needing design-doc-creator
first (Conflict Surface mandatory + the error-code/mechanism verification gates): **m-dx-match-hof**
(R4a, 2–3d) · **m-poly-arith-lambda** (R4b, 2–3d) · **m-arity-style-diagnostic** (R4c, 1–2d) ·
m-lambda-open-record-pattern (1d) · m-xmod-alias-poly (1–2d). Recommend R4c (arity-style, cheapest)
or R4a next. These are full inner-loop sprints (design→plan→execute→evaluate), NOT bookkeeping.
PARKED for human/coordinator (cumulative, unchanged): M-DX-JSON-BOOL Phase-1 firestore-package fix in
`ailang-packages`; tier-assignment ratification (release gate); feedback-gate production ops; haiku
causal re-run; scope-params release-gate re-score; frontier-failure validation of the 8 + 4 sketches;
issue #341 triage; rig A/B for m-syntax-ai-forgiving (GPU). Carry forward: %-row +
m-record-update-local-resolution doc-status re-check; the two dev-health flakies
(`PipedStdoutFlushesPerLine`; 5 effect-row example failures).

## 20 — 2026-07-13 — Iteration 19: versioned the never-committed iteration-13 bounded-wait safety fix (unversioned 6 iters) — clause-3 queue untouched, R4c next

**Picked**: NOT the queue's `[NEXT]` (clause-3 R4c/R4a NEW-DOC sprints). The Gate-1/Gate-2
reality-check surfaced a higher-priority **infrastructure liability that outranks feature work**
(analogous to the mission's "a red dev outranks the queue" rule): the mission's OWN iteration-13
anti-wedge remediation was never committed to git — SKILL.md Gate 3b bounded-poll rewrite +
Standing rule 6 ("Every wait is bounded"), and the `mission-control.sh` `_mc_stalled` stall
watchdog (idle-tree + long-lived-child fingerprint → early kill before HARD_TIMEOUT). Live for 6
iterations only because launchd reads the on-disk working tree; one `git checkout`/`reset`/`clean`
from permanent loss of the very mechanism that prevents a repeat of iteration-13's 4h loop-wide
wedge.

**Reality check**: Gate-1 origin-sync: local dev STALE at `8f505e633`, **2 commits behind**
origin/dev `76d15aeea` (iters 17/18 landed via #358/#359; the local ref never advanced —
iteration-12 stale-local class) → read all mission state from origin. dev CI **green per-workflow**
(CI/Build success @ `76d15aeea`, Docs @ `adde9e9d0`). Inbox: one routine `eval-suite` start
(3 local models × 3 benchmarks, 27 jobs) — not a regression/directive, did NOT outrank the queue.
Classified the 7 dirty main-tree files:
- **mission-log.md + mission.md** — uncommitted but byte-identical to origin/dev (already-merged
  iter-18 content; the local ref is just stale). Harmless.
- **SKILL.md + mission-control.sh** — the iteration-13 fix. Confirmed **never in git history**:
  `git log --all -S "Every wait is bounded" -- SKILL.md` and `-S "_mc_stalled" -- mission-control.sh`
  BOTH empty; `git show origin/dev:` greps = 0. Lives ONLY as uncommitted working-tree state.
- **3 auto-generated docs** (`docs/docs/design-docs.md` index, `docs/docs/prompts/current.md`
  v0.16.1→v0.16.2 sync, `docs/docs/roadmap/index.md`) — build-artifact drift, regenerated locally
  but never committed. Left untouched (lower-risk than the safety tooling; a future iteration
  commits them or confirms the docs build regenerates them).
- No sibling mission-control process running (only PID 82573 = this run); the other `claude`
  processes are unrelated ccd-cli desktop/remote sessions. So these are accumulated cross-iteration
  drift, NOT a sibling's active in-progress work — refuting iteration 18's assumption.

**Shipped**: worktree off origin/dev, branch `mission/iter19-bounded-wait-versioning`, **PR #360**.
- commit `de26d6a7a` — versions the exact live on-disk SKILL.md (+39) + mission-control.sh (+74);
  `bash -n` clean; **no behavioral change** (already-live content made durable + reviewable).
- commit (this entry) — iteration-19 log + mission-doc STATUS stamp (Gate 4).
- NO inner-loop skills invoked (design-doc-creator/planner/executor/evaluator): protective
  bookkeeping, not a feature. Zero Go production-code change.

**Routing evidence**: model=controller(opus) task-class=verify+bookkeeping. No plan/execute/evaluate
roles used. No routing-policy change (null case; 1 evidence row, not the ≥3 required).

**Ruled out**:
- "The iteration-13 bounded-wait fix is committed / landed somewhere" — REFUTED: absent from ALL
  of git history (`git log --all -S`) and from origin/dev; exists only as uncommitted working-tree
  state, live solely because launchd reads on-disk files.
- "The 5 non-log dirty files are a sibling agent's active in-progress work" (iteration 18's stated
  assumption) — REFUTED: no sibling mission-control process; 2 are the mission's own iter-13 fix,
  3 are auto-gen doc drift; nothing is being actively edited.
- "Local dev == origin/dev" (what the stale local ref implied at face value) — REFUTED by
  origin-sync: local is 2 commits behind.

**Gate 3b**: PR #360 — bounded 30-min CI poll; merged only on OBSERVED green required checks
(result recorded in the mission-doc STATUS + this entry updated at merge; see the iteration report).

**Retro lane**: **none** (no skill/process/backlog edit). The SKILL.md commit **versions an
EXISTING iteration-13 edit** — it is NOT a new Gate-5 skill improvement (the content was already
the running skill's text), so the "one skill edit per iteration" budget is untouched. No guardrail
change is warranted because the recurring blind spot (iters 13–18 kept re-seeing and mislabeling
these edits) is *closed by the act of committing them*, not by a new rule. Frictions cited in the
commit (≥2, same gap): (1) iteration-13's 4h wedge that burned a 6h slot; (2) the fix sitting
unversioned/at-risk for 6 iterations.

**Next**: Iteration 20 — clause-3 continues with a **full inner-loop NEW-DOC sprint**:
**m-arity-style-diagnostic** (R4c, cheapest, 1–2d) or **m-dx-match-hof** (R4a, 2–3d) via
design-doc-creator (Conflict Surface mandatory + error-code/mechanism verification gates) → planner
→ executor in worktree → evaluator. **Carry forward (new this iteration)**: the 3 uncommitted
auto-generated docs in the main tree remain unversioned drift — commit them or confirm the docs
build regenerates them. PARKED for human/coordinator (cumulative, unchanged): M-DX-JSON-BOOL Phase-1
firestore-package fix in `ailang-packages`; tier-assignment ratification (release gate);
feedback-gate production ops; haiku causal re-run; scope-params release-gate re-score;
frontier-failure validation of the 8 + 4 sketches; issue #341 triage; rig A/B for
m-syntax-ai-forgiving (GPU). Carry forward: %-row + m-record-update-local-resolution doc-status
re-check; the two dev-health flakies (`PipedStdoutFlushesPerLine`; 5 effect-row example failures).
