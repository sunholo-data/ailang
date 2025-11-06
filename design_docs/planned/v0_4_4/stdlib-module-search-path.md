# Global Stdlib Module Search Path

**Status**: Planned
**Target**: v0.4.4
**Priority**: P0 (High) - Blocks agent eval benchmarks
**Estimated**: 3 days (12h implementation + 8h testing + 4h docs)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No syntax changes |
| Preserve Semantic Clarity | + | +1 | Makes imports work consistently regardless of working directory |
| Increase Determinism | ++ | +2 | Same code + same stdlib version = identical results; version pinning prevents silent skew |
| Lower Token Cost | 0 | 0 | No change to code written by AI |
| **Net Score** | | **+3** | **Decision: Move forward with version pinning** |

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

Add a **global stdlib search path** mechanism to the module loader that checks standard locations for **`std/*` imports only** when relative paths fail.

**Key Design Decisions:**

1. **Scoped to `std/*` only** - Global search only applies to imports with `std/` prefix. Project modules continue using existing relative resolution. This prevents surprising shadowing and keeps resolution predictable.

2. **Version pinning** - Stdlib includes a `VERSION` file; compiler embeds expected version and checks for mismatches. Prevents silent version skew between compiler and stdlib.

3. **Platform-aware paths** - Uses OS-specific data directories (XDG on Linux, `%APPDATA%` on Windows, `~/Library/Application Support` on macOS).

4. **Path hygiene** - Sanitizes module names (allows only `[a-zA-Z0-9_/-]`; rejects `..`, absolute paths). Treats names as logical IDs, never evaluates raw filesystem traversals.

5. **Observable diagnostics** - `--trace-loader` flag shows search trace with helpful hints when modules not found.

**Search order (for `std/*` imports only):**
1. Project-relative (existing behavior - preserves local overrides)
2. Source-file-relative (existing behavior)
3. **NEW:** Binary-relative (`$BINARY_DIR/../std`) - "just works" for installed binaries
4. **NEW:** `--stdlib-path` CLI flag - explicit override for CI/testing
5. **NEW:** `AILANG_STDLIB_PATH` environment variable (colon/semicolon separated path list)
6. **NEW:** User data directory (platform-specific)
   - Linux: `$XDG_DATA_HOME/ailang/std` or `~/.local/share/ailang/std`
   - macOS: `~/Library/Application Support/ailang/std`
   - Windows: `%APPDATA%\ailang\std`
7. **NEW:** System data directories
   - `/usr/local/share/ailang/std`
   - `/usr/share/ailang/std`

This mirrors Go's module system, Python's site-packages, and Rust's CARGO_HOME.

### Architecture

**Module Loader Changes (`internal/loader/`):**

