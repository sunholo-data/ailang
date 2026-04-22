# Sprint Plan: M-BRAIN-MICRORAG

## Summary

Implement the **micro-RAG (μRAG)** just-in-time knowledge injection system: a harness-agnostic Go engine (`ailang micro-rag`) plus Claude Code hook frontend plus MCP server frontend, with token-window dedup, prompt-cache-aligned result caching, first-use builtin lint, glob-routed knowledge bases, eval-suite A/B toggle, and release-tied corpus reindexing.

**Duration:** 5 days (~26 hours implementation + 6 hours testing/dogfooding)
**Dependencies:** M-BRAIN (v0.9.2 ✅), M-BRAIN-VECTORS (v0.9.4 ✅), M-BRAIN-CONTEXT (v0.10.0 ✅)
**Risk Level:** Medium (touches multiple subsystems; new external surface via MCP)

## Current Status Analysis

### Completed Recently
- ✅ v0.14.1 release (cosign verification, codebase-stats fixes)
- ✅ M-PERF5 milestones M1–M3 (DocParse perf — bulk XML, str join, dtree investigation)
- ✅ M-BRAIN-CONTEXT (v0.10.0) — `brain_context.sh` PreToolUse(Read) hook
- ✅ Three-tier search (cosine + simhash + FTS) in `internal/effects/cache_ops.go`

### Velocity
- Recent (last 14d): mostly release work + dependency bumps; one perf sprint with ~310 LOC across 3 milestones in 1 day
- Reasonable target: 300–500 LOC/day for new Go code with tests; bash hooks faster
- This sprint: ~3000 LOC total (Go + bash + YAML + tests) across 5 days = 600 LOC/day average

### Remaining from Design Doc
All 7 phases — nothing implemented yet. The design doc (`m-brain-microrag.md`) is the source of truth; this sprint plan slices it into ordered, testable milestones.

### Key Pre-existing Surface
- `ailang cache search` exists but lacks `--json` flag (needed by engine).
- `ailang prompt` exists but lacks `--version-active` (needed by indexer).
- `ailang builtins show` exists but `--json` may be missing (needed by lint).
- `internal/effects/cache_ops.go` three-tier search reused as-is.

## Proposed Milestones

### Milestone 1: M1_ENGINE — Core engine CLI

**Goal:** Implement `ailang micro-rag context` CLI that does the full search → dedup → format pipeline end-to-end. This is the foundation; every other milestone depends on it.

**Estimated:** ~900 LOC implementation + ~400 LOC tests = ~1300 LOC
**Duration:** 1.5 days

**Tasks:**
- Day 1 AM: Scaffold `cmd/ailang/microrag.go` (~150 LOC) with subcommand dispatch + env-toggle early-exit (`AILANG_MICRORAG_ENABLED`, `AILANG_MICRORAG_ROUTES`, `AILANG_MICRORAG_DRYRUN`).
- Day 1 AM: Add `internal/microrag/config.go` (~120 LOC) — YAML loader for `~/.ailang/microrag.yaml`.
- Day 1 PM: Add `internal/microrag/engine.go` (~600 LOC):
  - Glob router using `doublestar` (already a transitive dep) or `path.Match`.
  - Search-result cache (filesystem JSON, 240s TTL, key = sha256(file+ns+content_prefix+limit)).
  - Token-window dedup (read tail of `injections.jsonl`, sum tokens since last `snippet_id`).
  - Per-namespace dedup windows (15K/30K/80K/40K from config).
  - Relevance-score bypass.
  - Format injection (≤150 tokens, stable ASCII borders, marker `🧠 μRAG [ns]` with `--marker-style=ascii` fallback).
  - Append to `injections.jsonl` after each injection (with `microrag_state` tag).
  - Output JSON contract: `{injection_text, snippet_id, tokens, ns, microrag_state}` or `null`.
