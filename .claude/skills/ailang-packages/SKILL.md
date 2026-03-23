# AILANG Package Development

Create, validate, and publish AILANG packages with correct conventions. Use when user asks to create a new package, fix package errors, add dependencies, publish to registry, or import packages. Also use when encountering `IMP010`, `LDR001`, `MOD010`, or `export type` errors during package development.

## When to Use This Skill

- User wants to create a new AILANG package
- User encounters package-related errors (IMP010, LDR001, MOD010, type unification across packages)
- User asks about `ailang.toml`, `ailang.lock`, package imports, or package publishing
- User wants to add dependencies between packages
- User asks "how do I import from another package" or "how do I export types"
- Code has `import pkg/` paths that aren't resolving

## Quick Reference

### Create a Package

```bash
ailang init package --name vendor/name
ailang init package --name sunholo/mylib --module-prefix mylib --dep sunholo/config
```

### Package Lifecycle

```bash
ailang lock                    # Resolve dependencies → ailang.lock
ailang check --package .       # Type-check all modules (cross-module resolution)
ailang test --package .        # Run *_test.ail files
ailang publish --dry-run       # Preview registry publication
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

### 2. Intra-package imports use `pkg/` prefix

Modules within the SAME package import siblings via `pkg/` prefix, just like external imports. The loader detects self-references automatically.

```ailang
-- In billing_entitlements/entitlement.ail, importing sibling plan.ail:
import pkg/sunholo/billing_entitlements/plan (Plan, lookupPlan, freePlan)
--     ^^^^ required, even for siblings
```

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
"sunholo/firestore" = { path = "../firestore" }
"sunholo/billing_entitlements" = { path = "../billing-entitlements" }
# OR git deps:
# "sunholo/firestore" = { git = "https://github.com/sunholo-data/ailang-packages", subdir = "packages/firestore", tag = "main" }
# OR registry deps:
# "sunholo/firestore" = "0.1.0"

[effects]
max = ["Net", "FS", "Env"]     # effect ceiling — functions can't exceed this

[metadata]
tags = ["billing", "firestore"]
ai_summary = "Firestore CRUD for billing records"

[stability]
level = "experimental"          # experimental | stable | frozen
```

## Common Error Solutions

See [resources/error_solutions.md](resources/error_solutions.md) for full troubleshooting guide.

| Error | Cause | Fix |
|-------|-------|-----|
| `IMP010: symbol not exported` | Type missing `export` keyword | Add `export type Foo = ...` |
| `LDR001: module not found` | Missing dependency or wrong import path | Add dep to ailang.toml + `ailang lock` |
| `cannot unify type constructor X with TRecord` | Type alias not propagating across packages | Add `export type` to the defining module |
| `package not found in ailang.lock` | Dependencies not resolved | Run `ailang lock` |
| `PAR_HYPHEN_IN_MODULE` | Hyphen in module path | Use underscores: `billing_store` not `billing-store` |

## Workflow

1. **Read** [resources/manifest_reference.md](resources/manifest_reference.md) for ailang.toml details
2. **Read** [resources/error_solutions.md](resources/error_solutions.md) when encountering errors
3. **Run** `scripts/validate_package.sh` after creating a package
4. **Follow** the critical rules above — they prevent the most common failures
