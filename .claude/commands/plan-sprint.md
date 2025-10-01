---
description: Create a sprint plan by reviewing design docs, implementation status, and proposing next steps
allowed-tools:
  - Read
  - Bash(git:*)
  - Bash(grep:*)
  - Bash(find:*)
  - Bash(wc:*)
  - Write
---

# Plan Sprint Command

Create a comprehensive sprint plan by analyzing design documentation, current implementation status, and proposing actionable next steps.

**Usage:** `/plan-sprint <design-doc-path>`

**Example:** `/plan-sprint @design_docs/20250929/v0_1_0_mvp_roadmap.md`

## Steps to Perform

### 1. **Read and Analyze Design Document**
   - Read the specified design document path
   - Identify completed milestones (marked with ✅)
   - Identify remaining milestones (marked with ❌, ⏳, or 📋)
   - Extract target metrics (LOC estimates, timeline, acceptance criteria)
   - Note any dependencies between milestones

### 2. **Review Current Implementation Status**
   - Read CHANGELOG.md to understand what was recently completed
   - Check git log for recent commits: `git log --oneline --since="1 week ago" | head -20`
   - Review test coverage if mentioned: `make test-coverage-badge` or grep for coverage %
   - Identify any gaps between design doc and changelog

### 3. **Code Review for Implementation Reality**
   - For recently completed features, verify implementation exists:
     - Check for new files mentioned in changelog
     - Verify test files exist: `find internal/ -name "*_test.go" -newer <reference-file>`
     - Check actual LOC vs estimated: `wc -l <files>`
   - Identify partial implementations (code exists but tests missing, etc.)
   - Note any technical debt or known issues

### 4. **Analyze Remaining Work**
   - List incomplete milestones from design doc
   - Prioritize based on:
     - Dependencies (what blocks what)
     - Estimated effort (from design doc)
     - Current velocity (LOC per day from recent milestones)
     - Critical path items
   - Identify any new issues discovered during implementation

### 5. **Propose Sprint Plan**
   Create a structured proposal with:

   **Sprint Summary:**
   - Sprint goal (1-2 sentences)
   - Duration estimate (days/weeks)
   - Key deliverables

   **Milestone Breakdown:**
   For each proposed milestone:
   - Name and brief description
   - Estimated LOC (implementation + tests)
   - Dependencies (what must be done first)
   - Acceptance criteria (how we know it's done)
   - Risk factors (what could go wrong)

   **Task List:**
   - Day-by-day breakdown if sprint is < 1 week
   - Weekly breakdown if longer
   - Each task should be concrete and achievable

   **Success Metrics:**
   - Test coverage target
   - Examples that should work
   - Documentation to update
   - Any other measurable outcomes

### 6. **Present for Feedback**
   - Show the proposed plan
   - Highlight any assumptions made
   - Point out areas where input is needed
   - Be ready to revise based on feedback

### 7. **Finalize and Document**
   Once plan is accepted:
   - Create a new design doc in `design_docs/<date>/M-<milestone>.md`
   - Follow naming convention: `M-P<number>` for parser, `M-T<number>` for types, etc.
   - Include:
     - Goal and motivation
     - Technical approach
     - Implementation plan (day by day)
     - Acceptance criteria
     - Estimated LOC
     - Dependencies
   - Commit the design doc: `git add design_docs/... && git commit -m "Add M-<milestone> sprint plan"`

## Analysis Framework

### Design Doc Analysis
Look for these sections:
- **Current Status** - What's marked as complete (✅) vs incomplete (❌, ⏳)
- **Timeline** - Days/weeks remaining, velocity metrics
- **Priority Matrix** - What's critical vs nice-to-have
- **Deferred Items** - Features explicitly pushed to later versions
- **Technical Debt** - Known issues or limitations

### Implementation Analysis
Check these sources:
- **CHANGELOG.md** - Recent features, LOC counts, test counts
- **Git History** - Actual work done (not just documented)
- **Test Files** - Coverage, test counts, test patterns
- **Code Files** - Actual implementation, not just stubs
- **TODO/FIXME** - Inline comments about future work
- **Example Files** - What works vs what's broken

### Gap Analysis
Identify discrepancies:
- Features in design doc but not implemented
- Features implemented but not in design doc
- Estimated LOC vs actual LOC (for velocity calculation)
- Planned vs actual timeline
- Test coverage gaps
- Documentation gaps

## Output Format

```markdown
# Sprint Plan: <Milestone Name>

## Summary
<1-2 sentence goal>

**Duration:** X days
**Dependencies:** <List>
**Risk Level:** Low/Medium/High

## Current Status Analysis

### Completed Recently
- ✅ <Feature>: <LOC> in <days>
- ✅ <Feature>: <LOC> in <days>

### Velocity
- Recent average: <LOC/day>
- Estimated capacity: <LOC> for this sprint

### Remaining from Design Doc
- ⏳ <Milestone>: <estimated LOC>
- 📋 <Milestone>: <estimated LOC>

## Proposed Milestones

### Milestone 1: <Name>
**Goal:** <Description>
**Estimated:** <LOC> implementation + <LOC> tests = <total LOC>
**Duration:** <days>

**Tasks:**
- Day 1: <Task>
- Day 2: <Task>
...

**Acceptance Criteria:**
- [ ] <Criteria>
- [ ] <Criteria>

**Risks:**
- <Risk> - Mitigation: <Strategy>

### Milestone 2: <Name>
...

## Success Metrics
- Test coverage: >X%
- Examples passing: Y+
- Documentation: <List>
- All tests passing: ✅

## Dependencies
- <External dependency>
- <Blocking milestone>

## Open Questions
- <Question> - Need input on: <details>
```

## Best Practices

1. **Be Conservative with Estimates**
   - Use actual velocity from recent work
   - Add 20-30% buffer for unknowns
   - Don't promise more than recent velocity suggests

2. **Prioritize Ruthlessly**
   - Focus on highest-value items first
   - Don't try to do everything in one sprint
   - Defer nice-to-haves to future sprints

3. **Make Tasks Concrete**
   - "Implement X" is too vague
   - "Write parser for X syntax (~100 LOC) + 15 test cases" is concrete
   - Each task should be achievable in 1 day or less

4. **Consider Technical Debt**
   - Don't just add features, also fix issues
   - Balance new work with quality improvements
   - Factor in time for bug fixes and refactoring

5. **Plan for Testing**
   - Every feature needs tests
   - Test LOC is usually 30-50% of implementation LOC
   - Include test writing in timeline estimates

6. **Document Assumptions**
   - Make implicit assumptions explicit
   - Note areas of uncertainty
   - Highlight where you need input

## Example Usage

```bash
# User provides design doc to analyze
/plan-sprint @design_docs/20250929/v0_1_0_mvp_roadmap.md

# AI analyzes current status
# - Reads roadmap: M-P4 complete, M-P5 next
# - Checks changelog: M-P4 was ~1,060 LOC in 3 days
# - Calculates velocity: ~350 LOC/day
# - Reviews remaining work in roadmap

# AI proposes sprint plan
# - Milestone: Stdlib Implementation
# - Estimated: ~600 LOC over 2 days
# - Breaking down into tasks

# User reviews and provides feedback
User: "Let's split this into 2 milestones and add more buffer time"

# AI revises plan based on feedback

# Once approved:
# AI creates design_docs/20251002/M-S1.md with finalized plan
```

## Notes

- This command is interactive - expect back-and-forth
- Sprint plans should be realistic, not aspirational
- Use actual data (velocity, LOC counts) over guesses
- Update design docs as reality diverges from plan
- Don't commit the plan until it's approved by user
