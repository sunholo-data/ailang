# AILANG Package Development

Create, validate, and publish AILANG packages with correct conventions. Use when user asks to create a new package, fix package errors, add dependencies, publish to registry, or import packages. Also use when encountering `IMP010`, `LDR001`, `MOD010`, or `export type` errors during package development.

## When to Use This Skill

- User wants to create a new AILANG package
- User encounters package-related errors (IMP010, LDR001, MOD010, type unification across packages)
- User asks about `ailang.toml`, `ailang.lock`, package imports, or package publishing
- User wants to add dependencies between packages
- User asks "how do I import from another package" or "how do I export types"
- Code has `import pkg/` or `import ./` paths that aren't resolving

## Quick Reference

### Create a Package

```bash
ailang init package --name vendor/name
ailang init package --name sunholo/mylib --module-prefix mylib --dep sunholo/config
```

### Install Dependencies

```bash
ailang install sunholo/auth           # Install latest version (resolves from registry)
ailang install sunholo/auth@latest    # Same as above
ailang install sunholo/auth@0.1.0     # Install exact version
```

`@latest` resolves once and writes the **exact version** to `ailang.toml`. Semver ranges (`^`, `~`, `>=`) are not supported — AILANG requires exact versions in manifests for determinism.

### Package Lifecycle

```bash
ailang lock                    # Resolve dependencies → ailang.lock
ailang check --package .       # Type-check all modules (cross-module resolution)
ailang test --package .        # Run *_test.ail files
ailang publish --dry-run       # Preview registry publication (rewrites path deps → registry versions)
ailang publish                 # Publish to registry
```

## Critical Rules (learned from production)

### 1. Module names use underscores, never hyphens

Hyphens parse as subtraction. Directory names can have hyphens, but module paths must use underscores.

```
Directory:  packages/billing-store/
Module:     module sunholo/billing_store/customers_repo
Import:     import pkg/sunholo/billing_store/customers_repo (getCustomer)
```

### 2. Use `./` for intra-package sibling imports

Three-way import distinction — locality is explicit at the point of use:

```ailang
import ./plan (Plan, lookupPlan)              -- LOCAL: sibling in same package
import pkg/sunholo/firestore/client (getDoc)  -- EXTERNAL: different package
import std/result (Ok, Err)                   -- STDLIB: bundled
```

`./` resolves in **module namespace** (not filesystem): if current module is `sunholo/billing_entitlements/entitlement`, then `./plan` normalizes to `sunholo/billing_entitlements/plan`.

```ailang
import ./plan (Plan, lookupPlan, freePlan)    -- sibling
import ./sub/helpers (validate)               -- child directory
```

`pkg/` self-imports also work (backward compatible) but `./` is preferred.

### 3. Cross-package types MUST use `export type`

Types used by other packages need `export type`, not just `type`. Without it, consumers get `IMP010: symbol not exported`.

```ailang
-- WRONG: only visible within this module
type Customer = { name: string, email: string }

-- CORRECT: visible to importing packages
export type Customer = { name: string, email: string }
```

This applies to record types AND ADTs:
```ailang
export type ProposalStatus = PendingApproval | AwaitingPayment | Approved
export type RequestedBy = Human | Agent(string)
```

### 4. Always import `Ok` and `Err` explicitly

They are NOT in the auto-imported prelude:
```ailang
import std/result (Ok, Err)
```

### 5. Package name is vendor/name format (exactly 2 segments)

```toml
[package]
name = "sunholo/firestore"    # CORRECT
name = "firestore"            # WRONG — single segment
name = "sunholo/billing/store" # WRONG — 3 segments
```

### 6. Use `module_prefix` for existing apps adopting packages

If your project uses `module myapp/...` but you want to publish as `sunholo/myapp`:

```toml
[package]
name = "sunholo/myapp"
module_prefix = "myapp"

[exports]
modules = ["myapp/services/api", "myapp/handlers/parse"]
```

Zero source changes needed — existing `module myapp/...` declarations work as-is.

### 7. Publishing rewrites path deps automatically

`ailang publish` automatically rewrites path deps (`{ path = "../firestore" }`) to registry version strings (`"0.1.0"`) in the tarball. Your local `ailang.toml` is restored after. You don't need to change deps manually before publishing.

**Publish in dependency order**: packages with no deps first, then packages that depend on already-published packages.

## ailang.toml Reference

See [resources/manifest_reference.md](resources/manifest_reference.md) for full field documentation.

