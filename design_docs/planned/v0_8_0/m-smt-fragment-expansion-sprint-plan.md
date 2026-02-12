# Sprint Plan: M-SMT-FRAGMENT-EXPANSION Phase A — Cross-Function Calls

**Sprint ID**: M-SMT-XFUNC
**Design Doc**: [m-smt-fragment-expansion.md](m-smt-fragment-expansion.md)
**Target**: v0.7.4+
**Duration**: 1 day (~6-8 hours)
**Risk Level**: Low

---

## Sprint Summary

Implement cross-function call verification in `ailang verify`. Currently, functions calling other user-defined functions (e.g., `netIncome` calling `calculateTax`) fail with Z3 "unknown constant" errors because each function is verified in isolation. This sprint adds `define-fun` emission for verified callees, enabling compositional verification.

**Why Phase A first?** It's the most common user complaint — example files must duplicate inline logic instead of calling helper functions. Fixing this unblocks natural AILANG coding patterns.

---

## Current State

- **SMT implementation**: 3,420 LOC across 8 files in `internal/smt/`
- **Verify command**: 498 LOC in `cmd/ailang/verify.go`
- **Existing tests**: All pass (codegen 870 LOC, encodable 456 LOC, solver 309 LOC, types 259 LOC)
- **Key limitation**: `encodeApp()` in `codegen.go:262-301` only handles builtins and ADT constructors — user-defined function calls fall through to error

---

## Milestones

### M1: Callee Resolver — Resolve and encode callee functions (~2h)

**What**: Create `callee_resolver.go` that, given a function call in the SMT encoder, finds the callee's Core body and emits a `define-fun` SMT-LIB declaration.

**Acceptance criteria**:
- [ ] `ResolveCallees(funcName, body, coreProg, adtTypes)` returns ordered list of callee `define-fun` declarations
- [ ] Topological ordering: if A calls B and B calls C, declarations are C → B → A
- [ ] Circular call detection with clear error message
- [ ] Functions not in the SMT fragment are skipped with warning
- [ ] Unit tests: direct call, transitive call, circular call, non-encodable callee

**Files**:
- Create `internal/smt/callee_resolver.go` (~180 LOC)
- Create `internal/smt/callee_resolver_test.go` (~250 LOC)

**Estimated LOC**: ~430

---

### M2: Integrate callee resolver into EncodeFunction (~1.5h)

**What**: Modify `codegen.go` and `verify.go` to use the callee resolver. Pass the full Core program to `EncodeFunction` so it can resolve cross-function calls.

**Acceptance criteria**:
- [ ] `EncodeFunction` accepts optional `*core.Program` parameter for callee resolution
- [ ] Callee `define-fun` declarations appear before the main function encoding
- [ ] `encodeApp()` recognizes user-defined function calls and encodes as SMT-LIB function application
- [ ] `verify.go` passes Core program to encoder
- [ ] Existing tests still pass (no regression)

**Files**:
- Modify `internal/smt/codegen.go` (+80 LOC)
- Modify `cmd/ailang/verify.go` (+15 LOC)
- Add tests to `internal/smt/codegen_test.go` (+100 LOC)

**Estimated LOC**: ~195

---

### M3: Update fragment checker (encodable.go) (~30min)

**What**: Update `IsSMTEncodable` to no longer reject functions just because they contain user-defined function calls. The rejection should only happen for calls to functions that are themselves not encodable AND have no contracts.

**Acceptance criteria**:
- [ ] Functions with user-defined calls are no longer rejected by `IsSMTEncodable`
- [ ] Body walk detects App(VarGlobal(...)) where module != "$builtin" and doesn't reject
- [ ] All existing `IsSMTEncodable` tests still pass
- [ ] New test: function with cross-function call is encodable

**Files**:
- Modify `internal/smt/encodable.go` (+20 LOC)
- Add tests to `internal/smt/encodable_test.go` (+40 LOC)

**Estimated LOC**: ~60

---

### M4: Update examples to use cross-function calls (~1h)

**What**: Update existing contract examples to use function calls instead of duplicated inline logic. Create a new example demonstrating cross-function verification.

**Acceptance criteria**:
- [ ] `finance.ail`: `netIncome` calls `calculateTax` (no more inline duplication)
- [ ] `access_control.ail`: `canAccess` calls `accessLevel` (no more inline duplication)
- [ ] New `examples/runnable/contracts/cross_function.ail` demonstrating compositional verification
- [ ] All examples verify correctly with `ailang verify`
- [ ] All examples run correctly with `ailang run`
- [ ] `examples/manifest.json` updated

**Files**:
- Modify `examples/runnable/contracts/finance.ail` (~-20 LOC)
- Modify `examples/runnable/contracts/access_control.ail` (~-15 LOC)
- Create `examples/runnable/contracts/cross_function.ail` (~80 LOC)
- Modify `examples/manifest.json` (+10 LOC)

**Estimated LOC**: ~55 net (lots of duplication removed)

---

### M5: Update documentation and prompts (~1h)

**What**: Update contracts.mdx to reflect the expanded fragment (cross-function calls now supported). Update the decidable fragment table.

**Acceptance criteria**:
- [ ] `contracts.mdx` "The Decidable Fragment" section updated
- [ ] Cross-function verification examples added to docs
- [ ] `prompts/v0.7.4.md` updated with cross-function call info (if exists)
- [ ] CHANGELOG.md updated

**Files**:
- Modify `docs/docs/guides/contracts.mdx` (+30 LOC)
- Modify `CHANGELOG.md` (+10 LOC)

**Estimated LOC**: ~40

---

## Success Metrics

| Metric | Target |
|--------|--------|
| All existing verify tests pass | ✅ |
| `finance.ail` netIncome→calculateTax verifies | ✅ |
| `access_control.ail` canAccess→accessLevel verifies | ✅ |
| New cross_function.ail verifies | ✅ |
| Circular call detection works | ✅ |
| Total new LOC | ~780 (impl + tests) |

---

## Velocity Reference

- Recent sprint M-SMT-BACKEND: ~3,420 LOC in ~2 days
- This sprint is much smaller: ~780 LOC in ~1 day
- Low risk: all Z3 theories already work, just plumbing

---

## Dependencies

- Z3 must be installed (`brew install z3`)
- All existing `internal/smt/` tests pass
- No external dependencies
