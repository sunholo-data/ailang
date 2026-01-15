# M-DASHBOARD-CLI: Dashboard API CLI Commands

**Status:** Planned
**Target:** v0.6.4
**Priority:** P1 (Medium)
**Estimated:** 1 day
**Dependencies:** Observatory backend (v0.6.0+), Inbox API (v0.5.9+)
**Created:** 2026-01-11

## Related Documents

- [design_docs/planned/v0_6_4/m-gemini-trace-investigation.md](../v0_6_4/m-gemini-trace-investigation.md) - Trace debugging
- [design_docs/implemented/v0_6_2/m-coordinator-feedback-loop-sprint-plan.md](../../implemented/v0_6_2/m-coordinator-feedback-loop-sprint-plan.md) - Coordinator APIs

## Problem Statement

**Current Pain:**
Testing dashboard API filters requires manual `curl` commands with complex URLs:

```bash
# Current workflow - verbose and error-prone
curl "http://localhost:1957/api/inbox?provider=gemini&model=gemini-2.5-flash&start_date=2026-01-10" | jq '.messages | length'
curl "http://localhost:1957/api/observatory/spans?provider=claude&status=error" | jq 'length'
curl "http://localhost:1957/api/observatory/aggregations?start_date=2026-01-01&end_date=2026-01-11" | jq
```

**Issues:**
1. Hard to remember API paths and parameter names
2. No tab completion or help
3. JSON output needs jq for readability
4. Must manually track server URL/port
5. No integration with `ailang` CLI ecosystem

## Goals

**Primary Goal:** Provide `ailang dashboard` subcommands for querying observatory and inbox APIs.

**Success Metrics:**
- Query any dashboard API in one command
- Tab completion for filter options
- Human-readable output by default (JSON with `--json`)
- Automatic server URL from config/environment

## Solution Design

### Command Structure

```
ailang dashboard
├── spans      # Query observatory spans
├── traces     # Query trace summaries
├── inbox      # Query unified inbox (messages + claude code events)
├── stats      # Query aggregation statistics
└── health     # Check server health
```

### Command: `ailang dashboard spans`

Query observatory spans with filtering.

```bash
# List recent spans
ailang dashboard spans

# Filter by provider
ailang dashboard spans --provider gemini

# Filter by model
ailang dashboard spans --model gemini-2.5-flash

# Filter by status
ailang dashboard spans --status error

# Combined filters
ailang dashboard spans --provider claude --status ok --limit 20

# Date range
ailang dashboard spans --start 2026-01-10 --end 2026-01-11

# JSON output
ailang dashboard spans --provider gemini --json

# By task ID
ailang dashboard spans --task-id task-abc123
```

**Output (default - table):**
```
ID              NAME             PROVIDER  MODEL              STATUS  DURATION
span-abc123...  claude.execute   claude    claude-sonnet-4-5  ok      45.2s
span-def456...  gemini.generate  gemini    gemini-2.5-flash   ok      12.8s
span-ghi789...  ailang.compile   -         -                  error   0.3s

Total: 3 spans
```

**Output (--json):**
```json
[
  {"id": "span-abc123", "name": "claude.execute", "provider": "claude", ...},
  ...
]
```

### Command: `ailang dashboard traces`

Query trace summaries.

```bash
# List recent traces
ailang dashboard traces

# Filter by task
ailang dashboard traces --task-id task-abc123

# Limit results
ailang dashboard traces --limit 5

# JSON output
ailang dashboard traces --json
```

**Output (default):**
```
TRACE ID        ROOT SPAN        SERVICE       SPANS  DURATION  STATUS
trace-abc123... claude.execute   claude-code   12     45.2s     ok
trace-def456... eval.benchmark   ailang-eval   8      120.5s    ok

Total: 2 traces
```

### Command: `ailang dashboard inbox`

Query unified inbox (messages + Claude Code events).

```bash
# List all events
ailang dashboard inbox

# Filter by provider (claude code events)
ailang dashboard inbox --provider gemini

# Filter by model
ailang dashboard inbox --model gemini-2.5-flash

# Filter by inbox/source type
ailang dashboard inbox --inbox coordinator

# Date range
ailang dashboard inbox --start 2026-01-10 --end 2026-01-11

# Status filter
ailang dashboard inbox --status completed

# Combined
ailang dashboard inbox --provider claude --start 2026-01-10 --limit 50 --json
```

**Output (default):**
```
ID              TYPE        FROM             TO            TITLE                    AGE
msg-abc123...   message     user             coordinator   Fix parser bug           2h ago
cc-def456...    claude_code claude-code      -             claude.execute (ok)      1h ago
cc-ghi789...    claude_code claude-code      -             gemini.generate (ok)     45m ago

Total: 3 events (1 messages, 2 claude_code)
```

### Command: `ailang dashboard stats`

Query aggregation statistics (what powers the breakdown panel).

```bash
# Get all stats
ailang dashboard stats

# Filter by date range
ailang dashboard stats --start 2026-01-01 --end 2026-01-11

# Filter by source type
ailang dashboard stats --source-type eval

# JSON output
ailang dashboard stats --json
```

