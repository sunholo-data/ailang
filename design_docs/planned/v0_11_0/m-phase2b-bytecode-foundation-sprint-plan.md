# Sprint Plan: M-PHASE2B-BYTECODE — Bytecode VM Foundation

## Summary

Build the foundation of the AILANG bytecode virtual machine: opcode encoding, bytecode image format, tagged Value type, and the core register-VM dispatch loop. This is Phase 2B of the M-BYTECODE-VM design doc. Scope is **hand-written bytecode only** — the Statement IR → bytecode compiler is Phase 2C and is explicitly out of scope.

**Duration:** 2 weeks (~10 working days)
**Dependencies:** Phase 2A benchmarks (✅ complete — gate passed 2026-04-03), M-CODEGEN-IR Statement IR (✅ complete, 4,413 LOC)
**Risk Level:** Medium — new subsystem, but isolated from existing code (no evaluator or pipeline changes)
**Design Doc:** [m-bytecode-vm.md](m-bytecode-vm.md) §10 Phase 2B
**Gate Result:** [phase2a-results.md](phase2a-results.md) — all 7 workloads failed 10x threshold, "miss across the board", build with high priority

## Current Status Analysis

### Completed Recently
- **Phase 2A (M-PHASE2A-BENCH)**: 7 benchmark workloads + native Go baselines + decision report (3 days, on plan)
- **M-CODEGEN-IR (M1-M5)**: Statement IR architecture, 4,413 LOC, 595 functions compile to gofmt-clean Go
- **M-BYTECODE-VM design doc**: reviewed and finalized with semantic contracts (§3.6, §3.7)

### Velocity
- Recent average: ~300 LOC/day (sustained across sprints)
- Phase 2B budget per design doc §12: **~950 LOC** (150 opcode + 200 image + 200 value + 400 vm) + ~300 LOC tests ≈ **1,250 LOC**
- At 300 LOC/day that's ~4 productive days; the 2-week window absorbs debugging, design iteration, and the hand-assembled test corpus

### Context

Phase 2A confirmed the evaluator is 400x-441,000x slower than native Go across every workload. The bytecode VM is now the highest-priority runtime performance initiative. Phase 2B establishes the execution substrate — a register VM that can run hand-written bytecode images — without yet touching the compiler. This isolates risk: if the VM design has fundamental issues, we discover them against trivial hand-assembled programs before investing in the Statement IR compiler (Phase 2C).

### Design Decisions (already made — do not relitigate)

| Decision | Choice | Source |
|----------|--------|--------|
| VM style | Register-based (not stack) | §3.1 |
| Value representation | Tagged Go struct first; NaN-boxing deferred to Phase 2D only if benchmarks show >20% in value dispatch | §3.2 |
| Closure capture | Flat (copy captures at creation) | §3.3 |
| Effects | Not in opcode set — trap to evaluator via EFFECT_TRAP | §3.4, §3.7 |
| Pattern matching | No match opcodes — Statement IR has already lowered patterns | §3.5 |
| Instruction width | 32-bit word: 8-bit opcode + A/B/C or A/Bx | §4.1 |
| Opcode count | ~30 opcodes | §4.2 |

### Existing Infrastructure (reuse, don't duplicate)
- `internal/gen/stmt/` — Statement IR (not touched in 2B, but 2B types must be compatible)
- `internal/eval/` — evaluator remains canonical semantic authority; will be imported by VM only via interface for EFFECT_TRAP / BUILTIN_TRAP in later phases
- `internal/builtins/` — builtin specs include `IsPure` flag used later by BUILTIN_CALL vs BUILTIN_TRAP classification (not wired in 2B)

### Out of Scope for Phase 2B
- Statement IR → bytecode compiler (Phase 2C)
- Register allocation (Phase 2C)
- CLI integration / `--bytecode` flag (Phase 2D)
- Disassembler (Phase 2D)
- Effect boundary / EFFECT_TRAP execution (Phase 2E)
- NaN-boxed values (Phase 2D optimization, gated on benchmark evidence)
- Bytecode serialization to disk (deferred — in-memory only for 2B)

## Proposed Milestones

### Milestone 1: Opcodes + Bytecode Image
**Goal:** Define the instruction encoding, opcode set, and in-memory bytecode image format.
**Estimated:** ~350 LOC (150 opcode + 200 image)
**Duration:** 2 days

