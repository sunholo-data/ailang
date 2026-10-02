### Changed — an imported name that the module also defines is compile error MOD015 (#1467)

- **Deliberate breaking change.** A module that imports a name explicitly and also defines it at
  module level no longer compiles. Explicit means `import M (tick)`, `import M (f as tick)`, or the
  selective list of `import M as L (tick)`. The definition can be a `func`, a `let`/`letrec`, or an ADT
  constructor (an imported constructor against one of the module's own). Before, the program compiled
  and **the import silently won**: every bare `tick` ran the imported function and the module's own
  definition was ignored. A program that relied on that now fails with MOD015 instead of changing
  meaning silently. This follows Haskell and Elm ("ambiguous occurrence") and Rust (E0255). It replaces
  the "local wins + MOD015 warning" design in m-elaborator-lexical-scope (ruling 2026-10-02).
- The error names both sites and the fix:
  `Error MOD015: 'tick' is both imported (import M (tick) at q.ail:2:1) and defined in this module as a
  func (at q.ail:6:6) — an imported name and a module-level definition may not share a name.`
  `Fix: rename the local definition, or alias the import: import M (tick as mTick)`. It is reported
  the same way by `ailang check` (and `--json`, located at the local definition), `run`, `test`, the
  REPL module loader, and LSP diagnostics. It is checked once, in `ElaborateFile`.
- **Unchanged:** lexical binders (parameters, `let`, match binders) still shadow imports. A bare
  module alias (`import M as L`) binds no bare names, and AILANG has no wildcard import. A local
  `type T` still wins over an imported type `T`. Two imports binding one bare name to different exports
  (`import std/list (length)` + `import std/string (length)`) are **not** covered yet. The later import
  still wins there, which needs its own ruling.
- Corpus: all 729 `.ail` files in the repo were checked. The only collision was `std/ai/streaming.ail`,
  which listed `onEvent, runEventLoop, disconnect` on its `import std/stream as Stream (...)` while
  defining same-named wrappers. The list is dropped; the wrappers already called `Stream.onEvent` etc.
- Error reference: `docs/docs/reference/errors/mod015.md`; modules guide section "Imports, Local
  Definitions and Shadowing".
