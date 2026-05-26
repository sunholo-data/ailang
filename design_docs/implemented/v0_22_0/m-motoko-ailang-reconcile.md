# M-MOTOKO-AILANG-RECONCILE: Restore working motoko_agent build on current AILANG

**Status**: Implemented (v0.22.0; registry publish pending user — 2026-05-26)
**Target**: v0.22.0 (small) or v0.25.0 (full migration)
**Priority**: P2 (Medium-Low — `agent:motoko` worker tag advertised but unfulfillable until this lands)
**Estimated**: 2-4 hours (best case: just find + merge an existing PR; worst case: write a migration shim)
**Dependencies**: None blocking; complements [M-COORD-MULTI-HOST-WORKERS](../implemented/v0_22_0/m-coord-multi-host-workers.md) which advertises the tag.

## Problem Statement

The Mac Studio rig advertises `agent:motoko` as a worker tag (per M-COORD-MULTI-HOST-WORKERS, v0.22.0), but `make build` on the user's motoko_agent clone fails against the current AILANG v0.21.0:

```
✗ compaction_ai       Error: failed to load std/ai/streaming: stdlib module not found
✗ context_mode         Error: failed to load std/ai/streaming: stdlib module not found
✗ exa_search           Error: failed to load std/ai/streaming: stdlib module not found
verify_extensions: 0 booted, 3 failed
```

Diagnosis from the 2026-05-25 evening session:

- motoko's `src/core/ai_compat.ail` imports `std/ai/streaming (callStream)` — this was the upstream API at AILANG v0.15.1.
- Current AILANG v0.21.0 exposes `std/ai` (no `/streaming` suffix) with exports `openaiCompatStream` / `anthropicStream` — NOT `callStream`.
- The motoko clone on disk (`~/dev/arniwesth/motoko_agent`) is on branch `docs/acp-integration-proposal`; the most recent ai_compat.ail commit is `a594347 fix(migrate): emit thinking_stream_* events from ai_compat shim`. No subsequent migration to v0.16+ AILANG API on this branch.

**2026-05-26 investigation update**: a second look identified the exact gap. The user's understanding ("the PRs are merged") was *partially* correct — `ailang-packages` did republish 12 newer extension versions (motoko_ext_abi 2.2.0, motoko_ext_compaction_ai 0.2.0, etc.) — but the migration is incomplete on **two** sides:

```
motoko_agent/ailang.toml (still pins old versions — needs bump)
   │
   ├─ motoko_ext_compaction_ai@0.1.5 → fixed by bumping to 0.2.0
   ├─ motoko_ext_compose@0.2.1 → bumped 0.2.2 exists BUT STILL DEPENDS ON
   │       motoko_ext_ai_compat@0.1.0 ◄── BROKEN, no newer version published
   │           └─ imports std/ai/streaming (removed in v0.21.0)
   │
   └─ motoko_ext_ai_compat@0.1.0 (direct dep) — same as above
```

And `motoko_agent/src/core/ai_compat.ail` (the master copy of the ai_compat code, synced into the published package) is also unmigrated — still imports `std/ai/streaming (callStream)`. Two consumers depend on it: `agent_loop_v2.ail` and `tool_dispatch_adapter.ail`.

**The actual three-part fix is now concrete** (see Implementation Plan below).

**Current State:**
- Studio's coordinator daemon advertises `agent:motoko` (worker_tags in `~/.ailang/config.yaml`)
- A task tagged `requires: agent:motoko` would be claimed by the Studio
- The Studio's motoko-cli executor would then fail HealthCheck at task-claim time
- Result: confused sender, parked-looking task that's actually broken-handler

**Impact:**
- Low frequency today (no one routes `requires: agent:motoko` yet) but a real correctness gap
- Blocks any "compare motoko vs opencode vs claude on AILANG benchmarks" experiments the user might run
- Background noise: every time we discuss multi-host workers, the asterisk on `agent:motoko` adds explanation overhead

## Goals

**Primary Goal:** A reproducible 5-minute sequence ("checkout branch X of motoko_agent, run `make build`, get a working binary, smoke-test it routes correctly through `make eval-smoke MODELS=motoko-glm-5`") on the current AILANG v0.21.0+.

**Success Metrics:**

