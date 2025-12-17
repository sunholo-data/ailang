# 🧠 AILANG: The Deterministic Language for AI Coders

![CI](https://github.com/sunholo-data/ailang/workflows/CI/badge.svg)
![Coverage](https://img.shields.io/badge/coverage-39.1%25-orange.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)
![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)

AILANG is a purely functional, effect-typed language designed as a **deterministic execution and evaluation substrate** for AI-generated code. Unlike human-oriented languages built around IDEs, concurrency, and sugar, AILANG's design goal is **machine decidability, semantic transparency, and compositional determinism**.

AILANG exists to ensure that when AI systems improve themselves, the improvement is real.

---

## 🧩 Core Philosophy

**For humans, a language is a tool for expression.**
**For AIs, it's a substrate for reasoning.**

AILANG minimizes ambiguity and maximizes predictability. Every construct — type, effect, or expression — has **deterministic semantics** that can be reflected, verified, and serialized.

## What AILANG Is (and Is Not)

AILANG is **not** an autonomous agent framework.
AILANG is **not** a productivity language for humans.

AILANG **is**:
- A deterministic semantic substrate for AI-generated programs
- An evaluation harness where reasoning, testing, and execution align
- A language where programs can be safely re-written, re-run, and compared by machines

Autonomous agents, self-training loops, and reflective rewriting are **enabled by AILANG’s guarantees**, not embedded as opaque runtime behavior.

Further reading on answering common misconceptions:
- AILANG is not another AI framework.
- AILANG is not trying to be a "better python"
- AILANG as a Semantic Control Surface

Read more about [AILANG vs Agents](docs/docs/roadmap/ailang-vs-agents.md) concepts.

## 🏗️ Architecture

AILANG is built in layers, from stable foundations to cutting-edge features:

**Stable Core:**
- Pure functional semantics with algebraic data types (ADTs)
- Hindley-Milner type inference with row polymorphism
- Effect system with capability-based security (IO, FS, Net, Clock)
- Deterministic evaluator with explicit effect tracking

**In Development:**
- Reflection & meta-programming (typed quasiquotes, structural type classes — design complete, implementation in progress)
- Deterministic tooling (normalize, suggest-imports, trace export)
- Schema registry for machine-readable type definitions

For detailed architecture documentation, see [docs/architecture/](docs/architecture/)


## 🔮 Vision & Roadmap

AILANG aims to be the first language optimized for autonomous AI code synthesis. Our roadmap focuses on:
- **Deterministic tooling** - Canonical formatting, import suggestion, trace export
- **Reflection & meta-programming** - Typed quasiquotes, structural type classes
- **Schema registry** - Machine-readable type/effect definitions
- **Cognitive autonomy** - Full round-trip reasoning with machine-verifiable self-modification loops

For detailed roadmap and design philosophy, see:
- [VISION.md](docs/VISION.md) - Long-term vision and design principles
- [CHANGELOG.md](CHANGELOG.md) - Completed features and upcoming releases

---

## 💡 Why AILANG Works Better for AIs

| Human Need | Human Feature | AI Equivalent in AILANG |
|-----------|---------------|------------------------|
| IDE assistance | LSP / autocompletion | Deterministic type/query API |
| Asynchronous code | Threads / goroutines | Static task DAGs with effects |
| Code reuse | Inheritance / traits | Structural reflection & records |
| Debugging | Interactive debugger | Replayable evaluation trace |
| Logging | `print` / `console` | `--emit-trace jsonl` structured logs |
| Macros | text substitution | Typed quasiquotes (semantic macros - (planned - v0.6+) |

---

## 🔁 Why AILANG Has No Loops (and Never Will)

AILANG intentionally omits `for`, `while`, and other open-ended loop constructs.
This isn't a missing feature — it's a design decision rooted in **determinism and compositional reasoning**.

### 🧠 For Humans, Loops Express Control. For AIs, Loops Obscure Structure.

Traditional loops compress time into mutable state:

```python
sum = 0
for i in range(0, 10):
    sum = sum + i
```

This is compact for humans but **semantically opaque** for machines:
the iteration count, state shape, and termination guarantee are **implicit**.

AILANG replaces this with **total, analyzable recursion**:

```ailang
foldl(range(0, 10), 0, \acc, i. acc + i)
```

or **pattern matching**:

```ailang
func sum(list: List[Int]) -> Int {
  match list {
    [] => 0,
    [x, ...xs] => x + sum(xs)
  }
}
```

Every iteration is a **pure function over data, not time** —
which makes it statically decidable, effect-safe, and perfectly compositional.

### ⚙️ The Deterministic Iteration Principle

| Goal | Imperative Loops | AILANG Alternative |
|------|-----------------|-------------------|
| Repeat a computation | `for` / `while` | `map`, `fold`, `filter`, `rec` |
| Aggregate results | mutable accumulator | `foldl` / `foldr` |
| Early termination | `break` | `foldWhile` / `find` |
| Parallel evaluation | scheduler threads | static task DAGs |
| Verification | undecidable | total + effect-typed |

### 🧩 Benefits

- **Deterministic semantics**: iteration defined by data, not by time
- **Static totality**: no halting ambiguity
- **Composable reasoning**: works algebraically with higher-order functions
- **Easier optimization**: map/fold can fuse or parallelize safely
- **Simpler runtime**: no mutable counters or loop scopes

**No hidden state. No implicit time. Fully analyzable by both compiler and AI.**

For the formal rationale and algebraic laws, see the [Why No Loops?](https://ailang.sunholo.com/docs/reference/no-loops) documentation.

---

## Quick Start

### Installation for AI Agents (Recommended)

AILANG is designed to be used with AI coding agents. Install via your agent's extension/plugin system:

**Claude Code:**
```
/plugin marketplace add sunholo-data/ailang_bootstrap
/plugin install ailang
```

**Gemini CLI:**
```bash
gemini extensions install https://github.com/sunholo-data/ailang_bootstrap.git
```

These plugins provide:
- AILANG binary (auto-installed)
- MCP tools for type-checking and running code
- Custom slash commands (`/ailang:prompt`, `/ailang:run`, etc.)
- Teaching prompts and coding challenges
- Full stdlib documentation via `ailang builtins list --verbose`

### Manual Installation

For standalone CLI usage or development:

```bash
# macOS (Apple Silicon)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-darwin-arm64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-darwin-amd64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# Linux
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/ailang-linux-amd64.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# From source
git clone https://github.com/sunholo-data/ailang.git
cd ailang && make install

# Verify
ailang --version
```

For detailed instructions, see the [Getting Started Guide](https://ailang.sunholo.com/docs/guides/getting-started).

### MCP Server (For Custom Integrations)

If you want to integrate AILANG tools into other AI systems via [Model Context Protocol](https://modelcontextprotocol.io/):

```bash
# Clone the bootstrap repo
git clone https://github.com/sunholo-data/ailang_bootstrap.git
cd ailang_bootstrap/mcp-server

# Install dependencies and start server
npm install
npm start
```

Configure in your MCP client's `settings.json`:
```json
{
  "mcpServers": {
    "ailang-tools": {
      "command": "node",
      "args": ["/path/to/ailang_bootstrap/mcp-server/server.js"]
    }
  }
}
```

**Available MCP Tools:**
| Tool | Description |
|------|-------------|
| `ailang_prompt` | Get teaching prompt (SOURCE OF TRUTH for syntax) |
| `ailang_check` | Type-check a file |
| `ailang_run` | Run with capabilities |
| `ailang_builtins` | Full stdlib docs with examples |
| `ailang_eval` | Evaluate expression in REPL |

### Hello World (Module Execution)

```ailang
-- examples/demos/hello_io.ail
module examples/demos/hello_io

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello from AILANG v0.3.14!")
}
```

```bash
ailang run --caps IO examples/demos/hello_io.ail
# Output: Hello from AILANG v0.3.14!
```

### Interactive REPL

The REPL features full type inference and deterministic evaluation:

```bash
ailang repl

λ> 1 + 2
3 :: Int

λ> "Hello " ++ "World"
Hello World :: String

λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α

λ> :quit
```

**REPL Commands**: `:help`, `:type <expr>`, `:instances`, `:import <module>`, `:history`, `:clear`

See [REPL Commands](docs/reference/repl-commands.md) for full reference.

### Property-Based Testing

AILANG includes QuickCheck-style property-based testing for deterministic validation:

```ailang
// Unit tests
test "addition works" = 1 + 1 == 2

// Property tests (100 random cases)
property "addition commutes" (x: int, y: int) =
  x + y == y + x

property "list reversal" (xs: list(int)) =
  reverse(reverse(xs)) == xs
```

Run tests:
```bash
ailang test examples/testing_basic.ail
```

Output:
```
→ Running tests in examples/testing_basic.ail

Test Results
Module: All Tests

Tests:
  ✓ addition works

Properties:
  ✓ addition commutes (100 cases)
  ✓ list reversal (100 cases)

✓ All tests passed

3 tests: 3 passed, 0 failed, 0 skipped (0.3s)
```

**Features**:
- **Automatic shrinking**: When a property fails, finds minimal counterexample
- **Configurable generation**: Control ranges, sizes, and random seeds
- **CI/CD integration**: JSON output, exit codes, GitHub Actions examples
- **Type-aware generators**: Built-in support for all AILANG types

**Examples**:
- [Basic testing examples](examples/testing_basic.ail) - Unit tests and simple properties
- [Advanced testing examples](examples/testing_advanced.ail) - ADTs, trees, algebraic laws

**Documentation**:
- [Testing Guide](docs/TESTING.md) - Complete user documentation
- [AI Testing Guide](prompts/testing_guide_ai.md) - Property patterns for AI agents

---

## What AILANG Can Do (Implementation Status)

### ✅ Core Language

- **Pure functional programming** - Lambda calculus, closures, recursion
- **Hindley-Milner type inference** - Row polymorphism, let-polymorphism
- **Built-in type class instances** - `Num`, `Eq`, `Ord`, `Show` (structural reflection planned for future release)
- **Algebraic effects** - Capability-based security (IO, FS, Clock, Net)
- **Pattern matching** - ADTs with exhaustiveness checking
- **Module system** - Runtime execution, cross-module imports
- **Block expressions** - `{ e1; e2; e3 }` for sequencing
- **JSON support** - Parsing (`std/json.decode`), encoding (`std/json.encode`)

### ✅ Development Tools

- **M-EVAL** - AI code generation benchmarks (multi-model support)
- **M-EVAL-LOOP v2.0** - Native Go eval tools with 90%+ test coverage
- **Structured error reporting** - JSON schemas for deterministic diagnostics
- **Effect system runtime** - Hermetic testing with `MockEffContext`

## AILANG as an Evaluation Harness

AILANG is intentionally designed to host AI evaluation loops:
- Deterministic execution
- Replayable traces
- Property-based testing
- Comparable outputs across models

M-EVAL is not a benchmark bolted onto the language — it is evidence that the language semantics are stable enough to reason about AI performance itself.

### 🔜 In Development

See [design documents](design_docs/planned/) for upcoming features and development roadmap.

See **[Full Implementation Status](https://ailang.sunholo.com/docs/examples#implementation-status)** for detailed breakdown with auto-updated table of all examples.

---

## 🔗 Go Interop (v0.5.x)

AILANG can compile to Go for game development and performance-critical applications:

```bash
# Generate Go code from AILANG
ailang compile --emit-go --package-name game world.ail
```

**Features:**
- Type generation (records → Go structs)
- Extern function stubs (implement in Go)
- Deterministic output (fixed seeds)

**ABI Stability (v0.5.x):** The Go interop ABI is "stable preview":
- Primitive type mapping is stable
- Record/struct generation is stable
- Breaking changes announced in CHANGELOG
- Full stability guaranteed in v0.6.0

📖 See **[Go Interop Guide](docs/docs/guides/go-interop.md)** for complete documentation.

---

## Documentation

📖 **[Complete Documentation](https://ailang.sunholo.com/)** - Visit our full documentation site

**Quick Links:**
- **[Vision](https://ailang.sunholo.com/docs/vision)** - Why AILANG exists and what makes it different
- **[Examples](https://ailang.sunholo.com/docs/examples)** - Interactive code examples with explanations
- **[Getting Started](https://ailang.sunholo.com/docs/guides/getting-started)** - Installation and tutorial
- **[Language Reference](https://ailang.sunholo.com/docs/reference/language-syntax)** - Complete syntax guide
- **[Benchmarks](https://ailang.sunholo.com/docs/benchmarks/performance)** - AI code generation metrics (49% improvement)

---

## Development

**Quick commands:**
```bash
make install              # Build and install
make test                 # Run all tests
make repl                 # Start REPL
make run FILE=<file>      # Run example file
```

**For detailed workflows and contribution guidelines:**
- [Development Guide](https://ailang.sunholo.com/docs/guides/development)
- [CONTRIBUTING.md](docs/CONTRIBUTING.md)
- [CLAUDE.md](CLAUDE.md) - Instructions for AI development assistants

---

### Project Structure

```
ailang/
├── cmd/ailang/         # CLI entry point
├── internal/           # Core implementation
│   ├── repl/           # Interactive REPL
│   ├── lexer/          # Tokenizer
│   ├── parser/         # Parser
│   ├── types/          # Type system
│   ├── eval/           # Evaluator
│   ├── effects/        # Effect system runtime
│   ├── builtins/       # Builtin registry
│   └── eval_harness/   # AI evaluation framework
├── stdlib/             # Standard library
├── examples/           # Example programs
├── docs/               # Documentation
└── design_docs/        # Design documents
```

---

## Contributing

AILANG is an experimental language in active development. Contributions are welcome! Please see the [Development Guide](https://ailang.sunholo.com/docs/guides/development) for guidelines.

---

## ⚖️ License & Philosophy

AILANG is **open infrastructure for Cognitive DevOps** — systems that write, test, and deploy themselves deterministically.

**Our design north star: build languages AIs enjoy using.**

Apache 2.0 - See [LICENSE](LICENSE) for details.

---

## Acknowledgments

AILANG draws inspiration from:
- **Haskell** (type system, purity)
- **OCaml** (module system, effects)
- **Rust** (capability-based security)
- **Idris/Agda** (reflection and metaprogramming)

---

*For AI agents: This is a deterministic functional language with Hindley-Milner type inference, algebraic effects, and explicit effect tracking. The REPL is fully functional. Module execution works with capability-based security. See [CLAUDE.md](CLAUDE.md) and [Complete Documentation](https://ailang.sunholo.com/) for exact capabilities.*
