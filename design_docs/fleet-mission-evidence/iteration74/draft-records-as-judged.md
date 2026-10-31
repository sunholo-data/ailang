# Fleet iteration 74 — draft records (as handed to the judge)

Measured on `origin/dev` = `3ca6d7e87acaf613f8c8f6659ebfe54944288d67` (Gate-1 base, 2026-10-31T01:24:04Z).
Controller: `claude:claude-opus-5-5`. Nothing built; every record below is a measurement plus a proposed
disposition. The judge's job is to refute them.

---

## Record 1 — `skills:agents-copies-stale-non-mission` (attended, 2026-10-03) → PARK, new decision row D-FLEET-38

**Ticket claim**: "27 of 40 paired `.agents/skills` SKILL.md files still differ from `.claude/skills` at
`790169359`; only mission-*/sprint-* are synced and checked. model-manager and design-doc-creator
reconciliation called V1 scope in the fleet charter."

**Measured at HEAD**:
- C1. Still real, same count: of the 40 skill names present in both trees with a `SKILL.md`, **27 differ**
  (`cmp`), **13 match**. None of the 27 is a `mission-*` or `sprint-*` skill.
- C2. The synced set is in sync (control): `tools/launchd/sync-agents-skills.sh --check` → `agents skills in
  sync (mission-control mission-brief mission-loop-change sprint-planner sprint-executor sprint-evaluator)`, rc 0.
  The script's own header says the other skills "carry .agents-only edits (model-manager, design-doc-creator)
  and need a reconciliation, not an overwrite".
- C3. Out of fleet scope: the pinned pre-push guard's `_scope_verdict fleet <path>` (functions sourced from
  `tools/launchd/githooks/pre-push` under `/bin/bash`, `type _scope_verdict` asserted first) returns
  `refuse path outside the fleet allowlist` for `.agents/skills/design-doc-creator/SKILL.md`,
  `.agents/skills/release-manager/SKILL.md`, `.agents/skills/motoko-analyzer/SKILL.md` and
  `.claude/skills/design-doc-creator/SKILL.md`; controls: `.agents/skills/mission-control/SKILL.md` and
  `tools/launchd/sync-agents-skills.sh` → allow, `internal/parser/x.go` → refuse language-core.
- C4. Shape of the drift (`drift.sh` / `norm.sh`, outputs `drift.txt` / `norm.txt` in this directory):
  after a lossy normalisation (lower-case; `.agents/`→`.claude/`; `claude code`, `claude-code`, `claude`,
  `codex` → one token), **16 of 27 are rename-only** (the 2026-07-28 "claude → Codex" rename plus path
  rewrites) and **11 carry real content divergence**: agent-inbox (4 residual lines), ailang-packages (43),
  benchmark-manager (20), cli-doc-maintainer (206; 3 files exist only in `.agents`), design-doc-creator (109;
  1 file only in `.claude`), docs-sync (22), headless-runner (2), model-manager (32), motoko-analyzer (2),
  post-release (387), release-manager (43). The divergence runs BOTH ways: e.g. `.agents/skills/design-doc-creator`
  carries a 2026-08-31 "Case study (m-mission-elo-routing …)" block absent from `.claude`, while `.claude`
  carries the "Quorum review (off-Anthropic …)" section absent from `.agents`. So a verbatim overwrite would
  delete real `.agents`-only content.
- C5. Harness relevance (why the fleet has a ticket at all): codex and pi read `.agents/skills`
  (sync script header; D-FLEET-16 measurement). The fleet's own designer fallback chain starts with
  `codex:gpt-6.1-sol` (`MISSION_DESIGNER_FALLBACK`), so a codex designer reads the `.agents` design-doc-creator,
  which lacks the quorum section. A grep of mission skills, `tools/launchd`, `missions/` and the mission
  charters for each of the 27 names finds design-doc-creator referenced in 9 files, docs-sync and post-release
  in 3, release-manager and use-ailang in 2, agent-inbox/builtin-developer/github-issue-triage/motoko-analyzer
  in 1, the other 18 in 0 (the grep is a coarse reference count, not a consumer proof).
- C6. Overlap: D-FLEET-36 (OPEN) already asks whether to widen fleet scope to the design-doc-creator skill.

**Proposed decision row D-FLEET-38 (OPEN, HD-2a scope)**: "27 non-mission skill copies still differ;
16 are rename-only, 11 carry two-way content edits, all outside fleet scope. Who reconciles them, and
should the mirror become generated for every skill?"
- **A (recommended)**: an attended (or V1) session reconciles the 11 content-divergent skills ONCE, by
  merging each `.agents`-only block into `.claude` (the source of truth); then the fleet extends
  `sync-agents-skills.sh` from the 6 mission skills to every paired skill and the guard gains
  `.agents/skills/*` as a generated mirror. The guard arm ships alone first (iteration-16 rule). The 16
  rename-only skills need no reconciliation, only the sync. If D-FLEET-36 = A lands first, design-doc-creator
  is the fleet's to reconcile.
- **B**: the fleet syncs only the 16 rename-only skills now (needs the same guard arm), and the 11 wait for
  an attended reconciliation.
- **C**: no change; codex and pi roles keep reading stale non-mission copies.
- Default if unanswered: C, ticket stays open.

**Ticket**: not resolved.

---

## Record 2 — `executor:codex-credential-collision-drops-automatic-planner-handoffs` (attended, 2026-10-12) → RESOLVE as a ghost of `79ede6ee9`

