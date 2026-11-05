M1 is implemented and left uncommitted. Only the six authorized files changed. No host installation or production launch wiring was performed.

| File | Change | Lines |
|---|---|---:|
| [secrets-scope.sh](tools/launchd/lib/secrets-scope.sh) | New pure library | 139 |
| [secrets-scope-zshenv.sh](tools/launchd/lib/secrets-scope-zshenv.sh) | New conditional bridge | 9 |
| [test_secrets_scope.sh](tools/launchd/test_secrets_scope.sh) | New synthetic-home suite and mutation bank | 394 |
| [Design record](design_docs/planned/m-fleet-secrets-scope-per-lane.md) | New proposed draft | 140 |
| [make/test.mk](make/test.mk) | Added one suite near the end | 455 |
| [Changelog fragment](changelogs/unreleased/2026-11-05-secrets-scope-per-lane.md) | New proposed-draft entry | 8 |

The implemented allowlists are:

| Lane | Allowed names |
|---|---|
| `claude-controller` | `CLAUDE_CODE_OAUTH_TOKEN`, `OLLAMA_API_KEY`, `OPENROUTER_API_KEY`, `OPENAI_API_KEY` |
| `codex-controller` | Same four |
| `pi-controller` | Same four |
| `claude-role` | `CLAUDE_CODE_OAUTH_TOKEN` |
| `codex-role` | None |
| `pi-role` | `OLLAMA_API_KEY`, `OPENROUTER_API_KEY` |

`GOOGLE_API_KEY`, `GEMINI_AGENTS_API_KEY`, and `ANTHROPIC_API_KEY` are stripped in every lane.

The exact documented host line is:

```bash
if [ -n "${AILANG_SECRETS_LANE-}" ]; then if [ -n "${MISSION_DRIVER_ROOT-}" ] && [ -r "$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope-zshenv.sh" ] && . "$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope-zshenv.sh" 2>/dev/null; then :; else for _ailang_scope_key in ANTHROPIC_API_KEY CLAUDE_CODE_OAUTH_TOKEN GEMINI_AGENTS_API_KEY GOOGLE_API_KEY OLLAMA_API_KEY OPENAI_API_KEY OPENROUTER_API_KEY; do (unset "$_ailang_scope_key") 2>/dev/null && unset "$_ailang_scope_key" 2>/dev/null || :; done; while IFS= read -r _ailang_scope_key; do (unset "$_ailang_scope_key") 2>/dev/null && unset "$_ailang_scope_key" 2>/dev/null || :; done < <(/usr/bin/awk '/^[ \t]*(export[ \t]+)?[A-Za-z_][A-Za-z0-9_]*=/ { n=$0; sub(/^[ \t]*(export[ \t]+)?/, "", n); sub(/=.*/, "", n); if (!seen[n]++) print n }' "$HOME/.config/ailang/secrets.env" 2>/dev/null); unset _ailang_scope_key; printf '%s\n' 'secrets-scope: FAIL-CLOSED; scope hook unavailable or failed; mission shell refused' >&2; exit 78; fi; fi
```

Gate results:

| Gate | rc | Result |
|---|---:|---|
| Bash syntax, both libraries and suite | 0 | 3 checks passed |
| Zsh syntax, both libraries | 0 | 2 checks passed |
| Normal suite | 0 | **57 passed, 0 failed** |
| `--mutations` | 0 | **56 passed, 0 failed** |
| ShellCheck, three new shell files | 0 | No findings |
| `make check-changelog` | 0 | 3 fragments valid |
| `git diff --check` | 0 | Clean; new files also checked separately |
| `make check-file-sizes` | 0 | Passed; all six deliverables below 800 lines |
| `make test-launchd-drivers` | 2 | **UNINFORMATIVE UNDER SANDBOX**; 51/54 suites returned rc 0 |

The three obstructed existing suites were:

