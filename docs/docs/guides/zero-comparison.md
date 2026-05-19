---
title: AILANG vs Zero
sidebar_label: AILANG vs Zero
description: How AILANG compares to Vercel Labs' Zero — two languages that arrived at "for agents" from opposite ends of the language-design spectrum.
---

# AILANG vs Zero

On 15 May 2026, [Vercel Labs](https://github.com/vercel-labs/zero) published **Zero**, a new language with the tagline _"The programming language for agents."_ AILANG has been working that same thesis since 2024. This page is a side-by-side comparison and an honest read on what each language gets right.

:::tip Bottom line
Zero and AILANG are not competitors. They're two halves of the same hypothesis: **agents need languages designed for them**, not languages designed for humans that agents tolerate. Zero stakes out _agent-authored systems software_ (Rust-shaped, native, manual memory). AILANG stakes out _agent-authored applied/reasoning code_ (Haskell/OCaml-shaped, GC, effect rows, LLM-native primitives).
:::

## The convergence

Both languages independently arrived at the same core principles:

| Principle | Zero | AILANG |
|---|---|---|
| Explicit effects in signatures | `raises { ErrSet }` | `! {IO, FS, Net, Clock, AI}` |
| Capability-object I/O (no globals) | `World` parameter | Capability handlers + budgets |
| Machine-readable diagnostics | `--json` everywhere | `--json` on key commands |
| Determinism as a design goal | Bounded compile-time eval | A1–A12 [Design Axioms](/docs/references/axioms) |
| Compiler as an agent tool | `zero fix --plan --json` | `ailang chains`, MCP server |
| Stable skill/prompt metadata | `zero skills` (7 modular skills) | `ailang prompt` (single versioned blob) |

When two teams arrive at the same answer independently, the answer is probably right.

## The divergence

Where the languages differ matters more than where they agree, because the differences reflect _what kind of agent code each language is for_.

| Dimension | Zero | AILANG |
|---|---|---|
| **Paradigm** | Imperative, C-family, statements | Pure functional, everything is an expression |
| **Compiler** | Native, written in C, emits `linux-musl-x64` / wasm executables | Go-hosted, tree-walking + bytecode VM, interpreted |
| **Type system** | Width-explicit primitives (`i8`–`u64`), generics via monomorphization, no inference | Hindley–Milner inference + **row polymorphism** + type classes |
| **Memory** | Manual: `owned<T>`, `drop()`, `defer`, `ref<T>` / `mutref<T>`, borrow checker | GC-backed; no manual lifetime management |
| **Effects** | `raises { ErrSet }` — primarily error effects + capability params | Full **row-polymorphic effect rows** |
| **Error handling** | `check expr` / `raise X` / `rescue err {}` (algebraic-effects-lite) | `Result[T,E]` ADT + `?` propagation |
| **Concurrency** | None documented | Effect-typed Fork / Call / Done |
| **LLM primitives** | None | `std/ai` with `call`, `callJson`, `callJsonSimple` |
| **FFI** | `extern c "config.h" as config` | Go interop via embedded mode |
| **Compile target** | Standalone native binaries | REPL + bytecode VM |
| **Repository age** | 2 days (May 2026) | v0.20, multi-year |

## Side-by-side: "hello world"

Both languages agree that hidden I/O is a bug, but they pay for that agreement in opposite ways.

**Zero — capability injection, mutable, explicit `check`:**

```zero
pub fun main(world: World) -> Void raises {
    check world.out.write("hello from zero\n")
}
```

**AILANG — capability via effect row, expression-oriented:**

```ailang
module examples/hello

import std/io (println)

export func main() -> () ! {IO} =
  println("hello from ailang")
```

Same insight (effects can't sneak in), different mechanism:
- Zero threads `World` as a _value parameter_ you destructure
- AILANG threads `IO` as a _type-level effect row_ the compiler tracks and the runtime sandboxes

## Side-by-side: error handling

**Zero — algebraic-effects-lite with `raises` / `check` / `rescue`:**

```zero
fun validate(ok: Bool) -> i32 raises { InvalidInput } {
    if ok == false {
        raise InvalidInput
    }
    return 42
}

fun run() -> Void raises { InvalidInput } {
    check validate(true)
}
```

**AILANG — `Result[T, E]` ADT with `?` propagation:**

```ailang
type ValidationError = InvalidInput

func validate(ok: bool) -> Result[int, ValidationError] =
  if ok
  then Ok(42)
  else Err(InvalidInput)

func run() -> Result[(), ValidationError] = {
  let _ = validate(true)?;
  Ok(())
}
```

Zero's design is closer to OCaml/Koka's effect handlers; AILANG's is closer to Rust's `Result` + `?`. Both are explicit, both check exhaustively at compile time — the ergonomic taste differs.

## What Zero has that AILANG doesn't

1. **Native binary compilation.** `zero build --emit exe --target linux-musl-x64` produces small standalone executables. AILANG bytecode-VM only — see [project_codegen_strategic_decision.md](https://github.com/sunholo-data/ailang/blob/dev/.brain/project_codegen_strategic_decision.md) for the deliberate trade-off.
2. **C FFI** via `extern c "header.h" as alias`.
3. **Manual memory + borrow checking** — for agents writing performance-sensitive systems code.
4. **Bounded compile-time evaluation** with sandboxed const-eval (no filesystem, no network, no ambient env).
5. **`zero fix --plan --json`** — the compiler proposes a structured patch in JSON the agent can apply. AILANG diagnostics are good but don't currently emit a repair plan as a first-class artifact.
6. **`zero explain ERR_CODE` / `zero doctor --json` / `zero size --json`** — every command has a JSON mode designed for tool consumption.
7. **Modular bundled skills.** `zero skills` serves seven focused prompts (`zero-language`, `zero-agent`, `zero-diagnostics`, `zero-builds`, `zero-packages`, `zero-stdlib`, `zero-testing`) bundled inside the compiler binary and version-matched to the installed version. An agent can load only the skill relevant to its current task instead of the whole language reference. AILANG's `ailang prompt` currently returns a single consolidated file — see the design doc for the proposed split.

## What AILANG has that Zero doesn't

1. **LLM-native primitives.** `std/ai.call / callJson / callJsonSimple` make calling models a typed effect, not a library decision. Zero is about _building tools agents run_; AILANG is about _agents calling models_.
2. **Row-polymorphic effects.** Functions can be generic over their callees' effects (`f<E>(x) -> y ! E`), which Zero's closed `raises { Set }` can't express.
3. **Eval-driven language design.** The [evaluation harness](/docs/guides/evaluation/) runs benchmarks across multiple models and languages, treating "what stops agents from succeeding" as a first-class language-design signal.
4. **MCP server + agent messaging.** `ailang messages`, `ailang chains`, MCP integration, and the [Collaboration Hub](/docs/guides/collaboration-hub) make AILANG installable as a peer agent.
5. **Pattern matching with ADTs.** Zero's `enum` and `choice` give tagged unions but the ergonomics aim at C-style switches; AILANG matches like Haskell/OCaml with exhaustiveness checking.
6. **Capability budgets and contracts.** [Capability budgets](/docs/reference/capability-budgets) and [contracts](/docs/guides/contracts) give pre-commitment guarantees Zero doesn't have.
7. **Concurrency.** Zero has no documented concurrency primitives.

## Where each language is _better suited_

| If your agent needs to… | Reach for |
|---|---|
| Generate a small native CLI tool | **Zero** |
| Write a parser, transformer, or DSL | **AILANG** |
| Manipulate binary buffers / wire protocols | **Zero** |
| Call an LLM and validate JSON output | **AILANG** |
| Interop with a C library | **Zero** |
| Compose effects across modules | **AILANG** |
| Ship a `linux-musl-x64` binary | **Zero** |
| Run inside a sandboxed REPL or eval loop | **AILANG** |

## What we're watching

Zero is two days old. AILANG plans to track:
1. **Diagnostic schema.** Does Zero's `--json` repair metadata stabilize into a format worth borrowing? See the design doc [M-ZERO-LANGUAGE-LEARNINGS](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_21_0/m-zero-language-learnings.md).
2. **`zero fix --plan`.** A compiler-emitted patch plan is a strong primitive for the eval-fix loop.
3. **Adoption.** If Vercel Labs scales Zero into Vercel's product surface, the systems-agent niche becomes well-defined and AILANG can confidently stop chasing it.
4. **Convergence.** Capability objects, explicit effects, JSON diagnostics — the design space is narrowing. Worth a periodic re-comparison.

## Eval-suite inclusion?

**Yes in principle — but blocked in practice as of Zero v0.1.1.**

The principled answer is _yes, absolutely_: AILANG's methodology already assumes models have minimal training exposure and learn the language from an in-context prompt (`ailang prompt`), and Zero ships the equivalent surface (`zero skills get zero-language`). Comparing AILANG-from-prompt vs Zero-from-prompt vs Python (memorized by every model) directly answers the question _"does language-for-agents design actually move pass-rate?"_

A hands-on smoke test against the released Zero v0.1.1 binary (17 May 2026) revealed concrete runtime blockers, however:

1. **The released binary's direct emit only supports trivial programs.** Verbatim diagnostic: _"restrict this program to exported no-parameter functions returning small integer literals."_ Programs with `String` parameters or locals fail to lower with `CGEN004`. `balanced_parens` type-checks but cannot run.
2. **The C bridge is deliberately removed** — `"cBridge":{"policy":"removed","explicitDirectFallback":"never-c-bridge"}`. Even with `--cc clang` (works for trivial `hello`), the lowerer rejects complex types before linking.
3. **No float / math stdlib.** `adt_option` (calls `safeSqrt`) literally cannot be expressed.
4. **`zero skills` is served by a wrapper script in the source checkout, not the released binary.**
5. **`zero explain <CODE>` doesn't recognize its own diagnostic codes** at v0.1.1.

The eval-suite work is scoped in the design doc as Phase 3, **gated on**: Zero v0.2.0+ shipping `zero run` for non-trivial programs, OR a 6-month timeline check (November 2026). A constrained "check-only" pass measuring _"fraction of LLM-generated Zero programs that type-check"_ is a meaningful optional intermediate signal (~3-4h to wire) — see the design doc for the implementation path.

The smoke test was still hugely valuable: it confirmed Zero's JSON diagnostic schema is genuinely best-in-class and worth directly studying for AILANG's Phase 1 diagnostic-schema-hardening work.

---

**Last updated**: 17 May 2026 (Zero v0.x, AILANG v0.20.0)
