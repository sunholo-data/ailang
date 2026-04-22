# Sprint Plan: M-EXEC-EXPAND (Codex + opencode Executors)

**Status**: Ready for execution
**Target**: v0.15.0
**Estimated**: 11.5 working days (~2 weeks, 2 sequenced sprints)
**Priority**: P1
**Role**: Extracts Sprints 3-4 from [M-EVAL-EXPAND](m-exec-expand-codex-opencode.md); supersedes [M-COORD-CODEX](../v0_13_0/m-coord-codex-executor.md); adds Gemini/Codex/opencode microrag frontends (bolt-on to [M-BRAIN-MICRORAG](m-brain-microrag.md))

## Source-of-Truth Design Doc

- [m-exec-expand-codex-opencode.md](m-exec-expand-codex-opencode.md) — the full design (axiom scoring, uniform CLI-subprocess contract, files to create, risks). This plan drives execution against that doc.

---

## Why This Sprint (and Why Now)

- **Blocking promise**: [models.yml](../../../internal/eval_harness/models.yml) has 11 `agent_cli: null  # OpenAI Codex CLI not yet implemented` lies — the infrastructure advertises Codex support that doesn't exist.
- **M-EVAL-SUITE-PREP just shipped** (v0.14.0) — the tier + tag infrastructure that gives these new harnesses somewhere useful to run.
- **Zero coordinator diff** — the [`ExecutorProvider`](../../../internal/coordinator/provider_executor.go) auto-discovery layer (landed Feb 2026) means adding Codex + opencode requires only 2 blank-import lines in one file. Every week we wait is a week we're leaving that leverage on the table.
- **M-EVAL-EXPAND's language track** (langreg + JS/Go) is independent and can land in parallel, but only after the executor track proves the "uniform shape" holds.

---

## Velocity Calibration

**Reference points** (from [internal/executor/](../../../internal/executor/)):
- `gemini/gemini.go` = 560 LOC impl, `gemini_test.go` = 156 LOC tests → **716 LOC total**, shipped as a 1-week sprint ([m-exec-gemini-sprint-plan](../../implemented/v0_6_1/m-exec-gemini-sprint-plan.md))
- `claude/claude.go` = 773 LOC impl, `claude_test.go` = 626 LOC tests → **1,399 LOC total** (larger because of auth + credentials handling)
- M-EVAL-SUITE-PREP (v0.14.0): ~1,060 LOC + 51 YAML updates in 5 days = **~210 LOC/day blended**

**Planning target**: 200 LOC/day sustained, 350 LOC/day peak.

**Total sprint estimate**: ~1,900 LOC impl + tests + docs → ~10 days at 200 LOC/day with buffer.

---

## Milestone Breakdown

Ten milestones across two weeks, structured: **Codex core → Codex wiring → shape contract → Gemini+Codex microrag shims → opencode spike → opencode core → opencode wiring → opencode microrag plugin → cross-harness E2E**.

| # | ID | Title | Est. LOC | Depends on |
|---|----|-------|----------|-----------|
| M1 | M1_CODEX_CORE | Codex executor + NDJSON parser | 420 | — |
| M2 | M2_CODEX_REGISTER_FLAGS | init() registration + HealthCheck + CLI flag research + README | 280 | M1 |
| M3 | M3_MODELS_YML_WIRING | Resolve 11 `agent_cli: null` lines; add `agent_suite` composite | 80 | M2 |
| M4 | M4_CODEX_E2E_SHAPE_DOC | Integration test + `docs/internal/EXECUTOR_SHAPE.md` | 280 | M3 |
| M4A | M4A_MICRORAG_GEMINI_CODEX_HOOKS | Gemini + Codex microrag hook shims + docs | 200 | M4 |
| M5 | M5_OPENCODE_SPIKE | Research spike + `opencode_compat_test.go` (design-freeze gate) | 200 | M4A |
| M6 | M6_OPENCODE_CORE | opencode executor + parser + tests | 700 | M5 |
| M7 | M7_OPENCODE_WIRING | models.yml opencode entries + blank import + README | 160 | M6 |
| M7A | M7A_MICRORAG_OPENCODE_PLUGIN | opencode TS plugin for microrag + docs | 150 | M7 |
| M8 | M8_CROSS_HARNESS_E2E | `agent_suite` E2E + evaluation guide + CHANGELOG | 150 | M7A |

