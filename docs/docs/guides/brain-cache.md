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

The **user** tier is populated only by hand — `--scope user` on a write, or
`ailang cache promote <key>` to lift a project frame. Nothing automates it, by
design: cross-project knowledge is a curation decision, not a side effect. An
empty user tier is the normal state, not a fault.

### Git worktrees share the main brain

`.ailang/state/{evaluations,sprints}/*.json` are git-tracked, so every linked
worktree materialises a `.ailang/` directory. The project-root walk used to stop
there and open an *empty* brain, shadowing the main checkout's — which silently
gave a cold brain to worktree-isolated agent sessions, the ones that most need a
warm one. `ailang cache` now detects a linked worktree (a `.git` **file** holding
`gitdir: …/.git/worktrees/<name>`) and resolves to the main worktree's brain, so
learning accumulates in one place instead of fragmenting per branch and
vanishing when the worktree is removed.

### Injection tuning

Two hooks inject brain content into Claude Code sessions on `UserPromptSubmit`.
Both fire on every turn, so their relevance floors are the main control on
per-session token cost:

| Hook | Env var | Default | Searches |
|------|---------|---------|----------|
| `scripts/hooks/brain_on_prompt.sh` | `AILANG_BRAIN_MIN_SCORE` | `0.70` | all namespaces, top 3 |
| μRAG (`ailang micro-rag user-prompt`) | `AILANG_MICRORAG_USERPROMPT_FLOOR` | `0.70` | `ailang-syntax` + `ailang-builtins`, top 1 |

Both floors were calibrated on 2026-07-31 against a labelled prompt panel; see
the comment on `userPromptRelevanceFloor` in `internal/microrag/userprompt.go`
for the measurements and the trade-off. They are corpus-dependent — re-run
`TestUserPrompt_FloorSeparatesCalibrationPanel` after any reindex. Set either
env var to re-tune without a rebuild; out-of-range values fall back to the
default rather than widening the gate.

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

Search by cosine similarity + SimHash + keyword matching (three-tier). Results are merged and deduplicated.

```bash
ailang cache search "type inference bug"
ailang cache search --cosine "semantic patterns"        # Force cosine (embedding) search
ailang cache search --simhash "quick lookup"             # Force SimHash only (fast)
ailang cache search --context internal/types/unify.go   # Find knowledge about specific files
ailang cache search "parser" --namespace patterns --limit 5
```

### `ailang cache put <key> --content "text"`

Store a frame manually. Use `--embed` to compute and store an embedding vector.

```bash
ailang cache put fix_unify --content "Always check occurs in unification" --ns learnings
ailang cache put --embed --content "Use sync.Pool for allocations" --ns patterns --scope user go_tip
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

### `ailang cache embed`

Backfill embeddings for frames that don't have them yet.

```bash
ailang cache embed                        # All frames, both tiers
ailang cache embed --namespace learnings  # Only learnings
ailang cache embed --scope project        # Only project tier
```

### `ailang cache put-vector`

Store a vector-only frame (embedding + payload, no text) from JSON on stdin.

```bash
echo '{"key":"v1","embedding":[0.1,0.2],"payload":{"type":"task"}}' | ailang cache put-vector
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

## Embedding Vectors

The brain supports real embedding vectors for cosine similarity search, enabling semantic matching that SimHash can't achieve.

### Storing with Embeddings

```bash
# Auto-embed: computes embedding using configured provider
ailang cache put --embed --content "Use sync.Pool for hot-path allocations" sync_pool_tip

# Backfill: add embeddings to existing frames
ailang cache embed                        # All frames in both tiers
ailang cache embed --namespace learnings  # Only learnings namespace
ailang cache embed --scope user           # Only user tier
```

### Cosine Search

```bash
# Three-tier search (default): cosine → SimHash → text
ailang cache search "memory allocation patterns"

# Force cosine-only search
ailang cache search --cosine "memory allocation patterns"

# Force SimHash-only (fast path, no embedder needed)
ailang cache search --simhash "memory allocation"
```

### Machine-to-Machine Vectors

Store embedding-only frames (no text content) for vector communication:

```bash
echo '{"key":"task_001","embedding":[0.1,0.2,0.3],"payload":{"type":"task"}}' | ailang cache put-vector
```

### Embedding Coverage

```bash
ailang cache stats
# Shows: With embeddings: 42 (85%)
#        ollama:embeddinggemma: 42
```

### Embedder Configuration

Configure via `~/.ailang/config.yaml` or environment variables:

```yaml
embeddings:
  provider: ollama  # ollama, openai, gemini, or none
  ollama:
    model: embeddinggemma
    endpoint: http://localhost:11434
```

Or via env: `AILANG_EMBED_PROVIDER=ollama`, `AILANG_OLLAMA_MODEL=embeddinggemma`.

## How Search Works

The brain uses a three-tier search strategy:

1. **Cosine similarity** (best quality, requires embeddings) — Computes cosine similarity between query embedding and stored frame embeddings. Results get a +0.1 boost over SimHash-only results.

2. **SimHash similarity** (fast, always available) — Locality-sensitive hashing computes a 64-bit fingerprint of text content. Similar texts have similar hashes (measured by Hamming distance). Score: `1.0 - (hamming_distance / 64.0)`.

3. **Keyword search** (fallback) — SQL LIKE matching on content and key fields. Always returns score 1.0 for matches.

Results from all tiers are merged, deduplicated by key, and sorted by score (descending). When an embedder is not available, the brain falls back to SimHash + text only.

## Storage

Brain databases are SQLite with WAL mode (same configuration as `ailang messages`):
- Write-ahead logging for concurrent access
- Busy timeout: 5 seconds
- Cache size: 64MB

Each frame stores: key, namespace, content, SimHash, version, timestamps, TTL, source metadata, and optional embedding (BLOB), embedding dimension, and embedding model.
