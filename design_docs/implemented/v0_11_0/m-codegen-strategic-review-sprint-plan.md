# Sprint Plan: M-CODEGEN-STRATEGIC-REVIEW — Statement IR + Clean-Room Go Emitter

## Summary

Delete the current 10,757 LOC Go codegen and replace it with a layered Statement IR (~800 LOC) + thin Go emitter (~500 LOC). The single acceptance gate is: **Stapledon's Voyage (17 modules, 4,549 LOC AILANG) compiles to Go and passes `go build`**.

No backward compatibility. One consumer. Delete and rebuild.

**Duration:** 10 working days (2 weeks), split across 5 milestones
**Dependencies:** None — Core IR pipeline is unchanged, evaluator is unchanged
**Risk Level:** Medium-High (large deletion, clean-room rebuild, but well-scoped acceptance gate)
**Design Doc:** [m-codegen-strategic-review.md](m-codegen-strategic-review.md)

---

## Current Status Analysis

### What Exists
- **Current Go codegen**: 30 source files, 10,757 LOC (+ 24 test files, 6,310 LOC)
- **Block IR**: 121 LOC in `internal/gen/block/` (proven concept, 58% IIFE reduction)
- **Core IR**: 567 LOC, ~20 expression types, 7 pattern types
- **Pipeline**: Fully functional (parse → elaborate → typecheck → monomorphize → lower)
- **Stapledon's Voyage**: 17 modules, 4,549 LOC, 106 match expressions, 44 cross-module imports, 5 effect annotations, 354 LOC protocol types (complex ADTs)

### Velocity (Last 30 Days)
- 435 commits, ~34,000 net LOC (71K insertions, 37K deletions)
- Mix of feature work, bug fixes, releases (v0.10.1, v0.10.2)
- Relevant precedent: M-CODEGEN-V2 (Block IR) was 121 LOC, done in 1 day
- Relevant precedent: M-STREAM-ZIP-XML was a significant new stdlib feature, done in ~2 days
- **Conservative estimate**: 300-500 LOC/day for new architecture work (lower than raw velocity due to design thinking)

### What We're Deleting
- `internal/gen/golang/` — all 30 source files (10,757 LOC)
- `internal/gen/golang/*_test.go` — all 24 test files (6,310 LOC) after extracting golden test data
- **Total removed**: ~17,000 LOC

### What We're Building
| Component | Estimated LOC | Purpose |
|-----------|--------------|---------|
| `internal/gen/stmt/` | ~250 | Statement IR type definitions |
| `internal/gen/lower/` | ~550 | Match lowering + type projection + block lowering |
| `internal/gen/golang/emitter.go` | ~500 | Thin Go emitter (ONLY file that knows Go syntax) |
| `internal/gen/golang/runtime.go` | ~300 | Minimal runtime helpers (CallFunc, type conversion) |
| `internal/gen/golang/types.go` | ~200 | ADT/record Go struct generation |
| `tests/golden/codegen/` | ~400 | Golden test corpus (extracted from old tests) |
| Tests for new code | ~600 | Statement IR + lowering + emitter tests |
| **Total new** | **~2,800** | **vs 17,000 deleted — net -14,200 LOC** |

---

## Proposed Milestones

### Milestone 1: M1_EXTRACT_GOLDEN_TESTS (Day 1)

**Goal:** Extract test input/output pairs from the existing codegen tests before deleting anything. These become the golden test corpus that validates the new pipeline.

**Estimated:** 200 LOC (golden test files + test runner)

**Tasks:**
- Day 1 AM: Read all 24 existing test files, identify input AILANG snippets and expected Go output patterns
- Day 1 AM: Create `tests/golden/codegen/` directory with `.ail` input files and `.go.golden` expected output files
- Day 1 PM: Write simple golden test runner that compiles `.ail` → Go via pipeline and compares to `.golden`
- Day 1 PM: Extract at least 20 golden test cases covering: literals, functions, ADTs, pattern matching, records, lists, cross-module, effects
- Day 1 PM: Also capture Stapledon's Voyage `sim/protocol.ail` and `sim/world.ail` as integration golden tests

**Acceptance Criteria:**
- [ ] `tests/golden/codegen/` contains 20+ `.ail` → `.go.golden` pairs
- [ ] Golden test runner exists and can be invoked with `go test`
- [ ] Stapledon's Voyage key modules captured as integration test inputs
- [ ] Old test cases documented (which features each covers)

**Risks:**
- Some old tests may test internal codegen details rather than input→output. Skip those — only extract end-to-end pairs.

