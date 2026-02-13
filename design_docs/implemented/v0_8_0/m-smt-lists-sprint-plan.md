# Sprint Plan: M-SMT-LISTS — List Verification (Phase D)

**Sprint ID**: M-SMT-LISTS
**Design Doc**: [m-smt-fragment-expansion.md](m-smt-fragment-expansion.md) (Phase D)
**Duration**: 1 day (~6 hours)
**Risk Level**: Medium (Z3 sequence theory is less commonly used than string theory)
**Estimated LOC**: ~600 (implementation + tests)

---

## Summary

Add list verification to `ailang verify`. Functions using list parameters, list
literals, concatenation, length, head, and nth operations will be encodable
in SMT-LIB using Z3's built-in sequence theory (`seq.*`).

**Before**: Any function touching lists is SKIPPED with "uses types not encodable in SMT"
**After**: Pure functions with list types can be VERIFIED or produce counterexamples

---

## Key Design Decisions

### New Builtins Required

The stdlib `std/list` functions (`length`, `head`, `nth`) are all **recursive AILANG functions** —
they cannot be encoded in SMT (recursion is rejected). We need Go-level builtins that map
directly to Z3 sequence operations:

| New Builtin | Type | Z3 Mapping | Purpose |
|-------------|------|-----------|---------|
| `_list_length` | `[a] -> int` | `(seq.len xs)` | List length |
| `_list_head` | `[a] -> a` | `(seq.nth xs 0)` | First element |
| `_list_nth` | `([a], int) -> a` | `(seq.nth xs idx)` | Element at index |

These follow the `_str_*` naming convention and can be used in contracts.

### Type Mapping

Both `TList{Element: T}` (deprecated) and `TApp{Constructor: TCon("list"), Args: [T]}`
(modern) must be handled:

- `[int]` → `(Seq Int)`
- `[string]` → `(Seq String)`
- `[bool]` → `(Seq Bool)`

### OpConcat Dispatch

After op_lowering, list `++` becomes `App(VarGlobal($builtin.concat_List), args)`.
The existing `OpConcat: "str.++"` fallback in `encodeIntrinsic` only triggers for
strings (since list concat is lowered to `concat_List`). No special dispatch needed.

---

## AILANG List Builtins → Z3 Sequence Theory Mapping

| AILANG | Z3 SMT-LIB | Arity | Notes |
|--------|-----------|-------|-------|
| `concat_List(xs, ys)` | `(seq.++ xs ys)` | 2 | List concatenation |
| `:: (x, xs)` | `(seq.++ (seq.unit x) xs)` | 2 | Cons (prepend) |
| `_list_length(xs)` | `(seq.len xs)` | 1 | **New builtin** |
| `_list_head(xs)` | `(seq.nth xs 0)` | 1 | **New builtin** |
| `_list_nth(xs, i)` | `(seq.nth xs i)` | 2 | **New builtin** |
| `[1, 2, 3]` literal | `(seq.++ (seq.unit 1) (seq.unit 2) (seq.unit 3))` | n | List construction |
| `[]` empty literal | `(as seq.empty (Seq T))` | 0 | Requires type context |

**Not encodable (deferred):**
- `map`, `filter`, `fold` — higher-order (fundamental limitation)
- `reverse`, `sort` — recursive (Phase E bounded recursion)
- `take`, `drop` — recursive

---

## Milestones

### M1: List Type Mapping + New Builtins (~180 LOC)

**Files to modify:**
- `internal/smt/types.go` — Map `TList` and `TApp("list",T)` → `(Seq T)` in `MapType`
- `internal/builtins/list.go` — Register `_list_length`, `_list_head`, `_list_nth` builtins
- `cmd/ailang/verify.go` — Handle `ListType` in `astTypeToSMTSort`
- `internal/smt/types_test.go` — Tests
- `internal/builtins/list_test.go` — Tests for new builtins

