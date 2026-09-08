# Role-spawn routing — the two measured traps

Read this when you are about to spawn a heavy role and something disagrees about which lane it
should run on. Both notes below were moved out of `SKILL.md` verbatim under the progressive-
disclosure gate (`scripts/check_context_docs.sh`); the operative rules stay inline there, the
evidence lives here.

---

## 1. The Agent tool DOES accept a `fable` pin (corrected 2026-08-20, V1 iteration 238)

**The stale rule was silently costing every mission its rotation's Fable designer slot.** From
2026-07-16 iteration 31 until then, `SKILL.md` read *"the Agent tool accepts ONLY
`sonnet`/`opus`/`haiku` as explicit pins; **`fable` is REJECTED** (InputValidationError,
live-observed)"*. That was true when measured and is **false in the current harness build**.

Proposed by `mission-world` iter-101 and corroborated **first-party in V1's own session** before
adoption (sibling-claim ghost discipline), on two independent readings: the Agent tool's `model`
enum in this build lists `sonnet`/`opus`/`haiku`/**`fable`**, and a role spawned with an explicit
`model="fable"` was **ACCEPTED and ran to completion** — no `InputValidationError`. World's
instance was a 15.6-minute designer run returning a 232-line revision; V1's was a bounded probe.

**Why a stale CAPABILITY rule is worse than a stale fact:** this one does not merely misinform, it
*instructs a re-route* — so the rotation's Fable entry is skipped **silently**, and the loop cannot
tell a deliberately-skipped designer from an unavailable one. World reached Fable at iter-101 only
because the *next* rotation entry (gemini) is read-only under `CapRemoteSandbox` and cannot author
a file at all.

**Scope, stated honestly and NOT widened:** what is established is that the pin is **accepted** and
the run **completes**. Neither mission verified which weights served the request, so
*"`fable` is pinnable"* is supported and *"the fable pin is enforced end-to-end"* is **not** — do
not quote this note for the stronger claim. The Fable diet is unchanged: pinnability makes the slot
reachable, it does not make it cheap, so it stays at most ONE bounded design DOC per iteration.

If a pin is ever rejected again, treat that as a harness change worth measuring — re-probe with one
bounded spawn and record the reading — rather than restoring the old rule from memory.
**Generalises past this one alias: a capability claim about the harness is a measurement with a date
on it, and the model table is exactly where such claims go stale unseen** — when a rule tells you a
route is unavailable, the cheapest possible probe beats inheriting a year-old observation. The tell:
you are about to skip a configured lane because a document says it cannot be pinned, and you have
not tried it.

---

## 2. The resolver and the spawn-pin hook give different answers (added 2026-09-05, V1 iteration 329)

**When `resolve-role-spawn.sh` returns a `fail-closed:*` reason and the role carries a
`provider:model` pin, the spawn-pin hook will DENY the alias spawn that Step 1b just told you to
make.** Instance 1 is iteration 327 on a doc-LESS pick, instance 2 is iteration 328 with a doc,
instance 3 is iteration 329, also with a doc — three iterations, each of which burned a spawn on a
guaranteed denial before routing correctly by hand.

Step 1b is complete on *deriving* a lane and says nothing about whether the lane it derives can be
*spawned*. Two instruments read the same configuration and disagree: the **hook** reads
`MISSION_<ROLE>_MODEL` directly and enforces it at the tool boundary, while the **resolver** layers
`derive-planner-lane.sh`'s fail-closed logic on top and can return an alias that contradicts the
pin. Note the denial message names, as its remedy, the very script whose answer it is refusing:

```
deny:provider-pin — planner is pinned to codex:gpt-5.6-sol; Agent-tool alias spawn refused
 — use the cross-provider recipe (resolve-role-spawn.sh planner)
```

**This is not a rare edge — it is the DEFAULT for essentially every real pick.** Measured at
iteration 328: `derive-planner-lane.sh` requires a `planner_lane` field that only **2** design docs
in the whole repo carry (`m-feature-provenance-chains`, `m-mission-elo-routing`), and it returns
`opus fail-closed:planner-lane-field-missing` even for the mission charter itself — so the resolver
says `opus` for nearly every doc, and the hook then denies `opus` wherever the role is pinned to a
`provider:model`. A third arm found the same iteration: run it from the driver's own CWD against a
doc authored in a sprint worktree and it fails one step earlier with `design document is missing or
unreadable`, because the path resolves against CWD — this file's own recurring shape, *a relative
path is a claim about where you are standing*.

**Rules.**

- **(a)** When the resolver's reason token begins `fail-closed:` **and** the role's
  `MISSION_<ROLE>_MODEL` is a `provider:model` value, do NOT spawn the alias — route straight to the
  pin under its own lane recipe (probe included), and record BOTH the resolver's answer and the pin
  you followed in the Gate-4 routing-evidence row.
- **(b)** If you spawn anyway and are denied, the denial is information, not an error: follow the
  pin, and do not retry the alias or treat the role as unavailable. This is neither a probe failure
  nor a capacity park (standing rule 8), because nothing is exhausted and nothing is waiting.
- **(c)** The hook is authoritative wherever the two disagree, for the plain reason that it is the
  boundary the spawn actually crosses; *"do not second-guess the resolver"* binds only where the
  hook has no opinion.
- **(d)** This is the SKILL half of the fix and it does not close the defect — the durable fix is in
  the TOOL (default the planner lane when the field is absent, or stop requiring a field almost no
  doc carries), tracked as `m-resolver-hook-disagree-on-docless-pick`, whose title understates it:
  the disagreement is not about doc-less picks.

Mission-independent — every mission on this rig runs this resolver, this hook and this env contract.
The tell: the resolver handed you an alias, the role's configured value names a provider, and you
have not asked which of the two the tool boundary will believe.
