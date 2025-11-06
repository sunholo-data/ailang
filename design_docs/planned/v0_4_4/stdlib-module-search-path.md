# Global Stdlib Module Search Path

**Status**: Planned
**Target**: v0.4.4
**Priority**: P0 (High) - Blocks agent eval benchmarks
**Estimated**: 2 days (8h implementation + 6h testing + 2h docs)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax changes |
| Preserve Semantic Clarity | + | +1 | Makes imports work consistently regardless of working directory |
| Increase Determinism | + | +1 | Same code produces same results from any directory |
| Lower Token Cost | 0 | 0 | No change to code written by AI |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current Failure Mode:**

AILANG's module loader cannot find stdlib modules when running from non-project directories (e.g., eval benchmark temp directories). This causes all stdlib imports to fail with `LDR001: module not found`.

**Root Cause:**

The module loader (`internal/loader/`) searches for modules relative to the current working directory only. It has no knowledge of a global stdlib location.

**Current State:**
- Module search: Only checks current working directory + relative paths
- Stdlib location: `$PROJECT_ROOT/std/` (not discoverable from temp dirs)
- Agent eval failures: 4/19 benchmarks (21%) timeout trying to import stdlib
- Agent AILANG success rate: **76.3%** (down from 86.8% in v0.4.2, **-10.5% regression**)
- Failing benchmarks: `effect_composition`, `effect_tracking_io_fs`, `deterministic_list_transform`, `exhaustive_pattern_matching`

**Impact:**
- **Who is affected?** AI code generation benchmarks, users running AILANG from non-project directories
- **How significant?** **P0 blocker** - Agent eval mode is fundamentally broken for any benchmark requiring stdlib imports
- **Data loss?** No, but evaluation results are incorrect/incomplete
- **User experience:** Agents spend entire 60-second timeout searching for modules that exist but can't be found

**Example Error:**
```
Error: module loading error: failed to load std/fs: LDR001: module not found: std/fs
```

**Agent Behavior:**
- Agent writes correct code: `import std/fs (readFile, writeFile)`
- Agent runs code from temp directory: `/tmp/ailang_eval/benchmark_xyz/`
- Loader searches: `./std/fs.ail`, `../std/fs.ail`, etc. (never finds it)
- Agent gets "module not found" error
- Agent spends 8+ turns debugging, then times out

## Goals

**Primary Goal:** Enable stdlib imports to work from any working directory by implementing global module search paths.

**Success Metrics:**
- Agent AILANG eval success rate returns to ≥85% (v0.4.2 baseline)
- All 4 failing benchmarks pass with correct stdlib imports
- stdlib modules findable from any working directory
- Zero new test failures in existing test suite
- Backwards compatible: existing project-relative imports continue working

## Solution Design

### Overview

Add a **global stdlib search path** mechanism to the module loader that checks standard locations for stdlib modules when relative paths fail.

**Search order:**
1. Current working directory (existing behavior)
2. Relative to source file location (existing behavior)
3. **NEW:** `AILANG_STDLIB_PATH` environment variable
4. **NEW:** Compiled-in default (`/usr/local/share/ailang/std`, `~/.ailang/std`)
5. **NEW:** Relative to `ailang` binary location (`$BINARY_DIR/../std`)

This mirrors Go's module system, Python's site-packages, and other language stdlib mechanisms.

### Architecture

**Module Loader Changes (`internal/loader/`):**

```go
// ModuleSearchPath defines where to look for modules
type ModuleSearchPath struct {
    // WorkingDir: Current directory (existing)
    WorkingDir string

    // SourceRelative: Relative to source file (existing)
    SourceRelative string

    // StdlibPaths: Global stdlib locations (NEW)
    StdlibPaths []string
}

// SearchForModule tries each path in order
func (l *Loader) SearchForModule(moduleName string) (string, error) {
    // 1. Try working directory
    // 2. Try source-relative
    // 3. Try each stdlib path (NEW)
    // 4. Return LDR001 if not found
}

// GetStdlibPaths returns stdlib search paths in priority order (NEW)
func GetStdlibPaths() []string {
    paths := []string{}

    // 1. Environment variable (highest priority)
    if envPath := os.Getenv("AILANG_STDLIB_PATH"); envPath != "" {
        paths = append(paths, envPath)
    }

    // 2. User directory
    if homeDir, err := os.UserHomeDir(); err == nil {
        paths = append(paths, filepath.Join(homeDir, ".ailang/std"))
    }

    // 3. Relative to binary
    if exePath, err := os.Executable(); err == nil {
        exeDir := filepath.Dir(exePath)
        paths = append(paths, filepath.Join(exeDir, "..", "std"))
    }

    // 4. System-wide (for installed binaries)
    paths = append(paths, "/usr/local/share/ailang/std")

    return paths
}
```

