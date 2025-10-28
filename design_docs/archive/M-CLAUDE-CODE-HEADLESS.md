# M-CLAUDE-CODE-HEADLESS: Automated Headless Workflows (Phase 3)

**Status**: Planned (v0.3.21+)
**Created**: October 25, 2025
**Priority**: P1 (Medium)
**Estimated**: 2-3 days
**Dependencies**: M-CLAUDE-CODE-INTEGRATION-HOOKS (v0.3.20 - ✅ Complete)

---

## Problem Statement

**Phases 1-2 Complete** (v0.3.20):
- ✅ Hooks integration (SessionStart, Stop)
- ✅ User inbox system
- ✅ Agent handoff from interactive sessions

**What's Missing**:
- ❌ No headless mode wrapper scripts
- ❌ No automated workflows (cron jobs)
- ❌ No examples for fully autonomous operation
- ❌ No documentation for headless usage

**Why This Matters**:
Headless mode enables **fully autonomous workflows** where agents operate without human interaction:

```
[Scheduled Task] → [Headless Claude] → [Agent Execution] → [User Notification]
```

**Use Cases**:
1. **Nightly eval baselines** - Run benchmarks automatically, create design docs for failures
2. **Continuous improvement** - Detect issues, implement fixes, create PRs while user sleeps
3. **Scheduled reports** - Generate weekly performance reports, cost analysis, DX metrics
4. **Security audits** - Automated PR reviews, vulnerability scanning
5. **Incident response** - Automated investigation and triage

---

## Background: Claude Code Headless Mode

**From**: https://docs.claude.com/en/docs/claude-code/headless

**What it is**: Run Claude Code programmatically without interactive UI using the `claude` CLI with the `--print` (or `-p`) flag.

**Basic Usage**:
```bash
# Non-interactive mode - prints final result
claude -p "Stage my changes and write a set of commits for them" \
  --allowedTools "Bash,Read" \
  --permission-mode acceptEdits

# With JSON output for programmatic parsing
claude -p "Analyze eval failures" --output-format json

# Resume conversation by session ID
claude --resume 550e8400-e29b-41d4-a716-446655440000 \
  "Fix linting issues" --no-interactive
```

**Key Flags**:
- `--print`, `-p` - Non-interactive mode
- `--output-format` - `text`, `json`, or `stream-json`
- `--resume` - Resume conversation by session ID
- `--continue` - Continue most recent conversation
- `--allowedTools` - Whitelist specific tools
- `--disallowedTools` - Blacklist specific tools
- `--append-system-prompt` - Add custom instructions

**Output Formats**:

1. **Text** (default): Plain text output
2. **JSON**: Structured data with metadata
   ```json
   {
     "type": "result",
     "subtype": "success",
     "total_cost_usd": 0.003,
     "result": "...",
     "session_id": "abc123"
   }
   ```
3. **Stream JSON**: Each message as separate JSON object (for real-time processing)

---

## Goals

**Primary Goal**: Enable fully autonomous AILANG workflows using headless Claude Code

**Success Metrics**:
- ✅ Can run `claude -p` with custom prompts or agent files
- ✅ JSON output captured with session_id, cost, result
- ✅ Errors sent to user inbox
- ✅ Output logged to `.ailang/state/headless_output/`
- ✅ Cron jobs work reliably
- ✅ > 90% headless runs complete without crash

---

## Solution Design

### Architecture

**Components**:

1. **Headless Wrapper** (`tools/run_headless_claude.sh`)
   - Basic wrapper around `claude -p`
   - Handles JSON output capture
   - Error notification to user inbox
   - Logging to `.ailang/state/headless_output/`

2. **Agent-Style Wrapper** (`tools/run_claude_agent.sh`)
   - Reads agent file (`.claude/agents/*.md`)
   - Combines with task description
   - Runs via headless wrapper

3. **Auto-Handoff Script** (`scripts/auto_handoff.sh`)
   - Runs after headless completion
   - Detects created design docs
   - Sends to sprint-planner agent

4. **Cron Job Examples** (`examples/cron/`)
   - Nightly eval baseline
   - Weekly performance report
   - Hourly DX friction checks

### Workflow Example: Nightly Eval Baseline

