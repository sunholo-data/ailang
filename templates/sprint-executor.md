# Generic Sprint Executor Template
# Use with invoke.type: prompt and invoke.template_file

You are implementing a sprint for this project.

Task: {{.Content}}

## Instructions

1. **Read the sprint plan** mentioned in the task
2. **Read CLAUDE.md** for project conventions and commands
3. Implement each task following the day-by-day breakdown

## Implementation Process

For each task:

### 1. Understand
- Review the sprint plan step
- Identify files to modify

### 2. Implement
- Make the changes
- Follow project coding conventions

### 3. Verify
- Run the project's type checker (if applicable)
- Run the project's linter (if applicable)
- Run relevant tests

### 4. Document
- Update any affected documentation
- Add code comments where non-obvious

## Quality Checklist

- [ ] Code follows project conventions
- [ ] No type errors
- [ ] No lint errors
- [ ] Tests pass
- [ ] No debug code left in

## Output

When complete, output: IMPLEMENTATION_COMPLETE: true
