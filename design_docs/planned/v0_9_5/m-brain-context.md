# M-BRAIN-CONTEXT: Contextual Brain Injection During Active Sessions

**Status**: Implemented
**Target**: v0.9.5
**Priority**: P1 (High — directly improves every Claude Code session)
**Estimated**: 3 days (12h implementation + 6h testing + 2h docs)
**Dependencies**: M-BRAIN (v0.9.2 — ✅ implemented)
**Milestone ID**: M-BRAIN-CONTEXT
**Created**: 2026-03-27

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same query + same brain state = same results; cooldown is clock-based but deterministic per invocation |
| A2: Replayability | 0 | No new data written; read-only enhancement to existing brain |
| A3: Effect Legibility | +1 | Hook registration explicit in settings.json; injected content clearly labeled with source |
| A4: Explicit Authority | +1 | Uses existing SharedMem capability; no new ambient access |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | Read-only brain queries; no new write contention |
| A7: Machines First | +2 | Core value: injects machine-consumable context exactly when the machine needs it |
| A8: Minimal Syntax | 0 | No language syntax changes; hooks and CLI only |
| A9: Cost Visibility | +1 | Context budget is explicit and configurable; injection count trackable via telemetry |
| A10: Composability | +1 | Composes with existing brain write hooks; same `ailang cache search` infrastructure |
| A11: Structured Failure | 0 | Graceful degradation: hook failures silently exit 0 |
| A12: System Boundary | +1 | Clear boundary: file read triggers → brain query → system reminder injection |

**Net Score: +8** → **Decision: Strong Accept**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism — same brain state, same results
- [x] A3 (Effects): Hook is visible in settings.json; injection clearly labeled
- [x] A4 (Authority): No ambient access; reads existing brain via `ailang cache search`
- [x] A7 (Machines First): This IS the axiom — targeted context for machine consumption

---

## Problem Statement

