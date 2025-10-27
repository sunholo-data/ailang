# M-CLAUDE-CODE-HEADLESS Sprint Plan

**Sprint Goal**: Enable headless Claude Code execution for automated eval runs and autonomous workflows

**Duration**: 2 days (based on recent velocity analysis)

**Target Version**: v0.3.22 (unblocks M-EVAL-AGENT v0.4.0)

**Status**: Ready to start

---

## Sprint Summary

**What We're Building**:
- Headless wrapper scripts for `claude -p` command
- Workspace isolation per eval benchmark
- Cost tracking and error handling
- Cron job examples for automated workflows
- Comprehensive documentation

**Why This Sprint**:
- **Blocks M-EVAL-AGENT**: Can't run agent evals without fresh session isolation
- **High ROI**: Enables fully autonomous workflows (nightly evals, continuous improvement)
- **Low risk**: No Go code changes, just bash scripts and docs

**Success Criteria**:
- [ ] Can run headless Claude Code with JSON output capture
- [ ] Workspace isolation working (no state bleed between runs)
- [ ] Cost tracking functional
- [ ] At least one cron example works end-to-end
- [ ] Documentation complete with troubleshooting guide

---

## Velocity Analysis (Last 14 Days)

**Recent Performance**:
- **Commits**: 421 commits / 14 days ≈ 30 commits/day
- **LOC**: 33,351 insertions, 1,051 deletions ≈ 2,400 net LOC/day
- **Recent features**:
  - Parser fix (v0.3.21): ~60 LOC + 425 LOC tests (completed in 1 day)
  - DX tools: ~180 LOC (delimiter tracer, enhanced errors)
  - Agent protocol enhancements: ~900 LOC (completed in 2 days)

**Realistic Velocity**:
- **Script/tooling work**: 500-800 LOC/day (shell scripts, no complex types)
- **Documentation**: 400-600 LOC/day (markdown)
- **Testing**: Manual testing for bash scripts (no unit tests needed)

**This Sprint**:
- **Total LOC**: ~1,410 LOC (scripts ~560, docs ~850)
- **Estimated**: 2 days at 700 LOC/day
- **Buffer**: Design doc estimates 12 hours, velocity supports 2 days

---

## Task Breakdown

### Day 1: Core Wrapper Scripts (4-5 hours)

#### **Task 1.1: Basic Headless Wrapper** (2.5 hours, ~150 LOC)

**File**: `tools/run_headless_claude.sh`

**What it does**:
```bash
# Wraps claude -p with:
# - JSON output capture
# - Cost tracking
# - Workspace isolation
# - Error handling
# - Logging

./tools/run_headless_claude.sh \
  "prompt_or_file.txt" \
  "output.json" \
  "Bash,Read,Write,Edit"
```

**Implementation checklist**:
- [ ] Parse CLI arguments (prompt, output file, allowed tools)
- [ ] Create workspace directory if needed
- [ ] Run `claude -p` with `--output-format json`
- [ ] Capture output to file
- [ ] Extract `session_id`, `total_cost_usd`, `result` from JSON
- [ ] Send errors to user inbox (via `ailang agent send --to-user`)
- [ ] Log to `.ailang/state/headless_output/YYYYMMDD_HHMMSS.log`
- [ ] Exit with claude's exit code

**Dependencies**: None (only uses existing `claude` CLI)

**Testing**:
```bash
# Test 1: Simple prompt
./tools/run_headless_claude.sh "echo hello" /tmp/test1.json "Bash"

# Test 2: File-based prompt
echo "List files in current directory" > /tmp/prompt.txt
./tools/run_headless_claude.sh /tmp/prompt.txt /tmp/test2.json "Bash"

# Test 3: Error handling (invalid tool)
./tools/run_headless_claude.sh "test" /tmp/test3.json "InvalidTool" # Should fail gracefully
```

---

#### **Task 1.2: Agent-Style Wrapper** (1.5 hours, ~100 LOC)

**File**: `tools/run_claude_agent.sh`

**What it does**:
```bash
# Reads .claude/agents/*.md file and combines with task
./tools/run_claude_agent.sh \
  .claude/agents/eval-analyzer.md \
  "Analyze v0.3.22 baseline"
```

**Implementation checklist**:
- [ ] Parse CLI arguments (agent file, task description)
- [ ] Read agent markdown file
- [ ] Combine agent instructions + task into prompt
- [ ] Call `tools/run_headless_claude.sh` with combined prompt
- [ ] Handle agent-specific logging (separate log file per agent)

**Dependencies**: Task 1.1 (headless wrapper)

**Testing**:
```bash
# Test with eval-analyzer agent
./tools/run_claude_agent.sh \
  .claude/agents/eval-analyzer.md \
  "Analyze latest eval results"
```

---

### Day 1 (continued): Auto-Handoff & Examples (2-3 hours)

#### **Task 1.3: Auto-Handoff Script** (1.5 hours, ~120 LOC)

