# AILANG Examples

This directory contains example programs demonstrating AILANG features.

## Status Levels

### ✅ **Working Examples** (Ready for documentation)
These examples work with the current implementation and can be run with `ailang run <file>`:

- **[hello.ail](hello.ail)** - Basic print statement
- **[simple.ail](simple.ail)** - Let bindings and arithmetic
- **[arithmetic.ail](arithmetic.ail)** - Show type class usage
- **[lambda_expressions.ail](lambda_expressions.ail)** - Comprehensive lambda examples (currying, closures, higher-order functions)
- **[adt_simple.ail](adt_simple.ail)** - ADT declarations and pattern matching
- **[adt_option.ail](adt_option.ail)** - Option type with pattern matching
- **[patterns.ail](patterns.ail)** - All pattern types (tuples, literals, constructors, nested)
- **[typeclasses.ail](typeclasses.ail)** - Type class usage (Num, Eq, Ord, Show)
- **[records.ail](records.ail)** - Record literals, field access, nesting

### ⚠️ **Experimental Examples** (Future features)
These examples demonstrate planned features and **will not run** with the current implementation. They are included for design validation and documentation of the language vision.

Located in [experimental/](experimental/):

- **[factorial.ail](experimental/factorial.ail)** - Requires: `func` declarations, `tests` syntax
- **[quicksort.ail](experimental/quicksort.ail)** - Requires: `func` declarations, list patterns
- **[concurrent_pipeline.ail](experimental/concurrent_pipeline.ail)** - Requires: CSP (channels, spawn, parallel)
- **[web_api.ail](experimental/web_api.ail)** - Requires: Quasiquotes, HTTP effects
- **[ai_agent_integration.ail](experimental/ai_agent_integration.ail)** - Requires: Full effect system

Each experimental file has a warning header explaining what features are missing.

### 🔬 **Development Examples** (In progress)
Located in [v3_3/](v3_3/):

These examples test features currently under development (module system, imports, exports). Status may change frequently.

## Running Examples

### File Execution
```bash
# Run a working example
ailang run examples/hello.ail
ailang run examples/lambda_expressions.ail
ailang run examples/patterns.ail

# Experimental examples will fail with parser errors
ailang run examples/experimental/factorial.ail  # ❌ Will fail
```

### REPL Usage
```bash
# Start the REPL
ailang repl

# Try expressions from examples
λ> 1 + 2 * 3
7 :: Int

λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> type Option[a] = Some(a) | None
λ> match Some(42) { Some(n) => n, None => 0 }
42 :: Int
```

## Example Coverage by Feature

| Feature | Example Files |
|---------|--------------|
| **Let bindings** | simple.ail, arithmetic.ail |
| **Lambdas** | lambda_expressions.ail |
| **Type classes** | typeclasses.ail, arithmetic.ail |
| **Records** | records.ail, lambda_expressions.ail |
| **ADTs** | adt_simple.ail, adt_option.ail |
| **Pattern matching** | patterns.ail, adt_simple.ail |
| **Effects** | *(M-P4 in progress)* |
| **Modules** | v3_3/* *(in development)* |
| **Concurrency** | experimental/concurrent_pipeline.ail *(future)* |
| **Quasiquotes** | experimental/web_api.ail *(future)* |

## Implementation Status

**Current Version:** v0.0.9 (M-P3 complete, M-P4 in progress)

**Completed Milestones:**
- ✅ M-P1: Core parser foundation
- ✅ M-P2: ADT syntax (sum/product types)
- ✅ M-P3: Pattern matching with ADT runtime
- ⏳ M-P4: Effect system (type-level only)

**Next Milestones:**
- Module system completion (imports/exports)
- Effect system runtime (handlers, capabilities)
- Standard library expansion
- Quasiquotes implementation
- CSP concurrency

See [design_docs/](../design_docs/) for detailed implementation plans.

## Contributing Examples

When adding new examples:

1. **Test first**: Verify the example runs with `ailang run`
2. **Add comments**: Explain what the example demonstrates
3. **Include output**: Show expected results in comments
4. **Mark status**:
   - Working examples go in root `examples/`
   - Future features go in `examples/experimental/` with warning headers
5. **Update this README**: Add to the appropriate status section
6. **Update manifest.json**: Keep the verification manifest in sync

## Reference Files

- **[type_classes_working_reference.ail](type_classes_working_reference.ail)** - Detailed type class implementation reference (not for beginners)

## Getting Help

- **Documentation**: See `CLAUDE.md` for language overview
- **Design Docs**: See `design_docs/` for feature specifications
- **Issues**: Report problems at https://github.com/sunholo/ailang/issues

---

**Last Updated:** 2025-10-01
**Examples Count:** 9 working, 5 experimental, 6 in development
