---
name: AILANG Sprint Planner
description: Analyze design docs, calculate velocity from recent work, and create realistic sprint plans with day-by-day breakdowns. Use when user asks to "plan sprint", "create sprint plan", or wants to estimate development timeline.
---

# AILANG Sprint Planner

Create comprehensive, data-driven sprint plans by analyzing design documentation, current implementation status, and recent velocity.

## Quick Start

**Most common usage:**
```bash
# User says: "Plan the next sprint based on v0.4.0 roadmap"
# This skill will:
# 1. Read design doc (design_docs/planned/v0.4-roadmap.md)
# 2. Analyze CHANGELOG for recent velocity
# 3. Review current implementation status
# 4. Propose realistic milestones with LOC estimates
# 5. Create day-by-day task breakdown
```

## When to Use This Skill

Invoke this skill when:
- User says "plan sprint", "create sprint plan", "plan next phase"
- User asks to estimate timeline for a feature or design doc
- User wants to know how long implementation will take
- User needs to prioritize work for upcoming development

## Documentation URLs

When planning sprints that involve adding error messages, help text, or documentation links:

**Website**: https://sunholo-data.github.io/ailang/

**Documentation Source**: The website documentation lives in this repo at `docs/`
- Markdown files: `docs/docs/` (guides, reference, etc.)
- Static assets: `docs/static/`
- Docusaurus config: `docs/docusaurus.config.js`

**Common Documentation Paths**:
- Language syntax: `/docs/reference/language-syntax`
- Module system: `/docs/guides/module_execution`
- Getting started: `/docs/guides/getting-started`
- REPL guide: `/docs/guides/getting-started#repl`
- Implementation status: `/docs/reference/implementation-status`
- Benchmarking: `/docs/guides/benchmarking`
- Evaluation: `/docs/guides/evaluation/README`

**Full URL Example**:
```
https://sunholo-data.github.io/ailang/docs/reference/language-syntax
```

**Best Practices**:
- When planning features that include documentation links, verify the URLs exist before including them in sprint estimates
- Look in `docs/docs/` to verify the file exists locally
- Use `ls docs/docs/reference/` or `ls docs/docs/guides/` to find available pages

## Available Scripts

### `scripts/analyze_velocity.sh [days]`
Analyze recent development velocity from CHANGELOG and git commits.

**Usage:**
```bash
# Analyze last 7 days (default)
.claude/skills/sprint-planner/scripts/analyze_velocity.sh

# Analyze last 14 days
.claude/skills/sprint-planner/scripts/analyze_velocity.sh 14
```

**Output:**
```
Analyzing velocity for last 7 days...

=== Recent CHANGELOG Entries ===
Total: ~1,200 LOC
Total: ~800 LOC

=== Recent Commits (last 7 days) ===
abc1234 Complete M-DX1.5: Migrate all builtins
def5678 Add Type Builder DSL

=== Files Changed (last 7 days) ===
15 files changed, 1200 insertions(+), 300 deletions(-)

=== Velocity Summary ===
Based on CHANGELOG entries and git history, estimate:
- Average LOC/day from recent milestones
- Typical milestone duration
- Current development pace
```

## Sprint Planning Workflow

### 1. Read and Analyze Design Document

**Input**: Path to design doc (e.g., `design_docs/planned/v0.4-roadmap.md`)

**What to extract:**
- Completed milestones (marked ✅)
- Remaining milestones (marked ❌, ⏳, 📋)
- Target metrics (LOC estimates, timeline, acceptance criteria)
- Dependencies between milestones

### 2. Review Current Implementation Status

**Check these sources:**
- `CHANGELOG.md` - Recent features and LOC counts
- `git log --oneline --since="1 week ago"` - Actual commits
- `make test-coverage-badge` - Current test coverage
- Design doc vs reality - gaps or partial implementations

### 3. Analyze Recent Velocity

**Use the velocity script:**
```bash
.claude/skills/sprint-planner/scripts/analyze_velocity.sh
```

**Calculate:**
- LOC per day from recent milestones
- Average milestone duration
- Actual completion rate vs estimates

### 4. Identify Remaining Work

**List incomplete milestones with:**
- Dependencies (what blocks what)
- Estimated effort (from design doc)
- Priority (based on dependencies, critical path)
- Current velocity (can we realistically do this?)

