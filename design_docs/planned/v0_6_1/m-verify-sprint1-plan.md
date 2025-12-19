# Sprint Plan: M-VERIFY Phase 0 + 0.5 - Contract Infrastructure & Runtime Checks

## Summary

Implement foundational contract infrastructure (AST, parser, lexer) and runtime contract checks, delivering immediate value before the high-uncertainty SMT phase. This sprint establishes `requires`/`ensures` syntax and generates Go runtime checks that panic on contract violations.

**Duration:** 3-4 days (~15-18 hours)
**Dependencies:** None - builds on existing Property/Debug infrastructure
**Risk Level:** Low - leverages existing patterns extensively
**Target Version:** v0.6.2

---

## ✅ SPRINT COMPLETE (December 2025)

**Completed:** 2025-12-19
**Actual Duration:** ~10-12 hours across 2 sessions

### Implementation Summary

All milestones completed. Runtime contract checking is now available via `--verify-contracts` flag.

| Milestone | Status | Actual vs Estimate | Notes |
|-----------|--------|-------------------|-------|
| M1-TOKENS | ✅ Complete | On target | Added `REQUIRES`, `ENSURES` tokens |
| M2-AST | ✅ Complete | Slight deviation | Used `core.Contract` instead of extending `ast.Property` |
| M3-PARSER | ✅ Complete | On target | `requires { ... }` and `ensures { ... }` syntax |
| M4-RUNTIME | ✅ Complete | Simplified | Generated inline, not via EffContext (for codegen path) |
| M5-CODEGEN | ✅ Complete | On target | `--verify-contracts` flag, runtime panics |
| M6-INTEGRATION | ✅ Complete | On target | Examples + end-to-end tests |

### Key Implementation Decisions

1. **Contract Storage**: Stored in `core.DeclMeta.Contracts` (not `ast.FuncDecl.Properties`)
   - Reason: Contracts need Core expressions for codegen, not surface AST
   - `core.Contract` struct has `Kind`, `Expr`, `Message`, `Location` fields

2. **Predicate Evaluation - Requires**: Uses runtime helpers (`GeInt`, `NeInt`, `LeInt`, etc.)
   - Reason: `_impl` functions use `interface{}` params; can't use raw Go operators
   - Added `mapIntrinsicToHelper()` to map Core intrinsics to runtime helpers

3. **Predicate Evaluation - Ensures**: Uses typed Go operators in typed wrapper
   - Reason: Typed wrapper has concrete types; can use native Go comparisons
   - Added `generateEnsuresPredicate()` with `result` → `_result` substitution
   - Handles `core.Intrinsic` nodes (elaborated comparison ops)

4. **Comments Always Generated**: `// Requires:` and `// Ensures:` comments generated regardless of `--verify-contracts`
   - Reason: Documentation value even without runtime checks
   - Panic checks only generated when flag is set

5. **Added `--relax-modules` to compile command**: Needed for example files in absolute paths
   - Bonus fix for developer ergonomics
   - Also supports `AILANG_RELAX_MODULES=1` environment variable

### Files Created/Modified

**New Files:**
- `internal/gen/golang/contracts_integration_test.go` - End-to-end tests (~250 LOC)

**Modified Files:**
- `internal/gen/golang/codegen.go` - Added `verifyContracts`, `currentFuncName` fields
- `internal/gen/golang/codegen_decl.go` - Call contract checks in `generateImplFunc`
- `internal/gen/golang/contracts.go` - Implemented `generateContractRequiresChecks()`
- `internal/gen/golang/codegen_ops.go` - Added `mapIntrinsicToHelper()` for comparisons
- `internal/gen/golang/codegen_runtime_arith.go` - Added `NeInt` runtime helper
- `cmd/ailang/compile.go` - Added `--verify-contracts` and `--relax-modules` flags
- `internal/gen/golang/contracts_test.go` - Updated tests for new behavior

### Generated Code Example

With `--verify-contracts`:
```go
func absolute_impl(x interface{}) interface{} {
    // Requires: (x >= 0)
    if !(GeInt(x, int64(0))).(bool) {
        panic("contract violation: requires: (x >= 0) at examples/runnable/contracts/basic.ail:13:12")
    }
    return x
}
```

Without `--verify-contracts` (documentation only):
```go
func absolute_impl(x interface{}) interface{} {
    // Requires: (x >= 0)
    return x
}
```

