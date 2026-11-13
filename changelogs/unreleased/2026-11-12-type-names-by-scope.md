### Changed — BREAKING: type variables are told from types by scope, not by case (#2211, #2240) (2026-11-12)

Whether a name in type position is a type variable was decided by the case of its first
letter, so two kinds of declaration were silently ignored:

- **#2211** — a type declared with a lowercase name (`type point = {x: int, y: int}`) was a
  fresh type variable at every use, so `func origin() -> point = {x: "zero", label: true}`
  and `idp(42)` for `idp(p: point)` type-checked and ran. `std/sem`'s five lowercase types
  (`sem_frame`, `search_result`, …) were never enforced either.
- **#2240** — an uppercase name in a type-parameter list was not a type variable:
  `func id[T](x: T) -> T` made `T` an opaque type constructor, so `id(5)` failed with
  `No instance for Num[T]`, and `type Box[T] = Box(T)` could not hold an `int`.

The parser now decides by scope (`internal/parser/parser_type_names.go`):

1. a name an enclosing type-parameter list binds (`func f[T]`, `type Box[T]`) is that type
   variable, whatever its case, and shadows a type of the same name within the declaration.
   It cannot be applied to arguments: `func f[m](x: m[int])` is the new parse error
   `PAR_TYPE_PARAM_APPLIED` (no higher-kinded types; before, `m[int]` silently named an
   opaque type no value could have);
2. a built-in (`int`, `string`, `float`, `bool`, `unit`, `char`, `bytes`) is that type, and
   naming a type parameter after one is the new parse error `PAR_TYPE_PARAM_BUILTIN`;
3. a lowercase name the file declares as a type, anywhere in the file, or imports by name
   (`import std/sem (sem_frame)`) is that type;
4. any other lowercase name is an implicitly quantified type variable, as before
   (`func f(x: a) -> a`, `(acc, elem) -> acc`);
5. any other name is a type, resolved by the module's type scope.

Two readings keep a lowercase name a type variable, as before, and now warn
`TC_UNBOUND_TYPE_002` (in `check --json` warnings, cold and warm): an implicit variable named
like a type that an imported module exports but this module does not import by name (fix:
`import M (name)` if the type was meant), and a name imported by name that is a value, not a
type (`import std/list (map)` … `(map) -> map`). A lowercase name applied to type arguments
that names no type (`zz[int]`) is the error `TC_UNBOUND_TYPE_001` (uppercase undeclared names
are #1485). A type declaration has no implicit variables: a lowercase name in the body of a `type`
declaration (alias, constructor fields, record fields) that is neither one of the
declaration's own `[..]` parameters nor a type in scope is the new error
`TC_UNBOUND_TYPE_003`, at the declaration: `type Count = integer` ("type variable 'integer' is
not a parameter of Count: write `type Count[integer] = …` to make it a parameter, or did you
mean the type `int`?"), `type Any = a`, `type Box = B(elem)`. Before, the name was a free
variable, so the type silently stood for any type (with alias expansion, such programs could
pass `check` and crash or print a wrong answer). The suggestion is a type within edit
distance 2 or differing only by case, or `import M (name)` when an import exports the type
but it is not imported by name. Measured: 0 such declarations in the repo (942 files), the
banked solutions (14,691), ailang-packages, the motoko fork, docparse, ailang-parse,
email-parse, daneel, ailang-world and the local package registry (4,586 files).

A type alias may not refer to itself: `type L[a] = [L[a]]`, `type T2[a] = (T2[a], T2[a])`,
`type W[a] = Option[W[a]]`, or `type A = B` with `type B = A` is the new error
`TC_TYPE_ALIAS_CYCLE_001` at the declaration ("type alias 'T2' refers to itself; a recursive
type must be an ADT: write `type T2[a] = T2((T2[a], T2[a]))` (a constructor) instead"). An alias
stands for its body, so such a cycle has no finite meaning: these were accepted and failed
later with confusing errors (`latent callback type nesting exceeds limit 1000`), or let alias
expansion run without bound. A cycle through an ADT declaration (`type Tree[a] = Leaf |
Node(Tree[a], a, Tree[a])`) is unaffected, and so — deliberately — is one through a record
declaration (`type Node = {label: string, kids: [Node]}`), which type-checks and runs correctly
and which real programs use (2 repo examples, 1 banked solution). Measured: 0 alias-only cycles
in the repo, the banked solutions, ailang-packages, the motoko fork, the other local downstream
checkouts and the package registry.

When a type error occurs in a module whose lowercase declared type is used in a
signature next to implicit variables (`type acc = …` and `(acc, elem) -> acc`), the error
carries a note naming the declaration.

Newly rejected: programs that relied on a lowercase type not being enforced, a type declaration with a free type variable, a type alias that refers to itself, a type parameter
named like a built-in (`type Result[int] = Ok(int) | Err(string)`), a type parameter applied
to arguments, and a program that declared a type `T` and used `[T]` while meaning that type
inside the declaration. Measured on 11,180 banked AI-written solutions that pass `check` on
dev: one is newly rejected (`type Result[int] = …`, `PAR_TYPE_PARAM_BUILTIN`); 0 newly failing
in the downstream corpora.

Newly accepted: `func toInt[T](x: T) -> int = x` was rejected before, for the wrong reason
(`T` was an opaque type); it is now accepted exactly like `func toInt[a](x: a) -> int = x`,
because signature type variables are not rigid (#2273). Call sites are still checked against
the inferred scheme, so this is not unsound at run time, but the annotation constrains
nothing. Residual hole, documented in `docs/LIMITATIONS.md`: an undeclared lowercase name (a
typo, a miscased type, a value name) is still a fresh type variable.
