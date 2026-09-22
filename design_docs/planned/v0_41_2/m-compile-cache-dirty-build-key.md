# M-COMPILE-CACHE-DIRTY-BUILD-KEY — a rebuilt compiler must not be served its predecessor's verdicts

**Status**: Planned
**Target**: v0.41.2
**Priority**: P1. It silently falsifies verification: a fixed compiler reports "No errors" on
programs it rejects, and `ailang run` executes them
**Estimated**: ~2.5h
**Dependencies**: None
**Tracking**: [#1275](https://github.com/sunholo-data/ailang/issues/1275)

## Problem Statement

The module compile cache (M-PERF6) is keyed by `compilerIdentity(version.Commit, cfg)`. Its
comment says this "invalidates cache on every rebuild, so bugfixes to elaboration, type-checking,
or op-lowering take effect without manual cache nukes". **That is only true across commits.**
Every build of a dirty working tree on one commit has the same identity:

- `make build` / `make quick-install` stamp `Commit` with the bare `git rev-parse HEAD`. Only
  `Version` carries `-dirty` (Makefile lines 27–28, 42).
- `go run` / `go test` builds fall back to `debug.ReadBuildInfo` and append a **constant**
  `-dirty`. `version.go`'s comment ("mark the commit as dirty so cache keys change on every edit")
  is false: the suffix does not change per edit.

So a compiler rebuilt after a type-checker fix is served cache entries its predecessor wrote.
**Reproduced deterministically 2026-09-22** (V1). Two binaries share `Commit=REPRO`; "old" has
the M-EQ-DERIVE-CONTAINERS R-D5 field check disabled, and "new" is correct:

| Step | Output |
|---|---|
| `ail-new check fn_field.ail` (cold cache) | `Error: cannot derive Eq for Handler: field 1 …` ✔ |
| `ail-old check` then `ail-new check` (warm) | `✓ No errors found!` ✘ |
| the same with `AILANG_NO_CACHE=1` | `✓ No errors found!` ✘ (`check` ignores the variable, V3) |
| `AILANG_NO_CACHE=1 ail-new run` | `Error: cannot derive Eq …` ✔ |
| `ail-new run` (warm, no env) | prints `unreachable`: **executes a program the compiler rejects** ✘ |

**Who it hits.** Anyone who rebuilds from an uncommitted tree and re-checks: every attended
sprint, and every agent in the shared main checkout (which is routinely dirty). It hit the
M-EQ-DERIVE-CONTAINERS sprint twice: `ailang check` reported "No errors" on `fn_field.ail` after
the fix was in. It is also a plausible mechanism for the reverted 08-29 attempt's false
"probes verified live" claims (not proven; the 08-29 cache state is gone).

**Second defect, same family:** `AILANG_NO_CACHE=1` is honored only by `internal/runner/run.go`.
The other twelve `pipeline.Config` constructions (`check`, `compile`, `verify`, `ai-check`, LSP,
the API server, the test executor…) never read it (V3). The escape hatch you reach for when you
suspect the cache does nothing on `ailang check`.

## Verification Log

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Same-commit rebuilds share cache entries and serve stale verdicts | Two `go build -ldflags "-X …version.Commit=REPRO"` binaries, one with R-D5 disabled; fresh dir; table above | Reproduced. Commands in the sprint transcript 2026-09-22; re-runnable in ~1 min |
| V2 | The identity is commit (+`+release`) only | Read `internal/pipeline/pipeline_module_cache.go` `compilerIdentity` and `cache_key.go` `ModuleCacheKey` | Confirmed: `commit` or `commit+"+release"`; no build fingerprint |
| V3 | `AILANG_NO_CACHE` is read in exactly one caller | `grep -rn "NoCache" --include='*.go' internal cmd` → `config.NoCache()` consumed only at `internal/runner/run.go:220`; the pipeline checks `cfg.NoCache` at `pipeline_module_phases.go:294`; 13 non-test files construct `pipeline.Config{` | Confirmed; the `check` probe in V1 bypasses it |
| V4 | `make` builds drop the dirty marker from `Commit` | Makefile:28 `COMMIT := $(shell git rev-parse HEAD …)`; :27 `VERSION` uses `--dirty` | Confirmed |
| V5 | The ReadBuildInfo fallback's dirty suffix is constant | Read `internal/version/version.go` `case "vcs.modified"` → `Commit + "-dirty"` | Confirmed |
| V6 | Cache location is per source directory | `newCacheRuntime(filepath.Dir(src.Filename), …)`; `cache_store.go`: `<projectDir>/.ailang/cache/compile/` unless `AILANG_CACHE_DIR` | Confirmed. Worktrees have separate caches; the shared checkout shares one |
| V7 | Hashing the executable is too slow per invocation | `shasum -a 256 ~/go/bin/ailang` (100 MB) = 0.20s | Measured. Too slow for every CLI call (CLI tests spawn dozens) |
| V8 | Existing key tests | `internal/pipeline/cache_key_test.go` (`TestModuleCacheKey_DifferentVersion` …), `cache_invalidation_test.go` | Present; none covers "same commit, different build" |

## Goals

**Primary goal:** a cache entry is served only to the exact compiler build that wrote it (or an
identical clean build of the same commit), and `AILANG_NO_CACHE=1` disables the cache for every
command.

**Success metrics:**
- The V1 reproduction prints the rejection at every step
- `AILANG_NO_CACHE=1 ailang check` performs no cache lookups (`--debug-compile` shows none)
- Clean release builds keep today's hit rate (identity unchanged for them)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: The dirty-build fingerprint is the executable's **size + mtime (ns)** from `os.Executable()`, not a content hash | 0.2s per call for a hash (V7); stat is µs and changes on every `go build`/`go install` | agent | design | low |
| D2: If the fingerprint is unavailable (stat fails), **disable the cache** for that process and say so once under `--debug-compile`. Never fall back to the commit-only key | No silent fallback (CLAUDE.md §2) | agent | design | low |
| D3: `NoCache` is decided inside the pipeline (`cfg.NoCache \|\| config.NoCache()`), not by each caller | One read site instead of 13 | agent | design | low |

No design-freeze items: every decision is agent-resolvable with an in-repo premise. **Quorum: no
trigger fires** (no freeze items, no shared-machinery override, no cost/banking schema, no external
premises).

## Solution Design

1. **Dirty detection that works for both build paths.** `version.Dirty() bool`: true when `Version`
   contains `-dirty` (ldflags builds, V4) or `vcs.modified=true` (ReadBuildInfo builds, V5). Fix the
   Makefile so `COMMIT` gets `-dirty` too; `version.Dirty()` must work without it, because old
   installers exist.
2. **Identity.** `compilerIdentity` appends `+build:<size>-<mtimeNs>` of `os.Executable()` when
   `version.Dirty()`. The fingerprint is computed once per process (`sync.Once`). Clean builds are
   unchanged.
3. **Fail loud (D2).** A fingerprint error means `cacheRuntime` is nil for the process, with a
   `[CACHE] disabled: …` line under `--debug-compile`.
4. **NoCache everywhere (D3).** `newPipelineModuleCache` checks `cfg.NoCache || config.NoCache()`.
   `runner/run.go` keeps setting the field (harmless).
5. **Correct the two false comments** (`cache_key.go`, `version.go`).

### Files to Modify/Create

- `internal/version/version.go`: `Dirty()`, fix the comment (~15 LOC)
- `internal/pipeline/pipeline_module_cache.go`: fingerprinted `compilerIdentity`, `sync.Once` (~35 LOC)
- `internal/pipeline/pipeline_module_phases.go`: central NoCache (~3 LOC)
- `internal/pipeline/cache_key.go`: comment (~4 LOC)
- `Makefile`: `COMMIT` dirty suffix (~2 LOC)
- `internal/pipeline/cache_dirty_identity_test.go`: new (~80 LOC)

## Conflict Surface

No parser/type-system change. Shared machinery touched: the cache identity, which every compile
path reads.
- **Must still work:** clean-build hits (`TestCachePipeline_*`), `+release` separation
  (M-DEBUG-SINK-STRUCTURED-LINES), `AILANG_CACHE_DIR` per-task caches (motoko parallel
  execution), the artifact stamp (M-COMPILE-CACHE-UNVERIFIED-ARTIFACTS: it binds hashes to the
  caller-computed key, so a new key simply misses).
- **Deliberately changes:** dirty builds get a cold cache after every rebuild (that is the fix),
  and `ailang check` honors `AILANG_NO_CACHE`.

## Testing Strategy

- Unit: `compilerIdentity` differs for two fingerprints when dirty and is identical when clean;
  a stat failure disables the cache.
- Integration (the V1 reproduction as a test): build two binaries with the same `Commit` and a
  behavior difference, share one project dir, and assert that the second reports the rejection.
  Mutation: drop the fingerprint and watch the test fail.
- `AILANG_NO_CACHE=1 ailang check`: no manifest is written in a fresh dir.

## Success Criteria

- [ ] V1 reproduction is a test and passes; it fails with the fingerprint removed
- [ ] `AILANG_NO_CACHE=1` honored by `check` (test)
- [ ] Clean build identity unchanged (existing cache tests green)
- [ ] Both false comments corrected; CHANGELOG entry
- [ ] `make test-core` green

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A given compiler build always produces its own verdict for a source |
| A2: Replayability | +1 | Reproducing a check result no longer depends on which build ran earlier in that directory |
| A3–A6 | 0 | |
| A7: Machines First | +1 | Agents verifying their own changes get true answers |
| A8–A10 | 0 | |
| A11: Structured Failure | +1 | No fingerprint → cache off and said so, never a silent weaker key |
| A12 | 0 | |

**Net: +4** → move forward. No hard violations.

## Non-Goals

- Path-dependency interface invalidation (`ailang-core-triage/compile-cache-path-dep-invalidation.md`: a different key component)
- Cache-dir sharing between concurrent agents in one checkout (separate question; this fix makes it safe per build)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Dirty-tree rig/eval binaries lose warm-cache speed after each rebuild | Low–Med | Only after a rebuild. Phase 0 measures cold vs warm stdlib compile time |
| mtime granularity collision | Negligible | Nanosecond mtime plus size |

## Related Documents

<!-- Auto-populated by Ollama neural search on "compile cache dirty build key"; duplicate gate passed (max 0.49) -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_35_2/m-compile-cache-unverified-artifacts.md](../../implemented/v0_35_2/m-compile-cache-unverified-artifacts.md) (0.49): the artifact stamp binds to the key; unaffected
- [design_docs/implemented/v0_6_0/DX-15-semantic-caching-MVP.md](../../implemented/v0_6_0/DX-15-semantic-caching-MVP.md) (0.44)

**Planned (check for overlap):**
- [design_docs/planned/ailang-core-triage/compile-cache-path-dep-invalidation.md](../ailang-core-triage/compile-cache-path-dep-invalidation.md) (0.49): distinct key component (dependency interfaces)
- [design_docs/planned/v0_36_0/m-cache-module-id-encoding.md](../v0_36_0/m-cache-module-id-encoding.md) (0.45): distinct (entry naming)
- [design_docs/implemented/v0_41_2/m-eq-derive-containers.md](../../implemented/v0_41_2/m-eq-derive-containers.md): the sprint where this bit

---

**Document created**: 2026-09-22
