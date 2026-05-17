# Sprint Plan: M-EXT-AUTHOR-DX (v0.20.1)

**Design doc**: [m-ext-author-dx.md](./m-ext-author-dx.md) (axiom score +6, committed `e906529a`)
**Target**: v0.20.1
**Estimated**: ~8h / 1 focused session, 4 milestones, ~670 LOC
**Risk level**: **Medium** — M1 ports cross-repo code; M3 has cascade-impact risk (false positives on currently-published packages). M2 + M4 are mechanical.
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-17

---

## Discovery (pre-planning)

Verified what's already in place vs needs creating:

| Component | State |
|---|---|
| Path-dep + `generate-extension-registry` end-to-end | ✅ Verified working (this session) — Q1 answered |
| `ailang check --package .` | ✅ Shipped (M-EXT-PORTABILITY-GATE, v0.18.11) — Q6 answered |
| `cmd/ailang/init_motoko_extension.go` | EXISTS (M-EXT-SCAFFOLD-AI-FIRST v0.18.5). Generates register.ail with 8 ExtensionHooks fields as no-op defaults. **Missing**: `_smoke.ail` template, `on_describe_tools` schema placeholders, README with publish checklist. |
| `cmd/registry-validator/main.go` | EXISTS (M-EXT-PORTABILITY-GATE). Already does smoke runs at publish time. **Missing**: tool-name validation gate. |
| `cmd/ailang/main.go` | EXISTS. Has flag-plumbing patterns from `--allow-unsafe-field-access` (M-WASM-AI-STEP-BYO-KEY v0.19.1) — mirror for `--allow-dotted-tool-names`. |
| `ailang-packages/packages/motoko-ext-context-mode/` | Currently at 0.2.1, effectively a stub (no real `provided_tools`, no `on_tool_handle`, no schemas). Arni's PR #22 `src/core/ext/context_mode/` has the full implementation. |
| `docs/docs/guides/` + `docs/sidebars.js` | EXISTS, has "Deploy & Embed" section. New guide slots in alongside `wasm-integration.md` and `wasm-ai-step-byo-key.md`. |

**Velocity calibration**: this sprint is ~670 LOC + cross-repo (ailang + ailang-packages). Compares to:
- M-WASM-AI-STEP-BYO-KEY (v0.19.1): ~835 LOC + cross-repo (publish), 1 session × 2 evaluator rounds = ~5h
- M-TYPECHECK-NO-AUTO-UNWRAP-RESULT M1 partial (v0.20.0): 1 session × 5h for typechecker work that turned out architecturally surprising

Realistic estimate: **1 session / ~8h** for code+test+publish+docs. The architectural risk surface here is much smaller than M-TYPECHECK because every milestone touches well-trodden code paths (scaffold, validator, docs).

---

## Milestones

### M1 — Port context_mode PR #22 → publish 0.2.2 (~200 LOC, ~2h)

**Goal**: Replace the stub `sunholo/motoko_ext_context_mode@0.2.1` with the full implementation Arni built locally in PR #22. Bump to 0.2.2 and publish via the cascade. **Highest-leverage delivery** — lets Arni drop his local override entirely.

**Tasks**:

1. **Read Arni's PR #22 implementation** at `arniwesth/motoko_agent` branch `fix_context_mode_extension`, files under `src/core/ext/context_mode/`. Capture: tool handlers, policy logic, schema definitions.

2. **Port to `ailang-packages/packages/motoko-ext-context-mode/`**:
   - `register.ail` — populate `provided_tools: [CtxExecute, ctx_execute, CtxDoctor, CtxStats, ctx_index]`, real `on_tool_handle` routing through the context-mode MCP bridge, `on_describe_tools` with concrete JSON schemas (required args per tool: `language`, `code`, `content`, `path`, `commands`, `queries`)
   - `context_mode.ail` — core logic
   - `exec.ail` — execution helpers
   - `prompts.ail` — prompt templates if any
   - `compress.ail` — compression helpers if any
   - Policy: deny direct `BashExec` calls when context-mode is loaded + the command targets context-mode

