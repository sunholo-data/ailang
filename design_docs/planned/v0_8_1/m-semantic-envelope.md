# M-SEMANTIC-ENVELOPE: Multi-Aspect Semantic Embeddings for Agent Messaging

**Status**: Planned
**Target**: v0.8.1
**Priority**: P1 (Medium-High)
**Estimated**: 5 days (20h implementation + 8h testing + 4h docs + buffer)
**Dependencies**: M-MSG-SEMANTIC (v0.5.11), M-DOC-SEM (v0.5.11)
**Milestone ID**: M-SEMANTIC-ENVELOPE
**Created**: 2026-02-19

---

## Problem Statement

The AILANG messaging system (v0.5.11+) stores messages with optional embeddings, but these serve only **search/retrieval** — SimHash shortlisting + neural reranking computed from message text. From an AI agent's perspective as both **sender and receiver** of messages, there is a much richer semantic layer that could be communicated — information that is difficult or impossible to express in plain text, but that another AI could meaningfully consume if embedded alongside the message.

**Current State:**
- Messages have a single `embedding` column (float32 vector from text via Ollama)
- Embeddings are only used for `ailang messages search --neural` and deduplication
- Each session start shows 10-20+ messages that must be manually triaged by reading each title
- No semantic clustering, no code-context embeddings, no resolution tracking
- When a task completes, the problem→solution knowledge is lost (not stored as embeddings)

**Impact:**
- AI agents spend significant context reading every message to triage (no clustering by code region)
- Agent routing is keyword-based (`category` field: bug/feature/etc.) rather than capability-matched
- No way to search "bugs affecting similar code" or "how was a similar bug fixed before?"
- Cross-session context is lost — sender's work context (code examined, errors seen) cannot be recovered
- Every completed task is a missed opportunity to build a searchable knowledge base

**The core insight: embeddings aren't just for search — they're a communication channel between AIs that operates at a different level than text.**

---

## Goals

**Primary Goal:** Transform message embeddings from a search index into a multi-aspect semantic communication layer, where each message carries named embedding vectors computed from different sources (intent, code, context, skill, resolution).

**Success Metrics:**
1. Messages support 5 named embedding slots (envelope), each searchable independently
2. Coordinator tasks that complete successfully auto-populate the `resolution` slot on the original message
3. `ailang messages search --space code "internal/types"` returns different results than `--space intent "fix type error"` (proving multi-space differentiation)
4. `ailang messages triage` clusters 10+ messages into ≤5 groups by code similarity in <2s
5. Provider-agnostic embedder supports Ollama, OpenAI, and Gemini (extends `Embedder` interface)
6. Backward compatible: messages without envelopes work identically to current behavior

---

## Non-Goals

- **ANN/vector database integration** — SQLite JSON column + scan is sufficient for message volumes (<100K). Deferred to cloud infra (M-CLOUD-INFRA)
- **Per-slot model configuration** — All slots use the same embedding model initially. Per-slot models (code-specific embeddings) deferred to future work
- **Automatic envelope computation on every send** — Only `intent` is auto-computed. Other slots require explicit `With*` options or CLI flags
- **Training custom embedding models** — Use off-the-shelf models only
- **Cross-inbox envelope search** — Search scoped to single inbox (consistent with M-MSG-SEMANTIC)

---

## Solution Design

### Overview

Instead of one embedding per message (text-derived, search-only), messages carry a **semantic envelope** — a set of named embedding vectors, each capturing a different *aspect* of the message's meaning. The existing `Embedder` interface (already provider-agnostic in shape) gains OpenAI and Gemini implementations via a factory function. An `EnvelopeBuilder` computes envelope vectors from contextual inputs. Multi-space search extends `SearchOptions` with an `EnvelopeSpace` field. A resolution feedback loop fills the `resolution` slot when coordinator tasks complete.

### Architecture

**Components:**

1. **Envelope** (`internal/messaging/envelope.go`) — Data type holding 5 named float32 vectors + model metadata
2. **EnvelopeBuilder** (`internal/messaging/envelope_builder.go`) — Functional-options builder that computes each slot from different sources
3. **Provider Embedders** (`internal/messaging/embedder_openai.go`, `embedder_gemini.go`) — New `Embedder` implementations using existing `internal/ai/` clients
4. **Embedder Factory** (`internal/messaging/embedder.go`) — `NewEmbedderFromConfig()` creates the right embedder from config
5. **Multi-Space Search** (`internal/messaging/search.go`) — Extended `SearchOptions` + `SearchByEnvelope()`
6. **Resolution Hook** (`internal/coordinator/daemon_tasks_exec.go`) — Post-completion envelope enrichment
7. **Triage CLI** (`cmd/ailang/messages_triage.go`) — Cluster messages by envelope slot similarity

