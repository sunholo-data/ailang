# Sprint Plan — M-PRELUDE-OPTION-RESULT (mission iter 27)

**Goal:** `Option`/`Some`/`None`/`Result`/`Ok`/`Err` resolve WITHOUT `import std/option` /
`import std/result` in **entry modules only**. User imports and local type definitions shadow
the prelude cleanly. Library (non-entry) modules still require explicit imports.

**Design doc:** `design_docs/planned/v0_29_0/m-prelude-option-result.md` (written 2026-06-03;
factual claims re-verified against code at commit 5cf7235b0 — corrections below).

**Estimate:** 1.5 days, ~330 LOC. **Risk:** medium (touches identifier resolution in entry modules).

---

## Verified root cause (behavioral repro, worktree binary @ 5cf7235b0)

`test/noimport.ail` (Some/None, no import) → **`ailang check` FAILS**:
```
Error: type error in tmp/iter27/noimport (decl 0): undefined variable: Some at :4:24
```
The identical file WITH `import std/option (Option, Some, None)` → `check` clean, `run` prints `42`.

**The failure is at TYPE-CHECK time**, raised by `internal/types/typechecker_literals.go:62`
(`undefined variable: %s`). Root cause chain (all verified with file:line):

1. **Elaborator** (`internal/elaborate/expr_calls.go:49`, `expressions.go:40`, `patterns.go:97`):
   `Some(x)` / bare `None` / match `Some(x)` are only rewritten to `$adt.make_Option_Some`
   factory calls **if `Some` is in `e.constructors`**. That map is populated ONLY from
   imported constructors (`pipeline_module.go:314-316` calls
   `elaborator.RegisterConstructor` for each `imports.ImportedCtorInfos`) or locally-declared
   types (`file_funcs.go:187`). With no import, `Some` stays a plain `Var` → type checker sees
   an unbound variable → the error above.

2. **Type checker** needs three things per constructor, all currently supplied only via imports
   in `internal/pipeline/pipeline_module_imports.go::resolveConstructorImport` (lines 250-317):
   - `imports.ExternalTypes["$adt.make_Option_Some"]` = factory Scheme (line 286)
   - `imports.ImportedCtorTypes["Some"] = "Option"` (line 297) → fed to
     `typeChecker.SetConstructorTypes` (`pipeline_module_compile.go:111`) for match inference
   - `imports.ImportedADTTypeParams["Option"] = 1` (line 305) → `SetADTTypeParams` (compile:112)

3. **Runtime** (`internal/runtime/resolver.go`): `$adt.make_Option_Some` is resolved by
   `resolveAdtFactory` (line 167) → `findConstructorMatches` (line 256), which scans
   `r.current.Iface` + **`r.current.Imports`** (line 278). `inst.Imports` is populated from
   `loaded.Imports` (the AST import list) at `runtime.go:191-197`. So **the runtime will not
   find the constructor unless std/option is in the entry module's import set.**

**Conclusion:** the cleanest fix is to make entry modules **implicitly import std/option and
std/result** — reusing the ENTIRE existing constructor-import path (compile-time
`resolveConstructorImport` + elaborator `RegisterConstructor` + runtime `inst.Imports`) rather
than building a parallel "prelude constructor registry." This is smaller, safer, and
inherits shadowing/dedup semantics that the import path already handles.

---

## Doc claims: VERIFIED vs CORRECTED

