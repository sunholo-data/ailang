# Mission Dashboard — Motoko

**Snapshot**: 2026-09-06, after iteration 36. Overwritten every iteration; history lives in
`motoko-mission.md` (STATUS) and `motoko-mission-log.md`.

## Where the mission is

- **Epic**: `m-motoko-dst-refactor-migration` (gated; ungated work runs first). The thread before
  this iteration was hardening `tools/eval/motoko_connection_probe.sh`, rows 6a–6t.
- **This iteration did not advance the epic — goal unmoved.** M-MISSION-LOOP-WORKBENCH Phase 1 was
  landed **attended**, straight to `dev`, 22:15–23:02 local, with no charter row and no log entry.
  It left CI red in six checks. The loop fired 37 minutes later and Gate 1's red-outranks-the-queue
  rule applied; the red is this charter's own territory (clause 6), so it was not handed to V1.

## In flight

- **PR [#1055](https://github.com/sunholo-data/ailang/pull/1055)** — seven commits unbreaking `dev`.
  Four defects from the CI logs, a **fifth found only by measurement** (`kill_unix.go` had no build
  constraint, so `internal/mission` did not compile on Windows at all — fixing the validation alone
  would have left both Windows checks red while looking fixed), plus two self-review commits.
  Evaluator **PASS 85/100, zero blocking**.

## Next picks (banked, ordered)

1. **Row 6p M2/M3** — wire the wall-clock class, enforce the floor, gate `p_obs` (M2); derive the
   node ceiling (M3). Iteration 35 landed M1 only; its executor was capped after one milestone.
2. **Row 16** — changelog debt for the whole Phase 1 arc (feature + fix in one entry). Bookkeeping,
   ≤1 iteration. `make check-changelog` is index hygiene and will never surface it.
3. **Row 7** — profile restoration design (5 profiles, 14 of 18 model entries).

## Watch

- The attended session was **still landing commits while the loop ran** (M5/M6/M7 arrived
  mid-iteration, forcing a rebase and a re-verify against fresh origin). Expect more Phase 2 work on
  `dev` from outside the loop; re-check defects against fresh origin before acting on any of them.
- `TestLive_DoctorReproducesTheMeasuredDivergences` was red on the rig earlier tonight and is **now
  green** — M6 fixed the underlying drift. Rig-only; it skips off-rig, so CI never saw either state.

## Parked on Mark

- **None.** Decision ledger valid, 6 rows, **0 OPEN**.

## Loop posture

- Controller `claude:claude-opus-5`; executor `codex:gpt-5.6-sol`; evaluator `sonnet` — distinct
  providers, so generator≠judge holds. **No designer, no planner**: both routing-table branches gate
  on artifacts a CI-red fix-forward has not got (`derive-planner-lane.sh` → `opus fail-closed:no-doc`).
- Rotation pointer untouched at `codex:gpt-6-astra`; **Fable unspent**.
- Metered **$0.00** of the $5 ceiling. No GPU, no `rig.lock`.
- Bookkeeping issue **#987**; next weekly rotation Monday 2026-09-07.
