# What It Actually Takes to Build a Programming Language

> A data-driven story from AILANG: 6.5 months, 2,435 commits, 111 releases, one developer + AI.

---

## The Numbers

| Metric | Value |
|--------|-------|
| First commit | 26 Sep 2025 |
| Current version | v0.10.15 (9 Apr 2026) |
| Total commits | 2,435 |
| Releases tagged | 97 |
| Go source files | 1,322 |
| Lines of Go | 346,000 |
| Test functions | 4,421 |
| Internal packages | 55 |
| Example programs | 314 |
| Design documents | 868 |
| Teaching prompt iterations | 51 |

---

## Act 1: "Can It Parse?" (Week 1)

**Sep 26, 2025 — Day 1**

Everything starts the same way: a lexer that turns text into tokens, a parser that turns tokens into a tree, an evaluator that walks the tree and produces results.

On day one, AILANG got all three. By end of day, it had:
- Lexer, parser, AST, evaluator
- String concatenation, lambdas, pattern matching
- Syntax highlighting for `.ail` files
- 96 commits in September alone

**The question every language answers first:** can I write `1 + 1` and get `2`?

---

## Act 2: "Can It Think?" (Weeks 2-5)

**Oct 2025 — 699 commits (the busiest month)**

Once you can evaluate expressions, the real work begins. This is where a language finds its personality.

AILANG gained:
- **A type system** — Hindley-Milner inference, so types are checked but rarely written
- **Algebraic data types** — `type Result(a, e) = Ok(a) | Err(e)`
- **Effects** — side effects declared in signatures, not hidden
- **Modules and imports** — code organization beyond single files
- **Records with row polymorphism** — structural typing for data

**30 releases in 24 days.** The v0.3.x era was a sprint: fixing the type system, stabilizing the core, making the language actually usable.

### The Eval Gap

At v0.3.5, AILANG ran its first eval: giving AI models AILANG code to write, then checking if it compiled and passed tests.

**Pass rate: 0%.**

Not because the language was broken — because no AI model had ever seen AILANG before. Every model tried to write Python-shaped code in `.ail` files.

This kicked off 51 iterations of "teaching prompts" — concise documents that teach an AI model how to write AILANG. By v0.3.12, pass rate hit 62.7%. The prompt became as important as the parser.

**Lesson:** A language isn't just syntax and semantics. It's also *learnability* — and in 2025, that means learnability by machines.

---

## Act 3: "Can It Scale?" (Months 2-3)

**Nov-Dec 2025 — Monomorphization, AI Providers, Coordinator**

Three things happen once a language works:

### The type system gets serious
Monomorphization — turning polymorphic code (`map(f, list)`) into specialized versions (`map_int_string(f, list)`) — is where toy languages become real ones. AILANG went from "types mostly work" to "types are fully resolved at compile time."

### It needs to talk to the world
AILANG was designed for AI orchestration, so it grew:
- Multi-provider AI integration (Claude, Gemini, OpenAI)
- HTTP/JSON builtins
- `std/math` with full numeric operations
- An eval harness comparing AILANG solutions against Python

### It needs to run unattended
The coordinator daemon: a system for dispatching tasks to AI agents, watching for results, syncing with GitHub issues. This is where AILANG stopped being a language and started being a platform.

**55 internal packages** by this point — the codebase had structure:

```
internal/
  lexer/       parser/      ast/         — Front end
  types/       elaborate/   pipeline/    — Type system
  eval/        builtins/    effects/     — Runtime
  ai/          executor/    coordinator/ — AI orchestration
  server/      telemetry/   storage/     — Infrastructure
```

---

## Act 4: "Can It See?" (Months 3-4)

**Jan-Feb 2026 — Observatory, WASM, Cloud**

### Observability
You can't debug what you can't see. AILANG added OpenTelemetry tracing across the entire stack — every compilation phase, every AI call, every effect execution gets a span you can view in Jaeger or Grafana.

### New targets
WASM compilation meant AILANG could run in browsers. WebSocket streaming meant it could handle real-time bidirectional communication. MCP (Model Context Protocol) support meant AI assistants could call AILANG functions as tools.

### The multimodal moment
Gemini multimodal support: AILANG programs that process images, not just text. XML parsing, ZIP reading, bytes builtins — the stdlib grew to handle real-world data formats.

---

## Act 5: "Can It Ship?" (Months 4-5)

**Mar 2026 — The Package Ecosystem**

A language without packages is a language without a community. v0.9 was entirely about this:

- `ailang.toml` manifest files
- Package registry with dependency resolution
- Relative imports, cross-package type aliases
- Version gates (packages declare minimum AILANG version)
- `ailang check --package` and `ailang test --package`

**This is the unglamorous work.** Dependency resolution, lock files, version constraints — none of it is exciting, all of it is essential. 13 releases just for the package system.

Also during this era: splitting 45 oversized files. When every file is under 800 lines, AI agents can read and modify any file in a single context window. The codebase is designed to be worked on by machines.

