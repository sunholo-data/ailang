---
sidebar_position: 4
title: Stability Promise (1.x)
description: What syntax, stdlib, CLI, package format and hosted MCP tool surface AILANG guarantees across the 1.x release line.
reviewed: 2026-11-26
reviewBy: 2027-05-26
---

# AILANG 1.x Stability Promise

> **Ratified 2026-11-09.** Tier semantics follow [M-V1-STABILITY-PROMISE](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_29_0/m-v1-stability-promise.md), verified against `cmd/ailang`, `.stdlib-golden/`, `internal/pkg`, and the hosted MCP tool surface. Rows marked **⚠ close call** explain judgment calls.

## Tiers

| Tier | Promise | Marked in docs |
|------|---------|----------------|
| **Stable** | Covered by the 1.x promise; breaking changes require a **major** bump (`2.0`). Deprecations last at least one minor. | Default (listed below) |
| **Experimental** | Shipped and usable; **may change in a 1.x minor** with a CHANGELOG migration note. | `Experimental` banner / table |
| **Internal** | **No promise.** May change or be removed at any time. | `Internal` banner / table |

Additive changes (new modules, functions, optional flags, manifest keys, or MCP tools) may land in any minor. Items are drafted **Stable** when documented, exercised by `examples/`/evals/packages, and free of P0 defects.

### Go packages

AILANG is a Go module so the binary can be built with `go install`. Its importable Go packages (`serveapi/...`, `std`, `examples`) are **Internal**. They are not covered by the 1.x promise and may change in any release, whatever Go's semantic-import-versioning convention suggests for a v1 tag. If you import them, pin an exact version.

### What ships in the binary

The `ailang` binary also contains the Internal tier: the coordinator, mission loops, eval harness, dashboards and the fleet tooling we use to build AILANG. It ships in the same binary today for convenience. During 1.x we intend to move it behind a separate Go module or binary. That is not a breaking change. Internal surface carries no promise, and every Stable and Experimental command, stdlib module and package format stays in `ailang`. If you depend on an Internal command, expect it to move.

## At a glance

| Area | Stable | Experimental | Internal | Source of truth |
|------|-------:|-------------:|---------:|-----------------|
| Stdlib modules | 17 | 31 | 3 | `std/STABILITY.toml` (module tiers) + `@experimental` (25 functions) → `.stdlib-golden/` |
| CLI commands | 19 | 23 | 46 | `cmd/ailang` dispatch table → [CLI Reference](/docs/reference/cli) |
| Package surface | 12 | 9 | 1 | `internal/pkg` manifest + lockfile code |
| Hosted MCP tools | 24 | 5 | 0 | `mcp_tools/`, `executor_tools/`, `executor_oauth/`, built-in `submit_feedback` |

## Syntax surface

Canonical syntax is defined by `ailang prompt` (`ailang prompt --version-active`) and [Language Syntax](/docs/reference/language-syntax).

- **Stable:** modules and imports (incl. `pkg/vendor/name/...`), `func`/`pure func` (`! {}`)/`export`, `let` and `{ …; … }`, lambdas (`\x. …`), `if … then … else …`, brace-form `match` with guards, ADTs, row-polymorphic records, `! {IO, FS, …}`, `::`/`++`, `"${expr}"` interpolation, `requires`/`ensures` contracts, and Stable effect capabilities (`IO`, `FS`, `Env`, `Clock`, `Rand`).
- **Error & option propagation (`let x = expr?`):** Unwraps `Ok`/`Some` on the return spine of a `Result`/`Option` function (`PAR_TRY_LET_ONLY` outside `let` on the spine).
- **Experimental syntax:** parameterised effects (`Net[scope=public]`, `@limit=N` — [Parameterised effects](/docs/guides/parameterised-effects)), IFC labels and `Declassify`, `serve-api` export annotations (`@route`, `@mcp_*`, `@noexpose`, `@nomcp`), and Experimental capabilities.
- **Retired / not implemented:** `match … with | …` emits `PAR019`; typed quasiquotes are not implemented ([Known Limitations](/docs/reference/limitations)).

