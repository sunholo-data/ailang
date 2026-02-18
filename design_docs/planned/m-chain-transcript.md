# M-CHAIN-TRANSCRIPT: Session Transcript Viewing via ailang chains

**Status**: Planned
**Target**: v0.8.1
**Priority**: P1 (Medium-High)
**Estimated**: 2-3 hours
**Dependencies**: None (data already exists in chat_messages table)

## Problem Statement

Agent eval sessions store turn-by-turn conversation data in `chat_messages` (imported by `claudehistory.Importer`), but this data is not exposed through `ailang chains`. When analyzing eval results (e.g., a 37-turn adt_option session that should have been 4 turns), there's no way to see what errors the agent hit and how it iterated.

**Current State:**
- `chat_messages` table has `content_text`, `turn_number`, `role`, `content_json` for every turn
- Data is imported during eval via `chatImporter.SyncSession()` (eval_benchmark.go:280-285)
- 102 messages exist for a single eval session — rich debugging data
- No CLI command to view this data

**Impact:**
- Cannot analyze agent prompt effectiveness (what syntax gaps cause extra iterations?)
- Cannot debug eval failures at the conversation level
- Must resort to raw SQLite queries to inspect agent behavior

## Goals

**Primary Goal:** Add `ailang chains chat <chain-id> [--stage N]` to view turn-by-turn conversation.

**Success Metrics:**
- View full conversation for any chain stage in seconds
- Filter by stage number for multi-stage chains (evals)
- Compact view showing turn summaries (role, tool names, truncated text)
- Detailed view showing full text content

## Solution Design

### CLI Interface

```bash
# View chat for entire chain (all stages)
ailang chains chat <chain-id>

# View chat for specific stage (e.g., stage 3 = adt_option)
ailang chains chat <chain-id> --stage 3

# Compact view (tool names + truncated text)
ailang chains chat <chain-id> --compact

# JSON output for scripting
ailang chains chat <chain-id> --json

# Show only errors/tool results
ailang chains chat <chain-id> --errors
```

### Output Format

**Default view:**
```
Session: 92d1a197-570... (Stage 3: adt_option)
37 turns, 36 tool calls

─── Turn 1 (user) ───
You are solving an AILANG benchmark...
[truncated to 200 chars]

─── Turn 2 (assistant) ───
Let me read the solution file first.
  [tool] Read: solution.ail
  [tool] Write: solution.ail (42 lines)

─── Turn 3 (assistant) ───
  [tool] Bash: ailang check solution.ail
  [result] ERROR: PAR_001 unexpected token...

─── Turn 4 (assistant) ───
I see the syntax error. Let me fix the ADT declaration...
  [tool] Edit: solution.ail
...
```

**Compact view (`--compact`):**
```
T1  user      [prompt: 1200 chars]
T2  assistant Read, Write (42 lines written)
T3  assistant Bash: ailang check → ERROR: PAR_001
T4  assistant Edit: solution.ail (fix ADT syntax)
T5  assistant Bash: ailang check → ERROR: PAR_001
...
T37 assistant Bash: ailang run → OK (stdout matches)
```

### Implementation

**New file:** `cmd/ailang/chains_chat.go` (~120 LOC)

1. Parse `--stage`, `--compact`, `--json`, `--errors` flags
2. Look up chain → get stage(s) → get session_id(s)
3. Query `chat_messages` by session_id, ordered by turn_number
4. Parse `content_json` for tool_use blocks (name, result)
5. Format and print

**Modified file:** `cmd/ailang/chains.go` — add `"chat"` case to subcommand dispatch

### Data Flow

```
chat_messages (observatory.db)
  ├── session_id → matches chain_stages.session_id
  ├── turn_number → ordering
  ├── role → user/assistant
  ├── content_text → readable text
  └── content_json → full JSON with tool_use blocks
```

## Files to Modify/Create

**New files:**
- `cmd/ailang/chains_chat.go` - Chat view command (~120 LOC)

**Modified files:**
- `cmd/ailang/chains.go` - Add "chat" to subcommand dispatch (~5 LOC)

## Success Criteria

- [ ] `ailang chains chat <id>` shows turn-by-turn conversation
- [ ] `--stage N` filters to specific eval benchmark
- [ ] `--compact` shows one-line summaries per turn
- [ ] Tool names and error messages extracted from content_json
- [ ] Works for both coordinator tasks and eval sessions

## Non-Goals

- Full chat UI (this is CLI text output only)
- Modifying or replaying sessions
- Storing additional data (data already exists)

---

**Document created**: 2026-02-18
**Last updated**: 2026-02-18
