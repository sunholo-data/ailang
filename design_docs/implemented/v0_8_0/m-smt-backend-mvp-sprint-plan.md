# M-SMT-BACKEND-MVP: SMT Verification Backend (Phase 1)

**Status**: Planned
**Target**: v0.8.0
**Priority**: P2 (Medium)
**Estimated**: 25-35 hours (HIGH UNCERTAINTY for solver integration)
**Dependencies**:
- M-VERIFY-RUNTIME-CONTRACTS (v0.6.2) ✅
- M-VERIFY-CONTRACTS (v0.7.1) ✅
- M-CONTRACTS-OPLOWERING (v0.7.4) ✅
**Parent Design Doc**: [m-verify-smt-verification.md](m-verify-smt-verification.md)

---

## Problem Statement

AILANG has runtime contract enforcement (`requires`/`ensures`), but no way to **statically prove** that contracts hold for all inputs. Runtime checks catch violations at execution time; SMT verification proves correctness at compile time.

**Current State (v0.7.3)**:
- `requires { x >= 0 }` — parsed, elaborated, enforced at runtime
- `ensures { result >= 0 }` — checked after function returns
- No static verification: `ailang verify` does not exist
- Contract violations only discovered when specific inputs trigger them

**What This Unlocks**:
- `ailang verify module.ail` — prove contracts hold for ALL inputs
- Counterexample generation when contracts can be violated
- Confidence signal for AI-generated code ("verified" vs "runtime-only")
- Foundation for redundant generation with contract filtering (Phase 2)

**Research Foundation**: AWS ARC system (Bayless et al., 2025, arXiv:2511.09008) — neurosymbolic verification adapted for language-level contracts.

---

## Goals

**Primary Goal**: Build `ailang verify` command that proves contract satisfaction using Z3.

**Success Metrics**:
- [ ] `ailang verify examples/runnable/contracts/basic.ail` — verifies 4 functions
- [ ] `ailang verify examples/runnable/contracts/park.ail` — verifies park admission policy
- [ ] Counterexample shown when contract violated (with concrete variable bindings)
- [ ] Functions outside verifiable fragment gracefully skipped with clear reason
- [ ] `float` restricted to `smt-best-effort` with explicit warning
- [ ] Z3 invocation < 5s per function for typical contracts

**Non-Goals (Phase 1)**:
- Recursive function verification (requires induction — Phase 3)
- Higher-order function encoding (not in verifiable fragment)
- Redundant generation / confidence scoring (Phase 2)
- SharedMem invariants (Phase 3)
- Cross-module contract verification

---

## Verifiable Fragment

A function is eligible for SMT verification if **all** of:

| Restriction | Check | Rationale |
|-------------|-------|-----------|
| Has contracts | `len(meta.Contracts) > 0` | Nothing to verify without contracts |
| Pure effects | Effect annotation is `! {}` | Side effects not encodable in QF logic |
| Non-recursive | No self-reference in body | Recursion requires induction |
| No higher-order | No lambda args/returns | Functions not encodable in QF |
| Shallow ADT patterns | Match depth ≤ 1 | Deep nesting explodes encoding |
| QF expressions | No quantifiers in contracts | Quantifiers leave decidable fragment |

Functions **outside** the fragment still get runtime checks — they're just skipped for SMT with a clear diagnostic.

---

## Solution Design

### Architecture

```
ailang verify module.ail
    │
    ├── 1. Parse + Elaborate + Type Check (existing pipeline)
    │
    ├── 2. For each function with contracts:
    │       ├── IsSMTEncodable(func) → (bool, []RejectionReason)
    │       │   ├── Skip if not encodable (print reasons)
    │       │   └── Continue if encodable
    │       │
    │       ├── 3. Encode to SMT-LIB:
    │       │   ├── Declare types (int→Int, Season→Datatype, etc.)
    │       │   ├── Declare symbolic variables (function params)
    │       │   ├── Assert preconditions (requires clauses)
    │       │   ├── Define function body (let→let, if→ite, match→match)
    │       │   ├── Assert negated postcondition (ensures → check-sat)
    │       │   └── (check-sat) + (get-model)
    │       │
    │       └── 4. Invoke Z3:
    │           ├── unsat → Contract VERIFIED ✓
    │           ├── sat → COUNTEREXAMPLE found (extract model)
    │           └── unknown/timeout → Report inconclusive
    │
    └── 5. Print summary
```

