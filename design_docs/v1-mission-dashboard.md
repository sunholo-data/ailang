# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) + `v1-mission-log.md`.*

**Last iteration:** 233 · 2026-08-20 · **LANDED** · metered **$0.00** of $5 · no GPU
**Release:** v0.33.1 · dev green at pick time (18 checks on `bc0b5a8d4`, only `test` in flight)
**Bookkeeping issue:** [#745](https://github.com/sunholo-data/ailang/issues/745) (week of 2026-08-17)

## Last iteration in one line
`m-ci-no-job-timeouts` — **no job in any of the repo's 10 workflows declared `timeout-minutes`**
(control: `runs-on` × 27), so all 27 inherited GitHub's **6-hour** default. All 27 now bounded at
~2× their measured max, the 3 `apt` steps get 5-minute *step* bounds, and `internal/cihygiene`
keeps it that way. PR [#796](https://github.com/sunholo-data/ailang/pull/796) → `547803584`.

## ⚠ The find that matters more than the pick
**`Dashboard UI Build` has been red since 2026-07-10 — forty days, 10 of 10 runs — and nothing in
this mission had ever mentioned it** (charter mentions: 0). Path-filtered *and* non-required, so it
never blocked and nobody looked. The failure is `npm ci` in `docker/Dockerfile.dashboard`'s
`ui-builder` stage — the **image cannot be built** — and `cloudbuild-release.yaml` /
`cloudbuild-dev.yaml` build and push that same stage. Recorded on the already-open
[`#503`](https://github.com/sunholo-data/ailang/issues/503), whose title names this mechanism.

## Next picks
- **`m-ui-dependency-tree-unbuildable`** (NEW, top candidate) — **three stacked** peer conflicts in
  `ui/`. The obvious fix is *already refuted by measurement*: pinning `@vitejs/plugin-react` back to
  `^4` still fails and exposes an `eslint@10` conflict beneath. Needs a bump-vs-hold decision plus
  identifying the third cause (the first red predates both known bumps).
- **`m-list-cons-cells`** (Mark's `D-19 : B`) — `PARKED-ON-LANE`, owed one designer revision + one
  re-quorum. **Not a human ask.** Lane reset 05:34 today; re-probe at next fire.
- Cheap + unblocked: `m-stdlib-reverse-delegates-to-builtin`. Outside `MISSION_REPO`: `#672`, `#656`.
## Loop cadence + routing
Controller `claude:claude-opus-5` inline for measured defects with well-specified fixes (iterations
219–233); designer/planner/executor/evaluator idle. Fable designer run **unspent 4 iterations
running**. codex `gpt-5.6-sol` quota-dry at 03:07 (re-probed as a command, rc=1); gemini read-only
under `CapRemoteSandbox`. `mission-motoko` ran **concurrently** throughout (its PR `#795`).

## Parked on Mark
**One ask, carried forward from iteration 232 and still unanswered** — rotate
`AILANG_REGISTRY_API_KEY`, whose value a Gate-0 environment dump printed into that iteration's
transcript. Not exposed externally. Ledger: 21 rows, **zero OPEN**.
