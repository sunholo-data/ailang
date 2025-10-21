# M-DX1 Day 3: Developer Tooling Polish

**Status**: Planned
**Target**: v0.3.17
**Priority**: P1 (Medium)
**Estimated**: 3 days
**Dependencies**: M-DX1 (v0.3.10), Entry-Module Prelude (v0.3.16)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Better introspection reduces debug cycles |
| Preserve Semantic Clarity | 0 | 0 | No semantic changes, tooling only |
| Increase Determinism | + | +1 | Capability detection more reliable |
| Lower Token Cost | 0 | 0 | Indirect benefit (faster iterations) |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

During the v0.3.16 entry-module prelude implementation, several developer tooling gaps became apparent:

**Current State:**
1. **No automated deprecation detection** - Old builtin calls (`_net_httpGet`) silently fail with cryptic errors
2. **Fragile capability detection** - String matching breaks with whitespace variations (`!{Net}` vs `! {Net}`)
3. **No type environment introspection** - Can't easily see what bindings are in scope (debugging prelude was manual)
4. **Slow example verification** - 88 examples take ~30s to run sequentially (no parallelization)
5. **Manual prompt releases** - SHA256, version updates, validation all manual steps

**Impact:**
- **AI agents**: Waste time debugging issues that tooling could catch
- **Human developers**: Slow edit-test-debug cycles
- **CI/CD**: Fragile scripts that break on syntax variations
- **Documentation**: Manual errors in prompt releases

**Concrete example from v0.3.16:**
```bash
# Problem: Net tests failed with "undefined: _net_httpGet"
# Root cause: Builtin deprecated in M-DX1 but no automated detection
# Time wasted: ~1 hour investigating + fixing 3 files
# Could have been: 5 seconds with `ailang doctor deprecated`
```

## Goals

**Primary Goal:** Reduce debugging time for common development tasks by 70%

**Success Metrics:**
- Deprecated builtin detection: 0s automated vs 60min manual investigation
- Capability detection: 100% accurate vs ~80% with string matching
- Type environment inspection: Interactive (instant) vs manual code reading
- Example verification: <10s parallel vs 30s sequential
- Prompt releases: 1 command vs 5 manual steps

## Solution Design

### Overview

Add five focused CLI commands and enhancements that address the specific pain points discovered in v0.3.16:

1. `ailang doctor deprecated` - Find outdated builtin calls in code
2. `ailang inspect --required-caps` - Parse and extract capability requirements
3. `ailang env` / REPL `:env` - Show type bindings in scope
4. `verify_examples.go --parallel` - Concurrent test execution
5. `make release-prompt` - Automated prompt versioning

Each tool is standalone, focused, and addresses a real workflow bottleneck.

### Architecture

**Design Principles:**
- **Reuse existing infrastructure** - All tools use existing parser/type checker/registry
- **Machine-readable output** - All tools support `--format json` for scripting
- **Progressive enhancement** - Each tool is optional, can be adopted incrementally
- **Zero semantic changes** - Pure tooling, no language changes

**Components:**
1. **Deprecation Scanner** (`cmd/ailang/doctor_deprecated.go`)
   - Scans AST for builtin calls
   - Cross-references with current registry
   - Suggests modern equivalents

2. **Capability Inspector** (`cmd/ailang/inspect.go`)
   - Parses and type checks file
   - Extracts effect requirements from typed AST
   - Returns JSON with capabilities + metadata

3. **Environment Viewer** (`cmd/ailang/env.go` + `internal/repl/env.go`)
   - Dumps type environment bindings
   - Filters by source (builtin/prelude/user)
   - Pretty-prints type schemes

4. **Parallel Verification** (`scripts/verify_examples.go`)
   - Worker pool pattern (N concurrent runners)
   - File hash caching (skip unchanged files)
   - Fast mode (parse-only, no execution)

5. **Prompt Automation** (`Makefile` + `tools/release_prompt.sh`)
   - Auto-generates SHA256 hash
   - Updates versions.json atomically
   - Validates prompt structure (required sections)

### Implementation Plan

**Phase 1: Deprecation Scanner** (~6 hours)
- [ ] Create `cmd/ailang/doctor_deprecated.go` with AST scanner
- [ ] Build mapping of old → new builtin names (config file)
- [ ] Add tests with sample deprecated code
- [ ] Integration: `ailang doctor deprecated examples/`

