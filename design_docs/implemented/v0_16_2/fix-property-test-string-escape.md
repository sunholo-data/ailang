---
title: Fix: Property-test codegen — backslash escaping in string literals
status: Implemented
version: v0.16.2
date: 2026-05-07
priority: P1
---

## Problem

The property-test runner generates AILANG source code in memory to bind
random values into `forall` expressions before evaluation. The code
generation path was:

1. `runner.valueToLiteral` converts an `eval.StringValue` →
   `ast.Literal{Kind: StringLit, Value: rawString}`
2. `runner.bindPropertyValues` wraps it in `ast.Let` nodes
3. `executor.EvaluateExpression` calls `fmt.Sprintf("%v", expr)` to
   render the `ast.Let` tree back to AILANG source text
4. `ast.Let.String()` calls `l.Value.String()` on the nested `ast.Literal`
5. `ast.Literal.String()` was: `return fmt.Sprintf("%v", l.Value)` —
   for a `string` value this returns the raw bytes with **no quoting and
   no escaping**

Result: any random `string` containing a backslash or double-quote made
the generated source unparseable. For example, `Message.content = "a\nb"`
produced the source fragment `(let x = a\nb in …)` instead of the valid
AILANG literal `"a\\nb"`.

The bug was latent since property testing shipped. It only surfaces when
the property has a `string` parameter and the generator produces a value
containing `\` or `"`.

## Root Cause

`ast.Literal.String()` returned `fmt.Sprintf("%v", l.Value)` for all
literal kinds. For `StringLit`, `l.Value` is a Go `string`, and `%v`
emits the raw string without quoting or escaping — not a valid AILANG
source token.

## Fix

Updated `ast.Literal.String()` in `internal/ast/ast_expr.go` to
special-case `StringLit`:

```go
func (l *Literal) String() string {
    if l.Kind == StringLit {
        if s, ok := l.Value.(string); ok {
            escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
            return `"` + escaped + `"`
        }
    }
    return fmt.Sprintf("%v", l.Value)
}
```

`strings.NewReplacer` applies substitutions left-to-right in a single
pass, avoiding double-escaping (`\` → `\\` before `"` → `\"`).

## Scope

Only `Literal.String()` for `StringLit` changed. Other literal kinds
(`IntLit`, `FloatLit`, `BoolLit`, `UnitLit`) already emit valid AILANG
source via `%v` and are unchanged.

No callers of `Literal.String()` outside the testing code path relied on
the unquoted behaviour — confirmed by grep.

## Regression Test

Added `TestLiteralString_StringLit` in `internal/ast/print_test.go`:

```go
cases := []struct{ raw, want string }{
    {"hello",     `"hello"`},
    {`back\slash`, `"back\\slash"`},
    {`say "hi"`,  `"say \"hi\""`},
    {`a\b\"c`,    `"a\\b\\\"c"`},
    {"",           `""`},
}
```

## Related

- Bug originally filed as `fb_ecd599e79d08bf7c` via
  `mcp__ailang-docs__submit_feedback` (does not appear in the ailang
  messages inbox — feedback MCP endpoint is separate from the agent
  message bus)
- Mentioned in motoko-explore message `af5857d0` as
  `elide_old_tool_results_property_1`
