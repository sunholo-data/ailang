# Mission Dashboard — V1

_Snapshot after iteration 292 (2026-08-27). Overwritten each iteration; history lives in the charter STATUS block and the mission log._

## Latest
- **Release**: v0.34.0 · `origin/dev` @ `445ccb550`
- **Landed this iteration**: [#937](https://github.com/sunholo-data/ailang/pull/937) → squash `445ccb550` — **M2** of `m-prompt-version-freeze-on-first-bank`: the frozen-prompt CI gate
- **The gate is proven to RUN in CI**, not merely declared: both new steps report `completed/success` in the `test` job on the merged head. `make ci` is never invoked by CI (measured, 0 occurrences), so `ci:` membership alone would have shipped a guard that never fires.
- **Evaluator FAILED round 1 (78/100)** on a real blocking defect and it was fixed before merge: the workflow this sprint documents (`create_prompt_version.sh` → step 5) left the tree inconsistent and the gate red. Its proposed remedy was declined with a measurement; the writer was fixed instead.

## In flight / next
1. **M3** — close the agent-mode verification hole (`internal/prompt/loader.go` never verifies the prompt hash; `langreg` converts a load FAILURE into a SUCCESS attributed `"default"`). Highest-risk milestone of this sprint.
2. **M4** — bank-time `prompt_sha256` byte evidence.
3. `m-prompt-freeze-mutable-mirror-unchecked` — **new, from the iteration-292 judge**: for a MUTABLE version the embedded `.md` is checked by nothing (L3(d) is frozen-only by design). Reproduced: registry hand-synced, mirror `.md` deleted, gate rc=0.
4. `m-fmt-corpus-gate-freeze` (`D-39` sequences it here) · `m-browser-session-serving-mode` (AITANA-DEMAND).

## Loop / routing
- Cadence: launchd `dev.ailang.mission-control`, pinned worktree `~/.ailang-driver-pin/v1`.
- This iteration: controller `opus` · designer **not spawned** (doc exists — Fable diet UNSPENT) · planner **not spawned** (plan exists) · executor `codex:gpt-5.6-sol` · evaluator `sonnet` (own worktree). generator≠judge held on both axes.
- metered **$0.00** of $5.

## Parked on Mark
- **`D-42`** (the only OPEN ledger row): standing authorisation to reconcile this checkout to `origin/dev` unattended? **Not exercised this iteration and not needed** — local dev was 0 ahead / 0 behind at Gate 1 for the first time in weeks.

## Quota posture
- Anthropic available (`MISSION_ANTHROPIC_AVAILABLE=1`); codex probe rc=0; billing tripwire **CLEAN**.
- `SonarCloud Code Analysis` red on 5 of the last 6 analysed dev commits — **inherited, non-required, named, not the pick**.
