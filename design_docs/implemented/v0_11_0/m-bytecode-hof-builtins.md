# M-BYTECODE-HOF-BUILTINS: VM Callback Mechanism for Higher-Order Builtins

**Status**: Planned
**Target**: v0.11.0
**Priority**: P1 (Strategic — biggest single unlock for docparse VM performance)
**Estimated**: 1-2 days
**Dependencies**: M-BYTECODE-STDLIB-BUILTINS (complete), M-BYTECODE-VM §18.6 (complete)
**Extends**: [m-bytecode-vm.md](../../implemented/v0_11_0/m-bytecode-vm.md) §19

## Problem Statement

After M-BYTECODE-STDLIB-BUILTINS wired 55 pure builtins, the docparse benchmark still has **163 / 1130 EvalOnly prototypes**. Of these, **6 are HOF builtins** that cascade to block **~82 additional prototypes** — the single largest category of remaining EvalOnly.

The 6 HOF builtins that cannot be wired today:
- `__list_map` — `(a -> b, [a]) -> [b]`
- `__list_filter` — `(a -> bool, [a]) -> [a]`
- `__list_foldl` — `((b, a) -> b, b, [a]) -> b`
- `__str_foldChars` — `((a, string) -> a, a, string) -> a`
- `__str_foldSlices` — `(string, string, a, (a, string) -> a) -> a`
- `__str_mapSlicesJoin` — `(string, string, (string) -> string) -> string`

**Root cause**: The current `BuiltinFunc` signature is `func(args []bytecode.Value) (bytecode.Value, error)`. It cannot call back into the VM because it has no access to the VM instance or the call stack. When a HOF builtin receives a closure argument, it has no way to invoke that closure.

**Impact**: These 6 builtins dominate docparse's hot parsing loops. Until they compile natively, the VM cannot execute the parsing core and must bridge every call through the evaluator. Wiring them should reduce EvalOnly from 163 to ~80, unblocking the path to measurable VM speedup.

## Goals

- **Primary**: Enable VM builtins to invoke closure arguments without bridging to the evaluator
- **Success metrics**:
  - All 6 HOF builtins wired to VM dispatch
  - docparse EvalOnly ≤ 90 / 1130 (down from 163)
  - Parity: no regressions from 129 MATCH baseline
  - `make test` and `make lint` clean

## Non-Goals

- Wiring effectful builtins (IO, FS, Net) — requires `OpEffectCall` with capability validation
- Adding new VM value types (TagMap, TagBytes) — separate design scope
- Compile-time HOF inlining (loop unrolling map/filter into bytecode) — future optimization

## High-Impact Decisions

### D1: Callback Mechanism — Extended BuiltinFunc vs New Opcode

**Option A: Extended BuiltinFunc signature (RECOMMENDED)**

Add a new function type that receives the VM as a callback interface:

```go
// HOFBuiltinFunc is a higher-order builtin that can call VM closures.
type HOFBuiltinFunc func(vm ClosureCaller, args []bytecode.Value) (bytecode.Value, error)

// ClosureCaller is the minimal interface a HOF builtin needs to invoke closures.
type ClosureCaller interface {
    CallClosure(closure bytecode.Value, args []bytecode.Value) (bytecode.Value, error)
}
```

The compiler emits a new opcode `OpBuiltinCallHOF` (or reuses `OpBuiltinCall` with a flag) that passes `vm` to the handler. The VM implements `CallClosure` by pushing a new frame, running the closure, and returning the result.

