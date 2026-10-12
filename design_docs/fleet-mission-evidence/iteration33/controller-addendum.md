# Iteration 33 — controller addendum (not part of the judge's verdict)

The judge reported `test_driver_notify.sh` rc=1 (113/2) in its tree, with the same two arms
(`lane sets the digest routing note`, `no routing note when healthy`) failing on pristine base
`06d524c59`. Controller re-ran the suite in the judge worktree under a clean environment:

    env -i HOME=$HOME PATH=$PATH USER=$USER TMPDIR=$TMPDIR /bin/bash tools/launchd/test_driver_notify.sh
    → rc=0, 115 passed, 0 failed

The two failures are environment leakage from the controller session (the driver exports
`MISSION_ROUTING_NOTE` and other `MISSION_*` values into the controller and its sub-agents),
not a defect in the change. The executor's run and CI run without those variables.
