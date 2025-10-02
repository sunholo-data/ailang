# AILANG Examples

This directory contains example programs demonstrating AILANG features.

## Quick Start

**Run an example:**
```bash
ailang run examples/hello.ail
```

**Type-check an example:**
```bash
ailang check examples/option_demo.ail
```

**Interactive exploration:**
```bash
ailang repl
```

## Example Categories

### ✅ Working Examples (Ready to Run)

Start here if you're new to AILANG! These examples work end-to-end:

- **`hello.ail`** - Hello world with builtin functions
- **`simple.ail`** - Let bindings and basic arithmetic
- **`arithmetic.ail`** - Operator precedence and type inference
- **`block_demo.ail`** - Multi-statement blocks with scoping
- **`adt_simple.ail`** - Algebraic data type definitions
- **`adt_option.ail`** - Using the Option ADT
- **`effects_pure.ail`** - Pure function annotations
- **`type_classes_working_reference.ail`** - Type classes and inference

### ⚠️ Type-Check Only (v0.2.0)

These examples demonstrate important features but cannot execute yet due to the module execution limitation:

- **`stdlib_demo.ail`** - Standard library showcase (IO, list, string, option)
- **`option_demo.ail`** - Option ADT with stdlib functions
- **`demos/hello_io.ail`** - IO effects demo

**What works**: Parsing ✓, Type-checking ✓
**What doesn't**: Execution (planned for v0.2.0)

### 📁 Module Examples

The `v3_3/` directory contains examples of the module system:

- Module imports and exports
- Polymorphic functions across modules
- Import conflict resolution
- Nested module structures

### 🔬 Experimental

The `experimental/` directory contains examples of planned features:

- Concurrency with CSP
- Web API integration
- AI agent pipelines
- Advanced algorithms

**Note**: These use syntax that isn't finalized yet.

## Example Status

For a complete status report of all examples, see [STATUS.md](STATUS.md).

**Summary (v0.1.0)**:
- Total: 42 examples
- ✅ Working: 12 (28.6%)
- ⚠️ Type-checks only: 3 (7.1%)
- ❌ Needs fixing: 27 (64.3%)

## Features Demonstrated

### Type System
- Hindley-Milner type inference (`simple.ail`, `arithmetic.ail`)
- Type classes with constraints (`type_classes_working_reference.ail`)
- Algebraic data types (`adt_simple.ail`, `adt_option.ail`)
- Row polymorphism for extensible records
- Effect tracking (`effects_pure.ail`)

### Language Features
- Let bindings and lambdas (`simple.ail`)
- Multi-statement blocks (`block_demo.ail`)
- Pattern matching (type-level only in v0.1.0)
- Module system (`v3_3/*.ail`)
- Pure function annotations

### Standard Library (Type Signatures)
- **io**: print, println, readLine
- **list**: map, filter, fold, length, etc.
- **option**: Some, None, map, getOrElse
- **result**: Ok, Err, map, unwrap
- **string**: length, toUpper, toLower, etc.

## Known Limitations (v0.1.0)

### Module Execution
Currently, AILANG can parse and type-check modules but cannot execute them. This affects:
- Calling standard library functions
- Running module exports
- Demo programs using stdlib

**Workaround**: Use the REPL for interactive testing, or write non-module files (simple expressions).

**Status**: Module execution is planned for v0.2.0.

### Parser Limitations
- Match expressions not yet implemented
- Properties syntax not finalized
- Some experimental syntax not stable

## Contributing Examples

When adding new examples:

1. **Test that they work**:
   ```bash
   ailang run examples/your_example.ail
   ```

2. **Add clear comments** explaining what the example demonstrates

3. **Update STATUS.md** with the new example's status

4. **Use appropriate warnings** for v0.2.0 features:
   ```ailang
   -- ⚠️ NOTE: This example type-checks but cannot execute until v0.2.0
   ```

## Getting Help

- **Documentation**: See `docs/` directory
- **REPL Help**: Type `:help` in the REPL
- **Issue Tracker**: https://github.com/sunholo-data/ailang/issues

## Next Steps

After exploring these examples:

1. Try modifying them to experiment
2. Use the REPL to test ideas interactively
3. Check out the design docs in `design_docs/` to understand the language
4. Watch for v0.2.0 which will add module execution!

---

*For detailed status of each example, see [STATUS.md](STATUS.md)*