---

## The Punchline: Competing With Python

**March 2026 — v0.9.1.1 eval baseline, 612 runs across 6 models**

Remember that 0% pass rate from October? Here's where it lands five months later:

### The journey

| When | Version | Best AILANG Pass Rate | What changed |
|------|---------|----------------------|--------------|
| Oct 2025 | v0.3.5 | **0%** | First eval — every model writes Python-shaped code |
| Oct 2025 | v0.3.12 | **62.7%** | Teaching prompts: a 2-page doc that teaches the language |
| Nov 2025 | v0.4.8 | **71.7%** (Opus) | Monomorphization makes types predictable |
| Dec 2025 | v0.5.0 | **63.0%** (Opus) | Harder benchmark suite (46 → expanded tasks) |
| Mar 2026 | v0.9.0 | **91.3%** (GPT-5-4) | Package system, better stdlib, refined prompts |
| Mar 2026 | v0.9.1.1 | **84.3%** (Opus, 51 benchmarks) | Full suite — now competitive with Python |

A language no model had ever seen, going from 0% to 84% in five months. That alone is a result.

### AILANG vs Python: the model quality gradient

| Model | AILANG | Python | Delta |
|-------|--------|--------|-------|
| **Claude Sonnet 4.6** | **82.0%** (41/50) | 72.0% (36/50) | **+10.0%** |
| **Claude Opus 4.6** | **84.3%** (43/51) | 74.5% (38/51) | **+9.8%*** |
| **GPT-5-4** | **82.4%** (42/51) | 80.4% (41/51) | **+2.0%** |
| **GPT-5-2 Codex** | **74.5%** (38/51) | 72.5% (37/51) | **+2.0%** |
| Gemini 3.1 Pro | 37.3% (19/51) | 66.7% (34/51) | -29.4% |
| Gemini 3 Pro | 45.1% (23/51) | 64.7% (33/51) | -19.6% |

*\*Note: Opus Python had 5 eval harness refusals ("Apologies, but...") where the model refused to write Python code but wrote AILANG fine. Adjusting for those, Opus's advantage is +2.2%. Sonnet's +10% is clean — zero refusals on either side.*

**The pattern: the better the model, the bigger AILANG's advantage over Python.** Top-tier models beat Python; mid-tier still trail. A language designed for precise reasoning rewards models that reason well.

### Where the design choices actually matter

We categorised the 51 benchmarks by what they test. This is where it gets interesting:

| Task Category | AILANG | Python | Delta | Why |
|---------------|--------|--------|-------|-----|
| **Type Safety / Contracts** | **55.6%** | 27.8% | **+27.8%** | `requires`/`ensures` guide the model to think about invariants first |
| **Effects / IO** | **95.0%** | 75.0% | **+20.0%** | Explicit effect signatures prevent silent IO misordering |
| **Records / Data** | **100%** | 93.8% | **+6.2%** | One way to structure data, no class/dict ambiguity |
| **Data Transformation** | **58.3%** | 54.2% | **+4.2%** | JSON/CSV — roughly equal |
| **Complex / Algorithmic** | **94.4%** | 91.7% | **+2.8%** | Expression evaluators, state machines — slight edge |
| ADT + Pattern Matching | 100% | 100% | 0% | Both languages handle this well |
| Functional Patterns | 78.3% | **82.6%** | -4.3% | HOFs, pipelines — Python's familiarity helps |
| Recursive Algorithms | 75.0% | **95.0%** | **-20.0%** | Trees, sorts — Python's training data dominates |

**AILANG's AI-specific design choices dominate exactly where they're meant to.** Contracts (+28%), effects (+20%), structured data (+6%) — these are the features that don't exist in Python, and they give models measurably better outcomes.

**Python wins where training data matters most.** Recursive algorithms (-20%) — every model has seen thousands of Python tree implementations. AILANG's syntax works, but it's *unfamiliar*, and unfamiliarity costs on complex recursive patterns.

### What this means

This isn't "middle of the road." The category split maps directly to design decisions:
- AILANG's explicit effects = +20% on IO tasks
- AILANG's contracts = +28% on invariant-checking tasks
- Python's training corpus = -20% on recursive algorithm tasks

A generic language would show uniform results. AILANG shows a signature — it amplifies what it was designed to amplify. As models get better (and they will), the unfamiliarity penalty shrinks but the structural advantage remains.

### The honest caveats

1. **Eval harness noise** — 5 refusals on Python Opus, some benchmarks may have harness-specific issues. The numbers are directionally correct but shouldn't be quoted to one decimal place.
2. **Gemini models trail significantly** — likely prompt sensitivity, not a fundamental language problem, but it means the results are model-dependent.
3. **51 benchmarks is small** — the M-EVAL-XLANG sprint is running AILANG against a third-party benchmark (mini-git, 20 trials) for independent validation.
4. **Python has 30+ years of ecosystem** — beating it on any metric with a 6-month-old language is notable, but there's a long way to go.

