# Fleet iteration 68 — done-gate and reality check

Ticket: `sprint-planner:template-make-test-changelog-filesize` (attended, 2026-10-09T06:22Z, blocking=none).
Base `4e6ea7b25091e2a50c8de64c68257c024e54a331`; branch `fleet/i68-planner-placement-stop-gates`, PR #2001.

## Reality check (controller, first-party, before routing)

| Ask | Disposition at base | Evidence |
|---|---|---|
| 1 focused executor gates | already landed | #1770 `dbe4b26a3` (2026-10-12), planner SKILL.md checklist bullet |
| 2 changelog fragments | already landed | same bullet |
| 3 800-line companion files | already landed | same bullet |
| 4 no plans in `ailang-core-triage/` | OPEN | no rule in the skill; plans were added there in `17254535d` and `a65e6c12b`; `internal/coordinator/triage_rubric_test.go:108` checks every row file there |
| 5 OPEN-only terminal STOP gates | OPEN | no rule in the skill |
| 6 jq in the planner image | already landed | `b2d21a070` (2026-10-09), `docker/Dockerfile.agent-base:55` |

Constraint found at pick time: `scripts/context_docs_baseline.txt:24` caps the planner SKILL.md at 597 lines (596 at base); the baseline is outside fleet scope.

## Gates (controller, outside the sandbox)

| Gate | Base `4e6ea7b25` | Head `249ee0d94` |
|---|---|---|
| `make test-launchd-drivers` | rc 0, 49 suites | rc 0, 50 suites (new `test_create_sprint_json.sh`: 49 passed) |
| `bash scripts/check_context_docs.sh` | rc 0 | rc 0 (SKILL.md 597 lines) |
| `make check-skills` | rc 0 | rc 0 |
| `make check-changelog` | rc 0 | rc 0 |
| `make check-file-sizes` | — | rc 0 |
| `shellcheck -S warning` (both scripts) | — | rc 0 |
| `sync-agents-skills.sh --check` | rc 0 | rc 0 |

Inside the executor's sandbox the driver suite read `4 of 50 suites FAILED` (`test_mission_kill_tree`, `test_mission_stall`, `test_stage_isolation`, `test_codex_controller_env`): all four pass outside it, so they were sandbox denials (process inspection, nested codex env), not regressions.

## Dry-runs (world, idle and armed; real HOME; `AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED=249ee0d94…`)

- Healthy: rc 0, `DRY RUN ok: mission=world`, `lanes=ok`.
- Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i68`): rc 0, `lanes=DEGRADED(1)`, executor handed to `claude:claude-haiku-5-5` (probe rc 1, `model_not_found`).
- Observations before and after both arms: sweep log sha prefix `87fe4fc2a0c40067` (173 lines), pin-listing hash `54060050452dcdd5`, world pidfile absent — all unchanged.

## Commit reconstruction

The executor could not commit (sandbox). The controller rebuilt one commit per milestone from `.snap/M1..M3`, running the new test and `check_context_docs.sh` at every boundary: `29b945e55` (M1), `0ea4dd233` (M2), `249ee0d94` (M3), then `298faf903` (sprint JSON M2 `passes: true`). `shasum -c` against the executor's final tree: 11 of 13 files identical; the 2 that differ are the two SKILL.md copies, where the controller moved one blank line so the new bullet sits inside the checklist list (line count unchanged at 597).

## Mutation drills

Executor (temporary copies): all 10 arms of the plan's table went red (raw, dotted, foreign-cwd, canonical, before-mkdir, before-import, diagnostic, valid-control, boundary-control, design-control). Judge: guard deleted → 30 of 49 arms fail; guard moved below `import-github` → 27 FAIL lines.
