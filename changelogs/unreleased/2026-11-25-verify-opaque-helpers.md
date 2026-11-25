### Added — `ailang verify` calls an unencodable helper opaquely

A helper whose body does not encode used to skip every function that called it. Causes include a tuple, a builtin with no SMT form, and a call cycle with another helper. If the helper has an `ensures`, it is now called through that contract. Otherwise it becomes an **opaque helper**, an uninterpreted function of its signature:

- A proof that holds for every value the helper could return is `✓ VERIFIED`. It carries a `helpers_not_checked` caveat naming each opaque helper and why its body did not encode. The checks inside that body (its callees' `requires`, overflow, NaN) were not made.
- A model that rests on the helper's value is `INCONCLUSIVE`, never a violation. Witness replay then runs the real function.
- The helper's own `requires` is still checked at each call (D1).
- A helper without a return annotation has no sort to be opaque at, so its callers are still skipped.
- Mutually recursive helpers are handled the same way, and a function reached back through its own helpers is called through its own `ensures`.

Downstream gap map: 6 newly verified (the docparse HTML renderers, `htmlParseTable`, the DOCX/PPTX span droppers, `apiKeyValidateResponse`), 2 skipped → inconclusive, no regressions.

### Fixed — verifier errors that those skips had hidden

- An `ensures` that names `result` in a function with a parameter named `result` now encodes the return value. It encoded the parameter, which was a Z3 sort error.
- An empty list passed to a helper (`mkTableFull([], [], c)`) takes the parameter's sort, whether it is written inline or hoisted into a `let`. It defaulted to `(Seq Int)`, which was a Z3 ERROR.
- A record update whose type is unknown no longer takes the first record in map order that has the field. That could build the wrong record, a Z3 ERROR, and differ from run to run. Without a type, a single matching record is used; several matching records make the function a skip.
- A `requires` checked at a call no longer binds the call's result. Over an opaque helper, that binding made a real `requires` violation look inconclusive.
- A helper's uninterpreted function whose signature names an undeclared sort is a skip. It was a Z3 ERROR.
