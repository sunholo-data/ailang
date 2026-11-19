# M-A2UI-V1 — A2UI v1.0 Adoption: ai-protocol-platform + sunholo/a2ui

**Status**: Planned
**Target**: cross-repo — `sunholo-data/ai-protocol-platform` (post-v6.8 template refresh) + `sunholo-data/ailang-packages` `sunholo/a2ui` 0.4.0. **No AILANG core/stdlib change** (this doc is the program-level record; AILANG-side work routes to the packages lane).
**Priority**: P1
**Estimated**: ~2 weeks platform migration + ~3 days package work (see Timeline)
**Dependencies**: A2UI v1.0 SDK releases (published: `@a2ui/react` 0.15.0, `@a2ui/web_core` latest, `a2ui-agent-sdk` 0.9.0, `a2ui-adk` 0.3.0, `a2ui-core` 0.6.0); A2UI spec v1.0 (a2ui.org, current)

---

## Problem Statement

A2UI (Agent-to-User Interface) protocol **v1.0 is released** (a2ui.org marks v1.0 current, v0.9.1/v0.9 "previous", v0.8 deprecated). Our AI-protocol-platform repo — the public template of the Aitana v6 platform (Skills + AG-UI + A2UI + MCP Apps + A2A on Google ADK) — is pinned end-to-end to **v0.9**:

**Current State (all verified, see Verification Log):**
- Backend: `a2ui-agent-sdk==0.2.2` (`backend/pyproject.toml`), `A2uiSchemaManager(version="0.9")` + `BasicCatalog.get_config("0.9")` (`backend/adk/a2ui.py:76`), a v0.9 wire-format example injected into prompts (`_A2UI_V09_EXAMPLE`), 16 `adk/*.py` files carrying v0.9 payload strings.
- Frontend: `@a2ui/react ^0.10.2` / `@a2ui/web_core ^0.10.5` (pre-v1.0-capable line; npm latest is 0.15.0), renderer capabilities declare `version: "v0.9"` (`A2UIRenderer.tsx:145`), three catalogs (basic + `workbenchCatalog` + `a2uiMarkdownCatalog`) joined by an aliasing shim (`basicCatalogAliases.ts`).
- Custom bidirectional loop: v0.9 has no function-RPC, so the platform built bespoke HTTP routes (`backend/protocols/a2ui_surface_action_routes.py` — a seven-auth-gate action push; `_run` variant bundling an agent turn; data/direct routes) and a result-dict sibling extension (`surface_id`/`update_mode` overlaid on tool results by `SurfaceAwareA2uiToolset`).
- A2A: `backend/protocols/a2a.py` registers `a2ui-v0.9` / `a2ui-basic-catalog-v0.9` extension IDs pointing at pre-fork spec URLs.
- AILANG packages lane: registry package `sunholo/a2ui` (v0.2.0 in registry; 0.3.0 at repo HEAD in `sunholo-data/ailang-packages`) ships `protocol_v091` emitting `"version": "v0.9.1"` envelopes with bare `{"path": …}` data bindings; consumed by `sunholo/ailang_parse`.

**Impact:**
- v0.9 payloads **cannot be used directly with v1.0 protocol runtimes** (spec: catalogs authored for v0.9 break under v1.0 due to structural changes — envelope-level `ComponentCommon`, `ValidationResult` checks, caller/composition constraints). Every external A2UI counterparty that upgrades (the 2026-11-05 integrations review already listed "A2UI v0.8/v0.9" for Gemini Enterprise — that surface will move) stops interoping with us.
- v0.8 is already **deprecated**; staying on v0.9 puts us one deprecation cycle behind with no negotiated upgrade path.
- The features v1.0 standardizes are exactly the ones the platform hand-rolled (bidirectional RPC vs. the seven-gate action endpoint; catalog mixing vs. the aliasing shim; `@metadata` extensions vs. result-dict siblings).

## What A2UI v1.0 Adds (Feature Inventory)

