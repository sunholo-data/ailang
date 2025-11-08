# Agent Execution Integration

**Status**: Planned
**Target**: v0.4.5
**Priority**: P0 (High) - Critical for making UI Collaboration Hub functional
**Estimated**: 15-20 hours (3-4 days)
**Dependencies**: UI Collaboration Hub (v0.4.4 - Complete)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | N/A | 0 | Infrastructure feature - no language syntax changes |
| Preserve Semantic Clarity | N/A | 0 | Infrastructure feature - no language semantics changes |
| Increase Determinism | + | +1 | Message-driven execution with reproducible traces |
| Lower Token Cost | + | +1 | Enables AI-to-AI communication without human context |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Rationale**: While this feature doesn't change AILANG language syntax, it enables deterministic agent coordination with message-based execution traces - critical for AI-first workflows.

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

The UI Collaboration Hub (v0.4.4) provides messaging infrastructure but **does not execute code**. Currently:

**Current State:**
- ✅ Messages can be sent from UI to database
- ✅ WebSocket broadcasts messages in real-time
- ✅ Approval workflow can gate capability requests
- ❌ **No component reads messages and executes directives**
- ❌ **No integration with Claude Code or AILANG runtime**
- ❌ **Results are never sent back to users**

**Impact:**
- Users can send directives like "Build a login system" but nothing happens
- The UI shows "○ Disconnected" for execution status
- The collaboration hub is a "phone system with nobody on the other end"
- Cannot demonstrate end-to-end AI agent workflows

**Metrics:**
- 0% of directives currently execute
- 100% manual intervention required after message send
- No automated agent-to-human result communication

## Goals

**Primary Goal:** Enable AILANG agents to poll messages from the collaboration hub, execute directives using Claude Code, and send results back to users.

**Success Metrics:**
- ✅ Agent polls messages every 2 seconds
- ✅ Directives execute automatically (0% manual intervention)
- ✅ Results appear in UI within 5 seconds of completion
- ✅ Approval workflow gates capability-requiring operations
- ✅ Multi-agent coordination works (2+ agents on same thread)

## Solution Design

### Overview

Create an **Agent Runtime** that:
1. Polls the message bus for new directives
2. Executes directives using Claude Code API
3. Handles approval requests for capability-gated operations
4. Sends results back to the collaboration hub
5. Supports multiple agents working on the same thread

**Integration Options** (choose one):

| Option | Pros | Cons | Estimated Effort |
|--------|------|------|------------------|
| **1. Go Agent Binary** | Full control, can embed Claude Code SDK | Requires Claude Code Go SDK (if exists) | 15-20 hours |
| **2. Headless Runner Extension** | Reuses existing `headless-runner` skill | Limited to AILANG-only execution | 10-12 hours |
| **3. Python Bridge** | Easy Claude Code API integration | Adds Python dependency | 12-15 hours |

**Recommendation**: **Option 1 (Go Agent Binary)** for maximum flexibility and performance.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        UI (Browser)                          │
│  - User sends: "Build login system"                         │
│  - WebSocket shows: "● Agent Working..."                     │
└───────────────────────────┬─────────────────────────────────┘
                            │ POST /api/messages
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Collaboration Hub (Go Server)                   │
│  - Stores message in SQLite (to_type="ailang_instance")     │
│  - Broadcasts via WebSocket                                  │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            │ Polling (every 2s)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│               Agent Runtime (cmd/ailang-agent)               │
│  1. client.PollMessages()                                    │
│  2. Execute directive via Claude Code API                    │
│  3. Capture results (code, logs, errors)                     │
│  4. client.PublishMessage(thread_id, "result", ...)          │
└───────────────────────────┬─────────────────────────────────┘
                            │ POST /api/messages (result)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Collaboration Hub (Go Server)                   │