### The 5 Envelope Slots

| Slot | Source | When Computed | Use Case |
|------|--------|---------------|----------|
| `intent` | Title + first 200 chars of payload | **Auto on send** (if embedder available) | "What is being asked?" — triage, dedup, priority |
| `code` | File paths + code snippets (`WithCodeContext`) | **Explicit** — sender provides | "What code is affected?" — cluster bugs by subsystem |
| `context` | Recent files, errors, tools (`WithSessionContext`) | **Explicit** — sender provides | "What was the sender working on?" — context recovery |
| `skill` | Compiler phases, AST nodes, file patterns (`WithSkillHints`) | **Explicit** — sender provides | "What expertise is needed?" — smart routing |
| `resolution` | Git diff + commit message (`WithResolution`) | **Auto on task completion** | "How was this resolved?" — problem→solution KB |

### Why Multiple Vectors?

A single text embedding conflates everything. Consider these messages:
- "Fix the parser crash on nested records" — intent: bug fix; code: parser; skill: parser expertise
- "Add parser support for nested records" — intent: feature; code: parser; skill: parser expertise

Same `code` and `skill` vectors, but opposite `intent` vectors. Multi-space search lets you query along the dimension you care about:

```
"Find bugs in same code area"     → search code space
"Find tasks needing parser work"  → search skill space
"How was a similar crash fixed?"  → search intent + retrieve resolution
```

### Data Flow

```
Sender                                          Receiver
──────                                          ────────
Message created
  ├── SimHash auto-computed (existing)
  ├── intent auto-computed (NEW, async)
  ├── code/context/skill if With* provided      ailang messages search --space code "parser"
  └── Stored in envelope column                   → clusters by code similarity

Task completes (coordinator)                    ailang messages search --space resolution
  ├── Git diff computed                           → finds past fixes for similar problems
  ├── resolution slot filled (NEW)
  └── Original message updated                  ailang messages triage --cluster-by code
                                                  → groups 18 messages into 5 clusters
```

### Concrete Use Cases

**Use Case 1: Triage 18 messages in 3 seconds**

Current: Read all 18 titles, manually group, decide priority.
With envelopes: `ailang messages triage --cluster-by code --inbox user` → "8 about type system, 5 about CLI, 3 about eval, 2 about docs".

**Use Case 2: "Have I seen this before?"**

Current: `ailang messages search "parser crash"` — text match only.
With envelopes: Search `code` space for the affected file → find past messages with similar code vectors → retrieve their `resolution` embeddings → guide approach.

**Use Case 3: Smart routing**

Current: Messages routed by inbox name and `category` string.
With envelopes: Coordinator computes similarity between message's `skill` vector and each agent's capability embedding → route to best-matched agent.

**Use Case 4: Problem→Solution knowledge base**

Current: Once acknowledged, messages are just history.
With envelopes: After resolution, the `resolution` slot is filled. Over time: (intent, code, resolution) triples accumulate. New problems auto-match past solutions. **The system gets smarter with use.**

---

## Implementation Plan

### Phase 1: Envelope Type + Schema (~4h)

- [ ] Create `internal/messaging/envelope.go` — `Envelope` struct, JSON serialization, slot accessors
- [ ] Add v1.8.0 migration in `internal/messaging/schema.go` — `ALTER TABLE inbox_messages ADD COLUMN envelope TEXT DEFAULT '{}'`
- [ ] Add `Envelope *Envelope` field to `InboxMessage` in `internal/messaging/inbox.go`
- [ ] Update `scanInboxMessage()` and `scanInboxMessageWithEmbedding()` to read envelope column
- [ ] Update `InsertInboxMessageWithContext()` to write envelope column
- [ ] Unit tests: `envelope_test.go` — marshal/unmarshal, slot access, empty envelope handling

### Phase 2: Provider-Agnostic Embedder (~4h)

