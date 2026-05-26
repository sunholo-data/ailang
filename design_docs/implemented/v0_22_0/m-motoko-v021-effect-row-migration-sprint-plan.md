# Sprint Plan: M-MOTOKO-V021-EFFECT-ROW-MIGRATION

**Design doc**: [m-motoko-v021-effect-row-migration.md](m-motoko-v021-effect-row-migration.md)
**Target**: v0.22.0 (operational follow-up; no AILANG core changes)
**Status**: Approved by user (2026-05-26) — direct follow-on from M-MOTOKO-AILANG-RECONCILE's M4 discovery
**Estimated**: 2-3 hours (1-line annotation best case; small refactor worst case)
**Risk level**: Low (well-isolated to one file in motoko_agent; reverts cleanly)

## Scope decisions

| Decision | Choice |
|---|---|
| Where the fix lives | `motoko_agent/src/core/agent_loop_v2.ail` first; if that doesn't unblock, `src/core/test/stub_step.ail::dispatch_step` |
| PR branch strategy | NEW `feature/v021-effect-row-migration` from motoko_agent main (NOT folded into the already-open ai_compat PR) |
| Docs update location | THIS AILANG repo: `docs/docs/guides/coordinator-workers.md`, `internal/executor/motoko/README.md`, `changelogs/v0.10-current.md` |
| Doc PR strategy | Single commit on `dev` (no feature branch — purely docs in this repo) |

## Recent velocity

- M-COORD-MULTI-HOST-WORKERS (2026-05-25): ~3h actual vs 22h estimated
- M-MOTOKO-AILANG-RECONCILE (2026-05-26): ~45 min actual vs 2.75h estimated
- **This sprint**: even smaller scope (~3-50 LOC change in motoko_agent + ~30 LOC docs). Estimating ~1.5h actual.

## Sprint Goal

`cd ~/dev/arniwesth/motoko_agent && make build` exits 0 on AILANG v0.21+. Then `make eval-smoke MODELS=motoko-glm-5` PASS from this AILANG repo. After that, drop the "aspirational" caveat on `agent:motoko` in M-COORD-MULTI-HOST-WORKERS docs.

## Milestones

### Step 0: Confirm root cause (~30 min, 0 LOC committed)

**What:**
- [ ] Add explicit type annotation on `on_chunk` at agent_loop_v2.ail:907:
  ```ailang
  let on_chunk: (StreamChunk) -> () ! {IO} =
    \chunk. emit_stream_chunk(session_id, step_idx, stream_id, chunk);
  ```
- [ ] Run `ailang check src/core/agent_loop_v2.ail`
- [ ] If clean: annotation is the answer → proceed to M1 with that fix
- [ ] If still fails: try adding `_ = on_chunk` in the Scripted branch of `dispatch_step` in stub_step.ail
- [ ] If still fails: minimal isolated `.ail` repro for AILANG upstream issue + use a workaround (path-dep + AILANG patch, or refactor)

**Acceptance:**
- [ ] Root cause documented (which fix-shape works)
- [ ] No commits yet — this is investigation

**Files**: Edits are exploratory; revert if not adopted.

### M1: Apply the fix (~30-90 min, 3-50 LOC)

**What** (in `~/dev/arniwesth/motoko_agent/src/core/agent_loop_v2.ail`):
- [ ] Apply whichever fix Step 0 confirmed (annotation OR refactor)
- [ ] Verify `ailang check src/core/agent_loop_v2.ail` clean
- [ ] Verify `ailang check src/core/rpc.ail` clean (consumer 1)
- [ ] Verify `ailang check src/core/supervisor.ail` clean (consumer 2)
- [ ] Create branch `feature/v021-effect-row-migration` from current main
- [ ] Commit with clear scope statement

**Acceptance:**
- [ ] All 3 previously-failing files type-check
- [ ] 23/23 src/core files pass `ailang check`
- [ ] Branch pushed to `sunholo-voight-kampff/motoko_agent` fork

**Files:**
- MODIFY: `src/core/agent_loop_v2.ail` (~3 LOC for annotation, more for refactor)
- POSSIBLY: `src/core/test/stub_step.ail` if Step 0 says refactor

### M2: Full make build exits 0 (~15 min, 0 LOC unless cascade)

**What:**
- [ ] `make build` from clean state
- [ ] Should pass `verify_extensions` (already passing per M-MOTOKO-AILANG-RECONCILE)
- [ ] Should pass `check_core` (now 23/23 after M1)
- [ ] Should produce the built TUI artifact if applicable

**Acceptance:**
- [ ] `make build` exits 0
- [ ] If a new cascade emerges (another file fails), assess in-scope vs out-of-scope; small wins land in this sprint, deep ones get their own follow-up doc

### M3: end-to-end eval-smoke (~30 min)

