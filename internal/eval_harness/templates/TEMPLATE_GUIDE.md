# Agent Prompt Template Customization Guide

## Quick Start

The agent prompt is now **externalized to a template file** for easy customization without code changes.

**Location**: `internal/eval_harness/templates/agent_prompt.txt`

**Edit the file**, save, and your changes take effect immediately - no recompilation needed!

## Template Placeholders

The template uses simple `{{PLACEHOLDER}}` syntax:

| Placeholder | Replaced With | Example |
|-------------|---------------|---------|
| `{{CAPS}}` | Comma-separated capabilities from benchmark | `IO,FS` |
| `{{TIMEOUT}}` | Timeout in seconds | `300` |
| `{{MAX_ITERATIONS}}` | Max agent iterations | `10` |

## Example Customization

### Before (default):
```markdown
## Tips

- Start simple: Get basic structure working first
- Use ailang check frequently to catch type errors early
```

### After (customized):
```markdown
## Tips

- Start simple: Get basic structure working first
- Use ailang check frequently to catch type errors early
- **For list operations**: Use map, filter, fold from std/prelude
- **For string conversion**: Always use show() builtin
- **Common mistake**: Don't forget effect signatures (! {IO})
```

## Testing Your Changes

After editing `agent_prompt.txt`:

```bash
# Test with a simple benchmark
ailang eval-suite --agent --benchmarks hello_world --models claude-sonnet-4-5

# Check the prompt is being used (look for your custom text in agent output)
DEBUG_EVAL=1 ailang eval-suite --agent --benchmarks hello_world
```

## Fallback Behavior

If `agent_prompt.txt` is missing or unreadable:
- Code falls back to hardcoded default template in `agent_prompt.go`
- No crash - evaluation continues with reasonable defaults

## Best Practices

1. **Keep it concise**: Claude Code processes the prompt token-by-token
2. **Be specific**: Vague instructions lead to vague code
3. **Test changes**: Run a pilot benchmark after significant edits
4. **Version control**: Commit template changes with descriptive messages
5. **Document**: If adding new sections, explain why in commit message

## Advanced: Multiple Templates

Want different prompts for different benchmark types?

Currently not supported, but could be added:
- `agent_prompt_simple.txt` - For basic benchmarks
- `agent_prompt_complex.txt` - For complex algorithmic challenges
- `agent_prompt_io.txt` - For IO-heavy benchmarks

To request this feature, open a GitHub issue or update the M-EVAL-AGENT design doc.

## Related Files

- `agent_prompt.txt` - The template (edit this!)
- `agent_prompt.go` - Code that loads and processes template
- `agent_prompt_test.go` - Tests for prompt generation
- `README.md` - Overview of all templates

## Questions?

See:
- `design_docs/planned/M-EVAL-AGENT.md` - Architecture and rationale
- `internal/eval_harness/templates/README.md` - Template directory overview
