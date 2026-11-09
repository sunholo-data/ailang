---
sidebar_position: 4
title: Stability Promise (1.x)
description: What syntax, stdlib, CLI, package format and hosted MCP tool surface AILANG guarantees across the 1.x release line.
reviewed: 2026-11-09
reviewBy: 2027-02-09
---

# AILANG 1.x Stability Promise

> **RATIFIED 2026-11-09 (Mark, ledger `D-65`; v1.0 bar v3 gate G1).** Re-drafted the same day
> against bar v3 (ledger `D-64`, gates G1, G3 and G4a). The tier *semantics* below are frozen by the
> [M-V1-STABILITY-PROMISE design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_29_0/m-v1-stability-promise.md).
> The tier *assignments* were re-done against what exists today: the dispatch table behind
> `ailang --help` after the CLI was cut down to 17 visible commands plus the `dev`, `ops`, `eval`
> and `pkg` groups, the frozen stdlib interfaces (`make freeze-stdlib`), the package manifest and
> lockfile code, and the hosted executor's MCP tool list. They were ratified as drafted. Rows marked
> **⚠ close call** were the assignments most open to argument; each note says why it landed where it did.

This page says what a consumer, whether a human developer or an AI coding agent, may rely on
across AILANG's `1.x` releases. `1.0.0` means a promise about what will not break. It does not
claim anything about adoption.

## Tiers

Every public surface is in exactly one of three tiers.

| Tier | Promise | Marked in docs |
|------|---------|----------------|
| **Stable** | Covered by the 1.x promise. A breaking change requires a **major** version bump. | (default; listed below) |
| **Experimental** | Shipped and usable, but **may change in a 1.x minor** with a CHANGELOG entry. | Explicitly listed as Experimental. |
| **Internal** | **No promise.** May change or be removed at any time. Listed here only so that anything missing from this page is not left ambiguous. | Explicitly listed as Internal. |

### What "Stable" means in 1.x

- **No breaking changes to the Stable surface within 1.x minor releases.** The behaviour,
  signatures and semantics of Stable syntax, stdlib functions, CLI commands, package formats and
  MCP tools stay the same across `1.0 → 1.1 → …`. Breaking any of them requires `2.0`.
- **Experimental surface may change in a minor**, but only with an explicit CHANGELOG entry
  that describes the change and how to migrate.
- **Deprecations last at least one minor before removal.** A Stable item that is going to be
  removed is first marked deprecated, in the docs and at compile time where the tooling supports
  it. It keeps working for at least one full minor release and can only be removed in a later
  major.
- **Additive changes are always allowed.** New stdlib modules or functions, new CLI commands or
  optional flags, new optional manifest keys, new MCP tools, and new syntax sugar can land in any
  minor, because they do not break existing programs or callers.

### Criteria used to draft the assignments

An item is drafted **Stable** when all of these hold: it is documented in the canonical reference
or teaching prompt, it is exercised by `examples/`, the eval suite, or a published package, and it
has no open P0 defect. Items that are newer, niche or still changing are drafted **Experimental**.
That is the cheaper way to be wrong, because an Experimental item can be promoted later without
breaking anyone. Development, diagnostic and fleet-operations surface that has no consumer
contract is **Internal**.

**Rulings applied (D-64, Mark, 2026-11-09):** `ailang messages` is **Experimental**. The
coordinator, missions, approvals, dashboards, eval tooling and chains/observatory are
**Internal**. The cloud executor's MCP tool surface is **Stable**.

## At a glance

| Area | Stable | Experimental | Internal | Source of truth |
|------|-------:|-------------:|---------:|-----------------|
| Stdlib modules | 17 | 31 | 3 | `.stdlib-golden/` (49 frozen modules) + 2 subdirectory modules |
| CLI commands | 19 | 23 | 46 | `cmd/ailang` dispatch table → [CLI Reference](/docs/reference/cli) |
| Package surface | 12 | 9 | 1 | `internal/pkg` manifest + lockfile code |
| Hosted MCP tools | 24 | 5 | 0 | `mcp_tools/`, `executor_tools/`, `executor_oauth/`, built-in `submit_feedback` |

