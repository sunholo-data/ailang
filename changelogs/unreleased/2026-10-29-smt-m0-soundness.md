### Fixed

- Encode Int division and remainder with truncation toward zero in verification, matching the interpreter and strict VM for nonzero divisors (#1960). Runtime zero-divisor errors and the mathematical-Int verification boundary remain unchanged.
- Preserve contract-only callee identity by actual arguments and instantiate guarded specification facts only on evaluated lexical paths. Avoid shared-result false proofs and global axioms over uncalled partial inputs (#1961).
- Keep specification arguments, match guards, helper environments and bounded predicate variables capture-free; filter nonnullary callee interpretations from displayed counterexample values.
