# Agent System Migration Guide

**For**: AILANG developers and agent creators
**Version**: v0.3.19 (Unreleased)
**Last Updated**: October 23, 2025

---

## Quick Start (5 minutes)

If you just want to get agents running:

```bash
# 1. Build demo agents
go build -o bin/echo-agent examples/agents/echo_agent.go
go build -o bin/send-message examples/agents/send_message.go
go build -o bin/check-inbox examples/agents/check_inbox.go

# 2. Start an agent
./bin/echo-agent &

# 3. Send it a message
./bin/send-message echo-agent '{"message": "Hello, agent!"}'

# 4. Check response
./bin/check-inbox cli-sender
```

**Full tutorial**: See [docs/AGENT_TUTORIAL.md](AGENT_TUTORIAL.md) for complete walkthrough.

---

## What Changed

### Before (v0.3.18 and earlier)

AILANG had no built-in agent coordination system. If you wanted autonomous agents:
- Had to build your own message passing
- Had to implement crash recovery manually
- Had to manage state tracking yourself
- No standard protocol for agent communication

### After (v0.3.19+)

AILANG now includes a production-ready agent protocol:
- **File-based messages** in `.ailang/state/messages/`
- **SQLite database** for state tracking
- **Lease-based crash recovery** (messages never lost)
- **Agent runner** with polling loop
- **Handler bridge** to integrate `.claude/agents/`

---

## Migration Paths

### Path 1: Using Existing `.claude/agents/` Files

**If you have**: Agent markdown files in `.claude/agents/`

**What to do**: Use `ClaudeAgentHandler` to bridge them:

```go
package main

import (
    "log"
    "github.com/yourusername/ailang/internal/agentprotocol"
    "github.com/yourusername/ailang/internal/agentrunner"
)

func main() {
    handler := agentrunner.NewClaudeAgentHandler(
        ".claude/agents/eval-analyzer.md",
        ".",
    )

    runner, err := agentrunner.NewRunner(&agentrunner.AgentConfig{
        AgentID:      "eval-analyzer",
        StateDir:     ".ailang/state",
        PollInterval: 3 * time.Second,
        Handler:      handler,
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Starting eval-analyzer agent...")
    runner.Run()
}
```

**Status**: ClaudeAgentHandler is currently a mock (returns placeholder text). Full Anthropic SDK integration coming in v0.3.20 (~1-2 days).

---

### Path 2: Building Custom Go Agents

**If you have**: Custom logic you want to run as an agent

**What to do**: Use `FunctionHandler`:

```go
package main

import (
    "log"
    "time"
    "github.com/yourusername/ailang/internal/agentprotocol"
    "github.com/yourusername/ailang/internal/agentrunner"
)

func main() {
    handler := agentrunner.NewFunctionHandler(func(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
        // Your custom logic here
        action := msg.Payload["action"].(string)

        switch action {
        case "analyze":
            return analyzeData(msg)
        case "report":
            return generateReport(msg)
        default:
            return map[string]interface{}{
                "error": "unknown action",
            }, nil
        }
    })

    runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
        AgentID:      "custom-agent",
        StateDir:     ".ailang/state",
        PollInterval: 2 * time.Second,
        Handler:      handler,
    })

    runner.Run()
}

func analyzeData(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
    // Implementation
    return map[string]interface{}{"status": "analyzed"}, nil
}

func generateReport(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
    // Implementation
    return map[string]interface{}{"status": "reported"}, nil
}
```

**Examples**: See `examples/agents/echo_agent.go` and `examples/agents/eval_analyzer_agent.go`

---

### Path 3: Running Shell Commands as Agents

**If you have**: Shell scripts or commands you want to run

**What to do**: Use `CommandHandler`:

```go
handler := agentrunner.NewCommandHandler(
    "/path/to/script.sh",
    []string{"--flag", "value"},
    ".",
)

runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
    AgentID:      "script-agent",
    StateDir:     ".ailang/state",
    PollInterval: 5 * time.Second,
    Handler:      handler,
})

runner.Run()
```

**Use case**: Wrapping existing scripts, calling external tools, integration with legacy systems.

---

### Path 4: Using Skills as Agents

**If you have**: Skills in `.claude/skills/`

**What to do**: Use `SkillHandler`:

```go
handler := agentrunner.NewSkillHandler(
    "eval-analyzer",  // Skill name
    ".",              // Working directory
)

runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
    AgentID:      "eval-analyzer-skill",
    StateDir:     ".ailang/state",
    PollInterval: 3 * time.Second,
    Handler:      handler,
})

runner.Run()
```

