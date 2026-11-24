# Fleet iteration 136 sprint plan

**FLEET-I136-EVIDENCE-SECRET-SCAN** · Base `16d168f19` · Ticket `publish:evidence-pack:no-secret-value-guard` · One focused day across three sequential milestones.

The supplied incident and missing-guard findings stand. The placement detail is that public posts already occur before and after Gate 4. Gate 0 prescribes a public issue comment and Gate 5 publishes the digest. A Gate 4 rule therefore needs an early pointer that applies whenever a mission publishes, including in sibling repositories. The driver already exports `MISSION_DRIVER_ROOT` for these repositories.

The sprint delivers a pre-publication value scan, a Bash 3.2 regression suite, and a common rule for public evidence and reports. The existing post-fire detector stays byte-identical. No open human decisions block the sprint. Registry reuse is `none` for M1–M3. These are local launcher, test, and skill changes. Reuse the detector's parsing conventions and threshold, the Python standard library already used in the launchd lane, and the existing suite runner and mirror generator.

## Base observations

The worktree was clean at `16d168f19`. The following focused checks ran on that base with genuine `/bin/bash` 3.2.57. Repeat them with a disposable in-worktree `TMPDIR` as in the M3 focused check. The planner left the mirror untouched.

| Command | Base result | Candidate expectation |
| --- | --- | --- |
| `/bin/bash tools/launchd/test_agents_skills_sync.sh` | rc 1, 6 passed and 1 failed. Stale `sprint-planner`, `sprint-executor`, `sprint-evaluator` copies. | rc 0 after regeneration. |
| `/bin/bash tools/launchd/test_quorum_artifacts.sh` | rc 1 at `generated_docs_and_mirror`, from that same `--check`. | rc 0 after regeneration. |
| `/bin/bash scripts/check_context_docs.sh` | rc 0. Mission-control `SKILL.md` is 628 lines against its existing 2,781-line ratchet. | rc 0 with a line-neutral pointer and the existing ratchet. |
| `make check-changelog` | rc 0. | rc 0 with the new fragment. |

Commit `b49dc1c59` changed the `.claude` sprint sources without regenerating their mirrors. The two focused failures are inherited. The full aggregate target was not run on the base. The controller should capture per-suite results from that target on base and candidate under the same environment. The two known failures must clear on the candidate.

## M1 — Load secret values and scan regular files

**Budget** 60–90 min. **Files** `scripts/mission_evidence_secret_scan.sh`, `scripts/test_mission_evidence_secret_scan.sh`.

- Use a Bash 3.2 entry point and a single embedded, isolated `python3 -I -B` standard-library worker for byte matching and, in M2, archive records. The launchd lane already uses Python 3 in `test_pin_root.sh` and `test_skill_sync.sh`. This keeps credential collection and matching in one process and avoids parsing ambiguous human-readable archive listings. Missing Python is an instrument failure. Start with tracing disabled and `umask 077`. Use a private `mktemp -d` directory and an EXIT trap, with HUP/INT/TERM handlers that stop and reap the worker before cleanup. Keep the control fixture and any status handoff inside that directory. Catch worker and cleanup failures and have the launcher emit the single final verdict. Keep raw subprocess and interpreter diagnostics off both output streams.
- Read `HOME/.config/ailang/secrets.env` as data. Accept literal `KEY=VALUE` lines, optional `export`, comments and blanks, and one matched pair of surrounding quotes. Reject unreadable or malformed existing sources. Allow an absent file so an env-only launch can still pass its control. Independently load exported names containing `TOKEN`, `KEY`, `SECRET`, `AUTH`, `PASSWORD`, or `CREDENTIAL`, case-insensitively. Keep eligible values from both sources even when the key names agree. Use a **20-byte minimum**, matching the transcript detector's default and avoiding noisy matches on short strings. No expansion, sourcing, or secret values in child command arguments, even for the matcher. Read the inherited environment directly in the worker.
- Use literal byte substring matching on streams, including binary files and matches spanning read boundaries. Run the same matcher on a private fixture populated with one real loaded value before scanning targets. Zero eligible values or a missed control means rc 2. Count distinct eligible values in `values`, successfully scanned regular files or regular archive members in `scanned`, and one finding per path/member and key in `hits`. Check all requested inputs. An instrument error wins over a leak. Traverse directories recursively with enumeration errors surfaced, and fail closed for symlinks, special files, missing inputs, or read failures.
- Emit `LEAK key=<key> path=<path>` with an optional `member=<member>` field and no matching text. Escape control characters in displayed names. Redact any output field that itself contains a loaded value, including names originating in input paths or headers. Emit one final line `mission_evidence_secret_scan: verdict=clean|LEAK|instrument-failure values=<n> scanned=<n> hits=<n> control=ok|failed`. Exit 0 for clean, 1 for leak, and 2 for instrument failure. A target failure after a successful control retains `control=ok`.
- Give the suite a default full run and optional `--helper PATH --arm NAME` for a disposable mutant. Run the suite in a scrubbed environment and set a synthetic HOME before fixture setup. Each arm uses a fresh synthetic HOME and captures stdout and stderr separately. Start each helper invocation with `env -i`, restore only the fixture HOME, fixture TMPDIR, minimal tool PATH and locale, then set synthetic credential exports using shell builtins inside that clean process. Check expected exit, hit records, counters, one final verdict, and absence of **every** synthetic value across both streams, even after assertion failures. Check scratch cleanup on clean, leak, error and interruption. Use an instrumented synthetic worker for the interruption arm so the test can send TERM after scratch exists and can reap it with a bounded wait.

