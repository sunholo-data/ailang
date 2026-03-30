# M-DX-AI-DISCOVERY: Improve AI Agent Stdlib Discovery

**Status**: Planned
**Target**: v0.10.1
**Priority**: P1
**Estimated**: 2 days
**Dependencies**: None
**Bug Report**: Discovered via benchmark analysis (M-EVAL-XLANG sprint)

## Problem Statement

When AI agents (Claude Code) write AILANG programs from scratch, 60-70% of their extra cost vs known languages comes from **stdlib discovery overhead**. Analysis of 12+ benchmark trials shows agents spend 8-17 turns:

1. Running `ailang docs std/string`, `ailang docs std/list` etc. to find function names
2. Writing test files to `/tmp` to verify syntax works
3. Getting MOD010 errors from incorrect module paths
4. Searching for `std/clock` (timestamp/hash builtins) repeatedly — 17 turns in worst case
5. Not knowing that `show()` and `println()` are prelude (no import needed)

### Benchmark Evidence

| Metric | AILANG (claude-md) | Ruby | Haskell |
|--------|-------------------|------|---------|
| v1 Turns | 35 avg | ~12 | ~20 |
| v1 Cost | $0.81 avg | $0.18 | $0.37 |
| Discovery turns | ~17 (49%) | 0 | ~3 |

The v1→v2 cost drop ($0.81→$0.61) proves that once the agent knows the stdlib, subsequent work is efficient. The problem is the cold-start discovery phase.

## Goals

1. **Reduce stdlib discovery turns by 50%** (from ~17 to ~8)
2. **Make `ailang docs` output AI-parseable** in a single command
3. **Add stdlib function search** by description, not just module name
4. **Improve error messages** for common AI mistakes (wrong module paths, missing imports)

## Solution Design

### 1. `ailang docs --all-functions` command

Single command that dumps ALL stdlib function signatures in a compact, parseable format:

```
$ ailang docs --all-functions
std/string.length: string -> int -- String length
std/string.contains: (string, string) -> bool -- Check substring
std/string.split: (string, string) -> [string] -- Split by delimiter
std/list.map: (a -> b, [a]) -> [b] -- Apply function to each element
std/list.foldl: ((b, a) -> b, b, [a]) -> b -- Left fold
std/fs.readFile: string -> string ! {FS} -- Read file contents
std/fs.writeFile: (string, string) -> () ! {FS} -- Write file
...
```

This gives an AI agent the entire stdlib in one turn instead of 6+ separate `ailang docs std/X` calls.

### 2. `ailang docs search <query>` command

Search stdlib functions by keyword/description:

```
$ ailang docs search "timestamp"
std/clock.now: () -> int ! {Clock} -- Unix timestamp in seconds
std/clock.nowMillis: () -> int ! {Clock} -- Unix timestamp in milliseconds

$ ailang docs search "hash"
std/string.charCode: string -> int -- Get Unicode code point (useful for hashing)
prelude.show: a -> string -- Convert any value to string
```

### 3. Better error messages for common AI mistakes

Current:
```
MOD010: Unknown module "std/time"
```

Proposed:
```
MOD010: Unknown module "std/time"
  Did you mean: std/clock?
  Available std/ modules: string, list, io, fs, env, math, clock, json, net, process, option, result
```

### 4. Prelude documentation in `ailang docs`

Add a `ailang docs prelude` command showing auto-imported functions:
```
$ ailang docs prelude
Auto-imported (no import needed):
  println: string -> () ! {IO}     -- Print with newline
  show: a -> string                -- Convert any value to string
  ==, !=, <, >, <=, >=            -- Comparison operators
```

## Implementation Plan

### M1: `--all-functions` flag (~100 LOC)
- Add flag to `cmd/ailang/docs.go`
- Iterate all registered builtins, format as `module.name: type -- description`
- Include prelude functions
- Test: verify output contains all 52+ builtins

### M2: `search` subcommand (~150 LOC)
- Add keyword search over builtin metadata (name, description, tags)
- Fuzzy match on description text
- Test: `ailang docs search "timestamp"` finds `std/clock.now`

### M3: Module suggestion in MOD010 errors (~80 LOC)
- In module resolver, on unknown module, compute edit distance to known modules
- Suggest closest match if distance <= 3
- List all `std/` modules in error message
- Test: `import std/time` suggests `std/clock`

### M4: `prelude` subcommand (~50 LOC)
- List all prelude-imported names with types
- Test: output includes `println`, `show`

## Success Criteria

- [ ] `ailang docs --all-functions` lists all stdlib functions in one command
- [ ] `ailang docs search "timestamp"` finds relevant functions
- [ ] MOD010 errors suggest nearest valid module
- [ ] `ailang docs prelude` shows auto-imported functions
- [ ] All tests passing
- [ ] Re-benchmark shows fewer discovery turns

## Timeline

- Day 1: M1 + M2 (~250 LOC)
- Day 2: M3 + M4 + testing (~130 LOC)
- Total: ~380 LOC implementation + ~200 LOC tests
