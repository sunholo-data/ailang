# M-DX-APP: Application Package Adoption — Module Prefix Mismatch

**Status**: Implemented (v0.9.11)
**Priority**: P1 (blocks DocParse and all existing apps from adopting packages)
**Estimated**: 2-3 days
**Dependencies**: None
**Author**: Mark + Claude
**Created**: 2026-03-23
**Triggered by**: DocParse feedback (msg `2527f192`)

---

## Problem Statement

Existing AILANG applications cannot adopt the package system without renaming all their modules.

DocParse uses modules like:
```ailang
module docparse/services/api_server
module docparse/middleware/auth
module docparse/handlers/parse
```

The package system requires `ailang.toml` with a **vendor/name** format:
```toml
[package]
name = "sunholo/docparse"
```

This means all exported modules must be prefixed with `sunholo/docparse/`:
```ailang
module sunholo/docparse/services/api_server   -- required by package system
module docparse/services/api_server            -- what exists today
```

The mismatch means DocParse must rename **every module declaration and every import** across its entire codebase to adopt `ailang.toml`. This is a breaking change that blocks adoption of the package ecosystem (dependencies, lockfiles, registry).

### Scale of the Problem

This affects **every existing AILANG application**, not just DocParse. Any project that started without `ailang.toml` used freeform module paths. Adding package management retroactively forces a mass rename.

---

## Root Cause

The package system enforces that module declarations must be prefixed with the package name (enforced in `internal/pkg/manifest.go:164-169`):

```go
for _, mod := range m.Exports.Modules {
    if !strings.HasPrefix(mod, m.Package.Name+"/") && mod != m.Package.Name {
        return fmt.Errorf("exported module %q must start with package name %q", mod, m.Package.Name)
    }
}
```

This is correct for **libraries** (published packages need namespaced modules to avoid collisions). But **applications** — top-level projects that aren't imported by others — don't benefit from vendor prefixing and are harmed by the rename cost.

---

## Proposed Solution: `module_prefix` Mapping

Add an optional `module_prefix` field to `[package]` that maps existing module paths to the package namespace without requiring source changes.

### ailang.toml

```toml
[package]
name = "sunholo/docparse"
module_prefix = "docparse"    # maps docparse/* modules to sunholo/docparse/*
```

**Behavior**:
- When `module_prefix` is set, the export validation accepts modules starting with either `sunholo/docparse/` OR `docparse/`
- When resolving `import pkg/sunholo/docparse/services/api_server`, the loader maps it to the file containing `module docparse/services/api_server`
- The source files are **not modified** — the mapping is a toml-level alias
- When publishing to registry, the mapping is stored in the manifest so consumers can import via the canonical `pkg/sunholo/docparse/...` path

### Example

DocParse's `ailang.toml`:
```toml
[package]
name = "sunholo/docparse"
module_prefix = "docparse"

[exports]
modules = [
  "docparse/services/api_server",
  "docparse/middleware/auth",
  "docparse/handlers/parse"
]
```

A consumer importing DocParse:
```ailang
-- Both resolve to the same module:
import pkg/sunholo/docparse/services/api_server (startServer)
```

Internally, the loader strips `sunholo/` from the import path if `module_prefix = "docparse"` is set.

---

## Implementation Plan

### Phase 1: Manifest Support

**File**: `internal/pkg/manifest.go`

1. Add `ModulePrefix string` field to `PackageConfig` struct
2. Update `Validate()` to accept modules starting with either:
   - `m.Package.Name + "/"` (current behavior)
   - `m.ModulePrefix + "/"` (new, when module_prefix is set)
3. Validate that `module_prefix` doesn't contain `/` (must be a single segment or simple path like `docparse`)

```go
type PackageConfig struct {
    Name         string `toml:"name"`
    Version      string `toml:"version"`
    Edition      string `toml:"edition"`
    ModulePrefix string `toml:"module_prefix"` // optional alias
    // ...
}
```

### Phase 2: Loader Resolution

**File**: `internal/pkg/loader.go`

Update `ResolveImport` to handle the prefix mapping:

```go
func (pl *PackageLoader) ResolveImport(importPath string) (string, error) {
    // Extract package name (first two segments)
    parts := strings.SplitN(importPath, "/", 3)
    pkgName := parts[0] + "/" + parts[1]

    pkg := pl.packages[pkgName]
    if pkg == nil {
        return "", fmt.Errorf("package %q not found", pkgName)
    }

    // If module_prefix is set, remap the import path
    if pkg.Manifest.Package.ModulePrefix != "" {
        // pkg/sunholo/docparse/services/api_server
        // → docparse/services/api_server (for file lookup)
        remainder := ""
        if len(parts) == 3 {
            remainder = "/" + parts[2]
        }
        modulePath = pkg.Manifest.Package.ModulePrefix + remainder
    }
    // ... rest of resolution
}
```

### Phase 3: MOD010 Relaxation

**File**: `internal/pipeline/pipeline_module.go`

Update the MOD010 check to accept module declarations matching either the canonical package path or the `module_prefix` alias:

```go
// If package has module_prefix, also accept that as valid
if pkg.ModulePrefix != "" && mod.File.Module.Path == pkg.ModulePrefix+"/"+relativePath {
    return nil  // Valid via module_prefix mapping
}
```

### Phase 4: Registry Support

**File**: `internal/pkg/registry.go`

- Store `module_prefix` in published package metadata
- Consumers resolve using canonical `pkg/vendor/name/...` paths
- The prefix mapping is transparent to consumers

---

## Alternative Considered: App-Mode Package Names

Allow single-segment names for applications:

```toml
[package]
name = "docparse"         # no vendor prefix
type = "application"      # opt into relaxed naming
```

**Rejected because**:
- Breaks the registry namespace model (collisions between unrelated `docparse` packages)
- Creates two classes of packages with different rules
- Apps may later become libraries (e.g., DocParse access gate is extracted from the app)
- The `module_prefix` approach solves the problem without fragmenting the namespace

---

## Alternative Considered: Mass Rename Tool

Add `ailang migrate module-prefix --from docparse --to sunholo/docparse`:

**Rejected as the sole solution because**:
- Destructive (modifies every .ail file)
- Doesn't preserve git blame
- Forces downstream consumers to update all imports
- Still valuable as a secondary tool, but shouldn't be required

Could be offered as an **optional convenience** for projects that want to fully adopt vendor/name prefixes.

---

## Affected Code Paths

| File | Change | Risk |
|------|--------|------|
| `internal/pkg/manifest.go:155-169` | Add module_prefix field, relax export validation | Low — additive |
| `internal/pkg/loader.go:26-79` | Map import paths via module_prefix | Medium — core resolution |
| `internal/pipeline/pipeline_module.go:378-423` | Accept module_prefix in MOD010 check | Low — validation only |
| `internal/pkg/registry.go:94-100` | Store/read module_prefix in metadata | Low — additive |
| `cmd/ailang/pkg_init.go` | Support `--module-prefix` flag | Low |

---

## Testing Strategy

1. **Unit tests** for manifest validation with/without module_prefix
2. **Integration test**: create a package with `module_prefix = "myapp"`, import it from another package via `pkg/vendor/name/module`
3. **Regression test**: existing packages without module_prefix continue to work unchanged
4. **Registry test**: publish a package with module_prefix, install and import it

---

## Migration Path for DocParse

1. Create `ailang.toml` with `module_prefix = "docparse"`:
   ```toml
   [package]
   name = "sunholo/docparse"
   module_prefix = "docparse"
   ```
2. List existing modules in `[exports]` using their current names
3. Run `ailang lock` to resolve dependencies
4. Add package dependencies (e.g., `sunholo/firestore`, `sunholo/billing_store`)
5. **Zero source file changes required**

---

## Open Questions

1. Should `module_prefix` support multi-segment paths (e.g., `"myorg/myapp"`)? Starting with single-segment seems safest.
2. Should we add an `ailang migrate module-prefix` command as a convenience for projects that want to fully normalize?
3. Should `ailang init package` detect existing module declarations and suggest a `module_prefix` automatically?

---

## Recommended Decisions

| Decision | Recommendation |
|----------|---------------|
| Approach | `module_prefix` mapping (no source changes required) |
| Scope | Single-segment prefix in v1 (`docparse`), multi-segment later |
| Backward compatibility | Fully backward compatible — new field is optional |
| Registry | Store prefix in manifest metadata, transparent to consumers |
| Tooling | Optional `ailang migrate` command as convenience, not required |