**Acceptance criteria:**
- `MapType(TList{Element: TCon("int")})` returns `"(Seq Int)"`, not error
- `MapType(TApp{Constructor: TCon("list"), Args: [TCon("string")]})` returns `"(Seq String)"`
- `_list_length([1,2,3])` returns `3` at runtime
- `_list_head([1,2,3])` returns `1` at runtime
- `_list_nth([1,2,3], 1)` returns `2` at runtime
- `astTypeToSMTSort` handles `ListType` → `"(Seq T)"`
- All existing tests still pass

### M2: List Literal + Builtin SMT Encoding (~200 LOC)

**Files to modify:**
- `internal/smt/codegen.go` — Encode `core.List` to `seq.unit` chains
- `internal/smt/codegen.go` — Encode list builtins to `seq.*` operations
- `internal/smt/types.go` — Add list builtins to maps (ListBuiltinSpecial)
- `internal/smt/codegen.go` — `inferResultSort` for List expressions
- `internal/smt/codegen_test.go` — Tests

**Acceptance criteria:**
- `core.List{Elements: [IntLit(1), IntLit(2)]}` encodes to `(seq.++ (seq.unit 1) (seq.unit 2))`
- Empty `core.List{}` encodes to `(as seq.empty (Seq Int))` (needs element type context)
- `concat_List` → `(seq.++ xs ys)`
- `::` → `(seq.++ (seq.unit elem) list)`
- `_list_length` → `(seq.len xs)` (unary)
- `_list_head` → `(seq.nth xs 0)` (unary + append 0)
- `_list_nth` → `(seq.nth xs idx)` (binary)
- `inferResultSort` returns appropriate sort for List
- All existing tests still pass

### M3: Fragment Checker Updates (~60 LOC)

**Files to modify:**
- `internal/smt/encodable.go` — Accept `core.List` nodes
- `internal/smt/encodable.go` — Update `isStringOrListBuiltin` for supported list builtins
- `internal/smt/encodable_test.go` — Tests

**Acceptance criteria:**
- `core.List` no longer triggers unencodable rejection
- `concat_List`, `::`, `_list_length`, `_list_head`, `_list_nth` accepted
- Higher-order list builtins (`map`, `filter`, `fold`) STILL rejected
- All existing fragment checker tests pass

### M4: Integration & Example (~80 LOC)

**Files to create:**
- `examples/runnable/contracts/list_verify.ail` — List verification example

**Files to modify:**
- `examples/manifest.json` — Add list_verify entry

**Acceptance criteria:**
- `list_verify.ail` has 3+ verified functions with list contracts
- `ailang verify examples/runnable/contracts/list_verify.ail` produces VERIFIED results
- At least one intentionally broken function for counterexample demo
- `ailang run --caps IO --entry main` works for the example

### M5: Documentation (~40 LOC)

**Files to modify:**
- `docs/docs/guides/contracts.mdx` — Update decidable fragment, add list section
- `CHANGELOG.md` — Add list verification entry

**Acceptance criteria:**
- "Currently skipped" table no longer lists Lists
- List verification section with example and output
- CHANGELOG updated

---

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Z3 `seq.*` performance for large lists | Medium | Contracts typically use small lists; timeout per function |
| Empty list encoding needs type context | Medium | Use `inferResultSort` from context or default to `(Seq Int)` |
| `::` (cons) operator lowering path unclear | Low | Check actual Core output; may need special pattern matching |
| `TList` vs `TApp("list",T)` dual representation | Low | Handle both in `MapType` |
| Polymorphic list builtins need monomorphization | Medium | Register monomorphic versions for int/string/bool |

---

## Velocity

- M-SMT-STRINGS completed in ~2.5 hours (5 milestones, ~450 LOC)
- M-SMT-RECORDS completed in ~2 hours (5 milestones, ~400 LOC)
- This sprint is slightly larger: ~600 LOC (new builtins add overhead)
- Estimated: ~4-5 hours of focused implementation