**What:**
- [ ] `cd ~/dev/sunholo-data/ailang`
- [ ] `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"`
- [ ] Verify PASS (or document specific failure for follow-up)

**Acceptance:**
- [ ] Run completes (NOT api_error / timeout)
- [ ] motoko binary actually invoked (visible in process list during run)
- [ ] Trial result is reported (PASS or FAIL — both are progress; SKIP / api_error is not)

**Notes**: This requires OPENROUTER_API_KEY in environment for motoko-glm-5. If not present, document as deferred-verification and proceed to M4.

### M4: Graduate the worker tag (~30 min, ~30 LOC)

**What** (in THIS AILANG repo):

- [ ] `docs/docs/guides/coordinator-workers.md`: drop "aspirational" caveat on `agent:motoko` (the line near the `Aspirational (won't run yet, but advertised)` section)
- [ ] `internal/executor/motoko/README.md`: add an "Pinned motoko revision" section with the motoko_agent SHA + AILANG version floor (v0.21.0+) + verified date
- [ ] `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md`: search for any motoko aspirational language; update if found
- [ ] `changelogs/v0.10-current.md`: cumulative entry for the whole 2-sprint motoko reconciliation (ai_compat migration + effect-row fix)
- [ ] Commit + push to dev

**Acceptance:**
- [ ] Worker-tag docs updated
- [ ] Motoko executor README pins the motoko revision (commit SHA from M1)
- [ ] CHANGELOG entry covers BOTH M-MOTOKO-AILANG-RECONCILE and M-MOTOKO-V021-EFFECT-ROW-MIGRATION
- [ ] Pushed to origin/dev

## Sequencing

```
Step 0 (investigation, no commits) ──► M1 (motoko_agent: feature branch + commit)
                                            │
                                            ├──► M2 (verify make build)
                                            │
                                            └──► M3 (verify make eval-smoke)
                                                       │
                                                       └──► M4 (this AILANG repo docs)
```

Strict sequential — each milestone gates the next. If Step 0 reveals it's an AILANG upstream bug (not motoko's pattern), STOP and escalate; no point in M4 if the underlying type system needs work.

## Total estimates

| Milestone | LOC est | Hours | Test path |
|---|---:|---:|---|
| Step 0 | 0 (investigation) | 0.5 | `ailang check` on agent_loop_v2.ail with candidate fixes |
| M1 | 3-50 | 0.5-1.5 | `ailang check` on all 3 affected files |
| M2 | 0 | 0.25 | `make build` exit 0 |
| M3 | 0 | 0.5 | `make eval-smoke MODELS=motoko-glm-5` PASS |
| M4 | 30 | 0.5 | Docs render; CHANGELOG accurate |
| **Total** | **~80** | **~2** | |

## Risk management

| Risk | Likelihood | Mitigation |
|---|---|---|
| Step 0 annotation doesn't unblock — needs a refactor | Med | Plan calls out the refactor fallback explicitly; budget +1h for that case |
| make build exposes ANOTHER v0.21 type debt | Med (we already saw this in M-MOTOKO-AILANG-RECONCILE) | Time-box: 1 cascade fix is in-scope (M2 still passes); 2+ → split into separate follow-up sprint |
| eval-smoke fails because OPENROUTER_API_KEY missing in env | Med | Check `~/.zshenv` early; if missing, document as deferred-verification and ship M4 anyway |
| The root cause IS an AILANG upstream issue requiring stdlib/compiler work | Low-Med | File in this AILANG repo as a separate issue; this sprint scopes to motoko-side workaround OR escalation, not upstream fix |
| M4 docs touch files we haven't read carefully | Low | Search-and-confirm before editing each location; small surface |

## Success criteria for the whole sprint

- [ ] All M1+M2+M3+M4 acceptance criteria met
- [ ] motoko_agent: new feature branch pushed to fork with the effect-row fix
- [ ] PR-ready URL surfaced in the sprint summary for the user to open in their browser
- [ ] AILANG repo's coordinator-workers docs no longer call `agent:motoko` aspirational
- [ ] `internal/executor/motoko/README.md` pins the verified motoko commit + AILANG version floor
- [ ] Sprint design doc + plan moved from `planned/v0_22_0/` to `implemented/v0_22_0/`

## Out of scope

- Fixing AILANG's type-system effect-row inference (if Step 0 reveals it's an upstream bug)
- Republishing any motoko_ext_* packages (none should need it; this sprint is motoko_agent-side only)
- Touching the other still-open M-MOTOKO-AILANG-RECONCILE PR
- Adding new motoko features

## Day-by-day plan

This is a single-session sprint estimated under 3h. No multi-day breakdown.

---

**Sprint plan created**: 2026-05-26
**Created by**: sprint-planner skill (claude-opus-4-7)
**Approved by**: user
