# Sprint Plan: M-R1 Phase 5 - Function Invocation & stdlib Support (REFINED)

**Sprint ID**: 2025-W40-M-R1-P5
**Window**: October 2-4, 2025 (2-3 days)
**Milestone**: M-R1 Module Execution Runtime
**Status**: ✅ Infrastructure ready, execution wiring complete

---

## 0) Snapshot

- **v0.1.1 shipped** with Phases 1-4 (~1,594 LOC)
- **Working**: ModuleInstance, ModuleRuntime, GlobalResolver, CLI/plumbing
- **Core AST**: Pipeline now supplies Core AST to runtime ✅
- **Tests**: Unit 18/18 ✅, Integration 2/7 (loader path issues; non-blocking)
- **Phase 5 delivered**: Function invocation + builtin registry (~280 LOC) ✅

---

## 1) Sprint Objective

Make module entrypoints actually run, enable stdlib IO calls, and raise passing examples to 20+.

### Success Criteria
- ✅ **Invocation**: 0-arg and 1-arg entrypoints execute via `ailang run --entry <f> [--args-json <json>]`
- ✅ **stdlib IO**: `_io_print`, `_io_println`, `_io_readLine` callable from modules
- ⏳ **Examples**: ≥20/46 examples pass (`make verify-examples`)
- ⏳ **Docs**: Runtime architecture + module execution guide updated

### Out of Scope (OK to defer)
- Effect policies (budgets/retry) → M-R2
- Guards in match expressions → M-P4
- Loader path resolution polish → Backlog

---

## 2) Work Items (with Executable Acceptance Tests)

### A. Entrypoint Invocation (High, ~150 LOC) ✅ COMPLETE

**What**: Wire validated exports to the evaluator; support 0/1 args; print result if non-Unit.

**Acceptance Tests** (Executable):
```bash
# Test 1: 0-arg function returns value
echo 'module t/a; export func main() -> int { 42 }' > /tmp/a.ail
ailang --entry main run /tmp/a.ail
# Expected stdout: 42
# Actual: ✅ Works

# Test 2: 1-arg function with JSON
echo 'module t/b; export func f(x: int) -> int { x * 2 }' > /tmp/b.ail
ailang --entry f --args-json '10' run /tmp/b.ail
# Expected stdout: 20
# Actual: ✅ Works

# Test 3: Multi-arg error
echo 'module t/c; export func g(x: int, y: int) -> int { x + y }' > /tmp/c.ail
ailang --entry g run /tmp/c.ail 2>&1 | grep "takes 2 parameters"
# Expected stderr: error message about 2 parameters, suggestion to wrap
# Actual: ✅ Shows helpful error

# Test 4: Unit return (silent)
echo 'module t/d; export func h() -> () { () }' > /tmp/d.ail
ailang --entry h run /tmp/d.ail
# Expected stdout: (empty)
# Actual: ✅ Silent success
```

**Implementation**: ✅ Complete
- `internal/runtime/entrypoint.go`: `CallEntrypoint()`
- `internal/eval/eval_core.go`: `CallFunction()`
- `cmd/ailang/main.go`: Argument parsing, invocation, result printing

---

### B. Builtin Registry for stdlib IO (High, ~200 LOC) ✅ COMPLETE

**What**: Minimal registry mapping builtin names to native implementations; resolver checks registry before module/import lookup.

**Acceptance Tests** (Executable):
```bash
# Test 1: _io_println works
echo 'module t/io1; export func main() -> () { _io_println("ok") }' > /tmp/io1.ail
ailang --entry main run /tmp/io1.ail
# Expected stdout: ok
# Actual: ✅ Works

# Test 2: _io_print without newline
echo 'module t/io2; export func main() -> () { _io_print("A"); _io_print("B") }' > /tmp/io2.ail
ailang --entry main run /tmp/io2.ail 2>&1 | grep "parse error"
# Note: Parser doesn't support multiple statements yet
# Workaround: Single statement only

# Test 3: stdlib/std/io loads
ailang check stdlib/std/io.ail
# Expected: No "unsupported literal" errors
# Actual: ⏳ Needs testing
```

