### Added — executor shadowing, M2: cloud dispatch of the shadow child (2026-10-19)

M-AILANG-LANE-EXECUTOR-SHADOW M2 (`internal/coordinator/shadow_dispatch.go`). After a primary task's cloud
dispatch succeeds, an agent with an enabled `shadow:` block and a sampling hit gets a child task
`<id>-shadow` (`ShadowOf` set; no message, thread, GitHub issue, stage or chain link) dispatched to the
shadow provider and variant with the primary's directive, policy and model. Sampling is a pure function of
the task ID (FNV-64a), so it can be recomputed. The shadow's dispatch clears every push and PR field and
any user-provided key, and carries `DispatchParams.Shadow`, sent to the job as the new registry variable
`AILANG_TASK_SHADOW` (always sent; `false` for ordinary tasks). A shadow is dispatched at most once per
primary (checked explicitly, because Firestore's create overwrites), is never retried, and never affects
the primary. A shadow child found pending in the dispatch list was orphaned and is failed, never dispatched
as an ordinary task. A shadow whose provider leases a credential lane (codex) is skipped. The job does not
act on the flag until M3, and no agent enables shadowing until M7.
