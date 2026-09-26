# Mission iteration canary activation packet

Status: **APPROVED AND DRY-RUN VALIDATED; waiting for legacy Docs idle before dispatch.** Refreshed 2026-09-07 at 20:14 UTC.
Implementation evidence is separate from a successful live adoption demonstration.

## Concrete placement and limits

- Candidate binary: `/private/tmp/ailang-mission-iteration/bin/ailang`; build SHA-256 is recorded in `checks.md` after final verification. Do not install or replace the fleet binary.
- Source inspected: `/Users/voightkampff/dev/sunholo-data/ailang-docs`, clean at `1217d6334a78c8e94ad59ee86861d5607dc7dee3`. This checkout has an older charter than the sprint base. Reconcile task/base selection before activation; never switch its branch implicitly.
- Sprint base: `a12ed330d` on `sprint/mission-iteration`; retain exact approved artifact and decision commit hashes in the eventual work-item JSON.
- Isolated local binding proposed for the attended canary host: state DB `/private/tmp/ailang-docs-canary/runtime.sqlite`, workspaces `/private/tmp/ailang-docs-canary/workspaces`. All runtime invocations must share the same installed binding while active; save and restore any existing binding explicitly.
- One work item, remaining executor/evaluator stages only, imported design and planning prerequisites with committed approval references. No automatic backlog selection.
- Iteration: 3,600 seconds, 100,000 fresh input+output tokens, $5 metered guard. Executor: 1,800 seconds, 70,000 tokens, $3 guard. Evaluator: 1,200 seconds, 30,000 tokens, $2 guard. Limits are observed at usage boundaries; they cannot guarantee an external bill never overshoots an in-flight response.
- Proposed explicit canary role policy: executor `gpt5-6-sol` through Codex subscription; evaluator `pi-or-deepseek-v4-flash` through Pi/OpenRouter. Freeze the exact registry entries and their pricing from `internal/modelreg/models.yml` using `AILANG_MODELS_PATH` to a reviewed canary registry. No fallback candidates. Do not silently translate current `opencode` defaults to another harness. Evaluator must also differ from actual imported prerequisite authors.

## Why the task field remains unfilled

At the inspected sprint base, Docs queue items 0–11 are landed or ruled out; docs-12 is parked with an attended ruling authorizing **planning**, not execution. The live Docs checkout is older still. There is no demonstrated unowned, execution-approved item to bind honestly. The user has been asked for the intended item. Supplying a task label without its approved artifacts would defeat the handover contract.

Once selected, attach a strict `work-item.json` containing its full base, existing approval hashes/locators, exact allowed paths, required artifacts, frozen check argv, and acceptance IDs. Recheck current mission/coordinator ownership immediately before activation. Dry-run must succeed using the candidate binary and frozen model registry. Until then this packet does **not** satisfy the sprint's complete approved-item criterion.

## Attended activation sequence (requires review; not executed)

1. Inspect `ailang coordinator status`, `ailang coordinator pending`, the Docs launchd job, and its PID file. Confirm no active Docs controller, child author, or held admission owns the selected item. Record observed state and time; a clean Git status alone is not an idle check.
2. Preserve the current local runtime binding and exact scheduler/env state. Install the reviewed isolated binding only after confirming all callers on the host will share it. After confirming Docs is idle, suspend its next scheduled fire with the existing `~/.ailang/state/mission-docs.disabled` marker, recording whether that marker already existed. The new SQLite admission cannot fence an older controller. Use the candidate binary by absolute path; leave interval and routing settings untouched.
3. Set `AILANG_MISSION_REGISTRY=/private/tmp/ailang-mission-iteration/missions` and `AILANG_MODELS_PATH` to the reviewed canary model file. Unset API-key variables that would redirect Codex subscription. Run `.../bin/ailang mission iterate --work-item /private/tmp/ailang-docs-canary/work-item.json --dry-run` and retain the resolved snapshot.
4. Run the same absolute binary without `--dry-run` once. Capture status/exit, receipts, candidate commits, resource provenance and acceptance digests. No shell retry loop. Repeating the completed command must produce zero new provider work.

## Stop and rollback

For a reviewed ID `docs-canary-1`, inspect and cancel using a fresh printed version:

```bash
/private/tmp/ailang-mission-iteration/bin/ailang mission status docs --work-item docs-canary-1 --json
/private/tmp/ailang-mission-iteration/bin/ailang mission cancel docs --work-item docs-canary-1 --version VERSION_FROM_STATUS
/private/tmp/ailang-mission-iteration/bin/ailang mission status docs --work-item docs-canary-1 --json
```

Do not treat cancellation as proof that a process stopped. An ambiguous child keeps admission held; inspect the recorded owner process tree and use the store's explicit confirmed-stop reconciliation only after termination is established. The current CLI does not expose that attestation API, which is an operational limitation to resolve before unattended adoption.

No candidate is merged by iterate. Retain the database, receipts and worktrees for review; do not delete them as rollback. After confirming the attended invocation and descendants stopped, restore the saved binding. If the canary created the Docs disable marker (it was absent in the recorded baseline), remove that marker with `rm "$HOME/.ailang/state/mission-docs.disabled"`; if it already existed, leave it in place. This restores the previous schedule state without changing the interval or fleet binary. If the one-shot driver env opt-in is later approved, remove only `AILANG_MISSION_WORK_ITEM` from the exact env file changed and restore its saved version; do not stop unrelated missions.

## Attended continuation — 19:54 UTC

