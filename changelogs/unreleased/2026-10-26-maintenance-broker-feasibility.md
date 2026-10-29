### Research — maintenance broker lifetime qualification remains blocked

Added a repeatable no-model macOS canary using the locked sandbox runtime and
the existing workspace reaper. A detached child that changes cwd survives normal
exit, timeout and fixture owner death even after process-group kill plus cwd
reaping. Positive controls verify owned Python/AILANG work, workspace-cwd cleanup
and an unaffected bystander; post-cleanup process states exclude zombies.

The expected-blocker regression passes while the strict lifetime assertion still
fails. Source-qualified reports and independent artifact review are banked under
`design_docs/validation/maintenance-broker/2026-10-26/`. The approved broker sprint
stopped at M1; M2–M7 and live inference remain unopened. This is evidence about
the current cleanup recipe, not a language-quality result or a general macOS
impossibility claim.
