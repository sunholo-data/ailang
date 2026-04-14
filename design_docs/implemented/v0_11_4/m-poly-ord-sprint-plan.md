# Sprint Plan: M-POLY-ORD (Cache Fix + Ord/Eq Defaulting Regression)

**Sprint ID**: M-POLY-ORD
**Design doc**: [m-poly-ord-defaulting-regression.md](m-poly-ord-defaulting-regression.md)
**Target version**: v0.11.4
**Estimated duration**: 2 days (~8-10 hours)
**Risk level**: Medium (touches type system — defaulting edge cases)

## Sprint Goal

Fix the polymorphic `Ord`/`Eq` lambda regression so
`polymorphic_comparison_simple.ail` and `polymorphic_lambdas_phase1.ail`
work again, AND fix the module-cache version-skew gotcha that blocked
debugging so future compiler fixes actually take effect without manual
cache nukes.

## Why cache fix first

During diagnosis of the polymorphic bug, my debug prints in the type
checker never ran — the module cache silently skipped compilation
because its cache key only hashes source + `"v1"` format constant, not
the compiler build. Any future bugfix to elaboration, type-checking, or
op-lowering hits the same silent-skip. Land the cache fix first so the
defaulting fix can be verified end-to-end in CI and by contributors.

**Phase 2 (scheme instantiation leak)** from the design doc is OUT OF
SCOPE for this sprint — it's a deeper investigation into
`generalizeWithConstraints` and worth its own sprint. This sprint nails
the user-visible regression and the cache foot-gun.

## Milestones

### M1: Cache key incorporates build commit (~120 LOC)

