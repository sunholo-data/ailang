# Headless Claude Examples

Comprehensive examples for various headless Claude workflows.

## Table of Contents

1. [Basic Examples](#basic-examples)
2. [CI/CD Integration](#cicd-integration)
3. [Agent Workflows](#agent-workflows)
4. [Data Processing](#data-processing)
5. [Monitoring & Alerts](#monitoring--alerts)
6. [Advanced Patterns](#advanced-patterns)

## Basic Examples

### Simple Analysis

```bash
# Analyze codebase for tech debt
claude -p "Analyze the codebase for technical debt and list top 5 issues"

# With JSON output
claude -p "Analyze the codebase for technical debt" --output-format json > analysis.json
jq -r '.result' analysis.json
```

### Generate Documentation

```bash
# Generate README for a module
claude -p "Generate a README.md for the internal/eval module" \
  --allowedTools "Read,Write,Grep"
```

### Code Review

```bash
# Review recent changes
claude -p "Review the last 3 commits and identify potential issues" \
  --allowedTools "Bash,Read,Grep" \
  --output-format json > review.json
```

## CI/CD Integration

### GitHub Actions - Eval Baseline

```yaml
# .github/workflows/eval-baseline.yml
name: Run Eval Baseline

on:
  release:
    types: [published]

jobs:
  eval-baseline:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Claude CLI
        run: |
          # Install Claude Code CLI
          curl -fsSL https://get.claude.ai/install.sh | sh

      - name: Run baseline
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        run: |
          claude -p "Use eval-orchestrator agent to run baseline for ${{ github.ref_name }}" \
            --output-format json \
            --allowedTools "Bash,Read,Write" \
            --timeout 1800 \
            > eval_result.json

      - name: Parse results
        run: |
          SUCCESS_RATE=$(jq -r '.successRate' eval_result.json)
          COST=$(jq -r '.cost' eval_result.json)

          echo "Success Rate: $SUCCESS_RATE"
          echo "Total Cost: \$$COST"

          # Fail if below threshold
          if (( $(echo "$SUCCESS_RATE < 0.70" | bc -l) )); then
            echo "::error::Success rate below 70%: $SUCCESS_RATE"
            exit 1
          fi

      - name: Update dashboard
        run: |
          claude -p "Update benchmark dashboard with results from eval_result.json" \
            --allowedTools "Bash,Read,Write,Edit"

      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: eval-results
          path: |
            eval_result.json
            eval_results/
```

### GitLab CI - Automated Testing

```yaml
# .gitlab-ci.yml
stages:
  - test
  - analyze

run-tests:
  stage: test
  script:
    - make test
    - claude -p "Analyze test failures and suggest fixes" \
        --allowedTools "Bash,Read,Grep" \
        --output-format json > test_analysis.json
  artifacts:
    paths:
      - test_analysis.json

analyze-coverage:
  stage: analyze
  script:
    - |
      claude -p "Use test-coverage-guardian agent to analyze coverage and identify gaps" \
        --output-format json > coverage_report.json

      COVERAGE=$(jq -r '.coverage' coverage_report.json)
      if (( $(echo "$COVERAGE < 80" | bc -l) )); then
        echo "Coverage below 80%: $COVERAGE"
        exit 1
      fi
```

## Agent Workflows

### Autonomous Sprint Cycle

```bash
#!/bin/bash
# autonomous_sprint.sh - Complete sprint automation

set -euo pipefail

FEATURE="$1"

echo "Starting autonomous sprint for: $FEATURE"

# Phase 1: Create design doc
echo "Creating design document..."
DESIGN_RESULT=$(claude -p "Use design-doc-creator to create planned doc for $FEATURE" \
  --output-format json \
  --allowedTools "Bash,Read,Write")

DESIGN_DOC=$(echo "$DESIGN_RESULT" | jq -r '.artifactPath')
echo "Design doc created: $DESIGN_DOC"

# Phase 2: Plan sprint
echo "Planning sprint..."
PLAN_RESULT=$(claude -p "Use sprint-planner to create sprint plan from $DESIGN_DOC" \
  --output-format json \
  --allowedTools "Bash,Read,Write,Grep")

PLAN_FILE=$(echo "$PLAN_RESULT" | jq -r '.planPath')
echo "Sprint plan created: $PLAN_FILE"

# Phase 3: Execute sprint
echo "Executing sprint..."
EXEC_RESULT=$(claude -p "Use sprint-executor to execute $PLAN_FILE" \
  --output-format json \
  --allowedTools "*" \
  --permission-mode acceptEdits \
  --timeout 3600)

STATUS=$(echo "$EXEC_RESULT" | jq -r '.status')
if [ "$STATUS" = "success" ]; then
  echo "Sprint completed successfully!"
else
  echo "Sprint execution encountered issues"
  echo "$EXEC_RESULT" | jq -r '.error'
  exit 1
fi
```

### Agent Communication

```bash
#!/bin/bash
# agent_pipeline.sh - Agent handoffs

# Agent A: Generate report
REPORT=$(claude -p "Generate quarterly metrics report" \
  --output-format json \
  --allowedTools "Bash,Read,Grep")

SESSION_A=$(echo "$REPORT" | jq -r '.session_id')
REPORT_FILE=$(echo "$REPORT" | jq -r '.artifactPath')

# Send message to Agent B
./bin/send-message data-analyst "{\"report\": \"$REPORT_FILE\", \"session\": \"$SESSION_A\"}"

# Agent B: Analyze report (triggered by inbox)
claude -p "Check agent inbox and analyze any new reports" \
  --output-format json \
  --allowedTools "Read,Write"
```

## Data Processing

### Batch File Processing

```bash
#!/bin/bash
# process_logs.sh - Process log files in parallel

LOG_DIR="logs/"
MAX_PARALLEL=4

find "$LOG_DIR" -name "*.log" -print0 | \
  xargs -0 -n 1 -P $MAX_PARALLEL -I {} \
    claude -p "Analyze log file {} and extract errors" \
      --output-format json \
      --allowedTools "Read" \
      > {}.analysis.json

# Combine results
jq -s '.' logs/*.analysis.json > combined_analysis.json
```

### Data Transformation

```bash
#!/bin/bash
# transform_data.sh - Multi-step data pipeline

# Step 1: Extract
EXTRACT=$(claude -p "Extract user data from database export.json" \
  --output-format json \
  --allowedTools "Read")

SESSION_ID=$(echo "$EXTRACT" | jq -r '.session_id')

# Step 2: Transform
claude --resume $SESSION_ID \
  "Transform the extracted data to CSV format" \
  --allowedTools "Write"

# Step 3: Validate
claude --resume $SESSION_ID \
  "Validate the CSV output and generate quality report" \
  --output-format json > validation.json
```

## Monitoring & Alerts

### Scheduled Health Check

```bash
#!/bin/bash
# cron_health_check.sh - Run via cron every hour

cd /path/to/project

# Check for issues
HEALTH=$(claude -p "Check codebase health: lint errors, test failures, security issues" \
  --output-format json \
  --allowedTools "Bash,Read,Grep,Glob" \
  2>&1)

# Parse results
ISSUES=$(echo "$HEALTH" | jq -r '.issuesFound')

# Alert if problems found
if [ "$ISSUES" -gt 0 ]; then
  # Send Slack notification
  curl -X POST $SLACK_WEBHOOK \
    -H 'Content-Type: application/json' \
    -d "{\"text\": \"Health check found $ISSUES issues\", \"details\": $(echo "$HEALTH" | jq -r '.summary')}"
fi
```

### Performance Monitoring

```bash
#!/bin/bash
# monitor_performance.sh

while true; do
  # Run performance analysis
  PERF=$(claude -p "Analyze current system performance and identify bottlenecks" \
    --output-format json \
    --allowedTools "Bash,Read")

  CPU_USAGE=$(echo "$PERF" | jq -r '.cpuUsage')

  # Alert if high
  if (( $(echo "$CPU_USAGE > 80" | bc -l) )); then
    echo "High CPU usage detected: $CPU_USAGE%"
    # Trigger alert
  fi

  sleep 300  # Check every 5 minutes
done
```

### Automated Incident Response

```bash
#!/bin/bash
# incident_response.sh - Triggered by monitoring system

ALERT_TYPE="$1"
ALERT_DETAILS="$2"

# Analyze incident
ANALYSIS=$(claude -p "Analyze incident: $ALERT_TYPE. Details: $ALERT_DETAILS. Suggest remediation steps." \
  --output-format json \
  --allowedTools "Bash,Read,Grep")

# Extract recommendations
ACTIONS=$(echo "$ANALYSIS" | jq -r '.recommendedActions[]')

# Log incident
echo "$ANALYSIS" | jq . > "incidents/$(date +%Y%m%d_%H%M%S).json"

# Notify team
echo "$ACTIONS" | mail -s "Incident Analysis: $ALERT_TYPE" ops@example.com
```

## Advanced Patterns

### Multi-Model Comparison

```bash
#!/bin/bash
# compare_models.sh - Compare output from different models

PROMPT="Explain quantum computing in simple terms"

# Run with different models (if supported)
for MODEL in claude-opus claude-sonnet gpt-5; do
  claude -p "$PROMPT" \
    --model $MODEL \
    --output-format json \
    > "output_${MODEL}.json" &
done

wait

# Compare results
jq -s '{opus: .[0].result, sonnet: .[1].result, gpt5: .[2].result}' \
  output_claude-opus.json \
  output_claude-sonnet.json \
  output_gpt-5.json \
  > comparison.json
```

### Iterative Refinement

```bash
#!/bin/bash
# refine_iteratively.sh - Improve output through iterations

INITIAL_PROMPT="Write a blog post about AI safety"
MAX_ITERATIONS=3

# Initial generation
RESULT=$(claude -p "$INITIAL_PROMPT" --output-format json)
SESSION_ID=$(echo "$RESULT" | jq -r '.session_id')

for i in $(seq 1 $MAX_ITERATIONS); do
  echo "Refinement iteration $i..."

  # Get feedback
  FEEDBACK=$(claude --resume $SESSION_ID \
    "Review the previous output and identify areas for improvement" \
    --output-format json)

  # Apply improvements
  RESULT=$(claude --resume $SESSION_ID \
    "Refine the blog post based on the feedback" \
    --output-format json)

  QUALITY=$(echo "$RESULT" | jq -r '.qualityScore')
  echo "Quality score: $QUALITY"

  # Stop if quality threshold reached
  if (( $(echo "$QUALITY > 0.9" | bc -l) )); then
    echo "Quality threshold reached!"
    break
  fi
done

# Save final version
echo "$RESULT" | jq -r '.result' > blog_post_final.md
```

### Distributed Processing

```bash
#!/bin/bash
# distributed_work.sh - Split work across multiple workers

TASKS_FILE="tasks.json"
NUM_WORKERS=8

# Split tasks into chunks
jq -c '.[]' "$TASKS_FILE" | split -l $(($(wc -l < "$TASKS_FILE") / $NUM_WORKERS)) - task_chunk_

# Process chunks in parallel
for CHUNK in task_chunk_*; do
  (
    while IFS= read -r TASK; do
      TASK_ID=$(echo "$TASK" | jq -r '.id')
      TASK_DESC=$(echo "$TASK" | jq -r '.description')

      claude -p "Process task: $TASK_DESC" \
        --output-format json \
        --allowedTools "Bash,Read,Write" \
        > "result_${TASK_ID}.json"
    done < "$CHUNK"
  ) &
done

wait

# Combine results
jq -s '.' result_*.json > final_results.json
```

### Conversation Context Management

```bash
#!/bin/bash
# context_manager.sh - Manage long conversations with context limits

TASK_LIST=(
  "Analyze module A"
  "Analyze module B"
  "Analyze module C"
  "Summarize findings"
)

SESSION_ID=""
MESSAGE_COUNT=0
MAX_MESSAGES=10  # Reset session after this many messages

for TASK in "${TASK_LIST[@]}"; do
  if [ -z "$SESSION_ID" ] || [ $MESSAGE_COUNT -ge $MAX_MESSAGES ]; then
    echo "Starting new session..."
    RESULT=$(claude -p "$TASK" --output-format json)
    SESSION_ID=$(echo "$RESULT" | jq -r '.session_id')
    MESSAGE_COUNT=1
  else
    echo "Continuing session ($MESSAGE_COUNT messages)..."
    RESULT=$(claude --resume $SESSION_ID "$TASK" --output-format json)
    MESSAGE_COUNT=$((MESSAGE_COUNT + 1))
  fi

  # Save result
  echo "$RESULT" | jq -r '.result' >> combined_results.txt
done
```

## Error Handling Examples

### Comprehensive Error Handling

```bash
#!/bin/bash
# robust_execution.sh

set -euo pipefail

ERROR_LOG="errors.log"

run_with_error_handling() {
  local PROMPT="$1"
  local OUTPUT_FILE="$2"

  # Try to execute
  if ! RESULT=$(claude -p "$PROMPT" --output-format json 2>"$ERROR_LOG"); then
    EXIT_CODE=$?

    echo "Execution failed with exit code: $EXIT_CODE" >&2
    cat "$ERROR_LOG" >&2

    # Check error type
    if grep -q "authentication" "$ERROR_LOG"; then
      echo "Authentication error - check ANTHROPIC_API_KEY" >&2
      exit 1
    elif grep -q "rate limit" "$ERROR_LOG"; then
      echo "Rate limited - waiting 60s..." >&2
      sleep 60
      # Retry
      RESULT=$(claude -p "$PROMPT" --output-format json)
    else
      echo "Unknown error - aborting" >&2
      exit 1
    fi
  fi

  # Validate JSON
  if ! echo "$RESULT" | jq -e '.' > /dev/null 2>&1; then
    echo "Invalid JSON response" >&2
    exit 1
  fi

  # Check status
  STATUS=$(echo "$RESULT" | jq -r '.status')
  if [ "$STATUS" != "success" ]; then
    ERROR=$(echo "$RESULT" | jq -r '.error')
    echo "Task failed: $ERROR" >&2
    exit 1
  fi

  # Save result
  echo "$RESULT" > "$OUTPUT_FILE"
  echo "Success: $OUTPUT_FILE"
}

# Use it
run_with_error_handling "Analyze codebase" "analysis.json"
```

## See Also

- [CLI Reference](cli_reference.md) - Complete flag documentation
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [SKILL.md](../SKILL.md) - Main skill documentation
