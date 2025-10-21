# Documentation Snippets

Code examples for documentation and learning. These are **not** meant to be run directly - they're single expressions or code fragments that demonstrate AILANG syntax.

## Purpose

These files show:
- Language syntax and features
- Common patterns and idioms
- Type system capabilities
- Code examples for tutorials

## Format

Most snippets are single expressions without module wrappers:

```ailang
-- Simple arithmetic
2 + 3  -- Returns 5

-- Lambda expression
\x. x * 2  -- Doubling function
```

## How to Use

**Option 1: Copy into REPL**
```bash
ailang repl
> 2 + 3
5
> \x. x * 2
<closure>
```

**Option 2: Wrap in a module to run**
```ailang
module examples/my_example

export func main() -> int {
  2 + 3
}
```

## Contents

### Basic Syntax
- `hello.ail` - Hello world expression
- `arithmetic.ail` - Math operations
- `func_expressions.ail` - Function syntax
- `lambda_expressions.ail` - Lambda examples
- `numeric_conversion.ail` - Type conversions

### Data Structures
- `records.ail` - Record syntax and field access
- `list_patterns.ail` - List pattern matching

### Type System
- `typeclasses.ail` - Type class instances
- `type_classes_working_reference.ail` - Working type class examples
- `showcase/01_type_inference.ail` - Type inference examples
- `showcase/03_type_classes.ail` - Type class showcase

### Advanced Features
- `showcase/02_lambdas.ail` - Lambda functions
- `showcase/03_lists.ail` - List operations
- `showcase/04_closures.ail` - Closures and environments

### Module System
- `v3_3/imports.ail` - Import syntax
- `v3_3/imports_basic.ail` - Basic imports
- `v3_3/math/gcd.ail` - GCD example

### Documentation
- `block_demo.ail` - Block expression syntax
- `option_demo.ail` - Option type usage
- `stdlib_demo.ail` - Standard library overview
- `stdlib_demo_simple.ail` - Simple stdlib examples

## Testing

Snippets are **not** verified by CI (they're not runnable programs). If you want to test them:

1. Open the AILANG REPL: `ailang repl`
2. Copy/paste expressions from the snippet files
3. See the results immediately

## Contributing

When adding new snippets:
- Focus on clarity over complexity
- Add comments explaining what each expression does
- Use realistic examples
- Show expected output in comments
