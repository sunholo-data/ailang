# Lambda Expressions Example Refactor

**Status:** Planned
**Target Version:** v0.3.15+
**Discovered:** October 2025 (during Phase 2.6 example updates)

## Problem

The `examples/snippets/lambda_expressions.ail` file is a comprehensive lambda tutorial with 187 lines, but it's structured as multiple top-level `let` expressions rather than as a single entry module.

### Current Structure

```ailang
-- Top-level expression
print("=== Lambda Expressions ===")

let x = ... in
print(...)

let y = ... in
print(...)

-- 30+ separate let-in-print sequences
```

### Issue

This structure doesn't fit the **entry-module pattern**:

```ailang
module examples/snippets/lambda_expressions

export func main() -> () ! {IO} = {
  -- Need to convert 30+ let-in-print to block expression
  -- But sequential let bindings don't compose well
}
```

**Problem:** Converting to a single block expression requires either:
1. Deep nesting (hard to read)
2. Block expression syntax with semicolons (verbose, loses tutorial structure)
3. Splitting into multiple functions (changes pedagogical intent)

## Impact

**Severity:** Low
**User Impact:** One example file unusable (186 others work fine)
**AI Impact:** Low (file is tutorial/demo, not critical functionality test)

## Current Workaround

File is committed with partial conversion but marked as needing manual cleanup in commit message c78f290.

## Proposed Solutions

### Option 1: Split into Multiple Example Files (Recommended)

Break the monolithic file into focused examples:

```
examples/snippets/showcase/
├── lambdas_basic.ail          # Basic syntax (10 lines)
├── lambdas_curried.ail        # Curried functions (15 lines)
├── lambdas_closures.ail       # Closures (20 lines)
├── lambdas_higher_order.ail   # Higher-order functions (25 lines)
├── lambdas_records.ail        # Lambdas with records (20 lines)
└── lambdas_advanced.ail       # Advanced patterns (25 lines)
```

**Benefits:**
- Each file has clear focus
- Easier to run individually
- Better for testing specific features
- Matches existing showcase structure

**Effort:** ~1 hour (mostly copy-paste and minor edits)

### Option 2: Convert to Tutorial Script

Keep as single file but use block expression properly:

```ailang
module examples/snippets/lambda_expressions

export func main() -> () ! {IO} = {
  print("=== Basic Lambda Syntax ===");

  -- Identity function
  let id = \x. x in {
    print("Identity: " ++ show(id(42)));

    -- Simple arithmetic lambda
    let add_one = \x. x + 1 in {
      print("Add one: " ++ show(add_one(5)));

      -- Continue nesting...
    }
  }
}
```

**Benefits:**
- Keeps tutorial structure
- Single file for comprehensive reference

**Drawbacks:**
- Deep nesting (hard to read)
- Verbose semicolon syntax
- Against AILANG's compositional style

**Effort:** ~30 minutes

### Option 3: Use Helper Functions

Define helper functions and call them from main:

```ailang
module examples/snippets/lambda_expressions

func demo_basic() -> () ! {IO} = {
  print("=== Basic Lambda Syntax ===");
  let id = \x. x in
  print("Identity: " ++ show(id(42)))
}

func demo_curried() -> () ! {IO} = {
  -- ...
}

export func main() -> () ! {IO} = {
  demo_basic();
  demo_curried();
  -- etc.
}
```

**Benefits:**
- Clean structure
- Each section is testable independently
- Matches functional style

**Drawbacks:**
- More verbose function declarations
- Loses "inline tutorial" feel

**Effort:** ~45 minutes

## Recommendation

**Choose Option 1** (Split into Multiple Files)

**Rationale:**
- Aligns with existing showcase structure
- Each file is runnable and testable
- Better pedagogical organization
- Easier to maintain
- More discoverable (clear file names)

## Implementation Plan

1. **Create new files** (~30 min):
   - `lambdas_basic.ail` - identity, simple lambdas
   - `lambdas_curried.ail` - currying, partial application
   - `lambdas_closures.ail` - environment capture
   - `lambdas_higher_order.ail` - functions as values
   - `lambdas_records.ail` - lambdas with records
   - `lambdas_advanced.ail` - Y-combinator, conditionals

2. **Extract content** (~20 min):
   - Copy relevant sections from original file
   - Add proper module declarations
   - Convert to entry module pattern
   - Add descriptive comments

3. **Test each file** (~10 min):
   - Run with `ailang run --caps IO --entry main`
   - Verify output matches expected behavior
   - Update comments if needed

4. **Clean up** (~5 min):
   - Archive or delete original `lambda_expressions.ail`
   - Update any references in docs
   - Commit with clear message

## File Structure (Detailed)

### lambdas_basic.ail
```ailang
-- Showcase: Basic Lambda Syntax
module examples/snippets/showcase/lambdas_basic

export func main() -> () ! {IO} = {
  print("=== Basic Lambda Syntax ===");

  let id = \x. x in
  print("Identity: " ++ show(id(42)));

  let add_one = \x. x + 1 in
  print("Add one: " ++ show(add_one(5)));

  let complex = \x. x * 2 + 1 in
  print("Complex: " ++ show(complex(3)))
}
```

### lambdas_curried.ail
```ailang
-- Showcase: Curried Functions
module examples/snippets/showcase/lambdas_curried

export func main() -> () ! {IO} = {
  print("=== Curried Functions ===");

  let add = \x y. x + y in
  print("Curried add: " ++ show(add(3)(4)));

  let multiply = \x y. x * y in
  let double = multiply(2) in
  print("Partial application: " ++ show(double(5)))
}
```

**Continue for other 4 files...**

## Alternative: Keep as Documentation

If the file is primarily documentation rather than executable example:

1. Move to `docs/examples/` or `docs/tutorials/`
2. Convert to markdown with embedded code blocks
3. Use for website documentation
4. Keep working examples in `examples/snippets/showcase/`

## Related Work

- Existing showcase examples already follow this pattern:
  - `type_inference.ail` - focused, single concept
  - `lambdas.ail` - composition example
  - `closures.ail` - environment capture
  - `type_classes.ail` - type class demo

- This refactor would **extend** that pattern with more comprehensive lambda examples

## Success Criteria

- ✅ All 6 new files run successfully
- ✅ Each file demonstrates specific lambda feature
- ✅ Total pass rate increases (6 working examples vs 1 broken)
- ✅ Documentation updated if needed
- ✅ Original comprehensive content preserved (just reorganized)

## References

- Original file: `examples/snippets/lambda_expressions.ail` (187 lines)
- Partial conversion commit: c78f290
- Related design: `design_docs/planned/v0_3_15/example-parity-vision-alignment.md`
- Showcase pattern: `examples/snippets/showcase/*.ail`
