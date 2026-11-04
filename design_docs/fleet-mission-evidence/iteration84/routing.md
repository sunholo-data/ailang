# Iteration 84 — routing evidence

- Fire `fleet-1793793627-91456`, attempt 1. `MISSION_ANTHROPIC_AVAILABLE=1`, `MISSION_OVER_RATION=codex`.
- Driver routing note (verbatim): ``lanes degraded: ; `planner`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5` ``
- Gate-1 base `abd4dfd821e919654ddaa0fafd36dd27197126fb@2026-11-04T12:01:32Z`; steady at Gate 3b (drift rc 0).

| Role | Resolver output | Ran | Tokens |
|---|---|---|---|
| Controller | — | `claude:claude-opus-5-5` | not reported |
| Designer | not run | not spawned: deterministic regeneration, no design surface (charter CURRENT GOAL; precedent iteration 63). Rotation pointer not advanced | — |
| Planner | `agent-tool opus fail-closed:env-pin` (`derive-planner-lane.sh` on the charter: same) | Agent tool `opus`, foreground; `MISSION_PLANNER_MODEL=opus` is a bare alias, so the hook had no provider pin to enforce and the spawn was allowed | 50,487 subagent tok, 9 tool uses, 65 s |
| Executor | `recipe claude:claude-haiku-5-5 declared:provider-pin` | `claude-sub -p --model claude-haiku-5-5 --permission-mode bypassPermissions` in the sprint worktree, with a 1800 s `date +%s` cap and an rc file; probe rc 0 (`ok`); billing CLEAN | not reported (claude-sub prints no token count) |
| Evaluator | `agent-tool sonnet declared:alias-pin` | Agent tool `sonnet`, foreground, own detached judge worktree (removed afterwards) | 52,616 subagent tok, 7 tool uses, 69 s |

- Judge independence: sonnet vs a haiku generator (same vendor, different model) and an opus planner. Generator ≠ judge holds at the model level. The verdict was checked on disk: the report exists (37 lines), the judge worktree is gone, and the controller replayed one mutant.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.