### Test Results

All tests pass:
- `TestContractViolation_Integration` - Compiles AILANG, runs Go tests, verifies panics
- `TestContractViolation_NoVerify` - Verifies comments generated without panics
- `TestGenerateContractRequiresChecks_WithPredicates` - Unit test for codegen
- `TestGenerateContractRequiresChecks_Disabled` - Unit test for disabled mode

---

## Suggested Next Steps

### Immediate (v0.6.2) - ALL COMPLETE ✅

1. ~~**Ensures checks**~~ ✅ DONE
   - Added `generateContractEnsuresChecks()` to inject checks before returns
   - Uses `_result` variable to capture return value
   - Handles `core.Intrinsic` nodes with `intrinsicOpToString()` helper

2. ~~**Park admission example**~~ ✅ DONE
   - Created `examples/runnable/contracts/park.ail` from ARC paper
   - Full policy model with ADTs, contracts, and test cases

3. ~~**Documentation**~~ ✅ DONE
   - Updated `docs/docs/guides/contracts.mdx` with:
     - Runtime contract checking section
     - Generated code examples
     - Contract violation message format
     - Updated implementation status

### Future Sprints

**Phase 1: SMT Backend MVP** (HIGH UNCERTAINTY - 25-35h)
- `ailang verify` command with Z3 integration
- `IsSMTEncodable()` function for verifiable fragment detection
- Incremental body encoding (let → ite → match)

**Phase 2: Redundant Generation** (10-15h)
- AST normalization for structural comparison
- Multi-sample LLM codegen with contract filtering
- Confidence scoring

---

## Current Status Analysis

### Infrastructure Ready for Reuse

| Component | Status | Reuse Potential |
|-----------|--------|-----------------|
| `Property` struct (`ast_decl.go:69-74`) | ✅ Ready | Extend with `ContractKind` |
| `Binder` struct (`ast_decl.go:76-80`) | ✅ Ready | Direct reuse for type annotations |
| `FuncDecl.Properties` field | ✅ Ready | Store contracts here |
| `parseProperty()` (`parser_testing.go:215-285`) | ✅ Ready | Adapt for contracts |
| `parseBinder()` (`parser_testing.go:288-318`) | ✅ Ready | Direct reuse |
| `DebugContext` pattern (`effects/debug.go`) | ✅ Ready | Clone for `ContractContext` |
| `FORALL`, `PROPERTY` tokens | ✅ Exist | Add `REQUIRES`, `ENSURES`, `INVARIANT` |

### Velocity Analysis

Recent development velocity (last 7 days):
- M-DX-CHECK-DIR: ~150 LOC in 1 day
- M-ENV-TYPE fix: ~30 LOC in 0.5 days
- M-LETREC-SCOPING fix: ~100 LOC in 1 day

**Estimated capacity:** 150-200 LOC/day for focused features
**Sprint budget:** 450-700 LOC over 3-4 days

### Remaining from Design Doc

- **Phase 0** (this sprint): Contract AST + parser + lexer tokens
- **Phase 0.5** (this sprint): Runtime contract checks
- Phase 1: SMT Backend MVP (HIGH UNCERTAINTY - future sprint)
- Phase 2: Redundant Generation (future sprint)
- Phase 3: SharedMem Invariants (future sprint)

## Proposed Milestones

### Milestone 1: Contract Lexer Tokens (M1-TOKENS)

**Goal:** Add `REQUIRES`, `ENSURES`, `INVARIANT` tokens to the lexer
**Estimated:** 15 LOC implementation + 10 LOC tests = 25 LOC
**Duration:** 0.5 hours

**Tasks:**
- Add token constants to `internal/lexer/token.go`
- Add keyword mapping in `keywords` map
- Add to `TestKeywords` group

**Files to Modify:**
- `internal/lexer/token.go` (+15 LOC)
- `internal/lexer/token_test.go` (+10 LOC)

**Acceptance Criteria:**
- [ ] `lexer.REQUIRES`, `lexer.ENSURES`, `lexer.INVARIANT` tokens exist
- [ ] Keywords `requires`, `ensures`, `invariant` tokenize correctly
- [ ] Existing tests pass
- [ ] `make lint` clean

**Risks:** None - trivial addition following existing pattern

---

### Milestone 2: ContractKind Enum in AST (M2-AST)

