---
reviewed: 2026-11-26
reviewBy: 2027-05-26
---

# AILANG REPL Commands Reference

## Starting the REPL

```bash
ailang repl
```

The REPL auto-imports `std/prelude` on startup, providing:
- Numeric defaults: `Num → Int`, `Fractional → Float`
- Type class instances for `Num`, `Fractional`, `Eq`, `Ord`, `Show`
- List/string utilities and `Option`/`Result` constructors
- Record literals and field access

## Basic Commands

- `:help, :h` - Show all available commands
- `:quit, :q, :exit` - Exit the REPL (also works: Ctrl+D)
- `:type <expr>, :t` - Show resolved type
- `:import <module>, :i` - Import module bindings and instances
- `:instances` - List available instances with superclass provisions
- `:history` - Show command history
- `:clear` - Clear the screen
- `:reset` - Reset environment (auto-reimports prelude)

## Debugging Commands

- `:dump-core` - Toggle Core AST display for debugging
- `:dump-typed` - Toggle Typed AST display
- `:dry-link` - Show required dictionary instances without evaluating
- `:strict` - Toggle strict type-checking mode
- `:trace-defaulting on/off` - Enable/disable defaulting trace

## AI-First Commands

- `:effects <expr>` - Inspect type and effects without evaluation
- `:test [--json]` - Run tests with optional JSON output
- `:compact on/off` - Toggle JSON compact mode for token efficiency
- `:propose <spec>` / `:scaffold <file>` - AI proposal and scaffolding helpers

## Interactive Features

- **Arrow Key History**: Navigate command history with ↑/↓ arrows
- **Persistent History**: Commands saved in `~/.ailang_history`
- **Tab Completion**: Auto-complete REPL commands with Tab key
- **Multi-line Input**: Automatic continuation for incomplete expressions

## Example Session

```ailang repl
λ[Debug,IO]> 1 + 2
3 :: Int

λ[Debug,IO]> 3.14 * 2.0
6.28 :: Float

λ[Debug,IO]> "Hello AILANG!"
Hello AILANG! :: String

λ[Debug,IO]> [1, 2, 3]
[1, 2, 3] :: [Int]

λ[Debug,IO]> {name: "Alice", age: 30}
{age: 30, name: "Alice"} :: {age: Int, name: String}

λ[Debug,IO]> :type \x. x + x
\x. x + x :: int -> int

λ[Debug,IO]> let double = \x. x * 2 in double(21)
42 :: Int

λ[Debug,IO]> :instances
Available instances:
  Ord:
    • Ord[Int] (provides Eq[Int])
    • Ord[Float] (provides Eq[Float])
  Fractional:
    • Fractional[Float] (provides Num[Float])
  Eq:
    • Eq[Float]
    • Eq[Int]
  Show:
    • Show[Int]
    • Show[String]
    • Show[Bool]
    • Show[Float]
  Num:
    • Num[Int]
    • Num[Float]

λ[Debug,IO]> :quit               # Exit REPL
Goodbye!
```

## Multi-line Input Support

The REPL supports multi-line expressions with automatic continuation:

```bash
λ[Debug,IO]> let user = {name: "Alice", age: 30} in
... user
{age: 30, name: "Alice"} :: {age: Int, name: String}
```

## Type Class Pipeline

The REPL executes the full pipeline:
1. **Parse** - Surface syntax to AST
2. **Elaborate** - AST to Core (ANF)
3. **TypeCheck** - Infer types with constraints and defaulting
4. **Dictionary Elaboration** - Transform operators to dictionary calls
5. **ANF Verification** - Ensure well-formed Core
6. **Link** - Resolve dictionary references
7. **Evaluate** - Execute with runtime dictionaries
