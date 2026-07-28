# Autonomous Agent Workflows with AILANG Messaging

Complete guide for building autonomous agent systems using headless Claude + AILANG agent messaging.

## Overview

The AILANG agent messaging system enables task queues, agent-to-agent handoffs, and autonomous workflows. Each agent:

1. **Checks inbox** for assigned tasks
2. **Acknowledges (acks)** message to claim task
3. **Works on task** using headless Claude
4. **Un-acknowledges (unacks)** if task fails (returns to queue)
5. **Sends result** to inbox for handoff or user notification

## Core Commands

### Inbox Management

```bash
# Check messages for specific agent
ailang agent inbox <agent-id> --unread-only

# Check user inbox (for agent → human notifications)
ailang agent inbox user --unread-only

# ⚠️ Important: Flags BEFORE agent-id!
# ✅ Correct:   ailang agent inbox --unread-only my-agent
# ❌ Wrong:     ailang agent inbox my-agent --unread-only
```

### Task Claiming

```bash
# Acknowledge message to claim task (moves from _unread to _processed)
ailang agent ack <message-id>

# Un-acknowledge to return task to queue (moves back to _unread)
ailang agent unack <message-id>

# Acknowledge all unread messages
ailang agent ack --all
```

### Sending Messages

```bash
# Send task to specific agent
ailang agent send <agent-id> '{"task": "...", "data": "..."}'

# Send result to user inbox (agent → human)
ailang agent send --to-user --from "my-agent" '{"status": "complete", "result": "..."}'

# Send with correlation ID for tracking
ailang agent send agent-b --correlation-id cycle_123 '{"task": "..."}'
```

## Pattern 1: Single Agent Task Processor

**Use case**: One agent processing tasks from a queue.

```bash
#!/bin/bash
# agent_worker.sh - Single agent processing loop

AGENT_ID="eval-worker"

while true; do
  echo "Checking for tasks..."

  # Check inbox
  MESSAGES=$(ailang agent inbox --unread-only $AGENT_ID 2>/dev/null)

  if echo "$MESSAGES" | grep -q "No messages"; then
    echo "No tasks, sleeping 30s..."
    sleep 30
    continue
  fi

  # Extract first message ID
  MESSAGE_ID=$(echo "$MESSAGES" | grep "ID:" | head -1 | awk '{print $2}')
  TASK=$(echo "$MESSAGES" | grep "Payload:" | head -1 | sed 's/.*Payload: //')

  echo "Processing task: $MESSAGE_ID"

  # Claim task
  ailang agent ack $MESSAGE_ID

  # Process with headless Claude
  RESULT=$(claude -p "Process this task: $TASK" \
    --output-format json \
    --allowedTools "Bash,Read,Write" \
    2>&1)

  # Check if successful
  if echo "$RESULT" | jq -e '.status == "success"' >/dev/null 2>&1; then
    echo "✓ Task completed successfully"

    # Send result to user inbox
    ailang agent send --to-user --from "$AGENT_ID" \
      "{\"message_id\": \"$MESSAGE_ID\", \"status\": \"complete\", \"result\": $(echo "$RESULT" | jq -c '.result')}"

  else
    echo "✗ Task failed, returning to queue"

    # Un-acknowledge to retry later
    ailang agent unack $MESSAGE_ID

    # Send error notification
    ERROR=$(echo "$RESULT" | jq -r '.error // "Unknown error"')
    ailang agent send --to-user --from "$AGENT_ID" \
      "{\"message_id\": \"$MESSAGE_ID\", \"status\": \"error\", \"error\": \"$ERROR\"}"
  fi

  sleep 5  # Brief pause between tasks
done
```

## Pattern 2: Multi-Agent Pipeline with Handoffs

**Use case**: Tasks flow through multiple specialized agents.

