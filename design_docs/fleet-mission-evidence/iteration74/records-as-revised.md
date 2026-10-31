# Fleet iteration 74 — records as revised after judge round 1

Measured on `origin/dev` = `3ca6d7e87`. Changes from [the draft](draft-records-as-judged.md) are marked **[R1]**.
Record 2 and record 3 are merged: both tickets stay open on one decision row. **[R1]**

---

## Record 1 — `skills:agents-copies-stale-non-mission` → PARK; design-doc-creator folds into D-FLEET-36, the other 26 skills go to new row D-FLEET-38

**Measured at HEAD** (unchanged unless marked):
- C1. 27 of 40 paired skills differ (`cmp`), none `mission-*`/`sprint-*`. Judge reproduced `n=40 d=27`.
- C2. `sync-agents-skills.sh --check` → in sync for its 6 mission skills, rc 0.
- C3. `_scope_verdict fleet` refuses `.agents/skills/{design-doc-creator,release-manager,motoko-analyzer}/SKILL.md`
  and `.claude/skills/design-doc-creator/SKILL.md`; controls allow `.agents/skills/mission-control/SKILL.md`
  and `tools/launchd/sync-agents-skills.sh`, refuse `internal/parser/x.go` as core.
- C4. 16 of 27 rename-only (lossy normalisation, spot-checked raw by the judge on use-ailang, eval-analyzer,
  github-issue-triage, parser-developer), 11 with two-way content divergence (list and residual counts in
  `norm.txt`). Example of `.agents`-only content: design-doc-creator's 2026-08-31 "Case study
  (m-mission-elo-routing …)" block. **[R1]** Both copies have a "Quorum review" section
  (`.agents` `:349` "OPTIONAL", `.claude` `:339` newer wording); the draft's "`.agents` lacks the quorum
  section" was wrong.
- **[R1] C4b.** The rename-only copies are not harmless: 21 non-mission `.agents` files reference
  `.Codex/skills/…` paths that do not exist (`.codex/` holds only `agents/`, `config.toml`, `hooks.json`), and
  eval-analyzer, benchmark-manager and post-release carry mangled model names (`Codex-haiku-4-5`). A verbatim
  sync from `.claude` repairs them.
- C5. Codex and pi read `.agents/skills`; the fleet's designer fallback starts at `codex:gpt-6.1-sol`, so a codex
  designer reads the older `.agents` design-doc-creator. **[R1]** The reference counts per skill (design-doc-creator
  9 files, docs-sync and post-release 3, …) are a coarse grep, not a consumer proof.
- C6. D-FLEET-36 already asks to widen fleet scope to design-doc-creator.

**[R1] Disposition**: design-doc-creator stays with D-FLEET-36 (its reconciliation is then the fleet's if that
row = A). New row D-FLEET-38 covers the other 26:
- **A (recommended)**: the fleet syncs the **16 rename-only** skills verbatim now-ish, by extending
  `sync-agents-skills.sh` to them and adding exactly those 16 `.agents/skills/<name>/` paths to
  `_scope_is_fleet_support` (not `_scope_is_harness`, which would also refuse them for product loops). The
  guard arm ships alone first (iteration-16 rule). This repairs the broken `.Codex/skills` paths. The **10**
  content-divergent skills (11 minus design-doc-creator) get an attended or V1 reconciliation, merging each
  `.agents`-only block into `.claude`; after that they join the sync list.
- **B**: one attended pass reconciles all 26 and makes `.agents` a generated mirror for every skill; no fleet
  scope change.
- **C**: no change.
- Default if unanswered: C; the ticket stays open.

**Ticket**: not resolved.

---

## Record 2+3 — `executor:codex-credential-collision-drops-automatic-planner-handoffs` and `executor:codex-credential-lease-collision-fails-concurrent-dispatch` → both stay OPEN; one new row D-FLEET-39

**Measured**:
- C7. `79ede6ee9` (2026-10-15, in `v0.54.10`) adds a provider-keyed credential lane
  (`internal/coordinator/credential_lane.go`; call site `daemon_tasks_exec.go:188`): any `codex` dispatch not on
  `AuthMode == "apikey"` occupies lane `codex-subscription`, whichever Cloud Run job runs it. Judge confirmed.
- C8. Live in prod: 32 `credential lane codex-subscription is held by … leaving it pending` lines, all
  2026-10-16 (first 08:32:49Z).
- C9. `credential already owned` lines in 25 days: 10 on 10-12, 2 on 10-13, 1 on 10-15 (before the release),
  4 on 10-16 (08:48–08:53Z), 0 after. **[R1]** The zero after 10-16 is absence of exposure (11 later codex
  dispatches, none concurrent), not evidence of a fix.
- **[R1] C10 (corrected).** The 10-16 collisions were with a LIVE owner that the gate could not see.
  Execution `ailang-agent-executor-codex-go-dgcmd`: attempt 0 was killed by the first missed heartbeat
  (08:44:00Z, `cmd/ailang/coordinator_cloud_codexauth.go` `heartbeat()` cancels on the first failed 10-s
  lease write) and published `status=failed` for task-d61964fa (08:44:03Z). Cloud Run retried it (attempt 1,
  `max_retries=1`, `internal/coordinator/completion_binding.go:55`), and the retry re-acquired the lease at
  08:44:53Z. The coordinator had already marked task-d61964fa failed; `credentialLaneHolder` counts only
  queued and running tasks, so the lane read free and task-6dac947e was dispatched at 08:47:47Z into the
  retry's lease (fails 08:48:20Z, its own retry 08:48:52Z), then task-65c96a92 at 08:52:49Z (fails 08:53:24Z,
  08:53:51Z). The retry finished and opened PR #1849 (09:10Z, merged), a result produced by a task the card
  says failed: the same shape as open ticket `coordinator:job-retry-success-dropped-after-failed-completion`.
- Ask (ii), contention retryable: not built (no re-queue keyed on the error; `ErrDispatchPermanent`,
  `daemon_tasks_exec.go:435`, covers dispatch errors only).
- Ask (iii), no attended recovery while the owner lives: the 10-16 case is exactly a live owner. The
  dead-owner case (`internal/credentiallease/lease.go`: "Crashed owners remain blocked until attended
  recovery") is by design and has no instance in the 25-day window.
- Not read from code: how attempt 0's lease was released before attempt 1 re-acquired it (`Close` calls
  `quarantine`, which would have blocked the retry). It must be checked before a fix is designed.
- Scope: `internal/coordinator/**`, `cmd/ailang/coordinator_cloud_codexauth.go`, `internal/credentiallease/**`
  are outside the fleet allowlist.

**Why not resolve the handoffs ticket [R1]**: its symptom (codex jobs dropped on `credential already owned`
while the coordinator believed the lane free) reproduced on 10-16, after the fix shipped.

**New row D-FLEET-39 (OPEN)**: "The codex credential lane (`79ede6ee9`) frees on the first failed completion
while Cloud Run's retry still owns the lease; on 10-16 that dropped two codex tasks. Three tickets share the
cause. Who fixes it?"
- **A (recommended)**: an attended or V1 session makes the coordinator not treat a task as terminal while its
  Cloud Run execution still has a retry pending (or the lane holder counts such a task), which also covers
  `coordinator:job-retry-success-dropped-after-failed-completion`; and makes a dispatch that fails at
  `acquire exclusive Codex credential` re-queue rather than bank failed (ask ii). The no-automatic-takeover
  rule stays.
- **B**: A, plus a heartbeat that tolerates N consecutive missed writes before it kills a healthy job.
- **C**: no change.
- Default if unanswered: C; both tickets stay open.
