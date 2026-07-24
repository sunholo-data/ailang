# Sprint Plan — M-CHECK-STRICT-FALLBACKS (Static "Ok contains default-valued literal" detection)

**Design doc**: `design_docs/planned/v1_0_0/m-check-strict-fallbacks.md`
**Status of architecture**: DECIDED by Mark 2026-07-17 ("go with 2") — do NOT re-quorum, do NOT re-open the fork.
**Base**: `origin/dev` @ `6993e5db0`. Executed in worktree `.claude/worktrees/m-check-strict-fallbacks`, branch `sprint/m-check-strict-fallbacks`.
**Diagnostic code**: `STRICT_FALLBACK_001`
**Estimated**: ~2 days (revised per Mark's directive; up from the doc-body's stale ~1d because the DECIDED Core-level + registry design is heavier than the superseded surface-AST plan).
**Risk**: medium — the ANF-let indirection (below) is the one real trap.

---

## CRITICAL: doc-premise corrections (reality-checked against HEAD `6993e5db0`)

The RETAINED doc body (below the Status header) plans a **purely-syntactic surface-AST pass**
(`DetectStrictFallbacks(astFile *ast.File)`, hooked at `pipeline_single.go:159`). **That plan is
SUPERSEDED** by the DECIDED architecture (post-name-resolution + resolved-identity registry). The
executor MUST follow the corrections below, NOT the doc body's `internal/pipeline` line-refs or its
`*ast.File`-only signature.

| # | Doc-body premise (superseded) | Verified reality at HEAD | Consequence for the plan |
|---|---|---|---|
| C1 | Pass is `DetectStrictFallbacks(astFile *ast.File)`, syntactic over surface AST | The DECIDED pass is Core-level, mirroring `warn_split_args.go`'s `DetectArgOrderWarnings(prog *core.Program)`. But suppression + return-type filter need the surface AST too. | New signature: `DetectStrictFallbacks(file *ast.File, prog *core.Program) []elaborate.Warning`. Correlate Core decls ↔ surface `FuncDecl` **by name**. |
| C2 | `Ok(x)` is a syntactic `Ok`-headed application | `Ok(x)` elaborates (`internal/elaborate/expr_calls.go:49-83`) to `core.App{Func: core.VarGlobal{Ref: GlobalRef{Module:"$adt", Name:"make_Result_Ok"}}, Args:[x]}`. `Err(x)` → `make_Result_Err`. | Detection keys on the `$adt.make_Result_Ok` VarGlobal — a **resolved constructor identity**, exactly the `swapTrapIndex` mechanism. This is the whole reason the pass is sound. |
| C3 | (absent) — surface plan reads `Ok([])` args directly | **ANF normalization** (`internal/elaborate/core.go:299`) makes App args ATOMIC. `Ok([])` → `let t = [] in make_Result_Ok(t)`; `Ok(jo([]))` → `let t = jo([]) in make_Result_Ok(t)`. `Args[0]` is `core.Var{t}`, NOT the literal/call. **Only** scalar `Lit` empties (`Ok("")`, `Ok(0)`) stay directly atomic in `Args[0]` (`core.IsAtomic` = Var/Lit/Lambda/DictRef/VarGlobal, `core.go:559`). | The detector MUST resolve `Args[0]` when it is a `core.Var` back through the **enclosing `Let` bindings** to find the real value expr. This is the single biggest implementation task (M2). |
| C4 | `jo([])` "would not flag" (gpt5-6-sol objection) | Imported `jo` → `core.App{Func: VarGlobal{Ref:{Module:"std/json", Name:"jo"}}, ...}`. A user-defined local `jo` → plain `core.Var{Name:"jo"}` (never VarGlobal). | Registry keyed by `core.GlobalRef{Module,Name}` catches `std/json.jo` and structurally excludes user-local `jo`. Objection resolved SOUNDLY. `jo` verified: `std/json.ail:3` `module std/json`, `:61` `export func jo`. |
| C5 | Pass runs in both single-file and module pipelines at `pipeline_single.go:159` / `pipeline_module.go:388` | Single-file pipeline does NOT resolve imports to VarGlobal (`warn_split_args.go:194-197` comment: "effectively a no-op today"). The MODULE pipeline DOES keep imported calls as `VarGlobal` post-lowering (`pipeline_module.go:373-389`). Any file with an `import` routes through the module pipeline (`pipeline.go:157,164-168`). | Wire the pass in the MODULE pipeline (`pipeline_module.go`, next to the `DetectArgOrderWarnings` loop ~L382-389) where `unit.Core` + `mod.File` are both available. Literal-empty (`Ok("")`/`Ok([])`) detection works in the single-file pipeline too (no import needed), so ALSO wire at `pipeline_single.go:198` for the literal cases — but `jo`-class detection there is a no-op by design. |
| C6 | `@allow_empty_ok` annotation is read "off the surface AST" (correct) but the plan assumes the pass IS surface-AST | Annotations live ONLY on `*ast.FuncDecl.Annotations` (`internal/ast/ast_decl.go:58,73` `GetAnnotation`). They are NOT carried onto Core decls. | Suppression check must read the surface `FuncDecl` (via the `file *ast.File` arg, per C1), NOT Core. Return-type `Result[_,_]` filter likewise reads the surface `FuncDecl.ReturnType`. |
| C7 | `Ok`/`Err` are always in scope | M-PRELUDE-OPTION-RESULT (v0.30.0, `examples/prelude_option_result.ail`): `Ok`/`Err` are implicit ONLY in **entry modules** (module with exported `main`). LIBRARY/PACKAGE modules must `import std/result (Result, Ok, Err)` or elaboration fails `undefined variable: Ok`. Verified empirically. | ALL package-mode fixtures (the publish-boundary case that matters) MUST include `import std/result (Result, Ok, Err)`. Entry-module fixtures may omit it. |
| C8 | Channel: dev=warning / `--package`=error | Verified. Dev: `result.Warnings []elaborate.Warning` → `printCheckWarnings` to stderr, never exits non-zero (`cmd/ailang/check.go:243,250-257`). Package: `check_package.go` runs `pipeline.Run` per file, promotes findings to `allErrors`→`failed++`→`os.Exit(1)` — the EXACT precedent is `checkInterFunctionRefs(result)` (`check_package.go:197-206,381-392`) walking `result.Modules[modID].Core`. | Dev channel = append to `result.Warnings` (M3). Package channel = new `checkStrictFallbacks(result)` helper mirroring `checkInterFunctionRefs`, walking `result.Modules[modID].Core` + `.File`, appended to `allErrors` (M4). |
| C9 | Target dir `v0_21_0`; changelog file guess | Item now lives under `design_docs/planned/v1_0_0/`. Changelog `[Unreleased]` is `changelogs/v0.18-current.md:5`. | Doc + changelog paths corrected in M5. |