```bash
# Cron: Daily at 3am
0 3 * * * cd /path/to/ailang && ./examples/cron/nightly_eval_baseline.sh
```

**Script Flow**:
```
1. Run headless Claude:
   claude -p "Run full eval baseline for v0.3.$(date +%Y%m%d)"

2. Claude executes:
   - make eval-baseline EVAL_VERSION=... FULL=true
   - Analyzes failures
   - Creates design docs in design_docs/planned/

3. Exits with JSON:
   {session_id, cost, result, status}

4. Auto-handoff script:
   - Checks exit code
   - Finds new design docs
   - Sends to sprint-planner: {task: "implement_design_doc", ...}

5. Autonomous agents:
   - sprint-planner → sprint-executor → post-release

6. User notification:
   - Message to user inbox: "v0.3.20251026 implemented and released"
```

---

## Implementation Plan

### Task 1: Basic Headless Wrapper

**File**: `tools/run_headless_claude.sh`

**Features**:
- Accept prompt as argument
- Run `claude -p` with JSON output
- Capture output to file
- Extract session_id, cost, result
- Send errors to user inbox
- Log all runs to `.ailang/state/headless_output/`

**Usage**:
```bash
./tools/run_headless_claude.sh "Analyze eval failures" \
  output.json \
  "Bash,Read,Write,Grep"
```

**Estimated**: 3 hours

### Task 2: Agent-Style Wrapper

**File**: `tools/run_claude_agent.sh`

**Features**:
- Read agent file (`.claude/agents/*.md`)
- Combine agent instructions with task
- Call headless wrapper
- Handle agent-specific logging

**Usage**:
```bash
./tools/run_claude_agent.sh \
  .claude/agents/eval-analyzer.md \
  "Analyze v0.3.20 baseline"
```

**Estimated**: 2 hours

### Task 3: Auto-Handoff Script

**File**: `scripts/auto_handoff.sh`

**Features**:
- Called after headless completion
- Detect new design docs
- Send to appropriate agent (usually sprint-planner)
- Log handoff actions

**Usage**:
```bash
# Called by cron script after headless run
./scripts/auto_handoff.sh
```

**Estimated**: 2 hours

### Task 4: Cron Job Examples

**Files**:
- `examples/cron/nightly_eval_baseline.sh`
- `examples/cron/weekly_performance_report.sh`
- `examples/cron/hourly_dx_friction_check.sh`

**Features**:
- Complete end-to-end examples
- Error handling
- Cost tracking
- Notification on completion

**Estimated**: 3 hours

### Task 5: Documentation

**Files**:
- `docs/HEADLESS_AGENTS.md` - Comprehensive guide
- Update `CLAUDE.md` - Reference headless workflows
- Update `README.md` - Mention autonomous capabilities

**Content**:
- Setup instructions
- Example workflows
- Troubleshooting guide
- Security best practices

**Estimated**: 2 hours

---

## Use Cases

### Use Case 1: Nightly Eval Baseline

**Scenario**: Run eval baseline every night, create design docs for failures, trigger autonomous fixes

**Cron**:
```bash
0 3 * * * cd /path/to/ailang && ./examples/cron/nightly_eval_baseline.sh
```

**Script** (`examples/cron/nightly_eval_baseline.sh`):
```bash
#!/bin/bash
set -euo pipefail

VERSION="v0.3.$(date +%Y%m%d)"

# Run eval baseline via headless Claude
./tools/run_headless_claude.sh \
  "Run full eval baseline for $VERSION and analyze failures" \
  "/tmp/eval_baseline_$VERSION.json" \
  "Bash,Read,Write,Grep,Glob"

EXIT_CODE=$?

if [[ $EXIT_CODE -eq 0 ]]; then
    # Extract metrics
    COST=$(jq -r '.total_cost_usd' "/tmp/eval_baseline_$VERSION.json")
    echo "✅ Baseline complete. Cost: \$$COST"

    # Trigger auto-handoff
    ./scripts/auto_handoff.sh
else
    echo "❌ Baseline failed"
    ailang agent send --to-user '{
      "type": "error",
      "source": "nightly_eval_baseline",
      "exit_code": '"$EXIT_CODE"'
    }'
fi
```

**Benefits**:
- No manual baseline runs
- Failures addressed automatically
- User reviews PRs in morning
- Time-efficient (uses off-hours)