- `motoko_agent` repo on a named branch (likely an existing PR; otherwise a new compatibility branch) builds cleanly against AILANG v0.21.0
- `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` from this AILANG repo completes — proves the executor adapter works end-to-end with the rebuilt motoko
- `agent:motoko` worker_tag on the Studio passes HealthCheck (no longer aspirational)
- The right motoko branch documented in `internal/executor/motoko/README.md` so future contributors don't get stuck

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Which motoko branch is canonical for AILANG v0.21+ | Affects every contributor; documenting the wrong one wastes hours | human | design | high |
| Whether to update motoko's `main` to that branch or keep them separate | Drives long-term maintenance burden + clarity | human (or arniwesth) | design | med |
| Whether the AILANG side needs a compatibility shim (`std/ai/streaming → std/ai`) | Compat shim helps motoko + any downstream user on older AILANG | human | design | med |
| What version of AILANG is the supported floor for motoko going forward | Affects the executor's HealthCheck version-min check | human | design | low |

### Design Freeze

Before implementation begins:

- [ ] **Locate the working PR/branch**. User to point at it OR (if lost) we cut a fresh `feature/ailang-v0.21-compat` branch.
- [ ] **Decide compat shim direction**: AILANG re-adds `std/ai/streaming` as an alias for `std/ai` (so old motoko code works unchanged) OR motoko's ai_compat.ail is updated to the new imports.
- [ ] **Choose the supported AILANG version floor**: the motoko `ailang.toml` currently has `ailang = ">=0.16.1"`. If we update motoko's imports, this stays. If AILANG ships a compat shim, this can stay as well, and any AILANG ≥0.16.1 works.

## Solution Design

### Overview

Two paths — work the easier one first. They are mutually exclusive; pick whichever the user's existing PR (if found) already implements.

**Path A: re-add `std/ai/streaming` as a shim module in AILANG** (likely 1h):
- Create `stdlib/std/ai/streaming.ail` that re-exports `callStream` (or whatever the historical name was) as a thin wrapper over `openaiCompatStream` / `anthropicStream`
- Motoko's existing code keeps working unchanged
- Carries a deprecation note for new users to migrate

**Path B: migrate motoko to v0.21+ AILANG API** (likely 2-3h):
- Update `src/core/ai_compat.ail` imports + call sites
- Update lockfile + any extension that imports `std/ai/streaming`
- Republish affected `motoko_ext_*` packages? — needs ecosystem decision

### Architecture (depending on which path)

```
Path A — AILANG-side shim:
  stdlib/std/ai/streaming.ail (NEW)
    └── re-exports: callStream
          └── delegates to std/ai.openaiCompatStream / anthropicStream
                                              (based on provider arg)

Path B — motoko-side migration:
  motoko_agent/src/core/ai_compat.ail (MODIFY)
    └── import std/ai (openaiCompatStream, anthropicStream)
    └── callStreamResult(...) — rewrites internal calls to dispatch by provider
```

### Implementation Plan

**Step 0 (information gathering, ~30min)**:
- [ ] User points the agent at the PR/branch they remember (motoko_agent + AILANG sides if both touched)
- [ ] OR: agent does a wider grep across all branches including ones not pushed to fork yet
- [ ] Document the finding: branch name(s), what they change, what state they're in

**Step 1 — if a working PR exists** (~30min):
- [ ] Checkout the motoko_agent branch from Step 0
- [ ] Run `make build` against the current AILANG; if green, smoke `make run`
- [ ] If it works, merge to motoko's `main` (or document the branch as canonical)
- [ ] Update `internal/executor/motoko/README.md` here with the branch name + AILANG version floor

**Step 1' — if no working PR exists, Path A** (~1h):
- [ ] Add `stdlib/std/ai/streaming.ail` to AILANG (this repo)
- [ ] Re-export `callStream` with the v0.15.x signature, delegating to current `std/ai`
- [ ] Unit test: existing motoko code's import resolves
- [ ] Smoke: `make build` in motoko_agent succeeds
- [ ] Update `docs/docs/reference/stdlib/std-ai.md` (or equivalent) noting the shim

**Step 2 — verify end-to-end** (~30min):
- [ ] `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` from this repo
- [ ] On success: remove the "aspirational" caveat from M-COORD-MULTI-HOST-WORKERS docs (mention this milestone closed the gap)
- [ ] Studio's `agent:motoko` tag now backed by a real working harness

**Step 3 — pin + document** (~30min):
- [ ] Pin the motoko version (commit SHA or release tag) in `internal/executor/motoko/README.md`
- [ ] Add a CI smoke (optional, only if affordable) that catches future drift

