# Fleet iteration 136 executor report

Ticket `publish:evidence-pack:no-secret-value-guard`. Branch `fleet/i136-evidence-secret-scan`. Base `16d168f1900d6d2ea05cd30f08924141ebb0da81`. The scanner, launchd wiring, and M3 skill text are complete.

## Files and snapshots

| Path | Final lines | Change from base |
| --- | ---: | ---: |
| `scripts/mission_evidence_secret_scan.sh` | 388 | +388 |
| `scripts/test_mission_evidence_secret_scan.sh` | 524 | +524 |
| `make/test.mk` | 464 | +3 |
| `.claude/skills/mission-control/SKILL.md` | 628 | +3 / −3 |
| `.claude/skills/mission-control/resources/gate-4-record.md` | 403 | +13 |

- `.snap/M1/` holds the 271-line regular-file helper and the 339-line M1 suite. I backported the private findings handoff and its diagnostic-suppression assertion to this snapshot after identifying the gap during M2, then reran the isolated M1 checks.
- `.snap/M2/` holds the two final scripts and `make/test.mk`.
- `.snap/M3/` holds the three M2 files and both mission-control skill files. The files match the candidate byte for byte. Both scripts are executable in each snapshot.
- `.i136_executor_report.md` and the snapshots are handoff artifacts. `.i136_plan.md` remains in the worktree.

## Acceptance

Fixture tests used a fresh worktree-local HOME and TMPDIR with `env -i`, a minimal tool PATH, and `LC_ALL=C`. The suite gave each helper invocation its own synthetic HOME and credential fixtures. Running the earlier test fixture before implementation failed as expected in M1 (`clean-dir`, suite rc 1, helper rc 127) and M2 (`tar`, suite rc 1, missing member and count assertions).

| Milestone | Command or check | Result |
| --- | --- | --- |
| M1 | `/bin/bash -n` on both original M1 scripts, followed by `shellcheck -s bash` and the scrubbed default suite | rc 0 for every command, 11 arms passed |
| M1 | Same syntax and ShellCheck commands against both `.snap/M1/scripts/` files, followed by that snapshot's scrubbed default suite | rc 0 for every command, 11 arms passed |
| M2 | `/bin/bash -n scripts/mission_evidence_secret_scan.sh` | rc 0 |
| M2 | `/bin/bash -n scripts/test_mission_evidence_secret_scan.sh` | rc 0 |
| M2 | `shellcheck -s bash scripts/mission_evidence_secret_scan.sh scripts/test_mission_evidence_secret_scan.sh` | rc 0, no diagnostics |
| M2 | `env -i HOME=… TMPDIR=… PATH=… LC_ALL=C /bin/bash scripts/test_mission_evidence_secret_scan.sh` | rc 0, 18 arms passed. The deflated ZIP and chmod-000 subcases ran. |
| M2 | `make -n test-launchd-drivers` piped to the plan's `awk` check with `pipefail` | rc 0. One syntax check per script and one suite registration. |
| M2 | `scripts/test_mission_evidence_secret_scan.sh --helper <copy> --arm <name>` in the scrubbed environment | All 15 named arms passed against the reference. The mutation drill returned rc 0 with 26 killed and 0 survived. |
| M3 | `/bin/bash tools/launchd/test_agents_skills_sync.sh` | rc 1, **BASE-RED** |
| M3 | `/bin/bash tools/launchd/test_quorum_artifacts.sh` | rc 1, **BASE-RED** at `generated_docs_and_mirror` |
| M3 | `/bin/bash scripts/check_context_docs.sh` | rc 0. The main skill stayed at 628 lines. |
| M3 | `make check-changelog` | rc 0 for the existing 27 fragments and root index |
| M3 | `git diff --check 16d168f19` | rc 0 |
| M3 | `git diff --exit-code 16d168f19 -- tools/launchd/transcript-secret-scan.sh` | rc 0 |
| M3 | `GIT_OPTIONAL_LOCKS=0 git status --short --untracked-files=all` | rc 0. Implementation paths, cumulative snapshots, plan, and handoff report only. |
| M3 | Scan exact local fixtures with the pinned-root command from a simulated sibling worktree, then run the local publish stand-in in a separate tool call | Scan batch rc 0 against expected scan statuses. Stand-in rc 0 with four byte-identical receipts and five cases withheld. |

### M3 local publication smoke

The scan call used the same files as the later publish call. The Git fixture was materialized with `git show` from a committed blob. The bucket fixture was a finished archive built from a named file. The stand-in used `cp` only after checking the scan rc, verdict, hits, and control, and verified each receipt with `cmp -s`.

| Fixture | Scanner rc | Verdict | Local receipt |
| --- | ---: | --- | --- |
| Gate 0 saved issue comment | 0 | clean, control ok | Byte-identical |
| Gate 5 saved digest | 0 | clean, control ok | Byte-identical |
| Committed Git evidence bytes | 0 | clean, control ok | Byte-identical |
| Finished bucket archive | 0 | clean, control ok | Byte-identical |
| Gate 0 and Gate 5 leaked prose, two cases | 1 | LEAK, control ok | Withheld |
| Gate 0 missing artifact | 2 | instrument-failure, control ok | Withheld |
| Gate 5 without eligible control values | 2 | instrument-failure, control failed | Withheld |
| Corrupt bucket archive | 2 | instrument-failure, control ok | Withheld |

