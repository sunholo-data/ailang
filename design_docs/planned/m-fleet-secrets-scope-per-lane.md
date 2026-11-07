# Per-lane mission secret scope

Status: **Proposed — draft for D-FLEET-42 option A; not merged; gated on Mark's ruling**

This draft reduces accidental exposure through inherited mission environments.
D-FLEET-42 remains open; unanswered still means option B. M1 supplies function
definitions, a startup bridge, tested host-line text and synthetic tests. It does
not install a host hook, wire launches, rotate credentials or redact transcripts.
The planner and executor inspected no host configuration or credential file;
the controller took the startup measurement below by key name only (iteration 88).

The controller's startup measurement (labelled M3 in the iteration-88 plan,
not milestone M3) found that Claude and Codex zsh tool shells re-source
`secrets.env` from `~/.zshenv:14`. Without Mark's host line those shells are
unchanged. Pi's `bash -c` tool path does not read `.zshenv`; the planned spawn
strip benefits that path only with `BASH_ENV` cleared (M2). M1 alone changes no
production lane. These are environment exposure controls, not a credential-file
sandbox. Retained credentials, deliberate re-sourcing, credential files and
native Agent-tool subagents inheriting controller scope remain exposed.

## Allowlist and evidence

Controllers orchestrate other providers, query quota and run quorum, so their
allowlist includes those consumers. Each name is exempted from unset only when
it appears in the inventory; an allowlist never restores a missing credential.

| Lane | Allowed secrets.env names | Evidence |
|---|---|---|
| `claude-controller` | `CLAUDE_CODE_OAUTH_TOKEN`, `OLLAMA_API_KEY`, `OPENROUTER_API_KEY`, `OPENAI_API_KEY` | C, O, R, Q |
| `codex-controller` | `CLAUDE_CODE_OAUTH_TOKEN`, `OLLAMA_API_KEY`, `OPENROUTER_API_KEY`, `OPENAI_API_KEY` | C, O, R, Q |
| `pi-controller` | `CLAUDE_CODE_OAUTH_TOKEN`, `OLLAMA_API_KEY`, `OPENROUTER_API_KEY`, `OPENAI_API_KEY` | C, O, R, Q |
| `claude-role` | `CLAUDE_CODE_OAUTH_TOKEN` | C |
| `codex-role` | None | Subscription auth; billing guard `tools/launchd/mission-control.sh:163-171`, probe `tools/launchd/lib/lane-probe.sh:126` |
| `pi-role` | `OLLAMA_API_KEY`, `OPENROUTER_API_KEY` | P: preserve quota gauges |
| Unknown | None; strip and return 64 | Refuse launch |

| Ref | Consumer and repository evidence |
|---|---|
| C | Claude subscription inference `tools/launchd/mission-control.sh:157`; quota fallback `internal/mission/anthropic_quota.go:107`; quorum subprocess `internal/mission/quorum/claude_reviewer.go:69` |
| O | Controller Ollama quota observation `cmd/ailang/mission_quota_cmd.go:58` |
| R | Controller OpenRouter quota observation `cmd/ailang/mission_quota_cmd.go:75` |
| Q | GPT quorum caller `internal/mission/quorum/call.go:89,120-128`; model `internal/modelreg/models.yml:577-582` |
| P | Both pi quota gauges `cmd/ailang/pi_assets/provider-quota.ts:35,69,119-120` |

Pi inference's OpenRouter credential comes from its custom-provider file
(`.claude/skills/mission-control/resources/gate-3-route.md:343`); its probe uses
`--no-extensions` (`tools/launchd/lib/lane-probe.sh:153-156`). The pi-role
allowlist preserves quota observation; it does not establish inference auth.

## Quorum-seat authentication

| Seat | Authentication |
|---|---|
| `gpt6-1-sol` | Metered OpenAI API, `OPENAI_API_KEY` |
| `gemini-4-argon` | Vertex ADC pinned to `ailang-dev`, `env_var: ""`; no Google API key fallback |
| `oc-glm-5-3` | Local Ollama daemon and device credential; no environment key |
| `oc-kimi-k3` | Same Ollama route |
| `claude-sonnet-5@claude-p` | Claude subscription subprocess, OAuth token |

Evidence: `internal/modelreg/models.yml:577,1633,2849,2869`,
`internal/mission/quorum/call.go:120-128`,
`internal/ai/factory/factory.go:207-231,252-272`, and
`internal/mission/quorum/claude_reviewer.go:69-79`.
The Gemini auth route is controller-verified. `GOOGLE_API_KEY` is not retained;
`GEMINI_AGENTS_API_KEY` belongs only to the managed-agents executor lane, which
no fleet chain routes to. Every lane strips those names and `ANTHROPIC_API_KEY`
when `secrets.env` assigns them: apply strips only names the inventory assigns
(the failure path clears the seven measured names).

The driver's existing billing unset remains in place. `OPENAI_API_KEY` therefore
stays absent from initial controller environments, including pi's bash path.
The host hook can retain it after zsh's existing secrets source; pi cannot
acquire it through an unset-only scope. Report that missing GPT seat rather
than silently reloading credentials, switching providers or claiming parity.

