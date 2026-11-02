# M-VM-ENTRY-MODULE-SHADOWING: `ailang run --bytecode --entry main` resolves an imported module's `main`, not the entry module's

**Status**: Planned
**Target**: v0.55.1
**Priority**: P0 (High)
**Estimated**: 3 days (1.5 honest × 2 convention)
**Dependencies**: None (extends the M-BYTECODE-MULTIMODULE image layout shipped in v0.11.0)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Which function `--entry main` executes stops depending on lexicographic module-path order; same file layout always resolves the same entry |
| A2: Replayability | 0 | No effect on trace semantics; the *correct* function is what gets traced |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Entry/args agreement becomes machine-checkable at dispatch instead of failing mid-run with a misleading GET_FIELD fault |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Ambiguous resolution becomes a structured, enumerated error instead of a silent wrong-function run |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No resource changes |
| A10: Composability | +1 | Multi-tool packages (generator `main` + verifier `main`) become a supported composition instead of a trap |
| A11: Structured Failure | +1 | The ambiguous case fails loudly with candidates listed, before any effect is attempted |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Removes an order-dependent silent pick (no new nondeterminism)
- [x] A3 (Effects): No hidden side effects introduced
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Error enumerates candidates; not optimized for human convenience

## Problem Statement

**Reported (v0.54.18, darwin arm64; task `task-582a4a2b`; game clone `stapledons-godot-worlds`, branch `design/r3-worlds`, draft PR371):** the R3-WORLDS exoplanet whole-data verifier fails reproducibly on the bytecode VM while the interpreter and the generator are both correct:

```bash
runtime/bin/ailang run --quiet --bytecode --caps IO,FS --package-dir sim \
  --entry main --args-json '{"input":"data/raw/exoplanets_ps_100pc_20261102.csv","registry":"data/starmap/stars.json"}' \
  sim/tools/exoplanets_verify.ail
```

```
Error: vm: vm: GET_FIELD: index 4 exceeds field count 2
  (in pkg/stapledons/sim/tools/exoplanets.main at .../sim/tools/exoplanets.ail:103, ip4, op GET_FIELD)
```

The layout (committed, reproducible after `tools/exoplanets.sh fetch`):

- **Entry** `sim/tools/exoplanets_verify.ail` (`module stapledons/sim/tools/exoplanets_verify`): `main(a: {input, registry})` — a **2-field** record.
- It imports `./exoplanets` (`build`, `parseRegistry`) and `stapledons/sim/data/exoplanets` (`planets`, `snapshotSha256`).
- The imported `sim/tools/exoplanets.ail` is the **generator**, with its own exported `main(a: {input, registry, output, moduleOut, differences})` — a **5-field** record.
- The generated data module imports `stapledons/sim/tools/exoplanets` (`Planet`, `archivePlanet`).

Note the failing frame is **attributed to the generator module** (`pkg/stapledons/sim/tools/exoplanets.main`, `exoplanets.ail:103`) even though `--entry main` was requested on `exoplanets_verify.ail`. The reporter's generic three-module diamond with two `main`s did **not** reproduce (analysis of why below — the bug is order-dependent, which is exactly why it survived the generic test).

**Current state (mechanism, verified by reading the dispatch — see Verification Log rows 1–8):**

