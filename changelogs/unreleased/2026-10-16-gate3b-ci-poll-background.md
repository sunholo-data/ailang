### Fixed — Gate 3b CI polling exceeds the Bash foreground ceiling (2026-10-16)

Run the SHA-pinned 30-minute poll in an explicitly timed background Bash call, with unique
SHA-bearing verdict files and bounded active reads. Foreground slices retain the original
absolute deadline. See the [sprint plan](../../design_docs/planned/sprint-plan-gate3b-ci-poll-background.md).
