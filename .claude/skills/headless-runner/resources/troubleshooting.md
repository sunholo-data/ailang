# Headless Claude Troubleshooting

Common issues and solutions when running Claude Code in headless mode.

## Installation Issues

### "Command not found: claude"

**Problem**: `claude` command not in PATH.

**Solutions**:
```bash
# Check if installed
which claude

# Check PATH
echo $PATH | grep -o '[^:]*claude[^:]*'

# Add to PATH (bash)
echo 'export PATH="$HOME/.claude/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Add to PATH (zsh)
echo 'export PATH="$HOME/.claude/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Verify
claude --version
```

### "Permission denied: claude"

**Problem**: Binary not executable.

**Solution**:
```bash
chmod +x ~/.claude/bin/claude
# or wherever claude is installed
chmod +x /usr/local/bin/claude
```

## Authentication Issues

### "Authentication failed"

**Problem**: Missing or invalid API key.

**Solutions**:
```bash
# Check API key is set
echo $ANTHROPIC_API_KEY

# Set API key (temporary)
export ANTHROPIC_API_KEY="sk-ant-..."

# Set API key (permanent - bash)
echo 'export ANTHROPIC_API_KEY="sk-ant-..."' >> ~/.bashrc
source ~/.bashrc

# Set API key (permanent - zsh)
echo 'export ANTHROPIC_API_KEY="sk-ant-..."' >> ~/.zshrc
source ~/.zshrc

# Or configure in settings
cat > ~/.claude/settings.json <<EOF
{
  "anthropicApiKey": "sk-ant-..."
}
EOF
```

### "Rate limit exceeded"

**Problem**: Too many requests.

**Solutions**:
```bash
# Add delay between requests
sleep 2  # Wait 2 seconds

# Implement exponential backoff
RETRY_COUNT=0
MAX_RETRIES=5

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  if claude -p "Task" --output-format json > result.json 2>&1; then
    break
  fi

  if grep -q "rate limit" result.json; then
    WAIT_TIME=$((2 ** RETRY_COUNT))
    echo "Rate limited, waiting ${WAIT_TIME}s..."
    sleep $WAIT_TIME
    RETRY_COUNT=$((RETRY_COUNT + 1))
  else
    echo "Non-rate-limit error"
    exit 1
  fi
done
```

## Configuration Issues

### "Settings not loading"

**Problem**: `.claude/` configuration not found.

**Check**:
```bash
# Verify you're in project directory
pwd
ls -la .claude/

# Check settings file exists
ls -la .claude/settings.json .claude/settings.local.json

# Test if config loads
claude -p "List all available skills" | grep -i "skill-builder"
```

**Solution**:
```bash
# Run from correct directory
cd /path/to/your/project
claude -p "Task"

# Or specify config directory
export CLAUDE_CONFIG_DIR=/path/to/project/.claude
claude -p "Task"
```

### "Skills/agents not available"

**Problem**: Skills not loading in headless mode.

**Debug**:
```bash
# Test skill loading
claude -p "List all available skills, agents, and slash commands" > skills_test.txt
cat skills_test.txt | grep -i "skill-builder"

# Check skill directory
ls -R .claude/skills/

# Verify YAML frontmatter
head -5 .claude/skills/*/SKILL.md
```

**Solution**:
```bash
# Ensure skills have proper YAML frontmatter
cat > .claude/skills/my-skill/SKILL.md <<EOF
---
name: My Skill
description: What it does. Use when user asks for X.
---
# My Skill
...
EOF
```

## Output Format Issues

### "Invalid JSON output"

**Problem**: JSON parsing fails.

**Debug**:
```bash
# Capture raw output
claude -p "Task" --output-format json 2>&1 | tee raw_output.txt

# Validate JSON
jq . raw_output.txt

# Check for stderr mixed in
claude -p "Task" --output-format json 2>/dev/null > clean_output.json
jq . clean_output.json
```

**Solutions**:
```bash
# ✅ Redirect stderr to /dev/null
RESULT=$(claude -p "Task" --output-format json 2>/dev/null)

# ✅ Check if valid JSON before parsing
if echo "$RESULT" | jq -e '.' >/dev/null 2>&1; then
  SESSION_ID=$(echo "$RESULT" | jq -r '.session_id')
else
  echo "Invalid JSON response"
  echo "$RESULT"
  exit 1
fi

# ✅ Use safe extraction with default
SESSION_ID=$(echo "$RESULT" | jq -r '.session_id // "unknown"')
```

### "Missing fields in JSON output"

**Problem**: Expected fields not present.

**Debug**:
```bash
# Inspect full JSON structure
claude -p "Task" --output-format json | jq .

# Check for specific field
claude -p "Task" --output-format json | jq -e '.fieldName'
```

**Solution**:
```bash
# Use safe field extraction with defaults
STATUS=$(echo "$RESULT" | jq -r '.status // "unknown"')
ERROR=$(echo "$RESULT" | jq -r '.error // "No error message"')
COST=$(echo "$RESULT" | jq -r '.cost // 0')
```

## Tool Permission Issues

### "Tool not allowed"

**Problem**: Tool not in allowedTools list.

