# Sprint Plan: M-SMT-BACKEND-MVP

## Summary

Build `ailang verify` — an SMT-based contract verification command that proves `requires`/`ensures` clauses hold for all inputs using Z3. This is the killer feature that turns AILANG contracts from runtime safety nets into compile-time proofs.

**Duration:** 5 days (~30 hours)
**Dependencies:** M-CONTRACTS-OPLOWERING (✅ complete), Z3 solver (install via `brew install z3`)
**Risk Level:** Medium-High (SMT encoding has scope balloon risk)

## Current Status Analysis

### Completed Recently
- ✅ M-CONTRACTS-OPLOWERING: Contract expressions lowered through OpLowering (~20 LOC, same session)
- ✅ M-VERIFY-CONTRACTS (v0.7.1): Language-wide runtime contract enforcement
- ✅ M-VERIFY-RUNTIME-CONTRACTS (v0.6.2): Phase 0+0.5 parsing + runtime checks
- ✅ M-STRUCTURED-AI-OUTPUT (v0.7.3): ~370 LOC
- ✅ M-STDLIB-XML (v0.7.3): ~530 LOC impl + ~530 LOC tests
- ✅ M-STDLIB-ZIP (v0.7.3): ~315 LOC impl + ~388 LOC tests

### Velocity
- Recent average: ~200-300 LOC/day (implementation + tests combined)
- AILANG infrastructure sprints: typically 3-5 days for ~600-1200 LOC
- Estimated capacity: ~1,500-1,800 LOC over 5 days (conservative)
- Target for this sprint: ~1,820 LOC (achievable given existing contract infrastructure)

### Remaining from Design Doc
- ⏳ Phase 1: IsSMTEncodable + rejection reasons (~150 LOC)
- ⏳ Phase 2: SMT-LIB type generation (~120 LOC)
- ⏳ Phase 3: SMT-LIB expression encoding (~400 LOC)
- ⏳ Phase 4: Z3 solver integration (~200 LOC)
- ⏳ Phase 5: CLI command + reporting (~200 LOC)
- ⏳ Phase 6: Tests (~750 LOC)

## Proposed Milestones

### Milestone 1: Fragment Checker + Type Mapping
**Goal:** Determine which functions can be SMT-verified and map AILANG types to SMT-LIB types
**Estimated:** 270 implementation + 300 tests = 570 LOC
**Duration:** 1 day

**Tasks:**
- Create `internal/smt/` package with `encodable.go`
- Implement `IsSMTEncodable()` with 6 fragment checks:
  - `hasContracts(meta)` — check contracts exist
  - `isPure(meta)` — check `IsPure` field on DeclMeta
  - `isNonRecursive(body, funcName)` — walk Core AST for self-references
  - `hasNoHigherOrder(body)` — reject Lambda in args/returns
  - `hasShallowPatterns(body)` — check Match depth ≤ 1
  - `hasEncodableTypes(body, typeInfo)` — reject string/list/record types
- Implement `SMTRejectionReason` struct with Code, Message, Location, Hint
- Implement `SMTContext` struct and `MapType()`:
  - `int` → `Int`, `float` → `Real`, `bool` → `Bool`
  - Enum ADT → `(declare-datatype Name ((Variant1) (Variant2)))`
  - Single-level ADT with fields → `(declare-datatype ...)`
- Implement `DeclareDatatype()` for enum/ADT generation
- Write unit tests for all 6 rejection checks
- Write unit tests for all type mappings

**Acceptance Criteria:**
- [ ] `IsSMTEncodable(admissionFee)` returns `(true, [])` for park.ail
- [ ] `IsSMTEncodable(recursiveFunc)` returns `(false, [{Code: "RECURSIVE", ...}])`
- [ ] `MapType(TInt)` returns `"Int"`
- [ ] `DeclareDatatype("Season", ["LOW_SEASON", "HIGH_SEASON"])` returns valid SMT-LIB
- [ ] All tests passing, `go vet` clean

**Risks:**
- ADT type information may not be easily accessible from Core AST — Mitigation: check elaborator's constructor map or type checker output
- `IsPure` may not be reliably set on all functions — Mitigation: also check effect annotation directly

---

