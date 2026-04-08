# M-BYTECODE-VM: Bytecode Virtual Machine from Statement IR

**Status**: Phase 2A COMPLETE. Phase 2B COMPLETE (2026-04-08, fib(25) gate passed). Phase 2C COMPLETE (2026-04-08, golden parity 33/0). **Phase 2D COMPLETE** (2026-04-08): `--bytecode` CLI flag, disassembler, line-info plumbing, evaluator fallback via per-function EvalOnly tagging + value bridge, benchmark harness (bytecode VM ~26x faster than evaluator on fib(25), ~55x fewer allocs), and full-corpus parity gate at **134/141 (95%)** of `examples/runnable/` — effectively 100% on eligible examples (the remaining 7 are 1 non-deterministic `uuid.ail` + 5 AI-keyed + 1 intentional `exit(42)`). See [m-bytecode-2d-bench.md](m-bytecode-2d-bench.md) and [m-bytecode-2d-parity.md](m-bytecode-2d-parity.md) for the detailed reports.
**Target**: v0.11.0+
**Priority**: P1 (Strategic — implements the chosen compilation architecture)
**Created**: 2026-04-03
**Reporter**: Design committee consensus (M-CODEGEN-STRATEGIC-REVIEW §15)
**Supersedes**: M-PERF4 (bytecode-interpreter, v1.0.0 stretch goal — now promoted and redesigned)
**Depends on**: M-CODEGEN-STRATEGIC-REVIEW Phase 1 (Statement IR) — **COMPLETE**

---

## TL;DR

**Context**: The Statement IR pipeline is proven (4,413 LOC, 595 functions compile to gofmt-clean Go). The design committee concluded that full Go source emission is structurally misaligned with AILANG's semantics. The recommended architecture is **evaluator-first + bytecode VM + Go host binary** (Options G+D from the strategic review).

**This document**: Designs a bytecode VM that compiles from Statement IR, hosted inside a Go binary. The evaluator remains the canonical semantic authority. The bytecode VM targets performance-critical paths where the tree-walking evaluator is insufficient.

**Key difference from M-PERF4**: M-PERF4 proposed compiling Core AST directly to bytecode (bypassing any IR). This design compiles **Statement IR** to bytecode, reusing all lowering passes already built. The Statement IR is the single compilation boundary — bytecode is just another emitter.

**Gate**: ✅ **PASSED** (2026-04-03). Phase 2A benchmarks confirmed the evaluator exceeds the 10x native-Go threshold on all 7 workloads, most by 3-5 orders of magnitude (fib30: ~6,900x, map/filter: ~191,000x, closures: ~441,000x, game step: ~127,000x / 149x over 16ms frame budget). This is a "miss across the board" result — bytecode VM is required with **higher priority**. See [phase2a-results.md](phase2a-results.md).

**Core principle**: The VM is not a new semantic layer; it is an execution projection over Statement IR with evaluator-checked equivalence.

---

## 1. Architecture: Three-Tier Execution Model

The strategic review established a three-tier architecture. This document implements Tier B.

```
                    AILANG Source
                         │
                    Parse → Elaborate → Typecheck → Monomorphize
                         │
                      Core AST + CoreTypeInfo
                         │
              ┌──────────┤──────────────────────────┐
              │          │                           │
         [Tier A]   [Statement IR]              [Tier A]
         Evaluator       │                      Evaluator
         (canonical)     │                      (REPL, debug,
              │     ┌────┴────┐                  rare features)
              │     │         │
              │  [Tier B]  [Tier C]
              │  Bytecode  Go Emitter
              │  Compiler  (debug/API only)
              │     │
              │  Bytecode Image
              │     │
              │  [Tier B]
              │  Bytecode VM
              │  (hosted in Go binary)
              │     │
              └─────┴──→ Single Go Binary
```

### Tier A: Evaluator (Canonical)

- **Semantic authority**: The evaluator defines what AILANG programs mean
- **Use for**: REPL, tooling, debugging, full language coverage, fallback
- **Unchanged**: No modifications to `internal/eval/`

### Tier B: Bytecode VM (This Document)

- **Performance path**: For hot loops, game logic, server handlers
- **Compiles from**: Statement IR (not Core AST)
- **Must be semantically equivalent**: Every bytecode program must produce the same result as the evaluator
- **Hosted in**: Go binary (the VM is a Go library)

### Tier C: Go Source Emitter (Existing, Demoted)

- **Debug/inspection artifact**: Readable projection for diagnostics
- **API surface generation**: Thin Go wrappers for host interop
- **Not a performance path**: Source emission is not a production compilation strategy

---

## 2. Why Statement IR → Bytecode (Not Core → Bytecode)

M-PERF4 proposed compiling Core AST directly to bytecode. This was reasonable pre-Statement-IR but is now suboptimal:

| Approach | Pros | Cons |
|----------|------|------|
| Core → Bytecode | Simpler pipeline | Duplicates lowering work; match/block/type lowering redone in bytecode compiler |
| **Statement IR → Bytecode** | Reuses all lowering passes; bytecode compiler is thin | Extra IR layer |

The Statement IR has already solved:
- **Match lowering**: 7 pattern types → if/switch decision trees
- **Block lowering**: Let-chain flattening, nested Let/LetRec extraction
- **Type projection**: AILANG types → resolved concrete types
- **DictApp dispatch**: Type class method resolution (Num/Eq/Ord/Show)
- **Function qualification**: Intra-module name resolution (centralizes qualification even if residual projection bugs remain)

The bytecode compiler just walks Statement IR and emits instructions. It does not need to understand Core patterns, type classes, or let-chains. This is the "thin emitter" principle from the strategic review.

