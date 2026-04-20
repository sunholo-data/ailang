# M-CONCAT-DISAMBIG: Eliminate `++` Operator Ambiguity Once and For All

**Status**: Planned
**Target**: v0.13.0 (re-dated 2026-04-17 after v0.12.0 eval data re-confirmed the problem)
**Priority**: P1 (High) — still driving compile_error failures in v0.12.0 baseline
**Estimated Time**: 14-19 hours (Phase 1: 6-8h, Phase 2: 8-11h including migration sweeps)
**Dependencies**: None (M-STRING-INTERP is absorbed into Phase 1)
**Bug History**: v0.3.16, v0.5.8, v0.6.1, v0.7.0 — four separate fix cycles across every layer
**Supersedes**: [m-string-interpolation.md](../v0_11_0/m-string-interpolation.md)

> **Re-commissioned 2026-04-17 (v0.12.0 post-release review)**: v0.12.0 eval results show the unification ambiguity is still a leading compile-error driver even with the newest frontier models. See "v0.12.0 Eval Update" below.

## AI-First Alignment Check

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | ++ | +2 | Removes an operator from string context; interpolation is terser |
| Preserve Semantic Clarity | ++ | +2 | Eliminates the #1 source of type inference ambiguity |
| Increase Determinism | ++ | +2 | One operator, one meaning — no heuristic guessing |
| Lower Token Cost | + | +1 | Removes teaching prompt warnings; interpolation is shorter |
| **Net Score** | | **+7** | **Strong move forward** |

---

## Problem Statement

### The Core Issue

The `++` operator is overloaded for both strings and lists. This has caused bugs at **every layer of the compiler** across **four separate fix cycles**:

| Version | Layer | Bug |
|---------|-------|-----|
| v0.3.16 | Runtime | `++` type-checked for lists but runtime panicked (only `_str_concat` existed) |
| v0.5.8 | Codegen | Go codegen emitted `Concat()` but only `ConcatString()` existed |
| v0.6.1 | Codegen | `Concat` renamed to `ConcatList` but naming mismatch with `ToPascalCase` |
| v0.7.0 | Type inference | Both operands as type vars → defaulted to wrong overload |

**Root cause**: All four bugs stem from one design decision — `++` means two different things depending on operand types, and every layer must independently resolve this ambiguity.

### Impact: 100% Agent Failure Rate

From the v0.9.0 eval baseline (552 runs, 6 models):

- **`type_unify` benchmark**: 6/6 models fail. Every agent writes `s1 ++ s2` on list types.
- **`config_file_parser`**: 4/6 failures related to list operations.
- **The benchmark spec itself** uses `++` for list concat — making it an **impossible task**.

From dogfooding (`m-dx-package-dogfooding.md`):
> "Every AI agent (and human) will try `++` for lists first."

From the teaching prompt (v0.4.10+):
> "**NO `++` for lists** → Use `concat` from std/list"

When you need a bold warning in the teaching prompt to fight your own syntax, the language design is wrong.

### v0.12.0 Eval Update (2026-04-17)

Re-analysing the v0.12.0 baseline (`eval_results/baselines/v0.12.0/summary.jsonl`, 408 runs,
6 frontier models, prompt `v0.11.4`) shows the ambiguity has **not** been resolved by prompt work:

| Benchmark | AILANG failures | Error signature |
|-----------|-----------------|-----------------|
| `type_unify` | 5/8 models | `cannot unify type constructor string with *types.TList`, `cannot unify list[(string, Type)] with string` |
| `config_file_parser` | 3/8 models | `compile_error` (runtime + codegen fallout from same list/string heuristic) |

The exact compile_error traces confirm step 5 of the type-checker heuristic is still guessing
wrong on polymorphic list code (`TVar ++ TVar` defaulting to string when the unifier later
learns the operand is a list). Every affected model used `++` on lists in the generated code
despite the `v0.11.4` prompt warning.

**Implication for scheduling**: frontier model upgrades (Sonnet 4.6, Opus 4.7, GPT-5.4,
Gemini-3.1 Pro) did not make this go away. This is a language defect, not a prompt defect,
and a prompt patch in v0.13.0 cannot close the gap — the ambiguity must be removed at the
type-checker / grammar layer.

### Eval Data: 338 String Uses vs 17 List Uses

Analysis of all AILANG eval baselines (v0.9.1.1, 306 eval results, 6 models):

| `++` Usage | Count | Example |
|------------|-------|---------|
| **String concat** | 338 | `println("Sum: " ++ show(total))` |
| **List concat** | 17 | `unify(a1, a2) ++ unify(r1, r2)` |

