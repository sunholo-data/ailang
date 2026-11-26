# Bar G5 — Simplification Phase 5 Metrics Diff (v1.0.0 Release Notes Fragment)

**Program**: [M-V1-SIMPLIFICATION-PROGRAM](../m-v1-simplification-program.md) (Phase 5 = Bar `G5`)
**Baseline**: Phase 0 (`2026-09-14`, commit `a21e5b24b`, [2026-09-14.json](../../.ailang/state/simplicity/2026-09-14.json))
**Intermediate**: Pre-wave snapshot (`2026-11-09`, commit `eedcd9972`, [2026-11-09.json](../../.ailang/state/simplicity/2026-11-09.json))
**Current (pre-merge Slice 3)**: `2026-11-26` @ `5dd83e40b` (`make simplicity-metrics-fast`) — *Note: `integrator` refreshes post-merge numbers with `make simplicity-metrics-fast` / `make simplicity-metrics`.*
**Decision**: **D4 = a** (ratified waivers for Phase 4 optional items, `D6` tracked baselines, and `testdata/` env probes) → **Bar `G5`: 0 open items (`N = 0`)**.

---

## Goals Table Diff (Phase 0 → Current)

| Metric | Phase 0 (`2026-09-14`) | `2026-11-09` (`eedcd9972`) | Current (`2026-11-26`, `5dd83e40b`) | v1.0.0 Gate | Status | Notes / Waiver Rationale |
|---|---:|---:|---:|---:|---|---|
| **Non-leaf** internal pkgs in language closure (`closure_nonleaf_packages`) | 30 *(41 total)* | 33 *(52 total)* | **35** *(54 total)* | ≤ 36 | **PASS** | Coupling boundary enforced by `internal/diag/closure_test.go`. |
| Third-party leak roots in language closure (`closure_leak_roots`) | 5 | 0 | **0** | 0 | **PASS** | `sqlite3`, OTel SDK/exporters, `grpc`, `websocket`, GCP client libs eliminated from language closure. |
| Platform packages reachable from language closure (`closure_platform_packages`) | 4 | 0 | **0** | 0 | **PASS** | `platform/*`, `coordinator`, `observatory`, `storage`, `executor`, `eval_harness`, `messaging` isolated behind registration seams. |
| Top-level commands in `ailang --help` (`commands_top_level`) | 89 | 17 | **17** | ≤ 20 | **PASS** | Single dispatch table in `cmd/ailang/commands.go`; `dev`/`ops`/`eval` grouped. |
| Routes where `<cmd> --help` exits 0 (`help_exit0_rate`) | ~70% | 100% | **100%** *(73 routes)* | 100% | **PASS** | Probed across top-level, `dev`, `ops`, and `eval` subcommands. |
| Output-format flag spellings (`output_format_spellings`) | — | 4 | **4** | ≤ 4 | **PASS** | Ratchet held at `{json, format, pretty, stream-json}` pending caller sweep. |
| Which-backend env switches (`backend_switches`) | 6 | 1 | **1** | ≤ 1 | **PASS** | Unified under `AILANG_STORAGE` (+ per-store overrides). |
| Root `sprint_*` / `sprint-*` clutter (`root_sprint_files`) | 0 *(11 pre-Phase 0)* | 0 | **0** | 0 | **PASS** | Clean repo root. |
| `os.Getenv` / `LookupEnv` outside `internal/config` (`getenv_outside_config`) | 388 | 5 | **5** | 0 | **WAIVED (D4)** | **0 in production code** (`forbidigo` clean); all 5 remaining sites are standalone `testdata/` probe binaries (`internal/docsearch/github/testdata/credentialhelper/main.go`, `cmd/ailang/testdata/publish_env_probe/main.go`). |
| Env vars documented (`env_vars_documented_pct`) | 11% *(24/211)* | 98% *(286/290)* | **98%** *(289/293)* | 100% | **WAIVED (D4)** | Generated `docs/docs/reference/env-vars.md` covers 100% of registered `internal/config` vars; the 4 undocumented literals are internal/test-probe env names (`DOCSEARCH_HELPER_LOG`, `PUBLISH_PROBE_*`). |
| `internal/` packages without `// Package` doc (`packages_without_doc`) | 17 | 14 | **14** | 0 | **WAIVED (D4)** | Phase 4 optional item (`m-v1-simplification-program.md`: Phase 4 does not gate `v1.0.0`). |
| Skill trees (`skill_trees`) | 2 | 2 | **2** | 1 | **WAIVED (D4)** | Phase 4 optional item; `.agents/skills` and `.claude/skills` are kept synchronized via `tools/launchd/sync-agents-skills.sh` and gated in CI by `tools/launchd/test_agents_skills_sync.sh`. |
| Always-on instruction surface (`instruction_surface_bytes`) | 26,780 B | 29,394 B | **29,608 B** | ≤ 25,600 B | **WAIVED (D4)** | Phase 4 optional item; reflects skill descriptions added for v1.0 release gates, verification, and eval routing. |
| Tracked files (`tracked_files`) | 24,829 | 30,431 | **31,338** | < 8,000 | **WAIVED (D4 + D6)** | `eval_results/baselines/` is intentionally kept tracked per `D6` (read by docs BenchmarkDashboard and `eval-weekly.yml`). |
| Unreachable funcs (`unreachable_funcs`) | — | — | *null (offline)* | ≤ 1,301 | **UNMEASURED (sandbox)** | Requires `proxy.golang.org` (`deadcode@v0.28.0`); integrator refreshes online if available. |
| `make test-core` wall time (`test_core_seconds`) | n/a | — | *null (`--no-test`)* | < 15 s | **DEFERRED TO INTEGRATOR** | Fast snapshot skips timed `make test-core`; integrator measures post-merge. |

---

## Structural Summary (Informational)

| Measure | Phase 0 (`2026-09-14`) | `2026-11-09` | Current (`2026-11-26`) |
|---|---:|---:|---:|
| `internal_packages` | 123 | 151 | **154** |
| `binary_internal_packages` | 112 | 140 | **141** |
| `internal_loc` (non-test Go) | 297,767 | 367,911 | **378,238** |
| `cmd_ailang_loc` (non-test Go) | 63,296 | 68,312 | **69,542** |
| `prod_loc_core` / `prod_loc_platform` | — | — | **127,161** / **251,077** *(66% platform)* |
| `dup_symbol_names` (≥3 pkgs) | 31 | 22 | **25** |
| `planned_design_docs` | 243 | 363 | **425** |

---

## Ratified D4 Waiver Summary

Per **D4 = a**, all remaining non-zero/above-target rows in the Goals table are explicitly waived for `v1.0.0`:

1. **`testdata/` env probes (`getenv_outside_config = 5`, `env_vars_documented_pct = 98%`)** — zero production violations; all 5 reads live in test fixture binaries (`credentialhelper`, `publish_env_probe`).
2. **Phase 4 optional navigation items (`packages_without_doc = 14`, `skill_trees = 2`, `instruction_surface_bytes = 29,608 B`)** — explicitly marked non-gating for `v1.0.0` in `m-v1-simplification-program.md`; CI already enforces `.agents/skills` parity.
3. **Tracked eval baselines (`tracked_files = 31,338`)** — governed by ratified `D6` (`eval_results/baselines/` remains tracked until dashboard/weekly-eval readers move to bucket storage).