**Phase 2: Capability Inspector** (~8 hours)
- [ ] Create `cmd/ailang/inspect.go` with parsing + type checking
- [ ] Extract effects from typed AST (walk function types)
- [ ] Return JSON: `{"capabilities": ["IO", "Net"], "entry_module": true}`
- [ ] Update `verify_examples.go` to use inspector instead of string matching
- [ ] Add tests with Net/Clock/IO examples

**Phase 3: Environment Viewer** (~4 hours)
- [ ] Create `cmd/ailang/env.go` for CLI command
- [ ] Add `:env` command to REPL in `internal/repl/env.go`
- [ ] Implement filtering: `--filter prelude`, `--filter builtin`
- [ ] Pretty-print type schemes with colors
- [ ] Add tests for type environment extraction

**Phase 4: Parallel Verification** (~6 hours)
- [ ] Add worker pool to `scripts/verify_examples.go`
- [ ] Implement file hash caching (`.ailang_cache/` directory)
- [ ] Add `--parallel N` flag (default: GOMAXPROCS)
- [ ] Add `--fast` mode (parse-only, no execution)
- [ ] Update Makefile targets to use parallelization

**Phase 5: Prompt Automation** (~4 hours)
- [ ] Create `tools/release_prompt.sh` script
- [ ] Auto-generate SHA256 hash
- [ ] Update versions.json atomically (temp file + rename)
- [ ] Validate prompt structure (check for required sections)
- [ ] Add `make release-prompt VERSION=vX.Y.Z` target
- [ ] Add tests for script edge cases

**Phase 6: Documentation & Polish** (~2 hours)
- [ ] Update CLAUDE.md with new workflow examples
- [ ] Add examples to docs/guides/
- [ ] Update CHANGELOG.md with all improvements
- [ ] Update README.md with new CLI commands

### Files to Modify/Create

**New files:**
- `cmd/ailang/doctor_deprecated.go` - Deprecation scanner (~200 LOC)
- `cmd/ailang/inspect.go` - Capability inspector (~150 LOC)
- `cmd/ailang/env.go` - Environment viewer CLI (~100 LOC)
- `internal/repl/env.go` - REPL `:env` command (~80 LOC)
- `tools/release_prompt.sh` - Prompt automation (~120 LOC)
- `internal/builtins/deprecated.yml` - Old → new mapping (~50 lines)

**Modified files:**
- `scripts/verify_examples.go` - Add parallelization (~+100 LOC)
- `Makefile` - Add new targets (~+20 LOC)
- `cmd/ailang/main.go` - Wire new commands (~+30 LOC)
- `internal/repl/repl.go` - Add `:env` command (~+15 LOC)

**Total new code: ~1,000 LOC**

## Examples

### Example 1: Deprecation Detection

**Current workflow (v0.3.16):**
```bash
# User encounters error
$ ailang run examples/tests/test_net_localhost.ail
Error: undefined global variable: _net_httpGet from $builtin

# Manual investigation (60+ minutes)
$ grep -r "_net_httpGet" internal/builtins/
# ... no results ...
$ git log --all -S "_net_httpGet"
# ... read commit history ...
# ... figure out it was deprecated in M-DX1 ...
# ... search for replacement ...
# ... update code to use httpRequest ...
```

**New workflow (v0.3.17):**
```bash
$ ailang doctor deprecated examples/tests/test_net_localhost.ail

❌ examples/tests/test_net_localhost.ail:10
   Found: _net_httpGet(url)
   Status: DEPRECATED (removed in v0.3.10)
   Fix: Use std/net.httpGet() or httpRequest() instead
   Example:
     -- Old
     let body = _net_httpGet(url) in

     -- New (Option 1: stdlib wrapper)
     import std/net (httpGet)
     let body = httpGet(url) in

     -- New (Option 2: direct builtin with Result)
     import std/net (httpRequest)
     match httpRequest("GET", url, [], "") {
       Ok(resp) => resp.body,
       Err(e) => "Error: " ++ show(e)
     }

Summary: 1 deprecated builtin found in 1 file
Run with --fix to auto-apply suggested changes
```

**Time saved: 60 minutes → 5 seconds**

### Example 2: Capability Detection

**Current approach (v0.3.16):**
```go
// scripts/verify_examples.go (FRAGILE - breaks on whitespace)
caps := []string{}
if strings.Contains(fileContent, "! {IO") ||
   strings.Contains(fileContent, "_io_") ||
   strings.Contains(fileContent, "import std/io") {
    caps = append(caps, "IO")
}
// Breaks on: !{IO}, ! { IO }, aliased imports, etc.
```

