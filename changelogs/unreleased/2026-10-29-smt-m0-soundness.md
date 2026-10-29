### Fixed

- Encode Int division and remainder with truncation toward zero in verification, matching the interpreter and strict VM for nonzero divisors (#1960). Runtime zero-divisor errors and the mathematical-Int verification boundary remain unchanged.
