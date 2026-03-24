# M-PKG: AILANG Package System & Multi-Agent Coordination

**Status**: Planned
**Target**: v1.0.0 (Phase 1–2), v1.x (Phase 3)
**Priority**: P0 — foundational for scale & ecosystem
**Estimated**: 3-4 weeks (Phase 1: 1.5 weeks, Phase 2: 1.5 weeks, Phase 3: deferred)
**Dependencies**:
- Module system (complete — M-R1, v0.2.0)
- Effect system (complete)
- SMT verification (partial, expanding — M-SMT-CROSS-MODULE-TYPES in progress)
- Module scope isolation (complete — v0.9.0)
- CLI runtime (complete)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Content-hash locking guarantees identical dependency resolution |
| A2: Replayability | +1 | Lock file + package hashes = bit-identical builds across environments |
| A3: Effect Legibility | +1 | Package-level effect ceilings make authority visible at distribution boundary |
| A4: Explicit Authority | +1 | Packages declare and are checked against max effect sets; no ambient propagation |
| A5: Bounded Verification | +1 | Package = verification unit; interface summaries compose without full source |
| A6: Safe Concurrency | +1 | Package boundaries define parallelism rules for multi-agent coordination |
| A7: Machines First | +1 | All metadata is structured (TOML/JSON), never prose; AI-browsable task views |
| A8: Minimal Syntax | 0 | Only `import pkg/...` syntax addition; manifest is TOML (existing tooling) |
| A9: Cost Visibility | +1 | Interface hashes enable targeted re-verification; no unnecessary rebuilds |
| A10: Composability | +1 | Packages compose at type/contract/effect boundaries |
| A11: Structured Failure | +1 | Resolution failures are typed: version conflict, hash mismatch, effect violation |
| A12: System Boundary | +1 | Package boundary = explicit system boundary with declared authority |

**Net Score: +11** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Content-addressed resolution is maximally deterministic
- [x] A3 (Effects): Effect ceilings make all authority visible
- [x] A4 (Authority): No ambient access — packages declare what they need
- [x] A7 (Machines First): Structured metadata throughout; no prose-dependent discovery

---

## Problem Statement

AILANG currently supports deterministic modules, explicit imports, and bundled stdlib (27 modules in `std/`). But it lacks:

- **External dependency management** — no way to use third-party packages
- **Reproducible builds** — no lock files, no content hashing
- **Package-level authority constraints** — effects are per-function, not per-package
- **Scalable coordination model** — no package boundaries for multi-agent work

**Current State:**
- All code lives in a single project directory or in bundled `std/`
- `import std/io (println)` works for stdlib; no mechanism for external imports
- `.ailang/cache/compile/manifest.json` exists as an empty stub (`{"version": "v1", "entries": {}}`)
- `internal/manifest/manifest.go` already has schema versioning, SHA256 digests, deterministic JSON serialization, and validation infrastructure
- `examples/manifest.json` tracks 132 examples with status, tags, expected outputs, and dependency metadata

**Core Issue:**

At small scale, missing packages is inconvenience. At large scale, it becomes:
- **Coordination failure** — agents can't reason about change boundaries
- **Context explosion** — every agent must understand the whole repo
- **Non-local reasoning** — no isolation between independent work
- **Unsafe authority propagation** — no package-level effect enforcement

**Impact:**
- External AILANG projects (docparse, web_api_demo) cannot share code
- AI agents working on multi-package repos have no coordination primitives
- No reproducibility guarantees for production deployments
- Python-style dependency hell is the default trajectory without explicit design

---

## Design Goals

| Goal | Axiom | Description |
|------|-------|-------------|
| G1 — Deterministic Resolution | A1 | Same inputs → identical dependency graph |
| G2 — Reproducibility | A2 | All builds are content-addressable and verifiable |
| G3 — Explicit Authority | A4 | Packages declare and are checked against effect ceilings |
| G4 — Minimal Reasoning Surface | A7 | Agents reason over bounded package graphs, not whole repos |
| G5 — Compositional Verification | A5 | Verification composes at package boundaries |
| G6 — Multi-Agent Scalability | A6 | Packages act as units of parallel work and coordination |
| G7 — Machine-Readable by Default | A7 | All package metadata must be structured, not prose |

---

## Non-Goals (v1)

- **No dynamic version ranges** (`^`, `~`) — exact versions only; eliminates resolution non-determinism
- **No install-time scripts/hooks** — packages are pure data + code; no arbitrary execution on install
- **No implicit transitive imports** — if package A depends on B, you must explicitly import B's symbols
- **No global mutable package store** — packages are local to the project; no `node_modules`-style global cache
- ~~**No git-based dependency resolution**~~ — **NOW INCLUDED** in Phase 1.5b (tag/rev pinning, cached clones)
- **No dynamic linking** — all dependencies resolved at compile time
- **No package-level mutation** (publish --force, yank) in Phase 1 — immutable once published

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| TOML manifest format (`ailang.toml`) vs JSON | Determines developer experience and tooling; TOML is more human-writable but JSON matches existing manifest infra | human | design | high |
| Content hash algorithm (SHA256 vs BLAKE3) | Baked into all lock files and registry; changing later requires full re-hash | human | design | high |
| Import syntax `import pkg/vendor/name/module (symbols)` vs `import vendor/name/module` | Affects parser, elaborator, and every user-written import statement | human | design | high |
| Dual hash model (content hash + interface hash) | Determines whether internal refactors require downstream re-verification; architectural commitment | human | design | high |
| Effect ceiling enforcement (compile-time vs install-time vs both) | Determines when effect violations are caught; compile-time is safer but requires more infrastructure | agent | compile | med |
| Lock file format (TOML vs JSON) | Lock files are machine-generated; JSON matches existing infra, TOML matches manifest | agent | design | low |
| Package naming convention (`vendor/name` vs `name` vs `vendor/name/subpkg`) | Affects registry structure, import resolution, and naming collisions | human | design | high |
| Registry model: centralized CRAN-style vs federated | Determines trust model, review process, and scalability | human | Phase 2 | high |
| Workspace support via path dependencies | Enables multi-package repos without registry; critical for v1.0 adoption | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Manifest format: TOML (`ailang.toml`) — strictly declarative, no scripting
- [ ] Content hash: SHA256 (matches existing `internal/manifest/` infrastructure)
- [ ] Import syntax for external packages: `import pkg/vendor/name/module (symbols)`
- [ ] Dual hash model: content hash (all files) + interface hash (exports + types + contracts + effects)
- [ ] Package naming: `vendor/name` (two-level, like Go modules)
- [ ] Lock file format: JSON (`ailang.lock`) — machine-generated, matches existing deterministic JSON serialization
- [ ] Effect ceilings enforced at compile time (not just install time) — **guarded Phase 1.5** if too invasive to compiler pipeline; may ship after core package imports
- [ ] Path dependencies (`{ path = "../utils" }`) are the primary mechanism for Phase 1
- [ ] Workspace model: emergent from path-linked packages, no workspace root manifest in v1
- [ ] Export-list-only privacy: `[exports].modules` is sole authority, no path-based semantic rules

