# M-ERROR-PROP: Error Propagation Operator (`?`)

**Status**: **PARKED — needs-human-review** (2026-07-23, mission iteration 88). Rev-1 (below) fixed the Rev-0 desugaring soundness flaw, but a second quorum round surfaced deeper **implementation-premise** objections that require a code-audit spike + an architecture decision (see "Quorum History & Open Questions" below). Do NOT route to sprint-planner until these are resolved by a human/arni design decision.

### Quorum History & Open Questions (mission iter 88)

- **Rev-0 → REJECTED** (2026-07-23): the "local match replacement" desugaring was semantically invalid — incompatible branch types (`T` vs `Result[_,E]`), no early return, `Ok(Err(e))` wrapping in nested blocks. `?` is non-local control flow and cannot be desugared by local node replacement in an expression-only ANF Core IR with no `Return` node.
- **Rev-1 → RE-QUORUM REJECTED** (2026-07-23): the continuation-threading (ANF-bind) rewrite below is the correct *shape*, but two premise-level objections remain **open for a human**:
  1. **(gpt5-6-sol) Unverified codebase premises.** The lowering assumes `internal/elaborate/core.go:309` safely flattens a let-RHS block into the enclosing continuation, that `normalizeToAtomic`/binding-discharge can float a new try-binding to the function-body spine without changing scope/eval-order, and that `NodeID`-based `TryOrigin` metadata can reach `CoreTypeChecker` diagnostics. **None is backed by a commit-SHA inspection or an executable characterization test.** → Needs a code-audit spike proving (or refuting) each, with a characterization test per premise.
  2. **(gemini-3-1-pro) Compiler↔stdlib linkage + re-boxing.** The elaborator emits Core `Match` using `Ok`/`Err`, but `Result` is a **user-space** type in `std/result.ail` — there is no lang-items registry for the compiler to discover the `ConstructorID`s. **This is the load-bearing architecture question** (add a lang-items/known-constructor registry vs. another mechanism — an arni/human decision). Also: the doc's "no re-boxing" claim is false (`Err($eN) => Err($eN)` re-allocates the variant to retype `Result[T,E]`→`Result[Ret,E]`); correct the Evaluation-Guarantees section when unparking.

*Original Rev-1 note:* revised to address quorum objections 1 (gpt5-6-sol) & 2 (gemini-3-1-pro). Replaced the invalid desugaring with a **whole-enclosing-body continuation-threading (ANF-bind) transform**, added the mandatory Conflict Surface section, and corrected the Files-to-Modify table against the live repo (several paths were stale).
**Target**: v1.1.0
**Priority**: P2 (Medium) - Ergonomic error handling, advances Axiom A11
**Dependencies**: Result type (std/result.ail) - already implemented

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Desugaring is pure syntactic transform, deterministic |
| A2: Replayability | +1 | Traceable via elaborated Core form (spans preserved) |
| A3: Effect Legibility | +1 | Error propagation preserves effect signatures unchanged |
| A4: Explicit Authority | 0 | No authority changes (Result is pure data) |
| A5: Bounded Verification | +2 | Compile-time check that enclosing function returns Result |
| A6: Safe Concurrency | 0 | No concurrency impact |
| A7: Machines First | +2 | **Key benefit** - 70% token reduction in error handling code |
| A8: Minimal Syntax | +1 | Single postfix character, low syntactic overhead |
| A9: Cost Visibility | 0 | No cost implications |
| A10: Composability | +2 | `a()?.b()?.c()` chains cleanly, desugars predictably |
| A11: Structured Failure | +2 | **Primary goal** - makes Result patterns ergonomic |
| A12: System Boundary | +1 | Errors remain typed across module boundaries |

**Net Score: +13** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Pure syntactic desugaring, no runtime nondeterminism
- [x] A3 (Effects): Does not hide or introduce effects
- [x] A4 (Authority): No ambient authority (Result is pure)
- [x] A7 (Machines First): Improves machine readability (less nesting)
- [x] A11 (Structured Failure): Enhances, does not weaken structured errors

### A2 Note (Traceability)

The `?` operator is visible in traces via its elaborated Core form. Elaboration preserves source spans through `CoreNode.OrigSpan`, so traces can map synthetic match expressions back to the original `?` position. Users see the surface `?` in error messages, not the internal match structure.

---

## Problem Statement

Error handling with `Result` types requires verbose nested pattern matching:

```ailang
-- Current (verbose) - 8 lines, deeply nested:
func processFile(path: string) -> Result[string, Error] ! {FS} =
  match readFile(path) {
    Ok(content) => match parseContent(content) {
      Ok(data) => match transform(data) {
        Ok(result) => Ok(result),
        Err(e) => Err(e)
      },
      Err(e) => Err(e)
    },
    Err(e) => Err(e)
  }

-- Desired (concise) - 4 lines, flat:
func processFile(path: string) -> Result[string, Error] ! {FS} =
  let content = readFile(path)? in
  let data = parseContent(content)? in
  let result = transform(data)? in
  Ok(result)
```

