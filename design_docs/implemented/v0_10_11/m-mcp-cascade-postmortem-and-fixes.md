# M-MCP-CASCADE: MCP Tool Names → Module Path Collisions → Startup Perf — Postmortem & Forward Plan

**Status**: PLANNED (post-mortem + v0.10.11 fix scope)
**Target**: v0.10.11 (perf fix), v0.11.0 (process changes)
**Priority**: P0 — production cold-start regression on docparse Cloud Run
**Dependencies**: None (this is the cleanup pass for v0.10.7 → v0.10.10)
**Milestone ID**: M-MCP-CASCADE
**Created**: 2026-04-08
**Author**: AILANG maintainer + docparse reporter (msg chain c287d189 → 1210b37d)

---

## Why This Doc Exists

Between 2026-04-07 morning and 2026-04-08 we shipped **four point releases in ~30 hours** (v0.10.7 → v0.10.10) chasing a single bug class reported by docparse. Each fix introduced or unmasked the next bug:

| Release | Intended fix | Actual outcome |
|---------|--------------|----------------|
| v0.10.7 | MCP-compliant tool names + `@mcp_name` | ✅ shipped clean |
| v0.10.8 | serve-api package routes leak via `module_prefix` | ✅ fixed registration; **left dispatch broken** (silent footgun) |
| v0.10.9 | MOD011 collision detector for the dispatch footgun | ❌ false-fired on every `module_prefix` package — blocked all type-check |
| v0.10.10 | Dedupe MOD011 by resolved file path | ✅ correctness fixed; **introduced 4× cold-start regression** |
| v0.10.11 | Fix dep-discovery double-load (this doc) | TBD |

None of v0.10.7-v0.10.10 had a design doc. They were all patch-level "see message → fix → release" loops. **The cascading nature is the cost of skipping the design doc process for an interconnected subsystem.** This doc captures the postmortem so we don't repeat it.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Loader output is already deterministic; this fixes only redundant work |
| A2: Replayability | 0 | No change to traces |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | +1 | MOD011 makes module-path ownership explicit (no silent shadowing) |
| A5: Bounded Verification | +1 | Cold start under Cloud Run probe budget is a verification bound |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | Predictable cold-start latency matters for agent infrastructure |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +2 | Eliminates a 60s+ silent overhead in `serve-api` startup |
| A10: Composability | +1 | Local + `module_prefix` package modules now compose without ambiguity |
| A11: Structured Failure | +1 | MOD011 errors are structured and pinpoint the colliding files |
| A12: System Boundary | +1 | serve-api is a system boundary; startup must be predictable |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: Loader cache is still last-writer-wins for legitimate reloads (hot-reload), so no determinism regression
- [x] A4: MOD011 *is* the authority fix — silent shadowing was an A4 violation
- [x] A9: Current v0.10.10 violates A9 (hidden 4× cost). Fix restores baseline.
- [x] A12: Cloud Run startup probe is a system-boundary contract this regression broke

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Yes. Three intertwined patterns:

### Pattern 1 — Protocol projection drift

The original M-MCP-QUALITY doc identified this exact pattern: "serve-api projects modules into multiple protocols (HTTP, OpenAPI, A2A, MCP); each projection needs the same filtering and metadata enrichment." The MCP tool name fix in v0.10.7 was correct under that model. But v0.10.8-v0.10.10 then exposed three additional projection paths nobody had tabulated:

| Concern | Loader cache | HTTP routes | MCP tools | Function dispatch |
|---------|-------------|-------------|-----------|-------------------|
| Filter: local vs package | ❌ no notion of "local" | ✅ v0.10.8 (`File.Path` check) | partial | ❌ uses loader cache |
| Dedupe key | canonical ID | normalized rel path | name+type | canonical ID |
| Collision policy | last-writer-wins | filter + skip | prefer-doc-comment | inherit from cache |

The same module flowing through these four paths gets four different keys. v0.10.8 fixed one column, v0.10.9 added MOD011 to enforce one constraint, v0.10.10 patched its dedup — but the underlying drift isn't documented anywhere.

### Pattern 2 — Empty fields as load-bearing dead code