**Goal:** Extend `Property` struct with `ContractKind` to distinguish requires/ensures/invariant from property-based tests
**Estimated:** 30 LOC implementation + 20 LOC tests = 50 LOC
**Duration:** 1 hour

**Tasks:**
- Add `ContractKind` enum with `PropertyKind`, `RequiresKind`, `EnsuresKind`, `InvariantKind`
- Add `Kind ContractKind` field to `Property` struct
- Update `Property` constructor usage (if any)
- Add `String()` method for debugging

**Files to Modify:**
- `internal/ast/ast_decl.go` (+30 LOC)
- `internal/ast/ast_decl_test.go` (+20 LOC if exists, or create)

**Acceptance Criteria:**
- [ ] `ContractKind` enum defined with 4 variants
- [ ] `Property.Kind` field added
- [ ] Existing code continues to work (backwards compatible)
- [ ] `make test` passes

**Risks:** Low - simple addition, backwards compatible

---

### Milestone 3: Contract Parser Extension (M3-PARSER)

**Goal:** Parse `requires { pred1, pred2 }` and `ensures { pred1, pred2 }` blocks after function signature
**Estimated:** 120 LOC implementation + 80 LOC tests = 200 LOC
**Duration:** 1.5-2 days

**Tasks:**
- Create `parseRequiresBlock()` - parse `requires { expr1, expr2, ... }`
- Create `parseEnsuresBlock()` - parse `ensures { expr1, expr2, ... }`
- Extend `parseFuncDecl()` to check for `requires`/`ensures` after effect annotation
- Handle `result` as reserved identifier in ensures clauses
- Store contracts in `FuncDecl.Properties` with appropriate `ContractKind`

**Syntax Target:**
```ailang
export func foo(x: int) -> int ! {}
requires { x >= 0 }
ensures  { result > x }
{
  x + 1
}
```

**Files to Modify:**
- `internal/parser/parser_contracts.go` (NEW, ~100 LOC)
- `internal/parser/parser_decl.go` (+20 LOC to wire in contract parsing)
- `internal/parser/parser_contracts_test.go` (NEW, ~80 LOC)

**Acceptance Criteria:**
- [ ] `requires { ... }` block parses into `Property` with `RequiresKind`
- [ ] `ensures { ... }` block parses into `Property` with `EnsuresKind`
- [ ] `result` identifier works in ensures expressions
- [ ] Multiple predicates separated by commas work
- [ ] Contracts stored in `FuncDecl.Properties`
- [ ] Error messages for malformed contracts are clear
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- **Medium:** Parser cursor positioning - Mitigation: Follow existing `parseProperty()` pattern exactly, use `DEBUG_PARSER=1` for debugging

---

### Milestone 4: ContractContext Runtime Support (M4-RUNTIME)

**Goal:** Create `ContractContext` following `DebugContext` pattern for collecting contract check results
**Estimated:** 120 LOC implementation + 60 LOC tests = 180 LOC
**Duration:** 0.5-1 day

**Tasks:**
- Create `ContractContext` struct with `checks []ContractCheck`, `mode ContractMode`
- Create `ContractCheck` struct (Kind, Passed, Message, Location, Function)
- Create `ContractMode` enum (Panic, Report, Off)
- Add `Contract *ContractContext` field to `EffContext`
- Implement `Check()`, `Collect()`, `Reset()`, `HasViolations()` methods
- Wire to `EffContext` initialization

**Files to Create:**
- `internal/effects/contracts.go` (NEW, ~120 LOC)
- `internal/effects/contracts_test.go` (NEW, ~60 LOC)

**Files to Modify:**
- `internal/effects/context.go` (+5 LOC - add Contract field)

**Acceptance Criteria:**
- [ ] `ContractContext` follows `DebugContext` API pattern
- [ ] `ContractMode` supports Panic/Report/Off modes
- [ ] `ContractCheck` captures Kind, Passed, Message, Location, Function
- [ ] Wired to `EffContext`
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:** Low - cloning proven pattern

---

### Milestone 5: Codegen for Runtime Contract Checks (M5-CODEGEN)

**Goal:** Generate Go runtime checks at function entry (requires) and exit (ensures)
**Estimated:** 150 LOC implementation + 70 LOC tests = 220 LOC
**Duration:** 1-1.5 days

