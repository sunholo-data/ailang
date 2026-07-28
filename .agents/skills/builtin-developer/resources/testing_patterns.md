# Testing Patterns for AILANG Builtins

Complete guide to hermetic testing using the Test Harness (`internal/effects/testctx/`).

## Basic Test Structure

### Pure Function Test

```go
func TestStrReverse(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    tests := []struct {
        input    string
        expected string
    }{
        {"hello", "olleh"},
        {"", ""},
        {"🎉", "🎉"},
        {"abc", "cba"},
    }

    for _, tt := range tests {
        result, err := strReverseImpl(ctx, []eval.Value{
            testctx.MakeString(tt.input),
        })
        assert.NoError(t, err)
        assert.Equal(t, tt.expected, testctx.GetString(result))
    }
}
```

### Multi-Argument Function Test

```go
func TestStrSlice(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    tests := []struct {
        str      string
        start    int
        end      int
        expected string
    }{
        {"hello", 0, 3, "hel"},
        {"world", 1, 4, "orl"},
        {"test", 0, 4, "test"},
    }

    for _, tt := range tests {
        result, err := strSliceImpl(ctx, []eval.Value{
            testctx.MakeString(tt.str),
            testctx.MakeInt(tt.start),
            testctx.MakeInt(tt.end),
        })
        assert.NoError(t, err)
        assert.Equal(t, tt.expected, testctx.GetString(result))
    }
}
```

## Testing with Records

### Simple Record

```go
func TestRecordBuiltin(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    // Create input record
    input := testctx.MakeRecord(map[string]eval.Value{
        "name": testctx.MakeString("Alice"),
        "age":  testctx.MakeInt(30),
    })

    result, err := myRecordBuiltin(ctx, []eval.Value{input})
    assert.NoError(t, err)

    // Extract and verify output record
    output := testctx.GetRecord(result)
    assert.Equal(t, "Alice", testctx.GetString(output["name"]))
    assert.Equal(t, 31, testctx.GetInt(output["age"]))
}
```

### Nested Records

```go
func TestNestedRecord(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    // Create nested record
    point := testctx.MakeRecord(map[string]eval.Value{
        "x": testctx.MakeInt(10),
        "y": testctx.MakeInt(20),
    })

    rect := testctx.MakeRecord(map[string]eval.Value{
        "topleft":     point,
        "bottomright": testctx.MakeRecord(map[string]eval.Value{
            "x": testctx.MakeInt(100),
            "y": testctx.MakeInt(200),
        }),
    })

    result, err := processRect(ctx, []eval.Value{rect})
    assert.NoError(t, err)

    // Extract nested values
    output := testctx.GetRecord(result)
    topleft := testctx.GetRecord(output["topleft"])
    assert.Equal(t, 10, testctx.GetInt(topleft["x"]))
}
```

## Testing with Lists

### Simple List

```go
func TestListOperation(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    // Create list of integers
    input := testctx.MakeList([]eval.Value{
        testctx.MakeInt(1),
        testctx.MakeInt(2),
        testctx.MakeInt(3),
    })

    result, err := sumList(ctx, []eval.Value{input})
    assert.NoError(t, err)
    assert.Equal(t, 6, testctx.GetInt(result))
}
```

### List of Records

```go
func TestListOfRecords(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    people := testctx.MakeList([]eval.Value{
        testctx.MakeRecord(map[string]eval.Value{
            "name": testctx.MakeString("Alice"),
            "age":  testctx.MakeInt(30),
        }),
        testctx.MakeRecord(map[string]eval.Value{
            "name": testctx.MakeString("Bob"),
            "age":  testctx.MakeInt(25),
        }),
    })

    result, err := getNames(ctx, []eval.Value{people})
    assert.NoError(t, err)

    names := testctx.GetList(result)
    assert.Equal(t, 2, len(names))
    assert.Equal(t, "Alice", testctx.GetString(names[0]))
    assert.Equal(t, "Bob", testctx.GetString(names[1]))
}
```

## Testing Effect Functions

### HTTP Request (Hermetic)