**String `++` is the dominant use case by 20:1.** Every model writes patterns like:
```ailang
println("Label: " ++ show(value))            -- universal (all 6 models)
"(" ++ showExpr(a) ++ " + " ++ showExpr(b) ++ ")"  -- recursive string building
```

But most of these 338 uses are the pattern `"literal" ++ show(x)` which string
interpolation handles perfectly:
```ailang
println("Label: ${value}")                    -- replaces ~90% of string ++ usage
```

### Do Agents Already Try String Interpolation?

**No.** 0/306 eval results contain `${...}` or `f"..."` patterns. Models follow the teaching
prompt faithfully — when it says use `++`, they use `++`. This means:

1. **Models are prompt-obedient** — once we teach `"${expr}"`, they'll adopt it immediately
2. **No current breakage** from attempted interpolation (unlike `++` for lists, which agents try and fail)
3. **The migration path is smooth** — update the prompt, agents switch naturally

### Why Agents Expect `++` for Lists

Every ML-family language uses `++` for list concatenation:

| Language | List concat | String concat |
|----------|-------------|---------------|
| Haskell | `++` | `++` (String = [Char]) |
| OCaml | `@` | `^` |
| Elm | `++` | `++` |
| Erlang | `++` | `++` |
| Elixir | `++` | `<>` |
| F# | `@` | `+` |
| AILANG (current) | `std/list.concat` | `++` |

AILANG is the **only** ML-family language where `++` doesn't work on lists.

### Current Type Checker Heuristic

In `typechecker_operators.go:104-221`:

```
1. If incompatible concrete types (string + list) → error
2. If at least one concrete list → list concat
3. If at least one concrete string → string concat
4. If expected type from context is known → use that
5. Else (both type vars) → default to STRING   ← the problem
```

Step 5 is a guess. When it guesses wrong (which is often in recursive/polymorphic code),
agents get confusing type unification errors.

---

## Goals

**Primary Goal**: Eliminate `++` ambiguity so that AI agents can use `++` for list
concatenation without workarounds, errors, or teaching prompt caveats.

**Success Metrics**:
- `type_unify` benchmark: 0/6 → 4+/6 pass rate (currently impossible)
- Teaching prompt: Remove "NO `++` for lists" warning
- Zero `++`-related type inference bugs going forward
- All existing string use cases covered by interpolation + stdlib

---

## Design Decision

### The Narrowest, Most Coherent Design

AILANG removes overloaded `++` and reserves it exclusively for list concatenation.
String composition is not expressed with an infix operator. Instead, AILANG uses string
interpolation for inline formatting and stdlib functions for algorithmic composition.

This reduces syntax, removes a persistent source of type ambiguity, and aligns string
construction with the common AI-generated case of template-like text assembly.

```
Operators
  :: = cons (prepend element to list)
  ++ = list concat only

Strings
  "${expr}" = string interpolation (inline formatting)
  join(sep, parts) = join list of strings with separator    (already exists)
  concat(parts) = join without separator                    (add as join("", parts))
```

No new infix string operator. No `<>`.

### Why Not `<>`?

Adding `<>` was considered but rejected. Compared with interpolation-only:

1. **Fewer operators** — AILANG's syntax discipline is better served by one fewer symbol
2. **Less to teach** — no second concat operator to explain in prompts
3. **Better fit for AI generation** — models are already good at template-shaped output;
   interpolation aligns with that
4. **Cleaner compiler story** — the compiler no longer needs ANY string/list overload
   logic. That is the real prize.

### Where Interpolation Is Enough

For the large majority of actual string usage, interpolation covers it cleanly:

```ailang
"Hello, ${name}"                         -- greeting
"${prefix}${value}${suffix}"             -- assembly
"Found ${show(count)} items"             -- logging
"${path}/${file}"                        -- paths
"Error: ${msg} at line ${show(line)}"    -- diagnostics
```

This covers logging, messages, paths, formatting, templates, prompts, and diagnostics
— the overwhelming majority of the 338 string `++` uses in the eval baseline.

### Where Interpolation Is Insufficient

Interpolation is excellent for inline assembly of a known shape. It is less good for
**algorithmic composition** — these cases need stdlib functions:

**1. Folding a list of strings**
```ailang
-- Awkward with interpolation:
foldl(\acc s. "${acc}${s}", "", parts)

-- Clean with stdlib:
join("", parts)
```

