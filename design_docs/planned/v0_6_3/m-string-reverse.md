# String Reverse Builtin (`_string_reverse`)

**Status**: Planned
**Target Version**: v0.6.2
**Priority**: P2 (Low)
**Estimated Effort**: 4-6 hours
**Dependencies**: None
**Created**: 2026-01-01

---

## Overview

Add a new `_string_reverse` builtin function to the standard library that reverses a string while correctly handling Unicode characters. This enhances the string manipulation capabilities of AILANG and provides a common utility function for string operations.

---

## Problem Statement

Currently, AILANG lacks a built-in function for reversing strings. While developers can implement string reversal using other primitives (slicing, concatenation, recursion), having a native builtin provides:

1. **Performance**: Native implementation is faster than pure AILANG code
2. **Clarity**: Explicit function name is clearer than ad-hoc algorithms
3. **Completeness**: Standard library should include common string operations
4. **UTF-8 Correctness**: Ensures proper handling of multi-byte characters and grapheme clusters

### Current State

- No existing `_string_reverse` builtin
- Developers must implement string reversal manually
- Other string builtins like `_str_len`, `_str_upper`, `_str_lower`, `_str_slice` already exist

---

## Goals

**Primary Goal**: Implement a UTF-8 aware `_string_reverse` builtin that correctly handles Unicode strings.

**Success Metrics**:
1. Function correctly reverses ASCII and Unicode strings (including emoji)
2. Empty strings return empty strings
3. All edge cases covered with comprehensive tests
4. Performance comparable to standard library string operations
5. Documentation and examples complete

---

## Solution Design

### Function Signature

```ailang
_string_reverse : string -> string
```

### Behavior

- **Input**: A UTF-8 encoded string
- **Output**: The string with characters in reverse order
- **Unicode Handling**: Reverses at the character (rune) level, not byte level, to correctly handle multi-byte UTF-8 sequences

### Examples

```ailang
_string_reverse("hello")        -- Returns "olleh"
_string_reverse("")             -- Returns ""
_string_reverse("a")            -- Returns "a"
_string_reverse("🎉🎊")        -- Returns "🎊🎉" (emoji handled correctly)
_string_reverse("café")         -- Returns "éfac" (accented characters handled)
```

### Implementation Architecture

#### 1. Registration (in `internal/builtins/string.go`)

Add function call to `init()`:
```go
func init() {
    // ... existing registrations ...
    registerStringReverse()
}
```

#### 2. Type Signature Builder

```go
// makeStringReverseType builds the type signature
// Type: (String) -> String
func makeStringReverseType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String()).Returns(T.String()).Build()
}
```

#### 3. Implementation Function

```go
// stringReverseImpl implements _string_reverse
// Reverses a UTF-8 string at the rune (character) level
func stringReverseImpl(ctx *effects.EffContext, args []eval.Value) (eval.Value, error) {
    // Extract string argument
    strVal, ok := args[0].(*eval.StringValue)
    if !ok {
        return nil, fmt.Errorf("_string_reverse: expected String, got %T", args[0])
    }

    // Convert string to runes for proper Unicode handling
    runes := []rune(strVal.Value)

    // Reverse the rune slice
    for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
        runes[i], runes[j] = runes[j], runes[i]
    }

    // Convert back to string
    reversed := string(runes)

    return &eval.StringValue{Value: reversed}, nil
}
```

#### 4. Registration Call

```go
func registerStringReverse() {
    err := RegisterEffectBuiltin(BuiltinSpec{
        Module:  "std/string",
        Name:    "_string_reverse",
        NumArgs: 1,
        IsPure:  true,
        Effect:  "",
        Type:    makeStringReverseType,
        Impl:    stringReverseImpl,

        Metadata: &BuiltinMetadata{
            Description: "Reverse a string",
            LongDesc:    "Returns a new string with Unicode characters in reverse order. Correctly handles multi-byte UTF-8 characters like emoji.",
            Params: []ParamDoc{
                {Name: "s", Description: "The string to reverse"},
            },
            Returns: "A new string with characters reversed",
            Examples: []Example{
                {Code: `_string_reverse("hello")`, Description: "Returns \"olleh\""},
                {Code: `_string_reverse("")`, Description: "Returns \"\""},
                {Code: `_string_reverse("🎉")`, Description: "Returns \"🎉\" (single character unchanged)"},
            },
            Since:     "v0.6.2",
            Stability: StabilityStable,
            Tags:      []string{"string", "reverse", "unicode", "utf8"},
            Category:  "string",
        },
    })
    if err != nil {
        panic(fmt.Sprintf("failed to register _string_reverse: %v", err))
    }
}
```

### Files to Modify

| File | Changes | LOC Impact |
|------|---------|-----------|
| `internal/builtins/string.go` | Add registration function and implementation | +40-50 |
| `internal/builtins/string_test.go` | Add comprehensive unit tests | +60-80 |

### Implementation Plan