```bash
#!/bin/bash
# agent_pipeline.sh - Agent A → Agent B → Agent C

CORRELATION_ID="cycle_$(date +%Y%m%d_%H%M%S)"

echo "Starting pipeline: $CORRELATION_ID"

# Agent A: Design
echo "Agent A: Creating design..."
DESIGN_RESULT=$(claude -p "Use design-doc-creator to create doc for feature X" \
  --output-format json \
  --allowedTools "Bash,Read,Write")

DESIGN_STATUS=$(echo "$DESIGN_RESULT" | jq -r '.status')
if [ "$DESIGN_STATUS" != "success" ]; then
  echo "Agent A failed, aborting pipeline"
  exit 1
fi

DESIGN_DOC=$(echo "$DESIGN_RESULT" | jq -r '.artifactPath')
echo "✓ Design doc created: $DESIGN_DOC"

# Hand off to Agent B: Planning
echo "Sending task to Agent B (sprint-planner)..."
ailang agent send sprint-planner \
  --correlation-id "$CORRELATION_ID" \
  "{\"task\": \"plan_sprint\", \"design_doc\": \"$DESIGN_DOC\"}"

# Wait for Agent B to complete (polling)
echo "Waiting for Agent B to complete..."
while true; do
  MESSAGES=$(ailang agent inbox --unread-only user 2>/dev/null)

  # Check for message from Agent B with our correlation ID
  if echo "$MESSAGES" | grep -q "$CORRELATION_ID"; then
    MESSAGE_ID=$(echo "$MESSAGES" | grep -B3 "$CORRELATION_ID" | grep "ID:" | awk '{print $2}')
    PLAN_FILE=$(echo "$MESSAGES" | grep -A2 "$CORRELATION_ID" | grep "plan_file" | sed 's/.*"plan_file": *"\([^"]*\)".*/\1/')

    # Acknowledge message
    ailang agent ack $MESSAGE_ID

    echo "✓ Agent B completed: $PLAN_FILE"
    break
  fi

  sleep 10
done

# Hand off to Agent C: Execution
echo "Sending task to Agent C (sprint-executor)..."
ailang agent send sprint-executor \
  --correlation-id "$CORRELATION_ID" \
  "{\"task\": \"execute_sprint\", \"plan_file\": \"$PLAN_FILE\"}"

echo "✓ Pipeline initiated: $CORRELATION_ID"
echo "  Monitor progress: ailang agent inbox user --unread-only"
```

## Pattern 3: Task Distributor (Load Balancer)

**Use case**: Distribute tasks across multiple worker agents.

```bash
#!/bin/bash
# task_distributor.sh - Load balance across workers

TASKS_FILE="tasks.json"
WORKERS=("worker-1" "worker-2" "worker-3" "worker-4")
WORKER_IDX=0

# Read tasks from file
jq -c '.[]' "$TASKS_FILE" | while read -r TASK; do
  # Select worker (round-robin)
  WORKER="${WORKERS[$WORKER_IDX]}"
  WORKER_IDX=$(( (WORKER_IDX + 1) % ${#WORKERS[@]} ))

  # Send task
  echo "Distributing task to $WORKER..."
  ailang agent send "$WORKER" "$TASK"
done

echo "All tasks distributed!"
echo "Monitor workers:"
for WORKER in "${WORKERS[@]}"; do
  echo "  ailang agent inbox --unread-only $WORKER | wc -l"
done
```

## Pattern 4: Agent with Retry and DLQ

**Use case**: Robust agent with automatic retry and dead-letter queue for failed tasks.

```bash
#!/bin/bash
# robust_agent.sh - With retry and DLQ

AGENT_ID="robust-worker"
MAX_RETRIES=3
DLQ_DIR=".ailang/state/dlq"

mkdir -p "$DLQ_DIR"

process_task() {
  local MESSAGE_ID="$1"
  local TASK="$2"
  local RETRY_COUNT="$3"

  echo "Processing task: $MESSAGE_ID (retry $RETRY_COUNT/$MAX_RETRIES)"

  # Execute with Claude
  RESULT=$(claude -p "Process: $TASK" \
    --output-format json \
    --allowedTools "Bash,Read,Write" \
    --timeout 300 \
    2>&1)

  EXIT_CODE=$?

  if [ $EXIT_CODE -eq 0 ] && echo "$RESULT" | jq -e '.status == "success"' >/dev/null 2>&1; then
    echo "✓ Task succeeded"

    # Send success notification
    ailang agent send --to-user --from "$AGENT_ID" \
      "{\"message_id\": \"$MESSAGE_ID\", \"status\": \"complete\"}"

    return 0
  else
    echo "✗ Task failed (attempt $RETRY_COUNT/$MAX_RETRIES)"

    # Check if retries exhausted
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
      echo "⚠️  Max retries reached, moving to DLQ"

      # Save to dead-letter queue
      echo "$TASK" > "$DLQ_DIR/${MESSAGE_ID}.json"
      echo "$RESULT" > "$DLQ_DIR/${MESSAGE_ID}.error.log"

      # Notify user
      ailang agent send --to-user --from "$AGENT_ID" \
        "{\"message_id\": \"$MESSAGE_ID\", \"status\": \"dlq\", \"error\": \"Max retries exceeded\"}"

      return 1
    else
      echo "Will retry after backoff..."

      # Exponential backoff
      BACKOFF=$((2 ** RETRY_COUNT))
      sleep $BACKOFF

      # Un-acknowledge for retry
      ailang agent unack $MESSAGE_ID

      return 2  # Retry
    fi
  fi
}

# Main loop
while true; do
  MESSAGES=$(ailang agent inbox --unread-only $AGENT_ID 2>/dev/null)

  if echo "$MESSAGES" | grep -q "No messages"; then
    sleep 30
    continue
  fi

  # Extract message details
  MESSAGE_ID=$(echo "$MESSAGES" | grep "ID:" | head -1 | awk '{print $2}')
  TASK=$(echo "$MESSAGES" | grep "Payload:" | head -1 | sed 's/.*Payload: //')

  # Extract retry count from task metadata (if exists)
  RETRY_COUNT=$(echo "$TASK" | jq -r '.retry_count // 0')

  # Claim task
  ailang agent ack $MESSAGE_ID

  # Process with retry logic
  process_task "$MESSAGE_ID" "$TASK" "$RETRY_COUNT"
done
```

