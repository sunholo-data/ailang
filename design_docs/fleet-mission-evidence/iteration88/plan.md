This sprint produces a **draft PR for D-FLEET-42 option A**, based on `9c7e997e0`. It does not merge, install a host hook, rotate credentials, or redact transcripts. D-FLEET-42 remains OPEN; unanswered still means option B.

The draft reduces accidental exposure through inherited environments. **Without Mark’s `~/.zshenv` change, Claude and Codex zsh tool shells retain the exposure measured in M3.** Pi’s non-login bash path benefits from the spawn strip, provided `BASH_ENV` is cleared. Retained credentials, credential files, deliberate re-sourcing, and native Agent-tool subagents’ inherited controller scope remain residual exposure.

Repository inspection was read-only. No host configuration, secret files, auth files, or session transcripts were opened.

**Allowlist and authentication decisions**

Controllers orchestrate other providers, query quota, and run quorum. Their allowlists therefore reflect those consumers, rather than just the controller’s inference provider.

| Lane | Allowed secrets.env names | Consumer justification |
|---|---|---|
| `claude-controller` | `CLAUDE_CODE_OAUTH_TOKEN`, `OLLAMA_API_KEY`, `OPENROUTER_API_KEY`, `OPENAI_API_KEY` | C, O, R, Q below |
| `codex-controller` | Same four | C, O, R, Q: cross-provider orchestration and quorum |
| `pi-controller` | Same four | C, O, R, Q: cross-provider orchestration and quorum |
| `claude-role` | `CLAUDE_CODE_OAUTH_TOKEN` | C: subscription inference |
| `codex-role` | None | Subscription auth is separate; existing billing guard at `tools/launchd/mission-control.sh:163–171`, probe at `tools/launchd/lib/lane-probe.sh:126` |
| `pi-role` | `OLLAMA_API_KEY`, `OPENROUTER_API_KEY` | P below: preserve the existing quota tool |
| Unknown lane | None; strip and refuse launch | Fail closed |

Each controller allowlist entry gets its own consumer comment:

| Reference | Kept key | Repository evidence |
|---|---|---|
| C | `CLAUDE_CODE_OAUTH_TOKEN` | Driver inference provenance at `tools/launchd/mission-control.sh:154`; quota fallback at `internal/mission/anthropic_quota.go:107`; Claude quorum subprocess at `internal/mission/quorum/claude_reviewer.go:69` |
| O | `OLLAMA_API_KEY` | Controller quota observation at `cmd/ailang/mission_quota_cmd.go:58` |
| R | `OPENROUTER_API_KEY` | Controller quota observation at `cmd/ailang/mission_quota_cmd.go:75` |
| Q | `OPENAI_API_KEY` | GPT quorum caller at `internal/mission/quorum/call.go:89,120–128`; model at `internal/modelreg/models.yml:577–582` |
| P | Both pi-role keys | `cmd/ailang/pi_assets/provider-quota.ts:35,69,119–120` reads both gauges |

Pi inference’s OpenRouter credential is documented as coming from its custom-provider file at `.claude/skills/mission-control/resources/gate-3-route.md:343`; its probe uses `--no-extensions` at `tools/launchd/lib/lane-probe.sh:153–156`. The two pi-role environment keys preserve quota observation, rather than being justified as inference credentials.

The quorum seats resolve as follows:

| Seat | Actual authentication |
|---|---|
| `gpt6-1-sol` | `OPENAI_API_KEY`; metered OpenAI API |
| `gemini-4-argon` | Vertex ADC pinned to `ailang-dev`; no Google API key fallback |
| `oc-glm-5-3` | Local Ollama daemon and device credential; no environment API key |
| `oc-kimi-k3` | Same Ollama route |
| `claude-sonnet-5@claude-p` | Claude subscription subprocess; preserve OAuth token |

Evidence: `internal/modelreg/models.yml:577,1633,2849,2869`, `internal/mission/quorum/call.go:120–128`, `internal/ai/factory/factory.go:207–231,252–272`, and `internal/mission/quorum/claude_reviewer.go:69–79`.

Consequently, every lane strips `ANTHROPIC_API_KEY`, `GEMINI_AGENTS_API_KEY`, and `GOOGLE_API_KEY`. The measured Gemini quorum seat does not need either Google-named key.

