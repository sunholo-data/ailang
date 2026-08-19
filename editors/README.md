# AILANG Editor Support

Syntax highlighting for AILANG (.ail files) in various editors.

## Quick Install via CLI (Recommended)

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

## VS Code Manual Install

**Do not copy a folder into `~/.vscode/extensions/`.** Since VS Code 1.74 that
directory's `extensions.json` is the registry of installed extensions, not a
cache — a folder with no entry in it is never scanned, so nothing loads.

Let VS Code install it:

```bash
# Command palette: "Extensions: Install from VSIX..." — or, on the CLI:
code --install-extension <path-to>.vsix --force
```

`ailang editor install vscode` builds that VSIX from the binary's embedded
assets and hands it to whichever editor CLI it finds (`code`, `code-insiders`,
`cursor`, `windsurf`, `codium`); with none on PATH it installs the directory
*and* writes the `extensions.json` entry itself.

Then restart VS Code.

### Workspace Settings (Alternative)

If you only want highlighting when working in the AILANG project, the grammar is automatically available when you open this folder in VS Code (no installation needed).

### Troubleshooting

**Colors not showing?**
1. Check file has `.ail` extension
2. Check bottom-right of VS Code shows "AILANG" as language
3. Try "Developer: Reload Window" (Cmd+Shift+P)
4. Check no conflicting extensions

**Extension not loading?**
```bash
# Checks the registry, not just whether a folder exists on disk
ailang editor status

# Or ask VS Code directly:
code --list-extensions --show-versions | grep ailang

# Check VS Code output for errors
# View > Output > Extension Host
```

Listing `~/.vscode/extensions/` is not a check: it shows folders VS Code has
marked obsolete and skips.

## Vim/Neovim Manual Install

```bash
# For Vim
cp -r editors/vim/* ~/.vim/

# For Neovim
cp -r editors/vim/* ~/.config/nvim/
```

## Sublime Text

1. Copy `syntaxes/ailang.tmLanguage.json` to your packages:
   - macOS: `~/Library/Application Support/Sublime Text/Packages/User/`
   - Linux: `~/.config/sublime-text/Packages/User/`
   - Windows: `%APPDATA%\Sublime Text\Packages\User\`

## TextMate

Double-click `syntaxes/ailang.tmLanguage.json` to install.

## Features

The syntax highlighting supports:

| Feature | Example |
|---------|---------|
| Comments | `-- comment` or `// comment` |
| Keywords | `let`, `func`, `match`, `if`, `then`, `else`, `type` |
| Booleans | `true`, `false` |
| Types | `int`, `float`, `bool`, `string`, `Option`, `Result` |
| Effects | `IO`, `FS`, `Net`, `Clock`, `Env` |
| Strings | `"hello"`, `'c'` |
| Numbers | `42`, `3.14`, `0xFF` |
| Lambdas | `\x. x + 1` |
| Operators | `->`, `=>`, `::`, `++`, `!` |
| ADTs | `type Option[a] = Some(a) \| None` |