**2. Conditional fragment assembly**
```ailang
let parts = filter(\s. length(s) > 0, [
  if showName then name else "",
  if showRole then role else ""
]);
join(", ", parts)
```

**3. Higher-order use (passing "combine strings" as a function)**
```ailang
-- Cannot pass interpolation as a function argument
-- Use stdlib instead:
foldl(\acc s. join("", [acc, s]), "", items)
```

**4. Recursive pretty-printing / serialization**
```ailang
-- Leaf nodes use interpolation:
pure func showExpr(e: Expr) -> string {
  match e {
    Lit(n) => show(n),
    Var(x) => x,
    Add(a, b) => "(${showExpr(a)} + ${showExpr(b)})",
    Mul(a, b) => "(${showExpr(a)} * ${showExpr(b)})"
  }
}
```
Note: recursive pretty-printing actually works well with interpolation because each
call site has a known template shape. This covers the `showExpr`/`showTerm` patterns
that appear frequently in the eval baseline.

### Stdlib Requirements

`join(sep, parts)` already exists (`_str_join` builtin, O(n) optimized).

**Additions needed:**
- `concat(parts: [string]) -> string` — convenience alias for `join("", parts)`.
  Trivial to add, but important because `concat` is what agents will reach for
  when they need non-interpolation string assembly.

**Already available:**
- `join(sep, parts)` — join with separator
- `repeat(s, n)` — repeat string n times
- `split(s, delim)` — split string

---

## Solution Options Considered

### Option A: `++` for Lists, `<>` for Strings
Add `<>` as infix string concat. Rejected: adds unnecessary syntax when interpolation covers
the same ground better.

### Option B: `++` Polymorphic with Better Inference
Keep `++` overloaded, fix with full bidirectional type checking. Rejected: attempted in
v0.7.0, still has TODO. 2-4 week undertaking, high regression risk. Every compiler layer
still needs both code paths.

### Option C: `++` for Lists, Interpolation for Strings (Selected)
No infix string operator. Interpolation for inline text, stdlib for algorithmic composition.
Fewest operators, cleanest compiler story, best AI fit.

### Option D: Runtime Dispatch (Status Quo+)
Keep overloading, improve heuristics. Rejected: this is what we've done for four versions
and it keeps breaking.

---

## Implementation Plan

### PHASE 1: String Interpolation (`"${expr}"`) — Ships Independently

*Subsumes M-STRING-INTERP design doc. ~6-8 hours. Zero breaking changes.*

After Phase 1, string `++` still works. Agents naturally shift to `"${...}"` once the
teaching prompt is updated. This phase alone delivers 80% of the value.

#### 1A: Lexer (~2 hours)

- [ ] Detect `${` inside string literals
- [ ] Emit token sequence: `STRING_PART`, `INTERP_START`, expr tokens, `INTERP_END`, `STRING_PART`
- [ ] Handle nested `{}` in expressions (brace counting)
- [ ] Handle escaped `\${` (literal dollar-brace)
- [ ] Strings without `${` remain plain `STRING` tokens (no change)

#### 1B: Parser + Desugaring (~1.5 hours)

- [ ] Parse interpolated string as sequence of string parts + expressions
- [ ] Desugar to concat chain with implicit `show()` calls:
  ```
  "Hello, ${name}! Count: ${x + 1}"
  →  concat_String(concat_String(concat_String("Hello, ", show(name)),
                                  "! Count: "), show(x + 1))
  ```
- [ ] Strings in interpolation don't get double-wrapped: `"${name}"` → `name` (not `show(name)`) when already string-typed
- [ ] Handle edge cases: `"${x}"` (expression only), `"no interp"` (plain string)

#### 1C: Type Checking (~1 hour)

- [ ] Interpolated expressions must resolve to a type with `show` support
- [ ] String-typed expressions pass through without `show()` wrapping
- [ ] Int, float, bool auto-show; other types need explicit `show()`

#### 1D: Evaluator + Codegen (~1.5 hours)

- [ ] Evaluator handles desugared concat chain (no new eval logic needed)
- [ ] Go codegen: interpolated strings compile to `ConcatString` chains
- [ ] WASM/SMT codegen: same desugaring

#### 1E: Testing & Prompt Update (~1 hour)

- [ ] Unit tests for lexer interpolation tokenization
- [ ] Integration tests for interpolated strings
- [ ] Update teaching prompt: prefer `"${expr}"` for string building
- [ ] Update a few key examples to demonstrate interpolation
- [ ] `make ci` clean