**Acceptance** Run these commands from repo root in Bash. The disposable parent keeps the focused fixtures in the worktree.

```bash
/bin/bash -n scripts/mission_evidence_secret_scan.sh &&
  /bin/bash -n scripts/test_mission_evidence_secret_scan.sh &&
  shellcheck -s bash scripts/mission_evidence_secret_scan.sh scripts/test_mission_evidence_secret_scan.sh &&
  ( TMPDIR=$(mktemp -d "$PWD/.i136-tests.XXXXXX") || exit 1
    trap 'rm -rf "$TMPDIR"' EXIT
    mkdir -p "$TMPDIR/home" || exit 1
    env -i HOME="$TMPDIR/home" TMPDIR="$TMPDIR" PATH="$PATH" LC_ALL=C \
      /bin/bash scripts/test_mission_evidence_secret_scan.sh
  )
```

At M1 the default suite includes the regular-file and lifecycle arms below. The final M2 run adds archive arms. The proof is the assertions and mutant results, including a clean run with a firing control.

## M2 — Scan archive members and wire the Bash 3.2 lane

**Budget** 60–90 min. **Depends on** M1. **Files** the two M1 scripts and `make/test.mk`.

- Use `tarfile` and `zipfile` to open regular members by archive record and stream bytes through the M1 matcher, keeping the original archive path and member name. Support `.tar`, `.tar.gz`, `.tgz`, and `.zip`. Identify those candidates by case-insensitive suffix or recognizable headers, including gzip and ZIP signatures and a tar header. Check nested member names and leading bytes before matching. A nested archive candidate gives rc 2, even with a renamed member or when another member has already leaked. Open member objects directly so duplicate names cannot cause a different record to be read. Refuse special members and links. Directory entries may be counted as metadata only. Never extract members to their recorded paths.
- Validate archive completion. Read each regular member to EOF so short reads, decompression errors, and ZIP CRC failures surface. For tar, validate the end records and drain the gzip stream through its integrity trailer. A malformed, encrypted, truncated, or otherwise unreadable archive is rc 2. Check stored member names for literal values as artifact data and sanitize them through the M1 output path.
- Register `scripts/test_mission_evidence_secret_scan.sh` exactly once in `LAUNCHD_SUITES`, beside the existing transcript suite, and add separate `/bin/bash -n` lines for both new scripts under `test-launchd-drivers`. `tools/launchd/lib/run-suites.sh` runs the list via `suite-env.sh`, whose allowlist preserves HOME, PATH and TMPDIR. The new suite still sets its own synthetic HOME and scrubs the helper environment. The existing macOS workflow matches the added shell paths and `make/test.mk` and already invokes the target.

**Acceptance** Repeat the M1 command block with all arms active and run the make wiring check below. Execute the actual mutations from the table, including archive mutants. ZIP reports an explicit skip if `unzip` is absent. The rig has `unzip`, so the controller run must exercise ZIP and kill its mutant.

```bash
( set -o pipefail
  make -n test-launchd-drivers | awk '
    $1=="/bin/bash" && $2=="-n" && $3=="scripts/mission_evidence_secret_scan.sh" { helper++ }
    $1=="/bin/bash" && $2=="-n" && $3=="scripts/test_mission_evidence_secret_scan.sh" { suite++ }
    $1=="/bin/bash" && $2=="tools/launchd/lib/run-suites.sh" {
      for (i=4; i<=NF && $i!="--serial"; i++)
        if ($i=="scripts/test_mission_evidence_secret_scan.sh") reach++
    }
    END { exit !(helper==1 && suite==1 && reach==1) }'
)
```

### Mutation drill

