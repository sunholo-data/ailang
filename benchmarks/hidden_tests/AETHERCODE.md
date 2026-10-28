# AetherCode hidden tests

The `cp_*` directories hold held-out tests for the `benchmarks/cp_*.yml` benchmarks
([M-EVAL-HIDDEN-TESTS](../../design_docs/planned/v0_55_0/m-eval-hidden-tests.md)).
Each `<n>.in` / `<n>.ans` pair is test `n` of the problem, copied unmodified.

**Source:** AetherCode, `v1_2024` split — https://huggingface.co/datasets/m-a-p/AetherCode
**Licence:** CC-BY-4.0. Problems are from the ICPC Central Europe 2024 regional and the
CCPC 2024 Final, as collected by the AetherCode authors.

Selection: tests with inputs of at most 64KB, minus the example the prompt already shows.
The visible example stays in the YAML as `stdin` / `expected_stdout`; these files are
never rendered into a prompt or an agent workspace.

| Benchmark | Hidden tests | Language gap it tracks |
|---|---|---|
| `cp_60166` | 12 | no BigInt: exact comparison of huge products |
| `cp_60159` | 17 | no cheap set: cycle detection (or Floyd's algorithm) |
| `cp_60164` | 8 | no cheap int accumulate: 2D difference array |
| `cp_60153` | 22 | algorithm downgrade under expensive updates |
| `cp_60149` | 34 | unattributed: consistent AILANG wrong answers (6 of 33 small tests, 3 of 3 trials) |
| `cp_60197` | 40 | unattributed: consistent AILANG timeouts (3 of 3 trials) |
