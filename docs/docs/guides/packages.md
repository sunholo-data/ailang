---
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# Package System

AILANG's package system provides deterministic dependency resolution, explicit module exports, effect ceilings, atomic lockfiles, and signature-aware upgrade cascades. Published package pages also render each package's `AGENT.md` guide for AI consumers.

![Package Lock & Cascade](/img/diagrams/package-lock-cascade.svg)

## Quick Start

```bash
ailang init package --name myorg/mylib
ailang init package --name sunholo/myapp --module-prefix myapp  # keep existing module paths
ailang add --path ../shared-utils
ailang lock                                                     # atomic ailang.lock write
ailang lock --check                                             # CI drift check (read-only)
ailang check --package .
ailang test --package .
ailang pkg docs sunholo/auth                                    # read package AGENT.md
ailang publish --dry-run && ailang publish
```

## Manifest (`ailang.toml`) & Nested Packages

```toml
[package]
name = "sunholo/docparse"
version = "0.3.0"
edition = "1"
ailang = ">=0.64.0"
description = "Parse documents into structured blocks"
license = "MIT"

[exports]
modules = ["sunholo/docparse/parser", "sunholo/docparse/types"]

[dependencies]
"sunholo/json" = { path = "../json" }

[effects]
max = ["IO", "FS"]

[stability]
level = "experimental"   # experimental | stable | frozen

[release]
kind = "feature"         # security | fix | feature | breaking
```

- **Module prefix (`module_prefix`):** Single-segment prefix mapping existing `module myapp/...` declarations to `pkg/sunholo/myapp/...` without renaming files.
- **Exports (`[exports].modules`):** Modules outside `[exports]` are package-private.
- **Nested `ailang.toml` packages:** Any subdirectory containing its own `ailang.toml` is treated as an independent package — excluded from the parent package's source discovery, compilation, content hash, and tarball.
- **Effect ceiling (`[effects].max`):** Every module in the package must stay within `[effects].max` (including effects referenced on external/stdlib/builtin symbols; `Debug` is exempt as a ghost effect). Empty `max = []` enforces a pure package.

## Commands (`[bin]`)

Packages can install CLI shims onto `~/.ailang/bin`:

```toml
[bin]
eparse   = "cli"
docparse = { module = "docparse/main", entry = "main", caps = "IO,FS,Env,AI", run_flags = ["--max-recursion-depth", "50000"] }
```

```bash
ailang install sunholo/email            # library + `eparse` shim on PATH
ailang install --path packages/email    # local dev shim
ailang bin list                         # inspect installed shims
```

## Importing from Packages & `export import` Status

```ailang parse
import std/io (println)                       -- Stdlib
import ./types (Profile)                      -- Intra-package sibling (works before ailang lock)
import myproject/parser (parse)               -- Canonical local module
import pkg/sunholo/json/parser (parseJson)    -- External package (resolved via ailang.lock)
```

:::note `export import` re-export facades
`export import ...` is **not yet supported** and fails with `PAR_EXPORT_REQUIRES_FUNC` (tracked in `#2475`). Export wrapper functions or import directly from the defining module.
:::

## Atomic & Portable Lockfile (`ailang.lock`)

`ailang lock` writes `ailang.lock` **atomically** (staging bytes to a sibling temp file, `fsync`ing, and renaming over `ailang.lock`) so concurrent agents sharing a checkout never observe a truncated JSON file.

- **Portable paths:** Path dependencies are recorded relative to `ailang.toml` with forward slashes (never absolute paths).
- **Deterministic metadata:** Unchanged resolutions preserve `generated_at`, `generator`, and `ailang_version`, keeping `ailang.lock` byte-identical across toolchain upgrades.
- **CI drift check:** `ailang lock --check` exits non-zero if `ailang.lock` is out of date without modifying it.
- **Flat resolution:** One version per `vendor/name` (priority: `path > git > registry`). Version conflicts abort loudly with actionable resolution suggestions.

## Dual Hash & v2 Contract Signatures

| Identity | Changes When | Purpose |
|---|---|---|
| **Content hash** | Any `.ail` source file changes | Reproducible builds & cache verification |
| **Interface hash (v1)** | Exports or effect rows change | Lockfile stability on internal refactors |
| **Interface hash (v2)** + `interface_signatures` | Export types, class constraints (`Eq[a]`, `Num[a]`), `literals`, or `requires`/`ensures` contracts change | Classifies publishes as **A** (internal), **B** (additive), or **C** (breaking/contract tightening) and notifies dependents with `affected_exports` |

## Package Docs & Registry Commands

| Command | Description |
|---|---|
| `ailang pkg docs vendor/name` *(or `ailang pkg-docs`)* | Display `AGENT.md` + `pkg/` import paths |
| `ailang search "query"` / `--tag gcp` | Search registry packages |
| `ailang install vendor/name@0.1.0` | Pin exact version in `ailang.toml` (no semver ranges) |
| `ailang pkg quality .` | Run local + server-equivalent publish gates (`PUBnnn`) |
| `ailang publish --dry-run` / `ailang publish` | Validate and upload tarball (immutable versions) |
| `ailang tree` | Display resolved dependency tree |
| `ailang pkg affected-by vendor/name` | List downstream packages depending on `vendor/name` |

<details>
<summary>Registry environment variables & cascade config</summary>

| Variable | Purpose |
|---|---|
| `AILANG_REGISTRY` | Registry URL (default `https://storage.googleapis.com/ailang-registry`) |
| `AILANG_REGISTRY_VALIDATOR` | Validator service URL |
| `AILANG_REGISTRY_API_KEY` | Publish API key (or pass `--key-file PATH`) |
| `AILANG_CLOUD_PROJECT` | GCP project for `*-cascade` Pub/Sub topic (e.g. `ailang-multivac-dev`) |
| `AILANG_TOPIC_PREFIX` | Cascade topic prefix (`ailang-dev`, `ailang-test`, `ailang`) |

</details>

## Common Pitfalls & Troubleshooting

- **Hyphens in module names:** Directory names may use hyphens (`billing-store/`), but AILANG module paths must use underscores (`sunholo/billing_store/...`) to avoid `PAR_HYPHEN_IN_MODULE`.
- **Missing `export type`:** Any ADT or record type crossing a package boundary must be declared `export type`.
- **Prelude types:** `Result`/`Ok`/`Err` and `Option`/`Some`/`None` are in the prelude outside `std/` (set `ailang = ">=0.64.0"` in `ailang.toml` for modern packages; helpers like `mapErr` still require `import std/result`).

| Error | Cause | Fix |
|---|---|---|
| `409 Conflict` | Version already published | Bump `[package].version` + `CHANGELOG.md` |
| `effect ceiling violation` | Used effect missing from `[effects].max` | Add effect to `[effects].max` (`Debug` exempt) |
| `IMP010: symbol not exported` | Missing `export` on type/func | Add `export` in defining module |
| `LDR001: module not found` | Missing dep or stale lock | Run `ailang add` + `ailang lock` |

## See Also

- [Publishing Your Package](./package-publishing.md) — quality gates, v2 identity, nested packages, scoped keys
- [Autonomous Package Updates](./autonomous-package-updates.md) — cloud upgrade cascades
- [Extension Packages](./extension-packages.md) — compile-time plugin registries (`experimental`)
