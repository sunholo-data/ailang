---
title: Stdlib Index
description: Every AILANG stdlib module at a glance — purpose, capability required, import path
sidebar_label: Stdlib Index
---

# Stdlib Index

Every module that ships with AILANG, grouped by what it's for. **Pure** modules can be imported anywhere; **capability** modules need their capability passed via `--caps` at run time.

```bash
ailang run --caps IO,Net,Clock examples/runnable/myprogram.ail
```

See [Effects](/docs/reference/effects) for how the capability system works, [Modules](/docs/reference/modules) for import syntax, and [Browse Packages](/docs/packages) for third-party modules published to the registry.

## Data & Collections

| Module | Purpose | Capability |
|---|---|---|
| `std/list` | Functional list operations: map, filter, fold, take, drop | — |
| `std/array` | O(1) indexed arrays (vs lists, which are O(n) indexed) | — |
| `std/map` | O(1) key-value lookup backed by Go hashmaps | — |
| `std/string` | Length, substring, trim, split, replace, case | — |
| `std/regex` | Linear-time (RE2) regex: match, find, replace, split | — |
| `std/bytes` | UTF-8 encoding, base64, byte-level operations | — |
| `std/iter` | `FoldStep` and bounded folds for early exit | — |

## Algebraic Data Types

| Module | Purpose | Capability |
|---|---|---|
| `std/option` | `Some(x)` / `None` for values that may not exist | — |
| `std/result` | `Ok(x)` / `Err(e)` for operations that can fail | — |

## Numerics & Randomness

| Module | Purpose | Capability |
|---|---|---|
| `std/math` | Trigonometric, exponential, logarithmic, rounding | — |
| `std/rand` | Random integers, floats, booleans (seeded) | `Rand` |

## Time

| Module | Purpose | Capability |
|---|---|---|
| `std/datetime` | Pure date/time math (parsing, formatting, arithmetic) | — |
| `std/clock` | Wall-clock time and sleep; deterministic mode supported | `Clock` |
| `std/game` | Frame timing for game loops: delta time, FPS | `Clock` |

## Encoding & Serialization

| Module | Purpose | Capability |
|---|---|---|
| `std/json` | JSON encode/decode | — |
| `std/yaml` | [Decode YAML into the same `Json` ADT as `std/json`](./std-yaml) via a pure, WASM-portable YAML→JSON bridge (`yamlToJson`, `decode`) | — |
| `std/xml` | Parse XML strings into `XmlNode` trees, query elements. v0.21.0+ also ships [tree-walk performance builtins](./std-xml) (`foldChildren`, `getAttrMap`, `nodeKind`). | — |
| `std/html` | Lenient HTML5 parse (WHATWG-spec) into the same `XmlNode` ADT as `std/xml` | — |
| `std/gzip` | Gzip compress/decompress (base64-encoded I/O) | — |
| `std/deflate` | Raw deflate (RFC 1951) and zlib-wrapped (RFC 1950) primitives — PDF FlateDecode, HTTP `Content-Encoding: deflate`, PNG IDAT | — |
| `std/zip` | Read/write ZIP archives (including `.docx`, `.xlsx`, `.epub`) | — |
| `std/tar` | Read entries from uncompressed tar archives | — |
| `std/jwt` | Parse and verify JSON Web Tokens | — |
| `std/crypto` | Hash, HMAC, symmetric/asymmetric primitives | — |

## I/O

| Module | Purpose | Capability |
|---|---|---|
| `std/io` | Print to stdout, read from stdin, exit codes, raw bytes | `IO` |
| `std/fs` | Read/write files; sandboxed via `AILANG_FS_SANDBOX` | `FS` |
| `std/env` | Environment variable access (snapshot, allowlist, redaction) | `Env` |
| `std/process` | Execute external commands; allowlist + timeout + size limits | `Process` |
| `std/secret` | Gated secret resolution (`op://…`); resolved value is tainted `<secret>` until an explicit `Declassify` step | `Secret` |

## Network

| Module | Purpose | Capability |
|---|---|---|
| `std/net` | HTTP GET/POST; HTTPS by default, DNS rebinding prevention | `Net` |
| `std/stream` | WebSocket (bidirectional), SSE (server-sent events) | `Stream` |

## AI & Semantic

| Module | Purpose | Capability |
|---|---|---|
| `std/ai` | General-purpose AI oracle: `string -> string`, JSON variants | `AI` |
| `std/embedding` | Compute embedding vectors via host-provided model | varies by host |
| `std/sem` | Semantic frame caching primitives | `Clock`, `SharedMem` |
| `std/sharedmem` | Key-value shared memory (effect wrappers for caching) | `SharedMem` |
| `std/sharedindex` | Namespace-partitioned similarity search index | `SharedIndex` |
| `std/simhash` | SimHash fingerprints for near-duplicate detection | — |

## Package & Extension Authoring

| Module | Purpose | Capability |
|---|---|---|
| `std/package` | Resolve paths to assets shipped inside a package (helper scripts, schemas, templates) | `FS` |
| `std/extension` | Conventions and helpers for packages that plug into a host runtime (workdir requirements, etc.) | `FS` |
| `std/smoke` | Pre-publish smoke-gate helpers: `dispatchAllTools`, `dispatchTool`, `okSuite` for v0.19.0+ canonical smoke suites | `IO` |

## Cognitive OS (Browser substrate)

| Module | Purpose | Capability |
|---|---|---|
| `std/dom` | Canonical DOM patches for replayable UI mutations (`AddPanel`, `ApplyPatch`, etc.) | `DOM` |
| `std/cognition` | Lamport-clocked Msg fabric (`sendMsgResult`, `recvMsg`) + event-log sinks | `Msg`, `Cog` |

## Tracing & Debug

| Module | Purpose | Capability |
|---|---|---|
| `std/debug` | Structured tracing and assertions; **erased in `--release` mode** | `Debug` (ghost) |
| `std/trace` | Emit custom spans and events into the trace pipeline | `Trace` |
| `std/trace_test` | Test helpers for trace-based assertions | `Trace` |

---

## Importing modules

```ailang
module myapp

import std/io (println)
import std/list (map, filter)
import std/result (Result, Ok, Err)

export func main() -> unit ! {IO} {
  let xs = [1, 2, 3, 4, 5]
  let evens = filter(\x. x % 2 == 0, xs)
  println("evens: ${evens}")
}
```

Run with: `ailang run --caps IO myapp.ail`

## See also

- **[Effects](/docs/reference/effects)** — how the capability system works
- **[Modules](/docs/reference/modules)** — import syntax, `module`, `export func`
- **[Browse Packages](/docs/packages)** — third-party modules published to the registry
- **[Language Syntax](/docs/reference/language-syntax)** — full reference