```
                  ALREADY BUILT (4,413 LOC)           NEW (~1,500 LOC)
                  ─────────────────────────           ────────────────
Core AST ──→ Match Lowering                     Statement IR ──→ Bytecode
         ──→ Block Lowering                                       Compiler
         ──→ Type Projection                                        │
         ──→ DictApp Dispatch                                   Bytecode
         ──→ Func Qualification                                   Image
                   │                                                │
              Statement IR                                     Bytecode VM
                   │
              Go Emitter (existing, ~780 LOC)
```

---

## 3. Design Decisions

### 3.1 Register VM (Not Stack VM)

**Decision**: Register-based virtual machine.

**Rationale**: M-PERF4 proposed a stack VM. Register VMs are better for AILANG because:

1. **Statement IR is already register-like**: VarDecl assigns to named locals; expressions reference locals by name. The IR naturally maps to registers, not stack operations.
2. **Fewer instructions**: Stack VMs need PUSH/POP around every operation. Register VMs encode operands directly: `ADD r3, r1, r2` vs `PUSH r1; PUSH r2; ADD; POP r3`.
3. **Better performance**: Lua 5.0 showed 20-30% speedup switching from stack to register VM. Dalvik (Android) chose register for the same reason.
4. **Simpler compiler**: Statement IR's flat variable declarations map directly to register allocation.

**Trade-off**: Instruction encoding is wider (3-address vs 0-address). This is acceptable for interpreted dispatch.

### 3.2 Value Representation: Two-Phase Strategy

**Semantic value model** (invariant — must survive representation changes):
- Values are tagged: int, float, bool, unit, string, list, tuple, record, closure, ADT
- Primitives (int, float, bool, unit) should be unboxed where possible
- Heap objects (string, list, record, closure, ADT) are reference types

**Phase 2B initial representation**: Simple tagged Go struct.

```go
// Initial: clear, debuggable, correct. Optimize later.
type Value struct {
    Tag  ValueTag
    Int  int64      // if Tag == TagInt
    Flt  float64    // if Tag == TagFloat
    Bool bool       // if Tag == TagBool
    Obj  any        // heap objects (string, list, record, closure, ADT)
}
```

This is 32 bytes — larger than ideal, but correct, portable, and easy to debug. Performance benchmarks after Phase 2B will reveal whether value representation is actually the bottleneck.

**Optimization path (gated by benchmark evidence)**: NaN-boxed `uint64`.

```go
// Future: if benchmarks show value representation is the bottleneck.
// NaN-boxing: 8 bytes, no allocation for int/float/bool/unit.
// IEEE 754 quiet NaN range encodes type tags in mantissa bits.
// 48-bit pointer for heap objects (covers 256 TB address space).
type Value uint64
```

NaN-boxing is attractive (8 bytes, zero-alloc primitives) but adds complexity:
- Go does not expose raw memory like C — pointer tagging interacts with GC/write barriers
- 48-bit pointer assumptions are platform-dependent
- Debugging is significantly harder (no readable field names)
- Must be validated against Go's escape analysis

**Decision rule**: Implement tagged struct first. If Phase 2B benchmarks show >20% time in value dispatch/allocation, implement NaN-boxing as Phase 2D optimization.

**Heap objects** (strings, lists, records, closures, ADTs) are reference types in either representation. These are less common in hot loops.

### 3.3 Closure Capture: Flat Closures

**Decision**: Flat closure capture (copy captured values into closure at creation time).

**Rationale**:
- AILANG is immutable-by-default — captured values don't change after capture
- No need for Lua-style upvalue indirection (which handles mutable captures)
- LetRec recursive bindings are the exception — handled via `IndirectSlot` (a single mutable pointer cell)
- Simpler implementation, better cache locality

### 3.4 Effect System Integration

**Decision**: Effects are **not** baked into the bytecode opcode set.

**Rationale**: The evaluator handles effects correctly. Rather than reimplementing effect handlers in the VM (which would create a second semantic authority), the VM should:

1. **Trap to evaluator** on effect boundaries: When a bytecode function performs an effect, the VM yields control to the evaluator's effect handler
2. **Resume bytecode** after the effect handler completes
3. This means: pure bytecode functions run at VM speed; effectful boundaries have evaluator overhead

The VM does not implement effect handlers. It only detects effectful boundary operations and delegates them to the evaluator. This is the "hybrid" principle — bytecode for hot pure paths, evaluator for effect boundaries. The boundary is explicit and well-defined.

See Section 3.7 (VM ↔ Evaluator Boundary Contract) for the full interop specification.

```
Bytecode execution (fast)
    │
    ├── pure computation ← VM handles directly
    ├── pure computation ← VM handles directly
    │
    ╞══ EFFECT BOUNDARY ══╡
    │                      │
    │              Evaluator handles effect
    │              (IO, Net, State, etc.)
    │                      │
    ╞══════════════════════╡
    │
    ├── pure computation ← VM resumes
    └── return
```

### 3.5 Pattern Matching: Already Lowered

**Decision**: No pattern-matching opcodes needed.

Statement IR has already lowered all 7 pattern types to if/switch trees. The bytecode compiler just emits conditional jumps. This is a major simplification over M-PERF4's proposed `OP_MATCH_TAG`, `OP_MATCH_LIT`, `OP_MATCH_FAIL`.

### 3.6 Semantic Equivalence Contract

A bytecode execution is **equivalent** to evaluator execution iff:

| Dimension | Contract |
|-----------|----------|
| **Return value** | Equal under canonical runtime equality (structural, deterministic) |
| **Effect order** | Observable effects occur in identical order |
| **Error semantics** | Same program point fails for unsupported operations; error messages may differ in formatting but must identify the same source location |
| **Divergence** | If evaluator diverges, VM diverges (and vice versa). Tail call optimization does not change termination behavior — it only changes stack depth |
| **Floating point** | Follows evaluator semantics exactly. No fast-math or precision deviation |
| **Budget/accounting** | If budgets are enabled, same exhaustion conditions trigger. Instruction counts may differ but budget semantics are preserved |
| **Traces/debug** | May differ in formatting but must preserve source location + causal ordering |
| **Map/dict ordering** | Not observable in AILANG (A1 determinism). VM may use different internal ordering |

