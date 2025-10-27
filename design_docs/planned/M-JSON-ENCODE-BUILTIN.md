# M-JSON-ENCODE: Implement Missing JSON Encode Builtin

**Status**: Planned
**Target**: v0.3.23
**Priority**: P0 (Critical - Blocks api_call_json benchmark)
**Estimated**: 4-6 hours

## Problem Statement

**Current State:**
- `std/json.ail` has `encode()` function **commented out** (lines 19-22)
- The underlying `_json_encode` builtin was never migrated to M-DX1's new builtin registry
- Only `_json_decode` exists in the registry
- The prompt teaches `import std/json (encode, jo, kv, js, jnum)` but these don't work

**Impact:**
- **api_call_json benchmark**: 0/6 models succeed (all hit IMP010 error)
- v0.3.22 validation: ALL 3 dev models fail with `IMP010: symbol 'encode' not exported by 'std/json'`
- HTTP/JSON examples in prompt lead to dead end
- AIs learn correct syntax but can't use it

**Evidence:**
```bash
$ ailang builtins list --by-module | grep "std/json"
# std/json (1)
  _json_decode                   [pure]
# ← Only decode! No encode!
```

**Discovery Context:**
Found during M-EVAL-HTTP-FIX sprint (Oct 27, 2025). Enhanced error messages and HTTP repositioning successfully taught AIs the correct AILANG syntax, but they immediately hit the missing encode() function.

## Solution Design

### Overview

Implement `_json_encode` builtin that converts AILANG Json ADT to JSON string:

- **Input**: Json ADT value (JNull, JBool, JNumber, JString, JArray, JObject)
- **Output**: JSON string
- **Implementation**: Recursive traversal with Go's strings.Builder for efficiency
- **Registry**: Register in M-DX1's new builtin registry pattern

### Architecture

**Components:**

1. **internal/builtins/json_encode.go** (~250 LOC)
   - `registerJSONEncode()` - Register with M-DX1 registry
   - `jsonEncodeImpl()` - Main entry point
   - `encodeValue()` - Recursive encoder
   - `escapeString()` - JSON string escaping (RFC 8259)

2. **internal/builtins/json_encode_test.go** (~150 LOC)
   - Unit tests for all JSON types
   - String escaping tests (quotes, backslashes, control chars, unicode)
   - Roundtrip tests (decode → encode → decode)
   - Edge cases (empty objects, nested structures, large numbers)

3. **std/json.ail** (1 line change)
   - Uncomment lines 19-22 to expose `encode()` function

### Implementation Details

#### Type Signature

```go
func makeJSONEncodeType() types.Type {
    T := types.NewBuilder()
    // Type signature: Json -> string
    jsonType := T.Con("Json")
    return T.Func(jsonType).Returns(T.String()).Build()
}
```

#### Core Encoder Logic

```go
func encodeValue(val eval.Value, buf *strings.Builder) error {
    switch v := val.(type) {
    case *eval.TaggedValue:
        switch v.CtorName {
        case "JNull":
            buf.WriteString("null")
        case "JBool":
            if v.Fields[0].(*eval.BoolValue).Value {
                buf.WriteString("true")
            } else {
                buf.WriteString("false")
            }
        case "JNumber":
            // Format float without unnecessary decimals
            num := v.Fields[0].(*eval.FloatValue).Value
            buf.WriteString(formatNumber(num))
        case "JString":
            str := v.Fields[0].(*eval.StringValue).Value
            buf.WriteByte('"')
            escapeString(str, buf)
            buf.WriteByte('"')
        case "JArray":
            // Recursively encode array elements
            buf.WriteByte('[')
            // ... (see implementation)
        case "JObject":
            // Recursively encode object key-value pairs
            buf.WriteByte('{')
            // ... (see implementation)
        }
    }
}
```

#### String Escaping (RFC 8259)

