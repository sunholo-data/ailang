### Changed
- **Dead code, batch 1 (platform packages): 103 unreachable functions and 2,879 lines deleted.**
  `deadcode -test=false ./cmd/...` falls from 1,649 to 1,536 reported functions (1,301 to 1,188
  outside the vendored `third_party/goopus`); `-test=true ./...` from 615 to 502. Every deleted
  function was unreachable under both modes, across every `main` in the module, with no string,
  registry or template reference and no recent rename in `git log -S`. Packages:
  `internal/observatory` (38 functions; `seed.go`, `outliers.go` and `hierarchy_turns.go` removed
  whole, their callers having gone with the `observatory` subcommands and the dashboard UI routes),
  `internal/server` (11; the never-constructed `AuditLogger` in `audit.go` and the never-mounted
  `RequireWorkspaceAccessMiddleware`), `internal/display` (13), `internal/eval_harness` (13),
  `internal/messaging` (10), `internal/eval_analyzer` (6; `issue_extractor.go` removed whole),
  `internal/eval_analysis` (6) and `internal/coordinator` (6). The language core is untouched.
  Workstream 3 of `design_docs/planned/HANDOVER-v1-codebase-health-2026-11-09.md`.
- **`internal/server/handlers_hooks_test_bench.go` becomes `handlers_hooks_bench_test.go`.** Its
  name did not end in `_test.go`, so its six hook-event tests and four benchmarks never ran and the
  production server package compiled `testing` and `httptest` in. Renamed with no content change;
  the six tests pass.
