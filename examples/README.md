# AILANG Examples

Example programs demonstrating AILANG features, organized by purpose and runnability.

## Directory Structure (v0.3.14+)

```
examples/
├── runnable/          # ✅ Full working programs (21 files) - CI verified
├── snippets/          # 📝 Documentation code snippets (21 files)
├── tests/             # 🧪 Test cases and edge cases (37 files)
├── experimental/      # 🔬 Future features (9 files) - Not working yet
└── plans/             # Design plan examples
```

## Quick Start

### Running Examples

```bash
# Run a simple program
ailang run --caps IO examples/runnable/simple.ail --entry main

# Run with multiple capabilities
ailang run --caps IO,FS,Net examples/runnable/effects_basic.ail --entry main

# Run JSON parsing example
ailang run --caps IO examples/runnable/json_basic_decode.ail --entry main
```

### Using Snippets in REPL

```bash
ailang repl
> 2 + 3
5
> let x = 10 in x * 2
20
```

Copy expressions from `snippets/` files and paste into the REPL.

## Categories

### [runnable/](runnable/) - Full Working Programs
**Best for: Learning how to write complete AILANG programs**

- ✅ All files have proper module structure with `export func main()`
- ✅ Verified by CI (`make verify-examples`)
- ✅ Expected to run without errors
- ✅ Demonstrate real-world usage patterns

**Examples**: ADTs, recursion, effects, blocks, JSON, I/O

### [snippets/](snippets/) - Documentation Examples
**Best for: Understanding AILANG syntax and features**

- 📝 Single expressions and code fragments for tutorials
- 📝 Not meant to be run directly (no module wrapper)
- 📝 Can be copied into REPL for experimentation
- 📝 Show syntax and language features clearly

**Examples**: hello world, arithmetic, lambdas, records, type classes

### [tests/](tests/) - Test Cases
**Best for: Understanding how specific features should behave**

- 🧪 Verify language behavior and edge cases
- 🧪 Some intentionally fail (testing error handling)
- 🧪 May require specific capabilities or external resources
- 🧪 Not all are verified by CI

**Examples**: effect system, pattern guards, module imports, builtins

### [experimental/](experimental/) - Future Features
**Best for: Seeing what's coming in future releases**

- ⚠️ **DO NOT WORK** with current version
- 🔬 Show intended future API and syntax
- 🔬 Will become working examples when features ship

**Examples**: AI agent integration, concurrency, HTTP servers

## Example Index

### By Feature

**Basic Language**
- Simple expressions: `runnable/simple.ail`
- Arithmetic: `snippets/arithmetic.ail`
- Functions: `snippets/func_expressions.ail`
- Lambdas: `snippets/lambda_expressions.ail`, `snippets/showcase/02_lambdas.ail`

**Data Types**
- Records: `snippets/records.ail`, `runnable/micro_record_person.ail`
- Lists: `snippets/list_patterns.ail`, `snippets/showcase/03_lists.ail`
- ADTs: `runnable/adt_simple.ail`, `runnable/adt_option.ail`
- Option type: `runnable/micro_option_map.ail`

**Control Flow**
- Pattern matching: `runnable/patterns.ail`
- Guards: `runnable/guards_basic.ail`, `tests/test_guard_bool.ail`
- Blocks: `runnable/micro_block_if.ail`, `runnable/micro_block_seq.ail`

**Recursion**
- Factorial: `runnable/recursion_factorial.ail`
- Fibonacci: `runnable/recursion_fibonacci.ail`
- Mutual recursion: `runnable/recursion_mutual.ail`
- Quicksort: `runnable/recursion_quicksort.ail`

**Effects**
- Basic I/O: `runnable/effects_basic.ail`, `runnable/demos/hello_io.ail`
- Pure vs effectful: `runnable/effects_pure.ail`
- Effect capabilities: `tests/test_effect_capability.ail`

**Type System**
- Type inference: `snippets/showcase/01_type_inference.ail`
- Type classes: `snippets/typeclasses.ail`, `snippets/showcase/03_type_classes.ail`
- Closures: `snippets/showcase/04_closures.ail`

**JSON**
- JSON parsing: `runnable/json_basic_decode.ail`

**Modules**
- Minimal module: `tests/test_module_minimal.ail`
- Imports: `snippets/v3_3/imports.ail`, `tests/test_import_func.ail`

## CI Verification

Only `runnable/` examples are verified by continuous integration:

```bash
make verify-examples  # Runs all runnable examples
```

Expected pass rate: **95-100%** (21 examples, all should work)

## Statistics (v0.3.14)

| Category | Count | CI Verified | Expected Pass Rate |
|----------|-------|-------------|-------------------|
| Runnable | 21 | ✅ Yes | 95-100% |
| Snippets | 21 | ❌ No | N/A (not runnable) |
| Tests | 37 | ❌ No | ~80% (some test errors) |
| Experimental | 9 | ❌ No | 0% (future features) |
| **Total** | **88** | **21** | **24%** |

## Contributing Examples

### Adding Runnable Examples

1. Create file in `examples/runnable/`
2. Use proper module structure:
```ailang
module examples/my_example

export func main() ! {IO} -> string {
  _io_println("Hello!");
  "ok"
}
```
3. Test it works: `ailang run --caps IO examples/runnable/my_example.ail --entry main`
4. CI will automatically verify it

### Adding Snippets

1. Create file in `examples/snippets/`
2. Use simple expressions (no module wrapper needed):
```ailang
-- Demonstrates feature X
let x = 10 in
x * 2  -- Returns 20
```
3. Add comments showing expected output
4. Test in REPL before committing

### Adding Tests

1. Create file in `examples/tests/` with `test_` prefix
2. Document expected behavior (pass/fail, error message)
3. Test manually with required capabilities

## Migration from Old Structure (v0.3.14)

**Before**:
- All examples in flat `examples/` directory
- CI verified everything (high failure rate)
- Unclear which examples should work

**After** (v0.3.14+):
- Organized by purpose and runnability
- CI only verifies `runnable/` examples
- Clear expectations for each category

This improves:
- **Clarity**: Users know which examples should work
- **CI reliability**: Only test runnable programs
- **Documentation**: Snippets are clearly not full programs

## See Also

- [AILANG Documentation](../docs/) - Full language reference
- [CHANGELOG.md](../CHANGELOG.md) - Version history
- [CONTRIBUTING.md](../docs/CONTRIBUTING.md) - Development guide
- Each subdirectory has its own README with details