```go
// StdlibResolver handles stdlib module resolution
type StdlibResolver struct {
    // Cached paths (computed once per process)
    searchPaths []string
    once        sync.Once

    // Negative cache: module name -> tried paths
    negativeCache map[string][]string
    cacheMutex    sync.RWMutex

    // CLI override
    cliOverridePath string

    // Version checking
    expectedVersion string // Embedded at compile time
}

// ResolveStdlib resolves std/* imports only
func (r *StdlibResolver) ResolveStdlib(moduleName string) (string, error) {
    // 1. Check this is a std/* import
    if !strings.HasPrefix(moduleName, "std/") {
        return "", ErrNotStdlibModule
    }

    // 2. Sanitize module name (security)
    if err := validateModuleName(moduleName); err != nil {
        return "", err
    }

    // 3. Check negative cache
    if cached, ok := r.checkNegativeCache(moduleName); ok {
        return "", errWithSearchTrace(moduleName, cached)
    }

    // 4. Try each search path
    r.once.Do(r.initializeSearchPaths)
    triedPaths := []string{}

    for _, base := range r.searchPaths {
        fullPath := filepath.Join(base, moduleName+".ail")
        if fileExists(fullPath) {
            // 5. Verify stdlib version
            if err := r.checkStdlibVersion(base); err != nil {
                log.Warnf("stdlib at %s: %v", base, err)
                continue
            }
            return fullPath, nil
        }
        triedPaths = append(triedPaths, fullPath)
    }

    // 6. Cache negative result
    r.cacheNegative(moduleName, triedPaths)
    return "", errWithSearchTrace(moduleName, triedPaths)
}

// initializeSearchPaths builds the search path list once
func (r *StdlibResolver) initializeSearchPaths() {
    paths := []string{}

    // 1. CLI flag (highest priority)
    if r.cliOverridePath != "" {
        paths = append(paths, r.cliOverridePath)
    }

    // 2. Binary-relative (tends to "just work")
    if exePath, err := os.Executable(); err == nil {
        realPath, _ := filepath.EvalSymlinks(exePath) // Follow symlinks
        exeDir := filepath.Dir(realPath)
        paths = append(paths, filepath.Join(exeDir, "..", "std"))
    }

    // 3. AILANG_STDLIB_PATH (colon/semicolon separated)
    if envPaths := os.Getenv("AILANG_STDLIB_PATH"); envPaths != "" {
        sep := getPathSeparator() // ':' on POSIX, ';' on Windows
        for _, p := range strings.Split(envPaths, sep) {
            if p = strings.TrimSpace(p); p != "" {
                paths = append(paths, p)
            }
        }
    }

    // 4. User data directory (platform-specific)
    if userDataDir := getUserDataDir(); userDataDir != "" {
        paths = append(paths, filepath.Join(userDataDir, "ailang", "std"))
    }

    // 5. System data directories
    paths = append(paths, "/usr/local/share/ailang/std")
    paths = append(paths, "/usr/share/ailang/std")

    r.searchPaths = paths
}

// getUserDataDir returns OS-specific user data directory
func getUserDataDir() string {
    switch runtime.GOOS {
    case "linux", "freebsd", "openbsd", "netbsd":
        if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
            return xdg
        }
        if home := os.Getenv("HOME"); home != "" {
            return filepath.Join(home, ".local", "share")
        }
    case "darwin":
        if home := os.Getenv("HOME"); home != "" {
            return filepath.Join(home, "Library", "Application Support")
        }
    case "windows":
        if appData := os.Getenv("APPDATA"); appData != "" {
            return appData
        }
    }
    return ""
}

// validateModuleName ensures path hygiene (security)
func validateModuleName(name string) error {
    // Allow only [a-zA-Z0-9_/-]
    for _, ch := range name {
        if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
             (ch >= '0' && ch <= '9') || ch == '_' || ch == '/' || ch == '-') {
            return fmt.Errorf("invalid character in module name: %q", ch)
        }
    }
    // Reject .. and absolute paths
    if strings.Contains(name, "..") || filepath.IsAbs(name) {
        return fmt.Errorf("module name cannot contain '..' or be absolute")
    }
    return nil
}

// checkStdlibVersion verifies VERSION file matches expected version
func (r *StdlibResolver) checkStdlibVersion(stdlibRoot string) error {
    versionFile := filepath.Join(stdlibRoot, "VERSION")
    data, err := os.ReadFile(versionFile)
    if err != nil {
        return fmt.Errorf("missing VERSION file")
    }
    actualVersion := strings.TrimSpace(string(data))
    if actualVersion != r.expectedVersion {
        return fmt.Errorf("version mismatch: expected %s, found %s",
            r.expectedVersion, actualVersion)
    }
    return nil
}

// errWithSearchTrace creates a helpful error message
func errWithSearchTrace(moduleName string, triedPaths []string) error {
    msg := fmt.Sprintf("LDR001: module not found: %s\nsearched:\n", moduleName)
    for _, p := range triedPaths {
        msg += fmt.Sprintf("  %s\n", p)
    }
    msg += "\ntip: set AILANG_STDLIB_PATH=/path/to/ailang/std or use --stdlib-path\n"
    return errors.New(msg)
}
```

**Components:**
1. **StdlibPath Resolution** - Find stdlib location at runtime with platform-aware logic
2. **Scoped Resolution** - Only applies to `std/*` imports; project modules unaffected
3. **Version Pinning** - Check `VERSION` file in stdlib root; warn/fail on mismatch
4. **Path Sanitization** - Validate module names to prevent directory traversal attacks
5. **Positive Caching** - Cache resolved stdlib root (computed once per process)
6. **Negative Caching** - Cache failed lookups to avoid repeated filesystem hits (O(1) retries)
7. **Diagnostics** - Detailed search trace with helpful hints when modules not found
8. **CLI Override** - `--stdlib-path` flag for CI/testing without environment variables

### Implementation Plan

**Phase 1: Core Infrastructure** (~6 hours)
- [ ] Create `std/VERSION` file with current version (v0.4.4)
- [ ] Embed expected stdlib version in binary at compile time
- [ ] Create `internal/loader/stdlib_resolver.go` with `StdlibResolver` struct
- [ ] Implement `validateModuleName()` with security checks
- [ ] Implement platform-specific `getUserDataDir()`
- [ ] Implement `getPathSeparator()` for Windows vs POSIX
- [ ] Add positive and negative caching logic
- [ ] Unit tests for path resolution (20+ test cases)
- [ ] Unit tests for module name sanitization (10+ test cases)