### Use Case 2: Weekly Performance Report

**Scenario**: Generate weekly report of performance, costs, and DX metrics

**Cron**:
```bash
0 9 * * 1 cd /path/to/ailang && ./examples/cron/weekly_performance_report.sh
```

**Script**:
```bash
#!/bin/bash
./tools/run_headless_claude.sh \
  "Generate weekly performance report for the past 7 days" \
  "/tmp/weekly_report_$(date +%Y%m%d).json"

# Send report to user inbox
REPORT=$(jq -r '.result' "/tmp/weekly_report_$(date +%Y%m%d).json")
ailang agent send --to-user '{
  "type": "weekly_report",
  "report": "'"$REPORT"'"
}'
```

### Use Case 3: Security PR Review

**Scenario**: Automatically review PRs for security issues

**GitHub Action Trigger**:
```yaml
# .github/workflows/security-review.yml
on: pull_request

jobs:
  security-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Review PR
        run: |
          gh pr diff ${{ github.event.pull_request.number }} | \
          ./tools/run_headless_claude.sh \
            "Review this PR for security vulnerabilities" \
            security-report.json
      - name: Post results
        run: |
          FINDINGS=$(jq -r '.result' security-report.json)
          gh pr comment ${{ github.event.pull_request.number }} \
            --body "$FINDINGS"
```

---

## File Structure

### New Files

```
tools/
├── run_headless_claude.sh              # ~150 LOC - Basic wrapper
└── run_claude_agent.sh                 # ~100 LOC - Agent-style wrapper

scripts/
└── auto_handoff.sh                     # ~120 LOC - Auto handoff

examples/cron/
├── nightly_eval_baseline.sh            # ~80 LOC
├── weekly_performance_report.sh        # ~60 LOC
└── hourly_dx_friction_check.sh         # ~50 LOC

docs/
└── HEADLESS_AGENTS.md                  # ~800 LOC - Comprehensive guide
```

### Modified Files

```
CLAUDE.md                               # +30 LOC - Reference headless workflows
README.md                               # +20 LOC - Mention autonomous capabilities
```

### Total New Code

```
Wrapper scripts:     ~250 LOC
Auto-handoff:        ~120 LOC
Cron examples:       ~190 LOC
Documentation:       ~850 LOC
-------------------------
Total:              ~1,410 LOC
```

---

## Security & Safety

### Concerns

**1. Headless Agent Access**
- Headless agents have full CLI access
- Could make unintended changes

**Mitigation**:
- ✅ Agent files reviewed and version controlled
- ✅ Logs all actions to `.ailang/state/headless_output/`
- ✅ Dry-run mode for testing (future)
- ✅ Capability restrictions in agent prompts

**2. Cron Job Failures**
- Silent failures could go unnoticed
- Resource exhaustion (long-running tasks)

**Mitigation**:
- ✅ All cron jobs send completion/error messages to user inbox
- ✅ Timeout enforcement (default: 30 minutes)
- ✅ Resource monitoring (future)

**3. Cost Control**
- Automated runs could incur unexpected costs

**Mitigation**:
- ✅ JSON output includes `total_cost_usd`
- ✅ Daily cost reports (future)
- ✅ Budget limits in agent prompts (future)

### Best Practices

**For Headless Scripts**:
1. Always use `set -euo pipefail` (fail fast)
2. Log all actions with timestamps
3. Send summary to user inbox on completion
4. Use timeouts (prevent runaway processes)
5. Test in dry-run mode first (future)

**For Cron Jobs**:
1. Use absolute paths
2. Set environment variables explicitly
3. Redirect output to log files
4. Monitor execution time
5. Send notifications (success + failure)

**For Cost Control**:
1. Log costs in every headless run
2. Review weekly cost reports
3. Set budget limits (future)
4. Use cheap models for simple tasks

---

## Testing Plan

### Unit Tests

**Wrapper Scripts**:
- ✅ Accepts valid prompts
- ✅ Captures JSON output correctly
- ✅ Extracts session_id, cost, result
- ✅ Handles errors gracefully
- ✅ Logs to correct location

### Integration Tests

