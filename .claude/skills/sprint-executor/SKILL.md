---
name: AILANG Sprint Executor
description: Execute approved sprint plans with test-driven development, continuous linting, progress tracking, and pause points. Supports parallel milestone execution via Task sub-agents. Use when user says "execute sprint", "start sprint", or wants to implement an approved sprint plan.
---

# AILANG Sprint Executor

Execute an approved sprint plan with continuous progress tracking, testing, and documentation updates. Supports **parallel execution** of independent milestones using Task sub-agents for faster sprints.

## Quick Start

**Sequential execution (default):**
```bash
# User says: "Execute the sprint plan in design_docs/20251019/M-S1.md"
# This skill will:
# 1. Validate prerequisites (tests pass, linting clean)
# 2. Create TodoWrite tasks for all milestones
# 3. Execute each milestone with test-driven development
# 4. Run checkpoint after each milestone (tests + lint)
# 5. Update CHANGELOG and sprint plan progressively
# 6. Pause after each milestone for user review
```

**Parallel execution (for independent milestones):**
```bash
# User says: "Execute sprint plan at docs/sprint-plans/M-FOO.md in parallel"
# This skill will:
# 1. Read sprint plan and identify all milestones
# 2. Analyze dependencies — group independent milestones for parallel execution
# 3. Spawn Task sub-agents per milestone (branch, TDD, implement, commit)
# 4. Act as integration agent — merge branches, run full test suite
# 5. Write sprint retrospective with timing and friction analysis
```

## When to Use This Skill

Invoke this skill when:
- User says "execute sprint", "start sprint", "begin implementation"
- User has an approved sprint plan ready to implement
- User wants guided execution with built-in quality checks
- User needs progress tracking and pause points

**Choose parallel mode when:**
- Sprint has 3+ milestones with independent work
- Milestones touch different files/packages (low merge conflict risk)
- User says "execute in parallel", "parallel sprint", or "use sub-agents"