**An allowlist never restores a missing credential.** Keep the billing unset at driver line 171 unchanged. `OPENAI_API_KEY` will therefore remain absent from initial controller environments, including pi’s bash environment. With Mark’s hook installed, controller zsh shells can retain that key after the existing secrets source, enabling the GPT quorum seat. Pi’s bash path cannot acquire it through this strip. Document and report that seat’s absence; do not silently reload credentials, switch providers, or claim quorum authentication parity.

**Files and implementation anchors**

All anchors below refer to `9c7e997e0`; new files have no existing anchor.

| Path | Work |
|---|---|
| `tools/launchd/lib/secrets-scope.sh` — new | Pure function definitions: lane mapping, names parser, apply, and scoped command runner |
| `tools/launchd/lib/secrets-scope-zshenv.sh` — new | Marker-conditional bridge to the library through `MISSION_DRIVER_ROOT` |
| `scripts/mission_scoped_run.sh` — new | Bash wrapper for provider-pinned Codex and Claude role commands |
| `tools/launchd/test_secrets_scope.sh` — new | Synthetic-HOME suite, selectable arms, mutation targets |
| `tools/launchd/mission-control.sh:941–943` | Load pure helper alongside existing libraries |
| `tools/launchd/mission-control.sh:2020–2030` | Add scoped controller dry-run preview/probe before the existing exit |
| `tools/launchd/mission-control.sh:2445–2455` | Scope Codex child **before** building forwarded arguments |
| `tools/launchd/mission-control.sh:2502,2504–2508` | Scope pi and Claude children |
| `tools/launchd/lib/codex-env-args.sh:35–48,74–79` | Forward the exact marker; reject unsafe scoped forwarding |
| `scripts/mission_pi_run.sh:147,289–302` | Use runner root for the helper; apply `pi-role` inside the pi child |
| `.claude/skills/mission-control/resources/gate-3-route.md:134–179,229–246,293–309,340–410` | Short changes to Codex probes/runs, planner variants, Claude wrapper invocation, and pi instructions |
| `.agents/skills/mission-control/resources/gate-3-route.md` | Regenerate from canonical skill |
| `make/test.mk:75–76` | Add the new suite to `LAUNCHD_SUITES` |
| `design_docs/planned/m-fleet-secrets-scope-per-lane.md` — new | Installation text, limits, allowlists, commands, and evidence requirements |
| `changelogs/unreleased/2026-11-05-secrets-scope-per-lane.md` — new | Proposed-draft changelog fragment |

Existing harnesses extracting production spawn blocks need fixture updates so the new helper cannot read the real HOME. Preserve their assertions:

| Fixture path | Anchor |
|---|---|
| `tools/launchd/test_codex_controller_env.sh` | Extraction/setup at `:18–38` |
| `tools/launchd/test_controller_capacity.sh` | Extraction at `:49`; case environment at `:134–155` |
| `tools/launchd/test_mission_kill_tree.sh` | Extraction/setup at `:43–54` |
| `tools/launchd/test_mission_stall.sh` | Extraction/setup at `:25–41` |
| `tools/launchd/test_pi_controller_note.sh` | Fixture/extraction at `:30–49` |
| `tools/launchd/test_quorum_artifacts.sh` | Actual Codex forwarding at `:204–215` |
| `scripts/test_mission_pi_run.sh` | Fixture setup at `:12–19` |
| `tools/launchd/test_mission_pi_run_commits.sh` | Fixture setup at `:6–12` |
| `tools/launchd/test_mission_pi_run_sandbox.sh` | Fixture setup at `:6–15` |
| `tools/launchd/test_mission_pi_run_provider_quota.sh` | Fixture setup at `:23–32` |

No changes under `cmd/`, `internal/`, or `tools/pi-extensions/`.

**Decisions on directions 1–7**

1. **Accept, refine failure handling:** pure, bash-3.2/zsh-compatible library; unknown lanes strip everything and return nonzero; unreadable inventory or unset failures refuse launch.
2. **Accept, refine isolation:** apply inside each controller child subshell, preserving the parent driver’s quota and detector environment across retries and provider switches.
3. **Accept:** ship the repo bridge and exact host line; missing bridge/library refuses marked mission shells; unset marker does nothing.
4. **Accept, bounded scope:** cover provider-pinned role recipes through a small wrapper and the existing pi runner; native Agent-tool subagents retain controller scope and are explicitly documented.
5. **Accept:** synthetic fixtures throughout the new suite, production-block extraction, and named killing mutations.
6. **Refine:** current dry-run exits before controller selection/spawn; add a scoped probe and preview, then separately test real spawn blocks using fake providers.
7. **Accept:** fragment begins `### Added — Per-lane mission secret scope (proposed)` and names both the D-FLEET-42 ruling gate and required host installation.

