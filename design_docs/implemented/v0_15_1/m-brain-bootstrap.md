# M-BRAIN-BOOTSTRAP: `ailang micro-rag bootstrap` for fresh installs

**Status**: Implemented
**Target**: v0.15.1
**Priority**: P0 (High — μRAG hooks ship in v0.15.0 but are no-ops on fresh installs until this lands)
**Estimated**: 1 day (~6h implementation + 2h testing + 1h docs/rollout)
**Dependencies**: M-BRAIN (v0.9.2 ✅), M-BRAIN-VECTORS (v0.9.4 ✅), M-BRAIN-CONTEXT (v0.10.0 ✅), M-BRAIN-MICRORAG (v0.15.0 ✅)
**Milestone ID**: M-BRAIN-BOOTSTRAP

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Same embedded prompt + same builtin registry = byte-identical brain corpus. Reset path is deterministic (drop → repopulate). |
| A2: Replayability | +1 | All writes logged via `BrainStore.Put` with stable keys (`syntax-<version>-<slug>`, `builtin-<name>`); rerunning is a no-op (upsert on key). |
| A3: Effect Legibility | +1 | Subcommand prints summary (`indexed N syntax chunks, M builtins`); `--json` mode emits machine-readable result envelope for install scripts. |
| A4: Explicit Authority | +1 | Reuses existing `BrainStore` capability — no new ambient access. Scope (`user` / `project`) is explicit; default `user` matches the install-time mental model. |
| A5: Bounded Verification | 0 | No verification surface change. |
| A6: Safe Concurrency | 0 | Single-writer, run-to-completion command; no concurrency surface. |
| A7: Machines First | +2 | Sole purpose: make μRAG retrieval surface populated for the next AI session. `--json` mode designed for machine consumption by `install.sh`. |
| A8: Minimal Syntax | 0 | No language syntax change. |
| A9: Cost Visibility | +1 | Reports counts per namespace; `--no-embed` lets users opt out of Ollama-dependent embedding cost. |
| A10: Composability | +2 | Reuses `prompt.LoadPrompt`, `builtins.AllSpecs`, `BrainStore.Put`, `messaging.NewEmbedderFromConfig` — no new abstractions. Replaces an external shell script with a binary-internal Go function. |
| A11: Structured Failure | +1 | Loud failures: missing prompt resolves to non-zero exit. Soft failures: Ollama down falls back to SimHash-only with a single stderr warning, matching `cache put` behaviour. |
| A12: System Boundary | +1 | Boundary is explicit: embedded FS → chunker → embedder → BrainStore. Each layer testable in isolation; no disk writes outside the brain DB path. |

**Net Score: +11** → **Decision: Strong Accept**

### Hard Violation Check

- [x] A1 (Determinism): Stable keys + `INSERT … ON CONFLICT(key) DO UPDATE`. Re-runs are byte-equivalent given the same binary.
- [x] A3 (Effects): All writes go through `BrainStore.Put`; counts are reported back to the caller.
- [x] A4 (Authority): No new caps; only writes the brain DB at the user-visible path.
- [x] A7 (Machines First): The whole feature exists so machine consumers get a populated retrieval surface immediately after install.

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

**Score +11** → strong accept.

## Problem Statement

μRAG (M-BRAIN-MICRORAG, v0.15.0) ships hooks that fire on `Edit`/`Write`/`Read` in any project that installs the `ailang_bootstrap` plugin. But the hooks are **useless until the brain DB has a populated `ailang-syntax` / `ailang-builtins` corpus**. Today that population happens **only** via `tools/index_ailang_syntax.sh` — a shell script that depends on the source repo (`prompts/v0.X.Y.md`, `docs/LIMITATIONS.md`, `examples/runnable/*.ail`, `awk`, `python3`, `find`).

**Current State (post-v0.15.0):**
- μRAG hooks installed by the `ailang_bootstrap` plugin — registered, firing on every Edit/Write/Read.
- Brain DB at `~/.ailang/state/brain.db` is **empty** (no `ailang-syntax`, no `ailang-builtins` namespace) on a fresh install.
- The shell indexer at `tools/index_ailang_syntax.sh` requires the source repo on disk + `awk` + `python3`.
- Result: every hook call returns "no relevant snippets"; injection rate = 0%; the installed plugin appears silently broken.