### 5. Propose Sprint Plan

**Use the template:**
See [`resources/sprint_plan_template.md`](resources/sprint_plan_template.md)

**Include:**
- **Sprint Summary**: Goal, duration, key deliverables
- **Milestone Breakdown**: For each milestone:
  - Name and description
  - Estimated LOC (implementation + tests)
  - Dependencies
  - Acceptance criteria
  - Risk factors
- **Task List**: Day-by-day breakdown (if < 1 week) or weekly (if longer)
- **Success Metrics**: Test coverage target, examples to work, docs to update

### 6. Present for Feedback

**Show user:**
- Proposed milestones with estimates
- Assumptions made
- Areas where input is needed
- Realistic timeline based on actual velocity

**Be ready to revise** based on user priorities or constraints.

### 7. Finalize and Document

**Once approved:**
```bash
# Create sprint plan document
# Naming: M-<type><number>.md (M-P1 for parser, M-T1 for types, etc.)
```

**Include in sprint plan:**
- Goal and motivation
- Technical approach
- Day-by-day implementation plan
- Acceptance criteria
- Estimated LOC
- Dependencies

**Commit:**
```bash
git add design_docs/YYYYMMDD/M-<milestone>.md
git commit -m "Add M-<milestone> sprint plan"
```

## Analysis Framework

### Design Doc Analysis Checklist
- [ ] Current status: What's ✅ vs ❌ vs ⏳
- [ ] Timeline: Days/weeks remaining, velocity metrics
- [ ] Priority matrix: Critical vs nice-to-have
- [ ] Deferred items: Features pushed to later versions
- [ ] Technical debt: Known issues or limitations

### Implementation Analysis Checklist
- [ ] CHANGELOG.md: Recent features, LOC counts, test counts
- [ ] Git history: Actual work done (not just documented)
- [ ] Test files: Coverage, test counts, test patterns
- [ ] Code files: Actual implementation, not just stubs
- [ ] TODO/FIXME: Inline comments about future work
- [ ] Example files: What works vs what's broken

### Gap Analysis Checklist
- [ ] Features in design doc but not implemented
- [ ] Features implemented but not in design doc
- [ ] Estimated LOC vs actual LOC (for velocity)
- [ ] Planned vs actual timeline
- [ ] Test coverage gaps
- [ ] Documentation gaps

## Resources

### Sprint Plan Template
See [`resources/sprint_plan_template.md`](resources/sprint_plan_template.md) for complete sprint plan structure.

## Best Practices

### 1. Be Conservative with Estimates
- Use actual velocity from recent work
- Add 20-30% buffer for unknowns
- Don't promise more than recent velocity suggests

### 2. Prioritize Ruthlessly
- Focus on highest-value items first
- Don't try to do everything in one sprint
- Defer nice-to-haves to future sprints

### 3. Make Tasks Concrete
- ❌ "Implement X" is too vague
- ✅ "Write parser for X syntax (~100 LOC) + 15 test cases" is concrete
- Each task should be achievable in 1 day or less

### 4. Consider Technical Debt
- Don't just add features, also fix issues
- Balance new work with quality improvements
- Factor in time for bug fixes and refactoring

### 5. Plan for Testing
- Every feature needs tests
- Test LOC is usually 30-50% of implementation LOC
- Include test writing in timeline estimates

### 6. Document Assumptions
- Make implicit assumptions explicit
- Note areas of uncertainty
- Highlight where you need user input

## Output Format

See [`resources/sprint_plan_template.md`](resources/sprint_plan_template.md) for full template.

**Key sections:**
- Summary (goal, duration, risk level)
- Current status analysis (completed, velocity, remaining)
- Proposed milestones (with tasks, criteria, risks)
- Success metrics
- Dependencies and open questions

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (YAML frontmatter + workflow)
2. **Execute as needed**: Scripts in `scripts/` (velocity analysis)
3. **Load on demand**: `resources/sprint_plan_template.md` (template)

Scripts execute without loading into context window, saving tokens.

## Notes

- This skill is interactive - expect back-and-forth with user
- Sprint plans should be realistic, not aspirational
- Use actual data (velocity, LOC counts) over guesses
- Update design docs as reality diverges from plan
- Don't commit plan until approved by user
