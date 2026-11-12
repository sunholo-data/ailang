### Fixed — pi role runs no longer leave processes behind (2026-11-12)

A pi role run could leave background processes running after the role exited. The field instances
(ticket `orphan:pi-runner:own-pgid-children-survive-role-exit`) were a Godot process orphaned under a
still-attached supervisor, an AILANG simulation child that sat in another process group, and a process
left behind by a deleted clone. The runner only signalled its own process group, and only when the
outcome was not `finished`, so a process that setpgrp()ed away, or any background descendant of a run
that finished cleanly, survived.

The runner (`scripts/mission_pi_run.sh`) now keeps an **ancestry ledger** in
`tools/launchd/lib/pi-run-reap.sh`. Every poll it records the `(pid, birth time)` of each live descendant
of the run, and of every descendant of a recorded process that is still alive. When the role exits it
signals that set on **every** outcome, including `ok`: TERM, a 2 s grace, a second ledger refresh to
catch children born during the grace, then KILL for the survivors. Membership is proven by pid, birth
time and ancestry only. It is not matched by process group, session or command line, so unrelated work
on the shared rig is never touched.

The verdict JSON gains two keys, `reaped_count` and `reaped` (the first 50 processes, each with its pid,
pgid, signal and command). Verdict names, exit codes and all existing defaults are unchanged. Reaping
never changes the verdict or the rc.

Residual: a double-fork whose intermediate process exits between two polls escapes the ledger
(R1 in the sprint plan `design_docs/planned/sprint-plan-pi-run-orphan-reap.md`).
