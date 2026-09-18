# Sprint Plan: M-AI-DECIDE-SYSTEM-ONE (Phase 1)

**Design doc**: [m-ai-decide-system-one.md](m-ai-decide-system-one.md)
**Created**: 2026-09-18 · **Approved**: Mark, attended, 2026-09-18 ("sprint plan and execute it as an extension")
**Duration**: 1.5–2 days · **Risk**: low (shadow-only; no core files; vendor risk bounded to ≈$0.10)
**Lane**: extension (PROGRAM.md §4)

## Goal

Ship `sunholo/decisions` (typed decisions over TypeSafe Jev via OpenRouter, pure AILANG), price the model in the observatory, and bank a shadow measurement of Jev vs an LLM control arm vs the declared PROGRAM lane on the 45 routed design docs. **Nothing acts on a Jev answer this sprint.**

## Velocity basis

Phase 0 (the spike) took one attended session: 87 LOC of AILANG that type-checked on the second try and ran first time. The package is that file split in two plus an error ADT, so M1 is mostly tests and docs. Repo velocity over the last 7 days is well above the ~600 LOC this sprint needs; the binding constraint is the two-repo publish/quality gate, not code.

## Milestones

### M1 — `sunholo/decisions` package (≈ 4 h, ~330 LOC)
**Repo**: `~/dev/sunholo-data/ailang-packages/packages/decisions/`

Tasks:
1. `decide.ail` (module `sunholo/decisions/decide`): from `examples/runnable/decide_jev.ail` — `Question`, `Answer`, `DecideError` (6 variants), `Decision`, `Gate`; pure `buildRequest`, `parseAnswers`, `questionsToJsonSchema`, `answer`, `gate`, `expectedScore`; effectful `decide` (`! {Net, Env}`). No default thresholds; `gate` is three-way; `answer`/`expectedScore` return `Result`.
2. `decide_test.ail`: fixture bodies inlined as strings (3 real responses from the spike; 3 malformed: no `answers`, unknown `type`, missing named question) → assert `Ok` shapes and the exact `Err` variant; `gate` exhaustiveness (ChoiceA→Act/Escalate at two thresholds, ScoreA→Act, NoulA→Ungateable); `questionsToJsonSchema` emits per-label number properties. Run `ailang test --package`.
3. `_smoke.ail` (offline: builds a request, parses a fixture, exits `OK:`) so `ailang pkg quality .` executes something.
4. `ailang.toml` (`[effects] max = ["Net","Env"]`, `[release] kind = "feature"`, `ailang = ">=0.40.0"`, repository URL for the `pkg:` inbox), `CHANGELOG.md` `## 0.1.0`, `AGENT.md` (protocol; D4/D6 as consumer rules; non-determinism; the "not in AI budget/trace cost" limitation; `Answer.confidence` provenance caveat; cross-link to `motoko_ext_decision_framework`).
5. `ailang pkg quality .` → report section by section; fix every `PUBnnn` exit-2 gate. `ailang publish --dry-run`, then `ailang publish` (attended).

Acceptance:
- [x] `ailang check decide.ail` clean; `ailang test --package` all green (16)
- [x] Mutation: score arm → 8 fail; empty-Ok → 1 fail; `>` for `>=` in gate → 1 fail
- [x] `ailang pkg quality .` no gates; 0.1.0 + 0.1.1 published; resolves from a fresh dir via the registry
- [x] AGENT.md states non-determinism + not-in-AI-budget limitations

### M2 — price the model + example manifest (≈ 1.5 h, ~60 LOC)
**Repo**: this one.