| Suite | rc | Observation |
|---|---:|---|
| `test_mission_stall.sh` | 1 | 49 assertions passed; live descendant snapshot requires blocked process inspection |
| `test_mission_kill_tree.sh` | 2 | Explicitly reported sandbox-blocked `ps` |
| `test_codex_controller_env.sh` | 1 | 14 assertions passed; four nested-sandbox assertions lacked output |

A separate benign Codex probe returned rc 71 with `sandbox_apply: Operation not permitted`. These existing suites were unchanged. The full gate preceded the final helper-name collision guard; the final targeted suite, mutation bank, syntax and lint checks were rerun afterward.

Every normal arm passed under **both bash 3.2.57 and zsh 5.9**. The mutation bank contains 27 mutants:

| Arm | Single-line killing mutation | Mutation result |
|---|---|---|
| Source-only | Append apply at library file scope | Killed both; rc 95 |
| `claude-controller` | Remove `OPENAI_API_KEY` from mapping | Killed both; rc 99 |
| `codex-controller` | Remove `OPENAI_API_KEY` | Killed both; rc 99 |
| `pi-controller` | Remove `OPENAI_API_KEY` | Killed both; rc 99 |
| `claude-role` | Return an empty allowlist | Killed both; rc 99 |
| `codex-role` | Add `OPENAI_API_KEY` | Killed both; rc 99 |
| `pi-role` | Remove `OPENROUTER_API_KEY` | Killed both; rc 99 |
| Unknown lane | Return an allowlist instead of rc 64 | Killed both; rc 107 |
| Future key | Omit `FUTURE_SCOPE_KEY` from parser output | Killed both; rc 121 |
| Helper-name collision | Skip clearing matching globals before declaring locals | Killed both; rc 93 |
| Assignment forms | Require `export`, rejecting plain assignments | Killed both; rc 111 |
| Deduplication | Remove `seen[n]++` guard | Killed both; rc 111 |
| Comment/decoy | Remove record anchor | Killed both; rc 111 |
| No evaluation | Source inventory inside apply | Killed both; sentinel assertion rc 112 |
| No disclosure | Print inventory records | Killed both; value-absence assertion |
| Parent mutation | Perform unset in a subshell | Killed both; caller assertion rc 121 |
| Missing inventory | Return 0 on inventory failure | Killed both; rc 114 |
| Failed unset | Skip emergency floor | Killed both; rc 116 |
| Bridge marker unset | Remove marker guard | Killed both; rc 117 |
| Bridge marker empty | Remove marker guard | Killed both; rc 117 |
| Host good root | Remove bridge apply | Killed both; retained-name mismatch |
| Host marker unset | Make marker guard unconditional | Killed both; expected body/rc mismatch |
| Host marker empty | Make marker guard unconditional | Killed both; expected body/rc mismatch |
| Host missing library | Return success from bridge | Killed both; expected refusal mismatch |
| Host missing root | Replace failure exit with successful return | Killed both; expected rc 78 mismatch |
| Host missing bridge | Replace failure exit with successful return | Killed both; expected rc 78 mismatch |
| Multiple disallowed names | Iterate unquoted `$names` | Killed under zsh, rc 100; equivalent for this bash fixture |

That is **53 shell-specific kills, zero survivors, and one fixture-equivalent bash result**, plus three extraction/syntax prechecks. Full per-mutant output is in [the mutation log](/tmp/fleet_i88_executor/checked-mutations.log); launchd findings are in [the gate log](/tmp/fleet_i88_executor/launchd-complete.log).

Implementation refinements were necessary for correctness: readonly unsets are probed before touching the caller because zsh otherwise aborts immediately; matching inventory globals are cleared before helper locals can shadow them; the host line suppresses bridge diagnostics so its refusal emits one FAIL-CLOSED line. The parent mutant uses a single-line subshell unset to reproduce the pipeline-loop failure.

No M1 deliverable remains unfinished. M2/M3 wiring, live dry-runs, independent evaluation, host installation and merging remain deferred as directed.