**Phase 1 deliverable**: String interpolation works. Teaching prompt teaches it as primary
string-building syntax. String `++` still works but is discouraged.

---

### PHASE 2: `++` List-Only — Requires Phase 1

*~4-6 hours. Ships separately, possibly v0.11.1 or v0.12.0.*

After Phase 1 absorbs most string `++` usage, this phase restricts `++` to lists only
and adds `concat(parts)` to stdlib for the remaining algorithmic cases.

#### 2A: Add `concat` Stdlib Function (~30 min)

- [ ] Add `concat(parts: [string]) -> string` — alias for `join("", parts)`
- [ ] Register in builtins
- [ ] Add to teaching prompt

#### 2B: Restrict `++` to Lists Only (~2 hours)

**Type Checker** (`internal/types/typechecker_operators.go`):
- [ ] Remove the entire string/list heuristic (lines 104-221 → ~30 lines)
- [ ] `++` always types as `[a] -> [a] -> [a]`
- [ ] If operand is string, emit helpful error:
  `"++ is for list concatenation. Use string interpolation or join() for strings."`

**Operator Lowering** (`internal/pipeline/op_lowering.go`):
- [ ] `OpConcat` always resolves to `concat_List` — remove "String" from types list
- [ ] Remove the default-to-string fallback

**Evaluator**:
- [ ] `OpConcat` always calls `_list_concat`

#### 2C: Example Migration (~135 files, ~636 `++` occurrences)

The audit found **135 example files** containing **636 `++` uses** (96% string, 4% list).
After Phase 1, most will already use interpolation. This sweep catches stragglers.

**Runnable examples** (`examples/runnable/` — 100+ files):
- [ ] Migrate `string_repeat.ail`, `string_contains.ail`, `json_parsing.ail`, etc.
- [ ] Pattern: `println("Label: " ++ show(x))` → `println("Label: ${show(x)}")`
- [ ] Verify each still compiles and produces correct output

**Bug/debug examples** (`examples/bugs/`, `examples/debug/`):
- [ ] Update `concat_operator_list_inference.ail` — now tests the *correct* behavior
- [ ] Update `list_concat_match.ail` — list `++` now works natively
- [ ] Update `list_concat.ail` — simplify debug example

**Doc examples** (`examples/docs/`, `examples/tests/`):
- [ ] Sweep all .ail files; convert string `++` to interpolation
- [ ] Leave list `++` as-is (now correct)

**Verification**: `make verify-examples` must pass after migration.

#### 2D: Benchmark Migration (~2 YML specs + solutions)

- [ ] `benchmarks/type_unify.yml` — **CRITICAL**: currently an impossible task because spec
  uses `++` for lists. After Phase 2, it becomes solvable. No spec change needed.
- [ ] `benchmarks/symbolic_diff.yml` — 4 lines with string `++` in examples.
  Migrate to interpolation in description.
- [ ] Sweep `benchmarks/**/*.ail` solution files for string `++`

#### 2E: Teaching Prompt Sweep (~49 prompt versions)

All 49 prompt versions in `prompts/` reference `++`. Only the latest active prompt
needs functional changes, but all should be consistent for historical reference.

**Active prompt** (`prompts/v0.9.0.md` or latest):
- [ ] Line ~109: `println(person.name ++ ", " ++ show(person.age))` → interpolation
- [ ] Line ~165: Update "What AILANG Does NOT Have" — remove `concat(a, b)` deprecation note
- [ ] Line ~354: `println("Hello " ++ name)` → `println("Hello ${name}")`
- [ ] Line ~411: `println("Got: " ++ line)` → `println("Got: ${line}")`
- [ ] Lines ~33-34: `"(" ++ showExpr(...)` → `"(${showExpr(...)})"` (recursive pattern)
- [ ] Add operator table:
  ```
  | `++`          | List concat          | `[1, 2] ++ [3, 4]` → `[1, 2, 3, 4]` |
  | `::`          | Cons (prepend)       | `1 :: [2, 3]` → `[1, 2, 3]`          |
  | `"${expr}"`   | String interpolation | `"Hello, ${name}"`                    |
  | `join(s, xs)` | Join strings         | `join(", ", ["a", "b"])` → `"a, b"`   |
  | `concat(xs)`  | Concat strings       | `concat(["a", "b"])` → `"ab"`         |
  ```
- [ ] Remove any remaining "NO `++` for lists" caveats
- [ ] Add section: "String building — prefer `${...}` for inline, `join()`/`concat()` for algorithmic"