**End-to-End Workflows**:
- ✅ Cron job → headless Claude → design doc → agent handoff
- ✅ Error handling → user inbox notification
- ✅ Cost tracking across multiple runs

### Manual Tests

**Real-World Scenarios**:
1. Run nightly eval baseline manually
2. Verify design docs created
3. Check agent receives handoff message
4. Verify user inbox notification

---

## Success Criteria

- [ ] All wrapper scripts implemented
- [ ] All cron examples work end-to-end
- [ ] Documentation complete with examples
- [ ] Manual testing passes all scenarios
- [ ] CLAUDE.md references headless workflows
- [ ] README.md mentions autonomous capabilities
- [ ] Cost tracking functional
- [ ] Error notifications to user inbox work

---

## Timeline

**Week 1** (3 days):
- Day 1: Implement wrapper scripts (Tasks 1-2)
- Day 2: Implement auto-handoff and cron examples (Tasks 3-4)
- Day 3: Documentation and testing (Task 5)

**Buffer**: 1 day for unexpected issues

**Total**: 3-4 days

---

## Future Enhancements (v0.4.0+)

### Enhancement 1: Dry-Run Mode

**Feature**: Test headless runs without making changes

**Usage**:
```bash
./tools/run_headless_claude.sh "Implement fix" \
  --dry-run \
  --show-plan
```

### Enhancement 2: Resource Monitoring

**Feature**: Track CPU, memory, disk usage

**Usage**:
```bash
# Abort if resource limits exceeded
./tools/run_headless_claude.sh "Build project" \
  --max-memory 4GB \
  --max-cpu 80%
```

### Enhancement 3: Cost Budgets

**Feature**: Abort if cost exceeds budget

**Usage**:
```bash
./tools/run_headless_claude.sh "Analyze codebase" \
  --max-cost-usd 1.00
```

### Enhancement 4: Multi-Agent Orchestration

**Feature**: Coordinate multiple agents in sequence

**Usage**:
```bash
# Run agent chain
./tools/run_agent_chain.sh \
  eval-analyzer → design-doc-creator → sprint-planner → sprint-executor
```

---

## Migration Path

### For Existing Users (v0.3.20 → v0.3.21)

**No breaking changes**:
- Phases 1-2 (hooks + inbox) continue to work
- Headless mode is optional (opt-in)

**To enable headless**:
1. Install Claude CLI (if not installed): `npm install -g @anthropic-ai/claude-code`
2. Test basic headless: `claude -p "Test message" --output-format json`
3. Test with wrapper: `./tools/run_headless_claude.sh "Test task"`
4. Set up cron jobs (optional)

**To test cron jobs**:
```bash
# Run manually first
./examples/cron/nightly_eval_baseline.sh

# If successful, add to crontab
crontab -e
# Add: 0 3 * * * cd /path/to/ailang && ./examples/cron/nightly_eval_baseline.sh
```

---

## Related Design Docs

**Implemented**:
- [M-CLAUDE-CODE-INTEGRATION-HOOKS](../implemented/v0_3_20/M-CLAUDE-CODE-INTEGRATION-HOOKS.md) - Phases 1-2

**Planned**:
- This document (Phase 3)

**Dependencies**:
- M-AGENT-PROTOCOL (v0.3.19 - ✅ Complete)
- M-CLAUDE-CODE-INTEGRATION-HOOKS (v0.3.20 - ✅ Complete)

---

## Summary

**What**: Headless mode wrappers and automated workflows for fully autonomous operation

**Why**: Enable AILANG to work 24/7 without human interaction

**How**:
- Wrapper scripts for `claude -p`
- Auto-handoff after headless completion
- Cron job examples
- Comprehensive documentation

**When**: v0.3.21+ (3-4 days estimated)

**Risk**: Low - builds on proven hooks/inbox system, headless mode exists

**Dependencies**:
- ✅ M-CLAUDE-CODE-INTEGRATION-HOOKS (v0.3.20 - complete)
- ✅ Claude CLI installed (`npm install -g @anthropic-ai/claude-code`)

---

**Created**: October 25, 2025
**Target Version**: v0.3.21+
**Estimated Time**: 3-4 days
**Priority**: P1 (Medium)
**Dependencies**: M-CLAUDE-CODE-INTEGRATION-HOOKS (v0.3.20 - ✅ Complete)
