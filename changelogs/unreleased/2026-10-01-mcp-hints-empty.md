### Fixed — `@mcp_hints()` can declare an additive, closed-world write

- `@mcp_hints` required at least one hint, so a tool that writes additively in a closed world,
  and is not idempotent, could not be declared at all (found annotating docparse's hosted tools).
  The hint list is complete, so the empty `@mcp_hints()` now parses and means exactly that:
  `readOnlyHint: false, destructiveHint: false, idempotentHint: false, openWorldHint: false`.
- The lexer reads `()` as one UNIT token, so the parser accepts it there. `ailang fmt` keeps the
  parens: a bare `@mcp_hints` would not re-parse. A round-trip test catches the regression when
  the fix is reverted.
