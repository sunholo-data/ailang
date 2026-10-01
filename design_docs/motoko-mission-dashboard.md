# Mission Dashboard — Motoko

*Snapshot, overwritten every iteration. History lives in the charter STATUS stamps and the log.*
**Last refreshed**: 2026-10-01 (attended queue grooming, `D-MOTOKO-GROOM-1` — no iteration has run since 39)

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

Groomed 2026-10-01 (`D-MOTOKO-GROOM-1`): every row re-checked against motoko main; only motoko
product work stays here.

1. **Row 23** [NEXT] — KPI instrument: paired motoko vs pi vs opencode (local qwen3.8 + one cloud model).
   First comparable window started 2026-09-30 10:00; early numbers (9 shared benches): motoko 8/9,
   pi 8/9, opencode 7/9 — too small to call.
2. **Row 22a** — `context_limit` resolves to 0 in eval workspaces; compaction and budgets run blind.
3. **Row 24** — `ailang_tools` on a set with headroom (local qwen3.8).
4. **Row 27** — upstream hygiene: 12 carried commits → only what must stay ours (#200, the dead
   `ai_options_json`, rig-lease forwarding, a #198 duplicate, the LOCAL-ONLY commit).
5. **Row 20** [PARKED on the next release] — cloud motoko executor verified in test. The dev image
   already builds motoko main (`4d4917cd`, verified in the dev Cloud Build log).
6. **Row 25** — executor-lane design + gate trial (World first).

Parked: 22b (fmt, only if row 24 shows extensions help), 9 (R3, after row 23).
Moved out as fleet tickets: 6t, 17/18, 19, and the orphaned 6s probe. To the AILANG backlog
(PROGRAM.md §5c): 6m and the 96,908-char eval teaching prompt.

## Parked on Mark — 1 open decision

- **`D-MOTOKO-RESTART-1`** — when to restart. Restart needs Mark's say-so to remove
  `~/.ailang/state/mission-motoko.disabled`; the loop then runs on the **shell driver**
  (`tools/launchd/mission-control.sh`, `MISSION_PROFILE=motoko`) and its first fire is iteration 40
  on row 23. No default: nothing times out into a restart.

## Blocked, not waiting on Mark

- Row 20 needs a release + promote to reach test (release cadence is outside the loop).
- Row 21 waits on the upstream maintainer's review of #200 (bounded: 14 days, then move on).

## Loop health

- Kill switch present (measured 2026-09-30). Plist repo copy: `StartInterval=46800` (13h).
- Routing: resolved live by the shell driver (`MISSION_PROFILE=motoko MISSION_DRY_RUN=1`); codex lanes
  are GPT-6.1 Sol since #1412. No iteration has been recorded since 39 (2026-09-07).
