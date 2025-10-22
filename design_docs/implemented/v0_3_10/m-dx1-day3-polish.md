# M-DX1 Day 3: Polish & Migration (v0.3.10+)

**Status**: Planned
**Target Release**: v0.3.10 (Q1 2025)
**Estimated Effort**: ~12-16 hours

## Context

M-DX1 Days 1-2 (v0.3.9-alpha3) delivered the core infrastructure for modern builtin development:
- ✅ Central Registry + Type Builder DSL
- ✅ Validation + CLI commands
- ✅ Test Harness with mocking
- ✅ 2 proof-of-concept migrations
- ✅ 67% time reduction (7.5h → 2.5h)

**What remains:**
1. Migrate all 50+ legacy builtins to new registry
2. Remove feature flag (make new registry default)
3. REPL developer tools
4. Enhanced diagnostics
5. Comprehensive documentation

## Goals

1. **Complete Migration**: All builtins use new registry
2. **Remove Feature Flag**: `AILANG_BUILTINS_REGISTRY=1` becomes default
3. **Developer Polish**: REPL tools + better diagnostics
4. **Documentation**: Complete guide for contributors

## Planned Work

### M-DX1.5: Complete Builtin Migration (~4-6h)

**Goal:** Migrate all 52 legacy builtins to new registry, remove feature flag

**Current state:**
- 2 builtins migrated (_str_len, _net_httpRequest)
- 50 builtins still in legacy system
- Feature flag required: `AILANG_BUILTINS_REGISTRY=1`

**Tasks:**

1. **Batch 1: Pure String/Math Builtins** (~2h)
   ```bash
   # Migrate these first (simple, well-tested):
   - _str_compare, _str_find, _str_slice, _str_trim
   - _str_lower, _str_upper
   - add_Int, add_Float, sub_Int, sub_Float
   - mul_Int, mul_Float, div_Int, div_Float
   - mod_Int, neg_Int, neg_Float
   ```

2. **Batch 2: Comparison/Logic Builtins** (~1h)
   ```bash
   - eq_Int, eq_Float, eq_String, eq_Bool
   - lt_Int, lt_Float, le_Int, le_Float
   - gt_Int, gt_Float, ge_Int, ge_Float
   - and_Bool, or_Bool, not_Bool
   ```

3. **Batch 3: IO Effect Builtins** (~1.5h)
   ```bash
   - _io_print, _io_println, _io_readLine
   - _io_writeFile, _io_readFile
   ```

4. **Batch 4: Net Effect Builtins** (~1h)
   ```bash
   - _net_httpGet, _net_httpPost
   - Already done: _net_httpRequest
   ```

5. **Batch 5: JSON/Misc Builtins** (~1h)
   ```bash
   - _json_encode, _json_decode
   - _list_length, _list_append, _list_head, _list_tail
   ```

6. **Remove Feature Flag** (~0.5h)
   - Delete legacy registration code
   - Make new registry always-on
   - Update tests to remove AILANG_BUILTINS_REGISTRY checks
   - Update documentation

**Success Criteria:**
- ✅ All 52 builtins in new registry
- ✅ Legacy code deleted
- ✅ `AILANG_BUILTINS_REGISTRY` env var no longer needed
- ✅ All tests passing
- ✅ `ailang builtins list` shows all 52

**Files to modify:**
- `internal/builtins/register.go` - Add all registrations
- `internal/runtime/builtins.go` - Delete legacy `registerArithmeticBuiltins()`
- `internal/link/builtin_module.go` - Delete legacy `registerLegacyBuiltins()`
- `internal/eval/builtins.go` - May need cleanup

**Testing strategy:**
- Run existing test suite after each batch
- Add test coverage for newly migrated builtins
- Ensure backward compatibility (same behavior)

### M-DX1.6: REPL Developer Tools (~3h)

**Goal:** Add `:type` and `:explain` commands to REPL for better developer experience

**Tasks:**

1. **:type command** (~2h)
   - Parse `:type <expr>` command in REPL
   - Run type inference on expression
   - Pretty-print the type signature
   - Handle polymorphic types (show generics)
   - Format multi-line types for readability

   ```bash
   > :type _str_len
   string -> int

   > :type map
   (a -> b) -> [a] -> [b]

   > :type _net_httpRequest
   string ->
   string ->
   [{name: string, value: string}] ->
   string ->
   <Net> Result<{status: int, headers: [{...}], body: string}, NetError>
   ```

2. **:explain command** (~1h)
   - Show type inference steps
   - Explain why type errors occur
   - Suggest fixes for common mistakes

   ```bash
   > :explain 1 + "hello"
   Type Error: Cannot add Int and String

   Explanation:
   - Left operand: 1 has type Int
   - Right operand: "hello" has type String
   - Operator '+' expects both operands to have the same numeric type

   Suggestion:
   - If you meant string concatenation, use _str_concat
   - If you meant to convert Int to String, use show(1)
   ```

