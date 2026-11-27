---
reviewed: 2026-11-26
reviewBy: 2027-02-26
---

# Editor Setup

Get syntax highlighting and language-server intelligence (diagnostics, quick-fix code actions, hover types, go-to-definition, find-references, document outline) for AILANG.

> **VS Code users**: `ailang editor install vscode` installs **both** syntax highlighting and the `ailang lsp` extension. See the [LSP guide](./lsp.md) for quick-fixes (`textDocument/codeAction`) and typed holes (`_?` / `?name`).

## Quick Install (Recommended)

```bash
ailang editor install vscode
ailang editor install vim
ailang editor install neovim
ailang editor status
```

All editor assets are embedded directly in the `ailang` binary.

## VS Code

| Capability | What you see |
|---|---|
| Syntax highlighting + brackets | Colored keywords, types, strings, lambdas |
| **Diagnostics** | Inline type/effect errors and `TC_HOLE` prompts on save (`ailang lsp`) |
| **Quick fixes** | `codeAction` fixes for `! {Effects}`, `++` -> `"${...}"`, `TC_MATCH_002` stubs |
| **Hover types** | Inferred type signatures on top-level exports |
| **Go to definition & references** | Jump to decls and find references across open files |
| **Document outline** | Functions, ADTs, and constructors in the outline sidebar |

```bash
ailang editor install vscode
# Then Cmd+Shift+P -> Developer: Reload Window
```

:::warning Copying into `~/.vscode/extensions/` does not register the extension
VS Code tracks installed extensions in `~/.vscode/extensions/extensions.json`. `ailang editor install vscode` installs via the editor CLI (`code`, `cursor`, `windsurf`, `codium`) or updates `extensions.json` automatically; manual folder copies are silently ignored.
:::

### Troubleshooting VS Code

```bash
which ailang                                     # Binary must be on PATH
ailang editor status                             # Checks extensions.json registration
code --list-extensions --show-versions | grep ailang
ps aux | grep "ailang lsp"                       # One server process per workspace
```

Check **View -> Output -> AILANG Language Server** for LSP startup errors.

## Vim, Neovim & Other Editors

```bash
# Vim
ailang editor install vim       # or: cp -r editors/vim/* ~/.vim/

# Neovim
ailang editor install neovim    # or: cp -r editors/vim/* ~/.config/nvim/
```

For Sublime Text, TextMate, or other TextMate-compatible editors, install `syntaxes/ailang.tmLanguage.json` into your editor's user packages directory. For LSP integration in Neovim/Emacs/Helix, configure `ailang lsp --stdio` (see [LSP guide](./lsp.md)).

## Managing & Updating Installations

```bash
ailang editor status
ailang editor install vscode    # Run after upgrading ailang to update the embedded extension
ailang editor uninstall vscode
```

## Ligature Fonts (Optional)

Programming fonts with ligatures (Fira Code, JetBrains Mono, Cascadia Code) render `=>`, `->`, `::`, `!=`, `<=`, `>=` cleanly. In VS Code `settings.json`:

```json
{
  "editor.fontFamily": "JetBrains Mono, Fira Code, Menlo, monospace",
  "editor.fontLigatures": true
}
```
