# M-SPAWN-PIN-ENFORCEMENT — Deterministic role-spawn path enforcement for mission-control

**Type**: Fleet infrastructure fix (shared skill + launchd tooling + one per-mission env widening)
**Raised**: 2026-09-01 (Mark, attended, after reviewing the docs mission's opus burns)
**Status**: DRAFT — awaiting Mark's approval (AGENTS.md feature/semantics gate)
**Traces to**: PROGRAM.md · mission-control SKILL.md Gate 3 · #493 (driver-side sibling) · #902 (executor-default sibling)

---

## Problem (measured, not inferred)

The fleet's cost ladder for planner/executor is `codex → ollama-cloud → openrouter-twin → opus`,
with opus as the designed **implicit tail** (SKILL.md:2545). Three times on this rig the loop
silently skipped to opus (or an alias) anyway:

| Date | Incident | Mechanism | Evidence |
|---|---|---|---|
| 2026-08-30 | docs-9 orphaned fire: planner+executor declared `codex:gpt-5.6-luna`, spawned via the Agent tool → fell back to the session alias `sonnet`; evaluator re-routed to **opus** to preserve generator≠judge | Agent tool enum is `sonnet\|opus\|haiku\|fable` — it cannot carry a `provider:model` pin (SKILL.md:2561) | PR #973 body (FLAGGED), docs-mission-log.md ITERATION 2 |
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

### Layer 2 — PreToolUse hook (hard block: makes the violation impossible, not just forbidden)

The rig already runs Claude Code hooks, and hooks inherit the driver-exported env. Add a
PreToolUse hook matching `Agent`/`Task` calls: if the subagent model param is a bare alias while
the corresponding `MISSION_<ROLE>_MODEL` is a provider pin → **DENY**, returning the resolver's
reason string. The controller's only escape is the recipe — which is the designed chain.

*(Honest risk: hook→session-env inheritance needs a live spike. If it proves unreliable, ship
Layer 1 + tests alone — that already converts silent→loud — and file the hook gap. Decide at
sprint-planning, not here.)*

### Layer 3 — driver exports the resolved plan (single source of truth)

`mission-control.sh` already probes lanes and logs `roles: ...`. Extend it to export
`MISSION_<ROLE>_RESOLVED` and `MISSION_<ROLE>_PATH` so the controller derives nothing, the hook
reads one source of truth, and the Gate-4 routing row is copied from the same bytes the driver
verified.

### Skill edit (shrinks, not grows)

SKILL.md Gate 3: replace the alias/fallback prose with *"for each role, run
`resolve-role-spawn.sh` and spawn per its VERBATIM output; a denied spawn is a routing FAILURE,
not a FLAG."* Keep the roles table as data. This is the same trust model as Step 1b.

### Per-mission env widening (separate knob; Mark's D-1 precedent)

Add `scripts/*` to `MISSION_PLANNER_ALLOWLIST` in **both** the live
`~/.config/ailang/mission-docs.env` and the versioned copy
`tools/launchd/mission-env/mission-docs.env`. Without it, docs-10-class items (anything touching
`scripts/verify_examples.go`, `scripts/validate_manifest.go`) fail-closed to opus-required **by
design**. Buys planner COST only — the executor pin issue is Layer 1/2's job.

### Tests (extend `tools/launchd/test_mission_routing.sh`)

Discriminating controls, replaying the real incidents:
1. `MISSION_EXECUTOR_MODEL=codex:gpt-5.6-luna` + agent-tool request → resolver emits
   `recipe codex` / hook DENY (docs-10 replay).
2. Alias executor + alias evaluator → `agent-tool ok` with generator≠judge asserted; colliding
   pair → re-route emission (docs-9 replay).
3. `scripts/verify_examples.go` brief → **codex lane** after the allowlist widening (was
   `opus fail-closed:path-not-in-codex-allowlist`).
4. Missing `MISSION_<ROLE>_MODEL` → `refuse fail-closed:*`, never opus.
5. All resolver assertions must pass under `launchd drivers (bash 3.2)` CI.

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
- **Driver-side only enforcement**: rejected — the driver cannot see in-session spawns; it can
  only export the plan (Layer 3).

## Rollout & blast radius

- One shared file (`.claude/skills/mission-control/SKILL.md`), two new small bash tools, one
  hook, driver export extension, one per-mission env widening. **No AILANG compiler/runtime
  code.**
- V1/motoko/world behavior change: none when their pins resolve as before; the rule only bites
  when a `provider:model`-pinned role was about to be silently degraded — exactly the case we
  want loud.
- Missions pick the change up on their next fire after merge (they run the skill from the repo
  checkout each iteration).

## Risks

- **Stall instead of spend**: a refused executor role delays an iteration rather than burning
  opus. Accepted — loud-stop beats silent-bill (CP2), and the chain's ollama/openrouter rungs
  make a full stall rare.
- **Hook false positives**: the hook only fires when THAT role's env pin is a provider pin;
  alias-pinned roles (designer, evaluator) are unaffected.
- **bash 3.2**: resolver mirrors `derive-planner-lane.sh`'s constraints; asserted in CI.