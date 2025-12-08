# M-DX11: Better Stdlib Discovery for AI Agents

**Status**: Planned
**Target**: v0.5.7
**Priority**: P1 (Medium)
**Estimated**: 3 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | AI agents discover stdlib without GitHub search |
| Preserve Semantic Clarity | + | +1 | Shows wrapper functions, not raw builtins |
| Increase Determinism | 0 | 0 | No impact on language semantics |
| Lower Token Cost | + | +1 | Fewer tokens to discover and import stdlib |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

AI agents building AILANG programs need to discover stdlib functions, but the current CLI tooling exposes **raw builtins** (e.g., `_io_print`) instead of the **wrapper functions** (e.g., `print`) that users actually import.

**Current State:**
- `ailang builtins list --by-module` shows `_io_print`, `_fs_readFile`, etc.
- No way to see wrapper function signatures (what users actually call)
- No way to discover available modules without knowing names
- No usage examples in CLI output
- AI agents must search GitHub as fallback

**Impact:**
- AI agents waste tokens searching for stdlib information
- Code generation uses wrong function names (raw builtins vs wrappers)
- Discovery requires external file access (GitHub, local files)
- Friction for AI-first development workflow

## Goals

**Primary Goal:** Enable AI agents to discover stdlib functions entirely from CLI, without external file access.

**Success Metrics:**
- `ailang docs --list` shows all available modules
- `ailang docs std/io` shows wrapper functions with signatures
- `ailang docs --examples std/array` shows usage examples
- AI agents can generate correct import statements and function calls

## Solution Design

### Overview

Add `ailang docs` command that reads stdlib `.ail` files and extracts documentation for wrapper functions. This complements `ailang builtins` (low-level) with a user-facing docs command (high-level).

### Architecture

**Components:**
1. **Module Discovery**: Scan `std/` directory for `.ail` files
2. **Doc Extraction**: Parse stdlib files to extract exports, comments, and signatures
3. **CLI Command**: New `docs` subcommand with flags for listing/viewing/examples

### Implementation Plan

**Phase 1: Core Implementation** (~2 hours)
- [ ] Add `docs.go` with `docsCommand()` function
- [ ] Implement module discovery (list `std/*.ail` files)
- [ ] Parse stdlib files to extract exported functions
- [ ] Display wrapper function signatures and doc comments

**Phase 2: Examples and Polish** (~1 hour)
- [ ] Extract and display usage examples from stdlib comments
- [ ] Add `--examples` flag for detailed examples
- [ ] Update help text and documentation
- [ ] Test with AI agent workflow

### Files to Modify/Create

**New files:**
- `cmd/ailang/docs.go` - Docs command implementation (~200 LOC)

**Modified files:**
- `cmd/ailang/main.go` - Add `docs` case to switch (~5 LOC)
- `cmd/ailang/help.go` - Add docs command to help text (~5 LOC)

## Examples

### Example 1: List Available Modules

**Before:**
```bash
$ # No way to list modules from CLI
$ # Must search GitHub or local files
```

**After:**
```bash
$ ailang docs --list
Available stdlib modules:

  std/ai       AI oracle calls (requires AI effect)
  std/array    Fixed-size arrays with O(1) access
  std/clock    Time operations (requires Clock effect)
  std/debug    Debug logging (requires Debug effect)
  std/env      Environment variables (requires Env effect)
  std/fs       File system operations (requires FS effect)
  std/game     Game development utilities
  std/io       Console I/O (requires IO effect)
  std/json     JSON encoding/decoding
  std/list     Linked list operations
  std/net      HTTP requests (requires Net effect)
  std/option   Option type for nullable values
  std/rand     Random number generation (requires Rand effect)
  std/result   Result type for error handling
  std/string   String manipulation
```

### Example 2: View Module Documentation

**Before:**
```bash
$ ailang builtins list --by-module | grep std/io
# std/io (3)
  _io_print                      [io]
  _io_println                    [io]
  _io_readLine                   [io]
```

**After:**
```bash
$ ailang docs std/io
# std/io - Console I/O

## Exports

  print(s: string) -> () ! {IO}
    Print string without newline

  println(s: string) -> () ! {IO}
    Print string with newline

  readLine() -> string ! {IO}
    Read line from stdin

## Usage

  import std/io (println, readLine)
  -- or --
  import std/io as IO
```

### Example 3: Show Examples

```bash
$ ailang docs --examples std/array
# std/array - Fixed-size Arrays

## Usage Examples

-- Create array of 10 zeros
let arr = Array.make(10, 0)

-- Convert list to array for O(1) access
let arr = Array.fromList([1, 2, 3, 4, 5])

-- Get element (0-based indexing)
let x = Array.get(arr, 0)  -- Returns 1

-- Safe get returns Option
match Array.getOpt(arr, 100) {
  Some(x) => print(show(x)),
  None    => println("out of bounds")
}

-- Set returns new array (immutable)
let arr2 = Array.set(arr, 0, 99)
```

## Success Criteria

- [ ] `ailang docs --list` lists all 15 stdlib modules
- [ ] `ailang docs std/io` shows wrapper function signatures
- [ ] `ailang docs --examples std/array` shows usage examples
- [ ] Documentation extracted from actual stdlib files (not hardcoded)
- [ ] All tests passing
- [ ] Help text updated
- [ ] CHANGELOG updated

## Testing Strategy

**Unit tests:**
- Module discovery finds all stdlib files
- Doc extraction parses exports correctly
- Output formatting is consistent

**Integration tests:**
- `ailang docs --list` returns expected modules
- `ailang docs std/io` shows correct signatures

**Manual testing:**
- Verify AI agent can use docs to write correct imports
- Check output readability in terminal

## Non-Goals

**Not in this feature:**
- Automatic doc generation for user modules - stdlib only for now
- Web/HTML documentation output - CLI text only
- IDE integration - AI agents use CLI

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Stdlib file format changes | Med | Extract from AST, not regex |
| Missing doc comments in stdlib | Low | Show signatures even without comments |

## References

- Agent inbox message: `4c679b5a-750e-4445-926f-54c8e0fc89ee`
- Stdlib files: `std/*.ail`
- Builtins command: `cmd/ailang/doctor.go`

## Future Work

- `ailang docs <user-module>` - extend to user modules
- `--json` flag for machine-readable output
- Search across all modules: `ailang docs --search "http"`

---

**Document created**: 2025-12-08
**Last updated**: 2025-12-08
