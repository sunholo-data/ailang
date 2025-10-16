# M-DX1 Future Polish (v0.3.11+)

**Status**: Planned
**Prerequisites**: M-DX1.5 complete (v0.3.10) ✅
**Estimated Total Effort**: ~10-12 hours

## Context

M-DX1.5 (v0.3.10) completed the core migration goal:
- ✅ All 49 builtins migrated to new spec-based registry
- ✅ Feature flag removed - new registry is default
- ✅ Development time reduced from 7.5h → 2.5h (-67%)
- ✅ Files to edit reduced from 4 → 1 (-75%)

**What remains**: Developer-facing polish features that improve the daily workflow but weren't blocking the migration.

---

## M-DX1.6: REPL `:type` Command (~3 hours)

### Goal
Add `:type <expr>` command to REPL for quick type inspection during development.

### Motivation
**Current pain point**: When developing builtins or debugging type errors, developers must:
1. Write a test file
2. Run the compiler
3. Read the error message
4. Repeat

**With `:type`**: Instant feedback in REPL.

### Tasks

**Hour 1-2: Implementation**
- Parse `:type <expr>` in REPL command loop
- Run type inference on expression
- Pretty-print type signature
- Handle polymorphic types (display type variables)
- Format multi-line types for readability

**Hour 3: Testing + Polish**
- Add REPL test cases for `:type` command
- Test with builtins, polymorphic functions, effect types
- Update REPL help text

### Example Usage

```ailang
λ> :type _str_len
string -> int

λ> :type map
(a -> b) -> [a] -> [b]

λ> :type _net_httpRequest
string -> string -> [{name: string, value: string}] -> string
  -> <Net> Result[HttpResponse, NetError]

λ> :type let f = \x -> x + 1 in f
int -> int

λ> :type 42
int

λ> :type "hello"
string
```

### Success Criteria
- [ ] `:type` works for builtins
- [ ] `:type` works for user-defined functions
- [ ] `:type` displays polymorphic types correctly
- [ ] Effect rows are formatted clearly
- [ ] 5+ test cases covering common scenarios

### Files to Modify
- `internal/repl/repl.go` - Add command parsing (~150 LOC)
- `internal/types/pretty.go` - Type formatting (may need new file, ~100 LOC)
- `internal/repl/repl_test.go` - Test cases (~50 LOC)

---

## M-DX1.7: Enhanced Error Diagnostics (~2 hours)

### Goal
Pattern-match common builtin errors and provide actionable hints.

### Motivation
**Current errors are generic:**
```
Error: undefined global variable: _my_new_op from $builtin
```

**With enhanced diagnostics:**
```
Error: Builtin '_my_new_op' not found in $builtin module

Hint: Did you forget to register it? Add to internal/builtins/register.go:

    RegisterEffectBuiltin(BuiltinSpec{
        Module: "std/...",
        Name: "_my_new_op",
        NumArgs: 2,
        ...
    })

Registered builtins in std/...: _other_op, _another_op

To see all builtins: ailang builtins list
```

### Tasks

**Hour 1: Pattern Detection**
- Detect "undefined global variable from $builtin" errors
- Extract builtin name from error
- Check if similar builtins exist (typo detection)
- Suggest module based on naming pattern

**Hour 2: Validation Enhancements**
- Add `ailang doctor builtins --verbose` mode
- Check registry consistency
- Show which builtins are missing from which subsystem
- Validate all specs at startup (optional flag)

### Error Patterns to Improve

1. **Missing Builtin Registration**
   ```
   Error: Cannot use effect 'IO' in pure function

   Suggestion:
   - Add effect annotation to function signature
   - Example: func myFunc(x: int): <IO> string
   ```

2. **Arity Mismatch**
   ```
   Error: Function expects 2 arguments, got 3

   Suggestion:
   - Check builtin registration: NumArgs field
   - Current: NumArgs: 2
   - Your call: _my_op(a, b, c)
   ```

3. **Missing Module Import**
   ```
   Error: Unbound variable '_str_len'

   Suggestion:
   - Builtin found in registry but not imported
   - Add: import "std/string"
   - Or check if module is in search path
   ```

### Success Criteria
- [ ] Better error message for missing builtins
- [ ] `ailang doctor builtins --verbose` shows detailed validation
- [ ] Hints include file paths and example code
- [ ] 3+ error patterns detected and improved

### Files to Modify
- `internal/errors/hints.go` - Pattern matching + suggestions (~150 LOC, new file)
- `internal/builtins/validator.go` - Enhanced validation (~50 LOC)
- `cmd/ailang/main.go` - Add `--verbose` flag (~30 LOC)

---

## M-DX1.8: Comprehensive Documentation (~2 hours)

### Goal
Create `docs/ADDING_BUILTINS.md` - the definitive guide for contributors.

### Contents

**Section 1: Quick Start** (10 minutes to add a builtin)
```markdown
# Adding Builtin Functions to AILANG

## Quick Start (10 minutes)

1. Register in `internal/builtins/register.go`:
   ```go
   RegisterEffectBuiltin(BuiltinSpec{
       Module:  "std/string",
       Name:    "_str_reverse",
       NumArgs: 1,
       IsPure:  true,
       Type:    makeStrReverseType,
       Impl:    strReverseImpl,
   })
   ```

2. Implement type and logic:
   ```go
   func makeStrReverseType() types.Type {
       T := types.NewBuilder()
       return T.Func(T.String()).Returns(T.String()).Build()
   }

   func strReverseImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
       s := args[0].(*eval.StringValue)
       runes := []rune(s.Value)
       for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
           runes[i], runes[j] = runes[j], runes[i]
       }
       return &eval.StringValue{Value: string(runes)}, nil
   }
   ```

3. Validate and test:
   ```bash
   ailang doctor builtins  # Validate registration
   make test               # Run tests
   ```

That's it! 🎉
```

