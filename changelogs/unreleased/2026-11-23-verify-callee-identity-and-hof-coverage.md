### Fixed — `ailang verify` could prove a contract about the wrong function (2026-11-23)

Callee bodies were looked up by bare name, the first hit taken while iterating a Go map. When
two loaded modules defined the same name, the verifier could inline the other one. Measured
before the fix:
- A module importing `a/good (clamp0 as goodClamp)` and `a/bad (clamp0)` got VERIFIED in 5 of
  6 runs for `usesBad(x) ensures { result >= 0 } { clamp0(x) }`, although `a/bad.clamp0` is `x - 1`.
- A local `clamp0` shadowing an imported one gave a false VERIFIED on every run.

All bare-name lookups now run in a fixed order: the checked module, then imports sorted by
path. Any call whose bare name would bind a different module's function than the call refers
to is SKIPPED, naming both modules. Measured over 1,840 contracts in five downstream
repositories, this cost no correct verdict (M-VERIFY-SOUNDNESS-COVERAGE M1, S7).

### Fixed — `map`/`filter`/`foldl` callers verify again, over any element type (#1375, #215) (2026-11-23)

The callee-sort gate rejected `std/list.map[a, b]`'s type variables before the HOF inliner ran,
so every literal-lambda HOF caller was SKIPPED (`examples/runnable/contracts/hof_verify.ail`
had silently stopped verifying). Behind the gate, the inliner had four further defects:
- It hard-coded `(Seq Int)` element sorts.
- It dropped `DictApp.Method` and node identity when substituting the lambda parameter.
- It gave every call in one function the same specialisation name.
- It failed silently, leaving the call in the body for the encoder to choke on.

Now:
- `map`, `filter`, `foldl`, `any` and `flatMap` with a literal lambda are specialised over the
  call's real element and result sorts (strings, floats, records).
- Lambdas may capture the enclosing function's parameters and lets.
- HOFs inside a called helper, and inside `requires`/`ensures` (`ensures { any(\r. r.who == who, result) }`), are specialised too.
- Captured variables are bound as `$c_<name>`, so a capture named like a record field (`filter(\p. p.actor == actor, rs)`) does not shadow the accessor.
- A specialisation is a bounded unrolling (depth 3), so it follows the bounded-recursion rules:
  VERIFIED (bounded) when `requires` bounds the list, and INCONCLUSIVE for any model, never a
  violation that may rest on the uninterpreted frontier.
- A HOF call whose types are unknown is refused (SKIPPED), never encoded at a guessed sort.

