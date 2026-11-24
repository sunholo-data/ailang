# Gemini file slicing M0 evidence

**2026-11-24 · Issue #2458 · D6 sign-off pending**

A live Vertex EU request reached `aiplatform.eu.rep.googleapis.com` at `/v1/projects/<project>/locations/eu/publishers/google/models/gemini-3.5-flash-lite:generateContent`. The fixture was a synthetic one-page PDF with a single text marker, passed as `contents[0].parts[0].inlineData` with MIME type `application/pdf`. The baseline and separate per-part low/high `mediaResolution.level` requests returned HTTP 403 with `aiplatform.endpoints.predict` denied. Separate requests with `pageOptions` and `contentLayers` directly on the part returned HTTP 400 identifying each as an unknown field. A separate parser probe with `documentMetadata` on a text part returned 403. That response supplies no evidence of document slicing support.

No same-residency fixture bucket was supplied, so the planned `gs://` smoke test and multi-page positive and negative controls remain outstanding. No alternate region was used. No request returned observable slice results or token counts. The Developer API v1beta `generateContent` text-only credential check returned HTTP 403 with `ACCESS_TOKEN_SCOPE_INSUFFICIENT`. The v1beta `POST /interactions` text-only credential check also returned HTTP 403.

## Capability matrix

The wire shapes in this table are discovery definitions unless marked as live results. Discovery does not establish model support or behavior.

| Control | Vertex EU v1 generateContent | Developer API v1beta generateContent |
|---|---|---|
| Document page range and layers | Discovery has no document metadata on `Part`. Direct `pageOptions` and `contentLayers` were rejected live. | `Part.documentMetadata` declares inclusive 1-indexed `startPage` and `endPage`, and layers `PAGE_TEXT` and `PAGE_IMAGE`. |
| Spreadsheet sheet, cells and layers | Discovery has no spreadsheet metadata on `Part`. | `Part.spreadsheetMetadata` declares **1-indexed** `sheetIndex`, A1 `visibleCells` and layers `DATA`, `FORMULAS` and **`VISUAL`**. The planned zero-based input would require translation. |
| Audio offsets and layers | Discovery has no audio metadata on `Part`. | `Part.audioMetadata` declares `startOffset` and `endOffset` as duration strings. It declares no layer or FPS fields. |
| Video offsets, FPS and layers | `Part.videoMetadata` declares duration-string offsets and `fps`. It declares no layer field. | `Part.videoMetadata` declares duration-string offsets, `fps` and layers `TRANSCRIPT`, `FRAMES` and `AUDIO`. |
| Per-part resolution | Discovery declares `Part.mediaResolution`. Low and high live trials reached the permission denial. | `Part.mediaResolution.level` declares `MEDIA_RESOLUTION_LOW`, `MEDIA_RESOLUTION_MEDIUM` and `MEDIA_RESOLUTION_HIGH`, among other values. |
| Tool-result attachment | Discovery declares `functionResponse.parts[]` with `fileData` and `inlineData`. | Discovery declares `functionResponse.parts[]` with file/inline data and media metadata. |
| Usage | No successful live request. | No successful live request. Discovery declares `usageMetadata.promptTokenCount` and `candidatesTokenCount`. |

Interactions media placement and controls remain untested. The Developer API discovery document returned revision `20261123`. Vertex EU v1 returned revision `20261124`. The current AILANG checkout already maps `eu` to the tested host in `internal/ai/gemini/client.go`, with coverage in `client_test.go`.

Sources · [Vertex EU v1 discovery](https://aiplatform.eu.rep.googleapis.com/$discovery/rest?version=v1) · [Gemini Developer API v1beta discovery](https://generativelanguage.googleapis.com/$discovery/rest?version=v1beta) · [Vertex inference reference](https://docs.cloud.google.com/gemini-enterprise-agent-platform/reference/models/inference)

## D6 decision

M0 remains open. The review needs to decide whether to supply an EU Vertex identity with prediction permission, a supported model and an authorized same-residency fixture bucket, or revise the approved transport scope given the Developer API discovery evidence. Live Developer API testing needs a credential accepted by that service. Successful requests must still verify marker inclusion and exclusion, tool-result placement, and whole-file versus sliced token counts. The implementation milestones await that decision.
