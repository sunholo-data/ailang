# Sprint Plan: M-V1-SIMPLIFY-S3 — One of each, part 1

**Design doc**: [m-v1-simplification-program.md](m-v1-simplification-program.md) Phase 2 items 2.1 (sqliteopen + config.yaml loader), 2.2, 2.3, 2.5, 2.7, 2.8, 2.9, 2.10, 2.11, 2.12, 2.13
**Sprint ID**: M-V1-SIMPLIFY-S3
**Created**: 2026-09-15
**Rulings applied**: D2 (pass predicate = `CompileOk && RuntimeOk && StdoutOk`; re-bank nothing; annotate the OS-history boundary)

## Summary

Every verified-duplicate cluster whose files do not overlap with another cluster's, run as five parallel worktree agents with explicit file ownership. Each PR names the survivor and cites `git log -S` for what it removes. Part 2 (Sprint 4) takes the cross-cutting items: the remaining behaviour-affecting fallbacks (2.6), the env-var migration into `internal/config` that `forbidigo` needs (2.14, ~340 sites), the `ai-check` verify-loop unification and the `sendHandoffMessage` duplicate found in Sprint 2.

**On the D2 quorum step.** The program doc asked for `ailang design-quorum` before Phase 2 because D2 touches banked-data semantics. Mark ruled D2 attended on 2026-09-15 with the doc's evidence in front of him — the decision the quorum protects is made. In its place M1 carries the measurement the quorum would have demanded: the discordant count between the two predicates over every banked results dir, recorded in the PR and in the OS-history annotation.

**Duration:** 4 days (5 agents in parallel + close-out)
**Risk Level:** Medium — M1 changes what "pass" means in analysis code (measured, annotated, nothing re-banked); M3 changes the backend-selection contract (old names become hard errors with the replacement named).

## Milestones and file ownership

### M1 — Eval rows, loaders, pass predicate (D2) · owner: `internal/eval_harness/{metrics,cost_tally,rotation_summary,repair,paired}*.go`, `internal/eval_analysis/**`, `internal/eval_analyzer/**`, `cmd/ailang/eval_*.go`
- `eval_analysis.BenchmarkResult` embeds `eval_harness.RunMetrics` (drifted fields — `reason_tokens`, `llm_wall_ms`, `ttft_ms`, `compaction_*`, `verify_*` — stop reading as zero)
- one `eval_harness.LoadRows(dirs, opts)` with the validity filter replaces the 5 loaders (`eval_analyzer/analyzer.go:110` has none today, so harness errors count as model failures in `eval-analyze`)
- one `(*RunMetrics).Passed()` = `CompileOk && RuntimeOk && StdoutOk`; the ~86 `.StdoutOk` reads that mean "passed" call it; reads that genuinely mean "stdout matched" stay
- **Measurement (required):** over every dir under `eval_results/baselines/`, count rows where the two formulas disagree, per version and per benchmark; put the table in the PR and add a dated boundary note to whatever the OS dashboard reads for history (find the annotation mechanism used for the v0.30.0 cost-data note and reuse it)

### M2 — Pricing onto `modelreg` · owner: `internal/observatory/{pricing,seed}.go`, `internal/executor/{executor.go CostModel, claude/cost.go, codex/cost.go}`, `internal/mission/quorum/run.go`, `internal/modelreg/**`
- delete `observatory/pricing.go`'s second `models.yml` loader, its 10%/125% cache hardcode and private alias table; aliases move into `models.yml`; observatory calls `modelreg.CalculateCostForModelWithCache`
- executor `CostModel` + the hard-coded dollar tables become a thin adapter over modelreg (keep `MinimumCharge` if a test pins it — say so); quorum `estimateCost` and `seed.go`'s rate map call modelreg
- **Measurement:** recompute cost for the last 200 observatory spans with the old and new paths; report the delta by provider (non-Anthropic cache rows are expected to change — that is the bug)

### M3 — One plane switch + `sqliteopen` + one config.yaml loader · owner: `internal/storage/**`, `internal/coordinator/{store_sqlite,agent_config,daemon_github,daemon,daemon_tasks_init}.go`, `internal/observatory/{backend_sqlite,store,models}.go`, `internal/messaging/{schema,config}.go`, `internal/platform/sharedmem/**`, `cmd/ailang/{storage,messages,coordinator_remote_store,chains_read_backend,chains_post,daemon,chains_data,chains_diagnostics,eval_confidence,eval_saturation}.go`, new `internal/sqliteopen`
- `AILANG_STORAGE=local|gcp|hybrid` plus optional `AILANG_STORAGE_{MESSAGING,COORDINATOR,OBSERVATORY}` overrides is the ONE switch; `AILANG_MESSAGES_STORE`, `AILANG_COORDINATOR_REMOTE`, `AILANG_CHAINS_READ`, `AILANG_CHAINS_CLOUD` become hard errors naming the replacement (one release), `COORDINATOR_MODE` is derived; `cmd/ailang/daemon.go:129`'s `os.Setenv("AILANG_STORAGE")` goes; `hybrid` gets the secret approver it silently lacked
- `ailang storage status` prints **per store**: resolved mode, its source (which var/default), and the path or project
- `internal/sqliteopen` (leaf): one WAL/busy_timeout/pool/foreign-keys recipe; the 9 openers call it (`observatory.db` is opened today with two different pool caps — pick one, say which, test it); the 5 raw `sql.Open` in cmd get WAL/busy_timeout and stop swallowing errors
- `~/.ailang/config.yaml`: one struct + one loader in `internal/config` (extend the leaf from S2; it may import yaml only) replacing the 3 loaders and 8 re-parses; `AILANG_CONFIG` honoured by all readers; `daemon_github.go`'s parse-error→`"sunholo-data/ailang"` fallback becomes an error
- CLAUDE.md's session-start block changes with this (`AILANG_MESSAGES_STORE=gcp` → `AILANG_STORAGE_MESSAGING=gcp`); update it, `.claude/rules/*`, `docs/docs/guides/*`, `tools/launchd/**` and any plist/env file that sets the old names — grep the whole repo, list every file changed

