# M-PACKAGE-PROTOCOL-MANIFESTS: let a repo declare its own agent protocol, without letting it disarm itself

**Status**: Planned — **split out of [m-coordinator-execution-trust.md](m-coordinator-execution-trust.md) 2026-09-02** (Mark, attended) after three quorum rounds established that this is a permission/authority design, not the bug fix it was bundled with.
**Target**: v0.36.0 (after M-COORDINATOR-EXECUTION-TRUST lands)
**Priority**: P1 — the ecosystem goal is real, but nothing is blocked on it; the execution unblock ships without it.
**Dependencies**: **M-COORDINATOR-EXECUTION-TRUST M1a** — the trusted `work_tier` boundary and the built-in prerequisite floor are the things a manifest is bounded *by*. This doc cannot be specified before they exist.

---

## Why this is a separate document

Mark's D2 ruling on the parent doc was *"gates apply outside of ailang repo — we want a general
ailang message system for our packages as well."* Correct goal. But bundling it with an executor
bug fix made a security design ride along with a deadlock fix, and the review process said so
three times in three different ways:

| Quorum round | Objection that landed here | Reviewer |
|---|---|---|
| 0 | A repo-declared protocol could replace the default prerequisites — repository-controlled content deciding its own permission requirements | `gpt5-6-sol` |
| 1 | "A repo declaring nothing gets today's AILANG behaviour" contradicts "a foreign workspace with no manifest satisfies the floor" | `gemini-3-1-pro` |
| 2 | The authority boundary is still asserted rather than established — no traced ACL for who controls the routing metadata | `gpt5-6-sol` |

**M2 and M3 of the parent doc had their objections closed and they stayed closed. This one's kept
multiplying.** That asymmetry is the evidence for the split, and it is recorded here so the split
does not read as scope-dodging.

## Problem statement

The parent doc's M1a ships **two built-in prerequisite sets** — a generic floor, and the AILANG
set — selected by trusted coordinator metadata. That is enough to unblock every cloud agent, and
it deliberately contains no repo-published content at all.

It is not enough for the ecosystem goal. A package repo with its own conventions (its own
protocol doc, its own pre-flight checks, its own approval expectations) cannot express them, and
adding a third built-in set per package does not scale.

**The hard part is not expressing them. It is that a manifest is content controlled by the thing
being governed.** A repo that can declare its own protocol can declare an empty one, and the gate
that was supposed to constrain it unlocks itself. Any design here must make that structurally
impossible rather than discouraged.

## Goals

**Primary Goal:** A package repo declares agent-protocol prerequisites that the gate enforces,
without any repo being able to weaken the floor that applies to it.

**Success Metrics:**
1. A repo manifest can ADD a prerequisite and the gate enforces it.
2. No manifest — malformed, empty, adversarial, or unsupported-version — can remove, weaken, or
   short-circuit a floor prerequisite. Asserted by mutation arms, not by review.
3. A malformed or unsupported-version manifest **blocks mutation** with a structured reason; it
   never falls back to a default (no silent fallbacks).
4. A repo that publishes nothing behaves exactly as it does after M1a.

## Open questions this doc must answer

These are the unresolved objections, carried forward verbatim rather than restated as solved:

- **Who may publish a manifest, and does committing to a repo grant that right?** If yes, then
  write access to a package equals influence over the protocol governing agents in that package.
- **What is the ACL on every tier-setting input** — inbox selection, agent config, workspace
  binding? V25 in the parent doc established that `to_inbox` is sender-supplied; the trusted link
  is inbox → registered agent → tier. A manifest adds another input to that chain and it needs the
  same tracing.
- **Versioning and unsupported versions.** An old gate meeting a new manifest must fail closed.
- **Distribution.** Is the manifest read from the cloned workspace (repo-controlled, needs the
  add-only bound), or from the coordinator's own config keyed by repo (trusted, but then it is not
  really the repo declaring anything)? **This choice is the whole design.**

## Non-Goals

- Everything in the parent doc: the tier boundary, the built-in floor, the no-op status value, the retry chain.
- Any change to the interactive TUI confirm path.

## Related Documents

- [m-coordinator-execution-trust.md](m-coordinator-execution-trust.md) — the parent; M1a is the dependency.
- [v0_35_0/m-dx-session-protocol-gate.md](v0_35_0/m-dx-session-protocol-gate.md) — the gate's original design and its A4 claim.
- [m-message-plane-trust.md](m-message-plane-trust.md) — the delivery layer beneath all of it.

---

**Document created**: 2026-09-02
**Last updated**: 2026-09-02
