# M-SMT-FRAGMENT-EXPANSION-V2: Z3 Verification Phase 2

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (Medium)
**Estimated**: 3-4 weeks (6 phases, each independently shippable)
**Dependencies**:
- M-SMT-FRAGMENT-EXPANSION (v0.8.0) ✅ — Phases A-E complete (strings, lists, records, cross-function calls, bounded recursion)
- M-SMT-BOUNDED-RECURSION (v0.8.0) ✅ — Dafny-style unrolling at configurable depth
- M-SMT-RECORD-DISCOVERY (v0.8.0) ✅ — Record type discovery from bodies/return types

**Parent Design Doc**: [m-smt-fragment-expansion.md](../implemented/v0_8_0/m-smt-fragment-expansion.md)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | All new encodings are deterministic — same input produces identical SMT-LIB output |
| A2: Replayability | +1 | Expanded proofs remain serializable (SMT-LIB text); bounded quantifiers are reproducible |
| A3: Effect Legibility | 0 | No change — effects remain excluded from SMT verification |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +2 | **Primary goal** — more functions locally provable; bounded quantifiers enable element-wise properties |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | Significantly more machine-checkable proofs; HOF inlining is a machine transformation |
| A8: Minimal Syntax | +1 | Only one small syntax addition (per-function `@verify` attribute); bounded quantifiers reuse existing `forall` in contracts |
| A9: Cost Visibility | +1 | Per-function depth control makes solver cost explicit |
| A10: Composability | +1 | HOF inlining + bounded recursion compose with existing cross-function calls |
| A11: Structured Failure | +1 | Counterexamples now cover more function shapes; quantifier violations show specific index |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +10** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): All expansions stay within decidable SMT theories
- [x] A3 (Effects): Effects remain excluded (fundamental limitation)
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Every expansion improves machine analysis

---

## Problem Statement

Phase 1 of SMT fragment expansion (v0.8.0) brought coverage from ~40% to ~70-80% of typical AILANG functions. The remaining gaps cluster around six categories:

| Gap | Current Status | Impact |
|-----|---------------|--------|
| `map(fn, xs)` with known lambda | SKIPPED (HIGHER_ORDER) | Common list processing pattern rejected even when lambda is statically known |
| `reverse(xs)`, `take(n, xs)` | SKIPPED (RECURSIVE) | Recursive list ops could use existing bounded unrolling but don't |
| `ensures { forall i: 0..length(xs) => xs[i] >= 0 }` | Not expressible | Cannot state element-wise properties — the highest-value contract pattern for lists |
| `{ inner: { x: int, y: int } }` | SKIPPED (nested record) | Nested records rejected even though Z3 handles them natively |
| `match p { {x, y} => x + y }` | SKIPPED (unsupported pattern) | `RecordPattern` exists in Core AST but SMT encoder doesn't handle it |
| Global `--verify-recursive-depth N` | Works but coarse | Can't set depth 5 for `fibonacci` and depth 2 for `sumTo` — one global setting |

**Current State:**
- ~70-80% of typical functions are verifiable (Phase 1 result)
- Higher-order list processing is the #1 remaining rejection reason
- Bounded quantifiers over list elements are the most-requested missing contract feature
- Nested records are common in real-world business logic

**Coverage estimate after Phase 2**: ~85-90% of typical AILANG functions

---

## Goals

**Primary Goal**: Expand SMT-verifiable coverage to ~85-90% of typical AILANG functions by handling monomorphic higher-order calls, recursive list operations, bounded quantifiers, nested records, record patterns, and per-function recursion depth.

**Success Metrics:**
- [ ] `map(\x -> x + 1, [1, 2, 3])` with known lambda verifies (HOF inlining)
- [ ] `reverse(xs)` verifies via bounded unrolling at depth N
- [ ] `ensures { forall i: 0..length(result) => result[i] >= 0 }` verifies with bounded quantifiers
- [ ] Nested record `{inner: {x: int, y: int}}` field access verifies
- [ ] Record pattern `match p { {x, y} => ... }` encodes correctly
- [ ] `@verify(depth: 5)` overrides global `--verify-recursive-depth`
- [ ] All existing `ailang verify` tests continue to pass
- [ ] 6 new example files demonstrating expanded capabilities

