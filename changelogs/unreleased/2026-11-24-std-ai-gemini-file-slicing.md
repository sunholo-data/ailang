### `std/ai`: Typed `MediaPart`, Gemini File Slicing Controls & Single-Shot Usage Metering (#2458)

- **Added `MediaPart` ADT** in `std/ai`:
  `Image({ source: string, mime: string, resolution: string }) | File({ uri: string, mime: string, resolution: string, options: string })`.
- **Added `callMediaResult(prompt, media, options) -> Result[MediaCallResult, AIError] ! {AI}`** returning `{ text, input_tokens, output_tokens }` for single-shot multimodal calls with per-call token metering.
- **Per-part server-side slicing validation & Gemini wire mapping** (`internal/ai/media.go`, `internal/ai/gemini/media.go`):
  - Documents/slides: `page_options` (`start_page`, `end_page`, 1-indexed inclusive) and `content_layers` (`page_text`, `page_image`) → `documentMetadata`.
  - Spreadsheets: `sheet_index` (0-based → 1-based `sheetIndex`), `visible_cells` (A1 range), `content_layers` (`data`, `formulas`, `visuals` → `VISUAL`) → `spreadsheetMetadata`.
  - Audio/video: `start_offset`, `end_offset`, `fps` (video), `content_layers` (`transcript`, `frames`, `audio`) → `audioMetadata` / `videoMetadata`.
  - Per-part `resolution` (`"" | "low" | "medium" | "high"`) → `mediaResolution.level`.
  - Vertex AI requests carrying slicing, resolution, or tool-result media route to `v1beta1` on the configured regional/EU host (e.g. `aiplatform.eu.rep.googleapis.com/v1beta1/...`); `gs://` URIs always map to `fileData` and fail closed on AI Studio API-key clients.
  - Multimodal tool results attach to `functionResponse.parts[]` on Gemini; non-Gemini providers fail closed with `CapabilityNotSupported` for `File` parts, explicit resolution, slicing options, or tool-role media while preserving default-resolution image input.
- **Breaking change (D1):** `Message.images: [ImagePart]` is replaced by `Message.media: [MediaPart]`. `ImagePart` remains exported for `callImageWithRefs` / `callImageBase64WithRefs`.
  - Migration: `images: []` → `media: []`; `images: [{ source, mime }]` → `media: [Image({ source, mime, resolution: "" })]`.
- **Stub fixtures:** `--ai-stub-fixtures` now supports `"kind": "media"` with optional `input_tokens` and `output_tokens`.
