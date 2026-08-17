---
title: Three Camps of AI-Native Languages
sidebar_label: Three Camps Comparison
description: A survey of 16 programming languages designed for LLM-generated code, grouped into three camps — and where AILANG fits.
---

# Three Camps of AI-Native Languages

In May 2026, Negroni Venture Studios [published a survey](https://negroniventurestudios.com/2026/05/20/three-camps-alike-in-dignity/) of programming languages designed specifically for LLM-generated code. The post catalogued ~20 such languages — most of them less than 12 months old — and grouped them into three camps based on what their authors believe is the critical failure mode of LLM codegen.

This page mirrors that survey, places AILANG inside it, and adds something the original post didn't: a **gap analysis** showing where each camp's hypothesis would shine on benchmarks AILANG (and the field) doesn't yet test.

:::tip Bottom line
Twenty teams arriving at the same three answers in six months is not coincidence. It's the early shape of a consensus that **agents need languages built for them, not languages tolerated by them**. The three camps disagree about *which* property matters most. AILANG's bet: verification + orchestration are the same problem.
:::

## The Three Camps

| Camp | Claim | Mechanism | Example languages |
|------|-------|-----------|-------------------|
| **Syntactic** | LLMs fail because *tokens are ambiguous* | Restructure syntax to remove token-level ambiguity | X07, NERD, Magpie, Laze |
| **Verification** | LLMs fail because *output isn't checked* | Make contracts mechanically verifiable; ship a checker | Vera, Aver, Raskell, Prove, Pact, MoonBit, Zero, **AILANG** |
| **Orchestration** | LLMs fail because *the loop around them is wrong* | Reframe as agent coordination, not language design | Pel, Marsha, Plumbing, Quasar, Boruna |

Each camp encodes a testable hypothesis about where LLM codegen breaks down. They're not abstract disagreements — they're concrete design choices producing measurable outcomes.

## The Camps in Detail

### Syntactic Camp — fix the tokens going in

Authors in this camp believe the LLM's tokenizer is the bottleneck: ambiguous operators, optional punctuation, and inconsistent whitespace handling produce more codegen failures than missing knowledge. Their fix is to restructure the language surface itself.

| Language | Distinguishing mechanism | URL |
|----------|--------------------------|-----|
| **X07** | Eliminates text syntax; programs are JSON ASTs edited through RFC 6902 patches | [x07lang.org](https://x07lang.org) |
| **NERD** | Replaces all operators with English keywords (`PLUS`, `EQUALS`) | [nerd-lang.org](https://www.nerd-lang.org/) |
| **Magpie** | Surfaces Static Single-Assignment (SSA) form as user-facing syntax | [magpie-lang.com](https://magpie-lang.com/) |
| **Laze** | Minimal indentation-based, no punctuation, compiles to C | [github.com/kerv/laze](https://github.com/kerv/laze) |

### Verification Camp — fix the contract on what comes out

This is the largest camp. Authors here believe LLM codegen looks plausible but is semantically wrong too often to trust — the fix is mechanical verification that runs *before* the human or downstream agent sees the output.

| Language | Distinguishing mechanism | URL |
|----------|--------------------------|-----|
| **Vera** | Z3 verification + De Bruijn slot references (no variable names) | [veralang.dev](https://veralang.dev) |
| **Aver** | Lean 4 proof export, co-located verify blocks, decision blocks (ADRs in code) | [averlang.dev](https://averlang.dev) |
| **Raskell** | Builds on Haskell; focus is fixing the tooling/runtime, not creating a new language | [raskell.io](https://raskell.io/) |
| **Prove** | Deliberately AI-resistant — license explicitly forbids use as training data | [prove.botwork.se](https://prove.botwork.se) |
| **Pact** | Intent annotations per function, explicit effects, MCP server | [github.com/KikotVit/pact-lang](https://github.com/KikotVit/pact-lang) |
| **MoonBit** | Semantics-aware token sampler that constrains LLM generation to valid code | [moonbitlang.com](https://www.moonbitlang.com) |
| **Zero** | "One canonical form" + structured diagnostics for agents; no mandatory contracts | [github.com/vercel-labs/zerolang](https://github.com/vercel-labs/zerolang) |
| **AILANG** | Z3-backed `requires`/`ensures` contracts, row-polymorphic effects, HM types | [ailang.sunholo.com](https://ailang.sunholo.com) |

### Orchestration Camp — fix the loop around the language

Authors here believe the language itself is not the problem; what's missing is *primitives for coordinating agents*. Their fix is at the runtime, harness, or coordination layer — sometimes wrapping a conventional language, sometimes baking coordination into the language itself.

| Language | Distinguishing mechanism | URL |
|----------|--------------------------|-----|
| **Pel** | Reframes the problem as agent coordination as a language primitive | [arxiv.org/abs/2505.13453](https://arxiv.org/abs/2505.13453) |
| **Marsha** | Agent coordination framework | [github.com/alantech/marsha](https://github.com/alantech/marsha) |
| **Plumbing** | Graph-level wiring connecting agents into typed, streaming pipelines with static well-formedness | [Baez blog](https://johncarlosbaez.wordpress.com/2026/03/11/a-typed-language-for-agent-coordination/) |
| **Quasar** | Python-subset transpile + automated parallelization + uncertainty quantification (UPenn, 42% time reduction) | [arxiv.org/abs/2506.12202](https://arxiv.org/abs/2506.12202) |
| **Boruna** | Capability-gated bytecode VM + hash-chained tamper-evident audit logs | [github.com/escapeboy/boruna](https://github.com/escapeboy/boruna) |

## Where AILANG Fits

AILANG sits in the **Verification camp** by the original survey's grouping. That's correct as far as it goes — `requires`/`ensures` with Z3 is squarely verification-camp machinery — but it understates AILANG's full position. The honest scorecard:

| Camp | Membership | Evidence |
|------|------------|----------|
| **Verification** | ✅ Full member | Z3-backed contracts via `ailang verify`; row-polymorphic effect rows that mechanically check what a function can do; HM type inference |
| **Orchestration** | ✅ Strong member (under-surfaced publicly) | `std/ai` as a first-class effect; coordinator + executor/provider architecture; managed_agents; chain telemetry; eval harness; agent messaging; MCP server |
| **Syntactic** | ❌ Deliberate non-member | ML-family readable syntax with conventional operators. Bets that *contract on output* matters more than *tokens on input* |

The interesting bit is that orchestration and verification turn out to be the same problem once you commit to both:

- **Contracts** specify *what* the agent must produce.
- **Effect rows** specify *how* the agent's code touches the world.
- **`std/ai`** makes the agent itself a typed citizen of the program it's writing.

No other language in the survey occupies this intersection.

## The Gap Analysis

The original post stopped at categorization. The natural next question is: **do the camps' hypotheses actually hold up under measurement?** This requires benchmarks designed specifically to probe each camp's claims — most of which don't yet exist in AILANG's eval suite or anywhere else in the field.

The table below maps each peer language's distinguishing capability to a benchmark gap. Each row is a **testable hypothesis** about why that camp exists.

### Gaps driven by Syntactic camp claims

| Gap benchmark | Inspired by | Hypothesis under test |
|---------------|-------------|------------------------|
| `ast_patch_roundtrip` | X07 | Does generating a code transformation as a structural diff produce fewer errors than free text? |
| `dense_operator_program` | NERD | Do tokenizer-ambiguous operators (`<<`, `>>`, `&&`, `==`) measurably hurt LLM pass rate? **Direct refutation test for AILANG.** |
| `explicit_dataflow_ssa` | Magpie | Does SSA-shaped code (heavy let-chain, single assignment) improve LLM reasoning? |

### Gaps driven by Verification camp claims

| Gap benchmark | Inspired by | Hypothesis under test |
|---------------|-------------|------------------------|
| `shadowing_heavy_contract` | Vera | Do named identifiers break down under heavy shadowing? AILANG's HM should hold. |
| `decision_block_capture` | Aver | Does requiring agents to emit *structured rationale alongside code* improve auditability? |
| `intent_annotated_solver` | Pact | Does `@intent("...")`-style prompting measurably improve LLM pass rate? **Direct test of Pact's hypothesis.** |
| `canonical_convergence` | Zero | Run N=20 generations of the same prompt; measure how often the LLM converges on semantically-equivalent code. |

### Gaps driven by Orchestration camp claims

| Gap benchmark | Inspired by | Hypothesis under test |
|---------------|-------------|------------------------|
| `multi_agent_handoff` | Pel / Marsha | Agent uses `std/ai` to delegate a subtask, composes the result. **AILANG's `std/ai` makes this expressible without external wrapping.** |
| `typed_stream_pipeline` | Plumbing | Static well-formedness of a streaming transform — does the type system catch wiring errors? |
| `parallel_independent_subtasks` | Quasar | Code structure that exposes parallelism — can the LLM produce code an optimizer could parallelize? |
| `audit_chain_replay` | Boruna | Execute → capture chain → replay; bit-identical output. **Direct test of AILANG's A2 (replayability).** |

### Gaps for AILANG's own untested differentiators

| Gap benchmark | Why it matters |
|---------------|----------------|
| `ai_effect_summarize` | `std/ai` is AILANG's biggest unique capability, **currently unbenchmarked** |
| `ai_effect_json_schema` | Structured AI calls with schema enforcement (`callJson`) |
| `unauthorized_fs_refused` | Tests A4 (explicit authority) — code that *should fail* because the FS capability wasn't granted |

Total: **~14 gap benchmarks**, each one a testable claim about why a particular camp exists. Some will refute their camp's hypothesis. Some will surface real AILANG weaknesses. Both outcomes are informative.

The self-audit results — AILANG running against all 14 — are on the [Three Camps Self-Audit](./three-camps-self-audit) companion page.

## Deep Dive: AILANG vs Zero

*(Verified against Zero v0.1.1 / AILANG v0.20.0, May 2026.)*

Vercel Labs' [Zero](https://github.com/vercel-labs/zerolang) shipped May 2026 with the tagline
*"The programming language for agents"* — the same thesis AILANG has worked since 2024, arrived
at from the opposite end of the design spectrum. The two are complementary, not competitors:
Zero stakes out **agent-authored systems software** (C-family, native binaries, manual memory,
borrow checking, C FFI); AILANG stakes out **agent-authored applied/reasoning code**
(ML-family, GC, effect rows, LLM-native primitives).

**The convergence is the headline.** Both independently landed on: explicit effects in
signatures (Zero's `raises { ErrSet }` / AILANG's `! {IO, Net, …}`), capability-object I/O with
no ambient globals (Zero's `World` parameter / AILANG's capability handlers + budgets),
machine-readable diagnostics (`--json` everywhere), determinism as a design goal, and the
compiler as an agent tool (`zero fix --plan --json` / `ailang chains` + MCP). When two teams
arrive at the same answers independently, the answers are probably right.

**What Zero has that AILANG doesn't**: native binary emit, C FFI, borrow checking, and —
worth studying — a best-in-class JSON diagnostic schema plus compiler-emitted repair plans
(`zero fix --plan`) and seven modular version-matched skills vs AILANG's single prompt blob
([M-ZERO-LANGUAGE-LEARNINGS](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_21_0/m-zero-language-learnings.md)).
**What AILANG has that Zero doesn't**: LLM calls as a typed effect (`std/ai`), row-polymorphic
effects (generic over callees' effects — inexpressible in Zero's closed `raises` sets),
Z3 contracts, capability budgets, concurrency, and an eval harness that treats agent failure
modes as language-design signal.

**Eval-suite inclusion**: yes in principle (both languages assume models learn from an
in-context prompt), but blocked in practice — a hands-on smoke test of Zero v0.1.1 found the
released binary only lowers trivial programs (`CGEN004` on string parameters), no float/math
stdlib, and `zero skills` served only from the source checkout. Gated on Zero v0.2.0+ shipping
a working `zero run`, with a re-check scheduled November 2026.

## Deep Dive: Guardians (Prompt Injection as a Type-System Problem)

*(AILANG v0.16.0+; full how-to in the [IFC Labels guide](./ifc-labels.mdx).)*

Erik Meijer's January 2026 CACM piece [*Guardians of the Agents*](https://cacm.acm.org/practice/guardians-of-the-agents/)
(with a ~1,900-LOC [Python reference implementation](https://github.com/metareflection/guardians))
frames prompt injection correctly: the LLM cannot distinguish instructions from data, so verify
the plan before any tool runs, using three orthogonal checks — **taint analysis**, **automaton
checks over tool-call sequences**, and **SMT over arguments**.

AILANG's position: **two of those three are already language features**, and that placement is
the point. The SMT half shipped in v0.9.x (`ailang verify`, Z3 contracts in the signature); the
taint half shipped in v0.16.0 as IFC labels — `string<email>` marks a source, `string{not email}`
marks a refusing sink, and lowering a label requires the `! {Declassify}` effect. The compiler
rejects the injection shape at build time; identity bindings can't launder labels. It is the
parameterised-SQL move applied to the LLM-tool boundary: separate code from data *in the type
system*, at zero runtime cost, instead of wrapping the runtime in a guardian library whose every
sink is enforced by convention and reviewer attention. The automaton third is still ahead —
deliberately unbuilt until IFC labels have been run in anger. The
[prompt-injection benchmark](https://github.com/sunholo-data/ailang/tree/dev/benchmarks/prompt_injection)
scores models on producing code that `ailang verify` accepts as safe *and* correctly trips on
the injected variant — a gate Python baselines structurally cannot offer.

## What This Map Tells Us

A few observations from sitting with the survey for a few days:

**The camps aren't separable.** AILANG's design proves verification and orchestration can be the same problem. Pact's MCP server is an orchestration-camp move from a verification-camp language. Zero's "canonical form" is a verification-camp argument applied to a systems-flavored design. The categories are useful shorthand but they're not load-bearing.

**The syntactic camp's claim is the most testable.** `dense_operator_program` directly refutes or confirms NERD's hypothesis. If LLMs hit pass-rate parity on operator-heavy AILANG code vs Python, the tokenizer-ambiguity bet doesn't hold up under measurement.

**No one is building "a normal language, but better."** Every team in the survey took a strong position about what to change. That's the real consensus — and it suggests the field's collective belief that conventional language design *cannot* be patched into agent-usability.

**AILANG's eval harness is reusable.** Adding peer languages to AILANG's benchmark grid is mechanically straightforward: write a teaching prompt (~3k tokens) from public docs, install the toolchain, register a runner. The harness then measures what actually matters: how well an LLM can learn a new language from documentation alone. This is the methodology AILANG already uses on itself — AILANG isn't in any LLM's training data either.

## What's Next

This survey is a 2026-05 snapshot. The post is still being edited; some languages may move between camps; new ones will be added. Tracking the field at this density is itself a contribution.

The follow-up work on this site:
- [Three Camps Self-Audit](./three-camps-self-audit) — AILANG's own results on the 14 gap benchmarks (initial run live)
- Peer-language comparison data (MoonBit, Vera, Aver — work in progress)
- Audit memos for borrowable ideas (decision blocks, intent annotations, audit chains, typed streaming pipelines)

If you're working on an AI-native language that should appear here, [open an issue](https://github.com/sunholo-data/ailang/issues) — the goal is for this page to be a useful reference for the whole field, not just AILANG's positioning.

## See Also

- [AILANG vs Agents](./ailang-vs-agents) — why AILANG exists at all
- [AI Effect Guide](./ai-effect) — how `std/ai` works
- [Design Axioms](/docs/references/axioms) — the A1–A12 framework AILANG's design follows
- [Original survey post](https://negroniventurestudios.com/2026/05/20/three-camps-alike-in-dignity/) by Negroni Venture Studios
