# Sprint Plan: M-MODEL-REGISTRY-SINGLE-SOURCE

**Design doc**: [m-model-registry-single-source.md](./m-model-registry-single-source.md)
**Created**: 2026-08-27
**Duration**: 8 milestones, ~1,180 LOC, 6–7 days
**Risk**: Medium-high — M6 removes a fallback that one live agent is currently using; M8 edits a shell driver with zero CI coverage that runs three live loops
**Decisions**: D1(a) · D2(a) · D3(a) ratified by Mark 2026-08-27; D4(a) by Claude

> **Estimate differs from the design doc.** The doc says ~950 LOC / 5–6 days. Decomposing to
> milestones totals ~1,180 / 6–7. The delta is the fixture-and-pin work (M5) and the CLI half of
> Phase 3 (M7), neither of which the doc sized. Proven velocity in this exact subsystem is
> 170–260 LOC/day (M-PIPELINE-RECONCILIATION 1,220/4d, M-MESSAGE-PLANE-FAIL-LOUD 670/4d, both
> completed), so 1,180 over 6–7 days is at the **conservative** end of demonstrated pace.

---

## Pre-flight

| Check | Result |
|---|---|
| Design doc decisions frozen | ✅ D1/D2/D3 ratified 2026-08-27; D4 decided |
| dev CI green? | ✅ Green at `395d1fef7` — CI green **is** an available success signal this sprint |
| `make check-file-sizes` | ✅ All files ≤800. `models.go` is 520, so the M1 move stays compliant |
| Dependencies implemented? | ✅ Both — `m-pipeline-reconciliation` and `m-message-plane-fail-loud` are in `implemented/v0_35_0/` |
| Does `ailang models role` need inventing? | ❌ **No — a proven shape exists.** `ailang coordinator routing <role>` ([coordinator_config.go:354](../../../cmd/ailang/coordinator_config.go#L354)) already prints a role's chain, reading the config **from the GCS bucket at runtime**. M7 re-points that shape at the registry rather than inventing one |
| Does the mission driver already read it? | ❌ **No.** M5's comment claims "Lane A's driver reads the same table via `ailang coordinator routing`" — grep of `tools/launchd/mission-control.sh` returns **zero** hits. The claim was aspirational; D3 is what makes it true |
| **Who actually breaks under D2(a)?** | ⚠️ **Exactly one agent: `motoko`.** Of 34 cloud agents, 33 carry an explicit `model:` pin. `motoko` has **neither a pin nor a role**, so `ResolveModel` returns `("", nil)` and it falls through to `factory.go`'s `MotokoModel: "openrouter/anthropic/claude-haiku-4-5"`. This is not a hypothetical — it is the live instance of the defect this sprint exists to remove, and it is **the** blocker on M6 |
| Is `model_routing` load-bearing today? | ❌ **It is dead in production.** `ResolveModel` takes the explicit pin first, and all 34 agents resolve that way — including the 4 role-bearing ones (`design-doc-creator`, `sprint-planner`, `sprint-executor`, `sprint-evaluator`), which carry pins *as well as* roles. Its deletion (M7) is therefore inert by construction, and the fixture test proves rather than hopes it |

---

## Milestones

### M1 — Extract `internal/modelreg` (Phase 0a, D4(a))

**~250 LOC** (180 move/impl + 70 test)

Move `internal/eval_harness/models.go` (520 LOC) into a new leaf package. Its only tie to
`executor` is the ten-line pure-string predicate delegated at `models.go:260`; that grammar moves
down with the registry, and `executor.IsOllamaCloudRoute` becomes the delegate in the other
direction (`executor` → `modelreg`, never the reverse).

`eval_harness` keeps `type ModelsConfig = modelreg.ModelsConfig`-style aliases. This is what makes
the milestone mechanical: **66 call sites across 17 non-test files compile unchanged.**

**Acceptance criteria**
- `go list -deps ./internal/modelreg` contains **no** `internal/executor` — the hard gate
- `go build ./...` and `make test` clean with zero call-site edits outside the two packages
- `make check-file-sizes` still green
- `executor.IsOllamaCloudRoute` retains its behavior (existing tests unmodified and passing)

**Files**: `internal/modelreg/` (new), `internal/eval_harness/models.go`, `internal/executor/cost.go`

---

### M2 — Prove the gcsfuse mount (Phase 0b) — **spike, can reopen D1**

**~60 LOC** (mostly job config + a read-back assertion)

The one assumption underneath D1(a) that no measurement supports. The coordinator mounts its
config this way; **no agent job does today.** Mount a registry into one agent job and read it back.

Runs in parallel with M1 — it is infrastructure, not Go, and shares no files.

**Acceptance criteria**
- One agent job starts with a registry mounted and logs the path it read
- The mounted content round-trips: publish a registry with a sentinel field, job reports it
- **If the mount does not work, STOP and reopen D1** — (a) has no delivery mechanism without it,
  and discovering that on day 1 costs a day, on day 4 costs the sprint

**Files**: `ailang-multivac/` job config, a read-back check

---

### M3 — `roles:` block and `ResolveRole` (Phase 1)

**~150 LOC** (90 impl + 60 test)

The `roles:` block beside `models:`, in the friendly names models.yml already uses, plus
`ResolveRole(role, lane)` returning the ordered chain — each entry carrying that model's
`agent_model_name` for the target harness and its `executor_variant`.

Lane-awareness is not decoration: ollama entries are rig-only and must never be offered to a
Cloud Run job. Seed the block from the four chains already in `config.cloud.yaml:60-64`, so the
content is a transcription rather than a new decision (Non-Goal: this sprint does not change
*which* models are chosen).

**Acceptance criteria**
- `ResolveRole` returns the ordered chain for each of designer/planner/executor/evaluator/package
- A `--lane cloud` resolution never returns an ollama entry; `--lane local` may
- A missing role returns a typed error naming the roles that **do** exist
- Chains transcribed from `config.cloud.yaml` resolve to byte-identical model strings

**Files**: `internal/modelreg/models.yml`, `internal/modelreg/roles.go`, tests

---

### M4 — Precedence, provenance, publish (Phase 2, D1(a))

**~250 LOC** (160 impl + 90 test)

`InitModelsConfig` gains explicit precedence — `AILANG_MODELS_PATH` → published registry → embedded
— inverting today's embed-first order (`models.go:199-227`), and **logs which source won and its
version at startup**. Embed becomes the floor, not the ceiling.

Publishing follows the repo-canonical rule: source lives in `ailang-multivac/config/`, the
config-only trigger deploys it, CAS + schema validation gate the write. Per
`reference_cloud_config_source_of_truth`, direct bucket writes get reverted — the plan does not
attempt them.

**Accepted consequence of D1(a)**: pricing rides this same file, so a bad publish can now reach
observatory cost accounting without a rebuild. Schema validation on publish is the mitigation, and
it must cover the pricing fields, not only `roles:`.

**Acceptance criteria**
- Precedence order verified by test at each of the three levels, including fall-through when the
  higher one is absent or unparseable
- An unparseable published registry falls back to embed **and says so loudly** in the log line
- Startup emits source + version; the line is greppable and asserted by a test
- Schema validation rejects a registry with malformed pricing, not just malformed roles

**Files**: `internal/modelreg/models.go`, `ailang-multivac/config/`, tests

---

### M5 — Fixture + close the `motoko` gap (D2(a) precondition) — **gates M6**

**~90 LOC** (30 impl + 60 fixture/test)

D2's ratification carried a sequencing condition: prove no live agent depends on a default
*before* removing the defaults. Pre-flight found exactly one that does.

1. Fixture test: all 34 cloud agents resolve post-migration exactly as their current pins specify.
   Because 33 are explicitly pinned, this is a strong, cheap assertion — it proves the M7 deletion
   of `model_routing` is inert.
2. Pin `motoko` to **GLM-5.3-Flash** (Mark, 2026-08-27), replacing the accidental
   `openrouter/anthropic/claude-haiku-4-5` it inherits from `factory.go`.

**The pin needs a registry entry that does not exist yet.** `or-glm-5-3-flash`
(`models.yml:3999`) is a **standard-mode** entry: no `agent_cli`, no `agent_model_name`. Motoko is
an agentic harness and resolves through `agent_model_name`, following the `motoko-*` twin pattern
(`motoko-glm-5-1`, `motoko-or-deepseek-v4-flash`, …). So M5 adds:

```yaml
  motoko-or-glm-5-3-flash:
    api_name: "z-ai/glm-5.3-flash"
    provider: "openrouter"
    agent_cli: "motoko"
    agent_model_name: "openrouter/z-ai/glm-5.3-flash"
    max_output_tokens: 65536   # always-thinking; the GLM-5.2 truncation lesson
```

**And the evidence behind the choice is standard-mode only.** GLM-5.3-Flash's smoke, core and
frontier records (2026-08-27) were all run in **standard** mode; it is not in `agent_suite` and has
never run under an agentic harness. Per the project's standing rule, standard-mode results do not
transfer to agent mode. At $0.075/$0.25 per 1M an agent-mode smoke tier costs on the order of
$0.06 — cheaper than the meeting about whether to run it — so M5 measures rather than assumes.

**Acceptance criteria**
- Fixture covers all 34 agents; pre/post resolution identical for the 33 pinned ones
- `motoko-or-glm-5-3-flash` registered with `agent_cli: motoko` and the `openrouter/z-ai/…` string
- **Agent-mode smoke tier on `motoko-or-glm-5-3-flash` passes** before the pin goes live — the
  first agentic measurement this model has; a failure routes back to Mark, not to a silent revert
- `motoko` resolves to it, recorded in the config with a comment naming the decision and date
- Test fails loudly if a new agent is added with neither pin nor role

**Files**: `internal/modelreg/models.yml`, `internal/coordinator/` tests,
`ailang-multivac/config/config.cloud.yaml`

---

### M6 — Delete all ten defaults + guard (Phase 3, D2(a))

**~130 LOC** (50 removal + 80 test)

Both layers, or neither. `factory.go:65-71` fills `cfg.*Model`, so the five per-executor defaults
only ever fire when it did not — **removing the downstream five alone is a no-op** (V10).

Unresolved model ⇒ typed error naming the agent and the roles the registry knows.

The guard test greps executor sources for hardcoded provider/model literals and **must include
`factory.go`**. A guard scoped to the five subpackages is precisely the guard that would pass
while the defect survives.

**Acceptance criteria**
- All ten literals gone; `factory.go` no longer names a model
- Guard test fails if any provider literal is reintroduced anywhere under `internal/executor/`,
  including `factory.go` — verified by temporarily reintroducing one
- ~10 `executor.DefaultConfig()` test call-sites rewritten to pin explicitly (per the testing
  policy these are rewritten, not kept compatible)
- An unpinned, unroled agent fails at construction with a message naming known roles

**Files**: `internal/executor/factory.go`, `internal/executor/{opencode,pi,motoko,claude,codex}/`, tests

---

### M7 — `ailang models role` + coordinator resolves + delete `model_routing` (Phase 3)

**~170 LOC** (110 impl + 60 test)

- `ailang models role <role> [--lane local|cloud]` prints the chain — the human audit tool and
  M8's read path. Modelled on `coordinatorRouting`, which already does this shape against a
  bucket-loaded config.
- Coordinator resolves an agent's `role:` through the registry. Note V7: this introduces the
  coordinator's **first** dependency on the registry package — M1 having made it a leaf is what
  keeps that safe.
- `model_routing` deleted from cloud config, superseding M-PIPELINE-RECONCILIATION M5. Inert by
  construction (pre-flight), proven by M5's fixture.
- Per-agent `model:` pins survive as deliberate overrides.

**Acceptance criteria**
- `ailang models role executor` returns identical output on rig and cloud
- Coordinator resolution equals the M5 fixture for all 34 agents after `model_routing` is gone
- `internal/coordinator/model_routing.go` and its config key removed; no dangling references
- `go build ./...` clean — no import cycle (the M1 gate re-asserted at the coordinator)

**Files**: `cmd/ailang/models.go` (new), `internal/coordinator/`, `ailang-multivac/config/`

---

### M8 — Mission driver adoption (D3(a)) — **attended window only**

**~80 LOC shell**

`tools/launchd/mission-control.sh` resolves roles via `ailang models role`, with env pins still
overriding.

Two conditions from the ratification, both load-bearing:

1. **The read path must degrade to today's env pins on any non-zero exit or empty output.**
   `~/go/bin/ailang` drifts by design — CLAUDE.md says to assume it is stale, because installing
   mid-run disturbs concurrent agents. The driver *will* meet a binary without this subcommand.
   Without an explicit fallback branch, that wedges three live loops at once.
2. **Attended window.** This is bash 3.2.57 with **zero CI coverage** driving ~290 battle-tested
   iterations. No `declare -A`, no `${v,,}`.

**Acceptance criteria**
- `MISSION_DRY_RUN` shows the same resolved models as today for every role
- Simulated stale binary (subcommand absent → non-zero) falls back to env pins, loop still runs
- `MISSION_PLANNER_MODEL=opus` still overrides — the rollback ergonomic survives
- One full attended iteration completes green before the loops are left unattended

**Files**: `tools/launchd/mission-control.sh`

---

## Order and dependencies

```
M1 (leaf extract) ──┬── M3 (roles) ──┬── M5 (fixture + motoko) ── M6 (delete defaults)
                    │                │
M2 (mount spike) ───┴── M4 (publish) ┴── M7 (CLI + coordinator + delete routing) ── M8 (driver)
```

- **M1 and M2 run in parallel** on day 1 — different languages, zero shared files, and M2 is the
  one that can invalidate the design.
- **M5 gates M6.** Non-negotiable: it is the ratification condition, and pre-flight found a real
  agent behind it.
- **M8 last**, attended.

## Success Metrics

- [ ] `go list -deps ./internal/modelreg` free of `internal/executor` (M1 gate)
- [ ] An agent job reads a gcsfuse-mounted registry (M2 gate — or D1 reopens)
- [ ] Adding a model and adopting it for a role = **one file, one push, no rebuild**
- [ ] `ailang models role executor` identical on rig and cloud
- [ ] Zero provider literals under `internal/executor/`, `factory.go` included
- [ ] All 34 agents resolve identically pre/post `model_routing` deletion
- [ ] Startup logs registry source + version
- [ ] Mission loops resolve through the registry, env pins still overriding, degrading safely
- [ ] `make ci` green; `make check-file-sizes` green

## Risks

| Risk | Mitigation |
|---|---|
| **M2 fails and D1(a) has no delivery mechanism** | Run it day 1 in parallel with M1. A negative reopens D1 with 7 days left, not 2 |
| M6 lands before `motoko` is pinned and the agent hard-fails | M5 gates M6 in the dependency graph; the fixture test is the gate, not a reviewer's memory |
| The guard test is scoped to the five subpackages and misses `factory.go` | Acceptance criterion requires proving the guard by reintroducing a literal |
| M1's 66 call sites churn and collide with concurrent agents | Type aliases keep them unchanged; land M1 as one early commit so later work rebases onto a settled tree |
| M8 wedges three live loops via a stale binary | Explicit fallback branch is an acceptance criterion, not a nicety; attended window; `MISSION_DRY_RUN` first |
| A bad publish reaches pricing without a rebuild (accepted D1(a) cost) | Schema validation on publish must cover pricing fields; CAS; embed remains the floor |

## Decisions taken during planning

**M5 — `motoko` runs GLM-5.3-Flash** (Mark, 2026-08-27). It was the one agent with neither a pin
nor a role, silently running `openrouter/anthropic/claude-haiku-4-5` from `factory.go` — an
accident, on the provider the fleet just migrated off. Two consequences folded into M5: a
`motoko-or-glm-5-3-flash` agent twin must be registered (the existing entry is standard-mode only),
and the model gets its first agent-mode measurement before the pin goes live, since every existing
GLM-5.3-Flash result is standard-mode.

## Scope: full, M8 included

**M8 confirmed in scope** (Mark, 2026-08-27), after the D3(b) defer option was put to him
explicitly. The sprint retires all four assignment sources, not three.

One scheduling consequence: M8 requires an **attended window** — it is the sprint's only milestone
that can take down live infrastructure (bash 3.2.57, zero CI coverage, three loops, ~290
iterations). Sequence it for a time when someone is watching rather than as an end-of-day landing.
The fallback-to-env-pins branch is an acceptance criterion, not a nicety.
