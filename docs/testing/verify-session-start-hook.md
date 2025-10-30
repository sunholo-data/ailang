# How to Verify Session Start Hook Works

## The Challenge

The session start hook (`scripts/hooks/session_start.sh`) runs automatically when a Claude Code session starts. To verify it works, we need to START A NEW SESSION and check if the environment detection message appears in system reminders.

## Why We Can't Test From Inside a Running Session

**IMPORTANT:** The `claude` CLI process you see running IS the current session. You cannot use it to spawn nested sessions from inside itself - this will hang indefinitely.

```bash
# ❌ DON'T DO THIS from inside a running Claude session:
claude -p "test" --output-format json
# This hangs because it tries to create a nested session!
```

## Proper Testing Methods

### Method 1: Manual Testing (Recommended)

**Step 1: Exit this session**
- Close the current Claude Code session
- Open a NEW Claude Code session

**Step 2: Look for the environment detection message**

At the start of the new session, you should see in system reminders:

```
🌐 Web UI environment detected. For ailang commands, use: export PATH=$PATH:/root/go/bin
```

(Or if running locally:)
```
💻 Local development environment detected.
```

**Step 3: Verify it worked**
- If you see the message → ✅ Hook is working!
- If you don't see it → Check logs (see below)

### Method 2: Check Hook Logs

The hook logs all execution to `~/.ailang/state/hooks.log`:

```bash
# View recent hook executions
tail -20 ~/.ailang/state/hooks.log

# Look for environment detection
grep "Execution environment" ~/.ailang/state/hooks.log
```

**Expected log entries:**
```
[2025-10-30T07:40:39Z] === Session Start Hook Started ===
[2025-10-30T07:40:39Z] Created lock file to prevent duplicate execution
[2025-10-30T07:40:39Z] Session ID:
[2025-10-30T07:40:39Z] User ID:
[2025-10-30T07:40:39Z] No unread messages in any inbox location
[2025-10-30T07:40:39Z] Execution environment: cloud
[2025-10-30T07:40:39Z] Outputted 'no messages' context to Claude
```

### Method 3: Direct Hook Invocation (Testing Only)

You can test the hook script directly (simulates what Claude Code does):

```bash
# Simulate Claude Code calling the hook
echo '{"sessionId":"test-session","userId":"test-user"}' | scripts/hooks/session_start.sh
```

**Expected output:**
```
📭 Agent inbox: No unread messages from autonomous agents.

🌐 Web UI environment detected. For ailang commands, use: export PATH=$PATH:/root/go/bin
```

**This tests:**
- ✅ Hook script executes without errors
- ✅ Environment detection logic works
- ✅ Output formatting is correct

**This does NOT test:**
- ❌ Whether Claude Code actually CALLS the hook
- ❌ Whether the hook output appears in system reminders
- ❌ Real session integration

### Method 4: From a Separate Machine (Advanced)

If you have access to a local installation of Claude Code:

```bash
# On your local machine (NOT from inside a Claude session)
cd /path/to/ailang
claude -p "What environment am I running in?" --output-format text
```

The output should mention the environment detection from the session start hook.

## Verification Checklist

**✅ Evidence the hook is working:**
1. **Logs show execution** - `grep "Execution environment" ~/.ailang/state/hooks.log`
2. **Direct invocation works** - `echo '{}' | scripts/hooks/session_start.sh`
3. **New sessions show message** - Start a new session and look for environment detection

**❌ What doesn't prove it's working:**
1. Testing from inside the current session
2. Trying to spawn nested claude processes
3. Only checking that the script file exists

## Current Status (As of This Testing)

**Confirmed Working:**
- ✅ Hook script executes successfully
- ✅ Environment detection logic works (detects "cloud")
- ✅ Output formatting is correct
- ✅ Logs show successful execution
- ✅ Cross-platform compatibility (Linux stat command fixed)

**Needs Verification:**
- ⏳ Whether new sessions display the message in system reminders
- ⏳ User experience in production (requires starting a new session)

## Expected User Experience

### First Session After These Changes

When you start your NEXT Claude Code session (not this one), you should see:

**In system reminders at session start:**
```
📭 Agent inbox: No unread messages from autonomous agents.

🌐 Web UI environment detected. For ailang commands, use: export PATH=$PATH:/root/go/bin
```

**What this means:**
- You immediately know you're in a cloud environment
- You get actionable guidance on how to use ailang commands
- No more confusion about PATH issues

### Subsequent Sessions

Every new session will show the same environment detection, keeping you informed about the execution context.

## Troubleshooting

### "I don't see the environment detection message in new sessions"

**Possible causes:**
1. **Hook not configured** - Check `.claude/settings.local.json` for SessionStart hook
2. **Hook script not executable** - Run `chmod +x scripts/hooks/session_start.sh`
3. **Hook failed silently** - Check logs: `tail ~/.ailang/state/hooks.log`
4. **Hook output truncated** - System reminders may have length limits

**Steps to debug:**
```bash
# 1. Check hook configuration
cat .claude/settings.local.json | jq '.hooks.SessionStart'

# 2. Check script is executable
ls -la scripts/hooks/session_start.sh

# 3. Test direct invocation
echo '{}' | scripts/hooks/session_start.sh

# 4. Check recent logs
tail -20 ~/.ailang/state/hooks.log
```

### "Hook logs show errors"

**Common errors and fixes:**

**Error: "jq: command not found"**
```bash
# Install jq
sudo apt-get install jq  # Debian/Ubuntu
brew install jq          # macOS
```

**Error: "stat: illegal option"**
- Fixed in latest version (cross-platform compatibility added)
- Update your scripts: `git pull`

**Error: "CLAUDE_ENV_FILE: No such file or directory"**
- This is normal - the variable may not be set in all environments
- The hook handles this gracefully with `${CLAUDE_ENV_FILE:-}`

## Technical Details

### How SessionStart Hooks Work

1. **Claude Code starts a new session**
2. **Before showing anything to Claude**, it checks for SessionStart hooks in `.claude/settings.local.json`
3. **Executes hook script** with JSON payload via stdin
4. **Captures stdout** from the hook
5. **Injects hook output into system reminders** shown to Claude
6. **Session begins** with Claude seeing the hook output

### Hook Configuration

Location: `.claude/settings.local.json`

```json
{
  "hooks": {
    "SessionStart": {
      "command": "$CLAUDE_PROJECT_DIR/scripts/hooks/session_start.sh"
    }
  }
}
```

### Hook Input (stdin)

Claude Code sends JSON with session metadata:

```json
{
  "sessionId": "session_011CUcz6mCvwcidTYYwayTje",
  "userId": "user_abc123",
  "hookEventName": "SessionStart"
}
```

### Hook Output (stdout)

The script outputs plain text that appears in system reminders:

```
📭 Agent inbox: No unread messages from autonomous agents.

🌐 Web UI environment detected. For ailang commands, use: export PATH=$PATH:/root/go/bin
```

## References

- **Hook script:** `scripts/hooks/session_start.sh`
- **Detection library:** `scripts/detect_environment.sh`
- **Hook logs:** `~/.ailang/state/hooks.log`
- **Hook config:** `.claude/settings.local.json`
- **Web UI guide:** `docs/guides/web-ui-execution.md`
