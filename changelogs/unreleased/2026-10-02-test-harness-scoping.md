### Fixed — `ailang test` resolved private helpers to another module's function (2026-10-02)

The test harness put every loaded module's functions into one environment under their bare
names, so the module that sorted last won. A private `helper` in one imported module could run
another module's same-named `helper` (#1461, "record has no field: s"), and a private `isErr` or
`words` in the module under test could be replaced by the std/result or std/string export the
file never imported (#1516, "no pattern matched", "_str_words: expected String"). `ailang run`
was always correct. Each module now gets its own environment, cross-module references go through
module-qualified names, and only the module under test exposes its bare names to test bodies — in
all four harness paths (named tests, inline tests, clusters, requires/ensures properties).
Design: `design_docs/implemented/v0_51_1/m-test-harness-module-scoped-envs.md`.

### Fixed — an interrupted `ailang test` left `_namedtest_body_*.ail` in the package (2026-10-02)

Named-test bodies were compiled from a copy of the test module written into the package
directory and removed only on normal exit, so a CI timeout or Ctrl-C left the copy for
`pkg quality`, `publish`, `git add -A` and the next `ailang test` to pick up (#1502). The copy now
lives in a private temp directory, so no interruption can leave it in the package, and the
`WARNING MOD010 ... does not match canonical path '_namedtest_body_N'` line no longer appears in
test output. `ailang test` also skips (and names) leftover copies written by older versions.
Design: `design_docs/implemented/v0_51_1/m-namedtest-tempfile-leak.md`.

### Fixed — `ailang test FILE1 FILE2` ran only FILE1 (2026-10-02)

Every path given is now tested, with one aggregate summary, and the run exits non-zero if any
file fails. Flags may follow the paths (`ailang test a.ail b.ail --json`); before, they were
silently dropped along with every path after the first. `--package` takes one directory and
refuses more.
