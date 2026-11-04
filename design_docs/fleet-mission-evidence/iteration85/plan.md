# Sprint plan: M-TRANSCRIPT-SECRET-SCAN

Base: `93ac5a915b1826ddb44d39e44c9265e429110709`  
**One milestone, one commit.** Estimate: 75–90 minutes, approximately 500–650 lines including tests.

Implement the post-fire detector only. Spawn environment changes remain parked under HD-2a. Historical transcript redaction and credential rotation are outside this commit.

## Files and driver anchors

| File | Planned change |
|---|---|
| `tools/launchd/transcript-secret-scan.sh` | Create executable scanner. |
| `tools/launchd/test_transcript_secret_scan.sh` | Create synthetic scanner and driver integration tests. |
| `tools/launchd/mission-control.sh` | Add marker creation, dry-run explanation, and one bounded post-fire scan. |
| `make/test.mk` | Add the new suite to `LAUNCHD_SUITES`, currently lines 75–126. |
| `changelogs/unreleased/2026-11-04-transcript-secret-scan.md` | Create fragment beginning `### Added — Transcript secret detection`. |

Driver line numbers below refer to the base commit:

| Anchor | Wiring |
|---|---|
| `STATE_DIR`, line 83; `log()`, line 123 | Reuse existing state and logging conventions. |
| `_mc_exit_notice()`, lines 359–384 | Reuse once per leaking fire; no new notification mechanism. |
| Dry-run branch, lines 2020–2028 | Before its exit, explain that a real fire would create a marker and scan afterward with a 120-second limit. Create no scanner artifacts and send no notice. |
| `MISSION_FIRE_ID`, line 2292 | Reuse this identifier; introduce no new environment configuration. |
| `_mc_slot_state`, lines 2385–2389 | Immediately afterward, create the per-fire marker under this resolved state root, before any controller spawn. |
| `_mc_run_once()`, lines 2410–2540; spawn branches, lines 2423–2488 | Leave controller launch and wait behavior intact. |
| Retry loop, lines 2552–2601 | Keep the same marker across every attempt. |
| `SLOT VERDICT END`, line 2638; PID removal, line 2640 | Insert the scan between these anchors. The verdict is already recorded, and the overlap PID remains held during scanning. |
| Final `exit "$RC"`, line 2697 | Preserve the controller’s original exit status. |

Reuse `_mc_bounded()` from `tools/launchd/lib/lane-probe.sh:23–55`. It already implements background execution with a `date +%s` deadline, bounded termination, and captured output without GNU `timeout`.

No edits are needed to `tools/launchd/lib/run-suites.sh`: `make/test.mk:148` already passes registered suites through that runner and `suite-env.sh`.

Registry reuse: **none**. This is shell harness instrumentation using existing driver helpers; it introduces no package capability or dependency.

## Scanner design

### Interface and parsing

Implement exactly these options:

- `--secrets-file F`, default `$HOME/.config/ailang/secrets.env`
- Required `--since FILE`
- Repeatable `--root DIR`; explicit roots replace the default root set
- `--min-len N`, default `20`, validated as a positive decimal integer

Default roots:

```text
$HOME/.codex-missions/sessions
$HOME/.codex/sessions
$HOME/.pi/agent/sessions
$HOME/.claude/projects
```

Read assignments using Bash builtins and `IFS= read -r`, including a final unterminated line. Accept optional `export`, validate shell identifier key names, split at the first `=`, and remove one matching surrounding quote pair. Skip blank lines and comments. Preserve embedded `=`, spaces inside values, and backslashes. Never source, evaluate, interpolate, or execute assignment contents.

Use indexed arrays compatible with Bash 3.2. Duplicate assignments use the last value for that key. Count only eligible distinct keys after length filtering. **Zero eligible keys is instrument failure**, including when all values are too short.

Disable tracing before reading secrets. Diagnostics must never include assignment text or values.

### Pattern delivery and JSON escaping

Choose **a here-string on a dedicated descriptor**:

```bash
grep -F -c -f /dev/fd/3 -- "$file" 3<<< "$patterns" >/dev/null
```

Here, `patterns` contains variants for one key. Values remain shell data and descriptor input; they never become arguments to an external process. Bash’s here-string storage avoids an application-managed named pattern file surviving forced termination.

Use `grep -c` with discarded stdout, rather than printing matching lines. Inspect its status explicitly: `0` means match, `1` means absence, and other statuses mean instrument failure. Suppress raw subprocess diagnostics and replace them with safe operation/path diagnostics.

Include these fixed-string patterns, deduplicated per key:

1. Literal value.
2. JSON string-body representation, escaping backslashes, double quotes, and standard control characters.
3. The same JSON representation with `/` encoded as `\/`.

Use Bash substitutions to construct variants; never pass a value to an external encoder through argv. This covers common JSONL serialization differences without adding a JSON parser. Arbitrary Unicode escaping, double encoding, base64, and other transformations remain outside detection coverage.

