# M-COORDINATOR-INBOX-WILDCARDS: glob-based inbox routing in the agent registry

**Status**: Planned
**Target**: v0.19.0 (small surface, deploy-coupled)
**Priority**: P1 — silent cascade failures hard to detect without
[`cloud-cascade-debug`](../../../.claude/skills/cloud-cascade-debug/) skill
**Estimated**: ~30 LOC core change + ~80 LOC tests + cloud config + ops doc
**Dependencies**: M-PKG-AUTONOMOUS-CASCADE-SAFE (v0.16.0), M-PKG-AUTO-UPDATE-DX (v0.18.0)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-12
**Source**: discovered while debugging the M-EXT-PORTABILITY-GATE follow-up
cascade — abi 2.1.0 republish triggered 10 dependent cascade tasks, all of
which silently failed because no agent was registered for `pkg:sunholo/motoko_ext_*`
inboxes in the cloud coordinator config.

---

## Problem statement

Today's coordinator dispatches cascade tasks via `AgentRegistry.GetAgentForInbox(inbox)`,
which is a plain hashmap exact-match
([`internal/coordinator/agent_registry.go:242`](../../../internal/coordinator/agent_registry.go#L242)).
For a published package's cascade to succeed, the **exact** inbox name
`pkg:sunholo/<name>` must already exist in the coordinator's agent registry —
loaded from `gs://ailang-multivac{,-dev}-ailang-config/config.yaml`.

When a new package family is added (today: `motoko_ext_*`), every individual
package in the family needs its own agent entry. Forget to add even one and:

1. The cascade message arrives at the coordinator
2. A task is created with `agent: ""` (empty AgentID)
3. The task is dispatched to a Cloud Run Job
4. The job exits 1 immediately:
   `error=AILANG_AGENT_ID environment variable is required`
5. The cascade message is acked (because the coordinator received and processed it)
6. **Nothing visible at the publish CLI** — `Cascade-topic notification published`
   is printed regardless

The publisher gets no signal that the cascade actually ran. The first
indication is a downstream consumer noticing a stale dependency, often days
later. Today this was caught only because the M-EXT-PORTABILITY-GATE manual
republish surfaced it.

This is a forever-tax: every new package family added to ailang-packages will
hit the same gap unless someone remembers to add cloud-config entries.

## Goals

1. **Wildcard inbox patterns** in agent config so families can be routed in
   one entry: `inbox: "pkg:sunholo/motoko_ext_*"`
2. **Catch-all support** so all unrouted packages fall through to a default
   agent: `inbox: "pkg:*"`. Optional but cheap.
3. **Longest-prefix-wins** precedence so explicit registrations override
   wildcard ones (e.g. an explicit `pkg:sunholo/motoko_ext_compaction_ai`
   agent overrides the family glob).
4. **Visible failure mode**: when a cascade task is created with empty
   AgentID, log a clear warning before dispatching (or refuse to dispatch
   and flag the message for manual triage).
5. **Ops runbook** documenting how to add a new package family to cloud
   config, with the wildcard option as the recommended path.

## Non-goals

- Full glob syntax (`?`, `[abc]`, etc.) — only `*` at the end is needed for
  v1.
- Multiple wildcards within a single pattern (`pkg:*/motoko_*`) — defer.
- Hot-reloading the cloud config — still requires Cloud Run revision
  redeploy or service restart. (Could be an M2 follow-up.)

## Conflict surface

Touches:
- `internal/coordinator/agent_registry.go::GetAgentForInbox` — extend lookup
  logic to fall back to longest-prefix wildcard match. Risk: changes the
  semantics for the existing `byInbox` exact-match path. Mitigation:
  exact-match still wins; wildcards are pure addition.
- `internal/coordinator/agent_config.go::AgentConfig` — add `IsWildcard
  bool` derived field (or just check `strings.HasSuffix(inbox, "*")`
  at lookup time).
- `internal/coordinator/daemon_tasks_polling.go` — when creating a task
  with empty AgentID after lookup, log a structured warning so cloud-
  cascade-debug can grep for it.
- Cloud config files (`gs://ailang-multivac{,-dev}-ailang-config/config.yaml`)
  — add motoko_ext family wildcard entry as the canonical example.
- `~/.ailang/config.yaml` template — same.

**Programs that MUST still work** (regression fixtures):

1. Existing exact-match registrations: `pkg:sunholo/auth`, `pkg:sunholo/gcp_auth`
   etc. continue to route to their explicit agents. (Test: 22-pkg-agent dev
   config + lookup of every key returns the original agent.)
2. Mixed exact + wildcard: explicit `pkg:sunholo/motoko_ext_compaction_ai`
   wins over `pkg:sunholo/motoko_ext_*`. (Test: longest-prefix-wins.)
3. No registrations for an inbox: still returns nil. (Test: lookup of
   `pkg:nonexistent/xyz` returns nil when no `pkg:*` catch-all exists.)
4. Catch-all only: `pkg:*` matches any pkg-prefixed inbox. (Test: lookup of
   any `pkg:vendor/name` finds the catch-all when no other match exists.)

## Solution sketch

```go
// agent_registry.go
type AgentRegistry struct {
    byInbox     map[string]*AgentConfig    // exact-match (existing)
    wildcards   []wildcardEntry            // NEW — sorted by prefix length DESC
}

type wildcardEntry struct {
    prefix string  // e.g. "pkg:sunholo/motoko_ext_"
    agent  *AgentConfig
}

func (r *AgentRegistry) GetAgentForInbox(inbox string) *AgentConfig {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // 1. Exact match wins
    if a, ok := r.byInbox[inbox]; ok {
        return a
    }
    // 2. Longest-prefix wildcard match (entries sorted DESC at register time)
    for _, w := range r.wildcards {
        if strings.HasPrefix(inbox, w.prefix) {
            return w.agent
        }
    }
    return nil
}

func (r *AgentRegistry) Register(agent *AgentConfig) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.agents[agent.ID] = agent
    if strings.HasSuffix(agent.Inbox, "*") {
        prefix := strings.TrimSuffix(agent.Inbox, "*")
        r.wildcards = append(r.wildcards, wildcardEntry{prefix, agent})
        sort.Slice(r.wildcards, func(i, j int) bool {
            return len(r.wildcards[i].prefix) > len(r.wildcards[j].prefix)
        })
    } else {
        r.byInbox[agent.Inbox] = agent
    }
}
```

Cloud config example (after this lands):

```yaml
coordinator:
  agents:
    # Catch-all for the motoko_ext_* family — auto-handles new packages
    - id: pkg-motoko-ext-cascade-bumper
      label: "Cascade bumper for motoko_ext_*"
      inbox: "pkg:sunholo/motoko_ext_*"
      workspace: sunholo-data/ailang-packages
      merge_branch: main
      capabilities: [code, package, cascade]

    # Explicit agent that overrides the family glob (optional)
    - id: pkg-motoko-ext-abi
      label: "Special-case agent for the abi root"
      inbox: "pkg:sunholo/motoko_ext_abi"
      workspace: sunholo-data/ailang-packages
      merge_branch: main
      capabilities: [code, package, cascade]
      max_cost_usd: 0.05  # tighter budget for the most-cascading root
```

## Acceptance

- [ ] `TestGetAgentForInbox_WildcardFallback` covers the 4 regression
  fixtures above.
- [ ] `TestGetAgentForInbox_LongestPrefixWins` — `pkg:sunholo/motoko_ext_abi`
  hits explicit agent, `pkg:sunholo/motoko_ext_other` hits family glob.
- [ ] `TestGetAgentForInbox_CatchAll` — `pkg:*` matches arbitrary
  `pkg:vendor/name` when no other match.
- [ ] `cloud-cascade-debug` skill updated with "wildcard registered? is
  match working?" diagnostic step.
- [ ] Ops doc at `docs/internal/cloud-coordinator-config.md` (NEW)
  documenting how to add a package family with the wildcard pattern.
- [ ] Cloud config in `gs://ailang-multivac-dev-ailang-config/config.yaml`
  uses `pkg:sunholo/motoko_ext_*` (after fix lands + dev coordinator
  redeployed).
- [ ] Validation: re-publish a no-op patch bump of motoko_ext_abi 2.1.x
  and verify auto-cascade fires for all 13 dependents end-to-end.

## Why this matters for AI-author workflows

Today's failure mode is invisible: agent publishes `motoko_ext_X 0.2.1`,
sees `Cascade-topic notification published`, assumes auto-bump will fire.
Days later a consumer hits a stale dep and the agent has to manually
republish all dependents (today's experience: 10 republishes triggered by
one abi bump). With the wildcard pattern + visible-failure logging:

1. Cloud config has one entry per family — no per-package maintenance.
2. New packages in the family are auto-routed without config changes.
3. Failed cascades log clearly so `cloud-cascade-debug` can spot them
   on the first invocation.

This eliminates an ongoing-tax that compounds with every new motoko_ext
(or any future family) addition.

## Refs

- Source bug: M-EXT-PORTABILITY-GATE follow-up cascade (2026-05-11) —
  see `.ailang/state/evaluations/eval_M-EXT-PORTABILITY-GATE_round_2.json`
  and `cloud-cascade-debug/SKILL.md` for the full triage trail.
- Cloud config bucket layout: see `docs/internal/cloud-coordinator-config.md`
  (NEW, sibling to this doc).
- Pairs with: M-PKG-AUTO-UPDATE-DX (v0.18.0), M-PKG-CASCADE-DETERMINISTIC-FIRST (v0.16.0).