## Mutation drill

Each row represents an independent edit to a helper copy in worktree-local scratch. The runner checked each copy with `/bin/bash -n`, compiled its embedded Python, and exercised its startup with a separate synthetic inventory before the named-arm run. The helper rc below is the rc that the suite recorded at the failing assertion. Failure output contained only assertion identifiers and rc values.

| # | Mutant | Arm | Suite rc | Helper rc | Failing assertion | Result |
| ---: | --- | --- | ---: | ---: | --- | --- |
| 1 | Force a hit in the target-file branch | `clean-dir` | 1 | 1 | `clean-rc` | KILLED |
| 2 | Omit file-source enrollment | `file-value` | 1 | 0 | `file-literal-finding` | KILLED |
| 3 | Use regex matching | `file-value` | 1 | 0 | `file-literal-finding` | KILLED |
| 4 | Source the fixture file | `file-value` | 1 | 1 | `no-shell-execution` | KILLED |
| 5 | Drop the OAuth branch | `env-only` | 1 | 1 | `oauth-finding` | KILLED |
| 6 | Make the env name test case-sensitive | `env-only` | 1 | 1 | `lowercase-finding` | KILLED |
| 7 | Deduplicate solely by key | `make-dump` | 1 | 1 | `file-side-finding` | KILLED |
| 8 | Require `NAME=value` in target evidence | `make-dump` | 1 | 0 | `file-side-finding` | KILLED |
| 9 | Lower the minimum to 19 bytes | `short-value` | 1 | 1 | `short-value-rc` | KILLED |
| 10 | Return clean for an empty inventory | `zero-values` | 1 | 0 | `zero-values-rc` | KILLED |
| 11 | Omit the control call | `control` | 1 | 2 | `firing-control` | KILLED |
| 12 | Stub the common matcher to return no hits | `control` | 1 | 2 | `firing-control` | KILLED |
| 13 | Treat failed target stat/open as a clean skip | `unreadable-path` | 1 | 0 | `unreadable-rc` | KILLED |
| 14 | Emit the raw path label | `output-guard` | 1 | 1 | `no-raw-values` | KILLED |
| 15 | Emit matching bytes | `output-guard` | 1 | 1 | `no-raw-values` | KILLED |
| 16 | Remove the TERM handler | `interrupted` | 1 | 143 | `interrupted-rc` | KILLED |
| 17 | Bypass EXIT scratch cleanup | `interrupted` | 1 | 2 | `scratch-cleanup` | KILLED |
| 18 | Scan the raw tar container | `tar` | 1 | 1 | `tar-member-finding` | KILLED |
| 19 | Scan the raw gzip container | `tgz` | 1 | 0 | `tgz-boundary-finding` | KILLED |
| 20 | Drop carry bytes between chunks | `tgz` | 1 | 1 | `tgz-boundary-finding` | KILLED |
| 21 | Skip the ZIP member loop | `zip` | 1 | 0 | `zip-member-finding` | KILLED |
| 22 | Skip nested checking | `nested-archive` | 1 | 1 | `nested-by-name.tar-rc` | KILLED |
| 23 | Check only nested suffixes | `nested-archive` | 1 | 1 | `nested-by-header.tar-rc` | KILLED |
| 24 | Let a leak override instrument failure | `nested-archive` | 1 | 1 | `nested-by-name.tar-rc` | KILLED |
| 25 | Ignore gzip decoder errors during drain | `bad-archive` | 1 | 1 | `invalid-bad-trailer.tgz-rc` | KILLED |
| 26 | Stop gzip validation at the final tar member | `bad-archive` | 1 | 1 | `invalid-bad-trailer.tgz-rc` | KILLED |

The disabled matcher also passed the dedicated fail-closed `--arm broken-matcher` probe with suite rc 0. The probe asserts helper rc 2, `control=failed`, `scanned=0`, and `hits=0`.

## Base-red findings and scope

- On the unchanged skill sources at `16d168f19`, `test_agents_skills_sync.sh` returned rc 1 with 6 passed and 1 failed. The sprint-planner, sprint-executor, and sprint-evaluator mirrors were already stale. `test_quorum_artifacts.sh` returned rc 1 at `generated_docs_and_mirror` on the same mirror check. The candidate adds the expected mission-control drift pending regeneration. Both failures are **BASE-RED**. The controller owns mirror regeneration and the focused rerun.
- A standalone `sync-agents-skills.sh --check` was omitted under the executor restriction. The two requested focused suites exercised check mode internally. The M3 changelog fragment remains controller work because this execution was scoped to M3 skill text.
- `make test-launchd-drivers` and the live configuration observation are **UNINFORMATIVE UNDER SANDBOX**. The controller-only aggregate was outside executor scope because it depends on sockets and host paths. The wiring dry-run was exercised locally.
- The shell suite and local publication smoke completed with synthetic fixtures. The controller has the original implementation files and cumulative snapshots for building commits.