**Ticket claim**: concurrent codex jobs on DIFFERENT Cloud Run jobs (`ailang-agent-executor-codex-go` and
`ailang-agent-executor-codex`) collide on the one exclusive Codex credential, so automatic design→planner
handoffs (task-fbaaf2f8, task-e0230f3b) fail with `credential already owned`; "serialising only attended
dispatch does not fix it".

**Measured**:
- C7. Fix exists on `origin/dev`: `79ede6ee9` "fix(coordinator): serialize codex dispatch on the shared
  subscription credential" (2026-10-15T08:06Z), naming task-e0230f3b and task-fbaaf2f8 in its message. It adds
  `internal/coordinator/credential_lane.go`: `credentialLane()` returns `codex-subscription` for any dispatch
  whose provider is `codex` (unless `AuthMode == "apikey"`), independent of which Cloud Run job runs it, and
  `credentialLaneHolder()` holds a task pending while any other queued/running task occupies the lane. Wired at
  `internal/coordinator/daemon_tasks_exec.go:188`. First release containing it: `v0.54.10` (tag commit
  `74e9835ca`, 2026-10-15T11:17Z).
- C8. Live in prod: Cloud Logging (`ailang-multivac`, 25 d window) has 32 `credential lane codex-subscription
  is held by … leaving it pending` lines, all on 2026-10-16 (first 08:32:49Z), i.e. the gate held tasks
  instead of dispatching them.
- C9. Lease failures (`credential already owned`) in the same 25 d window: 17 log lines — 10 on 10-12, 2 on
  10-13, 1 on 10-15 (09:03Z, job `ailang-agent-executor-go-claude-credit`, before the release), 4 on 10-16
  (08:48–08:53Z), **0 after 2026-10-16T08:53:51Z**. Codex dispatches after that: 11 (10-17 … 10-29), with no
  gate hold logged, i.e. none was concurrent.
- C10. The 4 lines on 10-16 are NOT concurrent dispatch: the gate serialised correctly (task-6dac947e and
  task-65c96a92 were held while task-d61964fa ran). At 08:44:00Z the holder's job logged
  `execute-job: Codex credential ownership unverified; cancelling process tree: context deadline exceeded`
  and died holding the lease (`cmd/ailang/coordinator_cloud_codexauth.go:266`; the heartbeat cancels on the
  FIRST failed 10-s lease write and does not release). The coordinator then dispatched the next codex task
  (08:47:47Z) into a lease still owned by the dead job, and both following tasks failed. That is a different
  defect (a dead owner's lease blocks every later codex job until attended recovery), and it belongs to the
  sibling ticket, record 3.
- Instrument note: the coordinator task store does NOT carry this error text — the 6 ticket-named tasks are
  `failed` with an empty `error` field — so task-store greps cannot see the signature (positive control 0/6);
  Cloud Logging can (control: 10 hits on 10-12).

**Disposition**: resolve with `--sha 79ede6ee9de55510fb2988dc2d8febb6f513618f`, resolution text naming C7–C10 and
pointing the dead-owner residual at record 3 / D-FLEET-39.

---

## Record 3 — `executor:codex-credential-lease-collision-fails-concurrent-dispatch` (attended, 2026-10-12) → PARK, new decision row D-FLEET-39

**Ticket asks**: (i) the coordinator serialises codex-provider dispatch, or the job waits for the lease with a
bounded timeout, rather than failing; (ii) a lease-contention failure is retryable, not terminal; (iii) it
never needs attended recovery when the owner is alive.

**Measured**:
- Ask (i) BUILT by `79ede6ee9` (C7, C8).
- Ask (ii) NOT built: `restore()` (`coordinator_cloud_codexauth.go:134-136`) returns
  `acquire exclusive Codex credential: …` and the coordinator banks the task failed. A Cloud Run job-level
  retry can re-run the attempt, but the coordinator ignores a later completion of a task it already marked
  failed (open ticket `coordinator:job-retry-success-dropped-after-failed-completion`, not re-measured here).
  Not grepped: any coordinator-side re-queue keyed on this error (controller grep for the literal found only
  the emitter).
- Ask (iii): the measured residual is the DEAD-owner case, not the live-owner one (C10). It is by design:
  `internal/credentiallease/lease.go` — "Crashed owners remain blocked until attended recovery verifies
  termination" — and the heartbeat comment — "No owner is ever automatically replaced, including after
  heartbeat expiry". The trigger on 10-16 was one heartbeat write that missed its 10-s deadline.
- Scope: `cmd/ailang/coordinator_cloud_codexauth.go`, `internal/credentiallease/**` and
  `internal/coordinator/**` are outside the fleet allowlist (same guard as C3); changing when a rotating
  OAuth credential may be taken over is a security/credential policy call.

**Proposed decision row D-FLEET-39 (OPEN)**: "The codex lease's dead-owner case blocks every later cloud
codex job until attended recovery; a single missed 10-s heartbeat write triggered it on 10-16. Who fixes it,
and how far?"
- **A (recommended)**: an attended/V1 session makes the heartbeat tolerate transient write failures (e.g.
  cancel only after N consecutive misses within the lease's diagnostic window) and has the coordinator
  re-queue a task whose job failed at `acquire exclusive Codex credential` instead of banking it failed
  (ask ii). The no-automatic-takeover rule stays.
- **B**: A plus an automatic release when the coordinator can prove the owner's Cloud Run execution has
  terminated (execution state from the Run API), removing the attended step for crashed owners.
- **C**: no change; attended recovery after each owner death.
- Default if unanswered: C, ticket stays open.

**Ticket**: not resolved.
