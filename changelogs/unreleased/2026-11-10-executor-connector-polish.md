### Fixed
- **The hosted executor's results no longer name the per-request directory.** `ailang_check` and
  `ailang_run` returned `/tmp/ailang-exec/<uuid>/main.ail` (and MOD010's canonical-path warning
  repeated it): an internal identifier, which OpenAI's review forbids in tool results. `stdout` and
  `stderr` are now scrubbed the way the logs already were (`scrubDir`, `executor_tools/executor.ail`).
  `executor_tools/smoke.sh` checks it and fails on the old code.
- **The hosted executor proves contracts.** Its image now carries z3 (the same pinned, sha256-checked
  4.13.3 release CI installs; `tools/ci/install_z3.sh` is the one pin and now skips `sudo` as root),
  on a `distroless/cc-debian12` base because the z3 binary links glibc and libstdc++. `ailang_check`
  returned `verify.available: false` for every program; it now returns per-function `verified`,
  `counterexample` or `inconclusive` (5 s per function, under the server's 90 s process limit).
- **`docs_search` returns excerpts, docs pages first.** Each hit is path, title, first 10 headings
  and ~800 characters around the match instead of the whole page, and docs pages rank above archived
  teaching prompts (`prompts/v*.md`), which matched almost every query. A search fell from ~30 KB to
  3–11 KB in a local run against a built snapshot.
