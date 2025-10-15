# Module Execution Guide

**AILANG v0.2.0** - Complete guide to executing modules with entrypoint functions

---

## Overview

AILANG v0.2.0 introduces the **Module Execution Runtime** (M-R1), enabling you to run modules with exported entrypoint functions. This guide covers basic usage, requirements, and common patterns.

## Quick Start

### Basic Usage

```bash
# Run a module with a main() function
ailang --entry main run examples/hello.ail

# Run with a different entrypoint
ailang --entry greet run examples/demo.ail

# Pass arguments via JSON
ailang --entry process --args-json '{"input": "data"}' run examples/processor.ail
```

### Minimal Example

```typescript
-- hello.ail
module examples/hello

export func main() -> () {
    _io_println("Hello from AILANG!")
}
```

Run with:
```bash
ailang --entry main run examples/hello.ail
# Output: Hello from AILANG!
```

---

## Module Structure

### Anatomy of an Executable Module

```typescript
-- my_module.ail
module examples/my_module
import stdlib/std/option (Some, None)

-- Private helper (not exported)
func helper(x: int) -> int {
    x * 2
}

-- Exported entrypoint (0 arguments)
export func main() -> int {
    helper(21)
}

-- Exported entrypoint (1 argument)
export func process(input: {value: int}) -> int {
    helper(input.value)
}
```

### Module Declaration

- **Required**: Every executable module must have a `module` declaration
- **Path matching**: Module path must match file path
  - File: `examples/demo.ail` → Module: `module examples/demo`
  - File: `src/utils/math.ail` → Module: `module src/utils/math`

---

## Entrypoint Functions

### Requirements

An entrypoint function must:
1. ✅ Be **exported** from the module (`export func`)
2. ✅ Be a **function** (not a value)
3. ✅ Have **0 or 1 parameters** (v0.2.0 limitation)
4. ✅ Be specified via `--entry <name>` flag

### Supported Arities

#### 0-Argument Functions

```typescript
export func main() -> int {
    42
}
```

Run with:
```bash
ailang --entry main run module.ail
# Output: 42
```

#### 1-Argument Functions

```typescript
export func greet(name: string) -> () {
    _io_println(name)
}
```

Run with:
```bash
ailang --entry greet --args-json '"World"' run module.ail
# Output: World
```

#### Record Parameters (Recommended Pattern)

```typescript
export func process(params: {input: string, count: int}) -> () {
    _io_println(params.input)
}
```

Run with:
```bash
ailang --entry process --args-json '{"input": "data", "count": 5}' run module.ail
# Output: data
```

### Multi-Argument Workaround

Functions with 2+ parameters are not directly supported. Wrap parameters in a record:

❌ **Not supported:**
```typescript
export func add(x: int, y: int) -> int {
    x + y
}
```

✅ **Supported pattern:**
```typescript
export func add(params: {x: int, y: int}) -> int {
    params.x + params.y
}
```

Run with:
```bash
ailang --entry add --args-json '{"x": 10, "y": 32}' run module.ail
# Output: 42
```

---

## stdlib Functions

### IO Builtins

AILANG v0.2.0 provides three builtin IO functions:

#### `_io_print(s: string) -> ()`

Print a string without a newline.

```typescript
export func main() -> () {
    _io_print("Hello")
    _io_print(" ")
    _io_print("World")
}
```

Output: `Hello World`

#### `_io_println(s: string) -> ()`

Print a string with a newline.

```typescript
export func main() -> () {
    _io_println("Line 1")
    _io_println("Line 2")
}
```

Output:
```
Line 1
Line 2
```

#### `_io_readLine() -> string`

Read a line from stdin (blocking).

```typescript
export func main() -> () {
    _io_println("Enter your name:")
    let name = _io_readLine() in
    _io_println(name)
}
```

---

## Return Values and Output

### Printing Results

- **Non-Unit values**: Printed to stdout automatically
- **Unit values**: Silent (no output)

```typescript
-- Returns int, prints to stdout
export func compute() -> int {
    42
}

-- Returns unit, no output
export func greet() -> () {
    _io_println("Hello")
}
```

### Exit Codes

- **Success**: Exit code 0
- **Runtime error**: Exit code 1
- **Type error**: Exit code 1
- **Parse error**: Exit code 1

---

## Effects and Type Checking

### Effect Annotations (v0.2.0)

Effects are **type-checked** but **not enforced** at runtime in v0.2.0.

```typescript
-- Effect annotation required for IO operations
export func main() -> () ! {IO} {
    _io_println("Hello")
}
```

**Note**: Runtime effect enforcement (capability checks) is planned for v0.3.0 (M-R2).

### Pure Functions

Pure functions have no effect annotation:

```typescript
export pure func add(x: int, y: int) -> int {
    x + y
}
```

---

## Common Patterns

### Simple Script

