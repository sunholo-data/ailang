# Claude Code Hooks + AILANG Agent Protocol Integration

**Status**: Planned (v0.4.0)
**Created**: October 23, 2025
**Purpose**: Use Claude Code hooks to bridge interactive sessions with autonomous agent messaging

---

## Vision: Interactive + Autonomous Agent Collaboration

**Current Limitation**:
- Claude Code sessions are **interactive** (human-in-the-loop)
- AILANG agents are **autonomous** (run in background)
- No bridge between the two!

**Solution**:
Use Claude Code hooks to send messages to autonomous agents when specific events occur.

---

## Claude Code Hooks Overview

**What are hooks?**
User-defined shell commands that execute at specific points in Claude Code's lifecycle.

**Why hooks?**
- Deterministic control (not LLM-dependent)
- Integrate with external systems
- Automate workflows
- Send notifications

**Hook Events** (8 total):
1. `SessionStart` - When Claude Code session begins
2. `UserPromptSubmit` - When user submits a prompt
3. `PreToolUse` - Before any tool runs
4. `PostToolUse` - After tool completes successfully
5. `Notification` - When Claude sends an alert
6. `Stop` - When AI agent finishes response
7. `TaskComplete` - When task is marked done
8. `SessionEnd` - When session closes

**Configuration**:
Hooks are configured in `.claude/hooks.json`:

```json
{
  "hooks": {
    "PostToolUse": {
      "command": "bash",
      "args": ["-c", "echo 'Tool used: $TOOL_NAME'"]
    }
  }
}
```

**Docs**: https://docs.claude.com/en/docs/claude-code/hooks-guide

---

## Integration Scenarios

### Scenario 1: Agent Handoff

**Use Case**: Claude Code analyzes eval failures, then hands off to autonomous agent for implementation.

**Hook**: `Stop` (when Claude finishes analysis)

**Workflow**:
```
1. User: "Analyze eval failures"
2. Claude Code: Analyzes, creates design doc
3. Hook fires: Stop event
4. Script sends message to sprint-planner agent:
   {
     "task": "implement_design_doc",
     "design_doc": "design_docs/planned/M-XXX.md",
     "priority": "high"
   }
5. sprint-planner agent picks up message
6. Autonomous execution begins
```

**Hook Configuration**:
```json
{
  "hooks": {
    "Stop": {
      "command": "bash",
      "args": ["-c", "./scripts/agent_handoff.sh"]
    }
  }
}
```

**Script** (`scripts/agent_handoff.sh`):
```bash
#!/bin/bash
# Check if design doc was created
DESIGN_DOC=$(ls -t design_docs/planned/*.md | head -1)

if [ -f "$DESIGN_DOC" ]; then
  echo "Handing off to sprint-planner: $DESIGN_DOC"

  # Send message to agent
  ./bin/send-message sprint-planner "{
    \"task\": \"implement_design_doc\",
    \"design_doc\": \"$DESIGN_DOC\",
    \"priority\": \"high\",
    \"context\": {
      \"session_id\": \"$CLAUDE_SESSION_ID\",
      \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
    }
  }"

  echo "✅ Handoff complete. Check agent inbox:"
  echo "  ./bin/check-inbox sprint-planner"
fi
```

---

### Scenario 2: Real-Time Monitoring

**Use Case**: Monitor Claude Code's tool usage and track it in agent database.

**Hook**: `PostToolUse` (after every tool)

**Workflow**:
```
1. Claude uses tool (e.g., Edit file)
2. Hook fires with tool metadata
3. Script logs to agent database
4. Monitoring dashboard shows activity
```

**Hook Configuration**:
```json
{
  "hooks": {
    "PostToolUse": {
      "command": "bash",
      "args": ["-c", "./scripts/log_tool_usage.sh \"$TOOL_NAME\" \"$TOOL_RESULT\""]
    }
  }
}
```

**Script** (`scripts/log_tool_usage.sh`):
```bash
#!/bin/bash
TOOL_NAME=$1
TOOL_RESULT=$2

# Log to agent database
sqlite3 .ailang/state/agents.db <<EOF
INSERT INTO agent_history (agent_id, event_type, event_data, timestamp)
VALUES (
  'claude-code-session',
  'tool_used',
  json_object('tool', '$TOOL_NAME', 'result', '$TOOL_RESULT'),
  datetime('now')
);
EOF

echo "✅ Logged tool usage: $TOOL_NAME"
```

---

### Scenario 3: Task Delegation

**Use Case**: User asks Claude to run benchmarks. Hook delegates to eval-orchestrator agent.

**Hook**: `UserPromptSubmit` (when user submits prompt)