From the official [v1.0 Evolution Guide](https://a2ui.org/specification/v1.0-evolution-guide/) and [messages reference](https://a2ui.org/concepts/messages/) (full changes list is long; these are the items material to our repos):

1. **Bidirectional function RPC** — `callRendererFunction` (agent→renderer) returning `rendererFunctionResponse`, and `callAgentFunction` (renderer→agent) returning `agentFunctionResponse`; shared `FunctionResponse` payloads; runtime-verified `allowedCallers` (`rendererOnly`/`agentOnly`/`rendererOrAgent`) and `requiresUserActivation` declared in catalogs. Envelope `version` string is now `"1.0"` (no `v` prefix).
2. **Catalog mixing within one surface** — `supportedCatalogIds` are mixable; per-component `@catalogId` / per-call `catalogId`; explicit resolution order (explicit → surface default → error; no capabilities fallback). Reserved `"Surface"` root; `allowedParents`/`allowedChildren` composition constraints.
3. **Single-message UI creation** — `createSurface` accepts initial `components` and `dataModel` inline (plus optional `metadata.extensions`; `catalogId` now optional = surface default).
4. **Reserved-`@` directive namespace** — data bindings become `{"@path": …}`, calls `{"@call": …}`, template index `@index` (with `offset`), envelope mixin keys `@metadata`/`@catalogId`/`@accessibility`/`@checks`; literal `@` keys escaped by doubling (`"@@path"`). Bare `"path"`/`"call"` keys in plain objects are literal data again.
5. **`ValidationResult`** — typed check results (`valid`, `code`, `message`, `severity`) from validation functions/data bindings; `CheckRule.message` now optional fallback; `DynamicBoolean` coercion rules.
6. **Data-model deletion semantics** — `updateDataModel.value` required; `null` deletes the key at the path.
7. **Component updates** — `TextField`: `value` now required, `placeholder` added, `validationRegexp` removed (use `@checks` + `regex`); `Slider.steps`; `Video.posterUrl`; `openUrl` is `requiresUserActivation: true`.
8. **`@metadata` extensions** — arbitrary extension key-value pairs on components, surfaces, catalog definitions, and actions (Unicode UAX #31 keys, reserved `a2ui_` namespace).
9. **Accessibility** — `@accessibility` gains WAI-ARIA `live` (`off`/`polite`/`assertive`) and `hidden`; normative plumbing requirements.
10. **Catalog-schema hygiene** — `protocolVersion: "1.0"` required in catalog roots; functions as a formalized name-keyed map; inline `instructions` (Markdown) replaces external `rules.txt`; standard JSON-Schema metadata fields allowed; UAX #31 identifier rules; `x-deprecated-reason`.
11. **Action events** — `action` gains optional `userMessage` (shown in chat) and `metadata.extensions`.
12. **Transport/A2A** — official MIME type `application/a2ui+json`; capabilities namespace bumped to `1.0`; A2A extension spec relocated under `v1_0/extensions/a2a/`; `renderer_capabilities.json` simplified.
13. **Terminology** — client→_renderer_, server→_agent_ across all schema filenames (`agent_to_renderer.json`, `renderer_to_agent.json`, etc.).

**Explicit non-issues for us** (verified): `theme`/`primaryColor` removal — the platform uses neither; MIME type — the platform doesn't emit A2UI MIME types today (payloads ride AG-UI tool events), though the A2A surface should adopt `application/a2ui+json`.

## What We Can Use (Mapping Table)

| # | v1.0 feature | Where it lands in our repos | Value |
|---|---|---|---|
| U1 | Bidirectional function RPC (`callAgentFunction`/`agentFunctionResponse`, `allowedCallers`) | Replaces the bespoke `a2ui_surface_action_routes.py` / `-run` / data routes envelope; the seven auth gates map to `allowedCallers` + existing middleware; elicitation's authoritative `read_submitted_values` loop gains a protocol-standard return channel | Deletes a custom wire format we maintain; standard interop |
| U2 | Catalog mixing + `@catalogId` | `workbenchCatalog` + `a2uiMarkdownCatalog` + basic catalog in one surface; retire `basicCatalogAliases.ts` aliasing shim | Removes a whole class of name-collision bugs (the aliasing shim exists only because v0.9 couldn't mix) |
| U3 | `@metadata` extensions | `surface_id`/`update_mode` result-dict siblings (the `SurfaceAwareA2uiToolset` overlay) become protocol-legal surface metadata instead of out-of-band keys | Our extension mechanism upstreamed into the spec |
| U4 | Single-message `createSurface` (inline components + dataModel) | All ~14 `adk/a2ui_*_render.py` modules emit createSurface→updateComponents→updateDataModel triplets; `reviewSurface` in `sunholo/a2ui` returns 3 envelopes → 1 | Fewer messages, fewer partial-render states, less prompt tokens |
| U5 | `@path`/`@call`/`@index` reserved namespace + `@@` escaping | All payload emitters (3 backend files with bare bindings today; the a2ui package's `binding()` helper) | Unambiguous directive-vs-data (kills the v0.9 "plain object with a `path` key is intercepted" trap) |
| U6 | `ValidationResult` + `@checks` | Migrate `validationRegexp` (ppa render, elicitation tests) to `@checks`+`regex`; typed check results for obligation/elicitation forms | Typed, localizable validation with error codes |
| U7 | Composition constraints (`allowedParents`/`allowedChildren`, `Surface`) | `workbenchCatalog` declares its layout rules; renderer reports `UNALLOWED_PARENT`/`UNALLOWED_CHILD` | Catalog self-describes; LLM authors get feedback instead of mis-nests |
| U8 | Catalog `instructions` field | `workbenchCatalog` embeds its usage rules instead of prompt-side notes; feeds the Agent-SDK's catalog-driven prompt generation | Prompt stays in sync with components automatically |
| U9 | Accessibility `@accessibility` live/hidden | Chat surfaces + workspace artefact tabs | Cheap a11y win; spec now normative |
| U10 | `action.userMessage` | Surface actions render a user-facing line in chat (today synthesized ad-hoc by `A2UIRenderer.onAction`) | Standardized |
| U11 | A2A extension v1.0 + MIME | `backend/protocols/a2a.py` extension IDs (`a2ui-v0.9` → `1.0` namespace); A2UI-over-A2A artifacts use `application/a2ui+json` | Cross-trust-boundary interop with v1.0 counterparties (Gemini Enterprise et al.) |
| U12 | Protocol helpers in AILANG | `sunholo/a2ui` 0.4.0: new `protocol_v1` module (`"1.0"` envelopes, `@path`/`@call`, `jn()` deletion, RPC message builders, updated `decodeAction`) | AILANG programs (`sunholo/ailang_parse` consumers) speak the current protocol; **zero AILANG core change needed** (verified, V12) |

## Goals

**Primary Goal:** Bring the ai-protocol-platform and the `sunholo/a2ui` package to A2UI v1.0, and adopt the v1.0 features that replace our hand-rolled v0.9 workarounds (bidirectional RPC, catalog mixing, metadata extensions, single-message surfaces).

**Success Metrics:**
- Platform backend emits v1.0 envelopes that pass the official v1.0 JSON-schema validation (AJV against `https://a2ui.org/specification/v1_0/` schemas), on the v1.0-capable SDK pins.
- Frontend renders v1.0 surfaces and negotiates `"1.0"` capabilities; all existing workshop demos and skills work unchanged end-to-end.
- The bespoke surface-action wire envelope is either replaced by or explicitly bridged to `callAgentFunction` (Decision D2) with the same auth guarantees (seven gates preserved in behavior).
- `sunholo/a2ui@0.4.0` published: `protocol_v1` module passes `ailang check --package .`, Discord-demo AJV pins updated to v1.0 schemas, both directions validated.
- No regression in `parseA2UIResult` chat rendering; MessageBubble/surface vitest suites green.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Cutover style — hard switch to v1.0 vs staged dual-version (upgrade SDKs, negotiate `1.0`, keep v0.9.1 rendering during transition) | Renderer SDKs route payloads per-envelope `version`; a staged path keeps every live skill working while renders migrate one by one | **human** | design | high |
| D2: Map the seven-gate surface-action endpoint onto `callAgentFunction` RPC (same HTTP transport, protocol envelope) vs keep the bespoke loop and adopt RPC only for new features | Security-sensitive: auth model must survive the move (actions are user-driven, context ≤4096B, per-skill opt-in `allow_surface_context_writes`) | **human** | design | high |
| D3: Move `surface_id`/`update_mode` result-dict siblings into `@metadata` extensions vs keep as out-of-band keys | Spec-legal vs custom; affects frontend `A2UISurfaceMount` read path | agent | design | low |
| D4: `sunholo/a2ui` — additive `protocol_v1` module (semver-minor 0.4.0, `protocol_v091` retained) vs in-place migration | Registry consumers (`sunholo/ailang_parse`) keep working either way; additive is reversible | agent | design | med |
| D5: Retire `basicCatalogAliases.ts` once mixing lands | Alias shim is pure workaround; removal touches every custom catalog registration | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] D1 — cutover style (human)
- [ ] D2 — action-endpoint → `callAgentFunction` mapping (human)

## Solution Design

### Overview

Three tracks, one per repo lane, ordered so that a renderable v1.0 path exists before backend emitters migrate:

1. **SDK/track upgrade first** (platform): bump `a2ui-agent-sdk`/`a2ui-adk`/`a2ui-core` and `@a2ui/react`/`@a2ui/web_core` to the v1.0-capable lines; renderer keeps accepting v0.9.1 envelopes (Core SDKs parse all supported protocol versions) so migration is per-skill, not big-bang.
2. **Payload migration** (platform backend): centralize wire emission (catalog pin, prompt example, envelope builders) in `adk/a2ui.py`; sweep the render modules for the mechanical v1.0 changes (`"version": "1.0"`, `@path`/`@call`, `@@` escaping, `@checks`, TextField `value`); update A2A extension IDs and MIME.
3. **Feature adoption** (platform + packages): `callAgentFunction` bridging, catalog mixing / alias retirement, `@metadata` for surface routing, single-message `createSurface`, and `sunholo/a2ui` 0.4.0 `protocol_v1`.

### Architecture

**Components:**
1. **`backend/adk/a2ui.py`** — single source of protocol version truth: `BasicCatalog.get_config("1.0")`-era config, regenerated `_A2UI_V10_EXAMPLE` prompt anchor (the v0.9 example exists because models confabulate the wire shape; it must be rewritten and re-validated, not just string-swapped — the evolution guide's WARNING forbids blanket `path`→`@path` find-and-replace).
2. **Renderer capabilities** (`A2UIRenderer.tsx` / `A2UISurfaceMount.tsx`) — declare `version: "1.0"` + mixable `supportedCatalogIds`; envelope-version routing already provided by the upgraded Core SDK.
3. **Action bridge** (`backend/protocols/a2ui_surface_action_routes.py` family) — same endpoints, same seven gates; payload becomes `callAgentFunction`/`agentFunctionResponse` envelopes (D2); `allow_surface_context_writes` continues to be the trust grant, `allowedCallers` the catalog-side mirror.
4. **`sunholo/a2ui` package** — additive `protocol_v1` module mirroring `protocol_v091`'s shape (pure functions only, no renderer/network — unchanged doctrine).

### Implementation Plan

**Phase 1: Pin upgrades + negotiation (platform)** (~2 days)
- [ ] Bump `backend/pyproject.toml`: `a2ui-agent-sdk`, `a2ui-adk`, `a2ui-core` (verify `google-adk==2.12.0` compat with `a2ui-adk` 0.3.0 in a spike first)
- [ ] Bump `frontend/package.json`: `@a2ui/react`, `@a2ui/web_core`; run vitest + typecheck
- [ ] Confirm renderer accepts both `"1.0"` and `"v0.9.1"` envelopes; declare `"1.0"` in capabilities
- [ ] Spike: emit one hand-written v1.0 surface (inline components + dataModel) end-to-end

**Phase 2: Backend payload migration (platform)** (~4 days)
- [ ] `adk/a2ui.py`: catalog pin → v1.0; write `_A2UI_V10_EXAMPLE` (mixed-envelope example incl. `@path`, `@@` escaping, `@checks`); SurfaceAware toolset unchanged until D3
- [ ] Sweep `adk/a2ui_*_render.py` (16 files with v0.9 strings; bare `{"path": …}` bindings in `a2ui.py`, `a2ui_inbox_render.py`, `a2ui_ppa_render.py`; `{"call": …}` in inbox + project renders): version literal, directive prefixes, envelope mixin keys
- [ ] Migrate `validationRegexp` (ppa render) → `@checks` + `regex`; ensure every `TextField` carries a bound `value`
- [ ] `backend/protocols/a2a.py`: extension IDs → v1.0 namespace/URLs; adopt `application/a2ui+json` for A2UI-over-A2A parts
- [ ] Update backend unit tests (`test_elicitation.py` etc.) + add AJV validation of emitted payloads against pinned v1.0 schemas

**Phase 3: Feature adoption (platform)** (~4 days)
- [ ] D2 outcome: `callAgentFunction` bridge for surface actions (or documented bridge plan if deferred); elicitation read-back unchanged (data-model paths, not protocol)
- [ ] Catalog mixing: register basic+workbench+markdown mixable; retire `basicCatalogAliases.ts` (D5); declare `allowedParents`/`allowedChildren` + `instructions` in `workbenchCatalog`
- [ ] D3 outcome: surface routing via `@metadata` (or keep siblings, documented)
- [ ] Collapse render triplets to single-message `createSurface` (pick the 2–3 hottest renderers first; rest optional)
- [ ] Adopt `action.userMessage`; add `@accessibility` live regions to chat-rendered surfaces

**Phase 4: `sunholo/a2ui` 0.4.0 (ailang-packages)** (~3 days)
- [ ] Add `packages/a2ui/protocol_v1.ail`: `v1Envelope` (`"1.0"`), `binding(path)` → `{"@path": …}`, `call(...)` → `{"@call": …}`, `escapeKey`, `v1DeleteKey` (`jn()`), `createSurface` with optional inline components/dataModel, `callRendererFunction`/`agentFunctionResponse` builders, `decodeAction` updated for optional `userMessage`/`metadata`
- [ ] Keep `protocol_v091` untouched (D4); bump `ailang.toml` to 0.4.0; update `AGENT.md`
- [ ] Update the Discord demo's pinned AJV schemas to v1_0; validate both directions
- [ ] `ailang check --package .`; publish to registry; note in `sunholo/ailang_parse` consumer

### Files to Modify/Create

**ai-protocol-platform — new:**
- `backend/tests/unit/test_a2ui_v10_wire.py` — envelope/schema conformance (~150 LOC)
- `frontend/src/components/protocols/__tests__/A2UIRenderer.v1.test.tsx` — v1.0 envelope routing (~120 LOC)

**ai-protocol-platform — modified:**
- `backend/pyproject.toml` — SDK pins (~3 lines)
- `backend/adk/a2ui.py` — catalog pin, v1.0 example block, module docstring (~120 LOC delta)
- `backend/adk/a2ui_*_render.py` (×14) + `adk/agent.py`, `adk/callbacks.py` — envelope sweep (~10–40 LOC each)
- `backend/adk/a2ui_ppa_render.py`, `adk/elicitation.py` — `@checks` migration (~40 LOC)
- `backend/protocols/a2a.py` — extension IDs + MIME (~15 LOC)
- `backend/protocols/a2ui_surface_action_routes.py` (+ `_run`, `_data`, `_direct` siblings) — RPC bridging per D2 (~100 LOC)
- `frontend/package.json` — pins (~3 lines)
- `frontend/src/components/protocols/A2UIRenderer.tsx` — capabilities `"1.0"` (~10 LOC)
- `frontend/src/components/protocols/basicCatalogAliases.ts` — retire (D5; delete)
- `frontend/src/components/protocols/workbenchCatalog.tsx`, `a2uiMarkdownCatalog.tsx` — mixable registration, constraints, `instructions` (~60 LOC)

**ailang-packages — new:**
- `packages/a2ui/protocol_v1.ail` (~120 LOC, pure)

**ailang-packages — modified:**
- `packages/a2ui/AGENT.md`, `ailang.toml` (0.4.0) (~20 LOC)

**AILANG repo (this repo):** none beyond this document.

## Examples

### Example 1: Data binding + envelope (the mechanical sweep)

**Before (v0.9, `adk/a2ui_result_render.py`-style):**
```json
[{"version": "v0.9",
  "updateComponents": {"surfaceId": "workspace", "components": [
    {"id": "root", "component": "Column", "children": ["count"]},
    {"id": "count", "component": "Text", "text": {"path": "/activeUsers"}, "variant": "h3"}]}}]
```

**After (v1.0):**
```json
[{"version": "1.0",
  "updateComponents": {"surfaceId": "workspace", "components": [
    {"id": "root", "component": "Column", "children": ["count"]},
    {"id": "count", "component": "Text", "text": {"@path": "/activeUsers"}, "variant": "h3"}]}}]
```
Note: only `DataBinding`/`FunctionCall` positions take `@`; `ChildList` template `path`, and the `updateDataModel` envelope's `path` field, stay unprefixed. **No blanket find-and-replace** (spec WARNING).

### Example 2: Single-message surface (U4)

**Before (v0.9 triplet, what `reviewSurface` and the render modules emit):**
```json
[{"version": "v0.9", "createSurface": {"surfaceId": "s", "catalogId": "https://a2ui.org/specification/v0_9/basic_catalog.json"}},
 {"version": "v0.9", "updateComponents": {"surfaceId": "s", "components": [ … ]}},
 {"version": "v0.9", "updateDataModel": {"surfaceId": "s", "value": { … }}}]
```

**After (v1.0, one message):**
```json
[{"version": "1.0", "createSurface": {
  "surfaceId": "s",
  "catalogId": "https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json",
  "components": [ … ],
  "dataModel": { … },
  "metadata": {"extensions": {"aitana:default_update_mode": "replace"}}}}]
```

### Example 3: AILANG package helper (U12)

**Before (`packages/a2ui/protocol_v091.ail`):**
```ail
export pure func binding(path: string) -> Json = jo([kv("path", js(path))])
```

**After (new `protocol_v1.ail`; compiles today — Verification Log V12):**
```ail
export pure func binding(path: string) -> Json = jo([kv("@path", js(path))])
export pure func v1Envelope(name: string, payload: Json) -> Json =
  jo([kv("version", js("1.0")), kv(name, payload)])
export pure func deleteKey(path: string) -> Json =
  v1Envelope("updateDataModel", jo([kv("surfaceId", js(sid)), kv("path", js(path)), kv("value", jn())]))
```

## Success Criteria

- [ ] All platform A2UI payloads validate against the pinned official v1.0 JSON schemas (AJV, both envelope directions) — acceptance: new wire test suite green
- [ ] Renderer negotiates `"1.0"` and still renders v0.9.1 envelopes during the transition window — acceptance: `A2UIRenderer.v1.test.tsx` + existing surface tests green
- [ ] Workshop demos (`make dev` + seeded skills) render end-to-end on v1.0 — acceptance: manual pass per WORKSHOP.md path
- [ ] Surface actions still require the same seven auth gates (D2 outcome implemented or bridged) — acceptance: route tests assert 401/403/404/413 paths unchanged
- [ ] `sunholo/a2ui@0.4.0` published with `protocol_v1`; `ailang check --package .` clean; Discord demo validates v1.0 envelopes both directions
- [ ] All tests passing (backend pytest, frontend vitest + typecheck + lint)
- [ ] Documentation updated (platform `docs/design/` migration note; package AGENT.md)
- [ ] No AILANG core/stdlib/compiler change (boundary respected)

## Testing Strategy

**Unit tests:**
- Envelope builders emit `"1.0"` + correct directive keys (per-message golden tests)
- `decodeAction` accepts v1.0 action incl. optional `userMessage`/`metadata`; rejects malformed
- Elicitation/`@checks`: `ValidationResult` objects flow; `regex` replaces `validationRegexp`

**Integration tests:**
- AJV 2020 validation of every emitted message type against pinned `v1_0` schemas (pin exact schema SHAs as the Discord demo already does)
- Renderer negotiation: mixed `"1.0"`/`"v0.9.1"` session; catalog-mixed surface (basic + workbench component in one tree)
- Action bridge: `callAgentFunction` → gate checks → `agentFunctionResponse` round-trip

**Manual testing:**
- WORKSHOP.md quickstart (`LOCAL_MODE=1 make dev`) on migrated skills; elicitation form submit loop; workspace/surface persistence across turns

## Deferred Decisions

The following are intentionally left open for the implementer:

- Which render modules collapse to single-message `createSurface` first (beyond the mandated 2–3) — agent may choose by traffic
- Whether to adopt A2UI Express / server-side `@macro` inference formats (Agent SDK feature) for token reduction — agent may spike, separate doc if promoted
- Custom-catalog `Surface`-rooted composition rule details (`allowedParents` lists per component) — agent may choose, validated by renderer error codes
- Whether the AILANG TUI ever consumes A2UI via a Go community renderer (charmbracelet et al.) — out of this doc, see Future Work

## Non-Goals

- **Any AILANG core/stdlib/compiler change** — v1.0 helpers are pure std/json usage (V12); frozen core stays frozen
- **v0.8 support** — deprecated upstream; nothing in our repos uses it
- **MCP Apps / AG-UI rework** — adjacent protocols, untouched except where `@a2ui/catalog-mcp` style mixing benefits fall out of U2 for free
- **Theme/branding system** — v1.0 *removed* theme from the protocol; we never used it (V6)
- **A2UI-over-MCP layouts** (`a2ui-over-mcp` guide) — separate integration, separate doc if wanted
- **Adopting every basic-catalog nicety** (`Slider.steps`, `Video.posterUrl`) — opportunistic, not gated

## Timeline

**Week 1** (~4 days): Phase 1 (pins + negotiation + spike) and Phase 2 (backend sweep) start; D1/D2 frozen at the top of the week
**Week 2** (~4 days): Phase 2 finish + tests; Phase 3 (RPC bridge, mixing, metadata)
**Week 3** (~3 days): Phase 4 (`sunholo/a2ui` 0.4.0), docs, registry publish, manual passes, release note in platform template

**Total: ~11 engineering days across 3 weeks** (2× the naive estimate, per house convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| SDK jump regressions (`@a2ui/react` 0.10→0.15; `a2ui-agent-sdk` 0.2.2→0.9.0 crosses many minors) | High | Phase 1 spike behind flags; dual-version window (D1 staged default); vitest surface suite is the tripwire |
| `google-adk==2.12.0` incompatibility with `a2ui-adk` 0.3.0 | High | Spike before any sweep; pin `a2ui-adk` 0.2.x + manual v1.0 envelopes as fallback (our toolset is already a subclass — the wrapper pattern held across SDK minors before) |
| LLM wire-format confabulation returns after the example-block rewrite | Med | Rewrite `_A2UI_V10_EXAMPLE` from a validated live payload; add a model-facing eval (one hand-written v1.0 render task per provider) before rollout |
| Action-bridge auth regression (seven gates) | High | Gate-for-gate route tests written first (characterization), kept green through the bridge; `allowedCallers` is additive, never a replacement for the HTTP gates |
| `sunholo/ailang_parse` breaks on package changes | Med | D4 additive module; 0.3.0 stays importable; consumer smoke test in the publish checklist |
| Registry/schema pinning drift (a2ui.org schemas move) | Low | Pin exact schema SHAs (existing Discord-demo practice) |

## Axiom Compliance

This design changes no AILANG language surface; the AILANG-relevant artifact is the pure `sunholo/a2ui` package update. Scoring applies to that lane (platform-repo work is governed by the platform's own docs).

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pure JSON builders; same input → byte-identical envelope |
| A2: Replayability | 0 | No trace/execution change |
| A3: Effect Legibility | +1 | `protocol_v1` stays `pure`; no new effects |
| A4: Explicit Authority | +1 | v1.0 moves function-call authorization into runtime-verified `allowedCallers`; matches the package doctrine "UI actions do not confer authority" |
| A5: Bounded Verification | +1 | `ailang check --package .` + AJV schema pinning give local, complete verification of wire output |
| A6: Safe Concurrency | 0 | No concurrency |
| A7: Machines First | +1 | Declarative, schema-validatable UI JSON is exactly the machine-decidable surface A2UI is for |
| A8: Minimal Syntax | +1 | No syntax change; reuses std/json (V12) |
| A9: Cost Visibility | 0 | No resource semantics |
| A10: Composability | +1 | Envelope builders compose; catalog mixing is upstream composability |
| A11: Structured Failure | +1 | `decodeAction`/`Result` unchanged in shape; v1.0 adds typed `ValidationResult` and error codes (`INVALID_FUNCTION_CALL`, `UNALLOWED_PARENT`) |
| A12: System Boundary | +1 | Explicit envelope version on every message; no ambient transport assumptions |

**Net Score: +8** → **Decision: Move forward** (subject to D1/D2 human freeze)

### Hard Violation Check

- [x] A1: no implicit nondeterminism
- [x] A3: no hidden side effects (pure module)
- [x] A4: no ambient access granted (trust grants stay opt-in flags)
- [x] A7: machine-decidable wire format, not human-convenience syntax

## Conflict Surface

Not applicable: this design touches no AILANG parser/typechecker/codegen/eval surface (no `internal/` or `cmd/` change). The AILANG-side change is an additive registry package module; platform-repo changes are governed by that repo's own design docs (`docs/design/v6.x`).

## Verification Log

| # | Claim | Verification | Result |
|---|-------|--------------|---------|
| V1 | Platform backend pinned to A2UI v0.9 | `grep` `backend/pyproject.toml`: `a2ui-agent-sdk==0.2.2`; `backend/adk/a2ui.py:76` `A2uiSchemaManager(version="0.9")`, `BasicCatalog.get_config("0.9")`; 16 `backend/adk/*.py` files carry v0.9 strings | Confirmed |
| V2 | Platform frontend pinned pre-v1.0 and declares v0.9 | `frontend/package.json`: `@a2ui/react ^0.10.2`, `@a2ui/web_core ^0.10.5`; `frontend/src/components/protocols/A2UIRenderer.tsx:145` `version: "v0.9"` | Confirmed |
| V3 | v1.0-capable SDK lines are published | npm registry: `@a2ui/react` latest **0.15.0**; PyPI: `a2ui-agent-sdk` latest **0.9.0**, `a2ui-adk` **0.3.0**, `a2ui-core` **0.6.0** | Confirmed |
| V4 | Platform uses bare `{"path": …}` data bindings | `grep '"path": "/' backend/adk/*.py` → `a2ui.py`, `a2ui_inbox_render.py`, `a2ui_ppa_render.py` | Confirmed |
| V5 | Platform uses `TextField.validationRegexp` | `grep` → `backend/adk/a2ui_ppa_render.py` (+ elicitation/ChatPlacementForms tests) | Confirmed |
| V6 | Platform never uses `theme`/`primaryColor` (negative) | `grep -rn "primaryColor\|\"theme\"" backend/adk backend/skills frontend/src` → zero hits | Confirmed |
| V7 | Platform emits no A2UI MIME type today (negative) | `grep -rn "a2ui+json\|json+a2ui" backend frontend/src` → zero hits (payloads ride AG-UI tool events) | Confirmed |
| V8 | Surface-action loop is a bespoke HTTP endpoint, not protocol RPC | Read `backend/protocols/a2ui_surface_action_routes.py` header: v0.9 `client_to_server` action push, seven auth gates, 4096-byte context cap | Confirmed |
| V9 | Platform runs 3 catalogs + an aliasing shim | `ls frontend/src/components/protocols/` → `workbenchCatalog.tsx`, `a2uiMarkdownCatalog.tsx`, `basicCatalogAliases.ts` | Confirmed |
| V10 | A2A registers v0.9 extension IDs | `grep` `backend/protocols/a2a.py:88-101` → `a2ui-v0.9`, `a2ui-basic-catalog-v0.9`, pre-fork URLs | Confirmed |
| V11 | `sunholo/a2ui` emits v0.9.1 envelopes with bare bindings | Read `packages/a2ui/protocol_v091.ail` in sunholo-data/ailang-packages: `version "v0.9.1"`, `binding(path) = {"path": …}`, v0_9_1 catalog URL; `ailang pkg info sunholo/a2ui` → registry at 0.2.0, repo HEAD 0.3.0 | Confirmed |
| V12 | v1.0 wire helpers are expressible in current AILANG with no language change | `ailang check` on a claim module using std/json (`@path` string keys, `"1.0"` version literal, `jn()` null deletion): **"✓ No errors found"**. Caveat: the installed binary is v0.60.1-era and rejects std@v0.60.2 (`@experimental`), so the check ran against a `git archive` of tag `v0.60.1` stdlib; the claim module imports only std/json APIs (`jo/kv/js/jn/encode`) that exist in both | Confirmed |
| V13 | v1.0 message inventory and change list | a2ui.org `concepts/messages.md` section headers (6 agent→renderer incl. 2 new; 3 renderer→agent incl. 2 new) + [v1.0 Evolution Guide](https://a2ui.org/specification/v1.0-evolution-guide/) | Cited |
| V14 | No open AILANG PR covers this request | Reviewed the 30+ open PRs listed with the directive: retirement follow-ups, dead code, platform split docs, CodeQL/Sonar, audit metrics, fleet records, match exhaustiveness, show-total, IFC, dependabot — none touch A2UI | Confirmed |
| V15 | v1.0 envelopes use `"1.0"` (no `v` prefix) and single-message `createSurface` exists | a2ui.org `concepts/messages.md`: envelope examples `"version": "1.0"`; `createSurface` v1.0 example carries `components`/`dataModel` inline | Cited |

## Related Documents

**Implemented (may inform design):**
- [m-notify-channels-framework.md](implemented/v0_23_0/m-notify-channels-framework.md) — the fail-closed, opt-in gating doctrine the action bridge preserves
- [m-std-jsonschema-sprint-plan.md](implemented/v0_57_1/m-std-jsonschema-sprint-plan.md) — registry dependency review noting `sunholo/a2ui` ("renders UI JSON, not schema validation")

**Planned (check for overlap):**
- [integrations-review-external-2026-11-05.md](planned/integrations-review-external-2026-11-05.md) — records "A2UI v0.8/v0.9" external support; this doc is the upgrade answer for our side (distinct: adoption/migration, not a capability survey)

## References

- [A2UI v1.0 Evolution Guide](https://a2ui.org/specification/v1.0-evolution-guide/) — canonical v0.9→v1.0 change list (all §"What v1.0 adds" claims)
- [A2UI Messages](https://a2ui.org/concepts/messages/) · [SDKs](https://a2ui.org/concepts/sdks/) · [v1.0 spec](https://a2ui.org/specification/v1.0-a2ui/) · [A2A extension](https://a2ui.org/specification/v1.0-a2ui-extension-specification/)
- `sunholo-data/ai-protocol-platform` @ HEAD `722e257`: `backend/adk/a2ui.py`, `backend/protocols/a2ui_surface_action_routes.py`, `frontend/src/components/protocols/`
- `sunholo-data/ailang-packages` @ HEAD: `packages/a2ui/protocol_v091.ail`, `AGENT.md`
- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [design_docs/PROGRAM.md](../PROGRAM.md) — routing: this work rides the packages/platform lanes; core stays frozen

## Future Work

- **AILANG TUI rendering of A2UI** via a Go community renderer (Bubble Tea/Ink line) — coordinator/mission agents could push live surfaces to the terminal without a browser
- **A2UI Express inference format** in the platform's agent prompts (token reduction, Agent SDK compile path)
- **`sunholo/a2ui` reviewSurface v2** using single-message `createSurface` + `@metadata` extensions end-to-end
- **A2UI over A2A** with `sunholo/ailang_parse` as a remote counterparty (cross-trust-boundary declarative review surfaces)

---

**Document created**: 2026-11-12
**Last updated**: 2026-11-12
**Author**: design-doc-creator (Daneel task-88a4015f), on the user request "A2UI v1.0 — look at our ai protocol platform repo and see what features it adds or we can use; design doc it"
