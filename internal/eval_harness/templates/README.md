# Agent Prompt Templates

This directory contains customizable templates for agent-based evaluation.

## Files

### agent_prompt.txt

The AILANG prompt template sent to Claude Code when running AILANG benchmarks.

### agent_prompt_python.txt

The Python prompt template sent to Claude Code when running Python benchmarks.

**Placeholders:**
- `{{CAPS}}` - Replaced with comma-separated capabilities (e.g., "IO,FS")
- `{{TIMEOUT}}` - Replaced with timeout in seconds (e.g., "300")
- `{{MAX_ITERATIONS}}` - Replaced with max iterations (e.g., "10")

**How to customize:**
1. Edit `agent_prompt.txt` directly
2. Changes take effect immediately (no recompilation needed)
3. Test your changes with: `ailang eval-suite --agent --benchmarks <test-benchmark>`

**Best practices:**
- Keep instructions clear and concise
- Provide examples of AILANG syntax if needed
- Emphasize the success criteria (output must match exactly)
- Remind the agent to use the tools available (Bash, Read, Write, Edit, Grep)

**Fallback:**
If this file is not found, the code falls back to a hardcoded default template in `agent_prompt.go`.

## Example Customizations

### Add more specific guidance:
```markdown
## Tips

- Start simple: Get basic structure working first
- Use ailang check frequently to catch type errors early
- **NEW**: For list operations, use std/list functions (map, filter, fold)
- **NEW**: For string operations, use show() to convert values to strings
```

### Adjust tone:
```markdown
## Your Task

You are an expert AILANG developer. Your mission:

1. Read README.md carefully - understand what's expected
2. Check syntax_reference.md - AILANG is different from other languages!
3. Write clean, idiomatic AILANG code in solution.ail
...
```

### Add debugging hints:
```markdown
## Common Issues

- **Type error "cannot unify int and float"**: Use intToFloat() or floatToInt()
- **"undefined variable"**: Did you import std/prelude? (auto-imported for main modules)
- **"effect not allowed"**: Add ! {IO} or ! {FS} to function signature
```

## Testing Your Changes

After modifying the template:

```bash
# Run a simple benchmark to verify the prompt works
ailang eval-suite --agent --benchmarks hello_world --models claude-sonnet-4-5

# Check the prompt actually uses your changes (debug mode shows prompts)
DEBUG_EVAL=1 ailang eval-suite --agent --benchmarks hello_world
```

## Version Control

This template should be version controlled. When making significant changes:

1. Document in CHANGELOG.md under the appropriate version
2. Update M-EVAL-AGENT design doc if changing the approach
3. Run pilot benchmarks to validate improvements

## Related Files

- `agent_prompt.go` - Code that loads and processes this template
- `agent_prompt_test.go` - Tests for prompt generation
- `agent_runner.go` - Uses the generated prompt to run headless Claude sessions