**Workflow**:
```
1. User: "Run eval baseline for v0.4.0"
2. Hook fires with prompt text
3. Script detects "eval baseline" keywords
4. Sends message to eval-orchestrator agent
5. Agent runs benchmarks autonomously
6. Results posted back to Claude Code session
```

**Hook Configuration**:
```json
{
  "hooks": {
    "UserPromptSubmit": {
      "command": "bash",
      "args": ["-c", "./scripts/delegate_task.sh \"$PROMPT_TEXT\""]
    }
  }
}
```

**Script** (`scripts/delegate_task.sh`):
```bash
#!/bin/bash
PROMPT="$1"

# Detect eval baseline requests
if echo "$PROMPT" | grep -qi "eval baseline"; then
  echo "📊 Delegating to eval-orchestrator agent..."

  # Extract version if mentioned
  VERSION=$(echo "$PROMPT" | grep -oP 'v\d+\.\d+\.\d+' | head -1)

  ./bin/send-message eval-orchestrator "{
    \"task\": \"run_baseline\",
    \"version\": \"${VERSION:-current}\",
    \"context\": {
      \"requested_by\": \"claude-code\",
      \"prompt\": \"$PROMPT\"
    }
  }"

  echo "✅ Task delegated. Agent will run benchmarks."
fi

# Detect design doc creation
if echo "$PROMPT" | grep -qi "create design doc"; then
  echo "📝 Delegating to design-doc-creator agent..."
  # Similar delegation logic
fi
```

---

### Scenario 4: Notification Routing

**Use Case**: Claude Code encounters errors. Hook notifies monitoring agent.

**Hook**: `Notification` (when Claude sends alert)

**Workflow**:
```
1. Claude Code encounters error
2. Sends notification
3. Hook fires with notification details
4. Script sends alert to monitoring agent
5. Agent logs error for analysis
```

**Hook Configuration**:
```json
{
  "hooks": {
    "Notification": {
      "command": "bash",
      "args": ["-c", "./scripts/route_notification.sh \"$NOTIFICATION_TYPE\" \"$NOTIFICATION_MESSAGE\""]
    }
  }
}
```

---

## Architecture: Hooks → Messages → Agents

```
┌─────────────────────────────────────────────────────────┐
│  Claude Code Session (Interactive)                      │
└─────────────────────────────────────────────────────────┘
                    │
                    │ Hook Events
                    ▼
┌─────────────────────────────────────────────────────────┐
│  Hook Scripts (.claude/scripts/)                        │
│  - agent_handoff.sh                                      │
│  - log_tool_usage.sh                                     │
│  - delegate_task.sh                                      │
│  - route_notification.sh                                 │
└─────────────────────────────────────────────────────────┘
                    │
                    │ send-message
                    ▼
┌─────────────────────────────────────────────────────────┐
│  Agent Protocol (.ailang/state/)                        │
│  - messages/                                             │
│  - agents.db                                             │
└─────────────────────────────────────────────────────────┘
                    │
                    │ Polling
                    ▼
┌─────────────────────────────────────────────────────────┐
│  Autonomous Agents (Background)                         │
│  - sprint-planner                                        │
│  - eval-orchestrator                                     │
│  - design-doc-creator                                    │
│  - monitoring-agent                                      │
└─────────────────────────────────────────────────────────┘
```

---

## Implementation Plan

### Phase 1: Basic Hook Integration (~1 day)

1. Create `.claude/hooks.json` configuration
2. Create hook scripts in `scripts/`:
   - `agent_handoff.sh`
   - `log_tool_usage.sh`
   - `delegate_task.sh`
3. Test with Claude Code session
4. Document setup in README

**Files to Create**:
- `.claude/hooks.json` (hook configuration)
- `scripts/agent_handoff.sh` (Stop hook)
- `scripts/log_tool_usage.sh` (PostToolUse hook)
- `scripts/delegate_task.sh` (UserPromptSubmit hook)

### Phase 2: Bidirectional Communication (~2 days)

5. Agent → Claude Code feedback
   - Agents write status files
   - Claude Code reads status on next prompt
6. Create `scripts/check_agent_status.sh`
7. Hook into SessionStart event

**Example Status File** (`.ailang/state/agent_status.json`):
```json
{
  "sprint-planner": {
    "status": "in_progress",
    "milestone": "2/5",
    "message": "Implementing Phase 2...",
    "updated_at": "2025-10-23T18:30:00Z"
  },
  "eval-orchestrator": {
    "status": "completed",
    "result": "Baseline complete: 92% success",
    "updated_at": "2025-10-23T18:25:00Z"
  }
}
```