**Phase 2: Module Loader Integration** (~3 hours)
- [ ] Modify `Loader.LoadModule()` to check `std/*` prefix
- [ ] Integrate `StdlibResolver` into loader
- [ ] Add `--stdlib-path` CLI flag to `cmd/ailang/main.go`
- [ ] Update search trace logging with detailed diagnostics
- [ ] Preserve existing behavior for project-relative imports
- [ ] Add integration test: load `std/io` from `/tmp` with `--stdlib-path`

**Phase 3: Version Checking & Diagnostics** (~3 hours)
- [ ] Implement `checkStdlibVersion()` with VERSION file parsing
- [ ] Add `--trace-loader` flag for verbose diagnostics
- [ ] Add `--strict` flag to fail (vs warn) on version mismatch
- [ ] Implement helpful error messages with search trace
- [ ] Add environment variable `AILANG_DEBUG=loader` support
- [ ] Unit tests for version checking (5+ test cases)

**Phase 4: Eval Harness & CLI Integration** (~2 hours)
- [ ] Update `internal/eval_harness/runner.go` to use `--stdlib-path` flag
- [ ] Point to actual project stdlib directory in eval runs
- [ ] Test agent eval benchmarks locally (4 failing benchmarks)
- [ ] Verify all 4 failing benchmarks now pass
- [ ] Add `--stdlib-path` to eval harness command recording

**Phase 5: Testing** (~6 hours)
- [ ] Run full test suite (ensure zero new failures)
- [ ] Run agent eval baseline (verify ≥85% success rate)
- [ ] Test Windows path separators and %APPDATA%
- [ ] Test XDG_DATA_HOME on Linux
- [ ] Test symlink resolution for binary-relative paths
- [ ] Test security: reject `std/../../etc/passwd`
- [ ] Test negative caching (repeated failures are O(1))
- [ ] Test version mismatch warnings
- [ ] Test `--stdlib-path` vs `AILANG_STDLIB_PATH` precedence
- [ ] Manual testing from `/tmp`, `~/Downloads`, Docker containers