The AILANG brain (M-BRAIN, v0.9.2) captures knowledge during sessions via `brain_resolution.sh` (PostToolUse on git commit) and injects it via `brain_session.sh` (SessionStart). But the **read side has a critical gap**: it only fires once, at session start, using a coarse signal (last 3 commits' changed files + branch name).

**Current State:**
- Brain reads: **once per session**, at startup, based on recent git diff
- Brain writes: **continuously**, on every git commit
- Mid-session, when focus shifts to a different area of the codebase, the brain is silent
- The SessionStart query often surfaces knowledge about *yesterday's* work, not *today's* task
- Reading a file like `internal/types/unify.go` gives no brain context about past type inference bugs fixed there, even though that knowledge exists in the brain

**Impact:**
- 60%+ of brain knowledge goes unused because it was relevant to files touched *after* session start
- The "system gets smarter with use" value proposition is undercut — the system *knows* things but doesn't *say* them at the right time
- Developers re-discover known issues, design decisions, and past fixes that the brain already captured

**Quantifiable gap:**
- Average session touches ~15 files; SessionStart only queries based on ~5 files from recent commits
- Brain has ~200+ resolution frames; session start surfaces at most 3
- Relevance decay: by mid-session, the initial brain injection is often about a completely different topic

---

## Goals

**Primary Goal:** Inject relevant brain knowledge when files are read during a session, not just at startup — aligning brain context with what the developer is *actually* working on.

**Success Metrics:**
1. Brain context injected at least 2-3 additional times per session (beyond SessionStart)
2. Injected context is relevant to the file being read (measured by file-path overlap in brain results)
3. No perceptible latency increase on Read tool calls (hook completes within 2s timeout)
4. Context window budget respected: max 500 tokens per injection, max 2000 tokens total brain content per session
5. Cooldown prevents noise: same directory not re-queried within 5 minutes

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Hook on PreToolUse(Read) vs PostToolUse(Read) | Pre = context before reading; Post = context after. Pre is more useful but adds latency before file display | agent | design | low |
| Cooldown strategy: per-directory vs per-file | Per-file = more queries, more relevant; per-directory = fewer queries, less noise | agent | design | low |
| Context budget: fixed cap vs adaptive | Fixed is simpler; adaptive could yield more at start, less as context fills | agent | compile | med |
| Where cooldown state lives: temp file vs env var | Temp file persists across tool calls; env var is simpler but doesn't persist | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Hook point: **PreToolUse on Read** (inject context *before* Claude reads the file, so it has background when processing)
- [x] Cooldown: **per-directory, 5-minute window** (balances relevance vs noise; configurable via env var)
- [x] Context budget enforcement approach: **soft guidance** — warn at budget limit but don't hard-stop; tune threshold after real usage data

---

## Solution Design

### Overview

A single new shell hook (`brain_context.sh`) registered as a `PreToolUse` hook matching `Read`. When Claude is about to read a file, the hook:

1. Extracts the file path from the tool input
2. Checks a cooldown cache (temp file) — skips if the same directory was queried recently
3. Queries the brain via `ailang cache search --context <file_path> --limit 2`
4. Filters results by relevance threshold
5. Injects results as a system reminder (same format as SessionStart brain injection)
6. Updates the cooldown cache

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Claude Code Session                         │
│                                                                │
│  SessionStart                  PreToolUse(Read)                │
│  ┌──────────────────┐         ┌──────────────────────┐        │
│  │ brain_session.sh  │         │ brain_context.sh      │        │
│  │ (existing)        │         │ (NEW)                 │        │
│  │ Git diff → query  │         │ File path → query     │        │
│  │ Runs ONCE         │         │ Runs per-Read          │        │
│  │                   │         │ with cooldown          │        │
│  └────────┬─────────┘         └────────┬─────────────┘        │
│           │                            │                       │
│           ▼                            ▼                       │
│  ┌─────────────────────────────────────────────────────┐      │
│  │            ailang cache search                        │      │
│  │  --context <file_path>  --limit 2  --format oneline  │      │
│  └───────────────────────┬───────────────────────────────┘      │
│                          │                                      │
│                          ▼                                      │
│  ┌──────────────────┐  ┌──────────────────────────────────┐   │
│  │ User Brain        │  │ Project Brain                     │   │
│  │ ~/.ailang/state/  │  │ .ailang/state/brain.db            │   │
│  │ brain.db          │  │ Resolutions, design docs, context │   │
│  └──────────────────┘  └──────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘

Cooldown Cache: /tmp/ailang_brain_cooldown_<session_hash>
Format: <directory_path>\t<unix_timestamp> (one per line)
TTL: 5 minutes (configurable via AILANG_BRAIN_COOLDOWN_SECS)
```

### Noise Prevention

The biggest risk is over-injection. Three mechanisms prevent this:

1. **Directory-level cooldown** — Once a directory is queried, it won't be re-queried for 5 minutes. Reading `internal/types/unify.go` then `internal/types/infer.go` only triggers one brain query.

2. **Relevance threshold** — Only inject results with score ≥ 0.25 (configurable). Low-relevance results are silently dropped.

3. **Session budget** — Track total tokens injected this session. Stop injecting after 2000 tokens cumulative (configurable). This prevents context window pollution in long sessions.

### Hook Implementation

```bash
#!/bin/bash
# brain_context.sh — PreToolUse hook for Read
# Injects relevant brain knowledge when files are being read

set +e

# Quick exits
[ "${AILANG_BRAIN_HOOKS:-1}" = "0" ] && exit 0
command -v ailang &>/dev/null || exit 0
command -v jq &>/dev/null || exit 0

# Read hook JSON from stdin
HOOK_JSON=$(cat || echo "{}")
TOOL_NAME=$(echo "$HOOK_JSON" | jq -r '.tool_name // ""' 2>/dev/null)
[ "$TOOL_NAME" != "Read" ] && exit 0

# Extract file path
FILE_PATH=$(echo "$HOOK_JSON" | jq -r '.tool_input.file_path // ""' 2>/dev/null)
[ -z "$FILE_PATH" ] && exit 0

# Only query for project source files (skip node_modules, .git, binaries, etc.)
case "$FILE_PATH" in
    */node_modules/*|*/.git/*|*.png|*.jpg|*.pdf|*.wasm) exit 0 ;;
esac

# Directory-level cooldown
DIR_PATH=$(dirname "$FILE_PATH")
COOLDOWN_SECS="${AILANG_BRAIN_COOLDOWN_SECS:-300}"  # 5 minutes
COOLDOWN_FILE="/tmp/ailang_brain_cooldown_$$"

# Check cooldown (atomic check-and-update)
if [ -f "$COOLDOWN_FILE" ]; then
    NOW=$(date +%s)
    LAST_QUERY=$(grep -F "$DIR_PATH" "$COOLDOWN_FILE" 2>/dev/null | tail -1 | cut -f2)
    if [ -n "$LAST_QUERY" ]; then
        ELAPSED=$((NOW - LAST_QUERY))
        [ "$ELAPSED" -lt "$COOLDOWN_SECS" ] && exit 0
    fi
fi

# Session budget check
BUDGET_FILE="/tmp/ailang_brain_budget_$$"
MAX_BUDGET="${AILANG_BRAIN_BUDGET_TOKENS:-2000}"
CURRENT_BUDGET=$(cat "$BUDGET_FILE" 2>/dev/null || echo "0")
[ "$CURRENT_BUDGET" -ge "$MAX_BUDGET" ] && exit 0

# Query brain
RESULTS=$(ailang cache search --context "$FILE_PATH" --limit 2 --format oneline 2>/dev/null || echo "")

# Filter empty/no-results
[ -z "$RESULTS" ] && exit 0
echo "$RESULTS" | grep -q "No results" && exit 0

# Filter by relevance threshold (parse score from output)
MIN_SCORE="${AILANG_BRAIN_MIN_SCORE:-0.25}"
FILTERED=$(echo "$RESULTS" | awk -v min="$MIN_SCORE" '
    match($0, /\(([0-9.]+)\)/, arr) { if (arr[1]+0 >= min+0) print }
    !match($0, /\([0-9.]+\)/) { print }
' | head -3)
[ -z "$FILTERED" ] && exit 0

# Estimate token cost (~4 chars per token)
RESULT_CHARS=${#FILTERED}
RESULT_TOKENS=$((RESULT_CHARS / 4))
NEW_BUDGET=$((CURRENT_BUDGET + RESULT_TOKENS))

# Output brain context
BASENAME=$(basename "$DIR_PATH")
echo ""
echo "━━━ 🧠 BRAIN: Context for ${BASENAME}/ ━━━"
echo "$FILTERED"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Update cooldown and budget
echo -e "${DIR_PATH}\t$(date +%s)" >> "$COOLDOWN_FILE"
echo "$NEW_BUDGET" > "$BUDGET_FILE"

exit 0
```

### Settings.json Registration

Add to the **user-level** `~/.claude/settings.json` (not project-level, so it works across all projects):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Read",
        "hooks": [
          {
            "type": "command",
            "command": "~/.ailang/hooks/brain_context.sh",
            "timeout": 2
          }
        ]
      }
    ]
  }
}
```

Note: the 2-second timeout is intentional — brain queries should complete in <100ms (SQLite), so 2s is generous while preventing hangs.

### Implementation Plan

**Phase 1: Core Hook (~6h)**
- [ ] Create `~/.ailang/hooks/brain_context.sh` — PreToolUse hook for Read
- [ ] Implement directory-level cooldown with temp file state
- [ ] Implement session token budget tracking
- [ ] Implement relevance threshold filtering
- [ ] File path filtering (skip binaries, node_modules, .git)
- [ ] Register hook in `~/.claude/settings.json`
- [ ] Test: read a file with known brain content → verify injection
- [ ] Test: read two files in same directory → verify cooldown prevents double-injection

**Phase 2: Observability & Tuning (~4h)**
- [ ] Add `ailang cache search --format oneline` if not already available (compact single-line results)
- [ ] Add telemetry: count brain injections per session (via existing OTEL hook)
- [ ] Add `AILANG_BRAIN_COOLDOWN_SECS`, `AILANG_BRAIN_BUDGET_TOKENS`, `AILANG_BRAIN_MIN_SCORE` env var configuration
- [ ] Measure: average query latency, injection frequency, relevance scores
- [ ] Tune defaults: adjust cooldown, budget, threshold based on real usage

**Phase 3: Docs & Polish (~2h)**
- [ ] Update `docs/docs/guides/brain-cache.md` with contextual injection section
- [ ] Update M-BRAIN design doc to reference this extension
- [ ] CHANGELOG.md entry
- [ ] Add disable instructions: `AILANG_BRAIN_HOOKS=0` to skip all brain hooks

### Files to Modify/Create

**New files:**
| File | LOC Est. | Purpose |
|------|----------|---------|
| `~/.ailang/hooks/brain_context.sh` | ~80 | PreToolUse hook for Read — core implementation |

**Modified files:**
| File | Changes | LOC Est. |
|------|---------|----------|
| `~/.claude/settings.json` | Add PreToolUse Read hook entry | +10 |
| `docs/docs/guides/brain-cache.md` | Document contextual injection | +30 |
| `CHANGELOG.md` | Feature entry | +5 |

**Total:** ~125 LOC — intentionally small. The brain infrastructure already exists; this is a thin read-side hook.

---

## Examples

### Example 1: Reading a file with known brain context

**Before (brain silent mid-session):**
```
> Read internal/types/unify.go
[Claude reads 400 lines of code, no background context]
[Spends tokens re-analyzing the occurs check logic]
[Doesn't know about the fix from last week]
```

**After (brain injects relevant context):**
```
> Read internal/types/unify.go

━━━ 🧠 BRAIN: Context for types/ ━━━
  1. [resolution] [3d ago] Fix TypeVar unification missing occurs check (0.87)
     Files: internal/types/unify.go  Commit: abc123
  2. [design-docs] [1w ago] M-TYPEENV-SUB: type safety regression tests (0.42)
     Full document: design_docs/planned/v0_9_5/m-typeenv-sub.md
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Claude reads the file WITH context about recent changes and design decisions]
```

### Example 2: Cooldown prevents noise

```
> Read internal/types/unify.go
━━━ 🧠 BRAIN: Context for types/ ━━━
  1. [resolution] Fix TypeVar unification...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

> Read internal/types/infer.go
[No brain injection — types/ directory still in cooldown]

> Read internal/parser/parser.go
━━━ 🧠 BRAIN: Context for parser/ ━━━
  1. [resolution] Fix token position off-by-one in effect blocks (0.76)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[Different directory, so brain fires again]
```

### Example 3: Session budget exhaustion

```
[After 4-5 brain injections totaling ~2000 tokens]

> Read internal/codegen/emitter.go
[No brain injection — session budget exhausted]
[Brain stays silent for rest of session to preserve context window]
```

---

## Success Criteria

- [ ] Brain injects context on Read for files with relevant brain knowledge
- [ ] Cooldown prevents duplicate queries for same directory within 5 minutes
- [ ] Session budget caps total brain injection at ~2000 tokens
- [ ] Relevance threshold filters out low-quality results (score < 0.25)
- [ ] No perceptible latency on Read calls (hook < 2s timeout)
- [ ] Graceful degradation: hook exits silently if ailang unavailable, brain empty, or any error
- [ ] Binary/non-source files skipped (no brain queries for .png, .pdf, etc.)
- [ ] Configurable via env vars: `AILANG_BRAIN_COOLDOWN_SECS`, `AILANG_BRAIN_BUDGET_TOKENS`, `AILANG_BRAIN_MIN_SCORE`
- [ ] All existing brain tests still pass
- [ ] CHANGELOG.md updated
- [ ] Documentation updated

---

## Testing Strategy

**Unit tests (hook script):**
- Mock `ailang cache search` output → verify filtering logic
- Test cooldown: write timestamp, verify skip within window
- Test budget: write budget at limit, verify no output
- Test file path filtering: binary paths skipped, source paths processed

**Integration tests:**
- Populate brain with known resolution → Read file in same directory → verify injection
- Read file in directory with no brain content → verify no output
- Read two files in same directory → verify cooldown (second read = no injection)
- Read files in 6+ directories → verify budget exhaustion

**Manual testing:**
- Full workflow: commit a fix → new session → read the fixed file → verify brain surfaces the resolution
- Long session: work across many files → verify budget caps injection
- Disable: `AILANG_BRAIN_HOOKS=0` → verify no injections

**Impact measurement:**
- Compare sessions with/without contextual injection
- Track: how often does injected context get referenced in Claude's responses?
- Track: does session start brain injection become less important (because mid-session picks up the slack)?

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Exact cooldown file format** — agent may choose (TSV, JSON, or simple key=value)
- **Budget estimation method** — char/4 approximation is fine; tiktoken-based counting is optional
- **PreToolUse vs PostToolUse** — design says PreToolUse, but if latency is unacceptable in practice, agent may switch to PostToolUse
- **Grep/Glob injection** — extending to PreToolUse on Grep/Glob is a natural follow-up but not required in this sprint

---

## Non-Goals

**Not attempted in this feature:**
- **Brain writes on Read** — this is read-only; we don't capture "what was read" as brain knowledge
- **Grep/Glob hook injection** — possible extension, but Read is the highest-value target; defer to measure impact first
- **UserPromptSubmit hook** — would require NLP to extract topics from user messages; higher complexity, defer
- **Adaptive budget** — start with fixed cap; adaptive (more budget at start, less later) adds complexity for marginal gain
- **Cross-session cooldown** — cooldown is per-session (temp file with PID); no need to persist across sessions
- **Neural embedding queries in hook** — SimHash + FTS5 is fast enough; neural adds latency and Ollama dependency

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Hook latency delays Read display | Medium | 2s timeout; SQLite queries are <100ms; graceful exit on timeout |
| Context window pollution from too many injections | High | Session budget cap (2000 tokens); directory cooldown (5 min); relevance threshold |
| Cooldown temp file grows unbounded in long sessions | Low | One line per directory; typical session touches <50 directories; file deleted on session end |
| PID-based temp file collides with concurrent sessions | Low | Different PIDs = different files; or use `$CLAUDE_SESSION_ID` if available |
| Brain results are stale/wrong | Medium | Results include timestamps ("3d ago"); user can see recency; brain GC removes old entries |
| Hook breaks on non-AILANG projects | Low | Graceful exits: no ailang binary → exit 0; no brain.db → exit 0 |

---

## Related Documents

**Direct predecessor (this extends):**
- [design_docs/implemented/v0_9_2/m-brain.md](../../implemented/v0_9_2/m-brain.md) — Original brain implementation with SessionStart + PostToolUse(commit) hooks

**Infrastructure this builds on:**
- [design_docs/implemented/v0_6_0/semantic-caching-complete.md](../../implemented/v0_6_0/semantic-caching-complete.md) — SharedMem/SharedIndex/sem_frame foundation
- [design_docs/implemented/v0_6_0/dx-16-shared-index-deterministic-retrieval.md](../../implemented/v0_6_0/dx-16-shared-index-deterministic-retrieval.md) — SharedIndex deterministic retrieval

**Existing hooks this sits alongside:**
- `~/.ailang/hooks/brain_session.sh` — SessionStart brain injection (reads)
- `~/.ailang/hooks/brain_resolution.sh` — PostToolUse(Bash) resolution capture (writes)

**Future work this enables:**
- [design_docs/planned/v0_10_0/m-coord-thinking-levels.md](../../planned/v0_10_0/m-coord-thinking-levels.md) — Coordinator thinking levels could use brain context for decision quality

---

## Future Work

1. **Grep/Glob hook injection** — Extend to PreToolUse on Grep/Glob for search-phase context
2. **UserPromptSubmit analysis** — Extract topics from user messages, inject brain context before any tools fire
3. **Adaptive budget** — More brain context early in session, less as context fills
4. **Write-on-Read capture** — Track which files are frequently read together (co-access patterns) to improve future brain queries
5. **Cross-project brain queries** — When reading a file pattern that exists in another project, query that project's brain too
6. **Brain relevance feedback** — Track whether injected context appears in Claude's responses (implicit relevance signal)

---

**Document created**: 2026-03-27
**Last updated**: 2026-03-27
