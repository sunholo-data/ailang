M3a and M3b are complete as a **proposed draft only**. Snapshots are verified: `.snap/M3a/` contains the preview and initial test harness; `.snap/M3b/` contains all five changed files.

Files changed:

- `tools/launchd/mission-control.sh`
- `tools/launchd/test_secrets_scope.sh`
- `tools/launchd/test_secrets_scope_dry_run.sh` — new
- `design_docs/planned/m-fleet-secrets-scope-per-lane.md`
- `changelogs/unreleased/2026-11-05-secrets-scope-per-lane.md`

All pre-existing driver lines remain byte-for-byte unchanged.

| Gate | RC | Result |
|---|---:|---|
| `/bin/bash -n`, all three changed scripts | 0 each | Informative; passed |
| Secrets-scope suite | 0 | Informative; 59 passed |
| Secrets-scope `--mutations` | 0 | Informative; 58 passed |
| `make test-launchd-drivers` | 2 | **UNINFORMATIVE UNDER SANDBOX** for affected arms |
| ShellCheck: new test script | 0 | Informative; clean |
| ShellCheck: existing test script | 0 | Informative; clean |
| ShellCheck: driver | 1 | Same 33 findings as base; zero new |
| Skill-sync `--check` | 0 | Informative; passed |
| `make check-changelog` | 0 | Informative; passed |
| `git diff --check` | 0 | Informative; passed |

The full launchd gate reported kill-tree rc 2, stall rc 1, and Codex-controller-env rc 1. Process inspection and nested sandbox execution are denied here. Additional process/loopback arms also reported uninformative results. No workaround was applied.

| Mutant | Arm | Killed | Failing assertion |
|---|---|---|---|
| Remove preview scope call | Dry-run preview | Yes | Probe child stripped synthetic keys |
| Force probe status to zero | Dry-run failure | Yes | Failure returns nonzero status |
| Apply scope in parent | Dry-run parent env | Yes | Parent environment hash/count unchanged |
| Drop forwarded marker | Dry-run codex forward | Yes | Codex preview exits zero |
| Remove skip line | Dry-run no pin | Yes | No-pin skip line |

Existing M1/M2 mutation banks also completed successfully; the existing Bash-equivalent `multiple` mutant was killed under zsh.

The six live dry-runs, outside-sandbox acceptance gates, independent evaluation, commit construction, and PR handling remain the controller’s evidence work. No live authentication probe or host configuration change was performed. The requested legacy launchd gate does create temporary Git fixtures; no Git write command was issued against this worktree.

Remaining risks: reused real probes can refresh authentication, write caches, temporary files, sessions, and managed Codex daemon/runtime state. Claude’s probe retains its existing session behavior and diagnostic logging. Synthetic tests establish neither live authentication nor quorum/extension availability. Without Mark’s host hook, measured Claude/Codex startup exposure remains; retained keys, credential files, deliberate re-sourcing, mutable driver roots, and native subagents inheriting controller scope remain exposed. Existing billing unsets remain intact, including the missing initial OpenAI key and resulting pi GPT-quorum limitation.