---

## Core Concepts

### 4.1 Module (existing)

- File-level unit
- Namespace boundary
- Compilation unit
- Already supports: `module myproject/parser`, `import std/io (println)`

### 4.2 Package (new)

A package is:
- A **distribution unit** — the thing you publish and install
- A **coordination boundary** — defines what an agent can see and change
- A **verification boundary** — contracts compose at package edges
- An **authority boundary** — effect ceilings are enforced here

### 4.3 Relationship

```
Package (ailang.toml)
  ├── src/module_a.ail      (module)
  ├── src/module_b.ail      (module)
  ├── src/internal/util.ail (internal module — not exported)
  └── tests/
      └── module_a_test.ail
```

A package contains one or more modules. Only explicitly exported modules are accessible to dependents.

---

## Package Structure

```
my-package/
  ailang.toml          # Package manifest (human-written)
  ailang.lock          # Resolved dependencies (machine-generated)
  src/
    module1.ail
    module2.ail
    internal/          # Not exported — package-private
      helpers.ail
  tests/
    module1_test.ail
  examples/
    demo.ail
```

---

## Manifest (`ailang.toml`)

Minimal, machine-first, strictly declarative.

```toml
[package]
name = "sunholo/docparse"
version = "0.3.0"
edition = "1"
description = "Parse DOCX/PPTX/PDF documents into structured blocks"
license = "MIT"

[exports]
modules = [
  "sunholo/docparse/parser",
  "sunholo/docparse/types",
  "sunholo/docparse/format_router"
]

[dependencies]
"sunholo/json" = "0.3.1"
"sunholo/xml"  = "0.7.3"

[dependencies.dev]
"sunholo/test-utils" = "0.1.0"

[effects]
max = ["IO", "FS"]

[metadata]
tags = ["document-processing", "xml", "office-formats"]
ai_summary = "Parse DOCX/PPTX/PDF documents into structured blocks with table extraction"
contracts_verified = 15
contracts_total = 28

[stability]
level = "experimental"  # or "stable" | "frozen"
```

**Section responsibilities:**
- `[package]` — identity and descriptive metadata (name, version, edition, description, license)
- `[exports]` — public API surface
- `[dependencies]` — direct dependencies
- `[effects]` — authority boundary
- `[metadata]` — AI/search/registry-specific structured fields only (tags, ai_summary, contracts_verified)
- `[stability]` — change management

**Design Constraints:**
- No scripting — no `[scripts]` section, no `postinstall`
- No dynamic logic — no conditionals, no platform-specific overrides
- Strictly declarative — parse once, no evaluation needed
- Small surface area — AI can read and write this without errors

### Why TOML over JSON

The existing manifest infrastructure (`internal/manifest/`) uses JSON, and the lock file will too (machine-generated). But `ailang.toml` is human-written, and TOML is:
- More readable for humans reviewing packages
- Standard for package manifests (Cargo.toml, pyproject.toml)
- Comments are supported (JSON has none)
- Less error-prone for AI agents (no trailing comma issues)

The `internal/manifest/` infrastructure is reused for the **lock file** (JSON), not the manifest (TOML).

---

## Lock File (`ailang.lock`)

Machine-generated, authoritative resolution. JSON format (reusing existing `internal/manifest/` deterministic serialization).

```json
{
  "schema": "ailang.lock/v1",
  "schema_version": "1.0.0",
  "schema_digest": "sha256:abc123def456",
  "generated_at": "2026-03-19T14:30:00Z",
  "generator": "ailang lock v1.0.0",
  "packages": [
    {
      "name": "sunholo/json",
      "version": "0.3.1",
      "content_hash": "sha256:a1b2c3d4e5f6...",
      "interface_hash": "sha256:f6e5d4c3b2a1...",
      "source": "registry",
      "effects": ["IO"],
      "exports": ["parseJson", "encodeJson", "Json", "JString", "JNumber"]
    },
    {
      "name": "sunholo/xml",
      "version": "0.7.3",
      "content_hash": "sha256:1a2b3c4d5e6f...",
      "interface_hash": "sha256:6f5e4d3c2b1a...",
      "source": "registry",
      "effects": [],
      "exports": ["parseXml", "XmlNode", "getAttribute", "getChildren"]
    }
  ]
}
```

**Guarantees:**
- Content-addressed resolution — same hash = same code
- Identical builds across environments — lock file is committed to VCS
- Dependency graph fully materialized — no resolution at build time
- Reuses existing `internal/manifest/` infrastructure: schema versioning, SHA256 digests, deterministic JSON, validation

### Lock File Field Semantics

Lock file fields are divided into two categories:

**Semantic fields** (define reproducibility — compared for equality):
- `name`, `version`, `source`, `content_hash`, `interface_hash`

**Informational fields** (useful but not part of identity):
- `generator`, `generated_at`, `effects`, `exports`

When comparing lock files for semantic equivalence, only semantic fields matter.

### Resolution Algorithm

`ailang lock` performs the following steps:

1. Parse root `ailang.toml`
2. Normalize direct dependencies (resolve path deps to canonical paths, registry deps to exact versions)
3. Recursively load dependency manifests (path → read `ailang.toml`; registry → fetch exact version)
4. Reject cycles at the package graph level (error: `circular dependency: A → B → C → A`)
5. Compute content hashes for all resolved packages
6. Compute interface hashes for all resolved packages
7. Emit canonical deterministic JSON lock file (sorted keys, sorted package list)

### Resolution Invariants

These invariants are normative — violations are bugs:

1. **Manifest is the only root.** Only dependencies declared in `ailang.toml` are resolved. No implicit discovery.
2. **Offline build with valid lock file.** Resolution never consults the network during build if `ailang.lock` exists and is valid. Network access only happens during `ailang lock` or `ailang install`.
3. **Lock file is authoritative.** Lock file entries are the sole authority for package source identity during build.
4. **Path dependencies are locked by canonical path + content hash.** The lock file records both the normalized relative path and the content hash at lock time.
5. **Manifest/lock file disagreement = build failure.** If `ailang.toml` declares a dependency not in `ailang.lock`, or vice versa, the build fails with a message to run `ailang lock`.
6. **Content hash mismatch = build failure.** If the resolved source content hash differs from the lock file entry, the build fails.
7. **Exact versions only (v1).** Registry version selection is exact match only. No range resolution, no constraint solving.

