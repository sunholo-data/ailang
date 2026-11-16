### Fixed — a by-name constructor one loaded type declares is decided at once (#2362 review)

After #2362 a constructor matched by name (#323) was always a deferred choice, which keeps a
let-bound helper monomorphic until a use decides it. With a single candidate type that was
needless: `let isSome = \o. match o { Some(_) => true, _ => false }` used at `Option[int]` and
`Option[string]` (in a module that does not import `std/option`) was `TC_AMBIGUOUS_CTOR_001` —
"which one this pattern means is not yet decided" — although only Option declares `Some`; it ran
on every engine before #2362. A by-name constructor that exactly one loaded type declares is now
typed as that type's constructor at once, at fresh type arguments, so helpers over it generalise
(with a class constraint such as `Some(x) => show(x)` too, also as an exported top-level
function). Only a constructor two or more loaded types declare is still a deferred choice.

### Fixed — a `let` in a contract clause or test body is checked like a `let` in a function body

A contract clause (and a test body) was inferred as if it were a module declaration, so a `let`
at its root was generalised as the declaration's own binding: `requires { let g = \q. match q {
Ok(_) => true, _ => false } in g(r) }` left the by-name `Ok` undecided (`TC_MATCH_007`, "nothing
fixes the type") although `r: Result[int, string]` is annotated. Contract clauses and test bodies
are now inferred as bodies (`CoreTypeChecker.InferBody`): their lets follow the same rules as a
let inside a function body.

### Fixed — `TC_MATCH_007` on a value that is not built from constructors

A constructor matched by name against a record, tuple, list, function, unit or primitive value
(or an alias of one) said "nothing fixes the type of the value matched" and suggested an import.
It now says the value's type — `{ a: int }`, `[int]`, `P (= { a: int })` — "is not built from
constructors here, so this arm can never match", with "remove the arm".
