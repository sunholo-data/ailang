# Agent Bridge Explained

## How the Bridge to `.claude/agents/` Works

The bridge connects the agent protocol (message passing system) with existing Claude agents (`.claude/agents/*.md` files).

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│  Agent Protocol (Message-Based Communication)                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  .ailang/state/messages/eval-analyzer/msg_001.pending.json  │
│  {                                                           │
│    "from_agent": "cli",                                     │
│    "to_agent": "eval-analyzer",                             │
│    "message_type": "request",                               │
│    "payload": {                                             │
│      "action": "analyze_failures",                          │
│      "eval_results": "eval_results/latest.json"             │
│    }                                                         │
│  }                                                           │
│                                                              │
└──────────────────┬───────────────────────────────────────────┘
                   │
                   │ AgentRunner polls for messages
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│  AgentRunner (Polling Loop)                                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Scans .ailang/state/messages/eval-analyzer/             │
│  2. Finds msg_001.pending.json                              │
│  3. Acquires lease (crash safety)                           │
│  4. Calls handler.HandleMessage(msg)                        │
│                                                              │
└──────────────────┬───────────────────────────────────────────┘
                   │
                   │ Handler dispatches
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│  ClaudeAgentHandler (Bridge)                                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. Reads .claude/agents/eval-analyzer.md                   │
│  2. Converts message to prompt:                             │
│                                                              │
│     "You are processing a message from agent 'cli'          │
│      Message ID: msg_001                                    │
│      Payload:                                               │
│      {                                                      │
│        'action': 'analyze_failures',                        │
│        'eval_results': 'eval_results/latest.json'           │
│      }                                                       │
│                                                              │
│      Please process this message and provide response."     │
│                                                              │
│  3. Executes Claude agent with prompt                       │
│  4. Returns response                                        │
│                                                              │
└──────────────────┬───────────────────────────────────────────┘
                   │
                   │ Agent executes
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│  Claude Agent (.claude/agents/eval-analyzer.md)              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Receives prompt, uses tools to:                            │
│  - Read eval_results/latest.json                            │
│  - Analyze failures                                         │
│  - Create design_docs/planned/M-DX9-fix.md                  │
│  - Return results                                           │
│                                                              │
└──────────────────┬───────────────────────────────────────────┘
                   │
                   │ Response flows back
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│  AgentRunner sends response                                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Writes: .ailang/state/messages/cli/msg_002.pending.json    │
│  {                                                           │
│    "from_agent": "eval-analyzer",                           │
│    "to_agent": "cli",                                       │
│    "message_type": "response",                              │
│    "parent_message_id": "msg_001",                          │
│    "payload": {                                             │
│      "status": "completed",                                 │
│      "design_doc": "design_docs/planned/M-DX9-fix.md",      │
│      "failures_analyzed": 15                                │
│    }                                                         │
│  }                                                           │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## The Four Handler Types

### 1. ClaudeAgentHandler - Execute `.claude/agents/*.md`

**Purpose**: Execute a Claude agent markdown file

**Usage**:
```go
handler := agentrunner.NewClaudeAgentHandler(
    ".claude/agents/eval-analyzer.md",
    ".", // working directory
)
```

**What it does**:
1. Reads the agent markdown file
2. Converts message payload to a prompt
3. Executes the agent (via Anthropic SDK)
4. Returns the agent's output

**Current Status**: Mock implementation (returns placeholder). Full Anthropic SDK integration pending.

### 2. SkillHandler - Execute `.claude/skills/*`

**Purpose**: Execute AILANG skills with their scripts

**Usage**:
```go
handler := agentrunner.NewSkillHandler("eval-analyzer", ".")
```

**What it does**:
1. Looks for `.claude/skills/eval-analyzer/`
2. Finds scripts in `scripts/` directory
3. Executes appropriate script based on message
4. Returns script output

**Use case**: Skills that have automated scripts (like `release-manager`, `post-release`)

### 3. CommandHandler - Run Shell Commands

**Purpose**: Execute any shell command with message as input

**Usage**:
```go
handler := agentrunner.NewCommandHandler(
    "python",
    []string{"analyze_evals.py"},
    ".",
)
```

**What it does**:
1. Serializes message to JSON
2. Pipes JSON to command's stdin
3. Executes command
4. Returns stdout/stderr

**Use case**: Integrate existing scripts into agent protocol

### 4. FunctionHandler - Wrap Go Functions

**Purpose**: Use pure Go code as a handler

**Usage**:
```go
handler := agentrunner.NewFunctionHandler(func(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
    // Your logic here
    return map[string]interface{}{"status": "ok"}, nil
})
```

**What it does**: Directly calls your function

**Use case**: Custom logic in Go

---

## Real-World Example: eval-analyzer Agent

Let's build a complete example showing how to bridge `.claude/agents/eval-analyzer.md`.

### Step 1: Create the Agent Runner