- [ ] Add `NewEmbedderFromConfig(cfg EmbedConfig) (Embedder, error)` factory in `internal/messaging/embedder.go`
- [ ] Extend `EmbedConfig` with `OpenAI OpenAIConfig` and `Gemini GeminiConfig` fields
- [ ] Create `internal/messaging/embedder_openai.go` — uses `internal/ai/openai/` client for `text-embedding-3-small`
- [ ] Create `internal/messaging/embedder_gemini.go` — uses `internal/ai/gemini/` client for `text-embedding-004`
- [ ] Unit tests with mock HTTP for each new embedder
- [ ] Update `LoadEmbedConfigFromEnv()` to support `AILANG_EMBED_PROVIDER=openai|gemini`

### Phase 3: Envelope Builder (~6h)

- [ ] Create `internal/messaging/envelope_builder.go` — `EnvelopeBuilder` struct with functional options
- [ ] Implement `WithCodeContext(filePaths, codeSnippets)` — reads files, chunks, embeds
- [ ] Implement `WithSessionContext(recentFiles, recentErrors, recentTools)` — structured summary embedding
- [ ] Implement `WithSkillHints(phases, nodeTypes)` — capability description embedding
- [ ] Implement `WithResolution(diff, commitMsg)` — diff + commit summary embedding
- [ ] `Build()` always computes `intent` from title+payload, other slots only if With* provided
- [ ] Unit tests with mock embedder — verify each slot computes from correct source

### Phase 4: Multi-Space Search (~4h)

- [ ] Add `EnvelopeSpace string` to `SearchOptions` in `internal/messaging/search.go`
- [ ] Add `SearchByEnvelope(opts SearchOptions) ([]SearchHit, error)` method to `Store`
- [ ] When `EnvelopeSpace` is set: embed query → compare against that envelope slot in all messages
- [ ] When empty: fall back to existing text-embedding search (backward compatible)
- [ ] Add `FindSimilarResolutions(problemMsg, limit)` — search intent space, retrieve resolution vectors
- [ ] Unit + integration tests: same query returns different results in different spaces

### Phase 5: Resolution Feedback Loop (~3h)

- [ ] In `internal/coordinator/daemon_tasks_exec.go`, after successful task completion:
  - Get git diff for the worktree (`git diff HEAD~1..HEAD`)
  - Get commit messages
  - Build resolution embedding via `EnvelopeBuilder.Build(msg, WithResolution(diff, commitMsg))`
  - Update original message's envelope with `resolution` slot
- [ ] Add `UpdateMessageEnvelope(msgID string, env *Envelope) error` to Store
- [ ] Handle partial envelopes — merge new slot into existing envelope, don't overwrite
- [ ] Test: send message → complete task → verify resolution slot populated

### Phase 6: CLI Integration (~4h)

- [ ] Add `--envelope-code FILE:LINES` and `--envelope-context "description"` flags to `cmd/ailang/messages_send.go`
- [ ] Add `--space NAME` flag to `cmd/ailang/messages_search.go`
- [ ] Create `cmd/ailang/messages_triage.go`:
  - `ailang messages triage --inbox NAME` — cluster unread messages by `code` similarity (default)
  - `--cluster-by SLOT` — choose which envelope slot to cluster on
  - `--top N` — show top-N priority clusters
  - Output: cluster labels with member counts and representative titles
- [ ] Add `ailang messages envelope compute MSG_ID --code FILE` for retroactive enrichment
- [ ] Integration tests for CLI commands

---

## Files to Create/Modify

**New files:**
| File | LOC Est. | Purpose |
|------|----------|---------|
| `internal/messaging/envelope.go` | ~200 | Envelope struct, JSON, slot access |
| `internal/messaging/envelope_builder.go` | ~300 | Builder with functional options |
| `internal/messaging/envelope_test.go` | ~250 | Unit tests for envelope + builder |
| `internal/messaging/embedder_openai.go` | ~120 | OpenAI embedding provider |
| `internal/messaging/embedder_gemini.go` | ~120 | Gemini embedding provider |
| `cmd/ailang/messages_triage.go` | ~150 | Triage subcommand |

**Modified files:**
| File | Changes | LOC Est. |
|------|---------|----------|
| `internal/messaging/embedder.go` | Factory, extended config | +50 |
| `internal/messaging/schema.go` | v1.8.0 migration | +30 |
| `internal/messaging/inbox.go` | Envelope field, insert/scan | +40 |
| `internal/messaging/search.go` | EnvelopeSpace, SearchByEnvelope | +80 |
| `internal/coordinator/daemon_tasks_exec.go` | Resolution hook | +40 |
| `cmd/ailang/messages_send.go` | --envelope-* flags | +40 |
| `cmd/ailang/messages_search.go` | --space flag | +30 |

