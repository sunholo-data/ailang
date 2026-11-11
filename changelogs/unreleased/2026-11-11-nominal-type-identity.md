### Fixed — Types are identified by name and module; two `T`s from different modules no longer unify (soundness, BREAKING) (2026-11-11)

`check` accepted programs that passed one module's `type T` where another module's `type T` was
expected, and they then failed at run time (`_str_upper: arg 0 - expected string, got int`, or
`no pattern matched`). A nominal type is now its defining module plus its name
(`types.TCon.Module`); mixing two of them is **`TC_NOMINAL_001`**, which names both modules
(`cannot unify T (defined in q/a) with T (defined in q/b)`). There is no wildcard: every place
that mints a type constructor stamps its module — annotations through one type-name resolver
(local → explicit import → prelude → direct dependency → import closure → builtin home table),
constructor patterns and factories, builtin signatures (`Option` → `std/option`, `Json` →
`std/json`, … via a completeness-tested home table), interface constructor fields and alias
bodies, and every cache decode path. Also fixed on the way:

- a **local** `type T` no longer captures an imported `T` (row a″), and two `Color = Red | Green`
  from different modules no longer cross (a‴);
- a single-constructor ADT wrapping a record (`type W = W({x: int})`) keeps its identity: it is
  expanded to its record only for field access, never to unify with another `W`;
- constructor factories are named per module (`make_T_A@q/a`), so two modules that each declare
  `type T = A(...)` no longer overwrite each other's constructor type, and the evaluator no longer
  fails a correct program with `ambiguous constructor T.A found in multiple modules`;
- `import M (T as U)` of an ADT now denotes M's `T` (it was an opaque `U` that unified with nothing);
- `deriving (Eq)` instances are keyed by module, so one module's `Row` cannot borrow another's;
- the alias environment is keyed by defining module, which retires `TC_TYPE_SHADOW_001`: an
  imported alias body can no longer be captured by a local type of the same name.

Interface hashes (`interface_hash`) are unchanged — a type still prints as its bare name — so no
package needs a republish. The internal interface digest includes the qualified names, and the
compile-cache key moves `v10` → `v11` (every entry misses once).

**Migration:** stop passing one module's `T` where another module's `T` is expected — import the
type (and constructors) you mean by name from one module, or convert explicitly. Programs that use
same-named types from different modules separately need no change. Ambiguity and unbound-name
diagnostics (`TC_AMBIGUOUS_TYPE_001`, `TC_UNBOUND_TYPE_001`), duplicate declarations (`MOD007`) and
the bytecode/SMT/REPL bare-name tables are later milestones. Design:
`design_docs/planned/v1_0_0/m-v1-nominal-type-identity.md`; error page
`docs/docs/reference/errors/tc_nominal_001.md`.