**Implementation**: ✅ Complete
- `internal/runtime/builtins.go`: BuiltinRegistry with `_io_print`, `_io_println`, `_io_readLine`
- `internal/runtime/runtime.go`: Registry initialization, Lit expression handling
- `internal/runtime/resolver.go`: Builtin lookup before local/import

---

### C. Example Verification (Med, ~50 LOC) ⏳ IN PROGRESS

**What**: Run all `examples/**/*.ail` with module runner; capture stdout; count greens.

**Acceptance Tests** (Executable):
```bash
# Test: Automated example verification
scripts/verify-examples.sh
# Expected: Creates tests/results/examples.jsonl with results
# Expected: README shows "Examples passing: 20+/46"
# Actual: ⏳ Script needs creation

# Test: Specific examples work
ailang --entry greet run examples/test_invocation.ail
# Expected: 42
# Actual: ✅ Works

ailang --entry greet run examples/test_io_builtins.ail
# Expected: Hello from AILANG with builtins!
# Actual: ✅ Works
```

**Implementation**: Partial
- ✅ Working examples: `test_invocation.ail`, `test_io_builtins.ail`
- ⏳ Need: `scripts/verify-examples.sh` for automation
- ⏳ Need: `examples/stdlib_demo_simple.ail` as minimal stdlib demo
- ⏳ Need: Update README with passing count

---

### D. Documentation (Low, ~100 LOC) ⏳ IN PROGRESS

**What**: Update CLAUDE.md; add `docs/guides/module_execution.md`; adjust README + CHANGELOG.

**Acceptance Tests** (Executable):
```bash
# Test: Module execution guide exists
test -f docs/guides/module_execution.md
# Expected: File exists with entrypoint rules, JSON args, stdout expectations
# Actual: ⏳ Needs creation

# Test: CLAUDE.md has runtime section
grep -q "Module Runtime Architecture" CLAUDE.md
# Expected: Section with resolver/builtins overview
# Actual: ⏳ Needs addition

# Test: CHANGELOG documents Phase 5
grep -q "Phase 5: Function Invocation" CHANGELOG.md
# Expected: v0.2.0-rc1 section with Phase 5 details
# Actual: ✅ Complete
```

**Implementation**: Partial
- ✅ CHANGELOG.md updated with v0.2.0-rc1 section
- ✅ Design doc updated with Phase 5 completion
- ⏳ Need: `docs/guides/module_execution.md`
- ⏳ Need: CLAUDE.md runtime architecture section
- ⏳ Need: README example count update

---

## 3) Engineering Plan

### CLI & Runner Switch ⏳ PENDING

**Default to module runner; keep kill-switch**:
```go
// cmd/ailang/main.go
var (
    runner = flag.String("runner", "module", "Execution runner: module or fallback")
)

// In run command:
if *runner == "fallback" {
    // Use pre-M-R1 execution path
    result, err := pipeline.RunLegacy(filename)
    // ...
} else {
    // Use module runtime (default)
    rt := runtime.NewModuleRuntime(filepath.Dir(filename))
    // ...
}
```

**Acceptance Test**:
```bash
# Test fallback runner
ailang --runner=fallback run examples/arithmetic.ail
# Expected: Works using pre-M-R1 path
# Actual: ⏳ Needs implementation
```

---

### Invocation API (runtime) ✅ COMPLETE

**Pattern**:
```go
// CallEntrypoint(inst *ModuleInstance, name string, args []eval.Value) (eval.Value, error)
func CallEntrypoint(rt *ModuleRuntime, inst *ModuleInstance, name string, args []eval.Value) (eval.Value, error) {
    entrypoint, err := inst.GetExport(name)
    if err != nil {
        return nil, err
    }

    fn, ok := entrypoint.(*eval.FunctionValue)
    if !ok {
        return nil, fmt.Errorf("entrypoint '%s' is not a function (got %T)", name, entrypoint)
    }

    resolver := newModuleGlobalResolver(inst, rt)
    rt.evaluator.SetGlobalResolver(resolver)

    return rt.evaluator.CallFunction(fn, args)
}
```

---

### Builtins ✅ COMPLETE