```toml
[package]
name = "sunholo/billing_store"
version = "0.2.0"
edition = "1"
ailang = ">=0.9.5"              # optional: minimum AILANG version
module_prefix = "myapp"         # optional: for existing apps
description = "Firestore CRUD for billing records"
license = "Apache-2.0"

[exports]
modules = [
  "sunholo/billing_store/customers_repo",
  "sunholo/billing_store/subscriptions_repo"
]

[dependencies]
# Path deps (local development):
"sunholo/firestore" = { path = "../firestore" }
# Git deps (version pinned):
# "sunholo/firestore" = { git = "https://github.com/sunholo-data/ailang-packages", subdir = "packages/firestore", tag = "main" }
# Registry deps (published packages — use exact versions only, no ranges):
# "sunholo/firestore" = "0.1.0"
# Install latest: ailang install sunholo/firestore

[effects]
max = ["Net", "FS", "Env"]     # effect ceiling — functions can't exceed this

[metadata]
tags = ["billing", "firestore"]
ai_summary = "Firestore CRUD for billing records"

[stability]
level = "experimental"          # experimental | stable | frozen
```

### Lock File Portability

`ailang.lock` is portable — it does **not** contain absolute paths. Registry and git package paths are resolved at runtime from the local cache (`~/.ailang/cache/`).

**Docker workflow:**
```dockerfile
COPY ailang.toml ailang.lock .
RUN ailang install sunholo/auth    # Populates cache from lock file versions
```

**Key facts:**
- Lock file stores: name, version, content hash, source type, git URL/rev
- Lock file does NOT store: absolute cache paths
- `ailang install` populates the cache; `ailang lock` resolves + downloads
- Old lock files with stored paths still work (backward compatible)

### Version Conflict Detection

The resolver enforces **flat dependencies** — one version per package name. If a transitive dependency requires a different version than the root manifest pins, `ailang lock` fails with a structured error:

```
version conflict: sunholo/firestore
  root requires: 0.2.0
  already resolved: 0.1.0
  transitive requires: 0.1.0 (via sunholo/billing_store)

resolution aborted

suggestion:
  - republish sunholo/billing_store against sunholo/firestore@0.2.0
  - or change root dependency to sunholo/firestore@0.1.0 explicitly
```

**Resolution rules:**
- Direct dependencies in the root `ailang.toml` are **authoritative**
- Transitive dependencies must match the direct pin exactly
- If they conflict, the resolver fails (never silently downgrades)
- Same version from multiple sources is fine (silently deduplicated)

**How to fix version conflicts:**
1. Republish the transitive package against the version you need
2. Or change your root `ailang.toml` to match the transitive version
3. The error message tells you exactly which package introduced the conflict

## Common Error Solutions

See [resources/error_solutions.md](resources/error_solutions.md) for full troubleshooting guide.

| Error | Cause | Fix |
|-------|-------|-----|
| `version conflict: pkg` | Direct and transitive deps disagree on version | Republish transitive dep or change root pin (see error message) |
| `IMP010: symbol not exported` | Type missing `export` keyword | Add `export type Foo = ...` |
| `LDR001: module not found` | Missing dependency or wrong import path | Add dep to ailang.toml + `ailang lock` |
| `cannot unify type constructor X with TRecord` | Type alias not exported across packages | Add `export type` to the defining module |
| `package not found in ailang.lock` | Dependencies not resolved | Run `ailang lock` |
| `PAR_HYPHEN_IN_MODULE` | Hyphen in module path | Use underscores: `billing_store` not `billing-store` |
| `Key has already been defined` | Duplicate dep entry in ailang.toml | Remove duplicate, keep one format |

## Publishing Checklist

1. All modules pass: `ailang check --package .`
2. Dependencies are listed in `[dependencies]` (path deps OK — publish rewrites them)
3. Exported types have `export type`
4. `[exports].modules` lists all public modules
5. `[effects].max` includes all effects used
6. `AGENT.md` exists with usage guide
7. Publish in dependency order (leaf packages first)

## Registry Validator

The registry validator (`/version` endpoint shows deployed version):
- Runs `ailang check --package .` on uploaded tarballs
- Rewrites any remaining path deps to registry versions (safety net for older clients)
- Runs `ailang lock` to download deps from registry before checking
- Validates compilation, effect ceilings, and contracts

## Workflow

1. **Read** [resources/manifest_reference.md](resources/manifest_reference.md) for ailang.toml details
2. **Read** [resources/error_solutions.md](resources/error_solutions.md) when encountering errors
3. **Run** `scripts/validate_package.sh` after creating a package
4. **Follow** the critical rules above — they prevent the most common failures
