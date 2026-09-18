# M-V1-SIMPLIFICATION-PROGRAM: One of Each — Simplify the Codebase for v1.0.0

**Status**: In progress — Sprint 1 (Phase 0 + 1.1–1.2 + weekly audit) landed 2026-09-14; Design Freeze ruled 2026-09-15
**Target**: v1.0.0
**Priority**: P0
**Estimated**: 6 phases, ~7 weeks elapsed (≈ 22 agent-days of sprint work; phases 0–3 are the release gate, phases 4–5 are the polish)
**Dependencies**: None. This is an umbrella program; each phase spawns its own sprint plan via sprint-planner.
**Created**: 2026-09-14
**Audit basis**: six read-only research passes on dev @ 7c56fd266 (core-vs-platform, duplicates, CLI surface, config sprawl, artefacts/docs, AI navigability). Every number below is from those passes; the Verification Log at the end lists the command behind each load-bearing claim.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One pass predicate and one result loader replace two formulas and five loaders that currently disagree (§P2). |
| A2: Replayability | 0 | Trace format untouched. |
| A3: Effect Legibility | 0 | No effect changes. `effects` loses accidental imports, not semantics. |
| A4: Explicit Authority | +1 | Prod defaults (`"ailang-multivac"`, `europe-west1`) become hard errors when unset; the fail-open daemon auth is closed. |
| A5: Bounded Verification | +1 | A `go list -deps` closure test makes the core boundary a checked invariant instead of a diagram. |
| A6: Safe Concurrency | 0 | No concurrency changes (the opencode process-group fix is a correctness fix, not a model change). |
| A7: Machines First | +1 | Fewer commands, one `--json`, generated help, one skill tree, ≤25 KB always-on instruction surface — every item is measured in agent tokens. |
| A8: Minimal Syntax | 0 | No language syntax changes. |
| A9: Cost Visibility | +1 | One pricing source (`modelreg`) replaces four; the observatory stops printing `$0.00` on a wrong cwd. |
| A10: Composability | 0 | — |
| A11: Structured Failure | +1 | Silent fallbacks in 17 behaviour-affecting sites become typed errors. |
| A12: System Boundary | +1 | Language core no longer transitively links Firestore/sqlite/grpc/otel. |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |

---

## Problem Statement

After ~38 minor releases the repo is large, and more importantly it has **more than one route to most things**. An AI agent working in it pays for that every session:

| Measure (dev @ 7c56fd266) | Value |
|---|---|
| Non-test Go | 370k LOC in 124 `internal/` packages + 63k LOC in a single `cmd/ailang` main package (229 files, one 89-case `switch`) |
| Language core (lexer→vm, stdlib, codegen, fmt/lsp/repl) | ~149k LOC, 40% |
| Operations platform (coordinator, missions, messaging, observatory, eval harness, executors, AI clients, servers, storage) | ~152k LOC, 41% |
| Packages linked by `ailang fmt` | 112 of 124 (a language-only closure needs 34) — the binary is 100 MB |
| Top-level CLI commands | 78 (+5 aliases), ~185 subcommands, 439 distinct flag names, zero short flags; 15 commands absent from `--help`; `--help` itself is rejected by 8 command groups |
| Env vars read via `os.Getenv` | ~225 distinct at 445 sites; 41 read from more than one package; **11% documented**; no `internal/config` package |
| "Which backend?" switches | 6 independent env vars + `--remote`; `ailang storage status` reads only one of them |
| Silent behaviour-affecting fallbacks | 17 sites, several defaulting to the **prod** GCP project |
| Duplicate implementations (verified) | eval result row ×2, result loaders ×5, pass predicate ×2 formulas, pricing sources ×4, approval paths ×3 (one dead, one buggy), executor supervisor loops ×4, process-group kill ×3 (opencode uses none), SQLite openers ×9, state-dir resolvers ×7, config-file loaders ×8, AI provider factories ×5, prompt loaders ×4, `writeJSON` ×4, WebSocket hubs ×2, SimHash ×3, `truncate` ×11 |
| Verified dead code | `Daemon.HandleApproval` family (~500 LOC), `eval_harness.runHeadlessSession` + streaming (~830 LOC), `observatory/backend_jaeger.go` (471), `backend_gcp*`+`composite` (~1,400, gated on an env var nothing sets), `internal/cognition` (0 importers), root `testutil/` (0 importers) |
| Packages with no package comment | 19 of 124; 5 more have a *wrong* one (`internal/eval` says it is the builtins registry) |
| Confusable package families | `eval`/`eval_harness`/`eval_analysis`/`eval_analyzer`; `trace`/`telemetry`/`observatory`; `testing`/`testutil`/`test`; `link`/`linked`; `server`/`apiserver`; `messaging`/`notify`/`pubsub`/`daemon`/`dispatch` |
| Logging mechanisms | 7 (no `slog`); 60% of `fmt.Errorf` calls drop the `%w` chain |
| Tracked files | 24,849, of which **17,410 (70%) are `eval_results/`** despite a `.gitignore` entry; 11 sprint JSON/plan files tracked at repo root |
| Skill trees | 2: `.agents/skills` is a 249-file copy of `.claude/skills`, diverged in 50 files since 07-24; codex/pi agents read the stale one |
| Always-on instruction surface (Claude Code) | ~51 KB (~13k tokens) before any work; MEMORY.md is 45% of it |
| Build / test | warm build 9 s, cold 32 s; test **compile floor 53 s**; no language-only test target; `make help` is 197 lines |
| `design_docs/planned` | 244 files; ~14 are marked LANDED/PARKED but never moved; `implemented/v1_0_0` already holds 13 docs for a release that does not exist |

Two of the duplications are **live bugs**, not just debt:
- `internal/executor/opencode/opencode.go` has seven bare `cmd.Process.Kill()` and no `SysProcAttr` — a timeout orphans grandchildren (same class as the port-8080 zombie in memory).
- `internal/server/handlers_coordinator.go:170` resolves an approval via `ResolveApprovalRequest` and never calls `ProcessApprovalRequest`, so a dashboard approve silently skips handoffs (the 2026-09-07 root cause is still present on this path).

None of this is file size: no file exceeds the 800-line gate. The complexity is **count and duplication**, which is exactly what a file-size gate cannot see.

## Goals

**Primary goal:** ship v1.0.0 with *one implementation per concept*, a language core that is a verified leaf, and a CLI/config/instruction surface an agent can hold in context.

**Success metrics** (each is produced by `make simplicity-metrics`, added in Phase 0, and reported in the v1.0.0 release notes):