Tasks:
1. `internal/modelreg/models.yml`: row `or-typesafe-jev-1-13` — `api_name: "typesafe/jev-1.13"`, `aliases: ["typesafe/jev-1-13"]`, `provider: openrouter`, `env_var: OPENROUTER_API_KEY`, `pricing: {input_per_1k: 0.000042, output_per_1k: 0.0}`, `max_output_tokens: 28800`, description naming `text->decisions` / not a chat model. No `agent_*` fields; not in any suite list.
2. `internal/modelreg/resolve_test.go`: `Resolve("typesafe/jev-1.13-20260917")` and `Resolve("typesafe/jev-1.13")` → the row; `PriceTokens(…, 424, 73)` = 1.7808e-05 ± 1e-9. A test that no suite list contains the key. Check the free/local labelling logic does not misread `output=0` with `input>0`.
3. `examples/manifest.json` entry for `runnable/decide_jev.ail`; `go run ./scripts/validate_manifest.go --ci` green.
4. `go test ./internal/modelreg/ ./internal/observatory/` green; `make test-core` unaffected.

Acceptance:
- [x] Both resolutions + price assertion pass; suite-exclusion test passes (3b98e1c4e)
- [x] Manifest validator green with the example listed

### M3 — shadow lane-router measurement (≈ 5 h, ~330 LOC + report)
**Repo**: this one, `tools/decisions/`.

Tasks:
1. `tools/decisions/routed_docs.sh`: list the design docs with a `PROGRAM.md Routing` section and extract `{path, declared_lane}` as JSONL (normalise lane spellings to `extension | ailang | core | mission | registry | other`).
2. `tools/decisions/lane_shadow.ail` (`! {FS, Net, Env, IO}`): per doc, `state = {problem_statement, goals, files_to_modify}` truncated to ≤ 6k chars; questions `lane: Choice(6 lanes)`, `touches_core: Noul`, `severity: Score(P2,P1,P0)`. Arm A = `decide(...)`. Arm B = same `httpRequest` to `openrouter.ai/api/v1/chat/completions` with `response_format: json_schema` (strict) from `questionsToJsonSchema`, own `LlmAnswer` type with `source: "llm:<model>"`, client-side renormalisation with a `sum_flag`. Persist one JSONL row per doc per arm with `{model, id, request_hash, answers, usage, latency_ms}` (D6) to `.ailang/state/decisions/lane_shadow_<date>.jsonl`. Both arms bounded by the Net 30 s deadline; timeouts counted, not retried.
3. `tools/decisions/lane_shadow_report.sh` (jq): agreement Jev↔label, LLM↔label, Jev↔LLM; accuracy by Jev-confidence tertile; timeouts, mean latency, total cost per arm. Writes `design_docs/planned/m-ai-decide-system-one-shadow-report.md`.
4. Run it (control model: pick from `dev_models`, record in the report header). Add the one-line summary to the design doc footer and a **candidate** row `decision-gate` to PROGRAM.md §5b ("measured, not acting"). CHANGELOG entry under Unreleased.

Acceptance:
- [x] Report committed with all three agreements, the tertile table, gate simulation, per-arm cost/latency/timeouts — on n=20, not 45 (see Key Fact 14 correction)
- [x] no importer of `pkg/sunholo/decisions/` outside `tools/decisions` (nothing acts)
- [x] PROGRAM.md §5b candidate row; CHANGELOG; design-doc footer

## Schedule

| Slot | Work |
|---|---|
| Day 1 AM | M1 |
| Day 1 PM | M2, start M3 (runner) |
| Day 2 AM | M3 (arm B, report, run), docs, close |

## Dependencies / assumptions

- `OPENROUTER_API_KEY` present (it is; used by the spike). ≈ 45 docs × 2 arms ≈ $0.10 total.
- OpenRouter `/api/alpha/decisions` stays up (100% uptime last 30 m at spike time). If it moves, M3 records `Http`/`Transport` rows and the report says so — that is data, not a blocker.
- Package tests use the `*_test.ail` + `ailang test --package` convention (agui, auth precedent).

## Out of scope (from the design doc's Non-Goals)

Acting consumers; `std/ai` builtin; direct TypeSafe transport; the `std/ai` deadline gap (filed separately).