#### 2F: Documentation Sweep (~30 files in `docs/`)

- [ ] `docs/docs/reference/language-syntax.md` — Update operator table (3 mentions of `++`)
- [ ] `docs/LIMITATIONS.md` — Remove any `++` limitations if present; add note about
  `++` being list-only as a **resolved** limitation
- [ ] `docs/docs/guides/testing.md` — Update example code using `++` for strings
- [ ] `docs/docs/guides/debugging.md` — Update example code
- [ ] `docs/docs/reference/effects.md` — Update example code
- [ ] `docs/docs/architecture/anf.md` — Update `++` reference in ANF discussion
- [ ] `docs/VISION.md` — Update operator philosophy if referenced
- [ ] Sweep remaining ~24 files with `++` references

#### 2G: Test Migration (~500-700 LOC across 4+ test files)

**Dedicated concat test file** (`internal/pipeline/concat_operator_test.go` — 189 lines):
- [ ] `TestConcatRecursiveString` — remove or convert to error-expects-failure test
- [ ] `TestConcatConcreteString` — remove (string `++` no longer valid)
- [ ] `TestConcatListWithSignature` — keep (now the primary use case)
- [ ] `TestConcatConcreteList` — keep
- [ ] Add: `TestConcatStringError` — verify `"a" ++ "b"` produces helpful error message
- [ ] Add: `TestConcatListPolymorphic` — verify `xs ++ ys` works with type vars

**Operator method tests** (`internal/types/operator_method_test.go`):
- [ ] Update type inference tests for `++` to expect list-only behavior
- [ ] Add test: `++` on string operands → clear error

**Op lowering tests** (`internal/pipeline/op_lowering_test.go`):
- [ ] Remove `concat_String` lowering tests for `++`
- [ ] Verify `++` always lowers to `concat_List`

**Builtin tests** (`internal/builtins/string_test.go`, `list_test.go`):
- [ ] `_str_concat` tests — keep (still used internally by interpolation desugaring)
- [ ] `_list_concat` tests — keep
- [ ] Add: `concat([string])` stdlib function tests

**Codegen tests** (`internal/gen/golang/codegen_test.go`):
- [ ] `++` codegen tests — verify `ConcatList` only (no `ConcatString` from `++`)
- [ ] Add: interpolation codegen tests (if not covered in Phase 1)

#### 2H: Codegen (~1 hour)

- [ ] Go codegen: `++` always emits `ConcatList` (no more `ConcatString` from `++`)
- [ ] WASM codegen: same
- [ ] SMT codegen: same
- [ ] Interpolation desugaring still uses `ConcatString` internally (no change needed)

#### 2I: Final Validation

- [ ] `make test` — all tests pass
- [ ] `make lint` — no linter warnings
- [ ] `make verify-examples` — all examples compile and run correctly
- [ ] `make ci` — full CI clean
- [ ] `type_unify` benchmark now passable by agents
- [ ] Run eval suite subset on affected benchmarks to verify improvement

---

## Full Impact Audit

### Codebase-Wide `++` Usage (from audit of all files)

| Category | Files | `++` Occurrences | String | List | Phase |
|----------|-------|-----------------|--------|------|-------|
| **Examples** (`examples/`) | 135 | 636 | ~610 | ~26 | Phase 2C |
| **Benchmarks** (`benchmarks/`) | 2 YML | 3 | 2 | 1 (CRITICAL) | Phase 2D |
| **Teaching prompts** (`prompts/`) | 49 | ~300 refs | Teaching content | — | Phase 1E + 2E |
| **Documentation** (`docs/`) | 30 | ~100 refs | Examples, guides | — | Phase 2F |
| **Compiler** (`internal/types/`) | 1 | 118 LOC | Heuristic logic | — | Phase 2B |
| **Compiler** (`internal/pipeline/`) | 2 | ~50 LOC | Lowering/dispatch | — | Phase 2B |
| **Builtins** (`internal/builtins/`) | 2 | 2 impls | `_str_concat` | `_list_concat` | Phase 2A |
| **Tests** (Go) | 4+ | 500-700 LOC | Type/lowering/eval | — | Phase 2G |
| **Design docs** | 5 | N/A | Historical context | — | Cross-ref only |

### Files to Modify/Create

#### Phase 1 (String Interpolation — ~165 LOC new)

