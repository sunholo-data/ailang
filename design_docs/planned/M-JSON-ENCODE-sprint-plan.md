# M-JSON-ENCODE Sprint Plan

**Sprint Goal**: Implement missing `_json_encode` builtin to unblock api_call_json benchmark
**Duration**: 4-6 hours (single day sprint)
**Target Version**: v0.3.23
**Priority**: P0 - Critical blocker
**Risk Level**: Low (well-defined, follows existing patterns)

## Sprint Summary

Implement the missing `_json_encode` builtin that was never migrated to M-DX1's new builtin registry. This is blocking the api_call_json benchmark where all models fail with `IMP010: symbol 'encode' not exported by 'std/json'`.

**Key Deliverables:**
1. `internal/builtins/json_encode.go` - JSON encoding implementation (~250 LOC)
2. `internal/builtins/json_encode_test.go` - Comprehensive tests (~150 LOC)
3. Uncomment `encode()` in `std/json.ail` (4 lines)
4. Update CHANGELOG.md for v0.3.23

**Success Metrics:**
- ✅ All unit tests passing (expect 20+ tests)
- ✅ Roundtrip tests pass: `decode(encode(x)) == Ok(x)`
- ✅ api_call_json benchmark works (no IMP010 errors)
- ✅ `ailang builtins list` shows `_json_encode`
- ✅ Full test suite passes: `make test`

## Current Status Analysis

### Completed (Blocking This Work)
- ✅ M-DX1: New Builtin Registry (v0.3.10)
- ✅ `_json_decode` builtin implemented and working
- ✅ `std/json.ail` module structure exists
- ✅ Helper functions (jo, kv, js, jnum, ja, jb) exist

### Velocity Analysis (Last 14 Days)
Recent work shows:
- **Bug fixes**: ~30-60 LOC/day (parser fixes, etc.)
- **Feature work**: ~200-400 LOC/day (agent protocol, hooks)
- **Test coverage**: Typically 30-50% of implementation LOC
- **M-EVAL-HTTP-FIX sprint**: Completed Milestones 1-2 in 2 days

**Estimated velocity for this work:**
- Implementation: ~250 LOC (json_encode.go)
- Tests: ~150 LOC (json_encode_test.go)
- Total: ~400 LOC
- **Realistic timeline**: 4-6 hours (half day to full day)

### Remaining Work
- ❌ `_json_encode` builtin (main blocker)
- ❌ Uncomment encode() in std/json.ail
- ❌ Test api_call_json benchmark
- ❌ Create v0.3.23 prompt (optional - can use v0.3.22)

## Milestone Breakdown

### Milestone 1: Core Implementation (2-3 hours)

**Goal**: Implement `_json_encode` builtin following M-DX1 pattern

**Tasks:**
1. **Create json_encode.go structure** (~30 min)
   - File header, imports, init() function
   - Register with `RegisterEffectBuiltin()`
   - Implement `makeJSONEncodeType()` - Type signature: `Json -> string`
   - **LOC**: ~50

2. **Implement recursive encoder** (~1 hour)
   - `jsonEncodeImpl()` - Entry point with EffContext
   - `encodeValue()` - Recursive traversal of Json ADT
   - Handle all 6 JSON types:
     - JNull → "null"
     - JBool(b) → "true"/"false"
     - JNumber(f) → format float intelligently
     - JString(s) → escaped quoted string
     - JArray(xs) → "[...]" with recursive encoding
     - JObject(kvs) → "{...}" with recursive encoding
   - **LOC**: ~120

