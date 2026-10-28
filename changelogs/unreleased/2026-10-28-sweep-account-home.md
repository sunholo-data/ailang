### Fixed — Synthetic-HOME sessions could delete live mission pins (2026-10-28)

A synthetic-HOME dry-run launched a SessionStart sweep that deleted three live mission pins.
The hook now resolves the account home independently of HOME and protects its `.ailang-driver-pin/`,
`.ailang-ab/`, `.ailang-nightly/`, and `.ailang/state/worktrees/` prefixes, followed by any caller-supplied
`AILANG_SWEEP_PROTECT` unchanged. Failed, empty, or nonexistent account-home lookup skips the sweep.
Both fleet Guardrails dry-run recipes explicitly set `AILANG_WORKTREE_SWEEP=0` to prevent side jobs.

Stamp and log destinations remain HOME-relative: any other removal under synthetic HOME still logs
there. See the [approved plan](../../design_docs/planned/sprint-plan-sweep-account-home.md).