**Output (default):**
```
AGGREGATION STATISTICS (2026-01-01 to 2026-01-11)

By Provider:
  claude    45 tasks   $12.34   89% success
  gemini    32 tasks    $3.21   94% success
  openai    12 tasks    $5.67   83% success

By Model:
  claude-sonnet-4-5     28 tasks   $8.90
  gemini-2.5-flash      20 tasks   $1.23
  claude-haiku-4-5      17 tasks   $3.44
  gemini-2.5-pro        12 tasks   $1.98

By Source Type:
  eval         45 tasks
  coordinator  30 tasks
  direct_api   14 tasks

Totals:
  Tasks:     89
  Cost:      $21.22
  Tokens In: 1,234,567
  Tokens Out:  456,789
```

### Command: `ailang dashboard health`

Check server health and configuration.

```bash
ailang dashboard health
```

**Output:**
```
Server:      http://localhost:1957 (running)
Database:    ~/.ailang/state/collaboration.db (245 MB)
Observatory: enabled (SQLite backend)
WebSocket:   ws://localhost:1957/ws (connected)
Uptime:      2h 34m

Recent Activity:
  Spans:     156 (last 24h)
  Messages:  23 (last 24h)
  Traces:    12 (last 24h)
```

### Server URL Configuration

**Priority order:**
1. `--server` flag: `ailang dashboard spans --server http://localhost:8080`
2. Environment variable: `AILANG_DASHBOARD_URL=http://localhost:8080`
3. Config file (`~/.ailang/config.yaml`):
   ```yaml
   dashboard:
     url: http://localhost:1957
   ```
4. Default: `http://localhost:1957`

### Implementation Plan

**Phase 1: Core Commands** (~3 hours)

- [ ] Create `cmd/ailang/dashboard.go` with subcommand structure
- [ ] Implement `spans` command with all filters
- [ ] Implement `inbox` command with all filters
- [ ] Add table formatting with `tablewriter` package

**Phase 2: Additional Commands** (~2 hours)

- [ ] Implement `traces` command
- [ ] Implement `stats` command
- [ ] Implement `health` command

**Phase 3: Polish** (~1 hour)

- [ ] Add `--json` output mode to all commands
- [ ] Add server URL configuration (flag, env, config)
- [ ] Add help text and examples to all commands
- [ ] Add bash/zsh completion hints

### Files to Create/Modify

| File | Action | LOC |
|------|--------|-----|
| `cmd/ailang/dashboard.go` | Create | ~400 |
| `cmd/ailang/dashboard_spans.go` | Create | ~150 |
| `cmd/ailang/dashboard_inbox.go` | Create | ~150 |
| `cmd/ailang/dashboard_traces.go` | Create | ~100 |
| `cmd/ailang/dashboard_stats.go` | Create | ~120 |
| `cmd/ailang/dashboard_health.go` | Create | ~80 |
| `cmd/ailang/main.go` | Modify | +10 |
| `go.mod` | Modify | +1 (tablewriter) |

**Total:** ~1,000 lines

### API Client

Create a simple HTTP client wrapper:

```go
// internal/dashboard/client.go
package dashboard

type Client struct {
    BaseURL string
    HTTP    *http.Client
}

func (c *Client) ListSpans(opts SpanListOptions) ([]Span, error)
func (c *Client) ListTraces(opts TraceListOptions) ([]TraceSummary, error)
func (c *Client) ListInbox(opts InboxListOptions) (*InboxResponse, error)
func (c *Client) GetStats(opts StatsOptions) (*AggregationStats, error)
func (c *Client) Health() (*HealthResponse, error)
```

## Examples

### Testing Filter Implementation

After implementing filters, verify with:

```bash
# Before (curl)
curl "http://localhost:1957/api/inbox?provider=gemini" | jq '.messages | length'

# After (ailang)
ailang dashboard inbox --provider gemini
# Shows: Total: 14 events

# Verify specific model
ailang dashboard inbox --model gemini-2.5-flash
# Shows: Total: 14 events (matching)

# Check spans API
ailang dashboard spans --provider claude --status error --json | jq length
# Shows: 3
```

### Debugging Workflow

```bash
# Check server is running
ailang dashboard health

# See what's in the inbox
ailang dashboard inbox --limit 10

# Investigate specific provider
ailang dashboard spans --provider gemini --limit 50

# Get aggregation breakdown
ailang dashboard stats --start 2026-01-10
```

## Success Criteria

- [ ] All 5 commands implemented with filtering support
- [ ] Table output is readable and properly formatted
- [ ] JSON output works for all commands
- [ ] Server URL configurable via flag/env/config
- [ ] Help text includes examples for each command
- [ ] Integration test verifies API connectivity

## Axiom Compliance

| Axiom | Score | Rationale |
|-------|-------|-----------|
| A7: Machines First | +1 | CLI enables scripting and automation |
| A11: Failure Is Data | +1 | Structured error responses from API |
| A3: Effect Legibility | 0 | Read-only queries, no side effects |

**Net Score: +2** (Accept)

## Timeline

| Day | Tasks |
|-----|-------|
| 1 | Phase 1 + Phase 2 |
| 2 | Phase 3 + Testing + Docs |

## Future Enhancements

- `ailang dashboard watch` - Real-time streaming of events
- `ailang dashboard export` - Export data to CSV/JSON files
- `ailang dashboard diff` - Compare two time periods
- Shell completion scripts for bash/zsh/fish
