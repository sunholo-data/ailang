# M-ARCH-BOUNDARIES-EVAL-EXCLUSION-TIGHTEN: scope the `eval` bridge exception file-level

**Status**: Planned (backlog stub, evidence-gated)
**Target**: post-v1.0 / opportunistic
**Priority**: P3 (low — no active violation today)
**Estimated**: ~0.5d
**Dependencies**: m-arch-boundaries Phases 1–3 (LANDED iter 68, `ee97fada6`)

## Problem

`scripts/check_boundaries.sh` Rule 2 (no dashboard package imports the compiler
surface directly) **excludes `internal/eval` at the PACKAGE level**. This is
because `internal/embed`'s public API forces callers to name `eval.Value`
(`embed.ToGo(v eval.Value)`; used by `internal/server/ailang_bridge.go` in
`convertHeatmapResultFromAILANG` / `convertBudgetStatusFromAILANG`). That is the
sanctioned bridge value type, not a behavioral dependency on the evaluator.

The residual risk (flagged by the sprint-evaluator, iter 68, non-blocking): the
package-level exclusion means **any** file in `internal/{server,coordinator,
observatory,messaging}` could import `eval` for **any** reason (e.g. `eval.Env`,
`eval.CallStack`) without the gate firing. Today only ONE file (`ailang_bridge.go`)
imports `eval`, solely for the `eval.Value` type — so the gap is theoretical.

## Trigger (evidence-gated — do NOT pick without this)

A **second** dashboard file importing `internal/eval`, OR any dashboard `eval.*`
use beyond the `eval.Value` bridge type. Until then this is a latent gap, not a bug.

## Proposed fix (when triggered)

Add `eval` back to Rule 2's deny-list (`CORE_SURFACE_PKGS`) and carve out the ONE
known bridge file with a per-file allowlist (e.g. skip `internal/server/ailang_bridge.go`
in the Rule-2 grep, with a comment pointing here). OR — the cleaner structural fix —
have `internal/embed` re-export its own `Value` alias so dashboard code never names
`eval` at all, then re-add `eval` to the deny-list with no carve-out. Prefer the
latter if `embed` is being touched anyway.

## Non-Goals

- Phase 4 physical restructure (separate deferred item, v1.0→v1.1).
- Any change to the `eval.Value` bridge contract itself.
