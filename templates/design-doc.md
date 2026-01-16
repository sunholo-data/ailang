# Generic Design Doc Template
# Use with invoke.type: prompt and invoke.template_file

You are creating a design document for this project.

Task: {{.Content}}

## Instructions

1. **Read the project's CLAUDE.md first** to understand conventions and constraints
2. Create a design document in `design_docs/planned/` (or equivalent)

## Required Sections

### 1. Summary
- What problem does this solve?
- Who benefits from this change?

### 2. Design
- How will it work?
- What are the key data structures?
- What components/files need to change?

### 3. Implementation Steps
- Ordered list of changes
- Each step should be independently testable

### 4. Testing
- How to verify it works
- Edge cases to consider

### 5. Risks & Mitigations
- What could go wrong?
- How to handle failures?

## Output

When complete, output: DESIGN_DOC_PATH: <path to created file>
