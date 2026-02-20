# M-PROVENANCE: Code Provenance Tracing & Agent Trace Export

**Status**: Planned
**Target**: v0.7.2
**Priority**: P1 - Medium
**Estimated**: 3 days (~18 hours)
**Dependencies**: None (uses existing observatory + coordinator data)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Queries existing deterministic data; export format is reproducible |
| A2: Replayability | +1 | Strengthens audit trail - links code to conversations for replay |
| A3: Effect Legibility | 0 | No effect changes; read-only queries |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Enables local verification of code origin |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | JSON export format for machine consumption; standard interop |
| A8: Minimal Syntax | +1 | No new syntax; CLI command only |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Composes with existing tracing infrastructure |
| A11: Structured Failure | 0 | Errors remain typed |
| A12: System Boundary | +1 | Makes boundary between AI generation and code explicit |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Export format designed for machine analysis

## Problem Statement

When all code in a repository is AI-generated, we need to trace **which conversation produced which code** for:
- Code review (understand the reasoning behind changes)
- Debugging (find the conversation that introduced a bug)
- Compliance (audit trail for AI-generated code)
- Interoperability (export to emerging standards like Agent Trace)

**Current State:**
- We capture tool_input for Edit/Write tools (including `file_path`, `old_string`, `new_string`)
- We link sessions to tasks via `session_id`
- We link tasks to GitHub issues via `github_issue`
- We use `git diff --name-only` for artifact discovery
- **BUT**: No unified query to traverse the full chain
- **BUT**: No export format for external tools

**Impact:**
- Developers cannot easily answer "which conversation changed this file?"
- No interoperability with code attribution standards
- Audit trail exists but is fragmented across databases

## Goals

**Primary Goal:** Enable full provenance tracing from any file/line back to the originating GitHub issue and conversation.

**Success Metrics:**
- `ailang provenance <file>` returns full chain in <500ms
- Agent Trace JSON export validates against spec
- Dashboard shows clickable provenance links
- 100% of coordinator-managed code changes are traceable

## Solution Design

### Overview

1. **CLI Command**: `ailang provenance <file>` - traces a file back to its origin
2. **Query Layer**: Cross-database joins between observatory.db and coordinator.db
3. **Agent Trace Export**: `ailang provenance --export agent-trace` outputs spec-compliant JSON
4. **Line Attribution**: Parse `git diff` with line numbers (not just `--name-only`)

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        ailang provenance                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Input: file path (or line range)                                   │
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │ observatory  │    │ coordinator  │    │   git repo   │          │
│  │     .db      │    │     .db      │    │              │          │
│  ├──────────────┤    ├──────────────┤    ├──────────────┤          │
│  │session_tools │───▶│    tasks     │───▶│  git blame   │          │
│  │  tool_input  │    │ github_issue │    │  git diff    │          │
│  │  session_id  │    │  session_id  │    │              │          │
│  └──────────────┘    └──────────────┘    └──────────────┘          │
│                                                                      │
│  Output:                                                             │
│  ├── Human-readable chain                                           │
│  ├── JSON provenance record                                         │
│  └── Agent Trace export (optional)                                  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Components:**

1. **ProvenanceQuery** (`internal/provenance/query.go`): Cross-database query engine
2. **LineAttribution** (`internal/provenance/lines.go`): Parse git diff for line ranges
3. **AgentTraceExporter** (`internal/provenance/agent_trace.go`): Export to standard format
4. **CLI Command** (`cmd/ailang/provenance.go`): User-facing interface

### Data Flow

```
File Path
    │
    ▼
┌───────────────────────────────────────────────────┐
│ 1. Find Edit/Write tools that touched this file   │
│    SELECT * FROM session_tools                    │
│    WHERE json_extract(tool_input, '$.file_path')  │
│          LIKE '%filename%'                        │
└───────────────────────────────────────────────────┘
    │
    ▼
┌───────────────────────────────────────────────────┐
│ 2. Get session → task mapping                     │
│    SELECT * FROM sessions WHERE session_id = ?    │
│    JOIN coordinator.tasks ON session_id           │
└───────────────────────────────────────────────────┘
    │
    ▼
┌───────────────────────────────────────────────────┐
│ 3. Get task → message → GitHub issue              │
│    SELECT github_issue, message_id, thread_id     │
│    FROM tasks WHERE id = ?                        │
└───────────────────────────────────────────────────┘
    │
    ▼
┌───────────────────────────────────────────────────┐
│ 4. Optional: Parse git diff for line ranges       │
│    git diff --unified=0 BASE..HEAD -- file        │
│    Extract @@ -a,b +c,d @@ hunks                  │
└───────────────────────────────────────────────────┘
    │
    ▼
Provenance Record
```