- Day 2 AM: Add `--json` flag to `ailang cache search` if missing (~30 LOC); add `--version-active` to `ailang prompt` (~15 LOC).
- Day 2 AM: Unit tests for engine (`internal/microrag/*_test.go`):
  - Env-toggle early-exit (no search call when disabled).
  - Allowlist filtering.
  - Dryrun mode logs but no injection.
  - Glob router (each route + `kb: skip`).
  - Dedup ledger arithmetic.
  - Cache TTL (240s window).
  - Relevance bypass.
  - Session budget enforcement.

**Acceptance Criteria:**
- [ ] `ailang micro-rag context --tool=Edit --file=foo.ail --content=...` returns valid JSON or `null`.
- [ ] `AILANG_MICRORAG_ENABLED=0` produces zero search calls (verified via mock).
- [ ] `AILANG_MICRORAG_ROUTES=ailang-syntax` filters routes correctly.
- [ ] `AILANG_MICRORAG_DRYRUN=1` writes to ledger but emits no injection.
- [ ] Same query within 240s returns identical bytes (cache hit).
- [ ] Token-window dedup suppresses `snippet_id` within configured window unless score ≥ bypass.
- [ ] All tests pass; `make lint` clean.

**Risks:**
- Engine spawn cost (~10ms) may stack on hook bursts — Mitigation: telemetry to confirm; daemon mode in Future Work.
- `doublestar` not yet a direct dep — Mitigation: vendor it explicitly or use stdlib `path.Match` with restrictions.

---

### Milestone 2: M2_LINT — Builtin first-use lint subcommand

**Goal:** Add `ailang micro-rag lint-builtin` subcommand that detects builtin identifiers in newly-written code and surfaces signatures on first use per session.

**Estimated:** ~150 LOC implementation + ~100 LOC tests = ~250 LOC
**Duration:** 0.5 days

**Tasks:**
- Day 2 PM: Add `internal/microrag/lint.go` (~150 LOC):
  - Identifier extraction regex (`\b[a-z_][A-Za-z0-9_]*\s*\(`).
  - Per-session `builtins_seen.txt` tracker.
  - Shell-out to `ailang builtins show <name> --json` (add `--json` if missing, ~50 LOC).
  - Cap at 2 nudges per event.
  - Output: same JSON contract as `context` subcommand.
- Day 2 PM: Unit tests for lint.

**Acceptance Criteria:**
- [ ] `ailang micro-rag lint-builtin --code='let x = concat_String(["a"])'` returns nudge for `concat_String`.
- [ ] Second call same input returns `null` (already in `builtins_seen.txt`).
- [ ] Non-builtin identifiers silently skipped (no error on `myLocalVar(`).
- [ ] Output capped at 2 nudges max.
- [ ] All tests pass.

**Risks:**
- Regex false positives on local vars — Mitigation: silent skip on `builtins show` non-zero exit; acceptable.

---

### Milestone 3: M3_INDEXER — Syntax corpus bootstrap + release-tied reindex

**Goal:** One-time + release-time corpus indexer for the `ailang-syntax` and `ailang-builtins` namespaces, using *active* prompt resolution (no version pinning).

**Estimated:** ~200 LOC bash + ~50 LOC Makefile/skill changes = ~250 LOC
**Duration:** 0.5 days

**Tasks:**
- Day 3 AM: Write `tools/index_ailang_syntax.sh` (~150 LOC):
  - Resolve active prompt: `ailang prompt --version-active`.
  - Chunk active prompt by h2/h3 headers → `ailang-syntax` namespace.
  - Chunk `docs/LIMITATIONS.md` similarly.
  - Index each builtin from `ailang builtins list --json` → `ailang-builtins` namespace.
  - Index `examples/runnable/*.ail` headers as `ailang-examples` pointers.
  - Stable keys for idempotent re-run; record resolved version in chunk metadata.
- Day 3 AM: Add Makefile targets (~20 LOC):
  - `make brain-index-syntax` — idempotent additive run.
  - `make brain-index-syntax-reset` — drops namespaces and rebuilds (used by releases).