---

### Milestone 2: M2_STATEMENT_IR_TYPES (Days 2-3)

**Goal:** Define the Statement IR type system — the ONLY representation that emitters will ever see. This is the keystone of the entire architecture.

**Estimated:** 250 LOC (types) + 150 LOC (tests) = 400 LOC

**Tasks:**
- Day 2 AM: Define `internal/gen/stmt/stmt.go` — FuncBody, Stmt (VarDecl, IfStmt, SwitchStmt, ReturnStmt, AssignStmt), Expr (VarRef, Literal, BinOp, UnOp, Call, FieldAccess, IndexAccess, StructLiteral, SliceLiteral, TypeAssert, Lambda)
- Day 2 PM: Define `internal/gen/stmt/types.go` — ResolvedType (Primitive, Struct, Slice, Func, ADT, Tuple, Interface) with pure projection from `types.Type`
- Day 3 AM: Define `internal/gen/stmt/program.go` — Program (TypeDecls, FuncDecls, RuntimeImports) representing a complete compilation unit
- Day 3 AM: Write property tests: every Statement IR node is printable, every ResolvedType is deterministic
- Day 3 PM: Write `internal/gen/stmt/validate.go` — validates IR invariants (all vars declared before use, no nested expressions > depth 1, all types resolved)

**Acceptance Criteria:**
- [ ] Statement IR types cover all 20 Core expression types (may combine some)
- [ ] ResolvedType covers: int64, float64, bool, string, struct, slice, func, ADT, tuple, interface{}
- [ ] Type projection is a **pure function** — same input always produces same output
- [ ] Type projection **errors** on unresolved type variables (no silent defaults)
- [ ] IR validation catches malformed programs
- [ ] No import of `core` package from `stmt` package (hard boundary)

**Risks:**
- Getting the type system right is critical. Under-specifying means emitters need Core access. Over-specifying means the IR is too complex.
- Mitigation: Start with what Stapledon's Voyage actually uses. Add generality only when needed.

---

### Milestone 3: M3_LOWERING_PASSES (Days 4-6)

**Goal:** Build the lowering passes that transform Core IR → Statement IR. This is where the complexity lives, but it's concentrated in one place rather than spread across 30 files.

**Estimated:** 550 LOC (lowering) + 250 LOC (tests) = 800 LOC

**Tasks:**
- Day 4 AM: `internal/gen/lower/block.go` — Adapt existing Block IR (let-chain flattening) to produce Statement IR VarDecls. Port from `internal/gen/block/` (51 LOC of types, 70 LOC of logic)
- Day 4 PM: `internal/gen/lower/typeres.go` — Type projection pass. Walk CoreTypeInfo, map every AILANG type to ResolvedType. Error on unresolved type vars. Handle: primitives, records→structs, ADTs→discriminator types, lists→slices, functions→func types, tuples→tuple structs
- Day 5 AM: `internal/gen/lower/match.go` — Match lowering. Core Match → Statement IR IfStmt/SwitchStmt. Handle 7 pattern types: VarPattern, LitPattern (int/float/string/bool), ConstructorPattern, ListPattern, RecordPattern, WildcardPattern, TuplePattern
- Day 5 PM: `internal/gen/lower/match.go` continued — Nested patterns (ConstructorPattern with sub-patterns), guard expressions, ADT tag dispatch
- Day 6 AM: `internal/gen/lower/expr.go` — Expression lowering. Core expressions → Statement IR expressions. Handle: Var, VarGlobal, Lit, Lambda, App, If, BinOp, UnOp, Record, RecordAccess, RecordUpdate, List, Array, Tuple, Intrinsic, DictApp, DictRef
- Day 6 PM: `internal/gen/lower/program.go` — Top-level orchestrator. Core Program → Statement IR Program. Handle: type declarations (ADTs, records, type aliases), function declarations (Let, LetRec with Lambda), module namespacing, export visibility
- Day 6 PM: Integration test — run full lowering on a simple multi-module AILANG program, verify Statement IR validates

**Acceptance Criteria:**
- [ ] Block lowering produces flat VarDecl sequences (no nested IIFEs)
- [ ] Type projection handles all types used by Stapledon's Voyage
- [ ] Match lowering handles all 7 pattern types
- [ ] Expression lowering handles all 20 Core expression types
- [ ] Program lowering handles multi-module compilation
- [ ] All lowering is deterministic (same input → same output)
- [ ] No emitter/Go-specific logic in lowering passes
- [ ] Unit tests for each lowering pass

