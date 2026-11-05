### Fix recursive effect checks when local names shadow module functions (#2078)

Call-graph analysis now respects match binders, a module function's own parameters,
lambda/function-literal parameters, a block `letrec`'s name inside its own value,
local let/letrec scopes, sequential blocks and quantified variables. A recursive
row-polymorphic helper no longer acquires a caller's concrete IO effects through
a false recursion group. Genuine recursive IO violations remain rejected.

The traversal also follows quantifier bounds, channel operations, assertions and
quasiquote interpolations. Effect inference and validation are unchanged.