Report a key/file pair once, even if several variants or repeated occurrences match.

### File selection and failures

Enumerate with:

```bash
find "$root" -type f -newer "$marker" -print0
```

Do not use `-newermt`, filename-extension filters, GNU utilities, or line-delimited pathname iteration. Capture enumeration status explicitly; avoid process substitution that hides a failed `find`. Deduplicate repeated candidate paths from overlapping roots.

Missing roots are skipped with a safe diagnostic naming the root. Existing roots that cannot be traversed, files that cannot be scanned, failed temporary-file operations, or failed matching commands produce instrument failure.

Temporary candidate lists contain paths only. Use a private temporary directory and cleanup traps. Track active `find`/`grep` children: run them in the background and wait so TERM/INT traps can terminate them promptly. This makes the existing driver timeout effective for descendants as well as the scanner shell.

Emit:

```text
LEAK key=<NAME> file=<path>
transcript-secret-scan: verdict=<clean|LEAK|instrument-failure> files=<N> keys=<K> leaks=<L> since=<marker>
```

Escape pathname control characters so each record occupies one line; ordinary paths, including spaces, remain recognizable.

Exit statuses:

- `0`: complete scan, no hits
- `1`: complete scan, one or more key/file hits
- `2`: instrument failure, including unreadable secrets, missing marker, or zero eligible keys

Instrument failure takes precedence over clean or partial results. Emit exactly one terminal summary whenever normal execution or a handled signal permits it.

## Driver behavior

Create a marker such as:

```text
$_mc_slot_state/mission-${MISSION_NAME}-transcript-scan-${MISSION_FIRE_ID}.since
```

Create it once before attempts begin. Marker creation failure logs `INSTRUMENT FAILURE`, skips scanning, and leaves execution unchanged.

After recording the slot verdict, invoke:

```bash
_mc_bounded 120 /bin/bash \
  "$MC_DRIVER_ROOT/tools/launchd/transcript-secret-scan.sh" \
  --since "$_mc_scan_marker"
```

Use a separate scan status variable; never assign scanner status to `RC`.

- **rc 0:** log the scanner summary.
- **rc 1:** log the summary and call `_mc_exit_notice` exactly once. Both notice bodies contain only the scanner’s key-name/path hit records. Include `MISSION_FIRE_ID` in the title to prevent a previous fire’s duplicate-title suppression from hiding a new incident.
- **rc 2:** log the summary and explicit `INSTRUMENT FAILURE`.
- **Timeout, unexpected status, or missing/malformed terminal summary:** log `INSTRUMENT FAILURE`; never infer clean.

Remove the marker afterward on all completed scan paths. Keep retries, routing, controller status, and the already-written verdict unchanged.

Add clearly delimited marker, scan, and dry-run blocks for functional extraction by tests. Avoid a global EXIT trap that could interfere with existing exit paths.

## Tests and killing mutations

Every arm runs inside a fresh `mktemp -d` fixture with synthetic `HOME`, `TMPDIR`, state directories, fake secrets, and fake transcripts. Set these before invoking code under test. Never execute the full production driver merely to source its functions.

Use the repository’s existing extraction approach from `test_driver_notify.sh` and `test_mission_heartbeat.sh`: extract real production blocks, require nonempty extraction, and execute them with stubbed logging/notices.

Capture stdout, stderr, driver logs, and fake notices. **Every arm checks that none contains any fixture secret or encoded variant.** On assertion failure, print only the arm name and failed condition—never dump captured output.

