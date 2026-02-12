# M-SMT-FRAGMENT-EXPANSION: Expanding the SMT Decidable Fragment

**Status**: Planned
**Target**: v0.8.0+
**Priority**: P2 (Medium)
**Estimated**: 4-6 weeks (phased, each phase independently shippable)
**Dependencies**:
- M-SMT-BACKEND (v0.7.4) ✅ — `ailang verify` with Z3 integration
- M-CONTRACTS-OPLOWERING (v0.7.4) ✅ — contracts without `--experimental-binop-shim`
- M-VERIFY-RUNTIME-CONTRACTS (v0.6.2) ✅ — runtime contract enforcement

**Parent Design Doc**: [m-verify-smt-verification.md](m-verify-smt-verification.md)

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Z3 verification remains deterministic for all new fragment areas |
| A2: Replayability | +1 | Expanded proofs are serializable (SMT-LIB output) |
| A3: Effect Legibility | 0 | No change — effects remain explicitly excluded from SMT |
| A4: Explicit Authority | 0 | No ambient access changes |
| A5: Bounded Verification | +2 | **Primary goal** — more functions locally provable without global reasoning |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | More machine-checkable correctness proofs |
| A8: Syntax Is Liability | 0 | No new syntax — reuses existing contracts |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Cross-function inlining enables compositional proofs |
| A11: Structured Failure | +1 | Counterexamples now cover more function shapes |
| A12: System Boundary | 0 | No change |

**Net Score: +8** → **Decision: Proceed**

**Hard Violation Check:**
- [x] A1 (Determinism): All expansions stay within decidable SMT theories
- [x] A3 (Effects): Effects remain excluded (fundamental limitation)
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Every expansion improves machine analysis

---

## Problem Statement

`ailang verify` (v0.7.4) proves contracts correct for functions using **int/bool/enum ADTs** with arithmetic, comparison, logical operators, if/else, let bindings, and match expressions. This covers ~40% of typical AILANG functions.

**Current limitations** (with classification):

| Feature | Current Status | Expandability | Z3 Theory |
|---------|---------------|---------------|-----------|
| Strings | SKIPPED | ✅ Implementation choice | `str.++`, `str.len`, `str.contains` |
| Lists/sequences | SKIPPED | ✅ Implementation choice | `seq.*` (Z3 sequence theory) |
| Records | SKIPPED | ✅ Implementation choice | `declare-datatype` with accessors |
| Cross-function calls | SKIPPED ("unknown constant") | ✅ Implementation choice | `define-fun` or inlining |
| Bounded recursion | SKIPPED | ⚠️ Partially expandable | Loop unrolling (bounded depth) |
| Higher-order functions | SKIPPED | ⚠️ Partially expandable | Monomorphization/inlining |
| Unbounded recursion | SKIPPED | ❌ Fundamental (undecidable) | Would require induction proofs |
| Effectful functions | SKIPPED | ❌ Fundamental | Nondeterminism breaks proofs |

**Impact of current limitations:**
- Functions calling other user-defined functions (e.g., `netIncome` calling `calculateTax`) must duplicate logic inline — poor UX
- String-heavy business logic (e.g., validation, formatting) cannot be verified
- List processing (e.g., filtering, aggregation) cannot be verified
- Record field access patterns cannot be verified

**Coverage estimate after expansion**: ~70-80% of typical AILANG functions (up from ~40%)

---

## Goals

**Primary Goal**: Expand the SMT-verifiable fragment from int/bool/enum to include strings, lists, records, cross-function calls, and bounded recursion — covering ~70-80% of typical AILANG functions.

**Success Metrics**:
- [ ] String contracts verify (e.g., `ensures { result.length >= 0 }`)
- [ ] List contracts verify (e.g., `ensures { result.length == xs.length }`)
- [ ] Record field access works in contracts
- [ ] Functions calling other verified functions compose
- [ ] Simple recursive functions verify via bounded unrolling (depth ≤ 5)
- [ ] Clear diagnostics for remaining non-verifiable cases
- [ ] All existing `ailang verify` tests continue to pass