**File**: `scripts/auto_handoff.sh`

**What it does**:
- Detects new design docs created in last 5 minutes
- Sends handoff messages to sprint-planner agent
- Logs handoff actions

**Implementation checklist**:
- [ ] Find design docs in `design_docs/planned/` modified in last 5 min
- [ ] For each doc, extract title and summary
- [ ] Send message to sprint-planner: `ailang agent send sprint-planner '{"task": "implement_design_doc", "doc": "..."}'`
- [ ] Log handoff actions to `.ailang/state/handoff.log`

**Dependencies**: Existing AILANG agent messaging system (`ailang agent send`, v0.3.19+)

**Testing**:
```bash
# Test: Create fake design doc
touch design_docs/planned/test-doc.md
./scripts/auto_handoff.sh
# Should detect and send message
```

---

#### **Task 1.4: Cron Job Example** (1 hour, ~80 LOC)

**File**: `examples/cron/nightly_eval_baseline.sh`

**What it does**:
- Runs headless Claude Code to execute eval baseline
- Handles success/failure notifications
- Triggers auto-handoff on completion

**Implementation checklist**:
- [ ] Generate version string: `v0.3.$(date +%Y%m%d)`
- [ ] Create prompt for eval baseline task
- [ ] Call headless wrapper with prompt
- [ ] Extract cost from JSON output
- [ ] If success: trigger auto-handoff
- [ ] If failure: send error to user inbox
- [ ] Log all actions

**Dependencies**: Tasks 1.1, 1.3

**Testing**:
```bash
# Test manually (don't actually run full eval!)
# Mock the claude command to return success JSON
export MOCK_MODE=1
./examples/cron/nightly_eval_baseline.sh
```

---

### Day 2: Additional Examples & Documentation (4-5 hours)

#### **Task 2.1: Additional Cron Examples** (1 hour, ~110 LOC)

**Files**:
- `examples/cron/weekly_performance_report.sh` (~60 LOC)
- `examples/cron/hourly_dx_friction_check.sh` (~50 LOC)

**What they do**:
- Weekly: Generate performance/cost report, send to user inbox
- Hourly: Check for common DX issues, create design docs for fixes

**Implementation checklist**:
- [ ] `weekly_performance_report.sh`: Run headless Claude with report prompt
- [ ] Extract report from JSON, send to user inbox
- [ ] `hourly_dx_friction_check.sh`: Check test failures, linter errors, stale TODOs
- [ ] If issues found, run headless Claude to create design doc

**Dependencies**: Task 1.1

**Testing**: Manual execution of each script

---

#### **Task 2.2: Comprehensive Documentation** (3 hours, ~850 LOC)

**Files**:
- `docs/HEADLESS_AGENTS.md` (new, ~600 LOC)
- `CLAUDE.md` (update, +30 LOC)
- `README.md` (update, +20 LOC)
- `design_docs/planned/M-CLAUDE-CODE-HEADLESS.md` (update, +200 LOC implementation report)

**Content**:

**`docs/HEADLESS_AGENTS.md`**:
- Overview: What is headless mode, why use it
- Setup: Installing claude CLI, testing basic headless command
- Wrapper scripts: Usage examples for each script
- Use cases: Step-by-step guides for common workflows
- Troubleshooting: Common errors and solutions
- Security: Best practices for automated runs
- Cost control: Budget limits, monitoring costs
- Examples: Nightly eval baseline, weekly reports, etc.

**`CLAUDE.md` updates**:
- Add section: "## 🤖 HEADLESS MODE & AUTOMATION"
- Reference headless wrapper scripts
- Link to `docs/HEADLESS_AGENTS.md`
- Document how headless mode integrates with agent inbox

**`README.md` updates**:
- Mention autonomous capabilities in feature list
- Link to headless mode documentation

**Implementation checklist**:
- [ ] Write `docs/HEADLESS_AGENTS.md` with all sections
- [ ] Add code examples for each use case
- [ ] Include troubleshooting guide with common errors
- [ ] Update `CLAUDE.md` with headless workflow section
- [ ] Update `README.md` feature list
- [ ] Move design doc to `implemented/v0_3_22/` with completion report

---

## Success Metrics

**Primary**:
- [ ] All wrapper scripts execute successfully
- [ ] JSON output captured with correct structure
- [ ] Cost tracking shows actual costs
- [ ] Workspace isolation prevents state bleed
- [ ] At least one cron example works end-to-end

**Secondary**:
- [ ] Documentation comprehensive and clear
- [ ] Troubleshooting guide covers common errors
- [ ] CLAUDE.md workflow updated
- [ ] No regressions (all existing tests pass)

**Validation**:
```bash
# Test sequence:
1. Run headless wrapper with simple prompt
2. Run agent wrapper with eval-analyzer
3. Run auto-handoff script (detect recent design docs)
4. Run nightly eval baseline example (mock mode)
5. Verify logs in .ailang/state/headless_output/
6. Verify user inbox messages for errors
```

