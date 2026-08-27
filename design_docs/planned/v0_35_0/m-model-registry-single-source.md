# M-MODEL-REGISTRY-SINGLE-SOURCE: models.yml becomes the one place models are assigned

**Status**: Planned — awaiting Mark's ratification of D1–D3
**Target**: v0.35.0
**Priority**: P1 — model assignment is edited "quite often as new models come out" (Mark, 2026-08-27) and currently requires editing four places, one of which needs a code release
**Estimated**: 3–4 days, ~700 LOC
**Dependencies**: M-PIPELINE-RECONCILIATION M5 (the `model_routing` table this supersedes); M-MESSAGE-PLANE-FAIL-LOUD M4 (config CAS)
**Author**: Claude Opus 5 + Mark
**Created**: 2026-08-27

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | One resolution order replaces four independent ones; the same role resolves identically in every lane |
| A2: Replayability | +1 | A run can record the registry version it resolved against |
| A3: Effect Legibility | 0 | No language-level effects |
| A4: Explicit Authority | 0 | No change to capability grants |
| A5: Bounded Verification | +1 | "Which model will this agent use?" becomes one command instead of a four-file audit |
| A6: Safe Concurrency | 0 | Registry publish reuses the CAS path |
| A7: Machines First | +1 | Role → model is machine-resolved from data; today it is shell exports plus per-agent copies |
| A8: Minimal Syntax | 0 | New YAML block in an existing file |
| A9: Cost Visibility | +1 | Pricing already lives in models.yml; assignment joining it means the file that says which model runs also says what it costs |
| A10: Composability | +1 | A new model is one entry; adopting it is one role-chain edit |
| A11: Structured Failure | +1 | Deletes five hardcoded Anthropic defaults that silently catch an unpinned agent |
| A12: System Boundary | +1 | Makes the embed/disk/bucket boundary explicit instead of an accident of build time |

**Net Score: +8** → **Proceed**

### Hard Violation Check
- [x] A1: removes nondeterminism (registration-order and file-precedence dependent behavior)
- [x] A3: no hidden effects
- [x] A4: no ambient authority
- [x] A7: strictly improves machine-resolution

---

## Problem Statement

Model assignment lives in **four** places today. Measured 2026-08-27:

| Source | Size | Consumers | Cost to change |
|---|---|---|---|
| `internal/eval_harness/models.yml` | 136 models, 79 with `agent_model_name` | eval harness, observatory pricing | edit + **rebuild + reinstall** |
| `tools/launchd/mission-control.sh` | 7 role/fallback exports | 3 unattended mission loops | edit shell + copy to `~/.config/ailang/` |
| `config/config.cloud.yaml` | 33 per-agent pins + `model_routing` | 34 cloud agents | edit + push (canonical since `b3982b0`) |
| **executor hardcoded defaults** | 5 | every executor when nothing sets a model | **code change + release** |

The fourth is the dangerous one. `opencode.go:55`, `pi.go:62`, and `motoko.go:147` all default to `anthropic/claude-haiku-4-5`; `claude.go:65` to `haiku`. The fleet was migrated off Anthropic on 2026-08-27 **because Claude-CLI OAuth for headless agents is being retired** — yet any agent whose pin is dropped silently lands back on Anthropic. That is a silent fallback on a billing-and-availability path (CLAUDE.md Critical Principle 2), and it is invisible until an invoice or an outage.

Adding a new model today means: add it to models.yml (rebuild), decide whether the missions adopt it (shell), decide whether the cloud fleet adopts it (config push), and hope no executor default is masking a gap. Four edits, three deploy mechanisms, no single answer to "what runs this role?".

### Why models.yml is the right home

It already is the registry of model FACTS: 136 models across six providers, with `api_name`, `agent_model_name` (the exact per-harness string executors need), pricing (which observatory already consumes), and context/output limits. What it lacks is model POLICY — which model plays which role — and a way to reach consumers without a rebuild.

---

## High-Impact Decisions

