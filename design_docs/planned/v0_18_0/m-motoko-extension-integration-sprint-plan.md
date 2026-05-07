# M-MOTOKO-EXTENSION-INTEGRATION Sprint Plan

**Sprint ID**: M-MOTOKO-EXTENSION-INTEGRATION  
**Design doc**: github.com/sunholo-data/motoko_explore/design_docs/M-MOTOKO-EXTENSION-INTEGRATION.md  
**Duration**: 2–2.5 weeks (12 working days)  
**Risk level**: Medium — spans three repos, has one external AILANG prerequisite (M-SERVE-API-LIVE-TOOL-REGISTRY)  
**Target**: motoko_agent v0.6.0  

---

## Sprint Goal

Replace motoko_agent's vendored `src/core/ext/` directory with independently-published AILANG packages in `sunholo-data/ailang-packages`, wired via `ailang generate-extension-registry`. Add MCP-HTTP transport support and a new A2A delegation tier.

---

## Current State (as of 2026-05-07)

**Already done — do not re-implement:**
- ✅ AILANG v0.17.1 `generate-extension-registry` command shipped
- ✅ `tools_with_extensions(rt)` implemented in `tool_catalog.ail` (local branch `motoko-dx-compaction-pending`, commit `a9b53e7`)
- ✅ `on_describe_tools` hook added to ExtensionHooks
- ✅ DP7 verifier gate made opt-in via `VerificationConfig` (commit `a78e4db`)
- ✅ `ailang-packages` repo exists at `github.com/sunholo-data/ailang-packages` with utility packages

**Vendored extensions still in `motoko_agent/src/core/ext/` (to be migrated):**
- `test_dummy/` — minimal proof-of-pattern (0 hooks, echo only)
- `context_mode/` — context type management
- `exa_search/` — web search via Exa API
- `omnigraph/` — graph reasoning tools
- `compose/` — orchestrator; largest extension (~800 LOC)
- `mcp/` — MCP bridge; needs dual transport (stdio + mcp-http)

**Not yet created:**
- `motoko-ext-abi` package (ABI contract: `ExtensionHooks`, `ToolPolicyDecision`, etc.)
- `motoko-ext-a2a` package (A2A delegation bridge)

---

## Milestones

### M1 — Publish `motoko-ext-abi@1.0.0` (0.5 day)

**Repo**: `sunholo-data/ailang-packages`

Extract `src/core/ext/types.ail` from motoko_agent and publish as the first `motoko-ext-*` package.

**Tasks:**
1. Create `packages/motoko-ext-abi/` directory structure
2. Copy `types.ail` content → `packages/motoko-ext-abi/types.ail`, update module declaration
3. Write `ailang.toml` for the package (version `1.0.0`, no deps except `std`)
4. Write README explaining the ABI contract and versioning policy
5. Publish to registry: `ailang publish`
6. Verify `ailang search motoko-ext-abi` returns the package

**Acceptance criteria:**
- `ailang install motoko-ext-abi@1.0.0` works from a fresh directory
- `motoko-ext-abi/types.ail` exports all types: `ExtensionHooks`, `ExtCtx`, `BudgetPlan`, `ToolPolicyDecision`, `ToolHandleDecision`, `ResponseInterceptDecision`, `FinalizeDecision`, `VerificationConfig`
- Package has stable `1.0.0` tag

**Estimated LOC**: 150 (package structure + types + README + ailang.toml)

---

### M2 — Proof-of-pattern: migrate `test_dummy` + wire `generate-extension-registry` (0.5 day)

**Repos**: `ailang-packages` (new package) + `motoko_agent` (ailang.toml change)

Migrate the simplest extension first to validate the full pipeline.

**Tasks:**
1. Create `packages/motoko-ext-test-dummy/` in ailang-packages
2. Write `register.ail` exporting `register_with_config(cfg: RuntimeConfig) -> ExtensionHooks ! {}`
3. Add `[extensions]` block to `motoko_agent/ailang.toml`:
   ```toml
   [extensions]
   packages      = ["motoko-ext-test-dummy@0.1.0"]
   config_import = "src/core/config.RuntimeConfig"
   hooks_import  = "motoko-ext-abi.ExtensionHooks"
   output        = "src/core/ext/registry_generated.ail"
   ```
4. Run `ailang lock && ailang generate-extension-registry`
5. Verify `make check_core` passes with the generated dispatcher
6. Keep hand-written `registry.ail` ONLY for remaining non-migrated extensions; generated file is additive for now

**Acceptance criteria:**
- `registry_generated.ail` contains `import motoko-ext-test-dummy/register (register_with_config) as register_test_dummy`
- `ailang check src/core/ext/registry_generated.ail` passes
- `make check_core` still green

**Estimated LOC**: 100 (package + register.ail + ailang.toml changes)

---

