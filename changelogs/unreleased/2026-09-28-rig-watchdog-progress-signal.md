### Fixed — rig watchdog killed every healthy rotation chunk at 2h

`tools/launchd/rig-watchdog.sh` judged a rotation chunk's progress by the newest
motoko session log in `~/dev/mk-ast`. Only motoko writes those, and motoko left
the rotation on 2026-09-27, so the log only aged and every opencode/pi chunk was
killed as "no-progress" the moment it passed the 2h soft limit (three kills on
2026-09-28). Progress is now the mtime of the rotation's own log
(`/tmp/ailang-os-filler.log`, override `RIG_WATCHDOG_PROGRESS_LOG`), which
eval-suite writes as each run starts and ends, whatever the harness. The stall
threshold default rises from 30 to 100 minutes, because one run may be silent for
its full per-run timeout (25 min, or 90 min for reimplement picks).
