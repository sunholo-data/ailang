# Sprint Plan: M-V1-SIMPLIFY-S2 — Fence the core for real, and one config leaf

**Design doc**: [m-v1-simplification-program.md](m-v1-simplification-program.md) (Phase 1.3–1.6 as amended 2026-09-15, plus Phase 2.1's `config`/`statedir` leaves and 2.4's project resolver under the D3 ruling)
**Sprint ID**: M-V1-SIMPLIFY-S2
**Created**: 2026-09-15
**Rulings applied**: D1 (split only when certain; this sprint delivers precondition 1, the empty violations list), D3 (prod defaults warn now, error in v1.0.0)

## Summary

Sprint 1 made the language-core boundary a test with seven listed leaks. This sprint clears all seven, so `internal/diag/closure_expected_violations.txt` is empty — the first precondition of the two-binary split — and lands the first configuration leaf so Phase 2 can start collapsing routes. Disruption is kept low by construction: every seam is a registration that `cmd/ailang` performs at init, so the one binary behaves exactly as before; the prod-default removal is a warning, not an error, until v1.0.0.

**Duration:** 4 days (5 milestones; M1 and M3 sequential, M2 and M4 parallel)
**Dependencies:** Sprint 1 (closure test, metrics). No Design Freeze item blocks this sprint.
**Risk Level:** Medium — M1 rewires every AI provider's span setup; M4 touches 36 project-id reads and 34 state-dir sites. Both are mechanical and test-covered, and the closure test plus `make test` gate every merge.

## Proposed Milestones

### Milestone 1: Telemetry seam — `internal/telemetry` API-only, `internal/platform/otel` owns the SDK
**Goal:** The one heavy carrier is cut. `internal/telemetry` keeps what the core calls (`StartSpan`, `Tracer`, `Truncate`, `CategorizeError`, `LineSnippet`, `NewEffectSpanWrapper`, `RecordSpan`, trace-context propagation, enable flags) on the OTel **API** only; `Init*`, `InitializationStatus`, `ExporterState`, `NewResource`, `GoogleCloudProject`, `IsGoogleCloudEnabled`, `IsDualExportEnabled`, `otel_init.go`, `resource.go` move to `internal/platform/otel`, which imports the SDK and the OTLP/GCP exporters and is called only from `cmd/ailang` and `internal/server`.
**Estimated:** ~400 LOC moved, ~60 changed at 13 call sites
**Acceptance:**
- [ ] `go list -deps ./internal/telemetry` has no `go.opentelemetry.io/otel/sdk`, `/exporters`, GCP exporter, grpc or `cloud.google.com/go/trace`
- [ ] With nothing registered, `telemetry.StartSpan` is the OTel no-op (the API's default global provider); with `platform/otel` initialised from cmd, spans export as before (existing `TestOTLP*` / `TestGoogleCloudTrace_Integration` move with the code and still pass)
- [ ] The five telemetry entries leave `closure_expected_violations.txt`; closure test green

### Milestone 2: `effects` seams — sqlite shared cache and websocket stream behind registration
**Goal:** `effects/sharedmem_sqlite.go` → `internal/platform/sharedmem`; `effects/stream.go`'s websocket transport → `internal/platform/stream`. `effects` exposes `RegisterSharedCacheOpener(func(path string, opts ...CacheOption) (SharedCache, error))` and `RegisterStreamTransport(scheme string, …)`; with nothing registered the brain/stream effects return a typed error naming the missing registration. `cmd/ailang` registers both at init. Tests that need sqlite import the platform package from `_test.go` files (test imports do not count in the closure).
**Estimated:** ~1,560 LOC moved, ~120 new
**Acceptance:**
- [ ] `go list -deps ./internal/effects` has no `go-sqlite3` or `gorilla/websocket`
- [ ] `ailang run` of an example that uses brain/shared memory behaves as before (registration from cmd)
- [ ] Unregistered path returns `effects.ErrBackendNotRegistered` (typed), tested
- [ ] Two entries leave the violations list

### Milestone 3: Compiler logic out of `cmd/ailang` (after M1 merges)
**Goal:** `verify.go`'s AST→SMT sort translation → `internal/smt`; `check_package.go`'s strict-fallback / unresolved-var analyses → new `internal/check`; `run_helpers.go` capability resolution + `main_run_exec.go runFile` → new `internal/runner`. The cmd files become flag parsing + one call each.
**Estimated:** ~2,000 LOC moved
**Acceptance:**
- [ ] `cmd/ailang/{verify,check_package,run_helpers,main_run_exec}.go` each under 250 LOC
- [ ] `ailang check`, `ailang verify`, `ailang run` byte-identical output on `examples/` (`make verify-examples`) and the `ai_check_exit_test.go` contract green
- [ ] `internal/check`, `internal/runner` have a `doc.go`

### Milestone 4: `internal/config` + `internal/statedir` leaves; project resolver under D3
**Goal:** Two leaf packages (stdlib + yaml only, enforced by a leaf test like `modelreg/leaf_test.go`). `config.CloudProject(ctx)` = `AILANG_CLOUD_PROJECT` > `GOOGLE_CLOUD_PROJECT` > `~/.ailang/config.yaml` `pubsub.project_id` > GCE metadata; every `os.Getenv("AILANG_CLOUD_PROJECT"|"GOOGLE_CLOUD_PROJECT"|"GCP_PROJECT")` (36 sites) becomes a call. `statedir.Dir()` honours `AILANG_STATE_DIR` and replaces the 34 `filepath.Join(home, ".ailang", "state")` sites. The 9 hard-coded `"ailang-multivac"` / `"europe-west1"` defaults go through `config.DeprecatedDefault(name, value)`, which returns the value **and prints one stderr warning per process** naming the env var to set and that v1.0.0 will refuse; `AILANG_STRICT_CONFIG=1` makes it an error today (the v1.0.0 behaviour, testable now).
**Estimated:** ~350 new, ~80 sites changed
**Acceptance:**
- [ ] Leaf tests for both packages (positive control included)
- [ ] `getenv_outside_config` in `make simplicity-metrics` drops by ≥ 40
- [ ] `AILANG_STATE_DIR=/x ailang coordinator status` and `ailang chains list` (offline) both resolve under `/x` — the two paths the audit found disagreeing
- [ ] Deprecation warning fires once per process; `AILANG_STRICT_CONFIG=1` errors; both tested

### Milestone 5: Close-out — trim the violations list to empty, regenerate ARCHITECTURE's boundary section, metrics, changelog
**Acceptance:**
- [ ] `closure_expected_violations.txt` has zero entries; closure test green
- [ ] ARCHITECTURE.md's layer section is generated from `go list` (a script under `scripts/`), not hand-written
- [ ] `make simplicity-audit` vs the 2026-09-15 snapshot: `closure_leak_roots` 7 → 0, no gated regression except any the sprint knowingly accepts (named in the changelog)
- [ ] `make test` green; all CI gates green

## Open Questions (not blocking)
- Post-split, does the language binary link `platform/sharedmem` so `ailang run` keeps brain effects? Product call for Phase 3b; this sprint keeps one binary and registers everything.
