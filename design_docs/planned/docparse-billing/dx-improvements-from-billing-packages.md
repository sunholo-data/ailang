# DX Improvements Discovered During Billing Package Creation

**Created**: 2026-03-23
**Context**: First production AILANG application packages (billing for DocParse)

These issues were discovered while creating 6 billing packages with 26 `.ail` files across `ailang-packages`. They represent real friction points for anyone building AILANG packages.

---

## 1. Hyphens in module names are silently parsed as subtraction — **FIXED**

**Severity**: High (confusing parse errors) — **FIXED: clear error with suggestion**

**Problem**: `module sunholo/billing-entitlements/plan` parses as `sunholo/billing` minus `entitlements/plan`, producing `PAR_NO_PREFIX_PARSE` errors that don't mention the hyphen.

**Fix applied**: Parser now detects hyphens in module and import paths and emits specific errors:
- `PAR_HYPHEN_IN_MODULE`: "hyphens in module paths are parsed as subtraction" with suggested underscore fix
- `PAR_HYPHEN_IN_IMPORT`: Same for import paths
- Changes in `internal/parser/parser_file.go`, tests in `internal/parser/module_test.go`

**Impact**: Every new package author will hit this. The existing `http-helpers` package already works around it (`http_helpers` in code).

---

## 2. `ailang check` cannot validate cross-module types within a package

**Severity**: Medium (blocks CI validation) — **Design doc: `m-dx-package-check.md`**

**Problem**: `ailang check capability_check.ail` fails because `Entitlements` (defined in `entitlement.ail`) is a nominal type that can't be resolved in standalone file check. This means:
- No way to type-check a package as a unit
- CI can only validate self-contained files
- Cross-module type references only work at runtime through package imports

**Suggested fix**: Add `ailang check --package .` or `ailang check --dir .` that loads all `.ail` files from a package directory (reading `ailang.toml` for module list) and resolves types across them.

**Workaround**: Only validate self-contained files in CI. Accept that cross-module type errors will only surface at package-import time.

**Design doc**: See `design_docs/planned/m-dx-package-check.md`

---

## 3. No `ailang init package` for packages with dependencies — **FIXED**

**Severity**: Low (manual toml editing works) — **FIXED: `--dep` flag added**

**Problem**: `ailang init package --name sunholo/billing_store` creates a basic `ailang.toml` but doesn't support `--deps` or `--add-dep` to pre-populate the `[dependencies]` section. Package authors must manually edit toml.

**Fix applied**: Added repeatable `--dep` flag to `ailang init package`:
```
ailang init package --name sunholo/billing_store --dep sunholo/config --dep sunholo/gcp_auth
ailang init package --dep sunholo/http_helpers@0.2.0
```
Changes in `cmd/ailang/pkg_init.go`.

---

## 4. `Ok` and `Err` require explicit import from std/result — **WON'T FIX**

**Severity**: Low (surprising for newcomers) — **WON'T FIX: intentionally lean prelude**

**Problem**: `Ok` and `Err` constructors are not in the auto-imported prelude. Every file that uses `Result` needs `import std/result (Ok, Err)`. Since nearly every non-trivial function returns `Result`, this import appears in almost every file.

**Current state**: Comparison operators (`<`, `>`, `==`) are auto-imported from prelude, but `Ok`/`Err` are not.

**Decision**: The prelude is intentionally lean. Explicit imports make dependencies clear and keep the language predictable. The one-line `import std/result (Ok, Err)` is acceptable boilerplate.

---

## 5. No package-level test runner

**Severity**: Medium (no standard test workflow) — **Design doc: `m-dx-package-test.md`**

**Problem**: No `ailang test` equivalent for packages. Self-contained test files can be run individually, but there's no way to run all tests in a package or validate that exports match `ailang.toml`.

**Suggested fix**: `ailang test --package .` that:
- Reads `ailang.toml` for module list
- Finds `*_test.ail` files
- Runs them with cross-module resolution
- Reports coverage per module

**Design doc**: See `design_docs/planned/m-dx-package-test.md`

---

## 6. Firestore and other GCP services require manual REST API wrapping

**Severity**: Medium (lots of boilerplate) — **FIXED: created `sunholo/firestore` package**

**Problem**: No Firestore builtin — every package that needs Firestore writes ~100 lines of REST API boilerplate (build URL, get ADC token, construct headers, make request, decode response, extract fields from Firestore's wrapped value format). The `billing-store/firestore.ail` module is pure infrastructure.

**Fix applied**: Created `sunholo/firestore` package with 3 modules:
- `client` — `getDoc`, `setDoc`, `deleteDoc`, `getSubDoc`, `setSubDoc`, `docExists`
- `fields` — `stringVal`/`intVal`/`boolVal`/`floatVal`/`timestampVal`/`arrayVal`/`mapVal` + decode helpers
- `query` — `runQuery` with `where` filters and `orderBy` clauses

**Also fixed**: Extended `httpRequest` method whitelist from GET/POST to GET/POST/PUT/PATCH/DELETE/HEAD (required for Firestore PATCH and DELETE operations). Change in `internal/effects/net.go:444`.

---

## 7. Linter auto-adds `ailang = ">=0.9.5"` to ailang.toml

**Severity**: Info (positive DX, just noting it)

**Observed**: The linter automatically added version constraints to all new ailang.toml files. This is helpful behavior.

---

## Summary

| Issue | Severity | Status | Design Doc |
|-------|----------|--------|------------|
| ~~Hyphens in module names~~ | ~~High~~ | **FIXED** (PAR_HYPHEN_IN_MODULE/IMPORT errors) | — |
| No package-level type check | Medium | Planned | `m-dx-package-check.md` |
| ~~Ok/Err not in prelude~~ | ~~Low~~ | **WON'T FIX** (intentionally lean prelude) | — |
| No package test runner | Medium | Planned | `m-dx-package-test.md` |
| ~~No Firestore builtin~~ | ~~Medium~~ | **FIXED** (`sunholo/firestore` package) | — |
| ~~init package lacks --dep~~ | ~~Low~~ | **FIXED** (`--dep` flag added) | — |
