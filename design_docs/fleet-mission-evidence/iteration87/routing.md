# Iteration 87 — routing evidence

- Fire `fleet-1793871568-48811`, attempt 1. `MISSION_ANTHROPIC_AVAILABLE=1`, `MISSION_OVER_RATION=openrouter`, `MISSION_ROUTING_NOTE` empty (no lane degraded this fire).
- Gate-1 base `f6a95d75da9845c13adc9c052f81499f314ab66b@2026-11-05T09:40:15Z`.
- Skill copies: the resolved symlink target's `SKILL.md` and all 13 `resources/*.md` byte-identical to `origin/dev`, checked per file; the pin is checked out at the same `origin/dev`.

| Role | Resolver output | Ran | Tokens |
|---|---|---|---|
| Controller | — | `claude:claude-opus-5-5` | not reported |
| Designer | not resolved | Not spawned. The ticket's evidence and suggested shape plus the controller's live reproduction are the spec; the fix extends a 389-line script (precedent: iterations 83, 85, 86). Rotation pointer not advanced. | — |
| Planner | `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin` (with the charter as doc; `derive-planner-lane.sh` agrees) | `codex exec --sandbox read-only -C .planner-wt-iter87` (detached at `f6a95d75d`, removed afterwards). Background wrapper, 1800 s cap, rc file. Probe rc 0 (`ok`). Run rc 0. | 95,151 tok |
| Executor | `recipe codex:gpt-6.1-sol declared:provider-pin` | `codex exec --sandbox workspace-write --add-dir GOCACHE --add-dir GOMODCACHE -C .wt-fleet-i87-gate0-transcript`, same wrapper, rc 0, no cap hit. Directive forbade git writes and any read of real secrets or transcripts; mutation drills on scratch copies only. The `.agents` mirror sync was sandbox-denied; the controller ran it outside. | 95,832 tok |
| Evaluator | `agent-tool sonnet declared:alias-pin` | Agent tool `sonnet`, foreground, own detached judge worktree `.wt-fleet-iter87-judge` (removed) | 65,511 subagent tok, 10 tool uses, 256 s |

- Judge independence: sonnet (Anthropic) judged a codex gpt-6.1-sol generator: cross-vendor. The controller's only edit to the generated tree was regenerating the `.agents` mirror with the repo's own sync script.
- The judge wrote a 4-line summary file and returned the full verdict as its final message; the controller saved it as `evaluation.md`.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.
- No GitHub import step: this mission's queue is tickets.
