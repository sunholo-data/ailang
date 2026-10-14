# Fleet mission — iteration 37, 2026-10-14

- origin/dev `42652c19e` is still **RED** on required checks (unchanged since iteration 36):
  - `test`: `check-git-exec` on `internal/executor/managed_agents/devapi_workspace.go:237,259` (`ffef3402a`).
  - `docs-gate`: MDX error on `env-vars.md:252`, generated from `internal/config/ai.go:43`.
  - `Build windows-latest` is red too (not required).
- **Every PR to dev is blocked.** V1 owns repo reds and is disabled. → **D-FLEET-17** (recommend: an attended fix now).
- **BUILT, NOT LANDED** P2 #12 quorum salvage (#941): PR #1824, judged PASS 88. Resume: `test` + `docs-gate` green → update branch → merge → done-gate → resolve both tickets.
- **MEASURED** P2 #13 launchd-suite flakes:
  - B (gh hang): gone since filing.
  - A (OpenRouter peer): last failed 10-02.
  - Untracked timer flake: 4× on dev between 10-07 and 10-12.
  - The live ones are in `tools/eval/`, outside fleet scope → **D-FLEET-18**. The ticket stays open. Judged PASS 88 (sonnet).
- Iteration 36's and 37's records ride PR #1824, so dev's charter lags until it merges.
- Quorum's OpenAI seat is absent (API org out of credit); the reserve seat fills it.
- Next: land #1824 when dev is green, then P2 #14. The `blocking=all` ticket `agent-tool:sonnet-unavailable` stays PARKED-ON-LANE.
- Thread: #1768. OPEN human decisions: D-FLEET-16 (Phase 3b skill source, default C), D-FLEET-17 (dev red while the owner is disabled, default C), D-FLEET-18 (`tools/eval` probe authority, default C).
- Clauses: 1 unmeasured · 2 unmet · 3 met (51 open) · 4 prior evidence · 5 preserved.
