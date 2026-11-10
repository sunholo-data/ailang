### Fixed — the `Maintenance instrument` workflow passes again on `dev` (2026-11-10)

After `JInt` landed, the workflow failed on every PR that touched the eval harness: the pinned
`ailang-packages` commit read counts and the task pack's `cap` by matching `JNumber` only, so integer
values were rejected. Both are fixed upstream with the number accessors, which work on old and new
compilers alike (ailang-packages #187, and #188 which re-freezes the resource-ownership task pack). The
workflow's pin moves from `9ac50f6a` to `70ddbf53`. `go test ./internal/eval_harness -run
TestMaintenance` passes against the new pin with a `dev` compiler.