Must escape:
- `"` → `\"`
- `\` → `\\`
- `/` → `\/` (optional but common)
- `\b` → `\b`
- `\f` → `\f`
- `\n` → `\n`
- `\r` → `\r`
- `\t` → `\t`
- Control chars (U+0000 to U+001F) → `\uXXXX`

### Files to Modify/Create

**New files:**
- `internal/builtins/json_encode.go` (~250 LOC)
- `internal/builtins/json_encode_test.go` (~150 LOC)

**Modified files:**
- `std/json.ail` - Uncomment lines 19-22 (4 lines)

**Total new code: ~400 LOC**

## Implementation Plan

### Phase 1: Core Implementation (~3 hours)

**Step 1.1: Create json_encode.go** (~1.5 hours)
- [ ] Set up file structure and imports
- [ ] Implement `registerJSONEncode()` with M-DX1 pattern
- [ ] Implement `makeJSONEncodeType()` type signature
- [ ] Implement `jsonEncodeImpl()` entry point
- [ ] Implement `encodeValue()` recursive encoder
- [ ] Implement `escapeString()` with RFC 8259 compliance
- [ ] Implement `formatNumber()` for clean float formatting

**Step 1.2: Handle All JSON Types** (~1 hour)
- [ ] JNull → "null"
- [ ] JBool(true/false) → "true"/"false"
- [ ] JNumber(float) → numeric string (no trailing .0)
- [ ] JString(string) → escaped quoted string
- [ ] JArray(List[Json]) → "[...]" with recursive encoding
- [ ] JObject(List[{key, value}]) → "{...}" with recursive encoding

**Step 1.3: Edge Cases** (~0.5 hours)
- [ ] Empty arrays: `[]`
- [ ] Empty objects: `{}`
- [ ] Nested structures (array of objects, etc.)
- [ ] Large numbers (preserve precision)
- [ ] Unicode strings (UTF-8 passthrough)

### Phase 2: Testing (~2 hours)

**Step 2.1: Unit Tests** (~1 hour)
- [ ] Test each JSON type individually
- [ ] Test string escaping (all special chars)
- [ ] Test nested structures
- [ ] Test edge cases (empty, large, unicode)

**Step 2.2: Integration Tests** (~0.5 hours)
- [ ] Roundtrip tests: `decode(encode(x)) == Ok(x)`
- [ ] Test with std/json.ail helper functions (jo, kv, js, etc.)
- [ ] Test with real-world JSON (API payloads)

**Step 2.3: Validation** (~0.5 hours)
- [ ] Run `ailang doctor builtins` (should pass)
- [ ] Run `ailang builtins list --by-module` (should show _json_encode)
- [ ] Test in REPL: `:type _json_encode`
- [ ] Run full test suite: `make test`

### Phase 3: Integration (~1 hour)

**Step 3.1: Uncomment encode() in std/json.ail** (~0.25 hours)
- [ ] Uncomment lines 19-22
- [ ] Test that `encode()` works end-to-end
- [ ] Verify `jo()`, `kv()`, `js()`, etc. work with encode()

**Step 3.2: Documentation** (~0.5 hours)
- [ ] Add metadata to builtin spec (description, examples, etc.)
- [ ] Update CHANGELOG.md with v0.3.23 entry
- [ ] Update M-EVAL-HTTP-FIX-FINDINGS.md with resolution

**Step 3.3: Verification** (~0.25 hours)
- [ ] Run api_call_json benchmark manually
- [ ] Verify no IMP010 errors
- [ ] Check that AIs can now use encode() successfully

## Success Metrics

**Functional:**
- [ ] `_json_encode` builtin registered in M-DX1 registry
- [ ] All unit tests passing (20+ tests expected)
- [ ] Roundtrip tests passing: `decode(encode(x)) == Ok(x)`
- [ ] `std/json.ail` exports `encode()` function
- [ ] api_call_json benchmark works (no IMP010 errors)

**Quality:**
- [ ] 100% test coverage on new code
- [ ] RFC 8259 compliance for string escaping
- [ ] Zero regressions in existing tests
- [ ] Linting passes: `make lint`

**Performance:**
- [ ] Encoding 1KB JSON: <1ms (not critical, but nice to verify)
- [ ] No memory leaks (use strings.Builder for efficiency)

## Testing Strategy

### Unit Tests

```go
func TestJSONEncodeNull(t *testing.T) {
    ctx := testctx.NewMockEffContext()
    jnull := makeJNull()
    result, err := jsonEncodeImpl(ctx, []eval.Value{jnull})
    assert.NoError(t, err)
    assert.Equal(t, "null", testctx.GetString(result))
}

func TestJSONEncodeBool(t *testing.T) {
    ctx := testctx.NewMockEffContext()
    jtrue := makeJBool(true)
    result, err := jsonEncodeImpl(ctx, []eval.Value{jtrue})
    assert.NoError(t, err)
    assert.Equal(t, "true", testctx.GetString(result))
}

func TestJSONEncodeString(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"hello", `"hello"`},
        {"hello\nworld", `"hello\nworld"`},
        {`quote"test`, `"quote\"test"`},
        {"tab\there", `"tab\there"`},
    }
    // ... test each case
}

