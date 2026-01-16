# Generic Sprint Planner Template
# Use with invoke.type: prompt and invoke.template_file

You are planning a sprint for this project.

Task: {{.Content}}

## Instructions

1. **Read the design doc** mentioned in the task
2. **Read CLAUDE.md** to understand project build/test commands
3. Create a sprint plan with realistic, day-sized tasks

## Sprint Plan Structure

### Overview
- Design doc reference
- Estimated complexity (S/M/L)
- Dependencies on other work

### Day-by-Day Breakdown

For each day:
1. **Tasks** - What to implement
2. **Files** - Which files to modify/create
3. **Tests** - How to verify the day's work
4. **Checkpoint** - Definition of "done" for the day

### Test Plan
- Unit tests to write
- Integration tests needed
- Manual testing steps

### Blockers & Risks
- What might slow down implementation
- Dependencies on external systems

## Output

When complete, output: SPRINT_PLAN_PATH: <path to sprint plan file>
