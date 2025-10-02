# AILANG v0.1.0 Known Limitations

**Last Updated**: October 2, 2025

This document honestly describes what doesn't work yet in AILANG v0.1.0 and what's planned for future releases.

## TL;DR

**What Works**: Type system, parser, REPL, type-checking
**What Doesn't**: Module execution, effect handlers, stdlib function calls
**When Fixed**: v0.2.0 (Module Evaluation milestone)

---

## Critical Limitation: Module Execution

### What This Means

Currently, AILANG can:
- ✅ Parse module files
- ✅ Type-check module code
- ✅ Extract module interfaces
- ✅ Resolve dependencies
- ❌ **Execute module code**

### Impact

This affects:
1. **Standard Library**: Can't call stdlib functions (map, filter, println, etc.)
2. **Module Exports**: Can't run exported functions
3. **Demo Programs**: Examples type-check but don't execute
4. **Import/Export**: Can import but can't call imported functions

### Examples

**Type-checks but doesn't execute:**
```ailang
module example
import stdlib/std/list (map)

export func main() -> [int] {
  map(\x. x * 2, [1, 2, 3])
}
```

**Status:**
- `ailang check example.ail` ✅ Works
- `ailang run example.ail` ❌ Fails: "Module evaluation not yet supported"

### Why This Limitation Exists

Module execution requires:
1. Evaluating module bindings in dependency order
2. Building runtime environments with closures
3. Resolving imported function values
4. Managing effect capabilities

This is substantial infrastructure work that was deferred to v0.2.0 to ship v0.1.0 sooner.

### Workarounds

**Option 1: Use the REPL**
```bash
ailang repl
λ> let double = \x. x * 2
λ> double(21)
42
```
The REPL has full execution support!

**Option 2: Write non-module files**
```ailang
-- No module declaration
let double = \x. x * 2 in
print(show(double(21)))
```

Non-module files execute fine, but:
- Can't use `func` keyword
- Can't use `import`
- Can't use `type` definitions
- Limited to single expression

### When Will This Be Fixed?

**Target**: v0.2.0 (estimated 1-2 weeks after v0.1.0)

**Milestone**: M-E1 (Module Evaluation Environment)
- Build module evaluation context
- Implement dependency-order evaluation
- Wire up function value extraction
- Enable entrypoint execution

---

## Parser Limitations

### 1. Match Expressions Not Implemented

**Status**: Type system supports patterns, parser doesn't

**What doesn't work:**
```ailang
match value {
  Some(x) => x,
  None => 0
}
```

**Error**: `PAR_NO_PREFIX_PARSE: unexpected token: =>`

**Workaround**: Use if-then-else for now

**Planned**: v0.2.0

---

### 2. Nested Let Limit

**Status**: Parser has depth limit on nested `let...in` expressions

**What doesn't work:**
```ailang
let a = 1 in
let b = 2 in
let c = 3 in
let d = 4 in  -- 4th level fails!
a + b + c + d
```

**Error**: `PAR_NO_PREFIX_PARSE at line N: unexpected token in expression: in`

**Workaround**: Keep lets to 3 levels or less, or use modules

**Planned**: Fix in v0.1.1

---

### 3. Properties Syntax Not Finalized

**Status**: Experimental syntax not stable

**What doesn't work:**
```ailang
property "associativity" {
  forall(x: int, y: int, z: int) =>
    (x + y) + z == x + (y + z)
}
```

**Planned**: v0.3.0 (Property-based testing milestone)

---

## Type System Limitations

### 1. Record Field Access Issues

**Status**: Some record operations have unification bugs

**What sometimes fails:**
```ailang
let person = {name: "Alice", age: 30} in
person.name  -- May fail with unification error
```

**Workaround**: Use pattern matching when available

**Planned**: Fix in v0.1.1

---

### 2. Effect Handlers Not Implemented

**Status**: Effect **tracking** works, effect **handling** doesn't

**What works:**
```ailang
-- Effect annotations in types
func readConfig() -> Config ! {IO, FS}
```

**What doesn't work:**
```ailang
-- Effect handlers
handle {
  readConfig()
} with {
  IO => mockIO,
  FS => mockFS
}
```

**Planned**: v0.2.0 (with module execution)

---

## Standard Library Limitations

### What Works

- ✅ Type signatures for all 32 exports
- ✅ Interface extraction and API freeze
- ✅ Type-checking code that uses stdlib
- ✅ Import resolution

### What Doesn't Work

- ❌ Calling stdlib functions
- ❌ Using ADT constructors (Some, None, Ok, Err)
- ❌ Running examples that import stdlib

