# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) + `v1-mission-log.md`.*

**Last iteration:** 231 · 2026-08-19 · **LANDED**
**Release:** v0.33.1 · dev green (21 checks, zero not-green on `3b18f60ce`)
**Bookkeeping issue:** [#745](https://github.com/sunholo-data/ailang/issues/745) (week of 2026-08-17)

## Last iteration in one line
`#671` — an AILANG program inside a package could not resolve its own imports unless you happened to
be standing in the right directory, and the error told you to create two files that were sitting
next to the source. PR [#790](https://github.com/sunholo-data/ailang/pull/790) → `3b18f60ce`.

## In flight / next picks
- **Queue head** — `m-sweep-orphans-2026-08-17`, **12 of 15 dispositioned, 3 remain**:
  `#694` (in-repo, `ailang editor install vscode` installs an extension VS Code marks obsolete —
  well-diagnosed with a verified workaround; the strongest of the three), `#672` (eparse-side,
  outside `MISSION_REPO`), `#656` (ailang-parse feature).
- **`m-list-cons-cells` programme** (Mark's `D-19 : B`) — roadmap `PARKED-ON-LANE`, owed one
  designer revision + one re-quorum. **Not a human ask.**
- Unblocked and cheap: `m-stdlib-reverse-delegates-to-builtin` (required under cons cells),
  `m-ci-no-job-timeouts`.

## Emerging theme worth a look
Three consecutive downstream-consumer reports where **the message is the bug** — `#679`, `RT_REC_003`,
`#671`. Each asserted a cause the emitting code could not know. Candidate for a systemic pass over
user-facing diagnostics rather than one report at a time.

## Loop cadence + routing
- Controller `claude:claude-opus-5` inline for measured defects with well-specified fixes
  (iterations 219–231). Designer/planner/executor/evaluator lanes idle.
- Designer rotation pointer: `claude:claude-fable-5`. **codex `gpt-5.6-sol` is quota-dry until
  2026-08-20 05:34** (re-probed as a command this iteration, rc=1); gemini is read-only under
  `CapRemoteSandbox` and cannot author. So the rotation currently resolves to Fable, one run per
  iteration — unspent for the last two iterations.

## Parked on Mark
**Nothing.** Decision ledger: 21 rows, **zero OPEN**.

## Quota posture
metered **$0.00** of $5 this iteration. Opus/Fable are subscription buckets. No GPU, no `rig.lock`.
