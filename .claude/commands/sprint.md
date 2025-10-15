---
description: Execute a sprint plan with progress tracking, testing, and documentation updates
allowed-tools:
  - Read
  - Edit
  - Write
  - Bash(make:*)
  - Bash(git:*)
  - Bash(go:*)
  - TodoWrite
---

# Sprint Command

Execute an approved sprint plan with continuous progress tracking, testing, linting, and documentation updates.

**Usage:** `/sprint <sprint-plan-path>`

**Example:** `/sprint @design_docs/20251002/M-S1.md`

## Core Principles

1. **Test-Driven**: All code must pass tests before moving to next milestone
2. **Lint-Clean**: All code must pass linting before moving to next milestone
3. **Document as You Go**: Update CHANGELOG.md and sprint plan progressively
4. **Pause for Breath**: Stop at natural breakpoints for review and approval
5. **Track Everything**: Use TodoWrite to maintain visible progress

## Execution Flow

### Phase 1: Initialize Sprint

1. **Read Sprint Plan**
   - Parse sprint plan document
   - Extract all milestones and tasks
   - Note dependencies and acceptance criteria
   - Identify estimated LOC and duration

2. **Validate Prerequisites**
   - Ensure current tests pass: `make test`
   - Ensure current linting passes: `make lint`
   - Check working directory is clean: `git status --short`
   - Verify branch is correct (usually `dev`)

3. **Create Todo List**
   - Use TodoWrite to create tasks for all milestones
   - Mark first milestone as `in_progress`
   - Keep remaining tasks as `pending`

4. **Initial Status Update**
   - Update sprint plan with "🔄 In Progress" status
   - Add start timestamp
   - Commit sprint plan update

### Phase 2: Execute Milestones

For each milestone in the sprint:

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
   - Run tests: `make test`
   - **CRITICAL**: If tests fail, fix immediately before proceeding
   - Run linting: `make lint`
   - **CRITICAL**: If linting fails, fix immediately before proceeding
   - Check formatting: `make fmt-check` or run `make fmt`

#### Step 5: Update Documentation
   - Update CHANGELOG.md with milestone completion:
     - What was implemented
     - LOC counts (implementation + tests)
     - Key design decisions
     - Files modified/created
   - Update sprint plan with completion status (✅)
   - Add metrics (actual LOC vs estimated, time spent)

#### Step 6: Pause for Breath
   After each milestone:
   - Show summary of what was completed
   - Show current sprint progress (X of Y milestones done)
   - Show velocity (LOC/day vs planned)
   - Ask user: "Ready to continue to next milestone?" or "Need to review/adjust?"
   - If user says "pause" or "stop", save current state and exit gracefully

### Phase 3: Finalize Sprint

When all milestones are complete:

1. **Final Testing**
   - Run full test suite: `make test`
   - Run linting: `make lint`
   - Run coverage check: `make test-coverage-badge`
   - Verify all examples work (if applicable)

2. **Documentation Review**
   - Verify CHANGELOG.md is complete
   - Verify sprint plan shows all milestones as ✅
   - Update sprint plan with final metrics:
     - Total LOC (actual vs estimated)
     - Total time (actual vs estimated)
     - Velocity achieved
     - Test coverage achieved
     - Any deviations from plan

3. **Final Commit**
   - Commit sprint plan with completion status
   - Commit CHANGELOG.md if not already committed
   - Add sprint completion message:
     ```
     Complete sprint: <sprint-name>

     Milestones completed:
     - <Milestone 1>: <LOC>
     - <Milestone 2>: <LOC>

     Total: <actual-LOC> LOC in <actual-time>
     Velocity: <LOC/day>
     Test coverage: <percentage>
     ```

4. **Summary Report**
   - Show sprint completion summary
   - Compare planned vs actual (LOC, time, milestones)
   - Highlight any issues or deviations
   - Suggest next steps (new sprint, release, etc.)

5. **Identify bumps**
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
- If tests fail: Show output, ask how to fix, don't proceed
- If linting fails: Show output, ask how to fix, don't proceed
- If implementation unclear: Ask for clarification, don't guess
- If milestone takes much longer than estimated: Pause and reassess

### Velocity Tracking
Calculate and display velocity:
- After each milestone: LOC completed / time spent
- Compare with sprint plan estimates
- Adjust remaining estimates if velocity differs significantly
- Warn if sprint is falling behind schedule

## Sprint Plan Updates

