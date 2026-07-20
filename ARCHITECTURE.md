# Architecture

This document is a high-level map of the AILANG repository for
contributors and reviewers. It is **not** a language reference — for
that, see <https://ailang.sunholo.com/docs/reference/syntax/>.

## What AILANG is

A statically-typed, effect-typed programming language designed as a
deterministic execution substrate for AI-generated code. The compiler
is written in Go and targets a tree-walking evaluator (with a
bytecode VM tracked in `design_docs/`). The runtime ships with a
capability-based effect system, a Hindley-Milner type system with
row-polymorphic records and effects, and a curated standard library.

```
.ail source
    │
    ├── lexer ───────────► tokens
    │
    ├── parser ──────────► AST              (internal/ast)
    │
    ├── elaborator ──────► Core IR + types  (internal/elaborate)
    │
    ├── type checker ────► typed Core       (internal/types)
    │     │
    │     └── effect inference + capability checks
    │
    ├── pipeline ────────► loadable module  (internal/pipeline)
    │
    └── evaluator ───────► values + effects (internal/eval, internal/effects)
```

## Top-level layout

| Path | Purpose |
|---|---|
| `cmd/ailang/` | Main CLI entry point; subcommand dispatch. |
| `cmd/ailang-microrag-mcp/` | MCP server exposing AILANG docs/builtins as tools. |
| `internal/` | All compiler and runtime code (see below). |
| `stdlib/` | AILANG-language standard library (`std/io`, `std/list`, etc.). |
| `examples/` | Runnable `.ail` programs grouped by feature. |
| `prompts/` | Teaching prompts loaded by `ailang prompt`. |
| `docs/` | Docusaurus site (https://ailang.sunholo.com). |
| `design_docs/` | Planned and implemented design documents per version. |
| `benchmarks/` | Eval suite definitions and baselines. |
| `eval_results/` | Historical eval outcomes per release. |
| `web/` | Browser REPL bundle (WASM). |
| `docker/` | Per-agent and dashboard container images. |
| `.github/workflows/` | CI, release, security scans (CodeQL, Scorecard). |

## `internal/` packages

Grouped by phase:

### Front end
- `internal/lexer` — tokenizer, position tracking.
- `internal/parser` — Pratt parser producing `internal/ast`. Fuzz
  tests live here (`fuzz_test.go`).
- `internal/ast` — AST types. See `.claude/rules/parser.md` for the
  golden rule about `ast.Type` switch exhaustiveness.

### Middle end
- `internal/elaborate` — desugars AST → Core IR.
- `internal/types` — Hindley-Milner inference + row polymorphism +
  effect inference. The single most security-relevant subsystem
  (capability checks happen here).
- `internal/iface` — module interfaces + cross-module type sharing.
- `internal/dispatch` — type-class dictionary construction and method
  resolution (dictionary-passing).
- `internal/pipeline` — orchestrates lexer → parser → elaborate →
  types into a loadable module artifact.

### Back end / runtime
- `internal/eval` — tree-walking evaluator over Core IR.
- `internal/effects` — effect handlers (IO, FS, Net, Clock, AI,
  Stream, Random). Each effect gates on a capability granted at
  module load time.
- `internal/builtins` — Go-implemented builtins exposed to AILANG.
- `internal/runtime` — value representation, scheduler, GC interop.

### Tooling and integration
- `internal/repl` — interactive REPL.
- `internal/apiserver` — local HTTP/MCP server for editor and agent
  integration.
- `internal/coordinator` — autonomous-agent orchestration.
- `internal/executor` — agent backends (Claude, Gemini, Codex, etc.)
  used by the coordinator.
- `internal/messaging` — inter-agent message bus + Pub/Sub.
- `internal/eval_harness` — runs benchmarks against language models.

## Architecture boundaries

The `internal/` tree is a single Go module today, but it is organized into
four **logical layers**. These layers are a *convention over the existing
physical tree* — there is deliberately **no** physical `core/` or `apps/`
directory (a physical restructure is a separately-scoped future step). The
layering exists so the v1.0 stability promise has a mechanical boundary to
scope to, and so contributors and AI agents know which subsystem they are in.

| Layer | Real packages |
|---|---|
| **core** (the compiler + runtime) | `internal/{parser,types,eval,core,elaborate,effects,builtins,lexer,ast,pipeline,runtime,link,iface}` |
| **dashboard / apps** (services + UI) | `internal/{server,coordinator,observatory,messaging}` (+ `ui/`) |
| **tools** | `internal/{eval_harness,eval_analysis,ai}` |
| **bridge / sdk** | `internal/embed` (+ `internal/runtime`, `internal/schema`) |

### Enforced import directions

The **bridge** (`internal/embed`) is the *only* sanctioned path from the
dashboard into the compiler. `internal/embed` exposes an `Engine` that loads
and calls AILANG modules and converts values to/from Go, so apps never need to
touch the parser, type checker, or elaborator directly.

| Direction | Allowed? | Enforced by |
|---|---|---|
| apps → core | **No** — only via `internal/embed` | `scripts/check_boundaries.sh` |
| core → apps | **No** — core must never depend on a service/UI | `scripts/check_boundaries.sh` |
| tools → core | Yes | (documented; not policed) |
| tools → apps | No | (documented; not policed) |
| bridge (`embed`) → core | Yes (controlled) | design |

The gate runs in CI (after the file-size check) and locally via:

```bash
make check-boundaries        # or: bash scripts/check_boundaries.sh
```

It matches quoted Go import paths, so it catches real `import` statements
rather than incidental string mentions. Two rules are enforced:

1. No **core** package imports any **dashboard** package.
2. No **dashboard** package imports the compiler surface
   (`parser`/`types`/`core`/`elaborate`/`pipeline`) directly.

**One deliberate exception:** `internal/eval` is *not* on rule 2's deny-list.
`internal/embed`'s own public API takes and returns `eval.Value`
(e.g. `embed.ToGo(v eval.Value)`), so a dashboard caller of the sanctioned
bridge is forced to name `eval.Value`. That is the bridge's value type, not a
behavioral dependency on the evaluator. If `embed` ever re-exports its own
`Value` alias, `eval` should be added back to the deny-list.

## Capability-effect system

This is the security-relevant core of AILANG. Every function that
performs I/O declares its effects in its type:

```ailang
func greet(name: string) -> () ! {IO} = println("Hello ${name}")
```

The type system **prevents** an `! {IO}` function from being called
in a `pure` context. At module load, the runtime grants capabilities
explicitly; an effect handler with no granted capability aborts.

This is deliberately a least-authority design — it's the property
that makes AILANG a candidate substrate for AI-generated code that
must be auditable. See <https://ailang.sunholo.com/docs/guides/effects/>
for the user-facing reference and `internal/effects/` for the
implementation.

## Build and release

- `make build` produces `bin/ailang`.
- `make ci` runs the full local CI suite (vet, lint, test,
  verify-examples, file-size check).
- `.github/workflows/release.yml` cuts a multi-platform release on
  `v*` tags, signs each archive with cosign keyless, and publishes
  SHA256 + signature artifacts. See SECURITY.md for the verification
  chain.

## AI-coordinator development model

A non-architectural but load-bearing fact: most code in `internal/`
is authored by AI agents under the [coordinator](https://ailang.sunholo.com/docs/guides/coordinator).
The third-party static analysers (CodeQL, Sonar, Scorecard, Go
Report Card) are how reviewers verify that AI-authored code meets
the same bar as human-authored code. See [GOVERNANCE.md](GOVERNANCE.md)
for the model and [CONTRIBUTING.md](CONTRIBUTING.md) for the
contribution flow.

## Where to look first

| If you're investigating… | Start in… |
|---|---|
| A type error | `internal/types/` + `internal/elaborate/` |
| A parse failure | `internal/parser/` (and `.claude/rules/parser.md`) |
| Why an effect didn't fire | `internal/effects/` + the handler for that effect |
| An eval-suite regression | `benchmarks/` + `internal/eval_harness/` |
| Coordinator hangs | `internal/coordinator/` + `internal/executor/<provider>/` |
| MCP-tool behavior | `cmd/ailang-microrag-mcp/` + `internal/apiserver/mcp.go` |
| Release-signing chain | `.github/workflows/release.yml` + `SECURITY.md` |