**Net architectural shape (executor: build THIS):** one Core+surface pass
`DetectStrictFallbacks(file *ast.File, prog *core.Program) []elaborate.Warning` in a new
`internal/pipeline/strict_fallbacks.go`, structured after `warn_split_args.go`. A registry
`knownEmptyBuilders` keyed by `core.GlobalRef` (like `swapTrapIndex`). Wired for warnings in both
pipelines; wired for hard-error in `check_package.go`. Suppression + return-type filter read the
surface `FuncDecl`.

---

## Registry design (the DECIDED "curated known-empty-builder registry")

```go
// strict_fallbacks.go — mirror of swapTrapIndex in warn_split_args.go.

// knownEmptyBuilder describes a resolved builder call that yields a semantically
// "empty" value when invoked with an empty argument (e.g. std/json.jo([]) => {}).
type knownEmptyBuilder struct {
    module string // e.g. "std/json"
    name   string // e.g. "jo"
    // isEmptyCall reports whether THIS particular App{VarGlobal, args} is the
    // empty form (e.g. jo with a single empty-list arg). Keeps it sound:
    // jo([realKV]) is NOT flagged.
    isEmptyCall func(app *core.App, resolve resolveFn) bool
}

// keyed by GlobalRef{Module,Name}, built once — identical pattern to swapTrapIndex.
var knownEmptyBuilders = map[core.GlobalRef]knownEmptyBuilder{
    {Module: "std/json", Name: "jo"}: {module: "std/json", name: "jo", isEmptyCall: joEmpty},
    // future entries added here (extensible table, like swapTraps).
}
```

- Keyed by **module-qualified resolved identity** `core.GlobalRef{Module,Name}` — NEVER bare name.
- `resolveFn` is the let-chain resolver (M2) so `isEmptyCall` can peer through ANF `Var` args
  (e.g. `jo(t)` where `t` binds `[]`).
- A local user `jo` is a `core.Var`, has no `GlobalRef`, never hits the map → structurally excluded.

