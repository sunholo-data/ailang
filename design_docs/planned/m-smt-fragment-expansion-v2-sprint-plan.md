# Sprint Plan: M-SMT-FRAGMENT-EXPANSION-V2 — Z3 Verification Phase 2

## Summary

Expand AILANG's Z3 verification coverage from ~70-80% to ~85-90% of typical functions by adding monomorphic HOF inlining, recursive list operations, bounded quantifiers, nested records, record patterns, and per-function recursion depth. Six independently shippable phases, each building on existing infrastructure.

**Duration:** 6 days (~58 hours)
**Dependencies:** All Phase 1 work complete (v0.8.0). Z3 installed locally.
**Risk Level:** Medium (Phase F HOF inlining is the most complex; Phases G-K are lower risk)

## Current Status Analysis

### Completed (Phase 1 — v0.8.0)
- ✅ Phase A: Cross-function calls (callee_resolver.go, ~450 LOC) — 2026-02-12
- ✅ Phase B: Strings (str.* theory, 12 builtins) — 2026-02-12
- ✅ Phase C: Records (declare-datatype, field access/update) — 2026-02-12
- ✅ Phase D: Lists (seq.* theory, 5 builtins) — 2026-02-12
- ✅ Phase E: Bounded recursion (Dafny-style unrolling, depth 1-10) — 2026-02-13
- ✅ Record discovery from bodies/return types — 2026-02-13
- ✅ File split for AI-friendly modules — 2026-02-13

### Current Codebase
- **Implementation**: 3,300 LOC across 12 files in `internal/smt/`
- **Tests**: 4,070 LOC across test files
- **Test ratio**: 1.23x (tests > implementation — good coverage)
- **All tests passing** ✅
- **15 contract examples** in `examples/runnable/contracts/`

### Velocity (from Phase 1 history)
- Phase 1 (A-E + record discovery + file split): ~7,370 LOC total in ~2 days
- Phases B+C (strings + records): ~4-5 hours each
- Phase D (lists): ~4-5 hours
- Phase E (bounded recursion): ~6-8 hours
- **Estimated sustained velocity**: ~400-600 LOC/day (implementation + tests)

### Remaining Work (Phase 2)
- ⏳ Phase F: HOF inlining — ~550 LOC (highest complexity)
- ⏳ Phase G: Recursive list ops — ~340 LOC
- ⏳ Phase H: Bounded quantifiers — ~460 LOC (new syntax)
- ⏳ Phase I: Nested records — ~200 LOC
- ⏳ Phase J: Record patterns — ~135 LOC
- ⏳ Phase K: Per-function depth — ~205 LOC

**Total estimated**: ~1,890 LOC implementation + tests

## Proposed Milestones

### M1: Nested Records (Phase I) — Day 1 morning
**Goal:** Support nested record types in Z3 verification (e.g., `{inner: {x: int, y: int}}`)
**Estimated:** ~100 LOC implementation + ~100 LOC tests = ~200 LOC
**Duration:** 3 hours

**Why first:** Lowest risk, self-contained, warms up the SMT codegen muscles. Unblocks real-world record patterns.

**Tasks:**
1. Extend `collectAndDeclareRecordTypes` in `codegen_records.go` to recursively discover record-typed fields
2. Add topological sort for record type declarations (inner before outer)
3. Verify nested field access chains encode correctly (`o.pos.x` → `(x (pos o))`)
4. Write tests for 1-level and 2-level nesting
5. Create `examples/runnable/contracts/nested_record_verify.ail`

**Acceptance Criteria:**
- [ ] `{inner: {x: int, y: int}}` declares both inner and outer types in correct order
- [ ] `o.inner.x` encodes to `(x (inner o))` in SMT-LIB
- [ ] Recursive record types (self-referential) are rejected with clear error
- [ ] `ailang verify nested_record_verify.ail` produces VERIFIED results
- [ ] All existing tests still pass
- [ ] Linting clean

**Risks:** None significant — Z3 handles nested datatypes natively.

---

### M2: Record Patterns in Match (Phase J) — Day 1 afternoon
**Goal:** Encode `core.RecordPattern` in SMT match expressions
**Estimated:** ~55 LOC implementation + ~80 LOC tests = ~135 LOC
**Duration:** 2.5 hours

**Tasks:**
1. Add `*core.RecordPattern` case to `encodePattern` in `codegen_control.go`
2. Map record field names to constructor argument positions (alphabetical order, matching `DeclareRecordDatatype`)
3. Handle partial record patterns (subset of fields)
4. Write tests for record pattern encoding
5. Create `examples/runnable/contracts/record_pattern_verify.ail`

**Acceptance Criteria:**
- [ ] `match p { {x, y} => x + y }` encodes to `(match p ((mk_Point x y) (+ x y)))`
- [ ] Field order in pattern matches alphabetical order in declaration
- [ ] Existing ADT constructor patterns still work
- [ ] `ailang verify record_pattern_verify.ail` produces VERIFIED results
- [ ] All existing tests still pass

**Risks:** Low — `RecordPattern` already exists in Core AST, just needs encoding.

