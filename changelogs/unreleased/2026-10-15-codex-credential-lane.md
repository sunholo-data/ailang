### Fixed — coordinator: codex tasks wait for the shared credential instead of failing

Cloud codex jobs on the ChatGPT subscription share one rotating `auth.json`,
guarded by an exclusive lease that fails fast. The dispatcher did not know
about it, so a codex task dispatched while another ran exited at once
("credential already owned", 0 turns) and was banked as failed with no
requeue — four sprint-planner handoffs were lost this way on 2026-10-12/13.
The cloud dispatcher now holds a codex task as pending while another codex
task is queued or running, and dispatches it on a later tick.