**Section 2: Tutorial** - Complete walkthrough with `_str_reverse` example

**Section 3: Common Patterns**
- Pure functions (math, string ops)
- Effect functions (IO, Net, FS)
- Complex types (records, ADTs, Result types)
- Polymorphic functions (when needed)

**Section 4: Testing Guide**
- Using `MockEffContext`
- Hermetic HTTP/FS tests
- Value constructors/extractors
- Property-based testing patterns

**Section 5: Troubleshooting**
- Type errors → Check Type Builder syntax
- Arity mismatches → Verify NumArgs matches impl
- Effect violations → Set Effect field correctly
- Registration failures → Run `ailang doctor builtins`

**Section 6: Architecture Overview**
Include the Mermaid diagram showing how all M-DX1 components connect.

### Success Criteria
- [ ] `docs/ADDING_BUILTINS.md` exists and is comprehensive
- [ ] Includes working examples (copy-pasteable)
- [ ] Covers all patterns from m-dx1-day3-polish.md
- [ ] Has troubleshooting section with common errors
- [ ] Includes Mermaid architecture diagram

### Files to Create
- `docs/ADDING_BUILTINS.md` (~500 LOC)

### Files to Update
- `CHANGELOG.md` - Reference the new guide
- `README.md` - Link to the guide
- `CLAUDE.md` - Update M-DX1 section with completion status

---

## M-DX1.9: Cleanup & Optimization (~2 hours)

### Goal
Remove legacy code and optimize the registry.

### Tasks

**Hour 1: Delete Legacy Code**
- Remove `registerArithmeticBuiltins()` from `internal/eval/builtins.go`
- Remove `registerComparisonBuiltins()` from `internal/eval/builtins.go`
- Remove legacy registration functions from `internal/runtime/builtins.go`
- Keep only the spec-based paths

**Hour 2: Optimize Registry**
- Benchmark registry lookup performance
- Add caching if needed (unlikely to be necessary)
- Consider lazy initialization for large registries
- Profile memory usage

### Success Criteria
- [ ] All legacy registration code deleted
- [ ] No references to old builtin registration patterns
- [ ] Tests still passing
- [ ] Registry performance measured (baseline for future)

### Files to Modify
- `internal/eval/builtins.go` - Delete legacy functions (~300 LOC removed)
- `internal/runtime/builtins.go` - Delete legacy paths (~100 LOC removed)

---

## M-DX1.10: Migrate `_json_encode` (~3 hours)

### Goal
Migrate the last remaining legacy builtin with complex ADT handling.

### Challenge
`_json_encode` has special logic for handling the `Json` ADT:
- Pattern matches on ADT constructors (JNull, JBool, JNumber, JString, JArray, JObject)
- Recursively encodes nested structures
- Requires special handling in the new registry

### Tasks

**Hour 1: Understand Current Implementation**
- Read `internal/eval/builtins.go` implementation
- Understand how ADT pattern matching works
- Identify dependencies on eval package internals

**Hour 2: Implement in New Registry**
- Create `_json_encode` registration in `internal/builtins/register.go`
- Handle ADT value inspection
- Preserve all existing behavior

**Hour 3: Test & Verify**
- Run existing JSON tests
- Add new tests for edge cases
- Verify no regressions

### Success Criteria
- [ ] `_json_encode` registered in new system
- [ ] All JSON tests passing
- [ ] No behavioral changes
- [ ] 100% test coverage maintained

### Files to Modify
- `internal/builtins/register.go` - Add `_json_encode` registration
- `internal/builtins/json.go` - May need new file for JSON logic (~200 LOC)

---

## Timeline

**Week 1** (4-5 hours):
- M-DX1.6: REPL `:type` command (3h)
- M-DX1.8: Documentation (2h)

**Week 2** (4-5 hours):
- M-DX1.7: Enhanced diagnostics (2h)
- M-DX1.9: Cleanup & optimization (2h)

**Week 3** (3 hours):
- M-DX1.10: Migrate `_json_encode` (3h)

**Total: ~10-12 hours across 3 weeks**

---

## Success Metrics (Post M-DX1 Complete)

**Development workflow:**
- Time to add builtin: 2.5h ✅ (already achieved)
- Files to edit: 1 ✅ (already achieved)
- Type construction: 10 LOC ✅ (already achieved)
- CLI validation available: Yes ✅ (already achieved)
- REPL type checking: Yes (M-DX1.6)
- Comprehensive docs: Yes (M-DX1.8)

**Code quality:**
- All builtins in new registry: 49/49 ✅ (already achieved)
- Legacy code deleted: Yes (M-DX1.9)
- 100% test coverage: Yes ✅ (already achieved)

**Developer experience:**
- Onboarding time: <30 minutes (M-DX1.8 docs)
- Error debugging: <10 minutes (M-DX1.7 diagnostics)
- Type checking: Instant (M-DX1.6 REPL)

---

## Non-Goals (Future Iterations)

**Not in M-DX1:**
- Automated migration tool (convert old → new syntax)
- CI check to prevent legacy builtin registration
- Performance benchmarks (new vs old registry)
- Hot-reload builtins for development
- Auto-generated API documentation from specs
- Builtin function discovery/search UI

These are valuable but not critical for the core DX improvement goal.

---

## References

- **Motivation**: `design_docs/planned/easier-ailang-dev.md`
- **Infrastructure**: M-DX1.1-1.4 (v0.3.9-alpha3)
- **Migration**: M-DX1.5 (v0.3.10)
- **Original plan**: `design_docs/planned/m-dx1-day3-polish.md`
