Implemented all planned work. **52/52 arms pass; all 27 named mutations plus one additional mutation produce PASS→FAIL.** No git writes.

| File | Final lines | Added / removed |
|---|---:|---:|
| [transcript-secret-scan.sh](tools/launchd/transcript-secret-scan.sh) | 290 | +124 / −8 |
| [mission-control.sh](tools/launchd/mission-control.sh) | 2,787 | +25 / −8 |
| [test_transcript_secret_scan.sh](tools/launchd/test_transcript_secret_scan.sh) | 626 | +268 / −16 |
| [Changelog fragment](changelogs/unreleased/2026-11-05-transcript-scan-byte-baseline.md) | 7 | +7 / −0 |

The scanner publishes atomic mode-0600 metadata snapshots, validates baselines completely, and matches appended bytes. Missing or invalid baselines retain visible whole-file fallback. The driver takes the bounded snapshot before spawning, parses the new summary grammar, preserves its recorded result, and removes scan artifacts. Public notices remain count-only.

Two refinements were necessary:

- A numeric baseline can still trigger truncation fallback. Private notices therefore use “write time is unknown” for that case too.
- Cleanup assertions check scanner-owned artifacts; Apple’s Python launcher creates an unrelated `xcrun_db` inside synthetic TMPDIR.

The suffix reader runs directly so cleanup tracks and terminates its actual PID. No environment variables were added.

Every row below passed at head and failed with the listed mutation on disposable copies; **each mutant returned rc 1**.

| PASS at head | Killing mutation | Result |
|---|---|---|
| Snapshot_contract | Replace NUL serialization with newlines | PASS→FAIL |
| Snapshot_metadata_only | Open secrets before snapshot branch | PASS→FAIL |
| Snapshot_no_values | Append secrets to snapshot | PASS→FAIL |
| Baseline_old_append | Set matched offset to zero | PASS→FAIL |
| Baseline_appended_leak | Use current size as offset | PASS→FAIL |
| Baseline_new_file | Skip paths absent from baseline | PASS→FAIL |
| Baseline_truncated | Skip truncated files | PASS→FAIL |
| Baseline_empty_suffix | Start tail at S | PASS→FAIL |
| Baseline_none | Skip matching without baseline | PASS→FAIL |
| Baseline_missing | Exit before fallback | PASS→FAIL |
| Baseline_unreadable | Accept directory as empty baseline | PASS→FAIL |
| Baseline_malformed | Retain valid prefix | PASS→FAIL |
| Offset_read_failure | Ignore reader failure | PASS→FAIL |
| Size_read_failure | Omit failure flag after failed stat | PASS→FAIL |
| Secret_free_argv | Pass pattern in grep argv | PASS→FAIL |
| Termination_cleanup | Remove active-child termination | PASS→FAIL |
| Baseline_scale | Report zero inventory entries | PASS→FAIL |
| Driver_marker | Remove snapshot invocation | PASS→FAIL |
| Driver_clean | Restore old summary regex | PASS→FAIL |
| Driver_old_append | Remove baseline argument | PASS→FAIL |
| Driver_snapshot_failure | Disable subsequent scan | PASS→FAIL |
| Driver_snapshot_timeout | Disable scan after rc 124 | PASS→FAIL |
| Driver_parser_invalid | Force parser validity | PASS→FAIL |
| Driver_leak | Restore former attribution wording | PASS→FAIL |
| Driver_fallback_notice | Always claim window-scoped writes | PASS→FAIL |
| Driver_public_privacy | Publish private hit records | PASS→FAIL |
| Driver_cleanup | Remove only marker | PASS→FAIL |
| Driver_dry_run | Create baseline artifact | PASS→FAIL |

The remaining 24 arms all passed. The plan specified no named mutation for these:

```text
PASS Clean
PASS Fresh_leak
PASS Literal_metacharacters
PASS Older_leak
PASS Marker_boundary
PASS Default_minimum
PASS Explicit_minimum
PASS Assignment_forms
PASS Export_prefix
PASS No_execution
PASS Missing_secrets
PASS Missing_marker
PASS Zero_eligible_keys
PASS Missing_root
PASS Repeated_roots
PASS Default_roots
PASS JSON_escaping
PASS Escaped_slash
PASS Deduplication
PASS Enumeration_failure
PASS Matcher_failure
PASS Driver_failure
PASS Driver_timeout
PASS Additional_contracts
```

Final gates ran against the frozen implementation with synthetic HOME/TMPDIR/AILANG_STATE_DIR and fake secrets:

| Gate | rc |
|---|---:|
| `/bin/bash -n tools/launchd/transcript-secret-scan.sh` | 0 |
| `/bin/bash -n tools/launchd/test_transcript_secret_scan.sh` | 0 |
| `/bin/bash -n tools/launchd/mission-control.sh` | 0 |
| `shellcheck --shell=bash tools/launchd/transcript-secret-scan.sh tools/launchd/test_transcript_secret_scan.sh` | 0 |
| `/bin/bash tools/launchd/test_transcript_secret_scan.sh` | 0 |
| `/bin/bash scripts/check_changelog.sh` | 0 |
| `git diff --check` | 0 |

No implementation or planned arms remain unfinished. The controller’s outside-sandbox reruns and `make test-launchd-drivers` remain outstanding.