`ast.File.Path` was always empty before v0.10.10. The dep-discovery loop in [internal/apiserver/server.go:323-383](internal/apiserver/server.go#L323-L383) had a `if filePath == "" { continue }` early-out that effectively dead-coded the entire 60-line block. v0.10.8 added the loop intending it to run; v0.10.10 finally populated `file.Path` to make MOD011 work and **incidentally re-animated 60 lines of zombie code** that had never been exercised in production.

This is a class: **"feature gated on a field nobody populates."** A grep for `if .* == "" { continue }` in code paths whose other dependency just learned how to populate that field is a real refactoring hazard.

### Pattern 3 — Test gaps along the cascade

| Bug | Why tests didn't catch it |
|-----|---------------------------|
| v0.10.7 → v0.10.8 dispatch bug | No test exercised `loader.Preload(canonicalID, ...)` collision behavior |
| v0.10.9 false-positive | Test fixtures all gave each module a unique disk path; the "same physical file under two canonical IDs" case was never represented |
| v0.10.10 perf regression | No serve-api cold-start benchmark exists; CI runs on small fixtures where the double-load is invisible |

**The unifying gap:** we have unit tests for individual functions but no fixture that mirrors the docparse layout (local file + `module_prefix` package + `serve-api --routes-only`). All four bugs would have shown up in such a fixture, automated.

---

## Problem Statement

### Problem 1 (immediate) — Cold-start regression in v0.10.10

**Symptom (from msg_20260407_171038_1210b37d):**

```
v0.10.7:  Loaded module: docparse/main at +22s — done
v0.10.10: Loaded module: docparse/main at +21s — then ANOTHER full pass starts and is still running at +90s
```

Cloud Run default startup probe = 10 attempts × 10s = 100s. v0.10.7 fit comfortably. v0.10.10 doesn't. Production workaround: bump `failureThreshold` 10 → 30 (5 minutes), which is a real degradation for scale-from-zero.

**Mechanism:**

1. v0.10.10 populated `ast.File.Path` in [internal/loader/loader.go:250](internal/loader/loader.go#L250) so MOD011's symlink-resolved dedup would work.
2. The "Loaded package module: ... routes discovered" log line in [internal/apiserver/server.go:382](internal/apiserver/server.go#L382) belongs to the dep-discovery loop at [server.go:323-383](internal/apiserver/server.go#L323-L383). That loop's first early-out is `if filePath == "" { continue }` — so before v0.10.10 it skipped every module.
3. With `file.Path` now populated, the loop runs for the first time. It computes its `s.modules` registration key via `strings.TrimPrefix(modPath, trimmedBase+"/")` on the pipeline canonical ID, which often produces a different string than the main path's `filepath.Rel(s.basePath, absPath)`. Result: each local module is registered under **two** keys.
4. The eager-load loop at [server.go:233-245](internal/apiserver/server.go#L233-L245) iterates `s.modules` and calls `engine.Load(modPath)` for every entry. Two keys → two full compiles → 2× the work on phase 2.
5. Phase 1's ~25s overhead is the dep-discovery loop running its `extract*` AST traversals and `log.Printf` for every dep across all 11 `loadFile` calls.

**This is not MOD011's fault.** MOD011 is ~10 lines of in-memory map iteration. It's the second-order effect of populating a field that re-animated dormant code.

### Problem 2 (process) — No design doc for v0.10.7-v0.10.10

Each of the four releases was a "see message → fix → release" loop without:

- Axiom scoring
- Systemic audit (we never tabulated the projection columns above)
- A test fixture mirroring the user's environment
- A pre-flight grep for code paths gated on now-populated fields
- A cold-start benchmark in CI

The cascade is the natural consequence. The user's quote: "we are not using the process and perhaps its causing these cascading bugs."

---

## Goals

**Primary goals (v0.10.11):**

1. Restore docparse cold-start to ≤25s on Cloud Run (back to v0.10.7 baseline).
2. Eliminate the duplicate `s.modules` registration in the dep-discovery loop.
3. Add a regression test that catches the exact "same module registered under two keys → eager-load runs twice" case without needing Cloud Run.

**Secondary goals (v0.11.0 process work):**

4. Document the projection-key matrix from the Systemic Audit so the next protocol addition has a checklist.
5. Add a serve-api cold-start fixture that mirrors a docparse-scale project (10+ local modules + a `module_prefix` package).
6. Establish: **any change touching the loader/pipeline/serve-api triangle requires a design doc**, even for "small" patches. The cascade proves these aren't small.

**Non-goals:**

- Not changing MOD011 semantics — it's correct as of v0.10.10.
- Not changing `module_prefix` resolution — it's working as designed.
- Not adding caching layers ("cache discovery-phase IRs" as the user suggested). The fix is simpler: don't double-register in the first place.

---

## Proposed Fix (v0.10.11)

### Fix 1 — Unify dep-discovery key derivation with main path

**File:** [internal/apiserver/server.go:344-349](internal/apiserver/server.go#L344-L349)

**Current** (dep-discovery loop):
```go
normalizedPath := modPath
trimmedBase := strings.TrimPrefix(s.basePath, "/")
if trimmedBase != "" && strings.HasPrefix(modPath, trimmedBase+"/") {
    normalizedPath = strings.TrimPrefix(modPath, trimmedBase+"/")
}
```

**Proposed:** use the same derivation as the main path at [server.go:406-414](internal/apiserver/server.go#L406-L414):
```go
relPath, relErr := filepath.Rel(s.basePath, absFile)
normalizedPath := strings.TrimSuffix(filepath.Base(absFile), ".ail")
if relErr == nil && !strings.HasPrefix(relPath, "..") {
    normalizedPath = filepath.ToSlash(strings.TrimSuffix(relPath, ".ail"))
}
```

This is the unique source-of-truth key. Both paths now produce identical strings for the same file → no double registration → eager-load only sees one entry per module.

### Fix 2 — Move existence check before extract* work

The current loop calls `extract*` (4 AST traversals + doc-comment loading) **before** checking `existsByKey`. Move the check up so repeat visits across the 11 `loadFile` calls are O(map lookup), not O(parse + 4 traversals):

```go
// Check existence FIRST.
s.mu.RLock()
_, exists := s.modules[normalizedPath]
s.mu.RUnlock()
if exists {
    continue
}

depInfo := extractModuleInfo(loaded.Iface)
extractParamInfo(depInfo, loaded.File)
// ... etc
```

### Fix 3 — Skip dep-discovery for the file currently being loaded

The main path at [server.go:386-431](internal/apiserver/server.go#L386-L431) already handles the entry-point file. The dep-discovery loop should skip it explicitly:

```go
if absFile == absPath {
    continue // main path will register this with the canonical key
}
```

### Fix 4 — Add cold-start regression test

`internal/apiserver/cold_start_test.go`:

- Build a fixture with N local modules (N ≥ 5) all under a single base dir.
- Call `LoadModules` once.
- Assert: `len(s.modules)` equals N (no duplicates).
- Assert: each `engine.compiledUnits` is hit exactly once across the eager-load step (instrument via test hook or count via existing telemetry).
- Bonus: include a `module_prefix` package fixture so MOD011 + dep-discovery interact in the test.

### Verification

```bash
# Before:
docker run docparse-v0.10.10  # ~90s to "ready"
# After v0.10.11:
docker run docparse-v0.10.11  # target ≤25s
```

Plus: new cold-start test under `make test`.

---

## Process Changes (v0.11.0)

### Change 1 — Design doc gate for the loader/pipeline/serve-api triangle

**Rule:** any change touching `internal/loader/`, `internal/pipeline/`, or `internal/apiserver/` that:

- Adds/removes a field on `LoadedModule`, `ast.File`, `ModuleInfo`, or `ExportInfo`, OR
- Adds/removes a registration key in `s.modules`, the loader cache, or `compiledUnits`, OR
- Adds a new error code (MOD\*, LDR\*, PIPE\*, ROUTE\*)

requires a design doc with the projection-key matrix updated. v0.10.7-v0.10.10 each met one or more of these conditions. None had docs.

This rule lives in `.claude/rules/api-server.md` and `.claude/rules/type-system.md`.

### Change 2 — Projection key matrix as living doc

Add `docs/docs/internal/projection-key-matrix.md` with the table from the Systemic Audit. Every PR touching one of the four columns updates the matrix in the same commit. Forces the implementer to think about cross-column drift before merging.

### Change 3 — docparse-shape fixture in CI

Add a fixture under `tests/fixtures/projects/docparse_shape/`:

```
docparse_shape/
├── ailang.toml         (depends on a module_prefix package)
├── main.ail
├── services/
│   ├── api_keys.ail    (declares module docparse/services/api_keys)
│   ├── mcp_tools.ail   (imports the package, has @route)
│   └── csv_parser.ail  (helper)
└── pkg/                (vendored package with module_prefix = "docparse")
    └── ...
```

Run `serve-api --routes-only --check-only` in CI. Assert: no MOD011, no double-registration, exit < 30s. This single fixture would have caught all four bugs in the v0.10.7-v0.10.10 cascade.

### Change 4 — Empty-field reanimation grep

When populating a previously-empty field, run:

```bash
git grep -n 'if [a-zA-Z.]*\.Path == ""' internal/  # or whichever field
```

Anything that matches is dormant code that may now run for the first time. PR description must enumerate which guards become live and confirm each is intended.

### Change 5 — Cold-start benchmark in CI

Add `make bench-cold-start` running the docparse-shape fixture and asserting wall-clock < 30s on CI hardware. Hook into the release-manager pre-flight checks.

---

## Postmortem Timeline

| Time (2026-04-07/08) | Event | Process gap |
|----------------------|-------|-------------|
| ~10:00 | Five MCP bugs reported by `cli` and docparse | (M-MCP-QUALITY had a doc — clean) |
| ~12:00 | v0.10.7 ships with `@mcp_name` and validation | No projection matrix → couldn't see dispatch was independent |
| ~13:30 | docparse reports route leak; we ship v0.10.8 | Patch-only; no doc; fix only the symptom we saw |
| ~14:30 | docparse reports dispatch footgun (silent shadowing) | First sign the cascade is real |
| ~15:00 | v0.10.9 ships with MOD011 | No fixture matching docparse layout; test used unique disk paths only |
| ~16:00 | docparse reports MOD011 false-fires on `module_prefix` packages | Same gap, second time |
| ~16:30 | v0.10.10 ships with EvalSymlinks dedup; populates `file.Path` | No grep for `Path == ""` guards; zombie code reanimated |
| ~17:10 | docparse reports 4× cold-start regression | Cloud Run probe deadline exceeded; production impact |
| ~17:20 | docparse bumps probe failureThreshold 10 → 30 | User mitigation |
| 2026-04-08 | This doc | Process change to break the loop |

**Five bug reports, four releases, zero design docs.** The cost of patch-only loops on a four-column projection isn't linear — each fix uncovers the next column.

---

## Risks & Open Questions

| Risk | Mitigation |
|------|-----------|
| Fix 1's key change might affect existing route lookups | Add a test asserting registration keys before/after on the docparse-shape fixture |
| Fix 3 might skip a legitimate edge case where the main path and dep-discovery do different things | Audit what each loop *uniquely* does before deduplicating |
| Process changes (design doc gate) might slow down genuinely small fixes | Carve out exceptions: doc updates, comment-only changes, test-only additions |
| Cold-start benchmark in CI is flaky on slow runners | Use ratio comparison (within 1.5× of baseline) instead of absolute deadline |

**Open questions:**

1. Should `loader.Preload` return an error on cache collision (not just same-canonical-ID — different physical files)? Currently MOD011 catches this *upstream* in the pipeline but not in serve-api's per-file `loadFile` calls. **Recommendation:** yes, in v0.11.0 — defense in depth.
2. Should `ast.File.Path` have been populated from day one? Probably yes. v0.10.10 fixed it incidentally. **Recommendation:** add a parser-side audit for "empty by default but should be set" fields.
3. Is dep-discovery the right pattern, or should serve-api just walk the project filesystem once and call `loadFile` per `.ail` file (no transitive dep registration)? **Open** — bigger refactor; out of scope for v0.10.11.

---

## Acceptance Criteria

### v0.10.11 (the fix)

- [ ] Fix 1: dep-discovery loop uses `filepath.Rel` matching the main path
- [ ] Fix 2: existence check moved before `extract*` work
- [ ] Fix 3: dep-discovery skips the entry-point file
- [ ] Fix 4: regression test asserts no duplicate `s.modules` entries on a 5-module fixture
- [ ] docparse cold-start measured at ≤25s on Cloud Run (reported back via msg)
- [ ] Changelog entry referencing this doc and the four-release cascade

### v0.11.0 (the process)

- [ ] Projection key matrix doc at `docs/docs/internal/projection-key-matrix.md`
- [ ] Rule update in `.claude/rules/api-server.md` (design doc gate)
- [ ] Fixture at `tests/fixtures/projects/docparse_shape/`
- [ ] `make bench-cold-start` target hooked into release-manager pre-flight
- [ ] Postmortem linked from `CHANGELOG.md` for v0.10.7-v0.10.10

---

## References

- Original M-MCP-QUALITY doc: [m-mcp-quality-and-route-headers.md](../../implemented/v0_11_0/m-mcp-quality-and-route-headers.md)
- Message chain (chronological):
  - msg_20260407_141818_258f3b49 (initial route leak)
  - msg_20260407_153029_c287d189 (dispatch footgun)
  - msg_20260407_161513_61000495 (MOD011 false-positive)
  - msg_20260407_171038_1210b37d (cold-start regression)
- Releases:
  - [v0.10.7](https://github.com/sunholo-data/ailang/releases/tag/v0.10.7)
  - [v0.10.8](https://github.com/sunholo-data/ailang/releases/tag/v0.10.8)
  - [v0.10.9](https://github.com/sunholo-data/ailang/releases/tag/v0.10.9)
  - [v0.10.10](https://github.com/sunholo-data/ailang/releases/tag/v0.10.10)
- Affected files (cumulative across cascade):
  - [internal/parser/parser_decl.go](../../../internal/parser/parser_decl.go)
  - [internal/apiserver/server.go](../../../internal/apiserver/server.go)
  - [internal/apiserver/routes.go](../../../internal/apiserver/routes.go)
  - [internal/apiserver/mcp.go](../../../internal/apiserver/mcp.go)
  - [internal/apiserver/mcp_schema.go](../../../internal/apiserver/mcp_schema.go)
  - [internal/loader/loader.go](../../../internal/loader/loader.go)
  - [internal/pipeline/pipeline_module.go](../../../internal/pipeline/pipeline_module.go)