- `FindEntryProto` (`internal/runner/vm.go:25-37`) resolves `--entry main` by **scanning the image's prototypes in insertion order** and returning the first prototype whose name is `main`, or merely **ends with `.main` or `_main`**. Module identity is never consulted.
- In a multi-module image every prototype is module-qualified (`canonicalFuncName`, `internal/bytecode/compiler/compiler.go:293-298`: `Module + "." + Name`), so a bare `main` never matches exactly — the first `.main` **suffix** match wins.
- Insertion order is **lexicographic module-ID order** (`CompileBytecodeFromResult`, `internal/runner/vm.go:66-73`: `sort.Strings(modIDs)`), and the entry module's ID is the canonical **file path** (`loader.CanonicalModuleID(src.Filename)`, `internal/pipeline/pipeline_module_phases.go:227`) while its sibling dependencies are rewritten to `pkg/<package>/…` IDs (`internal/loader/loader.go`, `extractImports`). `"pkg/…"` sorts before `"sim/…"`, so **every** `pkg/`-canonical dependency's `main` is inserted before the entry module's `main`.
- The generator's `main` therefore shadows the verifier's `main`. The CLI-level gates all pass **because they resolve against the entry module correctly**: entry existence/type/arity come from `params.Iface.Exports["main"]` and `inst.GetExport("main")` (`internal/runner/entrypoint.go:62, 117-131`), and the single-record parameter means the VM-side arity check `proto.NumParams == len(bcArgs)` compares 1 == 1 (`internal/runner/vm.go:190-196`). The 2-field args record is then handed to the generator's 5-field `main`, which faults on `GET_FIELD index 4` (`internal/vm/vm.go:531`) at `ip4`.
- The **interpreter** path is unaffected: it calls the evaluator value from `inst.GetExport(entry)` — the entry module's instance — so `ailang run` without `--bytecode` runs the verifier's `main`. Hence "passes on interpreter, fails on strict VM/bytecode".

**Impact:**

- **Blocking** for the R3-WORLDS acceptance: the compiled-data verifier cannot run on the VM at all. The retained failing state (`exoplanets-check`, no rename, no interpreter production fallback) stays broken.
- **Silent wrong-function execution**, not just a crash: any package where a dependency exports a `main` with a *compatible-enough* argument shape runs the wrong `main` to completion and returns its result. `--entry main` on a multi-tool package is currently a lottery on module-path spelling.
- Affects every `--bytecode` consumer of multi-module packages: `ailang run` (incl. `--batch`, which routes through the same `ExecuteModuleEntrypoint`, Verification Log row 13), `run --policy` entries, `ailang repl`, `ailang select_best`, and the bytecode test engine (`internal/testing/bytecode_engine.go:128`), which shares the resolver.

## Goals

**Primary Goal:** `ailang run --bytecode --entry <name>` executes the entry **module's** `<name>` when it exists, and fails loudly (enumerating candidates) when the name is ambiguous across modules — reproducibly, independent of module-ID spelling and sort order.

**Success Metrics:**