3. **Implement string escaping** (~30 min)
   - `escapeString()` - RFC 8259 compliance
   - Escape: `"`, `\`, `/`, `\b`, `\f`, `\n`, `\r`, `\t`
   - Control chars (U+0000 to U+001F) → `\uXXXX`
   - **LOC**: ~40

4. **Implement number formatting** (~15 min)
   - `formatNumber()` - Clean float formatting
   - `42.0` → `"42"` (no unnecessary decimals)
   - `42.5` → `"42.5"` (preserve actual decimals)
   - **LOC**: ~20

5. **Add builtin metadata** (~15 min)
   - Description, examples, params, returns
   - Since, stability, tags, category
   - **LOC**: ~20

**Acceptance Criteria:**
- [ ] File compiles without errors
- [ ] `ailang builtins list` shows `_json_encode` in std/json
- [ ] `ailang doctor builtins` passes validation
- [ ] Manual smoke test: encode JObject works

**Estimated LOC**: ~250 (implementation)

### Milestone 2: Testing (1.5-2 hours)

**Goal**: Comprehensive test coverage for all JSON types and edge cases

**Tasks:**
1. **Test individual JSON types** (~30 min)
   - TestJSONEncodeNull
   - TestJSONEncodeBool (true and false)
   - TestJSONEncodeNumber (integers, floats, edge cases)
   - TestJSONEncodeString (simple, with escapes, unicode)
   - TestJSONEncodeArray (empty, single, multiple, nested)
   - TestJSONEncodeObject (empty, single KV, multiple KVs, nested)
   - **Tests**: 12+ test functions
   - **LOC**: ~80

2. **Test string escaping** (~20 min)
   - All special chars: `"`, `\`, `/`, `\b`, `\f`, `\n`, `\r`, `\t`
   - Control characters (U+0000 to U+001F)
   - Unicode strings (passthrough UTF-8)
   - **Tests**: 5+ test cases
   - **LOC**: ~30

3. **Test edge cases** (~15 min)
   - Empty array: `[]`
   - Empty object: `{}`
   - Deeply nested structures
   - Large numbers (preserve precision)
   - **Tests**: 4+ test cases
   - **LOC**: ~20

4. **Roundtrip tests** (~25 min)
   - For each JSON type: `decode(encode(x)) == Ok(x)`
   - Verify structure equality (not string equality, key order varies)
   - Test with real-world JSON payloads
   - **Tests**: 3+ test functions
   - **LOC**: ~20

**Acceptance Criteria:**
- [ ] All tests passing: `go test ./internal/builtins/`
- [ ] Test coverage >90% on json_encode.go
- [ ] Roundtrip tests pass for all JSON types
- [ ] No race conditions: `go test -race`

**Estimated LOC**: ~150 (tests)

### Milestone 3: Integration (0.5-1 hour)

**Goal**: Uncomment encode() and verify end-to-end functionality

**Tasks:**
1. **Uncomment encode() in std/json.ail** (~5 min)
   - Remove comment markers on lines 19-22
   - Remove TODO note about migration
   - **LOC**: 4 lines changed

2. **Test std/json helpers** (~10 min)
   - Verify `jo()`, `kv()`, `js()`, `jnum()`, `ja()`, `jb()` work with encode()
   - Test in REPL: `import std/json (encode, jo, kv, js)`
   - **Manual test**, no new LOC

3. **Test api_call_json benchmark** (~15 min)
   - Run manually: `ailang eval-suite --models gpt5-mini --benchmarks api_call_json`
   - Verify no IMP010 errors
   - Check that AI can use encode() successfully
   - **Validation**, no new LOC

4. **Run full test suite** (~10 min)
   - `make test` - All tests should pass
   - `make lint` - No linting errors
   - `make test-coverage-badge` - Verify coverage didn't drop
   - **Validation**, no new LOC

**Acceptance Criteria:**
- [ ] `std/json` module exports `encode()` function
- [ ] REPL test works: `:type encode` shows `Json -> string`
- [ ] api_call_json benchmark runs without IMP010 errors
- [ ] Full test suite passes
- [ ] Linting passes

**Estimated LOC**: 4 (std/json.ail change)

### Milestone 4: Documentation (0.5 hours)

**Goal**: Document the fix and update version

**Tasks:**
1. **Update CHANGELOG.md** (~10 min)
   - Add v0.3.23 section
   - List changes: Added `_json_encode` builtin
   - Note: Unblocks api_call_json benchmark
   - Metrics: ~400 LOC (250 impl + 150 tests)
   - **LOC**: ~10 lines in CHANGELOG

