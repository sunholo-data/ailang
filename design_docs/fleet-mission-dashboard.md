# Fleet mission — iteration 38, 2026-10-14

- origin/dev `d15217d24` is still **RED** on required checks (unchanged since iteration 36):
  - `test`: `check-git-exec` on `internal/executor/managed_agents/devapi_workspace.go` (`ffef3402a`).
  - `docs-gate`: MDX error on `env-vars.md:252`, generated from `internal/config/ai.go:43`.
- **Every PR to dev is blocked.** V1 owns repo reds and is disabled. → **D-FLEET-17** (recommend: an attended fix now).
- **BUILT, NOT LANDED** (two PRs, merge in this order):
  1. P2 #12 quorum salvage (#941): PR #1824, judged PASS 88.
  2. P2 #14 quorum artifact dir: branch `fleet/i38-quorum-artifact-dir` (`b151064db`), stacked on #1824, judged PASS 90 (sonnet). The driver exports a repo-scoped `MISSION_QUORUM_ARTIFACT_DIR`, and Gate 2 reads the export, then repo-root.
- The done-gate passed at the branch heads; it must be re-run at each merge. Then resolve 3 tickets.
- Iterations 36–38 records ride these PRs, so dev's charter lags until they merge.
- Quorum's OpenAI seat is absent (API org out of credit); with `--author codex` it is benched anyway.
- Next: land #1824, then #14's PR, when dev is green; then P2 #15. `agent-tool:sonnet-unavailable` (`blocking=all`) stays PARKED-ON-LANE.
- Thread: #1768. OPEN human decisions: D-FLEET-16 (Phase 3b skill source, default C), D-FLEET-17 (dev red while the owner is disabled, default C), D-FLEET-18 (`tools/eval` probe authority, default C).
- Clauses: 1 unmeasured · 2 unmet · 3 met (51 open) · 4 prior evidence · 5 preserved.
