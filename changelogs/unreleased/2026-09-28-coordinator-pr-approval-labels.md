### Fixed — coordinator PRs say what they wait for and what merging starts (2026-09-28)

Every coordinator PR carried only `agent-task`, so a triage note, a design doc, a sprint plan and
code looked identical, and nothing said that merging a plan approves the task and starts
sprint-executor (feedback on PR #1339). PRs now also carry the agent's own approval label from the
registry (`needs-design-approval`, `needs-sprint-approval`, `needs-implementation-approval`;
`needs-merge-approval` for agents without one; none when the agent skips approval), and a PR whose
approval hands off opens with "⚠️ Merging this approves the task and starts **sprint-executor**".
Both come from the same registry edges as the approval card. Plumbed dispatcher → job via
`AILANG_PR_LABELS` / `AILANG_MERGE_STARTS`.
