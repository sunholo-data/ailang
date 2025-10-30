# Web UI Execution Guide

This guide explains how Claude Code behaves differently when running in the web UI (cloud environment) vs local CLI, and how to optimize your development experience.

## Environment Detection

Claude Code automatically detects whether it's running in a cloud or local environment at session start.

### Detection Indicators

**Cloud/Web UI Environment:**
- `CLAUDE_CODE_REMOTE=true`
- `CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE=cloud_default`
- `IS_SANDBOX=yes`
- `/.dockerenv` file exists
- `HOME=/root`
- Hostname: `runsc` (gVisor runtime)

**Local Development Environment:**
- `CLAUDE_CODE_REMOTE=false` or unset
- `HOME=/Users/<username>` or `/home/<username>`
- No `/.dockerenv` file
- Regular machine hostname

### Automatic Detection at Session Start

The session start hook (`scripts/hooks/session_start.sh`) automatically detects the environment and displays guidance:

**Cloud Environment:**
```
📭 Agent inbox: No unread messages from autonomous agents.

🌐 Web UI environment detected. For ailang commands, use: export PATH=$PATH:/root/go/bin
```

**Local Environment:**
```
📭 Agent inbox: No unread messages from autonomous agents.

💻 Local development environment detected.
```

## Key Differences: Cloud vs Local

### PATH Configuration

**Cloud (Web UI):**
- Go binaries install to: `/root/go/bin/ailang`
- PATH not configured by default
- **Solution**: Prefix commands with `export PATH=$PATH:/root/go/bin &&`

**Local:**
- Go binaries install to: `~/go/bin/ailang`
- Usually in PATH already
- **Solution**: Add `export PATH=$PATH:~/go/bin` to shell profile if needed

### Bash Tool Behavior

**Cloud (Web UI):**
- Each Bash tool call starts a fresh shell
- Environment variables don't persist between tool calls
- PATH exports must be repeated or chained

**Example - Multiple Independent Commands:**
```bash
# Tool call 1:
export PATH=$PATH:/root/go/bin && ailang --version

# Tool call 2 (separate invocation):
export PATH=$PATH:/root/go/bin && ailang doctor builtins
# ⚠️ Both need PATH - they don't share environment!
```

**Example - Chained Commands:**
```bash
# Single tool call with chained commands:
export PATH=$PATH:/root/go/bin && \
  ailang --version && \
  ailang doctor builtins && \
  ailang agent inbox --unread-only claude-code
# ✅ PATH persists through the chain
```

**Local:**
- Bash tool calls may share environment (depends on implementation)
- PATH usually configured in shell profile

### File System

**Cloud (Web UI):**
- `/root` for home directory
- `/home/user/ailang` for project directory
- Docker container file system

**Local:**
- `~/.config/claude-code` for settings
- Project directory wherever you opened it
- Host machine file system

## Best Practices for Web UI

### 1. Always Use PATH Exports

```bash
# ❌ WRONG - ailang won't be found
ailang doctor builtins

# ✅ CORRECT
export PATH=$PATH:/root/go/bin && ailang doctor builtins
```

### 2. Chain Related Commands

```bash
# ✅ CORRECT - Efficient chaining
export PATH=$PATH:/root/go/bin && \
  ailang eval-suite --models gpt5-mini && \
  ailang eval-report eval_results/baselines/v0.3.24 v0.3.24
```

### 3. Use Built-in Tools When Available

```bash
# ✅ PREFERRED - Use Read tool
# [Claude uses Read tool to read file]

# ❌ AVOID - Unnecessary bash
cat somefile.txt

# ✅ EXCEPTION - Use bash for system operations
git status
make test
```

### 4. Test Environment Detection

The `scripts/detect_environment.sh` script provides utilities for environment detection:

```bash
# Show full environment info
scripts/detect_environment.sh info

# Get environment type (cloud or local)
scripts/detect_environment.sh detect

# Get appropriate Go bin path
scripts/detect_environment.sh path

# Setup PATH (when sourced)
source scripts/detect_environment.sh && setup_ailang_path
```

### 5. Quick Session Start Verification

Use the `session_start_check.sh` script to verify everything is ready:

```bash
scripts/session_start_check.sh
```

Output:
```
=== 🚀 Session Start Verification ===

1️⃣  Environment: cloud
   🌐 Running in Claude Web UI

2️⃣  AILANG Binary:
   ✅ AILANG dev

3️⃣  Builtin Registry:
   ✅ Healthy (59 builtins)

4️⃣  Agent Inbox:
   ✅ Clear (no unread messages)

5️⃣  Git Repository:
   Branch: claude/test-web-ui-integration-011CUcz6mCvwcidTYYwayTje
   ✅ Clean working tree

6️⃣  Skills & Agents:
   ✅ 14 skills available
   ✅ 7 agents available

=== ✅ Session Ready! ===
```

## Troubleshooting

### "command not found: ailang"

**Problem:** AILANG binary not in PATH

**Solution:**
```bash
# Option 1: Prefix with PATH export
export PATH=$PATH:/root/go/bin && ailang --version

# Option 2: Rebuild and install
make quick-install

# Option 3: Use absolute path
/root/go/bin/ailang --version
```

### "No such file or directory: /Users/mark/go/bin/ailang"

**Problem:** Using local paths in cloud environment

**Solution:** The cloud environment uses `/root/go/bin`, not `~/go/bin`. Use environment detection:
```bash
GO_BIN=$(scripts/detect_environment.sh path)
export PATH=$PATH:$GO_BIN && ailang --version
```

### Commands work individually but not when chained

**Problem:** Missing `&&` or improper quoting

**Solution:**
```bash
# ❌ WRONG - Second command won't have PATH
export PATH=$PATH:/root/go/bin
ailang doctor builtins

# ✅ CORRECT - Chain with &&
export PATH=$PATH:/root/go/bin && ailang doctor builtins

# ✅ ALSO CORRECT - Multi-line with &&
export PATH=$PATH:/root/go/bin && \
  ailang doctor builtins && \
  ailang agent inbox --unread-only claude-code
```

## Performance Considerations

### Cloud Environment

**Pros:**
- Consistent environment across sessions
- No local setup required
- Containerized isolation

**Cons:**
- Each Bash call starts fresh shell (overhead)
- PATH must be explicitly managed
- Network latency for git operations

**Optimization Tips:**
- Chain commands to reduce shell startup overhead
- Use built-in tools (Read, Write, Grep, Glob) instead of bash when possible
- Cache environment detection results in variables

### Local Environment

**Pros:**
- Faster bash execution
- Persistent environment
- Direct file system access

**Cons:**
- Requires local setup and dependencies
- Environment may vary between developers

## Testing Your Setup

Run this comprehensive test to verify everything works:

```bash
# Environment detection
scripts/detect_environment.sh info

# Session verification
scripts/session_start_check.sh

# AILANG commands
export PATH=$PATH:/root/go/bin && \
  ailang --version && \
  ailang doctor builtins && \
  ailang agent inbox --unread-only claude-code

# Development tools
make test | head -20
git status
```

Expected output:
- ✅ Environment: cloud (or local)
- ✅ AILANG binary found and working
- ✅ All builtins valid (59 builtins)
- ✅ Agent inbox accessible
- ✅ Tests running
- ✅ Git repository accessible

## See Also

- [Session Start Hook](../../scripts/hooks/session_start.sh) - Automatic environment detection
- [Environment Detection Script](../../scripts/detect_environment.sh) - Manual detection utilities
- [Session Start Check](../../scripts/session_start_check.sh) - Comprehensive verification
- [CLAUDE.md](../../CLAUDE.md) - Full development guide