---

## Solution Design

### Overview: Six Independent Phases

Each phase is independently shippable and testable:

```
Phase F: Monomorphic Higher-Order Inlining (map/filter/fold with known lambdas)
    │
Phase G: Recursive List Operations (reverse/take/drop via bounded unrolling)
    │
Phase H: Bounded Quantifiers (forall i: range => property)
    │
Phase I: Nested Record Types (recursive declare-datatype)
    │
Phase J: Record Patterns in Match (encode RecordPattern)
    │
Phase K: Per-Function Recursion Depth (@verify attribute)
```

Phases F and G are the highest impact. Phase H is the highest value for contract expressiveness. Phases I-K are smaller but close annoying gaps.

---

### Phase F: Monomorphic Higher-Order Inlining (~16h)

**The key insight**: `map`, `filter`, and `fold` are rejected because `hasHigherOrder()` finds lambdas in argument position. But when the lambda is a **statically known** literal (not a function parameter), we can inline it into a recursive definition and then apply bounded unrolling.

**Transformation pipeline:**

```
map(\x -> x + 1, xs)
    ↓ [Lambda inlining]
map_inline(xs) where body: match xs { [] => [], x :: rest => (x + 1) :: map_inline(rest) }
    ↓ [Bounded unrolling — existing Phase E]
map_inline_0 (uninterpreted)
map_inline_1 (1 level deep)
map_inline_2 (2 levels deep)
```

**AILANG example:**
```ailang
export func incrementAll(xs: [int]) -> [int] ! {}
requires { _list_length(xs) <= 5 }
ensures { _list_length(result) == _list_length(xs) }
{
  map(\x -> x + 1, xs)
}
```

**What this enables:**
- `map(lambda, xs)` where lambda is a literal → inline + unroll
- `filter(lambda, xs)` where lambda is a literal → inline + unroll
- `foldl(lambda, init, xs)` where lambda is a literal → inline + unroll

**What this does NOT enable:**
- `map(f, xs)` where `f` is a function parameter → still HIGHER_ORDER (fundamental)

**Implementation approach:**

1. **Detect inlinable HOF calls**: In `encodable.go`, distinguish between:
   - `App(map, [Lambda(...), xs])` → inlinable (lambda is literal)
   - `App(map, [Var(f), xs])` → not inlinable (f is parameter)

2. **Lambda specialization**: Before encoding, rewrite `map(\x -> body, xs)` into a new recursive function `map_specialized(xs)` that pattern-matches on list structure

3. **Feed to existing unroller**: The specialized function is self-recursive, so `UnrollRecursiveFunction` handles it directly

**Files to modify:**
- `internal/smt/encodable.go` — Refine `hasHigherOrder` to allow known-lambda HOF calls (+40 LOC)
- `internal/smt/codegen.go` — Detect and specialize HOF calls before encoding (+30 LOC)

**Files to create:**
- `internal/smt/hof_inline.go` — Lambda specialization for map/filter/fold (~250 LOC)
- `internal/smt/hof_inline_test.go` — Tests (~300 LOC)

**Supported HOF patterns:**

| Pattern | Specialization | Z3 Encoding |
|---------|---------------|-------------|
| `map(\x -> body, xs)` | `map_spec(xs) = match xs { [] => [], h :: t => body[x→h] :: map_spec(t) }` | Bounded unrolling |
| `filter(\x -> pred, xs)` | `filter_spec(xs) = match xs { [] => [], h :: t => if pred[x→h] then h :: filter_spec(t) else filter_spec(t) }` | Bounded unrolling |
| `foldl(\acc x -> body, init, xs)` | `fold_spec(acc, xs) = match xs { [] => acc, h :: t => fold_spec(body[acc→acc, x→h], t) }` | Bounded unrolling |

**Soundness**: Same as Phase E bounded recursion — verified for inputs terminating within depth N. Output labels "VERIFIED (bounded: depth N, inlined: map)".