### Modules Affected

1. **stdlib/std/io** (3 exports)
   - print, println, readLine
   - Status: Type-checks ✓, Executes ✗

2. **stdlib/std/list** (10 exports)
   - map, filter, fold, length, etc.
   - Status: Type-checks ✓, Executes ✗

3. **stdlib/std/option** (6 exports)
   - Some, None, map, getOrElse, etc.
   - Status: Type-checks ✓, Executes ✗

4. **stdlib/std/result** (6 exports)
   - Ok, Err, map, unwrap, etc.
   - Status: Type-checks ✓, Executes ✗

5. **stdlib/std/string** (7 exports)
   - toUpper, toLower, length, etc.
   - Status: Type-checks ✓, Executes ✗

**Reason**: All blocked by module execution limitation

---

## What DOES Work

It's important to note what v0.1.0 accomplishes:

### Type System (Complete!)

- ✅ Hindley-Milner type inference
- ✅ Type classes (Num, Eq, Ord, Show)
- ✅ Dictionary passing for type classes
- ✅ Row polymorphism
- ✅ Effect tracking (type-level)
- ✅ Algebraic data types
- ✅ Pattern types in signatures
- ✅ Polymorphic recursion

### Parser (75% Coverage)

- ✅ Module syntax
- ✅ Function declarations
- ✅ Type definitions
- ✅ Effect annotations
- ✅ Equation-form syntax
- ✅ Multi-statement blocks
- ✅ All expressions

### REPL (Fully Functional!)

- ✅ Interactive type checking
- ✅ Full evaluation
- ✅ Type class inference
- ✅ Command history
- ✅ Multi-line input
- ✅ All stdlib types available

### Infrastructure

- ✅ Module dependency graph
- ✅ Interface extraction
- ✅ API stability verification (SHA256)
- ✅ JSON argument decoder
- ✅ Entrypoint resolution
- ✅ Comprehensive error messages

---

## Migration Path to v0.2.0

When v0.2.0 ships with module execution:

### Code That Will "Just Work"

All code that type-checks in v0.1.0 will execute in v0.2.0:

```ailang
module example
import stdlib/std/list (map)

export func main() -> [int] {
  map(\x. x * 2, [1, 2, 3])
}
```

**v0.1.0**: Type-checks ✓
**v0.2.0**: Executes ✓

### No Breaking Changes Expected

v0.2.0 will be a **pure addition** of functionality:
- Same type system
- Same parser
- Same stdlib API
- **Plus**: Module execution

### Recommended Development Approach

1. **Write code in v0.1.0**:
   - Use modules
   - Import stdlib
   - Write clean, modern code

2. **Type-check with `ailang check`**:
   - Verify types are correct
   - Catch errors early

3. **Test in REPL**:
   - Prototype logic interactively
   - Verify algorithms work

4. **Wait for v0.2.0**:
   - Your code will run without changes!

---

## Frequently Asked Questions

### Q: Can I use AILANG productively in v0.1.0?

**A**: Yes, for:
- Learning the type system
- Prototyping in the REPL
- Writing non-module scripts
- Preparing code for v0.2.0

### Q: When is v0.2.0 coming?

**A**: Estimated 1-2 weeks after v0.1.0 release.

### Q: Will my v0.1.0 code break in v0.2.0?

**A**: No. v0.2.0 only adds execution support. If it type-checks in v0.1.0, it will run in v0.2.0.

### Q: Should I wait for v0.2.0?

**A**: Depends on your use case:
- **Learning**: v0.1.0 is great
- **REPL usage**: v0.1.0 works perfectly
- **Production code**: Wait for v0.2.0

### Q: What about v0.3.0 and beyond?

**A**: Roadmap includes:
- v0.2.0: Module execution + effect handlers
- v0.3.0: Property-based testing
- v0.4.0: Concurrency (CSP)
- v0.5.0: Quasiquotes
- v1.0.0: Production-ready

---

## Getting Help

- **Documentation**: See `docs/` directory
- **Examples**: See `examples/STATUS.md` for working examples
- **REPL Help**: Type `:help` in the REPL
- **Issue Tracker**: https://github.com/sunholo-data/ailang/issues

---

## Conclusion

v0.1.0 represents a **complete type system** with an **incomplete runtime**.

This is intentional - we wanted to ship a solid foundation that demonstrates AILANG's type-level features before tackling the complexity of execution.

**The type system works beautifully. Execution is coming soon.**

---

*For details on what IS working, see the main [README.md](../README.md)*
