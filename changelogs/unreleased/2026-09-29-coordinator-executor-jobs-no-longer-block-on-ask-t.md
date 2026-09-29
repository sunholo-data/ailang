### Fixed — coordinator executor jobs no longer block on "ask the user about the inbox" (2026-09-29)

On 2026-09-28, two sprint-executor handoffs (GPU gateway Phase 2 and bytecode VM determinism)
BLOCKED on their first turn, with no files changed, asking for "user direction on the inbox". The
SessionStart hook had injected the unread-inbox and approvals banner into the headless Cloud Run
session, and the agent applied CLAUDE.md's attended rule, "summarize to the user and ask before
acking", with no user present. `session_start.sh` and `brain_on_prompt.sh` now emit nothing when
`AILANG_TASK_ID` is set (every coordinator task carries it), just as they already did for
`AILANG_MISSION_STAGE`. CLAUDE.md's session-start rule is scoped to attended sessions and says
an unattended task skips the inbox. `test_stage_isolation.sh` asserts both markers, and its
control arm now unsets them explicitly. Mutation-checked: removing the new guard fails the suite.