**Impact:**
- 70% token reduction in typical error-handling code
- Reduced nesting depth (AI models handle flat code better)
- Matches patterns from Rust, Swift, making AILANG more accessible

---

## Goals

**Primary Goal:** Add `?` postfix operator for early-return on `Err`.

**Success Metrics:**
- `expr?` syntax parses and type-checks
- Early return works correctly at runtime
- Compile error for `?` in non-Result functions
- Chained `?` works: `a()?.b()?.c()`
- Axiom A11 gap closed (score 1→2)

---

## Solution Design

### Syntax

The `?` operator is postfix and works on `Result[T, E]` expressions:

```ailang
expr?
```

**Semantics:**
- If `expr` evaluates to `Ok(v)`, the `?` expression evaluates to `v`
- If `expr` evaluates to `Err(e)`, early-return `Err(e)` from enclosing function

### Enclosing Return Boundary

**Definition:** The `?` operator targets the **nearest surrounding func or lambda** that introduces a return type. Blocks, `if`, `match` arms, and `let` bodies do **not** introduce a return boundary.

**Allowed contexts:**
- Function bodies returning `Result[_, E]`
- Lambda bodies returning `Result[_, E]`
- Blocks, `if`, `match` arms, `let` bodies within such functions

**Disallowed contexts:**
- Functions not returning `Result` (compile error)
- Lambdas not returning `Result` (compile error - `?` does NOT escape lambda)
- Top-level REPL expressions (v1.1: disallowed with clear error)

```ailang
-- OK: ? targets outer func
func outer() -> Result[int, string] =
  let x = {
    let y = foo()? in  -- targets outer, OK
    y + 1
  } in Ok(x)

-- ERROR: ? inside non-Result lambda
func outer() -> Result[int, string] =
  let f = \(). foo()? in  -- ERROR: lambda returns unit, not Result
  f()

-- OK: ? inside Result-returning lambda
func outer() -> Result[int, string] =
  let f = \(). let x = foo()? in Ok(x) in  -- OK: lambda returns Result
  f()
```

### Type Rules

```
expr : Result[T, E]
enclosing return boundary returns Result[_, E]    -- exact match in v1.1
──────────────────────────────────────────────────
expr? : T
```

**v1.1 Constraint:** The operand error type `E` must **exactly match** the enclosing function's error type. No implicit conversion.

**Future (v1.2+):** Support error type conversion via `Into<Error>` trait or sum type construction.

### Evaluation Guarantees

**Single evaluation:** `expr` in `expr?` is evaluated exactly once. The desugaring binds the result to a temporary variable.

**Value preservation:** The `Err(e)` value is preserved without re-boxing. No new allocation occurs on the error path.

These invariants are important for:
- Side-effecting expressions (no duplicate FS/IO calls)
- Performance (no allocation overhead)
- Debugging (error values are identical, not copies)

### Desugaring — whole-body continuation threading (Rev-1)

AILANG's Core IR is expression-only ANF with **no explicit `Return` node** ([internal/core/core.go](internal/core/core.go)). This rules out any purely **local** desugaring of `expr?`:

```ailang
-- ❌ INVALID (Rev-0 "general form" — do not implement):
match expr {
  Ok(v)  => v,        -- type T
  Err(e) => Err(e)    -- type Result[_, E]
}
```

The arms have incompatible types (`T` vs `Result[_, E]`), and even if it typed, the match's value would be consumed by the *surrounding* expression instead of escaping the function — nested in `let x = { ... } in Ok(x)` it would produce `Ok(Err(e))`. **`?` is non-local control flow; a local node replacement cannot express it.**

The correct desugaring is the standard Result-monad / Rust-`?` approach: a **transform over the entire enclosing func/lambda body** (the return boundary defined above) that threads the *continuation* — the rest of the body — into the `Ok` arm. Because the continuation is folded **inside** the match, the match ends up in the function's **tail position**, so the `Err` arm's value *is* the function's return value: genuine early return, and both arms have the function's return type `Result[_, E]`.

The transform `T⟦·⟧` is applied to each func/lambda body that contains at least one `TryExpr`. It has two cooperating parts:

