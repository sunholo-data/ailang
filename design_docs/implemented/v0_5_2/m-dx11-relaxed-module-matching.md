# M-DX11: Relaxed Module Matching

**Status**: Planned
**Target**: v0.5.2
**Priority**: P1 (Medium-High)
**Estimated**: 1.5 days
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes need to carefully sync module declaration with file path |
| Preserve Semantic Clarity | 0 | 0 | Semantics unchanged - just relaxed validation |
| Increase Determinism | 0 | 0 | Still deterministic, validation becomes optional |
| Lower Token Cost | + | +1 | Eliminates MOD010 error-fix cycles for AI coders |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 -> Move forward | <= 0 -> Reject or redesign

## Key Invariant

**This design maintains a critical invariant:**

> **Module identity IS the module declaration string.**
> The path check (MOD010) is a hygiene/lint validation, NOT the source of truth for identity.

This matters for the future dependency system:
- Package resolution and caching will be keyed by `module name + version`, not by filesystem paths
- MOD010 remains a "is your filesystem organized sensibly?" check for humans
- Relaxing MOD010 doesn't change module identity or resolution semantics

## Problem Statement

The current module matching system (MOD010 validation) is too strict, requiring module declarations to exactly match file paths. This creates significant DX friction, especially for:

1. **Temporary files** - AI code generation tools often create files in `/tmp/` or system temp directories
2. **Quick prototyping** - Testing snippets without worrying about file organization
3. **Code generation** - Tools that generate AILANG code in arbitrary locations
4. **Eval harness** - Benchmark runs that create temporary test files

**Current State:**
- Module declaration MUST match canonical file path exactly
- Error: `MOD010: module declaration 'foo/bar' doesn't match canonical path '/tmp/xyz123/foo/bar'`
- Only `std/*` modules are exempt from validation
- No environment variables or flags to relax this
- No temporary directory handling

**Example pain point:**
```bash
# AI generates code to /tmp/test_xyz.ail with:
#   module test/mymodule
#
# ailang run /tmp/test_xyz.ail
# ERROR: MOD010: module declaration 'test/mymodule' doesn't match canonical path 'tmp/test_xyz'
```

**Impact:**
- AI coders waste tokens on MOD010 error-fix cycles
- Eval harness tests require complex directory setup
- Quick prototyping is unnecessarily difficult
- Contributes to perception that AILANG is "fussy" to use

## Goals

**Primary Goal:** Allow AILANG files to run without strict module path validation in common development scenarios, while still surfacing mismatches as warnings.

**Success Metrics:**
- Files in temp directories run without MOD010 errors
- `--relax-modules` flag available for CLI commands
- `AILANG_RELAX_MODULES=1` environment variable works
- Mismatches still produce warnings (not silent success)
- Eval harness can run generated code without path gymnastics
- Zero breaking changes to existing strict behavior (opt-in relaxation)

## Solution Design

### Overview

Add multiple layers of module matching relaxation with **warnings instead of silent success**:

1. **Auto-relaxation for temp paths** - Automatically bypass MOD010 for known temp directories
2. **Environment variable** - `AILANG_RELAX_MODULES=1` for session-wide relaxation
3. **CLI flag** - `--relax-modules` for per-command relaxation
4. **Implicit modules** - Files without `module` declaration don't trigger MOD010 (existing behavior, codified)

**Critical UX Decision: Warnings, Not Silence**

In relaxed mode, mismatches emit a **single warning per file**, not silent success:

```
WARNING MOD010 (relaxed): module 'foo/bar' does not match canonical path 'tmp/hello.ail'
  (suppressed by --relax-modules)
```

This provides:
- AI/eval/scratch files don't break the build
- Humans still see something is off and can fix it when they care
- Teaching mechanism: users learn what's happening

### Architecture

**Components:**

1. **Temp Path Detection** (`internal/loader/loader.go`)
   - Conservative detection: strict prefix check on `os.TempDir()` + well-known patterns
   - If detection fails, fall back to normal behavior (not silent relaxation)
   - Patterns: `/tmp/`, `/var/folders/` (macOS), `os.TempDir()`, Windows `%TEMP%`

2. **Relaxation Config** (`internal/pipeline/config.go`)
   - Add `RelaxModules bool` to pipeline Config
   - Add `hasWarnedForPath map[string]bool` to avoid spamming warnings
   - Read from environment variable and CLI flag

3. **Validation with Warnings** (`internal/pipeline/pipeline_module.go`)
   - Check relaxation config before MOD010 validation
   - In relaxed mode: emit warning, don't fail
   - Track warned paths to avoid duplicate warnings

### Implementation Plan

