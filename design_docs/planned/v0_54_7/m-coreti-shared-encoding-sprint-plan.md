# Sprint Plan: M-CORETI-SHARED-ENCODING

**Design doc**: [m-coreti-shared-encoding.md](m-coreti-shared-encoding.md)
**Issue**: #1778 (triage #1779)
**Duration**: 1.5 days (3 milestones, each well inside one executor run)
**Risk**: Low-medium. The codec is new; correctness is pinned by a gob-equivalence oracle
**Registry reuse**: none for every milestone (compiler-internal cache codec; no package capability)
**AILANG prompt version loaded**: n/a. No showcase `.ail` modules; the guard fixture is test-embedded source checked by the pipeline itself.

## Goal

Make CoreTypeInfo blobs scale with distinct type structure: motoko `session.ail` from 188 MB (uncacheable) to under 2 MB (cached).

## Milestones

### M1: DAG codec (~250 LOC impl + ~250 LOC tests)
- `internal/pipeline/cache_coreti_dag.go`: `encodeCoreTIDAG` / `decodeCoreTIDAG`, with a hash-consed node table, post-order refs (index+1, 0 = nil), sorted IDs and keys, and a `Raw` fallback.
- `internal/pipeline/cache_coreti_dag_test.go`: per-variant round trip against the gob round trip; nil and typed-nil; shared-subtree collapse; byte determinism; malformed input errors (no panic); source enumeration of `Substitute` implementors; reflection field-count pin.

**Acceptance**: `go test ./internal/pipeline -run CoreTIDAG` green; 10k-entry nested-record CoreTI encodes to under 64 KB.

### M2: Wire-in, version bump, equivalence corpus, guard (~40 LOC impl + ~150 LOC tests)
- `productionCacheArtifactCodec` uses the DAG codec; `cacheKeyVersion` v7; update the `"v6"` seeds in `cache_invalidation_test.go`, `alias_body_closure_runtime_test.go` and `cmd/ailang/serve_api_mcp_surface_test.go`.
- Equivalence corpus: std modules (and examples) compiled; DAG round trip `reflect.DeepEqual` to gob round trip.
- Guard: two-module nested-record-alias fixture; written blob under budget and ≥10× smaller than plain gob.
- Changelog fragment `changelogs/unreleased/2026-10-12-coreti-shared-encoding.md`.

**Acceptance**: `make test-changed`, `make lint`, `make check-file-sizes`, `make check-boundaries`, `make check-changelog` pass.

### M3: Real-world verification + follow-up (~0 LOC)
- motoko `session.ail`: cold write succeeds, blob size, warm `check` time; recorded in the implementation report.
- File a follow-up issue for the 34 s → 83 s fresh-compile slowdown, with a CPU profile.

**Acceptance**: numbers recorded; follow-up issue linked from #1778.

## Validation

Focused gates only (`make test-changed` + targeted). CI runs the full suite.
