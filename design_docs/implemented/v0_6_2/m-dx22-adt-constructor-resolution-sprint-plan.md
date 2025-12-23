# M-DX22 Sprint Plan: ADT Constructor Resolution

**Sprint ID**: M-DX22
**Duration**: Single session (~3 hours)
**Goal**: Fix ADT constructor resolution to use type-qualified names
**Risk Level**: Medium (touches 6 codegen files)

---

## Sprint Summary

Fix the `adtConstructors` map to use type-qualified keys (`TypeName.CtorName`) instead of just constructor names. This prevents same-named constructors in different ADT types from colliding.

**Deliverables:**
1. Registry key format changed to `TypeName.CtorName`
2. All lookup sites updated (6 files)
3. Tests pass, stapledons_voyage compiles

---

## Velocity Context

**Recent work (today):**
- M-DX18: Function namespacing (~65 LOC)
- M-DX17: concat → ConcatList fix (~20 LOC)

**This sprint estimate:**
- ~75 LOC across 6 files
- Achievable in single session

---

## Milestones

### M1: Change Registry Keys (~1 hour, ~30 LOC)

**Description:** Modify ADT constructor registration to use `TypeName.CtorName` as the map key.

**Tasks:**
- [ ] Locate `registerADTConstructor` in `codegen.go`
- [ ] Modify key format to `TypeName + "." + CtorName`
- [ ] Update `RegisterADTConstructorWithFieldTypes` similarly
- [ ] Verify ADT type name is available during registration

**Files:**
- `internal/gen/golang/codegen.go` (~20 LOC)

**Acceptance Criteria:**
- [ ] ADT constructors registered with qualified keys
- [ ] `make test` passes

---

### M2: Update Lookup Sites (~1.5 hours, ~40 LOC)

**Description:** Update all locations that look up constructors by name to use qualified lookup.

**Tasks:**
- [ ] `codegen_expr_simple.go:95,105` - VarGlobal lookups
- [ ] `codegen_expr_app.go:261,267` - App expression lookups
- [ ] `codegen_match.go:78,513` - Pattern matching lookups
- [ ] `codegen_ops.go:448,453,463,468` - Operator lowering
- [ ] `codegen_decl.go:502,521,529,546` - Declaration processing

**Files:**
- `internal/gen/golang/codegen_expr_simple.go` (~10 LOC)
- `internal/gen/golang/codegen_expr_app.go` (~10 LOC)
- `internal/gen/golang/codegen_match.go` (~10 LOC)
- `internal/gen/golang/codegen_ops.go` (~5 LOC)
- `internal/gen/golang/codegen_decl.go` (~5 LOC)

**Acceptance Criteria:**
- [ ] All lookup sites use qualified names
- [ ] Fallback to unqualified lookup if needed
- [ ] `make test` passes

---

### M3: Testing & Validation (~30 min, ~5 LOC)

**Description:** Verify the fix works for same-named constructors.

**Tasks:**
- [ ] Create test case with two ADTs having same-named constructors
- [ ] Verify generated Go compiles
- [ ] Run full test suite
- [ ] Update design doc status

**Acceptance Criteria:**
- [ ] Test with same-named constructors passes
- [ ] All existing tests pass
- [ ] `make lint` passes

---

## Success Metrics

- [ ] Two ADTs with same-named constructors compile correctly
- [ ] stapledons_voyage sim/*.ail compiles (pending confirmation)
- [ ] No regressions in existing codegen tests
- [ ] `make test && make lint` passes

---

## Dependencies

- None (standalone fix)

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Type context not available | Check VarGlobal.Ref.Module for context |
| Breaking existing tests | Run tests after each file change |
| Missing lookup site | grep for `adtConstructors` to find all |

---

**Document created**: 2025-12-21