**Risks:**
- Match lowering is the most complex piece (the old codegen had 663 LOC for this). The decision tree approach from M-CODEGEN-IR-STRATEGY should keep it under 200 LOC.
- DictApp/DictRef handling may need special attention — these represent type class dispatch.
- Mitigation: Focus on patterns Stapledon's Voyage actually uses. Defer exotic patterns.

---

### Milestone 4: M4_GO_EMITTER (Days 7-8)

**Goal:** Build the thin Go emitter that walks Statement IR and produces Go source code. This is the ONLY file that knows Go syntax. Delete the old codegen.

**Estimated:** 500 LOC (emitter) + 300 LOC (runtime) + 200 LOC (types) + 200 LOC (tests) = 1,200 LOC

**Tasks:**
- Day 7 AM: `internal/gen/golang/emitter.go` — Statement IR → Go source. Walk FuncBody/Stmt/Expr, emit Go syntax. Handle: VarDecl→`var x Type = ...`, IfStmt→`if ... { } else { }`, SwitchStmt→`switch ... { case: }`, Call→`FuncName(args)`, BinOp→`x + y`, FieldAccess→`x.Field`, etc.
- Day 7 PM: `internal/gen/golang/types.go` — ADT struct generation (discriminator pattern), record struct generation, type alias emission. ResolvedType → Go type string (int64, float64, *StructName, []T, func(...) T)
- Day 7 PM: `internal/gen/golang/runtime.go` — Minimal runtime helpers: CallFunc (for higher-order functions via interface{}), type conversion helpers, list/slice operations. Keep this as small as possible.
- Day 8 AM: Delete `internal/gen/golang/` old files (all 30 source files, 10,757 LOC). Keep only the new emitter.go, types.go, runtime.go, and package.go
- Day 8 AM: Wire up CLI `cmd/ailang/compile.go` to new pipeline: Core → Lower → Statement IR → Go Emitter
- Day 8 PM: Run golden tests — fix any emission bugs until all 20+ golden tests pass
- Day 8 PM: Run `go vet` and `go build` on generated output

**Acceptance Criteria:**
- [ ] Go emitter is <600 LOC (hard constraint from design doc)
- [ ] Old `internal/gen/golang/` codegen deleted (10,757 LOC removed)
- [ ] Emitter ONLY reads Statement IR — no imports of `core` package
- [ ] All 20+ golden tests pass
- [ ] Generated Go code compiles with `go build`
- [ ] Generated Go code passes `go vet`
- [ ] `ailang compile --emit-go` works end-to-end on simple programs

**Risks:**
- The `interface{}` problem still exists in the Go emitter, but it's contained in ~500 LOC, not 10,757.
- Runtime helpers may grow larger than estimated if Stapledon's Voyage uses many stdlib functions. Mitigation: Start minimal, add only what's needed.
- The CLI wiring may have unexpected dependencies on old codegen types. Mitigation: Check `compile.go` imports early.

---

### Milestone 5: M5_STAPLEDON_ACCEPTANCE (Days 9-10)

**Goal:** Compile Stapledon's Voyage (17 modules, 4,549 LOC) through the new pipeline. Fix any remaining issues. Add CI. This is the single acceptance gate for the entire sprint.

**Estimated:** 200 LOC (fixes) + 100 LOC (CI) = 300 LOC

**Tasks:**
- Day 9 AM: Run `ailang compile --emit-go --out /tmp/sim_gen --package-name sim_gen /Users/mark/dev/sunholo/stapledons_voyage/sim/`
- Day 9 AM-PM: Debug and fix issues. Expected problem areas:
  - Complex ADTs in `protocol.ail` (40+ DrawCmd variants including nullary constructors)
  - Cross-module type imports (17 modules with 44 import statements)
  - Pattern matching with nested patterns (106 match expressions)
  - Effect annotations (5 effectful functions)
  - Record updates across module boundaries
- Day 9 PM: Generated code compiles with `go build`
- Day 10 AM: Add CI workflow `.github/workflows/test-codegen-multimodule.yml`
  - Step 1: `make build && make quick-install`
  - Step 2: `ailang compile --emit-go` on test corpus
  - Step 3: `go build` + `go vet` on generated output
- Day 10 AM: Add CI workflow that compiles Stapledon's Voyage (or a representative subset)
- Day 10 PM: Update docs — `docs/guides/go-compilation-status.md` with supported features
- Day 10 PM: Update CHANGELOG.md with the architectural change
- Day 10 PM: Clean up — remove any dead references to old codegen in the codebase