**Verification method**: During Phase 2B-2C, all golden tests run under both evaluator and bytecode. Results are compared structurally. The `--debug-bytecode` flag runs both in parallel and diffs outputs.

**What is NOT required to be equivalent**:
- Internal register layout, stack depth, instruction count
- Allocation patterns (VM may allocate less)
- Execution speed (VM should be faster, that's the point)
- Error message phrasing (source location must match)

### 3.7 VM ↔ Evaluator Boundary Contract

The boundary between VM and evaluator is the most critical design surface. Ambiguity here recreates the shadow implementation problem.

**Unit of compilation: function-granular.**

A function is either:
- **Compiled**: executed entirely by VM (including all sub-expressions)
- **Evaluated**: executed entirely by evaluator

No arbitrary mid-function fallback. If a function contains unsupported constructs, the **entire function** is marked evaluator-only at compile time. The only exception is explicit `EFFECT_TRAP` boundaries within compiled functions.

**Call boundary semantics:**

| Scenario | Behavior |
|----------|----------|
| VM function calls VM function | Normal CALL/TAIL_CALL (stays in VM) |
| VM function calls evaluated function | VM suspends frame, converts args → eval values, calls evaluator, converts result → VM value, resumes |
| Evaluated function calls VM function | Evaluator calls VM entry point with converted args |
| VM function hits effect boundary | EFFECT_TRAP: VM suspends, evaluator handles effect, VM resumes with result |

**Value conversion rules:**

| VM Value | Evaluator Value | Notes |
|----------|----------------|-------|
| Int | `*eval.IntValue` | Direct copy |
| Float | `*eval.FloatValue` | Direct copy |
| Bool | `*eval.BoolValue` | Direct copy |
| Unit | `*eval.UnitValue` | Singleton |
| String (heap) | `*eval.StringValue` | Shared reference (strings are immutable) |
| List (heap) | `*eval.ListValue` | Deep conversion (elements may need conversion) |
| Record (heap) | `*eval.RecordValue` | Deep conversion |
| Closure (VM) | Wrapped closure | VM closure wrapped as evaluator-callable function |
| ADT (heap) | `*eval.ADTValue` | Tag + deep field conversion |

**Ownership**: Values are copied at boundaries, not shared. The VM and evaluator have independent value heaps. This prevents GC/lifecycle coupling.

**Error propagation**: Errors from evaluator calls are converted to VM errors and propagated normally. Stack traces include both VM frames and evaluator frames, marked by execution tier.

**Reentrance**: Evaluator may call back into VM (e.g., evaluator calls a function that was compiled). This creates interleaved frames. Call depth is bounded by the same recursion limit as the evaluator.

### 3.8 Builtin Strategy: No Semantic Duplication

Builtins are the highest-risk area for creating a shadow runtime. The design must prevent semantic drift between evaluator builtins and VM builtins.

**Strategy: shared semantic core with thin adapters.**

```
                    ┌─────────────────────────┐
                    │  Builtin Semantic Core   │
                    │  (single implementation) │
                    └──────────┬──────────────┘
                               │
                    ┌──────────┼──────────┐
                    │                      │
              Evaluator Adapter      VM Adapter
              (eval.Value args)      (vm.Value args)
```

**Builtin classification:**

| Tier | Strategy | Examples |
|------|----------|---------|
| **Pure primitive** | Shared Go function, thin value adapters on each side | `string_length`, `int_add`, arithmetic |
| **Pure complex stdlib** | Compile from AILANG source to bytecode where possible; shared Go impl otherwise | `list_map`, `list_filter`, `foldl` |
| **Effectful** | Always trap to evaluator — no VM-native effect implementation | `println`, `readFile`, `httpGet` |
| **Semantically tricky** | Evaluator path first; VM-native only after differential testing confirms equivalence | `json_decode`, `regex_match` |

**Hard rule**: A builtin must not have two independent implementations of its semantics. Either:
1. Both sides call the same Go function (with value adapters), or
2. The VM delegates to the evaluator for that builtin

**Differential testing**: All builtins have property tests that run the same inputs through both evaluator and VM paths, comparing outputs.

---

## 4. Instruction Set

### 4.1 Encoding

Each instruction is a 32-bit word:

```
┌──────────┬──────────┬──────────┬──────────┐
│  OpCode  │    A     │    B     │    C     │
│  8 bits  │  8 bits  │  8 bits  │  8 bits  │
└──────────┴──────────┴──────────┴──────────┘

Or for wide operands:
┌──────────┬──────────┬─────────────────────┐
│  OpCode  │    A     │       Bx (16 bits)  │
│  8 bits  │  8 bits  │                     │
└──────────┴──────────┴─────────────────────┘
```

- **A**: Destination register (or first operand)
- **B, C**: Source registers (or operand + small constant)
- **Bx**: Wide operand for jumps, constant pool indices

### 4.2 Opcodes

```
; Loads
LOAD_CONST    A, Bx     ; R[A] = Constants[Bx]
LOAD_NIL      A         ; R[A] = Unit
MOVE          A, B      ; R[A] = R[B]
LOAD_GLOBAL   A, Bx     ; R[A] = Globals[Bx]

; Arithmetic (int and float, dispatched by value tag)
ADD           A, B, C   ; R[A] = R[B] + R[C]
SUB           A, B, C   ; R[A] = R[B] - R[C]
MUL           A, B, C   ; R[A] = R[B] * R[C]
DIV           A, B, C   ; R[A] = R[B] / R[C]
MOD           A, B, C   ; R[A] = R[B] % R[C]
NEG           A, B      ; R[A] = -R[B]

; Comparison
EQ            A, B, C   ; R[A] = R[B] == R[C]
LT            A, B, C   ; R[A] = R[B] < R[C]
LE            A, B, C   ; R[A] = R[B] <= R[C]

; Logic
NOT           A, B      ; R[A] = !R[B]
; AND/OR compiled as conditional jumps (short-circuit)

; String
CONCAT        A, B, C   ; R[A] = R[B] ++ R[C]

; Control flow
JUMP          sBx       ; IP += sBx (signed offset)
JUMP_IF_FALSE A, sBx    ; if !R[A] then IP += sBx
CALL          A, B, C   ; R[A..A+C-1] = R[A](R[A+1..A+B])
TAIL_CALL     A, B      ; return R[A](R[A+1..A+B])  ; reuse frame
RETURN        A         ; return R[A]

; Closures
CLOSURE       A, Bx     ; R[A] = new closure from Prototypes[Bx]
                        ; followed by MOVE instructions for captures

; Collections
MAKE_LIST     A, B, C   ; R[A] = [R[B]..R[B+C-1]]
MAKE_TUPLE    A, B, C   ; R[A] = (R[B]..R[B+C-1])
MAKE_RECORD   A, B, C   ; R[A] = {fields from R[B]..R[B+C-1]}
                        ; field names from constant pool
CONS          A, B, C   ; R[A] = R[B] :: R[C]
GET_FIELD     A, B, Cx  ; R[A] = R[B].Fields[Cx]  (Cx = field name index)
GET_INDEX     A, B, C   ; R[A] = R[B][R[C]]

; ADT
MAKE_ADT      A, B, C   ; R[A] = ADT{tag=B, fields=R[A+1..A+C]}
GET_TAG       A, B      ; R[A] = R[B].tag  (for switch dispatch)

; Builtins
BUILTIN_CALL  A, Bx, C  ; R[A] = BuiltinTable[Bx](R[A+1..A+C])

; Builtins (effectful variant — traps to evaluator for execution)
BUILTIN_TRAP  A, Bx, C  ; R[A] = evaluator.CallBuiltin(Bx, R[A+1..A+C])
                        ; for effectful builtins (IO, Net, etc.)

; Effects (trap to evaluator)
EFFECT_TRAP   A, Bx     ; yield to evaluator for effect Bx, arg in R[A]
                        ; evaluator resumes VM with result in R[A]
```

### 4.3 Opcode Design Notes

**BUILTIN_CALL vs BUILTIN_TRAP**: Pure builtins use `BUILTIN_CALL` (executed in VM via shared Go function). Effectful builtins use `BUILTIN_TRAP` (delegates to evaluator). The compiler classifies each builtin at compile time using the `IsPure` flag from `BuiltinSpec`.

**GET_INDEX semantics**: Operates on lists only (indexed by integer). Tuple field access uses `GET_FIELD` with positional index. Map lookups use `BUILTIN_CALL` to `map_lookup`. String character access uses `BUILTIN_CALL` to `str_charAt`. No single opcode tries to be polymorphic over collection types.

**MAKE_RECORD field ordering**: Fields are stored in **alphabetical order** by name (matching AILANG's canonical record ordering for A1 determinism). Field name indices reference the constant pool. This ordering is invariant — equality, access, and debugging all depend on it.

**MAKE_ADT tag values**: Tags are **per-type local ordinals** assigned during type elaboration (same ordering as the evaluator's tag assignment). Tags are NOT globally unique — `Option.Some` and `Result.Ok` may both have tag 0. The type is disambiguated by the function prototype's type metadata. Tags are stable within a compilation unit.

**Total: ~30 opcodes** — significantly smaller than M-PERF4's draft because pattern matching is pre-lowered.

---

## 5. Compilation: Statement IR → Bytecode

### 5.1 Pipeline

```
stmt.Program
    │
    ├── stmt.FuncDecl → FuncPrototype (register-allocated bytecode + constant pool)
    ├── stmt.FuncDecl → FuncPrototype
    ├── ...
    │
    └── BytecodeImage (all prototypes + global table + type metadata)
```

### 5.2 Register Allocation

Statement IR uses named variables. The bytecode compiler assigns register indices:

```
Statement IR:                    Bytecode:
  var x int = 42                   LOAD_CONST  R0, #0    ; 42
  var y int = x + 1               ADD         R1, R0, #1 ; (immediate 1)
  return y                         RETURN      R1
```

Algorithm: Linear scan over Statement IR statements. Each VarDecl gets the next available register. Temporaries for sub-expressions use short-lived registers that are recycled.

### 5.3 Statement Compilation

Each `stmt.Stmt` compiles to a sequence of instructions:

| Statement IR | Bytecode |
|-------------|----------|
| `VarDecl{Name, Value}` | Compile Value into register for Name |
| `ExprStmt{Expr}` | Compile Expr, discard result |
| `ReturnStmt{Value}` | Compile Value, RETURN |
| `IfStmt{Cond, Then, Else}` | JUMP_IF_FALSE over Then block, JUMP over Else |
| `SwitchStmt{Tag, Cases}` | GET_TAG, chain of EQ + JUMP_IF_FALSE |
| `AssignStmt{Target, Value}` | Compile Value, MOVE to Target register |

### 5.4 Expression Compilation

Each `stmt.Expr` compiles to instructions that produce a value in a target register:

| Statement IR Expr | Bytecode |
|-------------------|----------|
| `LitInt{N}` | LOAD_CONST |
| `VarRef{Name}` | MOVE (from named register) |
| `GlobalRef{Module, Name}` | LOAD_GLOBAL |
| `BinOp{Op, L, R}` | Compile L, Compile R, ADD/SUB/MUL/... |
| `Call{Func, Args}` | Compile Func + Args into consecutive registers, CALL |
| `Lambda{Params, Body}` | CLOSURE + capture MOVEs |
| `IfExpr{Cond, Then, Else}` | JUMP_IF_FALSE, compile branches |
| `ListLit{Elems}` | Compile elems, MAKE_LIST |
| `Cons{Head, Tail}` | Compile head + tail, CONS |
| `ADTConstructor{Tag, Args}` | Compile args, MAKE_ADT |
| `FieldAccess{Record, Field}` | Compile record, GET_FIELD |
| `BuiltinCall{Name, Args}` | Compile args, BUILTIN_CALL |

### 5.5 Builtin Dispatch

Statement IR's `BuiltinCall` maps to a builtin function table in the VM. Each builtin is a Go function:

```go
type BuiltinFunc func(vm *VM, args []Value) Value

var builtinTable = map[string]BuiltinFunc{
    "list_map":      builtinListMap,
    "list_filter":   builtinListFilter,
    "string_length": builtinStringLength,
    // ... all stdlib functions implemented once in Go
}
```

This solves the 90-stdlib-stub problem from Go source emission. The stubs are Go functions called from bytecode — not generated Go source code that must type-check.

---

## 6. Bytecode Image Format

```go
type BytecodeImage struct {
    Version     uint32           // Image format version
    SourceHash  [32]byte         // SHA-256 of source/IR (for staleness detection)
    Constants   []Value          // Constant pool (strings, large ints, etc.)
    Globals     []GlobalEntry    // Global variable table
    Prototypes  []FuncPrototype  // Function definitions
    EntryPoint  uint32           // Index of main function prototype
    Metadata    ImageMetadata    // Source file info, debug data (optional, strippable)
}

type FuncPrototype struct {
    Name        string           // Function name (for debugging)
    Module      string           // Module name
    NumParams   uint8            // Parameter count
    NumRegisters uint8           // Total registers needed
    Code        []uint32         // Instruction words
    Captures    []CaptureDesc    // Which registers to capture from parent
    LineInfo    []LineEntry      // Source line mapping (for errors)
}

type GlobalEntry struct {
    Module string
    Name   string
    Index  uint32  // Index in VM globals array
}
```

---

## 7. VM Runtime

### 7.1 Core Loop

```go
type VM struct {
    registers []Value         // Current frame's registers
    frames    []CallFrame     // Call stack
    globals   []Value         // Global values
    image     *BytecodeImage  // Loaded bytecode
    evaluator EvalInterop     // Interface for effect traps (Tier A fallback)
}

type CallFrame struct {
    proto    *FuncPrototype
    ip       int          // Instruction pointer within proto.Code
    baseReg  int          // Base index in register file
    retReg   int          // Where caller wants the return value
}

func (vm *VM) Run() (Value, error) {
    for {
        frame := &vm.frames[len(vm.frames)-1]
        inst := frame.proto.Code[frame.ip]
        frame.ip++

        op := OpCode(inst >> 24)
        // ... decode A, B, C from inst

        switch op {
        case OP_ADD:
            // Fast path: both NaN-boxed ints
            // Slow path: float, or heap string concat
        case OP_CALL:
            // Push new frame, set registers
        case OP_TAIL_CALL:
            // Reuse current frame (no stack growth)
        case OP_RETURN:
            // Pop frame, write result to caller's retReg
        case OP_EFFECT_TRAP:
            // Yield to evaluator, resume after
        // ...
        }
    }
}
```

### 7.2 Tail Call Optimization

Critical for recursive AILANG programs. When the compiler detects a call in tail position (the last expression in a function body), it emits `TAIL_CALL` instead of `CALL`:

```
TAIL_CALL: reuse current frame
  1. Copy arguments into parameter registers
  2. Reset IP to 0
  3. Continue execution (no new frame pushed)
```

This gives constant-stack-space recursion for patterns like:

```ailang
func foldl(f, acc, xs) =
  match xs with
  | [] -> acc
  | x :: rest -> foldl(f, f(acc, x), rest)  -- tail call
```

### 7.3 Evaluator Interop

The VM holds a reference to the tree-walking evaluator for:

1. **Effect handling**: `EFFECT_TRAP` saves VM state, calls evaluator's effect handler, resumes
2. **Fallback for uncompiled functions**: If a function wasn't compiled to bytecode (e.g., uses features not yet supported), the VM calls the evaluator
3. **Debugging**: `--debug-bytecode` flag runs both VM and evaluator, compares results

```go
func (vm *VM) effectTrap(effectID uint16, arg Value) Value {
    // Convert VM value to evaluator value
    evalArg := vm.toEvalValue(arg)
    // Call evaluator's effect handler
    result := vm.evaluator.HandleEffect(effectID, evalArg)
    // Convert back to VM value
    return vm.fromEvalValue(result)
}
```

---

## 8. Compilable Performance Subset

Not all AILANG features need bytecode compilation. The subset is defined by **semantic criteria**, not a historical feature checklist. A function is compilable iff:

1. **Pure or effect-bounded**: No effects, or effects only at explicit `EFFECT_TRAP` points
2. **Concrete resolved types**: All types resolved after monomorphization (no residual type variables)
3. **Closed-world call graph**: All callees are either compiled functions or known builtins
4. **No row-polymorphism at execution boundary**: Record types are fully resolved
5. **No dynamic dictionary ambiguity**: Type class methods resolved at compile time
6. **Representable in VM value model**: All values fit the tagged value representation
7. **Bounded closure capture**: Capture set is finite and statically known

Functions that fail any criterion are marked evaluator-only at compile time. Profiling decides whether a compilable function is *worth* compiling (hot vs cold) — that is a separate question from whether it *can* be compiled.

### Tier 1: Must compile (launch)

- Integer and float arithmetic
- Boolean logic
- String operations
- Local variables and function calls
- If/else expressions
- Pattern matching (already lowered to if/switch by Statement IR)
- List/tuple/record construction and access
- ADT construction and tag dispatch
- Closures (flat capture)
- Tail calls
- Builtins (via function table)

### Tier 2: Should compile (v0.12+)

- Cross-module calls
- Record update
- Type class dictionary dispatch
- Higher-order functions with complex capture

### Tier 3: Evaluator fallback (indefinite)

- Effect handlers (compile the pure part, trap for effects)
- Row-polymorphic operations (need runtime type info)
- Reflection/meta-programming
- Dynamic module loading

This subset covers the vast majority of hot-loop code. The evaluator handles everything else.

---

## 9. Phase 2A Gate: Benchmark First

**This document must NOT be implemented until Phase 2A benchmarks complete.**

### Required Benchmarks

| Workload | What It Tests | Metrics |
|----------|--------------|---------|
| `fib(30)` recursive | Pure function call overhead | Throughput, allocs/op |
| `map/filter` over 10K-element list | Collection throughput | Throughput, allocs/op |
| Pattern match (nested ADT, 5 levels) | Match dispatch cost | Throughput |
| Closure-heavy (curried HOFs, partial application) | Closure creation/call overhead | Throughput, allocs/op |
| Cross-boundary (compiled pure calling effectful helper in loop) | VM ↔ evaluator boundary cost | p95 latency per call |
| DocParse representative pipeline | Real-world mixed workload | End-to-end time, peak memory |
| Game frame step (if available) | Latency-sensitive loop | p95 latency per frame |

### Decision Rules

| Evaluator Result | Action |
|-----------------|--------|
| Meets all targets (< 5x native Go) | Ship embedded evaluator (Option C). Defer bytecode |
| Misses only on hot loops (> 10x native Go for fib/map) | Build bytecode VM (this document) |
| Misses across the board | Build bytecode VM with higher priority |

### How to Benchmark

```bash
# Evaluator performance
ailang bench fib30.ail --iterations 100 --warmup 10
ailang bench list_ops.ail --iterations 100

# Native Go comparison
go test -bench=. -benchmem internal/eval/bench_test.go

# Frame budget test
ailang bench game_step.ail --max-latency 16ms  # 60 FPS = 16.67ms per frame
```

---

## 10. Implementation Plan

### Phase 2A: Benchmarks (1 week) — ✅ COMPLETE (2026-04-03)

- [x] Create benchmark suite for evaluator (fib, list ops, pattern match, real workloads)
- [x] Run benchmarks, record baselines
- [x] Compare against native Go equivalents
- [x] Decision: proceed with bytecode or ship embedded evaluator → **BUILD BYTECODE VM (high priority)**

Results: all 7 workloads failed the 10x threshold. See [phase2a-results.md](phase2a-results.md).

### Phase 2B: Bytecode Foundation (2 weeks) — ✅ COMPLETE (2026-04-08)

**Sprint plan**: [m-phase2b-bytecode-foundation-sprint-plan.md](m-phase2b-bytecode-foundation-sprint-plan.md)

- [x] `internal/bytecode/opcode.go` — Instruction encoding + 32 opcodes
- [x] `internal/bytecode/image.go` — BytecodeImage, FuncPrototype, constant pool, Validate()
- [x] `internal/bytecode/value.go` — Tagged-struct Value type (NaN-boxing deferred to Phase 2D per §3.2)
- [x] `internal/vm/frame.go` — Call frame + tail-call frame reuse
- [x] `internal/vm/vm.go` — Core dispatch loop (all 30 non-trap opcodes)
- [x] `internal/vm/vm_test.go` — Per-opcode-group hand-assembled tests
- [x] `internal/vm/vm_fib_test.go` — Acceptance gate: hand-written `fib(25) = 75025`

**Acceptance**: ✅ Hand-written bytecode for `fib(25)` returns 75025. Tail-call accumulator form runs 10,000 iterations under MaxStack=5 (TCO verified).

**Total LOC**: 2,988 (1,552 production + 1,436 tests). Within the §12 budget (~2,650-3,500).

**Deviations from §11 layout**:
- `Value` is in `internal/bytecode/value.go` (not `internal/vm/value.go`). This preserves §11's import direction (`vm → bytecode`, never reverse) while letting bytecode's constant pool hold typed values directly. `internal/runtime/` was rejected as a host because it already imports the evaluator.

**Phase 2B notes**:
- BUILTIN_CALL, BUILTIN_TRAP, EFFECT_TRAP are wired as opcodes but return clear "not implemented in Phase 2B" errors. They will be implemented in Phase 2C/2E.
- Closure captures use Lua-style pseudo-MOVE instructions following CLOSURE; the dispatcher reads `NumCaptures` from the inner prototype to know how many to consume.
- LOAD_CONST's `Bx` indexes a prototype-local constant table (a slice of pool indices), not the image pool directly. Two-level indirection keeps prototypes small and centralizes dedup.

### Phase 2C: Statement IR Compiler (2 weeks)

- [ ] `internal/bytecode/compiler.go` — Statement IR → bytecode compiler
- [ ] Register allocation (linear scan over VarDecls)
- [ ] Expression compilation (all Tier 1 expression types)
- [ ] Statement compilation (VarDecl, Return, If, Switch)
- [ ] Closure compilation (flat capture)
- [ ] Tail call detection and TAIL_CALL emission
- [ ] Builtin function table

**Acceptance**: `ailang run --bytecode fib.ail` produces correct result for all 12 golden test inputs.

### Phase 2D: Integration and Polish (1 week) — **COMPLETE 2026-04-08**

- [x] CLI flag: `--bytecode` on `ailang run` (M2_CLI_FLAG)
- [x] Evaluator fallback for uncompiled functions (M3_EVAL_FALLBACK — per-function `EvalOnly` tag + bytecode↔eval value bridge for Tier-1 shapes)
- [x] Disassembler: `ailang disasm program.ail` (M1_DISASM)
- [x] Error messages with source locations via LineInfo (M4_LINEINFO)
- [x] Benchmark: bytecode vs evaluator (M5_BENCH — ~26x faster on fib(25), ~55x fewer allocs; [bench report](m-bytecode-2d-bench.md))
- [x] All existing evaluator tests pass under bytecode execution (M6_FULL_PARITY — 134/141, effectively 100% on eligible; [parity report](m-bytecode-2d-parity.md))

**Acceptance**: Bytecode path passes full test suite. Measurable speedup over evaluator on pure workloads. **MET.**

### Phase 2E: Effect Boundary (1 week, if needed)

- [ ] EFFECT_TRAP instruction
- [ ] VM ↔ evaluator value conversion
- [ ] Effect handler dispatch via evaluator
- [ ] Resume bytecode after effect completion

**Acceptance**: Effectful programs produce correct results. Pure hot paths still run at VM speed.

---

## 11. Package Structure

```
internal/
├── bytecode/
│   ├── opcode.go       # OpCode enum, instruction encoding/decoding
│   ├── image.go        # BytecodeImage, FuncPrototype, serialization
│   ├── compiler.go     # Statement IR → bytecode compiler
│   ├── compiler_test.go
│   └── disasm.go       # Disassembler for debugging
├── vm/
│   ├── value.go        # NaN-boxed Value type
│   ├── vm.go           # Virtual machine core loop
│   ├── vm_test.go
│   ├── builtins.go     # Builtin function table
│   └── interop.go      # VM ↔ evaluator value conversion
```

**Import rules** (hard constraints):
- `internal/bytecode/` imports `internal/gen/stmt/` (Statement IR) — nothing else from gen/
- `internal/vm/` imports `internal/bytecode/` — does NOT import stmt/, lower/, or core/
- `internal/vm/` imports `internal/eval/` only for effect interop (via interface, not direct)

---

## 12. LOC Estimates

| Component | Estimated LOC | Confidence |
|-----------|--------------|------------|
| Opcode definitions + encoding | ~150 | High |
| BytecodeImage + serialization | ~200 | High |
| Tagged Value type (struct initially, NaN-box later) | ~200 | Medium |
| VM core loop (30 opcodes) | ~400 | Medium |
| Statement IR → bytecode compiler | ~500 | Medium |
| Builtin function table | ~300 | High |
| Disassembler | ~150 | High |
| Tests | ~500 | Medium |
| CLI integration | ~100 | High |
| Value conversion (VM ↔ evaluator) | ~150 | Medium |
| **Total** | **~2,650-3,500** | |

At current velocity (~300 LOC/day), this is approximately **9-12 working days** after the Phase 2A benchmark gate. The range reflects debug tooling and interop complexity.

---

## 13. Success Criteria

### Launch Criteria (must hit)

- [ ] **Correctness**: All 12 golden test inputs produce identical output under bytecode and evaluator
- [ ] **Correctness**: Full evaluator test suite passes under bytecode execution (for compilable subset)
- [ ] **Correctness**: Tail calls do not grow stack (verified with recursion depth > 100K)
- [ ] **Correctness**: Effect boundary produces correct results for effectful programs
- [ ] **Performance**: Measurable speedup over evaluator on pure hot workloads (fib, list ops, pattern match)
- [ ] **Performance**: Meets frame budget on target benchmark (if game step benchmark exists)
- [ ] **Architecture**: Bytecode compiler imports only `internal/gen/stmt/` (no Core/AST leakage)
- [ ] **Architecture**: VM imports only `internal/bytecode/` + evaluator interface for effects
- [ ] **Architecture**: No semantic changes to evaluator. Minimal interface extraction permitted for VM interop
- [ ] **Architecture**: No builtin has two independent semantic implementations (shared core or evaluator delegation)

### Stretch Criteria (target, not required for launch)

- [ ] `fib(30)`: < 50ms (evaluator baseline ~260ms for fib(25))
- [ ] List map/filter 10K elements: within 3x of native Go
- [ ] Pattern match dispatch: within 5x of native Go switch
- [ ] Zero allocation for pure int/float/bool computation (requires NaN-boxing optimization)
- [ ] Total bytecode + VM code < 3,500 LOC

---

## 14. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Builtin semantic drift** | **High** | Single shared builtin semantic core + differential tests. No independent reimplementations |
| **Mixed execution complexity** | **High** | Function-granular compilation boundary. No arbitrary mid-function fallback. Explicit ABI (Section 3.7) |
| Semantic divergence between VM and evaluator | High | Semantic Equivalence Contract (Section 3.6). Run both in parallel during testing; diff outputs |
| Effect boundary overhead dominates | Medium | Benchmark pure vs effectful; if pure is fast enough, accept boundary cost. Cross-boundary benchmark in Phase 2A |
| Value representation overhead | Medium | Start with tagged struct; NaN-box only if benchmarks show >20% time in value dispatch. Two-phase approach (Section 3.2) |
| Register allocation for complex functions | Medium | Start with simple linear allocation; optimize later |
| Bytecode not needed (evaluator fast enough) | Low | Phase 2A gate prevents wasted work |

---

## 15. Relationship to Existing Documents

| Document | Relationship |
|----------|-------------|
| M-CODEGEN-STRATEGIC-REVIEW | **Parent** — this implements the G+D recommendation |
| M-PERF4 (bytecode-interpreter) | **Supersedes** — M-PERF4 compiled Core→bytecode; this compiles StmtIR→bytecode |
| M-CODEGEN-IR-STRATEGY | **Builds on** — Statement IR is the compilation boundary |
| M-CODEGEN-SUSTAINABILITY | **Obsoletes** — runtime bridge approach replaced by VM builtins |

---

## 16. Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Bytecode execution is deterministic; register VM has no non-deterministic dispatch |
| A2: Replayable Execution | +1 | Bytecode images are stable artifacts; same image → same execution |
| A5: Bounded Verification | +1 | Each opcode has a bounded, verifiable effect on registers |
| A7: Machines First | +2 | Bytecode is machine-optimized representation; disassembler provides inspectability |
| A9: Cost Is Part of Meaning | +2 | VM can count instructions, track allocation, enforce budgets per-instruction |
| A10: Composability | +1 | VM + evaluator compose at effect boundaries; bytecode functions compose with evaluated functions |
| **Net Score** | **+8** | Strong alignment |

---

## Changelog

| Date | Change |
|------|--------|
| 2026-04-03 | Initial design — promoted from M-PERF4, redesigned around Statement IR + register VM + NaN-boxing + evaluator interop |
| 2026-04-03 | **Review revisions**: Added Semantic Equivalence Contract (§3.6), VM ↔ Evaluator Boundary Contract (§3.7), Builtin Strategy (§3.8). Recast NaN-boxing as optimization path with tagged struct as initial impl. Made fallback function-granular. Added closure-heavy and cross-boundary benchmarks. Separated launch vs stretch success criteria. Added builtin drift and mixed execution risks. Refined LOC estimates to 2,650-3,500. Softened "zero evaluator modifications" to "no semantic changes" |
| 2026-04-08 | **M-BYTECODE-BATCH complete**: `--batch` now honors `--bytecode`. Per-function lower-panic recovery lets partial lowering succeed (panicking functions become EvalOnly stubs with preserved arity). Entry-level EvalOnly dispatch now uses `runtime.CallEntrypoint` (not `bridge.CallEvalFunc`) so Fork()/resolver/goroutine setup fires correctly — fixes Process effect ordering. Disassembler prints per-prototype EvalOnly reason and a header EvalOnly count. See §17 for the docparse finding. |

---

## 17. M-BYTECODE-BATCH (2026-04-08) — Real-World Benchmark & Next-Sprint Scope

### 17.1 Status after M-BYTECODE-BATCH

**Parity**: 133 MATCH / 2 NON_DET / 6 EVAL_SKIP of 141 runnable examples → **100% of eligible examples** produce byte-identical stdout on evaluator and bytecode VM. The two NON_DET entries (`uuid.ail`, `stream_process_source.ail`) are inherently non-deterministic and excluded by an explicit allow-list in `scripts/verify_bytecode_parity.go`.

**Batch mode**: `ailang run --bytecode --batch ...` now threads `bytecodeMode`/`strictBytecode`/`pipelineResult` into `executeBatchItem`, so the per-item entrypoint call takes the VM path. The `--batch` startup amortization hypothesis is now testable end-to-end.

### 17.2 Benchmark: `ailang-parse`

`ailang-parse` (sibling repo `/Users/mark/dev/sunholo/ailang-parse/`) is a 19-module document parser that uses `ailang run --batch --entry main` to convert DOCX/PPTX/XLSX/EPUB to structured JSON. It is the motivating real-world consumer of batch mode.

**Finding**: On a ~9 MB stress-test DOCX, wall-clock time with `--bytecode` is **statistically indistinguishable** from the evaluator path. The speedup we saw on single-module microbenchmarks (M5 reported **25× on `fib(30)`**) does not appear on a multi-module application.

**Root cause** — `ailang disasm` on `ailang-parse/main.ail` shows **28 / 34 prototypes are EvalOnly**, meaning almost every call is dispatched through the bridge back to the evaluator. Breakdown of the EvalOnly reasons:

| Reason | Count | Category |
|---|---:|---|
| `call to unknown global` (println, split, parseDocx, …) | ~16 | Cross-module function resolution |
| `unbound variable $tmp242/t` | 5 | Lower-pass artifact (temporaries escaping block scope) |
| `unknown ADT Block in switch` | 2 | Cross-module ADT type resolution |
| `cannot resolve field count access (no type info)` | 1 | Cross-module record field lookup |
| `lower panic: non-tail-position match` | 2 | Known §4 lowering gap (main + parseImageDocument) |
| **EvalOnly total** | **28 / 34** | |

**Interpretation**: M5's 25× `fib` speedup was for a self-contained single-module function where the bytecode compiler could see every call-site and every ADT constructor in one image. A 19-module application hits the bytecode compiler's **cross-module resolution wall** on almost every function. The bridge then dispatches back to the evaluator, so the VM never runs the hot paths, and the 25× advantage never materialises.

### 17.3 Recommended next sprint: M-BYTECODE-MULTIMODULE

Unblock multi-module speedups by teaching the bytecode compiler to resolve names across module boundaries at compile time. Three sub-items, ordered by impact on the docparse EvalOnly count:

1. **Cross-module function globals (~16 cases)**. Today `call to unknown global` fires whenever a function references another module's export. The compiler needs a module-aware global resolver so imported functions get real prototype indices (or global slots) rather than bridged bailouts.
2. **Cross-module ADT + record layouts (~3 cases)**. Switches over imported ADTs (`Block`, etc.) and record field access need the importer's type info surfaced into the compiler's layout table.
3. **Lower-pass temporary hygiene (~5 cases)**. `unbound variable $tmp242/t` is a pass-internal bug where `let`-bound temporaries from the block lowering escape their scope. Fix at the lower pass, independent of multi-module work.

Addressing (1) + (2) should drop docparse from 28/34 EvalOnly to ~4/34, at which point the VM actually executes the parser's hot path and we can measure the real speedup. (3) is an independent cleanup that unlocks a handful of functions in other examples.

### 17.4 What we deliberately did NOT do

- **Default `--bytecode` on**. With 28/34 functions bridged in docparse, turning it on by default would add compile-time cost with no runtime benefit on real apps. Defer until after M-BYTECODE-MULTIMODULE.
- **Fix the non-tail-position match lowering**. Only 2 cases in docparse; not on the critical path. Revisit after cross-module resolution.
- **Broaden the non-deterministic allow-list**. Kept to 2 known cases; future additions require an explicit code comment with the divergence reason.
