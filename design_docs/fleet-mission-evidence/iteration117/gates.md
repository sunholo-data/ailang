# Iteration 117 — controller gates and measurements

## Re-measure of row 13's two live signatures (Gate 2)

Instrument: `gh run list --workflow CI --branch dev` (300 runs, 2026-11-04..11-17) plus
`actions/workflows/ci.yml/runs?branch=dev&created=2026-10-08..2026-11-04` (519 runs; the windows overlap on 11-04) = 799 unique dev
CI runs. For every non-success run, the jobs list was read, and every failed or cancelled
`launchd drivers (bash 3.2)` job log was fetched with `gh api --allow-escape-sequences` and
ANSI-stripped. Log sizes were 120–197 KB, except one 32 KB log (10-09, which failed before any suite ran).
Positive control: a passing log carries `──── PASS tools/eval/test_motoko_connection_probe.sh`
(3 matches in job 131676783119).

Result: 28 unique failed launchd jobs (deduplicated by run id). 8 carry a probe signature: the timer arm 5× (10-10, 10-12 ×2, 10-19,
11-02) and the refusing-live-path arm 3× (10-15, 10-16 ×2, one with the
`treatment peer set contains OpenRouter endpoint: 203.0.113.8:443` line). The rest are
`test_quorum_artifacts.sh` / `test_agents_skills_sync.sh` (generated-docs/mirror sync, 10-27..11-09)
and `test_mission_pi_run_tool_stall.sh` (3×, 11-12/11-13). The last 300 runs carry no probe signature.

| date (UTC) | head | run | failing suite | signature |
|---|---|---|---|---|
| 2026-11-13T03:52:01Z | bfb1e6b75 | 43361761294 | tools/launchd/test_mission_pi_run_tool_stall.sh  | — |
| 2026-11-12T15:42:20Z | 5a1d85fc7 | 43250670913 | tools/launchd/test_mission_pi_run_tool_stall.sh  | — |
| 2026-11-12T15:30:33Z | 7b3629b95 | 43248541413 | tools/launchd/test_mission_pi_run_tool_stall.sh  | — |
| 2026-11-09T14:10:10Z | 7cb242a5a | 42632019365 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-09T14:08:10Z | c3b30f1f9 | 42631696403 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-09T13:56:33Z | a9f1c6e77 | 42629902478 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-09T13:52:10Z | 4b271f2d4 | 42629253237 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-05T07:26:26Z | c7c568a2f | 41987244588 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-05T07:24:00Z | f125d4851 | 41986957805 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-05T07:14:38Z | 4ddb10e5f | 41985849946 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-04T08:59:00Z | abd4dfd82 | 41803697385 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-04T08:53:17Z | 1fd47adbd | 41802955478 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-04T08:52:17Z | 8737f3416 | 41802827260 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-04T08:51:03Z | a0eba4892 | 41802669924 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-04T08:50:58Z | 321307bee | 41802659694 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-04T08:50:02Z | 69e012173 | 41802535106 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-04T08:43:55Z | f22bf1f4b | 41801751979 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-11-02T16:53:19Z | f00663ef1 | 41485837393 | tools/eval/test_motoko_connection_probe.sh  | bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 refusal= |
| 2026-10-27T13:42:48Z | 5f56181a1 | 40514469752 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-10-27T13:30:30Z | efcd7fe80 | 40512628345 | tools/launchd/test_agents_skills_sync.sh tools/launchd/test_quorum_artifacts.sh  | — |
| 2026-10-19T16:29:18Z | 793f0f67a | 39285812762 | — | bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 refusal= |
| 2026-10-16T14:20:25Z | a4b8f3211 | 38914956664 | — | refusing live path refuses with the control-void message lacked expected message: treatmen |
| 2026-10-16T11:26:15Z | 2e08e240f | 38893764503 | — | refusing live path refuses with the control-void message unexpectedly succeeded |
| 2026-10-15T20:55:45Z | 04496274e | 38805887596 | — | refusing live path refuses with the control-void message unexpectedly succeeded |
| 2026-10-12T14:54:34Z | 5257eca2f | 38266335577 | — | bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 refusal= |
| 2026-10-12T10:16:11Z | 3957d9f34 | 38235254228 | — | bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 refusal= |
| 2026-10-10T10:05:24Z | d1f2fc4c7 | 38043688355 | — | bound measurement refuses an unsuccessfully terminated timer (rc=0 expected_rc=73 refusal= |
| 2026-10-09T11:35:16Z | d9fbe2b24 | 37924652636 | — | — |

## Controller checks on the sprint head

- Diff vs plan: the changed lines of `git diff 22a72e7c5 85c446d61 -- tools/eval/test_motoko_connection_probe.sh`
  are byte-identical to the plan's §3 diff (59 lines, `cmp` rc 0). `/bin/bash -n` ok.
- Scope guard (`_scope_verdict fleet` sourced from the pinned `tools/launchd/githooks/pre-push`): the
  test file, the changelog fragment and both evidence docs → allow. Controls: `internal/types/unify.go` →
  refuse (language core); `tools/eval/qwen38-motoko-rerun.sh` → refuse (outside allowlist).
- Suite, first-party: `/bin/bash tools/eval/test_motoko_connection_probe.sh` rc 0, 77 s,
  `PASS: 61 probe self-test arms ran`, 0 `not ok`, `mode=timer rc=73 refusal_count=1 derived_count=0`.
- Hand mutant (not in the executor's set): lsof stub `*) exit 1 ;;` → `*) peer="192.0.2.1:51000->203.0.113.8:443" ;;`.
  The copy differs (`cmp`). rc 1 in 35 s, at arm 32: `not ok - a lane file read before the driver writes it
  samples no peer failed` + `INSTRUMENT FAILURE: treatment peer set contains OpenRouter endpoint: 203.0.113.8:443`
  (the CI line). Scratch copy removed; tree clean.
- `make test-launchd-drivers` at `34176f228`: rc 0, 373 s, `run-suites: all 58 suites passed`.
- Driver dry-runs (`AILANG_DRIVER_PINNED=34176f228…`, `MISSION_DRY_RUN=1`, `env -i`):
  - `MISSION_PROFILE=triage` (armed, idle): rc 0 in 5 s, but it exits at its idle fingerprint before lane
    resolution, so it cannot show a degraded lane. Not counted.
  - `MISSION_PROFILE=stapledon` (armed, idle): healthy rc 0 (56 s); degraded
    `MISSION_EXECUTOR_MODEL=codex:bogus-model-i117` rc 0 (67 s), executor → `claude:claude-haiku-5-5`.
    No pidfile residue. The driver is unchanged by this diff, so the dry-run is a done-gate formality here.