#### Phase 1: Core Implementation (2-3 hours)
- [ ] Add `registerStringReverse()` function to `internal/builtins/string.go`
- [ ] Implement type signature builder `makeStringReverseType()`
- [ ] Implement `stringReverseImpl()` function
- [ ] Add call to `registerStringReverse()` in `init()`
- [ ] Run type validation: `ailang doctor builtins`

#### Phase 2: Unit Tests (1-2 hours)
- [ ] Add test function `TestStringReverse` to `internal/builtins/string_test.go`
- [ ] Test empty string
- [ ] Test single character
- [ ] Test ASCII strings
- [ ] Test Unicode characters (emoji, accented)
- [ ] Test mixed content
- [ ] Verify performance is acceptable
- [ ] Run `make test` and verify all tests pass

#### Phase 3: Integration & Documentation (1 hour)
- [ ] Update CHANGELOG.md with new builtin
- [ ] Add example to `examples/` directory (if creating example programs)
- [ ] Verify `ailang builtins list` shows the new function
- [ ] Run `make verify-examples` to ensure no breakage
- [ ] Commit changes with appropriate message

---

## Edge Cases & Considerations

### Unicode Handling

The implementation uses Go's `[]rune` conversion, which correctly handles:
- ✅ ASCII characters
- ✅ Multi-byte UTF-8 sequences (emoji, accented characters)
- ✅ Right-to-left text (Hebrew, Arabic) - reversed at character level only

### Known Limitations

- **Grapheme Clusters**: May not handle complex grapheme clusters correctly (e.g., emoji with skin tone modifiers). This is acceptable for v0.6.2; future versions could use `unicode/norm` or `golang.org/x/text` for more sophisticated handling.
- **Performance**: Reversal requires O(n) time and O(n) space (creates intermediate rune slice)

### Acceptance Criteria

- [ ] Builtin registered in `internal/builtins/string.go`
- [ ] Type signature correctly defined
- [ ] Implementation handles empty strings
- [ ] Implementation handles Unicode correctly
- [ ] Unit tests pass (TestStringReverse)
- [ ] `make test` passes (all tests)
- [ ] `make lint` passes
- [ ] CHANGELOG.md updated
- [ ] Function appears in `ailang builtins list`
- [ ] No performance regression in benchmark tests

---

## Testing Strategy

### Unit Test Coverage

```go
func TestStringReverse(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty string", "", ""},
        {"single character", "a", "a"},
        {"two characters", "ab", "ba"},
        {"hello", "hello", "olleh"},
        {"emoji", "🎉", "🎉"},
        {"multiple emoji", "🎉🎊", "🎊🎉"},
        {"mixed", "a🎉b", "b🎉a"},
        {"accented", "café", "éfac"},
        {"numbers", "12345", "54321"},
        {"special chars", "!@#$%", "%$#@!"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Verify result matches expected
        })
    }
}
```

### Integration Testing

- Verify in REPL: `:load std/string` and call `_string_reverse`
- Run against example files that use string operations
- Verify no regression in existing string builtin tests

---

## Related Documents

This feature complements existing string builtins:
- `_str_len` - String length (v0.1.0)
- `_str_upper` - Convert to uppercase
- `_str_lower` - Convert to lowercase
- `_str_slice` - Extract substring
- `_str_concat` - Concatenate strings

---

## Timeline

| Phase | Duration | Dates |
|-------|----------|-------|
| **Phase 1: Implementation** | 2-3 days | Week 1 |
| **Phase 2: Testing** | 1-2 days | Week 1-2 |
| **Phase 3: Documentation** | 1 day | Week 2 |
| **Buffer/Review** | 2-3 days | Week 2 |
| **Total** | 6-9 days | ~1.5 weeks |

---

## Rollout Strategy

### v0.6.2 Release
1. Merge PR with `_string_reverse` implementation
2. Add to CHANGELOG.md under "Added" section
3. Tag as v0.6.2
4. Update website documentation

### Future Work (v0.7.0+)
- Consider advanced Unicode support (grapheme clusters)
- Consider performance optimizations (streaming reversal for large strings)
- Consider additional string functions based on user feedback

---

## Success Metrics

| Metric | Target | Method |
|--------|--------|--------|
| Tests passing | 100% | `make test` |
| Code coverage | ≥90% | Coverage report |
| Linting | No errors | `make lint` |
| Performance | <1ms for 10KB string | Benchmark test |
| Documentation | Complete | Example code + metadata |

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|-----------|
| UTF-8 handling issues | High | Comprehensive Unicode tests before merge |
| Performance regression | Medium | Benchmark comparison with existing builtins |
| Incomplete testing | High | Pair tests with implementation (TDD approach) |

---

## Notes

- This is a straightforward, low-risk feature with well-defined behavior
- Implementation follows established patterns from existing string builtins
- No API changes or breaking changes required
- Can be shipped as part of regular v0.6.2 release

---

**Owner**: Claude Code
**Last Updated**: 2026-01-01
**Next Review**: Upon implementation completion