**Literal-empty detection** (Pattern A literals + Pattern B/C zero-records) is separate from the
registry and handled by classifying the resolved `Args[0]` value directly:
- `core.Lit{Kind:StringLit, Value:""}` → empty string
- `core.Lit{Kind:IntLit, Value:0}` / `FloatLit 0.0` / `BoolLit false` → zero scalar (only flagged in
  fallback/match-arm context — see "what does NOT flag" in the doc; keep the doc's match-arm guard)
- `core.List{Elements: []}` → empty list
- `core.Record{Fields: {}}` or `core.Record` where every field value resolves to a zero literal → empty/zero record
- `core.App{Func: VarGlobal{ctor $adt.make_*}}` with all args resolving to zero literals → zero-ctor (Pattern C)

---

## Milestones

### M1 — Parser: `@allow_empty_ok("rationale")` annotation whitelist (~0.25 day, ~40 LOC + test)

**Files:**
- `internal/parser/parser_decl.go` — add `case "allow_empty_ok":` to the `parseAnnotation()` name-keyed switch (verified additive; existing cases `verify`/`route`/`mcp_name`/`raw` at `parser_decl.go:30-36`). Require a single string-literal rationale arg; reject empty/missing rationale.
- `internal/parser/route_attr_test.go` — parser test: `@allow_empty_ok("reason")` parses onto `FuncDecl.Annotations`; `@allow_empty_ok()` (no rationale) is a parse error.

**Acceptance:**
- [ ] `@allow_empty_ok("x")` parses; retrievable via `FuncDecl.GetAnnotation("allow_empty_ok")` with the rationale string.
- [ ] `@allow_empty_ok()` → parse error (rationale required).
- [ ] `go test ./internal/parser/...` green.

**Independently testable:** yes (parser-only).

---

### M2 — Core let-chain resolver + literal-empty classifier (~0.5 day, ~120 LOC + tests)

The engine that peers through ANF. Pure, no wiring yet.

**Files:**
- `internal/pipeline/strict_fallbacks.go` (new) —
  - `resolveAtomic(v core.CoreExpr, env map[string]core.CoreExpr) core.CoreExpr`: if `v` is
    `core.Var{Name}` and `Name` is in `env` (the accumulated enclosing `Let` bindings), return the
    bound value (recursively). Else return `v`.
  - `isEmptyValue(v core.CoreExpr, env, depth) (bool, string)`: classify the RESOLVED value as
    empty/zero (string/list/record/zero-scalar/zero-ctor) per the registry-design list above. Bound
    recursion depth (guard cycles — `depth` cap, mirror the codebase's cycle-safety rule).
  - `joEmpty(app *core.App, resolve) bool` and the `knownEmptyBuilders` map.
- `internal/pipeline/strict_fallbacks_test.go` (new) — unit tests for `isEmptyValue`/`resolveAtomic`
  on hand-built Core: `Var→[]` resolves+flags; `Var→jo([])` (VarGlobal `std/json.jo`) flags via
  registry; `Var→jo([kv])` does NOT flag; direct `Lit ""` flags; `Lit "real"` does not;
  `Record{name:"",age:0}` flags; `Record{name:Var(real)}` does not; `Var→core.Var(userLocalJo call)`
  i.e. a plain `core.Var` head does NOT flag.

**Acceptance:**
- [ ] Resolver walks `let t = <val> in ...` and returns `<val>` for `Var{t}`.
- [ ] Registry lookup keyed by `GlobalRef{"std/json","jo"}` fires only on the empty form.
- [ ] Zero false-positive on non-empty/non-literal values and on plain-`Var`-headed apps.
- [ ] Depth-bounded (no infinite loop on pathological let-chains).

**Independently testable:** yes (pure functions on synthetic Core).

---

### M3 — The pass + dev-channel (WARNING) wiring (~0.5 day, ~140 LOC + tests)

**Files:**
- `internal/pipeline/strict_fallbacks.go` —
  - `StrictFallbackWarning` type satisfying `elaborate.Warning` (`Position()`, `String()`), rendering
    the `STRICT_FALLBACK_001` diagnostic from the doc (file:line:col + fix suggestion + `@allow_empty_ok`
    hint). Use `app.OriginalSpan()` for location (as `warn_split_args.go:119` does).
  - `DetectStrictFallbacks(file *ast.File, prog *core.Program) []elaborate.Warning`:
    1. Build a `name → *ast.FuncDecl` map from `file` for functions whose declared return type is
       `Result[_,_]` AND that lack `@allow_empty_ok` (read `FuncDecl.ReturnType` + `GetAnnotation`).
    2. Walk `prog.Decls`; for each top-level `Let`/`LetRec` binding whose name is in that map, walk the
       body accumulating enclosing `Let` bindings into `env` (reuse the `warn_split_args.go` walk
       skeleton — extend it to thread `env` and to match `$adt.make_Result_Ok` App nodes).
    3. On a match: resolve `Args[0]` via M2; if empty/zero (literal OR registry), emit
       `STRICT_FALLBACK_001`.
- `internal/pipeline/pipeline_single.go` — after L198 (`DetectArgOrderWarnings`), append
  `DetectStrictFallbacks(astFile, coreProg)` to `result.Warnings`. (Literal-empty cases work here;
  `jo`-class is a no-op in single-file per C5 — acceptable.)
- `internal/pipeline/pipeline_module.go` — in the `DetectArgOrderWarnings` loop (~L382-389), also
  append `DetectStrictFallbacks(compiledUnits[modID].Core... )` — NOTE the module pipeline exposes
  surface AST as `compiledUnits[modID]`'s module file. Executor: confirm the `*ast.File` handle
  inside the compile loop (`mod.File`, elaborated at `pipeline_module.go:318`); pass BOTH `mod.File`
  and `unit.Core`. Deterministic sorted-module order (loop already sorts, L382-386).
- `internal/pipeline/strict_fallbacks_test.go` — end-to-end via `pipeline.Run` on fixtures:
  - `Ok("")`, `Ok([])`, `Ok(jo([]))`, `Ok({name:"",age:0})` each emit exactly one warning on the right line.
  - `Ok("real")`, `Ok(userLocalJo([]))` (local `func jo`), `Ok({name:realVar})` emit NONE.
  - Same patterns in a NON-`Result`-returning function emit none (return-type filter).
  - `@allow_empty_ok("reason")` on the function suppresses.

**Acceptance:**
- [ ] `ailang check fixture.ail` (lib module, imports `std/result` + `std/json`) prints
      `STRICT_FALLBACK_001` as a yellow warning and **exits 0**.
- [ ] Negative fixtures (`Ok("real")`, local-`jo`, annotated) print nothing.
- [ ] `go test ./internal/pipeline/...` green.

**Independently testable:** yes.

---

### M4 — Package-channel (HARD ERROR, exit 1) wiring (~0.25 day, ~50 LOC + test)

**Files:**
- `cmd/ailang/check_package.go` — add `checkStrictFallbacks(result pipeline.Result) []string`
  mirroring `checkInterFunctionRefs` (`:381`): iterate `result.Modules[modID]`, call
  `DetectStrictFallbacks(mod.File, mod.Core)`, format each warning as an error string. Invoke it in
  the per-file loop right after the `checkInterFunctionRefs` block (`:197-206`): if it returns
  findings, append to `allErrors`, `failed++`, print `✗`, `continue`. Existing `failed>0 → os.Exit(1)`
  (`:263-264`, `:282-301`) then fails the publish.
- `cmd/ailang/check_package_test.go` — a package fixture (with `ailang.toml`) containing an
  unannotated `Ok(jo([]))` fails `check --package` with exit 1 and a `STRICT_FALLBACK_001` message;
  the same fixture with `@allow_empty_ok("...")` passes (exit 0).

**Acceptance:**
- [ ] `ailang check --package <dir-with-violation>` exits 1, lists `STRICT_FALLBACK_001`.
- [ ] `ailang check --package <dir-annotated>` exits 0.
- [ ] Plain `ailang check` on the same file stays exit-0-with-warning (regression guard).
- [ ] `go test ./cmd/ailang/...` green.

**Independently testable:** yes (needs M2+M3).

---

### M5 — Stdlib audit, fixtures, docs, examples, CHANGELOG (~0.5 day, ~100 LOC + docs)

**Files:**
- Stdlib audit: run the pass over `std/*.ail`; add `@allow_empty_ok("...")` where an empty-Ok is
  legitimate, refactor to `Err(...)` where not. Expected touch points ~5-10 fns (`std/result`,
  `std/option`, `std/json`). MUST leave `make verify-examples` / `make ci` green.
- `internal/pipeline/testdata/strict_fallbacks/` — fixtures: `firestore_v071_getdoc.ail` (flags),
  `firestore_v072_getdoc.ail` (no flag), `firestore_listdocs_annotated.ail` (`@allow_empty_ok`, no
  flag). All lib-module fixtures import `std/result` (per C7).
- `examples/strict_fallbacks_demo.ail` — shows a violation AND the `@allow_empty_ok` opt-out. Verify
  it type-checks (`ailang check`); if it intentionally demonstrates a warning it must still compile.
- `docs/docs/guides/strict-fallbacks.md` — author-facing guide (the annotation, the two channels,
  the diagnostic).
- `changelogs/v0.18-current.md` — `[Unreleased]` entry (verified location, C9).

**Acceptance:**
- [ ] Pass over stdlib: zero unannotated violations after audit.
- [ ] `make ci` + `make verify-examples` green.
- [ ] Example + guide + changelog shipped.

**Independently testable:** partially (audit depends on M3/M4 binary).

---

## Day-by-day

**Day 1**
- AM: M1 (parser annotation) → M2 (resolver + classifier + registry, the ANF core).
- PM: M3 (pass + dev-channel wiring + fixture tests). Checkpoint: `ailang check` warns on the
  4 positive patterns, silent on the 3 negatives.

**Day 2**
- AM: M4 (package hard-error channel + test). Checkpoint: `check --package` exits 1 on violation,
  0 on annotated.
- PM: M5 (stdlib audit, fixtures, example, guide, CHANGELOG). `make ci`. Commit per milestone,
  final commit references any linked issue.

---

## Acceptance criteria (sprint-level)

- [ ] `STRICT_FALLBACK_001` fires on `Ok("")`, `Ok([])`, `Ok(jo([]))`, `Ok({name:"",age:0})` in a
      `Result`-returning function.
- [ ] Does NOT fire on `Ok("real")`, `Ok(userLocalJo([]))` (user-local `jo`), `Ok({name:realVar})`,
      or any pattern in a non-`Result` function.
- [ ] `@allow_empty_ok("rationale")` suppresses; missing rationale is a parse error.
- [ ] `ailang check file.ail` → warning, exit 0. `ailang check --package dir` → error, exit 1.
- [ ] Registry keyed by `core.GlobalRef{Module,Name}` (module-qualified), extensible table like
      `swapTraps`.
- [ ] Stdlib passes clean after audit. `make ci` green.
- [ ] Example + `docs/docs/guides/strict-fallbacks.md` + CHANGELOG shipped.

---

## Risks the executor MUST watch

1. **ANF let-indirection (highest risk).** `Ok(jo([]))` / `Ok([])` args are `core.Var`, not the
   literal/call. If M2's resolver is skipped, the pass silently detects NOTHING for the motivating
   `jo([])` incident and for empty-list/record literals — a false-green that defeats the whole
   sprint. The `warn_split_args` walk does NOT thread a let-env; you MUST add one. Write the M2
   let-resolver test FIRST and prove it on `Ok(jo([]))` before wiring anything.
2. **Single-file vs module pipeline (C5).** `jo`-class detection only works where imports resolve to
   `VarGlobal` = the MODULE pipeline. Don't be surprised when a single-file `ailang check` on a file
   that imports `jo` still catches it — that's fine (import ⇒ module pipeline). But a hand-built
   single-file Core in a unit test will NOT have the VarGlobal; use `pipeline.Run` on real fixtures
   for the `jo` cases, synthetic Core only for the resolver units.
3. **Prelude scope (C7).** Library fixtures without `import std/result` fail elaboration with
   `undefined variable: Ok` BEFORE the pass runs — you'll get a confusing "no warning" that's
   actually a compile error. Always import `std/result` in lib fixtures.
4. **Zero-scalar over-flagging.** `Ok(0)`/`Ok("")` where the value is legitimately the success value
   (not a fallback) must NOT flag outside match-arm/fallback context — honor the doc's "what does NOT
   flag" guard. Keep the match-arm-context restriction for bare scalars; empty collections/records
   and registry-builders can flag unconditionally.
5. **Annotation not on Core (C6).** Suppression + return-type filter read the surface `FuncDecl` via
   the `file *ast.File` arg. Don't look for annotations on Core decls — they aren't there.
6. **Determinism.** Iterate modules/decls/warnings in sorted order (the `DetectArgOrderWarnings` loop
   already sorts module IDs — match that) so diagnostic output is stable for golden tests.

---

## Velocity note

Recent comparable post-name-resolution warning passes: M-DX-SPLIT-ARG (`warn_split_args.go`, the
direct template — registry + Core walk + two-pipeline wiring) and `checkInterFunctionRefs` (the
package-mode error-channel template) are both already in-tree and small (~150-220 LOC each). This
sprint is essentially "combine those two patterns + add an ANF let-resolver." The ~2d estimate (vs
the stale ~1d in the doc body) is driven by the ANF indirection (M2) and the dual-channel wiring
that the superseded surface-AST plan didn't need. Medium confidence; the one unknown is stdlib-audit
churn (M5), bounded by the ~5-10 expected touch points.

SPRINT_PLAN_PATH: design_docs/planned/v1_0_0/m-check-strict-fallbacks-sprint-plan.md