### Relationship to Existing Manifests

| Existing | New Role |
|----------|----------|
| `.ailang/cache/compile/manifest.json` | Becomes the **local resolution cache** — tracks which packages are resolved and compiled locally |
| `examples/manifest.json` | Continues as **example tracking** — but `ailang.toml` subsumes it for package metadata |
| `internal/manifest/manifest.go` | **Reused** — schema versioning, SHA256, deterministic JSON, validation all applicable to lock files |

---

## Package Identity: Dual Hash Model

Each package has two hashes. The interface hash must be computed from a canonical representation independent of source ordering, formatting, comments, and internal implementation details.

### Content Hash

- SHA256 of all package source files (sorted by path, concatenated)
- Used for reproducibility — same hash = bit-identical code
- Changes on any source modification

### Interface Hash

SHA256 over canonical JSON of:
- Package name
- Edition
- Exported module list, sorted lexicographically
- For each exported symbol, sorted by name:
  - Symbol name
  - Symbol kind (function, type, constructor)
  - Canonical type representation (normalized, no source locations)
  - Canonical effect row (sorted effect names)
  - Canonical contract AST (requires/ensures bodies, normalized)
- Package max effects, sorted lexicographically

**Excluded from interface hash:**
- Source formatting, whitespace, comments
- Internal module contents
- Source locations and declaration order
- Docstrings and metadata fields

Used for coordination — determines whether dependents need re-verification.

### Why Two Hashes

| Change Type | Content Hash | Interface Hash | Downstream Impact |
|-------------|-------------|----------------|-------------------|
| Internal refactor | changes | **same** | No re-verification needed |
| Bug fix (same API) | changes | **same** | No re-verification needed |
| New export added | changes | changes | Dependents may want to use it |
| Contract change | changes | changes | Dependents MUST re-verify |
| Effect widening | changes | changes | Dependents MUST re-check authority |

This distinction enables:
- **Targeted re-verification** — only re-verify when interface changes
- **Coordination-aware scheduling** — agents know when their work is independent
- **Efficient dependency invalidation** — internal changes don't cascade

---

## Dependency Model

### Direct Dependencies Only

Packages only expose direct dependencies.

**Rule:** No implicit transitive imports.

If `sunholo/docparse` depends on `sunholo/json`, and you depend on `sunholo/docparse`, you **cannot** use `sunholo/json` unless you also declare it as your dependency.

```ailang
-- This works (json is your direct dependency):
import pkg/sunholo/json/parser (parseJson)

-- This is INVALID (accessing internal modules of a dependency):
import pkg/sunholo/docparse/internal/helpers (...)
```

### Import Syntax

External package imports use the `pkg/` prefix to distinguish from local and stdlib imports:

```ailang
-- Stdlib (bundled, always available)
import std/io (println)
import std/json (parseJson)

-- Local module (same project)
import myproject/parser (parse)

-- External package (from registry or path dependency)
import pkg/sunholo/json/parser (parseJson)
import pkg/acme/http/client (get, post)
```

The `pkg/` prefix:
- Makes external dependencies visually distinct
- Enables the compiler to resolve differently (look in lock file, not local filesystem)
- Prevents namespace collisions between local modules and external packages

### Canonical Package/Module Path Mapping

For a package named `vendor/name`:
- Every module in `src/` must declare a module name prefixed with `vendor/name/`
- `src/parser.ail` must contain `module vendor/name/parser`
- `src/internal/helpers.ail` must contain `module vendor/name/internal/helpers`
- `src/sub/deep.ail` must contain `module vendor/name/sub/deep`

**Enforcement:**
- Package name and module prefix mismatch is a **compile error**
- An `import pkg/vendor/name/module` must resolve to a module belonging to the package `vendor/name`
- Cross-package module imports are only valid when rooted in a declared package dependency

**Example:**

```toml
# ailang.toml
[package]
name = "sunholo/docparse"
```

```ailang
-- src/parser.ail
module sunholo/docparse/parser   -- OK: matches package prefix
-- module acme/other/parser      -- ERROR: prefix mismatch with package name
```

---

## Package Graph vs Module Graph

Two levels of dependency graph exist and must not be confused:

**Module import graph** — exists inside and across packages. This is what the compiler resolves. Modules import symbols from other modules.

**Package dependency graph** — coarser. Must remain acyclic for resolution. A package depends on other packages (declared in `ailang.toml`). This graph is what `ailang lock` resolves and what the coordinator uses for task scoping.

**Rule:** Cross-package module imports are only valid when rooted in a declared package dependency. If `import pkg/sunholo/json/parser (parseJson)` appears in your code, `sunholo/json` must be in your `[dependencies]`.

---

## Git-Based Dependencies (Phase 1.5b)

Git dependencies enable version pinning without a registry. This is the standard pre-registry pattern (Go used it for years before `proxy.golang.org`).

### TOML Syntax

```toml
[dependencies]
# Path dependency (Phase 1 — local)
"sunholo/auth" = { path = "../ailang-packages/packages/auth" }

# Git dependency with tag (Phase 1.5b — pinned to release)
"sunholo/auth" = { git = "https://github.com/sunholo-data/ailang-packages", subdir = "packages/auth", tag = "auth-v0.1.0" }

# Git dependency with exact commit (maximum reproducibility)
"sunholo/auth" = { git = "https://github.com/sunholo-data/ailang-packages", subdir = "packages/auth", rev = "abc123def456" }
```

### Resolution Algorithm

1. `ailang lock` encounters a git dependency
2. Hash the git URL → deterministic cache key
3. Clone (or fetch if cached) to `~/.ailang/cache/git/<hash>/`
4. Checkout the specified tag or rev
5. If `subdir` specified, resolve to `cache/<hash>/<subdir>/`
6. Load `ailang.toml` from resolved directory
7. Compute content hash + interface hash (same as path deps)
8. Record `git_url`, `git_rev` (resolved commit hash), `git_subdir` in lock file

### Cache Structure

```
~/.ailang/cache/git/
  a1b2c3d4e5f6/           # sha256(git_url)[:16]
    .git/
    packages/
      auth/ailang.toml
      logging/ailang.toml
```

### Semantics

