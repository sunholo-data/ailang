### Fixed
- **`std/ai` tool loops work on Gemini 3.** Gemini 3 signs each `functionCall` part with a
  `thoughtSignature` and refuses the next turn unless it comes back on the same part (`400: Function
  call is missing a thought_signature`), so any `step` / `runTools` loop failed on its second turn
  (measured 2026-11-11, `gemini-3-8-flash` on Vertex). The Gemini adapter now keeps the signature in
  the tool-call ID, which was already opaque and produced only by the adapter (`0_0:<signature>`),
  and sends it back on the matching part, in both `step` and `stepWithStream`. Nothing changes for
  AILANG code: `ToolCall` keeps its three fields. `internal/ai/gemini/thought_signature_test.go`;
  mutation-checked.
