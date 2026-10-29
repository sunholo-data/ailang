### Fixed: streaming callback effects propagate to callers (refs #1958)

- `std/ai.stepWithStream` and `stepWithStreamRecorded` accept pure callbacks under
  `AI` alone and propagate effectful callback rows to their callers. An IO
  renderer requires `AI, IO`; its IO is no longer silently discharged.
- Both builtin types share the callback row with the enclosing call. The
  `ai_streaming_effect_rows` example demonstrates pure and IO renderers with
  deterministic `--ai-stub` execution.
