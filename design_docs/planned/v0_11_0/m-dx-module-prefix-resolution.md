# M-DX-MODULE-PREFIX: module_prefix Ignored During Package File Resolution

**Status**: Planned
**Target**: v0.11.0
**Priority**: P1 (High — blocks all packages that use module_prefix from being consumed)
**Estimated**: 1 day (well-understood bug, mapping functions already exist)
**Dependencies**: None
**Milestone ID**: M-DX-MODULE-PREFIX
**Created**: 2026-03-30
**Source**: DocParse agent message `ce130505` (sunholo/ailang_parse@0.8.0 publish)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Import resolution should be deterministic regardless of directory layout convention |
| A2: Replayability | 0 | No change |
| A3: Effect Legibility | 0 | No change |
| A4: Explicit Authority | 0 | No change |
| A5: Bounded Verification | +1 | module_prefix is declared in ailang.toml — verifiable at package-load time |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +2 | module_prefix exists specifically to let AI agents organise code naturally; broken resolution defeats the purpose |
| A8: Minimal Syntax | 0 | No new syntax — uses existing module_prefix field |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +2 | Fundamental — packages with module_prefix can't be composed into other projects at all |
| A11: Structured Failure | +1 | Currently gives "module not found" with unhelpful candidates; should try the correct path |
| A12: System Boundary | 0 | No change |

**Net Score: +7** -> **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): module_prefix provides deterministic mapping — just needs to be applied
- [x] A10 (Composability): Without this, any package using module_prefix is broken for consumers

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

Partially. The `module_prefix` feature was implemented in three out of four places where it matters, but the fourth (file path construction) was missed:

| Layer | Uses module_prefix? | Location |
|-------|-------------------|----------|
| Export visibility | Yes | `loader.go:175` — `MapImportToModulePath()` correctly remaps before checking exports |
| MOD010 validation | Yes | `pipeline_module.go:397-410` — allows `module docparse/...` declaration for `pkg/sunholo/ailang_parse/...` |
| Manifest validation | Yes | `manifest.go:165-184` — exports can start with either `vendor/name/` or `module_prefix/` |
| **File path resolution** | **NO** | `loader.go:69-76` — uses raw `parts[2]` without applying prefix |

The fix is localised to one function (`ResolveImport`) and the mapping function already exists.

---

## Problem Statement

### The Bug

A package `sunholo/ailang_parse` has `module_prefix = "docparse"`. Its files live at `docparse/types/document.ail`, and the module declaration is `module docparse/types/document`.

When another package imports it:

```ailang
import pkg/sunholo/ailang_parse/types/document
```

The resolver:
1. Strips `pkg/` -> `sunholo/ailang_parse/types/document`
2. Extracts package name -> `sunholo/ailang_parse`
3. Finds the package directory from lock file -> `/path/to/ailang_parse/`
4. **Passes export visibility** (correctly uses `MapImportToModulePath` -> `docparse/types/document`)
5. Constructs file path from `parts[2]` = `types/document` -> `types/document.ail`
6. Tries candidates:
   - `/path/to/ailang_parse/src/types/document.ail` — NOT FOUND
   - `/path/to/ailang_parse/types/document.ail` — NOT FOUND
7. **Fails** with "module not found"

The file actually exists at `/path/to/ailang_parse/docparse/types/document.ail`.

### Why This Matters

- `sunholo/ailang_parse@0.8.0` is published to the registry with 32 exported modules
- All use `module_prefix = "docparse"`, so all files are under `docparse/`
- No consumer can use this package until the resolver is fixed
- Workaround (path deps) breaks in Docker/CI environments

### Root Cause

In `internal/pkg/loader.go` lines 69-76, `ResolveImport()` constructs the module path from raw import segments without consulting the manifest's `module_prefix`:

```go
// Current (broken):
var modulePath string
if len(parts) == 2 {
    modulePath = "core.ail"
} else {
    modulePath = parts[2] + ".ail"  // "types/document.ail" — missing prefix!
}
```

The manifest IS loaded at line 83, and `MapImportToModulePath()` correctly produces `docparse/types/document`, but the mapped path is never used for file lookup.

---

## Proposed Fix

### Change to `ResolveImport()`

After constructing `modulePath` from `parts[2]`, also construct a prefix-based path using the already-loaded manifest. Add it to the candidate list:

```go
var modulePath string
if len(parts) == 2 {
    modulePath = "core.ail"
} else {
    modulePath = parts[2] + ".ail"
}

// If package has module_prefix, also try prefix-based file path.
// e.g., import "sunholo/ailang_parse/types/document" with module_prefix="docparse"
//   → also try "docparse/types/document.ail"
var prefixModulePath string
if manifest != nil && manifest.Package.ModulePrefix != "" && len(parts) == 3 {
    remapped := manifest.MapImportToModulePath(importPath)
    // remapped = "docparse/types/document"
    prefixModulePath = remapped + ".ail"
}

// Try src/ subdirectory first, then root, then prefix-based paths
candidates := []string{
    filepath.Join(pkgDir, "src", modulePath),
    filepath.Join(pkgDir, modulePath),
}
if prefixModulePath != "" {
    candidates = append(candidates,
        filepath.Join(pkgDir, "src", prefixModulePath),
        filepath.Join(pkgDir, prefixModulePath),
    )
}
```

### Candidate Search Order

For `import pkg/sunholo/ailang_parse/types/document` with `module_prefix = "docparse"`:

| Priority | Path | Rationale |
|----------|------|-----------|
| 1 | `<pkgDir>/src/types/document.ail` | Standard layout (no prefix) |
| 2 | `<pkgDir>/types/document.ail` | Flat layout (no prefix) |
| 3 | `<pkgDir>/src/docparse/types/document.ail` | Standard layout with prefix |
| 4 | `<pkgDir>/docparse/types/document.ail` | Flat layout with prefix (this is the match) |

Non-prefixed paths stay first so packages without module_prefix are unaffected.

### Error Message Improvement

The "module not found" error already lists candidates. With the fix, it will show all 4 candidates tried, making it clear what paths were checked.

### Fix 2: Bare intra-package imports via module_prefix

**Problem**: When a consumed package's module has `import docparse/types/document` (a bare import, not `pkg/`-prefixed), the module loader resolves it against the consumer's working directory, not the package directory.

**Root cause**: In `internal/loader/loader.go`, bare imports fall through to project-relative resolution (`filepath.Join(basePath, canonPath) + ".ail"`). The loader has no awareness of which package owns the `docparse` prefix.

**Fix**:
1. Add `modulePrefixMap` (prefix → pkgName) to `ModuleLoader`
2. Pipeline passes the inverted `currentModulePrefixMap` via `SetModulePrefixMap()`
3. In `Load()`, before project-relative fallback, check if the import's first segment matches a known prefix
4. If it does, remap to canonical `pkg/` path and resolve via `PackageResolver`

```go
// In Load(), new branch before project-relative fallback:
} else if ml.pkgLoader != nil && ml.modulePrefixMap != nil {
    firstSeg := canonPath[:strings.Index(canonPath, "/")]
    for prefix, pkgName := range ml.modulePrefixMap {
        if firstSeg == prefix {
            canonImport := pkgName + strings.TrimPrefix(canonPath, prefix)
            resolvedPath, _ := ml.pkgLoader.ResolveImport(canonImport)
            // ...
        }
    }
}
```

---

## Testing Plan

### Unit Tests (`internal/pkg/loader_test.go`)

1. **Prefix resolution**: Package with `module_prefix = "docparse"`, file at `docparse/services/api.ail` — resolves correctly
2. **No-prefix unchanged**: Package without `module_prefix` — existing behavior unaffected
3. **Prefix with src/**: File at `src/docparse/types/document.ail` — resolves via candidate 3
4. **Self-reference with prefix**: Intra-package import with module_prefix works
5. **Root module (core.ail)**: `pkg/vendor/name` still resolves to `core.ail` regardless of prefix

### Unit Tests (`internal/loader/loader_test.go`)

6. **Bare prefix import**: Bare `docparse/types/document` resolves via prefix map + PackageResolver
7. **Non-matching fallback**: Bare import not matching any prefix falls through to project-relative

### Integration Test (`examples/` or `tests/`)

Create a minimal test package with `module_prefix` and verify round-trip:
- Publish package with prefix-based layout
- Import from another package via `pkg/` path
- Verify symbols are accessible

### Regression

- Run `make test` — all existing tests pass
- Run `make verify-examples` — existing examples unaffected

---

## Files Changed

| File | Change |
|------|--------|
| `internal/pkg/loader.go` | Add prefix-based candidates to `ResolveImport()` |
| `internal/pkg/loader_test.go` | Add tests for prefix resolution (4 tests) |
| `internal/loader/loader.go` | Add `modulePrefixMap` field, `SetModulePrefixMap()`, prefix resolution in `Load()` |
| `internal/loader/loader_test.go` | Add tests for bare prefix import resolution (2 tests) |
| `internal/pipeline/pipeline_module.go` | Wire `currentModulePrefixMap` to module loader |
| `CHANGELOG.md` | Document both fixes |

---

## Out of Scope

- **Changing `MapImportToModulePath` signature** — it already works correctly
- **Module declaration validation changes** — MOD010 already handles prefix correctly
- **Registry changes** — the published package is already correct; only the consumer-side resolver needs fixing
- **New syntax or config** — uses the existing `module_prefix` field as designed
