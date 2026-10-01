# HANDOVER — motoko in the ailang_only lane (2026-10-01)

Design: [m-motoko-ailang-only-lane.md](m-motoko-ailang-only-lane.md). Goal (Mark): motoko is the best
harness to write AILANG; the ailang_only executor boxes are the most secure place to do it. This
session built the lane end to end; **the pi-vs-motoko A/B it ran is VOID** (see below). Start the next
session here.

## What is built and where

| Piece | Where | State |
|---|---|---|
| motoko refuses tool restrictions it cannot enforce | ailang #1429 | merged |
| `ailang run --chdir`, `ailang policy-tool --request-file` | ailang #1438 | merged |
| Executor lane gate: profile via motoko's own `print_config_json`, strict-enforcement probe, post-run session check (`policy_violation`) | ailang #1442 | merged |
| Version-query 30s + warning; lane canary absolute path; `pi-or-glm-5-3-flash` | ailang #1445 | merged |
| **Per-run lane policy (`fs_sandbox = "${WORKSPACE}"`), step-exhausted runs graded + costed** | ailang **#1451** | open → merge when green |
| `extensions.strict` enforced | arniwesth/motoko_agent #205 | **merged upstream** (the rollout gate) |
| strict follow-ups (sandbox-proof verify target, empty-registration fixture, blank entries) | arniwesth #206 | open |
| strict is fail-open if the profile itself does not load | arniwesth issue #207 | open (our executor guards it) |
| hide native tools / no duplicate schemas | arniwesth issue #204 | open |
| `motoko_ext_ailang_policy` + `ailang_only` profile (max_steps 300) + `ailang_edit` native-shape fix | `sunholo-voight-kampff/motoko_agent` branch **`sunholo/main-dst-20261001`** at `780b9abd` (= upstream main `fe108db7` + our commits) | pushed |
| Cloud image pin → `780b9abd` | ailang branch `feat/motoko-image-pin-lane` | **pushed, NO PR** — decide after a valid A/B |
| Lane model entries | `motoko-lane-or-glm-5-3-flash`, `pi-or-glm-5-3-flash` | in dev |

`~/dev/mk-main` (the rig's live eval checkout, rotation uses it) is still on the old
`sunholo/main-dst`; switch it to `sunholo/main-dst-20261001` only while the rotation is idle.

## What is proven, and what is not

- **Proven — the boundary.** Direct lane runs: no policy → refused to start; forced `BashExec` /
  `ReadFile` → `denied_by_policy`; `ailang_read` outside the sandbox → refused by policy-tool; no
  secret leaked. In the A/B, motoko attempted 8 non-lane calls, all denied, zero executed.
- **NOT proven — quality or cost vs pi.** The 2026-10-01 A/B (eval_results/lane_ab_20261001 and
  _motoko) is VOID. Its own transcripts showed: one shared `fs_sandbox` = the eval root sent relative
  paths outside each run's workspace (8/12 wrong answers graded the untouched template); arms ran
  concurrently and collided; step-exhausted rows banked $0 (motoko's real total $3.89 vs pi $3.39);
  stdlib changed mid-run (v0.50.1 released). Do not cite its numbers.

## Protocol for a valid re-run (do it exactly)

1. Merge ailang #1451; build ailang from dev ONCE and use that one binary for both arms
   (`go build -o <dir>/ailang ./cmd/ailang`; put `<dir>` first on PATH with a `motoko` shim →
   `~/dev/mk-rebase/scripts/run-agent.sh` or a checkout of `sunholo/main-dst-20261001`).
2. Policy file: `security_mode = "restricted"`, `allowed_caps = ["IO","FS"]`,
   **`fs_sandbox = "${WORKSPACE}"`**, `entry = "main"`, `timeout_ms = 60000`. (#1451 refuses anything
   else.)
3. Arms **sequentially**, not concurrently: `ailang eval-suite --agent --models pi-or-glm-5-3-flash
   ... --tool-policy ailang_only --policy-file <policy> --trials 3`, then the same for
   `motoko-lane-or-glm-5-3-flash`. Benchmarks: the frontier tier (or the 23 in the scratch
   `ab_benches.txt`: 8 frontier + 15 stretch). Check the rig rotation is not mid-chunk.
4. **Before reporting any number**, open transcripts: for each arm, at least 5 failing rows — did the
   solution land at the workspace path the grader reads? Is `.code` the template? (motoko: session
   JSONL in the motoko repo's `.motoko/logfile/`; pi runs with `--no-session`, so only
   `agent_transcript` + stdout.)
5. Then `ailang eval-paired <motoko-dir> <pi-dir>` (discordant pairs, headroom), turns and cost per
   solve, zero `policy_violation` rows. Known asymmetry: pi's output cap is 32000 (wire), motoko's
   65536 — check `finish_reason=length` on both.

## Known open issues (not yet fixed)

- motoko `empty_stop_guard` ended a run on a `finish=length` (output-cap) response as if final
  (gauntlet_10). Probably an upstream issue for Arni.
- Lane-prompt contradiction: the benchmark task text still says "you have the `ailang` command:
  `ailang run ...`" while the lane prompt says there is no shell (both arms).
- pi's lane prompt tells agents to use `.ailang-scratch/`, which trips MOD010 module naming.
- Cloud executors still run the old fork image until the pin PR + a release + promote.