**Quantifiable gap:**
- v0.15.0 dogfooding metric (target): ≥80% of Edit/Write events on `.ail` files trigger a relevance check.
- Fresh-install reality: 0% of events return content because the corpus is empty.
- Estimated affected user base: every `gemini extensions install …` and `/plugin install ailang@…` install on a host without the AILANG source repo.

**For users installing via `ailang_bootstrap`** (Claude Code plugin or `gemini extensions install …`), this is fatal:
- No source repo on disk
- No guarantee of `awk` / `python3` (Windows)
- Hooks are wired and firing but every call returns "no relevant snippets"

**What's already in the binary:**
- `cmd/ailang/main.go:19` declares `//go:embed all:prompts` — the active prompt corpus AND `versions.json` are bundled.
- The builtin registry is also compiled in (`builtins.AllSpecs()` returns ~280 specs from any binary).

A fresh install therefore has *most* of what we need; we just lack a CLI verb to consume it.

**Impact:**
- Plugin/extension users get the hooks but no benefit from them.
- The dogfooding metrics for v0.15.0 cannot be honestly evaluated outside the source repo.
- "Fresh install" demos to evaluators / new users silently degrade — first impression is broken.
- Locks the brain corpus to source-repo developers, contradicting the harness-agnostic engine design from M-BRAIN-MICRORAG.

## Goals

**Primary Goal:** A fresh `gemini extensions install …` or `/plugin install ailang@…` ends with a populated brain corpus and working μRAG injection — using only resources bundled in the `ailang` binary.

**Success Metrics:**
1. **Coverage:** After `ailang micro-rag bootstrap --scope user`, `ailang cache stats | grep ailang-syntax` reports ≥40 frames; `ailang-builtins` reports ≥250 frames.
2. **Time-to-first-injection:** From a clean `~/.ailang/`, install → bootstrap → first hook call → injected content ≤ 5s end-to-end on a warm Ollama, ≤ 2s with `--no-embed`.
3. **Cross-platform:** Bootstrap completes on macOS / Linux / Windows without `awk` / `python3` / `find` (no shell dependencies).
4. **Idempotency:** Running bootstrap twice in a row produces identical row counts (upsert on stable keys, no duplicates).
5. **Graceful degradation:** With Ollama down, bootstrap completes with a single stderr warning and frames written without embeddings; subsequent FTS+SimHash queries still return content.
6. **Auto-install:** The `ailang_bootstrap` plugin's `install.sh` runs `ailang micro-rag init && ailang micro-rag bootstrap --scope user` after the binary lands; user does nothing manually.

## High-Impact Decisions + Design Freeze

These decisions are locked before implementation begins to prevent scope drift.

### Decision 1: Default scope is `--scope user`

**Locked.** Writes to `~/.ailang/state/brain.db` (cross-project) by default.

**Rationale:** The existing shell indexer defaults to `project`, which is wrong for the bootstrap flow — fresh-install users don't have a project DB and end up with nothing. User-scope means the corpus survives across every project on the host, matching how the hooks work (they read from both scopes via `BrainStore`'s union semantics). Override available via `--scope project` for repo developers who want isolation.

### Decision 2: Binary-only — no disk reads outside the brain DB

**Locked.** Bootstrap consumes only resources bundled via `//go:embed all:prompts`. It does NOT read `examples/`, `docs/LIMITATIONS.md`, or any path under the source repo.

**Rationale:** Fresh-install users don't have the source repo. Mixing disk reads with embedded reads creates a "works in dev, fails in prod" trap. The shell indexer remains the "full-fat" tool for repo developers (it can index examples + LIMITATIONS via disk reads); bootstrap is a strict subset.

### Decision 3: Two namespaces in scope, examples deferred

**Locked.** This sprint populates `ailang-syntax` and `ailang-builtins` only. `ailang-examples` is explicitly out of scope and deferred to a follow-up sprint.

