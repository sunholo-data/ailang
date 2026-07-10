# LIMITATIONS re-verification transcripts
Verified at: AILANG v0.28.0-141-g379990ad5 on 2026-07-10
Binary: v0.28.0-141-g379990ad5 (worktree build, non-dirty)

## RESOLVED (were documented as broken; now work)

### Polymorphic arithmetic lambdas -> 5.85
5.85

### match in block-body lambda in HOF -> [zero, ok, ok]
[zero, ok, ok]

### String interpolation -> Value: 42
Value: 42

### Pattern guards -> big
big

### Block expressions (multi-stmt) -> step 1/2/3
step 1
step 2
step 3

## STILL OPEN (verified reproduces)

### Y-combinator occurs check
Error: type error in tmp/limtx/ycomb (decl 0): type unification failed at [function application at /tmp/limtx/ycomb.ail:3:23]: occurs check failed: α2 occurs in α2 -> α3 ! {...ε4}

### if-else requires braces
Error: if-else branches require explicit braces when using let bindings

### ? operator not implemented (parse error)
PAR_NO_PREFIX_PARSE at /tmp/limtx/qmark.ail:4:12: unexpected token in expression: ?

### Retired 'match ... with |' syntax rejected (PAR019)
PAR019 at /tmp/limtx/oldmatch.ail:3:14: 'match ... with' is not valid AILANG syntax (ML/Haskell pattern detected)

### ++ is list-only (string ++ is a type error)
Error: type error in tmp/limtx/plusplus (decl 0): ++ operator at /tmp/limtx/plusplus.ail:2:47: `++` is for lists only. For strings use "${expr}" interpolation, concat([parts]), or join(sep, parts).

### Duplicate record types run fine in interpreter (codegen-only issue) -> 1.0
1.0