- Day 3 AM: Update `release-manager` skill (~30 LOC delta in `.claude/skills/release-manager/SKILL.md` + script):
  - Add post-tag step: `make brain-index-syntax-reset`.
- Day 3 AM: Update `post-release` skill (~30 LOC delta):
  - Spot-check ≥5 chunks reference active prompt version.
  - Append one-line audit to release notes.
- Day 3 AM: Run indexer manually; verify ≥250 chunks indexed.

**Acceptance Criteria:**
- [ ] `make brain-index-syntax` populates `ailang-syntax` namespace with ≥50 chunks from active prompt.
- [ ] `ailang-builtins` namespace has ≥250 chunks (one per builtin).
- [ ] Chunk metadata records resolved prompt version (no hardcoded version anywhere in indexer or chunks).
- [ ] `make brain-index-syntax-reset` drops + rebuilds without error.
- [ ] `release-manager` skill calls reset on release; `post-release` verifies.

**Risks:**
- Active prompt resolution fails on dev branch with no published version — Mitigation: indexer fails loudly with explicit error pointing to `ailang prompt --list`.

---

### Milestone 4: M4_HOOKS — Claude Code frontend hooks

**Goal:** Thin bash shims that adapt Claude Code's hook event JSON to the engine CLI; register both hooks in project `.claude/settings.json`.

**Estimated:** ~100 LOC bash + ~50 LOC YAML/JSON config = ~150 LOC
**Duration:** 0.5 days

**Tasks:**
- Day 3 PM: Write `~/.ailang/hooks/microrag_context.sh` (~50 LOC):
  - Read tool-call JSON from stdin.
  - Extract `tool_name`, `tool_input.file_path`, content.
  - Shell out: `ailang micro-rag context ...`.
  - Wrap result in Claude Code `additionalContext` JSON envelope.
  - `exit 0` on any error (graceful degradation).
- Day 3 PM: Write `~/.ailang/hooks/microrag_lint.sh` (~50 LOC):
  - Same pattern, wraps `ailang micro-rag lint-builtin`.
- Day 3 PM: Update `/Users/mark/dev/sunholo/ailang/.claude/settings.json`:
  - Add PreToolUse(Edit|Write|Read) entry for `microrag_context.sh`.
  - Add PostToolUse(Edit|Write) entry for `microrag_lint.sh`.
  - Replace existing `brain_context.sh` PreToolUse(Read) entry (keep `brain_session.sh` and `brain_resolution.sh` unchanged).
- Day 3 PM: Ship `~/.ailang/microrag.yaml` template with documented routes.
- Day 3 PM: Smoke test in fresh AILANG repo session — fire each hook once, verify injection appears.

**Acceptance Criteria:**
- [ ] `bash ~/.ailang/hooks/microrag_context.sh < fixtures/edit_event.json` returns valid `additionalContext` JSON.
- [ ] Hook returns `exit 0` even when engine errors (graceful degradation).
- [ ] Settings file registers both hooks; no conflict with existing brain hooks.
- [ ] Manual session smoke test: editing a `.ail` file with `++ "string"` shows breaking-changes nudge.

**Risks:**
- Hook timeout (2s) — Mitigation: engine cache-hit p95 well under 200ms; cold path bounded by 1.5s search.

---

### Milestone 5: M5_MCP — MCP server frontend

**Goal:** Stand up `ailang-microrag-mcp` MCP server that exposes the engine to Cursor / Continue / Cline / any MCP-compatible harness. This is the proof-of-portability deliverable.

**Estimated:** ~250 LOC Go + ~80 LOC docs = ~330 LOC
**Duration:** 0.75 days