- The committed stapledons verifier layout (`sim/tools/exoplanets_verify.ail` importing a sibling that exports its own 5-field `main`) runs on `--bytecode` and `--strict-bytecode` with output identical to the interpreter (today: hard VM error attributed to the wrong module).
- An image with two same-named entries in different modules and **no** match in the entry module exits with a structured error naming both candidates (today: silently runs the lexicographically first).
- Zero change to prototype naming, image layout, lowering order, or disassembly output — the fix is confined to entry resolution (`internal/runner/vm.go`), plus one new informational field on `pipeline.Result`.
- All existing strict-VM parity tests pass unchanged (no fixture churn).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Entry resolution is keyed by **module identity** (entry module's prototype preferred), not name-suffix scan order | This is the bug: which function runs must not depend on lexicographic module order | human (this doc) | design | med |
| Ambiguous resolution (entry name absent from the entry module, multiple suffix matches) is a **hard error** listing candidates, not a silent first-match | Silent wrong-function execution is the failure mode being fixed; CLAUDE.md principle 2 (no silent fallbacks) | human (this doc) | design | med |
| The legacy `_<name>` suffix tolerance is **removed** from entry resolution | It matches unrelated functions (e.g. `run_main` for entry `main`); already documented ambiguous and deliberately excluded by the batch resolver | human (this doc) | design | low |
| The entry module's canonical ID is plumbed via a new `pipeline.Result.RootModuleID` field (single source of truth) rather than re-derived by the runner | Duplicating `CanonicalModuleID` derivation in two packages invites drift if the ID scheme changes | agent (implementer may instead re-derive in the runner if plumbing proves invasive, provided a test pins the equivalence) | design | low |
| Unique-suffix fallback is **kept** for entries that genuinely name an imported function (entry absent from the entry module) | Back-compat: `--entry <depfunc>` and the testing engine's synthesized entries rely on it; only the *multi-match* case becomes an error | human (this doc) | design | med |
| No new `MODxxx`/`VMxxx` error code; the ambiguity error is a plain runner `fmt.Errorf` like the existing `entry function %q not found in bytecode image` | Runner-level dispatch errors are not in the check-time diagnostic registry; no allocation question arises | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Entry resolution prefers the entry module's own prototype (row 1)
- [x] Ambiguity is a hard error with enumerated candidates (row 2)
- [x] `_<name>` suffix tolerance removed from entry resolution (row 3)
- [x] Unique-suffix fallback retained for dep-named entries (row 5)

## Solution Design

### Overview

Add the entry module's canonical ID to the pipeline result, and replace `FindEntryProto`'s order-sensitive suffix scan with an explicit resolution policy: **entry module first, exact bare/qualified names second, unique suffix match last, ambiguity → structured error**. The evaluator path is already correct and is not touched. The image layout, prototype naming, and lowering order are not touched — `disasm` output and determinism tests are unaffected.

### Architecture

**Components:**

1. **`pipeline.Result.RootModuleID string`** (`internal/pipeline/pipeline.go`): set in `validateAndSort` right after `st.rootCanonical` is computed (`internal/pipeline/pipeline_module_phases.go:227-228`). It is the exact key under which the entry module is stored in `res.Modules` (Verification Log row 4), so it can be prefixed onto the entry name to address the entry module's prototype directly. Left empty by `runSingle` (bare-expression runs have no modules and no VM dispatch).
2. **`ResolveEntryProto(img *bytecode.BytecodeImage, rootModuleID, name string) (*bytecode.FuncPrototype, error)`** (`internal/runner/vm.go`), the new policy, in order:
   1. **Entry module exact**: if `rootModuleID != ""`, a prototype named `rootModuleID + "." + name` → return it. (This is the case the bug report hits: the verifier's own `main`.)
   2. **Exact name**: a prototype named exactly `name` → return it. Covers single-module images (`FuncDecl.Module == ""`, bare names) and callers that pass an already-qualified entry (e.g. `--entry pkg/stapledons/sim/tools/exoplanets_verify.main`).
   3. **Unique suffix**: prototypes whose name ends in `"." + name`:
      - exactly one → return it (back-compat: `--entry <depfunc>`, testing-engine synthesized entries);
      - more than one → **error**: `entry %q is ambiguous across modules: <c1>, <c2>, … (pass a module-qualified name or run the module that defines it)` — enumerated, deterministic order (image order), nothing executed;
      - zero → not-found error (existing wording plus the entry module ID for context).
   4. The `_<name>` suffix form is gone (row 3).
3. **Call-site updates**: `tryRunEntryViaVM` (`internal/runner/vm.go:175`) passes `params.PipelineResult.RootModuleID`; the test engine (`internal/testing/bytecode_engine.go:128`) does the same; `FindEntryProto` remains as a deprecated thin wrapper (bare exact + suffix scan) only if an external caller still needs it — otherwise it is deleted and its two test callers (`vm_bench_test.go:68`, `vm_quickwins_test.go:77`) migrate to `ResolveEntryProto` with the root ID they already have in scope.

### Implementation Plan

**Phase 1: Resolution policy + plumbing** (~4 hours)
- [ ] Add `RootModuleID` to `pipeline.Result`; set it in `validateAndSort`; confirm empty on `runSingle`.
- [ ] Implement `ResolveEntryProto` with the 4-step policy; unit table test covering every branch (root hit, bare hit, unique suffix hit, ambiguous error, not found, qualified user input).
- [ ] Wire `tryRunEntryViaVM` and the bytecode test engine to it.

**Phase 2: Incident regression + ambiguity fixtures** (~4 hours)
- [ ] New CLI-level regression test (pattern: `cmd/ailang/bytecode_long_list_literal_test.go`, `buildAilang` + `runWithStdin`): a tempdir pair — `gen.ail` exporting `main(a: {five fields})` and `<verify>.ail` importing it and exporting `main(a: {two fields})`, with the dependency's module ID sorting **before** the entry module's (mirror the incident: import path spelling vs canonical file path). Assert interpreter and `--strict-bytecode` outputs are identical **and** come from the verifier's `main`.
- [ ] Ambiguity fixture: entry module without the entry name, two dependency modules exporting it → assert the structured ambiguity error, exit ≠ 0, and no effect attempts (caps denied/allowed output unchanged).
- [ ] Regression-surface sweep: the "Programs that MUST still work" list below runs unchanged under `--strict-bytecode`.

**Phase 3: Docs + parity** (~2 hours)
- [ ] CHANGELOG entry (v0.55.1, Fixed — bytecode VM correctness).
- [ ] `docs/docs/reference/limitations.md` or VM reference note: multiple `main`s per package are legal; VM entry resolution is module-scoped since this fix.
- [ ] Full `make test`; game-clone re-run is a downstream step (see Success Criteria).

### Pipeline Pass Coverage

Not applicable — this design adds **no compiler pipeline pass and touches no lowering**: the AST/Core artifacts, `LowerProgram`, `LowerMultiModule`, and `compiler.Compile` are unchanged. The change is confined to the runner's *dispatch* of an already-compiled image.

### Files to Modify/Create

**New files:**
- `cmd/ailang/vm_entry_shadowing_test.go` (~140 LOC) — incident regression + ambiguity fixtures (CLI-level, strict VM).

**Modified files:**
- `internal/pipeline/pipeline.go` (+4 LOC) — `RootModuleID` field on `Result`.
- `internal/pipeline/pipeline_module_phases.go` (+2 LOC) — set `RootModuleID` from `st.rootCanonical`.
- `internal/runner/vm.go` (+55/−15 LOC) — `ResolveEntryProto`, call-site update, `FindEntryProto` wrapper or removal.
- `internal/runner/vm_bench_test.go`, `internal/runner/vm_quickwins_test.go` (+~10 LOC) — migrate to `ResolveEntryProto`.
- `internal/testing/bytecode_engine.go` (+~4 LOC) — pass the root ID.
- `internal/runner/vm_entry_test.go` (~120 LOC, new) — unit table test for the policy (if placed in `internal/runner` rather than cmd).

## Examples

### Example 1: The incident (entry module's `main` shadowed by an imported `main`)

**Before** — image prototypes (sorted module IDs; entry = `sim/tools/exoplanets_verify`, args decoded from the verifier's 2-field type):

```
pkg/stapledons/sim/data/exoplanets.planets      ← data module (no main)
pkg/stapledons/sim/tools/exoplanets.main         ← generator main, 5-field record
pkg/stapledons/sim/tools/exoplanets.build        ← generator helpers
sim/tools/exoplanets_verify.main                ← verifier main, 2-field record (requested)
std/json.decode …
```

`FindEntryProto(img, "main")` scans in this order, hits the **first `.main` suffix** → the generator's `main`; runs it with the verifier's 2-field args → `GET_FIELD index 4 exceeds field count 2` at `exoplanets.ail:103`.

**After** — `ResolveEntryProto(img, "sim/tools/exoplanets_verify", "main")`:

1. `sim/tools/exoplanets_verify.main` exists → return it. The verifier's `main` runs with its 2-field record; output matches the interpreter.

### Example 2: Ambiguity becomes a loud, structured error

```ailang
module pkgdemo/tools/report          -- exports main
module pkgdemo/tools/report_probe     -- imports ./report, NO main of its own
```

**Before:** `ailang run --bytecode --entry main report_probe.ail` silently executes `report.main` (the only suffix match — actually correct here by luck of uniqueness).

With **two** imported `main`s and no `main` in the entry module:

**After:**

```
Error: entry "main" is ambiguous across modules: pkgdemo/tools/report.main, pkgdemo/tools/audit.main
  (pass a module-qualified --entry or run the module that defines it)
```

Exit non-zero before any effect is attempted; under `--strict-bytecode` the same error surfaces without fallback.

## Success Criteria

- [ ] The committed stapledons layout (verifier importing a sibling that exports its own differently-shaped `main`) runs on `--bytecode` and `--strict-bytecode`, matching the interpreter byte-for-byte on stdout (CLI-level regression test).
- [ ] Ambiguous entry (absent from entry module, ≥2 dependency matches) exits non-zero with an error enumerating every candidate; no effect attempted.
- [ ] `_<name>` suffix no longer resolves entries (unit test: a prototype `foo_run_main` is *not* matched by `--entry main`).
- [ ] Unique-suffix fallback still resolves a dep-named entry (unit test) — the testing engine's `__namedtest_entry` dispatch keeps working.
- [ ] All tests passing (`make test`), including `internal/bytecode/compiler/multimodule_test.go` (naming unchanged) and the strict-VM parity suite.
- [ ] Documentation updated (CHANGELOG entry; VM reference note on multi-`main` packages).
- [ ] Downstream (tracked by the reporter, not this sprint): bump `stapledons-godot-worlds` to the fixed binary and re-run the R3-WORLDS exoplanet `exoplanets-check` + native5 tests.

Engine parity means the **evaluator and the strict VM agree** on the verifier's output; generated-Go parity is not required for this fix.

## Conflict Surface

*This design touches `internal/runner` dispatch (semantics of `--entry`) but no parser/typechecker/lowering position. Enumerated anyway — the resolution policy is a semantic change to a name-matching position shared by several callers.*

### Syntactic positions touched

- The `--entry <name>` value of `ailang run` (`cmd/ailang/main_run.go:40`), `run --policy` (`run_policy.go:266 Entry`), `ailang repl` (`repl.go:77` passes `"main"`), `ailang select_best` (`select_best.go:26`), and the testing engine's synthesized entry (`internal/testing/bytecode_engine.go:40` `__namedtest_entry`).

### What else lives here

| Form of `--entry` seen today | Where it comes from | Resolution after this change |
|---|---|---|
| bare `main` | default; the overwhelmingly common case | entry module's `main` if it has one; else unique suffix; else error |
| bare non-`main` export (`greet`) | `examples/intra_package_imports` service-style tools | exact/unique-suffix as today (single-module images: exact) |
| module-qualified `path.to.func` | policy files, power users | exact match, unchanged |
| `<anything>_main` | **nobody legitimate** — the tolerance was a hedge for a spelling that `canonicalFuncName` never produces (it always uses `.`) | **no longer resolves** (deliberate change) |

### Disambiguation strategy

Policy order (root → exact → unique-suffix → error) is total: at most one prototype can be returned, and every multi-match case is the error. There is no input for which the old and new policies both succeed but pick different prototypes *except* the bug being fixed (root module's `main` shadowed) — which is the point.

### Programs that MUST still work

- `examples/multi_module/main.ail` — `--entry main`, dep `core/types` has no `main` (resolution moves from suffix-scan to root-exact; same prototype).
- `examples/intra_package_imports/service.ail` — `--entry greet` with `--package-dir` (dep `types.ail` has no `greet`).
- `examples/imports/symbol_alias.ail`, `examples/imports/module_alias.ail`, `examples/imports/combined_alias.ail` — entry functions reached through exact/unique-suffix matching today.
- `internal/bytecode/compiler/multimodule_test.go` — pins `main.run` / `lib/math.double` canonical naming (untouched).
- `cmd/ailang/bytecode_long_list_literal_test.go` and the strict-VM parity tests — single-module images resolve via exact bare names (unchanged).

### What deliberately changes

- An entry name that matches **multiple** dependency prototypes and is absent from the entry module now **errors** instead of running the lexicographically-first match.
- The `_<name>` suffix tolerance is removed (see table above); any script relying on it was relying on matching an *unrelated* function.
- When the entry module itself defines the entry name **and** a dependency defines the same name, the entry module wins (previously: the dependency won whenever it sorted first — the incident).

## Testing Strategy

**Unit tests:**
- `ResolveEntryProto` table: root-module hit, bare exact hit, unique-suffix hit, ambiguous → error with candidate list, not-found, qualified input, empty `RootModuleID` (single-module) fallthrough, `_<name>` rejection.

**Integration tests:**
- CLI-level incident regression (Phase 2): tempdir two-`main` layout with dependency sorting first; interpreter vs `--strict-bytecode` parity on the **verifier's** output.
- Ambiguity fixture: structured error, exit code, no effects attempted.
- Batch mode (`ailang run --batch --bytecode`) on the same layout — same entry resolution through `ExecuteModuleEntrypoint`.

**Regression-surface tests:**
- One run per "Programs that MUST still work" entry under `--strict-bytecode`; failures there are regressions, not test churn.

**Manual testing:**
- After landing: re-run the stapledons `exoplanets-check` on a bumped `runtime/bin/ailang` (downstream, reporter-tracked — this container has no Go toolchain and no game clone, so the live repro is the reporter's transcript, Verification Log row 0).

## Deferred Decisions

- Placement of the CLI-level regression test (`cmd/ailang` vs `internal/runner`) and the fixture spelling (bare imports vs a tempdir `ailang.toml` package) — agent may choose, as long as the dependency's module ID provably sorts before the entry's.
- Whether `FindEntryProto` is deleted outright or kept as a deprecated wrapper — agent may choose once all callers are migrated.
- Exact wording of the ambiguity error — agent may choose (must enumerate candidates and suggest a qualified `--entry`).
- Whether `internal/testing/named_batch.go`'s private `findBatchedProto` migrates to the shared resolver — agent may choose; it already excludes the `_` form and is single-module today.

## Non-Goals

**Not in this feature:**
- Forbidding or linting multiple `main` exports per package — the generator+verifier layout is legitimate tool composition; the fix is correct resolution, not policing. (A package-level info diagnostic could follow in `ailang pkg lint` if it ever earns its keep.)
- Any change to evaluator-side entry resolution — it is already module-scoped and correct.
- Any change to image layout, prototype naming, lowering order, or `disasm` output.
- Import-vs-import name binding (types/values/constructors, evaluator-side) — covered by [M-IMPORT-VS-IMPORT-COLLISION](../v0_54_20/m-import-vs-import-collision.md), a sibling failure class in the same package layout.
- Fixing strict-bytecode fallback semantics — untouched.

## Timeline

**Week 1** (10 hours):
- Phase 1: plumbing + resolver + unit tests (4h)
- Phase 2: incident regression + ambiguity fixtures (4h)
- Phase 3: docs + full parity sweep (2h)

**Total: ~10 hours across 1 week** (3-day estimate at convention ×2)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A caller relied on the silent first-match to run a dependency's `main` deliberately | Medium | Unique-suffix fallback preserved; only the *multi-match* case changes, and it now names every candidate, so the fix is self-documenting at the point of failure |
| `RootModuleID` empty on a path where VM dispatch happens (e.g. a new single-file-with-imports route) | Medium | Resolver falls through to exact/unique-suffix — old behavior, loud only on true ambiguity; unit test pins the empty-root fallthrough |
| Testing engine migration breaks named-test dispatch | Medium | `__namedtest_entry` is unique by construction (synthesized per run); Phase 2 sweep runs the engine's own suite |
| Fix lands in v0.55.1 but the game clone stays on v0.54.18 | Low | Downstream bump is tracked as an explicit acceptance step (Success Criteria), not assumed |

## Verification Log

*No Go toolchain exists in this container (`go` not found), so no live `ailang check`/run rows are possible; every mechanism claim below is a code-read with `file:line`, per the skill's rule that mechanism claims are verified by reading the dispatch, and every language-behavior claim cites an existing fixture/test. The reporter's transcript (v0.54.18 binary, committed repro) is the live evidence for the incident.*

| # | Claim | How verified |
|---|---|---|
| 0 | The incident reproduces: `--bytecode --entry main` on `sim/tools/exoplanets_verify.ail` fails `GET_FIELD index 4 exceeds field count 2` attributed to `pkg/stapledons/sim/tools/exoplanets.main` at `exoplanets.ail:103` | Reporter's live run (directive, task `task-582a4a2b`); frame attribution mechanics confirmed by rows 2 and 8 |
| 1 | `FindEntryProto` scans prototypes in insertion order and returns the **first** `.name`/`_name` suffix match, never consulting module identity | Read `internal/runner/vm.go:25-37` |
| 2 | Multi-module prototypes are `Module + "." + Name` (bare `main` never matches exactly); single-module prototypes are bare names | Read `internal/bytecode/compiler/compiler.go:85-98, 293-298`; pinned by `internal/bytecode/compiler/multimodule_test.go:105` (`main.run`) |
| 3 | Prototype insertion order is `sort.Strings(modIDs)` then per-module decl order; entry module is included in `res.Modules` | Read `internal/runner/vm.go:64-90` (`CompileBytecodeFromResult`) |
| 4 | The entry module's `res.Modules` key is `CanonicalModuleID(src.Filename)` (the bare canonical file path); dependency keys are their import-path canonical IDs; both come from `LoadedModule.Path` | Read `internal/loader/loader.go:130, 324` (`Load`; `Path: canonicalID`), `internal/loader/loader.go:375` (`CanonicalModuleID`), `internal/pipeline/pipeline_module_phases.go:227-228` |
| 5 | Sibling/relative imports are rewritten to `pkg/<package>/…` canonical IDs (`pkg/` sorts before `sim/`, `tools/`, `src/`, … — any path whose first byte ≥ `p`) | Read `internal/loader/loader.go` `extractImports` (imp.Path = `"pkg/" + canonical`), `NormalizeRelativeImport`; byte-order sort |
| 6 | The evaluator path resolves `--entry` on the **entry module instance** (`inst.GetExport(entry)`), so the interpreter is unaffected | Read `internal/runner/entrypoint.go:62, 117-131` |
| 7 | The arity gate cannot catch the mismatch: evaluator arity (entry's 1 record param) and shadowing proto's `NumParams` (1 record param) both equal `len(bcArgs)` = 1 | Read `internal/runner/entrypoint.go:125-131`, `internal/runner/vm.go:190-196` |
| 8 | The GET_FIELD fault text and the frame's `Func` = prototype name, `File`/`Line` from the prototype's line info — so the observed attribution *is* the shadowing prototype's identity | Read `internal/vm/vm.go:531`, `internal/vm/diagnostics.go:14-32` |
| 9 | No prior design doc specifies VM entry resolution (the suffix policy was an undocumented implementation detail) | `grep -rln "FindEntryProto" design_docs/` → only unrelated hits; M-BYTECODE-MULTIMODULE sprint plan (`design_docs/implemented/v0_11_0/`) has no entry-resolution section |
| 10 | `pipeline.Result` has **no** existing root/module-ID field to reuse (negative existence — the plumbing premise of this design) | `grep -rn "RootModuleID" internal/` → empty; `grep -n "rootCanonical" internal/pipeline/*.go` → only the unexported phase-state field |
| 11 | The `_<name>` suffix tolerance is a known ambiguity, already deliberately excluded by the testing engine's own resolver | Read `internal/testing/named_batch.go:306-313` ("also accepts a `_<entry>` suffix, which is ambiguous") |
| 12 | `--batch`, `--policy`, `repl`, and `select_best` all route through `ExecuteModuleEntrypoint` → `tryRunEntryViaVM`, so one resolver fixes all CLI surfaces | Read `internal/runner/batch.go:111-132`, `cmd/ailang/main_run.go:246-269`, `cmd/ailang/repl.go:77`, `cmd/ailang/select_best.go:26` |
| 13 | The testing engine shares `runner.FindEntryProto` for its synthesized entry | Read `internal/testing/bytecode_engine.go:40, 128` |
| 14 | A generic diamond "did not reproduce" *consistently with this mechanism*: it reproduces only when a dependency exporting the entry name sorts before the entry module's ID (e.g. any `pkg/…` dep vs a root path starting ≥ `p`, or `core/types` vs `main`); a diamond whose entry sorts first cannot hit it | Derived from rows 3–5 (sort keys), not asserted about the reporter's test |
| 15 | Two exported `main`s in one package are **legal at check time** (MOD010 and `detectModulePathCollisions` govern module *paths*, not export names) | Read `internal/pipeline/pipeline_module_phases.go:213-215`; reporter: the full 139-module package `ailang check` passes while the VM run fails |
| 16 | `std/` modules export no `main` (no std prototype can shadow a `main` entry) | `grep -rn "func main" std/` → only a doc-comment mention in `std/ai/streaming.ail:30` |
| 17 | `canonicalFuncName` qualifies with `.` only — the `_` spelling the old tolerance matched is never a prototype name the compiler produces | Read `internal/bytecode/compiler/compiler.go:293-298` (`module + "." + name`, no `_` form anywhere in the function) |
| 18 | `runSingle` (bare-expression/REPL pipeline) leaves `Result.Modules` and `Result.Interface` empty, so a bare-expression run never reaches multi-module image compilation or `tryRunEntryViaVM` | `grep -n "result\\.Modules\\|result\\.Interface" internal/pipeline/pipeline_single.go` → empty; `ExecuteModuleEntrypoint` requires `params.Iface` (`internal/runner/entrypoint.go:47-57`) |
| 19 | No new error-code allocation is required (runner dispatch errors are plain `fmt.Errorf`, e.g. `entry function %q not found in bytecode image`) | Read `internal/runner/vm.go:177`; no `MODxxx`/`VMxxx` identifier introduced by this design |

## Related Documents

**Implemented (may inform design):**
- [M-BYTECODE-VM](../implemented/v0_11_0/m-bytecode-vm.md) and [M-BYTECODE-MULTIMODULE sprint plan](../implemented/v0_11_0/m-bytecode-multimodule-sprint-plan.md) — the image layout and module-qualified naming this fix builds on.
- [M-RUN-ENTRY-NO-EXPORTS-FAIL-LOUD](../implemented/v0_54_14/m-run-entry-no-exports-fail-loud.md) — the adjacent entry-point failure-mode (missing exports) made loud; this doc does the same for ambiguity.

**Planned (check for overlap):**
- [M-IMPORT-VS-IMPORT-COLLISION](../v0_54_20/m-import-vs-import-collision.md) — same package layout, different collision class (import *bindings* last-win in the elaborator, vs entry *dispatch* first-suffix-win in the VM runner). Distinct mechanisms, distinct fixes; both were surfaced by the stapledons game-clone work.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Incident: task `task-582a4a2b`, draft PR371 (`stapledons-godot-worlds`, branch `design/r3-worlds`), binary v0.54.18 `cd33d050b2276633c82624453309ebf7fcea9e2f`
- [M-IMPORT-VS-IMPORT-COLLISION](../v0_54_20/m-import-vs-import-collision.md) - sibling failure class
- `changelogs/v0.10-v0.17-bytecode-vm.md` - bytecode VM history

## Future Work

- Package-level info diagnostic ("this package exports `main` from N modules") in a future `ailang pkg lint` — only if real ambiguity incidents keep occurring after resolution is correct.
- Consider exposing the entry module's canonical ID in `ailang disasm` output headers so humans can verify `--entry` addressing without reading Go.

---

**Document created**: 2026-11-02
**Last updated**: 2026-11-02