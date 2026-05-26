# M-MOTOKO-V021-EFFECT-ROW-MIGRATION: Finish motoko_agent's AILANG v0.21+ migration

**Status**: IMPLEMENTED
**Target**: v0.24.x (operational follow-up; no AILANG core changes)
**Priority**: P1 (Medium — blocks `agent:motoko` graduation from aspirational to verified; no production users impacted today, but the loop with M-COORD-MULTI-HOST-WORKERS won't close until this lands)
**Estimated**: 2-4 hours (depends on whether the fix is a type annotation or a deeper refactor in `dispatch_step`)
**Dependencies**: [M-MOTOKO-AILANG-RECONCILE](m-motoko-ailang-reconcile.md) (parent migration that fixed ai_compat) — already implemented and registry-published 2026-05-26.

## Problem Statement

M-MOTOKO-AILANG-RECONCILE landed the ai_compat migration + the 12 extension version bumps + ailang-packages republishes. Registry now has `motoko_ext_ai_compat@0.2.0` and `motoko_ext_compose@0.2.3`. Build progress on motoko_agent:

| `make build` step | Result after M-MOTOKO-AILANG-RECONCILE |
|---|---|
| `verify_extensions` | ✓ All extensions boot cleanly (compaction_ai, context_mode, exa_search) |
| `check_core` | ✗ **3 of 23 src/core files fail** to type-check |

The 3 remaining failures share one root error:

```
Error: type error in src/core/agent_loop_v2 (decl 31):
type unification failed at [function application at src/core/agent_loop_v2.ail:908:35]:
failed to unify parameter 4: failed to unify effect rows:
incompatible closed rows: r1 has extra labels [], r2 has extra labels [IO]
```

Cascade:
- `src/core/agent_loop_v2.ail:908` — primary site, calls `dispatch_step(provider, model, compacted_msgs, rt, on_chunk)`
- `src/core/rpc.ail` — fails because it imports `agent_loop_v2`
- `src/core/supervisor.ail` — fails for the same transitive reason

The relevant code in `agent_loop_v2.ail`:
```ailang
let on_chunk = \chunk. emit_stream_chunk(session_id, step_idx, stream_id, chunk);
let dispatched = dispatch_step(provider, model, compacted_msgs, rt, on_chunk);
                                                                     ^^^^^^^^
                                                                     // line 908 parameter 4
```

And `dispatch_step`'s declared signature in `src/core/test/stub_step.ail`:
```ailang
export func dispatch_step(
  provider: StepProvider,
  model: string,
  msgs: [Message],
  rt: ExtRuntime,
  on_chunk: (StreamChunk) -> () ! {IO}    -- ← declares IO effect
) -> { result: Result[StepResult, AIError], next_provider: StepProvider } ! {AI, IO} {
  match provider {
    LiveAI => {
      { result: stepWithStream(model, msgs, tools_with_extensions(rt), ..., on_chunk), ... }
    },
    Scripted(script) => {
      match script {
        [] => { result: Ok(terminal_step()), next_provider: Scripted([]) },
        s :: rest => { result: Ok(scripted_to_step_result(s)), next_provider: Scripted(rest) }
      }
    }
  }
}
```

The signature says `on_chunk` parameter has effect `{IO}`. The caller passes a closure whose body calls `emit_stream_chunk` (effect `{IO}`). Both sides should match.

**Working hypothesis** (to be confirmed during sprint execution): AILANG v0.21+ tightened closed-row vs open-row effect inference. In `dispatch_step`'s body, the `Scripted(script)` branch never CALLS `on_chunk` — that branch only returns terminal_step() / scripted_to_step_result(). The type-checker may be inferring on_chunk's actual usage in the body as effect `{}` (unused in one branch → no effect required) and conflicting with the declared `{IO}`. The asymmetry between LiveAI (calls on_chunk via stepWithStream) and Scripted (ignores on_chunk) is the suspicious shape.

**Current State:**
- `motoko_ext_ai_compat@0.2.0` + `motoko_ext_compose@0.2.3` published ✓
- Registry resolution works ✓
- `src/core/ai_compat.ail` uses `std/ai.stepWithStream` correctly ✓
- 20/23 src/core files type-check ✓
- 3/23 fail on this effect-row mismatch in agent_loop_v2 ✗
- Studio's `agent:motoko` worker tag remains aspirational until this lands ✗

**Impact:**
- No production traffic broken (the tag is advertised but no one routes to it today)
- Blocks: closing the `agent:motoko` loop from M-COORD-MULTI-HOST-WORKERS, which would let any client send `requires: agent:motoko` and have it land on the Studio's rig
- Blocks: motoko-driven evals (`make eval-smoke MODELS=motoko-glm-5`) against the current AILANG release

## Goals

**Primary Goal:** `make build` exits 0 on motoko_agent against AILANG v0.21+, and `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` from this AILANG repo PASS via the motoko adapter.

**Success Metrics:**

- `cd ~/dev/arniwesth/motoko_agent && make build` exits 0 (currently exits 2 on `check_core`)
- All 23 src/core/*.ail files type-check (currently 20/23)
- `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` passes — proves the motoko CLI adapter works end-to-end with the rebuilt agent
- Studio's `ailang coordinator workers list` shows `studio.eval-rig` advertising `agent:motoko` with a verified-working footnote (no longer aspirational)
- M-COORD-MULTI-HOST-WORKERS docs updated: drop the aspirational caveat on `agent:motoko`

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Where the fix lives: `dispatch_step` signature, agent_loop_v2 call site, or both | Determines blast radius (test_dummy adapter has same shape; might cascade) | human | design | med |
| Whether to refactor `dispatch_step` so all branches use on_chunk consistently | Symmetric branches are easier for the type-checker; but motoko's Scripted test path INTENTIONALLY ignores the callback | human | design | med |
| Whether to land the fix as a separate PR or fold into the still-open M-MOTOKO-AILANG-RECONCILE PR | Separate is cleaner for review; folding minimizes user/maintainer PR-review overhead | human | design | low |
| Whether to surface this to upstream AILANG as a type-system limitation worth tracking | If the effect-row issue is an AILANG bug/quirk, upstream should know; if it's motoko's pattern, no | agent | design | low |

### Design Freeze

Before implementation begins:

- [ ] **Confirm the root cause** during sprint Step 0 (smallest repro that triggers the error). Determines whether the fix is type annotation, refactor, or upstream AILANG concern.
- [ ] **PR strategy**: recommend a NEW branch `feature/v021-effect-row-migration` from current main (NOT folded into the open `feature/ailang-v0.21-migration` PR, which is ai_compat-scoped).
- [ ] **Whether to ALSO add a fixture test** (e.g., `src/core/test/agent_loop_v2_compile_test.ail`) that pins this build-pass property so future regressions are caught.

## Solution Design

### Overview

Most likely outcomes, in order of decreasing likelihood:

1. **Annotation fix (~30 min)**: Add an explicit type annotation on `on_chunk` at the call site:
   ```ailang
   let on_chunk: (StreamChunk) -> () ! {IO} =
     \chunk. emit_stream_chunk(session_id, step_idx, stream_id, chunk);
   ```
   This tells the type-checker the explicit effect row before it tries to infer from usage. If AILANG was inferring `{}` from how `dispatch_step`'s Scripted branch ignores the callback, the explicit annotation halts that inference.

2. **dispatch_step refactor (~1-2h)**: Restructure `dispatch_step` so both branches reference `on_chunk` in a type-pinning way. Possibly:
   - Add a `_ = on_chunk` reference in the Scripted branch (no-op call); OR
   - Split `dispatch_step` into a LiveAI variant (takes on_chunk) and a Scripted variant (doesn't), with a thin wrapper.

3. **Both (defensive)**: Apply the annotation AND tighten dispatch_step. Belt-and-braces, slightly more LOC.

4. **Upstream AILANG fix (out of scope here)**: If the root cause is an AILANG type-system bug (open vs closed row inference for effect rows in branched function bodies), file in this AILANG repo. The motoko-side workaround still lands.

### Architecture

```
M-MOTOKO-AILANG-RECONCILE (parent, done):
  ├ motoko_ext_ai_compat 0.2.0   ✓ published
  ├ motoko_ext_compose 0.2.3      ✓ published
  ├ ai_compat.ail migrated        ✓
  ├ 12 ext version bumps          ✓
  └ ext/runtime.ail jnum fix      ✓ (1-line bonus during M4)

M-MOTOKO-V021-EFFECT-ROW-MIGRATION (this sprint):
  ├ Step 0: minimal repro          ← confirm hypothesis
  ├ M1: annotation OR refactor     ← the fix
  ├ M2: full make build PASS       ← acceptance test
  ├ M3: e2e eval-smoke PASS        ← end-to-end test
  └ M4: drop "aspirational" caveat ← docs update
```

### Implementation Plan

**Step 0 — confirm root cause (~30 min):**
- [ ] Try smallest annotation fix first (just adding `: (StreamChunk) -> () ! {IO}` at the call site).
- [ ] If that compiles → root cause is inference. M1 = annotation. Done.
- [ ] If it doesn't compile → dig into `dispatch_step` body. Try adding a no-op `_ = on_chunk` to the Scripted branch.
- [ ] If neither works → minimal isolated `.ail` repro showing the closed-row issue, suitable for filing in AILANG.

**M1 — apply the fix (~1h):**
- [ ] Whatever Step 0 revealed: annotation, refactor, or both
- [ ] `ailang check src/core/agent_loop_v2.ail` passes
- [ ] `ailang check src/core/rpc.ail` passes (transitive consumer)
- [ ] `ailang check src/core/supervisor.ail` passes (transitive consumer)

**M2 — make build green (~30 min):**
- [ ] `make build` exits 0 from clean state
- [ ] All 23 src/core/*.ail files in check_core green
- [ ] All 11 enabled extensions boot in verify_extensions

**M3 — end-to-end smoke (~30 min):**
- [ ] `cd ~/dev/sunholo-data/ailang && make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` — PASS
- [ ] Studio's coordinator can now CLAIM a `requires: agent:motoko` message and execute it via the motoko CLI adapter

**M4 — graduate the worker tag (~30 min):**
- [ ] Update `docs/docs/guides/coordinator-workers.md`: drop the "aspirational" caveat on `agent:motoko`
- [ ] Update `internal/executor/motoko/README.md` (in this AILANG repo): pin the supported motoko revision (commit SHA from the merged motoko_agent PR) + AILANG version floor (v0.21.0+)
- [ ] Update `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` if it has any reference to motoko status
- [ ] CHANGELOG entry under v0.24.x

### Files to Modify/Create

**In `motoko_agent` (new feature branch from main; ~30-100 LOC depending on fix shape):**
- MODIFY: `src/core/agent_loop_v2.ail` (one of):
  - Annotation: 1-line edit at line 907
  - Or refactor: ~20 LOC restructuring how `on_chunk` is passed
- POSSIBLY: `src/core/test/stub_step.ail` if `dispatch_step` needs restructuring
- POSSIBLY: NEW `src/core/test/agent_loop_v2_compile_test.ail` (~20 LOC, regression fixture)

**In this AILANG repo (docs only, ~30 LOC):**
- MODIFY: `docs/docs/guides/coordinator-workers.md` — drop aspirational caveat
- MODIFY: `internal/executor/motoko/README.md` — pin supported motoko SHA
- MODIFY: `changelogs/v0.10-current.md` — close out the M-MOTOKO loop

## Examples

### Example 1: The likely fix (annotation)

```diff
-    let on_chunk = \chunk. emit_stream_chunk(session_id, step_idx, stream_id, chunk);
+    let on_chunk: (StreamChunk) -> () ! {IO} =
+      \chunk. emit_stream_chunk(session_id, step_idx, stream_id, chunk);
     let dispatched = dispatch_step(provider, model, compacted_msgs, rt, on_chunk);
```

Two extra lines. Type-checker now uses the explicit annotation instead of inferring from how dispatch_step's body uses (or doesn't use) the callback.

### Example 2: End-state — Studio actually serves an agent:motoko message

After this sprint lands:

```bash
# From anywhere (laptop, GitHub Action, cloud):
curl -X POST <coordinator-host>/api/messages -d '{
  "inbox": "eval-rig",
  "title": "smoke fizzbuzz on motoko via the Studio",
  "requires": ["agent:motoko", "ollama:gemma4-26b-ailang"]
}'

# Studio claims (only host with both tags), routes to motoko CLI adapter,
# motoko drives the rig's local Ollama, result posts back. Real end-to-end.
```

## Success Criteria

- [ ] `cd ~/dev/arniwesth/motoko_agent && make build` exits 0
- [ ] All 23 src/core/*.ail files type-check cleanly
- [ ] `cd ~/dev/sunholo-data/ailang && make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` PASS
- [ ] PR opened from a new `feature/v021-effect-row-migration` branch into `arniwesth/motoko_agent` main
- [ ] `docs/docs/guides/coordinator-workers.md` updated (drop aspirational caveat on `agent:motoko`)
- [ ] `internal/executor/motoko/README.md` updated (motoko commit pin + AILANG version floor)
- [ ] CHANGELOG entry in this AILANG repo
- [ ] If a regression fixture was added: `agent_loop_v2_compile_test.ail` (or similar) passes

## Testing Strategy

**Unit tests (motoko-side):**
- Per-file `ailang check` on agent_loop_v2.ail, rpc.ail, supervisor.ail

**Integration tests:**
- `make build` exits 0 — covers both `verify_extensions` and `check_core`
- `make eval-smoke MODELS=motoko-glm-5 EXTRA="-trials 1 -benchmarks fizzbuzz"` PASS — covers the full motoko CLI adapter loop including the executor in this AILANG repo

**Manual testing:**
- After the PR lands + the motoko_agent change is on main: install motoko binary, run a single eval-smoke trial, confirm PASS

## Deferred Decisions

- **Whether to file the underlying issue with AILANG upstream** — agent investigates during Step 0 and decides. If the issue is a closed-vs-open row inference bug in AILANG itself, file. If it's a motoko code shape that just needs an annotation, skip.
- **Whether to add a CI smoke** that cross-builds motoko_agent against AILANG's `dev` branch — out of scope for tonight; tracked as a separate operational item.

## Non-Goals

- **Modernising motoko's harness beyond AILANG-v0.21 compatibility** — this sprint is strict reconciliation, not feature work.
- **Fixing other motoko v0.21 migration issues that aren't currently failing the build** — if `make build` exits 0, we're done. Further migration debt can wait.
- **Touching this AILANG repo's stdlib or compiler** — purely operational/docs-side changes in AILANG here. Code work is all in motoko_agent.

## Timeline

**Best case (annotation fix):** ~2 hours total
- 30 min: Step 0 + M1
- 30 min: M2 (make build green)
- 30 min: M3 (eval-smoke)
- 30 min: M4 (docs + CHANGELOG)

**Worst case (deeper refactor):** ~4 hours
- 1 hour: Step 0 + understanding `dispatch_step`'s branch structure
- 1.5 hours: M1 (refactor + verify)
- 30 min: M2-M4

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| The "obvious" annotation fix doesn't work and the root cause is an actual AILANG type-system bug | Med | Step 0 has a fallback path — if neither annotation nor refactor unblocks, file in AILANG repo and use a path-dep override or AILANG patch to unblock motoko-side work |
| Fixing agent_loop_v2 exposes ANOTHER type error elsewhere (more v0.21 debt) | Med | Already saw this pattern in M-MOTOKO-AILANG-RECONCILE (ext/runtime.ail jnum). Plan iteratively — each new error is at most a few lines to fix. If we hit 4+ cascading errors, pause and reassess scope. |
| Eval-smoke fails post-build due to runtime issues (motoko CLI args changed, etc.) | Low-Med | Mitigation: known-good test against `make run TASK="hello"` first before eval-suite integration. Document any runtime fixes needed. |
| The motoko PR ends up touching files not declared in this design doc | Low | Step 0's minimal-repro investigation locks the change set before M1 implementation. If scope grows beyond agent_loop_v2 + maybe stub_step, pause and revise this doc. |

## Related Documents

**Direct parent (now implemented):**
- [m-motoko-ailang-reconcile.md](m-motoko-ailang-reconcile.md) (planned) / `design_docs/implemented/v0_24_0/m-motoko-ailang-reconcile.md` — fixed ai_compat. This sprint finishes what that sprint left as scope discovery.

**Operational ecosystem:**
- [m-coord-multi-host-workers.md](../implemented/v0_24_0/m-coord-multi-host-workers.md) — introduced the `agent:motoko` worker tag whose graduation depends on this sprint
- [m-motoko-executor-adapter.md](v0_18_0/m-motoko-executor-adapter.md) — original motoko CLI executor in this AILANG repo

**Cross-repo state:**
- motoko_agent fork: `https://github.com/sunholo-voight-kampff/motoko_agent` (the user's fork — PR origin)
- ailang-packages: `https://github.com/sunholo-data/ailang-packages` (already updated by M-MOTOKO-AILANG-RECONCILE)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Build pass/fail is the same input→output; no AILANG-level determinism change |
| A2: Replayability | 0 | No new effects; no semantic change |
| A3: Effect Legibility | +1 | If the fix is an explicit annotation, makes the effect row of `on_chunk` legible at the call site (currently inferred and confusing the type-checker) |
| A4: Explicit Authority | 0 | No new ambient access |
| A5: Bounded Verification | +1 | Brings agent_loop_v2 + rpc + supervisor back into the type-checked set (currently failing). Restores 3 files of bounded verification we lost. |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Build-clean is a precondition for machines (agents, eval rig, coordinator) to use motoko; making the build pass is structurally machine-affirming |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | n/a |
| A10: Composability | +1 | Restores composability of motoko + AILANG benchmarks (motoko was a documented executor variant in models.yml that doesn't currently work) |
| A11: Structured Failure | 0 | n/a (failure mode unchanged) |
| A12: System Boundary | 0 | No change to system boundary |

**Net Score: +4** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No new nondeterminism
- [x] A3 (Effects): If anything, makes effects more explicit
- [x] A4 (Authority): No new ambient access
- [x] A7 (Machines First): Build-pass restoration

### Conflict Surface

**Not applicable** — this sprint touches `motoko_agent` and (optionally) docs in this AILANG repo. No changes to `internal/parser/`, `internal/types/`, `internal/elaborate/`, `internal/codegen/`, or any other AILANG-language-semantic surface. The migration debt is on motoko's side of the AILANG boundary.

## References

- [Design Axioms](/docs/references/axioms)
- M-MOTOKO-AILANG-RECONCILE evaluation report: `.ailang/state/evaluations/eval_M-MOTOKO-AILANG-RECONCILE_round_1.json` — documents the M4 discovery that led to this sprint
- The error trace: motoko_agent commit `8834a47` on `feature/ailang-v0.21-migration` is the state where the build fails on agent_loop_v2.ail:908

## Future Work

- **CI smoke**: cross-build motoko_agent against AILANG's `dev` branch on every AILANG release — catches v0.X drift before it accumulates into a multi-day reconciliation sprint
- **Effect-row inference hardening**: if Step 0 surfaces an AILANG type-system limitation, file in the AILANG repo as a separate milestone
- **Bundled motoko + AILANG release artifact**: longer-term, ship a "known good" tagged combination so users don't manage two version flows

---

**Document created**: 2026-05-26
**Last updated**: 2026-05-26