| # | Decision | Options | Who | Cost to change later |
|---|---|---|---|---|
| **D1** | **How does a published registry reach an installed binary?** `InitModelsConfig` tries **embed FIRST** and only falls back to disk paths for development (`models.go:200-227`) — so a bucket-published models.yml is IGNORED by every installed binary. This is the central mechanism problem. | **(a)** Precedence inversion with provenance: explicit `AILANG_MODELS_PATH` → published registry (gcsfuse/known path) → embed, logging which won at startup; embed becomes the *floor*, not the ceiling. **(b)** Embed stays authoritative; the registry is published for *reading* only and every consumer takes a rebuild. **(c)** Split files: facts stay embedded, a small `roles.yml` is published and read at runtime. | **Mark** | High — this is the contract every consumer depends on |
| **D2** | **What replaces the hardcoded executor defaults?** | **(a)** Fail loudly: no model resolved ⇒ typed error naming the agent and the roles the registry does know. **(b)** Per-executor "safe default" role resolved from the registry (still fails if the role is absent). | **Mark** | Low |
| **D3** | **Does the mission driver read the registry directly, or keep its env pins?** The loops are live and their pins are battle-tested (~290 iterations). | **(a)** Driver resolves roles via `ailang models role <role>` with env pins still overriding — one source, escape hatch intact. **(b)** Registry is authoritative for cloud only; missions keep shell pins (documented divergence). | **Mark** | Medium |

### Design Freeze
- [ ] D1 registry reach / precedence
- [ ] D2 executor default behavior
- [ ] D3 mission driver adoption

**Recommendation**: D1(a) — precedence inversion is the only option that satisfies "updated quite often" without a release per update, and provenance logging keeps it honest. D2(a) — the migration's entire purpose is defeated by an Anthropic fallback. D3(a) — one source with env override preserves the loops' rollback ergonomics while ending the divergence.

---

## Solution Design

### Phase 1 — roles in models.yml (~150 LOC)

A `roles:` block beside the existing `models:`, expressed in the friendly names models.yml already uses:

```yaml
roles:
  designer:  [or-kimi-k3, fable-5]
  planner:   [codex-gpt-5-6-sol, or-kimi-k3]
  executor:  [codex-gpt-5-6-sol, or-deepseek-v4-flash]
  evaluator: [or-minimax-m3, or-deepseek-v4-flash]
  package:   [or-deepseek-v4-flash]
```

`ResolveRole(role, lane)` returns the ordered chain, each entry carrying the model's `agent_model_name` for the target harness and its `executor_variant`. Lane-awareness matters: ollama entries are rig-only and must not be offered to a Cloud Run job (the missions' fallback chains already encode this distinction informally).

### Phase 2 — the registry reaches consumers (~250 LOC, D1(a))

`InitModelsConfig` gains explicit precedence — `AILANG_MODELS_PATH` → published registry path → embedded — and **logs which source won and its version at startup**, so "which registry is this process using?" is answerable from a log line rather than inferred. The registry publishes to the config bucket next to `config.cloud.yaml` (same CAS tooling, same repo-canonical source: `ailang-multivac/config/`), and cloud jobs mount it exactly as they already mount the coordinator config.

### Phase 3 — consumers resolve through it (~300 LOC)

- Coordinator: agent `role:` resolves via the registry; `model_routing` in cloud config is **deleted** (superseded).
- Executors: hardcoded defaults become D2(a) typed failures.
- `ailang models role <role> [--lane local|cloud]` prints the chain — the mission driver's read path (D3(a)) and a human's audit tool.
- Per-agent `model:` pins survive as deliberate overrides.

### Files to Modify/Create
- `internal/eval_harness/models.yml` — `roles:` block
- `internal/eval_harness/models.go` — precedence, provenance, `ResolveRole`
- `internal/executor/{opencode,pi,motoko,claude,codex}/*.go` — remove the five defaults
- `internal/coordinator/model_routing.go` — resolve through the registry
- `cmd/ailang/models.go` — **new**, the `models role` command
- `tools/launchd/mission-control.sh` — read the chain (D3(a))
- `ailang-multivac/config/` — publish the registry

