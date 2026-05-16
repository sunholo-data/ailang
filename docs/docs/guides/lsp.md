# `ailang lsp` — Language Server for AI agents and IDEs

`ailang lsp` is a [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) server for `.ail` source files. It exposes the AILANG type checker, elaborator, and module loader as **structured wire-protocol responses** so any LSP client — a Claude Code session, a VS Code extension, an Emacs `lsp-mode` config — gets diagnostics, hover types, go-to-definition, find-references, and document symbols without having to shell out to `ailang check` and parse text.

> **Why it exists.** AILANG's pitch — *machine decidability and semantic transparency* — assumes the analyses are reachable by tools. The LSP is the natural exposure surface, especially for AI coding agents working at scale on `.ail` codebases (where every grep-then-reread costs tokens that a typed `definition` request would not). See the [design doc](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_21_0/m-ailang-lsp-for-ai.md) for the position-reversal background.

## What it does (MVP)

| Capability | Behaviour |
|---|---|
| `textDocument/diagnostics` | On `didOpen` and `didSave`, runs the type-check pipeline against the buffer and pushes structured diagnostics over `publishDiagnostics`. Errors carry source positions when the underlying error type does. |
| `textDocument/hover` | On a callsite of a top-level export, returns the inferred type signature (`add : (int, int) -> int`) as a Markdown body. Local bindings get a documented "deferred" message. |
| `textDocument/definition` | Returns a `Location` pointing at the function name (NOT the parser-anchored keyword position) — same-file, session-open documents, and best-effort stdlib. |
| `textDocument/references` | Walks every document opened in the current LSP session for matching identifiers. Honors `includeDeclaration`. |
| `textDocument/documentSymbol` | Hierarchical outline: top-level functions, ADTs (with constructor children), type classes. Imports omitted on purpose. |

**Out of MVP scope** (deferred to follow-up milestones, see CHANGELOG entry for the full list): completion, signature help, code actions, formatting, rename, semantic tokens, code lens, inlay hints, per-keystroke incremental re-typecheck, workspace-wide reference scan, and shared cross-session server caching.

## Installing for use with Claude Code (recommended)

The repo ships an `ailang-lsp` plugin in its local marketplace at [`.claude-plugin/marketplace.json`](https://github.com/sunholo-data/ailang/blob/dev/.claude-plugin/marketplace.json) — installing it in your Claude Code session takes two commands:

```bash
/plugin marketplace add /path/to/your/ailang/checkout
/plugin install ailang-lsp@ailang-tools
```

After install, `/plugin` should list `ailang-lsp` under **Installed** with no entry under **Errors**. Open a `.ail` file in a Claude Code session and the agent will get diagnostics, hover, and the rest automatically — no further wiring.

**Prerequisite**: the `ailang` binary must be on your PATH. From the repo root:

```bash
make install   # installs to /Users/<you>/go/bin/ailang
```

Verify:

```bash
ailang lsp --stdio < /dev/null   # reads zero LSP frames; should print nothing and exit cleanly
which ailang                     # /Users/<you>/go/bin/ailang
```

## Installing for a generic LSP client (VS Code / Emacs / Helix / nvim)

Any LSP client can connect to `ailang lsp --stdio`. The client config needs to:

1. Spawn `ailang lsp --stdio` as the language server for files matching `*.ail`.
2. Set the language ID to `ailang`.

Example (VS Code, via the [generic LSP extension](https://marketplace.visualstudio.com/items?itemName=vsls-contrib.lsp-client)):

```json
{
  "lspClient.serverCommands": {
    "ailang": {
      "command": "ailang",
      "args": ["lsp", "--stdio"],
      "documentSelector": ["ailang"]
    }
  },
  "files.associations": { "*.ail": "ailang" }
}
```

## Workspace setup (avoid MOD010 surprises)

AILANG's module loader expects the file's `module` declaration to match its on-disk path relative to a project root. When `ailang lsp` is started from a Claude Code session whose working directory IS the project root (the normal case), this Just Works. When started against an absolute path *outside* a workspace context, the loader emits `MOD010` and cross-module navigation falls back to "no result".

For agent use: open the `.ail` file from inside its workspace (Claude Code's `rootUri` should be the project root containing the `.ail` files). The diagnostics path will surface MOD010 with a `Fix:` suggestion if you hit it.

For human use under VS Code: open the *folder*, not the file.

## Token-cost rationale (for AI-agent users)

The MVP capability set is deliberately tuned for the AI consumer, not the human IDE consumer. An AI agent navigating a 50k-LOC `.ail` codebase without an LSP burns tokens on:

- `Grep` for symbol definitions (often false-positive: matches in strings/comments)
- `Read` of whole candidate files to disambiguate
- `Bash: ailang check` to discover type errors after edits

The five MVP LSP capabilities replace these one-for-one with structured single-round-trip responses. M-AILANG-LSP-FOR-AI M6 ships a small token-cost eval harness ([`bench/lsp_token_cost/`](https://github.com/sunholo-data/ailang/tree/dev/bench/lsp_token_cost)) that measures the LSP-on vs LSP-off delta on a fixed task; see the README in that directory for the published number.

## Implementation reference

- Server: [`internal/lsp/`](https://github.com/sunholo-data/ailang/tree/dev/internal/lsp)
- Subcommand entry: [`cmd/ailang/lsp.go`](https://github.com/sunholo-data/ailang/blob/dev/cmd/ailang/lsp.go)
- Cross-module fixture: [`examples/lsp_xref_fixture/`](https://github.com/sunholo-data/ailang/tree/dev/examples/lsp_xref_fixture)
- Design doc: [`design_docs/planned/v0_21_0/m-ailang-lsp-for-ai.md`](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_21_0/m-ailang-lsp-for-ai.md)
- LSP libraries: [`go.lsp.dev/protocol`](https://pkg.go.dev/go.lsp.dev/protocol), [`go.lsp.dev/jsonrpc2`](https://pkg.go.dev/go.lsp.dev/jsonrpc2) (same family gopls uses internally)
