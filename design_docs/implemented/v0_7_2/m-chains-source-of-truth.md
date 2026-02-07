# M-CHAINS-SOURCE-OF-TRUTH: Make `ailang chains` the Canonical Data CLI

**Status:** IMPLEMENTED (v0.7.2)
**Date:** 2026-02-07

## Summary

Made `ailang chains` the canonical CLI for examining all past and current execution history. This involved three phases:

1. **Data Foundation** — Moved raw SQL from CLI into proper Store methods, expanded Backend interface
2. **Feature Absorption** — Enhanced existing commands with session/tool details, added stats and active subcommands
3. **API + Metrics** — Added missing REST endpoints, fixed $0.00 chain costs by wiring metric rollup

## Problem

`ailang chains` and `ailang dashboard` overlapped significantly but served different architectures:
- **chains**: Direct SQLite access, works offline, chain-centric view
- **dashboard**: HTTP client to running server, span-centric view

Chain cost/token metrics were always $0.00 because `UpdateChainMetrics()` existed in the Store but was never called from the coordinator daemon.

## Architecture Decision

**Dashboard and chains are complementary, not competing.** No deprecation.

| Aspect | `ailang chains` | `ailang dashboard` |
|--------|----------------|-------------------|
| Data source | Direct SQLite | HTTP API (requires server) |
| Abstraction | Chain/agent execution flow | Span/trace telemetry |
| Works offline | Yes | No |
| Use case | "What did agents do?" | "What providers/models were used?" |

## Phase 1: Data Foundation

### Files Created
- `internal/observatory/store_chat.go` (~150 lines) — ChatMessage Store methods

### Files Modified
- `internal/observatory/backend.go` — Added 12+ method signatures to Backend interface
- `internal/observatory/backend_sqlite.go` — Delegate methods
- `internal/observatory/backend_composite.go` — Delegate methods
- `internal/observatory/backend_gcp.go` — Stub methods (returns ErrNotImplemented)
- `internal/observatory/backend_jaeger.go` — Stub methods
- `cmd/ailang/chains_data.go` — Refactored to use Store methods instead of raw SQL

### Key Methods Added to Backend
```go
UpdateChainMetrics(ctx, id string, cost float64, tokens, turns int) error
UpdateStageApproval(ctx, stageID string, status, approvalType, feedback string) error
UpdateStageError(ctx, stageID, errorMessage string) error
GetChainByGitHubIssue(ctx, repo string, issueNumber int) (*ExecutionChain, error)
GetChainStats(ctx) (*ChainStats, error)
GetSpansByStageID(ctx, stageID string) ([]*Span, error)
LinkSpanToChain(ctx, spanID, chainID, stageID string) error
GetChatMessagesByTaskID(ctx, taskID string) ([]*ChatMessage, error)
GetChatMessagesBySession(ctx, sessionID string, start, end time.Time) ([]*ChatMessage, error)
GetSession(ctx, sessionID string) (*Session, error)
GetSessionTools(ctx, sessionID string) ([]SessionTool, error)
```

## Phase 2: Feature Absorption

### Files Created
- `cmd/ailang/chains_stats.go` (~185 lines) — `ailang chains stats` subcommand

### Files Modified
- `cmd/ailang/chains.go` — Enhanced `view --spans`, added `active` alias, updated help text
- `cmd/ailang/chains_tree.go` — Enhanced `--detailed` with tool timeline per stage

### New Commands
| Command | Description |
|---------|-------------|
| `ailang chains stats` | Cost/token aggregation with `--hours`, `--by-agent`, `--json` |
| `ailang chains active` | Convenience alias for `list --status active` |
| `ailang chains view --spans` | Shows session metadata + tool usage per stage |
| `ailang chains tree --detailed` | Shows tool timeline in tree view |

## Phase 3: API + Metrics

### Files Modified
- `internal/coordinator/daemon_tasks.go` — Added `updateStageMetrics()` and `updateChainMetrics()` helpers, called in both skip-approval and normal-approval task completion paths
- `internal/server/handlers_chains.go` — Added `GET /api/chains/stats` and `GET /api/chains/active` handlers

### Metric Rollup Fix
Root cause: `UpdateChainMetrics()` and `UpdateStageMetrics()` existed in the Store but were never called from the coordinator daemon when tasks completed. Added helper methods that are called in both task completion paths (skip-approval and normal-approval).

### New API Endpoints
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/chains/stats` | GET | Cost/token aggregation (params: `hours`, `by_agent`) |
| `/api/chains/active` | GET | Active chains filter (params: `limit`) |

### Full API Coverage
| CLI Command | API Equivalent |
|------------|----------------|
| `ailang chains list` | `GET /api/chains` |
| `ailang chains active` | `GET /api/chains/active` |
| `ailang chains view <id>` | `GET /api/chains/{id}` |
| `ailang chains stats` | `GET /api/chains/stats` |
| (by message) | `GET /api/chains/by-message/{id}` |
| (by task) | `GET /api/chains/by-task/{id}` |
| (pending) | `GET /api/chains/pending` |

## Documentation Updates
- **CLAUDE.md** — Added "Chain Execution Monitoring" section with full command reference, REST API table, enhanced "Auditing Agent Work" section
- **REST API Reference** — Added complete `/api/chains/` section with curl examples
- **coordinator-helper skill** — Added chains commands alongside dashboard commands
- **design-spec-auditor skill** — Added chains commands alongside dashboard commands
- **ID Relationships** — Added `chain_id` and `stage_id` to the ID format table

## Verification

```bash
go build ./...                              # Build passes
go vet ./...                                # Vet passes
go test ./internal/observatory/... -count=1 # Observatory tests pass
ailang chains list --json                   # Functional
ailang chains active                        # Functional
ailang chains stats --hours 720             # Functional (costs will populate on next task)
ailang chains stats --by-agent --json       # Functional
ailang chains view <id> --spans             # Shows session + tools
ailang chains tree <id> --detailed          # Shows tool timeline
ailang chains health                        # System validation
```