**Status**: SkillHandler is currently a placeholder. Full skill integration coming in v0.3.20.

---

## Breaking Changes

### None (New Feature)

The agent protocol system is a **pure addition**. No existing code is affected:
- ✅ Existing CLI commands still work
- ✅ Existing REPL still works
- ✅ Existing agents (`.claude/agents/`) still work
- ✅ No changes to AILANG language syntax

### New Dependencies

**Added**: `github.com/mattn/go-sqlite3`

If you're building from source:
```bash
go get github.com/mattn/go-sqlite3
go mod tidy
```

---

## File Structure Changes

### New Directories

```
.ailang/
└── state/                          # NEW: Agent state directory
    ├── agents.db                   # SQLite database
    ├── agents.db-shm               # Shared memory (WAL mode)
    ├── agents.db-wal               # Write-ahead log
    └── messages/                   # Message files
        ├── echo-agent/
        │   └── msg_*.pending.json
        ├── eval-analyzer/
        │   └── msg_*.pending.json
        └── cli-sender/
            └── msg_*.pending.json
```

### New Example Files

```
examples/agents/                    # NEW: Agent examples
├── echo_agent.go                   # Simple echo agent
├── eval_analyzer_agent.go          # Complex agent with capabilities
├── send_message.go                 # CLI message sender
└── check_inbox.go                  # CLI inbox checker
```

### New Documentation

```
docs/
├── AGENT_TUTORIAL.md               # NEW: Step-by-step guide
├── AGENT_BRIDGE_EXPLAINED.md       # NEW: Architecture explanation
└── AGENT_MIGRATION.md              # NEW: This file

AGENT_SYSTEM_COMPLETE.md            # NEW: System overview
AGENT_SYSTEM_VALIDATION.md          # NEW: Test results
MILESTONE_1_COMPLETE.md             # NEW: Message passing milestone
MILESTONE_2_COMPLETE.md             # NEW: Database milestone
MILESTONE_3_COMPLETE.md             # NEW: Integration milestone
```

---

## Upgrade Path

### Step 1: Update Dependencies

```bash
cd /path/to/ailang
git pull
go get github.com/mattn/go-sqlite3
go mod tidy
```

### Step 2: Build New Binaries

```bash
make build
make install  # Optional: Install to system
```

### Step 3: Test Agent System

```bash
# Build demo agents
go build -o bin/echo-agent examples/agents/echo_agent.go

# Test it works
./bin/echo-agent &
sleep 3
pkill echo-agent

# Check state directory was created
ls -la .ailang/state/
```

### Step 4: Read Tutorial

Work through [docs/AGENT_TUTORIAL.md](AGENT_TUTORIAL.md) to understand the system.

### Step 5: Build Your First Agent

Copy one of the examples and customize:
```bash
cp examples/agents/echo_agent.go my_agent.go
# Edit my_agent.go to add your logic
go build -o bin/my-agent my_agent.go
./bin/my-agent
```

---

## Common Migration Scenarios

### Scenario 1: Converting a Shell Script to an Agent

**Before**:
```bash
#!/bin/bash
# analyze.sh
echo "Analyzing eval results..."
cat eval_results/latest.json | jq '.failures'
```

**After** (using CommandHandler):
```go
package main

import (
    "log"
    "time"
    "github.com/yourusername/ailang/internal/agentrunner"
)

func main() {
    handler := agentrunner.NewCommandHandler(
        "./analyze.sh",
        []string{},
        ".",
    )

    runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
        AgentID:      "analyzer",
        StateDir:     ".ailang/state",
        PollInterval: 5 * time.Second,
        Handler:      handler,
    })

    log.Println("Starting analyzer agent...")
    runner.Run()
}
```

---

### Scenario 2: Making a Claude Agent Autonomous

**Before**: Manually invoking `.claude/agents/eval-analyzer.md`

**After** (using ClaudeAgentHandler):
```go
package main

import (
    "log"
    "time"
    "github.com/yourusername/ailang/internal/agentrunner"
)

func main() {
    handler := agentrunner.NewClaudeAgentHandler(
        ".claude/agents/eval-analyzer.md",
        ".",
    )

    runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
        AgentID:      "eval-analyzer",
        StateDir:     ".ailang/state",
        PollInterval: 3 * time.Second,
        Handler:      handler,
    })

    log.Println("Starting eval-analyzer agent...")
    runner.Run()
}
```

