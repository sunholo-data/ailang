---
sidebar_position: 4
title: Stability Promise (1.x)
description: What syntax, stdlib, and CLI surface AILANG guarantees across the 1.x release line.
---

# AILANG 1.x Stability Promise

> **RATIFICATION: pending (Mark, at release).** The tier *semantics* below are frozen by the
> [M-V1-STABILITY-PROMISE design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v1_0_0/m-v1-stability-promise.md).
> The per-module / per-command tier *assignments* are drafted from usage evidence and require
> human ratification before the `v1.0.0` tag. Rows marked **⚠ proposed** are the ones most likely
> to move at ratification. This is a release-gate line-item, not a merge blocker.

This page defines what a consumer — a human developer or an AI coding agent — may rely on across
AILANG's `1.x` releases. A `1.0.0` without a written compatibility contract is a version number,
not a promise; this is the promise.

## Tiers

AILANG classifies every public surface into exactly one of three tiers.

| Tier | Promise | Marked in docs |
|------|---------|----------------|
| **Stable** | Covered by the 1.x promise. A breaking change requires a **major** version bump. | (default; listed below) |
| **Experimental** | Shipped and usable, but **may change in a 1.x minor** with a CHANGELOG entry. | Explicitly listed as Experimental. |
| **Internal** | **No promise.** May change or be removed at any time. Listed here only so absence is unambiguous. | Explicitly listed as Internal. |

### What "Stable" means in 1.x

- **No breaking changes to the Stable surface within 1.x minor releases.** Behavior, signatures,
  and semantics of Stable syntax, stdlib functions, and CLI commands are held constant across
  `1.0 → 1.1 → …`. Breaking any of them requires `2.0`.
- **Experimental surface may change in a minor** — but only with an explicit CHANGELOG entry
  describing the change and migration.
- **Deprecations live ≥1 minor before removal.** A Stable item slated for removal is first marked
  deprecated (in docs and, where tooling supports it, at compile time) and remains functional for
  at least one full minor release before it can be removed in a subsequent major.
- **Additive changes are always allowed.** New stdlib modules, new CLI commands, new optional
  flags, and new syntax sugar may land in any minor — they do not break existing programs.

### Criteria used to draft the assignments

An item is drafted **Stable** when all hold: documented in the canonical reference / teaching
prompt, exercised by `examples/` and/or the eval suite, and carrying no open P0 defect. Items that
are newer, niche, or still evolving are drafted **Experimental** (the cheap direction to be wrong
in — an Experimental item can be promoted later without breaking anyone). Development/diagnostic
surface with no consumer contract is **Internal**.

## Syntax surface

The Stable syntax set is defined **by reference** to the canonical teaching prompt / language
reference rather than re-enumerated here (single source of truth):

- **Canonical reference:** [`ailang prompt --source=embedded`](/docs/reference/language-syntax)
  (embedded prompt version **v0.16.2** at the time of writing).
- The **Stable** syntax constructs are those documented as current in that reference: modules and
  imports, `func` / `pure func` declarations, `let … in` and block-body `{ … ; … }` forms,
  lambdas (`\x. …`), `if/then/else`, brace-form `match` with pattern guards, algebraic data types,
  records and row-polymorphic record access, effect rows (`! {IO, FS, …}`), the `::`/`++` list
  operators, string interpolation (`"${expr}"`), and `requires`/`ensures` contracts.
- **Retired syntax is not Stable and is rejected by the parser** — e.g. the ML/Haskell
  `match … with | …` form now produces `PAR019` ("`match ... with` is not valid AILANG syntax").
  Use brace-form `match x { pat => expr, … }`.
- **Experimental syntax:** the effect-refinement surface (parameterised effects / capability
  refinement) is Experimental-pending its own decomposition; typed quasiquotes and the `?`
  error-propagation operator are **not yet implemented** (see
  [Known Limitations](/docs/reference/limitations)).

## Stdlib surface

Enumerated from `ls std/*.ail` (42 top-level modules) plus the subdir module
`std/ai/streaming.ail`. Every module appears in exactly one tier.

### Stable stdlib

| Module | Purpose | Evidence |
|--------|---------|----------|
| `std/io` | Console input/output | 142 example files; core effect surface |
| `std/list` | Linked-list operations | 43 example files |
| `std/result` | Error handling (`Ok`/`Err`) | 41 example files |
| `std/option` | Optional values (`Some`/`None`) | 29 example files |
| `std/string` | String manipulation | 36 example files |
| `std/json` | JSON encoding/decoding | 16 example files; ships `decode`/`encode` |
| `std/math` | Mathematical functions | documented; core numerics |
| `std/fs` | Sandboxed file system | 8 example files; capability-gated |
| `std/env` | Environment variables (secure snapshot) | 8 example files |
| `std/clock` | Time operations (virtual-time deterministic) | 4 example files |
| `std/rand` | Random number generation | 4 example files |
| `std/array` | Fixed-size arrays, O(1) access | 3 example files |
| `std/map` | Hash maps, O(1) lookup | 2 example files |
| `std/bytes` | Byte-slice operations | 5 example files |
| `std/iter` | Iteration control primitives | documented; general-purpose |
| `std/datetime` | Pure date/time operations | documented; pure |

### Experimental stdlib

May change in a 1.x minor with a CHANGELOG entry.