3. **Add `_smoke.ail`** mirroring Arni's `scripts/smoke_context_mode_dispatch.ail` shape:
   - Assert `register_with_config` returns without panic
   - Assert `CtxDoctor` / `CtxStats` dispatch path works
   - Assert `BashExec(context-mode ...)` denied (no AI handler in publish sandbox — uses stub)
   - Assert `on_describe_tools` returns non-empty schemas

4. **Bump `ailang.toml`**: `version 0.2.1 → 0.2.2`, dependencies as needed

5. **Publish**: `ailang publish --dry-run` then `ailang publish` (registry credentials required)

6. **Smoke-test locally** against motoko_agent (path-dep override → confirm `verify_extensions` boots cleanly)

**Acceptance**:
- [ ] `ailang-packages/packages/motoko-ext-context-mode/` updated with full implementation (no longer stub)
- [ ] `_smoke.ail` exercises register + dispatch + denial path
- [ ] `ailang check --package .` clean
- [ ] `ailang publish` succeeds, registry shows 0.2.2 as latest
- [ ] motoko_agent path-dep against this package boots cleanly via `verify_extensions`
- [ ] CHANGELOG entry in ailang-packages records the port + Arni's PR attribution

**Risk**: Medium. Cross-repo port; potential gotchas with how MCP bridge is invoked from package-context vs motoko_agent local-context.

---

### M2 — Scaffold enhancements: smoke + schemas + README (~120 LOC, ~2h)

**Goal**: `ailang init motoko-extension` produces a 5-file scaffold (was 3) that includes everything an author needs for a publishable extension on first run.

**Depends on**: None (independent of M1/M3)

**Tasks**:

1. **`cmd/ailang/init_motoko_extension.go::scaffoldMotokoExtension`** — add three file generators:
   - `_smoke.ail` template (~40 LOC of generated content) modeled on `motoko_ext_compaction_ai 0.1.5` + Arni's `smoke_context_mode_dispatch.ail`: register/dispatch/policy assertion patterns; sections marked `-- optional: drop if not applicable`
   - `README.md` template (~60 LOC) including: path-dep dev workflow link, publish checklist (bump version → `ailang publish --dry-run` → `ailang publish` → consumer-side switch back to version pin), provider-safe tool naming reminder

2. **Per-tool `on_describe_tools` placeholder**: each `--tools NAME` arg produces a block in register.ail:
   ```ailang
   { name: "NAME",
     -- TODO: fill in required + properties per Anthropic typed tool use
     -- See: https://docs.anthropic.com/en/docs/tools
     schema: """{"type":"object","required":[],"properties":{}}""" }
   ```
   (vs current bare `{}` which silently degrades against typed tool use)

3. **Update scaffold tests** in `cmd/ailang/init_motoko_extension_test.go` (or add one if missing) — verify the 5 files are generated and pass `ailang check --package .`

**Acceptance**:
- [ ] `ailang init motoko-extension --name X --tools "A,B"` generates: `ailang.toml`, `register.ail`, `types.ail`, `_smoke.ail`, `README.md`
- [ ] `_smoke.ail` covers register + dispatch + policy patterns with `-- optional` markers
- [ ] Each `--tools NAME` produces an `on_describe_tools` block with `{"type":"object","required":[],"properties":{}}` placeholder + comment
- [ ] `README.md` includes path-dep workflow link + publish checklist + naming guidance
- [ ] Generated scaffold passes `ailang check --package .` with zero edits
- [ ] Existing scaffold test still passes; new test exercises the 5-file structure

**Risk**: Low. Mechanical scaffold work in a file we already own.

---

### M3 — Registry-validator naming gate + migration flag (~60 LOC, ~2h)

**Goal**: `ailang publish` rejects packages whose advertised tool names contain `.` or non-`[A-Za-z0-9_]` characters (rejected by Anthropic Bedrock + Vertex AI). One-cycle migration grace via `--allow-dotted-tool-names`.

**Depends on**: None (independent of M1/M2)

**Tasks**:

