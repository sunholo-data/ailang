# AILANG v3.2 REPL Commands Demo

## New AI-First Features

AILANG v3.2 introduces several commands designed for AI-assisted development. These features provide structured JSON output for easy consumption by AI agents.

### 1. Effects Inspector (`:effects`)

Inspect the type and effects of an expression without evaluating it:

```ailang
λ> :effects 1 + 2
{
  "effects": [],
  "schema": "ailang.effects/v1",
  "type": "<type inference pending>"
}

λ> :effects \x. x * 2
{
  "effects": [],
  "schema": "ailang.effects/v1",
  "type": "<type inference pending>"
}
```

### 2. Test Reporter (`:test --json`)

Run tests and get structured JSON output:

```ailang
λ> :test --json
{
  "cases": [],
  "counts": {
    "errored": 0,
    "failed": 0,
    "passed": 0,
    "skipped": 0,
    "total": 0
  },
  "duration_ms": 0,
  "platform": {
    "arch": "amd64",
    "go_version": "go1.19.2",
    "os": "darwin",
    "timestamp": "2025-09-28T17:16:41Z"
  },
  "run_id": "5a71641df5b487b0",
  "schema": "ailang.test/v1"
}
```

### 3. Compact Mode (`:compact`)

Toggle between pretty and compact JSON output for token efficiency:

```ailang
λ> :compact on
Compact JSON mode on

λ> :effects 5 + 5
{"effects":[],"schema":"ailang.effects/v1","type":"<type inference pending>"}

λ> :compact off
Compact JSON mode off

λ> :effects 5 + 5
{
  "effects": [],
  "schema": "ailang.effects/v1",
  "type": "<type inference pending>"
}
```

## Error Reporting (Coming Soon)

When fully integrated, errors will be reported in structured JSON:

```json
{
  "schema": "ailang.error/v1",
  "sid": "N#42",
  "phase": "typecheck",
  "code": "TC001",
  "message": "Type mismatch: expected Int, got String",
  "fix": {
    "suggestion": "Convert string to int using parseInt",
    "confidence": 0.85
  },
  "source_span": "example.ail:10:5",
  "context": {
    "constraints": ["Num a", "a ~ String"],
    "decisions": ["failed to unify Int with String"]
  }
}
```

## Decision Ledger (Planned)

Track type inference decisions for context stability:

```ailang
λ> :why
Last 3 decisions:
1. [N#51] Defaulted Num α → Int at line 5
2. [N#52] Unified α → β at line 6  
3. [N#53] Resolved instance Eq[Int] at line 7
```

## Usage Tips

1. **For AI Agents**: Use `:compact on` to reduce token usage
2. **For Debugging**: Use `:effects` to understand types before evaluation
3. **For Testing**: Use `:test --json` to get machine-readable test results
4. **For Learning**: Keep `:trace-defaulting on` to see type inference decisions

## Complete Command List

Run `:help` in the REPL to see all available commands:

```
λ> :help
REPL Commands:
  :help, :h                Show this help
  :quit, :q                Exit the REPL
  :type <expr>             Show type of expression
  :effects <expr>          Show type and effects without evaluating
  :import <module>         Load module instances
  :dump-core              Toggle Core AST display
  :dump-typed             Toggle Typed AST display
  :dry-link               Show required instances without evaluating
  :trace-defaulting on|off Enable/disable defaulting trace
  :instances              Show available type class instances
  :test [--json]          Run tests (with optional JSON output)
  :compact on|off         Enable/disable compact JSON mode
  :history                Show command history
  :clear                  Clear the screen
  :reset                  Reset the environment
```