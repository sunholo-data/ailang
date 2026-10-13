### Fixed — Gate 0 ledger resolution provenance (2026-10-13)

Gate 0 now checks the earliest verified RESOLVED addition instead of the row's creation or a
later reword, reporting separate flip and creation commits without exposing identity addresses.
GitHub-like committers yield `undetermined:squash-merge` before author comparison, so ambiguous
squash history cannot trigger reopening. Only a fleet-authored non-squash flip triggers
SELF-RESOLUTION; missing or unavailable provenance surfaces an instrument failure. Git display
names remain conventions, not authenticated rulings. The Bash 3.2 fixture suite is wired into
`test-launchd-drivers`. See [the sprint plan](../../design_docs/planned/sprint-plan-gate0-ledger-provenance.md).