**Files to modify:**
- `internal/repl/repl.go` - Add command parsing
- `internal/types/pretty.go` - Type formatting (may need new file)
- `internal/errors/explain.go` - Error explanation logic

### M-DX1.7: Enhanced Diagnostics (~3h)

**Goal:** Pattern-match common errors and provide tailored hints

**Common error patterns to detect:**

1. **Effect in Pure Context** (~1h)
   ```
   Error: Cannot use effect 'IO' in pure function

   Suggestion:
   - Add effect annotation to function signature
   - Example: func myFunc(x: int): <IO> string
   ```

2. **Nullary Constructor Application** (~0.5h)
   ```
   Error: Constructor 'None' expects 0 arguments, got 1

   Suggestion:
   - Remove parentheses: None instead of None()
   ```

3. **Missing Module Import** (~0.5h)
   ```
   Error: Unbound variable '_str_len'

   Suggestion:
   - Import std/string module
   - Add: import "std/string"
   ```

4. **Type Mismatch in Record** (~1h)
   ```
   Error: Field 'age' has type String but expected Int

   Suggestion:
   - Convert to int: {age: parseI int(user.age)}
   - Or change type annotation to String
   ```

**Files to modify:**
- `internal/errors/hints.go` - Pattern matching + suggestions
- `internal/types/errors.go` - Enhanced error messages

### M-DX1.8: Documentation (~2h)

**Goal:** Create comprehensive guide for contributors

**Tasks:**

1. **docs/ADDING_BUILTINS.md** (~1.5h)
   - Quick Start (10 minutes)
   - Step-by-step tutorial
   - Common patterns (pure, effect, complex types)
   - Testing guide
   - Troubleshooting
   - Migration from legacy

2. **Update existing docs** (~0.5h)
   - CHANGELOG.md with v0.3.10 features
   - README.md with M-DX1 completion
   - CLAUDE.md (already done in v0.3.9)

**Structure:**
```markdown
# Adding Builtin Functions to AILANG

## Quick Start (10 minutes)
[Step-by-step minimal example]

## Tutorial: String Reverse Builtin
[Complete walkthrough with tests]

## Common Patterns
- Pure functions
- Effect functions (IO, Net, FS)
- Complex types (records, ADTs)
- Polymorphic functions

## Testing Guide
- Unit tests with MockEffContext
- Hermetic HTTP/FS tests
- Property-based testing

## Troubleshooting
- Type errors
- Arity mismatches
- Effect violations
- Registration failures

## Migration from Legacy
[For maintainers migrating old builtins]
```

## Timeline

**Week 1: Migration + Flag Removal** (~6h)
- Day 1: Batch 1-2 migrations (3h)
- Day 2: Batch 3-5 migrations (3h)
- Day 3: Remove feature flag, test (1h)

**Week 2: Polish + Docs** (~6h)
- Day 1: REPL :type command (2h)
- Day 2: REPL :explain + diagnostics (2h)
- Day 3: Documentation (2h)

**Total: ~12 hours across 2 weeks**

## Success Criteria

**For v0.3.10 release:**
- ✅ All 52 builtins migrated to new registry
- ✅ No feature flag required
- ✅ `:type` command working in REPL
- ✅ 4+ enhanced diagnostic patterns
- ✅ docs/ADDING_BUILTINS.md complete
- ✅ All tests passing
- ✅ Changelog updated

**Metrics:**
- 0 legacy builtins remaining
- 100% test coverage maintained
- 4+ diagnostic patterns
- Documentation complete

## Risks & Mitigations

**Risk 1: Migration breaks existing code**
- Mitigation: Test suite must pass after each batch
- Keep feature flag until ALL builtins migrated
- Extensive testing before removal

**Risk 2: Type Builder DSL doesn't cover all cases**
- Mitigation: Already proven with complex _net_httpRequest type
- Add methods as needed during migration
- Fallback to manual construction if necessary

**Risk 3: REPL commands interact poorly with existing features**
- Mitigation: Parse `:` prefix first, before normal expressions
- Add comprehensive REPL tests
- Document command precedence

## Future Work (v0.4.0+)

**Not in this sprint:**
- Automated migration tool (convert old → new syntax)
- CI check to prevent legacy builtin registration
- Performance benchmarks (new vs old registry)
- Hot-reload builtins for development

## References

- Original design: `design_docs/planned/easier-ailang-dev.md`
- Implementation: M-DX1 commits (v0.3.9-alpha1 through alpha3)
- Test coverage: `internal/builtins/*_test.go`, `internal/effects/testctx/*_test.go`
