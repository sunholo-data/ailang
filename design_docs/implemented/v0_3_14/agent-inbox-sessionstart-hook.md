# SessionStart Hook for Agent Inbox

**Status**: ⚠️ PARTIALLY WORKING (October 2025)
**Version**: v0.3.14+

## Overview

The SessionStart hook is designed to automatically check for agent messages when a new Claude Code session starts and inject them into Claude's context.

**Current Reality**: The hook executes successfully but its output doesn't reliably appear in Claude Code's context.

## What Works

✅ **Hook Execution**:
- Hook runs on every session start (verified in logs)
- Checks both inbox locations (user + claude-code)
- Outputs formatted message summaries to stdout
- Lock file prevents duplicate execution
- Logged to `~/.ailang/state/hooks.log`

✅ **Message Detection**:
- Finds all `.pending.json` files in claude-code inbox
- Finds all unread messages in user inbox
- Counts messages correctly
- Formats JSON payloads for readability

## What Doesn't Work

❌ **Context Injection**:
- Hook output to stdout doesn't appear in Claude Code's context
- System reminders don't show the message summaries
- Claude doesn't see the messages automatically

❌ **Automatic Awareness**:
- Claude must be explicitly asked to check inbox
- No proactive notification of new messages

## Pragmatic Workaround

**Instead of relying on automatic injection:**

1. **User asks Claude to check inbox** at session start
2. **Claude runs**: `ailang agent inbox --unread-only claude-code`
3. **Claude sees messages** and can process them
4. **Claude acknowledges**: `ailang agent ack --all` when done

**This workflow is reliable and works well in practice.**

## Related Files

- Hook script: `scripts/hooks/session_start.sh`
- Hook config: `.claude/settings.local.json`
- Hook logs: `~/.ailang/state/hooks.log`
- Main doc: `CLAUDE.md` (SESSION START ROUTINE section)
- Design doc: `design_docs/implemented/v0_3_14/agent-inbox-acknowledgment.md`