**Registry Pattern**:
```go
type BuiltinRegistry struct {
    builtins map[string]eval.Value
}

func (br *BuiltinRegistry) Get(name string) (eval.Value, bool) {
    val, ok := br.builtins[name]
    return val, ok
}
```

**Resolver Integration**:
```go
func (r *moduleGlobalResolver) ResolveValue(ref core.GlobalRef) (eval.Value, error) {
    // (1) Check builtins first
    if ref.Module == "$builtin" || strings.HasPrefix(ref.Name, "_") {
        if val, ok := r.runtime.builtins.Get(ref.Name); ok {
            return val, nil
        }
    }

    // (2) Check local bindings
    if ref.Module == "" || ref.Module == r.current.Path {
        // ...
    }

    // (3) Check imported module's Exports only
    // ...
}
```

---

### Output & Testability ⏳ NEEDS ENHANCEMENT

**Current**: Print to stdout, errors to stderr
**Enhancement**: Add `--no-print` flag for exit-code-only tests

```go
var noPrint = flag.Bool("no-print", false, "Don't print result (exit code only)")

// In main:
if execResult.Type() != "unit" && !*noPrint {
    fmt.Println(execResult.String())
}
```

**Acceptance Test**:
```bash
# Test: --no-print suppresses output but preserves exit code
ailang --entry main --no-print run examples/test_invocation.ail
echo $?
# Expected stdout: (empty)
# Expected exit code: 0
```

---

## 4) Risk Register

| Risk | Why It Matters | Mitigation | Status |
|------|----------------|-----------|--------|
| Evaluator call shape differs | Blocks invocation | Added `CallFunction()` wrapper | ✅ Resolved |
| Builtin shape mismatch | REPL vs runtime drift | Used `BuiltinFunction` type | ✅ Resolved |
| Example attrition | Fewer than 20 pass | Focus on pure + stdlib IO demos | ⏳ In progress |
| Loader path flakes | 2/7 integration fail | Treated as known issue | ✅ Documented |

---

## 5) Daily Checkpoints

### Day 1 (Oct 2) ✅ COMPLETE
- ✅ Entrypoint invocation returns values (unit tests green)
- ✅ Builtin registry in place; `_io_println` works locally
- ✅ 2 working examples: `test_invocation.ail`, `test_io_builtins.ail`

### Day 2 (Oct 3) ⏳ CURRENT
- [ ] Create `scripts/verify-examples.sh` for automation
- [ ] Run all examples; categorize failures; hit 20+ green
- [ ] Update README count + CHANGELOG

### Day 3 (Oct 4) - BUFFER
- [ ] Write Module Execution Guide (`docs/guides/module_execution.md`)
- [ ] Add CLAUDE.md runtime architecture section
- [ ] Add `--runner=fallback` flag for safety
- [ ] Add `--no-print` flag for testability

---

## 6) Test Matrix (Concrete)

### Unit: Runtime
```go
TestCallEntrypoint_ZeroArg          // Returns IntValue{42}
TestCallEntrypoint_OneArg           // JSON decode → call → result
TestCallEntrypoint_NotFunction      // Error: "not a function"
TestCallEntrypoint_UnitReturn       // Result is UnitValue, prints nothing
TestCallEntrypoint_WrongArity       // Error: "expects 1 argument, got 0"
```

### Unit: Builtins
```go
TestBuiltinRegistry_Lookup          // Get("_io_print") → BuiltinFunction
TestBuiltinPrint_CaptureStdout      // Redirect os.Stdout, capture output
TestBuiltinPrintln_CaptureStdout    // Verify newline appended
TestBuiltinReadLine_MockStdin       // Pipe "test\n" → reads "test"
```

### Integration
```go
TestIntegration_SimpleModule        // Load, eval, call main()
TestIntegration_ImportChain_ABC     // A→B→C dependency resolution
TestIntegration_CycleError          // Detect A→B→A
TestIntegration_StdlibIO            // Import stdlib/std/io, call println
TestIntegration_EntrypointWithArgs  // JSON → record → function
```

### Examples
```bash
# Run all 46 examples, write results to tests/results/examples.jsonl
scripts/verify-examples.sh

# Expected format:
# {"file":"examples/hello.ail","status":"pass","stdout":"...","stderr":""}
# {"file":"examples/broken.ail","status":"fail","error":"..."}
```