**Choose sequential mode when:**
- Milestones have strict dependencies (M2 depends on M1's output)
- All milestones touch the same files
- Sprint is small (1-2 milestones)

## Coordinator Integration

**When invoked by the AILANG Coordinator** (detected by GitHub issue reference in the prompt), you MUST output these markers at the end of your response:

```
IMPLEMENTATION_COMPLETE: true
BRANCH_NAME: coordinator/task-XXXX
FILES_CREATED: file1.go, file2.go
FILES_MODIFIED: file3.go, file4.go
```

**Why?** The coordinator uses these markers to:
1. Track implementation completion
2. Post updates to GitHub issues
3. Trigger the merge approval workflow

**Example completion:**
```
## Implementation Complete

All milestones have been completed and tests pass.

**IMPLEMENTATION_COMPLETE**: true
**BRANCH_NAME**: `coordinator/task-abc123`
**FILES_CREATED**: `internal/new_file.go`, `internal/new_test.go`
**FILES_MODIFIED**: `internal/existing.go`
```

## Core Principles

1. **Test-Driven**: All code must pass tests before moving to next milestone
2. **Lint-Clean**: All code must pass linting before moving to next milestone
3. **Document as You Go**: Update CHANGELOG.md and sprint plan progressively
4. **Pause for Breath**: Stop at natural breakpoints for review and approval
5. **Track Everything**: Use TodoWrite to maintain visible progress
6. **DX-First**: Improve AILANG development experience as we go - make it easier next time
7. **Parallelize When Possible**: Independent milestones run as concurrent Task sub-agents for speed
8. **Failing Tests First**: Sub-agents MUST write failing tests before implementation — no exceptions

## Multi-Session Continuity (NEW)

**Sprint execution can now span multiple Claude Code sessions!**

Based on [Anthropic's long-running agent patterns](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents), sprint-executor implements the "Coding Agent" pattern:

- **Session Startup Routine**: Every session starts with `session_start.sh`
  - Checks working directory
  - Reads JSON progress file (`.ailang/state/sprints/sprint_<id>.json`)
  - Reviews recent git commits
  - Validates tests pass
  - Prints "Here's where we left off" summary

- **Structured Progress Tracking**: JSON file tracks state
  - Features with `passes: true/false/null` (follows "constrained modification" pattern)
  - Velocity metrics updated automatically
  - Clear checkpoint messages
  - Session timestamps

- **Pause and Resume**: Work can be interrupted at any time
  - Status saved to JSON: `not_started`, `in_progress`, `paused`, `completed`
  - Next session picks up exactly where you left off
  - No loss of context or progress

**For JSON schema details**, see [`resources/json_progress_schema.md`](resources/json_progress_schema.md)

## Available Scripts

### `scripts/session_start.sh <sprint_id>` **NEW**
Resume sprint execution across multiple sessions.

**When to use:** ALWAYS at the start of EVERY session continuing a sprint.

**What it does:**
1. **Syncs GitHub issues** via `ailang messages import-github`
2. Loads sprint JSON progress file
3. Shows linked GitHub issues with titles from local messages
4. Displays feature progress summary
5. Shows velocity metrics
6. Runs tests to verify clean state
7. Prints "Here's where we left off" summary

### `scripts/validate_prerequisites.sh`
Validate prerequisites before starting sprint execution.

**What it checks:**
1. **Syncs GitHub issues** via `ailang messages import-github`
2. Working directory status (clean/uncommitted changes)
3. Current branch (dev or main)
4. Test suite passes
5. Linting passes
6. Shows unread messages (potential issues/feedback)

### `scripts/validate_sprint_json.sh <sprint_id>` **NEW**
**REQUIRED before starting any sprint.** Validates that sprint JSON has real milestones (not placeholders).

**What it checks:**
- No placeholder milestone IDs (`MILESTONE_ID`)
- No placeholder acceptance criteria (`Criterion 1/2`)
- At least 2 milestones defined
- All milestones have custom values (not defaults)
- Dependencies reference valid milestone IDs

**Exit codes:**
- `0` - Valid JSON, ready for execution
- `1` - Invalid JSON or placeholders detected (sprint-planner must fix)

### `scripts/milestone_checkpoint.sh <milestone_name> [sprint_id]`
Run checkpoint after completing a milestone.

**CRITICAL: Tests passing ≠ Feature working!** This script verifies with REAL DATA.

**What it does:**
1. Runs `make test` and `make lint`
2. Shows git diff of changed files
3. Checks file sizes (AI-friendly codebase guidelines)
4. **Runs milestone-specific verification** (database queries, API calls, etc.)
5. Shows JSON update reminder with current milestone statuses

**Sprint-specific verification (e.g., M-TASK-HIERARCHY):**
- **M1**: Checks observatory_sync.go exists, tasks in DB with non-empty IDs
- **M2**: Checks OTEL_RESOURCE_ATTRIBUTES in executor, spans with task attributes
- **M3**: Checks spans have task_id linked
- **M4**: Checks task aggregates (span_count, tokens) are populated
- **M5**: Checks hierarchy API returns data for existing tasks
- **M6**: Checks TaskHierarchy component imported AND connected in UI
- **M7**: Checks backfill command exists and responds to --help

**Example:**
```bash
# Verify M1 (should pass if entity sync works)
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh M1 M-TASK-HIERARCHY

# Verify M2 (will fail if OTEL attributes not propagated)
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh M2 M-TASK-HIERARCHY
```

**Exit codes:**
- `0` - Checkpoint passed (all verifications succeeded)
- `1` - Checkpoint FAILED (DO NOT mark milestone complete)

### `scripts/acceptance_test.sh <milestone_id> <test_type>` **NEW**
Run end-to-end acceptance tests (parser, builtin, examples, REPL, e2e).

### `scripts/finalize_sprint.sh <sprint_id> [version]` **NEW**
Finalize a completed sprint by moving design docs and updating status.

**What it does:**
- Moves design doc from `planned/` to `implemented/<version>/`
- Moves sprint plan markdown to `implemented/<version>/`
- Updates design doc status to "IMPLEMENTED"
- Updates sprint JSON status to "completed"
- Updates file paths in sprint JSON

**When to use:** After all milestones pass and sprint is complete.

**Example:**
```bash
.claude/skills/sprint-executor/scripts/finalize_sprint.sh M-BUG-RECORD-UPDATE-INFERENCE v0_4_9
```

## Execution Flow

### Phase 0: Session Resumption (for continuing sprints)

**If this is NOT the first session for this sprint:**

```bash
# ALWAYS run session_start.sh first!
.claude/skills/sprint-executor/scripts/session_start.sh <sprint-id>
```

This prints "Here's where we left off" summary. **Then skip to Phase 2** to continue with the next milestone.

### Phase 1: Initialize Sprint (first session only)

1. **Validate Sprint JSON** - Run `validate_sprint_json.sh <sprint-id>` **REQUIRED FIRST**
   - If validation fails, STOP and notify user that sprint-planner must fix the JSON
   - Do NOT proceed with placeholder milestones
2. **Read Sprint Plan** - Parse markdown + load JSON progress file (`.ailang/state/sprints/sprint_<id>.json`)
3. **Validate Prerequisites** - Run `validate_prerequisites.sh` (tests, linting, git status)
4. **Create Todo List** - Use TodoWrite to track all milestones
5. **Initial Status Update** - Mark sprint as "🔄 In Progress"
6. **Initial DX Review** - Consider tools/helpers that would make sprint easier (see [resources/dx_improvement_patterns.md](resources/dx_improvement_patterns.md))

### Phase 2: Choose Execution Mode

After Phase 1 initialization, choose between sequential or parallel execution based on milestone dependencies.

**Decision criteria:**
- If milestones form a strict chain (each depends on previous) → **Phase 2A: Sequential**
- If 2+ milestones are independent (no shared file overlap) → **Phase 2B: Parallel**
- Mixed dependencies → Execute dependent chains sequentially, independent groups in parallel

### Phase 2A: Sequential Execution (default)

**For each milestone:**

1. **Pre-Implementation** - Mark milestone as `in_progress` in TodoWrite
2. **Implement** - Write code with DX awareness (helper functions, debug flags, better errors)
3. **Write Tests** - TDD recommended for complex logic, comprehensive coverage required
4. **Verify Quality** - Run `milestone_checkpoint.sh <milestone-name>` (tests + lint must pass)
5. **Update Documentation**:
   - Active changelog in `changelogs/` (what, LOC, key decisions) — find active file with `ls changelogs/ | grep current`
   - Example files (REQUIRED for new language features):
     - Create `examples/runnable/<feature>.ail` with comprehensive examples
     - Add entry to `examples/manifest.json` (path, status, tags, description)
     - Update statistics in manifest (total, working counts)
     - Verify searchable: `ailang examples search "<feature>"`
   - Sprint plan markdown (mark milestone as ✅)
6. **Update Sprint JSON** ⚠️ **CRITICAL** - The checkpoint script reminds you!
   - Update `passes: true/false` in `.ailang/state/sprints/sprint_<id>.json`
   - Set `completed: "<ISO timestamp>"`
   - Add `notes: "<summary of what was done>"`
7. **DX Reflection** - Identify and implement quick wins (<15 min), defer larger improvements
8. **Pause for Breath** - Show progress, ask user if ready to continue

After all milestones complete, proceed to **Phase 4: Finalize Sprint**.

**Quick tips:**
- Use parser test helpers from `internal/parser/test_helpers.go`
- Use `DEBUG_PARSER=1` for token flow tracing
- Use `make doc PKG=<package>` for API discovery
- See [resources/parser_patterns.md](resources/parser_patterns.md) for parser/pattern matching
- See [resources/codegen_patterns.md](resources/codegen_patterns.md) for Go code generation (builtins, expressions)
- See [resources/api_patterns.md](resources/api_patterns.md) for common API patterns
- See [resources/dx_improvement_patterns.md](resources/dx_improvement_patterns.md) for DX opportunities

### Phase 2B: Parallel Milestone Execution

**Use this mode when independent milestones can be developed concurrently using Task sub-agents.**

#### Step 1: Dependency Analysis

Read the sprint plan and build a dependency graph from the milestones:

```
For each milestone M:
  - Parse M.dependencies[] from sprint JSON
  - Identify files M will touch (use Grep to scope relevant files)
  - Check for file-level conflicts between milestones
```

**Group milestones into parallelizable waves:**
```
Wave 1: [M1, M3, M5]  ← no dependencies, no shared files
Wave 2: [M2, M4]       ← depend on Wave 1 results
Wave 3: [M6]           ← depends on Wave 2 results
```

Report the dependency graph to the user before proceeding:
```
Dependency Analysis:
  Wave 1 (parallel): M1 (parser), M3 (stdlib), M5 (docs)
  Wave 2 (parallel): M2 (depends on M1), M4 (depends on M3)
  Wave 3 (sequential): M6 (depends on M2 + M4)

Estimated speedup: ~2.5x over sequential execution
Proceed with parallel execution? [y/n]
```

#### Step 2: Spawn Task Sub-Agents Per Milestone

For each parallelizable wave, spawn one `Task` sub-agent per milestone **in a single message** (so they execute concurrently). Each sub-agent receives a detailed prompt:

```python
# Example: spawning Wave 1 in parallel (all in ONE message with multiple Task calls)
Task(
    description=f"Sprint milestone {milestone.id}",
    subagent_type="general-purpose",
    prompt=f"""
    You are executing milestone {milestone.id}: {milestone.description}
    Sprint plan: {plan_path}

    ## Your Scope
    Branch: sprint/{milestone_slug}
    Relevant files (from dependency analysis): {milestone.relevant_files}

    ## MANDATORY: Test-Driven Development

    **Step 1: Create branch**
    ```bash
    git checkout -b sprint/{milestone_slug}
    ```

    **Step 2: Write FAILING tests FIRST**
    For each acceptance criterion in the milestone:
    - Write a test that captures the expected behavior
    - Run `make test` — confirm the new tests FAIL
    - Do NOT proceed to implementation until you have failing tests

    Acceptance criteria:
    {milestone.acceptance_criteria}

    **Step 3: Implement until all tests pass**
    - Write the minimum code to make each failing test pass
    - Run `make test` after each change
    - Run `make lint` — fix any issues immediately
    - Keep functions small and focused (<50 lines)

    **Step 4: Commit with milestone reference**
    ```bash
    git add <specific-files>
    git commit -m "Complete {milestone.id}: {milestone.description}

    Acceptance criteria met:
    {acceptance_criteria_checklist}

    Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
    ```

    **Step 5: Report back**
    At the END of your response, output this EXACT format:
    ```
    MILESTONE_REPORT:
    milestone_id: {milestone.id}
    status: success|failure
    tests_passed: <count>
    tests_failed: <count>
    files_changed: <comma-separated list>
    branch: sprint/{milestone_slug}
    notes: <brief summary of what was done>
    ```

    ## RULES
    - Do NOT skip the failing-test-first step
    - Do NOT commit with failing tests
    - Do NOT modify files outside your scope: {milestone.relevant_files}
    - If blocked, report status: failure with clear explanation
    """
)
```

**Critical rules for sub-agent spawning:**
- All sub-agents in the same wave MUST be launched in a **single message** with multiple `Task` calls
- Each sub-agent works on its **own branch** (`sprint/<milestone-slug>`)
- Sub-agents must NOT touch files outside their scoped set
- Sub-agents must use `Grep` to verify they're only reading relevant files before editing

#### Step 3: Collect Sub-Agent Results

After all sub-agents in a wave complete, parse their `MILESTONE_REPORT` blocks:

```json
{
  "wave": 1,
  "results": [
    {"milestone_id": "M1", "status": "success", "tests_passed": 12, "tests_failed": 0, "branch": "sprint/m1-parser-fix"},
    {"milestone_id": "M3", "status": "success", "tests_passed": 8, "tests_failed": 0, "branch": "sprint/m3-stdlib-list"},
    {"milestone_id": "M5", "status": "failure", "tests_passed": 3, "tests_failed": 2, "branch": "sprint/m5-docs"}
  ]
}
```

**If any milestone in a wave failed:**
- Do NOT proceed to the next wave
- Report the failure to the user with the sub-agent's notes
- Options: (a) retry the failed milestone, (b) fix manually, (c) skip and continue
- Failed milestones block any downstream milestones that depend on them

**If all milestones in a wave succeeded:**
- Proceed to Phase 3: Integration for this wave
- Then continue with the next wave

#### Step 4: Repeat for Each Wave

Execute waves sequentially (Wave 1 → integrate → Wave 2 → integrate → ...). Within each wave, milestones run in parallel.

### Phase 3: Integration Agent

**After parallel milestones complete, act as the integration agent.**

This phase merges all milestone branches and verifies the combined result. The executor (you) performs this directly — do NOT delegate integration to sub-agents.

#### Step 3.1: Create Integration Branch

```bash
# Start from the base branch (dev or main)
git checkout dev
git checkout -b sprint/integration
```

#### Step 3.2: Merge Milestone Branches

Merge each successful milestone branch into `sprint/integration`, one at a time:

```bash
# Merge in dependency order (Wave 1 first, then Wave 2, etc.)
git merge sprint/m1-parser-fix --no-ff -m "Integrate M1: parser fix"
git merge sprint/m3-stdlib-list --no-ff -m "Integrate M3: stdlib list"
git merge sprint/m5-docs --no-ff -m "Integrate M5: docs"
```

**If a merge conflict occurs:**
1. Identify which milestones conflict (check the conflicting files)
2. Analyze the conflict — determine which milestone's changes take priority
3. Resolve the conflict, preserving both milestones' functionality
4. Run `make test` after resolution to verify nothing broke
5. Document the conflict and resolution in the sprint retrospective

#### Step 3.3: Run Full Test Suite

```bash
make test          # ALL tests must pass
make lint          # ALL linting must pass
make fmt-check     # Formatting must be clean
```

**If integration tests fail:**
1. Identify which milestone introduced the failure using `git bisect` or selective reverts:
   ```bash
   # Revert last merge to isolate
   git revert -m 1 HEAD
   make test
   # If tests pass now, the reverted milestone caused the failure
   ```
2. Fix the conflict between milestones
3. Re-run full test suite
4. Do NOT proceed until all tests pass

#### Step 3.4: Update Sprint Progress JSON

After successful integration, update `.ailang/state/sprints/sprint_<id>.json`:
- Set `passes: true` for each integrated milestone
- Update `actual_loc` from git diff stats
- Set `completed` timestamps
- Update velocity metrics
- Set sprint `status: "in_progress"` (or `"completed"` if all waves done)

### Phase 3.5: Sprint Retrospective

**After ALL waves are integrated and tests pass, write a retrospective.**

Create `docs/sprint-retros/<sprint-id>-retro.md` with:

```markdown
# Sprint Retrospective: <sprint-id>

## Summary
- **Sprint**: <sprint-id>
- **Duration**: <actual days> (estimated: <planned days>)
- **Execution mode**: Parallel (Wave count: N)
- **Total milestones**: N (passed: X, failed: Y)

## Milestone Timing

| Milestone | Estimated LOC | Actual LOC | Time Taken | Status |
|-----------|--------------|------------|------------|--------|
| M1        | 200          | 185        | 45 min     | ✅     |
| M2        | 150          | 210        | 1h 10min   | ✅     |
| ...       |              |            |            |        |

## Parallelization Results
- **Waves executed**: 3
- **Max parallelism**: 3 agents (Wave 1)
- **Speedup vs sequential**: ~2.1x
- **Integration conflicts**: 1 (M1 + M3 shared `types.go`)

## Friction Encountered
- <Description of any blockers, unexpected complexity, or tooling gaps>
- <Merge conflicts and how they were resolved>
- <Sub-agent failures and root causes>

## Recommendations for Next Sprint
- <Suggestions for improving parallelization>
- <Files that should NOT be parallelized (high conflict risk)>
- <DX improvements identified during execution>
```

### Phase 4: Finalize Sprint

1. **Final Testing** - Run `make test`, `make lint`, `make test-coverage-badge`
2. **Documentation Review** - Verify CHANGELOG.md, example files, sprint plan complete
3. **Example Manifest** - For new language features, verify `examples/manifest.json` updated
4. **Final Commit** - Git commit with sprint summary (milestones, LOC, velocity)
5. **Move Design Docs** - Run `finalize_sprint.sh <sprint-id> [version]` to:
   - Move design docs from `planned/` to `implemented/<version>/`
   - Update design doc status to IMPLEMENTED
   - Update sprint JSON status to "completed"
   - Update file paths in sprint JSON
6. **Summary Report** - Compare planned vs actual (LOC, time, velocity)
7. **DX Impact Summary** - Document improvements made during sprint

## Key Features

### Continuous Testing
- Run `make test` after every file change
- Never proceed if tests fail
- Track test count increase

### Continuous Linting
- Run `make lint` after implementation
- Fix linting issues immediately
- Use `make fmt` for formatting

### Progress Tracking
- TodoWrite shows real-time progress
- Sprint plan updated at each milestone
- CHANGELOG.md grows incrementally
- **JSON file tracks structured state** (NEW)
- Git commits create audit trail

### GitHub Issue Integration (NEW)
**Uses `ailang messages` for GitHub sync and issue tracking!**

**Automatic sync:**
- `session_start.sh` and `validate_prerequisites.sh` run `ailang messages import-github` first
- Linked issues shown with titles from local message database

If `github_issues` is set in sprint JSON:
- `validate_sprint_json.sh` shows linked issues
- `session_start.sh` displays issue titles from messages
- `milestone_checkpoint.sh` reminds you to include `Refs #...` in commits
- `finalize_sprint.sh` suggests commit message with issue references

**Commit message format:**
```bash
# During development - use "refs" to LINK without closing
git commit -m "Complete M1: Parser foundation, refs #17"

# Final sprint commit - use "Fixes" to AUTO-CLOSE issues on merge
git commit -m "Finalize sprint M-BUG-FIX

Fixes #17
Fixes #42"
```

**Important: "refs" vs "Fixes"**
- `refs #17` - Links commit to issue (NO auto-close)
- `Fixes #17`, `Closes #17`, `Resolves #17` - AUTO-CLOSES issue when merged

**Workflow:**
1. Sprint JSON has `github_issues: [17, 42]` (set by sprint-planner, deduplicated)
2. During development: Use `refs #17` to link commits without closing
3. Final commit: Use `Fixes #17` to auto-close issues on merge
4. No duplicates: `ailang messages import-github` checks existing issues before importing

### Pause Points
- After each milestone completion
- When tests or linting fail (fix before continuing)
- When user requests "pause"
- When encountering unexpected issues

### Error Handling
- **If tests fail**: Show output, ask how to fix, don't proceed
- **If linting fails**: Show output, ask how to fix, don't proceed
- **If implementation unclear**: Ask for clarification, don't guess
- **If milestone takes much longer than estimated**: Pause and reassess

## Resources

### Multi-Session State
- **JSON Schema**: [`resources/json_progress_schema.md`](resources/json_progress_schema.md) - Sprint progress format
- **Session Startup**: Use `session_start.sh` ALWAYS for continuing sprints

### Development Tools
- **Parser & Patterns**: [`resources/parser_patterns.md`](resources/parser_patterns.md) - Parser development + pattern matching pipeline
- **Codegen Patterns**: [`resources/codegen_patterns.md`](resources/codegen_patterns.md) - Go code generation for new features/builtins
- **API Patterns**: [`resources/api_patterns.md`](resources/api_patterns.md) - Common constructor signatures and API gotchas
- **Developer Tools**: [`resources/developer_tools.md`](resources/developer_tools.md) - Make targets, ailang commands, workflows
- **DX Improvements**: [`resources/dx_improvement_patterns.md`](resources/dx_improvement_patterns.md) - Identifying and implementing DX wins
- **DX Quick Reference**: [`resources/dx_quick_reference.md`](resources/dx_quick_reference.md) - ROI calculator, decision matrix
- **Milestone Checklist**: [`resources/milestone_checklist.md`](resources/milestone_checklist.md) - Step-by-step per milestone

### External Documentation
- **Parser Guide**: [docs/guides/parser_development.md](../../docs/guides/parser_development.md)
- **Website**: https://ailang.sunholo.com/

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + execution workflow) - ~250 lines
2. **Execute as needed**: Scripts in `scripts/` directory (validation, checkpoints, testing)
3. **Load on demand**: Resources in `resources/` directory (detailed guides, patterns, references)

## Prerequisites

- Working directory clean (or only sprint-related changes)
- Current branch `dev` (or specified in sprint plan)
- All existing tests pass
- All existing linting passes
- Sprint plan approved and documented
- **JSON progress file created AND POPULATED by sprint-planner** (not just template!)
- **JSON must pass validation**: `scripts/validate_sprint_json.sh <sprint-id>`

## Failure Recovery

### If Tests Fail During Sprint
1. Show test failure output
2. Ask user: "Tests failing. Options: (a) fix now, (b) revert change, (c) pause sprint"
3. Don't proceed until tests pass

### If Linting Fails During Sprint
1. Show linting output
2. Try auto-fix: `make fmt`
3. If still failing, ask user for guidance
4. Don't proceed until linting passes

### If Implementation Blocked
1. Show what's blocking progress
2. Ask user for guidance or clarification
3. Consider simplifying the approach
4. Document the blocker in sprint plan

### If Velocity Much Lower Than Expected
1. Pause and reassess after 2-3 milestones
2. Calculate actual velocity
3. Propose: (a) continue as-is, (b) reduce scope, (c) extend timeline
4. Update sprint plan with revised estimates

### If Sub-Agent Fails (Parallel Mode)
1. Parse the `MILESTONE_REPORT` from the failed sub-agent
2. Show failure notes and any test output to the user
3. Options: (a) retry milestone with a new sub-agent, (b) fix manually on the milestone branch, (c) skip milestone and continue
4. Do NOT merge a failed milestone's branch into `sprint/integration`
5. Do NOT proceed to next wave if a failed milestone blocks downstream milestones

### If Integration Merge Conflicts (Parallel Mode)
1. Identify conflicting files between milestone branches
2. Merge non-conflicting branches first
3. For conflicts: resolve preserving both milestones' functionality
4. Run `make test` after each conflict resolution
5. Document all conflicts in the sprint retrospective
6. If resolution is unclear, ask the user before proceeding

## Coordinator Integration (v0.6.2+)

The sprint-executor skill integrates with the AILANG Coordinator for automated workflows.

### Autonomous Workflow

When configured in `~/.ailang/config.yaml`, the sprint-executor agent:
1. Receives handoff messages from sprint-planner
2. Executes sprint plans with TDD and continuous linting
3. Creates approval request for code review before merge

```yaml
coordinator:
  agents:
    - id: sprint-executor
      inbox: sprint-executor
      workspace: /path/to/ailang
      capabilities: [code, test, docs]
      trigger_on_complete: []  # End of chain
      auto_approve_handoffs: false
      auto_merge: false
      session_continuity: true
      max_concurrent_tasks: 1
```

### Receiving Handoffs from sprint-planner

The sprint-executor receives:
```json
{
  "type": "plan_ready",
  "correlation_id": "sprint_M-CACHE_20251231",
  "sprint_id": "M-CACHE",
  "plan_path": "design_docs/planned/v0_6_3/m-cache-sprint-plan.md",
  "progress_path": ".ailang/state/sprints/sprint_M-CACHE.json",
  "session_id": "claude-session-xyz",
  "estimated_duration": "3 days",
  "total_loc_estimate": 650
}
```

### Session Continuity

With `session_continuity: true`:
- Receives `session_id` from sprint-planner handoff
- Uses `--resume SESSION_ID` for Claude Code CLI
- Preserves full conversation context from design → planning → execution
- Maintains understanding of design decisions and rationale

### Human-in-the-Loop

With `auto_merge: false`:
1. Sprint is executed in isolated worktree
2. All milestones completed with tests passing
3. Approval request shows:
   - Git diff of all changes
   - Test results
   - CHANGELOG updates
4. Human reviews implementation quality
5. Approve → Changes merged to main branch
6. Reject → Worktree preserved for fixes

### End of Pipeline

As the last agent in the chain:
- `trigger_on_complete: []` means no automatic handoff
- Final approval merges directly to main branch
- Sprint JSON updated to `status: completed`
- Design docs moved to `implemented/` directory

## Notes

- This skill is long-running - expect it to take hours or days
- Pause points are built in - you're not locked into finishing
- Sprint plan is the source of truth - but reality may require adjustments
- Git commits create a reversible audit trail
- TodoWrite provides real-time visibility into progress
- Test-driven development is non-negotiable - tests must pass
- **Multi-session continuity** - Sprint can span multiple Claude Code sessions
- **JSON state tracking** - Structured progress in `.ailang/state/sprints/sprint_<id>.json`
- **Parallel execution** - Independent milestones run as concurrent Task sub-agents (Phase 2B)
- **Failing tests first** - Sub-agents MUST write failing tests before implementation — no exceptions
- **No merging failing branches** - Only branches with all tests passing enter `sprint/integration`
- **Sprint retrospectives** - Written to `docs/sprint-retros/` after parallel sprints complete
