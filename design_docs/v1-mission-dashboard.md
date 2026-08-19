# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) + `v1-mission-log.md`.*

**Last iteration:** 232 · 2026-08-20 · **LANDED**
**Release:** v0.33.1 · dev green (16 checks, zero not-green on `44aa3cab4`)
**Bookkeeping issue:** [#745](https://github.com/sunholo-data/ailang/issues/745) (week of 2026-08-17)

## Last iteration in one line
`#694` — `ailang editor install vscode` shipped an extension VS Code refused to load, because the
installer called a helper whose **name** said "refresh a cache" and whose **effect** was "uninstall".
The same helper is correct on the uninstall path. PR [#792](https://github.com/sunholo-data/ailang/pull/792) → `241221047`.

## In flight / next picks
- **Queue head** — `m-sweep-orphans-2026-08-17`: the **in-repo half is now CLOSED**. The 2 remaining
  orphans, `#672` (eparse) and `#656` (ailang-parse), are both outside `MISSION_REPO`.
- **`m-ci-no-job-timeouts`** — promoted by evidence, not opinion: this iteration watched **two**
  workflows wedge on `apt` install steps with no `timeout-minutes` (`Install z3` >26 min and 17m37s
  vs a 49s/100s/9s control; `Install jq` 1h30m). Cheap, and the loop pays for it every time.
- **`m-list-cons-cells` programme** (Mark's `D-19 : B`) — roadmap `PARKED-ON-LANE`, owed one designer
  revision + one re-quorum. **Not a human ask.** Lane returns **today 05:34**.
- Unblocked and cheap: `m-stdlib-reverse-delegates-to-builtin`.

## Emerging theme worth a look
**Four** consecutive downstream-consumer reports where the thing that *describes* the behaviour is
the bug — `#679` a stale warning, `RT_REC_003` a nonexistent option, `#671` an impossible
instruction, and now `#694` a **function name**. The first three were emitted strings, so a
"diagnostics pass" would have missed this one. The shared shape is an artefact asserting something
its author could not know, surviving because nothing tests a name.

## Loop cadence + routing
- Controller `claude:claude-opus-5` inline for measured defects with well-specified fixes
  (iterations 219–232). Designer/planner/executor/evaluator lanes idle.
- Designer rotation pointer: `claude:claude-fable-5`. **codex `gpt-5.6-sol` quota-dry until
  2026-08-20 05:34** (re-probed as a command this iteration, rc=1); gemini is read-only under
  `CapRemoteSandbox` and cannot author. The Fable designer run has now gone **unspent three
  iterations running**.

## Parked on Mark
**One ask** — rotate `AILANG_REGISTRY_API_KEY`. A Gate-0 environment dump I wrote
(`env | grep -iE "^AILANG_"`) printed its value into this iteration's transcript. Not exposed to
GitHub or any external service. Decision ledger otherwise: 21 rows, **zero OPEN**.

## Quota posture
metered **$0.00** of $5 this iteration. Opus is a subscription bucket. No GPU, no `rig.lock`.