**Tasks:**
- Day 4 AM: Write `cmd/ailang-microrag-mcp/main.go` (~250 LOC):
  - Use existing MCP Go library (check what's used elsewhere in repo first).
  - Tool: `microrag_context_for_file(file_path, tool_name, content?)`.
  - Tool: `microrag_lint_builtin(code)`.
  - Internally shell-out to `ailang micro-rag` engine.
  - Return same JSON payload as direct CLI invocation.
- Day 4 AM: Add `Makefile` target `microrag-mcp-build`.
- Day 4 PM: Write `docs/docs/guides/microrag.md` (~80 LOC) with:
  - μRAG concept explanation.
  - Claude Code install (auto via `.claude/settings.json`).
  - Cursor MCP config example.
  - Continue / Cline notes.
  - `microrag.yaml` reference.
  - Eval toggle reference.
- Day 4 PM: Smoke-test against Cursor (or Continue if Cursor not installed): install MCP server, edit `.ail` file, verify model can call tool and receives expected payload.

**Acceptance Criteria:**
- [ ] `ailang-microrag-mcp` binary builds via `make microrag-mcp-build`.
- [ ] MCP tool `microrag_context_for_file` returns same JSON as direct CLI call.
- [ ] Smoke test in at least one non-Claude-Code harness (Cursor preferred).
- [ ] `docs/docs/guides/microrag.md` covers Claude Code + Cursor install.

**Risks:**
- MCP library churn — Mitigation: pin to current MCP Go library version; document tested version in guide.
- No Cursor available locally — Mitigation: Continue or Cline as fallback; explicit message if neither tested.

---

### Milestone 6: M6_EVAL — Eval-suite A/B integration

**Goal:** Wire μRAG on/off into the eval runner so we can measure the actual mistake-reduction delta.

**Estimated:** ~120 LOC implementation + ~60 LOC tests = ~180 LOC
**Duration:** 0.5 days

**Tasks:**
- Day 4 PM: Add `--microrag=on|off|auto` flag to `cmd/ailang/eval_run.go` (~50 LOC):
  - `on`: subprocess env `AILANG_MICRORAG_ENABLED=1`.
  - `off`: `AILANG_MICRORAG_ENABLED=0`.
  - `auto`: run each benchmark twice (sequentially for v1; parallel deferred), tag results.
- Day 4 PM: Update `internal/eval_harness/result.go` to include `microrag_state` field (~20 LOC).
- Day 4 PM: Update result writer to include the tag in `eval_results/` JSON (~20 LOC).
- Day 4 PM: Run small benchmark group ON and OFF; verify delta computation.

**Acceptance Criteria:**
- [ ] `ailang eval-run --microrag=auto --benchmarks=string-ops --models=claude-sonnet-4-6` runs both arms and tags.
- [ ] `eval_results/.../results.json` carries `microrag_state` per benchmark.
- [ ] Pilot run completes successfully; no eval-suite regressions.

**Risks:**
- Existing eval-runner is large — Mitigation: minimal touch, use existing env-injection pattern.
- Statistical weakness on small benchmark counts — Mitigation: documented in design doc; user runs on full suite for production measurement.

---

### Milestone 7: M7_TEST_TELEMETRY — Telemetry, integration tests, dogfood validation

**Goal:** Production-grade observability + integration tests + 1-week dogfooding window before global rollout.

**Estimated:** ~400 LOC tests + ~150 LOC telemetry hooks = ~550 LOC
**Duration:** 1 day implementation + 1 week passive observation

**Tasks:**
- Day 5 AM: Telemetry plumbing (~150 LOC) — emit via existing pipeline:
  - `microrag.injection.count`, `microrag.injection.tokens`.
  - `microrag.cache.hit`, `microrag.cache.miss`.
  - `microrag.dedup.suppressed`, `microrag.bypass.fired`.
  - `microrag.disabled` (when env-toggle off).
  - `microrag.spawn.duration_ms` (for daemon-mode decision).
- Day 5 AM: Integration tests (~250 LOC):
  - Full-session simulation: 30 synthetic Edit/Write events; verify count/dedup/budget.
  - Multi-route session: `.ail` + `.go` + `CLAUDE.md`; verify routing.
  - 5-edit burst cache stability: identical injection bytes calls 2–5.
  - MCP smoke: tools/call returns same payload as CLI.
- Day 5 PM: Negative tests (~100 LOC):
  - `AILANG_MICRORAG_ENABLED=invalid` → defaults to enabled.
  - Ollama down → FTS fallback returns something.
  - Empty/missing `microrag.yaml` → graceful no-op.
  - Corrupt `injections.jsonl` → truncate-and-continue.
- Day 5 PM: Eval A/B run on small benchmark group; document delta.
- Day 5 PM: Manual relevance sample of 50 injections; rate ≥70% relevant.
- Days 6–12 (passive): dogfood in AILANG repo only (project `.claude/settings.json`); collect telemetry; confirm cache hit rate ≥0.6 over 5-edit bursts.

**Acceptance Criteria:**
- [ ] All integration tests pass.
- [ ] Telemetry visible via existing dashboard / `ailang trace` commands.
- [ ] AI prompt-cache hit rate ≥0.6 over 5-edit burst (telemetry-verified).
- [ ] Manual relevance ≥70% on 50-sample audit.
- [ ] Eval A/B delta computed on at least one benchmark group.
- [ ] No crashes during 1-week dogfooding window.
- [ ] CHANGELOG.md updated with v0.15.0 entry.

**Risks:**
- 1-week dogfooding window stretches into the next release cycle — Mitigation: ship to project-level only initially; promote to global `~/.claude/settings.json` only after the dogfooding gate.
- Prompt-cache metric may be hard to extract — Mitigation: existing `claude_telemetry.sh` already captures `cache_creation_input_tokens` vs `cache_read_input_tokens`; aggregate from there.

---

## Success Metrics

- **Coverage:** ≥80% of Edit/Write events on `.ail` files trigger a relevance check.
- **Precision:** ≥70% of injected snippets judged relevant on 50-sample audit.
- **Cache stability:** ≥0.6 prompt-cache hit rate over 5-edit bursts.
- **Token economy:** ≤5000 injection tokens per session at p95; per-injection ≤150 tokens.
- **Latency:** Engine p95 <200ms (hit), <1.5s (cold). Frontend overhead <20ms.
- **Mistake reduction:** ≥20% PAR_001 + wrong-operator drop on `claude-sonnet-4-6` (ON vs OFF) within 2 weeks of rollout.
- **Harness portability:** MCP smoke test passes on at least one non-Claude-Code harness.
- **Test coverage:** New code in `internal/microrag/` ≥75% line coverage.
- **Examples passing:** No regression in `make verify-examples`.
- **Documentation:** `docs/docs/guides/microrag.md` complete; CHANGELOG updated.

## Dependencies

- M-BRAIN, M-BRAIN-VECTORS, M-BRAIN-CONTEXT (all shipped).
- Ollama with `nomic-embed-text` running locally (existing brain dependency).
- `release-manager` and `post-release` skills (will be modified in M3).
- One non-Claude-Code MCP harness installed locally for M5 smoke test (Cursor preferred, Continue/Cline acceptable).

## Open Questions

- **MCP library choice:** does the repo already use a Go MCP library? If not, pick one (`mark3labs/mcp-go` is common). Resolved at M5 start.
- **`--microrag=auto` parallel vs sequential:** sequential is simpler for v1; parallel deferred to follow-up.
- **Dogfooding gate:** do we promote to global `~/.claude/settings.json` automatically after 1 week, or wait for explicit human approval? Defaulting to explicit approval.

## Notes

- Engine + frontends split is the load-bearing architectural choice — keeps the eval toggle, dedup, and cache logic in one Go binary regardless of harness.
- Release-tied reindex (M3) is a small but high-impact piece — protects against corpus drift forever, at zero ongoing maintenance cost.
- The 1-week dogfooding window in M7 is real elapsed time, not implementation work — sprint can be marked "complete" after M7 implementation lands; rollout-to-global is a separate post-sprint decision.
- This sprint does NOT modify the existing `brain_session.sh` or `brain_resolution.sh` — they continue to fire alongside the new μRAG hooks.