**Pros**: Simple, composable, testable. Each HOF builtin is a plain Go function. The `ClosureCaller` interface keeps the dependency clean (builtins don't import `*VM` directly).
**Cons**: Each closure call re-enters `vm.run()` recursively (or needs a trampoline). Stack depth must be tracked.

**Option B: Dedicated opcodes per HOF pattern**

Add `OpListMap`, `OpListFilter`, `OpListFoldl`, etc. as first-class VM opcodes that handle the iteration loop inline in the dispatch switch.

**Pros**: Maximum performance — no function call overhead per element.
**Cons**: Explosion of opcodes (one per HOF). Each opcode duplicates the "call closure, collect result" pattern. Hard to extend for new HOFs.

**Decision**: Option A. The interface approach is cleaner, more maintainable, and the per-element overhead of a Go function call is negligible compared to the VM dispatch loop. Option B can be pursued later as an optimization if profiling shows it matters.

### D2: Separate Opcode or Table Split

Two sub-options for Option A:

**D2a: New opcode `OpBuiltinCallHOF`** — A separate opcode that indexes into a second table (`HOFBuiltinTable`). The compiler checks whether a builtin is HOF at compile time and emits the appropriate opcode.

**D2b: Unified table with type assertion** — Keep one `OpBuiltinCall` opcode. Each table entry is `any` (either `BuiltinFunc` or `HOFBuiltinFunc`). The VM type-asserts at dispatch time.

**Decision**: D2a. A separate opcode and table avoids runtime type assertions on every builtin call (the hot path). The compiler already knows which builtins are HOF vs pure — encoding this in the opcode is zero runtime cost.

## Solution Design

### Architecture

```
Compiler                          VM
────────                          ──
stmt.BuiltinCall("__list_map")    OpBuiltinCallHOF idx=0, argc=2
  → compiler checks:                → vm looks up HOFBuiltinTable[0]
    is it in BuiltinTable? no          → hofBuiltinListMap(vm, args)
    is it in HOFBuiltinTable? yes        → vm.CallClosure(closure, [elem])
    → emit OpBuiltinCallHOF               → push frame, run, return result
```

### New Types

```go
// internal/vm/builtins.go

// ClosureCaller is the interface HOF builtins use to invoke closure arguments.
// Implemented by *VM. Keeps HOF builtins decoupled from VM internals.
type ClosureCaller interface {
    CallClosure(closure bytecode.Value, args []bytecode.Value) (bytecode.Value, error)
}

// HOFBuiltinFunc is a builtin that takes closure arguments and can call them.
type HOFBuiltinFunc func(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error)

// HOFBuiltinTable is the dispatch table for OpBuiltinCallHOF.
// Order must match compiler.HOFBuiltinTable.
var HOFBuiltinTable = []HOFBuiltinFunc{
    hofBuiltinListMap,        // __list_map
    hofBuiltinListFilter,     // __list_filter
    hofBuiltinListFoldl,      // __list_foldl
    hofBuiltinStrFoldChars,   // __str_foldChars
    hofBuiltinStrFoldSlices,  // __str_foldSlices
    hofBuiltinStrMapSlicesJoin, // __str_mapSlicesJoin
}
```

### VM.CallClosure Implementation

```go
// internal/vm/vm.go

// CallClosure invokes a closure value with the given arguments and returns
// the result. Used by HOF builtins to call their function arguments.
// This pushes a new frame onto the VM stack, runs it, and returns.
func (vm *VM) CallClosure(closure bytecode.Value, args []bytecode.Value) (bytecode.Value, error) {
    if closure.Tag != bytecode.TagClosure {
        return bytecode.Value{}, fmt.Errorf("CallClosure: expected closure, got %s", closure.Tag)
    }
    c := closure.AsClosure()
    proto, ok := c.Proto.(*bytecode.FuncPrototype)
    if !ok {
        return bytecode.Value{}, fmt.Errorf("CallClosure: non-FuncPrototype")
    }
    if int(proto.NumParams) != len(args) {
        return bytecode.Value{}, fmt.Errorf("CallClosure: %s expects %d args, got %d",
            proto.Name, proto.NumParams, len(args))
    }
    if proto.EvalOnly {
        // Closure compiled to EvalOnly — dispatch through bridge
        if vm.Interop == nil {
            return bytecode.Value{}, fmt.Errorf("CallClosure: %s is evaluator-only but no bridge", proto.Name)
        }
        return vm.Interop.CallEvalFunc(proto.Name, args)
    }
    if len(vm.Stack) >= vm.MaxStack {
        return bytecode.Value{}, ErrStackOverflow
    }
    // Push frame with a dummy return register (result returned directly).
    frame := newFrame(proto, 0, nil)
    copy(frame.Regs, args)
    for i, cap := range c.Captures {
        frame.Regs[int(proto.NumParams)+i] = cap
    }
    vm.Stack = append(vm.Stack, frame)
    result, err := vm.run(frame)
    // Frame is already popped by OpReturn in vm.run
    return result, err
}
```

**Key insight**: `vm.run(frame)` is re-entrant. When the closure calls `OpReturn`, it returns from `vm.run` with the result. The frame is already popped from `vm.Stack` by `OpReturn` when `caller == nil`. We set `frame.Caller = nil` so OpReturn knows this is a "top-level" call that should return the value directly.

### New Opcode

```go
// internal/bytecode/opcodes.go
OpBuiltinCallHOF  // A=dst, B=hofBuiltinIdx, C=argc — like OpBuiltinCall but passes VM

// internal/vm/vm.go (dispatch loop addition)
case bytecode.OpBuiltinCallHOF:
    hofIdx := int(inst.B())
    argc := int(inst.C())
    if hofIdx >= len(HOFBuiltinTable) {
        return ..., vm.errAt(frame, "unknown HOF builtin index", inst)
    }
    argBase := int(inst.A()) + 1
    args := frame.Regs[argBase : argBase+argc]
    result, err := HOFBuiltinTable[hofIdx](vm, args)
    if err != nil {
        return ..., vm.errAt(frame, fmt.Sprintf("HOF_BUILTIN_CALL: %v", err), inst)
    }
    frame.Regs[inst.A()] = result
    frame.IP++
```

### Compiler Changes

```go
// internal/bytecode/compiler/builtins.go

// HOFBuiltinTable lists builtins that take closure arguments.
var HOFBuiltinTable = []string{
    "__list_map",
    "__list_filter",
    "__list_foldl",
    "__str_foldChars",
    "__str_foldSlices",
    "__str_mapSlicesJoin",
}

// In compileBuiltinCall:
// 1. Check BuiltinTable (pure) → emit OpBuiltinCall
// 2. Check HOFBuiltinTable (HOF) → emit OpBuiltinCallHOF
// 3. Neither → compile error (EvalOnly fallback)
```

### HOF Builtin Implementations

Example — `__list_map`:

```go
func hofBuiltinListMap(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
    if len(args) != 2 {
        return bytecode.Value{}, fmt.Errorf("__list_map: expected 2 args")
    }
    fn := args[0]   // closure: a -> b
    list := args[1]  // [a]
    if list.Tag != bytecode.TagList {
        return bytecode.Value{}, fmt.Errorf("__list_map: arg 1 must be list")
    }
    elems := list.AsList()
    result := make([]bytecode.Value, len(elems))
    for i, e := range elems {
        val, err := caller.CallClosure(fn, []bytecode.Value{e})
        if err != nil {
            return bytecode.Value{}, fmt.Errorf("__list_map: element %d: %w", i, err)
        }
        result[i] = val
    }
    return bytecode.NewList(result), nil
}
```

Example — `__list_foldl`:

```go
func hofBuiltinListFoldl(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error) {
    if len(args) != 3 {
        return bytecode.Value{}, fmt.Errorf("__list_foldl: expected 3 args")
    }
    fn := args[0]    // closure: (b, a) -> b
    acc := args[1]   // b (initial accumulator)
    list := args[2]  // [a]
    if list.Tag != bytecode.TagList {
        return bytecode.Value{}, fmt.Errorf("__list_foldl: arg 2 must be list")
    }
    var err error
    for _, e := range list.AsList() {
        acc, err = caller.CallClosure(fn, []bytecode.Value{acc, e})
        if err != nil {
            return bytecode.Value{}, fmt.Errorf("__list_foldl: %w", err)
        }
    }
    return acc, nil
}
```

### Files to Modify

| File | Change | Est. LOC |
|------|--------|----------|
| `internal/bytecode/opcodes.go` | Add `OpBuiltinCallHOF` | 5 |
| `internal/bytecode/compiler/builtins.go` | Add `HOFBuiltinTable`, update `compileBuiltinCall` | 40 |
| `internal/vm/vm.go` | Add `CallClosure`, add `OpBuiltinCallHOF` dispatch | 40 |
| `internal/vm/builtins.go` | Add `ClosureCaller`, `HOFBuiltinFunc`, `HOFBuiltinTable` | 15 |
| `internal/vm/builtins_hof.go` | 6 HOF builtin implementations | 150 |
| `internal/vm/builtins_hof_test.go` | Unit tests | 100 |
| Total | | ~350 |

## Risks and Mitigations

### R1: Re-entrant vm.run() stack depth
**Risk**: HOF builtins call `vm.CallClosure` which calls `vm.run()` recursively. Deep nesting (map inside foldl inside map) could cause Go stack overflow.
**Mitigation**: The existing `vm.MaxStack = 1000` limit protects against this. Each `CallClosure` pushes a frame. The Go stack can handle 1000 nested function calls easily.

### R2: EvalOnly closures passed to HOF builtins
**Risk**: A lambda that failed to compile (e.g., it uses an effectful builtin) is marked EvalOnly. If passed as a callback to `__list_map`, CallClosure must bridge it.
**Mitigation**: `CallClosure` checks `proto.EvalOnly` and dispatches through `vm.Interop` if set — exactly like `OpCall` does. This is a graceful degradation, not a failure.

### R3: Frame lifecycle in CallClosure
**Risk**: `vm.run(frame)` modifies `vm.Stack`. When the closure returns via `OpReturn`, it pops the frame. We must ensure the HOF builtin's caller frame is correctly restored.
**Mitigation**: `CallClosure` sets `frame.Caller = nil`, so `OpReturn` returns the value to `vm.run` which returns it to `CallClosure`. The HOF builtin's caller frame is never modified — it's the frame that was active when `OpBuiltinCallHOF` dispatched, and it resumes after the builtin returns.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | HOF builtins produce deterministic results — same inputs, same outputs |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No new effects introduced |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No impact on type checking |
| A6: Safe Concurrency | 0 | No concurrency changes — VM is single-threaded |
| A7: Machines First | +1 | Reduces EvalOnly count, enabling faster VM execution |
| A8: Minimal Syntax | +1 | No new syntax — internal VM plumbing only |
| A9: Cost Visibility | 0 | No impact on cost model |
| A10: Composability | +1 | HOF builtins compose with closures naturally |
| A11: Structured Failure | 0 | Errors remain typed VMError |
| A12: System Boundary | 0 | No boundary changes |

**Net score: +4** (no violations)

## Design Freeze Checklist

- [x] D1: Extended BuiltinFunc via ClosureCaller interface (Option A)
- [x] D2: Separate opcode OpBuiltinCallHOF with dedicated table (D2a)
- [ ] Implementation approved by user

## Deferred Decisions

- **Performance optimization**: If profiling shows per-element closure call overhead is significant, consider Option B (dedicated opcodes) for hot-path builtins like `__list_map`. This is an optimization, not a correctness concern.
- **Tail-call optimization in HOF closures**: If the closure passed to `foldl` makes a tail call, the current design doesn't optimize it (it still allocates a frame per iteration). This could be addressed by a `CallClosureTailOpt` variant.

## Related Documents

- [M-BYTECODE-VM](../../implemented/v0_11_0/m-bytecode-vm.md) — Parent design doc (§18.6 results, §18.7 next steps)
- [M-BYTECODE-STDLIB-BUILTINS sprint plan](m-bytecode-stdlib-builtins-sprint-plan.md) — Previous sprint (55 pure builtins)
- [M-PERF4 Bytecode Interpreter](../../planned/v1_0_0/m-perf4-bytecode-interpreter.md) — Original bytecode concept
