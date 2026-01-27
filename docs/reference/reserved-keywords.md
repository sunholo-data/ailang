# Reserved Keywords Reference

This document lists all 43 reserved keywords in AILANG. These words cannot be used as variable or function names.

## Complete Keyword List

### Control Flow (7 keywords)
- `if` - Conditional expression: `if condition then a else b`
- `then` - True branch of if expression
- `else` - False branch of if expression
- `match` - Pattern matching: `match expr { pat1 => e1, pat2 => e2 }`
- `with` - Pattern guard in match (future feature)
- `select` - CSP channel selection (future feature)
- `timeout` - Timeout on channel operation (future feature)

### Function & Binding (5 keywords)
- `func` - Function definition: `func name(args) -> type { body }`
- `pure` - Pure (effect-free) function marker: `pure func`
- `let` - Variable binding: `let x = expr; body`
- `letrec` - Recursive binding: `letrec f = expr in body`
- `in` - Scope terminator for let/letrec expressions

### Type System (6 keywords)
- `type` - Type/ADT definition: `type Tree = | Leaf | Node`
- `class` - Type class definition (future feature)
- `instance` - Type class instance (future feature)
- `forall` - Universal type quantification: `forall[T]`
- `exists` - Existential type quantification (parser keyword)
- `deriving` - Type class derivation: `deriving (Show, Eq)`

### Module System (4 keywords)
- `module` - Module declaration: `module path/to/module`
- `import` - Import declaration: `import std/list (map, filter)`
- `export` - Mark definition as public: `export func foo`
- `extern` - Interop with Go: `extern func name(args) -> type`

### Testing (4 keywords)
- `test` - Single test: `test "description" { ... }`
- `tests` - Test block (contextual keyword)
- `property` - Property-based test (contextual keyword)
- `properties` - Property test block (contextual keyword)
- `assert` - Test assertion: `assert(condition, "message")`

### Verification (3 keywords)
- `requires` - Precondition (M-VERIFY feature)
- `ensures` - Postcondition (M-VERIFY feature)
- `invariant` - Loop invariant (M-VERIFY feature)

### Concurrency (4 keywords)
- `spawn` - Create goroutine (future feature)
- `parallel` - Run in parallel (future feature)
- `channel` - Channel type: `channel[T]`
- `send` - Send to channel (future feature)
- `recv` - Receive from channel (future feature)

### Boolean & Logic (3 keywords)
- `true` - Boolean true value
- `false` - Boolean false value
- `and` - Logical AND: `a and b`
- `or` - Logical OR: `a or b`
- `not` - Logical NOT: `not x`

---

## Contextual Keywords

These keywords can **sometimes** be used as identifiers in specific contexts, but we **recommend avoiding them**:

| Keyword | Context | Description |
|---------|---------|-------------|
| `test` | After `func` keyword | `func test` declares a test, not a function named "test" |
| `tests` | At statement level | Introduces a tests block, not a variable |
| `property` | After `func` keyword | Similar to `test` |
| `properties` | At statement level | Introduces a properties block |

**Why avoid them?** Even though contextual parsing allows them in certain positions, using them as identifiers makes code confusing for readers.

**Example of confusion:**
```ailang
-- Confusing - does this declare a test or function?
func test() -> int = 42

-- Clear - avoid the keyword
func runTest() -> int = 42
```

---

## Common Mistakes

### Mistake 1: Using `exists` as Variable Name

```ailang
-- ❌ WRONG - 'exists' is reserved
let exists = fileExists(path);
-- Error: PAR_UNEXPECTED_TOKEN: expected next token to be IDENT, got exists instead
```

**Fix: Use alternative name**
```ailang
-- ✅ CORRECT
let found = fileExists(path);
let doesExist = fileExists(path);
let fileIsPresent = fileExists(path);
```

**Why is `exists` reserved?**
- Used for existential type quantification (future feature)
- Placeholder in parser for when existential types are implemented
- Similar to `forall` for universal types

### Mistake 2: Using Contextual Keywords

```ailang
-- ⚠️ CONFUSING - 'test' is contextual
func test(x: int) -> int = x + 1

-- ✅ BETTER - Avoid the keyword
func addOne(x: int) -> int = x + 1
```

### Mistake 3: Using Control Flow Keywords as Names

```ailang
-- ❌ WRONG
let if = 5;
let match = "pattern";

-- ✅ CORRECT
let condition = 5;
let pattern = "pattern";
```

---

## Discovering Undefined Keywords

If you get an error like:

```
PAR_UNEXPECTED_TOKEN: expected next token to be IDENT, got SOMEWORD instead
```

1. Check if `SOMEWORD` is in the keywords list above
2. If yes, use a different variable name
3. If no, the error is likely a different parse issue - check your syntax

---

## Why These Keywords Exist

### Already Implemented
- **Control Flow:** `if`, `then`, `else`, `match` - Core language features
- **Functions:** `func`, `let`, `pure` - Definition syntax
- **Types:** `type`, `forall` - Type system
- **Modules:** `module`, `import`, `export` - Module system
- **Testing:** `test`, `assert` - Built-in testing
- **Boolean:** `true`, `false`, `not`, `and`, `or` - Logic operators

### Reserved for Future Features
- **Concurrency:** `spawn`, `parallel`, `channel`, `send`, `recv` - Static task graphs (v0.4.0+)
- **Type Classes:** `class`, `instance`, `deriving` - Structural reflection (v0.4.0+)
- **Verification:** `requires`, `ensures`, `invariant` - Contract system (M-VERIFY)
- **Channels:** `select`, `timeout` - CSP operations (future)
- **Existentials:** `exists` - Existential type quantification

---

## Related Resources

- [Module System Guide](../../guides/modules.md) - How imports work
- [Pattern Matching](../../guides/pattern-matching.md) - Using `match` expressions
- [Type System](../../guides/types.md) - Type declarations with `type`
- [Function Definitions](../../guides/functions.md) - Creating functions with `func`

---

## See Also

- **Teaching Prompt:** Run `ailang prompt` to see complete syntax reference
- **Error Messages:** Run `ailang check file.ail` for detailed parse errors
- **Interactive Testing:** Use `ailang repl` to experiment with keywords
