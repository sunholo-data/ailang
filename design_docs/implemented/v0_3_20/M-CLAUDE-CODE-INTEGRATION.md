# M-CLAUDE-CODE-INTEGRATION: Document Split

**Status**: Split into multiple documents
**Date**: October 25, 2025

---

## Overview

This design doc has been split into two documents based on implementation status:

### ✅ Phases 1-2: IMPLEMENTED (v0.3.20)

**Document**: [design_docs/implemented/v0_3_20/M-CLAUDE-CODE-INTEGRATION-HOOKS.md](../implemented/v0_3_20/M-CLAUDE-CODE-INTEGRATION-HOOKS.md)

**What was built**:
- Claude Code hooks integration (SessionStart, Stop)
- Two inbox locations (user + claude-code)
- Full CLI commands for agent messaging
- Agent-inbox skill
- Automatic inbox checking on session start

**Key Files**:
- `scripts/hooks/session_start.sh`
- `scripts/hooks/agent_handoff.sh`
- `.claude/settings.local.json`
- CLAUDE.md (session start workflow)

**Bugs Fixed** (October 25, 2025):
- Hooks now read JSON from stdin (not CLAUDE_HOOK_JSON env var)
- CLAUDE.md clarified for session start workflow

### ❌ Phase 3: NOT IMPLEMENTED (Planned for v0.3.21+)

**Document**: [design_docs/planned/M-CLAUDE-CODE-HEADLESS.md](M-CLAUDE-CODE-HEADLESS.md)

**What's planned**:
- Headless mode wrapper scripts
- Agent-style wrapper for agent files
- Auto-handoff script for autonomous workflows
- Cron job examples
- Documentation and examples

**Key Files** (to be created):
- `tools/run_headless_claude.sh`
- `tools/run_claude_agent.sh`
- `scripts/auto_handoff.sh`
- `examples/cron/*.sh`
- `docs/HEADLESS_AGENTS.md`

**Estimated Time**: 3-4 days

---

## Why Split?

**Phases 1-2 provide core functionality**:
- Interactive sessions can trigger agents
- Agents can notify users
- Automatic inbox checking works

**Phase 3 is separate use case**:
- Fully autonomous operation (no human interaction)
- Different testing requirements
- Can be implemented independently

**Benefits of splitting**:
- Clearer scope boundaries
- Ship working functionality early (v0.3.20)
- Defer headless mode to v0.3.21+

---

## Original Document

The complete original design doc (all 3 phases) is archived at:
- `design_docs/archive/M-CLAUDE-CODE-INTEGRATION.md`

This archive is kept for historical reference but should not be used for current work.

---

**Split Date**: October 25, 2025
**Implemented**: [M-CLAUDE-CODE-INTEGRATION-HOOKS.md](../implemented/v0_3_20/M-CLAUDE-CODE-INTEGRATION-HOOKS.md)
**Planned**: [M-CLAUDE-CODE-HEADLESS.md](M-CLAUDE-CODE-HEADLESS.md)