**Library and driver contract**

- Parse only anchored `export NAME=` or `NAME=` records, allowing leading horizontal whitespace and valid shell identifiers.
- Emit deduplicated names from the parser. Never source the inventory, evaluate its values, retain values in shell variables, or print source records.
- Consume names with `while IFS= read -r name` in the invoking shell; avoid pipeline subshells and unquoted word splitting.
- `mc_secrets_scope_apply <lane>` emits one stderr summary containing lane, kept/stripped names, and counts. Here, **kept means exempted from unset**, not proof the credential exists.
- Known lane and successful application: rc 0. Unknown lane: strip all parsed names, rc 64. Inventory/unset failure: clear the seven measured names as an emergency floor, rc 78, and refuse the child.
- Library sourcing itself performs no reads, writes, exports, unsets, or output.
- Preserve the existing billing guard and all existing provider arguments, stdin handling, session handling, watchdog behavior, and transcript-scanner invocation.
- Keep driver additions in marked blocks; put reusable logic in the library.

The marker needs a deliberate Codex exception: **`AILANG_SECRETS_LANE` contains `SECRET`**, matching `codex-env-args.sh:44`. Add that exact name to selection and exempt only its validated six-lane value from the deny rule. Do not weaken the pattern for other names. Invalid marker values refuse forwarding.

Forward `MISSION_DRIVER_ROOT` through the existing `MISSION_*` route. For marked invocations, an `include_only` policy must refuse rather than merely warn: otherwise Codex can discard the marker and disable the hook. Keep ordinary unmarked behavior unchanged.

`tools/pi-extensions/sandbox/index.ts:152–156` passes no explicit `env` to `spawn("bash", ["-c", ...])`. Therefore inherited `BASH_ENV` **does matter**: non-interactive bash can read it. Clear it before pi controller and pi role spawn. This does not protect a parent shell that already executed its own startup files.

**Exact proposed host integration**

The bridge resolves:

```text
$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope.sh
```

It never falls back to an unpinned checkout. It returns nonzero if loading or applying the scope fails.

Mark would add the following **single physical line immediately after the existing secrets.env source**, preserving the later Anthropic billing unset. The draft stores this text in the design document, and the suite extracts that exact example for testing:

```bash
if [ -n "${AILANG_SECRETS_LANE-}" ]; then if [ -n "${MISSION_DRIVER_ROOT-}" ] && [ -r "$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope-zshenv.sh" ] && . "$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope-zshenv.sh"; then :; else unset ANTHROPIC_API_KEY CLAUDE_CODE_OAUTH_TOKEN GEMINI_AGENTS_API_KEY GOOGLE_API_KEY OLLAMA_API_KEY OPENAI_API_KEY OPENROUTER_API_KEY; while IFS= read -r _ailang_scope_key; do unset "$_ailang_scope_key" || exit 78; done < <(/usr/bin/awk '/^[ \t]*(export[ \t]+)?[A-Za-z_][A-Za-z0-9_]*=/ { n=$0; sub(/^[ \t]*(export[ \t]+)?/, "", n); sub(/=.*/, "", n); if (!seen[n]++) print n }' "$HOME/.config/ailang/secrets.env" 2>/dev/null); unset _ailang_scope_key; printf '%s\n' 'secrets-scope: FAIL-CLOSED; scope hook unavailable or failed; mission shell refused' >&2; exit 78; fi; fi
```

The emergency parser strips future inventory names even when the repo bridge is unavailable. Refusing the shell also covers an unreadable inventory. With no marker, the line performs no file access and leaves interactive shells unchanged.

**Role recipe changes**

Introduce this wrapper interface:

```bash
/bin/bash "$MISSION_DRIVER_ROOT/scripts/mission_scoped_run.sh" codex-role -- codex exec ...
/bin/bash "$MISSION_DRIVER_ROOT/scripts/mission_scoped_run.sh" claude-role -- claude-sub ...
```