---

### Phase G: Recursive List Operations (~8h)

**Problem**: `reverse`, `take`, `drop` are recursive AILANG functions in `std/list`. They're rejected by `isRecursive()` and `hasUnencodableTypes()` (std/list without SMT mapping). But we already have bounded recursion unrolling.

**Solution**: Register these as **builtin recursive patterns** that can be inlined as `define-fun` chains using the existing unroller, rather than relying on their stdlib implementation.

**Approach**: For each recursive list operation, provide a Core AST template that uses only SMT-encodable primitives (list cons, seq.nth, seq.len, etc.), then unroll it.

**Builtin templates:**

```smt2
; reverse: unroll to depth N
(define-fun reverse_0 ((xs (Seq Int))) (Seq Int)
  (as seq.empty (Seq Int)))  ; base: uninterpreted
(define-fun reverse_1 ((xs (Seq Int))) (Seq Int)
  (ite (= (seq.len xs) 0)
    (as seq.empty (Seq Int))
    (seq.++ (reverse_0 (seq.extract xs 1 (- (seq.len xs) 1)))
            (seq.unit (seq.nth xs 0)))))
```

```smt2
; take: take first n elements
(define-fun take_0 ((n Int) (xs (Seq Int))) (Seq Int)
  (as seq.empty (Seq Int)))
(define-fun take_1 ((n Int) (xs (Seq Int))) (Seq Int)
  (ite (or (<= n 0) (= (seq.len xs) 0))
    (as seq.empty (Seq Int))
    (seq.++ (seq.unit (seq.nth xs 0))
            (take_0 (- n 1) (seq.extract xs 1 (- (seq.len xs) 1))))))
```

**New builtins to register:**

| New Builtin | Type | Recursive? | Z3 Strategy |
|-------------|------|-----------|-------------|
| `_list_reverse` | `[a] -> [a]` | Yes | Bounded unrolling of `seq.extract` + `seq.++` |
| `_list_take` | `(int, [a]) -> [a]` | Yes | Bounded unrolling with `seq.extract` |
| `_list_drop` | `(int, [a]) -> [a]` | Yes | Bounded unrolling with `seq.extract` |
| `_list_contains` | `([a], a) -> bool` | No | `seq.contains (seq.unit elem)` |
| `_list_extract` | `([a], int, int) -> [a]` | No | `seq.extract` |

**Non-recursive additions** (`_list_contains`, `_list_extract`) map directly to Z3 sequence operations — no unrolling needed.

**Files to modify:**
- `internal/smt/types.go` — Add new list builtins to `ListBuiltinSpecial` map (+30 LOC)
- `internal/smt/codegen_apps.go` — Handle recursive list builtins via unrolling (+40 LOC)
- `internal/smt/encodable.go` — Accept new builtins in fragment checker (+10 LOC)
- `internal/builtins/list.go` — Register runtime builtins (+60 LOC)

**Files to create:**
- `internal/smt/list_unroll.go` — Recursive list operation templates for unrolling (~200 LOC)
- `internal/smt/list_unroll_test.go` — Tests (~200 LOC)

---

### Phase H: Bounded Quantifiers (~15h)

**The highest-value missing feature for list contracts.** Users want to express:

```ailang
export func clampAll(xs: [int], lo: int, hi: int) -> [int] ! {}
requires { lo <= hi }
ensures { forall i: 0..(_list_length(result)) => _list_nth(result, i) >= lo }
ensures { forall i: 0..(_list_length(result)) => _list_nth(result, i) <= hi }
{
  map(\x -> max(lo, min(hi, x)), xs)
}
```

**Z3 bounded quantifier encoding:**

```smt2
; forall i: 0..N => P(i)  →  (forall ((i Int)) (=> (and (>= i 0) (< i N)) P(i)))
(assert (not
  (forall ((i Int))
    (=> (and (>= i 0) (< i (seq.len result)))
        (>= (seq.nth result i) lo)))))
```

