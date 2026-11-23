---
sidebar_position: 4
title: Stability Promise (1.x)
description: What syntax, stdlib, CLI, package format and hosted MCP tool surface AILANG guarantees across the 1.x release line.
reviewed: 2026-11-23
reviewBy: 2027-05-23
---

# AILANG 1.x Stability Promise

> **Ratified 2026-11-09.** The tier *semantics* below are defined by the
> [M-V1-STABILITY-PROMISE design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/implemented/v0_29_0/m-v1-stability-promise.md).
> The tier *assignments* reflect the current surface: the dispatch table behind
> `ailang --help` (17 visible commands plus the `dev`, `ops`, `eval`
> and `pkg` groups), the frozen stdlib interfaces (`make freeze-stdlib`), the package manifest and
> lockfile code, and the hosted executor's MCP tool list. Rows marked
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

**Tiering decisions:** `ailang messages` is **Experimental**. The
coordinator, missions, approvals, dashboards, eval tooling and chains/observatory are
**Internal**. The cloud executor's MCP tool surface is **Stable**.

## At a glance

| Area | Stable | Experimental | Internal | Source of truth |
|------|-------:|-------------:|---------:|-----------------|
| Stdlib modules | 17 | 31 | 3 | `std/STABILITY.toml` (module tiers) + `@experimental` (25 functions) → `.stdlib-golden/` |
| CLI commands | 19 | 23 | 46 | `cmd/ailang` dispatch table → [CLI Reference](/docs/reference/cli) |
| Package surface | 12 | 9 | 1 | `internal/pkg` manifest + lockfile code |
| Hosted MCP tools | 24 | 5 | 0 | `mcp_tools/`, `executor_tools/`, `executor_oauth/`, built-in `submit_feedback` |

## Syntax surface

The Stable syntax set is defined **by reference** to the canonical teaching prompt and language
reference rather than listed again here, so there is one source of truth:

- **Canonical reference:** `ailang prompt` and the [Language Syntax](/docs/reference/language-syntax)
  page. Read the active prompt revision from `ailang prompt --version-active` (`prompts/versions.json`
  `.active`), not from this page.
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