Run each arm against the unmodified helper first. Apply each edit independently to a private worktree scratch copy of the helper, confirm the changed copy still parses and initializes, then run the named arm against that copy. Require the specified assertion to fail, with the suite reporting only the assertion name and return code. Restore the scratch copy between mutants. Fixture values and captured logs stay private. A syntax error, skipped arm, or unrelated initialization failure is insufficient.

| Arm | Expected result and assertion | Concrete mutant |
| --- | --- | --- |
| `clean-dir` | rc 0, `control=ok`, recursive file counts and zero hits. | Unconditionally return a hit in the target-file branch while leaving the control intact. |
| `file-value` | rc 1 and the expected key/path for a quoted literal containing regex metacharacters and shell syntax. Keep a separate synthetic control value. Assert no execution of the embedded shell syntax. | Omit file-source enrollment while retaining an env control. Also run independent mutants that use regex matching or source the fixture file. |
| `env-only` | rc 1 with the synthetic `CLAUDE_CODE_OAUTH_TOKEN` key and no secrets file. Retain a separate env control if dropping that key. Include a lowercase credential-shaped export in the same arm. | Drop the OAuth key branch while preserving the control, or make the name test case-sensitive. |
| `make-dump` | rc 1 for `NAME = value`, with different file and env values under one key and an assertion that both are found in distinct files. | Deduplicate only by key, or require evidence to match `NAME=value` syntax. |
| `short-value` | rc 0 and `control=ok` with a 19-byte near-threshold input alongside a 20-byte loaded control. | Lower the minimum to 19. |
| `zero-values` | rc 2, `values=0`, `control=failed`. | Route an empty inventory to the clean verdict. |
| `control` | A normal clean scan reports `control=ok`. In a broken-matcher probe, disabling the common matcher must instead give rc 2 and `control=failed` before scanning any target. | Omit the control call while leaving its default failed state. Separately stub the matcher to return no matches and verify the helper refuses to scan. |
| `unreadable-path` | rc 2 for an existing chmod-000 file on the non-root rig and for a missing path. | Turn a failed open into a clean skip. If the test runs as root, use the missing-path case for kill evidence and report the permission subcase as skipped. |
| `output-guard` | rc 1 with a redacted secret-bearing member or path label, and zero raw values on stdout and stderr. | Print the raw name or matching bytes. The suite must reject the copy without echoing the captured value. |
| `interrupted` | TERM exits 2 and the private fixture scratch is empty after the worker is reaped. | Remove TERM cleanup or bypass EXIT cleanup. |
| `tar` | rc 1 with exact archive/member/key for a plain tar, including a member name with spaces. | Skip opening an uncompressed tar and scan only its raw container bytes. Raw hits lack the required member field. |
| `tgz` | rc 1 with exact member/key and consumed-member count. Include `.tar.gz` and a value crossing a matcher chunk boundary. | Treat the gzip container as a regular file. Separately drop the carry bytes between matcher chunks. |
| `zip` | rc 1 with exact member/key for a deflated member. | Skip the zip member loop. |
| `nested-archive` | rc 2 for a nested archive by name and for one renamed to `.bin`. Put a leaking member first to prove failure precedence. | Skip nested checking, classify only by suffix, or let accumulated leakage override instrument failure. |
| `bad-archive` | rc 2 for a corrupt archive and a gzip with a bad or missing trailer after complete member data. | Ignore decoder errors or stop reading gzip at the last tar member. |

For the `control` arm, the executor records both the omitted-probe mutant's failing normal assertion and rc 2 from the broken-matcher probe. The latter is evidence that the instrument failed closed. The suite's mutant interface is exercised by the following runnable example against the reference helper. Each mutant uses the same interface with its disposable copy and the arm from the table.

```bash
( TMPDIR=$(mktemp -d "$PWD/.i136-mutation.XXXXXX") || exit 1
  trap 'rm -rf "$TMPDIR"' EXIT
  mkdir -p "$TMPDIR/home" || exit 1
  env -i HOME="$TMPDIR/home" TMPDIR="$TMPDIR" PATH="$PATH" LC_ALL=C \
    /bin/bash scripts/test_mission_evidence_secret_scan.sh \
      --helper "$PWD/scripts/mission_evidence_secret_scan.sh" --arm tgz
)
```

## M3 — Require a clean scan before any public publish

**Budget** 30–45 min plus the aggregate launchd check. **Depends on** M2. **Files** `.claude/skills/mission-control/SKILL.md`, `.claude/skills/mission-control/resources/gate-4-record.md`, their two `.agents/skills/mission-control/` mirrors, the generated `.agents/skills/sprint-{planner,executor,evaluator}/` catch-up from `b49dc1c59`, and `changelogs/unreleased/2026-11-24-evidence-secret-scan.md`.

