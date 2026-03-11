---
paths:
  - "internal/parser/**"
  - "internal/lexer/**"
  - "internal/ast/**"
---

# Parser & Lexer Rules

## NEWLINE Tokens Don't Exist

The lexer NEVER generates NEWLINE tokens — `skipWhitespace()` consumes `\n`. Never check for `lexer.NEWLINE` in parser code. Multi-line syntax "just works" because whitespace is already skipped.

## Critical Conventions

1. Parser leaves cursor AT last token (not after)
2. Use `DEBUG_PARSER=1` for token position tracing
3. Use `make doc PKG=<package>` for API discovery

## Common Gotchas

- IntLit is `int64`, not `int` (will panic!)
- Never check for `lexer.NEWLINE`
- Print errors BEFORE `t.Fatalf` in tests
- `string(rune(i))` produces unprintable chars (use `fmt.Sprintf`)

## Constructors

- `parser.New(lexer)` — Takes lexer instance
- `elaborate.NewElaborator()` — No arguments
- `types.NewTypeChecker(core, imports)` — Takes Core prog + imports
- `link.NewLinker()` — No arguments

## Adding a New Language Feature

1. Token definitions: `internal/lexer/token.go`
2. Lexer: `internal/lexer/lexer.go`
3. AST nodes: `internal/ast/ast.go`
4. Parser: `internal/parser/parser.go`
5. Type rules: `internal/types/`
6. Evaluation: `internal/eval/`
7. Tests, examples, CHANGELOG, README (all REQUIRED)

Use the `parser-developer` skill for guided development.