### Milestone 2: Expression Encoder (Core → SMT-LIB)
**Goal:** Translate AILANG Core AST expressions to SMT-LIB format
**Estimated:** 400 implementation + 300 tests = 700 LOC
**Duration:** 1.5 days

**Tasks:**
- Implement `encodeExpr()` recursive encoder in `codegen.go`:
  - Step 1: `Lit` → SMT literals (`42`, `3.14`, `true`)
  - Step 2: `Var` → SMT symbolic variables
  - Step 3: `Intrinsic` → SMT operators (`>=` → `>=`, `&&` → `and`, etc.)
  - Step 4: `If` → `(ite condition then else)`
  - Step 5: `Let` → `(let ((name value)) body)` or `(define-const name type value)`
  - Step 6: `Match` over enum → `(match var ((Variant1 body1) (Variant2 body2)))`
  - Step 7: `Match` over ADT with fields → `(match var ((Ctor field ...) body))`
- Implement `EncodeFunction()` — complete SMT-LIB program:
  - Type declarations (datatypes)
  - Symbolic variable declarations (function params)
  - Precondition assertions (`requires` → `(assert expr)`)
  - Function body definition (`(define-fun body ...)`)
  - Negated postcondition (`ensures` → `(assert (not expr))`)
  - `(check-sat)` + `(get-model)`
- Handle `result` variable in `ensures` (bind to body expression)
- Handle post-lowered expressions: detect `App($builtin.ge_Int, ...)` → map back to SMT `>=`

**Key Design Decision:** Read from **post-OpLowering** Core (since M-CONTRACTS-OPLOWERING now lowers contract expressions). Detect `VarGlobal` with `$builtin` module and map builtin names back to SMT operators.

**Acceptance Criteria:**
- [ ] `encodeExpr(Lit{IntLit, 42})` returns `"42"`
- [ ] `encodeExpr(If{...})` returns `"(ite ...)"`
- [ ] `EncodeFunction(admissionFee)` produces valid SMT-LIB
- [ ] Generated SMT-LIB for `basic.ail:absolute` is syntactically correct
- [ ] Unit tests for each Core node type

**Risks:**
- Post-lowered expressions use `App(VarGlobal($builtin.ge_Int), [arg1, arg2])` which requires reverse-mapping — Mitigation: create `builtinToSMTOp` map from builtin name → SMT operator
- Nested `App` for curried builtins — Mitigation: handle `App(App(builtin, arg1), arg2)` pattern

---

### Milestone 3: Z3 Solver Integration
**Goal:** Invoke Z3 on generated SMT-LIB and parse results
**Estimated:** 200 implementation + 150 tests = 350 LOC
**Duration:** 1 day

**Pre-requisite:** Install Z3 (`brew install z3`)

**Tasks:**
- Implement Z3 binary discovery:
  - Check `AILANG_Z3_PATH` env var
  - Check PATH (`exec.LookPath("z3")`)
  - Check common locations (`/opt/homebrew/bin/z3`, `/usr/local/bin/z3`)
- Implement `Solve()` function:
  - Write SMT-LIB to temp file
  - Invoke `z3 -smt2 -T:5 file.smt2` (5s timeout)
  - Capture stdout + stderr
- Parse solver output:
  - `unsat` → `Verified`
  - `sat` + `(model ...)` → `Counterexample` with variable bindings
  - `unknown` → `Unknown`
  - Non-zero exit → `SolverError`
- Implement model extraction:
  - Parse `(define-fun varName () Type value)` lines
  - Extract variable name → value mappings
  - Handle Int, Bool, Real, Datatype values
- Implement `SolverResult` struct with Status, Model, Time, SMTLib fields
- Write mock-solver tests (test parsing logic without Z3)
- Write Z3-gated integration test (skip if Z3 not available)

**Acceptance Criteria:**
- [ ] `Solve("(check-sat) (exit)")` invokes Z3 and returns result
- [ ] `Verified` returned for unsat results
- [ ] `Counterexample` returned with parsed model for sat results
- [ ] Graceful error message when Z3 not installed
- [ ] Timeout handling works (5s default)
- [ ] Tests pass with or without Z3 installed

**Risks:**
- Z3 output format varies between versions — Mitigation: test with Z3 4.15.4 (Homebrew current), parse conservatively
- Model parsing for datatypes — Mitigation: start with Int/Bool models, add datatype parsing incrementally