The wrapper derives its authoritative root from its own location, exports marker/root, applies scope, and refuses on failure. The Codex arm uses the existing forwarding helper to append the marker, root, and existing permitted mission controls before executing Codex. Retain existing sandbox, model, spawn-override, output, and deadline arguments.

Use it for role probes and actual runs, including planner variants. Correct the stale prose saying the Codex executor requires `OPENAI_API_KEY`.

Apply `pi-role` inside the existing runner’s child, before line 302, and clear `BASH_ENV` there. Replace the raw pi invocation example with the guarded runner reference so the instructions do not advertise an unscoped alternative.

Native Agent-tool subagents cannot receive an independently scoped environment through these recipes. Document their inherited controller scope; provider-pinned Claude roles use `claude-role`.

Regenerate:

```bash
/bin/bash tools/launchd/sync-agents-skills.sh
/bin/bash tools/launchd/sync-agents-skills.sh --check
```

Review the generated diff: only the intended mission-control mirror should change.

**Test arms and killing mutations**

Every new-suite arm starts under a fresh temporary HOME with synthetic secrets. Set HOME, ZDOTDIR, CODEX_HOME, state paths, and startup controls before invoking code under test. Use fake providers and no network. Failure output must never print fixture contents.

| Arm | Required observation | One-line mutation that kills it |
|---|---|---|
| Source-only | Sourcing library changes nothing and prints nothing | Append an apply call at file scope |
| Six lane arms | Each lane retains exactly its table allowlist | Add/remove one name in that lane mapping |
| Unknown lane | All inventory names unset; rc 64 | Replace unknown-lane handling with controller allowlist |
| Future key | Additional synthetic key stripped | Replace parsed inventory with fixed seven-name list |
| Assignment forms | Plain/export/indented records counted once | Remove support for plain assignments |
| Deduplication | Repeated name counted once | Remove parser deduplication |
| Comment/decoy | Comments and `VALUE="export X=..."` create no extra name | Remove beginning-of-record anchor |
| No evaluation | Shell-looking value causes no sentinel side effect | Source inventory instead of parsing it |
| No disclosure | Sentinel values absent from stdout, stderr, logs and argv captures | Print an inventory record in summary |
| Parent mutation | Disallowed names actually disappear from caller environment | Pipe parser into the unset loop |
| Missing inventory | Emergency clearing and launch refusal | Return success on inventory read failure |
| Unknown marker | Forwarding/launch refused | Accept arbitrary marker strings |
| Codex marker/root | Both survive `.set`; other secret names remain denied | Remove marker forwarding |
| Deny isolation | `MISSION_X_SECRET` and token fixtures remain denied | Exempt all names containing `SECRETS` |
| Scoped `include_only` | Marked invocation refuses | Leave warning-only behavior |
| Root quoting | Root containing spaces survives TOML encoding | Concatenate root without encoder |
| Controller spawn ×3 | Fake child sees correct marker and allowed environment | Remove that provider’s apply call |
| Codex spawn ordering | Forward list built after marker/scope | Move argument construction before marker export |
| Retry/provider switch | Scope recalculated; parent credentials unchanged | Apply scope in the parent driver |
| Role wrapper ×2 | Codex/Claude child has role scope | Keep inherited controller marker |
| Pi runner | Real runner’s fake child has `pi-role` | Remove runner’s apply call |
| Pi `BASH_ENV` | Synthetic startup exporter does not restore a stripped key | Delete `unset BASH_ENV` |
| Dry-run preview | Scoped probe and preview run; controller body never runs | Remove scope call from dry-run path |
| Dry-run failure | Scoped fake auth failure appears and returns nonzero | Force probe status to zero |
| Detector regression | Detector still finds a synthetic stripped credential value | Route scanner through scoped launch wrapper |

Run each host-line arm under **both `/bin/zsh -c` and `/bin/bash -c`**:

| Host-line arm | Required observation | Killing mutation |
|---|---|---|
| Marker set | Synthetic startup source is followed by correct strip | Remove bridge apply |
| Marker absent/empty | Environment unchanged; hook never accessed | Apply scope unconditionally |
| Missing library | All fixture keys removed; command body cannot run | Return success when library is unreadable |
| Missing bridge/root | Emergency strip and exit 78 | Remove outer failure branch |
| Multiple parsed keys | All disallowed keys unset under zsh | Replace read loop with unquoted `$names` iteration |

