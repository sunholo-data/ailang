### Added — `ailang verify` encodes tuples

A tuple of 2 to 8 components is now the SMT datatype `TupleN` of its components, so `(int, string)` is `(Tuple2 Int String)`. Before, any function that built, matched or passed a tuple was skipped ("tuple expressions cannot be encoded"), along with every caller of a helper whose signature had one. What now encodes:

- tuple expressions;
- tuple patterns in `match`, including literals inside (`(0, _)`) and tuples nested in a constructor or list pattern (`Some((a, b))`, `[(k, _), ..._]`);
- `let (a, b) = p`;
- tuple equality, which is component-wise;
- tuples in parameters, results, record fields, lists and type arguments (`Option[(int, int)]`).

Counterexamples show tuple values as `(3, "a")`, and witness replay passes them to the real function.

A module that declares its own type named `Tuple2` … `Tuple8` keeps it. A function that needs that sort is skipped, and the skip names the clash.

Downstream gap map: see the M-VERIFY-SOUNDNESS-COVERAGE program doc (item 4d).

### Fixed — an unannotated helper's return sort was assumed to be `Int`

A helper with no return annotation was defined as returning `Int`. Any other return type was a Z3 sort error (`! ERROR`). The return sort now comes from the helper's checked type. If that type has no SMT sort, the helper is not defined and its callers are handled like those of any unencodable helper. Such a helper can now also be called opaquely.

### Fixed — the `helpers_not_checked` caveat named helpers the proof did not go through

A helper reached only inside the body of a helper called by its contract was listed in the caveat, although that body is never encoded. The caveat now names only the opaque helpers the script applies.
