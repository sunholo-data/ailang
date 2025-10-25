# SessionStart Hook Test Procedure

**Goal**: Verify that the SessionStart hook automatically injects inbox messages into Claude's context at the start of a new session.

## Test Setup (Already Done)

Two test messages have been planted:

1. **Home location**: `~/.ailang/state/messages/inbox/user/_unread/test_home_location.json`
2. **Project location**: `.ailang/state/messages/claude-code/test_project_location.json`

## Testing Steps

### Step 1: Verify Test Messages Exist

```bash
# Check home directory message
ls -la ~/.ailang/state/messages/inbox/user/_unread/test_*.json

# Check project directory message
ls -la .ailang/state/messages/claude-code/test_*.json
```

Expected: Both files should exist.

### Step 2: Exit Current Claude Code Session

- **Method 1**: Type `/exit` in Claude Code
- **Method 2**: Close the Claude Code window/tab
- **Method 3**: Press Ctrl+C in terminal if running CLI

### Step 3: Start a NEW Claude Code Session

```bash
# Start fresh session in the ailang project
cd /Users/mark/dev/sunholo/ailang
claude-code
```

### Step 4: Ask Claude About Messages

In the new session, immediately ask:

```
Do you see any inbox messages from the SessionStart hook?
```

or simply:

```
What messages did you receive at session start?
```

## Expected Results

Claude should respond with something like:

> Yes! I received 2 messages at session start:
>
> 1. **From test-harness-home** (home directory):
>    - "If you see this in a NEW session, the home location works!"
>
> 2. **From test-harness-project** (project directory):
>    - "If you see this in a NEW session, the project location works!"

## Verification Checklist

After the new session starts:

- [ ] Claude sees the message from home directory (`test-harness-home`)
- [ ] Claude sees the message from project directory (`test-harness-project`)
- [ ] Total message count is 2
- [ ] Messages appear in Claude's initial context (no manual checking needed)
- [ ] Home message moved to: `~/.ailang/state/messages/inbox/user/_read/`
- [ ] Project message moved to: `.ailang/state/messages/claude-code/_processed/`

## Manual Hook Test (Optional)

If you want to test the hook script directly without starting a new session:

```bash
# Run hook manually to see output
bash scripts/hooks/session_start.sh
```

This shows what WOULD be injected into Claude's context, but doesn't test the actual hook integration.

## Troubleshooting

### If messages don't appear in new session:

1. **Check hook configuration**:
   ```bash
   cat .claude/hooks.json
   # Should have SessionStart hook configured
   ```

2. **Check hook execution logs**:
   ```bash
   tail -f ~/.ailang/state/hooks.log
   # Start new session and watch for hook execution
   ```

3. **Verify Claude Code version**:
   - SessionStart hook stdout injection requires Claude Code to support this feature
   - Check: https://docs.claude.com/en/docs/claude-code/hooks

4. **Test hook manually first**:
   ```bash
   bash scripts/hooks/session_start.sh
   # Should output formatted messages to stdout
   ```

### If messages appear but aren't moved:

- Check file permissions on directories
- Check hook logs for "WARN: Could not mark message as read"
- Verify directories exist and are writable

## Cleanup

After successful test, remove test messages:

```bash
# If test failed and messages are still unread
rm ~/.ailang/state/messages/inbox/user/_unread/test_*.json
rm .ailang/state/messages/claude-code/test_*.json

# If test succeeded, messages should be in processed directories
rm ~/.ailang/state/messages/inbox/user/_read/test_*.json
rm .ailang/state/messages/claude-code/_processed/test_*.json
```

## Alternative: Test with Real Agent Message

Instead of manual test messages, trigger a real agent to send a message:

```bash
# Example: Send message from sprint-planner agent
./bin/send-message claude-code '{
  "from_agent": "sprint-planner",
  "payload": {
    "message": "Real test from sprint-planner",
    "status": "testing"
  }
}'

# Then exit and start new session
```

This tests the complete workflow: agent → inbox → hook → Claude's context.