| Doc claim | Verdict | Evidence |
|---|---|---|
| `internal/pipeline/prelude.go` exists, injects only `println`, has AI-first philosophy comment | **VERIFIED** | prelude.go:12-55; comment lines 18-21; only `println` scheme injected (line 53) |
| Prelude is a type-level + value-level split (`InjectPrelude` + `InjectPreludeValues`) | **CORRECTED** | `InjectPreludeValues` **does NOT exist** — it is a TODO comment only (prelude.go:57-59). `println` value is resolved as a builtin via the global resolver, not injected. So there is NO existing value-injection path to copy for constructors. |
| Prelude injection is entry-module only, gated by `IsEntryModuleFromAST` | **VERIFIED** | pipeline_module_compile.go:69-71; also REPL paths repl.go:111, module_registry_load.go:135 |
| `Some`/`None`/`Ok`/`Err` require explicit import today; fails `undefined variable: Some` | **VERIFIED** | reproduced with worktree binary (see above) |
| Prelude has "shadowing semantics" for functions | **PARTIAL** | Doc comment claims it (prelude.go:28), but `println` shadowing is trivial (a scheme in the type env that a local decl overrides). There is **no existing precedent for TYPE/constructor shadowing via the prelude.** Constructor shadowing DOES exist via the import path: `pipeline_module_compile.go:104-110` ("local constructors may override imports if same name") and `resolver.go` ambiguity detection. The fix rides that path, so shadowing is inherited, not net-new. |
| The prelude has never carried types/constructors; this is the first | **VERIFIED** | prelude.go injects only the `println` function scheme |
| Files-to-Modify: `prelude.go +60`, `internal/eval/* (InjectPreludeValues) +40`, `prelude_test.go +80` | **CORRECTED** | (a) The primary change is NOT in `prelude.go` — it belongs where imports are resolved (`internal/pipeline/pipeline_module_imports.go` / `pipeline_module.go`), gated by the same entry-module check. (b) There is **no `InjectPreludeValues` in `internal/eval`** to extend — the runtime fix is adding std/option+std/result to the entry module's `inst.Imports` (runtime.go) so the existing `$adt` resolver works. (c) Tests span `internal/pipeline` AND `internal/runtime` + example + stdlib verify, not just `prelude_test.go`. |
| Custom local `type Result = Pending \| Done` must shadow prelude | **VERIFIED it works TODAY with an explicit import present** | `customshadow.ail` (local `type Result` + `import std/option`) checks clean today. The fix must preserve this with the implicit import. |
| Target v0.24.0 | **STALE** | current `std/VERSION` = v0.29.2; retarget to v0.29.x / next release. |
| AliasParams (PR #381) interaction | **NO COLLISION** | Option/Result are ADTs registered through `ctorTypes`/`ExternalTypes` factory schemes + `adtTypeParams`, a DIFFERENT env from `RegisterTypeAliasParams` (the `expandAlias *TApp` path). They do not share a namespace. M2 acceptance still asserts non-interaction. |

---

## Milestones

### M1 — Implicit import injection (entry-only)  (~130 LOC, ~0.5d)
Wire std/option + std/result as implicit dependencies of entry modules, reusing
`resolveConstructorImport`. Two injection points, both gated on `IsEntryModuleFromAST`:
- **Compile:** in `pipeline_module.go` around import resolution (line 298) — after
  `resolveModuleImports`, if entry module, load std/option + std/result ifaces and run their
  constructors/types through the same `resolveConstructorImport` + `RegisterConstructor`
  (pipeline_module.go:314) machinery. **Dedupe** against symbols the user already imported.
- **Runtime:** in `runtime.go` (`LoadAndEvaluate`, ~line 191) — for entry modules, ensure
  std/option + std/result are in `inst.Imports` even if absent from `loaded.Imports`.
- Prefer a single shared helper (e.g. `entryPreludeModules() []string`) so compile + runtime
  stay in lockstep (avoid the "guard the call-site not the helper" recurrence).

### M2 — Shadowing + match resolution  (~70 LOC, ~0.25d)
- Register implicit imports FIRST (lowest precedence) so a local `type Option`/`type Result`
  or an explicit user import overrides in `elaborator.constructors` and `ctorTypes`
  (existing "local overrides import" logic, compile.go:104-110).
- Ensure runtime `findConstructorMatches` never sees two same-type/same-ctor matches (no
  spurious ambiguity) when the user also imports or redefines.
- Assert AliasParams non-interaction.

### M3 — Runtime + full fixtures + example + docs  (~130 LOC, ~0.5d+)
- Prove `ailang run` (not just `check`) works for construct + match + Result.
- Full Conflict-Surface fixture suite (doc §4) as tests (pipeline + runtime).
- `examples/prelude_option_result.ail`; `make verify-examples`, `make verify-stdlib`, `make test` green.
- Prompt + CHANGELOG; **cache-key decision documented** (see below).
- Design doc → `implemented/`, `std/VERSION` bump.

**Phasing note:** the doc's Phase 1/2/3 (type-level / constructor+match / evaluator-values) assumes
three separate mechanisms. Because the fix reuses ONE machinery (the import path), the natural
phasing is **injection (M1) / shadowing+dedup (M2) / runtime+fixtures+docs (M3)**. Same 3
milestones, but M1 already delivers type-level AND factory-scheme AND ctor-map in one step
(they are all produced by `resolveConstructorImport`), and there is no separate
`InjectPreludeValues` to write.

---

## Cache-key decision

**No `cacheKeyVersion` bump required.** Rationale:
- `cacheKeyVersion` (`internal/pipeline/cache_key.go:24`, currently `"v3"`) is bumped only when
  the on-disk Iface gob/JSON struct shape changes (v1→v2 RecordPattern.Rest; v2→v3
  AliasParams). This sprint adds **no new persisted struct field** — Option/Result ifaces
  already serialize today.
- `ModuleCacheKey` (cache_key.go:37) folds `depDigests` into the key. Making std/option +
  std/result **implicit deps of entry modules changes the entry module's dependency set**, so
  entry-module cache keys shift automatically → stale entries invalidate naturally. Non-entry
  modules are unaffected.
- **Guard:** if implementation ends up adding a persisted field to `Iface` or `ConstructorInfo`
  (it should NOT), THEN bump to `v4` with a justification comment matching the existing style.

---

## Acceptance criteria (fixtures from doc Conflict Surface §4)

1. `import std/option (Option, Some, None)` then use them — explicit import unchanged (dedupe, no ambiguity).
2. Local `type Option[a] = Some(a) | None` — local shadows prelude, no conflict.
3. Local `type Result = Pending | Done` (different constructors) — local shadows, no clash.
4. Non-entry library module using Option WITHOUT import — STILL requires import (verified fails today at lib.ail:4:3).
5. `match opt { Some(x) => x, None => 0 }` no import — resolves.
6. No-import construct + match + Result all EXECUTE under `ailang run` (baseline output `42`).
7. `make test` / `make verify-stdlib` / `make verify-examples` green; example file added.
8. `ailang check` AND `ailang run` both covered (check-only would miss the runtime `findConstructorMatches` path).

---

## Risks / blockers

- **R1 (primary):** runtime resolution. `check` passing is NOT sufficient — the entry module
  must have std/option/std/result in `inst.Imports` or `$adt.make_Option_Some` fails at eval
  with "constructor not found in scope" (resolver.go:185). M3 MUST exercise `ailang run`, and
  M1 MUST touch both compile AND runtime injection (mirror the two-call-site lesson).
- **R2:** dedup vs ambiguity. If the user ALSO imports std/option explicitly, the implicit +
  explicit registration must not produce a duplicate `$adt` factory or an "ambiguous
  constructor" runtime error. Register implicit first + skip-if-present.
- **R3:** entry-module detection timing. `IsEntryModuleFromAST` reads the surface AST (available
  pre-elaboration), so injection can run before `ElaborateFile` (needed so the elaborator's
  `constructors` map is populated before it rewrites `Some(x)`). Verify ordering in
  pipeline_module.go (imports resolved at :298, elaborator constructors registered at :314,
  ElaborateFile at :318 — inject before :314).
- **R4 (non-blocking):** AliasParams — confirmed separate env; assert non-interaction in M2, do
  not expect a real clash.

## Files to modify (corrected)

| File | Change | ~LOC |
|---|---|---|
| `internal/pipeline/pipeline_module.go` (+ maybe `pipeline_module_imports.go`) | entry-only implicit import of std/option + std/result via existing `resolveConstructorImport`; dedupe | +80 |
| `internal/runtime/runtime.go` | entry-only std/option+std/result into `inst.Imports` | +30 |
| `internal/pipeline/prelude.go` | (optional) shared `entryPreludeModules()` helper + doc comment update | +15 |
| `internal/pipeline/*_test.go`, `internal/runtime/*_test.go` | fixture suite (6 cases, compile + runtime) | +120 |
| `examples/prelude_option_result.ail` | example | +15 |
| `prompts/<current>.md`, `CHANGELOG.md`, `std/VERSION`, docs | prompt line + changelog + version + design-doc move | +20 |

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_29_0/m-prelude-option-result-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-PRELUDE-OPTION-RESULT.json`
