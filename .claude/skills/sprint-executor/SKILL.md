---
name: AILANG Sprint Executor
description: Execute approved sprint plans with test-driven development, continuous linting, progress tracking, and pause points. Use when user says "execute sprint", "start sprint", or wants to implement an approved sprint plan.
---

# AILANG Sprint Executor

Execute an approved sprint plan with continuous progress tracking, testing, and documentation updates.

## Quick Start

**Most common usage:**
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

## When to Use This Skill

Invoke this skill when:
- User says "execute sprint", "start sprint", "begin implementation"
- User has an approved sprint plan ready to implement
- User wants guided execution with built-in quality checks
- User needs progress tracking and pause points

## Core Principles

1. **Test-Driven**: All code must pass tests before moving to next milestone
2. **Lint-Clean**: All code must pass linting before moving to next milestone
3. **Document as You Go**: Update CHANGELOG.md and sprint plan progressively
4. **Pause for Breath**: Stop at natural breakpoints for review and approval
5. **Track Everything**: Use TodoWrite to maintain visible progress

## Available Scripts

### `scripts/validate_prerequisites.sh`
Validate prerequisites before starting sprint execution.

**Usage:**
```bash
.claude/skills/sprint-executor/scripts/validate_prerequisites.sh
```

**Output:**
```
Validating sprint prerequisites...

1/4 Checking working directory...
  ✓ Working directory clean

2/4 Checking current branch...
  ✓ On branch: dev

3/4 Running tests...
  ✓ All tests pass

4/4 Running linter...
  ✓ Linting passes

✓ All prerequisites validated!
Ready to start sprint execution.
```

**Exit codes:**
- `0` - All prerequisites pass
- `1` - One or more prerequisites fail

### `scripts/milestone_checkpoint.sh <milestone_name>`
Run checkpoint after completing a milestone.

**Usage:**
```bash
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh "M-S1.1: Parser foundation"
```

**Output:**
```
Running checkpoint for: M-S1.1: Parser foundation

1/3 Running tests...
  ✓ Tests pass

2/3 Running linter...
  ✓ Linting passes

3/3 Files changed in this milestone...
 internal/parser/parser.go   | 125 ++++++++++++++++++
 internal/parser/parser_test.go | 89 +++++++++++++
 2 files changed, 214 insertions(+)

✓ Milestone checkpoint passed!
Ready to proceed to next milestone.
```

## Execution Flow

### Phase 1: Initialize Sprint

#### 1. Read Sprint Plan
- Parse sprint plan document (e.g., `design_docs/20251019/M-S1.md`)
- Extract all milestones and tasks
- Note dependencies and acceptance criteria
- Identify estimated LOC and duration

#### 2. Validate Prerequisites

**Use the validation script:**
```bash
.claude/skills/sprint-executor/scripts/validate_prerequisites.sh
```

**Manual checks:**
- Working directory clean: `git status --short`
- Current tests pass: `make test`
- Current linting passes: `make lint`
- On correct branch (usually `dev`)

**If validation fails:**
- Fix issues before starting
- Don't proceed with dirty working directory
- Don't start with failing tests or linting

#### 3. Create Todo List

**Use TodoWrite to create tasks:**
- Extract all milestones from sprint plan
- Mark first milestone as `in_progress`
- Keep remaining tasks as `pending`
- This provides real-time progress visibility

#### 4. Initial Status Update
- Update sprint plan with "🔄 In Progress" status
- Add start timestamp
- Commit sprint plan update (optional)

### Phase 2: Execute Milestones

**For each milestone in the sprint:**

#### Step 1: Pre-Implementation
- Mark milestone as `in_progress` in TodoWrite
- Review milestone goals and acceptance criteria
- Identify files to create/modify
- Estimate LOC if not already specified

#### Step 2: Implement
- Write implementation code following the task breakdown
- Follow design patterns from sprint plan
- Add inline comments for complex logic
- Keep functions small and focused

#### Step 3: Write Tests
- Create/update test files (*_test.go)
- Aim for comprehensive coverage (all acceptance criteria)
- Include edge cases and error conditions
- Test both success and failure paths

#### Step 4: Verify Quality

**Run checkpoint script:**
```bash
.claude/skills/sprint-executor/scripts/milestone_checkpoint.sh "Milestone name"
```

