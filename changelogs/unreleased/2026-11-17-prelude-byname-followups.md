### Changed — BREAKING: a library's own Ok/Err/None/Some matched by name is now the prelude's (#2371)

Since #2371 `Option`/`Some`/`None`/`Result`/`Ok`/`Err` are in scope in every module outside
`std/`. A library module that matched, by name and without importing its type, a constructor of a
user type called `Ok`, `Err`, `None` or `Some` now gets the prelude's constructor, as entry modules
always did. With `type Status = Ok | Err(string) | Pending` two imports away, `match ws(i) { Ok =>
1, Err(_) => 2, _ => 3 }` printed `[1, 2, 3]` and is now "constructor Ok in pattern expects 1
argument(s) — write Ok(...)"; with `type Maybe = Just(int) | None`, `match wm(i) { Just(v) => v,
None => 0 }` is now "match arm constructor 'None' belongs to ADT 'Option', not 'Maybe'". Fix: import
the type and its constructors — `import lib/st (Status, Err, Ok, Pending)` — which the error now
names (below). #2371's sweep found no program that passed before and fails now in the repo
(989), banked (14,817), ailang-packages (622) or motoko-fork (575) corpora.

### Added — both errors name the import that fixes them

When a type in the module's import closure declares a constructor of the same name (with no
arguments, for the arity error; the scrutinee's own type, for the foreign-constructor error), the
error ends "if you mean lib/st's Status.Ok, import it: import lib/st (Status, Err, Ok, Pending)".
The import lists the type's constructors: importing the type alone does not bring them into scope,
so the prelude's would still win. Applying it verbatim makes the program check and run unchanged
on every engine.

### Added — "undefined variable: Some/None/Ok/Err" names its import

Where the prelude is not injected — `std/` modules and a file of top-level expressions (#2372) —
an expression using `Some`, `None`, `Ok`, `Err`, `Option` or `Result` was a bare "undefined
variable"; it now ends "— add `import std/option (Option, Some, None)`" (resp. `std/result`), as
`getOrElse` already did.
