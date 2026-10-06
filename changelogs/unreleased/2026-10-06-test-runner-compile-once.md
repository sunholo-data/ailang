### Changed — `ailang test` compiles a file's named tests once, not once per test (2026-10-06)

Every `test "…" { … }` block used to be its own full compile of the test module (a temp file the compile cache never stores), so a run cost the number of tests × one module compile. On stapledons-godot:

| | Before | After |
|---|---|---|
| `sim/protocol_test.ail` (29 tests) | 3 min 36 s | 10.3 s (8.1 s with `--bytecode`) |
| whole package (287 tests) | 9 min 52 s | 99 s |

How it works now:
- All bodies compile together, as one `pure func __namedtest_<k>()` entry each, and each test runs on a fresh evaluator or VM over that one compile.
- Runtime-error positions now name the user's file: `<file>:<line>`, or `<file>:<line> (test body)` for an error in the body itself. They used to name a random temp file that was gone by the time anyone read the message.

When the shared compile fails, typically because one body does not type-check:
- Each test is compiled on its own as before, so only the broken test fails, with its own message.
- The fallback is never silent. stderr prints `named tests in <file>: could not share one compile (…)`, and `--json` gains `named_test_batch_failures`.
- If every body then compiles alone, the notice calls it a harness bug to report.

One routing change under `--bytecode`: a body that does not evaluate to a bool now runs on the VM. It still fails "expected bool result", but it no longer counts as an evaluator fallback, and it is no longer a `--strict-bytecode` failure.

Design: `design_docs/planned/v0_53_0/m-test-runner-compile-once.md` (Phase 2, properties compiled once, is separate). Refs #1328.
