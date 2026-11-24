# Fleet iteration 136, fix round 1

`input_parent()` now follows directory symlinks in the explicit parent prefix. Each open is relative to the previous directory descriptor, and the final lookup stays relative to the pinned parent. Final and in-tree symlinks, dangling paths, special files, and nested archives still fail closed.

The new `symlinked-prefix` arm uses `alias -> real` inside the suite's worktree-local fixture. It checks that `alias/clean.txt` returns `0` with `clean` and `scanned=1`, `alias/leak.txt` returns `1` and reports `CONTROL_TOKEN`, and both `alias/link.txt -> clean.txt` and a recursive scan of `real/` return `2`.

## Verification

Checks ran on macOS with Bash 3.2.57. The test invocations used `env -i` with a synthetic `HOME` and a worktree-local `TMPDIR`. `$scratch` below denotes the worktree-local directory that held the disposable mutant copies.

| Command | Exit | Result |
| --- | --- | --- |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh` | 0 | All 19 default arms passed |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --arm unreadable-path` | 0 | Passed without a permission skip |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --arm clean-dir` | 0 | Passed |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --arm symlinked-prefix` | 0 | Passed |
| `shellcheck scripts/mission_evidence_secret_scan.sh scripts/test_mission_evidence_secret_scan.sh` | 0 | Clean |
| `/bin/bash -n scripts/mission_evidence_secret_scan.sh` | 0 | Passed |
| `/bin/bash -n scripts/test_mission_evidence_secret_scan.sh` | 0 | Passed |
| `git --no-optional-locks diff --check` | 0 | Clean |

## Mutations

| Command | Test exit | Result |
| --- | --- | --- |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --helper "$scratch/prefix-nofollow.sh" --arm symlinked-prefix` | 1 | **KILLED**. Restoring `O_NOFOLLOW` on prefix opens caused both prefix file scans to return `2`. |
| `/bin/bash scripts/test_mission_evidence_secret_scan.sh --helper "$scratch/follow-final.sh" --arm symlinked-prefix` | 1 | **KILLED**. Substituting the fd-relative `readlink()` target for the explicit final name made `alias/link.txt` return `0`. |