Mark authorized continuation and preparation of the canary. Read-only checks found
legacy Docs active (launchd PID 11260, latest inspected heartbeat gate-3 at 19:46 UTC);
its previous completion named docs-12 as its next task. Do not borrow that work item.
Coordinator is running (PID 763); its pending command returned no pending task approvals,
which does not establish that the separate legacy Docs controller is idle.

The original proposed Codex route is presently inadmissible: quota observation at
19:54 UTC returned `over`, weekly usage 50% against allowance 10%. The binary hash
was rechecked and matches checks.md. No runtime binding file existed at the inspected
installed path; repeat the check before writing anything there.

A concrete replacement scope is prepared in
[the work-item guide proposal](../../planned/m-docs-mission-work-item-guide.md):
complete the operator walkthrough in `docs/docs/guides/mission-iteration.md` only.
Proposed executor is `claude-sonnet-5`, evaluator `pi-or-deepseek-v4-flash`, retaining
the same caps. These are explicit proposed route changes, not silent fallbacks.
Use a dedicated source worktree from this sprint history and an isolated reviewed
registry copy, rather than switching the older live Docs checkout.

Pending: task scope approval, sprint-planner/check freeze, execution authority, committed
prerequisite/approval hashes, exact source base and registry snapshots, input dry-run,
then idle/ownership recheck and reviewed activation. No provider inference, installation,
scheduler mutation, outbound message, acknowledgement or live adoption occurred.

## Approved packet — 20:14 UTC

Mark approved the replacement scope and instructed “yep approved - proceed and execute”.
Design and sprint plan are committed; exact content authority is in `canary/authority.md`.
The frozen input is `canary/work-item.json`, copied byte-for-byte to
`/private/tmp/ailang-docs-canary/work-item.json`. Binary, input, registry and check-helper
hashes are in `canary/manifest.json`. These supersede the initial proposal above.

Actual source is `/private/tmp/ailang-docs-canary/source`, base
`ad1bf98d3fd8271ac8399f0f83e7e3ddf2675cb9`. Registry:
`/private/tmp/ailang-docs-canary/registry`; frozen models:
`/private/tmp/ailang-docs-canary/models.yml`. Executor Claude Sonnet 5; evaluator Pi/OpenRouter
DeepSeek V4 Flash; no fallbacks. Task `docs-canary-work-item-guide-1`, mission `docs`.

First dry-run found missing M5 wiring: AILANG_MISSION_REGISTRY was documented but ignored.
A red test reproduced wrong source selection and invalid-path fallback; ad1bf98d3 repairs
the documented behavior. Focused tests, full CLI/mission package suites, lint and build pass.
The rebuilt binary validates actual committed authority and resolves the intended source.
See `canary/dry-run-summary.json`. Original runtime fault-matrix evidence is retained and
was not rerun for this loader-only repair.

The installed runtime binding was absent; the temporary binding's exact bytes are at
`/private/tmp/ailang-docs-canary/installed-binding.toml`. Dry-run created no DB or runtime
workspace. Legacy Docs remains active; scheduler has not changed. Dispatch after idle only.

### Invocation

Run the absolute `/private/tmp/ailang-mission-iteration/bin/ailang` with arguments
`mission iterate --work-item /private/tmp/ailang-docs-canary/work-item.json` and the
explicit registry/model paths above in AILANG_MISSION_REGISTRY/AILANG_MODELS_PATH.
Use existing local authentication; unset ANTHROPIC_API_KEY and AILANG_AUTH_MODE for
Claude subscription, and use existing OpenRouter credentials without printing them.

### Rollback for this selected item

Read a fresh version with the absolute binary and
`mission status docs --work-item docs-canary-work-item-guide-1 --json`; if needed call
`mission cancel docs --work-item docs-canary-work-item-guide-1 --version N`, then inspect
again. Confirm owner/descendants stopped before restoring placement. Remove installed
binding only if its bytes equal installed-binding.toml and baseline records it absent.
Remove Docs disable marker only if the canary created it; preserve pre-existing markers.
Retain DB/receipts/worktrees; do not restore over another caller's changes.

## Evaluator budget amendment — 2026-09-08

Original canary produced/validated guide commit 58f9fd4bc2d5003dd9f04760edb8054a4367c114.
Independent evaluation stopped at 34,202 cumulative tokens against 30,000; no final
acceptance or completed-repeat check was claimed. The failed record is unchanged.

Mark authorized a reasonable evaluator limit. `canary/review-work-item.json` freezes
100,000 cumulative input+output tokens, the same 1,200-second/$2 evaluator guards,
and imports the accepted author artifact. Work ID: docs-canary-guide-review-2.
Its aggregate cap is also 100,000 because this successor has only an evaluator stage.
Authority: `canary/review-authority.md`. Source branch: sprint/docs-canary-review.
Actual CLI dry-run passed; evaluator-only runtime dispatch started 2026-09-08 05:25 UTC.
The initial plan/failed input remain historical evidence, not mutable runtime policy.

Evaluator-only retry ended failed at 100,216 fresh tokens (94,912 input + 5,304 output),
24 turns and 34 tool calls; metered cost $0.02533176. Its retained dispatch receipt shows
repeated reads of the same design, plan and candidate diff. This is a separate review
progress problem, not evidence that this bounded task needs another token increase.
No author rerun or final acceptance. No further retry launched. The 100,000-token
configuration remains the latest frozen setting. Both temporary binding and canary-created
Docs marker were removed after terminal-state/process checks; all evidence retained.
See review-status.json and review-rollback.json.
