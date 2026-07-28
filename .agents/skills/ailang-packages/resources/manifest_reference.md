# ailang.toml Manifest Reference

## [package] Section

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Package name in `vendor/name` format (e.g., `sunholo/firestore`) |
| `version` | Yes | Semantic version (e.g., `0.1.0`) |
| `edition` | Yes | Language edition (currently always `"1"`) |
| `ailang` | No | Minimum AILANG version (e.g., `">=0.9.5"`). Auto-set by `ailang init package` |
| `module_prefix` | No | Maps existing module paths to package namespace. Single segment, no slashes |
| `description` | No | Human-readable description |
| `license` | No | SPDX license identifier (e.g., `Apache-2.0`) |

### module_prefix

For existing applications adopting the package system without renaming modules:

```toml
[package]
name = "sunholo/docparse"
module_prefix = "docparse"
```

- Source files keep `module docparse/...` declarations
- Consumers import via `import pkg/sunholo/docparse/...`
- Exports can use either prefix: `["docparse/api", "sunholo/docparse/new_module"]`
- Must be a single segment (no `/`)

## [exports] Section

```toml
[exports]
modules = ["sunholo/firestore/client", "sunholo/firestore/fields"]
```

- Only listed modules are importable by other packages
- Unlisted modules are package-private
- If `module_prefix` is set, modules can use either prefix
- Empty list = all modules accessible (permissive default)

## [dependencies] Section

Three dependency types:

```toml
[dependencies]
# Path dependency (local development)
"sunholo/firestore" = { path = "../firestore" }

# Git dependency (version pinned)
"sunholo/auth" = { git = "https://github.com/sunholo-data/ailang-packages", subdir = "packages/auth", tag = "main" }

# Registry dependency (published packages)
"sunholo/firestore" = "0.1.0"
```

**Git deps require `tag` or `rev`:**
```toml
"sunholo/auth" = { git = "https://...", tag = "v0.1.0" }       # by tag
"sunholo/auth" = { git = "https://...", rev = "abc123" }        # by commit
"sunholo/auth" = { git = "https://...", subdir = "packages/auth", tag = "main" }  # monorepo
```

## [effects] Section

```toml
[effects]
max = ["Net", "FS", "Env"]
```

- Declares the maximum effects any function in the package can use
- Functions exceeding this ceiling cause a compile error
- `max = []` means the package is pure (no side effects)
- Available effects: `IO`, `FS`, `Net`, `Env`, `Clock`, `AI`, `Debug`

## [metadata] Section

```toml
[metadata]
tags = ["firestore", "gcp", "database"]
ai_summary = "Firestore REST API client for AILANG"
```

- `tags`: searchable keywords for `ailang search`
- `ai_summary`: one-line description for AI agent discovery

## [stability] Section

```toml
[stability]
level = "experimental"
```

- `experimental`: API may change between versions
- `stable`: Semver guarantees (breaking changes = major version bump)
- `frozen`: No changes planned

## Import Conventions

Three-way import distinction:

```ailang
import ./plan (Plan)                          -- LOCAL: sibling in same package
import pkg/sunholo/firestore/client (getDoc)  -- EXTERNAL: different package
import std/result (Ok, Err)                   -- STDLIB: bundled
```

- `./` resolves in module namespace (not filesystem): `./plan` in module `a/b/c` → `a/b/plan`
- `./sub/bar` supported for child directories
- `pkg/` self-imports also work (backward compatible) but `./` is preferred
- Interfaces and hashes always use canonical paths (no `./` in metadata)

## AGENT.md

Every package should include an `AGENT.md` file for AI agent discovery:

```markdown
# sunholo/firestore

## When to use this package
One paragraph explaining when to import this package.

## Quick start
Working code example with imports.

## Exported functions
Table of function signatures.

## Common patterns
Usage tips and effect requirements.
```
