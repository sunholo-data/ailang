# Dev CI recovery: Gemini MediaPart unused-wrapper lint + builtin signature goldens

**Status**: Planned
**Target**: v0.60.7 (same unreleased train as the change that caused it)
**Priority**: P0 (dev CI is red; blocks the train)
**Estimated**: ~2 hours
**Dependencies**: None — recovery for already-landed commit `1b11307` (M-STD-AI-GEMINI-FILE-SLICING, #2458)

**Triage iteration 55 · dev `c6b634d55e88f36fe97dbecfb374ae354290f7e0`**
**Lint job**: 138560725982 · **Test job**: 138560725486 · **CI**: <https://github.com/sunholo-data/ailang/actions/runs/45426753758>
**Baseline**: green on run 45386743786 (pre-change); red from first push carrying `1b11307` (run 45388079628).
**Design review**: evidence verified by OpenAI GPT-6 Luna; route judged by MiniMax M3.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Restores the golden signature snapshot as a deterministic, reviewed checkpoint of the shipped surface |
| A2: Replayability | 0 | No trace/runtime changes |
| A3: Effect Legibility | +1 | Golden diff review confirms `{AI}` effect rows survive unchanged on all 5 signatures |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Recovery is locally checkable: one lint run + one `go test` package |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | CI gates (golden snapshot, `unused`) stay authoritative and machine-checkable |
| A8: Minimal Syntax | +1 | Net LOC change is negative (dead wrapper deleted); no syntax added |
| A9: Cost Visibility | 0 | No cost-path changes |
| A10: Composability | 0 | No API shape changes |
| A11: Structured Failure | 0 | No error-surface changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

Commit `1b11307` landed the Gemini MediaPart typed-media work (#2458) but left two artifacts behind, turning dev CI red on the next runs:

**Failure 1 — lint job 138560725982 (`unused` linter, enabled in `.golangci.yml`):**

`internal/ai/gemini/generate.go:286-289` defines `(*Client).vertexModelURL(model, method string)` — a v1-only wrapper delegating to `vertexModelURLVersion(model, method, "v1")`. The same commit introduced the version-parameterized `vertexModelURLVersion` and **migrated both live call sites to it** (`generate.go:279` in `buildURLVersion`, `generate.go:326` in `buildStreamURLVersion`) so Vertex requests can pick `v1` vs `v1beta1` per request. The wrapper was superseded and left behind with **zero callers** — dead code that `unused` correctly flags. This is exactly the pre-change-green → post-change-red pattern (run 45386743786 → 45388079628).

**Failure 2 — test job 138560725486 (`TestBuiltinTypes_GoldenSnapshot`, `internal/pipeline/builtin_golden_types_test.go`):**

The registered builtin signature snapshot diverges from `internal/pipeline/testdata/builtin_types.golden` in two intended ways that the landing commit never regenerated:

1. **New builtin `_ai_call_media_result`** (registered in `internal/builtins/ai_media.go:31`; `(prompt: string, media: [MediaPart], options: string) -> Result[MediaCallResult, AIError] ! {AI}`) — the golden file contains no such line.
2. **Four typed-media step input signatures changed**: `messageRecordType` (`internal/builtins/ai_step.go:57-68`) replaced `images: list[{mime: string, source: string}]` with `media: list[MediaPart]` (the `MediaPart` ADT: `Image({source, mime, resolution}) | File({uri, mime, resolution, options})`). The step builtins whose signatures embed the Message record — `_ai_step`, `_ai_step_with_cache`, `_ai_step_with_stream`, `_ai_step_with_stream_recorded` (golden lines 21-24) — all still show the old `images:` shape.

**Impact:** every dev CI run on this branch fails lint + the golden test; real signal (new regressions) is drowned out on both jobs from the first push carrying the change (run 45388079628) through the current run (45426753758).

## Goals

**Primary Goal:** Restore dev CI to green on the `c6b634d5` branch by deleting the superseded dead wrapper and regenerating the builtin signature golden to match the intentionally shipped v0.60.7 surface — with zero behavior change.

**Success Metrics:**
- `make lint` (golangci-lint v2, `unused` enabled) reports no finding in `internal/ai/gemini`
- `go test ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot` passes
- The regenerated golden diff is exactly 5 signature lines (1 added, 4 modified) — nothing else moves
- All existing Vertex URL-path tests pass **unedited**
- Full `make test` / `make test-core` green

## Intended Compatibility Surface (assessment)

Both flagged diffs are the **intended, already-landed semantics** of M-STD-AI-GEMINI-FILE-SLICING (v0.60.7, #2458), not accidental regressions:

- **Language surface (already shipped in this tree):** `std/ai.ail` exports `MediaPart` (lines 101-103), `Message` with the closed `media: [MediaPart]` field (line 113), `MediaCallResult` (line 117), and `func callMediaResult(prompt: string, media: [MediaPart], options: string) -> Result[MediaCallResult, AIError] ! {AI}` (lines 188-189, delegating to `_ai_call_media_result`).
- **One deliberate language-level breaking change:** `Message` is a closed record — `images: [ImagePart]` is gone, so every existing Message literal must now carry `media: []` for text-only messages. `ImagePart` remains exported (used by `callImageWithRefs`). This is documented in `std/ai.ail` and the M0 evidence doc; **the recovery adds no further compat change and must not soften this one.**
- **Compiler-side counterpart:** the builtin registration (`_ai_call_media_result`) and the `messageRecordType` change are the internal mirror of that std surface. The golden test exists precisely to force such signature changes through review (its header: "New builtins added without review / Lost effect rows").
- **Vertex request paths:** the v1/v1beta1 per-request routing (`requestNeedsVertexBeta`, `internal/ai/gemini/media.go:15`; `buildURLForRequest` / `buildStreamURLForRequest`, `generate.go:253-262, 308-318`) is **intentional and load-bearing** — Vertex `v1` rejects the v1beta1-only Part fields (documentMetadata etc.) with HTTP 400, so slicing/resolution/tool-media requests must stay on `v1beta1` on the same regional/EU host. The dead wrapper is *not* part of that surface: it hardcodes `"v1"`, which the callers now pass explicitly per request.

**Conclusion:** the narrowest recovery is artifact hygiene — delete dead code, regenerate the golden — with no revert of any landed semantics.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Regenerate the golden (vs revert the builtins to old signatures) | Reverting would un-ship the #2458 feature surface that `std/ai.ail` already exports; the golden test documents `UPDATE_GOLDEN=1` as the sanctioned update path | human (this doc records the ruling) | design | low |
| Delete `vertexModelURL` (vs keep/export/re-purpose) | It hardcodes `"v1"`, which duplicates the callers' per-request default and would re-introduce the wrong-version hazard if ever called; keeping dead code fails `unused` | agent (implementer, per this doc) | design | low |
| No edits to test assertions or Vertex URL logic | Editing tests to pass is the classic silent-fallback failure mode; the golden **data** is the only test artifact that changes | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Regenerate-not-revert ruling for the golden (recorded above; backed by std/ai.ail already shipping the surface)
- [x] Delete-not-keep ruling for the wrapper (zero callers, verified below)

## Solution Design

### Overview

Two independent, minimal changes, one commit each (or one commit; they touch disjoint files):

1. **Lint:** delete the dead `vertexModelURL` method (`generate.go:286-289`, including its doc comment).
2. **Test:** regenerate `internal/pipeline/testdata/builtin_types.golden` via the test's own documented update path.

### Architecture

No architectural change. Both fixes remove drift between the code and its checked-in verification artifacts.

**Components:**
1. **`internal/ai/gemini/generate.go`** — remove 4 lines (comment + wrapper). `vertexModelURLVersion` remains the single Vertex URL builder; both call sites keep passing their per-request `vertexVersion`.
2. **`internal/pipeline/testdata/builtin_types.golden`** — regenerated snapshot: adds `_ai_call_media_result`, rewrites the 4 `_ai_step*` lines with `media: list[MediaPart]` in place of `images: list[{mime: string, source: string}]`.

### Implementation Plan

**Phase 1: Fix the lint finding (~15 min)**
- [ ] Delete `vertexModelURL` (comment + method) at `generate.go:286-289`
- [ ] `grep -rn "vertexModelURL(" internal/ cmd/` — expect zero hits outside `vertexModelURLVersion`
- [ ] Run `make lint` (or `golangci-lint run internal/ai/gemini/...`) — expect green
- [ ] Run `go test ./internal/ai/gemini/` — expect green, zero test edits

**Phase 2: Regenerate the golden (~15 min)**
- [ ] `UPDATE_GOLDEN=1 go test -v ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot`
- [ ] `git diff internal/pipeline/testdata/builtin_types.golden` — **review the diff**; it must show exactly:
  - `+1` line `_ai_call_media_result : (string, list[MediaPart], string) -> Result[MediaCallResult, AIError] ! {AI}` (exact `Type.String()` form as generated)
  - 4 modified `_ai_step*` lines: `images: list[{mime: string, source: string}]` → `media: list[MediaPart]` (twice per line: parameter list and result record)
  - No other lines added/removed; `! {AI}` effect rows unchanged; sort order intact
- [ ] `go test ./internal/pipeline -run TestBuiltinTypes` — expect green (both snapshot and `TestBuiltinTypes_CriticalSignatures`, whose critical map is untouched by this change)

**Phase 3: Verify the whole gate (~30 min)**
- [ ] `make test-core` green
- [ ] `make test` green (or the CI-equivalent suite)
- [ ] Push and confirm lint job + test job green on the follow-up run

### Files to Modify/Create

**New files:** none.

**Modified files:**
- `internal/ai/gemini/generate.go` — delete lines 286-289 (~4 LOC removed)
- `internal/pipeline/testdata/builtin_types.golden` — regenerated (~5 lines changed)

## Examples

### Example 1: The lint fix

**Before (`generate.go:286-289`):**
```go
// vertexModelURL is a Vertex AI v1 model method URL on the host for c.location.
func (c *Client) vertexModelURL(model, method string) (string, error) {
	return c.vertexModelURLVersion(model, method, "v1")
}
```

**After:** deleted. The live call sites already read:

```go
// generate.go:279 (inside buildURLVersion, AuthADC case)
return c.vertexModelURLVersion(model, "generateContent", vertexVersion)
// generate.go:326 (inside buildStreamURLVersion, AuthADC case)
return c.vertexModelURLVersion(model, "streamGenerateContent?alt=sse", vertexVersion)
```

Byte-identical output for the `"v1"` case — the wrapper added nothing.

### Example 2: The golden diff shape (expected)

```diff
+_ai_call_media_result : (string, list[MediaPart], string) -> Result[MediaCallResult, AIError] ! {AI}
 _ai_step : (string, list[{content: string, images: list[{mime: string, source: string}], ...
-_ai_step : (string, list[{content: string, images: list[{mime: string, source: string}], ...
+_ai_step : (string, list[{content: string, media: list[MediaPart], ...
 ... (same pattern for _ai_step_with_cache, _ai_step_with_stream, _ai_step_with_stream_recorded)
 (N signatures unchanged)
```

If the diff shows **anything else** — an effect row change, an unrelated signature move — stop: that is a real regression hiding behind the expected one.

## Success Criteria

- [ ] `make lint` green; no `unused` finding for `vertexModelURL` (acceptance: lint job passes on the follow-up CI run)
- [ ] Golden regenerated; `TestBuiltinTypes_GoldenSnapshot` passes (acceptance: test job passes)
- [ ] Golden diff reviewed line-by-line and contains exactly the 5 expected signature deltas — 1 add + 4 media-field rewrites, effect rows `{AI}` unchanged
- [ ] Vertex request paths preserved: `TestBuildURLForRequest_VertexEUV1Beta1ForSlicing` (`internal/ai/gemini/media_step_test.go:10`) and `location_test.go` host-selection tests pass **unedited** — plain requests stay on `v1` (`https://aiplatform.eu.rep.googleapis.com/v1/projects/...`), slicing/resolution/tool-media requests stay on `v1beta1`
- [ ] `TestBuiltinTypes_CriticalSignatures` passes with its critical map untouched (proves no critical signature regression hides in the snapshot)
- [ ] `make test-core` green
- [ ] All tests passing; no test assertion edits anywhere in the diff
- [ ] No `std/ai.ail`, `internal/builtins/ai_*`, `internal/effects/`, or `internal/ai/gemini/media.go` changes in the diff

## Testing Strategy

**Unit tests:**
- Golden snapshot test itself (`TestBuiltinTypes_GoldenSnapshot`) — green after regen
- `TestBuiltinTypes_CriticalSignatures` — unchanged, must pass
- `internal/ai/gemini` suite (URL construction, media step, streaming) — unchanged, must pass

**Integration tests:**
- `make test` / CI matrix on the branch — lint job + test job both green

**Manual testing:**
- `git diff` review of the golden file (the 5-line shape check above)
- Confirm the deleted wrapper's absence: `grep -rn "vertexModelURL(" internal/ cmd/` returns only `vertexModelURLVersion` definition/calls

## Verification Log (premise evidence)

| # | Claim | Verification | Result |
|---|-------|--------------|--------|
| 1 | `vertexModelURL` has zero callers | `grep -rn "vertexModelURL" internal/ cmd/` | Only `generate.go:287` (definition), `:288` (its own body), and `vertexModelURLVersion` call sites `:279`, `:291(def)`, `:326`. No test file references it. **Confirmed dead** |
| 2 | The golden file lacks `_ai_call_media_result` | `grep -n "_ai_call_media_result" internal/pipeline/testdata/builtin_types.golden` | No match (only `_ollama_embed_media` at line 225 matches "media"). **Confirmed stale** |
| 3 | The four `_ai_step*` golden lines still carry `images:` | `sed`/`grep` of golden lines 21-24 | All four show `images: list[{mime: string, source: string}]`; source of truth `messageRecordType` (`ai_step.go:57-68`) now builds `media: [MediaPart]`. **Confirmed stale** |
| 4 | `Type()` for `MediaPart` renders as `list[MediaPart]` (a named ADT, not a structural record) | `ai_step.go:47-51`: `mediaPartType` returns `T.App("MediaPart")` | The golden's 4 rewritten step lines will show `media: list[MediaPart]`, not an inline record. **Confirmed** |
| 5 | `unused` is an enabled linter (this is what job 138560725982 runs) | `.golangci.yml` (`linters.enable`) | `unused` listed alongside govet/staticcheck/misspell/forbidigo. **Confirmed** |
| 6 | The language surface for these signatures is already shipped (not something this recovery must design) | `std/ai.ail:101-113, 115-117, 182-189` | `MediaPart`, `Message.media`, `MediaCallResult`, `callMediaResult` exported. **Confirmed intended compat surface** |
| 7 | The only stale copy of the old `images:` signatures is the pipeline golden | `grep -rln "images: list\[" internal/ goldens/ tests/ docs/` | Sole hit: `internal/pipeline/testdata/builtin_types.golden`. **Confirmed single artifact** |
| 8 | Vertex v1-vs-v1beta1 routing is intentional and tested | `internal/ai/gemini/media.go:10-33`; `media_step_test.go:10-45` | `requestNeedsVertexBeta` routes slicing/resolution/tool-media to `v1beta1`; tests assert both plain-`v1` and sliced-`v1beta1` EU URLs. **Do not touch** |
| 9 | `ailang check` on the new std surface | attempted with installed binary | **Blocked locally**: installed `ailang` is v0.16.16 and its bundled std predates `MediaPart` (`IMP010: symbol 'MediaPart' not exported by 'std/ai'`). The claim is instead cited to `std/ai.ail` source lines (#6). Sprint executor with a freshly built binary (`make build` from this tree, std v0.60.6+unreleased) should re-run the check before closing the sprint |
| 10 | Lint green before / red after the change | CI runs per triage | 45386743786 green pre-change; 45388079628 red on first push carrying `1b11307`. **Consistent with both root causes being in that commit** |

## Deferred Decisions

The following are intentionally left open for the implementer:

- Commit granularity (one commit vs two) — agent may choose; files are disjoint either way
- Whether to add a CHANGELOG `changelogs/` entry for the CI fix — agent may choose; note that M0 evidence already documents the wire schema, and the media feature itself carries its own unreleased changelog. This recovery is internal hygiene, not a language change

## Non-Goals

**Not attempted in this recovery:**
- Any change to `std/ai.ail`, builtin registrations, effect ops, or the MediaPart/Message type shapes — the #2458 surface stays as landed
- Any change to Vertex URL construction, `requestNeedsVertexBeta`, or the v1/v1beta1 routing policy
- Any test-assertion edits — only the golden **data** file changes
- Fixing the pre-existing v0.16.16 stale binary on this machine (separate environment issue; noted in Verification Log #9)

## Timeline

**Day 1 (~2 hours total):**
- Phase 1: lint fix (~15 min)
- Phase 2: golden regen + diff review (~15 min)
- Phase 3: full gate verification + push (~30 min)
- Reviewer back-and-forth buffer (~1 hour)

**Total: ~2 hours.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Golden regen masks a real signature regression (another builtin changed unnoticed in `1b11307`) | Med | The line-by-line diff review is a hard acceptance criterion: exactly 5 lines, effect rows unchanged. Anything else is investigated, not regenerated |
| Deleting the wrapper breaks a future caller that expected the old v1-only entry point | Low | Zero callers today (verified, Log #1); per-request version selection is the only correct path — a hardcoded `"v1"` wrapper is the hazard |
| `Type.String()` format drift (golden regenerated with a different formatter than CI) | Low | Regen uses the test's own documented path (`UPDATE_GOLDEN=1`), not a hand edit — deterministic by construction |
| A stale local `ailang` binary misleads verification of the std surface | Low | Recorded in Log #9; sprint uses `make build` output for any language-surface check |

## Related Documents

<!-- Auto-populated by Ollama neural search on "dev ci media signature goldens" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_16_0/m-pkg-autonomous-cascade-safe-sprint-plan.md](../../implemented/v0_16_0/m-pkg-autonomous-cascade-safe-sprint-plan.md) (1.00 — keyword overlap only; unrelated CI-cascade topic)
- [design_docs/implemented/v0_54_19/m-cloud-native-session-jsonl.md](../../implemented/v0_54_19/m-cloud-native-session-jsonl.md) (0.95 — unrelated)

**Planned (check for overlap):**
- [design_docs/planned/v0_60_7/std-ai-gemini-file-slicing-m0.md](std-ai-gemini-file-slicing-m0.md) (1.00 similarity — **deliberately not treated as a duplicate**: that doc is the M0 wire-schema/live-probe *evidence* record for the feature; it does not cover either CI regression, the golden update procedure, or the dead wrapper (grep: no "golden", "vertexModelURL", "lint", or "CI" mentions). This doc is the recovery design for the two red jobs on that landed work. The two are complementary artifacts of #2458.)
- [design_docs/planned/v0_33_2/m-coverage-cross-package-attribution.md](../../planned/v0_33_2/m-coverage-cross-package-attribution.md) (0.95 — unrelated)

## References

- Issue #2458 — Gemini MediaPart typed media + file slicing
- [std-ai-gemini-file-slicing-m0.md](std-ai-gemini-file-slicing-m0.md) — Vertex v1beta1 discovery evidence (D6 signed off, 2026-11-24)
- `internal/pipeline/builtin_golden_types_test.go` — the failing golden test and its documented `UPDATE_GOLDEN=1` path
- CI run 45426753758 (current red); 45386743786 (last green); 45388079628 (first red)
- [Design Axioms](/docs/references/axioms)

## Future Work

- Machine-level guard for "landing a builtin registration requires regenerating the golden in the same commit" — e.g., a CI check that `1b11307`-style commits fail fast with the regen command in the failure message (the test already prints it, but the commit landed goldens-stale; consider a PR-stage bot comment). Track separately; not blocking this recovery
- Refresh the stale v0.16.16 `ailang` binary on triage machines (Verification Log #9)

---

**Document created**: 2026-11-24
**Last updated**: 2026-11-24