### M3 — Migrate 4 simple extensions to packages (2 days)

**Repos**: `ailang-packages` (4 new packages) + `motoko_agent` (ailang.toml, lockfile, regenerate)

Migrate the four simpler extensions. Order: `context_mode` → `exa_search` → `omnigraph` → `test_dummy` (test_dummy already done in M2).

For each extension:
1. Create `packages/motoko-ext-<name>/` with `ailang.toml` + `register.ail` + implementation `.ail` files
2. Declare `motoko-ext-abi@1.0.0` as dependency in each package's `ailang.toml`
3. Add to `[extensions].packages` in motoko_agent `ailang.toml`
4. Re-lock, regenerate, verify check_core

**Extension-specific notes:**
- `context_mode`: self-contained, only `{Env}` effects
- `exa_search`: needs `EXA_API_KEY` env var handling — keep env access in the implementation, not the ABI
- `omnigraph`: has `provided_tools` — verify `tools_with_extensions` test passes after migration
- All three: delete from `src/core/ext/` after migration + tests green

**Acceptance criteria:**
- All 4 packages published and installable
- motoko_agent `[extensions]` lists all 4 (+ test_dummy from M2)
- `registry_generated.ail` replaces relevant sections of hand-written `registry.ail`
- `make test_integration` (or equivalent) passes
- `tools_with_extensions(rt)` integration test passes for omnigraph tools

**Estimated LOC**: 600 (150 per extension × 4)

---

### M4 — Migrate `compose` extension (largest, 1.5 days)

**Repos**: `ailang-packages` + `motoko_agent`

`compose` is the orchestrator extension (~800 LOC in source). Its internal structure likely imports from other in-tree motoko modules.

**Tasks:**
1. Audit `compose/compose.ail` imports — identify any that reference `src/core/` modules that need to stay in motoko (these become package dependencies or function parameters)
2. Extract to `packages/motoko-ext-compose/` with proper dependency graph
3. Any `src/core/` imports that can't be packages → pass as configuration via `RuntimeConfig` (the standard pattern for host-side data)
4. Write `register.ail`, update ailang.toml [extensions], re-lock, regenerate
5. Delete `src/core/ext/compose/` after tests green

**Acceptance criteria:**
- `ailang check packages/motoko-ext-compose` passes standalone (no `../` imports)
- `make check_core` + `make test` green
- hand-written `registry.ail` can be DELETED (all extensions now in generated file)

**Estimated LOC**: 300 (package + refactor to remove host-side deps)

---

### M5 — Migrate `mcp` extension with dual transport (1.5 days)

**Repos**: `ailang-packages` + `motoko_agent`  
**Prerequisite**: AILANG M-SERVE-API-LIVE-TOOL-REGISTRY (for the mcp-http transport validation)  
**Can proceed without prerequisite**: stdio transport works now; mcp-http can be stubbed with a TODO comment

**Tasks:**
1. Extract `mcp/mcp.ail` → `packages/motoko-ext-mcp/`
2. **Dual transport in the dispatcher** — extend the profile JSON `mcp.json` schema with a `transport` field:
   ```json
   { "transport": "stdio",    "command": "...", ... }  // existing
   { "transport": "mcp-http", "url": "...", ...     }  // new
   ```
3. Implement `http_post_jsonrpc(url, method, args)` for mcp-http transport (AILANG `{Net}` effect)
4. Keep existing stdio handler unchanged (backwards-compatible)
5. If M-SERVE-API-LIVE-TOOL-REGISTRY is not done: add `transport = "mcp-http"` stub that returns `Err("mcp-http transport not yet supported")` clearly
6. Add to ailang.toml [extensions], lock, regenerate

**Acceptance criteria:**
- Existing `transport: "stdio"` configs continue working without change
- `transport: "mcp-http"` with a local `ailang serve-api --mcp-http` instance routes correctly (or fails with a clear NOT-YET error if AILANG prerequisite not done)
- `ailang check packages/motoko-ext-mcp` passes standalone

**Estimated LOC**: 350 (dual transport + profile schema + tests)

---

### M6 — `motoko-ext-a2a` v0.1 (blocking delegation) (2 days)

**Repo**: `ailang-packages` (new package)  
**No motoko_agent dependency required** — a2a is a new extension, not a migration

**Tasks:**
1. Create `packages/motoko-ext-a2a/` from scratch
2. `register.ail`: register an `on_tool_handle` hook that intercepts tool calls named `delegate_to_<agent>` and routes them to an A2A endpoint
3. Profile config `a2a.json` schema: `[{ name, url, discover_at, timeout_ms }]`
4. Implement `a2a_send(url, task)` + `a2a_poll_until_complete(url, task_id, timeout_ms)` using AILANG `{Net}` effect
5. Discovery: `GET /.well-known/agent.json` at startup to validate endpoint + read capabilities
6. Register the tool name(s) in `provided_tools` so `tools_with_extensions` surfaces them to the LLM
7. Smoke test: `examples/smoke_a2a_delegate.ail` that mocks an A2A endpoint

