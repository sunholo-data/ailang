# M-SPAWN-PIN-ENFORCEMENT — Deterministic role-spawn path enforcement for mission-control

**Type**: Fleet infrastructure fix (shared skill + launchd tooling + one per-mission env widening)
**Raised**: 2026-09-01 (Mark, attended, after reviewing the docs mission's opus burns)
**Status**: IMPLEMENTED 2026-09-03 (iteration 324 M1+M2, iteration 325 M3+M4 — 4/4 milestones) — round-3 text after quorum rounds 1 and 2 (both 3/3 reject; round 2 localised on ONE surface: the hook's role mapping, plus one multi-hook platform premise — all fixes reviewer-authored and applied by the controller under the ratified narrow-refinement carve-out); design DIRECTION approved by Mark attended 2026-09-01
**Traces to**: PROGRAM.md · mission-control SKILL.md Gate 3 · #493 (driver-side sibling) · #902 (executor-default sibling)
**Planner-Lane**: codex-ok

## Files

- `tools/launchd/resolve-role-spawn.sh`
- `tools/launchd/spawn-pin-hook.sh`
- `tools/launchd/test_mission_routing.sh` (or `tools/launchd/test_spawn_pin_hook.sh`)
- `tools/launchd/mission-control.sh`
- `tools/launchd/mission-env/mission-docs.env`
- `.claude/settings.json`
- `.claude/skills/mission-control/SKILL.md`

Some of these are outside V1's planner allowlist; that is fine — the planner lane then resolves to
opus fail-closed, which is the correct, LOUD outcome for a doc that edits the skill and settings.

---

## Problem (measured, not inferred)

The fleet's cost ladder for planner/executor is `codex → ollama-cloud → openrouter-twin → opus`,
with opus as the designed **implicit tail** (SKILL.md, "COMMA-SEPARATED CHAIN walked left to
right, with opus as the implicit tail"). Three times on this rig the loop silently skipped to opus
(or an alias) anyway:

| Date | Incident | Mechanism | Evidence |
|---|---|---|---|
| 2026-08-30 | docs-9 orphaned fire: planner+executor declared `codex:gpt-5.6-luna`, spawned via the Agent tool → fell back to the session alias `sonnet`; evaluator re-routed to **opus** to preserve generator≠judge | Agent tool enum is alias-only — it cannot carry a `provider:model` pin (SKILL.md, "the Agent tool's `model` enum in this build lists `sonnet`/`opus`/`haiku`/**`fable`**") | PR #973 body (FLAGGED), docs-mission-log.md ITERATION 2 |
| 2026-09-01 | docs-10 (PR #1010): planner **and** executor pinned **opus** via the Agent tool; FLAGGED. Planner would have been opus-required anyway: `scripts/*` is not in `MISSION_PLANNER_ALLOWLIST` | same + per-mission allowlist gap | PR #1010 body, `derive-planner-lane.sh` output `opus declared:opus-required` |
| 2026-07-27 | World loop: driver PATH omitted `/opt/homebrew/bin` → codex probe rc=127 → silent opus demote | driver env | #493 (driver attribution refuted there; fixed per-mission via plist/env `PATH=` lines) |

**Key fact**: in both opus burns the cheaper lanes were HEALTHY — the driver's pre-flight probe
passed (`executor=codex:gpt-5.6-luna | lanes=ok` in the fire log). The degradation happened later,
inside the controller session, at spawn-time.

## Root cause

The spawn-path choice (Agent tool vs cross-provider recipe) is made by the **LLM controller at
spawn time**, guided only by prose in the shared skill. The Agent tool's alias-only enum makes
`provider:model` pins inexpressible on that path; the fallback to an alias/opus is a **silent
fallback affecting cost** — a violation of Critical Principle 2. The skill records the demotion in
the Gate-4 routing row AFTER the money is spent. Advisory text is not enforcement (AGENTS.md:
"Advisory text gets skipped — this has been measured"; the docs mission has now paid twice).

## Design principle

**Move enforcement from prompt to code, at the layer where the decision is made.** The skill text
shrinks to one mechanical instruction ("run the resolver, follow its output VERBATIM") — the exact
pattern of the existing, working `derive-planner-lane.sh` Step 1b. No new prose rules to ignore.

## The fix — three layers

### Layer 1 — `tools/launchd/resolve-role-spawn.sh <role>` (deterministic gate)

A bash resolver mirroring `derive-planner-lane.sh` (POSIX-bash 3.2 safe, pure text, fail-closed,
reason tokens, exit 0):

- Reads `MISSION_<ROLE>_MODEL` (+ `_FALLBACK`) from the environment.
- **Provider pin (contains `:`)** → emit `recipe <provider>`: the Agent tool is NOT a valid spawn
  path for this role; the cross-provider recipe (codex exec / pi) must be used, which walks the
  full chain `codex → ollama-cloud → openrouter-twin → opus`. If the recipe provider's probe fails,
  emit the next chain rung — never jump to opus past untried rungs.
- **Bare Anthropic alias** → emit `agent-tool <alias>`, plus the generator≠judge assertion output:
  if the resolved evaluator model equals the executor's resolved model, emit the re-route instead.
- **Missing/unparsable** → emit `refuse fail-closed:<reason>`. Never a default. A refused role is a
  routing FAILURE recorded in Gate 4 — the iteration continues without that role rather than
  spending un-budgeted opus.

### Layer 2 — PreToolUse hook (RELEASE-BLOCKING hard block; the core of this design)

The premise is now **measured, not asserted** (see Verification Log M1): a PreToolUse hook inherits
the controller session's environment (including `MISSION_*` exports), receives `tool_input.model`
and `tool_input.subagent_type` for Agent calls, and a hook deny is honoured even under
`--permission-mode bypassPermissions` (the mission driver's mode). Layer 2 is the only hard
enforcement in this design; Layer 1 alone remains an advisory command the LLM controller can skip —
the exact failure mode this document exists to close. Therefore Layer 2 is a **release-blocking
prerequisite**, not an optional spike.

**The hook, precisely:**

- **File**: `tools/launchd/spawn-pin-hook.sh` (bash 3.2; lives with the other launchd tooling so the
  same CI job covers it); wired in `.claude/settings.json` PreToolUse with matcher `"Agent|Task"`.
- **Input**: the PreToolUse JSON on stdin (`tool_name`, `tool_input.model`, `tool_input.subagent_type`,
  `tool_input.prompt`, `tool_input.description`).
- **ROLE MAPPING** (deterministic, machine-readable, no prompt inference): the role is declared by
  an explicit token — a line `MISSION-ROLE: <role>` as the FIRST line of `tool_input.prompt` (or the
  whole of `tool_input.description`), where `<role>` is exactly one of `{designer, planner,
  executor, evaluator}`. The hook reads the token and nothing else: it does NOT infer a role from
  skill names in prose (Verification Log M8 measured a false positive on exactly that — an
  evaluator-flavoured prompt that mentioned `sprint-planner` mapped to the planner).
  - The driver exports a trusted marker `MISSION_CONTROL_ACTIVE=1` (Layer 3). **When the marker is
    present**, every Agent/Task call must map to exactly one role: a missing token → deny
    `fail-closed:role-missing`; an unknown value → deny `fail-closed:role-unknown`. The ONE
    machine-readable exception is `tool_input.subagent_type == "Explore"` with no token → allow,
    because the Explore agent's tool set excludes Edit/Write/Agent by definition, so it cannot
    author, plan, execute or judge — read-only reality-checks stay available during a mission
    through an explicit field, not through prompt omission. **When the marker is absent**
    (attended sessions, other repos), an unmapped call is allowed — the status quo is untouched.
  - This closes evasion by omission while a mission is running: an unlabelled role spawn is denied
    at the tool boundary, before any money is spent, and the denial reason names the token to add.
  - **Residual surface, tracked**: a controller could LABEL a spawn with the wrong role (e.g. tag an
    executor spawn `evaluator` to reach the sonnet alias). Compensating controls: (1) the hook
    appends every decision (role, pin, model, subagent_type, verdict) to
    `~/.ailang/state/mission-${MISSION_NAME}-spawn-hook.log`; (2) Gate 4's routing row is
    cross-checked against that log and the resolver output — a role whose resolver says `recipe`
    but whose only spawn record is an Agent-tool call is a routing FAILURE, not a FLAG; (3) the
    evaluator arm still applies generator≠judge, so mislabelling an executor as `evaluator` to reach
    the executor's own alias is denied by the re-route rule.
- **DECISION**: for the mapped role read `MISSION_<ROLE>_MODEL` (upper-cased role, e.g.
  `MISSION_EXECUTOR_MODEL`).
  - If it contains `:` (a `provider:model` pin) and `tool_input.model` is absent or any bare alias →
    **DENY** with reason `"<role> is pinned to <pin>; Agent-tool alias spawn refused — use the
    cross-provider recipe (resolve-role-spawn.sh <role>)"`. "Any alias" means the deny does not
    depend on WHICH alias, so retrying with opus/haiku/fable is denied identically — no silent
    retry through another alias.
  - If `MISSION_<ROLE>_MODEL` is unset → **DENY** `fail-closed:<role>-model-missing` (never a default).
  - Bare alias pin → allow, plus the generator≠judge check for the evaluator role: if the evaluator
    alias equals the executor's RESOLVED model, deny with the re-route reason.
- **Output**: the documented JSON
  `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"..."}}`;
  exit 0 in every branch; the two PASSTHROUGH branches (marker absent; tool_name not Agent/Task)
  print NO decision at all — an explicit `allow` would bypass the platform's permission flow in
  every attended session, the opposite of "status quo untouched" (judge finding F1, iteration 324)
  — and an empty/unparsable payload while the marker is set is a deny
  (`fail-closed:payload-unparsable`, judge finding F2). A crashed hook must not become an allow — a hook error is an ALLOW in
  Claude Code's semantics, which is why the script must be defensive and tested for every branch.

### Layer 3 — driver exports the resolved plan (single source of truth)

`mission-control.sh` already probes lanes and logs `roles: ...`. Extend it to export
`MISSION_<ROLE>_RESOLVED` and `MISSION_<ROLE>_PATH` so the controller derives nothing, the hook reads
one source of truth, and the Gate-4 routing row is copied from the same bytes the driver verified.
Layer 3's `MISSION_<ROLE>_RESOLVED/PATH` exports go beside the existing role exports (see
Verification Log M4 for the current export line numbers). The driver ALSO exports
`MISSION_CONTROL_ACTIVE=1` for the whole controller session — the trusted marker the hook uses to
decide whether an unlabelled Agent/Task call is a mission-role spawn (deny) or an attended-session
call (allow).

### Skill edit (shrinks, not grows)

SKILL.md Gate 3: replace the alias/fallback prose with *"for each role, run
`resolve-role-spawn.sh` and spawn per its VERBATIM output; a denied spawn is a routing FAILURE,
not a FLAG."* Keep the roles table as data. This is the same trust model as Step 1b. The spawn pattern gains
one mechanical line: every role prompt begins `MISSION-ROLE: <role>` (the hook's only mapping
input); read-only reality-checks are spawned as `subagent_type: Explore` and need no token.

### Per-mission env widening (separate knob; Mark's D-1 precedent)

Add `scripts/*` to `MISSION_PLANNER_ALLOWLIST` in **both** the live
`~/.config/ailang/mission-docs.env` and the versioned copy
`tools/launchd/mission-env/mission-docs.env`. Without it, docs-10-class items (anything touching
`scripts/verify_examples.go`, `scripts/validate_manifest.go`) fail-closed to opus-required **by
design**. Buys planner COST only — the executor pin issue is Layer 1/2's job.

## Conflict Surface

`derive-planner-lane.sh` and the proposed `resolve-role-spawn.sh` answer **different questions**;
there is nothing to parameterize. Measured by count (Verification Log M2):

- `grep -c MISSION_EXECUTOR tools/launchd/derive-planner-lane.sh` → **0**
- `grep -c MISSION_EVALUATOR tools/launchd/derive-planner-lane.sh` → **0**
- `grep -c MISSION_DESIGNER tools/launchd/derive-planner-lane.sh` → **0**
- `grep -c MISSION_PLANNER tools/launchd/derive-planner-lane.sh` → **5** (known-positive control)

`derive-planner-lane.sh` resolves ONLY the planner lane, and its input is a DESIGN DOC PATH plus a
per-mission path allowlist (Steps 1–5: Planner-Lane field, Files section, allowlist match). The
executor/evaluator/designer roles have no doc dependence — their inputs are env only. So the two
scripts answer "may a cheap lane PLAN this doc?" vs "which spawn PATH does this role's pin permit?"
For the PLANNER role the new resolver **CONSUMES `derive-planner-lane.sh`'s output verbatim** (it
does not re-derive it), so the two scripts compose rather than overlap. Reuse only the
emit/reason-token CONVENTION (`<path> <reason-token>`, exit 0, bash 3.2, `set -f` where globs are
matched).

## Tests (extend `tools/launchd/test_mission_routing.sh`)

Discriminating controls, replaying the real incidents. The test harness feeds the PreToolUse JSON on
stdin and asserts the exact `permissionDecision` and reason substring; it lives in
`tools/launchd/test_mission_routing.sh` (extend) or a sibling `tools/launchd/test_spawn_pin_hook.sh`
invoked by `make test-launchd-drivers`. Keep the existing 36 arms green (Verification Log M6).

1. Provider-pinned executor + alias request → deny (docs-10 replay).
2. Same with a DIFFERENT alias (opus) → identical deny (no silent retry through another alias).
3. Unset `MISSION_EXECUTOR_MODEL` → deny `fail-closed:executor-model-missing`.
4. Alias-pinned evaluator (sonnet) with executor resolved to sonnet → deny re-route (generator≠judge).
5. Alias-pinned evaluator with codex executor → allow.
6. `MISSION_CONTROL_ACTIVE=1` + a prompt that names two role skills but carries no token → deny
   `fail-closed:role-missing` (prose is not a role signal).
7. `MISSION_CONTROL_ACTIVE=1` + no role token, `subagent_type: general-purpose` → deny
   `fail-closed:role-missing`; **control**: the identical call with the marker ABSENT → allow.
7a. `MISSION_CONTROL_ACTIVE=1` + no token + `subagent_type: Explore` → allow (the read-only path);
   **control**: the same call with `subagent_type: general-purpose` → deny (arm 7).
7b. `MISSION_CONTROL_ACTIVE=1` + token `MISSION-ROLE: judge` (unknown value) → deny
   `fail-closed:role-unknown`.
7c. Two overlapping PreToolUse hooks (a catch-all allow listed first, the spawn-pin deny second) →
   the deny wins; asserted against the repo's real `.claude/settings.json`, not a `--settings` file
   (closes the two "NOT measured" gaps below).
8. Hook script must run under `/bin/bash 3.2` (the `launchd drivers (bash 3.2)` CI job,
   macos-latest).
9. Resolver (Layer 1): `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna` → `resolve-role-spawn.sh executor`
   emits `recipe codex …` — the Agent tool is not a valid path (docs-10 replay at Layer 1).
10. Resolver: alias executor + alias evaluator → `agent-tool <alias>` with generator≠judge asserted;
    a colliding pair → the re-route emission (docs-9 replay).
11. Resolver: missing `MISSION_<ROLE>_MODEL` → `refuse fail-closed:*`, never opus.
12. Allowlist widening: a docs-mission brief declaring `scripts/verify_examples.go` → the pinned
    codex/pi lane after `scripts/*` is added (was `opus fail-closed:path-not-in-codex-allowlist`);
    the unwidened env is the control and must still fail closed.

### Bookkeeping

- File the Agent-tool mechanism as its own issue (docs-9/docs-10 PR bodies as evidence);
  cross-link #493 and #902 so the three variants are connected but tracked separately.
- Note issue #544 (stale `.agents/skills` mirror): this change edits the AUTHORITATIVE
  `.claude/skills/mission-control/SKILL.md` only; mirror sync is #544's battle.

## Alternatives considered

- **Prompt-only rule in SKILL.md**: rejected — advisory text is skipped (measured; AGENTS.md), and
  both incidents occurred *after* the limitation was documented in the skill.
- **Extend the Agent tool's model enum**: not our surface (Claude Code platform).
- **Remove the opus tail entirely**: rejected — it is the designed last resort when every
  flat-rate/metered rung is exhausted; the bug is the shortcut past healthy rungs, not the tail.
  **The opus tail STAYS.**
- **Driver-side only enforcement**: rejected — the driver cannot see in-session spawns; it can
  only export the plan (Layer 3).

## Rollout & blast radius

**Do not ship Layer 1 or the skill edit as enforcement unless the PreToolUse hook passes these
tests. If hook enforcement is unavailable, retain the current behavior and open a blocked
platform-gap issue; Layer 1 may land only as diagnostics and must not be claimed to convert silent
fallback to loud failure.** M1 already shows the hook IS available on this rig, so this conditional
is a guard, not an expectation.

- One shared file (`.claude/skills/mission-control/SKILL.md`), two new small bash tools, one
  hook, driver export extension, one per-mission env widening. **No AILANG compiler/runtime code.**
- V1/motoko/world behavior change: none when their pins resolve as before; the rule only bites
  when a `provider:model`-pinned role was about to be silently degraded — exactly the case we want
  loud.
- Missions pick the change up on their next fire after merge (they run the skill from the repo
  checkout each iteration).

## Risks

- **Stall instead of spend**: a refused executor role delays an iteration rather than burning
  opus. Accepted — loud-stop beats silent-bill (CP2), and the chain's ollama/openrouter rungs
  make a full stall rare.
- **Hook false positives**: the hook only fires when THAT role's env pin is a provider pin;
  alias-pinned roles (designer, evaluator) are unaffected.
- **bash 3.2**: resolver mirrors `derive-planner-lane.sh`'s constraints; asserted in CI.
- **Every non-Explore Agent spawn in a mission session must carry a role token.** A
  `general-purpose` reality-check agent with no token is denied while `MISSION_CONTROL_ACTIVE=1`;
  the remedy is the denial reason itself (use `subagent_type: Explore`, or label the role). Loud by
  design; this is the behaviour change the fleet takes on.
- **Wrong-label residual** (see Layer 2): tracked, with the spawn-hook log + Gate-4 cross-check +
  generator≠judge as compensating controls; not claimed closed.

## Verification Log

Every claim below was measured first-party by the controller on 2026-09-03. An empty/negative result
is a claim, not a fact, so each row carries a known-positive control.

| Claim | Command | Observed | Control |
|---|---|---|---|
| M1: hook inherits controller env incl. `MISSION_*` | live PreToolUse spike, `claude-sub -p --model claude-sonnet-5 --settings /tmp/spike_settings_iter324.json --permission-mode bypassPermissions`, parent shell exporting `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna MISSION_EVALUATOR_MODEL=sonnet SPIKE_MARKER=iter324` | hook log `/tmp/hook_spike_iter324.log` (2 firings 2026-09-03T15:10:43Z): `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna`, `MISSION_EVALUATOR_MODEL=sonnet`, `SPIKE_MARKER=iter324`; Bash call `tool_name=Bash tool_input.keys=command,description`; Agent call `tool_name=Agent tool_input.model=sonnet tool_input.subagent_type=general-purpose` | `SPIKE_MARKER=iter324` is a variable that exists nowhere else — its presence proves inheritance, not coincidence |
| M1: hook deny honoured under bypassPermissions | same spike; nested session asked to (1) `echo spike-ok`, (2) spawn Agent (subagent_type general-purpose, model sonnet) | hook returned `{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"SPIKE-DENY-iter324: ..."}}`; nested report verbatim: "1. Bash `echo spike-ok`: succeeded — output "spike-ok". 2. Agent spawn: denied — reason verbatim: "SPIKE-DENY-iter324: MISSION_EXECUTOR_MODEL is a provider pin; Agent-tool alias spawn refused". Not retrying." rc=0 | Bash call succeeded in the same session, proving the deny was specific to the Agent call, not a session failure |
| M2: derive-planner-lane.sh resolves only the planner lane | `grep -c MISSION_EXECUTOR / MISSION_EVALUATOR / MISSION_DESIGNER / MISSION_PLANNER tools/launchd/derive-planner-lane.sh` | 0 / 0 / 0 / **5** | `MISSION_PLANNER` count of 5 is the known-positive control proving grep works |
| M3: cited SKILL.md line numbers are stale | `wc -l .claude/skills/mission-control/SKILL.md`; `grep -n` for each anchor | 2819 lines; "implicit tail"→1320; Agent-tool enum note→1110 (paragraph starting 1105; fable IS accepted); "Spawn pattern (heavy roles)"→1099; "Cross-provider spawn recipe"→1151; line 2561 is standing-rule-7 text, unrelated | four distinct anchors each resolve to a real, distinct line |
| M4: driver lane-probe log format + export lines | read `/tmp/ailang-mission-control.log` (2026-09-03 16:54 fire); `grep -n MISSION_*_MODEL tools/launchd/mission-control.sh` | `[16:54:58] codex model 'gpt-5.6-sol' unusable: probe failed (rc=1)`; `[16:54:59] codex executor lane -> falling back to 'pi:ollama/deepseek-v4-flash:0731-cloud'`; `[16:59:30] LANE DEGRADED this fire: - planner: codex lane gpt-5.6-sol unusable (probe rc=1) → handed to pi:ollama/kimi-k3:cloud`; `[16:59:31] === mission iteration starting (controller=claude:claude-fable-5-1 ... roles: designer=... planner=... executor=... evaluator=sonnet) ===`; exports at lines 544 (DESIGNER), 569 (PLANNER), 586 (EXECUTOR), 630 (EXECUTOR_FALLBACK), 637 (PLANNER_FALLBACK), 793 (EVALUATOR), 821 (EVALUATOR_FALLBACK); per-fire degradation rewrites `MISSION_EXECUTOR_MODEL` at line 904 | the `roles:` line in the same log confirms the exports are the bytes the driver verified |
| M5: existing PreToolUse hooks on the rig | `grep -n PreToolUse .claude/settings.json`; inspect `~/.claude/settings.json` | `.claude/settings.json` PreToolUse has ONE entry, matcher `"Bash\|Edit\|Write\|Read\|Agent\|WebFetch\|WebSearch\|mcp__.*"`, command `"$CLAUDE_PROJECT_DIR"/scripts/hooks/coordinator_hook.sh` (advisory, does not deny); `~/.claude/settings.json` has SessionStart hooks only, no PreToolUse | the existing matcher already includes `Agent`, so a new `Agent\|Task` deny hook runs alongside it; Claude Code runs all matching hooks and any deny wins |
| M6: routing suite baseline | `bash tools/launchd/test_mission_routing.sh` on pristine tree (08ab6ba7c) | `==== 36 passed, 0 failed ====`, rc=0 | the 36 passing arms are the control that the suite is green before new assertions are added |
| M7: two overlapping PreToolUse hooks — both run, deny wins | second live spike (2026-09-03T15:16:56Z), `--settings /tmp/spike2_settings_iter324.json` listing a catch-all ALLOW hook FIRST (matcher `Bash\|Edit\|Write\|Read\|Agent\|WebFetch\|WebSearch\|mcp__.*`, mirroring `coordinator_hook.sh`'s position) and an `Agent\|Task` DENY hook second; nested session asked to `echo spike2-ok` then spawn `subagent_type: Explore`, `model: opus` | hook log: `ALLOW-hook fired tool=Bash`, `ALLOW-hook fired tool=Agent`, `DENY-hook fired tool=Agent model=opus subagent_type=Explore`; nested report verbatim: "1. Bash `echo spike2-ok` — succeeded. 2. Agent (Explore, opus) — denied: `SPIKE2-DENY-iter324: second hook denies`." rc=0 | the allow hook fired on BOTH calls and returned an explicit `allow` decision, so the deny was not the only hook that ran; a SECOND alias (opus) and a different subagent_type were denied identically |
| M8: role extraction — real prompts are not on disk; representative replay | `grep -ciE "invoke the (sprint-executor\|sprint-planner\|sprint-evaluator\|design-doc-creator)"` over `docs-mission-log.md`, `v1-mission-log.md`, `motoko-mission-log.md`; PR #973 and #1010 bodies grepped for the four skill names; then a prototype extractor (`grep -oiE` the four names, `sort -u`, count) over 8 representative prompts | **0 / 0 / 0** captured spawn prompts in the three logs and **0** skill-name mentions in either PR body — the ACTUAL docs-9/docs-10 prompts were never banked, so they cannot be replayed; replay: 4 of 4 role prompts → exactly one role, 2 of 2 read-only reality-check prompts → zero, 1 two-skill prompt → 2 (ambiguous), and **1 FALSE POSITIVE**: "Evaluate whether the sprint-planner's plan was followed by the executor" → `planner` | the four role prompts are the known-positive control for the extractor; the false positive is the measurement that retired prose-based mapping in favour of the explicit `MISSION-ROLE:` token |

**What was NOT measured (must be asserted in the sprint's tests):** the hooks in M1/M7 were loaded
via `--settings`, not from the repo's `.claude/settings.json` (arm 7c); and `MISSION_CONTROL_ACTIVE`
does not exist yet, so the marker-present/absent split (arms 7/7a) is design, not measurement.

## Quorum verification log

- **Round 1** (2026-09-03T15:08Z): reviewers gpt5-6-sol / gemini-3-1-pro / oc-glm-5-2 all present,
  all reject.
  - gpt5-6-sol: Layer 2 was optional, so Layer 1 alone could not prevent silent degradation →
    answered by EDIT 1 (Layer 2 is now release-blocking) + EDIT 2 (rollout guard).
  - gemini-3-1-pro: new `resolve-role-spawn.sh` "mirrors" `derive-planner-lane.sh` without
    justifying non-parameterization; hook premise unverified → answered by EDIT 3 (Conflict
    Surface, M2 counts) + Verification Log M1.
  - oc-glm-5-2: Layer 2 rests on an unverified hook→env-inheritance premise; stale SKILL.md
    citations; no conflict-surface comparison → answered by Verification Log M1, EDIT 8 (quoted
    anchors, M3), and EDIT 3 (Conflict Surface, M2).
  - Controller edit after the designer pass (iteration 324): restored test arms 9–12 (resolver and
    allowlist controls from round 1) that the revision had dropped; no other controller-authored change.
- **Round 2** (2026-09-03T15:15Z): same three reviewers, all present, all reject — objections now
  localised on ONE surface (Layer 2's role mapping) plus one platform premise. No design-direction
  dispute; every objection carried a concrete reviewer-authored fix. Applied by the controller under
  the narrow-refinement carve-out (ratified first use iter-95), no re-quorum:
  - gpt5-6-sol: unlabelled spawns map to NONE and are allowed, so enforcement still rests on prose.
    FIX (verbatim): "Have mission-control.sh export a trusted `MISSION_CONTROL_ACTIVE=1`. When that
    marker is present, make the hook deny every Agent/Task call whose role mapping is NONE or
    ambiguous ... Outside mission-control sessions, unmapped calls may remain allowed. Replace test 7
    ... add a control showing the same unmapped call is allowed when the marker is absent. If
    read-only Explore calls must remain available during missions, route them through a separately
    enforced, explicit machine-readable path rather than prompt omission." → Layer 2 marker rule,
    the `subagent_type: Explore` machine-readable path, Layer 3 export, test arms 7/7a.
  - gemini-3-1-pro: "Claude Code runs all matching hooks and any deny wins" was unverified. FIX
    (verbatim): "Add an M7 verification log row executing a spike with two overlapping PreToolUse
    hooks (one allowing, one denying) to prove both execute and the deny takes precedence." →
    Verification Log M7 (measured: both fired, deny won), test arm 7c.
  - oc-glm-5-2: prose-based role extraction unverified and evadable. FIX (verbatim, the second
    branch): "switch to ... a machine-readable role signal ... or accept that the hook can only
    enforce on a NAMED spawn and document the residual evasion surface as a tracked risk with a
    compensating control". → M8 measured the real prompts are not on disk and found a false
    positive in prose matching; mapping switched to the explicit `MISSION-ROLE:` token; residual
    wrong-label surface documented under Risks with three compensating controls.
