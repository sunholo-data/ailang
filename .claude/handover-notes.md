# M-INCREMENTAL-TYPECHECK Sprint Handover

## Status: M1 + M2 COMMITTED, M3 CODE DONE BUT UNCOMMITTED

### What's Done

**M1: types.Type + Kind JSON serialization** — COMMITTED (7c4ff925)
- All 14 Type implementations + 4 Kind implementations have MarshalJSON/UnmarshalJSON
- Tagged-union pattern: `{"tag": "tcon", "data": {"Name": "int"}}`
- Also serializes Scheme, CoreTypeInfo
- 20 round-trip tests pass
- Files: `internal/types/json.go` (480 LOC), `internal/types/json_test.go` (300 LOC)

**M2: CompileUnit + CoreExpr serialization** — COMMITTED (d96de92f)
- CachedModule struct: Core (gob), CoreTypeInfo (JSON), Iface (JSON), Constructors (JSON)
- All 22 CoreExpr + 7 CorePattern types registered for gob in `internal/core/gob.go`
- StoreArtifacts/LoadArtifacts on CacheStore
- Full Iface reconstruction including Exports (Scheme), Constructors (ConstructorScheme), TypeAliases
- 2 round-trip tests pass with diverse CoreExpr types
- Files: `internal/core/gob.go` (48 LOC), `internal/pipeline/cache_store.go` (+380 LOC), `internal/pipeline/cache_store_test.go` (+200 LOC)

**M3: Pipeline skip logic** — CODE WRITTEN, NOT COMMITTED
- Modified `internal/pipeline/pipeline_module.go`:
  - On cache HIT, tries `cacheStore.LoadArtifacts()` → if success, SKIP compilation
  - Registers cached Iface with modLinker via `RegisterIface()` for downstream imports
  - Falls through to normal compilation on load error
  - After successful compilation, stores artifacts via `StoreArtifacts()`
  - Shows `SKIP` instead of `HIT` in `--debug-compile` output
- **VERIFIED WORKING** with cold/warm test cycle:
  - `arithmetic.ail`: Cold=MISS, Warm=SKIP, output identical ✅
  - `imports_basic.ail`: 2 modules, both SKIP on warm ✅
  - `imported_adt_types.ail`: SKIP on warm ✅
  - Full test suite: 0 failures ✅
  - verify-examples: 159 pass, 2 fail (pre-existing) ✅

### What's NOT Done

1. **Commit M3** — Bash tool hit exit code 126 bug, couldn't run `git add/commit`
2. **Docparse benchmarks** — Couldn't run timing benchmarks due to Bash bug
   - Target: Alice EPUB ≤1.0s (from 1.67s), Moby Dick ≤1.8s (from 2.35s)
   - Need to run from `/Users/mark/dev/sunholo/ailang-parse`:
     ```bash
     rm -rf .ailang/cache/compile
     time ailang run --debug-compile -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub
     # Then repeat for warm cache timing
     ```
3. **CHANGELOG update** — needs docparse benchmark results
4. **Sprint JSON update** — needs `passes: true` for all milestones
5. **Design doc status update** — from "Planned" to "Implemented"

### Files Changed (Uncommitted)

- `internal/pipeline/pipeline_module.go` — Cache hit skip logic + artifact storage after compile

### How to Resume

1. **Commit M3**:
   ```bash
   cd /Users/mark/dev/sunholo/ailang
   git add internal/pipeline/pipeline_module.go
   git commit -m "M-INCREMENTAL-TYPECHECK M3: wire cache hit → skip compilation

   On cache hit, load cached artifacts (Core, CoreTypeInfo, Iface, Constructors)
   and skip elaboration→typecheck→monomorph→lower. Register cached Iface with
   linker for downstream module imports.

   Verified: cold/warm cycle shows MISS→SKIP, 159 examples pass, 0 test failures.

   Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
   ```

2. **Run docparse benchmarks** (from ailang-parse dir):
   ```bash
   cd /Users/mark/dev/sunholo/ailang-parse
   rm -rf .ailang/cache/compile
   time ailang run -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub  # cold
   time ailang run -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_alice.epub  # warm
   time ailang run -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_moby_dick.epub  # cold
   time ailang run -caps IO,FS,Env docparse/main.ail -- data/test_files/gutenberg_moby_dick.epub  # warm
   ```

3. **Update CHANGELOG, sprint JSON, design doc** with results

### Architecture Notes

- **Cache location**: per-project `.ailang/cache/compile/modules/<mod_id>/`
- **Cache format**: `core.gob` (gob-encoded Program), `coretypeinfo.json`, `iface.json`, `constructors.json`
- **Module ID sanitization**: `std/list` → `std__list` for directory names
- **Cache key unchanged**: Still `SHA-256(compiler_version + source_hash + dep_iface_digests)`
- **Invalidation**: Automatic via cache key mismatch (source change or dependency change)
- **Bypass**: `AILANG_NO_CACHE=1` environment variable
- **Fallback**: If LoadArtifacts fails, falls through to normal compilation silently

### Known Issue

The warm cache currently shows "20 hits, 37 misses" on first run of docparse because the old manifest entries (from system ailang) don't have artifact files. After one full cold run with the new binary, all entries will have artifacts and subsequent runs will SKIP.

The `ailang cache compile-clear` command only clears the root project's cache, not caches in subdirectories. Each `.ail` file's cache is in the directory where that file lives.