---

### Milestone 4: CLI Command + End-to-End Integration
**Goal:** Working `ailang verify` command with human-readable output
**Estimated:** 200 implementation + 200 tests = 400 LOC
**Duration:** 1.5 days

**Tasks:**
- Register `ailang verify <module>` subcommand in `cmd/ailang/main.go`
- Implement `verify.go`:
  - Parse + Elaborate + Type Check (reuse existing pipeline)
  - Iterate functions with contracts
  - For each: IsSMTEncodable → EncodeFunction → Solve → Report
- Implement flags:
  - `--verbose` — show generated SMT-LIB
  - `--json` — machine-readable output
  - `--strict` — fail if any function cannot be verified
  - `--timeout <duration>` — per-function Z3 timeout (default: 5s)
- Implement human-readable output:
  - `VERIFIED ✓` (green) / `VIOLATION ✗` (red) / `SKIPPED ⚠` (yellow)
  - Counterexample display with variable bindings
  - Rejection reasons for non-verifiable functions
  - Summary line: N verified, N violations, N skipped
- Implement JSON output format
- End-to-end integration tests:
  - `basic.ail` — all 4 functions verified
  - `park.ail` — admissionFee verified
  - Intentionally broken contract → counterexample shown
  - Non-verifiable function → skip with reasons
  - Z3 not available → clear error
- Update `examples/manifest.json` with verify examples
- Update `docs/docs/guides/contracts.mdx` with verification section
- Update `CHANGELOG.md`

**Acceptance Criteria:**
- [ ] `ailang verify examples/runnable/contracts/basic.ail` runs end-to-end
- [ ] `ailang verify examples/runnable/contracts/park.ail` verifies admissionFee
- [ ] Counterexample displayed for broken contracts
- [ ] `--json` output is valid JSON
- [ ] `--verbose` shows SMT-LIB
- [ ] Non-verifiable functions show rejection reasons
- [ ] All tests passing, linting clean
- [ ] CHANGELOG.md updated
- [ ] Documentation updated

**Risks:**
- Pipeline integration may need adjustments for verify-only mode (no codegen) — Mitigation: reuse `ailang check` pipeline path, stop after type checking
- Output formatting complexity — Mitigation: start with minimal text output, add colors/JSON later

---

## Pause Point: Review After Milestone 2

After Milestone 2 (expression encoding), pause for review before proceeding to solver integration. This is because:
1. Expression encoding is the highest-risk milestone (scope balloon)
2. We can validate SMT-LIB output manually before connecting to Z3
3. If encoding is harder than expected, we can ship contracts-only verification

**Review criteria:**
- Does generated SMT-LIB for `park.ail:admissionFee` look correct?
- Are all needed Core node types handled?
- Is the encoding maintainable?

---

## Success Metrics

- Test coverage: >90% for new `internal/smt/` package
- Examples verified: `basic.ail` (4 functions) + `park.ail` (1 function)
- Documentation: `docs/docs/guides/contracts.mdx` updated with verification section
- All tests passing: ✅
- All linting passing: ✅
- New example: `examples/runnable/contracts/verify_demo.ail`

## Dependencies

- **Z3 solver**: Must be installed (`brew install z3`) for integration tests
- **M-CONTRACTS-OPLOWERING**: ✅ Complete (just done in this session)
- **Core AST types**: May need to understand ADT constructor info from elaborator

## Open Questions

1. **ADT constructor info**: Where does the elaborator store ADT variant information? Need this to generate `declare-datatype`. Check `internal/elaborate/` for constructor maps.
2. **Effect annotation access**: Is `DeclMeta.IsPure` reliably set, or do we need to check effect annotations directly from the type checker?
3. **CI Z3**: Should we add Z3 to CI? Or gate integration tests with `testing.Short()`?

## Notes

- Sprint uses incremental encoding strategy: each step is independently testable
- The "killer demo" is `park.ail:admissionFee` — prioritize getting this to verify end-to-end
- If scope balloons on expression encoding, ship with contracts-only verification (uninterpreted function model) as fallback
- Z3 4.15.4 available via Homebrew (`brew install z3`)