| Metric | Today | v1.0.0 gate |
|---|---|---|
| **Non-leaf** internal packages in the `run/check/fmt/prompt/repl` closure | 30 (44 incl. 14 leaves; 124 linked by the binary) | ≤ 36, enforced by test. Metric split 2026-09-15 (S3): a leaf (imports nothing under the module — `config`, `statedir`, `proctree`, `simhash`, `strutil`, `httpjson`…) adds code, not coupling, so every consolidation into a leaf was reading as closure growth. The gated number is the coupling that can ripple. |
| Third-party roots in that closure from {sqlite3, otel-sdk/exporters, grpc, websocket, cloud.google.com/go/{firestore,pubsub,storage,trace}} | 5 | 0 |
| Top-level commands visible in default `ailang --help` | 78 | ≤ 20 |
| Commands where `<cmd> --help` exits 0 | ~70% | 100% |
| Distinct `os.Getenv` call sites outside `internal/config` (excluding `DEBUG_*`) | ~400 | 0, enforced by `forbidigo` |
| Env vars documented | 11% | 100% (generated from `internal/config`) |
| Behaviour-affecting silent fallbacks (the 17 listed in the config audit) | 17 | 0 |
| Verified-duplicate clusters open (§P2 list) | 16 | ≤ 4 (the four deferred L-sized ones) |
| `internal/` packages with a correct package comment | 100 / 124 | 124 / 124, enforced |
| Skill trees | 2 | 1 |
| Always-on instruction surface | ~51 KB | ≤ 25 KB |
| `make test-core` wall time | n/a | < 15 s |
| Tracked files | 24,849 | < 8,000 |

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1. Binary shape: one `ailang` with `ops`/`dev` groups (hidden) **vs** two binaries (`ailang` + `ailang-ops`) | 286 ops references to `ailang messages`, 107 to `ailang coordinator` live in launchd drivers, mission-control.sh and skills; five build paths ship the binary (go install, release matrix, Cloud Build buildpack, agent Docker base, CI) and the agent containers run `ailang execute-job`. A rename without aliases, or a split that misses one build path, breaks a loop. | human | design | high |
| D2. Canonical pass predicate: `CompileOk && RuntimeOk && StdoutOk` (8 sites) **vs** `StdoutOk` (~45 sites) | Changes published pass rates for the runtime-fail/stdout-match slice; touches banked-data semantics (quorum trigger 3). | human | design | high |
| D3. Prod defaults become hard errors (`"ailang-multivac"`, `europe-west1`, repo `sunholo-data/ailang`, provider `claude`, model `haiku`, workspace `default`) | Every fleet plist and Cloud Run env that relied on the default fails loudly on first run after the change. That is the point, but it must be scheduled, not discovered. | human | design | med |
| D4. `internal/eval` → `internal/interp` rename and `eval_analysis`+`eval_analyzer` merge | Import-system rename; the Sept-2025 disaster class. Mechanical with gofmt -r, but must run `make test-imports` + `make verify-examples` between steps. | human | design | med |
| D5. `.agents/skills`: symlink to `.claude/skills` **vs** CI-diffed generated copy | `coordinator_cloud_executor.go` and AGENTS.md route codex/pi there; a symlink may not survive the cloud container clone. | human | design | med |
| D6. Sprint state (`.ailang/state/sprints/*.json`) and `eval_results/` baselines: commit all **vs** untrack all | 17,410 tracked baseline files; docs site loaders and the OS-publish pipeline read them (memory says hosting was decoupled to the bucket — verify). | human | design | med |
| D7. Which zero-reference commands are deleted vs hidden (`access-control`, `dashboard`, 8 `observatory` subs, `trace` group, `eval-trend`, `eval-censored-pairs`, `watch`, `axioms`, `policy-check`, `budget`, `storage`, `models source/publish`) | Coding-standards rule: never delete on "unused" alone. Each has a last-commit date in the CLI audit; the owner ratifies the list. | human | design | low |

### Design Freeze

**Ruled by Mark, 2026-09-15** (attended; recommendations accepted except D1, which is sharpened):

- [x] **D1 — two binaries, but split only when it is certain not to disrupt.** Sequence: Phase 3 ships ONE binary with a dispatch table, `ops`/`dev` groups and every old name aliased (v0.39.x). The physical split into `cmd/ailang` + `cmd/ailang-ops` is Phase 3b and lands before v1.0.0 **only if all of its preconditions hold** (listed under Phase 3b); otherwise v1.0.0 ships one binary and the split moves to v1.1. Either way no caller changes: `ailang <ops-command>` keeps working by delegating to `ailang-ops`.
- [x] D2 — canonical pass predicate is `CompileOk && RuntimeOk && StdoutOk`; re-bank nothing; annotate the OS history boundary date.
- [x] D3 — prod defaults become a deprecation warning in v0.39 and a hard error in v1.0.0.
- [x] D4 — rename `internal/eval` → `internal/interp`, merge `eval_analysis` + `eval_analyzer`; `link`/`linked` and `server`→`hub` stay Future Work.
- [x] D5 — `.agents/skills` becomes a generated copy with a CI diff gate (not a symlink; the cloud container clone is the consumer).
- [x] D6 — untrack sprint JSON per the recommendation. **`eval_results/baselines/` stays tracked**: Sprint 1 found it is deliberately re-included in `.gitignore` and read by the docs BenchmarkDashboard and `eval-weekly.yml`; moving those readers to the bucket is a separate item before any untracking.
- [x] D7 — the removal list stands as written; each removal PR cites the audit's last-commit date and reference counts.

**Ruled by Mark, 2026-09-15 (second round, from the Sprint 3 leftovers):**

- [x] **D8 — mission dispatch prod pin: keep it, make it explicit.** `internal/mission/dispatch/run.go` keeps filing every stage to the prod message plane regardless of the parent environment; the literal becomes a named constant (`config.MissionMessagePlaneProject`, a fixed value that is deliberately NOT a registered Var, so no environment variable can move it) with a comment saying it is deliberate. No behaviour change. (Applied in S4 close-out, M4.)
- [x] **D9 — pass flags unified, bank-forward.** Standard mode gates `stdout_ok` on `runtime_ok` from the next run, as agent mode already does. Nothing re-banked; the D2 read-side predicate already makes published rates identical, so this only makes stored flags consistent. A dated note joins the D2 annotation. (Applied in S4 close-out.)

## Solution Design

### Overview

Six phases, ordered so that **each phase makes the next one safe to do**: measure first, then fix the live bugs and delete verified-dead code (trust), then make the core boundary a test (so later moves cannot regress it), then collapse duplicates behind that boundary, then reshape the CLI on top of a clean core, then fix the navigation layer (docs, skills, instruction surface) once names have settled. Phases 0–3 gate v1.0.0; 4–5 can overlap the release.

Every phase lands as small PRs from worktrees (concurrent agents share the main checkout), and the mission loops are restarted only on a stable base — harness debt is cleared attended, per memory.

### Architecture

