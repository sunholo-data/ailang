# M-PERF4: Bytecode Interpreter (Stretch Goal)

**Status**: Planned (Uncertain)
**Target**: v0.9.0+ (stretch goal - may be deferred to v2.0)
**Priority**: P3 (Exploratory)
**Estimated**: 4-6 weeks (major rewrite)
**Dependencies**: M-PERF3 (Quick Wins) should be completed first

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Execution order unchanged, just faster |
| A2: Replayability | 0 | Traces unchanged (same semantics) |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Type checking unchanged |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Dramatically faster compilation/execution |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Faster = more predictable costs |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | 0 | Error handling unchanged |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Worth investigating**

### Hard Violation Check

- [x] A1 (Determinism): Same semantics, different execution model
- [x] A3 (Effects): Effects still explicit and tracked
- [x] A4 (Authority): Capabilities unchanged
- [x] A7 (Machines First): Major improvement for AI tooling

## Why This Is a Stretch Goal

**Reasons for uncertainty:**

1. **Major architecture change** - Requires rewriting the entire evaluation pipeline
2. **Risk of regressions** - Could introduce subtle semantic differences
3. **M-PERF3 may be sufficient** - Quick wins might achieve acceptable performance
4. **Go codegen alternative** - `ailang compile --emit-go` already provides native speed
5. **Development time** - 4-6 weeks is significant investment

**Decision criteria for proceeding:**

- [ ] M-PERF3 completed and benchmarked
- [ ] Still >3x slower than Python after quick wins
- [ ] User demand for faster interpreted execution
- [ ] Available development bandwidth

## Problem Statement

**Current Architecture (Tree-Walking Interpreter):**

```
Source → Parse → Elaborate → Type Check → Core AST → EVALUATE (traverse tree)
                                                          ↓
                                              For each node:
                                              1. Switch on type
                                              2. Recurse into children
                                              3. Allocate new values
                                              4. Clone environments
```

**Benchmark Results (December 2025):**

| Implementation | fib(25) Time | vs Native Go |
|----------------|--------------|--------------|
| Native Go | 3ms | baseline |
| Python (bytecode) | 48ms | 16x slower |
| **AILANG (tree-walk)** | **260ms** | **87x slower** |

**Why tree-walking is slow:**

1. **Pointer chasing**: Every node requires following pointers through the AST
2. **Type switches**: Go type switches have overhead (~10-20ns per switch)
3. **Allocations**: Each call creates new closures, environments, values
4. **No instruction locality**: CPU cache misses from random memory access
5. **No tail-call optimization**: Stack grows linearly with recursion depth

## Solution Design

### Bytecode Compilation

**Proposed Architecture:**

```
Source → Parse → Elaborate → Type Check → Core AST → COMPILE → Bytecode → VM
                                                         ↓
                                              Linear instruction stream:
                                              [PUSH_INT 10]
                                              [LOAD_VAR 0]
                                              [CALL 1]
                                              [ADD]
                                              [RETURN]
```

### Instruction Set (Draft)

```go
type OpCode byte

const (
    // Stack operations
    OP_PUSH_INT OpCode = iota   // Push integer literal
    OP_PUSH_FLOAT               // Push float literal
    OP_PUSH_STRING              // Push string literal
    OP_PUSH_BOOL                // Push boolean literal
    OP_PUSH_UNIT                // Push unit value
    OP_POP                      // Pop top of stack
    OP_DUP                      // Duplicate top of stack

    // Variable operations
    OP_LOAD_LOCAL               // Load local variable by index
    OP_STORE_LOCAL              // Store to local variable
    OP_LOAD_UPVALUE             // Load from closure
    OP_STORE_UPVALUE            // Store to closure
    OP_LOAD_GLOBAL              // Load global/builtin

    // Control flow
    OP_JUMP                     // Unconditional jump
    OP_JUMP_IF_FALSE            // Conditional jump
    OP_CALL                     // Function call
    OP_TAIL_CALL                // Tail-recursive call
    OP_RETURN                   // Return from function

    // Arithmetic
    OP_ADD                      // Integer/float addition
    OP_SUB                      // Subtraction
    OP_MUL                      // Multiplication
    OP_DIV                      // Division
    OP_NEG                      // Negation

    // Comparison
    OP_EQ                       // Equality
    OP_LT                       // Less than
    OP_GT                       // Greater than
    OP_LE                       // Less or equal
    OP_GE                       // Greater or equal

    // Data structures
    OP_MAKE_CLOSURE             // Create closure
    OP_MAKE_RECORD              // Create record
    OP_MAKE_LIST                // Create list
    OP_MAKE_TUPLE               // Create tuple
    OP_GET_FIELD                // Record field access
    OP_GET_INDEX                // List/tuple index

    // Pattern matching
    OP_MATCH_TAG                // Match ADT constructor
    OP_MATCH_LIT                // Match literal
    OP_MATCH_FAIL               // Pattern match failure

    // Effects
    OP_EFFECT_CALL              // Call effect handler
    OP_EFFECT_RESUME            // Resume from handler
)
```

### VM Architecture