### M4 — One AI client factory, one error classifier, one prompt loader · owner: `internal/ai/**`, `internal/prompt/**`, `internal/agentprompt/**`, `internal/devtoolsprompt/**`, `internal/eval_harness/{ai_provider,ai_agent,prompt_loader}.go`, `cmd/ailang/{exec,ai_handlers,coordinator_lifecycle}.go`, `internal/mission/quorum/call.go`, `internal/coordinator/task_executor.go`
- `ai.NewProviderFor(name, opts)` replaces the 5 env-key→NewClient switches (`exec.go` pins `127.0.0.1:11434` — decide the one default and test it); `ai.ClassifyError` replaces the 3 retryable matchers (`configdriven` never maps 429 to `CodeRateLimit` today — fix, test); `openrouter/{chat,types}.go` reuse openai's request types; one `ai.doJSON` for the HTTP boilerplate the 5 clients repeat
- `internal/prompt` parameterised by subdir with a frozen-check option replaces `agentprompt/loader.go`, `devtoolsprompt/loader.go` (identical but for a literal) and `eval_harness/prompt_loader.go`; the four `findProjectRoot`/`SetEmbeddedFS` copies collapse with it; `make check-prompt-freeze` must still pass

### M5 — HTTP layer, proctree leaf, simhash leaf, observatory dupes, small helpers · owner: `internal/server/**`, `internal/apiserver/**`, `internal/coordinator/{daemon_http,event_handler,evaluation_verdict,backstop_sweep,human_interaction,history_collector,analyzer}.go`, `internal/observatory/{api,api_spans,api_enrichment,outliers,websocket,store_aggregates_hierarchy,importer_motoko}.go`, `internal/websocket/**`, new `internal/proctree` (from `internal/executor/proctree`; consumers `executor/*`, `executor/motoko`, `internal/smt/process_*.go`, `internal/pkg/process_*.go`, `eval_harness/process_unix.go`), new `internal/simhash` (from `builtins/simhash.go`, `messaging/simhash.go`, `effects/sharedindex*.go`, `docsearch/search.go`, `coordinator/analyzer.go`, `storage/firestore/messaging_search.go`, `platform/sharedmem/sqlite.go`), new `internal/strutil` for `truncate`/`truncateString`/`fileExists`/`sortedKeys` **except** files owned by M1–M4 (leave those; list them for Sprint 4)
- `writeJSON` ×7 → one `httpjson.Write` (or `strutil`; agent picks, one place); `internal/websocket/server.go` vs `observatory/websocket.go` → one hub; `coordinator/daemon_http.go:19` auth constant-time and **fail-closed** when the env var is unset (a test that proves a request with no token is rejected); the `/api/messages` route collision named and resolved (one of the two gets a distinct path; docs updated)
- **proctree must be a leaf** under `internal/proctree`: `smt` and `pkg` are core packages and must not import anything under `internal/executor` (the closure test will fail if they do — that is the check); delete the three byte-equivalent copies
- **simhash must be a leaf**: one algorithm (state which of the three — FNV/MD5/FNV-variant — survives and why; the stored hashes in messaging/sharedmem are then a migration question — if hash spaces change, add a schema-version bump and a re-index path, do not silently mix); one Hamming, one cosine
- observatory: one span-name classifier (3), one tree builder (4); `importer_motoko.go` reuses `executor/motoko/parser.go`'s event decoder so imported chains keep cache-token fields
- `dashboard.go:87 getInt` reads a Firestore int64 as 0 — fix with a test

### M6 — Close-out (orchestrator)
- `make simplicity-audit` vs the 2026-09-15 snapshot: `dup_symbol_names` and `backend_switches` down; no unexplained gated regression
- `make test`, all CI gates, `internal/diag` closure test green (proctree/simhash leaves inside the core, no executor import)
- Sprint 4 backlog written from every "left for Sprint 4" line in the five reports

## Success Metrics
- `backend_switches` 6 → 1; `dup_symbol_names` 31 → ≤ 20; pricing sources 4 → 1; result loaders 5 → 1; pass-predicate formulas 2 → 1; prompt loaders 4 → 1; AI factories 5 → 1; `writeJSON` 7 → 1; procgroup copies 4 → 1; simhash 3 → 1
- Every removal PR cites `git log -S`; every behaviour change carries a measurement (M1 discordant table, M2 cost delta, M3 per-store status output, M4 429 classification test, M5 fail-closed auth test)