**Components:**
1. **StdlibPath Resolution** - Find stdlib location at runtime
2. **Module Search Enhancement** - Check stdlib paths after relative paths
3. **Caching** - Cache resolved stdlib path (computed once per process)
4. **Diagnostics** - Log search trace when module not found (for debugging)

### Implementation Plan

**Phase 1: Core Search Path Logic** (~4 hours)
- [ ] Add `GetStdlibPaths()` function to `internal/loader/loader.go`
- [ ] Add `AILANG_STDLIB_PATH` environment variable support
- [ ] Implement binary-relative path resolution
- [ ] Add stdlib path caching
- [ ] Unit tests for path resolution (15+ test cases)

**Phase 2: Module Loader Integration** (~2 hours)
- [ ] Modify `Loader.LoadModule()` to use stdlib paths
- [ ] Update search trace logging to include stdlib paths tried
- [ ] Preserve existing behavior for project-relative imports
- [ ] Add integration test with temp directory

**Phase 3: Eval Harness Integration** (~2 hours)
- [ ] Update `internal/eval_harness/runner.go` to set `AILANG_STDLIB_PATH`
- [ ] Point to actual project stdlib directory
- [ ] Test agent eval benchmarks locally
- [ ] Verify all 4 failing benchmarks now pass

**Phase 4: Testing & Documentation** (~8 hours)
- [ ] Run full test suite (ensure zero new failures)
- [ ] Run agent eval baseline (verify ≥85% success rate)
- [ ] Update CLAUDE.md with `AILANG_STDLIB_PATH` usage
- [ ] Add examples to documentation
- [ ] Update installation guide for different deployment scenarios

### Files to Modify/Create

**New files:**
- `internal/loader/stdlib_path.go` - Stdlib path resolution logic (~100 LOC)
- `internal/loader/stdlib_path_test.go` - Unit tests (~200 LOC)

**Modified files:**
- `internal/loader/loader.go` - Integrate stdlib paths into search (~30 LOC added)
- `internal/loader/loader_test.go` - Integration tests (~50 LOC added)
- `internal/eval_harness/runner.go` - Set `AILANG_STDLIB_PATH` in eval environment (~10 LOC)
- `CLAUDE.md` - Document `AILANG_STDLIB_PATH` usage (~20 LOC)
- `docs/guides/modules.md` - Document module search behavior (~50 LOC)

**Total new code:** ~460 LOC (test-heavy, production code is ~140 LOC)

## Examples

### Example 1: Agent Eval Benchmark (Current Failure)

**Before (v0.4.3 - FAILS):**
```bash
# Agent writes correct code
$ cat /tmp/ailang_eval/benchmark_xyz/benchmark/solution.ail
module benchmark/solution
import std/fs (readFile, writeFile)
export func main() -> () ! {IO, FS} { ... }

# Agent runs from temp directory
$ cd /tmp/ailang_eval/benchmark_xyz
$ ailang run --entry main --caps IO,FS benchmark/solution.ail
Error: LDR001: module not found: std/fs
```

**After (v0.4.4 - WORKS):**
```bash
# Eval harness sets AILANG_STDLIB_PATH
$ export AILANG_STDLIB_PATH=/Users/mark/dev/sunholo/ailang/std
$ cd /tmp/ailang_eval/benchmark_xyz
$ ailang run --entry main --caps IO,FS benchmark/solution.ail
→ Type checking...
→ Effect checking...
✓ Running benchmark/solution.ail
[output]
```

### Example 2: Manual Usage from Any Directory

**Scenario:** User wants to run AILANG code from Downloads folder

**Before (v0.4.3 - FAILS):**
```bash
$ cd ~/Downloads
$ ailang run --entry main test.ail
Error: LDR001: module not found: std/io
```

**After (v0.4.4 - WORKS):**
```bash
$ cd ~/Downloads
$ ailang run --entry main test.ail
# Works! Loader finds std/io via binary-relative path
```

### Example 3: CI/CD Deployment

**Scenario:** Docker container with installed `ailang` binary

```dockerfile
# Install ailang to /usr/local/bin
COPY ailang /usr/local/bin/ailang
COPY std /usr/local/share/ailang/std

# Works from any directory
WORKDIR /app
CMD ["ailang", "run", "--entry", "main", "app.ail"]
```