**Step 1 — ANF hoisting (`?`-operand normalization).** Within the body, every `?`-suffixed subexpression is hoisted to a fresh `let`-binding, in left-to-right **evaluation order**. Any non-`?` subexpression that evaluates *before* a hoisted `?` operand is hoisted with it, so evaluation order and single-evaluation are preserved by construction. (This rides the elaborator's existing ANF normalization — `normalizeToAtomic` + binding lists in `internal/elaborate/core.go` — which already performs exactly this kind of hoist for compound subexpressions.)

```ailang
Ok(1)? + Ok(2)?          ⇒  let $try1 = Ok(1)? in let $try2 = Ok(2)? in $try1 + $try2
f(g(), h()?)             ⇒  let $t1 = g() in let $try2 = h()? in f($t1, $try2)
```

**Step 2 — bind fold (T-LET).** After hoisting, every `?` appears only as `let x = E? in K`, where the continuation `K` is *everything remaining* in the enclosing body. Fold:

```ailang
-- T⟦ let x = E? in K ⟧   (E is ?-free after Step 1)
match E {
  Ok(x)     => T⟦K⟧,
  Err($eN)  => Err($eN)
}
```

**Tail-position recursion.** `T` recurses only into positions whose value is the body's value:

| Form | Rule |
|------|------|
| `let x = E in K` (E `?`-free) | `let x = E in T⟦K⟧` |
| `if c then a else b` (tail) | hoist `?`s in `c` above the `if`; then `if c' then T⟦a⟧ else T⟦b⟧` |
| `match s { pᵢ => eᵢ }` (tail) | hoist `?`s in `s`; then arms `pᵢ => T⟦eᵢ⟧` |
| block `{ s1; …; sn }` | statements let-flatten into the continuation: `T⟦let _ = s1 in … in sn⟧` |
| block as **let-RHS** | flattened by the elaborator's existing ANF completion (`internal/elaborate/core.go:309` extracts inner bindings of a `Let` appearing as a let-RHS), which linearizes the block into the outer continuation — see worked example (iii) |
| lambda / nested func | **not descended** — a new return boundary; its body gets its own independent transform |

**Correctness property.** By construction every generated `match` sits on the *spine* of the function body (each enclosing context is a tail position), so the `Err(t) => Err(t)` arm produces the **function's** return value directly. Both arms have type `Result[_, E]` where `E` is the function's error type — no `Ok(Err(e))` wrapping is possible.

**v1 linearity restriction (conservative scope).** The fold requires the continuation `K` to appear exactly once. That holds everywhere except when `?` occurs inside a *branch* (then/else or match arm) of an `if`/`match` that is **not itself in tail position** — there, folding would duplicate `K` per branch (exponential growth) or require synthesizing a join-point lambda. **v1 rejects this case** with a compile error (see Edge Cases) rather than silently duplicating code. `?` in an `if` *condition* or `match` *scrutinee* is fine (it hoists above). Join-point lowering is deferred to v1.2+.

**Worked examples (the objection cases):**

**(i) Let position:**

```ailang
func f() -> Result[int, E] =
  let x = foo()? in
  Ok(x + 1)

-- ⇒ (whole body)
func f() -> Result[int, E] =
  match foo() {
    Ok(x)    => Ok(x + 1),
    Err($e1) => Err($e1)
  }
-- Both arms : Result[int, E] = f's return type. The Err arm IS f's result.
```

**(ii) Operator position — `Ok(1)? + Ok(2)?`:**

```ailang
func g() -> Result[int, E] =
  let x = Ok(1)? + Ok(2)? in
  Ok(x)

-- Step 1 (ANF hoist, left-to-right):
  let $try1 = Ok(1)? in
  let $try2 = Ok(2)? in
  let x = $try1 + $try2 in
  Ok(x)

-- Step 2 (fold, outermost first — continuation nests into each Ok arm):
  match Ok(1) {
    Ok($try1) =>
      match Ok(2) {
        Ok($try2) => let x = $try1 + $try2 in Ok(x),
        Err($e2)  => Err($e2)
      },
    Err($e1) => Err($e1)
  }
-- Every match is on the body spine; every arm : Result[int, E].
```

(Note the invalid local desugar `match Ok(1) { Ok(v) => v, ... }` would have arms `int` vs `Result` — this is exactly what the fold avoids.)

**(iii) Nested block — early return from the FUNCTION, not the block:**

```ailang
func outer() -> Result[int, string] =
  let x = { let y = foo()? in y + 1 } in
  Ok(x)

-- Step 0: let-RHS block flattening (existing ANF completion, core.go:309):
  let y = foo()? in
  let x = y + 1 in
  Ok(x)

-- Step 2 (fold):
  match foo() {
    Ok(y)    => let x = y + 1 in Ok(x),
    Err($e1) => Err($e1)
  }
```

The generated `match` is `outer`'s **entire body**; on `foo() = Err(e)` the function returns `Err(e)` — **not** `Ok(Err(e))`. Contrast the invalid local desugar: the block's value would have been `match foo() { Ok(y) => y + 1, Err(e) => Err(e) }` (arms `int` vs `Result`, ill-typed; and had it typed, `Ok(x)` would wrap the error). The flatten-then-fold order is what makes the block case sound.

Name hygiene during flattening: flattening happens on elaborator-side names after resolution; on collision the elaborator fresh-renames, so lifting `y` past `K` cannot capture.

**(iv) Chain — `a()?.b()?`:**

```ailang
let x = a()?.b()? in K

-- Step 1 (hoist the postfix chain left-to-right):
  let $try1 = a()? in
  let $try2 = $try1.b()? in
  let x = $try2 in K        -- $try2 coalesces with x

-- Step 2 (fold):
  match a() {
    Ok($try1) =>
      match $try1.b() {
        Ok(x)    => K,
        Err($e2) => Err($e2)
      },
    Err($e1) => Err($e1)
  }
```

**Tail `?` note:** `func f() -> Result[int, E] = foo()?` desugars to `match foo() { Ok($t) => $t, Err($e) => Err($e) }` with arms `int` vs `Result[int, E]` — a type error unless `T` is itself `Result[_, E]`. As in Rust, a tail `?` must be wrapped (`Ok(foo()?)`); the diagnostic should hint this.

**Key points:**
- No `Return` construct needed — but only because the continuation is folded into the `Ok` arm, keeping every generated match in the function's tail position. Early return is structural.
- The transform is **per-boundary** (func/lambda body), not per-node. `TryExpr` never reaches Core: elaboration output is ordinary `Match`/`Let` nodes.
- Generated names follow the elaborator's existing fresh-name discipline (`freshVar()` → `$tmpN` in `internal/elaborate/core.go:293`); try-desugar uses a `$tryN` / `$eN` family — `$`-prefixed names are not user-addressable, so hygiene is automatic. (Rev-0's `__try_` prefix did not match repo convention.)
- In emitted Core, match scrutinees must be atomic (ANF) — non-atomic operands get an extra `let $sN = E in match $sN {…}`, which also enforces single evaluation.
- Source spans preserved: each generated node's `CoreNode.OrigSpan` points to the `?` token; a `TryOrigin` side-table (NodeID → `?` span) lets the type checker render `?`-specific diagnostics.