**Total**: ~2,620 LOC (impl + tests + docs + YAML + microrag shims)
**Schedule**: 11.5 working days across two 1-week sprints (half-day buffer consumed)

---

## Sprint 1 — M-EXEC-CODEX (Week 1, Days 1-5)

### M1 — Codex Executor Core

**Goal**: Author `internal/executor/codex/codex.go` by copy-modifying `gemini/gemini.go`; port the NDJSON parser from the `codex_compat_test.go` blueprint.

**Files created:**
- `internal/executor/codex/codex.go` — Executor implementation (~500 LOC, mirrors [gemini.go](../../../internal/executor/gemini/gemini.go))
- `internal/executor/codex/codex_test.go` — NDJSON parser unit tests (~200 LOC)
- `internal/executor/codex/testdata/codex_response.jsonl` — NDJSON fixture from a recorded live run

**Acceptance criteria:**
- All 7 methods of the `executor.Executor` interface implemented (`Name`, `Execute`, `ExecuteStreaming`, `Capabilities`, `CostModel`, `HealthCheck`, `Close`)
- NDJSON parser accepts the Codex `{"type":"message","turn_number":N,"text":"...","tokens_used":{"input":N,"output":N}}` schema documented in [codex_compat_test.go](../../../internal/executor/codex_compat_test.go)
- Parser tolerates unknown fields (stores raw event in `Result.ProviderData`) and truncated streams without panicking
- Unit test coverage ≥ 70% for `codex/`
- `make test` passes; `make lint` clean

**Dependencies**: none
**Estimate**: 420 LOC total (300 impl + 120 test) — ~2 days

### M2 — Registration, HealthCheck, Flag Research

**Goal**: Wire Codex into the global factory and discover the correct CLI flags for non-interactive JSON mode.

**Files created:**
- `internal/executor/codex/README.md` — CLI invocation reference, flag table, known limits (~80 LOC)

**Files modified:**
- `internal/executor/codex/codex.go` — Add `Register()` + `init()` block (~20 LOC), `HealthCheck()` via `codex --version` (~40 LOC)

**Research tasks (document findings in README):**
- `--model` flag name (probe with `codex --help`)
- Non-interactive JSON output flag (likely `--json` or `exec --json`)
- Permission bypass equivalent to Claude's `--dangerously-skip-permissions`
- Auth mode (OpenAI API key env var name; confirm not tied to `codex login` cache)

**Acceptance criteria:**
- `executor.GlobalFactory().ListAvailable()` includes `"codex"` after package import
- `TestInit_RegistersCodex` passes (verifies registration)
- `HealthCheck(ctx)` returns nil when `codex --version` succeeds, error otherwise
- `README.md` documents every flag used, every known limitation, and the cost model source
- If any flag cannot be resolved confidently, **PAUSE** and surface the question (design-freeze item)

**Dependencies**: M1
**Estimate**: 280 LOC total (180 impl additions + 20 test + 80 README) — ~1 day

### M3 — models.yml Wiring + agent_suite

**Goal**: Resolve every lie in models.yml and define the `agent_suite` composite for cross-harness sweeps.

**Files modified:**
- `internal/eval_harness/models.yml`:
  - Replace 11 `agent_cli: null  # OpenAI Codex CLI not yet implemented` lines (15, 31, 44, 63, 82, 101, 119) with `agent_cli: "codex"` for models that support Codex CLI
  - Set `agent_model_name` to the correct flag value per model (document in comment)
  - Add `agent_suite` composite at the suite region (~lines 521-574 per the design doc): `members: [claude-opus-4-7, gemini-3-pro, codex-<default>]` (opencode added in M7)
  - Leave lines 141/162/182 (`# Uses Responses API for text generation`) as `null` — these are truly API-only models, not Codex-supported

**Acceptance criteria:**
- Zero remaining `agent_cli: null  # OpenAI Codex CLI not yet implemented` lines in models.yml
- `agent_suite` composite is loadable: `ailang eval-suite --models agent_suite --help` resolves without error
- `ailang eval-suite --models agent_suite --benchmarks fizzbuzz --dry-run` enumerates 3 harness runs (claude + gemini + codex)
- YAML schema validation passes (existing validator in `models.go`)