## Stdlib surface

`make freeze-stdlib` writes `.stdlib-golden/` for top-level `std/*.ail` modules (`make verify-stdlib` checks `49` stdlib interfaces); `std/ai/streaming` and `std/stream/bridge` are subdirectory Experimental modules.

- **Stable contract:** export names, types, class constraints (`"constraints"`), literal-carrying numeric variables (`"literals"`), and effect rows.
- **Tier metadata:** module tiers live in [`std/STABILITY.toml`](https://github.com/sunholo-data/ailang/blob/dev/std/STABILITY.toml) and per-function `@experimental` annotations surface in `ailang iface` and `ailang docs`.
- **`show` guarantee:** `show` is total, deterministic across evaluator and VM, and injective per type without NaN or functions (top-level strings unquoted; nested strings escaped; functions print `<function>`). `std/string.showPreview` is Stable with `std/string` (`…(+N chars)`).

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

### Experimental functions in Stable modules (25)

| Module | Functions | Why Experimental |
|--------|-----------|------------------|
| `std/json` | `repair`, `asBoolLoose`, `getStringArrayStrict`, `getNumberArrayStrict`, `decodeFloatArray` | boundary and niche helpers; no downstream use |
| `std/string` | `decodeQuotedPrintable`, `foldSlices`, `mapSlicesJoin`, `startsWithIgnoreCase` | niche; `foldSlices`/`mapSlicesJoin` take the function last, unlike every other higher-order export, and may be reordered |
| `std/list` | `takeMap`, `takeFlatMap` | workarounds for strict evaluation; redundant if fusion lands |
| `std/array` | `dot`, `axpy`, `scale`, `add`, `sub`, `mul`, `sum`, `argmax`, `scatterAdd`, `accumulate`, `encodeF64LE`, `decodeF64LE`, `encodeF32LE`, `decodeF32LE` | the numerics kernels, still growing; `axpy`/`scale` keep BLAS argument order |

### Deprecated stdlib names

Sixteen legacy 0.6x aliases were **removed in `v0.64.0`**; importing any removed alias fails with `IMP010` naming its replacement (run `sed -E -f tools/migrate-stdlib-renames.sed` for the 12 mechanical renames, and update `glob`, `slice`, `member`, `unsafeGet` by hand).

| Module | Deprecated | Use instead |
|--------|------------|-------------|
| `std/list` | `head_or`, `last_or`, `nth_or` | `headOr`, `lastOr`, `nthOr` |
| `std/list` | `member(x, xs)` | `contains(xs, x)` — the arguments swap |
| `std/string` | `charAt_or` | `charAtOr` |
| `std/math` | `abs_Int`, `abs_Float` | `absInt`, `absFloat` |
| `std/rand` | `rand_int`, `rand_float`, `rand_bool`, `rand_seed` | `randInt`, `randFloat`, `randBool`, `randSeed` |
| `std/fs` | `readFileBytes` | `readFileBase64` (it returns base64 text; `readFileRaw` returns bytes) |
| `std/fs` | `glob` | `walkWithSuffix` (a suffix filter, not a glob; the name stays retired for 1.x) |
| `std/io` | `printErr` | `eprint` (no newline; `eprintln` adds one) |
| `std/bytes` | `slice(b, start, len)` | `sliceLen` (the third argument is a length; the name stays retired for 1.x) |
| `std/array` | `unsafeGet` | `get` (identical behaviour) |

User packages may still apply Experimental `@deprecated("replacement")` (`TC_DEPRECATED_001`) and `@experimental` annotations to their own exports.

### Conventions of the Stable stdlib

- **Naming:** lowerCamelCase (exceptions: `PI()`, `E()`, `Json` builders `jn`/`jb`/`jnum`/`jint`/`js`/`ja`/`jo`/`kv`, and `intToStr`/`floatToStr` vs `stringToInt`/`stringToFloat`).
- **Argument order:** data first (`nth(xs, i)`, `map.lookup(m, k)`), except HOFs take function first (`map(f, xs)`, `foldl(f, acc, xs)`) and `list.take`/`list.drop`/`string.join`.
- **`Json` numbers:** `decode` produces `JInt(n)` for int64 literals and `JNumber(x)` otherwise; read numbers via `asNumber`/`getNumber`/`getInt`.

### Internal stdlib (3)

| Module | Purpose |
|--------|---------|
| `std/trace` | Custom trace span and event emission (telemetry-facing) |
| `std/trace_test` | Assertions over trace spans in tests |
| `std/smoke` | Helpers for package `_smoke.ail` boot probes |

## CLI surface

Command tiers match `cmd/ailang/commands.go` and [CLI Reference](/docs/reference/cli) (`make check-cli-docs`). Grouped top-level aliases and compatibility spellings (`ailang install` → `ailang pkg install`) inherit their canonical command's tier.

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

- **Experimental flags on Stable commands:** `run --bytecode`, `run --policy`, `--ai*` / `--routing-*` / `--allow-routing`, `--budget-report`, resource caps (`--max-memory`, `--net-*`, `--fs-max-bytes`, `--process-allowlist`), `check --format agent`, and `ai-check` JSON report schema (`--no-verify`).
- **Stable contract behaviour:** `check --verify` exit codes follow [Contracts Guide](/docs/guides/contracts#exit-codes); `ailang run` checks contracts only with `--verify-contracts` (evaluator only; refused with `CTR004` on `--bytecode`).

### Experimental CLI (22)

| Command | Listed in | Also accepted | Why Experimental |
|---------|-----------|---------------|------------------|
| `examples` | top | — | tooling convenience |
| `fmt` | top | — | ⚠ close call — the flags (`--write`, `--check`) are settled, but the canonical layout can still change, which changes what `--check` reports |
| `lsp` | top | — | the editor and agent protocol surface is still changing |
| `verify` | top | — | ⚠ close call — the verdict and exit contract are settled (a bounded-unrolling model is `inconclusive`, not a violation, since v0.57.5), but coverage is still growing: call-site `requires`, int64 overflow, HOF callers and induction are open |
| `pkg quality` | pkg | — | the quality ladder is still growing gates |
| `pkg unpublish` | pkg | `unpublish` | the policy on immutable versions is still settling |
| `pkg history` | pkg | — | the provenance timeline is new |
| `pkg provenance` | pkg | — | the provenance chain is new |
| `pkg stats` | pkg | — | ecosystem statistics |
| `pkg affected-by` | pkg | — | package coordination over the message plane (which is Experimental) |
| `pkg notify-upgrade` | pkg | — | package coordination over the message plane |
| `pkg cascade` | pkg | — | the autonomous upgrade cascade |
| `messages` | ops | `msg` | Heavily used, still changing |
| `serve-api` | ops | — | ⚠ close call — the Stable MCP tools are served by it, and published packages deploy with it, but the flags and annotations are still growing (v0.51–v0.55) |
| `mcp` | ops | — | `ailang mcp check` connector readiness |
| `agent-prompt` | dev | — | ⚠ close call — the agent teaching prompt is versioned separately from `prompt` |
| `axioms` | dev | — | axiom scorecard; still changing |
| `editor` | dev | — | tooling convenience |
| `export-training` | dev | — | depends on the trace format |
| `replay` | dev | — | depends on the trace format |
| `sandbox-check` | dev | — | close to diagnostics |
| `watch` | dev | — | tooling convenience |

*(Removed: `ailang compile --emit-go` was retired after `v0.60.5`; use `ailang run` / `--bytecode`.)*

### Internal CLI (46)

| Command | Listed in | Also accepted | Purpose |
|---------|-----------|---------------|---------|
| `eval` | top | `eval run` | Benchmark runner (eval tooling is Internal) |
| `eval analyze` · `browser-profile` · `censored-pairs` · `compare` · `elo` · `matrix` · `paired` · `publish` · `report` · `run` · `suite` · `summary` · `sweet-spot` · `trend` | eval | `eval-*` top-level spellings, `browser-profile` | The eval harness family (14 subcommands) |
| `iface` | top | — | Normalised module-interface JSON. ⚠ close call — the freeze gate consumes it, but its JSON shape carries no promise; the promise is on the interface it describes |
| `serve` | top | `server` | Observatory dashboard server (dashboards are Internal) |
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
| `access-approvals` | ops | — | Durable access approvals (approvals are Internal) |
| `access-control` | ops | — | Access-control management |
| `budget` | ops | — | Budget monitoring |
| `chains` | ops | `trace`, `observatory`, `dashboard`, `eval-chains` | Chains, traces, observatory and dashboard |
| `coordinator` | ops | — | Autonomous agent daemon |
| `credit-gateway` | ops | — | Guarded Claude API credit gateway |
| `daemon` | ops | — | Local ailang daemon |
| `design-quorum` | ops | — | N-reviewer design quorum |
| `design-review` | ops | — | AI review of a design document |
| `exec` | ops | — | Unified AI execution for harnesses |
| `mission` | ops | — | Mission registry and iteration |
| `models` | ops | — | Model registry |
| `rig-gate` | ops | — | GPU admission gateway |
| `storage` | ops | — | Storage-plane status |
| `workspaces` | ops | — | Workspace management |

*(Hidden `internal-dump-iface` / `dump-iface` routes are Internal; the 14 `eval` subcommands are counted individually in the 46.)*

## Packages

The 1.x promise covers manifest and lockfile **format and resolution semantics** ([Packages Guide](/docs/guides/packages), [Publishing Guide](/docs/guides/package-publishing)); individual packages declare their own maturity in `[stability].level`.

### Stable package surface (12)

| Surface | What is promised |
|---------|------------------|
| Manifest `[package]` | `name` (`vendor/name`), `version` (semver), `edition` (`"1"`), optional `ailang` min version, `module_prefix`, `description`, `license` |
| Manifest `[exports].modules` | Only listed modules are public; all others are package-private |
| Manifest `[dependencies]` | Exact pins only: `{ path = … }`, `{ git = …, tag/rev, subdir }`, or `"x.y.z"` (priority: path > git > registry) |
| Flat resolution | One version per package name; conflicts fail `ailang lock` explicitly |
| **Authority: effect ceiling** `[effects].max` | Compilation fails if package code declares or references an effect outside `max` (`max = []` = pure) |
| **Authority: ghost effects** | `Debug` is exempt from `[effects].max` and erased by `--release` |
| **Authority: run-time grant** | Capabilities come only from `run --caps` or `[bin].caps`. `--caps auto` grants the entry's declared effects (never `Process`) and discloses them to `stderr` unless `--quiet` is explicit; `[bin].caps` defaults to `auto`, resolved once at install, printed, and pinned in the shim (widening requires `--grant-new-caps`). `run` refuses missing capabilities pre-flight (`E_CAPS_MISSING`) before executing user code. |
| Manifest `[stability].level` | `experimental` / `stable` / `frozen` package-declared maturity |
| Lockfile `ailang.lock` | `schema: "ailang.lock/v1"` JSON recording hashes, effects, exports, and portable relative locators; `ailang lock --check` verifies drift |
| Dual hash model | `content_hash` tracks source bytes; `interface_hash` tracks exports/effects to prevent internal-refactor cascades |
| Import form | `import pkg/vendor/name/module (…)` resolves strictly via `ailang.lock` |
| Registry resolution | Default `https://storage.googleapis.com/ailang-registry` (`AILANG_REGISTRY` overrides); published versions are immutable |

### Experimental package surface (9)

| Surface | Why Experimental |
|---------|------------------|
| `[bin]` commands (`module`, `entry`, `caps`, `run_flags`) | ⚠ close call — used by `sunholo/email` / `eparse`; promote once shim layout settles |
| `[release].kind` and `CHANGELOG.md` `## <version>` | quality ladder still growing |
| Publish validator gates (`PUBnnn`, `ailang pkg quality`) | gates may tighten in a minor |
| Interface identity v2 hashing algorithm | ⚠ close call — hash meaning is Stable, algorithm may re-version |
| Confined-agent authority (`AILANG_AGENT_POLICY`, `AILANG_PACKAGE_ROOT`) | offline/sandbox policy still hardening |
| `[dependencies_dev]` | undocumented in package guide |
| `[ai_provider]`, `[extensions]`, `[assets]`, `[smoke]` | extension/provider wiring still changing |
| `[cascade]` and `pkg cascade` / `notify-upgrade` / `affected-by` | runs over Experimental message plane |
| `[metadata]` | free-form; no schema promised |

### Internal package surface (1)

| Surface | Purpose |
|---------|---------|
| Registry administration (`pkg key`, `/rebuild-index`, `AILANG_REGISTRY_API_KEY`) | Operator-only |

### Worked example: `sunholo/mcp_oauth`

[`sunholo/mcp_oauth`](https://ailang.sunholo.com/docs/packages/sunholo/mcp_oauth) is consumed by `executor_oauth/` with exact pins, `[effects].max` covering referenced hook types (including `SharedMem`), and `ailang lock --check` verification:

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
"sunholo/mcp_oauth" = "0.1.2"
"sunholo/firestore" = "0.7.2"
"sunholo/firebase_auth" = "0.1.2"

[effects]
max = ["IO", "FS", "Net", "Env", "Rand", "Clock", "Debug", "Declassify", "SharedMem"]
```

## Hosted MCP tool surface

Served by `serve-api` over `mcp_tools/`, `executor_tools/`, `executor_oauth/`, and built-in `submit_feedback` at `/mcp/` (IAM) and `/mcp/connect/` (OAuth 2.1, production host `https://executor.ailang.sunholo.com/mcp/connect/`).

For **Stable** tools, the promise covers tool name, input schema, hints, and response envelope (`{served_for, data}` / `{ok, exit_code, stdout, stderr, truncated}` / `ailang_check`'s `verification` field).

### Stable MCP tools (24)

| Tool | Source | What it does |
|------|--------|--------------|
| `ailang_prompt` | `executor_tools/` | The teaching prompt, in two parts (`part`: `core`, `reference`); `core` is meant to be called first |
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
| `benchmarks_list` | `mcp_tools/benchmarks.ail` | ⚠ close call — eval data (eval tooling is Internal); the record shape follows the eval schema |
| `benchmarks_for_model` | `mcp_tools/benchmarks.ail` | ⚠ close call — as above |
| `benchmarks_compare` | `mcp_tools/benchmarks.ail` | ⚠ close call — as above |
| `submit_feedback` | built-in (`internal/apiserver/feedback_tool.go`) | ⚠ close call — the only write tool; rate-limited, and egress depends on deployment configuration |

## Support Window

| Release line | Bug fixes | Security fixes | Notes |
|---|---|---|---|
| **Latest `1.x` minor** | Yes | Yes | Upgrade to latest `1.x` minor (non-breaking by promise). |
| **Final `0.x` (`v0.64.x`)** | No | **90 days after `1.0.0`** | Critical/high security fixes only during the 90-day transition window. |
| **Older `1.x` / earlier `0.x`** | No | No | Upgrade to latest `1.x`. |

See [SECURITY.md](https://github.com/sunholo-data/ailang/blob/dev/SECURITY.md) for reporting channels and [Known Limitations](/docs/reference/limitations) / [Roadmap](/docs/roadmap) for current bounds and future work.

*Ratification status: **ratified 2026-11-09**.*
