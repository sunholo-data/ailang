# M-BYTECODE-MULTIMODULE — Sprint Plan

**Sprint ID**: `M-BYTECODE-MULTIMODULE`
**Target version**: v0.11.0
**Owner**: sprint-executor (TDD workflow)
**Status**: 📋 Planned
**Risk**: Medium
**Created**: 2026-04-08

---

## 1. Motivation

M-BYTECODE-BATCH shipped `--bytecode --batch` end-to-end and proved the VM is ~25× faster than the evaluator on single-module microbenchmarks (`fib(30)`, M5). On the motivating real-world workload — `ailang-parse` parsing a 9 MB DOCX — there is **no measurable speedup**, because the bytecode compiler cannot resolve cross-module references.

`ailang disasm ailang-parse/main.ail` shows **28 / 34 prototypes are EvalOnly**. Every call to another module's export hits `compiler: call to unknown global` and is bridged back to the evaluator. See [m-bytecode-vm.md §17](../../implemented/v0_11_0/m-bytecode-vm.md#17-m-bytecode-batch-2026-04-08--real-world-benchmark--next-sprint-scope) for the full breakdown.

This sprint closes that gap so that multi-module AILANG applications actually benefit from `--bytecode`.

### Acceptance threshold

- `ailang-parse/main.ail` disassembly: **≤ 4 / 34 EvalOnly prototypes** (down from 28/34)
- Parity harness: **stays at 133 MATCH / 2 NON_DET / 6 EVAL_SKIP** (no regression)
- `ailang-parse` 9 MB DOCX benchmark: **≥ 3× wall-clock speedup** vs evaluator in `--batch` mode (target — 5× would be ideal)

---

## 2. Current State Analysis

### Where the gaps live

