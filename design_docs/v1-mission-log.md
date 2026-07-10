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
