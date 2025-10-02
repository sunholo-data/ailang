# Showcase Example Creation Issues (v0.1.0)

**Date**: October 2, 2025
**Context**: Creating polished showcase examples for v0.1.0 release

## Issues Encountered

### 1. Multiple `let...in` Syntax Limitation

**Issue**: Cannot use multiple `let...in` statements in sequence at the top level without proper nesting.

**What Doesn't Work**:
```ailang
let x = 1 in
-- comment here
let y = 2 in
-- another comment
x + y
```

**Error**: `PAR_NO_PREFIX_PARSE: unexpected token in expression: in`

**Root Cause**: The parser expects each `in` to be followed immediately by the continuation expression. Comments between nested lets confuse the parser.

**What Works**:
```ailang
let x = 1 in
let y = 2 in
x + y
```

**Workaround**:
- Remove intermediate comments between nested lets
- Or use block syntax with modules (requires module execution in v0.2.0)
- Keep all lets tightly nested without intervening content

**Impact**:
- Showcase examples need to be more compact
- Can't intersperse educational comments between definitions
- Reduces readability of tutorial-style examples

**Recommendation for v0.2.0**:
- Consider allowing comments between `let` bindings
- Or provide better error messages indicating the nesting requirement
- Or support top-level `let` statements in files (without `in`)

---

### 2. Non-Module File Limitations

**Issue**: Simple expression files (non-modules) are limited to single expressions.

**Current Reality**:
- ✅ Can use: Nested `let...in` expressions
- ✅ Can use: Single complex expression
- ❌ Cannot use: Multiple top-level statements
- ❌ Cannot use: Function definitions (func keyword)
- ❌ Cannot use: Type definitions

**Example - What Doesn't Work**:
```ailang
-- Can't have multiple statements at top level
func helper(x: int) -> int { x * 2 }
helper(5)
```

**Error**: Parse error - `func` requires module context

**Workaround**:
- Use lambda expressions instead: `let helper = \x. x * 2 in ...`
- Or create a module (but then execution blocked until v0.2.0)

**Impact**:
- Showcase examples must use lambda syntax exclusively
- Can't demonstrate `func` syntax in executable examples
- Educational examples can't show "proper" function definitions

**Recommendation for v0.2.0**:
- Allow top-level `func` in non-module files
- Or provide "script mode" that treats file as implicit module with execution

---

### 3. Module vs Non-Module Trade-off

**Issue**: Forced choice between executability and modern syntax.

**Current Situation**:

| Feature | Non-Module Files | Module Files |
|---------|-----------------|--------------|
| Execute | ✅ Yes | ❌ No (v0.2.0) |
| `func` keyword | ❌ No | ✅ Yes |
| `type` definitions | ❌ No | ✅ Yes |
| `export` | ❌ No | ✅ Yes |
| Multiple statements | ❌ No | ✅ Yes |
| Stdlib imports | ❌ No | ✅ Yes |
| Clean syntax | ❌ No | ✅ Yes |

**Impact**:
- For v0.1.0, executable examples must use "ugly" nested lambda syntax
- Modern, clean examples don't execute
- Documentation must explain this awkward trade-off

**Example Trade-off**:

*Clean but non-executable*:
```ailang
module showcase
import stdlib/std/list (map)

pure func double(x: int) -> int {
  x * 2
}

export func main() -> [int] {
  map(double, [1, 2, 3])
}
```

*Executable but awkward*:
```ailang
let double = \x. x * 2 in
let xs = [1, 2, 3] in
-- Can't use stdlib map...
xs  -- Just return the list
```

**Recommendation**:
- v0.2.0: Module execution removes this trade-off
- v0.1.0: Document honestly, provide both styles

---

### 4. Stdlib Function Calls in Examples

**Issue**: Can't demonstrate stdlib usage in executable examples.

**Current State**:
- ✅ Stdlib modules type-check perfectly
- ✅ Can import and reference functions
- ❌ Can't actually call stdlib functions

**Example**:
```ailang
module example
import stdlib/std/list (map, filter)

export func main() -> [int] {
  let double = \x. x * 2 in
  let numbers = [1, 2, 3, 4, 5] in
  let doubled = map(double, numbers) in  -- Type-checks ✓, executes ✗
  doubled
}
```

**Status**: `ailang check` works, `ailang run` fails

**Impact**:
- Showcase examples can't demonstrate real stdlib usage
- Must show type signatures only
- Users can't "try it out" with stdlib

**Workaround**:
- Show stdlib type signatures with comments
- Demonstrate similar operations with lambdas
- Mark clearly as "v0.2.0 feature"

---

### 5. Pattern Matching Syntax

**Issue**: Match expressions not implemented in parser.

**Current State**:
- ✅ Type system understands patterns
- ✅ ADT constructors work
- ❌ `match` keyword not parsed

**Example - Doesn't Parse**:
```ailang
match opt {
  Some(x) => x,
  None => 0
}
```

**Error**: `PAR_NO_PREFIX_PARSE: unexpected token: =>`

**Impact**:
- Can show ADT types but not pattern matching usage
- Must use hypothetical examples in comments
- Pattern matching showcase is theoretical only

**Recommendation for v0.2.0**:
- Implement `match` expression parser
- Add exhaustiveness checking
- Enable runtime pattern matching

---

## Lessons Learned

### For Showcase Examples

1. **Keep it Simple**: Single-expression examples work best
2. **Lambda Over Func**: Use lambdas for executable examples
3. **Comment Carefully**: Don't put comments between nested lets
4. **Set Expectations**: Clearly mark what works vs what's coming

### For Documentation

1. **Be Honest**: Clearly state what doesn't work
2. **Show Both**: Provide clean (non-executable) and working (awkward) versions
3. **Future Vision**: Show what v0.2.0 will enable
4. **Type Signatures**: When execution doesn't work, show types

### For v0.2.0 Planning

1. **Module Execution**: Top priority - unblocks clean examples
2. **Parser Improvements**: Allow more flexible syntax
3. **Script Mode**: Consider implicit module wrapping
4. **Better Errors**: Guide users to correct syntax

---

## Recommendations for v0.1.0 Release

### Documentation Strategy

1. **Dual Examples**:
   - "Working Now" - executable but uses lambdas
   - "Coming Soon" - clean syntax, marked as v0.2.0

2. **Clear Warnings**:
   ```ailang
   -- ⚠️ NOTE: This example type-checks but cannot execute until v0.2.0
   -- Current status: Parses ✓, Type-checks ✓, Executes ✗
   ```

3. **REPL First**:
   - Encourage REPL usage for exploration
   - REPL has full execution support
   - Perfect for interactive learning

### Example Organization

```
examples/
├── working/          # Executable examples (lambda syntax)
├── showcase/         # Clean examples (type-check only)
├── modules/          # Module system examples (v0.2.0)
└── experimental/     # Future features
```

---

## Summary

The showcase example creation revealed important UX gaps in v0.1.0:

1. **Parser limitations** make tutorial examples awkward
2. **Module execution gap** forces trade-off between clean syntax and executability
3. **Documentation must be honest** about these limitations
4. **v0.2.0 will dramatically improve** the example experience

**Verdict**: Ship v0.1.0 with honest documentation, focus v0.2.0 on execution.

---

*This document will inform v0.2.0 planning and help set realistic expectations for v0.1.0.*
