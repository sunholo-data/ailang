# M-WASM-REPL Sprint Plan: Browser Module Loading

**Sprint ID**: M-WASM-REPL
**Duration**: 3.5 days (28 hours)
**Risk Level**: Medium (new WASM API surface)
**Design Doc**: [m-wasm-repl-module-loading.md](m-wasm-repl-module-loading.md)

## Sprint Summary

**Goal**: Enable AILANG browser demos by adding module loading capability to WASM REPL.

**Key Deliverables**:
1. `window.ailangLoadModule(name, code)` JavaScript API
2. Import resolution from module registry in REPL
3. `AILANGWrapper` JavaScript class for easy integration
4. Working invoice processor demo

**Total LOC**: ~400 (300 implementation + 100 tests)

## Current Status

**Velocity Analysis** (last 14 days):
- ~630 LOC shipped (apiserver, telemetry, budget enforcement)
- Average: ~45 LOC/day
- Recent milestones: M-DX24, M-DX25, wasm foundations

**Blocking Issue**: Invoice processor demo shows "undefined variable: processInvoice" because REPL doesn't persist function definitions.

## Milestones

### M1: Module Registry Core (Day 1-2, ~150 LOC)

**Goal**: Create `ModuleRegistry` struct that stores compiled modules with their exports.

**Files**:
| File | LOC | Description |
|------|-----|-------------|
| `internal/repl/module_registry.go` | ~100 | Registry implementation |
| `internal/repl/module_registry_test.go` | ~50 | Unit tests |

**Tasks**:
- [ ] Create `ModuleRegistry` struct with `modules map[string]*RegisteredModule`
- [ ] Create `RegisteredModule` and `Export` types
- [ ] Implement `NewModuleRegistry()` constructor
- [ ] Implement `LoadModule(name, code string) error` with full pipeline
- [ ] Implement `GetExport(moduleName, funcName string) (*Export, error)`
- [ ] Implement `ListModules() []string` for debugging
- [ ] Write unit tests for registry operations
- [ ] Test error cases (parse error, type error, missing export)

**Acceptance Criteria**:
- [ ] `LoadModule` compiles AILANG source to evaluated closures
- [ ] Exports are retrievable by module + function name
- [ ] Clear error messages for compilation failures
- [ ] Tests pass: `go test ./internal/repl/... -v -run Registry`

**Risks**:
- Pipeline API may differ from design doc assumptions (mitigate: check existing REPL code first)

---

### M2: WASM JavaScript API (Day 2, ~100 LOC)

**Goal**: Expose `ailangLoadModule` to JavaScript via syscall/js.

**Files**:
| File | LOC | Description |
|------|-----|-------------|
| `cmd/ailang/wasm.go` | ~80 | JS bridge functions |
| Modify existing | ~20 | Add registry to global REPL |

**Tasks**:
- [ ] Add `registry *ModuleRegistry` field to REPL struct
- [ ] Initialize registry in `New()` constructor
- [ ] Implement `ailangLoadModule(this js.Value, args []js.Value)` with panic recovery
- [ ] Return structured `{success, exports, error}` object
- [ ] Register function with `js.Global().Set("ailangLoadModule", ...)`
- [ ] Add `ailangListModules()` helper for debugging

**Acceptance Criteria**:
- [ ] `window.ailangLoadModule('test', code)` returns `{success: true, exports: ['func1']}`
- [ ] Parse errors return `{success: false, error: "parse error: ..."}`
- [ ] Type errors return `{success: false, error: "type error: ..."}`
- [ ] Go panics are caught and return error (not crash WASM runtime)

**Risks**:
- `syscall/js` API quirks (mitigate: check existing WASM code patterns)

---

### M3: Import Resolution (Day 3, ~80 LOC)

**Goal**: Make REPL resolve imports from module registry.

**Files**:
| File | LOC | Description |
|------|-----|-------------|
| `internal/repl/repl.go` | ~60 | Import resolution |
| `internal/repl/repl_test.go` | ~20 | Integration tests |

**Tasks**:
- [ ] Add `resolveImport(imp *ast.Import) error` method
- [ ] Modify `ProcessExpression` to check for import statements
- [ ] Look up module in registry, check exports exist
- [ ] Add exported functions to REPL environment (`env.Set`, `typeEnv.BindScheme`)
- [ ] Write test: load module, import function, call it