### Postfix Chain Grammar

The `?` operator participates in the **postfix chain** alongside calls and member access:

```
PostfixExpr := Primary { PostfixOp }*

PostfixOp :=
  | '(' Args ')'      -- function call
  | '.' IDENT         -- member access
  | '.' IDENT '(' Args ')'  -- method call
  | '?'               -- try operator
```

**Parse examples:**

| Expression | Parse tree |
|------------|------------|
| `foo()?` | Primary(`foo`) → Call(`()`) → Try(`?`) |
| `foo()?.bar()` | Primary(`foo`) → Call → Try → Member(`.bar`) → Call |
| `a()?.b()?.c()` | Primary(`a`) → Call → Try → Member → Call → Try → Member → Call |
| `foo()? + bar()?` | BinOp(`+`, TryExpr(`foo()`), TryExpr(`bar()`)) |

**Precedence:** `?` binds tighter than all infix operators (`+`, `|>`, etc.) but is part of the postfix chain with calls and member access.

**Note:** AILANG does not have optional chaining (`?.`). The sequence `?` followed by `.` is always Try followed by Member access, never a single token.

### Chained `?` Operators

```ailang
let x = foo()?.bar()? in K
```

Chains are handled by the same two steps: Step 1 hoists each link of the postfix chain to a fresh binding in left-to-right evaluation order, Step 2 folds each binding with the **entire remaining continuation** `K` in its `Ok` arm:

```ailang
match foo() {
  Ok($try1) => match $try1.bar() {
    Ok(x)    => K,        -- K = the rest of the enclosing body, folded in
    Err($e2) => Err($e2)
  },
  Err($e1) => Err($e1)
}
```

Both `Err` arms sit on the body spine, so either failure returns from the enclosing function. See Desugaring worked example (iv).

---

## Conflict Surface

What this feature touches, in pipeline order, and how it interacts with each existing pass. (Pipeline: parse → elaborate to Core ANF → **Core** type check → dictionary elaboration → eval. There is **no** surface-AST type checker — see Types below.)

### Parser (`internal/lexer/`, `internal/parser/`)

- **Lexer: no change.** `QUESTION` already exists and is lexed (`internal/lexer/token.go:92`, `internal/lexer/lexer.go:234`). Rev-0's "add QUESTION token" work items are stale.
- **Parser: `QUESTION` is currently unused** by the Pratt parser (no registration; today `?` in expression position is a parse error), so adding it is conflict-free with the existing grammar. Register it as a high-binding infix/postfix handler alongside `DOT` (registered at `internal/parser/parser.go:131`) at the postfix-chain precedence tier (`CALL` = 15): `p.registerInfix(lexer.QUESTION, p.parseTryExpression)`. No `?.` combined token; `?` then `.` is Try-then-Member (unchanged from Rev-0).
- New `ast.TryExpr` node goes in `internal/ast/ast_expr.go` (expression nodes live there, not `ast.go`); `internal/ast/print.go` needs a case. **File-size note:** `parser_expr.go` is at 798 lines against the 800-line CI cap — the parse function goes in a new small `parser_try.go`.

### Elaborate / Desugar (`internal/elaborate/`)

The whole-body transform is the biggest new surface. There is no `elaborate.go`; the package is split per concern. Touch points:

