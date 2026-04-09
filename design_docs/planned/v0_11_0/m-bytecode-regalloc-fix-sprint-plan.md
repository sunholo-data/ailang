# M-BYTECODE-REGALLOC-FIX Sprint Plan

**Design doc**: [m-bytecode-regalloc-fix.md](m-bytecode-regalloc-fix.md)
**Sprint ID**: `M-BYTECODE-REGALLOC-FIX`
**Estimated**: 0.5 day (~300 LOC)
**Dependencies**: M-BYTECODE-PURE-EFFECTS (complete)

## Milestones

### M1: Diagnose and Fix NumRegs Undercount (9 prototypes)

**Scope**: Find and fix the root cause of 9 prototypes where compiled bytecode uses registers beyond NumRegs.

**Approach**:
1. Add temporary debug assertion in `funcCompiler.emit()` that checks all register operands < `regs.next`
2. Compile docparse with assertions enabled — the assertion will identify the exact codepath
3. Fix the root cause (likely 1-20 lines)
4. Remove the debug assertion (or keep as `DEBUG_STRICT=1` check)
5. Verify all 9 prototypes now compile

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/bytecode/compiler/compiler.go` | Debug assertion in emit() | 30 |
| TBD (based on diagnosis) | Root cause fix | 5-20 |
| `internal/bytecode/compiler/regalloc_test.go` | Regression test | 30 |

**Acceptance criteria**:
- Root cause identified and documented
- All 9 "exceeds NumRegs" prototypes compile successfully
- Regression test covers the specific pattern
- `make test` passes
- `make lint` passes

**Est. LOC**: 60-80

---

### M2: Scope-Aware Register Recycling (4 prototypes)

**Scope**: Implement register recycling when scopes are popped, fixing the 4 register-overflow prototypes.

**Approach**:
1. Add `pinnedRegs []uint8` to each scope frame in `scopeStack`
2. Add `allocScoped(name string) (uint8, error)` method that calls `allocPinned` and records the register in the current scope
3. Modify `pop()` to call `freeTemp()` for all registers pinned in the popped scope
4. Update `switch.go` case bindings and `compiler.go` let bindings to use `allocScoped`
5. Optionally: make `allocContig` scan free list for adjacent slots before bumping

**Files**:
| File | Change | Est. LOC |
|------|--------|----------|
| `internal/bytecode/compiler/regalloc.go` | scopeFrame.pinnedRegs, allocScoped, pop() releases | 60 |
| `internal/bytecode/compiler/regalloc_test.go` | Unit tests for scope recycling | 80 |
| `internal/bytecode/compiler/switch.go` | Use allocScoped for case bindings | 5 |
| `internal/bytecode/compiler/compiler.go` | Use allocScoped for let bindings in nested scopes | 10 |

**Acceptance criteria**:
- All 4 "exceeds 256" prototypes compile successfully (or ≤ 1 remaining)
- Existing tests pass (no regressions from register recycling)
- New unit tests cover: scope push/pop recycling, allocContig after recycling
- `make test` passes
- `make lint` passes

**Est. LOC**: 155

---

### M3: Verify and Close

**Scope**: Run full docparse disasm, verify EvalOnly reduction, update docs.

**Acceptance criteria**:
- `ailang disasm docparse/main.ail` EvalOnly count ≤ 76 (down from 89)
- Parity: ≥ 129 MATCH, no regressions
- CHANGELOG updated
- Design doc updated with results

**Dependencies**: M1, M2
**Est. LOC**: 0 (documentation only)
