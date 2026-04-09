# M-BYTECODE-REGALLOC-FIX: Fix Register Allocation Bugs

**Status**: Planned
**Priority**: P1 (13 EvalOnly prototypes, pure compiler work, no new VM types)
**Target**: v0.11.0
**Estimated LOC**: ~300
**Dependencies**: M-BYTECODE-PURE-EFFECTS (complete)

## 1. Problem Statement

After M-BYTECODE-PURE-EFFECTS, **89 / 1204** prototypes remain EvalOnly. Of these, **13 are caused by register allocation bugs** — the next-cheapest win after effectful builtins (which require new VM types or EffContext plumbing).

Two distinct bug categories:

### Category A: NumRegs Undercount (9 prototypes)

The compiled bytecode references registers that exceed the prototype's `NumRegs` field. Image validation catches this post-compilation and marks the parent function EvalOnly.

**Affected prototypes:**
| Prototype | Function | Error |
|-----------|----------|-------|
| 223 | eml_parser.emlFindDescendants | lambda: r3 exceeds NumRegs=3 |
| 266 | eval.evalCollectWords | lambda: r6 exceeds NumRegs=4 |
| 271 | eval.evalCheckTables | lambda: r7 exceeds NumRegs=3 |
| 272 | eval.evalCheckTrackChanges | lambda: r7 exceeds NumRegs=3 |
| 274 | eval.evalCheckComments | lambda: r5 exceeds NumRegs=3 |
| 275 | eval.evalCheckHeadersFooters | lambda: r5 exceeds NumRegs=3 |
| 276 | eval.evalCheckTextBoxes | lambda: r5 exceeds NumRegs=3 |
| 277 | eval.evalCheckImages | lambda: r7 exceeds NumRegs=3 |
| 804 | document.cellText | r3 exceeds NumRegs=2 |

**Root cause hypothesis**: A codepath in the compiler emits instructions that reference registers outside the allocator's tracking. The `regAlloc.highWater()` faithfully reports what was allocated, but some instruction operand uses a register that was never allocated via `allocPinned`, `allocTemp`, or `allocContig`. Likely candidates:

1. `compileExprIntoSlot` writes to a caller-supplied `dst` register — if the caller computed `dst` arithmetically (e.g., `base + uint8(i)`) but only allocated the base, the offset registers bypass the allocator. However, inspection shows all current callers use `allocContig` first. The bug may be in a more subtle path.

2. The lambda's GET_FIELD instruction at ip 0 uses registers that were set by the OUTER function's capture setup but exceed the INNER prototype's NumRegs. This would mean the inner `funcCompiler` didn't allocate enough pinned registers for captures or params.

3. A match/switch binding path allocates registers that are scoped (via `locals.push/pop`) but the register is used after the scope is popped — the allocator still tracks the high-water mark, but some other interaction causes the undercount.

**Investigation approach**: Add a debug assertion in `funcCompiler.emit()` that checks every register operand in the emitted instruction is < `regs.next`. This will catch the exact codepath that emits an out-of-bounds register. Then fix the root cause.

### Category B: 256-Register Ceiling Overflow (4 prototypes)

Large functions exhaust the 256-register limit during `allocContig` calls.

**Affected prototypes:**
| Prototype | Function | Error |
|-----------|----------|-------|
| 419 | mcp/tools.mcpGenerate | contiguous block of 2 would exceed 256 |
| 423 | mcp/tools.mcpFormats | contiguous block of 5 would exceed 256 |
| 519 | output_formatter.blockToJson | contiguous block of 2 would exceed 256 |
| 653 | samples.samples | contiguous block of 7 would exceed 256 |

**Root cause**: The bump allocator never reclaims pinned registers from completed scopes. In functions with many `let` bindings or long sequences of calls, each let-bound variable pins a register forever. The `allocContig` path bypasses the free list entirely (it needs contiguous fresh registers), so even freed temps don't help.

**Fix approaches** (in order of complexity):

1. **Scope-aware register recycling** (~100 LOC): When `scopeStack.pop()` is called, release the registers pinned in that scope back to the free list. This requires tracking which registers were pinned in each scope frame (add a `[]uint8` to each scope frame). This alone may be sufficient — the 4 affected functions have many scoped let bindings.

2. **Contiguous-from-freelist** (~50 LOC): Make `allocContig` scan the free list for N adjacent slots before bumping. Currently it always bumps, wasting any freed temps in the gap. This is a complementary optimization.

3. **Register spilling** (~500 LOC): Full spilling with OpSpill/OpRestore instructions. Likely overkill for this sprint but noted for completeness.

## 2. Why This Sprint

1. **Pure compiler work** — No new VM types, no bridge changes, no runtime modifications.
2. **13 prototypes at once** — Second-highest density after effectful builtins (73), but dramatically less effort.
3. **Validates allocator correctness** — Fixing the NumRegs undercount is a correctness bug, not just a performance issue. These prototypes may crash the VM if they weren't caught by validation.
4. **Unblocks future work** — Better register reuse reduces pressure for all future builtins.

## 3. Implementation Plan

### Phase 1: Diagnose NumRegs Undercount

Add a debug assertion in `funcCompiler.emit()` that validates all register operands in the instruction are < `regs.next`. Compile docparse and observe which codepath triggers.

### Phase 2: Fix NumRegs Bug

Fix the root cause based on Phase 1 findings. Expected: 1-3 lines in a specific compilation path.

### Phase 3: Scope-Aware Register Recycling

Modify `scopeStack` to track registers pinned per scope frame. When `pop()` is called, release them to `regAlloc.freeTemp()`. Update `allocPinned()` call sites (switch bindings, let bindings in nested scopes) to use a new `allocScoped()` method that records the register in the current scope frame.

### Phase 4: Contiguous-from-Freelist (Optional)

If Phase 3 doesn't fix all 4 overflow prototypes, implement contiguous allocation from the free list.

## 4. Files

| File | Change | Est. LOC |
|------|--------|----------|
| `internal/bytecode/compiler/regalloc.go` | Scope-aware recycling, allocScoped | 80 |
| `internal/bytecode/compiler/regalloc_test.go` | Unit tests for recycling | 100 |
| `internal/bytecode/compiler/switch.go` | Use allocScoped for case bindings | 5 |
| `internal/bytecode/compiler/compiler.go` | Use allocScoped for let bindings | 10 |
| `internal/bytecode/compiler/lambda.go` | Fix NumRegs undercount (TBD) | 5-20 |
| Various compiler files | Possible emit-time assertion | 30 |

**Total estimated**: ~250-300 LOC

## 5. Acceptance Criteria

- All 9 "exceeds NumRegs" prototypes compile without validation errors
- All 4 "exceeds 256" prototypes compile without overflow errors (or reduce to ≤ 1)
- EvalOnly count: ≤ 76 (down from 89, saving 13)
- Parity: ≥ 129 MATCH, no regressions
- `make test` passes
- `make lint` passes
- Existing regalloc tests still pass
- New unit tests cover scope-aware recycling

## 6. Risks

- **NumRegs root cause may be deeper**: If the bug is in the Statement IR lowering rather than the compiler, the fix scope expands. Mitigation: Phase 1 diagnosis will reveal the exact location.
- **Scope recycling may break existing code**: Freeing pinned registers that are still referenced by later instructions would cause register reuse bugs. Mitigation: Only free registers when the scope that pinned them is popped, and validate with the full test suite + parity check.
