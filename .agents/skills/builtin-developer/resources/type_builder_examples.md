# Type Builder Examples

Complete examples of using the Type Builder DSL for AILANG builtins.

## Basic Types

### Pure String Function

```go
func makeStrLenType() types.Type {
    T := types.NewBuilder()
    return T.Func(T.String()).Returns(T.Int())
}
```

### Multi-Argument Function

```go
func makeStrSliceType() types.Type {
    T := types.NewBuilder()
    return T.Func(
        T.String(),  // string to slice
        T.Int(),     // start index
        T.Int(),     // end index
    ).Returns(T.String())
}
```

## Complex Types with Records

### HTTP Request Function

```go
func makeHTTPRequestType() types.Type {
    T := types.NewBuilder()

    // Define header type: {name: string, value: string}
    headerType := T.Record(
        types.Field("name", T.String()),
        types.Field("value", T.String()),
    )

    // Define response type: {status: int, headers: [Header], body: string}
    responseType := T.Record(
        types.Field("status", T.Int()),
        types.Field("headers", T.List(headerType)),
        types.Field("body", T.String()),
    )

    // Function signature with Result type
    return T.Func(
        T.String(),           // url
        T.String(),           // method
        T.List(headerType),   // headers
        T.String(),           // body
    ).Returns(
        T.App("Result", responseType, T.Con("NetError")),
    ).Effects("Net")
}
```

### JSON Decode Function

```go
func makeJSONDecodeType() types.Type {
    T := types.NewBuilder()

    // Use type variable for polymorphic return
    alpha := T.Var("α")

    // Result[α, JSONError]
    resultType := T.App("Result", alpha, T.Con("JSONError"))

    return T.Func(T.String()).Returns(resultType)
}
```

## List and Tuple Types

### List Operations

```go
func makeListMapType() types.Type {
    T := types.NewBuilder()

    alpha := T.Var("α")
    beta := T.Var("β")

    // (α -> β) -> [α] -> [β]
    return T.Func(
        T.Func(alpha).Returns(beta),  // mapper function
        T.List(alpha),                 // input list
    ).Returns(T.List(beta))
}
```

### Tuple Constructor

```go
func makeTupleType() types.Type {
    T := types.NewBuilder()

    alpha := T.Var("α")
    beta := T.Var("β")

    // α -> β -> (α, β)
    return T.Func(alpha, beta).Returns(
        T.Tuple(alpha, beta),
    )
}
```

## Effect Functions

### File System Read

```go
func makeFSReadFileType() types.Type {
    T := types.NewBuilder()

    // Result[string, FSError]
    resultType := T.App("Result", T.String(), T.Con("FSError"))

    return T.Func(T.String()).Returns(resultType).Effects("FS")
}
```

### Multiple Effects

```go
func makeHTTPWithLoggingType() types.Type {
    T := types.NewBuilder()

    headerType := T.Record(
        types.Field("name", T.String()),
        types.Field("value", T.String()),
    )

    responseType := T.Record(
        types.Field("status", T.Int()),
        types.Field("body", T.String()),
    )

    return T.Func(
        T.String(),
        T.String(),
        T.List(headerType),
    ).Returns(
        T.App("Result", responseType, T.Con("NetError")),
    ).Effects("Net", "IO")  // Multiple effects
}
```

## Advanced Patterns

### Option/Maybe Types

```go
func makeFindType() types.Type {
    T := types.NewBuilder()

    alpha := T.Var("α")

    // (α -> bool) -> [α] -> Option[α]
    return T.Func(
        T.Func(alpha).Returns(T.Bool()),  // predicate
        T.List(alpha),                     // list
    ).Returns(
        T.App("Option", alpha),  // Option[α]
    )
}
```

### Nested Records

```go
func makeNestedRecordType() types.Type {
    T := types.NewBuilder()

    // Inner record: {x: int, y: int}
    pointType := T.Record(
        types.Field("x", T.Int()),
        types.Field("y", T.Int()),
    )

    // Outer record: {topleft: Point, bottomright: Point}
    rectType := T.Record(
        types.Field("topleft", pointType),
        types.Field("bottomright", pointType),
    )

    return T.Func().Returns(rectType)
}
```

### Higher-Order Functions

```go
func makeComposeType() types.Type {
    T := types.NewBuilder()

    alpha := T.Var("α")
    beta := T.Var("β")
    gamma := T.Var("γ")

    // (β -> γ) -> (α -> β) -> α -> γ
    return T.Func(
        T.Func(beta).Returns(gamma),   // f
        T.Func(alpha).Returns(beta),   // g
        alpha,                          // x
    ).Returns(gamma)
}
```

## Type Builder API Quick Reference

### Primitive Types
- `T.String()` → `string`
- `T.Int()` → `int`
- `T.Float()` → `float`
- `T.Bool()` → `bool`
- `T.Unit()` → `()`

### Compound Types
- `T.List(elementType)` → `[T]`
- `T.Tuple(t1, t2, ...)` → `(T1, T2, ...)`
- `T.Record(fields...)` → `{field1: T1, field2: T2, ...}`

### Type Constructors
- `T.Con("TypeName")` → Named type constructor
- `T.Var("α")` → Type variable (polymorphism)
- `T.App("Con", arg1, arg2)` → Type application (e.g., `Result[T, E]`)

### Function Types
- `T.Func(arg1, arg2, ...).Returns(retType)` → `arg1 -> arg2 -> retType`
- `.Effects("IO", "FS")` → Add effect annotation

### Field Construction (for Records)
```go
types.Field("fieldName", fieldType)
```

## Common Mistakes

### ❌ Wrong: Forgetting Returns()

```go
// This doesn't compile!
T.Func(T.String())  // Missing .Returns()
```

### ✅ Correct: Always use Returns()

```go
T.Func(T.String()).Returns(T.Int())
```

### ❌ Wrong: Effects without Returns()

```go
// This doesn't work!
T.Func(T.String()).Effects("IO")
```

### ✅ Correct: Returns() before Effects()

```go
T.Func(T.String()).Returns(T.Unit()).Effects("IO")
```

### ❌ Wrong: Creating Record fields directly

```go
// Don't do this!
T.Record(map[string]types.Type{
    "name": T.String(),
})
```

### ✅ Correct: Use types.Field()

```go
T.Record(
    types.Field("name", T.String()),
    types.Field("age", T.Int()),
)
```

## See Also

- **Testing**: [testing_patterns.md](testing_patterns.md) - How to test these builtins
- **Registry**: `internal/builtins/spec.go` - Registration system
- **Examples**: `internal/builtins/*.go` - Real builtin implementations
