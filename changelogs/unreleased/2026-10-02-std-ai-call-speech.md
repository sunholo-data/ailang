### Added — `std/ai.callSpeech`: text-to-speech through the AI effect (#1495)

- `callSpeech(text: string, voice: string, style: string, options: string) -> Result[bytes, AIError] ! {AI}`
  returns raw PCM in one fixed format: signed 16-bit little-endian, mono, 24000 Hz, which `std/audio`'s
  `wavFromPcm` and `encode` take. `speechSampleRate()` returns the rate.
- Implemented for Gemini TTS (AI Studio and Vertex) through the provider registry, so it uses the same
  `--ai` handler and credentials as every other AI call. `options` is JSON with `model` (default: the bound
  model if it is a TTS model, else `gemini-2.5-flash-preview-tts`) and `language_code`; unknown keys are
  `Err(SchemaValidation)`. A reply in any other audio format is `Err(ProtocolError)`. Other providers return
  `Err(CapabilityNotSupported)`.
- `--ai-stub` returns a fixed 250 ms 400 Hz tone, identical on every platform, and `--ai-stub-fixtures`
  replays `speech` fixtures keyed by `{style, text, voice}` (`pcm_base64` payload).
- Example: `examples/runnable/ai_call_speech.ail`.
