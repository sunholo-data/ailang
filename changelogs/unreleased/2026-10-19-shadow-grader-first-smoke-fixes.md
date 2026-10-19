### Fixed — the shadow grader no longer fails correct work; never-started runs are ungraded (2026-10-19)

The first dev-plane shadow comparison graded a correct lane program as failing, twice over. (1) `check` ran
without relaxed module names, so a loose lane program whose module name does not mirror its path failed MOD010;
the lane does not enforce that rule, so grading now sets `AILANG_RELAX_MODULES=1`. (2) `test` on a module with
no tests exits 1 ("No tests found"); test gates now run with `--json`, and zero tests run with zero failed is
vacuous, not a failure. And a run that failed before its agent's first turn with no diff (an image, lane-check or
dispatch failure, as both motoko runs of that smoke were) is now `ungraded(did_not_start)` rather than scored
as the harness losing. The opt-in Firestore live test also deleted its documents after closing its client, which
silently failed and left five test comparisons on the dev plane; it now deletes first.

### Fixed — the shadow grader applies real patches (2026-10-19)

The job's evidence collector trims git's output, so a stored diff loses its final newline and `git apply` rejects
it as a corrupt patch: the first real primary diff (ailang-parse#105's) graded `apply_failed`. The grader now
restores the newline. Its test fixture had appended one itself, which is why the suite never saw it; the fixture
now trims exactly as the collector does.