**Note**: Full SDK integration coming in v0.3.20.

---

### Scenario 3: Coordinating Multiple Agents

**Before**: Sequential scripts, manual coordination

**After**: Agent-to-agent messaging

```go
// Agent 1: Design doc creator
package main

import (
    "log"
    "github.com/yourusername/ailang/internal/agentprotocol"
    "github.com/yourusername/ailang/internal/agentrunner"
)

func main() {
    handler := agentrunner.NewFunctionHandler(func(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
        // Create design doc
        designDoc := createDesignDoc(msg.Payload)

        // Send to sprint-planner
        writer := agentprotocol.NewMessageWriter(".ailang/state")
        writer.WriteMessage(&agentprotocol.Envelope{
            MessageType: "request",
            FromAgent:   "design-doc-creator",
            ToAgent:     "sprint-planner",
            Payload: map[string]interface{}{
                "design_doc": designDoc,
            },
        })

        return map[string]interface{}{"status": "sent_to_planner"}, nil
    })

    runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
        AgentID:      "design-doc-creator",
        StateDir:     ".ailang/state",
        PollInterval: 5 * time.Second,
        Handler:      handler,
    })

    runner.Run()
}
```

**Full workflow**: eval-analyzer → design-doc-creator → sprint-planner → sprint-executor → release-manager → post-release

---

## Testing Your Agent

### Manual Testing

```bash
# 1. Start agent in one terminal
./bin/my-agent

# 2. Send test message in another terminal
./bin/send-message my-agent '{"action": "test", "data": "hello"}'

# 3. Check response
./bin/check-inbox cli-sender

# 4. Inspect database
sqlite3 .ailang/state/agents.db "SELECT * FROM agents;"
sqlite3 .ailang/state/agents.db "SELECT * FROM messages ORDER BY created_at DESC LIMIT 5;"
```

### Integration Testing

```go
// my_agent_test.go
package main

import (
    "testing"
    "time"
    "github.com/yourusername/ailang/internal/agentprotocol"
    "github.com/yourusername/ailang/internal/agentrunner"
)

func TestMyAgent(t *testing.T) {
    // Setup
    stateDir := t.TempDir() + "/.ailang/state"

    // Create test message
    writer := agentprotocol.NewMessageWriter(stateDir)
    writer.WriteMessage(&agentprotocol.Envelope{
        MessageType: "request",
        FromAgent:   "test",
        ToAgent:     "my-agent",
        Payload: map[string]interface{}{
            "action": "test",
        },
    })

    // Run agent once
    handler := myAgentHandler()
    runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
        AgentID:      "my-agent",
        StateDir:     stateDir,
        PollInterval: 1 * time.Second,
        Handler:      handler,
    })
    runner.RunOnce()

    // Verify response
    reader := agentprotocol.NewMessageReader(stateDir)
    pending, _ := reader.ScanPendingMessages("test")
    if len(pending) != 1 {
        t.Fatalf("Expected 1 response, got %d", len(pending))
    }
}
```

---

## Troubleshooting

### Problem: Agent Not Finding Messages

**Symptom**: `[agent-id] No pending messages` repeating forever

**Check**:
1. Message file exists: `ls .ailang/state/messages/agent-id/`
2. Agent ID matches exactly (case-sensitive)
3. File extension is `.pending.json`
4. File is not empty: `cat .ailang/state/messages/agent-id/*.pending.json`

**Fix**:
```bash
# Verify message
./bin/send-message my-agent '{"test": "data"}'

# Check file was created
ls -la .ailang/state/messages/my-agent/

# Check file contents
cat .ailang/state/messages/my-agent/*.pending.json | jq .
```

---

### Problem: Database Locked

**Symptom**: `database is locked` error

**Cause**: SQLite WAL mode needs proper cleanup

**Fix**:
```bash
# Stop all agents
pkill -f bin/.*-agent

# Check for stale locks
sqlite3 .ailang/state/agents.db "SELECT * FROM agent_locks WHERE expires_at > datetime('now');"

# Reap expired leases (run this periodically)
sqlite3 .ailang/state/agents.db "DELETE FROM agent_locks WHERE expires_at <= datetime('now');"
```

---

### Problem: Messages Processed Twice

**Symptom**: Handler called multiple times for same message

**Cause**: Database deduplication not working

