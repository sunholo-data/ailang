Refs #495

Inline test rows now evaluate composite expected values (lists, tuples, records and
local ADT constructors) through the existing module-scoped harness evaluator.
Check, ai-check and LSP reject unsupported row expressions with TST001/TST002;
the runner validates rows before execution and reports errors without panics.
Invalid neighboring rows cannot contaminate a shared harness compile, and check
validation runs before cache lookup so a runner cache entry cannot bypass it.

F3 was re-verified with Z3 4.8.12: composed contracted predicates verify; the known
float-division encoding error exits 1 in verify and ai-check. The original 213-line
sketch remains unavailable. This change does not close #495 or modify SMT encoding.

Validation: composite and negative fixtures, row grammar/evaluation parity,
conversion error propagation, mixed valid/invalid rows with a named neighbor,
warm-cache check refusal, CLI JSON reports, LSP locations and capability denial.
Full make test, lint, architecture/format/file-size gates and verify-examples pass.
Cloud test setup uses local CGO tools, a temporary orphan-reaping supervisor and
CI=1 (the existing coordinator shell-timeout test skips in CI). Example gate:
232 passed, 0 failed, 9 skipped; manifest zero drift.

Limits: rows still exclude binary operators and unsupported AST forms; name/type
resolution and imported constructors remain runtime limitations. The test CLI does
not expose --caps; this sprint preserves the existing capability-denial behavior.
