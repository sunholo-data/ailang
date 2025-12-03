# Editor Setup

Get syntax highlighting for AILANG in your favorite editor.

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
