# Coordinator: enable auto-merge on design-doc PRs (docs-only paths)

- **Date**: 2026-09-15
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `auto-merge|autoMerge` (design_docs/, internal/), `gh pr create` (internal/, tools/), direct reads of `internal/coordinator/daemon_tasks_exec.go`, `internal/coordinator/agent_registry.go`, `internal/coordinator/autonomy_router.go`, `internal/dispatch/cloudrun/dispatcher.go`
- **Existing coverage**: none found for docs-only auto-merge policy specifically; the auto-merge machinery itself already exists.

<The mechanism the report asks for is already built — the omission is in configuration, not
capability. Dispatch carries `params.AutoMerge = agent.AutoMerge`
(`internal/coordinator/daemon_tasks_exec.go`, `AutoMerge` field) sourced from per-agent
registry YAML (`internal/coordinator/agent_registry.go`, `auto_merge` yaml tag), and the
auto-merge guard deliberately refuses when an agent has no `ArtifactPatterns` declared
(`internal/dispatch/cloudrun/dispatcher.go`, "an agent that never declared its artifacts
gets no auto-merge instead of unlimited scope") — which is precisely the docs-only path
scoping the requester wants: `DefaultArtifactPatterns("design-doc-creator")` already returns
`design_docs/**/*.md` (`internal/coordinator/agent_registry.go`). Non-agent PRs get
auto-merge; agent PRs don't because their registry entries don't set it.

This is a design-doc rather than a config one-liner for two rubric reasons. (1) There is
more than one acceptable way to do it: flipping `auto_merge: true` + explicit
`artifact_patterns: ["design_docs/**"]` on the design-doc-producing agents in the registry
YAML, versus a code-level rule ("docs-only diff ⇒ auto-merge allowed") that would apply to
any agent whose PR touches only docs — the second is more general and changes what the
guard's contract means. (2) Auto-merge is an authority/gate surface: the code comment is
explicit that "a message cannot ask for its own auto-merge", so widening who may auto-merge,
even scoped to `design_docs/**`, is exactly the class of decision someone could disagree
with and deserves a written rationale — including whether the gate is the registry entry
(config, auditable per agent) or the diff contents (code, generalizes but widens authority).
The narrow-version request ("scope it to docs-only paths") is itself a design constraint the
doc should pin.>