**Non-Goals**:
- Full dependent types or proof assistant semantics
- Unbounded recursion verification (requires induction — deferred)
- Effectful function verification (fundamental limitation)
- Higher-order function verification beyond monomorphic inlining

---

## Solution Design

### Overview: Phased Expansion

Each phase is **independently shippable** and adds a new Z3 theory:

```
Phase A: Cross-Function Calls (define-fun)
    │
Phase B: String Verification (Z3 String Theory)
    │
Phase C: Record Verification (SMT-LIB Datatypes)
    │
Phase D: List Verification (Z3 Sequence Theory)
    │
Phase E: Bounded Recursion (Loop Unrolling)
```

Phases A-D are pure implementation work (Z3 has the theories). Phase E requires careful design for soundness.

---

### Phase A: Cross-Function Calls (~8h)

**Current problem**: Functions calling other user-defined functions produce `unknown constant` errors because the SMT encoder verifies each function in isolation.

```ailang
-- Currently FAILS with Z3: "unknown constant calculateTax"
export func netIncome(gross: int, bracket: TaxBracket) -> int ! {}
requires { gross >= 0 }
ensures { result >= 0 }
{
  gross - calculateTax(gross, bracket)
}
```

**Solution**: Two complementary strategies:

#### Strategy 1: `define-fun` for verified callees

If the callee was already verified, emit its body as a `define-fun` in the SMT-LIB context:

```smt2
; calculateTax already verified — inline its definition
(define-fun calculateTax ((income Int) (bracket TaxBracket)) Int
  (match bracket
    ((EXEMPT 0)
     (REDUCED (div income 10))
     (STANDARD (div income 5)))))

; Now netIncome can reference calculateTax
(define-fun netIncome_body ((gross Int) (bracket TaxBracket)) Int
  (- gross (calculateTax gross bracket)))

(assert (>= gross 0))
(assert (not (>= (netIncome_body gross bracket) 0)))
(check-sat)
```

#### Strategy 2: Contract-based abstraction for unverified callees

If the callee has contracts but wasn't verified (e.g., outside fragment), use its contracts as axioms:

```smt2
; calculateTax not verified, but has contracts — use as uninterpreted with axioms
(declare-fun calculateTax (Int TaxBracket) Int)
(assert (forall ((income Int) (bracket TaxBracket))
  (=> (>= income 0)
      (and (>= (calculateTax income bracket) 0)
           (<= (calculateTax income bracket) income)))))
```

#### Implementation

**Files to modify:**
- `internal/smt/codegen.go` — Add `encodeFunctionCall()`, resolve callee by name (+100 LOC)
- `internal/smt/codegen.go` — Add function body cache for `define-fun` emission (+50 LOC)
- `internal/smt/encodable.go` — Remove "cross-function call" from rejection list (+10 LOC)

**Files to create:**
- `internal/smt/callee_resolver.go` — Resolve and encode callee functions (~150 LOC)
- `internal/smt/callee_resolver_test.go` — Tests (~200 LOC)

**Acceptance criteria:**
- [ ] `netIncome` calling `calculateTax` verifies when both are in same module
- [ ] Clear error when callee is not in verifiable fragment AND has no contracts
- [ ] Circular calls detected and rejected with helpful message
- [ ] `access_control.ail` `canAccess` calling `accessLevel` verifies

---

### Phase B: String Verification (~10h)

**Z3 String Theory** supports: `str.++` (concat), `str.len` (length), `str.contains`, `str.prefixof`, `str.suffixof`, `str.at`, `str.substr`, `str.indexof`, `str.replace`, and regex matching.

**AILANG string builtins to encode:**

| AILANG Builtin | SMT-LIB | Notes |
|---------------|---------|-------|
| `concat_String(a, b)` | `(str.++ a b)` | String concatenation |
| `length_String(s)` | `(str.len s)` | String length |
| `eq_String(a, b)` | `(= a b)` | String equality |
| `lt_String(a, b)` | `(str.< a b)` | Lexicographic comparison |

