### Fixed — Vertex speech uses the GA Flash TTS default (2026-10-29)

`std/ai.callSpeech` now defaults to `gemini-2.5-flash-tts` on Vertex AI, where the preview ID returned internal errors. AI Studio retains `gemini-2.5-flash-preview-tts`; an explicit TTS model still takes precedence. Refs #1949.