**New approach (v0.3.17):**
```bash
$ ailang inspect --required-caps examples/http_client.ail --format json
{
  "file": "examples/http_client.ail",
  "entry_module": true,
  "required_capabilities": ["IO", "Net"],
  "imported_modules": ["std/io", "std/net"],
  "exported_symbols": ["main"]
}

# Use in scripts
CAPS=$(ailang inspect --required-caps file.ail --format json | jq -r '.required_capabilities | join(",")')
ailang run --caps "$CAPS" file.ail
```

**Accuracy: 100% (parses actual types) vs ~80% (string matching)**

### Example 3: Type Environment Inspection

**Current debugging (v0.3.16):**
```bash
# Question: "Is print in the type environment?"
# Answer: Read source code manually

$ grep -r "ExtendScheme.*print" internal/
# ... read pipeline.go ...
# ... read prelude.go ...
# ... read builtins/io.go ...
# ... maybe add debug prints? ...
```

**New debugging (v0.3.17):**
```bash
# CLI version
$ ailang env examples/hello.ail --filter prelude
Bindings in scope (1):
  print : string -> () ! {IO}  [prelude]

# REPL version
$ ailang repl
> :env
Bindings in scope (52):
  print      : string -> () ! {IO}                   [prelude]
  show       : forall a. Show a => a -> string       [builtin]
  _io_println: string -> () ! {IO}                   [builtin]
  _str_len   : string -> int                         [builtin]
  ...

> :env --filter prelude
Bindings in scope (1):
  print      : string -> () ! {IO}                   [prelude]

> :type print
string -> () ! {IO}
```

**Time saved: 5+ minutes reading code → instant answer**

### Example 4: Parallel Verification

**Current (v0.3.16):**
```bash
$ time go run ./scripts/verify_examples.go --all
Testing 88 examples...
[====================================] 88/88
Examples: 64/88 passing (72.7%)

real    0m32.451s  # SLOW - sequential execution
```

**New (v0.3.17):**
```bash
$ time go run ./scripts/verify_examples.go --all --parallel 8
Testing 88 examples (8 workers)...
[====================================] 88/88
Examples: 64/88 passing (72.7%)

real    0m8.234s  # 4x faster with parallelization

# With caching (second run, no changes)
$ time go run ./scripts/verify_examples.go --all --parallel 8
Testing 88 examples (8 workers, 64 cached)...
[====================================] 88/88
Examples: 64/88 passing (72.7%)

real    0m2.156s  # 15x faster with cache hits

# Fast mode (parse-only, no execution)
$ time go run ./scripts/verify_examples.go --all --fast
Testing 88 examples (parse-only)...
[====================================] 88/88
Parse errors: 0/88

real    0m1.823s  # 18x faster, catches syntax errors only
```

**Time saved: 32s → 2s (with cache) or 8s (without cache)**

### Example 5: Prompt Release Automation

**Current (v0.3.16):**
```bash
# Step 1: Create prompt manually
$ cp prompts/v0.3.15.md prompts/v0.3.16.md
$ vim prompts/v0.3.16.md  # Edit content

# Step 2: Calculate SHA256 manually
$ shasum -a 256 prompts/v0.3.16.md
a1b2c3d4...  prompts/v0.3.16.md

# Step 3: Update versions.json manually
$ vim prompts/versions.json
{
  "active": "v0.3.16",  # Change this
  "versions": {
    "v0.3.16": {  # Add this
      "file": "v0.3.16.md",
      "sha256": "a1b2c3d4...",  # Paste hash
      "released": "2025-10-21"
    },
    ...
  }
}

# Step 4: Validate manually
$ jq . prompts/versions.json  # Check JSON is valid
$ ls prompts/v0.3.16.md       # Check file exists

# Step 5: Commit manually
$ git add prompts/v0.3.16.md prompts/versions.json
$ git commit -m "Release teaching prompt v0.3.16"

# TOTAL TIME: ~10 minutes
# ERROR RISK: High (typos in SHA256, JSON syntax, version mismatch)
```

**New (v0.3.17):**
```bash
$ make release-prompt VERSION=v0.3.16

✓ Found prompt: prompts/v0.3.16.md
✓ Calculated SHA256: a1b2c3d4e5f6...
✓ Updated versions.json (active: v0.3.16)
✓ Validated JSON structure
✓ Validated prompt has required sections:
  - Problem Statement
  - Syntax Reference
  - Examples
✓ Ready to commit

Next step:
  git add prompts/versions.json
  git commit -m "Release teaching prompt v0.3.16"

# TOTAL TIME: 30 seconds
# ERROR RISK: Low (automated validation)
```