1. **`cmd/registry-validator/main.go`** — add `validateToolNames()` (~30 LOC):
   - Read manifest's `provided_tools` + extract tool names from `on_describe_tools` output
   - For each name, check `len([^A-Za-z0-9_]) == 0`
   - On violation: structured error `{kind: "tool_name_invalid", tool: "ctx.execute", reason: "contains '.'", suggestion: "use 'ctx_execute' or 'CtxExecute'"}`
   - Wire into the existing publish-time check pipeline

2. **`cmd/ailang/main.go`** — add `--allow-dotted-tool-names` flag plumbing (~10 LOC) mirroring `--allow-unsafe-field-access` from M-WASM-AI-STEP-BYO-KEY:
   - Flag plumbed through publish command
   - Sent to registry-validator via env var or HTTP header
   - Validator downgrades to WARN-level diagnostic when set

3. **Pre-launch verification** (~20 LOC of script): run the new gate against all 13 currently-published `sunholo/motoko_ext_*` packages — confirm zero false positives. If any trip, refine the rule.

**Acceptance**:
- [ ] `cmd/registry-validator/main.go` has `validateToolNames()` rejecting `.` / non-`[A-Za-z0-9_]` names
- [ ] Error message names offending tool + suggests fix
- [ ] `--allow-dotted-tool-names` flag exists and downgrades to warning
- [ ] All 13 currently-published `sunholo/motoko_ext_*` packages pass the gate (no false positives — captured as a one-liner test script)
- [ ] `make test` covers the validator gate

**Risk**: Medium. Cascade risk if a currently-published package trips it on re-publish — pre-launch script mitigates.

---

### M4 — Docs guide + CHANGELOG + design-doc move (~290 LOC, ~2h)

**Goal**: Single canonical docs page teaches the full extension-author loop. CHANGELOG entry honest about what shipped. Design doc + sprint plan moved to `implemented/v0_20_1/`.

**Depends on**: M1 + M2 + M3 (documents what they shipped)

**Tasks**:

1. **`docs/docs/guides/motoko-extension-development.md`** (~250 LOC):
   - Why extensions exist + when to write one
   - 4-step canonical loop (Arni's proposal verbatim): implement in `ailang-packages` → path-dep from consumer → smoke → publish → version-pin
   - Smoke-test patterns (template from M2 scaffold + canonical examples)
   - `on_describe_tools` cookbook (per-tool schema examples for the common patterns: command-execution, search, fetch, structured-data)
   - Provider-safe tool naming + why dots break Bedrock + Vertex AI (link to the migration-flag M3 ships)
   - Publish checklist
   - Cross-link to `motoko_ext_test_dummy` as the canonical minimal example
   - Cross-link to discussion #23 as the historical motivation

2. **`docs/sidebars.js`** — wire into "Deploy & Embed" alongside `wasm-integration` and `wasm-ai-step-byo-key`

3. **`changelogs/v0.10-current.md`** under `[v0.20.1]` — entry covering:
   - context_mode 0.2.2 published (M1)
   - Scaffold enhancements (M2)
   - Registry naming gate + migration flag (M3)
   - Docs guide (M4)
   - Cross-link to design doc + sprint plan + discussion #23

4. **Move design doc + sprint plan** from `design_docs/planned/v0_20_1/` to `design_docs/implemented/v0_20_1/`. Update status field on both files: Planned → Implemented.

**Acceptance**:
- [ ] `motoko-extension-development.md` ships at ~250 LOC, covers all 5 sections
- [ ] Sidebar wires the new guide
- [ ] CHANGELOG entry under `[v0.20.1]` cross-links design doc + discussion + 4 milestone deliverables
- [ ] Both design doc and sprint plan moved to `implemented/v0_20_1/`, status fields updated
- [ ] `make test` + `make lint` clean

**Risk**: Low. Documentation + admin work.

---

## Day-by-day breakdown

Single focused session, sequential execution:

| Block | Milestone | Hours |
|---|---|---|
| 1 | M1: context_mode 0.2.2 port + publish | 2h |
| 2 | M2: scaffold enhancements | 2h |
| 3 | M3: registry naming gate + migration flag | 2h |
| 4 | M4: docs guide + CHANGELOG + design-doc move | 2h |

**Total: ~8 hours.**

---

## Repo coordination

| Repo | Branch | What lands |
|---|---|---|
| `sunholo-data/ailang` | `dev` | M2 + M3 + M4 (scaffold, validator, docs, CHANGELOG, design-doc move) |
| `sunholo-data/ailang-packages` | `feat/portability-v0.18.11` (current) | M1 (context_mode 0.2.2 port + publish via cascade) |
| `arniwesth/motoko_agent` | follow-up after M1 publishes | Bump pin to 0.2.2 (Arni can do this himself or we PR) |

---

## Success metrics

- **Arni can drop his PR #22 local override**: after M1 publishes 0.2.2, his `src/core/ext/context_mode/` becomes deletable + his `ailang.toml` pin to `motoko_ext_context_mode@0.2.2` provides the same functionality
- **Future scaffolds are publishable on first run**: M2's enhanced output passes `ailang check --package .` + `ailang publish --dry-run` with no manual additions for the smoke and schema bits
- **Bedrock-breaking names rejected at publish time**: M3's gate would have caught the `ctx.execute`-style names we fixed manually at `5e2eaef`
- **Single source of truth for extension authoring**: M4's docs guide replaces "scattered knowledge across design docs + PR comments + Slack" with one URL

---

## Risks

| Risk | Mitigation |
|---|---|
| M1: cross-repo port introduces subtle behavioral difference | Smoke-test locally against motoko_agent before publishing 0.2.2 (`verify_extensions` boot path) |
| M1: MCP bridge invocation differs between package-context vs motoko_agent-context | Read Arni's `register.ail` carefully; ask in PR #22 if anything ambiguous |
| M3: false positives on currently-published packages | Pre-launch script runs gate against all 13 published packages before merging M3 |
| M3: re-publishing existing packages now fails | `--allow-dotted-tool-names` flag provides one-cycle escape |
| M4: docs guide becomes stale fast | Cross-link to discussion #23 and design doc so updates land in those canonical places first |

---

## Files modified

| File | Repo | Change | LOC |
|---|---|---|---|
| `packages/motoko-ext-context-mode/*.ail` + `_smoke.ail` + `ailang.toml` | ailang-packages | M1 port (replace stub with full impl) | +200 |
| `cmd/ailang/init_motoko_extension.go` | ailang | M2 smoke + schema + README templates | +120 |
| `cmd/ailang/init_motoko_extension_test.go` | ailang | M2 test for 5-file structure | +40 |
| `cmd/registry-validator/main.go` | ailang | M3 naming gate | +30 |
| `cmd/ailang/main.go` | ailang | M3 `--allow-dotted-tool-names` flag | +10 |
| `scripts/check_published_tool_names.sh` (NEW) | ailang | M3 pre-launch verification | +20 |
| `docs/docs/guides/motoko-extension-development.md` (NEW) | ailang | M4 canonical guide | +250 |
| `docs/sidebars.js` | ailang | M4 sidebar wiring | +1 |
| `changelogs/v0.10-current.md` | ailang | M4 entry under [v0.20.1] | +40 |
| `design_docs/planned/v0_20_1/m-ext-author-dx.md` → `implemented/v0_20_1/` | ailang | M4 move + status update | 0 (rename) |
| `design_docs/planned/v0_20_1/m-ext-author-dx-sprint-plan.md` → `implemented/v0_20_1/` | ailang | M4 move + status update | 0 (rename) |
| **Total** | | | **~710** |

---

## Notes for the executor

- **Sequential execution recommended** — milestones touch overlapping concerns (M4 documents M1+M2+M3 outputs; context-switching cost is high for cross-repo work)
- **M1 first** for highest user value: unblocks Arni completely
- **M3 pre-launch script is non-negotiable** — verify against all 13 published packages BEFORE merging the gate
- **M4 design-doc move happens LAST** after all CHANGELOG/docs updates are committed
- **Cross-repo commits**: M1 lands in ailang-packages; M2/M3/M4 land in ailang. Separate PRs for clarity.
- **No CHANGELOG entry until M1+M2+M3 complete** — write the entry knowing what actually shipped

---

**Document created**: 2026-05-17
**Last updated**: 2026-05-17