- **New pass, new file** (`internal/elaborate/try.go`): the per-boundary transform of the Desugaring section (hoist + fold + tail recursion + v1 linearity check). Invoked at each return boundary: `funcToLambda` (`file_funcs.go`) and `normalizeLambda`/`normalizeFuncLit` (`expr_simple.go`). Bodies without `TryExpr` are untouched (zero cost on existing code).
- **Ordering vs existing ANF/let normalization:** the transform *composes with*, not replaces, `normalizeToAtomic` + binding-list discharge (`core.go`). Concretely: extend the elaborator's binding record with a kind (`let` | `try`); a `TryExpr` operand normalizes to a *try-binding*; at discharge, a try-binding wraps the **remaining continuation** in a `Match` `Ok`-arm instead of emitting a `Let`. **Invariant: a try-binding may only be discharged at a spine (function-tail) position** — it floats past non-tail insertion points (if-conditions, call args, let-RHS: all evaluation-order-safe hoists) until it reaches the body spine; if it would have to float out of a *branch* of a non-tail `if`/`match`, elaboration rejects (v1 linearity restriction). The existing ANF completion (let-RHS `Let` flattening, `core.go:309`) runs as today and is what linearizes let-RHS blocks before the fold (worked example iii).
- **Fresh-name discipline:** reuse `freshVar()` (`core.go:293`, emits `$tmpN`) with a `$try`/`$e` family. `$`-names are not user-addressable → hygienic by construction. (Rev-0's `__try_` prefix corrected to match repo convention.)
- **Exhaustiveness** (`internal/elaborate/exhaustiveness.go`): generated matches cover exactly `Ok`/`Err` — exhaustive by construction; the pass must accept them (set `Match.Exhaustive` or let the checker verify — no special case expected, but it is on the surface).
- **Dictionary elaboration** (`internal/elaborate/dictionaries.go`) runs after type checking on ordinary Core; the desugar output is ordinary `Match`/`Let`, so no interaction.

### Types (`internal/types/`)

- **Stale-path correction:** `internal/types/typecheck.go` **does not exist**. Type checking is `types.CoreTypeChecker` over **Core**, post-elaboration (`typechecker.go`, `typechecker_core.go`, `typechecker_functions.go`). Consequently Rev-0's "type check `TryExpr`" phase is impossible as written — **`TryExpr` is eliminated during elaboration and never reaches the checker.**
- **Enforcement model:** the type rules fall out of ordinary unification on the desugared Core: in `match E { Ok(x) => K, Err(t) => Err(t) }` on the body spine, unification forces `E : Result[T, E']`, both arms to the function's return type `Result[_, E'']`, and `E' = E''` — the **exact-E-match rule** (AILANG has no subtyping/conversion, so exact match is what unification gives; nothing extra to implement for the rule itself).
- **Diagnostics are the real types-side work:** raw unification failures at synthesized nodes would be unreadable. Elaboration exports a `TryOrigin` side-table (`map[NodeID] → ?-span + operand info`, avoiding any change to the frozen `CoreNode` struct); `CoreTypeChecker` consults it when a unification error is rooted at a try-generated node and renders TYPE003 (`?` in non-Result boundary) / TYPE004 (error-type mismatch) with the `?` span and the "enclosing function returns X" framing. This replaces Rev-0's `TCContext.EnclosingResultErrorType` — an equivalent context thread lives in the *message renderer* (which walks the checker's current function-return type), not in the rule system. Codes registered in `internal/errors/` (existing IMPxxx/MODxxx convention).
- **Early structural checks at elaboration time** (better errors, pre-typecheck): `?` with no enclosing func/lambda (REPL/top level) → error; enclosing boundary whose *declared* return annotation is present and not `Result[_, _]`-headed → error. Unannotated-lambda cases fall through to the Core-unification path above.
- `internal/types/traverse/` (Core traversal): **no change** — no new Core node exists after desugaring. (Rev-0's ~10 LOC line item removed.)

### Other passes / runtime

- **Eval, effects, link, runtime:** no changes — they see ordinary `Match`/`Let` Core. No effect-row interaction (`?` neither adds nor masks effects; A3 unchanged).
- **REPL:** top-level `?` rejected at elaboration (structural check above) with the existing hint text.
- **Traces/spans:** `OrigSpan` on generated nodes points at `?` (A2 note unchanged).

---

## Implementation Plan

### Phase 1: Parser

**Lexer** (`internal/lexer/`): no change — `QUESTION` already lexed (token.go:92). Add lexer tests only if missing.

**Parser** (`internal/parser/`):
- New `ast.TryExpr{Operand: Expr, Pos: Position}` in `internal/ast/ast_expr.go` (+ `print.go` case)
- Register `QUESTION` at postfix-chain precedence; `parseTryExpression` in new `parser_try.go` (`parser_expr.go` is at the 800-line cap)

### Phase 2: Elaboration (the whole-body transform)

**Elaboration** (`internal/elaborate/`, new file `try.go`):
- Per-boundary transform (hoist + continuation fold) as specified in Desugaring, integrated with `normalizeToAtomic`/binding discharge via try-bindings; spine-only discharge invariant; v1 linearity check
- Structural checks: no enclosing boundary → error; declared non-Result boundary → error
- Fresh names via existing `freshVar()` family (`$tryN`/`$eN`); spans → `OrigSpan` = `?` token; emit `TryOrigin` side-table

### Phase 3: Type-check diagnostics

**Core type checker** (`internal/types/`):
- No new rules — exact-E-match and Result-boundary enforcement fall out of unification on the desugared Core (see Conflict Surface → Types)
- `TryOrigin`-aware rendering of TYPE003/TYPE004 at try-generated nodes; register codes in `internal/errors/`

### Phase 4: Integration & Testing

- REPL: disallow `?` at top level with clear error
- Error message formatting (actionable messages)
- Documentation updates
- Example files

---

## Files to Modify/Create

Table verified against the live repo (Rev-1). Corrections from Rev-0: lexer work already exists (0 LOC); `internal/ast/ast.go` → `ast_expr.go`; `internal/types/typecheck.go` **does not exist** (Core checker files); `internal/elaborate/elaborate.go` **does not exist** (new `try.go`); traverse needs no change (no new Core node).

**New files:**
- `internal/elaborate/try.go` — the whole-body continuation transform
- `internal/parser/parser_try.go` — `parseTryExpression` (`parser_expr.go` is at the 800-line CI cap)

**Modified files:**

| File | Changes | LOC |
|------|---------|-----|
| `internal/lexer/token.go`, `lexer.go` | none — `QUESTION` already lexed (token.go:92, lexer.go:234) | 0 |
| `internal/ast/ast_expr.go` | `TryExpr` definition | ~20 |
| `internal/ast/print.go` | print case for `TryExpr` | ~10 |
| `internal/parser/parser.go` | register `QUESTION` at postfix precedence | ~5 |
| `internal/parser/parser_try.go` (new) | parse `?` in postfix chain | ~30 |
| `internal/elaborate/try.go` (new) | hoist + continuation fold + tail recursion + linearity check + boundary/structural checks | ~300 |
| `internal/elaborate/core.go` | try-binding kind in binding discharge; `$tryN` fresh-name family; `TryOrigin` side-table | ~50 |
| `internal/elaborate/expressions.go`, `expr_simple.go`, `file_funcs.go` | dispatch `ast.TryExpr`; invoke transform at func/lambda boundaries | ~40 |
| `internal/elaborate/exhaustiveness.go` | accept/mark synthesized Ok/Err matches as exhaustive | ~10 |
| `internal/types/typechecker_core.go` (+ helpers) | `TryOrigin`-aware TYPE003/TYPE004 diagnostic rendering | ~80 |
| `internal/errors/` | register TYPE003/TYPE004 codes | ~10 |
| `internal/types/traverse/` | no change — desugar output is existing Core nodes | 0 |

**Estimated total:** ~555 LOC production code, plus ~350–450 LOC tests. (Rev-0's ~235 LOC assumed the invalid local desugar; the whole-body transform is honestly larger — roughly 2.5×.)

---

## Edge Cases

### `?` in Non-Result Context

```ailang
func main() -> int =
  let x = foo()? in  -- Error: main returns int, not Result
  x
```

**Error message:**
```
TYPE003 at file.ail:2:12: cannot use ? operator here
  Expression type: Result[int, Error]
  Enclosing function returns: int
  The ? operator requires the enclosing function to return Result[_, E]
```

### `?` in Non-Result Lambda

```ailang
func outer() -> Result[int, string] =
  let f = \(). foo()? in  -- Error: lambda returns unit
  f()
```

**Error message:**
```
TYPE003 at file.ail:2:17: cannot use ? operator here
  Expression type: Result[int, string]
  Enclosing lambda returns: unit
  Hint: ? does not propagate out of lambdas. Make the lambda return Result.
```

### Multiple Error Types (v1.1: Error)

```ailang
func process() -> Result[int, Error] =
  let a = foo()? in   -- Returns Result[_, FooError]
  let b = bar()? in   -- Returns Result[_, BarError]
  Ok(a + b)
```

**Error message:**
```
TYPE004 at file.ail:2:12: error type mismatch in ? operator
  Expression error type: FooError
  Function error type: Error
  Hint: Use mapErr to convert error types, or use a common error type.
```

**Future (v1.2+):** Support error type unification via:
- `Into<Error>` trait for automatic conversion
- Sum type construction `FooError | BarError`

### Nested in Blocks

```ailang
let x = {
  let y = foo()? in  -- ? targets enclosing func, not block
  y + 1
} in x
```

Blocks don't introduce a return boundary. The `?` still targets the enclosing function. This works because the let-RHS block is **flattened into the outer continuation before the fold** (existing ANF completion), so the generated match wraps the whole rest of the body — see Desugaring worked example (iii) for the full expansion proving the function (not the block) early-returns, with no `Ok(Err(e))` wrapping.

### `?` Inside a Branch of a Non-Tail `if`/`match` (v1.1: Error)

```ailang
func f() -> Result[int, E] =
  let x = (if c then foo()? else 0) in  -- ERROR: ? inside branch of non-tail if
  Ok(x + 1)
```

The fold requires a linear continuation; folding here would duplicate the rest of the body into each branch (or need a join-point lambda). v1.1 rejects with:

```
TYPE005 at file.ail:2:21: cannot use ? inside a branch of an if/match that is not in tail position (v1.1 restriction)
  Hint: bind the branch's Result with let and apply ? afterwards:
    let r = (if c then foo() else Ok(0)) in
    let x = r? in ...
```

`?` in an `if` condition or `match` scrutinee is allowed (it hoists above the conditional). Join-point lowering lifts this restriction in v1.2+.

### `?` in Tail Position

```ailang
func f() -> Result[int, E] = foo()?   -- ERROR unless T = Result[_, E]
```

A tail `?` unwraps to `T` where the function must return `Result[_, E]` — type error via the desugared match arms. Diagnostic hint: wrap the value (`Ok(foo()?)`), as in Rust.

---

## Testing Strategy

### Unit Tests

**Lexer tests** (`internal/lexer/lexer_test.go`):
- `?` tokenizes as QUESTION
- `foo()?.bar()` tokenizes as: IDENT LPAREN RPAREN QUESTION DOT IDENT LPAREN RPAREN (no combined token)

**Parser tests** (`internal/parser/parser_test.go`):
- `foo()?` parses as TryExpr wrapping CallExpr
- `foo()?.bar()` parses as CallExpr(MemberExpr(TryExpr(CallExpr)))
- `foo()? + bar()?` parses with correct precedence (both `?` bind before `+`)

**Elaboration tests** (`internal/elaborate/try_test.go`, new):
- Fold shape: `let x = E? in K` elaborates to a spine `Match` with `K` inside the `Ok` arm
- ANF hoist order: `Ok(1)? + Ok(2)?`, `f(g(), h()?)` — evaluation order preserved
- Let-RHS block flattening before fold (worked example iii shape)
- Linearity restriction: `?` in branch of non-tail `if`/`match` → error
- Structural checks: no enclosing boundary; declared non-Result boundary

**Type checker tests** (`internal/types/` — Core checker; note `typecheck.go`/`typecheck_test.go` do not exist, tests go next to `typechecker_core.go`):
- Ok case: Result function with `?` in body
- Error case: Non-Result function with `?` (TYPE003 rendered at `?` span via TryOrigin)
- Error case: Non-Result lambda with `?`
- Ok case: Result-returning lambda with `?`
- Chained `?` type checking
- Error type mismatch detection (TYPE004, exact-E rule via unification)

### Integration Tests

**New test file:** `tests/error_propagation_test.ail`

```ailang
-- Test: basic ? operator
func testBasic() -> Result[int, string] =
  let x = Ok(42)? in
  Ok(x + 1)
-- Expected: Ok(43)

-- Test: early return on Err
func testEarlyReturn() -> Result[int, string] =
  let x = Err("oops")? in
  Ok(x + 1)  -- Never reached
-- Expected: Err("oops")

-- Test: chained ?
func testChained() -> Result[int, string] =
  let a = Ok(1)? in
  let b = Ok(2)? in
  let c = Ok(3)? in
  Ok(a + b + c)
-- Expected: Ok(6)
```

### Critical "Gotcha" Tests

**1. No double evaluation:**
```ailang
-- Verify expr in expr? is evaluated exactly once
let counter = ref 0

func sideEffect() -> Result[int, string] =
  counter := !counter + 1;
  Ok(!counter)

func testSingleEval() -> Result[int, string] =
  let x = sideEffect()? in
  let y = sideEffect()? in
  Ok(x + y)
-- Expected: Ok(3) with counter == 2, not Ok(6) with counter == 4
```

**2. Precedence with infix operators:**
```ailang
func testPrecedence() -> Result[int, string] =
  let x = Ok(1)? + Ok(2)? in  -- Should parse as (Ok(1)?) + (Ok(2)?)
  Ok(x)
-- Expected: Ok(3)
```

**3. Lambda boundary (error case):**
```ailang
-- This should NOT compile
func testLambdaBoundary() -> Result[int, string] =
  let f = \(). Err("x")? in  -- ERROR: lambda doesn't return Result
  Ok(1)
-- Expected: TYPE003 error
```

**4. No `Ok(Err(e))` wrapping (objection-2 regression):**
```ailang
-- ? inside a let-RHS block must early-return from the FUNCTION
func testBlockEscape() -> Result[int, string] =
  let x = { let y = Err("boom")? in y + 1 } in
  Ok(x)
-- Expected: Err("boom")  -- NOT Ok(Err("boom")), NOT a type error
```

**5. Linearity restriction:**
```ailang
-- This should NOT compile (v1.1)
func testNonTailBranch() -> Result[int, string] =
  let x = (if true then Ok(1)? else 0) in
  Ok(x)
-- Expected: TYPE005 error with restructuring hint
```

**6. Chaining shape:**
```ailang
-- Verify foo()?.bar() is (foo()?).bar() not foo()?.(bar())
func testChainingShape() -> Result[int, string] =
  let r = Ok({ value: 42 }) in
  let x = r?.value in  -- Should access .value on unwrapped record
  Ok(x)
-- Expected: Ok(42)
```

### REPL Testing

**v1.1 rule:** `?` is disallowed at REPL top level.

```
ailang> Ok(42)?
Error: ? operator cannot be used at top level
  The ? operator requires an enclosing function that returns Result.
  Hint: Define a function that returns Result, then call it.
```

### Manual Testing

- Error messages are clear and actionable
- Source positions point to `?`, not synthetic match
- Performance: no measurable overhead vs. explicit match

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Precedence conflicts with future syntax | Medium | Low | Document postfix chain grammar explicitly |
| Lambda boundary confusion | Medium | Medium | Clear error message with hint |
| Whole-body transform bugs (spine invariant violated → `Ok(Err(e))` or ill-typed Core) | High | Medium | Spine-only discharge invariant enforced in code + `Ok(Err)` regression tests; Core type checker backstops any escape |
| v1.1 linearity restriction surprises users | Medium | Medium | Dedicated TYPE005 with restructuring hint; lift via join-points in v1.2+ |
| Diagnostic quality on desugared Core (raw unification errors) | Medium | Medium | `TryOrigin` side-table → TYPE003/TYPE004 rendered at the `?` span |
| Desugaring hygiene (name collision) | Low | Low | Elaborator `freshVar()` `$`-prefix family (`$tryN`/`$eN`) — not user-addressable |
| User confusion (Rust-like but not identical) | Low | Medium | Document differences clearly |
| Source span mapping in errors | Medium | Low | Preserve OrigSpan through elaboration |

---

## Workaround (Current)

Until `?` is implemented, use explicit match or `flatMap`:

```ailang
-- Option 1: Explicit match
match readFile(path) {
  Ok(content) => process(content),
  Err(e) => Err(e)
}

-- Option 2: flatMap from std/result
import std/result (flatMap)

readFile(path) |> flatMap(\c. parseContent(c)) |> flatMap(\d. transform(d))
```

---

## Success Criteria

- [x] `?` lexes as QUESTION token (already true today — token.go:92; no combined `?.`)
- [ ] `foo()?` parses as TryExpr in postfix chain
- [ ] Whole-body continuation fold: every generated match sits on the function-body spine (spine invariant)
- [ ] `?` in a let-RHS block early-returns from the FUNCTION — `testBlockEscape` returns `Err(e)`, never `Ok(Err(e))`
- [ ] Result operand + exact error type match enforced (via Core unification on desugared form, v1.1)
- [ ] Enclosing return boundary tracked through lambdas (new boundary = new transform)
- [ ] v1.1 linearity restriction enforced with TYPE005 + hint
- [ ] Source spans preserved (TYPE003/TYPE004 point to `?` via TryOrigin side-table)
- [ ] Chained `?` works: `a()?.b()?.c()`
- [ ] Lambda boundary enforced (error if lambda doesn't return Result)
- [ ] REPL disallows top-level `?`
- [ ] Single evaluation guaranteed
- [ ] All existing tests pass
- [ ] New test file with gotcha tests
- [ ] Documentation updated
- [ ] Limitations page updated (remove this limitation)
- [ ] Axiom scorecard A11 updated to 2/2

---

## Open Questions

1. **Do-block integration:** Should `?` work in future `do`-block-like constructs? (Defer to v1.2+)

2. **Error conversion:** Prefer `Into<Error>` trait or explicit `mapErr`? (v1.2+ decision)

3. **Trace visibility:** Current plan preserves surface spans. Should traces also show the desugared match for debugging? (Implementation detail)

---

## Related Documents

**Axiom References:**
- [Design Axioms](/docs/references/axioms) - A11: Failure Must Be Representable
- [Axiom Scorecard](docs/static/benchmarks/axiom_scorecard.json) - Current A11 gap

**Implementation References:**
- [std/result.ail](std/result.ail) - Result type definition
- [Limitations](/docs/reference/limitations#error-propagation-operator-) - Current limitation
- [internal/core/core.go](internal/core/core.go) - Core IR (expression-only, no Return)

**Language Inspirations:**
- [Rust ? operator](https://doc.rust-lang.org/book/ch09-02-recoverable-errors-with-result.html)
- [Swift try? operator](https://docs.swift.org/swift-book/LanguageGuide/ErrorHandling.html)

**Future Work:**
- [m-contracts-assert.md](../v0_8_0/m-contracts-assert.md) - Precondition assertions
- Error type conversion (`Into<Error>` or sum types)

---

## Website Links

**Update these when this feature is implemented:**
- [Limitations page](/docs/reference/limitations) — Remove from limitations list
- [Implementation Status](/docs/reference/implementation-status) — Update status
- Move this doc from `planned/` to `implemented/`
- Update axiom scorecard A11 to 2/2

---

**Document created**: 2024 (original)
**Last updated**: 2026-07-23 (**Rev-1** — quorum objections 1 & 2: replaced invalid local-match desugar with whole-body continuation-threading transform; added Conflict Surface; live-verified Files-to-Modify against repo)
**Previous revision**: 2025-12-29 (axiom compliance, semantic precision, feedback integration)
