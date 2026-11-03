Committed `9ba6889e0` on `fleet/i79-dfleet26-capacity-transient`. Not pushed, no PR opened. All verification passed; nothing needed adjusting from the directive.

**Diff stat (vs origin/dev, 3 files, +50/−1):**
```
changelogs/unreleased/2026-11-03-codex-capacity-transient-retry.md | 10 +++++++
tools/launchd/mission-control.sh                                   |  6 +++-
tools/launchd/test_controller_capacity.sh                          | 35 ++++++++++++++++++++++
```

**Verification**
- `/bin/bash -n tools/launchd/mission-control.sh` → rc=0
- `/bin/bash tools/launchd/test_controller_capacity.sh` → `controller-capacity: 18 cases passed, fail=0`, rc=0 (baseline was 16/0). The two new cases are `codex-capacity-is-transient` and `prose-codex-capacity-not-transient`.
- **M1 (revert pattern):** `codex-capacity-is-transient` FAILS, the other 17 pass, `controller-capacity: 17 cases passed, fail=1`, rc=1.
- **M2 (de-anchor):** `prose-codex-capacity-not-transient` FAILS, `codex-capacity-is-transient` passes, `controller-capacity: 17 cases passed, fail=1`, rc=1.
- Scratch copies under /tmp were removed.
- `make check-changelog` → rc=0 (2 fragments valid).
- `make test-launchd-drivers` → rc=0. Final lines: `run-suites: all 52 suites passed` and `launchd drivers: tests + bash 3.2 syntax OK`. The capacity suite shows `PASS ... test_controller_capacity.sh (10s, rc=0)`. Log is at `/tmp/fleet_i79_exec_suite.log`.

**Commit:** `9ba6889e00e5ce33abf64d5958d292e54483db0e`
`fix(launchd): codex 'Selected model is at capacity' is transient (D-FLEET-26 A, proposed)`

**Observation for the reviewer:** `TRANSIENT_SIG` is matched with `grep -qiE`, so the new `^ERROR:` anchor is case-insensitive. A line like `error: Selected model is at capacity` at column 0 would also match. The directive asked for that exact pattern, so I left it as written. It's a narrow edge case, but the reviewer should decide whether it matters.
