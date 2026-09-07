# M-Surface: Remote approval must not ignore agent-registry load failure

**Status:** Planned
**Scope:** `cmd/ailang/coordinator_approvals_remote.go` (single call site)

## Problem

The remote-plane approval path loads the agent registry with
`agentRegistry, _ := coordinator.LoadAgentRegistry()`, discarding the error.
Worse, `LoadAgentRegistry` returns `nil` error when the local config simply
contains no agents, so an approval against a remote plane with an empty
registry is silently approved — no `TriggerOnComplete` handoff can fire, and
the task stays `pending_approval` forever while the CLI reports success.

## Proposal

Surface the failure at the boundary: propagate `LoadAgentRegistry`'s error,
and — when acting on a remote plane — assert the registry is non-empty (or
contains at least one agent capable of the task's `TriggerOnComplete`
handoff) before recording the approval. Refuse with a clear error otherwise.

## Rationale

An approval whose handoff cannot fire is a lie: it marks the task approved,
dispatches nothing, and strands the task with no signal. Local config
(`~/.ailang`) routinely has no cloud agents, so this failure mode is the
*default* on laptops — silent by construction. Failing loudly at approval
time converts an invisible stall into an actionable error, and costs one
check at a single call site.

## Acceptance Criteria

1. If `LoadAgentRegistry` returns an error, the remote approval command
   exits non-zero and surfaces that error; the approval is not recorded.
2. Approving a remote-plane task with an empty agent registry fails with an
   explicit "no agents configured for remote dispatch" error instead of
   recording a success; covered by a regression test.
