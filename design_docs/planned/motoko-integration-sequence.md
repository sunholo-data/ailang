# Motoko Integration Sequence

**Status**: Active master plan (spans v0.15.0 + v0.17.0)
**Owner**: Mark + Claude
**Created**: 2026-05-04
**Type**: Cross-version coordination doc — not a milestone in itself

---

## Purpose

Master plan for integrating the [arniwesth/ailang motoko fork](https://github.com/arniwesth/ailang/tree/motoko) and [arniwesth/motoko_agent](https://github.com/arniwesth/motoko_agent) consumer back into upstream AILANG. This doc tracks the **multi-release sequence** of work items that together let arni drop his binary fork and run on upstream. Individual milestones live in their own design docs; this doc is the index, status board, and cross-cutting decisions log.

**Headline outcome**: arni's `motoko_agent` runs on upstream AILANG with no fork dependency. His provider/streaming patches are either upstream (M-AI-OPENROUTER, already done) or expressible as `[[ai_provider]]` config in a package.

---

## Why This Doc Exists

The motoko fork exists because three pieces of upstream AILANG were closed-extensible at v0.13.0:

1. **AI provider registry** — hardcoded if/else; no plugin path
2. **Token streaming** — built into provider clients; not exposed via `std/stream`
3. **External-consumer DX** — error codes / module-prefix overlap / effect-row diagnostics gave poor signal for an external project

Each of these has now (or will soon have) a corresponding upstream fix. This doc sequences those fixes and the matching motoko PRs so the work doesn't deadlock on itself.

---

## Status Board

| Milestone | Doc | Phase | Status | Dependencies |
|-----------|-----|-------|--------|--------------|
| **M-AI-OPENROUTER** | [implemented/v0_16_x/m-ai-openrouter-provider.md](../implemented/v0_16_x/m-ai-openrouter-provider.md) | 0 | ✅ Implemented (v0.16.x) — wired into `ailang run` (commit `67254452`, 2026-05-04) | — |
| **M-EFFECT-REFINEMENT Phase 1** | [implemented/v0_15_x/m-effect-refinement-phase1.md](../implemented/v0_15_x/m-effect-refinement-phase1.md) | 0/parallel | ✅ Implemented (v0.15.0; Rand pilot; AI port via M-AI-EFFECT-MODES below) | None blocking |
| **M-AI-EFFECT-MODES** | [implemented/v0_15_x/m-ai-effect-modes.md](../implemented/v0_15_x/m-ai-effect-modes.md) | 0/parallel | ✅ Implemented (v0.15.0) — bare `!{AI}` desugars to `!{AI[mode=fixed]}`; `!{AI[mode=routeable]}` skips `--allow-routing` gate at type level. Validates D12. | M-EFFECT-REFINEMENT Phase 1 |
| **M-AI-PROVIDER-CONFIG** | [planned/v0_15_0/m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) | 1 | ✅ **Code + docs landed** (M1-M4, 2026-05-04). 95 tests passing. Awaiting release. | None |
| **Reference package** | [examples/configdriven_provider_demo/](../../examples/configdriven_provider_demo/) | 1 | ✅ Shipped (in-repo example). External `ailang-packages` package can follow if needed. | M-AI-PROVIDER-CONFIG schema |
| **M-AI-EFFECT-MODES-FOLLOWUPS** | [planned/v0_15_0/m-ai-effect-modes-followups.md](v0_15_0/m-ai-effect-modes-followups.md) | 1/parallel | 🟢 Planned, optional bundle for v0.15.0 (handler defence + replay-only runtime + byok stub + replay-engine pin) | M-AI-EFFECT-MODES |
| **v0.15.0 release** | [release-manager skill](/.claude/skills/release-manager) | 1 | 🔵 **Ready** — M-AI-PROVIDER-CONFIG complete; M-AI-EFFECT-MODES-FOLLOWUPS optional add-on | M-AI-PROVIDER-CONFIG (✅) |
| **M-AI-STREAMING-HELPER** | [planned/v0_17_0/m-ai-streaming-helper.md](v0_17_0/m-ai-streaming-helper.md) | 1 | ✅ **Code + docs landed** (M1-M3, 2026-05-04). Pulled forward from v0.17.0 to v0.15.0 because all prerequisites had shipped. ~700 LOC, 12 tests passing. Awaiting release. | v0.15.0 release |
| **M-EXTERNAL-CONSUMER-DX** | [planned/v0_17_0/m-external-consumer-dx.md](v0_17_0/m-external-consumer-dx.md) | 2 | 🟢 Planned | None hard |
| **M-AI-TOOL-LOOP** | [implemented/v0_17_x/m-ai-tool-loop.md](../implemented/v0_17_x/m-ai-tool-loop.md) | 2 | ✅ **Implemented** (8/8 milestones, 2026-05-05). std/ai gains step / runTools / callResult / callJsonResult + 5 record types (AIError, Message, ToolCall, ToolSchema, StepResult). Real Step impls in anthropic / gemini / openai / openrouter; ollama rejects tools at boundary. ~6h wall-clock vs 7d plan estimate (parallel sub-agents on M2/M3/M4). Closes the last upstream gap that motoko_agent's tool_contract.ail / tool_runtime.ail filled. | None hard |
| **v0.17.0 release** | [release-manager skill](/.claude/skills/release-manager) | 2 | ⏳ After M-EXTERNAL-CONSUMER-DX (M-AI-TOOL-LOOP ✅, M-AI-STREAMING-HELPER ✅) | Phase 2 milestones |
| **PR-A: drop fork code** (arniwesth/ailang) | This doc § Phase 3 | 3 | ⏳ After v0.17.0 ships | Phase 1 + Phase 2 |
| **PR-B: migrate consumer** (arniwesth/motoko_agent) | This doc § Phase 3 | 3 | ⏳ After v0.17.0 ships | Phase 1 + Phase 2, PR-A optional |

**Legend**: ✅ done · 🟢 planned (active) · ⏳ blocked · 🔴 risk

---

## Architectural Decisions Log

Cross-cutting decisions made during sequence design. Each is captured here even though it's also recorded in its origin doc, so future maintainers reading this master doc don't have to reconstruct the chain.

| # | Decision | Rationale | Doc(s) |
|---|----------|-----------|--------|
| D1 | Config-driven providers (Option B) over pure-AILANG-via-`std/net` (Option A) | Option A bypasses AI cap (A4) and budget tracking (A9). Hard architectural constraint, not a tradeoff | [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) |
| D2 | Reject Go plugin / WASM extension API (Option C) | Cross-platform pain, security review burden, build complexity. Config-driven covers >90% of HTTP-shaped providers; long tail stays Go-side | [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) |
| D3 | Streaming `! {AI, Stream, Net}` not `! {Stream, Net}` | Initial sketch bypassed AI cap. Same A4/A9 reasoning as D1 — preserves authority + cost visibility uniformly across streaming and non-streaming AI calls | [m-ai-streaming-helper.md](v0_17_0/m-ai-streaming-helper.md) Architectural Correction Note |
| D4 | Built-in providers stay built-in | OpenAI/Anthropic/Gemini/Ollama/OpenRouter need features (tool use, image input, OpenRouter routing) that aren't in v1 schema. Config-driven is additive, not a migration | [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) |
| D5 | Three v1 request shapes (openai_chat, anthropic_messages, simple_completion) without scoping survey | Iteration-over-survey: ship the three obvious shapes + escape hatch, expand based on real usage rather than speculation. Schema-versioned for clean v2 additions | [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) Design Freeze |
| D6 | Stdlib `std/ai/streaming` vs external package | Discovery: `ailang stdlib search` and brain ingestion surface stdlib in ways external packages don't. Streaming is the headline AI-cap operation; belongs in stdlib | [m-ai-streaming-helper.md](v0_17_0/m-ai-streaming-helper.md) |
| D7 | Ship two PRs to motoko (PR-A fork removal + PR-B consumer migration) together, not staggered | PR-B without PR-A leaves the fork as dead weight; PR-A without PR-B breaks arni's setup. Atomic release | This doc § Phase 3 |
| D8 | Skip the bridge PR (v0.13 → current dev rebase) | Decision (Mark, 2026-05-04): the v0.16/v0.17 path obviates the bridge — no need to land an interim rebase. arni waits for the structural answer | This doc § Phase 0 |
| D9 | Skip the scoping question to arni | Decision (Mark, 2026-05-04): "we can assume Arni is ok with these three request shapes and will add more or edit later on." Concrete artifacts (release + reference package) solicit better feedback than scoping questions | This doc § Phase 0; [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) Receive Path |
| D10 | `AIProviderCapabilities` TOML keys match `internal/ai/routing.go AICapability` wire identifiers | One vocabulary serves both registration-side declaration (this milestone) and request-side `AIRoutingPolicy.Require` (M-AI-OPENROUTER M2). Keys: `tool_calling`, `json_mode`, `streaming`, `vision`, `structured_outputs` | [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) Relationship to M-AI-OPENROUTER |
| D11 | Config-driven providers reject `AIRoutingPolicy` (inherit non-OpenRouter rejection rule) | OpenRouter design (`internal/ai/routing.go:6`) requires non-OpenRouter providers to reject non-zero policies via `ErrRoutingNotSupported`. Config-driven providers inherit this — no silent routing-policy fallback | [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) Relationship to M-AI-OPENROUTER |
| D12 | Config-driven providers map onto `AI[mode=fixed]` once M-EFFECT-REFINEMENT ports AI | Static endpoint/auth + routing rejection = `mode=fixed`. v1 schema needs no `mode` field; v2 may add one if routing aggregators ship as packages. **Validated 2026-05-04**: M-AI-EFFECT-MODES (v0.15.0, parallel thread) shipped the AI default-mode entry — bare `!{AI}` already desugars to `!{AI[mode=fixed]}` today. Config-driven providers inherit this automatically. | [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) Relationship to M-EFFECT-REFINEMENT, [implemented/v0_15_x/m-ai-effect-modes.md](../implemented/v0_15_x/m-ai-effect-modes.md) |

---

## Phases

### Phase 0: Setup (now — completed)

- [x] Triage motoko fork: identify what's redundant vs genuinely new (research summary captured in [m-ai-provider-config.md Motivating Evidence](v0_15_0/m-ai-provider-config.md#motivating-evidence))
- [x] Lock architectural decisions D1-D9 (above)
- [x] Write planned design docs:
  - [x] [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) (v0.15.0)
  - [x] [m-ai-streaming-helper.md](v0_17_0/m-ai-streaming-helper.md) (v0.17.0)
  - [x] [m-external-consumer-dx.md](v0_17_0/m-external-consumer-dx.md) (v0.17.0, pre-existing)
- [x] Skip bridge PR (D8)
- [x] Skip arni scoping question (D9)

### Phase 1: v0.15.0 — Open the AI extension surface

**Goal**: Packages can register AI providers via TOML config. AI cap, budget, traces work uniformly across built-in and config-driven providers.

- [ ] **M1**: Schema + parsing
  - `ConfigDrivenProviderSpec` struct in `internal/pkg/`
  - TOML parsing + validation (required fields, schema version, JSONPath syntax)
  - Golden parsing tests
  - Conflict detection (duplicate provider names)
  - Manifest extension; loader hook (no-op registration initially)
- [ ] **M2**: Generic provider with `openai_chat` + `simple_completion` shapes
  - `internal/ai/configdriven/provider.go` implementing `Provider` interface
  - `bearer` auth shape
  - Response/error JSONPath extraction
  - Budget tracking integration (cost calculation, ledger update)
  - Trace span emission identical in shape to built-in providers
  - Integration tests (httptest.Server)
- [ ] **M3**: `anthropic_messages` shape + remaining auth
  - `anthropic_messages` request shape
  - `x-api-key`, `query-param`, `none` auth shapes
  - `auth_headers` escape with `${ENV_VAR}` interpolation
  - Routing: prefix matcher in `cmd/ailang/exec.go` + `ai_handlers.go`
  - `models.allowed` enforcement
- [ ] **M4**: Docs + reference package + eval baseline
  - `docs/docs/guides/custom-ai-providers.md` with three full recipes
  - `examples/runnable/custom_provider_demo.ail`
  - **Reference external package** (e.g. `pkg/sunholo/ai_vllm`) in [ailang-packages](/Users/mark/dev/sunholo/ailang-packages) — concrete proof for arni
  - Eval baseline with at least one config-driven provider
  - CHANGELOG entry
- [ ] **Release v0.15.0** via release-manager skill

### Phase 2: v0.17.0 — Streaming + external-consumer DX

**Goal**: Token streaming works through the AI effect machinery for all providers (built-in + config-driven). External-consumer pain points (MOD012, effect-row provenance, error_codes.json) are fixed.

- [ ] **M-AI-STREAMING-HELPER M1**: AI handler streaming hook + `std/ai/streaming.ail`
  - Streaming dispatch path in AI handler (consumes `[ai_provider.streaming]` from M-AI-PROVIDER-CONFIG)
  - `stdlib/std/ai/streaming.ail` (≤150 LOC)
  - Cross-link docstrings (`std/stream` ↔ `std/ai`)
  - Snapshot test: streaming AI span shape == non-streaming AI span shape
- [ ] **M-AI-STREAMING-HELPER M2**: Recipe page + prompt + lint
  - `docs/docs/recipes/ai-token-streaming.md`
  - `ailang prompt` audit + patch
  - Cross-domain discovery lint extending `docs/scripts/check-stdlib-index.sh`
- [ ] **M-EXTERNAL-CONSUMER-DX**: All four items per [its design doc](v0_17_0/m-external-consumer-dx.md)
  - MOD012 module_prefix overlap diagnostic
  - Effect-row mismatch with call-site pointer
  - `error_codes.json` release artifact
  - `docs/docs/guides/external-consumers.md` — incorporates rebase playbook, fence-validator pattern, install-script template, `.agent/` taxonomy reference, link to custom-ai-providers guide and ai-token-streaming recipe
- [ ] **Release v0.17.0** via release-manager skill

### Phase 3: Motoko PRs

**Goal**: arni's fork is empty; his consumer runs on upstream AILANG.

#### PR-A: drop fork code (`arniwesth/ailang@motoko`)

Title: **"Drop fork: migrate to v0.17 config-driven providers + streaming"**

Removes:
- All 6 streaming Go files (`internal/ai/stream_motoko.go`, `internal/ai/openai/stream_motoko.go`, `internal/ai/openai/stream_motoko_test.go`, `internal/effects/ai_motoko.go`, `internal/effects/ai_motoko_test.go`, `internal/builtins/ai_motoko.go`) — now subsumed by M-AI-STREAMING-HELPER
- Custom OpenRouter prefix routing — already upstream as M-AI-OPENROUTER
- Custom OpenAI base-URL routing — now achievable via `[[ai_provider]]` config
- New builtins (`_ai_call_stream_result`) — no longer needed
- `std/ai_motoko.ail` — replaced by `std/ai/streaming` + provider configs

Keeps (likely empty after removals):
- `FORK.md` — repurpose as historical note: "this fork existed pre-v0.17 to add OpenRouter and streaming; both upstream now"
- `scripts/verify_fork_surface.sh` — could move upstream as a generic external-fork validator template

Result: fork can be deleted, or kept as a tag-pinned snapshot for reproducibility.

#### PR-B: migrate consumer (`arniwesth/motoko_agent`)

Title: **"Migrate from forked AILANG to upstream + provider configs"**

Changes:
- `scripts/install-prerequisites.sh`: clone `sunholo-data/ailang` (fix the `motoko` branch silent-bug we found earlier; pin to `v0.17.0` tag)
- `motoko_agent/ailang.toml`: add `[[ai_provider]]` blocks for each provider arni currently uses (OpenRouter, custom-OpenAI-endpoint, etc.) — or import a shared `pkg/arniwesth/motoko_providers` package
- Replace `import std/ai_motoko` → `import std/ai/streaming`
- `ailang.lock`: pin to v0.17.0 compiler hash (M-AILANG-COMPILER-PIN if available; otherwise document the upstream version explicitly)
- `.agent/learnings/` entry documenting the migration path for future fork maintainers
- Eval/benchmark scripts: update to test against upstream AILANG
- CHANGELOG entry

Result: motoko_agent runs on upstream AILANG with no fork binary.

Both PRs land together. PR-B is user-facing; PR-A is cleanup.

### Phase 4 (post-integration): Future work

After arni is on upstream, additional items become tractable:

- **M-AILANG-COMPILER-PIN**: `ailang.toml` `ailang_version` constraint + lockfile compiler hash. Surfaced by motoko's silent-drift bug. Probably v0.18.
- **M-AI-PROVIDER-CONFIG schema v2**: tool-use templating, image input templating, batch endpoint templating. Driven by real usage of v1 plus follow-on consumer feedback.
- **Provider marketplace**: `ailang search --tag ai-provider`. Once the format is stable.
- **Cost-aware routing**: given multiple providers serving the same model, route to cheapest available.
- **Generic fork-validator** (lifting `verify_fork_surface.sh`): if other external consumers want to maintain forks, ship a `ailang fork verify` CLI subcommand.

---

## Risks (cross-cutting)

| Risk | Phase | Impact | Mitigation |
|------|-------|--------|-----------|
| arni ships breaking changes to motoko_agent during our v0.16/v0.17 work, invalidating our PR-B target | All | Med | Phase 3 PRs land *together* and against motoko_agent's `main` at PR time, not against a snapshot |
| v0.16 schema turns out insufficient for arni's actual provider mix | Phase 3 | High | D9: ship + iterate. v2 schema absorbs new shapes without breaking v1 packages |
| arni declines to migrate (prefers the fork) | Phase 3 | Low | The work is independently valuable for any future external consumer. PRs become reference examples even if not merged |
| M-AI-PROVIDER-CONFIG slips, blocking M-AI-STREAMING-HELPER | Phase 2 | High | Streaming slips to v0.18 alongside; doc explicitly flags this dep. Don't compress streaming work to compensate |
| `arniwesth/motoko_agent`'s `install-prerequisites.sh` silent-bug (points at non-existent `sunholo-data/ailang@motoko` branch) is breaking users today | Phase 0 | Low | Documented; PR-B fixes. Could file a quick standalone issue to arni without waiting for full migration if the user count justifies |
| Reference package signals the wrong precedent (e.g. tied too tightly to vLLM) | Phase 1 M4 | Low | Pick a generic name (`ai_openai_compat` rather than `ai_vllm`) so the example reads as "any OpenAI-shape endpoint" not "specifically vLLM" |

---

## Communication

- **Inbound feedback**: existing receive path works (`mcp.ailang.sunholo.com` → SessionStart hook). arni and others can submit feedback via MCP `submit_feedback` tool.
- **Release announcements**: when v0.15.0 and v0.17.0 ship, send a release-note message to arni (and broader external-consumer audience if any) pointing at:
  - The `[[ai_provider]]` schema doc
  - The reference package
  - The `custom-ai-providers.md` guide
  - The `ai-token-streaming` recipe
  - This master doc
- **Migration support**: when arni starts PR-B, expect questions; allocate small bandwidth for direct support during that window. The friction encountered will inform v0.18 schema work.

---

## Related Documents

**Planned (this sequence):**
- [m-ai-provider-config.md](v0_15_0/m-ai-provider-config.md) — Phase 1 milestone
- [m-ai-streaming-helper.md](v0_17_0/m-ai-streaming-helper.md) — Phase 2 milestone
- [m-external-consumer-dx.md](v0_17_0/m-external-consumer-dx.md) — Phase 2 milestone

**Implemented (background):**
- [implemented/v0_16_x/m-ai-openrouter-provider.md](../implemented/v0_16_x/m-ai-openrouter-provider.md) — Phase 0 done
- [implemented/v0_5_10/m-unified-ai-providers.md](../implemented/v0_5_10/m-unified-ai-providers.md) — Provider interface foundation

**Future (post-integration):**
- M-AILANG-COMPILER-PIN (proposed, no doc yet) — Phase 4
- M-AI-PROVIDER-CONFIG v2 schema (proposed, no doc yet) — Phase 4

---

## References

- [arniwesth/ailang FORK.md](https://github.com/arniwesth/ailang/blob/motoko/FORK.md) — fork manifest (23 fence locations, 15 new `_motoko` files, rebase strategy)
- [arniwesth/motoko_agent](https://github.com/arniwesth/motoko_agent) — consumer project (~5,200 LOC `src/core/`, TS/Bun TUI, Python benchmarks, omnigraph data layer)
- BlackMage feedback (2026-05-04, Discord) — external-consumer pain summary
- Session conversation 2026-05-04 — full architectural reasoning chain that led to this sequence

---

**Document created**: 2026-05-04
**Last updated**: 2026-05-04
