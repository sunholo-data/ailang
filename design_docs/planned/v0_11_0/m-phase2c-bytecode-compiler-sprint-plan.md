# Sprint Plan: M-PHASE2C-BYTECODE-COMPILER — Statement IR → Bytecode Compiler

## Summary

Build the compiler that lowers Statement IR (`internal/gen/stmt`) into bytecode images runnable by the Phase 2B VM. The acceptance gate from M-BYTECODE-VM §10 Phase 2C is: **`ailang run --bytecode <file>` produces the correct result for all 12 golden tests in [tests/golden/codegen/](../../../tests/golden/codegen/)**.

**Duration:** 2 weeks (~10 working days)
**Dependencies:** Phase 2B (✅ complete — fib(25) gate passed 2026-04-08, commit 01431a13), M-CODEGEN-IR Statement IR (✅ complete, 4,413 LOC, 595 functions)
**Risk Level:** Medium-high — first end-to-end compile path, register allocation is the dominant unknown
**Design Doc:** [m-bytecode-vm.md](m-bytecode-vm.md) §10 Phase 2C, §12 (LOC budget)

## Current Status Analysis

### Completed Recently
- **Phase 2B (M-PHASE2B-BYTECODE)**: opcodes, image format, Value type, VM dispatch loop, fib(25) acceptance test (~1,800 LOC including tests, 1 day under budget)
- **M-CODEGEN-IR**: Statement IR architecture finalized; 360 LOC of clean types in `internal/gen/stmt/stmt.go`

### Velocity
- Recent average: ~300 LOC/day (sustained)
- Phase 2C budget per design doc §12: **~500 LOC compiler + ~300 LOC builtin shims** ≈ **800 LOC** + ~400 LOC tests ≈ **1,200 LOC total**
- At 300 LOC/day that's ~4 productive days; the 2-week window absorbs register-allocator iteration, debugging the long-tail of Statement IR shapes, and golden-test parity work

### Context

Phase 2B proved the VM dispatch loop works on hand-assembled bytecode. Phase 2C closes the loop: real `.ail` source → Core → Statement IR → **bytecode** → VM → result. The 12 golden tests in `tests/golden/codegen/` already exercise the breadth of Statement IR features (literals, arithmetic, control flow, recursion, lists, records, tuples, ADTs, pattern matching) and have known-good outputs verified by the existing Go emitter. Reusing them as the Phase 2C acceptance corpus gives us "one set of inputs, two backends" parity from day one.

### Design Decisions (already made — do not relitigate)

