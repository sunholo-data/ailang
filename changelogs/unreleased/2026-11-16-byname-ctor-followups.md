### Fixed — a by-name constructor's candidates come from the module's own import closure (#2362 review)

The candidates of a constructor matched by name (#323, #2353) — and the table the #1478
foreign-constructor and arity checks read — were collected from every interface the program had
loaded so far, including siblings outside the module's import closure. A library's error text
(and, for an orphan library that imports no declaring module, its error code) depended on the
root's import order and on which root compiled it, which no compile-cache key covers. They now
come from the module's transitive import closure only, the same set the transitive alias pull
already used (`TC_MATCH_001` now says no "module this one imports, directly or not" defines the
constructor, rather than no "loaded module"). A by-name constructor stays a deferred choice even when one type in the closure
declares it: deciding it from the candidates alone would make a library's verdict depend on what
else its closure declares. Known limitation, unchanged since #2362: a let-bound helper matching a
by-name `Some`/`Ok` in a library module that does not import `std/option`/`std/result` is
monomorphic, so using it at `Option[int]` and `Option[string]` is `TC_AMBIGUOUS_CTOR_001` (the
message now says the constructor "is matched by name and not imported" when there is one
candidate); importing the type makes it generic. #2371 proposes the prelude constructors in every
module, which removes the realistic case.

### Fixed — a `let` in a contract clause is checked like a `let` in a function body

A contract clause was inferred as if it were a module declaration, so a `let` at its root was
generalised as the declaration's own binding: `requires { let g = \q. match q { Ok(_) => true,
_ => false } in g(r) }` with a by-name constructor left the choice undecided. Contract clauses are
now inferred as bodies (`CoreTypeChecker.InferBody`): their lets follow the same rules as a let
inside a function body. (Test bodies already did: they are a lambda or an `if`, never a `let`.)

### Changed — the integer-literal restriction applies to a clause-root `let` in contracts

Because a clause-root `let` is now a body let, an integer literal bound by it is no longer
generalised (the declaration-root exemption no longer applies): `requires { let z = 0 in x >= z
&& y >= z }` with `x: int, y: float` was accepted and is now `TC_CONTRACT_001` (`z` is an int, `y`
a float), as the same `let` in a function body already was. Write the literal at each use, or
`0.0` for the float. 0 of 937 real contract programs (repo and banked corpus) are affected.

### Fixed — `TC_MATCH_007` on a value that is not built from constructors

A constructor matched by name against a record, tuple, list, function, unit or primitive value
(or an alias of one) said "nothing fixes the type of the value matched" and suggested an import.
It now says the value's type — `{ a: int }`, `[int]`, `P (= { a: int })` — "is not built from
constructors here, so this arm can never match", with "remove the arm".