### Agent Trace Export Format

Following the [agent-trace.dev](https://agent-trace.dev) specification:

```json
{
  "version": "0.1.0",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-01-29T10:30:00Z",
  "vcs": {
    "type": "git",
    "revision": "abc123def456",
    "remote": "https://github.com/sunholo-data/ailang"
  },
  "tool": {
    "name": "ailang",
    "version": "0.7.2"
  },
  "files": [
    {
      "path": "internal/observatory/otlp_receiver.go",
      "conversations": [
        {
          "url": "ailang://task/task-29404032/session/4df60536-caed-4e2f",
          "contributor": {
            "type": "ai",
            "model_id": "anthropic/claude-opus-4-5-20251101"
          },
          "ranges": [
            {
              "start_line": 65,
              "end_line": 78,
              "content_hash": "murmur3:9f2e8a1b"
            }
          ],
          "related": [
            {
              "type": "github_issue",
              "url": "https://github.com/sunholo-data/ailang/issues/123"
            },
            {
              "type": "task",
              "url": "ailang://task/task-29404032"
            }
          ]
        }
      ]
    }
  ],
  "metadata": {
    "dev.ailang": {
      "task_id": "task-29404032",
      "session_id": "4df60536-caed-4e2f-af2c-e386c361f4e7",
      "message_id": "29404032-74b3-40c6-acc3-23d6bbe14b68",
      "thread_id": "thread_1768064360112_5f96886d"
    }
  }
}
```

### Implementation Plan

**Phase 1: Core Query Layer** (~6 hours)
- [ ] Create `internal/provenance/` package
- [ ] Implement `ProvenanceRecord` struct
- [ ] Cross-database query (observatory + coordinator)
- [ ] Unit tests for query logic

**Phase 2: CLI Command** (~4 hours)
- [ ] Add `ailang provenance <file>` command
- [ ] Human-readable output format
- [ ] JSON output (`--json` flag)
- [ ] Filter by time range (`--since`, `--until`)

**Phase 3: Line Attribution** (~4 hours)
- [ ] Parse `git diff --unified=0` for line ranges
- [ ] Compute content_hash (murmur3) for line ranges
- [ ] Map tool_input old_string/new_string to line numbers
- [ ] Handle renamed/moved files

**Phase 4: Agent Trace Export** (~4 hours)
- [ ] Implement Agent Trace JSON schema
- [ ] `ailang provenance --export agent-trace` command
- [ ] Export for single file or entire commit
- [ ] Validate against spec

### Files to Modify/Create

**New files:**
- `internal/provenance/query.go` - Cross-database provenance queries (~200 LOC)
- `internal/provenance/record.go` - ProvenanceRecord and related types (~100 LOC)
- `internal/provenance/lines.go` - Git diff parsing for line ranges (~150 LOC)
- `internal/provenance/agent_trace.go` - Agent Trace exporter (~200 LOC)
- `internal/provenance/query_test.go` - Unit tests (~300 LOC)
- `cmd/ailang/provenance.go` - CLI command (~150 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Register provenance command (~5 LOC)
- `internal/observatory/store_sessions.go` - Add file path index query (~20 LOC)

## Examples

### Example 1: Basic Provenance Query

**Command:**
```bash
$ ailang provenance internal/observatory/otlp_receiver.go
```

**Output:**
```
File: internal/observatory/otlp_receiver.go

Last modified by:
  Session: 4df60536-caed-4e2f-af2c-e386c361f4e7
  Task:    task-29404032
  Title:   "Fix session ID extraction from span attributes"
  Issue:   #123 (https://github.com/sunholo-data/ailang/issues/123)
  When:    2026-01-29 10:30:00 UTC

Tool calls that modified this file:
  1. Edit at 10:28:42 - lines 65-78 (added session ID fallback)
  2. Edit at 10:29:15 - lines 80-85 (updated comment)

View conversation: ailang coordinator logs task-29404032
```

### Example 2: JSON Output

**Command:**
```bash
$ ailang provenance --json internal/observatory/otlp_receiver.go
```

**Output:**
```json
{
  "file": "internal/observatory/otlp_receiver.go",
  "modifications": [
    {
      "session_id": "4df60536-caed-4e2f-af2c-e386c361f4e7",
      "task_id": "task-29404032",
      "task_title": "Fix session ID extraction from span attributes",
      "github_issue": 123,
      "github_url": "https://github.com/sunholo-data/ailang/issues/123",
      "timestamp": "2026-01-29T10:28:42Z",
      "tool_calls": [
        {
          "tool_name": "Edit",
          "start_line": 65,
          "end_line": 78,
          "content_hash": "murmur3:9f2e8a1b"
        }
      ]
    }
  ]
}
```

### Example 3: Agent Trace Export

**Command:**
```bash
$ ailang provenance --export agent-trace --commit HEAD > .agent-trace.json
```

**Output:** Creates spec-compliant Agent Trace JSON for all files in commit.

### Example 4: Line-Level Query

**Command:**
```bash
$ ailang provenance internal/observatory/otlp_receiver.go:65-78
```

**Output:**
```
Lines 65-78 of internal/observatory/otlp_receiver.go

Origin:
  Added in task-29404032 at 2026-01-29 10:28:42
  GitHub Issue: #123 - "Session ID not captured from Claude Code spans"

Conversation excerpt:
  User: "The session ID is coming through in span attributes, not resource attributes"
  Agent: "I'll add a fallback to check span attributes if resource attributes are empty"

Content hash: murmur3:9f2e8a1b
```

## Success Criteria

- [ ] `ailang provenance <file>` returns results for any file modified by coordinator
- [ ] JSON output matches documented schema
- [ ] Agent Trace export validates against agent-trace.dev spec
- [ ] Line ranges accurately map to git diff hunks
- [ ] Query completes in <500ms for typical files
- [ ] All tests passing
- [ ] Documentation updated with examples
- [ ] CLAUDE.md updated with provenance command

## Testing Strategy

**Unit tests:**
- ProvenanceQuery with mock databases
- Git diff parsing (various hunk formats)
- Agent Trace JSON serialization
- Line range calculation

**Integration tests:**
- Real database queries with test data
- End-to-end CLI command
- Export/import round-trip

**Manual testing:**
- Verify against actual AILANG repo
- Test with files modified by multiple sessions
- Test with renamed/moved files

## Non-Goals

**Not in this feature:**
- **Real-time tracking** - We query after-the-fact, not during execution
- **Blame integration** - We use our own data, not `git blame` (which doesn't know sessions)
- **IDE integration** - CLI only; IDE plugins are future work
- **Training data export** - Agent Trace is for attribution, not training

## Timeline

**Day 1** (6 hours):
- Phase 1: Core query layer
- Basic cross-database joins working

**Day 2** (6 hours):
- Phase 2: CLI command
- Phase 3: Line attribution
- Human-readable output

**Day 3** (6 hours):
- Phase 4: Agent Trace export
- Testing and documentation
- Integration with existing commands

**Total: ~18 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Cross-database queries slow | Medium | Add indexes on session_id, file_path; cache results |
| Git diff parsing edge cases | Low | Use well-tested diff parsing library or stick to simple format |
| Agent Trace spec changes | Low | Version lock to 0.1.0; spec is new but stable |
| Files not tracked by coordinator | Medium | Graceful fallback; show "unknown origin" for non-coordinator files |

## Related Documents

**Implemented (inform design):**
- [m-dx11-type-provenance.md](../../implemented/v0_6_0/m-dx11-type-provenance.md) - Different provenance (types vs code) but similar concept
- [artifact_discovery.go](../../../internal/coordinator/artifact_discovery.go) - Existing git diff usage

**Planned (check for overlap):**
- [m-trace-bridge-sprint-plan.md](../v0_7_1/m-trace-bridge-sprint-plan.md) - Trace infrastructure we build on

## References

- [Agent Trace Specification](https://agent-trace.dev) - The standard we export to
- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [CLAUDE.md ID Relationships](../../../CLAUDE.md#-id-relationships-across-databases) - Database schema docs

## Future Work

- **Dashboard integration**: Click file in diff viewer → show provenance
- **VS Code extension**: Hover on line → show which task created it
- **Commit hooks**: Auto-generate `.agent-trace.json` on commit
- **Import from other tools**: Parse `.agent-trace.json` from Cursor, Copilot, etc.
- **Training data export**: Export conversation+code pairs for fine-tuning

---

**Document created**: 2026-01-29
**Last updated**: 2026-01-29