## Success Criteria

- [ ] Agent eval `effect_composition` benchmark passes (both Claude models)
- [ ] Agent eval `effect_tracking_io_fs` benchmark passes (both Claude models)
- [ ] Agent eval `deterministic_list_transform` benchmark passes (both Claude models)
- [ ] Agent eval `exhaustive_pattern_matching` benchmark passes (both Claude models)
- [ ] Agent AILANG success rate ≥85% (return to v0.4.2 baseline)
- [ ] Can import `std/io`, `std/fs`, etc. from `/tmp` directory
- [ ] Existing test suite passes (zero new failures)
- [ ] `AILANG_STDLIB_PATH` environment variable works correctly
- [ ] Binary-relative path resolution works after `make install`
- [ ] Error messages include search trace when module not found
- [ ] Documentation updated with module search behavior
- [ ] Examples added showing usage from different directories

## Testing Strategy

**Unit tests (`internal/loader/stdlib_path_test.go`):**
- `GetStdlibPaths()` with `AILANG_STDLIB_PATH` set
- `GetStdlibPaths()` without environment variable
- Binary-relative path resolution (mock executable path)
- Home directory expansion (`~/.ailang/std`)
- Search path priority order
- Edge cases: no home dir, no executable path, etc.

**Integration tests (`internal/loader/loader_test.go`):**
- Load `std/io` from temp directory with `AILANG_STDLIB_PATH` set
- Load `std/fs` from temp directory with binary-relative path
- Load project module (existing behavior preserved)
- Load fails with clear error message when module truly doesn't exist

**Agent eval tests:**
- Run `ailang eval-suite --benchmarks effect_composition,effect_tracking_io_fs,deterministic_list_transform,exhaustive_pattern_matching`
- Verify all 4 benchmarks pass
- Compare agent AILANG success rate to v0.4.2 baseline

**Manual testing:**
- `cd /tmp && ailang run --entry main /tmp/test.ail` (with `import std/io`)
- Run from `~/Downloads` directory
- Run from Docker container with installed binary
- Verify search trace in error messages

## Non-Goals

**Not in this feature:**
- **Custom module registries** - Deferred to future "package manager" work
- **Network-based module loading** - Security concerns, needs separate design
- **Version pinning for stdlib** - stdlib is versioned with compiler
- **Module caching** - Already handled by existing loader logic
- **IDE integration** - AILANG is not optimized for IDE features

## Timeline

**Day 1** (8 hours):
- Phase 1: Implement `GetStdlibPaths()` and unit tests (4h)
- Phase 2: Integrate into module loader (2h)
- Phase 3: Update eval harness (2h)

**Day 2** (8 hours):
- Phase 4: Full testing (4h)
  - Run test suite
  - Run agent eval benchmarks
  - Manual testing from temp directories
- Phase 4: Documentation (2h)
  - Update CLAUDE.md
  - Update module guide
- Buffer for unexpected issues (2h)

**Total: ~16 hours across 2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking existing imports | High | Preserve search order: project-relative first, stdlib last |
| Performance regression | Medium | Cache stdlib path (computed once), minimal overhead |
| Security: Untrusted stdlib injection | Medium | `AILANG_STDLIB_PATH` requires explicit user configuration |
| Platform differences (Windows paths) | Medium | Use `filepath.Join()` for cross-platform compatibility |
| Binary not in PATH | Low | Provide clear error if stdlib not found, suggest `AILANG_STDLIB_PATH` |

## References

- **v0.4.3 Release Post-mortem**: Agent eval regression investigation (this analysis)
- **Module Loader Implementation**: `internal/loader/loader.go`
- **Eval Harness**: `internal/eval_harness/runner.go`
- **Similar Systems**:
  - Go: `GOROOT`, `GOPATH` environment variables
  - Python: `sys.path`, `PYTHONPATH`, site-packages
  - Rust: `CARGO_HOME`, registry caching
  - Node.js: `NODE_PATH`, node_modules resolution

## Future Work

- **Package manager** (v0.5.0+): Third-party module registry with versioning
- **Module aliasing**: `import std/io as io`
- **Selective imports with renaming**: `import std/fs (readFile as read)`
- **Module preloading**: Cache parsed stdlib modules for faster startup
- **Stdlib versioning**: Pin specific stdlib version independently of compiler

---

**Document created**: 2025-11-06
**Last updated**: 2025-11-06
