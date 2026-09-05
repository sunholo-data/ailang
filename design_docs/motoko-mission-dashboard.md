# Mission Dashboard — Motoko

**Snapshot**: 2026-09-05, after iteration 35. Overwritten every iteration; history lives in
`motoko-mission.md` (STATUS) and `motoko-mission-log.md`.

## Where the mission is

- **Epic**: `m-motoko-dst-refactor-migration` (gated; ungated work runs first). **Active thread**:
  hardening `tools/eval/motoko_connection_probe.sh` + its self-test suite, rows 6a–6t.
- **Bar clause 1** (the tree gates green from source): suite arm count **46 → 57** this iteration.

## In flight

- **PR [#1048](https://github.com/sunholo-data/ailang/pull/1048)** — row 6p **M1 of 3**: the suite now
  measures the host fork rate in-test and derives its bounds from it. Additive; nothing consumes the
  scale yet. Evaluator PASS 81/100, its one blocking finding fixed in the PR.

## Next picks (banked, ordered)

1. **6p M2/M3** — wire the wall-clock class + enforce the floor + gate `p_obs` (M2); derive the node
   ceiling on the discovery arm (M3). Both specified and unblocked; the executor's 30-min cap is the
   only reason they are not in #1048.
2. **6s** — no in-suite gate notices a self-test ARM disappearing; corroborated this iteration when
   the evaluator added an arm and the suite stayed green.
3. **6t** — Gate 3b's poll cannot reach its 30-min deadline inside a 10-min foreground tool call.

## Loop cadence + routing

- Controller `claude:claude-opus-5`. Designer **fable** (astra's first real run failed — see below).
  Planner **codex:gpt-5.6-sol**. Executor **codex:gpt-5.6-sol**. Evaluator **sonnet**, own worktree.
- **Two routing defects, both in the spawn-pin hook**: it enforces the DECLARED pin and knows nothing
  about (a) fallbacks — a failed primary cannot be re-routed via the Agent tool at all — or (b)
  `resolve-role-spawn.sh`'s derivation, which it contradicted for the planner.
- **`codex:gpt-6-astra` failed its first real designer run**: rc=0, zero artifact, ~3 min, after a
  passing probe. Fell back to fable, flagged.

## Parked on Mark

**Nothing.** Decision ledger: 6 rows, **0 OPEN**.

## Quota / cost posture

Metered **$0.53** of the $5 iteration ceiling (2 quorum rounds + 1 absentee re-run). Quota buckets:
opus (controller), fable (designer ×2 — the diet's one-doc allowance), sonnet (evaluator), codex
subscription (planner, executor). No GPU, no `rig.lock`.