**Check**:
```bash
# Check if message recorded in database
sqlite3 .ailang/state/agents.db "SELECT * FROM messages WHERE message_id = 'msg_xxx';"
```

**Fix**: Ensure `db.MessageExists()` is called before processing. The runner does this automatically.

---

### Problem: Agent Crashes and Loses Work

**Symptom**: Agent dies mid-processing, message lost

**Expected Behavior**: Message should be re-processed automatically

**Check Lease Expiration**:
```bash
sqlite3 .ailang/state/agents.db "SELECT * FROM agent_locks;"
```

If lease expired, another agent (or restarted agent) will pick it up.

**Manual Recovery**:
```bash
# Reap expired leases
sqlite3 .ailang/state/agents.db "DELETE FROM agent_locks WHERE expires_at <= datetime('now');"

# Restart agent
./bin/my-agent
```

---

## Performance Tuning

### Adjust Poll Interval

**Default**: 2-3 seconds

**Trade-offs**:
- **Lower** (0.5s): Faster response, higher CPU usage
- **Higher** (10s): Lower CPU usage, slower response

```go
runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
    PollInterval: 500 * time.Millisecond,  // Fast
    // or
    PollInterval: 10 * time.Second,        // Slow
})
```

---

### Batch Processing

If processing many messages, use `RunOnce()` in a loop:

```go
for {
    err := runner.RunOnce()
    if err != nil {
        log.Printf("Error: %v", err)
    }

    // Check if more messages pending
    reader := agentprotocol.NewMessageReader(stateDir)
    pending, _ := reader.ScanPendingMessages(agentID)
    if len(pending) == 0 {
        time.Sleep(pollInterval)
    }
    // Otherwise, process immediately
}
```

---

### Database Optimization

**WAL mode** is already enabled (best for concurrency).

**Additional tuning**:
```sql
-- Increase cache size
PRAGMA cache_size = 10000;

-- Reduce fsync frequency (trade durability for speed)
PRAGMA synchronous = NORMAL;  -- Default: FULL

-- Enable memory-mapped I/O
PRAGMA mmap_size = 268435456;  -- 256MB
```

Add to `db.go` in `NewDB()`:
```go
db.Exec("PRAGMA cache_size = 10000")
db.Exec("PRAGMA synchronous = NORMAL")
```

---

## Next Steps

### Immediate (Next Release - v0.3.20)

1. **Anthropic SDK Integration** (~1-2 days)
   - ClaudeAgentHandler will actually execute `.claude/agents/*.md`
   - Full integration with Anthropic API

2. **CLI Commands** (~30 minutes)
   - `ailang agent start <agent-id>`
   - `ailang agent send <to> <payload>`
   - `ailang agent inbox <agent-id>`
   - `ailang agent db <query>`

3. **More Demo Agents** (~2 hours)
   - design-doc-creator
   - sprint-planner
   - sprint-executor

### Short-term (v0.4.0)

4. **Phase 2 Milestones**
   - Dead-letter queue (DLQ)
   - Metrics aggregation
   - HMAC signatures
   - Verification contracts

5. **Multi-Agent Workflows**
   - Workflow orchestration (DAGs)
   - Agent dependencies
   - Scheduling and priorities

### Long-term (v0.5.0+)

6. **Distributed Agents**
   - Multi-machine coordination
   - Shared state via network filesystem
   - Agent discovery across machines

7. **Self-Improvement Loop**
   - Agents report DX friction
   - Design docs auto-created
   - Sprints auto-planned and executed
   - AILANG improves itself

---

## Getting Help

**Documentation**:
- [docs/AGENT_TUTORIAL.md](AGENT_TUTORIAL.md) - Step-by-step guide
- [docs/AGENT_BRIDGE_EXPLAINED.md](AGENT_BRIDGE_EXPLAINED.md) - Architecture
- [AGENT_SYSTEM_COMPLETE.md](../AGENT_SYSTEM_COMPLETE.md) - Complete overview

**Examples**:
- [examples/agents/echo_agent.go](../examples/agents/echo_agent.go) - Simple agent
- [examples/agents/eval_analyzer_agent.go](../examples/agents/eval_analyzer_agent.go) - Complex agent

**Code**:
- [internal/agentprotocol/](../internal/agentprotocol/) - Protocol implementation
- [internal/agentrunner/](../internal/agentrunner/) - Agent runner

**Issues**: https://github.com/yourusername/ailang/issues

---

**Last Updated**: October 23, 2025
**Version**: v0.3.19 (Unreleased)