Use a synthetic `.zshenv` containing the measured source ordering. Execute the actual new wrapper, actual bridge, exact documented host line, and extracted production spawn blocks. Assert extraction is nonempty. Mutation mode accepts scratch copies and an arm selector; no mutant edits the checkout.

A missing zsh interpreter is reported **UNINFORMATIVE**, never counted as a passing zsh arm. Controller acceptance requires those arms to run on the rig.

**Execution milestones**

| Milestone | Deliverable and acceptance | Estimate |
|---|---|---|
| M1 | Library, bridge, exact host text; parsing, disclosure and dual-shell failure tests | ½–1 day |
| M2 | Controller/role wiring, narrow Codex forwarding exception, pi startup clearing; actual spawn tests and fixture updates | 1 day |
| M3 | Mutation results, full gates, six dry-runs, changelog, independent evaluation, draft PR | ½–1 day |

Estimated total: **2–3 focused days**, dominated by security tests and existing harness integration. This is an engineering estimate, not measured velocity. Registry reuse is `none` for each milestone: this is host-shell launch plumbing using existing repo helpers, with no package dependency or registry/network lookup required.

**Gate commands**

Run from the proposed worktree. The executor records each result; the controller repeats acceptance gates outside the sandbox.

```bash
base=9c7e997e0f28b6f7b8e491818c2ee4e113390214

for f in \
  tools/launchd/lib/secrets-scope.sh \
  tools/launchd/lib/secrets-scope-zshenv.sh \
  scripts/mission_scoped_run.sh \
  tools/launchd/test_secrets_scope.sh \
  tools/launchd/mission-control.sh \
  scripts/mission_pi_run.sh \
  tools/launchd/test_codex_controller_env.sh \
  tools/launchd/test_controller_capacity.sh \
  tools/launchd/test_mission_kill_tree.sh \
  tools/launchd/test_mission_stall.sh \
  tools/launchd/test_pi_controller_note.sh \
  tools/launchd/test_quorum_artifacts.sh \
  scripts/test_mission_pi_run.sh \
  tools/launchd/test_mission_pi_run_commits.sh \
  tools/launchd/test_mission_pi_run_sandbox.sh \
  tools/launchd/test_mission_pi_run_provider_quota.sh
do
  /bin/bash -n "$f" || exit 1
done

/bin/zsh -n tools/launchd/lib/secrets-scope.sh
/bin/zsh -n tools/launchd/lib/secrets-scope-zshenv.sh

/bin/bash tools/launchd/test_secrets_scope.sh
/bin/bash tools/launchd/test_secrets_scope.sh --mutations

shellcheck \
  tools/launchd/lib/secrets-scope.sh \
  tools/launchd/lib/secrets-scope-zshenv.sh \
  scripts/mission_scoped_run.sh \
  tools/launchd/test_secrets_scope.sh

/bin/bash tools/launchd/sync-agents-skills.sh --check
make test-launchd-drivers
make check-changelog
git diff --check "$base"
make check-file-sizes
```

The new suite also syntax-checks the extracted host line under both shells. New-file ShellCheck must return zero.

For changed shell files, compare ShellCheck findings against base without treating shifted line numbers as new findings:

```bash
gate_tmp=$(mktemp -d)
mkdir -p "$gate_tmp/base"

git diff --name-only --diff-filter=M "$base" -- '*.sh' > "$gate_tmp/changed"

while IFS= read -r f; do
  mkdir -p "$gate_tmp/base/$(dirname "$f")"
  git show "$base:$f" > "$gate_tmp/base/$f"
done < "$gate_tmp/changed"

python3 - "$gate_tmp" <<'PY'
import collections, json, pathlib, subprocess, sys
root = pathlib.Path(sys.argv[1])
def findings(path):
    p = subprocess.run(["shellcheck", "-f", "json", str(path)],
                       capture_output=True, text=True)
    if p.returncode not in (0, 1):
        raise SystemExit("ShellCheck execution failure")
    return collections.Counter(
        (x["code"], x["level"], x["message"])
        for x in json.loads(p.stdout))
failed = False
for name in (root / "changed").read_text().splitlines():
    extra = findings(pathlib.Path(name)) - findings(root / "base" / name)
    for finding, count in extra.items():
        print(name, count, finding)
        failed = True
raise SystemExit(1 if failed else 0)
PY
```

