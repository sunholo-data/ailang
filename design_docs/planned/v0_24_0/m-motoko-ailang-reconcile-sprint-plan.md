# Sprint Plan: M-MOTOKO-AILANG-RECONCILE

**Design doc**: [m-motoko-ailang-reconcile.md](m-motoko-ailang-reconcile.md)
**Target**: v0.24.x (operational follow-up; no AILANG core changes)
**Status**: Approved by user (2026-05-26) following the diagnostic session that identified the exact gap
**Estimated**: 2-3 hours (mostly code work in motoko_agent; mechanical bumps in ailang-packages)
**Risk level**: Medium-Low (real code change in load-bearing `callStreamResult`, but the new API has a near-equivalent shape; rollback is `git revert`)

## Scope decisions (locked in tonight)

| Decision | Choice |
|---|---|
| What repos this sprint touches | motoko_agent (~/dev/arniwesth/motoko_agent) + ailang-packages (~/dev/sunholo-data/ailang-packages). **This AILANG repo is read-only.** |
| Whether to take Path A (AILANG shim) or Path B (motoko migration) from design doc | **Path B** — migrate motoko + ailang-packages to current `std/ai` API. AILANG itself stays untouched. |
| What we can complete from this Studio | A: ai_compat migration in both repos. B: motoko_agent version bumps. C: ailang-packages config bumps + git push. |
| What's deferred | `ailang publish` for the new package versions (no registry API key on this Studio). End-to-end `make eval-smoke` verify (depends on publish). |
| Whether to keep `AIStreamResult` return shape stable | YES — preserves the two consumers (`agent_loop_v2.ail`, `tool_dispatch_adapter.ail`) unchanged. Adapter pattern over `stepWithStream`. |

## Recent velocity (data-driven)

- **2026-05-25 sprint M-COORD-MULTI-HOST-WORKERS**: ~2200 LOC + 75 tests in ~3 actual hours (vs 22h estimated, 7x faster than padded estimate)
- **Pattern**: mechanical version bumps + small surgical code changes go fast; real algorithmic work takes longer
- **This sprint**: ~80 LOC of real code (ai_compat migration) + ~30 LOC of config bumps. Estimating ~2h actual.

## Sprint Goal

Get motoko_agent building cleanly on AILANG v0.21+ by closing the migration gap that the ailang-packages republishes left open. After this sprint, the Studio's `agent:motoko` worker tag (M-COORD-MULTI-HOST-WORKERS) graduates from aspirational to verified-working.

**Concrete end-state win**:
```bash
$ cd ~/dev/arniwesth/motoko_agent && make build
✓ verify_extensions: N booted, 0 failed
$ cd ~/dev/sunholo-data/ailang && make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"
[fizzbuzz] PASS via motoko-glm-5
```
(M5 is gated on registry publish — performed by the user, not in this sprint.)

## The dependency chain we're fixing

```
motoko_agent/ailang.toml
   │
   ├─ motoko_ext_compaction_ai@0.1.5    →  bump to 0.2.0   ← M2
   ├─ motoko_ext_compose@0.2.1          →  bump to 0.2.3   ← M3 publishes 0.2.3
   │       │
   │       └─ depends on motoko_ext_ai_compat (path dep)   ← M3 keeps this
   │             │
   │             └─ imports std/ai/streaming  →  std/ai     ← M1 migrates
   │
   └─ (11 other extensions — bump to current registry versions)  ← M2
```

## Milestones

### M1: Migrate ai_compat callStreamResult to std/ai.stepWithStream (~2h, ~80 LOC)

**What** (in `~/dev/arniwesth/motoko_agent/src/core/ai_compat.ail` AND `~/dev/sunholo-data/ailang-packages/packages/motoko-ext-ai-compat/ai_compat.ail`):
- Replace `import std/ai/streaming (callStream)` with `import std/ai (stepWithStream, StreamChunk, Message, ContentDelta, Usage)`
- Rewrite the inner call: `callStream(prompt_json, output_path, status_path, options)` → adapter over `stepWithStream(model, [Message{role,content}], [], [], on_chunk)` where `on_chunk` appends to a local `AIStreamChunk[]` and tracks usage
- **Keep `callStreamResult`'s public signature + `AIStreamResult` return shape exactly stable** — this is the contract its consumers depend on
- Update or remove `messages_json_for(input)` (no longer needed if we pass typed `[Message]` directly)