**Dependencies**: M2 (executor must be registered before models claim it)
**Estimate**: 80 LOC YAML changes — ~0.5 days

### M4 — E2E Integration Test + Uniform Shape Contract

**Goal**: Prove Codex runs end-to-end through the eval harness; formalize the executor shape as a living contract.

**Files created:**
- `docs/internal/EXECUTOR_SHAPE.md` — Uniform CLI-subprocess executor contract (~200 LOC). Contents per the design doc §"Uniform CLI-Subprocess Executor Shape": required package layout, required symbols, required coordinator wiring (blank import), required models.yml fields, required test coverage.

**Files modified:**
- `internal/coordinator/provider_executor.go` — Add `_ "github.com/sunholo-data/ailang/internal/executor/codex"` to the blank-import block at line 10 (1 LOC)
- `.claude/rules/coordinator.md` — Cross-reference `docs/internal/EXECUTOR_SHAPE.md` (~5 LOC)
- `internal/executor/codex/codex_test.go` — Add gated `TestLiveRun_Codex` that skips cleanly when `codex` is not on PATH (~80 LOC)

**Acceptance criteria:**
- `TestLiveRun_Codex` passes when `codex` is installed; skips with clear message when absent
- `ailang eval-suite --models agent_suite --benchmarks fizzbuzz` (with `codex` installed) produces 3 result rows with identical JSON schema (tokens, cost_usd, num_turns, duration_ms all populated for Codex row)
- `ailang messages send coordinator "test" --provider codex --dry-run` resolves the executor via `ExecutorProvider` without any coordinator code changes
- `docs/internal/EXECUTOR_SHAPE.md` exists, is referenced from `.claude/rules/coordinator.md`, and documents the 4 required elements (package layout, symbols, coordinator wiring, models.yml wiring)
- Audit diff: `internal/coordinator/provider_executor.go` shows **exactly 1 line added** (the blank import) — no other coordinator changes

**Dependencies**: M3
**Estimate**: 280 LOC total (200 shape doc + 80 integration test) — ~1.5 days

### M4A — Microrag Frontends for Gemini + Codex

**Goal**: Bolt the existing `ailang micro-rag context` engine ([M-BRAIN-MICRORAG](m-brain-microrag.md)) onto Gemini CLI and Codex CLI via their respective hook systems.