```
                 ┌──────────────── cmd/ailang (dispatch table, groups, hidden) ───────────────┐
                 │  run check fmt test repl iface verify prompt docs examples lsp init serve   │
                 │  pkg …   eval …   [dev …]   [ops … → messages coordinator mission chains …] │
                 └───────────────┬──────────────────────────────────────┬─────────────────────┘
                                 │ language closure (≤36 pkgs, tested)  │ platform
   internal/config ◄── leaf ─────┤                                      │
   internal/statedir ◄── leaf ───┤  lexer parser ast types elaborate    │ coordinator mission messaging
   internal/sqliteopen ◄─ leaf ──┤  eval(interp) effects builtins       │ observatory eval_harness executor
                                 │  pipeline link loader format repl    │ ai/* storage/* server apiserver
                                 │  gen vm bytecode smt lsp prompt      │ telemetry(otel impl) …
                                 │  telemetry = no-op interface here    │
                                 └──────────────────────────────────────┴─────────────────────
   Registration seams (platform registers into core at cmd init, never imported by core):
     effects.RegisterAIHandler / RegisterSharedMem / RegisterStream   telemetry.SetExporter
```

The only architectural addition is **three leaf packages** (`config`, `statedir`, `sqliteopen`). They must be leaves because `internal/storage` implements the Store interfaces of coordinator/messaging/observatory and therefore cannot be imported by them — which is *why* seven state-dir resolvers and nine SQLite openers exist today.

### Implementation Plan

#### Phase 0 — Measure, fix the live bugs, delete the verified-dead (week 1, ~3 days)

Goal: a baseline nobody can argue with, and the two bugs closed before anything is moved.

1. `make simplicity-metrics` → `tools/simplicity_metrics.sh` emitting the Goals table as JSON + markdown (closure size via `go list -deps`, command count via the dispatch table, `os.Getenv` sites, tracked files, always-on KB, `make test-core` time). Banked under `.ailang/state/simplicity/<date>.json` so the release notes can diff.
2. **Live bug**: opencode executor adopts `internal/proctree` (`SysProcAttr` + group kill); replace the seven bare `Process.Kill()`. Regression test with a grandchild that must die.
3. **Live bug**: `server/handlers_coordinator.go` approve route calls `coordinator.ProcessApprovalRequest`; delete the `ResolveApprovalRequest`-only path. Test: dashboard approve triggers the handoff hook.
4. Delete verified-dead code (each with a `git log -S` note in the PR body, per coding-standards): `daemon_approval.go` HandleApproval family; `eval_harness/agent_runner.go runHeadlessSession` + `agent_runner_streaming.go`; `observatory/backend_jaeger.go`; `backend_gcp*.go` + `backend_composite.go` + the `AILANG_ENABLE_GCP_TRACE` read; `internal/cognition`; root `testutil/`; `benchmark/` (2025-11 "temporary"); `output/`.
5. Root housekeeping: move the 11 tracked `sprint_*` files to `.ailang/state/sprints/` / `design_docs/planned/`; delete the six eval-cwd droppings and fix the leak (eval workspace must never resolve to repo root); `make build` emits to `bin/`; `.gitignore` gets `gen/`, `.eval_workspace/`; per D6, `git rm --cached` `eval_results/` and sprint JSON.
6. `make test-core` (lexer, parser, types, elaborate, eval, effects, pipeline, link, loader, module, format, stdlib) and mention it in CLAUDE.md as the inner loop.

#### Phase 1 — Make the core boundary a test (weeks 1–2, ~4 days)

Goal: the language closure is a checked invariant; the platform registers into it, never the reverse.

