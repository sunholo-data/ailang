---
paths:
  - "internal/coordinator/**"
  - "internal/executor/**"
  - "internal/server/**"
  - "cmd/ailang/coordinator*"
  - "ui/**"
---

# Coordinator & Infrastructure Rules

The coordinator daemon executes tasks autonomously using AI agents in isolated git worktrees.
Config: `~/.ailang/config.yaml`. Day-to-day operation (start/stop, delegation, approvals) is
covered by the `coordinator-helper` skill and `ailang coordinator --help`; the Collaboration Hub
UI by the `collaboration-hub` skill; execution monitoring by `ailang chains --help` (works
offline, direct SQLite).

## Multi-host workers — two gotchas

- **`worker_tags` are coordinator routing attributes; they are NOT AILANG's `--caps IO,FS`
  effect-system capabilities.** Don't conflate the two.
- Tag-routed messages need HTTP `POST /api/messages` with a `requires: ["..."]` array — the CLI
  `messages send` does not support `--requires` (SQLite-only path).

Setup and details: [docs/docs/guides/coordinator-workers.md](../../docs/docs/guides/coordinator-workers.md).

## Adding a New Executor

Claude, Codex, motoko, opencode, pi (CLI-subprocess), and managed_agents (HTTP/SSE) follow one
uniform contract. A conforming package is auto-discovered by both coordinator and eval harness
with zero changes to either — only a blank import in
`internal/coordinator/provider_executor.go` and an `agent_cli` string in
`internal/eval_harness/models.yml`. **Full contract:**
[`docs/internal/EXECUTOR_SHAPE.md`](../../docs/internal/EXECUTOR_SHAPE.md).

- `agent_cli: "gemini"` is rejected at config load (Gemini CLI retired v0.22.0; use
  `managed_agents`). Standard-mode gemini calls go through `internal/ai/gemini`, not an executor.
- Executors running in a remote sandbox without shared filesystem access must advertise
  `executor.CapRemoteSandbox`; the eval harness then bridges solution files via
  `internal/eval_harness/managed_agents_bridge.go`.

## Auditing Agent Work

After a task completes: `ailang chains view <chain-id> --spans`, then
`ailang coordinator logs <task-id>` and `coordinator diff <task-id>`. Key checks: model used
(Haiku is too weak for compiler work), turn count/cost, code changes in `internal/`, runtime vs
compile testing.

## Database Architecture

Three SQLite databases: `observatory.db` (spans/traces), `coordinator.db` (tasks/approvals),
`collaboration.db` (messages). Full reference: `docs/docs/guides/database-architecture.md`.

## TRACEPARENT Not Propagated

Claude Code does NOT propagate TRACEPARENT to subprocess environments, so child spans land in
DIFFERENT traces. Known, accepted limitation — do not try to inject TRACEPARENT, attempt runtime
fixes, or re-investigate. Workaround: `task_id`/`parent_task_id` attributes for cross-trace
linking.
