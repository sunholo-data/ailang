Plan for fleet iteration 87: extend Gate 0’s self-notice reader to report transcript-scan notices while preserving exit codes 0/1/2. This is a mechanical detection/reporting fix. No driver, routing, billing, or approval-policy change is needed.

Base verified: `f6a95d75da9845c13adc9c052f81499f314ab66b`; working tree clean. No files were changed.

**Files and base anchors**

All line references below are at `f6a95d75d`.

| File | Anchors and planned changes |
|---|---|
| `scripts/mission_gate0_self_notices.sh` | Lines 2–12: header contract. `usage()`, lines 23–42: describe both classes and informational transcript reporting. Signature definitions and `parse_comments()`, lines 59–111: add anchored transcript recognition. F8, lines 212–220: require the current transcript prefix when `--driver-src` is supplied. Classification/reporting, lines 318–389: add transcript rows, counters, arithmetic, and verdict suffixes. |
| `scripts/test_mission_gate0_self_notices.sh` | Helpers, lines 25–53: source the companion after helper definitions. `arm_signature_source()`, lines 363–392: update positive source fixture and preserve independent crash-signature coverage. `arm_report()`, lines 395–430: update exact verdict expectations. Snapshot assertions at lines 449, 469, 714: add `transcript=0`. `arm_driver_literal()`, lines 627–644: check the current transcript contract too. Dispatcher/default list, lines 718–774: register companion arms. Update stale header wording at lines 2–14. |
| **Create** `scripts/test_mission_gate0_self_notices_transcript.sh` | Sourced companion containing transcript test functions, literal per-arm fixture heredocs, and a transcript-arm dispatcher/list. Reuse the parent’s `make_json`, `make_stub_gh`, `run_instr`, scratch directory, and assertion counters. |
| `.claude/skills/mission-control/resources/gate-0-preflight.md` | Step 6a, lines 256–271: broaden its heading/introduction and add the short disposition below. Preserve the invocation and rc contract. |
| `.agents/skills/mission-control/resources/gate-0-preflight.md` | Regenerate from `.claude` using `tools/launchd/sync-agents-skills.sh`; never hand-edit this mirror. |
| **Create** `changelogs/unreleased/2026-11-05-gate0-self-notices-transcript-count.md` | Follow `changelogs/unreleased/README.md`: open with `### Fixed — Gate 0 reports transcript-secret scan notices (2026-11-05)`, followed by a short explanation linking the related work in #2127. |

The parent test file is already 778 lines. Keep new test functions in the companion and keep both files below 800 lines. Existing captured fixtures under `scripts/mission_gate0_self_notices_testdata/` remain unchanged; all three captures contain zero transcript-prefix matches.

Reference-only files:

- `tools/launchd/mission-control.sh:359–377`: `_mc_exit_notice` sends argument **3** privately and argument **4** to the public issue.
- `tools/launchd/mission-control.sh:2708–2718`: privacy boundary, fallback qualification, and current public wording.
- `tools/launchd/sync-agents-skills.sh:21–29`: regeneration covers mission and sprint skills.
- `make/test.mk:93`: existing suite registration already reaches the sourced companion; no makefile change is needed.
- `tools/launchd/test_quorum_artifacts.sh:298–301`: mirror consistency remains a required check.

**Decisions on items 1–7**

| Item | Decision and reason |
|---|---|
| **1. Recognition/output** | **Accept, refine output to include fire ID.** It is public metadata needed for the iteration report; extracting it avoids requiring controllers to dump comment bodies. |
| **2. Table/windowing** | **Accept.** Count transcript notices separately, use the existing strictly-after-`SINCE` window, and compute `other = self − crash − transcript`. |
| **3. Exit codes** | **Accept.** Transcript notices alone retain rc 0; both rc-0 and rc-1 verdicts include the in-window transcript count. No rc 3. |
| **4. F8 source control** | **Accept.** Require the crash signature and current transcript prefix independently, with distinct diagnostics. The old transcript prefix remains comment-only compatibility. |
| **5. Positive control** | **Accept.** Retain the existing all-time crash controls. Fixture coverage and F8 protect transcript recognition; a changing bookkeeping issue is unsuitable as a fixed-count live control. |
| **6. Documentation** | **Accept with a factual qualification.** The requested unconditional “written during that fire” claim conflicts with the driver’s explicit whole-file fallback, whose write time is unknown. Preserve that distinction in four sentences. |
| **7. Changelog** | **Accept.** Use the requested dated fragment and required `### Fixed — ...` format. |