| Arm | Required assertion | One-line killing mutation |
|---|---|---|
| Clean | rc0; exact clean summary and counts | Change clean exit from `0` to `1`. |
| Fresh leak | rc1; exact key name/path and one hit | Delete the hit-emission statement. |
| Literal metacharacters | A fake value containing regex metacharacters matches literally | Change `grep -F` to `grep -E`. |
| Older leak | Old leaking file excluded; fresh clean file counted | Remove `-newer "$marker"`. |
| Marker boundary | Equal-mtime file excluded | Admit files with mtime equal to marker. |
| Default minimum | Short value ignored while another eligible key is loaded | Remove the minimum-length filter. |
| Explicit minimum | Override changes which key is eligible | Hardcode minimum to `20`. |
| Assignment forms | Plain, single-quoted, and exported double-quoted keys all detected | Remove surrounding-quote stripping. |
| Export prefix | Exported assignment produces its key name | Remove export-prefix handling. |
| No execution | Shell-looking value remains literal; sentinel file absent | Replace literal parsing with `eval`. |
| Missing secrets | rc2 and instrument-failure summary | Return `0` on secrets-open failure. |
| Missing marker | rc2; no enumeration | Remove marker validation. |
| Zero eligible keys | rc2, never clean | Remove the zero-key refusal. |
| Missing root | Missing root named; another root still scanned | Remove the missing-root diagnostic. |
| Repeated roots | A leak in the second explicit root is detected | Retain only the first `--root`. |
| Default roots | Fixtures in all four synthetic default roots are scanned | Remove one default-root entry. |
| JSON escaping | JSON-escaped backslash/quote value detected | Remove JSON pattern generation. |
| Escaped slash | Value present only with `\/` detected | Remove slash-escaped variant. |
| Deduplication | Repeated roots/occurrences produce one key/file hit | Remove candidate-path deduplication. |
| Enumeration failure | Stubbed `find` failure produces rc2 | Ignore `find` status. |
| Matcher failure | Stubbed grep rc2 produces rc2 | Treat all nonzero grep statuses as no match. |
| Secret-free argv | Synthetic grep spy observes no value in its arguments | Replace descriptor patterns with `grep -F "$value"`. |
| Termination cleanup | Scanner and active child terminate; scratch files removed | Remove child termination from the TERM trap. |
| Driver marker | Correct state override and fire namespace; marker predates synthetic session write | Create marker after the session write. |
| Driver leak | One notice contains every expected key/path; `RC` and verdict unchanged | Assign scanner status to `RC`. |
| Driver clean | No notice; marker removed | Make notice unconditional. |
| Driver failure | rc2 logs `INSTRUMENT FAILURE`; no clean report or notice | Map rc2 to clean. |
| Driver timeout | Real extracted `_mc_bounded` cuts off a synthetic hanging scanner; controller result preserved | Replace `_mc_bounded` with a direct scanner call. |
| Driver dry-run | Explains intended scan; no scanner invocation, marker, or notice | Remove the dry-run explanation. |

For the timeout arm, pass a short deadline to the extracted bounded helper through a test-local function argument; retain the literal production call of `120`. Introduce no production timeout environment variable.

Allow optional **test-only positional paths** for scanner/driver copies so mutations can be exercised in disposable fixture trees without editing production files.

## Gates and base evidence

Executor commands after implementation:

```bash
/bin/bash -n tools/launchd/transcript-secret-scan.sh
/bin/bash -n tools/launchd/test_transcript_secret_scan.sh
/bin/bash -n tools/launchd/mission-control.sh

shellcheck --shell=bash \
  tools/launchd/transcript-secret-scan.sh \
  tools/launchd/test_transcript_secret_scan.sh

/bin/bash tools/launchd/test_transcript_secret_scan.sh

/bin/bash tools/launchd/lib/run-suites.sh 1 \
  tools/launchd/test_transcript_secret_scan.sh

/bin/bash scripts/check_changelog.sh
```

Exercise the listed mutations against disposable copies; each applicable arm must turn red. Verify both new scripts have executable mode.

Controller integration gate:

```bash
make test-launchd-drivers
```

| Gate | Evidence at base |
|---|---|
| Bash compatibility | `/bin/bash` is **3.2.57**. |
| Existing launchd syntax | **Passed:** all `tools/launchd/*.sh` and `tools/launchd/lib/*.sh`, plus inspected existing harnesses. |
| New scanner/test syntax and runtime | Files do not exist at base. **UNINFORMATIVE UNDER SANDBOX**; execute after implementation. |
| ShellCheck | Version **0.11.0** available. New-file checks are **UNINFORMATIVE UNDER SANDBOX** until files exist. Existing mandatory ShellCheck scope is auto-push scripts, not the whole launchd driver. |
| Narrow suite runner | Exists and passed syntax validation. Runtime gate **UNINFORMATIVE UNDER SANDBOX** because it requires writes. |
| Changelog validation | **Passed** at base, including fragment validation. |
| `make test-launchd-drivers` | Target and runner wiring confirmed. **UNINFORMATIVE UNDER SANDBOX**: executable suite not run. `make -n` resolved the recipe but emitted denied macOS cache-write diagnostics; that is not a test pass. |

No Go, compiler, stdlib, CI, or full `make test` gate is needed for this shell-only change.

## Risks and scope limits

- **Coverage gap at base:** `mission-control.sh:2100–2108` directly `exec`s `ailang mission iterate` when `AILANG_MISSION_WORK_ITEM` is set. It bypasses the requested legacy-controller insertion point. Preserve that behavior in this commit and explicitly avoid claiming binary-iteration coverage.
- `mtime > marker` selects recently modified files, not files uniquely attributable to this fire. Concurrent sessions can be included, and old contents in a newly modified transcript can trigger detection.
- Traversal still visits directory metadata across the roots. The 120-second bound limits cost; timeout means instrumentation failure.
- Literal assignment parsing cannot discover values produced by shell expansion or multiline execution. This detector also does not establish that excluded or transformed secrets are absent.
- This detects exposure after it happens. It does not redact transcripts, rotate credentials, or prevent environment inspection.
- Reach follows the committed shared driver pin. No installation, reload, external fork edit, or live transcript scan belongs in this implementation commit.

The milestone is complete when the detector, nonfatal wiring, registered synthetic suite, mutation checks, and changelog fragment pass the stated gates in **one commit**.