**Example contract:**
```ailang
export func formatCode(prefix: string, num: int) -> string ! {}
requires { num >= 0 }
ensures { length(result) >= length(prefix) }
{
  concat(prefix, intToString(num))
}
```

**Implementation:**
- `internal/smt/types.go` — Map `string` → `String` SMT sort (+10 LOC)
- `internal/smt/codegen.go` — Encode string builtins to `str.*` operations (+80 LOC)
- `internal/smt/encodable.go` — Remove string from rejection list (+5 LOC)
- Tests for string encoding (~150 LOC)

**Limitations:**
- `intToString` / `stringToInt` conversions are complex in SMT — may need uninterpreted function abstraction
- Regex matching could be supported but adds complexity — defer to follow-up

---

### Phase C: Record Verification (~8h)

**SMT-LIB datatypes** naturally model records with named fields and accessor functions.

**AILANG record → SMT-LIB:**
```ailang
type Point = { x: int, y: int }

export func manhattan(p: Point) -> int ! {}
ensures { result >= 0 }
{
  abs(p.x) + abs(p.y)
}
```

```smt2
(declare-datatype Point ((mk-Point (x Int) (y Int))))

(declare-const p Point)
(define-fun manhattan_body ((p Point)) Int
  (+ (ite (>= (x p) 0) (x p) (- (x p)))
     (ite (>= (y p) 0) (y p) (- (y p)))))

(assert (not (>= (manhattan_body p) 0)))
(check-sat)
```

**Implementation:**
- `internal/smt/types.go` — Encode record types as `declare-datatype` with accessors (+60 LOC)
- `internal/smt/codegen.go` — Encode `RecordAccess` as SMT accessor call (+40 LOC)
- `internal/smt/codegen.go` — Encode `Record` construction as constructor call (+30 LOC)
- `internal/smt/encodable.go` — Remove record from rejection list (+5 LOC)
- Tests for record encoding (~150 LOC)

**Record updates** (`{ p with x = 5 }`) encode as constructor calls: `(mk-Point 5 (y p))`.

---

### Phase D: List Verification (~12h)

**Z3 Sequence Theory** (`seq.*`) supports: `seq.++` (concat), `seq.len` (length), `seq.nth` (index), `seq.extract` (subsequence), `seq.contains`, `seq.unit` (singleton).

This is the most complex expansion because AILANG lists are polymorphic and AILANG has list combinators (`map`, `filter`, etc.) that are higher-order.

**What can be verified:**
- List length properties
- Element access by index
- Concatenation properties
- Simple list construction and deconstruction

**What cannot be verified (fundamental):**
- `map(f, xs)` — higher-order (f is a parameter)
- `filter(pred, xs)` — higher-order
- `foldl(f, init, xs)` — higher-order + recursive

**AILANG → SMT-LIB mapping:**

| AILANG | SMT-LIB | Notes |
|--------|---------|-------|
| `[int]` | `(Seq Int)` | Typed sequence |
| `length(xs)` | `(seq.len xs)` | List length |
| `head(xs)` | `(seq.nth xs 0)` | First element |
| `xs ++ ys` | `(seq.++ xs ys)` | Concatenation |
| `[1, 2, 3]` | `(seq.++ (seq.unit 1) (seq.unit 2) (seq.unit 3))` | List literal |

**Example:**
```ailang
export func safeConcat(xs: [int], ys: [int]) -> [int] ! {}
ensures { length(result) == length(xs) + length(ys) }
{
  xs ++ ys
}
```

**Implementation:**
- `internal/smt/types.go` — Map `[T]` → `(Seq T)` (+30 LOC)
- `internal/smt/codegen.go` — Encode list builtins to `seq.*` operations (+120 LOC)
- `internal/smt/codegen.go` — Encode list literals as `seq.unit` chains (+40 LOC)
- `internal/smt/encodable.go` — Remove list from rejection list, keep higher-order rejection (+15 LOC)
- Tests (~200 LOC)

---

### Phase E: Bounded Recursion (~15h)