**Tasks:**
1. Create `internal/bytecode/opcode.go`:
   - `OpCode` uint8 enum with all ~30 opcodes from §4.2 (LOAD_CONST, MOVE, ADD/SUB/MUL/DIV/MOD/NEG, EQ/LT/LE, NOT, CONCAT, JUMP/JUMP_IF_FALSE, CALL/TAIL_CALL/RETURN, CLOSURE, MAKE_LIST/MAKE_TUPLE/MAKE_RECORD/CONS, GET_FIELD/GET_INDEX, MAKE_ADT/GET_TAG, BUILTIN_CALL, BUILTIN_TRAP, EFFECT_TRAP, LOAD_NIL, LOAD_GLOBAL)
   - `Instruction` = uint32 with encode/decode helpers: `EncodeABC(op, a, b, c)`, `EncodeABx(op, a, bx)`, `(i Instruction).Op()`, `.A()`, `.B()`, `.C()`, `.Bx()`, `.SBx()` (signed wide for jumps)
   - `String()` method on OpCode for debug output
   - Unit tests round-tripping encode/decode for every opcode

2. Create `internal/bytecode/image.go`:
   - `BytecodeImage` struct: `Prototypes []FuncPrototype`, `Constants []Value` (forward-ref to vm.Value via interface or late binding — see note), `EntryPoint int`
   - `FuncPrototype` struct: `Name string`, `NumRegs uint8`, `NumParams uint8`, `Instructions []Instruction`, `Constants []int` (indices into image constant pool), `NestedProtos []int` (for CLOSURE), `LineInfo []int` (instruction index → source line, for errors), `IsVariadic bool`
   - Basic constructor helpers: `NewImage()`, `(*BytecodeImage).AddPrototype(p FuncPrototype) int`, `.AddConstant(v Value) int` (dedup by equality)
   - No serialization yet — in-memory only

**Note on circular imports — RESOLVED:** §11 sketches the file layout as `internal/vm/value.go`, but bytecode's constant pool needs to hold values, which would create a `bytecode → vm` import. The cleanest fix is to put `Value` in `internal/bytecode/value.go` (Milestone 2). This preserves §11's actual import *direction* (`vm → bytecode`, never the reverse) and only relocates the file. `internal/runtime/` is **not** an option — it already exists as the module runtime and imports the evaluator, which would defeat the isolation goal.

**Acceptance Criteria:**
- [ ] All ~30 opcodes defined with stable numeric values
- [ ] Encode/decode round-trip tests pass for every opcode variant (ABC and ABx forms)
- [ ] Signed wide operand (SBx) correctly handles negative jump offsets
- [ ] `BytecodeImage` and `FuncPrototype` types compile and have constructor coverage
- [ ] `make test` passes; new code has unit tests; linting clean
- [ ] Import graph respects §11 (or documented deviation via `internal/runtime/`)

---

### Milestone 2: Tagged Value Type
**Goal:** Implement the Phase 2B tagged-struct `Value` type per §3.2, with primitive unboxing and heap object references.
**Estimated:** ~200 LOC
**Duration:** 1 day

**Tasks:**
1. Create `internal/bytecode/value.go` (resolved in M1 — see import-graph note above):
   - `ValueTag` uint8 enum: `TagInt`, `TagFloat`, `TagBool`, `TagUnit`, `TagString`, `TagList`, `TagTuple`, `TagRecord`, `TagClosure`, `TagADT`
   - `Value` struct exactly as §3.2:
     ```go
     type Value struct {
         Tag  ValueTag
         Int  int64
         Flt  float64
         Bool bool
         Obj  any
     }
     ```
   - Constructor helpers: `NewInt(int64)`, `NewFloat(float64)`, `NewBool(bool)`, `Unit()`, `NewString(string)`, `NewList([]Value)`, `NewTuple([]Value)`, `NewRecord(fields []RecordField)`, `NewADT(tag int, fields []Value)`, `NewClosure(proto *FuncPrototype, captures []Value)`
   - Heap object types: `StringObj`, `ListObj` (cons cell or slice-backed — **choose slice-backed for 2B**, revisit in 2D if benchmarks demand), `TupleObj`, `RecordObj` (fields sorted alphabetically per §4.3), `ClosureObj`, `ADTObj`
   - `(v Value) String()` for debugging
   - `(v Value) Equal(other Value) bool` implementing canonical structural equality per §3.6

