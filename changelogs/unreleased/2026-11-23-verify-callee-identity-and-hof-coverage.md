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