**Phase 6: Documentation** (~4 hours)
- [ ] Create `docs/guides/stdlib-resolution.md` (comprehensive guide)
- [ ] Update CLAUDE.md with `AILANG_STDLIB_PATH` and `--stdlib-path` usage
- [ ] Update `cmd/ailang/help.go` with stdlib-path flag help
- [ ] Add Docker/container deployment examples
- [ ] Document version pinning mechanism
- [ ] Document shadowing behavior (local std/ doesn't override global)
- [ ] Add troubleshooting section with common errors
- [ ] Update installation guide for different deployment scenarios

### Files to Modify/Create

**New files:**
- `std/VERSION` - Version identifier file (1 line: `v0.4.4`)
- `internal/loader/stdlib_resolver.go` - Stdlib resolution logic (~300 LOC)
- `internal/loader/stdlib_resolver_test.go` - Unit tests (~400 LOC)
- `docs/guides/stdlib-resolution.md` - Comprehensive guide (~150 LOC)

**Modified files:**
- `internal/loader/loader.go` - Integrate `StdlibResolver` (~50 LOC added)
- `internal/loader/loader_test.go` - Integration tests (~80 LOC added)
- `cmd/ailang/main.go` - Add `--stdlib-path` and `--trace-loader` flags (~30 LOC)
- `cmd/ailang/help.go` - Document new flags (~40 LOC)
- `internal/eval_harness/runner.go` - Use `--stdlib-path` flag (~15 LOC)
- `Makefile` - Embed version at build time via `-ldflags` (~5 LOC)
- `CLAUDE.md` - Document stdlib resolution (~100 LOC)
- `docs/guides/modules.md` - Update module search behavior (~80 LOC)

**Total new code:** ~1250 LOC (60% tests, 40% production code + docs)

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
# Eval harness uses --stdlib-path flag
$ cd /tmp/ailang_eval/benchmark_xyz
$ ailang run --entry main --caps IO,FS --stdlib-path /Users/mark/dev/sunholo/ailang/std benchmark/solution.ail
→ Type checking...
→ Effect checking...
✓ Running benchmark/solution.ail
[output]

# OR use environment variable
$ export AILANG_STDLIB_PATH=/Users/mark/dev/sunholo/ailang/std
$ ailang run --entry main --caps IO,FS benchmark/solution.ail
[works!]
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

### Example 3: Helpful Error Diagnostics

**Scenario:** User forgets to set stdlib path

```bash
$ cd ~/Downloads
$ ailang run test.ail --trace-loader
Error: module loading error: failed to load std/io
LDR001: module not found: std/io
searched:
  ./std/io.ail
  ~/Downloads/std/io.ail
  /usr/local/bin/../std/io.ail
  ~/.local/share/ailang/std/io.ail
  /usr/local/share/ailang/std/io.ail
  /usr/share/ailang/std/io.ail

tip: set AILANG_STDLIB_PATH=/path/to/ailang/std or use --stdlib-path
```

### Example 4: CI/CD Deployment

**Scenario:** Docker container with installed `ailang` binary

```dockerfile
# Install ailang to /usr/local/bin
COPY ailang /usr/local/bin/ailang
COPY std /usr/local/share/ailang/std

# Binary-relative path resolution "just works"
WORKDIR /app
CMD ["ailang", "run", "--entry", "main", "app.ail"]
```

**Scenario:** GitHub Actions CI

```yaml
- name: Run AILANG tests
  run: |
    ailang test --stdlib-path ./std tests/*.ail
```

### Example 5: Version Mismatch Warning

**Scenario:** User has outdated stdlib

```bash
$ ailang run --strict test.ail
Warning: stdlib version mismatch at /usr/local/share/ailang/std
  expected: v0.4.4
  found: v0.4.2

Use --stdlib-path to specify a different stdlib location
Error: strict mode: refusing to run with mismatched stdlib version
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

**Unit tests (`internal/loader/stdlib_resolver_test.go`):**
- **Path resolution** (8 tests):
  - Binary-relative path with symlink resolution
  - `AILANG_STDLIB_PATH` with single path
  - `AILANG_STDLIB_PATH` with multiple paths (colon-separated on POSIX, semicolon on Windows)
  - `--stdlib-path` CLI flag overrides environment variable
  - Platform-specific user data dir (XDG, %APPDATA%, ~/Library)
  - Search order precedence
  - Empty/missing environment variables
  - Edge cases: no home dir, no executable path

- **Module name sanitization** (10 tests):
  - Valid names: `std/io`, `std/fs`, `std/result`
  - Invalid characters rejected: `std/../../etc/passwd`, `std/foo@bar`, `std/foo\0bar`
  - Absolute paths rejected: `/etc/passwd`, `C:\Windows\System32`
  - Relative traversal rejected: `std/../etc`, `std/./../../secrets`
  - Valid hyphen/underscore: `std/json-parser`, `std/http_client`
  - Empty module name rejected
  - Module name without `std/` prefix (returns ErrNotStdlibModule)

- **Caching** (5 tests):
  - Positive cache: resolved path reused on subsequent calls
  - Negative cache: failed lookups return cached error immediately
  - Negative cache stores tried paths for diagnostics
  - Cache is thread-safe (concurrent access)
  - Cache cleared on resolver recreation

- **Version checking** (7 tests):
  - VERSION file matches expected version (success)
  - VERSION file mismatch (warning logged, continues in non-strict mode)
  - VERSION file mismatch with `--strict` (fails immediately)
  - Missing VERSION file (warning logged, continues)
  - Malformed VERSION file (warning logged)
  - Multiple stdlib paths: skip mismatched versions, use matching one
  - Version embedded correctly at compile time via `-ldflags`

**Integration tests (`internal/loader/loader_test.go`):**
- Load `std/io` from temp directory with `--stdlib-path` flag
- Load `std/fs` from temp directory with `AILANG_STDLIB_PATH` env var
- Load project module `foo/bar` (not `std/*`) uses existing relative resolution
- Load fails with clear error message and search trace
- Error message includes helpful hints
- `--trace-loader` outputs verbose search path
- Project-local `std/io.ail` shadows global stdlib (if present)
- Stdlib resolution works from any working directory (`/tmp`, `~/Downloads`, etc.)

**Platform-specific tests:**
- **Windows**: Path separator (semicolon), %APPDATA%, drive letters, UNC paths
- **macOS**: ~/Library/Application Support, symlinks in /usr/local/bin
- **Linux**: XDG_DATA_HOME, /usr/local/share, /usr/share

**Security tests:**
- Reject `std/../../etc/passwd` (directory traversal)
- Reject `std/foo\0bar` (null byte injection)
- Reject absolute paths in module names
- Reject special characters in module names
- symlink attacks: ensure realpath resolution

**Performance tests:**
- Negative cache: repeated failures are O(1) (not O(n) filesystem hits)
- Positive cache: resolved path reused (measured with benchmarks)
- Search path initialization only happens once per process

**Agent eval tests:**
- Run `ailang eval-suite --benchmarks effect_composition,effect_tracking_io_fs,deterministic_list_transform,exhaustive_pattern_matching`
- Verify all 4 benchmarks pass with `--stdlib-path` flag
- Compare agent AILANG success rate to v0.4.2 baseline (target: ≥85%)
- Ensure eval harness records `--stdlib-path` in run metadata

**Manual testing:**
- `cd /tmp && ailang run --entry main /tmp/test.ail` (with `import std/io`)
- Run from `~/Downloads` directory (binary-relative path should work)
- Run from Docker container with installed binary to /usr/local/bin
- Verify search trace in error messages with `--trace-loader`
- Test GitHub Actions CI workflow with `--stdlib-path ./std`
- Test version mismatch warning with outdated stdlib
- Test `--strict` mode fails on version mismatch

## Non-Goals

**Not in this feature:**
- **Custom module registries** - Deferred to future "package manager" work
- **Network-based module loading** - Security concerns, needs separate design
- **Version pinning for stdlib** - stdlib is versioned with compiler
- **Module caching** - Already handled by existing loader logic
- **IDE integration** - AILANG is not optimized for IDE features

## Timeline

**Day 1** (8 hours):
- Phase 1: Core infrastructure (6h)
  - VERSION file, sanitization, platform-specific paths, caching
  - Unit tests for path resolution and sanitization
- Phase 2: Module loader integration (2h)

**Day 2** (8 hours):
- Phase 3: Version checking & diagnostics (3h)
  - VERSION file checking, --trace-loader, error messages
- Phase 4: Eval harness integration (2h)
  - Update eval harness to use --stdlib-path
  - Test 4 failing benchmarks locally
- Phase 5: Testing (3h)
  - Run full test suite
  - Platform-specific tests
  - Security tests

**Day 3** (8 hours):
- Phase 5: Testing (continued) (4h)
  - Agent eval baseline (full run)
  - Manual testing (Docker, CI, etc.)
  - Performance benchmarks
- Phase 6: Documentation (4h)
  - Create stdlib-resolution.md guide
  - Update CLAUDE.md, help text, installation docs
  - Add troubleshooting section

**Total: ~24 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking existing imports | High | Preserve search order: project-relative first, stdlib last; scope global search to `std/*` only |
| Performance regression | Medium | Cache stdlib path (computed once per process); negative caching for failed lookups |
| Security: Directory traversal attacks | High | Sanitize module names (allow only `[a-zA-Z0-9_/-]`); reject `..` and absolute paths |
| Security: Untrusted stdlib injection | Medium | `AILANG_STDLIB_PATH` requires explicit user configuration; version pinning detects mismatches |
| Platform differences (Windows paths) | Medium | Use `filepath.Join()` for cross-platform compatibility; test on Windows, macOS, Linux |
| Binary not in PATH | Low | Provide clear error if stdlib not found with search trace and hints |
| Version skew (mismatched stdlib) | High | VERSION file checked at runtime; `--strict` mode fails on mismatch; warnings logged |
| Symlink attacks | Medium | Use `filepath.EvalSymlinks()` for binary-relative paths; validate resolved paths |
| Shadow behavior confusion | Medium | Document clearly that project-local `std/` overrides global stdlib (first in search order) |

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

**Post-v0.4.4 enhancements:**
- **Stdlib SHA256 checksums** - Add CHECKSUMS file for integrity verification
- **--allow-stdlib-shadow** - Opt-in flag to allow project-local `std/` to shadow global stdlib
- **Module aliasing** - `import std/io as io` (syntax change, deferred)
- **Selective imports with renaming** - `import std/fs (readFile as read)` (syntax change, deferred)
- **Module preloading** - Cache parsed stdlib modules for faster startup (~10% speedup potential)
- **Package manager** (v0.5.0+) - Third-party module registry with versioning
- **Stdlib versioning independence** - Pin specific stdlib version independently of compiler (requires package manager)

**Out of scope (not aligned with AILANG vision):**
- IDE-specific features (autocompletion based on stdlib path)
- Network-based module loading (security concerns, determinism impact)
- Dynamic stdlib discovery (breaks determinism)

---

**Document created**: 2025-11-06
**Last updated**: 2025-11-06 (revised with version pinning, scoping, and security enhancements)