2. Write unit tests covering:
   - Every constructor produces a value with the right tag
   - Structural equality: `Equal` is reflexive, symmetric, and correctly distinguishes records with different field orderings (must match alphabetical normalization)
   - Float NaN handling matches evaluator semantics (NaN != NaN at the comparison operator level, but `Equal` must be defined for deduplication)

**Acceptance Criteria:**
- [ ] All 10 ValueTag variants have constructors and test coverage
- [ ] RecordObj enforces alphabetical field ordering on construction
- [ ] Equal() is consistent with evaluator's structural equality (spot-checked against evaluator output on 5 shared inputs)
- [ ] `make test` passes; linting clean

---

### Milestone 3: VM Core Dispatch Loop
**Goal:** Implement the register VM — frames, dispatch, and execution of all ~30 opcodes except the TRAP variants.
**Estimated:** ~400 LOC
**Duration:** 4 days

**Tasks:**
1. Create `internal/vm/frame.go`:
   - `Frame` struct: `Proto *FuncPrototype`, `IP int`, `Regs []Value` (sized to `Proto.NumRegs`), `ReturnReg uint8` (where caller wants the result), `Caller *Frame`
   - Frame pool (sync.Pool) for reuse to avoid per-call allocation — keep simple, profile before optimizing

2. Create `internal/vm/vm.go`:
   - `VM` struct: `Image *BytecodeImage`, `Globals []Value`, `Stack []*Frame`, `MaxStack int`
   - `NewVM(img *BytecodeImage) *VM`
   - `(*VM).Run(entryProto *FuncPrototype, args []Value) (Value, error)` — main entry
   - Core dispatch loop: classic `for { switch op := frame.Proto.Instructions[frame.IP].Op(); op { ... } }` — a computed-goto trick isn't available in Go, so the switch is the canonical form
   - Implement every non-TRAP opcode from §4.2:
     - **Loads:** LOAD_CONST, LOAD_NIL, MOVE, LOAD_GLOBAL
     - **Arith:** ADD/SUB/MUL/DIV/MOD/NEG with int/float dispatch by tag
     - **Compare:** EQ/LT/LE
     - **Logic:** NOT (AND/OR handled by compiler as jumps, no opcode)
     - **String:** CONCAT
     - **Control:** JUMP, JUMP_IF_FALSE, CALL, TAIL_CALL, RETURN
     - **Closures:** CLOSURE (reads following MOVEs for captures per §4.2)
     - **Collections:** MAKE_LIST, MAKE_TUPLE, MAKE_RECORD, CONS, GET_FIELD, GET_INDEX
     - **ADT:** MAKE_ADT, GET_TAG
   - **Stubs for TRAPs:** BUILTIN_CALL / BUILTIN_TRAP / EFFECT_TRAP return a clear "not implemented in Phase 2B" error. They are wired in Phase 2C/2E.
   - Tail call optimization: TAIL_CALL reuses the current frame's register slab (resize if callee has different NumRegs), preserving the semantic contract that divergence behavior matches the evaluator (§3.6)
   - Error type: `VMError` with source location from `FuncPrototype.LineInfo`
   - Stack overflow detection via `MaxStack` (default: 1000 frames, matching evaluator recursion limit)

3. Hand-written micro-tests in `internal/vm/vm_test.go` — one per opcode group:
   - `TestVM_Arith` — hand-assembled ADD/SUB/MUL/DIV with constants
   - `TestVM_Compare` — EQ/LT/LE on ints and floats
   - `TestVM_Jumps` — forward and backward JUMP, JUMP_IF_FALSE
   - `TestVM_Call` — CALL a trivial function, check return value in caller reg
   - `TestVM_TailCall` — TAIL_CALL that would otherwise blow the stack (e.g., 10,000-iter countdown); test passes if frame count stays bounded
   - `TestVM_Closure` — create a closure with captures, call it, verify capture values
   - `TestVM_List` — MAKE_LIST / CONS / GET_INDEX
   - `TestVM_Record` — MAKE_RECORD with alphabetical ordering / GET_FIELD
   - `TestVM_ADT` — MAKE_ADT / GET_TAG / switch-dispatch pattern via JUMP_IF_FALSE chain

**Acceptance Criteria:**
- [ ] Every non-TRAP opcode has at least one test exercising it
- [ ] TAIL_CALL test completes without stack growth (verified by checking `len(vm.Stack)` before/after)
- [ ] Stack overflow is detected and returns `VMError` (not Go panic)
- [ ] Source locations from LineInfo appear in error messages
- [ ] `make test` passes; `go test -race ./internal/vm/...` clean; linting clean
- [ ] No imports from `internal/eval/`, `internal/gen/stmt/`, `internal/lower/`, `internal/core/` (§11 enforced)