**Manual verification:**
```bash
make test  # MUST PASS
make lint  # MUST PASS
```

**CRITICAL**: If tests or linting fail, fix immediately before proceeding.

#### Step 5: Update Documentation

**Update CHANGELOG.md:**
- What was implemented
- LOC counts (implementation + tests)
- Key design decisions
- Files modified/created

**Update sprint plan:**
- Mark milestone as ✅
- Add actual LOC vs estimated
- Note any deviations from plan

#### Step 6: Pause for Breath

**After each milestone:**
- Show summary of what was completed
- Show current sprint progress (X of Y milestones done)
- Show velocity (LOC/day vs planned)
- Ask user: "Ready to continue to next milestone?" or "Need to review/adjust?"
- If user says "pause" or "stop", save current state and exit gracefully

### Phase 3: Finalize Sprint

**When all milestones are complete:**

#### 1. Final Testing
```bash
make test                # Full test suite
make lint                # All linting
make test-coverage-badge # Coverage check
```

#### 2. Documentation Review
- Verify CHANGELOG.md is complete
- Verify sprint plan shows all milestones as ✅
- Update sprint plan with final metrics:
  - Total LOC (actual vs estimated)
  - Total time (actual vs estimated)
  - Velocity achieved
  - Test coverage achieved
  - Any deviations from plan

#### 3. Final Commit
```bash
git commit -m "Complete sprint: <sprint-name>

Milestones completed:
- <Milestone 1>: <LOC>
- <Milestone 2>: <LOC>

Total: <actual-LOC> LOC in <actual-time>
Velocity: <LOC/day>
Test coverage: <percentage>"
```

#### 4. Summary Report
- Show sprint completion summary
- Compare planned vs actual (LOC, time, milestones)
- Highlight any issues or deviations
- Suggest next steps (new sprint, release, etc.)

#### 5. Identify Bumps
- What could AILANG do better to make a smoother coding sprint?
- Is it worth adding a new design doc to help ease how we make AILANG?

## Key Features

### Continuous Testing
- Run `make test` after every file change
- Never proceed if tests fail
- Show test output for visibility
- Track test count increase

### Continuous Linting
- Run `make lint` after implementation
- Fix linting issues immediately
- Use `make fmt` for formatting issues
- Verify with `make fmt-check`

### Progress Tracking
- TodoWrite shows real-time progress
- Sprint plan updated at each milestone
- CHANGELOG.md grows incrementally
- Git commits create audit trail

### Pause Points
- After each milestone completion
- When tests fail (fix before continuing)
- When linting fails (fix before continuing)
- When user requests "pause"
- When encountering unexpected issues

### Error Handling
- **If tests fail**: Show output, ask how to fix, don't proceed
- **If linting fails**: Show output, ask how to fix, don't proceed
- **If implementation unclear**: Ask for clarification, don't guess
- **If milestone takes much longer than estimated**: Pause and reassess

## Resources

### Developer Tools Reference
See [`resources/developer_tools.md`](resources/developer_tools.md) for comprehensive reference of all available make targets, ailang commands, scripts, and workflows. Load this when you need to:
- Know which test targets to use
- Update golden files after parser changes
- Verify stdlib changes
- Run evals or compare baselines
- Troubleshoot build/test/lint issues
- Find the right tool for any development task

### Milestone Checklist
See [`resources/milestone_checklist.md`](resources/milestone_checklist.md) for complete step-by-step checklist per milestone.

## Prerequisites

- Working directory should be clean (or have only sprint-related changes)
- Current branch should be `dev` (or specified in sprint plan)
- All existing tests must pass before starting
- All existing linting must pass before starting
- Sprint plan must be approved and documented

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

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + execution workflow)
2. **Execute as needed**: Scripts in `scripts/` directory (validation, checkpoints)
3. **Load on demand**: `resources/milestone_checklist.md` (detailed checklist)

Scripts execute without loading into context window, saving tokens while ensuring quality.

## Notes

- This skill is long-running - expect it to take hours or days
- Pause points are built in - you're not locked into finishing
- Sprint plan is the source of truth - but reality may require adjustments
- Git commits create a reversible audit trail
- TodoWrite provides real-time visibility into progress
- Test-driven development is non-negotiable - tests must pass
