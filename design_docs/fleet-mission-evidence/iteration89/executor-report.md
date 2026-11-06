M2 executor report — fleet iteration 89

M2a and M2b are built, with synthetic acceptance arms green. M2c canonical prose and status updates are built; its proposed generated mirror is supplied in the snapshot, but applying it in the live worktree was denied by the sandbox. Full external acceptance is not claimed. This remains a proposed draft on fleet/i89-dfleet42-m2, stacked on #2141, uncommitted and unmerged, pending Mark’s D-FLEET-42 ruling.

Cumulative snapshots: .snap/M2a (10 source files), .snap/M2b (16), .snap/M2c (20, including the proposed mirror). Each contains manifest.txt and SHA256SUMS. The final snapshot matches all 19 current repository source files byte-for-byte; its .agents mirror matches the canonical .claude resource. Snapshot identity check rc 0. Both new scripts are executable. Generated evidence and snapshot files live under .snap/.

Files changed (19 in the worktree; the final entry is proposed snapshot content only):

- .claude/skills/mission-control/resources/gate-3-route.md
- changelogs/unreleased/2026-11-05-secrets-scope-per-lane.md
- design_docs/planned/m-fleet-secrets-scope-per-lane.md
- scripts/mission_pi_run.sh
- scripts/test_mission_pi_run.sh
- tools/launchd/lib/codex-env-args.sh
- tools/launchd/mission-control.sh
- tools/launchd/test_codex_controller_env.sh
- tools/launchd/test_controller_capacity.sh
- tools/launchd/test_mission_kill_tree.sh
- tools/launchd/test_mission_pi_run_commits.sh
- tools/launchd/test_mission_pi_run_provider_quota.sh
- tools/launchd/test_mission_pi_run_sandbox.sh
- tools/launchd/test_mission_stall.sh
- tools/launchd/test_pi_controller_note.sh
- tools/launchd/test_quorum_artifacts.sh
- tools/launchd/test_secrets_scope.sh
- tools/launchd/test_secrets_scope_spawn.sh
- scripts/mission_scoped_run.sh
- .agents/skills/mission-control/resources/gate-3-route.md — snapshot only; live .agents path protected

Gate results (final results unless explicitly described):

| Gate | RC | Informative? |
|---|---:|---|
| /bin/bash -n on every changed shell file and the two M1 libraries (18 files) | 0 each | Yes |
| /bin/zsh -n secrets-scope.sh | 0 | Yes |
| /bin/zsh -n secrets-scope-zshenv.sh | 0 | Yes |
| test_secrets_scope.sh | 0 | Yes; both interpreters, all 15 M2 arms |
| test_secrets_scope.sh --mutations | 0 | Yes; 53 M1 + 15 M2 killing outcomes |
| test_codex_controller_env.sh, complete suite | 1 | UNINFORMATIVE UNDER SANDBOX for real Codex e2e; fake-provider/forwarding assertions pass |
| test_codex_controller_env.sh, PATH without real Codex | 0 | Unit/fixture arms informative; existing real-Codex arms skipped, not counted as proof |
| test_controller_capacity.sh | 0 | Yes |
| test_mission_kill_tree.sh | 2 | UNINFORMATIVE UNDER SANDBOX: ps denied |
| test_mission_stall.sh | 1 | UNINFORMATIVE UNDER SANDBOX: live descendant CPU snapshot requires denied ps |
| test_pi_controller_note.sh | 0 | Yes |
| test_quorum_artifacts.sh | 0 | Yes for default fixture arms; optional external writer not supplied |
| scripts/test_mission_pi_run.sh | Not run; no RC | No: unchanged fixture performs prohibited Git writes |
| test_mission_pi_run_commits.sh | Not run; no RC | No: unchanged fixture performs prohibited Git writes |
| test_mission_pi_run_sandbox.sh | Not run; no RC | No: unchanged fixture performs prohibited Git writes |
| test_mission_pi_run_provider_quota.sh | Not run; no RC | No: unchanged fixture performs prohibited Git writes |
| ShellCheck: both M1 libraries, new role wrapper, main scope suite and new spawn suite | 0 | Yes |
| Changed-file ShellCheck comparator against plan base 9c7e997e0 | 0 | Yes; no new findings |
| sync-agents-skills.sh | 1 | UNINFORMATIVE UNDER SANDBOX: .agents writes denied |
| sync-agents-skills.sh --check | 1 | UNINFORMATIVE UNDER SANDBOX: only mission-control mirror stale after denied sync |
| git diff --check | 0 | Yes |
| git diff --check 9c7e997e0f28b6f7b8e491818c2ee4e113390214 | 0 | Yes |
| Billing lines 163–176 and both scanner blocks compared to HEAD | 0 | Yes; byte-identical |
| Standalone ps permission probe | 126 | UNINFORMATIVE UNDER SANDBOX: Operation not permitted |
| Standalone real Codex sandbox probe with synthetic config | 71 | UNINFORMATIVE UNDER SANDBOX: sandbox_apply: Operation not permitted |

The four pi fixture suites were updated without removing assertions. Their actual execution is withheld under the explicit “NO git write operations of any kind” rule: they call init/add/commit/worktree in disposable repositories. No exemption was supplied. The new pi scope arms execute the actual runner using fake providers and read-only fake Git metadata; they verify scope, startup clearing and failed-apply refusal, not real commit/fingerprint behavior. No Git writes were performed.