**Acceptance criteria:**
- [ ] `~/dev/arniwesth/motoko_agent/src/core/ai_compat.ail` updated; `ailang check src/core/ai_compat.ail` passes
- [ ] `~/dev/sunholo-data/ailang-packages/packages/motoko-ext-ai-compat/ai_compat.ail` updated (mirrors motoko_agent's version, sans module-path line)
- [ ] `ailang check src/core/agent_loop_v2.ail` passes (consumer 1)
- [ ] `ailang check src/core/tool_dispatch_adapter.ail` passes (consumer 2)
- [ ] `scripts/smoke_callstream.ail` still type-checks (smoke test of the public API)
- [ ] Output JSON shape (`ok`, `output`, `chunks`, `provider`, `status_code`, etc.) unchanged from before

**Files:**
- MODIFY: `~/dev/arniwesth/motoko_agent/src/core/ai_compat.ail` (~50 LOC changed)
- MODIFY: `~/dev/sunholo-data/ailang-packages/packages/motoko-ext-ai-compat/ai_compat.ail` (~50 LOC, mirror)

**Dependencies**: None.

**Risk**: Real code change in load-bearing function. Streaming protocol differences between old `callStream` and new `stepWithStream` may require care around when chunks arrive vs final Usage record.

### M2: Bump motoko_agent ailang.toml + run lock (~15 min, ~30 LOC)

**What** (in `~/dev/arniwesth/motoko_agent/ailang.toml`):
- Bump these 11 packages to their newer registry versions (already published, only motoko_agent's pins are stale):
  - `motoko_ext_abi` 2.1.0 → 2.2.0
  - `motoko_ext_test_dummy` 0.2.1 → 0.2.2
  - `motoko_ext_omnigraph` 0.2.2 → 0.2.3
  - `motoko_ext_context_mode` 0.2.1 → 0.2.2
  - `motoko_ext_mcp` 0.2.5 → 0.2.7
  - `motoko_ext_exa_search` 0.2.5 → 0.2.7
  - `motoko_ext_compose` 0.2.1 → 0.2.3 (the new version M3 will publish; this only works after M3 lands in the registry)
  - `motoko_ext_a2a` 0.2.1 → 0.2.2
  - `motoko_ext_decision_framework` 0.2.1 → 0.2.2
  - `motoko_ext_microrag` 0.4.1 → 0.4.2
  - `motoko_ext_compaction_ai` 0.1.5 → 0.2.0
  - `motoko_ext_ailang_docs` 0.1.2 → 0.1.4
  - `motoko_ext_ai_compat` 0.1.0 → 0.2.0 (the new version M3 will publish)
- Also bump the `[extensions] packages = [...]` list to the same versions
- Run `ailang lock` to regenerate `ailang.lock`

**Acceptance criteria:**
- [ ] `ailang.toml` updated with all 12 version bumps
- [ ] `[extensions] packages` list updated to match
- [ ] `ailang lock` runs cleanly and regenerates `ailang.lock`
- [ ] `make build` SHOULD pass after M3 lands in registry (gated on publish — verification deferred to user)

**Files:**
- MODIFY: `~/dev/arniwesth/motoko_agent/ailang.toml` (~25 LOC bumped values)
- MODIFY: `~/dev/arniwesth/motoko_agent/ailang.lock` (regenerated, ~auto)

**Dependencies**: M1 (the new ai_compat code) and M3 (publish gate before lock can resolve to 0.2.0).

**Note**: The bumps for `motoko_ext_compose@0.2.3` and `motoko_ext_ai_compat@0.2.0` will FAIL the lock step UNTIL the user publishes M3's work. Either:
- (a) Stage M2's changes locally without `ailang lock`, and call out that the user runs `ailang lock` after their publish; or
- (b) Use path-dep overrides temporarily to bypass the registry for these two. Default: (a) — simpler, fewer moving parts.

### M3: ailang-packages source + version bumps (~30 min, ~30 LOC)

**What** (in `~/dev/sunholo-data/ailang-packages/packages/`):
- `motoko-ext-ai-compat/ai_compat.ail`: already updated by M1
- `motoko-ext-ai-compat/ailang.toml`: bump `version = "0.1.0"` → `version = "0.2.0"` (Note: AILANG version compatibility may need updating from `>=0.15.1` to `>=0.21.0`)
- `motoko-ext-compose/ailang.toml`: bump `version = "0.2.2"` → `version = "0.2.3"` (rev because its transitive ai_compat dep got an API change)
- `motoko-ext-compose/CHANGELOG.md` (or equivalent): note the ai_compat 0.2.0 dep pin
- Commit + push to ailang-packages git (do NOT run `ailang publish` — defer to user)

**Acceptance criteria:**
- [ ] `motoko-ext-ai-compat/ailang.toml` version = 0.2.0; ailang version constraint updated
- [ ] `motoko-ext-compose/ailang.toml` version = 0.2.3
- [ ] `ailang publish --dry-run` on motoko-ext-ai-compat shows the new version + new source
- [ ] `ailang publish --dry-run` on motoko-ext-compose shows the new version
- [ ] Committed + pushed to ailang-packages remote (default branch)
- [ ] User-action notes for publish included in final sprint summary

**Files:**
- MODIFY: `~/dev/sunholo-data/ailang-packages/packages/motoko-ext-ai-compat/ailang.toml` (~3 LOC)
- MODIFY: `~/dev/sunholo-data/ailang-packages/packages/motoko-ext-compose/ailang.toml` (~1 LOC version bump)
- ALREADY-MODIFIED: `~/dev/sunholo-data/ailang-packages/packages/motoko-ext-ai-compat/ai_compat.ail` (by M1)

**Dependencies**: M1 (ai_compat source migration).

**Risk**: Two packages need fresh registry publishes after this sprint completes. If the user pushes and CI auto-publishes, no extra step needed. Otherwise, manual `ailang publish` per package.

### M4 (DEFERRED — user-side acceptance, not executed in this sprint)

**What** (user runs after `ailang publish` completes for M3's new versions):
- In ailang-packages: `ailang publish` for motoko-ext-ai-compat 0.2.0 and motoko-ext-compose 0.2.3
- In motoko_agent: `ailang lock` finishes successfully (resolves to the now-published versions)
- In motoko_agent: `make build` succeeds with `verify_extensions: N booted, 0 failed`
- In AILANG: `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` PASS
- Updates to motoko_agent's executor README (`internal/executor/motoko/README.md` in THIS repo) noting the verified-on date + AILANG version floor
- M-COORD-MULTI-HOST-WORKERS docs updated to drop the "aspirational" caveat on `agent:motoko`

**This milestone is documented here for completeness, but the sprint-executor SHOULD NOT attempt it**.

## Sequencing

```
M1 (ai_compat migration in both repos) ─┬──► M2 (motoko_agent ailang.toml bumps)
                                        │      [git push to fork, NO ailang lock yet]
                                        └──► M3 (ailang-packages version bumps + push)
                                                   [user publishes → unblocks M2's lock + M4]
```

**M1 must complete first** (both repos updated in parallel). **M2 and M3 are independent** of each other but both depend on M1.

## Total estimates

| Milestone | LOC est | Hours | Has tests | Status |
|---|---:|---:|---|---|
| M1 | 100 (50 × 2 mirrors) | 2 | `ailang check` on 3+ consumer files | sprint-executor |
| M2 | 25 (config) + 1 regen | 0.25 | `ailang lock` smoke | sprint-executor |
| M3 | 30 (configs) | 0.5 | `ailang publish --dry-run` smoke | sprint-executor |
| M4 | — (deferred) | — (user) | `make eval-smoke` PASS | **user** |
| **Sprint total** | **~155** | **~2.75** | | |

## Risk management

| Risk | Likelihood | Mitigation |
|---|---|---|
| `stepWithStream`'s streaming protocol semantics differ in ways `callStreamResult`'s output shape can't fully express | Med | Keep an on_chunk accumulator that collects ContentDelta + Usage; the final `AIStreamResult.chunks` shape stays a flat `[AIStreamChunk]` list as before |
| Consumers `agent_loop_v2.ail` / `tool_dispatch_adapter.ail` rely on undocumented behavior of the old `callStream` | Med | After M1, `ailang check` each consumer; if type-check passes, we're aligned. Runtime behavior is the user's M4 problem. |
| The ailang-packages CI auto-publishes on push, accidentally shipping pre-review | Low | Check ailang-packages workflows before pushing; if auto-publish exists, request review-only PR. |
| `motoko_ext_compose@0.2.3` would require republishing OTHER extensions too if they transitively use ai_compat APIs | Low-Med | Audit compose's source for `pkg/sunholo/motoko_ext_ai_compat/*` imports during M3; if no direct API usage, version bump is purely a constraint refresh |
| `ailang lock` in M2 fails because registry hasn't received M3's publish yet | High (by design) | Document this in M2's notes — user must run `ailang lock` again after their publish. Sprint-executor stops at lock-fail and surfaces a clear "next step" to user. |

## Success criteria for the whole sprint

- [ ] All M1 + M2 + M3 acceptance criteria met
- [ ] motoko_agent: ai_compat.ail + ailang.toml committed + pushed to `sunholo-voight-kampff/motoko_agent` (your fork)
- [ ] ailang-packages: ai_compat + compose source/configs committed + pushed (registry publish deferred to user)
- [ ] Sprint summary message lists explicit next-step commands for the user to run for M4
- [ ] M-MOTOKO-AILANG-RECONCILE design doc updated with what was actually done

## Out of scope (explicitly NOT in this sprint)

- Running `ailang publish` (no API key on this Studio)
- Running `make eval-smoke` end-to-end against the rebuilt motoko (gated on publish)
- Updating motoko_agent's main branch (we work on a feature branch; user reviews + merges)
- Other motoko_ext_* packages that may or may not need similar migration audits (compaction_ai 0.2.0 already migrated; this sprint trusts that)
- Anything in this AILANG repo proper (no Go code changes, no stdlib changes — verified by tonight's diagnostic)

## Coordinator marker output

After this sprint completes, the final summary should make these next steps explicit for the user:

```
NEXT STEPS (user-only, requires registry API key):
1. cd ~/dev/sunholo-data/ailang-packages && ailang publish [for motoko-ext-ai-compat]
2. cd ~/dev/sunholo-data/ailang-packages && ailang publish [for motoko-ext-compose]
3. cd ~/dev/arniwesth/motoko_agent && ailang lock && make build  # should now succeed
4. cd ~/dev/sunholo-data/ailang && make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"
```

---

**Sprint plan created**: 2026-05-26
**Created by**: sprint-planner skill (claude-opus-4-7)
**Approved by**: user
