# Sprint Plan: M-V1-SIMPLIFY-S4 — One of each, part 2 (closes Phase 2)

**Design doc**: [m-v1-simplification-program.md](m-v1-simplification-program.md) Phase 2 items 2.6, 2.14, plus every "left for Sprint 4" line from the five Sprint 3 reports
**Sprint ID**: M-V1-SIMPLIFY-S4
**Created**: 2026-09-15 (from the Sprint 3 close-out)
**Status**: Planned — starts after Sprint 3 is green on origin

## Summary

Sprint 3 collapsed the seven disjoint duplicate clusters. What remains of Phase 2 is cross-cutting: the behaviour-affecting silent fallbacks (2.6), the env-var migration into `internal/config` that the `forbidigo` rule needs (2.14, ~300 `os.Getenv` sites outside config after S3), the generated env-var reference, and the specific leftovers each S3 agent could not touch because another agent owned the file. Two items need a ruling from Mark and are listed first.

**Duration:** 4 days
**Risk Level:** Medium — 2.6 turns defaults into errors (each behind the S2 `DeprecatedDefault` warning path first); 2.14 is wide but mechanical and lint-enforced at the end.

## Needs a ruling before its milestone starts
- **`internal/mission/dispatch/run.go:235`** pins stage executions to the prod message plane by literal (`ailang-multivac`). Converting it to `config.CloudProject` means a parent `AILANG_MESSAGES_PROJECT`/plane could re-route mission stages. Loop-behaviour ruling.
- **Write-side pass-flag asymmetry** (S3 M1): agent mode gates `stdout_ok` on `runtime_ok`, standard mode does not. Making the three flags mean the same in both modes is a bank-forward change (no re-bank). Ruling: unify, or document as intended.

## Milestones

### M1 — Remaining silent fallbacks → typed errors (2.6), via DeprecatedDefault first
Sites (config audit, re-verify each): provider `"claude"` `coordinator/daemon_tasks_exec_run.go:~210`; model `"haiku"` `eval_harness/agent_runner.go:~67`; workspace `"default"` `cmd/ailang/coordinator_cloud.go:~68,82`, `coordinator/daemon_tasks_init.go:~447`; embedding model `messaging/embedder.go:~146,157` (vector-dimension mismatch = silent search corruption); gemini location `"global"` `ai/gemini/client.go:~84`; loader stdlib `"."` `loader/loader.go:~369`; `AILANG_BIN`→PATH `eval_harness/agent_validation.go:~215` (the stale-binary trap); approval timeout 0→1h `coordinator/approval_checkpoint.go:~82`; budget 0→unlimited `coordinator/daemon_tasks_budget.go:~42-48`; secret approver gcp-without-URL un-gated `internal/runner/secret_approver.go` (M-SECRET-REMOTE-APPROVAL-WIRING M2 note). Each: warn-once now via `config.DeprecatedDefault`, error under `AILANG_STRICT_CONFIG=1`, test both.
Also from S3: `codex.go:~663` / `managed_agents.go:~289` bank `$0` when `cm.Unpriced` → `CostProvenanceUnknown`; `otlp_receiver.go:389,634` stamp `ailang.cost.unpriced=true`; `eval_harness/agent_runner_multi.go:~299` + `mission/dispatch/run.go:~233` hand-copy rates → `executor.CostModelFromPricing`.

### M2 — Env-var migration into `internal/config` + `forbidigo` + generated reference (2.14)
~300 `os.Getenv`/`LookupEnv` sites outside `internal/config`/`internal/testutil` (`DEBUG_*` compiler knobs exempt by rule). Typed getters grouped by area (`config.Ollama*`, `config.Eval*`, `config.Mission*`, …) with the precedence flag > env > file > default written once in `doc.go`. Then `.golangci.yml` `forbidigo` on `os.Getenv|os.LookupEnv` outside the two packages, and `make docs-env` generating `docs/docs/reference/env-vars.md` from the getters (name, default, precedence, one line each) so `env_vars_documented_pct` reaches 100 by construction. `AILANG_TOPIC_PREFIX` fallbacks (`coordinator/daemon_tasks_init.go:~417`, `cmd/ailang/coordinator_lifecycle.go:~162`) → `pubsub.TopicPrefixFromEnv`. `ollama.NewClient`'s `os.Setenv("OLLAMA_HOST")` side effect → explicit plumbing.

### M3 — Sprint 3 leftovers, by former owner
- eval (M1-owned): `eval_harness/process_unix.go` → `internal/proctree`; `eval_analysis/dashboard_io.go:154 writeJSONAtomic` (file writer — leave or `strutil`); `truncate` `eval_analysis/formatter.go:249`, `eval_analyzer/analyzer.go:285`; `sortedKeys` `eval_analyzer/analyzer.go:292`, `cmd/ailang/eval_matrix_sections.go:132`; `fileExists` `eval_analysis/loader.go:383`; `display.Truncate`/`telemetry.Truncate` delegates → `strutil.Truncate` at callers in `internal/ai/**`, `cmd/ailang/eval_benchmark*.go`, then delete the delegates; `messages_util.go:475 truncateString` survives only for `exec.go:219`; `tools/gen-anchor/main.go:80`, `tools/direction-fit/main.go:120` → `r.Passed()`; `tools/os-release-snapshot.sh` copies `notes` into `os/history.json`; `benchmarks/events.yml` gets the D2 `kind: taxonomy` entry at v0.38.8; `eval_cache_warmup.go:156` stale comment.
- storage (M3-owned): `storage/firestore/messaging_search.go:273 simhashSimilarity`, `platform/sharedmem/sqlite.go:730/753 cosineSimilarityF32` + `hammingDistance64` → `internal/simhash`; coordinator `tasks.fingerprint` re-index (ASCII-only variant → survivor; `store_sqlite.go` + `storage/firestore/coordinator_transitions.go`); `getInt`/`getString` `storage/firestore/coordinator_convert.go:273`, `server/ailang_bridge.go:286`.
- ai/prompt (M4-owned): delete `internal/agentprompt` + `internal/devtoolsprompt` (thin aliases now) and point the 4 cmd callers at `internal/prompt`; `eval_harness/prompt_frozen.go` dead error types; `error_categorizer.go:isQuotaExhaustion` → `ai.IsQuotaExhausted`; `internal/module` vs `prompt` `findProjectRoot` (last two copies).
- Sprint 1/2 findings: `sendHandoffMessage` vs `sendAgentHandoffMessage` → one sender; `ai-check`'s private verification loop vs `verify` (three measured divergences) → one loop behind an option, with `ai_check_exit_test.go` extended to pin the contract explicitly.
- observatory: `chat_messages`/`chain_stages` gain cache-token columns so imported motoko chains store what the importer now decodes (schema migration).

### M4 — Close-out
- `make simplicity-audit`: `getenv_outside_config` → 0, `env_vars_documented_pct` → 100, `dup_symbol_names` ≤ 12; forbidigo green in `make lint`
- Every Phase 2 row of the design doc's table marked closed or explicitly deferred with a reason; Phase 3 (CLI dispatch table) is next

## Verification carried from Sprint 3 that must stay green
`internal/diag` closure test (leaves must stay leaves), `check-architecture-closure`, `check-home-isolation`, `check-prompt-freeze`, `TestRequireAPIKey_FailsClosed`, the D2 `TestD2DiscordantCount` (skipped unless `AILANG_EVAL_RESULTS_DIR` is set — run it once more after M3's eval changes).