**Time saved: 10 minutes → 30 seconds**
**Error reduction: Manual typos → automated validation**

## Success Criteria

- [ ] `ailang doctor deprecated` detects old builtin calls in all 88 examples (dry run)
- [ ] `ailang inspect --required-caps` achieves 100% accuracy on Net/Clock/IO examples
- [ ] `ailang env` and `:env` show prelude bindings correctly in entry modules
- [ ] `verify_examples.go --parallel 8` reduces runtime from 32s to <10s
- [ ] `make release-prompt` successfully releases a test prompt with validation
- [ ] All 2,800+ existing tests passing
- [ ] Documentation updated with workflow examples
- [ ] CHANGELOG.md updated with all improvements

## Testing Strategy

**Unit tests:**
- Deprecation scanner: Test with sample files containing old builtins
- Capability inspector: Test with IO, Net, Clock, combined effects
- Environment viewer: Test with prelude, builtins, user bindings
- Prompt automation: Test SHA256 calculation, JSON updates, validation

**Integration tests:**
- Run `ailang doctor deprecated` on real examples/ directory
- Verify `ailang inspect` output matches manual analysis
- Test parallel verification with cache hits/misses
- Test prompt release with sample prompt file

**Manual testing:**
- Interactive `:env` command in REPL (scrolling, filtering)
- Verify colorized output looks good in terminal
- Test parallel execution under load (96 workers)
- Verify prompt validation catches malformed prompts

## Non-Goals

**Not in this feature:**
- **Auto-fixing deprecated code** (`--fix` flag) - Deferred to v0.3.18 (requires AST rewriting)
- **Cache invalidation logic** - Simple hash-based cache, manual `rm -rf .ailang_cache/`
- **Distributed test execution** - Parallelization is local-only (no CI cluster support)
- **Prompt diffing** - Just releases, no `ailang prompt diff v0.3.15 v0.3.16`
- **Type environment editing** - Read-only inspection, no `:bind x = 42` in REPL

## Timeline

**Day 1** (8 hours):
- Phase 1: Deprecation scanner (6h)
- Phase 2: Start capability inspector (2h)

**Day 2** (8 hours):
- Phase 2: Finish capability inspector (6h)
- Phase 3: Environment viewer (2h)

**Day 3** (8 hours):
- Phase 3: Finish environment viewer (2h)
- Phase 4: Parallel verification (4h)
- Phase 5: Prompt automation (2h)

**Buffer** (4 hours):
- Phase 6: Documentation and polish
- Bug fixes and edge cases

**Total: ~28 hours across 3.5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Parallel execution causes file system race conditions | Medium | Use read-only operations, separate output dirs per worker |
| Cache invalidation too aggressive (false misses) | Low | Use file content hash + ailang binary hash as cache key |
| Deprecation mapping incomplete (missing old builtins) | Low | Start with known cases (_net_httpGet, _net_httpPost), extend as found |
| Environment viewer output too verbose (52+ bindings) | Low | Add filtering (`--filter prelude`), pagination for REPL |
| Prompt validation too strict (rejects valid prompts) | Medium | Make validation warnings, not errors; validate structure, not content |

## References

- [M-DX1 Design Doc](../implemented/v0_3_10/M-DX1_developer_experience.md) - Original developer experience milestone
- [Entry-Module Prelude Design](../v0_3_15/example-parity-vision-alignment.md) - Context for environment inspection need
- [Builtin Registry](../../internal/builtins/spec.go) - Source of truth for builtins
- [Verify Examples Script](../../scripts/verify_examples.go) - Current verification implementation

## Future Work

**v0.3.18+:**
- **Auto-fix mode** (`ailang doctor deprecated --fix`) - Apply suggested changes automatically
- **Distributed testing** - Run examples on CI cluster (100+ workers)
- **Prompt diffing** (`ailang prompt diff v0.3.15 v0.3.16`) - Show what changed between versions
- **Cache server** - Shared cache across developers (Redis-backed)
- **REPL environment editing** - `:bind x = 42` to add runtime bindings
- **Watch mode** - Auto-rerun examples on file changes

---

**Document created**: 2025-10-21
**Last updated**: 2025-10-21
