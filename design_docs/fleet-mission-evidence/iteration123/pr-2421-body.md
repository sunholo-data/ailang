Fleet mission iteration 123. Ticket: `gate0:self-notices-rc2-no-watermark-file` (filed by stapledon iteration 99, 2026-11-18).

## Problem
Gate 0 step 6a (`scripts/mission_gate0_self_notices.sh`) exited 2 (instrument floor) whenever every `--watermark-file` was absent. Since #981 (2026-10-26) `mission_directives.sh` creates `mission-<issue>-last-seen` only on `--mark`, i.e. only when a human directive is processed, so on most missions both files are absent and the died-fire read was blind on every fire. The fleet itself hit this for 26 consecutive fires, and the same was measured on stapledon (#450/#410).

## Fix
- When **every** given watermark file is absent, the script does a loudly-labelled `DEGRADED` read of the whole of each issue (since `1970-01-01T00:00:00Z`, matching how `mission_directives.sh` reads an absent file), and exits 0 or 1 by hits.
- A file that exists but is unreadable, empty or invalid is still rc 2 (F10).
- The mixed present/absent path and the `--since` path are unchanged.
- The wider window can only over-report, never drop a notice. Repeat crash notices are absorbed by step 6a's existing "credit once per `url=`" rule, which is now documented in both skill copies.

## Evidence
- `scripts/test_mission_gate0_self_notices.sh`: 168 → **176 passed, 0 failed**. The old F9 arm is rewritten as an all-absent arm.
- Agents-skills sync check: 7/7. `make test-launchd-drivers`: 58/58, rc 0.
- Live, read-only:
  - fleet (#2341/#2172): base rc 2, now rc 0 with DEGRADED.
  - stapledon (#450/#410): now rc 1, surfacing a 2026-11-13 crash notice on #410 that the floor had hidden.
- Independent judge (sonnet): **PASS 91**, 0 blocking, 11/11 mutants killed. Controller re-ran the restore-`exit 2` mutant itself: 168 passed / 8 failed.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
