<p align="center">
  <img src="docs/static/img/ailang-logo.svg" alt="AILANG Logo" width="128" height="128">
</p>

# AILANG: The Deterministic Language for AI Coders

<!-- EXAMPLES_STATUS_START -->
![Examples](https://img.shields.io/endpoint?url=https://ailang.sunholo.com/badges/examples.json)
![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)

[Example status](https://ailang.sunholo.com/docs/examples)
<!-- EXAMPLES_STATUS_END -->

[![Reliability](https://sonarcloud.io/api/project_badges/measure?project=sunholo-data_ailang&metric=reliability_rating)](https://sonarcloud.io/component_measures?id=sunholo-data_ailang&metric=reliability_rating)
[![Security](https://sonarcloud.io/api/project_badges/measure?project=sunholo-data_ailang&metric=security_rating)](https://sonarcloud.io/component_measures?id=sunholo-data_ailang&metric=security_rating)
[![Maintainability](https://sonarcloud.io/api/project_badges/measure?project=sunholo-data_ailang&metric=sqale_rating)](https://sonarcloud.io/component_measures?id=sunholo-data_ailang&metric=sqale_rating)
[![CodeQL](https://github.com/sunholo-data/ailang/actions/workflows/codeql.yml/badge.svg)](https://github.com/sunholo-data/ailang/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/sunholo-data/ailang/badge)](https://securityscorecards.dev/viewer/?uri=github.com/sunholo-data/ailang)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/12676/badge)](https://www.bestpractices.dev/projects/12676)

> **Third-party verification.** AILANG is written autonomously by AI agents via its own [coordinator](https://ailang.sunholo.com/docs/guides/coordinator). The badges above are independent static-analysis and supply-chain scores — not self-reported. The [benchmark dashboard](https://ailang.sunholo.com/docs/benchmarks/performance) publishes the live correctness signal across current benchmark and model cohorts.

AILANG is a purely functional, effect-typed language designed as a **deterministic execution substrate** for AI-generated code. Every construct has deterministic semantics that can be reflected, verified, and serialized.

**[Documentation](https://ailang.sunholo.com/)** | **[Examples](https://ailang.sunholo.com/docs/examples)** | **[Live Demos](https://www.sunholo.com/ailang-demos/)** | **[Vision](https://ailang.sunholo.com/docs/vision)** | **[Benchmarks](https://ailang.sunholo.com/docs/benchmarks/performance)**

---

## Quick Start

AILANG is designed to be used by AI coding agents. The
[AILANG Bootstrap](https://github.com/sunholo-data/ailang_bootstrap) packages the
current language guidance, reusable skills, and local MCP tools for Claude Code
and OpenAI Codex.

Install the AILANG CLI first:

```bash
curl -fsSL https://ailang.sunholo.com/install.sh | bash
ailang --version
```

The agent integrations call the local `ailang` executable, so it must be
available on `PATH`.

### With Claude Code

```text
/plugin marketplace add sunholo-data/ailang_bootstrap
/plugin install ailang@sunholo-data/ailang_bootstrap
```

### With OpenAI Codex

AILANG Bootstrap also ships a Codex plugin manifest. Add its stable marketplace:

```bash
# Required by the plugin's local MCP server on first launch
node --version
npm --version

codex plugin marketplace add sunholo-data/ailang_bootstrap --ref stable
codex plugin add ailang@ailang-marketplace
codex plugin list
```

Start a new Codex session after installation. Alternatively, after adding the
marketplace, launch `codex`, enter `/plugins`, and install `ailang` from the
AILANG marketplace. The plugin can also be installed from the Plugins directory
in the Codex desktop app. Codex plugins are not currently available in the IDE
extension.

Codex reads this repository's [`AGENTS.md`](AGENTS.md) automatically, whether or
not the plugin is installed. The plugin adds the reusable AILANG skills and MCP
tools; `AGENTS.md` supplies repository-specific workflows and guardrails.

### What the Agent Package Provides

- **CLI integration** - Skills and tools use the separately installed `ailang` executable
- **MCP tools** - `ailang_prompt`, `ailang_check`, `ailang_run`, `ailang_builtins`, `ailang_eval`
- **Agent skills** - AILANG authoring, debugging, messaging, design-doc, and sprint workflows
- **Teaching guidance** - Current syntax rules via repository guidance, skills, and `ailang prompt`

The packaging is intentionally native to each host:

| Capability | Claude Code | Codex |
|---|---|---|
| Persistent guidance | `CLAUDE.md` | `AGENTS.md` |
| Reusable workflows | Plugin skills | Plugin skills |
| Local AILANG MCP tools | Yes | Yes |
| Host-native commands | Claude slash commands | Skills and MCP tools |
| AILANG Bootstrap hooks | Claude hooks | Not currently packaged |

Once installed, ask the agent to write AILANG code. All agents should run
`ailang prompt` before authoring `.ail` files; it is the installed CLI's source
for current syntax and idioms.

See [ailang_bootstrap](https://github.com/sunholo-data/ailang_bootstrap) for details.

The installer detects your OS and architecture automatically. To pin a release:

```bash
curl -fsSL https://ailang.sunholo.com/install.sh | VERSION=v0.30.0 bash
```

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
module examples/hello_world

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello, World!")
}
```

```bash
ailang run --caps IO examples/hello_world.ail
# Output: Hello, World!
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
- **LSP** - `ailang lsp --stdio` ships in the binary: diagnostics, hover types, go-to-def, references, document symbols. One-command VS Code install: `ailang editor install vscode` ([guide](https://ailang.sunholo.com/docs/guides/lsp))

Learn more: [Why AILANG?](https://ailang.sunholo.com/docs/vision) | [No Loops Design](https://ailang.sunholo.com/docs/reference/no-loops) | [Go Interop](https://ailang.sunholo.com/docs/guides/go-interop) | [Stability Promise (1.x)](https://ailang.sunholo.com/docs/reference/stability)

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
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [AGENTS.md](AGENTS.md) - Cross-agent repository instructions (including Codex)
- [CLAUDE.md](CLAUDE.md) - Detailed AILANG operational workflows

---

## Project Structure

```
ailang/
├── cmd/ailang/     # CLI
├── internal/       # Compiler (lexer, parser, types, eval, effects)
├── std/            # Standard library (std/io, std/fs, std/json, std/zip, std/xml, etc.)
├── examples/       # Example and reference programs
├── docs/           # Documentation website source
└── design_docs/    # Design documents
```

---

## License

Apache 2.0 - See [LICENSE](LICENSE)

AILANG draws inspiration from Haskell, OCaml, Rust, and Idris/Agda.

---

*For AI agents: Deterministic functional language with Hindley-Milner type inference, algebraic effects, and explicit effect tracking. Read [AGENTS.md](AGENTS.md), then use `ailang prompt` for current language guidance.*
