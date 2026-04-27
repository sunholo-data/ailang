---
name: use-ailang
description: Write and run AILANG code with correct syntax. Use when user asks to write AILANG programs, fix AILANG syntax errors, run AILANG code, or needs help with AILANG syntax. Includes version checking, syntax validation, and common patterns.
---

# AILANG Code Writing

Write correct AILANG code using the live MCP server as the canonical reference.

## Quick Start

```bash
ailang run --caps IO --entry main solution.ail   # Run with effects
ailang repl                                       # Interactive development
ailang check solution.ail                         # Type-check only
ailang verify solution.ail                        # Prove contracts with Z3
```

## When to Use This Skill

- User asks to write AILANG code, fix syntax errors, or run programs
- User asks about AILANG features, capabilities, or version
- User wants to add Z3 verification or contracts

## Source of Truth: the AILANG MCP Server

**`mcp.ailang.sunholo.com`** is the canonical, live, version-locked source of truth for AILANG syntax, stdlib, examples, and limitations. Prefer it over any embedded reference in this file.

**If the user's harness supports remote MCP** (Claude Desktop, Cursor, Cline, Continue, Claude Code), have them add it once:

```json
{
  "mcpServers": {
    "ailang-docs": {
      "url": "https://mcp.ailang.sunholo.com/mcp/",
      "transport": "streamable-http"
    }
  }
}
```

Or use the one-click button at [ailang.sunholo.com](https://ailang.sunholo.com/).

Then the agent has typed tools to query AILANG knowledge directly:

| Tool | When to call |
|---|---|
| `prompt_get(for_version, kind="agent")` | Need full syntax reference (replaces reading `resources/syntax_quick_ref.md`) |
| `stdlib_modules(for_version)` | Discover what's available in stdlib |
| `stdlib_module(name, for_version)` | Look up exports of a specific module before importing |
| `stdlib_search(query, for_version)` | Find a function by name/keyword |
| `examples_for_concept(concept, for_version)` | Find a working example of a feature |
| `example_get(path, for_version)` | Read full source of a specific example |
| `limitations_list(for_version)` | Check what AILANG can't do (Y-combinator etc.) before proposing it |
| `effects_catalog(for_version)` | Find which capability flag a function needs |
| `submit_feedback(...)` | File a bug/limitation report mid-session — lands in our triage queue |

Pass `for_version=$(ailang prompt --version-active)` (or empty for "latest") so the response is guaranteed to match the user's CLI capabilities.

**If the harness does NOT support remote MCP** — fall back to the local CLI:

```bash
ailang prompt                            # Full teaching prompt (embedded)
ailang prompt --source mcp               # Force fresh copy from MCP if reachable
ailang prompt --source embedded          # Force embedded copy (eval reproducibility)
ailang docs std/list                     # Inspect a stdlib module locally
ailang examples search "pattern matching"
ailang mcp status                        # Check MCP reachability + drift
```

`ailang prompt --source auto` (the default) prefers MCP-served content when reachable AND its `served_for` matches the CLI version, otherwise silently falls back to embedded. Reproducible eval runs MUST set `--source embedded`.

## Local CLI Reference (always works, even offline)

### Core commands

```bash
ailang repl                                       # Interactive
ailang run --caps IO,FS --entry main file.ail     # Flags BEFORE filename!
ailang check file.ail                             # Type-check only
ailang verify file.ail                            # Z3 contract verification
ailang watch file.ail                             # Auto-reload on change
ailang iface mymodule                             # Module interface as JSON
```

**Critical**: `ailang run` flags MUST come before the filename. `ailang run file.ail --caps IO` silently ignores the flags.

### Examples (already on disk)

```bash
ailang examples search "pattern matching"
ailang examples list --tags adt
ailang examples show adt_option --run
```

## Three Module Conventions (memorize these — they bite)

### 1. REPL — no module declaration

```bash
ailang repl
```

### 2. Module file — standard

```ailang
module myapp/main
import std/io (println)
export func main() -> () ! {IO} = println("Hello, AILANG!")
```

Module path uses **underscores, not hyphens** (`mcp_tools/lib`, not `mcp-tools/lib`).

### 3. Eval-harness benchmark — must use `module benchmark/solution`

```ailang
module benchmark/solution
import std/io (println)
export func main() -> () ! {IO} = println("Benchmark solution")
```

## Workflow

When the user asks for AILANG code:

1. **Check that the agent has MCP access** (if applicable): `ailang mcp status` or look for `mcp.ailang.sunholo.com` in the agent's MCP config.
2. **Get current syntax reference**: call `prompt_get(for_version=...)` via MCP, or `ailang prompt` locally.
3. **Discover stdlib + examples** before writing: call `stdlib_search` / `examples_for_concept` instead of guessing imports.
4. **Write the code** following the prompt's syntax. Prefer pure functions (`! {}`) for core logic — these are Z3-verifiable; keep effectful code (`! {IO}`) as thin wrappers.
5. **Add contracts** to pure functions when verification matters (`requires`/`ensures`). Then `ailang verify solution.ail`.
6. **Type-check before running**: `ailang check solution.ail`.
7. **Run with the right caps**: `ailang run --caps IO,FS --entry main solution.ail`.

## Available Scripts

### `scripts/check_version.sh`
Print the active AILANG prompt version (uses `prompts/versions.json`).

```bash
.claude/skills/use-ailang/scripts/check_version.sh
```

### `scripts/validate_code.sh <file.ail>`
Type-check an AILANG file via `ailang check`.

```bash
.claude/skills/use-ailang/scripts/validate_code.sh solution.ail
```

## Migration Note (M-AGENT-MCP)

This skill used to embed extensive syntax reference (`resources/syntax_quick_ref.md`, `common_patterns.md`, `z3_verification_patterns.md`). Those files are still on disk for offline use, but **prefer the MCP server** — it's version-locked to your CLI and updates without skill edits. See [docs/guides/agent-mcp.md](https://ailang.sunholo.com/docs/guides/agent-mcp) for the full tool catalog.

## Getting Help

- **AILANG MCP**: https://mcp.ailang.sunholo.com/mcp/ (the canonical reference)
- **Documentation**: https://ailang.sunholo.com/
- **Examples**: `ailang examples search ...` or [examples/](https://github.com/sunholo-data/ailang/tree/dev/examples) directory
- **Issues**: https://github.com/sunholo-data/ailang/issues — or call `submit_feedback` from inside an MCP-connected session
- **REPL**: type `:help` in `ailang repl`
