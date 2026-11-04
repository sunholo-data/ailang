# Iteration 85 — routing evidence

- Fire `fleet-1793818806-71655`, attempt 1. `MISSION_ANTHROPIC_AVAILABLE=1`, `MISSION_OVER_RATION` empty, `MISSION_ROUTING_NOTE` empty (no lane degraded this fire).
- Gate-1 base `93ac5a915b1826ddb44d39e44c9265e429110709@2026-11-04T19:01:05Z`; steady at Gate 3b (`mission-base.sh drift gate1` rc 0).
- Skill copies: the resolved symlink target's `SKILL.md` and all 13 `resources/*.md` were byte-identical to `origin/dev`, checked per file.

| Role | Resolver output | Ran | Tokens |
|---|---|---|---|
| Controller | — | `claude:claude-opus-5-5` | not reported |
| Designer | not run | Not spawned. The ticket evidence plus D-TRIAGE-36's attended ruling ("a fix should be routed as a harness ticket") is the spec, as at iteration 83. Rotation pointer not advanced (`codex:gpt-6.1-sol`). | — |
| Planner | `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin` (with the charter as doc argument) | `codex exec --sandbox read-only -C .wt-fleet-iter85` at `93ac5a915`. Background wrapper, 1800 s cap, rc file. Probe rc 0 (`ok`). Result rc 0. | 95,712 tok |
| Executor | `recipe codex:gpt-6.1-sol declared:provider-pin` | `codex exec --sandbox workspace-write --add-dir GOCACHE --add-dir GOMODCACHE`, same wrapper, rc 0. Directive forbade git writes and any read of real secrets or transcripts. | 91,919 tok |
| Evaluator | `agent-tool sonnet declared:alias-pin` | Agent tool `sonnet`, foreground, own detached judge worktree `.wt-fleet-iter85-judge` at `74daa5741` (removed afterwards) | 69,157 subagent tok, 15 tool uses, 267 s |

- Judge independence: sonnet (Anthropic) judged a codex gpt-6.1-sol generator: cross-vendor. The controller made one edit to the generated code (the public issue body, below), which the judge also reviewed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.
- No GitHub import step: this mission's queue is tickets.
- The planner ran in the build worktree, not a separate `.planner-wt-iter85`, as at iterations 81 to 83. That tree was identical to origin/dev.
