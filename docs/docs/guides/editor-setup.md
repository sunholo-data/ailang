# Editor Setup

Get syntax highlighting + language-server intelligence (diagnostics, hover types, go-to-definition, find-references, document outline) for AILANG in your favorite editor.

> **VS Code users**: `ailang editor install vscode` now ships **both** syntax highlighting AND the AILANG Language Server in one command. See [LSP guide](./lsp.md) for what the LSP gives you and how it works under the hood.

## Quick Install (Recommended)

The easiest way to install editor support is via the AILANG CLI:

```bash
# Install for VS Code
ailang editor install vscode

# Install for Vim
ailang editor install vim

# Install for Neovim
ailang editor install neovim

# Check installation status
ailang editor status
```

The CLI embeds all editor files, so it works from anywhere after `ailang` is installed.

## VS Code

The extension shipped via `ailang editor install vscode` provides:

| Capability | What you see |
|---|---|
| Syntax highlighting + bracket matching | Colored keywords, types, strings, etc. |
| **Diagnostics** | Red squigglies on type errors after save (powered by `ailang lsp`) |
| **Hover types** | Type signature popups for top-level identifiers |
| **Go to definition** | Cmd+click on a function name jumps to its decl |
| **Find references** | Right-click → "Find All References" |
| **Document outline** | Functions + ADT constructors in the outline sidebar |

### CLI Install

```bash
ailang editor install vscode
```

Then restart VS Code or run **Developer: Reload Window** (Cmd+Shift+P on Mac, Ctrl+Shift+P on Windows/Linux).

### Manual Install

1. Copy the extension folder to your VS Code extensions:
   ```bash
   # macOS/Linux
   cp -r .vscode/extensions/ailang ~/.vscode/extensions/

   # Windows (PowerShell)
   Copy-Item -Recurse .vscode\extensions\ailang $env:USERPROFILE\.vscode\extensions\
   ```

2. Restart VS Code

### Troubleshooting

**Colors not showing?**
1. Check file has `.ail` extension
2. Check bottom-right of VS Code shows "AILANG" as language
3. Try "Developer: Reload Window" (Cmd+Shift+P)
4. Check for conflicting extensions

**Diagnostics / hover / go-to-def not working?**
1. Check `which ailang` works — the LSP needs the binary on PATH
2. Open VS Code's Output panel: View → Output → pick **"AILANG Language Server"** from the dropdown — any LSP startup errors surface here
3. Verify the LSP process is alive: `ps aux | grep "ailang lsp"` should show one process per open `.ail` workspace
4. See the [LSP guide](./lsp.md) for deeper troubleshooting (workspace-setup, the MOD010 relax behaviour, the token-cost rationale)

**Extension not loading?**
```bash
# Verify extension is installed
ls ~/.vscode/extensions/ | grep ailang

# Check VS Code output for errors
# View > Output > Extension Host
```

## Vim

### CLI Install

```bash
ailang editor install vim
```

### Manual Install

```bash
cp -r editors/vim/* ~/.vim/
```

Files with `.ail` extension will automatically use AILANG syntax highlighting.

## Neovim

### CLI Install

```bash
ailang editor install neovim
```

### Manual Install

```bash
# macOS/Linux
cp -r editors/vim/* ~/.config/nvim/

# Windows
Copy-Item -Recurse editors\vim\* $env:LOCALAPPDATA\nvim\
```

## Sublime Text

Copy `syntaxes/ailang.tmLanguage.json` to your packages directory:

| Platform | Path |
|----------|------|
| macOS | `~/Library/Application Support/Sublime Text/Packages/User/` |
| Linux | `~/.config/sublime-text/Packages/User/` |
| Windows | `%APPDATA%\Sublime Text\Packages\User\` |

## TextMate

Double-click `syntaxes/ailang.tmLanguage.json` to install.

## Other Editors

Most modern editors that support TextMate grammars can use the `syntaxes/ailang.tmLanguage.json` file. Consult your editor's documentation for adding custom language support.

## Syntax Highlighting Features

The grammar supports all AILANG language features:

| Feature | Example |
|---------|---------|
| Comments | `-- comment` or `// comment` |
| Keywords | `let`, `letrec`, `func`, `match`, `if`, `then`, `else`, `type` |
| Booleans | `true`, `false` |
| Primitive types | `int`, `float`, `bool`, `string`, `char`, `unit` |
| Effect types | `IO`, `FS`, `Net`, `Clock`, `Env`, `Rand`, `Debug`, `AI` |
| Standard types | `Option`, `Result`, `List`, `Tuple`, `Array` |
| Strings | `"hello"`, `'c'` |
| Numbers | `42`, `3.14`, `0xFF` |
| Lambdas | `\x. x + 1` |
| Operators | `->`, `=>`, `::`, `++`, `!`, `\|` |
| ADTs | `type Option[a] = Some(a) \| None` |
| Modules | `module examples/hello` |
| Imports | `import std/io (println)` |
| Exports | `export func main()` |

## Managing Installations

```bash
# Check what's installed
ailang editor status

# Uninstall from an editor
ailang editor uninstall vscode
ailang editor uninstall vim
ailang editor uninstall neovim
```

## Updating

When you update AILANG, the embedded editor files are updated too. Simply re-run:

```bash
ailang editor install vscode
```

This will replace the old extension with the new version.

## Recommended: Ligature Fonts

AILANG uses operators like `=>`, `->`, `::`, and `\` (lambda). Programming fonts with **ligatures** render these as beautiful single glyphs:

| Characters | Ligature | AILANG Usage |
|------------|----------|--------------|
| `=>` | ⇒ | Match arms |
| `->` | → | Function types |
| `::` | ∷ | Cons, type annotations |
| `++` | ⧺ | Append |
| `\` | λ | Lambda |
| `!=` | ≠ | Not equal |
| `<=` `>=` | ≤ ≥ | Comparisons |

### Recommended Fonts

| Font | Install | Notes |
|------|---------|-------|
| [Fira Code](https://github.com/tonsky/FiraCode) | `brew install --cask font-fira-code` | Most popular |
| [JetBrains Mono](https://www.jetbrains.com/lp/mono/) | `brew install --cask font-jetbrains-mono` | Excellent readability |
| [Cascadia Code](https://github.com/microsoft/cascadia-code) | `brew install --cask font-cascadia-code` | Microsoft's font |

### VS Code Settings

Add to your `settings.json`:

```json
{
  "editor.fontFamily": "Fira Code, Menlo, Monaco, monospace",
  "editor.fontLigatures": true
}
```

### Vim/Neovim

Set in your terminal emulator (iTerm2, Alacritty, etc.) - Vim uses the terminal's font.
