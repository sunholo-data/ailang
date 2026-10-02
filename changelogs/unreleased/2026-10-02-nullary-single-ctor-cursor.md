### Fixed — `type J = | Idle` hid the next declaration's `export`/`pure` (2026-10-02)

A one-constructor ADT written with a leading pipe and no `deriving` clause (`export type J = | Idle`)
advanced the parser cursor past its own last token, so the following declaration lost its first
token: `export type W` parsed as non-exported (importers got `IMP010: symbol 'W' not exported`),
`pure func` silently parsed as non-pure, and `func` produced cascading parse errors. The nullary
leading-pipe branch of `parseTypeDeclBody` now advances only when another `|` variant follows,
matching the with-fields branch. `type J = Idle` (no pipe) is still a type alias. Fixes #1466;
design: `design_docs/implemented/v0_51_1/m-parser-nullary-single-ctor-cursor.md`.