**Acceptance Criteria:**
- [ ] **Stapledon's Voyage compiles to Go and passes `go build`** (THE acceptance gate)
- [ ] CI workflow runs on every PR touching `internal/gen/`
- [ ] `docs/guides/go-compilation-status.md` documents supported features
- [ ] CHANGELOG.md updated
- [ ] No references to deleted codegen files remain in codebase
- [ ] `make lint` passes
- [ ] `make test` passes (no regressions in non-codegen tests)

**Risks:**
- Stapledon's Voyage may exercise codegen paths not covered by golden tests. Day 9 is the buffer day for this.
- The protocol.ail types (40+ ADT variants) are the biggest risk — they stress-test ADT generation heavily.
- Mitigation: If Day 9 reveals a fundamental issue in the Statement IR design, we have Day 10 AM as overflow before CI work.

---

## Day-by-Day Summary

| Day | Milestone | Key Deliverable | LOC |
|-----|-----------|----------------|-----|
| 1 | M1: Extract Golden Tests | 20+ golden test pairs extracted | ~200 |
| 2 | M2: Statement IR Types (pt 1) | stmt.go + types.go defined | ~200 |
| 3 | M2: Statement IR Types (pt 2) | Validation + property tests | ~200 |
| 4 | M3: Lowering (pt 1) | Block lowering + type projection | ~250 |
| 5 | M3: Lowering (pt 2) | Match lowering | ~300 |
| 6 | M3: Lowering (pt 3) | Expression + program lowering | ~250 |
| 7 | M4: Go Emitter (pt 1) | Emitter + types + runtime | ~700 |
| 8 | M4: Go Emitter (pt 2) | Delete old codegen, wire CLI, golden tests pass | ~500 |
| 9 | M5: Stapledon (pt 1) | Stapledon's Voyage compiles | ~200 |
| 10 | M5: Stapledon (pt 2) | CI + docs + cleanup | ~200 |
| **Total** | | | **~2,800** |

## Success Metrics

| Metric | Before | After | Verification |
|--------|--------|-------|-------------|
| Codegen source LOC | 10,757 | ~1,500 | `wc -l internal/gen/golang/*.go` |
| Codegen files | 30 | ~4 | `ls internal/gen/golang/*.go` |
| Total gen/ LOC (incl. stmt, lower) | 10,757 | ~2,300 | `find internal/gen -name "*.go" \| xargs wc -l` |
| Packages that inspect Core AST | many | 1 (lower/) | `grep -r "core\." internal/gen/golang/` returns 0 |
| Golden tests | 0 | 20+ | `ls tests/golden/codegen/*.ail` |
| CI codegen test | none | multi-module | `.github/workflows/` |
| Stapledon's Voyage | stale/broken | compiles | `go build` on generated output |
| Net LOC change | — | **-14,200** | `git diff --stat` |

## Dependencies

- **Core pipeline unchanged**: Parse → Elaborate → Typecheck → Monomorphize → Lower — no modifications needed
- **Evaluator unchanged**: `internal/eval/` is untouched
- **Stapledon's Voyage source**: Must be accessible for M5 integration test
- **Go 1.21+**: For generics in generated runtime helpers (if needed)

## Open Questions

1. **Should the Go emitter use Go generics (1.18+)?** This could reduce `interface{}` usage significantly but limits compatibility. Recommendation: start with `interface{}`, add generics as a follow-up optimization.
2. **How much runtime.go is needed?** The old runtime was 1,100+ LOC. Goal is <300 but Stapledon's Voyage may force expansion. We'll discover on Day 9.
3. **Should Statement IR be effect-aware?** Design doc follow-up 14.1 raises this. Recommendation: defer to Phase 2, effects are a small surface in Stapledon's Voyage (5 annotations).
4. **Block IR reuse vs rewrite?** The existing 121 LOC Block IR works but targets old codegen types. Recommendation: rewrite the 70 LOC of logic to target Statement IR types (it's small enough).

## Notes

- **No backward compatibility required.** Stapledon's Voyage is the only consumer.
- **Delete aggressively.** The old codegen is 10,757 LOC of architectural debt. Every line removed is maintenance saved.
- **The hard constraint**: No emitter may import `internal/core`. If the emitter needs information, extend Statement IR — don't reach around it.
- **Risk buffer**: Days 9-10 are specifically sized as buffer for Stapledon's Voyage integration issues. If M1-M4 go smoothly, M5 may finish in 1 day, giving a day for polish.
- **Velocity assumption**: 300-500 LOC/day for new architecture work. The 2,800 LOC total at ~280 LOC/day over 10 days is conservative.
