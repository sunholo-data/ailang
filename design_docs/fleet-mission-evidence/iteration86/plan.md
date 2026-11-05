**Sprint plan: transcript scan byte baseline — fleet iteration 86**

Base: `c2fbb9b5886f81328552d7fc28c290f501a8de37`. Planning only; no files changed.

Implement the controller’s byte-offset direction. A pre-existing hit followed by a harmless append must stop generating a fresh alarm. Preserve conservative whole-file scanning when a baseline cannot be used.

**Files and anchors at the base revision**

| Action | Exact path | Anchors |
|---|---|---|
| Modify | `tools/launchd/transcript-secret-scan.sh` | State and cleanup: lines 6–8, 25–53; arguments: 54–74; secrets opened at 75; shared roots: 123–126; enumeration: 130–153; matching: 154–167. |
| Modify | `tools/launchd/test_transcript_secret_scan.sh` | Synthetic isolation: 11–16, 349–362; summary helper: 39–43; disclosure guard: 52–64; driver extraction/setup: 75–118; driver arms: 213–260; termination/timeout: 273–319; malformed-summary test: 337–342; arm registration: 345. |
| Modify | `tools/launchd/mission-control.sh` | Dry-run logging: 2020–2030; marker block: 2394–2402; scan invocation: 2655–2659; strict parser and consistency checks: 2660–2688; notices: 2694–2701; cleanup: 2706–2708. |
| Create | `changelogs/unreleased/2026-11-05-transcript-scan-byte-baseline.md` | Begin with `### Fixed — Transcript secret scan byte baselines`. |

`make/test.mk:76` already registers the suite in `LAUNCHD_SUITES`; no Makefile change is needed. The integration recipe is at lines 131–150. CI explicitly uses macOS and verifies Bash 3.x in `.github/workflows/ci.yml:860–882`; no workflow change is needed.

**Design decisions**

1. **Snapshot mode and format**

   Add `--snapshot-out FILE` and `--baseline FILE`. Move the existing default-root initialization into one shared location before branching into snapshot or scan mode.

   Snapshot mode runs before marker validation or opening `secrets.env`. It requires neither transcript timestamps nor eligible secrets.

   Write consecutive binary pairs:

   ```text
   ASCII decimal byte size NUL pathname bytes NUL
   ```

   Use raw filesystem pathname bytes. Spaces, tabs, backslashes and newlines require no escaping. Record each distinct pathname once; hard links at different paths remain different entries.

   Use one small embedded Python 3 metadata routine, invoked once per snapshot. It walks the supplied roots without following directory symlinks and uses `lstat` to retain regular files only. A pathname set handles overlapping roots. Python receives paths only and never opens transcript contents or the secrets file.

   This avoids GNU-specific flags and approximately 8,000 subprocess launches. Cost is O(files + directories) metadata operations, one Python process, and baseline storage proportional to pathname lengths—no read of the approximately 7.2 GB of transcript content. Metadata timing remains unmeasured here.

   Write `${FILE}.tmp` beside the destination, explicitly mode `0600`, then atomically rename it only after complete success. Suppress raw subprocess exceptions; emit only the scanner’s controlled diagnostics. Snapshot success is quiet and returns 0; failure returns 2. It does not emit a scan verdict.

2. **Baseline loading and lookup**

   Load the baseline once into Bash 3.2 indexed pathname/size arrays using NUL-delimited `read`. Validate complete pairs, nonempty paths, unsigned decimal sizes and arithmetic range. Missing, unreadable, nonregular or malformed input invalidates the entire baseline; never retain a partially loaded prefix.

   Use a linear lookup for each candidate. A repository-independent, synthetic in-memory benchmark on this machine’s Bash 3.2.57 completed 50 worst-position lookups against 8,000 entries in **5.22 seconds**, including array construction. That leaves substantial headroom within the 120-second scan cap without adding a persistent index or another lookup process.

   Add a synthetic scale arm to verify the complete implementation rather than treating that microbenchmark as end-to-end evidence.

3. **Candidate selection and offset reads**

   Retain the existing `find … -type f -newer "$marker" -print0` selection and candidate deduplication.

   For each candidate:

   | Condition | Content matched |
   |---|---|
   | Baseline contains size S; current size ≥ S | Bytes starting at S + 1 |
   | Path absent from baseline | Whole file |
   | Current size < S | Whole file |
   | Baseline unavailable or invalid | Whole file |

   Obtain current size using BSD `stat -f '%z'`; never use GNU `stat -c`.

   For an offset read, run `tail -c "+$((S + 1))"` with input redirected from the candidate into a private scratch file. Check its status before matching. Stage the suffix once per candidate, then retain the existing per-key `grep -F -c -f /dev/fd/3` matching contract. Values stay out of argv and output.

   A failed size query or suffix read is an instrument failure, never a clean result. Track the active reader using the existing child cleanup mechanism. The existing EXIT/TERM cleanup must also remove suffix scratch and unfinished snapshot output.

   Do not add overlap bytes or a historical-content pass.