Z3's quantifier reasoning is **decidable** for bounded integer quantifiers with array/sequence theory (this is the quantified fragment of the theory of sequences). The key constraint: the quantifier bound must be a **concrete expression** over function parameters and `seq.len` — no uninterpreted functions in bounds.

**AILANG syntax**: Reuse existing `forall` from the property/contract AST. The `Binders` field already exists in `ast.Property` (currently nil for requires/ensures).

```ailang
-- Already parseable (ast.Property has Binders field):
ensures { forall i: 0..length(result) => property(i) }
```

**Parser changes**: Extend contract expression parsing to handle `forall IDENT: EXPR..EXPR => EXPR`.

**SMT encoding:**
- `forall i: lo..hi => P(i)` → `(forall ((i Int)) (=> (and (>= i lo) (< i hi)) P_encoded))`
- The `=> P(i)` body can reference `_list_nth(result, i)`, which encodes to `(seq.nth result i)`
- Negation wrapping: `(assert (not (forall ...)))` — Z3 checks if there exists a counterexample

**Decidability boundary:**
- Bounded `forall` over integers with sequence indexing: **decidable** (QF_LIA + Seq)
- Nested `forall`: **decidable** but may cause solver timeout — limit to depth 1
- `forall` over non-integer types: **not supported** (reject)
- `exists`: **not supported** (defer to future work)

**Implementation:**

1. **Parser**: Extend `parser_contracts.go` to parse `forall IDENT: EXPR..EXPR => EXPR` (+80 LOC)
2. **AST**: Add `ForallExpr` to `ast_expr.go` with binder, lower bound, upper bound, body (+30 LOC)
3. **Core lowering**: Lower `ForallExpr` to `core.Forall{Var, Lo, Hi, Body}` (+40 LOC)
4. **SMT encoding**: Encode `core.Forall` to bounded `(forall ...)` with range guard (+60 LOC)
5. **Fragment checker**: Allow `core.Forall` nodes, validate bounds are encodable (+20 LOC)

**Files to modify:**
- `internal/parser/parser_contracts.go` — Parse `forall i: lo..hi => body` (+80 LOC)
- `internal/ast/ast_expr.go` — Add `ForallExpr` node (+30 LOC)
- `internal/elaborate/file.go` — Lower `ForallExpr` to Core (+40 LOC)
- `internal/core/core.go` — Add `Forall` Core node (+30 LOC)
- `internal/smt/codegen_control.go` — Encode `core.Forall` to SMT-LIB bounded quantifier (+60 LOC)
- `internal/smt/encodable.go` — Accept `core.Forall`, validate bounds (+20 LOC)

**Files to create:**
- `internal/smt/codegen_quantifier.go` — Quantifier encoding logic (~100 LOC)
- `internal/smt/codegen_quantifier_test.go` — Tests (~200 LOC)

---

### Phase I: Nested Record Types (~6h)

**Problem**: Records with record-typed fields are rejected. Z3 handles nested datatypes natively.

```ailang
type Inner = { x: int, y: int }
type Outer = { name: string, pos: Inner }

export func getX(o: Outer) -> int ! {}
ensures { result == o.pos.x }
{
  o.pos.x
}
```

**SMT encoding:**
```smt2
(declare-datatype Inner ((mk_Inner (x Int) (y Int))))
(declare-datatype Outer ((mk_Outer (name String) (pos Inner))))

(declare-const o Outer)
(define-const result Int (x (pos o)))
(assert (not (= result (x (pos o)))))
(check-sat)
```

**The fix is straightforward**: `collectAndDeclareRecordTypes` must recursively discover record-typed fields and emit declarations in dependency order.

**Implementation:**
- `internal/smt/codegen_records.go` — Recursive record type collection (+60 LOC)
- `internal/smt/types.go` — Topological sort of record type declarations (+40 LOC)
- `internal/smt/codegen_records_test.go` — Tests for nested records (+100 LOC)

**Key constraint**: No recursive record types (a record that contains itself). These are rejected at the type level. Only finite nesting depth is supported.

---

### Phase J: Record Patterns in Match (~5h)

