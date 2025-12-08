# Sprint Plan: M-BUG-LOCAL-IMPORTS - Fix Local Module Import Resolution

## Summary
Fix the local module import resolution bug that causes `LDR001: module not found` errors when importing from subdirectories (e.g., `import sim/protocol` from `sim/world.ail`). This is blocking multi-file project development for external users.

**Duration:** 1 day (4-6 hours)
**Dependencies:** None
**Risk Level:** Low (isolated fix, clear root cause identified)
**Design Doc:** [m-bug-local-imports.md](m-bug-local-imports.md)

## Bug Verification

**Confirmed:** The bug was reproduced with minimal test case:
```
/tmp/import-test/
├── sim/
│   ├── protocol.ail   # module sim/protocol - exports type Coord
│   └── world.ail      # module sim/world - imports sim/protocol (Coord)
```

Running `cd /tmp/import-test && ailang run --entry main sim/world.ail` produces:
```
→ Type checking...
→ Effect checking...
✓ Running sim/world.ail
Error: LDR001: module not found: sim/protocol
```

**Note:** Same-directory imports work fine (`import mylib` from `main.ail` where `mylib.ail` is in same directory).

## Root Cause Analysis

**Primary Issue:** In `cmd/ailang/main.go:377`, the ModuleRuntime is created with:
```go
rt := runtime.NewModuleRuntime(filepath.Dir(filename))
```

For `ailang run sim/world.ail`:
- `filename = "sim/world.ail"`
- `filepath.Dir(filename) = "sim"`
- Runtime's `basePath = "sim"`

When loading dependencies, `sim/protocol` resolves to:
```go
filepath.Join("sim", "sim/protocol") + ".ail" = "sim/sim/protocol.ail"
```

This path doesn't exist! The correct path should be `./sim/protocol.ail`.

**Secondary Issue:** The error format `"LDR001: module not found: sim/protocol"` comes from `module_linker.go:63`, which doesn't show the search trace for debugging.

## Proposed Milestones

### Milestone 1: Fix Runtime BasePath (M1-RUNTIME-BASEPATH)
**Goal:** Use project root (CWD) as basePath for ModuleRuntime, not file directory
**Estimated:** 10 LOC change + 30 LOC tests = ~40 LOC
**Duration:** 2 hours

**Files to Modify:**
- `cmd/ailang/main.go:377` - Change `filepath.Dir(filename)` to `"."`

**Before:**
```go
rt := runtime.NewModuleRuntime(filepath.Dir(filename))
```

**After:**
```go
// Use project root (CWD) for module resolution, not file directory
// This ensures imports like "sim/protocol" resolve from project root
rt := runtime.NewModuleRuntime(".")
```

**Tasks:**
1. Modify `cmd/ailang/main.go:377` to use `"."` as basePath
2. Add comment explaining why CWD is used
3. Verify fix with reproduction case

**Acceptance Criteria:**
- [ ] `ailang run sim/world.ail` from project root works when `sim/protocol.ail` exists
- [ ] Nested directory imports work (`sim/core/types` from `sim/entities/npc.ail`)
- [ ] Existing tests still pass

### Milestone 2: Add Integration Test (M2-INTEGRATION-TEST)
**Goal:** Ensure regression test prevents this bug from returning
**Estimated:** 80 LOC tests + 20 LOC test fixtures = ~100 LOC
**Duration:** 2 hours

**Files to Create:**
- `tests/integration/multi_module_test.go` - Integration test
- `tests/integration/fixtures/multi_module/` - Test project structure

**Test Structure:**
```
tests/integration/fixtures/multi_module/
├── lib/
│   └── types.ail        # export type Coord = { x: int, y: int }
├── game/
│   ├── player.ail       # import lib/types (Coord)
│   └── main.ail         # import game/player, lib/types
└── main.ail             # import game/main - entry point
```

**Tasks:**
1. Create test fixture directory structure
2. Write test AILANG files with cross-directory imports
3. Create Go integration test that:
   - Changes to fixture directory
   - Runs `ailang run --entry main main.ail`
   - Verifies successful execution
   - Tests error case (import non-existent module)

**Acceptance Criteria:**
- [ ] Test runs as part of `make test`
- [ ] Test passes with the fix
- [ ] Test would have caught original bug

### Milestone 3: Improve Error Message (M3-BETTER-ERROR)
**Goal:** Include search paths in LDR001 error for easier debugging
**Estimated:** 15 LOC = ~15 LOC
**Duration:** 30 minutes

**Files to Modify:**
- `internal/link/module_linker.go:63` - Add search trace to error

**Before:**
```go
return nil, diag, fmt.Errorf("LDR001: module not found: %s", imp.Path)
```

**After:**
```go
return nil, diag, fmt.Errorf("LDR001: module not found: %s\nSearch path: %s\nSuggestions: %s",
    imp.Path, ml.loader.GetBasePath(), strings.Join(diag.Suggestions, ", "))
```

**Tasks:**
1. Add `GetBasePath()` method to ModuleLoader if not exists
2. Update error message to include search path
3. Update any tests that check exact error message format

**Acceptance Criteria:**
- [ ] Error message shows what path was searched
- [ ] Easier to debug import resolution issues

### Milestone 4: Create Example Project (M4-EXAMPLE)
**Goal:** Add multi-module example to examples/ for documentation
**Estimated:** 40 LOC = ~40 LOC
**Duration:** 30 minutes

**Files to Create:**
- `examples/multi_module/README.md` - Usage instructions
- `examples/multi_module/types.ail` - Shared types
- `examples/multi_module/main.ail` - Entry point that imports types

**Tasks:**
1. Create minimal multi-module example
2. Verify it works with `ailang run --entry main examples/multi_module/main.ail`
3. Add to examples verification list

**Acceptance Criteria:**
- [ ] Example runs successfully
- [ ] `make verify-examples` includes this example
- [ ] Serves as documentation for multi-file projects

## Success Metrics
- Test coverage: Maintained (no regression)
- All existing tests passing
- New integration test passing
- Multi-module example working
- stapledons_voyage project should work after fix (manual verification)

## Velocity Analysis
Based on recent work:
- v0.4.8 features: ~200 LOC/day (aliasing, testing)
- This is a focused bug fix with clear scope
- Estimated total: ~195 LOC across 4 milestones
- Should complete in 4-6 hours

## Open Questions
None - root cause is clear and fix is straightforward.

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Changing basePath breaks existing behavior | Medium | Run full test suite; fix is localized |
| Edge cases with stdlib imports | Low | Stdlib uses `std/` prefix, resolved separately |
| Windows path separators | Low | Already using `filepath.Join` throughout |

## Notes
- This is a P0 bug blocking external game development (stapledons_voyage project)
- The fix is minimal and focused - single line change plus tests
- Error message improvement is nice-to-have but not blocking

---

**Created:** 2025-11-29
**Status:** Ready for execution
