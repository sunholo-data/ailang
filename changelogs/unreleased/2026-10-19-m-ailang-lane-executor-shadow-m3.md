### Added — executor shadowing, M3: a shadow job writes nothing outside its run directory (2026-10-19)

M-AILANG-LANE-EXECUTOR-SHADOW M3 (`cmd/ailang/coordinator_cloud_shadow.go`). With `AILANG_TASK_SHADOW=true`
the cloud job enforces no side effects through three independent gates: (1) it skips the deploy-key write
pre-flight and returns after collecting evidence, before the push and the PR; (2) `childGitCredential` gives
the agent no git credential; (3) right after the clone, origin's push URL is pointed at a transport git does
not have, so any push from the clone fails whatever credential is present (an SSH deploy key's alias lives in
the job's HOME). The local commit is kept: the diff against the clone point is the shadow's evidence. Tested
end-to-end against a local bare remote, including an agent that tries to push on its own; an ordinary run
still pushes and opens its PR. Each gate was mutation-tested. Test seams added: `cloudWorkspaceRoot`,
`runExecutorFn`, `openPullRequestFn`.
