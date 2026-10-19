### Added — executor shadowing, M7 (code): the shadow report, the weekly post, and the shadow key (2026-10-19)

M-AILANG-LANE-EXECUTOR-SHADOW M7, everything that does not need the key. `ailang coordinator shadow report
[--since 7d|Nd|all] [--json] [--post] [--remote gcp]` prints per-harness runs, pass rate over graded runs,
runs with no output, cost, wall time, turns and wins; the verdict tally with the causes of `ungraded`; and every
disagreement with how to open both runs, the primary PR's fate and the reviewer's objection (golden-tested). The
cloud coordinator posts the same report to the `user` inbox on Mondays from 09:00 UTC, exactly once per ISO
week: the message ID is the week, written with `PutMessageIfAbsent`, so a scaled-to-zero restart cannot post
twice. It posts only while some agent is shadowed. A shadow job now puts `OPENROUTER_API_KEY_SHADOW` (new
registry var; D7) into `OPENROUTER_API_KEY` before the agent starts, and refuses to run without it rather than
bill the fleet key. A primary's recorded harness now falls back to its agent's image when the task record names
none. The coordinator guide gains an "Executor Shadowing" section. Still open: provisioning the shadow key,
mounting it on the pi and motoko jobs, the dev-plane enable on two agents, and the dev smoke.