---

### Milestone 4: fib(25) Hand-Assembled Golden Test
**Goal:** Deliver the Phase 2B acceptance criterion — hand-written bytecode for `fib(25)` runs correctly on the VM.
**Estimated:** ~150 LOC test + ~50 LOC assembly helper
**Duration:** 1 day

**Tasks:**
1. Create `internal/vm/assemble_test.go` — a small test-only assembler helper:
   - `asm(lines ...string) *FuncPrototype` — parses a minimal text form like `LOAD_CONST r0, #0` into `Instruction`s + constant pool. Keep it dumb: regex or hand-tokenized; exists only to keep the test readable.
   - Alternative if assembler is too much: build `FuncPrototype` directly with a `[]Instruction{EncodeABx(OpLoadConst, 0, 0), ...}` literal. **Pick whichever is less code.**

2. Write `TestVM_Fib25` in `internal/vm/vm_fib_test.go`:
   - Hand-assemble the recursive fib function:
     ```
     fib(n) = if n < 2 then n else fib(n-1) + fib(n-2)
     ```
   - Expected: `fib(25) == 75025`
   - Include a correctness assertion plus a timing floor check (must complete in < 1s — sanity check, not a perf claim)

3. Write `TestVM_FibTailCallAccumulator` — tail-recursive fib using accumulator pair to exercise TAIL_CALL under non-trivial control flow:
   - `fibTail(n, a, b) = if n == 0 then a else fibTail(n-1, b, a+b)`
   - `fibTail(25, 0, 1) == 75025`
   - Verify frame count stays at 1 (TCO working)

4. Update [m-bytecode-vm.md](m-bytecode-vm.md) §10 Phase 2B to mark all checkboxes complete and link to this milestone's commit.

**Acceptance Criteria:**
- [ ] `TestVM_Fib25` passes — hand-assembled recursive fib(25) returns 75025
- [ ] `TestVM_FibTailCallAccumulator` passes with bounded frame count (TCO verified)
- [ ] Test output includes the raw bytecode listing for documentation purposes
- [ ] m-bytecode-vm.md §10 Phase 2B checkboxes all marked complete
- [ ] CHANGELOG entry added under v0.11.0 unreleased section

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Circular import between `bytecode/` and `vm/` (Value type) | High | Medium | Resolve in M1 via `internal/runtime/value.go` shared package; document the §11 deviation |
| TAIL_CALL semantics subtly wrong (stack growth in some path) | Medium | High | Dedicated frame-count test in M3 and M4; fail loudly on growth |
| Record alphabetical ordering drift vs evaluator (§4.3, §3.6) | Medium | High | Shared equality test against evaluator output on 5 inputs in M2 |
| Opcode set incomplete for fib (missed something) | Low | Medium | M4 is explicitly the integration gate; if missing, scope a follow-up |
| Scope creep into Phase 2C (compiler work) | Medium | Medium | Hard rule: no `gen/stmt/` imports in `vm/` or `bytecode/` during 2B |
| Tagged struct Value performance surprises | Low | Low | Phase 2B is correctness-first; perf gate is Phase 2D |

## Success Metrics
- Phase 2B deliverables match §10 checklist exactly (5 files, all tests)
- `TestVM_Fib25` passes — design doc acceptance criterion met
- Zero imports from evaluator / pipeline / Statement IR in new packages
- VM is fully exercised by hand-assembled tests — no dependency on a compiler for validation
- ~1,250 LOC total, within the design doc budget
- All existing tests still pass; no regression in evaluator or pipeline

## What Phase 2B Does NOT Deliver
- Ability to run any `.ail` file through the VM (that's 2C)
- Performance numbers vs evaluator (that's 2D after compiler exists)
- Effect handling (that's 2E)
- `ailang run --bytecode` CLI flag (that's 2D)

The only way to exercise the VM after Phase 2B is hand-assembled test bytecode. This is intentional — it isolates VM correctness from compiler correctness.

## Next Sprint Preview: Phase 2C (Statement IR Compiler)
Once Phase 2B lands, Phase 2C is the Statement IR → bytecode compiler (~500 LOC per §12). Acceptance: `ailang run --bytecode fib.ail` produces correct result for all 12 golden test inputs. Estimated 2 weeks.