│  - Stores result in SQLite                                   │
│  - Broadcasts to UI via WebSocket                            │
└───────────────────────────┬─────────────────────────────────┘
                            │ WebSocket: {type: "message", ...}
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                        UI (Browser)                          │
│  - Shows result in conversation                              │
│  - Updates status: "● Completed"                             │
└─────────────────────────────────────────────────────────────┘
```

**Components:**

1. **Agent Runtime (`cmd/ailang-agent/`)**: New Go binary that polls and executes
2. **Claude Code Client (`internal/claude/`)**: Wrapper around Claude Code API
3. **Capability Checker (`internal/agent/capabilities.go`)**: Checks and requests approvals
4. **Result Formatter (`internal/agent/formatter.go`)**: Formats execution results as messages
5. **Agent Manager (UI)**: Shows agent status and execution progress

### Implementation Plan

**Phase 1: Basic Agent Polling** (~5 hours)
- [ ] Create `cmd/ailang-agent/main.go` binary
- [ ] Integrate with messaging client (`internal/messaging/client.go`)
- [ ] Implement polling loop (2s interval)
- [ ] Parse directive messages and log them
- [ ] Add `ailang agent start` CLI command
- [ ] Test: Agent receives messages from UI

**Phase 2: Claude Code Integration** (~6 hours)
- [ ] Research Claude Code API (or use Claude Desktop automation)
- [ ] Create `internal/claude/client.go` wrapper
- [ ] Implement directive execution
- [ ] Capture stdout/stderr/files created
- [ ] Handle timeouts and errors
- [ ] Test: Agent executes simple directives ("create hello.txt")

**Phase 3: Result Communication** (~4 hours)
- [ ] Create `internal/agent/formatter.go` for result formatting
- [ ] Send results back to thread via `client.PublishMessage()`
- [ ] Include execution logs and artifacts
- [ ] Update UI to display results with syntax highlighting
- [ ] Test: End-to-end flow (UI → agent → result → UI)

**Phase 4: Approval Workflow** (~4 hours)
- [ ] Create `internal/agent/capabilities.go` for capability checking
- [ ] Before executing, check required capabilities (FS, Net, IO)
- [ ] If insufficient, create approval request via `client.RequestApproval()`
- [ ] Poll approval status with timeout
- [ ] Execute only after approval granted
- [ ] Test: Approval workflow blocks execution until approved

**Phase 5: Multi-Agent Coordination** (~3 hours)
- [ ] Support multiple agents on same thread
- [ ] Implement work claiming (first agent claims message)
- [ ] Add agent status broadcasting ("working", "idle", "blocked")
- [ ] Update UI to show multiple agent statuses
- [ ] Test: 2 agents on same thread, work is distributed

### Files to Modify/Create

**New files:**
- `cmd/ailang-agent/main.go` - Agent runtime entry point (~200 LOC)
- `cmd/ailang-agent/agent.go` - Core agent logic (~300 LOC)
- `internal/claude/client.go` - Claude Code API client (~250 LOC)
- `internal/agent/capabilities.go` - Capability checking (~150 LOC)
- `internal/agent/formatter.go` - Result formatting (~100 LOC)
- `cmd/ailang/agent.go` - CLI commands for agent management (~100 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Add `agent` subcommand (~20 LOC)
- `ui/src/components/MessageCenter/MessageCenter.tsx` - Show agent status (~50 LOC)
- `ui/src/components/MessageCenter/ConversationView.tsx` - Format code results (~80 LOC)
- `internal/messaging/client.go` - Add work claiming (~50 LOC)

**Total new code**: ~1,300 LOC
**Total modifications**: ~200 LOC

## Examples

### Example 1: Simple Directive Execution

**User Action in UI:**
```
Thread: "Create a CLI tool"
Message: "Create a hello.go file that prints 'Hello, World!'"
```

**Agent Processing:**
```go
// cmd/ailang-agent/agent.go
func (a *Agent) ProcessMessage(msg *messaging.Message) error {
    log.Printf("Received directive: %s", msg.Content)

    // Execute via Claude Code
    result, err := a.claude.Execute(msg.Content, ExecuteOptions{
        WorkDir: a.workDir,
        Timeout: 5 * time.Minute,
    })

    if err != nil {
        return a.sendError(msg.ThreadID, err)
    }

    // Send result back
    return a.client.PublishMessage(
        msg.ThreadID,
        "ailang_instance", a.instanceID,
        "human", "user",
        "result",
        a.formatResult(result),
    )
}
```

**Result in UI:**
```
Agent: agent1
Status: Completed
Duration: 2.3s

Created files:
  - hello.go (45 bytes)