4. **Summary contract and driver parser**

   Emit exactly one terminal scan summary:

   ```text
   transcript-secret-scan: verdict=clean files=1 keys=1 leaks=0 baseline=8000 since=<escaped-marker>
   ```

   Define `baseline` as:

   - Decimal number: entries successfully loaded; `0` is a valid empty snapshot.
   - `none`: no `--baseline` option.
   - `failed`: an explicitly supplied baseline could not be used.

   Baseline failure alone does not force verdict `instrument-failure`: the whole-file fallback still produces the normal clean/LEAK verdict. Its fallback status remains visible.

   Replace the driver regex at line 2669 in lock-step with:

   ```text
   ^transcript-secret-scan:\ verdict=(clean|LEAK|instrument-failure)\ files=([0-9]+)\ keys=([0-9]+)\ leaks=([0-9]+)\ baseline=([0-9]+|none|failed)\ since=.+$
   ```

   Existing captures 1, 3 and 4 retain their meanings; capture 5 supplies baseline status. Preserve every current consistency check: one terminal summary, no later output, hit count equals `leaks`, and verdict agrees with exit status and eligible-key count. Do not compare baseline inventory count with candidate count.

5. **Driver lifecycle**

   Immediately after successful marker creation, set:

   ```text
   _mc_scan_baseline="${_mc_scan_marker}.baseline"
   ```

   Run scanner snapshot mode synchronously through `_mc_bounded 30`, before controller spawn. Capture failure through an explicit conditional so it cannot disrupt the driver’s result handling.

   On snapshot failure or timeout, remove final/pending baseline files, log a controlled fallback message, and keep `_mc_scan_ready=1`. The post-fire scan must still execute.

   Pass `--baseline "$_mc_scan_baseline"` to the existing `_mc_bounded 120` scan. A missing file therefore becomes `baseline=failed`.

   Extend cleanup at lines 2706–2708 to remove marker, baseline and pending baseline on clean, LEAK, malformed-summary, instrument-failure and timeout paths. Preserve `RC`, `_mc_slot_verdict` and the durable verdict file.

   Update only the existing dry-run log line to describe the bounded snapshot and scan. The dry-run branch still exits before the marker block and invokes no scanner, snapshot or side job.

   Add no environment variables.

6. **Historical hits and notice wording**

   Add a scanner comment and changelog explanation that historical exposures remain parked under D-FLEET-42. No additional “report once” mechanism is warranted: it would introduce persistence and another historical scan for evidence already recorded.

   For a usable baseline, the private notice must say **“written during fire `<id>`’s window”**, with key/path records below it. Concurrent attended sessions remain within that window.

   **Required refinement:** fallback notices must say that exposure was detected after the fire using a whole-file fallback and that write time is unknown. Calling those hits newly written would repeat the attribution defect.

   Keep the public body count-only, using neutral wording such as:

   ```text
   Transcript secret scan after fire <id>: <N> key/file hit(s).
   ```

**Milestones**

| Milestone | Work and acceptance | Estimate |
|---|---|---|
| M1 — Snapshot and byte-window matching | Shared roots; atomic metadata snapshot; baseline validation; offset, new-file, truncation and fallback behaviour; scanner regression arms pass. | 60–90 minutes |
| M2 — Driver integration | Bounded snapshot; new summary grammar; fallback preserved; private/public wording; cleanup; extracted-driver arms pass. | 60–90 minutes |
| M3 — Mutation verification and release note | Kill each listed mutant using disposable copies; scale/termination coverage; fragment; controller integration gate. | 60–90 minutes |

Allow approximately 4–5 hours including review, with roughly 350–500 added/changed implementation and test lines. Iteration 85’s scanner change was 634 inserted lines across five files; this estimate is a smaller follow-up, not a measured daily velocity.

`registry_reuse`: **none** for all milestones. This extends existing launchd shell instrumentation using operating-system metadata; no package-like AILANG capability is introduced.

**Test arms and killing mutations**