**Files created:**
- `.claude/skills/microrag/frontends/gemini/settings.json` — Template config for `.gemini/settings.json` with `PreToolUse` / `AfterTool` / `SessionStart` hook declarations (~40 LOC)
- `.claude/skills/microrag/frontends/gemini/microrag_context.sh` — Shim script (copy or symlink of existing Claude Code shim; Gemini's JSON protocol is near-identical) (~20 LOC, mostly import comment)
- `.claude/skills/microrag/frontends/gemini/README.md` — Install instructions (~40 LOC)
- `.claude/skills/microrag/frontends/codex/hooks.json` — Template config for Codex CLI `hooks.json` per developers.openai.com/codex/hooks (~40 LOC)
- `.claude/skills/microrag/frontends/codex/microrag_context.sh` — Codex-compatible shim (~20 LOC)
- `.claude/skills/microrag/frontends/codex/README.md` — Install instructions (~40 LOC)

**Files modified:**
- `design_docs/planned/v0_15_0/m-brain-microrag.md` — Add §Frontend D (Gemini CLI) and §Frontend E (Codex CLI) to the Frontends section; reference this sprint as the delivery vehicle

**Research tasks:**
- Verify Gemini CLI hook JSON protocol field names match Claude Code's (documented at geminicli.com/docs/hooks/) — fork the shim only if shape diverges
- Verify Codex CLI hook JSON protocol (developers.openai.com/codex/hooks) — same check
- Confirm `AILANG_MICRORAG_ENABLED` env var propagates through each harness's subprocess spawn

**Acceptance criteria:**
- Running a toy Gemini session with `AILANG_MICRORAG_ENABLED=true` and the template `.gemini/settings.json` installed produces a `🧠 μRAG` marker in the session transcript when editing a `.ail` file
- Running a toy Codex session with `AILANG_MICRORAG_ENABLED=true` and the template `hooks.json` installed produces the same marker
- Both shims `exit 0` silently when `AILANG_MICRORAG_ENABLED=false` or `0` (graceful degradation, per M-BRAIN-MICRORAG A11 score)
- Shim scripts are either byte-identical to the Claude Code `microrag_context.sh` or diverge only in harness-specific JSON field translation (documented in each README)
- `m-brain-microrag.md` Frontends section updated with D + E

**Dependencies**: M4 (Codex executor must be landed so live verification works); no dependency on M5+
**Estimate**: 200 LOC total (~1 day)

**Sprint 1 total**: ~1,260 LOC, 6 days

---

## Sprint 2 — M-EXEC-OPENCODE (Week 2, Days 6-10)

### M5 — Research Spike + opencode_compat_test.go

**Goal**: Install opencode locally, capture its stream format, author the compat test that documents the schema. **This milestone is a design-freeze gate** — if the schema is infeasible (no JSON mode, no stable schema), Sprint 2 halts and this doc ships Codex-only.

**Files created:**
- `internal/executor/opencode_compat_test.go` — Format analysis (~200 LOC, mirrors [codex_compat_test.go](../../../internal/executor/codex_compat_test.go))

**Research tasks:**
- Install opencode CLI locally (`brew install opencode` or per upstream docs); pin version in README
- Run a toy prompt non-interactively; capture raw stdout to `testdata/opencode_response.jsonl`
- Document: event type field, content field, token usage structure, turn markers, tool-use events
- Compare with Claude/Gemini `stream_event` schema and Codex `message` schema — does opencode align with either, or is it a third shape?

**Acceptance criteria (design-freeze gate):**
- `opencode_compat_test.go` passes and documents the schema with fixture assertions
- A recorded fixture exists at `internal/executor/opencode/testdata/opencode_response.jsonl`
- If schema is infeasible: halt Sprint 2, file a message to user, leave Sprint 1 deliverables intact

**Dependencies**: M4 (Sprint 1 must be complete; EXECUTOR_SHAPE.md informs the spike's shape choices)
**Estimate**: 200 LOC — ~1 day (spike)

### M6 — opencode Executor Core

**Goal**: Author `internal/executor/opencode/opencode.go` using EXECUTOR_SHAPE.md as checklist and the spike's compat test as parser blueprint.

**Files created:**
- `internal/executor/opencode/opencode.go` — Executor implementation (~500 LOC)
- `internal/executor/opencode/opencode_test.go` — Parser + registration tests (~200 LOC)

**Acceptance criteria:**
- All 7 `executor.Executor` methods implemented
- Parser accepts the schema from `opencode_compat_test.go` fixtures
- Unit test coverage ≥ 70% for `opencode/`
- `TestInit_RegistersOpencode` passes
- `make test` passes; `make lint` clean

**Dependencies**: M5 (parser design frozen by spike output)
**Estimate**: 700 LOC (500 impl + 200 test) — ~3 days

### M7 — opencode Wiring + README

**Goal**: Wire opencode into coordinator and models.yml; document flags.

**Files created:**
- `internal/executor/opencode/README.md` — Flag reference, known limits (~80 LOC)

**Files modified:**
- `internal/coordinator/provider_executor.go` — Add `_ "github.com/sunholo-data/ailang/internal/executor/opencode"` (1 LOC)
- `internal/eval_harness/models.yml` — Add opencode-capable model entries with `agent_cli: "opencode"`; update `agent_suite` to include opencode (~80 LOC)

**Acceptance criteria:**
- `executor.GlobalFactory().ListAvailable()` includes `"opencode"`
- `agent_suite` now enumerates 4 harnesses in `--dry-run` mode
- Audit diff: `provider_executor.go` gains exactly 1 line (bringing Sprint total to 2)

**Dependencies**: M6
**Estimate**: 160 LOC (80 YAML + 80 README) — ~0.5 days

### M7A — Microrag Frontend for opencode

**Goal**: Ship a TypeScript plugin for opencode that calls `ailang micro-rag context` on tool events, completing microrag coverage across all four harnesses.

**Files created:**
- `internal/executor/opencode/plugins/microrag-plugin.ts` — Plugin module exporting hook functions for `preToolUse` / `postToolUse` / `sessionStart`, shelling out to `ailang micro-rag context` (~100 LOC)
- `internal/executor/opencode/plugins/package.json` — Plugin manifest with name, version, entry point (~20 LOC)
- `internal/executor/opencode/plugins/microrag-plugin.test.ts` — Vitest unit tests: verify shell-out invocation, env-var gating, output parsing (~80 LOC)
- `internal/executor/opencode/plugins/README.md` — Install instructions, opencode plugin loading config, `AILANG_MICRORAG_ENABLED` toggle docs (~40 LOC)

**Files modified:**
- `internal/executor/opencode/README.md` — Reference the plugin in a new §Microrag Integration section (~20 LOC)
- `design_docs/planned/v0_15_0/m-brain-microrag.md` — Add §Frontend F (opencode) to the Frontends section

**Acceptance criteria:**
- `microrag-plugin.test.ts` passes (vitest or equivalent — mock `child_process.spawn` to verify the shell-out arguments)
- Plugin module loads cleanly in opencode's plugin loader (manual verification with a toy session)
- Running a toy opencode session with `AILANG_MICRORAG_ENABLED=true` and plugin installed produces a `🧠 μRAG` marker in the transcript when editing a `.ail` file
- Plugin silently no-ops when `AILANG_MICRORAG_ENABLED=false` or `0`
- `m-brain-microrag.md` Frontends section updated with F

**Dependencies**: M7 (opencode executor + README must land before plugin references them)
**Estimate**: 150 LOC total (~1 day) — include Node.js/TS toolchain setup in the budget if not already present in the repo

### M8 — Cross-Harness E2E + Docs + CHANGELOG

**Goal**: Prove all four harnesses produce identical `RunMetrics` schema; document the new capabilities.

**Files modified:**
- `internal/executor/opencode/opencode_test.go` — Gated `TestLiveRun_Opencode` (~80 LOC)
- `docs/docs/guides/evaluation/*.md` — Document `agent_suite` and new harness options (~40 LOC)
- `CHANGELOG.md` — v0.15.0 entry detailing M-EXEC-EXPAND (~30 LOC)
- `docs/internal/EXECUTOR_SHAPE.md` — Update with any refinements uncovered during opencode implementation (~20 LOC)

**Acceptance criteria:**
- `ailang eval-suite --models agent_suite --benchmarks fizzbuzz` runs all 4 harnesses (skips missing binaries cleanly) and emits 4 result rows with identical JSON schema
- `make ci` passes
- Docusaurus build green (`docs/`)
- CHANGELOG entry under v0.15.0 references both the design doc and sprint plan
- Microrag A/B across all four harnesses: `AILANG_MICRORAG_ENABLED=true|false` toggle measurable against `agent_suite`, results tagged with `microrag_state` per M-BRAIN-MICRORAG §Goal 8

**Dependencies**: M7A
**Estimate**: 150 LOC total — ~0.5 days

**Sprint 2 total**: ~1,360 LOC, 5.5 days

---

## Success Metrics (measurable on v0.15.0 release)

- [ ] `executor.GlobalFactory().ListAvailable()` returns `["claude", "gemini", "codex", "opencode"]` (order-independent)
- [ ] Zero `agent_cli: null  # OpenAI Codex CLI not yet implemented` lines remain in [models.yml](../../../internal/eval_harness/models.yml)
- [ ] `ailang eval-suite --models agent_suite --benchmarks fizzbuzz` emits 4 result rows with byte-identical schema
- [ ] Coordinator routes to both new executors via `--provider codex` / `--provider opencode` with zero new `provider_<name>.go` files
- [ ] [`provider_executor.go`](../../../internal/coordinator/provider_executor.go) gains exactly 2 lines (blank imports); diff audit confirms no other coordinator changes
- [ ] `docs/internal/EXECUTOR_SHAPE.md` exists and is referenced from [.claude/rules/coordinator.md](../../../.claude/rules/coordinator.md)
- [ ] All tests passing (`make ci`)
- [ ] CHANGELOG entry added under v0.15.0
- [ ] Both integration tests (`TestLiveRun_Codex`, `TestLiveRun_Opencode`) skip cleanly when binaries absent
- [ ] Microrag injection verified across all four harnesses (Claude Code, Gemini CLI, Codex CLI, opencode) with `AILANG_MICRORAG_ENABLED` toggle
- [ ] `m-brain-microrag.md` §Frontends updated with D (Gemini), E (Codex), F (opencode)

---

## High-Impact Decisions (carried from design doc)

| Decision | Resolved | Note |
|----------|----------|------|
| Both Codex and opencode are CLI-subprocess executors | ✅ | Design doc §High-Impact Decisions |
| Ship Codex in Sprint 1, opencode in Sprint 2 after spike | ✅ | M5 is the design-freeze gate |
| Uniform shape formalized in `docs/internal/EXECUTOR_SHAPE.md` | ✅ | M4 delivers |
| `agent_suite` = {claude, gemini, codex, opencode} | ✅ | M3 ships with 3, M7 adds opencode |
| Integration tests gated on CLI presence | ✅ | Standard pattern for M4/M8 |
| Codex CLI target = `codex` (OpenAI official); opencode = `opencode` (opencode-ai) | ✅ | Pinned in READMEs |
| Exact Codex CLI flags (model/JSON/permission/auth) | ⏳ | Resolved by M2 research |

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Codex CLI flag names have drifted since `codex_compat_test.go` (Feb 2026) | Med | M2 probes current CLI; tolerant parser stores raw event in `ProviderData` |
| Codex has no non-interactive JSON mode / only exposes via `codex mcp-server` | Med | Re-verify during M2; if true, halt Sprint 1 M3-M4 and pivot to MCP-client executor (separate design doc track) |
| opencode spike finds no stable schema | High | M5 is explicitly a design-freeze gate; Sprint 2 halts cleanly if needed |
| `agent_suite` runs 4× cost per benchmark sweep | Low | Document in evaluation guide; `make eval-smoke` stays single-model |
| LOC overrun on opencode (500 LOC estimate) | Med | Gemini reference is 560 LOC → estimate realistic; buffer is 1.5 days at end of Sprint 2 |
| `make lint` rejects direct test fixtures in `testdata/` | Low | Use `_test.go` naming and `//go:embed` where needed; standard Go tooling handles this |

---

## Timeline

**Week 1 (Sprint 1 — M-EXEC-CODEX):**
- Days 1-2: **M1** Codex core + parser (420 LOC)
- Day 3: **M2** Registration + flag research + README (280 LOC)
- Day 3 (afternoon): **M3** models.yml wiring (80 LOC)
- Days 4-5: **M4** Integration test + EXECUTOR_SHAPE.md (280 LOC)
- Day 6: **M4A** Gemini + Codex microrag hook shims (200 LOC)

**Week 2 (Sprint 2 — M-EXEC-OPENCODE):**
- Day 7: **M5** Spike + opencode_compat_test.go (200 LOC) — **design-freeze checkpoint**
- Days 8-10: **M6** opencode core + tests (700 LOC)
- Day 10 (afternoon): **M7** Wiring + README (160 LOC)
- Day 11: **M7A** opencode microrag TS plugin (150 LOC)
- Day 11 (afternoon) - Day 12 (morning): **M8** Cross-harness E2E + docs + CHANGELOG (150 LOC)

**Total: ~2,620 LOC across 11.5 days**

---

## Non-Goals (from design doc)

- **langreg refactor / JS+Go languages** — M-EVAL-EXPAND Sprints 1-2, independent track
- **openbench adapter / ollama-gemma4** — M-EVAL-EXPAND Sprint 4 tail
- **MCP-client executor** — strategic follow-up doc; not part of v0.15.0
- **Aider/cline/roo executors** — enabled by `EXECUTOR_SHAPE.md`; ship separately
- **Direct OpenAI Responses API executor** — remains an `internal/ai/` concern, not `internal/executor/`
- **Changing default executor** — `gemini` stays default ([factory.go:44](../../../internal/executor/factory.go))

---

## Follow-on Sprints

After this sprint lands, two tracks become easier:

1. **M-EVAL-EXPAND Sprints 1-2** (langreg + JS/Go) can land without executor friction.
2. **MCP-client executor** (tracked separately, v0.16+) can use `EXECUTOR_SHAPE.md` as the contract's v2.

---

**Created**: 2026-04-22
**Author**: sprint-planner (Claude Opus 4.7)

---
SPRINT_PLAN_PATH: design_docs/planned/v0_15_0/m-exec-expand-codex-opencode-sprint-plan.md
