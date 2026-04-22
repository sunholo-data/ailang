# M-BRAIN-MICRORAG: Micro-RAG — Just-in-Time Knowledge Injection on Tool Calls

> Marketed as **"micro-RAG"** (μRAG): retrieval-augmented generation, but with ≤150-token *pointer* snippets injected at the moment of action — not multi-KB context dumps. Engineered for AI prompt-cache stability and harness portability.

**Status**: IMPLEMENTED
**Target**: v0.15.0
**Priority**: P1 (High — directly improves every Claude Code AILANG session, and any other harness wired to the engine)
**Estimated**: 5 days (20h implementation + 8h testing + 4h docs/rollout)
**Dependencies**: M-BRAIN (v0.9.2 ✅), M-BRAIN-VECTORS (v0.9.4 ✅), M-BRAIN-CONTEXT (v0.10.0 ✅)
**Milestone ID**: M-BRAIN-MICRORAG

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same brain state + same query + same dedup ledger = same injection. Token-window dedup is a deterministic function of the session's injection history. |
| A2: Replayability | +1 | All injections logged to `injections.jsonl` with content hash + token position; sessions are replayable from the ledger. |
| A3: Effect Legibility | +1 | Hooks declared in `settings.json`; injected content carries an explicit `🧠 μRAG [namespace]` marker; the model sees what was injected and why. |
| A4: Explicit Authority | +1 | Reuses existing `effects.SharedMem` capability via `ailang cache search`; no new ambient access. Glob router config is explicit, not inferred. |
| A5: Bounded Verification | 0 | No verification surface change. |
| A6: Safe Concurrency | 0 | Hooks fire synchronously per tool call; cache files are session-scoped (single-writer). |
| A7: Machines First | +2 | Core value: surfaces machine-consumable context exactly when the machine is about to act. Token-window dedup is calibrated to model recall, not human convenience. |
| A8: Minimal Syntax | 0 | No language syntax change. |
| A9: Cost Visibility | +1 | Per-injection token cost tracked in ledger; session budget is configurable; AI prompt-cache hit rate observable via existing telemetry. |
| A10: Composability | +1 | Reuses three-tier search, embedder abstraction, and hook plumbing. Engine + frontend split composes additional harnesses (MCP, CLI shim) without engine changes. |
| A11: Structured Failure | +1 | Graceful degradation: hook failures `exit 0` silently; corrupt ledger truncated and rebuilt; missing embedder falls back to FTS. |
| A12: System Boundary | +1 | Boundary is explicit: tool-call event → frontend adapter → engine CLI → glob match → search → dedup → injection. Each step inspectable. |

**Net Score: +9** → **Decision: Strong Accept**

### Hard Violation Check

- [x] A1 (Determinism): Token-window dedup is deterministic given ledger state; clock-based wall-clock cap is a cache-safety belt, not the primary signal.
- [x] A3 (Effects): Hooks visible in `settings.json`; injection format clearly labeled.
- [x] A4 (Authority): No new capabilities; uses existing `ailang cache search`.
- [x] A7 (Machines First): The whole feature exists to serve machine consumers.

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

**Score +9** → strong accept.

## Problem Statement

AILANG is not in any model's training data. Today's brain context injection is **front-loaded**: SessionStart fires `brain_session.sh` once, and `brain_context.sh` (M-BRAIN-CONTEXT v0.10.0) fires only on `PreToolUse(Read)`. When Claude is *writing* AILANG — `Edit` or `Write` on a `.ail` file — there is no JIT injection at all.

**Current State:**
- SessionStart injection: 1 query per session, based on last 3 commits' files (coarse signal).
- PreToolUse(Read) injection: only when the model is *reading*, not when it's *writing*.
- 253 builtins discoverable via `ailang builtins show`, but the model never sees a signature unless it asks.
- Recent breaking changes (e.g., v0.13.0: `++` is list-only) are silently re-violated by models trained on older AILANG examples.
- The `use-ailang` skill exists but is all-or-nothing: progressive-disclose nothing or pull a 5–10KB block.
- No way to A/B test the value of context injection — it's either on for everyone or off for everyone.
- All injection logic lives in Claude-Code-specific bash hooks; no path to other coding harnesses.

**Quantifiable gap:**
- M-EVAL-GAP analysis catalogs 90+ PAR_001 (parse/syntax) errors, 40+ missing-import errors, 15+ wrong-operator errors (string vs list `++`).
- Eval suite shows ~75% pass rate (recent 99/132 run); a meaningful slice of the 25% failure tail is preventable mistakes the brain already knows about.
- The brain currently has 438 project frames + 403 user frames; only ~3 of those surface per session.

**Impact:**
- Models repeat known mistakes mid-session because the nudge arrives too late or never.
- Token waste: re-reading the `use-ailang` skill costs more than a 150-token pointer would.
- AI prompt-cache thrash risk: today's per-Read hook can re-inject different content on each Read of the same file, breaking cache-friendly stable suffixes.
- Cannot measure value: without an on/off toggle, no eval evidence for whether brain injection is actually helping.
- Locked to Claude Code: investment in hook-only architecture means Cursor/Continue/Aider users get nothing.

## Goals

**Primary Goal:** Inject relevant, deduplicated, ≤150-token pointer-style context exactly when a model is about to act on (or just acted on) a `.ail` file — without thrashing the AI prompt cache, without overwhelming the context window, with a kill switch for eval A/B testing, and via an engine that any coding harness can call.