### Type Mapping: AILANG → SMT-LIB

| AILANG Type | SMT-LIB | Notes |
|-------------|---------|-------|
| `int` | `Int` | Direct mapping |
| `float` | `Real` | **Not IEEE754** — `smt-best-effort` only, with warning |
| `bool` | `Bool` | Direct mapping |
| Enum ADT (`Season = LOW \| HIGH`) | `(declare-datatype Season ((LOW) (HIGH)))` | Nullary constructors |
| ADT with fields (`Tree = Leaf(int) \| ...`) | `(declare-datatype ...)` | Single-level only in Phase 1 |
| `string` | Not supported | Skip function, report reason |
| `[T]` (lists) | Not supported | Skip function, report reason |
| Records | Not supported | Skip function, report reason |

### Expression Encoding: Core → SMT-LIB

Incremental approach — each step builds on the previous:

**Step 1: Contracts only (uninterpreted function model)**

Encode preconditions and negated postconditions without encoding the body. This checks if the contracts themselves are internally consistent.

```smt2
; For: func f(x: int) -> int requires { x >= 0 } ensures { result >= 0 }
(declare-const x Int)
(declare-const result Int)
(assert (>= x 0))            ; requires
(assert (not (>= result 0))) ; negated ensures
(check-sat)
; sat → postcondition is NOT implied by precondition alone (expected for most functions)
; unsat → postcondition follows from precondition alone (trivially true)
```

**Step 2: Straight-line arithmetic**

```ailang
let y = x + 1;
let z = y * 2;
z
```
→
```smt2
(define-const y Int (+ x 1))
(define-const z Int (* y 2))
; result = z
```

**Step 3: `if/else` → SMT `ite`**

```ailang
if x > 0 then x else 0 - x
```
→
```smt2
(ite (> x 0) x (- 0 x))
```

**Step 4: `match` over enum → SMT datatype match**

```ailang
match season { LOW_SEASON => 5, HIGH_SEASON => 10 }
```
→
```smt2
(match season ((LOW_SEASON 5) (HIGH_SEASON 10)))
```

**Step 5: `match` over single-level ADT**

```ailang
match shape { Circle(r) => r * r, Rect(w, h) => w * h }
```
→
```smt2
(match shape
  ((Circle r) (* r r))
  ((Rect w h) (* w h)))
```

### Operator Mapping: Core → SMT-LIB

After OpLowering, contract expressions use builtin calls. For SMT encoding, we map the **original operators** (from pre-lowered contract `Expr` or from the builtin names):

| AILANG Op | SMT-LIB | Notes |
|-----------|---------|-------|
| `+` / `OpAdd` | `+` | Works for Int and Real |
| `-` / `OpSub` | `-` | Works for Int and Real |
| `*` / `OpMul` | `*` | Works for Int and Real |
| `/` / `OpDiv` | `div` (Int) / `/` (Real) | Integer vs real division |
| `%` / `OpMod` | `mod` | Int only |
| `==` / `OpEq` | `=` | All types |
| `!=` / `OpNe` | `(not (= ...))` | All types |
| `<` / `OpLt` | `<` | Int and Real |
| `<=` / `OpLe` | `<=` | Int and Real |
| `>` / `OpGt` | `>` | Int and Real |
| `>=` / `OpGe` | `>=` | Int and Real |
| `&&` / `OpAnd` | `and` | Bool |
| `\|\|` / `OpOr` | `or` | Bool |
| `not` / `OpNot` | `not` | Bool |
| `-` / `OpNeg` | `-` (unary) | Int and Real |