| Module | Purpose | Why Experimental |
|--------|---------|------------------|
| `std/ai` | AI as a typed effect (`call`, `callJson`, `step`) | surface still evolving with the agent loop |
| `std/ai/streaming` | Streaming AI responses | new; narrow usage |
| `std/net` | Network operations | ⚠ proposed — widely used (13 examples) but security/format-sensitive; held Experimental pending a network-surface review |
| `std/stream` | Async stream sources / `selectEvents` | concurrency-adjacent; evolving |
| `std/secret` | Gated secret resolution (M-SECRET-EFFECT, v0.26.0) | new effect (v0.26.0) |
| `std/cognition` | Cognitive-OS message fabric (M-COG-RUNTIME) | named Experimental in design doc |
| `std/dom` | Cognitive-OS DOM substrate (M-COG-RUNTIME) | named Experimental in design doc |
| `std/game` | Game-development utilities | named Experimental in design doc |
| `std/extension` | Helpers for AILANG extension packages | named Experimental in design doc |
| `std/sem` | Semantic-caching types (M-DX15) | named Experimental in design doc |
| `std/sharedmem` | Shared memory | named Experimental in design doc |
| `std/sharedindex` | Similarity-based semantic index | named Experimental in design doc |
| `std/embedding` | Vector-embedding utilities | niche; 0 examples |
| `std/simhash` | Locality-sensitive hashing | niche; 0 examples |
| `std/crypto` | Cryptographic operations | ⚠ proposed — security-sensitive; conservative tiering |
| `std/jwt` | JWT parsing/verification | ⚠ proposed — security-sensitive |
| `std/xml` | XML parsing/querying | ⚠ proposed — format surface may evolve |
| `std/regex` | Linear-time (RE2) regex: compile / isMatch / findFirst / findAll / replaceAll / split | new (v0.30.0); RE2 subset (no backref/lookaround); `RegexMatch` record shape may evolve |
| `std/html` | HTML5 parsing | format surface may evolve |
| `std/zip` | ZIP archive read/write/stream | ⚠ proposed — format surface may evolve |
| `std/tar` | tar archive read/extract | format surface may evolve |
| `std/gzip` | gzip compression | format surface may evolve |
| `std/deflate` | raw deflate (RFC 1951) / zlib (RFC 1950) | low-level; niche |
| `std/process` | External command execution | ⚠ proposed — host-boundary surface, conservative tiering |
| `std/package` | Resolve files bundled in an installed package | packaging surface still evolving |

### Internal stdlib

No promise. Development / diagnostic utilities.

| Module | Purpose |
|--------|---------|
| `std/debug` | Debug helpers |
| `std/trace` | Custom trace span/event emission (telemetry-facing) |
| `std/smoke` | Helpers for package `_smoke.ail` boot probes |

## CLI surface

Tiered by top-level command / family (not by raw `--help` line count). Every top-level command
appears in exactly one tier.

### Stable CLI

| Command | Purpose |
|---------|---------|
| `version` | Show version / commit / build time |
| `run` | Run an AILANG program (`--caps`, `--entry`) |
| `check` | Type-check without running |
| `repl` | Interactive REPL |
| `test` | Run tests |
| `prompt` | Emit the canonical teaching prompt |
| `docs` | Stdlib module documentation |
| `install` / `search` / `publish` / `add` / `lock` / `tree` | Package + registry workflow |

### Experimental CLI

May change in a 1.x minor with a CHANGELOG entry.

| Command / family | Purpose | Why Experimental |
|------------------|---------|------------------|
| `ai-check` | Unified check+verify JSON for AI | JSON shape still stabilising |
| `watch` | Watch-and-reload | ⚠ proposed |
| `serve-api` / `server` / `serve` / `lsp` | Web / API / LSP servers | evolving interfaces |
| `replay` / `export-training` | Trace replay / training export | trace-format-dependent |
| `sandbox-check` | FS sandbox path diagnostics | diagnostic-adjacent |
| `editor` | Editor syntax-highlighting install | tooling convenience |
| `axioms` | Axiom compliance scorecard | evolving |
| `examples` | Search/explore examples | tooling convenience |
| `trace` / `observatory` | Telemetry management / analytics | telemetry surface evolving |
| `pkg` (+ `notify-upgrade`, `affected-by`) / `init` / `unpublish` / `pkg-docs` | Package coordination | packaging surface still evolving |
| `messages` | Agent messaging | evolving agent surface |

### Internal CLI

No promise. Diagnostic / harness / agent-infra surface.

| Command / family | Purpose |
|------------------|---------|
| `debug` | AST / type debugging |
| `doctor` | Registry / ADC diagnostics |
| `builtins` | Builtin-registry inspection |
| `iface` | Normalized module-interface JSON (shape is Internal) |
| `eval` / `eval-suite` / `eval-analyze` / `eval-report` / `eval-compare` / `eval-matrix` / `eval-sweet-spot` / `eval-summary` | Eval harness |
| `coordinator` / `chains` / `dashboard` / `workspaces` | Autonomous-agent coordination infra |
| `generate-extension-registry` | Static extension-dispatch codegen |

## Known limitations & version promises

- Live-verified [Known Limitations](/docs/reference/limitations) — every entry carries a repro,
  transcript, and verified-at date.
- Future-version language is confined to the [Roadmap](/docs/roadmap) — that is the one
  allowlisted place for "planned for vX.Y" statements.

## Follow-ups (post-ratification)

- A `make check-doc-promises` CI guard to prevent stale past-version promises from re-accumulating
  (tracked as a Deferred Decision in the design doc).
- Compile-time deprecation warnings for Experimental → removed surface (post-v1).
- The v1.1 promise review re-examines effect handlers and CSP/session-type surface.

---

*Ratification status: **pending (Mark, at release)**. Drafted headless, mission iteration 5.*
