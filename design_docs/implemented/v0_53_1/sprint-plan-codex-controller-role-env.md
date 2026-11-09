# Sprint Plan: codex controller role-env delivery

**Ticket**: `agent-tool:mission-role-pins-unavailable` (blocking=all; World iterations 239–240
and fleet iteration 24 each lost a whole slot)
**Mission**: fleet, iteration 25. Branch `fleet/iter25-codex-controller-role-env` (base origin/dev `fcce2394c`)
**Size**: one milestone, about half a session
**Policy class**: harness bug fix confined to the codex controller call site. It does **not**
touch routing, lane order, quota, rations or billing guards (see "Policy boundary").

## Problem (measured)

When the driver falls back to a codex controller, the invocation in
`tools/launchd/mission-control.sh` `_mc_run_once` (line 2275 at `fcce2394c`) is

```
codex exec --skip-git-repo-check --dangerously-bypass-approvals-and-sandbox \
  --model "$MODEL" -C "$REPO" "$PROMPT"
```

Codex builds the environment for every shell tool call from `shell_environment_policy`. Since
2026-10-06 13:02 the rig's `~/.codex/config.toml` sets `inherit = "core"`, plus a `.set` table
holding two rig variables (one of them a credential). Under `core`, the controller's shells see
only `HOME LOGNAME PATH SHELL TMPDIR USER`, the two `.set` vars and codex's own
`CODEX_SANDBOX*`. Everything the driver exported is lost: every `MISSION_*`, the
`AILANG_DRIVER_*` pin vars, the messaging-plane switch and **the HD-4 scope guard's
`GIT_CONFIG_*` entries** (see finding 2). `resolve-role-spawn.sh` then fails closed for every
role, so the slot is lost.

## Findings from planning (beyond the ticket)

1. **The `.set` route works and leaves the rig policy alone.** `-c shell_environment_policy.set.NAME="…"`
   adds a single variable and *merges* with the rig's `.set` table. The rig's two vars survived
   in every arm. It works under `inherit=core`, `all` and `none` (arms E, G, H in the log).
2. **The HD-4 mission scope guard has been OFF for codex controllers since 2026-10-06.** The
   driver enables it with `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_n=core.hooksPath`/`GIT_CONFIG_VALUE_n`
   (mission-control.sh:2213-2215). Under `inherit=core` a codex controller's
   `git config core.hooksPath` returns the shared clone's `.git/hooks`, so a codex controller
   can push paths its mission is forbidden to push. Forwarding the entry restores the guard
   (arm I). This is a restoration of an existing control, not a policy change, so it is in scope.