## Library contract

`tools/launchd/lib/secrets-scope.sh` contains pure function definitions, usable
from bash 3.2.57 and zsh 5.9. Sourcing it performs no actions. The names parser
reads anchored shell identifier assignments with optional `export` and leading
horizontal whitespace, deduplicates names and never evaluates or prints values.
Apply unsets parsed disallowed names in the calling shell and reports one
names-only stderr line. `kept` means exempted from unset, not present.

Unknown lanes strip every parsed name and return 64. Unreadable inventories or
failed unsets attempt to clear the seven measured names and return 78. Readonly
variables cannot be cleared; callers must refuse launch on failure. Helpers
use local variables and do not leave helper state set after apply returns.
Parsed names that match internal helper names are cleared before those locals
are declared, so local shadowing cannot preserve a future inventory key.

The bridge sources the library only through `MISSION_DRIVER_ROOT`; it never
falls back to another checkout. An unset or empty marker returns 0 without
accessing the library. All other failures propagate a nonzero return.

## Proposed host installation

Only after Mark's ruling: add the following one physical line immediately after
the existing secrets.env source in `~/.zshenv`, BEFORE the existing Anthropic
unset. This document is the single source of the tested host line.

<!-- SECRETS-SCOPE HOST LINE START -->
```bash
if [ -n "${AILANG_SECRETS_LANE-}" ]; then if [ -n "${MISSION_DRIVER_ROOT-}" ] && [ -r "$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope-zshenv.sh" ] && . "$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope-zshenv.sh" 2>/dev/null; then :; else for _ailang_scope_key in ANTHROPIC_API_KEY CLAUDE_CODE_OAUTH_TOKEN GEMINI_AGENTS_API_KEY GOOGLE_API_KEY OLLAMA_API_KEY OPENAI_API_KEY OPENROUTER_API_KEY; do (unset "$_ailang_scope_key") 2>/dev/null && unset "$_ailang_scope_key" 2>/dev/null || :; done; while IFS= read -r _ailang_scope_key; do (unset "$_ailang_scope_key") 2>/dev/null && unset "$_ailang_scope_key" 2>/dev/null || :; done < <(/usr/bin/awk '/^[ \t]*(export[ \t]+)?[A-Za-z_][A-Za-z0-9_]*=/ { n=$0; sub(/^[ \t]*(export[ \t]+)?/, "", n); sub(/=.*/, "", n); if (!seen[n]++) print n }' "$HOME/.config/ailang/secrets.env" 2>/dev/null); unset _ailang_scope_key; printf '%s\n' 'secrets-scope: FAIL-CLOSED; scope hook unavailable or failed; mission shell refused' >&2; exit 78; fi; fi
```
<!-- SECRETS-SCOPE HOST LINE END -->

The line sources the bridge with `2>/dev/null`, so its names-only
`lane= kept= stripped=` audit line is not visible on this path; only the
FAIL-CLOSED line is. No marker means no file access by this line. With a marker, successful bridge
application permits the command body. A missing root, bridge, library, or other
failure triggers emergency clearing of the measured seven and every parsed
inventory name, prints a FAIL-CLOSED message and exits 78 before the body.
Refusing the shell is intentional. Failed emergency unsets still refuse launch.

## Follow-ups — NOT YET BUILT

M2 wires the driver children and adds provider-pinned role wrappers through
`scripts/mission_scoped_run.sh`. Scope applies within each child so the parent
retains quota/detector credentials through retries and provider changes.
Codex forwarding must carry `MISSION_DRIVER_ROOT` and the exact validated
`AILANG_SECRETS_LANE` marker before argument construction. That marker contains
`SECRET`: exempt only that exact six-lane name/value from the current deny
rule, never all names containing `SECRETS`. Scoped `include_only` must refuse
launch rather than drop the marker. Pi controller and role paths clear
`BASH_ENV`. Existing harness extraction fixtures get synthetic HOME/startup
isolation; provider arguments, billing unset and detector access remain intact.
Role recipes and their generated mirror are M2 work as well.

M3 performs healthy/degraded dry-runs per controller lane, actual fake-provider
spawn observations, mutation testing of the complete wiring, full gates and
independent evaluation. M1's mutation bank covers only the library, bridge and
host line. Any draft PR remains unmerged pending Mark's ruling; successful
local checks establish neither fleet reach nor real host installation.

## Risks

A removed consumer key can break authentication or quota observation; controller
probes and explicit quorum/extension observations are still needed in M2/M3.
Missing pinned checkouts deliberately refuse marked shells. The host hook must
follow the secrets source and precede billing unsets; later re-sourcing bypasses
this control. Allowed keys and credential files remain accessible. Native
Agent-tool subagents inherit controller scope. The transcript detector must
remain outside the strip, retaining its direct file access and regression tests.
A mutable or malicious driver root can replace the helper: pin integrity is a
separate launch control. No host changes are performed by this draft.