Every arm retains the existing per-arm `mktemp -d` synthetic `HOME`, fake `secrets.env`, synthetic `TMPDIR`, `AILANG_STATE_DIR` and trusted base PATH. Create those before invoking any production block. Never inspect real home state to establish isolation.

Extend the disclosure guard to cover snapshot stdout/stderr, binary baseline bytes and captured child argv, as well as existing output/log/notice captures. Preserve guard execution on failed assertions. Report authored assertion labels only.

Each mutation below applies only to a disposable scanner/driver copy.

| Arm | Required observation | One-line mutation that must kill it |
|---|---|---|
| `Snapshot_contract` | Decoded entries exactly equal fixture paths/sizes, including newline/tab names, empty files and overlapping roots; mode is 0600. | Replace NUL pair serialization with newline serialization. |
| `Snapshot_metadata_only` | Snapshot succeeds with an explicitly nonexistent secrets-file argument; fake HOME secrets still exist. | Open `"$secrets_file"` before entering snapshot mode. |
| `Snapshot_no_values` | No protected fake value occurs in baseline or captured output. | Append `cat "$secrets_file" >> "$snapshot_tmp"` before publication. |
| `Baseline_old_append` | Old secret-bearing content, snapshot, harmless append: rc 0, clean, one candidate, no hit. | Set the matched candidate’s offset to `0`. |
| `Baseline_appended_leak` | Snapshot existing harmless content, append fake value: rc 1 and expected key/path. | Set offset to current file size instead of saved size. |
| `Baseline_new_file` | Create a leaking file after snapshot: rc 1 and expected hit. | `continue` when baseline lookup misses. |
| `Baseline_truncated` | Baseline a larger file; replace it with a shorter leaking file: rc 1. | `continue` when current size is less than saved size. |
| `Baseline_empty_suffix` | A selected file with current size exactly S supplies zero bytes to the matcher and stays clean. | Start `tail` at S instead of S + 1. |
| `Baseline_none` | Omit baseline; existing leaking content is scanned whole; summary has `baseline=none`. | Treat an omitted baseline as “skip matching.” |
| `Baseline_missing` | Supply absent baseline path; whole-file LEAK and `baseline=failed`. | Exit before scanning when baseline open fails. |
| `Baseline_unreadable` | Supply a directory as baseline, avoiding permission assumptions; whole-file LEAK and `failed`. | Accept a directory as a valid empty baseline. |
| `Baseline_malformed` | Truncated pair and invalid-size fixtures discard all loaded entries; whole-file LEAK and `failed`. | Retain the valid prefix after a baseline parse error. |
| `Offset_read_failure` | Injected failing `tail` produces rc 2/instrument failure, never clean. | Treat nonzero suffix-reader status as an empty successful suffix. |
| `Secret_free_argv` — extend existing | Exercise offset scanning; protected values remain absent from stdout/stderr and argv; `/dev/fd/3` is observed. | Pass the pattern as a grep argument instead of using `-f /dev/fd/3`. |
| `Termination_cleanup` — extend existing | TERM during suffix reading kills the reader and removes scratch/pending output. | Remove active-child termination from `cleanup()`. |
| `Baseline_scale` | 8,000 baseline entries and 50 fresh absent paths yield the exact clean counts within a test deadline of 30 seconds. | Replace the reported baseline count with `0`. |
| `Driver_marker` — extend existing | Baseline exists before spawn, uses resolved state root, is 0600, and the snapshot call requests 30 seconds. | Remove the snapshot invocation. |
| `Driver_clean` — extend existing | A genuine scanner summary with numeric baseline is logged without instrument failure. | Restore the old driver summary regex. |
| `Driver_old_append` | Seed the old hit before the production marker/snapshot block, append harmless content afterward; no notice. | Remove `--baseline` from the production scan invocation. |
| `Driver_snapshot_failure` | Failed snapshot still invokes the real scanner; a fresh fake leak is noticed with `baseline=failed`. | Set `_mc_scan_ready=0` on snapshot failure. |
| `Driver_snapshot_timeout` | Real bounded helper, test-only short deadline: timed-out snapshot leaves no usable partial baseline; subsequent scan still runs. | Skip the post-fire scan after snapshot rc 124. |
| `Driver_parser_invalid` | Reject missing/invalid baseline token, duplicate/nonterminal summary, count mismatch and rc/verdict mismatch. | Set `_mc_scan_valid=1` immediately before final validation dispatch. |
| `Driver_leak` — extend existing | Private message contains window wording and key/path records; public body contains count only. | Replace the private window wording with the former fire-attribution wording. |
| `Driver_fallback_notice` | Whole-file fallback notice explicitly states write time is unknown. | Unconditionally select the window-scoped notice text. |
| `Driver_public_privacy` | Public body has no key name or transcript path. | Pass private hit lines as the fourth `_mc_exit_notice` argument. |
| `Driver_cleanup` | Marker, baseline and pending file are absent after clean, leak, failure and timeout; controller result remains unchanged. | Omit baseline removal from driver cleanup. |
| `Driver_dry_run` — extend existing | Logs snapshot/scan explanation; creates no artifacts and invokes no bounded operation; extracted block remains log-only. | Add `: > "$_mc_scan_baseline"` to the dry-run block. |