**High uncertainty. Ship as experimental (`--verify-recursive-depth N`).**

Recursive functions can be verified by **unrolling** the recursion to a fixed depth and checking contracts for all inputs that terminate within that depth.

**Example:**
```ailang
export func factorial(n: int) -> int ! {}
requires { n >= 0 }
ensures { result >= 1 }
{
  if n <= 1 then 1
  else n * factorial(n - 1)
}
```

**Unrolled to depth 3:**
```smt2
(define-fun factorial_0 ((n Int)) Int 1)  ; base case only
(define-fun factorial_1 ((n Int)) Int
  (ite (<= n 1) 1 (* n (factorial_0 (- n 1)))))
(define-fun factorial_2 ((n Int)) Int
  (ite (<= n 1) 1 (* n (factorial_1 (- n 1)))))
(define-fun factorial_3 ((n Int)) Int
  (ite (<= n 1) 1 (* n (factorial_2 (- n 1)))))
```

**Soundness caveat**: Bounded unrolling can only prove the contract for inputs that terminate within the depth bound. It does NOT prove the contract for all inputs. Output must clearly state this:

```
  ✓ VERIFIED factorial (bounded: depth 5)
    Note: Verified for inputs terminating within 5 recursive calls.
          Not a full proof for all inputs.
```

**Implementation:**
- `internal/smt/unroll.go` — Recursive function unrolling (~200 LOC)
- `internal/smt/unroll_test.go` — Tests (~200 LOC)
- `internal/smt/encodable.go` — Allow recursive functions under `--verify-recursive-depth` flag (+20 LOC)
- CLI flag: `--verify-recursive-depth N` (default: off)

**Key design decisions:**
1. **Default off** — user must opt in with `--verify-recursive-depth N`
2. **Clear output labeling** — "VERIFIED (bounded: depth N)"
3. **Depth limit** — maximum depth 10 (prevent solver timeout explosion)
4. **Mutual recursion** — not supported (too complex for bounded unrolling)

---

## Implementation Plan

### Sprint 1: Cross-Function Calls (Phase A) — ~8h

**Highest impact, lowest risk.** Removes the most common user complaint.

- [ ] Implement callee resolution in SMT encoder
- [ ] Add `define-fun` emission for verified callees
- [ ] Add contract-based axioms for unverified callees
- [ ] Detect circular call graphs
- [ ] Update `access_control.ail` and `finance.ail` to use cross-function calls
- [ ] Tests for cross-function verification

### Sprint 2: Records (Phase C) — ~8h

**Second highest impact.** Records are pervasive in AILANG business logic.

- [ ] Encode record types as SMT-LIB datatypes with accessors
- [ ] Encode record construction, access, and update
- [ ] Remove record rejection from fragment checker
- [ ] Create `examples/runnable/contracts/record_verify.ail`
- [ ] Tests for record verification

### Sprint 3: Strings (Phase B) — ~10h

- [ ] Map `string` to SMT `String` sort
- [ ] Encode string builtins to `str.*` operations
- [ ] Handle `intToString`/`stringToInt` as uninterpreted functions
- [ ] Remove string rejection from fragment checker
- [ ] Create `examples/runnable/contracts/string_verify.ail`
- [ ] Tests for string verification

### Sprint 4: Lists (Phase D) — ~12h

- [ ] Map `[T]` to `(Seq T)` sort
- [ ] Encode list construction, concatenation, length, indexing
- [ ] Keep higher-order combinators rejected (fundamental)
- [ ] Remove list rejection from fragment checker (partial)
- [ ] Create `examples/runnable/contracts/list_verify.ail`
- [ ] Tests for list verification

### Sprint 5: Bounded Recursion (Phase E) — ~15h

**Highest risk. Ship as experimental.**

- [ ] Implement recursive function unrolling
- [ ] Add `--verify-recursive-depth N` CLI flag
- [ ] Add bounded verification labeling in output
- [ ] Depth limit enforcement (max 10)
- [ ] Create `examples/runnable/contracts/recursive_verify.ail`
- [ ] Tests for bounded recursion