---

### M3: Recursive List Operations (Phase G) — Day 2
**Goal:** Register and encode recursive list operations (`reverse`, `take`, `drop`, `contains`, `extract`) for Z3 verification
**Estimated:** ~200 LOC implementation + ~140 LOC tests = ~340 LOC
**Duration:** 6 hours

**Tasks:**
1. Add non-recursive list builtins to `ListBuiltinSpecial`: `_list_contains` → `seq.contains`, `_list_extract` → `seq.extract`
2. Register runtime implementations for `_list_contains`, `_list_extract` in `builtins/list.go`
3. Create `list_unroll.go` with Core AST templates for `_list_reverse`, `_list_take`, `_list_drop`
4. Integrate recursive templates with existing `UnrollRecursiveFunction`
5. Update `encodable.go` to accept new builtins in fragment checker
6. Update `StdlibListToSMT` mappings for `reverse`, `take`, `drop`
7. Create `examples/runnable/contracts/list_recursive_verify.ail`

**Acceptance Criteria:**
- [ ] `_list_contains([1,2,3], 2)` returns `true` at runtime and encodes to `(seq.contains xs (seq.unit 2))`
- [ ] `_list_extract([1,2,3], 1, 2)` returns `[2,3]` at runtime and encodes to `(seq.extract xs 1 2)`
- [ ] `_list_reverse(xs)` verifies via bounded unrolling: `ensures { _list_length(result) == _list_length(xs) }`
- [ ] `_list_take(n, xs)` verifies: `ensures { _list_length(result) <= n }`
- [ ] All existing tests still pass
- [ ] Linting clean

**Risks:**
- `seq.extract` semantics differ from AILANG `take`/`drop` slightly — validate Z3 behavior
- Recursive templates need correct cons/nil encoding

---

### M4: Per-Function Recursion Depth (Phase K) — Day 3 morning
**Goal:** Add `@verify(depth: N)` attribute to override global `--verify-recursive-depth`
**Estimated:** ~115 LOC implementation + ~90 LOC tests = ~205 LOC
**Duration:** 4 hours

**Tasks:**
1. Parse `@verify(key: value)` attribute before function declarations in `parser.go`
2. Add `Attributes map[string]any` field to `ast.FuncDecl`
3. Add `VerifyDepth int` to `core.DeclMeta` and carry through elaboration
4. Read per-function depth in `cmd/ailang/verify.go`, override global setting
5. Write parser tests, elaboration tests, and integration test
6. Create `examples/runnable/contracts/per_function_depth_verify.ail`

**Acceptance Criteria:**
- [ ] `@verify(depth: 5)` parses and attaches to next function declaration
- [ ] Per-function depth overrides `--verify-recursive-depth N` CLI flag
- [ ] Functions without `@verify` use global depth (or default behavior)
- [ ] Output shows correct depth: "VERIFIED (bounded: depth 5)"
- [ ] All existing tests still pass

**Risks:**
- `@` prefix may conflict with future attribute system — keep minimal (only `verify` attribute)
- Parser token position tracking for attribute nodes needs care

---

### M5: Monomorphic HOF Inlining (Phase F) — Days 3 afternoon + Day 4
**Goal:** Verify `map(\x -> body, xs)`, `filter(\x -> pred, xs)`, `foldl(\acc x -> body, init, xs)` when lambda arguments are known literals
**Estimated:** ~250 LOC implementation + ~300 LOC tests = ~550 LOC
**Duration:** 12 hours (spread across 2 half-days)

**This is the most complex milestone.** The approach: detect known-lambda HOF calls, specialize them into recursive functions, then feed to existing bounded unrolling.