**Rationale:** The `examples/` directory is *not* `go:embed`'d. Embedding it (~50 LOC `//go:embed all:examples/runnable/*.ail` plus binary size analysis) is a separate concern that deserves its own discussion. Skipping cleanly with a "0 example pointers indexed (binary-only mode)" note is better than half-doing it.

### Decision 4: Reuse the existing engine — no new packages

**Locked.** All work lives in `cmd/ailang/microrag.go` (currently ~210 LOC, target ~360 LOC). No new packages, no new abstractions. Calls into existing `prompt.LoadPrompt`, `builtins.AllSpecs`, `BrainStore.Put`, `messaging.NewEmbedderFromConfig`.

**Rationale:** The engine already exists; the gap is a thin CLI verb. Adding a package would invent a layer for a single caller.

### Decision 5: `--no-embed` flag for envs without Ollama

**Locked.** Default behaviour computes embeddings if an embedder is available, falls back to SimHash silently if not. `--no-embed` skips even attempting the embedder, useful for CI / Windows / minimal envs.

**Rationale:** Symmetric with how `cache put` already behaves. No surprise — users explicitly opt out of the network call.

## Solution Design

### Subcommand surface

```bash
ailang micro-rag bootstrap                    # default: --scope user, --embed (graceful)
ailang micro-rag bootstrap --reset            # drop the 2 namespaces and rebuild
ailang micro-rag bootstrap --scope project    # override (repo dev workflow)
ailang micro-rag bootstrap --no-embed         # skip Ollama; SimHash + FTS only
ailang micro-rag bootstrap --json             # machine-readable result for install scripts
```

Wires into existing switch at `cmd/ailang/microrag.go:24-37` alongside `init`, `context`, `lint-builtin`.

### Architecture

```
                     ailang micro-rag bootstrap
                              │
                              ▼
            ┌─────────────────────────────────────┐
            │  runMicroragBootstrap(args []string) │
            │  - parse flags (scope/reset/embed)  │
            │  - resolve active prompt version    │
            │  - open BrainStore (with embedder?) │
            │  - optional: drop namespaces        │
            └────────────┬───────────────┬────────┘
                         │               │
                         ▼               ▼
        ┌─────────────────────┐  ┌──────────────────────┐
        │ chunkPromptByH2     │  │ bootstrapBuiltins    │
        │ - LoadPrompt("")    │  │ - builtins.AllSpecs()│
        │ - scan ## headings  │  │ - format signature   │
        │ - emit BrainFrames  │  │ - emit BrainFrames   │
        └─────────┬───────────┘  └──────────┬───────────┘
                  │                         │
                  └────────────┬────────────┘
                               ▼
                  ┌────────────────────────────┐
                  │  BrainStore.Put(frame, sc) │
                  │  (auto-embeds if embedder) │
                  └────────────┬───────────────┘
                               ▼
                  ┌────────────────────────────┐
                  │  ~/.ailang/state/brain.db  │
                  │  ailang-syntax    (~50)    │
                  │  ailang-builtins  (~280)   │
                  └────────────────────────────┘
```

### Implementation map (what reuses what)

