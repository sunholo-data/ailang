# M-GIT-GUARDRAILS-HOOKLESS-EXECUTORS

**Status:** Proposed. Raised by the Daneel agent 2026-09-13; scope agreed with Mark
the same day ("make the design doc for anything not doing now").
**Created:** 2026-09-13. **Target:** unassigned.
**Priority:** P1 — it is a silent, fleet-wide gap, not a defect in one agent.
**Predecessor:** [`m-git-guardrails.md`](../implemented/v0_9_2/m-git-guardrails.md) (v0.9.2),
which built the guard and is not wrong — it is enforced on exactly one harness.
**Not in scope:** the four wrapper fixes shipped in v0.38.4/v0.38.5 (push refspec,
post-push assertion, fatal PR failure, `head_commit` on the payload). Those make the
failure *loud*. This document is about making the behaviour *impossible*.

## Problem

`AILANG_GIT_MODE` is declared per agent, set by the dispatcher
([`dispatcher.go:248`](../../internal/dispatch/cloudrun/dispatcher.go)), defaulted by
the wrapper ([`coordinator_cloud.go:452`](../../cmd/ailang/coordinator_cloud.go)) and
documented in the registry
([`agent_registry.go:236`](../../internal/coordinator/agent_registry.go)).

**Nothing in Go reads it.** Its only reader is
`ailang_bootstrap/scripts/hooks/git_guard.sh`, a **PreToolUse hook** — a Claude Code
surface. An executor with no hook mechanism receives the variable and ignores it.

Measured 2026-09-13 against `config.cloud.yaml`:

| provider | agents | hook surface | guard enforced |
|---|---:|---|---|
| pi | 35 | none | **no** |
| codex | 2 | none | **no** |
| motoko | 1 | none | **no** |
| claude | 0 | PreToolUse | n/a |

So the guard is enforced for **zero** of the 38 registered cloud agents. It was
enforced when written; the fleet migrated off the harness it depends on, and the
control did not migrate with it. Nothing reported this, because a hook that is never
invoked and a hook that permits everything are indistinguishable from outside.

### The incident that exposed it

Tasks `task-70b77905` and `task-1063e5fd` (`design-doc-creator-daneel`, pi running
glm-5.3-flash, 2026-09-13). The wrapper created `coordinator/task-<id>` at the clone
point. The agent then ran `git checkout -b design-doc/<slug>`, committed there, and
pushed that branch itself. Nothing in its skill told it to; it simply did.

`git_guard.sh` blocks precisely this — `checkout -b` and off-branch pushes — and never
ran. The consequences were the wrapper bugs already fixed, but the *cause* was an agent
doing something a guard exists to prevent.

## Current behaviour

1. The registry declares `git_mode` per agent.
2. The dispatcher sets `AILANG_GIT_MODE` on the Cloud Run job.
3. The wrapper defaults it to `guardrails` when unset.
4. On pi/codex/motoko, no component consults it. The agent has an unrestricted `git`
   on `PATH` and a credential able to push.

Steps 1–3 are working correctly and produce no effect. This is the shape of the
role-fallback defect of 2026-09-05 and of `auto_merge` before 2026-09-11: **declared is
not enforced, and the declaration is what people read.**

## Proposed design

Move enforcement from the *harness* to the *environment*, so it holds for every
executor including ones not yet written.

### Option A — a `git` shim on `PATH` (recommended)

Ship a small executable named `git` earlier on `PATH` than the real one, in
`agent-base` so every executor variant inherits it. It reads the same
`AILANG_GIT_MODE`, applies the same rules as `git_guard.sh`, and `exec`s the real
`git` when the command is permitted.

- Enforcement no longer depends on the harness having hooks.
- One rule set, one place; `git_guard.sh` becomes a thin caller of the same logic, or
  is retired.
- The wrapper's own git calls must bypass it. They already route through
  `internal/gitexec`, which gives us a single place to set an escape variable — and
  `check-git-exec` already refuses new bare-name `git` exec sites outside that package
  (47 sites baselined), so the bypass cannot silently spread.

**Risks.** A shim on `PATH` is invisible in a stack trace and mystifying if it
misfires; it must log every refusal with the rule that fired. It must be transparent
for read-only commands, since agents legitimately run `git log`/`status`/`diff`
constantly. And `check-git-exec`'s baseline will need a deliberate entry, not a drift.

### Option B — remove the push credential from the agent's environment

The agent commits; only the wrapper pushes. Strictly stronger than any filter: a
credential that is absent cannot be misused.

- Kills the whole class, including branch tricks nobody has thought of.
- Requires the deploy-key path (`AILANG_SSH_KEY_SECRET`) to be wrapper-only, which is
  already how it is written — `configureSSHDeployKey` runs in the wrapper, though it
  writes `~/.ssh/config`, which the agent process can also read.
- **Does not compose with agents that legitimately push**, and the v0.38.4 fix exists
  precisely because some do. Needs a decision on whether that is a capability we want
  at all.

### Option C — skill text only (defence in depth, never the guarantee)

Add to the design-doc skills: *commit on the branch you were given; do not create
branches and do not push.* Cheap, immediate, and worth doing alongside either option
above — but it is a prompt. A model that ignores it fails silently and we are back
here. It must not be recorded as the fix.

**Recommendation: A now, C alongside it, B as a separate ruling** — B changes what
agents are allowed to be, which is Mark's call and not a bug fix.

## Constraints

- `agent-base` is the single root of all 14 executor images; the shim belongs there
  and nowhere else.
- Images reach prod only by tag → test → `promote`, so this ships on a release, not a
  config push.
- The shim must be inert when `AILANG_GIT_MODE` is unset or `off`, or local runs and
  eval harnesses break.
- bash 3.2 on the rig if the shim is shell (see `.claude/rules/`); a Go binary avoids
  that entirely and we already build one into the image.

## Implementation plan

1. Extract the `git_guard.sh` rule set into a table with a self-test — one definition,
   two callers.
2. Build the shim (Go, into `agent-base`), `PATH`-ordered ahead of `/usr/bin/git`.
3. Route the wrapper's own calls around it via `internal/gitexec`.
4. Prove enforcement per executor, not per rule: a task per provider that attempts
   `checkout -b` and is refused.
5. Repoint `git_guard.sh` at the shared rules, or retire it.
6. Skill text (Option C).

## Validation criteria

- A pi agent that runs `git checkout -b x` is refused, and the refusal names the rule.
- The same for codex and motoko — **enforcement is asserted per harness**, because
  "the rule is defined" is exactly the claim that was already true and already useless.
- `git log`, `status`, `diff`, `add`, `commit` are unaffected.
- The wrapper still pushes, and `assertRemoteMatchesHead` still passes.
- With `AILANG_GIT_MODE` unset, behaviour is byte-identical to today.
- A test proves the shim is *reached*: a positive control that fails when the shim is
  removed from `PATH`. An empty search is a claim, not a fact — and a guard nobody can
  prove ran is the defect this document is about.

## Unknowns

- Whether pi exposes any pre-execution hook in 0.84.x that would make Option A
  unnecessary on the rig (cloud is pinned to 0.73.1 — a known version split).
- Whether any agent legitimately needs `checkout -b`. The cascade path does not; the
  website builder is unaudited.
- Whether `git_mode` values beyond `guardrails`/`off` were ever specified. If not, the
  enum should shrink to what is enforced.