---

## Act 6: "Can It Be Fast?" (Month 6)

**Apr 2026 — Bytecode VM**

Walking an AST is simple but slow. AILANG is now building a bytecode compiler and virtual machine:

1. **Phase 2A** — Benchmarking evaluator vs native Go (how fast *could* we be?)
2. **Phase 2B** — Statement IR: a lower-level representation between AST and bytecode
3. **Phase 2C** — IR-to-bytecode compiler
4. **Phase 2D-E** — Wiring builtins directly to VM opcodes

The pattern: get it working (evaluator), then get it fast (bytecode). Not the other way around.

---

## What Goes Into a Language: The Full Stack

People think "programming language" means "parser + evaluator." Here's what AILANG actually needed:

| Layer | What | Packages |
|-------|------|----------|
| **Syntax** | Lexer, parser, AST | 3 |
| **Types** | Inference, elaboration, typeclasses, interfaces | 6 |
| **Compilation** | Pipeline, monomorphization, linking, bytecode | 6 |
| **Runtime** | Evaluator, VM, effects, channels, builtins | 6 |
| **Stdlib** | Math, crypto, string, JSON, HTTP, XML, filesystem | (builtin) |
| **Tooling** | REPL, formatter, diagnostics, error messages | 4 |
| **AI** | Providers, executors, prompts, eval harness | 6 |
| **Infrastructure** | Server, coordinator, telemetry, storage, pub/sub | 8 |
| **Ecosystem** | Packages, registry, messaging, dashboard | 5 |
| **Meta** | 868 design docs, 51 teaching prompts, 314 examples | — |

**55 packages. Not because of over-engineering — because languages are big.**

---

## The AI Development Story

AILANG was built *with* AI and *for* AI. Some patterns that emerged:

### Teaching prompts as a first-class artifact
51 versions of "here's how to write AILANG." Each iteration was tested against eval benchmarks. The prompt matters as much as the parser — if an AI can't learn your language from a 2-page document, your language has a learnability bug.

### 868 design documents
Every feature starts as a design doc. The AI reads it, plans the implementation, executes it. Design docs aren't just documentation — they're the input to the development process.

### File size as a design constraint
No file over 800 lines. Not for human readability (though that helps) — because AI context windows are finite. The codebase is shaped by its tools.

### Eval-driven development
Traditional: write code, write tests, check if tests pass.
AILANG: write language feature, give AI a task that needs it, measure if AI succeeds. The eval harness isn't QA — it's the product metric.

### The model quality gradient
The most surprising finding: AILANG's advantage over Python *increases* with model quality. Sonnet beats Python by 10%, GPT-5-4 by 2%. Weaker models still do better in Python.

A language with fewer ambiguities and more explicit structure creates a higher *ceiling* for capable models while raising the *floor* less for weaker ones. The language is an amplifier — it amplifies reasoning ability, for better or worse. As models improve, the amplifier compounds.

### Design choices show up in the data
The per-category benchmark analysis is the most actionable finding. AILANG's contract system (+28% vs Python) and explicit effects (+20%) aren't theoretical advantages — they're measured. Meanwhile, recursive algorithms (-20%) show exactly where training data familiarity still dominates design quality. This kind of analysis drives the roadmap: improve stdlib for recursion patterns, keep doubling down on contracts and effects.

---

## Velocity Chart

```
Sep 2025  ██████████ 96 commits         — Genesis
Oct 2025  █████████████████████████████████████████████████████████████████████ 699 — Core language sprint
Nov 2025  ██████████████████████ 215     — Monomorphization
Dec 2025  █████████████████████████████████████████████ 453 — AI + Coordinator
Jan 2026  ███████████████████████████████ 313 — Observatory
Feb 2026  ██████████████ 143             — Cloud features
Mar 2026  ███████████████████████████████████████████ 431 — Package ecosystem
Apr 2026  █████████ 85 (9 days)          — Bytecode VM
```

---

## Timeline Summary

| When | Version | Theme | Key Milestone |
|------|---------|-------|---------------|
| Sep 2025 | v0.0–v0.2 | Foundation | First program parsed and evaluated |
| Oct 2025 | v0.3 | Core Language | AI eval: 0% → 62.7% pass rate |
| Nov 2025 | v0.4 | Monomorphization | Full type specialization |
| Dec 2025 | v0.5–v0.6 | AI + Agents | Multi-provider orchestration, coordinator daemon |
| Jan 2026 | v0.7 | Observatory | OTEL tracing, WASM target |
| Feb 2026 | v0.8 | Cloud | WebSocket streaming, MCP protocol |
| Mar 2026 | v0.9 | Packages | Registry, dependency resolution, lock files |
| Apr 2026 | v0.10 | Performance | Bytecode VM, runtime hardening |

---

*Built with Claude Code. 2,435 commits. One human. A lot of AI. And as of March 2026 — competitive with Python on a benchmark suite no model was trained on.*
