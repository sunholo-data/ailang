# M-AILANG-EXT-REGISTRY-GEN: `ailang generate-extension-registry` command

**Status**: IMPLEMENTED
**Target**: v0.17.0
**Priority**: P1 — unblocks motoko package ecosystem
**Estimated**: 2-3 days (~200 LOC)
**Dependencies**: Package system Phase 1+1.5 (complete in v0.9.x), M-AI-PROVIDER-CONFIG pattern (v0.16.0)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Generated file is deterministic — same `ailang.toml` + `ailang.lock` always produces the same output |
| A2: Replayability | +1 | Generated file is committed alongside `ailang.lock`; builds are reproducible without network |
| A3: Effect Legibility | 0 | No effect system changes — pure code generation, no new effects introduced |
| A4: Explicit Authority | +1 | Extensions loaded via explicit `ailang.toml` declaration, not ambient discovery |
| A5: Bounded Verification | +1 | Generated file is a normal AILANG source file — `ailang check --package` verifies it at compile time |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Machine-readable registry declaration in `ailang.toml` replaces hand-written ladder; AI agents can `ailang add motoko-ext-X` |
| A8: Minimal Syntax | +1 | No new syntax; new TOML section is additive |
| A9: Cost Visibility | 0 | No cost model changes |
| A10: Composability | +1 | Extension packages compose via the standard dependency system |
| A11: Structured Failure | +1 | Generator fails with named errors (unknown package, missing `register` export, version mismatch) |
| A12: System Boundary | 0 | No new system boundary crossings |

**Net Score: +8** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Generated output is purely a function of `ailang.toml` + `ailang.lock`
- [x] A3 (Effects): No hidden effects; generated code has explicit effect annotations
- [x] A4 (Authority): No ambient extension loading; all extensions declared explicitly
- [x] A7 (Machines First): Machine-readable config replaces human-maintained code ladder

---

## Problem Statement

**Current State:**

motoko_agent's extension system has a hardcoded `if name == "..."` ladder in `src/core/ext/registry.ail`. Adding an extension requires:
1. Writing the extension in `src/core/ext/<name>/`
2. Manually editing `registry.ail` to add an import and a new branch
3. Submitting a PR to motoko_agent itself

Extensions cannot be versioned independently, cannot be published by third parties, and cannot be added/removed per-project without touching core motoko code.

**Blocker: AILANG has no runtime-resolved imports.** All `import` statements are resolved at compile time. The `ModuleLoader.Load()` path is fully static — there is no `import_symbol(pkg_name, symbol)` builtin. This means the registry cannot load extensions by name at runtime from a config string.

**Impact:**
- motoko contributors who want to ship a custom extension must fork motoko_agent
- Projects with different extension needs (e.g. one project uses exa_search, another uses MCP) must maintain separate forks
- No versioned, published extension ecosystem is possible without this tool

---

## Goals

**Primary Goal:** A `ailang generate-extension-registry` command that reads `[extensions]` from `ailang.toml` and emits a valid, compilable AILANG source file wiring the listed packages into a dispatch function — replacing the hand-written registry ladder.

**Success Metrics:**
1. `ailang add motoko-ext-exa-search@0.4.1` + `ailang generate-extension-registry` = working exa_search extension with no manual code edits
2. Removing a package from `ailang.toml` + regenerating = extension no longer loaded (verified by `ailang check --package`)
3. Generated file passes `ailang check --package` and all motoko_agent tests
4. Generator fails with a clear, named error when a package lacks the required `register_with_config` export

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Where does `[extensions]` live in `ailang.toml`? | Determines whether it's part of `PackageManifest` or a separate file | human | design | high |
| How is the extension's short name derived? | Determines the `if name == "..."` key at runtime | human | design | med |
| What type does `register_with_config` receive? | Determines whether the ABI is motoko-specific or generic | human | design | high |
| Is the generated file committed to the repo? | Determines CI reproducibility model | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] **`[extensions]` in `PackageManifest`** — follows `[ai_provider]` pattern (additive TOML section, `Extensions ExtensionRegistryConfig \`toml:"extensions"\``). Consistent with how `AIProviders []AIProviderSpec` was added in v0.16.0. *(Decided: yes, same pattern)*
- [x] **Short name from package name** — strip the `motoko-ext-` prefix: `motoko-ext-exa-search` → `exa_search`. If the package does not start with `motoko-ext-`, use the full name with `-` replaced by `_`. *(Decided: strip prefix)*
- [x] **`register_with_config` receives `RuntimeConfig` from consumer** — the `[extensions]` section in `ailang.toml` specifies `config_import` and `hooks_import` (the module paths for `RuntimeConfig` and `ExtensionHooks`). This keeps the generator generic and not hard-coded to motoko_agent's internal module paths. *(Decided: config-driven import paths)*
- [x] **Generated file is committed** — same model as `ailang.lock`. Regenerate with `ailang generate-extension-registry` when `ailang.toml` changes. CI checks via `make check_core`. *(Decided: commit generated file)*

