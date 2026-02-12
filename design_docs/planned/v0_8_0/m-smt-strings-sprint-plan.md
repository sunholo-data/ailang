# Sprint Plan: M-SMT-STRINGS — String Verification (Phase B)

**Sprint ID**: M-SMT-STRINGS
**Design Doc**: [m-smt-fragment-expansion.md](m-smt-fragment-expansion.md) (Phase B)
**Duration**: 1 day (~6 hours)
**Risk Level**: Low (Z3 string theory is well-established)
**Estimated LOC**: ~500 (implementation + tests)

---

## Summary

Add string verification to `ailang verify`. Functions using string parameters, string
literals, concatenation, length, comparison, and substring operations will be encodable
in SMT-LIB using Z3's built-in string theory.

**Before**: Any function touching strings is SKIPPED with "uses types not encodable in SMT"
**After**: Pure functions with string types can be VERIFIED or produce counterexamples

---

## AILANG String Builtins → Z3 String Theory Mapping

| AILANG Builtin | Z3 SMT-LIB | Arity | Notes |
|---------------|-------------|-------|-------|
| `concat_String(a, b)` | `(str.++ a b)` | 2 | String concatenation |
| `eq_String(a, b)` | `(= a b)` | 2 | String equality |
| `ne_String(a, b)` | `(distinct a b)` | 2 | String inequality |
| `lt_String(a, b)` | `(str.< a b)` | 2 | Lexicographic less-than |
| `le_String(a, b)` | `(str.<= a b)` | 2 | Z3 4.8.8+ |
| `gt_String(a, b)` | `(str.< b a)` | 2 | Flipped operands |
| `ge_String(a, b)` | `(str.<= b a)` | 2 | Flipped operands |
| `_str_len(s)` | `(str.len s)` | 1 | String length |
| `_str_find(s, t)` | `(str.indexof s t 0)` | 2 | Find substring |
| `_str_slice(s, i, j)` | `(str.substr s i (- j i))` | 3 | Substring |
| `_str_startsWith(s, p)` | `(str.prefixof p s)` | 2 | Prefix check |
| `_str_endsWith(s, x)` | `(str.suffixof x s)` | 2 | Suffix check |
| `OpConcat` (intrinsic) | `(str.++ a b)` | 2 | Pre-lowered `++` |

**Not encodable (deferred):**
- `_str_trim`, `_str_upper`, `_str_lower` — No Z3 equivalent
- `_str_split`, `_str_chars` — Returns list (requires Phase D)
- `_stringToInt`, `_stringToFloat` — Complex conversion semantics

---

## Milestones

### M1: String Type Mapping + Literal Encoding (~120 LOC)

**Files to modify:**
- `internal/smt/types.go` — Map `string` → `String` sort in `mapTCon`
- `internal/smt/codegen.go` — Encode `StringLit` as SMT-LIB string literal
- `internal/smt/codegen.go` — Add `string` to `inferResultSort`
- `cmd/ailang/verify.go` — Handle `string` return type in `astTypeToSMTSort`
- `internal/smt/types_test.go` — Tests
- `internal/smt/codegen_test.go` — Tests

**Acceptance criteria:**
- `MapType(TCon("string"))` returns `"String"`, not error
- `encodeLit(StringLit("hello"))` returns `"\"hello\""`
- String escape sequences handled (`\"`, `\\`)
- `inferResultSort` recognizes string literals → `"String"`
- `astTypeToSMTSort` handles `string` surface type
- All existing tests still pass

### M2: String Builtin Operations (~200 LOC)

**Files to modify:**
- `internal/smt/types.go` — Add 12 entries to `BuiltinToSMTOp`
- `internal/smt/codegen.go` — Special-case handlers for non-standard arity builtins
- `internal/smt/codegen.go` — Handle `OpConcat` intrinsic → `str.++`
- `internal/smt/codegen_test.go` — Tests for each builtin mapping

**Acceptance criteria:**
- `concat_String` → `str.++`
- `eq_String`/`ne_String` → `=`/`distinct`
- `lt_String`/`le_String`/`gt_String`/`ge_String` → correct Z3 operators
- `_str_len` (unary) encodes correctly
- `_str_find` (binary + implicit 0) encodes correctly
- `_str_slice` (ternary with offset calc) encodes correctly
- `_str_startsWith`/`_str_endsWith` with flipped operand order
- `OpConcat` intrinsic handled
- All existing 103+ tests still pass

### M3: Fragment Checker Updates (~80 LOC)

**Files to modify:**
- `internal/smt/encodable.go` — Remove string rejection from `walkForUnencodableTypes`
- `internal/smt/encodable.go` — Update `isStringOrListBuiltin` to only reject `_List`
- `internal/smt/encodable.go` — Update error message in `IsSMTEncodable`
- `internal/smt/encodable_test.go` — Update tests

**Acceptance criteria:**
- `StringLit` no longer triggers unencodable rejection
- `OpConcat` no longer triggers unencodable rejection
- `concat_String`, `eq_String`, `_str_len`, etc. no longer rejected
- `_str_split`, `_str_chars` STILL rejected (they return lists)
- `_str_trim`, `_str_upper`, `_str_lower` STILL rejected (no Z3 equivalent)
- List builtins still properly rejected
- All existing fragment checker tests updated/passing

### M4: Integration & Example (~70 LOC)

**Files to create:**
- `examples/runnable/contracts/string_verify.ail` — String verification example

**Files to modify:**
- `examples/manifest.json` — Add string_verify entry

**Acceptance criteria:**
- `string_verify.ail` has 3+ verified functions with string contracts
- `ailang verify examples/runnable/contracts/string_verify.ail` produces VERIFIED results
- At least one intentionally broken function for counterexample demo
- `ailang run --caps IO --entry main` works for the example
- E2E: all 4 milestones produce correct SMT-LIB output

### M5: Documentation (~40 LOC)

**Files to modify:**
- `docs/docs/guides/contracts.mdx` — Update decidable fragment, add string section
- `CHANGELOG.md` — Add string verification entry

**Acceptance criteria:**
- "Currently skipped" table no longer lists Strings
- String verification section with example and output
- CHANGELOG updated under current version

---

## Velocity

- Recent velocity: M-SMT-RECORDS completed in ~2 hours (5 milestones, ~400 LOC)
- This sprint is similar scope: ~500 LOC across 5 milestones
- Estimated: ~4 hours of focused implementation

---

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| `str.<=` not available in older Z3 | Low | Use `(or (str.< a b) (= a b))` fallback |
| String escape edge cases | Low | Use Go `strconv.Quote` for encoding |
| `_str_slice` offset calculation mismatch | Medium | Z3 `str.substr` takes start+length, AILANG takes start+end |
| Some string builtins need non-standard encoding | Low | Use special-case handlers in encodeApp |
