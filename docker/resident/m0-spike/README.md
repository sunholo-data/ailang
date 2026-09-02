# M0 spike artefacts — kept for re-measurement, not for deployment

These built the throwaway image that answered the RESIDENT-P1 gate: whether
`herdr server` runs with no controlling terminal, and whether the shared-vCPU
budget carries a real compile. Findings are in
`docs/design/v6.40.0/resident-agent-instances.md` § Phase 0.

Keep them because the measurements are worth repeating when the product leaves
Preview, when instance hardware changes, or when a new vCPU/memory shape is
proposed. They are not part of the resident image.