Local `let` bindings are now bound as `$l_<name>` in the SMT output, so a local named like a
record accessor no longer makes Z3 reject the query as ambiguous (#215). Unsupported-construct
skip reasons no longer repeat their prefix.

### Changed — encoder capability limits are skips, not verifier errors (2026-11-23)

Two encoder limits used to surface as `! ERROR` (exit 1, `verifier_error`): a call with no SMT
form ("unsupported application: …") and a cycle among inlined helpers ("circular function call
detected"). They are now `unsupported_encoding` skips, like the other capability boundaries (#757).

### Added — more of real code reaches a verdict (2026-11-23)

- `map(double, xs)`, `filter(isTable, blocks)`, `foldl(step, 0, xs)`: a named top-level function
  argument is eta-expanded and specialised like a lambda. A function-typed *parameter* is still
  skipped.
- A map/filter specialisation's frontier is now a length-preserving `seq.map` (map) or a list no
  longer than its input (filter). So `length(result) == length(xs)` / `<= length(xs)` are proved
  for lists of any length, not only within the unrolling depth. This is quantifier-free: a
  quantified fact made Z3 time out on false contracts.
- Encoded directly, as full proofs over any element sort:
  - `std/list` `nthOr`/`nth_or`, `headOr`/`head_or`, `lastOr`/`last_or`;
  - the Option-returning `head`/`nth`/`last`;
  - `std/option` `getOrElse`/`isSome`/`isNone`. `Option` is declared on demand.

### Fixed — `floatToInt` was verified as floor, but runs as truncation (2026-11-23)

The verifier encoded `floatToInt` as Z3's `to_int`, which rounds toward −∞, while the runtime
truncates toward zero. So `ensures { result == -3 }` for `floatToInt(-2.5)` was VERIFIED, and the
program returns −2. It is now encoded as truncation toward zero.

### Added — witness replay turns inconclusive bugs into counterexamples (2026-11-23)

A sat on a bounded encoding (recursion, or a map/filter/foldl/any/flatMap specialisation) used to
be reported INCONCLUSIVE, because its model may rest on the uninterpreted frontier. That held even
when the function had a genuine bug. `verify`, `check --verify`, `ai-check` and
`verify --package` now run the real function on the model's parameter values, with runtime
contract checking, through the property-test harness.

If every `requires` holds and an `ensures` is false, the result is a VIOLATION with
`witness_replayed: true` and the replayed inputs (`witness replayed at runtime: ensures clause 1
is false for n = 2`). Otherwise it stays INCONCLUSIVE, with what the replay saw. A runtime
error, an unconvertible value or a timeout decides nothing. Replay can only confirm what the
runtime shows. Inputs of type int, float, bool, string, list and record are replayed; Option
and other ADTs are not yet.

Counterexample models are now read as S-expressions, so `(Seq Int)` sorts and multi-line values
print whole (they printed as `(Seq = Int`).

### Fixed — `take`/`drop` in verified code were always skipped (2026-11-23)

`std/list.take`/`drop` were on the encodable list, but the only encoding was an unrolling
generator that nothing called, so every function using them was skipped ("unsupported
application"). They are now encoded exactly with `seq.extract`, matching the runtime's edge
cases: `take(-1, xs)` is `[]` and `drop(-1, xs)` is `xs`. `reverse` is still not encodable; the
contracts guide no longer lists it.

### Added — `std/math` in verified code (2026-11-23)

Every function calling `std/math` was skipped: the wrappers call `_math_*` builtins with no
encoding. Now:
- `floor`, `ceil`, `round` (half away from zero, as Go's `math.Round`), `absFloat` and `absInt`
  are defined exactly, and `PI()`/`E()` are the exact float64 values.
- `sqrt`, `exp`, `pow`, `log`, `sin` and `cos` are uninterpreted functions. Each call adds ground
  facts that the IEEE result satisfies for every non-NaN input:
  - `x >= 0 ⇒ sqrt(x) >= 0`, and `sqrt` maps `[0,1]` into `[0,1]` and `[1,∞)` into `[1,x]`;
  - `exp(x) >= 0`, and `>= 1` for `x >= 0`;
  - `pow(x, 0) == 1` and `pow(x, 1) == x`;
  - `log(x) >= 0` for `x >= 1`;
  - `-1 <= sin, cos <= 1`.
- Values are never assumed: `sqrt(2.0) * sqrt(2.0) == 2.0` is not proved.
- A helper calling `sqrt` (the stapledons Kerr `horizon` shape) now verifies.
- A model of an encoding that uses an uninterpreted math function is **inconclusive**, not a
  violation. The solver may pick a value the real function never returns (`exp(x) = 0`).
  Witness replay then confirms the genuine ones (`sqrt(2.0) * sqrt(2.0) == 2.0` is a replayed
  violation: it is 2.0000000000000004). Without this, 77 physics contracts in the measured
  corpus would have been reported as false counterexamples.

### Fixed — `[]` under a `let` was typed `(Seq Int)` (2026-11-23)

The sort hint that types an empty list from the declared return type stopped at `let`.
`let room = … in if room <= 0 then [] else take(room, jobs)` therefore built a `(Seq Int)`, and
Z3 rejected the query for any other element type. The hint now flows through `let` bodies.

### Changed — witness replay reads more model shapes (2026-11-23)

Replay now expands Z3's `(let ((a!1 …)) …)` model terms and converts nullary enum constructors
(`status: Failed`), so record-and-enum inputs (job queues, inboxes) are replayed rather than left
inconclusive.
