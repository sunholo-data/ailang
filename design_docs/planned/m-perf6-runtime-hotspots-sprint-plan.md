# M-PERF6 Sprint Plan: Runtime Performance Hotspots

**Sprint ID**: M-PERF6
**Design Doc**: [m-perf6-runtime-hotspots.md](m-perf6-runtime-hotspots.md)
**Duration**: 1 day (3 milestones)
**Risk Level**: Low
**Total LOC Estimate**: ~200

## Velocity Context

Recent sprints:
- M-INCREMENTAL-TYPECHECK: 870 LOC estimated, completed in 1 session (3 milestones)
- M-PERF-DOCPARSE: 430 LOC, completed in 1 session (3 milestones)
- M-PERF-GOROUTINE-ID: 300 LOC, completed in 1 session (3 milestones)

Current velocity: ~150-200 LOC/hour for performance optimization work.

## Baseline (Warm Cache)

| File | Current | Target |
|------|---------|--------|
| Alice EPUB (185KB) | 2.97s | <2.0s |
| Moby Dick EPUB (797KB) | 7.27s | <5.0s |
| 10MB DOCX | 2.18s | <1.5s |

## Milestones

### M1: CoreTypeInfo serialization → gob (~80 LOC, ~1 hour)

**Why first:** 15% of CPU is `LoadArtifacts` → JSON unmarshal. Gob is 3-5x faster and the
pattern is already proven in `core/gob.go`. Lowest risk, highest certainty of impact.

**Tasks:**
1. Add gob registration for all 14 Type + 4 Kind implementations in new `internal/types/gob.go`
2. Modify `cache_store.go`: `StoreArtifacts` writes `coretypeinfo.gob` instead of `.json`
3. Modify `cache_store.go`: `LoadArtifacts` reads gob, falls back to JSON for migration
4. Add round-trip gob test for CoreTypeInfo (extend existing JSON tests)
5. Bump cache format version to invalidate old entries
6. Run benchmarks: Alice EPUB warm cache

**Acceptance Criteria:**
- CoreTypeInfo stored as gob, not JSON
- Old JSON caches gracefully invalidated (cache miss, recompile)
- Gob round-trip test passes for all Type variants
- `make test` passes
- Benchmark shows measurable improvement in LoadArtifacts time

### M2: Output buffering for println (~30 LOC, ~30 min)

**Why second:** 11% of CPU is `.String()` calls from println output. While we can't avoid
`.String()`, we can buffer stdout to reduce syscall overhead and improve throughput.

**Tasks:**
1. Add buffered writer wrapping `os.Stdout` in eval initialization
2. Ensure flush at program exit
3. Verify output is identical (no dropped output)
4. Run benchmarks

**Acceptance Criteria:**
- println uses buffered writer
- Buffer is flushed before program exit
- Output matches unbuffered (diff test)
- `make test` passes
- `make verify-examples` passes

### M3: Allocation profiling + targeted fixes (~90 LOC, ~1.5 hours)

**Why third:** 44% of CPU is memory management but requires profiling to identify which
allocations to target. Higher uncertainty, but highest potential gain.

**Tasks:**
1. Run allocation profile (`-memprofile`) on Alice EPUB
2. Identify top 5 allocation sites by object count
3. Apply targeted fixes (pre-allocation, capacity hints, pointer passing)
4. Run CPU profile to verify memory management % decreased
5. Final benchmark comparison (all 3 files, cold+warm)
6. Update CHANGELOG with results

**Acceptance Criteria:**
- Allocation profile captured and analyzed
- At least 2 targeted allocation reductions implemented
- CPU profile shows memory management < 30% (from 44%)
- Alice EPUB warm < 2.0s
- Moby Dick EPUB warm < 5.0s
- CHANGELOG updated with final benchmarks
- `make test` and `make verify-examples` pass

## Success Metrics

- [ ] Alice EPUB warm < 2.0s (from 2.97s, target 33% improvement)
- [ ] Moby Dick EPUB warm < 5.0s (from 7.27s, target 31% improvement)
- [ ] Memory management CPU < 30% (from 44%)
- [ ] No test regressions
- [ ] CHANGELOG updated

## Dependencies

- M-INCREMENTAL-TYPECHECK M2 (`cache_store.go`) — already committed
- Existing gob pattern in `core/gob.go` — proven, reuse

---

**Created**: 2026-04-10