Final mutation table (mutant and selected arm have the same name):

| Mutant / arm | Shells | Killed? |
|---|---|---|
| source | bash, zsh | Yes |
| lane-claude-controller | bash, zsh | Yes |
| lane-codex-controller | bash, zsh | Yes |
| lane-pi-controller | bash, zsh | Yes |
| lane-claude-role | bash, zsh | Yes |
| lane-codex-role | bash, zsh | Yes |
| lane-pi-role | bash, zsh | Yes |
| unknown | bash, zsh | Yes |
| future | bash, zsh | Yes |
| helper-collision | bash, zsh | Yes |
| forms | bash, zsh | Yes |
| dedup | bash, zsh | Yes |
| decoy | bash, zsh | Yes |
| noeval | bash, zsh | Yes |
| disclosure | bash, zsh | Yes |
| parent | bash, zsh | Yes |
| missing | bash, zsh | Yes |
| unset-failure | bash, zsh | Yes |
| bridge-unset | bash, zsh | Yes |
| bridge-empty | bash, zsh | Yes |
| host-good | bash, zsh | Yes |
| host-unset | bash, zsh | Yes |
| host-empty | bash, zsh | Yes |
| host-missing-lib | bash, zsh | Yes |
| host-missing-root | bash, zsh | Yes |
| host-missing-bridge | bash, zsh | Yes |
| multiple | zsh | Yes |
| controller-codex | bash | Yes |
| controller-pi | bash | Yes |
| controller-claude | bash | Yes |
| codex-ordering | bash | Yes |
| retry-switch | bash | Yes |
| codex-marker-root | bash | Yes |
| deny-isolation | bash | Yes |
| scoped-include-only | bash | Yes |
| root-quoting | bash | Yes |
| unknown-marker | bash | Yes |
| detector | bash | Yes |
| role-codex | bash | Yes |
| role-claude | bash | Yes |
| pi-runner | bash | Yes |
| pi-bash-env | bash | Yes |

The `multiple` mutant is equivalent under bash’s splitting for this fixture; that outcome is explicitly labelled EQUIVALENT, not a kill. Its zsh arm kills it. Final total: 68 kills, no survivors. The banked log .snap/evidence/secrets-scope-mutations.log contains each changed line, target, selected arm and failing assertion for the M2 mutants; M1 mutant assertion codes are also captured. An intermediate mutation constructor matched the wrong nested conditional after a compatibility fix; anchoring it to the include_only branch corrected that constructor, and the final complete mutation run is green.

Deliberately omitted: all M3 dry-run preview/probe code, the six live dry-runs, broad M3 gates and independent evaluation; host-hook installation, credential rotation, transcript redaction, commits, PR changes and merge. No implementation edits were made under cmd/, internal/, tools/pi-extensions/, .pi/extensions/, .github/ or go.mod. The new mutation harness copies existing extension assets into disposable scratch roots solely to run the real pi runner.

Plan deviations and execution limits:

- Integration arms live in test_secrets_scope_spawn.sh, invoked by the existing registered test_secrets_scope.sh in normal and mutation modes; the make launchd route therefore reaches them without double registration.
- The detector mutant disables its direct file inventory in a scratch copy. Wrapping that detector in a scoped environment is equivalent while its intentional direct inventory access remains intact, so that proposed mutation would not test the invariant. The arm executes the real driver scanner invocation and detects a synthetic key stripped from the controller child.
- Unencodable scope controls refuse marked calls; unmarked calls retain their original skip behavior. This preserves ordinary helper compatibility.
- The sandbox denied live .agents synchronization. The complete intended mission-control mirror is included at .snap/M2c/.agents/skills/mission-control/resources/gate-3-route.md for controller reconstruction; no other mirror content changes are proposed.
- An early detector control inherited no TMPDIR and used the scanner’s default /tmp scratch, which its cleanup removed. This violated the worktree-only temporary-write rule. Every integration child now receives a worktree-local TMPDIR; final scope and mutation runs pass with that correction. No command explicitly inspected host configuration or credential-file contents. Initial inspection calls used the tool’s default shell startup before subsequent calls switched to /bin/bash with login disabled; automatic startup reads during those initial calls were not measured and cannot be ruled out. All tests used synthetic homes; no real credential values were printed.

Remaining risks:

- Acceptance still requires the four real-Git pi suites, process-tree/stall gates and real Codex sandbox checks outside this sandbox, followed by a successful mirror sync/check.
- Without Mark’s host line, Claude/Codex zsh tool shells can restore the measured secret exposure. This draft is not installed and establishes no fleet reach.
- The unchanged billing guard removes OPENAI_API_KEY initially. Pi cannot restore it through an unset-only scope, so its GPT quorum seat remains unavailable; no auth parity, live quorum or quota-extension behavior is claimed.
- Retained credentials and credential files remain accessible. Deliberate re-sourcing and native Agent-tool subagents’ inherited controller scope remain residual exposure.
- Missing/unreadable inventories, readonly unsets, invalid markers, unavailable pinned helpers, and scoped include_only intentionally refuse launch. Actual provider/config compatibility needs M3 observation.
- Pin integrity remains external: a mutable or malicious driver root can replace the helper. Scope only strips inventory names; it never restores missing keys and does not protect a parent shell that already ran startup code.
- D-FLEET-42 remains open. No merge or host install follows from these synthetic results.