```go
func TestNetHTTPRequest(t *testing.T) {
    // Create test HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "GET", r.Method)
        assert.Equal(t, "/api/test", r.URL.Path)

        // Return response
        w.WriteHeader(200)
        w.Write([]byte(`{"status": "ok"}`))
    }))
    defer server.Close()

    // Create mock context with HTTP capability
    ctx := testctx.NewMockEffContext()
    ctx.GrantAll("Net")
    ctx.SetHTTPClient(server.Client())

    // Create request arguments
    url := testctx.MakeString(server.URL + "/api/test")
    method := testctx.MakeString("GET")
    headers := testctx.MakeList([]eval.Value{})
    body := testctx.MakeString("")

    // Execute builtin
    result, err := effects.NetHTTPRequest(ctx,
        url,
        method,
        headers,
        body,
    )

    // Verify result
    assert.NoError(t, err)
    resp := testctx.GetRecord(result)
    assert.Equal(t, 200, testctx.GetInt(resp["status"]))
    assert.Contains(t, testctx.GetString(resp["body"]), "ok")
}
```

### File System Operations (Mock)

```go
func TestFSReadFile(t *testing.T) {
    ctx := testctx.NewMockEffContext()
    ctx.GrantAll("FS")

    // Note: In real tests, you'd mock the filesystem
    // For now, this shows the pattern
    path := testctx.MakeString("/tmp/test.txt")

    result, err := fsReadFile(ctx, []eval.Value{path})

    // Handle Result[string, FSError]
    assert.NoError(t, err)
    resultVariant := testctx.GetVariant(result)
    if resultVariant.Tag == "Ok" {
        content := testctx.GetString(resultVariant.Value)
        assert.NotEmpty(t, content)
    }
}
```

### Testing Capability Denial

```go
func TestCapabilityDenied(t *testing.T) {
    ctx := testctx.NewMockEffContext()
    // Don't grant Net capability

    url := testctx.MakeString("http://example.com")
    method := testctx.MakeString("GET")
    headers := testctx.MakeList([]eval.Value{})
    body := testctx.MakeString("")

    result, err := effects.NetHTTPRequest(ctx, url, method, headers, body)

    // Should fail with capability error
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "capability")
}
```

## Testing Result Types

### Ok Variant

```go
func TestResultOk(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    result, err := myBuiltin(ctx, []eval.Value{
        testctx.MakeString("valid-input"),
    })

    assert.NoError(t, err)

    // Extract Result variant
    variant := testctx.GetVariant(result)
    assert.Equal(t, "Ok", variant.Tag)

    // Get the Ok value
    value := testctx.GetString(variant.Value)
    assert.Equal(t, "expected", value)
}
```

### Err Variant

```go
func TestResultErr(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    result, err := myBuiltin(ctx, []eval.Value{
        testctx.MakeString("invalid-input"),
    })

    assert.NoError(t, err)  // Builtin executed successfully

    // Extract Result variant
    variant := testctx.GetVariant(result)
    assert.Equal(t, "Err", variant.Tag)

    // Get the Err value
    errValue := testctx.GetString(variant.Value)
    assert.Contains(t, errValue, "invalid")
}
```

## Testing Option Types

### Some Variant

```go
func TestOptionSome(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    list := testctx.MakeList([]eval.Value{
        testctx.MakeInt(1),
        testctx.MakeInt(2),
        testctx.MakeInt(3),
    })

    result, err := findInList(ctx, []eval.Value{
        testctx.MakeInt(2),
        list,
    })

    assert.NoError(t, err)

    variant := testctx.GetVariant(result)
    assert.Equal(t, "Some", variant.Tag)
    assert.Equal(t, 2, testctx.GetInt(variant.Value))
}
```

### None Variant

```go
func TestOptionNone(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    list := testctx.MakeList([]eval.Value{
        testctx.MakeInt(1),
        testctx.MakeInt(2),
    })

    result, err := findInList(ctx, []eval.Value{
        testctx.MakeInt(99),  // Not in list
        list,
    })

    assert.NoError(t, err)

    variant := testctx.GetVariant(result)
    assert.Equal(t, "None", variant.Tag)
}
```

## Test Harness API Reference

### Value Constructors

