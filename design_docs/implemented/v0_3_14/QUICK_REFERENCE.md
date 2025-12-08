# Agent Inbox - Quick Reference

**Last Updated**: October 2025 (v0.3.14)

## 🎯 Pragmatic Workflow (USE THIS)

### When Starting a New Session

**User says:** "Check inbox" or "Any agent messages?"

**Claude does:**
```bash
ailang agent inbox --unread-only claude-code
```

**Claude sees messages, processes them, then:**
```bash
# If all tasks completed successfully:
ailang agent ack --all

# If specific task completed:
ailang agent ack msg_20251025_155729_a5f3e77ee975

# If task failed (to retry in next session):
ailang agent unack msg_20251025_155729_a5f3e77ee975
```

## 📋 Command Cheat Sheet

### Check Inboxes

```bash
# Claude-code inbox (project-specific, most common)
ailang agent inbox --unread-only claude-code

# User inbox (global, all projects)
ailang agent inbox --unread-only user

# ⚠️ FLAGS BEFORE AGENT ID!
# ✅ Correct:   ailang agent inbox --unread-only claude-code
# ❌ Wrong:     ailang agent inbox claude-code --unread-only
```

### Manage Messages

```bash
# Acknowledge single message
ailang agent ack msg_20251025_155729_a5f3e77ee975

# Acknowledge all messages
ailang agent ack --all

# Un-acknowledge (move back to unread)
ailang agent unack msg_20251025_155729_a5f3e77ee975
```

### Debugging

```bash
# Check logs
tail -20 ~/.ailang/state/hooks.log

# Run hook manually
bash scripts/hooks/session_start.sh

# List pending messages
ls .ailang/state/messages/claude-code/*.pending.json

# List processed messages
ls .ailang/state/messages/claude-code/_processed/
```

## 📂 Two Inbox Locations

| Location | Path | Purpose |
|----------|------|---------|
| **User Inbox** | `~/.ailang/state/messages/inbox/user/` | Global, cross-project messages TO user |
| **Claude-Code Inbox** | `.ailang/state/messages/claude-code/` | Project-specific messages TO claude-code agent |

**Most common**: Use claude-code inbox for agent-to-agent messages in this project.

## 🔄 Message Lifecycle

```
1. Agent sends message
   ↓
2. Lands in _unread or .pending.json
   ↓
3. Claude checks inbox (user asks)
   ↓
4. Claude processes task
   ↓
5a. SUCCESS → `ailang agent ack`
   ↓
   Moves to _processed or _read
   
5b. FAILURE → `ailang agent unack`
   ↓
   Moves back to _unread
   ↓
   Next session sees it again
```

## ⚠️ Known Issues

1. **SessionStart hook doesn't inject context** - Hook runs but output doesn't appear in Claude's context
   - **Workaround**: User asks Claude to check inbox manually

2. **Flag ordering matters** - Flags must come BEFORE agent ID
   - **Why**: Go flag package limitation

3. **Two inboxes** - Can be confusing which one to check
   - **Rule of thumb**: Use `claude-code` for agent messages, `user` for human messages

## ✅ What Actually Works

- ✅ Manual inbox checking (`ailang agent inbox --unread-only claude-code`)
- ✅ Acknowledgment system (`ack` / `unack`)
- ✅ Message persistence (survives crashes, stays until acknowledged)
- ✅ Lock file prevents duplicate hook execution
- ✅ Logging shows hook activity

## ❌ What Doesn't Work

- ❌ Automatic context injection via SessionStart hook
- ❌ Proactive notification of new messages
- ❌ System reminders showing agent messages

## 🚀 Best Practices for Claude

1. **Check inbox when user asks** - Don't assume hook worked
2. **Acknowledge after completing** - Use `ack --all` if all tasks done
3. **Un-acknowledge if blocked** - Use `unack` if you can't complete the task
4. **Tell user about messages** - Summarize what agents reported
5. **Ask for next steps** - Don't just acknowledge silently

## 📚 Full Documentation

- Main guide: `CLAUDE.md` (SESSION START ROUTINE section)
- Implementation: `design_docs/implemented/v0_3_14/agent-inbox-acknowledgment.md`
- Hook details: `design_docs/implemented/v0_3_14/agent-inbox-sessionstart-hook.md`
