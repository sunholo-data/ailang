# Iteration 124 — independent judge (round 1)

Evaluator: sonnet, Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree
`fleet-i124-judge` at `93e6e4604` (origin/dev). 79,110 tok, 28 tool uses, 634 s. The directive
gave the ticket text and five open questions. It did not include the controller's findings or probe results.

**VERDICT: RESOLVE-WITH-RESIDUAL-ESCALATED. Score 72/100** for how well-supported the
ticket-closure case is.

## Findings (condensed from the judge's report)

- **Q1, literal defect at HEAD:** it does not reproduce. A HEAD build's `design-quorum --help` names the
  `codex:gpt-6.2-sol` default, and `AllowedOpenAIModels` contains `gpt-6.2-sol`. The fix is `8e04a29c5`
  (2026-11-17 18:55Z, about 9 h 47 m before stapledon's 04:42Z run), with fixture follow-up `55788b010`.
  Stapledon ran a stale binary. The current PATH binary (`9828cda4a`, rebuilt 11-18 10:03 local) has
  `8e04a29c5` as an ancestor (`merge-base --is-ancestor` rc 0).
- **Q2, guard:** 1 of 3 mutants killed. M1 drops `gpt-6.2-sol` from the allowlist and is **killed** by
  `TestCodexReviewerID_IsAnAllowlistedOpenAISeat` and `TestRoutingScriptAllowlistParity`. M2 sets
  `CodexReviewerModel` back to 6.1 and **survives**. M3 sets the `--help` default pool back to 6.1
  (`cmd/ailang/design_quorum.go`, outside fleet scope) and **survives**. The guard covers the
  `not-allowlisted` defect itself, but not the seat default or the help text.
- **Q3, seat on the mission path today:** it cannot be filled. `codex_reviewer.go` takes the `daemon` branch,
  and the stapledon/world envs set `CODEX_HOME=~/.codex-missions` and `AILANG_CODEX_RUNTIME=daemon`. The
  daemon is pid 2752, started 11-02, `current` → 0.162.1. Probes:
  - P1, daemon path with 6.2: 400 `model_not_found`.
  - P2, the positive control, daemon path with 6.1: `OK`.
  - P3, 0.169.0 CLI on the default home with 6.2: `OK`.
  - The driver logs show 36 identical 400s.
  - The judge did NOT run the 0.169.0-on-mission-home arm, so it left account entitlement vs daemon
    version open.
- **Q4, where the fix lives:** rig state (the mission daemon version and its restart), outside fleet
  authority, and it parks for the human. Optional in-scope follow-up: the driver probe could report
  "daemon version < 0.169.0" as its own reason.
- **Q5, disposition:** RESOLVE-WITH-RESIDUAL-ESCALATED, citing `8e04a29c5`.
- **Q6, other findings:**
  - The ticket's diagnosis was a stale binary plus a stale daemon.
  - `seating_*_test.go` fixtures still use 6.1 ids.
  - The judge did not open the stapledon artifacts.
  - Its first mutation drill ran past the shell timeout. It restored every file, and checked that.
- **Worktree:** clean at `93e6e4604`; sha256 of `rolemodels.go` matches its original. No rig service,
  `~/.codex*` file or config was modified.

## Controller checks on the judge

- **The version-isolation gap was closed first-party, before the judge ran, by a different route.** The
  controller's 2×2 on `CODEX_HOME=~/.codex-missions` used the same ChatGPT account as the missions (see
  `codex-probes.md`):
  - 0.162.1 with 6.2: 400.
  - 0.162.1 with 6.1: OK.
  - 0.169.0 with 6.2: OK.

  So account entitlement is ruled out, and the client version alone decides it.
- **The judge's token-rotation caution was checked.** The controller's in-process probes on the mission
  home ran at 08:48–08:49Z. The judge's later daemon-path 6.1 control (P2) returned OK, so the daemon's
  auth survived them. The 11-18 07:49–07:58Z refresh errors in `mission-owner.log` predate this
  iteration.
- **M1 matches what the controller found independently.** The controller ran an overlay mutation of the
  same allowlist entry, and the same two tests failed. Re-derived by the controller first and the judge
  second, without the controller's method in the directive (rule (c)).
- **M2 and M3 were weighed and left non-blocking.** Reverting the default seat to 6.1 would restore a
  working seat on 0.162.1, not this ticket's defect. M3's file is outside fleet scope.
