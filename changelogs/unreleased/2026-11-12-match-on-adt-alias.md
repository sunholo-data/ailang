### Fixed — a value whose type is an alias of an ADT can be matched (#2244) (2026-11-12)

`type Hue = Color` names Color; it is not a type of its own. `check` nevertheless rejected
`match h { Red => …, Green => …, Blue => … }` for `h: Hue` with "constructor 'Red' belongs to ADT
'Color', not 'Hue'", and the same for an alias of a prelude type (`type MaybeInt = Option[int]` with
`Some`/`None` arms). The early foreign-constructor check compared the scrutinee's type name before alias
expansion; unification, which expands aliases, never got the chance.

The question "which ADT does this value belong to?" now has one answer, computed in one place after
alias expansion (`internal/types/nominal_head.go`): the constructor-ownership check, the by-name
constructor scope check, exhaustiveness and the choice between two imported constructors of one name
all consult it. Alias chains (`type Tint = Hue`), parameterised aliases (`type IntOpt = Option[int]`,
`type Opt[a] = Option[a]`), aliases exported from another module, aliases nested in a constructor
argument (`Opt[Hue]`), and lambdas with or without an annotation all match. An alias still creates no
new type: two aliases of one ADT stay interchangeable, and a newtype ADT (`type W = W({ x: int })`)
is never expanded, so an alias of a newtype is that newtype.

Messages name both: a missing arm reads `non-exhaustive match on Hue (= Color) — no arm matches: Blue`,
and a constructor of another type reads `… belongs to ADT 'Option', not 'Color' (the scrutinee's type,
written as its alias Hue)`. A non-exhaustive match on an alias of a non-ADT type now names the
expansion the same way (`non-exhaustive match on Age (= int) — no arm matches: _ (any int other than
0, 1)`, `on P (= { x: int })`): the missing patterns are always described in terms of the expansion, so
the message says what the alias stands for.

The back ends dispatched on the scrutinee's written type too. The VM failed to compile such a
function ("unknown ADT Hue in switch", so it ran on the evaluator through the bridge, and
`--strict-bytecode` refused it), and `compile --emit-go` emitted `HueKindRed`, which does not exist.
They now dispatch on each constructor pattern's own ADT: the elaborator's, or — for a constructor
matched by name (#323), whose type the module did not import — the one the type checker resolved
from the type it matched it against, which the match check now records on the pattern. The back ends
fall back to the scrutinee's type only for a pattern whose type nothing fixed. The VM's if-chain tests
and the Go emitter's if-chains (`internal/gen/lower`, `internal/gen/golang/codegen_match*.go`) use the
same ADT.

### Fixed — the VM tested a nested constructor against the wrong type (2026-11-12)

No alias needed: a constructor pattern nested in another (`Some(Red)`) reached the VM with no type, and
the VM read its tag from the first loaded type declaring a constructor of that name. With `type Light =
Amber | Red | Green` declared before `type Color = Red | Green | Blue`, `match o { Some(Red) => 10,
Some(Green) => 20, … }` on an `Option[Color]` missed every arm on `--strict-bytecode` (RT002, where the
evaluator printed 10, 20, 30), and with two modules' types sharing a constructor the VM silently took
another arm (`[31, 0]` where the evaluator printed `[32, 0]`). Each constructor pattern now carries its
own ADT to the VM, at any depth. The same applies to a by-name constructor in an if-chain (a guarded or
repeated arm): the VM printed `[2, 1, 3]` where the evaluator printed `[1, 2, 3]`. The Go emitter's
if-chain picked among same-named constructors by map order; it now uses the pattern's ADT too.

The evaluator, `--bytecode`, `--strict-bytecode` and the emitted Go agree on all of the above, except
where two pre-existing Go emitter bugs apply, aliases or not: a nullary constructor nested in a
constructor pattern (`Some(Red)`) is not tested by the emitted Go, and a typed wrapper whose signature
mentions an ADT alias comes out as `**Color`.

Nothing previously accepted is rejected. No compile-cache key change: the checker only accepts more;
the recorded ADTs live in the cached Core, and a cache entry is only ever served to the build that wrote
it (the compiler identity is in the key); a warm and a cold compile agree.
