### Added — Eval hidden tests and a per-benchmark exec timeout (2026-10-28)

A benchmark can now declare `hidden_tests_dir` (a sibling directory of `<name>.in` / `<name>.ans` pairs) and `exec_timeout` (seconds). Hidden tests are graded after the visible test passes, smallest input first, stopping at the first failure, in both standard and agent mode. They are never rendered into a prompt or an agent workspace; the visible `expected_stdout` still is, so it is a sample, not an oracle. Rows of such benchmarks bank `hidden_total`, `hidden_passed` (the passing prefix) and `hidden_first_fail` (`<test>:wrong_output|timeout|runtime_error`), and `stdout_ok` requires every hidden test. A hidden failure banks the same `error_category` a visible failure of the same kind does, and its output never reaches the self-repair prompt. `exec_timeout` replaces every code-execution limit for that benchmark (standard `--timeout`, and agent mode's 10s AILANG / 30s Python, JS/Go and probe limits), so one budget applies to every language. Benchmarks without the keys are unchanged. Unpaired files, an empty directory, or combining the key with `grading: quine` / `grade_entrypoint` fail at load.

Motivation, measured on a 2026-10-28 competitive-programming spike: AILANG solutions that passed the single visible test survived the problems' own hidden tests 3 of 14 and 2 of 17 times, against 10 of 15 and 9 of 12 for Python, so the one-sample oracle overstated AILANG far more than Python. See [M-EVAL-HIDDEN-TESTS](../../design_docs/planned/v0_55_0/m-eval-hidden-tests.md).

### Added — Competitive-programming benchmark pack (2026-10-28)

`cp_60166`, `cp_60159` (`stretch`, with hand-written AILANG references) and `cp_60164`, `cp_60153` (`experimental`, no AILANG reference yet): four ICPC/CCPC 2024 problems from AetherCode (CC-BY-4.0, `benchmarks/hidden_tests/AETHERCODE.md`), each with 8–22 hidden tests of at most 64KB and `exec_timeout: 30`. Each was chosen because the spike showed an AILANG-vs-Python delta that held over three or more trials: no BigInt (`cp_60166`), no cheap set for cycle detection (`cp_60159`), no cheap int accumulate (`cp_60164`), and algorithm downgrade under expensive updates (`cp_60153`). The references are in `benchmarks/frontier_refs/cp_*`.

### Changed — Agent-mode validation runs report timeouts (2026-10-28)

`ValidationResult` now carries `TimedOut` from every lane, so a timeout can be told apart from a crash. The banked categories are unchanged.
