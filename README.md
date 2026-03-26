<p align="center">
  <img src="docs/static/img/ailang-logo.svg" alt="AILANG Logo" width="128" height="128">
</p>

# AILANG: The Deterministic Language for AI Coders

<!-- EXAMPLES_STATUS_START -->
![Examples](https://img.shields.io/badge/examples-152%20passing-brightgreen.svg)
![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)

**152/152 examples passing (100%)** | [Full status](https://ailang.sunholo.com/docs/examples)
<!-- EXAMPLES_STATUS_END -->

AILANG is a purely functional, effect-typed language designed as a **deterministic execution substrate** for AI-generated code. Every construct has deterministic semantics that can be reflected, verified, and serialized.

**[Documentation](https://ailang.sunholo.com/)** | **[Examples](https://ailang.sunholo.com/docs/examples)** | **[Live Demos](https://www.sunholo.com/ailang-demos/)** | **[Vision](https://ailang.sunholo.com/docs/vision)** | **[Benchmarks](https://ailang.sunholo.com/docs/benchmarks/performance)**

---

## Quick Start

AILANG is designed to be used by AI agents. The easiest way to get started is via your agent's plugin/extension system.

### With Claude Code

```
/plugin marketplace add sunholo-data/ailang_bootstrap
/plugin install ailang
```

### With Gemini CLI

```bash
gemini extensions install https://github.com/sunholo-data/ailang_bootstrap.git
```

### What the Plugin Provides

- **AILANG binary** - Auto-installed for your platform
- **MCP tools** - `ailang_prompt`, `ailang_check`, `ailang_run`, `ailang_builtins`
- **Slash commands** - `/ailang:prompt`, `/ailang:new`, `/ailang:run`, `/ailang:challenge`
- **Teaching prompts** - Current syntax rules loaded automatically

Once installed, just ask your agent to write AILANG code - it handles the rest.

See [ailang_bootstrap](https://github.com/sunholo-data/ailang_bootstrap) for details.

### Quick Install

```bash
curl -fsSL https://ailang.sunholo.com/install.sh | bash
```

Detects your OS/architecture automatically. Pin a version with `VERSION=v0.9.0` before the curl.

<details>
<summary>Click to expand manual installation instructions</summary>

```bash
# macOS (Apple Silicon)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/darwin.arm64.ailang.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/darwin.x64.ailang.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# Linux
curl -L https://github.com/sunholo-data/ailang/releases/latest/download/linux.x64.ailang.tar.gz | tar -xz
sudo mv ailang /usr/local/bin/

# From source
git clone https://github.com/sunholo-data/ailang.git
cd ailang && make install

# Verify
ailang --version
```

</details>

For complete setup instructions, see the [Getting Started Guide](https://ailang.sunholo.com/docs/guides/getting-started).

### Hello World

```ailang
module examples/hello

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello from AILANG!")
}
```

```bash
ailang run --caps IO examples/hello.ail
# Output: Hello from AILANG!
```

### Interactive REPL

```bash
ailang repl

λ> 1 + 2
3 :: Int

λ> let double = \x. x * 2 in double(21)
42 :: Int

λ> :type \x. x + x
\x. x + x :: ∀α. Num α ⇒ α → α

λ> :quit
```

---

## Key Features

- **Pure functional** - Lambda calculus, closures, pattern matching, ADTs
- **Type inference** - Hindley-Milner with row polymorphism
- **Effect system** - Capability-based security (IO, FS, Net, Clock, AI)
- **Deterministic** - Replayable execution, structured traces
- **AI-first** - Designed for machine reasoning, not human convenience

Learn more: [Why AILANG?](https://ailang.sunholo.com/docs/vision) | [No Loops Design](https://ailang.sunholo.com/docs/reference/no-loops) | [Go Interop](https://ailang.sunholo.com/docs/guides/go-interop)

---

## Development

```bash
make install    # Build and install
make test       # Run all tests
make repl       # Start REPL
make lint       # Run linter
```

**Guides:**
- [Development Guide](https://ailang.sunholo.com/docs/guides/development)
- [CONTRIBUTING.md](docs/CONTRIBUTING.md)
- [CLAUDE.md](CLAUDE.md) - For AI development assistants

---

## Project Structure

```
ailang/
├── cmd/ailang/     # CLI
├── internal/       # Compiler (lexer, parser, types, eval, effects)
├── stdlib/         # Standard library (std/io, std/fs, std/json, std/zip, std/xml, etc.)
├── examples/       # Example programs (97 files)
├── docs/           # Documentation website source
└── design_docs/    # Design documents
```

---

## License

Apache 2.0 - See [LICENSE](LICENSE)

AILANG draws inspiration from Haskell, OCaml, Rust, and Idris/Agda.

---

*For AI agents: Deterministic functional language with Hindley-Milner type inference, algebraic effects, and explicit effect tracking. See [CLAUDE.md](CLAUDE.md) and [Documentation](https://ailang.sunholo.com/) for capabilities.*