```go
type VM struct {
    stack     []Value      // Operand stack
    frames    []CallFrame  // Call stack
    globals   []Value      // Global bindings
    constants []Value      // Constant pool
    code      []byte       // Bytecode
    ip        int          // Instruction pointer
}

type CallFrame struct {
    closure  *Closure     // Current function
    ip       int          // Return address
    basePtr  int          // Stack base for this frame
}

func (vm *VM) Run() (Value, error) {
    for {
        op := OpCode(vm.code[vm.ip])
        vm.ip++

        switch op {
        case OP_PUSH_INT:
            val := vm.readInt64()
            vm.push(IntValue(val))

        case OP_ADD:
            b := vm.pop()
            a := vm.pop()
            vm.push(a.Add(b))

        case OP_CALL:
            argCount := vm.readByte()
            vm.call(argCount)

        case OP_RETURN:
            result := vm.pop()
            if len(vm.frames) == 0 {
                return result, nil
            }
            vm.return(result)

        // ... other opcodes
        }
    }
}
```

### Compiler Changes

**New compiler pass (Core → Bytecode):**

```
internal/
├── bytecode/
│   ├── opcodes.go      # Instruction definitions
│   ├── compiler.go     # Core AST → Bytecode compiler
│   ├── chunk.go        # Bytecode container
│   └── disasm.go       # Disassembler for debugging
├── vm/
│   ├── vm.go           # Virtual machine
│   ├── value.go        # Runtime values
│   ├── stack.go        # Operand stack
│   └── frame.go        # Call frames
```

### Expected Performance

**Target metrics:**

| Metric | Tree-Walking | Bytecode | Improvement |
|--------|--------------|----------|-------------|
| fib(25) | 260ms | ~30-50ms | 5-8x faster |
| vs Python | 5x slower | ~same speed | Parity |
| Memory | High (allocations) | Low (stack-based) | 2-3x less |

**Why bytecode is faster:**

1. **Linear execution**: No pointer chasing, better cache locality
2. **No type switches**: Direct opcode dispatch (computed goto or switch)
3. **Stack-based**: Values on stack, minimal allocations
4. **Tail-call optimization**: Constant stack space for recursion

## Implementation Plan

**Phase 1: Core VM** (~2 weeks)
- [ ] Define opcode set
- [ ] Implement basic VM loop
- [ ] Stack operations (push, pop, arithmetic)
- [ ] Local variables
- [ ] Control flow (jumps, calls, returns)

**Phase 2: Compiler** (~2 weeks)
- [ ] Compile Core expressions to bytecode
- [ ] Handle closures and upvalues
- [ ] Compile pattern matching
- [ ] Compile effect handlers

**Phase 3: Integration** (~1 week)
- [ ] Wire bytecode path into pipeline
- [ ] Add `--bytecode` flag to `ailang run`
- [ ] Benchmark against tree-walking
- [ ] Debug and optimize

**Phase 4: Polish** (~1 week)
- [ ] Disassembler for debugging
- [ ] Error messages with source locations
- [ ] Documentation
- [ ] Performance tuning

## Alternatives Considered

### 1. JIT Compilation

**Pros:**
- Even faster than bytecode interpreter
- Native code performance

**Cons:**
- Much more complex (code generation, memory management)
- Platform-specific
- Security considerations

**Decision:** Out of scope for v0.9.0. Consider for v2.0+.

### 2. Transpile to Go and Compile

**Already exists:** `ailang compile --emit-go`

**Pros:**
- Native Go performance
- Already implemented

**Cons:**
- Requires Go toolchain
- Compilation overhead
- Not suitable for REPL/interactive use

**Decision:** Keep as alternative. Bytecode is for fast interpreted execution.

### 3. Use Existing VM (WASM, Lua, etc.)

**Pros:**
- Mature, well-tested VMs
- Community support

**Cons:**
- Impedance mismatch with AILANG semantics
- Effect system integration challenges
- Dependency on external runtime

**Decision:** Custom VM better fits AILANG's unique requirements.

## Success Criteria

- [ ] fib(25) completes in <60ms (down from 260ms)
- [ ] Performance within 1.5x of Python for recursive workloads
- [ ] All existing tests pass under bytecode execution
- [ ] REPL works with bytecode backend
- [ ] `--bytecode` flag available on `ailang run`

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Semantic differences | High | Extensive test suite, property tests |
| Effect system complexity | Medium | Design effects into VM from start |
| Development time overrun | Medium | Time-box phases, cut scope if needed |
| Not worth the investment | Low | M-PERF3 quick wins first, evaluate need |

## Non-Goals

**Explicitly out of scope:**

- JIT compilation
- LLVM backend
- WebAssembly output
- Parallel VM execution
- Debugging support beyond basic disassembly

## Related Documents

**Prerequisites:**
- [M-PERF3: Performance Quick Wins](../v0_7_0/m-perf3-performance-quick-wins.md) - Complete first

**Background:**
- [perf-reviewer skill](.claude/skills/perf-reviewer/) - Benchmark tooling
- [Crafting Interpreters](https://craftinginterpreters.com/) - Reference implementation

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-12-29 | Marked as stretch goal | Major investment, uncertain ROI |
| 2025-12-29 | Target v0.9.0 (not v0.7.0) | Allow M-PERF3 quick wins first |
| 2025-12-29 | May defer to v2.0 | Depends on benchmark results after M-PERF3 |

---

**Document created**: 2025-12-29
**Last updated**: 2025-12-29