**Hook** (SessionStart):
```json
{
  "hooks": {
    "SessionStart": {
      "command": "bash",
      "args": ["-c", "./scripts/check_agent_status.sh"]
    }
  }
}
```

**Script Output**:
```
🤖 Agent Status Update:

sprint-planner: In Progress (Milestone 2/5)
  └─ Implementing Phase 2...

eval-orchestrator: Completed
  └─ Baseline complete: 92% success

Use 'check-inbox cli-sender' to see full results.
```

### Phase 3: Advanced Workflows (~3 days)

8. Multi-agent coordination via hooks
9. Conditional delegation (keyword detection)
10. Error recovery workflows
11. Dashboard integration

---

## Hook Script Template

**Template**: `scripts/hook_template.sh`

```bash
#!/bin/bash
# Claude Code Hook Script Template
# Usage: Invoked by Claude Code hooks system

set -e  # Exit on error

# ═════════════════════════════════════════════════════════
# Configuration
# ═════════════════════════════════════════════════════════

STATE_DIR=".ailang/state"
SEND_MESSAGE="./bin/send-message"

# ═════════════════════════════════════════════════════════
# Hook Logic
# ═════════════════════════════════════════════════════════

hook_main() {
    local event_type=$1
    local event_data=$2

    echo "🪝 Hook triggered: $event_type"

    # TODO: Implement hook logic
    # - Parse event_data
    # - Make decisions
    # - Send messages to agents

    echo "✅ Hook complete"
}

# ═════════════════════════════════════════════════════════
# Helper Functions
# ═════════════════════════════════════════════════════════

send_to_agent() {
    local agent_id=$1
    local payload=$2

    "$SEND_MESSAGE" "$agent_id" "$payload"
}

log_to_db() {
    local event_type=$1
    local event_data=$2

    sqlite3 "$STATE_DIR/agents.db" <<EOF
INSERT INTO agent_history (agent_id, event_type, event_data, timestamp)
VALUES ('claude-code-hook', '$event_type', '$event_data', datetime('now'));
EOF
}

# ═════════════════════════════════════════════════════════
# Entry Point
# ═════════════════════════════════════════════════════════

hook_main "$@"
```

---

## Benefits of Hooks Integration

### 1. Seamless Handoff

**Problem**: User works interactively, then wants autonomous execution
**Solution**: Hook detects completion, hands off to agent

**Example**:
```
User (interactive): "Analyze this code and create a fix plan"
Claude Code: Analyzes, creates design doc
Hook (Stop): Detects design doc, sends to sprint-planner
Agent (autonomous): Implements fix without user intervention
```

### 2. Observability

**Problem**: Hard to track what agents are doing
**Solution**: Hooks log all agent activity to database

**Dashboard Query**:
```sql
SELECT event_type, COUNT(*) as count
FROM agent_history
WHERE timestamp > datetime('now', '-1 day')
GROUP BY event_type;
```

### 3. Smart Delegation

**Problem**: User has to manually invoke agents
**Solution**: Hooks detect intent and delegate automatically

**Example**:
```
User: "Run benchmarks and update dashboard"
Hook (UserPromptSubmit): Detects "benchmarks" → sends to eval-orchestrator
Hook (Stop): Detects completion → sends to post-release agent
Result: Fully automated workflow
```

### 4. Error Recovery

**Problem**: Claude Code errors are ephemeral
**Solution**: Hooks capture errors and route to monitoring

**Example**:
```
Claude Code: Tool fails with error
Hook (Notification): Captures error details
Script: Logs to database, sends to monitoring agent
Agent: Analyzes patterns, suggests fixes
```

---

## Example: Complete Workflow

**User Goal**: "Fix failing eval benchmarks"

### Step 1: User Prompt
```
User: "Analyze the 5 failing benchmarks and create a plan to fix them"
```

### Step 2: Claude Code Analysis
Claude Code:
- Reads eval results
- Analyzes failures
- Creates `design_docs/planned/M-EVAL-FIX-5.md`
- Stops (task complete)

### Step 3: Hook Fires (Stop Event)
```bash
# .claude/hooks.json
{
  "hooks": {
    "Stop": {
      "command": "bash",
      "args": ["-c", "./scripts/agent_handoff.sh"]
    }
  }
}
```

### Step 4: Handoff Script Executes
```bash
# scripts/agent_handoff.sh
DESIGN_DOC=$(ls -t design_docs/planned/*.md | head -1)

./bin/send-message sprint-planner "{
  \"task\": \"implement_design_doc\",
  \"design_doc\": \"$DESIGN_DOC\",
  \"priority\": \"high\"
}"

echo "✅ Handed off to sprint-planner agent"
```