## Pattern 5: Scheduled Agent Supervisor

**Use case**: Cron job that monitors agents and redistributes stale tasks.

```bash
#!/bin/bash
# agent_supervisor.sh - Run via cron every hour

AGENTS=("eval-worker-1" "eval-worker-2" "sprint-executor")
STALE_THRESHOLD=3600  # 1 hour

for AGENT in "${AGENTS[@]}"; do
  echo "Checking agent: $AGENT"

  # Check for old _processed messages (tasks claimed but not completed)
  PROCESSED_DIR=".ailang/state/messages/$AGENT/_processed"

  if [ -d "$PROCESSED_DIR" ]; then
    # Find stale messages (older than threshold)
    STALE=$(find "$PROCESSED_DIR" -name "*.json" -mmin +$((STALE_THRESHOLD / 60)))

    if [ -n "$STALE" ]; then
      echo "⚠️  Found stale tasks in $AGENT"

      # Un-acknowledge stale tasks
      echo "$STALE" | while read -r FILE; do
        MESSAGE_ID=$(basename "$FILE" .json)
        echo "  Returning to queue: $MESSAGE_ID"
        ailang agent unack $MESSAGE_ID
      done

      # Notify user
      STALE_COUNT=$(echo "$STALE" | wc -l)
      ailang agent send --to-user --from "supervisor" \
        "{\"alert\": \"agent_stale_tasks\", \"agent\": \"$AGENT\", \"count\": $STALE_COUNT}"
    fi
  fi
done
```

## Pattern 6: Interactive Agent with Human-in-the-Loop

**Use case**: Agent pauses for human approval before proceeding.