**Total:** ~1,450 LOC new/modified

---

## Examples

### Before: Single-dimension search

```bash
# All searches operate on the same text-derived embedding
ailang messages search "parser crash"           # Finds messages with "parser" and "crash" in text
ailang messages search "type inference bug"     # Finds messages with "type" and "inference" in text
# No way to distinguish: same code region vs same intent vs same skill needed
```

### After: Multi-space search

```bash
# Search by what code is affected
ailang messages search --space code "internal/types/unify.go"
# → Finds all messages about unification code, regardless of how they're described

# Search by what action is needed
ailang messages search --space intent "fix crash"
# → Finds crash-fix requests, excludes feature requests even if about same code

# Find past resolutions for similar problems
ailang messages search --space resolution --similar-to MSG_ID
# → Retrieves how similar bugs were resolved, guiding the current fix approach

# Triage inbox by code clusters
ailang messages triage --cluster-by code --inbox user
# Cluster 1 (8 msgs): Type system [internal/types/]
# Cluster 2 (5 msgs): CLI commands [cmd/ailang/]
# Cluster 3 (3 msgs): Eval harness [internal/eval_harness/]
# Cluster 4 (2 msgs): Documentation [docs/]
```

### Sending with context

```bash
# Current: just text
ailang messages send executor "Fix the type variable bug" --title "Bug: TypeVar"

# New: with semantic envelope
ailang messages send executor "Fix the type variable bug" \
  --title "Bug: TypeVar" \
  --envelope-code internal/iface/builder.go:50-80 \
  --envelope-context "Reviewing ast.Type switches, found missing TypeVar case"
```

---

## Success Criteria

- [ ] `Envelope` struct stores 5 named float32 vectors in JSON column
- [ ] `EnvelopeBuilder` computes each slot from the correct source
- [ ] `NewEmbedderFromConfig()` creates Ollama, OpenAI, or Gemini embedders
- [ ] `SearchByEnvelope()` returns different results for different spaces (tested)
- [ ] Resolution hook fires on coordinator task completion and populates slot
- [ ] `ailang messages search --space code` works end-to-end
- [ ] `ailang messages triage` clusters messages by similarity
- [ ] Messages without envelopes work identically to current behavior
- [ ] Schema migration v1.8.0 applies cleanly to existing databases
- [ ] All existing tests pass (`make test`)
- [ ] New code has 90%+ test coverage
- [ ] CHANGELOG.md updated
- [ ] CLAUDE.md messaging section updated

---

## Testing Strategy

**Unit tests:**
- Envelope serialization (JSON round-trip, empty envelope, partial slots)
- EnvelopeBuilder with mock embedder (verify each slot source)
- OpenAI/Gemini embedder with mock HTTP responses
- Multi-space search with pre-populated envelopes (different results per space)

**Integration tests:**
- Send message with envelope → read back → verify envelope preserved
- Send message → coordinator completes task → verify resolution slot filled
- Schema migration on database with existing messages (no data loss)

**CLI tests:**
- `--envelope-code FILE` reads file and embeds
- `--space code` routes to envelope search path
- `triage` produces clusters with expected groupings

**Manual testing:**
- Start session with 10+ unread messages → run `triage` → verify useful clustering
- Send message with `--envelope-code` → search `--space code` → verify retrieval

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Envelope JSON bloat (5 x 768 x 4 bytes = 15KB per message) | Medium | Use base64 binary encoding instead of JSON float arrays; slots are optional (most messages have only `intent`) |
| Ollama unavailable for intent auto-computation | Low | Intent computation is async and best-effort; message send never blocks on embedding |
| OpenAI/Gemini embedding dimension mismatch with Ollama | Medium | Normalize to fixed dimension per config; document that changing provider invalidates existing envelopes |
| Resolution hook fails (no git diff available) | Low | Resolution is optional enrichment; failure logged but doesn't affect task completion |
| Triage clustering quality poor with small message sets | Low | Minimum cluster size threshold; fall back to flat list for <5 messages |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Envelope slots are deterministic given same model + input; deterministic search ordering (score DESC, ID ASC) |
| A2: Replayability | +1 | Envelopes persist in SQLite; resolution vectors create auditable problem-to-solution history |
| A3: Effect Legibility | +1 | Embedding computation is explicit (requires embedder config); no hidden network calls |
| A4: Explicit Authority | 0 | No new capability requirements |
| A5: Bounded Verification | +1 | Envelope validation is local (check slot presence, dimension); no global reasoning needed |
| A6: Safe Concurrency | 0 | No concurrency changes; SQLite WAL handles concurrent access |
| A7: Machines First | +1 | Core value: vector communication channel *designed for* machine-to-machine semantic transfer |
| A8: Minimal Syntax | 0 | No new language syntax; CLI flags only |
| A9: Cost Visibility | +1 | Embedding costs visible via provider config; lazy computation avoids surprise API calls |
| A10: Composability | +1 | Builds on M-MSG-SEMANTIC (SimHash + single embedding); envelope is additive, not replacement |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | +1 | Clear boundary between message text (human-readable) and envelope (machine-readable) |

