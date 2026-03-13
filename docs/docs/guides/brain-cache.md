# Brain Cache: Persistent Semantic Knowledge

The AILANG brain is a two-tier persistent semantic cache that accumulates coding knowledge across sessions. It stores resolutions, patterns, and learnings as searchable frames — automatically captured by Claude Code hooks and queryable via CLI.

## Quick Start

```bash
# Store a learning
ailang cache put type_unify_tip --content "Always check occurs in unification before substituting type variables" --ns learnings

# Search your brain
ailang cache search "type inference"

# View what's stored
ailang cache stats
ailang cache list

# Promote useful project knowledge to your global brain
ailang cache promote type_unify_tip
```

## Two-Tier Architecture

The brain operates at two levels, matching the `ailang messages` scoping pattern:

| Tier | Location | Scope |
|------|----------|-------|
| **User** | `~/.ailang/state/brain.db` | Cross-project knowledge (follows you everywhere) |
| **Project** | `.ailang/state/brain.db` | Repo-specific knowledge (stays with the project) |

By default, searches query both tiers. Project-local results get a small relevance boost (+0.05) since they're more likely to be contextually relevant.

### Controlling Scope

```bash
# Store in project brain (default)
ailang cache put fix_parser --content "Parser needs lookahead for pipe operator" --scope project

# Store in user brain (cross-project)
ailang cache put go_tip --content "Use sync.Pool for hot-path allocations" --scope user --ns patterns

# Search only user brain
ailang cache search "sync pool" --scope user

# Promote from project to user (found something universally useful)
ailang cache promote fix_parser
```

## Namespaces

Frames are organized by namespace for filtering and lifecycle management:

| Namespace | Purpose | Default TTL |
|-----------|---------|-------------|
| `resolutions` | Git commit summaries (auto-captured by hooks) | 90 days |
| `code-context` | File/function context from editing | 30 days |
| `learnings` | Manual insights and tips | No expiry |
| `patterns` | Reusable coding patterns | No expiry |
| `session` | Session-specific scratch data | 7 days |
| `ephemeral` | Temporary working memory | 24 hours |

## CLI Reference

### `ailang cache search <query>`

Search by SimHash similarity + keyword matching. Results are merged and deduplicated.

```bash
ailang cache search "type inference bug"
ailang cache search --context internal/types/unify.go   # Find knowledge about specific files
ailang cache search "parser" --namespace patterns --limit 5
```

### `ailang cache put <key> --content "text"`

Store a frame manually.

```bash
ailang cache put fix_unify --content "Always check occurs in unification" --ns learnings
ailang cache put go_tip --ns patterns --scope user --content "Use sync.Pool for allocations"
ailang cache put temp_note --ns ephemeral --ttl 24h --content "Investigating race in scheduler"
```

### `ailang cache put-resolution`

Store a commit resolution frame (typically called by hooks, not manually).

```bash
ailang cache put-resolution --commit-msg "Fix race condition in scheduler" --files "internal/sched/run.go"
```

### `ailang cache list`

List recent frames.

```bash
ailang cache list               # All recent frames
ailang cache list --scope user  # Only user-tier frames
```

### `ailang cache stats`

Show brain statistics per tier and namespace.

### `ailang cache gc`

Garbage collect expired frames.

```bash
ailang cache gc                         # Remove TTL-expired frames
ailang cache gc --older-than 90d        # Remove frames older than 90 days
ailang cache gc --namespace ephemeral   # Only clean ephemeral namespace
```

### `ailang cache export` / `import`

Backup and restore via JSONL.

```bash
ailang cache export > brain_backup.jsonl
ailang cache import < brain_backup.jsonl
```

### `ailang cache promote <key>`

Copy a frame from project brain to user brain for cross-project reuse.

## Claude Code Hooks

The brain integrates with Claude Code via hooks that run automatically:

### Session Start (Context Injection)

When a Claude Code session starts, the hook:
1. Checks recently modified files (last 3 commits)
2. Searches the brain for relevant knowledge
3. Injects top-3 frames into system reminders

This means Claude starts each session with relevant past learnings already in context.

### Post-Commit (Resolution Capture)

After every `git commit`, the hook:
1. Detects the commit via PostToolUse:Bash
2. Extracts commit message, diff stats, changed files
3. Stores a resolution frame in the `resolutions` namespace

Over time, this builds a searchable history of what was fixed and how.

### Disabling Hooks

Set `AILANG_BRAIN_HOOKS=0` in your environment to disable all brain hooks.

## How Search Works

The brain uses two complementary search strategies:

1. **SimHash similarity** — Locality-sensitive hashing computes a 64-bit fingerprint of text content. Similar texts have similar hashes (measured by Hamming distance). Score: `1.0 - (hamming_distance / 64.0)`.

2. **Keyword search** — SQL LIKE matching on content and key fields. Always returns score 1.0 for matches.

Results from both methods are merged, deduplicated by key, and sorted by score (descending).

## Storage

Brain databases are SQLite with WAL mode (same configuration as `ailang messages`):
- Write-ahead logging for concurrent access
- Busy timeout: 5 seconds
- Cache size: 64MB

Each frame stores: key, namespace, content, SimHash, version, timestamps, TTL, and source metadata.