**Solution**:
```bash
# Specify all needed tools
claude -p "Task" --allowedTools "Bash,Read,Write,Grep,Glob"

# For automation with file operations
claude -p "Task" --allowedTools "Bash,Read,Write,Edit"

# Allow all (use with caution)
claude -p "Task" --allowedTools "*"
```

### "Permission denied for file operations"

**Problem**: `--permission-mode` not set for automation.

**Solution**:
```bash
# Accept file edits automatically
claude -p "Refactor code" \
  --allowedTools "Read,Edit,Write" \
  --permission-mode acceptEdits

# Accept all operations (use carefully)
claude -p "Full automation" \
  --allowedTools "*" \
  --permission-mode acceptAll
```

## Session Management Issues

### "Session not found"

**Problem**: Session ID invalid or expired.

**Debug**:
```bash
# List recent sessions
ls -lt ~/.claude/sessions/ | head -10

# Check if session exists
SESSION_ID="abc123..."
ls ~/.claude/sessions/$SESSION_ID/
```

**Solution**:
```bash
# Start new session instead of resuming
claude -p "New task" --output-format json

# Or extract session from previous result
SESSION_ID=$(cat previous_result.json | jq -r '.session_id')
claude --resume $SESSION_ID "Continue"
```

### "Context lost in resumed session"

**Problem**: Previous context not available.

**Workaround**:
```bash
# Explicitly provide context when resuming
claude --resume $SESSION_ID "Continue task X. Previously we were doing Y..."

# Or start fresh session with full context
claude -p "Complete task. Context: [provide full context]"
```

## Timeout Issues

### "Command hangs indefinitely"

**Problem**: No timeout set for long-running tasks.

**Solution**:
```bash
# Use system timeout command
timeout 300 claude -p "Long task" --output-format json

# Or set in Claude CLI (if supported)
claude -p "Long task" --timeout 300 --output-format json

# With error handling
if timeout 300 claude -p "Task" --output-format json > result.json; then
  echo "Completed successfully"
else
  EXIT_CODE=$?
  if [ $EXIT_CODE -eq 124 ]; then
    echo "Timed out after 300s"
  else
    echo "Failed with exit code $EXIT_CODE"
  fi
fi
```

## AILANG Agent Messaging Issues

### "Message not appearing in inbox"

**Problem**: Sent message not received.

**Debug**:
```bash
# Check message was sent successfully
ailang agent send my-agent '{"task": "test"}' && echo "Sent!"

# Verify inbox directory exists
ls -la .ailang/state/messages/my-agent/

# Check _unread directory
ls -la .ailang/state/messages/my-agent/_unread/

# Check _pending.json for real-time messages
cat .ailang/state/messages/my-agent/.pending.json
```

**Solutions**:
```bash
# Ensure agent ID is correct
ailang agent inbox my-agent  # Not my_agent or my-agent-1

# Check for typos in agent ID
ls .ailang/state/messages/  # List all agent inboxes

# Send with explicit from
ailang agent send --to-user --from "sender-name" '{"test": "message"}'
```

### "Cannot acknowledge message"

**Problem**: `ailang agent ack` fails.

**Debug**:
```bash
# Verify message ID format
MESSAGE_ID="msg_20251025_155729_a5f3e77ee975"
echo $MESSAGE_ID | grep -E '^msg_[0-9]{8}_[0-9]{6}_[a-f0-9]+$'

# Check message exists in _unread
ls .ailang/state/messages/my-agent/_unread/$MESSAGE_ID.json

# Try with --verbose (if available)
ailang agent ack $MESSAGE_ID --verbose
```

**Solutions**:
```bash
# Ensure correct message ID (no spaces, exact format)
MESSAGE_ID=$(ailang agent inbox --unread-only my-agent | grep "ID:" | head -1 | awk '{print $2}')
ailang agent ack $MESSAGE_ID

# Check file permissions
chmod -R u+w .ailang/state/messages/

# Acknowledge all if specific ID fails
ailang agent ack --all
```

### "Flags in wrong order"

**Problem**: `ailang agent inbox my-agent --unread-only` fails.

**Solution**:
```bash
# ❌ WRONG - Flags after agent ID
ailang agent inbox my-agent --unread-only

# ✅ CORRECT - Flags before agent ID
ailang agent inbox --unread-only my-agent
```

## Performance Issues

### "Headless mode is slow"

**Problem**: High latency or slow responses.

**Solutions**:
```bash
# Use smaller model (if option exists)
claude -p "Task" --model claude-haiku-3-5

# Reduce max tokens
claude -p "Task" --max-tokens 1000

# Run tasks in parallel
claude -p "Task A" --output-format json > taskA.json &
claude -p "Task B" --output-format json > taskB.json &
wait

# Use streaming for real-time updates
claude -p "Long task" --output-format stream-json | while read -r line; do
  echo "$line" | jq -r '.event'
done
```

### "Too many sessions consuming disk"

**Problem**: Session directory growing large.

