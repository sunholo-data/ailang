# Package Error Solutions

## IMP010: symbol not exported

```
IMP010: symbol 'Proposal' not exported by 'pkg/sunholo/billing_proposals/proposal'
```

**Cause**: The type or function exists but isn't marked as `export`.

**Fix**: Add `export` keyword to the type or function definition in the source package:

```ailang
-- BEFORE (not visible to other packages)
type Proposal = { ... }

-- AFTER (visible to other packages)
export type Proposal = { ... }
```

This applies to ALL types used cross-package: record types, ADTs, and type aliases.

---

## LDR001: module not found

```
LDR001: module not found: sunholo/billing_entitlements/plan
```

**Cause 1**: Missing dependency in ailang.toml

**Fix**: Add the package to `[dependencies]` and run `ailang lock`:

```toml
[dependencies]
"sunholo/billing_entitlements" = { path = "../billing-entitlements" }
```

```bash
ailang lock
```

**Cause 2**: Import path uses bare module name (no prefix)

**Fix**: Use `./` for intra-package siblings or `pkg/` for external dependencies:

```ailang
-- WRONG (bare path — loader can't resolve)
import sunholo/billing_entitlements/plan (Plan)

-- CORRECT: sibling in same package
import ./plan (Plan)

-- ALSO CORRECT: explicit full path (works for both self and external)
import pkg/sunholo/billing_entitlements/plan (Plan)
```

---

## cannot unify type constructor X with TRecord

```
cannot unify type constructor Customer with *types.TRecord
```

**Cause**: A record type alias defined in Package A isn't propagating its structural definition to Package C when passing through Package B's function signatures.

**Fix**: Ensure the type has `export type` in the defining module:

```ailang
-- In customers_repo.ail
export type Customer = { name: string, email: string, ... }
```

If the error persists after adding `export type`, verify AILANG is rebuilt from latest dev (the transitive alias propagation fix is required).

---

## PAR_HYPHEN_IN_MODULE / PAR_HYPHEN_IN_IMPORT

```
PAR_HYPHEN_IN_MODULE: hyphens in module paths are parsed as subtraction
```

**Cause**: Module or import path contains a hyphen.

**Fix**: Use underscores in all module/import paths. Directory names can keep hyphens:

```
Directory:  packages/billing-store/       (hyphens OK)
Module:     module sunholo/billing_store  (underscores required)
Import:     import pkg/sunholo/billing_store/repo (getDoc)
```

---

## package not found in ailang.lock

```
package "sunholo/firestore" not found in ailang.lock
```

**Cause**: Dependencies haven't been resolved.

**Fix**:
```bash
ailang lock
```

If the package is the current package itself (self-reference), ensure you're using AILANG >= 0.9.11 which supports intra-package self-references.

---

## effect ceiling violation

```
effect ceiling violation in package sunholo/mylib: effects [FS] not in max [IO]
```

**Cause**: A function uses an effect not listed in `[effects].max`.

**Fix**: Add the missing effect to the ceiling:

```toml
[effects]
max = ["IO", "FS"]
```

---

## MOD010: module declaration doesn't match canonical path

```
MOD010: module declaration 'sunholo/firestore/client' doesn't match canonical path '...'
```

**In `ailang check --package` mode**: This is automatically relaxed (warning only). The manifest validates module names instead.

**In single-file mode**: Either:
- Use `AILANG_RELAX_MODULES=1` environment variable
- Use `--relax-modules` flag
- Move the file to match the declared module path
