# Iteration 90 — routing evidence

| Role | Resolver | What ran | Tokens |
|---|---|---|---|
| Controller | — | `claude:claude-opus-5-5` (`MISSION_ANTHROPIC_AVAILABLE=1`) | not reported |
| Designer | `recipe claude:claude-opus-5-5 declared:provider-pin` | Not spawned: the D-FLEET-42 row and iteration 88's design record are the spec. Rotation not advanced. | — |
| Planner | `agent-tool opus fail-closed:no-doc` (the design doc exists only on the draft branch) | Not spawned: iteration 88's plan (`iteration88/plan.md`) specifies M3 (arm table, dry-run section). The anchors moved with M2; the directive named the current lines. | — |
| Idle-screen judge | `agent-tool sonnet declared:alias-pin` | sonnet, Agent tool, foreground, read-only. The first spawn was refused by the spawn-pin hook (`fail-closed:role-missing`: the first line carried a suffix after `MISSION-ROLE: evaluator`); it was re-spawned with the bare token line. | 77,288 |
| Executor | `recipe codex:gpt-6.1-sol declared:provider-pin` | Probe rc 0 (`ok`). Run: `codex exec --sandbox workspace-write --add-dir GOCACHE --add-dir GOMODCACHE -C .wt-fleet-iter90-secrets-m3`, background wrapper, 3000 s cap, rc file. rc 0, about 25 min, cap not hit. | 159,062 |
| Diff judge | `agent-tool sonnet declared:alias-pin` | sonnet, Agent tool, foreground, own worktree; 2 rounds (resumed). | 97,695 (cumulative) |

- Judge independence: cross-vendor (sonnet judged a codex generator, plus two controller commits disclosed to it).
- Quorum: none. Metered spend: $0. No lane recorded dead.
- Live dry-run probes: 18 bounded ~1-token probes across claude opus, codex gpt-6.1-sol and pi ollama/glm-5.3:cloud, plus the degraded arms' bogus codex model and the triage profile's own role probes.