Update `summary()` with an optional expected baseline argument, defaulting to `none`, so existing arms assert the new exact line. Extend `setup_driver()` and its bounded stub to distinguish snapshot’s 30-second deadline from scan’s 120-second deadline. Register every new arm in the existing selector list.

For timeout arms, reuse the suite’s extraction of the real `_mc_bounded` implementation from `tools/launchd/lib/lane-probe.sh:23–55`. Do not introduce GNU `timeout`.

**Gate commands and base evidence**

| Exact command | Owner | Result at base |
|---|---|---|
| `/bin/bash -n tools/launchd/transcript-secret-scan.sh` | Executor | PASS, rc 0 |
| `/bin/bash -n tools/launchd/test_transcript_secret_scan.sh` | Executor | PASS, rc 0 |
| `/bin/bash -n tools/launchd/mission-control.sh` | Executor | PASS, rc 0 |
| `shellcheck --shell=bash tools/launchd/transcript-secret-scan.sh tools/launchd/test_transcript_secret_scan.sh` | Executor | PASS, rc 0 |
| `/bin/bash tools/launchd/test_transcript_secret_scan.sh` | Executor | Present and runnable at base; **UNINFORMATIVE UNDER SANDBOX**—not run because it writes fixtures. |
| `make test-launchd-drivers` | Controller | Existing target includes this suite; **UNINFORMATIVE UNDER SANDBOX**—not run because it writes fixtures and runs integration suites. |
| `git diff --check` | Executor/controller | PASS, rc 0 at unchanged base |

After implementation, the executor runs the syntax checks, ShellCheck and targeted suite, then checks the fragment with `/bin/bash scripts/check_changelog.sh`.

Use the existing optional arm selector for mutation checks:

```bash
/bin/bash tools/launchd/test_transcript_secret_scan.sh \
  "$lab/mutant-scanner.sh" "$lab/mutant-driver.sh" Baseline_old_append
```

The controller reruns the targeted suite and integration target with an outer synthetic home:

```bash
/bin/bash -c '
  lab=$(mktemp -d /tmp/i86-controller.XXXXXX) || exit 2
  trap '\''rm -rf "$lab"'\'' EXIT
  mkdir -p "$lab/home" "$lab/tmp" "$lab/state"
  HOME="$lab/home" TMPDIR="$lab/tmp" AILANG_STATE_DIR="$lab/state" \
    make test-launchd-drivers
'
```

No live-driver dry-run, real transcript reproduction or historical scan is required. Completion requires runtime passes outside this sandbox and recorded mutation failures, not merely syntax passes.

**Risks and explicit limits**

- Snapshot collection is not atomic across the tree: a concurrent attended append between marker creation and that file’s size measurement can be absorbed into the baseline. Controller spawn occurs afterward, protecting its own writes from this startup race; exact marker-time coverage of concurrent sessions remains approximate.
- Same-size/larger rewrites cannot be distinguished from appends using pathname and size alone; truncation observed below S receives the requested conservative whole-file scan.
- A value straddling S could be missed; whole-record JSONL appends make this an accepted residual risk, and overlap would recreate historical false positives.
- Python 3 is an optional snapshot dependency; absence or failure must produce explicit whole-file fallback. Current-size checks use BSD `stat`, and all shell paths remain Bash 3.2 compatible.
- Truncation and baseline-failure fallbacks may still report historical content. That conservatism is deliberate; notices must avoid unconditional new-write attribution.
- I agree with deferring per-session attribution: controller IDs are unavailable for some lanes, and restricting roots would change coverage.
- Baseline failure may restore expensive whole-file reads and encounter the existing 120-second limit. It must remain an explicit fallback or instrument failure, never silence.
- D-FLEET-42 remains Mark’s historical-exposure decision. This sprint neither reopens that scan nor changes credential inheritance.