```bash
#!/bin/bash
# interactive_agent.sh - Requires human approval

AGENT_ID="interactive-planner"

# Check for tasks
MESSAGES=$(ailang agent inbox --unread-only $AGENT_ID)

if echo "$MESSAGES" | grep -q "No messages"; then
  echo "No tasks"
  exit 0
fi

MESSAGE_ID=$(echo "$MESSAGES" | grep "ID:" | head -1 | awk '{print $2}')
TASK=$(echo "$MESSAGES" | grep "Payload:" | head -1 | sed 's/.*Payload: //')

# Claim task
ailang agent ack $MESSAGE_ID

# Phase 1: Generate plan
echo "Generating plan..."
PLAN_RESULT=$(claude -p "Create plan for: $TASK" \
  --output-format json \
  --allowedTools "Read,Write")

PLAN=$(echo "$PLAN_RESULT" | jq -r '.result')

# Send plan to user for approval
ailang agent send --to-user --from "$AGENT_ID" \
  "{\"message_id\": \"$MESSAGE_ID\", \"status\": \"approval_required\", \"plan\": \"$PLAN\", \"approve_command\": \"ailang agent send $AGENT_ID '{\\\"action\\\": \\\"approve\\\", \\\"message_id\\\": \\\"$MESSAGE_ID\\\"}'\"}"

echo "Plan sent to user for approval"
echo "Waiting for user response..."

# Poll for approval (timeout after 1 hour)
TIMEOUT=3600
ELAPSED=0
APPROVED=false

while [ $ELAPSED -lt $TIMEOUT ]; do
  # Check for approval message
  APPROVAL_MSGS=$(ailang agent inbox --unread-only $AGENT_ID)

  if echo "$APPROVAL_MSGS" | grep -q "\"action\": \"approve\""; then
    APPROVAL_ID=$(echo "$APPROVAL_MSGS" | grep -B5 "\"action\": \"approve\"" | grep "ID:" | head -1 | awk '{print $2}')
    ailang agent ack $APPROVAL_ID
    APPROVED=true
    break
  fi

  sleep 10
  ELAPSED=$((ELAPSED + 10))
done

if [ "$APPROVED" = true ]; then
  echo "✓ Plan approved, executing..."

  # Phase 2: Execute plan
  EXEC_RESULT=$(claude -p "Execute this plan: $PLAN" \
    --output-format json \
    --allowedTools "Bash,Read,Write,Edit" \
    --permission-mode acceptEdits)

  # Send completion
  ailang agent send --to-user --from "$AGENT_ID" \
    "{\"message_id\": \"$MESSAGE_ID\", \"status\": \"complete\", \"result\": $(echo "$EXEC_RESULT" | jq -c '.result')}"

else
  echo "✗ Approval timeout"

  # Return task to queue
  ailang agent unack $MESSAGE_ID

  # Notify user
  ailang agent send --to-user --from "$AGENT_ID" \
    "{\"message_id\": \"$MESSAGE_ID\", \"status\": \"timeout\", \"error\": \"Approval not received within 1 hour\"}"
fi
```

## Pattern 7: Parallel Eval Workers

**Use case**: Run eval benchmarks with multiple parallel workers.

```bash
#!/bin/bash
# parallel_eval_workers.sh - Distribute eval tasks

MODELS=("gpt5-mini" "claude-haiku-4-5" "gemini-2-5-flash")
VERSION="v0.3.14"

# Start worker for each model
for MODEL in "${MODELS[@]}"; do
  (
    AGENT_ID="eval-worker-$MODEL"

    echo "[$AGENT_ID] Starting worker for $MODEL..."

    # Send task to self
    ailang agent send "$AGENT_ID" \
      "{\"task\": \"run_eval\", \"model\": \"$MODEL\", \"version\": \"$VERSION\"}"

    # Process task
    MESSAGES=$(ailang agent inbox --unread-only $AGENT_ID)
    MESSAGE_ID=$(echo "$MESSAGES" | grep "ID:" | head -1 | awk '{print $2}')

    ailang agent ack $MESSAGE_ID

    # Run eval
    claude -p "Use eval-orchestrator to run baseline for $VERSION with $MODEL only" \
      --output-format json \
      --allowedTools "Bash,Read,Write" \
      > "eval_${MODEL}.json"

    # Send result
    ailang agent send --to-user --from "$AGENT_ID" \
      "{\"model\": \"$MODEL\", \"status\": \"complete\", \"results\": \"eval_${MODEL}.json\"}"

  ) &
done

# Wait for all workers
wait

echo "All eval workers complete!"
echo "Results:"
ls -lh eval_*.json
```

## Error Handling Best Practices

### 1. Always Handle Failures

```bash
# ❌ WRONG - Silent failure
claude -p "Task"
ailang agent ack $MESSAGE_ID  # Always acks, even if task failed!

# ✅ CORRECT - Check result
RESULT=$(claude -p "Task" --output-format json)
if echo "$RESULT" | jq -e '.status == "success"' >/dev/null 2>&1; then
  # Success - ack stays
  echo "Task complete"
else
  # Failure - return to queue
  ailang agent unack $MESSAGE_ID
fi
```

### 2. Use Correlation IDs

```bash
# Track related messages across agents
CORRELATION_ID="cycle_$(date +%Y%m%d_%H%M%S)"

ailang agent send agent-a --correlation-id "$CORRELATION_ID" '{"task": "..."}'
ailang agent send agent-b --correlation-id "$CORRELATION_ID" '{"task": "..."}'

# Filter messages by correlation
ailang agent inbox user --unread-only | grep "$CORRELATION_ID"
```

### 3. Implement Timeouts

```bash
# Don't let agents hang forever
timeout 300 claude -p "Task" --output-format json > result.json || {
  echo "Task timed out"
  ailang agent unack $MESSAGE_ID
  exit 1
}
```