### SMT-LIB Generation: Concrete Example

For `park.ail`'s `admissionFee`:

```ailang
export func admissionFee(age: int, season: Season) -> int ! {}
requires { age >= 0 }
ensures { result >= 0 }
{
  match season {
    LOW_SEASON => if age < 5 then 0 else if age >= 65 then 5 else 15,
    HIGH_SEASON => if age >= 65 then 10 else 20
  }
}
```

Generated SMT-LIB:
```smt2
; Type declarations
(declare-datatype Season ((LOW_SEASON) (HIGH_SEASON)))

; Symbolic variables (function parameters)
(declare-const age Int)
(declare-const season Season)

; Preconditions
(assert (>= age 0))

; Function body encoding (result = body expression)
(define-fun body () Int
  (match season
    ((LOW_SEASON (ite (< age 5) 0 (ite (>= age 65) 5 15)))
     (HIGH_SEASON (ite (>= age 65) 10 20)))))

; Check ensures: negate postcondition
(assert (not (>= body 0)))

(check-sat)
; Expected: unsat (contract holds for all valid inputs)
```

### CLI Interface

```bash
# Verify all functions with contracts in a module
ailang verify examples/runnable/contracts/park.ail

# Output:
# Verifying examples/runnable/contracts/park...
#
# admissionFee:
#   requires: age >= 0                              [precondition]
#   ensures:  result >= 0                           [VERIFIED ✓] (0.03s)
#
# canEnterPark:
#   requires: age >= 0                              [precondition]
#   ⚠ No ensures clause — nothing to verify
#
# classifyAge:
#   requires: age >= 0                              [precondition]
#   ⚠ No ensures clause — nothing to verify
#
# Summary: 1 verified, 0 violations, 2 skipped (no ensures)

# Verbose mode — show SMT-LIB
ailang verify --verbose park.ail

# JSON output for tooling
ailang verify --json park.ail

# Strict mode — fail if any function not verifiable
ailang verify --strict park.ail
```

### Counterexample Display

When Z3 returns `sat` (contract violation found):

```
admissionFee:
  ensures: result >= 0                           [VIOLATION ✗]

  Counterexample:
    age    = -1
    season = LOW_SEASON

  Trace: admissionFee(-1, LOW_SEASON)
    → match LOW_SEASON → if -1 < 5 then 0 ...
    → result = 0
    → ensures (0 >= 0) = true

  Note: Precondition (age >= 0) would catch this.
        The ensures clause holds when preconditions are satisfied.
```

### Error: Function Not in Verifiable Fragment

```
admissionFee:
  ⚠ Skipped for SMT verification (runtime checks still active)
  Reasons:
    • RECURSIVE: Function body contains self-reference at park.ail:20:5
    • Hint: Extract non-recursive core logic into a helper function

  To override: ailang verify --force park.ail (may timeout or produce unsound results)
```

---

## Implementation Plan

### Phase 1: IsSMTEncodable + Rejection Reasons (~4h)

New file: `internal/smt/encodable.go` (~150 LOC)

```go
type SMTRejectionReason struct {
    Code     string // "NO_CONTRACTS", "RECURSIVE", "HIGHER_ORDER", "DEEP_ADT", "EFFECTFUL", "UNSUPPORTED_TYPE"
    Message  string // Human-readable explanation
    Location string // Source position
    Hint     string // Suggested fix
}

// IsSMTEncodable checks if a function can be verified with SMT.
// Returns (encodable, reasons) where reasons explain rejections.
func IsSMTEncodable(meta *core.DeclMeta, body core.CoreExpr, typeInfo types.CoreTypeInfo) (bool, []SMTRejectionReason)
```

