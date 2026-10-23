### Fixed — typed error variants in SMT contracts

Preserve native parametric ADT declarations and concrete constructor types when
verifying contracts such as `Result[int, AudioError]`. Imported `ResourceLimit`
payloads and contract-only datatype dependencies are declared before use;
unsupported or ambiguous shapes produce structured skips. The valid size guard
now verifies, while a broken successful result produces a counterexample through
the same encoding in `verify`, package verification and `ai-check`. Existing
callee-signature limits and numeric/runtime evidence scope remain unchanged.
