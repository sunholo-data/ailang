# Iteration 81 — routing notes

- `resolve-role-spawn.sh`:
  - designer `recipe claude:claude-opus-5-5 declared:provider-pin`. Not spawned; the D-FLEET-25 ledger row is the spec, and no design doc is needed for a comment and prose fix.
  - planner `agent-tool opus fail-closed:no-doc`. `derive-planner-lane.sh` (no doc) gave `opus fail-closed:no-doc`.
  - executor `recipe codex:gpt-6.1-sol declared:provider-pin`.
  - evaluator `agent-tool sonnet declared:alias-pin`.
- The opus planner Agent spawn was DENIED by the spawn-pin hook: `deny:provider-pin — planner is pinned to codex:gpt-6.1-sol; Agent-tool alias spawn refused — use the cross-provider recipe`. This is the fifth instance of the documented Step 1b ⚠ case, after iterations 327–329 and fleet 80. The planner was routed to the pin under the codex planner recipe.
- codex probe rc 0.
- Planner: `codex exec --sandbox read-only -C <build worktree>`, rc 0, 97,389 tokens, wrote `plan.md` (its final message). Deviation from the recipe: it ran in the clean build worktree at `b23cadced` rather than a separate detached `.planner-wt-iter81`. The planner only reads, and that tree was identical to origin/dev.
- Planner scope was wider than adopted. It proposed editing the D-FLEET-25 ledger row and charter line 289; both were kept out of the code PR, and the record carries the note instead. It also proposed deleting the skill's reach-check snippet. The controller found the snippet broken for every mission (its grep matches `$REPO`, but the driver now sources `$MC_DRIVER_ROOT`) and had it repaired rather than deleted.
- Executor: `codex exec --sandbox workspace-write` in `.wt-fleet-iter81-dfleet25`, rc 0, 68,119 tokens. It edited `.claude` only. The controller synced `.agents` with `tools/launchd/sync-agents-skills.sh` (pre-check rc 1 STALE, sync rc 0, post-check rc 0), the same sanctioned writer as iterations 63 and 80. In the sandbox, `go test` (socket bind) and `check-context-docs` were labelled UNINFORMATIVE UNDER SANDBOX. The controller re-ran every gate outside the sandbox: all rc 0, `make test-launchd-drivers` 52/52.
- Evaluator, two spawns, both sonnet through the Agent tool in the foreground (cross-vendor from the codex generator, same vendor as the controller):
  1. Idle-verdict judge, at Gate 2: REFUTED (narrow). It found D-FLEET-25's mechanical sub-item routable. 64,455 tokens, 9 tool uses.
  2. Diff judge: round 1 PASS-WITH-EDITS 90 (63,800 tokens, 8 tool uses). Round 2 PASS 94 (68,540 cumulative tokens, resumed with SendMessage).
