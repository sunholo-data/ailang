### Fixed — Breaking (soundness): effectful callbacks nested in data arguments are charged; concrete-annotation tails no longer escape (#1718, #1826) (2026-10-26)

**#1718 (unsound).** A function value inside a list, tuple, `Option`/ADT payload or record
argument was paid by nobody: the callee body erased the callback's implicit effect tail as
"the caller's to pay", but the call site charged only *top-level* arrow parameters. A `pure`
function could therefore perform IO and `ailang check` reported it clean:

```ailang
func applyHead(fs: [(int) -> int], x: int) -> int = match fs { [] => x, [f, ..._] => f(x) }
pure func sneaky(x: int) -> int = applyHead([logIt], x)   -- now: Missing effects: IO
```

The same held for `applyFst((logIt, x))`, `applyOpt(Some(logIt), x)`, record parameters,
record aliases across modules, let-bound tuples, returned lists and lists of tuples. All are
now rejected before execution on both engines.

**#1826 (false rejection / crash).** A concrete annotation such as
`(string) -> bool ! {IO}` keeps an L1-open compatibility tail. Only the direct-callee read
closed it, so the tail escaped into `mapE`'s row: `mapE(\d. g(d), xs)` inside an `! {IO}`
function failed with `Unresolved effect tail`, and the record-field form
(`h.act(d)`) tripped `internal invariant: missing or malformed ApplicationEffects`.
This broke `sunholo/mcp_oauth` 0.1.0/0.1.1 (`revokeFamily`); both check clean again with no
pin bump. Annotated generic containers (`func applyOpt(o: Option[(int) -> int], …)`,
`Holder[(int) -> int]`) are valid definitions again instead of failing with the same tail
error.

**How.** One data-position walker (`internal/types/typechecker_latent_data.go`) now feeds the
call-site mask, the per-argument callback rows, the callee-body implicit-ownership set and the
App constraint row, so every tail a callee erases is charged at every call site (pinned by an
exhaustive erase ⊆ charge test). A concrete contract's tail is closed at every read of a
function value's row. No syntax changes. Design:
`design_docs/implemented/v0_56_0/m-effect-latent-nested-callbacks.md`.

**Intended rejections (migration).** Corpus sweep: 0 status flips over the 515
`examples/**` + `std/**` files, the 45 `ailang-packages` roots and `ailang-parse`. Code outside
those corpora can hit:

- A pure (or narrower-row) caller handing an effectful function inside a list, tuple,
  `Option`, ADT or record to an unannotated callback parameter → `Missing effects: <E>`.
  Declare the effect on the caller (the Suggested fix prints it).
- Store-only nested hand-off, e.g. `keepAll([logIt])` from `pure` code → `Missing effects: IO`,
  matching the existing top-level rule (`keep(logIt)`). Declare the effect, or annotate the
  stored arrow concretely (`[(int) -> int ! {IO}]`) to make it the callee's contract.
- A non-generic ADT whose constructor payload nests an unannotated arrow, built in pure code
  (`type Reg = Reg([(int) -> int])`; `pure func mkReg() -> Reg = Reg([logIt])`) → charged at
  construction, as `Box(logIt)` already was.
- `map(\d. g(d), xs)` with a concrete `g: (string) -> bool ! {IO}` and closed-pure
  `std/list.map` was accepted only through a row-unifier label hole; it is now the same type
  error (`incompatible closed rows`) as `map(\d. println(d), xs)`. **Use `mapE`.**
- Two distinct declared effect-row variables nested in one argument
  (`applyBoth((f, g), x)` with `f: … ! {e1}`, `g: … ! {e2}`) checked clean on base while the
  second row's effects ran unpaid; it now fails loudly with the explicit callback-row
  invariant instead of picking one row.

**Review repair (2026-10-29).** Callback discovery now tracks active type nodes,
so cyclic data/aliases terminate without dropping a reachable callback, and
finite nesting beyond sixteen containers is fully charged. Pathological nesting
or an indefinitely expanding generic alias beyond the defensive traversal limit
returns a normal type error instead of publishing incomplete effects. Regression
controls cover depths 16, 17, 32 and 256, recursive aliases, an expanding alias,
and rejection before IO on both execution engines.

Teaching prompt v0.16.9 adds the nested-callback sentence on top of v0.16.8.
Both prior prompt versions remain byte-for-byte unchanged; embedded, docs and
llms mirrors use the new version.