**Phase 1: Temp Path Auto-Relaxation** (~2 hours)
- [ ] Add `IsTempPath(path string) bool` function to loader
- [ ] Conservative detection: strict prefix on `os.TempDir()`, `/tmp/`, `/var/folders/`
- [ ] If detection uncertain, don't relax (fall back to normal)
- [ ] Add unit tests for temp path detection

**Phase 2: Warning Infrastructure** (~1.5 hours)
- [ ] Add `hasWarnedForPath` set to pipeline to track warned files
- [ ] Create `warnMOD010Relaxed(declaredPath, canonicalPath, reason string)` function
- [ ] Emit warning once per file when mismatch occurs in relaxed mode

**Phase 3: Environment Variable Support** (~1 hour)
- [ ] Check `AILANG_RELAX_MODULES` in pipeline config
- [ ] Values: `1`, `true`, `yes` = relaxed mode
- [ ] Document in CLAUDE.md and CLI help

**Phase 4: CLI Flag Support** (~1 hour)
- [ ] Add `--relax-modules` flag to `ailang run`
- [ ] Add `--relax-modules` flag to `ailang check`
- [ ] Pass through to pipeline config
- [ ] Update help text

**Phase 5: Improved Error Messaging** (~1 hour)
- [ ] In strict mode: suggest `--relax-modules` or `AILANG_RELAX_MODULES=1`
- [ ] In relaxed mode: mention how to get strict again

### Files to Modify/Create

**New files:**
- None (all changes to existing files)

**Modified files:**
- `internal/loader/loader.go` - Add `IsTempPath()` function (~25 LOC)
- `internal/pipeline/config.go` - Add `RelaxModules` field + warning tracker (~15 LOC)
- `internal/pipeline/pipeline_module.go` - Add relaxation checks + warnings (~30 LOC)
- `cmd/ailang/run.go` - Add `--relax-modules` flag (~10 LOC)
- `cmd/ailang/check.go` - Add `--relax-modules` flag (~10 LOC)
- `CLAUDE.md` - Document the feature (~30 lines)

**Total: ~120 LOC changes**

## Error/Warning Message Design

### Strict Mode (Default)

```
MOD010: module 'foo/bar' does not match canonical path 'scratch/experiment'
Suggestions:
  1. Rename module to: module scratch/experiment
  2. Move file to: foo/bar.ail
  3. For temp/scratch files: use --relax-modules or AILANG_RELAX_MODULES=1
```

### Relaxed Mode (Warning)

```
WARNING MOD010 (relaxed): module 'foo/bar' does not match canonical path 'scratch/experiment'
  Running under --relax-modules; mismatch ignored.
  For strict checking, omit --relax-modules flag.
```

### Temp Path Auto-Relaxed (Warning)

```
WARNING MOD010 (temp-path): module 'foo/bar' does not match canonical path 'tmp/xyz123'
  Auto-relaxed for temporary directory. For strict checking, move file outside /tmp/.
```

## Implicit Modules (Codifying Existing Behavior)

Files without a `module` declaration already work today. This design **codifies existing behavior**:

### Import Behavior

- **Files without `module` declaration cannot be imported by other modules**
- They function as "script modules" / REPL snippets only
- This fits the "quick prototyping" story: one-off scripts don't need module headers

### Derived Module IDs

When no `module` declaration exists:
- A module ID is derived from the file path (implementation detail)
- Users **should not rely on this for imports**
- MOD010 is not applied because there is no declaration to check

**This is NOT a new semantic change** - it's documenting and confirming existing behavior.

## Examples

### Example 1: Temp Directory Auto-Relaxation

**Before:**
```bash
echo 'module test/hello
let main = print("Hello")' > /tmp/hello.ail

ailang run --caps IO --entry main /tmp/hello.ail
# ERROR: MOD010: module declaration 'test/hello' doesn't match canonical path 'tmp/hello'
```

**After:**
```bash
ailang run --caps IO --entry main /tmp/hello.ail
# WARNING MOD010 (temp-path): module 'test/hello' does not match canonical path 'tmp/hello'
#   Auto-relaxed for temporary directory.
# Output: Hello
```

### Example 2: Environment Variable

```bash
export AILANG_RELAX_MODULES=1

for file in generated/*.ail; do
    ailang check "$file"
    # Works! Warnings emitted but build doesn't fail
done
```

### Example 3: CLI Flag

```bash
ailang run --relax-modules --caps IO --entry main ./scratch/experiment.ail
# WARNING MOD010 (relaxed): module 'mylib/experiment' does not match canonical path 'scratch/experiment'
#   Running under --relax-modules; mismatch ignored.
# Output: (runs successfully)
```

### Example 4: Implicit Modules (No Declaration)

```bash
# File without module declaration - existing behavior, now documented
echo 'let main = print("Quick test")' > /tmp/quick.ail

ailang run --caps IO --entry main /tmp/quick.ail
# Works: no module declaration = no MOD010 check
# Cannot be imported by other modules (script-only)
```