Tasks:
- [ ] Implement `IsSMTEncodable()` with all 6 fragment checks
- [ ] `hasContracts(meta)` — trivial check
- [ ] `isPure(meta)` — check IsPure field
- [ ] `isNonRecursive(body, funcName)` — walk Core for self-references
- [ ] `hasNoHigherOrder(body)` — check for Lambda in args/returns
- [ ] `hasShallowPatterns(body)` — check Match depth ≤ 1
- [ ] `hasEncodableTypes(body, typeInfo)` — check no strings/lists/records in signature
- [ ] Unit tests for each check

### Phase 2: SMT-LIB Type Generation (~3h)

New file: `internal/smt/types.go` (~120 LOC)

```go
type SMTContext struct {
    types      map[string]string // AILANG type name → SMT-LIB declaration
    vars       []SMTVar          // Symbolic variables
    assertions []string          // SMT-LIB assertions
}

type SMTVar struct {
    Name    string
    SMTType string // "Int", "Real", "Bool", or datatype name
}

// MapType converts an AILANG type to SMT-LIB type string
func (ctx *SMTContext) MapType(t types.Type) (string, error)

// DeclareDatatype generates SMT-LIB datatype declaration for an ADT
func (ctx *SMTContext) DeclareDatatype(name string, variants []Variant) string
```

Tasks:
- [ ] Implement type mapping: `int`→`Int`, `float`→`Real`, `bool`→`Bool`
- [ ] Implement enum ADT → `declare-datatype`
- [ ] Implement single-level ADT with fields → `declare-datatype`
- [ ] Float warning when used with `Real`
- [ ] Unit tests for all type mappings

### Phase 3: SMT-LIB Expression Encoding (~8h)

New file: `internal/smt/codegen.go` (~400 LOC)

```go
// EncodeFunction generates complete SMT-LIB for a function's contracts
func (ctx *SMTContext) EncodeFunction(
    name string,
    params []Param,
    body core.CoreExpr,
    contracts []*core.Contract,
) (string, error)

// encodeExpr recursively translates Core AST to SMT-LIB
func (ctx *SMTContext) encodeExpr(expr core.CoreExpr) (string, error)
```

Incremental steps (each testable independently):
- [ ] Step 1: Encode `Lit` (int, float, bool literals)
- [ ] Step 2: Encode `Var` (symbolic variable references)
- [ ] Step 3: Encode binary ops (`App` with builtin refs → SMT operators)
- [ ] Step 4: Encode `If` → `ite`
- [ ] Step 5: Encode `Let` → `define-const` or nested `let`
- [ ] Step 6: Encode `Match` over enum → `match`
- [ ] Step 7: Encode `Match` over ADT with fields → `match` with bindings
- [ ] Step 8: Encode contract assertions (requires → assert, ensures → assert-not)
- [ ] Step 9: Generate complete SMT-LIB program (declarations + body + check-sat)
- [ ] Unit tests for each step

**Critical design choice**: We encode the **pre-lowered** Core AST (with `Intrinsic` nodes), not the post-lowered version (with `$builtin` calls). The SMT encoder reads directly from contract expressions and function bodies before OpLowering. This is simpler because `Intrinsic` nodes have clear operator semantics, whereas lowered `App($builtin.ge_Int, ...)` would need reverse-mapping.

**Pipeline integration**: The verify command runs the compilation pipeline up to type checking, then passes the Core program to the SMT encoder **before** OpLowering. This means contracts still have `Intrinsic` nodes with clear operator types.

### Phase 4: Z3 Solver Integration (~4h)

New file: `internal/smt/solver.go` (~200 LOC)

```go
type SolverResult struct {
    Status  SolverStatus // Verified, Counterexample, Unknown, Error
    Model   map[string]string // Variable → value (on counterexample)
    Time    time.Duration
    SMTLib  string // Generated SMT-LIB (for --verbose)
}

type SolverStatus int
const (
    Verified      SolverStatus = iota // unsat — contract holds
    Counterexample                     // sat — violation found
    Unknown                            // solver timeout/unknown
    SolverError                        // Z3 not found or crashed
)

// Solve invokes Z3 on SMT-LIB input and parses the result
func Solve(ctx context.Context, smtlib string, timeout time.Duration) (*SolverResult, error)
```