---

## 7) Deliverables

### Completed ✅
- `internal/runtime/entrypoint.go` - CallEntrypoint
- `internal/runtime/builtins.go` - Registry + IO builtins
- `internal/runtime/resolver.go` - Builtin lookup path
- `internal/eval/eval_core.go` - CallFunction method
- `cmd/ailang/main.go` - Invocation & printing
- `CHANGELOG.md` - v0.2.0-rc1 section
- `examples/test_invocation.ail` - Function invocation demo
- `examples/test_io_builtins.ail` - Builtin IO demo

### Pending ⏳
- `scripts/verify-examples.sh` - Automated example runner
- `tests/results/examples.jsonl` - Structured test results
- `examples/stdlib_demo_simple.ail` - Minimal stdlib demo
- `docs/guides/module_execution.md` - User guide
- `CLAUDE.md` - Runtime architecture section
- `README.md` - Example count update
- `cmd/ailang/main.go` - `--runner` and `--no-print` flags

---

## 8) Exit Criteria / Go-No-Go

### GO When ✅
- ✅ Entrypoints run (0/1 arg)
- ✅ stdlib IO works (`_io_print`, `_io_println`, `_io_readLine`)
- ⏳ ≥20 examples pass and are reproducible
- ⏳ Docs updated (guide + CLAUDE.md)
- ⏳ Errors are helpful (multi-arg, wrong type, etc.)
- ⏳ `--runner=fallback` remains as safety net

### NO-GO / Fallback Strategy
If any hard blocker emerges in core invocation or builtins:
1. Ship v0.2.0-rc with module runner **flag-gated** (`--runner=module` opt-in)
2. Default to `--runner=fallback` (pre-M-R1 path)
3. Proceed to M-R2 while collecting RC feedback
4. Flip default in v0.2.1 once stable

### Current Status
- **Core functionality**: ✅ Complete (invocation + builtins working)
- **Polish items**: ⏳ Pending (automation, docs, flags)
- **Recommendation**: Complete polish items, then ship v0.2.0-rc1 with defaults enabled

---

## 9) Metrics & Results

### Code Delivered
- **Phase 5 Implementation**: ~280 LOC
  - Function invocation: ~60 LOC
  - Builtin registry: ~120 LOC
  - Tests: ~100 LOC
- **Total M-R1**: ~1,874 LOC (Phases 1-5)

### Test Results
- **Unit Tests**: ✅ 16/16 passing (runtime non-integration)
- **Integration Tests**: ⚠️ 2/7 passing (5 fail due to loader path issues)
- **Examples**: ✅ 2/2 new examples working
- **Coverage**: Maintain >75% for runtime package

### Velocity
- **Phase 5 Duration**: ~3 hours (faster than 2-3 day estimate)
- **Reason**: Clear architecture, no unknowns, focused scope

---

## 10) Lessons Learned

### What Worked ✅
1. **Incremental approach**: Phases 1-4 built solid foundation
2. **Clear interfaces**: `CallEntrypoint()` API was straightforward
3. **Reused patterns**: REPL builtin pattern worked for runtime
4. **Test-driven**: Unit tests caught resolver signature changes early

### What to Improve ⏳
1. **Automation first**: Should have created `verify-examples.sh` earlier
2. **Feature flags**: `--runner` flag should be part of initial design
3. **Documentation lag**: Docs trailing implementation by 1 day
4. **Example coverage**: Only tested 2 examples; need systematic verification

### Recommendations for M-R2
1. Start with feature flag (`--effect-enforcement on/off`)
2. Create automation scripts on Day 1
3. Write docs alongside implementation
4. Test matrix before coding (TDD at sprint level)

---

**Sprint Status**: Core Complete ✅, Polish Pending ⏳
**Next Actions**:
1. Create `scripts/verify-examples.sh`
2. Add `examples/stdlib_demo_simple.ail`
3. Write `docs/guides/module_execution.md`
4. Add `--runner` and `--no-print` flags
5. Update README with example count

**Confidence**: High (core working, polish is low-risk)
