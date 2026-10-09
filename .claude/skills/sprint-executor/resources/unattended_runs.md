# Unattended runs (coordinator tasks, mission fires, `claude -p`)

When no human is in the session to answer, every "pause", "ask the user" or
"wait for approval" step in this skill becomes **commit and continue**:

- **Commit after each milestone** on the work branch. Uncommitted work is lost work:
  the coordinator pushes your branch when you finish. Never wait for permission to commit.
- **A blocker on one milestone is not a stop.** Record it (sprint JSON `notes`, the plan's
  checkbox left open with a one-line reason), then carry on with every milestone that
  does not depend on it. A failure that already exists at the starting commit (a broken
  docs build, a pre-existing failing test) is recorded and worked around, not escalated.
- **Stop only when no remaining milestone can make progress**, and say exactly what
  blocks each one in the final report.
- Decisions the plan leaves open: take the option the design doc's decisions point to,
  write down why in the sprint JSON, and continue.

Measured 2026-10-08: Claude Haiku 5.5 replaying four merged sprint-executor tasks stopped
twice at these pause points — once holding a finished sprint uncommitted for permission,
once halting after M1 over a docs failure that predated the task — where the original
executor completed both. Haiku now sits on the executor fallback chain.