- Add one short `Public publish` rule near the top of `gate-4-record.md`. Use a line-neutral pointer in the main skill's standing guidance before Gate 0. The pointer applies before public writes in any gate and tells controllers to carry the canonical named-file instruction into every designer brief that asks for evidence. This covers designers launched without access to the skill and avoids duplicating the rule in the `design-doc-creator` sources.
- The canonical order is **assemble, scan, read, publish**. Assemble from explicitly named files. Exclude whole tool-output directories and environment dumps from `make -p`, `env`, `set`, and `printenv`. In its own tool call, run `/bin/bash "$MISSION_DRIVER_ROOT/scripts/mission_evidence_secret_scan.sh"` on the exact bytes going to a public bucket, public repository, or issue/PR comment or attachment. Use the pinned driver root even from a sibling repo or its scratch worktree. A missing helper blocks publication. For Git, check the staged or committed evidence bytes that will actually be pushed. For GitHub prose, scan the final saved text and use those bytes as the publication input. For archives, scan the finished archive. Read the exit status and final line. Require rc 0, `verdict=clean`, `hits=0`, and `control=ok` before making a separate, later publish call. Rescan if any bytes change. On rc 1 or 2, a missing verdict, or a failed control, withhold publication and report only safe status and redacted metadata through an appropriate private channel.
- Run `/bin/bash tools/launchd/sync-agents-skills.sh` after editing the source. The provided sandbox protects `.agents` from writes, so regeneration is **controller-only** there. Expect the two mission-control mirrors plus the three previously stale sprint trees. Inspect the generated path list and stop on any unexpected out-of-scope mirror churn. Use a `### Added — ...` opening line in the changelog fragment.

**Acceptance** After regeneration, the following focused checks must pass. Run from Bash at the candidate root. A fresh top-level policy read should lead a controller through the exact artifact and two separate tool calls for each public sink, including an early Gate 0 post and a Gate 5 digest. Demonstrate on synthetic inputs with a local stand-in for publication. Review the tool calls to verify that clean goes through and both rc 1 and rc 2 stop before the stand-in. No public write is needed for validation.

```bash
( TMPDIR=$(mktemp -d "$PWD/.i136-doc-checks.XXXXXX") || exit 1
  export TMPDIR
  trap 'rm -rf "$TMPDIR"' EXIT
  /bin/bash tools/launchd/sync-agents-skills.sh --check &&
    /bin/bash tools/launchd/test_agents_skills_sync.sh &&
    /bin/bash tools/launchd/test_quorum_artifacts.sh &&
    /bin/bash scripts/check_context_docs.sh &&
    make check-changelog &&
    git diff --check 16d168f19 &&
    git diff --exit-code 16d168f19 -- tools/launchd/transcript-secret-scan.sh &&
    git --no-optional-locks status --short --untracked-files=all
)
```

**Controller-only aggregate and observations.** `make test-launchd-drivers` includes suites with socket and host/path dependencies. Their verdicts are **UNINFORMATIVE UNDER SANDBOX**. Run the block below on unchanged base before execution and on the candidate after regeneration, using the same rig and recording SHA, command status, and failing suite or arm names. The test suite still uses synthetic credentials. Observe the live `secrets.env` before and after with an in-memory SHA and line count, comparing without displaying either value. A missing file is recorded as absent. Never seed guard files in real HOME. Use a bounded poll if the aggregate runs asynchronously. Require the new suite and both mirror-dependent checks green and investigate any other difference against base.

```bash
( observe_config() {
    if [ ! -e "$1" ] && [ ! -L "$1" ]; then printf 'absent\n'; return; fi
    [ -f "$1" ] && [ -r "$1" ] || return 2
    shasum -a 256 < "$1" && wc -l < "$1"
  }
  config_path="$HOME/.config/ailang/secrets.env"
  before=$(observe_config "$config_path") || exit 2
  lab=$(mktemp -d "$PWD/.i136-launchd.XXXXXX") || exit 2
  trap 'rm -rf "$lab"' EXIT
  mkdir -p "$lab/home" "$lab/tmp" || exit 2
  git rev-parse HEAD || exit 2
  env -i HOME="$lab/home" TMPDIR="$lab/tmp" PATH="$PATH" LC_ALL=C make test-launchd-drivers
  result=$?
  after=$(observe_config "$config_path") || exit 2
  [ "$before" = "$after" ] || { printf '%s\n' 'Live config observation changed' >&2; exit 2; }
  printf 'launchd suite rc=%s\n' "$result"
  exit "$result"
)
```

Review tracked and untracked paths against the milestone file lists before handoff. Keep the generated sprint mirror catch-up explicit. The plan and JSON are planning outputs and stay outside the implementation diff.