Update sprint plan document with these markers:

### Before Starting
```markdown
**Status**: 🔄 In Progress (Started: 2025-10-02)
```

### After Each Milestone
```markdown
### ✅ Milestone 1 Complete (October 2, 2025) - <Name> (~<actual-LOC> LOC)

**Estimated**: <est-LOC> LOC in <est-days> days
**Actual**: <actual-LOC> LOC in <actual-hours> hours

**What Was Built:**
- <Feature/component>
- <Test coverage>
- <Key decisions>

**Velocity**: <LOC/day> (vs planned: <planned-LOC/day>)
```

### After Sprint Completion
```markdown
**Status**: ✅ Complete (Finished: 2025-10-02)

**Sprint Summary:**
- Total LOC: <actual> (planned: <estimated>)
- Duration: <actual-days> days (planned: <estimated-days>)
- Velocity: <LOC/day> (planned: <planned-LOC/day>)
- Test coverage: <percentage>
- Milestones: <completed>/<total>
```

## CHANGELOG Updates

Add entries progressively as milestones complete:

```markdown
## [Unreleased] - 2025-10-02

### Added - <Sprint Name>: <Milestone Name>

**<Brief description>**

**Implementation** (`<files>`)
- <Feature 1>
- <Feature 2>
- ...

**Tests** (`<test-files>`)
- <Test coverage>
- X test cases added
- All tests passing ✅

**Files Modified:**
- `<file1>` (+X LOC)
- `<file2>` (+Y LOC)
- Total: ~<total> LOC

**Key Decisions:**
1. <Decision and rationale>
2. <Decision and rationale>

---
```

## Example Execution

```bash
# User initiates sprint
/sprint @design_docs/20251002/M-S1.md

# AI reads plan: "Stdlib Implementation - 2 milestones, ~600 LOC"

# AI initializes
TodoWrite: [
  { content: "String operations", status: "in_progress" },
  { content: "List operations", status: "pending" }
]

# AI implements Milestone 1
# - Writes internal/stdlib/string.go (~200 LOC)
# - Writes internal/stdlib/string_test.go (~100 LOC)
# - Runs: make test (✅)
# - Runs: make lint (✅)
# - Updates CHANGELOG.md
# - Updates sprint plan (✅ Milestone 1)
# - Commits

# AI pauses
"Milestone 1 complete! String operations implemented (300 LOC).
Sprint progress: 1/2 milestones done.
Ready to continue to Milestone 2 (List operations)?"

# User: "yes, continue"

# AI implements Milestone 2
# ... (same process)

# AI completes sprint
"Sprint complete! Both milestones done (610 LOC total).
All tests passing ✅
All linting passing ✅
Test coverage: 32.1%

Summary:
- Estimated: 600 LOC in 2 days
- Actual: 610 LOC in 1.8 days
- Velocity: 339 LOC/day (vs planned: 300 LOC/day)

Ready for next steps!"
```

## Best Practices

### 1. Commit Granularity
- Commit after each milestone (not after each task)
- Keep commits focused and atomic
- Write clear commit messages with context

### 2. Test Coverage
- Every function should have tests
- Test both success and error cases
- Aim for >80% coverage on new code

### 3. Documentation
- Update CHANGELOG.md as you go (not at the end)
- Keep sprint plan in sync with reality
- Document design decisions inline

### 4. Code Quality
- Follow existing patterns in codebase
- Keep functions small (<50 lines)
- Add comments for complex logic
- Use descriptive variable names

### 5. Velocity Management
- If falling behind: Simplify remaining tasks or cut scope
- If ahead of schedule: Consider adding polish or tests
- Always finish with passing tests and clean lint

### 6. Communication
- Pause frequently for user feedback
- Don't make assumptions - ask questions
- Show progress clearly with TodoWrite
- Be transparent about challenges

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

## Output Format

Throughout sprint execution, maintain this structure:

```
🚀 Sprint: <Name>
📋 Milestone: <X>/<Total> - <Current Milestone Name>
⏱️ Progress: <LOC-done>/<LOC-total> LOC
✅ Tests: Passing
✅ Lint: Clean

Current Task: <What I'm doing right now>
```

Update this status before each major action.

## Notes

- This command is long-running - expect it to take hours or days
- Pause points are built in - you're not locked into finishing
- Sprint plan is the source of truth - but reality may require adjustments
- Git commits create a reversible audit trail
- TodoWrite provides real-time visibility into progress
