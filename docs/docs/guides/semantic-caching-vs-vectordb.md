---
sidebar_position: 9
title: Semantic Caching vs Vector DBs
description: When to use AILANG's semantic caching versus a dedicated vector database like ChromaDB
---

# Semantic Caching vs Vector Databases

AILANG's semantic caching is a **semantic memoization layer** at the agent/tool boundary. It's not a replacement for vector databases - it's a complementary tool that excels in different scenarios.

This guide helps you choose the right tool for your use case.

## The Key Insight

Traditional vector databases (ChromaDB, Pinecone, Weaviate) are designed for **retrieval-augmented generation (RAG)** - finding relevant documents from a large corpus.

AILANG's semantic cache is designed for **decision/tool/result memoization** - avoiding redundant work in agent loops.

| Aspect | Semantic Cache | Vector Database |
|--------|---------------|-----------------|
| **Primary purpose** | Memoization | Retrieval |
| **Scope** | Ephemeral, bounded | Long-lived corpus |
| **Trust model** | Heuristic (re-validate) | Source of truth |
| **Typical TTL** | Hours to days | Months to years |
| **Typical size** | 100s-1000s entries | Millions of documents |

---

## When Semantic Caching Wins

### 1. Agent-to-Agent Dedupe and "Already Handled" Suppression

**The problem**: Multiple agents rediscover the same issue with slightly different phrasing, causing fan-out storms.

**Why cache wins**: It's at the message boundary - closer to causality than retrieval.

```bash
# Detect near-duplicate bug reports before processing
ailang messages search "parser crash on nested records" --threshold 0.90

# If similar exists, link rather than create new
ailang messages list --similar-to MSG_ID
```

**Use cases**:
- Prevent repeated escalations of the same issue
- Stop N agents from generating N fixes for the same bug
- Recognize "known bad" states during triage

### 2. Tool-Result Caching (The Real Cost Sink)

**The problem**: Agents repeatedly call expensive tools with semantically similar inputs.

**Why cache wins**: You want **idempotence and latency collapse**, not long-lived retrieval.

**What to cache**:
- `git diff` / `ripgrep` results with similar queries
- Test failures + stack traces + environment fingerprint
- API responses ("this endpoint returned 429")
- Schema introspection, SQL EXPLAIN output
- Build artifacts keyed by dependency fingerprint

```ailang
-- Cache expensive tool results with SimHash key
func cached_git_diff(commit: string) -> string ! {IO, SharedMem, SharedIndex} {
  let key = "tool:git_diff:" ++ commit;
  match load_frame(key) {
    Some(frame) => _bytes_to_string(frame.opaque),
    None => {
      let result = _shell("git diff " ++ commit);
      let _ = store_frame(key, make_frame_at(key, "git diff " ++ commit, _bytes_from_string(result), _clock_now(())));
      result
    }
  }
}
```

### 3. Build/CI Acceleration (Semantic "Same Failure" Recognition)

**The problem**: CI fails with a similar error to yesterday's failure, but agents re-investigate from scratch.

**Why cache wins**: You can enforce TTL, scope (repo/branch), and "only trust if build fingerprint matches."

```bash
# Match current failure to prior one
ailang messages search "undefined: parseModuleDecl" --inbox ci-failures

# Retrieve prior fix plan + patch + commit reference
ailang messages read PRIOR_MSG_ID
```

**Cache can store**:
- Failure signature → successful fix mapping
- Build fingerprint → known issues
- Test name → common root causes

### 4. Multi-Turn Local Coherence: Don't Re-Derive Conclusions

**The problem**: RAG finds *new* relevant context. But what about context you *already derived* in this session?

**Why cache wins**: It's a **decision cache**, not a knowledge base.

**Examples**:
- "This codebase uses effect rows - we checked in turn 3"
- "We decided: no raw SQL, only query builders"
- "The chosen approach for DX-15 was tiered similarity"

