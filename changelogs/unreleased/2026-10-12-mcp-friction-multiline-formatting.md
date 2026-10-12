### Changed

- `ailang fmt` preserves verified source multiline string layout, including interpolation, match patterns, and interior escapes. Escaped one-line strings retain canonical escaping; unverified source attachments use the AST payload. Formatting remains idempotent. Refs #1618.