| File | Change | LOC |
|------|--------|-----|
| `internal/lexer/lexer.go` | Interpolation tokenization (`${` handling) | ~40 |
| `internal/lexer/token.go` | Add `STRING_PART`, `INTERP_START`, `INTERP_END` tokens | ~10 |
| `internal/parser/parser.go` | Parse interpolated strings, desugar to concat chain | ~30 |
| `internal/types/typechecker.go` | Type-check interpolated exprs, auto-`show()` | ~15 |
| `internal/eval/evaluator.go` | No change (desugared before eval) | 0 |
| `internal/gen/golang/codegen_*.go` | No change (desugared before codegen) | 0 |
| Tests (lexer + parser + integration) | New test cases | ~60 |
| `prompts/v0.X.md` (active prompt only) | Add interpolation section + examples | ~10 |

#### Phase 2 (++ List-Only — net **simpler**)

| File | Change | Added | Removed |
|------|--------|-------|---------|
| **Compiler** | | | |
| `internal/types/typechecker_operators.go` | Replace 118-line heuristic with list-only | +20 | -90 |
| `internal/pipeline/op_table.go` | Remove "String" from concat types | 0 | -2 |
| `internal/pipeline/op_lowering.go` | Remove string fallback logic | 0 | -15 |
| `internal/builtins/string_ops.go` | Add `concat(parts: [string])` stdlib | +10 | 0 |
| `internal/gen/golang/codegen_*.go` | `++` always → `ConcatList` | +5 | -10 |
| **Tests** | | | |
| `internal/pipeline/concat_operator_test.go` | Migrate string→error tests, add list tests | +30 | -40 |
| `internal/types/operator_method_test.go` | Update ++ inference tests | +10 | -10 |
| `internal/pipeline/op_lowering_test.go` | Remove string lowering tests | +5 | -15 |
| `internal/builtins/string_test.go` | Add `concat()` stdlib tests | +15 | 0 |
| **Examples** (135 files, 636 occurrences) | | | |
| `examples/runnable/*.ail` (~100 files) | String `++` → interpolation | ~200 | ~200 |
| `examples/bugs/*.ail` (~5 files) | Update concat test cases | ~10 | ~10 |
| `examples/debug/*.ail` (~3 files) | Simplify list concat debug examples | ~5 | ~5 |
| `examples/docs/*.ail`, `examples/tests/*.ail` | Sweep remaining | ~20 | ~20 |
| **Benchmarks** | | | |
| `benchmarks/type_unify.yml` | No change needed (spec already uses ++ for lists) | 0 | 0 |
| `benchmarks/symbolic_diff.yml` | Migrate description examples | ~4 | ~4 |
| **Teaching Prompts** (active prompt) | | | |
| `prompts/v0.X.md` | Rewrite operator table, remove caveats, add examples | ~30 | ~20 |
| **Documentation** (30 files) | | | |
| `docs/docs/reference/language-syntax.md` | Update operator table | ~10 | ~5 |
| `docs/LIMITATIONS.md` | Add resolved limitation note | ~5 | 0 |
| `docs/docs/guides/testing.md` | Update example code | ~5 | ~5 |
| `docs/docs/guides/debugging.md` | Update example code | ~5 | ~5 |
| `docs/docs/reference/effects.md` | Update example code | ~3 | ~3 |
| `docs/docs/architecture/anf.md` | Update ++ ANF reference | ~2 | ~2 |
| Remaining ~24 doc files | Sweep string `++` examples | ~20 | ~20 |

#### Summary

| Phase | LOC Added | LOC Removed | Net | Files Touched |
|-------|-----------|-------------|-----|---------------|
| Phase 1 | ~165 | 0 | +165 | ~8 |
| Phase 2 (compiler) | ~95 | ~182 | -87 | ~8 |
| Phase 2 (examples) | ~235 | ~235 | 0 | ~135 |
| Phase 2 (prompts/docs) | ~80 | ~60 | +20 | ~32 |
| **Total** | **~575** | **~477** | **+98** | **~183** |

The compiler becomes **87 LOC simpler**. The net +98 comes from new interpolation
infrastructure and `concat()` stdlib — features, not complexity.

---

## Examples

### List Concatenation (Currently Broken → Fixed)

```ailang
-- Agent writes this (natural, matches Haskell/Elm):
export func unify(t1: Type, t2: Type) -> Option[Subst] {
  match (t1, t2) {
    (TFunc(a1, r1), TFunc(a2, r2)) => {
      let s1 = unify(a1, a2);
      let s2 = unify(r1, r2);
      Some(s1 ++ s2)          -- Currently FAILS. After: Works!
    }                          -- ++ is always list concat, no heuristic.
  }
}

-- List operations are clean and unambiguous:
export func flatten[a](xss: [[a]]) -> [a] {
  match xss {
    [] => [],
    xs :: rest => xs ++ flatten(rest)   -- Always list concat
  }
}
```

