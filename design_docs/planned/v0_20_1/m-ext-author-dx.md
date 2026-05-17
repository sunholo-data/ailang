# M-EXT-AUTHOR-DX: Close the motoko_ext author-loop gaps

**Status**: Planned
**Target**: v0.20.1
**Priority**: P1
**Estimated**: ~1 day (~8 hours)
**Dependencies**: None — builds on M-EXT-SCAFFOLD-AI-FIRST (v0.18.5), M-EXT-PORTABILITY-GATE (v0.18.11), and the existing path-dependency machinery in `internal/pkg/`.
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-17
**Source**: 2026-05-17 [arniwesth/motoko_agent#23 discussion](https://github.com/arniwesth/motoko_agent/discussions/23) — Arni's extension development proposal + PR #22 (fix_context_mode_extension). He asked 6 questions; 2 already have crisp answers in shipped code; 4 surface real gaps in our author DX that this sprint closes.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | 0 | DX/tooling change |
| A2 Replayability | 0 | No effect on traces |
| A3 Effect Legibility | +1 | Schema-placeholder + naming-gate push effect-relevant info into manifest+scaffold rather than discovered-by-Bedrock-failure-in-production |
| A4 Explicit Authority | 0 | No capability-system change |
| A5 Bounded Verification | +1 | Registry-validator naming gate catches a bug class at publish time that today only surfaces in downstream consumer integration tests |
| A6 Safe Concurrency | 0 | N/A |
| A7 Machines First | +2 | AI agents authoring extensions are the primary audience. Scaffold-with-schema + scaffold-with-smoke move median first-attempt success up; naming-gate prevents an entire class of "looked fine, broke in Bedrock" reports |
| A8 Minimal Syntax | 0 | No syntax change |
| A9 Cost Visibility | 0 | N/A |
| A10 Composability | +1 | Officially deprecating `scripts/sync-extension-packages.sh` (replaced by path-deps) removes a parallel mechanism |
| A11 Structured Failure | +1 | The Bedrock dotted-name failure is today a runtime error in user-facing demos; gate moves it to publish time |
| A12 System Boundary | 0 | N/A |

**Net Score: +6** → ✅ Move forward. A1/A3/A4/A7 all clean.

---

## Problem Statement

Arni's discussion #23 lays out the correct extension author workflow:
1. Implement in `ailang-packages/packages/motoko-ext-<name>/`
2. Consume from `motoko_agent` via path-dep in `ailang.toml`
3. Run `ailang lock` + `ailang generate-extension-registry`
4. Smoke-test against the host
5. Publish, then switch the dep back to a version pin

He asked 6 questions. Independent verification (2026-05-17):

| # | Question | Status |
|---|---|---|
| 1 | `generate-extension-registry` resolves path-deps + emits the right `pkg/...` import? | **✅ Yes** — verified end-to-end against `motoko_ext_test_dummy`. Lock reports `(path: ...)`, generated registry emits `import pkg/sunholo/motoko_ext_test_dummy/register`. |
| 2 | Docs explicitly recommend the path-dep workflow? | **❌ Gap** |
| 3 | `ailang init motoko-extension` scaffold smoke + `on_describe_tools` + naming guidance + publish checklist? | **❌ Partial** — register.ail stubs only |
| 4 | Registry validator enforces provider-safe tool names? | **❌ Gap** — manual fix shipped at `5e2eaef` after Anthropic Bedrock rejected dotted names |
| 5 | `motoko_agent/scripts/sync-extension-packages.sh` deprecated now? | **❌ Should officially be** |
| 6 | `ailang check --package .` is the package-wide gate? | **✅ Yes** — shipped in M-EXT-PORTABILITY-GATE |

The 4 gaps cost Arni real time: PR #22 ships a local-override fix for `context_mode` that bypasses the registry path entirely because the gaps made the registry path frustrating to use. That's the symptom we're closing.

There's also a **concrete delivery** sitting in PR #22: the published `sunholo/motoko_ext_context_mode@0.2.1` is effectively a stub (no real `provided_tools`, no `on_tool_handle`, no schemas). Arni built the full version locally; we need to port it back to the registry package and publish 0.2.2.

---

## Goals

**Primary Goal**: Close every author-DX gap that pushed Arni toward a local override. After this sprint, the canonical loop (path-dep → develop → smoke → publish → version-pin) works without friction, is the documented + scaffolded default, and the registry catches the obvious naming traps.

**Success Metrics**:
- `docs/docs/guides/motoko-extension-development.md` ships with the exact path-dep workflow Arni proposed
- `ailang init motoko-extension` generates: register.ail (existing) + `_smoke.ail` template + `on_describe_tools` schema placeholders + README with provider-safe naming guidance + publish checklist
- Registry validator hard-fails publish if any advertised tool name contains a dot or other characters known to break Anthropic Bedrock
- `--allow-dotted-tool-names` migration flag exists (mirrors `--allow-unsafe-field-access` pattern from M-TYPECHECK)
- `motoko_agent/scripts/sync-extension-packages.sh` shows a deprecation warning
- `motoko_ext_context_mode 0.2.2` published with Arni's full implementation (ported from PR #22)
- Reply posted to discussion #23

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Naming gate: hard-fail or warn first? | Hard-fail could surprise authors of already-published packages on a patch republish | human | design | high |
| Smoke template: minimal (register-only) or rich (per-tool dispatch + policy)? | Rich is what Arni used; minimal is faster to scaffold | agent | scaffold | low |
| Schema placeholder format | Empty `{}` silently degrades; `{"type":"object","required":[],"properties":{}}` actually works | agent | scaffold | low |

### Design Freeze
- [ ] **Naming gate severity**: hard-fail in v0.20.1 with `--allow-dotted-tool-names` for one-version migration grace. Error message names the offending tool + suggests the fix. Locked.
- [ ] **Smoke-test template**: rich version with register/dispatch/policy patterns. Authors delete what doesn't apply. Locked.
- [ ] **Schema placeholder**: scaffold `{"type":"object","required":[],"properties":{}}` (valid empty JSON-Schema object). Locked.

---

## Solution Design

### Components

1. **`docs/docs/guides/motoko-extension-development.md`** (~250 LOC):
   - Path-dep workflow end-to-end (Arni's 4-step loop)
   - Smoke-test patterns + `on_describe_tools` schema cookbook
   - Provider-safe tool naming + why dots break Bedrock + Vertex AI
   - Publish checklist
   - Wired into `docs/sidebars.js` under "Deploy & Embed"

2. **`cmd/ailang/init_motoko_extension.go` enhancements** (~120 LOC):
   - Generate `_smoke.ail` mirroring the shape of `scripts/smoke_context_mode_dispatch.ail`
   - Each `--tools NAME` arg gets an `on_describe_tools` block with `{"type":"object","required":[],"properties":{}}` + inline comment
   - Generated `README.md` includes the path-dep dev workflow + publish checklist

3. **Registry-validator naming gate** (~50 LOC in `cmd/registry-validator/main.go`):
   - Reject publish if any advertised tool name (from `provided_tools` or `on_describe_tools`) contains `.` or non-`[A-Za-z0-9_]`
   - Error: `"tool name 'ctx.execute' contains '.' — Anthropic Bedrock and Vertex AI reject names with dots. Use 'ctx_execute' or 'CtxExecute' instead."`
   - `--allow-dotted-tool-names` flag provides one-cycle migration grace; removed in v0.21.0

4. **`scripts/sync-extension-packages.sh` deprecation** (in `motoko_agent` repo, separate PR):
   - Add `DEPRECATION_WARNING` echo at the top
   - README points at the new docs guide
   - Schedule removal for motoko_agent v0.16.x

5. **`motoko_ext_context_mode 0.2.2`** (in `ailang-packages` repo, ~200 LOC delta — port from Arni's PR #22):
   - Real `provided_tools`: `CtxExecute`, `ctx_execute`, `CtxDoctor`, `CtxStats`, `ctx_index`
   - Real `on_tool_handle` routing through the context-mode MCP bridge
   - `on_describe_tools` with concrete JSON schemas
   - Policy: deny direct `BashExec` calls when context-mode is loaded
   - Smoke test in package's `_smoke.ail`

### Implementation Plan

**Phase 1 — Docs guide + scaffold enhancements** (~3h)
**Phase 2 — Registry-validator gate + flag** (~2h, test against all 13 published packages)
**Phase 3 — Port context_mode PR #22 → publish 0.2.2** (~2h)
**Phase 4 — Reply to #23 + motoko_agent deprecation PR** (~1h)

### Files

**ailang repo (this PR)**:
- `docs/docs/guides/motoko-extension-development.md` (NEW, ~250)
- `docs/sidebars.js` (+1)
- `cmd/ailang/init_motoko_extension.go` (+120)
- `cmd/registry-validator/main.go` (+50)
- `cmd/ailang/main.go` (+10)
- `changelogs/v0.10-current.md` (+40 under [v0.20.1])

**ailang-packages repo**:
- `packages/motoko-ext-context-mode/{register,context_mode,exec,prompts,compress,_smoke}.ail` + `ailang.toml` (~200 LOC delta)

**motoko_agent repo (follow-up PR)**:
- `scripts/sync-extension-packages.sh` (+5 deprecation warning)
- `README.md` (+15 linking to new docs guide)

---

## Conflict Surface

**Touches `cmd/ailang/init_motoko_extension.go` + `cmd/registry-validator/main.go` + `cmd/ailang/main.go`** — none in the regression-surface trigger list. Conditional category does NOT fire. Pure DX/tooling.

**Programs that MUST still work** (regression fixtures):
1. Existing `ailang init motoko-extension` output still passes `ailang check --package .`
2. All 13 currently-published `sunholo/motoko_ext_*` packages re-publish without the new naming gate hitting them (verify pre-launch)
3. `motoko_agent`'s existing extension stack still builds + `make verify_extensions` green

**Deliberately changes**:
- Scaffold output gains `_smoke.ail` + populated `on_describe_tools` + README (forward-only; existing scaffolds untouched)
- Registry validator hard-fails dotted tool names (after one-version grace)

---

## Success Criteria

- [ ] `motoko-extension-development.md` guide ships + wired into sidebar
- [ ] `ailang init motoko-extension` produces 5-file scaffold (was 3)
- [ ] `_smoke.ail` template + schema placeholders + README
- [ ] Registry validator rejects dotted tool names by default
- [ ] `--allow-dotted-tool-names` flag exists
- [ ] All 13 currently-published packages re-publishable (no false positives)
- [ ] `motoko_ext_context_mode@0.2.2` published with full implementation
- [ ] Reply posted to discussion #23
- [ ] motoko_agent deprecation PR opened
- [ ] CHANGELOG entry under [v0.20.1]
- [ ] `make test` + lint clean

---

## Timeline

**Total: ~8 hours / 1 focused session**

| Phase | Hours |
|---|---|
| Docs guide + scaffold | 3h |
| Registry-validator gate | 2h |
| context_mode 0.2.2 port + publish | 2h |
| Reply + motoko_agent deprecation | 1h |

---

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Naming gate falsely rejects a published package | Pre-launch: run gate against all 13 published packages; refine if false-positives |
| Smoke template too prescriptive | Template is a starting point; comments mark sections as optional |
| context_mode 0.2.2 port introduces a bug | Cascade test: motoko_agent's `verify_extensions` catches at boot |
| Authors miss the deprecation warning | Warning is loud (red), points at docs, script still functions for one minor version |

---

## Related Documents

**Source**:
- [arniwesth/motoko_agent#23](https://github.com/arniwesth/motoko_agent/discussions/23)
- [arniwesth/motoko_agent#22](https://github.com/arniwesth/motoko_agent/pull/22) — the context_mode fix we port

**Builds on**:
- M-EXT-SCAFFOLD-AI-FIRST (v0.18.5) — scaffold foundation
- M-EXT-PORTABILITY-GATE (v0.18.11) — `ailang check --package .` + registry-validator harness
- M-WASM-AI-STEP-BYO-KEY (v0.19.1) — migration-flag pattern (`--allow-unsafe-field-access` is the template for `--allow-dotted-tool-names`)

**Companion v0.20.1**:
- M-TYPECHECK-NO-AUTO-UNWRAP-RESULT M1b — parallel typechecker work for the same minor release

---

**Document created**: 2026-05-17
**Last updated**: 2026-05-17