```ailang
-- Store a decision for session coherence
let _ = store_frame("decision:sql_policy", make_frame_at(
  "decision:sql_policy",
  "Policy: no raw SQL, only query builders",
  _bytes_from_string("Decided in sprint planning, rationale: ..."),
  _clock_now(())
))
```

### 5. Near-Duplicate Document Intake

**The problem**: Before storing in a vector DB, you need to detect duplicates and extract deltas.

**Why cache wins**: Vector stores don't want to be your dedupe gate. Semantic cache is perfect as the "front door."

```bash
# Before ingesting a PDF
ailang messages search "Q3 sales report" --threshold 0.95

# If 95%+ match exists, store only delta
# Otherwise, proceed to full embedding
```

### 6. Policy & Safety Guardrails as Effects

**The problem**: Vector DBs do filtering, but the semantics live outside your language/runtime.

**Why cache wins**: Constraints are first-class effects in AILANG.

```ailang
-- Cache with provenance requirements
func store_with_provenance(key: string, content: string, source_hash: string)
  -> unit ! {SharedMem, SharedIndex} {
  let frame = make_frame_at(key, content, _bytes_from_string(source_hash), _clock_now(()));
  -- Effect system ensures this runs with proper capabilities
  store_frame(key, frame)
}
```

**Constraints you can enforce**:
- Forbid reusing cached results if provenance doesn't match
- Require `source_hash` / `tool_fingerprint`
- Enforce "don't retrieve across tenants/projects"
- Enforce TTL for sensitive content

### 7. Experience Replay (Plans, Not Facts)

**The problem**: You want agents to learn from past successes, but not treat retrieval as "new truth."

**Why cache wins**: Retrieved items are **heuristics that must be re-validated**, not facts.

```ailang
-- Store successful trajectory keyed by problem signature
let problem_sig = _simhash("type error on line 42 in parser.go");
let _ = store_frame("trajectory:" ++ show(problem_sig), make_frame_at(
  "trajectory:" ++ show(problem_sig),
  "Fix: check for nil pointer before dereferencing",
  _bytes_from_string(patch_content),
  _clock_now(())
))

-- Later: retrieve as heuristic, not truth
let candidates = _sharedindex_find_simhash("trajectory", problem_sig, 3, 100, true);
-- Re-validate each candidate before applying!
```

### 8. Observability Compression

**The problem**: Long traces are expensive to store and search.

**Why cache wins**: Compress traces to semantic summaries, match new traces to old ones.

```ailang
-- Compress trace to summary frame
let trace_summary = summarize_trace(full_trace);  -- AI call or heuristic
let _ = store_frame("trace:" ++ trace_id, make_frame_at(
  "trace:" ++ trace_id,
  trace_summary,
  _bytes_from_string(full_trace),  -- Original in opaque
  _clock_now(())
))
```

### 9. Cache as Coordination Primitive

**The problem**: Vector stores don't give you atomic coordination.

**Why cache wins**: CAS (compare-and-swap) + similarity enables distributed coordination.

**Patterns**:
- Distributed "claim this task if similar to X" locks
- "Only one agent generates final patch for this issue signature"
- Consensus convergence: "we already have a canonical plan for this cluster"

```ailang
-- Atomic claim with CAS
match update_frame("claim:" ++ issue_sig, \frame.
  if frame.content == "unclaimed" then
    {frame | content: "claimed:" ++ agent_id}
  else
    frame  -- Already claimed, no change
) {
  Updated(_) => proceed_with_fix(),
  Conflict(_) => skip_already_claimed(),
  Missing => create_and_claim()
}
```

---

## When Vector Databases Win

Use ChromaDB, Pinecone, or similar when you need:

| Requirement | Why Vector DB? |
|-------------|----------------|
| **Long-lived corpus search** | Designed for millions of documents |
| **Hybrid search** | Keyword + semantic + metadata filtering |
| **Ranking tuning** | MMR, custom re-rankers, query expansion |
| **Index lifecycle** | Backfills, migrations, versioning |
| **Knowledge as a product** | Auditable, exportable, queryable by others |
| **Cross-context retrieval** | Find relevant docs from anywhere |