- **`pure func` is Stable** (`pure` means exactly `! {}`; enforced by the type checker since #1443).
- **Retired syntax is not Stable and the parser rejects it.** For example, the ML/Haskell form
  `match … with | …` produces `PAR019` ("'match ... with' is not valid AILANG syntax"). Use the
  brace form `match x { pat => expr, … }`.
- **Experimental syntax:**
  - Parameterised effects and capability refinement, such as `Net[scope=public]` and `@limit=N`
    (see [Parameterised effects](/docs/guides/parameterised-effects)).
  - IFC labels and `Declassify` (cross-module label checking landed in #1134 / #2185).
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
names, types, class constraints (`"constraints": ["Eq[a]"]`, listed only when an export has any,
with the type's variable names), literal-carrying numeric variables (`"literals": ["a"]`, #2378,
listed only when there are any) and effect rows. Changing a Stable module's golden in a way that is
not additive is a breaking change; adding a constraint to an export is not additive.

**Where tiers are recorded.** One source per fact: the tier of every module is in
[`std/STABILITY.toml`](https://github.com/sunholo-data/ailang/blob/dev/std/STABILITY.toml), and a
function that is Experimental inside a Stable module says so in its source with `@experimental`.
`ailang iface std/<module>` stamps the result on every export as `"tier"`, and the goldens in
`.stdlib-golden/` record it. `ailang docs` shows the same marks. The tables on this page are checked
against both by a test, so they cannot drift. Marking a function changes neither its type nor any
package's `interface_hash`.

"Uses" below means files under `examples/` that import the module, counted at HEAD.

**The text `show` returns is part of the Stable promise** (the `show` builtin is Stable): it is
total (nothing elided, at any width or depth), deterministic (the same on the evaluator and the
bytecode VM, on every OS), and injective per type for values without NaN or functions. A
top-level string is returned unchanged; a string nested in a list, tuple, record, map or
constructor is quoted with the escapes `ailang fmt` uses, and that literal reads back as the same
string through the normal loader (valid UTF-8 only). Function values of every kind, constructors
used as values included, print `<function>` on every engine. `std/string.showPreview` is Stable
with `std/string`: it quotes a string even at the top level, keeps a head-only prefix of at most
`maxChars` code points, and marks a cut with `…(+N chars)` (N exact), so a cut preview never
equals a complete one. The marker is for people, not for parsing.

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

### Experimental functions in Stable modules (25)

These functions are in Stable modules but are Experimental themselves (`@experimental`): they may
change in a 1.x minor, with a CHANGELOG entry.

| Module | Functions | Why Experimental |
|--------|-----------|------------------|
| `std/json` | `repair`, `asBoolLoose`, `getStringArrayStrict`, `getNumberArrayStrict`, `decodeFloatArray` | boundary and niche helpers; no downstream use |
| `std/string` | `decodeQuotedPrintable`, `foldSlices`, `mapSlicesJoin`, `startsWithIgnoreCase` | niche; `foldSlices`/`mapSlicesJoin` take the function last, unlike every other higher-order export, and may be reordered |
| `std/list` | `takeMap`, `takeFlatMap` | workarounds for strict evaluation; redundant if fusion lands |
| `std/array` | `dot`, `axpy`, `scale`, `add`, `sub`, `mul`, `sum`, `argmax`, `scatterAdd`, `accumulate`, `encodeF64LE`, `decodeF64LE`, `encodeF32LE`, `decodeF32LE` | the numerics kernels, still growing; `axpy`/`scale` keep BLAS argument order |

### Deprecated stdlib names

A function marked `@deprecated("replacement")` keeps working, with the same type, effects and
behaviour, and every use of it from another module gives the warning `TC_DEPRECATED_001` naming the
replacement — on stderr, in `check --json` `warnings[]` and in `check --format agent`, on a warm
compile cache too. Deprecation is not a tier: a deprecated Stable function stays Stable until it is
removed. These 16 names were renamed or deprecated before 1.0 and are **removed before the v1.0.0
soak**, so they are not part of the 1.x surface. After removal, importing one still names the
replacement (`IMP010`).

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

`sed -E -f tools/migrate-stdlib-renames.sed` rewrites the unique names; edit `glob`, `slice`,
`member` and `unsafeGet` by hand, then run `ailang check` until no `TC_DEPRECATED_001` is left.

**Migration order.** None of the 14 new names (`headOr` … `sliceLen`) exists in v0.59.0 or
v0.60.0, so code pinned to those releases cannot use them: a project moves to them only once
its pin is on the first release that contains them. `member(x, xs)` → `contains(xs, x)` and
`unsafeGet` → `get` use names that already exist, so those two can change at any time. The
order is: (1) release the new names, with the old ones warning; (2) downstream projects move
their pins and migrate until `check` shows no `TC_DEPRECATED_001`; (3) the aliases are removed
in the last 0.x release before the v1.0.0 soak (roadmap step 16).

**Where the warning appears.** Every use from another module — in function bodies, contracts,
named `test` blocks, `property` blocks, inline `tests` rows and module-level `let`s — in
`ailang check` (stderr, `--json`, `--format agent`, `--package`), `ailang run` (stderr) and
`ailang test` (stderr). Uses inside an installed dependency package are reported by that
package's own `check`, not by its importers. A deprecation can only ever produce
`TC_DEPRECATED_001` in code that uses the name: a std function deprecated in a later release therefore gives an unchanged library warnings only, never an error, and those warnings are not reported inside an installed dependency.
When a replacement is deprecated in turn, the warning names the first live function along the
chain. The command-line REPL cannot import std modules
(`:import std/list` is "Unknown module"), so a deprecated std name cannot be reached there;
modules loaded into the browser REPL's registry do not warn.
Package authors may use `@deprecated` and `@experimental` too; the annotation syntax itself is
Experimental in 1.0.

### Conventions of the Stable stdlib

- **Names** are lowerCamelCase. The exceptions are kept on purpose: the mathematical constants
  `PI()` and `E()`, the terse `Json` builders (`jn`, `jb`, `jnum`, `jint`, `js`, `ja`, `jo`, `kv`),
  and the `intToStr`/`floatToStr` vs `stringToInt`/`stringToFloat` spellings.
- **Argument order** is data first: the collection, string, path or `Json` value comes first
  (`nth(xs, i)`, `map.lookup(m, k)`, `set.member(s, x)`), except that a higher-order function takes
  the function first and the traversed collection last (`map(f, xs)`, `foldl(f, acc, xs)`).
  `list.take(n, xs)`, `list.drop(n, xs)` and `string.join(sep, xs)` are documented exceptions.
- **`Json` numbers.** `decode` gives `JInt(n)` for an integer literal that fits in 64 bits and
  `JNumber(x)` for every other number; `encode` prints a `JInt` exactly. The numeric value
  round-trips through text and the constructor does not: a whole-number `JNumber` prints as `3`,
  so `decode(encode(jnum(3.0)))` is `JInt(3)`. Read numbers with `asNumber`/`getNumber` (both
  constructors) or `getInt` (`None` for a fractional value, never truncated).

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

**Stable behaviour of those commands, stated so it is not frozen by omission:**
- The exit codes of `check --verify` (and `ai-check`) follow the table in the
  [contracts guide](/docs/guides/contracts#exit-codes): 0 for verified, no obligations, or partial (at
  least one obligation proved and the skipped ones listed on stderr and in `results`); 1 for a failed
  check, nothing proved (`unproved`: every obligation skipped), a counterexample, a solver error or
  timeout, an inconclusive result, or obligations with no solver. `--allow-inconclusive` makes an
  inconclusive result exit 0 even when it is the only obligation (nothing proved); the outcome stays
  `inconclusive`. An obligation is a function with an `ensures` clause; a `requires`-only function, or
  one whose only `ensures` is the literal `ensures { true }`, is not one. Other tautologies (`1 == 1`,
  `result == result`, `true && true`) are not detected and count as verified. Contradictory flags
  (`--strict` with `--allow-inconclusive`, either with `--no-verify`, `--strict` without `--verify`)
  are usage errors (exit 2). The JSON `outcome` field may gain values in a minor; existing values keep
  their meaning. `ai-check --no-verify` (type check only, outcome `not_requested`) is Experimental.
- `ailang run` does not check `requires`/`ensures` unless `--verify-contracts` is given. With
  `--bytecode`/`--strict-bytecode` that flag is refused (CTR004, exit 2) until the VM enforces contracts;
  lifting the refusal is additive.

### Experimental CLI (22)

These may change in a 1.x minor, with a CHANGELOG entry.

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

**Removed: `compile`.** `ailang compile --emit-go` / `--emit-go-v2` (the Go emitter) was removed
after v0.60.5, so that 1.x has two engines that agree: the evaluator and the bytecode VM. The
`compile` route is still accepted, but only to fail (exit 1) with a message pointing at
`ailang run`, plus `--bytecode` for the VM.

### Internal CLI (46)

No promise. Diagnostic, harness, and fleet-operations surface.

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

The hidden routes `internal-dump-iface` / `dump-iface` are Internal. The 14 eval subcommands are
counted individually in the 46.

## Packages

The package system is part of the 1.x promise. This section covers **the format and resolution semantics**, meaning what
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
| Interface identity v2 hashing algorithm | ⚠ close call — the *meaning* of `interface_hash` is Stable, but the algorithm behind it may be re-versioned. The v2 hash covers exports' class constraints since #2290, and their literal-carrying numeric variables since #2378, so it differs across that release for a package with a constrained export, and a publisher and validator on either side of it disagree (`PUB005`): the validator is deployed in step with the release. The lockfile `interface_hash` (v1, manifest-only) is unaffected |
| Confined-agent authority (`AILANG_AGENT_POLICY`, `AILANG_PACKAGE_ROOT`) | ⚠ close call — under an agent policy, package verbs refuse network and `~/.ailang` writes with a `confined by AILANG_AGENT_POLICY` error (#1720); landed 2026-10-26 |
| `[dependencies_dev]` | undocumented in the package guide |
| `[ai_provider]`, `[extensions]`, `[assets]`, `[smoke]` | extension and provider wiring, still changing |
| `[cascade]` and the upgrade cascade (`pkg cascade`, `notify-upgrade`, `affected-by`) | runs over the message plane, which is Experimental |
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
  `{ok: false, error, message}`. `ailang_check` adds two
  fields to that envelope: `verification` (the verification result: the report's `verify.outcome`,
  or `unavailable` when the host has no solver and ran the type check only) and, only when it is
  `unavailable`, `verification_message`. `ok` is still `exit_code == 0`, and `exit_code` is still the
  real status of the command the tool ran.

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
| `benchmarks_list` | `mcp_tools/benchmarks.ail` | ⚠ close call — eval data (eval tooling is Internal); the record shape follows the eval schema |
| `benchmarks_for_model` | `mcp_tools/benchmarks.ail` | ⚠ close call — as above |
| `benchmarks_compare` | `mcp_tools/benchmarks.ail` | ⚠ close call — as above |
| `submit_feedback` | built-in (`internal/apiserver/feedback_tool.go`) | ⚠ close call — the only write tool; rate-limited, and egress depends on deployment configuration |

## Known limitations & version promises

- [Known Limitations](/docs/reference/limitations) is live-verified: every entry carries a repro,
  a transcript, and the date it was verified.
- Language about future versions belongs only on the [Roadmap](/docs/roadmap). That is the one
  place where "planned for vX.Y" statements are allowed.

## Follow-ups (post-ratification)

- ~~Close [#1443](https://github.com/sunholo-data/ailang/issues/1443) (`pure func`) and [#1134](https://github.com/sunholo-data/ailang/issues/1134) (IFC labels)~~: done (PRs `#2184` and `#2185`).
- A gate that checks this page's tables against the dispatch table, the stdlib goldens and the
  `@mcp_name` list, the way `check-cli-docs` checks the CLI Reference.
- ~~Compile-time deprecation warnings~~: done — `@deprecated` and `TC_DEPRECATED_001` (see
  [Deprecated stdlib names](#deprecated-stdlib-names)).

---

*Ratification status: **ratified 2026-11-09**.*