---

## Solution Design

### Overview

The generator is a new `ailang generate-extension-registry` subcommand in `cmd/ailang/`. It:
1. Reads `[extensions]` from `ailang.toml`
2. Resolves each package from `ailang.lock` (for the locked module path prefix)
3. Emits a `.ail` file with explicit static imports and an `if name == "..."` dispatch function

No AILANG language changes are required. The output is an ordinary AILANG source file.

### TOML Schema (`[extensions]` section)

```toml
[extensions]
# Packages that provide extensions via the motoko-ext-* convention
packages = [
  "motoko-ext-compaction@0.2.0",
  "motoko-ext-exa-search@0.4.1",
]

# Module path for the RuntimeConfig type used by register_with_config
config_import = "src/core/config.RuntimeConfig"

# Module path for the ExtensionHooks type returned by register_with_config
hooks_import = "src/core/ext/types.ExtensionHooks"

# Where to write the generated file (relative to ailang.toml)
output = "src/core/ext/registry_generated.ail"

# Module name for the generated file's 'module' declaration
module_name = "src/core/ext/registry_generated"
```

All keys except `packages` have defaults (see Files section). `config_import` and `hooks_import` default to the `motoko-ext-abi` package exports if `motoko-ext-abi` is in `[dependencies]`; otherwise they are required.

### Generated File Shape

```ailang
-- GENERATED by ailang generate-extension-registry
-- Source: ailang.toml [extensions]
-- Do not edit by hand. Regenerate with: ailang generate-extension-registry

module src/core/ext/registry_generated

import motoko-ext-compaction/register (register_with_config) as register_compaction
import motoko-ext-exa-search/register (register_with_config) as register_exa_search
import src/core/config (RuntimeConfig)
import src/core/ext/types (ExtensionHooks)
import std/option (Option, Some, None)

export func resolve(name: string, cfg: RuntimeConfig) -> Option[ExtensionHooks] {
  if name == "compaction" then Some(register_compaction(cfg))
  else if name == "exa_search" then Some(register_exa_search(cfg))
  else None
}
```

### Short Name Derivation

| Package name | Short name |
|---|---|
| `motoko-ext-compaction` | `compaction` |
| `motoko-ext-exa-search` | `exa_search` |
| `motoko-ext-mcp` | `mcp` |
| `my-custom-ext` | `my_custom_ext` (no prefix to strip; `-` → `_`) |

### Extension Package Convention (`motoko-ext-*`)

Every extension package must:
1. Export `register_with_config` from a module named `<pkg-module-prefix>/register`
2. Declare `motoko-ext-abi` as a dependency (for `ExtensionHooks`)
3. Use semantic versioning; breaking ABI change = major bump

```ailang
-- motoko-ext-exa-search/register.ail
module motoko-ext-exa-search/register

import motoko-ext-abi/types (ExtensionHooks)
import src/core/config (RuntimeConfig)  -- via config_import resolution

export func register_with_config(cfg: RuntimeConfig) -> ExtensionHooks ! {} {
  { id: "exa_search", provided_tools: ["ExaSearch"], ... }
}
```

### Architecture