**Success Metrics:**
1. **Coverage:** ≥80% of Edit/Write events on `.ail` files trigger a relevance check (the rest excluded by config or dedup).
2. **Precision:** ≥70% of injected snippets are judged relevant by manual sampling of 50 sessions.
3. **AI cache stability:** Prompt-cache hit rate over 5-edit bursts ≥0.6 (measured via `claude_telemetry.sh`).
4. **Token economy:** ≤5000 injection tokens per session at p95; per-injection p95 ≤150 tokens.
5. **Latency:** Engine p95 <200ms (cache hit), <1.5s (cold embed). Frontend overhead <20ms.
6. **Mistake reduction:** PAR_001 + wrong-operator error count in eval suite drops by ≥20% on `claude-sonnet-4-6` baseline within 2 weeks of rollout (measured against the OFF arm of the A/B).
7. **Knowledge-base-agnostic:** Adding a second route (e.g., `*.go` → `project-resolutions`) requires only a config edit, no code change.
8. **Eval-suite A/B measurable:** Eval runner can flip μRAG on/off per benchmark group via a single env var; results are tagged with `microrag_state` for direct comparison. ON vs OFF deltas reported in `eval_results/`.
9. **Harness-portable:** Same μRAG engine usable from at least one non-Claude-Code harness (MCP server proof-of-concept) before v0.15.0 ships.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Token-window dedup vs wall-clock dedup | Wall-clock thrashes AI cache; token-window is calibrated to model recall. Wrong choice = wasted tokens or wasted recall. | human | design | high |
| Per-injection cap (150 tokens) vs adaptive | Fixed cap is predictable for cache; adaptive could over-inject early in session. | agent | design | med |
| PreToolUse on Edit/Write vs PostToolUse | Pre = nudge before mistake; Post = catch what was actually written. Both serve different purposes. | human | design | high |
| Generic glob-router config vs AILANG-only hardcoded | Generic enables future knowledge bases; hardcoded is faster to ship. | human | design | high |
| Search-result cache TTL (4 min) vs no cache | Must stay under Anthropic 5-min cache TTL to keep injection prefix stable. | agent | design | med |
| Bootstrap `ailang-syntax` namespace from prompts/ vs reuse `resolutions` | Resolutions is commit-history; syntax needs a curated corpus. | human | design | low |
| Engine in CLI (`ailang micro-rag context`) vs inline in hook script | CLI engine = harness-portable; inline = simpler but Claude-Code-only. CLI cost is one extra process spawn (~10ms). | human | design | high |
| Ship MCP server in v0.15.0 vs defer | Shipping with MCP proves harness portability; deferring keeps scope tight but risks Claude-Code-only lock-in shaping decisions. | human | design | med |
| Eval toggle as env var vs config flag | Env var is harness-agnostic and zero-overhead; config flag requires reload. | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **Token-window dedup** is the primary signal; wall-clock cap (240s) is a cache-safety belt only.
- [x] **Both PreToolUse(Edit|Write|Read) and PostToolUse(Edit|Write)** hooks ship together (different purposes).
- [x] **Generic glob-router** with YAML config (`~/.ailang/microrag.yaml`); AILANG is the first route, not the only one.
- [x] **Search-result cache TTL = 240s** (under Anthropic's 5-min prompt-cache TTL).
- [x] **Per-namespace dedup windows + relevance-score bypass** (see Solution Design table).
- [x] **Replace `brain_context.sh`** with generic `microrag_context.sh`; old script kept in repo for one release as fallback.
- [x] **Engine lives in CLI (`ailang micro-rag context`)**, not inline in the bash hook. The hook is a thin shim. Enables harness portability.
- [x] **Eval toggle**: `AILANG_MICRORAG_ENABLED` (0/1) is the master switch read by both CLI and hook on every invocation. `AILANG_MICRORAG_ROUTES` (comma-list) is a per-route allowlist for granular A/B.
- [x] **MCP server (`ailang-microrag-mcp`)** ships in v0.15.0 as the second frontend (proof of harness portability).

## Solution Design

### Overview

A pair of frontends (Claude Code hooks, MCP server) plus a harness-agnostic engine (`ailang micro-rag`) plus a glob-router config plus a one-time syntax-corpus indexer. The system fires on every `Edit|Write|Read` (PreToolUse) and `Edit|Write` (PostToolUse), but **suppresses 90%+ of would-be injections** via cached search and token-window dedup. When something does inject, it's a ≤150-token pointer with a clear marker, formatted to keep the AI prompt-cache prefix stable. A single env var (`AILANG_MICRORAG_ENABLED=0`) disables the system end-to-end for eval A/B.

### Architecture

The engine is harness-agnostic. Frontends adapt the engine's I/O contract to whatever a given coding harness exposes (hooks, MCP tools, plain CLI).

```
┌──────────────────────────────────────────────────────────────────────┐
│                          FRONTENDS (adapters)                          │
│                                                                        │
│  Claude Code           MCP-compatible          Any harness with        │
│  hooks                 harness (Cursor,        shell access            │
│                        Continue, Cline)        (Aider, Codex CLI)      │
│  ┌──────────────────┐ ┌──────────────────┐    ┌──────────────────┐   │
│  │microrag_context.sh│ │ailang-microrag- │    │direct CLI calls  │   │
│  │microrag_lint.sh   │ │mcp (MCP server) │    │from harness ext  │   │
│  └─────────┬─────────┘ └────────┬────────┘    └────────┬─────────┘   │
│            │                    │                       │              │
│            └────────────────────┴───────────────────────┘              │
│                                 │                                      │
│                                 ▼                                      │
│  ┌──────────────────────────────────────────────────────────────┐    │
│  │                ENGINE (harness-agnostic, Go)                   │    │
│  │  ailang micro-rag context --tool=Edit --file=foo.ail \        │    │
│  │                           --content=...                        │    │
│  │  ailang micro-rag lint-builtin --code=...                     │    │
│  │                                                                │    │
│  │  Honors: AILANG_MICRORAG_ENABLED, AILANG_MICRORAG_ROUTES      │    │
│  │          (eval toggle — zero-cost early-exit if disabled)     │    │
│  │  Reads:  ~/.ailang/microrag.yaml                              │    │
│  │  Performs:                                                     │    │
│  │    1. Glob router (file → KB namespace)                       │    │
│  │    2. Search-result cache (hash(file+ns+content), 240s TTL)   │    │
│  │    3. Embed query (24h cache, Ollama nomic-embed-text)        │    │
│  │    4. ailang cache search (three-tier: cosine/simhash/FTS)    │    │
│  │    5. Token-window dedup + relevance bypass                   │    │
│  │    6. Format injection (≤150 tokens, stable bytes)            │    │
│  │    7. Append to ledger                                         │    │
│  │  Returns:                                                      │    │
│  │    JSON {injection_text, snippet_id, tokens, ns, microrag_state} │ │
│  └──────────────────────────────────────────────────────────────┘    │
│                                 │                                      │
│                                 ▼                                      │
│  ~/.ailang/state/sessions/<sid>/                                      │
│    ├─ injections.jsonl   (token-window dedup ledger)                  │
│    ├─ search_cache/      (4-min TTL, hash-keyed JSON)                 │
│    ├─ embed_cache/       (24h TTL, query-hash-keyed)                  │
│    └─ builtins_seen.txt  (first-use tracker, append-only)             │
└──────────────────────────────────────────────────────────────────────┘
```

**Why an engine + frontends split:** the search/dedup/format logic should run identically regardless of harness. Each frontend is then a small adapter (≤100 LOC) translating between the harness's hook/extension API and the engine's CLI. Cost is one extra process spawn per call (~10ms) — negligible vs the 200ms–1.5s search latency. Crucially, the eval toggle works in *all* frontends because it's checked in the engine.

**Components:**

1. **Engine: `ailang micro-rag context` / `ailang micro-rag lint-builtin`** (Go, harness-agnostic). Does all the real work: env-toggle check, glob routing, search, dedup, formatting. Outputs JSON.
2. **Frontend A — Claude Code hooks (`microrag_context.sh`, `microrag_lint.sh`).** ~50 LOC bash each. Read tool-call JSON from stdin, shell out to `ailang micro-rag`, wrap result in Claude Code's `additionalContext` format.
3. **Frontend B — MCP server (`ailang-microrag-mcp`).** Exposes two MCP tools: `microrag_context_for_file(file_path, tool_name, content?)` and `microrag_lint_builtin(code)`. For Cursor, Continue, Cline, and any other MCP-compatible harness. Model invokes the tool explicitly (vs auto-injection in Claude Code).
4. **Frontend C — direct CLI shell-out.** Documented invocation pattern for any harness without hook or MCP support. Same `ailang micro-rag` binary.
5. **Glob router config (`~/.ailang/microrag.yaml`).** Maps file globs → KB namespace + per-namespace dedup window + relevance bypass threshold. Knowledge-base-agnostic.
6. **Cache layers.** `embed_cache/` (24h, query-text-hash) avoids re-embedding identical queries. `search_cache/` (240s, query+kb+limit hash) keeps injection prefix stable across edit bursts so Anthropic's 5-min prompt cache stays warm.
7. **Token-window dedup ledger (`injections.jsonl`).** Append-only per session. Each line: `{ts, snippet_id, tokens, file_path, kb, microrag_state}`. New injections check this tail to suppress recently-shown snippets. `microrag_state` field enables retroactive A/B analysis.
8. **Syntax-corpus indexer (`tools/index_ailang_syntax.sh`).** Resolves the *active* prompt version via `ailang prompt --version-active` (never hardcoded), chunks the resolved prompt by h2/h3 headers, indexes each builtin's signature, indexes `examples/runnable/*.ail` headers, all into a new `ailang-syntax` brain namespace. Auto-runs at every release via the `release-manager` skill (see "Corpus Freshness" below).
9. **Eval-suite integration.** `ailang eval-run --microrag=on|off|auto` flag sets `AILANG_MICRORAG_ENABLED` in benchmark subprocess env. Results in `eval_results/` carry `microrag_state` per benchmark, so dashboard can show ON-vs-OFF deltas.

### Per-namespace dedup configuration (the calibration table)

The default windows are calibrated against actual model recall, not paranoia. With 200K context windows, even 80K of dedup is only 40% — well within the high-recall range.

| Namespace | Token window | Relevance bypass | Rationale |
|---|---|---|---|
| `ailang-breaking-changes` | 15000 | ≥0.60 | Cost of slipping (e.g. `++` v0.13.0) is high; re-remind aggressively. |
| `ailang-syntax` | 30000 | ≥0.70 | Most syntax stays in recall within this range. |
| `ailang-builtins` | 80000 | ≥0.80 | Once the model sees a signature, it rarely needs it again. |
| `project-resolutions` | 40000 | ≥0.70 | Project context decays slower than syntax facts. |
| `default` | 30000 | ≥0.70 | Conservative default for new namespaces. |

**Why relevance bypass:** if a search result scores very high, the snippet matches the *exact* situation the model is in — recall is moot, contextual urgency wins. Bypass injects regardless of dedup window.

### Glob router config (`~/.ailang/microrag.yaml`)

```yaml
routes:
  - glob: "**/*.ail"
    kb: ailang-syntax
    max_tokens_per_injection: 150
    relevance_floor: 0.30
  - glob: "**/CHANGELOG.md"
    kb: ailang-breaking-changes
    max_tokens_per_injection: 150
    relevance_floor: 0.40
  - glob: "**/*.go"
    kb: project-resolutions
    max_tokens_per_injection: 200
    relevance_floor: 0.25
  - glob: "**/CLAUDE.md"
    kb: skip                         # don't inject for meta files

dedup:
  windows:
    ailang-breaking-changes: 15000
    ailang-syntax: 30000
    ailang-builtins: 80000
    project-resolutions: 40000
    default: 30000
  relevance_bypass:
    ailang-breaking-changes: 0.60
    ailang-syntax: 0.70
    ailang-builtins: 0.80
    default: 0.70
  wall_clock_max: 240                # 4-min hard cap (under Anthropic 5-min cache TTL)

session_budget: 5000                 # total injection tokens per session

# Eval toggle (runtime override of env vars, mostly for dev)
# Production: set AILANG_MICRORAG_ENABLED=0/1 instead.
enabled: true
```

### Eval-toggle contract

| Mechanism | Where checked | Effect |
|---|---|---|
| `AILANG_MICRORAG_ENABLED=0` | Engine entry (first 5 LOC) | Engine exits with `null` JSON; frontends emit no injection. Zero search cost. |
| `AILANG_MICRORAG_ENABLED=1` (default) | Engine entry | Normal operation. |
| `AILANG_MICRORAG_ROUTES=ailang-syntax,ailang-builtins` | After glob router | Suppress routes not in allowlist. Enables granular A/B (e.g., test only the breaking-changes route). |
| `AILANG_MICRORAG_DRYRUN=1` | After search, before format | Logs what *would* have been injected to `injections.jsonl` with `dryrun: true`, but emits no `additionalContext`. For shadow-mode measurement. |
| `microrag.yaml: enabled: false` | Engine entry | Same as `ENABLED=0` but config-driven. Env var wins on conflict. |

### Injection format (cache-friendly stable suffix)

```
━━━ 🧠 μRAG [ailang-syntax] ━━━
→ prompts/<active>.md §String Operators
  Since v0.13.0: ++ is list-only. Use "${expr}" for strings.
  See examples/string_interp.ail
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

The `<active>` placeholder is resolved at index time to whatever `ailang prompt --version-active` reports. Pointers always reference the prompt version that was current when the corpus was built — there is no version pinning in the doc or the engine. When a release ships a new prompt, the post-release reindex regenerates pointers automatically.

**Why this format:** clear marker (`🧠 μRAG [ns]`), pointer not body, fixed-width borders that the model can recognize as a system reminder vs. content. Repeated injections with identical content produce identical bytes → AI prompt cache stays warm.

### Corpus Freshness (release-tied reindexing)

A μRAG corpus that drifts behind the language is worse than no corpus — it confidently injects out-of-date guidance. Two design rules close this gap:

**Rule 1 — Always resolve the active prompt, never pin a version.**
- The indexer (`tools/index_ailang_syntax.sh`) calls `ailang prompt --version-active` to determine which prompt to chunk. Output is whatever the user's installed binary considers current.
- No design-doc, hook, config, or comment names a specific `prompts/vX.Y.Z.md`. References use either `<active>` placeholders (in templates) or the resolved version captured *at index time* (in the injected snippet body, so the model can verify the source).
- Builtins are pulled from `ailang builtins list --json` at index time — also unpinned.
- `examples/runnable/*.ail` is globbed at index time — new examples auto-included on next reindex.

**Rule 2 — Reindex on every release.** Manual `make brain-index-syntax` is fine for development, but the safe production cadence is *complete reindex at each release*:
- The `release-manager` skill (which owns the release workflow) gains a post-tag step: `make brain-index-syntax --reset`. The `--reset` flag (new) drops the existing `ailang-syntax` namespace and rebuilds from scratch — guarantees no stale chunks survive a prompt rewrite.
- The `post-release` skill verifies the reindex by spot-checking that the active prompt version appears in at least 5 indexed chunks; fails the release if not.
- Reindex is fast (~30s for ~300 chunks at local Ollama embedding rate); does not gate the release on slow infra.
- A short reindex audit (chunk count, namespace size, active version) is appended to the release notes, giving humans a one-line confirmation.

**What stays manual:** `make brain-index-syntax` (without `--reset`) remains available for dev-loop testing — adds new chunks, leaves existing ones, idempotent via stable keys. The destructive `--reset` form is reserved for releases.

**What this rules out:**
- No "smart diffing" of prompts to incrementally re-embed only changed chunks (over-engineering for a 30s job).
- No version-pinned `ailang-syntax-v0.12.1` namespaces (would multiply storage + create stale-fallback foot-guns).
- No injection of "stale corpus" warnings to the model (the corpus is *always* the latest, by construction).

### Implementation Plan

**Phase 1: Engine (Go CLI)** (~6 hours)
- [ ] Add `ailang micro-rag context` and `ailang micro-rag lint-builtin` subcommands in `cmd/ailang/microrag.go`.
- [ ] Implement env-toggle early-exit (`AILANG_MICRORAG_ENABLED`, `AILANG_MICRORAG_ROUTES`, `AILANG_MICRORAG_DRYRUN`).
- [ ] YAML config loader for `~/.ailang/microrag.yaml`.
- [ ] Glob router (Go's `path.Match` or `doublestar` lib).
- [ ] Search-result cache (filesystem JSON, 240s TTL).
- [ ] Token-window dedup (read tail of `injections.jsonl`, sum tokens since last `snippet_id`).
- [ ] Relevance-score bypass.
- [ ] Format injection (≤150 tokens, stable bytes).
- [ ] Append to `injections.jsonl` after each injection (with `microrag_state` tag).
- [ ] JSON output contract: `{injection_text, snippet_id, tokens, ns, microrag_state}` or `null`.

**Phase 2: Builtin first-use lint (engine subcommand)** (~3 hours)
- [ ] Implement identifier extraction (Go regex).
- [ ] Wire to `ailang builtins show <name> --json` (add `--json` flag if missing).
- [ ] Per-session `builtins_seen.txt`.
- [ ] Cap at 2 nudges per event.

**Phase 3: Syntax corpus bootstrap + release-tied reindex** (~4 hours)
- [ ] Write `tools/index_ailang_syntax.sh`.
- [ ] Add `--version-active` flag to `ailang prompt` (prints just the active version, machine-parseable). Today only `--list` exposes it (with `*` marker).
- [ ] Resolve active prompt via `ailang prompt --version-active` (no hardcoded version anywhere).
- [ ] Chunk the resolved prompt by h2/h3 headers (~50 chunks).
- [ ] Chunk `docs/LIMITATIONS.md` similarly.
- [ ] Index each builtin from `ailang builtins list --json` (253 chunks).
- [ ] Index `examples/runnable/*.ail` headers as tagged pointers.
- [ ] Stable keys for idempotency on re-run; record resolved prompt version in each chunk's metadata.
- [ ] Add `Makefile` targets: `make brain-index-syntax` (idempotent add) and `make brain-index-syntax-reset` (drop + rebuild — used by releases).
- [ ] Update `release-manager` skill to call `make brain-index-syntax-reset` after the version tag.
- [ ] Update `post-release` skill to spot-check the indexed corpus (≥5 chunks reference active prompt version) and append a one-line audit to release notes.

**Phase 4: Frontend A — Claude Code hooks** (~2 hours)
- [ ] Write `~/.ailang/hooks/microrag_context.sh` (~50 LOC bash shim).
- [ ] Write `~/.ailang/hooks/microrag_lint.sh` (~50 LOC bash shim).
- [ ] Update `/Users/mark/dev/sunholo/ailang/.claude/settings.json` to register both hooks.
- [ ] Replace existing `brain_context.sh` PreToolUse(Read) entry.
- [ ] Keep `brain_session.sh` (SessionStart) and `brain_resolution.sh` (PostToolUse on git commit) unchanged.
- [ ] Ship `microrag.yaml` template.

**Phase 5: Frontend B — MCP server** (~3 hours)
- [ ] Write `cmd/ailang-microrag-mcp/main.go` — MCP server exposing two tools.
- [ ] Tool: `microrag_context_for_file(file_path, tool_name, content?)`.
- [ ] Tool: `microrag_lint_builtin(code)`.
- [ ] Internally calls `ailang micro-rag` engine subcommands.
- [ ] Document MCP install for Cursor / Continue / Cline (`docs/docs/guides/microrag.md`).
- [ ] Smoke-test against at least one MCP harness (Cursor or Continue).

**Phase 6: Eval-suite integration** (~2 hours)
- [ ] Add `--microrag=on|off|auto` flag to `ailang eval-run`.
- [ ] `auto` = run each benchmark twice (ON, OFF) and tag.
- [ ] Subprocess env injection for `AILANG_MICRORAG_ENABLED`.
- [ ] Result JSON includes `microrag_state` field per benchmark.
- [ ] Dashboard view: ON vs OFF delta per benchmark, aggregated by error category.

**Phase 7: Testing + telemetry** (~6 hours)
- [ ] Unit tests for engine: glob matching, dedup ledger, cache TTL, env-toggle early-exit.
- [ ] Integration test: full session simulation with synthetic Edit/Write events.
- [ ] Telemetry: emit `microrag.injection.count`, `microrag.injection.tokens`, `microrag.cache.hit`, `microrag.dedup.suppressed`, `microrag.bypass.fired`, `microrag.disabled` via existing telemetry pipeline.
- [ ] Eval A/B validation: run small benchmark group ON and OFF; verify deltas computed correctly.
- [ ] Manual test: 1-week dogfooding in AILANG repo before global rollout.
- [ ] MCP smoke test in Cursor: confirm tool callable, context returned.

### Files to Modify/Create

**New files:**
- `cmd/ailang/microrag.go` — ~400 LOC Go; engine entry point + subcommands.
- `internal/microrag/engine.go` — ~600 LOC Go; glob router, cache, dedup, format.
- `internal/microrag/lint.go` — ~150 LOC Go; builtin first-use lint logic.
- `internal/microrag/config.go` — ~120 LOC Go; YAML config loader.
- `cmd/ailang-microrag-mcp/main.go` — ~250 LOC Go; MCP server.
- `~/.ailang/hooks/microrag_context.sh` — ~50 LOC bash; Claude Code shim.
- `~/.ailang/hooks/microrag_lint.sh` — ~50 LOC bash; Claude Code shim.
- `~/.ailang/microrag.yaml` — ~50 LOC YAML; glob router config (template).
- `tools/index_ailang_syntax.sh` — ~150 LOC bash; one-time syntax corpus indexer.
- `internal/builtins/cmd_show_json.go` — ~80 LOC Go; `--json` flag for `ailang builtins show` (if not already present).
- `cmd/ailang/prompt.go` — add `--version-active` flag (~10 LOC) so the indexer can resolve the latest prompt without parsing `--list`'s `*` marker.
- `docs/docs/guides/microrag.md` (new) — full user guide incl. MCP install.

**Modified files:**
- `/Users/mark/dev/sunholo/ailang/.claude/settings.json` — replace `brain_context.sh` PreToolUse entry with `microrag_context.sh`; add PostToolUse(Edit|Write) entry for `microrag_lint.sh`.
- `Makefile` — add `brain-index-syntax`, `microrag-mcp-build` targets.
- `internal/cache/search.go` (or wherever `ailang cache search` lives) — add `--json` flag if missing.
- `cmd/ailang/eval_run.go` — add `--microrag=on|off|auto` flag and subprocess env injection.
- `internal/eval_harness/result.go` — add `microrag_state` field to result schema.

**Reused as-is:**
- `internal/effects/cache_ops.go` — three-tier search.
- `effects.Embedder` Ollama provider — local nomic-embed-text.
- `~/.ailang/hooks/brain_session.sh` — SessionStart (unchanged).
- `~/.ailang/hooks/brain_resolution.sh` — PostToolUse(git commit) (unchanged).

## Examples

### Example 1: Edit on `.ail` file with `++ "string"`

**Before (today):**
1. Model writes `let x = "hello " ++ name in ...`
2. Compiler errors: `++ is list-only since v0.13.0`.
3. Model reads error, re-edits to use `"${name}"`.
4. ~2-3 turns wasted.

**After (with μRAG):**
1. PreToolUse(Edit) fires `microrag_context.sh` → calls `ailang micro-rag context`.
2. Engine searches `ailang-breaking-changes` namespace, hits with score 0.78 (above 0.60 bypass threshold).
3. Injects:
   ```
   ━━━ 🧠 μRAG [ailang-breaking-changes] ━━━
   → CHANGELOG.md §v0.13.0
     ++ is list-only. For strings use "${expr}" interpolation.
     See examples/string_interp.ail
   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   ```
4. Model writes `"${name}"` directly. 1 turn instead of 3.

### Example 2: First use of `concat_String`

**Before:** model has to either know the signature or call `ailang builtins show concat_String` (extra tool call).

**After:** PostToolUse fires `microrag_lint.sh`, detects `concat_String` not in `builtins_seen.txt`, injects:

```
━━━ 🔧 μRAG-BUILTIN ━━━
concat_String : ([String]) -> String
  module: builtins/string
  Concatenates a list of strings.
━━━━━━━━━━━━━━━━━━━━━━
```

Next turn, model has the signature in front of it without a separate lookup.

### Example 3: 5 edits on the same file in 30 seconds

**Without dedup:** 5 identical injections → AI cache invalidated 5 times.

**With μRAG:** first edit injects; edits 2–5 hit search-result cache (240s TTL) → identical bytes → AI cache stays warm → ~80% cache read rate.

### Example 4: Eval A/B run

```bash
# Single benchmark group, both arms automatically
ailang eval-run --microrag=auto --benchmarks=string-ops --models=claude-sonnet-4-6

# Result tagging in eval_results/2026-04-23-string-ops/results.json:
[
  {"benchmark": "concat_strings", "microrag_state": "off", "passed": false, "error": "PAR_001 ++ string"},
  {"benchmark": "concat_strings", "microrag_state": "on",  "passed": true,  "error": null},
  ...
]

# Dashboard:
# string-ops, claude-sonnet-4-6: μRAG ON 8/10 vs OFF 5/10 (+30%)
```

### Example 5: Cursor user via MCP

```jsonc
// .cursor/mcp.json
{
  "mcpServers": {
    "ailang-microrag": {
      "command": "ailang-microrag-mcp",
      "args": []
    }
  }
}
```

Cursor's model can then call `microrag_context_for_file({file_path: "examples/foo.ail", tool_name: "Edit", content: "..."})` and receive the same pointer-snippet payload Claude Code sees — *without any AILANG-team change to Cursor itself*.

## Success Criteria

- [ ] `ailang micro-rag context` and `ailang micro-rag lint-builtin` ship as engine CLI.
- [ ] `microrag_context.sh` + `microrag_lint.sh` ship as Claude Code shims, registered in project `.claude/settings.json`.
- [ ] `ailang-microrag-mcp` ships as MCP server binary; documented for at least one non-Claude-Code harness (Cursor or Continue) with a working smoke test.
- [ ] `~/.ailang/microrag.yaml` template installed via `ailang micro-rag init` or shipped in repo.
- [ ] `make brain-index-syntax` populates `ailang-syntax` namespace with ≥250 chunks.
- [ ] Engine p95 <200ms cache hit, <1.5s cold embed (measured over 100 fires); frontend overhead <20ms.
- [ ] Token-window dedup verified: same `snippet_id` not re-injected within configured window.
- [ ] Relevance bypass verified: scores ≥ threshold inject regardless of window.
- [ ] AI prompt-cache hit rate over 5-edit burst ≥0.6 (telemetry).
- [ ] Per-session injection budget (5000 tokens) enforced; warning emitted on overflow.
- [ ] **Eval toggle verified:** `AILANG_MICRORAG_ENABLED=0` produces zero injections in a full session; `AILANG_MICRORAG_ROUTES` allowlist filters as configured.
- [ ] **Eval A/B run completes:** `ailang eval-run --microrag=auto` runs both arms, results tagged with `microrag_state`, dashboard shows ON vs OFF deltas.
- [ ] Eval baseline run shows ≥20% reduction in PAR_001 + wrong-operator errors on `claude-sonnet-4-6` (ON vs OFF) after 2 weeks.
- [ ] Generic config verified: a second route (`*.go` → `project-resolutions`) works without code change.
- [ ] All tests passing.
- [ ] CHANGELOG.md updated.
- [ ] `docs/docs/guides/microrag.md` written.

## Testing Strategy

**Unit tests** (in `internal/microrag/*_test.go`):
- Env-toggle early-exit: `AILANG_MICRORAG_ENABLED=0` returns null with no search call (assert via mock).
- `AILANG_MICRORAG_ROUTES` allowlist suppresses non-listed routes.
- `AILANG_MICRORAG_DRYRUN=1` logs to ledger but emits no injection.
- Glob router: each route matches expected paths; `kb: skip` short-circuits.
- Dedup ledger reader: correctly sums tokens between current position and last `snippet_id` appearance.
- Cache TTL: search-result cache returns hit within 240s, miss after.
- Relevance bypass: score above threshold bypasses dedup.
- Session budget: 5001th injection token suppressed with warning.
- Builtin extraction regex: matches `concat_String(`, doesn't match `_concat_string` (private convention) or local vars.
- Corrupt ledger: truncated and rebuilt without crash.

**Integration tests:**
- Full session simulation: 30 synthetic Edit/Write events on `.ail` files; verify injection count, dedup effectiveness, budget compliance.
- Multi-route session: edits on `.ail`, `.go`, `CLAUDE.md`; verify each route fires correct namespace or skips.
- Cache stability: 5-edit burst on same file → first call cache miss, calls 2–5 cache hits, identical injection bytes.
- **Eval A/B integration:** `ailang eval-run --microrag=auto --benchmarks=string-ops` runs both arms, results carry correct `microrag_state` tags, dashboard renders deltas.
- **MCP smoke test:** start `ailang-microrag-mcp`, send MCP `tools/call` request for `microrag_context_for_file`, verify same JSON payload as direct CLI invocation.

**Manual testing:**
- 1-week dogfooding in AILANG repo: ship to project `.claude/settings.json` only.
- Sample 50 random injections; rate relevance manually; target ≥70% relevant.
- Run eval suite ON vs OFF; compare PAR_001 + wrong-operator counts; require ≥20% reduction.
- Install MCP server in Cursor; manually edit a `.ail` file; verify model can call the tool and receives expected payload.

**Negative tests:**
- `AILANG_MICRORAG_ENABLED=invalid` → defaults to enabled (don't fail closed).
- Disable Ollama → fall back to FTS-only search, still returns something.
- Empty/missing `microrag.yaml` → engine exits gracefully (no injection, no error).
- Non-`.ail` file Edit → no injection.
- `builtins show` returns non-zero (not a builtin) → silent skip.
- MCP server with engine binary missing → MCP tool returns structured error, doesn't crash harness.

## Deferred Decisions

- Specific token counts for `prompts/` chunk sizing — agent may choose (target ~500 tokens per chunk).
- Telemetry storage backend — agent may choose; reuse existing telemetry pipeline.
- `injections.jsonl` rotation strategy at session end — agent may choose; default: keep for 7 days then GC.
- Whether to ship `microrag.yaml` in-repo or via CLI bootstrap — agent at implementation.
- Format of the `🧠 μRAG` marker (emoji vs ASCII) — agent may choose; current draft uses emoji to distinguish from existing `🧠 BRAIN` SessionStart marker. The Greek μ may not render in all terminals — fallback to `[uRAG]` is acceptable.
- Whether `microrag_lint.sh` should also run on Read (e.g., reading an example) — human at review (defer to dogfooding feedback).
- Which non-Claude-Code harness to target first for MCP smoke test — agent (Cursor or Continue most likely).
- Whether eval `--microrag=auto` should run ON and OFF in parallel or sequentially — agent (parallel = faster, sequential = simpler tagging).

## Non-Goals

- **Full AILANG linter.** `microrag_lint.sh` is first-use signature surfacing only, not a syntax/semantic linter. A real linter is a separate project.
- **Cross-session memory of injections.** Dedup ledger is per-session. Re-injecting in a new session is acceptable (cheap re-grounding).
- **Custom embedding training for AILANG.** Reuse `nomic-embed-text` (Ollama, free, local). A bespoke model is over-investment until corpus volume justifies it.
- **Continuous prompt re-indexing.** Reindex cadence is *per-release*, not per-commit. Between releases, dev-loop changes to `prompts/` require manual `make brain-index-syntax` if you want them surfaced — but the canonical fresh state is always "what shipped at the last release."
- **Auto-discovery of new knowledge bases.** Glob router is config-driven; new KBs require a YAML edit.
- **Replacing the `use-ailang` skill.** The skill remains for explicit invocation; μRAG is for ambient nudging. They complement.
- **Native plugin in every harness.** v0.15.0 ships Claude Code hooks + MCP server. Aider plugin / Cursor extension / Continue config helpers are Future Work.
- **Eval-runner overhaul.** v0.15.0 adds the `--microrag` flag, not a redesign of how eval results are stored or visualized.

## Timeline

**Week 1** (12 hours):
- Phase 1: Engine (Go CLI) (6h)
- Phase 2: Builtin lint engine subcommand (3h)
- Phase 3: Syntax corpus bootstrap (3h)

**Week 2** (10 hours):
- Phase 4: Claude Code frontend hooks (2h)
- Phase 5: MCP server frontend (3h)
- Phase 6: Eval-suite integration (2h)
- Phase 7 start: Unit tests (3h)

**Week 2–3** (8 hours):
- Phase 7 finish: integration + MCP smoke test (3h)
- Dogfooding observation in AILANG repo (1h passive)
- Eval A/B run + analysis (2h)
- Documentation (2h)

**Total: ~32 hours across 2.5 weeks** (matches "5 days" estimate at 6h/day)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| AI prompt-cache thrash from per-edit injection | High | Search-result cache TTL (240s) keeps injection bytes stable; token-window dedup suppresses repeats. Verified via telemetry hit rate metric. |
| Engine CLI process-spawn cost makes hook too slow | Medium | Measured at ~10ms per spawn; well under 200ms target. If problematic, add long-lived daemon mode (`ailang micro-rag daemon`) — deferred to Future Work. |
| Embedding latency (cold call) blocks Edit | Medium | 2s hook timeout + embed cache (24h). Cold path still completes, but worst-case bounded. Falls back to FTS if Ollama down. |
| Token-window default too aggressive (over-suppresses) | Medium | Per-namespace tunable; relevance bypass for high-confidence matches. Calibration revisited after 1-week dogfooding. |
| Token-window default too loose (under-suppresses, wastes tokens) | Medium | Session budget (5000 tokens) is hard cap with warning. Telemetry shows total injection tokens per session for tuning. |
| Eval A/B comparison is statistically weak (small benchmark count) | Medium | Use `--microrag=auto` to run *every* eval both ways, accumulating sample size. Document min-N for significance. |
| MCP harness integration breaks on harness updates | Medium | Smoke test against current Cursor/Continue versions; document tested versions; MCP protocol is standardized. |
| Glob router config drift between users | Low | Ship sensible default in repo; document in `docs/docs/guides/microrag.md`. CLI bootstrap available via `ailang micro-rag init`. |
| Corrupt `injections.jsonl` from concurrent writes | Low | Sessions are single-writer (one Claude process per session). Defensive: parse-error → truncate-and-continue. |
| Bootstrap indexer breaks on prompt version bump | Low | Stable keys + idempotent re-run; indexer resolves active prompt dynamically (no pinned version). |
| **Stale corpus drifts behind the language** | **High** | Reindex tied to release workflow (`release-manager` runs `make brain-index-syntax-reset`); `post-release` verifies active version present in ≥5 chunks. Drift bounded by release cadence (typically <2 weeks). |
| Indexer resolves a non-existent "active" prompt (e.g., dev branch with no published version) | Low | Indexer fails loudly with explicit error pointing to `ailang prompt --list`; never falls back to a pinned default. |
| `μ` character doesn't render in some terminals | Low | Fallback marker `[uRAG]` documented as acceptable substitute. Engine accepts a `--marker-style=unicode|ascii` flag. |

## Related Documents

<!-- Auto-populated by Ollama neural search on "brain jit / micro-rag" -->

**Implemented (foundational):**
- [design_docs/implemented/v0_9_2/m-brain-sprint-plan.md](../../implemented/v0_9_2/m-brain-sprint-plan.md) (0.29) — M-BRAIN foundation: SQLite backend + CLI + two-tier architecture
- [design_docs/implemented/v0_10_0/m-brain-context-sprint-plan.md](../../implemented/v0_10_0/m-brain-context-sprint-plan.md) (0.30) — M-BRAIN-CONTEXT: PreToolUse(Read) hook with cooldown/budget/relevance threshold (this doc generalizes to Edit/Write + token-window dedup + harness portability)

**Implemented (related):**
- [design_docs/implemented/v0_11_1/m-wasm-trace-sprint-plan.md](../../implemented/v0_11_1/m-wasm-trace-sprint-plan.md) — telemetry plumbing (reused for μRAG metrics)
- [design_docs/implemented/v0_5_10/m-string-conversion.md](../../implemented/v0_5_10/m-string-conversion.md) (0.30) — string-related history (relevant to `++` migration nudges)

**Planned (potential overlap to monitor):**
- [design_docs/planned/v0_13_0/m-coord-thinking-levels.md](../v0_13_0/m-coord-thinking-levels.md) (0.29) — coordinator-side context shaping
- [design_docs/planned/v0_13_0/m-eval-lazy-pipeline.md](../v0_13_0/m-eval-lazy-pipeline.md) (0.25) — eval changes that may inform success-metric definitions

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [M-BRAIN-CONTEXT (v0.10.0)](../../implemented/v0_10_0/m-brain-context-sprint-plan.md) — direct predecessor
- [M-EVAL-GAP-ANALYSIS (v0.7.4)](../../implemented/v0_7_4/m-eval-gap-analysis.md) — catalog of common agent mistakes (calibrates which nudges matter most)
- Anthropic prompt caching docs — 5-minute cache TTL informs the 240s search-cache TTL
- [Model Context Protocol](https://modelcontextprotocol.io) — standard for harness-portable tool exposure
- User plan: `~/.claude/plans/yes-i-m-thinking-a-declarative-puddle.md` (private; design originated here)

## Future Work

- **Cross-session learning loop.** When a μRAG injection precedes a successful edit (no compile error within N turns), boost that snippet's score. When an injection precedes a failed edit, demote. Closes the "system gets smarter with use" loop with measurable signal.
- **Long-lived daemon mode.** `ailang micro-rag daemon` keeps the engine warm to drop process-spawn cost from ~10ms to ~1ms; useful if telemetry shows spawn cost matters.
- **Mid-cycle reindex hook.** Watch `prompts/` and `docs/LIMITATIONS.md` for changes between releases and trigger a partial reindex without waiting for the next release. Lower priority — release-tied reindex is sufficient at current release cadence.
- **Multi-KB ranking.** When multiple routes match (e.g., a `.ail` file in a Go-heavy project), surface results from both with provenance markers.
- **Native harness plugins.** Aider extension, Cursor `.cursorrules` integration, Continue config templates — beyond the MCP fallback.
- **Generalized to non-AILANG projects.** Same engine + config in any repo with any KB; AILANG is the proof-of-concept, not the only use case. Marketing position: "μRAG: bring your own corpus, your own harness."

---

**Document created**: 2026-04-22
**Last updated**: 2026-04-22

DESIGN_DOC_PATH: design_docs/planned/v0_15_0/m-brain-microrag.md