## Future Compatibility

### Dependency System Compatibility

This design is compatible with a future package/dependency system:

1. **Module identity remains the declaration string** - not filesystem paths
2. **No "search all directories for module X"** behavior introduced
3. **Relaxation only affects "does this file's module match its own path?"** - not resolution

### Command-Level Strictness (Future)

When `ailang build` or `ailang package` are added:
- They should behave as if `--strict-modules` is on
- `AILANG_RELAX_MODULES` should be ignored for library/package builds
- `ailang run` / `ailang check` remain relaxable

**Current arrangement:**
- Strict-by-default for all commands now
- Reserve stricter behavior for future build/package workflows
- Prevents bad layouts from leaking into published libraries

### Multiple Files with Same Module Name

Even with relaxation, if someone has:
- `src/foo/bar.ail` with `module foo/bar`
- `scratch/bar.ail` with `module foo/bar`

The **active module** is determined by the path passed to CLI or by project config, NOT by search. Relaxation doesn't introduce any "find first match" behavior.

## Guidance for AI Tool Authors

If you're generating ephemeral code into temp directories (`os.TempDir()`):
- Rely on built-in temp relaxation
- Warnings will be emitted but won't fail the build

If you're generating project code:
- Write correct module headers matching file paths, OR
- Run with `--relax-modules` only for local dev/CI eval, not in production builds

**Don't treat `--relax-modules` as a permanent project setting.**

## Success Criteria

- [ ] Files in `/tmp/` run without MOD010 errors (auto-relaxation)
- [ ] Files in OS temp dir run without MOD010 errors
- [ ] Mismatches emit warnings (not silent success)
- [ ] `AILANG_RELAX_MODULES=1` relaxes MOD010 with warnings
- [ ] `--relax-modules` flag works on `ailang run`
- [ ] `--relax-modules` flag works on `ailang check`
- [ ] Files without `module` declaration don't trigger MOD010
- [ ] Strict mode (default) unchanged - no breaking changes
- [ ] Error messages suggest relaxation options
- [ ] All existing tests passing
- [ ] Documentation updated (CLAUDE.md, CLI help)

## Testing Strategy

**Unit tests:**
- `IsTempPath()` function with various paths (Unix, macOS, Windows)
- Conservative: uncertain paths return false (don't auto-relax)
- Config parsing for `AILANG_RELAX_MODULES` env var
- Warning deduplication (same file only warned once)

**Integration tests:**
- Create temp file with mismatched module, verify it runs with warning
- Test env var enables relaxation with warning
- Test CLI flag enables relaxation with warning
- Verify strict mode still fails when relaxation disabled
- Verify warning messages match expected format

**Manual testing:**
- Run eval harness with relaxed modules
- Test on macOS and Linux temp paths
- Verify CLAUDE.md examples work

## Non-Goals

**Not in this feature:**
- Changing module identity semantics - identity remains the declaration string
- Changing how imports resolve modules - this only affects validation, not resolution
- Auto-fixing module declarations - that's a separate tool (`ailang fix`)
- Removing MOD010 entirely - strict mode remains default
- Supporting arbitrary module aliases - this is just about path matching
- Silent success in relaxed mode - we always emit warnings on mismatch

## Timeline

**Day 1** (~5 hours):
- Phase 1: Temp path auto-relaxation (2h)
- Phase 2: Warning infrastructure (1.5h)
- Phase 3: Environment variable support (1h)

**Day 2** (~3 hours):
- Phase 4: CLI flag support (1h)
- Phase 5: Improved error messaging (1h)
- Documentation and testing (1h)

**Total: ~8 hours across 1.5 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking existing strict behavior | High | Relaxation is opt-in, default behavior unchanged |
| False positive temp detection | Low | Conservative detection: uncertain = don't relax |
| Silent failures hiding real issues | Medium | Warnings always emitted, not silent success |
| Confusion about when to use | Medium | Clear error messages suggest options |
| Bad layouts leaking into packages | Medium | Future `build`/`package` commands ignore relaxation |

## References

- Error code definition: `internal/errors/codes.go` (MOD010)
- Current validation: `internal/pipeline/pipeline_module.go:87-95`
- Canonical path logic: `internal/loader/loader.go:215-234`
- Stdlib bypass: Already implemented for `std/*` modules

## Future Work

- `ailang fix-modules` command to auto-rewrite module headers to match canonical paths
- `--strict-modules` flag (opposite of relax, for explicit strictness)
- Per-directory `.ailangrc` config for default relaxation
- `ailang build` / `ailang package` with mandatory strict mode
- IDE/LSP integration to suggest fixes

---

**Document created**: 2025-12-03
**Last updated**: 2025-12-03