- **Tag** — resolved to commit hash at `ailang lock` time. Re-running `ailang lock` may pick up a new commit if the tag moved (tags are mutable in git). The lock file pins the resolved commit.
- **Rev** — exact commit hash. Fully immutable. Preferred for reproducibility.
- **No branch deps** — branches are inherently mutable. Use tags or revs.
- **Subdir** — required for monorepo packages (like `ailang-packages`). Points to the package root within the cloned repo.

### Lock File Entry (git dep)

```json
{
  "name": "sunholo/auth",
  "version": "0.1.0",
  "content_hash": "sha256:a1b2c3...",
  "interface_hash": "sha256:d4e5f6...",
  "source": "git",
  "git_url": "https://github.com/sunholo-data/ailang-packages",
  "git_rev": "abc123def456789...",
  "git_subdir": "packages/auth",
  "effects": [],
  "exports": ["sunholo/auth/keys", "sunholo/auth/bearer"]
}
```

---

## AGENT.md — AI Discovery

Each package may include an `AGENT.md` file at its root — a structured guide for AI agents explaining what the package does, when to use it, and common patterns. This is the package equivalent of `CLAUDE.md` but for **consumers** of the package.

### Purpose

- `ai_summary` in `[metadata]` provides one-line discovery (for search/listing)
- `AGENT.md` provides the full usage guide (for implementation)
- AI agents read `AGENT.md` after adding a dependency, like reading API docs

### Format

```markdown
# vendor/name

## When to use this package
<1-2 sentences explaining when an AI agent should reach for this package>

## Quick start
<Code example showing the most common usage pattern>

## Exported functions
<Table: function name, module, type signature, description>

## Common patterns
<Bullet points with idioms, gotchas, and integration advice>
```

### Discovery Flow

1. Agent searches: `ailang.toml` `[metadata].ai_summary` fields (one-line, for filtering)
2. Agent adds dep: `ailang add --path ...` or `ailang add --git ...`
3. Agent reads: `AGENT.md` in the added package (detailed usage guide)
4. Future: `ailang docs sunholo/auth` outputs AGENT.md content

### Manifest Support

```toml
[metadata]
ai_summary = "API key validation, HMAC signing, bearer token extraction"
agent_doc = "AGENT.md"
```

---

## Effect System Integration

Packages declare their maximum effect set:

```toml
[effects]
max = ["IO", "Net"]
```

### Enforcement

**Package effect ceiling = max effects used by package-owned modules.**

A package's declared `max` effects must be sufficient for all effects reachable from its own modules and exported surface. Dependency packages may declare broader ceilings, but unreachable authority does not automatically taint the depending package in v1.

**Compile-time validation:**
- No module in the package may use effects beyond its own `max`
- Effect widening across versions is flagged as a **Class E change** (see Change Classes)
- The effect ceiling is checked against actual effect usage in owned modules, not against dependency ceilings

**Key distinction:** If you depend on `acme/http` (which declares `max = ["IO", "Net"]`), but you only call pure functions from it, your package does NOT need to declare `Net`. Only effects that are actually reachable from your code paths and exported surface matter. Future policy tooling (Phase 3) may impose stricter "dependency authority" audits.

**Example violation — own module exceeds ceiling:**

```toml
# sunholo/json declares:
[effects]
max = []  # Pure package — no effects

# But a module inside contains:
import std/io (println)
export func debug(x: string) -> () ! {IO} = println(x)
# ERROR: effect IO not in package max effects []
```

**Example — dependency has broader ceiling (OK):**

```toml
# Your package:
[effects]
max = ["IO"]

[dependencies]
"acme/http" = "0.1.0"  # declares max = ["IO", "Net"]
# OK if you only call pure functions from acme/http
# ERROR only if your code actually reaches Net-effectful paths
```

### Future Extensions (Phase 3)
- Effect diffing on version upgrade (warn when dependency widens effects)
- Install-time policy checks (organization-level effect allowlists)
- Agent authority gating (coordinator limits which packages an agent can use)

---

## Multi-Agent Coordination Model

### Packages as Coordination Units

Each package defines:
- **Reasoning boundary** — agent only needs to understand the package it's working on
- **Ownership boundary** — one agent per package avoids conflicts
- **Verification boundary** — contracts compose at package edges
- **Authority boundary** — effect ceilings are enforced per package

### Change Classification

| Class | Description | Coordination Cost | Example |
|-------|-------------|-------------------|---------|
| A | Internal only (no interface hash change) | Low | Refactor, optimize, fix internal bug |
| B | Additive API (new exports, no removals) | Medium | Add new function, new module |
| C | Contract/effect change | High | Strengthen precondition, add effect |
| D | Breaking change (removed/renamed export) | Very High | Remove function, change signature |
| E | Authority widening | Critical | Add new effect to max set |

### Parallelism Rules

Agents may operate independently when:
- Confined to a single package (Class A change)
- No interface hash change
- No effect widening