### String Building (Before → After)

```ailang
-- BEFORE: 338 occurrences of this pattern in eval baseline
println("Label: " ++ show(value))
"(" ++ showExpr(a) ++ " + " ++ showExpr(b) ++ ")"
"IDLE @ floor " ++ show(floor)

-- AFTER: string interpolation
println("Label: ${show(value)}")
"(${showExpr(a)} + ${showExpr(b)})"
"IDLE @ floor ${show(floor)}"
```

### Algorithmic String Composition

```ailang
-- Joining a list of strings
let csv = join(",", fields)

-- Concatenating fragments without separator
let path = concat([dir, "/", file])

-- Conditional assembly
let parts = filter(\s. length(s) > 0, [
  if showName then name else "",
  if showRole then role else ""
]);
let label = join(", ", parts)
```

### Recursive Pretty-Printing

```ailang
-- Interpolation works naturally for recursive tree-walking:
pure func showExpr(e: Expr) -> string {
  match e {
    Lit(n) => show(n),
    Var(x) => x,
    Add(a, b) => "(${showExpr(a)} + ${showExpr(b)})",
    Mul(a, b) => "(${showExpr(a)} * ${showExpr(b)})"
  }
}
-- Each call site has a known template shape — interpolation is ideal here.
```

---

## Migration Guide

### For Existing AILANG Code

```
BEFORE                                    AFTER
─────                                     ─────
"Hello, " ++ name                         "Hello, ${name}"
"Sum: " ++ show(total)                    "Sum: ${show(total)}"
show(x) ++ sep ++ show(y)                 "${show(x)}${sep}${show(y)}"
[1, 2] ++ [3, 4]                          [1, 2] ++ [3, 4]  (unchanged)
s1 ++ s2 (on lists)                        s1 ++ s2           (now works!)
```

### For Teaching Prompts

Remove:
```
**NO `++` for lists** → Use `concat` from std/list
```

Replace with:
```
| Syntax | Purpose | Example |
|--------|---------|---------|
| `++` | List concat | `[1, 2] ++ [3, 4]` → `[1, 2, 3, 4]` |
| `::` | Cons (prepend) | `1 :: [2, 3]` → `[1, 2, 3]` |
| `"${expr}"` | String interpolation | `"Hello, ${name}"` |
| `join(sep, parts)` | Join strings | `join(", ", ["a", "b"])` → `"a, b"` |
| `concat(parts)` | Concat strings | `concat(["a", "b"])` → `"ab"` |
```

### For Benchmarks

Update benchmark specs like `type_unify.yml` — they already use `++` for lists, so they'll
now match the language.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking change for `"a" ++ "b"` | Medium | Phase 1 absorbs ~90% first; migration is mechanical |
| No infix string concat feels limiting | Low | Interpolation covers the dominant case; stdlib covers the rest |
| Agents write `foldl(\acc s. "${acc}${s}", ...)` | Low | Teaching prompt steers to `join()`; we explicitly document this |
| Lexer complexity for `${` inside strings | Medium | Well-understood problem; JS/Kotlin/Dart all solve it the same way |
| Phase 2 deferred indefinitely | Low | Phase 1 alone is a significant DX win; Phase 2 is the cleanup |

---

## Success Criteria

### Phase 1

- [ ] `"Hello, ${name}"` compiles and runs correctly
- [ ] Nested expressions work: `"Result: ${compute(a, b)}"`
- [ ] Escaped `\${` produces literal `${`
- [ ] Type errors for non-showable types in interpolation
- [ ] String-typed expressions don't get double-wrapped with `show()`
- [ ] Teaching prompt updated to prefer interpolation
- [ ] `make ci` passes
- [ ] `make verify-examples` passes (no examples broken)

### Phase 2

**Compiler:**
- [ ] `++` on lists works without type errors in all contexts (recursive, polymorphic)
- [ ] `++` on strings produces clear error: "Use string interpolation or join() for strings"
- [ ] `concat(["a", "b"])` returns `"ab"` (new stdlib function)
- [ ] Type checker heuristic removed — `++` is `[a] -> [a] -> [a]`, period

