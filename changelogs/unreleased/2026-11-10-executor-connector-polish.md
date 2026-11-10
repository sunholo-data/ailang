### Fixed
- **The hosted executor's results no longer name the per-request directory.** `ailang_check` and
  `ailang_run` returned `/tmp/ailang-exec/<uuid>/main.ail` (and MOD010's canonical-path warning
  repeated it): an internal identifier, which OpenAI's review forbids in tool results. `stdout` and
  `stderr` are now scrubbed the way the logs already were (`scrubDir`, `executor_tools/executor.ail`).
  `executor_tools/smoke.sh` checks it and fails on the old code.
- **`ailang_check`'s description no longer promises Z3.** The executor image (distroless static) has
  no Z3, so `verify.available` is always false hosted. The description now says contracts are
  type-checked but not proved there, and points at `ailang verify` locally.
- **`docs_search` returns excerpts, docs pages first.** Each hit is path, title, first 10 headings
  and ~800 characters around the match instead of the whole page, and docs pages rank above archived
  teaching prompts (`prompts/v*.md`), which matched almost every query. A search fell from ~30 KB to
  3–11 KB in a local run against a built snapshot.