3. **`include_only` beats `.set`.** Codex applies `include_only` *after* `.set` (arm F: a
   `.set` var is stripped by `include_only=["HOME","PATH"]`, and so are the rig's own `.set`
   vars). The rig config has no `include_only` today. The design does not override
   `include_only`, because resetting it could widen exposure on a rig that used it as a filter.
   It logs a loud warning instead (see the helper's contract below). This is the one rig-config
   shape the fix cannot be independent of. It is recorded, not hidden.
4. **bash 3.2 traps that would bite this exact change** (both measured under `/bin/bash` 3.2.57):
   - `"${arr[@]}"` on an empty array under `set -u` aborts: `a[@]: unbound variable`, rc 127.
     The driver runs `set -uo pipefail` (line 38). The call site must use
     `${MC_CODEX_ENV_ARGS[@]+"${MC_CODEX_ENV_ARGS[@]}"}`.
   - `. <(…)` (sourcing a process substitution) silently defines nothing in bash 3.2. Tests
     must source the helper file directly.
5. **The driver's dry-run never reaches the call site.** `MISSION_DRY_RUN=1` exits at line
   1926, before the role env is exported (2180-2203) and before `_mc_run_once` (2274).
   `MISSION_PRINT_CONFIG=1` exits even earlier. See "Dry-run note".

## Design choice

**Chosen: per-variable `-c shell_environment_policy.set.NAME=<TOML basic string>`, built by a
bash-3.2 helper from a name allowlist plus a secret-name denylist, evaluated against the
driver's live exported environment at call time.**

| Option | Rejected / chosen because |
|---|---|
| B. `-c shell_environment_policy.inherit=all` | Rejected. It newly exposes every secret in the driver env: `AILANG_REGISTRY_API_KEY`, `OPENAI_API_KEY`, `OPENROUTER_API_KEY`, `GOOGLE_API_KEY`, `ZAI_API_KEY`, `OLLAMA_API_KEY`, `DOCPARSE_API_KEY`, `LYCEUM_API_KEY`, `TYPESAFE_API_KEY`, `SENTRY_ACCESS_TOKEN`, `CLAUDE_CODE_MESSAGING_TOKEN` (arm B, names only). Violates requirement 2. |
| C. `inherit=all` + `include_only=[…,"MISSION_*","AILANG_*"]` | Rejected. The `AILANG_*` glob lets `AILANG_REGISTRY_API_KEY` through, and `include_only` strips the rig's `.set` vars (`CLAUDE_CODE_OAUTH_TOKEN`, which a nested `claude-sub` recipe may need). Violates requirement 2 twice. |
| D. keep `inherit` + `include_only` | Rejected. `include_only` only filters, it cannot add. |
| **E. per-var `.set` overrides** | **Chosen.** Adds exactly the named vars, leaves `inherit`/`.set`/`exclude` untouched, works for any `inherit` value (arms E/G/H), and keeps the rig's `.set` vars (arm E). The cost is TOML escaping, which arm RT proves round-trips backticks, `"`, `\`, `$`, newline, tab, `'`, `→`, `**` byte-exactly under bash 3.2. The real `MISSION_ROUTING_NOTE` also round-trips (arm R). |
| Write a temp `CODEX_HOME`/profile file | Rejected. A second config source drifts from the rig's (auth, model providers, MCP), and moving `CODEX_HOME` would detach the controller from its OAuth `auth.json`. |

Values travel in argv, so they are visible to `ps` for the life of the controller. That is
acceptable only because the denylist keeps secrets out. The same values are already readable
via `ps -E` from the driver's own environment.

## Milestone M1: forward driver env into the codex controller

### Files

| File | Change |
|---|---|
| `tools/launchd/lib/codex-env-args.sh` | **NEW** (~80 lines). Pure functions, no side effects at source time. |
| `tools/launchd/mission-control.sh` | Source the helper next to `lane-probe.sh` (line 885 pattern, `$MC_DRIVER_ROOT/tools/launchd/lib/…`). Wrap the codex branch of `_mc_run_once` in `# --- CODEX CONTROLLER EXEC START ---` / `END` markers (the house pattern the heartbeat tests extract with awk), call the builder, and splice the array into `codex exec`. Nothing else changes. |
| `tools/launchd/test_codex_controller_env.sh` | **NEW** suite (arms below). |
| `make/test.mk` | One line in `test-launchd-drivers`: `@$(LAUNCHD_SUITE) tools/launchd/test_codex_controller_env.sh` |
| `changelogs/unreleased/2026-10-07-codex-controller-role-env.md` | `### Fixed`: codex controller role env + scope guard restored. |

### Helper contract (`lib/codex-env-args.sh`)

```bash
# mc_toml_basic_string VALUE -> prints a TOML basic string ("…") on stdout, rc 0.
#   Escapes, in this order: \ -> \\ , " -> \" , LF -> \n , TAB -> \t , CR -> \r.
#   UTF-8 bytes pass through untouched. If any other control character remains
#   (case "$v" in *[[:cntrl:]]*), print nothing and return 1. The caller skips that var LOUDLY.
#   Use bash parameter expansion only. No sed/awk: BSD awk counts bytes, and sed can't
#   see newlines.
#   Proven form (bash 3.2.57): local bs='\'; v=${v//"$bs"/"$bs$bs"}; v=${v//\"/"$bs\""};
#   v=${v//$'\n'/"${bs}n"}; v=${v//$'\t'/"${bs}t"}; v=${v//$'\r'/"${bs}r"}

# mc_codex_env_args -> fills the GLOBAL indexed array MC_CODEX_ENV_ARGS with
#   -c shell_environment_policy.set.<NAME>=<toml>   pairs; writes ONE summary line to stderr:
#   "codex-env: forwarded=<n> denied=<names…> skipped=<names…>"  (NAMES ONLY, never values).
#   Returns 0 even when nothing is forwarded (the resolver then fails closed loudly as today).
#
#   Selection, over `compgen -e` (exported names only; bash-3.2 builtin, newline-safe,
#   unlike parsing `env`):
#     ALLOW (name):  MISSION_*  AILANG_DRIVER_*  AILANG_STORAGE_MESSAGING
#                    AILANG_MESSAGES_PROJECT  AILANG_MISSION_REGISTRY  AILANG_STATE_DIR
#                    CONTROLLER_PROVIDER  CONTROLLER_ID
#     DENY  (name, applied AFTER allow, wins): *KEY* *TOKEN* *SECRET* *PASSWORD* *PASSWD*
#                    *CREDENTIAL* *AUTH* *COOKIE* *PRIVATE*
#     Deliberately NOT a bare AILANG_* glob: that is exactly how AILANG_REGISTRY_API_KEY
#     leaked in option C.
#
#   Scope-guard entry (finding 2), handled separately because its NAME contains "KEY":
#     scan i in 0..GIT_CONFIG_COUNT-1 for the entry whose KEY_i is literally core.hooksPath,
#     and forward it RENUMBERED as GIT_CONFIG_COUNT=1, GIT_CONFIG_KEY_0=core.hooksPath,
#     GIT_CONFIG_VALUE_0=<its value>. Other GIT_CONFIG entries are NOT forwarded: under
#     inherit=core they are absent today, and a VALUE_n may be an http.extraHeader credential.
#     The only KEY-named var this adds holds the literal string "core.hooksPath".
#
#   include_only tripwire (finding 3): if ${CODEX_HOME:-$HOME/.codex}/config.toml has a line
#   matching ^[[:space:]]*include_only, append "WARNING include_only-present: forwarded vars
#   may be stripped" to the summary line. Warn only. Never rewrite the rig config.
```

Call site (shape only):

```bash
# --- CODEX CONTROLLER EXEC START ---
  if [ "$CONTROLLER_PROVIDER" = "codex" ]; then
    MC_CODEX_ENV_ARGS=()
    mc_codex_env_args 2>>"$LOG"
    codex exec ${MC_CODEX_ENV_ARGS[@]+"${MC_CODEX_ENV_ARGS[@]}"} --skip-git-repo-check \
      --dangerously-bypass-approvals-and-sandbox \
      --model "$MODEL" -C "$REPO" "$PROMPT" >>"$LOG" 2>&1 &
# --- CODEX CONTROLLER EXEC END ---
```

The builder runs inside `_mc_run_once`, after `MISSION_ATTEMPT` is exported, so the per-attempt
value is the one forwarded. `-c` goes **before** the prompt positional.

### Test arms (`test_codex_controller_env.sh`, bash 3.2, runs under `lib/suite-env.sh`'s clean env)

Each arm sets its own variables (never inherits). A stub `codex` that dumps argv one element
per NUL (`printf '%s\0' "$@"`) sits first on PATH for arms A–D. The call-site block is
extracted from the real driver with `awk '/# --- CODEX CONTROLLER EXEC START ---/,/END ---/'`,
so the test exercises the driver's own text, not a copy.

| Arm | Proves |
|---|---|
| `argv-carries-mission-vars` (a) | With `MISSION_NAME`, `MISSION_DESIGNER_MODEL`, `MISSION_EVALUATOR_MODEL`, `MISSION_OVER_RATION`, `AILANG_MESSAGES_PROJECT`, `AILANG_DRIVER_PINNED` exported, the extracted block hands the stub one `-c shell_environment_policy.set.<NAME>=…` per var, all before the `$PROMPT` positional. |
| `argv-scope-guard-forwarded` | With `GIT_CONFIG_COUNT=2`, `KEY_0=http.extraHeader`/`VALUE_0=Authorization: Bearer x`, `KEY_1=core.hooksPath`/`VALUE_1=/g`: argv carries `GIT_CONFIG_COUNT="1"`, `GIT_CONFIG_KEY_0="core.hooksPath"`, `GIT_CONFIG_VALUE_0="/g"`, and the bearer string appears nowhere in argv. |
| `secret-names-denied` (c) | `FOO_API_KEY`, `MISSION_TEST_TOKEN`, `AILANG_REGISTRY_API_KEY`, `MISSION_X_SECRET` and `MISSION_AUTH_COOKIE` exported. None of their names or values appear in argv, and the stderr summary lists the MISSION_* ones under `denied=` (names only). |
| `empty-env-set-u-safe` | With no allowlisted var exported, under `set -u`, the block still invokes the stub (rc 0, argv starts `exec`). Guards bash-3.2 trap 4a. |
| `toml-encode-unit` | `mc_toml_basic_string` on the hostile value (backtick, `"`, `\`, `$HOME` literal, LF, TAB, `'`, `→`, `**`) yields the exact expected literal. A value with `\001` returns 1 and lands under `skipped=`. |
| `roundtrip-codex-e2e` (b) | **Only if `command -v codex`**: `codex sandbox "${MC_CODEX_ENV_ARGS[@]}" -- /usr/bin/printenv MISSION_RT` returns the hostile value byte-exactly (append `x` sentinel to preserve the trailing newline, then strip it), and `-- /usr/bin/env` names include every forwarded name. When codex is absent it prints `SKIP roundtrip-codex-e2e: codex not on PATH (CI) — run on the rig to prove arm (b)` to **stderr and stdout**, and the summary line counts it (`passed=N skipped=1`). It is never a silent pass. |
| `resolver-e2e` | **Only if `codex` present**: under `codex sandbox` with the forwarded args, `resolve-role-spawn.sh designer` emits a `recipe …` line. With no args it emits `refuse fail-closed:designer-model-missing`. That is the ticket's symptom and its cure in one arm. Skipped loudly like (b). |
| `mutation-drop-forwarding` (d) | Copy the helper to a `mktemp -d` scratch dir and delete the line that appends to `MC_CODEX_ENV_ARGS` (`sed '/MC_CODEX_ENV_ARGS+=/d'`). Re-run arm (a)'s assertion against the mutant and REQUIRE it to fail. Do the same with a mutant call site whose expansion is removed from `codex exec`. If either mutant passes, the suite fails ("arm (a) cannot see a dropped forward"). Never mutate the tracked file in place. |

### Acceptance criteria

- [ ] All arms above are green on the rig under `/bin/bash` 3.2.57, with `roundtrip-codex-e2e`
      and `resolver-e2e` actually RUN (not skipped). Paste the suite's summary line.
- [ ] `make test-launchd-drivers` green, run **unpiped**: `make test-launchd-drivers; MAKE_RC=$?; echo MAKE_RC=$MAKE_RC`.
- [ ] `bash -n` on `mission-control.sh`, `lib/codex-env-args.sh` and the new test.
- [ ] `shellcheck` shows no new findings on the touched files (diff the finding count against `fcce2394c`).
- [ ] No `declare -A`, `${v,,}`, `${v^^}`, `mapfile`/`readarray` or `. <(…)` in the new files.
- [ ] The diff touches only the five files listed. `git diff fcce2394c --stat` proves it.
- [ ] Changelog fragment `changelogs/unreleased/2026-10-07-codex-controller-role-env.md` exists
      and names both the role-env fix and the restored scope guard.
- [ ] Live proof on the next codex-controller fire: the log line `codex-env: forwarded=N …` with
      N ≥ 30, and the iteration's resolver calls return `recipe`/`reroute` lines (no
      `fail-closed:*-model-missing` / `env-pin`).

## Dry-run note

`AILANG_DRIVER_PINNED=<sha> MISSION_DRY_RUN=1` under an idle sibling profile is still required
by the charter, and it proves that the driver still parses and sources the new helper (the source line
sits at the line-885 position, well before the dry-run exit at 1926). It does **not** reach the
codex call site: it exits before role env is built. To exercise the seam itself, the
controller runs, on the rig:

1. `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_codex_controller_env.sh` with
   codex on PATH, so arms `roundtrip-codex-e2e` and `resolver-e2e` run live against the rig's
   real `config.toml` (they make no model call and spend no quota).
2. From the controller's own mission shell, which carries the real driver env:
   `( . tools/launchd/lib/codex-env-args.sh; MC_CODEX_ENV_ARGS=(); mc_codex_env_args; for r in designer executor evaluator; do codex sandbox "${MC_CODEX_ENV_ARGS[@]}" -- /bin/bash tools/launchd/resolve-role-spawn.sh $r; done )`.
   Expect three `recipe`/`reroute`/`agent-tool` lines and zero `fail-closed`.

## Policy boundary (requirement 5)

The change touches **none** of: routing, lane order, controller/role fallback chains, quota
probes, rations (`MISSION_OVER_RATION` is forwarded read-only, not recomputed) or billing guards.
It does not alter which model any role resolves to. It only lets the already-resolved values
reach the controller's shell. Not HD-2a.

Restoring the scope guard's `GIT_CONFIG` entry *re-enables* an HD-4 control that was silently
bypassed. It does not widen or narrow what HD-4 allows.

## Non-goals and follow-up ticket candidates

1. **Follow-up candidate (do NOT fix here, may be routing policy): a codex controller can still
   be handed `agent-tool <alias>`.** Codex's native `spawn_agent` rejects `opus`/`sonnet`
   ("Unknown model"), as World reported. Evidence, with env present:
   - `resolve-role-spawn.sh` emits `agent-tool $PIN declared:alias-pin` for any bare (no `:`)
     pin (lines 133-135) and never consults `CONTROLLER_PROVIDER`/`MISSION_ANTHROPIC_AVAILABLE`.
   - The evaluator default is the bare alias `sonnet` (mission-control.sh:1457), and its fallback
     tail is the bare `opus` (line 1572).
     `MISSION_EVALUATOR_MODEL=sonnet MISSION_EXECUTOR_RESOLVED=codex:gpt-6.1-sol MISSION_ANTHROPIC_AVAILABLE=0 resolve-role-spawn.sh evaluator`
     → `agent-tool sonnet declared:alias-pin`. The same with `opus` gives `agent-tool opus declared:alias-pin`.
   - It has happened live. In `/tmp/ailang-mission-world.log`, 2026-09-30 08:12:35,
     `controller=codex:gpt-6-sol … evaluator=sonnet`. That is 1 of the 26 distinct codex-controller
     fires across the mission logs. Each `.log` has a `.launchd.log` mirror, so the raw grep counts
     below are doubled. The rest resolved the evaluator to `claude:claude-sonnet-4-6` (20) or
     `pi:openrouter/minimax/minimax-m3` (5).
   - Planner is safe under a codex controller. The driver sets `MISSION_ANTHROPIC_AVAILABLE=0`
     for codex (line 929), and `derive-planner-lane.sh`'s wrapper rewrites every `opus …` to
     `MISSION_PLANNER_ANTHROPIC_FALLBACK` (→ `recipe codex:gpt-6.1-sol anthropic-fallback:…`).
     The designer is safe while its pin carries a provider (default `claude:claude-opus-5-5`).
     A mission env that sets a bare designer alias would hit the same gap.
   Candidate ticket: "resolver: under a non-claude controller map a bare alias to its provider
   pin (`sonnet`→`claude:claude-sonnet-*`, `opus`→`claude:claude-opus-*`) as a `recipe` line".
   This changes the dispatch mechanism, which is exactly the distinction the 2026-09-22
   evaluator-order comment (mission-control.sh:1554-1557) says is deliberate, so it routes to
   policy review.
2. The `include_only` residual (finding 3). If the rig ever adds `include_only`, the tripwire
   warns. Deciding whether the driver should then also override `include_only` is a separate
   call.
3. World runs its own driver copy. Reach is pin-gated, so the fix reaches World when its pin
   advances past this commit. Porting checks belong to the `mission-loop-change` skill, not to
   this sprint.

## Verification Log (2026-10-07, rig, codex-cli 0.159.2, names only)

```
$ stat -f '%Sm' ~/.codex/config.toml                 -> Oct  6 13:02:28 2026
$ (policy section, values redacted)                  -> inherit = <redacted>; .set: NODE_REPL_TRUSTED_BROWSER_CLIENT_SHA256S, CLAUDE_CODE_OAUTH_TOKEN
A  codex sandbox -- /usr/bin/env | sed 's/=.*//'
   -> CLAUDE_CODE_OAUTH_TOKEN CODEX_SANDBOX CODEX_SANDBOX_NETWORK_DISABLED HOME LOGNAME
      NODE_REPL_TRUSTED_BROWSER_CLIENT_SHA256S PATH SHELL TMPDIR USER          (no MISSION_*, no GIT_CONFIG_*)
B  -c shell_environment_policy.inherit=all  (filtered to MISSION|PROBE|KEY|TOKEN|SECRET)
   -> all MISSION_* + PROBE_MARK, AND AILANG_REGISTRY_API_KEY CLAUDE_CODE_MESSAGING_TOKEN DOCPARSE_API_KEY
      GOOGLE_API_KEY LYCEUM_API_KEY OLLAMA_API_KEY OPENAI_API_KEY OPENROUTER_API_KEY SENTRY_ACCESS_TOKEN
      TYPESAFE_API_KEY ZAI_API_KEY                                             (rejects option B)
E  -c 'shell_environment_policy.set.MISSION_NAME="probe-via-set"'
   -> baseline names + MISSION_NAME; rig .set vars still present; printenv -> probe-via-set
F  -c include_only=["HOME","PATH"] -c set.MISSION_NAME="x"
   -> CODEX_SANDBOX CODEX_SANDBOX_NETWORK_DISABLED HOME PATH                  (include_only strips .set)
G  -c inherit=all  -c set.MISSION_NAME="via-set"   -> printenv MISSION_NAME = via-set
H  -c inherit=none -c set.MISSION_NAME="via-set"
   -> CLAUDE_CODE_OAUTH_TOKEN CODEX_SANDBOX CODEX_SANDBOX_NETWORK_DISABLED MISSION_NAME
      NODE_REPL_TRUSTED_BROWSER_CLIENT_SHA256S                                (works under any inherit)
RT prototype toml_str under /bin/bash 3.2.57, value with ` " \ $HOME LF TAB ' → **
   -> codex sandbox -c "shell_environment_policy.set.MISSION_RT=$enc" -- printenv MISSION_RT : ROUNDTRIP-OK
R  prototype allowlist+denylist over compgen -e of a live fleet mission shell
   -> forwarded=44; every MISSION_*/AILANG_DRIVER_* name present inside the sandbox (MISSING count 0);
      MISSION_ROUTING_NOTE round-trips byte-exactly; AILANG_REGISTRY_API_KEY absent;
      rig .set vars still present
   (first attempt sourced the function via `. <(sed …)` -> "toml_str: command not found" in bash 3.2 -> finding 4b)
RS resolve-role-spawn.sh under codex sandbox WITH forwarded args:
   designer: recipe claude:claude-opus-5-5 declared:provider-pin
   executor: recipe claude:claude-sonnet-5-5 declared:provider-pin
   evaluator: reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge
   WITHOUT args: refuse fail-closed:designer-model-missing                   (the ticket's symptom)
I  git config core.hooksPath inside codex sandbox:
   baseline -> <shared clone>/.git/hooks ;  with GIT_CONFIG_* via .set -> /scope/guard/githooks
   driver shell -> <driver>/tools/launchd/githooks                           (scope guard off under codex today)
U  /bin/bash -c 'set -u; a=(); echo "${a[@]}"'      -> a[@]: unbound variable, rc=127
   /bin/bash -c 'set -u; a=(); echo ${a[@]+"${a[@]}"}' -> rc=0
D  grep: MISSION_DRY_RUN exit at mission-control.sh:1926; role env exported 2180-2203; codex call 2274
8  env -i … MISSION_EVALUATOR_MODEL=sonnet MISSION_EXECUTOR_RESOLVED=codex:gpt-6.1-sol resolve-role-spawn.sh evaluator
   -> agent-tool sonnet declared:alias-pin   (opus -> agent-tool opus declared:alias-pin)
   grep 'controller=codex.*roles:' /tmp/ailang-mission-*.log | evaluator=… : claude:claude-sonnet-4-6 x40,
   pi:openrouter/minimax/minimax-m3 x10, sonnet x2 (world 2026-09-30 08:12:35; .log+.launchd.log mirror -> halve)
```
