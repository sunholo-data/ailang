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

| Scenario | Why the cache wins |
|----------|-------------------|
| **Agent-to-agent dedupe** — N agents rediscover the same issue with different phrasing | Sits at the message boundary, closer to causality than retrieval; `ailang messages search "..." --threshold 0.90` before creating new work |
| **Tool-result caching** — repeated expensive calls (`git diff`, test runs, API responses) | You want idempotence and latency collapse, not long-lived retrieval |
| **CI "same failure" recognition** | Failure signature → prior fix mapping, scoped by repo/branch/fingerprint with TTL |
| **Session coherence** — don't re-derive conclusions already reached this session | It's a *decision cache*, not a knowledge base |
| **Ingestion dedupe gate** — detect duplicates before embedding into a vector DB | Vector stores don't want to be your dedupe front door |
| **Provenance/policy guardrails** | Constraints are first-class effects in AILANG, not external filter config |
| **Experience replay** — reuse past successful plans | Retrieved items are heuristics to re-validate, never new truth |
| **Trace compression** — match new traces to semantic summaries of old ones | Summary in `content`, full trace in `opaque` |
| **Coordination primitive** — "one agent claims this issue signature" | CAS + similarity gives atomic distributed claims |

Two of these patterns in code — tool-result caching:

```ailang
-- Cache expensive tool results with SimHash key
func cached_git_diff(commit: string) -> string ! {IO, SharedMem, SharedIndex} {
  let key = "tool:git_diff:${commit}";
  match load_frame(key) {
    Some(frame) => _bytes_to_string(frame.opaque),
    None => {
      let result = _shell("git diff ${commit}");
      let _ = store_frame(key, make_frame_at(key, "git diff ${commit}", _bytes_from_string(result), _clock_now(())));
      result
    }
  }
}
```

and atomic claim via CAS:

```ailang
match update_frame("claim:${issue_sig}", \frame.
  if frame.content == "unclaimed" then
    {frame | content: "claimed:${agent_id}"}
  else
    frame  -- Already claimed, no change
) {
  Updated(_) => proceed_with_fix(),
  Conflict(_) => skip_already_claimed(),
  Missing => create_and_claim()
}
```

The full pattern catalog with runnable code lives in the
[Semantic Caching Guide](/docs/guides/semantic-caching-how-to).

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

## Design Trade-off

The cache is **useful, not sound**: SimHash is approximate, so false positives
(treating different things as similar) are accepted in exchange for speed.
The mitigation is inspectability — thresholds are configurable per use case,
dedupe is report-only by default, and neural search is opt-in for higher
accuracy. Always re-validate retrieved results before acting on them.

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
