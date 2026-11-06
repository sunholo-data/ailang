M2a and M2b are implemented. M2c prose is implemented; its generated mirror is supplied in the snapshot because the sandbox denied writes to `.agents/`. Full acceptance remains incomplete.

Cumulative snapshots and SHA-256 manifests are in `.snap/M2a/`, `.snap/M2b/`, and `.snap/M2c/`. No Git writes or merge occurred.

The [executor report](/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter89-secrets-m2/.snap/evidence/executor-report.md) contains the complete gate ledger, 42-row mutation table, deviations, and risks.

Files changed:

```text
.claude/skills/mission-control/resources/gate-3-route.md
changelogs/unreleased/2026-11-05-secrets-scope-per-lane.md
design_docs/planned/m-fleet-secrets-scope-per-lane.md
scripts/mission_pi_run.sh
scripts/mission_scoped_run.sh
scripts/test_mission_pi_run.sh
tools/launchd/lib/codex-env-args.sh
tools/launchd/mission-control.sh
tools/launchd/test_codex_controller_env.sh
tools/launchd/test_controller_capacity.sh
tools/launchd/test_mission_kill_tree.sh
tools/launchd/test_mission_pi_run_commits.sh
tools/launchd/test_mission_pi_run_provider_quota.sh
tools/launchd/test_mission_pi_run_sandbox.sh
tools/launchd/test_mission_stall.sh
tools/launchd/test_pi_controller_note.sh
tools/launchd/test_quorum_artifacts.sh
tools/launchd/test_secrets_scope.sh
tools/launchd/test_secrets_scope_spawn.sh
```

Snapshot-only proposed mirror:

```text
.agents/skills/mission-control/resources/gate-3-route.md
```

Gate results:

| Gate | RC | Result |
|---|---:|---|
| Bash syntax: all 18 shell files | 0 each | Informative |
| Zsh syntax: both M1 libraries | 0 each | Informative |
| Scope suite | 0 | Informative |
| Scope suite `--mutations` | 0 | Informative |
| New-file ShellCheck | 0 | Informative |
| Changed-file ShellCheck comparator | 0 | No new findings |
| Controller capacity fixture | 0 | Informative |
| Pi controller-note fixture | 0 | Informative |
| Quorum artifacts fixture | 0 | Informative for default arms |
| Codex controller fixture, complete | 1 | **UNINFORMATIVE UNDER SANDBOX** for real sandbox arms |
| Codex fixture without real Codex on PATH | 0 | Fixture arms pass; real sandbox arms skipped |
| Kill-tree fixture | 2 | **UNINFORMATIVE UNDER SANDBOX**: `ps` denied |
| Stall fixture | 1 | **UNINFORMATIVE UNDER SANDBOX**: live process inspection denied |
| Four pi runner fixtures | Not run | Their disposable Git writes conflict with the blanket prohibition |
| Skill sync / sync `--check` | 1 / 1 | **UNINFORMATIVE UNDER SANDBOX**: `.agents/` protected |
| `git diff --check`, current and plan base | 0 each | Informative |
| Billing/scanner byte comparison | 0 | Required lines and both scanner blocks unchanged |

The four unrun suites are `scripts/test_mission_pi_run.sh` and the launchd `test_mission_pi_run_commits.sh`, `test_mission_pi_run_sandbox.sh`, and `test_mission_pi_run_provider_quota.sh`. Their assertions remain intact.

All final mutation outcomes were killed:

| M2 mutant / selected arm | Killed |
|---|---|
| controller-codex | Yes |
| controller-pi | Yes |
| controller-claude | Yes |
| codex-ordering | Yes |
| retry-switch | Yes |
| codex-marker-root | Yes |
| deny-isolation | Yes |
| scoped-include-only | Yes |
| root-quoting | Yes |
| unknown-marker | Yes |
| detector | Yes |
| role-codex | Yes |
| role-claude | Yes |
| pi-runner | Yes |
| pi-bash-env | Yes |

The inherited M1 bank adds 53 killing outcomes, giving **68 kills and no survivors**. Its `multiple` mutant is equivalent under bash and killed under zsh. Changed lines and assertion evidence are banked in `.snap/evidence/secrets-scope-mutations.log`.

M3 was deliberately omitted. Integration arms run through the existing registered scope suite. The detector mutation disables its direct inventory in a scratch copy: merely scoping its environment would leave its intentional file access intact.

Two execution deviations are recorded candidly: an early detector control used default `/tmp` scratch before worktree-local `TMPDIR` was fixed and successfully retested; initial inspection calls used default shell startup, so automatic startup reads were not measured. No command explicitly inspected host credential contents, and no real credential values were printed.

Remaining risks are the unrun and sandbox-limited integration gates, live mirror reconstruction, M3 authentication/quorum observation, and host-hook installation. Without that hook, Claude/Codex zsh exposure remains. Pi’s initially stripped OpenAI key remains absent; retained credentials, credential files, deliberate re-sourcing, and native Agent-tool inheritance remain accessible. Missing inventories/helpers, invalid markers, readonly unsets, and scoped `include_only` intentionally refuse launch. Pin integrity remains an external control. This draft establishes neither fleet reach nor authorization to merge.