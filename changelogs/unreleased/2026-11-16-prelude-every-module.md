### Changed — the Option/Result prelude is in scope in every module, not only entry modules (#2371)

`Option`, `Some`, `None`, `Result`, `Ok` and `Err` resolve without an import in every module
outside `std/` — library modules too, not only modules with an exported `main`. The same source
used to mean different things depending on whether the file had a `main`: in a library module
`Some` was matched by name (#323), so a helper over it could not be used at `Option[int]` and
`Option[string]` (`TC_AMBIGUOUS_CTOR_001`), and building `Some(x)` needed
`import std/option (Some)`.

The rules are the entry-module rules, unchanged (`internal/loader/prelude_imports.go`):

- lowest precedence: an import from `std/option`/`std/result` replaces that implicit import
  (`import std/option (Option, Some, None)` remains valid and changes nothing), and an explicit
  import of another type's `Some` wins for that name, with `Option`/`None` still the prelude's;
- a module declaring a prelude type or constructor itself does not get that prelude module;
- the type names `Option` and `Result` are in scope, as in entry modules;
- `println` (the other prelude binding) stays entry-only.

Not changed: `std/*` modules, and a file of top-level expressions — with any import such a file's
expression is not type-checked (#2372), so it gets no implicit import, as before. A library
module now depends on `std/option` and `std/result`, so its compile-cache key includes their
interfaces, and a project's library modules need a usable stdlib (a bogus
`AILANG_STDLIB_PATH` now fails for them as it already did for entry modules).
