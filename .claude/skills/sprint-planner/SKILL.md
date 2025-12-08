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

## Role in Long-Running Agent Pattern

**sprint-planner acts as the "Initializer" agent** in the two-phase pattern from [Anthropic's long-running agent article](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents):

- **Initializer (sprint-planner)**: Creates infrastructure for execution
  - Analyzes design docs and calculates velocity
  - Creates markdown sprint plan (human-readable)
  - **NEW**: Creates JSON progress file (machine-readable)
  - Sets up session resumption infrastructure
  - Sends handoff message to sprint-executor

- **Coding Agent (sprint-executor)**: Works incrementally across sessions
  - Reads JSON progress file on each session start
  - Updates only `passes` field as milestones complete
  - Can pause/resume work across multiple sessions

This separation enables **multi-session continuity** - sprints can span days or weeks with Claude resuming work from where it left off.

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

### `scripts/create_sprint_json.sh <sprint_id> <sprint_plan_md> [design_doc_md]`
**NEW**: Create structured JSON progress file for multi-session sprint execution.

**Usage:**
```bash
# Create JSON progress file from sprint plan
.claude/skills/sprint-planner/scripts/create_sprint_json.sh \
  "M-S1" \
  "design_docs/planned/v0_4_0/m-s1-sprint-plan.md" \
  "design_docs/planned/v0_4_0/m-s1-parser-improvements.md"
```

**What it does:**
- Creates `.ailang/state/sprints/sprint_<id>.json` with feature list
- Implements "constrained modification" pattern (only `passes` field changes)
- Enables session resumption via structured state
- Provides template for milestone details (to be filled in)

**Output:**
- JSON progress file at `.ailang/state/sprints/sprint_<id>.json`
- Validation check
- Next steps instructions (edit JSON, send handoff message)

**File Organization:**
Sprint JSON files are stored in `.ailang/state/sprints/` to keep the state directory organized.

**Integration with sprint-executor:**
After creating the JSON file, sprint-executor can:
- Resume work across multiple Claude Code sessions
- Track progress programmatically
- Update velocity metrics automatically

## Sprint Planning Workflow

**CRITICAL: Always end by handing off to sprint-executor after user approval!**

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
  - **Example files to create/update** (CRITICAL - required for every new feature)
  - Dependencies
  - Acceptance criteria
  - Risk factors
- **Task List**: Day-by-day breakdown (if < 1 week) or weekly (if longer)
- **Success Metrics**:
  - Test coverage target
  - **Example files created and verified working** (CRITICAL - see CLAUDE.md)
  - Docs to update

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
# Create sprint plan document (markdown - human-readable)
# Naming: M-<type><number>.md (M-P1 for parser, M-T1 for types, etc.)
```

**Include in sprint plan:**
- Goal and motivation
- Technical approach
- Day-by-day implementation plan
- Acceptance criteria
- Estimated LOC
- Dependencies

**NEW: Create JSON progress file (machine-readable):**
```bash
# Create structured progress file for multi-session execution
.claude/skills/sprint-planner/scripts/create_sprint_json.sh \
  "<sprint-id>" \
  "design_docs/planned/vX_Y/<sprint-id>-plan.md" \
  "design_docs/planned/vX_Y/<feature>-design.md"
```

### 8. MANDATORY: Populate JSON with Real Milestones

**The script creates a TEMPLATE - you MUST populate it with real data!**

The `create_sprint_json.sh` script generates placeholder content. **Before handing off to sprint-executor, you MUST edit the JSON file to include actual milestones.**

**Required edits to `.ailang/state/sprints/sprint_<id>.json`:**

1. **Replace placeholder features array** with real milestones:
   ```json
   "features": [
     {
       "id": "M1_ACTUAL_NAME",
       "description": "Real description from your sprint plan",
       "estimated_loc": 150,
       "dependencies": [],
       "acceptance_criteria": [
         "Actual criterion from sprint plan",
         "Another real criterion"
       ],
       "passes": null,
       "started": null,
       "completed": null,
       "notes": null
     }
   ]
   ```

2. **Update velocity estimates** to match your sprint plan:
   ```json
   "velocity": {
     "target_loc_per_day": 150,
     "estimated_total_loc": 670,
     "estimated_days": 4
   }
   ```

**Validation checklist before handoff:**
- [ ] No milestone has `"id": "MILESTONE_ID"` (placeholder)
- [ ] Each milestone has real acceptance criteria (not "Criterion 1")
- [ ] `estimated_total_loc` matches sum of milestone LOC
- [ ] `estimated_days` matches sprint plan duration
- [ ] At least 2 milestones defined

**sprint-executor will REJECT the sprint if placeholders remain!**

### 9. ALWAYS Hand Off to sprint-executor

**CRITICAL: After creating an approved sprint plan, ALWAYS hand off to sprint-executor immediately.**

This is the standard workflow:
1. **sprint-planner** (this skill): Creates plan + JSON progress file
2. **sprint-executor** (implementation agent): Executes the plan with TDD

**Send handoff message:**
```bash
ailang agent send sprint-executor '{
  "type": "plan_ready",
  "correlation_id": "sprint_<sprint-id>_<date>",
  "sprint_id": "<sprint-id>",
  "plan_path": "design_docs/planned/vX_Y/<sprint-id>-plan.md",
  "progress_path": ".ailang/state/sprints/sprint_<id>.json",
  "estimated_duration": "X days (Y hours)",
  "milestones": [
    {"id": "M1", "name": "...", "estimated_hours": X},
    {"id": "M2", "name": "...", "estimated_hours": Y}
  ],
  "discovery": "Key findings from analysis",
  "total_loc_estimate": N,
  "risk_level": "low|medium|high"
}'
```

**Why this workflow?**
- sprint-executor specializes in TDD, continuous linting, progress tracking
- Enables multi-session work (sprints can span days/weeks)
- Proper separation of concerns: planning vs execution

**Optional: Commit before handoff:**
```bash
git add design_docs/YYYYMMDD/M-<milestone>.md
git add .ailang/state/sprints/sprint_<id>.json
git commit -m "Add M-<milestone> sprint plan with JSON progress tracking"
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
- [ ] **Example files: What works vs what's broken** (check `examples/` directory)
- [ ] **Example coverage: Does every new feature have a working example?** (CRITICAL)

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
