# M-DX-EXAMPLES-COVERAGE: Stdlib Examples for Every Module

**Status**: Planned
**Target**: v0.10.1
**Priority**: P2
**Estimated**: 1 day
**Dependencies**: None
**Bug Report**: Discovered via benchmark analysis (M-EVAL-XLANG sprint)

## Problem Statement

During benchmark trials, AI agents write throwaway test files to `/tmp` to verify AILANG syntax before using it in the actual solution. This wastes 3-5 turns per trial. The root cause is that agents don't trust the documentation alone — they want to see **working examples** before committing to a pattern.

Currently AILANG has examples in `examples/runnable/` but they don't cover all stdlib modules systematically. An agent looking for "how to use std/clock" finds nothing in examples and resorts to trial-and-error.

### Benchmark Evidence

From claude-md trial 1 session analysis:
- 4 turns writing test files to `/tmp/test_*.ail`
- 2 turns running them to verify syntax
- Pattern: agent reads docs, writes minimal test, runs it, then uses confirmed syntax in solution

## Goals

1. **One working example per stdlib module** in `examples/runnable/`
2. **Examples show common patterns** (not just function calls)
3. **`ailang docs` links to relevant examples**
4. **CI verifies all examples compile and run**

## Solution Design

### New Example Files

| Module | Example File | Key Patterns |
|--------|-------------|--------------|
| std/string | `examples/runnable/stdlib_string.ail` | split, join, contains, find, substring, replace |
| std/list | `examples/runnable/stdlib_list.ail` | map, filter, foldl, sortBy, nth, flatMap |
| std/fs | `examples/runnable/stdlib_fs.ail` | readFile, writeFile, listDir, isDir, mkdirAll |
| std/clock | `examples/runnable/stdlib_clock.ail` | now(), nowMillis(), formatted time |
| std/json | `examples/runnable/stdlib_json.ail` | encode, decode, nested objects |
| std/env | `examples/runnable/stdlib_env.ail` | getArgs, getEnv, getEnvOr |
| std/math | `examples/runnable/stdlib_math.ail` | abs, floor, ceil, sqrt, pow |

### `ailang docs` integration

When showing module docs, include a "Try it" section:
```
$ ailang docs std/clock
...
Try it:
  ailang run examples/runnable/stdlib_clock.ail --caps IO,Clock
```

### CI verification

Add `make verify-stdlib-examples` target that runs all `stdlib_*.ail` files and checks for non-zero exit.

## Success Criteria

- [ ] Every std/ module has a working example in `examples/runnable/`
- [ ] `make verify-stdlib-examples` passes
- [ ] Examples updated in `examples/manifest.json`
- [ ] `ailang docs` shows example file reference

## Timeline

- Half day: Write 7 example files (~50 LOC each = ~350 LOC)
- Half day: CI integration + docs linking (~100 LOC)
- Total: ~450 LOC
