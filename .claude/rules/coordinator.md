---
paths:
  - "internal/coordinator/**"
  - "internal/executor/**"
  - "internal/server/**"
  - "cmd/ailang/coordinator*"
  - "ui/**"
---

# Coordinator & Infrastructure Rules

## Coordinator Daemon

The coordinator executes tasks autonomously using AI agents in isolated git worktrees.

```bash
make services-start                          # Start server + coordinator
ailang coordinator status                    # Check if running
ailang messages send coordinator "Fix bug"   # Delegate a task
ailang coordinator pending                   # Review pending approvals
ailang coordinator workers list              # Show all workers (bare-metal + Cloud Run) — v0.24.0
```

**Agent workflow:** GitHub Issue → design-doc-creator → [Approval] → sprint-planner → [Approval] → sprint-executor → [Approval] → Merged

**Config**: `~/.ailang/config.yaml` | **Cloud mode**: Pub/Sub + Cloud Run (v0.9.0+) | **Multi-host workers**: tag-routed via `worker_tags` (v0.24.0+, M-COORD-MULTI-HOST-WORKERS)

## Multi-Host Workers (v0.24.0+)

Bare-metal hosts (e.g., the Mac Studio eval rig) can advertise tags so the same Pub/Sub topic carries tag-routed messages: a task tagged `requires: ollama:gemma4-26b-ailang` is claimed only by a worker advertising that tag. Set up via:

```bash
# On the worker host:
make coord-install TAGS="ollama:gemma4-26b-ailang,gpu:m4-max" HOST_ID="studio.eval-rig"

# Add the YAML block the installer prints to the agent in ~/.ailang/config.yaml.
```

Send tag-routed messages via HTTP `POST /api/messages` with a `requires: ["..."]` array (the CLI `messages send` does not yet support `--requires` — uses SQLite-only path).

**Important terminology**: `worker_tags` are routing attributes for the coordinator; they are NOT AILANG's `--caps IO,FS` effect-system capabilities. Don't conflate the two.

Full guide: [docs/docs/guides/coordinator-workers.md](../../docs/docs/guides/coordinator-workers.md).

## Adding a New Executor

Claude, Codex, motoko, opencode, pi (CLI-subprocess), and managed_agents
(HTTP/SSE — Vertex AI Managed Agents API) all follow a single uniform
contract. If a new executor package conforms, it is auto-discovered by both
the coordinator and eval harness with **zero changes to either** — only a
one-line blank import in `internal/coordinator/provider_executor.go` and an
`agent_cli` string in `internal/eval_harness/models.yml`.

**Full contract:** [`docs/internal/EXECUTOR_SHAPE.md`](../../docs/internal/EXECUTOR_SHAPE.md)

Four required elements: package layout, required symbols (`Register()` + `init()`),
coordinator wiring (blank import), and `models.yml` wiring (`agent_cli: "<name>"`).

> **Note (v0.22.0, M-MANAGED-AGENTS):** Gemini CLI was retired. `agent_cli:
> "gemini"` is rejected at config load with a clear next-step error pointing
> at `managed_agents` (Vertex AI Managed Agents API via ADC). Direct Vertex
> `generateContent` for standard-mode (single-shot) gemini calls is
> unaffected — that goes through `internal/ai/gemini`, not via an executor.
>
> For executors that run the agent in an isolated sandbox without shared
> filesystem access (currently only `managed_agents`), advertise
> `executor.CapRemoteSandbox` in `Capabilities()`. The eval harness
> recognises this flag and uses `internal/eval_harness/managed_agents_bridge.go`
> to bridge solution files via the agent's text response.

## Collaboration Hub Server

Use the `collaboration-hub` skill for development.
```bash
make services-start     # Start server + coordinator
make services-status    # Check both services
make services-stop      # Stop both services
```

## Chain Execution Monitoring

`ailang chains` is the canonical CLI for examining executions. Works offline (direct SQLite).

```bash
ailang chains list                       # List all chains
ailang chains list --agent X --since 24h # Filter by agent/time
ailang chains view <chain-id> --spans    # Full execution with sessions + tools
ailang chains tree <chain-id> --detailed # ASCII tree with tool timeline
ailang chains stats --by-agent           # Cost/token breakdown
ailang chains diagnose <chain-id>        # Quick health report
```

## Auditing Agent Work

After a coordinator task completes:
1. `ailang chains view <chain-id> --spans` — execution flow
2. `ailang coordinator logs <task-id> --limit 500` — conversation text
3. `ailang coordinator diff <task-id>` — git changes

**Key checks:** Model used (Haiku too weak for compiler), turn count/cost, code changes in `internal/`, runtime vs compile testing.

## Database Architecture

Three SQLite databases: `observatory.db` (spans/traces), `coordinator.db` (tasks/approvals), `collaboration.db` (messages).

**Full reference:** See `docs/docs/guides/database-architecture.md`

## TRACEPARENT Not Propagated

Claude Code does NOT propagate TRACEPARENT to subprocess environments. Child spans are in DIFFERENT traces. Known, accepted limitation.

**DO NOT:** Try to inject TRACEPARENT, attempt runtime fixes, or re-investigate.
**Workaround:** Use `task_id`/`parent_task_id` attributes for cross-trace linking.
