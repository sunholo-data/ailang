### Fixed — local binders no longer captured by imports, builtins or constructors of the same name (2026-10-02)

The elaborator resolved every identifier against the constructor table and the global environment
(every import and every builtin) before considering local binders, so a lambda parameter named like
an imported function (`\tick. {tick: tick}` next to `import ./a (tick)`) resolved to the import:
a false type error ("cannot unify type constructor int with *types.TFunc2"), or, when the types
happened to line up, a program that silently ran the import. Lambda and function parameters,
`let`, `letrec`, block statement-lets, match-arm binders (in guards and bodies) and `forall`
variables now shadow imports, builtins (`let show = 3 in show`), constructors and module aliases
(`let L = {map: 42} in L.map`) within their scope. Pattern position is unchanged (constructor-first,
#323). A module-level function that shares a name with an import still loses to the import; that
change is awaiting a design decision. Fixes #1467; design:
`design_docs/implemented/v0_51_1/m-elaborator-lexical-scope.md`.
