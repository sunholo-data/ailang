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
