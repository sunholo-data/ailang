# Fleet iteration 136 fix round 2

## Diff summary

- `scripts/mission_evidence_secret_scan.sh` detects bzip2, xz, zstd, and 7z signatures at byte zero through the archive classifier shared by files and archive members. Each signature produces `instrument-failure` with rc 2 regardless of the suffix. The supported readers still handle tar, tar.gz/tgz, and zip. LZMA-alone and zlib headers lack an unambiguous signature for this check.
- The scanner treats input basenames and traversed file and directory names as literal data. Name-only matches report the inventory key and a redacted path, return `LEAK` with rc 1 on an otherwise successful scan, and leave the content scan count intact.
- `scripts/test_mission_evidence_secret_scan.sh` adds `compressed-unsupported` and `name-value` to the default suite and updates the output-redaction case to expect the name finding. Compression cases include tool-built bzip2 and xz payloads and compressed tars, zstd and 7z signature fixtures, a renamed `.bin`, and a compressed payload inside both tar and zip. Name cases verify the key, the redacted path, and the absence of raw values in both output streams.
- The Public publish rule in `.claude/skills/mission-control/resources/gate-4-record.md` documents literal-only matching and supported archive formats in one sentence.

## Commands and exit codes

The suite and lint commands ran under `env -i` with synthetic `HOME`, worktree-local `TMPDIR`, an explicit tool `PATH`, and `LC_ALL=C`. `/bin/bash` reported version 3.2.57. Both `bzip2` and `xz` were available. `$scratch` below denotes a disposable directory inside the worktree.

| Command | rc | Result |
| --- | ---: | --- |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --arm compressed-unsupported` before the scanner fix | 1 | All nine fixture scans returned rc 0. |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --arm name-value` before the scanner fix | 1 | All three name-only scans returned rc 0. |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --arm compressed-unsupported` with the fix | 0 | All nine fixture scans returned rc 2 and `instrument-failure`. |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --arm name-value` with the fix | 0 | All three scans returned rc 1 with redacted findings. |
| `/bin/bash "$scratch/fallback-suite.sh" --helper "$PWD/scripts/mission_evidence_secret_scan.sh" --arm compressed-unsupported` | 0 | A scratch suite copy forced the tool-absent fallback for both compressors. |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh` | 0 | All 21 default arms passed. |
| `/bin/bash -n scripts/mission_evidence_secret_scan.sh` | 0 | Bash 3.2 syntax check passed. |
| `/bin/bash -n scripts/test_mission_evidence_secret_scan.sh` | 0 | Bash 3.2 syntax check passed. |
| `shellcheck scripts/mission_evidence_secret_scan.sh scripts/test_mission_evidence_secret_scan.sh` | 0 | No diagnostics. |
| `git --no-optional-locks diff --check` | 0 | No whitespace errors. |

## Mutation results

Each helper copy contained one deliberate change to the fixed scanner.

| Arm | Mutation and command | Suite rc | Result |
| --- | --- | ---: | --- |
| `compressed-unsupported` | Removed the unsupported-signature check from `archive_kind`. Ran `/bin/bash scripts/test_mission_evidence_secret_scan.sh --helper "$scratch/no-magic.sh" --arm compressed-unsupported`. | 1 | **KILLED**. All nine fixture scans reverted to rc 0. |
| `name-value` | Removed the name match from `scan_entry`. Ran `/bin/bash scripts/test_mission_evidence_secret_scan.sh --helper "$scratch/no-name.sh" --arm name-value`. | 1 | **KILLED**. All three name-only scans reverted to rc 0. |
