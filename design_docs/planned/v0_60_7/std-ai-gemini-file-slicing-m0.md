# Gemini file slicing M0 evidence

**2026-11-24 · Issue #2458 · D6 signed off**

## Discovery & Live Probe Summary

1. **Vertex EU `v1beta1` (`https://aiplatform.eu.rep.googleapis.com/$discovery/rest?version=v1beta1`, rev `20261124`)** declares the full slicing wire schema on `GoogleCloudAiplatformV1beta1Part`, matching **Gemini Developer API `v1beta` (`rev 20261123`)**:
   - `documentMetadata` (`GoogleCloudAiplatformV1beta1DocumentMediaMetadata`): `startPage`, `endPage` (1-indexed inclusive), `contentLayers` (`PAGE_TEXT`, `PAGE_IMAGE`).
   - `spreadsheetMetadata`: 1-indexed `sheetIndex`, `visibleCells` (A1 range), `contentLayers` (`DATA`, `FORMULAS`, `VISUAL`).
   - `audioMetadata`: `startOffset`, `endOffset` (protobuf duration strings; no `contentLayers` or `fps`).
   - `videoMetadata`: `startOffset`, `endOffset`, `fps`, `contentLayers` (`TRANSCRIPT`, `FRAMES`, `AUDIO`).
   - `mediaResolution`: `level` (`MEDIA_RESOLUTION_LOW`, `MEDIA_RESOLUTION_MEDIUM`, `MEDIA_RESOLUTION_HIGH`, `MEDIA_RESOLUTION_ULTRA_HIGH`).
   - `functionResponse.parts[]`: supports multimodal `fileData`/`inlineData` + media metadata.
   - *(Vertex `v1` does NOT declare `documentMetadata`, `spreadsheetMetadata`, or `audioMetadata`; therefore `internal/ai/gemini` routes Vertex requests carrying slicing/resolution/tool-media to `v1beta1` on the exact same regional/EU host.)*

2. **Live 3-page synthetic PDF probe on Vertex EU `v1beta1` (`aiplatform.eu.rep.googleapis.com`, project `ailang-multivac`, `inlineData`):**
   - `gemini-3.5-flash-lite` baseline (whole 3 pages): `promptTokenCount = 1580`, `candidatesTokenCount = 39`, all 3 markers returned.
   - `gemini-3.5-flash-lite` with `documentMetadata: {startPage: 1, endPage: 1}`: HTTP 200 accepted, `promptTokenCount = 1580` (all 3 markers returned; `documentMetadata` accepted on wire, not yet enforced on `inlineData` for `gemini-3.5-flash-lite`).
   - `gemini-3.5-flash-lite` with `documentMetadata` + `mediaResolution: {level: "MEDIA_RESOLUTION_LOW"}`: HTTP 200, **`promptTokenCount = 818`** (~48% token reduction verified live).
   - `gemini-3.5-flash` with `documentMetadata: {startPage: 1, endPage: 1}`: HTTP 200 accepted, `promptTokenCount = 1577`.
   - `gs://` `fileData` live EU bucket check and Developer API `v1beta` live key check remain follow-up operational checks (local `GOOGLE_API_KEY` returned `API_KEY_INVALID` during probe).