---

## Success Criteria

- [ ] Adding a model and adopting it for a role = **one file, one push**, no rebuild
- [ ] `ailang models role executor` answers identically on rig and cloud
- [ ] An agent with no pin and no resolvable role FAILS with a message naming the agent and known roles — verified by a test that greps the executor sources for hardcoded provider strings
- [ ] Startup logs the registry source and version
- [ ] `model_routing` deleted from cloud config with no behavior change (fixture test: pre/post resolution identical for all 34 agents)
- [ ] Mission loops resolve through the registry with env pins still overriding

## Testing Strategy
- **Unit**: role resolution incl. lane filtering, missing-role errors, precedence order
- **Guard**: a test asserting no executor contains a hardcoded provider/model literal — the regression that motivates D2
- **Fixture**: all 34 cloud agents resolve post-migration exactly as their current pins specify
- **End-to-end**: publish a registry with one role changed; confirm a cloud job picks it up **without a rebuild** — the claim D1 exists to make true

## Non-Goals
- Changing which models are chosen (that is the routing table's content, not this mechanism)
- Pricing/observatory changes — models.yml already serves those
- Per-task dynamic model selection

## Risks & Mitigations
| Risk | Mitigation |
|---|---|
| A bad published registry breaks every lane at once | CAS validation before write; embed remains the floor if the published one fails to parse; startup logs the source |
| Precedence inversion surprises a dev with a stale local file | Provenance line at startup; `AILANG_MODELS_PATH` is explicit and wins |
| Mission loops destabilised mid-flight | D3(a) keeps env pins overriding; adopt during an attended window |
| models.yml grows a second audience with different needs | Roles are additive; eval consumers ignore the block |

## Verification Log

Measured 2026-08-27 at HEAD `fb6084f4b`.

| # | Claim | Method | Result |
|---|---|---|---|
| V1 | Four independent assignment sources | grep/count per source | Confirmed — 136 / 7 / 33 / 5 |
| V2 | Executors hardcode Anthropic defaults (**the motivating defect**) | read the five executor files | Confirmed — `opencode.go:55`, `pi.go:62`, `motoko.go:147` = `anthropic/claude-haiku-4-5`; `claude.go:65` = `haiku`; `codex.go:45` = `gpt-5-codex` |
| V3 | **Embed wins over disk; disk is dev-only fallback** | read `models.go:199-227` | Confirmed — this is why a published registry is ignored today, and what D1 must invert |
| V4 | A disk load path already exists | `LoadModelsConfig(path)` at `models.go:184`, used by observatory pricing | Confirmed — Phase 2 extends rather than invents |
| V5 | models.yml already carries per-harness model strings | count `agent_model_name` | Confirmed — 79 entries |
| V6 | Roles exist ONLY as shell exports today (**negative**) | grep roles across Go sources | Confirmed — no Go-side role concept |
| V7 | Coordinator does not import eval_harness (**negative**) | grep imports | Confirmed — Phase 3 introduces the first dependency; watch for an import cycle (`internal/storage` → `coordinator` already forced one workaround this week) |
| V8 | models.yml is `go:embed`-ed | `models.go:15-16` | Confirmed |

**Not verified**: that gcsfuse-mounting the registry into agent jobs works — the coordinator mounts its config that way, but no agent job does today. Phase 2 must prove it before Phase 3 depends on it.

## Related Documents
- [m-pipeline-reconciliation.md](../../implemented/v0_35_0/m-pipeline-reconciliation.md) — M5 built `model_routing`; this supersedes it with one registry
- [m-message-plane-fail-loud.md](../../implemented/v0_35_0/m-message-plane-fail-loud.md) — M4's CAS tooling publishes the registry
- `reference_cloud_config_source_of_truth` (memory) — the repo-canonical rule the registry must follow
