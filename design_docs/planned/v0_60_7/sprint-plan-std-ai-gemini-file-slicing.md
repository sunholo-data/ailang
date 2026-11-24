# Sprint plan for std/ai Gemini file slicing

- **Sprint ID** `M-STD-AI-GEMINI-FILE-SLICING`
- **Target** v0.60.7
- **Estimate** 11 working days, 1,440 LOC including tests and documentation, high risk
- **Status** Paused at M0 for D6 sign-off

Build the approved typed media surface for single-shot and multi-turn calls, enable verified server-side slicing on Vertex first, and surface token counts for docparse's whole-file comparison. [Approved design](std-ai-gemini-file-slicing.md) · [Issue #2458](https://github.com/sunholo-data/ailang/issues/2458)

## Baseline and estimate

- `std/ai.ail` has closed image-only messages and step usage fields. Gemini single-shot generation has a legacy string-encoded media path and already parses provider usage. Gemini's step adapter builds media for user messages and encodes tool responses separately. Its stream adapter shares the step request builder.
- PR #2077's routing is already in `internal/ai/gemini/client.go`, with EU coverage in `client_test.go`. Reuse the configured Vertex host and transport. Search all message producers, including `std/ai.ail` itself, the four runnable examples, executor tools, WASM, experimental examples, motoko tools and embedded Go/website snippets. The checkout contains more candidates than the four examples listed in the design.
- The velocity script found one commit in the shallow checkout and no usable seven-day diff. The archive changelog samples do not establish current throughput. Treat 131 LOC/day as a planning average, derived from the milestone estimates across 11 days. The schedule retains roughly two days for integration, probe follow-up and regression fixes. Human review and provider availability can add elapsed time. Coverage baseline was unavailable because `make` is absent in this workspace.
- Public Vertex references document per-part resolution and video metadata. Treat PDF page/layer controls, spreadsheet controls, audio controls, tool-result media and specific endpoint/model combinations as unverified until the probe measures them. [Vertex REST content reference](https://docs.cloud.google.com/gemini-enterprise-agent-platform/reference/rest/Shared.Types/Content) · [Vertex inference reference](https://docs.cloud.google.com/gemini-enterprise-agent-platform/reference/models/inference)

## Rulings and probe sign-off

D1 migrates `Message.images` to ordered `Message.media` with `Image` and `File` constructors. Keep `ImagePart` for generation refs. D2 attaches slicing per file. D3 uses strict JSON options by MIME class. D4 stages Vertex first and gates Developer API support on evidence. D5 adds `callMediaResult` and preserves existing single-shot return contracts. Existing step usage fields carry provider counts. Schema-enforced media JSON output is listed under the approved design's **Future Work**. The example can request JSON as text and decode it with `std/json`.

The design's **Deferred Decisions** and **High-Impact Decisions D6** explicitly require human sign-off on the live wire probe. Before starting M1A, publish the sanitized M0 evidence for review. **STOP: commit current work, report `BLOCKED: D6 Vertex and Developer API wire probe awaiting sign-off`, and end the run.** If Vertex EU PDF slicing is unsupported or cannot be verified, record that outcome for the same decision. M0 completion requires evidence and sign-off. A resumed run uses the approved support matrix and records any approved scope change. Other deferred details are executor decisions within the design. Validate the requested controls once at the effect boundary, following Implementation Plan Phase 1 and Risks & Mitigations. Use adapters for capability checks and wire encoding.

## Registry reuse audit

Searched the registry for `gemini`, `multimodal` and `docparse`. Inspected `pkg info` and `pkg docs` for `sunholo/gemini_files@0.2.3` (URI upload/lifecycle), `sunholo/gemini_agents@0.1.1` (managed Deep Research interactions) and `sunholo/gemini_live@0.5.1` (live voice protocol and TTS). The upload package can supply URIs to existing consumers. The runtime accepts caller-provided URIs without a registry dependency.

| Milestone | Decision | Reason |
|---|---|---|
| M0 | `none` | Direct provider probes establish model, endpoint and field behavior. |
| M1A | `none` | The stdlib, builtin and effect contracts belong in core. |
| M1B | `none` | Reuse the existing Gemini transport and usage decoding in core. |
| M1C | `none` | Tool turns and provider capability checks live in the existing adapters. |
| M2 | `none` | Endpoint gating and repository demos exercise the native AI effect. |

## Milestones

### M0 — Verify Gemini wire contracts

[Sanitized M0 evidence and capability matrix](std-ai-gemini-file-slicing-m0.md). The credential checks prevented a successful live comparison. Both acceptance criteria remain open pending D6 review.

**Day 1.** Estimate 45 LOC evidence + 0 test LOC. Use ADC, an authorized same-residency GCS bucket with synthetic fixtures and a supported model. Start with the design's one-page smoke test, then use a multi-page PDF with distinct text and image markers and compare requested, excluded and whole-file content. Test the inclusive page boundaries, layer behavior, usage and invalid-option controls. Check spreadsheet ranges/layers, audio/video offsets/fps/layers, and tool-result attachment placement with distinct markers and negative controls. Compare generateContent with Interactions where supported. Probe Developer API separately with a suitable file URI and credentials. Record untested combinations explicitly.

- [ ] Record sanitized host, endpoint, model, part placement, option mapping, returned usage and observable slice behavior by control and media class.
- [ ] Provide a capability matrix and invoke the D6 sign-off gate before landing wire code. Preserve provider and regional selection throughout the probe.

### M1A — Add the typed effect surface and migrate callers

**Days 2–4.** Estimate 335 implementation LOC + 170 test LOC. Sequence the provider request types, builtin/runtime glue and atomic stdlib/caller migration as focused checkpoints. Add `internal/builtins/ai_media.go` and `internal/effects/ai_media.go` for the new operation. Update builtin registration, effect dispatch and AI budget accounting, `ai_decode.go`, `ai_encode.go`, `std/ai.ail`, Go and AILANG message producers, interface fixtures and snapshots. Keep `ImagePart` generation refs and legacy JSON prompting. Extend `ai_stub_fixtures.go` with media and usage fixtures. Preserve default model selection and the existing per-call override semantics.

- [ ] `Message.media` and builtin types agree. Migrate tool, cache, stream and vision examples with existing image behavior and order intact.
- [ ] Shared decoding rejects malformed or unknown JSON fields, invalid resolution, wrong MIME-class layers, invalid inclusive pages, sheet ranges and offset/fps values with non-retryable `SchemaValidation`. Validation tests cover each media class and call-level options.
- [ ] `callMediaResult` deterministically returns text and usage under the stub and accounts for AI budget using the existing rules. Existing text and JSON call signatures and image generation refs pass their regressions.

### M1B — Implement verified Vertex single-shot and user-turn mapping

**Days 5–6.** Estimate 205 implementation LOC + 130 test LOC. Add the confirmed wire types and shared part builder, preferably in `internal/ai/gemini/media.go`. Reuse it from `generate.go` and `step.go`. Keep the legacy multimodal builder reachable. Map provider usage to the existing Go token fields. Check endpoint/model/MIME support for every requested option before sending any part. Verify the PDF example against Vertex EU. If Interactions is the only proven surface, use the signed-off routing and capability matrix as the boundary of the adapter work.

- [ ] Request capture proves `gs://` uses `fileData` at the verified Vertex EU host and supported inline sources use `inlineData`. Each part keeps its ordering and verified controls.
- [ ] Fixtures assert provider input/output counts on single-shot and user-turn calls. The live comparison records exact counts for whole and sliced synthetic PDFs.
- [ ] Unsupported valid options or resolution return non-retryable `CapabilityNotSupported` with zero outbound generation requests. Legacy text and multimodal payloads retain their existing wire shapes.

### M1C — Preserve media in tool loops and enforce provider capabilities

**Days 7–8.** Estimate 135 implementation LOC + 110 test LOC. Use the M0-verified tool result encoding, preserving tool call IDs, response ordering and multiple results. If the approved matrix calls for a follow-on, surface typed `CapabilityNotSupported` for tool-role media on the affected route. Verify step, cache and stream request paths. Check Anthropic, OpenAI, OpenRouter and Ollama for image conversion and typed file capability errors. Place new wire/sequence cases in `media_step_test.go` and new bridge tests in `ai_media_test.go` instead of expanding the large existing step test files.

- [ ] Tests prove media remains attached to the intended tool result or returns a typed capability error before dispatch. Step usage and the existing function-call and streaming regressions pass.
- [ ] All providers preserve default-resolution image handling. Unsupported explicit controls and file inputs fail closed across single-shot, step and streaming paths.

### M2 — Gate Developer API and finish examples, docs and validation

**Days 9–11.** Estimate 155 implementation/docs LOC + 155 test and fixture LOC. Enable Developer API combinations confirmed by M0. Keep unsupported combinations gated and record the follow-on scope. Add `examples/runnable/ai_file_slicing.ail` with deterministic fixture inputs for PDF, spreadsheet and audio/video validation. Update `docs/docs/recipes/ai-file-slicing.md` and the existing docs index if needed. Regenerate `llms.txt` via the repository tool and add `changelogs/unreleased/YYYY-MM-DD-std-ai-gemini-file-slicing.md` with a `### ` heading and the breaking migration. Reserve the final two days for the live report, targeted regressions and fixes.

- [ ] Developer API tests assert supported wire shapes where evidenced and explicit rejection of unverified controls without sending the whole file.
- [ ] The example type-checks and runs with `--ai-stub` using `AI` and `IO`. The documented EU live scenario and its token comparison match the approved matrix.
- [ ] Migration docs show constructor imports, text-only `media: []`, vision migration and usage fields. Track unsupported issue requirements as pending rather than treating the issue as fully delivered.

## AILANG example checklist

AILANG prompt version loaded: v0.16.16. The available binary reported it may be stale. Execution uses the branch-built CLI to check all examples and snippets.

| Showcase | Contracts | Effects | Inline tests |
|---|---|---|---|
| New file-slicing example and recipe | skip — provider response semantics are verified with fixtures and live controls | include `main() -> () ! {AI, IO}` | skip — exercise the effectful example with stub fixtures and asserted usage |
| Migrated tool loop, caching, streaming and vision examples | skip — migration changes the message literal shape only | include existing declared rows with `AI` and their current I/O capabilities | skip — use the existing runnable example regressions |

## Validation and completion

- Run `make test-changed` at code checkpoints and `make verify-stdlib`, `make verify-examples`, `make check-file-sizes` and `make check-changelog` at integration. Run focused Go tests for touched provider, builtin and effect packages, plus `make check-boundaries` for the cross-layer changes.
- Check the new and migrated examples with the branch-built CLI. Cover every media-class validation branch, both usage paths, and supported and rejected transport branches. Keep touched-package coverage at or above the execution baseline.
- Use only the approved support matrix in docs and release notes. Report provider limitations and any live checks that could not run. Reference #2458 in implementation commits. Close it on merge only when the accepted requirements are delivered.