func TestJSONEncodeObject(t *testing.T) {
    ctx := testctx.NewMockEffContext()
    // Build: {name: "Alice", age: 30}
    obj := makeJObject([]{
        makeKV("name", makeJString("Alice")),
        makeKV("age", makeJNumber(30.0)),
    })
    result, err := jsonEncodeImpl(ctx, []eval.Value{obj})
    assert.NoError(t, err)
    // Parse result and verify structure
}
```

### Integration Tests

```go
func TestRoundtrip(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    // Original JSON string
    input := `{"name":"Alice","age":30,"active":true}`

    // Decode
    decoded, err := jsonDecodeImpl(ctx, []eval.Value{testctx.MakeString(input)})
    assert.NoError(t, err)
    jsonVal := testctx.GetResultOk(decoded)

    // Encode
    encoded, err := jsonEncodeImpl(ctx, []eval.Value{jsonVal})
    assert.NoError(t, err)
    output := testctx.GetString(encoded)

    // Verify structure (order may differ)
    var original, roundtripped map[string]interface{}
    json.Unmarshal([]byte(input), &original)
    json.Unmarshal([]byte(output), &roundtripped)
    assert.Equal(t, original, roundtripped)
}
```

## Example Usage

### Before (Broken)

```ailang
import std/json (encode, jo, kv, js, jnum)  -- FAILS: IMP010 error

let payload = jo([
  kv("message", js("Hello")),
  kv("count", jnum(42.0))
])

let jsonString = encode(payload)  -- Function doesn't exist!
```

### After (Working)

```ailang
import std/json (encode, jo, kv, js, jnum)  -- ✓ Works!

let payload = jo([
  kv("message", js("Hello")),
  kv("count", jnum(42.0))
])

let jsonString = encode(payload)  -- ✓ Returns: {"message":"Hello","count":42}
```

### REPL Test

```
λ[IO]> import std/json (encode, jo, kv, js, jnum)
λ[IO]> let x = jo([kv("test", js("hello"))])
λ[IO]> encode(x)
"{\"test\":\"hello\"}" : string
```

## Risks and Mitigations

**Risk 1: String escaping bugs**
- **Mitigation**: Comprehensive test coverage for all special chars
- **Reference**: RFC 8259 Section 7 (Strings)
- **Validation**: Roundtrip tests ensure correctness

**Risk 2: Floating-point formatting**
- **Example**: `42.0` should be `42`, not `42.0`
- **Mitigation**: Use Go's `strconv.FormatFloat()` with intelligent formatting
- **Test**: Verify `JNumber(42.0)` → `"42"` and `JNumber(42.5)` → `"42.5"`

**Risk 3: Performance for large JSON**
- **Mitigation**: Use `strings.Builder` (amortized O(1) appends)
- **Note**: Not critical for M-EVAL (benchmark JSONs are <5KB)

**Risk 4: Object key ordering**
- **Note**: JSON spec doesn't guarantee key order
- **Mitigation**: Don't test exact string equality, parse both and compare structure

## Dependencies

**Before this milestone:**
- M-DX1 (New Builtin Registry) - ✅ Complete (v0.3.10)

**After this milestone:**
- v0.3.23 prompt update (reference JSON encoding in teaching prompt)
- Re-run api_call_json benchmark to verify fix

## Version Planning

**v0.3.23 Changes:**
1. Add `_json_encode` builtin
2. Uncomment `encode()` in std/json.ail
3. Update CHANGELOG.md
4. Update M-EVAL-HTTP-FIX-FINDINGS.md with resolution

**Breaking Changes:** None (purely additive)

## Related Documents

- [M-DX1: New Builtin Registry](implemented/v0_3_10/m-dx1-easier-ailang-dev.md)
- [M-EVAL-HTTP-FIX Sprint Plan](planned/M-EVAL-HTTP-FIX-sprint-plan.md)
- [M-EVAL-HTTP-FIX Findings](M-EVAL-HTTP-FIX-FINDINGS.md)
- [std/json.ail](std/json.ail) - Current implementation with encode() commented out
- [internal/builtins/json_decode.go](internal/builtins/json_decode.go) - Reference implementation

## Success Criteria

✅ **Definition of Done:**
1. `_json_encode` builtin registered and passing all tests
2. `std/json.ail` exports working `encode()` function
3. All helper functions (jo, kv, js, jnum, ja, jb) work end-to-end
4. Roundtrip tests pass: `decode(encode(x)) == Ok(x)`
5. api_call_json benchmark succeeds (no IMP010 errors)
6. Full test suite passes: `make test`
7. Linting passes: `make lint`
8. Documentation updated (CHANGELOG, comments, metadata)

---

*Created: 2025-10-27*
*Blocks: M-EVAL-HTTP-FIX Milestone 3 completion*
*Priority: P0 - Critical blocker for API benchmarks*