**Problem**: [cache_key.go:22](../../../internal/pipeline/cache_key.go#L22)
takes a `compilerVersion` param, but callers pass the format constant
`"v1"` — so rebuilding `ailang` doesn't invalidate cache.

**Approach**:
1. Create `internal/version/version.go` package exposing `Commit`,
   `Version`, `BuildTime` vars populated via `-ldflags -X` (Makefile
   already does this for `main.*` — move target to
   `internal/version.*` so non-main packages can read it).
2. Update [Makefile:42](../../../Makefile#L42) ldflags target from
   `main.Commit` → `github.com/sunholo/ailang/internal/version.Commit`
   (and friends).
3. Add `runtime/debug.ReadBuildInfo` fallback in `internal/version`
   init() so `go run` / `go test` without ldflags still get a stable
   marker instead of empty string.
4. Thread `version.Commit` into `ModuleCacheKey` callers
   ([pipeline_module.go:219](../../../internal/pipeline/pipeline_module.go#L219)
    — grep for all `ModuleCacheKey(` call sites).
5. Update `cmd/ailang` version command to import from
   `internal/version` instead of local `main.*` vars.

**Files**:
- **new**: `internal/version/version.go` (~40 LOC)
- `Makefile` (ldflags target rename)
- `internal/pipeline/cache_key.go` (doc comment update)
- `internal/pipeline/pipeline_module.go` (caller)
- `internal/pipeline/cache_store.go` (if any other callers)
- `cmd/ailang/version.go` (import from internal/version)

**Acceptance**:
- [ ] After `make quick-install`, running a previously-cached module
      reports `CACHE MISS` and recompiles.
- [ ] Subsequent run of the same module (no rebuild) reports
      `CACHE SKIP`.
- [ ] `go run ./cmd/ailang run <file>` without ldflags still works
      (uses `debug.ReadBuildInfo` fallback, doesn't error).
- [ ] `ailang version` still prints commit / build time.
- [ ] No test regressions in `internal/pipeline/cache_*_test.go`.
- [ ] New test: `TestModuleCacheKey_CommitChange` asserts different
      commit → different key.

### M2: Remove premature Ord/Eq → Int defaulting (~80 LOC)

**Problem**: [typechecker_defaulting.go:217-223](../../../internal/types/typechecker_defaulting.go#L217-L223)
defaults Ord-only / Eq-only constraints to `Int` at inner
generalization boundaries. This monomorphizes `max`, `gt`, and similar
polymorphic lambdas so they crash at runtime on `Float` / `String`.

**Approach**:
1. Remove the `classes["Ord"] || classes["Eq"]` and `classes["Show"]`
   branches from `pickDefault`. Leave them unsolved — the caller
   (`defaultAmbiguities` at inner boundaries) already falls through to
   `default: fmt.Errorf("mixed constraints require type annotation")`
   for non-handled cases.
2. At inner (let) boundaries, we don't want an error — just leave
   `Ord α` as an unsolved constraint for generalization to pick up.
   Update `defaultAmbiguities` (NOT `defaultAmbiguitiesTopLevel`) to
   **skip** vars whose only classes are Ord/Eq/Show — don't try to
   default them, don't error on them.
3. At **top-level** program boundary
   (`defaultAmbiguitiesTopLevel`), if an Ord/Eq/Show-only var is
   genuinely ambiguous (no numeric context, no generalization to
   rescue it) — report an unambiguous error that says "type
   annotation required", not silently default.

**Files**:
- `internal/types/typechecker_defaulting.go` (pickDefault +
  defaultAmbiguities helper logic)
- `internal/types/defaulting_test.go` (update any test that expected
  Ord → Int defaulting)
- `internal/types/simple_defaulting_test.go` (likewise)

**Acceptance**:
- [ ] `ailang run examples/runnable/polymorphic_comparison_simple.ail`
      succeeds (prints the record of max results for float/int/string).
- [ ] `ailang run examples/runnable/polymorphic_lambdas_phase1.ail`
      succeeds.
- [ ] `ailang run` on the minimal repros:
      - `let max = \x.\y. if x>y then x else y in max(3.14)(2.71)` → `3.14`
      - `let max = \x.\y. if x>y then x else y in max(42)(17)` → `42`
      - `let max = \x.\y. if x>y then x else y in max("a")("b")` → `"b"`
- [ ] Existing defaulting tests still pass, OR are updated to reflect
      that Ord-only is no longer defaulted.
- [ ] New test: polymorphic `max` lambda round-trips through Float, Int, String.
- [ ] `make test` passes.
- [ ] `make verify-examples` passes.

### M3: Regression tests + docs (~80 LOC)

**Approach**:
1. Add unit tests:
   - `internal/types/defaulting_test.go`:
     `TestPickDefault_OrdOnly_NotDefaulted` — pickDefault on `{Ord}`
     no longer returns `TInt`.
   - `internal/pipeline/op_lowering_comparison_test.go`:
     polymorphic lambda lowers per call-site.
2. Add example: confirm the two existing Phase 1 examples are verified
   by `make verify-examples` (update expected outputs if needed).
3. Update [CHANGELOG.md](../../../CHANGELOG.md) with:
   - Fixed: polymorphic Ord/Eq lambdas regressed since v0.2.0
   - Fixed: module cache didn't invalidate on compiler rebuild
4. Move design doc from `planned/v0_11_4/` → `implemented/v0_11_4/`
   once sprint completes.

**Files**:
- `internal/types/defaulting_test.go`
- `internal/pipeline/op_lowering_comparison_test.go`
- `CHANGELOG.md`
- `design_docs/planned/v0_11_4/m-poly-ord-defaulting-regression.md` → move

**Acceptance**:
- [ ] New unit tests pass.
- [ ] `make verify-examples` green.
- [ ] CHANGELOG entry under `### Fixed` in v0.11.4 section.
- [ ] Design doc moved to `implemented/`.

## Day-by-day breakdown

### Day 1 (~5h)
- **M1 (cache key fix)** — ~3h implementation + 1h testing
  - Create `internal/version` package, wire ldflags, update callers
  - Verify cache invalidation manually
  - Add `TestModuleCacheKey_CommitChange`
- **M2 (defaulting fix) — start** — ~1h
  - Remove Ord/Eq branch from `pickDefault`
  - Run `ailang run` on reproducers to confirm basic fix

### Day 2 (~5h)
- **M2 — finish** — ~2h
  - Update `defaultAmbiguities` to skip Ord-only vars at inner boundaries
  - Handle top-level ambiguity errors cleanly
  - Update existing defaulting tests
- **M3 (regression tests + docs)** — ~2h
- **Final verification** — ~1h
  - `make ci` green
  - Both example files run end-to-end
  - Move design doc to `implemented/`
  - CHANGELOG updated

## Out of scope (deferred)

- **Phase 2 from design doc**: scheme instantiation leak investigation.
  `Ord[α2]` leaking past scheme boundary suggests deeper bug in
  `generalizeWithConstraints` — worth its own sprint once this sprint's
  fix is in place.
- **Monomorphization restoration**: `0 specializations` on non-module
  top-level `let` lambdas — orthogonal, track separately.
- **Haskell-style `default (...)` declarations**: too broad for this sprint.

## Success metrics

- Both previously-failing example files run cleanly.
- `make ci` green.
- Cache behaviour: rebuild → miss; no rebuild → hit.
- No new eval baseline regressions (may need to re-run baselines if
  Ord-only defaulting was silently helping some benchmarks — track
  separately).

## Risk factors

- **Medium risk**: Removing Ord→Int defaulting may expose other tests
  that implicitly depended on it (`grep` for test cases that compare
  untyped `Ord` values). Mitigation: run full test suite early, triage
  any breakage case by case.
- **Low risk (cache fix)**: Contained to pipeline package; ldflags
  already exist, just moving target.

## Dependencies

None. Both milestones are self-contained; M1 before M2 is a pragmatic
ordering (verify M2 actually takes effect), not a hard dependency.

## Open questions

- Should `defaultAmbiguitiesTopLevel` error on Ord-only ambiguity at
  top-level, or silently default to `Int` as a pragmatic REPL
  experience? Design doc Phase 1 proposes error; user may prefer the
  Haskell-ish "keep it abstract" behaviour since AILANG files are
  typically programs, not REPL one-liners. **Will ask during M2
  implementation.**