Output:
$ go run hello.go
Hello, World!
```

### Example 2: Approval Workflow

**User Action:**
```
Message: "Fetch API documentation from https://api.example.com/docs"
```

**Agent Processing:**
```go
func (a *Agent) ProcessMessage(msg *messaging.Message) error {
    // Check required capabilities
    caps := a.capabilities.Analyze(msg.Content)
    // caps.Required = [Net], caps.Available = [IO]

    if !caps.HasAll() {
        // Request approval
        approvalID, err := a.client.RequestApproval(
            msg.ThreadID,
            &messaging.EffectDelta{
                CapType: "Net",
                Paths: []string{"https://api.example.com"},
                BudgetDelta: 1.50,
            },
            "Fetch API docs from external URL",
            "medium",
            1.50,
        )

        // Wait for approval (timeout 5 min)
        approved, err := a.client.WaitForApproval(approvalID, 5*time.Minute)
        if !approved {
            return a.sendMessage(msg.ThreadID, "status",
                "Waiting for approval to access Net capability")
        }

        // Get capability token
        token, err := a.client.GetCapabilityToken(approvalID)
        // Proceed with token
    }

    // Execute with approved capabilities
    return a.executeWithCaps(msg, caps)
}
```

**UI Flow:**
```
1. User sends directive
2. Agent requests approval (UI shows approval card)
3. User approves in Approval Queue tab
4. Agent receives token and executes
5. Result appears in Messages tab
```

## Success Criteria

- [ ] Agent polls messages every 2 seconds
- [ ] Simple directive executes and returns result ("create hello.txt")
- [ ] Code execution results include stdout/stderr/files
- [ ] Approval workflow blocks execution until granted
- [ ] Multi-agent coordination distributes work (2+ agents tested)
- [ ] UI shows agent status ("● Working", "● Idle", "⏸ Awaiting Approval")
- [ ] End-to-end latency < 10 seconds for simple directives
- [ ] Agent handles errors gracefully (syntax errors, timeouts, crashes)
- [ ] All tests passing (80%+ coverage)
- [ ] Documentation updated (README, QUICKSTART)
- [ ] Examples added (`examples/agent-workflow.md`)

## Testing Strategy

**Unit tests:**
- `internal/claude/client_test.go` - Mock Claude Code API responses
- `internal/agent/capabilities_test.go` - Capability analysis logic
- `internal/agent/formatter_test.go` - Result formatting edge cases

**Integration tests:**
- `cmd/ailang-agent/agent_test.go` - Full message → execute → result flow
- Test with real messaging.Client (in-memory SQLite)
- Mock Claude Code responses to avoid API costs

**Manual testing:**
- [ ] Start agent: `ailang agent start --instance-id agent1`
- [ ] Send directive from UI: "Create hello.go"
- [ ] Verify result appears in UI
- [ ] Send capability-requiring directive: "Fetch https://example.com"
- [ ] Approve in UI, verify execution continues
- [ ] Start 2 agents, verify work distribution

## Non-Goals

**Not in this feature:**
- Long-running task management (progress updates, cancellation) - Defer to v0.4.6
- Agent scheduling/orchestration (Kanban board) - Defer to v0.4.7
- Agent learning/memory (RAG, vector DB) - Defer to v0.5.0
- Web UI for agent configuration - Defer to v0.4.8
- Distributed agents across machines - Single-machine only for v0.4.5

## Timeline

**Week 1** (12 hours):
- Phase 1: Basic agent polling (5 hours)
- Phase 2: Claude Code integration (6 hours)
- Buffer time (1 hour)

**Week 2** (10 hours):
- Phase 3: Result communication (4 hours)
- Phase 4: Approval workflow (4 hours)
- Testing and debugging (2 hours)

**Week 3** (6 hours):
- Phase 5: Multi-agent coordination (3 hours)
- Documentation and examples (2 hours)
- Release preparation (1 hour)

**Total: ~28 hours across 3 weeks** (includes 30% buffer)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Claude Code API unavailable/undocumented | High | Fallback: Automate Claude Desktop via AppleScript or use Anthropic Messages API |
| Execution hangs/crashes agent | Medium | Implement timeouts, process isolation, automatic restart |
| Approval workflow too slow (blocks work) | Medium | Add approval queue batching, pre-approve common patterns |
| Multi-agent race conditions | Medium | Use database transactions for work claiming, add message locking |
| Results too large for UI (GB logs) | Low | Truncate at 10MB, provide "download full logs" button |

## References

- [UI Collaboration Hub Design Doc](ui-collaboration-hub.md) - Messaging infrastructure
- [UI Collaboration Hub Implementation](../../../IMPLEMENTATION_COMPARISON.md) - What was built
- [Headless Runner Skill](.claude/skills/headless-runner/SKILL.md) - Existing automation
- [Anthropic Messages API](https://docs.anthropic.com/claude/reference/messages_post) - Alternative execution backend

## Future Work

**v0.4.6 - Long-Running Task Management:**
- Progress updates for multi-step tasks
- Task cancellation from UI
- Pause/resume execution

**v0.4.7 - Orchestration Board:**
- Kanban view of tasks
- Drag-and-drop task assignment
- Agent workload visualization

**v0.4.8 - Agent Configuration UI:**
- Set agent capabilities/budget
- Configure execution environment
- Agent templates (specialist agents)

**v0.5.0 - Agent Learning:**
- RAG integration for context
- Vector DB for semantic search
- Self-improving prompts based on results

---

**Document created**: 2025-11-08
**Last updated**: 2025-11-08
