### Fixed — Types are identified by name and module; two `T`s from different modules no longer unify (soundness, BREAKING) (2026-11-11)

`check` accepted programs that passed one module's `type T` where another module's `type T` was
expected, and they then failed at run time (`_str_upper: arg 0 - expected string, got int`, or
`no pattern matched`). A nominal type is now its defining module plus its name
(`types.TCon.Module`); mixing two of them is **`TC_NOMINAL_001`**, which names both modules
(`cannot unify T (defined in q/a) with T (defined in q/b)`; a built-in is named "the built-in
Map"). There is no wildcard: every place that mints a type constructor stamps its module —
annotations through one type-name resolver, constructor patterns and factories, builtin
signatures (`Option` → `std/option`, `Json` → `std/json`, … via a home table that registration
enforces), interface constructor fields and alias bodies, and every cache decode path.

A bare type name resolves in this order: the module's own declaration → an explicit
`import M (T [as U])` → a built-in (`Map`, `List`, `Int`, …) → the prelude (`Option`, `Result`)
→ the one direct dependency that exports it → the one module of the import closure that declares
it → the builtin home table → an opaque type of this module. A module may therefore declare its
own `Map` or `List`. Nothing depends on import order:

- two explicit imports binding one bare type or constructor name to different modules
  (`import a (T)` + `import b (T)`) are **`MOD015`** (it used to be "first import wins");
- a type name two direct dependencies (or, failing those, two modules of the import closure)
  declare is **`TC_AMBIGUOUS_TYPE_001`** unless declared locally or imported by name;
- a constructor auto-imported from two modules is resolved in a pattern by the scrutinee's
  type, and is **`TC_AMBIGUOUS_CTOR_001`** where nothing decides it (an expression, or a pattern
  on a value of unknown type).

Also fixed on the way:

- a **local** `type T` no longer captures an imported `T` (row a″), and two `Color = Red | Green`
  from different modules no longer cross (a‴);
- a single-constructor ADT wrapping a record (`type W = W({x: int})`) is a distinct type from
  every other nominal type, including another module's `W` of the same shape; it still converts
  implicitly to and from its own record for field access (pre-existing, #2233, #2234);
- constructor factories are named per module (`make_T_A@q/a`), so two modules that each declare
  `type T = A(...)` no longer overwrite each other's constructor type, and the evaluator no longer
  fails a correct program with `ambiguous constructor T.A found in multiple modules`;
- `import M (T as U)` of an ADT now denotes M's `T` (it was an opaque `U` that unified with nothing);
- `deriving (Eq)` on an ADT is keyed by module, so one module's `Row` cannot borrow another's
  (records stay structural: a record's `==` is still decided by its declared name);
- the alias environment is keyed by defining module, which retires `TC_TYPE_SHADOW_001`: an
  imported alias body can no longer be captured by a local type of the same name;
- the REPL/WASM module registry keeps one constructor per module (loading a module that declares
  a same-named constructor no longer breaks another module's) and resolves names with the same
  rules.

Interface hashes (`interface_hash`) are unchanged — a type still prints as its bare name — so no
package needs a republish. The internal interface digest includes the qualified names, and the
compile-cache key moves `v10` → `v11` (every entry misses once).

**Migration:** stop passing one module's `T` where another module's `T` is expected — import the
type (and constructors) you mean by name from one module, or convert explicitly
(`import a (T as AT, A as AA)`). Programs that use same-named types from different modules
separately need no change. For `MOD015`/`TC_AMBIGUOUS_*`, import the one you mean by name or alias
one import with `as`. Not covered yet: undeclared type names (`TC_UNBOUND_TYPE_001`), duplicate
declarations (`MOD007`), and three back-end tables keyed by bare type name — bytecode ADT tables,
the SMT encoder and the Go emitter. Design: `design_docs/planned/v1_0_0/m-v1-nominal-type-identity.md`;
error page `docs/docs/reference/errors/tc_nominal_001.md`.