Define the suite’s `--mutations` interface in this sprint. Bank each mutant’s changed line, selected arm, and failing assertion. Independent evaluation must test the proposed diff; no merge follows its PASS.

**Healthy and degraded dry-runs per controller lane**

The existing dry-run at driver line 2020 precedes `select_model` at line 2117. Extend it to:

1. Use the explicit `MISSION_MODEL` pin to identify the preview lane.
2. Run that provider’s bounded authentication probe inside the same scope used for spawn.
3. Exercise Codex marker/root forwarding for its preview.
4. Emit scope/probe evidence and exit before controller/session/watchdog launch.

Do not move normal controller selection or add side jobs.

The controller first selects an **idle, armed sibling** and verifies its kill switch/pidfile. This planner cannot establish that live state. Substitute its name below; do not use the active fleet profile.

```bash
patched_sha=$(git rev-parse HEAD)
dry_profile='<idle-armed-mission>'

for controller_pin in \
  claude:claude-opus-5-5 \
  codex:gpt-6.1-sol \
  pi:ollama/glm-5.3:cloud
do
  provider=${controller_pin%%:*}

  env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR-/tmp}" \
    AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED="$patched_sha" \
    MISSION_PROFILE="$dry_profile" MISSION_DRY_RUN=1 \
    MISSION_MODEL="$controller_pin" \
    /bin/bash tools/launchd/mission-control.sh \
    > "/tmp/fleet-i88-${provider}-healthy.log" 2>&1
  healthy_rc=$?

  env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR-/tmp}" \
    AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED="$patched_sha" \
    MISSION_PROFILE="$dry_profile" MISSION_DRY_RUN=1 \
    MISSION_MODEL="$controller_pin" \
    MISSION_EXECUTOR_MODEL=codex:secrets-scope-nonexistent-model \
    MISSION_PROBE_TIMEOUT=10 \
    /bin/bash tools/launchd/mission-control.sh \
    > "/tmp/fleet-i88-${provider}-degraded.log" 2>&1
  degraded_rc=$?

  printf '%s healthy_rc=%s degraded_rc=%s\n' \
    "$provider" "$healthy_rc" "$degraded_rc"
done
```

Required evidence in each log:

```text
secrets-scope: lane=<provider>-controller kept=<count>[names] stripped=<count>[names]
DRY RUN secrets-scope: lane=<provider>-controller marker=ok controller-probe=ok
```

Codex additionally reports:

```text
DRY RUN secrets-scope: codex-marker-forward=ok driver-root-forward=ok
```

Healthy requires `DRY RUN ok` with `lanes=ok`. Degraded requires the existing `lanes=DEGRADED(...)` ledger naming the injected invalid executor and its handoff. If the selected profile is already degraded, record that honestly; it is not healthy evidence.

A controller authentication failure must emit `controller-probe=failed rc=<n>` and return nonzero. These dry-runs demonstrate scoped probe authentication and routing behavior. They do not prove full role execution, quorum availability, or installation of the real host hook. Those limits belong beside the evidence.

**Risks and completion boundary**

- Removing a needed key can break authentication or quota observation. Scoped probes expose controller inference failures; synthetic missing-key arms prove the failure instrument. Full quorum and extension behavior still need explicit observation.
- Pi’s GPT quorum seat remains unavailable where `OPENAI_API_KEY` was removed and no zsh startup restores it. Report the missing seat rather than restoring a metered key.
- Without Mark’s host line, zsh tool-shell exposure remains as measured. The draft must say this in its description and changelog.
- Missing pin paths intentionally refuse marked tool shells. No fallback to a stale checkout.
- Allowed credentials remain inspectable, and Agent-tool subagents inherit controller allowances.
- The transcript detector keeps its direct file access and runs outside the strip. Its existing regression suite must remain green.
- Draft delivery does not establish fleet reach through `origin/dev`; merging and host installation remain separate ruling-controlled steps.

All post-change gates and live dry-runs are **UNINFORMATIVE UNDER SANDBOX** in this planning session: the proposed implementation does not yet exist. Completion is a reviewed, independently evaluated **draft PR**, with evidence and residuals attached, left unmerged for Mark’s ruling.