# CLI Documentation Best Practices

Guidelines for maintaining AILANG CLI help as the authoritative source of truth.

## Formatting Conventions

### Command Entry Format

```go
fmt.Printf("  %s <args>        Brief description (one line)\n", cyan("command"))
```

**Rules:**
- Command in `cyan()` color
- Args in angle brackets: `<required>`, `[optional]`
- Description starts after consistent spacing (~20 chars)
- Keep description under 60 characters
- End with newline

### Category Headers

```go
fmt.Println("Category Name:")
// Commands here
fmt.Println()  // Blank line after category
```

**Rules:**
- Title case for category names
- No formatting (plain text, not colored)
- Followed immediately by commands
- Blank line after category section

### Examples Format

```go
fmt.Printf("  %s    # Comment\n", cyan("ailang command --flag value"))
```

**Rules:**
- Example command in `cyan()`
- Inline comment starts with `#`
- Group related examples together
- Show realistic, working examples
- Include flags when relevant

## Category Organization

### Current Categories (in order)

1. **Commands** - Core execution (run, repl, check, test, watch)
2. **Evaluation & Benchmarking** - AI benchmarks
3. **Development Tools** - Debugging, docs, builtins
4. **Messages** - Collaboration system
5. **Web & API** - Server and API tools
6. **Observatory & Collaboration** - Dashboard
7. **Agent Coordination** - Coordinator, tasks
8. **Telemetry & Debugging** - Traces, metrics
9. **Advanced Tools** - Low-level tools
10. **Run Command Flags** - Main flags reference
11. **Global Flags** - Universal flags
12. **Environment Variables** - Debug and config vars
13. **Examples** - Practical usage examples

### Adding New Categories

**When to add:**
- 3+ related commands without a clear home
- Distinct functional area (e.g., "Security & Auth")
- Different user persona (e.g., "Admin Tools")

**Where to place:**
- Core commands first (most used)
- Specialized tools later
- Advanced/internal last
- Examples always at end

## Environment Variable Naming

### Patterns

**Debug flags:**
- Format: `DEBUG_<COMPONENT>=1`
- Examples: `DEBUG_PARSER`, `DEBUG_STRICT`, `DEBUG_CODEGEN`
- Boolean only (presence = enabled)

**Configuration:**
- Format: `AILANG_<FEATURE>=<value>`
- Examples: `AILANG_RELAX_MODULES=1`, `AILANG_PARENT_TASK_ID=abc123`
- Can be boolean or value

**External integrations:**
- Use standard names: `GOOGLE_CLOUD_PROJECT`, `OTEL_EXPORTER_OTLP_ENDPOINT`
- Don't prefix with `AILANG_` if external standard exists

### Documentation Format

```go
fmt.Println("Debug Flags (for troubleshooting):")
fmt.Println("  DEBUG_STRICT=1               Fail loudly on unhandled cases (recommended for CI)")
```

**Rules:**
- Variable name left-aligned
- Padding to ~30 chars for alignment
- Purpose description (one line)
- Use case hint in parentheses if helpful

## Example Writing Guidelines

### Structure

Group examples by category:

```go
fmt.Println("Basic Usage:")
// Basic examples here

fmt.Println("Evaluation & Benchmarking:")
// Eval examples here

fmt.Println("Debugging & Telemetry:")
// Debug examples here
```

### What Makes a Good Example

**✅ DO:**
- Show realistic, working commands
- Include necessary flags
- Add inline comments explaining purpose
- Use actual file/path names that make sense
- Demonstrate flag ordering (`ailang run --caps IO file.ail`)

**❌ DON'T:**
- Show commands that don't work
- Use placeholder values (`<file>`, `<value>`)
- Omit critical flags
- Show incorrect flag order

### Example Types

**Basic usage** (every command needs one):
```go
fmt.Printf("  %s              # Run program with IO capability\n", cyan("ailang run --caps IO hello.ail"))
```

**Complex flags** (show proper syntax):
```go
fmt.Printf("  %s  # Run full suite with multiple models\n", cyan("ailang eval-suite --models gpt5,claude-sonnet-4-5"))
```

**Environment variables** (show usage pattern):
```go
fmt.Printf("  %s                 # Debug parser issues\n", cyan("DEBUG_PARSER=1 ailang run test.ail"))
```

**Pipes and composition** (show advanced patterns):
```go
fmt.Printf("  %s         # List recent compiler traces\n", cyan("ailang trace list --filter compile --hours 2"))
```

## File Size Targets

**Main help.go:**
- Target: ≤250 lines
- Max acceptable: 300 lines
- Current: ~220 lines ✅

**Why this matters:**
- AIs need to scan help quickly
- Users should see full help on one screen (with scrollback)
- Detailed docs belong in per-command help

**If exceeding target:**
1. Move details to per-command help (e.g., `coordinator --help`)
2. Condense examples (keep most useful ones)
3. Remove redundant descriptions
4. Consider splitting categories

## Per-Command Help

Commands with many subcommands should have dedicated help:

**Example: coordinator**
```go
case "coordinator":
    if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
        printCoordinatorHelp()
        return nil
    }
    // ... handle subcommands
```

**Pattern:**
- Function: `printCoordinatorHelp()`
- Lists all subcommands with descriptions
- Shows common flags
- Includes examples
- No line limit (can be detailed)

## Validation Checklist

Before committing changes to help.go:

- [ ] All commands exist in main.go
- [ ] All env vars are used in codebase
- [ ] No stale/removed commands documented
- [ ] Examples actually work (tested)
- [ ] Formatting is consistent
- [ ] File under 250 lines (target)
- [ ] Run `make quick-install && ailang --help`
- [ ] Verify examples visually in output

## Common Mistakes to Avoid

**❌ Documenting before implementing:**
```go
fmt.Printf("  %s          Compile to WebAssembly\n", cyan("wasm"))
// But "wasm" doesn't exist in main.go yet!
```

**❌ Wrong flag order in examples:**
```go
fmt.Printf("  %s\n", cyan("ailang run test.ail --caps IO"))
// Flags must come BEFORE filename!
```

**❌ Non-working examples:**
```go
fmt.Printf("  %s\n", cyan("ailang eval --model gpt5"))
// But --model doesn't exist, should be --models
```

**❌ Placeholder values:**
```go
fmt.Printf("  %s\n", cyan("ailang run <file>"))
// Use a real example: "ailang run hello.ail"
```

**❌ Missing critical flags:**
```go
fmt.Printf("  %s\n", cyan("ailang run hello.ail"))
// Need --caps flag for IO operations!
```

## Design Philosophy

1. **CLI as Source of Truth** - Help text IS the documentation
2. **AI-First** - Optimize for AI discoverability (they can't read CLAUDE.md in external repos)
3. **Accurate** - Zero tolerance for stale documentation
4. **Practical** - Every command needs a working example
5. **Scannable** - Users should grasp structure in 30 seconds
6. **Progressive** - Overview in main help, details in per-command help

## Update Frequency

**When to update:**
- ✅ Every time a command is added
- ✅ Every time flags change
- ✅ Every time env vars are added
- ✅ Quarterly review for accuracy
- ✅ After major version releases

**How to stay synchronized:**
- Run audit scripts before every release
- Include help.go in code review checklists
- Test examples manually (or automate)
- Keep CHANGELOG.md in sync with CLI changes