---

## Dependencies & Blockers

**External Dependencies**:
- ✅ Claude Code CLI installed (`npm install -g @anthropic-ai/claude-code`)
- ✅ `ailang agent send` command (v0.3.19+)
- ✅ Agent protocol system (v0.3.19+)
- ✅ User inbox system (v0.3.20+)

**Internal Dependencies**:
- Task 1.2 depends on Task 1.1 (headless wrapper)
- Task 1.4 depends on Tasks 1.1, 1.3
- Task 2.1 depends on Task 1.1

**Potential Blockers**:
- ⚠️ **Claude Code headless API changes**: If `claude -p` output format changes, wrapper needs update
- ⚠️ **Cost tracking accuracy**: JSON output must include `total_cost_usd` field
- ⚠️ **Workspace permissions**: Ensure `.ailang/state/` directories are writable

**Mitigation**:
- Test with latest claude CLI version at start of sprint
- Add version check to wrapper script
- Create directories with proper permissions in wrapper

---

## Risks & Mitigation

### Risk 1: Claude Code API Instability
**Likelihood**: Low
**Impact**: High (breaks all automation)

**Mitigation**:
- Test with current `claude` CLI version at start
- Add version detection to wrapper script
- Document tested versions in README
- Fall back to error message if API changes detected

### Risk 2: Cost Overruns in Automated Runs
**Likelihood**: Medium
**Impact**: Medium (unexpected costs)

**Mitigation**:
- All cron examples include cost tracking
- Send cost reports to user inbox
- Document budget recommendations in `docs/HEADLESS_AGENTS.md`
- Add cost limit checks (future enhancement)

### Risk 3: Workspace Isolation Failures
**Likelihood**: Low
**Impact**: High (eval benchmarks contaminate each other)

**Mitigation**:
- Use unique workspace directories per run: `/tmp/ailang_headless_$(date +%s)_$$`
- Clean up workspaces after completion
- Test with multiple concurrent runs
- Add lock file to prevent race conditions

### Risk 4: Silent Failures in Cron Jobs
**Likelihood**: Medium
**Impact**: Medium (missed eval baselines)

**Mitigation**:
- All cron scripts send completion/error messages to user inbox
- Log all actions with timestamps
- Include health check in cron scripts
- Document monitoring best practices

---

## Open Questions

1. **Workspace cleanup**: Should we keep workspaces for debugging or always clean up?
   - **Proposal**: Keep last 5 runs, delete older ones

2. **Cost budgets**: Should wrapper scripts enforce hard cost limits?
   - **Proposal**: Defer to Phase 2 (v0.4.0), document manual monitoring for now

3. **Parallel execution**: Can multiple headless runs happen concurrently?
   - **Proposal**: Yes, with unique workspace directories

4. **Error retry**: Should wrapper scripts retry failed runs?
   - **Proposal**: No auto-retry (let cron handle), log failures for manual investigation

5. **Notification channels**: Just user inbox or also email/Slack?
   - **Proposal**: User inbox only for v0.3.22, defer integrations to future

---

## Post-Sprint Actions

**After completion**:
1. Update M-EVAL-AGENT design doc to reference completed headless wrapper
2. Unblock M-EVAL-AGENT Phase 1 (Claude Code integration)
3. Run pilot eval baseline using headless mode
4. Create v0.3.22 release with headless mode support
5. Update CHANGELOG.md with feature summary

**Next sprint priorities** (v0.4.0):
1. M-EVAL-AGENT Phase 1: Claude Code integration for eval suite
2. Cost budget enforcement
3. Resource monitoring (CPU, memory)
4. Dry-run mode for testing

---

## Related Work

**Implemented**:
- M-AGENT-PROTOCOL (v0.3.19): Agent messaging system
- M-CLAUDE-CODE-INTEGRATION-HOOKS (v0.3.20): Interactive Claude Code integration

**Planned**:
- M-EVAL-AGENT (v0.4.0): Agent-based eval benchmarks (BLOCKED by this sprint)

**Future**:
- Cost budget enforcement (v0.4.0+)
- Resource monitoring (v0.4.0+)
- Multi-agent orchestration (v0.5.0+)

---

## Sprint Retrospective (Post-Completion)

**What went well**:
- (To be filled after sprint)

**What could improve**:
- (To be filled after sprint)

**Lessons learned**:
- (To be filled after sprint)

**Actual vs estimated**:
- Estimated: 2 days, ~1,410 LOC
- Actual: ___ days, ___ LOC

---

**Created**: October 27, 2025
**Sprint Start**: TBD (after user approval)
**Estimated Completion**: 2 days from start
**Depends On**: M-CLAUDE-CODE-INTEGRATION-HOOKS (v0.3.20 ✅)
**Unblocks**: M-EVAL-AGENT Phase 1 (v0.4.0)