### 4. Log Everything

```bash
# Preserve full output for debugging
claude -p "Task" --output-format json 2>&1 | tee "logs/${MESSAGE_ID}.log"
```

### 5. Send Progress Updates

```bash
# For long-running tasks, send periodic updates
for STEP in 1 2 3 4 5; do
  echo "Processing step $STEP/5..."

  # Do work
  claude -p "Step $STEP" --output-format json > "step_${STEP}.json"

  # Update user
  ailang agent send --to-user --from "$AGENT_ID" \
    "{\"message_id\": \"$MESSAGE_ID\", \"status\": \"progress\", \"step\": $STEP, \"total\": 5}"
done
```

## Monitoring and Debugging

### Check Agent Status

```bash
# View inbox sizes for all agents
for AGENT in eval-worker-1 eval-worker-2 sprint-planner; do
  COUNT=$(ailang agent inbox --unread-only $AGENT 2>/dev/null | grep -c "ID:")
  echo "$AGENT: $COUNT unread tasks"
done
```

### Inspect Message Details

```bash
# View full message payload
ailang agent inbox --unread-only my-agent | less

# Extract specific field from payload
ailang agent inbox --unread-only my-agent | \
  grep "Payload:" | \
  sed 's/.*Payload: //' | \
  jq -r '.task'
```

### Monitor Logs

```bash
# Watch agent logs in real-time
tail -f .ailang/state/logs/agent-worker.log

# Check for errors
grep "ERROR\|FAILED" .ailang/state/logs/*.log
```

### Dead Letter Queue Analysis

```bash
# List failed tasks
ls -lht .ailang/state/dlq/

# Analyze failure patterns
for FILE in .ailang/state/dlq/*.error.log; do
  echo "=== $FILE ==="
  grep -A 5 "error" "$FILE"
done
```

## CI/CD Integration

### GitHub Actions - Autonomous Eval Pipeline

```yaml
# .github/workflows/autonomous-eval.yml
name: Autonomous Eval Pipeline

on:
  workflow_dispatch:
    inputs:
      version:
        description: 'Version to test'
        required: true

jobs:
  dispatch-tasks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Send tasks to agents
        run: |
          # Send tasks to worker pool
          ailang agent send eval-worker-1 \
            '{"task": "run_eval", "model": "gpt5-mini", "version": "${{ github.event.inputs.version }}"}'

          ailang agent send eval-worker-2 \
            '{"task": "run_eval", "model": "claude-haiku-4-5", "version": "${{ github.event.inputs.version }}"}'

          ailang agent send eval-worker-3 \
            '{"task": "run_eval", "model": "gemini-2-5-flash", "version": "${{ github.event.inputs.version }}"}'

  monitor-completion:
    needs: dispatch-tasks
    runs-on: ubuntu-latest
    steps:
      - name: Wait for results
        run: |
          TIMEOUT=1800  # 30 minutes
          ELAPSED=0
          EXPECTED=3

          while [ $ELAPSED -lt $TIMEOUT ]; do
            COMPLETED=$(ailang agent inbox --unread-only user | grep -c "status.*complete" || true)

            echo "Completed: $COMPLETED/$EXPECTED"

            if [ $COMPLETED -ge $EXPECTED ]; then
              echo "All workers complete!"
              break
            fi

            sleep 30
            ELAPSED=$((ELAPSED + 30))
          done

          if [ $ELAPSED -ge $TIMEOUT ]; then
            echo "Timeout waiting for workers"
            exit 1
          fi
```

## Best Practices Summary

1. **Always acknowledge (ack) messages** when claiming tasks
2. **Always un-acknowledge (unack) on failure** to return to queue
3. **Send results to user inbox** for visibility
4. **Use correlation IDs** for multi-agent workflows
5. **Implement timeouts** to prevent hanging
6. **Log everything** for debugging
7. **Send progress updates** for long-running tasks
8. **Monitor dead-letter queues** for failed tasks
9. **Use exponential backoff** for retries
10. **Test with one agent** before scaling to multiple

## See Also

- [SKILL.md](../SKILL.md) - Main headless runner documentation
- [CLI Reference](cli_reference.md) - Complete command reference
- [Examples](examples.md) - More workflow examples
- [agent-inbox skill](../../agent-inbox/SKILL.md) - Inbox management details