---

## A Clean Architecture Split

Avoid "accidental RAG" by keeping boundaries clear:

### Semantic Cache = Ephemeral, Scoped, Causal

- **Key by**: `(repo, branch, tool_fingerprint, task_type)`
- **TTL**: Manual cleanup (use `ailang messages cleanup --older-than 7d`)
- **Stores**: Tool outputs, failure signatures, plans, patches, summaries
- **Trust**: Heuristic - always re-validate before acting

### Vector Store = Durable, Cross-Context, Informational

- **Key by**: Document ID, stable over time
- **TTL**: Months/years, governance-controlled
- **Stores**: Docs, ADRs, manuals, contracts, product knowledge
- **Trust**: Source of truth (with appropriate access controls)

### Wiring Them Together

```
┌─────────────────────────────────────────────────────────────┐
│                    Incoming Document                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Semantic Cache (Ingestion Gate)                │
│  • Dedupe: Is this 95% same as existing?                    │
│  • Delta: Extract only changed sections                     │
│  • Skip: Don't re-embed if already processed                │
└─────────────────────────────────────────────────────────────┘
                              │
                    (only new/changed content)
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                Vector Store (The Library)                   │
│  • Full embedding                                           │
│  • Rich metadata indexing                                   │
│  • Long-term retention                                      │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│          Semantic Cache (Loop Accelerator)                  │
│  • Memoize: Cache RAG results for similar queries           │
│  • Compress: Store query→answer for fast replay             │
│  • Coordinate: CAS for "one agent handles this"             │
└─────────────────────────────────────────────────────────────┘
```

---

## Design Decisions & Future Work

### Sound vs Useful?

**Current design**: Useful (sometimes wrong but fast).

- SimHash is approximate by nature
- Thresholds are configurable per use case
- Dedupe is "safe by default" (report-only mode)
- Neural search is opt-in for higher accuracy

**Trade-off**: We accept false positives (treating different things as similar) in exchange for speed. The mitigation is making it easy to inspect and override.

### Frame Identity

**Current implementation**:
- Content: `title + payload` → SimHash
- Namespace: `inbox` / custom prefix
- Ordering: `timestamp`, `version`

**Not yet implemented** (future work):
- `tool_fingerprint` (hash of tool version + config)
- `repo_hash` (git SHA or content hash)
- `env_hash` (environment fingerprint)

### Actionable Artifacts vs Raw Evidence

**Current**: Both stored in `opaque` field, distinguished by:
- `category`: bug, feature, general
- `message_type`: notification, request, response
- Convention: `content` is summary, `opaque` is full data

**Future work**: Explicit trust levels, provenance tracking.

### Negative Caching ("This Failed")

**Current**: Not explicit. `dup_of` is for positive deduplication.

**Future work**:
- `failed_attempts` field or dedicated namespace
- "Don't retry this approach" markers
- Failure signature → known dead-end mapping

---

## Quick Decision Guide

| Scenario | Use |
|----------|-----|
| "Is this bug report a duplicate?" | **Semantic cache** |
| "Find all docs about authentication" | **Vector DB** |
| "Cache this expensive git diff" | **Semantic cache** |
| "Build a searchable knowledge base" | **Vector DB** |
| "Prevent agents from re-deriving same conclusion" | **Semantic cache** |
| "Enable RAG over product documentation" | **Vector DB** |
| "Dedupe before ingesting to vector DB" | **Semantic cache** |
| "Atomic task claiming across agents" | **Semantic cache** |

---

## See Also

- [Semantic Caching Guide](/docs/guides/semantic-caching-how-to) - How to use SharedMem/SharedIndex
- [Semantic Search](/docs/guides/semantic-search) - SimHash and neural embeddings
- [Agent Messaging](/docs/guides/agent-messaging) - Message search and deduplication
