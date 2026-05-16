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

## Prerequisite (both install paths)

You need the `ailang` binary on your PATH. **No AILANG source clone required** — `ailang lsp` is built into the same binary that runs `ailang check`, `ailang run`, etc.

```bash
# Install from a release (no clone required):
curl -sSL https://ailang.sunholo.com/install.sh | sh

# Verify:
which ailang        # should print a path
ailang lsp -h       # should print usage for the lsp subcommand
```

If you've cloned this repo and want to use a development build instead, `make install` from the repo root installs the dirty build to `~/go/bin/ailang`.

## Installing for use with Claude Code (recommended for AI agents)

The `ailang-lsp` plugin ships in the public `ailang-marketplace` hosted at [`sunholo-data/ailang_bootstrap`](https://github.com/sunholo-data/ailang_bootstrap). Install in your Claude Code session — **no clone required**:

```bash
/plugin marketplace add sunholo-data/ailang_bootstrap
/plugin install ailang-lsp@ailang-marketplace
```

After install, `/plugin` should list `ailang-lsp` under **Installed** with no entry under **Errors**. Open a `.ail` file in a Claude Code session and the agent will get diagnostics, hover, and the rest automatically — no further wiring.

> **AILANG contributors**: this repo also ships a developer-facing marketplace at [`.claude-plugin/marketplace.json`](https://github.com/sunholo-data/ailang/blob/dev/.claude-plugin/marketplace.json) (`ailang-tools`). Install from a local checkout via `/plugin marketplace add /path/to/your/ailang/checkout` + `/plugin install ailang-lsp@ailang-tools` to dogfood against the dev binary.

## Installing for VS Code (recommended for humans)

One command — the `ailang` binary ships an embedded VS Code extension that installs both syntax highlighting AND the LSP wiring:

```bash
ailang editor install vscode
```

Then in VS Code: **Cmd+Shift+P → Developer: Reload Window**. Open any `.ail` file — colors appear immediately, and diagnostics arrive inline on save. The extension auto-spawns `ailang lsp --stdio` for `.ail` files via the standard `vscode-languageclient` runtime.

**Verifying it works:**
1. After reload, open any `.ail` file
2. Check `ps aux | grep "ailang lsp"` — you should see one process
3. Errors in VS Code's Output panel under "AILANG Language Server" if anything went wrong
4. To uninstall: `ailang editor uninstall vscode`

The extension version that ships embedded is `0.3.0` — bumps to the AILANG binary include the latest extension version. Re-run `ailang editor install vscode` after upgrading `ailang` to pick up extension changes.

### For other LSP clients (Emacs / Helix / nvim)

Any LSP client can connect to `ailang lsp --stdio`. Configure your client to:

1. Spawn `ailang lsp --stdio` as the language server for files matching `*.ail`
2. Set the language ID to `ailang`

For VS Code users who'd rather use a generic LSP client extension (instead of the embedded one above), the [Generic LSP Client (v2)](https://marketplace.visualstudio.com/items?itemName=zsol.vscode-glspc) extension works with:

```json
{
  "files.associations": { "*.ail": "ailang" },
  "glspc.server.command": "ailang",
  "glspc.server.commandArguments": ["lsp", "--stdio"],
  "glspc.server.languageId": ["ailang"]
}
```

This path skips the syntax-highlighting bundle but is useful if you want full control over the LSP-client lifecycle.

## Workspace behaviour

The LSP always runs with `RelaxModules` on, so the `MOD010` module-path-vs-file-path strict check that fires from `ailang check` is downgraded inside the LSP. This is by design — the LSP receives absolute file URIs from the editor and has no reliable way to compute the relative-to-project-root path the strict mode wants, so without the relax-mode every file would surface only an MOD010 error and drown out real diagnostics. Strict checking stays the default for `ailang check` (CI gate); the LSP is for navigation + edit-time feedback.

If you do want strict module-path checking at the same time, run `ailang check <file>` from the terminal — it's the same compiler, just with strict mode on.

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
