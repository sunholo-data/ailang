### Fixed — cloud execute-job publishes commits made after an agent's own push (2026-11-16)

Step 5b of `cmd/ailang/coordinator_cloud_task.go` decided whether anything was left to push from
`git log origin/<branch>..HEAD`. The job clones `--depth 1 --branch <base>`, which is single-branch,
so an agent that pushed its own branch created no `refs/remotes/origin/<branch>`: `git log` exited
128, the fallback found the branch on the remote, and the wrapper concluded "nothing to push". Any
later commit was silently never published, and `assertRemoteMatchesHead` was skipped with it
(measured at fleet iteration 49; design-doc-creator-daneel pushed itself on 2026-09-12/13). Step 5b
now compares `git ls-remote origin refs/heads/<branch>` with HEAD (`remoteBranchHead`, also used by
the post-push assertion) and pushes whenever they differ. Arm A09 of the sprint-executor
`test_cloud_branch_sync.sh` simulator now asserts the corrected decision. Fleet ruling D-FLEET-23 (A),
part (3).