**Solution**:
```bash
# Clean old sessions (older than 7 days)
find ~/.claude/sessions/ -type d -mtime +7 -exec rm -rf {} +

# Check session directory size
du -sh ~/.claude/sessions/

# Limit session retention in settings
cat > ~/.claude/settings.json <<EOF
{
  "sessionRetentionDays": 7
}
EOF
```

## CI/CD Issues

### "API key not available in CI"

**Problem**: Environment variable not set in CI pipeline.

**Solution for GitHub Actions**:
```yaml
- name: Run headless Claude
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: |
    claude -p "Task" --output-format json
```

**Solution for GitLab CI**:
```yaml
variables:
  ANTHROPIC_API_KEY: $CI_ANTHROPIC_API_KEY

script:
  - claude -p "Task" --output-format json
```

### "Exit code not being checked"

**Problem**: Pipeline continues even when Claude fails.

**Solution**:
```bash
# ❌ WRONG - Doesn't check exit code
claude -p "Task" --output-format json > result.json
# Pipeline continues even if claude failed!

# ✅ CORRECT - Check exit code
if ! claude -p "Task" --output-format json > result.json; then
  echo "Claude command failed"
  exit 1
fi

# ✅ ALSO CORRECT - Set errexit
set -e
claude -p "Task" --output-format json > result.json
```

## Debugging Tips

### Enable Verbose Logging

```bash
# Verbose mode
claude -p "Task" --verbose --output-format json 2>&1 | tee debug.log

# Check environment
env | grep CLAUDE
env | grep ANTHROPIC

# Test with simple prompt
claude -p "Echo: test" --output-format json | jq .
```

### Trace Full Workflow

```bash
#!/bin/bash
set -x  # Enable command tracing

echo "=== Testing headless Claude ==="

# Test 1: Basic invocation
echo "Test 1: Basic invocation"
claude -p "Echo: hello" 2>&1 | head -5

# Test 2: JSON output
echo "Test 2: JSON output"
RESULT=$(claude -p "List skills" --output-format json 2>&1)
echo "$RESULT" | jq -e '.session_id'

# Test 3: Tool access
echo "Test 3: Tool access"
claude -p "List files in current directory" --allowedTools "Bash" 2>&1 | head -10

# Test 4: Configuration
echo "Test 4: Configuration loading"
claude -p "Can you see CLAUDE.md instructions?" 2>&1 | grep -i "CLAUDE.md"

set +x
```

### Check System Resources

```bash
# Check disk space
df -h

# Check memory
free -h  # Linux
vm_stat  # macOS

# Check Claude process
ps aux | grep claude

# Check file descriptors
lsof -p $(pgrep claude)
```

## Getting Help

### Collect Diagnostic Information

```bash
#!/bin/bash
# diagnostic_report.sh

echo "=== Claude Headless Diagnostic Report ===" > diagnostic.txt
echo >> diagnostic.txt

echo "1. Environment" >> diagnostic.txt
echo "   - Date: $(date)" >> diagnostic.txt
echo "   - PWD: $(pwd)" >> diagnostic.txt
echo "   - User: $(whoami)" >> diagnostic.txt
echo "   - PATH: $PATH" >> diagnostic.txt
echo >> diagnostic.txt

echo "2. Claude CLI" >> diagnostic.txt
echo "   - Which: $(which claude)" >> diagnostic.txt
echo "   - Version: $(claude --version 2>&1 | head -1)" >> diagnostic.txt
echo >> diagnostic.txt

echo "3. Configuration" >> diagnostic.txt
echo "   - .claude/ exists: $([ -d .claude ] && echo yes || echo no)" >> diagnostic.txt
echo "   - Settings: $(ls -la .claude/settings*.json 2>/dev/null || echo "not found")" >> diagnostic.txt
echo "   - Skills: $(ls -d .claude/skills/*/ 2>/dev/null | wc -l) found" >> diagnostic.txt
echo >> diagnostic.txt

echo "4. API Key" >> diagnostic.txt
echo "   - Set: $([ -n "$ANTHROPIC_API_KEY" ] && echo yes || echo no)" >> diagnostic.txt
echo >> diagnostic.txt

echo "5. Test Invocation" >> diagnostic.txt
echo "   - Running test..." >> diagnostic.txt
if claude -p "Echo: test" --output-format json 2>&1 | jq -e '.session_id' >> diagnostic.txt 2>&1; then
  echo "   - Status: SUCCESS" >> diagnostic.txt
else
  echo "   - Status: FAILED" >> diagnostic.txt
  echo "   - Output:" >> diagnostic.txt
  claude -p "Echo: test" --output-format json 2>&1 | head -20 >> diagnostic.txt
fi

echo >> diagnostic.txt
echo "=== End Diagnostic Report ===" >> diagnostic.txt

cat diagnostic.txt
```

### Report Issues

When reporting issues, include:
1. Diagnostic report (above)
2. Exact command that failed
3. Full error output
4. Expected vs actual behavior
5. Environment (OS, shell, Claude version)

## See Also

- [CLI Reference](cli_reference.md) - Complete flag documentation
- [Examples](examples.md) - Working code examples
- [Agent Workflows](agent_workflows.md) - AILANG messaging patterns
- [SKILL.md](../SKILL.md) - Main skill documentation