Implement recognition inside the existing Python block. Define the old/current prefixes once in Bash, pass them into Python, and construct an escaped expression equivalent to:

```text
\A(?:<old-prefix>|<current-prefix>)(\S+): ([0-9]+) key/file hit\(s\)\.
```

Use literal prefix matching, ASCII digits, and the literal terminal period. Trailing body text is ignored. Apply the existing case-insensitive self-author filter. Keep crash parsing unchanged and the classes mutually exclusive.

Emit this fixed-field internal record:

```text
TRANSCRIPT <createdAt> <hits> <url-or-dash> <fire-id>
```

Produce this public row:

```text
transcript: issue=<n> at=<ts> hits=<N> in-window=<yes|no> url=<u> fire=<id>
```

All fields remain whitespace-free. Read the new record with Bash 3.2’s `read -r` into fixed fields rather than allowing pathname expansion through `set -- $line`; leave existing crash/count parsing alone.

Track **notice count**, independently of each notice’s hit count. Add `transcript=<k>` immediately after `crash=<k>` in each issue row. Keep control counting restricted to `CRASH` records, and never include a control-only issue in the windowed verdict.

Append an explicit transcript count to both verdict branches. When positive, also say:

```text
exposure reported: key names are in the private controlplane notice; report the count and fire id, never a value or path
```

Keep rc 1’s “A FIRE DIED”, newest-crash metadata, and recovery instructions intact. Transcript timestamps must never update newest-crash metadata.

Proposed four-sentence Step 6a disposition:

> An in-window transcript-scan notice reports exposure detected after the named fire: normally a write during that fire’s window, but the private controlplane notice identifies whole-file fallbacks whose write time is unknown. Report the notice count, hit counts, and fire IDs from the extracted rows in the iteration report. Read key names only from the private controlplane notice; never print a value or path. This read grants nothing, unparks nothing, and does not change the exit-code contract.

**Execution milestones**

| Milestone | Work and acceptance | Estimate |
|---|---|---|
| **M1 — Recognise transcript notices and guard the contract** | Add prefix constants, Python recognition, F8 check, companion test integration, classification/grammar/source-control tests. Both wording classes work; foreign/unanchored/malformed matches do not; missing current source signature floors before any gh-stub call. | 60–90 minutes |
| **M2 — Report, document, and verify** | Add transcript rows/counts/verdicts; update existing assertions; add window, arithmetic, privacy, previous-issue, and control-isolation tests; update prose/header/usage/changelog; regenerate mirror; run gates and mutation checks. | 60–90 minutes |

M2 depends on M1. Estimated implementation and test delta: roughly 500–650 lines, mostly companion tests. This is an estimate, not a measured velocity claim.

`registry_reuse: none` for both milestones: this extends the existing Bash/Python instrument and test harness; no package-like capability or dependency is introduced. No `.ail` changes or examples are needed.

**Fixture and test matrix**

Each new arm builds its own literal `5.json`/`5.meta.json` fixtures, plus issue 6 where needed, beneath its own `$SCRATCH` directory. Every instrument invocation uses the existing `make_stub_gh` through `--gh-bin`. Use explicit historical `--since` values; no arm reads real issue comments, real `$HOME`, or real `~/.ailang/state`.

New fixtures live as literal heredocs in the companion. Avoid helpers that synthesise comments, authors, or expected counts. Assertion failures print labels and expected metadata, never fixture bodies or captured output.

Each mutation below is applied only to a scratch copy of the instrument or driver source.