**Problem**: `core.RecordPattern` exists in the Core AST (defined at [core.go:371](internal/core/core.go#L371)) but `encodePattern` in [codegen_control.go:49](internal/smt/codegen_control.go#L49) doesn't handle it — falls through to `"unsupported pattern type"`.

```ailang
export func getDistance(p: Point) -> int ! {}
ensures { result >= 0 }
{
  match p {
    { x, y } => abs(x) + abs(y)
  }
}
```

**SMT encoding:**
```smt2
; Record pattern { x, y } destructures to accessor calls
(match p ((mk_Point x y) (+ (ite (>= x 0) x (- x)) (ite (>= y 0) y (- y)))))
```

Record patterns `{ x, y }` desugar to constructor patterns `(mk_RecordName field1 field2)` where field names become bound variables in alphabetical order (matching the deterministic field ordering used in `DeclareRecordDatatype`).

**Implementation:**
- `internal/smt/codegen_control.go` — Add `*core.RecordPattern` case to `encodePattern` (+30 LOC)
- `internal/smt/codegen_control.go` — Map field names to constructor arg positions (+20 LOC)
- `internal/smt/encodable.go` — Remove `RecordPattern` from `hasDeepPatterns` if applicable (+5 LOC)
- Tests (+80 LOC)

---

### Phase K: Per-Function Recursion Depth (~8h)

**Problem**: `--verify-recursive-depth N` is a global setting. Users may want depth 5 for a complex recursive function but depth 2 (faster) for simpler ones.

**Solution**: A `@verify` attribute on function declarations.

```ailang
@verify(depth: 5)
export func fibonacci(n: int) -> int ! {}
requires { n >= 0 }
ensures { result >= 0 }
{
  if n <= 1 then n
  else fibonacci(n - 1) + fibonacci(n - 2)
}

@verify(depth: 2)
export func sumTo(n: int) -> int ! {}
requires { n >= 0 }
ensures { result >= 0 }
{
  if n <= 0 then 0
  else n + sumTo(n - 1)
}
```

**Resolution order:**
1. `@verify(depth: N)` on the function → use N
2. `--verify-recursive-depth N` CLI flag → use N
3. Default: no unrolling (recursive functions rejected)

**Implementation:**
1. **Parser**: Parse `@verify(key: value)` attribute before function declarations (+60 LOC)
2. **AST**: Add `Attributes map[string]any` to `FuncDecl` (+10 LOC)
3. **DeclMeta**: Carry `VerifyDepth int` through elaboration (+10 LOC)
4. **CLI**: Read per-function depth in `cmd/ailang/verify.go`, override global (+20 LOC)

**Files to modify:**
- `internal/parser/parser.go` — Parse `@verify(...)` attribute (+60 LOC)
- `internal/ast/ast_decl.go` — Add `Attributes` to `FuncDecl` (+10 LOC)
- `internal/core/core.go` — Add `VerifyDepth` to `DeclMeta` (+10 LOC)
- `internal/elaborate/file.go` — Carry attribute to `DeclMeta` (+15 LOC)
- `cmd/ailang/verify.go` — Per-function depth override (+20 LOC)

---

## Examples

### Example 1: Map with Known Lambda (Phase F)

**Before:**
```
incrementAll:
  ⚠ Skipped for SMT verification
  Reasons:
    • HIGHER_ORDER: Function body contains lambda at main.ail:5:7
```

**After:**
```
incrementAll:
  ensures:  _list_length(result) == _list_length(xs)    [VERIFIED (bounded: depth 2, inlined: map)] (0.12s)
```

### Example 2: Bounded Quantifier (Phase H)

**Before:** Not expressible — no way to assert element-wise properties.

**After:**
```ailang
export func clampAll(xs: [int], lo: int, hi: int) -> [int] ! {}
requires { lo <= hi }
ensures { forall i: 0.._list_length(result) => _list_nth(result, i) >= lo }
ensures { forall i: 0.._list_length(result) => _list_nth(result, i) <= hi }
{
  map(\x -> max(lo, min(hi, x)), xs)
}
```

```
clampAll:
  requires: lo <= hi                                                  [precondition]
  ensures:  forall i: 0..length(result) => result[i] >= lo           [VERIFIED (bounded: depth 3)] (0.25s)
  ensures:  forall i: 0..length(result) => result[i] <= hi           [VERIFIED (bounded: depth 3)] (0.22s)
```

### Example 3: Nested Records (Phase I)

**Before:**
```
getX:
  ⚠ Skipped for SMT verification
  Reasons:
    • UNENCODABLE_TYPE: Function "getX" uses types not encodable in SMT
```

**After:**
```
getX:
  ensures:  result == o.pos.x    [VERIFIED ✓] (0.01s)
```

### Example 4: Per-Function Depth (Phase K)

**Before:** `--verify-recursive-depth 5` applies to ALL functions, making simple ones slow.

**After:**
```ailang
@verify(depth: 5)
export func fibonacci(n: int) -> int ! {} ...

@verify(depth: 2)
export func sumTo(n: int) -> int ! {} ...
```

```
fibonacci:
  ensures:  result >= 0    [VERIFIED (bounded: depth 5)] (0.45s)
sumTo:
  ensures:  result >= 0    [VERIFIED (bounded: depth 2)] (0.03s)
```

---

## Success Criteria

- [ ] `map(\x -> x + 1, xs)` verifies when lambda is a literal
- [ ] `filter(\x -> x > 0, xs)` verifies when lambda is a literal
- [ ] `foldl(\acc x -> acc + x, 0, xs)` verifies when lambda is a literal
- [ ] `map(f, xs)` where `f` is a parameter still REJECTED (correct behavior)
- [ ] `reverse(xs)` verifies via bounded unrolling
- [ ] `take(n, xs)` and `drop(n, xs)` verify via bounded unrolling
- [ ] `forall i: 0..length(result) => result[i] >= 0` verifies
- [ ] Nested record field access `o.inner.x` encodes correctly
- [ ] Record patterns in match encode to constructor patterns
- [ ] `@verify(depth: N)` overrides global `--verify-recursive-depth`
- [ ] All existing `ailang verify` tests continue to pass (no regressions)
- [ ] 6 new example files in `examples/runnable/contracts/`
- [ ] Documentation updated with expanded fragment table
- [ ] Total verifiable fragment coverage: ~85-90% of typical functions

---

## Testing Strategy

**Unit tests (~1,200 LOC):**
- HOF inlining: lambda specialization for map/filter/fold, non-lambda rejection
- Recursive list templates: reverse/take/drop unrolling at depths 1-5
- Bounded quantifiers: range encoding, nested quantifier rejection, non-integer rejection
- Nested records: recursive declaration emission, topological sort of dependencies
- Record patterns: field-to-constructor mapping, alphabetical ordering
- Per-function depth: attribute parsing, override resolution

**Integration tests (Z3-gated):**
- `hof_verify.ail` — map/filter with known lambdas + contracts
- `list_recursive_verify.ail` — reverse/take/drop with length/element contracts
- `quantifier_verify.ail` — forall over list indices with element properties
- `nested_record_verify.ail` — nested record construction + field access
- `record_pattern_verify.ail` — record destructuring in match
- `per_function_depth_verify.ail` — mixed depths on different functions

**Property tests:**
- Lambda inlining produces a valid recursive Core AST
- Bounded quantifier encoding produces syntactically valid SMT-LIB
- Nested record declarations are topologically ordered (no forward references)

---

## Non-Goals

**Not in this feature:**
- `exists` quantifiers — requires dual encoding strategy (defer to Phase 3)
- Mutual recursion — too complex for bounded unrolling without call graph analysis (fundamental limitation)
- `map(f, xs)` where `f` is a function parameter — fundamental HOF limitation, not solvable via inlining
- Unbounded recursion (induction proofs) — fundamental limitation of SMT
- Effectful function verification — nondeterminism breaks proofs (fundamental)
- Array verification — AILANG arrays are mutable; mutability requires a different SMT strategy
- Regex matching for strings — complex SMT encoding, low priority

---

## Implementation Plan

### Sprint 1: HOF Inlining (Phase F) — ~16h

**Highest impact.** Unblocks the most common user complaint after Phase 1.

- [ ] Detect inlinable HOF calls (lambda literals in arg position)
- [ ] Implement lambda specialization for `map`
- [ ] Implement lambda specialization for `filter`
- [ ] Implement lambda specialization for `foldl`
- [ ] Integrate with existing bounded unrolling
- [ ] Update fragment checker to allow known-lambda HOF
- [ ] Create `examples/runnable/contracts/hof_verify.ail`
- [ ] Tests for HOF inlining

### Sprint 2: Recursive List Operations (Phase G) — ~8h

- [ ] Register `_list_reverse`, `_list_take`, `_list_drop`, `_list_contains`, `_list_extract` builtins
- [ ] Create recursive list operation templates
- [ ] Integrate with existing bounded unrolling
- [ ] Add runtime implementations for new builtins
- [ ] Create `examples/runnable/contracts/list_recursive_verify.ail`
- [ ] Tests for recursive list operations

### Sprint 3: Bounded Quantifiers (Phase H) — ~15h

**Highest value for contract expressiveness.**

- [ ] Parse `forall i: lo..hi => body` in contract expressions
- [ ] Add `ForallExpr` to AST
- [ ] Lower to `core.Forall` in elaboration
- [ ] Encode `core.Forall` to SMT-LIB bounded `(forall ...)`
- [ ] Update fragment checker to validate quantifier bounds
- [ ] Create `examples/runnable/contracts/quantifier_verify.ail`
- [ ] Tests for bounded quantifiers

### Sprint 4: Nested Records (Phase I) — ~6h

- [ ] Recursive record type discovery in `collectAndDeclareRecordTypes`
- [ ] Topological sort of record type declarations
- [ ] Encode nested field access chains
- [ ] Create `examples/runnable/contracts/nested_record_verify.ail`
- [ ] Tests for nested records

### Sprint 5: Record Patterns (Phase J) — ~5h

- [ ] Add `*core.RecordPattern` case to `encodePattern`
- [ ] Map field names to constructor argument positions (alphabetical)
- [ ] Update fragment checker if needed
- [ ] Create `examples/runnable/contracts/record_pattern_verify.ail`
- [ ] Tests for record patterns

### Sprint 6: Per-Function Depth (Phase K) — ~8h

- [ ] Parse `@verify(depth: N)` attribute before function declarations
- [ ] Add `Attributes` to `ast.FuncDecl`
- [ ] Carry `VerifyDepth` through elaboration to `DeclMeta`
- [ ] Per-function depth override in `cmd/ailang/verify.go`
- [ ] Create `examples/runnable/contracts/per_function_depth_verify.ail`
- [ ] Tests for per-function depth

---

## Files to Create/Modify

**New files (~1,050 LOC implementation + ~1,200 LOC tests):**

```
internal/smt/
├── hof_inline.go               -- Lambda specialization for map/filter/fold (~250 LOC)
├── hof_inline_test.go          -- Tests (~300 LOC)
├── list_unroll.go              -- Recursive list operation templates (~200 LOC)
├── list_unroll_test.go         -- Tests (~200 LOC)
├── codegen_quantifier.go       -- Bounded quantifier encoding (~100 LOC)
└── codegen_quantifier_test.go  -- Tests (~200 LOC)

examples/runnable/contracts/
├── hof_verify.ail              -- HOF inlining demo
├── list_recursive_verify.ail   -- Recursive list ops demo
├── quantifier_verify.ail       -- Bounded quantifier demo
├── nested_record_verify.ail    -- Nested records demo
├── record_pattern_verify.ail   -- Record patterns demo
└── per_function_depth_verify.ail -- Per-function depth demo
```

**Modified files (~500 LOC changes):**

| File | Change | LOC |
|------|--------|-----|
| `internal/smt/encodable.go` | Refine `hasHigherOrder` for known lambdas; accept new builtins and `Forall` | +75 |
| `internal/smt/codegen.go` | HOF detection before encoding | +30 |
| `internal/smt/codegen_control.go` | Record pattern encoding | +50 |
| `internal/smt/codegen_apps.go` | Recursive list builtin handling | +40 |
| `internal/smt/types.go` | New list builtins; nested record handling | +70 |
| `internal/smt/codegen_records.go` | Recursive record collection + topo sort | +60 |
| `internal/parser/parser_contracts.go` | Parse `forall i: lo..hi => body` | +80 |
| `internal/ast/ast_expr.go` | Add `ForallExpr` | +30 |
| `internal/ast/ast_decl.go` | Add `Attributes` to `FuncDecl` | +10 |
| `internal/core/core.go` | Add `Forall` node + `VerifyDepth` to `DeclMeta` | +40 |
| `internal/elaborate/file.go` | Lower `ForallExpr` + carry attributes | +55 |
| `internal/builtins/list.go` | Register new list builtins | +60 |
| `cmd/ailang/verify.go` | Per-function depth override | +20 |

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| HOF inlining produces invalid Core AST | Medium | High | Extensive property testing on specialization output; validate before encoding |
| Z3 bounded quantifier timeout on large ranges | High | Medium | Default to `requires { length(xs) <= N }` for bounded lists; timeout per function |
| Nested record declarations create forward references | Low | Medium | Topological sort with cycle detection; reject recursive records |
| `forall` syntax conflicts with existing parser rules | Medium | Medium | Use `forall` only in contract context (already reserved keyword) |
| Lambda specialization interacts badly with cross-function calls | Low | High | Test composition: `map(\x -> helper(x), xs)` where helper is verified callee |
| `@verify` attribute syntax sets precedent for attribute system | Low | Low | Keep minimal — only `depth` key for now; defer full attribute system to separate design |

---

## Related Documents

**Implemented (direct parent):**
- [m-smt-fragment-expansion.md](../implemented/v0_8_0/m-smt-fragment-expansion.md) — Phase 1 master design (A-E)
- [m-smt-bounded-recursion.md](../implemented/v0_8_0/m-smt-bounded-recursion.md) — Phase E bounded unrolling (reused by F, G)
- [m-smt-lists-sprint-plan.md](../implemented/v0_8_0/m-smt-lists-sprint-plan.md) — Phase D list basics
- [m-smt-records-sprint-plan.md](../implemented/v0_8_0/m-smt-records-sprint-plan.md) — Phase C records
- [m-smt-strings-sprint-plan.md](../implemented/v0_8_0/m-smt-strings-sprint-plan.md) — Phase B strings
- [m-smt-record-discovery.md](../implemented/v0_8_0/m-smt-record-discovery.md) — Record type discovery
- [m-verify-smt-verification.md](../implemented/v0_8_1/m-verify-smt-verification.md) — Parent verification roadmap

**Inspiration:**
- Dafny `{:fuel N}` attribute — per-function recursion depth
- F* quantifier encoding — bounded integer quantifiers with E-matching
- Liquid Haskell PLE — selective unfolding at call sites (our HOF inlining is a simpler version)

---

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Z3 Sequence Theory](https://microsoft.github.io/z3guide/docs/theories/Sequences/) - seq.* operations
- [Z3 Quantifiers](https://microsoft.github.io/z3guide/docs/logic/Quantifiers/) - bounded quantifier patterns
- [SMT-LIB Standard](https://smtlib.cs.uiowa.edu/) - SMT-LIB 2.6 reference
- [Dafny Fuel Attribute](https://dafny.org/dafny/DafnyRef/DafnyRef#sec-fuel) - per-function recursion depth inspiration

---

## Future Work (Phase 3+)

- **`exists` quantifiers** — dual encoding for existential properties
- **Invariant inference** — automatically infer loop/recursion invariants
- **Proof caching** — cache verified function contracts across compilation runs
- **Mutual recursion** — extend bounded unrolling to mutually recursive function groups
- **Counter-example guided refinement** — use Z3 counterexamples to suggest contract strengthening
- **SMT-LIB2 emission for external provers** — export proofs for CVC5, Alt-Ergo

---

**Document created**: 2026-03-12
**Last updated**: 2026-03-12
**Author**: Claude Code