2. **Update M-EVAL-HTTP-FIX-FINDINGS.md** (~10 min)
   - Add resolution section
   - Link to v0.3.23 implementation
   - Note impact on api_call_json results
   - **LOC**: ~20 lines

3. **Commit and build** (~10 min)
   - `make quick-install` - Rebuild ailang binary
   - Git commit with detailed message
   - Verify binary has new builtin

**Acceptance Criteria:**
- [ ] CHANGELOG.md updated with v0.3.23
- [ ] M-EVAL-HTTP-FIX findings updated
- [ ] Code committed with clear message
- [ ] Binary rebuilt and tested

**Estimated LOC**: ~30 (documentation)

## Day-by-Day Breakdown

### Session 1: Implementation (2-3 hours)

**Hour 1: Core Structure**
- [x] Create json_encode.go file
- [x] Set up init() and registration
- [x] Implement type signature
- [x] Implement jsonEncodeImpl() entry point
- **Checkpoint**: File compiles, builtin registers

**Hour 2: Recursive Encoding**
- [ ] Implement encodeValue() for all 6 JSON types
- [ ] Implement escapeString() with RFC 8259 compliance
- [ ] Implement formatNumber() for clean floats
- **Checkpoint**: Manual smoke test works

**Hour 3: Metadata & Validation**
- [ ] Add builtin metadata (description, examples)
- [ ] Run `ailang doctor builtins` - should pass
- [ ] Run `ailang builtins list` - verify _json_encode appears
- **Checkpoint**: Milestone 1 complete

### Session 2: Testing (1.5-2 hours)

**Hour 4: Unit Tests**
- [ ] Test all JSON types (12+ test functions)
- [ ] Test string escaping (5+ test cases)
- [ ] Test edge cases (4+ test cases)
- **Checkpoint**: `go test ./internal/builtins/` passes

**Hour 5: Integration Tests**
- [ ] Roundtrip tests (3+ test functions)
- [ ] Uncomment encode() in std/json.ail
- [ ] Test std/json helpers in REPL
- **Checkpoint**: Milestone 2 and 3 complete

### Session 3: Validation & Documentation (0.5-1 hour)

**Hour 6: Final Validation**
- [ ] Test api_call_json benchmark manually
- [ ] Run full test suite: `make test`
- [ ] Update CHANGELOG.md and findings
- [ ] Commit and rebuild binary
- **Checkpoint**: Sprint complete!

## Technical Approach

### Following json_decode.go Pattern

**Reference implementation**: `internal/builtins/json_decode.go`

**Key patterns to follow:**
1. **Registration**:
   ```go
   func init() {
       registerJSONEncode()
   }

   func registerJSONEncode() {
       err := RegisterEffectBuiltin(BuiltinSpec{
           Module:  "std/json",
           Name:    "_json_encode",
           NumArgs: 1,
           IsPure:  true,
           Type:    makeJSONEncodeType,
           Impl:    jsonEncodeImpl,
           Metadata: &BuiltinMetadata{ /* ... */ },
       })
       if err != nil {
           panic(fmt.Sprintf("failed to register _json_encode: %v", err))
       }
   }
   ```

2. **Type Signature**:
   ```go
   func makeJSONEncodeType() types.Type {
       T := types.NewBuilder()
       jsonType := T.Con("Json")
       return T.Func(jsonType).Returns(T.String()).Build()
   }
   ```

3. **Implementation Entry Point**:
   ```go
   func jsonEncodeImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
       // Extract Json ADT value
       jsonVal, ok := args[0].(*eval.TaggedValue)
       if !ok {
           return nil, fmt.Errorf("_json_encode: expected Json, got %T", args[0])
       }

       // Encode to string
       var buf strings.Builder
       if err := encodeValue(jsonVal, &buf); err != nil {
           return nil, err
       }

       // Return string value
       return &eval.StringValue{Value: buf.String()}, nil
   }
   ```