Tasks:
- [ ] Find Z3 binary (check PATH, common install locations, `AILANG_Z3_PATH` env var)
- [ ] Write SMT-LIB to temp file, invoke `z3 -smt2 file.smt2`
- [ ] Parse stdout: `sat`/`unsat`/`unknown`
- [ ] Parse `(model ...)` output for counterexample extraction
- [ ] Timeout handling (Z3 `-T:5` flag for 5-second timeout)
- [ ] Graceful error when Z3 not installed (suggest install command)
- [ ] Unit tests with mock solver (test parsing, not Z3 itself)
- [ ] Integration test with Z3 (skipped if Z3 not available)

### Phase 5: CLI Command + Reporting (~4h)

New file: `cmd/ailang/verify.go` (~200 LOC)

Tasks:
- [ ] Register `ailang verify <module>` command
- [ ] Flags: `--verbose` (show SMT-LIB), `--json` (machine output), `--strict` (fail on skip), `--timeout` (per-function)
- [ ] Run compilation pipeline up to type checking
- [ ] For each function: check encodability → encode → solve → report
- [ ] Human-readable output with colors (VERIFIED ✓ / VIOLATION ✗ / SKIPPED ⚠)
- [ ] JSON output format for tooling integration
- [ ] Summary line: N verified, N violations, N skipped

### Phase 6: Integration Testing + Examples (~4h)

Tasks:
- [ ] End-to-end test: `basic.ail` — all 4 functions verified
- [ ] End-to-end test: `park.ail` — admissionFee verified
- [ ] Test with intentionally broken contract (counterexample displayed)
- [ ] Test with non-verifiable function (graceful skip with reasons)
- [ ] Test with Z3 not available (clear error message)
- [ ] Test with float contracts (warning displayed)
- [ ] Add new example: `examples/runnable/contracts/verify_demo.ail`
- [ ] Update `examples/manifest.json` with verify examples
- [ ] Documentation: update `docs/docs/guides/contracts.mdx`

---

## Files to Create

```
internal/smt/
├── encodable.go      -- IsSMTEncodable() + rejection reasons (~150 LOC)
├── encodable_test.go -- Fragment check tests (~200 LOC)
├── types.go          -- AILANG→SMT-LIB type mapping (~120 LOC)
├── types_test.go     -- Type mapping tests (~100 LOC)
├── codegen.go        -- Core→SMT-LIB expression encoding (~400 LOC)
├── codegen_test.go   -- Expression encoding tests (~300 LOC)
├── solver.go         -- Z3 invocation + result parsing (~200 LOC)
└── solver_test.go    -- Solver integration tests (~150 LOC)

cmd/ailang/
└── verify.go         -- CLI command registration (~200 LOC)
```

**Total new code**: ~1,820 LOC (implementation + tests)

## Files to Modify

| File | Change |
|------|--------|
| `cmd/ailang/main.go` | Register `verify` subcommand |
| `examples/manifest.json` | Add verify demo example |
| `docs/docs/guides/contracts.mdx` | Add verification section |
| `CHANGELOG.md` | Add entry |

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | SMT verification is deterministic (same input → same proof) |
| A2: Replayability | +1 | Verification results are serializable proof artifacts |
| A5: Bounded Verification | +2 | **Primary goal** — enables local, automated contract proving |
| A7: Machines First | +2 | Machine-checkable correctness proofs |
| A8: Syntax Is Liability | 0 | No new syntax (reuses existing contracts) |
| A11: Structured Failure | +1 | Counterexamples are structured data with variable bindings |

**Net Score: +8** → **Decision: Proceed**

### Hard Violation Check
- [x] A1 (Determinism): Z3 is deterministic for QF fragment
- [x] A3 (Effects): Only verifies pure functions
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Core purpose is machine verification

---

## Sprint Plan

