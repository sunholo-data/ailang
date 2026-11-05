Implemented transcript recognition, metadata reporting, notice counts, F8 checks, tests, documentation, and changelog. Exit codes remain **0/1/2**. The fixture suite passes **168 assertions**, and all **22 mutants** fail their intended assertions.

One planned step remains: sandbox permissions blocked regeneration of the `.agents` mirror. No git writes or unrelated content changes were made; driver and makefile sources are unchanged.

Files and final line counts:

| File | Lines | State |
|---|---:|---|
| `scripts/mission_gate0_self_notices.sh` | 415 | Modified |
| `scripts/test_mission_gate0_self_notices.sh` | 786 | Modified |
| `scripts/test_mission_gate0_self_notices_transcript.sh` | 370 | New |
| `.claude/skills/mission-control/resources/gate-0-preflight.md` | 305 | Modified |
| `.agents/skills/mission-control/resources/gate-0-preflight.md` | 299 | Unchanged; regeneration denied |
| `changelogs/unreleased/2026-11-05-gate0-self-notices-transcript-count.md` | 6 | New |

The only plan deviation is the incomplete mirror regeneration, caused by sandbox denials. The sync attempt changed no unrelated file content.

Gate results follow. **U** means **UNINFORMATIVE UNDER SANDBOX**. [Full command record](/tmp/g087-gates/results.jsonl).

| Command | rc | Result / last summary line |
|---|---:|---|
| Baseline `/bin/bash scripts/test_mission_gate0_self_notices.sh` | 0 | `108 passed, 0 failed` |
| `/bin/bash -n scripts/mission_gate0_self_notices.sh` | 0 | Pass; no output |
| `/bin/bash -n scripts/test_mission_gate0_self_notices.sh` | 0 | Pass; no output |
| `/bin/bash -n scripts/test_mission_gate0_self_notices_transcript.sh` | 0 | Pass; no output |
| `/bin/bash scripts/mission_gate0_self_notices.sh --help` | 0 | `signature-source, watermark unavailable/invalid, timestamp parser unusable)` |
| `/bin/bash scripts/test_mission_gate0_self_notices.sh` | 0 | `168 passed, 0 failed` |
| `/bin/bash tools/launchd/sync-agents-skills.sh && /bin/bash tools/launchd/sync-agents-skills.sh --check` | 1 | **U**; `rsync(97676): error: scripts: utimensat (2): Operation not permitted` |
| `/bin/bash tools/launchd/sync-agents-skills.sh --check` | 1 | **U**; `STALE: .agents/skills/mission-control differs from .claude/skills/mission-control` |
| `cmp .claude/skills/mission-control/resources/gate-0-preflight.md .agents/skills/mission-control/resources/gate-0-preflight.md` | 1 | **U**; copies differ at char 23322, line 256 after denied sync |
| Paired ShellCheck comparison: `python3 /tmp/g087-shellcheck.py` | 0 | `current findings: 91`; base 93, **no new findings** |
| Plan’s exact `python3 -c` shell-file size check | 0 | Reader 415; parent 786; companion 370 |
| `git diff --check` | 0 | Pass; no output |
| `go test ./internal/mission/quorum -run '^TestQuorumArtifactShellSuite$' -count=1` | 1 | **U**; `FAIL` — mirror-consistency check stopped on stale mirror |
| `make check-changelog` | 0 | `✓ CHANGELOG.md is index-only and links changelogs/v0.32-current.md` |
| `make check-file-sizes` | 0 | `✓ All files within 800 line limit` |
| `make test-changed TEST_CHANGED_BASE=f6a95d75d` | 2 | **U**; `make: *** [test-changed] Error 2` |
| `git diff --check f6a95d75d` | 0 | Pass; no output |

Every mutant below was a scratch copy under `mktemp -d`, verified different with `cmp` before running its named `--only` arm. Repository sources remained unchanged.

| Arm | Mutation | Differs (`cmp`) | Suite rc | Named assertion that failed |
|---|---|---:|---:|---|
| `transcript-old` | Remove old prefix | Yes (1) | 1 | `transcript-old row` |
| `transcript-current` | Remove current prefix | Yes (1) | 1 | `transcript-current row` |
| `transcript-foreign` | Remove self filter | Yes (1) | 1 | `transcript-foreign no row` |
| `transcript-offset` | Substring search | Yes (1) | 1 | `transcript-offset no row` |
| `transcript-ascii` | Allow Unicode digits | Yes (1) | 1 | `transcript-ascii rejects unicode` |
| `transcript-header-shape` | Optional terminal period | Yes (1) | 1 | `transcript-header-shape no row` |
| `transcript-window` | Include equal timestamp | Yes (1) | 1 | `transcript-window equal excluded` |
| `transcript-verdict-zero` | Sum hits as notices | Yes (1) | 1 | `transcript-verdict-zero notices not hits` |
| `transcript-verdict-crash` | Omit rc-1 suffix | Yes (1) | 1 | `transcript-verdict-crash suffix` |
| `transcript-other` | Restore self minus crash | Yes (1) | 1 | `transcript-other arithmetic` |
| `transcript-no-bodies` | Print body in transcript record | Yes (1) | 1 | `transcript-no-bodies sentinel absent stdout and stderr` |
| `transcript-prev` | Remove previous-issue emission | Yes (1) | 1 | `transcript-prev issue6 row` |
| `transcript-control-isolation` | Include transcripts in control | Yes (1) | 1 | `transcript-control-isolation crash only control` |
| `transcript-f8-current-missing` | Delete current F8 check | Yes (1) | 1 | `transcript-f8-current-missing diagnostic` |
| `transcript-single-literal` | Hardcode Python prefix | Yes (1) | 1 | `transcript-single-literal changed constant unclassified` |
| `signature-source` A | Require old prefix | Yes (1) | 1 | `signature-source A rc=2` |
| `signature-source` B | Delete crash F8 check | Yes (1) | 1 | `signature-source B rc=0` |
| `report` | Omit zero transcript count | Yes (1) | 1 | `report A rc=0` |
| `snapshot-107` | Omit issue transcript count | Yes (1) | 1 | `snapshot-107 issue line missing` |
| `snapshot-129` | Omit issue transcript count | Yes (1) | 1 | `snapshot-129 issue line missing` |
| `snapshot-1380` | Omit issue transcript count | Yes (1) | 1 | `snapshot-1380 issue line` |
| `driver-literal` | Change driver public prefix | Yes (1) | 1 | `driver-literal current transcript drift` |

The controller must regenerate the mirror outside the sandbox, rerun all gates at the reviewed revision, and run the designated `make test-launchd-drivers` aggregate. `test-changed` encountered denied loopback binds and a denied cache-directory write; those results cannot establish pass or fail.