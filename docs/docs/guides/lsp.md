---
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# `ailang lsp` — Language Server for AI Agents and IDEs

`ailang lsp` is a [Language Server Protocol](https://microsoft.github.io/language-server-protocol/) server for `.ail` source files. It exposes AILANG's type checker, elaborator, quick-fix engine, and module loader over JSON-RPC (`ailang lsp --stdio`) for editors and AI coding agents.

## Capabilities

| Capability | Behaviour |
|---|---|
| `textDocument/diagnostics` | On `didOpen` and `didSave`, runs the type/effect checker and publishes position-accurate diagnostics (including multi-hole `TC_HOLE` diagnostics). |
| `textDocument/codeAction` | Provides `QuickFix` actions powered by `internal/quickfix` (exact parity with `ailang check --fix`). |
| `textDocument/hover` | Returns inferred type signatures (`add : (int, int) -> int`) on callsites of top-level exports. |
| `textDocument/definition` | Jumps to function definitions across same-file, open session buffers, and stdlib. |
| `textDocument/references` | Walks open session documents for identifier references (`includeDeclaration` supported). |
| `textDocument/documentSymbol` | Returns hierarchical outline for top-level functions, ADTs (with constructors), and type classes. |

## Typed Holes (`_?` and `?name`) {#typed-holes}

Write `_?` (anonymous) or `?name` (named) in any expression position to query the expected type and local bindings in scope without guessing. `ailang check` emits a `TC_HOLE` diagnostic for every hole in the module in a single pass, and `ailang fmt` preserves holes verbatim.

```ailang parse
module docs/hole_demo

export pure func formatTotal(items: [int], prefix: string) -> string =
  ?buildSummary
```

```bash
ailang check --format=agent docs/hole_demo.ail
# docs/hole_demo.ail:4:3 TC_HOLE: typed hole `?buildSummary` has expected type `string` → in scope: items: list[int], prefix: string
```

## Mechanical Quick-Fixes (`ailang check --fix` & `codeAction`)

Both `ailang check --fix` and LSP `textDocument/codeAction` share `internal/quickfix` and apply deterministic, compiler-verified edits:

| Diagnostic | Mechanical Quick-Fix |
|---|---|
| Missing effects / `pure` clash | Stamps missing `! {Effects}` onto function signature (preserving `@limit=N` / params) and strips contradictory `pure` |
| String `++` | Rewrites `"a" ++ b` on strings to `"a${b}"` interpolation (`++` is list-only) |
| `TC_MATCH_002` | Inserts `<missing_pattern> => _?` typed-hole stubs inside non-exhaustive `match` blocks |
| `PAR017` | Removes `=` before `{ ... }` in block-body functions (`func f() = { ... }` -> `func f() { ... }`) |
| `PAR018` | Replaces `let rec` with `letrec` |
| `PAR_HASH_COMMENT` | Replaces `# comment` with `-- comment` |
| `IMP010` / undefined stdlib | Rewrites renamed stdlib imports, drops prelude builtins, or inserts unambiguous missing stdlib imports |

```bash
# Apply mechanical fixes in place, then type-check
ailang check --fix path/to/file.ail
```

## Installation

`ailang lsp` is built into the `ailang` binary (`which ailang && ailang lsp -h`).

### Claude Code

```bash
/plugin marketplace add sunholo-data/ailang_bootstrap
/plugin install ailang-lsp@ailang-marketplace
```

### VS Code

```bash
ailang editor install vscode
```

Reload VS Code (`Cmd+Shift+P` -> **Developer: Reload Window**). Open any `.ail` file to activate syntax highlighting, diagnostics, hover, go-to-definition, and `QuickFix` actions.

### Other Editors (Neovim, Emacs, Helix)

Configure your LSP client to spawn `ailang lsp --stdio` for `*.ail` with language ID `ailang`. For generic VS Code LSP clients (`zsol.vscode-glspc`):

```json
{
  "files.associations": { "*.ail": "ailang" },
  "glspc.server.command": "ailang",
  "glspc.server.commandArguments": ["lsp", "--stdio"],
  "glspc.server.languageId": ["ailang"]
}
```

## Workspace Behaviour (`MOD010`)

The LSP runs with `RelaxModules` enabled because editors send absolute file URIs without a guaranteed repo root. Run `ailang check <file>` from the CLI for strict `MOD010` module-path enforcement in CI.

## Implementation Reference

- Server & code actions: [`internal/lsp/`](https://github.com/sunholo-data/ailang/tree/dev/internal/lsp) and [`internal/quickfix/`](https://github.com/sunholo-data/ailang/tree/dev/internal/quickfix)
- CLI entrypoints: [`cmd/ailang/lsp.go`](https://github.com/sunholo-data/ailang/blob/dev/cmd/ailang/lsp.go) and [`cmd/ailang/check.go`](https://github.com/sunholo-data/ailang/blob/dev/cmd/ailang/check.go)
- Token-cost harness: [`bench/lsp_token_cost/`](https://github.com/sunholo-data/ailang/tree/dev/bench/lsp_token_cost)