**Tasks:**
1. Create `hof_inline.go` with `IsInlinableHOF()` detector: checks for `map`/`filter`/`foldl` with literal lambda args
2. Implement `SpecializeMap(lambda, listExpr)` → generates recursive Core AST
3. Implement `SpecializeFilter(lambda, listExpr)` → generates recursive Core AST
4. Implement `SpecializeFold(lambda, init, listExpr)` → generates recursive Core AST
5. Refine `hasHigherOrder()` in `encodable.go` to exempt known-lambda HOF calls
6. Hook into `EncodeFunction`: detect HOF → specialize → unroll → encode
7. Test lambda substitution correctness (body variables don't escape)
8. Test composition: `map(\x -> helper(x), xs)` where `helper` is a resolved callee
9. Create `examples/runnable/contracts/hof_verify.ail`

**Acceptance Criteria:**
- [ ] `map(\x -> x + 1, xs)` verifies with length preservation contract
- [ ] `filter(\x -> x > 0, xs)` verifies with `length(result) <= length(xs)` contract
- [ ] `foldl(\acc x -> acc + x, 0, xs)` verifies with sum contract
- [ ] `map(f, xs)` where `f` is a parameter still REJECTED (correct)
- [ ] Output labels: "VERIFIED (bounded: depth N, inlined: map)"
- [ ] No regressions on existing HOF rejection tests
- [ ] All existing tests still pass

**Risks:**
- Lambda variable capture: specialized body must correctly substitute bound vars
- Interaction with cross-function calls (helper functions inside lambda)
- Z3 timeout on deeply nested unrolled map/filter — mitigate with reasonable depth

---

### M6: Bounded Quantifiers (Phase H) — Days 5 + 6
**Goal:** Support `forall i: lo..hi => P(i)` in ensures clauses, encoding to Z3 bounded quantifiers
**Estimated:** ~260 LOC implementation + ~200 LOC tests = ~460 LOC
**Duration:** 10 hours (spread across 2 days)

**This adds new syntax** — requires parser, AST, Core, elaboration, and SMT changes.

**Tasks:**
1. **Day 5 morning**: Parse `forall IDENT: EXPR..EXPR => EXPR` in contract expressions (`parser_contracts.go`)
2. **Day 5 morning**: Add `ForallExpr` to `ast_expr.go` with binder name, lower bound, upper bound, body
3. **Day 5 afternoon**: Lower `ForallExpr` to `core.Forall{Var, Lo, Hi, Body}` in elaboration
4. **Day 5 afternoon**: Add `core.Forall` node to `core.go` with `String()` method
5. **Day 6 morning**: Create `codegen_quantifier.go` — encode `core.Forall` to SMT-LIB bounded `(forall ((i Int)) (=> (and (>= i lo) (< i hi)) body))`
6. **Day 6 morning**: Update `encodable.go` to accept `core.Forall` nodes, validate bounds are encodable
7. **Day 6 afternoon**: Update `walkForHigherOrder`, `containsRef`, `walkForUnencodableTypes`, `ReplaceSelfCalls` to handle `core.Forall`
8. **Day 6 afternoon**: Integration tests with Z3, create `examples/runnable/contracts/quantifier_verify.ail`

**Acceptance Criteria:**
- [ ] `forall i: 0.._list_length(result) => _list_nth(result, i) >= 0` parses and type-checks
- [ ] Encodes to `(forall ((i Int)) (=> (and (>= i 0) (< i (seq.len result))) (>= (seq.nth result i) 0)))`
- [ ] Z3 returns `unsat` for correct contracts (VERIFIED)
- [ ] Z3 returns `sat` with counterexample index for violated contracts
- [ ] Nested `forall` rejected with clear error
- [ ] `forall` over non-integer types rejected
- [ ] All existing tests still pass
- [ ] Linting clean

**Risks:**
- Z3 quantifier reasoning can be slow — add per-function timeout (5s default)
- `forall` keyword may conflict with existing token — check lexer
- AST walker updates are tedious — many switch statements to update

---

## Day-by-Day Schedule

| Day | Morning (4h) | Afternoon (4-6h) | LOC |
|-----|-------------|-------------------|-----|
| **1** | M1: Nested records | M2: Record patterns | ~335 |
| **2** | M3: Recursive list ops (builtins) | M3: Recursive list ops (templates + tests) | ~340 |
| **3** | M4: Per-function depth | M5: HOF inlining (detector + map specialization) | ~350 |
| **4** | M5: HOF inlining (filter + fold + tests) | M5: HOF inlining (integration + example) | ~300 |
| **5** | M6: Parser + AST + Core for forall | M6: Elaboration + fragment checker | ~250 |
| **6** | M6: SMT encoding + walker updates | M6: Integration tests + examples + docs | ~315 |

**Total: ~1,890 LOC across 6 days**

## Success Metrics

- Test coverage: Maintain existing 1.2x test:implementation ratio
- Examples passing: 6 new contract examples (21 total)
- Documentation: Update `docs/docs/guides/contracts.mdx` decidable fragment table
- CHANGELOG: Add v0.10.0 entries for each phase
- All tests passing: ✅ (run `make test` after each milestone)
- All linting passing: ✅ (run `make lint` after each milestone)
- Z3 integration: All new examples verify with `ailang verify`

## Dependencies

- Z3 installed at `/opt/homebrew/bin/z3` or equivalent
- Go 1.24+ (per go.mod)
- All Phase 1 tests passing before starting

## Open Questions

1. **Attribute syntax**: Is `@verify(depth: 5)` the right syntax, or should it be a comment annotation like `-- @verify depth=5`? The `@` prefix is cleaner but commits to an attribute system.
2. **Bounded quantifier timeout**: What's the right default timeout per function when quantifiers are involved? 5 seconds? 10 seconds?
3. **`foldl` inlining priority**: Is `foldl` inlining worth the complexity? It's the hardest HOF to specialize due to the accumulator. Could defer to a follow-up.

## Notes

- Each milestone is independently shippable — if Phase F (HOF) proves too complex, ship everything else first
- Milestone order is optimized for risk reduction: easy wins first (M1-M2), then building complexity (M3-M4), then the hard parts (M5-M6)
- Run `make test && make lint` after every milestone completion
- Run `ailang verify examples/runnable/contracts/<new_example>.ail` for integration validation
- The bounded recursion infrastructure from Phase E is the foundation for M3, M4, and M5

---

**Document created**: 2026-03-12
**Author**: Claude Code
