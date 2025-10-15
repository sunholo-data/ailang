# AILANG Syntax Highlighting

## VS Code

### Option 1: Local Extension
1. Copy the `.vscode/extensions/ailang/` folder to your VS Code extensions directory:
   - Windows: `%USERPROFILE%\.vscode\extensions\`
   - macOS/Linux: `~/.vscode/extensions/`
2. Restart VS Code

### Option 2: Workspace Settings
The syntax highlighting will automatically work if you open the project folder in VS Code, as the grammar is included in the project.

## Vim/Neovim

1. Copy the syntax files to your Vim configuration:
   ```bash
   # For Vim
   cp -r editors/vim/* ~/.vim/

   # For Neovim
   cp -r editors/vim/* ~/.config/nvim/
   ```

2. Files with `.ail` extension will automatically use AILANG syntax highlighting.

## Sublime Text

1. Copy `syntaxes/ailang.tmLanguage.json` to your Sublime Text packages directory:
   - Windows: `%APPDATA%\Sublime Text\Packages\User\`
   - macOS: `~/Library/Application Support/Sublime Text/Packages/User/`
   - Linux: `~/.config/sublime-text/Packages/User/`

2. Rename the file to `ailang.sublime-syntax` if needed.

## TextMate

1. Double-click the `syntaxes/ailang.tmLanguage.json` file
2. TextMate will install it automatically

## Other Editors

Most modern editors that support TextMate grammars can use the `syntaxes/ailang.tmLanguage.json` file. Consult your editor's documentation for instructions on adding custom language support.

## Features

The syntax highlighting includes:
- Keywords (let, func, if, match, etc.)
- Built-in types and functions
- Comments (-- single line)
- Strings and escape sequences
- Numbers (integers, floats, hex)
- Quasiquotes (sql""", html""", regex/, etc.)
- Operators and special symbols
- Type names (capitalized identifiers)

## Customization

You can modify the color scheme by editing your editor's theme settings. The grammar uses standard TextMate scopes that work with most color themes.