## Syntax surface

The Stable syntax set is defined **by reference** to the canonical teaching prompt and language
reference rather than listed again here, so there is one source of truth:

- **Canonical reference:** `ailang prompt` and the [Language Syntax](/docs/reference/language-syntax)
  page. The active prompt version is **v0.16.12** at the time of writing (`prompts/versions.json`
  `.active`). Read the version from there, not from this page.
- The **Stable** constructs are the ones that reference documents as current:
  - modules and imports, including `pkg/vendor/name/...` package imports
  - `func`, `pure func` and `export` declarations
  - `let` bindings and block bodies `{ …; … }`
  - lambdas (`\x. …`) and `if … then … else …`
  - brace-form `match` with pattern guards
  - algebraic data types, records and row-polymorphic record access
  - effect rows (`! {IO, FS, …}`)
  - the `::` and `++` list operators
  - string interpolation (`"${expr}"`)
  - `requires` / `ensures` contracts

  Every one of these was checked with `ailang check` and `ailang run` on a build of this branch on
  2026-11-09.
- **`pure func` is Stable ⚠ close call, on the condition that P0 [#1443](https://github.com/sunholo-data/ailang/issues/1443)
  is closed.** Until then the type checker accepts a `pure func` that declares an effect row.
  This is a v1.0 G2 item.
- **Retired syntax is not Stable and the parser rejects it.** For example, the ML/Haskell form
  `match … with | …` produces `PAR019` ("'match ... with' is not valid AILANG syntax"). Use the
  brace form `match x { pat => expr, … }`.
- **Experimental syntax:**
  - Parameterised effects and capability refinement, such as `Net[scope=public]` and `@limit=N`
    (see [Parameterised effects](/docs/guides/parameterised-effects)).
  - IFC labels and `Declassify` (⚠ close call): P0 [#1134](https://github.com/sunholo-data/ailang/issues/1134)
    is open, because label refinements are lost across module boundaries.
  - The `serve-api` export annotations (`@route`, `@mcp_name`, `@mcp_title`, `@mcp_hints`,
    `@noexpose`, `@nomcp`, …), which follow the `serve-api` tier below.
- **Not implemented:** the `?` error-propagation operator, which gives `PAR_UNEXPECTED_TOKEN` at
  HEAD, and typed quasiquotes. See [Known Limitations](/docs/reference/limitations).
- **Effect names** take the tier of the stdlib module that performs them. `IO`, `FS`, `Env`,
  `Clock` and `Rand` are Stable. The other capabilities that `--caps` accepts are Experimental.

## Stdlib surface

The module list comes from the frozen interface set: `make freeze-stdlib` writes one golden file
per top-level `std/*.ail` module into `.stdlib-golden/`, and `make verify-stdlib` passes at HEAD
(`✓ All 49 stdlib interfaces stable`). The gate also stops a dev push that changes a frozen
interface without a re-freeze. The two subdirectory modules, `std/ai/streaming` and
`std/stream/bridge`, are outside the freeze gate. They are tiered here as Experimental, so every
importable module appears exactly once.

For a **Stable** module, the promise covers the interface that the golden file records: export
names, types and effect rows. Changing a Stable module's golden in a way that is not additive is a
breaking change.

"Uses" below means files under `examples/` that import the module, counted at HEAD.

### Stable stdlib (17)

| Module | Purpose | Evidence |
|--------|---------|----------|
| `std/io` | Console input/output | 221 example files, 28 benchmarks; the core effect surface |
| `std/list` | Linked-list operations | 126 examples, 21 benchmarks |
| `std/string` | String manipulation | 106 examples, 20 benchmarks |
| `std/result` | Error handling (`Ok`/`Err`) | 65 examples |
| `std/option` | Optional values (`Some`/`None`) | 57 examples |
| `std/json` | JSON encoding and decoding | 34 examples |
| `std/bytes` | Byte-slice operations | 29 examples |
| `std/fs` | Sandboxed file system | 20 examples; capability-gated |
| `std/env` | Environment variables (secure snapshot) | 12 examples |
| `std/array` | Fixed-size arrays with O(1) access | 11 examples, 4 benchmarks |
| `std/math` | Mathematical functions | 8 examples |
| `std/map` | Maps with deterministic key order | 7 examples |
| `std/clock` | Time operations (deterministic virtual time) | 6 examples |
| `std/rand` | Random number generation | 5 examples |
| `std/iter` | Iteration control primitives | 4 examples |
| `std/datetime` | Pure date and time operations | 4 examples |
| `std/set` | Immutable sets with deterministic order | ⚠ close call — newer (3 examples, 1 benchmark), but its surface mirrors `std/map` |

### Experimental stdlib (31)

These may change in a 1.x minor, with a CHANGELOG entry.

| Module | Purpose | Why Experimental |
|--------|---------|------------------|
| `std/ai` | AI as a typed effect (`call`, `callJson`, `step`) | surface still changing along with the agent loop (20 examples) |
| `std/ai/streaming` | Token streaming for configured AI providers | subdirectory module, outside the freeze gate; narrow usage |
| `std/net` | HTTP client, plus pure `parseUrl` / `parseQuery` | ⚠ close call — widely used (17 examples), but security- and format-sensitive; the `Url` record shape may change |
| `std/web` | Web search and page fetch through a fixed backend | new; depends on one vendor's backend endpoint |
| `std/stream` | Async stream sources / `selectEvents` | close to concurrency; still changing |
| `std/stream/bridge` | WebSocket frame relay with an AILANG step | subdirectory module, outside the freeze gate; one example |
| `std/secret` | Gated secret resolution | newer effect |
| `std/process` | External command execution | ⚠ close call — host-boundary surface, tiered conservatively; the hosted executor is built on it |
| `std/crypto` | Cryptographic operations | ⚠ close call — security-sensitive, tiered conservatively |
| `std/jwt` | JWT parsing and verification | ⚠ close call — security-sensitive |
| `std/regex` | Linear-time (RE2) regex | RE2 subset; the `RegexMatch` record shape may change |
| `std/xml` | XML parsing and querying | ⚠ close call — the format surface may change |
| `std/yaml` | YAML to the `std/json` `Json` ADT | format surface; first-document-only semantics may widen |
| `std/html` | HTML5 parsing | the format surface may change |
| `std/zip` | ZIP archive read, write and stream | ⚠ close call — the format surface may change |
| `std/tar` | tar archive read and extract | the format surface may change |
| `std/gzip` | gzip compression | the format surface may change |
| `std/deflate` | Raw deflate (RFC 1951) and zlib (RFC 1950) | low-level and niche |
| `std/audio` | WAV/PCM boundaries and Ogg Opus encoding | new; niche |
| `std/terminal` | Native terminal host operations | new; host-specific scopes |
| `std/package` | Resolve files bundled in an installed package | the package-asset surface is still changing |
| `std/extension` | Helpers for AILANG extension packages | marked Experimental in its design doc |
| `std/cognition` | Cognitive-OS message fabric | marked Experimental in its design doc |
| `std/dom` | Cognitive-OS DOM substrate | marked Experimental in its design doc |
| `std/game` | Game-development utilities | marked Experimental in its design doc |
| `std/sem` | Semantic-caching types | marked Experimental in its design doc |
| `std/sharedmem` | Shared memory | marked Experimental in its design doc |
| `std/sharedindex` | Similarity-based semantic index | marked Experimental in its design doc |
| `std/embedding` | Vector-embedding utilities | niche |
| `std/simhash` | Locality-sensitive hashing | niche |
| `std/debug` | `Debug` ghost-effect logging | ⚠ close call — was Internal; moved because published packages (`sunholo/logging`) and the hosted executor depend on it |

### Internal stdlib (3)

No promise. Test and diagnostic helpers.

| Module | Purpose |
|--------|---------|
| `std/trace` | Custom trace span and event emission (telemetry-facing) |
| `std/trace_test` | Assertions over trace spans in tests |
| `std/smoke` | Helpers for package `_smoke.ail` boot probes |

## CLI surface

The rows come from the dispatch table in `cmd/ailang/commands.go`, which also generates the
[CLI Reference](/docs/reference/cli) (`make check-cli-docs` byte-compares the two). Every command
below appears in the `--help` of a binary built from the tree it was drafted on. The "Listed in"
column says which help screen it is on: `top` is `ailang --help`, and `dev`, `ops`, `eval` and
`pkg` are `ailang <group> --help`.

Tier rules for spellings:

- **A grouped command's top-level spelling has the same tier as the command.** `ailang ops messages`
  and `ailang messages` are one route.
- **A compatibility spelling** (a route the binary accepts but no longer lists, such as
  `ailang install` for `ailang pkg install`) **has the tier of its canonical command and keeps
  working through 1.x when that tier is Stable.** ⚠ close call.
- A Stable command's documented flags are Stable, apart from the Experimental flags listed under
  the table.

### Stable CLI (19)

| Command | Listed in | Also accepted | What it does |
|---------|-----------|---------------|--------------|
| `check` | top | `ai-check` | Type-check a file or directory; `--verify` adds contract verification |
| `docs` | top | — | Stdlib module documentation |
| `help` | top | — | Help for ailang, a group or a command |
| `init` | top | — | Scaffold a package (`init package`) or web app. ⚠ close call — it writes the Stable manifest; the web-app template may change |
| `prompt` | top | — | Emit the canonical teaching prompt |
| `repl` | top | — | Interactive REPL |
| `run` | top | — | Run a program (`--caps`, `--entry`, `--args-json`, `--args-file`) |
| `test` | top | — | Run tests (`--package`, `--json`) |
| `version` | top | — | Version, commit and build time |
| `pkg add` | pkg | `add` | Add a path, git or registry dependency |
| `pkg install` | pkg | `install` | Install a package (exact version or latest) and shim its `[bin]` commands |
| `pkg lock` | pkg | `lock` | Resolve dependencies and write `ailang.lock` (`--check` for read-only drift detection) |
| `pkg tree` | pkg | `tree` | Show the dependency tree |
| `pkg search` | pkg | `search` | Search the registry by keyword or tag |
| `pkg info` | pkg | — | Package detail: exports, effects, versions |
| `pkg versions` | pkg | — | Every version with its hashes |
| `pkg docs` | pkg | `pkg-docs` | Show a package's `AGENT.md` |
| `pkg publish` | pkg | `publish` | Publish the current package. ⚠ close call — the command and its manifest inputs are Stable, but the validator's `PUBnnn` gate set may tighten in a minor |
| `pkg bin` | pkg | `bin` | List or uninstall `[bin]` command shims. ⚠ close call — added in v0.40.0 |

**Experimental flags of Stable commands:** `run --bytecode` (the VM path), `run --policy`, `run
--ai*` and the `--routing-*` / `--allow-routing` family, `run --budget-report`, the resource-cap
flags (`--max-memory`, `--net-timeout`, `--net-max-bytes`, `--fs-max-bytes`,
`--process-allowlist`), and `check --format agent`. The `ai-check` alias is Stable as a route, but
the JSON report it prints is Experimental ⚠ close call, because its shape is still settling.

### Experimental CLI (23)

These may change in a 1.x minor, with a CHANGELOG entry.

| Command | Listed in | Also accepted | Why Experimental |
|---------|-----------|---------------|------------------|
| `examples` | top | — | tooling convenience |
| `fmt` | top | — | ⚠ close call — the flags (`--write`, `--check`) are settled, but the canonical layout can still change, which changes what `--check` reports |
| `lsp` | top | — | the editor and agent protocol surface is still changing |
| `verify` | top | — | ⚠ close call — verification coverage and known false counterexamples (bounded unrolling) are v1.1 work |
| `pkg quality` | pkg | — | the quality ladder is still growing gates |
| `pkg unpublish` | pkg | `unpublish` | the policy on immutable versions is still settling |
| `pkg history` | pkg | — | the provenance timeline is new |
| `pkg provenance` | pkg | — | the provenance chain is new |
| `pkg stats` | pkg | — | ecosystem statistics |
| `pkg affected-by` | pkg | — | package coordination over the message plane (which is Experimental) |
| `pkg notify-upgrade` | pkg | — | package coordination over the message plane |
| `pkg cascade` | pkg | — | the autonomous upgrade cascade |
| `messages` | ops | `msg` | **D-64 ruling.** Heavily used, still changing |
| `serve-api` | ops | — | ⚠ close call — the Stable MCP tools are served by it, and published packages deploy with it, but the flags and annotations are still growing (v0.51–v0.55) |
| `mcp` | ops | — | `ailang mcp check` connector readiness |
| `agent-prompt` | dev | — | ⚠ close call — the agent teaching prompt is versioned separately from `prompt` |
| `axioms` | dev | — | axiom scorecard; still changing |
| `compile` | dev | — | ⚠ close call — the emit-go backend is not a speed path and has no tail calls |
| `editor` | dev | — | tooling convenience |
| `export-training` | dev | — | depends on the trace format |
| `replay` | dev | — | depends on the trace format |
| `sandbox-check` | dev | — | close to diagnostics |
| `watch` | dev | — | tooling convenience |

### Internal CLI (46)

No promise. Diagnostic, harness, and fleet-operations surface.

| Command | Listed in | Also accepted | Purpose |
|---------|-----------|---------------|---------|
| `eval` | top | `eval run` | Benchmark runner (**D-64: eval tooling is Internal**) |
| `eval analyze` · `browser-profile` · `censored-pairs` · `compare` · `elo` · `matrix` · `paired` · `publish` · `report` · `run` · `suite` · `summary` · `sweet-spot` · `trend` | eval | `eval-*` top-level spellings, `browser-profile` | The eval harness family (14 subcommands) |
| `iface` | top | — | Normalised module-interface JSON. ⚠ close call — the freeze gate consumes it, but its JSON shape carries no promise; the promise is on the interface it describes |
| `serve` | top | `server` | Observatory dashboard server (**D-64: dashboards are Internal**) |
| `pkg key` | pkg | — | Scoped publish keys, superuser only |
| `ast-edit` | dev | — | Structural declaration edits |
| `builtins` | dev | — | Builtin-registry inspection |
| `cache` | dev | `brain` | Project brain and semantic cache |
| `debug` | dev | — | AST and type debugging |
| `devtools-prompt` | dev | — | Dev-tools reference for agents |
| `disasm` | dev | — | Disassemble compiled AILANG |
| `doctor` | dev | — | Environment diagnostics |
| `generate-extension-registry` | dev | — | Static extension-dispatch codegen |
| `micro-rag` | dev | `microrag`, `urag` | Micro-RAG index and retrieval |
| `pi` | dev | — | pi harness extensions |
| `policy-check` | dev | — | Evaluate a policy program against its cases |
| `policy-tool` | dev | — | Typed tool endpoint of the `ailang_only` lane |
| `select-best` | dev | — | Pick the best generated variant |
| `access-approvals` | ops | — | Durable access approvals (**D-64: approvals are Internal**) |
| `access-control` | ops | — | Access-control management |
| `budget` | ops | — | Budget monitoring |
| `chains` | ops | `trace`, `observatory`, `dashboard`, `eval-chains` | Chains, traces, observatory and dashboard (**D-64**) |
| `coordinator` | ops | — | Autonomous agent daemon (**D-64**) |
| `credit-gateway` | ops | — | Guarded Claude API credit gateway |
| `daemon` | ops | — | Local ailang daemon |
| `design-quorum` | ops | — | N-reviewer design quorum |
| `design-review` | ops | — | AI review of a design document |
| `exec` | ops | — | Unified AI execution for harnesses |
| `mission` | ops | — | Mission registry and iteration (**D-64**) |
| `models` | ops | — | Model registry |
| `rig-gate` | ops | — | GPU admission gateway |
| `storage` | ops | — | Storage-plane status |
| `workspaces` | ops | — | Workspace management |

The hidden routes `internal-dump-iface` / `dump-iface` are Internal. The 14 eval subcommands are
counted individually in the 46.

## Packages

The package system is part of the promise: bar v3 gate G3 replaces the v2 "orchestration
flagship" with it. This section covers **the format and resolution semantics**, meaning what
`ailang.toml`, `ailang.lock` and the registry mean. It does not cover the contents of any
particular package. Each package declares its own maturity in `[stability].level`. Reference:
the [Package System guide](/docs/guides/packages) and the
[publishing guide](/docs/guides/package-publishing).

### Stable package surface (12)

| Surface | What is promised |
|---------|------------------|
| Manifest `[package]` | `name` (two-level `vendor/name`), `version` (semver), `edition` (`"1"`), and the optional `ailang` (minimum version, enforced on install), `module_prefix` (one segment), `description` and `license` |
| Manifest `[exports].modules` | Only listed modules can be imported by dependents. Everything else is package-private |
| Manifest `[dependencies]` | Three sources: `{ path = … }`, `{ git = …, tag/rev, subdir }`, and a registry `"x.y.z"`. Versions are **exact**: there are no `^`/`~`/`>=` ranges. Resolution priority is path > git > registry |
| Flat resolution | One version per package name. A conflict fails `ailang lock` with a structured error, and the resolver never silently downgrades a direct dependency |
| **Authority: effect ceiling** `[effects].max` | Compilation fails if the package's own code declares, or references the type of, an effect outside `max`. Imported bodies are not charged. `max = []` means the package is pure |
| **Authority: ghost effects** | `Debug` is exempt from the ceiling, granted automatically, and erased by `--release` |
| **Authority: run-time grant** | A package gets no capability unless it is granted at run time (`run --caps`, or `[bin].caps`, which defaults to `auto` = the entry's declared effects). Declaring an effect does not grant it |
| Manifest `[stability].level` | `experimental` / `stable` / `frozen`: the package's own declared maturity |
| Lockfile `ailang.lock` | JSON with `schema: "ailang.lock/v1"`. Each package entry has `name`, `version`, `source` (`path`/`git`/`registry`), `content_hash`, `interface_hash`, `effects`, `exports` and the source locator. It is portable: path dependencies are recorded relative with forward slashes, and cache paths are resolved at run time. An unchanged resolution rewrites byte-identically, and `ailang lock --check` detects drift without writing |
| Dual hash model | `content_hash` changes with any source change. `interface_hash` changes only when exports or effects change, so internal refactors do not cascade |
| Import form | `import pkg/vendor/name/module (…)` resolves against the lockfile, never the local filesystem |
| Registry resolution | Default registry `https://storage.googleapis.com/ailang-registry` (`AILANG_REGISTRY` overrides it), read through `index.json`. `install x` / `x@latest` resolves once and writes the **exact** version into `ailang.toml`. A published version is immutable, and duplicate versions are rejected |

### Experimental package surface (9)

| Surface | Why Experimental |
|---------|------------------|
| `[bin]` commands (`module`, `entry`, `caps`, `run_flags`) | ⚠ close call — shipped in v0.40.0, used by `sunholo/email` / `eparse`; promote once the shim layout is settled |
| `[release].kind` and the `CHANGELOG.md` `## <version>` requirement | ⚠ close call — publish has required them since v0.41.0; part of the quality ladder that is still growing |
| Publish validator gates (`PUBnnn`, `ailang pkg quality`) | gates may tighten in a minor |
| Interface identity v2 hashing algorithm | ⚠ close call — the *meaning* of `interface_hash` is Stable, but the algorithm behind it may be re-versioned |
| Confined-agent authority (`AILANG_AGENT_POLICY`, `AILANG_PACKAGE_ROOT`) | ⚠ close call — under an agent policy, package verbs refuse network and `~/.ailang` writes with a `confined by AILANG_AGENT_POLICY` error (#1720); landed 2026-10-26 |
| `[dependencies_dev]` | undocumented in the package guide |
| `[ai_provider]`, `[extensions]`, `[assets]`, `[smoke]` | extension and provider wiring, still changing |
| `[cascade]` and the upgrade cascade (`pkg cascade`, `notify-upgrade`, `affected-by`) | runs over the message plane, which is Experimental (D-64) |
| `[metadata]` | free-form; no schema promised |

### Internal package surface (1)

| Surface | Purpose |
|---------|---------|
| Registry administration (`pkg key`, the `/rebuild-index` endpoint, `AILANG_REGISTRY_API_KEY`) | Operator-only |

### Worked example: `sunholo/mcp_oauth` ⚠ close call

[`sunholo/mcp_oauth`](https://ailang.sunholo.com/docs/packages/sunholo/mcp_oauth)
([source](https://github.com/sunholo-data/ailang-packages/tree/main/packages/mcp-oauth)) is an
OAuth 2.1 authorisation server for MCP services. It is published in the registry (latest
`v0.2.0`, four versions since 2026-10-03) and is in production use. Its consumer in this
repository is `executor_oauth/`, the sign-in server in front of the hosted executor MCP described
below. That consumer shows each Stable package rule above in a real file:

```toml
# executor_oauth/ailang.toml (abridged)
[package]
name = "sunholo/ailang_executor_oauth"
module_prefix = "executor_oauth"
version = "0.1.0"
edition = "1"
ailang = ">=0.54.0"

[exports]
modules = ["executor_oauth/oauth"]

[dependencies]
"sunholo/mcp_oauth" = "0.1.2"      # exact pin: v0.2.0 being published did not move it
"sunholo/firestore" = "0.7.2"
"sunholo/firebase_auth" = "0.1.2"

[effects]
max = ["IO", "FS", "Net", "Env", "Rand", "Clock", "Debug", "Declassify", "SharedMem"]
```

- **Lockfile.** `executor_oauth/ailang.lock` (`schema: "ailang.lock/v1"`) records
  `sunholo/mcp_oauth@0.1.2` from `source: "registry"`, with its `content_hash`,
  `interface_hash`, `effects` and `exports`. It also records the transitive
  `sunholo/gcp_auth@0.8.1` and `sunholo/config@0.1.2`. With the binary built from this branch,
  `ailang lock --check` reports `✓ ailang.lock is current`.
- **Import.** `import pkg/sunholo/mcp_oauth/flow (Hooks, beginAuthorize, …)` in
  `executor_oauth/oauth.ail`. `ailang check executor_oauth/oauth.ail` reports no errors.
- **Authority.** The consumer's ceiling has to name every effect that its own code references
  through the package's hook types. That is why it lists `SharedMem`, which it never performs,
  and the manifest says so in a comment.
- **Maturity is declared separately.** `ailang pkg info sunholo/mcp_oauth` shows
  `Stability: experimental`. That is the package author's promise about the package's API. It is
  separate from the 1.x promise about the format that carries it.

## Hosted MCP tool surface

The hosted AILANG executor (`docker/Dockerfile.executor-mcp`) runs `serve-api` over
`mcp_tools/` (the docs tools), `executor_tools/` (the code tools) and, on the OAuth deployment,
`executor_oauth/`. It also serves the built-in Go `submit_feedback` tool. It exposes two endpoints:
`/mcp/` (IAM-only, for Gemini Enterprise) and `/mcp/connect/` (OAuth 2.1, for connector
directories). The tool names below were read from the `@mcp_name` annotations at HEAD, and they
match the tool list a connected client sees (29 tools).

For a **Stable** tool, the promise covers:

- the tool name;
- the input parameter names and types;
- the read-only / idempotent hints;
- the top-level response envelope. Version-scoped docs tools return `{served_for, data}` or
  `{error, detail}`. Executor tools return `{ok, exit_code, stdout, stderr, truncated}` or
  `{ok: false, error, message}`.

The *content* inside a response, such as prompt text, docs pages or stdlib listings, changes with
every release by design. The **hosted URL is part of the promise:** the production OAuth endpoint
is `https://executor.ailang.sunholo.com/mcp/connect/`, and its OAuth issuer and protected-resource
metadata name that host. Raw `*.run.app` URLs, including the dev deployment's, are not.

### Stable MCP tools (24)

| Tool | Source | What it does |
|------|--------|--------------|
| `ailang_prompt` | `executor_tools/` | The teaching prompt; meant to be called first |
| `ailang_check` | `executor_tools/` | Type-check source and verify contracts. ⚠ close call — the tool is Stable, but its `stdout` carries the `ai-check` JSON report, which is Experimental (see CLI) |
| `ailang_run` | `executor_tools/` | Run source in the sandbox. Only `IO` is granted, and the policy timeout applies |
| `ailang_eval` | `executor_tools/` | Evaluate one pure expression with `show()` |
| `ailang_builtins` | `executor_tools/` | Builtins with docs, filtered by `search` |
| `ailang_versions` | `mcp_tools/language.ail` | Versions the docs tools can serve |
| `prompt_get` | `mcp_tools/language.ail` | Teaching prompt for a version and kind |
| `stdlib_modules` | `mcp_tools/language.ail` | Stdlib module list |
| `stdlib_module` | `mcp_tools/language.ail` | One module's interface and docs |
| `stdlib_search` | `mcp_tools/language.ail` | Search stdlib functions |
| `limitations_list` | `mcp_tools/language.ail` | Known limitations |
| `effects_catalog` | `mcp_tools/language.ail` | Effect and capability catalog |
| `examples_list` | `mcp_tools/examples.ail` | Examples by feature and status |
| `example_get` | `mcp_tools/examples.ail` | One example's source |
| `example_for_concept` | `mcp_tools/examples.ail` | Best example for a concept |
| `docs_nav` | `mcp_tools/discovery.ail` | Docs navigation tree |
| `docs_search` | `mcp_tools/discovery.ail` | Search the docs |
| `changelog_for_version` | `mcp_tools/discovery.ail` | Changelog for a version |
| `install_guide` | `mcp_tools/install.ail` | Install steps for a harness and platform |
| `onboarding_guide` | `mcp_tools/onboarding.ail` | ⚠ close call — the `role` vocabulary may change |
| `design_docs_list` | `mcp_tools/design.ail` | ⚠ close call — exposes the repo's internal design-doc layout (`state`) |
| `design_doc` | `mcp_tools/design.ail` | ⚠ close call — as above |
| `roadmap` | `mcp_tools/design.ail` | ⚠ close call — as above |
| `executor_whoami` | `executor_oauth/oauth.ail` | ⚠ close call — the signed-in account, served only on `/mcp/connect/` at `executor.ailang.sunholo.com` |

### Experimental MCP tools (5)

| Tool | Source | Why Experimental |
|------|--------|------------------|
| `benchmarks_list` | `mcp_tools/benchmarks.ail` | ⚠ close call — eval data (D-64 makes eval tooling Internal); the record shape follows the eval schema |
| `benchmark_run` | `mcp_tools/benchmarks.ail` | ⚠ close call — as above |
| `benchmarks_for_model` | `mcp_tools/benchmarks.ail` | ⚠ close call — as above |
| `benchmarks_compare` | `mcp_tools/benchmarks.ail` | ⚠ close call — as above |
| `submit_feedback` | built-in (`internal/apiserver/feedback_tool.go`) | ⚠ close call — the only write tool; rate-limited, and egress depends on deployment configuration |

## Known limitations & version promises

- [Known Limitations](/docs/reference/limitations) is live-verified: every entry carries a repro,
  a transcript, and the date it was verified.
- Language about future versions belongs only on the [Roadmap](/docs/roadmap). That is the one
  place where "planned for vX.Y" statements are allowed.

## Follow-ups (post-ratification)

- G2: close [#1443](https://github.com/sunholo-data/ailang/issues/1443) (`pure func`) and
  [#1134](https://github.com/sunholo-data/ailang/issues/1134) (IFC labels). Both affect rows on
  this page.
- A gate that checks this page's tables against the dispatch table, the stdlib goldens and the
  `@mcp_name` list, the way `check-cli-docs` checks the CLI Reference.
- Compile-time deprecation warnings for Experimental items on their way to removal (post-v1).

---

*Ratification status: **ratified 2026-11-09 (Mark, D-65)**. First drafted at mission iteration 5
(2026-07-10); re-drafted and ratified 2026-11-09 against bar v3 (D-64).*
