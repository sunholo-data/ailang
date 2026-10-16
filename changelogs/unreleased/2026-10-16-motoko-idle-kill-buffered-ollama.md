### Fixed — motoko runs on local ollama no longer killed mid-generation by our idle watchdog

Root cause of the 9/39 zero-duration motoko "timeouts" in the v0.54.9 OS rotation: rotation runs used the
buffered ollama `/v1` path, so a whole generation arrives at once and motoko's session log is silent while
the model works; the motoko executor's 10-minute idle watchdog read that silence as a hang. Every killed
session ended on `provider_call_prepared`, and rig-gate logged ollama answering 200 with the call cancelled
at 618 s. 63 of 4,142 qwen3.8 calls since 2026-10-12 ran ≥590 s.

- The motoko executor opts `ollama/*` models into streaming (`AILANG_OLLAMA_V1_STREAM=1`) unless the
  operator set the flag, or the configured HTTP timeout is non-positive (streaming refuses it; the run stays
  buffered and says so). Needs motoko's child-env allowlist to forward the flag (motoko_agent
  `fix/forward-ollama-stream-env`).
- The idle watchdog defers to the provider's own deadline (+1 min) while a provider call is in flight, so
  a buffered call is no longer killed inside its bound.
- An idle kill now finishes `idle_killed` and banks as an **invalid** `harness_error` row (retried by
  `--skip-existing`, excluded from pass rates) carrying its real duration, tokens and cost. These rows
  previously banked as valid model `timeout` failures with 0 ms / 0 tokens / $0.
