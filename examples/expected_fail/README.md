# Expected-to-Fail Examples

These examples are known to fail and are tracked for future implementation. They were moved from `examples/runnable/` to avoid noise in `verify-examples` output.

## Categories

### Contracts (advanced verification)
- `contracts/hof_verify.ail` — Higher-order function contract verification
- `contracts/list_recursive_verify.ail` — Recursive list contract verification
- `contracts/per_function_depth_verify.ail` — Per-function depth limit contracts
- `contracts/quantifier_verify.ail` — Quantifier-based contracts

### Effect Budgets
- `effect_budgets.ail` — Basic effect budget tracking
- `effect_budgets_exhausted.ail` — Budget exhaustion behavior
- `effect_budgets_multi.ail` — Multi-effect budgets
- `effect_budgets_rand.ail` — Random effect budgets

### Process/Stream (OS-level effects)
- `process_demo.ail` — Process spawning
- `process_stdin_write.ail` — Writing to process stdin
- `stream_multi_source.ail` — Multi-source streams
- `stream_process_source.ail` — Process-backed streams
- `stream_sse.ail` — Server-sent events
- `stream_websocket.ail` — WebSocket streams

### Package Demo
- `package_demo/` — Multi-module package import demo

### Archive/Binary
- `xml_zip_roundtrip.ail` — XML zip round-trip
- `zip_reader.ail` — Zip file reading

## When to move back

Move examples back to `examples/runnable/` once the underlying feature is implemented and the example passes `ailang run`.
