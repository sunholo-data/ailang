# Sprint Plan: M-TRANSITIVE-ALIAS-ENV-IMPORT

## Summary

Fix the cross-module nested record-type alias unification bug (`cannot unify type constructor Inner with *types.TRecord`) that blocks `motoko_agent` PR #28 and silently breaks any package splitting record-type aliases across multiple modules. The fix is a ~15-line addition to `resolveModuleImports` that walks the linker's full closure of loaded ifaces, plus a regression test.

**Duration:** 1 day (~4 hours)
**Dependencies:** None — design doc is locked, no schema changes required
**Risk Level:** Low — additive change (`aliasEnv` only widens), no public API touched, no iface digest churn

## Current Status Analysis

### Completed Recently (last 7 days)

- ✅ `ad84b68d` — M-WASM-TYPECHECK-FLOAT-DIVERGENCE (~50 LOC) — the WASM-side analog of this exact bug. Mirrors what we need on the CLI path; gives us a known-good code shape to reference.
- ✅ `3325d39f` — M-SCHEME-IMPORT-PRESERVE-ADT-HEAD (~30 LOC) — the commit that tightened generalization and exposed this pre-existing gap.
- ✅ `4f4dda3e` — M-PROMPT-STDLIB-COVERAGE — prompt work, not relevant to LOC velocity for type-system fixes.

### Velocity

- Recent comparable fix (ad84b68d): ~50 LOC implementation + ~30 LOC test, single-session.
- This fix is even more surgical (~15 LOC core change) — the linker already exposes `GetLoadedModules()`, so no plumbing.

### Remaining from Design Doc

- ⏳ M1: implement transitive alias closure pass — ~15 LOC
- ⏳ M2: regression test cloning the 3-file repro — ~120 LOC
- ⏳ M3: verify end-to-end + ack inbox message — runtime only

## Proposed Milestones

### Milestone 1: Implement transitive alias closure pass

**Goal:** Widen `imports.ImportedTypeAliases` to include aliases from every loaded iface, not just direct deps. Local aliases continue to win via existing first-wins ordering.

**Estimated:** 15 LOC implementation + 0 test LOC (test in M2)
**Duration:** ~30 min

**Tasks:**

- Edit [internal/pipeline/pipeline_module_imports.go](internal/pipeline/pipeline_module_imports.go) — add a closure pass at the end of `resolveModuleImports`, after the existing `for _, imp := range fileImports` loop:
  ```go
  // M-TRANSITIVE-ALIAS-ENV-IMPORT: pull aliases from every loaded module,
  // not just direct deps. Required when A declares an alias, B imports A
  // and re-uses the alias inside its own exported types, C imports B but
  // not A — C's unifier needs A's alias to expand TCon → TRecord.
  // First-wins ordering preserves direct-import precedence.
  for modPath, modIface := range modLinker.GetLoadedModules() {
      if modPath == "$builtin" || modIface == nil {
          continue
      }
      for aliasName, aliasTarget := range modIface.TypeAliases {
          if _, exists := imports.ImportedTypeAliases[aliasName]; !exists {
              imports.ImportedTypeAliases[aliasName] = aliasTarget
          }
      }
  }
  ```
- Build with `make quick-install`.
- Smoke-test against `/tmp/typebug-repro/` repro from the design doc — must type-check clean.

**Acceptance Criteria:**

- [ ] Repro at `/tmp/typebug-repro/typebug/main.ail` type-checks under `AILANG_RELAX_MODULES=1 ailang check`
- [ ] No regression in `make test` (full suite green)
- [ ] No regression in `make verify-examples`
- [ ] `make lint` clean
- [ ] Commit references `Fixes` for the inbox issue if a GitHub issue exists, else just descriptive title

**Risks:**

- Alias-name collision when two transitively-loaded modules export same-named aliases with different bodies. *Mitigation:* first-wins preserves direct-import precedence; documented as out-of-scope in the design doc; will revisit if it surfaces in real packages.

### Milestone 2: Regression test

**Goal:** Lock the fix in with a unit test that clones the 3-file repro pattern. Test must FAIL on `dev` HEAD without M1 and PASS with M1 applied.

**Estimated:** ~120 LOC (test fixture + harness call + assertions)
**Duration:** ~1.5 hours

**Tasks:**