```go
// Primitives
testctx.MakeString(s string) *eval.StringValue
testctx.MakeInt(i int) *eval.IntValue
testctx.MakeFloat(f float64) *eval.FloatValue
testctx.MakeBool(b bool) *eval.BoolValue
testctx.MakeUnit() *eval.UnitValue

// Compound types
testctx.MakeList(items []eval.Value) *eval.ListValue
testctx.MakeRecord(fields map[string]eval.Value) *eval.RecordValue
testctx.MakeTuple(items []eval.Value) *eval.TupleValue

// Variants (ADTs)
testctx.MakeVariant(tag string, value eval.Value) *eval.VariantValue
```

### Value Extractors

```go
// Primitives
testctx.GetString(v eval.Value) string
testctx.GetInt(v eval.Value) int
testctx.GetFloat(v eval.Value) float64
testctx.GetBool(v eval.Value) bool

// Compound types
testctx.GetList(v eval.Value) []eval.Value
testctx.GetRecord(v eval.Value) map[string]eval.Value
testctx.GetTuple(v eval.Value) []eval.Value

// Variants
testctx.GetVariant(v eval.Value) *eval.VariantValue
// Then access variant.Tag and variant.Value
```

### Mock Context Methods

```go
// Create context
ctx := testctx.NewMockEffContext()

// Grant capabilities
ctx.GrantAll(effect string)
ctx.RevokeAll(effect string)

// Mock HTTP
ctx.SetHTTPClient(client *http.Client)

// Mock Filesystem (future)
ctx.SetFS(fs FS)
```

## Common Patterns

### Table-Driven Tests

```go
func TestMathOperations(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    tests := []struct {
        name     string
        a        int
        b        int
        expected int
    }{
        {"positive", 5, 3, 8},
        {"negative", -5, -3, -8},
        {"zero", 0, 0, 0},
        {"mixed", -5, 10, 5},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := addInts(ctx, []eval.Value{
                testctx.MakeInt(tt.a),
                testctx.MakeInt(tt.b),
            })
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, testctx.GetInt(result))
        })
    }
}
```

### Error Testing

```go
func TestErrorHandling(t *testing.T) {
    ctx := testctx.NewMockEffContext()

    tests := []struct {
        name        string
        input       string
        expectError bool
        errorMsg    string
    }{
        {"valid", "valid", false, ""},
        {"invalid", "", true, "empty string"},
        {"too long", strings.Repeat("a", 1000), true, "too long"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := validateString(ctx, []eval.Value{
                testctx.MakeString(tt.input),
            })

            if tt.expectError {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errorMsg)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}
```

### Benchmark Tests

```go
func BenchmarkStrReverse(b *testing.B) {
    ctx := testctx.NewMockEffContext()
    input := testctx.MakeString("hello world")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := strReverseImpl(ctx, []eval.Value{input})
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Best Practices

1. **Always use MockEffContext** - Never use real effects in tests
2. **Test edge cases** - Empty strings, zero, negative numbers, etc.
3. **Use table-driven tests** - Makes adding test cases easy
4. **Test error conditions** - Not just happy path
5. **Use subtests** - For better test organization
6. **Hermetic HTTP tests** - Use httptest.NewServer()
7. **Don't test the harness** - Test your builtin logic
8. **Keep tests fast** - No network, no filesystem I/O
9. **Clear assertions** - Use descriptive error messages
10. **One assertion per concept** - Don't overload tests

## Common Mistakes

### ❌ Wrong: Using real HTTP

```go
// Don't do this!
resp, err := http.Get("https://api.example.com")
```

### ✅ Correct: Use httptest

```go
server := httptest.NewServer(handler)
defer server.Close()
ctx.SetHTTPClient(server.Client())
```

### ❌ Wrong: Type assertion without checking

```go
// This can panic!
str := result.(*eval.StringValue).Value
```

### ✅ Correct: Use testctx extractors

```go
str := testctx.GetString(result)
```

### ❌ Wrong: Testing without EffContext

```go
// Don't skip the context!
result := myFunction(args)
```

### ✅ Correct: Always use EffContext

```go
ctx := testctx.NewMockEffContext()
result, err := myFunction(ctx, args)
```

## See Also

- **Type Builder**: [type_builder_examples.md](type_builder_examples.md) - How to define types
- **Test Harness**: `internal/effects/testctx/` - Implementation
- **Examples**: `internal/builtins/*_test.go` - Real test examples