Agents **must coordinate** when:
- Modifying exported APIs (Class B-D)
- Changing effect ceilings (Class E)
- Adding new dependencies (may affect other packages' effect budgets)

### Coordination Workflow

```
1. Task received → scoped to package(s)
2. Dependency analysis → which packages are affected?
3. Change classification → what class is this change?
4. Local verification → contracts pass within package
5. Interface validation → interface hash unchanged? (Class A = done)
6. Dependent re-check → only affected packages re-verify (Class B-E)
```

### Package Compatibility Rules

Change classes map to version compatibility semantics (relevant once registry publishing arrives):

| Change | Compatibility | Version Impact |
|--------|--------------|----------------|
| Internal-only (Class A) | Patch-compatible | `0.3.1` → `0.3.2` |
| Additive export (Class B) | Minor-compatible | `0.3.1` → `0.4.0` |
| Contract strengthening (Class C) | Breaking | `0.3.1` → `1.0.0` |
| Contract weakening (Class C) | Usually compatible | Case-by-case |
| Export removal or type change (Class D) | Breaking | Major version bump |
| Effect widening on exported symbols (Class E) | Breaking or critical | Major version bump |

### Integration with Existing Coordinator

The AILANG coordinator daemon (`internal/coordinator/`) already manages:
- Task assignment to agents (Claude, Gemini)
- Worktree isolation per task
- Approval workflows

Package boundaries provide the coordinator with:
- **Task scoping** — assign packages, not files
- **Conflict detection** — interface hash changes signal coordination needs
- **Parallel scheduling** — Class A changes on different packages can run simultaneously
- **Authority limits** — agent's effect budget derived from package effect ceiling

---

## Encapsulation Rules

### Strict Export Model

Any module not listed in `[exports].modules` is inaccessible from outside the package. The export list is the sole authority for visibility — there is no path-based semantic rule. `internal/` is a recommended naming convention for clarity, but has no special compiler meaning unless later elevated to a semantic rule.

**Enforcement:** The package loader checks every `import pkg/...` against the target package's export list. Imports of non-exported modules produce a compile error with the list of available exports.

### Rationale

- **Prevents hidden coupling** — dependents can't rely on implementation details
- **Reduces reasoning surface** — agents only see the public API
- **Improves parallelism** — internal changes are always Class A (independent)
- **Matches existing module scoping** — v0.9.0 already isolates non-exported functions within modules; this extends the pattern to package level
- **Machine-friendly** — a single list to check, no path-pattern matching

---

## Workspace Model (Phase 1)

**In v1, workspaces are emergent from path-linked packages, not a separate first-class manifest type.**

There is no `ailang.workspace.toml` in v1. Each package has its own `ailang.toml` and is independently valid. Multi-package repos are expressed through path dependencies between sibling packages. A workspace root may optionally be added in a future version if coordination tooling requires it.

This means:
- Each package has its own `ailang.lock` (authoritative for that package)
- Sibling packages resolve each other via path dependencies
- No shared dependency version pinning across packages (each package pins independently)
- The coordinator scopes tasks to individual packages, not workspace roots

Path dependencies enable multi-package repos without requiring a registry:

```toml
[dependencies]
"shared/utils" = { path = "../utils" }
"shared/types" = { path = "../types" }
```

**Purpose:**
- Enables multi-package repos (monorepo pattern)
- Immediate support for docparse and other AILANG projects
- Avoids registry dependency in Phase 1
- Content hash computed from resolved path (for lock file consistency)

**Example: docparse as path-linked packages**

```
docparse/
  packages/
    types/
      ailang.toml          # name = "sunholo/docparse-types"
      ailang.lock           # own lock file
      src/document.ail
    parser/
      ailang.toml          # name = "sunholo/docparse-parser"
      ailang.lock           # own lock file, includes path dep on types
      src/docx_parser.ail
      src/pptx_parser.ail
    main/
      ailang.toml          # depends on types + parser via path
      ailang.lock           # own lock file
      src/main.ail
```

---

## Registry Model (Phase 2)

### CRAN-Style Curation, Rethought for AI

R's CRAN works because:
- Packages are reviewed before inclusion
- Checksums ensure reproducibility
- Task Views organize by domain

AILANG's registry extends this for AI coders:

| CRAN Feature | AILANG Equivalent | AI Enhancement |
|-------------|-------------------|----------------|
| Package review | Submission review | Automated: compile, verify contracts, check effects |
| Checksums | Content hash + interface hash | Dual hashing enables targeted re-verification |
| Task Views | **AI Task Views** | Machine-readable domain groupings with capability summaries |
| DESCRIPTION file | `ailang.toml` | Structured metadata, not prose; `ai_summary` field |
| Depends/Imports | `[dependencies]` | Effect-aware: dependency effects must fit within package ceiling |

### AI Task Views

Instead of prose descriptions, task views are structured:

```json
{
  "name": "document-processing",
  "description": "Parse, extract, and transform document formats",
  "packages": [
    {
      "name": "sunholo/docparse",
      "ai_summary": "Parse DOCX/PPTX/PDF into structured blocks",
      "effects": ["IO", "FS"],
      "contracts_verified": 15,
      "stability": "experimental",
      "exports_count": 12
    }
  ],
  "common_patterns": [
    "import pkg/sunholo/docparse/parser (parseDocument)",
    "import pkg/sunholo/docparse/types (Block, TableCell)"
  ]
}
```

An AI agent can:
1. `ailang registry search --task document-processing` → get structured results
2. Read the `ai_summary` to understand capabilities
3. See `effects` to know authority requirements
4. Check `contracts_verified` for reliability signals
5. Copy `common_patterns` directly into code

### Required Registry Fields

Every published package must provide:
- Content hash + interface hash
- Effect summary (max effects)
- Export surface (list of exported symbols with types)
- Compatibility version (edition)
- `ai_summary` (one-line structured description)
- Tags (for task view membership)

---

## Verification Boundaries

Packages act as compositional verification units:

### Internal Verification

Full SMT verification is possible within a package:
- All source is available
- All contracts can be checked
- Cross-function verification works (existing `ailang verify` infrastructure)

### Boundary Verification

At package boundaries, only summarized guarantees are available:
- **Types** — exported function signatures
- **Contracts** — requires/ensures blocks on exported functions
- **Effects** — package effect ceiling

### Outcome

Downstream verification does not require full source of dependencies. An AI agent verifying `myapp` that depends on `sunholo/json` only needs:
- `sunholo/json`'s interface hash (has it changed?)
- `sunholo/json`'s exported contracts (what does it guarantee?)
- `sunholo/json`'s effect summary (what authority does it need?)

This connects directly to the M-SMT-CROSS-MODULE-TYPES work: demand-driven type filtering (Phase 1, complete) already collects only the types a function needs. Package boundaries formalize this into a first-class concept.

---

## Semantic Freezing (Phase 3)

Packages may declare stability levels:

```toml
[stability]
level = "experimental"  # default
# or "stable" — API changes require major version bump
# or "frozen" — no API changes; security fixes only
```

**Purpose:**
- Guide agent behavior — agents should not refactor `frozen` packages
- Reduce churn — `stable` packages need explicit approval for interface changes
- Enforce coordination discipline — stability level affects change classification cost

---

## Stdlib Status

Stdlib is **not** a package in v1. `std/...` imports remain a compiler-resolved special namespace, bundled into the `ailang` binary. Stdlib modules do not appear in `ailang.lock` and do not have content or interface hashes.

`ailang tree` may display stdlib dependencies as `[bundled stdlib]` for user clarity, but they are not part of the package dependency graph.

This avoids pulling stdlib into the package system prematurely. Stdlib may become a set of packages in a future version if modularization is needed.

---

## Backward Compatibility

Projects without `ailang.toml` continue to work as single-package local module projects with:
- Local module imports (`import myproject/parser (parse)`)
- Stdlib imports (`import std/io (println)`)
- No external package imports (no `import pkg/...`)
- No lock file

The package system is entirely opt-in. Creating `ailang.toml` enables package features; its absence preserves existing behavior. This ensures migration is non-threatening — existing projects need zero changes to continue working on v1.0.

---

## CLI Surface

### Phase 1 (v1.0)

```bash
ailang init                          # Create ailang.toml in current directory
ailang init --name sunholo/mylib     # Create with specific package name
ailang add sunholo/json@0.3.1       # Add dependency (path or registry)
ailang add ../shared/utils --path    # Add path dependency
ailang lock                          # Resolve dependencies, generate ailang.lock
ailang tree                          # Show dependency tree
ailang verify                        # Verify contracts (already exists, now package-aware)
ailang build                         # Compile with package resolution
```

### Phase 2

```bash
ailang publish                       # Publish to registry
ailang install                       # Install from lock file
ailang search "document processing"  # Search registry (structured results)
ailang registry task-views           # List AI task views
ailang audit                         # Check dependencies for issues
```

### Phase 3

```bash
ailang verify --boundary             # Verify at package boundaries only
ailang diff sunholo/json@0.3.0..0.3.1  # Show interface hash diff
ailang policy check                  # Check against organization effect policy
```

---

## Solution Design

### Overview

Three-phase approach, each independently shippable. Phase 1 delivers local package management (manifest, lock file, path dependencies, export enforcement). Phase 2 adds the registry. Phase 3 adds AI-native coordination features.

### Architecture

**Current flow:**
```
source.ail → parser → elaborator → type checker → evaluator
                                        ↓
                          import std/io (println)
                                ↓
                      stdlib embedded in binary
```

**Proposed flow (Phase 1):**
```
ailang.toml → resolver → dependency graph
                              ↓
source.ail → parser → elaborator → type checker → evaluator
                                        ↓
                          import pkg/vendor/name/mod (fn)
                                       ↓
                            ailang.lock → local cache
                                       ↓
                              resolved source files
```

### Components

1. **Package Manifest Parser** (`internal/pkg/manifest.go`, ~200 LOC)
   - Parse `ailang.toml` (TOML format)
   - Validate required fields, effect declarations
   - Reuse `internal/manifest/` for schema versioning and digests

2. **Dependency Resolver** (`internal/pkg/resolver.go`, ~300 LOC)
   - Resolve path dependencies to absolute paths
   - Build dependency graph
   - Detect cycles (reuse existing DFS from `internal/runtime/`)
   - Generate `ailang.lock`

3. **Content Hasher** (`internal/pkg/hasher.go`, ~150 LOC)
   - Compute content hash: SHA256 of sorted source files
   - Compute interface hash: SHA256 of exported types + contracts + effects
   - Reuse existing `crypto/sha256` from `internal/manifest/manifest.go`

4. **Package Loader** (`internal/pkg/loader.go`, ~250 LOC)
   - Resolve `import pkg/...` in the parser/elaborator
   - Load package from lock file resolution
   - Enforce export visibility (only exported modules accessible)
   - Effect ceiling checking

5. **CLI Commands** (`cmd/ailang/pkg_*.go`, ~400 LOC)
   - `init`, `add`, `lock`, `tree` commands
   - Integration with existing CLI infrastructure

6. **Lock File Manager** (`internal/pkg/lockfile.go`, ~200 LOC)
   - Generate deterministic JSON lock file
   - Reuse `internal/manifest/` serialization
   - Validate lock file against manifest

### Implementation Plan

**Phase 1: Local Package Management** (~1.5 weeks)

*Milestone 1A: Manifest & Lock File (~3 days)*
- [ ] TOML parser integration (use `github.com/BurntSushi/toml`)
- [ ] `internal/pkg/manifest.go` — parse `ailang.toml`
- [ ] `internal/pkg/lockfile.go` — generate/validate `ailang.lock` (reuse `internal/manifest/`)
- [ ] `internal/pkg/hasher.go` — content hash + interface hash computation
- [ ] `cmd/ailang/pkg_init.go` — `ailang init` command
- [ ] `cmd/ailang/pkg_lock.go` — `ailang lock` command
- [ ] Tests: manifest parsing, hash computation, lock file roundtrip

*Milestone 1B: Path Dependencies (~2 days)*
- [ ] `internal/pkg/resolver.go` — resolve `{ path = "..." }` dependencies
- [ ] Dependency graph construction with cycle detection
- [ ] `cmd/ailang/pkg_add.go` — `ailang add --path` command
- [ ] `cmd/ailang/pkg_tree.go` — `ailang tree` command
- [ ] Tests: path resolution, cycle detection, multi-package workspace

*Milestone 1C: Package Imports & Export Enforcement (~3 days)*
- [ ] Parser: recognize `import pkg/...` syntax
- [ ] Elaborator: resolve `pkg/` imports via lock file
- [ ] Loader: load package source from resolved path
- [ ] Export enforcement: reject access to non-exported modules
- [ ] Effect ceiling enforcement: compile-time check
- [ ] Tests: import resolution, export visibility, effect violations
- [ ] Example: `examples/runnable/package_imports.ail`

*Milestone 1D: Integration & Polish (~2 days)*
- [ ] Integrate with existing `ailang run`, `ailang compile`, `ailang verify`
- [ ] Update `ailang build` to resolve packages first
- [ ] Error messages: missing dependency, hash mismatch, effect violation
- [ ] Documentation: `docs/docs/guides/packages.md`
- [ ] Convert docparse to use `ailang.toml` as proof-of-concept

**Phase 2: Registry** (~1.5 weeks, may defer to v1.1)

*Milestone 2A: Registry Client (~3 days)*
- [ ] Registry API client (`internal/pkg/registry.go`)
- [ ] `ailang search` — structured search with AI-friendly output
- [ ] `ailang install` — download from registry, verify hashes
- [ ] `ailang add vendor/name@version` — add registry dependency

*Milestone 2B: Registry Server (~4 days)*
- [ ] Package submission endpoint
- [ ] Automated validation: compile, verify contracts, check effects
- [ ] Content + interface hash verification
- [ ] AI task view generation
- [ ] Storage backend (Cloud Storage / Firestore — matches existing infra)

*Milestone 2C: Publishing (~2 days)*
- [ ] `ailang publish` — package, hash, submit
- [ ] Immutability enforcement (no re-publish same version)
- [ ] `ailang audit` — dependency vulnerability check

**Phase 3: AI Coordination Features** (deferred to v1.x)

- [ ] Change classification engine (A-E based on interface hash diff)
- [ ] Coordinator integration (assign packages to agents)
- [ ] Verification summaries (interface-level contract exports)
- [ ] AI task views in registry
- [ ] Trust scoring (based on verification %, stability, community usage)
- [ ] Effect policy engine (organization-level effect allowlists)
- [ ] Semantic freezing enforcement

### Files to Modify/Create

**New files (Phase 1):**
- `internal/pkg/manifest.go` — TOML manifest parser (~200 LOC)
- `internal/pkg/manifest_test.go` — Tests (~150 LOC)
- `internal/pkg/resolver.go` — Dependency resolver (~300 LOC)
- `internal/pkg/resolver_test.go` — Tests (~200 LOC)
- `internal/pkg/hasher.go` — Content + interface hashing (~150 LOC)
- `internal/pkg/hasher_test.go` — Tests (~100 LOC)
- `internal/pkg/loader.go` — Package loader for imports (~250 LOC)
- `internal/pkg/loader_test.go` — Tests (~150 LOC)
- `internal/pkg/lockfile.go` — Lock file generation (~200 LOC)
- `internal/pkg/lockfile_test.go` — Tests (~100 LOC)
- `cmd/ailang/pkg_init.go` — `init` command (~80 LOC)
- `cmd/ailang/pkg_add.go` — `add` command (~100 LOC)
- `cmd/ailang/pkg_lock.go` — `lock` command (~80 LOC)
- `cmd/ailang/pkg_tree.go` — `tree` command (~60 LOC)

**Modified files (Phase 1):**
- `internal/parser/parser.go` — Recognize `import pkg/...` syntax (~30 LOC)
- `internal/elaborate/imports.go` — Resolve pkg imports via package loader (~50 LOC)
- `internal/loader/loader.go` — Integrate package loader (~40 LOC)
- `cmd/ailang/main.go` — Register new CLI commands (~20 LOC)
- `go.mod` — Add `github.com/BurntSushi/toml` dependency

**Phase 1 Total: ~2,340 LOC new, ~140 LOC modified**

---

## Examples

### Example 1: Creating a Package

```bash
$ cd my-project
$ ailang init --name sunholo/my-lib
✓ Created ailang.toml
```

```toml
# ailang.toml
[package]
name = "sunholo/my-lib"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/my-lib/core"]

[effects]
max = []
```

### Example 2: Adding a Path Dependency

```bash
$ ailang add ../shared/json --path
✓ Added shared/json as path dependency
$ ailang lock
✓ Generated ailang.lock (2 packages)
```

```toml
# ailang.toml
[dependencies]
"shared/json" = { path = "../shared/json" }
```

### Example 3: Using Package Imports

```ailang
module sunholo/my-lib/core

import std/io (println)
import pkg/shared/json/parser (parseJson)

export pure func processData(raw: string) -> string =
  let parsed = parseJson(raw)
  in "processed: " ++ raw
```

### Example 4: Effect Ceiling Violation

```toml
# ailang.toml
[effects]
max = []  # Pure package
```

```ailang
-- module in the package:
import std/io (println)
export func debug(x: string) -> () ! {IO} = println(x)
```

```
$ ailang build
ERROR: effect violation in sunholo/my-lib
  Module sunholo/my-lib/core uses effect IO
  But package max effects = [] (pure)
  Add IO to [effects].max in ailang.toml if intended
```

### Example 5: Dependency Tree

```bash
$ ailang tree
sunholo/docparse@0.3.0
├── sunholo/json@0.3.1 (path: ../json)
│   └── (no dependencies)
├── sunholo/xml@0.7.3 (path: ../xml)
│   └── (no dependencies)
└── std/io (bundled)
```

---

## Success Criteria

### Phase 1 (v1.0)
- [ ] `ailang init` creates valid `ailang.toml`
- [ ] `ailang add --path` adds path dependencies
- [ ] `ailang lock` generates deterministic `ailang.lock` with content + interface hashes
- [ ] `import pkg/vendor/name/module (symbols)` resolves correctly
- [ ] Non-exported modules are inaccessible from outside package
- [ ] Effect ceiling violations caught at compile time
- [ ] `ailang tree` shows dependency graph
- [ ] docparse converted to use `ailang.toml` as proof-of-concept
- [ ] All existing tests pass (3400+)
- [ ] Linting clean
- [ ] Documentation: packages guide

### Phase 2
- [ ] `ailang publish` submits to registry with automated validation
- [ ] `ailang install` downloads and verifies hashes
- [ ] `ailang search` returns structured AI-readable results
- [ ] Registry rejects packages that fail compilation or exceed declared effects

### Phase 3
- [ ] Change classification (A-E) computed from interface hash diffs
- [ ] Coordinator assigns packages to agents based on change class
- [ ] AI task views available in registry
- [ ] Trust scores computed from verification %, stability, community data

---

## Testing Strategy

**Unit tests:**
- Manifest TOML parsing (valid, invalid, missing fields)
- Hash computation (deterministic, same input = same hash)
- Lock file generation and validation
- Dependency resolution (path, cycles, diamond)
- Export enforcement (allowed, denied)
- Effect ceiling checking (within bounds, violation)

**Integration tests:**
- Multi-package workspace with path dependencies
- `import pkg/...` end-to-end resolution
- Package-aware `ailang verify` (contracts compose at boundaries)
- CLI commands (`init`, `add`, `lock`, `tree`)

**End-to-end tests:**
- Convert docparse to packages, verify all existing functionality works
- Create synthetic multi-package project, test full workflow
- Verify existing `std/` imports continue to work unchanged

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether `ailang.toml` parsing uses `BurntSushi/toml` or `pelletier/go-toml` — agent may choose based on API ergonomics
- Whether the `pkg/` prefix in imports is a parser-level keyword or an elaborator convention — agent may choose
- Lock file indentation and formatting details — agent may choose, must be deterministic
- Whether `ailang init` uses interactive prompts or flags-only — agent may choose
- Whether the content hasher ignores whitespace/comments or hashes raw bytes — agent should document tradeoffs and choose

---

## Open Questions (Require Human Input)

**Q1 — Import syntax final form:** RESOLVED
- **Decision:** `import pkg/vendor/name/module (symbols)` — explicit namespace class is worth the verbosity

**Q2 — Interface hash definition:** RESOLVED
- **Decision:** Include exported symbol names, canonical types, canonical effects, canonical contracts, exported module names, package max effects, edition. Exclude docstrings, comments, declaration order, internal code. See "Interface Hash" section for canonical spec.

**Q3 — Registry trust model:** DEFERRED to Phase 2
- **Leaning:** Centralized first. Federated later only if needed.
- Who can publish? To be determined — open vs invitation-only vs review-required.

**Q4 — Cross-package SMT:** DEFERRED to Phase 3
- **Leaning:** Export verification summaries (pass/fail per contract), not Z3 proof objects.
- Full proof export is a future optimization if needed.

**Q5 — Version semantics:** RESOLVED
- **Decision:** Semantic versions for humans, content hashes for machines. Both appear in lock file. Human-readable versions for registry discovery; hashes for build reproducibility.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| TOML parser adds Go dependency | Low | Single dependency, well-maintained, standard |
| Import syntax `pkg/` conflicts with module named `pkg` | Medium | Reserve `pkg` as keyword; lint for it |
| Effect ceiling too restrictive for real packages | Medium | Start permissive (effects optional in Phase 1); tighten in Phase 2 |
| Interface hash instability (changes on minor type system updates) | High | Version the hash algorithm; document what's included |
| Registry becomes bottleneck for adoption | Medium | Path dependencies work without registry (Phase 1) |
| Multi-package workspaces are complex to manage | Medium | `ailang tree` and clear error messages; workspace docs |
| AI agents generate incorrect `ailang.toml` | Low | Validation with structured error messages; `ailang init` generates valid defaults |
| Existing projects break on upgrade to v1.0 | High | Packages are opt-in; bare module files continue to work without `ailang.toml` |

---

## Failure Modes Addressed

**Without this system:**
- Hidden dependencies (import anything from anywhere)
- Authority leakage (effects propagate without bounds)
- Global reasoning requirements (agent must understand whole repo)
- Agent interference (two agents modify same code)
- Non-deterministic builds (different environments, different results)

**With this system:**
- Bounded reasoning (agent reasons about one package)
- Explicit authority (effects declared and enforced per package)
- Deterministic resolution (content-addressed, locked)
- Composable verification (contracts compose at package boundaries)
- Schedulable work units (packages = coordination primitives)

---

## Related Documents

**Implemented (directly informs this design):**
- [design_docs/implemented/v0_2_0/m_r1_module_execution.md](design_docs/implemented/v0_2_0/m_r1_module_execution.md) — Module runtime: ModuleInstance, ModuleRuntime, topo sort, cycle detection
- [design_docs/implemented/v0_9_0/m-module-scope.md](design_docs/implemented/v0_9_0/m-module-scope.md) — Module scope isolation (non-exported function collision fix)
- [design_docs/implemented/v0_5_7/m-dx11-stdlib-discovery.md](design_docs/implemented/v0_5_7/m-dx11-stdlib-discovery.md) — AI-first stdlib discovery via CLI
- [design_docs/implemented/v0_5_10/m-codegen-cross-module-impl.md](design_docs/implemented/v0_5_10/m-codegen-cross-module-impl.md) — Cross-module codegen patterns
- [design_docs/implemented/v0_4_8/m-import-aliasing.md](design_docs/implemented/v0_4_8/m-import-aliasing.md) — Import aliasing

**Planned (check for overlap):**
- [design_docs/planned/m-smt-cross-module-types.md](design_docs/planned/m-smt-cross-module-types.md) — Z3 cross-module type resolution (in progress)
- [design_docs/planned/v1_0_0/m-type-v2-migration.md](design_docs/planned/v1_0_0/m-type-v2-migration.md) — TFunc→TFunc2 cleanup for v1.0
- [design_docs/planned/v1_0_0/m-agent-orchestration.md](design_docs/planned/v1_0_0/m-agent-orchestration.md) — std/agent module for AI agent governance
- [design_docs/planned/v0_9_4/m-serve-api-transitive-imports.md](design_docs/planned/v0_9_4/m-serve-api-transitive-imports.md) — Transitive import resolution

**Existing infrastructure:**
- [internal/manifest/manifest.go](internal/manifest/manifest.go) — Schema versioning, SHA256 digests, deterministic JSON, validation
- [examples/manifest.json](examples/manifest.json) — Example tracking manifest (132 examples)
- `.ailang/cache/compile/manifest.json` — Compile cache stub (future local resolution cache)

---

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [R CRAN](https://cran.r-project.org/) — Curated package repository model
- [Cargo.toml](https://doc.rust-lang.org/cargo/reference/manifest.html) — Rust's manifest format (inspiration for TOML choice)
- [Go modules](https://go.dev/ref/mod) — Two-level naming, content hashing, `go.sum`
- [Nix](https://nixos.org/) — Content-addressable, reproducible builds (inspiration for hash model)

---

## Implementation Status

| Phase | Status | What |
|-------|--------|------|
| Phase 1 (v1.0) | **DONE** | Manifest, lock file, path deps, `import pkg/...`, export enforcement |
| Phase 1.5 (v1.0) | **DONE** | Interface hash, content validation at build, effect ceilings |
| Phase 1.5b (v1.0) | **IN PROGRESS** | Git-based deps, AGENT.md, ailang-packages repo |
| Phase 2 (v1.1) | Planned | Registry (centralized CRAN-style), `ailang publish/install/search` |
| Phase 3 (v1.x) | Planned | AI coordination: change classification, trust scoring, effect policies |

### ailang-packages Repository

URL: `https://github.com/sunholo-data/ailang-packages`

Curated first-party packages extracted from production projects (docparse, ecommerce demos, streaming agents). Becomes the seed for the Phase 2 registry.

| Package | ai_summary | Effects |
|---------|-----------|---------|
| `sunholo/gcp-auth` | GCP ADC OAuth2 token exchange, project detection | FS, Net |
| `sunholo/auth` | API key validation, HMAC hashing, bearer tokens | Pure |
| `sunholo/http-helpers` | HTTP request builders, auth headers, JSON response parsing | Net |
| `sunholo/logging` | Structured JSON logging for Cloud Run | IO |
| `sunholo/config` | Config loading from env vars with validation | Env |
| `sunholo/testing-utils` | Test assertion helpers | Pure |

---

## Future Work

- **Binary packages** — compiled AILANG bytecode distribution (depends on M-PERF4 bytecode interpreter)
- **Cross-language interop packages** — packages that wrap Go/Python libraries via FFI
- **Package-level benchmarks** — performance regression tracking per package version
- **Incremental verification** — cache verification results per interface hash; skip re-verification when unchanged
- **Organization policies** — `ailang-policy.toml` at org level defining allowed effects, required verification %, approved registries
- **`ailang docs <package>`** — output AGENT.md content for installed packages

---

## Key Principle

> **Packages in AILANG are not just for reuse. They are the primary unit of autonomous coordination.**

They define:
- What an agent can **see**
- What it can **change**
- What **authority** it has
- What **guarantees** it must preserve

---

**Document created**: 2026-03-19
**Last updated**: 2026-03-20 (rev 3 — git deps, AGENT.md, implementation status, ailang-packages repo)