### Files to Modify/Create

**Path A files (AILANG-side):**
- NEW `stdlib/std/ai/streaming.ail` (~30 LOC) — shim re-exporting callStream
- MODIFY `internal/executor/motoko/README.md` (~10 LOC) — branch + AILANG version floor

**Path B files (motoko-side, in motoko_agent repo, not this one):**
- MODIFY `src/core/ai_compat.ail` (~20 LOC) — import + call-site updates
- MODIFY `ailang.toml` (~1 LOC) — bump `ailang = ">=0.21.0"`
- MODIFY any of the 14 `sunholo/motoko_ext_*` packages that import `std/ai/streaming` (likely 3: compaction_ai, context_mode, exa_search) — bump + republish

## Examples

### Example 1: From "broken" to "verified" on the Studio

**Before this milestone:**
```
$ ailang coordinator workers list
HOST             TYPE        TAGS                                       ...
studio.eval-rig  bare-metal  ...,agent:motoko,...                       ...
                              ↑
                              advertised but motoko-cli HealthCheck would fail
                              if a tag-routed message arrived
```

**After:**
```
$ cd ~/dev/arniwesth/motoko_agent
$ git checkout <branch-X-from-Step-0>
$ make build
✓ verify_extensions: 14 booted, 0 failed
✓ binary at $PWD/motoko (or installed to PATH)

$ cd ~/dev/sunholo-data/ailang
$ make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"
[fizzbuzz] PASS in 12.3s via motoko-glm-5

$ ailang coordinator workers ping studio.eval-rig
studio.eval-rig: pong via system:heartbeat (12ms round-trip)
```

### Example 2: A user routes a real task to motoko

```bash
curl -X POST <coordinator>/api/messages -d '{
  "inbox": "eval-rig",
  "from": "user",
  "title": "compare motoko vs opencode on dense_operator_program",
  "requires": ["agent:motoko", "ollama:gemma4-26b-ailang"]
}'

# Coordinator routes to Studio (only host with both tags)
# Studio's motoko-cli executor claims, runs `motoko "<task>"` subprocess
# Result + session JSONL post back to Firestore + Pub/Sub completion
```

## Success Criteria

- [ ] Either: AILANG repo has `stdlib/std/ai/streaming.ail` shim AND motoko's existing code builds, OR motoko_agent has a v0.21-compatible branch documented as canonical
- [ ] `make build` in motoko_agent (on the documented branch) succeeds with no skipped or failed extensions
- [ ] `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` returns PASS
- [ ] `internal/executor/motoko/README.md` cites: branch name, AILANG version floor, build command, last-verified date
- [ ] Studio's `workers list` row for `agent:motoko` is no longer aspirational
- [ ] M-COORD-MULTI-HOST-WORKERS docs updated: remove the "aspirational" caveat on motoko
- [ ] All AILANG tests pass; lint clean

## Testing Strategy

**Unit tests (if Path A taken):**
- AILANG: smoke import test verifying `import std/ai/streaming (callStream)` resolves AND that `callStream` dispatches correctly to the underlying provider implementations

**Integration tests:**
- `make build` in motoko_agent succeeds (manual check; cannot be automated without a CI runner that has the motoko clone)
- `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` passes

**Manual testing:**
- The 5-minute reproducible sequence in Goals — confirm on a clean Studio + a fresh laptop

## Deferred Decisions

- **Republish affected motoko_ext_* packages** if Path B taken — agent may file this as a sub-task; the user's call whether to bump within this milestone or defer
- **Adding `agent:codex` to the same milestone** — separate executor, separate install (deferred per user earlier); maybe a future M-CODEX-HARNESS milestone
- **Adding a CI smoke that catches AILANG-vs-motoko-vs-extension drift** — nice to have, depends on the user's CI infrastructure for cross-repo testing

## Non-Goals

- **Modernising motoko's harness itself** (its TUI, agent loop logic, etc.) — this is purely an AILANG version compatibility milestone
- **Adding new motoko features** — strictly reconciliation
- **Resolving the broader `agent:codex` install** — separate work

## Timeline