```typescript
module scripts/hello

export func main() -> () {
    _io_println("Hello from AILANG!")
}
```

### CLI Tool with Arguments

```typescript
module tools/greeter

export func greet(config: {name: string, greeting: string}) -> () {
    _io_print(config.greeting)
    _io_print(" ")
    _io_println(config.name)
}
```

Usage:
```bash
ailang --entry greet --args-json '{"name":"Alice","greeting":"Hello"}' run tools/greeter.ail
```

### Interactive Program

```typescript
module apps/echo

export func main() -> () {
    _io_println("Enter text:")
    let input = _io_readLine() in
    _io_println(input)
}
```

---

## Error Handling

### Common Errors

#### Entrypoint Not Found

```
Error: entrypoint 'main' not found in module examples/demo
  Available exports: greet, process
```

**Solution**: Use `--entry <name>` with an exported function name.

#### Wrong Arity

```
Error: entrypoint 'process' takes 2 parameters. v0.2.0 supports 0 or 1.
  Suggestion: wrap as 'wrapper(p:{...}) -> ...' and pass --args-json
```

**Solution**: Wrap parameters in a record type.

#### Not a Function

```
Error: entrypoint 'config' is not a function (got RecordValue)
```

**Solution**: Only functions can be entrypoints. Values cannot be executed.

#### Module Path Mismatch

```
Error: module declaration 'hello' doesn't match canonical path 'examples/hello'
Suggestions:
  1. Rename module to: module examples/hello
  2. Move file to: hello.ail
```

**Solution**: Ensure module path matches file path.

---

## Known Limitations (v0.2.0)

### Supported ✅
- 0-argument and 1-argument entrypoints
- Builtin IO functions (`_io_print`, `_io_println`, `_io_readLine`)
- JSON argument parsing
- Module imports and dependency resolution
- Type checking with effects
- Pure functions

### Not Yet Supported ⏳
- **Multi-argument functions** (2+ parameters)
  - Workaround: Use record parameter
- **Effect enforcement** at runtime
  - Effects are type-checked only
  - Runtime capability checks coming in M-R2 (v0.3.0)
- **Pattern matching guards**
  - Planned for M-P4
- **Multi-statement function bodies**
  - Parser limitation, planned for future release

---

## Advanced Topics

### Module Dependencies

Modules can import other modules:

```typescript
-- math/utils.ail
module math/utils

export func double(x: int) -> int {
    x * 2
}
```

```typescript
-- app/main.ail
module app/main
import math/utils (double)

export func main() -> () {
    let result = double(21) in
    _io_println(show(result))
}
```

### Encapsulation

Only **exported** bindings are accessible from other modules:

```typescript
module lib/secret

-- Private (not accessible from imports)
func private_helper() -> int {
    42
}

-- Public (accessible via import)
export func public_api() -> int {
    private_helper()
}
```

---

## CLI Reference

### Flags

- `--entry <name>`: Specify entrypoint function (required for modules)
- `--args-json <json>`: Pass arguments as JSON (for 1-arg functions)
- `--runner <mode>`: Choose execution runner (`module` or `fallback`)
- `--no-print`: Suppress output (exit code only)

### Examples

```bash
# Basic execution
ailang --entry main run app.ail

# With arguments
ailang --entry process --args-json '{"data": [1,2,3]}' run app.ail

# Use fallback runner (pre-M-R1 execution)
ailang --runner fallback run app.ail

# Exit code only (no output)
ailang --entry main --no-print run app.ail
echo $?  # Check exit code
```

---

## Troubleshooting

### Module won't load

1. Check module path matches file path
2. Ensure all imports exist
3. Verify no circular imports
4. Check for syntax errors: `ailang check module.ail`

### Function won't execute

1. Ensure function is exported: `export func ...`
2. Check arity (0 or 1 parameters only)
3. Verify entrypoint name: `--entry <name>`
4. Check type errors: `ailang check module.ail`

### No output

- Functions returning `()` (Unit) produce no output
- Use `_io_println()` for explicit output
- Check stderr for errors: `ailang ... 2>&1 | grep Error`

---

## What's Next?

### v0.3.0 (M-R2: Effect Runtime)
- Runtime effect enforcement
- Capability-based security
- IO and FS capabilities
- Deny-by-default model

### v0.4.0 (Pattern Matching Polish)
- Guards in match expressions
- Exhaustiveness checking
- Decision tree optimization

### Future
- Multi-statement function bodies
- Async/await concurrency
- Quasiquotes (SQL, HTML, regex)
- Training data export

---

## Resources

- **Examples**: `examples/` directory
- **stdlib**: `stdlib/std/io.ail`, `stdlib/std/option.ail`
- **Design docs**: `design_docs/20251002/m_r1_module_execution.md`
- **CHANGELOG**: Track new features and breaking changes

---

**Version**: v0.2.0-rc1
**Last Updated**: October 2, 2025
**Status**: Complete