**Tasks:**
- Detect contracts in `FuncDecl` during codegen
- Generate requires checks at function entry
- Generate ensures checks before return
- Handle `result` identifier mapping to return value
- Support panic mode (immediate panic) and report mode (collect + continue)
- Add `--verify-contracts[=panic|report]` CLI flag

**Generated Go Pattern (panic mode):**
```go
func (r *Runtime) Foo(x int64) int64 {
    // Requires check
    if !(x >= 0) {
        panic(fmt.Sprintf("contract violation: requires x >= 0, got x=%v", x))
    }

    result := r.Foo_impl(x)

    // Ensures check
    if !(result > x) {
        panic(fmt.Sprintf("contract violation: ensures result > x, got result=%v, x=%v", result, x))
    }

    return result
}
```

**Files to Modify:**
- `internal/gen/golang/codegen_contracts.go` (NEW, ~100 LOC)
- `internal/gen/golang/codegen_decl.go` (+30 LOC - wire in contract codegen)
- `internal/gen/golang/codegen_contracts_test.go` (NEW, ~50 LOC)
- `cmd/ailang/main.go` (+20 LOC - add CLI flag)

**Acceptance Criteria:**
- [ ] Requires checks generated at function entry
- [ ] Ensures checks generated before return
- [ ] `result` maps correctly to return value
- [ ] Panic mode triggers Go panic on violation
- [ ] Report mode collects violations in ContractContext
- [ ] `--verify-contracts` flag enables checks
- [ ] Generated code compiles and runs
- [ ] `make test` passes
- [ ] `make lint` clean

**Risks:**
- **Medium:** Ensures check placement with multiple returns - Mitigation: Generate single return point wrapper
- **Low:** String interpolation of expressions - Mitigation: Use AST pretty-printer

---

### Milestone 6: Integration & Examples (M6-INTEGRATION)

**Goal:** End-to-end integration test and example files demonstrating contracts
**Estimated:** 50 LOC examples + 50 LOC tests = 100 LOC
**Duration:** 0.5 days

**Tasks:**
- Create `examples/contracts/basic.ail` - simple requires/ensures
- Create `examples/contracts/park.ail` - ARC paper showcase (from design doc)
- Create integration test that compiles and runs contract-annotated code
- Verify panic on violation
- Document in example file headers

**Files to Create:**
- `examples/contracts/basic.ail` (NEW, ~30 LOC)
- `examples/contracts/park.ail` (NEW, ~50 LOC from design doc Example 0)
- `internal/pipeline/contracts_integration_test.go` (NEW, ~50 LOC)

**Acceptance Criteria:**
- [ ] `examples/contracts/basic.ail` compiles and runs
- [ ] `examples/contracts/park.ail` reproduces ARC paper example
- [ ] Contract violations trigger panic with clear message
- [ ] Integration test covers happy path and violation path
- [ ] `make verify-examples` includes contract examples
- [ ] `make test` passes

**Risks:** Low - straightforward integration

---

## Success Metrics

- **Test coverage:** >80% for new contract packages
- **Examples passing:** 2 new contract examples working
- **Total LOC:** ~775 LOC (implementation + tests)
- **Documentation:** Design doc updated with implementation notes
- **All tests passing:** ✅
- **All linting passing:** ✅

## Dependencies

- None - builds on existing infrastructure

## Open Questions

1. **Should `result` be a reserved keyword or context-sensitive identifier?**
   - Recommendation: Context-sensitive (only special in ensures blocks)

2. **Should contracts be enabled by default or opt-in?**
   - Recommendation: Opt-in via `--verify-contracts` flag initially

3. **Should we support contracts on lambdas or only named functions?**
   - Recommendation: Named functions only for v0.6.2 (simpler)

## Notes

- This sprint delivers Phase 0 + Phase 0.5 from the M-VERIFY design doc
- Runtime checks provide immediate value before SMT phase
- Following `DebugContext` pattern ensures consistency and reduces implementation risk
- Contracts stored in existing `FuncDecl.Properties` field minimizes AST churn
- Phase 1 (SMT) is HIGH UNCERTAINTY and should be a separate sprint

## Next Sprint (Phase 1 - SMT Backend)

After this sprint completes, the next sprint will tackle SMT backend:
- Estimated: 25-35 hours (HIGH UNCERTAINTY)
- Key deliverable: `ailang verify` command with Z3 integration
- Risk: SMT encoding complexity may balloon scope
