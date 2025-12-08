# Claude Code Headless CLI Reference

Complete reference for `claude` command-line flags and options.

## Command Syntax

```bash
claude [options] [prompt]
claude -p "prompt" [options]
claude --resume SESSION_ID "prompt" [options]
claude --continue "prompt" [options]
```

## Global Options

### `--print, -p <prompt>`
Run command in headless mode (non-interactive).

**Example:**
```bash
claude -p "Analyze this codebase"
```

### `--output-format <format>`
Specify output format: `text`, `json`, or `stream-json`.

**Default:** `text`

**Example:**
```bash
claude -p "Task" --output-format json
```

**JSON Structure:**
```json
{
  "session_id": "string",
  "result": "string",
  "status": "success" | "error",
  "error": "string (if status=error)",
  "cost": number,
  "duration": number,
  "tokensUsed": {
    "input": number,
    "output": number
  },
  "artifacts": ["string"]
}
```

### `--allowedTools <tools>`
Comma-separated list of allowed tools.

**Common tools:** `Bash`, `Read`, `Write`, `Edit`, `Grep`, `Glob`, `WebFetch`, `WebSearch`

**Special:** `*` allows all tools (use with caution)

**Example:**
```bash
claude -p "Task" --allowedTools "Bash,Read,Write"
```

### `--permission-mode <mode>`
How to handle permissions for file operations.

**Options:**
- `ask` - Prompt for each operation (default in interactive)
- `acceptEdits` - Auto-accept file edits
- `acceptAll` - Auto-accept all operations

**Example:**
```bash
claude -p "Refactor code" --permission-mode acceptEdits
```

### `--verbose, -v`
Enable detailed logging.

**Example:**
```bash
claude -p "Task" --verbose
```

### `--resume <session-id>`
Resume previous conversation session.

**Example:**
```bash
claude --resume abc123def456 "Continue task"
```

### `--continue`
Continue most recent session.

**Example:**
```bash
claude --continue "Next step"
```

### `--max-tokens <number>`
Maximum tokens for response (model-dependent).

**Example:**
```bash
claude -p "Task" --max-tokens 4000
```

### `--temperature <number>`
Sampling temperature (0.0-1.0). Higher = more creative.

**Default:** Model-dependent (usually ~1.0)

**Example:**
```bash
claude -p "Brainstorm ideas" --temperature 1.5
```

### `--timeout <seconds>`
Maximum time to wait for response.

**Example:**
```bash
claude -p "Long task" --timeout 300
```

## Configuration Loading

### Project Settings

Claude automatically loads from `.claude/` directory:

```
.claude/
├── settings.json           # Shared team settings
├── settings.local.json     # Personal settings (git-ignored)
├── agents/                 # Project-specific agents
├── skills/                 # Project-specific skills
└── commands/               # Slash commands
```

### Settings Precedence (highest to lowest)

1. Command-line flags
2. `.claude/settings.local.json`
3. `.claude/settings.json`
4. `~/.claude/settings.json` (user global)
5. Defaults

### `--config <path>`
Load custom settings file.

**Example:**
```bash
claude -p "Task" --config custom-settings.json
```

### `--mcp-config <path>`
Load MCP (Model Context Protocol) server config.

**Example:**
```bash
claude -p "Task" --mcp-config .mcp.json
```

## Session Management

### Session IDs

Every invocation creates or resumes a session with unique ID.

**Extract from JSON:**
```bash
SESSION_ID=$(claude -p "Task" --output-format json | jq -r '.session_id')
```

**Resume session:**
```bash
claude --resume $SESSION_ID "Continue"
```

### Session Storage

Sessions stored in:
```
~/.claude/sessions/
└── <session-id>/
    ├── messages.json
    ├── context.json
    └── metadata.json
```

### List Sessions

```bash
ls -lt ~/.claude/sessions/ | head -10
```

### Clean Old Sessions

```bash
# Remove sessions older than 7 days
find ~/.claude/sessions/ -type d -mtime +7 -exec rm -rf {} +
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 130 | Interrupted (Ctrl+C) |

## Environment Variables

### `CLAUDE_API_KEY`
API key for Claude (if not in config).

### `CLAUDE_CONFIG_DIR`
Override config directory (default: `~/.claude`).

### `CLAUDE_LOG_LEVEL`
Logging level: `debug`, `info`, `warn`, `error`.

### `CLAUDE_NO_COLOR`
Disable colored output.

## Advanced Usage

### Piping Input

```bash
cat prompt.txt | claude -p "$(cat)"
```

### Combining with jq

```bash
# Extract specific field
claude -p "Task" --output-format json | jq -r '.result'

# Check status
if claude -p "Task" --output-format json | jq -e '.status == "success"'; then
  echo "Success!"
fi
```

### Parallel Execution

```bash
# Start multiple tasks
claude -p "Task A" --output-format json > taskA.json &
claude -p "Task B" --output-format json > taskB.json &
wait

# Combine results
jq -s '.' taskA.json taskB.json
```

### Timeout with Fallback

```bash
if timeout 60 claude -p "Quick task" --output-format json > result.json; then
  echo "Completed in time"
else
  echo "Timed out, using fallback"
  # Fallback logic
fi
```

## Tool Permissions Reference

### Read-Only Tools
Safe for analysis, no side effects:
- `Read` - Read files
- `Grep` - Search file contents
- `Glob` - Find files by pattern

### Write Tools
Modify files (use with caution):
- `Write` - Create/overwrite files
- `Edit` - Edit existing files

### Execution Tools
Run commands (carefully review permissions):
- `Bash` - Execute shell commands

### External Tools
Network access:
- `WebFetch` - Fetch URLs
- `WebSearch` - Search the web

### Special Tools
- `Task` - Launch sub-agents
- `Skill` - Execute skills
- `SlashCommand` - Run slash commands

## Common Patterns

### Minimal Permissions
```bash
# Read-only analysis
claude -p "Analyze code" --allowedTools "Read,Grep,Glob"
```

### Development Workflow
```bash
# Can read, write, and run tests
claude -p "Implement feature" --allowedTools "Bash,Read,Write,Edit"
```

### CI/CD Safe
```bash
# Read, analyze, run commands, but no file writes
claude -p "Run tests" --allowedTools "Bash,Read,Grep"
```

### Full Automation
```bash
# All tools (use only in trusted environments)
claude -p "Deploy" --allowedTools "*" --permission-mode acceptAll
```

## Troubleshooting

### "Command not found: claude"
- Install Claude Code CLI
- Check PATH: `echo $PATH | grep claude`
- Verify installation: `which claude`

### "Invalid JSON output"
- Add `2>/dev/null` to suppress stderr
- Check for errors: `claude -p "Task" 2>&1 | tee output.log`
- Validate JSON: `jq . output.json`

### "Session not found"
- Session may have expired
- Check session directory: `ls ~/.claude/sessions/`
- Start new session without `--resume`

### "Permission denied"
- Check `--permission-mode` setting
- Use `acceptEdits` for automation
- Review tool permissions with `--allowedTools`

### "Timeout"
- Increase with `--timeout`
- Use `stream-json` for progress updates
- Split into smaller tasks

## See Also

- [Examples](examples.md) - Comprehensive workflow examples
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
- [SKILL.md](../SKILL.md) - Main skill documentation