```
ailang.toml [extensions]
      │
      ▼
cmd/ailang/ext_registry_gen.go
  ├── LoadManifest()           -- existing internal/pkg function
  ├── resolvePackagePrefix()   -- looks up module prefix in ailang.lock
  ├── deriveShortName()        -- strips motoko-ext- prefix
  ├── renderTemplate()         -- text/template → .ail source
  └── writeOutput()            -- atomic write to output path
```

### Implementation Plan

**Phase 1: TOML schema (~0.5 day, ~50 LOC)**
- [ ] Add `ExtensionRegistryConfig` struct to `internal/pkg/manifest.go`
- [ ] Add `Extensions ExtensionRegistryConfig \`toml:"extensions"\`` to `PackageManifest` (follow `AIProviders` pattern at line 27)
- [ ] Add `Validate()` checks: `config_import` and `hooks_import` required when `packages` non-empty and `motoko-ext-abi` not in deps
- [ ] Unit tests: parse `[extensions]` from TOML fixture

**Phase 2: Generator command (~1 day, ~120 LOC)**
- [ ] New file `cmd/ailang/ext_registry_gen.go` with `extRegistryGenCommand(args []string) error`
- [ ] `--config` flag (default: `ailang.toml`), `--output` flag (overrides `ailang.toml` value), `--dry-run` flag
- [ ] Short name derivation: `deriveShortName(pkgName string) string`
- [ ] Module prefix resolution: read `ailang.lock` for each package's resolved module prefix
- [ ] Template rendering: `text/template` → emit AILANG source
- [ ] Atomic write (write to `.tmp`, rename)
- [ ] Wire into `cmd/ailang/main.go` dispatch: `case "generate-extension-registry":`

**Phase 3: Integration test + docs (~0.5 day)**
- [ ] Test: render from a fixture `ailang.toml` with 2 extension packages → golden-file comparison
- [ ] Test: `--dry-run` prints output without writing
- [ ] Test: error when package not in `ailang.lock`
- [ ] Update `ailang help` output
- [ ] Add to `docs/docs/guides/packages.md` (or new `extension-packages.md`)

### Files to Modify/Create

**New files:**
- `cmd/ailang/ext_registry_gen.go` — generator command (~120 LOC)
- `cmd/ailang/ext_registry_gen_test.go` — golden-file tests (~60 LOC)

**Modified files:**
- `internal/pkg/manifest.go` — add `ExtensionRegistryConfig` struct + field (~40 LOC)
- `internal/pkg/manifest_test.go` — TOML parse tests (~30 LOC)
- `cmd/ailang/main.go` — dispatch `case "generate-extension-registry":` (~5 LOC)

**Total: ~255 LOC**

---

## Examples

### Before: hand-maintained registry ladder

```ailang
-- src/core/ext/registry.ail (HAND-WRITTEN — must edit to add/remove extensions)
func resolve(name: string, cfg: RuntimeConfig) -> Option[ExtensionHooks] {
  if name == "compaction" then Some(register_compaction(cfg))
  else if name == "exa_search" then Some(register_exa_search(cfg))
  else if name == "mcp" then Some(register_mcp(cfg))  -- added in PR #47
  else None
}
```

### After: config-driven

```toml
# ailang.toml
[extensions]
packages = ["motoko-ext-compaction@0.2.0", "motoko-ext-exa-search@0.4.1"]
config_import = "src/core/config.RuntimeConfig"
hooks_import  = "src/core/ext/types.ExtensionHooks"
output        = "src/core/ext/registry_generated.ail"
module_name   = "src/core/ext/registry_generated"
```

```bash
# Add a new extension
ailang add motoko-ext-mcp@0.1.0
ailang generate-extension-registry
# → src/core/ext/registry_generated.ail updated, includes mcp

# Remove an extension
# (delete from ailang.toml [extensions].packages)
ailang generate-extension-registry
# → mcp removed from generated file
```

---

## Success Criteria