---

## Files to Create/Modify

**New files (~770 LOC implementation + ~900 LOC tests):**

```
internal/smt/
├── callee_resolver.go       -- Cross-function call resolution (~150 LOC)
├── callee_resolver_test.go  -- Tests (~200 LOC)
├── unroll.go                -- Recursive function unrolling (~200 LOC)
└── unroll_test.go           -- Tests (~200 LOC)

examples/runnable/contracts/
├── record_verify.ail        -- Record verification demo
├── string_verify.ail        -- String verification demo
├── list_verify.ail          -- List verification demo
└── recursive_verify.ail     -- Bounded recursion demo
```

**Modified files (~420 LOC changes):**

| File | Change | LOC |
|------|--------|-----|
| `internal/smt/types.go` | Add string, list, record type mappings | +100 |
| `internal/smt/codegen.go` | String/list/record expression encoding, cross-function calls | +200 |
| `internal/smt/encodable.go` | Remove rejections for expanded types, add bounded recursion | +50 |
| `internal/smt/codegen_test.go` | Tests for all new encodings | +300 |
| `cmd/ailang/verify.go` | Add `--verify-recursive-depth` flag | +20 |
| `examples/manifest.json` | Add new example entries | +50 |
| `docs/docs/guides/contracts.mdx` | Update decidable fragment table | +30 |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Z3 string theory performance | Medium | Medium | Timeout per function; skip slow functions |
| Z3 sequence theory undecidable edge cases | Medium | High | Restrict to quantifier-free sequence operations |
| Cross-function call graph explosion | Low | Medium | Limit call depth; detect cycles |
| Bounded recursion soundness confusion | High | High | Clear labeling: "bounded: depth N", not "VERIFIED" |
| Record types with nested records | Low | Low | Reject nested records initially; expand in follow-up |

---

## Success Criteria

- [ ] All existing `ailang verify` tests continue to pass (no regressions)
- [ ] Cross-function calls resolve for functions in same module
- [ ] String contracts verify using Z3 string theory
- [ ] Record contracts verify using SMT-LIB datatypes
- [ ] List contracts verify using Z3 sequence theory (non-higher-order)
- [ ] Bounded recursion verifies with clear depth labeling
- [ ] Documentation updated with expanded fragment table
- [ ] 4 new example files demonstrating expanded capabilities
- [ ] Total verifiable fragment coverage: ~70-80% of typical functions

---

## Testing Strategy

**Unit tests** (~900 LOC):
- Cross-function call resolution (define-fun, contract axioms, circular detection)
- String type mapping and builtin encoding
- Record type mapping, construction, access, update encoding
- List type mapping, construction, concatenation encoding
- Recursive function unrolling (depth 1, 3, 5)

**Integration tests** (Z3-gated):
- `finance.ail` with cross-function `netIncome` calling `calculateTax`
- String verification with concat + length contracts
- Record verification with field access contracts
- List verification with length + concatenation contracts
- Bounded recursive factorial verification

**Property tests:**
- Type mapping round-trips for all new types
- Unrolling depth N produces exactly N+1 `define-fun` declarations

---

## Related Documents

- [m-verify-smt-verification.md](m-verify-smt-verification.md) — Parent design doc (full verification roadmap)
- [m-smt-backend-mvp-sprint-plan.md](m-smt-backend-mvp-sprint-plan.md) — Phase 1 sprint (✅ complete)
- [../../implemented/v0_7_1/m-verify-contracts.md](../../implemented/v0_7_1/m-verify-contracts.md) — Runtime contracts
- [../../implemented/v0_6_1/m-verify-runtime-contracts.md](../../implemented/v0_6_1/m-verify-runtime-contracts.md) — Phase 0+0.5
- [m-contracts-oplowering.md](m-contracts-oplowering.md) — OpLowering prerequisite (✅ complete)
- `internal/smt/` — Current implementation (3,420 LOC across 8 files)

---

**Document created**: 2026-02-12
**Author**: Claude Code