| Step | Source | Reused API |
|------|--------|-----------|
| Resolve active prompt version | `versions.json` in embedded FS | `prompt.GetActiveVersion()` at [internal/prompt/loader.go:149](internal/prompt/loader.go#L149) |
| Get prompt content | embedded `prompts/v0.X.Y.md` | `prompt.LoadPrompt("")` at [internal/prompt/loader.go:41](internal/prompt/loader.go#L41) |
| Chunk prompt by `## ` headings | new Go helper in `cmd/ailang/microrag.go` | port of the awk in [tools/index_ailang_syntax.sh:80-108](tools/index_ailang_syntax.sh#L80) — ~40 LOC of `bufio.Scanner` |
| Enumerate builtins | `internal/builtins` package | `builtins.AllSpecs()` returns `map[string]*BuiltinSpec` (already used at [cmd/ailang/doctor.go:130](cmd/ailang/doctor.go#L130)) |
| Per-builtin signature/effect/desc | `BuiltinSpec` struct | direct field access (`Module`, `Name`, `IsPure`, `Effect`, `Metadata.Description`) + `formatBuiltinSignature` from [cmd/ailang/doctor.go:232](cmd/ailang/doctor.go#L232) |
| Resolve user/project DB path | existing `cmd/ailang/cache.go` helpers | `getUserBrainPath()`, `getProjectBrainPath()` |
| Open BrainStore | existing helper | `openBrainStoreWithOpts(effects.WithEmbedder(...))` |
| Write to brain DB | `internal/effects/brain.go` | `BrainStore.Put(frame, scope)` at [internal/effects/brain.go:309](internal/effects/brain.go#L309) |
| Construct embedder | `messaging.NewEmbedderFromConfig` | mirrors [cmd/ailang/cache.go:82-90](cmd/ailang/cache.go#L82) — gracefully nil if Ollama down |
| Drop namespaces (`--reset`) | `internal/effects/sharedmem_sqlite.go` | `(c *SQLiteSharedCache).DeleteNamespace(namespace)` at line 519 |

### Frame schema

Each chunk written via `BrainStore.Put`:

```go
effects.BrainFrame{
    Key:       "syntax-v0.4.10-pattern-matching",   // stable: same input → same key
    Namespace: "ailang-syntax",                      // routed by glob in microrag.yaml
    Content:   "[ns:ailang-syntax] [version:v0.4.10] [section:Pattern Matching]\n...",
    Source:    "bootstrap-v0.15.1",                  // distinguishes from manual `cache put`
    // Embedding/EmbeddingDim/EmbedModel auto-populated by PutFrame if embedder present
}
```

Builtin frames mirror the pattern:

```go
effects.BrainFrame{
    Key:       "builtin-_net_httpRequest",
    Namespace: "ailang-builtins",
    Content:   "[ns:ailang-builtins] [version:v0.4.10] [module:std/net] [effect:Net]\n_net_httpRequest: ...\nMake an HTTP request with custom headers and body",
    Source:    "bootstrap-v0.15.1",
}
```

The `[ns:…] [version:…] [section:…]` header tags match the format produced by `tools/index_ailang_syntax.sh` so retrieval-side filters are unchanged.

### What this sprint does NOT do

- **Examples corpus (`ailang-examples`)**: The `examples/` directory is *not* `go:embed`'d ([cmd/ailang/examples.go:669](cmd/ailang/examples.go#L669) walks up from the executable looking for the dir on disk). Skipping in this sprint is intentional: the script will note "0 example pointers indexed (binary-only mode)" and exit clean. Embedding examples is a separate concern (~50 LOC `//go:embed all:examples/runnable/*.ail`, possible binary size impact, deserves its own discussion).
- **`docs/LIMITATIONS.md`**: Same situation. ~20 LOC to embed if we want it, or skip cleanly. Deferred — the prompt itself already covers most of what LIMITATIONS does.
- **No changes to the existing shell script.** `tools/index_ailang_syntax.sh` continues to be the "full-fat" indexer for repo developers (it can index examples + LIMITATIONS via disk reads, which the binary can't). Bootstrap is a strict subset.
- **No new dedup logic.** The engine's existing token-window dedup applies at retrieval time; bootstrap just populates the corpus.

### Files to Create / Modify

**Modify:**

- [cmd/ailang/microrag.go](cmd/ailang/microrag.go) — add `case "bootstrap":` to switch (line ~30); add `runMicroragBootstrap(args []string)` (~150 LOC) + helpers `chunkPromptByH2`, `bootstrapSyntax`, `bootstrapBuiltins`, `bootstrapResetNamespaces`. Update help text at line 197-210.
- [/Users/mark/dev/sunholo/ailang_bootstrap/skills/ailang/scripts/install.sh](/Users/mark/dev/sunholo/ailang_bootstrap/skills/ailang/scripts/install.sh) — append after binary install:
  ```bash
  # Populate μRAG brain (syntax + builtins) — graceful no-op on failure
  if command -v ailang >/dev/null 2>&1; then
      ailang micro-rag init >/dev/null 2>&1 || true
      ailang micro-rag bootstrap --scope user 2>&1 | tail -5 || true
  fi
  ```
- [/Users/mark/dev/sunholo/ailang_bootstrap/README.md](/Users/mark/dev/sunholo/ailang_bootstrap/README.md) — note under Hooks table that the install script auto-bootstraps the brain corpus.
- [docs/docs/guides/microrag.md](docs/docs/guides/microrag.md) — document the new subcommand alongside `init`.
- [changelogs/v0.10-current.md](changelogs/v0.10-current.md) — add entry under v0.15.1.

**Create:**

- [cmd/ailang/microrag_test.go](cmd/ailang/microrag_test.go) — unit tests for the chunker and bootstrap helpers.

**No new files in `internal/`.** All work fits in existing `cmd/ailang/microrag.go` (currently ~210 LOC, target ~360 LOC — well under the 800-line guideline).

**Reuse as-is** (no changes):
- `internal/prompt/loader.go` (`LoadPrompt` + `GetActiveVersion` are already perfect)
- `internal/builtins/spec.go` (`AllSpecs` is already perfect)
- `internal/effects/brain.go` (`BrainStore.Put` is already perfect)
- `internal/effects/sharedmem_sqlite.go` (`DeleteNamespace` is already perfect)
- All hook scripts (`microrag_context.sh`, `microrag_lint.sh`)
- All engine code in `internal/microrag/`

## Implementation Milestones

### M1 — Subcommand wiring + flag parsing (~30 LOC)

- Add `case "bootstrap":` to switch at `cmd/ailang/microrag.go:24-37`.
- Stub `runMicroragBootstrap(args []string)` parsing `--scope`, `--reset`, `--no-embed`, `--json`, `--config`.
- Update `printMicroragHelp` block at line 197-210.

**Acceptance:** `ailang micro-rag bootstrap --help` prints the documented usage; `ailang micro-rag bootstrap` runs (even if no-op) without panic.

### M2 — Prompt chunking + indexing (~80 LOC)

- Implement `chunkPromptByH2(content string) []promptChunk` — `bufio.Scanner`, accumulate lines per `## `, slug-ify section name. Skip bodies < 200 bytes.
- Implement `bootstrapSyntax(store *BrainStore, scope BrainScope, version string) (count int, err error)` — calls `prompt.LoadPrompt("")`, chunks, writes via `BrainStore.Put`.

**Acceptance:** After `bootstrap --scope user`, `ailang cache list --namespace ailang-syntax --scope user` shows ≥40 keys.

### M3 — Builtins indexing (~50 LOC)

- Implement `bootstrapBuiltins(store *BrainStore, scope BrainScope, version string) (count int, err error)` — iterates `builtins.AllSpecs()`, formats `[ns:…] [version:…] [module:…] [effect:…]\n<signature>\n<description>`, writes via `BrainStore.Put`. Effect label = `"pure"` for `IsPure`, otherwise `spec.Effect`.

**Acceptance:** After `bootstrap`, `ailang cache list --namespace ailang-builtins` shows ≥250 keys; spot-check one frame contains the formatted signature.

### M4 — `--reset` + `--no-embed` + `--json` flags (~30 LOC)

- `--reset`: call `store.User.DeleteNamespace("ailang-syntax")` and `("ailang-builtins")` (or `.Project` per scope) before indexing.
- `--no-embed`: skip `createEmbedder()` entirely; pass no-opt to `openBrainStoreWithOpts`.
- `--json`: emit `{active_version, syntax_indexed, builtins_indexed, scope, embed_used, reset, took_ms}` instead of human summary.

**Acceptance:** Re-running `bootstrap --reset` produces identical counts to the first run; `bootstrap --json` emits parseable JSON; `bootstrap --no-embed` writes frames with `embedding_dim = 0`.

### M5 — Install script integration + docs (~10 LOC bash + docs)

- Append the auto-bootstrap stanza to `ailang_bootstrap/skills/ailang/scripts/install.sh`.
- Add a "Bootstrap" section to `docs/docs/guides/microrag.md` documenting the new subcommand and the auto-install flow.
- Add a v0.15.1 entry to the active changelog.
- Update `ailang_bootstrap/README.md` with a one-liner about the auto-bootstrap.

**Acceptance:** Simulated fresh install (`rm -rf ~/.ailang/`, run `install.sh`, check `cache stats`) shows populated namespaces.

## Testing Strategy

### Unit tests (`cmd/ailang/microrag_test.go`, new file)

1. **`TestChunkPromptByH2`**: Given a 3-section markdown string, returns 3 chunks with correct `key|section|version` triples and bodies. Verify slug normalisation (`"Pattern Matching"` → `pattern-matching`). Verify `<200 byte` skip.
2. **`TestBootstrapEmitsBuiltins`**: With a fake `BrainStore` (in-memory SQLite via `NewSQLiteSharedCache(":memory:")`), `bootstrapBuiltins` writes one frame per `builtins.AllSpecs()` entry. Assert frame count == `len(specs)` and namespace == `"ailang-builtins"`.
3. **`TestBootstrapGracefulNoOllama`**: Open store without an embedder; bootstrap completes successfully, frames written without `Embedding` field populated. No panic, no `os.Exit`.

### Integration smoke test (manual, run from a clean `~/.ailang/`)

```bash
rm -rf ~/.ailang/state/brain.db
ailang micro-rag init
ailang micro-rag bootstrap --scope user
ailang cache stats | grep -E 'ailang-(syntax|builtins)'
# Expect: ailang-syntax ~50 frames, ailang-builtins ~280 frames
```

### End-to-end with hooks (validates the actual fresh-install scenario)

```bash
# Simulate fresh install: blow away DB
rm -rf ~/.ailang/state/brain.db ~/.ailang/state/sessions/

# Run bootstrap
ailang micro-rag bootstrap --scope user

# Now exercise the hook the way Claude Code would
echo '{"tool_name":"Edit","tool_input":{"file_path":"/tmp/foo.ail","new_string":"let s = \"a\" ++ \"b\""}}' \
  | ~/.ailang/hooks/microrag_context.sh | jq -r '.hookSpecificOutput.additionalContext'
# Expect: ++ string-vs-list warning text from the syntax corpus
```

### Cross-platform (`--no-embed` mode for envs without Ollama)

```bash
ailang micro-rag bootstrap --no-embed --scope user
# Expect: completes, frames written with embedding_dim=0, FTS+SimHash search still works
```

### `ailang_bootstrap` install flow (end-to-end)

```bash
# Simulate plugin install in a clean env
rm -rf ~/.ailang/
bash /Users/mark/dev/sunholo/ailang_bootstrap/skills/ailang/scripts/install.sh
ls ~/.ailang/microrag.yaml                          # should exist
ailang cache stats | grep ailang-syntax             # should show ~50 frames
```

## Rollout

- Ship in **v0.15.1** (patch on the M-BRAIN-MICRORAG release that introduced the hooks).
- `ailang_bootstrap` plugin needs a sympathetic version bump (e.g., 0.10.0 → 0.11.0) so the new install logic ships to plugin users.
- No DB migration; bootstrap is purely additive (new keys in existing namespaces, or `--reset` to clear first).
- Backward-compatible: existing repo developers can keep using `tools/index_ailang_syntax.sh` (which has examples + LIMITATIONS support); fresh-install users get the binary-only subset.
- Doc / changelog entry calls out the binary-only / examples-deferred constraint so users with custom workflows aren't surprised.

## Estimated effort

~250 LOC Go (subcommand + helpers + tests) + ~10 LOC bash (install.sh tail) + docs. ~1 day.

## Related Documents

- [M-BRAIN-MICRORAG (implemented v0.15.0)](../../implemented/v0_15_0/m-brain-microrag.md) — the engine + hooks this populates
- [tools/index_ailang_syntax.sh](../../../tools/index_ailang_syntax.sh) — the "full-fat" repo-developer indexer that this is a binary-only subset of
- [docs/docs/guides/microrag.md](../../../docs/docs/guides/microrag.md) — user-facing guide updated by this sprint