1. `internal/diag/closure_test.go` (pattern: `modelreg/leaf_test.go` with its positive control): closure of `{pipeline, eval, effects, builtins, format, repl, prompt, lsp, vm, gen/golang, smt}` must not contain `ai, telemetry(impl), secrets, mcp_client, coordinator, observatory, storage/*, executor, eval_harness, messaging` nor third-party roots `{sqlite3, otel, grpc, cloud.google.com, ollama}`. Lands day one with an **expected-violations list** that must only shrink (the test fails if a listed violation disappears without being removed from the list, and if a new one appears).
2. Hot path: remove `observatory.CheckHealth` and `checkStaleBinary` from `main.go:75,100`; run them only for ops commands and `doctor`. `ailang version` must open no database.
3. Registration seams — **amended 2026-09-15 after measuring each leak's carrier** (per-package `go list -deps`; the original list was inferred from direct imports):
   - `internal/telemetry` is the one carrier of every heavy root. It imports the OTel SDK, the OTLP and GCP exporters (→ grpc, `cloud.google.com/go/{trace,auth,compute}`) *and* `effects` + `eval`, and every `internal/ai/<provider>` imports it directly. **The seam: `internal/telemetry` keeps the types and a no-op default; `internal/platform/otel` holds the SDK and exporters and registers itself from cmd/ailang's platform init.** One move clears otel/grpc/cloud-trace from ai, effects, pipeline and repl at once.
   - `effects/sharedmem_sqlite.go` (sqlite) and `effects/stream.go` (websocket) → `internal/platform/{sharedmem,stream}` behind `effects.Register…` seams, as planned.
   - **`internal/ai` is part of the language closure, not a leak.** The AI effect is a language feature; `ailang run` of a program that calls a model must link the provider clients. `internal/secrets` (1Password via exec, no cloud deps), `internal/mcp_client` and `internal/auth/gcp` (Vertex ADC for gemini) measured clean and stay too. `effects/ai*.go` therefore does **not** move.
   - The ollama client SDK (`github.com/ollama/ollama/api`, used by `ai/ollama` and `builtins/ollama_embed.go`) is an HTTP client library and is **not** a boundary violation; it leaves the leak-root list. Phase 2.8's single `ai.doJSON` may retire it later on its own merits.
   - Leak roots become specific: `go-sqlite3`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters`, `GoogleCloudPlatform/opentelemetry-operations-go`, `google.golang.org/grpc`, `gorilla/websocket`, `cloud.google.com/go/{firestore,pubsub,storage,trace}`. The OTel *API* (`internal/trace` uses it for span types) is allowed. Platform deny-list becomes `telemetry-impl (platform/otel), coordinator, observatory, storage, executor, eval_harness, messaging`.
4. Pull compiler logic out of cmd: `verify.go` AST→SMT into `internal/smt`; `check_package.go` analyses into `internal/pipeline` (or new `internal/check`); `run_helpers.go` capability resolution + `main_run_exec.go runFile` into `internal/runner`. These are what a WASM or embedded caller needs and cannot reach today.
5. Widen `scripts/check_boundaries.sh` sets as the interim gate until (1) is green: add `ai, secrets, telemetry, mcp_client, storage, executor, eval_harness` to the deny side and `builtins, loader, link, runtime, effects` to the core side.
6. Rewrite ARCHITECTURE.md **from the closure test output** so the map is derived, not aspirational (it currently names ~27 of 124 packages).

#### Phase 2 — One of each (weeks 2–4, ~6 days)

Goal: close the verified-duplicate list, cheapest and most integrity-critical first. Each item is one PR with the survivor named.

| # | Cluster | Survivor | Effort |
|---|---|---|---|
| 2.1 | Leaf packages: `internal/statedir` (one `StateDir()` honouring `AILANG_STATE_DIR` at all ≥8 hard-coded sites), `internal/sqliteopen` (one WAL/busy_timeout/pool/FK recipe replacing 9 openers; `observatory.db` currently opened with two different pool caps), `internal/config` (one `~/.ailang/config.yaml` struct + loader honouring `AILANG_CONFIG`, replacing 3 loaders + 8 re-parses) | new leaves | M |
| 2.2 | Pricing: delete `observatory/pricing.go` second `models.yml` loader and cache 10%/125% hardcode; executor `CostModel` tables in `claude/cost.go`, `codex/cost.go`; `quorum/run.go:179`; `seed.go:378` | `modelreg.CalculateCostForModelWithCache` + aliases moved into `models.yml` | S–M |
| 2.3 | Eval rows: `eval_analysis.BenchmarkResult` embeds `eval_harness.RunMetrics`; one `eval_harness.LoadRows(dirs, opts)` with the validity filter (the `eval_analyzer` loader has none today, so harness errors count as model failures); one `(*RunMetrics).Passed()` per D2 | `eval_harness` | M |
| 2.4 | Project id: `config.CloudProject()` = `AILANG_CLOUD_PROJECT` > `GOOGLE_CLOUD_PROJECT` > yaml > metadata; delete the 12 routes and every hard-coded `"ailang-multivac"` / `"europe-west1"` (D3); remove the `os.Setenv` leak in `coordinator_remote_store.go:83` | `internal/config` | S |
| 2.5 | Plane switch: `AILANG_STORAGE=local\|gcp\|hybrid` + optional `AILANG_STORAGE_{MESSAGING,COORDINATOR,OBSERVATORY}`; delete `AILANG_MESSAGES_STORE`, `AILANG_COORDINATOR_REMOTE`, `AILANG_CHAINS_READ/CLOUD`; derive `COORDINATOR_MODE`; `ailang storage status` prints the resolved value **per store with its source**; old names are hard errors for one release (the CLAUDE.md session-start recipe changes with this) | `storage.NewBackendsForMode` | M |
| 2.6 | The other 13 behaviour-affecting silent fallbacks from the config audit (provider `claude`, model `haiku`, workspace `default`, embedder model, gemini location, loader stdlib `.`, `AILANG_BIN`→PATH, approval timeout, budget 0→unlimited, secret approver no-URL, daemon_github repo, `hybrid` secret approver) → typed errors, following the `evalMaxRSS` pattern | — | S |
| 2.7 | Prompt loaders ×4 → `internal/prompt` parameterised by subdir with a frozen-check option | `internal/prompt` | S |
| 2.8 | AI client: one `ai.NewProviderFor` (5 factories, `exec.go` pins `127.0.0.1:11434`), one `ai.ClassifyError` (3 retryable matchers; `configdriven` never maps 429), `openrouter` reuses openai request types, one `ai.doJSON` | `internal/ai` | M |
| 2.9 | HTTP: one `httpjson.Write`; one WebSocket hub; `daemon_http.go:19` auth constant-time and **fail-closed** when unset; resolve the `/api/messages` route collision | `internal/server` | S |
| 2.10 | Process-group kill ×3 byte-equivalent → `internal/proctree` (motoko, eval_harness) | `proctree` | S |
| 2.11 | Small helpers: `truncate` ×11 (three semantics), `fileExists` ×4, `getEnv` ×3, map readers ×3 (`dashboard.go:87` reads Firestore int64 as 0), Pub/Sub client ×8 with inconsistent project precedence → `internal/config` + one `strutil` | — | S |
| 2.12 | Observatory: one span-name classifier (3), one tree builder (4), `importer_motoko.go` reuses `executor/motoko/parser.go` event decoder (importer drops cache-token fields today) | `observatory` | S |
| 2.13 | SimHash/Hamming/cosine maths ×3/×4/×3 → one `internal/simhash` (hash spaces are currently incomparable; a prior silent-empty-search came from exactly this) | new leaf | S |
| 2.14 | `forbidigo` rule: `os.Getenv`/`LookupEnv` outside `internal/config` (and `DEBUG_*` compiler knobs) fails lint; `docs/docs/reference/env-vars.md` generated from the config package by `make docs-env` | lint + generator | S |

**Phase 2 status, 2026-09-15 (Sprints 3 + 4):** every row 2.1–2.14 is closed. Measured on the merged tree: backend switches 6 → 1; `os.Getenv` outside `internal/config` 391 → 0 with `forbidigo` enforcing it; env vars documented 11% → 100% (generated from the registry, 222 names, plus the `DEBUG_*` knobs in the debugging guide); pricing sources 4 → 1; result loaders 5 → 1; pass predicate 2 formulas → 1 (D2, D9); prompt loaders 4 → 1; AI factories 5 → 1; `writeJSON` 7 → 1; process-group copies 4 → 1; SimHash 3 → 1 (persisted hashes proven bit-identical; coordinator fingerprints re-indexed inside the dedup window); duplicate symbol names 32 → 14. Found and fixed on the way: every prod OpenRouter span banked with a null cost; a 429 from a config-driven provider never classified as rate-limit; ai-check failing cross-module verification that `verify` passed; the daemon's auto-handoff thread trail writing nothing since it landed; the lint config's v1 exclusion keys silently ignored under v2.

Deferred to Future Work (L-sized, and two sit exactly where motoko extensions plug in): `executor.Supervise` unifying the four supervisor loops; the two `effects` vector stores; approval-model merge (`ApprovalRequest`/`Record`/`messaging.Approval`); `slog` migration; single-implementation interface removal.

#### Phase 3 — CLI surface (weeks 4–5, ~5 days)

Goal: an agent can read `ailang --help` in one screen and every command answers `--help`.

1. Dispatch table in `cmd/ailang/commands.go`: `{Name, Aliases, Group, Hidden, Run, Summary}`; `help.go`'s hand-written 500 lines become generated; unknown command prints one line, not 16 KB; `--help` honoured at every level (today 8 groups reject it and `daemon` with no args starts the daemon); `--json` is the one output-format flag; `--dry-run` semantics fixed to "default acts".
2. Groups per the CLI audit: visible top level = `run check fmt test repl iface verify prompt docs examples lsp init serve version help pkg eval`; `check --verify --format agent` absorbs `ai-check` (alias kept; its exit-code test and the DP7 gate depend on it); `eval` becomes a group (`run suite analyze report paired elo publish chains browser-profile`) with the 15 `eval-*` names as aliases for one release; `pkg` gains `docs info versions …`; `dev` (hidden) takes `disasm compile replay export-training builtins doctor axioms select-best ast-edit dump-iface sandbox-check policy-check devtools-prompt agent-prompt editor pi micro-rag cache` (this list said `prompt-freeze`, which is **not a CLI command** — it is `prompt freeze`, a subcommand of `prompt`, invoked by the make target `check-prompt-freeze`; corrected in S5 M6 after `ailang prompt-freeze` was measured against the binary and rejected); `ops` (hidden) takes `messages coordinator mission chains models exec daemon server budget workspaces design-review design-quorum mcp` **with every current top-level name kept as an alias** (D1).
3. `trace` + `observatory` + `dashboard` + `eval-chains` fold into `chains` (the one everyone uses: 55 ops refs vs 2 for `dashboard`); ~5,000 LOC deleted after confirming the web dashboard uses `server` HTTP, not these CLI paths.
4. Flag normalisation for the four families that matter to agents: output format (`--json`), model (`--model`; `--models` accepted as alias on `eval suite`), output location (`--output`/`-o`), mutation gating (`--dry-run`). The other six families are Future Work.
5. Removals per D7, each PR citing the audit's last-commit date and reference counts.
6. Generated `docs/docs/reference/cli.md` from the dispatch table (there is no CLI reference page today; `guides/cli.md` is an 11-line orphan). Fix the docs-cited commands the binary rejects. **The three named here — `eval-chains`, `daemon`, `disk` — were all measured against the post-M5 binary in S5 M6 and none of them is a defect:** `ailang eval-chains` and `ailang daemon` both exit 0 (M3 kept `eval-chains` as an alias into `chains`; `daemon` was never removed), and `ailang disk` appears only in `design_docs/planned/v0_36_0/m-state-disk-retention.md`, which *proposes* the command — a design doc for unbuilt work is not a stale citation. The citation survey that replaced this list, run with the safe probe from `tools/check_prompt_commands.sh` over `docs/docs`, `docs/internal`, the changelogs, `README.md` and `CLAUDE.md`, found exactly one live page teaching a route the binary rejects: `ailang models list` in `docs/docs/guides/mission-role-dispatch.md` (the registry has `role`, `publish`, `source`; `list` has never existed). The rest of the survey's hits are prose ("the `ailang` binary") or changelog entries that correctly *record* a removal.
7. Alias sweep: rename callers in Makefile, `make/*.mk`, `tools/`, `tools/launchd/*.sh`, `.claude/skills/**`, `.github/workflows`, docs — in a **separate** PR after the aliases are live, verified by `make test-launchd-drivers` and a grep that the old spellings are gone.

#### Phase 3b — Physical split, gated (before v1.0.0 only if every precondition holds)

Goal: `cmd/ailang` links the language closure only; `cmd/ailang-ops` carries the platform. **Zero caller disruption** is the design constraint, not a hope:

- `ailang` keeps accepting every ops command and alias. Its `ops` group is a thin delegator: it execs `ailang-ops` found **beside its own executable** first, then on `PATH`, forwarding args, env, stdin/stdout/stderr and the exit code. If neither is found it fails with one line naming the install step. So `ailang messages list` in a launchd plist, a skill or a script behaves identically before and after.
- Every build path ships both binaries in the same change: `make install` / `quick-install` (both `go install`s + both symlinks), `.github/workflows/build.yml` release matrix (both per OS), `cloudbuild-*.yaml` (the buildpack's single `GOOGLE_BUILDABLE` becomes a Dockerfile that builds both, or the coordinator image builds `ailang-ops` only and `ailang` is not needed there — decide by reading what the Cloud Run service actually invokes), `docker/Dockerfile.agent-base` (the containers run `ailang execute-job`, an ops command, so they need `ailang-ops`), CI's `go install ./cmd/ailang` steps.
- `checkStaleBinary` and the version/commit stamp apply to both; `ailang version` reports the ops binary's version too when it is present, so a mismatched pair is visible.

**Preconditions, all required** (the closure test is the instrument for the first two):
1. `internal/diag/closure_expected_violations.txt` is empty — the language binary is genuinely small; splitting earlier ships a second 100 MB binary and proves nothing.
2. Phase 3's aliases have shipped in a tagged release and one attended iteration of each mission loop has run on that release.
3. All five build paths build and install both binaries in CI; the release workflow's artifact list shows both for every OS.
4. One dev Cloud Run deploy and one agent-container job (`execute-job`) have run on the split images.
5. `make simplicity-metrics` shows `binary_internal_packages` for `cmd/ailang` at or below the closure count.

If any precondition is not met at the v1.0.0 cut, v1.0.0 ships one binary with groups and the split is the first v1.1 item. That is the ruling: two binaries, only when certain.

#### Phase 4 — Navigation layer (weeks 5–6, ~3 days)

Goal: the repo explains itself to an agent in ≤25 KB.

1. `doc.go` per `internal/` package with a three-line contract (what it is / what it is not / which confusable sibling to use instead); fix the 5 wrong package comments; gate with `make check-package-docs`.
2. D4 renames: `internal/eval` → `internal/interp`; merge `eval_analysis` + `eval_analyzer`; fold `projecteval`/`bestof` under `eval_harness/`. `gofmt -r`/`gopls rename` only; `make test-imports` and `make verify-examples` after each step (coding-standards import-system rule).
3. One skill tree (D5): `.agents/skills` generated + CI diff; merge `eval-analyzer` + `eval-gap-finder` + `benchmark-manager` into one; retire `benchmark-runner` (external bench, two dead script refs); delete the empty `microrag/` dir; fix the ~20 dangling `ailang <cmd>`/script citations; extend `make check-referenced-paths` to scan SKILL.md citations so this class cannot recur.
4. Instruction surface diet: AGENTS.md becomes a 15-line pointer that carries the `AILANG_MESSAGES_STORE=gcp` export and stops contradicting CLAUDE.md on parked work and skill location; MOTOKO.md drops the two foreign make targets and gets the DST-refactor update; MEMORY.md incident hooks move to topic files behind a 20-line index (protocol already exists in memory); `make help` becomes a curated 30-line top with `make help-all`; `coding-standards.md` gets the trivial-fix lane so it stops contradicting AGENTS.md.
5. `design_docs` triage with a status-driven rule (the archive lane from 2026-07-29 exists, it was never re-run): move the ~14 LANDED/PARKED planned docs and their sprint-plan twins; relocate `implemented/v1_0_0` (13) and `v1_1_0` (3) to their real versions or back to `planned/`; fix `design_docs/README.md`; move the 40 loose mission files (32k lines of logs) under `missions/<name>/` — this one needs its own design doc because `mission-control.sh`, `mission_decisions.sh --check` and the Stop hook read those paths.
6. Test tiering: `testing.Short()` / build tags on the integration-shaped packages (observatory, executor/*, ai/ollama, browser) so `make test-quick` exists; the 53 s compile floor is the target of a follow-up.

#### Phase 5 — Release gate and docs collapse (week 7, ~2 days, overlaps release)

1. `make simplicity-metrics` diff against the Phase 0 baseline goes into the v1.0.0 changelog entry; every Goals-table gate must be green or explicitly waived by the owner.
2. Docs site cluster merges **after** the release so `docs-sync` sees a stable CLI: coordinator ×3 → 1, semantic-cache/RAG ×5 → 2, three-camps ×3 → 1, agent-integration ×5 → 2.
3. PROGRAM.md §8 "Where we are now" updated (last written 2026-06-28) with the frozen-core boundary now being a test.

### Files to Modify/Create

Phase 0
- `tools/simplicity_metrics.sh` — new, ~150 LOC; `make/code-health.mk` target `simplicity-metrics`
- `internal/executor/opencode/opencode.go` — adopt `proctree`, remove 7 bare kills (~40 LOC delta)
- `internal/server/handlers_coordinator.go` — approve route through `ProcessApprovalRequest` (~20)
- `internal/coordinator/daemon_approval.go` — delete HandleApproval family (−500)
- `internal/eval_harness/agent_runner.go`, `internal/eval_harness/agent_runner_streaming.go` — delete headless spawner (−830)
- `internal/observatory/backend_jaeger.go`, `internal/observatory/backend_gcp*.go`, `internal/observatory/backend_composite.go` — delete (−1,900)
- `internal/cognition/`, `testutil/`, `benchmark/`, `output/` — delete
- `.gitignore`, `Makefile` — build outputs to `bin/`, ignore `gen/` and `.eval_workspace/`
- `make/test.mk` — `test-core`

Phase 1
- `internal/diag/closure_test.go` — new, ~120 LOC + `expected_violations.txt`
- `cmd/ailang/main.go` — remove `CheckHealth`/`checkStaleBinary` from the hot path
- `internal/platform/aieffects/`, `internal/platform/sharedmem/`, `internal/platform/stream/` — new homes for `effects/ai*.go`, `sharedmem_sqlite.go`, `stream.go` (moved, ~2,500 LOC)
- `internal/effects/registry.go` — `RegisterAIHandler` etc. (~60)
- `internal/telemetry/` — split into `internal/telemetry` (interface, no-op) and `internal/platform/otel` (impl)
- `internal/builtins/ollama_embed.go` — move to platform
- `internal/smt/`, `internal/pipeline/`, `internal/runner/` — receive `verify.go` sort translation, `check_package.go` analyses, `run_helpers.go`+`main_run_exec.go runFile` (~2,000 LOC moved)
- `scripts/check_boundaries.sh` — widened sets
- `ARCHITECTURE.md` — regenerated

Phase 2
- `internal/config/` — new leaf, ~400 LOC (`CloudProject`, `StateDir`, `StorageMode`, yaml struct, precedence flag > env > file > default)
- `internal/statedir/`, `internal/sqliteopen/`, `internal/simhash/` — new leaves, ~100 LOC each
- `internal/observatory/pricing.go` — delete; `internal/modelreg/models.yml` gains aliases
- `internal/executor/claude/cost.go`, `internal/executor/codex/cost.go` — delete tables
- `internal/eval_analysis/types.go`, `internal/eval_analysis/loader.go`, `internal/eval_analyzer/analyzer.go`, `internal/eval_harness/cost_tally.go`, `internal/eval_harness/rotation_summary.go`, `cmd/ailang/eval_skip_existing.go` — one loader, one `Passed()`
- `internal/storage/backend.go`, `cmd/ailang/storage.go`, `cmd/ailang/messages.go`, `cmd/ailang/coordinator_remote_store.go`, `cmd/ailang/chains_read_backend.go`, `cmd/ailang/chains_post.go`, `cmd/ailang/daemon.go` — one plane switch
- `.golangci.yml` — `forbidigo` on `os.Getenv`
- `docs/docs/reference/env-vars.md` — generated

Phase 3
- `cmd/ailang/commands.go` — new dispatch table (~400 LOC); `cmd/ailang/main.go` switch → table; `cmd/ailang/help.go` → generated
- `cmd/ailang/trace*.go`, `cmd/ailang/observatory*.go`, `cmd/ailang/dashboard*.go`, `cmd/ailang/eval_chains.go` — fold into `chains` (−5,000)
- `docs/docs/reference/cli.md` — generated; `docs/sidebars.js`

Phase 4
- `internal/*/doc.go` — 19 new + 5 fixed; `scripts/check_package_docs.sh`
- `internal/eval/` → `internal/interp/`; `internal/eval_analysis/` ← `internal/eval_analyzer/`
- `.agents/skills/` — generated; `scripts/check_skill_tree.sh`; `scripts/check_referenced_paths.sh` extended
- `AGENTS.md`, `MOTOKO.md`, `Makefile` help, `.claude/rules/coding-standards.md`
- `design_docs/planned/*`, `design_docs/implemented/v1_0_0/`, `design_docs/implemented/v1_1_0/`, `design_docs/README.md`

## Examples

### Example 1: what an agent sees before and after

Before (`ailang version` today):
```
$ ailang version
2026/09/14 10:02:11 observatory: startup cleanup …      ← opens the 548 MB collaboration SQLite
2026/09/14 10:02:11 observatory: …
ailang v0.38.7 (7c56fd266)
$ ailang --help | wc -l
268
$ ailang trace --help
Unknown subcommand: --help
```

After:
```
$ ailang version
ailang v1.0.0 (…)
$ ailang --help | wc -l
24
$ ailang ops chains --help        # every level answers --help, exit 0
$ ailang trace list                # still works: alias → ops chains list (deprecation note on stderr)
```

### Example 2: one route to configuration

Before: `AILANG_STORAGE`, `AILANG_MESSAGES_STORE`, `AILANG_COORDINATOR_REMOTE`, `AILANG_CHAINS_READ`, `AILANG_CHAINS_CLOUD`, `COORDINATOR_MODE`, `--remote` — and `ailang storage status` reads only the first.

After:
```
$ AILANG_STORAGE=local AILANG_STORAGE_MESSAGING=gcp ailang storage status
messaging     gcp   (AILANG_STORAGE_MESSAGING)   project ailang-multivac (AILANG_CLOUD_PROJECT)
coordinator   local (AILANG_STORAGE)             ~/.ailang/state/coordinator.db (default)
observatory   local (AILANG_STORAGE)             ~/.ailang/state/observatory.db (default)
$ AILANG_MESSAGES_STORE=gcp ailang messages list
error: AILANG_MESSAGES_STORE was removed in v1.0.0; use AILANG_STORAGE_MESSAGING=gcp
```

## Success Criteria

- [ ] Every row of the Goals table is at or beyond its v1.0.0 gate, as reported by `make simplicity-metrics`
- [ ] `internal/diag/closure_test.go` passes with an empty expected-violations list
- [ ] The two live bugs have regression tests (grandchild dies on opencode timeout; dashboard approve fires the handoff)
- [ ] 12 of the 16 verified-duplicate clusters closed (2.1–2.14 minus the four deferred), each PR naming the survivor
- [ ] Zero `os.Getenv` outside `internal/config` (lint-enforced); env-var reference generated
- [ ] `ailang --help` ≤ 24 lines; every command answers `--help` with exit 0; every old top-level name still resolves via alias
- [ ] `ailang version` and `ailang fmt` open no database and log nothing to stderr
- [ ] One skill tree; `make check-referenced-paths` scans SKILL.md citations and is green
- [ ] Always-on instruction surface ≤ 25 KB (measured by the metrics script)
- [ ] `make test-core` < 15 s; `make test` still green; `make verify-examples` and `make test-imports` green after every rename step
- [ ] `make test-launchd-drivers` green after the alias sweep; all four mission loops complete one iteration on the new binary before v1.0.0 is tagged
- [ ] ARCHITECTURE.md regenerated from the closure test; CHANGELOG entry carries the metrics diff

## Testing Strategy

- **Boundary**: the `go list -deps` closure test is the primary instrument; `check_boundaries.sh` stays as the fast grep gate. Both run in CI.
- **Renames**: `make test-imports` + `make verify-examples` between every step (the import-system rule); `gopls rename`, never sed.
- **Executors**: every change to a supervisor/kill path runs the existing stream fixtures per harness; the opencode fix adds a grandchild-orphan test.
- **Eval semantics (D2)**: run `ailang eval-paired` on one existing results dir before and after the predicate change and record the discordant count in the PR; annotate the OS history boundary.
- **Config**: a table test over `(flag, env, file, default)` precedence per getter; an unset-project test asserting a typed error, not a prod default.
- **CLI**: a generated test that walks the dispatch table and asserts `--help` exits 0 for every command and every alias resolves.
- **Fleet**: after Phase 3, one attended iteration of each mission loop on the new binary before the alias sweep PR merges.

## Deferred Decisions

- Exact `Group` names (`dev`/`ops` vs `tools`/`fleet`) — agent may choose, must be consistent across help and docs.
- Whether `internal/runner` or `internal/pipeline` receives `runFile` — agent decides by which produces the smaller closure.
- Order within Phase 2 after 2.1–2.4 — agent may reorder by conflict avoidance with in-flight PRs.
- `test-quick` tag mechanism (`testing.Short()` vs build tags) — agent may choose; build tags preferred if the compile floor is the bottleneck.
- Where experiment scripts go (`tools/eval/qwen38-*.sh`, `ab_convergence_card.sh`, `compaction_bench.sh`): `experiments/` vs the design doc that used them — agent may choose; none are deleted.

## Non-Goals

- **Language or stdlib changes** — nothing in this program touches syntax, types, effects semantics or `std/`. Those are the v1_0_0 planned docs (effect modes, bytecode parity) and run in their own lane.
- **The motoko core** — PROGRAM.md's frozen core is untouched; the four deferred L-sized consolidations were deferred precisely because they sit on the extension seam.
- **An unconditional binary split** — Phase 3b lands only behind its preconditions (D1 ruling); v1.0.0 never ships a split that any build path or caller would notice.
- **`slog` migration, single-impl interface removal, approval-model merge, vector-store merge** — Future Work.
- **Docs-site content rewrite** — only the cluster merges in Phase 5; the guides' substance is out of scope.

## Timeline

| Week | Phase | Deliverable | Gate |
|---|---|---|---|
| 1 | 0 | Metrics baseline; two bug fixes; dead code gone; root clean; `test-core` | Baseline JSON banked; `make test` green |
| 1–2 | 1 | Closure test with shrinking list; hot path clean; registration seams; compiler logic out of cmd | Closure ≤ 36 pkgs |
| 2–4 | 2 | Leaves (`config`, `statedir`, `sqliteopen`); 12 clusters closed; plane switch; fallbacks fail loud; forbidigo | `os.Getenv` outside config = 0 |
| 4–5 | 3 | Dispatch table; groups + aliases; chains fold; 4 flag families; generated CLI ref; alias sweep | `--help` 100%; loops run one iteration |
| 6–7 | 3b | Physical `ailang` + `ailang-ops` split with delegation, all five build paths | every Phase 3b precondition, else deferred to v1.1 |
| 5–6 | 4 | doc.go × 124; D4 renames; one skill tree; instruction diet; design_docs triage | Surface ≤ 25 KB |
| 7 | 5 | Metrics diff in changelog; docs cluster merges; PROGRAM.md §8 | v1.0.0 tag |

Estimates are already doubled from the audits' per-item figures. Phases 0–3 are sequential; 4 can start once Phase 2's renames are decided; 5 trails the tag.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Alias-less rename breaks launchd drivers, mission-control.sh and skills (286 + 107 + 55 references) | Aliases land first and stay through v1.0.x; the caller sweep is a separate PR gated on `make test-launchd-drivers` and one attended iteration per loop |
| Import-system rename regression (Sept 2025 class) | `gopls rename` only; `make test-imports` + `make verify-examples` between steps; never delete on "unused" — every deletion PR cites `git log -S` |
| D2 shifts published pass rates | Discordant count recorded before/after; OS history annotated with the boundary date; nothing re-banked |
| D3 breaks a plist or Cloud Run env that relied on the prod default | One-release deprecation warning first; the metrics script greps plists for the affected vars and lists them in the PR |
| Concurrent agents on the shared checkout; loops restarting mid-refactor | Every phase in worktrees with small PRs; loops paused via the HOLD protocol only for the alias-sweep window; harness debt cleared attended |
| Registration seams hide a platform dependency the core tests assumed present | `effects/testctx` updated in the same PR; closure test catches any re-import |
| Cloud container clone does not preserve a symlinked skill tree | D5 recommends a generated copy with a CI diff, not a symlink |
| Quorum: D2 fires trigger 3 (banked-data semantics) | Run `ailang design-quorum` on this doc before Phase 2 starts; Phases 0–1 do not wait on it |

## Related Documents

- [M-MODEL-REGISTRY-SINGLE-SOURCE](v0_35_0/m-model-registry-single-source.md) — declared `modelreg` the single pricing source; `observatory/pricing.go` was never migrated (§2.2 finishes it). Still filed under `planned/` although shipped — itself an instance of Phase 4.5
- [M-MESSAGE-PLANE-FAIL-LOUD](../implemented/v0_35_0/m-message-plane-fail-loud.md) — the same "remaining silent seams" pattern, now applied to config (§2.6)
- [M-CHECK-STRICT-FALLBACKS](../implemented/v1_0_0/m-check-strict-fallbacks.md) — static detection of default-valued literals; the Go-side counterpart is the `forbidigo` rule (§2.14)
- [design_docs/archive/README.md](../archive/README.md) — the 2026-07-29 triage criteria reused in Phase 4.5
- [m-cachesrc-cognitive-complexity-sprint-plan](v0_35_2/m-cachesrc-cognitive-complexity-sprint-plan.md) — precedent for a pure-refactor sprint with measured gates
- [m-eval-validity-discipline](m-eval-validity-discipline.md) — the validity filter that §2.3 makes the only loader honour
- [PROGRAM.md](../PROGRAM.md) — routing lanes; this program is an AILANG-repo lane and never touches the motoko core
- Memory: "Handoff never fires: ROOT CAUSE 2026-09-07", "motoko fixed-port-8080 zombie", "Verify the SEAM, not the artifact", "Harness debt is cleared ATTENDED" — all four are instances this program closes structurally

## Verification Log

Every load-bearing claim, with the command that produced it (all on dev @ 7c56fd266, 2026-09-14).

| Claim | Evidence |
|---|---|
| 124 `internal/` packages, 370k non-test LOC; cmd/ailang 63,270 LOC, 229 non-test files | `go list ./internal/...`; `find … ! -name '*_test.go' \| xargs cat \| wc -l` |
| Language-only closure = 43 packages, 34 without leaks; full binary links 112 | `go list -deps` over `{pipeline,eval,effects,builtins,format,repl,prompt,loader,link,lsp,vm,gen/golang,smt}` and `./cmd/ailang` |
| Binary 100,073,586 bytes; warm build 7.3 s; cold 31.6 s; test compile floor 52.8 s | `ls -la ailang`; `time go build -o /dev/null ./cmd/ailang`; `go test -run XXX_NONE ./...` |
| `main.go` runs `observatory.CheckHealth` and `checkStaleBinary` on every invocation | `grep -n 'CheckHealth\|checkStaleBinary' cmd/ailang/main.go` → lines 75, 100 |
| Dispatch is one hand-written switch with 89 `case` labels; no `Hidden` attribute exists | `grep -c 'case "' cmd/ailang/main.go` → 89; `grep -rn 'Hidden\b' cmd/ailang/main.go cmd/ailang/help.go` → empty |
| **Negative**: no `internal/config`, `internal/statedir`, `internal/sqliteopen` package exists | `ls -d internal/config internal/statedir internal/sqliteopen` → all "No such file" |
| **Negative**: no `test-core`/`test-quick` make target | `grep -rn 'test-core\|test-quick' Makefile make/*.mk` → empty |
| **Negative**: no `forbidigo` rule in lint config | `grep -c forbidigo .golangci.yml` → 0 |
| **Negative**: no CLI reference page; `docs/docs/guides/cli.md` is an 11-line orphan not in `sidebars.js` | read of the file and `grep cli docs/sidebars.js` |
| opencode executor: 7 bare `Process.Kill`, 0 `SysProcAttr` | `grep -c 'Process.Kill' internal/executor/opencode/opencode.go` → 7; `grep -c SysProcAttr internal/executor/opencode/*.go` → 0 in every file |
| Dashboard approve bypasses `ProcessApprovalRequest` | `sed -n '165,175p' internal/server/handlers_coordinator.go` shows `ResolveApprovalRequest(ctx, id, status, "dashboard-user")` only |
| `Daemon.HandleApproval` family, `runHeadlessSession`, Jaeger backend, `internal/cognition` have zero non-test callers/importers | `grep -rn 'HandleApproval(' --include='*.go' internal cmd \| grep -v _test \| grep -v 'func (d \*Daemon)'` → 0; same pattern for `runHeadlessSession(` → 0; `NewJaegerBackend\|JaegerBackend{` outside its file → 0; importers of `internal/cognition` → 0 |
| `observatory/pricing.go` loads a second `models.yml` by relative path search | `grep -n 'models.yml\|go/src' internal/observatory/pricing.go` → lines 14–28 (search-path list incl. `internal/modelreg/models.yml`, `../internal/…`) |
| ~225 distinct env vars, 445 literal read sites, 41 multi-package, 24/211 documented | `grep -rhoE 'os\.(Getenv\|LookupEnv)\("[A-Z0-9_]+"' --include='*.go' internal cmd` piped through `sort \| uniq -c`; documented set = grep of CLAUDE.md, `.claude/rules/*.md`, `docs/docs/guides/debugging.md`, README |
| Six backend switches; `ailang storage status` reads only `AILANG_STORAGE` | readers: `internal/storage/backend.go:60`, `cmd/ailang/messages.go:138`, `coordinator_remote_store.go:49`, `chains_read_backend.go:25`, `chains_post.go:177`, `COORDINATOR_MODE` ×10; `cmd/ailang/storage.go:190` |
| 17 behaviour-affecting silent fallbacks with prod defaults | file:line list in the config audit, e.g. `cmd/ailang/coordinator_config.go:177`, `coordinator_config_roll.go:61,65,69`, `coordinator_agent_check.go:320`, `internal/coordinator/daemon_github.go:125,135`, `daemon_tasks_budget.go:42-48` |
| `.agents/skills` is a 249-file tracked copy diverged in 50 files | `git ls-files .agents/skills \| wc -l` → 249; `diff -rq .agents/skills .claude/skills` → 50 differing |
| `eval_results/` has 17,410 tracked files | `git ls-files eval_results \| wc -l` → 17410 |
| 19 packages without a package comment; 5 wrong | per-package `grep -L '^// Package'` over non-test files; wrong ones read directly (`internal/eval` opens with "Package eval/builtins provides the built-in function registry") |
| 78 top-level commands, ~185 subcommands, 439 flag names, 15 absent from help, 8 groups reject `--help` | recursive `./ailang <cmd> --help` walk (depth 3) captured in the CLI audit; `FlagSet` definition census over `cmd/ailang/*.go` |
| Always-on instruction surface ~51 KB | `wc -c` of CLAUDE.md, the two unscoped rules, MEMORY.md, 40 SKILL.md `description:` lines, plus hook output length from `~/.ailang/state/hooks.log` |
| Pass predicate has two formulas that differ in practice | `CompileOk && RuntimeOk && StdoutOk` at 8 sites vs `StdoutOk` at ~45; `internal/eval_harness/repair.go:260` sets `stdoutOk` without gating on runtime in standard mode |
| Design-doc duplicate gate | `create_planned_doc.sh` neural search: best match 0.32 (implemented) / 0.29 (planned) — below the 0.45 warn threshold |

## References

- Audit scratch data (read-only, session-local): `getenv_counts.txt`, `multipkg.txt`, `undocumented.txt`, `fallback_after_getenv.txt`, `help.txt`, `script_refs.txt`, `planned.txt` under the session scratchpad — regenerate with the commands in the Verification Log rather than relying on them
- `scripts/check_boundaries.sh`, `internal/modelreg/leaf_test.go` — the two existing boundary instruments this program generalises
- `.claude/rules/coding-standards.md` — the "never delete on unused" rule every deletion PR must cite

## Future Work

- `executor.Supervise(cmd, opts)` with per-harness event decoders (four supervisor loops → one).
- Approval model merge (`ApprovalRequest` / `ApprovalRequestRecord` / `messaging.Approval` / observatory stage approvals).
- `slog` + `telemetry` as the only two logging mechanisms; `%w` on every wrap.
- Remove the 25 single-implementation interfaces once the frozen-core seam is decided (the four `eval.*Enforcer/Checker/Recorder` seams all resolve to `effects.EffContext`).
- Remaining six flag families (time windows, timeouts, IDs, namespaces, `--version` overloads, snake_case leaks).
- `link`/`linked`, `server`→`hub`, `messaging`/`notify`/`pubsub`/`daemon`/`dispatch` consolidation.
