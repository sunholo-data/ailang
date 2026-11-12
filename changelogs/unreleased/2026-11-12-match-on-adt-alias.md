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
written as its alias Hue)`.

The back ends dispatched on the scrutinee's written type too. The VM failed to compile such a
function ("unknown ADT Hue in switch", so it ran on the evaluator through the bridge, and
`--strict-bytecode` refused it) and `compile --emit-go` emitted `HueKindRed`, which does not exist. Both
now take the ADT from the constructor the elaborator resolved (`internal/gen/lower/match.go`,
`internal/gen/golang/codegen_match.go`), falling back to the scrutinee's type only for a constructor
pattern that carries no resolved ADT. The evaluator, `--bytecode` and `--strict-bytecode` agree, and so
does the emitted Go except where two pre-existing emitter bugs apply, aliases or not: a nullary
constructor nested in a constructor pattern (`Some(Red)`) is not tested, and a typed wrapper whose
signature mentions an ADT alias comes out as `**Color`.

Nothing previously accepted is rejected. No compile-cache key change: the checker only accepts more,
and lowering is not cached.