| Location | Issue | Count (docparse) |
|---|---|---:|
| [compiler.go:73-82](../../../internal/bytecode/compiler/compiler.go#L73) | `funcIdx` populated from `prog.FuncDecls` only — single module | ~16 bridges |
| [call.go:85-89](../../../internal/bytecode/compiler/call.go#L85) | `GlobalRef` case ignores `e.Module` field, only looks up `e.Name` | (same path) |
| [disasm.go:96-104](../../../cmd/ailang/disasm.go#L96) | `prog.TypeDecls` seeded from current file's AST only | ~3 cross-mod ADT/record |
| [compiler.go:45-70](../../../internal/bytecode/compiler/compiler.go#L45) | `adtTypes`/`recordTypes` maps keyed by bare name, no module qualifier | (same path) |
| [collections.go:200-203](../../../internal/bytecode/compiler/collections.go#L200) | `lookupFieldIndex` uses CoreTI from current file only | ~1 field access |
| [elaborate/core.go:254-257](../../../internal/elaborate/core.go#L254) | ANF `$tmpNNN` bindings escape their let scope in some lower-pass paths | ~5 cases |
| [lower/match.go:69](../../../internal/gen/lower/match.go#L69) | Non-tail-position match panic (deferred — only 2 cases, not on critical path) | 2 cases |

### What's already in place (free wins)

- `stmt.GlobalRef{Module, Name}` already carries the module — lower pass emits it correctly. Just need the compiler to **use** the Module field.
- `pipeline.Result.Modules map[string]*loader.LoadedModule` already holds every reachable module with `Core`, `Iface`, and `CoreTI` populated. The data is there; the bytecode compile path just doesn't walk it.
- M3 bridge infrastructure handles the fallback case perfectly — so a partial migration is safe. A function that still fails to compile can stay EvalOnly without breaking parity.

---

## 3. Milestones

### M1: MULTIMOD_LOWER — Feed all reachable modules into `stmt.Program`
**Estimated LOC**: ~200 (impl ~130 + tests ~70)
**Depends on**: none
**Files**:
- `cmd/ailang/disasm.go` (`compileBytecodeFromResult`)
- `cmd/ailang/compile_v2.go` (if separate from disasm.go)
- `internal/gen/lower/program.go` (accept multi-module input or add multi-module helper)

**Scope**: Iterate `res.Modules` + current-file Core. For each module, lower its Core + AST TypeDecls into `stmt.Program`, then concatenate into a single program. Canonicalize function names to `"module/path.funcname"` so the compiler's `funcIdx` can key on fully-qualified names. Stable ordering (sorted by module path) for deterministic bytecode images.

**Non-goals**: Actually **using** the new entries from call sites — that's M2. This milestone just makes the data reach `compiler.Compile`.

**Acceptance criteria**:
- [ ] `compileBytecodeFromResult` emits a `stmt.Program` whose `FuncDecls` contains entries for every function in every reachable module (current file + `res.Modules`)
- [ ] All function names are canonicalized to `"module/path.name"` format; a new unit test verifies no collisions across modules
- [ ] `TypeDecls` merges ADT/record decls from all modules (same canonicalization for constructor names)
- [ ] Order is deterministic: sort by module path, then by source order within the module
- [ ] Existing golden tests in `tests/golden/bytecode/` still pass (single-module programs behave identically)
- [ ] New test: `TestMultiModuleLower_CollectsAllFuncs` — given a 2-module fixture, confirms both modules' funcs land in `prog.FuncDecls` with the expected canonical names
- [ ] Parity harness stays at 133 MATCH / 2 NON_DET / 6 EVAL_SKIP

---

### M2: CROSS_MOD_FUNCALL — Resolve `GlobalRef` by canonical name
**Estimated LOC**: ~150 (impl ~80 + tests ~70)
**Depends on**: M1
**Files**:
- `internal/bytecode/compiler/compiler.go` (`funcIdx` keying)
- `internal/bytecode/compiler/call.go` (`GlobalRef` dispatch in `classifyCallee`)
- `internal/bytecode/compiler/expr.go` (other `funcIdx` lookups at lines ~61, ~188)
- `internal/bytecode/disasm.go` (show canonical name in listing)

**Scope**: Switch `funcIdx` key from `fd.Name` to the canonical fully-qualified name set up in M1. In `call.go`'s `classifyCallee` `GlobalRef` branch, build the lookup key from `e.Module + "." + e.Name` (or whatever canonicalization M1 settles on) and dispatch through `funcIdx`. Update the two VarRef/expr.go sites that also read `funcIdx` so in-module calls (where `VarRef.Name` is unqualified) still resolve — we keep a secondary `localFuncIdx` that maps bare names to canonical names for the current module.

**Non-goals**: ADT constructors and record fields (M3).

**Acceptance criteria**:
- [ ] `ailang disasm ailang-parse/main.ail` shows `call to unknown global` count drop from ~16 to 0 (all function calls resolve)
- [ ] New test: `TestCrossModuleCall_ResolvesImportedFunc` — 2-module fixture, module B calls an exported function in module A, bytecode image has a direct `OpCall` (no EvalOnly bridge)
- [ ] Existing `call_test.go` cases still green (local calls behave identically)
- [ ] Disassembly prints canonical name, e.g. `; stdlib/std/io.println` rather than just `println`
- [ ] Parity harness stays at 133 MATCH / 2 NON_DET / 6 EVAL_SKIP

---

### M3: CROSS_MOD_ADT_RECORD — Imported ADTs, record fields, CoreTI merging
**Estimated LOC**: ~180 (impl ~110 + tests ~70)
**Depends on**: M1, M2
**Files**:
- `internal/bytecode/compiler/compiler.go` (`adtTypes`, `recordTypes` keyed by canonical type name)
- `internal/bytecode/compiler/switch.go` (`unknown ADT` path)
- `internal/bytecode/compiler/collections.go` (`compileADTConstructor`, `lookupFieldIndex`)
- `cmd/ailang/disasm.go` (merge imported `CoreTI` into the single one passed to lower)
- `internal/gen/stmt/stmt.go` (`SwitchStmt`/`ADTConstructor`: confirm these carry module-qualified type names; add field if missing)

**Scope**:
1. Merge imported modules' `ast.TypeDecl`s into `stmt.Program.TypeDecls` with canonical type names (covered by M1, but verify).
2. Key `adtTypes` / `recordTypes` maps by canonical type name so `SwitchStmt.ADTName` and `ADTConstructor.TypeName` resolve across modules.
3. Ensure `stmt.SwitchStmt` and `stmt.ADTConstructor` carry the originating module (may already be the case via a type-name prefix convention — audit and fix if not).
4. `lookupFieldIndex` currently reads `fc.coreTI` from the current file. Merge all imported modules' `CoreTI` maps into one and pass it to the compiler (via the lower-pass or as a compiler construction arg).

**Non-goals**: Anything that isn't covered by the docparse disasm error list.

**Acceptance criteria**:
- [ ] `ailang disasm ailang-parse/main.ail` shows `unknown ADT` errors gone (at least for the `Block` case)
- [ ] `ailang disasm ailang-parse/main.ail` shows `cannot resolve field` errors gone
- [ ] New test: `TestCrossModuleADT_SwitchCompiles` — module A defines `type Block = …`, module B `switch`es on a Block; compile succeeds, no EvalOnly
- [ ] New test: `TestCrossModuleRecord_FieldAccess` — imported record field lookup compiles to `OpGetField` with correct index
- [ ] Parity harness stays at 133 MATCH / 2 NON_DET / 6 EVAL_SKIP

---

### M4: LOWER_TMP_SCOPE — Fix `$tmpNNN` escape bug
**Estimated LOC**: ~100 (impl ~40 + tests ~60)
**Depends on**: none (can run in parallel with M1-M3)
**Files**:
- `internal/gen/lower/expr.go` or `internal/gen/lower/block.go` (wherever let-lowering happens)
- `internal/gen/lower/lower_test.go`

**Scope**: ANF temporaries (`$tmp242`, etc.) introduced by the elaborator end up as `VarRef`s that have no binding by the time the compiler sees them. This is because the lower pass has a path where a `let $tmpN = …` binding drops out of its enclosing block, leaving the use hanging. Find the dropped binding (likely in match-branch lowering or if-then-else with a temporary shared across branches) and restore it.

**Diagnosis hint**: Start by grepping the docparse disasm for the 5 functions that hit this; their Core should show a `Let` that the lower pass is missing. Write a minimal repro first, fix the pass, verify via `lower_test.go`.

**Acceptance criteria**:
- [ ] Minimal repro test added to `lower_test.go` (`TestLower_PreservesTmpBindings`) — currently fails, passes after fix
- [ ] `ailang disasm ailang-parse/main.ail` shows 0 `unbound variable $tmpN` errors
- [ ] No new `unbound variable` errors anywhere in the parity corpus
- [ ] Parity harness stays at 133 MATCH / 2 NON_DET / 6 EVAL_SKIP

---

### M5: BENCHMARK_AND_GATE — Measure wall-clock speedup, close the sprint
**Estimated LOC**: ~80 (script + doc updates, minimal code)
**Depends on**: M1, M2, M3, M4
**Files**:
- `scripts/bench_ailang_parse.sh` (new — wall-clock bench on the 9 MB DOCX)
- `design_docs/planned/v0_11_0/m-bytecode-multimodule-sprint-plan.md` → move to `implemented/v0_11_0/`
- `design_docs/implemented/v0_11_0/m-bytecode-vm.md` (add §18 with results)
- `CHANGELOG.md`

**Scope**:
1. Write `scripts/bench_ailang_parse.sh` that runs `ailang run --batch --bytecode ...` vs `ailang run --batch ...` on the 9 MB DOCX stress file N times (N=5, take median) and reports wall-clock ratio.
2. Re-run `ailang disasm ailang-parse/main.ail` and record the new EvalOnly count in the design doc.
3. Re-run the full parity harness to confirm 133 MATCH / 2 NON_DET / 6 EVAL_SKIP preserved.
4. Update `design_docs/implemented/v0_11_0/m-bytecode-vm.md` with a §18 "M-BYTECODE-MULTIMODULE results" section: before/after EvalOnly count, wall-clock speedup, any new findings.
5. Move this sprint plan to `implemented/v0_11_0/`.
6. CHANGELOG entry.

**Acceptance criteria**:
- [ ] `scripts/bench_ailang_parse.sh` produces reproducible wall-clock numbers
- [ ] Recorded speedup ≥ 3× (target; 5× ideal). If < 3×, re-open the disasm and find the next bottleneck (update the design doc with findings, discuss next sprint).
- [ ] EvalOnly count on docparse ≤ 4/34
- [ ] Parity harness: 133 MATCH / 2 NON_DET / 6 EVAL_SKIP
- [ ] Design doc and CHANGELOG updated
- [ ] Sprint plan moved to `implemented/v0_11_0/`

---

## 4. Day-by-day Plan

| Day | Milestone | Focus |
|---|---|---|
| 1 | M1 | Walk `res.Modules`, canonicalize names, merge into `stmt.Program`. Golden tests green. |
| 2 | M2 | Switch `funcIdx` to canonical keys. `classifyCallee` resolves `GlobalRef.Module.Name`. Disasm docparse — expect ~16 EvalOnly gone. |
| 3 | M3 | ADT + record layouts + CoreTI merge. Disasm docparse — expect ~3 more EvalOnly gone. |
| 4 | M4 (parallel) + M5 | Fix `$tmp` escape (~5 more). Write bench script. Record numbers. Update docs. Move to implemented/. |

Total estimate: **~4 days, ~710 LOC**. Medium risk — M3 has the most unknowns (CoreTI merging may surface cross-module type-id collisions we haven't thought about).

---

## 5. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| CoreTI type-id collisions across modules | High | Start with a single merged CoreTI keyed by `(module, local-id)`; write a collision-detection test before touching the compiler |
| Canonical naming breaks golden tests that assert bare names | Medium | Update goldens in the same commit that introduces canonicalization; parity harness is the true regression gate |
| `$tmp` escape bug is deeper than expected (multiple lower-pass paths) | Medium | M4 is parallel/independent — if it balloons, ship M1-M3 separately and fold M4 into the next sprint |
| Wall-clock speedup < 3× even after all bridges resolved | Medium | Re-profile: if bytecode is still slow, the bottleneck is elsewhere (builtins, effect trap overhead) and we open a follow-up sprint with the new data |
| Cross-module test fixtures bloat the repo | Low | Use `testdata/multimodule/` with minimal 2-file fixtures; don't pull ailang-parse as a dependency |

---

## 6. Success Metrics

- [ ] `ailang-parse/main.ail` disassembly: ≤ 4 / 34 EvalOnly prototypes
- [ ] Parity harness: 133 MATCH / 2 NON_DET / 6 EVAL_SKIP
- [ ] 9 MB DOCX wall-clock speedup ≥ 3× in `--batch --bytecode` vs `--batch`
- [ ] Test coverage on `internal/bytecode/compiler` unchanged or higher
- [ ] No new files > 800 LOC
- [ ] Design doc §18 published with before/after numbers

---

## 7. Handoff Notes for sprint-executor

- **Start with**: Read [m-bytecode-vm.md §17](../../implemented/v0_11_0/m-bytecode-vm.md) for the breakdown of the 28 EvalOnly prototypes — that's your success metric to beat.
- **Benchmark fixture**: `/Users/mark/dev/sunholo/ailang-parse/` (sibling repo). The 9 MB DOCX is mentioned as the stress-test input; find it in that repo's `testdata/` or ask.
- **Regression gate**: `go run ./scripts/verify_bytecode_parity.go` — run after every milestone. Must stay at 133 MATCH.
- **Per-milestone disasm check**: `./bin/ailang disasm /path/to/ailang-parse/main.ail 2>&1 | grep EvalOnly` — use this to confirm the per-milestone EvalOnly reduction.
- **TDD discipline**: Write the failing test first for each milestone (especially M1's cross-module fixture — that's the foundation for everything else).
- **Deferred**: The 2 `non-tail-position match` panics are explicitly out of scope; don't fix them in this sprint.