- [ ] `ailang generate-extension-registry` produces a valid AILANG file from `ailang.toml`
- [ ] Generated file passes `ailang check --package` with 0 errors
- [ ] Adding `motoko-ext-exa-search` to `ailang.toml` + running generator = working extension (verified end-to-end in motoko_agent)
- [ ] Removing a package from `ailang.toml` + regenerating = extension not loaded
- [ ] Generator fails with a clear error when a package is not in `ailang.lock`
- [ ] `--dry-run` prints without writing
- [ ] All existing `internal/pkg` tests still pass
- [ ] `ailang help` shows `generate-extension-registry`

---

## Testing Strategy

**Unit tests (`ext_registry_gen_test.go`):**
- Golden-file: 2-package `ailang.toml` → exact expected `.ail` output
- Short name derivation: `motoko-ext-exa-search` → `exa_search`, `my-tool` → `my_tool`
- Error: package not in `ailang.lock`
- Error: `config_import` / `hooks_import` missing and `motoko-ext-abi` not in deps

**Integration test (motoko_agent):**
- Replace `src/core/ext/registry.ail` with generated `registry_generated.ail`
- Run `make check_core` — all modules pass
- Smoke: `run_v2_with_stub` with an extension loaded via the generated registry

**Manual testing:**
- `ailang add motoko-ext-exa-search@0.4.1 && ailang generate-extension-registry` on a fresh motoko_agent clone

---

## Deferred Decisions

- **`motoko-ext-abi` package publication** — who publishes it and when is a motoko_agent-side decision; the generator works with inline import paths today and switches to `motoko-ext-abi` references once it exists. Agent may implement Phase 1-2 without waiting.
- **`ailang check` integration** — whether `ailang check --package` automatically re-runs the generator when `ailang.toml` is newer than the generated file is deferred to a follow-up. For now: re-run manually or via Makefile.
- **Template customisation** — whether consumers can supply a custom Jinja/Go template for the generated file is deferred. v1 ships one hardcoded template.

---

## Non-Goals

- **Runtime-resolved imports** — not adding dynamic import to the AILANG language. This generator is the build-step workaround; dynamic import is a separate future feature.
- **Extension discovery/search** — `ailang search motoko-ext-*` is a registry feature (M-PKG-REGISTRY, v1.1). This doc only covers code generation.
- **Automatic regeneration in CI** — out of scope; consumer adds `ailang generate-extension-registry` to their Makefile.
- **`motoko-ext-abi` package creation** — motoko_agent-side work, tracked in `m-motoko-extensions-as-packages.md`.
- **Extensions for non-motoko consumers** — the `[extensions]` schema is generic but the `register_with_config` convention targets motoko-style `ExtensionHooks`. Other consumers may use a different convention; generator is configurable via `config_import`/`hooks_import`.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Generated file's module imports break when package path changes | Med | Generator reads `ailang.lock` for exact resolved path; `ailang.lock` is the ground truth |
| ABI drift: `ExtensionHooks` gains a field, old extensions miss it | Med | `motoko-ext-abi` major version bump required; `ailang check --package` will surface type errors at compile time |
| Template emits invalid AILANG syntax | Low | Golden-file tests + `ailang check --package` in test suite catches this before release |
| `[extensions]` section name collides with future AILANG feature | Low | TOML section is only read by this generator; AILANG runtime never reads it |

---

## Related Documents

**Implemented (patterns followed):**
- [design_docs/implemented/v0_9_7/m-pkg-registry.md](../../implemented/v0_9_7/m-pkg-registry.md) — package registry architecture; `ailang add` command patterns
- [design_docs/implemented/v0_9_7/m-ai-provider-config.md](../../implemented/v0_15_0/m-ai-provider-config.md) — `AIProviders []AIProviderSpec \`toml:"ai_provider"\`` additive TOML pattern (direct precedent for `[extensions]`)

**Planned (consumer of this feature):**
- [`/Users/mark/dev/sunholo/motoko_agent/design_docs/planned/m-motoko-extensions-as-packages.md`](../../../../motoko_agent/design_docs/planned/m-motoko-extensions-as-packages.md) — motoko_agent-side design; this doc is the AILANG prerequisite for Phase 2 of that doc

---

**Document created**: 2026-05-07
**Last updated**: 2026-05-07
