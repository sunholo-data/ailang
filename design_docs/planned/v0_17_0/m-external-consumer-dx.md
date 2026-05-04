# M-EXTERNAL-CONSUMER-DX: Diagnostics & Artifacts for External AILANG Projects

**Status**: Planned
**Target**: v0.17.0
**Priority**: P1 (High — first non-trivial external consumer surfaced concrete DX gaps within hours of integrating)
**Estimated**: 2-3 days (~14-18 hours)
**Dependencies**: None hard; benefits compose with M-AI-TOOL-LOOP (same release).
**Author**: Claude + Mark
**Created**: 2026-05-04

---

## Executive Summary

[motoko_agent](https://github.com/sunholo-data/motoko_agent) is the first production-scale external AILANG project: ~5,200 LOC in `src/core/`, 68 `.ail` modules, vendored fork with documented rebase-forward policy. Within 24 hours of public release they recorded three independent DX failures in `.agent/learnings/` that map cleanly to fixable AILANG-core gaps. This milestone bundles those fixes plus two release artifacts that unblock follow-on tooling (grammar-constrained decoding, deterministic error→hint tables) for any future external consumer.

**Scope (in priority order):**

1. **`MOD012` — module_prefix overlap diagnostic** when root and dep share `module_prefix` and contain modules with the same path, producing silent wrong-package resolution today.
2. **Effect-row mismatch pinpointing** — when inferred `! {Env}` ≠ declared `! {Env, FS}`, the error message names the call site whose effect contributes the disputed label.
3. **`error_codes.json` release artifact** — machine-readable `{code, category, fix_hint, doc_url}` rows for every emitted parser/type/effect error code.
4. *(Stretch)* **`ailang.ebnf` release artifact** — versioned EBNF grammar file shipped alongside the binary, unlocking grammar-constrained decoding (xgrammar/llguidance) and small-model fine-tuning.

Out of scope: AST-builder API (separate v0.18 milestone), fork-stability surface declaration (separate doc), `ailang verify` UX polish (separate doc).

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Diagnostics and artifacts only; no semantic change |
| A2: Replayability | +1 | `error_codes.json` makes parser failures replayable as structured data, not just human prose |
| A3: Effect Legibility | +2 | The whole point of (2) is making effect rows readable — currently the user knows the row is wrong but not which call to fix |
| A4: Explicit Authority | 0 | No authority change |
| A5: Bounded Verification | +1 | Better diagnostics shorten the type-check feedback loop |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | +2 | `error_codes.json` and `ailang.ebnf` are explicitly machine-first artifacts; this is the axiom's headline use case |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | None |
| A10: Composability | +1 | External consumers compose these artifacts into their own tooling (motoko's planned hint-table extension; future fine-tuning corpora) |
| A11: Structured Failure | +2 | (1) and (2) replace silent or vague failures with structured, localized ones |
| A12: System Boundary | +1 | Release artifacts make the AILANG↔consumer boundary explicit instead of implicit |

**Net Score: +10** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): Diagnostics are deterministic functions of the input program
- [x] A3 (Effects): No new effect-system change; (2) makes existing effect inference more legible
- [x] A4 (Authority): No ambient authority added
- [x] A11 (Failure): All three core items reduce silent/vague failure modes

---

## Motivating Evidence

### Issue 1: module_prefix overlap silently misroutes imports

From `motoko_agent/.agent/learnings/2026-05-03-motoko-core-package-sync.md`:

> Root project (`local/motoko_agent`, `module_prefix = "src"`) had `sunholo/motoko_core` (also `module_prefix = "src"`) as a dependency. Both packages claim the same source tree. When `rpc.ail` (which is *not* exported by `motoko_core`) imported `src/core/tool_contract`, the compiler crossed the package boundary and stripped the root's `module_prefix`, producing the path `pkg/sunholo/motoko_core/tool_contract` — but the correct path is `pkg/sunholo/motoko_core/core/tool_contract`. Resolution failed with no clear pointer to the underlying ambiguity.

Workaround they found: remove `motoko_core` from root deps. This is correct — but the compiler should have *said* that.

Current code path: `internal/pipeline/pipeline_module.go:565-585` (`validateModulePath`) and `internal/pipeline/package_resolver.go` already maintain a `modulePrefixMap` that supports multiple package names sharing a prefix. The gap is in error reporting on the lookup miss — when the constructed path doesn't resolve, the compiler doesn't surface the prefix-overlap diagnostic that would explain why.

### Issue 2: effect-row mismatches don't cite the contributing call

Same learning, root cause #4:

> `register()` returned `ExtensionHooks` whose `on_budget_plan` field had type `(ExtCtx, BudgetPlan) -> BudgetPatch ! {Env, FS}`. The lambda assigned to it captured `cfg` from an `Env`-only scope, so its inferred effect was `! {Env}`. AILANG correctly rejected the assignment under closed effect rows. But the diagnostic gives the user only the inferred and expected rows — not which body call introduced (or *failed to introduce*) the FS label.

Without that pointer, the user has to read the entire body and trace effects mentally. Multiplied across the codebase, this is the dominant friction point per their `ailang_csp_implementation.md` learning ("Exhaustive Effect Mapping" rule #4).

### Issue 3: error codes are human prose, not data

Their `AILANG_performance_evidence_gates.md` research doc (lever 4, "Deterministic error → corrective-hint table") proposes building a hint table by *scraping* AILANG documentation:

> Cheap to build from the AILANG docs' "Common Mistakes" + "What AILANG Does NOT Have" sections.

The hint table would be more useful, more durable, and more accurate if AILANG itself shipped a structured `error_codes.json` artifact per release. Same data, machine-first, version-aligned.

---

## Design

### Item 1: `MOD012` module_prefix overlap diagnostic

**Trigger:** during `pipeline_module.go:detectModulePathCollisions` (already in place for `MOD011`), additionally check whether any dep package has a `module_prefix` that overlaps with the root project's `module_prefix`. If so, walk the root's local module set and the dep's exported set; for any module path that exists in both, emit:

```
Error MOD012: ambiguous module ownership under shared module_prefix

  Module path:        src/core/tool_contract
  Declared by root:   /workspaces/motoko_agent/src/core/tool_contract.ail
  Declared by dep:    sunholo/motoko_core (module_prefix = "src")
                      → pkg/sunholo/motoko_core/core/tool_contract

  Both your project and the dependency `sunholo/motoko_core` use
  `module_prefix = "src"` and contain a module at the same logical path.
  Imports of `src/core/tool_contract` from this project are ambiguous.

  Fix one of:
    1. Remove `sunholo/motoko_core` from your `[dependencies]` if your
       project IS the canonical source of these modules.
    2. Change one side's `module_prefix` to disambiguate.
    3. Use the explicit `pkg/...` import form to opt into the dep:
       `import pkg/sunholo/motoko_core/core/tool_contract (...)`
```

**Detection point:** after `MOD011` runs (lines 112-130 of `pipeline_module.go`). Reuse the `modulePrefixMap` already passed via `currentModulePrefixMap` — it has all the data needed.

**Deliverable:** new `MOD012` error code; reproduction test that mirrors the motoko_agent scenario in `internal/pipeline/pipeline_module_test.go`; doc page at `docs/docs/reference/errors/mod012.md`.

### Item 2: effect-row mismatch with call-site pointer

**Current behavior** (in `internal/types/typechecker.go` and `internal/elaborate/`): when row unification fails, the error reports inferred row, expected row, and source location of the *function being assigned* — not the body subexpression that contributed the row.

**Proposed:** during effect-row inference, retain a per-label provenance map: `effect_label → first source span that introduced it`. On unification failure where the inferred row is *narrower* than expected (i.e. expected has a label inferred lacks), report:

```
Error TYP_EFFECT_ROW_MISMATCH:
  Expected effect row: ! {Env, FS}
  Inferred effect row: ! {Env}
  Missing labels:      FS

  The expected slot type requires `FS`, but the assigned function body
  does not call any `FS`-effectful operation.

  If the body should perform FS work, add a call that requires FS
  (e.g. `readFile`, `writeFile`).

  If FS is not needed, change the slot's declared row to omit FS.

  Slot: src/core/ext/test_dummy/dummy.ail:42
        on_budget_plan: (ExtCtx, BudgetPlan) -> BudgetPatch ! {Env, FS}
  Lambda: src/core/ext/test_dummy/dummy.ail:47
        \(ctx, plan) -> ...
```

And the symmetric case where inferred is *wider* than expected:

```
Error TYP_EFFECT_ROW_MISMATCH:
  Expected effect row: ! {Env}
  Inferred effect row: ! {Env, FS}
  Unexpected labels:   FS  (introduced at src/core/ext/foo.ail:84
                            via call to `readFile`)

  Either declare FS in the function's effect row, or remove the
  FS-effectful call from the body.
```

**Implementation surface:** `internal/types/effects.go` (or wherever row unification lives — the `Eff` type carries no provenance today). Add a thin `Provenance map[string]ast.SourceSpan` field on row records during inference; populate it when an effectful builtin or call adds a label. The data flows through unification but is not used for type identity (so it doesn't break existing equality checks).

**Deliverable:** updated `TYP_EFFECT_ROW_MISMATCH` formatter in `internal/types/`; provenance tracking through inference; reproduction test mirroring the motoko `register()` bug; doc update in `docs/docs/reference/errors/typ_effect_row_mismatch.md`.

### Item 3: `error_codes.json` release artifact

**What it is:** a single JSON file shipped with each release, schema:

```json
{
  "schema": "ailang.error_codes/v1",
  "ailang_version": "v0.17.0",
  "generated_at": "2026-05-04T...",
  "codes": [
    {
      "code": "MOD010",
      "category": "module",
      "summary": "module declaration does not match file path",
      "fix_hint": "set `module <relative_path>` so it mirrors the file system path from project root",
      "doc_url": "https://ailang.sunholo.com/docs/reference/errors/mod010"
    },
    {
      "code": "TYP_EFFECT_ROW_MISMATCH",
      "category": "type",
      "summary": "function body's inferred effect row differs from expected",
      "fix_hint": "add the missing label(s) to the declared row, or remove the call introducing the unexpected label",
      "doc_url": "..."
    },
    ...
  ]
}
```

**Generation:** every error code already has a Go-side string constant and a doc page (or should). Add a `make error-codes` target that walks `internal/errors/` (or wherever codes are defined) and emits the JSON. CI verifies every emitted code in `internal/` has a row in the JSON; fails the build if not.

**Distribution:** publish to `https://ailang.sunholo.com/error_codes/v<version>.json` and as a GitHub release asset.

**Consumer story (motoko):** their planned error→hint extension reads `error_codes.json`, indexes by code, and surfaces structured hints to their author loop without screen-scraping our docs. The hint table they author on top stays in motoko; we only own the canonical mapping.

**Deliverable:** `make error-codes` target; CI gate; release-publish hook; one paragraph in `docs/docs/reference/errors/index.md` documenting the artifact.

### Item 4 (Stretch): `ailang.ebnf` release artifact

A versioned EBNF grammar file checked into the repo and published as a release asset. Initially synthesized from the parser implementation (with manual review); subsequently the source of truth for grammar-constrained decoding consumers.

Defer detailed design to a follow-up if items 1-3 fill the sprint. Listed here so it's on the radar.

---

## Implementation Plan

### Day 1: Item 1 (MOD012)

- Reproduction test using the motoko scenario (two packages with `module_prefix = "src"` claiming overlapping module paths).
- Detect in `detectModulePathCollisions` after the existing MOD011 check.
- Error formatter; doc page; verify-examples gate update if needed.
- Acceptance: reproducer fails with MOD012, message names both claimants.

### Day 2: Item 2 (effect-row provenance)

- Add `Provenance` to row records during inference.
- Update `TYP_EFFECT_ROW_MISMATCH` formatter to consume provenance.
- Reproduction test mirroring `register()`/`on_budget_plan` mismatch.
- Acceptance: error message names the contributing call site for both narrower-than-expected and wider-than-expected mismatches.

### Day 3: Item 3 (error_codes.json) + release wiring

- Inventory all error codes emitted from `internal/`.
- `make error-codes` target generates JSON.
- CI gate: every emitted code has a row.
- Release workflow publishes `error_codes/v<ver>.json` to docs site + GitHub release.
- Acceptance: CI fails if a new code is added without a row; the JSON is downloadable for the next release.

### Day 4 (if available): Item 4 stretch (ailang.ebnf)

- Author EBNF source; CI gate that round-trips representative programs.
- Publish as release asset.

---

## Acceptance Criteria

- `make ci` passes.
- New tests (MOD012 reproduction, effect-row provenance reproduction) pass.
- `make error-codes` produces a JSON file validated against the schema.
- Release workflow publishes `error_codes/v0.17.0.json`.
- Doc pages exist for `MOD012` and the updated `TYP_EFFECT_ROW_MISMATCH`.
- `error_codes.json` is referenced from `docs/docs/reference/errors/index.md` with consumption instructions.
- A test in `internal/messaging/` or similar verifies the artifact path is reachable post-release (smoke test).

## Telemetry

- Count occurrences of `MOD012` and `TYP_EFFECT_ROW_MISMATCH` in the eval baselines pre- and post-implementation.
  - Expectation: MOD012 stays near zero (rare, structural). TYP_EFFECT_ROW_MISMATCH counts unchanged but downstream pass-rate after error-driven retries should improve.

## Risks

1. **Provenance plumbing touches hot inference path.** Mitigation: provenance fields are populated only on label introduction (cheap), looked up only on unification failure (rare). Should be near-zero overhead. Benchmark with the existing `make perf` gate.
2. **Error code inventory has stragglers.** Many codes emitted from deep call chains; first sweep may miss some. Mitigation: CI gate reports the gap and is informational for one release before becoming fatal.
3. **Schema choice for `error_codes.json` may need to evolve.** Mitigation: version field included; v1 is intentionally minimal.

## Documentation Impact

- New: `docs/docs/reference/errors/mod012.md`.
- Updated: `docs/docs/reference/errors/typ_effect_row_mismatch.md`.
- Updated: `docs/docs/reference/errors/index.md` (add `error_codes.json` consumption section).
- New: `docs/docs/guides/external-consumers.md` — short guide for projects vendoring/consuming AILANG, linking to the artifacts and explaining how to file feedback (GitHub issues, MCP `submit_feedback`, `ailang messages send --github`).
- CHANGELOG.md: entry under v0.17.0.

## Receive Path (Already Working)

External submissions via `mcp.ailang.sunholo.com` → Pub/Sub → Firestore (`inbox_messages`, `to_inbox=public-feedback`) and surface to the maintainer via `scripts/hooks/check_public_feedback.sh` — a SessionStart hook (107 LOC, wired in `.claude/settings.json` with an 8s timeout) that reads Firestore directly via ADC and prints up to 5 unread items at session start. Verified end-to-end on 2026-05-04 with motoko_agent ticket `fb_d3920906975b66e2`. Between-session pings (real-time macOS notifications) are tracked separately under [M-MAC-NOTIFY-DAEMON](../m-mac-notify-daemon.md).

## Related Work

- M-AGENT-MCP M7.1 — `submit_feedback` MCP tool already wired to Pub/Sub. External consumers can already submit structured feedback; this milestone gives them better diagnostics to *base* that feedback on.
- M-MAC-NOTIFY-DAEMON — between-session real-time notifications. Complementary to the SessionStart hook above; not required for receive path correctness.
- M-AI-TOOL-LOOP — same release; complementary (one improves diagnostics for AILANG-as-target-language; the other lets AILANG drive agent loops).
- Future: M-AILANG-AUTHOR-API (proposed v0.18) — programmatic AST-builder for structured tool-call authoring, which depends on the canonical pretty-printer being decoupled from the parser. Out of scope here.
