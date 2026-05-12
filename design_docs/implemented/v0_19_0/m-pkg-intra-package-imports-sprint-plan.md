# M-PKG-INTRA-PACKAGE-IMPORTS — Sprint Plan

**Design doc**: [m-pkg-intra-package-imports.md](m-pkg-intra-package-imports.md)
**Sprint ID**: M-PKG-INTRA-PACKAGE-IMPORTS
**Target**: v0.19.0
**Estimated**: ~4–6 hours (one focused session)
**Risk**: Low–Medium
**Created**: 2026-05-12
**Source**: inbox message [913c0aa4](msg_20260512_144239_913c0aa4) from `demos/linkedin`

---

## Pre-flight checks (done)

- ✅ Self-reference logic already exists at [internal/pkg/loader.go:41-44](../../../internal/pkg/loader.go#L41-L44) — just isn't reached when no lock file
- ✅ `PackageLoader` is constructible from `(LockFile, rootDir)` — empty lock works, see [internal/pkg/loader.go:18](../../../internal/pkg/loader.go#L18)
- ✅ Dispatch site is single-point in [internal/pipeline/pipeline_module.go:89](../../../internal/pipeline/pipeline_module.go#L89) — one place to add fallback
- ✅ Bare-canonical routing has precedent in modulePrefixMap branch at [internal/loader/loader.go:190-219](../../../internal/loader/loader.go#L190-L219) — same pattern, different trigger
- ✅ Local repro confirmed: all three import forms fail on v0.18.11 (LDR001 / "requires ailang.lock")
- ✅ Validator already generates a lock file at [cmd/registry-validator/main.go:482-489](../../../cmd/registry-validator/main.go#L482-L489) — but verify it's wired correctly post-M2

**Conclusion**: Plumbing exists. The fix is a two-point change — wire a self-only resolver when no lock exists, and route bare canonical paths through the existing self-reference branch.

---

## Velocity context

Recent comparable work (last 7 days):
- M-EXT-PORTABILITY-GATE F1–F5 — 5 small features in ~1 day
- M-PARSER-ROW-POLY-EFFECTS Phase 1 + 2 — 2 features, ~1 day each
- M-NET-BINARY-BODIES sprint plan landed today (parallel work)

Pattern: small, additive, loader/pkg changes at ~1 feature per 1–2h. This sprint = 5 milestones of ~30–90 min each ⇒ 4–6h total is realistic.

---

## Milestones

### M1 — Self-only `PackageLoader` constructor (~45 min, ~50 LOC + 80 LOC tests)

**Goal**: Make a `PackageLoader` constructible from `ailang.toml` alone (no lock file) that resolves self-references and clearly errors on external imports.

**Files**:
- `internal/pkg/loader.go` — add `NewSelfOnlyPackageLoader(rootDir)` constructor
- `internal/pkg/loader_self_only_test.go` — new test file

**Tasks**:
1. Add `NewSelfOnlyPackageLoader(rootDir string) (*PackageLoader, error)` that loads `ailang.toml` to learn the package name, then constructs the loader with an empty `LockFile{}`.
2. Verify `ResolveImport` for `<self>/<sibling>` returns the sibling path; for `<other>/<x>` returns a clear "no lock file" error (not generic LDR001).
3. Tests:
   - self-reference resolves to sibling file
   - external import errors with explicit "run `ailang lock`" message
   - missing `ailang.toml` errors clearly (used by caller as a "not a package" signal)

**Acceptance**:
- [ ] `NewSelfOnlyPackageLoader` returns a working loader for a dir with `ailang.toml` only
- [ ] Self-reference resolution succeeds
- [ ] External imports return an error containing "ailang.lock"
- [ ] `make test ./internal/pkg/...` passes

**Risk**: Low. New constructor, no existing code paths changed.

---

### M2 — Wire self-only resolver as fallback in pipeline (~30 min, ~25 LOC + 60 LOC tests)

**Goal**: When `tryLoadPackageResolver` fails (no lock), fall back to `NewSelfOnlyPackageLoader` so intra-package imports work during authoring.

**Files**:
- `internal/pipeline/pipeline_module.go` — add fallback branch at line 89
- `internal/pipeline/pipeline_module_intra_pkg_test.go` — new test fixture

**Tasks**:
1. Add fallback after the existing `if pkgResolver := tryLoadPackageResolver("."); pkgResolver != nil` block.
2. New helper `tryLoadSelfOnlyPackageResolver(dir)` returns the self-only loader or nil.
3. Tests with fixture `internal/pipeline/testdata/intra_pkg_fresh/`:
   - `ailang.toml` + `types.ail` + `service.ail` (imports `pkg/<self>/types`)
   - `ailang check --package .` succeeds without `ailang.lock`
   - Adding an external `pkg/<other>/x` import errors clearly

**Acceptance**:
- [ ] Test fixture compiles via the pipeline with no lock file present
- [ ] `pkg/<self>/<sibling>` form works
- [ ] `./` (relative) form works (was broken pre-M2 because the relative normalizer routes through `pkg/`)
- [ ] External `pkg/<other>/...` imports still error with the lock-file message

**Risk**: Low. Pure additive fallback. Existing path (with lock) is unchanged.

---

### M3 — Bare canonical form routing (~60 min, ~40 LOC + 100 LOC tests)

**Goal**: `import sunholo/linkedin/types` (no `pkg/` prefix, no `./`) — the form that matches the module declaration — resolves when the first two segments match the current package's name.

**Files**:
- `internal/loader/loader.go` — add bare-canonical check before modulePrefixMap fallback at line 190
- `internal/loader/loader_intra_pkg_test.go` — new test file

**Tasks**:
1. In `Load`, after the `pkg/` branch and before `modulePrefixMap`, check: if `ml.pkgLoader != nil` AND `canonPath` starts with `<vendor>/<name>` of the loader's current package, treat as self-reference. Construct `pkgImportPath := "pkg/" + canonPath`, route through `pkgLoader.ResolveImport`.
2. Get current package name via the resolver itself — `PackageResolver` doesn't expose this today, so add a `CurrentPackageName() string` method on the interface (no-op for mock resolvers in existing tests).
3. **Option B fallback**: if path matches package prefix but `pkgLoader == nil`, error with "use `./` or `pkg/<vendor>/<name>/<mod>` within the same package" instead of generic LDR001.
4. Tests for all three forms resolving identically.

**Acceptance**:
- [ ] `import sunholo/repro_pkg/types` (bare) resolves from within `sunholo/repro_pkg`
- [ ] All three import forms produce identical resolved file paths (verify via search trace)
- [ ] Non-matching bare paths still fall through to project-relative (no regression)
- [ ] Helpful error when bare path matches package prefix but pkgLoader is unset
- [ ] All existing `loader_test.go` cases pass unchanged

**Risk**: Medium. Touches the dispatch order in `Load`. Mitigation: keep the new check tightly scoped (must start with `<vendor>/<name>` from `ailang.toml`), with a fixture test for the "mismatch between module decl and ailang.toml name" edge case.

---

### M4 — End-to-end registry validator test (~60 min, ~150 LOC tests)

**Goal**: Confirm `ailang publish` works for a multi-module package with intra-package imports — closes the loop on the original LinkedIn report.

**Files**:
- `cmd/registry-validator/main_intra_pkg_test.go` — new test (or extend existing `main_test.go`)
- Optionally: `internal/pkg/publish_validator_test.go` — extend if smoke validation needs updating

**Tasks**:
1. Test: construct a tarball with `ailang.toml`, `types.ail`, `service.ail` (sibling-import). POST to a local validator instance (in-process via `httptest.Server`). Assert 200 OK and that the published response includes both modules.
2. Test: same tarball but with bare-canonical import form (matches LinkedIn's actual layout). Assert identical success.
3. If the validator's lock-generation path is broken independently of `ailang check`, fix it here and document in the milestone notes.

**Acceptance**:
- [ ] Validator accepts a multi-module tarball with `./` imports
- [ ] Validator accepts a multi-module tarball with bare-canonical imports (LinkedIn's form)
- [ ] Validator accepts a multi-module tarball with `pkg/<self>/...` imports
- [ ] External deps still require `ailang.lock` in the tarball

**Risk**: Medium. The validator may have its own bug we discover here. Budget includes time to scope/fix it as part of this sprint (rather than splitting).

---

### M5 — Docs + example file (~45 min, ~80 LOC)

**Goal**: Update teaching surfaces so future authors know how intra-package imports work.

**Files**:
- `examples/intra_package_imports/` — new example dir with `ailang.toml`, `types.ail`, `service.ail`, `README.md` showing all three forms
- `docs/docs/reference/language-syntax.md` — add "Intra-package imports" subsection under module system
- `prompts/v0.19.md` (or current latest prompt) — add the canonical pattern
- `CHANGELOG.md` — entry under v0.19.0

**Tasks**:
1. Create the example. `make verify-examples` must pass on it.
2. Docs subsection covers: what an intra-package import is, the three valid forms, which to prefer (`./` for short, `pkg/<full>` for explicit), what happens during authoring vs after `ailang lock`.
3. CHANGELOG entry: "fix: intra-package imports now work without `ailang.lock` (M-PKG-INTRA-PACKAGE-IMPORTS)".

**Acceptance**:
- [ ] `examples/intra_package_imports/` compiles via `ailang check --package .` with no lock file
- [ ] `make verify-examples` passes
- [ ] Docs page renders correctly (`cd docs && npm run build`)
- [ ] CHANGELOG entry committed

**Risk**: Low. Pure docs/examples.

---

## Day-by-day plan

**Single-session sprint (4–6h):**

| Block | Milestone | Duration |
|---|---|---|
| 1 | M1 — Self-only loader + tests | ~45 min |
| 2 | M2 — Pipeline fallback + fixture | ~30 min |
| 3 | M3 — Bare canonical routing + tests | ~60 min |
| Break | Verify with original LinkedIn layout from git history | ~15 min |
| 4 | M4 — Validator end-to-end test | ~60 min |
| 5 | M5 — Docs + example + CHANGELOG | ~45 min |
| Close | Commit + ack LinkedIn message | ~15 min |

Total: ~4.5h work + ~30 min buffer.

---

## Success metrics

- [ ] All 5 milestones merged
- [ ] All three import forms resolve identically on a no-lock-file authoring fixture
- [ ] Registry validator accepts multi-module packages with intra-package imports
- [ ] LinkedIn's original cross-module layout (from this morning's git history in `sunholo-data/ailang-packages`) publishes successfully
- [ ] Example file in `examples/intra_package_imports/` works end-to-end
- [ ] `make ci` clean
- [ ] No regression in existing registry packages (`ailang check` on each `sunholo/*` package)
- [ ] CHANGELOG entry + docs subsection + prompt update
- [ ] Reply to inbox message [913c0aa4](msg_20260512_144239_913c0aa4) closing the loop

---

## Dependencies and open questions

**Dependencies**: None. Additive change.

**Open questions** (resolve during M3):
1. Should `PackageResolver` interface gain `CurrentPackageName()`, or should `ModuleLoader` cache it from `ailang.toml` at setup time? — Lean toward the latter; resolver doesn't need to know.
2. When `ailang.toml` `name` and the module declaration's first two segments disagree (`name = "sunholo/foo"` but `module sunholo/bar/types`), is that an authoring bug we should error on, or silently honor the module decl? — Probably warn during M3 implementation; defer to follow-up if non-trivial.

**Out of scope** (per design doc):
- Refactoring published packages to drop their inline-types workarounds
- Autoderiving module names from filesystem paths
- A general "module name = filesystem path" enforcement pass