**Acceptance Criteria**:
- [ ] `import test_module (add)` makes `add` available in REPL
- [ ] Error if module not loaded: "module test not loaded (use ailangLoadModule first)"
- [ ] Error if function not exported: "symbol foo not exported by test"
- [ ] Imported functions have correct types

**Risks**:
- AST import structure may differ from expectation (mitigate: check parser output first)

---

### M4: JavaScript Wrapper & Demo (Day 3.5, ~70 LOC)

**Goal**: High-level JavaScript API and working invoice processor demo.

**Files**:
| File | LOC | Description |
|------|-----|-------------|
| `docs/static/js/ailang-wrapper.js` | ~70 | Wrapper class |

**Tasks**:
- [ ] Create `AILANGWrapper` class with init/loadModule/eval/call methods
- [ ] Add `waitForGo()` and `waitForREPL()` helpers
- [ ] Track loaded modules in `loadedModules` Set
- [ ] Create simple test HTML page
- [ ] Test with invoice processor module from design doc

**Acceptance Criteria**:
- [ ] `new AILANGWrapper().init()` initializes WASM correctly
- [ ] `wrapper.loadModule('invoice', code)` loads module
- [ ] `wrapper.call('invoice', 'processInvoice', args)` works
- [ ] Invoice processor demo produces correct output

**Risks**:
- Browser CORS/loading issues (mitigate: test locally with python -m http.server)

---

## Day-by-Day Schedule

### Day 1 (8 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 0-2h | Review existing REPL code, pipeline API | M1 |
| 2-4h | Implement ModuleRegistry core | M1 |
| 4-6h | Implement LoadModule with pipeline | M1 |
| 6-8h | Write unit tests, fix issues | M1 |

**End of Day 1**: Module registry compiles and stores modules.

### Day 2 (8 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 0-2h | Finish registry tests, GetExport | M1 |
| 2-4h | Add registry to REPL, WASM bridge setup | M2 |
| 4-6h | Implement ailangLoadModule JS function | M2 |
| 6-8h | Test WASM loading in browser console | M2 |

**End of Day 2**: `window.ailangLoadModule` works in browser.

### Day 3 (8 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 0-2h | Implement resolveImport in REPL | M3 |
| 2-4h | Test import resolution, fix issues | M3 |
| 4-6h | Create AILANGWrapper JavaScript class | M4 |
| 6-8h | Test with invoice processor module | M4 |

**End of Day 3**: Full workflow works (load → import → call).

### Day 3.5 (4 hours)

| Time | Task | Milestone |
|------|------|-----------|
| 0-2h | Polish wrapper API, error handling | M4 |
| 2-4h | Update WASM docs, final testing | M4 |

**End of Sprint**: Invoice processor demo fully functional.

## Success Criteria

- [ ] **M1**: ModuleRegistry stores compiled modules with exports
- [ ] **M2**: `window.ailangLoadModule()` API works from browser
- [ ] **M3**: `import module (func)` resolves from registry
- [ ] **M4**: Invoice processor demo works end-to-end
- [ ] All unit tests passing
- [ ] WASM binary size < 5% increase
- [ ] Documentation updated

## Dependencies

**None** - WASM build infrastructure already exists.

**External**: Invoice processor AILANG code (from design doc examples).

## Open Questions

1. **Pipeline API**: Does `pipeline.Parse/Elaborate/TypeCheck` match design doc assumptions?
   - **Action**: Check existing REPL code patterns first

2. **Module name matching**: Should we enforce module name matches `ailangLoadModule` name?
   - **Recommendation**: Yes, with clear error message

3. **Browser testing**: How to test WASM in CI?
   - **Recommendation**: Manual testing for v0.7.2, add Playwright tests in v0.8.0

## Rollback Plan

If implementation hits blockers:
1. **Phase 1 fails**: Fall back to inline expressions only (no change from current)
2. **Phase 2 fails**: Can still test registry from Go tests
3. **Phase 3 fails**: Document JavaScript workarounds

---

**Created**: January 28, 2026
**Approved**: Pending user approval
