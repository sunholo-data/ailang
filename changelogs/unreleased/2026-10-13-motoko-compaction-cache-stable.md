### Fixed — motoko keeps the provider prompt cache warm after compaction (pin b1ffa339)

- motoko's re-applied compaction summary carried the current step number, and it sits right
  after the system prompt, so every request after a compaction missed the provider's prompt
  cache from that point on. On 2026-10-13, every GLM-5.3-Flash run that compacted cached 23–59%
  of input, against 95% for the one run that didn't and 92–98% for pi. That accounted for much of
  motoko's 3.8× cost.
- The summariser is now told to reply with text only. Its first attempt was being sent the
  agent's tool list and answered with a tool call, which cost a 75 s retry.
