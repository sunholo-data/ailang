# M-DX-RELIMPORT: Local Import Syntax for Intra-Package Modules

**Status**: Approved
**Priority**: P1 (DX — every multi-module package hits this)
**Estimated**: 1 day
**Dependencies**: Package self-reference loader (already implemented)
**Author**: Mark + Claude
**Created**: 2026-03-23

---

## Problem

AILANG requires the full `pkg/vendor/name/module` path for intra-package imports — the syntax does not make locality explicit at the point of use:

```ailang
-- These look identical but mean completely different things:
import pkg/sunholo/billing_entitlements/plan (Plan)       -- sibling module (local)
import pkg/sunholo/firestore/client (getDoc)               -- external dependency (remote)
```

### Axiom Analysis

- **A8 (Syntax Is a Liability)**: The full `pkg/sunholo/billing_entitlements/plan` path repeats the package name in every import within the same package. The package prefix is already implied by the module declaration, the package root, and the file's location.
- **A7 (Machines Are Primary Readers)**: `pkg/` currently conflates self-package and external-package imports syntactically. This hides locality at the source level. Locality is important for AI reasoning about refactors, dependency surfaces, and safe modification scope.

---

## Solution: `./` Local Import Syntax

Add `./` prefix for intra-package module imports. This creates a clean three-way distinction:

```ailang
import ./plan (Plan, lookupPlan)              -- local: same package
import pkg/sunholo/firestore/client (getDoc)  -- external: different package
import std/result (Ok, Err)                   -- stdlib: bundled
```

### Canonical Rule

If current module is `a/b/c`, then `import ./d` means `import a/b/d` within the same package.

`./` is relative to the **current module's canonical path prefix in module-space**, not to filesystem paths.

### Supported Forms (v1)

```ailang
import ./plan (Plan)           -- sibling module
import ./sub/bar (doThing)     -- child directory module
```

**Not supported in v1**: `../` (parent traversal). Deferred until evidence shows `./` is insufficient.

---

## Semantic Invariants

1. **Intra-package only**: `./` imports must resolve to a module in the same package.
2. **Normalized before elaboration**: After parsing, the compiler resolves `./` to the canonical module path. All downstream phases (typechecking, elaboration, codegen) work only with canonical identities.
3. **Cannot escape package boundary**: Resolution must never cross package boundaries, even if `../` is later added.
4. **Canonical paths in all metadata**: Interfaces, hashes, lockfiles, and diagnostics always use canonical module paths. No `./` in any generated artifact.
5. **`pkg/` self-imports remain valid**: Backward compatibility preserved during transition.

---

## Implementation

### Parser (`internal/parser/parser_file.go`)

Detect `./` prefix and record it as a local import:

```go
if strings.HasPrefix(importPath, "./") {
    imp.IsRelative = true
    imp.RelativePath = strings.TrimPrefix(importPath, "./")
}
```

### Elaboration / Normalization

Resolve relative imports against the importing module's canonical module prefix within the same package, then pass the normalized module path to the existing loader.

```go
// If current module is "sunholo/billing_entitlements/entitlement"
// and import is "./plan"
// → canonical path is "sunholo/billing_entitlements/plan"

func normalizeRelativeImport(currentModulePath, relativePath string) string {
    // Strip last segment (current module name) to get prefix
    lastSlash := strings.LastIndex(currentModulePath, "/")
    if lastSlash == -1 {
        return relativePath
    }
    prefix := currentModulePath[:lastSlash]
    return prefix + "/" + relativePath
}
```

### Loader

After normalization, the resolved canonical path is loaded through the existing package loader (self-reference support already implemented). No loader changes needed.

### Interface / Hashing

Interfaces always emit canonical paths. `ailang iface` normalizes `./` to canonical paths. No `./` appears in any interface, hash, or lockfile.

---

## Style Guidance

```
Within a package:
  - PREFER ./... for intra-package imports
  - USE pkg/... only for external packages

ailang check --package may warn on self-package pkg/... imports and offer autofix.
```

---

## Examples

### Before (current workaround)

```ailang
module sunholo/billing_entitlements/entitlement

import std/result (Ok, Err)
import pkg/sunholo/billing_entitlements/plan (Plan, lookupPlan, freePlan)
```

### After (proposed)

```ailang
module sunholo/billing_entitlements/entitlement

import std/result (Ok, Err)
import ./plan (Plan, lookupPlan, freePlan)
```

### Child directory

```ailang
module sunholo/docparse/services/api_server

import ./xml_helpers (parseXml)           -- same directory
import ./utils/validation (validateInput) -- child directory
```

---

## Migration

### Phase 1: Add `./` support (backward compatible)

- Parser accepts `./` imports alongside `pkg/` and `std/`
- Elaborator resolves to canonical path before typechecking
- Existing `pkg/` self-reference imports continue to work

### Phase 2: Lint recommendation

- `ailang check --package` warns when `pkg/` is used for intra-package imports
- Suggests `./` alternative with autofix
- No breaking change — both forms work

### Phase 3: Update all packages

- Convert existing `pkg/` self-imports in ailang-packages billing suite
- Update skill documentation and teaching prompts

---

## Decisions

| Decision | Resolution |
|----------|-----------|
| Syntax | `./module` for intra-package siblings |
| Semantics | Relative to module namespace, not filesystem paths |
| `../` support | Deferred — ship `./` first |
| Backward compatibility | Full — `pkg/` self-imports remain valid |
| Interface normalization | Mandatory — only canonical paths in metadata |
| Lint guidance | Prefer `./` for intra-package, `pkg/` for external only |
