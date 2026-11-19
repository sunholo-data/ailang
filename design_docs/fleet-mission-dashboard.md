# Fleet mission dashboard — iteration 124

Updated 2026-11-19; scheduled unattended. [Bookkeeping #2341](https://github.com/sunholo-data/ailang/issues/2341).

- **Outcome: RESOLVED as a ghost, plus a measured residual [HARNESS].** Row 65 `quorum:codex-gpt-6.2-sol-not-allowlisted`.
  - The defect was already fixed by `8e04a29c5` on 11-17. Stapledon's run used a PATH `ailang` built before that commit (the D-FLEET-29 stale-binary class).
  - A test already guards it: `TestCodexReviewerID_IsAnAllowlistedOpenAISeat` fails if the fix is reverted. Both the controller and the judge checked that by mutation.
  - Independent judge (sonnet): **RESOLVE-WITH-RESIDUAL-ESCALATED, 72**.
- **Residual, now ask D-FLEET-47: every `codex:gpt-6.2-sol` lane is dead fleet-wide.**
  - The mission codex daemon (`dev.ailang.codex-mission-owner`, `~/.codex-missions`) runs **codex-cli 0.162.1**. gpt-6.2 needs 0.169.0 or later.
  - Measured on the mission account: 0.162.1 with 6.2 returns 400 `model_not_found`; 0.162.1 with 6.1 returns OK; 0.169.0 with 6.2 returns OK.
  - Driver logs since 11-18 10:38Z: 0 of 42 `gpt-6.2-sol` probes succeeded.
  - **Recommendation A:** an attended session upgrades and restarts the daemon at an idle boundary. **B:** roll the defaults back to 6.1.
- **Dev:** green at `93e6e4604`.
- **Tickets:** 17 open (was 18). The rest are ruling-gated, out of scope, or waiting on attended steps.
- **Routing:** controller opus. Only the evaluator was spawned (sonnet, Agent tool, own worktree); nothing needed building.
- **Codex lanes:** planner falls back to opus and executor to haiku on every fire until D-FLEET-47 is answered.
- **Cost:** metered $0. Judge 79,110 tok, plus six tiny subscription codex probes.
- **Correction:** iteration 123's observation (d) was wrong. The routing-models check at HEAD already allows `gpt-6.2-sol`.
- **Records 95–124** sit on stacked PRs (… → #2422 → this fire).
- **Parked on Mark:** D-FLEET-47 (new), D-FLEET-44, D-FLEET-45 and D-FLEET-46 (unchanged).
- **Next:** #2411 once the attended `make pi-assets` sync lands. Otherwise D-42's fleet half.
- **Evidence:** [log](fleet-mission-log.md) and [iteration124/](fleet-mission-evidence/iteration124/).