| Arm / fixture | Required observations | One-line mutation that must kill it |
|---|---|---|
| `transcript-old` | One old-wording self notice with hits `12`: one transcript row, correct fire ID, zero crash rows, rc 0. | Remove the old prefix from the transcript expression. |
| `transcript-current` | One current-wording notice: correct count/hits/fire/URL, including case-insensitive self login. | Remove the current prefix from the transcript expression. |
| `transcript-foreign` | Valid notice from a different login: no transcript row or verdict count. | Remove the transcript branch’s self-login condition. |
| `transcript-offset` | Leading-space and quoted-in-prose notices remain unclassified. | Replace anchored matching with substring search. |
| `transcript-ascii` | Unicode digits and an empty count are rejected; an ASCII-count positive control is classified. | Change `[0-9]+` to `\d+`. |
| `transcript-header-shape` | Missing literal period, wrong colon spacing, and a space inside the fire ID are rejected. | Make the literal terminal period optional. |
| `transcript-window` | Notices before, equal to, and after `SINCE`: rows show no/no/yes; issue total is 3; verdict count is 1; rc 0. | Change the transcript window comparison from `>` to `>=`. |
| `transcript-verdict-zero` | Transcript-only window with two notices, carrying hits `12` and `1`: rc 0 and verdict count **2**, plus private-notice disposition. | Increment the window count by hits instead of one per notice. |
| `transcript-verdict-crash` | One crash plus a later transcript: rc 1, transcript count 1, and newest-crash metadata still names the crash. | Omit the transcript suffix from the rc-1 verdict. |
| `transcript-other` | Six comments: one crash, two transcript notices, one other self comment, two foreign comments. Assert `comments=6 self=4 crash=1 transcript=2 other=1`. | Restore `other=self-crash`. |
| `transcript-no-bodies` | Both wordings carry `BODY_TAIL_SENTINEL_G087` after the count/header. Allowed metadata appears; sentinel is absent from **both stdout and stderr**. | Print the transcript body in the parser’s matching branch. |
| `transcript-prev` | A notice on issue 6 is reported with `--issue 5 --prev-issue 6 --control 5:0`. A second run with current=previous proves no double count. | Remove `emit_for "$PREV_ISSUE"`. |
| `transcript-control-isolation` | Control-only issue 6 contains one crash and one transcript; control expects 1. Issue 5 is plain: rc 0, transcript verdict count 0, no issue-6 transcript row. | Count `CRASH` and `TRANSCRIPT` together for `CTRL_GOT`. |
| `transcript-f8-current-missing` | Synthetic source contains crash and **old** transcript signatures, but no current signature: rc 2, transcript-specific diagnostic, empty stub argv log. | Delete the current-transcript F8 check. |
| `transcript-single-literal` | Current notice classifies normally; changing only the current-prefix Bash constant in a scratch instrument makes that notice stop classifying. | Hardcode the current prefix in Python instead of consuming its passed argument. |

Changed existing arms:

| Arm / fixture | Required change and observation | One-line mutation that must kill it |
|---|---|---|
| `signature-source` A | Add the current transcript signature to the positive driver fixture; omit the old prefix; rc 0 and source `ok`. | Require the old transcript prefix in F8. |
| `signature-source` B | Use a current-transcript-only source missing the crash signature; retain crash-specific rc-2 diagnostic and empty argv log. | Delete the crash F8 check. |
| `report` | Update exact rc-0/rc-1 verdict expectations to include zero transcript notices; retain recovery/provenance assertions. | Omit the zero transcript count from verdict output. |
| `snapshot-107`, `snapshot-129`, `snapshot-1380` | Preserve captures; add `transcript=0` to exact issue-row assertions. | Omit `transcript=` from issue output. |
| `driver-literal` | Through the stub, accept the repository driver and verify its current transcript prefix equals the script constant. Never execute the driver. | Change the current public prefix in a scratch driver supplied through `G0_DRIVER_SRC`. |

Preserve the remaining existing arms. The supplied “108 passed” is the assertion total; the base dispatcher has 28 named arms. Record the new assertion total rather than predicting it.

**Exact executor gates**

Run these outside the read-only planner sandbox. Stop on failures; do not reinterpret permission-denied tests as passes.

```bash
/bin/bash -n scripts/mission_gate0_self_notices.sh
/bin/bash -n scripts/test_mission_gate0_self_notices.sh
/bin/bash -n scripts/test_mission_gate0_self_notices_transcript.sh

/bin/bash scripts/mission_gate0_self_notices.sh --help
/bin/bash scripts/test_mission_gate0_self_notices.sh

/bin/bash tools/launchd/sync-agents-skills.sh
/bin/bash tools/launchd/sync-agents-skills.sh --check
cmp .claude/skills/mission-control/resources/gate-0-preflight.md \
    .agents/skills/mission-control/resources/gate-0-preflight.md

go test ./internal/mission/quorum \
    -run '^TestQuorumArtifactShellSuite$' -count=1

make check-changelog
make check-file-sizes
make test-changed TEST_CHANGED_BASE=f6a95d75d
git diff --check f6a95d75d
```

The sync command covers more than mission skills. Inspect its resulting diff: only the intended mission-control mirror should change. Do not include unrelated generated changes outside the write scope.

Explicit shell-file size check:

```bash
python3 -c 'from pathlib import Path; paths=["scripts/mission_gate0_self_notices.sh","scripts/test_mission_gate0_self_notices.sh","scripts/test_mission_gate0_self_notices_transcript.sh"]; sizes={p:len(Path(p).read_text().splitlines()) for p in paths}; print(sizes); assert all(n <= 800 for n in sizes.values())'
```

For ShellCheck, compare the same installation against the base. The following compares finding multiplicities while ignoring shifted line numbers; every companion-file finding is new. Unexpected ShellCheck failures remain failures.

```bash
python3 - <<'PY'
from collections import Counter
from pathlib import Path
import json
import subprocess

base = "f6a95d75d"
paths = [
    "scripts/mission_gate0_self_notices.sh",
    "scripts/test_mission_gate0_self_notices.sh",
    "scripts/test_mission_gate0_self_notices_transcript.sh",
]

def findings(path, source):
    result = subprocess.run(
        ["shellcheck", "--format=json", "--shell=bash", "-"],
        input=source, text=True, capture_output=True,
    )
    if result.returncode not in (0, 1):
        raise SystemExit("ShellCheck failed for " + path)
    return Counter(
        (path, f["code"], f["level"], f["message"])
        for f in json.loads(result.stdout)
    )

before, after = Counter(), Counter()
for path in paths:
    old = subprocess.run(
        ["git", "show", base + ":" + path],
        text=True, capture_output=True,
    )
    if old.returncode == 0:
        before.update(findings(path, old.stdout))
    elif path != paths[-1]:
        raise SystemExit("Cannot read base: " + path)
    after.update(findings(path, Path(path).read_text()))

added = after - before
print("base findings:", sum(before.values()))
print("current findings:", sum(after.values()))
for finding, count in sorted(added.items()):
    print("NEW", count, finding)
raise SystemExit(bool(added))
PY
```

Use the same command before and after implementation. Handle sourced-companion annotations narrowly; do not suppress warning classes broadly to satisfy the comparison.

For every mutation in the matrix, run its named arm against a scratch instrument:

```bash
G0_SCRIPT_UT="$mutation_script" \
    /bin/bash scripts/test_mission_gate0_self_notices.sh --only "$arm"
```

Require nonzero suite status and the intended named assertion failure. For the driver-wording mutation:

```bash
G0_DRIVER_SRC="$mutation_driver" \
    /bin/bash scripts/test_mission_gate0_self_notices.sh --only driver-literal
```

Verify each mutation actually changed its scratch file. Repository sources stay untouched throughout mutation checks.

**Controller done-gate**

At the exact reviewed revision, reproduce syntax, direct suite, ShellCheck comparison, mirror comparison/check, quorum shell integration, changelog, and whitespace gates above. Then run the required aggregate once:

```bash
make test-launchd-drivers
```

Verify scope and the absence of driver/makefile changes:

```bash
git diff --name-only f6a95d75d
git ls-files --others --exclude-standard
git diff --exit-code f6a95d75d -- \
    tools/launchd/mission-control.sh make/test.mk
```

The changed/new file set must match the six implementation paths listed above. No reload or live GitHub probe is required.

**Planner evidence and limitations**

- Both existing scripts passed `/bin/bash -n`.
- The Gate 0 copies passed `cmp`; the complete skill mirror check passed.
- ShellCheck **0.11.0** reported **93 findings** locally, versus the supplied baseline’s 87. Use the paired comparison, not either hardcoded count.
- Direct suite, mutations, regeneration, Go integration, make gates, and aggregate done-gate: **UNINFORMATIVE UNDER SANDBOX**; not run because they require writes. The supplied controller baseline remains external evidence.

**Risks and pushback**

- **Item 6 needs qualification:** the driver explicitly supports unknown-write-time fallback notices (`mission-control.sh:2711–2714`), tested at `test_transcript_secret_scan.sh:510–526`. Public wording alone cannot prove attribution to that fire’s write window.
- There is no stable live transcript positive control. Fixtures plus F8 detect parser and source-wording regressions, but do not prove successful live notice delivery.
- F8 checks literal source presence, not execution of the notice call. Preserve that existing mechanism and document its limit.
- Notices at exactly `SINCE` remain out of window, matching existing crash behavior. Changing watermark semantics is outside this ticket.
- Transcript counts are comment counts, not unique incidents or summed key/file hits. Rows retain hit counts and URLs for interpretation.
- A transcript-only result intentionally remains rc 0. Visibility depends on the explicit verdict count and Step 6a disposition; a new exit code would require a separate consumer-contract change.