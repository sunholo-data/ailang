# Mission Dashboard — Motoko

*Snapshot, overwritten every iteration. History lives in the charter STATUS stamps and the log.*
**Last refreshed**: 2026-09-30 (attended charter reset — no iteration has run since 39) · base `91eb860a2`

## Where the mission is

**PAUSED, and RESET for motoko main** (`D-MOTOKO-RESET-1`, Mark, attended 2026-09-30). The
migration epic is done: evals and the `motoko` shim run `~/dev/mk-main` (`sunholo/main-dst` =
Arni's `main`, DST core, extension ABI 8.0, plus our carried commits); extensions are in-repo
packages; the ABI 2.2 fork and its registry packages are retired. Goal unchanged — the best harness
for writing AILANG, graduating to a mission executor. **Headline KPI**: motoko's pass rate vs pi and
opencode on the GPU rotation (same model, benchmarks, window), plus the cloud equivalent — **no
comparable reading exists yet** (row 23 builds it).

What we know so far (Premise log P5–P9, reported by the attended session): local
`motoko-local-qwen3-8-27b-microrag` 34/41 (83%) on the rotation since 2026-09-29 12:00, all 7
failures 1h timeouts driven by qwen3.8's hidden reasoning (thinking stays ON by Mark's ruling);
pi/opencode broken on that window by a since-fixed rig provider gap; the `ailang_tools` A/B on
deepseek-v4-flash is at ceiling, not a win.

## Queue top

1. **Row 20** — cloud motoko executor on motoko main, verified end-to-end in test
   (#1413 → release → promote → one real cloud task banked).
2. **Row 21** — upstream #200 (`motoko_ext_ailang_tools`) merged, then drop our carried commit.
3. **Row 22** — port `fmt` to ABI 8.0; fix `context_limit` resolving to 0 in eval workspaces.
4. **Row 23** — KPI instrument: paired motoko vs pi vs opencode (local + cloud).
5. **Row 24** — `ailang_tools` on a set with headroom (local qwen3.8).
6. **Row 25** — executor-lane design + gate trial (World first).

Carried loop-health rows: 6s (unblocked by `D-MOTOKO-P2-1`), 6m, 6t, 17/18, 19; R3 (row 9) parked
behind row 23.

## Parked on Mark — 1 open decision

- **`D-MOTOKO-RESTART-1`** — when to restart. Restart needs Mark's say-so to remove
  `~/.ailang/state/mission-motoko.disabled`; the loop then runs on the **shell driver**
  (`tools/launchd/mission-control.sh`, `MISSION_PROFILE=motoko`) and its first fire is iteration 40
  on row 20. No default: nothing times out into a restart.

## Blocked, not waiting on Mark

- Row 20 needs a release + promote to reach test (release cadence is outside the loop).
- Row 21 waits on the upstream maintainer's review of #200 (bounded: 14 days, then move on).

## Loop health

- Kill switch present (measured 2026-09-30). Plist repo copy: `StartInterval=46800` (13h).
- Routing: resolved live by the shell driver (`MISSION_PROFILE=motoko MISSION_DRY_RUN=1`); codex lanes
  are GPT-6.1 Sol since #1412. No iteration has been recorded since 39 (2026-09-07).