**Best case** (existing PR found, builds out of the box on v0.21.0): ~1 hour
**Path A (AILANG shim)**: ~2 hours implementation + ~30min docs
**Path B (motoko migration)**: ~3 hours code + republish dance + verification

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The user's remembered "working PR" is on a private remote we can't access | Med | Step 0 surfaces this early; fall back to Path A (shim) which doesn't need the missing branch |
| AILANG's `std/ai` exports don't cover all callStream semantics — shim would lie | Med | Path A's tests force exact-API parity; if signatures don't line up, escalate to Path B |
| Republishing motoko_ext_* packages cascades through ailang-packages registry | Low | Path A avoids this entirely; if Path B is chosen, the user's existing release flow handles it |
| Motoko has its own internal deps that reference v0.15.x APIs beyond what we see | Med | Step 0 surveys all motoko_core + extension imports for stale `std/ai/streaming` references |

## Related Documents

**Implemented (this completes a known gap from):**
- [m-coord-multi-host-workers.md](../implemented/v0_22_0/m-coord-multi-host-workers.md) — introduced the `agent:motoko` worker tag

**Implemented (motoko foundation):**
- [m-motoko-executor-adapter.md](v0_18_0/m-motoko-executor-adapter.md) — original motoko CLI executor adapter (v0.18.0)
- [m-motoko-executor-adapter-sprint-plan.md](v0_18_0/m-motoko-executor-adapter-sprint-plan.md)

**Operational reference:**
- `internal/executor/motoko/README.md` — current motoko adapter docs (will be updated by this milestone)
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` — the rig that will benefit when motoko is back

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Compatibility work; no AILANG-level determinism change |
| A2: Replayability | +1 | Path A's shim is a deterministic re-dispatch; traces unchanged |
| A3: Effect Legibility | 0 | No change to AILANG effects |
| A4: Explicit Authority | 0 | No new ambient access |
| A5: Bounded Verification | 0 | No type-system change |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Reduces the "is motoko available?" ambiguity for agents querying workers list |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | No change to cost surface |
| A10: Composability | +1 | Restores composition of motoko-harness + AILANG benchmarks |
| A11: Structured Failure | +1 | Removes a soft-fail (HealthCheck at claim time) for tag-routed messages |
| A12: System Boundary | 0 | No change to system boundary |

**Net Score: +4** → **Decision: ✅ Proceed pending Step 0 (locate the working PR/branch)**

### Hard Violation Check

- [x] A1 (Determinism): No nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access
- [x] A7 (Machines First): Tooling clarity improved, not degraded

### Conflict Surface

**Triggered** if Path A taken — modifies AILANG stdlib by adding `std/ai/streaming` shim. Pre-emptively addressing:

1. **What syntactic/semantic positions does this extend?** Module-path lookup for `std/ai/streaming` — currently absent (load fails). After: a real module returns at that path.
2. **What OTHER valid constructs live in those positions?** No conflict — the path was unused. The only ambiguity is with `std/ai` (parent), but AILANG's module resolver doesn't confuse parents and children.
3. **How does the typechecker disambiguate?** Module paths are exact-match; no disambiguation needed.
4. **Which existing programs MUST still work post-change?** All AILANG programs that import `std/ai` (current API) — Path A is purely additive at the module-path level.
5. **What deliberately changes?** Programs that imported `std/ai/streaming` (currently broken) now resolve. Programs that imported `std/ai` are unaffected.

Path B (motoko-side migration) has zero conflict surface in this AILANG repo — all changes are in motoko_agent.

## References

- [Design Axioms](/docs/references/axioms)
- [M-COORD-MULTI-HOST-WORKERS](../implemented/v0_22_0/m-coord-multi-host-workers.md) — sets up the `agent:motoko` tag
- [M-MOTOKO-EXECUTOR-ADAPTER](v0_18_0/m-motoko-executor-adapter.md) — original adapter (v0.18.0)
- Commit `a594347` in motoko_agent (`fix(migrate): emit thinking_stream_* events from ai_compat shim`) — current ai_compat.ail state
- AILANG v0.15.x `std/ai/streaming` (M-AI-STREAMING-HELPER) — the historical API motoko depends on
- 2026-05-25 evening session transcript — discovery of the gap

## Future Work

- **`agent:codex` install + verification** — companion milestone for the other deferred harness
- **CI smoke**: cross-repo build test that catches AILANG-vs-motoko-vs-extension drift before users hit it
- **A "supported harnesses" matrix** in AILANG docs: which harnesses are verified against which AILANG release
- **Motoko's own AILANG-version migration tooling** (out of scope for AILANG, but could be a motoko-side asset)

---

**Document created**: 2026-05-25
**Last updated**: 2026-05-25
