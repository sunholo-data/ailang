### Fixed

- Named tests and properties now use the lexer's string-aware structural brace scan for extraction. Unbalanced braces in strings and comments no longer corrupt sibling tests. Refs #1618.