- Add `TestCrossModuleNestedRecordAlias` to [internal/pipeline/pipeline_module_compile_test.go](internal/pipeline/pipeline_module_compile_test.go) (or new file `pipeline_module_alias_transitive_test.go` if the existing file is large — check first, prefer extending if file is <600 LOC).
- Test fixture: 3 modules (`pkg/a` declares `Inner`, `pkg/b` imports `Inner` and exports `Outer` + functions, `pkg/c` imports `pkg/b` and calls them). Use the same shape as the verified `/tmp/typebug-repro/`.
- Assert: type-check succeeds, no unification error mentioning `TCon`.
- Add a second sub-test for collision precedence: two modules each declare `type Status = { ... }` with different bodies; importer pulls `Status` explicitly from one + a function from the other; assert the explicitly-imported `Status` wins.

**Acceptance Criteria:**

- [ ] Test fails on `dev~1` (pre-M1) — verify by stashing M1, running test, expecting failure
- [ ] Test passes on `dev` after M1
- [ ] Test runs in <500ms
- [ ] No external fixture files (test fixtures inline as strings to keep it self-contained)

**Risks:**

- The pipeline test harness may require non-trivial setup (linker, elaborator, ModuleRegistry). *Mitigation:* read neighboring tests in the same file first; reuse existing fixture helpers if they exist.

### Milestone 3: Verify end-to-end + reply on inbox

**Goal:** Confirm fix lands, reply to motoko_agent on `msg_20260522_170317_f850eba4` with commit SHA so they can rebase and republish.

**Estimated:** 0 LOC (runtime verification + comms)
**Duration:** ~30 min

**Tasks:**

- Run full `make test` and `make verify-examples` — both green.
- Run `make ci` if M1+M2 looks clean — final pre-flight.
- Send reply via `ailang messages send motoko_agent ...` with the fix commit SHA + a one-liner that `motoko_ext_mcp` can now republish.
- Ack inbox message `msg_20260522_170317_f850eba4`.

**Acceptance Criteria:**

- [ ] `make test` exit 0
- [ ] `make verify-examples` exit 0
- [ ] `make ci` exit 0 (optional — skip if running too long, but at minimum lint+test+verify)
- [ ] Reply sent to motoko_agent referencing commit SHA
- [ ] Inbox message acked

**Risks:**

- `make ci` may surface unrelated flakes from in-progress work in `internal/eval_harness/` (per git status — there are uncommitted changes to ai_agent.go, ai_provider.go, models.yml). *Mitigation:* if a CI failure is unrelated to type-system, document and proceed; do not block on unrelated dirty state.

## Success Metrics

- Repro: `/tmp/typebug-repro/typebug/main.ail` type-checks cleanly: ✅
- Regression test: locked in `internal/pipeline/`: ✅
- Test coverage: no decrease (this is a small additive change, no new uncovered code)
- Examples passing: no regression (`make verify-examples` green)
- Documentation: design doc moves from `planned/v0_22_0/` to `implemented/v0_22_0/` on completion (handled by post-release skill, not in-sprint)
- Inbox: `f850eba4` acked, reply sent

## Dependencies

- None. The fix is a leaf change with no upstream/downstream blockers.

## Open Questions

- **Target release version**: design doc lists "v0.21.x patch / v0.22.0". A patch release feels right (it's a bug fix with no schema change), but the user may prefer to bundle this with whatever else is queued for v0.22.0. **Default: ship in next patch if one is cut soon, otherwise hold for v0.22.0.**
- **motoko_ext_mcp republish**: should we proactively rebuild and republish `motoko_ext_mcp@0.2.8` after the fix lands, or hand back to motoko_agent? **Default: hand back to motoko_agent — the package is in their repo and they own the version cut.**

## Notes

- The fix is symmetric to `ad84b68d` (WASM path) — when in doubt about implementation shape, mirror that commit's approach but in `internal/pipeline/` instead of `internal/repl/`.
- The bug report from `motoko_agent` is unusually thorough: it includes the bisect, the suspected file:line, the WASM-fix reference. Honor that work — keep the implementation minimal and matched to the diagnosis.
- Pre-existing collision behavior (flat alias namespace) is preserved. Namespacing (`module.AliasName` qualified keys) is explicitly out of scope; revisit at v1.x if it bites.