### Step 5: Agent Receives Message
```
[sprint-planner] Found 1 pending message(s)
[sprint-planner] Processing message from claude-code-hook
[sprint-planner] Task: implement_design_doc
[sprint-planner] Design doc: M-EVAL-FIX-5.md
[sprint-planner] Creating sprint plan...
```

### Step 6: Autonomous Execution
```
[sprint-planner] Sprint plan created (5 milestones)
[sprint-planner] Handing off to sprint-executor...
[sprint-executor] Milestone 1/5: Write tests... ✅
[sprint-executor] Milestone 2/5: Implement fix... ✅
[sprint-executor] Milestone 3/5: Run benchmarks... ✅
[sprint-executor] Milestone 4/5: Update docs... ✅
[sprint-executor] Milestone 5/5: Release... ✅
```

### Step 7: User Notification (Optional Hook)
```bash
# When agent completes, notify user
./bin/send-message claude-code-session "{
  \"notification\": \"Sprint complete! 5 benchmarks fixed. v0.4.1 released.\",
  \"results\": \"eval_results/v0.4.1/\"
}"
```

**Total time**: 2 hours (fully autonomous after initial prompt!)

---

## Configuration Examples

### Minimal Setup

**.claude/hooks.json**:
```json
{
  "hooks": {
    "Stop": {
      "command": "bash",
      "args": ["-c", "echo 'Task complete'"]
    }
  }
}
```

### Full Agent Integration

**.claude/hooks.json**:
```json
{
  "hooks": {
    "SessionStart": {
      "command": "bash",
      "args": ["-c", "./scripts/check_agent_status.sh"]
    },
    "UserPromptSubmit": {
      "command": "bash",
      "args": ["-c", "./scripts/delegate_task.sh \"$PROMPT_TEXT\""]
    },
    "PostToolUse": {
      "command": "bash",
      "args": ["-c", "./scripts/log_tool_usage.sh \"$TOOL_NAME\" \"$TOOL_RESULT\""]
    },
    "Notification": {
      "command": "bash",
      "args": ["-c", "./scripts/route_notification.sh \"$NOTIFICATION_TYPE\" \"$NOTIFICATION_MESSAGE\""]
    },
    "Stop": {
      "command": "bash",
      "args": ["-c", "./scripts/agent_handoff.sh"]
    }
  }
}
```

---

## Security Considerations

### 1. Script Permissions

**Problem**: Hooks run with full shell access
**Mitigation**:
- Review all hook scripts before committing
- Use `chmod +x` only on trusted scripts
- Avoid running untrusted code in hooks

### 2. Input Validation

**Problem**: Hook payload could contain malicious data
**Mitigation**:
```bash
# Sanitize inputs
PROMPT=$(echo "$PROMPT_TEXT" | sed 's/[^a-zA-Z0-9 ]//g')
```

### 3. Agent Permissions

**Problem**: Agents could be invoked with escalated permissions
**Mitigation**:
- Agents run with same permissions as hook scripts
- Use principle of least privilege
- Sandbox agent execution

---

## Future Enhancements

### 1. Hook Marketplace

**Vision**: Shared library of useful hooks
- Community-contributed scripts
- Tested and reviewed
- One-click installation

### 2. Visual Hook Builder

**Vision**: GUI for creating hooks (no shell scripting needed)
- Drag-and-drop workflow builder
- Template library
- Test mode

### 3. Agent Discovery

**Vision**: Hooks auto-discover available agents
```bash
# Auto-detect agents from database
AGENTS=$(sqlite3 .ailang/state/agents.db "SELECT agent_id FROM agents WHERE status='active'")
```

### 4. Webhook Integration

**Vision**: External systems can trigger agents via hooks
- GitHub webhooks → agent
- CI/CD → agent
- Monitoring alerts → agent

---

## Next Steps

**Immediate** (v0.4.0):
1. Document hooks integration (this file) ✅
2. Create example hook scripts
3. Test with Claude Code session
4. Update README with setup instructions

**Short-term** (v0.4.1):
5. Implement bidirectional communication
6. Create monitoring dashboard
7. Add more delegation patterns

**Long-term** (v0.5.0):
8. Hook marketplace
9. Visual builder
10. Webhook integration

---

## Resources

- **Claude Code Hooks Guide**: https://docs.claude.com/en/docs/claude-code/hooks-guide
- **Agent Protocol Docs**: `docs/AGENT_TUTORIAL.md`
- **Message Format Spec**: `design_docs/planned/M-AGENT-PROTOCOL.md`
- **Example Scripts**: `scripts/` (to be created)

---

**Created**: October 23, 2025
**Version**: v0.4.0 (planned)
**Status**: Design complete, implementation pending