4. **Recursive Encoder**:
   ```go
   func encodeValue(val eval.Value, buf *strings.Builder) error {
       tagged, ok := val.(*eval.TaggedValue)
       if !ok {
           return fmt.Errorf("expected TaggedValue, got %T", val)
       }

       switch tagged.CtorName {
       case "JNull":
           buf.WriteString("null")
       case "JBool":
           // ... (see design doc)
       // ... (handle all 6 types)
       }
       return nil
   }
   ```

### RFC 8259 Compliance

**String escaping rules**:
- `"` → `\"`
- `\` → `\\`
- `/` → `\/` (optional but common)
- `\b` → `\b`
- `\f` → `\f`
- `\n` → `\n`
- `\r` → `\r`
- `\t` → `\t`
- Control chars (U+0000 to U+001F) → `\uXXXX`

**Implementation**:
```go
func escapeString(s string, buf *strings.Builder) {
    for _, r := range s {
        switch r {
        case '"':
            buf.WriteString(`\"`)
        case '\\':
            buf.WriteString(`\\`)
        case '\b':
            buf.WriteString(`\b`)
        case '\f':
            buf.WriteString(`\f`)
        case '\n':
            buf.WriteString(`\n`)
        case '\r':
            buf.WriteString(`\r`)
        case '\t':
            buf.WriteString(`\t`)
        default:
            if r < 0x20 {
                buf.WriteString(fmt.Sprintf(`\u%04x`, r))
            } else {
                buf.WriteRune(r)
            }
        }
    }
}
```

## Risk Factors

### Low Risk
- ✅ Well-defined specification (RFC 8259)
- ✅ Reference implementation exists (json_decode.go)
- ✅ M-DX1 registry pattern is well-established
- ✅ Comprehensive test strategy planned

### Potential Issues
- ⚠️ **Floating-point formatting**: `42.0` should be `42`, not `42.0`
  - **Mitigation**: Use `strconv.FormatFloat()` with intelligent formatting
- ⚠️ **Object key ordering**: JSON spec doesn't guarantee order
  - **Mitigation**: Don't test exact string equality, parse and compare structure
- ⚠️ **Unicode edge cases**: UTF-8 encoding, surrogate pairs
  - **Mitigation**: Go handles UTF-8 natively, just pass through

### Blocked By
- None! All dependencies are complete (M-DX1, json_decode, std/json module)

## Success Metrics

**Functional Requirements:**
- [ ] `_json_encode` builtin registered in registry
- [ ] All 6 JSON types encode correctly
- [ ] String escaping is RFC 8259 compliant
- [ ] Roundtrip tests pass: `decode(encode(x)) == Ok(x)`
- [ ] `std/json.ail` exports `encode()` function
- [ ] api_call_json benchmark works (no IMP010 errors)

**Quality Requirements:**
- [ ] 100% test coverage on new code
- [ ] Zero regressions in existing tests
- [ ] Linting passes: `make lint`
- [ ] No race conditions: `go test -race`

**Performance (non-critical):**
- [ ] Encoding 1KB JSON: <1ms (verify but not blocking)
- [ ] No memory leaks (use strings.Builder)

## Dependencies

**Completed:**
- ✅ M-DX1: New Builtin Registry (v0.3.10)
- ✅ `_json_decode` builtin (v0.2.0)
- ✅ `std/json.ail` module structure (v0.3.9)

**After This Sprint:**
- v0.3.23 prompt update (optional - can reuse v0.3.22)
- Re-run api_call_json benchmark to measure improvement
- Consider v0.3.23 baseline with full model suite

## Open Questions

None! This is a well-defined implementation task with clear requirements.

## Notes

- This sprint is intentionally small and focused - just implement the missing builtin
- No new features, no architectural changes, just completing existing work
- Follows established patterns from json_decode.go and M-DX1
- Main risk is underestimating test time, but 1.5-2 hours should be sufficient

---

**Created**: 2025-10-27
**Sprint Type**: Bug Fix / Completion
**Estimated Completion**: 2025-10-27 (same day)