| Decision | Choice | Source |
|----------|--------|--------|
| Compiler input | Statement IR (`internal/gen/stmt`), NOT Core AST | M-BYTECODE-VM §3.5, M-CODEGEN-IR |
| Register allocator | Linear scan with simple liveness on Statement IR; no SSA | §6 |
| Closure capture analysis | Free-variable scan over Lambda body, captures emitted as pseudo-MOVE block after `OpClosure` | §3.3, Phase 2B vm.go closure dispatch |
| ADT tag ordinals | Per-type, alphabetical-by-position-in-source-order, assigned at compile time | §3.6 |
| Boolean short-circuit | `&&`/`\|\|` lowered to conditional jumps, **never** as `OpAnd/OpOr` BinOps (those don't exist as opcodes) | §4.2 |
| Comparison operator set | Only `OpEq`, `OpLt`, `OpLe` exist; rewrite `Neq → NOT Eq`, `Gt → swap+Lt`, `Gte → swap+Le` | Phase 2B vm.go |
| Builtin classification | Pure builtins lowered to `OpBuiltinCall` (in-VM), effectful builtins lowered to `OpBuiltinTrap` (errors in 2C; wired to evaluator in 2E) | §3.4, §3.7 |
| String concat | `OpConcat` opcode (already in Phase 2B), not a `BuiltinCall` | Phase 2B vm.go |
| Pattern matching | Lowered upstream by `internal/gen/lower/match.go` into `SwitchStmt` + `Binding` — compiler just emits `OpGetTag` + jump table + `OpGetField` | §3.5 |
| Effects | Not in scope; effectful examples deferred to Phase 2E | §3.7 |

### Existing Infrastructure (reuse, don't duplicate)
- `internal/gen/stmt/` — IR types (read-only in 2C)
- `internal/gen/lower/` — Core → Stmt lowering (read-only in 2C; consume its output)
- `internal/bytecode/` — opcodes, image, Value (read-write — may need small additions like `RecordTypeIndex` to `FuncPrototype` if record field-order needs to be carried per type)
- `internal/vm/` — dispatch loop (no changes expected; if compiler uncovers VM bugs, fix in place)
- `tests/golden/codegen/` — 12 `.ail` files + `.go.golden` reference outputs (the acceptance corpus)
- `internal/builtins/` — 100+ builtin specs with `IsPure` flag (use this to drive `OpBuiltinCall` vs `OpBuiltinTrap` classification)

### Out of Scope for Phase 2C
- CLI integration / `--bytecode` flag wiring (Phase 2D — but the compiler will be invokable from a test harness)
- Disassembler / pretty-printer (Phase 2D)
- Effect-trap to evaluator bridge (Phase 2E)
- Optimization passes (constant folding, dead-code elimination, peephole) — explicit non-goals
- Multi-module imports across the bytecode boundary (deferred until Phase 2E)
- Performance benchmarking vs evaluator (Phase 2D after disassembler exists)

## Milestones

### M1 — Compiler skeleton + literals + arithmetic + locals (2 days, ~250 LOC)

**Goal:** A `compiler.Compile(prog *stmt.Program) (*bytecode.Image, error)` entry point that handles the four simplest golden tests.

**Deliverables:**
- New package `internal/bytecode/compiler/`
- `compiler.go`: `Compile()`, per-function compilation, `funcCompiler` with register allocator
- `regalloc.go`: simple bump allocator with named-local table (`map[string]uint8`) and a free-list for spillable temps
- `expr.go`: lowering for `LitInt/Float/Bool/String/Unit`, `VarRef` (locals + params), `BinOp(OpAdd/Sub/Mul/Div/Mod)`, `UnOp(OpNeg)`
- `stmt.go`: lowering for `VarDecl`, `ReturnStmt`, function body driver
- Constant pool dedup via existing `Image.AddConstant()`

**Acceptance:**
- `literals.ail` compiles and all 5 functions return correct values via the VM
- `arithmetic.ail` compiles; `addInts`, `mulFloats`, `negate` work
- `let_bindings.ail` compiles; `nested(5)` and `withRebind(5)` return correct values
- New unit tests in `compiler_test.go` covering each `Expr` and `Stmt` variant added

**Risks:**
- Register allocator scope creep — keep it dumb. No graph coloring, no liveness intervals; just "allocate next free, free at scope end". Spilling is out of scope for 2C.

### M2 — Control flow: if/else, comparison, boolean short-circuit (1.5 days, ~150 LOC)

**Goal:** Conditional expressions and statements compile correctly, with proper short-circuit semantics.

**Deliverables:**
- `IfExpr` and `IfStmt` lowering — emit forward jumps, patch offsets after compiling each branch
- `BinOp(OpEq/Neq/Lt/Lte/Gt/Gte)` with the comparison rewrites from §Design Decisions
- `BinOp(OpAnd)` lowered to: `eval lhs; jump-if-false to false_label; eval rhs; jump end_label; false_label: load false; end_label:`
- `BinOp(OpOr)` symmetric with `jump-if-true` (synthesized via `JumpIfFalse` over a `Jump`)
- `UnOp(OpNot)` → `OpNot`
- A `jumpPatcher` helper to fix up forward `OpJump`/`OpJumpIfFalse` operands once the target IP is known

**Acceptance:**
- `if_else.ail`: `abs`, `classify`, `clamp` all return correct values for representative inputs
- `arithmetic.ail`: `comparison` and `logical` work
- Unit test exercising short-circuit: `false && (1/0)` must not divide

**Risks:**
- Off-by-one in jump offsets is the most common bytecode-compiler bug. Add a `compileTest_jumpOffsets` golden suite that compiles small fragments and asserts the exact instruction stream.

### M3 — Function calls, recursion, factorial parity (1.5 days, ~150 LOC)

**Goal:** Top-level functions can call each other and recurse. `factorial(10)` matches the Go emitter.

**Deliverables:**
- `Call` lowering: evaluate args into contiguous registers, emit `OpClosure` for the callee (top-level functions are looked up via a `funcIndex map[string]int`), emit `OpCall` with arity
- `GlobalRef` lowering — same as a function reference for now (top-level only; no globals-as-values yet in 2C)
- Tail call detection: if a `ReturnStmt` returns a `Call` whose callee is a `GlobalRef`/`VarRef` of a known function and parameter shapes match, emit `OpTailCall` instead of `OpCall + OpReturn`
- Function prototype registration phase before bodies are compiled, so forward and mutual recursion work

**Acceptance:**
- `functions.ail`: `identity`, `apply`, `factorial`, `compose` all work
- `factorial(10) = 3628800` matches Go emitter
- VM stack-overflow test: a deliberately non-tail recursive function blows the stack at the configured `MaxStack`, but a tail-recursive equivalent runs unbounded (confirming TCO is wired)

**Risks:**
- `apply(f, x)` requires first-class functions — `f` is a parameter (a `Closure` value at runtime). The compiler must treat `Call` whose `Func` is a `VarRef` to a parameter as "load the closure register, `OpCall` it" — not as a `OpClosure` lookup. Plan a clear branch in `compileCall`.

### M4 — Lambdas, free-variable analysis, closures (2 days, ~200 LOC)

**Goal:** `Lambda` expressions in Statement IR are hoisted into nested `FuncPrototype`s with proper capture handling.

**Deliverables:**
- `lambda.go`: free-variable scan over `Lambda.Body`/`Lambda.Return`, producing an ordered capture list
- Lambda hoisting: each `Lambda` becomes a child `FuncPrototype` registered under the enclosing function's `NestedProtos`; captures become the lambda's first N register parameters (loaded from the closure object via `OpGetCapture`-equivalent — or just laid out so the VM's existing capture-loading does the right thing)
- `OpClosure` emission followed by N pseudo-`OpMove` instructions (one per capture) — matching the Phase 2B `vm.go` closure-capture dispatch path
- Verify the existing VM correctly reads the captures back; if not, that's a Phase 2B bug to fix in place

**Acceptance:**
- A unit test compiling a hand-written `Lambda` with 2 captures, calling it from the enclosing function, and asserting the captured values are visible
- A higher-order test: `apply(\x. x + k, 5)` where `k` is a captured local

**Risks:**
- The Statement IR `Lambda` emitted by `lower/` may not preserve enough type info to size the capture list cleanly. If so, derive capture types from the enclosing function's local table.
- Verify the existing VM's `OpClosure` dispatch handles `NumCaptures > 0` correctly — Phase 2B added the field but the fib test only exercised `NumCaptures == 0`.

### M5 — Collections, ADTs, pattern matching (2 days, ~200 LOC)

**Goal:** Records, tuples, lists, and ADT constructors + pattern matches all compile through to working bytecode.

**Deliverables:**
- `RecordLit` → `OpMakeRecord` with alphabetically-sorted field name table (matches `bytecode.NewRecord` invariant)
- `RecordUpdate` → `OpMakeRecord` from base + overrides (emit a copy followed by per-field overwrites; or a dedicated path if we add `OpUpdateRecord` later)
- `FieldAccess` → `OpGetField` with field-name index
- `TupleLit` → `OpMakeTuple`
- `ListLit` → `OpMakeList`; `Cons` → `OpCons`
- `ADTConstructor` → `OpMakeAdt` with tag ordinal from a per-type tag table built during type-decl pass
- `SwitchStmt` lowering: `OpGetTag scrutinee_reg → tag_reg`, then a chain of `OpEq + OpJumpIfFalse` per case (a flat decision list, not a jump table — the corpus has small case counts so this is fine)
- `Binding` extraction: `OpGetField` for each binding before entering the case body, allocating fresh registers for each binding

**Acceptance:**
- All 4 remaining golden tests work: `lists.ail`, `records.ail`, `tuples.ail`, `match_patterns.ail`, `adt_simple.ail`, `adt_multiarg.ail`, `string_ops.ail`
- ADT tag ordering: `Color = Red | Green | Blue` assigns Red=0, Green=1, Blue=2 (source-order, not alphabetical, since the tag is private to the type)
- Pattern with literal scrutinee in `match_patterns.ail`: `Num(0) => true` correctly compiles to a tag-check followed by a field-equality check

**Risks:**
- `match_patterns.ail` mixes tag matching with literal sub-patterns (`Num(0)`). Verify the upstream `lower/match.go` already lowers literal sub-patterns into nested `IfStmt` over `Binding`s — if not, the compiler has to do that work and the milestone grows.
- Record field name → index mapping needs to be stable across the program. Build a `recordTypeIndex map[string][]string` during the type-decl pass.

### M6 — Builtins + golden test parity gate (1 day, ~150 LOC)

**Goal:** Pure builtins called from the corpus work via `OpBuiltinCall`; effectful builtins emit `OpBuiltinTrap` with a clear "not yet wired" error. Acceptance gate runs.

**Deliverables:**
- `builtins.go`: a `compileBuiltinCall` function that consults `builtins.Registry` for `IsPure` and emits the right opcode
- A small `bytecodeBuiltinTable` mapping the names of all *pure* builtins reachable from the corpus to per-builtin handler indices
- `internal/vm/builtins.go`: VM-side dispatch table for `OpBuiltinCall` covering exactly the builtins the corpus touches (likely just `_str_concat` if anything — most golden examples use opcodes already, not builtins)
- Acceptance test: a new `tests/golden/bytecode/golden_test.go` that loads each `.ail` file in `tests/golden/codegen/`, compiles to bytecode, runs through the VM, and asserts the output matches a hand-computed reference for the test's exported entry points (not the `.go.golden` source — that's Go code, not values)

**Acceptance — THE GATE:**
- All 12 golden tests compile to bytecode with no errors
- For each test's exported functions called with representative inputs, the bytecode VM result equals the reference value
- `go test ./internal/bytecode/compiler/... ./tests/golden/bytecode/...` is green
- `go vet`, `golangci-lint run`, and race detector all pass on the new packages

**Risks:**
- Discovering a Statement IR shape the compiler doesn't handle. Mitigation: M1-M5 should already exhaust the `Expr`/`Stmt` variants used by the corpus; this milestone is mostly integration. If a new shape appears, it goes on the backlog and the corpus is reduced (with a documented exception) rather than expanding scope.

## Total LOC Estimate

| Milestone | Compiler LOC | Test LOC |
|-----------|-------------:|---------:|
| M1 — skeleton + literals + arithmetic | 250 | 100 |
| M2 — control flow | 150 | 80 |
| M3 — calls + recursion | 150 | 60 |
| M4 — lambdas + closures | 200 | 80 |
| M5 — collections + ADTs | 200 | 100 |
| M6 — builtins + gate | 150 | 150 |
| **Total** | **~1,100** | **~570** |

This is ~300 LOC over the design doc §12 budget (800), absorbed by the explicit lambda/closure milestone and the golden harness.

## Open Questions (resolve in M1)

1. **`compileTest_jumpOffsets` golden format:** raw byte-stream golden, or a per-instruction `String()` golden? Recommend the latter — it survives benign Image.Validate changes.
2. **Register pressure on the corpus:** a quick survey of `match_patterns.ail` and `factorial` should confirm that 8-bit register fields are not a constraint at the corpus scale. If any function needs >256 regs, we have a much bigger problem and should stop and discuss.
3. **Where does `bytecode.Image.Validate` live in the compile pipeline?** Recommend: call it at the end of `Compile()` and treat any failure as a compiler bug, not a user error.

## Verification Commands

```bash
# Build & test new packages
go build ./internal/bytecode/compiler/...
go test -race ./internal/bytecode/compiler/... ./internal/vm/... ./internal/bytecode/...

# Acceptance gate
go test -race -run TestGoldenBytecode ./tests/golden/bytecode/...

# Lint + file size
make lint
make check-file-sizes
```

## Exit Criteria for Phase 2C → Phase 2D

- [ ] All 12 golden corpus tests pass via the bytecode VM
- [ ] Unit tests cover every `stmt.Stmt` and `stmt.Expr` variant the corpus exercises
- [ ] No new public API in `internal/bytecode/` or `internal/vm/` other than the compiler entry point and the builtin dispatch table
- [ ] Compiler entry point documented for Phase 2D CLI integration
- [ ] CHANGELOG entry under `[Unreleased]` describing the compiler and the gate result
- [ ] Design doc M-BYTECODE-VM §10 Phase 2C marked complete with deviations noted