**Acceptance criteria:**
- `ailang check packages/motoko-ext-a2a` passes
- Given a local `ailang serve-api --a2a` endpoint, `delegate_to_<agent>` tool call returns the endpoint's response
- Tool appears in `tools_with_extensions(rt)` output
- `examples/smoke_a2a_delegate.ail` passes `ailang test`

**Estimated LOC**: 400 (client + profile parsing + dispatch + smoke test)

---

### M7 — Validation + delete vendored directory (1 day)

**Repos**: `motoko_agent` + `ailang-packages`

**Tasks:**
1. **Round-trip community extension test**: write a minimal new `motoko-ext-hello@0.1.0` in a fresh directory (not ailang-packages), publish it, add to motoko_agent's `ailang.toml`, lock, generate, verify check_core. Proves the community workflow.
2. **Hot-reload smoke**: `ailang serve-api --watch packages/motoko-ext-exa-search/ --mcp` — edit a function, verify the MCP tool list reflects the change without restart (requires AILANG M-SERVE-API-LIVE-TOOL-REGISTRY)
3. **A2A federation smoke**: start a hello-world `ailang serve-api --a2a` endpoint, motoko delegates a task, result returns
4. **Delete `src/core/ext/`**: remove the entire vendored directory. Only `registry_generated.ail` remains.
5. Update motoko documentation: `agents.md` with three-tier architecture explanation

**Acceptance criteria:**
- `src/core/ext/` contains ONLY `registry_generated.ail` (no vendored AILANG source)
- All smoke tests pass
- `make check_core && make test` green on final state
- Community extension test passes end-to-end

**Estimated LOC**: 200 (smoke scripts + docs)

---

## LOC Summary

| Milestone | LOC | Days |
|-----------|-----|------|
| M1 — motoko-ext-abi@1.0.0 | 150 | 0.5 |
| M2 — test_dummy proof-of-pattern | 100 | 0.5 |
| M3 — 4 simple extensions | 600 | 2.0 |
| M4 — compose (largest) | 300 | 1.5 |
| M5 — mcp dual transport | 350 | 1.5 |
| M6 — motoko-ext-a2a v0.1 | 400 | 2.0 |
| M7 — validation + cleanup | 200 | 1.0 |
| **Total** | **2,100** | **9.0** |

With 30% buffer: **~12 days (2.5 weeks)**

---

## Dependency Graph

```
M1 (abi)
  └── M2 (test_dummy + wire generate-extension-registry)
        └── M3 (4 simple extensions)
              ├── M4 (compose)
              │     └── M7 (validation)
              └── M5 (mcp dual transport) ← also needs AILANG M-SERVE-API-LIVE-TOOL-REGISTRY
                    └── M7
M6 (a2a) ─────────────────────────────────── M7
```

M6 can run in parallel with M3–M5 (no shared files).

---

## External Prerequisites

| Prerequisite | Status | Blocks |
|---|---|---|
| AILANG v0.17.1 `generate-extension-registry` | ✅ Shipped | M2+ |
| `tools_with_extensions(rt)` in tool_catalog.ail | ✅ Done (branch: motoko-dx-compaction-pending) | M3 validation |
| DP7 opt-in VerificationConfig | ✅ Done | — |
| **AILANG M-SERVE-API-LIVE-TOOL-REGISTRY** (`--tool-dir`) | ❌ Not started | M5 full validation, M7 hot-reload smoke |

M-SERVE-API-LIVE-TOOL-REGISTRY does NOT block M1–M4 or M6. M5 can be started with a `transport: "mcp-http"` stub; full validation requires the AILANG work.

---

## Parallel work (not in this sprint)

- motoko `--resume <session-id>` (msg 67ff70cc) — handles session continuity when tier 1 rebuild IS needed; parallel, independent
- M-AGENT-LOOP-ARCHITECTURE Option B (`runTools` hooks) — deferred to v0.19.0+

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `compose/compose.ail` has deep `src/core/` deps that can't be packageised | Medium | High (M4 scope doubles) | Audit imports in M4 step 1 before committing to migration |
| AILANG M-SERVE-API-LIVE-TOOL-REGISTRY slips | Medium | Low (M5 stub unblocks progress; hot-reload validation slips to M7.2) | Build the stub in M5; M7 hot-reload smoke becomes conditional |
| Registry publish tooling (ailang publish) flakiness | Low | Medium | Can use path deps during development, switch to registry for final validation |
| `motoko-ext-abi` v1.0 ABI needs a breaking change mid-sprint | Low | High (all published packages need updates) | Do one careful ABI review in M1; publish only after all extensions have been sketched |