**Migration sweeps:**
- [ ] 0 remaining string `++` in `examples/` (135 files migrated)
- [ ] 0 remaining string `++` in `prompts/` active prompt
- [ ] 0 remaining string `++` in `docs/` (30 files updated)
- [ ] `benchmarks/type_unify.yml` now solvable (was impossible)
- [ ] `benchmarks/symbolic_diff.yml` examples use interpolation

**Tests:**
- [ ] `make test` — all Go tests pass
- [ ] `make verify-examples` — all .ail examples compile and run
- [ ] `make lint` — clean
- [ ] `make ci` — full CI green

**Eval validation:**
- [ ] Run eval suite subset on `type_unify`, `config_file_parser`, `graph_bfs`
- [ ] Verify improved pass rates (target: type_unify 4+/6, up from 0/6)

---

## Relationship to Other Features

- **M-STRING-INTERP** (planned): Absorbed into Phase 1 of this doc. This doc supersedes it.
- **M-CONS-EXPRESSION** (v0.10.0, implemented): `::` in expressions already landed. With `++` as list-only, AILANG has a complete, unambiguous list operator set: `::` (cons) and `++` (concat).
- **Type classes** (future): If AILANG adds type classes, `++` could become `Monoid.mconcat` for lists. No string operator to migrate.

---

## Timeline

### Phase 1: String Interpolation (can ship as v0.10.1)

**Day 1** (4-5 hours):
- 1A: Lexer — `${` detection, token emission, brace counting
- 1B: Parser — parse interpolated strings, desugar to concat chain
- 1C: Type checking — auto-show, string passthrough

**Day 2** (2-3 hours):
- 1D: Verify evaluator/codegen handle desugared form (should be zero-change)
- 1E: Tests — lexer tokenization, parser desugaring, integration
- 1E: Update active teaching prompt — add interpolation as preferred syntax
- Phase 1 **complete** — can release independently

**Phase 1 total: ~6-8 hours**

### Phase 2: ++ List-Only (ships as v0.11.1 or v0.12.0)

**Day 3** (3-4 hours):
- 2A: Add `concat()` stdlib function (~30 min)
- 2B: Restrict `++` to list-only in type checker, lowering, evaluator
- 2G: Migrate Go tests — update concat_operator_test.go, op_lowering_test.go

**Day 4** (3-4 hours):
- 2C: Example sweep — migrate ~135 .ail files (mostly mechanical find-replace,
  but each needs verification: `make verify-examples`)
- 2D: Benchmark migration — update symbolic_diff.yml description

**Day 5** (2-3 hours):
- 2E: Teaching prompt sweep — rewrite operator section of active prompt
- 2F: Documentation sweep — update ~30 files in `docs/`
- 2H: Codegen — verify `++` always emits `ConcatList`
- 2I: Final validation — `make ci`, eval suite subset

**Phase 2 total: ~8-11 hours**

**Combined total: ~14-19 hours across ~5 days**

Note: The example migration (2C) is the largest single task by file count (135 files)
but is mostly mechanical. Could be parallelized or automated with a migration script.

---

## Design Rationale (Summary)

> AILANG removes overloaded `++` and reserves it exclusively for list concatenation.
> String composition is not expressed with an infix operator. Instead, AILANG uses
> string interpolation for inline formatting and stdlib functions such as `join` and
> `concat` for algorithmic composition.
>
> This reduces syntax, removes a persistent source of type ambiguity, and aligns
> string construction with the common AI-generated case of template-like text assembly.

---

## References

- [list-concatenation-operator-fix.md](../../implemented/v0_3_16/list-concatenation-operator-fix.md) — v0.3.16 runtime fix
- [m-codegen-list-concat.md](../../implemented/v0_5_8/m-codegen-list-concat.md) — v0.5.8 codegen fix
- [m-dx17-codegen-concatlist-closure-scoping.md](../../implemented/v0_6_1/m-dx17-codegen-concatlist-closure-scoping.md) — v0.6.1 naming fix
- [concat-operator-type-inference-bug.md](../../implemented/v0_7_0/concat-operator-type-inference-bug.md) — v0.7.0 inference fix
- [m-dx-agent-eval-gaps.md](../v0_11_0/m-dx-agent-eval-gaps.md) — v0.9.0 eval gap analysis (100% failure evidence)
- [m-dx-package-dogfooding.md](../../planned/v1_0_0/m-dx-package-dogfooding.md) — Dogfooding friction report
- [m-string-interpolation.md](../v0_11_0/m-string-interpolation.md) — Original string interpolation doc (superseded by Phase 1)

---

**Document created**: 2026-03-30
**Last updated**: 2026-03-30