```go
// examples/agents/eval_analyzer_agent.go
package main

import (
    "github.com/sunholo/ailang/internal/agentrunner"
    "github.com/sunholo/ailang/internal/agentprotocol"
)

func main() {
    // Option A: Use Claude agent file (when SDK integrated)
    handler := agentrunner.NewClaudeAgentHandler(
        ".claude/agents/eval-analyzer.md",
        ".",
    )

    // Option B: Use custom function (works now)
    handler := agentrunner.NewFunctionHandler(evalAnalyzerHandler)

    // Configure runner
    config := &agentrunner.AgentConfig{
        AgentID:  "eval-analyzer",
        StateDir: ".ailang/state",
        Handler:  handler,
    }

    runner, _ := agentrunner.NewRunner(config)
    runner.Run()
}

func evalAnalyzerHandler(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
    // 1. Read eval results
    results := readEvalResults(msg.Payload["eval_results"].(string))

    // 2. Analyze failures
    failures := analyzeFailures(results)

    // 3. Create design doc
    designDocPath := createDesignDoc(failures)

    // 4. Return response
    return map[string]interface{}{
        "status":            "completed",
        "design_doc":        designDocPath,
        "failures_analyzed": len(failures),
    }, nil
}
```

### Step 2: Send Message to eval-analyzer

```bash
# Terminal 1: Start agent
$ go run examples/agents/eval_analyzer_agent.go
[eval-analyzer] Agent runner started...
[eval-analyzer] Scanning for messages...

# Terminal 2: Send message
$ go run examples/agents/send_message.go eval-analyzer '{
  "action": "analyze_failures",
  "eval_results": "eval_results/baselines/v0.3.18/results.json"
}'

✓ Message sent to eval-analyzer
  Message ID: msg_20251023_180000_abc123
```

### Step 3: Agent Processes Message

```
# Terminal 1 output:
[eval-analyzer] Found 1 pending message(s)
[eval-analyzer] Processing message msg_20251023_180000_abc123 from cli-sender
[eval-analyzer] Reading eval results...
[eval-analyzer] Analyzing 15 failures...
[eval-analyzer] Creating design doc: design_docs/planned/M-DX9-list-builtins.md
[eval-analyzer] Sending response to cli-sender
✅ Completed message msg_20251023_180000_abc123 in 2.3s
```

### Step 4: Check Response

```bash
$ go run examples/agents/check_inbox.go cli-sender

📬 Pending messages for cli-sender (1 total):

─────────────────────────────────────────────────────────
Message #1
─────────────────────────────────────────────────────────
Message ID:     msg_20251023_180002_def456
From:           eval-analyzer
To:             cli-sender
Type:           response
Parent ID:      msg_20251023_180000_abc123
Payload:
                {
                  "status": "completed",
                  "design_doc": "design_docs/planned/M-DX9-list-builtins.md",
                  "failures_analyzed": 15
                }
```

---

## Integration with Anthropic Agent SDK

**Current Status**: The bridge is implemented but uses mock execution.

**To integrate with real Claude agents**:

```go
// In internal/agentrunner/claude_bridge.go

func (h *ClaudeAgentHandler) executeClaudeAgent(prompt string) (string, error) {
    // TODO: Replace this mock with actual Anthropic SDK call

    // Pseudo-code (actual SDK integration):
    /*
    client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))

    response, err := client.Messages.Create(context.Background(), &anthropic.MessageRequest{
        Model: "claude-3-5-sonnet-20241022",
        Messages: []anthropic.Message{
            {Role: "user", Content: prompt},
        },
        // Tools available to agent
        Tools: []anthropic.Tool{
            {Type: "read_file", ...},
            {Type: "write_file", ...},
            // ... other tools from .claude/agents/*.md
        },
    })

    return response.Content, nil
    */

    // For now, return mock
    return "Mock response from Claude agent", nil
}
```

---

## Why This Design?

### 1. Loose Coupling
- Agents don't need to know about the protocol
- Protocol doesn't need to know about agents
- Bridge connects them

### 2. Multiple Execution Modes
- Claude agents (when SDK integrated)
- Skills (existing .claude/skills/)
- Shell commands (integrate external tools)
- Go functions (pure Go logic)

### 3. Observability
- All messages are files (can inspect)
- Database tracks everything (can query)
- Logs show what happened (can debug)

### 4. Extensibility
- Add new handler types easily
- Compose handlers (chain them)
- Wrap existing agents without modification

---

## Next Steps

1. **Integrate Anthropic SDK** in `ClaudeAgentHandler.executeClaudeAgent()`
2. **Create demo agents** for each handler type
3. **Document agent contracts** (what messages they accept/return)
4. **Build orchestration** (meta-agent that coordinates others)

---

## FAQ

**Q: Why not just call Claude agents directly?**
A: The protocol provides crash recovery, deduplication, observability, and coordination across multiple agents.

**Q: Can I mix handler types?**
A: Yes! One agent can use ClaudeAgentHandler, another can use FunctionHandler, etc.

**Q: How do agents discover each other?**
A: Via the `agents` table in `.ailang/state/agents.db`. Call `db.ListActiveAgents()`.

**Q: What if an agent crashes mid-processing?**
A: The lease expires, another agent can reap it and retry.

**Q: Can agents run on different machines?**
A: Yes, as long as they share `.ailang/state/` (via NFS, S3, etc.).

**Q: How do I debug agent communication?**
A: 1) Check `.ailang/state/messages/` (files), 2) Query `.ailang/state/agents.db` (history), 3) Read logs

---

## Summary

The bridge makes it possible to:
- ✅ Use existing `.claude/agents/*.md` files with the protocol
- ✅ Execute skills with their scripts
- ✅ Integrate shell commands
- ✅ Write custom Go handlers
- ✅ All with crash recovery, deduplication, and observability

**Current limitation**: ClaudeAgentHandler needs Anthropic SDK integration (mock for now).

**Workaround**: Use FunctionHandler to implement agent logic in Go until SDK is integrated.
