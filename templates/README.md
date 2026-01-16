# Coordinator Prompt Templates

Generic, reusable templates for coordinator agents. These work with any project.

## Available Templates

| Template | Purpose | Output Marker |
|----------|---------|---------------|
| `design-doc.md` | Create design documents | `DESIGN_DOC_PATH:` |
| `sprint-planner.md` | Plan implementation sprints | `SPRINT_PLAN_PATH:` |
| `sprint-executor.md` | Implement sprint tasks | `IMPLEMENTATION_COMPLETE:` |

## Usage

Reference these templates in your `~/.ailang/config.yaml`:

```yaml
# Option 1: Absolute path to AILANG repo
invoke:
  type: prompt
  template_file: /path/to/ailang/templates/design-doc.md

# Option 2: Copy to ~/.ailang/templates/ and reference from there
invoke:
  type: prompt
  template_file: ~/.ailang/templates/design-doc.md

# Option 3: Copy to your project and use relative path
invoke:
  type: prompt
  template_file: .claude/templates/design-doc.md
```

## Template Variables

Templates can use these variables (resolved at runtime):

| Variable | Description |
|----------|-------------|
| `{{.TaskID}}` | Unique task identifier |
| `{{.GithubIssue}}` | GitHub issue number (if linked) |
| `{{.Content}}` | Task content/message body |
| `{{.Stage}}` | Current workflow stage |
| `{{.OutputMarkers}}` | Expected output markers |

## Customizing for Your Project

These templates are intentionally generic. To customize:

1. **Copy** the template to your project or `~/.ailang/templates/`
2. **Modify** project-specific sections (build commands, paths, conventions)
3. **Reference** your customized version in config

Example customization for a React project:
```markdown
## Verify
- Run `npm run typecheck` after changes
- Run `npm run lint` for style check
- Run `npm test` for unit tests
```

## Best Practices

1. **Keep templates project-agnostic** - Use CLAUDE.md for project specifics
2. **Include clear output markers** - Coordinator needs to detect completion
3. **Add quality checklists** - Help agents remember verification steps
4. **Reference design docs** - Sprint plans should link back to design docs