**Net Score: +8** Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): Same model + input = same embedding; no implicit nondeterminism
- [x] A3 (Effects): Embedding requires explicit provider config; no ambient network access
- [x] A4 (Authority): No new ambient capabilities granted
- [x] A7 (Machines First): Explicitly designed for machine communication; strengthens this axiom

---

## Related Documents

- [M-MSG-SEMANTIC](../../implemented/v0_6_0/m-msg-semantic-caching.md) — Semantic search + deduplication (v0.5.11) — **direct predecessor**
- [M-DOC-SEM](../../implemented/v0_6_0/m-doc-sem-lazy-embeddings.md) — Lazy embeddings for doc search — **lazy computation pattern reused**
- [Semantic Caching Future](../v0_8_1/semantic-caching-future.md) — Future work roadmap — **this doc implements a subset**
- [M-AGENT-PROTOCOL](../../implemented/v0_5_0/M-AGENT-PROTOCOL.md) — Agent-to-agent communication — **envelope extends the protocol semantically**
- [M-ARCH1: AI Provider Base Class](../v0_8_1/m-arch1-ai-provider-base-class.md) — Provider abstraction — **new embedders should follow this pattern when M-ARCH1 lands**
- [DX-16: SharedIndex](../../implemented/v0_6_0/dx-16-shared-index-deterministic-retrieval.md) — Deterministic retrieval — **envelope search may use SharedIndex as accelerator**

---

## Open Design Questions

1. **Storage format:** JSON float arrays (~45KB for full envelope) vs base64 binary (~20KB)? JSON is debuggable with `sqlite3`; binary is 2x more compact. Recommend: start with JSON (consistent with existing `embedding` column), add binary option later if storage becomes an issue.

2. **Dimension normalization:** If user switches from Ollama (768-dim) to OpenAI (1536-dim), existing envelopes become incomparable. Options: (a) invalidate and recompute on model change, (b) store dimension in envelope metadata and refuse to compare mismatched. Recommend: (b) with model+dimension tracking.

3. **Resolution granularity:** Should resolution embed the full diff, or just the commit message + file list? Full diffs can be large (>6000 chars, requiring chunking). Recommend: commit message + file paths + first 2000 chars of diff summary.

4. **Triage clustering algorithm:** K-means (fixed K) vs DBSCAN (auto-detect clusters) vs simple threshold grouping? Recommend: threshold-based greedy clustering (consistent with existing dedup approach in `FindDuplicates()`).

---

## Future Work

- **Per-slot model config** — Use code-specific embedding models (CodeLlama, StarCoder) for `code` slot, general models for `intent`
- **Agent capability profiles** — Each agent carries a persistent skill embedding; coordinator matches against message `skill` vectors for dynamic routing
- **Envelope-aware dedup** — Extend `FindDuplicates()` to compare specific envelope slots rather than just text SimHash
- **SharedIndex acceleration** — Use SharedIndex effect as an in-memory accelerator for envelope search (consistent with M-MSG-SEMANTIC design)
- **Fixed-size vector type** — When `vector[float; 768]` lands (semantic-caching-future.md section 5), use for compile-time envelope dimension checking in AILANG code

---

## Timeline

**Week 1** (16 hours):
- Phase 1: Envelope type + schema (4h)
- Phase 2: Provider-agnostic embedder (4h)
- Phase 3: Envelope builder (6h)
- Buffer (2h)

**Week 2** (16 hours):
- Phase 4: Multi-space search (4h)
- Phase 5: Resolution feedback loop (3h)
- Phase 6: CLI integration (4h)
- Testing + documentation (4h)
- Buffer (1h)

**Total: ~32 hours across 2 weeks**
