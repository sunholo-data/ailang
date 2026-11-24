### Changed — `ailang verify` accepts functions with no effect annotation

A function with neither `pure` nor a `!` clause was refused by the verifier
("has effects", or "callee … has an inadmissible effect row" for a helper),
although `ailang check` already rejects any undeclared effect in such a body.
It is now treated like `! {}` when none of its parameters has a function type,
both as the verified function and as a helper, so callers of unannotated
helpers such as `std/json.get` move on to their next blocker instead of
skipping here.

A function with no `!` clause that takes a function-typed parameter is still
refused (its callback's effects could flow into it); give it `! {}` if it is
pure. A helper that declares an effect other than `Declassify` is refused with
a hint saying so.

Across the eight downstream corpora (`make verify-gap-map`), 3 results moved
from skipped to verified and 2 to inconclusive; nothing regressed.
