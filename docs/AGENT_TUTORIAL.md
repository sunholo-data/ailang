# Agent Protocol Tutorial: From Zero to Working System

This tutorial walks you through building and running autonomous agents using the AILANG agent protocol.

**Time required**: 30 minutes
**Prerequisites**: Go 1.22+, AILANG installed

---

## Tutorial Overview

We'll build a complete system with:
1. **Echo agent** - Simple agent that echoes messages
2. **Eval analyzer agent** - Analyzes evaluation failures
3. **Message sender** - CLI tool to send messages
4. **Inbox checker** - CLI tool to check responses

---

## Part 1: Understanding the System (5 minutes)

### The Big Picture

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Sender    │ ──msg──>│  Agent       │ ──resp─>│  Receiver   │
│   (CLI)     │         │  (Running)   │         │  (CLI/Agent)│
└─────────────┘         └──────────────┘         └─────────────┘
       │                       │                         │
       └───────────────────────┴─────────────────────────┘
                               │
                   .ailang/state/ (Shared State)
                   ├── messages/
                   │   ├── agent-1/*.pending.json
                   │   └── agent-2/*.pending.json
                   └── agents.db (SQLite)
```

### Key Concepts

1. **Messages are files** - `.ailang/state/messages/agent-id/*.pending.json`
2. **Agents poll** - Check for new messages every N seconds
3. **Leases prevent conflicts** - Only one agent processes a message
4. **Database tracks everything** - Full audit trail in SQLite

---

## Part 2: Setup (5 minutes)

### Step 1: Create State Directory

```bash
mkdir -p .ailang/state
```

This is where all messages and database state live.

### Step 2: Build Example Agents

```bash
# From AILANG root directory
cd examples/agents

# Build echo agent
go build -o ../../bin/echo-agent echo_agent.go

# Build eval analyzer
go build -o ../../bin/eval-analyzer eval_analyzer_agent.go

# Build utilities
go build -o ../../bin/send-message send_message.go
go build -o ../../bin/check-inbox check_inbox.go

cd ../..
```

### Step 3: Verify Setup

```bash
ls -la bin/
# Should see:
#   echo-agent
#   eval-analyzer
#   send-message
#   check-inbox
```

---

## Part 3: Running the Echo Agent (10 minutes)

### Step 1: Start the Echo Agent

Open a terminal and run:

```bash
./bin/echo-agent
```

You should see:
```
🤖 Echo Agent Starting...
   State dir: .ailang/state
   Agent ID: echo-agent
   Poll interval: 2 seconds

✓ Echo agent started. Press Ctrl+C to stop.

[echo-agent] Agent runner started (poll interval: 2s)
[echo-agent] No pending messages
```

**Keep this terminal open!**

### Step 2: Send a Message

Open a **second terminal** and run:

```bash
./bin/send-message echo-agent '{"message": "Hello, echo agent!"}'
```

You should see:
```
✓ Message sent to echo-agent
  Message ID: msg_20251023_180000_abc123
  Correlation ID: corr_xyz789
  Path: .ailang/state/messages/echo-agent/msg_20251023_180000_abc123.pending.json

To check for response:
  go run examples/agents/check_inbox.go cli-sender
```

### Step 3: Watch the Agent Process It

In the **first terminal** (echo agent), you should see:
```
[echo-agent] Found 1 pending message(s)
📨 Received message from cli-sender
   Message ID: msg_20251023_180000_abc123
   Correlation ID: corr_xyz789
   Type: request
   Payload: map[message:Hello, echo agent!]

✅ Echoing message back to cli-sender

[echo-agent] Completed message msg_20251023_180000_abc123 in 123µs
```

### Step 4: Check Your Inbox

In the **second terminal**:

```bash
./bin/check-inbox cli-sender
```

You should see:
```
📬 Pending messages for cli-sender (1 total):

─────────────────────────────────────────────────────────
Message #1
─────────────────────────────────────────────────────────
Message ID:     msg_20251023_180002_def456
From:           echo-agent
To:             cli-sender
Type:           response
Parent ID:      msg_20251023_180000_abc123
Payload:
                {
                  "echo": {
                    "message": "Hello, echo agent!"
                  },
                  "message": "Message echoed successfully",
                  "received_at": "2025-10-23T18:00:02Z"
                }
```

🎉 **Success!** You've sent a message and received a response!

### Step 5: Inspect the Files

```bash
# See all messages
ls -la .ailang/state/messages/

# See echo-agent's inbox
ls .ailang/state/messages/echo-agent/

# See your inbox
ls .ailang/state/messages/cli-sender/

# Read a message file
cat .ailang/state/messages/echo-agent/msg_*.pending.json | jq
```

The messages are just JSON files!

---

## Part 4: Running the Eval Analyzer (10 minutes)

### Step 1: Start Eval Analyzer

In a **third terminal**:

```bash
./bin/eval-analyzer
```

You should see:
```
🔍 Eval Analyzer Agent Starting...
   State dir: .ailang/state
   Agent ID: eval-analyzer
   Capabilities: analyze failures, create design docs

✓ Eval analyzer started. Press Ctrl+C to stop.

Capabilities:
  - analyze_failures: Analyze eval benchmark failures
  - report_dx_friction: Report developer experience issues
```

### Step 2: Send Analysis Request

In your **second terminal**:

```bash
./bin/send-message eval-analyzer '{
  "action": "analyze_failures",
  "eval_results": "eval_results/latest.json"
}'
```

### Step 3: Watch It Work

In the **eval-analyzer terminal** (third terminal):
```
[eval-analyzer] Found 1 pending message(s)
📨 Received message from cli-sender
   Action: analyze_failures

🔍 Analyzing eval failures...
   Reading: eval_results/latest.json
   Found 3 failures
   Creating design doc: design_docs/planned/M-DX9-fix-eval-failures.md
✅ Design doc created
[eval-analyzer] Completed message msg_... in 823ms
```

### Step 4: Check the Response

```bash
./bin/check-inbox cli-sender
```

You'll see:
```
Payload:
                {
                  "design_doc": "design_docs/planned/M-DX9-fix-eval-failures.md",
                  "design_doc_content": "# M-DX9: Fix Eval Failures\n\n...",
                  "failures": [
                    {
                      "benchmark": "list_comprehension",
                      "error": "missing builtin: list.map",
                      "priority": "high"
                    },
                    ...
                  ],
                  "failures_analyzed": 3,
                  "high_priority": 2,
                  "medium_priority": 1,
                  "recommendation": "Implement high-priority fixes first",
                  "status": "completed"
                }
```

🎉 **Success!** The eval-analyzer processed your request and created a design doc!

---

## Part 5: Agent-to-Agent Communication

### Scenario: eval-analyzer → design-doc-creator → sprint-planner

Let's simulate multiple agents working together.

### Step 1: Create a Test Scenario

```bash
# eval-analyzer sends to design-doc-creator
./bin/send-message design-doc-creator '{
  "action": "create_design_doc",
  "from_agent": "eval-analyzer",
  "failures": ["list_comprehension", "imports"],
  "priority": "high"
}'

# design-doc-creator sends to sprint-planner
./bin/send-message sprint-planner '{
  "action": "create_sprint",
  "from_agent": "design-doc-creator",
  "design_doc": "design_docs/planned/M-DX9.md",
  "estimated_days": 1.5
}'
```

### Step 2: Check Inboxes

```bash
# Check design-doc-creator inbox
./bin/check-inbox design-doc-creator

# Check sprint-planner inbox
./bin/check-inbox sprint-planner
```

Even though the agents aren't running, you can see the messages waiting!

---

## Part 6: Database Inspection

### View Agent Registry

```bash
sqlite3 .ailang/state/agents.db

sqlite> SELECT agent_id, status, last_heartbeat FROM agents;
```

Output:
```
echo-agent|active|2025-10-23 18:05:30
eval-analyzer|active|2025-10-23 18:05:28
```

### View Message History

```sqlite
SELECT
    message_id,
    from_agent,
    to_agent,
    status,
    created_at
FROM messages
ORDER BY created_at DESC
LIMIT 10;
```

### View Leases

```sqlite
SELECT
    resource_id,
    locked_by,
    locked_at,
    expires_at
FROM agent_locks;
```

### View Metrics

```sqlite
SELECT
    agent_id,
    metric_name,
    AVG(metric_value) as avg_value,
    COUNT(*) as count
FROM agent_metrics
GROUP BY agent_id, metric_name;
```

---

## Part 7: Crash Recovery Demo

### Scenario: Agent crashes while processing

### Step 1: Send a Message

```bash
./bin/send-message echo-agent '{"test": "crash-recovery"}'
```

### Step 2: Kill the Agent Mid-Processing

In the echo-agent terminal, press `Ctrl+C` **immediately** after it starts processing.

### Step 3: Check the Database

```bash
sqlite3 .ailang/state/agents.db

sqlite> SELECT * FROM agent_locks;
```

You'll see the message is locked!

### Step 4: Wait for Lease to Expire

The lease is 60 seconds by default. After 60 seconds, the lease expires.

### Step 5: Restart the Agent

```bash
./bin/echo-agent
```

After 60 seconds, the agent will:
1. Find the expired lease
2. Reap it
3. Process the message
4. Complete successfully

🎉 **Crash recovery works!**

---

## Part 8: Building Your Own Agent

### Template

```go
package main

import (
    "log"
    "github.com/sunholo/ailang/internal/agentrunner"
    "github.com/sunholo/ailang/internal/agentprotocol"
)

func main() {
    handler := agentrunner.NewFunctionHandler(myHandler)

    runner, _ := agentrunner.NewRunner(&agentrunner.AgentConfig{
        AgentID:  "my-agent",
        StateDir: ".ailang/state",
        Handler:  handler,
    })

    runner.Run()
}

func myHandler(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
    log.Printf("Processing: %v", msg.Payload)

    // Your logic here

    return map[string]interface{}{
        "status": "completed",
        "result": "...",
    }, nil
}
```

### Compile and Run

```bash
go build -o bin/my-agent examples/agents/my_agent.go
./bin/my-agent
```

---

## Part 9: Troubleshooting

### Agent Not Receiving Messages

**Check:**
1. Is agent running? (look for "Agent runner started" log)
2. Is message in correct inbox? (`ls .ailang/state/messages/agent-id/`)
3. Is file named correctly? (must end in `.pending.json`)

**Fix:**
```bash
# Check agent status
sqlite3 .ailang/state/agents.db "SELECT * FROM agents WHERE agent_id='echo-agent';"

# Check messages
ls -la .ailang/state/messages/echo-agent/
```

### Message Stuck Processing

**Symptom**: Message never completes

**Cause**: Lease still held (agent crashed)

**Fix:**
```bash
# Check for expired leases
sqlite3 .ailang/state/agents.db "SELECT * FROM agent_locks WHERE expires_at < datetime('now');"

# Manually reap
sqlite3 .ailang/state/agents.db "DELETE FROM agent_locks WHERE expires_at < datetime('now');"
```

### Database Locked

**Symptom**: "database is locked" error

**Cause**: WAL mode not enabled or multiple processes

**Fix:**
```bash
sqlite3 .ailang/state/agents.db "PRAGMA journal_mode=WAL;"
```

---

## Part 10: Next Steps

### Production Deployment

1. **Use systemd** (Linux):
```bash
sudo systemctl start ailang-echo-agent
```

2. **Use Docker Compose**:
```yaml
services:
  echo-agent:
    image: ailang-agent:latest
    command: /bin/echo-agent
```

3. **Use GitHub Actions**:
```yaml
- name: Run agents
  run: |
    ./bin/eval-analyzer --once
    ./bin/design-doc-creator --once
```

### Monitoring

```bash
# Watch agent activity
watch -n 1 'sqlite3 .ailang/state/agents.db "SELECT * FROM agents"'

# Monitor messages
watch -n 1 'ls -lh .ailang/state/messages/*/*.pending.json | wc -l'
```

### Create More Agents

Look at the examples:
- `examples/agents/echo_agent.go` - Simple echo
- `examples/agents/eval_analyzer_agent.go` - Complex analysis

Build your own by copying the template!

---

## Summary

You've learned:
- ✅ How to run agents
- ✅ How to send messages
- ✅ How to check responses
- ✅ How to inspect the database
- ✅ How crash recovery works
- ✅ How to build your own agent

**Next**: Read [AGENT_BRIDGE_EXPLAINED.md](AGENT_BRIDGE_EXPLAINED.md) to learn about integrating with `.claude/agents/` files!