### Day 1: Foundation (~6h)
- [ ] Create `internal/smt/` package
- [ ] Implement `IsSMTEncodable()` with all 6 checks + tests
- [ ] Implement type mapping (`int`→`Int`, `bool`→`Bool`, enum→datatype) + tests
- [ ] Scaffold `SMTContext` struct

### Day 2: Expression Encoding (~8h)
- [ ] Encode literals, variables, binary operators
- [ ] Encode `If` → `ite`
- [ ] Encode `Let` → `define-const`
- [ ] Encode `Match` over enum → `match`
- [ ] Contract assertion encoding (requires/ensures)
- [ ] Unit tests for each encoding step

### Day 3: Solver + CLI (~6h)
- [ ] Z3 binary discovery + invocation
- [ ] Result parsing (sat/unsat/model extraction)
- [ ] `ailang verify` command with flags
- [ ] Human-readable reporting

### Day 4: Integration + Polish (~6h)
- [ ] End-to-end tests with `basic.ail` and `park.ail`
- [ ] Counterexample display
- [ ] Non-verifiable function skip reporting
- [ ] JSON output format
- [ ] Documentation + examples

### Day 5: Buffer + Edge Cases (~4h)
- [ ] Float handling with `Real` warnings
- [ ] ADT with fields (Match over constructors)
- [ ] Timeout handling
- [ ] Z3-not-installed error message
- [ ] Final review + cleanup

**Total: ~30 hours across 5 working days**

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Z3 not available on target systems | High | Medium | Clear install instructions, `AILANG_Z3_PATH` env var, skip with warning |
| SMT encoding scope balloon | Medium | High | Ship Steps 1-4 only; defer ADT fields to follow-up if needed |
| Function body encoding harder than expected | Medium | High | Fall back to contracts-only verification (uninterpreted function) |
| Float/Real semantic mismatch | Low | Medium | Restrict to `smt-best-effort` with explicit warning |
| Performance (solver timeout) | Low | Medium | 5s per-function timeout, `--timeout` flag |
| Pre-lowered vs post-lowered Core confusion | Medium | Low | SMT encoder reads Core before OpLowering; document clearly |

### De-Risk Strategy

The implementation is **incrementally shippable**:

1. **Minimum viable**: Contracts-only verification (no body encoding) — proves contract consistency
2. **Useful**: Straight-line + if/else body encoding — covers `basic.ail` examples
3. **Showcase**: Match encoding — covers `park.ail` (killer demo)

If scope balloons, ship at level 2 and defer match encoding.

---

## Testing Strategy

**Unit tests** (~750 LOC):
- `IsSMTEncodable()` — one test per rejection reason
- Type mapping — all AILANG→SMT-LIB conversions
- Expression encoding — one test per Core node type
- Solver result parsing — mock Z3 output

**Integration tests** (~200 LOC, Z3-gated):
- `basic.ail` — 4 functions verified end-to-end
- `park.ail` — admissionFee verified end-to-end
- Broken contract — counterexample extracted
- Non-verifiable function — graceful skip

**Test gating**: Integration tests that need Z3 use `testing.Short()` or check `exec.LookPath("z3")` — CI can skip if Z3 not available.

---

## Related Documents

- [m-verify-smt-verification.md](m-verify-smt-verification.md) — Parent design doc (Phases 1-3)
- [m-contracts-oplowering.md](m-contracts-oplowering.md) — Prerequisite (✅ complete)
- [m-contracts-assert.md](m-contracts-assert.md) — Parallel work (@assert syntax)
- [../../implemented/v0_7_1/m-verify-contracts.md](../../implemented/v0_7_1/m-verify-contracts.md) — Runtime contract enforcement
- [../../implemented/v0_6_1/m-verify-runtime-contracts.md](../../implemented/v0_6_1/m-verify-runtime-contracts.md) — Phase 0+0.5 implementation

---

**Document created**: 2026-02-10
**Author**: Claude Code
